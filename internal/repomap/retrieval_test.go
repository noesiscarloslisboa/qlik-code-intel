package repomap

import (
	"testing"

	"github.com/noesiscarloslisboa/qlik-code-intel/internal/qlik"
)

func TestDependencyDirectionRanksEligibleEndpoints(t *testing.T) {
	t.Parallel()
	idx := &qlik.Index{Files: []qlik.File{accuracyFile(t, "pipeline.qvs", "Input: LOAD ID FROM report.qvd (qvd);\nSTORE Input INTO [lib://Exports/report.qvd] (qvd);")}}
	up := Dependencies(idx, "report.qvd", "qvd", "upstream")
	if len(up) != 1 || up[0].Kind != "store" || up[0].From.Name != "lib://Exports/report.qvd" || up[0].To.Name != "Input" {
		t.Fatalf("an ineligible consumer hid the producer: %+v", up)
	}
	down := Dependencies(idx, "report.qvd", "qvd", "downstream")
	if len(down) != 1 || down[0].Kind != "from" || down[0].To.Name != "report.qvd" {
		t.Fatalf("consumer changed: %+v", down)
	}
	if both := Dependencies(idx, "report.qvd", "qvd", "both"); len(both) != 2 {
		t.Fatalf("each direction must retain its best endpoints: %+v", both)
	}
}

func TestSymbolicSourceBasenamesKeepMatchingConsumers(t *testing.T) {
	t.Parallel()
	idx := &qlik.Index{Files: []qlik.File{accuracyFile(t, "paths.qvs", "Literal: LOAD ID FROM [data\\events.qvd] (qvd);\nSymbolic: LOAD ID FROM [$(vRoot)events.qvd] (qvd);\nNested: LOAD ID FROM [$(vRoot_$(vEnv))$(vFolder)events.qvd] (qvd);\nBackup: LOAD ID FROM [$(vRoot)events.qvd.backup.qvd] (qvd);\nSTORE Symbolic INTO [$(vOut)events.qvd] (qvd);")}}
	down := Dependencies(idx, "events.qvd", "qvd", "downstream")
	if len(down) != 3 {
		t.Fatalf("symbolic basename consumers lost or partial source included: %+v", down)
	}
	want := map[string]bool{`data\events.qvd`: false, "$(vRoot)events.qvd": false, "$(vRoot_$(vEnv))$(vFolder)events.qvd": false}
	for _, d := range down {
		if _, ok := want[d.To.Name]; !ok {
			t.Errorf("endpoint changed or unrelated endpoint included: %+v", d)
		}
		want[d.To.Name] = true
	}
	for name, found := range want {
		if !found {
			t.Errorf("missing literal endpoint %s", name)
		}
	}
	for _, m := range Find(idx, Search{Query: "events.qvd", Kind: "qvd"}) {
		if m.Line != 4 && m.Score < 900 {
			t.Errorf("symbolic filename is still only a partial match: %+v", m)
		}
		if m.Line == 4 && m.Score >= 900 {
			t.Errorf("backup incorrectly treated as a basename: %+v", m)
		}
	}
}

func TestSymbolicBasenameRankingDoesNotApplyToVariables(t *testing.T) {
	t.Parallel()
	idx := &qlik.Index{Files: []qlik.File{accuracyFile(t, "variables.qvs", "LET $(vPrefix)Value = 1;\nLET Value = 2;\nLET ValueExtra = 3;")}}
	got := Find(idx, Search{Query: "Value", Kind: "variable", Role: "definition"})
	if len(got) != 3 || got[0].Name != "Value" || got[1].Name != "ValueExtra" || got[2].Name != "$(vPrefix)Value" {
		t.Fatalf("source-path ranking leaked into symbolic variables: %+v", got)
	}
}
