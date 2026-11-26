package behaviour

import (
	"strings"
	"testing"

	"github.com/sixnineone/meui/platform"
)

func TestCopyPasteRoundTrip(t *testing.T) {
	h := Load(t, textQML)
	tf := h.TextField("field2") // "pre-filled"
	cx, cy := Center(tf)
	h.Click(cx, cy)
	tf.SelectAll()
	tf.Copy()
	if got := platform.GlobalClipboard.GetText(); got != "pre-filled" {
		t.Fatalf("copy: clipboard = %q", got)
	}
	tf.SetText("")
	tf.CursorPos = 0
	tf.Paste()
	if tf.Text != "pre-filled" {
		t.Fatalf("paste: got %q, want %q", tf.Text, "pre-filled")
	}
}

func TestCutRemovesAndCopies(t *testing.T) {
	h := Load(t, textQML)
	tf := h.TextField("field2")
	cx, cy := Center(tf)
	h.Click(cx, cy)
	tf.SelectAll()
	tf.Cut()
	if tf.Text != "" {
		t.Fatalf("cut left text: %q", tf.Text)
	}
	if got := platform.GlobalClipboard.GetText(); got != "pre-filled" {
		t.Fatalf("cut clipboard: got %q", got)
	}
	tf.CursorPos = 0
	tf.Paste()
	if tf.Text != "pre-filled" {
		t.Fatalf("paste after cut: %q", tf.Text)
	}
}

// Paste at a cursor position mid-text inserts there, not at the end.
func TestPasteAtCursorMidText(t *testing.T) {
	h := Load(t, textQML)
	tf := h.TextField("field2") // "pre-filled"
	cx, cy := Center(tf)
	h.Click(cx, cy)
	tf.CursorPos = 3 // after "pre"
	tf.SetSelection(3, 3)
	platform.GlobalClipboard.SetText("-MID-")
	tf.Paste()
	// "pre-filled" contains a dash at index 3; inserting at 3 gives
	// pre + -MID- + -filled.
	if tf.Text != "pre-MID--filled" {
		t.Fatalf("mid paste: got %q", tf.Text)
	}
}

// Insert characters mid-text via typed runes.
func TestInsertMidText(t *testing.T) {
	h := Load(t, textQML)
	tf := h.TextField("field2")
	cx, cy := Center(tf)
	h.Click(cx, cy)
	tf.CursorPos = 3
	tf.SetSelection(3, 3)
	h.Type("-X-")
	if tf.Text != "pre-X--filled" {
		t.Fatalf("mid insert: got %q", tf.Text)
	}
}

func TestDeleteForwardAndBackward(t *testing.T) {
	h := Load(t, textQML)
	tf := h.TextField("field")
	cx, cy := Center(tf)
	h.Click(cx, cy)
	h.Type("abc")
	tf.CursorPos = 0
	tf.SetSelection(0, 0)
	h.KeyDown(platform.KeyDelete, 0) // delete forward at pos 0 removes 'a'
	if tf.Text != "bc" {
		t.Fatalf("delete forward: got %q", tf.Text)
	}
	h.KeyDown(platform.KeyBackspace, 0) // at pos 0: no-op
	if tf.Text != "bc" {
		t.Fatalf("backspace at 0 changed text: %q", tf.Text)
	}
	h.KeyDown(platform.KeyEnd, 0)
	h.KeyDown(platform.KeyBackspace, 0) // removes 'c'
	if tf.Text != "b" {
		t.Fatalf("backspace: got %q", tf.Text)
	}
}

func TestBackspaceDeletesSelection(t *testing.T) {
	h := Load(t, textQML)
	tf := h.TextField("field2")
	cx, cy := Center(tf)
	h.Click(cx, cy)
	tf.SelectAll()
	h.KeyDown(platform.KeyBackspace, 0)
	if tf.Text != "" {
		t.Fatalf("backspace with selection: got %q", tf.Text)
	}
}

func TestDeleteDeletesSelection(t *testing.T) {
	h := Load(t, textQML)
	tf := h.TextField("field2")
	cx, cy := Center(tf)
	h.Click(cx, cy)
	tf.SelectAll()
	h.KeyDown(platform.KeyDelete, 0)
	if tf.Text != "" {
		t.Fatalf("delete with selection: got %q", tf.Text)
	}
}

func TestUndoRedo(t *testing.T) {
	h := Load(t, textQML)
	tf := h.TextField("field")
	cx, cy := Center(tf)
	h.Click(cx, cy)
	h.Type("abc")
	// Undo granularity is per-insert: three undos empty the field.
	for i := 0; i < 3; i++ {
		h.KeyDown(platform.KeyZ, platform.ModCtrl)
	}
	if tf.Text != "" {
		t.Fatalf("undo: got %q, want empty", tf.Text)
	}
	for i := 0; i < 3; i++ {
		h.KeyDown(platform.KeyZ, platform.ModCtrl|platform.ModShift)
	}
	if tf.Text != "abc" {
		t.Fatalf("redo: got %q", tf.Text)
	}
}

func TestTextFieldSubmit(t *testing.T) {
	h := Load(t, textQML)
	// The QML loader has no submit binding — attach directly.
	tf := h.TextField("field")
	var submitted string
	tf.SetOnSubmit(func(s string) { submitted = s })
	cx, cy := Center(tf)
	h.Click(cx, cy)
	h.Type("done")
	h.KeyDown(platform.KeyReturn, 0)
	if submitted != "done" {
		t.Fatalf("submit: got %q", submitted)
	}
}

func TestTextFieldReadOnly(t *testing.T) {
	h := Load(t, textQML)
	tf := h.TextField("field")
	tf.ReadOnly = true
	cx, cy := Center(tf)
	h.Click(cx, cy)
	h.Type("no")
	if tf.Text != "" {
		t.Fatalf("readonly field accepted input: %q", tf.Text)
	}
	// Selection must still work for copy.
	tf.SetText("copyme")
	tf.SelectAll()
	if got := tf.GetSelectedText(); got != "copyme" {
		t.Fatalf("readonly selection: %q", got)
	}
}

func TestTextAreaNewline(t *testing.T) {
	h := Load(t, textQML)
	ta := h.TextArea("area")
	ta.SetText("one")
	cx, cy := Center(ta)
	h.Click(cx, cy)
	h.KeyDown(platform.KeyEnd, 0)
	h.Type("\n") // via runes
	h.Type("two")
	if ta.ValueText() != "one\ntwo" {
		t.Fatalf("textarea newline: got %q", ta.ValueText())
	}
	h.Click(cx, cy)
	h.KeyDown(platform.KeyEnd, 0)
	h.KeyDown(platform.KeyReturn, 0) // via Enter key
	h.Type("three")
	if !strings.Contains(ta.ValueText(), "\nthree") {
		t.Fatalf("textarea enter key: got %q", ta.ValueText())
	}
}

func TestTextAreaCopyPaste(t *testing.T) {
	h := Load(t, textQML)
	ta := h.TextArea("area")
	ta.SetText("line-one")
	cx, cy := Center(ta)
	h.Click(cx, cy)
	ta.SelectAll()
	ta.Copy()
	if got := platform.GlobalClipboard.GetText(); got != "line-one" {
		t.Fatalf("textarea copy: clipboard %q", got)
	}
	ta.SetText("") // cursor resets to offset 0 on empty content
	ta.Paste()
	if ta.ValueText() != "line-one" {
		t.Fatalf("textarea paste: %q", ta.ValueText())
	}
}
