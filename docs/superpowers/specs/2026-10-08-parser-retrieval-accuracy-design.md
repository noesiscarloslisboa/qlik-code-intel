# Parser and retrieval accuracy increment

The user approved continuing after the first milestone. Keep the existing Go module,
CLI commands, conservative budgets, and in-memory index. Do not add evaluated
lineage, MCP, a GUI, persistence, or dependencies.

## Problems and resulting behavior

1. LET expressions currently classify bare variable identifiers as fields, then
   omit them from the index. Introduce a `bare_variable_reference` syntax node in
   LET expression context. Function identifiers and string contents remain distinct.
   Add variable-use symbols/edges and correct the definition/reference queries.
2. Recognize interpretation function identifiers ending in `#` and pure symbolic
   table names/field aliases such as `$(vTable)` and `AS $(vAlias)`. Keep macro names
   and values unevaluated. Bracketed composite names already work.
3. A source-less LOAD followed by an unlabeled, unprefixed LOAD is a preceding
   LOAD relationship. Record a direct consumer-to-input `preceding` edge between
   their syntactic owners. Each semicolon-terminated LOAD keeps its own source
   range and output fields. Comments do not break adjacency; another labeled or
   prefixed LOAD, assignment, include, unsupported statement/control flow, or parse
   error does. A source-less LOAD with no recognized adjacent input receives a
   diagnostic rather than an invented source. SQL source semantics remain unsupported.
4. Add `File.loads` records for recognized LOAD statement ranges and owner identities.
   This preserves wildcard-only stages without fabricating table/field definitions.
   Maps display direct preceding links; context can retrieve an anonymous neighbor
   stage by its recorded range. Do not copy intermediate fields into the outer
   table, expand wildcard fields, or walk dependencies recursively.
5. Focused map ranking must compare the strongest symbol match per statement,
   then structural importance, rather than summing arbitrarily many weak matches.
   When a full summary does not fit, try a summary retaining the strongest matching
   field/reference/variable before dropping field details entirely. All variants
   remain complete records and count every UTF-8 byte against the budget.
6. A field lookup must not expand every include in its file merely because its
   occurrence shares that file. Include and file-path lookups can retrieve direct,
   literal indexed include targets. Preserve deterministic, duplicate-free context.

## Choices

Use a small LOAD-range record and direct syntactic edges. Folding an entire chain
into one table schema would silently promote intermediate fields; evaluated lineage
would exceed the user's scope. Retain the simple conservative budget estimator;
adding a model-specific tokenizer would add a dependency and a new configuration
surface without addressing the demonstrated ranking/selection defects.

## Acceptance

Add failing tests first for each defect. Test LET identifiers nested in calls and
binary expressions, escaped strings, interpretation functions, symbolic labels and
RESIDENT/STORE/JOIN names, and executed query captures. Test two/three-stage
preceding LOADs, wildcard-only stages, separate labeled tables, unsupported/control
boundaries, exact lines/byte ranges, and no promotion of intermediate fields. Test
focused compact maps, exact-versus-partial ranking, context include isolation, and
budget/determinism invariants. Run grammar regeneration/corpus checks, race-enabled
Go tests, actual executable integration tests, vet, formatting, and build.
