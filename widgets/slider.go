package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

type Slider struct {
	dotAnim meui.Animation // Fluent thumb-dot hover growth
	meui.BaseWidget
	*theme.SliderTheme
	Value          float32
	Minimum        float32
	Maximum        float32
	StepSize       float32
	SnapToStep     bool
	ShowTicks      bool
	Dragging       bool
	Orientation    int // 0=horizontal, 1=vertical
	OnValueChanged func(float32)
}

func NewSlider() *Slider {
	s := &Slider{
		BaseWidget:  *meui.NewBaseWidget(),
		SliderTheme: theme.DefaultSliderTheme,
		Minimum:     0,
		Maximum:     100,
		Value:       50,
		StepSize:    1,
	}
	s.SetTabIndex(0)
	return s
}

func (s *Slider) SetValue(value float32) *Slider {
	s.Value = value
	if s.Value < s.Minimum {
		s.Value = s.Minimum
	}
	if s.Value > s.Maximum {
		s.Value = s.Maximum
	}
	s.MarkForRedraw()
	return s
}

func (s *Slider) SetMinimum(min float32) *Slider {
	s.Minimum = min
	if s.Value < s.Minimum {
		s.Value = s.Minimum
	}
	s.MarkForRedraw()
	return s
}

func (s *Slider) SetMaximum(max float32) *Slider {
	s.Maximum = max
	if s.Value > s.Maximum {
		s.Value = s.Maximum
	}
	s.MarkForRedraw()
	return s
}

func (s *Slider) SetOnValueChanged(handler func(float32)) *Slider {
	s.OnValueChanged = handler
	return s
}

func (s *Slider) SetStepSize(step float32) *Slider {
	s.StepSize = step
	if s.StepSize <= 0 {
		s.StepSize = 1
	}
	s.MarkForRedraw()
	return s
}

func (s *Slider) SetSnapToStep(snap bool) *Slider {
	s.SnapToStep = snap
	s.MarkForRedraw()
	return s
}

func (s *Slider) SetShowTicks(show bool) *Slider {
	s.ShowTicks = show
	s.MarkForRedraw()
	return s
}

func (s *Slider) SetOrientation(orient int) *Slider {
	s.Orientation = orient
	s.MarkForRedraw()
	return s
}

func (s *Slider) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !s.Visible() {
		return
	}

	canvas.Save()

	disabled := !s.Enabled()

	if s.Orientation == 1 {
		s.paintVertical(canvas, rect, disabled)
	} else {
		s.paintHorizontal(canvas, rect, disabled)
	}

	canvas.Restore()
}

func (s *Slider) paintHorizontal(canvas *graphics.Canvas, rect graphics.Rect, disabled bool) {
	trackHeight := s.SliderTheme.TrackHeight
	if trackHeight <= 0 {
		trackHeight = 6
	}
	trackRadius := s.SliderTheme.TrackRadius
	if trackRadius < 0 {
		trackRadius = trackHeight / 2
	}
	trackY := rect.Y + (rect.Height-trackHeight)/2

	trackRect := graphics.NewRect(rect.X, trackY, rect.Width, trackHeight)
	trackPaint := graphics.NewPaint()
	if disabled {
		trackPaint.SetColor(theme.DefaultTheme.DisabledBg)
	} else {
		trackPaint.SetColor(s.SliderTheme.TrackInk)
	}
	canvas.DrawRoundedRect(trackRect, trackRadius, trackRadius, trackPaint)

	borderPaint := graphics.NewPaint()
	borderPaint.SetStyle(paintstyle.Stroke)
	borderPaint.SetStrokeWidth(theme.BorderStrokeWidth)
	if disabled {
		borderPaint.SetColor(theme.DefaultTheme.DisabledBorder)
	} else {
		borderPaint.SetColor(s.SliderTheme.EdgeInk)
	}
	if s.SliderTheme.TrackBorder {
		canvas.DrawRoundedRect(trackRect, trackRadius, trackRadius, borderPaint)
	}

	range_ := s.Maximum - s.Minimum
	if range_ > 0 {
		fillPercent := (s.Value - s.Minimum) / range_
		fillWidth := rect.Width * fillPercent

		fillRect := graphics.NewRect(rect.X, trackY, fillWidth, trackHeight)
		fillPaint := graphics.NewPaint()
		if disabled {
			fillPaint.SetColor(theme.DefaultTheme.Disabled)
		} else {
			fillPaint.SetColor(s.SliderTheme.TrackFilled)
		}
		canvas.DrawRoundedRect(fillRect, trackRadius, trackRadius, fillPaint)
	}

	if s.ShowTicks && s.StepSize > 0 {
		tickPaint := graphics.NewPaint()
		if disabled {
			tickPaint.SetColor(theme.DefaultTheme.DisabledBorder)
		} else if s.SliderTheme.TickColor != 0 {
			tickPaint.SetColor(s.SliderTheme.TickColor)
		} else {
			tickPaint.SetColor(s.SliderTheme.EdgeInk)
		}
		tickLen := s.SliderTheme.TickLength
		if tickLen == 0 {
			tickLen = 4
		}
		for v := s.Minimum; v <= s.Maximum; v += s.StepSize {
			percent := (v - s.Minimum) / range_
			tickX := rect.X + percent*rect.Width
			canvas.DrawLine(tickX, trackY-tickLen, tickX, trackY+tickLen, tickPaint)
		}
	}

	thumbSize := s.SliderTheme.ThumbSize
	if thumbSize == 0 {
		thumbSize = 16
	}
	range2 := s.Maximum - s.Minimum
	var thumbX float32
	if range2 > 0 {
		thumbX = rect.X + (s.Value-s.Minimum)/range2*(rect.Width-thumbSize)
	} else {
		thumbX = rect.X
	}
	thumbCX := thumbX + thumbSize/2
	thumbCY := rect.Y + rect.Height/2
	thumbRadius := thumbSize / 2

	if s.Focused() {
		outerPaint := graphics.NewPaint()
		if disabled {
			outerPaint.SetColor(theme.DefaultTheme.Disabled)
		} else {
			c := s.SliderTheme.ThumbHover
			outerPaint.SetColor(graphics.Color((uint32(c) & 0x00FFFFFF) | (0x40 << 24)))
		}
		canvas.DrawCircle(thumbCX, thumbCY, thumbRadius+3, outerPaint)
	}

	innerPaint := graphics.NewPaint()
	if disabled {
		innerPaint.SetColor(theme.DefaultTheme.Disabled)
	} else if s.BaseWidget.Hovered() || s.Dragging {
		innerPaint.SetColor(s.SliderTheme.ThumbHover)
	} else {
		innerPaint.SetColor(s.SliderTheme.ThumbInk)
	}
	canvas.DrawCircle(thumbCX, thumbCY, thumbRadius, innerPaint)
	if s.SliderTheme.ThumbBorder != 0 && !disabled {
		ring := graphics.NewPaint()
		ring.SetStyle(paintstyle.Stroke)
		ring.SetStrokeWidth(1)
		ring.SetColor(s.SliderTheme.ThumbBorder)
		// Stroke at the FULL radius: the inside-band stroke now covers
		// [r-1, r], so the ring reaches the outer edge and the fill
		// cannot leak beyond it (visible as a light halo on white).
		canvas.DrawCircle(thumbCX, thumbCY, thumbRadius, ring)
	}
	if s.SliderTheme.ThumbInner != 0 && !disabled {
		// Fluent grows the accent centre dot on hover/drag.
		target := float32(0)
		if s.BaseWidget.Hovered() || s.Dragging {
			target = 1
		}
		s.dotAnim.Set(target, theme.DefaultTheme.Motion.Hover)
		dot := graphics.NewPaint()
		dot.SetColor(s.SliderTheme.ThumbInner)
		canvas.DrawCircle(thumbCX, thumbCY, thumbRadius*(0.42+0.18*s.dotAnim.Value()), dot)
	}
}

func (s *Slider) paintVertical(canvas *graphics.Canvas, rect graphics.Rect, disabled bool) {
	trackWidth := s.SliderTheme.TrackHeight
	if trackWidth <= 0 {
		trackWidth = 6
	}
	trackRadius := s.SliderTheme.TrackRadius
	if trackRadius < 0 {
		trackRadius = trackWidth / 2
	}
	trackX := rect.X + (rect.Width-trackWidth)/2

	trackRect := graphics.NewRect(trackX, rect.Y, trackWidth, rect.Height)
	trackPaint := graphics.NewPaint()
	if disabled {
		trackPaint.SetColor(theme.DefaultTheme.DisabledBg)
	} else {
		trackPaint.SetColor(s.SliderTheme.TrackInk)
	}
	canvas.DrawRoundedRect(trackRect, trackRadius, trackRadius, trackPaint)

	borderPaint := graphics.NewPaint()
	borderPaint.SetStyle(paintstyle.Stroke)
	borderPaint.SetStrokeWidth(theme.BorderStrokeWidth)
	if disabled {
		borderPaint.SetColor(theme.DefaultTheme.DisabledBorder)
	} else {
		borderPaint.SetColor(s.SliderTheme.EdgeInk)
	}
	if s.SliderTheme.TrackBorder {
		canvas.DrawRoundedRect(trackRect, trackRadius, trackRadius, borderPaint)
	}

	range_ := s.Maximum - s.Minimum
	if range_ > 0 {
		fillPercent := (s.Value - s.Minimum) / range_
		fillHeight := rect.Height * fillPercent

		fillRect := graphics.NewRect(trackX, rect.Y+rect.Height-fillHeight, trackWidth, fillHeight)
		fillPaint := graphics.NewPaint()
		if disabled {
			fillPaint.SetColor(theme.DefaultTheme.Disabled)
		} else {
			fillPaint.SetColor(s.SliderTheme.TrackFilled)
		}
		canvas.DrawRoundedRect(fillRect, trackRadius, trackRadius, fillPaint)
	}

	if s.ShowTicks && s.StepSize > 0 {
		tickPaint := graphics.NewPaint()
		if disabled {
			tickPaint.SetColor(theme.DefaultTheme.DisabledBorder)
		} else if s.SliderTheme.TickColor != 0 {
			tickPaint.SetColor(s.SliderTheme.TickColor)
		} else {
			tickPaint.SetColor(s.SliderTheme.EdgeInk)
		}
		tickLen := s.SliderTheme.TickLength
		if tickLen == 0 {
			tickLen = 4
		}
		for v := s.Minimum; v <= s.Maximum; v += s.StepSize {
			percent := (v - s.Minimum) / range_
			tickY := rect.Y + (1-percent)*rect.Height
			canvas.DrawLine(trackX-tickLen, tickY, trackX+trackWidth+tickLen, tickY, tickPaint)
		}
	}

	thumbHeight := s.SliderTheme.ThumbSize
	if thumbHeight == 0 {
		thumbHeight = 16
	}
	range2 := s.Maximum - s.Minimum
	var thumbY float32
	if range2 > 0 {
		thumbY = rect.Y + (1-(s.Value-s.Minimum)/range2)*(rect.Height-thumbHeight)
	} else {
		thumbY = rect.Y
	}
	thumbCX := rect.X + rect.Width/2
	thumbCY := thumbY + thumbHeight/2
	thumbRadius := thumbHeight / 2

	if s.Focused() {
		outerPaint := graphics.NewPaint()
		if disabled {
			outerPaint.SetColor(theme.DefaultTheme.Disabled)
		} else {
			c := s.SliderTheme.ThumbHover
			outerPaint.SetColor(graphics.Color((uint32(c) & 0x00FFFFFF) | (0x40 << 24)))
		}
		canvas.DrawCircle(thumbCX, thumbCY, thumbRadius+3, outerPaint)
	}

	innerPaint := graphics.NewPaint()
	if disabled {
		innerPaint.SetColor(theme.DefaultTheme.Disabled)
	} else if s.BaseWidget.Hovered() || s.Dragging {
		innerPaint.SetColor(s.SliderTheme.ThumbHover)
	} else {
		innerPaint.SetColor(s.SliderTheme.ThumbInk)
	}
	canvas.DrawCircle(thumbCX, thumbCY, thumbRadius, innerPaint)
	if s.SliderTheme.ThumbBorder != 0 && !disabled {
		ring := graphics.NewPaint()
		ring.SetStyle(paintstyle.Stroke)
		ring.SetStrokeWidth(1)
		ring.SetColor(s.SliderTheme.ThumbBorder)
		// Stroke at the FULL radius: the inside-band stroke now covers
		// [r-1, r], so the ring reaches the outer edge and the fill
		// cannot leak beyond it (visible as a light halo on white).
		canvas.DrawCircle(thumbCX, thumbCY, thumbRadius, ring)
	}
	if s.SliderTheme.ThumbInner != 0 && !disabled {
		// Fluent grows the accent centre dot on hover/drag.
		target := float32(0)
		if s.BaseWidget.Hovered() || s.Dragging {
			target = 1
		}
		s.dotAnim.Set(target, theme.DefaultTheme.Motion.Hover)
		dot := graphics.NewPaint()
		dot.SetColor(s.SliderTheme.ThumbInner)
		canvas.DrawCircle(thumbCX, thumbCY, thumbRadius*(0.42+0.18*s.dotAnim.Value()), dot)
	}
}

func (s *Slider) OnMouseDown(x, y float32, button int) bool {
	if !s.Enabled() {
		return false
	}
	if !s.ContainsPoint(x, y) {
		return false
	}
	s.Dragging = true
	if s.Orientation == 1 {
		s.setValueFromY(y)
	} else {
		s.setValueFromX(x)
	}
	s.MarkForRedraw()
	return true
}

func (s *Slider) OnMouseUp(x, y float32, button int) bool {
	s.Dragging = false
	s.MarkForRedraw()
	if !s.Enabled() {
		return false
	}
	return true
}

func (s *Slider) OnMouseMove(x, y float32) {
	if !s.Enabled() {
		return
	}
	if s.Dragging {
		if s.Orientation == 1 {
			s.setValueFromY(y)
		} else {
			s.setValueFromX(x)
		}
		s.MarkForRedraw()
	}
}

func (s *Slider) OnMouseWheel(x, y float32, dx, dy float32) bool {
	if !s.Enabled() || dy == 0 {
		return false
	}
	step := s.StepSize
	if step <= 0 {
		step = 1
	}
	oldVal := s.Value
	if s.Orientation == Vertical {
		s.SetValue(s.Value + dy*step*5)
	} else {
		s.SetValue(s.Value + dy*step*5)
	}
	if s.Value != oldVal && s.OnValueChanged != nil {
		s.OnValueChanged(s.Value)
	}
	return true
}

func (s *Slider) setValueFromX(x float32) {
	frame := s.FrameRect()
	thumbSize := s.SliderTheme.ThumbSize
	if thumbSize == 0 {
		thumbSize = 16
	}
	percent := x / (frame.Width - thumbSize)
	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}
	newValue := s.Minimum + percent*(s.Maximum-s.Minimum)
	if s.SnapToStep && s.StepSize > 0 {
		newValue = s.Minimum + float32(int((newValue-s.Minimum)/s.StepSize+0.5))*s.StepSize
	}
	if newValue != s.Value {
		s.Value = newValue
		if s.OnValueChanged != nil {
			s.OnValueChanged(s.Value)
		}
	}
}

func (s *Slider) setValueFromY(y float32) {
	frame := s.FrameRect()
	thumbHeight := s.SliderTheme.ThumbSize
	if thumbHeight == 0 {
		thumbHeight = 16
	}
	percent := 1 - y/(frame.Height-thumbHeight)
	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}
	newValue := s.Minimum + percent*(s.Maximum-s.Minimum)
	if s.SnapToStep && s.StepSize > 0 {
		newValue = s.Minimum + float32(int((newValue-s.Minimum)/s.StepSize+0.5))*s.StepSize
	}
	if newValue != s.Value {
		s.Value = newValue
		if s.OnValueChanged != nil {
			s.OnValueChanged(s.Value)
		}
	}
}

// OnKeyRepeat is a key the user is HOLDING — Backspace over a word, an arrow along a row, a scrub.
// The window classifies an OS auto-repeat as a repeat rather than a fresh press (a widget without
// this method is asked once, deliberately), so this is what makes a held key work here.
func (s *Slider) OnKeyRepeat(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	return s.OnKeyDown(keyCode, mod)
}

func (s *Slider) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if !s.Enabled() {
		return false
	}
	step := float32(1)
	if mod.ShiftDown() {
		step = float32(10)
	}
	if mod.ControlDown() {
		step = float32(25)
	}
	switch keyCode {
	case platform.KeyLeft, platform.KeyDown:
		s.SetValue(s.Value - step)
		if s.OnValueChanged != nil {
			s.OnValueChanged(s.Value)
		}
		return true
	case platform.KeyRight, platform.KeyUp:
		s.SetValue(s.Value + step)
		if s.OnValueChanged != nil {
			s.OnValueChanged(s.Value)
		}
		return true
	case platform.KeyHome:
		s.SetValue(s.Minimum)
		if s.OnValueChanged != nil {
			s.OnValueChanged(s.Value)
		}
		return true
	case platform.KeyEnd:
		s.SetValue(s.Maximum)
		if s.OnValueChanged != nil {
			s.OnValueChanged(s.Value)
		}
		return true
	}
	return false
}

func (s *Slider) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "value":
		if f, ok := value.(float64); ok {
			s.SetValue(float32(f))
		}
	case "minimumValue", "min":
		if f, ok := value.(float64); ok {
			s.SetMinimum(float32(f))
		}
	case "maximumValue", "max":
		if f, ok := value.(float64); ok {
			s.SetMaximum(float32(f))
		}
	case "stepSize", "step":
		if f, ok := value.(float64); ok {
			s.SetStepSize(float32(f))
		}
	case "snapToStep":
		if b, ok := value.(bool); ok {
			s.SetSnapToStep(b)
		}
	case "showTicks", "tickMarksEnabled":
		if b, ok := value.(bool); ok {
			s.SetShowTicks(b)
		}
	}
}

func (s *Slider) Focusable() bool {
	return s.Enabled()
}
