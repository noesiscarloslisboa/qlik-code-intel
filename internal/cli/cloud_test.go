package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noesiscarloslisboa/qlik-code-intel/internal/cloud"
)

const cloudSource = "\ufeff///$tab Main\r\nSET vRoot = lib://Warehouse/;\r\nSales:\r\nLOAD RawAmount AS Revenue FROM [$(vRoot)sales.qvd] (qvd);\r\nTRACE 'unsupported';\r\nSummary:\r\nLOAD Revenue RESIDENT Sales;\r\nSTORE Summary INTO [lib://Exports/summary.qvd] (qvd);"

func TestCloudPullAndOfflineRetrieval(t *testing.T) {
	requests := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer local-test-token" {
			t.Error("expected authenticated GET")
		}
		var data any
		switch r.URL.RequestURI() {
		case "/api/v1/apps/app-1":
			data = map[string]any{"attributes": map[string]string{"id": "app-1", "name": "Original test app"}}
		case "/api/v1/apps/app-1/scripts?limit=1":
			data = map[string]any{"scripts": []map[string]string{{"scriptId": "saved-9", "modifiedTime": "2026-10-01T10:00:00Z"}}}
		case "/api/v1/apps/app-1/scripts/saved-9":
			data = map[string]string{"script": cloudSource}
		default:
			http.Error(w, "unexpected endpoint", http.StatusNotFound)
			return
		}
		if err := json.NewEncoder(w).Encode(data); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	deps := cloudDependencies{httpClient: server.Client(), lookupEnv: func(key string) (string, bool) {
		if key != "QLIK_CLOUD_TOKEN" {
			t.Fatalf("unexpected environment lookup: %s", key)
		}
		return "local-test-token", true
	}}
	for _, asJSON := range []bool{false, true} {
		destination := filepath.Join(cloudTempDir(t), "snapshot")
		args := []string{"cloud", "pull", "--tenant", server.URL, "--app", "app-1", "--out", destination}
		if asJSON {
			args = append(args, "--json")
		}
		var out, diagnostics bytes.Buffer
		code := run(context.Background(), args, &out, &diagnostics, deps)
		if code != 0 || diagnostics.Len() != 0 || strings.Contains(out.String(), "local-test-token") || strings.Contains(out.String(), "RawAmount") {
			t.Fatalf("pull: %d stdout=%s stderr=%s", code, out.String(), diagnostics.String())
		}
		if asJSON {
			var result cloud.Result
			if err := json.Unmarshal(out.Bytes(), &result); err != nil || result.AppID != "app-1" || result.ScriptID != "saved-9" || result.SnapshotPath != destination {
				t.Fatalf("summary: %+v, %v", result, err)
			}
		} else if !strings.Contains(out.String(), destination) || !strings.Contains(out.String(), "saved-9") {
			t.Fatalf("missing snapshot identity: %s", out.String())
		}
		source, err := os.ReadFile(filepath.Join(destination, "script.qvs"))
		if err != nil || string(source) != cloudSource {
			t.Fatalf("source changed: %v", err)
		}
		metadata, err := os.ReadFile(filepath.Join(destination, "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		var manifest cloud.Manifest
		digest := sha256.Sum256(source)
		if err := json.Unmarshal(metadata, &manifest); err != nil || manifest.SourceSHA256 != hex.EncodeToString(digest[:]) || manifest.SourceBytes != len(source) || manifest.Tenant != server.URL || manifest.ScriptID != "saved-9" {
			t.Fatalf("manifest mismatch: %+v, %v", manifest, err)
		}
		// These use the public offline runner, with no injected Cloud transport.
		for _, tt := range []struct {
			args []string
			code int
			want string
		}{
			{[]string{"scan", "--strict", "--json"}, 1, `"name": "Summary"`},
			{[]string{"map", "--query", "Revenue", "--tokens", "512"}, 0, "script.qvs:3 table Sales"},
			{[]string{"find", "Revenue", "--kind", "field", "--role", "definition"}, 0, "script.qvs:4:19 definition field Revenue"},
			{[]string{"find", "sales.qvd", "--kind", "qvd"}, 0, "$(vRoot)sales.qvd"},
			{[]string{"deps", "Summary", "--direction", "upstream"}, 0, "table:Summary -> table:Sales [resident]"},
			{[]string{"context", "Revenue", "--tokens", "512"}, 0, "4 | LOAD RawAmount AS Revenue FROM [$(vRoot)sales.qvd] (qvd);"},
		} {
			code, text, stderr := execute(t, append(tt.args, "--root", destination)...)
			if code != tt.code || !strings.Contains(text, tt.want) || !strings.Contains(stderr, "script.qvs:5:1 [unsupported]") {
				t.Fatalf("%v: code=%d stdout=%s stderr=%s", tt.args, code, text, stderr)
			}
			if tt.args[0] == "scan" && !json.Valid([]byte(text)) {
				t.Error("invalid scan JSON")
			}
			if (tt.args[0] == "context" || tt.args[0] == "map") && len(text) > 512 {
				t.Fatalf("budget overflow: %d", len(text))
			}
		}
	}
	if requests != 6 {
		t.Fatalf("offline retrieval made unexpected requests: %d", requests)
	}
	// A broken output stream reports failure but retains the completed snapshot.
	destination := filepath.Join(cloudTempDir(t), "snapshot")
	var diagnostics bytes.Buffer
	args := []string{"cloud", "pull", "--tenant", server.URL, "--app", "app-1", "--out", destination, "--json"}
	if code := run(context.Background(), args, brokenWriter{}, &diagnostics, deps); code != 1 || !strings.Contains(diagnostics.String(), "broken output") {
		t.Fatalf("summary write failure: %d %s", code, diagnostics.String())
	}
	if source, err := os.ReadFile(filepath.Join(destination, "script.qvs")); err != nil || string(source) != cloudSource {
		t.Fatalf("completed snapshot lost after summary failure: %v", err)
	}
}

func TestCloudHelpAndUsage(t *testing.T) {
	deps := cloudDependencies{lookupEnv: func(string) (string, bool) {
		t.Fatal("help and usage errors must not read credentials")
		return "", false
	}}
	for _, args := range [][]string{{"cloud"}, {"cloud", "--help"}, {"cloud", "pull", "--help"}} {
		var out, diagnostics bytes.Buffer
		if code := run(context.Background(), args, &out, &diagnostics, deps); code != 0 || diagnostics.Len() != 0 || !strings.Contains(out.String(), "QLIK_CLOUD_TOKEN") {
			t.Fatalf("%v: %d %s %s", args, code, out.String(), diagnostics.String())
		}
		if code := run(context.Background(), args, brokenWriter{}, &diagnostics, deps); code != 1 {
			t.Fatalf("help write failure: %d", code)
		}
	}
	valid := []string{"cloud", "pull", "--tenant", "https://example.qlikcloud.com", "--app", "app-1", "--out", filepath.Join(cloudTempDir(t), "snapshot")}
	for _, args := range [][]string{
		{"cloud", "bad"}, {"cloud", "--help", "extra"}, {"cloud", "pull"},
		append(append([]string{}, valid...), "extra"), append(append([]string{}, valid...), "--bogus"),
		{"cloud", "pull", "--tenant", "http://example.com", "--app", "app-1", "--out", "x"},
		{"cloud", "pull", "--tenant", "https://user:secret@example.com", "--app", "app-1", "--out", "x"},
		{"cloud", "pull", "--tenant", "https://example.com", "--app", " ", "--out", "x"},
	} {
		var out, diagnostics bytes.Buffer
		if code := run(context.Background(), args, &out, &diagnostics, deps); code != 2 || out.Len() != 0 || diagnostics.Len() == 0 || strings.Contains(diagnostics.String(), "secret") {
			t.Fatalf("%v: %d %s %s", args, code, out.String(), diagnostics.String())
		}
	}
}

func TestCloudRuntimeFailures(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "private-source local-test-token", http.StatusForbidden)
	}))
	t.Cleanup(server.Close)
	for _, token := range []string{"", "bad\ncredential", "local-test-token"} {
		destination := filepath.Join(cloudTempDir(t), "snapshot")
		args := []string{"cloud", "pull", "--tenant", server.URL, "--app", "app-1", "--out", destination, "--json"}
		deps := cloudDependencies{httpClient: server.Client(), lookupEnv: func(string) (string, bool) { return token, token != "" }}
		var out, diagnostics bytes.Buffer
		if code := run(context.Background(), args, &out, &diagnostics, deps); code != 1 || out.Len() != 0 || diagnostics.Len() == 0 || strings.Contains(diagnostics.String(), "local-test-token") || strings.Contains(diagnostics.String(), "private-source") {
			t.Fatalf("runtime failure: %d stdout=%s stderr=%s", code, out.String(), diagnostics.String())
		}
		if _, err := os.Lstat(destination); !os.IsNotExist(err) {
			t.Fatalf("failure left destination: %v", err)
		}
	}
	// Invalid destinations fail before reading authentication or making requests.
	deps := cloudDependencies{lookupEnv: func(string) (string, bool) { t.Fatal("unexpected credentials lookup"); return "", false }}
	if code := run(context.Background(), []string{"cloud", "pull", "--tenant", server.URL, "--app", "app-1", "--out", cloudTempDir(t)}, io.Discard, io.Discard, deps); code != 1 {
		t.Fatalf("existing destination: %d", code)
	}
}

func cloudTempDir(t *testing.T) string {
	t.Helper()
	path, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return path
}
