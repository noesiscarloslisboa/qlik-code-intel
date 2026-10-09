# Companion skill validation — 2026-10-09

The `qlik-code-intelligence` skill and v0.1.1 packaging were tested from commit
`b7d17dedf97cc8b046435b94119a4dfac02fdc34`. The skill is an optional Agent Skills
entry point with Codex UI metadata. It invokes the existing CLI; it adds no parser
behavior, runtime dependency, database, MCP server, or Qlik execution.

The [candidate CI](https://github.com/noesiscarloslisboa/qlik-code-intel/actions/runs/37912941248)
passed on Go 1.23.x and stable. The
[pre-tag dry run](https://github.com/noesiscarloslisboa/qlik-code-intel/actions/runs/37912958567)
passed all required jobs with publication disabled, using that exact source SHA.
Only afterward was the annotated v0.1.1 tag created. The existing v0.1.0 tag still
resolves to `c03711dbdc3b2908eacfcc594dde941ca42b4236`.

The [tag-triggered publication](https://github.com/noesiscarloslisboa/qlik-code-intel/actions/runs/37913322630)
repeated all required jobs successfully and published the
[v0.1.1 release](https://github.com/noesiscarloslisboa/qlik-code-intel/releases/tag/v0.1.1):
three native binary archives, the portable skill ZIP, and `SHA256SUMS`.

| Target | Native runner | Go | C compiler | Dry run |
| --- | --- | --- | --- | --- |
| darwin/arm64 | macos-15 | 1.27.2 | Apple Clang 17.0.0 | Pass |
| darwin/amd64 | macos-15-intel | 1.27.2 | Apple Clang 17.0.0 | Pass |
| windows/amd64 | windows-2025 | 1.27.2 | MSYS2 UCRT64 GCC 16.2.0 | Pass |

Each native job ran uncached race-enabled Go tests, executable integration tests,
vet, and extracted-archive checks. Every executable passed forty selected public
retrieval checks and all eight commands from the skill's example block. Windows
also passed DLL inspection and smoke checks without compiler paths. Ubuntu ran
`make check`, including grammar corpus and reproducible generation, on Go 1.23.x.

`scripts/check-skill.py` executes the actual documented commands with argument
vectors against a temporary copy of the original five-file fixture repository.
Its temporary path contains spaces. Source and dependency oracles check:

- The focused Revenue map starts with `20_summary.qvs:1` and retains the matching
  field; `find` locates the definition at its original line 4.
- RESIDENT points Daily Sales upstream to Orders. QVD upstream retrieval finds
  the STORE producer; downstream retrieval finds its consumer while retaining
  the symbolic `$(vData)/orders.qvd` endpoint.
- Context contains original numbered source lines, and every budgeted stdout
  fits its declared conservative UTF-8 byte allowance.
- LOAD expression inputs and output aliases have distinct reference/definition
  roles. The baseline has five files and no diagnostics.
- A separate original unsupported-SQL example returns exit 1 and diagnostic
  stderr while preserving a supported following LOAD in a valid partial index.

Local `make test integration vet build`, actionlint 1.7.12, and twelve Python
packaging/benchmark self-tests passed. Tests cover the skill's archive inventory,
license and source metadata, missing files, old-tag omission, and checksum
tampering/missing/extra files with and without a skill. The skill-creator validator
accepted the source skill and an extracted local archive; the extracted examples
passed as well. Validation uses temporary PyYAML only for that external validator;
the project release tooling remains Python stdlib.

The downloaded dry-run archives passed independent SHA-256 and source-commit
checks. macOS archives contain eight files, Windows contains fourteen, and the
portable skill contains exactly four: `SKILL.md`, `agents/openai.yaml`, `LICENSE`,
and `build-info.json`, under one `qlik-code-intelligence` folder. Its metadata
records v0.1.1, the tested SHA, and minimum CLI version v0.1.0. No fixtures, private
apps, recovered scripts, module caches, or external GPL source are packaged.

All four public archive checksums were independently verified after publication.
Every archive matches its dry-run counterpart byte for byte and identifies the
same tested source SHA. This is an observed repeat build with these toolchains,
not a reproducibility promise across toolchain versions. The skill-creator
validator accepted the extracted public skill. All eight extracted skill examples
and forty public retrieval checks passed again against the downloaded Apple
Silicon executable on the local Mac.

These are executable-example and source-fact checks. Automatic skill selection,
prompt adherence, and model behavior across assistant products have not been
evaluated. Selected retrieval cases do not establish exhaustive recall, complete
runtime lineage, or Qlik reload validity. The skill requires a separately installed
CLI. Existing syntax/recovery limits apply; binaries remain unsigned.
