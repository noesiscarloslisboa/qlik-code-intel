# Statement boundaries and backtick identifiers implementation plan

**Goal:** Preserve labeled LOAD retrieval after unfinished unsupported statements
and support documented backtick names without guessing execution semantics.

**Architecture:** A stateless C external scanner bounds opaque unsupported bodies.
The grammar and normalization gain a backtick node; the existing Go index and CLI
continue extracting AST facts. No new runtime dependencies or public commands.

**Tech stack:** Tree-sitter CLI 0.25.10, official Go binding 0.25.0, Go standard
library, existing CGO toolchain. Execute inline under the user's approval.

## Reproduce

- [x] Add original Go cases for missing terminators before labeled LOADs, protected
  text, EOF, conservative unlabeled handling, and nested macro contexts.
- [x] Add backtick names/escaping/query cases and confirm failures in the baseline.

## Implement

- [x] Add `tree-sitter-qlik/src/scanner.c`, external body tokens/error sentinel,
  opaque fallback grammar, and the Go binding include. Diagnose unterminated bodies.
- [x] Add backtick nodes to names, paths, SET values, format specs, and unsupported
  literal recovery; update Go name normalization.
- [x] Regenerate, inspect corpus changes, update reviewed expectations, clear the
  CGO cache, and fix focused parser/retrieval tests.

## Retrieval and verification

- [x] Add original representative fixtures and CLI acceptance questions for tables,
  aliases, QVDs, direct dependencies, maps, and numbered source context.
- [x] Verify meaningful executed query captures and byte/UTF-8 budget invariants.
- [x] Re-run the pinned external audit; record recovery improvements and remaining
  malformed/unsupported cases. Keep previous audit counts identifiable as history.
- [x] Update README, grammar README, and AGENTS with scanner/recovery/quoting rules.
- [x] Run complete project verification and record the results.
- [x] Record application-level acceptance as pending until known-valid scripts
  are available; do not imply independent fixtures were reloaded in Qlik.

## Verification result

Completed with `make check`: reproducible generated artifacts, 15 grammar cases,
formatting, race-enabled Go tests, built-executable integration tests, vet, and
build all pass. The handwritten scanner also passes Clang's C11 syntax check with
`-Wall -Wextra -Werror`. Incremental terminator edits match fresh parse trees.

The rebuilt CLI passes the pinned external audit: 28 LOADs, 105 definitions,
217 references, 94 direct edges, and 78 retained diagnostics across ten verified
samples. All 12 map/context budget and determinism checks pass; the report matches
the final executable. The original five-file fixture remains at 21 definitions,
26 references, nine edges, and no diagnostics. The original hardening fixture has
11 definitions, ten references, six edges, and two intentional diagnostics.

Documentation review corrected the initial doubled-backtick assumption. Backtick
names are supported; doubled backticks are not treated as an escape. Tests cover
documented doubled double quotes and alternative quotation around literal backticks.

Known-valid application acceptance remains **pending**. The user does not have
confirmed reloadable scripts yet and may provide them later. No Qlik runtime or
application-level acceptance is claimed. No advanced lineage, MCP, GUI, database,
or new runtime dependency was added.
