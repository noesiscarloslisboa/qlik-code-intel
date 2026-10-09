# tree-sitter-qlik

An independently implemented Tree-sitter grammar for a practical subset of Qlik load scripts (`.qvs`), including shared Qlik Sense and QlikView syntax. Part of [qlik-code-intelligence](../README.md), licensed under [MIT](../LICENSE).

The grammar produces structured assignments, labels, LOAD fields/aliases, sources, RESIDENT inputs, STORE outputs, include directives, and JOIN/CONCATENATE prefixes. It recognizes common expressions, function calls (including interpretation functions ending in `#`), comments, quoted identifiers, and symbolic dollar expansions in labels/aliases as well as paths. Unsupported statements are explicit nodes; Tree-sitter error recovery handles malformed input.

## Generate and test

```sh
npm ci
npm run generate
npm test
```

Generator: Tree-sitter CLI **0.25.10**. Generated language ABI: **15**. Runtime: official `go-tree-sitter` **v0.25.0**. Generated sources are included in `src/`; `src/scanner.c` contains handwritten recovery and name-prefix logic. Regeneration is checked for reproducibility with `npm run check-generated`.

From the repository root, `make generate` also clears the Go CGO cache after regeneration. Corpus expectations are syntax-tree contracts; inspect differences and unexpected ERROR/MISSING nodes before updating them.
Clear the Go cache after handwritten scanner edits too; CGO does not track these
external C includes reliably.

## Go binding

Within this checkout:

```go
import (
    sitter "github.com/tree-sitter/go-tree-sitter"
    qlik "github.com/noesiscarloslisboa/qlik-code-intel/tree-sitter-qlik/bindings/go"
)

func parseExample() {
    parser := sitter.NewParser()
    defer parser.Close()
    if err := parser.SetLanguage(sitter.NewLanguage(qlik.Language())); err != nil {
        panic(err)
    }
    tree := parser.Parse([]byte("Orders: LOAD ID FROM [orders.qvd] (qvd);"), nil)
    if tree == nil {
        panic("parser returned no tree")
    }
    defer tree.Close()
    // Inspect tree.RootNode().ToSexp() or execute queries against it.
}
```

The binding is part of the root Go module. It exposes a `tree_sitter_qlik` C language pointer and requires CGO/a C compiler. All allocated parsers, trees, and query resources must be closed.
The binding compiles both `src/parser.c` and `src/scanner.c`. Native integrations
must include both files; compiling the generated parser alone is insufficient.

## Queries and syntax nodes

| File | Purpose |
| --- | --- |
| `queries/tags.scm` | Combined Aider-compatible definitions and references |
| `queries/definitions.scm` | Variable/table/field/output-source definitions |
| `queries/references.scm` | Variable/field/table/source/include references |
| `queries/highlights.scm` | Basic semantic highlighting |

Definitions pair `@name.definition.*` with `@definition.*`; references use `@name.reference.*` and `@reference.*`. Aliased inputs are references, while aliases are output definitions. Unaliased simple fields are both output definitions and input references. Bare names in LET expression context are variable references; the same syntax in LOAD expressions remains field references. Function identifiers and string contents are not bare variable/field references, while dollar expansions inside strings retain explicit reference nodes. Wildcards have no fabricated field definitions.

Important nodes and fields:

```text
set_statement / let_statement: name, value
variable_reference: name
bare_variable_reference: identifier / bracket_name / quoted_name / backtick_name
function_call: function (identifier / interpretation_function), child expressions
table_label: name
load_statement: fields
load_field: expression, alias
from_clause: source
resident_clause: table
join_prefix / concatenate_prefix: table
hierarchy_prefix: node_id, parent_id, node_name, parent_name, path_source,
                  path_name, path_delimiter, depth
bundle_prefix: BUNDLE, optional INFO
store_statement: fields, table, source
include_statement: mode, source
rename_field_statement / rename_table_statement: rename pairs or rename_using
field_rename / table_rename: old, new
rename_using: table
drop_field_statement: drop_field_list, optional drop_from
drop_table_statement: optional mapping keyword, drop_table_list
trace_statement: optional trace_value (text, quoted text, explicit expansions)
```

Names use `identifier`, `bracket_name`, `quoted_name`, `backtick_name`, or `expanded_name` children; variable/table names and field aliases also accept a pure `variable_reference`. An `expanded_name` preserves composed spelling such as `vMetric_$(vRun)_tail`, with `name_fragment` and nested `variable_reference` children. References inside assignment names are separate from their whole symbolic definitions. Backtick names support expansions, including in variable names, field expressions, data/include paths, SET values, and format options. Expansions remain `variable_reference` nodes even inside strings or quoted paths. LET-specific parsing retains the same public `function_call`, `binary_expression`, `unary_expression`, and `parenthesized_expression` containers. `src/node-types.json` is the complete machine-readable node schema.

HIERARCHY's three required inputs and optional path-source parameter use
`field_reference` nodes, or an explicit variable expansion. Its optional parent,
path, and depth output names use `field_name` nodes, including single-quoted
names in these parameter positions. Queries capture those explicit definitions
without inventing numbered hierarchy fields. The path delimiter does not become
a field reference. Optional argument positions retain their commas.

Empty LET values have no `value` child. Function calls can contain omitted
argument positions; only present expressions have syntax nodes and references.
GROUP BY items use structured expressions with component field references.
The parser does not validate function arity or grouping semantics. BUNDLE with
optional INFO precedes LOAD; its INLINE data has no field/source nodes beyond
the opaque bracketed data container, while explicit dollar expansions remain
variable references. No bundled file is opened.

RENAME pairs use `field_reference` / `table_name` for old names and `field_name`
/ `table_name` for new definitions. Queries preserve both roles. `rename_using`
references its mapping table without generating names from its contents. DROP
lists contain explicit references, including optional FROM table scopes and the
MAPPING modifier. Single-quoted operation names normalize like other quoted
names. The CLI does not assign field ownership or modify earlier definitions.
TRACE has a text container rather than a field/variable expression: ordinary
words and paths have no symbol captures, while explicit expansions are retained.
TRACE macro arguments also stay text, including function-like fragments and
nested expansions inside quotes; they do not create LOAD field references.
Comments stay opaque even within a TRACE value. These statements break preceding
LOAD adjacency. The additive `File.statements` index records their source ranges,
including TRACE statements with no symbols, for explicit operation summaries.

The stateless external scanner recognizes a name prefix immediately before `$(`,
allowing the ordinary identifier token to retain literal dollars elsewhere.
Continuation fragments require adjacency, so whitespace does not merge separate
names. File-format options accept literal dollars, including Excel sheet suffixes
such as `table is Sheet1$`, while genuine `$(` expansions keep reference nodes.
Drive, UNC, relative, wildcard, and library paths retain their source spelling.

Each semicolon-terminated LOAD has its own `load_statement`, including preceding LOADs. The Go indexer links a source-less LOAD to an adjacent unlabeled, unprefixed input LOAD; the grammar does not collapse stages or infer field lineage. `File.loads` records preserve the ranges of wildcard-only anonymous stages for maps and direct source context. Unsupported boundaries and unresolved inputs produce index diagnostics.

Balanced standalone strings or quoted names become
`unsupported_literal_statement` nodes, allowing following statements to recover.
They are invalid statements and produce index diagnostics without index symbols.
Valid quoted table labels and literals inside expressions are unaffected.

An `unsupported_statement` contains its keyword and a single opaque
`unsupported_body` or `unterminated_unsupported_body` token. The terminated token
includes its semicolon. The stateless external scanner protects quoted strings,
brackets, comments, and nested dollar expansions, and bounds an unfinished body
before an unquoted line-start table label or EOF. The Go indexer emits `unsupported`
and, for an unfinished body, `unterminated-statement`. Body contents have no
variable/field nodes and cannot leak into query captures. The scanner leaves
unlabeled and same-line LOAD text opaque, excludes URI schemes from label detection,
and opts out of Tree-sitter's generic error-recovery state. Unterminated protected
contexts can still obscure later input. No execution semantics are inferred.

Register the parser and extension with your consumer to use the queries. Shipping this grammar does not add Qlik to stock Aider or to a prebuilt language pack. These files are not a complete LSP locals/scoping implementation.

Capture text retains its original Qlik spelling, including enclosing brackets or quotes. The Go CLI normalizes these names. A native consumer should normalize capture text when connecting differently quoted spellings of the same symbol.
Backtick names lose only their enclosing delimiters during CLI normalization.
Qlik's documentation excludes backticks from its list of escape delimiters, so
doubled backticks are not accepted as an escape. Use double quotes or brackets
around a name that contains a literal backtick. Doubled double quotes and single
quotes retain their existing decoding behavior.

See the root README for supported syntax, retrieval behavior, limits, provenance, and verification commands. No GPL-licensed code or fixtures from the language reference are included.
