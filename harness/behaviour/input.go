package behaviour

import (
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
)

// Event builders shared by the headless Harness and the windowed VisualApp.
//
// Both harnesses drive the same input pipeline — meui.Window.HandleEvent for
// the headless one, the SDL pump for the windowed one — so the events they
// synthesize have to be identical. They used to be written out twice and had
// drifted: VisualApp had no Drag, Wheel or KeyDown at all, and the two Click
// implementations could disagree without any test noticing.

func mouseDown(x, y float32, clickCount int) *platform.Event {
	return &platform.Event{
		Type:       platform.EventMouseDown,
		Pos:        graphics.Point{X: x, Y: y},
		Button:     0,
		ClickCount: clickCount,
	}
}

func mouseUp(x, y float32, clickCount int) *platform.Event {
	return &platform.Event{
		Type:       platform.EventMouseUp,
		Pos:        graphics.Point{X: x, Y: y},
		Button:     0,
		ClickCount: clickCount,
	}
}

// clickEvents is a press and release at (x, y).
func clickEvents(x, y float32) []*platform.Event {
	return []*platform.Event{mouseDown(x, y, 1), mouseUp(x, y, 1)}
}

// dblClickEvents is two clicks, the second carrying ClickCount 2.
func dblClickEvents(x, y float32) []*platform.Event {
	return []*platform.Event{
		mouseDown(x, y, 1), mouseUp(x, y, 1),
		mouseDown(x, y, 2), mouseUp(x, y, 2),
	}
}

// dragEvents presses at (x1, y1), moves in steps, and releases at (x2, y2).
// The press carries no click count, matching a real drag (the window's own
// click counter decides double-clicks).
func dragEvents(x1, y1, x2, y2 float32) []*platform.Event {
	const steps = 8
	evs := []*platform.Event{mouseDown(x1, y1, 0)}
	for i := 1; i <= steps; i++ {
		t := float32(i) / float32(steps)
		evs = append(evs, &platform.Event{
			Type:   platform.EventMouseDrag,
			Pos:    graphics.Point{X: x1 + (x2-x1)*t, Y: y1 + (y2-y1)*t},
			Button: 0,
		})
	}
	return append(evs, mouseUp(x2, y2, 0))
}

// wheelEvent is a wheel event at (x, y) with deltas (dx, dy).
func wheelEvent(x, y, dx, dy float32) *platform.Event {
	return &platform.Event{
		Type:   platform.EventMouseWheel,
		Pos:    graphics.Point{X: x, Y: y},
		DeltaX: dx,
		DeltaY: dy,
	}
}

// keyDownEvent presses a key (key-up has no behaviour yet).
func keyDownEvent(k platform.KeyCode, mods platform.Modifiers) *platform.Event {
	return &platform.Event{Type: platform.EventKeyDown, Key: k, Mods: mods}
}

// runeEvents types a string as rune events to the focused widget.
func runeEvents(s string) []*platform.Event {
	evs := make([]*platform.Event, 0, len(s))
	for _, r := range s {
		evs = append(evs, &platform.Event{Type: platform.EventRuneTyped, Rune: r})
	}
	return evs
}
