# Qlik Cloud script import assessment

Validated on 2026-10-09 with Go 1.27.2 on macOS arm64. This feature is in the
development source and is not part of the published v0.1.1 release.

## Implemented scope

`cloud pull` reads one selected app's metadata, newest saved script history entry,
and the source at that exact script ID. It writes a new directory containing
`script.qvs` and a schema-version-1 manifest. Source SHA-256 and byte length bind
local source to the selected version. Authentication uses `QLIK_CLOUD_TOKEN`;
there are no new dependencies, app changes, reloads, or lineage API requests.

## Executed checks

- `make test`: race-enabled Go tests passed, including the new HTTP client,
  snapshot writer, and CLI handler tests.
- `make integration`: actual executable help, invalid flags, missing credentials,
  and offline imported-snapshot retrieval passed, alongside the existing checks.
- `make vet`, `make build`, and `make format-check`: passed.
- `make benchmark`: all 40 selected retrieval checks passed across the existing
  20 source-reviewed questions at the configured budgets.
- `python3 scripts/check-skill.py --binary bin/qlik-repomap`: all eight documented
  offline commands, source/role/direction/budget checks, and strict partial indexing
  passed.
- `python3 -m unittest discover -s scripts -p 'test_*.py'`: 12 checks passed.
- The bundled skill-creator `quick_validate.py` accepted the updated skill.

The Go suites require local listening sockets for their HTTPS fixture servers.
They were run with execution permission after the workspace sandbox rejected
localhost binding. This is a test-environment constraint, not a production TLS
override; the CLI has no insecure-TLS flag.

## Original fixture coverage

Client tests assert the three authenticated GET requests and pinned version path;
a moving head cannot replace the selected source. BOM, CRLF, Unicode, tabs, section
markers, literal expansions, and a missing final newline retain their exact bytes.
Valid empty scripts and Unicode pairs are accepted. Missing history/source,
wrong app identity, malformed JSON/Unicode, transport failures, untrusted TLS,
HTTP access/rate-limit/server errors, redirects, cancellation, timeouts, and
oversized responses fail without exposing remote bodies or credentials.

Writer tests check manifest/source identity, permissions where supported,
exclusive destinations, symlink rejection, write failure, cancellation, and
cleanup that preserves unrelated concurrent files. CLI handler tests pull an
original script through HTTPS, then run scan/map/find/deps/context offline. They
retain unsupported-statement diagnostics, original source lines, symbolic QVD
paths, direct dependency direction, and output budgets. A completed snapshot
survives a summary-output failure.

## Live acceptance status

Pending: the user selected a tenant and app, but no local credential was available
at validation time. No real Cloud script was downloaded. Successful executable
pulls against that tenant and comparison with its saved script version remain
unverified. Tenant details and any later source/audit output belong in ignored
local storage, not this report.

Fixture tests establish the documented request and local retrieval behavior;
they do not prove live tenant permissions, every API response variation, complete
Qlik syntax coverage, or script reload validity. Last reload time is separate
metadata, and saved source excludes unsaved editor/session changes. The skill's
Cloud provenance instructions have been reviewed, but autonomous assistant
behavior against a live tenant has not been evaluated.
