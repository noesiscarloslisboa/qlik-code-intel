---
name: qlik-code-intelligence
description: Find Qlik Sense and QlikView QVS script definitions, references, direct dependencies, and focused source context using qlik-repomap. Use when navigating or explaining a QVS repository, locating tables, fields, variables, or QVD producers and consumers, or gathering context before script edits.
license: MIT
---

# Qlik code intelligence

Use `qlik-repomap` to retrieve source facts from UTF-8 `.qvs` scripts. It works
offline without Qlik. It accepts exported scripts, not binary `.qvw` apps.

## Locate the CLI and repository

Use the user's executable path or `qlik-repomap` on PATH; a source checkout may
have `./bin/qlik-repomap`. Windows executables end in `.exe`. Check `version` and
command `--help` when the installed interface is uncertain. These commands are
available from v0.1.0. Select `--root` to include the relevant script repository;
the current directory is only the default, not evidence of the intended root.

If the CLI is unavailable, point to the project's
[release downloads](https://github.com/noesiscarloslisboa/qlik-code-intel/releases).
Do not silently download, install, or build it. Continue useful source inspection
when possible and state that the index could not be checked.

## Choose focused retrieval

- Start with `map` for an unfamiliar repository or `map --query NAME` for a
  focused overview. Avoid dumping a full JSON index into the model context.
- Use `find` to distinguish definitions from references and narrow by `--kind`
  and `--role`. Add `--json` for structured facts; `--limit 0` preserves all
  matches when repeated definitions matter.
- Use `deps` with an explicit direction for direct inputs or consumers. For QVD
  queries, upstream finds producer tables and downstream finds consuming tables.
- Use `context` to retrieve numbered original source before explaining or editing
  a statement. Cite its relative file path and original one-based line numbers.
- Use `scan --strict --json` to audit unsupported syntax and index coverage when
  that matters to the task. Every command rescans current files; no persistent
  index or prior `scan` is required.

For example, if the chosen repository's QVS files are under `./scripts`:

```sh
qlik-repomap version
qlik-repomap map --root ./scripts --query Revenue --tokens 1200
qlik-repomap find Revenue --root ./scripts --kind field --role definition --json
qlik-repomap deps 'Daily Sales' --root ./scripts --kind table --direction upstream --json
qlik-repomap deps daily_sales.qvd --root ./scripts --kind qvd --direction upstream --json
qlik-repomap deps orders.qvd --root ./scripts --kind qvd --direction downstream --json
qlik-repomap context Revenue --root ./scripts --tokens 1200
qlik-repomap scan --root ./scripts --strict --json
```

Replace example names/root with the user's target. Quote names, especially spaces
and dollar expansions; single quotes preserve `$()` in POSIX shells and PowerShell.
With an executable path containing spaces, use the shell's invocation syntax
(for example PowerShell's `& 'C:\Tools\Qlik Tools\qlik-repomap.exe' version`).

## Budgets, diagnostics, and source evidence

`--tokens` currently counts **UTF-8 bytes**, including headers and line numbers;
it is a conservative allowance, not a model tokenizer. Small budgets may return
empty output because records and source lines are never cut in half. Increase
the budget or narrow the query/root before interpreting an empty result as absence.

Diagnostics go to stderr; supported facts remain retrievable. A strict scan
returns 1 when diagnostics exist and can still emit a valid partial JSON index.
Exit 1 can also mean a runtime failure: inspect stderr and whether an index was
produced. Exit 2 means invalid arguments. Keep relevant diagnostics in the answer;
a clean scan does not prove that a script reloads successfully. Search matches are
ranked and case-insensitive, while dependency endpoint identities are typed and
case-sensitive. Preserve repeated definitions and their separate locations.

## Interpret facts accurately

- Output aliases are field definitions; expression inputs are references and
  function names are not fields. In LET expressions, bare names are variables;
  in LOAD expressions they are fields.
- Edges point consumer to upstream input. `Daily Sales -> Orders` describes a
  RESIDENT input; a STORE edge points output source to producing table. These are
  direct syntactic relationships, not evaluated execution order or complete
  field-level lineage. Follow additional edges only when the task requires it.
- Keep variables, composed names, library paths, drive/UNC paths, and includes
  symbolic/literal. Do not evaluate expansions, equate basename matches, guess
  implicit JOIN/CONCATENATE targets, or flatten preceding LOAD stages.
- The CLI does not execute scripts/includes, open QVD data, or follow symlinks.
  Do not use this retrieval workflow to execute Qlik or resolve runtime paths.
  Unsupported syntax or incomplete exported scripts can limit the available facts.

Give the requested explanation or edit using the retrieved source, its locations,
and any material uncertainty. If the task includes edits, re-run the focused
retrieval afterward and distinguish static checks from a real Qlik reload.
