# Parser and retrieval accuracy implementation plan

**Goal:** Improve practical QVS parsing and precise retrieval while retaining direct
source facts and the existing CLI contract.

**Architecture:** Extend the grammar's expression/name nodes, add explicit LOAD
range records and adjacent-stage edges, and correct statement ranking/selection.
No additional runtime dependencies or persistent state.

**Tech stack:** Existing Go standard library, Tree-sitter Go v0.25.0, generator
0.25.10, CGO. Execute inline under the user's continuation authorization.

## 1. Reproduce accuracy defects

- [x] Add `internal/qlik/accuracy_test.go` covering bare LET variables,
  interpretation functions, symbolic names, and preceding LOAD stage ownership.
- [x] Add `internal/repomap/accuracy_test.go` covering focused compact maps,
  exact-match ranking, anonymous-stage context, and unrelated-include isolation.
- [x] Run the new tests and confirm failures in the existing implementation.

## 2. Grammar and index

- [x] Update `tree-sitter-qlik/grammar.js` with LET-specific expressions whose
  bare names are variable-reference nodes, `#` function names, and symbolic names.
- [x] Add original corpus/fixture examples and query assertions. Regenerate the
  parser, review actual trees, then update the expected corpus trees.
- [x] Update query/highlight files for the new variable/function node kinds.
- [x] Add LOAD records in `internal/qlik/model.go`; update `parse.go` to record
  stages, direct preceding edges, and diagnostics at chain boundaries.
- [x] Clear CGO cache and run `go test ./internal/qlik`; fix all failures.

## 3. Retrieval

- [x] Seed map candidates from LOAD records, including wildcard-only stages;
  display direct preceding inputs without flattening schemas.
- [x] Rank the strongest match before structural importance. Try a compact
  focused summary retaining one best matching field/reference/variable.
- [x] Add anonymous LOAD ranges to context's direct-neighbor lookup; avoid
  seeding unrelated file includes from every symbol lookup.
- [x] Run `go test ./internal/repomap` and the existing CLI tests.

## 4. Finish and verify

- [x] Add executable tests and standalone accuracy fixtures under
  `testdata/accuracy/`, preserving the original sample repository and its counts.
- [x] Update README, grammar README, and AGENTS with syntax/JSON/diagnostic changes.
- [x] Run `make check`, uncached race/integration tests, and inspect actual
  `scan`, `map`, `find`, `deps`, and `context` output against the accuracy fixtures.

## Verified result — 2026-10-08

- `make check` passed: reproducible generation, all nine grammar cases,
  formatting, race tests, executable integration tests, vet, and build.
- Fresh `go test -race -count=1 ./...` and
  `go test -count=1 -tags=integration ./...` passed after the final regression case.
- The original repository still scans to 21 definitions, 26 references, and
  nine direct dependencies. The accuracy fixtures scan strictly with no diagnostics:
  20 definitions, 21 references, and 15 direct dependencies across two files.
- Inspected actual scan/map/find/deps/context output. The Revenue lookup retrieves
  its direct anonymous input LOAD and keeps the third stage outside that context.
- No runtime dependencies, persistent state, evaluated lineage, MCP, or GUI added.
