package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/theme"
)

type RadioButton struct {
	meui.BaseWidget
	*theme.RadioButtonTheme
	Selectable
	Text         string
	OnSelected   func(bool)
	OnChanged    func(bool)
	textCache    *graphics.Text
	textCacheVer uint64
}

func NewRadioButton() *RadioButton {
	r := &RadioButton{
		BaseWidget:       *meui.NewBaseWidget(),
		RadioButtonTheme: theme.DefaultRadioButtonTheme,
	}
	r.SetTabIndex(0)
	return r
}

func (r *RadioButton) GetSelectable() *Selectable {
	return &r.Selectable
}

func (r *RadioButton) SetText(text string) *RadioButton {
	r.Text = text
	r.textCache = nil
	r.MarkForRedraw()
	return r
}

func (r *RadioButton) SetSelected(selected bool) *RadioButton {
	r.Selectable.SetSelected(selected, r, r.OnSelected)
	r.MarkForRedraw()
	if r.OnChanged != nil {
		r.OnChanged(selected)
	}
	return r
}

func (r *RadioButton) SetGroupID(id string) *RadioButton {
	r.Selectable.SetGroupID(id, r)
	return r
}

func (r *RadioButton) SetOnSelected(handler func(bool)) *RadioButton {
	r.OnSelected = handler
	return r
}

func (r *RadioButton) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !r.Visible() {
		return
	}

	canvas.Save()

	circleSize := r.RadioButtonTheme.BoxSize
	circleY := rect.Y + (rect.Height-circleSize)/2

	circleRect := graphics.NewRect(rect.X, circleY, circleSize, circleSize)

	bgPaint := graphics.NewPaint()
	if !r.Enabled() {
		bgPaint.SetColor(theme.DefaultTheme.DisabledBg)
	} else if r.BaseWidget.Hovered() {
		bgPaint.SetColor(r.RadioButtonTheme.BackgroundHover)
	} else {
		bgPaint.SetColor(r.RadioButtonTheme.BackgroundInk)
	}
	canvas.DrawOval(circleRect, bgPaint)

	borderPaint := graphics.NewPaint()
	borderPaint.SetStyle(paintstyle.Stroke)
	borderPaint.SetStrokeWidth(theme.BorderStrokeWidth)
	if !r.Enabled() {
		borderPaint.SetColor(theme.DefaultTheme.DisabledBorder)
	} else if r.BaseWidget.Focused() {
		borderPaint.SetColor(r.RadioButtonTheme.EdgeFocused)
	} else {
		borderPaint.SetColor(r.RadioButtonTheme.EdgeInk)
	}
	canvas.DrawOval(circleRect, borderPaint)

	if r.Selected {
		innerPaint := graphics.NewPaint()
		if !r.Enabled() {
			innerPaint.SetColor(theme.DefaultTheme.DisabledText)
		} else {
			innerPaint.SetColor(r.RadioButtonTheme.SelectionInk)
		}
		innerRect := graphics.NewRect(rect.X+5, circleY+5, circleSize-10, circleSize-10)
		canvas.DrawOval(innerRect, innerPaint)
	}

	textX := rect.X + r.RadioButtonTheme.BoxSize + r.RadioButtonTheme.BoxSpacing
	font := r.RadioButtonTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	ascent := font.Baseline()
	textHeight := font.LineHeight()
	textY := rect.Y + (rect.Height-textHeight)/2 + ascent

	textPaint := graphics.NewPaint()
	if !r.Enabled() {
		textPaint.SetColor(r.RadioButtonTheme.TextDisabled)
	} else {
		textPaint.SetColor(r.RadioButtonTheme.TextInk)
	}
	ver := theme.ThemeVersion()
	if r.textCache == nil || r.textCacheVer != ver {
		r.textCache = graphics.NewText(r.Text, &graphics.TextDecoration{Font: font, OnBackgroundInk: textPaint.Color()})
		r.textCacheVer = ver
	}
	r.textCache.Draw(canvas, textX, textY)

	canvas.Restore()
}

// PreferredDimensions reports the circle + spacing + text width so stack
// layouts assign a frame wide enough for the label.
func (r *RadioButton) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	font := r.RadioButtonTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	textW := font.SimpleWidth(r.Text)
	width := r.RadioButtonTheme.BoxSize + r.RadioButtonTheme.BoxSpacing + textW + 6
	height := r.RadioButtonTheme.BoxSize + 4
	return width, height
}

func (r *RadioButton) OnMouseDown(x, y float32, button int) bool {
	if !r.Enabled() {
		return false
	}
	if r.ContainsPoint(x, y) && !r.Selected {
		r.SetSelected(true)
	}
	return true
}

func (r *RadioButton) OnMouseMove(x, y float32) {
	if !r.Enabled() {
		return
	}
	// Hover managed by Window.handleMouseMove
}

func (r *RadioButton) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "text":
		if s, ok := value.(string); ok {
			r.SetText(s)
		}
	case "selected", "checked":
		if b, ok := value.(bool); ok {
			r.SetSelected(b)
		}
	case "group":
		if g, ok := value.(string); ok {
			r.SetGroupID(g)
		}
	}
}

func (r *RadioButton) Dispose() {
	r.Selectable.Dispose(r)
	r.BaseWidget.Dispose()
}

func (r *RadioButton) Focusable() bool {
	return r.Enabled()
}
