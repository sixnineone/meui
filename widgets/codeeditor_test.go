package widgets

import (
	"strings"
	"testing"
)

func TestCodeEditorBasic(t *testing.T) {
	editor := NewCodeEditor()
	editor.SetText("package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}")

	text := editor.Text()
	if !strings.Contains(text, "package main") {
		t.Errorf("Text should contain 'package main'")
	}

	if editor.LineCount() != 5 {
		t.Errorf("Expected 5 lines, got %d", editor.LineCount())
	}
}

func TestCodeEditorFilePath(t *testing.T) {
	editor := NewCodeEditor()
	editor.SetFilePath("main.go")
	if editor.Language != "Go" {
		t.Errorf("Expected language Go for .go file, got %s", editor.Language)
	}

	editor2 := NewCodeEditor()
	editor2.SetFilePath("test.py")
	if editor2.Language != "Python" {
		t.Errorf("Expected language Python for .py file, got %s", editor2.Language)
	}
}

func TestCodeEditorUndoRedo(t *testing.T) {
	editor := NewCodeEditor()
	editor.SetText("hello")
	editor.MoveCursorEnd(false)

	editor.InsertText(" world")
	if editor.Text() != "hello world" {
		t.Errorf("InsertText failed: %q", editor.Text())
	}

	if !editor.Undo() {
		t.Errorf("Undo should return true")
	}
	if editor.Text() != "hello" {
		t.Errorf("Undo failed: %q", editor.Text())
	}

	if !editor.Redo() {
		t.Errorf("Redo should return true")
	}
	if editor.Text() != "hello world" {
		t.Errorf("Redo failed: %q", editor.Text())
	}
}

func TestCodeEditorCursorMovement(t *testing.T) {
	editor := NewCodeEditor()
	editor.SetText("abc\ndef\nghi")

	if editor.CursorLine != 0 || editor.CursorCol != 0 {
		t.Errorf("Initial cursor: (%d,%d)", editor.CursorLine, editor.CursorCol)
	}

	editor.MoveCursorDown(false)
	if editor.CursorLine != 1 || editor.CursorCol != 0 {
		t.Errorf("After down: (%d,%d)", editor.CursorLine, editor.CursorCol)
	}

	editor.MoveCursorRight(false)
	editor.MoveCursorRight(false)
	editor.MoveCursorRight(false)
	if editor.CursorCol != 3 {
		t.Errorf("After 3 right: col=%d", editor.CursorCol)
	}

	editor.MoveCursorUp(false)
	if editor.CursorLine != 0 || editor.CursorCol != 3 {
		t.Errorf("After up: (%d,%d)", editor.CursorLine, editor.CursorCol)
	}

	editor.MoveCursorHome(false)
	if editor.CursorCol != 0 {
		t.Errorf("After home: col=%d", editor.CursorCol)
	}

	editor.MoveCursorEnd(false)
	if editor.CursorCol != 3 {
		t.Errorf("After end: col=%d", editor.CursorCol)
	}
}

func TestCodeEditorSelection(t *testing.T) {
	editor := NewCodeEditor()
	editor.SetText("hello world")

	editor.SelectAll()
	if editor.SelectStart >= editor.SelectEnd {
		t.Errorf("SelectAll: start=%d end=%d", editor.SelectStart, editor.SelectEnd)
	}

	// Test that selection clears after cursor movement without shift
	editor.MoveCursorHome(false)
	if editor.SelectStart != editor.SelectEnd {
		t.Errorf("Selection should clear after non-shift movement")
	}

	// Test selection with shift
	editor.MoveCursorEnd(true)
	if editor.SelectStart == editor.SelectEnd {
		t.Errorf("Selection should exist after shift+movement")
	}
}

func TestCodeEditorInsertNewline(t *testing.T) {
	editor := NewCodeEditor()
	editor.SetText("func main() {")

	editor.MoveCursorEnd(false)
	editor.InsertNewline()

	if editor.LineCount() != 2 {
		t.Errorf("Expected 2 lines, got %d", editor.LineCount())
	}
	if editor.CursorLine != 1 {
		t.Errorf("Cursor should be on line 1, got %d", editor.CursorLine)
	}
}

func TestCodeEditorBackspaceDelete(t *testing.T) {
	editor := NewCodeEditor()
	editor.SetText("hello")

	editor.MoveCursorEnd(false)
	editor.Backspace()
	if editor.Text() != "hell" {
		t.Errorf("Backspace failed: %q", editor.Text())
	}

	editor.MoveCursorHome(false)
	editor.MoveCursorRight(false)
	editor.Delete()
	if editor.Text() != "hll" {
		t.Errorf("Delete failed: %q", editor.Text())
	}
}

func TestCodeEditorIndentOutdent(t *testing.T) {
	editor := NewCodeEditor()
	editor.SetText("hello")

	editor.Indent()
	parts := strings.Split(editor.Text(), "\t")
	if len(parts) < 2 && editor.Text() != "\thello" {
		t.Errorf("Indent should insert tab")
	}

	editor2 := NewCodeEditor()
	editor2.SetText("hello")
	editor2.Indent()
	editor2.Outdent()
	if editor2.Text() != "hello" {
		t.Errorf("Outdent after indent should restore: %q", editor2.Text())
	}
}

func TestCodeEditorFind(t *testing.T) {
	editor := NewCodeEditor()
	editor.SetText("hello world hello")

	editor.Find("hello")
	if len(editor.searchMatches) == 0 {
		t.Errorf("Should find 'hello' matches")
	}

	editor.FindNext()
}

func TestCodeEditorGoToLine(t *testing.T) {
	editor := NewCodeEditor()
	editor.SetText("line1\nline2\nline3")

	editor.GoToLine(1)
	if editor.CursorLine != 1 || editor.CursorCol != 0 {
		t.Errorf("GoToLine 1: (%d,%d)", editor.CursorLine, editor.CursorCol)
	}
}

func TestCodeEditorFoldOperations(t *testing.T) {
	editor := NewCodeEditor()
	editor.SetText("func main() {\n\tfmt.Println(\"hello\")\n}")

	editor.ToggleFold(0)
	editor.FoldAll()
	editor.UnfoldAll()
}

func TestCodeEditorDiagnostics(t *testing.T) {
	editor := NewCodeEditor()
	editor.SetDiagnostics(nil)

	diags := []Diagnostic{
		{Line: 0, Col: 0, EndLine: 0, EndCol: 5, Severity: 1, Message: "error", Source: "test"},
		{Line: 1, Col: 2, EndLine: 1, EndCol: 6, Severity: 2, Message: "warning", Source: "test"},
	}

	editor.SetDiagnostics(diags)
	editor.SetDiagnostics(nil)
}

func TestCodeEditorTheme(t *testing.T) {
	theme := DefaultEditorTheme()
	if theme == nil || theme.Background == 0 {
		t.Errorf("Invalid editor theme")
	}

	st := DefaultSyntaxTheme()
	if st == nil || st.Default.Color == 0 {
		t.Errorf("Invalid syntax theme")
	}
}

func TestGoTokenizer(t *testing.T) {
	tokenizer := &GoTokenizer{}
	if tokenizer.LanguageName() != "Go" {
		t.Errorf("LanguageName: %s", tokenizer.LanguageName())
	}

	tokens, state := tokenizer.Tokenize("package main", 0, nil)
	if len(tokens) == 0 {
		t.Errorf("Expected tokens, got none")
	}

	hasKeyword := false
	for _, tok := range tokens {
		if tok.Type == TokenKeyword {
			hasKeyword = true
			break
		}
	}
	if !hasKeyword {
		t.Errorf("Expected 'package' to be keyword")
	}

	tokens2, _ := tokenizer.Tokenize("// comment", 0, state)
	if len(tokens2) > 0 && tokens2[0].Type != TokenComment {
		t.Errorf("Comment should be TokenComment")
	}
}

func TestPythonTokenizer(t *testing.T) {
	tokenizer := &PythonTokenizer{}
	if tokenizer.LanguageName() != "Python" {
		t.Errorf("LanguageName: %s", tokenizer.LanguageName())
	}

	tokens, _ := tokenizer.Tokenize("def hello():", 0, nil)
	hasKeyword := false
	for _, tok := range tokens {
		if tok.Type == TokenKeyword {
			hasKeyword = true
			break
		}
	}
	if !hasKeyword {
		t.Errorf("Expected 'def' to be keyword")
	}
}

func TestJavaScriptTokenizer(t *testing.T) {
	tokenizer := &JavaScriptTokenizer{}
	if tokenizer.LanguageName() != "JavaScript" {
		t.Errorf("LanguageName: %s", tokenizer.LanguageName())
	}

	tokens, _ := tokenizer.Tokenize("function hello() {", 0, nil)
	hasKeyword := false
	for _, tok := range tokens {
		if tok.Type == TokenKeyword && tok.End-tok.Start >= 8 {
			hasKeyword = true
			break
		}
	}
	if !hasKeyword {
		t.Errorf("Expected keyword in JS code")
	}
}

func TestMultiLangTokenizer(t *testing.T) {
	m := NewMultiLangTokenizer()

	tok := m.GetTokenizer("main.go", "")
	if tok == nil || tok.LanguageName() != "Go" {
		t.Errorf("Expected Go tokenizer for .go")
	}

	tok = m.GetTokenizer("test.py", "")
	if tok == nil || tok.LanguageName() != "Python" {
		t.Errorf("Expected Python tokenizer for .py")
	}

	tok = m.GetTokenizer("app.js", "")
	if tok == nil || tok.LanguageName() != "JavaScript" {
		t.Errorf("Expected JS tokenizer for .js, got %s", tok.LanguageName())
	}

	tok = m.GetTokenizer("unknown.xyz", "go")
	if tok == nil || tok.LanguageName() != "Go" {
		t.Errorf("Expected Go tokenizer for explicit language")
	}
}

func TestLangIDDetection(t *testing.T) {
	tests := []struct {
		file     string
		expected string
	}{
		{"main.go", "go"},
		{"test.py", "python"},
		{"app.js", "javascript"},
		{"index.ts", "typescript"},
		{"style.css", "css"},
		{"data.json", "json"},
		{"readme.md", "markdown"},
		{"Dockerfile", "plaintext"},
	}

	for _, tt := range tests {
		result := LangIDForFile(tt.file)
		if result != tt.expected {
			t.Errorf("LangIDForFile(%s) = %s, want %s", tt.file, result, tt.expected)
		}
	}
}

func TestServerForLanguage(t *testing.T) {
	if ServerForLanguage("go") != "gopls" {
		t.Errorf("Expected gopls for Go")
	}
	if ServerForLanguage("python") != "pylsp" {
		t.Errorf("Expected pylsp for Python")
	}
	if ServerForLanguage("unknown") != "" {
		t.Errorf("Expected empty for unknown language")
	}
}

func TestLSPClientCreate(t *testing.T) {
	lsp := NewLSPClient("gopls", "go", "file:///test")
	if lsp == nil {
		t.Errorf("LSP client should not be nil")
	}
}

func TestCodeEditorSetBoundProperty(t *testing.T) {
	editor := NewCodeEditor()

	editor.SetBoundProperty("text", "new text")
	if editor.Text() != "new text" {
		t.Errorf("SetBoundProperty text: %q", editor.Text())
	}

	editor.SetBoundProperty("readOnly", true)
	if !editor.ReadOnly {
		t.Errorf("SetBoundProperty readOnly")
	}

	editor.SetBoundProperty("fontSize", float64(16))
	if editor.FontSize != 16 {
		t.Errorf("SetBoundProperty fontSize: %f", editor.FontSize)
	}

	editor.SetBoundProperty("showLineNumbers", false)
	if editor.ShowLineNumbers {
		t.Errorf("SetBoundProperty showLineNumbers")
	}
}

func TestCodeEditorSelectWithShift(t *testing.T) {
	editor := NewCodeEditor()
	editor.SetText("hello world")

	editor.MoveCursorRight(true)
	selStart := editor.SelectStart
	selEnd := editor.SelectEnd
	if selStart == selEnd {
		t.Errorf("MoveCursorRight with shift should create selection (got start=%d end=%d)", selStart, selEnd)
	}

	editor.MoveCursorRight(true)
	editor.MoveCursorRight(true)

	// After moving right 3 times with shift, selection should be different from single char
	if editor.SelectEnd-editor.SelectStart <= 1 {
		t.Errorf("Selection should be bigger after 3 shift-right moves")
	}

	editor.MoveCursorLeft(false)
	if editor.SelectStart != editor.SelectEnd {
		t.Errorf("MoveCursorLeft without shift should clear selection, got start=%d end=%d", editor.SelectStart, editor.SelectEnd)
	}
}

func TestCodeEditorToggleComment(t *testing.T) {
	editor := NewCodeEditor()
	editor.SetText("hello")

	editor.ToggleLineComment()
	if !strings.Contains(editor.Text(), "//") && !strings.Contains(editor.Text(), "#") {
		t.Errorf("ToggleLineComment should add comment prefix")
	}
}

func TestCodeEditorCompletion(t *testing.T) {
	editor := NewCodeEditor()
	editor.SetText("")

	items := []CompletionItem{
		{Label: "fmt", Detail: "package", InsertText: "fmt"},
		{Label: "func", Detail: "keyword", InsertText: "func"},
		{Label: "for", Detail: "keyword", InsertText: "for"},
	}

	editor.ShowCompletion(items, "", 100, 100)
	if !editor.completionVisible {
		t.Errorf("Completion should be visible")
	}

	editor.HideCompletion()
	if editor.completionVisible {
		t.Errorf("Completion should be hidden")
	}
}

func TestCodeEditorHover(t *testing.T) {
	editor := NewCodeEditor()

	editor.ShowHover("test hover", 100, 100)
	if !editor.hoverVisible {
		t.Errorf("Hover should be visible")
	}

	editor.HideHover()
	if editor.hoverVisible {
		t.Errorf("Hover should be hidden")
	}
}

func TestCodeEditorParameterHints(t *testing.T) {
	editor := NewCodeEditor()

	hints := []ParameterHint{
		{Label: "func(a, b int)", Documentation: "Test function"},
	}

	editor.ShowParameterHints(hints)
	if !editor.parameterHintVisible {
		t.Errorf("Parameter hints should be visible")
	}

	editor.HideParameterHints()
	if editor.parameterHintVisible {
		t.Errorf("Parameter hints should be hidden")
	}
}
