# Assistant Skill Implementation Plan

**Goal:** Ship a small, tested Qlik repository-navigation skill with v0.1.1.

**Architecture:** An Agent Skills entry point invokes the existing CLI. Release
packaging emits an additional portable zip from the exact tagged source and
includes it in the verified checksum set. Parser and CLI behavior stay intact.

**Tech Stack:** Markdown, YAML metadata, Python 3.12 stdlib release tooling,
existing Go/Tree-sitter CLI, GitHub Actions native matrix.

Implementation runs inline in this session; the user already approved this scope.

- [x] Write the skill and generate concise Codex UI metadata. Provide a runnable
  example using `./scripts` as the explicitly selected sample repository root.
- [x] Add `scripts/check-skill.py` to execute documented command examples against
  a temporary copy of `testdata/repository`, checking the Revenue definition at
  `20_summary.qvs:4`, its upstream Orders input, the STORE QVD producer, original
  numbered context, strict diagnostics, and byte budgets. Invoke it with:
  `python3 scripts/check-skill.py --binary bin/qlik-repomap`.
- [x] Extend `scripts/release.py` with `skill` packaging and explicit optional
  skill inclusion in checksum assembly. Add archive inventory, missing-file,
  checksum tampering, and old-tag regressions in `scripts/test_release.py`.
- [x] Add a read-only skill packaging job to `.github/workflows/release.yml`,
  include its artifact/checksum only when tagged source contains the skill, and
  run example checks against each extracted native executable. Add ordinary CI
  checks against the built CLI. Lint with actionlint.
- [x] Document binary prerequisites, archive extraction, supported skill
  installation, example prompts, and v0.1.1 release notes in README and docs.
- [x] Run skill-creator validation, Python tests, documented examples, public
  benchmark, `make test integration vet build`, and local archive verification.
- [x] Push the verified source; confirm ordinary CI. Run a pre-tag native dry run
  with `gh workflow run release.yml --ref main -f tag=v0.1.1 -f source-ref=main
  -f publish=false`. Tag that exact successful candidate commit v0.1.1 and push
  the tag to trigger publication after all checks pass. Do not move v0.1.0.
- [x] Download public assets, verify all four archive checksums and metadata,
  validate the extracted skill, run its examples against the downloaded native
  executable, and record results and testing limits.
