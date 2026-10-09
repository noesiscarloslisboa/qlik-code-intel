# Reference sample validation implementation plan

**Goal:** Validate the CLI on the requested external samples and repair bounded
literal-fragment recovery and unary NOT recognition without importing GPL code or
adding runtime dependencies.

**Architecture:** A pinned external corpus feeds the existing executable. A small
explicit unsupported syntax node bounds recovery; standard-library validation
records source-backed results and budgets without executing Qlik.

**Tech stack:** Existing Go/Tree-sitter toolchain; Python standard library for the
optional audit runner. Execute inline within the user's sample-validation request.

## External baseline

- [x] Enumerate the full upstream tree, pin a commit, and fetch only QVS samples
  and LICENSE into a temporary directory outside this repository.
- [x] Verify all Git blob hashes and run scan/JSON against all ten files.
- [x] Inspect sample diagnostics and syntax trees; distinguish invalid and
  unsupported constructs from the balanced-literal recovery defect.

## Recovery regression

- [x] Add an original Go regression and corpus example for standalone balanced
  literals followed by a supported include and LOAD; verify the regression fails.
- [x] Add `unsupported_literal_statement` to the root grammar and extraction
  diagnostics. Never extract phantom references from those fragments.
- [x] Regenerate, inspect expected trees, clear CGO cache, and run focused tests.
- [x] Verify that unsupported literal fragments break preceding LOAD adjacency.
- [x] Reproduce unary NOT keyword/identifier confusion independently, prioritize
  the operator in LOAD/LET expressions, and retain longer `Notable` identifiers.

## Repeatable audit and verification

- [x] Add a manifest of the ten external file paths/Git blob hashes and an
  opt-in validation runner accepting `--samples-dir` and `--binary`.
- [x] Verify scan facts, selected locations, deterministic maps/context, and
  UTF-8 byte budgets against the external corpus; preserve aggregate output.
- [x] Add executable coverage using original recovery input; update README/AGENTS
  and write a source-linked assessment with remaining limits and exact revision.
- [x] Run full project checks and fresh race/executable integration tests.

## Verification result — 2026-10-08

- Both recovery and unary NOT regressions failed before their fixes and passed
  after regeneration. Reviewed all new syntax trees before updating expectations.
- `make check` passed: generated parser reproducibility, all 11 grammar cases,
  formatting, fresh race-enabled Go tests, executable integration, vet, and build.
- The opt-in audit passed against all ten pinned external blobs. The final build
  produced an aggregate report identical to the checked-in results: 24 LOADs,
  85 definitions, 170 references, 76 direct edges, and 71 retained diagnostics.
- All 12 map/context budget/determinism cases passed. Selected repeated definitions,
  tabbed aliases, direct dependencies, recovery locations, and source text matched.
- A deliberately different sample revision was rejected before indexing.
- Original sample repository counts remain 21 definitions / 26 references / nine
  edges. The accuracy fixtures still scan strictly without diagnostics.
- GPL samples remain only in the separate temporary evaluation directory; reports
  and the manifest contain no upstream source. No runtime dependencies added.
