# Reference sample validation and bounded recovery

Use the user-selected `muehan/vs-qlik-languageserver` validation scripts as an
external black-box corpus. Pin revision `681d077f85cf277dc578f4aa4ef34ae0048d6ff6`,
enumerate all ten QVS blobs, and verify their Git blob hashes before scanning.
Keep the original samples and GPL license in a temporary directory outside this
MIT project; do not vendor or rewrite those samples into project fixtures.

Assess scan diagnostics, locations, symbols, maps, and retrieval separately.
These are editor/linter examples, not ten known-valid applications. Do not treat
unsupported statements or malformed input as evidence that all Qlik constructs
must be accepted. Record the remaining limits without hiding diagnostics.

Fix one existing recovery requirement: balanced standalone string/quoted-name
fragments can cause Tree-sitter recovery to obscure later supported statements.
Recognize those root-level fragments explicitly as unsupported literal statements,
with diagnostics and no extracted references. They remain invalid Qlik statements;
normal literal expressions, table labels, and comments retain their existing AST.
Unterminated quotes may still obscure subsequent source.

A second independently reproduced defect is lexical priority for unary NOT:
the supported operator can be mistaken for a field/variable identifier before a
function call. Prefer the reserved operator token in LOAD/LET expressions, while
preserving longer names such as `Notable`. Test all keyword cases and avoid
phantom NOT symbols; the reference source's valid `NOT IsNull(...)` clause should
no longer generate a syntax diagnostic. This repairs existing expression support.

Write original minimal recovery tests and a corpus case from the documented
literal/name distinction, rather than adapting upstream code. Preserve paths,
lines, and all existing query conventions. A literal fragment breaks a preceding
LOAD chain, as other unsupported statements do.

Re-run the external corpus and all project checks after regeneration. Save an
aggregate assessment and an opt-in reproducible validation runner using only
Python's standard library and the built Go executable. The runner reads an
externally supplied checkout; it never executes Qlik scripts or downloads files.
Its manifest contains upstream paths/hashes, not GPL script contents. Ordinary
builds and tests remain offline and require no Python or external corpus.

Acceptance: all ten files scanned without CLI crashes; exact upstream content
verified; whole-record UTF-8 budgets and deterministic maps/context checked;
selected definitions/references and source lines checked; recovery regression
fails before the fix and passes after it; generated parser/corpus, race tests,
executable integration, formatting, vet, and build all pass. Report unsupported
and malformed cases candidly rather than claiming complete language coverage.
