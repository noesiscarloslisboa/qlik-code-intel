# Milestone One Implementation Plan

**Goal:** Deliver an end-to-end QVS scanner and repository map generator with reusable grammar and basic retrieval commands.

**Architecture:** A single Go module combines a generated Tree-sitter C grammar and binding with focused parsing, index, retrieval, and CLI packages. No persistence or runtime script execution.

**Tech Stack:** Go standard library, official Tree-sitter Go bindings, pinned Tree-sitter CLI for development, C compiler for CGO.

Implementation stays inline in this session. The user's requested milestone authorizes building and testing the design.

## 1. Grammar and binding

- [x] Create `tree-sitter-qlik/grammar.js`, `package.json`, `tree-sitter.json`, `bindings/go/binding.go`, and generated `src/` artifacts.
- [x] Add corpus examples in `tree-sitter-qlik/test/corpus/` for assignments, expansions, quoted names, loads, sources, stores, includes, joins, and concatenation.
- [x] Run `npm ci && npm run generate && npm test` in `tree-sitter-qlik/`; fix generation conflicts and unexpected errors.
- [x] Add `queries/tags.scm`, `queries/definitions.scm`, `queries/references.scm`, and `queries/highlights.scm`. Execute query compilation/captures in Go tests.

## 2. Source index

- [x] Create `internal/qlik/model.go`, `parse.go`, and `scan.go` with owned symbol/edge/diagnostic records, locations, deterministic scanning, and resource cleanup.
- [x] Add original multi-file fixtures under `testdata/repository/`.
- [x] Write parser/extractor tests asserting exact definitions, field aliases/references, expansions inside sources/strings, edge direction, syntax recovery, and one-based lines.
- [x] Run `go test ./internal/qlik/...`; fix failures before retrieval implementation.

## 3. Maps and retrieval

- [x] Create `internal/repomap/budget.go`, `map.go`, `find.go`, `deps.go`, and `context.go`.
- [x] Test exact-name ranking, repeated symbols, deterministic ordering, bounded UTF-8 output, source-line fidelity, overlapping context, and direct upstream/downstream edges.
- [x] Run `go test ./internal/repomap/...`.

## 4. CLI

- [x] Create `internal/cli/run.go` and `cmd/qlik-repomap/main.go` using injected output/error writers and signal cancellation.
- [x] Support commands `scan`, `map`, `find`, `deps`, `context`; validate argument counts, flags, positive budgets, and kinds/directions.
- [x] Add unit CLI tests for help, invalid usage, JSON index, diagnostics, and commands against the fixture repository.
- [x] Add a tagged integration test that builds and invokes the executable against fixtures.

## 5. Documentation and milestone verification

- [x] Add root `README.md`, `AGENTS.md`, `LICENSE`, `CONTRIBUTING.md`, grammar README, `Makefile`, and `.github/workflows/ci.yml`.
- [x] Document install prerequisites, commands, output/locations, budgets, supported/unsupported syntax, no script execution, query integration limitations, reference provenance, and module naming before publication.
- [x] Run grammar tests, `go test -race ./...`, `go test -tags=integration ./...`, `go vet ./...`, `go build ./cmd/qlik-repomap`, and clean generated-artifact checks.
- [x] Run actual `scan`, `map`, `find`, `deps`, and `context` commands on the sample repository. Inspect output and fix misleading or missing symbols before declaring completion.
