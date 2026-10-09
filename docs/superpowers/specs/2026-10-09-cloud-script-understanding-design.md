# Qlik Cloud script understanding

Status: implemented and live-checked on 2026-10-09. The user selected understanding
and debugging app scripts as the first Cloud workflow and approved this scope.
The [validation assessment](../../validation/cloud-script-assessment.md) records
live acceptance separately from local tests, including remaining limitations.

## Outcome

Fetch a saved script version from one explicitly selected Qlik Cloud app into a
local snapshot. Use the existing Tree-sitter parser and retrieval commands to
explain fields, variables, tables, direct source dependencies, and relevant source
locations. Keep the fetched app/version identifiable when an assistant cites code.

Debugging means gathering source context for a reported problem and retaining
parser diagnostics. It does not establish Qlik runtime validity, execute a reload,
or automatically retrieve logs in this increment.

## Approach and alternatives

1. A read-only script importer is recommended first. It makes the current CLI
   useful with Cloud while keeping source extraction and retrieval independent
   of network access.
2. A lineage-only connector can describe reported dependencies but does not
   supply the complete editable script needed for precise explanations.
3. A combined script/lineage integration is the longer-term direction. It needs
   separate identity, provenance, freshness, and matching rules and is deferred
   until the source importer works end to end.

## User interface

Implemented command (not available in v0.1.1):

```sh
mkdir -p .cache/qlik-cloud
qlik-repomap cloud pull --tenant https://example.eu.qlikcloud.com \
  --app APP_ID --out .cache/qlik-cloud/app-snapshot
```

Require tenant URL, app ID, and destination. Read a supplied API key or OAuth
access token from `QLIK_CLOUD_TOKEN`, outside command arguments and the snapshot.
The caller obtains that credential through their existing tenant setup. Automatic
OAuth login, token refresh, credential storage, and browser authentication are
outside this increment. Tenant permissions still govern access to scripts.

The destination must not already exist, and its parents must already exist
without symlinks; repeated pulls use a new destination.
Report the snapshot directory and chosen script version. An optional `--json`
returns a summary with snapshot/manifest paths and app/script IDs, without source
text or credentials. Usage errors return 2; request or filesystem failures return
1. Help and successful results go to stdout; errors go to stderr.

Existing commands then work offline:

```sh
qlik-repomap scan --root .cache/qlik-cloud/app-snapshot --strict
qlik-repomap map --root .cache/qlik-cloud/app-snapshot --query Revenue --tokens 2000
qlik-repomap find Revenue --root .cache/qlik-cloud/app-snapshot --kind field
qlik-repomap context Revenue --root .cache/qlik-cloud/app-snapshot --tokens 2000
```

Default examples use the already ignored `.cache/` directory. It is skipped by
ordinary repository scans, while an explicit snapshot root remains scannable.
Snapshot contents stay local; they are not fixtures or release assets.

## Acquisition and source identity

Use these documented, read-only Apps API requests:

1. `GET /api/v1/apps/{appId}` for app identity/name and reported last reload time.
2. `GET /api/v1/apps/{appId}/scripts?limit=1` for the latest saved script metadata.
3. `GET /api/v1/apps/{appId}/scripts/{scriptId}` for that exact version's source.

The history endpoint documents newest-first ordering. Pin the returned script ID
when fetching text so a concurrent save cannot silently substitute another
version. The snapshot is the latest saved version selected by the history request,
not a guarantee of the latest version at the end of the pull. Unsaved browser or
Engine-session edits are outside that contract. If there is no saved version or
the selected version disappears before retrieval, return a clear error.

Write exactly two files:

- `script.qvs`: the decoded script string encoded as UTF-8, preserving newlines,
  tabs, BOM, comments, section markers, and symbolic paths/variables. Add no
  headers or trailing newline. Reject malformed source; do not repair it.
- `manifest.json`: schema version, tenant base URL, app ID/name, pinned script ID,
  script modification time/message when supplied, retrieval time in UTC, exact
  source SHA-256 and byte length, source filename, and last reload time when
  available. Missing metadata stays absent, rather than guessed.

The script response and metadata are not an atomic tenant snapshot. In particular,
last reload time does not prove that the downloaded script was successfully
reloaded. Source citations refer to `script.qvs` and original one-based lines;
the assistant associates them with the manifest's tenant/app/version. Do not
claim exported line numbers are tab-local Cloud editor coordinates.

Keep one app per retrieval root. Do not combine scripts from multiple apps into
the current name-based dependency index. Later cross-app retrieval must introduce
explicit app/tenant identity rather than merge equal table or field labels.

## Implementation boundaries and failure behavior

- Add focused `internal/cloud/` HTTP and snapshot code using Go's standard library.
  Add CLI dispatch/flags without changing grammar or existing retrieval behavior.
- Accept only an HTTPS tenant base URL without user information, query, fragment,
  or a non-root path. Encode app/version path segments. Verify TLS normally and
  reject redirects so credentials cannot be forwarded to another endpoint.
- Use context cancellation, a 30-second timeout per request, and a 32 MiB limit
  per response. Reject larger responses rather than truncate them. Report
  HTTP status and useful access/rate-limit hints without dumping remote response
  bodies, script contents, or authorization headers. Do not retry indefinitely.
- Validate all fetched inputs before creating the snapshot. Create only a new
  destination, never overwrite existing files or follow a destination symlink,
  and clean up files created by this invocation if writing fails. Use owner-only
  permissions where supported by the operating system.
- Unsupported Qlik syntax is retained in the downloaded source. Subsequent scans
  produce the existing diagnostics and partial facts; `cloud pull` does not make
  a clean-parse or reload-validity claim.
- Teach the companion skill to inspect a snapshot manifest and cite its identity
  when present. It may fetch only a user-selected app when the task authorizes
  Cloud access; ordinary retrieval commands remain offline.

## Verification and acceptance

Use original API-response fixtures and Go `httptest` servers, with injected HTTP
clients for tests. No credentials, private script exports, or upstream GPL source
belong in tests. Cover the request methods/paths, authentication header, version
pinning across a concurrent save, preserved UTF-8/CRLF/section markers, empty or
missing script versions, malformed responses, unauthorized/missing app responses,
rate limits, cancellation, redirects, response size limits, and destination
collisions/write failures. Assert that failures expose no credentials or source.

Exercise the CLI handler end to end with an injected client that trusts the TLS
test server. Pull an original script, scan the snapshot, and assert its field
definition, direct dependency, numbered context, diagnostics, and output budgets.
Validate manifest hash and version identity. Built-executable tests cover help,
invalid flags, missing credentials before network access, and offline snapshot
retrieval. A successful Cloud pull in the actual executable remains part of live
acceptance; do not claim that a handler test establishes it. Add no insecure-TLS
flag to production. Run `make test integration vet build` and `make benchmark`;
run the skill example checks and packaging self-tests if their files change.

A live acceptance check requires a tenant URL, a supplied authorized credential,
and one app ID. Compare the snapshot with that app's saved script version and
confirm source citations. Report mock-server tests and live tenant validation
separately; a missing live check must remain explicit.

## Follow-ups

After this increment, consider reload-log retrieval for reported failures, then
optional Cloud lineage/impact context. Both need their own version/freshness
rules. Cloud lineage must retain Qlik resource identifiers and its provenance;
unresolved matches must never acquire fabricated source locations. Deploying
scripts, triggering reloads, space-wide discovery, advanced inferred lineage,
MCP, and a GUI are outside this increment.

## References

- [Qlik Apps API](https://qlik.dev/apis/rest/apps/): app metadata, saved script
  history, and source retrieval. The newer analytics namespace currently has no
  replacement for these script endpoints.
- [Qlik lineage API](https://www.qlik.dev/apis/rest/lineage-graphs/) and
  [lineage refresh/coverage limits](https://help.qlik.com/en-US/cloud-services/Subsystems/Hub/Content/Sense_Hub/Catalog/lineage.htm)
  inform the deferred integration.
- Existing `internal/qlik/`, `internal/repomap/`, and `AGENTS.md` define source
  accuracy, diagnostics, direct dependency semantics, and conservative budgets.
