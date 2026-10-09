# Release readiness assessment — 2026-10-09

The local `v0.1.0` candidate has passed a code review and clean-source verification.
Its Go module and imports use `github.com/noesiscarloslisboa/qlik-code-intel`, based
on the requested repository name and the authenticated GitHub account.
Dependency versions remain unchanged. No repository, commit, tag, or public
release was created during this work.

## Review findings and repairs

1. **Invalid source encoding could corrupt output silently.** A script with an
   invalid byte in a field name previously returned success, emitted invalid
   UTF-8 in map/context output, and silently substituted a character in JSON.
   Parse now rejects invalid UTF-8 with the source path before extracting facts.
   All indexing commands return exit 1 with empty stdout. Valid UTF-8, BOM,
   Unicode names, and CRLF continue to pass their source-location tests.
2. **Subcommand help swallowed write errors.** All five command help paths
   previously returned success with a failing output writer. Help now assembles
   its text before writing and reports output failure with exit 1, matching
   ordinary CLI output. Both repairs have regressions that failed before the fix
   and pass afterward.
3. **Checkout metadata made otherwise identical builds differ.** Go embedded
   local VCS state in the working-tree binary and omitted it in the source copy.
   The versioned release command now uses `-trimpath -buildvcs=false` with an
   explicit version. Working-tree and clean-source binaries built with the same
   Go/C toolchains and target were byte-identical. This is a local reproducibility
   check, not a claim of identical binaries across toolchains or platforms.

The review also covered scanner stack/boundary behavior, Tree-sitter resource
lifetime, symbol roles and source ranges, direct-edge selection, stable ranking,
whole-record/source budgets, argument handling, symlink rules, dependency
requirements, and licensing. No new grammar support or dependencies were added.
Review and selected tests do not establish the absence of all defects.

## Clean-source verification

There is no initial Git commit or configured remote. Instead of claiming a clean
clone, the check copied Git's non-ignored source inventory into a fresh temporary
directory and verified every copied file's SHA-256. Private samples, built
binaries, local caches, compiled parser libraries, and `node_modules` were absent.
Go module/build caches and the npm cache started empty. The build used the included
generated parser; grammar tooling was installed separately using `npm ci`.

Executed platform: **macOS, darwin/arm64**. Development tooling: Node.js 26.11.0,
npm 11.20.0, and pinned Tree-sitter CLI 0.25.10.

| Check | Result |
| --- | --- |
| Fresh `go mod download` and `go mod verify` | Pass; all modules verified |
| Fresh `npm ci` | Pass; pinned CLI runs |
| Go 1.27.2 `make check` | Pass: reproducible generated parser, 20/20 corpus cases, formatting, race tests, executable integration, vet, build |
| Go 1.23.12 race tests | Pass |
| Go 1.23.12 integration tests, vet, build | Pass |
| Versioned release builds with both Go versions | Pass; report `v0.1.0` |
| `make benchmark` | Pass: five runner self-tests and 40/40 public checks |
| Clang C11 scanner syntax check with `-Wall -Wextra -Werror` | Pass |
| Documented scan/map/find/deps/context and help commands | Pass against the versioned binary |
| Invalid-encoding executable probes | All five commands fail clearly without stdout |

The documented baseline still yields five files, 21 definitions, 26 references,
nine direct edges, and zero diagnostics. Map output fits its byte budgets; field
lookup and numbered context retain the expected original path and line.

## Sample results and remaining release work

The private retrieval benchmark remains 40/40. Its separate audit passes 22
source-occurrence oracles and 24 earlier budget checks; original application hashes
are unchanged. Extraction totals remain 569 LOADs, 4,869 definitions, 6,846
references, 2,724 direct edges, and 1,068 diagnostics, including zero syntax-error
and zero missing-token diagnostics. The pinned external reference audit passes
its twelve retrieval checks and matches its previous complete aggregate report.
Private source and detailed results stay ignored; this report includes aggregates
only.

The existing GitHub Actions workflow is configured for Ubuntu with Go 1.23.x and
the current stable Go version. No remote CI run has occurred because the public
repository has not been created. Linux and Windows execution were not tested
locally; Windows distribution and cross-compilation are not claimed. A public
release should follow [the publication checklist](../releasing.md), including CI
on the actual published commit.

The [candidate changelog](../../CHANGELOG.md) and [source-release instructions](../releasing.md)
are prepared. Remaining work is repository creation/publication and verification
of its remote CI run. The documented Qlik syntax limits remain; private recovered
script completeness still needs authoritative export comparison. No Qlik app,
script, include, SQL connection, or data source was executed.
