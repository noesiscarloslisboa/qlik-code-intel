# Reference sample assessment — 2026-10-08

All ten QVS files in the requested [reference validation directory](https://github.com/muehan/vs-qlik-languageserver/tree/681d077f85cf277dc578f4aa4ef34ae0048d6ff6/validation)
were scanned with the built executable at upstream revision
`681d077f85cf277dc578f4aa4ef34ae0048d6ff6`. Git blob hashes and byte lengths were
verified before testing. The original scripts and license were downloaded into a
temporary directory outside this MIT project. No upstream script source is included
in this report, the project fixtures, or the audit JSON.

These are editor/linter validation examples, not a known-valid application corpus.
One file scans without diagnostics; nine retain unsupported or malformed syntax.
Scanning all files without crashing is useful recovery evidence, not evidence of
complete syntax coverage or Qlik runtime validity.

## Results after statement-boundary hardening

| External file under `validation/` | LOADs | Definitions | References | Direct edges | Diagnostics |
| --- | ---: | ---: | ---: | ---: | ---: |
| `asAlingment.qvs` | 3 | 25 | 30 | 6 | 9 |
| `comment.qvs` | 3 | 9 | 12 | 6 | 7 |
| `function.qvs` | 0 | 0 | 0 | 0 | 2 |
| `keywords.qvs` | 2 | 4 | 4 | 2 | 5 |
| `linter.qvs` | 4 | 23 | 70 | 30 | 6 |
| `linter_as_withtab.qvs` | 3 | 15 | 12 | 3 | 0 |
| `small.qvs` | 6 | 10 | 16 | 9 | 8 |
| `strings.qvs` | 1 | 3 | 5 | 1 | 1 |
| `test.qvs` | 6 | 16 | 68 | 37 | 17 |
| `trace.qvs` | 0 | 0 | 0 | 0 | 23 |
| **Total** | **28** | **105** | **217** | **94** | **78** |

Counts describe recovered syntax occurrences, including repeated definitions. They
do not assert that every fact around malformed syntax is complete or semantically
valid. Diagnostic codes and per-command byte counts are in
[reference-results.json](reference-results.json).

## Defects found and repaired

Balanced quoted literals outside a statement could cause parser recovery to obscure
later valid constructs. A minimal independently authored fixture reproduced lost
include/table/field/source facts before the fix. The grammar now bounds those
fragments with an explicit unsupported node. The index diagnoses them and extracts
no symbols from them. They also break preceding LOAD adjacency.

In the external [test.qvs sample](https://github.com/muehan/vs-qlik-languageserver/blob/681d077f85cf277dc578f4aa4ef34ae0048d6ff6/validation/test.qvs),
the earlier balanced-literal fix increased recovered LOAD statements from one to
five, with includes on lines 18–20
and table definitions later in the file visible again. Across the corpus, LOADs
increased from 20 to 24, definitions from 75 to 85, references from 133 to 170, and
direct edges from 53 to 76. Diagnostics increased from 59 to 71 because later
unsupported constructs are now visible too; no diagnostics were suppressed.

That 24-LOAD result is the previous audit baseline. Statement-boundary hardening
now recovers a further four LOADs across `asAlingment.qvs`, `comment.qvs`,
`linter.qvs`, and `test.qvs`. The latter retains its table/LOAD range at line 24
after an unfinished unsupported clause. Compared with the prior baseline,
definitions increase from 85 to 105, references from 170 to 217, direct edges from
76 to 94, and diagnostics from 71 to 78. These counts measure recovered syntax,
including facts around malformed input; more occurrences do not establish complete
semantic accuracy.

The independently authored recovery scanner protects quotes, brackets, comments,
and nested dollar expansions. It recovers missing terminators only before
unquoted line-start table labels, preserving source positions and adding an
`unterminated-statement` diagnostic. Unsupported bodies are opaque tokens, so
neither the CLI nor the definition/reference queries extract names from them.
Unlabeled and same-line LOAD text remains opaque. Backtick names, doubled double-quote escapes,
and raw-versus-normalized captures are verified with original project fixtures;
they were not inferred by copying external samples. Tests also keep identifiers
beginning with REM distinct from the comment keyword, following
[Qlik's REM syntax](https://help.qlik.com/en-us/cloud-services/Subsystems/Hub/Content/Sense_Hub/Scripting/ScriptRegularStatements/Rem.htm).

The fix preserves valid quoted table labels and literal expressions. It does not
make standalone expressions executable Qlik statements. The original regression
checks source locations and ensures literal contents do not produce phantom index
references. Quoting distinctions follow [Qlik's quotation documentation](https://help.qlik.com/en-us/cloud-services/Subsystems/Hub/Content/Sense_Hub/Scripting/use-quotes-in-script.htm).

The recovered body also exposed a lexical-priority defect in existing unary NOT
support. `NOT IsNull(...)` could be parsed as an identifier followed by malformed
syntax. NOT now receives operator priority in LOAD and LET expressions, in every
keyword case, while longer identifiers such as `Notable` remain intact. Original
regressions failed before this fix and now preserve the intended field/variable
references without indexing NOT as a symbol. The diagnostic on line 62 of the
external `test.qvs` is gone. Qlik uses this clause in its
[documented generated LOAD scripts](https://help.qlik.com/en-US/connectors/Subsystems/Salesforce_Connector_help/Content/Connectors_Salesforce/Load-salesforce-data.htm).

## Retrieval checks

- Exact upstream content, all ten repository-relative paths, line/byte columns,
  and statement byte ranges were checked.
- Three repeated table definitions and three tabbed aliases in
  `linter_as_withtab.qvs` retain separate source locations.
- The preceding LOAD in `keywords.qvs` retains its direct anonymous input endpoint.
- An include and a later table in `test.qvs` remain retrievable after balanced
  literal fragments.
- The labeled LOAD at line 24 of `test.qvs` remains indexed after an unfinished
  unsupported clause; both its table occurrence and LOAD range are checked.
- Whole-repository maps, focused field maps, and source context were each checked
  at 256, 512, 1024, and 2048 conservative byte-token budgets. Every output stayed
  within its budget and repeated identically. Context lines were compared with
  the original external source, including CRLF normalization for display.

These checks validate selected facts and output invariants. They are not an
exhaustive precision/recall benchmark or a comparison against Qlik execution.

## Remaining limits exposed by the samples

- TRACE, DROP, CALL, subroutines, and control flow remain outside the implemented
  semantics. Supported body statements are source facts, not claims of execution.
- Missing semicolons/commas, stray clauses, and inconsistent quotation characters
  remain diagnostics. Standalone expression fragments are not application scripts.
- SQL-style null-test syntax and single-quoted field aliases in these examples
  are not recognized by this grammar. Their presence in an editor fixture does
  not establish runtime validity; no syntax was added merely to silence them.
- Unlabeled and same-line LOAD text inside an unfinished unknown statement remains
  opaque. Unterminated quotes/comments/expansions can obscure later input. A clear
  label is a recovery heuristic, not proof that malformed surrounding code executes.
  Recovery around malformed samples is deliberately reported as incomplete.

The user subsequently supplied three QlikView apps with confirmed prior successful
reloads. Their separate [assessment](qlikview-sample-assessment.md) uses heuristically
recovered script candidates; authoritative exports are still needed to confirm
completeness. The pinned reference audit was rerun after those repairs with the
same 28 LOADs, 105 definitions, 217 references, 94 edges, and 78 diagnostics.
The independent [retrieval checks](retrieval-acceptance.md) do not claim Qlik
runtime validity. Advanced lineage is still outside scope.

## Repeat the audit

Separately obtain the upstream repository at the pinned revision, then build this
project and run:

```sh
go build -o bin/qlik-repomap ./cmd/qlik-repomap
python3 scripts/validate-reference.py \
  --samples-dir /path/to/vs-qlik-languageserver \
  --output /tmp/qlik-reference-results.json
```

The [manifest](reference-samples.json) contains hashes/paths only. The runner uses
Python's standard library and the built Go CLI. It performs no network requests
and never executes scripts or reads QVD data. A wrong revision fails hash checks.
Normal project tests run independently authored fixtures without this external
checkout. The audit JSON excludes external source text.
