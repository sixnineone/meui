package platform

// keycode_test.go — the platform package must never invent a keycode VALUE.
//
// A KeyCode is an alias of mobilegles.Keycode, and the pump is the only thing that produces one. This
// package had a second table of numbers, spelled out as "USB HID scancodes", and half of them were
// SDL's scancode numbers instead: KeyDelete was 76 while a real Delete key arrives as 212, so Delete
// never matched anything in any widget — and neither did Home, End, PageUp, PageDown, or Ctrl+F
// (which was spelled 58, the value of F1). The values are all aliases now, and this test is what
// keeps them that way.

import (
	"testing"

	"github.com/sixnineone/mobilegles"
)

func TestEveryPlatformKeycodeIsThePumps(t *testing.T) {
	cases := []struct {
		name string
		p    KeyCode
		m    mobilegles.Keycode
	}{
		{"Return", KeyReturn, mobilegles.KeyEnter},
		{"NumPadEnter", KeyNumPadEnter, mobilegles.KeypadEnter},
		{"Home", KeyHome, mobilegles.KeyHome},
		{"End", KeyEnd, mobilegles.KeyEnd},
		{"Insert", KeyInsert, mobilegles.KeyInsert},
		{"Delete", KeyDelete, mobilegles.KeyDelete},
		{"PageUp", KeyPageUp, mobilegles.KeyPageUp},
		{"PageDown", KeyPageDown, mobilegles.KeyPageDown},

		{"F", KeyF, mobilegles.KeyF},
		{"F1", KeyF1, mobilegles.KeyF1},
		{"F2", KeyF2, mobilegles.KeyF2},
		{"F3", KeyF3, mobilegles.KeyF3},
		{"F4", KeyF4, mobilegles.KeyF4},
		{"F5", KeyF5, mobilegles.KeyF5},
		{"F6", KeyF6, mobilegles.KeyF6},
		{"F7", KeyF7, mobilegles.KeyF7},
		{"F8", KeyF8, mobilegles.KeyF8},
		{"F9", KeyF9, mobilegles.KeyF9},
		{"F10", KeyF10, mobilegles.KeyF10},
		{"F11", KeyF11, mobilegles.KeyF11},
		{"F12", KeyF12, mobilegles.KeyF12},

		{"Minus", KeyMinus, mobilegles.KeyMinus},
		{"Equal", KeyEqual, mobilegles.KeyEqual},
		{"OpenBracket", KeyOpenBracket, mobilegles.KeyOpenBracket},
		{"CloseBracket", KeyCloseBracket, mobilegles.KeyCloseBracket},
		{"Backslash", KeyBackslash, mobilegles.KeyBackslash},
		{"Semicolon", KeySemicolon, mobilegles.KeySemicolon},
		{"Apostrophe", KeyApostrophe, mobilegles.KeyApostrophe},
		{"Grave", KeyGrave, mobilegles.KeyGrave},
		{"Comma", KeyComma, mobilegles.KeyComma},
		{"Period", KeyPeriod, mobilegles.KeyPeriod},
		{"Slash", KeySlash, mobilegles.KeySlash},

		{"KeypadPlus", KeypadPlus, mobilegles.KeypadPlus},
		{"KeypadMinus", KeypadMinus, mobilegles.KeypadMinus},
		{"KeypadEquals", KeypadEquals, mobilegles.KeypadEquals},

		{"Up", KeyUp, mobilegles.KeyUp},
		{"Down", KeyDown, mobilegles.KeyDown},
		{"Left", KeyLeft, mobilegles.KeyLeft},
		{"Right", KeyRight, mobilegles.KeyRight},
		{"Space", KeySpace, mobilegles.KeySpace},
		{"Tab", KeyTab, mobilegles.KeyTab},
		{"Escape", KeyEscape, mobilegles.KeyEscape},
		{"Backspace", KeyBackspace, mobilegles.KeyBackspace},
		{"Enter", KeyEnter, mobilegles.KeyEnter},
	}
	for _, c := range cases {
		if c.p != c.m {
			t.Errorf("%s: platform says %d, the pump produces %d — a widget comparing the platform "+
				"constant would never see a real key", c.name, c.p, c.m)
		}
	}
}
