# Changelog

## 0.1.0 — release candidate

First end-to-end implementation of qlik-code-intelligence. Publication is pending;
this entry describes the local candidate rather than a published tag.

- Original Tree-sitter grammar, generated parser, Go binding, and Aider-style
  definition/reference queries for the documented Qlik script subset.
- `qlik-repomap scan`, `map`, `find`, `deps`, and `context` with source paths,
  one-based line/byte columns, direct typed edges, deterministic ranking, and
  conservative UTF-8 byte budgets.
- Qlik Sense library paths and QlikView filesystem paths remain literal. Composed
  names and nested variable expansions retain their symbolic spelling.
- Graceful unsupported-syntax diagnostics, bounded recovery, preceding LOAD
  ranges, and explicit HIERARCHY input/output roles.
- Original grammar, extraction, query, retrieval, and executable CLI tests;
  reproducible parser generation; optional source-reviewed retrieval benchmark.
- Invalid UTF-8 scripts fail before indexing. Output failures, including command
  help, return a runtime error instead of a success status.

This version requires CGO and a C compiler. It does not execute Qlik, resolve
variable values, reload apps, or implement advanced lineage, MCP, or a GUI.
Authoritative export comparison for the private recovered samples remains pending.
