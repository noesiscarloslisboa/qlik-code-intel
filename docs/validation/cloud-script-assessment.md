# Qlik Cloud script import assessment

Validated on 2026-10-09 with Go 1.27.2 on macOS arm64. This feature is included in
the published v0.2.0 release; see the
[release assessment](v0.2.0-release-assessment.md) for native checks and downloads.

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

A retry after the user reported updating script access also returned HTTP 403,
with no snapshot created. A fresh metadata read reported the same privilege hints.
An authenticated current-user lookup confirmed that the token's account was active
and did not own the selected app. These observations did not establish a specific
role assignment. Qlik documents a session-permission refresh requirement:
close all affected app tabs and wait at least two minutes before reopening.

A later retry against the same selected app succeeded. The actual executable
downloaded a pinned saved version containing 17,545 UTF-8 bytes and 608 lines into
a new snapshot with exactly `script.qvs` and `manifest.json`. The source hash and
length matched the manifest. An independent authenticated read of that exact
script ID returned HTTP 200 and identical decoded source bytes. No permission
changes or app writes were made by the tool; the specific access or session change
that resolved the earlier denial was not established.

The offline scan retained 141 symbol occurrences, including 12 table definitions,
36 field definitions, and 43 variable definitions, plus 12 LOAD records and 21
direct dependency edges. It reported 72 diagnostics: 66 `unsupported` and six
`unsupported-control`. Strict scan therefore returned exit code 1 while preserving
the supported facts in its JSON output. The diagnostics do not establish runtime
invalidity, and the extracted facts do not imply complete script coverage.

A private, source-reviewed, SHA-256-pinned oracle checked selected table, field,
variable, and literal source matches; compact maps; original numbered context;
and dependencies with explicit endpoint kinds and directions. All 24 checks
passed across 14 questions, with map/context budgets of 256, 512, and 1,024 UTF-8
bytes. The runner also checked deterministic results, source bounds, first
answers/blocks, and budget feasibility. These selected cases are not exhaustive
precision or recall measurements. A useful general map used 2,022 bytes within a
2,048-byte budget.

Source, manifests, detailed audit output, hashes, private queries, and tenant
metadata remain under the ignored `.cache/qlik-cloud/` directory; this report
contains only aggregate results.

### Parser follow-up on the same snapshot

The 2026-10-09 operation-support increment recognized all 60 targeted statements:
18 field renames, five table renames, 18 field drops, four table drops, and 15
TRACE statements. Diagnostics fell from 72 to 12 (six `unsupported` and six
`unsupported-control`). Strict scan still returns 1, preserving partial-coverage
reporting. No new source was fetched or modified for this comparison.

Every baseline symbol occurrence, edge, and LOAD range remained present. The
updated index contains 221 symbol occurrences: 43 variable definitions, 20
variable references, 17 table definitions, 54 field definitions, 66 field
references, 13 table references, and eight source references. It records 32
direct edges, 12 LOADs, and 115 supported statement ranges. New table/field
definitions here include explicit rename targets; these are source facts, not
assertions about a final runtime data model. The 11 additional edges are explicit
variable uses, with no invented rename/drop lineage.

All original 24 private retrieval checks still passed. Seven additional
source-reviewed questions cover explicit renamed definitions, focused maps,
numbered context, and a TRACE variable reference. The expanded oracle passed
all 39 checks across 21 questions at the same byte budgets. Three additional
source checks verified drop references and a TRACE expansion at their original
locations. All indexed symbol/statement ranges stayed within the source bounds.
A general map used 2,042 bytes within a 2,048-byte budget. Detailed artifacts and
new private queries remain ignored. These checks remain selected examples, not
exhaustive accuracy or runtime validation.

Qlik documents that shared-space `Owner` and `Can manage` roles do not grant
editing access to other users' load scripts; `Can edit data in applications`
provides that capability. This distinction informed the access troubleshooting.
See [shared-space permissions](https://help.qlik.com/en-US/cloud-services/Subsystems/Hub/Content/Sense_Hub/Spaces/managing-shared-spaces.htm).

Fixture tests establish the documented request and local retrieval behavior. The
live check confirms acquisition and selected retrieval for one authorized app;
it does not prove access across other tenants or role combinations, every API
response variation, complete Qlik syntax coverage, or script reload validity.
Last reload time is separate metadata, and saved source excludes unsaved
editor/session changes. The skill's Cloud provenance instructions have been
reviewed, but autonomous assistant behavior on downloaded live source has not
been evaluated.
