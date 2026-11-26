package behaviour

import (
	"strings"
	"testing"

	"github.com/sixnineone/meui/platform"
)

const textQML = `
Window {
    width: 400
    height: 300
    VStack {
        margins: 12
        spacing: 10
        TextField { id: field; width: 200 }
        TextField { id: field2; width: 200; text: "pre-filled" }
        TextArea { id: area; width: 300; height: 120; text: "alpha beta gamma" }
        TextField { id: pass; width: 200; echoMode: "password" }
    }
}
`

func TestTextFieldFocusAndType(t *testing.T) {
	h := Load(t, textQML)
	tf := h.TextField("field")
	cx, cy := Center(tf)
	h.Click(cx, cy)
	if !tf.Focused {
		t.Fatal("click did not focus the text field")
	}
	h.Type("hello")
	if tf.Text != "hello" {
		t.Fatalf("typed text: got %q, want %q", tf.Text, "hello")
	}
	h.KeyDown(platform.KeyBackspace, 0)
	if tf.Text != "hell" {
		t.Fatalf("after backspace: got %q, want %q", tf.Text, "hell")
	}
}

func TestTextFieldCursorArrows(t *testing.T) {
	h := Load(t, textQML)
	tf := h.TextField("field")
	cx, cy := Center(tf)
	h.Click(cx, cy)
	h.Type("abc")
	if tf.CursorPos != 3 {
		t.Fatalf("cursor after typing: got %d, want 3", tf.CursorPos)
	}
	h.KeyDown(platform.KeyLeft, 0)
	if tf.CursorPos != 2 {
		t.Fatalf("cursor after Left: got %d, want 2", tf.CursorPos)
	}
	h.KeyDown(platform.KeyHome, 0)
	if tf.CursorPos != 0 {
		t.Fatalf("cursor after Home: got %d, want 0", tf.CursorPos)
	}
	h.KeyDown(platform.KeyEnd, 0)
	if tf.CursorPos != 3 {
		t.Fatalf("cursor after End: got %d, want 3", tf.CursorPos)
	}
}

func TestTextFieldDoubleClickSelectsWord(t *testing.T) {
	h := Load(t, textQML)
	tf := h.TextField("field2") // "pre-filled"
	cx, cy := Center(tf)
	h.DblClick(cx, cy)
	if got := tf.GetSelectedText(); got == "" {
		t.Fatal("double-click selected nothing")
	}
}

func TestTextFieldSelectAll(t *testing.T) {
	h := Load(t, textQML)
	tf := h.TextField("field2")
	cx, cy := Center(tf)
	h.Click(cx, cy)
	tf.SelectAll()
	if got := tf.GetSelectedText(); got != "pre-filled" {
		t.Fatalf("select-all: got %q, want %q", got, "pre-filled")
	}
}

func TestTextFieldTypeReplacesSelection(t *testing.T) {
	h := Load(t, textQML)
	tf := h.TextField("field2")
	cx, cy := Center(tf)
	h.Click(cx, cy)
	tf.SelectAll()
	h.Type("X")
	if tf.Text != "X" {
		t.Fatalf("typing over selection: got %q, want %q", tf.Text, "X")
	}
}

func TestTextFieldDragSelection(t *testing.T) {
	h := Load(t, textQML)
	tf := h.TextField("field2") // "pre-filled"
	f := tf.FrameRect()
	// Drag from before the first char to past the last char: the whole
	// text must end up selected (charIndexAtX uses nearest-boundary
	// semantics, so intermediate positions are font-metric dependent).
	h.Drag(f.X+4, f.Y+f.Height/2, f.X+f.Width-8, f.Y+f.Height/2)
	if got := tf.GetSelectedText(); got != "pre-filled" {
		t.Fatalf("drag selection: got %q, want %q", got, "pre-filled")
	}
}

func TestTextFieldShiftArrowSelection(t *testing.T) {
	h := Load(t, textQML)
	tf := h.TextField("field2")
	cx, cy := Center(tf)
	h.Click(cx, cy)
	tf.SetText("")
	tf.CursorPos = 0
	h.Type("abc")
	h.KeyDown(platform.KeyLeft, platform.ModShift)
	h.KeyDown(platform.KeyLeft, platform.ModShift)
	if got := tf.GetSelectedText(); got != "bc" {
		t.Fatalf("shift-arrow selection: got %q, want %q", got, "bc")
	}
}

func TestPasswordFieldTyping(t *testing.T) {
	h := Load(t, textQML)
	tf := h.TextField("pass")
	cx, cy := Center(tf)
	h.Click(cx, cy)
	h.Type("secret")
	if tf.Text != "secret" {
		t.Fatalf("password text: got %q", tf.Text)
	}
}

func TestTextAreaEdit(t *testing.T) {
	h := Load(t, textQML)
	ta := h.TextArea("area")
	ta.SetText("start: ")
	cx, cy := Center(ta)
	h.Click(cx, cy)
	h.KeyDown(platform.KeyEnd, 0)
	h.Type("end")
	if got := ta.ValueText(); !strings.HasSuffix(got, "end") {
		t.Fatalf("textarea edit: got %q, want suffix %q", got, "end")
	}
}
