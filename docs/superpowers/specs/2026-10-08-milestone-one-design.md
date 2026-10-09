# qlik-code-intelligence: milestone one

## Goal and acceptance

Deliver an independently written, MIT-licensed Tree-sitter grammar and Go CLI that scan `.qvs` repositories and produce useful, bounded maps. Include `scan`, `map`, `find`, `deps`, and `context`; dependencies describe direct syntactic relationships, not evaluated Qlik lineage. No MCP, GUI, database, Qlik runtime, or network requests during CLI use.

Acceptance: a fresh checkout builds with Go and a C compiler, parses representative fixtures, produces table/variable/field/source symbols with relative paths and one-based lines, handles unsupported statements without crashing or losing following supported statements, and produces deterministic maps within the documented budget estimate. All grammar, extraction, CLI, and budget tests must pass.

## Architecture

One Go module named `qlik-code-intelligence` until a public repository URL is chosen. `tree-sitter-qlik/` contains the grammar, generated C source, Go binding, corpus tests, and queries. `cmd/qlik-repomap/` is a thin entry point. `internal/qlik/` parses source into an in-memory index. `internal/repomap/` ranks and renders maps, searches symbols, walks direct edges, and selects source context. `internal/cli/` owns flags, diagnostics, and output.

The grammar identifies SET/LET assignments, dollar expansions, table labels, LOAD field expressions/aliases, FROM/RESIDENT sources, STORE outputs, include directives, JOIN and CONCATENATE prefixes. It supports case-insensitive keywords, quoted/bracketed names, common expression operators/functions, comments, and format specifications. Explicit unsupported nodes and Tree-sitter recovery preserve useful surrounding syntax.

Definitions and references use Aider's `@name.definition.*` and `@name.reference.*` captures paired with `@definition.*`/`@reference.*` captures. These are conventions for a custom language integration; stock Aider does not automatically acquire Qlik support.

## Index and retrieval

Each symbol records kind, role, name, owner, path, line/column, and statement line/byte range. Edges point from consumers to upstream tables, data sources, variables, or literal include targets, with kind and source location. STORE edges point from the output source to the stored table. No execution, automatic include expansion, relative include path resolution, or variable evaluation is implied. Repeated definitions remain distinct by file/position. Unlabeled LOADs have `@path:line:column` identities rather than invented runtime table names.

Scan files in sorted order, skip common dependency/build/version-control directories, and never follow symlinks. Parse errors become diagnostics, while read/walk errors fail visibly. Commands scan fresh source instead of persisting stale indexes. `scan --json` exports the index for other tools.

Maps prioritize table definitions, direct relationships, and matching symbols; render compact field lists and low-cost locations. Search ranks exact names above substring matches. Context selects defining/reference statements and direct neighbors, merges overlapping windows, and prints numbered source. Budgets count UTF-8 bytes conservatively (one byte per estimated token); this bounds output without a model-specific tokenizer or dependency. Tiny budgets may yield empty output. Every emitted byte, including headers and numbering, counts.

## Alternatives and decisions

SQLite would add distribution and schema work without improving the initial scan-and-map loop. A persisted JSON cache would require invalidation before it adds value. Use an in-memory index and optional JSON export. Use standard-library flags and explicit command dispatch instead of a CLI framework to minimize dependencies. Keep grammar and CLI in a single module to test binding compatibility together.

## Validation and licensing

Use original fixtures covering every requested construct, joins/concatenations, alias/reference distinctions, malformed input, unsupported statements, mixed-case keywords, strings containing semicolons, and duplicate names across files. Test queries by executing them against parsed fixtures. Test budgets from tiny to large, determinism, ranking, direct edge direction, cancellation, paths/lines, file selection, and built executable behavior.

The GPL language-server repository is a language reference only. Do not copy or translate its implementation, grammar, queries, or fixtures. Record public references in README. Commit generated parser artifacts and pin generator/runtime versions. Include CI, Makefile, contribution guidance, and AGENTS.md.
