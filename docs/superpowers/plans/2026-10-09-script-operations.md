# Script Operations Implementation Plan

**Goal:** Retrieve explicit RENAME/DROP names and TRACE expansions without
evaluating Qlik execution.

**Architecture:** Dedicated grammar nodes feed AST extraction and existing
definition/reference queries. Minimal supported-statement ranges distinguish
operation summaries from LOAD summaries; roles and symbolic identities remain
source facts. Execution is inline in this approved session.

**Tech Stack:** Existing Tree-sitter 0.25, Go standard library, existing Go
bindings and Python standard-library benchmark runner. No new dependencies.

## 1. Original acceptance fixtures and failing tests

- [x] Create `testdata/accuracy/operations.qvs` with original LOAD/SET setup,
  explicit and mapped field/table renames, scoped/global drops, mapping drops,
  quoted/composed names, and TRACE text. Add `internal/qlik/operations_test.go`
  asserting exact reference/definition counts, normalized names, original ranges,
  no inferred ownership/runtime edges, retained earlier definitions, and no
  message words as symbols. Example contract:
  ```qlik
  RENAME FIELDS [Raw Price] TO Price, Code TO [Public Code];
  RENAME TABLE Stage TO Published;
  RENAME FIELDS USING FieldNames;
  DROP FIELDS Price, [Public Code] FROM Published, Archive;
  TRACE Completed $(vBatch);
  ```
- [x] Add malformed-operation and preceding-boundary regressions; retain opaque
  unknown-statement coverage by using unknown keywords instead of TRACE in
  existing recovery tests. Keep a separate TRACE no-phantom-symbol regression.
- [x] Run `go test ./internal/qlik -run Operations -count=1`; observe missing
  facts before implementation.

## 2. Grammar, trees, queries, and extraction

- [x] Add `rename_field_statement`, `rename_table_statement`,
  `drop_field_statement`, `drop_table_statement`, `trace_statement`, pair/list
  nodes, and text/name rules in `tree-sitter-qlik/grammar.js`. Reuse existing
  variable expansion nodes and name normalization. New statements require
  terminating semicolons; unknown forms remain diagnosed.
- [x] Run `npm ci` in `tree-sitter-qlik`, then `make generate`. Inspect the tree
  of `testdata/accuracy/operations.qvs` with the Tree-sitter CLI before adding
  reviewed expectations in `tree-sitter-qlik/test/corpus/operations.txt`.
- [x] Extend `definitions.scm`, `references.scm`, and `tags.scm` for rename old
  references/new definitions, USING table references, and drop targets/scopes.
  Assert exact captures, including raw quote delimiters and separate expansion
  references, in the new Go query tests.
- [x] Add `Statement` location/kind/byte-range records in `internal/qlik/model.go`.
  Append supported valid statements in `parse.go`; route operations to focused
  extraction in `internal/qlik/operations.go`. Extract operation variable edges
  from `file -> variable`, not invented runtime table/field dependencies.
- [x] Run grammar tests and focused Go extraction/query tests. Fix failures,
  preserving unknown-body opacity and LOAD adjacency rules.

## 3. Compact retrieval

- [x] Teach `internal/repomap/map.go` to use the statement kind when grouping
  candidates. Add `internal/repomap/operations.go` for summaries using indexed
  roles: `rename field old -> new`, `rename table old -> new`, `rename fields
  using map`, `drop fields names | from: tables`, `drop tables names`, and
  `trace | uses: $(variable)`.
- [x] Add `internal/repomap/operations_test.go` for useful general/focused maps,
  first matching pairs/targets in compact output, deterministic output, budgets,
  preserved source lines, and no inferred rename/drop deps. Use the original
  fixture and multiple byte budgets; reject infeasible expected lines.
- [x] Add an executable integration check for scan/map/find/context on the
  original operation fixture, alongside existing binary checks.
- [x] Run `make benchmark` and focused retrieval/integration tests. Existing
  public benchmark outputs and direct-edge semantics must remain accurate.

## 4. Verification, private comparison, and documentation

- [x] Run `make test-grammar check-generated format-check test integration vet
  build benchmark`, fixing every relevant failure. HTTPS test listeners need
  normal execution permission in this sandbox, never a TLS bypass.
- [x] Re-run the saved private Cloud `questions.json` oracle on unchanged
  source with the rebuilt executable. Save fresh scan/map/benchmark artifacts
  under ignored cache; publish aggregate counts only.
- [x] Update README, grammar README, AGENTS, CHANGELOG, and validation record
  with syntax roles, additive statement JSON, no runtime mutation, remaining
  unsupported diagnostics, and checks actually executed.
- [x] Run `git diff --check`, inspect public diff for private data and generated
  artifact consistency, and commit the verified increment locally. Report exact
  validation results and the release boundary.

## Execution record

Implemented inline on 2026-10-09. All 22 grammar cases and generated consistency
checks passed. Race/integration/vet/build/format checks passed; the public
benchmark passed 40 checks. The unchanged private Cloud snapshot retained every
baseline fact and all 60 targeted operations, reducing diagnostics from 72 to
12. Its expanded oracle passed 39 checks across 21 questions, plus three
source-reviewed operation checks. A demonstrated whole-file variable-context
regression was fixed and the affected suites re-run. A TRACE macro query
regression was also fixed with text-only arguments and nested expansions, then
the full checks were re-run. See the
[assessment](../../validation/script-operations-assessment.md) for evidence and
limits. No release was published by this parser increment.
