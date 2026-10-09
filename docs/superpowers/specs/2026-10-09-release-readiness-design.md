# Release readiness design

The user approved a final code review, clean-checkout verification, and release
preparation, then selected `qlik-code-intel` as the public repository name.
The authenticated GitHub account is `noesiscarloslisboa`; use
`github.com/noesiscarloslisboa/qlik-code-intel` for the module and imports.
No repository is currently visible at that location. Prepare release material
locally without creating a remote repository, committing, tagging, or publishing.

Review scanner recovery, extraction, ranking/budgets, source locations, CLI I/O,
dependency constraints, and development instructions. Preserve the documented
milestone scope. Repairs should address demonstrated failures with original
regressions, rather than adding syntax or dependencies.

A probe found that a non-UTF-8 field name produces invalid map/context bytes
with exit zero and no diagnostics; JSON silently substitutes a replacement
character. Reject non-UTF-8 source at the Parse boundary with a path-bearing
error, before Tree-sitter extraction. Keep valid UTF-8 source, BOM, CRLF, and byte
locations intact. Automatic transcoding would require guessing legacy encodings;
replacement decoding would lose source spelling. Explicit failure preserves the
source contract. Also verify all command help output propagates writer failures,
using the same exit-one behavior as ordinary output.

The repository has no first commit. Create a fresh temporary source tree from
Git's non-ignored file inventory, excluding `.git`, private samples, local caches,
and built binaries. Verify hashes and absence of generated local dependencies,
then build/test from an empty Go build cache, exercise documented commands, and
install pinned grammar tooling via `npm ci`. Check the minimum Go 1.23 toolchain
as well as the installed Go version where available. Platform claims must be
limited to platforms actually executed; configured CI is not a remote test result.

Record checks, corrections, and remaining limitations in a public assessment.
Add a concise candidate changelog and source-release instructions with the actual
module path and version embedding command. Keep all original QVW data and detailed
private reports ignored. Authoritative export comparison remains separate.
