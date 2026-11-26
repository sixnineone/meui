package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/theme"
)

// NavigationRail is the SIDE half of the adaptive pair the NavigationBar is the bottom
// half of: the same destinations, laid out vertically down the left edge for a surface
// with room for them.
//
// Android's own guidance is that the two are one control with two presentations chosen by
// window size class — a bottom bar on a compact surface, a rail on a medium one — which is
// exactly what platform.FormFactor reports. An app holds one list of destinations and asks
// the form factor which of these to build; it does not choose by platform, because a
// tablet and a desktop window of the same width want the same thing.
//
// It shares NavigationBarItem and the selection contract with NavigationBar deliberately:
// two controls that mean the same thing should not have two vocabularies.
type NavigationRail struct {
	meui.BaseWidget
	*theme.TabTheme

	items    []NavigationBarItem
	selected int
	hovered  int
	pressed  int

	// OnSelectionChanged fires when the selection moves.
	OnSelectionChanged func(index int)
}

// NewNavigationRail builds an empty rail selected on the first destination.
func NewNavigationRail() *NavigationRail {
	return &NavigationRail{
		BaseWidget: *meui.NewBaseWidget(),
		TabTheme:   theme.DefaultTabTheme.Copy(),
		selected:   0,
		hovered:    -1,
		pressed:    -1,
	}
}

// ── items ───────────────────────────────────────────────────────────

func (r *NavigationRail) SetItems(items []NavigationBarItem) *NavigationRail {
	r.items = items
	if r.selected >= len(items) {
		r.selected = 0
	}
	r.MarkForRedraw()
	return r
}

func (r *NavigationRail) Items() []NavigationBarItem { return r.items }
func (r *NavigationRail) Count() int                 { return len(r.items) }
func (r *NavigationRail) Selected() int              { return r.selected }

func (r *NavigationRail) SelectedID() string {
	if r.selected >= 0 && r.selected < len(r.items) {
		return r.items[r.selected].ID
	}
	return ""
}

// SetSelected moves the selection and reports it, so a programmatic move behaves like a
// tap. It refuses a disabled destination.
func (r *NavigationRail) SetSelected(index int) bool {
	if index < 0 || index >= len(r.items) || r.items[index].Disabled {
		return false
	}
	if index == r.selected {
		return true
	}
	r.selected = index
	r.MarkForRedraw()
	if r.OnSelectionChanged != nil {
		r.OnSelectionChanged(index)
	}
	return true
}

func (r *NavigationRail) SelectID(id string) bool {
	for i := range r.items {
		if r.items[i].ID == id {
			return r.SetSelected(i)
		}
	}
	return false
}

// SetBoundProperty implements the QML binding hook.
func (r *NavigationRail) SetBoundProperty(name string, value interface{}) {
	if name == "selected" {
		r.SetSelected(int(toFloat32(value)))
	}
}

// ── geometry ────────────────────────────────────────────────────────

// RailWidth is the rail's own width. A rail is a fixed-width column, so unlike the bottom
// bar it does not divide the surface — the destinations stack down it instead.
func (r *NavigationRail) RailWidth() float32 {
	if w := theme.DefaultTheme.Metrics.NavRailWidth; w > 0 {
		return w
	}
	return 80
}

// ItemHeight is one destination's height: enough for the icon, the indicator pill around
// it and the label beneath it, and at least the theme's tab height and the touch floor.
//
// It used to be max(TabTheme.TabHeight, MinTouchTarget), which is only right when the
// theme's tab height happens to be big enough for that construction. On a DESKTOP theme
// TabHeight is 28 and MinTouchTarget is 0, so each destination was 28px while its own
// content needed ~54 — and every label was drawn on top of the NEXT destination's icon.
// The Android theme (TabHeight 64) hid it, which is why it only appeared at a
// tablet-sized window on a desktop theme. The rail's tests missed it because they assert
// geometry and hit-testing, never that an item is tall enough to draw what it draws.
func (r *NavigationRail) ItemHeight() float32 {
	h := r.contentHeight()
	if t := r.TabTheme.TabHeight; t > h {
		h = t
	}
	if floor := theme.DefaultTheme.Metrics.MinTouchTarget; h < floor {
		h = floor
	}
	return h
}

// contentHeight is what one destination actually draws: the indicator pill (iconSize+12
// tall), the label's line beneath it, and a little slack. The LARGEST icon among the
// items is used, because one oversized icon must not overflow the destination carrying
// it — the slots are all the same height.
func (r *NavigationRail) contentHeight() float32 {
	icon := float32(24)
	for _, it := range r.items {
		if it.IconSize > icon {
			icon = it.IconSize
		}
	}
	return icon + 12 + r.font().LineHeight() + 4
}

// ItemRect is a destination's rect in the rail's own coordinates.
func (r *NavigationRail) ItemRect(i int) (graphics.Rect, bool) {
	if i < 0 || i >= len(r.items) {
		return graphics.Rect{}, false
	}
	h := r.ItemHeight()
	// The rail starts its destinations a little below the top, where a rail's own leading
	// control (a menu or FAB) would sit.
	top := r.topInset()
	return graphics.NewRect(0, top+float32(i)*h, r.RailWidth(), h), true
}

func (r *NavigationRail) topInset() float32 { return 8 }

// ItemAt reports which destination holds a point, or -1.
func (r *NavigationRail) ItemAt(x, y float32) int {
	if len(r.items) == 0 || x < 0 || x > r.RailWidth() {
		return -1
	}
	h := r.ItemHeight()
	if h <= 0 {
		return -1
	}
	i := int((y - r.topInset()) / h)
	if i < 0 || i >= len(r.items) {
		return -1
	}
	return i
}

func (r *NavigationRail) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	return r.RailWidth(), h
}

// ── interaction ─────────────────────────────────────────────────────

func (r *NavigationRail) OnMouseDown(x, y float32, button int) bool {
	if !r.Enabled() || button != mouseButtonLeft {
		return false
	}
	i := r.ItemAt(x, y)
	if i < 0 || r.items[i].Disabled {
		return false
	}
	r.pressed = i
	r.MarkForRedraw()
	return true
}

func (r *NavigationRail) OnMouseUp(x, y float32, button int) bool {
	if button != mouseButtonLeft {
		return false
	}
	was := r.pressed
	r.pressed = -1
	if was < 0 {
		return false
	}
	// Select on RELEASE, and only when the pointer is still on the same destination —
	// the same contract as the bottom bar, for the same reason.
	if i := r.ItemAt(x, y); i == was {
		r.SetSelected(i)
	}
	r.MarkForRedraw()
	return true
}

func (r *NavigationRail) OnMouseMove(x, y float32) {
	if i := r.ItemAt(x, y); i != r.hovered {
		r.hovered = i
		r.MarkForRedraw()
	}
}

// HoverAt implements HoverTarget.
func (r *NavigationRail) HoverAt(x, y float32) { r.OnMouseMove(x, y) }

// ── paint ───────────────────────────────────────────────────────────

func (r *NavigationRail) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !r.Visible() || len(r.items) == 0 || rect.Width < 1 || rect.Height < 1 {
		return
	}
	canvas.Save()
	canvas.Translate(rect.X, rect.Y)

	w := r.RailWidth()
	if w > rect.Width {
		w = rect.Width
	}
	bg := graphics.NewPaint()
	bg.SetColor(r.BackgroundInk)
	canvas.DrawRect(graphics.NewRect(0, 0, w, rect.Height), bg)
	if r.EdgeInk != 0 {
		// The rail's separation is a hairline on its RIGHT edge: it is a column, so the
		// edge that matters is the one against the content.
		edge := graphics.NewPaint()
		edge.SetColor(r.EdgeInk)
		canvas.DrawRect(graphics.NewRect(w-1, 0, 1, rect.Height), edge)
	}

	font := r.font()
	for i, item := range r.items {
		ir, ok := r.ItemRect(i)
		if !ok {
			continue
		}
		ir.Width = w
		r.paintItem(canvas, i, item, font, ir)
	}

	canvas.Restore()
}

func (r *NavigationRail) font() graphics.Font {
	if r.TabTheme.Font != nil {
		return r.TabTheme.Font
	}
	if f := theme.DefaultTheme.BaseFont; f != nil {
		return f
	}
	return graphics.SystemFont
}

func (r *NavigationRail) paintItem(canvas *graphics.Canvas, i int, item NavigationBarItem, font graphics.Font, ir graphics.Rect) {
	active := i == r.selected
	disabled := item.Disabled

	ink := r.TextInk
	switch {
	case disabled:
		ink = theme.DefaultTheme.DisabledText
	case active:
		ink = r.TextActive
	case i == r.hovered:
		ink = r.TextHover
	}

	iconSize := item.IconSize
	if iconSize <= 0 {
		iconSize = 24
	}
	// The active indicator is a pill WIDER than the icon, centred on the rail — the same
	// construction as the bottom bar, so the two read as one control.
	indW := iconSize + 32
	if indW > ir.Width-8 {
		indW = ir.Width - 8
	}
	indH := iconSize + 12
	cx := ir.X + ir.Width/2
	indY := ir.Y + (ir.Height-indH-font.Baseline()-6)/2
	if active && r.BackgroundActive != 0 {
		p := graphics.NewPaint()
		p.SetColor(r.BackgroundActive)
		canvas.DrawRoundedRect(graphics.NewRect(cx-indW/2, indY, indW, indH), indH/2, indH/2, p)
	}

	iconY := indY + (indH-iconSize)/2
	if item.Icon != "" {
		if doc := loadIconDoc(item.Icon); doc != nil {
			meui.DrawSVG(canvas, doc, cx-iconSize/2, iconY, iconSize, iconSize)
		} else {
			p := graphics.NewPaint()
			p.SetColor(ink)
			t := graphics.NewText(item.Icon, &graphics.TextDecoration{Font: font, OnBackgroundInk: ink})
			t.Draw(canvas, cx-t.Width()/2, iconY+iconSize/2+font.Baseline()/2)
		}
	}

	if item.Label != "" {
		p := graphics.NewPaint()
		p.SetColor(ink)
		t := graphics.NewText(item.Label, &graphics.TextDecoration{Font: font, OnBackgroundInk: ink})
		labelY := indY + indH + font.Baseline()
		canvas.Save()
		canvas.ClipRect(graphics.NewRect(ir.X+2, labelY-font.Baseline(), ir.Width-4, font.Baseline()+4),
			pathop.Intersect, false)
		t.Draw(canvas, cx-t.Width()/2, labelY)
		canvas.Restore()
	}
}

var _ meui.EventListener = (*NavigationRail)(nil)
var _ meui.HoverTarget = (*NavigationRail)(nil)
