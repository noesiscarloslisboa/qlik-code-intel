package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noesiscarloslisboa/qlik-code-intel/internal/qlik"
)

func execute(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, diagnostics bytes.Buffer
	code := Run(context.Background(), args, &out, &diagnostics)
	return code, out.String(), diagnostics.String()
}

func TestCommandsAgainstRepository(t *testing.T) {
	t.Parallel()
	root := "../../testdata/repository"
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{"scan", []string{"scan", "--root", root}, "Scanned 5 QVS files"},
		{"map", []string{"map", "--root", root, "--tokens", "1024"}, "table Orders"},
		{"find flags after query", []string{"find", "Revenue", "--root", root, "--kind", "field"}, "definition field Revenue"},
		{"deps", []string{"deps", "Daily Sales", "--root", root, "--direction", "upstream"}, "table:Daily Sales -> table:Orders [resident]"},
		{"context", []string{"context", "Revenue", "--root", root, "--tokens", "2048"}, "4 |     Sum(GrossAmount) AS Revenue"},
		{"help", []string{"--help"}, "Commands:"},
		{"command help", []string{"map", "--help"}, "--"},
		{"version", []string{"version"}, "dev"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			code, out, stderr := execute(t, tt.args...)
			if code != 0 || stderr != "" {
				t.Fatalf("code=%d stderr=%q stdout=%q", code, stderr, out)
			}
			if tt.name == "command help" {
				if !strings.Contains(out, "tokens") {
					t.Errorf("help: %s", out)
				}
			} else if !strings.Contains(out, tt.want) {
				t.Errorf("missing %q: %s", tt.want, out)
			}
		})
	}
}

func TestUsageErrorsDoNotScanOrPolluteStdout(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		args []string
	}{
		{"unknown command", []string{"bad"}},
		{"missing query", []string{"find"}},
		{"too many queries", []string{"find", "one", "two"}},
		{"empty query", []string{"context", " "}},
		{"zero budget", []string{"map", "--tokens", "0"}},
		{"negative budget", []string{"context", "x", "--tokens", "-1"}},
		{"bad kind", []string{"find", "x", "--kind", "function"}},
		{"bad role", []string{"find", "x", "--role", "any"}},
		{"bad direction", []string{"deps", "x", "--direction", "sideways"}},
		{"negative limit", []string{"find", "x", "--limit", "-2"}},
		{"unknown flag", []string{"map", "--bogus"}},
		{"missing value", []string{"scan", "--root"}},
		{"scan positional", []string{"scan", "somewhere"}},
		{"map json not supported", []string{"map", "--json"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			code, out, stderr := execute(t, tt.args...)
			if code != 2 || out != "" || stderr == "" {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out, stderr)
			}
		})
	}
}

func TestMachineReadableOutput(t *testing.T) {
	t.Parallel()
	code, out, stderr := execute(t, "scan", "--root", "../../testdata/repository", "--json")
	if code != 0 || stderr != "" {
		t.Fatalf("code=%d: %s", code, stderr)
	}
	var idx qlik.Index
	if err := json.Unmarshal([]byte(out), &idx); err != nil {
		t.Fatal(err)
	}
	if len(idx.Files) != 5 || len(idx.Symbols()) == 0 || len(idx.Edges()) == 0 {
		t.Fatalf("incomplete index: %+v", idx)
	}
	if strings.Contains(out, "Sum(GrossAmount) AS Revenue") {
		t.Error("JSON index should omit source text")
	}
	for _, cmd := range []string{"find", "deps"} {
		code, out, stderr := execute(t, cmd, "Orders", "--root", "../../testdata/repository", "--json")
		if code != 0 || stderr != "" || !json.Valid([]byte(out)) {
			t.Fatalf("%s JSON code=%d stdout=%s stderr=%s", cmd, code, out, stderr)
		}
	}
}

func TestDiagnosticsAndStrictMode(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "mixed.qvs"), []byte("TRACE 'unsupported';\nT: LOAD ID FROM [x.qvd] (qvd);"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, strict := range []bool{false, true} {
		args := []string{"scan", "--root", root, "--json"}
		wantCode := 0
		if strict {
			args = append(args, "--strict")
			wantCode = 1
		}
		code, out, stderr := execute(t, args...)
		if code != wantCode || !json.Valid([]byte(out)) || !strings.Contains(stderr, "mixed.qvs:1:1 [unsupported]") {
			t.Fatalf("code=%d stdout=%s stderr=%s", code, out, stderr)
		}
		if !strings.Contains(out, `"name": "T"`) {
			t.Error("lost supported table after unsupported statement")
		}
	}
}

func TestEmptyRepositoryNoMatchAndIOFailures(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, cmd := range []string{"map", "find", "deps", "context"} {
		args := []string{cmd, "--root", root}
		if cmd != "map" {
			args = append(args, "missing")
		}
		code, out, stderr := execute(t, args...)
		if code != 0 || out != "" || stderr != "" {
			t.Fatalf("empty %s: %d %q %q", cmd, code, out, stderr)
		}
	}
	code, _, stderr := execute(t, "scan", "--root", filepath.Join(root, "absent"))
	if code != 1 || stderr == "" {
		t.Fatal("missing repository should fail")
	}
	var diagnostics bytes.Buffer
	if code := Run(context.Background(), []string{"scan", "--root", root}, brokenWriter{}, &diagnostics); code != 1 {
		t.Errorf("broken pipe exit: %d", code)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if code := Run(ctx, []string{"scan", "--root", root}, io.Discard, &diagnostics); code != 1 {
		t.Errorf("canceled exit: %d", code)
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("broken output") }

func TestRejectInvalidUTF8(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "invalid.qvs"), []byte("T: LOAD ID AS [Bad\xff] FROM [x.qvd] (qvd);"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"scan", "--json"}, {"map"}, {"find", "Bad", "--json"}, {"deps", "T", "--json"}, {"context", "Bad"},
	} {
		code, out, stderr := execute(t, append(args, "--root", root)...)
		if code != 1 || out != "" || !strings.Contains(stderr, "invalid.qvs") || !strings.Contains(stderr, "UTF-8") {
			t.Errorf("%s: code=%d stdout=%q stderr=%q", args[0], code, out, stderr)
		}
	}
}

func TestHelpOutputErrors(t *testing.T) {
	t.Parallel()
	for _, command := range []string{"scan", "map", "find", "deps", "context"} {
		var diagnostics bytes.Buffer
		code := Run(context.Background(), []string{command, "--help"}, brokenWriter{}, &diagnostics)
		if code != 1 || !strings.Contains(diagnostics.String(), "broken output") {
			t.Errorf("%s help: code=%d stderr=%q", command, code, diagnostics.String())
		}
	}
}
