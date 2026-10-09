# Working on qlik-code-intelligence

## Goal and scope

Keep the first milestone useful end to end: parse `.qvs` files with Tree-sitter,
index source facts, and retrieve compact maps and source context. The project is
MIT licensed. Do not add advanced lineage, MCP, a GUI, script execution, or a
database without a new requirement. Prefer Go's standard library and focused
packages. The generated parser and the Go binding are part of the deliverable.
The public module path is `github.com/noesiscarloslisboa/qlik-code-intel`.

## Structure

- `tree-sitter-qlik/grammar.js`: the language source of truth.
- `tree-sitter-qlik/src/`: generated artifacts; regenerate, never hand-edit
  `parser.c`, `grammar.json`, or `node-types.json`. `src/scanner.c` is handwritten
  recovery and name-prefix logic and must be included alongside the parser in every binding.
- `tree-sitter-qlik/queries/`: actual Tree-sitter queries, tested in Go.
- `internal/qlik/`: AST extraction and deterministic filesystem scanning.
- `internal/repomap/`: search, ranking, budgets, direct edges, source selection.
- `internal/cli/`: command dispatch/flags and injected stdout/stderr.
- `cmd/qlik-repomap/`: thin entry point with cancellation.
- `testdata/`: independently authored QVS examples; `real_samples/` is private and ignored.

## Accuracy rules

- Extract symbols from the syntax tree, not regular-expression scans over source.
- Keep input field references separate from output aliases. Functions are not fields.
- HIERARCHY inputs (`node_id`, `parent_id`, `node_name`, `path_source`) are field
  references. Explicit `parent_name`, `path_name`, and `depth` names are field
  definitions owned by the LOAD. Never fabricate numbered levels or relationships
  between hierarchy fields. Delimiter literals are not field references.
- Empty assignments and omitted call arguments create no fabricated values or
  symbols. Retain reference roles for present LOAD/LET expressions. GROUP BY
  expressions contribute component references, without validating grouping semantics.
- BUNDLE/INFO retains LOAD facts without opening bundled files. INLINE headers and
  values create no field/source symbols; explicit variable expansions remain references.
- Bare identifiers in LET expressions are variable references; in LOAD expressions
  they are field references. Keep this distinction in syntax nodes and queries.
- Preserve paths, one-based lines/byte columns, and zero-based statement byte ranges.
- Reject invalid UTF-8 source before extraction. Do not silently replace bytes or
  guess legacy encodings; original source spelling and byte locations must agree.
- Keep symbolic variables and data paths intact; do not infer evaluated values.
- Preserve whole composed names and nested expansion nodes. Expansions in an
  assignment target contribute references separately from its symbolic definition.
- Keep drive, UNC, relative, wildcard, and library paths literal. Recognize both
  slash forms only for basename ranking; do not normalize endpoint identities.
- Source/include discovery may rank a literal filename after leading balanced
  expansions as a basename match. Do not apply this rule to variables, tables,
  or fields, evaluate expansions, or equate the resulting source identities.
- A literal dollar in a format option is not a variable expansion. External name
  prefix tokens distinguish `$(` from ordinary dollar-containing identifiers;
  immediate continuation tokens must not consume intervening whitespace.
- Dependencies point consumer to upstream input; STORE points output source to table.
- Rank matching dependency endpoints independently for upstream and downstream
  directions. A stronger match in the opposite direction must not hide an edge.
- Do not infer implicit JOIN/CONCATENATE targets or Qlik execution order.
- Preceding LOAD edges describe adjacent syntax only. Keep each stage's fields,
  owner, and source range separate; never promote intermediate fields or expand
  wildcards. Preserve wildcard-only stages in `File.Loads` without fabricated symbols.
- Comments may separate preceding stages; labels, prefixes, other statements,
  parse errors, and unsupported/control syntax break adjacency. Diagnose unresolved
  source-less LOADs. JOIN cannot consume a preceding LOAD stage.
- Preserve repeated definitions and their locations. Case-insensitive search does
  not imply case-insensitive endpoint identity.
- Unsupported syntax must produce diagnostics. Keep supported surrounding
  statements retrievable and never claim complete Qlik syntax support.
- Balanced root-level literals are explicitly unsupported recovery boundaries;
  never extract index references from them. Preserve valid quoted table labels
  and literal LOAD expressions. Unterminated quotes may still obscure later input.
- Unknown bodies are opaque external tokens, including protected quotes, comments,
  brackets, and expansions. Recover missing semicolons only before unquoted
  line-start table labels; do not promote unlabeled or same-line LOAD text to facts.
  Keep `unsupported` and `unterminated-statement` diagnostics. The scanner is
  stateless, uses an iterative context stack, and opts out of generic error recovery.
- Backtick names retain raw delimiters in query captures. Remove enclosing
  backticks in the CLI, keeping dollar expansions symbolic. Do not invent doubled
  backtick escaping: Qlik lists only double quotes, brackets, and single quotes
  as escape delimiters. Use alternative quotes for a name containing a backtick.
- Include paths stay literal; the Qlik app working directory is unknown. Never
  execute includes or read QVD contents. Do not follow repository symlinks.
- Budget paths, headers, numbering, and newlines too. Current budget units are
  conservative UTF-8 byte counts. Never truncate source lines/UTF-8 characters.
- Stable results and ranking are public behavior. Avoid map iteration in output.
- Rank a focused statement by its strongest match before structural importance.
  Preserve the matching symbol in compact summaries when possible. Context may
  retrieve direct neighbors only; symbol queries must not expand unrelated includes.

## Licensing

The GPL-3.0 `muehan/vs-qlik-languageserver` repository is a language reference
only. Do not copy, translate, or adapt its code, grammar, queries, or fixtures.
Write tests/grammar independently using public language documentation. Retain
Tree-sitter support-header licensing in `THIRD_PARTY_NOTICES.md`.

External upstream samples may be tested separately in a temporary directory at
the user's request. Keep their GPL source out of this project. The opt-in runner
and manifest in `docs/validation/` record hashes and aggregate source facts only;
do not put upstream source snippets in reports or adapt their fixtures into tests.

Keep user-supplied apps, recovered scripts, manifests, and detailed audit output
under the ignored `testdata/real_samples/` folder. Publish only aggregate results
and independently authored regressions. Heuristic recovery does not prove that
the whole application script was recovered, even when prior reloads are confirmed.

## Verification

Before claiming completion, run the checks relevant to the change and fix failures:

```sh
make test           # race-enabled Go tests
make integration    # builds and invokes the actual executable
make vet
make build
```

After grammar/query changes:

```sh
cd tree-sitter-qlik && npm ci && cd ..
make generate       # includes go clean -cache for CGO's external C includes
make test-grammar
make check-generated
make test
make integration
```

Review syntax trees before updating corpus expectations. Add original fixtures
for new syntax, assert query captures when queries change, and add regression
tests for accuracy/recovery bugs. `make check` runs the complete milestone checks.
Use `gofmt`; keep diagnostics on stderr and machine-readable output on stdout.
Always close Tree-sitter parsers, trees, queries, and cursors.
For ranking/context changes, run `make benchmark` as well (optional Python 3.9+
stdlib runner). Keep manifests source-reviewed and SHA-256 pinned. Check the first
answer/block and original numbered context lines at every budget; dependency
oracles must retain endpoint kinds and explicit directions. Reject infeasible
expected lines instead of counting an empty response as a successful answer.
Keep private manifests/reports ignored; publish only aggregate outcomes and
original public regressions. Selected benchmark cases are not exhaustive recall.
After handwritten scanner changes, also run `go clean -cache`: CGO does not track
changes to C sources included from another directory. The user confirmed prior
successful reloads for three local QlikView apps. Their recovered script candidates
have selected source/retrieval checks, but completeness still requires comparison
with authoritative `.qvs` exports or project `LoadScript.txt` files. Independent
fixtures and editor examples do not establish runtime validity; this tool never
reloads apps.
