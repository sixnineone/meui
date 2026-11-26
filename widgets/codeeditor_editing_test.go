package widgets

// codeeditor_editing_test.go — the code editor's half of the editing gestures the text widgets got
// in text_editing_test.go: rune-safe steps and deletes, collapse direction, a goal column, and word
// motion that crosses lines.
//
// Columns stay BYTE offsets — that is the editor's model everywhere, including the click mapping —
// but a step never lands inside a rune, and nothing counts a byte as a character.

import "testing"

func newTestEditor(lines ...string) *CodeEditor {
	e := NewCodeEditor()
	e.Lines = lines
	e.SelectStart, e.SelectEnd = 0, 0
	e.selectionAnchor = 0
	return e
}

func TestCodeEditorStepsByRune(t *testing.T) {
	e := newTestEditor("aé")
	e.CursorLine, e.CursorCol = 0, len("aé") // byte column, at the end
	e.Backspace()
	if e.Lines[0] != "a" {
		t.Errorf("Backspace left %q, want %q — it split the rune", e.Lines[0], "a")
	}
	if e.CursorCol != 1 {
		t.Errorf("the caret is at byte %d, want 1", e.CursorCol)
	}

	e = newTestEditor("éa")
	e.CursorLine, e.CursorCol = 0, 0
	e.Delete()
	if e.Lines[0] != "a" {
		t.Errorf("Delete left %q, want %q", e.Lines[0], "a")
	}

	// One cursor step is one RUNE, not one byte.
	e = newTestEditor("aé")
	e.CursorLine, e.CursorCol = 0, len("aé")
	e.MoveCursorLeft(false)
	if e.CursorCol != 1 {
		t.Errorf("Left stepped to byte %d, want the start of the rune (1)", e.CursorCol)
	}
	e.MoveCursorRight(false)
	if e.CursorCol != len("aé") {
		t.Errorf("Right stepped to byte %d, want the end of the text", e.CursorCol)
	}
}

func TestCodeEditorCollapsesTowardsTheArrow(t *testing.T) {
	e := newTestEditor("abcdef")
	e.SelectStart, e.SelectEnd = 1, 4
	e.MoveCursorLeft(false)
	if e.CursorCol != 1 || e.SelectStart != e.SelectEnd {
		t.Errorf("Left left the caret at %d with selection %d-%d, want the START (1) collapsed",
			e.CursorCol, e.SelectStart, e.SelectEnd)
	}

	e.SelectStart, e.SelectEnd = 1, 4
	e.MoveCursorRight(false)
	if e.CursorCol != 4 || e.SelectStart != e.SelectEnd {
		t.Errorf("Right left the caret at %d with selection %d-%d, want the END (4) collapsed",
			e.CursorCol, e.SelectStart, e.SelectEnd)
	}
}

func TestCodeEditorKeepsTheGoalColumn(t *testing.T) {
	e := newTestEditor("abcdef", "ab", "abcdef")
	e.CursorLine, e.CursorCol = 0, 4

	e.MoveCursorDown(false)
	if e.CursorLine != 1 || e.CursorCol != 2 {
		t.Fatalf("Down landed at %d:%d, want 1:2 (clamped)", e.CursorLine, e.CursorCol)
	}
	e.MoveCursorDown(false)
	if e.CursorLine != 2 || e.CursorCol != 4 {
		t.Errorf("the second Down landed at %d:%d, want the ORIGINAL column 4", e.CursorLine, e.CursorCol)
	}
	e.MoveCursorUp(false)
	if e.CursorLine != 1 || e.CursorCol != 2 {
		t.Errorf("Up landed at %d:%d, want 1:2", e.CursorLine, e.CursorCol)
	}
	e.MoveCursorUp(false)
	if e.CursorLine != 0 || e.CursorCol != 4 {
		t.Errorf("the second Up landed at %d:%d, want the ORIGINAL column 4", e.CursorLine, e.CursorCol)
	}
}

func TestCodeEditorWordMotionCrossesLines(t *testing.T) {
	e := newTestEditor("foo.bar", "baz")
	e.CursorLine, e.CursorCol = 1, len("baz")

	e.MoveCursorWordLeft(false)
	if e.CursorLine != 1 || e.CursorCol != 0 {
		t.Errorf("Ctrl+Left landed at %d:%d, want 1:0 (start of \"baz\")", e.CursorLine, e.CursorCol)
	}
	e.MoveCursorWordLeft(false)
	if e.CursorLine != 0 || e.CursorCol != 4 {
		t.Errorf("Ctrl+Left landed at %d:%d, want 0:4 (start of \"bar\", past the dot)", e.CursorLine, e.CursorCol)
	}
	e.MoveCursorWordLeft(false)
	if e.CursorLine != 0 || e.CursorCol != 3 {
		t.Errorf("Ctrl+Left landed at %d:%d, want 0:3 (the dot is its own run)", e.CursorLine, e.CursorCol)
	}
}

func TestCodeEditorDeletesByWord(t *testing.T) {
	e := newTestEditor("one two")
	e.CursorLine, e.CursorCol = 0, len("one two")
	e.BackspaceWord()
	if e.Lines[0] != "one " {
		t.Errorf("BackspaceWord left %q, want %q", e.Lines[0], "one ")
	}

	e = newTestEditor("one two")
	e.CursorLine, e.CursorCol = 0, 0
	e.DeleteWord()
	if e.Lines[0] != "two" {
		t.Errorf("DeleteWord left %q, want %q", e.Lines[0], "two")
	}
}
