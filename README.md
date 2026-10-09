# qlik-code-intelligence

Source-backed code intelligence for Qlik load scripts (`.qvs`), including shared Qlik Sense and QlikView syntax, built for AI coding assistants.

The first milestone provides two components:

- **tree-sitter-qlik**: an independently implemented Tree-sitter grammar, generated C parser, Go binding, and definition/reference queries.
- **qlik-repomap**: a Go CLI that scans QVS repositories and retrieves compact maps, symbols, direct dependencies, and numbered source context.

MIT licensed. Repository retrieval works offline without a Qlik installation.
An explicit `cloud pull` can import a saved Qlik Cloud app script over HTTPS.
No script execution, database, MCP server, or GUI. Repository commands scan
current files into memory; `scan --json` exports the source index.

## Download

Download `qlik-repomap` from [GitHub Releases](https://github.com/noesiscarloslisboa/qlik-code-intel/releases):

| Platform | Archive suffix |
| --- | --- |
| macOS Apple Silicon | `darwin_arm64.tar.gz` |
| macOS Intel | `darwin_amd64.tar.gz` |
| Windows x64 | `windows_amd64.zip` |

Extract the archive and run `./qlik-repomap version` on macOS or
`.\qlik-repomap.exe version` in PowerShell. Add its directory to `PATH` to run it
from elsewhere. The downloaded executables include the parser and need no Go,
C compiler, or Qlik installation. Archives include licenses and build metadata;
`SHA256SUMS` provides their checksums. Binaries are unsigned. See the
[release process](docs/releasing.md) for build and validation details.

## Build and try it

Requires **Go 1.23+**, a C compiler (GCC or Clang), and **CGO enabled**. On macOS install the Xcode command-line tools; on Linux install your distribution's C build tools. The generated parser is included, so Node.js and Tree-sitter CLI are unnecessary for ordinary builds.

```sh
git clone https://github.com/noesiscarloslisboa/qlik-code-intel.git
cd qlik-code-intel
go build -o bin/qlik-repomap ./cmd/qlik-repomap
./bin/qlik-repomap scan --root testdata/repository
./bin/qlik-repomap map --root testdata/repository --tokens 1500
```

The sample scan indexes five files with 21 definitions, 26 references, and nine direct dependency edges. A map contains records like:

```text
10_orders.qvs:1 table Orders | fields: OrderID, CustomerID, GrossAmount, OrderDate, Cutoff | <- from $(vData)/orders.qvd | uses: $(vCutoff), $(vData)
20_summary.qvs:1 table Daily Sales | fields: OrderDate, Revenue | <- resident Orders
10_orders.qvs:11 extend Orders | fields: CustomerID, CustomerName | <- from lib://Warehouse/customers.qvd
20_summary.qvs:8 store Daily Sales -> lib://Exports/daily_sales.qvd
00_config.qvs:2 SET vData = lib://Warehouse
```

Map records are ranked, so their order can differ from execution/source order. Locations always refer to the original source.

## Commands

Repository commands accept `--root DIR` (default `.`). Flags may appear before or after a query. Quote names containing spaces. `--help` documents each command.

| Command | Purpose | Useful flags |
| --- | --- | --- |
| `scan` | Index all selected QVS files | `--json`, `--strict` |
| `map` | Render ranked statement summaries | `--tokens N`, `--query TEXT` |
| `find QUERY` | Find symbol occurrences | `--kind KIND`, `--role ROLE`, `--limit N`, `--json` |
| `deps QUERY` | Show direct upstream/downstream edges | `--kind KIND`, `--direction upstream\|downstream\|both`, `--json` |
| `context QUERY` | Retrieve numbered original source | `--tokens N` |
| `cloud pull` | Import one saved Cloud app script | `--tenant URL`, `--app ID`, `--out NEW_DIR`, `--json` |
| `version` | Print build version | — |

```sh
# A machine-readable index; exact source text is excluded from JSON.
./bin/qlik-repomap scan --root /path/to/scripts --json > index.json

# Rank summaries relevant to a table, field, variable, source, or path.
./bin/qlik-repomap map --root /path/to/scripts --query Revenue --tokens 2000

./bin/qlik-repomap find Orders --kind table --root testdata/repository
./bin/qlik-repomap find vData --kind variable --root testdata/repository
./bin/qlik-repomap find Revenue --kind field --role definition --root testdata/repository
./bin/qlik-repomap find orders.qvd --kind qvd --root testdata/repository --json

./bin/qlik-repomap deps 'Daily Sales' --direction upstream --root testdata/repository
./bin/qlik-repomap deps Orders --direction downstream --root testdata/repository
./bin/qlik-repomap context Revenue --root testdata/repository --tokens 1500
```

`find` supports `table`, `variable`, `field`, `source`, `qvd`, and `include`. `qvd` selects source names ending in `.qvd`, including STORE destinations. Roles are `definition` and `reference`. The default limit is 50; `--limit 0` returns all matches. Search is case-insensitive, with exact names and source basenames ranked above partial names, owners, and file paths. Basename comparison recognizes both `/` and `\` on every operating system. For sources/includes, a literal filename after leading expansions, such as `$(vRoot)events.qvd`, also matches a basename query for `events.qvd`. This helps discovery without evaluating the expansion or merging endpoint identities. Repeated definitions retain separate occurrences.

`deps` also accepts `--kind file` for including-file endpoints. It prioritizes exact endpoint names/basenames, then partial endpoint matches, independently for each requested direction. Thus a consumer matching the exact filename cannot hide an upstream producer matching its basename. It returns direct edges only. Endpoint identities are typed and case-sensitive; two different scripts defining the same table name may both contribute edges. Use the paths/lines to distinguish occurrences; no global execution order is assumed.

No matches return empty text or an empty JSON array with exit code 0. Unsupported syntax and parse diagnostics go to stderr. Runtime failures return 1; invalid arguments return 2. `scan --strict` still writes the available index and returns 1 if any diagnostics occur.

### Budgets and ranking

`map` and `context` default to `--tokens 4096`. To avoid adding a model-specific tokenizer, the current estimator uses **one estimated token per UTF-8 byte**. Every byte of stdout, including paths, headers, line numbers, and newlines, counts. This conservative allowance generally underfills a model's actual token budget; it is deliberately not a BPE token count. Set a larger budget when more context is useful.

Maps prioritize table definitions and referenced tables, then source relationships and other statements. With `--query`, the strongest occurrence in each statement ranks before structural importance; a large number of partial matches cannot outrank an exact match. Large field lists have a compact fallback that tries to retain the best matching field, input reference, or variable alongside the table and source. An ellipsis marks omitted output fields. Records are never cut in half; very small budgets can produce empty output.

Context prefers exact symbol definitions, references, and definitions of direct dependency neighbors. A field lookup also considers its owning table's direct inputs/outputs, including the recorded range of an anonymous preceding LOAD stage. Neighbors are not expanded recursively. Overlapping source lines appear once per file. When a statement is too large, the matching line and nearby lines take priority. Source lines are never truncated to satisfy a budget. Explicit include/file-path lookups can retrieve literal include targets that exactly match indexed repository paths; a symbol lookup does not expand unrelated includes in its file. This is retrieval, not Qlik path resolution or include execution.

### Index and locations

The JSON index contains `root` and `files`. Each file contains `path`, `symbols`, `edges`, `loads`, `statements`, and `diagnostics`:

- Symbols carry `kind`, `role`, normalized `name`, optional `owner`, path, one-based line/byte column, end line, and statement line/byte ranges. Enclosing brackets/quotes are removed; dollar expansions stay literal.
- Edges carry typed `from` and `to` endpoints, relationship kind, statement start byte, and the source location where the relationship occurs. Byte offsets are zero-based with exclusive statement end offsets.
- LOAD records carry `owner`, `anonymous`, the statement location, and statement byte range. Wildcard-only stages retain ranges without fabricated table or field definitions. Anonymous owners use `@path:line:column` identities.
- Statement records carry the supported syntax `kind`, location, and statement byte range, including symbol-free TRACE statements. Kinds omit the `_statement` suffix; `drop_mapping_table` distinguishes mapping-table drops. Malformed/unsupported statements retain diagnostics instead of a supported statement record.
- Diagnostics carry a code, message, and source location.

Edges point **consumer → upstream input**. `Daily Sales → Orders` means `RESIDENT Orders`. For STORE, `lib://Exports/daily_sales.qvd → Daily Sales` means that output is produced from the table. `file → include` preserves the literal include target. Variable-use edges point to the variable. A `preceding` edge points to the next LOAD stage, with its location on that input stage and `statement_start_byte` on the consuming statement. These are syntactic facts, not evaluated lineage.

Scanning is deterministic and case-insensitive for the `.qvs` extension. It skips `.git`, `.hg`, `.svn`, `node_modules`, `vendor`, `bin`, `dist`, `build`, and `.cache` directories. Symlinks are not followed, including a symlink supplied as the root. A repository `.gitignore` is not interpreted; other QVS files, including untracked files, are scanned. Filesystem errors fail visibly. Files must contain valid UTF-8; UTF-8 BOM and CRLF are supported. Invalid UTF-8 fails with the source path and exit code 1 before any index is written. Convert legacy or UTF-16 exports to UTF-8 before scanning them.

## Understand a Qlik Cloud app

**Available in source builds; not included in the published v0.1.1 binaries.**
Build the current checkout using the instructions above. Supply an API key or
OAuth access token through the `QLIK_CLOUD_TOKEN` environment variable, with
permission to read the selected app and its script. The CLI does not obtain,
refresh, or persist credentials. It accepts an HTTPS tenant origin, not an app URL.

```sh
# Set QLIK_CLOUD_TOKEN in your shell before running this.
mkdir -p .cache/qlik-cloud
./bin/qlik-repomap cloud pull --tenant https://example.eu.qlikcloud.com \
  --app APP_ID --out .cache/qlik-cloud/app-snapshot --json
./bin/qlik-repomap scan --root .cache/qlik-cloud/app-snapshot --strict
./bin/qlik-repomap map --root .cache/qlik-cloud/app-snapshot --query Revenue --tokens 2000
./bin/qlik-repomap context Revenue --root .cache/qlik-cloud/app-snapshot --tokens 2000
```

`cloud pull` reads app metadata, selects the latest saved script from its history,
and downloads **that exact version ID** through the [Qlik Apps API](https://qlik.dev/apis/rest/apps/).
A save during the download cannot substitute a newer version. Unsaved editor
changes are not included; the selected version may no longer be the newest when
the pull finishes. No script writes or reloads occur.

The new snapshot directory contains exactly:

- `script.qvs`: the decoded UTF-8 script, preserving BOM, newlines, section markers,
  tabs, and symbolic paths. No generated headers or newline are added.
- `manifest.json`: schema version 1, tenant/app identity, script version ID,
  retrieval time in UTC, source filename, SHA-256, and byte length. App name,
  script modification time/message, and last reload time are included when supplied.

Use a new destination for each pull. Its parent directories must already exist
and contain no symlinks; if necessary, use their physical path. Existing files
are never overwritten. Snapshot directories use mode `0700` and files `0600`
where the operating system supports these permissions. A failed write removes
only files created by that invocation. If writing the result to stdout fails,
the completed snapshot remains available.

Keep snapshots private: `.cache/` is ignored by this repository and skipped by
ordinary scans, while an explicit snapshot `--root` is scannable. Use one app
per root because the current dependency index does not distinguish equal names
across apps. Subsequent `scan`, `map`, `find`, `deps`, and `context` run offline.
Unsupported constructs remain intact and produce the usual scan diagnostics.

When citing Cloud source, associate `script.qvs` lines with the manifest's
tenant/app/script ID after checking the source hash. Edited files no longer
represent the exact downloaded version. Exported line numbers are not tab-local
Cloud editor coordinates. The reported last reload time does not prove that this
script version reloaded successfully. This increment does not retrieve reload
logs or Cloud lineage.

Requests have a 30-second timeout and a 32 MiB response limit. Normal TLS
verification applies; redirects are rejected. Errors report HTTP status and
access/rate-limit hints without printing response bodies or credentials.

If metadata is readable but script history returns **403**, check script access
for the account represented by the token. In a shared space, Qlik's `Owner` and
`Can manage` roles alone do not grant access to edit other users' load scripts;
the relevant role is **Can edit data in applications**. Alternatively, select
an app that account owns. See [Qlik's shared-space permissions](https://help.qlik.com/en-US/cloud-services/Subsystems/Hub/Content/Sense_Hub/Spaces/managing-shared-spaces.htm).
After changing access, Qlik advises closing all tabs for the affected app and
waiting at least two minutes before reopening it. Verify that the token's account
can open the Data load editor; a different browser account may have different access.
The importer still issues only GET requests, even when the account can edit.

## Grammar support

| Construct | Representation and behavior |
| --- | --- |
| `SET`, `LET` | Variable definitions; SET remains text and LET remains an unevaluated expression; empty values have no invented references; bare names in LET expressions are variable references |
| `$(vName)`, `$(#vNumber)`, `$(vMacro, arg)` | Variable-reference nodes, including nested expansions inside strings, names, and paths |
| `vMetric_$(vRun)`, `Batch_$(vRun)_tail` | Full symbolic variable, table, and field names; separate references to expansion variables, including assignment targets |
| `Table:`, `[Table Name]:`, `"Table Name":`, `$(vTable):` | Literal or symbolic table definitions |
| Backtick names and paths | Variables, tables, fields, sources, and includes; enclosing delimiters normalize for lookup |
| `LOAD`, `DISTINCT`, field expressions, `AS` | Output fields and separate input field references; function names, including `Date#()`/`Num#()`, are not fields; aliases may be symbolic |
| Preceding LOAD statements | Separate stage ranges, output fields, and direct `preceding` edges |
| `FROM`, `RESIDENT` | Source/table references and direct edges |
| Drive, UNC, relative, wildcard, and `lib://` paths | Literal data sources; no filesystem or library resolution |
| File-format options such as `table is Sheet1$` | Literal dollar suffixes and genuine variable expansions remain distinct |
| `STORE [fields FROM] Table INTO source` | Output-source definition, table reference, and output → table edge |
| `RENAME FIELD(S)` / `RENAME TABLE(S)` | Explicit old names are references and new names are definitions; comma-separated pairs retain separate occurrences; `USING` references only the mapping table |
| `DROP FIELD(S) [FROM tables]`, `DROP [MAPPING] TABLE(S)` | Explicit field/table references, including lists and optional field scopes; previous definitions remain indexed |
| `TRACE text` | Text has no field/table/source symbols; explicit dollar expansions remain variable references; no log output is executed |
| `$(Include=...)`, `$(Must_Include=...)` | Literal include references and distinct optional/required edge kinds |
| `JOIN`, `LEFT/RIGHT/INNER/OUTER JOIN`, `CONCATENATE` | Explicit target references; source inputs attach to the named target |
| `NOCONCATENATE`, `MAPPING` | Recognized LOAD prefixes |
| `HIERARCHY(...)` | Input field references and explicit parent/path/depth field definitions; no generated level names or runtime transformation |
| `BUNDLE [INFO] LOAD` | Recognized prefix; retains the LOAD's table and direct source |
| `WHERE`, `WHILE`, `GROUP BY`, `ORDER BY` | Common expressions, unary `NOT`, and field references |
| Function calls with omitted argument positions | Present expressions retain references; commas do not fabricate argument values or names |
| `AUTOGENERATE`, bracketed `INLINE` | Recognized LOAD sources; INLINE data contributes no field/source symbols, while explicit expansions retain variable references |
| `//`, `/* ... */`, `REM ...;` | Comments; no symbols extracted from them |

The grammar has meaningful `variable_name`, `variable_reference`, `bare_variable_reference`, `table_name`, `table_label`, `load_field`, `field_name`, `field_reference`, `data_source`, and `include_path` nodes. See [the grammar README](tree-sitter-qlik/README.md) for binding/query details.

Rename and drop summaries describe source operations, without applying them to
a running data model. Field renames have no inferred table owner. No rename/drop
lineage edges or mapped output names are invented, and later literal RESIDENT
names are not rewritten. Variable uses in these operations and TRACE contribute
file → variable edges; a variable context query retains its matching statements
without expanding unrelated source from the containing file. See Qlik's
[rename](https://help.qlik.com/en-US/cloud-services/Subsystems/Hub/Content/Sense_Hub/Scripting/ScriptRegularStatements/rename-field.htm),
[drop](https://help.qlik.com/en-US/cloud-services/Subsystems/Hub/Content/Sense_Hub/Scripting/ScriptRegularStatements/drop-field.htm), and
[TRACE](https://help.qlik.com/en-US/cloud-services/Subsystems/Hub/Content/Sense_Hub/Scripting/ScriptRegularStatements/Trace.htm)
syntax references. The original operation examples are in
[operations.qvs](testdata/accuracy/operations.qvs).

Preceding LOADs retain separate scopes and source locations:

```qlik
Result:
LOAD NetAmount AS Revenue;
LOAD Amount * Rate AS NetAmount;
LOAD Amount, Rate FROM [raw.qvd] (qvd);
```

Here `Revenue` belongs to `Result`, `NetAmount` belongs to the anonymous stage on line 3, and `Amount`/`Rate` belong to the stage on line 4. Maps show the direct input stages; they do not flatten fields into the outer table or expand wildcards. Try the original accuracy examples with `scan --root testdata/accuracy --strict` and `map --root testdata/accuracy --query Revenue`.

### Deliberate limits

This milestone is a useful subset of Qlik script syntax. SQL semantics, set-analysis expressions, expression-based dollar expansions (`$(=...)`), control-flow evaluation, subroutine execution, inferred table names, automatic concatenation, wildcard field expansion, and field-level lineage are not implemented.

Preceding edges require adjacent semicolon-terminated LOAD statements, with an unlabeled, unprefixed input stage. Comments may separate the stages. A new table label, prefix, assignment, include, unsupported/control statement, or parse error breaks the chain. JOIN cannot be a preceding LOAD. A source-less LOAD without a recognized adjacent input receives an `unresolved-source` diagnostic; a following SQL statement is not interpreted as a source.

Unknown semicolon-terminated statements become `unsupported_statement` nodes and diagnostics. Common line-terminated control headers (`FOR`, `IF`, `NEXT`, `END`, and related keywords) receive diagnostics while supported body statements remain indexable. Those body statements are source facts regardless of whether a Qlik execution would run them. Malformed syntax uses Tree-sitter recovery; affected facts may be incomplete, while surrounding valid statements are retained when possible. An unterminated quote/comment can necessarily obscure later source.

Unsupported statement bodies are opaque: quoted text, comments, bracketed content,
and dollar expansions inside them contribute no symbols or query captures. When
a semicolon is missing, a stateless external scanner can recover before the next
unquoted table label at the start of a line, including indented, quoted, or symbolic
labels. It adds an `unterminated-statement` diagnostic and preserves the recovered
LOAD's original path and line. A URI scheme such as `lib://` is not a label.
Recovery is a conservative syntax boundary heuristic; it does not prove that the
surrounding script would reload. Unlabeled and same-line LOAD text remains opaque.

Balanced standalone literals become `unsupported_literal_statement` nodes with
`unsupported-literal` diagnostics. They contribute no index symbols and do not
hide supported statements that follow them. A quoted table label or literal
inside a valid LOAD expression retains its normal meaning.

Bare JOIN/CONCATENATE targets depend on execution order. They produce an `implicit-target` diagnostic and retain location-based ownership instead of guessing the target. Unlabeled LOADs use `@path:line:column` identities. Variables and all source paths remain symbolic or literal, preserving their original separators. Relative includes remain literal because Qlik's app working directory is unknown. No include file outside the scan is opened, and no QVD data is read.

The CLI scans `.qvs` text files, not binary `.qvw` applications. Export scripts from QlikView before indexing them. HIERARCHY parameters retain their input/output roles, but generated numbered fields and transformed records are not inferred. BUNDLE supports optional INFO before LOAD; SQL, IMAGE_SIZE, and standalone INFO remain outside this support.

Empty LET values and omitted call arguments are accepted as source forms without evaluating them or checking function arity. GROUP BY items can contain structured expressions so their component references remain retrievable. Qlik's LOAD documentation describes a grouping field list; accepting expressions here does not establish runtime validity or validate grouping semantics.

## Aider and other assistants

### Companion skill

The optional [qlik-code-intelligence skill](skills/qlik-code-intelligence/SKILL.md)
guides an assistant through focused maps, symbol lookup, QVD producers/consumers,
and numbered source retrieval. It explains diagnostic handling, byte budgets,
and the limits of direct syntactic dependencies. It requires `qlik-repomap`
v0.1.0 or later on `PATH` (or an explicitly selected executable).

Download `qlik-code-intelligence-skill_v0.1.1.zip` from the
[v0.1.1 release](https://github.com/noesiscarloslisboa/qlik-code-intel/releases/tag/v0.1.1)
and verify its entry in `SHA256SUMS`. Extract its `qlik-code-intelligence` folder
into your assistant's configured skills directory. For Codex, the default is
`~/.codex/skills/`; the skill is available on the next turn. Alternatively, ask
Codex's skill-installer to install from this versioned repository URL:

```text
Install the skill from https://github.com/noesiscarloslisboa/qlik-code-intel/tree/v0.1.1/skills/qlik-code-intelligence
```

Then try prompts such as:

- “Use $qlik-code-intelligence to explain how Revenue is loaded in this repository.”
- “Find the producer and consumers of orders.qvd, citing script paths and lines.”
- “Retrieve the source needed to change Daily Sales, within a small context budget.”

The skill uses the [Agent Skills format](https://agentskills.io/specification)
and includes Codex UI metadata. Its command examples are tested against extracted
release binaries on macOS and Windows. Automatic selection and model behavior
have not been evaluated across assistant products. See the
[skill validation assessment](docs/validation/assistant-skill-assessment.md).
Installation does not bundle the CLI or execute Qlik scripts.

### Query integration

The queries under `tree-sitter-qlik/queries/` use Aider's repository-map conventions:

```scheme
(table_label name: (table_name) @name.definition.table) @definition.table
(resident_clause table: (table_name) @name.reference.table) @reference.table
```

`tags.scm` combines definitions and references; the separate files allow selective use. Tests compile and execute the queries against the actual grammar. **Stock Aider does not automatically support this new language.** Native integration requires registering the Qlik parser and `.qvs` extension in its language provider and loading these tags queries. The standalone CLI works immediately: generate a map or context file and add it to your assistant's prompt/context.

```sh
./bin/qlik-repomap map --root /path/to/scripts --tokens 3000 > qlik-map.txt
```

## Development and verification

```sh
# Build and run the Go tests using the included generated parser.
make build
make test
make integration
make vet

# Validate the companion skill's executable examples (Python stdlib).
python3 scripts/check-skill.py --binary bin/qlik-repomap

# Grammar work also requires Node.js/npm and downloads pinned Tree-sitter 0.25.10.
cd tree-sitter-qlik
npm ci
npm test
cd ..
make check-generated
make check
```

After changing `grammar.js`, run `make generate` and review corpus tree changes. This regenerates the parser and clears the Go build cache: CGO's cache does not reliably notice C files included from another directory. After changing the handwritten `tree-sitter-qlik/src/scanner.c`, also run `go clean -cache`. Review expected syntax trees before using `tree-sitter test --update`.

CI checks reproducible generation, the grammar corpus, race-enabled Go tests, built-executable integration tests, formatting, and vet. Tests cover original multi-file fixtures, every requested construct, recovery, aliases, LET variable references, symbolic names, interpretation functions, preceding LOAD boundaries/scopes, query captures, source locations, exact-match ranking, focused budgets, include isolation, duplicate definitions, CLI errors/JSON, symlink exclusion, and cancellation.

The [retrieval acceptance checks](docs/validation/retrieval-acceptance.md) add
backtick names, documented quote escaping, opaque unsupported bodies, missing-semicolon boundaries,
incremental terminator edits, and exact source retrieval. The handwritten scanner
uses an iterative stack for protected contexts and distinguishes literal name prefixes from dollar expansions. Original QlikView fixtures cover composed names, nested references, Windows basename ranking, and Excel format dollars without adding runtime dependencies.

The CLI uses the standard library for flags and output. Its runtime module dependencies are the official `go-tree-sitter` binding and its `go-pointer` bridge. There is no SQLite dependency. The Go module path is `github.com/noesiscarloslisboa/qlik-code-intel`.

### Release

Version 0.1.1 adds the companion skill and its portable archive. See the [changelog](CHANGELOG.md)
and [release instructions](docs/releasing.md) for version embedding,
verification, and publication steps. Release notes and source archives are on
[GitHub Releases](https://github.com/noesiscarloslisboa/qlik-code-intel/releases/tag/v0.1.1).
The release review records executed checks and platform limits in
[the readiness assessment](docs/validation/release-readiness-assessment.md).

### Retrieval benchmark

An optional benchmark checks twenty source-reviewed questions against original
fixtures. It verifies the first `find` result, first focused map record, first
context block, and expected typed dependency edges. Map/context checks run at
256, 512, and 1024 UTF-8 byte budgets. Commands repeat to check determinism, and
every numbered context line must match the source.

```sh
make benchmark  # Python 3.9+; standard library only
```

The [benchmark assessment](docs/validation/retrieval-benchmark-assessment.md)
records the public results and aggregate results from twenty separate private
questions. To check another source-reviewed manifest:

```sh
python3 scripts/benchmark-retrieval.py \
  --samples-dir /path/to/qvs \
  --manifest /path/to/questions.json \
  --output /tmp/qlik-retrieval-results.json
```

See `testdata/retrieval/questions.json` for the manifest format. Pin every fixture
with its SHA-256 and review expected source locations and edges independently of
CLI output. The runner rejects changed sources and expected context lines that
cannot fit the smallest budget. Reports include case IDs and aggregate outcomes,
without query or source text. These selected checks do not measure exhaustive
precision/recall or validate Qlik reloads. Natural-language question descriptions
are for reviewers; the CLI receives the explicit symbol query in each case.

### External sample validation

The ten QVS samples from the reference repository were tested separately at a
pinned revision. The [assessment](docs/validation/reference-sample-assessment.md)
records results and remaining limits. They include malformed and unsupported
editor examples; a successful scan does not mean each file is valid Qlik or that
every recovered fact is complete.

The GPL samples are not included here. To repeat the audit, separately obtain the
reference repository at the revision in
`docs/validation/reference-samples.json`, build the CLI, then run:

```sh
python3 scripts/validate-reference.py \
  --samples-dir /path/to/vs-qlik-languageserver \
  --output /tmp/qlik-reference-results.json
```

This optional runner uses only Python's standard library. It verifies sample Git
blob hashes, selected symbols/locations, direct dependencies, original context
lines, deterministic output, and budgets. It does not download or execute scripts.
Its aggregate JSON contains no upstream source. Normal project tests use original
fixtures and need neither Python nor the external checkout.

The user also supplied three QlikView apps with confirmed successful reload history.
A local audit of heuristically recovered script candidates indexes 569 LOADs and
checks selected source locations and budgeted retrieval. The
[QlikView assessment](docs/validation/qlikview-sample-assessment.md) records the
results and remaining gaps. The apps and recovered source stay Git-ignored in
`testdata/real_samples/`. Completeness remains pending comparison with authoritative
QlikView exports; no app reload was performed by this project.

To obtain an authoritative script, use QlikView's **Edit Script (Ctrl+E) → File →
Export to Script File…** and save the `.qvs` separately from the recovered copies.
See [Qlik's script editor documentation](https://help.qlik.com/en-US/qlikview/May2024/Subsystems/Client/Content/QV_QlikView/Edit_Script_dialog_box.htm).

## Repository layout

```text
cmd/qlik-repomap/          executable entry point
internal/cli/             commands, flags, output, executable integration test
internal/qlik/            AST extraction, source index, repository scanning
internal/repomap/         ranking, maps, search, dependencies, source context
tree-sitter-qlik/         grammar, generated parser, external scanner, binding, queries
skills/qlik-code-intelligence/  companion assistant instructions and Codex UI metadata
testdata/                 original QVS examples; ignored private samples stay local
docs/superpowers/         milestone design and implementation plan
```

See [AGENTS.md](AGENTS.md) for agent guidance and [CONTRIBUTING.md](CONTRIBUTING.md) for contributor workflow.

## Language references and licensing

[muehan/vs-qlik-languageserver](https://github.com/muehan/vs-qlik-languageserver) was consulted as the requested language/tooling reference. Its GPL-3.0 implementation, grammar, queries, and fixtures are not incorporated into this project. Its QVS validation samples were used externally for the audit described above. This project's grammar and QVS fixtures were written independently.

Syntax was checked against Qlik's [LOAD documentation](https://help.qlik.com/en-US/sense/May2025/Subsystems/Hub/Content/Sense_Hub/Scripting/ScriptRegularStatements/Load.htm), [LET documentation](https://help.qlik.com/en-US/sense/May2025/Subsystems/Hub/Content/Sense_Hub/Scripting/ScriptRegularStatements/Let.htm), [preceding LOAD documentation](https://help.qlik.com/en-US/sense/May2025/Subsystems/Hub/Content/Sense_Hub/Scripting/load-data-from-previously-loaded-table.htm), and [Include/Must_Include documentation](https://help.qlik.com/en-US/sense/May2025/Subsystems/Hub/Content/Sense_Hub/Scripting/SystemVariables/Include.htm). Query conventions follow [Aider's repository mapper](https://github.com/Aider-AI/aider/blob/main/aider/repomap.py). Parsing uses the [official Go bindings](https://github.com/tree-sitter/go-tree-sitter) and [Tree-sitter grammar DSL](https://tree-sitter.github.io/tree-sitter/creating-parsers/2-the-grammar-dsl.html).

Quoting and comments follow Qlik's [quotation rules](https://help.qlik.com/en-us/cloud-services/Subsystems/Hub/Content/Sense_Hub/Scripting/use-quotes-in-script.htm)
and [REM documentation](https://help.qlik.com/en-us/cloud-services/Subsystems/Hub/Content/Sense_Hub/Scripting/ScriptRegularStatements/Rem.htm).
Recovery uses Tree-sitter's [external scanner API](https://tree-sitter.github.io/tree-sitter/creating-parsers/4-external-scanners.html).

Project code is licensed under [MIT](LICENSE). Generated Tree-sitter support headers retain their upstream license in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md). Qlik and Qlik Sense are trademarks of their respective owners; this is an independent project.
