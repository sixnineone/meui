package meui

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
)

// SetOnMouseDown stored its handler and nothing ever called it, so the public
// hook was dead: a caller wiring it saw no presses at all, which reads as "input
// is broken". It is a window-level notification and must fire for every press,
// including one a child widget consumes — handleMouseDown returns true as soon
// as it hits a child, so a fall-through placement would still never run.
func TestWindowOnMouseDownFires(t *testing.T) {
	w := NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 200, 200))

	fired := 0
	w.SetOnMouseDown(func(x, y float32, button int) { fired++ })

	w.handleMouseDown(10, 10, 1)
	if fired != 1 {
		t.Fatalf("handler fired %d times for an unhandled press, want 1", fired)
	}

	// Also for a press that lands on a child.
	child := NewPanel()
	child.SetFrameRect(graphics.NewRect(0, 0, 200, 200))
	w.AddChild(child)
	fired = 0
	w.handleMouseDown(50, 50, 1)
	if fired != 1 {
		t.Fatalf("handler fired %d times for a press on a child, want 1", fired)
	}
}
