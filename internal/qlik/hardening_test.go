package qlik

import (
	"context"
	"os"
	"strings"
	"testing"

	grammar "github.com/noesiscarloslisboa/qlik-code-intel/tree-sitter-qlik/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func TestUnfinishedUnsupportedStatementPreservesLabeledLOAD(t *testing.T) {
	t.Parallel()
	for _, label := range []string{"Recovered", "[Recovered]", `"Recovered"`, "`Recovered`", "$(vTable)", "Vendas", "[Vendas Diárias]", "R", "RE", "REM_TABLE"} {
		t.Run(label, func(t *testing.T) {
			source := "UNKNOWN unfinished header\r\n\t" + label + ":\r\nNOCONCATENATE LOAD ID FROM [items.qvd] (qvd);"
			f := parseAccuracy(t, source)
			if len(f.Loads) != 1 || f.Loads[0].Line != 2 || f.Loads[0].Column != 2 {
				t.Fatalf("labeled LOAD lost or misplaced: %+v", f.Loads)
			}
			if f.Loads[0].Owner != Name(label) {
				t.Errorf("owner=%q want %q", f.Loads[0].Owner, Name(label))
			}
			codes := map[string]bool{}
			for _, d := range f.Diagnostics {
				codes[d.Code] = true
			}
			if !codes["unsupported"] || !codes["unterminated-statement"] || codes["syntax-error"] {
				t.Errorf("boundary diagnostics: %+v", f.Diagnostics)
			}
		})
	}
}

func TestRecoveryScannerPreservesCommentSeparatedLabel(t *testing.T) {
	t.Parallel()
	for _, separator := range []string{" /* comment */ ", " // comment\n", "\n/* comment\n continues */\n", " REM ignored; ", "\f\v"} {
		f := parseAccuracy(t, "T"+separator+": LOAD ID AUTOGENERATE 1;")
		if len(f.Diagnostics) != 0 || len(f.Loads) != 1 || f.Loads[0].Owner != "T" {
			t.Errorf("separator %q hid valid label: %+v", separator, f)
		}
	}
}

func TestUnsupportedBodyRemainsOpaque(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		"'text;\nGhost: LOAD Fake FROM [fake.qvd];'",
		"\"text;\nGhost: LOAD Fake FROM [fake.qvd];\"",
		"`text;\nGhost: LOAD Fake FROM [fake.qvd];`",
		"[text;\nGhost: LOAD Fake FROM 'fake.qvd';]",
		"/* ;\nGhost: LOAD Fake FROM [fake.qvd]; */ raw",
		"// ; Ghost: LOAD Fake FROM [fake.qvd];\nraw",
		"$(vGhost, 'text;\nGhost: LOAD Fake FROM [fake.qvd];')",
		"'outer $(vGhost, 'inner;\nGhost: LOAD Fake;') end'",
		"raw\nlib://path/to/file.qvd",
		"raw REM comment\nGhost: LOAD Fake FROM [fake.qvd]; tail",
		"\nREM comment\nGhost: LOAD Fake FROM [fake.qvd]; tail",
	} {
		t.Run(body, func(t *testing.T) {
			f := parseAccuracy(t, "UNKNOWN "+body+";\nReal: LOAD ID AUTOGENERATE 1;")
			if len(f.Loads) != 1 || f.Loads[0].Owner != "Real" {
				t.Fatalf("opaque text extracted or following LOAD lost: %+v", f.Loads)
			}
			for _, s := range f.Symbols {
				if s.Name == "Ghost" || s.Name == "Fake" || s.Name == "vGhost" || s.Name == "fake.qvd" {
					t.Errorf("phantom symbol: %+v", s)
				}
			}
		})
	}
}

func TestUnsupportedRecoveryIsConservative(t *testing.T) {
	t.Parallel()
	for _, source := range []string{
		"UNKNOWN text LOAD ID FROM [fake.qvd] (qvd);",
		"UNKNOWN text\nLOAD ID FROM [fake.qvd] (qvd);",
		"UNKNOWN text SameLine: LOAD ID FROM [fake.qvd] (qvd);",
	} {
		f := parseAccuracy(t, source)
		if len(f.Symbols) != 0 || len(f.Loads) != 0 {
			t.Errorf("unlabeled/same-line body became facts: %+v", f)
		}
	}
	for _, source := range []string{"UNKNOWN", "UNKNOWN text without a terminator"} {
		f := parseAccuracy(t, source)
		found := false
		for _, d := range f.Diagnostics {
			found = found || d.Code == "unterminated-statement"
		}
		if !found {
			t.Errorf("EOF terminator diagnostic missing: %+v", f.Diagnostics)
		}
	}
}

func TestUnsupportedDeepMacroText(t *testing.T) {
	t.Parallel()
	source := "UNKNOWN " + strings.Repeat("$(", 2000) + "'Ghost: LOAD Fake;'" + strings.Repeat(")", 2000) + ";\nReal: LOAD ID AUTOGENERATE 1;"
	f := parseAccuracy(t, source)
	if len(f.Loads) != 1 || f.Loads[0].Owner != "Real" {
		t.Fatalf("deep protected body lost its boundary: %+v", f.Loads)
	}
}

func TestBacktickNamesAndNormalization(t *testing.T) {
	t.Parallel()
	source := "SET `vRoot` = lib://Warehouse;\nLET vTotal = `vBase` + 1;\n`Quarter One`:\nLOAD `Raw Amount` AS `Net Amount`, 1 AS `$(vAlias)`\nFROM `lib://$(vRoot)/items.qvd` (qvd);\nSnapshot: LOAD * RESIDENT `Quarter One`;\nSTORE `Quarter One` INTO `lib://Out/result.qvd` (qvd);\n$(Include=`config/$(vRegion).qvs`);\n\"Quoted\"\" Table\": LOAD \"Raw\"\" Amount\" AS \"Net\"\" Amount\" AUTOGENERATE 1;\n\"Tick` Table\": LOAD \"Raw` Amount\" AS [Net` Amount] AUTOGENERATE 1;"
	f, err := Parse(context.Background(), "backticks.qvs", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Diagnostics) != 0 {
		t.Fatalf("backtick diagnostics: %+v", f.Diagnostics)
	}
	want := map[string]bool{
		"definition:variable:vRoot": false, "reference:variable:vBase": false,
		"definition:table:Quarter One": false, "reference:table:Quarter One": false,
		"reference:field:Raw Amount": false, "definition:field:Net Amount": false,
		"definition:field:$(vAlias)": false, "reference:variable:vAlias": false,
		"reference:source:lib://$(vRoot)/items.qvd": false,
		"reference:include:config/$(vRegion).qvs":   false, "reference:variable:vRegion": false,
		"definition:table:Quoted\" Table": false, "reference:field:Raw\" Amount": false,
		"definition:field:Net\" Amount": false, "definition:table:Tick` Table": false,
		"reference:field:Raw` Amount": false, "definition:field:Net` Amount": false,
	}
	for _, s := range f.Symbols {
		key := s.Role + ":" + string(s.Kind) + ":" + s.Name
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for key, found := range want {
		if !found {
			t.Errorf("missing %s", key)
		}
	}
	if got := Name("`a b`"); got != "a b" {
		t.Errorf("backtick name=%q", got)
	}
}

func TestDoubledBackticksAreNotAssumedValidEscapes(t *testing.T) {
	t.Parallel()
	f := parseAccuracy(t, "T: LOAD `Bad``Name` AUTOGENERATE 1;\nReal: LOAD ID AUTOGENERATE 1;")
	if len(f.Diagnostics) == 0 {
		t.Fatal("undocumented doubled-backtick escape was silently accepted")
	}
}

func TestHardeningQueriesCaptureNamesWithoutOpaqueReferences(t *testing.T) {
	t.Parallel()
	source := []byte("UNKNOWN '$(vGhost);';\n`T One`: LOAD `Raw Value` AS `Net Value` FROM `$(vRoot)/x.qvd` (qvd);\nSTORE `T One` INTO `out.qvd` (qvd);\nLET vTotal = `vBase`;\n")
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
	for _, file := range []string{"tags", "definitions", "references"} {
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
			matches := cursor.Matches(query, tree.RootNode(), source)
			got := map[string]map[string]bool{}
			for m := matches.Next(); m != nil; m = matches.Next() {
				for _, c := range m.Captures {
					capture, raw := query.CaptureNames()[c.Index], c.Node.Utf8Text(source)
					if strings.Contains(raw, "vGhost") {
						t.Errorf("%s captured opaque text %q", capture, raw)
					}
					if got[capture] == nil {
						got[capture] = map[string]bool{}
					}
					got[capture][raw] = true
				}
			}
			want := map[string]string{}
			if file != "references" {
				want["name.definition.table"] = "`T One`"
				want["name.definition.field"] = "`Net Value`"
				want["name.definition.source"] = "`out.qvd`"
			}
			if file != "definitions" {
				want["name.reference.field"] = "`Raw Value`"
				want["name.reference.table"] = "`T One`"
				want["name.reference.source"] = "`$(vRoot)/x.qvd`"
				want["name.reference.variable"] = "`vBase`"
				if !got["name.reference.variable"]["vRoot"] {
					t.Errorf("missing vRoot expansion capture: %v", got)
				}
			}
			for capture, raw := range want {
				if !got[capture][raw] {
					t.Errorf("missing %s %q: %v", capture, raw, got)
				}
			}
		})
	}
}

func TestUnfinishedUnsupportedBreaksPrecedingLOAD(t *testing.T) {
	t.Parallel()
	f := parseAccuracy(t, "First: LOAD ID;\nUNKNOWN unfinished\nNextTable: LOAD ID FROM [next.qvd] (qvd);")
	if len(f.Loads) != 2 {
		t.Fatalf("LOADs missing: %+v", f.Loads)
	}
	for _, e := range f.Edges {
		if e.Kind == "preceding" {
			t.Errorf("unsupported boundary became preceding input: %+v", e)
		}
	}
	found := false
	for _, d := range f.Diagnostics {
		found = found || d.Code == "unresolved-source"
	}
	if !found {
		t.Errorf("missing unresolved first LOAD: %+v", f.Diagnostics)
	}
}

func TestRecoveryScannerIncrementalTerminatorEdits(t *testing.T) {
	t.Parallel()
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(grammar.Language())); err != nil {
		t.Fatal(err)
	}
	prefix := "UNKNOWN '$(vGhost)'"
	source := []byte(prefix + "\n`Recovered`: LOAD ID AUTOGENERATE 1;\n")
	tree := parser.Parse(source, nil)
	defer func() { tree.Close() }()
	// Add, then remove, the missing semicolon using the previous edited tree.
	for _, insert := range []bool{true, false} {
		at := uint(len(prefix))
		edit := sitter.InputEdit{StartByte: at, OldEndByte: at, NewEndByte: at,
			StartPosition:  sitter.Point{Row: 0, Column: at},
			OldEndPosition: sitter.Point{Row: 0, Column: at},
			NewEndPosition: sitter.Point{Row: 0, Column: at}}
		if insert {
			edit.NewEndByte++
			edit.NewEndPosition.Column++
			source = []byte(prefix + ";\n`Recovered`: LOAD ID AUTOGENERATE 1;\n")
		} else {
			edit.OldEndByte++
			edit.OldEndPosition.Column++
			source = []byte(prefix + "\n`Recovered`: LOAD ID AUTOGENERATE 1;\n")
		}
		tree.Edit(&edit)
		next := parser.Parse(source, tree)
		tree.Close()
		tree = next
		fresh := parser.Parse(source, nil)
		if tree.RootNode().HasError() || tree.RootNode().ToSexp() != fresh.RootNode().ToSexp() {
			t.Errorf("incremental tree differs from fresh parse: %s / %s", tree.RootNode().ToSexp(), fresh.RootNode().ToSexp())
		}
		fresh.Close()
		load := tree.RootNode().NamedChild(1)
		if load.Kind() != "load_statement" || load.StartPosition().Row != 1 || load.StartByte() != edit.NewEndByte+1 {
			t.Errorf("incremental recovery changed LOAD range: %s at %v byte %d", load.Kind(), load.StartPosition(), load.StartByte())
		}
	}
}
