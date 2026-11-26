// Package behaviour drives the meui widget toolkit through its event
// sink and asserts widget state — headless behavioral testing of every
// widget's expected interactions (tab switching, text editing and
// selection, tree expand/collapse, panel splitting).
//
// Events are injected via meui.Window.HandleEvent, the same entry point
// the platform layer uses, so the tests exercise the real input
// pipeline without a window server or desktop session.
package behaviour

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/qml"
	"github.com/sixnineone/meui/widgets"
)

// Harness hosts a QML-loaded widget tree for behavioral testing.
type Harness struct {
	T   *testing.T
	Win *meui.Window
	Reg *qml.BoundWidgetRegistry
}

// Load parses qmlSrc (root must be a Window), validates layout, and
// returns a harness ready to receive injected events.
func Load(t *testing.T, qmlSrc string) *Harness {
	t.Helper()
	res, err := qml.ParseWithBindings(qmlSrc)
	if err != nil {
		t.Fatalf("qml parse: %v", err)
	}
	win, ok := res.Widget.(*meui.Window)
	if !ok {
		t.Fatalf("qml root is %T, want *meui.Window", res.Widget)
	}
	win.ValidateLayout()
	return &Harness{T: t, Win: win, Reg: res.Widgets}
}

// Widget returns the widget registered under a QML id.
func (h *Harness) Widget(id string) meui.Widget {
	h.T.Helper()
	w := h.Reg.Get(id)
	if w == nil {
		h.T.Fatalf("no widget registered with id %q", id)
	}
	return w
}

// TabPanel fetches a TabPanel by id.
func (h *Harness) TabPanel(id string) *widgets.TabPanel {
	h.T.Helper()
	tp, ok := h.Widget(id).(*widgets.TabPanel)
	if !ok {
		h.T.Fatalf("widget %q is %T, want *widgets.TabPanel", id, h.Widget(id))
	}
	return tp
}

// Tree fetches a Tree by id.
func (h *Harness) Tree(id string) *widgets.Tree {
	h.T.Helper()
	tr, ok := h.Widget(id).(*widgets.Tree)
	if !ok {
		h.T.Fatalf("widget %q is %T, want *widgets.Tree", id, h.Widget(id))
	}
	return tr
}

// TextField fetches a TextField by id.
func (h *Harness) TextField(id string) *widgets.TextField {
	h.T.Helper()
	tf, ok := h.Widget(id).(*widgets.TextField)
	if !ok {
		h.T.Fatalf("widget %q is %T, want *widgets.TextField", id, h.Widget(id))
	}
	return tf
}

// TextArea fetches a TextArea by id.
func (h *Harness) TextArea(id string) *widgets.TextArea {
	h.T.Helper()
	ta, ok := h.Widget(id).(*widgets.TextArea)
	if !ok {
		h.T.Fatalf("widget %q is %T, want *widgets.TextArea", id, h.Widget(id))
	}
	return ta
}

// Send delivers one event to the window.
func (h *Harness) Send(ev *platform.Event) {
	h.T.Helper()
	h.Win.HandleEvent(ev)
}

// sendAll delivers a sequence built by input.go, so the headless and windowed
// harnesses inject byte-identical input.
func (h *Harness) sendAll(evs []*platform.Event) {
	for _, ev := range evs {
		h.Send(ev)
	}
}

// Click presses and releases button 0 at window coords (x, y).
func (h *Harness) Click(x, y float32) { h.sendAll(clickEvents(x, y)) }

// DblClick double-clicks at (x, y) — the second press carries ClickCount 2.
func (h *Harness) DblClick(x, y float32) { h.sendAll(dblClickEvents(x, y)) }

// Drag presses at (x1, y1), moves in steps, releases at (x2, y2).
func (h *Harness) Drag(x1, y1, x2, y2 float32) { h.sendAll(dragEvents(x1, y1, x2, y2)) }

// Wheel sends a wheel event at (x, y) with deltas (dx, dy).
func (h *Harness) Wheel(x, y, dx, dy float32) { h.Send(wheelEvent(x, y, dx, dy)) }

// KeyDown presses a key (no release — key-up has no behavior yet).
func (h *Harness) KeyDown(k platform.KeyCode, mods platform.Modifiers) {
	h.Send(keyDownEvent(k, mods))
}

// Type sends rune events to the focused widget.
func (h *Harness) Type(s string) { h.sendAll(runeEvents(s)) }

// Center returns the center of a widget's frame in window coordinates.
func Center(w meui.Widget) (float32, float32) {
	f := w.FrameRect()
	return f.X + f.Width/2, f.Y + f.Height/2
}

// MustEqual fails the test when got != want, with a formatted message.
func MustEqual(t *testing.T, what string, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", what, got, want)
	}
}

// PaintOnce drives one paint pass over the widget tree. Paint-time
// behaviors (popup creation, toolbar height self-correction) run here —
// call it after clicks when a widget defers work to its Paint.
func (h *Harness) PaintOnce() {
	c := graphics.NewCanvas(graphics.NewTestDevice(1280, 800))
	h.Win.Paint(c, graphics.NewRect(0, 0, 1280, 800))
}

// Settle runs a paint + layout cycle, mirroring what the live window
// loop does every frame. Call between interactions when a widget defers
// work to paint time.
func (h *Harness) Settle() {
	h.PaintOnce()
	h.Win.ValidateLayout()
	h.PaintOnce()
}
