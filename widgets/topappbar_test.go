package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

func newTestAppBar(w float32) *TopAppBar {
	b := NewTopAppBar()
	b.SetTitle("Inbox")
	b.SetLeading("back", "Back")
	b.SetActions([]TopAppBarAction{{ID: "search", Icon: "search"}, {ID: "star", Icon: "user"}})
	b.SetOverflow(true)
	b.SetFrameRect(graphics.NewRect(0, 0, w, b.BarHeight()))
	return b
}

// TestAppBarControlsReachTheTouchFloor is the Phase A contract on the shell's most-tapped
// controls. A 24px icon in a 24px target is the classic way to make a mobile app feel
// broken, and an app bar's leading and trailing controls are exactly where it happens.
func TestAppBarControlsReachTheTouchFloor(t *testing.T) {
	saved := theme.DefaultTheme
	t.Cleanup(func() { theme.SetDefaultTheme(saved) })
	// Set the metrics AFTER SetDefaultTheme: withStyleDefaults REPLACES the whole
	// Metrics struct when ControlHeight is 0, so a metric set on the copy beforehand is
	// silently discarded — which is exactly what happened the first time.
	th := *saved
	theme.SetDefaultTheme(&th)
	theme.DefaultTheme.Metrics.MinTouchTarget = theme.TouchTargetAndroid
	theme.DefaultTheme.Metrics.ToolbarHeight = 64

	b := newTestAppBar(360)
	floor := theme.TouchTargetAndroid
	if h := b.BarHeight(); h < floor {
		t.Errorf("the bar is %.0f tall, under the %v floor", h, floor)
	}
	if r, ok := b.LeadingRect(); !ok || r.Width < floor || r.Height < floor {
		t.Errorf("the leading control is %v, under the %v floor", r, floor)
	}
	for i := range b.Actions() {
		r, ok := b.ActionRect(i)
		if !ok || r.Width < floor || r.Height < floor {
			t.Errorf("action %d is %v, under the %v floor", i, r, floor)
		}
	}
	if r, ok := b.OverflowRect(); !ok || r.Width < floor {
		t.Errorf("the overflow control is %v, under the %v floor", r, floor)
	}
}

// TestAppBarControlsDoNotOverlap guards the arithmetic that a wider bar hides: the
// actions are laid out from the RIGHT and the overflow takes the last slot, so a mistake
// there overlaps two buttons and one of them becomes unreachable.
func TestAppBarControlsDoNotOverlap(t *testing.T) {
	b := newTestAppBar(360)
	var rects []graphics.Rect
	if r, ok := b.LeadingRect(); ok {
		rects = append(rects, r)
	}
	for i := range b.Actions() {
		if r, ok := b.ActionRect(i); ok {
			rects = append(rects, r)
		}
	}
	if r, ok := b.OverflowRect(); ok {
		rects = append(rects, r)
	}
	for i := range rects {
		for j := i + 1; j < len(rects); j++ {
			if rects[i].Contains(rects[j].X+1, rects[j].Y+1) ||
				rects[j].Contains(rects[i].X+1, rects[i].Y+1) {
				t.Errorf("controls %d and %d overlap: %v and %v", i, j, rects[i], rects[j])
			}
		}
	}
	// And the title must stay clear of both ends.
	tr := b.TitleRect()
	if tr.X < rects[0].X+rects[0].Width-1 {
		t.Errorf("the title starts at %.0f, under the leading control", tr.X)
	}
	if last := rects[len(rects)-1]; tr.X+tr.Width > last.X+1 {
		t.Errorf("the title runs to %.0f, into the trailing controls at %.0f", tr.X+tr.Width, last.X)
	}
}

// TestAppBarReportsTheControlThatWasUsed pins the dispatch: leading, each action by ID,
// and the overflow with a position the app can open a menu at.
func TestAppBarReportsTheControlThatWasUsed(t *testing.T) {
	b := newTestAppBar(360)
	var leading, action string
	var overflowAt [2]float32
	b.OnLeading = func() { leading = "yes" }
	b.OnAction = func(id string) { action = id }
	b.OnOverflow = func(x, y float32) { overflowAt = [2]float32{x, y} }

	press := func(r graphics.Rect) {
		cx, cy := r.X+r.Width/2, r.Y+r.Height/2
		if !b.OnMouseDown(cx, cy, 0) {
			t.Fatalf("a press at (%.0f,%.0f) was not taken", cx, cy)
		}
		b.OnMouseUp(cx, cy, 0)
	}

	if r, ok := b.LeadingRect(); ok {
		press(r)
	}
	if leading == "" {
		t.Error("the leading control did not report")
	}
	if r, ok := b.ActionRect(1); ok {
		press(r)
	}
	if action != "star" {
		t.Errorf("action reported %q, want star (actions are laid out from the RIGHT)", action)
	}
	if r, ok := b.OverflowRect(); ok {
		press(r)
	}
	if overflowAt[1] == 0 {
		t.Error("the overflow control did not report a position to open a menu at")
	}
}

// TestAppBarIgnoresAPressOutsideItsControls keeps the bar from swallowing presses meant
// for the content behind it: an app bar is a strip, not a sheet.
func TestAppBarIgnoresAPressOutsideItsControls(t *testing.T) {
	b := newTestAppBar(360)
	tr := b.TitleRect()
	if b.OnMouseDown(tr.X+tr.Width/2, tr.Y+tr.Height/2, 0) {
		t.Error("a press on the title was taken; the title is not a control")
	}
}

var _ meui.EventListener = (*TopAppBar)(nil)
