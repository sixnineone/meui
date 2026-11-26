package widgets

import (
	"crypto/sha256"
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

// The arrow keys and "no modifiers held", named once so the tests read as intent.
const platformKeyLeft = platform.KeyLeft
const platformKeyRight = platform.KeyRight

func noMods() platform.Modifiers { return platform.Modifiers(0) }

// paintStrip renders a strip at a fixed size and returns its pixels. The soft device
// type is private to the graphics package, so the pixels are what crosses the boundary.
func paintStrip(t *testing.T, s *TabStrip, w, h int) []uint32 {
	t.Helper()
	s.SetFrameRect(graphics.NewRect(0, 0, float32(w), float32(h)))
	dev := graphics.NewSoftDevice(w, h)
	s.Paint(graphics.NewCanvas(dev), graphics.NewRect(0, 0, float32(w), float32(h)))
	return dev.Pixels()
}

func hashPixels(px []uint32) [32]byte {
	buf := make([]byte, 0, len(px)*4)
	for _, p := range px {
		buf = append(buf, byte(p>>24), byte(p>>16), byte(p>>8), byte(p))
	}
	return sha256.Sum256(buf)
}

// The whole point of the widget: a strip's bar and a TabPanel's bar must be the SAME
// drawing. This renders both at the same size with the same theme, same font and the
// same labels, and requires the pixels to match exactly — "looks similar" is what
// drifts.
func TestATabStripBarMatchesATabPanelBar(t *testing.T) {
	const w, h = 520, 40
	labels := []string{"Alpha", "Beta", "Gamma"}

	panel := NewTabPanel()
	panel.SetFrameRect(graphics.NewRect(0, 0, w, h))
	for _, l := range labels {
		panel.AddTabEx(l, l, "", meui.NewPanel(), false)
	}
	panel.SetActiveTab(1)
	// The panel's bar is at the top of its frame; the strip IS the bar.
	panelDev := graphics.NewSoftDevice(w, h)
	panel.Paint(graphics.NewCanvas(panelDev), graphics.NewRect(0, 0, w, h))

	strip := NewTabStrip()
	strip.SetFrameRect(graphics.NewRect(0, 0, w, h))
	items := make([]TabStripItem, 0, len(labels))
	for _, l := range labels {
		items = append(items, TabStripItem{ID: l, Label: l})
	}
	strip.SetItems(items)
	strip.SetSelected(1)
	stripDev := graphics.NewSoftDevice(w, h)
	strip.Paint(graphics.NewCanvas(stripDev), graphics.NewRect(0, 0, w, h))

	// Compare only the bar: a TabPanel paints nothing below it when its tabs have no
	// content, but it still owns that area, so compare the bar's rows exactly.
	barH := int(strip.TabTheme.TabHeight)
	a := panelDev.Pixels()
	b := stripDev.Pixels()
	for y := 0; y < barH; y++ {
		for x := 0; x < w; x++ {
			// Pixels are bottom-up.
			ia := (h-1-y)*w + x
			pa, pb := a[ia], b[ia]
			if pa != pb {
				t.Fatalf("the strip and the panel disagree at (%d,%d): panel %08x, strip %08x",
					x, y, pa, pb)
			}
		}
	}
}

// A strip is a radio group: exactly one item selected, and the callback fires with the
// new index — that is the client's whole interface to it.
func TestSelectingFiresTheCallback(t *testing.T) {
	s := NewTabStrip()
	s.SetFrameRect(graphics.NewRect(0, 0, 400, 40))
	s.SetItems([]TabStripItem{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}, {ID: "c", Label: "C"}})

	// The first item is selected when the items arrive: a strip with no selection is not
	// a radio group.
	if got := s.Selected(); got != 0 {
		t.Fatalf("the initial selection is %d, want 0", got)
	}
	var got []int
	s.OnSelectionChanged = func(i int) { got = append(got, i) }

	if !s.SetSelected(2) {
		t.Fatal("SetSelected(2) was refused")
	}
	if s.Selected() != 2 || s.SelectedID() != "c" || s.SelectedLabel() != "C" {
		t.Errorf("selection is %d/%q/%q, want 2/c/C", s.Selected(), s.SelectedID(), s.SelectedLabel())
	}
	// Re-selecting the same item is not a change, so it does not fire.
	s.SetSelected(2)
	if len(got) != 1 || got[0] != 2 {
		t.Errorf("the callback saw %v, want exactly [2]", got)
	}
	// Out of range is refused.
	if s.SetSelected(9) {
		t.Error("SetSelected(9) was accepted")
	}
	if s.SetSelected(-1) {
		t.Error("SetSelected(-1) was accepted")
	}
	// By ID.
	if !s.SelectID("a") || s.Selected() != 0 {
		t.Error("SelectID(a) did not select the first item")
	}
	if s.SelectID("nope") {
		t.Error("SelectID found an item that does not exist")
	}
}

// Clicking an item selects it, at the position the painter draws it — the hit test and
// the painter share one source of an item's geometry.
func TestClickingAnItemSelectsIt(t *testing.T) {
	s := NewTabStrip()
	s.SetFrameRect(graphics.NewRect(0, 0, 400, 40))
	s.SetItems([]TabStripItem{{Label: "A"}, {Label: "B"}, {Label: "C"}})
	s.SetSelected(0)

	r, ok := s.ItemRect(2)
	if !ok {
		t.Fatal("ItemRect(2) not found")
	}
	if !s.OnMouseDown(r.X+r.Width/2, r.Y+r.Height/2, 0) {
		t.Fatal("a press on the third item was not consumed")
	}
	if got := s.Selected(); got != 2 {
		t.Errorf("the selection is %d after clicking item 2, want 2", got)
	}
	// The item under a point is the one the painter drew there.
	if got := s.ItemAt(r.X+1, r.Y+1); got != 2 {
		t.Errorf("ItemAt says %d over item 2's rect, want 2", got)
	}
	// A press on the bar but past the last item is consumed and selects nothing new.
	before := s.Selected()
	if !s.OnMouseDown(399, 5, 0) {
		t.Error("a press on the empty part of the bar was not consumed")
	}
	if s.Selected() != before {
		t.Error("a press on empty bar moved the selection")
	}
	// A right-click is not the strip's.
	if s.OnMouseDown(r.X+1, r.Y+1, mouseButtonRight) {
		t.Error("a right-click was consumed")
	}
}

// A disabled item is drawn dimmed, cannot be selected by click or by keyboard, and is
// skipped when the arrows walk the strip. Disabling the SELECTED item moves the
// selection off it, because a radio group sitting on a dead option is unusable.
func TestDisabledItemsAreSkipped(t *testing.T) {
	s := NewTabStrip()
	s.SetFrameRect(graphics.NewRect(0, 0, 400, 40))
	s.SetItems([]TabStripItem{
		{ID: "a", Label: "A"},
		{ID: "b", Label: "B", Disabled: true},
		{ID: "c", Label: "C"},
	})
	// The middle item is disabled, so the initial selection skips it.
	if got := s.Selected(); got != 0 {
		t.Fatalf("the initial selection is %d, want 0 (item 1 is disabled)", got)
	}
	if s.SetSelected(1) {
		t.Error("a disabled item accepted the selection")
	}
	if s.Selected() != 0 {
		t.Error("the refused selection moved anyway")
	}
	// A click on it is consumed but does not select.
	r, _ := s.ItemRect(1)
	s.OnMouseDown(r.X+2, r.Y+2, 0)
	if s.Selected() != 0 {
		t.Errorf("clicking the disabled item selected %d", s.Selected())
	}
	// The keyboard walks past it, wrapping.
	if !s.OnKeyDown(platformKeyRight, noMods()) {
		t.Fatal("the right arrow was not consumed")
	}
	if got := s.Selected(); got != 2 {
		t.Errorf("the right arrow moved to %d, want 2 (skipping the disabled 1)", got)
	}
	if !s.OnKeyDown(platformKeyRight, noMods()) {
		t.Fatal("the right arrow was not consumed at the end")
	}
	if got := s.Selected(); got != 0 {
		t.Errorf("the right arrow at the end moved to %d, want 0 (wrapping)", got)
	}
	if !s.OnKeyDown(platformKeyLeft, noMods()) {
		t.Fatal("the left arrow was not consumed")
	}
	if got := s.Selected(); got != 2 {
		t.Errorf("the left arrow moved to %d, want 2 (wrapping back, skipping 1)", got)
	}

	// Disabling the selected item moves the selection off it and says so.
	s.SetSelected(2)
	fired := -1
	s.OnSelectionChanged = func(i int) { fired = i }
	s.SetItemDisabled(2, true)
	if s.Selected() == 2 {
		t.Error("the selection stayed on the item that was just disabled")
	}
	if fired != s.Selected() {
		t.Errorf("the callback saw %d but the selection is %d", fired, s.Selected())
	}
	// Re-enabling it does not steal the selection back.
	s.SetItemDisabled(2, false)
	if s.Selected() == 2 {
		t.Error("re-enabling an item selected it")
	}
}

// A strip with every item disabled has no selection at all, which is honest: there is
// nothing to select.
func TestAStripWithNoSelectableItemHasNoSelection(t *testing.T) {
	s := NewTabStrip()
	s.SetFrameRect(graphics.NewRect(0, 0, 400, 40))
	s.SetItems([]TabStripItem{{Label: "A", Disabled: true}, {Label: "B", Disabled: true}})
	if got := s.Selected(); got != -1 {
		t.Errorf("the selection is %d with every item disabled, want -1", got)
	}
	if s.SelectedLabel() != "" || s.SelectedID() != "" {
		t.Error("a strip with no selection reported a label or an id")
	}
	// The keyboard consumes the arrow but cannot move.
	if !s.OnKeyDown(platformKeyRight, noMods()) {
		t.Error("the arrow key was not consumed")
	}
	if s.Selected() != -1 {
		t.Errorf("the selection moved to %d with nothing selectable", s.Selected())
	}
	// And it paints without panicking.
	paintStrip(t, s, 400, 40)
}

// Disabling the whole strip dims it and refuses presses, like every other widget.
func TestDisablingTheStrip(t *testing.T) {
	s := NewTabStrip()
	s.SetFrameRect(graphics.NewRect(0, 0, 400, 40))
	s.SetItems([]TabStripItem{{Label: "A"}, {Label: "B"}})
	r, _ := s.ItemRect(1)

	s.SetEnabled(false)
	if s.OnMouseDown(r.X+2, r.Y+2, 0) {
		t.Error("a disabled strip consumed a press")
	}
	if s.Selected() != 0 {
		t.Error("a disabled strip moved its selection")
	}
	if s.OnKeyDown(platformKeyRight, noMods()) {
		t.Error("a disabled strip consumed a key")
	}
	enabled := paintStrip(t, s, 400, 40)
	s.SetEnabled(true)
	disabled := paintStrip(t, s, 400, 40)
	// Re-enabling changes the drawing (the dim is gone), which is what proves the
	// disabled state reaches the painter rather than only the event handlers.
	same := true
	for i := range enabled {
		if enabled[i] != disabled[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("a disabled strip paints identically to an enabled one")
	}
}

// The hovered item is the one the pointer is over, and leaving clears it.
func TestHoverFollowsThePointer(t *testing.T) {
	s := NewTabStrip()
	s.SetFrameRect(graphics.NewRect(0, 0, 400, 40))
	s.SetItems([]TabStripItem{{Label: "A"}, {Label: "B"}})
	r, _ := s.ItemRect(1)
	s.OnMouseMove(r.X+2, r.Y+2)
	if s.hovered != 1 {
		t.Errorf("the hovered item is %d, want 1", s.hovered)
	}
	// Moving off the bar clears it.
	s.OnMouseMove(r.X+2, 39)
	if s.hovered != -1 {
		t.Errorf("the hovered item is %d with the pointer below the bar, want -1", s.hovered)
	}
	s.OnMouseMove(r.X+2, r.Y+2)
	s.OnMouseLeave()
	if s.hovered != -1 {
		t.Errorf("the hovered item is %d after the pointer left, want -1", s.hovered)
	}
}

// Icons take room in the item's width and are drawn, so an icon-only item is not a
// zero-width sliver.
func TestIconsWidenAnItem(t *testing.T) {
	plain := NewTabStrip()
	plain.SetFrameRect(graphics.NewRect(0, 0, 400, 40))
	plain.SetItems([]TabStripItem{{Label: "A"}})

	iconed := NewTabStrip()
	iconed.SetFrameRect(graphics.NewRect(0, 0, 400, 40))
	iconed.SetItems([]TabStripItem{{Label: "A", Icon: "\u2699"}}) // a gear glyph

	rp, _ := plain.ItemRect(0)
	ri, _ := iconed.ItemRect(0)
	if ri.Width <= rp.Width {
		t.Errorf("an item with an icon is %v wide, the plain one %v: the icon took no room",
			ri.Width, rp.Width)
	}
	// An icon-only item still has a usable width.
	only := NewTabStrip()
	only.SetFrameRect(graphics.NewRect(0, 0, 400, 40))
	only.SetItems([]TabStripItem{{Icon: "\u2699"}})
	ro, _ := only.ItemRect(0)
	if ro.Width < 12 {
		t.Errorf("an icon-only item is %v wide, too narrow to hit", ro.Width)
	}
	// Both paint without panicking.
	paintStrip(t, iconed, 400, 40)
	paintStrip(t, only, 400, 40)
}

// The strip reports the size its items need, so an app that sizes it itself gets a strip
// wide enough for every item. (The layout package does not consult this — it uses
// explicit sizes — so the demo sets the height and lets the layout set the width.)
func TestPreferredSizeCoversTheItems(t *testing.T) {
	s := NewTabStrip()
	s.SetItems([]TabStripItem{{Label: "One"}, {Label: "Two"}, {Label: "Three"}})
	w, h := s.PreferredSize()
	if h != s.TabTheme.TabHeight {
		t.Errorf("the preferred height is %v, want the bar height %v", h, s.TabTheme.TabHeight)
	}
	// The width must at least cover the last item's right edge.
	s.SetFrameRect(graphics.NewRect(0, 0, w, h))
	last, _ := s.ItemRect(2)
	if last.X+last.Width > w {
		t.Errorf("the preferred width %v does not cover the last item (ends at %v)",
			w, last.X+last.Width)
	}
}

// An empty strip paints nothing and consumes nothing, rather than drawing a bare bar.
func TestAnEmptyStripIsHarmless(t *testing.T) {
	s := NewTabStrip()
	s.SetFrameRect(graphics.NewRect(0, 0, 400, 40))
	dev := paintStrip(t, s, 400, 40)
	for _, px := range dev {
		if px&0x00FFFFFF != 0 {
			t.Fatal("an empty strip painted something")
		}
	}
	if s.OnMouseDown(10, 10, 0) {
		t.Error("an empty strip consumed a press")
	}
	if s.OnKeyDown(platformKeyRight, noMods()) {
		t.Error("an empty strip consumed a key")
	}
	if s.Selected() != -1 {
		t.Errorf("an empty strip reports selection %d", s.Selected())
	}
}

// Too many items for the width makes the strip scrollable rather than silently dropping
// the ones that do not fit, and the arrows appear only then.
func TestOverflowingItemsScroll(t *testing.T) {
	s := NewTabStrip()
	s.SetFrameRect(graphics.NewRect(0, 0, 200, 40))
	var items []TabStripItem
	for _, l := range []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon", "Zeta"} {
		items = append(items, TabStripItem{Label: l})
	}
	s.SetItems(items)
	s.SetSelected(0)
	paintStrip(t, s, 200, 40)
	if !s.showScrollR {
		t.Error("the strip does not report overflow with six items in 200px")
	}
	// Scrolling right reveals the end.
	for i := 0; i < 6; i++ {
		s.OnMouseDown(198, 5, 0)
	}
	paintStrip(t, s, 200, 40)
	if !s.showScrollL {
		t.Error("the strip does not report a scroll offset after scrolling right")
	}
	if s.scrollOffset <= 0 {
		t.Error("the scroll offset did not move")
	}
	// A wide strip fits the same items and shows no arrows.
	wide := NewTabStrip()
	wide.SetFrameRect(graphics.NewRect(0, 0, 1200, 40))
	wide.SetItems(items)
	paintStrip(t, wide, 1200, 40)
	if wide.showScrollL || wide.showScrollR {
		t.Error("a strip wide enough for its items still reports overflow")
	}
}

// Two strips that are configured identically paint identically, which is what makes the
// pixel comparison against TabPanel meaningful rather than a coincidence.
func TestTwoIdenticalStripsPaintIdentically(t *testing.T) {
	build := func() *TabStrip {
		s := NewTabStrip()
		s.SetItems([]TabStripItem{{Label: "Alpha"}, {Label: "Beta"}})
		s.SetSelected(1)
		return s
	}
	a := hashPixels(paintStrip(t, build(), 300, 40))
	b := hashPixels(paintStrip(t, build(), 300, 40))
	if a != b {
		t.Errorf("two identical strips painted differently: %x vs %x", a, b)
	}
}

// ItemRectForDebug exposes an item's rect to a debug comparison.
func (s *TabStrip) ItemRectForDebug(i int) graphics.Rect {
	r, _ := s.ItemRect(i)
	return r
}

// An ACTIVE tab's label must be drawn in the active ink. It was not: TabPanel pre-built
// each label with the inactive ink and reused it, because the reuse gate compared the
// cache's theme version (0) against a theme version that is 0 until an app calls
// SetDefaultTheme. The label therefore kept whatever colour the tab had when it was
// built, and only came out right after a theme was set — which made the cache rebuild on
// every paint, so it was either wrong or useless, never useful.
//
// The inks are antialiased, so no pixel is exactly the ink. The measurement is the SAME
// tab in two states — selected and not — so the glyphs, the rect and the coverage are
// identical and only the ink can differ. No theme is ever set here, so the old gate
// stays at 0 and the bug would show.
func TestAnActiveTabLabelUsesTheActiveInk(t *testing.T) {
	const w, h = 300, 40

	// A PRIVATE theme with unmistakable inks. The ambient default is shared and other
	// tests change it (a light theme's "active" ink is darker than its inactive one), so
	// a test that assumed a direction would pass alone and fail in the suite.
	th := *theme.DefaultTabTheme
	th.TextInk = graphics.Color(0xFF404040)
	th.TextActive = graphics.Color(0xFFFFFFFF)
	th.BackgroundInk = graphics.Color(0xFF000000)
	th.BackgroundActive = graphics.Color(0xFF000000)
	th.BackgroundHover = graphics.Color(0xFF000000)
	th.IndicatorInk = graphics.Color(0xFF000000)

	// inkOver sums the RGB under a rect: a proxy for how bright the label is.
	inkOver := func(px []uint32, r graphics.Rect) int {
		sum := 0
		for y := int(r.Y) + 2; y < int(r.Y+r.Height)-2; y++ {
			for x := int(r.X) + 4; x < int(r.X+r.Width)-4; x++ {
				if x < 0 || y < 0 || x >= w || y >= h {
					continue
				}
				v := px[(h-1-y)*w+x]
				sum += int((v>>16)&0xFF) + int((v>>8)&0xFF) + int(v&0xFF)
			}
		}
		return sum
	}

	// The first tab, selected and not, in a strip.
	stripInk := func(selected int) int {
		s := NewTabStrip()
		s.TabTheme = &th
		s.SetItems([]TabStripItem{{Label: "Alpha"}, {Label: "Beta"}})
		s.SetSelected(selected)
		r, _ := s.ItemRect(0)
		return inkOver(paintStrip(t, s, w, h), r)
	}
	stripActive, stripInactive := stripInk(0), stripInk(1)

	// The same tab, active and not, in a panel.
	panelInk := func(active int) int {
		p := NewTabPanel()
		p.TabTheme = &th
		p.SetFrameRect(graphics.NewRect(0, 0, w, h))
		p.AddTab("Alpha", meui.NewPanel())
		p.AddTab("Beta", meui.NewPanel())
		p.SetActiveTab(active)
		dev := graphics.NewSoftDevice(w, h)
		p.Paint(graphics.NewCanvas(dev), graphics.NewRect(0, 0, w, h))
		// The panel paints tab 0 at the same place the strip does.
		s := NewTabStrip()
		s.TabTheme = &th
		s.SetItems([]TabStripItem{{Label: "Alpha"}, {Label: "Beta"}})
		r, _ := s.ItemRect(0)
		return inkOver(dev.Pixels(), r)
	}
	panelActive, panelInactive := panelInk(0), panelInk(1)

	if stripActive <= stripInactive {
		t.Errorf("the strip's active label is not brighter than the same label inactive "+
			"(%d vs %d): it is not using the active ink", stripActive, stripInactive)
	}
	if panelActive <= panelInactive {
		t.Errorf("the panel's active label is not brighter than the same label inactive "+
			"(%d vs %d): it kept the inactive colour it was built with", panelActive, panelInactive)
	}
	// And the two widgets agree, which is the point of sharing the painter.
	if panelActive != stripActive || panelInactive != stripInactive {
		t.Errorf("the panel and the strip draw different labels: active %d/%d, inactive %d/%d",
			panelActive, stripActive, panelInactive, stripInactive)
	}
}

// A strip sized to its own PreferredSize must NOT open already overflowing: the whole
// point of the method is that a caller who uses it gets a strip that fits. This is the
// property the charts demo relies on.
func TestPreferredSizeLeavesRoomForEveryItem(t *testing.T) {
	for _, labels := range [][]string{
		{"Log", "Linear"},
		{"One"},
		{"Alpha", "Beta", "Gamma", "Delta"},
		{"", "\u2699", "Mixed label"},
	} {
		s := NewTabStrip()
		items := make([]TabStripItem, 0, len(labels))
		for i, l := range labels {
			it := TabStripItem{Label: l}
			if i == 1 {
				it.Icon = "\u2699"
			}
			items = append(items, it)
		}
		s.SetItems(items)
		w, h := s.PreferredSize()
		s.SetFrameRect(graphics.NewRect(0, 0, w, h))
		paintStrip(t, s, int(w), int(h))
		if s.showScrollR || s.showScrollL {
			last, _ := s.ItemRect(len(items) - 1)
			t.Errorf("labels %v: PreferredSize %v still overflows (last item ends at %v)",
				labels, w, last.X+last.Width)
		}
	}
}
