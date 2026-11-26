package behaviour

import (
	"testing"

	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/widgets"
)

// These tests pin the two halves of on-screen-keyboard support, headlessly:
// the toolkit reports whether a text-editing widget has focus and where it is,
// and the backend turns that into the OS call (SDL_StartTextInput plus the input
// area). Android needs both — the area is what stops the IME refusing to serve a
// zero-sized edit view, and the flag is what raises the keyboard at all — so a
// regression in either half shows up as a keyboard that never appears, which is
// easy to blame on the platform.

// TestTextInputAreaFollowsFocus: nothing focused means off, a focused TextField
// means on with a real rect, and moving focus to a Button means off again.
//
// The last one is the one that matters most in practice: text input left on
// after focus leaves a field is a keyboard that never goes away.
func TestTextInputAreaFollowsFocus(t *testing.T) {
	h := Load(t, `
Window {
  width: 400
  height: 300
  VStack {
    TextField { id: field; width: 380; height: 40 }
    Button { id: button; text: "x"; width: 380; height: 40 }
  }
}`)

	if active, _ := h.Win.TextInputArea(); active {
		t.Error("nothing has focus, so text input must be off")
	}

	field := h.Widget("field").(*widgets.TextField)
	fr := field.FrameRect()
	h.Click(fr.X+fr.Width/2, fr.Y+fr.Height/2)

	active, area := h.Win.TextInputArea()
	if !active {
		t.Fatal("a focused TextField must turn text input on, or no keyboard ever appears")
	}
	if area.Width <= 0 || area.Height <= 0 {
		t.Errorf("area = %+v, want a non-degenerate rect: Android's InputMethodManager "+
			"refuses to serve the IME for a 0x0 edit view", area)
	}
	if !field.Focused {
		t.Error("focus reached the window but not the widget, so it would ignore what is typed")
	}

	btn := h.Widget("button").(*widgets.Button)
	br := btn.FrameRect()
	h.Click(br.X+br.Width/2, br.Y+br.Height/2)

	if active, _ := h.Win.TextInputArea(); active {
		t.Error("focus moved to a Button, so text input must be off — otherwise the " +
			"keyboard never goes away")
	}
	if field.Focused {
		t.Error("the TextField still reports Focused after focus moved away")
	}
}

// TestTypingReachesOnlyTheFocusedField guards a real hazard in the content tree:
// a Panel with no focus of its own broadcasts OnCharacter to EVERY child that
// accepts one. Today that is harmless because a TextField self-gates on its own
// Focused flag — but "harmless because the other end checks" is exactly the
// arrangement that breaks silently when a new widget forgets to check, and the
// symptom would be one typed character landing in two fields.
func TestTypingReachesOnlyTheFocusedField(t *testing.T) {
	h := Load(t, `
Window {
  width: 400
  height: 300
  VStack {
    TextField { id: first; width: 380; height: 40 }
    TextField { id: second; width: 380; height: 40 }
  }
}`)

	first := h.Widget("first").(*widgets.TextField)
	second := h.Widget("second").(*widgets.TextField)

	fr := first.FrameRect()
	h.Click(fr.X+fr.Width/2, fr.Y+fr.Height/2)
	if !first.Focused {
		t.Fatal("the first TextField did not take focus from the click")
	}

	h.Send(&platform.Event{Type: platform.EventRuneTyped, Rune: 'x'})

	if first.Text != "x" {
		t.Errorf("focused field Text = %q, want %q", first.Text, "x")
	}
	if second.Text != "" {
		t.Errorf("unfocused field Text = %q, want empty — a typed character reached a "+
			"field that does not have focus", second.Text)
	}
}
