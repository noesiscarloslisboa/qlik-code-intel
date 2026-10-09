# Native binary release validation — 2026-10-09

GitHub Actions built and tested macOS arm64, macOS amd64, and Windows amd64
binaries from the unchanged `v0.1.0` source commit
`c03711dbdc3b2908eacfcc594dde941ca42b4236`. Release automation runs from
`22e683e7cf98d52cd764789fc5273c543b687014` in a separate checkout.

The [successful dry run](https://github.com/noesiscarloslisboa/qlik-code-intel/actions/runs/37908679939)
tested every native archive before the
[publication run](https://github.com/noesiscarloslisboa/qlik-code-intel/actions/runs/37909145005)
repeated all checks and uploaded three archives plus `SHA256SUMS` to the existing
[v0.1.0 release](https://github.com/noesiscarloslisboa/qlik-code-intel/releases/tag/v0.1.0).
The existing tag was not moved. All required jobs, including publication, passed.

| Target | Native runner | Go | C compiler | Result |
| --- | --- | --- | --- | --- |
| darwin/arm64 | macos-15 | 1.27.2 | Apple Clang 17.0.0 | Pass |
| darwin/amd64 | macos-15-intel | 1.27.2 | Apple Clang 17.0.0 | Pass |
| windows/amd64 | windows-2025 | 1.27.2 | MSYS2 UCRT64 GCC 16.2.0 | Pass |

Every native job ran uncached race-enabled Go tests, the executable integration
suite, and vet. Windows used an external Go test overlay to add `.exe` to the
original integration harness's temporary executable name. This changes only test
input; `git diff --exit-code` confirms the original checkout remains intact.
The portable test harness is fixed on main. Production Go, grammar, generated
parser, handwritten scanner, queries, and module dependencies remain unchanged.

Each job built a versioned binary, packaged an explicit file allowlist, extracted
the archive, and checked version, strict baseline scan, map, find, deps, and
context. Each extracted binary also passed all forty selected public retrieval
checks across the reviewed 256/512/1024-byte budgets. The strict baseline remains
five files, 21 definitions, 26 references, nine direct edges, and zero diagnostics.
The Windows binary passed DLL inspection and smoke tests with only the Windows
system directory on PATH, without compiler paths.

Ubuntu validated the exact tagged source with Go 1.23.x, `npm ci`, and `make check`,
including twenty grammar corpus cases and reproducible generated artifacts.
Nine Python packaging/benchmark self-tests also passed. Workflow lint passed with
actionlint 1.7.12; local Go race tests, integration, vet, build, and the downloaded
Apple Silicon executable passed as well.

macOS archives contain eight files; Windows contains fourteen, including GCC's
Runtime Library Exception, MinGW CRT notices, and winpthreads notices. All include
project/Go/module/Unicode licenses and build metadata identifying the original
source commit. Archives exclude fixtures, private applications, recovered
scripts, caches, and module source trees. SHA-256 checksums are verified before
publication and independently against the public downloads afterward.
All three published archive hashes also match the independently downloaded dry-run
artifacts; this is an observed repeat build with these toolchains, not a promise
of identical bytes across different toolchain versions.

Binaries are unsigned. Older OS versions, Windows ARM64, signing/notarization,
and cross-compilation were not tested. Selected retrieval checks are not exhaustive
recall, and no Qlik application or script was executed. Existing grammar and
private-script recovery limits remain as documented in the README.
