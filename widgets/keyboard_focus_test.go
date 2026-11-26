package widgets

// keyboard_focus_test.go — the keyboard as an APPLICATION experiences it, through the real window:
// a held key in a text field, typing in a dialog the moment it opens, Tab through a dialog's fields
// and its buttons, and a tab switch that gives the keyboard back.
//
// These are the integration half of meui/keyboard_focus_test.go: that file pins the window's rules,
// this one pins the widgets that depend on them.

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
)

func keyWin() *meui.Window {
	w := meui.NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 600, 400))
	return w
}

func keyDown(k platform.KeyCode) *platform.Event {
	return &platform.Event{Type: platform.EventKeyDown, Key: k}
}

func keyRepeat(k platform.KeyCode) *platform.Event {
	return &platform.Event{Type: platform.EventKeyDown, Key: k, Repeat: true}
}

func runeEvent(r rune) *platform.Event {
	return &platform.Event{Type: platform.EventRuneTyped, Rune: r}
}

// A held key is what Backspace and the arrows are FOR. The window classifies an OS auto-repeat as a
// repeat rather than a fresh press, and a widget that does not implement OnKeyRepeat is deliberately
// asked only once — so a text field needs OnKeyRepeat or holding Backspace deletes exactly one
// character while the user holds the key.
func TestHoldingBackspaceDeletesUntilReleased(t *testing.T) {
	w := keyWin()
	tf := NewTextField()
	tf.SetText("abcdef")
	tf.CursorPos = len(tf.Text)
	w.AddChild(tf)
	w.SetFocusedChild(tf)
	// Focusing a field selects its text (HTML behaviour); collapse the selection so Backspace is
	// deleting one character, which is what a user sees after clicking into the field.
	tf.SelectStart, tf.SelectEnd = len(tf.Text), len(tf.Text)

	w.HandleEvent(keyDown(platform.KeyBackspace))
	if tf.Text != "abcde" {
		t.Fatalf("after one Backspace the text is %q, want %q", tf.Text, "abcde")
	}
	for i := 0; i < 3; i++ {
		w.HandleEvent(keyRepeat(platform.KeyBackspace))
	}
	if tf.Text != "ab" {
		t.Errorf("holding Backspace left %q, want %q: the repeats never reached the field", tf.Text, "ab")
	}
}

// A dialog TAKES the keyboard when it opens. It used to leave focus on whatever the window had, so the
// first characters typed after a dialog appeared went into the window behind it.
func TestATextFieldInADialogTakesTypingWhenItOpens(t *testing.T) {
	w := keyWin()
	behindField := NewTextField()
	behindField.SetFrameRect(graphics.NewRect(0, 0, 120, 24))
	w.AddChild(behindField)
	w.SetFocusedChild(behindField)

	content := meui.NewPanel()
	dialogField := NewTextField()
	dialogField.SetFrameRect(graphics.NewRect(0, 0, 160, 24))
	content.AddChild(dialogField)

	w.ShowModal(content, 300, 160)

	if w.FocusedChild() != dialogField {
		t.Fatalf("the dialog opened with focus on %T, want its text field", w.FocusedChild())
	}
	for _, r := range "hi" {
		w.HandleEvent(runeEvent(r))
	}
	if dialogField.Text != "hi" {
		t.Errorf("typing in the dialog produced %q, want %q", dialogField.Text, "hi")
	}
	if behindField.Text != "" {
		t.Errorf("typing leaked into the field BEHIND the dialog: %q", behindField.Text)
	}
}

// Tab in a dialog walks the dialog's fields and then its buttons. It used to be offered to the whole
// content tree and the first widget that accepted it took it — a tab character in a text area, a page
// change in a tab panel — so the keyboard could not move at all.
func TestTabWalksTheFieldsAndButtonsOfADialog(t *testing.T) {
	w := keyWin()
	content := meui.NewPanel()
	first := NewTextField()
	first.SetFrameRect(graphics.NewRect(0, 0, 160, 24))
	second := NewTextField()
	second.SetFrameRect(graphics.NewRect(0, 0, 160, 24))
	content.AddChild(first)
	content.AddChild(second)

	ok := NewButton().SetText("OK")
	popup := NewModalPopupPanel("T", "", content, ok)
	w.ShowModal(popup, 320, 200)

	if w.FocusedChild() != first {
		t.Fatalf("the dialog opened with focus on %T, want the first field", w.FocusedChild())
	}
	w.HandleEvent(keyDown(platform.KeyTab))
	if w.FocusedChild() != second {
		t.Fatalf("Tab went to %T, want the second field", w.FocusedChild())
	}
	w.HandleEvent(keyDown(platform.KeyTab))
	if w.FocusedChild() != ok {
		t.Errorf("Tab went to %T, want the dialog's OK button", w.FocusedChild())
	}
}

// Switching pages cannot leave the keyboard on the page that is no longer on screen: Children() only
// exposes the active page, so a hidden field keeps the focus, keeps the focus ring, and keeps eating
// characters the user is typing at the new page.
func TestSwitchingTabsReleasesFocusFromTheHiddenPage(t *testing.T) {
	w := keyWin()
	tabs := NewTabPanel()
	tabs.SetFrameRect(graphics.NewRect(0, 0, 500, 300))

	pageOne := meui.NewPanel()
	field := NewTextField()
	field.SetFrameRect(graphics.NewRect(0, 0, 160, 24))
	pageOne.AddChild(field)
	pageTwo := meui.NewPanel()
	other := NewTextField()
	other.SetFrameRect(graphics.NewRect(0, 0, 160, 24))
	pageTwo.AddChild(other)

	tabs.AddTab("One", pageOne)
	tabs.AddTab("Two", pageTwo)
	w.AddChild(tabs)
	w.SetFocusedChild(field)

	tabs.SetActiveTab(1)
	if w.FocusedChild() == field {
		t.Error("the hidden page's field still holds the keyboard")
	}
	if w.FocusedChild() != tabs {
		t.Errorf("focus went to %T, want the tab panel that is now on screen", w.FocusedChild())
	}
	// And a character typed now must not land in the old page's field.
	w.HandleEvent(runeEvent('x'))
	if field.Text != "" {
		t.Errorf("typing after the switch reached the HIDDEN field: %q", field.Text)
	}
}

// A dropdown is a LAYER, and Escape is how a keyboard user closes it. The layer carried no widget it
// could tell (it is painted by a callback), so the combo never learned it had been closed and stayed
// "expanded" — reopening on the next click as if the list were still up.
func TestEscapeClosesAComboBoxDropdown(t *testing.T) {
	w := keyWin()
	cmb := NewComboBox()
	cmb.Options = []string{"one", "two", "three"}
	cmb.SelectedIndex = 0
	cmb.SetFrameRect(graphics.NewRect(10, 10, 160, 26))
	w.AddChild(cmb)

	cmb.SetExpanded(true)
	if !cmb.Expanded {
		t.Fatal("the combo did not expand")
	}
	before := w.LayerManager.LayerCount()
	if before == 0 {
		t.Fatal("expanding the combo created no layer")
	}

	if !w.HandleEvent(keyDown(platform.KeyEscape)) {
		t.Error("Escape was not handled")
	}
	if cmb.Expanded {
		t.Error("the combo still reports itself expanded after Escape")
	}
	if got := w.LayerManager.LayerCount(); got >= before {
		t.Errorf("Escape left %d layers open, want fewer than %d", got, before)
	}
}

// The chart is in the focus order because New() sets a tab index, not because it overrides Focusable.
// Pinned here so the reason survives a later "simplification" of the chart.
func TestTheChartIsInTheFocusOrderWithoutAnOverride(t *testing.T) {
	// CandleChart lives in another module; the rule under test is BaseWidget's: a widget that sets a
	// tab index is focusable, and a disabled one is not.
	tf := NewTextField()
	if !tf.Focusable() {
		t.Error("a text field is not focusable, so Tab can never reach it")
	}
	tf.SetEnabled(false)
	if tf.Focusable() {
		t.Error("a disabled text field is still focusable")
	}
}
