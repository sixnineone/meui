package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/layout"
)

// TestFrameHitMiddleWidgetsClickable pins the drag-area contract: the
// bar withholds the drag on a fresh press (NORMAL) so a second press
// can register as a double-click; motion beyond a threshold flips to
// DRAGGABLE for the WM move. Middle widgets being NORMAL keeps their
// clicks interactive.
func TestFrameHitMiddleWidgetsClickable(t *testing.T) {
	// Backend seam: X11 mode first (app handles bar presses itself).
	x11 := false
	barBackendDraggable = func() bool { return x11 }
	defer func() { barBackendDraggable = func() bool { return sdl3.VideoDriver() != "x11" } }()

	f := NewWindowFrame()

	host := meui.NewPanel()
	host.SetLayout(layout.NewHStack())
	btn := NewButton()
	btn.SetText("New Window")
	btn.SetWidthDirect(90)
	btn.SetHeightDirect(FrameBarHeight - 4)
	combo := NewComboBox()
	combo.SetWidthDirect(190)
	combo.SetHeightDirect(FrameBarHeight - 6)
	host.AddChild(btn)
	host.AddChild(combo)

	f.SetMiddle(host)
	f.layout(800)

	// The combo (last child) must sit inside the middle host.
	hf := host.FrameRect()
	cf := combo.FrameRect()
	if cf.Width <= 0 || cf.Height <= 0 {
		t.Fatalf("combo has no frame: %v", cf)
	}
	cx := hf.X + cf.X + cf.Width/2
	cy := hf.Y + cf.Y + cf.Height/2

	if got := f.FrameHit(cx, cy, 800, 600); got != 0 {
		t.Errorf("FrameHit over combo = %d (1=DRAGGABLE); want 0 (NORMAL)", got)
	}
	if got := f.FrameHit(hf.X+btn.FrameRect().X+10, hf.Y+btn.FrameRect().Y+5, 800, 600); got != 0 {
		t.Errorf("FrameHit over button = %d; want 0", got)
	}
	// Empty bar space: X11 mode classifies NORMAL (the app moves the
	// window manually and detects double-clicks itself); other
	// backends use the native WM drag (DRAGGABLE).
	if got := f.FrameHit(400, FrameBarHeight/2, 800, 600); got != 0 {
		t.Errorf("FrameHit on empty bar (x11) = %d; want 0 (NORMAL)", got)
	}
	x11 = true // Wayland-style backend
	if got := f.FrameHit(400, FrameBarHeight/2, 800, 600); got != 1 {
		t.Errorf("FrameHit on empty bar (wayland) = %d; want 1 (DRAGGABLE)", got)
	}
	x11 = false

	// App-side double-click path: BarDoubleClick on the plain bar
	// toggles (deduped); buttons and middle widgets decline.
	toggled := 0
	f.OnMaximize(func() { toggled++ })
	if !f.BarDoubleClick(400, FrameBarHeight/2) {
		t.Errorf("BarDoubleClick on empty bar = false; want true")
	}
	if toggled != 1 {
		t.Errorf("double-click toggle count = %d; want 1", toggled)
	}
	if f.BarDoubleClick(400, FrameBarHeight/2) {
		// Still consumed (plain bar), but the toggle itself dedupes.
	}
	if toggled != 1 {
		t.Errorf("second BarDoubleClick re-toggled; count = %d; want 1", toggled)
	}

	// Middle parenting: the host must chain to the bar node so
	// findWindow/PointToRoot reach the window root.
	if host.Parent() != f.bar {
		t.Errorf("middle host parent = %v; want the bar node", host.Parent())
	}

	// The middle hit-test used by FrameHit agrees.
	if !f.middleHitTest(cx-hf.X, cy-hf.Y) {
		t.Errorf("middleHitTest over combo = false; want true")
	}
	if f.middleHitTest(400-hf.X, FrameBarHeight/2-hf.Y) {
		t.Errorf("middleHitTest on empty bar = true; want false")
	}
}
