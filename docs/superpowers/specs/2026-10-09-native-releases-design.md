# Native binary releases

Build macOS arm64, macOS amd64, and Windows amd64 on native GitHub-hosted
runners. Tree-sitter uses CGO: macOS uses Clang and Windows uses MSYS2 UCRT64
GCC. Test the Go packages, actual CLI, and packaged executable on each target.
Run grammar reproducibility and minimum-Go checks on Ubuntu before publishing.

Version tags trigger publication. Manual dispatch supports a dry run and an
explicit existing tag for backfilling binaries, initially v0.1.0 at the user's
request. Resolve the tag once and check out that exact source commit for every
job. Keep workflow/packaging tooling in a separate checkout so existing tags can
be built without moving them. Fail if the remote tag changes before publication.

Use Python's standard library for packaging and verification, with no additional
CLI runtime dependencies. Archives contain only the binary, project notices,
dependency/runtime licenses, and build metadata. macOS uses tar.gz; Windows uses
zip. Publish SHA-256 checksums after all targets pass. Test an extracted archive
with version, strict baseline scan, map, find, deps, context, and the public
retrieval benchmark. On Windows remove compiler paths for this smoke test and
reject non-system DLL imports so the downloadable executable needs no compiler.

Use read-only workflow permissions except the final release job. Stage new
releases as drafts until uploads finish; backfill an existing release without
editing its existing notes. Repeated publication accepts byte-identical assets
and rejects conflicting assets. Do not sign binaries without signing credentials.
No parser, retrieval, lineage, MCP, GUI, or database work is included.

Runner/toolchain references:
- https://github.com/actions/runner-images
- https://github.com/msys2/setup-msys2
- https://github.com/actions/upload-artifact

Only actual completed native jobs establish the tested platform claims.
