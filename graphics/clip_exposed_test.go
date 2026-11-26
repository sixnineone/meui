package graphics

import (
	"testing"

	"github.com/sixnineone/meui/graphics/enums/pathop"
)

// TestQuickRejectRectUsesTheClip pins the culling primitive now that it is on the
// seam. A reject that is wrong in the safe direction only costs a draw; one that is
// wrong the other way makes a visible widget vanish, so the cases are asserted
// rather than the mechanism.
func TestQuickRejectRectUsesTheClip(t *testing.T) {
	c := NewCanvas(NewTestDevice(200, 200))
	c.ClipRect(NewRect(50, 50, 100, 100), pathop.Intersect, true)

	if c.QuickRejectRect(NewRect(60, 60, 10, 10)) {
		t.Error("a rect well inside the clip was rejected")
	}
	if !c.QuickRejectRect(NewRect(0, 0, 10, 10)) {
		t.Error("a rect well outside the clip was not rejected")
	}
	if !c.QuickRejectRect(NewRect(300, 300, 10, 10)) {
		t.Error("a rect past the far edge was not rejected")
	}
	// Straddling the edge must NOT be rejected: it is partly visible.
	if c.QuickRejectRect(NewRect(45, 60, 10, 10)) {
		t.Error("a rect straddling the clip edge was rejected")
	}
}

// TestClipBoundsAndIsClipEmpty pins the two introspection calls, including the
// distinction that matters: an empty clip is not the same as no clip.
func TestClipBoundsAndIsClipEmpty(t *testing.T) {
	c := NewCanvas(NewTestDevice(200, 200))
	if c.IsClipEmpty() {
		t.Error("a fresh canvas reports an empty clip")
	}
	if b := c.ClipBounds(); b.Width != 200 || b.Height != 200 {
		t.Errorf("unclipped bounds = %vx%v, want the whole 200x200 surface", b.Width, b.Height)
	}

	c.ClipRect(NewRect(50, 50, 100, 100), pathop.Intersect, true)
	if c.IsClipEmpty() {
		t.Error("a real clip reports empty")
	}
	if b := c.ClipBounds(); b.X != 50 || b.Y != 50 || b.Width != 100 || b.Height != 100 {
		t.Errorf("bounds = (%v,%v %vx%v), want (50,50 100x100)", b.X, b.Y, b.Width, b.Height)
	}

	// A second, disjoint clip intersects to nothing — empty, but still a clip.
	c.ClipRect(NewRect(500, 500, 10, 10), pathop.Intersect, true)
	if !c.IsClipEmpty() {
		t.Error("two disjoint clips did not intersect to empty")
	}
}

// TestWithClipRestoresWhateverHappens is the point of the transaction form: the
// clip must come back even though fn never restores it. The manual Save/ClipRect/
// Restore form is what leaked (the historic TextField bug).
func TestWithClipRestoresWhateverHappens(t *testing.T) {
	c := NewCanvas(NewTestDevice(200, 200))
	before := c.ClipBounds()

	c.WithClip(NewRect(10, 10, 20, 20), func(inner *Canvas) {
		if b := inner.ClipBounds(); b.Width != 20 {
			t.Errorf("inside WithClip the clip is %vx%v, want 20x20", b.Width, b.Height)
		}
		// No Restore call here on purpose.
	})

	after := c.ClipBounds()
	if after != before {
		t.Errorf("the clip was not restored: %v, want %v", after, before)
	}
}
