package widgets

import (
	"strings"
	"testing"

	"github.com/sixnineone/meui/platform"
)

// ── Button ─────────────────────────────────────────────

func TestButtonCheckableToggle(t *testing.T) {
	b := NewButton()
	b.SetText("Toggle")
	fired := false
	b.SetCheckable(true)
	b.SetOnToggled(func(toggled bool) {
		fired = true
		if !toggled {
			t.Errorf("first toggle should be true, got %v", toggled)
		}
	})
	// Simulate a click (press + release inside)
	b.OnMouseDown(10, 10, 0)
	b.OnMouseUp(10, 10, 0)
	if !b.IsToggled() {
		t.Error("button should be toggled after click")
	}
	if !fired {
		t.Error("OnToggled should fire on click")
	}
	// Toggle back off — SetToggled does NOT fire the callback
	b.SetToggled(false)
	if b.IsToggled() {
		t.Error("SetToggled(false) should untoggle")
	}
}

func TestButtonShortcut(t *testing.T) {
	b := NewButton()
	clicked := false
	b.SetOnClicked(func() { clicked = true })
	b.SetShortcut('S')
	// unison keycodes for letters are ASCII uppercase; 'A'+18 == 'S'
	keyCode := platform.KeyCode('A' + 18)
	b.OnKeyDown(keyCode, 0)
	if !clicked {
		t.Error("OnClicked should fire when shortcut key pressed")
	}
}

// ── CheckBox ───────────────────────────────────────────

func TestCheckBoxOnCheckChanged(t *testing.T) {
	c := NewCheckBox()
	got := false
	c.SetOnCheckChanged(func(checked bool) {
		got = checked
	})
	c.SetChecked(true)
	if !got {
		t.Error("OnCheckChanged should fire with true")
	}
	// Setting the same value again must not re-fire
	fired := false
	c.SetOnCheckChanged(func(checked bool) { fired = true })
	c.SetChecked(true)
	if fired {
		t.Error("OnCheckChanged must not fire when value unchanged")
	}
}

// ── Switch ─────────────────────────────────────────────

func TestSwitchText(t *testing.T) {
	s := NewSwitch()
	s.SetText("Dark mode")
	if s.Text != "Dark mode" {
		t.Errorf("Text = %q", s.Text)
	}
	toggled := false
	s.SetOnToggled(func(bool) { toggled = true })
	s.SetChecked(true)
	if !toggled {
		t.Error("OnToggled should fire")
	}
}

// ── TextField ──────────────────────────────────────────

// safeClipboard runs fn, recovering from GLFW clipboard panics in headless tests.
func safeClipboard(fn func()) {
	defer func() { _ = recover() }()
	fn()
}

func TestTextFieldReadOnly(t *testing.T) {
	tf := NewTextField()
	tf.SetText("hello")
	tf.SetReadOnly(true)
	if !tf.IsReadOnly() {
		t.Error("IsReadOnly should be true")
	}
	// typing must be ignored
	tf.Focused = true
	tf.OnCharacter('X')
	if tf.Text != "hello" {
		t.Errorf("ReadOnly TextField changed text to %q", tf.Text)
	}
	// cut must be ignored
	tf.SelectAll()
	tf.Cut()
	if tf.Text != "hello" {
		t.Errorf("ReadOnly Cut modified text to %q", tf.Text)
	}
	// paste must be ignored
	tf.Paste()
	if tf.Text != "hello" {
		t.Errorf("ReadOnly Paste modified text to %q", tf.Text)
	}
	// Clear must be ignored
	tf.Clear()
	if tf.Text != "hello" {
		t.Errorf("ReadOnly Clear modified text to %q", tf.Text)
	}
	// copy still works (clipboard needs GLFW — recover headless panic)
	safeClipboard(func() {
		tf.SelectAll()
		tf.Copy()
	})
	if tf.GetSelectedText() != "hello" {
		t.Errorf("GetSelectedText = %q, want hello", tf.GetSelectedText())
	}
}

func TestTextFieldSelectionAPI(t *testing.T) {
	tf := NewTextField()
	tf.SetText("hello world")
	tf.SetSelection(0, 5)
	if got := tf.GetSelectedText(); got != "hello" {
		t.Errorf("GetSelectedText = %q, want hello", got)
	}
	if tf.SelectionLength() != 5 {
		t.Errorf("SelectionLength = %d, want 5", tf.SelectionLength())
	}
	// clamping beyond the end
	tf.SetSelection(6, 100)
	if got := tf.GetSelectedText(); got != "world" {
		t.Errorf("clamped selection = %q, want world", got)
	}
	if tf.SelectionLength() != 0 {
		// fresh textfield without selection
	}
	tf2 := NewTextField()
	tf2.SetText("abc")
	if tf2.SelectionLength() != 0 {
		t.Error("fresh textfield should have no selection")
	}
}

func TestTextFieldClear(t *testing.T) {
	tf := NewTextField()
	tf.SetText("data")
	changed := ""
	tf.SetOnTextChanged(func(s string) { changed = s })
	tf.Clear()
	if tf.Text != "" || changed != "" {
		t.Errorf("Clear: text=%q changed=%q", tf.Text, changed)
	}
}

// ── TextArea ───────────────────────────────────────────

func TestTextAreaReadOnlyAndHelpers(t *testing.T) {
	ta := NewTextArea()
	ta.SetText("line1\nline2")
	ta.SetReadOnly(true)
	ta.Focused = true
	ta.OnCharacter('X')
	if ta.Value != "line1\nline2" {
		t.Errorf("ReadOnly TextArea changed value to %q", ta.Value)
	}
	ta.SelectAll()
	if got := ta.GetSelectedText(); got != "line1\nline2" {
		t.Errorf("GetSelectedText = %q", got)
	}
	if ta.SelectionLength() != len("line1\nline2") {
		t.Errorf("SelectionLength = %d", ta.SelectionLength())
	}
	ta.AppendText("more")
	if strings.Contains(ta.Value, "more") {
		t.Error("ReadOnly AppendText should be ignored")
	}
	ta.SetReadOnly(false)
	ta.AppendText("more")
	if !strings.HasSuffix(ta.Value, "more") {
		t.Errorf("AppendText failed: %q", ta.Value)
	}
	ta.Clear()
	if ta.Value != "" {
		t.Errorf("Clear failed: %q", ta.Value)
	}
}

// ── ComboBox ───────────────────────────────────────────

func TestComboBoxItemManagement(t *testing.T) {
	cb := NewComboBox()
	cb.SetOptions([]string{"A", "B", "C"})
	if cb.SelectedIndex != 0 {
		t.Errorf("auto-select should pick index 0, got %d", cb.SelectedIndex)
	}
	if cb.GetSelectedText() != "A" {
		t.Errorf("GetSelectedText = %q", cb.GetSelectedText())
	}
	cb.AddItem("D")
	if cb.ItemCount() != 4 {
		t.Errorf("ItemCount = %d, want 4", cb.ItemCount())
	}
	cb.RemoveItem(1) // remove "B"
	if cb.ItemCount() != 3 || cb.Options[1] != "C" {
		t.Errorf("RemoveItem: %v", cb.Options)
	}
	cb.ClearItems()
	if cb.ItemCount() != 0 || cb.SelectedIndex != -1 {
		t.Error("ClearItems should empty the combo")
	}
	if cb.GetSelectedText() != "" {
		t.Error("GetSelectedText after clear should be empty")
	}
}

func TestComboBoxEditableAndPlaceholder(t *testing.T) {
	cb := NewComboBox()
	cb.SetEditable(true)
	if !cb.Editable {
		t.Error("SetEditable should set Editable")
	}
	cb.SetPlaceholder("Pick one…")
	if cb.PlaceholderText != "Pick one…" {
		t.Errorf("PlaceholderText = %q", cb.PlaceholderText)
	}
	cb.ClearSelection()
	if cb.SelectedIndex != -1 {
		t.Error("ClearSelection should reset index")
	}
}

// ── SpinBox ────────────────────────────────────────────

func TestSpinBoxRangePrefixSuffixWrapping(t *testing.T) {
	sb := NewSpinBox()
	sb.SetRange(0, 10)
	sb.SetStep(2)
	sb.SetDecimals(0)
	sb.SetValue(9)
	sb.increment()
	if sb.Value != 10 {
		t.Errorf("increment clamp: %v, want 10", sb.Value)
	}
	sb.increment()
	if sb.Value != 10 {
		t.Errorf("no wrap by default: %v, want 10", sb.Value)
	}
	sb.SetWrapping(true)
	sb.increment()
	if sb.Value != 0 {
		t.Errorf("wrap should go to minimum, got %v", sb.Value)
	}
	sb.decrement()
	if sb.Value != 10 {
		t.Errorf("wrap down should go to maximum, got %v", sb.Value)
	}

	sb2 := NewSpinBox()
	sb2.SetDecimals(1)
	sb2.SetPrefix("$")
	sb2.SetSuffix(" kg")
	sb2.SetValue(5)
	if got := sb2.GetText(); got != "$5.0 kg" {
		t.Errorf("GetText = %q, want $5.0 kg", got)
	}
}

// ── ListView ───────────────────────────────────────────

func TestListViewItemManagement(t *testing.T) {
	lv := NewListView()
	lv.SetItems([]string{"one", "two", "three"})
	lv.SetSelectedIndex(1)
	if lv.GetSelectedItem() != "two" {
		t.Errorf("GetSelectedItem = %q", lv.GetSelectedItem())
	}
	lv.RemoveItem(0)
	if lv.ItemCount() != 2 || lv.GetItem(0) != "two" {
		t.Errorf("RemoveItem: %v", lv.Items)
	}
	// selection shifted down
	if lv.SelectedIndex != 0 {
		t.Errorf("selection after remove = %d, want 0", lv.SelectedIndex)
	}
	lv.ScrollToIndex(1)
	lv.ClearItems()
	if lv.ItemCount() != 0 || lv.SelectedIndex != -1 {
		t.Error("ClearItems should reset the list")
	}
}

// ── Tree ───────────────────────────────────────────────

func TestTreeSelectionExpandRemove(t *testing.T) {
	tr := NewTree()
	// Root is a hidden container node (Label == "" is not rendered).
	root := &TreeNode{Label: "", Expanded: true}
	tr.SetRoot(root)
	childA := tr.AddChild(root, "A")
	tr.AddChild(root, "B")
	grand := tr.AddChild(childA, "A1")

	// flat order: [A, B] (children not expanded yet)
	tr.SelectedIndex = 1
	if n := tr.GetSelectedNode(); n == nil || n.Label != "B" {
		t.Errorf("GetSelectedNode = %+v", n)
	}

	// expand all → [A, A1, B]
	tr.ExpandAll()
	if !childA.Expanded {
		t.Error("ExpandAll should expand childA")
	}
	if len(tr.flatNodes) != 3 {
		t.Fatalf("flatNodes after ExpandAll = %d, want 3", len(tr.flatNodes))
	}
	if tr.flatNodes[0].Label != "A" || tr.flatNodes[0].Depth != 0 {
		t.Errorf("first flat node = %q depth %d, want A depth 0", tr.flatNodes[0].Label, tr.flatNodes[0].Depth)
	}

	// collapse all → [A, B]
	tr.CollapseAll()
	if childA.Expanded {
		t.Error("CollapseAll should collapse childA")
	}
	if len(tr.flatNodes) != 2 {
		t.Fatalf("flatNodes after CollapseAll = %d, want 2", len(tr.flatNodes))
	}

	// expanding a collapsed child must also expand its ancestors
	tr.ExpandNode(childA, true)
	if len(tr.flatNodes) != 3 {
		t.Fatalf("flatNodes after ExpandNode = %d, want 3", len(tr.flatNodes))
	}

	// remove a child
	tr.RemoveNode(grand)
	if len(childA.Children) != 0 {
		t.Error("RemoveNode should detach grandchild")
	}

	// selection callback via setter
	fired := false
	tr.SetOnNodeSelected(func(index int, node *TreeNode) { fired = true })
	tr.OnItemSelected(0, root)
	if !fired {
		t.Error("SetOnNodeSelected handler should be invocable")
	}
}

// ── SplitView ──────────────────────────────────────────

func TestSplitViewHandleSizeAndSplitMoved(t *testing.T) {
	sv := NewSplitView()
	sv.SetHandleSize(10)
	if sv.HandleSize != 10 {
		t.Errorf("HandleSize = %v", sv.HandleSize)
	}
	moved := false
	sv.SetOnSplitMoved(func(pos float32) { moved = true })
	// drag end while dragging → fires with the final position
	sv.dragging = true
	sv.SplitPosition = 0.5
	sv.OnMouseUp(0, 0, 0)
	if !moved {
		t.Error("OnSplitMoved should fire when a drag ends")
	}
	// plain mouse-up without drag → no fire
	moved = false
	sv.OnMouseUp(0, 0, 0)
	if moved {
		t.Error("OnSplitMoved should NOT fire without an active drag")
	}
}

// ── Calendar / ColorPicker ─────────────────────────────

func TestCalendarSetters(t *testing.T) {
	cal := NewCalendar()
	fired := false
	cal.SetOnDateSelected(func(y, m, d int) {
		fired = true
	})
	if cal.OnDateSelected == nil {
		t.Fatal("SetOnDateSelected should store the handler")
	}
	cal.OnDateSelected(2026, 8, 29)
	if !fired {
		t.Error("handler should be invocable")
	}
	cal.SetDate(2026, 8, 29)
	y, m, d := cal.GetSelectedDate()
	if y != 2026 || m != 8 || d != 29 {
		t.Errorf("GetSelectedDate = %d-%d-%d", y, m, d)
	}
}

func TestColorPickerSetters(t *testing.T) {
	cp := NewColorPicker()
	fired := false
	cp.SetOnColorChanged(func(r, g, b, a uint8) { fired = true })
	if cp.OnColorChanged == nil {
		t.Fatal("SetOnColorChanged should store the handler")
	}
	cp.OnColorChanged(1, 2, 3, 4)
	if !fired {
		t.Error("handler should be invocable")
	}
	cp.SetColor(255, 128, 0, 255)
	r, g, b, a := cp.GetColor()
	if r != 255 || g != 128 || b != 0 || a != 255 {
		t.Errorf("GetColor = %d,%d,%d,%d", r, g, b, a)
	}
}

// ── StatusBar ──────────────────────────────────────────

func TestStatusBarShowMessage(t *testing.T) {
	sb := NewStatusBar()
	sb.SetText("ready")
	sb.ShowMessage("saving…", 0) // 0 = no auto-clear
	if sb.tempMessage != "saving…" {
		t.Errorf("tempMessage = %q", sb.tempMessage)
	}
	sb.ClearMessage()
	if sb.tempMessage != "" {
		t.Error("ClearMessage should reset tempMessage")
	}
	// timeout version schedules a timer
	sb.ShowMessage("temp", 10)
	if sb.tempTimer == nil {
		t.Error("ShowMessage with timeout should create a timer")
	}
	sb.ClearMessage()
}

// ── Label ──────────────────────────────────────────────

func TestLabelFontAPIAndCopyOnWrite(t *testing.T) {
	// SetColor must not mutate the shared default theme
	l1 := NewLabel()
	l2 := NewLabel()
	l1.SetColor(0xFFFF0000)
	if l2.LabelTheme.TextInk == 0xFFFF0000 {
		t.Error("SetColor leaked into the shared default theme")
	}

	// SetFontSize changes metrics
	l3 := NewLabel()
	l3.SetFontSize(20)
	if l3.LabelTheme.Font == nil {
		t.Error("SetFontSize should set a font")
	}
	// bold/italic builders
	l4 := NewLabel()
	l4.SetFontBold(true)
	if l4.LabelTheme.Font == nil {
		t.Error("SetFontBold should set a font")
	}
	l5 := NewLabel()
	l5.SetFontItalic(true)
	if l5.LabelTheme.Font == nil {
		t.Error("SetFontItalic should set a font")
	}
}
