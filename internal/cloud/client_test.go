package cloud

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchPinsSavedVersionAndPreservesSource(t *testing.T) {
	source := "\ufeff///$tab Main\r\nSET vRoot = lib://Data;\r\nSales:\r\n\tLOAD [Café] AS Revenue FROM [$(vRoot)/orders.qvd] (qvd);"
	var paths []string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())
		if r.Method != "GET" || r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected method or authentication")
		}
		switch r.URL.RequestURI() {
		case "/api/v1/apps/app-1":
			io.WriteString(w, `{"attributes":{"id":"app-1","name":"Sales","lastReloadTime":"2026-10-01T12:00:00Z"}}`)
		case "/api/v1/apps/app-1/scripts?limit=1":
			io.WriteString(w, `{"scripts":[{"scriptId":"saved-7","modifiedTime":"2026-10-09T12:00:00Z","versionMessage":"Reviewed"}]}`)
		case "/api/v1/apps/app-1/scripts/saved-7":
			json.NewEncoder(w).Encode(map[string]string{"script": source})
		default: // The current head can change after the history request.
			json.NewEncoder(w).Encode(map[string]string{"script": "different saved version"})
		}
	}))
	defer server.Close()
	client, err := NewClient(server.URL+"/", "test-token", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := client.Fetch(context.Background(), "app-1")
	if err != nil {
		t.Fatal(err)
	}
	if string(snapshot.Source) != source || snapshot.Manifest.ScriptID != "saved-7" || len(paths) != 3 || paths[2] != "/api/v1/apps/app-1/scripts/saved-7" {
		t.Fatalf("source/version/request mismatch: %v %+v", paths, snapshot.Manifest)
	}
	m := snapshot.Manifest
	hash := sha256.Sum256([]byte(source))
	if m.SourceSHA256 != hex.EncodeToString(hash[:]) || m.SourceBytes != len([]byte(source)) || m.SourceFile != "script.qvs" || m.SchemaVersion != 1 {
		t.Fatalf("source identity: %+v", m)
	}
	if m.Tenant != server.URL || m.AppID != "app-1" || m.AppName != "Sales" || m.ScriptModifiedTime != "2026-10-09T12:00:00Z" || m.ScriptVersionMessage != "Reviewed" || m.LastReloadTime != "2026-10-01T12:00:00Z" || m.FetchedAt.IsZero() || m.FetchedAt.Location() != time.UTC {
		t.Fatalf("metadata: %+v", m)
	}
}

func TestFetchRejectsIncompleteOrMalformedResponses(t *testing.T) {
	for _, tt := range []struct {
		name, app, history, source string
	}{
		{"wrong app", `{"attributes":{"id":"other"}}`, "", ""},
		{"null app", `null`, "", ""},
		{"empty history", "", `{"scripts":[]}`, ""},
		{"missing version", "", `{"scripts":[{}]}`, ""},
		{"moving alias", "", `{"scripts":[{"scriptId":"current"}]}`, ""},
		{"missing script", "", "", `{}`},
		{"null script", "", "", `{"script":null}`},
		{"wrong script type", "", "", `{"script":1}`},
		{"broken JSON", "", "", `{"script":`},
		{"trailing JSON", "", "", `{"script":""} {}`},
		{"invalid UTF8", "", "", "{\"script\":\"\xff\"}"},
		{"high surrogate", "", "", `{"script":"\ud800"}`},
		{"low surrogate", "", "", `{"script":"\udfff"}`},
		{"wrong surrogate pair", "", "", `{"script":"\ud800\u0041"}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := fixtureServer(tt.app, tt.history, tt.source)
			defer server.Close()
			client, _ := NewClient(server.URL, "test-token", server.Client())
			_, err := client.Fetch(context.Background(), "app-1")
			if err == nil || strings.Contains(err.Error(), "test-token") || strings.Contains(err.Error(), tt.source) && tt.source != "" {
				t.Fatalf("expected sanitized failure, got %v", err)
			}
		})
	}
}

func TestFetchAcceptsEmptyScriptAndValidUnicode(t *testing.T) {
	for _, text := range []string{``, `\ud83d\ude00`, `\\ud800`, `\ufffd`} {
		server := fixtureServer("", "", `{"script":"`+text+`"}`)
		client, _ := NewClient(server.URL, "test-token", server.Client())
		got, err := client.Fetch(context.Background(), "app-1")
		server.Close()
		var expected struct{ Script string }
		json.Unmarshal([]byte(`{"script":"`+text+`"}`), &expected)
		if err != nil || string(got.Source) != expected.Script {
			t.Fatalf("valid source rejected: %q %v", text, err)
		}
	}
}

func TestRequestErrorsDoNotExposeRemoteBodiesOrFollowRedirects(t *testing.T) {
	for _, status := range []int{301, 401, 403, 404, 429, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Location", "/secret-target")
				w.WriteHeader(status)
				io.WriteString(w, "test-token private-script-body")
			}))
			defer server.Close()
			injected := server.Client()
			client, _ := NewClient(server.URL, "test-token", injected)
			_, err := client.Fetch(context.Background(), "app-1")
			if err == nil || calls != 1 || strings.Contains(err.Error(), "test-token") || strings.Contains(err.Error(), "private-script") {
				t.Fatalf("unsafe status handling: calls=%d err=%v", calls, err)
			}
			if injected.CheckRedirect != nil {
				t.Fatal("modified injected client")
			}
		})
	}
}

func TestFetchCancellationTimeoutAndSizeLimit(t *testing.T) {
	server := fixtureServer("", "", `{"script":"large source"}`)
	defer server.Close()
	client, _ := NewClient(server.URL, "test-token", server.Client())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Fetch(ctx, "app-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation not retained: %v", err)
	}
	client.maxResponseBytes = 10
	if _, err := client.Fetch(context.Background(), "app-1"); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("size limit: %v", err)
	}
	slow := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer slow.Close()
	short := slow.Client()
	short.Timeout = 20 * time.Millisecond
	client, _ = NewClient(slow.URL, "test-token", short)
	if _, err := client.Fetch(context.Background(), "app-1"); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("timeout: %v", err)
	}
}

func TestFetchSanitizesTransportErrorsAndVerifiesTLS(t *testing.T) {
	injected := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("test-token private-script-body")
	})}
	client, err := NewClient("https://example.qlikcloud.com", "test-token", injected)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Fetch(context.Background(), "app-1"); err == nil || strings.Contains(err.Error(), "test-token") || strings.Contains(err.Error(), "private-script") {
		t.Fatalf("transport error was not sanitized: %v", err)
	}
	server := fixtureServer("", "", `{"script":""}`)
	defer server.Close()
	client, _ = NewClient(server.URL, "test-token", nil) // No trust override in production.
	if _, err := client.Fetch(context.Background(), "app-1"); err == nil || !strings.Contains(err.Error(), "TLS") {
		t.Fatalf("untrusted certificate accepted: %v", err)
	}
	server = fixtureServer("", "", `{"script":"`+strings.Repeat("x", 200)+`"}`)
	defer server.Close()
	client, _ = NewClient(server.URL, "test-token", server.Client())
	client.maxResponseBytes = 128 // Metadata/history fit; the pinned script does not.
	if _, err := client.Fetch(context.Background(), "app-1"); err == nil || !strings.Contains(err.Error(), "get saved script: response exceeds") {
		t.Fatalf("script response was not bounded: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestValidationAndPathEncoding(t *testing.T) {
	for _, tenant := range []string{"", "http://example.com", "https://user:secret@example.com", "https://example.com/a", "https://example.com?x=1", "https://example.com?", "https://example.com#x", "https:///", "//example.com"} {
		if _, err := NewClient(tenant, "token", nil); err == nil {
			t.Errorf("accepted tenant %q", tenant)
		}
	}
	for _, token := range []string{"", " ", "token\nprivate", "token\tprivate"} {
		if _, err := NewClient("https://example.com", token, nil); err == nil {
			t.Errorf("accepted invalid token")
		}
	}
	for _, id := range []string{"", " ", ".", "..", "app\nprivate"} {
		if ValidateAppID(id) == nil {
			t.Errorf("accepted app ID %q", id)
		}
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/api/v1/apps/app%2Fpart%3F%23" {
			t.Errorf("unescaped path: %s", r.URL.EscapedPath())
		}
		w.WriteHeader(404)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL, "token", server.Client())
	client.Fetch(context.Background(), "app/part?#")
}

func fixtureServer(app, history, source string) *httptest.Server {
	if app == "" {
		app = `{"attributes":{"id":"app-1"}}`
	}
	if history == "" {
		history = `{"scripts":[{"scriptId":"saved-1"}]}`
	}
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/apps/app-1":
			io.WriteString(w, app)
		case "/api/v1/apps/app-1/scripts":
			io.WriteString(w, history)
		default:
			io.WriteString(w, source)
		}
	}))
}
