package widgets

import (
	"fmt"
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

type ColorPicker struct {
	meui.BaseWidget
	*theme.ColorPickerTheme
	Hue            float32
	Saturation     float32
	Value          float32
	Alpha          float32
	Red            uint8
	Green          uint8
	Blue           uint8
	draggingHue    bool
	draggingSV     bool
	OnColorChanged func(r, g, b, a uint8)
}

func NewColorPicker() *ColorPicker {
	cp := &ColorPicker{
		BaseWidget:       *meui.NewBaseWidget(),
		ColorPickerTheme: theme.DefaultColorPickerTheme,
		Hue:              0,
		Saturation:       1,
		Value:            1,
		Alpha:            1,
	}
	cp.updateRGB()
	return cp
}

// SetOnColorChanged registers the live color-changed callback.
func (c *ColorPicker) SetOnColorChanged(handler func(r, g, b, a uint8)) *ColorPicker {
	c.OnColorChanged = handler
	return c
}

// GetColor returns the currently selected color components.
func (c *ColorPicker) GetColor() (r, g, b, a uint8) {
	a = uint8(c.Alpha*255.0 + 0.5)
	return c.Red, c.Green, c.Blue, a
}

func (c *ColorPicker) SetColor(r, g, b, a uint8) *ColorPicker {
	c.Red = r
	c.Green = g
	c.Blue = b
	c.Alpha = float32(a) / 255.0
	c.updateHSV()
	c.MarkForRedraw()
	return c
}

func (c *ColorPicker) SetHue(h float32) *ColorPicker {
	c.Hue = h
	c.updateRGB()
	c.MarkForRedraw()
	return c
}

func (c *ColorPicker) UpdateRGB() {
	c.updateRGB()
}

func (c *ColorPicker) UpdateHSV() {
	c.updateHSV()
}

func (c *ColorPicker) updateRGB() {
	h := c.Hue / 360.0
	s := c.Saturation
	v := c.Value

	var r, g, b float32
	i := int(h * 6)
	f := h*6 - float32(i)
	p := v * (1 - s)
	q := v * (1 - f*s)
	t := v * (1 - (1-f)*s)

	switch i % 6 {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	case 5:
		r, g, b = v, p, q
	}

	c.Red = uint8(r * 255)
	c.Green = uint8(g * 255)
	c.Blue = uint8(b * 255)
}

func (c *ColorPicker) updateHSV() {
	r := float32(c.Red) / 255.0
	g := float32(c.Green) / 255.0
	b := float32(c.Blue) / 255.0

	max := r
	if g > max {
		max = g
	}
	if b > max {
		max = b
	}
	min := r
	if g < min {
		min = g
	}
	if b < min {
		min = b
	}

	c.Value = max
	if max == 0 {
		c.Saturation = 0
		c.Hue = 0
		return
	}
	c.Saturation = (max - min) / max

	diff := max - min
	if max == r {
		c.Hue = (g - b) / diff * 60
	} else if max == g {
		c.Hue = (2 + (b-r)/diff) * 60
	} else {
		c.Hue = (4 + (r-g)/diff) * 60
	}
	if c.Hue < 0 {
		c.Hue += 360
	}
}

func (c *ColorPicker) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !c.Visible() {
		return
	}

	canvas.Save()

	bgPaint := graphics.NewPaint()
	bgPaint.SetColor(c.ColorPickerTheme.BackgroundInk)
	canvas.DrawRoundedRect(rect, c.ColorPickerTheme.CornerRadius, c.ColorPickerTheme.CornerRadius, bgPaint)

	svSize := rect.Width - 30
	if svSize > rect.Height {
		svSize = rect.Height
	}
	hueBarW := float32(20)
	hueBarX := rect.X + svSize + 5
	hueBarY := rect.Y
	hueBarH := svSize

	// SV rectangle
	for y := 0; y < int(svSize); y++ {
		for x := 0; x < int(svSize); x++ {
			s := float32(x) / svSize
			v := 1 - float32(y)/svSize
			r, g, b := hsvToRGB(c.Hue, s, v)
			paint := graphics.NewPaint()
			paint.SetColor(graphics.Color(uint32(0xFF)<<24 | uint32(r)<<16 | uint32(g)<<8 | uint32(b)))
			canvas.DrawRect(graphics.NewRect(rect.X+float32(x), rect.Y+float32(y), 1, 1), paint)
		}
	}

	// SV cursor
	cursorX := rect.X + c.Saturation*svSize
	cursorY := rect.Y + (1-c.Value)*svSize
	cursorPaint := graphics.NewPaint()
	cursorPaint.SetStyle(paintstyle.Stroke)
	cursorPaint.SetStrokeWidth(theme.CursorStrokeWidth)
	cursorPaint.SetColor(graphics.Color(0xFFFFFFFF))
	canvas.DrawOval(graphics.NewRect(cursorX-5, cursorY-5, 10, 10), cursorPaint)

	// Hue bar
	for i := 0; i < int(hueBarH); i++ {
		h := float32(i) / hueBarH * 360
		r, g, b := hsvToRGB(h, 1, 1)
		paint := graphics.NewPaint()
		paint.SetColor(graphics.Color(uint32(0xFF)<<24 | uint32(r)<<16 | uint32(g)<<8 | uint32(b)))
		canvas.DrawRect(graphics.NewRect(hueBarX, hueBarY+float32(i), hueBarW, 1.5), paint)
	}

	// Hue cursor
	hueCursorY := hueBarY + (c.Hue/360)*hueBarH
	hueCursorPaint := graphics.NewPaint()
	hueCursorPaint.SetStyle(paintstyle.Stroke)
	hueCursorPaint.SetStrokeWidth(theme.CursorStrokeWidth)
	hueCursorPaint.SetColor(graphics.Color(0xFFFFFFFF))
	canvas.DrawRect(graphics.NewRect(hueBarX-2, hueCursorY-3, hueBarW+4, 6), hueCursorPaint)

	// Color preview
	previewY := rect.Y + svSize + 5
	previewH := float32(25)
	previewRect := graphics.NewRect(rect.X, previewY, svSize, previewH)
	previewPaint := graphics.NewPaint()
	previewPaint.SetColor(graphics.Color(uint32(c.Alpha*255)<<24 | uint32(c.Red)<<16 | uint32(c.Green)<<8 | uint32(c.Blue)))
	canvas.DrawRect(previewRect, previewPaint)

	// Hex label
	hexStr := fmt.Sprintf("#%02X%02X%02X", c.Red, c.Green, c.Blue)
	font := c.ColorPickerTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	hexPaint := graphics.NewPaint()
	lum := float32(c.Red)*0.299 + float32(c.Green)*0.587 + float32(c.Blue)*0.114
	if lum > 128 {
		hexPaint.SetColor(graphics.Color(0xFF000000))
	} else {
		hexPaint.SetColor(graphics.Color(0xFFFFFFFF))
	}
	hexW := font.SimpleWidth(hexStr)
	ht := graphics.NewText(hexStr, &graphics.TextDecoration{Font: font, OnBackgroundInk: hexPaint.Color()})
	ht.Draw(canvas, rect.X+svSize/2-hexW/2, previewY+previewH/2+font.Baseline()/2)

	canvas.Restore()
}

func (c *ColorPicker) OnMouseDown(x, y float32, button int) bool {
	if !c.Enabled() {
		return false
	}

	svSize := c.Width() - 30
	if svSize > c.Height() {
		svSize = c.Height()
	}
	hueBarX := svSize + 5
	hueBarH := svSize

	previewY := svSize + 5
	previewH := float32(25)

	if y >= previewY && y <= previewY+previewH && x >= 0 && x <= svSize {
		platform.GlobalClipboard.SetText(fmt.Sprintf("#%02X%02X%02X", c.Red, c.Green, c.Blue))
		return true
	}

	if x >= hueBarX && x <= hueBarX+20 && y >= 0 && y <= hueBarH {
		c.draggingHue = true
		c.updateHueFromY(y, hueBarH)
		return true
	}

	if x >= 0 && x <= svSize && y >= 0 && y <= svSize {
		c.draggingSV = true
		c.updateSVFromXY(x, y, svSize)
		return true
	}
	return false
}

func (c *ColorPicker) OnMouseMove(x, y float32) {
	if c.draggingHue {
		svSize := c.Width() - 30
		if svSize > c.Height() {
			svSize = c.Height()
		}
		c.updateHueFromY(y, svSize)
		return
	}
	if c.draggingSV {
		svSize := c.Width() - 30
		if svSize > c.Height() {
			svSize = c.Height()
		}
		c.updateSVFromXY(x, y, svSize)
	}
}

func (c *ColorPicker) OnMouseUp(x, y float32, button int) bool {
	c.draggingHue = false
	c.draggingSV = false
	return false
}

func (c *ColorPicker) OnMouseWheel(x, y float32, dx, dy float32) bool {
	if !c.Enabled() || dy == 0 {
		return false
	}

	svSize := c.Width() - 30
	if svSize > c.Height() {
		svSize = c.Height()
	}
	hueBarX := svSize + 5
	hueBarH := svSize

	if x >= hueBarX && x <= hueBarX+20 && y >= 0 && y <= hueBarH {
		c.Hue -= dy * 10
		if c.Hue < 0 {
			c.Hue += 360
		}
		if c.Hue >= 360 {
			c.Hue -= 360
		}
		c.updateRGB()
		c.MarkForRedraw()
		if c.OnColorChanged != nil {
			c.OnColorChanged(c.Red, c.Green, c.Blue, uint8(c.Alpha*255))
		}
		return true
	}

	if x >= 0 && x <= svSize && y >= 0 && y <= svSize {
		c.Saturation -= dy * 0.05
		if c.Saturation < 0 {
			c.Saturation = 0
		}
		if c.Saturation > 1 {
			c.Saturation = 1
		}
		c.Value += dy * 0.05
		if c.Value < 0 {
			c.Value = 0
		}
		if c.Value > 1 {
			c.Value = 1
		}
		c.updateRGB()
		c.MarkForRedraw()
		if c.OnColorChanged != nil {
			c.OnColorChanged(c.Red, c.Green, c.Blue, uint8(c.Alpha*255))
		}
		return true
	}
	return false
}

func (c *ColorPicker) updateHueFromY(y float32, hueBarH float32) {
	h := y / hueBarH
	if h < 0 {
		h = 0
	}
	if h > 1 {
		h = 1
	}
	c.Hue = h * 360
	c.updateRGB()
	c.MarkForRedraw()
	if c.OnColorChanged != nil {
		c.OnColorChanged(c.Red, c.Green, c.Blue, uint8(c.Alpha*255))
	}
}

func (c *ColorPicker) updateSVFromXY(x, y float32, svSize float32) {
	s := x / svSize
	v := 1 - y/svSize
	if s < 0 {
		s = 0
	}
	if s > 1 {
		s = 1
	}
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	c.Saturation = s
	c.Value = v
	c.updateRGB()
	c.MarkForRedraw()
	if c.OnColorChanged != nil {
		c.OnColorChanged(c.Red, c.Green, c.Blue, uint8(c.Alpha*255))
	}
}

func (c *ColorPicker) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "red":
		if f, ok := value.(float64); ok {
			c.Red = uint8(f)
			c.updateHSV()
		}
	case "green":
		if f, ok := value.(float64); ok {
			c.Green = uint8(f)
			c.updateHSV()
		}
	case "blue":
		if f, ok := value.(float64); ok {
			c.Blue = uint8(f)
			c.updateHSV()
		}
	case "hue":
		if f, ok := value.(float64); ok {
			c.Hue = float32(f)
			c.updateRGB()
		}
	}
}

func hsvToRGB(h, s, v float32) (uint8, uint8, uint8) {
	h = h / 360.0
	var r, g, b float32
	i := int(h * 6)
	f := h*6 - float32(i)
	p := v * (1 - s)
	q := v * (1 - f*s)
	t := v * (1 - (1-f)*s)

	switch i % 6 {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	case 5:
		r, g, b = v, p, q
	}

	return uint8(r * 255), uint8(g * 255), uint8(b * 255)
}
