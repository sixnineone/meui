package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

// The splitter is a divider line from the theme, not a band of lightened panel
// background. An imported theme used to lighten the panel background into
// StackPanelTheme.BackgroundInk, which painted a bright strip along every
// split — the owner's screenshot marked three of them.
func TestSplitViewPaintsTheThemeDivider(t *testing.T) {
	saved := theme.DefaultStackPanelTheme
	defer func() { theme.DefaultStackPanelTheme = saved }()

	const (
		divider = graphics.Color(0xFF3366AA)
		hover   = graphics.Color(0xFFCC5533)
	)
	theme.DefaultStackPanelTheme = &theme.StackPanelTheme{
		BackgroundInk:    0x00000000, // transparent: no band
		SplitterInk:      divider,
		SplitterHoverInk: hover,
		GripInk:          0, // no grip marks
		GripSize:         4,
		SplitterSize:     4,
	}

	const w, h = 400, 200
	sv := NewSplitView()
	sv.SetPanels(meui.NewPanel(), meui.NewPanel())
	sv.SetFrameRect(graphics.NewRect(0, 0, w, h))
	sv.SplitPosition = 0.5
	sv.ValidateLayout()

	paint := func() []uint32 {
		dev := graphics.NewSoftDevice(w, h)
		sv.Paint(graphics.NewCanvas(dev), graphics.NewRect(0, 0, w, h))
		return dev.Pixels()
	}

	px := paint()
	if got := px[100*w+200]; got != uint32(divider) {
		t.Errorf("divider pixel = %#08x, want the theme's SplitterInk %#08x", got, uint32(divider))
	}
	// The band either side must not be filled with anything (the old bug).
	for _, x := range []int{198, 199, 201, 202} {
		if got := px[100*w+x]; got == uint32(divider) {
			t.Errorf("pixel x=%d is part of a band, not a 1px divider", x)
		}
	}

	sv.hovering = true
	px = paint()
	if got := px[100*w+200]; got != uint32(hover) {
		t.Errorf("hovered divider = %#08x, want SplitterHoverInk %#08x", got, uint32(hover))
	}

	// A theme that wants a band still gets one.
	theme.DefaultStackPanelTheme.BackgroundInk = graphics.Color(0x40FF00FF)
	sv.hovering = false
	px = paint()
	if got := px[100*w+198]; uint32(got)>>24 == 0 {
		t.Errorf("band pixel is transparent with an opaque BackgroundInk: %#08x", got)
	}
}

// The default look is one line and nothing else: no band, no grip marks.
func TestSplitViewIsOnlyOneLineByDefault(t *testing.T) {
	saved := theme.DefaultStackPanelTheme
	defer func() { theme.DefaultStackPanelTheme = saved }()
	theme.DefaultStackPanelTheme = theme.DefaultStackPanelTheme.Copy()

	const w, h = 400, 200
	sv := NewSplitView()
	sv.SetPanels(meui.NewPanel(), meui.NewPanel())
	sv.SetFrameRect(graphics.NewRect(0, 0, w, h))
	sv.SplitPosition = 0.5
	sv.ValidateLayout()

	dev := graphics.NewSoftDevice(w, h)
	sv.Paint(graphics.NewCanvas(dev), graphics.NewRect(0, 0, w, h))
	px := dev.Pixels()
	// Pixels() is bottom-up, so the divider's column is the same at any row.
	divider := uint32(theme.DefaultStackPanelTheme.SplitterInk)
	painted := map[int]int{}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if px[y*w+x] != 0 {
				painted[x]++
			}
		}
	}
	if len(painted) != 1 {
		t.Errorf("painted columns = %v, want only the divider column", painted)
	}
	if painted[200] != h {
		t.Errorf("divider column covers %d rows, want the full height %d", painted[200], h)
	}
	if got := px[100*w+200]; got != divider {
		t.Errorf("divider pixel = %#08x, want SplitterInk %#08x", got, divider)
	}
}

// A theme that opts into grips gets short marks centred on the handle, never
// full-length lines beside the divider.
func TestSplitViewGripsAreOptInAndShort(t *testing.T) {
	saved := theme.DefaultStackPanelTheme
	defer func() { theme.DefaultStackPanelTheme = saved }()
	theme.DefaultStackPanelTheme = theme.DefaultStackPanelTheme.Copy()
	theme.DefaultStackPanelTheme.GripInk = graphics.Color(0xFF88FF88)

	const w, h = 400, 200
	sv := NewSplitView()
	sv.SetPanels(meui.NewPanel(), meui.NewPanel())
	sv.SetFrameRect(graphics.NewRect(0, 0, w, h))
	sv.SplitPosition = 0.5
	sv.ValidateLayout()

	dev := graphics.NewSoftDevice(w, h)
	sv.Paint(graphics.NewCanvas(dev), graphics.NewRect(0, 0, w, h))
	px := dev.Pixels()
	divider := uint32(theme.DefaultStackPanelTheme.SplitterInk)
	rows := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Antialiased line ends, so anything that is not the divider
			// (or empty) is grip ink.
			if p := px[y*w+x]; p != 0 && p != divider {
				rows++
				break
			}
		}
	}
	// Three dashes of GripSize*2 pixels at most, not the full 200 rows.
	if rows == 0 {
		t.Fatal("no grip marks painted with an opaque GripInk")
	}
	if rows > 12 {
		t.Errorf("grips span %d rows, want short centred marks", rows)
	}
}

// The hover ink follows the pointer AND clears when it leaves the handle.
// The window reports every pointer move along the hit chain to HoverAt;
// before that the splitter only saw the pointer arrive, so the divider stuck
// in its hover ink (the owner's orange line that never switched off).
func TestSplitViewHoverInkClearsWhenThePointerLeaves(t *testing.T) {
	saved := theme.DefaultStackPanelTheme
	defer func() { theme.DefaultStackPanelTheme = saved }()
	theme.DefaultStackPanelTheme = theme.DefaultStackPanelTheme.Copy()
	const (
		divider = graphics.Color(0xFF3366AA)
		hover   = graphics.Color(0xFFCC5533)
	)
	theme.DefaultStackPanelTheme.SplitterInk = divider
	theme.DefaultStackPanelTheme.SplitterHoverInk = hover
	theme.DefaultStackPanelTheme.GripInk = 0

	const w, h = 400, 200
	sv := NewSplitView()
	sv.SetPanels(meui.NewPanel(), meui.NewPanel())
	sv.SetFrameRect(graphics.NewRect(0, 0, w, h))
	sv.SplitPosition = 0.5
	sv.ValidateLayout()

	paint := func() func(x, y int) uint32 {
		dev := graphics.NewSoftDevice(w, h)
		sv.Paint(graphics.NewCanvas(dev), graphics.NewRect(0, 0, w, h))
		px := dev.Pixels()
		return func(x, y int) uint32 { return px[(h-1-y)*w+x] }
	}

	sv.HoverAt(200, 100) // the handle
	if !sv.hovering {
		t.Fatal("HoverAt on the handle did not set hovering")
	}
	if got := paint()(200, 100); got != uint32(hover) {
		t.Errorf("hovered divider = %#08x, want the hover ink %#08x", got, uint32(hover))
	}

	sv.HoverAt(40, 100) // a panel, not the handle
	if sv.hovering {
		t.Fatal("HoverAt off the handle did not clear hovering")
	}
	if got := paint()(200, 100); got != uint32(divider) {
		t.Errorf("divider after leaving = %#08x, want the normal ink %#08x", got, uint32(divider))
	}
}

// HoverColorAlwaysOn draws every divider in the hover ink, so the highlight
// does not depend on where the pointer is (the owner liked the accent line
// and asked for it to be deliberate rather than a stuck hover). The
// per-widget setter overrides the theme either way.
func TestSplitViewHoverColorAlwaysOn(t *testing.T) {
	saved := theme.DefaultStackPanelTheme
	defer func() { theme.DefaultStackPanelTheme = saved }()
	theme.DefaultStackPanelTheme = theme.DefaultStackPanelTheme.Copy()
	const (
		divider = graphics.Color(0xFF3366AA)
		hover   = graphics.Color(0xFFCC5533)
	)
	theme.DefaultStackPanelTheme.SplitterInk = divider
	theme.DefaultStackPanelTheme.SplitterHoverInk = hover
	theme.DefaultStackPanelTheme.GripInk = 0

	const w, h = 400, 200
	sv := NewSplitView()
	sv.SetPanels(meui.NewPanel(), meui.NewPanel())
	sv.SetFrameRect(graphics.NewRect(0, 0, w, h))
	sv.SplitPosition = 0.5
	sv.ValidateLayout()

	paint := func() func(x, y int) uint32 {
		dev := graphics.NewSoftDevice(w, h)
		sv.Paint(graphics.NewCanvas(dev), graphics.NewRect(0, 0, w, h))
		px := dev.Pixels()
		return func(x, y int) uint32 { return px[(h-1-y)*w+x] }
	}

	// Theme off, pointer away: the plain divider.
	if got := paint()(200, 100); got != uint32(divider) {
		t.Fatalf("divider = %#08x, want the normal ink %#08x", got, uint32(divider))
	}
	// The widget pins the highlight on.
	sv.SetHoverColorAlwaysOn(true)
	if got := paint()(200, 100); got != uint32(hover) {
		t.Errorf("with SetHoverColorAlwaysOn(true) = %#08x, want the hover ink %#08x", got, uint32(hover))
	}
	// The widget can also pin it off against a theme that wants it on.
	theme.DefaultStackPanelTheme.HoverColorAlwaysOn = true
	sv.SetHoverColorAlwaysOn(false)
	if got := paint()(200, 100); got != uint32(divider) {
		t.Errorf("with SetHoverColorAlwaysOn(false) over a theme on = %#08x, want the normal ink %#08x", got, uint32(divider))
	}
	// And the theme alone is enough when the widget says nothing.
	sv = NewSplitView()
	sv.SetPanels(meui.NewPanel(), meui.NewPanel())
	sv.SetFrameRect(graphics.NewRect(0, 0, w, h))
	sv.SplitPosition = 0.5
	sv.ValidateLayout()
	if got := paint()(200, 100); got != uint32(hover) {
		t.Errorf("theme HoverColorAlwaysOn = %#08x, want the hover ink %#08x", got, uint32(hover))
	}
}
