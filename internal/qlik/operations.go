package qlik

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

func (x *extractor) statement(n *sitter.Node) {
	if n.HasError() {
		return
	}
	switch n.Kind() {
	case "set_statement", "let_statement", "load_statement", "store_statement", "include_statement",
		"rename_field_statement", "rename_table_statement", "drop_field_statement", "drop_table_statement", "trace_statement":
		kind := strings.TrimSuffix(n.Kind(), "_statement")
		if kind == "drop_table" && n.ChildByFieldName("mapping") != nil {
			kind = "drop_mapping_table"
		}
		x.file.Statements = append(x.file.Statements, Statement{
			Location: x.loc(n), Kind: kind,
			StartByte: n.StartByte(), StopByte: n.EndByte(),
		})
	}
}

func (x *extractor) operation(stmt *sitter.Node) {
	if stmt.HasError() {
		return
	}
	// Only explicit names are facts. Do not assign field ownership, apply a
	// mapping table, delete earlier definitions, or infer rename/drop lineage.
	walk(stmt, func(n *sitter.Node) {
		switch n.Kind() {
		case "field_rename":
			x.symbol(n.ChildByFieldName("old"), stmt, Field, "reference", "")
			x.symbol(n.ChildByFieldName("new"), stmt, Field, "definition", "")
		case "table_rename":
			x.symbol(n.ChildByFieldName("old"), stmt, Table, "reference", "")
			x.symbol(n.ChildByFieldName("new"), stmt, Table, "definition", "")
		case "rename_using":
			x.symbol(n.ChildByFieldName("table"), stmt, Table, "reference", "")
		case "drop_field_list", "drop_table_list":
			kind := Field
			if n.Kind() == "drop_table_list" {
				kind = Table
			}
			for i := uint(0); i < n.NamedChildCount(); i++ {
				name := n.NamedChild(i)
				if name.Kind() == "field_reference" || name.Kind() == "table_name" {
					x.symbol(name, stmt, kind, "reference", "")
				}
			}
		}
	})
	x.references(stmt, stmt, Entity{Kind: FileKind, Name: x.file.Path}, false)
}
