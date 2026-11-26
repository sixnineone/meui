package widgets

// The virtual list's contract: only the rows the viewport can show are BUILT, and the
// window follows the scroll. A list that builds everything is the defect this widget
// exists to remove, so the assertions are about what exists, not what paints.

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

func newTestList(t *testing.T, count int) (*VirtualList, map[int]int) {
	t.Helper()
	theme.SetDefaultTheme(theme.DefaultTheme)
	vl := NewVirtualList()
	vl.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	vl.SetRowHeight(50)
	built := make(map[int]int)
	vl.SetRowBuilder(func(i int) meui.Widget {
		built[i]++
		p := meui.NewPanel()
		p.SetBackground(graphics.Color(0xFFFFFFFF))
		return p
	})
	vl.SetRowCount(count)
	vl.Refresh()
	return vl, built
}

func TestOnlyTheVisibleRowsAreBuilt(t *testing.T) {
	const count = 1000
	vl, built := newTestList(t, count)

	// 300px viewport / 50px rows = 6 visible rows, +1 partial, +2 overscan each side.
	if n := len(built); n > 12 {
		t.Fatalf("built %d rows for a 300px viewport: the list is not virtualising", n)
	}
	if vl.RowWidget(0) == nil {
		t.Fatal("row 0 was not built")
	}
	if vl.RowWidget(count-1) != nil {
		t.Fatal("the last row was built before it could be seen")
	}
	if got, want := len(vl.stage.Children()), len(built); got != want {
		t.Fatalf("stage holds %d rows, builder ran %d times", got, want)
	}
	// Row geometry: index 1 sits one row height down.
	if fr := vl.RowWidget(1).FrameRect(); fr.Y != 50 || fr.Height != 50 || fr.Width != 400 {
		t.Fatalf("row 1 frame = %v, want y=50 h=50 w=400", fr)
	}
}

func TestScrollingMovesTheBuiltWindow(t *testing.T) {
	const count = 1000
	vl, built := newTestList(t, count)
	firstBefore := built
	if _, ok := firstBefore[500]; ok {
		t.Fatal("row 500 was built at the top")
	}

	vl.EnsureVisible(500)
	vl.Refresh()
	if vl.RowWidget(500) == nil {
		t.Fatal("the row the list was asked to reveal was not built")
	}
	if vl.ScrollY() == 0 {
		t.Fatal("EnsureVisible did not scroll")
	}
	if vl.RowWidget(0) != nil {
		t.Fatal("row 0 is still built after scrolling 500 rows away")
	}
	// And the window is still small: scrolling must not accumulate rows.
	if n := len(built); n > 2*12 {
		t.Fatalf("built %d rows after scrolling: rows accumulate instead of being dropped", n)
	}
}

func TestEnsureVisibleRevealsABelowTheFoldRow(t *testing.T) {
	vl := NewVirtualList()
	theme.SetDefaultTheme(theme.DefaultTheme)
	vl.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	vl.SetRowHeight(50)
	vl.SetRowCount(100)
	vl.SetRowBuilder(func(int) meui.Widget { return meui.NewPanel() })
	vl.Refresh()

	vl.EnsureVisible(90)
	vl.Refresh()
	top := float32(90) * 50
	off := vl.ScrollY()
	if off != top+50-300 {
		t.Fatalf("revealing row 90 scrolled to %.0f, want %.0f", off, top+50-300)
	}
	if vl.RowWidget(90) == nil {
		t.Fatal("the revealed row was not built")
	}
}

func TestTopInsetOffsetsRowsAndReveals(t *testing.T) {
	vl := NewVirtualList()
	theme.SetDefaultTheme(theme.DefaultTheme)
	vl.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	vl.SetRowHeight(50)
	vl.SetTopInset(40)
	vl.SetRowCount(100)
	vl.SetRowBuilder(func(int) meui.Widget { return meui.NewPanel() })
	vl.Refresh()

	if fr := vl.RowWidget(0).FrameRect(); fr.Y != 40 {
		t.Fatalf("row 0 starts at y=%.0f, want 40 (below the sticky header)", fr.Y)
	}
	// Revealing row 0 with the header down keeps it below the glass.
	vl.EnsureVisible(90)
	vl.Refresh()
	if got, want := vl.ScrollY(), float32(40+90*50+50-300); got != want {
		t.Fatalf("reveal scrolled to %.0f, want %.0f", got, want)
	}
}
