package qlik

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	grammar "github.com/noesiscarloslisboa/qlik-code-intel/tree-sitter-qlik/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Parse extracts syntax facts without evaluating scripts, variables, or includes.
func Parse(ctx context.Context, filePath string, source []byte) (File, error) {
	f := File{
		Path: filePath, Source: append([]byte(nil), source...), Symbols: []Symbol{},
		Edges: []Edge{}, Diagnostics: []Diagnostic{}, Loads: []Load{}, Statements: []Statement{},
	}
	if err := ctx.Err(); err != nil {
		return f, err
	}
	if !utf8.Valid(source) {
		return f, fmt.Errorf("parse %s: script is not valid UTF-8", filePath)
	}
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(grammar.Language())); err != nil {
		return f, fmt.Errorf("load Qlik grammar: %w", err)
	}
	tree := parser.ParseWithOptions(func(offset int, _ sitter.Point) []byte {
		return f.Source[offset:]
	}, nil, &sitter.ParseOptions{ProgressCallback: func(sitter.ParseState) bool { return ctx.Err() != nil }})
	if tree == nil {
		if err := ctx.Err(); err != nil {
			return f, err
		}
		return f, fmt.Errorf("parse %s: no syntax tree", filePath)
	}
	defer tree.Close()
	x := extractor{file: &f}
	root := tree.RootNode()
	x.statements(root)
	walk(root, func(n *sitter.Node) {
		if n.IsError() {
			x.diagnostic(n, "syntax-error", "unrecognized or malformed syntax; surrounding facts may be incomplete")
		} else if n.IsMissing() {
			x.diagnostic(n, "missing-token", "expected "+n.Kind())
		}
	})
	return f, ctx.Err()
}

type extractor struct{ file *File }

func (x *extractor) statements(root *sitter.Node) {
	var pending *sitter.Node
	var owner Entity
	unresolved := func() {
		if pending != nil {
			x.diagnostic(pending, "unresolved-source", "source-less LOAD has no recognized adjacent input; SQL and runtime source resolution are not inferred")
			pending = nil
		}
	}
	for i := uint(0); i < root.NamedChildCount(); i++ {
		n := root.NamedChild(i)
		if n.Kind() == "comment" {
			continue
		}
		x.statement(n)
		if n.Kind() == "load_statement" {
			load := x.load(n)
			if pending != nil {
				if !n.HasError() && child(n, "table_label") == nil && child(n, "load_prefix") == nil {
					x.edgeInStatement(n, pending, owner, Entity{Table, load.Owner}, "preceding")
					pending = nil
				} else {
					unresolved()
				}
			}
			if !n.HasError() && !loadHasInput(n) {
				pending, owner = n, Entity{Table, load.Owner}
				// JOIN is explicitly excluded from preceding loads by Qlik.
				for j := uint(0); j < n.NamedChildCount(); j++ {
					if p := n.NamedChild(j); p.Kind() == "load_prefix" && child(p, "join_prefix") != nil {
						unresolved()
						break
					}
				}
			}
			continue
		}
		unresolved()
		switch n.Kind() {
		case "set_statement", "let_statement":
			x.assignment(n)
		case "store_statement":
			x.store(n)
		case "include_statement":
			x.include(n)
		case "rename_field_statement", "rename_table_statement", "drop_field_statement", "drop_table_statement", "trace_statement":
			x.operation(n)
		case "unsupported_statement":
			x.diagnostic(n, "unsupported", "unsupported statement: "+x.text(n.ChildByFieldName("keyword")))
			if child(n, "unterminated_unsupported_body") != nil {
				x.diagnostic(n, "unterminated-statement", "unsupported statement has no terminating semicolon before a line-start table label or end of file")
			}
		case "unsupported_literal_statement":
			x.diagnostic(n, "unsupported-literal", "standalone literal is not a supported statement; no symbols extracted")
		case "unsupported_control_statement":
			x.diagnostic(n, "unsupported-control", "control flow is not evaluated; body statements are indexed as source facts")
		}
	}
	unresolved()
}

func loadHasInput(n *sitter.Node) bool {
	for _, kind := range []string{"from_clause", "resident_clause", "autogenerate_clause", "inline_clause"} {
		if child(n, kind) != nil {
			return true
		}
	}
	return false
}

func (x *extractor) text(n *sitter.Node) string {
	if n == nil {
		return ""
	}
	return n.Utf8Text(x.file.Source)
}

// Name removes one enclosing Qlik quote/bracket pair and decodes doubled quotes.
func Name(text string) string {
	s := strings.TrimSpace(text)
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if first == '[' && last == ']' {
			return s[1 : len(s)-1]
		}
		if first == '`' && last == first {
			return s[1 : len(s)-1]
		}
		if (first == '\'' || first == '"') && last == first {
			q := string(first)
			return strings.ReplaceAll(s[1:len(s)-1], q+q, q)
		}
	}
	return s
}

func (x *extractor) loc(n *sitter.Node) Location {
	p, end := n.StartPosition(), n.EndPosition()
	endLine := int(end.Row) + 1
	if end.Column == 0 && end.Row > p.Row {
		endLine--
	}
	return Location{Path: x.file.Path, Line: int(p.Row) + 1, Column: int(p.Column) + 1, EndLine: endLine}
}

func (x *extractor) symbol(n, stmt *sitter.Node, kind Kind, role, owner string) {
	if n == nil || n.IsMissing() || n.HasError() {
		return
	}
	name := Name(x.text(n))
	if name == "" {
		return
	}
	rangeLoc := x.loc(stmt)
	x.file.Symbols = append(x.file.Symbols, Symbol{Location: x.loc(n), Kind: kind, Role: role, Name: name, Owner: owner, StartLine: rangeLoc.Line, StopLine: rangeLoc.EndLine, StartByte: stmt.StartByte(), StopByte: stmt.EndByte()})
}

func (x *extractor) edge(n *sitter.Node, from, to Entity, kind string) {
	if n == nil {
		return
	}
	stmt := n
	for parent := stmt.Parent(); parent != nil; parent = stmt.Parent() {
		if strings.HasSuffix(stmt.Kind(), "_statement") {
			break
		}
		stmt = parent
	}
	x.edgeInStatement(n, stmt, from, to, kind)
}

func (x *extractor) edgeInStatement(n, stmt *sitter.Node, from, to Entity, kind string) {
	if n.IsMissing() || n.HasError() || from.Name == "" || to.Name == "" {
		return
	}
	x.file.Edges = append(x.file.Edges, Edge{Location: x.loc(n), From: from, To: to, Kind: kind, StartByte: stmt.StartByte()})
}

func (x *extractor) diagnostic(n *sitter.Node, code, message string) {
	x.file.Diagnostics = append(x.file.Diagnostics, Diagnostic{Location: x.loc(n), Code: code, Message: message})
}

func (x *extractor) assignment(stmt *sitter.Node) {
	name := stmt.ChildByFieldName("name")
	x.symbol(name, stmt, Variable, "definition", "")
	x.references(name, stmt, Entity{Variable, Name(x.text(name))}, false)
	x.references(stmt.ChildByFieldName("value"), stmt, Entity{Variable, Name(x.text(name))}, false)
}

func (x *extractor) load(stmt *sitter.Node) Load {
	owner := Entity{Table, fmt.Sprintf("@%s:%d:%d", x.file.Path, x.loc(stmt).Line, x.loc(stmt).Column)}
	anonymous := true
	var hierarchyOutputs []*sitter.Node
	if label := child(stmt, "table_label"); label != nil {
		name := label.ChildByFieldName("name")
		x.symbol(name, stmt, Table, "definition", "")
		owner.Name = Name(x.text(name))
		anonymous = false
	}
	for i := uint(0); i < stmt.NamedChildCount(); i++ {
		prefix := stmt.NamedChild(i)
		if prefix.Kind() != "load_prefix" {
			continue
		}
		for j := uint(0); j < prefix.NamedChildCount(); j++ {
			p := prefix.NamedChild(j)
			if p.Kind() == "hierarchy_prefix" {
				for _, name := range []string{"parent_name", "path_name", "depth"} {
					hierarchyOutputs = append(hierarchyOutputs, p.ChildByFieldName(name))
				}
			}
			if p.Kind() != "join_prefix" && p.Kind() != "concatenate_prefix" {
				continue
			}
			if target := p.ChildByFieldName("table"); target != nil {
				x.symbol(target, stmt, Table, "reference", owner.Name)
				targetEntity := Entity{Table, Name(x.text(target))}
				if !anonymous {
					x.edge(target, targetEntity, owner, strings.TrimSuffix(p.Kind(), "_prefix"))
				}
				owner = targetEntity
				anonymous = false
			} else {
				x.diagnostic(p, "implicit-target", "JOIN/CONCATENATE target depends on execution order and is not inferred")
			}
		}
	}
	for _, name := range hierarchyOutputs {
		x.symbol(name, stmt, Field, "definition", owner.Name)
	}
	fields := stmt.ChildByFieldName("fields")
	if fields != nil {
		for i := uint(0); i < fields.NamedChildCount(); i++ {
			f := fields.NamedChild(i)
			if f.Kind() != "load_field" {
				continue
			}
			if alias := f.ChildByFieldName("alias"); alias != nil {
				x.symbol(alias, stmt, Field, "definition", owner.Name)
			} else if expr := f.ChildByFieldName("expression"); expr != nil && expr.Kind() == "field_reference" {
				x.symbol(expr, stmt, Field, "definition", owner.Name)
			}
		}
	}
	for _, clauseKind := range []string{"from_clause", "resident_clause"} {
		clause := child(stmt, clauseKind)
		if clause == nil {
			continue
		}
		field, kind, edgeKind := "source", Source, "from"
		if clauseKind == "resident_clause" {
			field, kind, edgeKind = "table", Table, "resident"
		}
		n := clause.ChildByFieldName(field)
		x.symbol(n, stmt, kind, "reference", owner.Name)
		x.edge(n, owner, Entity{kind, Name(x.text(n))}, edgeKind)
	}
	x.references(stmt, stmt, owner, true)
	load := Load{Location: x.loc(stmt), Owner: owner.Name, Anonymous: anonymous, StartByte: stmt.StartByte(), StopByte: stmt.EndByte()}
	x.file.Loads = append(x.file.Loads, load)
	return load
}

func (x *extractor) store(stmt *sitter.Node) {
	table, source := stmt.ChildByFieldName("table"), stmt.ChildByFieldName("source")
	owner := Entity{Source, Name(x.text(source))}
	x.symbol(table, stmt, Table, "reference", owner.Name)
	x.symbol(source, stmt, Source, "definition", Name(x.text(table)))
	x.edge(source, owner, Entity{Table, Name(x.text(table))}, "store")
	x.references(stmt, stmt, owner, true)
}

func (x *extractor) include(stmt *sitter.Node) {
	n := stmt.ChildByFieldName("source")
	x.symbol(n, stmt, Include, "reference", x.file.Path)
	name := Name(x.text(n))
	// Qlik resolves relative includes against its app working directory, which
	// a repository alone cannot determine. Preserve the literal target.
	x.edge(n, Entity{FileKind, x.file.Path}, Entity{Include, name}, strings.ToLower(x.text(stmt.ChildByFieldName("mode"))))
	x.references(n, stmt, Entity{FileKind, x.file.Path}, false)
}

func (x *extractor) references(root, stmt *sitter.Node, owner Entity, fields bool) {
	if root == nil {
		return
	}
	walk(root, func(n *sitter.Node) {
		switch n.Kind() {
		case "variable_reference":
			name := n.ChildByFieldName("name")
			x.symbol(name, stmt, Variable, "reference", owner.Name)
			x.edge(name, owner, Entity{Variable, Name(x.text(name))}, "variable")
		case "field_reference":
			if fields {
				x.symbol(n, stmt, Field, "reference", owner.Name)
			}
		case "bare_variable_reference":
			x.symbol(n, stmt, Variable, "reference", owner.Name)
			x.edge(n, owner, Entity{Variable, Name(x.text(n))}, "variable")
		}
	})
}

func child(n *sitter.Node, kind string) *sitter.Node {
	for i := uint(0); i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		if c.Kind() == kind {
			return c
		}
	}
	return nil
}

// Iterative traversal also visits anonymous missing tokens; avoid recursion on
// deeply nested user input and do not mistake comments for variable references.
func walk(root *sitter.Node, visit func(*sitter.Node)) {
	stack := []*sitter.Node{root}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		visit(n)
		for i := n.ChildCount(); i > 0; i-- {
			stack = append(stack, n.Child(i-1))
		}
	}
}
