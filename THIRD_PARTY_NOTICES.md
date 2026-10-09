# Third-party notices

## Tree-sitter generated support headers

`tree-sitter-qlik/src/tree_sitter/{parser,alloc,array}.h` are supplied by
Tree-sitter CLI 0.25.10 when generating a parser. Upstream:
[Tree-sitter 0.25.10](https://github.com/tree-sitter/tree-sitter/tree/v0.25.10).

The MIT License (MIT)

Copyright (c) 2018-2024 Max Brunsfeld

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

## Go runtime dependencies

The Go module resolves `github.com/tree-sitter/go-tree-sitter` v0.25.0 (MIT),
which bundles its Tree-sitter C runtime and uses `github.com/mattn/go-pointer`
v0.0.1 (MIT). Their original license files are included in their downloaded
modules. `go.sum` records resolved module checksums; those dependency sources
are not vendored in this repository.

Binary release archives additionally include the Go toolchain's original license,
both Go module licenses, and the bundled Unicode/ICU notice under `licenses/`.
Windows archives also retain GCC's license and Runtime Library Exception, plus
the MinGW CRT and winpthreads notices supplied with the selected UCRT64 toolchain.
