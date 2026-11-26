//go:build cgo

// Tree-sitter syntax parsing is a cgo dependency (github.com/smacker/go-tree-sitter).
// Gating it on the cgo tag is what makes the toolkit buildable with CGO_ENABLED=0
// (D6 rule 5), which is also the mode the pure-go FFI stack requires - see
// MIGRATION.md P6a/P6b. Without cgo, treesitter_stub.go provides a no-op
// TSFindFoldRanges and the code editor simply has no AST fold ranges.

package widgets

import (
	"context"
	"strings"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/javascript"
	python "github.com/smacker/go-tree-sitter/python"
)

var (
	tsParsers   map[string]*sitter.Parser
	tsLanguages map[string]*sitter.Language
	tsOnce      sync.Once
	tsMu        sync.Mutex
)

func getTSParser(lang string) *sitter.Parser {
	tsOnce.Do(initTS)

	tsMu.Lock()
	defer tsMu.Unlock()

	if p, ok := tsParsers[lang]; ok {
		p.SetLanguage(tsLanguages[lang])
		return p
	}
	return nil
}

func initTS() {
	tsParsers = make(map[string]*sitter.Parser)
	tsLanguages = make(map[string]*sitter.Language)

	langs := map[string]*sitter.Language{
		"go":         golang.GetLanguage(),
		"python":     python.GetLanguage(),
		"javascript": javascript.GetLanguage(),
		"typescript": javascript.GetLanguage(),
	}

	for name, lang := range langs {
		p := sitter.NewParser()
		p.SetLanguage(lang)
		tsParsers[name] = p
		tsLanguages[name] = lang
	}
}

// TSFindFoldRanges returns fold ranges using tree-sitter AST
func TSFindFoldRanges(lines []string, lang string) map[int]int {
	result := make(map[int]int)
	src := []byte(strings.Join(lines, "\n"))
	parser := getTSParser(lang)
	if parser == nil {
		return result
	}

	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil || tree == nil {
		return result
	}
	defer tree.Close()

	root := tree.RootNode()
	if root == nil {
		return result
	}

	collectFolds(root, src, result)
	return result
}

func collectFolds(node *sitter.Node, src []byte, result map[int]int) {
	if node == nil {
		return
	}

	nt := node.Type()
	startLine := int(node.StartPoint().Row)
	endLine := int(node.EndPoint().Row)

	// Foldable block types by language
	isFold := false
	switch nt {
	case "block", "body", "program":
		isFold = true
	case "function_body", "class_body", "interface_body":
		isFold = true
	case "compound_statement", "declaration_list":
		isFold = true
	case "argument_block", "statement_block":
		isFold = true
	case "switch_body", "case_alternative":
		isFold = true
	case "object", "array", "pair":
		isFold = true
	}

	// Also fold any braced expression
	if !isFold && (nt == "expression_statement" || nt == "parenthesized_expression") {
		child := node.Child(0)
		if child != nil && (child.Type() == "object" || child.Type() == "array") {
			isFold = true
			node = child
			startLine = int(node.StartPoint().Row)
			endLine = int(node.EndPoint().Row)
		}
	}

	if isFold && endLine > startLine+1 {
		result[startLine] = endLine
	}

	// Recurse into children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			collectFolds(child, src, result)
		}
	}
}
