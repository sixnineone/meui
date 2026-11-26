package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/theme"
)

// ActionSheetItem is one choice in an ActionSheet.
type ActionSheetItem struct {
	// ID is what OnChoose reports.
	ID string
	// Label is the choice's text.
	Label string
	// Destructive draws it in the error ink: the choice that deletes something. It is a
	// COLOUR and not a confirmation dialog — an action sheet is where the platform puts
	// "Delete", and the confirmation belongs after it, not instead of it.
	Destructive bool
	// Disabled draws it dimmed and refuses to choose it.
	Disabled bool
}

// ActionSheet is a choice presented from the bottom of the screen: iOS's action sheet, and the
// phone's answer to "pick one of these". It IS a BottomSheet — it inherits the drag handle, the
// width cap, the modal hosting and every dismissal path — with a list of actions and a Cancel
// below them.
//
// Two rules from the guidance are structural rather than advisory:
//
//   - **It does not scroll.** The sheet sizes itself to its content, so an action sheet that
//     would need to scroll is one with too many choices — the guidance caps it at four buttons
//     including Cancel, and a scrolling action sheet is a menu wearing its clothes.
//   - **Cancel is a separate block**, below a gap, with its own rounded corners, because the way
//     out must not look like one of the choices.
type ActionSheet struct {
	*BottomSheet

	header  string
	message string
	items   []ActionSheetItem
	cancel  string

	pressed int
	hovered int

	// OnChoose fires with the chosen item's ID, AFTER the sheet is dismissed, so the callback is
	// free to open something else.
	OnChoose func(id string)
}

// The action sheet's own metrics. The rows reach the touch floor through rowHeight.
const (
	actionSheetPad = float32(8)
	actionSheetGap = float32(8)
	// ActionSheetMaxItems is the guidance's cap. It is enforced by the AUDIT rather than by
	// refusing items — an app with five choices is better served by a warning in review than by a
	// sheet that silently drops the fifth.
	ActionSheetMaxItems = 4
)

// NewActionSheet builds an empty modal action sheet.
func NewActionSheet() *ActionSheet {
	a := &ActionSheet{
		BottomSheet: NewBottomSheet(),
		pressed:     -1,
		hovered:     -1,
	}
	a.SetModal(true)
	return a
}

// ── content ─────────────────────────────────────────────────────────

// SetHeader sets an optional title and message above the actions. Either may be empty.
func (a *ActionSheet) SetHeader(title, message string) *ActionSheet {
	a.header, a.message = title, message
	a.refreshHeight()
	return a
}

func (a *ActionSheet) SetItems(items []ActionSheetItem) *ActionSheet {
	a.items = items
	a.pressed, a.hovered = -1, -1
	a.refreshHeight()
	return a
}

func (a *ActionSheet) Items() []ActionSheetItem { return a.items }

// SetCancel sets the cancel label. An empty label removes the Cancel block, which is what a
// sheet of pure choices with no way out wants.
func (a *ActionSheet) SetCancel(label string) *ActionSheet {
	a.cancel = label
	a.refreshHeight()
	return a
}

func (a *ActionSheet) CancelLabel() string { return a.cancel }

// SetBoundProperty implements the QML binding hook.
func (a *ActionSheet) SetBoundProperty(name string, value interface{}) {
	switch name {
	case "title":
		if str, ok := value.(string); ok {
			a.SetHeader(str, a.message)
		}
	case "text":
		if str, ok := value.(string); ok {
			a.SetHeader(a.header, str)
		}
	case "cancel":
		if str, ok := value.(string); ok {
			a.SetCancel(str)
		}
	}
}

// ── geometry ────────────────────────────────────────────────────────

// RowHeight is one action's height: at least the touch floor, because every row is a target.
func (a *ActionSheet) RowHeight() float32 {
	h := theme.DefaultTheme.Metrics.ControlHeight
	if floor := theme.DefaultTheme.Metrics.MinTouchTarget; h < floor {
		h = floor
	}
	if h <= 0 {
		h = 48
	}
	return h
}

// HeaderHeight is the title/message block's height, zero when there is neither.
func (a *ActionSheet) HeaderHeight() float32 {
	if a.header == "" && a.message == "" {
		return 0
	}
	line := a.font().LineHeight()
	if line <= 0 {
		line = 18
	}
	h := float32(0)
	if a.header != "" {
		h += line + 4
	}
	if a.message != "" {
		h += line + 4
	}
	return h + 8
}

// ContentHeight is what the sheet has to be to show everything: the padding, the header, one row
// per action and the Cancel block. The sheet sizes to it, which is what keeps the thing from
// needing to scroll.
func (a *ActionSheet) ContentHeight() float32 {
	h := actionSheetPad + a.HeaderHeight()
	h += float32(len(a.items)) * a.RowHeight()
	if a.cancel != "" {
		h += actionSheetGap + a.RowHeight()
	}
	return h + actionSheetPad
}

// refreshHeight hands the content height to the underlying sheet, which owns the cap and the top
// margin. It runs on every content change, so a caller cannot forget it.
func (a *ActionSheet) refreshHeight() {
	a.SetSheetHeight(a.ContentHeight())
	a.MarkForRedraw()
}

// ItemRect is one action's rect, in the sheet's own coordinates.
func (a *ActionSheet) ItemRect(i int, bounds graphics.Rect) (graphics.Rect, bool) {
	if i < 0 || i >= len(a.items) {
		return graphics.Rect{}, false
	}
	p := a.PanelRect(bounds)
	y := p.Y + actionSheetPad + a.HeaderHeight() + float32(i)*a.RowHeight()
	return graphics.NewRect(p.X+actionSheetPad, y, p.Width-2*actionSheetPad, a.RowHeight()), true
}

// CancelRect is the Cancel block: pinned to the bottom of the panel, so it is the last thing
// under the thumb and does not move when the number of choices changes above it.
func (a *ActionSheet) CancelRect(bounds graphics.Rect) (graphics.Rect, bool) {
	if a.cancel == "" {
		return graphics.Rect{}, false
	}
	p := a.PanelRect(bounds)
	y := p.Y + p.Height - actionSheetPad - a.RowHeight()
	return graphics.NewRect(p.X+actionSheetPad, y, p.Width-2*actionSheetPad, a.RowHeight()), true
}

// localBounds is the widget's own rect in the LOCAL space the event methods are handed; FrameRect is
// parent-relative, and this sheet is shown at the window's bottom edge, so it is never at the origin.
// See ListItem.localBounds.
func (a *ActionSheet) localBounds() graphics.Rect {
	f := a.FrameRect()
	return graphics.NewRect(0, 0, f.Width, f.Height)
}

// ItemAt reports which action holds a point, or -1. It does NOT include Cancel: the two are
// different things and the callers treat them differently.
func (a *ActionSheet) ItemAt(x, y float32) int {
	bounds := a.localBounds()
	for i := range a.items {
		if r, ok := a.ItemRect(i, bounds); ok && r.Contains(x, y) {
			return i
		}
	}
	return -1
}

func (a *ActionSheet) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	return w, a.ContentHeight()
}

// ── interaction ─────────────────────────────────────────────────────

func (a *ActionSheet) OnMouseDown(x, y float32, button int) bool {
	if !a.Enabled() || button != mouseButtonLeft {
		return false
	}
	if i := a.ItemAt(x, y); i >= 0 && !a.items[i].Disabled {
		a.pressed = i
		a.MarkForRedraw()
		return true
	}
	if r, ok := a.CancelRect(a.localBounds()); ok && r.Contains(x, y) {
		// Cancel is a sentinel index, one past the actions, so a single `pressed` field can
		// track both without a second flag.
		a.pressed = len(a.items)
		a.MarkForRedraw()
		return true
	}
	// The drag handle and the tap outside are the SHEET's: it owns the dismissal paths, and
	// duplicating them here is how the two would drift.
	return a.BottomSheet.OnMouseDown(x, y, button)
}

func (a *ActionSheet) OnMouseUp(x, y float32, button int) bool {
	if button != mouseButtonLeft {
		return false
	}
	was := a.pressed
	a.pressed = -1
	if was < 0 {
		return a.BottomSheet.OnMouseUp(x, y, button)
	}
	if was == len(a.items) { // Cancel
		if r, ok := a.CancelRect(a.localBounds()); ok && r.Contains(x, y) {
			a.Dismiss()
			return true
		}
		a.MarkForRedraw()
		return true
	}
	// A choice commits on RELEASE and only when the finger is still on the same row, so a
	// mis-touch can be recovered by sliding off it.
	if i := a.ItemAt(x, y); i == was && !a.items[i].Disabled {
		id := a.items[i].ID
		do := a.OnChoose
		a.Dismiss()
		if do != nil {
			do(id)
		}
		return true
	}
	a.MarkForRedraw()
	return true
}

func (a *ActionSheet) OnMouseMove(x, y float32) {
	i := a.ItemAt(x, y)
	if r, ok := a.CancelRect(a.localBounds()); ok && r.Contains(x, y) {
		i = len(a.items)
	}
	if i != a.hovered {
		a.hovered = i
		a.MarkForRedraw()
	}
}

// HoverAt implements HoverTarget, so the sheet is told when the pointer leaves it.
func (a *ActionSheet) HoverAt(x, y float32) { a.OnMouseMove(x, y) }

// ── paint ───────────────────────────────────────────────────────────

func (a *ActionSheet) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !a.Visible() || rect.Width < 1 || rect.Height < 1 {
		return
	}
	// The panel, its top edge and the drag handle are the sheet's, and drawing them here again
	// is how the two would drift.
	a.BottomSheet.Paint(canvas, rect)

	p := a.PanelRect(rect)
	if p.Width < 1 || p.Height < 1 {
		return
	}
	font := a.font()
	line := font.LineHeight()
	if line <= 0 {
		line = 18
	}

	// The header: a title, then a message. Both are CENTRED, which is the action sheet's own
	// convention — the header is a caption for the choices below it, not a sheet title.
	y := p.Y + actionSheetPad
	if a.header != "" {
		a.paintLine(canvas, a.header, p.X+actionSheetPad, y, p.Width-2*actionSheetPad, font, a.HeadingTextInk)
		y += line + 4
	}
	if a.message != "" {
		a.paintLine(canvas, a.message, p.X+actionSheetPad, y, p.Width-2*actionSheetPad, font, theme.DefaultTheme.TextSecondary)
	}

	for i, it := range a.items {
		r, ok := a.ItemRect(i, rect)
		if !ok {
			continue
		}
		ink := a.HeadingTextInk
		switch {
		case it.Disabled:
			ink = theme.DefaultTheme.DisabledText
		case it.Destructive:
			ink = theme.DefaultTheme.Error
		}
		a.paintRow(canvas, r, it.Label, font, ink, a.pressed == i, a.hovered == i)
	}

	if r, ok := a.CancelRect(rect); ok {
		a.paintRow(canvas, r, a.cancel, font, a.HeadingTextInk, a.pressed == len(a.items), a.hovered == len(a.items))
	}
}

// paintRow draws one action: a fill when it is hovered or pressed, then a centred label.
func (a *ActionSheet) paintRow(canvas *graphics.Canvas, r graphics.Rect, label string, font graphics.Font, ink graphics.Color, pressed, hovered bool) {
	if pressed || hovered {
		fill := theme.DefaultTheme.Hovered
		if pressed {
			fill = theme.DefaultTheme.Pressed
		}
		if fill != 0 {
			p := graphics.NewPaint()
			p.SetColor(fill)
			radius := theme.DefaultTheme.Metrics.Shape.Popup
			if radius > r.Height/2 {
				radius = r.Height / 2
			}
			canvas.DrawRoundedRect(r, radius, radius, p)
		}
	}
	a.paintLine(canvas, label, r.X, r.Y, r.Width, font, ink)
}

// paintLine draws one centred, clipped line of text. Everything in an action sheet is centred —
// the header as much as the choices — because the whole surface is a caption plus a list of
// equally-weighted answers, not a titled panel.
func (a *ActionSheet) paintLine(canvas *graphics.Canvas, text string, x, y, w float32, font graphics.Font, ink graphics.Color) {
	if text == "" {
		return
	}
	p := graphics.NewPaint()
	p.SetColor(ink)
	t := graphics.NewText(text, &graphics.TextDecoration{Font: font, OnBackgroundInk: ink})
	line := font.LineHeight()
	if line <= 0 {
		line = 18
	}
	clip := graphics.NewRect(x, y, w, line)
	canvas.Save()
	canvas.ClipRect(clip, pathop.Intersect, false)
	t.Draw(canvas, x+(w-t.Width())/2, y+font.Baseline())
	canvas.Restore()
}

var _ meui.EventListener = (*ActionSheet)(nil)
var _ meui.HoverTarget = (*ActionSheet)(nil)
