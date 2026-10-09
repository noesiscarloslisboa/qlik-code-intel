# Rename, drop, and trace source facts

Status: scope approved by the user's “go” on 2026-10-09 after the proposal to
add syntax support and explicit names without simulating execution. Execution
is inline in the existing checkout. No additional permission gate is needed.

Implemented and verified on the same date; see the
[assessment](../../validation/script-operations-assessment.md) for executed
checks, private aggregate results, and remaining limitations.

## Contract

Recognize case-insensitive RENAME FIELD(S)/TABLE(S), with explicit comma-separated
old TO new pairs or USING a mapping table. Recognize DROP FIELD(S), optionally
FROM a comma-separated table list, and DROP [MAPPING] TABLE(S). Recognize TRACE
text with explicit variable expansions. Preserve quoted, backtick, composed,
and symbolic names and original source locations.

Explicit old rename names and all drop targets are references; explicit new
rename names are source definitions. A field rename has no inferred table owner.
USING contributes only a mapping-table reference, never evaluated output names.
Drop statements do not remove previous definitions. Rename statements do not
rewrite previous names or copy fields between tables. No rename/drop dependency
edges, runtime validation, or automatic name resolution are introduced. Variable
expansions in these operations reference variables from the containing file.

TRACE text is unevaluated text, not a LOAD/LET expression. Ordinary words,
paths, function-like text, and quoted LOAD-like text create no field/table/source
symbols. Explicit expansions retain variable references. TRACE never emits log
messages or runs Qlik. Existing recovery must remain opaque for unknown statements.
All new statement kinds break preceding LOAD adjacency.

## Implementation

Add dedicated Tree-sitter nodes and context-specific quoted-name rules. Extend
definition/reference/tag queries with explicit roles. Keep the existing external
scanner unchanged unless a demonstrated recovery regression requires a fix.

Add a minimal `Statement` record (kind, location, byte range) to `File.Statements`
for recognized supported statements. This additive JSON field allows even a
TRACE with no symbols to retain a source range and lets maps distinguish an
operation from a LOAD. The existing LOAD records and symbol/edge fields retain
their meaning. Malformed operations produce diagnostics and no operation facts;
surrounding supported statements remain retrievable when recovery permits.

Map operation summaries use explicit source symbol order and roles. Compact
rename summaries retain the focused old/new pair; compact drop summaries retain
the focused target. TRACE summaries expose variable uses, not arbitrary long
message bodies. Every output byte still counts toward the budget. Context keeps
original numbered source and its existing direct-neighbor policy.

Alternatives considered: suppressing diagnostics without indexing would hide
useful facts; modeling a live data model would require execution semantics and
is outside the approved scope. Explicit syntactic operations fit the current
index without adding dependencies or a database.

## Verification and privacy

Author public fixtures independently from Qlik documentation. Test singular,
plural, lists, quoting, symbolic names, mapping forms, malformed input, unknown
syntax opacity, source ranges, unchanged earlier definitions, query captures,
operation maps, focused compact summaries, and exact numbered context.

Regenerate the parser, review trees before corpus expectations, and run npm ci,
make generate, test-grammar, check-generated, test, integration, vet, build,
format-check, and benchmark. Re-run the existing private source-pinned Cloud
retrieval oracle and audit aggregate diagnostic/symbol changes. Keep all private
names, source, hashes, tenant identifiers, and detailed reports ignored.

Release packaging is a subsequent step after this increment passes. No published
tag is moved and no release is created by this parser increment.

## Language references

- [Rename field](https://help.qlik.com/en-US/cloud-services/Subsystems/Hub/Content/Sense_Hub/Scripting/ScriptRegularStatements/rename-field.htm)
- [Rename table](https://help.qlik.com/en-US/cloud-services/Subsystems/Hub/Content/Sense_Hub/Scripting/ScriptRegularStatements/rename-table.htm)
- [Drop field](https://help.qlik.com/en-US/cloud-services/Subsystems/Hub/Content/Sense_Hub/Scripting/ScriptRegularStatements/drop-field.htm)
- [Drop table](https://help.qlik.com/en-US/cloud-services/Subsystems/Hub/Content/Sense_Hub/Scripting/ScriptRegularStatements/drop-table.htm)
- [Trace](https://help.qlik.com/en-US/cloud-services/Subsystems/Hub/Content/Sense_Hub/Scripting/ScriptRegularStatements/Trace.htm)
