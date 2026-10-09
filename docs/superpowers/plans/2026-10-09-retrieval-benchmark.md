# Retrieval benchmark implementation plan

**Goal:** Measure the first useful answer and budgeted source retention on fixed,
source-reviewed questions, and repair demonstrated retrieval defects.

**Architecture:** An optional stdlib Python runner invokes the built CLI using
pinned manifests. Private data stays ignored; original Go regressions and public
fixtures guard fixes in normal tests. Execute inline in the authorized scope.

- [x] Inspect ranking, direct dependencies, source selection, prior audit oracles,
  and the relevant original source lines.
- [x] Create `scripts/benchmark-retrieval.py` for pinned find/map/context/deps
  answers, first-result checks, repeated invocations, and aggregate results.
- [x] Curate twenty private questions and record a baseline at 256/512/1024 byte
  budgets before modifying retrieval behavior.
- [x] Reproduce any failures in independently authored public fixtures and
  `internal/repomap/retrieval_test.go`, then implement focused fixes.
- [x] Add public manifest/executable benchmark checks and document repeat commands.
- [x] Run focused regressions, the public/private benchmarks, and `make check`.
- [x] Rerun private source audit and the pinned reference audit, documenting any
  intentional retrieval output changes and unchanged extraction counts.
- [x] Record aggregate benchmark results, limitations, and the milestone-one
  acceptance status without private source or query text.

Use the existing temporary Go module/build caches. Source hashes, exact numbered
lines, typed edges, and expected locations must be reviewed independently of CLI
output. Retain all diagnostic visibility and source identity rules in AGENTS.md.

## Verified outcome

- Private baseline: 38/40 checks. Independent direction ranking and symbolic
  source filename discovery repair both failed QVD dependency lookups.
- Private final and separate public suite: 40/40 each. `make benchmark` also
  passes five runner self-tests. Negative probes reject a wrong first answer,
  changed sample bytes, an empty benchmark, and an infeasible context line.
- `make check`: reproducible parser, 20/20 corpus cases, formatting, race-enabled
  Go tests, actual CLI integration, vet, and build pass.
- Private audit: 22 source occurrence oracles and 24 earlier budget checks pass;
  original apps are unchanged. Extraction counts remain unchanged.
- Pinned reference audit: 12 retrieval checks pass; the complete aggregate report
  matches `docs/validation/reference-results.json` exactly.
- Public aggregate assessment: `docs/validation/retrieval-benchmark-assessment.md`.
  Authoritative export comparison remains pending; no claim of complete script
  recovery or runtime validity is added.
