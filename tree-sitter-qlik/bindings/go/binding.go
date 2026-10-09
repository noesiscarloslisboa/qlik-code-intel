// Package tree_sitter_qlik exposes the independently generated Qlik language.
package tree_sitter_qlik

// #cgo CFLAGS: -std=c11 -I../../src
// #include "../../src/parser.c"
// #include "../../src/scanner.c"
import "C"

import "unsafe"

// Language returns the Tree-sitter Qlik language for use with go-tree-sitter.
func Language() unsafe.Pointer {
	return unsafe.Pointer(C.tree_sitter_qlik())
}
