package repomap

import (
	"context"
	"strings"
	"testing"

	"github.com/noesiscarloslisboa/qlik-code-intel/internal/qlik"
)

func TestOperationsMapAndContext(t *testing.T) {
	t.Parallel()
	source := "RENAME FIELDS Alpha TO First, Beta TO Second, Gamma TO Target;\n" +
		"RENAME TABLE Stage TO Published;\n" +
		"RENAME FIELDS USING Names;\n" +
		"DROP FIELDS Alpha, Beta, Target FROM Published, Archive;\n" +
		"DROP MAPPING TABLE Names;\n" +
		"TRACE plain words $(vBatch);\nTRACE no symbols;"
	f, err := qlik.Parse(context.Background(), "ops.qvs", []byte(source))
	if err != nil || len(f.Diagnostics) != 0 {
		t.Fatalf("parse: %v, diagnostics: %+v", err, f.Diagnostics)
	}
	idx := &qlik.Index{Files: []qlik.File{f}}
	full := Map(idx, "", 2048)
	for _, want := range []string{
		"rename field Alpha -> First, Beta -> Second, Gamma -> Target",
		"rename table Stage -> Published", "rename fields using Names",
		"drop fields Alpha, Beta, Target | from: Published, Archive",
		"drop mapping tables Names", "trace | uses: $(vBatch)", "ops.qvs:7 trace",
	} {
		if !strings.Contains(full, want) {
			t.Errorf("map missing %q:\n%s", want, full)
		}
	}
	for _, budget := range []int{64, 96, 128, 256, 512} {
		mapped := Map(idx, "Target", budget)
		if len(mapped) > budget || Map(idx, "Target", budget) != mapped {
			t.Errorf("map budget or stability failure: %d: %s", budget, mapped)
		}
		first, _, _ := strings.Cut(mapped, "\n")
		if !strings.Contains(first, "Gamma -> Target") {
			t.Errorf("compact rename lost focused pair at %d: %s", budget, mapped)
		}
	}
	for _, budget := range []int{128, 256, 512} {
		text := Context(idx, "Target", budget)
		if len(text) > budget || !strings.Contains(text, "1 | RENAME FIELDS Alpha TO First, Beta TO Second, Gamma TO Target;") {
			t.Errorf("original numbered operation context lost at %d: %s", budget, text)
		}
	}
	if edges := Dependencies(idx, "Target", "", "both"); len(edges) != 0 {
		t.Errorf("rename/drop invented lineage: %+v", edges)
	}
}

func TestOperationsDropFocusAndNoRuntimeMutation(t *testing.T) {
	t.Parallel()
	source := "DROP TABLES VeryLongFirstName, VeryLongSecondName, Target;\n" +
		"RENAME TABLE Old TO New;\nOld: LOAD ID FROM [input.qvd] (qvd);\n" +
		"DROP TABLE Old;\nLater: LOAD ID RESIDENT Old;"
	f, err := qlik.Parse(context.Background(), "ops.qvs", []byte(source))
	if err != nil || len(f.Diagnostics) != 0 {
		t.Fatalf("parse: %v, diagnostics: %+v", err, f.Diagnostics)
	}
	idx := &qlik.Index{Files: []qlik.File{f}}
	for _, budget := range []int{48, 64, 96} {
		mapped := Map(idx, "Target", budget)
		first, _, _ := strings.Cut(mapped, "\n")
		if len(mapped) > budget || !strings.Contains(first, "drop tables ") || !strings.Contains(first, "Target") {
			t.Errorf("compact drop lost focused target at %d: %s", budget, mapped)
		}
	}
	for _, e := range f.Edges {
		if e.Kind == "resident" && e.To.Name != "Old" {
			t.Errorf("renamed/deleted a literal RESIDENT endpoint: %+v", e)
		}
	}
}

func TestOperationsVariableContextDoesNotExpandWholeFile(t *testing.T) {
	t.Parallel()
	source := "SET vBatch = 3;\nTRACE $(vBatch);\n$(Include=unrelated.qvs);\n" +
		"Unrelated: LOAD Secret FROM [unrelated.qvd] (qvd);"
	f, err := qlik.Parse(context.Background(), "ops.qvs", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	other, err := qlik.Parse(context.Background(), "unrelated.qvs", []byte("Hidden: LOAD Sensitive AUTOGENERATE 1;"))
	if err != nil {
		t.Fatal(err)
	}
	idx := &qlik.Index{Files: []qlik.File{f, other}}
	text := Context(idx, "vBatch", 4096)
	if !strings.Contains(text, "2 | TRACE $(vBatch);") || strings.Contains(text, "Unrelated") || strings.Contains(text, "Sensitive") {
		t.Fatalf("variable use expanded unrelated source: %s", text)
	}
}
