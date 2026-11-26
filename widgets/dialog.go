package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

// DialogPresentation selects WHERE a dialog appears, which is the one thing that changes between a
// tablet and a phone. It is a behaviour axis, so it follows the SURFACE's size class (FormFactor)
// rather than the OS name — a narrow desktop window is a phone by the same rule.
type DialogPresentation uint8

const (
	// DialogAuto is the M3 rule and the reason both forms exist: a CENTRED dialog on a surface
	// with room, and a BOTTOM SHEET on a phone, where a centred box would be under the keyboard
	// and out of thumb reach.
	DialogAuto DialogPresentation = iota
	// DialogCentred forces the centred form.
	DialogCentred
	// DialogSheet forces the bottom-sheet form.
	DialogSheet
)

// DialogAction is one button in a Dialog.
type DialogAction struct {
	// ID is what OnAction reports.
	ID string
	// Label is the button's text.
	Label string
	// Primary draws it in the accent ink: the confirming action. M3 puts it last, on the right.
	Primary bool
	// Destructive draws it in the error ink.
	Destructive bool
	Disabled    bool
}

// AlertDialog is the phone-and-tablet alert: a title, a message, and one to three actions. Its ONE
// variable is the presentation — centred on a surface with room, a bottom sheet on a phone — so an
// app writes one dialog and gets the platform's own placement.
//
// It is not a BottomSheet with a flag. The two share the metrics (the 640 cap, the 72 top margin)
// and the dismissal contract, but the centred form has its own geometry and rounds all four
// corners, and pretending otherwise would mean a widget that is a sheet everywhere and looks
// centred only by accident.
type AlertDialog struct {
	meui.BaseWidget
	*theme.PopupTheme

	title        string
	message      string
	actions      []DialogAction
	presentation DialogPresentation
	content      meui.Widget
	window       *meui.Window

	pressed int
	hovered int

	// body measures and paints the wrapped message. It is a Label rather than a second wrap
	// implementation: AGENTS.md's rule is that text geometry has ONE implementation, and this is
	// how the dialog borrows it.
	body *Label

	// OnAction fires with the action's ID, AFTER the dialog is dismissed, so the callback is free
	// to open something else.
	OnAction func(id string)
	// OnDismissed fires when the dialog goes away, however it went away.
	OnDismissed func()
}

// The dialog's metrics. DialogMaxWidth is M3's own cap for a dialog; the rest are the spacing the
// guidance describes, in density-independent pixels.
const (
	DialogMaxWidth   = float32(560)
	DialogSideMargin = float32(24)
	DialogTopMargin  = float32(48)
	DialogPad        = float32(24)
	DialogActionH    = float32(40)
	DialogActionPadX = float32(12)
	DialogActionGap  = float32(8)
)

// NewDialog builds an empty auto-presented dialog.
func NewAlertDialog() *AlertDialog {
	return &AlertDialog{
		BaseWidget: *meui.NewBaseWidget(),
		PopupTheme: theme.DefaultPopupTheme,
		pressed:    -1,
		hovered:    -1,
	}
}

// ── content ─────────────────────────────────────────────────────────

func (d *AlertDialog) SetTitle(title string) *AlertDialog {
	d.title = title
	d.MarkForRedraw()
	return d
}

func (d *AlertDialog) Title() string { return d.title }

// SetMessage sets the body text, which wraps to the panel's width.
func (d *AlertDialog) SetMessage(message string) *AlertDialog {
	d.message = message
	d.MarkForRedraw()
	return d
}

func (d *AlertDialog) Message() string { return d.message }

// SetContent installs a widget in place of the message, for a dialog that is not just words.
func (d *AlertDialog) SetContent(w meui.Widget) *AlertDialog {
	d.content = w
	if w != nil {
		w.SetParent(d)
	}
	d.MarkForRedraw()
	return d
}

func (d *AlertDialog) SetActions(actions []DialogAction) *AlertDialog {
	d.actions = actions
	d.pressed, d.hovered = -1, -1
	d.MarkForRedraw()
	return d
}

func (d *AlertDialog) Actions() []DialogAction { return d.actions }

func (d *AlertDialog) SetPresentation(p DialogPresentation) *AlertDialog {
	d.presentation = p
	d.MarkForRedraw()
	return d
}

func (d *AlertDialog) Presentation() DialogPresentation { return d.presentation }

// SetWindow is how a widget hosted in the window's modal layer learns its window.
func (d *AlertDialog) SetWindow(w *meui.Window) { d.window = w }

// SetBoundProperty implements the QML binding hook.
func (d *AlertDialog) SetBoundProperty(name string, value interface{}) {
	switch name {
	case "title":
		if s, ok := value.(string); ok {
			d.SetTitle(s)
		}
	case "text", "message":
		if s, ok := value.(string); ok {
			d.SetMessage(s)
		}
	}
}

// ── geometry ────────────────────────────────────────────────────────

// Sheet reports whether this dialog takes the bottom-sheet form ON THIS SURFACE. Auto resolves it
// from the size class, so resizing the window changes the presentation rather than needing the app
// to know the platform.
func (d *AlertDialog) Sheet(bounds graphics.Rect) bool {
	switch d.presentation {
	case DialogCentred:
		return false
	case DialogSheet:
		return true
	default:
		return platform.FormFactorFor(bounds.Width, bounds.Height, 1) == platform.FormFactorPhone
	}
}

// PanelRect is the dialog's box within the area it was given: centred, or anchored to the bottom as
// a sheet. It is capped, and it never grows past the top margin, so the content behind it stays
// visible enough to explain what it is about.
func (d *AlertDialog) PanelRect(bounds graphics.Rect) graphics.Rect {
	if d.Sheet(bounds) {
		w := bounds.Width
		if w > BottomSheetMaxWidth {
			w = BottomSheetMaxWidth
			if m := bounds.Width - 2*BottomSheetSideMargin; w > m {
				w = m
			}
		}
		h := d.contentHeight(w)
		if maxH := bounds.Height - BottomSheetTopMargin; h > maxH {
			h = maxH
		}
		return graphics.NewRect(bounds.X+(bounds.Width-w)/2, bounds.Y+bounds.Height-h, w, h)
	}
	w := bounds.Width - 2*DialogSideMargin
	if w > DialogMaxWidth {
		w = DialogMaxWidth
	}
	if w < 0 {
		w = 0
	}
	h := d.contentHeight(w)
	if maxH := bounds.Height - 2*DialogTopMargin; h > maxH {
		h = maxH
	}
	return graphics.NewRect(bounds.X+(bounds.Width-w)/2, bounds.Y+(bounds.Height-h)/2, w, h)
}

// contentHeight is what the box has to be to hold its parts, at the given outer width.
func (d *AlertDialog) contentHeight(w float32) float32 {
	inner := w - 2*DialogPad
	if inner < 0 {
		inner = 0
	}
	h := DialogPad
	if d.title != "" {
		h += d.lineHeight() + 8
	}
	if d.message != "" {
		h += d.MessageHeight(inner) + 8
	}
	if d.content != nil {
		if m, ok := d.content.(interface {
			PreferredDimensions(interface{}, float32, float32) (float32, float32)
		}); ok {
			if _, ch := m.PreferredDimensions(d.content, inner, 0); ch > 0 {
				h += ch + 8
			}
		}
	}
	if len(d.actions) > 0 {
		h += d.actionHeight()
	}
	return h + DialogPad
}

// MessageHeight is the wrapped message's height at a given inner width. It asks the Label's OWN
// wrapText, which is the one wrap implementation (AGENTS.md) — Label.PreferredDimensions cannot
// answer it, because it measures a single unwrapped line.
func (d *AlertDialog) MessageHeight(inner float32) float32 {
	if d.message == "" {
		return 0
	}
	b := d.messageLabel()
	b.SetText(d.message)
	lines := b.wrapText(d.font(), inner)
	if len(lines) == 0 {
		return d.lineHeight()
	}
	return float32(len(lines)) * d.lineHeight()
}

func (d *AlertDialog) messageLabel() *Label {
	if d.body == nil {
		d.body = NewLabel()
		d.body.SetWrapMode(WrapWordWrap)
		d.body.SetVAlign("top")
	}
	return d.body
}

func (d *AlertDialog) lineHeight() float32 {
	if h := d.font().LineHeight(); h > 0 {
		return h
	}
	return 18
}

// actionHeight is one action's hit height: M3 draws a text button 40 tall, and the Phase A floor
// says the TARGET must still reach 48, so the hit rect is the larger of the two.
func (d *AlertDialog) actionHeight() float32 {
	h := DialogActionH
	if floor := theme.DefaultTheme.Metrics.MinTouchTarget; h < floor {
		h = floor
	}
	return h
}

// ActionRect is the i-th action's rect. The actions are laid out from the RIGHT, LAST first,
// because M3 puts the confirming action last — and "last" means rightmost.
func (d *AlertDialog) ActionRect(i int, bounds graphics.Rect) (graphics.Rect, bool) {
	if i < 0 || i >= len(d.actions) {
		return graphics.Rect{}, false
	}
	p := d.PanelRect(bounds)
	h := d.actionHeight()
	y := p.Y + p.Height - DialogPad - h
	right := p.X + p.Width - DialogPad
	for j := len(d.actions) - 1; j >= 0; j-- {
		w := d.actionWidth(d.actions[j])
		x := right - w
		if j == i {
			return graphics.NewRect(x, y, w, h), true
		}
		right = x - DialogActionGap
	}
	return graphics.Rect{}, false
}

// actionWidth is a text button's width: its label plus padding, never under the touch floor.
func (d *AlertDialog) actionWidth(a DialogAction) float32 {
	font := d.font()
	t := graphics.NewText(a.Label, &graphics.TextDecoration{Font: font, OnBackgroundInk: d.HeadingTextInk})
	w := t.Width() + 2*DialogActionPadX
	if floor := theme.DefaultTheme.Metrics.MinTouchTarget; w < floor {
		w = floor
	}
	return w
}

// localBounds is the widget's own rect in the LOCAL space the event methods are handed; FrameRect is
// parent-relative. This dialog is shown CENTRED, so its frame is never at the origin — the reason a
// hit test against the frame is off by the layer's position, and a click lands on the row below.
func (d *AlertDialog) localBounds() graphics.Rect {
	f := d.FrameRect()
	return graphics.NewRect(0, 0, f.Width, f.Height)
}

// ActionAt reports which action holds a point, or -1.
func (d *AlertDialog) ActionAt(x, y float32) int {
	bounds := d.localBounds()
	for i := range d.actions {
		if r, ok := d.ActionRect(i, bounds); ok && r.Contains(x, y) {
			return i
		}
	}
	return -1
}

func (d *AlertDialog) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	return w, d.contentHeight(w)
}

// ── interaction ─────────────────────────────────────────────────────

// Dismiss takes the dialog down, modal layer included.
func (d *AlertDialog) Dismiss() {
	if !d.Visible() {
		return
	}
	d.SetVisible(false)
	d.pressed, d.hovered = -1, -1
	if d.window != nil {
		d.window.DismissTopModal()
	}
	d.MarkForRedraw()
	if d.OnDismissed != nil {
		d.OnDismissed()
	}
}

func (d *AlertDialog) OnMouseDown(x, y float32, button int) bool {
	if !d.Enabled() || button != mouseButtonLeft {
		return false
	}
	if i := d.ActionAt(x, y); i >= 0 && !d.actions[i].Disabled {
		d.pressed = i
		d.MarkForRedraw()
		return true
	}
	// A press OUTSIDE the box dismisses: the dialog is modal, and the scrim is the way out.
	if !d.PanelRect(d.localBounds()).Contains(x, y) {
		d.Dismiss()
		return true
	}
	return false
}

func (d *AlertDialog) OnMouseUp(x, y float32, button int) bool {
	if button != mouseButtonLeft {
		return false
	}
	was := d.pressed
	d.pressed = -1
	if was < 0 {
		return false
	}
	// A choice commits on RELEASE and only on the same action, so a mis-touch can be recovered.
	if i := d.ActionAt(x, y); i == was && !d.actions[i].Disabled {
		id := d.actions[i].ID
		do := d.OnAction
		d.Dismiss()
		if do != nil {
			do(id)
		}
		return true
	}
	d.MarkForRedraw()
	return true
}

func (d *AlertDialog) OnMouseMove(x, y float32) {
	i := d.ActionAt(x, y)
	if i != d.hovered {
		d.hovered = i
		d.MarkForRedraw()
	}
}

// HoverAt implements HoverTarget.
func (d *AlertDialog) HoverAt(x, y float32) { d.OnMouseMove(x, y) }

// ── paint ───────────────────────────────────────────────────────────

func (d *AlertDialog) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !d.Visible() || rect.Width < 1 || rect.Height < 1 {
		return
	}
	p := d.PanelRect(rect)
	if p.Width < 1 || p.Height < 1 {
		return
	}
	radius := theme.DefaultTheme.Metrics.Shape.Popup
	if radius <= 0 {
		radius = 16
	}

	bg := graphics.NewPaint()
	bg.SetColor(d.BackgroundInk)
	if d.Sheet(rect) {
		// A sheet is anchored to the screen's edge, so only its TOP corners round; a rounded
		// bottom would show the content behind through the gap.
		canvas.Save()
		canvas.ClipRect(p, pathop.Intersect, false)
		canvas.DrawRoundedRect(graphics.NewRect(p.X, p.Y, p.Width, p.Height+radius), radius, radius, bg)
		canvas.Restore()
	} else {
		canvas.DrawRoundedRect(p, radius, radius, bg)
	}

	font := d.font()
	inner := p.Width - 2*DialogPad
	y := p.Y + DialogPad

	if d.title != "" {
		ink := d.HeadingTextInk
		tp := graphics.NewPaint()
		tp.SetColor(ink)
		t := graphics.NewText(d.title, &graphics.TextDecoration{Font: font, OnBackgroundInk: ink})
		clip := graphics.NewRect(p.X+DialogPad, y, inner, d.lineHeight())
		canvas.Save()
		canvas.ClipRect(clip, pathop.Intersect, false)
		t.Draw(canvas, p.X+DialogPad, y+font.Baseline())
		canvas.Restore()
		y += d.lineHeight() + 8
	}

	if d.message != "" {
		h := d.MessageHeight(inner)
		b := d.messageLabel()
		b.SetText(d.message)
		b.SetFrameRect(graphics.NewRect(p.X+DialogPad, y, inner, h))
		b.Paint(canvas, graphics.NewRect(p.X+DialogPad, y, inner, h))
		y += h + 8
	}

	if d.content != nil {
		if m, ok := d.content.(interface {
			PreferredDimensions(interface{}, float32, float32) (float32, float32)
		}); ok {
			if _, ch := m.PreferredDimensions(d.content, inner, 0); ch > 0 {
				cr := graphics.NewRect(p.X+DialogPad, y, inner, ch)
				d.content.SetFrameRect(cr)
				d.content.Paint(canvas, cr)
				y += ch + 8
			}
		}
	}

	for i, a := range d.actions {
		r, ok := d.ActionRect(i, rect)
		if !ok {
			continue
		}
		ink := d.HeadingTextInk
		switch {
		case a.Disabled:
			ink = theme.DefaultTheme.DisabledText
		case a.Destructive:
			ink = theme.DefaultTheme.Error
		case a.Primary:
			ink = theme.DefaultTheme.Accent
		}
		if (d.hovered == i || d.pressed == i) && theme.DefaultTheme.Hovered != 0 {
			hp := graphics.NewPaint()
			hp.SetColor(theme.DefaultTheme.Hovered)
			rr := r.Height / 2
			canvas.DrawRoundedRect(r, rr, rr, hp)
		}
		tp := graphics.NewPaint()
		tp.SetColor(ink)
		t := graphics.NewText(a.Label, &graphics.TextDecoration{Font: font, OnBackgroundInk: ink})
		canvas.Save()
		canvas.ClipRect(r, pathop.Intersect, false)
		t.Draw(canvas, r.X+(r.Width-t.Width())/2, r.Y+(r.Height+font.Baseline())/2-font.Baseline()/2)
		canvas.Restore()
	}
}

func (d *AlertDialog) font() graphics.Font {
	if d.PopupTheme != nil && d.PopupTheme.Font != nil {
		return d.PopupTheme.Font
	}
	if f := theme.DefaultTheme.BaseFont; f != nil {
		return f
	}
	return graphics.SystemFont
}

var _ meui.EventListener = (*AlertDialog)(nil)
var _ meui.HoverTarget = (*AlertDialog)(nil)
