# Retrieval benchmark assessment — 2026-10-09

The first milestone now has a repeatable acceptance benchmark for symbol lookup,
focused repository maps, direct dependencies, and source context. Twenty questions
reviewed against the private recovered QlikView candidates produced forty checks.
The baseline passed 38/40. After two retrieval fixes, all forty pass.

| Command | Questions | Checks | Private baseline | Private final | Original public fixtures |
| --- | ---: | ---: | ---: | ---: | ---: |
| `find` | 6 | 6 | 6/6 | 6/6 | 6/6 |
| `map` | 5 | 15 | 15/15 | 15/15 | 15/15 |
| `context` | 5 | 15 | 15/15 | 15/15 | 15/15 |
| `deps` | 4 | 4 | 2/4 | 4/4 | 4/4 |
| **Total** | **20** | **40** | **38/40** | **40/40** | **40/40** |

The public suite uses twenty separate questions against independently authored
fixtures in `testdata/retrieval/`. Private app source, question text, filenames,
and detailed reports remain ignored under `testdata/real_samples/`. This report
contains only aggregate private results. Public examples below come from the
original fixtures.

## What the checks measure

Manifests pin source bytes with SHA-256 and specify expected locations reviewed
from source rather than generated from the index. Questions cover table
definitions, composed variable definitions/uses, output field aliases and
hierarchy tables, QVD producers/consumers, and direct RESIDENT inputs. They use
explicit symbol queries; natural-language descriptions do not add a CLI feature.

- `find`: the first result must match an expected name and source location, with
  the requested kind/role filters.
- `map`: the first record must identify the expected statement and retain the
  requested answer text, including field aliases in compact summaries.
- `context`: the first contiguous block must contain an expected occurrence.
  Every displayed line must match its original source, allowing only CRLF
  normalization. Duplicate, changed, or out-of-bounds lines fail the check.
- `deps`: expected direct edges must retain kinds, endpoint names, directions,
  paths, and lines. Public cases also reject unrelated backup-filename edges.
  These checks assert edge membership rather than relevance order.

Each map/context question runs at 256, 512, and 1024 conservative UTF-8 byte
budgets. Paths, headers, line numbers, and newlines count toward the budget.
Every command runs twice and must return identical stdout/stderr, valid UTF-8,
complete records, and a successful exit status. Expected context lines must fit
the smallest budget; the runner rejects infeasible manifests and changed samples.
Unsupported-syntax diagnostics remain visible and do not fail a normal scan.

## Defects repaired

1. Dependency search previously chose its strongest endpoint across both
   directions before filtering. An exact consumer such as `snapshot.qvd` could
   hide the basename-matching producer `lib://Exports/snapshot.qvd` from an
   upstream query. Ranking now chooses eligible endpoints independently for
   upstream and downstream; `both` retains each direction's best matches.
2. A source such as `$(vRoot)events.qvd` previously ranked below a literal source
   such as `data\events.qvd`, so the dependency query could omit its consumer.
   Source/include discovery now recognizes the literal filename after leading
   balanced expansions, including nested expansions. It retains the original
   spelling and typed identity, without evaluating variables or asserting that
   both paths identify the same file. Composed variables keep their prior ranking.

Original Go regressions reproduced both failures before the fixes. Executable
integration tests retain these cases and check a late field alias under a
256-byte budget. The optional runner has five self-tests that reject wrong first
answers, altered source, duplicate context lines, and incorrectly typed edges;
Unicode separators within source lines remain intact.

## Repeat and acceptance status

```sh
make check
make benchmark  # optional Python 3.9+ standard library runner
```

Both pass after the fixes. `make check` verifies reproducible generated artifacts,
all twenty grammar cases, formatting, race-enabled Go tests, executable CLI tests,
vet, and build. `make benchmark` verifies the original public manifest. The
optional runner can write aggregate JSON using `--output`; see the README for
custom-manifest commands.

The private source audit also passes its twenty-two occurrence oracles and
twenty-four earlier budget checks. Its extraction totals remain 569 LOADs,
4,869 definitions, 6,846 references, 2,724 direct edges, and 1,068 diagnostics,
including zero syntax-error and zero missing-token diagnostics. Original QVW
hashes remain unchanged. The pinned reference audit passes its twelve retrieval
checks, and its complete aggregate report is unchanged.

This completes the first milestone's implementation and selected acceptance
checks. Parser/retrieval scope is frozen at the documented supported constructs;
advanced lineage, MCP, and a GUI remain outside the milestone. Selected answers
do not establish exhaustive precision/recall, runtime validity, or support for
every Qlik construct. Private script completeness still requires comparison
with authoritative QlikView exports, as described in the
[sample assessment](qlikview-sample-assessment.md).
