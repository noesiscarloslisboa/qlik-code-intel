# QlikView syntax coverage implementation plan

**Goal:** Retrieve explicit source facts from the remaining local QlikView syntax
gaps, with regression tests and verified queries.

**Architecture:** Extend existing grammar rules and hierarchy extraction. Keep
the public symbol/edge model, scanners, ranking, and budget implementation intact.
Execute inline within the user's authorized parser accuracy work.

**Tech stack:** Tree-sitter CLI 0.25.10, official Go bindings, Go standard library.

- [x] Review the remaining private syntax/missing-token locations and official
  Qlik documentation. Record the distinction between documented syntax and
  observed forms without runtime validation.
- [x] Add original failing cases in `internal/qlik/coverage_test.go`: empty LET,
  omitted LOAD/LET call arguments, GROUP BY references, BUNDLE opacity, typed
  HIERARCHY inputs/outputs, optional holes, malformed prefixes, and query captures.
- [x] Modify `tree-sitter-qlik/grammar.js`: make LET value optional; permit omitted
  call arguments; add BUNDLE/INFO and position-specific HIERARCHY prefix nodes;
  accept structured GROUP BY expressions. Regenerate with `make generate`.
- [x] Modify `internal/qlik/parse.go` to index explicit hierarchy output fields.
  Add hierarchy definitions to tags/definitions queries. Check source locations,
  output ownership, input references, and absence of fabricated hierarchy fields.
- [x] Add original corpus cases in `tree-sitter-qlik/test/corpus/coverage.txt`;
  inspect actual trees before recording expectations. Run parser/query tests.
- [x] Add `testdata/qlikview/coverage.qvs` and executable checks for strict scan,
  hierarchy field lookup, dependencies, maps, and original numbered context.
- [x] Run `make check`; fix any failures before proceeding.
- [x] Rerun `testdata/real_samples/audit.py` and the pinned reference audit.
  Add private source oracles for repaired locations, keep app hashes unchanged,
  and compare map/context bounds, determinism, and original source lines.
- [x] Update README, grammar README, AGENTS, acceptance checks, and aggregate
  QlikView assessment with verified results and the pending export comparison.

Commands use `GOMODCACHE=/private/tmp/qlik-code-intel-modcache` and
`GOCACHE=/private/tmp/qlik-code-intel-gocache`. `make generate` clears the CGO cache;
Tree-sitter tests need their normal parser cache. No private source is committed.

Verification: all 20 corpus cases pass, generated artifacts reproduce exactly,
race-enabled Go tests and built-executable checks pass, and formatting/vet/build
pass. The unchanged handwritten scanner passes Clang warnings as errors.
The original repository retains 21 definitions, 26 references, and nine edges.
Both original QlikView fixtures scan strictly with no diagnostics.

The private audit verifies unchanged app hashes, 22 selected source occurrences,
and 24 deterministic budget/source checks. It records 569 LOADs, 4,869 definitions,
6,846 references, 2,724 edges, zero syntax errors/missing tokens, and 1,068 remaining
unsupported/control/implicit-target/unresolved-source diagnostics. The pinned
reference audit passes twelve checks with decoded JSON equal to its prior report.
Authoritative export comparison remains pending and no runtime reload was run.
