package qlik

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	grammar "github.com/noesiscarloslisboa/qlik-code-intel/tree-sitter-qlik/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func parseAccuracy(t *testing.T, source string) File {
	t.Helper()
	f, err := Parse(context.Background(), "accuracy.qvs", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestAccuracyBareLETVariables(t *testing.T) {
	t.Parallel()
	source := "LET vTotal = If(vFlag, (vBase + Alt(vOffset, 0)), 'vFake') & '$(vExtra)';"
	f := parseAccuracy(t, source)
	if len(f.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", f.Diagnostics)
	}
	got := map[string]int{}
	for _, s := range f.Symbols {
		if s.Kind == Field {
			t.Errorf("LET expression produced a field: %+v", s)
		}
		if s.Kind == Variable && s.Role == "reference" {
			got[s.Name]++
		}
	}
	for _, name := range []string{"vFlag", "vBase", "vOffset", "vExtra"} {
		if got[name] != 1 {
			t.Errorf("variable %s count=%d; references=%v", name, got[name], got)
		}
	}
	if len(got) != 4 || len(f.Edges) != 4 {
		t.Errorf("unexpected references/edges: %v / %+v", got, f.Edges)
	}
	for _, e := range f.Edges {
		if e.From != (Entity{Variable, "vTotal"}) || e.To.Kind != Variable || e.Kind != "variable" {
			t.Errorf("wrong LET edge: %+v", e)
		}
	}

	parser := sitter.NewParser()
	defer parser.Close()
	language := sitter.NewLanguage(grammar.Language())
	if err := parser.SetLanguage(language); err != nil {
		t.Fatal(err)
	}
	tree := parser.Parse([]byte(source), nil)
	defer tree.Close()
	for _, file := range []string{"tags", "references"} {
		scm, err := os.ReadFile("../../tree-sitter-qlik/queries/" + file + ".scm")
		if err != nil {
			t.Fatal(err)
		}
		query, queryErr := sitter.NewQuery(language, string(scm))
		if queryErr != nil {
			t.Fatal(queryErr)
		}
		cursor := sitter.NewQueryCursor()
		matches := cursor.Matches(query, tree.RootNode(), []byte(source))
		variables := map[string]bool{}
		for m := matches.Next(); m != nil; m = matches.Next() {
			for _, capture := range m.Captures {
				name := query.CaptureNames()[capture.Index]
				if name == "name.reference.variable" {
					variables[capture.Node.Utf8Text([]byte(source))] = true
				}
				if name == "name.reference.field" {
					t.Errorf("%s query captured a LET field", file)
				}
			}
		}
		cursor.Close()
		query.Close()
		for _, name := range []string{"vFlag", "vBase", "vOffset", "vExtra"} {
			if !variables[name] {
				t.Errorf("%s query missed %s: %v", file, name, variables)
			}
		}
	}
}

func TestAccuracyInterpretationFunctions(t *testing.T) {
	t.Parallel()
	f := parseAccuracy(t, "LET vDate = Date#('2024-01-01', 'YYYY-MM-DD');\nT: LOAD Date#(RawDate, 'YYYY-MM-DD') AS Date, Num#(RawAmount, '#,##0.00') AS Amount FROM [raw.qvd] (qvd);")
	if len(f.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", f.Diagnostics)
	}
	definitions, references := map[string]bool{}, map[string]bool{}
	for _, s := range f.Symbols {
		if s.Kind != Field {
			continue
		}
		if s.Role == "definition" {
			definitions[s.Name] = true
		} else {
			references[s.Name] = true
		}
	}
	if !definitions["Date"] || !definitions["Amount"] || !references["RawDate"] || !references["RawAmount"] {
		t.Errorf("fields: defs=%v refs=%v", definitions, references)
	}
	if references["Date#"] || references["Num#"] {
		t.Errorf("functions are not fields: %v", references)
	}
}

func TestAccuracySymbolicTableNames(t *testing.T) {
	t.Parallel()
	f := parseAccuracy(t, `$(vTarget): LOAD Value AS $(vAlias) FROM [$(vRoot)/source.qvd] (qvd);
Other: LOAD * RESIDENT $(vTarget);
LEFT JOIN ($(vTarget)) LOAD ID FROM [joined.qvd] (qvd);
STORE $(vTarget) INTO [$(vRoot)/out.qvd] (qvd);`)
	if len(f.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", f.Diagnostics)
	}
	variables := map[string]bool{}
	tables, alias := 0, false
	for _, s := range f.Symbols {
		if s.Kind == Variable && s.Role == "reference" {
			variables[s.Name] = true
		}
		if s.Kind == Table && s.Name == "$(vTarget)" {
			tables++
		}
		if s.Kind == Field && s.Role == "definition" && s.Name == "$(vAlias)" && s.Owner == "$(vTarget)" {
			alias = true
		}
	}
	if tables != 4 || !alias {
		t.Errorf("symbolic names were lost: tables=%d alias=%v symbols=%+v", tables, alias, f.Symbols)
	}
	for _, name := range []string{"vTarget", "vAlias", "vRoot"} {
		if !variables[name] {
			t.Errorf("missing variable %s", name)
		}
	}
}

func TestAccuracyPrecedingLOADStageOwnership(t *testing.T) {
	t.Parallel()
	source := "Result:\nLOAD NetAmount AS Revenue;\n// intermediate stage\nLOAD Amount * Rate AS NetAmount;\nLOAD Amount, Rate FROM [raw.qvd] (qvd);\n"
	f := parseAccuracy(t, source)
	if len(f.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", f.Diagnostics)
	}
	preceding := map[string]Edge{}
	for _, e := range f.Edges {
		if e.Kind == "preceding" {
			preceding[e.From.Name] = e
		}
	}
	first, ok := preceding["Result"]
	if !ok || first.To.Name != "@accuracy.qvs:4:1" || first.StartByte != 0 || first.Line != 4 {
		t.Errorf("missing first-stage input: %+v", f.Edges)
	}
	second, ok := preceding["@accuracy.qvs:4:1"]
	if !ok || second.To.Name != "@accuracy.qvs:5:1" {
		t.Errorf("missing second-stage input: %+v", f.Edges)
	}
	for _, s := range f.Symbols {
		if s.Kind == Field && s.Role == "definition" {
			if s.Name == "Revenue" && (s.Owner != "Result" || s.Line != 2 || s.StopLine != 2) {
				t.Errorf("outer field: %+v", s)
			}
			if s.Name == "NetAmount" && s.Owner != "@accuracy.qvs:4:1" {
				t.Errorf("intermediate field promoted: %+v", s)
			}
		}
	}
	encoded, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Loads []json.RawMessage `json:"loads"`
	}
	if err := json.Unmarshal(encoded, &record); err != nil {
		t.Fatal(err)
	}
	if len(record.Loads) != 3 {
		t.Errorf("LOAD stage records=%d, want 3", len(record.Loads))
	}
}

func TestAccuracyPrecedingBoundaries(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ name, middle, next string }{
		{"assignment", "SET v = 1;\n", "LOAD ID FROM [x.qvd] (qvd);"},
		{"include", "$(Include=config.qvs);\n", "LOAD ID FROM [x.qvd] (qvd);"},
		{"unsupported", "TRACE 'boundary';\n", "LOAD ID FROM [x.qvd] (qvd);"},
		{"parse error", "LET = ;\n", "LOAD ID FROM [x.qvd] (qvd);"},
		{"control", "IF 1 = 1 THEN\n", "LOAD ID FROM [x.qvd] (qvd);\nEND IF"},
		{"new label", "", "Second: LOAD ID FROM [x.qvd] (qvd);"},
		{"new prefix", "", "JOIN (Existing) LOAD ID FROM [x.qvd] (qvd);"},
		{"end of file", "", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := parseAccuracy(t, "First: LOAD ID;\n"+tt.middle+tt.next)
			for _, e := range f.Edges {
				if e.Kind == "preceding" {
					t.Errorf("invented relationship across boundary: %+v", e)
				}
			}
			unresolved := false
			for _, d := range f.Diagnostics {
				if d.Code == "unresolved-source" && d.Line == 1 {
					unresolved = true
				}
			}
			if !unresolved {
				t.Errorf("missing unresolved-source diagnostic: %+v", f.Diagnostics)
			}
		})
	}
}

func TestAccuracyWildcardStageRecords(t *testing.T) {
	t.Parallel()
	f := parseAccuracy(t, "Result: LOAD *;\nLOAD *;\nLOAD ID FROM [x.qvd] (qvd);")
	encoded, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"loads":[`) {
		t.Errorf("wildcard stages have no source records: %s", encoded)
	}
	count := 0
	for _, e := range f.Edges {
		if e.Kind == "preceding" {
			count++
		}
	}
	if count != 2 {
		t.Errorf("preceding edges=%d, want 2", count)
	}
	for _, s := range f.Symbols {
		if s.Kind == Field && s.Role == "definition" && s.Owner != "@accuracy.qvs:3:1" {
			t.Errorf("wildcard fabricated output field: %+v", s)
		}
	}
}

func TestAccuracyJOINCannotPrecedeLOAD(t *testing.T) {
	t.Parallel()
	f := parseAccuracy(t, "JOIN (T) LOAD ID;\nLOAD ID FROM [x.qvd] (qvd);")
	for _, e := range f.Edges {
		if e.Kind == "preceding" {
			t.Errorf("JOIN cannot precede a LOAD: %+v", e)
		}
	}
	if len(f.Diagnostics) != 1 || f.Diagnostics[0].Code != "unresolved-source" {
		t.Errorf("missing JOIN source diagnostic: %+v", f.Diagnostics)
	}
}

func TestAccuracyUnaryNOTUsesOperatorInsteadOfField(t *testing.T) {
	t.Parallel()
	for _, operator := range []string{"NOT", "not", "Not"} {
		t.Run(operator, func(t *testing.T) {
			f := parseAccuracy(t, "Stock: LOAD SKU, "+operator+" IsNull(UnitCost) AS HasCost FROM [stock.qvd] (qvd) WHERE "+operator+" IsNull(SKU) AND Active = 1;\nLET vReady = "+operator+" IsNull(vInput);\nLET vCopy = Notable;\nNotable: LOAD Notable AUTOGENERATE 1;")
			if len(f.Diagnostics) != 0 {
				t.Fatalf("valid unary NOT produced diagnostics: %+v", f.Diagnostics)
			}
			input := false
			for _, s := range f.Symbols {
				if strings.EqualFold(s.Name, "not") {
					t.Errorf("operator indexed as a name: %+v", s)
				}
				if s.Kind == Variable && s.Role == "reference" && s.Name == "vInput" {
					input = true
				}
			}
			if !input {
				t.Errorf("LET variable reference lost: %+v", f.Symbols)
			}
		})
	}
}
