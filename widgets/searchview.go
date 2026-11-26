package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/theme"
)

// SearchViewResult is one suggestion or result row.
type SearchViewResult struct {
	// ID is what OnChoose reports.
	ID string
	// Label is the row's primary text.
	Label string
	// Secondary is an optional second line.
	Secondary string
}

// SearchView is the full-screen search a phone opens from its app bar: a back affordance, a query
// field that is focused on arrival, and a list of results the app feeds it. It is the app bar's
// search action's destination — the plan's "full-screen search, opened from the app bar".
//
// It fills the area it is given, so the app shows it over the whole window (`Window.ShowModal`) or
// puts it in the shell; either way the field is a REAL `TextField`, so typing, the caret, selection
// and the on-screen keyboard are the ones the toolkit already has rather than a second
// implementation of a text box.
//
// It does not search. The app owns the query and the results: the widget reports what was typed
// (`OnQueryChanged`), what was submitted (`OnSubmit`), which result was chosen (`OnChoose`) and when
// the user asked to leave (`OnClose`).
type SearchView struct {
	meui.BaseContainer
	*theme.PopupTheme

	placeholder string
	results     []SearchViewResult
	field       *TextField

	pressed int
	hovered int
	// backHover/backPressed and clearHover/clearPressed are the two drawn controls, which have no
	// widget of their own — a chevron and a cross are not worth a Button each.
	backHover, backPressed   bool
	clearHover, clearPressed bool

	window *meui.Window

	// OnQueryChanged fires as the user types.
	OnQueryChanged func(query string)
	// OnSubmit fires when the user commits the query (Enter).
	OnSubmit func(query string)
	// OnChoose fires with the chosen result's ID.
	OnChoose func(id string)
	// OnClose fires when the user asks to leave the search.
	OnClose func()
}

// SearchView metrics. The header is a search app bar; the two drawn controls reach the touch floor
// through controlSize, and the rows are two-line list items.
const (
	SearchViewHeaderH = float32(56)
	SearchViewRowH    = float32(56)
	SearchViewPad     = float32(16)
)

// NewSearchView builds an empty search view.
func NewSearchView() *SearchView {
	s := &SearchView{
		BaseContainer: meui.BaseContainer{BaseWidget: *meui.NewBaseWidget()},
		PopupTheme:    theme.DefaultPopupTheme,
		pressed:       -1,
		hovered:       -1,
	}
	s.field = NewTextField()
	s.field.SetPlaceholder("Search")
	s.field.SetOnTextChanged(func(text string) {
		if s.OnQueryChanged != nil {
			s.OnQueryChanged(text)
		}
		s.MarkForRedraw()
	})
	s.field.SetOnSubmit(func(text string) {
		if s.OnSubmit != nil {
			s.OnSubmit(text)
		}
	})
	s.AddChild(s.field)
	return s
}

// ── content ─────────────────────────────────────────────────────────

func (s *SearchView) SetPlaceholder(text string) *SearchView {
	s.placeholder = text
	s.field.SetPlaceholder(text)
	s.MarkForRedraw()
	return s
}

func (s *SearchView) SetQuery(text string) *SearchView {
	s.field.SetText(text)
	s.MarkForRedraw()
	return s
}

func (s *SearchView) Query() string { return s.field.Text }

// SetResults replaces the rows. The app filters; the widget only draws and reports.
func (s *SearchView) SetResults(results []SearchViewResult) *SearchView {
	s.results = results
	s.pressed, s.hovered = -1, -1
	s.MarkForRedraw()
	return s
}

func (s *SearchView) Results() []SearchViewResult { return s.results }

// Field exposes the query field, for an app that wants to focus or style it.
func (s *SearchView) Field() *TextField { return s.field }

// SetWindow is how a widget hosted in the window's modal layer learns its window — and it FOCUSES
// the field, because a search view that opens without the keyboard up is a list with a decoration.
func (s *SearchView) SetWindow(w *meui.Window) {
	s.window = w
	if w != nil {
		w.SetFocusedChild(s.field)
	}
}

// Close asks to leave: it reports, and takes its own modal down when it has one.
func (s *SearchView) Close() {
	if s.window != nil {
		s.window.DismissTopModal()
	}
	if s.OnClose != nil {
		s.OnClose()
	}
}

// SetBoundProperty implements the QML binding hook.
func (s *SearchView) SetBoundProperty(name string, value interface{}) {
	switch name {
	case "text", "query":
		if str, ok := value.(string); ok {
			s.SetQuery(str)
		}
	case "placeholderText":
		if str, ok := value.(string); ok {
			s.SetPlaceholder(str)
		}
	}
}

// ── geometry ────────────────────────────────────────────────────────

// controlSize is the square a drawn control gets: at least the touch floor, because the back
// affordance and the clear control are the two most-tapped things on this screen.
func (s *SearchView) controlSize() float32 {
	if floor := theme.DefaultTheme.Metrics.MinTouchTarget; floor > SearchViewHeaderH {
		return floor
	}
	return SearchViewHeaderH
}

// HeaderRect is the search app bar's strip.
func (s *SearchView) HeaderRect(bounds graphics.Rect) graphics.Rect {
	h := SearchViewHeaderH
	if h > bounds.Height {
		h = bounds.Height
	}
	return graphics.NewRect(bounds.X, bounds.Y, bounds.Width, h)
}

// BackRect is the leading control's rect.
func (s *SearchView) BackRect(bounds graphics.Rect) graphics.Rect {
	h := s.HeaderRect(bounds)
	size := s.controlSize()
	if size > h.Height {
		size = h.Height
	}
	return graphics.NewRect(h.X, h.Y+(h.Height-size)/2, size, size)
}

// ClearRect is the trailing clear control, which exists only while there is something to clear.
func (s *SearchView) ClearRect(bounds graphics.Rect) (graphics.Rect, bool) {
	if s.field.Text == "" {
		return graphics.Rect{}, false
	}
	h := s.HeaderRect(bounds)
	size := s.controlSize()
	if size > h.Height {
		size = h.Height
	}
	return graphics.NewRect(h.X+h.Width-size, h.Y+(h.Height-size)/2, size, size), true
}

// FieldRect is what is left of the header for the query field, between the two controls.
func (s *SearchView) FieldRect(bounds graphics.Rect) graphics.Rect {
	h := s.HeaderRect(bounds)
	left := s.BackRect(bounds).Width
	right := float32(0)
	if r, ok := s.ClearRect(bounds); ok {
		right = r.Width
	}
	w := h.Width - left - right
	if w < 0 {
		w = 0
	}
	// The field is inset vertically so the app bar's strip is visible around it.
	pad := float32(8)
	fh := h.Height - 2*pad
	if fh < 0 {
		fh = h.Height
	}
	return graphics.NewRect(h.X+left, h.Y+pad, w, fh)
}

// ResultRect is the i-th result's rect, in the list below the header.
func (s *SearchView) ResultRect(i int, bounds graphics.Rect) (graphics.Rect, bool) {
	if i < 0 || i >= len(s.results) {
		return graphics.Rect{}, false
	}
	h := s.HeaderRect(bounds)
	return graphics.NewRect(bounds.X, h.Y+h.Height+float32(i)*SearchViewRowH, bounds.Width, SearchViewRowH), true
}

// localBounds is the widget's own rect in the LOCAL space the event methods are handed; FrameRect is
// parent-relative. See ListItem.localBounds.
func (s *SearchView) localBounds() graphics.Rect {
	f := s.FrameRect()
	return graphics.NewRect(0, 0, f.Width, f.Height)
}

// ResultAt reports which row holds a point, or -1.
func (s *SearchView) ResultAt(x, y float32) int {
	bounds := s.localBounds()
	for i := range s.results {
		if r, ok := s.ResultRect(i, bounds); ok && r.Contains(x, y) {
			return i
		}
	}
	return -1
}

// PreferredDimensions fills the area it is given: a full-screen view is not sized by its content.
func (s *SearchView) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	return w, h
}

// ValidateLayout places the field, which is the only child.
func (s *SearchView) ValidateLayout() {
	frame := s.FrameRect()
	if frame.Width <= 0 || frame.Height <= 0 {
		s.ClearNeedsLayout()
		return
	}
	s.field.SetFrameRect(s.FieldRect(graphics.NewRect(0, 0, frame.Width, frame.Height)))
	s.ClearNeedsLayout()
}

// ── interaction ─────────────────────────────────────────────────────

func (s *SearchView) OnMouseDown(x, y float32, button int) bool {
	if !s.Enabled() || button != mouseButtonLeft {
		return false
	}
	bounds := s.localBounds()
	if s.BackRect(bounds).Contains(x, y) {
		s.backPressed = true
		s.MarkForRedraw()
		return true
	}
	if r, ok := s.ClearRect(bounds); ok && r.Contains(x, y) {
		s.clearPressed = true
		s.MarkForRedraw()
		return true
	}
	if i := s.ResultAt(x, y); i >= 0 {
		s.pressed = i
		s.MarkForRedraw()
		return true
	}
	return false
}

func (s *SearchView) OnMouseUp(x, y float32, button int) bool {
	if button != mouseButtonLeft {
		return false
	}
	bounds := s.localBounds()

	if s.backPressed {
		s.backPressed = false
		s.MarkForRedraw()
		if s.BackRect(bounds).Contains(x, y) {
			s.Close()
			return true
		}
		return true
	}
	if s.clearPressed {
		s.clearPressed = false
		s.MarkForRedraw()
		if r, ok := s.ClearRect(bounds); ok && r.Contains(x, y) {
			s.SetQuery("")
			if s.OnQueryChanged != nil {
				s.OnQueryChanged("")
			}
			return true
		}
		return true
	}
	was := s.pressed
	s.pressed = -1
	if was < 0 {
		return false
	}
	// A result commits on RELEASE and only on the same row, so a mis-touch is recoverable.
	if i := s.ResultAt(x, y); i == was {
		id := s.results[i].ID
		if s.OnChoose != nil {
			s.OnChoose(id)
		}
	}
	s.MarkForRedraw()
	return true
}

func (s *SearchView) OnMouseMove(x, y float32) {
	bounds := s.localBounds()
	bh := s.BackRect(bounds).Contains(x, y)
	ch := false
	if r, ok := s.ClearRect(bounds); ok {
		ch = r.Contains(x, y)
	}
	i := s.ResultAt(x, y)
	if bh != s.backHover || ch != s.clearHover || i != s.hovered {
		s.backHover, s.clearHover, s.hovered = bh, ch, i
		s.MarkForRedraw()
	}
}

// HoverAt implements HoverTarget.
func (s *SearchView) HoverAt(x, y float32) { s.OnMouseMove(x, y) }

// ── paint ───────────────────────────────────────────────────────────

func (s *SearchView) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !s.Visible() || rect.Width < 1 || rect.Height < 1 {
		return
	}
	if s.NeedsLayout() {
		s.ValidateLayout()
	}
	bounds := graphics.NewRect(0, 0, rect.Width, rect.Height)

	bg := graphics.NewPaint()
	bg.SetColor(theme.DefaultTheme.Background)
	canvas.DrawRect(rect, bg)

	h := s.HeaderRect(bounds)
	hb := graphics.NewPaint()
	hb.SetColor(s.BackgroundInk)
	canvas.DrawRect(h, hb)
	if s.EdgeInk != 0 {
		edge := graphics.NewPaint()
		edge.SetColor(s.EdgeInk)
		canvas.DrawRect(graphics.NewRect(h.X, h.Y+h.Height-1, h.Width, 1), edge)
	}

	ink := s.HeadingTextInk
	s.paintControl(canvas, s.BackRect(bounds), s.backHover, s.backPressed, ink, func(c *graphics.Canvas, r graphics.Rect) {
		// The back chevron, drawn rather than looked up: the commonest control in the view should
		// not depend on an icon being in the set.
		p := graphics.NewPaint()
		p.SetColor(ink)
		p.SetStrokeWidth(2)
		cx, cy := r.X+r.Width/2, r.Y+r.Height/2
		size := r.Width * 0.25
		c.DrawLine(cx+size/2, cy-size, cx-size/2, cy, p)
		c.DrawLine(cx-size/2, cy, cx+size/2, cy+size, p)
	})
	if r, ok := s.ClearRect(bounds); ok {
		s.paintControl(canvas, r, s.clearHover, s.clearPressed, ink, func(c *graphics.Canvas, r graphics.Rect) {
			p := graphics.NewPaint()
			p.SetColor(ink)
			p.SetStrokeWidth(2)
			d := r.Width * 0.2
			cx, cy := r.X+r.Width/2, r.Y+r.Height/2
			c.DrawLine(cx-d, cy-d, cx+d, cy+d, p)
			c.DrawLine(cx+d, cy-d, cx-d, cy+d, p)
		})
	}

	// The query field is a real widget and paints itself.
	s.field.Paint(canvas, s.field.FrameRect())

	// The results.
	font := s.font()
	for i, res := range s.results {
		r, ok := s.ResultRect(i, bounds)
		if !ok {
			continue
		}
		if i == s.hovered || i == s.pressed {
			fill := theme.DefaultTheme.Hovered
			if i == s.pressed {
				fill = theme.DefaultTheme.Pressed
			}
			if fill != 0 {
				p := graphics.NewPaint()
				p.SetColor(fill)
				canvas.DrawRect(r, p)
			}
		}
		s.paintRowText(canvas, r, res, font)
	}
}

// paintControl draws one square control's hover/press fill and then its glyph.
func (s *SearchView) paintControl(canvas *graphics.Canvas, r graphics.Rect, hover, pressed bool, ink graphics.Color,
	glyph func(*graphics.Canvas, graphics.Rect)) {
	fill := theme.DefaultTheme.Hovered
	if pressed {
		fill = theme.DefaultTheme.Pressed
	}
	if (hover || pressed) && fill != 0 {
		p := graphics.NewPaint()
		p.SetColor(fill)
		canvas.DrawRoundedRect(r, r.Width/2, r.Height/2, p)
	}
	glyph(canvas, r)
}

// paintRowText draws a result's one or two lines, clipped to its row.
func (s *SearchView) paintRowText(canvas *graphics.Canvas, r graphics.Rect, res SearchViewResult, font graphics.Font) {
	canvas.Save()
	canvas.ClipRect(r, pathop.Intersect, false)
	ink := s.HeadingTextInk
	p := graphics.NewPaint()
	p.SetColor(ink)
	t := graphics.NewText(res.Label, &graphics.TextDecoration{Font: font, OnBackgroundInk: ink})
	line := font.LineHeight()
	if line <= 0 {
		line = 18
	}
	if res.Secondary == "" {
		t.Draw(canvas, r.X+SearchViewPad, r.Y+(r.Height+font.Baseline())/2-font.Baseline()/2)
	} else {
		// Two lines: the label above the secondary, the pair centred as a block.
		block := line * 2
		top := r.Y + (r.Height-block)/2
		t.Draw(canvas, r.X+SearchViewPad, top+font.Baseline())
		sp := graphics.NewPaint()
		sp.SetColor(theme.DefaultTheme.TextSecondary)
		st := graphics.NewText(res.Secondary, &graphics.TextDecoration{Font: font, OnBackgroundInk: theme.DefaultTheme.TextSecondary})
		st.Draw(canvas, r.X+SearchViewPad, top+line+font.Baseline())
	}
	canvas.Restore()
}

func (s *SearchView) font() graphics.Font {
	if s.PopupTheme != nil && s.PopupTheme.Font != nil {
		return s.PopupTheme.Font
	}
	if f := theme.DefaultTheme.BaseFont; f != nil {
		return f
	}
	return graphics.SystemFont
}

var _ meui.EventListener = (*SearchView)(nil)
var _ meui.HoverTarget = (*SearchView)(nil)
