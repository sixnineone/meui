package widgets

import (
	"strings"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/layout"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

const (
	popupTitleH   = 32
	popupBtnH     = 32
	popupBtnMinW  = 64
	popupPadding  = 8
	popupBtnSpace = 8
)

// ModalPopupPanel is a reusable modal popup widget with title bar,
// content area, and button row. Use with Window.ShowModal().
type ModalPopupPanel struct {
	meui.BaseContainer
	Title           string
	Heading         string // larger heading text shown below title bar
	Content         *meui.Panel
	buttons         []*Button
	dismissFn       func()
	Modal           *meui.ModalPopup // set by ShowModal caller
	window          *meui.Window     // set by ShowModal
	PopupTheme      *theme.PopupTheme
	Draggable       bool // enable title bar drag
	DismissOnEscape bool // press Escape to dismiss
	textCache       *graphics.Text
	textCacheVer    uint64
	hoveredChild    meui.Widget
	capturedChild   meui.Widget
	dragging        bool
	dragStartX      float32
	dragStartY      float32
	dragFrameX      float32
	dragFrameY      float32
}

func (p *ModalPopupPanel) SetWindow(win *meui.Window) {
	p.window = win
}

func (p *ModalPopupPanel) MarkForRedraw() {
	p.BaseWidget.MarkForRedraw()
	if p.window != nil {
		p.window.MarkForRedraw()
	}
}

// NewModalPopupPanel creates a new modal popup panel.
// title: window title bar text
// heading: main body heading text
// content: optional widget for the body area (can be nil)
// buttons: buttons to show in the bottom button row
func NewModalPopupPanel(title, heading string, content *meui.Panel, buttons ...*Button) *ModalPopupPanel {
	p := &ModalPopupPanel{
		BaseContainer: meui.BaseContainer{BaseWidget: *meui.NewBaseWidget()},
		Title:         title,
		Heading:       heading,
		buttons:       buttons,
		PopupTheme:    theme.DefaultPopupTheme,
	}
	if content != nil {
		p.Content = content
		content.SetParent(p)
	}
	for _, btn := range buttons {
		btn.SetParent(p)
	}
	return p
}

func (p *ModalPopupPanel) SetDraggable(d bool) *ModalPopupPanel {
	p.Draggable = d
	return p
}

func (p *ModalPopupPanel) SetDismissOnEscape(d bool) *ModalPopupPanel {
	p.DismissOnEscape = d
	return p
}

func (p *ModalPopupPanel) SetDismissFn(fn func()) {
	p.dismissFn = fn
}

func (p *ModalPopupPanel) Dismiss() {
	if p.dismissFn != nil {
		p.dismissFn()
	}
}

// OnClosed is the window's "your layer is gone" notification. A dialog dismissed from outside its own
// buttons (Escape, the backdrop, a host that closes the top layer) still has to run the app's dismiss
// function, or the app keeps a popup it no longer owns.
func (p *ModalPopupPanel) OnClosed() {
	p.Dismiss()
}

// Children exposes the dialog's content and its button row. A dialog's widgets are NOT in the window's
// child tree — the window hosts the panel as a layer — so without this the focus traversal (Tab) could
// never see anything inside a dialog.
func (p *ModalPopupPanel) Children() []meui.Widget {
	out := make([]meui.Widget, 0, len(p.buttons)+1)
	if p.Content != nil {
		out = append(out, p.Content)
	}
	for _, b := range p.buttons {
		out = append(out, b)
	}
	return out
}

func (p *ModalPopupPanel) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !p.Visible() {
		return
	}
	canvas.Save()

	t := p.PopupTheme
	if t == nil {
		t = theme.DefaultPopupTheme
	}
	font := t.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
	}
	if font == nil {
		font = graphics.SystemFont
	}

	// Translate to popup-local coordinate space
	canvas.Translate(rect.X, rect.Y)

	// Shadow — token-driven platform soft shadow; themes without one
	// (Metro) fall back to the classic offset plate.
	popupR := theme.Shape().Popup
	localRect := graphics.NewRect(0, 0, rect.Width, rect.Height)
	if theme.DefaultTheme.Effects.PopupShadow.Color != 0 {
		drawPopupShadow(canvas, localRect, popupR)
	} else {
		shadow := graphics.NewPaint()
		shadow.SetColor(t.ShadowInk)
		canvas.DrawRoundedRect(graphics.NewRect(4, 4, rect.Width, rect.Height), popupR, popupR, shadow)
	}

	// Background
	bg := graphics.NewPaint()
	bg.SetColor(t.BackgroundInk)
	canvas.DrawRoundedRect(localRect, popupR, popupR, bg)

	// Border
	border := graphics.NewPaint()
	border.SetStyle(paintstyle.Stroke)
	border.SetStrokeWidth(1)
	border.SetColor(t.EdgeInk)
	canvas.DrawRoundedRect(localRect, popupR, popupR, border)

	// Title bar background
	titleRect := graphics.NewRect(0, 0, rect.Width, popupTitleH)
	titleBg := graphics.NewPaint()
	titleBg.SetColor(t.TitleBgInk)
	canvas.DrawRoundedRect(titleRect, popupR, popupR, titleBg)
	// Fill bottom of title to remove rounding
	titleFill := graphics.NewRect(0, 4, rect.Width, popupTitleH-4)
	titleBg2 := graphics.NewPaint()
	titleBg2.SetColor(t.TitleBgInk)
	canvas.DrawRect(titleFill, titleBg2)

	// Title text
	titlePaint := graphics.NewPaint()
	titlePaint.SetColor(t.TitleTextInk)
	ascent := font.Baseline()
	ver := theme.ThemeVersion()
	if p.textCache == nil || p.textCacheVer != ver {
		p.textCache = graphics.NewText(p.Title, &graphics.TextDecoration{Font: font, OnBackgroundInk: titlePaint.Color()})
		p.textCacheVer = ver
	}
	p.textCache.Draw(canvas, 12, popupTitleH/2+ascent/2)

	// Divider line under title
	divider := graphics.NewPaint()
	divider.SetStyle(paintstyle.Stroke)
	divider.SetStrokeWidth(1)
	divider.SetColor(t.DividerInk)
	canvas.DrawLine(
		1, popupTitleH,
		rect.Width-1, popupTitleH,
		divider,
	)

	bodyH := rect.Height - popupTitleH - popupBtnH - popupPadding*3

	// Heading text (if set). The advance comes from headingAdvance(), the same value bodyY() uses,
	// so the content is painted exactly where input will find it.
	headingY := float32(popupTitleH + popupPadding)
	if p.Heading != "" {
		headingPaint := graphics.NewPaint()
		headingPaint.SetColor(t.HeadingTextInk)
		hf := p.headingFont()
		if hf == nil {
			hf = font
		}
		DrawText(canvas, p.Heading, popupPadding, headingY+hf.Baseline(), hf, headingPaint)
		headingY += p.headingAdvance()
	}

	// Content widget
	if p.Content != nil {
		cH := bodyH - (headingY - float32(popupTitleH+popupPadding))
		contentFrame := graphics.NewRect(popupPadding, headingY, rect.Width-popupPadding*2, cH)
		p.Content.SetFrameRect(contentFrame)
		p.Content.ValidateLayout()
		canvas.Save()
		canvas.Translate(contentFrame.X, contentFrame.Y)
		p.Content.Paint(canvas, graphics.NewRect(0, 0, contentFrame.Width, contentFrame.Height))
		canvas.Restore()
	}

	// Button row at bottom
	btnY := rect.Height - popupBtnH - popupPadding
	btnX := rect.Width - popupPadding
	for i := len(p.buttons) - 1; i >= 0; i-- {
		btn := p.buttons[i]
		btnFrame := btn.FrameRect()
		bw := btnFrame.Width
		if bw <= 0 {
			bw = popupBtnMinW
		}
		btnX -= bw
		btn.SetFrameRect(graphics.NewRect(btnX, btnY, bw, popupBtnH))
		canvas.Save()
		canvas.Translate(btnX, btnY)
		btn.Paint(canvas, graphics.NewRect(0, 0, bw, popupBtnH))
		canvas.Restore()
		btnX -= popupBtnSpace
	}

	canvas.Restore()
}

func (p *ModalPopupPanel) bodyY() float32 {
	return float32(popupTitleH+popupPadding) + p.headingAdvance()
}

// headingAdvance is the vertical space the heading takes, and it is the ONE number both the paint
// pass and the input offset use. They used to disagree: Paint advanced a hard-coded 14 while bodyY
// advanced the font's LineHeight, so as soon as a heading appeared (a validation message) the content
// was painted a few pixels away from where it could be clicked.
func (p *ModalPopupPanel) headingAdvance() float32 {
	if p.Heading == "" {
		return 0
	}
	font := p.headingFont()
	if font == nil {
		return 14 + popupPadding
	}
	return font.LineHeight() + popupPadding
}

// headingFont is the font the heading is drawn with.
func (p *ModalPopupPanel) headingFont() graphics.Font {
	if p.PopupTheme != nil && p.PopupTheme.Font != nil {
		return p.PopupTheme.Font
	}
	if theme.DefaultTheme.BaseFont != nil {
		return theme.DefaultTheme.BaseFont
	}
	return graphics.SystemFont
}

func (p *ModalPopupPanel) bodyH() float32 {
	frame := p.FrameRect()
	return frame.Height - p.bodyY() - popupBtnH - popupPadding*2
}

func (p *ModalPopupPanel) OnMouseDown(x, y float32, button int) bool {
	if !p.Visible() || !p.Enabled() {
		return false
	}
	if button != 0 {
		return false
	}
	frame := p.FrameRect()
	lx := x
	ly := y

	// Title bar drag
	if p.Draggable && ly <= popupTitleH {
		p.dragging = true
		p.dragStartX = lx
		p.dragStartY = ly
		p.dragFrameX = frame.X
		p.dragFrameY = frame.Y
		return true
	}

	// Check buttons
	var hitBtn *Button
	btnY := frame.Height - popupBtnH - popupPadding
	btnX := frame.Width - popupPadding
	for i := len(p.buttons) - 1; i >= 0; i-- {
		btn := p.buttons[i]
		bw := btn.FrameRect().Width
		if bw <= 0 {
			bw = popupBtnMinW
		}
		btnX -= bw
		bx := btnX
		btnX -= popupBtnSpace
		if lx >= bx && lx <= bx+bw && ly >= btnY && ly <= btnY+popupBtnH {
			hitBtn = btn
			break
		}
	}
	if hitBtn != nil {
		p.capturedChild = nil
		p.focusChild(hitBtn)
		return hitBtn.OnMouseDown(lx-hitBtn.FrameRect().X, ly-btnY, button)
	}

	// Forward to content widget if click is within content area
	if p.Content != nil {
		by := p.bodyY()
		bh := p.bodyH()
		if ly >= by && ly <= by+bh && lx >= popupPadding && lx <= frame.Width-popupPadding {
			clx := lx - popupPadding
			cly := ly - by
			p.Content.ValidateLayout()
			clicked := p.Content.PanelAt(clx, cly)
			p.capturedChild = clicked
			// FOCUS FIRST, then the press. A text field selects its contents on focus and places the
			// caret on the press, so the other order (press, then focus) wiped the caret the user just
			// placed and left the whole value selected. Focus goes to the nearest focusable ancestor,
			// through the window, so there is ONE focus owner.
			if clicked != nil {
				p.focusChild(meui.NearestFocusable(clicked))
			}
			p.Content.OnMouseDown(clx, cly, button)
			return true // always consume — inside popup bounds
		}
	}

	// Clicks outside content area and not on buttons — consume event (modal)
	p.capturedChild = nil
	return true
}

// focusChild moves the window's keyboard focus to a widget in this dialog.
func (p *ModalPopupPanel) focusChild(child meui.Widget) {
	if p.window == nil || child == nil {
		return
	}
	p.window.SetFocusedChild(child)
}

func (p *ModalPopupPanel) OnMouseWheel(x, y float32, dx, dy float32) bool {
	if p.Content != nil {
		frame := p.FrameRect()
		lx := x
		ly := y
		by := p.bodyY()
		bh := p.bodyH()
		if ly >= by && ly <= by+bh && lx >= popupPadding && lx <= frame.Width-popupPadding {
			p.Content.ValidateLayout()
			p.Content.OnMouseWheel(lx-popupPadding, ly-by, dx, dy)
			return true
		}
	}
	return false
}

func (p *ModalPopupPanel) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	// Escape dismisses. The window normally gets there first (it closes the top modal), but a host
	// that dispatches straight to a layer still works.
	if p.DismissOnEscape && keyCode == platform.KeyEscape {
		p.Dismiss()
		return true
	}
	// Everything else reaches the focused widget and this panel's ancestors through the WINDOW's
	// dispatch. This used to walk the whole content tree and hand the key to the first widget that
	// accepted it: Tab was inserted as a tab character by a text area, Left/Right flipped a tab panel,
	// arrows nudged a spin box — none of them focused.
	return false
}

// contentToChild converts coordinates from a parent's local space to a child's
// local space by walking the ancestor chain.
func contentToChild(x, y float32, child, parent meui.Widget) (float32, float32) {
	for cur := child; cur != nil && cur != parent; cur = cur.Parent() {
		fr := cur.FrameRect()
		x -= fr.X
		y -= fr.Y
	}
	return x, y
}

// movePopupTo repositions the popup and its layer.
func (p *ModalPopupPanel) movePopupTo(x, y float32) {
	p.SetFrameRect(graphics.NewRect(x, y, p.FrameRect().Width, p.FrameRect().Height))
	if p.Modal != nil && p.Modal.ContentLayer != nil {
		p.Modal.ContentLayer.SetFrameRect(p.FrameRect())
	}
	if p.window != nil {
		p.window.MarkForRedraw()
	}
}

func (p *ModalPopupPanel) OnMouseUp(x, y float32, button int) bool {
	p.dragging = false
	p.capturedChild = nil
	frame := p.FrameRect()
	lx := x
	ly := y

	// Forward to buttons
	btnY := frame.Height - popupBtnH - popupPadding
	btnX := frame.Width - popupPadding
	for i := len(p.buttons) - 1; i >= 0; i-- {
		btn := p.buttons[i]
		bw := btn.FrameRect().Width
		if bw <= 0 {
			bw = popupBtnMinW
		}
		btnX -= bw
		bx := btnX
		btnX -= popupBtnSpace
		if lx >= bx && lx <= bx+bw && ly >= btnY && ly <= btnY+popupBtnH {
			return btn.OnMouseUp(lx-bx, ly-btnY, button)
		}
	}

	// Forward to content
	if p.Content != nil {
		by := p.bodyY()
		bh := p.bodyH()
		if ly >= by && ly <= by+bh && lx >= popupPadding && lx <= frame.Width-popupPadding {
			return p.Content.OnMouseUp(lx-popupPadding, ly-by, button)
		}
	}
	return true
}

func (p *ModalPopupPanel) OnMouseMove(x, y float32) {
	frame := p.FrameRect()
	lx := x
	ly := y

	// Title bar dragging
	if p.dragging {
		dx := lx - p.dragStartX
		dy := ly - p.dragStartY
		p.movePopupTo(p.dragFrameX+dx, p.dragFrameY+dy)
		return
	}

	// Track hover state for the topmost widget under cursor
	var underCursor meui.Widget

	// Check buttons (iterate in visual order: right-to-left)
	btnY := frame.Height - popupBtnH - popupPadding
	btnX := frame.Width - popupPadding
	for i := len(p.buttons) - 1; i >= 0; i-- {
		btn := p.buttons[i]
		bw := btn.FrameRect().Width
		if bw <= 0 {
			bw = popupBtnMinW
		}
		btnX -= bw
		bx := btnX
		btnX -= popupBtnSpace
		if lx >= bx && lx <= bx+bw && ly >= btnY && ly <= btnY+popupBtnH {
			underCursor = btn
			btn.OnMouseMove(lx-bx, ly-btnY)
			break
		}
	}

	// Check content area
	if p.Content != nil {
		by := p.bodyY()
		bh := p.bodyH()
		if ly >= by && ly <= by+bh && lx >= popupPadding && lx <= frame.Width-popupPadding {
			clx := lx - popupPadding
			cly := ly - by

			// Normal dispatch to whatever child is under the cursor
			p.Content.ValidateLayout()
			p.Content.OnMouseMove(clx, cly)

			// During drag, also forward to the captured child even if cursor
			// left its FrameRect (needed for text selection dragging).
			if p.capturedChild != nil {
				cx, cy := contentToChild(clx, cly, p.capturedChild, p.Content)
				if el, ok := p.capturedChild.(interface{ OnMouseMove(float32, float32) }); ok {
					el.OnMouseMove(cx, cy)
				}
			}

			underCursor = p.Content.PanelAt(clx, cly)
		} else {
			// Cursor left content area — still forward to captured child during drag
			if p.capturedChild != nil && p.window != nil {
				cx, cy := contentToChild(lx, ly, p.capturedChild, p)
				if el, ok := p.capturedChild.(interface{ OnMouseMove(float32, float32) }); ok {
					el.OnMouseMove(cx, cy)
				}
			}
		}
	}

	// Fire hover state changes
	if underCursor != p.hoveredChild {
		if p.hoveredChild != nil {
			if hh, ok := p.hoveredChild.(meui.HoverHandler); ok {
				hh.SetHoveredState(false)
			}
		}
		p.hoveredChild = underCursor
		if underCursor != nil {
			if hh, ok := underCursor.(meui.HoverHandler); ok {
				hh.SetHoveredState(true)
			}
		}
	}
}

// ── Factory: MessageBox ──────────────────────────────────────────

// ShowMessageBox shows a simple message dialog with text and buttons.
// onDone receives the button index (0-based) when the user clicks.
func ShowMessageBox(win *meui.Window, title, message string, buttons []string, onDone func(int)) *ModalPopupPanel {
	font := theme.DefaultTheme.BaseFont
	if font == nil {
		font = graphics.SystemFont
	}

	lines := strings.Split(message, "\n")
	maxLineW := float32(260)
	for _, line := range lines {
		lw := font.SimpleWidth(line) + 40
		if lw > maxLineW {
			maxLineW = lw
		}
	}
	numBtns := len(buttons)
	totalBtnW := float32(numBtns)*popupBtnMinW + float32(numBtns-1)*popupBtnSpace
	if maxLineW < totalBtnW+40 {
		maxLineW = totalBtnW + 40
	}
	if maxLineW < 240 {
		maxLineW = 240
	}

	content := meui.NewPanel()
	content.SetLayout(&layout.VStack{Spacing: 6})
	content.SetPadding(4)

	msgLabel := NewLabel()
	msgLabel.SetText(message)
	content.AddChild(msgLabel)

	var btns []*Button
	for i, lbl := range buttons {
		idx := i
		btn := NewButton()
		btn.SetText(lbl)
		btns = append(btns, btn)
		_ = idx
	}

	pw := maxLineW
	ph := float32(len(lines))*font.LineHeight() + 100
	if ph < 140 {
		ph = 140
	}

	popup := NewModalPopupPanel(title, "", content, btns...)
	popup.DismissOnEscape = true
	popup.SetDismissFn(func() {
		if popup.Modal != nil {
			win.DismissModal(popup.Modal)
		}
		if onDone != nil {
			onDone(-1)
		}
	})
	// Wire button indices to onDone + dismiss
	for i, btn := range btns {
		idx := i
		prev := btn.OnClicked
		btn.OnClicked = func() {
			if prev != nil {
				prev()
			}
			if popup.Modal != nil {
				win.DismissModal(popup.Modal)
			}
			if onDone != nil {
				onDone(idx)
			}
		}
	}

	popup.Modal = win.ShowModal(popup, pw, ph)
	return popup
}

// ── Factory: InputDialog ─────────────────────────────────────────

// ShowInputDialog shows a dialog with a label and text field for user input.
// onDone receives (text, accepted) where accepted is true if OK was pressed.
func ShowInputDialog(win *meui.Window, title, label string, onDone func(string, bool)) *ModalPopupPanel {
	content := meui.NewPanel()
	content.SetLayout(&layout.VStack{Spacing: 8})
	content.SetPadding(6)

	lbl := NewLabel()
	lbl.SetText(label)
	content.AddChild(lbl)

	tf := NewTextField()
	tf.SetPlaceholder("Enter value")
	content.AddChild(tf)

	okBtn := NewButton()
	okBtn.SetText("OK")
	cancelBtn := NewButton()
	cancelBtn.SetText("Cancel")

	popup := NewModalPopupPanel(title, "", content, okBtn, cancelBtn)
	popup.DismissOnEscape = true
	popup.SetDismissFn(func() {
		if popup.Modal != nil {
			win.DismissModal(popup.Modal)
		}
		if onDone != nil {
			onDone("", false)
		}
	})

	okBtn.OnClicked = func() {
		if onDone != nil {
			onDone(tf.Text, true)
		}
		popup.Dismiss()
	}
	cancelBtn.OnClicked = func() {
		if onDone != nil {
			onDone(tf.Text, false)
		}
		popup.Dismiss()
	}

	popup.Modal = win.ShowModal(popup, 320, 180)
	return popup
}

// ── Compatibility: NewDialog wraps ModalPopupPanel as Dialog ─────

// NewDialog creates a draggable popup. Used by QML/flow wrappers.
func NewDialog() *ModalPopupPanel {
	p := NewModalPopupPanel("", "", nil)
	p.Draggable = true
	return p
}

// NewMessageBox creates a message box. Used by QML registry.
func NewMessageBox(title, message string, buttons []string) *ModalPopupPanel {
	content := meui.NewPanel()
	content.SetLayout(&layout.VStack{Spacing: 6})
	content.SetPadding(4)
	lbl := NewLabel()
	lbl.SetText(message)
	content.AddChild(lbl)
	var btns []*Button
	for _, lbl := range buttons {
		btn := NewButton()
		btn.SetText(lbl)
		btns = append(btns, btn)
	}
	p := NewModalPopupPanel(title, "", content, btns...)
	p.DismissOnEscape = true
	return p
}

// NewInputDialog creates an input dialog. Used by QML registry.
func NewInputDialog(title, label string) *ModalPopupPanel {
	content := meui.NewPanel()
	content.SetLayout(&layout.VStack{Spacing: 8})
	content.SetPadding(6)
	lbl := NewLabel()
	lbl.SetText(label)
	content.AddChild(lbl)
	tf := NewTextField()
	tf.SetPlaceholder("Enter value")
	content.AddChild(tf)
	okBtn := NewButton()
	okBtn.SetText("OK")
	cancelBtn := NewButton()
	cancelBtn.SetText("Cancel")
	p := NewModalPopupPanel(title, "", content, okBtn, cancelBtn)
	p.DismissOnEscape = true
	return p
}
