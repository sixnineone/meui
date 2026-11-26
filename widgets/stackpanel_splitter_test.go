package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

// StackPanel painted its own splitter with the old tokens: a band filled with
// a lightened panel background, plus three full-length grip lines. Once
// BackgroundInk went transparent and GripInk defaulted to 0, a StackPanel's
// horizontal dividers painted nothing at all — the owner could see the
// vertical dividers (SplitView) and not the horizontal ones. Both widgets now
// share one painter, pinned here.
func stackPanelFixture(t *testing.T) (*StackPanel, *theme.StackPanelTheme) {
	t.Helper()
	saved := theme.DefaultStackPanelTheme
	t.Cleanup(func() { theme.DefaultStackPanelTheme = saved })
	th := &theme.StackPanelTheme{
		BackgroundInk:    0x00000000,
		SplitterInk:      graphics.Color(0xFF3366AA),
		SplitterHoverInk: graphics.Color(0xFFCC5533),
		GripInk:          0,
		GripSize:         4,
		SplitterSize:     4,
	}
	theme.DefaultStackPanelTheme = th
	sp := NewStackPanel("vertical")
	sp.StackTheme = th
	sp.AddItem(meui.NewPanel(), 100)
	sp.AddItem(meui.NewPanel(), 100)
	sp.SetFrameRect(graphics.NewRect(0, 0, 200, 206))
	sp.ValidateLayout()
	return sp, th
}

// stackPanelPaint paints the panel on demand: the hover ink changes between
// paints, so a snapshot taken once would not see it.
func stackPanelPaint(t *testing.T, sp *StackPanel) func() func(x, y int) uint32 {
	t.Helper()
	const w, h = 200, 206
	return func() func(x, y int) uint32 {
		dev := graphics.NewSoftDevice(w, h)
		sp.Paint(graphics.NewCanvas(dev), graphics.NewRect(0, 0, w, h))
		px := dev.Pixels()
		return func(x, y int) uint32 { return px[(h-1-y)*w+x] }
	}
}

func TestStackPanelPaintsItsDivider(t *testing.T) {
	sp, th := stackPanelFixture(t)
	at := stackPanelPaint(t, sp)()

	// Items are 100 tall with a 6px splitter after the first, so the band
	// spans y=100..106 and its divider line sits on the middle row.
	line := uint32(th.SplitterInk)
	if got := at(100, 103); got != line {
		t.Errorf("divider pixel = %#08x, want the theme SplitterInk %#08x", got, line)
	}
	// One line, nothing else: no band above or below it.
	for _, y := range []int{101, 102, 104, 105} {
		if got := at(100, y); got != 0 {
			t.Errorf("y=%d painted %#08x, want only the divider row", y, got)
		}
	}
	if got := at(100, 50); got != 0 {
		t.Errorf("item area painted %#08x, want nothing", got)
	}
}

// The splitter lights up under the pointer and — the reported bug — goes back
// to its normal ink once the pointer is over an item again.
func TestStackPanelSplitterHoverFollowsThePointer(t *testing.T) {
	sp, th := stackPanelFixture(t)
	paint := stackPanelPaint(t, sp)

	sp.HoverAt(100, 103) // on the band
	if !sp.hovering || sp.hoverIdx != 0 {
		t.Fatalf("hovering=%v idx=%d, want the first splitter hot", sp.hovering, sp.hoverIdx)
	}
	if got := paint()(100, 103); got != uint32(th.SplitterHoverInk) {
		t.Errorf("hovered divider = %#08x, want SplitterHoverInk %#08x", got, uint32(th.SplitterHoverInk))
	}

	sp.HoverAt(100, 50) // an item, not the band
	if sp.hovering {
		t.Fatalf("still hovering idx=%d after the pointer left the band", sp.hoverIdx)
	}
	if got := paint()(100, 103); got != uint32(th.SplitterInk) {
		t.Errorf("divider after leaving = %#08x, want SplitterInk %#08x", got, uint32(th.SplitterInk))
	}

	// The band the pointer has to reach is the band that was painted: the
	// hit test and the painter share splitterRect's geometry.
	if idx := sp.hitSplitterLocal(100, 103); idx != 0 {
		t.Errorf("hitSplitterLocal on the divider = %d, want 0", idx)
	}
}

// The same always-on flag, on StackPanel: the theme can ask for it and the
// widget can override it either way.
func TestStackPanelHoverColorAlwaysOn(t *testing.T) {
	sp, th := stackPanelFixture(t)
	paint := stackPanelPaint(t, sp)
	if got := paint()(100, 103); got != uint32(th.SplitterInk) {
		t.Fatalf("divider = %#08x, want the normal ink %#08x", got, uint32(th.SplitterInk))
	}
	sp.SetHoverColorAlwaysOn(true)
	if got := paint()(100, 103); got != uint32(th.SplitterHoverInk) {
		t.Errorf("with SetHoverColorAlwaysOn(true) = %#08x, want the hover ink %#08x", got, uint32(th.SplitterHoverInk))
	}
	th.HoverColorAlwaysOn = true
	sp.SetHoverColorAlwaysOn(false)
	if got := paint()(100, 103); got != uint32(th.SplitterInk) {
		t.Errorf("with SetHoverColorAlwaysOn(false) over a theme on = %#08x, want the normal ink %#08x", got, uint32(th.SplitterInk))
	}
}
