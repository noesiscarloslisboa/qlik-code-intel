# Retrieval acceptance — independently authored fixtures

`testdata/hardening/` contains three original QVS files with 11 definitions,
10 references, six direct edges, and three LOAD ranges. The recovery file
intentionally contains an unfinished unsupported statement and produces
`unsupported` plus `unterminated-statement`. Its quoted text must not produce
symbols, dependencies, or source context for its imaginary LOAD.

These are parser/retrieval acceptance checks, not scripts verified by a Qlik
reload. The user has since supplied three QlikView apps with confirmed prior
successful reloads. Their [local assessment](qlikview-sample-assessment.md) checks
heuristically recovered text; comparison with authoritative exports remains
pending. Do not treat the malformed external editor corpus as a substitute.

## Questions checked

| Question | Expected first result |
| --- | --- |
| Where is the backtick-quoted `Quarter Sales` table defined? | `10_sales.qvs:1` |
| Where is the `Net" Amount` output field, spelled with doubled double quotes in source, defined? | `10_sales.qvs:2` |
| Where is the backtick-quoted `Raw Amount` input referenced? | `10_sales.qvs:2`; no output definition |
| Where is `vRoot` defined? | `00_config.qvs:2`; its path expansion remains symbolic |
| Where is `sales.qvd` read? | `10_sales.qvs:3`, source `$(vRoot)/sales.qvd` |
| Where is `revenue.qvd` written? | `10_sales.qvs:6`, source `lib://Exports/revenue.qvd` |
| Which table feeds `Summary`? | A direct `resident` edge to `Quarter Sales` at `10_sales.qvs:5` |
| Which table produces `revenue.qvd`? | A direct `store` edge to `Summary` |
| Does the unfinished unknown statement hide the next table? | No; the backtick-quoted `Recovered Table` definition is at `20_recovery.qvs:3` |
| Can its field be retrieved as original source? | `Recovered ID` at `20_recovery.qvs:4`, with the original backtick delimiters on the table label |
| Do unsupported contents become facts? | No matches for `vGhost`, `Ghost`, `Fake`, or `fake.qvd` |

The tests exercise normalized symbol names and raw query captures separately.
They also check double-quoted names containing literal backticks. Doubled backticks
are not assumed to be valid escapes; Qlik's documented escape list excludes them.
Focused maps and context use budgets of 0, 32, 128, 256, 512, and 2048 conservative
UTF-8 byte tokens. Every output remains valid UTF-8, deterministic, and within
budget; the requested text remains visible at budgets of 256 and above. Recovered
source context excludes the unsupported statement and preserves numbered source.

## Automated checks

- `internal/qlik/hardening_test.go`: CRLF/indentation, literal/symbolic/Unicode
  labels, protected comments/quotes/brackets/macros, deep nesting, EOF, conservative
  unlabeled handling, preceding LOAD boundaries, escaped names, executed query
  captures, and incremental insertion/removal of a terminator.
- `tree-sitter-qlik/test/corpus/hardening.txt`: reviewed syntax trees for recovery,
  opaque bodies, backticks, and REM keyword boundaries.
- `internal/repomap/hardening_test.go`: the questions and budget/source invariants.
- `internal/qlik/qlikview_test.go` and `test/corpus/qlikview.txt` in the grammar:
  original composite variable/table/field names, nested expansions, opaque-body
  recovery before composite labels, exact query captures, Windows paths, and
  literal dollars in Excel format options.
- `internal/repomap/qlikview_test.go`: Windows and library source basename ranking,
  direct dependency selection, and focused maps under budget.
- `internal/repomap/retrieval_test.go`: independent direction ranking for QVD
  producers/consumers, literal filename discovery after symbolic path prefixes,
  exclusion of partial backup filenames, and unchanged composed-variable ranking.
- `internal/qlik/coverage_test.go` and the grammar's `test/corpus/coverage.txt`:
  empty LET assignments, omitted call arguments, component GROUP BY references,
  HIERARCHY input/output roles and optional positions, BUNDLE sources, explicit
  INLINE expansions, malformed-expression diagnostics, prefix boundaries, and
  executed definition/reference captures.
- `internal/cli/integration_test.go`: actual executable scan/find/map/deps/context
  checks, diagnostic separation, valid JSON, and strict-scan exit status, including
  the independently authored `testdata/qlikview/filesystem.qvs` and `coverage.qvs`
  fixtures. Hierarchy output lookup retains its prefix line and exact original
  source context under a 256-byte-token budget.
  The original `testdata/retrieval/` fixtures also exercise a late field alias
  under a small budget and both repaired QVD dependency lookups.

Run `make check` from the project root. The optional pinned external audit is
documented in [reference-sample-assessment.md](reference-sample-assessment.md).
Run `make benchmark` for twenty source-reviewed public questions and forty checks.
The [benchmark assessment](retrieval-benchmark-assessment.md) records their
results and aggregate outcomes for a separate set of private questions.
