package qlik

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"

	grammar "github.com/noesiscarloslisboa/qlik-code-intel/tree-sitter-qlik/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func TestOperationsSourceFacts(t *testing.T) {
	t.Parallel()
	source, err := os.ReadFile("../../testdata/accuracy/operations.qvs")
	if err != nil {
		t.Fatal(err)
	}
	f, err := Parse(context.Background(), "operations.qvs", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Diagnostics) != 0 {
		t.Fatalf("supported operations diagnosed: %+v", f.Diagnostics)
	}
	if len(f.Statements) != 17 || f.Statements[10].Kind != "drop_mapping_table" {
		t.Fatalf("supported statement records lost: %+v", f.Statements)
	}
	lines := strings.Split(string(source), "\n")
	for _, stmt := range f.Statements {
		text := string(source[stmt.StartByte:stmt.StopByte])
		if stmt.Path != f.Path || stmt.Line != stmt.EndLine || text != lines[stmt.Line-1] {
			t.Errorf("statement range does not match original source: %+v: %q", stmt, text)
		}
	}
	want := map[int][]string{
		4:  {"reference:field:Raw Price", "definition:field:Price", "reference:field:Code", "definition:field:Public Code"},
		5:  {"reference:table:Stage", "definition:table:Published"},
		6:  {"reference:table:Names"},
		7:  {"reference:table:Table Names"},
		8:  {"reference:field:Price", "reference:field:Public Code", "reference:table:Published", "reference:table:Archive"},
		9:  {"reference:field:Obsolete"},
		10: {"reference:table:Scratch Work", "reference:table:Archive"},
		11: {"reference:table:Names"},
		12: {"reference:field:Cost_$(vBatch)", "definition:field:Value_$(vBatch)"},
		13: {"reference:table:Batch_$(vBatch)", "definition:table:Archive $(vBatch)"},
		14: {"reference:field:Unused_$(vBatch)", "reference:table:Archive $(vBatch)"},
	}
	got := map[int][]string{}
	variableRefs := 0
	for _, s := range f.Symbols {
		if s.StartLine < 4 {
			continue
		}
		if s.StartLine != s.StopLine || s.StartByte >= s.StopByte || s.StopByte > uint(len(source)) {
			t.Errorf("lost operation range: %+v", s)
		}
		if s.Kind == Variable {
			if s.Role != "reference" || s.Name != "vBatch" {
				t.Errorf("unexpected expansion: %+v", s)
			}
			variableRefs++
			continue
		}
		if s.Owner != "" {
			t.Errorf("inferred operation owner: %+v", s)
		}
		got[s.Line] = append(got[s.Line], s.Role+":"+string(s.Kind)+":"+s.Name)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("operation symbols = %#v, want %#v", got, want)
	}
	if variableRefs != 8 {
		t.Errorf("expansion references = %d, want 8", variableRefs)
	}
	for _, e := range f.Edges {
		if e.Line >= 4 && (e.Kind != "variable" || e.From != (Entity{Kind: FileKind, Name: f.Path})) {
			t.Errorf("invented runtime operation dependency: %+v", e)
		}
	}
	var earlierDefinitions int
	for _, s := range f.Symbols {
		if s.Line == 2 && s.Role == "definition" {
			earlierDefinitions++
		}
	}
	if earlierDefinitions != 3 {
		t.Errorf("rename/drop changed earlier definitions: %+v", f.Symbols)
	}
}

func TestOperationsQuotesAndTraceComments(t *testing.T) {
	t.Parallel()
	f := parseAccuracy(t, "rename tables 'Old Table' to `New Table`, [Étage] to \"Public\";\n"+
		"rename field 'Old''s value' to \"Quoted \"\"name\"\"\";\n"+
		"drop mapping tables 'Unused', `Other`;\n"+
		"TRACE text /* $(vHidden) LOAD Fake; */ $(vVisible) // $(vComment)\n"+
		"/path/file.qvd cost$ [$(vBracket)] ;")
	if len(f.Diagnostics) != 0 {
		t.Fatalf("quoted operations or TRACE comments diagnosed: %+v", f.Diagnostics)
	}
	want := []string{"reference:table:Old Table", "definition:table:New Table",
		"reference:table:Étage", "definition:table:Public",
		"reference:field:Old's value", "definition:field:Quoted \"name\"",
		"reference:table:Unused", "reference:table:Other",
		"reference:variable:vVisible", "reference:variable:vBracket"}
	got := []string{}
	for _, s := range f.Symbols {
		got = append(got, s.Role+":"+string(s.Kind)+":"+s.Name)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("quoted names or TRACE opacity: got %v, want %v", got, want)
	}
}

func TestOperationsMalformedAndPrecedingBoundaries(t *testing.T) {
	t.Parallel()
	for _, operation := range []string{
		"RENAME TABLE A TO;", "RENAME FIELDS USING;", "RENAME FIELDS A TO B,;",
		"DROP TABLE;", "DROP FIELDS A FROM;", "DROP MAPPING FIELD A;",
	} {
		f := parseAccuracy(t, operation+"\nReal: LOAD ID AUTOGENERATE 1;")
		if len(f.Diagnostics) == 0 {
			t.Errorf("malformed operation accepted: %s", operation)
		}
		for _, s := range f.Symbols {
			if s.Line == 1 {
				t.Errorf("malformed operation produced facts: %+v", s)
			}
		}
		if len(f.Loads) != 1 || f.Loads[0].Owner != "Real" {
			t.Errorf("surrounding LOAD lost after %s: %+v", operation, f)
		}
	}
	for _, operation := range []string{
		"RENAME FIELD A TO B;", "RENAME TABLE A TO B;", "DROP FIELD A;", "DROP TABLE A;", "TRACE step;",
	} {
		f := parseAccuracy(t, "Result: LOAD ID;\n"+operation+"\nLOAD ID FROM [input.qvd] (qvd);")
		if len(f.Diagnostics) != 1 || f.Diagnostics[0].Code != "unresolved-source" {
			t.Errorf("operation adjacency boundary changed: %+v", f.Diagnostics)
		}
		for _, e := range f.Edges {
			if e.Kind == "preceding" {
				t.Errorf("operation bridged preceding stages: %+v", e)
			}
		}
	}
}

func TestOperationsQueries(t *testing.T) {
	t.Parallel()
	source := []byte("RENAME FIELD `Old Value` TO [New $(vBatch)];\nRENAME TABLE Stage TO Published;\n" +
		"RENAME FIELDS USING Names;\nDROP FIELDS [Old Value] FROM Published;\nDROP TABLE Stage;\n" +
		"TRACE 'literal $(vBatch)' bogus.qvd Sum(Fake);\n" +
		"TRACE $(vMessage, Sum(Fake), [$(vBatch)], 'literal $(vQuoted, Fake)');")
	language := sitter.NewLanguage(grammar.Language())
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(language); err != nil {
		t.Fatal(err)
	}
	tree := parser.Parse(source, nil)
	defer tree.Close()
	if tree.RootNode().HasError() {
		t.Fatal(tree.RootNode().ToSexp())
	}
	for _, file := range []string{"definitions", "references", "tags", "highlights"} {
		t.Run(file, func(t *testing.T) {
			scm, err := os.ReadFile("../../tree-sitter-qlik/queries/" + file + ".scm")
			if err != nil {
				t.Fatal(err)
			}
			query, queryErr := sitter.NewQuery(language, string(scm))
			if queryErr != nil {
				t.Fatal(queryErr)
			}
			defer query.Close()
			cursor := sitter.NewQueryCursor()
			defer cursor.Close()
			captures := map[string][]string{}
			matches := cursor.Matches(query, tree.RootNode(), source)
			for match := matches.Next(); match != nil; match = matches.Next() {
				for _, c := range match.Captures {
					key := query.CaptureNames()[c.Index]
					captures[key] = append(captures[key], c.Node.Utf8Text(source))
				}
			}
			if file == "highlights" {
				return
			}
			want := map[string][]string{}
			if file != "references" {
				want["name.definition.field"] = []string{"[New $(vBatch)]"}
				want["name.definition.table"] = []string{"Published"}
			}
			if file != "definitions" {
				want["name.reference.field"] = []string{"`Old Value`", "[Old Value]"}
				want["name.reference.table"] = []string{"Stage", "Names", "Published", "Stage"}
				want["name.reference.variable"] = []string{"vBatch", "vBatch", "vMessage", "vBatch", "vQuoted"}
			}
			for name, values := range want {
				if !reflect.DeepEqual(captures[name], values) {
					t.Errorf("%s: got %v, want %v", name, captures[name], values)
				}
			}
			for name, values := range captures {
				if strings.HasPrefix(name, "name.") {
					if _, ok := want[name]; !ok && len(values) != 0 {
						t.Errorf("invented capture %s: %v", name, values)
					}
				}
			}
		})
	}
}
