package widgets

import (
	"fmt"
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
	"os"
)

type CheckBox struct {
	meui.BaseWidget
	*theme.CheckBoxTheme
	Text         string
	Checked      bool
	Pressed      bool
	OnChanged    func(bool)
	textCache    *graphics.Text
	textCacheVer uint64
}

func NewCheckBox() *CheckBox {
	c := &CheckBox{
		BaseWidget: *meui.NewBaseWidget(),
	}
	c.SetTabIndex(0)
	c.CheckBoxTheme = theme.DefaultCheckBoxTheme
	return c
}

func (c *CheckBox) SetText(text string) *CheckBox {
	c.Text = text
	c.textCache = nil
	c.MarkForRedraw()
	return c
}

func (c *CheckBox) SetChecked(checked bool) *CheckBox {
	if c.Checked == checked {
		return c
	}
	c.Checked = checked
	c.MarkForRedraw()
	if c.OnChanged != nil {
		c.OnChanged(checked)
	}
	return c
}

// SetOnCheckChanged registers the toggled callback (stateChanged in Qt).
func (c *CheckBox) SetOnCheckChanged(handler func(checked bool)) *CheckBox {
	c.OnChanged = handler
	return c
}

func (c *CheckBox) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if os.Getenv("MEUI_FLUSH_DEBUG") != "" {
		fmt.Printf("[cbPaint] visible=%v enabled=%v rect=(%.0f,%.0f %.0fx%.0f)\n", c.Visible(), c.Enabled(), rect.X, rect.Y, rect.Width, rect.Height)
	}
	if !c.Visible() {
		return
	}

	canvas.Save()

	boxSize := c.CheckBoxTheme.BoxSize
	boxY := rect.Y + (rect.Height-boxSize)/2

	boxRect := graphics.NewRect(rect.X, boxY, boxSize, boxSize)
	radius := c.CheckBoxTheme.CornerRadius

	bgPaint := graphics.NewPaint()
	switch {
	case !c.Enabled():
		bgPaint.SetColor(theme.DefaultTheme.DisabledBg)
	case c.Checked:
		// Every target platform fills the checked box with the accent
		// (an outline-only box reads as indeterminate, not checked).
		bgPaint.SetColor(c.CheckBoxTheme.SelectionInk)
	case c.Pressed:
		bgPaint.SetColor(theme.DefaultTheme.DisabledBg)
	case c.BaseWidget.Hovered():
		bgPaint.SetColor(c.CheckBoxTheme.BackgroundHover)
	default:
		bgPaint.SetColor(c.CheckBoxTheme.BackgroundInk)
	}
	canvas.DrawRoundedRect(boxRect, radius, radius, bgPaint)

	borderPaint := graphics.NewPaint()
	borderPaint.SetStyle(paintstyle.Stroke)
	borderPaint.SetStrokeWidth(theme.BorderStrokeWidth)
	if !c.Enabled() {
		borderPaint.SetColor(theme.DefaultTheme.DisabledBorder)
	} else if c.BaseWidget.Focused() {
		borderPaint.SetColor(c.CheckBoxTheme.EdgeFocused)
	} else if c.Checked {
		borderPaint.SetColor(c.CheckBoxTheme.SelectionInk)
	} else {
		borderPaint.SetColor(c.CheckBoxTheme.EdgeInk)
	}
	canvas.DrawRoundedRect(boxRect, radius, radius, borderPaint)

	if c.Checked {
		checkPaint := graphics.NewPaint()
		if !c.Enabled() {
			checkPaint.SetColor(theme.DefaultTheme.DisabledText)
		} else {
			// On-accent colour: white on dark accents, black on the
			// light accents Fluent dark mode uses.
			checkPaint.SetColor(theme.DefaultTheme.OnAccentColor())
		}
		checkPaint.SetStyle(paintstyle.Stroke)
		checkPaint.SetStrokeWidth(theme.CheckStrokeWidth)

		// Draw a proper checkmark path
		margin := boxSize * 0.2
		checkSize := boxSize - margin*2
		startX := rect.X + margin
		startY := boxY + margin
		midX := startX + checkSize*0.3
		endX := startX + checkSize
		endY := startY + checkSize*0.25

		canvas.DrawLine(startX, startY+checkSize*0.55, midX, startY+checkSize*0.85, checkPaint)
		canvas.DrawLine(midX, startY+checkSize*0.85, endX, endY, checkPaint)
	}

	font := c.CheckBoxTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	textX := rect.X + boxSize + c.CheckBoxTheme.BoxSpacing
	ascent := font.Baseline()
	textY := rect.Y + (rect.Height-ascent)/2 + ascent

	textPaint := graphics.NewPaint()
	if !c.Enabled() {
		textPaint.SetColor(c.CheckBoxTheme.TextDisabled)
	} else {
		textPaint.SetColor(c.CheckBoxTheme.TextInk)
	}
	ver := theme.ThemeVersion()
	if c.textCache == nil || c.textCacheVer != ver {
		c.textCache = graphics.NewText(c.Text, &graphics.TextDecoration{Font: font, OnBackgroundInk: textPaint.Color()})
		c.textCacheVer = ver
	}
	c.textCache.Draw(canvas, textX, textY)

	canvas.Restore()
}

// PreferredDimensions reports the box + spacing + text width so stack
// layouts assign a frame wide enough for the label (a bare minimum frame
// clipped the text under the next widget).
func (c *CheckBox) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	font := c.CheckBoxTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	textW := font.SimpleWidth(c.Text)
	width := c.CheckBoxTheme.BoxSize + c.CheckBoxTheme.BoxSpacing + textW + 6
	height := c.CheckBoxTheme.BoxSize + 4
	return width, height
}

func (c *CheckBox) OnMouseDown(x, y float32, button int) bool {
	if !c.Enabled() {
		return false
	}
	c.Pressed = true
	c.MarkForRedraw()
	return true
}

func (c *CheckBox) OnMouseUp(x, y float32, button int) bool {
	wasPressed := c.Pressed
	c.Pressed = false
	if wasPressed && c.ContainsPoint(x, y) {
		c.SetChecked(!c.Checked)
	}
	c.MarkForRedraw()
	return true
}

func (c *CheckBox) OnMouseMove(x, y float32) {
	// Hover managed by Window.handleMouseMove
}

func (c *CheckBox) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "text":
		if s, ok := value.(string); ok {
			c.SetText(s)
		}
	case "checked":
		if b, ok := value.(bool); ok {
			c.SetChecked(b)
		}
	}
}

func (c *CheckBox) Focusable() bool {
	return c.Enabled()
}

func (c *CheckBox) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if !c.Enabled() {
		return false
	}
	if keyCode == platform.KeySpace {
		c.SetChecked(!c.Checked)
		return true
	}
	return false
}
