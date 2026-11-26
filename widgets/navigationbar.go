package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/theme"
)

// NavigationBarItem is one destination in a NavigationBar.
type NavigationBarItem struct {
	// ID is an optional caller-defined key, reported alongside the index.
	ID string
	// Label is the destination's text. M3 always shows it in a bottom bar.
	Label string
	// Icon is an SVG name (see meui.LoadSVGByName) or, failing that, a glyph string.
	Icon string
	// IconSize overrides the icon box; zero uses the bar's default.
	IconSize float32
	// Disabled draws the item dimmed and refuses to select it.
	Disabled bool
}

// NavigationBar is the bottom navigation bar a phone wants: three to five destinations,
// icon over label, with an active indicator behind the current one.
//
// It exists because a TOP tab bar is the wrong control on a phone — it is out of thumb
// reach and it is not what either platform's own guidance describes. The same
// destinations in a TabStrip and a NavigationBar are the same DATA; only the control
// differs, so an app can hold one list and choose the control by form factor.
//
// Three to five is the platform's own range, and it is enforced softly rather than by
// refusing items: beyond five the items get narrow enough that their hit targets stop
// reaching the touch floor, and that is what the audit catches.
type NavigationBar struct {
	meui.BaseWidget
	*theme.TabTheme

	items    []NavigationBarItem
	selected int
	hovered  int
	pressed  int

	// OnSelectionChanged fires when the selection moves. It is the whole point of the
	// control: the bar reports, the app decides what a destination means.
	OnSelectionChanged func(index int)
}

// NewNavigationBar builds an empty bar, selected on the first destination.
func NewNavigationBar() *NavigationBar {
	return &NavigationBar{
		// BaseWidget must be initialised, not left zero: NewBaseWidget is what sets
		// enabled, and a zero BaseWidget is a DISABLED widget that silently refuses
		// every press. The constructor test caught exactly that.
		BaseWidget: *meui.NewBaseWidget(),
		TabTheme:   theme.DefaultTabTheme.Copy(),
		selected:   0,
		hovered:    -1,
		pressed:    -1,
	}
}

// ── items ───────────────────────────────────────────────────────────

func (n *NavigationBar) SetItems(items []NavigationBarItem) *NavigationBar {
	n.items = items
	if n.selected >= len(items) {
		n.selected = 0
	}
	n.MarkForRedraw()
	return n
}

func (n *NavigationBar) AddItem(item NavigationBarItem) *NavigationBar {
	n.items = append(n.items, item)
	n.MarkForRedraw()
	return n
}

func (n *NavigationBar) Items() []NavigationBarItem { return n.items }
func (n *NavigationBar) Count() int                 { return len(n.items) }
func (n *NavigationBar) Selected() int              { return n.selected }

func (n *NavigationBar) SelectedID() string {
	if n.selected >= 0 && n.selected < len(n.items) {
		return n.items[n.selected].ID
	}
	return ""
}

// SetSelected moves the selection and reports it, so a programmatic move behaves like a
// tap. It refuses a disabled destination.
func (n *NavigationBar) SetSelected(index int) bool {
	if index < 0 || index >= len(n.items) || n.items[index].Disabled {
		return false
	}
	if index == n.selected {
		return true
	}
	n.selected = index
	n.MarkForRedraw()
	if n.OnSelectionChanged != nil {
		n.OnSelectionChanged(index)
	}
	return true
}

func (n *NavigationBar) SelectID(id string) bool {
	for i := range n.items {
		if n.items[i].ID == id {
			return n.SetSelected(i)
		}
	}
	return false
}

// ── geometry ────────────────────────────────────────────────────────

// BarHeight is the bar's own height, from the platform's own number: 64 on Android, 49
// on iOS. It falls back to the tab height so a theme that has not set one still draws,
// and it is CLAMPED to the touch floor.
//
// The clamp is not decoration. The fallback is a desktop tab height — 24 on the meui
// default — and a 24px bar on a phone is a bar whose destinations cannot be hit. The
// five-destination test found this by asserting the height on a theme that sets no
// NavBarHeight, and got 24 against a floor of 48. A control that is a tap target is at
// least the floor tall, whatever the preset says.
func (n *NavigationBar) BarHeight() float32 {
	h := theme.DefaultTheme.Metrics.NavBarHeight
	if h <= 0 {
		h = n.TabTheme.TabHeight
	}
	if floor := theme.DefaultTheme.Metrics.MinTouchTarget; h < floor {
		h = floor
	}
	return h
}

// height is the bar's own height AS LAID OUT.
//
// It is deliberately not BarHeight: that is what the bar ASKS for, and the layout lays
// widgets out from their explicit size, so a QML `height: 64` gives a 64-tall bar whose
// preferred height is 24. Using the preferred height for the geometry meant the bar's
// visible height and its hittable height disagreed — a tap in the lower 40 pixels landed
// on the bar and did nothing, which is the worst kind of miss because the target is
// plainly there.
func (n *NavigationBar) height() float32 {
	if h := n.Height(); h > 0 {
		return h
	}
	return n.BarHeight()
}

// ItemRect is a destination's rect in the bar's own coordinates.
func (n *NavigationBar) ItemRect(i int) (graphics.Rect, bool) {
	if i < 0 || i >= len(n.items) {
		return graphics.Rect{}, false
	}
	w := n.Width() / float32(len(n.items))
	return graphics.NewRect(float32(i)*w, 0, w, n.height()), true
}

// ItemAt reports which destination holds a point, or -1.
func (n *NavigationBar) ItemAt(x, y float32) int {
	if len(n.items) == 0 || y < 0 || y > n.height() {
		return -1
	}
	w := n.Width() / float32(len(n.items))
	if w <= 0 {
		return -1
	}
	i := int(x / w)
	if i < 0 || i >= len(n.items) {
		return -1
	}
	return i
}

// SetBoundProperty implements the QML binding hook, so `selected: 2` in markup works
// the same way it does on a TabStrip.
func (n *NavigationBar) SetBoundProperty(name string, value interface{}) {
	switch name {
	case "selected":
		n.SetSelected(int(toFloat32(value)))
	}
}

func (n *NavigationBar) PreferredSize() (float32, float32) {
	return float32(len(n.items)) * 80, n.BarHeight()
}

// PreferredDimensions is how the LAYOUT learns the bar's height. The demo happens to give
// it an explicit height in markup, which is why this one worked before the app bar did —
// but a bar should not depend on the caller remembering.
func (n *NavigationBar) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	return w, n.BarHeight()
}

// ── interaction ─────────────────────────────────────────────────────

func (n *NavigationBar) OnMouseDown(x, y float32, button int) bool {
	if !n.Enabled() || button != mouseButtonLeft {
		return false
	}
	i := n.ItemAt(x, y)
	if i < 0 || n.items[i].Disabled {
		return false
	}
	n.pressed = i
	n.MarkForRedraw()
	return true
}

func (n *NavigationBar) OnMouseUp(x, y float32, button int) bool {
	if button != mouseButtonLeft {
		return false
	}
	was := n.pressed
	n.pressed = -1
	if was < 0 {
		return false
	}
	// Select on RELEASE, and only when the finger is still on the same destination —
	// a drag that leaves the bar cancels, which is what every platform does and what
	// makes a mis-touch recoverable.
	if i := n.ItemAt(x, y); i == was {
		n.SetSelected(i)
	}
	n.MarkForRedraw()
	return true
}

func (n *NavigationBar) OnMouseMove(x, y float32) {
	i := n.ItemAt(x, y)
	if i != n.hovered {
		n.hovered = i
		n.MarkForRedraw()
	}
}

// HoverAt implements HoverTarget, so the bar is told when the pointer leaves it rather
// than keeping the last destination lit.
func (n *NavigationBar) HoverAt(x, y float32) {
	n.OnMouseMove(x, y)
}

// ── paint ───────────────────────────────────────────────────────────

func (n *NavigationBar) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !n.Visible() || len(n.items) == 0 {
		return
	}
	h := n.height()
	if rect.Width < 1 || rect.Height < h {
		return
	}
	canvas.Save()
	canvas.Translate(rect.X, rect.Y)

	// The bar's own surface, then a hairline on the top edge — the separation M3 uses
	// instead of a shadow.
	bg := graphics.NewPaint()
	bg.SetColor(n.BackgroundInk)
	canvas.DrawRect(graphics.NewRect(0, 0, rect.Width, h), bg)
	if n.EdgeInk != 0 {
		edge := graphics.NewPaint()
		edge.SetColor(n.EdgeInk)
		canvas.DrawRect(graphics.NewRect(0, 0, rect.Width, 1), edge)
	}

	font := n.font()
	itemW := rect.Width / float32(len(n.items))
	for i, item := range n.items {
		n.paintItem(canvas, i, item, font, float32(i)*itemW, itemW, h)
	}

	canvas.Restore()
}

func (n *NavigationBar) font() graphics.Font {
	if n.TabTheme.Font != nil {
		return n.TabTheme.Font
	}
	if bf := theme.DefaultTheme.BaseFont; bf != nil {
		return bf
	}
	return graphics.SystemFont
}

func (n *NavigationBar) paintItem(canvas *graphics.Canvas, i int, item NavigationBarItem, font graphics.Font, x, w, h float32) {
	active := i == n.selected
	disabled := item.Disabled

	ink := n.TextInk
	switch {
	case disabled:
		ink = theme.DefaultTheme.DisabledText
	case active:
		ink = n.TextActive
	case i == n.hovered:
		ink = n.TextHover
	}

	// The active indicator: a pill behind the icon, which is what tells you where you
	// are without moving anything. Its width is the icon box plus padding, and it is
	// centred so the icon does not shift when the selection moves.
	iconSize := item.IconSize
	if iconSize <= 0 {
		iconSize = 24
	}
	indW := iconSize + 32
	if indW > w-8 {
		indW = w - 8
	}
	indH := iconSize + 12
	cx := x + w/2
	indY := float32(8)
	if active && n.BackgroundActive != 0 {
		ind := graphics.NewPaint()
		ind.SetColor(n.BackgroundActive)
		canvas.DrawRoundedRect(
			graphics.NewRect(cx-indW/2, indY, indW, indH), indH/2, indH/2, ind)
	}

	iconY := indY + (indH-iconSize)/2
	if item.Icon != "" {
		// An SVG name first, then a glyph — the same resolution the rest of the toolkit
		// uses, so an icon that is not in the set still draws something.
		if doc := loadIconDoc(item.Icon); doc != nil {
			meui.DrawSVG(canvas, doc, cx-iconSize/2, iconY, iconSize, iconSize)
		} else {
			p := graphics.NewPaint()
			p.SetColor(ink)
			t := graphics.NewText(item.Icon, &graphics.TextDecoration{Font: font, OnBackgroundInk: ink})
			t.Draw(canvas, cx-t.Width()/2, iconY+iconSize/2+font.Baseline()/2)
		}
	}

	// The label sits BELOW the icon, always: a bottom bar's whole job is to be
	// scannable at a glance, and an icon-only bar makes the user guess.
	//
	// It is CLIPPED to the destination, not truncated character by character. The first
	// version shrank the string until it measured under the item width, which drew
	// "Widgets" as "W" — a measuring loop that runs to one character is a bug wearing a
	// layout's clothes, and the neighbouring destination is exactly what a clip is for.
	if item.Label != "" {
		p := graphics.NewPaint()
		p.SetColor(ink)
		t := graphics.NewText(item.Label, &graphics.TextDecoration{Font: font, OnBackgroundInk: ink})
		labelW := t.Width()
		labelY := indY + indH + font.Baseline() + 2
		canvas.Save()
		canvas.ClipRect(graphics.NewRect(x+2, labelY-font.Baseline(), w-4, font.Baseline()+4),
			pathop.Intersect, false)
		t.Draw(canvas, cx-labelW/2, labelY)
		canvas.Restore()
	}
}

// loadIconDoc resolves an icon name to an SVG document, or nil. It lives here rather
// than inline so the widget can fall back to drawing the name as a glyph.
func loadIconDoc(name string) *meui.SVGDocument {
	doc, err := meui.LoadSVGByName(name)
	if err != nil {
		return nil
	}
	return doc
}

var _ meui.EventListener = (*NavigationBar)(nil)
var _ meui.HoverTarget = (*NavigationBar)(nil)
