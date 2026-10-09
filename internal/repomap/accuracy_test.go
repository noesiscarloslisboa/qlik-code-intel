package repomap

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/noesiscarloslisboa/qlik-code-intel/internal/qlik"
)

func accuracyFile(t *testing.T, name, source string) qlik.File {
	t.Helper()
	f, err := qlik.Parse(context.Background(), name, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Diagnostics) != 0 {
		t.Fatalf("%s: %+v", name, f.Diagnostics)
	}
	return f
}

func TestAccuracyFocusedCompactMapKeepsMatchingField(t *testing.T) {
	t.Parallel()
	var src strings.Builder
	src.WriteString("Large: LOAD ")
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&src, "Field%d, ", i)
	}
	src.WriteString("CriticalField FROM [huge.qvd] (qvd);")
	idx := &qlik.Index{Files: []qlik.File{accuracyFile(t, "large.qvs", src.String())}}
	for _, budget := range []int{110, 150, 256} {
		t.Run(fmt.Sprint(budget), func(t *testing.T) {
			got := Map(idx, "CriticalField", budget)
			if !strings.Contains(got, "CriticalField") || !strings.Contains(got, "table Large") {
				t.Errorf("matching field lost:\n%s", got)
			}
			if EstimateTokens(got) > budget {
				t.Errorf("budget overflow: %d > %d", EstimateTokens(got), budget)
			}
		})
	}
}

func TestAccuracyMapStrongestMatchBeatsManyPartialMatches(t *testing.T) {
	t.Parallel()
	var src strings.Builder
	src.WriteString("Target: LOAD ID AUTOGENERATE 1;\nTargetBackup: LOAD ")
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&src, "Field%d, ", i)
	}
	src.WriteString("FinalField AUTOGENERATE 1;")
	idx := &qlik.Index{Files: []qlik.File{accuracyFile(t, "rank.qvs", src.String())}}
	got := Map(idx, "Target", 4096)
	if !strings.HasPrefix(got, "rank.qvs:1 table Target |") {
		t.Fatalf("partial matches outranked exact table:\n%s", got)
	}
}

func TestAccuracyContextDoesNotExpandUnrelatedIncludes(t *testing.T) {
	t.Parallel()
	idx := &qlik.Index{Files: []qlik.File{
		accuracyFile(t, "main-revenue.qvs", "T: LOAD Value AS Revenue AUTOGENERATE 1;\n$(Include=unrelated.qvs);"),
		accuracyFile(t, "unrelated.qvs", "SET vNoise = 'unrelated context';"),
	}}
	got := Context(idx, "Revenue", 2048)
	if !strings.Contains(got, "Revenue") {
		t.Fatalf("missing relevant source:\n%s", got)
	}
	if strings.Contains(got, "vNoise") || strings.Contains(got, "Include=") {
		t.Errorf("unrelated include filled context:\n%s", got)
	}
	if included := Context(idx, "unrelated.qvs", 2048); !strings.Contains(included, "vNoise") {
		t.Errorf("explicit include lookup lost target:\n%s", included)
	}
	if byFile := Context(idx, "main-revenue.qvs", 2048); !strings.Contains(byFile, "vNoise") {
		t.Errorf("explicit file lookup lost include source:\n%s", byFile)
	}
}

func TestAccuracyFocusedMapKeepsInputReferenceAndVariable(t *testing.T) {
	t.Parallel()
	var src strings.Builder
	src.WriteString("T: LOAD ")
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&src, "Field%d, ", i)
	}
	src.WriteString("Date#(RawDate, 'YYYY-MM-DD') AS Date, $(vCutoff) AS Cutoff FROM [raw.qvd] (qvd);")
	idx := &qlik.Index{Files: []qlik.File{accuracyFile(t, "focus.qvs", src.String())}}
	for _, query := range []string{"RawDate", "vCutoff"} {
		for budget := 100; budget <= 256; budget++ {
			got := Map(idx, query, budget)
			if !strings.Contains(got, query) || EstimateTokens(got) > budget {
				t.Fatalf("query %s budget %d lost match or overflowed:\n%s", query, budget, got)
			}
		}
	}
}

func TestAccuracyPrecedingWildcardMapAndDirectContext(t *testing.T) {
	t.Parallel()
	idx := &qlik.Index{Files: []qlik.File{accuracyFile(t, "wild.qvs", "Result: LOAD *;\nLOAD *;\nLOAD ID FROM [x.qvd] (qvd);\n")}}
	got := Map(idx, "", 2048)
	for _, want := range []string{"wild.qvs:1 table Result", "wild.qvs:2 load", "wild.qvs:3 load", "preceding @wild.qvs:2:1", "preceding @wild.qvs:3:1"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q:\n%s", want, got)
		}
	}
	context := Context(idx, "Result", 2048)
	if !strings.Contains(context, "2 | LOAD *;") {
		t.Errorf("anonymous direct input stage missing:\n%s", context)
	}
	if strings.Contains(context, "x.qvd") {
		t.Errorf("context recursively expanded dependencies:\n%s", context)
	}
}
