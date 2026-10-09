# QlikView sample validation and source retrieval

The user supplied three QVW apps in `testdata/real_samples` and confirmed their
successful reload history. Use them locally for the already-authorized accuracy
pass. Keep the folder Git-ignored; no binary app, extracted script, credentials,
or business-data contents belong in the open-source deliverable.

A bounded, read-only examination of the first compressed header found UTF-8
script candidates with Qlik tab markers. Local copies retain exact bytes and a
manifest with QVW/script hashes, stream/candidate offsets, and line counts. This
is heuristic recovery, not a production QVW reader. Authoritative `.qvs` exports
or QlikView project `LoadScript.txt` files are still needed to confirm completeness;
no Qlik app, SQL, include, connection, or filesystem source is executed.

Fix independently reproduced gaps within the existing source-fact model:

- Windows drive, UNC, relative, wildcard, and symbolic paths remain literal.
  Compare source basenames using both slash forms on every host, without changing
  endpoint identities, source text, path resolution, or file access.
- Composite names retain their full symbolic spelling, with nested expansion
  nodes and separate references to their constituent variables. A stateless
  external name-prefix token separates a literal name prefix from `$(` without
  breaking ordinary dollar-containing identifiers. Assignment names contribute
  expansion references as well as values. Queries continue capturing whole names.
- Literal dollars in file-format options are opaque characters, not malformed
  expansions or variable references. Protect existing nested format parentheses
  and genuine expansions.

Retain diagnostics for unsupported SQL, control semantics, implicit targets, and
other language features outside this pass. Do not add lineage, a binary-app command,
a database, or runtime execution. Empty LET values and BUNDLE are recorded as
remaining coverage gaps unless required by a regression introduced here.

Verify original fixtures and source/query locations, execute the CLI against the
recovered text, check map/context budgets and determinism, compare context with
original numbered source, and rerun the pinned reference audit plus `make check`.
Report source-fact coverage and extraction limitations separately; do not label
heuristically recovered text as complete application acceptance.
