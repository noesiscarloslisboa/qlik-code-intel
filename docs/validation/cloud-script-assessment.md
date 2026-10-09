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

Attempted on 2026-10-09 with the user's locally supplied credential and a freshly
built executable. App metadata was accessible (HTTP 200), but the saved-script
history request returned HTTP 403. The CLI returned exit code 1 with a sanitized
permission hint and created no snapshot. A separate metadata read confirmed app
access; its reported privilege hints omitted `update` and `reload`. These hints
are consistent with the script-history denial but do not establish the account's
exact assigned space roles.

Successful script acquisition, saved-version comparison, and retrieval against
this live app remain pending script access or another explicitly selected app.
No real Cloud script was downloaded. Detailed audit output and tenant metadata
are stored under the ignored `.cache/qlik-cloud/` directory; this report contains
only aggregate results.

Qlik documents that shared-space `Owner` and `Can manage` roles do not grant
editing access to other users' load scripts; `Can edit data in applications`
provides that capability. Check access for the account represented by the token
or select an app it owns. See [shared-space permissions](https://help.qlik.com/en-US/cloud-services/Subsystems/Hub/Content/Sense_Hub/Spaces/managing-shared-spaces.htm).

Fixture tests establish the documented request and local retrieval behavior;
they do not prove live tenant permissions, every API response variation, complete
Qlik syntax coverage, or script reload validity. Last reload time is separate
metadata, and saved source excludes unsaved editor/session changes. The skill's
Cloud provenance instructions have been reviewed, but autonomous assistant
behavior on downloaded live source has not been evaluated.
