package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/theme"
)

// TopAppBarAction is one control in an app bar.
type TopAppBarAction struct {
	// ID is the key reported to OnAction.
	ID string
	// Icon is an SVG name (see meui.LoadSVGByName) or, failing that, a glyph string.
	Icon string
	// Tooltip is shown on hover and, on a touch surface, is what a long-press would
	// report; it is also the fallback label when there is no icon.
	Tooltip  string
	Disabled bool
}

// TopAppBar is the bar at the top of a mobile screen: a leading affordance, a title, and
// trailing actions, with an overflow control when there are more actions than fit.
//
// It is the top half of the shell the NavigationBar is the bottom half of. Both are
// form-factor controls — on a desktop the same commands belong in a menu bar — but the
// bar itself has no opinion about that: it draws, reports, and lets the app decide.
//
// It does NOT own its overflow popup. A popup needs window-level z-order and dismissal,
// which the toolkit already has in Popup/ContextMenu; a bar that drew its own menu would
// be clipped by its own rect and would fight the window for input. So the overflow button
// reports a position and the app opens the menu it already has.
type TopAppBar struct {
	meui.BaseWidget
	*theme.ToolBarTheme

	title      string
	leading    string // icon name; empty draws no leading control
	leadingAlt string // the leading control's accessible name
	actions    []TopAppBarAction
	overflow   bool

	hovered int // index into leading(-1 is none) and actions
	pressed int

	// OnLeading fires when the leading affordance is used — back, menu, or close.
	OnLeading func()
	// OnAction fires with the action's ID.
	OnAction func(id string)
	// OnOverflow fires when the overflow control is used, with the button's position in
	// the bar's own coordinates, so the app can open a menu there.
	OnOverflow func(x, y float32)
}

// NewTopAppBar builds an empty bar with no leading control and no actions.
func NewTopAppBar() *TopAppBar {
	return &TopAppBar{
		BaseWidget:   *meui.NewBaseWidget(),
		ToolBarTheme: theme.DefaultToolBarTheme,
		hovered:      -1,
		pressed:      -1,
	}
}

// ── content ─────────────────────────────────────────────────────────

func (b *TopAppBar) SetTitle(title string) *TopAppBar {
	b.title = title
	b.MarkForRedraw()
	return b
}

func (b *TopAppBar) Title() string { return b.title }

// SetLeading sets the leading affordance: "back", "menu", or any other icon name. An
// empty name draws nothing, which is right for a root screen.
func (b *TopAppBar) SetLeading(icon, alt string) *TopAppBar {
	b.leading, b.leadingAlt = icon, alt
	b.MarkForRedraw()
	return b
}

func (b *TopAppBar) Leading() string { return b.leading }

func (b *TopAppBar) SetActions(actions []TopAppBarAction) *TopAppBar {
	b.actions = actions
	b.MarkForRedraw()
	return b
}

func (b *TopAppBar) Actions() []TopAppBarAction { return b.actions }

// SetOverflow draws the overflow control when true, which the app turns on only when it
// actually has hidden actions — a "more" button with nothing behind it is a dead end.
func (b *TopAppBar) SetOverflow(on bool) *TopAppBar {
	b.overflow = on
	b.MarkForRedraw()
	return b
}

func (b *TopAppBar) Overflow() bool { return b.overflow }

// ── geometry ────────────────────────────────────────────────────────

// BarHeight is the bar's height. It comes from Metrics.ToolbarHeight because a top app
// bar IS a toolbar — the same control family, and the presets already carry the
// platform's number for it (64 on Android, 44 on iOS) — rather than a second metric for
// the same quantity. The ToolBarTheme has no height of its own, so the metric is the
// only source and there is nothing to drift.
func (b *TopAppBar) BarHeight() float32 {
	if h := theme.DefaultTheme.Metrics.ToolbarHeight; h > 0 {
		return h
	}
	return 40
}

// controlSize is one control's square. It is at least the touch floor: an app bar's
// leading and trailing controls are the most-tapped things on a mobile screen, and a
// 24px icon in a 24px target is the classic way to make an app feel broken.
func (b *TopAppBar) controlSize() float32 {
	h := b.BarHeight()
	if floor := theme.DefaultTheme.Metrics.MinTouchTarget; floor > h {
		return floor
	}
	return h
}

// LeadingRect is the leading control's rect in the bar's own coordinates.
func (b *TopAppBar) LeadingRect() (graphics.Rect, bool) {
	if b.leading == "" {
		return graphics.Rect{}, false
	}
	s := b.controlSize()
	return graphics.NewRect(0, (b.BarHeight()-s)/2, s, s), true
}

// ActionRect is the i-th trailing action's rect, counting from the RIGHT.
func (b *TopAppBar) ActionRect(i int) (graphics.Rect, bool) {
	if i < 0 || i >= len(b.actions) {
		return graphics.Rect{}, false
	}
	s := b.controlSize()
	right := b.Width()
	if b.overflow {
		right -= s
	}
	return graphics.NewRect(right-s-float32(i)*s, (b.BarHeight()-s)/2, s, s), true
}

// OverflowRect is the overflow control's rect.
func (b *TopAppBar) OverflowRect() (graphics.Rect, bool) {
	if !b.overflow {
		return graphics.Rect{}, false
	}
	s := b.controlSize()
	return graphics.NewRect(b.Width()-s, (b.BarHeight()-s)/2, s, s), true
}

// TitleRect is where the title is drawn: after the leading control and before the
// trailing ones, so a long title cannot run under a button.
func (b *TopAppBar) TitleRect() graphics.Rect {
	left := float32(8)
	if _, ok := b.LeadingRect(); ok {
		left += b.controlSize()
	}
	right := b.Width() - 8
	n := len(b.actions)
	if b.overflow {
		n++
	}
	right -= float32(n) * b.controlSize()
	if right < left {
		right = left
	}
	return graphics.NewRect(left, 0, right-left, b.BarHeight())
}

func (b *TopAppBar) PreferredSize() (float32, float32) {
	return 240, b.BarHeight()
}

// PreferredDimensions is how the LAYOUT learns the bar's height, and it is not optional:
// a bar that only implements PreferredSize gets whatever its frame happened to be — 20px
// in the demo — and then PAINTS at its real height, drawing over the page below it. That
// is what happened, and it looked like a content-overflow bug rather than a bar that
// never declared itself.
func (b *TopAppBar) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	return w, b.BarHeight()
}

// ── interaction ─────────────────────────────────────────────────────

func (b *TopAppBar) hit(x, y float32) (kind string, index int) {
	if r, ok := b.LeadingRect(); ok && r.Contains(x, y) {
		return "leading", 0
	}
	for i := range b.actions {
		if r, ok := b.ActionRect(i); ok && r.Contains(x, y) {
			return "action", i
		}
	}
	if r, ok := b.OverflowRect(); ok && r.Contains(x, y) {
		return "overflow", 0
	}
	return "", -1
}

func (b *TopAppBar) OnMouseDown(x, y float32, button int) bool {
	if !b.Enabled() || button != mouseButtonLeft {
		return false
	}
	if _, i := b.hit(x, y); i < 0 {
		return false
	} else {
		b.pressed = i
	}
	b.MarkForRedraw()
	return true
}

func (b *TopAppBar) OnMouseUp(x, y float32, button int) bool {
	if button != mouseButtonLeft || b.pressed < 0 {
		return false
	}
	b.pressed = -1
	kind, i := b.hit(x, y)
	b.MarkForRedraw()
	switch kind {
	case "leading":
		if b.OnLeading != nil {
			b.OnLeading()
		}
	case "action":
		if b.actions[i].Disabled {
			return true
		}
		if b.OnAction != nil {
			b.OnAction(b.actions[i].ID)
		}
	case "overflow":
		if b.OnOverflow != nil {
			if r, ok := b.OverflowRect(); ok {
				b.OnOverflow(r.X+r.Width/2, r.Y+r.Height)
			}
		}
	}
	return true
}

func (b *TopAppBar) OnMouseMove(x, y float32) {
	if _, i := b.hit(x, y); i != b.hovered {
		b.hovered = i
		b.MarkForRedraw()
	}
}

// HoverAt implements HoverTarget so the bar is told when the pointer leaves it.
func (b *TopAppBar) HoverAt(x, y float32) { b.OnMouseMove(x, y) }

// ── paint ───────────────────────────────────────────────────────────

func (b *TopAppBar) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !b.Visible() || rect.Width < 1 || rect.Height < 1 {
		return
	}
	canvas.Save()
	canvas.Translate(rect.X, rect.Y)

	bg := graphics.NewPaint()
	bg.SetColor(b.BackgroundInk)
	canvas.DrawRect(graphics.NewRect(0, 0, rect.Width, rect.Height), bg)
	if b.EdgeInk != 0 {
		edge := graphics.NewPaint()
		edge.SetColor(b.EdgeInk)
		canvas.DrawRect(graphics.NewRect(0, rect.Height-1, rect.Width, 1), edge)
	}

	// The leading affordance. "back" is drawn as a chevron rather than looked up, so the
	// commonest control in the shell does not depend on an icon being in the set.
	if r, ok := b.LeadingRect(); ok {
		b.paintControl(canvas, r, b.leading, b.leadingAlt, false, b.pressed == 0 && b.hovered == 0)
	}

	// The title, clipped to its own rect so it cannot run under the buttons.
	if b.title != "" {
		tr := b.TitleRect()
		font := b.font()
		ink := b.TextInk
		p := graphics.NewPaint()
		p.SetColor(ink)
		t := graphics.NewText(b.title, &graphics.TextDecoration{Font: font, OnBackgroundInk: ink})
		canvas.Save()
		canvas.ClipRect(tr, pathop.Intersect, false)
		t.Draw(canvas, tr.X, tr.Y+(tr.Height+font.Baseline())/2-font.Baseline()/2)
		canvas.Restore()
	}

	for i, a := range b.actions {
		r, ok := b.ActionRect(i)
		if !ok {
			continue
		}
		b.paintControl(canvas, r, a.Icon, a.Tooltip, a.Disabled, b.hovered == i)
	}
	if r, ok := b.OverflowRect(); ok {
		b.paintControl(canvas, r, "more", "More", false, b.hovered == 0 && b.pressed == 0)
	}

	canvas.Restore()
}

func (b *TopAppBar) font() graphics.Font {
	if b.ToolBarTheme.Font != nil {
		return b.ToolBarTheme.Font
	}
	if f := theme.DefaultTheme.BaseFont; f != nil {
		return f
	}
	return graphics.SystemFont
}

// paintControl draws one square control: a press/hover fill, then the icon or glyph.
func (b *TopAppBar) paintControl(canvas *graphics.Canvas, r graphics.Rect, icon, label string, disabled, active bool) {
	if active && b.ButtonHover != 0 {
		p := graphics.NewPaint()
		p.SetColor(b.ButtonHover)
		canvas.DrawRoundedRect(r, r.Width/2, r.Height/2, p)
	}
	ink := b.TextInk
	if disabled {
		ink = theme.DefaultTheme.DisabledText
	}
	size := r.Width * 0.5
	iconX := r.X + (r.Width-size)/2
	iconY := r.Y + (r.Height-size)/2

	if icon == "back" {
		// A chevron drawn from two strokes: no icon-set dependency for the one control
		// every screen has.
		p := graphics.NewPaint()
		p.SetColor(ink)
		p.SetStrokeWidth(2)
		cx := r.X + r.Width/2
		cy := r.Y + r.Height/2
		canvas.DrawLine(cx+3, cy-size/2, cx-3, cy, p)
		canvas.DrawLine(cx-3, cy, cx+3, cy+size/2, p)
		return
	}
	if icon == "more" {
		p := graphics.NewPaint()
		p.SetColor(ink)
		d := size / 5
		cx := r.X + r.Width/2
		cy := r.Y + r.Height/2
		for i := -1; i <= 1; i++ {
			canvas.DrawCircle(cx, cy+float32(i)*(d*2), d/2, p)
		}
		return
	}
	if icon != "" {
		if doc := loadIconDoc(icon); doc != nil {
			meui.DrawSVG(canvas, doc, iconX, iconY, size, size)
			return
		}
		// Fall back to the label as a glyph, then to the icon name itself.
		text := label
		if text == "" {
			text = icon
		}
		p := graphics.NewPaint()
		p.SetColor(ink)
		t := graphics.NewText(text, &graphics.TextDecoration{Font: b.font(), OnBackgroundInk: ink})
		canvas.Save()
		canvas.ClipRect(r, pathop.Intersect, false)
		t.Draw(canvas, r.X+(r.Width-t.Width())/2, r.Y+(r.Height+b.font().Baseline())/2)
		canvas.Restore()
	}
}

var _ meui.EventListener = (*TopAppBar)(nil)
var _ meui.HoverTarget = (*TopAppBar)(nil)
