package qlik

import (
	"context"
	"os"
	"testing"
)

func TestBalancedLiteralFragmentsPreserveFollowingStatements(t *testing.T) {
	t.Parallel()
	source, err := os.ReadFile("../../testdata/recovery-balanced.qvs")
	if err != nil {
		t.Fatal(err)
	}
	f, err := Parse(context.Background(), "recovery.qvs", source)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"Recovered": 4, "Accepted": 5, "config.qvs": 3, "items.qvd": 6}
	for _, s := range f.Symbols {
		if s.Name == "vGhost" || s.Name == "Imaginary" {
			t.Errorf("phantom occurrence from unsupported literal: %+v", s)
		}
		if line, ok := want[s.Name]; ok {
			if s.Line != line {
				t.Errorf("%s line = %d, want %d", s.Name, s.Line, line)
			}
			delete(want, s.Name)
		}
	}
	if len(want) != 0 {
		t.Errorf("lost following facts: %v; symbols=%+v", want, f.Symbols)
	}
	if len(f.Diagnostics) != 2 {
		t.Fatalf("want two bounded literal diagnostics, got %+v", f.Diagnostics)
	}
	for i, d := range f.Diagnostics {
		if d.Code != "unsupported-literal" || d.Line != i+1 {
			t.Errorf("unexpected diagnostic: %+v", d)
		}
	}
}

func TestQuotedLabelAndLiteralExpressionRemainSupported(t *testing.T) {
	t.Parallel()
	f, err := Parse(context.Background(), "quoted.qvs", []byte(`"Quarterly Totals": LOAD 'active' AS Status AUTOGENERATE 1;`))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Diagnostics) != 0 {
		t.Fatalf("valid label or expression treated as stray literal: %+v", f.Diagnostics)
	}
	for _, s := range f.Symbols {
		if s.Kind == Table && s.Role == "definition" && s.Name == "Quarterly Totals" {
			return
		}
	}
	t.Fatalf("quoted table label lost: %+v", f.Symbols)
}

func TestLiteralFragmentBreaksPrecedingLOAD(t *testing.T) {
	t.Parallel()
	f, err := Parse(context.Background(), "boundary.qvs", []byte("First: LOAD ID;\n'broken statement'\nLOAD ID FROM [items.qvd] (qvd);"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range f.Edges {
		if e.Kind == "preceding" {
			t.Errorf("preceding edge crossed unsupported fragment: %+v", e)
		}
	}
	if len(f.Loads) != 2 {
		t.Errorf("LOADs lost around unsupported fragment: %+v", f.Loads)
	}
}
