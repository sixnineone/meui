package widgets

import (
	"fmt"
	"os"
	"time"
	"unicode"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

type Button struct {
	meui.BaseWidget
	*theme.ButtonTheme
	Text           string
	Icon           string
	ButtonType     string // "square", "round", "" (default)
	Pressed        bool
	Checkable      bool // clicking toggles Toggled instead of acting like a momentary press
	Toggled        bool // current state when Checkable
	Shortcut       rune // optional activation key (letters/digits); fires OnClicked
	parentWin      *meui.Window
	OnClicked      func()
	OnRightClicked func()
	OnToggled      func(toggled bool)
	Selectable
	iconSVG      *meui.SVGDocument
	textCache    *graphics.Text
	textCacheVer uint64
	bgAnim       meui.Animation // hover/press background blend (Motion tokens)
}

func NewButton() *Button {
	b := &Button{
		BaseWidget: *meui.NewBaseWidget(),
		ButtonType: "",
	}
	b.SetTabIndex(0)
	b.ButtonTheme = theme.DefaultButtonTheme
	return b
}

func (b *Button) SetIcon(icon string) *Button {
	b.Icon = icon
	b.iconSVG = nil
	if icon != "" {
		if doc, err := meui.LoadSVGByName(icon); err == nil {
			b.iconSVG = doc
		}
	}
	b.MarkForRedraw()
	return b
}

func (b *Button) GetSelectable() *Selectable {
	return &b.Selectable
}

func (b *Button) SetText(text string) *Button {
	b.Text = text
	b.textCache = nil
	b.MarkForRedraw()
	return b
}

func (b *Button) SetOnClicked(handler func()) *Button {
	b.OnClicked = handler
	return b
}

func (b *Button) SetGroupID(id string) *Button {
	b.Selectable.SetGroupID(id, b)
	return b
}

func (b *Button) SetSelected(selected bool) *Button {
	b.Selectable.SetSelected(selected, b, nil)
	b.MarkForRedraw()
	return b
}

func (b *Button) SetTooltip(text string) *Button {
	b.BaseWidget.SetTooltip(text)
	return b
}

// SetCheckable makes the button a toggle: clicking flips its Toggled state
// (like QToolButton.setCheckable). The OnClicked handler still fires.
func (b *Button) SetCheckable(checkable bool) *Button {
	b.Checkable = checkable
	b.MarkForRedraw()
	return b
}

func (b *Button) IsCheckable() bool { return b.Checkable }

// SetToggled sets the checkable button's state without firing OnToggled.
func (b *Button) SetToggled(toggled bool) *Button {
	if b.Toggled == toggled {
		return b
	}
	b.Toggled = toggled
	b.MarkForRedraw()
	return b
}

func (b *Button) IsToggled() bool { return b.Toggled }

func (b *Button) SetOnToggled(handler func(toggled bool)) *Button {
	b.OnToggled = handler
	return b
}

// SetShortcut sets an activation key (letter or digit). When pressed the
// button fires OnClicked as if clicked.
func (b *Button) SetShortcut(key rune) *Button {
	b.Shortcut = key
	return b
}

// toggleToggled flips the checkable state and notifies.
func (b *Button) toggleToggled() {
	if !b.Checkable {
		return
	}
	b.Toggled = !b.Toggled
	b.MarkForRedraw()
	if b.OnToggled != nil {
		b.OnToggled(b.Toggled)
	}
}

func (b *Button) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !b.Visible() {
		return
	}

	canvas.Save()

	// ButtonType: "" = theme radius, "square" = 0, "round" = pill
	radius := b.ButtonTheme.CornerRadius
	switch b.ButtonType {
	case "square":
		radius = 0
	case "round":
		radius = rect.Height / 2
	}

	bgPaint := graphics.NewPaint()
	switch {
	case !b.Enabled():
		b.bgAnim.Set(0, theme.DefaultTheme.Motion.Hover)
		bgPaint.SetColor(theme.DefaultTheme.DisabledBg)
	case (b.Checkable && b.Toggled) || b.Selected:
		b.bgAnim.Set(0, theme.DefaultTheme.Motion.Hover)
		bgPaint.SetColor(b.ButtonTheme.BackgroundActive)
	default:
		// Animate the hover/press background blend from the theme's
		// Motion tokens — platform timing intents become visible.
		target, timing := float32(0), theme.DefaultTheme.Motion.Hover
		if b.Pressed {
			target, timing = 2, theme.DefaultTheme.Motion.Press
		} else if b.BaseWidget.Hovered() {
			target = 1
		}
		b.bgAnim.Set(target, timing)
		f := b.bgAnim.Value()
		if f <= 1 {
			bgPaint.SetColor(meui.ColorLerp(b.ButtonTheme.BackgroundInk, b.ButtonTheme.BackgroundHover, f))
		} else {
			bgPaint.SetColor(meui.ColorLerp(b.ButtonTheme.BackgroundHover, b.ButtonTheme.BackgroundPressed, f-1))
		}
	}
	canvas.DrawRoundedRect(rect, radius, radius, bgPaint)

	active := (b.Checkable && b.Toggled) || b.Selected
	if b.BaseWidget.Focused() {
		drawFocusIndicator(canvas, rect, radius)
	} else if !active && b.ButtonTheme.EdgeWidth > 0 {
		// Active (checked/selected) buttons skip the container border:
		// a light grey ring over the accent fill reads as a leak around
		// the corners (WinUI/macOS accent buttons are borderless).
		borderPaint := graphics.NewPaint()
		borderPaint.SetStyle(paintstyle.Stroke)
		borderPaint.SetStrokeWidth(b.ButtonTheme.EdgeWidth)
		borderPaint.SetColor(b.ButtonTheme.EdgeInk)
		canvas.DrawRoundedRect(rect, radius, radius, borderPaint)
	}

	font := b.ButtonTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}

	textPaint := graphics.NewPaint()
	if !b.Enabled() {
		textPaint.SetColor(b.ButtonTheme.TextDisabled)
	} else if b.Selected {
		textPaint.SetColor(b.ButtonTheme.TextActive)
	} else {
		textPaint.SetColor(b.ButtonTheme.TextInk)
	}

	// Draw icon and/or text
	ascent := font.Baseline()
	hMargin := b.ButtonTheme.HMargin
	if hMargin == 0 {
		hMargin = 8
	}
	vMargin := b.ButtonTheme.VMargin
	if vMargin == 0 {
		vMargin = 3
	}
	textWidth := font.SimpleWidth(b.Text)
	totalW := textWidth
	iconSize := float32(14)
	if b.iconSVG != nil {
		if textWidth > 0 {
			totalW += iconSize + 4
		} else {
			totalW = iconSize
		}
	}
	drawX := rect.X + (rect.Width-totalW)/2
	iconY := rect.Y + (rect.Height-iconSize)/2
	if b.iconSVG != nil {
		meui.DrawSVG(canvas, b.iconSVG, drawX, iconY, iconSize, iconSize)
		drawX += iconSize + 4
	}
	// Standard optical centering (matches FlatLaf: text ink centered in
	// the button). The old vMargin idiom landed the baseline at
	// H/2 + ascent/2 — ~1.5px low.
	y := rect.Y + (rect.Height-font.LineHeight())/2 + ascent
	ver := theme.ThemeVersion()
	if b.textCache == nil || b.textCacheVer != ver {
		b.textCache = graphics.NewText(b.Text, &graphics.TextDecoration{Font: font, OnBackgroundInk: textPaint.Color()})
		b.textCacheVer = ver
	}
	if os.Getenv("MEUI_BTN_DEBUG") != "" && b.Text != "" {
		fmt.Printf("[btn] %q rect=(%.0f,%.0f %vx%v) drawX=%.1f y=%.1f w=%.1f cache=%v\n",
			b.Text, rect.X, rect.Y, rect.Width, rect.Height, drawX, y, font.SimpleWidth(b.Text), b.textCache != nil)
	}
	b.textCache.Draw(canvas, drawX, y)

	canvas.Restore()
}

func (b *Button) OnMouseDown(x, y float32, button int) bool {
	if !b.Enabled() {
		return false
	}
	if button == 1 && b.ContainsPoint(x, y) {
		if b.OnRightClicked != nil {
			b.OnRightClicked()
		}
		return true
	}
	if button == 0 && b.ContainsPoint(x, y) {
		b.Pressed = true
		b.MarkForRedraw()
		return true
	}
	return false
}

func (b *Button) OnMouseUp(x, y float32, button int) bool {
	wasPressed := b.Pressed
	b.Pressed = false
	b.MarkForRedraw()
	if !b.Enabled() {
		return false
	}
	if button == 0 && b.ContainsPoint(x, y) {
		if b.GroupID != "" && !b.Selected {
			b.SetSelected(true)
		}
		b.toggleToggled()
		if b.OnClicked != nil {
			b.OnClicked()
		}
		b.MarkForRedraw()
		return true
	}
	_ = wasPressed
	return false
}

func (b *Button) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if !b.Enabled() {
		return false
	}
	if keyCode == platform.KeySpace || keyCode == platform.KeyReturn || keyCode == platform.KeyNumPadEnter {
		b.Pressed = true
		b.MarkForRedraw()
		if b.OnClicked != nil {
			// Fire onClick after a tiny delay so the press flash is visible
			go func() {
				time.Sleep(80 * time.Millisecond)
				b.Pressed = false
				b.MarkForRedraw()
				b.toggleToggled()
				b.OnClicked()
			}()
		} else {
			b.Pressed = false
			b.toggleToggled()
			b.MarkForRedraw()
		}
		return true
	}
	// Shortcut key activation (letters/digits)
	if b.Shortcut != 0 {
		if r := rune(keyCode); r != 0 && unicode.ToUpper(r) == unicode.ToUpper(b.Shortcut) {
			b.toggleToggled()
			if b.OnClicked != nil {
				b.OnClicked()
			}
			b.MarkForRedraw()
			return true
		}
	}
	return false
}

func (b *Button) OnMouseMove(x, y float32) {
	// Hover state is managed by Window.handleMouseMove via PanelAt/SetHoveredState.
	// This method exists to satisfy EventListener interface for position updates.
}

func (b *Button) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "text":
		if s, ok := value.(string); ok {
			b.SetText(s)
		}
	case "onClicked":
		if h, ok := value.(func()); ok {
			b.SetOnClicked(h)
		}
	case "selected":
		if bl, ok := value.(bool); ok {
			b.SetSelected(bl)
		}
	case "icon":
		if s, ok := value.(string); ok {
			b.SetIcon(s)
		}
	case "checkable":
		if bl, ok := value.(bool); ok {
			b.SetCheckable(bl)
		}
	case "toggled", "checked":
		if bl, ok := value.(bool); ok {
			b.SetToggled(bl)
		}
	case "shortcut":
		if s, ok := value.(string); ok && len(s) > 0 {
			b.SetShortcut([]rune(s)[0])
		} else if ch, ok := value.(rune); ok {
			b.SetShortcut(ch)
		}
	case "group":
		if g, ok := value.(string); ok {
			b.SetGroupID(g)
		}
	}
}

func (b *Button) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	font := b.ButtonTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	tw := font.SimpleWidth(b.Text) + 24
	if tw < 80 {
		tw = 80
	}
	lh := theme.DefaultTheme.Metrics.ControlHeight
	if lh <= 0 {
		lh = font.LineHeight() + 8
	}
	if lh < 25 {
		lh = 25
	}
	// An explicitly chosen size (QML `width`/`height` applied via
	// SetWidthDirect) wins over the text-based floor — compact buttons in
	// HStacks must honor it.
	if b.WidthExplicit() && b.Width() > 0 {
		tw = b.Width()
	}
	if b.HeightExplicit() && b.Height() > 0 {
		lh = b.Height()
	}
	return tw, lh
}

func (b *Button) MinimumDimensions() (float32, float32) {
	w, h := float32(50), float32(25)
	if b.WidthExplicit() {
		w = b.Width()
	}
	if b.HeightExplicit() {
		h = b.Height()
	}
	return w, h
}

func (b *Button) OnKeyUp(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if keyCode == platform.KeySpace || keyCode == platform.KeyReturn {
		b.Pressed = false
		b.MarkForRedraw()
		return true
	}
	return false
}

func (b *Button) Dispose() {
	b.Selectable.Dispose(b)
	b.OnClicked = nil
	b.BaseWidget.Dispose()
}

func (b *Button) Focusable() bool {
	return b.Enabled()
}
