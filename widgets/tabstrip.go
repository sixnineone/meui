package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

// tabstrip.go — TabStrip: a segmented control that looks like a tab bar.
//
// It behaves as a radio group — exactly one item is selected, clicking selects, and a
// client listens for the change — but it draws its items with the SAME painter a
// TabPanel's bar uses (tabbar.go), so a strip and a panel's tabs in one window look
// like the same control.
//
// It is deliberately NOT a TabPanel with the body hidden. TabPanel keeps growing
// features a radio group has no use for — content panes, close buttons, drag reorder,
// tab detaching, a tab context menu — and inheriting those would mean every one of them
// had to be switched off, then re-checked whenever TabPanel grew another. The two share
// their APPEARANCE and nothing else.
type TabStrip struct {
	meui.BaseWidget
	*theme.TabTheme

	items    []TabStripItem
	selected int
	hovered  int

	padding float32
	// scrollOffset slides the items when they are wider than the strip, and the arrows
	// appear only then: a strip usually fits, and arrows on a control that does not need
	// them are noise.
	scrollOffset float32
	showScrollL  bool
	showScrollR  bool

	// OnSelectionChanged fires when the selection moves, by click or by keyboard. It is
	// the radio group's whole interface: the index is the payload.
	OnSelectionChanged func(index int)
}

// TabStrip must satisfy the toolkit's input interfaces. Without this assertion a signature
// drift (like the bool that OnMouseMove used to return) compiles cleanly and leaves the
// widget dead to clicks — the failure mode is silence, so it has to be caught at build time.
var (
	_ meui.EventListener = (*TabStrip)(nil)
	_ meui.FocusHandler  = (*TabStrip)(nil)
)

// TabStripItem is one item in a strip: a label, an optional icon, and whether it can be
// selected. A disabled item is drawn dimmed and skipped by the keyboard, which is what
// makes a strip usable for "this option is not available right now" — a radio group
// with a dead option.
type TabStripItem struct {
	// ID is an optional caller-defined key, reported alongside the index.
	ID string
	// Label is the item's text. An item with no label and an icon is icon-only.
	Label string
	// Icon is an SVG name (see meui.LoadSVGByName) or, failing that, a glyph string.
	Icon string
	// IconSize overrides the icon box; zero uses the bar's default.
	IconSize float32
	// Disabled draws the item dimmed, refuses to select it, and skips it when the
	// keyboard walks the strip.
	Disabled bool
}

// NewTabStrip builds an empty strip pointing at the shared default tab theme, so it
// restyles with TabPanel and with a theme switch.
func NewTabStrip() *TabStrip {
	return &TabStrip{
		BaseWidget: *meui.NewBaseWidget(),
		TabTheme:   theme.DefaultTabTheme,
		selected:   -1,
		hovered:    -1,
		padding:    8,
	}
}

// SetItems replaces the items. The selection is kept when the index still exists and
// otherwise moves to the first selectable item, because a strip with no selection is
// not a radio group.
func (s *TabStrip) SetItems(items []TabStripItem) *TabStrip {
	s.items = items
	if s.selected >= len(items) {
		s.selected = -1
	}
	if s.selected < 0 || (s.selected < len(items) && items[s.selected].Disabled) {
		s.selected = s.firstEnabled()
	}
	s.SetNeedsLayout()
	s.MarkForRedraw()
	return s
}

// AddItem appends an item, selecting it if it is the first selectable one.
func (s *TabStrip) AddItem(item TabStripItem) *TabStrip {
	s.items = append(s.items, item)
	if s.selected < 0 && !item.Disabled {
		s.selected = len(s.items) - 1
	}
	s.SetNeedsLayout()
	s.MarkForRedraw()
	return s
}

// Items returns the items, for a caller that rebuilds or inspects them.
func (s *TabStrip) Items() []TabStripItem { return s.items }

// Count is the number of items.
func (s *TabStrip) Count() int { return len(s.items) }

// Selected is the selected index, or -1 when there is nothing selectable.
func (s *TabStrip) Selected() int { return s.selected }

// SelectedID is the selected item's ID, or "" when there is no selection.
func (s *TabStrip) SelectedID() string {
	if s.selected < 0 || s.selected >= len(s.items) {
		return ""
	}
	return s.items[s.selected].ID
}

// SelectedLabel is the selected item's label, or "" when there is no selection.
func (s *TabStrip) SelectedLabel() string {
	if s.selected < 0 || s.selected >= len(s.items) {
		return ""
	}
	return s.items[s.selected].Label
}

// SetSelected selects an index. It reports false when the index is out of range or the
// item is disabled — a disabled item is not selectable, which is the point of having
// one — and fires OnSelectionChanged when the selection actually moves.
func (s *TabStrip) SetSelected(index int) bool {
	if index < 0 || index >= len(s.items) {
		return false
	}
	if s.items[index].Disabled {
		return false
	}
	if index == s.selected {
		return true
	}
	s.selected = index
	s.MarkForRedraw()
	if s.OnSelectionChanged != nil {
		s.OnSelectionChanged(index)
	}
	return true
}

// SelectID selects by ID and reports whether it found a selectable item.
func (s *TabStrip) SelectID(id string) bool {
	for i, item := range s.items {
		if item.ID == id {
			return s.SetSelected(i)
		}
	}
	return false
}

// SetItemDisabled enables or disables one item at runtime. Disabling the SELECTED item
// moves the selection to the next selectable one, because leaving the selection on a
// dead item would show a control that cannot be used.
func (s *TabStrip) SetItemDisabled(index int, disabled bool) bool {
	if index < 0 || index >= len(s.items) {
		return false
	}
	if s.items[index].Disabled == disabled {
		return true
	}
	s.items[index].Disabled = disabled
	if disabled && s.selected == index {
		s.selected = s.firstEnabled()
		if s.OnSelectionChanged != nil {
			s.OnSelectionChanged(s.selected)
		}
	}
	s.MarkForRedraw()
	return true
}

// SetEnabled disables the whole strip, which dims every item and refuses every press.
func (s *TabStrip) SetEnabled(enabled bool) *TabStrip {
	s.BaseWidget.SetEnabled(enabled)
	s.MarkForRedraw()
	return s
}

// firstEnabled is the first item a user can actually pick.
func (s *TabStrip) firstEnabled() int {
	for i, item := range s.items {
		if !item.Disabled {
			return i
		}
	}
	return -1
}

// barItem describes an item to the shared painter.
func (s *TabStrip) barItem(i int) tabBarItem {
	item := s.items[i]
	return tabBarItem{
		Title:    item.Label,
		Icon:     item.Icon,
		IconSize: item.IconSize,
		Active:   i == s.selected,
		Hovered:  i == s.hovered,
		Disabled: item.Disabled || !s.Enabled(),
	}
}

// font is the strip's resolved font, through the same fallback chain TabPanel uses.
func (s *TabStrip) font() graphics.Font { return tabBarFont(s.TabTheme) }

// PreferredSize is the strip's natural size: the width its items need and the bar's
// height. Note that the layout package does NOT consult this — it lays widgets out from
// their explicit width/height — so a caller still sets the strip's height (the bar
// height) and lets the layout give it a width. PreferredSize is for an app that sizes
// the strip itself.
func (s *TabStrip) PreferredSize() (float32, float32) {
	font := s.font()
	w := s.padding
	for i := range s.items {
		w += s.itemWidth(font, i) + 2
	}
	return w + s.padding, s.TabTheme.TabHeight
}

// itemWidth measures one item through the shared bar measurement.
func (s *TabStrip) itemWidth(font graphics.Font, i int) float32 {
	return tabBarItemWidth(font, s.barItem(i), s.padding, s.TabTheme.CloseSize)
}

// barWidth is the width the strip lays its items out across.
func (s *TabStrip) barWidth() float32 {
	return s.FrameRect().Width
}

// itemRect is the rect of item i in the strip's own coordinates, including the scroll
// offset. It is the single source of an item's geometry: the painter, the hit test and
// the keyboard all ask it.
func (s *TabStrip) itemRect(font graphics.Font, i int) graphics.Rect {
	tabH := s.TabTheme.TabHeight
	x := s.padding - s.scrollOffset
	for j := 0; j < i && j < len(s.items); j++ {
		x += s.itemWidth(font, j) + 2
	}
	return graphics.NewRect(x, 2, s.itemWidth(font, i), tabH-2)
}

// ItemRect is item i's rect in widget-local coordinates, for a caller that wants to
// position something over it (a badge, a test click).
func (s *TabStrip) ItemRect(i int) (graphics.Rect, bool) {
	if i < 0 || i >= len(s.items) {
		return graphics.Rect{}, false
	}
	return s.itemRect(s.font(), i), true
}

// ItemAt returns the item under a widget-local point, or -1.
func (s *TabStrip) ItemAt(x, y float32) int {
	if y < 0 || y > s.TabTheme.TabHeight {
		return -1
	}
	font := s.font()
	for i := range s.items {
		r := s.itemRect(font, i)
		if x >= r.X && x <= r.X+r.Width {
			return i
		}
	}
	return -1
}

// updateScroll recomputes the overflow state, so the arrows appear only when the items
// do not fit and the offset stays inside the range.
func (s *TabStrip) updateScroll(font graphics.Font) {
	total := s.padding
	for i := range s.items {
		total += s.itemWidth(font, i) + 2
	}
	total += s.padding
	width := s.barWidth()
	maxScroll := total - width
	if maxScroll < 0 {
		maxScroll = 0
	}
	if s.scrollOffset > maxScroll {
		s.scrollOffset = maxScroll
	}
	if s.scrollOffset < 0 {
		s.scrollOffset = 0
	}
	s.showScrollL = s.scrollOffset > 0
	s.showScrollR = s.scrollOffset < maxScroll
}

// Paint draws the bar. There is no body: the strip IS the bar.
func (s *TabStrip) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !s.Visible() || len(s.items) == 0 {
		return
	}
	// The bar DRAWS WHAT FITS rather than refusing to draw.
	//
	// It used to return early whenever the rect was shorter than the theme's tab
	// height, which made a strip vanish entirely: `meui-dark` has a tab height of 24
	// and drew happily in a 26px rect, `breeze-dark` has 28 and drew NOTHING in the
	// same 26px rect. So the same markup rendered or not depending on the preset, with
	// no error anywhere — and it is why the demo's document pane had no visible tab
	// strip while the same widget was perfect in the behaviour harness, which loads no
	// preset.
	//
	// A control that is given less room than it would like should shrink. One that
	// disappears is indistinguishable from one that is broken.
	tabH := s.TabTheme.TabHeight
	if rect.Height < tabH {
		tabH = rect.Height
	}
	if rect.Width < 1 || tabH < 1 {
		return
	}
	canvas.Save()
	canvas.Translate(rect.X, rect.Y)

	font := s.font()
	disabled := !s.Enabled()
	s.updateScroll(font)

	paintTabBar(canvas, rect.Width, tabH, s.TabTheme, disabled)

	const scrollBtnW = float32(16)
	if s.showScrollL {
		p := graphics.NewPaint()
		p.SetColor(graphics.Color(0xFFA0A0A0))
		t := graphics.NewText("◀", &graphics.TextDecoration{Font: font, OnBackgroundInk: p.Color()})
		t.Draw(canvas, 2, 4+font.Baseline())
	}
	if s.showScrollR {
		p := graphics.NewPaint()
		p.SetColor(graphics.Color(0xFFA0A0A0))
		t := graphics.NewText("▶", &graphics.TextDecoration{Font: font, OnBackgroundInk: p.Color()})
		t.Draw(canvas, rect.Width-scrollBtnW-2, 4+font.Baseline())
	}

	clipX := float32(0)
	if s.showScrollL {
		clipX = scrollBtnW
	}
	clipW := rect.Width - clipX
	if s.showScrollR {
		clipW -= scrollBtnW
	}
	canvas.Save()
	canvas.ClipRect(graphics.NewRect(clipX, 0, clipW, tabH), pathop.Intersect, false)
	for i := range s.items {
		paintTab(canvas, s.itemRect(font, i), s.barItem(i), font, s.TabTheme, s.padding)
	}
	canvas.Restore()
	canvas.Restore()

	s.DrawTooltip(canvas, rect)
}

// ── interaction ──────────────────────────────────────────────────────

// OnMouseDown selects the item under the press. There is no drag: a radio group does
// not reorder itself, and a press that moved an item would be a surprise.
func (s *TabStrip) OnMouseDown(x, y float32, button int) bool {
	// An empty strip is not a control: it must not swallow presses meant for whatever
	// is behind it.
	if !s.Enabled() || button != mouseButtonLeft || len(s.items) == 0 {
		return false
	}
	const scrollBtnW = float32(16)
	if s.showScrollL && x >= 0 && x <= scrollBtnW {
		s.scrollOffset -= 100
		s.MarkForRedraw()
		return true
	}
	if s.showScrollR && x >= s.barWidth()-scrollBtnW && x <= s.barWidth() {
		s.scrollOffset += 100
		s.MarkForRedraw()
		return true
	}
	if i := s.ItemAt(x, y); i >= 0 {
		s.SetSelected(i) // a disabled item refuses, which is the point of one
		return true
	}
	// A press on the bar but not on an item is still the strip's: it must not fall
	// through to whatever is behind it.
	return y >= 0 && y <= s.TabTheme.TabHeight
}

// OnMouseMove tracks the hovered item, so the bar highlights what a press would select.
// OnMouseMove tracks the hovered item. It takes no return value because that is the
// signature meui.EventListener declares, and a widget whose signature differs does not
// satisfy the interface — the window then silently skips its OnMouseDown, so it renders
// perfectly and takes no input at all. It returned bool once, and that is exactly what
// happened: the strip was visible, correct and completely unclickable.
func (s *TabStrip) OnMouseMove(x, y float32) {
	i := s.ItemAt(x, y)
	if i == s.hovered {
		return
	}
	s.hovered = i
	s.MarkForRedraw()
}

// OnMouseLeave clears the hover, so a strip the pointer left is not left highlighted.
func (s *TabStrip) OnMouseLeave() {
	if s.hovered < 0 {
		return
	}
	s.hovered = -1
	s.MarkForRedraw()
}

// OnKeyDown walks the selection with the arrow keys, skipping disabled items, and wraps
// at the ends — what a radio group does.
// OnKeyRepeat is a key the user is HOLDING — Backspace over a word, an arrow along a row, a scrub.
// The window classifies an OS auto-repeat as a repeat rather than a fresh press (a widget without
// this method is asked once, deliberately), so this is what makes a held key work here.
func (s *TabStrip) OnKeyRepeat(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	return s.OnKeyDown(keyCode, mod)
}

func (s *TabStrip) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if !s.Enabled() || len(s.items) == 0 {
		return false
	}
	step := 0
	switch keyCode {
	case platform.KeyLeft, platform.KeyUp:
		step = -1
	case platform.KeyRight, platform.KeyDown:
		step = 1
	default:
		return false
	}
	next := s.seek(step)
	if next < 0 || next == s.selected {
		return true // consumed: the key belongs to the strip even when it cannot move
	}
	s.SetSelected(next)
	return true
}

// seek finds the next selectable item in a direction, wrapping once around.
func (s *TabStrip) seek(step int) int {
	n := len(s.items)
	if n == 0 {
		return -1
	}
	at := s.selected
	if at < 0 {
		at = 0
		if step < 0 {
			at = n - 1
		}
		if !s.items[at].Disabled {
			return at
		}
	}
	for i := 1; i <= n; i++ {
		j := ((at+step*i)%n + n) % n
		if !s.items[j].Disabled {
			return j
		}
	}
	return -1
}

// CursorAt reports the pointer cursor: a hand over a selectable item, so the strip
// reads as clickable.
func (s *TabStrip) CursorAt(x, y float32) *platform.Cursor {
	if !s.Enabled() {
		return nil
	}
	if i := s.ItemAt(x, y); i >= 0 && !s.items[i].Disabled {
		return platform.PointingCursor()
	}
	return nil
}

// Focusable lets the strip take focus, which is what makes the arrow keys reach it.
func (s *TabStrip) Focusable() bool { return s.Enabled() }

// OnFocus and OnBlur are no-ops beyond the redraw the focus ring needs.
func (s *TabStrip) OnFocus() { s.MarkForRedraw() }
func (s *TabStrip) OnBlur()  { s.MarkForRedraw() }
