package repomap

import (
	"strings"
	"testing"

	"github.com/noesiscarloslisboa/qlik-code-intel/internal/qlik"
)

func TestWindowsSourceBasenamesRankBeforePartialMatches(t *testing.T) {
	t.Parallel()
	idx := &qlik.Index{Files: []qlik.File{accuracyFile(t, "paths.qvs", "Backup: LOAD ID FROM [C:\\Archive\\sales.qvd.backup.qvd] (qvd);\nDrive: LOAD ID FROM [C:\\Data\\sales.qvd] (qvd);\nNetwork: LOAD ID FROM [\\\\server\\share\\sales.qvd] (qvd);\nLibrary: LOAD ID FROM [lib://Data/sales.qvd] (qvd);")}}
	got := Find(idx, Search{Query: "sales.qvd", Kind: "qvd"})
	if len(got) != 4 || got[0].Line != 2 || got[1].Line != 3 || got[2].Line != 4 || got[3].Line != 1 {
		t.Fatalf("platform-dependent basename ranking: %+v", got)
	}
	deps := Dependencies(idx, "sales.qvd", "qvd", "downstream")
	if len(deps) != 3 {
		t.Fatalf("basename dependency endpoints lost or partial included: %+v", deps)
	}
	for _, d := range deps {
		if d.Line == 1 {
			t.Error("partial source selected over basenames")
		}
	}
	for _, budget := range []int{128, 256, 512} {
		if got := Map(idx, "sales.qvd", budget); !strings.HasPrefix(got, "paths.qvs:2") || len(got) > budget {
			t.Errorf("focused path map wrong or over budget: %q", got)
		}
	}
}
