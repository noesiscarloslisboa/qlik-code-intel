# QlikView samples implementation plan

**Goal:** Audit the user's reloadable QVW apps locally and repair Windows path,
composed-name, and format-dollar retrieval defects.

**Architecture:** Keep heuristic script recovery and app provenance private. Add
original regressions to the existing grammar/index/map layers, a stateless name
prefix external token, and platform-independent basename comparison. Execute
inline as part of the approved parser/retrieval accuracy work.

- [x] Inventory app files; confirm reload history; ignore the local sample folder.
- [x] Recover bounded UTF-8 candidates into the ignored folder and record hashes,
  offsets, line counts, and the lack of authoritative export comparison.
- [x] Scan the candidates and identify concrete syntax/retrieval gaps.
- [x] Add original failing parser/query cases for nested/composite names and
  literal format dollars; add Windows path/basename retrieval regressions.
- [x] Update grammar, scanner, extraction, and basename comparison; regenerate and
  inspect syntax trees before accepting corpus expectations.
- [x] Run focused parser/retrieval/query/executable checks and fix failures.
- [x] Audit source locations, expected private sample occurrences, direct edges,
  deterministic budgeted maps/context, and original numbered source.
- [x] Update aggregate assessment and documentation; retain unsupported diagnostics
  and the separate pending authoritative-export completeness check.
- [x] Run `make check`, C scanner warnings check, and the pinned reference audit.

Verification completed after the final scanner change:

- `make check`: reproducible generation, 17/17 grammar cases, formatting,
  race-enabled Go tests, built-executable integration tests, vet, and build pass.
- `clang -std=c11 -Wall -Wextra -Werror -fsyntax-only`: scanner passes.
- The private audit verifies unchanged app hashes, nine selected source locations,
  and sixteen deterministic map/context budget checks against original lines.
  It records 567 LOADs and six syntax-error diagnostics, with six missing tokens
  and unsupported/implicit-target/control/source diagnostics retained.
- The pinned external audit passes twelve budget checks; its decoded JSON exactly
  matches the committed aggregate report.
- The baseline repository still scans strictly with 21 definitions, 26 references,
  and nine edges; the original QlikView fixture also scans strictly without diagnostics.

Authoritative `.qvs` export comparison remains pending as a separate completeness
check. The user-confirmed reload history does not make heuristic recovery complete.
