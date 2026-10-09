# Cloud Script Understanding Implementation Plan

**Goal:** Import one pinned saved Qlik Cloud script and reuse offline retrieval.

**Architecture:** A standard-library HTTP client reads app metadata and saved
script history, then fetches the selected version. A separate snapshot writer
creates a new directory containing exact UTF-8 source and a manifest. CLI routing
injects the HTTP client/environment for tests; offline parsing and retrieval stay
independent of Cloud. Execution is inline in this approved session.

**Tech Stack:** Go 1.23+, net/http, encoding/json, existing Tree-sitter CLI and
Go tests/httptest. No new module dependencies.

## Files and interfaces

- `internal/cloud/client.go`: URL/input validation and bounded GET requests.
  `NewClient(tenant, token string, httpClient *http.Client) (*Client, error)`;
  `(*Client).Fetch(ctx context.Context, appID string) (Snapshot, error)`.
- `internal/cloud/json.go`: reject invalid UTF-8 and unpaired escaped Unicode
  surrogates before JSON decoding, preventing silent source replacement.
- `internal/cloud/snapshot.go`: manifest/source types and new-directory writing.
  `WriteSnapshot(ctx context.Context, destination string, snapshot Snapshot)
  (Result, error)`. Validate hash/source identity before writing; reject existing
  destinations, symlink parents, and collisions; clean only owned files on error.
- `internal/cloud/client_test.go`, `snapshot_test.go`: original TLS API responses,
  request/version/source/error/cancellation/filesystem regressions.
- `internal/cli/cloud.go`, `cloud_test.go`: `cloud pull` flags/help and injected
  environment/client test path. Keep public `Run` unchanged as a wrapper around
  an internal runner accepting a Cloud command dependency.
- `internal/cli/run.go`: root help and Cloud routing only; existing output and
  ranking remain unchanged.
- `internal/cli/integration_test.go`: actual binary help/usage/missing-auth and
  offline imported-snapshot retrieval.
- README, AGENTS, CHANGELOG, skill, design and validation report: document actual
  behavior, source identity, credential setup, and mock/live test distinction.

## 1. Client and pinned source

- [x] Add TLS server tests expecting three GET requests: app metadata, history
  with `limit=1`, and the returned version ID (never `current`). Change the head
  version after history retrieval to prove pinning. Preserve BOM, CRLF, Unicode,
  literal expansions, tabs and no trailing newline.
- [x] Run `go test ./internal/cloud`; confirm missing implementation fails.
- [x] Implement request validation, 30-second timeout, 32 MiB response cap,
  cancellation, redirect rejection, authenticated GETs and sanitized errors.
- [x] Test empty history, missing/null source, wrong app identity, invalid JSON,
  invalid UTF-8/surrogates, HTTP 401/403/404/429/500, transport failures, oversized
  bodies and cancellation. Assert source/token never appears in errors.
- [x] Run `go test -race -count=1 ./internal/cloud`.

## 2. Snapshot writer

- [x] Test a new snapshot with exactly `script.qvs` and `manifest.json`, matching
  source hash/length, version ID and API timestamps. Include an empty valid script.
- [x] Test collision/symlink/missing-parent destinations, cancellation and a
  write failure with cleanup. Verify an existing unrelated directory is intact.
- [x] Implement owner-only files in an exclusively created new destination after
  input validation; close files and propagate write/close errors.
- [x] Run `go test -race -count=1 ./internal/cloud`.

## 3. CLI and retrieval

- [x] Test root/subcommand help, required flags, unknown flags/args, invalid URLs,
  empty credentials and stdout errors. Usage returns 2, runtime errors 1; help
  never needs credentials or contacts the network.
- [x] Implement injected Cloud routing and `cloud pull` with `--tenant`, `--app`,
  `--out`, `--json`; read only `QLIK_CLOUD_TOKEN` for authentication.
- [x] Pull a TLS fixture using the real CLI handler. Run existing scan/map/find/
  deps/context against its snapshot; check exact source lines, symbolic paths,
  role/direction, diagnostics and conservative byte budgets.
- [x] Extend the built executable tests for help/errors and offline snapshot
  retrieval. Keep live successful executable pulls distinct from handler tests.
- [x] Run `go test -race -count=1 ./internal/cli` and `make integration`.

## 4. Docs, skill and verification

- [x] Document the source-build command and environment-based authentication;
  the feature is not in the already published v0.1.1 release.
- [x] Teach the skill to read a snapshot manifest, verify its source hash before
  calling it a Cloud version, and cite tenant/app/script ID alongside source lines.
  Preserve its eight offline examples; Cloud access requires an explicit task.
- [x] Run `make test integration vet build format-check benchmark`;
  `python3 scripts/check-skill.py --binary bin/qlik-repomap`;
  `python3 -m unittest discover -s scripts -p 'test_*.py'`.
- [x] Validate skill frontmatter with the bundled skill-creator validator when
  available. Review generated diff and secrets/private-file inventory.
- [x] Record executed results. A live acceptance check needs a user-supplied
  tenant/app and credential configured outside chat; report it pending if absent.

## Execution record

Implemented and locally verified on 2026-10-09. See
[the validation assessment](../../validation/cloud-script-assessment.md) for
executed checks and limits. Live acceptance on 2026-10-09 initially reached app
metadata but received HTTP 403 for script history. A later retry against the same
selected app downloaded a pinned saved script successfully. An independent read
of that exact Cloud version matched the snapshot bytes, and all 24 selected
private retrieval checks passed across 14 source-reviewed questions. Unsupported
construct diagnostics remain visible. Private source, manifests, hashes, queries,
and detailed audits remain ignored; the assessment publishes aggregate results.
