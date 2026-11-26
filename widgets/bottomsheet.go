package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/theme"
)

// BottomSheetPresentation selects the two constructions M3 defines, because they differ in
// BEHAVIOUR and not only in paint.
type BottomSheetPresentation uint8

const (
	// BottomSheetModal has a scrim, blocks the content behind it, and dismisses on a tap
	// outside the panel or a downward drag. It is the phone's answer to a dialog.
	BottomSheetModal BottomSheetPresentation = iota
	// BottomSheetStandard is part of the layout: the content behind stays interactive and
	// there is nothing to dismiss by tapping away. It is the persistent form.
	BottomSheetStandard
)

// The published metrics, in density-independent pixels. They are constants rather than theme
// tokens because M3 states them as numbers, and a sheet whose width cap drifted per preset
// would no longer be the control the guidance describes.
const (
	// BottomSheetMaxWidth is the widest a sheet gets. Beyond it the sheet stops growing and
	// the margin does the work.
	BottomSheetMaxWidth = float32(640)
	// BottomSheetSideMargin is the minimum gap on each side once the width cap applies.
	BottomSheetSideMargin = float32(56)
	// BottomSheetTopMargin is how much of the screen a sheet must leave uncovered, so the
	// content behind it stays visible enough to explain what the sheet is about.
	BottomSheetTopMargin = float32(72)

	bottomSheetHandleW = float32(32)
	bottomSheetHandleH = float32(4)
	bottomSheetHandleY = float32(8)
)

// BottomSheet is a surface anchored to the bottom of the screen: the phone's dialog, menu
// and picker host. It is deliberately a SURFACE and not a dialog with fixed buttons — the
// app puts whatever it needs inside, and the widget owns the geometry, the drag handle and
// the dismissal.
//
// Two presentations, and the difference is behaviour, not looks: a MODAL sheet is shown
// through the window's modal layer (Window.ShowModal) with the window's scrim behind it and
// dismisses on a tap outside or a drag down; a STANDARD sheet is placed in the shell and
// stays.
//
// Its rect is the area it may use, not the panel: the panel is anchored to the BOTTOM of
// that rect and is `SheetHeight` tall, so the same widget serves a window-sized modal host
// and an in-flow strip that is already exactly the panel's size.
type BottomSheet struct {
	meui.BaseWidget
	*theme.PopupTheme

	presentation BottomSheetPresentation
	title        string
	content      meui.Widget
	height       float32

	window *meui.Window

	dragging bool
	pressY   float32
	dragY    float32

	// OnDismissed fires when the sheet goes away, however it went away.
	OnDismissed func()
}

// NewBottomSheet builds a modal sheet with no title and no content.
func NewBottomSheet() *BottomSheet {
	return &BottomSheet{
		BaseWidget: *meui.NewBaseWidget(),
		PopupTheme: theme.DefaultPopupTheme,
	}
}

// ── content ─────────────────────────────────────────────────────────

func (s *BottomSheet) SetTitle(title string) *BottomSheet {
	s.title = title
	s.MarkForRedraw()
	return s
}

func (s *BottomSheet) Title() string { return s.title }

// SetContent installs the widget the sheet hosts, which is laid out inside the panel below
// the title. A nil content is allowed: a sheet may be nothing but a title.
func (s *BottomSheet) SetContent(w meui.Widget) *BottomSheet {
	s.content = w
	if w != nil {
		w.SetParent(s)
	}
	s.MarkForRedraw()
	return s
}

func (s *BottomSheet) Content() meui.Widget { return s.content }

// SetPresentation selects the modal or the standard construction.
func (s *BottomSheet) SetPresentation(p BottomSheetPresentation) *BottomSheet {
	s.presentation = p
	s.MarkForRedraw()
	return s
}

func (s *BottomSheet) Presentation() BottomSheetPresentation { return s.presentation }

// SetModal is the boolean spelling, for QML and for an app that thinks in one flag.
func (s *BottomSheet) SetModal(on bool) *BottomSheet {
	if on {
		return s.SetPresentation(BottomSheetModal)
	}
	return s.SetPresentation(BottomSheetStandard)
}

func (s *BottomSheet) Modal() bool { return s.presentation == BottomSheetModal }

// SetSheetHeight sets the panel's height. Zero lets the sheet size itself: half of the area
// it was given, which is the common modal proportion and always leaves the top margin.
func (s *BottomSheet) SetSheetHeight(h float32) *BottomSheet {
	if h < 0 {
		h = 0
	}
	s.height = h
	s.MarkForRedraw()
	return s
}

func (s *BottomSheet) SheetHeight() float32 { return s.height }

// SetWindow is how a widget hosted in the window's modal layer learns its window, so that
// Dismiss can take the modal down. The window's own popup path calls it.
func (s *BottomSheet) SetWindow(w *meui.Window) { s.window = w }

// SetBoundProperty implements the QML binding hook.
func (s *BottomSheet) SetBoundProperty(name string, value interface{}) {
	switch name {
	case "title":
		if str, ok := value.(string); ok {
			s.SetTitle(str)
		}
	case "modal":
		if b, ok := value.(bool); ok {
			s.SetModal(b)
		}
	case "height":
		s.SetSheetHeight(toFloat32(value))
	}
}

// ── geometry ────────────────────────────────────────────────────────

// PanelRect is the sheet's panel within the area the widget was given: centred, capped at
// BottomSheetMaxWidth with a BottomSheetSideMargin gap each side, and anchored to the bottom.
// The drag offset is included, so a drag moves the panel as it is painted.
//
// The margin applies only once the CAP is in play: a compact window gets a full-width sheet
// (M3's own rule — the 56dp margin is what a large window uses to keep a capped sheet off its
// edges), so a phone does not get a sheet inset by 56 on each side.
func (s *BottomSheet) PanelRect(bounds graphics.Rect) graphics.Rect {
	w := bounds.Width
	if w > BottomSheetMaxWidth {
		w = BottomSheetMaxWidth
		if maxW := bounds.Width - 2*BottomSheetSideMargin; w > maxW {
			w = maxW
		}
	}
	if w < 0 {
		w = 0
	}
	h := s.PanelHeight(bounds)
	x := bounds.X + (bounds.Width-w)/2
	y := bounds.Y + bounds.Height - h + s.dragY
	return graphics.NewRect(x, y, w, h)
}

// PanelHeight is the panel's height: the requested one, else half the area, and never more
// than the area minus BottomSheetTopMargin.
func (s *BottomSheet) PanelHeight(bounds graphics.Rect) float32 {
	h := s.height
	if h <= 0 {
		h = bounds.Height / 2
	}
	if maxH := bounds.Height - BottomSheetTopMargin; h > maxH {
		h = maxH
	}
	if h < 0 {
		h = 0
	}
	return h
}

// HandleRect is the drag handle, centred at the top of the panel.
func (s *BottomSheet) HandleRect(bounds graphics.Rect) graphics.Rect {
	p := s.PanelRect(bounds)
	return graphics.NewRect(
		p.X+(p.Width-bottomSheetHandleW)/2,
		p.Y+bottomSheetHandleY,
		bottomSheetHandleW, bottomSheetHandleH)
}

// TitleRect is where the title is drawn: below the handle, inset from the panel's edges.
func (s *BottomSheet) TitleRect(bounds graphics.Rect) graphics.Rect {
	p := s.PanelRect(bounds)
	top := p.Y + bottomSheetHandleY + bottomSheetHandleH + 12
	h := s.font().LineHeight()
	if h <= 0 {
		h = 20
	}
	return graphics.NewRect(p.X+24, top, p.Width-48, h)
}

// ContentRect is what is left of the panel for the content, below the title.
func (s *BottomSheet) ContentRect(bounds graphics.Rect) graphics.Rect {
	p := s.PanelRect(bounds)
	top := s.TitleRect(bounds)
	top2 := top.Y + top.Height + 8
	if s.title == "" {
		top2 = p.Y + bottomSheetHandleY + bottomSheetHandleH + 12
	}
	h := p.Y + p.Height - top2 - 8
	if h < 0 {
		h = 0
	}
	return graphics.NewRect(p.X+24, top2, p.Width-48, h)
}

// PreferredDimensions is how the layout learns a STANDARD sheet's height. A modal one is
// hosted by the window, which sets the frame itself.
func (s *BottomSheet) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	return w, s.PanelHeight(graphics.NewRect(0, 0, w, h))
}

func (s *BottomSheet) PreferredSize() (float32, float32) {
	return BottomSheetMaxWidth, s.height
}

// ── interaction ─────────────────────────────────────────────────────

// Dismiss takes the sheet down. A MODAL sheet also takes its modal layer down, which is why
// the widget keeps the window it was hosted by; a standard one is the app's to hide.
func (s *BottomSheet) Dismiss() {
	if !s.Visible() {
		return
	}
	s.SetVisible(false)
	s.dragging = false
	s.dragY = 0
	if s.modal() && s.window != nil {
		s.window.DismissTopModal()
	}
	s.MarkForRedraw()
	if s.OnDismissed != nil {
		s.OnDismissed()
	}
}

func (s *BottomSheet) modal() bool { return s.presentation == BottomSheetModal }

// DismissThreshold is how far the panel must be dragged down before releasing dismisses it:
// a third of its own height, so a small accidental drag springs back instead of losing the
// sheet. It is a fraction of the SHEET, not of the screen, because a short sheet would
// otherwise be impossible to dismiss by dragging.
func (s *BottomSheet) DismissThreshold(bounds graphics.Rect) float32 {
	return s.PanelHeight(bounds) / 3
}

// localBounds is the widget's own rect in the LOCAL space the event methods are handed; FrameRect is
// parent-relative, and a sheet is anchored to the window's bottom edge, so it is never at the origin.
// See ListItem.localBounds.
func (s *BottomSheet) localBounds() graphics.Rect {
	f := s.FrameRect()
	return graphics.NewRect(0, 0, f.Width, f.Height)
}

func (s *BottomSheet) OnMouseDown(x, y float32, button int) bool {
	if !s.Enabled() || button != mouseButtonLeft {
		return false
	}
	bounds := s.localBounds()
	if s.HandleRect(bounds).Contains(x, y) {
		s.dragging = true
		s.pressY = y
		s.dragY = 0
		s.MarkForRedraw()
		return true
	}
	// A tap ABOVE the panel dismisses a modal sheet — that is the scrim's job, and the
	// widget does it itself so a host does not have to know which layer it is in.
	if s.modal() && y < s.PanelRect(bounds).Y {
		s.Dismiss()
		return true
	}
	return false
}

func (s *BottomSheet) OnMouseMove(x, y float32) {
	if !s.dragging {
		return
	}
	// Downward only: dragging UP should not lift the panel off its edge and reveal a gap.
	d := y - s.pressY
	if d < 0 {
		d = 0
	}
	if d != s.dragY {
		s.dragY = d
		s.MarkForRedraw()
	}
}

func (s *BottomSheet) OnMouseUp(x, y float32, button int) bool {
	if !s.dragging {
		return false
	}
	s.dragging = false
	if s.dragY >= s.DismissThreshold(s.localBounds()) {
		s.Dismiss()
		return true
	}
	s.dragY = 0
	s.MarkForRedraw()
	return true
}

// ── paint ───────────────────────────────────────────────────────────

func (s *BottomSheet) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !s.Visible() || rect.Width < 1 || rect.Height < 1 {
		return
	}
	p := s.PanelRect(rect)
	if p.Width < 1 || p.Height < 1 {
		return
	}

	bg := graphics.NewPaint()
	bg.SetColor(s.BackgroundInk)

	radius := theme.DefaultTheme.Metrics.Shape.Popup
	if radius <= 0 {
		radius = 16
	}
	// Rounded TOP corners only: the panel is anchored to the screen's edge, and a rounded
	// bottom would show the content behind through the gap. The plate is drawn taller than
	// the panel and clipped to it, which leaves the bottom corners square.
	canvas.Save()
	canvas.ClipRect(p, pathop.Intersect, false)
	plate := graphics.NewRect(p.X, p.Y, p.Width, p.Height+radius)
	canvas.DrawRoundedRect(plate, radius, radius, bg)
	canvas.Restore()

	if s.EdgeInk != 0 {
		edge := graphics.NewPaint()
		edge.SetColor(s.EdgeInk)
		canvas.DrawRect(graphics.NewRect(p.X, p.Y, p.Width, 1), edge)
	}

	// The drag handle: the affordance that says "this moves".
	handle := s.HandleRect(rect)
	hp := graphics.NewPaint()
	hp.SetColor(s.DividerInk)
	if s.DividerInk == 0 {
		hp.SetColor(s.HeadingTextInk)
	}
	canvas.DrawRoundedRect(handle, bottomSheetHandleH/2, bottomSheetHandleH/2, hp)

	if s.title != "" {
		tr := s.TitleRect(rect)
		tp := graphics.NewPaint()
		tp.SetColor(s.HeadingTextInk)
		t := graphics.NewText(s.title, &graphics.TextDecoration{Font: s.font(), OnBackgroundInk: s.HeadingTextInk})
		canvas.Save()
		canvas.ClipRect(tr, pathop.Intersect, false)
		t.Draw(canvas, tr.X, tr.Y+s.font().Baseline())
		canvas.Restore()
	}

	if s.content != nil {
		cr := s.ContentRect(rect)
		if cr.Width >= 1 && cr.Height >= 1 {
			if p, ok := s.content.(interface{ SetFrameRect(graphics.Rect) }); ok {
				p.SetFrameRect(cr)
			}
			s.content.Paint(canvas, cr)
		}
	}
}

func (s *BottomSheet) font() graphics.Font {
	if s.PopupTheme != nil && s.PopupTheme.Font != nil {
		return s.PopupTheme.Font
	}
	if f := theme.DefaultTheme.BaseFont; f != nil {
		return f
	}
	return graphics.SystemFont
}

var _ meui.EventListener = (*BottomSheet)(nil)
