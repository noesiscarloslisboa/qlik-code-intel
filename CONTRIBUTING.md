# Contributing

Build with Go 1.23+ and a C compiler. Grammar changes additionally need Node.js/npm.
Start with the root README and [AGENTS.md](AGENTS.md).

1. Describe the QVS behavior or retrieval problem with an original minimal fixture.
2. Add a failing behavioral test for symbols, paths/lines, queries, or map/context output.
3. Change the smallest relevant package. Avoid adding dependencies for simple tasks.
4. For grammar changes, run `make generate`, review corpus diffs, and update expected
   trees only when the parse is correct. Never manually edit generated C/JSON.
5. Run `make check` after `cd tree-sitter-qlik && npm ci && cd ..`.
6. Explain the resulting behavior, test evidence, and any remaining syntax limits.

Use `gofmt` and standard Go tests. Unit tests must be independent and deterministic.
Executable integration tests use the `integration` build tag. Keep source locations
accurate and preserve duplicate definitions. Include budgets and unsupported input
when a change affects retrieval or parsing.

Code and fixtures must be compatible with MIT licensing. Do not port implementation
or tests from the GPL language-reference repository. Record new public language
references in the README, and update third-party notices for bundled code.

The Go module path is `github.com/noesiscarloslisboa/qlik-code-intel`.
For release binaries, embed the version using:

```sh
go build -ldflags '-X github.com/noesiscarloslisboa/qlik-code-intel/internal/cli.Version=v0.1.0' \
  -o bin/qlik-repomap ./cmd/qlik-repomap
```

Follow [the release instructions](docs/releasing.md) before publishing. Parser
changes must preserve valid UTF-8 source bytes; invalid encodings fail explicitly.
