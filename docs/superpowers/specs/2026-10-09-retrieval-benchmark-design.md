# Retrieval benchmark design

The user approved a small benchmark of realistic questions against the local
sample scripts, followed by fixes for ranking and source selection failures.
Use manually reviewed source answers, without executing Qlik or inferring lineage.
Authoritative script export comparison remains a separate pending check.

Use a manifest-driven, optional Python standard-library runner against the actual
CLI. Pin source bytes with SHA-256 hashes and keep expected locations independent
of the generated index. A private manifest contains twenty questions spanning
tables, variables, fields, QVD producers/consumers, and direct RESIDENT/FROM edges.
An independently authored public fixture and manifest provide reproducible checks.
An alternative index-generated oracle would hide extraction errors. A statistical
or model-based judge would add dependencies and make results less repeatable.

Find questions check the first JSON occurrence against an explicit relevant set.
Maps check the first record's statement path/line and required answer text.
Context checks the first contiguous source block for an expected occurrence and
compares every numbered line with original source. Dependency questions check
manually specified typed edges and locations, not an assumed relevance ordering.
Map/context run at 256, 512, and 1024 conservative byte-token budgets. Source answer
lines must fit those budgets; an impossible whole-line answer is not counted as a
ranking failure or silently excluded. Reject infeasible manifests explicitly.

Repeat each invocation to check determinism. Keep UTF-8, complete records, byte
bounds, and original-line invariants. Capture diagnostics separately and do not
mistake unsupported-feature diagnostics for a runtime failure. Record aggregate
metrics and question IDs without source snippets or query text. A nonzero result
signals a failed expectation or invariant. Keep the private manifest and detailed
outputs under ignored real_samples; publish only aggregate private results.

Repair only demonstrated retrieval defects, with original failing Go regressions
and executable checks. Preserve exact-name priority, repeated definitions, direct
dependency scope, source bytes, and budgets. No natural-language query engine,
grammar expansion, new lineage, persistent index, or tokenizer is introduced.
Keep Python optional; normal Go tests continue without private samples or Python.
