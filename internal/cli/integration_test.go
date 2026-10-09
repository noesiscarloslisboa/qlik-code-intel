//go:build integration

package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/noesiscarloslisboa/qlik-code-intel/internal/cloud"
)

func TestBuiltExecutableEndToEnd(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "qlik-repomap")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, "./cmd/qlik-repomap")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	t.Run("cloud help and preflight", func(t *testing.T) {
		destination := filepath.Join(cloudTempDir(t), "snapshot")
		for _, tt := range []struct {
			args []string
			code int
			want string
		}{
			{[]string{"--help"}, 0, "cloud"},
			{[]string{"cloud", "pull", "--help"}, 0, "QLIK_CLOUD_TOKEN"},
			{[]string{"cloud", "pull", "--bogus"}, 2, "flag provided but not defined"},
			{[]string{"cloud", "pull"}, 2, "requires --tenant"},
			{[]string{"cloud", "pull", "--tenant", "https://example.qlikcloud.com", "--app", "original-app", "--out", destination}, 1, "set QLIK_CLOUD_TOKEN"},
		} {
			command := exec.Command(binary, tt.args...)
			// Never allow an ambient credential into a subprocess acceptance test.
			for _, entry := range os.Environ() {
				if key, _, _ := strings.Cut(entry, "="); !strings.EqualFold(key, "QLIK_CLOUD_TOKEN") {
					command.Env = append(command.Env, entry)
				}
			}
			command.Env = append(command.Env, "QLIK_CLOUD_TOKEN=")
			var stdout, stderr bytes.Buffer
			command.Stdout, command.Stderr = &stdout, &stderr
			err := command.Run()
			code := 0
			if failure, ok := err.(*exec.ExitError); ok {
				code = failure.ExitCode()
			} else if err != nil {
				t.Fatal(err)
			}
			text := stdout.String()
			if tt.code != 0 {
				text = stderr.String()
				if stdout.Len() != 0 {
					t.Fatalf("failure polluted stdout: %s", stdout.String())
				}
			} else if stderr.Len() != 0 {
				t.Fatalf("help polluted stderr: %s", stderr.String())
			}
			if code != tt.code || !strings.Contains(text, tt.want) {
				t.Fatalf("%v: code=%d stdout=%s stderr=%s", tt.args, code, stdout.String(), stderr.String())
			}
		}
		if _, err := os.Lstat(destination); !os.IsNotExist(err) {
			t.Fatalf("preflight wrote destination: %v", err)
		}
	})
	t.Run("offline cloud snapshot", func(t *testing.T) {
		source := []byte(cloudSource)
		digest := sha256.Sum256(source)
		destination := filepath.Join(cloudTempDir(t), "snapshot")
		_, err := cloud.WriteSnapshot(context.Background(), destination, cloud.Snapshot{Source: source, Manifest: cloud.Manifest{
			SchemaVersion: 1, Tenant: "https://example.qlikcloud.com", AppID: "original-app", ScriptID: "saved-1",
			FetchedAt: time.Unix(1, 0).UTC(), SourceFile: "script.qvs", SourceSHA256: hex.EncodeToString(digest[:]), SourceBytes: len(source),
		}})
		if err != nil {
			t.Fatal(err)
		}
		for _, tt := range []struct {
			args []string
			want string
		}{
			{[]string{"scan", "--json"}, `"name": "Revenue"`},
			{[]string{"map", "--query", "Revenue", "--tokens", "512"}, "table Sales"},
			{[]string{"find", "sales.qvd", "--kind", "qvd"}, "$(vRoot)sales.qvd"},
			{[]string{"deps", "Summary", "--direction", "upstream"}, "table:Summary -> table:Sales [resident]"},
			{[]string{"context", "Revenue", "--tokens", "512"}, "4 | LOAD RawAmount AS Revenue FROM [$(vRoot)sales.qvd] (qvd);"},
		} {
			command := exec.Command(binary, append(tt.args, "--root", destination)...)
			var stderr bytes.Buffer
			command.Stderr = &stderr
			out, err := command.Output()
			if err != nil || !strings.Contains(string(out), tt.want) || !strings.Contains(stderr.String(), "script.qvs:5:1 [unsupported]") {
				t.Fatalf("%v: %v stdout=%s stderr=%s", tt.args, err, out, stderr.String())
			}
			if (tt.args[0] == "map" || tt.args[0] == "context") && len(out) > 512 {
				t.Fatalf("budget overflow: %d", len(out))
			}
		}
	})
	repository := filepath.Join(root, "testdata/repository")
	accuracy := filepath.Join(root, "testdata/accuracy")
	qlikview := filepath.Join(root, "testdata/qlikview")
	retrieval := filepath.Join(root, "testdata/retrieval")
	for _, tt := range []struct {
		name       string
		repository string
		args       []string
		want       string
		absent     string
	}{
		{"scan", repository, []string{"scan", "--json"}, `"name": "Orders"`, ""},
		{"map", repository, []string{"map", "--tokens", "1024"}, "table Daily Sales", ""},
		{"find", repository, []string{"find", "Revenue", "--kind", "field"}, "20_summary.qvs:4", ""},
		{"deps", repository, []string{"deps", "Orders", "--direction", "downstream"}, "Daily Sales", ""},
		{"context", repository, []string{"context", "Revenue", "--tokens", "2048"}, "Sum(GrossAmount) AS Revenue", ""},
		{"accuracy scan", accuracy, []string{"scan", "--strict", "--json"}, `"anonymous": true`, ""},
		{"accuracy map", accuracy, []string{"map", "--query", "Revenue", "--tokens", "512"}, "preceding @preceding.qvs:5:1", ""},
		{"accuracy find", accuracy, []string{"find", "vRate", "--kind", "variable", "--role", "reference"}, "variables.qvs:7", ""},
		{"accuracy deps", accuracy, []string{"deps", "vNet", "--kind", "variable", "--direction", "upstream"}, "variable:vNet -> variable:vBase [variable]", ""},
		{"accuracy context", accuracy, []string{"context", "Revenue", "--tokens", "2048"}, "Num#(RawAmount", "FROM [lib://Warehouse/transactions.qvd]"},
		{"qlikview scan", qlikview, []string{"scan", "--strict", "--json"}, `"name": "vMetric_$(vRun)"`, ""},
		{"qlikview map", qlikview, []string{"map", "--query", "Amount_$(vRun)", "--tokens", "512"}, `$(vRoot)\sales.qvd`, ""},
		{"qlikview find", qlikview, []string{"find", "sales.qvd", "--kind", "qvd"}, "filesystem.qvs:5", ""},
		{"qlikview deps", qlikview, []string{"deps", "sales.qvd", "--kind", "qvd", "--direction", "upstream"}, `source:\\server\share\sales.qvd -> table:Batch_$(vRun) [store]`, ""},
		{"qlikview context", qlikview, []string{"context", "vMetric_$(vRun)", "--tokens", "512"}, "3 | LET vMetric_$(vRun) = 42;", ""},
		{"hierarchy scan", qlikview, []string{"scan", "--strict", "--json"}, `"name": "Branch Path"`, ""},
		{"hierarchy map", qlikview, []string{"map", "--query", "Branch Path", "--tokens", "256"}, "table Tree | fields: ParentCaption, Branch Path, Depth", ""},
		{"hierarchy find", qlikview, []string{"find", "Depth", "--kind", "field", "--role", "definition"}, "coverage.qvs:5", ""},
		{"hierarchy deps", qlikview, []string{"deps", "Tree", "--kind", "table", "--direction", "upstream"}, "table:Tree -> source:nodes.qvd [from]", ""},
		{"hierarchy context", qlikview, []string{"context", "Branch Path", "--tokens", "256"}, "5 | Hierarchy(ID, Parent, Caption", ""},
		{"retrieval map", retrieval, []string{"map", "--query", "LateValue", "--tokens", "256"}, "10_pipeline.qvs:1 table Events | fields: LateValue", ""},
		{"retrieval context", retrieval, []string{"context", "LateValue", "--tokens", "256"}, "25 |     FinalValue AS LateValue", ""},
		{"retrieval producer", retrieval, []string{"deps", "snapshot.qvd", "--kind", "qvd", "--direction", "upstream"}, "source:lib://Exports/snapshot.qvd -> table:Snapshot [store]", ""},
		{"retrieval consumers", retrieval, []string{"deps", "events.qvd", "--kind", "qvd", "--direction", "downstream"}, "table:Symbolic -> source:$(vRoot)events.qvd [from]", "Backup"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			args := append(append([]string{}, tt.args...), "--root", tt.repository)
			command := exec.Command(binary, args...)
			var stderr bytes.Buffer
			command.Stderr = &stderr
			out, err := command.Output()
			if err != nil || stderr.Len() != 0 {
				t.Fatalf("command: %v\n%s", err, stderr.String())
			}
			if !strings.Contains(string(out), tt.want) {
				t.Errorf("missing %q:\n%s", tt.want, out)
			}
			if tt.absent != "" && strings.Contains(string(out), tt.absent) {
				t.Errorf("unexpected %q:\n%s", tt.absent, out)
			}
			if tt.args[0] == "scan" && !json.Valid(out) {
				t.Error("invalid scan JSON")
			}
			if tt.args[0] == "map" && len(out) > 1024 {
				t.Errorf("budget overflow: %d", len(out))
			}
		})
	}
	t.Run("balanced literal recovery", func(t *testing.T) {
		recovery := t.TempDir()
		source, err := os.ReadFile(filepath.Join(root, "testdata/recovery-balanced.qvs"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(recovery, "recovery.qvs"), source, 0644); err != nil {
			t.Fatal(err)
		}
		command := exec.Command(binary, "scan", "--json", "--strict", "--root", recovery)
		var stderr bytes.Buffer
		command.Stderr = &stderr
		out, err := command.Output()
		if failure, ok := err.(*exec.ExitError); !ok || failure.ExitCode() != 1 {
			t.Fatalf("strict scan should return 1 for unsupported literals: %v", err)
		}
		if !json.Valid(out) || !strings.Contains(string(out), `"name": "Recovered"`) || strings.Count(stderr.String(), "[unsupported-literal]") != 2 {
			t.Fatalf("recovery scan: stdout=%s stderr=%s", out, stderr.String())
		}
		command = exec.Command(binary, "context", "Accepted", "--tokens", "512", "--root", recovery)
		stderr.Reset()
		command.Stderr = &stderr
		out, err = command.Output()
		if err != nil || !strings.Contains(string(out), "5 | LOAD RecordID") || len(out) > 512 {
			t.Fatalf("recovered context: %v stdout=%s stderr=%s", err, out, stderr.String())
		}
	})
	t.Run("backticks and unfinished statement recovery", func(t *testing.T) {
		hardening := filepath.Join(root, "testdata/hardening")
		for _, tt := range []struct {
			args []string
			want string
		}{
			{[]string{"scan", "--json"}, `"name": "Recovered Table"`},
			{[]string{"find", "Net\" Amount", "--kind", "field", "--role", "definition"}, "10_sales.qvs:2"},
			{[]string{"find", "recovered.qvd", "--kind", "qvd"}, "20_recovery.qvs:4"},
			{[]string{"map", "--query", "Recovered ID", "--tokens", "256"}, "Recovered ID"},
			{[]string{"deps", "Summary", "--kind", "table", "--direction", "upstream"}, "table:Quarter Sales [resident]"},
			{[]string{"context", "Recovered ID", "--tokens", "256"}, "4 | LOAD `Raw ID` AS `Recovered ID`"},
		} {
			args := append(append([]string{}, tt.args...), "--root", hardening)
			command := exec.Command(binary, args...)
			var stderr bytes.Buffer
			command.Stderr = &stderr
			out, err := command.Output()
			if err != nil || !strings.Contains(string(out), tt.want) || !strings.Contains(stderr.String(), "[unterminated-statement]") {
				t.Fatalf("%v: %v stdout=%s stderr=%s", tt.args, err, out, stderr.String())
			}
			if strings.Contains(string(out), "vGhost") || strings.Contains(string(out), "fake.qvd") {
				t.Errorf("opaque text leaked into output: %s", out)
			}
			if tt.args[0] == "scan" && !json.Valid(out) {
				t.Error("scan JSON invalid")
			}
			if (tt.args[0] == "map" || tt.args[0] == "context") && len(out) > 256 {
				t.Errorf("budget overflow: %d", len(out))
			}
		}
		command := exec.Command(binary, "scan", "--json", "--strict", "--root", hardening)
		out, err := command.Output()
		if failure, ok := err.(*exec.ExitError); !ok || failure.ExitCode() != 1 || !json.Valid(out) {
			t.Fatalf("strict scan should retain JSON and fail for unfinished syntax: %v stdout=%s", err, out)
		}
	})
}
