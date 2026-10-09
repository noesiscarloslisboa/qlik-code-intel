# Release Readiness Implementation Plan

**Goal:** Review and verify the milestone implementation from a clean source tree
and prepare a local v0.1.0 release candidate under the selected GitHub namespace.

**Architecture:** Keep the current grammar and package boundaries. Add only
demonstrated correctness fixes and source-release documentation. Execute inline
within the user's approved review scope; publishing remains a separate action.

**Tech Stack:** Go, CGO/C, Tree-sitter 0.25.10, npm, optional stdlib Python benchmark.

- [x] Review parsing/recovery, extraction, retrieval, CLI behavior, dependencies,
  CI configuration, and documentation. Confirm repository name/account read-only.
- [x] Add invalid UTF-8 parser/CLI regressions and a command-help writer regression.
  Run `go test ./internal/qlik ./internal/cli -run 'TestRejectInvalidUTF8|TestHelpOutputErrors' -count=1`
  and confirm the failures before fixing production code.
- [x] Reject invalid UTF-8 before parsing, with `utf8.Valid(source)` and a
  path-bearing error; build command help in a `strings.Builder` and propagate the
  final stdout write error. Repeat the focused tests and confirm they pass.
- [x] Set `go.mod` to `module github.com/noesiscarloslisboa/qlik-code-intel` and
  update Go imports, version embedding comments, and contributing instructions.
  Run `go mod tidy`; dependency versions must remain unchanged.
- [x] Add `CHANGELOG.md`, `docs/releasing.md`, and README release instructions
  describing a source release, CGO requirements, tested platforms, exact checks,
  version embedding via the module path, and remaining syntax/recovery limits.
- [x] Copy only non-ignored publishable source into a temporary clean tree, verify
  hashes, and run documented build/CLI commands using a new Go build cache.
- [x] Run clean-tree `npm ci`, `make check`, and `make benchmark`; confirm installed
  Go version and run build/tests/integration/vet with Go 1.23.12 if obtainable.
- [x] Rerun private/reference audits against the candidate binary, compare aggregate
  results, and record the final review findings and executed platform/toolchains in
  `docs/validation/release-readiness-assessment.md`. Check privacy exclusions and
  local documentation links. Do not commit or publish.

## Verified outcome

Executed on darwin/arm64 with Go 1.27.2 and 1.23.12. Full clean-source checks pass,
including all 20 grammar cases; both public/private benchmarks remain 40/40.
The separate private source audit and pinned reference report remain unchanged.
Versioned working-tree and clean-source builds are byte-identical when built with
the same toolchain using `-trimpath -buildvcs=false` and explicit version injection.
Details and target limitations are in the readiness assessment. Remote creation,
commits, tags, publication, and authoritative export comparison remain separate.
