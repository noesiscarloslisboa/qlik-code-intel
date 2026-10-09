package repomap

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/noesiscarloslisboa/qlik-code-intel/internal/qlik"
)

func TestHardeningRetrievalQuestions(t *testing.T) {
	t.Parallel()
	idx, err := qlik.Scan(context.Background(), "../../testdata/hardening")
	if err != nil {
		t.Fatal(err)
	}
	for _, question := range []struct {
		query, kind, role, name, path string
		line                          int
	}{
		{"Quarter Sales", "table", "definition", "Quarter Sales", "10_sales.qvs", 1},
		{"Net\" Amount", "field", "definition", "Net\" Amount", "10_sales.qvs", 2},
		{"Raw Amount", "field", "reference", "Raw Amount", "10_sales.qvs", 2},
		{"vRoot", "variable", "definition", "vRoot", "00_config.qvs", 2},
		{"sales.qvd", "qvd", "reference", "$(vRoot)/sales.qvd", "10_sales.qvs", 3},
		{"revenue.qvd", "qvd", "definition", "lib://Exports/revenue.qvd", "10_sales.qvs", 6},
		{"Recovered Table", "table", "definition", "Recovered Table", "20_recovery.qvs", 3},
		{"Recovered ID", "field", "definition", "Recovered ID", "20_recovery.qvs", 4},
	} {
		t.Run(question.query, func(t *testing.T) {
			got := Find(idx, Search{Query: question.query, Kind: question.kind, Role: question.role})
			if len(got) == 0 || got[0].Name != question.name || got[0].Path != question.path || got[0].Line != question.line {
				t.Fatalf("wrong best occurrence: %+v", got)
			}
		})
	}
	for _, ghost := range []string{"vGhost", "Ghost", "Fake", "fake.qvd"} {
		if got := Find(idx, Search{Query: ghost}); len(got) != 0 {
			t.Errorf("opaque symbol %q retrieved: %+v", ghost, got)
		}
	}
	up := Dependencies(idx, "Summary", "table", "upstream")
	if len(up) != 1 || up[0].To.Name != "Quarter Sales" || up[0].Kind != "resident" || up[0].Line != 5 {
		t.Errorf("normalized resident input: %+v", up)
	}
	output := Dependencies(idx, "revenue.qvd", "qvd", "upstream")
	if len(output) != 1 || output[0].To.Name != "Summary" || output[0].Kind != "store" {
		t.Errorf("output dependency: %+v", output)
	}
	for _, query := range []string{"Net\" Amount", "Recovered ID", "sales.qvd"} {
		for _, budget := range []int{0, 32, 128, 256, 512, 2048} {
			t.Run(fmt.Sprintf("%s/%d", query, budget), func(t *testing.T) {
				for name, render := range map[string]func(*qlik.Index, string, int) string{"map": Map, "context": Context} {
					got := render(idx, query, budget)
					if len(got) > budget || !utf8.ValidString(got) || got != render(idx, query, budget) {
						t.Errorf("%s budget/UTF-8/determinism failure: %q", name, got)
					}
					// Source context retains the escaped spelling; map names are normalized.
					match := query
					if name == "context" {
						match = strings.ReplaceAll(match, "\"", "\"\"")
					}
					if budget >= 256 && !strings.Contains(got, match) {
						t.Errorf("%s lost matching text %q:\n%s", name, match, got)
					}
				}
			})
		}
	}
	got := Context(idx, "Recovered ID", 512)
	if !strings.Contains(got, "20_recovery.qvs:3\n3 | `Recovered Table`:\n4 | LOAD `Raw ID` AS `Recovered ID` FROM [recovered.qvd] (qvd);\n") || strings.Contains(got, "UNKNOWN") {
		t.Errorf("recovered numbered source changed or leaked opaque text:\n%s", got)
	}
}
