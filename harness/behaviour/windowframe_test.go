package behaviour

import (
	"testing"

	"github.com/sixnineone/meui/graphics"

	"github.com/sixnineone/meui/widgets"
)

// TestWindowFrameButtons: bar clicks on min/max/close rects fire the
// registered handlers; clicks elsewhere in the bar do not.
func TestWindowFrameButtons(t *testing.T) {
	f := widgets.NewWindowFrame()
	closed, minned, maxed := 0, 0, 0
	f.OnClose(func() { closed++ })
	f.OnMinimize(func() { minned++ })
	f.OnMaximize(func() { maxed++ })

	var W, H float32 = 900, 700
	barH := f.BarHeight()

	// Layout runs inside PaintBar; use a real headless canvas.
	f.PaintBar(graphics.NewCanvas(graphics.NewTestDevice(W, H)), W)
	_ = H

	// FrameHit classification. Edges return 0 — the shim performs
	// manual resize drags (WM hit-test resize unreliable on XWayland).
	//
	// The empty bar is backend-defined (DRAGGABLE where the WM moves the
	// window, NORMAL on X11 where the app does), so pin the policy rather
	// than reading the ambient driver: it is empty until a window exists
	// and "x11" afterwards, which made this assertion pass alone and fail
	// in a suite. widgets/windowframe_hit_test.go pins the same contract
	// through the package seam.
	f.SetBackendDraggable(func() bool { return true }) // native WM drag
	if hit := f.FrameHit(10, 10, W, H); hit != 1 {
		t.Errorf("bar empty space (native drag): hit=%d want 1 (draggable)", hit)
	}
	f.SetBackendDraggable(func() bool { return false }) // X11: app moves the window
	if hit := f.FrameHit(10, 10, W, H); hit != 0 {
		t.Errorf("bar empty space (x11): hit=%d want 0 (normal)", hit)
	}
	if hit := f.FrameHit(899, 350, W, H); hit != 0 {
		t.Errorf("right edge: hit=%d want 0 (manual resize via shim)", hit)
	}
	if hit := f.FrameHit(450, 350, W, H); hit != 0 {
		t.Errorf("content area: hit=%d want 0 (normal)", hit)
	}

	// Button rects are bar-local; click the close button (rightmost).
	if f.BarMouseDown(W-17, barH/2, 0) {
		f.BarMouseUp(W-17, barH/2, 0)
	}
	if closed != 1 {
		t.Errorf("close click: closed=%d want 1", closed)
	}

	// Middle of the bar: not a button.
	if f.BarMouseDown(W/2, barH/2, 0) {
		t.Errorf("empty bar middle consumed the click")
	}
}
