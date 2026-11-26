package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/theme"
)

// ListItemVariant selects a row's height by how many lines it carries. M3 publishes the three
// heights, and they are what makes a list scannable: a two-line row that is the same height as a
// one-line row makes the eye re-read every line.
type ListItemVariant uint8

const (
	// ListItemOneLine is the default: a single line of primary text.
	ListItemOneLine ListItemVariant = iota
	// ListItemTwoLine adds a secondary line.
	ListItemTwoLine
	// ListItemThreeLine is the tallest: a secondary line that may wrap.
	ListItemThreeLine
)

// ListItem is one row of a list: an optional leading icon, one to three lines of text, and an
// optional trailing icon or control. It is the workhorse the plan names — a `DataTable` becomes a
// list of these on a phone, and a `ListView` row should be one.
//
// The text is drawn by the toolkit's own `Label`, so truncation, measurement and font derivation
// are the one implementation rather than a second one written for rows (AGENTS.md). The row itself
// owns only the geometry and the interaction: what a tap MEANS is the app's, reported through
// OnClicked.
type ListItem struct {
	meui.BaseWidget
	*theme.LabelTheme

	leading        string
	trailing       string
	trailingWidget meui.Widget
	variant        ListItemVariant
	selected       bool
	divider        bool

	primaryLabel   *Label
	secondaryLabel *Label

	hovered bool
	pressed bool

	// OnClicked fires when the row is tapped.
	OnClicked func()
}

// ListItem metrics. The slot is a 24px icon inside 16px of padding, and never under the touch
// floor; the text is inset by the slot on the side it is on.
const (
	ListItemIconSize = float32(24)
	ListItemPad      = float32(16)
	// ListItemSlot is the leading/trailing slot's width: an icon plus its padding on both sides.
	ListItemSlot = ListItemIconSize + 2*ListItemPad
)

// NewListItem builds an empty one-line row.
func NewListItem() *ListItem {
	l := &ListItem{
		BaseWidget: *meui.NewBaseWidget(),
		LabelTheme: theme.DefaultLabelTheme.Copy(),
		variant:    ListItemOneLine,
	}
	l.primaryLabel = NewLabel()
	l.primaryLabel.SetWrapMode(WrapEllipsis)
	l.primaryLabel.SetVAlign("middle")
	l.primaryLabel.SetParent(l)

	l.secondaryLabel = NewLabel()
	l.secondaryLabel.SetWrapMode(WrapEllipsis)
	l.secondaryLabel.SetVAlign("middle")
	l.secondaryLabel.SetSecondary(true)
	l.secondaryLabel.SetParent(l)
	return l
}

// ── content ─────────────────────────────────────────────────────────

// SetLeading sets the leading icon's SVG name; an empty name draws no icon.
func (l *ListItem) SetLeading(icon string) *ListItem {
	l.leading = icon
	l.MarkForRedraw()
	return l
}

func (l *ListItem) SetPrimary(text string) *ListItem {
	l.primaryLabel.SetText(text)
	l.MarkForRedraw()
	return l
}

func (l *ListItem) SetSecondary(text string) *ListItem {
	l.secondaryLabel.SetText(text)
	// A secondary line makes it a two-line row unless the app asked for three; the height follows
	// the content rather than the caller having to remember both.
	if text != "" && l.variant == ListItemOneLine {
		l.variant = ListItemTwoLine
	}
	l.MarkForRedraw()
	return l
}

func (l *ListItem) SetTrailing(icon string) *ListItem {
	l.trailing = icon
	l.MarkForRedraw()
	return l
}

// SetTrailingWidget installs a control at the trailing edge — a switch, a checkbox, a button. It
// wins over the trailing icon.
func (l *ListItem) SetTrailingWidget(w meui.Widget) *ListItem {
	l.trailingWidget = w
	if w != nil {
		w.SetParent(l)
	}
	l.MarkForRedraw()
	return l
}

func (l *ListItem) SetVariant(v ListItemVariant) *ListItem {
	l.variant = v
	l.MarkForRedraw()
	return l
}

func (l *ListItem) Variant() ListItemVariant { return l.variant }

func (l *ListItem) SetSelected(on bool) *ListItem {
	l.selected = on
	l.MarkForRedraw()
	return l
}

func (l *ListItem) Selected() bool { return l.selected }

// SetDivider draws a hairline on the row's bottom edge. It is opt-in: a list of three rows reads
// better without them, and a table of thirty reads better with.
func (l *ListItem) SetDivider(on bool) *ListItem {
	l.divider = on
	l.MarkForRedraw()
	return l
}

// SetBoundProperty implements the QML binding hook.
func (l *ListItem) SetBoundProperty(name string, value interface{}) {
	switch name {
	case "text", "primary":
		if s, ok := value.(string); ok {
			l.SetPrimary(s)
		}
	case "secondary", "secondaryText":
		if s, ok := value.(string); ok {
			l.SetSecondary(s)
		}
	case "leading":
		if s, ok := value.(string); ok {
			l.SetLeading(s)
		}
	case "trailing":
		if s, ok := value.(string); ok {
			l.SetTrailing(s)
		}
	case "selected":
		if b, ok := value.(bool); ok {
			l.SetSelected(b)
		}
	case "variant":
		// 0 one line, 1 two, 2 three — the markup's way of saying which height it wants.
		l.SetVariant(ListItemVariant(int(toFloat32(value))))
	}
}

// ── geometry ────────────────────────────────────────────────────────

// RowHeight is the row's height: the theme's row metric, plus a line's worth for each extra line,
// and never under the touch floor.
func (l *ListItem) RowHeight() float32 {
	h := theme.DefaultTheme.Metrics.ListRowHeight
	if h <= 0 {
		h = 56
	}
	switch l.variant {
	case ListItemTwoLine:
		h += 16
	case ListItemThreeLine:
		h += 32
	}
	if floor := theme.DefaultTheme.Metrics.MinTouchTarget; h < floor {
		h = floor
	}
	return h
}

// slotSize is a leading/trailing slot's width: the icon plus padding, never under the floor.
func (l *ListItem) slotSize() float32 {
	s := ListItemSlot
	if floor := theme.DefaultTheme.Metrics.MinTouchTarget; s < floor {
		s = floor
	}
	return s
}

// LeadingRect is the leading slot's rect.
func (l *ListItem) LeadingRect(bounds graphics.Rect) graphics.Rect {
	s := l.slotSize()
	if s > bounds.Width {
		s = bounds.Width
	}
	return graphics.NewRect(bounds.X, bounds.Y, s, bounds.Height)
}

// TrailingRect is the trailing slot's rect, pinned to the right edge.
func (l *ListItem) TrailingRect(bounds graphics.Rect) (graphics.Rect, bool) {
	if l.trailing == "" && l.trailingWidget == nil {
		return graphics.Rect{}, false
	}
	s := l.slotSize()
	if s > bounds.Width {
		s = bounds.Width
	}
	return graphics.NewRect(bounds.X+bounds.Width-s, bounds.Y, s, bounds.Height), true
}

// TextRect is what is left for the text, between the two slots.
func (l *ListItem) TextRect(bounds graphics.Rect) graphics.Rect {
	x := bounds.X + ListItemPad
	if l.leading != "" {
		x = l.LeadingRect(bounds).X + l.LeadingRect(bounds).Width
	}
	right := bounds.X + bounds.Width - ListItemPad
	if r, ok := l.TrailingRect(bounds); ok {
		right = r.X
	}
	w := right - x
	if w < 0 {
		w = 0
	}
	return graphics.NewRect(x, bounds.Y, w, bounds.Height)
}

// localBounds is the widget's own rect in the LOCAL space the event methods are handed — which is
// what FrameRect is NOT. FrameRect is PARENT-relative, so a containment test against it is right
// only for a widget sitting at its parent's origin, which is exactly the case a test that places one
// at (0,0) cannot tell apart from a correct one.
func (l *ListItem) localBounds() graphics.Rect {
	f := l.FrameRect()
	return graphics.NewRect(0, 0, f.Width, f.Height)
}

// IconRect is an icon's square inside a slot, vertically centred.
func (l *ListItem) IconRect(slot graphics.Rect) graphics.Rect {
	size := ListItemIconSize
	if size > slot.Width {
		size = slot.Width
	}
	if size > slot.Height {
		size = slot.Height
	}
	return graphics.NewRect(slot.X+(slot.Width-size)/2, slot.Y+(slot.Height-size)/2, size, size)
}

func (l *ListItem) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	return w, l.RowHeight()
}

// ── interaction ─────────────────────────────────────────────────────

func (l *ListItem) OnMouseDown(x, y float32, button int) bool {
	if !l.Enabled() || button != mouseButtonLeft {
		return false
	}
	// A control in the trailing slot owns its own presses; the row must not steal them.
	if l.trailingWidget != nil {
		if r, ok := l.TrailingRect(l.localBounds()); ok && r.Contains(x, y) {
			return false
		}
	}
	l.pressed = true
	l.MarkForRedraw()
	return true
}

func (l *ListItem) OnMouseUp(x, y float32, button int) bool {
	if button != mouseButtonLeft || !l.pressed {
		return false
	}
	l.pressed = false
	l.MarkForRedraw()
	// A tap commits on RELEASE and only while the pointer is still on the row, so a mis-touch is
	// recoverable by sliding off.
	if l.ContainsPoint(x, y) && l.OnClicked != nil {
		l.OnClicked()
	}
	return true
}

func (l *ListItem) OnMouseMove(x, y float32) {
	h := l.ContainsPoint(x, y)
	if h != l.hovered {
		l.hovered = h
		l.MarkForRedraw()
	}
}

// HoverAt implements HoverTarget, so the row is told when the pointer leaves it.
func (l *ListItem) HoverAt(x, y float32) { l.OnMouseMove(x, y) }

// ── paint ───────────────────────────────────────────────────────────

func (l *ListItem) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !l.Visible() || rect.Width < 1 || rect.Height < 1 {
		return
	}
	canvas.Save()
	canvas.Translate(rect.X, rect.Y)
	bounds := graphics.NewRect(0, 0, rect.Width, rect.Height)

	// The background by state: selected first, because a selected row that is also hovered must
	// still read as selected.
	fill := graphics.Color(0)
	switch {
	case l.pressed:
		fill = theme.DefaultTheme.Pressed
	case l.selected:
		fill = theme.DefaultTheme.Selected
	case l.hovered:
		fill = theme.DefaultTheme.Hovered
	}
	if fill != 0 {
		p := graphics.NewPaint()
		p.SetColor(fill)
		canvas.DrawRect(bounds, p)
	}

	primaryInk := l.TextInk
	secondaryInk := l.TextSecondary
	if !l.Enabled() {
		primaryInk = theme.DefaultTheme.DisabledText
		secondaryInk = theme.DefaultTheme.DisabledText
	}

	if l.leading != "" {
		l.paintIcon(canvas, l.IconRect(l.LeadingRect(bounds)), l.leading, primaryInk)
	}

	// The text block: one line centred, or two lines centred as a pair.
	tr := l.TextRect(bounds)
	font := l.font()
	line := font.LineHeight()
	if line <= 0 {
		line = 18
	}
	primary := l.primaryLabel.Text
	secondary := l.secondaryLabel.Text

	switch {
	case secondary == "":
		l.paintText(canvas, tr, primary, font, primaryInk, true)
	case l.variant == ListItemThreeLine:
		// Three-line rows let the secondary WRAP, so it gets the room the extra height bought.
		l.paintText(canvas, graphics.NewRect(tr.X, tr.Y+ListItemPad, tr.Width, line), primary, font, primaryInk, true)
		wrap := l.secondaryLabel
		wrap.SetWrapMode(WrapWordWrap)
		wrap.SetFrameRect(graphics.NewRect(tr.X, tr.Y+ListItemPad+line, tr.Width, tr.Height-2*ListItemPad-line))
		wrap.Paint(canvas, wrap.FrameRect())
	default:
		block := line * 2
		top := tr.Y + (tr.Height-block)/2
		l.paintText(canvas, graphics.NewRect(tr.X, top, tr.Width, line), primary, font, primaryInk, true)
		l.paintText(canvas, graphics.NewRect(tr.X, top+line, tr.Width, line), secondary, font, secondaryInk, true)
	}

	if r, ok := l.TrailingRect(bounds); ok {
		if l.trailingWidget != nil {
			cr := l.IconRect(r)
			l.trailingWidget.SetFrameRect(cr)
			l.trailingWidget.Paint(canvas, cr)
		} else {
			l.paintIcon(canvas, l.IconRect(r), l.trailing, secondaryInk)
		}
	}

	if l.divider && theme.DefaultTheme.BorderSubtle != 0 {
		// BorderSubtle is the theme's own separator role ("separators, table grids"), so a divider
		// follows the palette rather than being a colour this widget invented.
		p := graphics.NewPaint()
		p.SetColor(theme.DefaultTheme.BorderSubtle)
		canvas.DrawRect(graphics.NewRect(ListItemPad, bounds.Height-1, bounds.Width-ListItemPad, 1), p)
	}
	canvas.Restore()
}

// paintIcon draws an SVG icon, falling back to the icon's NAME as a glyph so an icon that is not in
// the set still says something.
func (l *ListItem) paintIcon(canvas *graphics.Canvas, r graphics.Rect, name string, ink graphics.Color) {
	// "chevron" is what an app writes for a disclosure row; the set spells it chevron_right.
	if name == "chevron" {
		name = "chevron_right"
	}
	if doc := loadIconDoc(name); doc != nil {
		meui.DrawSVG(canvas, doc, r.X, r.Y, r.Width, r.Height)
		return
	}
	font := l.font()
	p := graphics.NewPaint()
	p.SetColor(ink)
	t := graphics.NewText(name, &graphics.TextDecoration{Font: font, OnBackgroundInk: ink})
	canvas.Save()
	canvas.ClipRect(r, pathop.Intersect, false)
	t.Draw(canvas, r.X+(r.Width-t.Width())/2, r.Y+(r.Height+font.Baseline())/2)
	canvas.Restore()
}

// paintText draws one clipped line, vertically centred in its rect.
func (l *ListItem) paintText(canvas *graphics.Canvas, r graphics.Rect, text string, font graphics.Font, ink graphics.Color, ellipsis bool) {
	if text == "" {
		return
	}
	canvas.Save()
	canvas.ClipRect(r, pathop.Intersect, false)
	if ellipsis {
		// Truncate to the row's width rather than letting the clip cut mid-glyph: the Label's own
		// ellipsis rule, applied to a single line.
		label := l.primaryLabel
		if ink == l.TextSecondary {
			label = l.secondaryLabel
		}
		label.SetFrameRect(r)
		label.Paint(canvas, r)
		canvas.Restore()
		return
	}
	p := graphics.NewPaint()
	p.SetColor(ink)
	t := graphics.NewText(text, &graphics.TextDecoration{Font: font, OnBackgroundInk: ink})
	t.Draw(canvas, r.X, r.Y+(r.Height+font.Baseline())/2-font.Baseline()/2)
	canvas.Restore()
}

func (l *ListItem) font() graphics.Font {
	if l.LabelTheme != nil && l.LabelTheme.Font != nil {
		return l.LabelTheme.Font
	}
	if f := theme.DefaultTheme.BaseFont; f != nil {
		return f
	}
	return graphics.SystemFont
}

var _ meui.EventListener = (*ListItem)(nil)
var _ meui.HoverTarget = (*ListItem)(nil)
