package qlik

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	grammar "github.com/noesiscarloslisboa/qlik-code-intel/tree-sitter-qlik/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func TestScanRepresentativeRepository(t *testing.T) {
	t.Parallel()
	idx, err := Scan(context.Background(), "../../testdata/repository")
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Files) != 5 {
		t.Fatalf("files = %d, want 5", len(idx.Files))
	}
	for _, f := range idx.Files {
		if len(f.Diagnostics) != 0 {
			t.Errorf("%s diagnostics: %+v", f.Path, f.Diagnostics)
		}
	}
	for _, want := range []struct {
		kind                    Kind
		role, name, owner, path string
		line                    int
	}{
		{Variable, "definition", "vData", "", "00_config.qvs", 2},
		{Table, "definition", "Orders", "", "10_orders.qvs", 1},
		{Field, "definition", "GrossAmount", "Orders", "10_orders.qvs", 5},
		{Field, "reference", "Amount", "Orders", "10_orders.qvs", 5},
		{Variable, "reference", "vData", "Orders", "10_orders.qvs", 8},
		{Variable, "reference", "vCutoff", "Orders", "10_orders.qvs", 7},
		{Table, "definition", "Daily Sales", "", "20_summary.qvs", 1},
		{Source, "definition", "lib://Exports/daily_sales.qvd", "Daily Sales", "20_summary.qvs", 8},
	} {
		found := false
		for _, s := range idx.Symbols() {
			if s.Kind == want.kind && s.Role == want.role && s.Name == want.name && s.Owner == want.owner && s.Path == want.path && s.Line == want.line {
				found = true
			}
		}
		if !found {
			t.Errorf("missing occurrence %+v", want)
		}
	}
	for _, want := range []struct {
		from, to Entity
		kind     string
	}{
		{Entity{Table, "Orders"}, Entity{Source, "$(vData)/orders.qvd"}, "from"},
		{Entity{Table, "Orders"}, Entity{Source, "lib://Warehouse/customers.qvd"}, "from"},
		{Entity{Table, "Orders"}, Entity{Source, "lib://Archive/orders.qvd"}, "from"},
		{Entity{Table, "Daily Sales"}, Entity{Table, "Orders"}, "resident"},
		{Entity{Source, "lib://Exports/daily_sales.qvd"}, Entity{Table, "Daily Sales"}, "store"},
		{Entity{FileKind, "00_config.qvs"}, Entity{Include, "includes/common.qvs"}, "include"},
	} {
		found := false
		for _, e := range idx.Edges() {
			if e.From == want.from && e.To == want.to && e.Kind == want.kind {
				found = true
			}
		}
		if !found {
			t.Errorf("missing edge %+v; edges: %+v", want, idx.Edges())
		}
	}
}

func TestParseRecoveryAndNoPhantomSymbols(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, source, want string
		diagnostics        bool
	}{
		{"unsupported", "UNKNOWN 'LOAD Fake; $(vFake)';\nReal: LOAD ID FROM [data.qvd] (qvd);", "Real", true},
		{"malformed", "LET = ;\nReal: LOAD ID FROM [data.qvd] (qvd);", "Real", true},
		{"comments", "// Fake: LOAD bogus; $(vFake)\n/* SET vFake = 1; */\nReal: LOAD ID AUTOGENERATE 1;", "Real", false},
		{"implicit target", "JOIN LOAD ID RESIDENT Earlier;\nReal: LOAD ID AUTOGENERATE 1;", "Real", true},
		{"empty assignment", "SET vEmpty =;", "vEmpty", false},
		{"bare path", "Real: LOAD ID FROM lib://Data/orders.qvd (qvd);", "Real", false},
		{"unicode", "[Vendas Diárias]: LOAD [Preço] AS Valor AUTOGENERATE 1;", "Vendas Diárias", false},
		{"utf8 BOM and CRLF", "\uFEFFSET vRoot = lib://Data;\r\nReal: LOAD ID FROM [x.qvd] (qvd);\r\n", "Real", false},
		{"unsupported control body", "FOR vFile IN FileList('*.qvd')\nReal: LOAD ID FROM [x.qvd] (qvd);\nNEXT vFile\n", "Real", true},
		{"unsupported conditional body", "IF 1 = 1 THEN\nReal: LOAD ID FROM [x.qvd] (qvd);\nEND IF\n", "Real", true},
		{"keyword prefix table", "NextTable: LOAD ID AUTOGENERATE 1;", "NextTable", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f, err := Parse(context.Background(), "input.qvs", []byte(tt.source))
			if err != nil {
				t.Fatal(err)
			}
			if (len(f.Diagnostics) > 0) != tt.diagnostics {
				t.Errorf("diagnostics: %+v", f.Diagnostics)
			}
			found := false
			for _, s := range f.Symbols {
				if s.Name == tt.want && s.Role == "definition" {
					found = true
				}
				if strings.Contains(s.Name, "Fake") || s.Name == "bogus" {
					t.Errorf("phantom symbol: %+v", s)
				}
			}
			if !found {
				t.Errorf("lost definition %q: %+v", tt.want, f)
			}
		})
	}
}

func TestRejectInvalidUTF8(t *testing.T) {
	t.Parallel()
	for _, source := range [][]byte{
		[]byte("T: LOAD ID AS [Bad\xff] FROM [x.qvd] (qvd);"),
		[]byte("// invalid comment \xc3\nT: LOAD ID AUTOGENERATE 1;"),
		{0xff, 0xfe, 'S', 0, 'E', 0, 'T', 0}, // UTF-16 BOM.
	} {
		f, err := Parse(context.Background(), "invalid.qvs", source)
		if err == nil || !strings.Contains(err.Error(), "invalid.qvs") || !strings.Contains(err.Error(), "UTF-8") {
			t.Errorf("invalid encoding must fail with path and encoding: %v", err)
		}
		if len(f.Symbols) != 0 || len(f.Edges) != 0 || len(f.Loads) != 0 {
			t.Error("invalid source produced index facts")
		}
	}
}

func TestAliasIsDefinitionAndFunctionIsNotField(t *testing.T) {
	t.Parallel()
	f, err := Parse(context.Background(), "a.qvs", []byte("T: LOAD Sum(Amount) AS Revenue, CustomerID FROM [x.qvd] (qvd);"))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, s := range f.Symbols {
		if s.Kind == Field {
			got[s.Role+":"+s.Name] = true
		}
	}
	for _, key := range []string{"definition:Revenue", "definition:CustomerID", "reference:Amount", "reference:CustomerID"} {
		if !got[key] {
			t.Errorf("missing %s in %v", key, got)
		}
	}
	if got["definition:Amount"] || got["reference:Sum"] {
		t.Fatalf("incorrect fields: %v", got)
	}
}

func TestQueriesCompileAndCapture(t *testing.T) {
	t.Parallel()
	language := sitter.NewLanguage(grammar.Language())
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(language); err != nil {
		t.Fatal(err)
	}
	src := []byte("SET vRoot = lib://Data;\nT: LOAD Raw AS Alias FROM [$(vRoot)/x.qvd] (qvd);\nSTORE T INTO [out.qvd] (qvd);")
	tree := parser.Parse(src, nil)
	defer tree.Close()
	if tree.RootNode().HasError() {
		t.Fatal(tree.RootNode().ToSexp())
	}
	for _, name := range []string{"tags", "definitions", "references", "highlights"} {
		t.Run(name, func(t *testing.T) {
			scm, err := os.ReadFile("../../tree-sitter-qlik/queries/" + name + ".scm")
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
			matches := cursor.Matches(query, tree.RootNode(), src)
			captures := map[string][]string{}
			for m := matches.Next(); m != nil; m = matches.Next() {
				for _, c := range m.Captures {
					key := query.CaptureNames()[c.Index]
					captures[key] = append(captures[key], c.Node.Utf8Text(src))
				}
			}
			if len(captures) == 0 {
				t.Fatal("no captures")
			}
			if name == "tags" {
				for key, want := range map[string]string{"name.definition.variable": "vRoot", "name.definition.table": "T", "name.definition.field": "Alias", "name.reference.variable": "vRoot", "name.reference.source": "[$(vRoot)/x.qvd]"} {
					if !strings.Contains(strings.Join(captures[key], "|"), want) {
						t.Errorf("%s missing %q: %v", key, want, captures)
					}
				}
				for _, field := range captures["name.definition.field"] {
					if field == "Raw" {
						t.Error("aliased input incorrectly captured as definition")
					}
				}
			}
		})
	}
}

func TestScanSelectionAndCancellation(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, name := range []string{"a.QVS", "nested/b.qvs", "node_modules/skip.qvs", ".git/skip.qvs", "note.txt"} {
		full := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("SET v = 1;"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(root, "a.QVS"), filepath.Join(root, "link.qvs")); err != nil {
		t.Fatal(err)
	}
	idx, err := Scan(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Files) != 2 || idx.Files[0].Path != "a.QVS" || idx.Files[1].Path != "nested/b.qvs" {
		t.Fatalf("selected files: %+v", idx.Files)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Scan(ctx, root); err == nil {
		t.Error("canceled scan succeeded")
	}
	if _, err := Parse(ctx, "a.qvs", []byte("SET x = 1;")); err == nil {
		t.Error("canceled parse succeeded")
	}
	if _, err := Scan(context.Background(), filepath.Join(root, "missing")); err == nil {
		t.Error("missing root succeeded")
	}
	if _, err := Scan(context.Background(), filepath.Join(root, "a.QVS")); err == nil {
		t.Error("file root succeeded")
	}
}

func TestSelfReferenceAndAnonymousLoadIdentities(t *testing.T) {
	t.Parallel()
	f, err := Parse(context.Background(), "same.qvs", []byte("LET vSelf = $(vSelf) + 1; LOAD ID FROM [a.qvd] (qvd); LOAD ID FROM [b.qvd] (qvd);"))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", f.Diagnostics)
	}
	if len(f.Edges) != 3 {
		t.Fatalf("edges: %+v", f.Edges)
	}
	if f.Edges[0].From != f.Edges[0].To || f.Edges[0].Kind != "variable" {
		t.Errorf("lost self-reference: %+v", f.Edges[0])
	}
	if f.Edges[1].From == f.Edges[2].From {
		t.Errorf("same-line anonymous loads share identity: %+v", f.Edges)
	}
}
