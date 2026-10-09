# Script operations assessment

Verified on 2026-10-09 with Go 1.27.2 on macOS arm64 and pinned Tree-sitter CLI
0.25.10. This increment adds source-backed rename/drop facts and TRACE expansion
references. It introduces no runtime dependencies or script execution.

## Executed verification

- `npm ci` installed the pinned grammar dependency. The first sandboxed attempt
  failed DNS access; the permitted retry succeeded. Tree-sitter's local parser
  cache also required execution permission. No TLS bypass was used.
- New extraction/query tests first failed on missing operation facts. After
  implementation, original name/role/range, quoting, mapping, comments,
  malformed-input, and preceding-boundary cases passed.
- Syntax trees were inspected before recording the original operation corpus.
  `make test-grammar` passed all 22 corpus cases and `make check-generated`
  confirmed reproducible generated artifacts. The scanner was unchanged.
- `make check benchmark` passed. A subsequent context regression demonstrated
  that a file-owned variable edge could expand unrelated file source. The fix
  passed that regression; `make test integration vet build format-check benchmark`
  then passed again, including race tests and five new executable operation checks.
- Final query review reproduced field captures from TRACE macro arguments. A
  text-only argument grammar fixed the regression, retaining nested expansions
  inside quotes. The final `make check benchmark` passed after this change.
- The public retrieval benchmark passed all 40 selected checks across 20
  questions. Operation retrieval tests additionally checked focused pairs/drop
  targets, stable compact maps, exact source context, and feasible byte budgets.
- `python3 scripts/check-skill.py --binary bin/qlik-repomap` passed all eight
  documented commands, strict partial indexing, source, role, direction, and
  budget checks.
- `python3 -m unittest discover -s scripts -p 'test_*.py'`: all 12 checks passed.

## Private comparison

The unchanged, hash-verified Cloud snapshot retained every original symbol,
edge, and LOAD range. All 60 targeted operations were recognized; diagnostics
fell from 72 to 12. The original 24 private checks passed, and the expanded
source-reviewed oracle passed 39 checks across 21 questions, plus three explicit
source checks for drop/TRACE facts. See the aggregate
[Cloud assessment](cloud-script-assessment.md). Private source, identifiers,
hashes, queries, and detailed reports remain ignored.

## Limits

Rename targets are explicit source definitions, not evaluated table/field
existence. Mapping-table contents are not read, field ownership is not inferred,
and earlier facts are neither renamed nor deleted. No rename/drop lineage edges
are fabricated. TRACE never executes or prints its message. Malformed operations
produce diagnostics; broader Qlik syntax and runtime reload validity remain
outside this validation. These selected cases do not establish exhaustive recall.
