// Package cloud imports saved Qlik Cloud scripts without executing or changing them.
package cloud

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"
)

const requestTimeout = 30 * time.Second
const responseLimit = 32 << 20

// Client issues bounded, authenticated GET requests to one HTTPS tenant.
type Client struct {
	tenant           string
	token            string
	http             http.Client
	maxResponseBytes int64
}

// ValidateTenant accepts an HTTPS origin, optionally with a trailing slash.
func ValidateTenant(tenant string) error {
	u, err := url.Parse(tenant)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Opaque != "" || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.ForceQuery || strings.Contains(tenant, "#") {
		return errors.New("--tenant must be an HTTPS base URL without credentials, path, query or fragment")
	}
	if _, err := http.NewRequest(http.MethodGet, tenant, nil); err != nil {
		return errors.New("invalid HTTPS tenant URL")
	}
	return nil
}

// ValidateAppID checks a nonempty identity; request path segments are escaped separately.
func ValidateAppID(id string) error {
	if id == "" || strings.TrimSpace(id) != id || id == "." || id == ".." || strings.ContainsFunc(id, unicode.IsControl) {
		return errors.New("app ID must be nonempty and contain no surrounding whitespace or control characters")
	}
	return nil
}

// NewClient copies an optional injected client, retaining its transport but
// limiting timeouts and disabling redirects. A nil client uses normal system TLS.
func NewClient(tenant, token string, injected *http.Client) (*Client, error) {
	if err := ValidateTenant(tenant); err != nil {
		return nil, err
	}
	if token == "" || strings.ContainsFunc(token, func(r rune) bool { return r <= ' ' || r >= 127 }) {
		return nil, errors.New("QLIK_CLOUD_TOKEN must contain a nonempty API key or OAuth access token without whitespace")
	}
	h := http.Client{Timeout: requestTimeout}
	if injected != nil {
		h = *injected
		if h.Timeout <= 0 || h.Timeout > requestTimeout {
			h.Timeout = requestTimeout
		}
	}
	h.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{tenant: strings.TrimSuffix(tenant, "/"), token: token, http: h, maxResponseBytes: responseLimit}, nil
}

// Fetch selects the latest saved version from history, then fetches that exact ID.
// It does not fetch unsaved session edits or prove a script was successfully reloaded.
func (c *Client) Fetch(ctx context.Context, appID string) (Snapshot, error) {
	if err := ValidateAppID(appID); err != nil {
		return Snapshot{}, err
	}
	path := "/api/v1/apps/" + url.PathEscape(appID)
	var app struct {
		Attributes struct {
			ID             string `json:"id"`
			Name           string `json:"name"`
			LastReloadTime string `json:"lastReloadTime"`
		} `json:"attributes"`
	}
	if err := c.get(ctx, path, "get app metadata", &app); err != nil {
		return Snapshot{}, err
	}
	if app.Attributes.ID != appID {
		return Snapshot{}, errors.New("app metadata is missing or does not match the requested app ID")
	}
	var history struct {
		Scripts []struct {
			ID             string `json:"scriptId"`
			ModifiedTime   string `json:"modifiedTime"`
			VersionMessage string `json:"versionMessage"`
		} `json:"scripts"`
	}
	if err := c.get(ctx, path+"/scripts?limit=1", "get saved script history", &history); err != nil {
		return Snapshot{}, err
	}
	if len(history.Scripts) == 0 {
		return Snapshot{}, errors.New("app has no available saved script versions")
	}
	version := history.Scripts[0]
	if ValidateAppID(version.ID) != nil || version.ID == "current" {
		return Snapshot{}, errors.New("saved script history did not provide a stable version ID")
	}
	var script struct {
		Source *string `json:"script"`
	}
	if err := c.get(ctx, path+"/scripts/"+url.PathEscape(version.ID), "get saved script", &script); err != nil {
		return Snapshot{}, err
	}
	if script.Source == nil {
		return Snapshot{}, errors.New("saved script response is missing script text")
	}
	source := []byte(*script.Source)
	digest := sha256.Sum256(source)
	return Snapshot{Source: source, Manifest: Manifest{
		SchemaVersion: 1, Tenant: c.tenant, AppID: appID, AppName: app.Attributes.Name,
		ScriptID: version.ID, ScriptModifiedTime: version.ModifiedTime, ScriptVersionMessage: version.VersionMessage,
		FetchedAt: time.Now().UTC(), LastReloadTime: app.Attributes.LastReloadTime,
		SourceFile: "script.qvs", SourceSHA256: hex.EncodeToString(digest[:]), SourceBytes: len(source),
	}}, nil
}

func (c *Client) get(ctx context.Context, path, operation string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.tenant+path, nil)
	if err != nil {
		return fmt.Errorf("%s: could not create request", operation)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("%s: %w", operation, ctx.Err())
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("%s: request timed out", operation)
		}
		// Transport errors can contain request URLs and server-controlled text.
		return fmt.Errorf("%s: HTTPS request failed; check tenant URL, TLS and network access", operation)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		hint := "request failed"
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			hint = "unauthorized; check QLIK_CLOUD_TOKEN and its expiry"
		case http.StatusForbidden:
			hint = "forbidden; check permission to read this app and its script"
		case http.StatusNotFound:
			hint = "app or saved script not found or unavailable"
		case http.StatusTooManyRequests:
			hint = "rate limited; retry later"
		default:
			if resp.StatusCode >= 300 && resp.StatusCode < 400 {
				hint = "redirects are not allowed"
			}
		}
		return fmt.Errorf("%s: HTTP %d (%s)", operation, resp.StatusCode, hint)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxResponseBytes+1))
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("%s: %w", operation, ctx.Err())
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("%s: request timed out", operation)
		}
		return fmt.Errorf("%s: could not read response", operation)
	}
	if int64(len(body)) > c.maxResponseBytes {
		return fmt.Errorf("%s: response exceeds byte limit", operation)
	}
	if err := decodeResponse(body, out); err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return nil
}
