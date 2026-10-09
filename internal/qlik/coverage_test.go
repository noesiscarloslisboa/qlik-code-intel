package qlik

import (
	"os"
	"testing"

	grammar "github.com/noesiscarloslisboa/qlik-code-intel/tree-sitter-qlik/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func TestEmptyLETAndOmittedFunctionArguments(t *testing.T) {
	t.Parallel()
	f := parseAccuracy(t, "LET vEmpty = ;\nLET vResult = If(vFlag, vYes,);\nResult: LOAD If(Flag, If(Other, Amount,),) AS Value, Pick(1,, Fallback) AS Choice AUTOGENERATE 1;")
	if len(f.Diagnostics) != 0 {
		t.Fatalf("empty values or omitted arguments diagnosed: %+v", f.Diagnostics)
	}
	want := map[string]int{
		"definition:variable:vEmpty": 1, "definition:variable:vResult": 1,
		"reference:variable:vFlag": 1, "reference:variable:vYes": 1,
		"definition:table:Result": 1, "definition:field:Value": 1,
		"definition:field:Choice": 1, "reference:field:Flag": 1,
		"reference:field:Other": 1, "reference:field:Amount": 1,
		"reference:field:Fallback": 1,
	}
	for _, s := range f.Symbols {
		key := s.Role + ":" + string(s.Kind) + ":" + s.Name
		want[key]--
	}
	for key, remaining := range want {
		if remaining != 0 {
			t.Errorf("unexpected occurrence count for %s: difference=%d", key, remaining)
		}
	}
	for _, malformed := range []string{
		"LET vBroken = 1 + ;", "Broken: LOAD A,,B AUTOGENERATE 1;",
		"Broken: LOAD If(Flag +, Amount,) AS Value AUTOGENERATE 1;",
	} {
		if f := parseAccuracy(t, malformed); len(f.Diagnostics) == 0 {
			t.Errorf("malformed expression accepted: %s", malformed)
		}
	}
}

func TestGroupByExpressionsPreserveComponentReferences(t *testing.T) {
	t.Parallel()
	f := parseAccuracy(t, "Totals:\nLOAD Customer & Num(Line, '000') AS Key, Sum(Amount) AS Total\nFROM [input.qvd] (qvd)\nGROUP BY Customer & Num(Line, '000'), Region;\nAfter: LOAD Key RESIDENT Totals;")
	if len(f.Diagnostics) != 0 {
		t.Fatalf("GROUP BY expression diagnosed: %+v", f.Diagnostics)
	}
	want := map[string]bool{"Customer": false, "Line": false, "Region": false}
	for _, s := range f.Symbols {
		if s.Line == 4 && s.Kind == Field {
			if _, ok := want[s.Name]; !ok || s.Role != "reference" || s.Owner != "Totals" {
				t.Errorf("wrong GROUP BY occurrence: %+v", s)
			}
			want[s.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("GROUP BY reference missing: %s", name)
		}
	}
	if len(f.Loads) != 2 || len(f.Edges) != 2 {
		t.Fatalf("surrounding LOADs/inputs lost: %+v", f)
	}
}

func TestHierarchyAndBundleExplicitSourceFacts(t *testing.T) {
	t.Parallel()
	source := "Tree:\nNoConcatenate\nHIERARCHY(NodeID, ParentID, Title, ParentTitle, Title, [Node Path], '/', Depth)\nLOAD NodeID, ParentID, Title FROM [nodes.qvd] (qvd);\nAssets:\nBUNDLE INFO LOAD FileKey, FileName FROM [assets.csv] (txt);\nIcons: BUNDLE LOAD * INLINE [ID,Path\n1,$(vAssetRoot)/icon.png\n];"
	f := parseAccuracy(t, source)
	if len(f.Diagnostics) != 0 {
		t.Fatalf("HIERARCHY/BUNDLE diagnosed: %+v", f.Diagnostics)
	}
	if len(f.Loads) != 3 || len(f.Edges) != 3 {
		t.Fatalf("LOAD ownership or direct inputs lost: %+v", f)
	}
	want := map[string]int{"reference:NodeID": 1, "reference:ParentID": 1, "reference:Title": 2, "definition:ParentTitle": 1, "definition:Node Path": 1, "definition:Depth": 1}
	var inlineRefs int
	for _, s := range f.Symbols {
		if s.Name == "Title1" || s.Name == "icon.png" || s.Name == "/" || s.Name == "HIERARCHY" || (s.Line >= 8 && s.Kind != Variable) {
			t.Errorf("invented field or opaque input reference: %+v", s)
		}
		if s.Line == 8 && s.Kind == Variable && s.Name == "vAssetRoot" && s.Role == "reference" {
			inlineRefs++
		}
		if s.Line == 3 && s.Kind == Field {
			want[s.Role+":"+s.Name]--
			if s.Owner != "Tree" || s.StartLine != 1 || s.StopLine != 4 {
				t.Errorf("prefix field lost owner/source range: %+v", s)
			}
		}
	}
	if inlineRefs != 1 {
		t.Errorf("explicit INLINE expansion references=%d", inlineRefs)
	}
	for name, remaining := range want {
		if remaining != 0 {
			t.Errorf("hierarchy role mismatch for %s: %d", name, remaining)
		}
	}
}

func TestHierarchyOptionalPositionsAndPrefixBoundaries(t *testing.T) {
	t.Parallel()
	for _, prefix := range []string{
		"Hierarchy(ID, Parent, Name)",
		"Hierarchy(ID, Parent, Name,,, 'Branch Path', $(vSep), 'Distance')",
		"Hierarchy(ID, Parent, Name, Parent_$(vRun), Name, Path_$(vRun), '/', Depth_$(vRun))",
	} {
		f := parseAccuracy(t, "Tree: "+prefix+" LOAD ID, Parent, Name FROM [nodes.qvd] (qvd);")
		if len(f.Diagnostics) != 0 || len(f.Loads) != 1 || f.Loads[0].Owner != "Tree" {
			t.Errorf("optional hierarchy positions lost: %+v", f)
		}
	}
	for _, prefix := range []string{"Hierarchy(ID, Parent)", "Hierarchy(ID, Parent, Name,,,,,, Extra)"} {
		if f := parseAccuracy(t, "Tree: "+prefix+" LOAD ID AUTOGENERATE 1;"); len(f.Diagnostics) == 0 {
			t.Errorf("invalid hierarchy argument count accepted: %s", prefix)
		}
	}
	f := parseAccuracy(t, "Result: LOAD ID;\nHierarchy(ID, Parent, Name) LOAD ID, Parent, Name FROM [nodes.qvd] (qvd);")
	for _, e := range f.Edges {
		if e.Kind == "preceding" {
			t.Errorf("prefixed input became a preceding stage: %+v", e)
		}
	}
	if len(f.Diagnostics) != 1 || f.Diagnostics[0].Code != "unresolved-source" || f.Diagnostics[0].Line != 1 {
		t.Errorf("prefix boundary diagnostic changed: %+v", f.Diagnostics)
	}
}

func TestHierarchyQueriesSeparateInputsAndOutputNames(t *testing.T) {
	t.Parallel()
	source := []byte("Tree: Hierarchy(ID, Parent, Title, 'Parent Label', Title, Path_$(vRun), $(vSep), Depth) LOAD ID, Parent, Title FROM [nodes.qvd] (qvd);")
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
			if file != "references" {
				for _, name := range []string{"'Parent Label'", "Path_$(vRun)", "Depth"} {
					if got["name.definition.field"][name] != 1 {
						t.Errorf("output definition missing/duplicated: %s: %v", name, got)
					}
				}
			}
			if file != "definitions" {
				if got["name.reference.field"]["Title"] != 3 || got["name.reference.variable"]["vRun"] != 1 || got["name.reference.variable"]["vSep"] != 1 {
					t.Errorf("prefix inputs/expansions missing: %v", got)
				}
				for _, name := range []string{"'Parent Label'", "Path_$(vRun)", "Depth"} {
					if got["name.reference.field"][name] != 0 {
						t.Errorf("output name became an input reference: %s", name)
					}
				}
			}
		})
	}
}
