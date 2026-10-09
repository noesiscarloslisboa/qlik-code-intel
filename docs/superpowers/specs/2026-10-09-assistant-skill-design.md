# Companion assistant skill

The user approved a small skill alongside the existing parser and CLI. Add
`skills/qlik-code-intelligence/SKILL.md` using the Agent Skills format, with optional
Codex UI metadata in `agents/openai.yaml`. Keep all operational instructions in
the entry point: no extra runtime scripts, copied manuals, or bundled executables.

Activate for QVS repository navigation: locating table/field/variable definitions,
QVD consumers/producers, and retrieving source context for explanation or edits.
Discover an installed `qlik-repomap`, inspect its version/help, and choose only
the commands required by the task. Treat repository maps as retrieval summaries;
inspect numbered source before explaining or changing code. Preserve literal
variables/paths, separate aliases from inputs, and describe direct syntactic
dependencies without claiming complete runtime lineage or reload validity.

Document conservative UTF-8 byte budgets, diagnostic stderr and strict-scan
behavior, unsupported source, UTF-8 QVS inputs, and root selection. The skill
must never execute Qlik/includes/QVDs or silently install/download software.
Installation remains a documented user action; normal automatic skill discovery
stays enabled. Avoid advertising unsupported Aider integration.

Publish v0.1.1 with a platform-independent skill zip plus the three native binary
archives. Keep a single top-level `qlik-code-intelligence` folder in the skill
archive, with skill instructions, UI metadata, the MIT license, and build metadata.
Package only the explicit files from the exact tagged source. Older source-only
tags must still build without a skill. Include every archive in SHA256SUMS and
gate publication on native tests and skill example checks.

Validate skill structure with the skill-creator validator. Execute every documented
CLI example against original fixtures in a temporary repository, checking real
source lines, field roles, dependency directions, diagnostics, and output budgets.
Use the existing public retrieval benchmark for broader command coverage. These
checks validate executable examples, not automatic selection or model behavior
in every assistant product. Record that distinction in the validation report.
