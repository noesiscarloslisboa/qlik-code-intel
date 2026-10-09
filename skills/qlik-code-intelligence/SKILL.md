---
name: qlik-code-intelligence
description: Find Qlik Sense and QlikView QVS script definitions, references, direct dependencies, and focused source context using qlik-repomap. Use when explaining repositories or saved Qlik Cloud script snapshots, locating tables, fields, variables, or QVD producers and consumers, or gathering context before script edits.
license: MIT
---

# Qlik code intelligence

Use `qlik-repomap` to retrieve source facts from UTF-8 `.qvs` scripts. It works
offline without Qlik. It accepts exported scripts and Cloud snapshots, not binary
`.qvw` apps. Cloud fetching is a separate explicit command.

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

## Qlik Cloud snapshots

For an existing snapshot, use its directory as `--root` and inspect `manifest.json`.
Check schema version 1, `source_file`, `source_bytes`, and `source_sha256` against
the local source before attributing it to the recorded Cloud version. For example,
`shasum -a 256 script.qvs` or PowerShell's `Get-FileHash script.qvs -Algorithm SHA256`
checks the hash. Cite the tenant, app ID, and script ID alongside original
`script.qvs` lines. If the hash differs, identify the source as locally modified;
do not present it as the exact saved Cloud version or silently refresh it.

When the task requests a Cloud import, use the user-selected tenant/app and
`cloud pull --tenant HTTPS_ORIGIN --app APP_ID --out NEW_DIRECTORY`, with
`QLIK_CLOUD_TOKEN` supplied outside chat/command arguments. Check `cloud pull --help`
first: this command requires v0.2.0 or a current source build and is absent from v0.1.1.
The parent must already exist without symlinks; choose a new private destination,
such as under this project's ignored `.cache/qlik-cloud/`. Then retrieve offline.
Ordinary local navigation does not require fetching Cloud source.

Keep one app per root. A pull pins a saved script history ID, excluding unsaved
editor changes. Last reload time does not prove that this script version reloaded;
exported lines are not tab-local editor coordinates. This workflow retrieves no
reload logs or Cloud lineage and makes no app changes.

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
- From v0.2.0, explicit rename targets are source definitions and old/drop names
  are references. Earlier facts remain indexed; these operations do not establish
  a final runtime data model. USING maps are not evaluated. TRACE retains explicit
  variable uses without executing its message or turning its words into symbols.
- The CLI does not execute scripts/includes, open QVD data, or follow symlinks.
  Do not use this retrieval workflow to execute Qlik or resolve runtime paths.
  Unsupported syntax or incomplete exported scripts can limit the available facts.

Give the requested explanation or edit using the retrieved source, its locations,
and any material uncertainty. If the task includes edits, re-run the focused
retrieval afterward and distinguish static checks from a real Qlik reload.
