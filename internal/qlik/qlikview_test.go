package qlik

import (
	"os"
	"strings"
	"testing"

	grammar "github.com/noesiscarloslisboa/qlik-code-intel/tree-sitter-qlik/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func TestCompositeVariableNamesAndNestedExpansions(t *testing.T) {
	t.Parallel()
	source := "LET vRun = 2;\nLET vMeasure_$(vRun) = 42;\nSET vLabel_$(vRun)_tail = '$(vMeasure_$(vRun))';\nBatch_$(vRun): LOAD Raw_$(vRun) AS Output_$(vRun)_tail FROM [C:\\Data\\$(vFolder)\\input.qvd] (qvd);\nCopy: LOAD * RESIDENT Batch_$(vRun);\nDollar$: LOAD Amount$ AS Output$ AUTOGENERATE 1;"
	f := parseAccuracy(t, source)
	if len(f.Diagnostics) != 0 {
		t.Fatalf("composite names diagnosed: %+v", f.Diagnostics)
	}
	want := map[string]bool{
		"definition:variable:vMeasure_$(vRun)":    false,
		"definition:variable:vLabel_$(vRun)_tail": false,
		"reference:variable:vMeasure_$(vRun)":     false,
		"definition:table:Batch_$(vRun)":          false,
		"reference:table:Batch_$(vRun)":           false,
		"reference:field:Raw_$(vRun)":             false,
		"definition:field:Output_$(vRun)_tail":    false,
		"definition:table:Dollar$":                false,
		"reference:field:Amount$":                 false,
		"definition:field:Output$":                false,
	}
	var targetNameRefs int
	for _, s := range f.Symbols {
		key := s.Role + ":" + string(s.Kind) + ":" + s.Name
		if _, ok := want[key]; ok {
			want[key] = true
		}
		if s.Kind == Variable && s.Role == "reference" && s.Name == "vRun" && s.Line == 2 {
			targetNameRefs++
		}
	}
	for key, found := range want {
		if !found {
			t.Errorf("missing %s", key)
		}
	}
	if targetNameRefs != 1 {
		t.Errorf("assignment target expansion references=%d", targetNameRefs)
	}
}

func TestCompositeNameQueriesCaptureWholeNamesAndNestedReferences(t *testing.T) {
	t.Parallel()
	source := []byte("LET vMetric_$(vRun) = 42;\nLET vCopy = '$(vMetric_$(vRun))';\nBatch_$(vRun): LOAD Raw_$(vRun) AS Output_$(vRun)_tail AUTOGENERATE 1;")
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
			got := map[string]map[string]int{}
			for m := matches.Next(); m != nil; m = matches.Next() {
				for _, c := range m.Captures {
					capture := query.CaptureNames()[c.Index]
					if got[capture] == nil {
						got[capture] = map[string]int{}
					}
					got[capture][c.Node.Utf8Text(source)]++
				}
			}
			want := map[string]string{}
			if file != "references" {
				want["name.definition.variable"] = "vMetric_$(vRun)"
				want["name.definition.table"] = "Batch_$(vRun)"
				want["name.definition.field"] = "Output_$(vRun)_tail"
			}
			if file != "definitions" {
				want["name.reference.variable"] = "vMetric_$(vRun)"
				want["name.reference.field"] = "Raw_$(vRun)"
				if got["name.reference.variable"]["vRun"] != 5 {
					t.Errorf("nested/name expansion captures: %v", got)
				}
			}
			for capture, raw := range want {
				if got[capture][raw] != 1 {
					t.Errorf("whole name %s %q missing/duplicated: %v", capture, raw, got)
				}
			}
		})
	}
}

func TestAdjacentExpansionsAndNumericNameFragments(t *testing.T) {
	t.Parallel()
	source := []byte("LET $(vPrefix)$(vSuffix) = 1;\nLET vMetric_$(vRun)2$(vPart)_tail = 42;\nBatch_$(vRun)$(vPart): LOAD Raw_$(vRun)2$(vPart) AS Output_$(vRun)$(vPart) AUTOGENERATE 1;")
	f := parseAccuracy(t, string(source))
	if len(f.Diagnostics) != 0 {
		t.Fatalf("adjacent expansions diagnosed: %+v", f.Diagnostics)
	}
	for _, name := range []string{"$(vPrefix)$(vSuffix)", "vMetric_$(vRun)2$(vPart)_tail", "Batch_$(vRun)$(vPart)", "Output_$(vRun)$(vPart)"} {
		found := false
		for _, s := range f.Symbols {
			if s.Role == "definition" && s.Name == name {
				found = true
			}
		}
		if !found {
			t.Errorf("composed definition lost: %s", name)
		}
	}
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(grammar.Language())); err != nil {
		t.Fatal(err)
	}
	tree := parser.Parse(source, nil)
	defer tree.Close()
	walk(tree.RootNode(), func(n *sitter.Node) {
		if n.Kind() == "name_fragment" && n.StartByte() == n.EndByte() {
			t.Error("adjacent expansions created an empty literal name fragment")
		}
	})
}

func TestFilesystemPathsAndFormatDollars(t *testing.T) {
	t.Parallel()
	source := "Drive: LOAD ID FROM C:\\Data\\sales.qvd (qvd);\nNetwork: LOAD ID FROM [\\\\server\\share\\sales.qvd] (qvd);\nRelative: LOAD ID FROM ..\\archive\\*.qvd (qvd);\nSheet: LOAD ID FROM excel\\items.xls (biff, embedded labels, table is Sheet1$);\nDynamic: LOAD ID FROM $(vRoot)\\items.xls (biff, table is $(vSheet)$);\nSTORE Drive INTO [D:\\Exports\\sales.qvd] (qvd);\n$(Include=..\\config\\setup.qvs);"
	f := parseAccuracy(t, source)
	if len(f.Diagnostics) != 0 {
		t.Fatalf("filesystem/format diagnostics: %+v", f.Diagnostics)
	}
	want := map[string]bool{`C:\Data\sales.qvd`: false, `\\server\share\sales.qvd`: false, `..\archive\*.qvd`: false, `excel\items.xls`: false, `$(vRoot)\items.xls`: false, `D:\Exports\sales.qvd`: false, `..\config\setup.qvs`: false}
	for _, s := range f.Symbols {
		if _, ok := want[s.Name]; ok {
			want[s.Name] = true
		}
		if s.Kind == Variable && s.Name != "vRoot" && s.Name != "vSheet" {
			t.Errorf("format dollar became variable: %+v", s)
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("path changed/lost: %q", name)
		}
	}
}

func TestCompositeLabelRecoversAfterUnsupportedBody(t *testing.T) {
	t.Parallel()
	for _, label := range []string{"Batch_$(vRun)", "Run_$(vRun)_tail", "$(vTable)_tail"} {
		f := parseAccuracy(t, "UNKNOWN unfinished\n"+label+": LOAD ID AUTOGENERATE 1;")
		if len(f.Loads) != 1 || f.Loads[0].Owner != label || f.Loads[0].Line != 2 {
			t.Errorf("composite recovery lost: %+v", f)
		}
		for _, d := range f.Diagnostics {
			if strings.Contains(d.Code, "syntax") {
				t.Errorf("composite label recovery error: %+v", d)
			}
		}
	}
}
