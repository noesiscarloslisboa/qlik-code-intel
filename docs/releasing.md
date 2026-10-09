# Release process

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
v0.1.0 publication contained source only; native distribution is described below.
The npm grammar package remains private;
source queries and the Go binding are included in the repository.

## GitHub Actions binary releases

The `release` workflow builds on native runners: `macos-15` for Apple Silicon,
`macos-15-intel` for Intel Macs, and `windows-2025` for Windows x64. These are
the tested operating-system versions; older OS versions are not established by
these jobs. macOS uses Clang. Windows uses MSYS2 UCRT64 GCC and static compiler
runtime linking. Native jobs run race tests, executable integration tests, vet,
then build and test the extracted archive. Windows checks PE imports and runs
smoke tests without compiler paths. Ubuntu also runs `make check` on Go 1.23.x.

The original v0.1.0 integration harness used an extensionless temporary executable
name, which Go cannot launch on Windows. Its Windows backfill uses a Go test
overlay changing only that `_test.go` filename to end in `.exe`. The checkout,
production sources, and tag remain unchanged. Later tags include the portable
harness directly. Windows archives include the GCC runtime exception and MinGW
runtime notices alongside the other dependency licenses.

Release packaging uses Python 3.12+ and its standard library, with no new CLI
runtime dependencies. Each archive contains only the executable, `LICENSE`,
`THIRD_PARTY_NOTICES.md`, `licenses/`, and `build-info.json` recording the source
commit, target, Go version, and C compiler. Every packaged binary must report the
requested version, pass the strict five-file baseline, support all retrieval
commands, and pass all forty selected public retrieval checks. The workflow
publishes three archives and `SHA256SUMS` only after every required job succeeds.
These are unsigned binaries; code signing and notarization are not configured.
Completed platform runs and validation results are recorded in the
[native release assessment](validation/native-release-assessment.md).

For a new release, commit the reviewed changelog and source, let ordinary CI pass,
then push an annotated version tag on that commit:

```sh
git tag -a v0.1.1 -m 'Release v0.1.1'
git push origin v0.1.1
```

Pushing `v*` tags triggers a build and publication. All builds use the exact
resolved tag commit. A new release is created as a draft, receives the complete
asset set, then becomes public.

To test or backfill an existing tag, run the workflow from the current main
branch, which supplies packaging tooling separately from the tagged source:

```sh
# Build and test only; inspect the run's artifacts before publishing.
gh workflow run release.yml --ref main -f tag=v0.1.0 -f publish=false

# Build, test, and attach binaries to the existing source release.
gh workflow run release.yml --ref main -f tag=v0.1.0 -f publish=true
```

Manual dispatch defaults to no publication. Backfills preserve the tag and
existing release notes. The publish job checks that the tag still identifies
the tested commit, verifies all checksums, and has the workflow's only write
permission. It accepts already-published byte-identical assets; conflicting or
partial existing asset sets fail without replacement. Investigate a failed
upload before deliberately removing incomplete assets and retrying. Builds with
different Go/C toolchains are not promised to produce identical bytes.

To verify downloads, compare the desired archive's hash with its line in
`SHA256SUMS`:

```sh
# macOS
shasum -a 256 qlik-repomap_v0.1.0_darwin_arm64.tar.gz
```

In PowerShell use `Get-FileHash .\qlik-repomap_v0.1.0_windows_amd64.zip -Algorithm SHA256`.

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
