//go:build !cgo

package widgets

// TSFindFoldRanges is the cgo-free fallback for the tree-sitter implementation in
// treesitter.go (which is behind //go:build cgo). There is no AST without
// tree-sitter, so folding falls back to the editor's non-AST behaviour: no ranges.
//
// This exists so the toolkit builds and links with CGO_ENABLED=0, which is what
// the pure-go FFI stack (pureffi/goffi) requires; a cgo package anywhere in the
// dependency graph makes those binaries fail to link. See MIGRATION.md P6a/P6b.
func TSFindFoldRanges(lines []string, lang string) map[int]int { return nil }
