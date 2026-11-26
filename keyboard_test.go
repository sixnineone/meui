package meui

// keyboard_test.go — what a UI framework's keyboard handling has to do beyond mapping keys.
//
// `sdl3_keyboard_test.go` in mobilegles covers WHICH key a scancode means. This covers what the WINDOW
// does with it: what repeats, what does not, what a release means, and which widget a key belongs to when
// something is open on top.
//
// Every test here fails on the code as it was. They are grouped by the gap they close.

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
)

// ── fixtures ────────────────────────────────────────────────────────────

// keyWidget records every key it is given, so a test can assert what reached a widget and in what order.
type keyWidget struct {
	BaseWidget
	downs     []platform.KeyCode
	repeats   []platform.KeyCode
	ups       []platform.KeyCode
	runes     []rune
	presses   int // OnKeyDown calls, which a FRESH press must be and a repeat must not
	released  int
	consume   bool
	width     float32
	height    float32
	focusable bool
}

func (k *keyWidget) FrameRect() graphics.Rect { return graphics.NewRect(0, 0, k.width, k.height) }
func (k *keyWidget) OnKeyDown(key platform.KeyCode, _ platform.Modifiers) bool {
	k.downs = append(k.downs, key)
	k.presses++
	return k.consume
}
func (k *keyWidget) OnKeyRepeat(key platform.KeyCode, _ platform.Modifiers) bool {
	k.repeats = append(k.repeats, key)
	return k.consume
}
func (k *keyWidget) OnKeyUp(key platform.KeyCode, _ platform.Modifiers) bool {
	k.ups = append(k.ups, key)
	k.released++
	return k.consume
}
func (k *keyWidget) OnCharacter(r rune) { k.runes = append(k.runes, r) }
func (k *keyWidget) Focusable() bool    { return k.focusable }
func (k *keyWidget) Enabled() bool      { return true }

// windowWith builds a window with one focused key widget, for the keyboard tests.
func windowWith(t *testing.T, k *keyWidget) *Window {
	t.Helper()
	w := NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	w.AddChild(k)
	w.SetFocusedChild(k)
	return w
}

func keyDown(k platform.KeyCode) *platform.Event {
	return &platform.Event{Type: platform.EventKeyDown, Key: k}
}
func keyDownRepeat(k platform.KeyCode) *platform.Event {
	return &platform.Event{Type: platform.EventKeyDown, Key: k, Repeat: true}
}
func keyUp(k platform.KeyCode) *platform.Event {
	return &platform.Event{Type: platform.EventKeyUp, Key: k}
}

// ── 1. repeat ───────────────────────────────────────────────────────────

// TestAHeldKeyRepeatsToTheWidgetThatAsksForRepeats: the flag must survive the whole way from the pump,
// and a widget that implements OnKeyRepeat must see the repeats as repeats.
func TestAHeldKeyRepeatsToTheWidgetThatAsksForRepeats(t *testing.T) {
	k := &keyWidget{}
	w := windowWith(t, k)

	w.HandleEvent(keyDown(platform.KeyRight))
	for i := 0; i < 3; i++ {
		w.HandleEvent(keyDownRepeat(platform.KeyRight))
	}

	if k.presses != 1 {
		t.Errorf("a held key produced %d key presses, want 1: the repeats were delivered as fresh presses",
			k.presses)
	}
	if len(k.repeats) != 3 {
		t.Errorf("the widget saw %d repeats, want 3", len(k.repeats))
	}
	for i, r := range k.repeats {
		if r != platform.KeyRight {
			t.Errorf("repeat %d is %v, want right", i, r)
		}
	}
}

// TestAWidgetWithNoInterestInRepeatsIsNotAskedAgain is the other half, and the reason the repeat path
// RETURNS instead of falling through: a widget that does not implement OnKeyRepeat must not be handed the
// repeats as OnKeyDown, or holding any key turns one action into an endless stream of them.
func TestAWidgetWithNoInterestInRepeatsIsNotAskedAgain(t *testing.T) {
	plain := &plainKeyWidget{}
	w := NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	w.AddChild(plain)
	w.SetFocusedChild(plain)

	w.HandleEvent(keyDown(platform.KeyDown))
	for i := 0; i < 5; i++ {
		w.HandleEvent(keyDownRepeat(platform.KeyDown))
	}
	if plain.presses != 1 {
		t.Errorf("a widget that does not implement OnKeyRepeat was asked %d times for one held key, "+
			"want 1", plain.presses)
	}
}

// plainKeyWidget is a widget with the ORIGINAL listener shape and no repeat interest.
type plainKeyWidget struct {
	BaseWidget
	presses int
}

func (p *plainKeyWidget) OnKeyDown(platform.KeyCode, platform.Modifiers) bool {
	p.presses++
	return false
}

func (k *plainKeyWidget) FrameRect() graphics.Rect { return graphics.NewRect(0, 0, 10, 10) }

// TestAModifierNeverRepeats: holding Shift must not deliver Shift over and over, and an OS will happily
// repeat it. A widget that increments on a key press would count to a hundred while nobody touched it.
func TestAModifierNeverRepeats(t *testing.T) {
	for _, k := range []platform.KeyCode{
		platform.KeyLeftShift, platform.KeyRightShift,
		platform.KeyLeftCtrl, platform.KeyRightCtrl,
		platform.KeyLeftAlt, platform.KeyRightAlt,
		platform.KeyLeftMeta, platform.KeyRightMeta,
	} {
		if !isModifierKey(k) {
			t.Errorf("%v is not recognised as a modifier, so it would repeat", k)
			continue
		}
		widget := &keyWidget{}
		w := windowWith(t, widget)
		w.HandleEvent(keyDown(k))
		w.HandleEvent(keyDownRepeat(k))
		w.HandleEvent(keyDownRepeat(k))
		if len(widget.repeats) != 0 || widget.presses > 1 {
			t.Errorf("holding %v delivered %d presses and %d repeats; a modifier is a state, not an action",
				k, widget.presses, len(widget.repeats))
		}
	}
}

// TestTheWindowDispatchesARepeatOnlyOnce covers the path every key takes, fresh or held: a repeat is
// never turned into a fresh press, whatever produced the event. (An app SHORTCUT has the same rule
// through platform.Event.Actions, pinned by TestEventActionsReachTheAppAfterTheWidgetsDecline.)
func TestTheWindowDispatchesARepeatOnlyOnce(t *testing.T) {
	k := &keyWidget{}
	w := windowWith(t, k)

	w.dispatchKeyDown(platform.KeyA, platform.ModNone, false)
	w.dispatchKeyDown(platform.KeyA, platform.ModNone, true)
	w.dispatchKeyDown(platform.KeyA, platform.ModNone, true)

	if k.presses != 1 {
		t.Errorf("a held key produced %d presses, want 1", k.presses)
	}
	if len(k.repeats) != 2 {
		t.Errorf("the held key produced %d repeats, want 2", len(k.repeats))
	}
}

// ── 2. key release ──────────────────────────────────────────────────────

// TestAKeyUpIsDelivered: the window dropped key-ups with a note saying "no key-up behavior yet". A widget
// cannot tell "let go" from "never pressed", which is the whole of a held-key interaction.
func TestAKeyUpIsDelivered(t *testing.T) {
	k := &keyWidget{}
	w := windowWith(t, k)

	w.HandleEvent(keyDown(platform.KeyLeft))
	w.HandleEvent(keyUp(platform.KeyLeft))

	if len(k.ups) != 1 {
		t.Fatalf("the widget saw %d key releases, want 1", len(k.ups))
	}
	if k.ups[0] != platform.KeyLeft {
		t.Errorf("the release is %v, want the key that was pressed", k.ups[0])
	}
	if len(k.downs) != 1 {
		t.Errorf("the release was reported as %d presses", len(k.downs))
	}
}

// TestAReleaseIsDeliveredToALayerToo: a popup lives in a layer, and a layer could never observe a
// release for the same reason the window could not.
func TestAReleaseIsDeliveredToALayerToo(t *testing.T) {
	w := NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	k := &keyWidget{}
	w.LayerManager.AddLayer(0, k)

	// The return value is the widget's "I handled it", which this one does not claim — what matters is
	// that the release ARRIVED.
	w.LayerManager.HandleKeyUp(platform.KeyEnter, platform.ModNone)
	if len(k.ups) != 1 {
		t.Errorf("the layer widget saw %d releases, want 1", len(k.ups))
	}
}

// ── 3. held-key state ───────────────────────────────────────────────────

// TestTheWindowKnowsWhichKeysAreHeld: a UI framework needs "is shift down RIGHT NOW", which an event's
// modifier bits cannot answer — they describe the event that arrived.
func TestTheWindowKnowsWhichKeysAreHeld(t *testing.T) {
	w := NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 400, 300))

	if w.KeyDown(platform.KeyLeftShift) {
		t.Error("shift is reported as held before any key was pressed")
	}
	w.HandleEvent(keyDown(platform.KeyLeftShift))
	if !w.KeyDown(platform.KeyLeftShift) {
		t.Error("shift is not reported as held while it is down")
	}
	if w.HeldModifiers()&platform.ModShift == 0 {
		t.Error("HeldModifiers does not report shift while shift is held")
	}
	w.HandleEvent(keyUp(platform.KeyLeftShift))
	if w.KeyDown(platform.KeyLeftShift) {
		t.Error("shift is still reported as held after its release")
	}
	if w.HeldModifiers()&platform.ModShift != 0 {
		t.Error("HeldModifiers still reports shift after its release")
	}
}

// TestHeldModifiersIsTrueBetweenEvents is the case that motivated a second method: Modifiers() answers
// "what did THIS event carry", which a paint cannot use — a mouse move carries no modifiers, so a
// shift-held UI asked during a drag would report no shift and flicker. HeldModifiers() answers from the
// keys.
//
// (An earlier version changed Modifiers() itself to do this, and broke the node graph's shift+wheel and
// ctrl+wheel, which are exactly the handlers that read it: a wheel event carries no modifiers, so the pump
// queries the OS and the widget reads it back. Two questions, two methods.)
func TestHeldModifiersIsTrueBetweenEvents(t *testing.T) {
	w := NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 400, 300))

	w.HandleEvent(&platform.Event{Type: platform.EventKeyDown, Key: platform.KeyLeftCtrl,
		Mods: platform.ModCtrl})
	w.HandleEvent(keyDown(platform.KeyLeftCtrl))
	// A MOUSE MOVE carries no modifiers — it is the event that was breaking the answer.
	w.HandleEvent(&platform.Event{Type: platform.EventMouseMove, Pos: graphics.Point{X: 10, Y: 10}})

	if w.HeldModifiers()&platform.ModCtrl == 0 {
		t.Error("ctrl stopped being reported after a mouse move, though it is still held: HeldModifiers " +
			"is answering with the last event's bits rather than with the keys that are down")
	}
	// And Modifiers() keeps ITS contract — what the LAST EVENT carried — which is what the node graph's
	// wheel handling depends on. It reporting nothing after a modifier-less mouse move is the CONTRACT, not
	// a bug: that is the event that arrived. What a wheel does about it is the pump's job, and it puts the
	// OS's state on the wheel event so the widget can read it here.
	if w.Modifiers() != 0 {
		t.Errorf("Modifiers reports %v after a modifier-less mouse move; it must keep answering with "+
			"the last event's bits", w.Modifiers())
	}
	// A wheel event's modifiers reach a widget through here, so the contract has to carry them when the
	// event has them.
	w.HandleEvent(&platform.Event{Type: platform.EventMouseWheel, Mods: platform.ModCtrl})
	if w.Modifiers()&platform.ModCtrl == 0 {
		t.Error("a wheel event's modifiers did not survive to Modifiers(); this is the path the node " +
			"graph's ctrl+wheel reads, and losing it is what broke when Modifiers was redefined")
	}
}

// ── 4. escape ───────────────────────────────────────────────────────────

// TestEscapeClosesTheTopLayer: a popup whose contents are all non-focusable had no path to the keyboard,
// because the window walked the focused child and the children. Escape is the universal "back".
func TestEscapeClosesTheTopLayer(t *testing.T) {
	w := NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	k := &keyWidget{}
	w.LayerManager.AddLayer(0, k)

	if w.LayerManager.LayerCount() != 1 {
		t.Fatalf("the window has %d layers, want 1", w.LayerManager.LayerCount())
	}
	if !w.HandleEvent(keyDown(platform.KeyEscape)) {
		t.Error("Escape did not close the top layer")
	}
	if w.LayerManager.LayerCount() != 0 {
		t.Errorf("after Escape the window still has %d layers", w.LayerManager.LayerCount())
	}
}

// TestEscapeTellsTheWidgetItClosed: a popup that opened a list has to release it, or it leaves a list open
// behind a closed popup.
func TestEscapeTellsTheWidgetItClosed(t *testing.T) {
	w := NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	k := &closableWidget{}
	w.LayerManager.AddLayer(0, k)

	w.HandleEvent(keyDown(platform.KeyEscape))
	if !k.closed {
		t.Error("the closed layer was not told it closed")
	}
}

type closableWidget struct {
	BaseWidget
	closed bool
}

func (c *closableWidget) OnClosed() { c.closed = true }

// TestEscapeWithNothingOpenFallsThroughToTheWidget: Escape is the WINDOW's first case, but it must not
// swallow the key when there is no layer and no drag — a text box's Escape, a menu bar's Escape, a
// combo box's Escape all have to still work.
func TestEscapeWithNothingOpenFallsThroughToTheWidget(t *testing.T) {
	k := &keyWidget{consume: true}
	w := windowWith(t, k)

	w.HandleEvent(keyDown(platform.KeyEscape))
	if len(k.downs) != 1 {
		t.Errorf("with nothing open, Escape reached the focused widget %d times, want 1", len(k.downs))
	}
}
