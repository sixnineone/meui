package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
)

// Clicking a tab header must switch to that tab. TabHeaderRect exposes the same
// rect the painter uses, so this drives the hit test exactly where the tab is
// drawn rather than guessing screen coordinates.
func TestTabHeaderClickSwitchesTab(t *testing.T) {
	tp := NewTabPanel()
	tp.SetFrameRect(graphics.NewRect(100, 50, 600, 400))
	tp.AddTab("First", meui.NewPanel())
	tp.AddTab("Second", meui.NewPanel())
	tp.SetActiveTab(0)

	r, ok := tp.TabHeaderRect(1)
	if !ok {
		t.Fatal("TabHeaderRect(1) not found")
	}
	if !tp.OnMouseDown(r.X+2, r.Y+2, 0) {
		t.Fatal("click on the second tab header was not consumed")
	}
	if got := tp.ActiveTab(); got != 1 {
		t.Fatalf("active tab = %d after clicking header 1, want 1", got)
	}
}

// Widgets inside a TabPanel's content must sit in the same coordinate space the
// input path assumes: PointFromRoot subtracts each ancestor's FrameRect, which is
// only correct when FrameRect is in ROOT coordinates. This checks the nesting the
// Settings Charts tab uses (outer tab -> panel -> inner tab -> panel -> combo) by
// asserting PointToRoot and FrameRect agree about where the widget is.
func TestNestedTabPanelContentCoordinatesAgree(t *testing.T) {
	win := meui.NewWindow()

	deepPanel := meui.NewPanel()
	combo := NewComboBox()
	combo.SetOptions([]string{"M1", "ZM1"})
	deepPanel.AddChild(combo)

	inner := NewTabPanel()
	inner.AddTab("Grid", meui.NewPanel())
	inner.AddTab("Candles", deepPanel)

	page := meui.NewPanel()
	page.AddChild(inner)

	outer := NewTabPanel()
	outer.AddTab("Theme", meui.NewPanel())
	outer.AddTab("Charts", page)

	win.AddChild(outer)
	outer.SetFrameRect(graphics.NewRect(0, 0, 760, 640))
	// Lay the tree out the way a paint pass would, then inspect the frames.
	outer.ValidateLayout()
	page.ValidateLayout()
	inner.ValidateLayout()

	root := combo.PointToRoot(0, 0)
	fr := combo.FrameRect()
	if root.X != fr.X || root.Y != fr.Y {
		t.Fatalf("coordinate space mismatch for a nested content widget: "+
			"PointToRoot(0,0)=(%.0f,%.0f) but FrameRect origin=(%.0f,%.0f) - "+
			"input would be delivered at the wrong place",
			root.X, root.Y, fr.X, fr.Y)
	}
}
