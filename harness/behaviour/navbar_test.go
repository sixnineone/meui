package behaviour

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
	"github.com/sixnineone/meui/widgets"
)

// TestTheNavigationBarRespondsToATap is the test Phase B should have had from the start.
//
// A widget whose listener signature drifts from meui.EventListener compiles, paints, and is
// DEAD TO INPUT — AGENTS.md records that shipping, and the guard for it is a registry walk in
// the qml package. So the bar being in the interactive registry is not evidence it works: the
// only evidence is a tap that changes something, through the same HandleEvent the platform
// uses.
func TestTheNavigationBarRespondsToATap(t *testing.T) {
	h := Load(t, `
Window { width: 400; height: 700
  VStack { anchors.fill: parent; spacing: 0
    NavigationBar { id: nav; Layout.fillWidth: true; height: 64 }
    Label { id: page; text: "first"; width: 200; height: 20 }
  }
}`)
	nav, ok := h.Widget("nav").(*widgets.NavigationBar)
	if !ok {
		t.Fatalf("the navigation bar is a %T", h.Widget("nav"))
	}
	nav.SetItems([]widgets.NavigationBarItem{
		{ID: "one", Label: "One"},
		{ID: "two", Label: "Two"},
		{ID: "three", Label: "Three"},
	})
	// The bar is the FIRST child, so the VStack puts it at the top — and any frame set
	// here is overwritten by the layout, which is the point of the first version of this
	// test failing: it placed the bar at y=600 and clicked at a bar-local y, so the click
	// landed on the top of the window and read as "the widget is dead".
	nav.SetFrameRect(graphics.NewRect(0, 0, 400, 64))

	reported := -1
	nav.OnSelectionChanged = func(i int) { reported = i }

	// Three destinations across 400px, so the second is centred at x=200 and the bar is
	// 64 tall at the top.
	h.Click(200, 32)
	if nav.Selected() != 1 {
		t.Errorf("a tap on the second destination selected %d, want 1", nav.Selected())
	}
	if reported != 1 {
		t.Errorf("the selection callback reported %d, want 1", reported)
	}

	// And the FIRST: a bar that only responds past its first destination is a bar whose
	// first item is unreachable.
	h.Click(60, 32)
	if nav.Selected() != 0 {
		t.Errorf("a tap on the first destination selected %d, want 0", nav.Selected())
	}
}

// TestTheNavigationBarReachesTheTouchFloorWithFiveDestinations: the platform's own range is
// three to five, and five across a phone is the tight case — the one where a bar that simply
// divided its width would put a destination under the floor.
func TestTheNavigationBarReachesTheTouchFloorWithFiveDestinations(t *testing.T) {
	saved := theme.DefaultTheme
	t.Cleanup(func() { theme.SetDefaultTheme(saved) })
	th := *saved
	theme.SetDefaultTheme(&th)
	theme.DefaultTheme.Metrics.MinTouchTarget = theme.TouchTargetAndroid

	nav := widgets.NewNavigationBar()
	nav.SetItems([]widgets.NavigationBarItem{
		{ID: "a", Label: "A"}, {ID: "b", Label: "B"}, {ID: "c", Label: "C"},
		{ID: "d", Label: "D"}, {ID: "e", Label: "E"},
	})
	nav.SetFrameRect(graphics.NewRect(0, 0, 320, 64))
	for i := 0; i < nav.Count(); i++ {
		r, ok := nav.ItemRect(i)
		if !ok {
			t.Fatalf("no rect for destination %d", i)
		}
		if r.Width < theme.TouchTargetAndroid {
			t.Errorf("five destinations on a 320px bar give %.0f each, under the %v floor",
				r.Width, theme.TouchTargetAndroid)
		}
	}
	if nav.BarHeight() < theme.TouchTargetAndroid {
		t.Errorf("the bar is %.0f tall, under the %v floor", nav.BarHeight(), theme.TouchTargetAndroid)
	}
}
