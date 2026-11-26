package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

func newTestRail() *NavigationRail {
	r := NewNavigationRail()
	r.SetItems([]NavigationBarItem{
		{ID: "home", Label: "Home", Icon: "document"},
		{ID: "search", Label: "Search", Icon: "search"},
		{ID: "settings", Label: "Settings", Icon: "user"},
	})
	r.SetFrameRect(graphics.NewRect(0, 0, r.RailWidth(), 400))
	return r
}

// TestRailAndBarShareTheSelectionContract is the point of having both: the two
// presentations of the same control must not drift apart. An app that swaps one for the
// other when the window changes size should not have to learn a second set of rules.
func TestRailAndBarShareTheSelectionContract(t *testing.T) {
	rail, bar := newTestRail(), newTestNavBar()

	for _, x := range []struct {
		widget interface {
			SetSelected(int) bool
			Selected() int
			Count() int
		}
		name string
	}{{rail, "rail"}, {bar, "bar"}} {
		if x.widget.SetSelected(2) != true || x.widget.Selected() != 2 {
			t.Errorf("%s: SetSelected(2) did not take", x.name)
		}
		if x.widget.SetSelected(99) {
			t.Errorf("%s: SetSelected accepted an out-of-range index", x.name)
		}
		if x.widget.SetSelected(-1) {
			t.Errorf("%s: SetSelected accepted a negative index", x.name)
		}
		if x.widget.Count() != 3 {
			t.Errorf("%s: Count is %d, want 3", x.name, x.widget.Count())
		}
	}
}

// TestRailSelectsOnRelease matches the bottom bar: the selection moves when the pointer
// lifts, and only if it is still on the same destination.
func TestRailSelectsOnRelease(t *testing.T) {
	r := newTestRail()
	moved := 0
	r.OnSelectionChanged = func(int) { moved++ }

	r2, ok := r.ItemRect(2)
	if !ok {
		t.Fatal("no rect for item 2")
	}
	cy := r2.Y + r2.Height/2
	if !r.OnMouseDown(20, cy, 0) {
		t.Fatal("a press on a destination was not taken")
	}
	if moved != 0 {
		t.Error("the selection moved on PRESS")
	}
	r.OnMouseUp(20, cy, 0)
	if r.Selected() != 2 {
		t.Errorf("selected %d, want 2", r.Selected())
	}

	// Press on 0, release on 1: cancels.
	r0, _ := r.ItemRect(0)
	r1, _ := r.ItemRect(1)
	r.OnMouseDown(20, r0.Y+r0.Height/2, 0)
	r.OnMouseUp(20, r1.Y+r1.Height/2, 0)
	if r.Selected() != 2 {
		t.Errorf("a release on a different destination moved the selection to %d", r.Selected())
	}
}

// TestRailDestinationsReachTheTouchFloor applies the Phase A contract to the side
// presentation: the same targets, rotated.
func TestRailDestinationsReachTheTouchFloor(t *testing.T) {
	saved := theme.DefaultTheme
	t.Cleanup(func() { theme.SetDefaultTheme(saved) })
	th := *saved
	theme.SetDefaultTheme(&th)
	theme.DefaultTheme.Metrics.MinTouchTarget = theme.TouchTargetAndroid
	theme.DefaultTheme.Metrics.NavRailWidth = 80

	r := newTestRail()
	floor := theme.TouchTargetAndroid
	if h := r.ItemHeight(); h < floor {
		t.Errorf("a destination is %.0f tall, under the %v floor", h, floor)
	}
	for i := 0; i < r.Count(); i++ {
		ir, ok := r.ItemRect(i)
		if !ok {
			t.Fatalf("item %d has no rect", i)
		}
		if ir.Width < floor || ir.Height < floor {
			t.Errorf("destination %d is %.0fx%.0f, under the %v floor", i, ir.Width, ir.Height, floor)
		}
	}
}

// TestRailDestinationsAreTallEnoughForTheirOwnContent is the assertion the rail's other
// tests were missing, and it is the one that caught a real defect found by RENDERING the
// tablet form factor: ItemHeight was max(TabHeight, MinTouchTarget), so on a desktop
// theme (TabHeight 28, no touch floor) each destination was 28px while its icon, the
// indicator pill around it and the label beneath it needed ~54. Every label was drawn on
// top of the NEXT destination's icon. The Android theme's 64px hid it, so it only showed
// at a tablet-sized window on a desktop theme.
func TestRailDestinationsAreTallEnoughForTheirOwnContent(t *testing.T) {
	saved := theme.DefaultTheme
	t.Cleanup(func() { theme.SetDefaultTheme(saved) })
	th := *saved
	theme.SetDefaultTheme(&th)
	// Set AFTER SetDefaultTheme: withStyleDefaults replaces Metrics wholesale when
	// ControlHeight is zero, so anything set on the copy beforehand is discarded.
	theme.DefaultTheme.Metrics.MinTouchTarget = 0

	r := newTestRail()
	// The exact failing condition, pinned on the widget rather than the ambient theme:
	// a short tab height and no touch floor.
	r.TabTheme.TabHeight = 28

	if h, want := r.ItemHeight(), r.contentHeight(); h < want {
		t.Errorf("a destination is %.0f tall but its own content needs %.0f", h, want)
	}
	// The regression stated directly: the old formula returned the tab height.
	if h := r.ItemHeight(); h <= 28 {
		t.Errorf("a destination is %.0f tall — the tab height again, not the content", h)
	}
	// And what the user actually sees: consecutive destinations must not overlap.
	for i := 1; i < r.Count(); i++ {
		prev, _ := r.ItemRect(i - 1)
		cur, _ := r.ItemRect(i)
		if cur.Y < prev.Y+prev.Height {
			t.Errorf("destination %d starts at %.0f, inside destination %d which ends at %.0f",
				i, cur.Y, i-1, prev.Y+prev.Height)
		}
	}
}

// TestRailAndBarAgreeOnWhichDestinationWasHit guards the hit arithmetic, which differs
// between them (a rail divides by HEIGHT, the bar by WIDTH) and is the easiest thing to
// get wrong when adding the second one.
func TestRailAndBarAgreeOnWhichDestinationWasHit(t *testing.T) {
	rail := newTestRail()
	rail.SetFrameRect(graphics.NewRect(0, 0, rail.RailWidth(), 400))
	for i := 0; i < rail.Count(); i++ {
		ir, _ := rail.ItemRect(i)
		if got := rail.ItemAt(ir.X+ir.Width/2, ir.Y+ir.Height/2); got != i {
			t.Errorf("rail: the centre of destination %d reports %d", i, got)
		}
	}
	// Outside the rail entirely.
	if got := rail.ItemAt(rail.RailWidth()+10, 100); got != -1 {
		t.Errorf("rail: a point outside the rail reports %d, want -1", got)
	}

	bar := newTestNavBar()
	for i := 0; i < bar.Count(); i++ {
		br, _ := bar.ItemRect(i)
		if got := bar.ItemAt(br.X+br.Width/2, br.Y+br.Height/2); got != i {
			t.Errorf("bar: the centre of destination %d reports %d", i, got)
		}
	}
}

var _ meui.EventListener = (*NavigationRail)(nil)
