# Source release process

The first release version is `v0.1.0`. The GitHub repository/module path is
`github.com/noesiscarloslisboa/qlik-code-intel`; the project title remains
qlik-code-intelligence. Prepare and verify changes before tagging and publishing
a new source release. Publication requires an explicit user request.

## Verify from source

Use a clean checkout or a source copy containing only publishable files. Include
the generated parser, handwritten scanner, queries, original fixtures, Go module
files, license, and third-party notices. Exclude private `testdata/real_samples/`,
local binaries, caches, `node_modules/`, and compiled parser libraries.

Ordinary builds need Go 1.23+, CGO, and GCC or Clang. They do not need Node.js,
Tree-sitter CLI, Python, Qlik, or network access after Go modules are downloaded.
The grammar/development checks additionally need Node.js/npm and pinned
Tree-sitter CLI 0.25.10. The optional benchmark needs Python 3.9+ with its standard
library. Run from the project root:

```sh
go mod download
go mod verify
cd tree-sitter-qlik
npm ci
cd ..
make check
make benchmark
```

Build a versioned candidate:

```sh
go build -trimpath -buildvcs=false \
  -ldflags '-X github.com/noesiscarloslisboa/qlik-code-intel/internal/cli.Version=v0.1.0' \
  -o bin/qlik-repomap ./cmd/qlik-repomap
./bin/qlik-repomap version
./bin/qlik-repomap scan --root testdata/repository --strict
./bin/qlik-repomap map --root testdata/repository --tokens 1500
./bin/qlik-repomap find Revenue --kind field --role definition --root testdata/repository
./bin/qlik-repomap deps 'Daily Sales' --direction upstream --root testdata/repository
./bin/qlik-repomap context Revenue --root testdata/repository --tokens 1500
```

Expect version `v0.1.0`, five indexed QVS files, 21 definitions, 26 references,
nine direct dependency edges, and no baseline diagnostics. Check output paths and
numbered source, and retain all unsupported-syntax diagnostics on stderr. Repeat
the tests with the minimum supported Go toolchain before a release.
The version is embedded explicitly; `-buildvcs=false` avoids differing checkout
metadata between a Git checkout and a source archive. Binary reproducibility
still depends on using the same Go/C toolchains and target.

CGO requires a target C toolchain when cross-compiling. Build and test separately
on each supported target; setting `GOOS`/`GOARCH` alone does not establish runtime
support. The [readiness assessment](validation/release-readiness-assessment.md)
lists platforms and toolchains actually executed and links the initial passing
Ubuntu CI run. A configured job alone is not evidence of a completed run. This
source release has no binary distribution automation or Windows validation claim.
The npm grammar package remains private;
source queries and the Go binding are included in the repository.

## Publication checklist

When publication is explicitly requested:

1. Review the complete publishable file inventory and license notices. Keep private
   apps, recovered scripts, detailed manifests, and reports excluded.
2. Commit and push the verified source to the repository, and run the configured
   GitHub Actions checks on that commit.
3. Confirm both minimum and current Go CI results, and update the readiness
   assessment with any newly executed targets.
4. Change the changelog entry from candidate to released with the actual date,
   tag that verified commit `v0.1.0`, and publish source release notes containing
   supported syntax, CGO/build requirements, validation evidence, and known limits.
5. Attach binaries only for targets that were built and tested, with SHA-256
   checksums and the applicable license notices.

Complete recovery of the private QVW scripts still needs comparison with native
QlikView `.qvs` exports or `LoadScript.txt` files. That work does not change this
tool's input contract: it scans UTF-8 `.qvs` text and never executes applications,
includes, SQL connections, or data sources.
