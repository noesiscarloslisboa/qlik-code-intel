# Local QlikView sample assessment — 2026-10-08

The user supplied three binary QlikView applications and confirmed that they have
reloaded successfully. A read-only local examination recovered UTF-8 script
candidates from their first compressed header streams. The candidates contain
10,742 lines and 69 script-tab markers across 340,147 bytes.

This is heuristic recovery, not a QVW reader or a complete script export. No
authoritative export has been compared, and the recovery may omit other serialized
fragments. Historical successful reloads apply to the apps, not to the completeness
of these candidates. No app, script, include, SQL connection, or data source was
executed; QVD contents were not read. Hash checks confirm that the original apps
are unchanged.

The apps, candidate source, provenance manifest, detailed scan results, and local
audit runner remain Git-ignored under `testdata/real_samples/`. This public report
contains aggregate results only. Committed regression fixtures were independently
authored and contain no copied application source.

## Repaired defects

- Composed variable, table, and field names retain their whole symbolic spelling,
  including nested expansions. Assignment-target expansions contribute separate
  variable references. Definition/reference queries capture the whole names.
  Adjacent expansions produce no empty literal fragments, and numeric fragments
  between expansions retain their spelling.
- The recovery scanner recognizes composed table labels after opaque unsupported
  bodies without extracting facts from those bodies.
- Literal dollars in file-format options, including Excel sheet suffixes, no
  longer create malformed variable expansions. Genuine expansions remain nodes.
- Basename retrieval recognizes Windows backslashes on every host. Drive, UNC,
  relative, wildcard, and library paths remain literal; endpoint identities and
  source text are unchanged.
- Empty LET assignments preserve their targets without inventing a value or
  references. Omitted function arguments no longer create missing-token nodes;
  present expressions retain their LOAD field or LET variable roles.
- HIERARCHY prefixes recover their LOADs and distinguish explicit input fields
  from named parent/path/depth outputs. No numbered level fields or transformed
  records are inferred. BUNDLE with optional INFO retains ordinary LOAD facts.
- GROUP BY expressions retain structured component references. These candidate
  forms are accepted for retrieval; grouping semantics are not validated.

Original parser/query and retrieval regressions failed before these repairs and
pass afterward. The built-executable checks exercise scan, map, find, deps, and
context against the original `testdata/qlikview/filesystem.qvs` and `coverage.qvs`
fixtures.

## Local scan results

| Candidate | LOADs | Definitions | References | Direct edges | Diagnostics |
| --- | ---: | ---: | ---: | ---: | ---: |
| A | 282 | 2,097 | 3,348 | 1,220 | 363 |
| B | 220 | 2,196 | 2,848 | 1,373 | 653 |
| C | 67 | 576 | 650 | 131 | 52 |
| **Total** | **569** | **4,869** | **6,846** | **2,724** | **1,068** |

Before the initial sample pass, the same candidate bytes yielded 555 LOADs, 4,802
definitions, 5,656 references, 1,620 direct edges, and 2,147 diagnostics. That pass
reached 567 LOADs with six syntax-error and six missing-token diagnostics. This
follow-up recovers two HIERARCHY LOADs and resolves all twelve remaining parser
diagnostics. More recovered occurrences do not prove complete accuracy or recall.

| Remaining diagnostic code | Count |
| --- | ---: |
| `unsupported` | 617 |
| `implicit-target` | 183 |
| `unresolved-source` | 137 |
| `unsupported-control` | 131 |
| `syntax-error` | 0 |
| `missing-token` | 0 |

SQL, control semantics, implicit targets, and source-less LOADs followed by SQL
retain diagnostics. Parsing the newly recognized source forms resolves their
parser diagnostics; the remaining diagnostic categories are unchanged.
No advanced lineage was added.

HIERARCHY parameter roles and BUNDLE/INFO syntax follow Qlik's
[HIERARCHY](https://help.qlik.com/en-US/qlikview/May2024/Subsystems/Client/Content/QV_QlikView/Scripting/ScriptPrefixes/Hierarchy.htm)
and [BUNDLE](https://help.qlik.com/en-US/qlikview/May2024/Subsystems/Client/Content/QV_QlikView/Scripting/ScriptPrefixes/Bundle.htm)
documentation. The optional IF else parameter is documented in
[IF](https://help.qlik.com/en-US/qlikview/May2024/Subsystems/Client/Content/QV_QlikView/Scripting/ConditionalFunctions/if.htm).
Empty LET values, explicit trailing empty arguments, and expression-bearing
GROUP BY clauses are observed in the private candidates; the documentation alone
does not establish all of their runtime semantics. The
[LOAD documentation](https://help.qlik.com/en-US/qlikview/May2024/Subsystems/Client/Content/QV_QlikView/Scripting/ScriptRegularStatements/Load.htm)
describes grouping by fields. Accepting broader source expressions does not prove
that every branch reloads or that recovered text is complete.

## Retrieval checks

The local audit verifies original app and candidate hashes, all reported path and
line/byte-column bounds, and statement byte ranges. Twenty-two selected table,
field, variable, and source occurrences are compared with manually reviewed source lines and
actual CLI lookup results. A Windows-basename dependency query retains the
expected consuming table and source location.

Twenty-four map/context checks cover whole-repository and focused queries at 256,
512, 1024, and 2048 conservative byte-token budgets. Every command is repeated to
check determinism; output remains valid UTF-8, within budget, and on whole lines.
Every numbered context line is compared with the original candidate source,
normalizing only CRLF for display. The entire recovered candidate set has no
syntax-error or missing-token diagnostics.

These are selected source-fact and retrieval checks, not exhaustive precision or
recall measurements. The separate pinned reference corpus retains its previous
aggregate counts and passes its twelve retrieval/budget checks.

A follow-up on 2026-10-09 added twenty source-reviewed retrieval questions and
forty checks, including map/context budgets of 256, 512, and 1024 bytes. The
baseline passed 38/40; fixes for producer direction ranking and symbolic source
filename discovery brought the result to 40/40. Extraction counts and diagnostic
categories above are unchanged. The [benchmark assessment](retrieval-benchmark-assessment.md)
contains aggregate results; the question manifest and detailed reports remain
private. The twenty-two source-location and twenty-four earlier budget checks
also pass after these fixes.

Final `make check` passes reproducible generation, all 20 grammar cases,
formatting, race-enabled Go tests, built-executable integration tests, vet, and
build. The handwritten C scanner also passes Clang's `-Wall -Wextra -Werror`
syntax check. The original five-file repository baseline remains unchanged and
both original QlikView fixtures scan with no diagnostics under `--strict`.

## Pending completeness check

Export each application's entire script using QlikView **Edit Script (Ctrl+E) →
File → Export to Script File…**, as described in
[Qlik's script editor documentation](https://help.qlik.com/en-US/qlikview/May2024/Subsystems/Client/Content/QV_QlikView/Edit_Script_dialog_box.htm).
Save those `.qvs` files separately from the recovered candidates, then compare
tab coverage, source bytes, and scan results. A project `LoadScript.txt` created
by saving the app with its corresponding `-prj` folder is another documented
source; see [QlikView project files](https://help.qlik.com/en-US/qlikview/May2024/Subsystems/Client/Content/QV_QlikView/QlikView_Project_Files.htm).

The CLI continues to accept `.qvs` text only. Complete application-script
acceptance remains pending this export comparison; historical reloads and parser
tests alone cannot establish recovered-script completeness.
