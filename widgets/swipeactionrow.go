package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/theme"
)

// SwipeAction is one action behind a row.
type SwipeAction struct {
	// ID is what OnAction reports.
	ID string
	// Label is the action's text. A swipe row's actions are text buttons: an icon alone gives no
	// warning about what is behind the row.
	Label string
	// Destructive fills it with the error ink — the action that deletes something.
	Destructive bool
	// OnAction is this action's own callback; the row's OnAction fires as well, with the ID.
	OnAction func()
}

// SwipeActionRow is a list row you swipe sideways to reveal the actions behind it — the gesture that
// makes a list feel like a phone rather than a table. It hosts one content widget (a `ListItem`, in
// practice) and draws the actions in the strip the content uncovers.
//
// It is a CONTAINER THAT IS ALWAYS THE HIT (the `ScrollPane` pattern: `PanelAt` returns the row
// itself), because a swipe cannot be recognized by the child that owns the press. So the row takes
// the press, decides whether the gesture was a TAP or a SWIPE, and forwards a tap to its content —
// which is what keeps a row's own click working through the gesture.
//
// Only TRAILING actions are implemented: the destructive ones a left swipe reveals. A leading swipe
// is a different gesture with a different meaning (usually "mark" or "pin"), and guessing at it
// would be worse than not having it.
type SwipeActionRow struct {
	meui.BaseContainer
	*theme.PopupTheme

	content meui.Widget
	actions []SwipeAction

	// offset is the content's displacement: 0 closed, negative as the trailing actions are revealed.
	offset float32

	dragging bool
	sliding  bool
	pressX   float32
	pressY   float32

	actionPressed int

	// OnAction fires with the action's ID.
	OnAction func(id string)
	// OnOpened and OnClosed report the state change, so an app can close the other rows.
	OnOpened func()
	OnClosed func()
}

// SwipeActionRow metrics.
const (
	// swipeSlop is how far the pointer moves before a press becomes a swipe. Without it every tap
	// with a pixel of tremor would nudge the row.
	swipeSlop = float32(8)
	// SwipeFullFraction is how much of the ROW a swipe must cover to run the first action outright
	// on release, which is the platform's "swipe all the way to delete".
	SwipeFullFraction = float32(0.5)
	// swipeActionMinW is one action's width floor: an action is a target.
	swipeActionMinW = float32(72)
)

// NewSwipeActionRow builds an empty row with no content and no actions.
func NewSwipeActionRow() *SwipeActionRow {
	return &SwipeActionRow{
		BaseContainer: meui.BaseContainer{BaseWidget: *meui.NewBaseWidget()},
		PopupTheme:    theme.DefaultPopupTheme,
		actionPressed: -1,
	}
}

// ── content ─────────────────────────────────────────────────────────

func (s *SwipeActionRow) SetContent(w meui.Widget) *SwipeActionRow {
	s.content = w
	if w != nil {
		w.SetParent(s)
	}
	s.MarkForRedraw()
	return s
}

func (s *SwipeActionRow) Content() meui.Widget { return s.content }

// AddChild makes a QML child the CONTENT, the way ScrollPane does: the loader adds children through
// AddChild, so a container that keeps its content elsewhere has to say so or the markup's child
// lands in an unlaid-out list and never paints.
func (s *SwipeActionRow) AddChild(child meui.Widget) { s.SetContent(child) }

func (s *SwipeActionRow) SetActions(actions []SwipeAction) *SwipeActionRow {
	s.actions = actions
	s.actionPressed = -1
	if s.offset < -s.ActionsWidth() {
		s.offset = -s.ActionsWidth()
	}
	s.MarkForRedraw()
	return s
}

func (s *SwipeActionRow) Actions() []SwipeAction { return s.actions }

// ── state ───────────────────────────────────────────────────────────

// Offset is the content's displacement; 0 is closed.
func (s *SwipeActionRow) Offset() float32 { return s.offset }

// IsOpen reports whether any action is revealed.
func (s *SwipeActionRow) IsOpen() bool { return s.offset < 0 }

// Open reveals the actions, Close hides them.
func (s *SwipeActionRow) Open() {
	if len(s.actions) == 0 || s.offset == -s.ActionsWidth() {
		return
	}
	s.offset = -s.ActionsWidth()
	s.MarkForRedraw()
	if s.OnOpened != nil {
		s.OnOpened()
	}
}

func (s *SwipeActionRow) Close() {
	if s.offset == 0 {
		return
	}
	s.offset = 0
	s.actionPressed = -1
	s.MarkForRedraw()
	if s.OnClosed != nil {
		s.OnClosed()
	}
}

// ── geometry ────────────────────────────────────────────────────────

// ActionsWidth is the strip the actions occupy.
func (s *SwipeActionRow) ActionsWidth() float32 {
	w := float32(0)
	for _, a := range s.actions {
		w += s.actionWidth(a)
	}
	return w
}

// actionWidth is one action's width: its label plus padding, never under the floor, because an
// action is a target a thumb has to find without looking.
func (s *SwipeActionRow) actionWidth(a SwipeAction) float32 {
	font := s.font()
	t := graphics.NewText(a.Label, &graphics.TextDecoration{Font: font, OnBackgroundInk: graphics.RGB(255, 255, 255)})
	w := t.Width() + 32
	if w < swipeActionMinW {
		w = swipeActionMinW
	}
	return w
}

// ActionRect is the i-th action's rect. The actions are laid out from the RIGHT edge leftward in the
// order given, so the FIRST one is the outermost — which is where the thumb already is when the row
// finishes sliding, and the destructive action is the one an app puts first.
func (s *SwipeActionRow) ActionRect(i int, bounds graphics.Rect) (graphics.Rect, bool) {
	if i < 0 || i >= len(s.actions) {
		return graphics.Rect{}, false
	}
	right := bounds.X + bounds.Width
	for j, a := range s.actions {
		w := s.actionWidth(a)
		r := graphics.NewRect(right-w, bounds.Y, w, bounds.Height)
		if j == i {
			return r, true
		}
		right -= w
	}
	return graphics.Rect{}, false
}

// localBounds is the widget's own rect in the LOCAL space the event methods are handed; FrameRect is
// parent-relative. See ListItem.localBounds.
func (s *SwipeActionRow) localBounds() graphics.Rect {
	f := s.FrameRect()
	return graphics.NewRect(0, 0, f.Width, f.Height)
}

// ActionAt reports which action holds a point, or -1. An action that is still COVERED by the content
// is not a target: the row's own body must not fire the delete behind it.
func (s *SwipeActionRow) ActionAt(x, y float32) int {
	bounds := s.localBounds()
	revealed := bounds.X + bounds.Width + s.offset
	if x < revealed {
		return -1
	}
	for i := range s.actions {
		if r, ok := s.ActionRect(i, bounds); ok && r.Contains(x, y) {
			return i
		}
	}
	return -1
}

// ContentRect is the content's rect while closed. The paint translates it by the offset, so the
// content's own frame stays put and the layout is not disturbed by a gesture.
func (s *SwipeActionRow) ContentRect(bounds graphics.Rect) graphics.Rect {
	return graphics.NewRect(bounds.X, bounds.Y, bounds.Width, bounds.Height)
}

// fullSwipeThreshold is how far the swipe must travel to run the first action on release: half the
// row, or the whole action strip when that is SHORTER.
//
// The second term is load-bearing. The content can only slide as far as the actions are wide — there
// is nothing behind them to uncover — so a threshold of half the row is UNREACHABLE on a row whose
// actions are narrower than that, and the shortcut silently never fires. Dragging far enough to
// expose every action IS "all the way", and that is what this measures.
func (s *SwipeActionRow) fullSwipeThreshold(bounds graphics.Rect) float32 {
	half := bounds.Width * SwipeFullFraction
	if aw := s.ActionsWidth(); aw > 0 && aw < half {
		return aw
	}
	return half
}

func (s *SwipeActionRow) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	if s.content != nil {
		if m, ok := s.content.(interface {
			PreferredDimensions(interface{}, float32, float32) (float32, float32)
		}); ok {
			return m.PreferredDimensions(s.content, w, h)
		}
	}
	return w, h
}

// PanelAt returns the ROW itself, always: a swipe cannot be recognized by the child that would
// otherwise take the press. The row then decides tap-versus-swipe and forwards a tap.
func (s *SwipeActionRow) PanelAt(x, y float32) meui.Widget { return s }

func (s *SwipeActionRow) ValidateLayout() {
	frame := s.FrameRect()
	if frame.Width <= 0 || frame.Height <= 0 {
		s.ClearNeedsLayout()
		return
	}
	if s.content != nil {
		s.content.SetFrameRect(graphics.NewRect(0, 0, frame.Width, frame.Height))
	}
	s.ClearNeedsLayout()
}

// ── interaction ─────────────────────────────────────────────────────

func (s *SwipeActionRow) OnMouseDown(x, y float32, button int) bool {
	if !s.Enabled() || button != mouseButtonLeft {
		return false
	}
	// An OPEN row's actions own the press, and a press on its exposed body closes it rather than
	// acting on the row underneath.
	if s.IsOpen() {
		if i := s.ActionAt(x, y); i >= 0 {
			s.actionPressed = i
			s.MarkForRedraw()
			return true
		}
		s.Close()
		return true
	}
	s.pressX, s.pressY = x, y
	s.dragging = true
	s.sliding = false
	s.actionPressed = -1
	// The content is told, so a row still shows its pressed state through the gesture.
	s.forwardDown(x, y, button)
	return true
}

func (s *SwipeActionRow) OnMouseMove(x, y float32) {
	if !s.dragging {
		// Not dragging: hover belongs to the content, which is not the hit (the row is).
		s.forwardMove(x, y)
		return
	}
	dx := x - s.pressX
	if !s.sliding {
		if dx > -swipeSlop && dx < swipeSlop {
			return
		}
		s.sliding = true
	}
	off := dx
	if off > 0 {
		off = 0 // trailing actions only: there is nothing to reveal to the right
	}
	if max := -s.ActionsWidth(); off < max {
		off = max
	}
	if off != s.offset {
		s.offset = off
		s.MarkForRedraw()
	}
}

func (s *SwipeActionRow) OnMouseUp(x, y float32, button int) bool {
	if button != mouseButtonLeft {
		return false
	}
	if s.actionPressed >= 0 {
		i := s.actionPressed
		s.actionPressed = -1
		if r, ok := s.ActionRect(i, s.localBounds()); ok && r.Contains(x, y) {
			s.fire(i)
			return true
		}
		s.MarkForRedraw()
		return true
	}
	if !s.dragging {
		return false
	}
	s.dragging = false
	slid := s.sliding
	s.sliding = false

	if !slid {
		// A TAP: hand the release to the content, which is what keeps a row's own click working
		// through a gesture the row had to intercept.
		s.forwardUp(x, y, button)
		return true
	}
	// A full swipe runs the first action outright — the platform's "swipe all the way to delete".
	if s.offset <= -s.fullSwipeThreshold(s.localBounds()) && len(s.actions) > 0 {
		s.fire(0)
		return true
	}
	// Otherwise it snaps: past half the strip stays open, short of it closes.
	if s.offset <= -s.ActionsWidth()/2 {
		s.Open()
	} else {
		s.Close()
	}
	return true
}

// HoverAt implements HoverTarget, so the row is told when the pointer leaves it.
func (s *SwipeActionRow) HoverAt(x, y float32) { s.forwardMove(x, y) }

// fire runs an action and closes the row.
func (s *SwipeActionRow) fire(i int) {
	if i < 0 || i >= len(s.actions) {
		return
	}
	a := s.actions[i]
	s.Close()
	if a.OnAction != nil {
		a.OnAction()
	}
	if s.OnAction != nil {
		s.OnAction(a.ID)
	}
}

// forwardDown/Move/Up pass a point to the content in ITS coordinates: the content is drawn offset by
// the swipe, so a row's own hit test has to be shifted by the same amount.
func (s *SwipeActionRow) forwardDown(x, y float32, button int) {
	if s.content == nil {
		return
	}
	if el, ok := s.content.(interface {
		OnMouseDown(float32, float32, int) bool
	}); ok {
		el.OnMouseDown(x-s.offset, y, button)
	}
}

func (s *SwipeActionRow) forwardMove(x, y float32) {
	if s.content == nil {
		return
	}
	if el, ok := s.content.(interface{ OnMouseMove(float32, float32) }); ok {
		el.OnMouseMove(x-s.offset, y)
	}
}

func (s *SwipeActionRow) forwardUp(x, y float32, button int) {
	if s.content == nil {
		return
	}
	if el, ok := s.content.(interface {
		OnMouseUp(float32, float32, int) bool
	}); ok {
		el.OnMouseUp(x-s.offset, y, button)
	}
}

// ── paint ───────────────────────────────────────────────────────────

func (s *SwipeActionRow) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !s.Visible() || rect.Width < 1 || rect.Height < 1 {
		return
	}
	if s.NeedsLayout() {
		s.ValidateLayout()
	}
	canvas.Save()
	canvas.Translate(rect.X, rect.Y)
	bounds := graphics.NewRect(0, 0, rect.Width, rect.Height)

	// The actions, in the strip the content uncovers.
	font := s.font()
	for i, a := range s.actions {
		r, ok := s.ActionRect(i, bounds)
		if !ok {
			continue
		}
		ink := theme.DefaultTheme.SurfaceVariant
		if a.Destructive {
			ink = theme.DefaultTheme.Error
		}
		if i == s.actionPressed {
			ink = theme.DefaultTheme.Pressed
		}
		if ink != 0 {
			p := graphics.NewPaint()
			p.SetColor(ink)
			canvas.DrawRect(r, p)
		}
		textInk := theme.DefaultTheme.TextPrimary
		if a.Destructive || i == s.actionPressed {
			textInk = graphics.Color(0xFFFFFFFF)
		}
		tp := graphics.NewPaint()
		tp.SetColor(textInk)
		t := graphics.NewText(a.Label, &graphics.TextDecoration{Font: font, OnBackgroundInk: textInk})
		canvas.Save()
		canvas.ClipRect(r, pathop.Intersect, false)
		t.Draw(canvas, r.X+(r.Width-t.Width())/2, r.Y+(r.Height+font.Baseline())/2-font.Baseline()/2)
		canvas.Restore()
	}

	// The content, offset by the swipe, over an OPAQUE plate: a row that paints no background of its
	// own would otherwise let the actions show through it while the row is closed.
	if s.content != nil {
		plate := graphics.NewPaint()
		plateInk := s.BackgroundInk
		if plateInk == 0 {
			plateInk = theme.DefaultTheme.Background
		}
		plate.SetColor(plateInk)
		cr := s.ContentRect(bounds)
		canvas.Save()
		canvas.Translate(s.offset, 0)
		canvas.DrawRect(cr, plate)
		s.content.SetFrameRect(cr)
		s.content.Paint(canvas, cr)
		canvas.Restore()
	}
	canvas.Restore()
}

func (s *SwipeActionRow) font() graphics.Font {
	if s.PopupTheme != nil && s.PopupTheme.Font != nil {
		return s.PopupTheme.Font
	}
	if f := theme.DefaultTheme.BaseFont; f != nil {
		return f
	}
	return graphics.SystemFont
}

var _ meui.EventListener = (*SwipeActionRow)(nil)
var _ meui.HoverTarget = (*SwipeActionRow)(nil)
