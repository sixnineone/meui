package meui

// keyboard_focus_test.go — the FOCUS half of the keyboard: which widget owns the keyboard, what it
// takes to move it, and what happens when the widget that had it goes away.
//
// These are the contracts `keyboard_test.go` does not cover, and every one of them failed before the
// fix: a dialog left the keyboard behind it (and Tab cycled the window behind it), Escape left a
// modal's backdrop and its bookkeeping in place, a removed widget kept the keyboard, setting focus
// left the old widget believing it still had it, and a key could not reach an ancestor.

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
)

// focusProbe is a widget that records what the focus and key machinery does to it. It is built through
// NewBaseWidget so it is a LEGITIMATE widget (visible, enabled) — a zero-value BaseWidget is neither,
// which is a property of the fixture, not of the toolkit.
type focusProbe struct {
	BaseWidget
	focusable bool
	consume   bool
	focused   int
	blurred   int
	keys      []platform.KeyCode
	clicks    int
}

func newProbe(w, h float32) *focusProbe {
	p := &focusProbe{}
	p.BaseWidget = *NewBaseWidget()
	p.SetFrameRect(graphics.NewRect(0, 0, w, h))
	return p
}

func (p *focusProbe) Focusable() bool { return p.focusable }
func (p *focusProbe) OnFocus()        { p.focused++; p.BaseWidget.OnFocus() }
func (p *focusProbe) OnBlur()         { p.blurred++; p.BaseWidget.OnBlur() }
func (p *focusProbe) OnKeyDown(k platform.KeyCode, _ platform.Modifiers) bool {
	p.keys = append(p.keys, k)
	return p.consume
}

// ── Escape dismisses a dialog as a UNIT ─────────────────────────────────

// A modal is two layers (content + backdrop) and one entry in activeModals. Escape used to close ONE
// layer through LayerManager.CloseTopLayer, which left the dimming, click-blocking backdrop over a
// window with no dialog in it, and left the popup in activeModals and the app's dismiss function unrun.
func TestEscapeDismissesADialogAndItsBackdrop(t *testing.T) {
	w := NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	behind := newProbe(400, 300)
	w.AddChild(behind)
	w.SetFocusedChild(behind)

	content := NewPanel()
	dialogField := newProbe(120, 24)
	dialogField.focusable = true
	content.AddChild(dialogField)

	mp := w.ShowModal(content, 200, 100)
	if mp == nil {
		t.Fatal("ShowModal returned nothing")
	}
	if got := w.LayerManager.LayerCount(); got != 2 {
		t.Fatalf("a modal opened with %d layers, want 2 (content + backdrop)", got)
	}
	if w.FocusedChild() != dialogField {
		t.Errorf("the dialog did not take the keyboard: focus is %T, want the dialog's first field", w.FocusedChild())
	}

	if !w.HandleEvent(keyDown(platform.KeyEscape)) {
		t.Error("Escape was not consumed by the dialog")
	}
	if got := w.LayerManager.LayerCount(); got != 0 {
		t.Errorf("after Escape the window still has %d layers — a leftover backdrop blocks every click", got)
	}
	if len(w.activeModals) != 0 {
		t.Errorf("after Escape the window still lists %d modals", len(w.activeModals))
	}
	if w.FocusedChild() != behind {
		t.Errorf("after the dialog closed the keyboard is on %T, want the widget that had it before", w.FocusedChild())
	}
	// The widget behind was focused once, blurred when the dialog took over, and focused again when
	// the dialog gave the keyboard back — the exchange that proves the focus really moved.
	if behind.focused != 2 || behind.blurred != 1 {
		t.Errorf("the widget behind the dialog saw %d focuses and %d blurs, want 2 and 1",
			behind.focused, behind.blurred)
	}
	if dialogField.focused != 1 || dialogField.blurred != 1 {
		t.Errorf("the dialog's field saw %d focuses and %d blurs, want 1 and 1",
			dialogField.focused, dialogField.blurred)
	}
}

// ── Tab keeps the keyboard in the dialog ────────────────────────────────

// Tab walked the WINDOW's children, and a dialog's widgets are in a layer — not in that tree. So Tab
// with a dialog open cycled the controls BEHIND it while the dialog sat on top, and a keyboard-only
// user could never reach the dialog at all.
func TestTabStaysInsideAnOpenDialog(t *testing.T) {
	w := NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	behind := newProbe(100, 24)
	behind.focusable = true
	w.AddChild(behind)

	content := NewPanel()
	first := newProbe(120, 24)
	first.focusable = true
	second := newProbe(120, 24)
	second.focusable = true
	content.AddChild(first)
	content.AddChild(second)

	w.ShowModal(content, 240, 120)
	if w.FocusedChild() != first {
		t.Fatalf("the dialog opened with focus on %T, want its first field", w.FocusedChild())
	}

	w.HandleEvent(keyDown(platform.KeyTab))
	if w.FocusedChild() != second {
		t.Errorf("Tab moved focus to %T, want the dialog's second field", w.FocusedChild())
	}
	w.HandleEvent(keyDown(platform.KeyTab))
	if w.FocusedChild() == behind {
		t.Error("Tab escaped the dialog and focused a widget behind it")
	}
	if w.FocusedChild() != first && w.FocusedChild() != second {
		t.Errorf("Tab left the dialog: focus is %T", w.FocusedChild())
	}
}

// ── a removed widget cannot keep the keyboard ───────────────────────────

func TestRemovingTheFocusedWidgetReleasesFocus(t *testing.T) {
	w := NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	p := newProbe(100, 24)
	p.focusable = true
	w.AddChild(p)
	w.SetFocusedChild(p)

	if !w.RemoveChild(p) {
		t.Fatal("the child was not removed")
	}
	if w.FocusedChild() != nil {
		t.Errorf("the removed widget still holds the keyboard (%T)", w.FocusedChild())
	}
	if p.blurred != 1 {
		t.Errorf("the removed widget was blurred %d times, want 1", p.blurred)
	}
}

// Setting focus must BLUR the old widget. Assigning the field left two controls both believing they
// were focused — both painted the focus ink, and a text widget kept accepting characters.
func TestSettingFocusBlursThePreviousWidget(t *testing.T) {
	w := NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	a := newProbe(100, 24)
	b := newProbe(100, 24)
	a.focusable, b.focusable = true, true
	w.AddChild(a)
	w.AddChild(b)

	w.SetFocusedChild(a)
	w.SetFocusedChild(b)
	if a.focused != 1 || a.blurred != 1 {
		t.Errorf("the first widget saw %d focuses / %d blurs, want 1 / 1", a.focused, a.blurred)
	}
	if b.focused != 1 || b.blurred != 0 {
		t.Errorf("the second widget saw %d focuses / %d blurs, want 1 / 0", b.focused, b.blurred)
	}
}

// ── keys bubble through the focused widget's ancestors ──────────────────

// A key used to reach the focused widget and then the window's DIRECT children, so a container between
// them could never act on a key for the control it owns. The app documented the symptom: the trades
// table's Delete key was unreachable because the table's tab panel never saw the key.
func TestAKeyReachesTheFocusedWidgetsAncestors(t *testing.T) {
	w := NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	owner := &keyOwner{}
	owner.BaseContainer.BaseWidget = *NewBaseWidget()
	owner.SetFrameRect(graphics.NewRect(0, 0, 400, 300))

	child := newProbe(100, 24)
	child.focusable = true
	owner.AppendChild(child)
	child.SetParent(owner)
	w.AddChild(owner)
	w.SetFocusedChild(child)

	w.HandleEvent(keyDown(platform.KeyDelete))
	if len(child.keys) != 1 {
		t.Errorf("the focused widget saw %d keys, want 1", len(child.keys))
	}
	if len(owner.keys) != 1 {
		t.Errorf("the key did not bubble to the container (it saw %d keys, want 1)", len(owner.keys))
	}
}

// keyOwner is a container that claims every key it is given, for the bubbling test. AppendChild +
// SetParent(outer) is what a container in another package does, so the chain names the OUTER widget.
type keyOwner struct {
	BaseContainer
	keys []platform.KeyCode
}

func (o *keyOwner) OnKeyDown(k platform.KeyCode, _ platform.Modifiers) bool {
	o.keys = append(o.keys, k)
	return true
}

// ── an unfocused sibling does not act ───────────────────────────────────

// Panel.OnKeyDown broadcast every key to every child, so an UNFOCUSED tab panel switched page when the
// user pressed Right — in a window where the focus was on a table or a button. A key belongs to the
// focused widget and its ancestors, and to nothing else.
func TestAnUnfocusedSiblingDoesNotActOnAKey(t *testing.T) {
	w := NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	focused := newProbe(100, 24)
	focused.focusable = true
	sibling := newProbe(100, 24)
	w.AddChild(focused)
	w.AddChild(sibling)
	w.SetFocusedChild(focused)

	w.HandleEvent(keyDown(platform.KeyRight))
	if len(sibling.keys) != 0 {
		t.Errorf("an UNFOCUSED sibling was given the key %d times", len(sibling.keys))
	}
}

// ── Tab order ───────────────────────────────────────────────────────────

// TabIndex exercised the lowering, and nothing ever READ it: the traversal was tree order, so a
// widget's declared place in the cycle was ignored. Equal indices keep tree order.
func TestTabFollowsTheDeclaredTabIndex(t *testing.T) {
	w := NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	late := newProbe(100, 24)
	late.focusable = true
	late.SetTabIndex(5)
	early := newProbe(100, 24)
	early.focusable = true
	early.SetTabIndex(1)
	w.AddChild(late) // tree order puts "late" first; its tab index says otherwise
	w.AddChild(early)

	w.HandleEvent(keyDown(platform.KeyTab))
	if w.FocusedChild() != early {
		t.Errorf("Tab went to %T (tabIndex 5), want the tabIndex-1 widget", w.FocusedChild())
	}
	w.HandleEvent(keyDown(platform.KeyTab))
	if w.FocusedChild() != late {
		t.Errorf("the second Tab went to %T, want the tabIndex-5 widget", w.FocusedChild())
	}
}

// ── a click focuses the CONTROL, not the thing under the cursor ─────────

// The window focused the DEEPEST hit widget. A click on a framed field therefore focused its label, and
// a click on a table row focused the cell — neither of which acts on the keyboard. Focus goes to the
// nearest focusable ancestor.
func TestAClickFocusesTheNearestFocusableAncestor(t *testing.T) {
	w := NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	box := &clickBox{focusable: true}
	box.BaseWidget = *NewBaseWidget()
	box.SetFrameRect(graphics.NewRect(10, 10, 200, 100))

	inner := newProbe(40, 20)
	inner.SetFrameRect(graphics.NewRect(5, 5, 40, 20))
	box.AppendChild(inner)
	inner.SetParent(box)
	w.AddChild(box)

	ev := &platform.Event{Type: platform.EventMouseDown, Pos: graphics.NewPoint(20, 20), Button: 0}
	w.HandleEvent(ev)
	if w.FocusedChild() != box {
		t.Errorf("a click on a child focused %T, want the focusable container", w.FocusedChild())
	}
}

// clickBox is a focusable container that hit-tests its children, for the click-focus test.
type clickBox struct {
	BaseWidget
	focusable bool
	children  []Widget
}

func (b *clickBox) Focusable() bool      { return b.focusable }
func (b *clickBox) Children() []Widget   { return b.children }
func (b *clickBox) ValidateLayout()      {}
func (b *clickBox) AppendChild(c Widget) { b.children = append(b.children, c) }
func (b *clickBox) PanelAt(x, y float32) Widget {
	for i := len(b.children) - 1; i >= 0; i-- {
		c := b.children[i]
		fr := c.FrameRect()
		if x >= fr.X && x <= fr.X+fr.Width && y >= fr.Y && y <= fr.Y+fr.Height {
			return c
		}
	}
	return b
}

// ── semantic actions reach the app ──────────────────────────────────────

// The adapter put the keymap's matches on the event and NOTHING read them, so an app could not bind a
// command (Ctrl+W, Ctrl+F) at all. Actions are offered last — a shortcut must never steal a key a
// widget needed — and never on a repeat, or a held key fires the command forever.
func TestEventActionsReachTheAppAfterTheWidgetsDecline(t *testing.T) {
	w := NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	p := newProbe(100, 24)
	p.focusable = true
	w.AddChild(p)
	w.SetFocusedChild(p)

	var seen []string
	w.SetOnAction(func(a string) bool {
		seen = append(seen, a)
		return true
	})

	ev := keyDown(platform.KeyW)
	ev.Actions = []string{"tab.close"}
	if !w.HandleEvent(ev) {
		t.Error("the app's action handler did not consume the key")
	}
	if len(seen) != 1 || seen[0] != "tab.close" {
		t.Errorf("the app saw actions %v, want [tab.close]", seen)
	}

	// The widget takes priority: a control that wants the key keeps it.
	p.consume = true
	seen = nil
	ev2 := keyDown(platform.KeyF)
	ev2.Actions = []string{"view.focus-search"}
	w.HandleEvent(ev2)
	if len(seen) != 0 {
		t.Errorf("the app's handler ran for a key the widget had already consumed: %v", seen)
	}

	// A held key is one command, not a stream.
	p.consume = false
	seen = nil
	ev3 := keyDownRepeat(platform.KeyW)
	ev3.Actions = []string{"tab.close"}
	w.HandleEvent(ev3)
	if len(seen) != 0 {
		t.Errorf("a REPEATED key fired the action %d times", len(seen))
	}
}

// ── IME composition ─────────────────────────────────────────────────────

// composeProbe records the composition events it is given.
type composeProbe struct {
	BaseWidget
	texts []string
}

func (c *composeProbe) OnComposition(text string, start, length int) bool {
	c.texts = append(c.texts, text)
	return true
}

// The IME's in-progress text belongs to the focused widget, and MOVING FOCUS ends it: a preedit is
// not text yet, so it must not be left in a field the user is no longer typing into.
func TestTheWindowRoutesCompositionToTheFocusedWidget(t *testing.T) {
	w := NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	a := &composeProbe{}
	a.BaseWidget = *NewBaseWidget()
	b := &composeProbe{}
	b.BaseWidget = *NewBaseWidget()
	w.AddChild(a)
	w.AddChild(b)
	w.SetFocusedChild(a)

	if !w.HandleEvent(&platform.Event{Type: platform.EventTextEditing, Text: "にほ", CompStart: 1, CompLength: 1}) {
		t.Error("the composition event was not consumed")
	}
	if len(a.texts) != 1 || a.texts[0] != "にほ" {
		t.Errorf("the focused widget saw compositions %q, want [にほ]", a.texts)
	}

	w.SetFocusedChild(b)
	if len(a.texts) != 2 || a.texts[1] != "" {
		t.Errorf("moving focus did not cancel the composition: the old widget saw %q", a.texts)
	}
	if len(b.texts) != 0 {
		t.Errorf("the newly focused widget was given a composition it never started: %q", b.texts)
	}
}
