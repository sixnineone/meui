package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

func newTestNavBar() *NavigationBar {
	nb := NewNavigationBar()
	nb.SetItems([]NavigationBarItem{
		{ID: "home", Label: "Home", Icon: "document"},
		{ID: "search", Label: "Search", Icon: "search"},
		{ID: "settings", Label: "Settings", Icon: "user"},
	})
	nb.SetFrameRect(graphics.NewRect(0, 0, 360, nb.BarHeight()))
	return nb
}

// TestNavigationBarSelectsOnRelease pins the interaction every platform uses and the
// one that makes a mis-touch recoverable: the selection moves when the finger LIFTS,
// and only if it is still on the same destination. Selecting on press means a drag that
// leaves the bar still changes the page.
func TestNavigationBarSelectsOnRelease(t *testing.T) {
	nb := newTestNavBar()
	var got []int
	nb.OnSelectionChanged = func(i int) { got = append(got, i) }

	w := nb.Width() / 3
	// Press on the second destination, release on it: selects.
	if !nb.OnMouseDown(w+w/2, 10, 0) {
		t.Fatal("a press on a destination was not taken")
	}
	if len(got) != 0 {
		t.Error("the selection moved on PRESS; it must wait for the release")
	}
	nb.OnMouseUp(w+w/2, 10, 0)
	if nb.Selected() != 1 {
		t.Errorf("selected %d, want 1", nb.Selected())
	}

	// Press on the third, release outside the bar: cancels.
	nb.OnMouseDown(2*w+w/2, 10, 0)
	nb.OnMouseUp(2*w+w/2, nb.BarHeight()+40, 0)
	if nb.Selected() != 1 {
		t.Errorf("a release outside the bar moved the selection to %d", nb.Selected())
	}
}

// TestNavigationBarReportsSelectionOnce guards the callback contract: a selection that
// does not MOVE must not report, or an app that loads a page on the callback reloads it
// every time the user taps the destination they are already on.
func TestNavigationBarReportsSelectionOnce(t *testing.T) {
	nb := newTestNavBar()
	calls := 0
	nb.OnSelectionChanged = func(int) { calls++ }
	nb.SetSelected(0) // already selected
	if calls != 0 {
		t.Errorf("re-selecting the current destination reported %d times, want 0", calls)
	}
	nb.SetSelected(2)
	nb.SetSelected(2)
	if calls != 1 {
		t.Errorf("reported %d times for one move, want 1", calls)
	}
}

// TestNavigationBarReachesTheTouchFloor is the Phase A contract applied to the new
// widget: three to five destinations across a phone must each be a finger target.
func TestNavigationBarReachesTheTouchFloor(t *testing.T) {
	// Set the metric on a COPY rather than loading a preset: theme.LoadPreset mutates
	// the shared DefaultTabTheme as well, and restoring only DefaultTheme leaves the
	// tab height changed for every later test in the package — which showed up as a
	// panic in an unrelated pixel comparison.
	saved := theme.DefaultTheme
	t.Cleanup(func() { theme.SetDefaultTheme(saved) })
	th := *saved
	theme.SetDefaultTheme(&th)
	// After, not before: see the note in topappbar_test.go.
	theme.DefaultTheme.Metrics.MinTouchTarget = theme.TouchTargetAndroid

	nb := NewNavigationBar()
	nb.TabTheme.TabHeight = theme.TouchTargetAndroid * 4 / 3 // 64: the Material bottom bar
	nb.SetItems([]NavigationBarItem{{Label: "A"}, {Label: "B"}, {Label: "C"}, {Label: "D"}, {Label: "E"}})
	nb.SetFrameRect(graphics.NewRect(0, 0, 320, nb.BarHeight()))

	floor := theme.TouchTargetAndroid
	for i := 0; i < nb.Count(); i++ {
		r, ok := nb.ItemRect(i)
		if !ok {
			t.Fatalf("item %d has no rect", i)
		}
		if r.Width < floor {
			t.Errorf("five destinations on a 320px bar gives %.0f each, under the %v floor — "+
				"the bar's own range is three to five for this reason", r.Width, floor)
		}
	}
	if nb.BarHeight() < floor {
		t.Errorf("the bar is %.0f tall, under the %v floor", nb.BarHeight(), floor)
	}
}

var _ meui.EventListener = (*NavigationBar)(nil)

// TestHidingATabPanelsBarKeepsItAsThePageHost is what a phone shell needs: the
// destinations move to a NavigationBar at the bottom, so the panel must stop drawing
// its own top strip WITHOUT stopping being the thing that switches pages.
func TestHidingATabPanelsBarKeepsItAsThePageHost(t *testing.T) {
	const w, h = 320, 200
	p := NewTabPanel()
	p.SetFrameRect(graphics.NewRect(0, 0, w, h))
	for _, l := range []string{"A", "B", "C"} {
		p.AddTab(l, meui.NewPanel())
	}
	if !p.TabsVisible() {
		t.Fatal("a new TabPanel hid its tab strip; the default must be visible")
	}

	p.SetTabsVisible(false)
	if p.TabsVisible() {
		t.Error("SetTabsVisible(false) did not take")
	}
	if bh := p.tabBarHeight(); bh != 0 {
		t.Errorf("the hidden bar still reports a height of %v, so the content is laid out below nothing", bh)
	}

	// The page must still fill the panel and still switch.
	dev := graphics.NewSoftDevice(w, h)
	p.Paint(graphics.NewCanvas(dev), graphics.NewRect(0, 0, w, h))
	p.SetActiveTab(2)
	if p.ActiveTab() != 2 {
		t.Errorf("the panel stopped switching pages with its bar hidden: active %d", p.ActiveTab())
	}

	// And it must come back.
	p.SetTabsVisible(true)
	if p.tabBarHeight() == 0 {
		t.Error("the tab strip did not come back")
	}
}
