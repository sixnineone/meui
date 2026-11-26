package widgets

import (
	"time"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

type ScrollBar struct {
	meui.BaseWidget
	*theme.ScrollBarTheme
	Orientation    int
	Value          float32
	Min            float32
	Max            float32
	VisibleAmount  float32
	OnValueChanged func(float32)
	dragging       bool
	dragStartPos   float32
	dragStartVal   float32
	// orientationExplicit is set when the caller chooses an orientation;
	// otherwise the bar auto-orients from its frame aspect ratio.
	orientationExplicit bool
	// Overlay mode (macOS/GNOME): the bar paints a thin pill and fades
	// out shortly after the last scroll.
	lastActive time.Time
	fadeAnim   meui.Animation
}

// touch marks scroll activity so the overlay pill fades out after the
// theme's ScrollFade timing.
func (s *ScrollBar) touch() { s.lastActive = time.Now() }

const (
	Horizontal = iota
	Vertical
)

func NewScrollBar() *ScrollBar {
	return &ScrollBar{
		BaseWidget:     *meui.NewBaseWidget(),
		ScrollBarTheme: theme.DefaultScrollBarTheme,
		Min:            0,
		Max:            100,
		Value:          0,
		Orientation:    Vertical,
	}
}

func (s *ScrollBar) SetOrientation(orientation int) *ScrollBar {
	s.Orientation = orientation
	s.orientationExplicit = true
	s.MarkForRedraw()
	return s
}

// effOrientation returns the explicit orientation if set, otherwise derives
// it from the frame aspect ratio (wide = horizontal, tall = vertical).
func (s *ScrollBar) effOrientation() int {
	if s.orientationExplicit {
		return s.Orientation
	}
	frame := s.FrameRect()
	if frame.Width >= frame.Height {
		return Horizontal
	}
	return Vertical
}

func (s *ScrollBar) SetValue(value float32) *ScrollBar {
	// Overlay bars fade after inactivity: only a VALUE CHANGE is
	// activity. Hosts re-sync the same value every frame during layout,
	// and treating those as activity kept the pill permanently visible.
	if value != s.Value {
		s.touch()
	}
	max := s.Max
	if s.VisibleAmount > 0 && s.VisibleAmount < s.Max-s.Min {
		// Absolute-pixel units: the offset can never exceed the
		// scrollable range (content extent - viewport extent).
		max = s.Max - s.VisibleAmount
	}
	s.Value = value
	if s.Value < s.Min {
		s.Value = s.Min
	}
	if s.Value > max {
		s.Value = max
	}
	s.MarkForRedraw()
	return s
}

func (s *ScrollBar) SetMin(min float32) *ScrollBar {
	s.Min = min
	s.MarkForRedraw()
	return s
}

func (s *ScrollBar) SetMax(max float32) *ScrollBar {
	s.Max = max
	s.MarkForRedraw()
	return s
}

func (s *ScrollBar) SetOnValueChanged(handler func(float32)) *ScrollBar {
	s.OnValueChanged = handler
	return s
}

func (s *ScrollBar) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !s.Visible() {
		return
	}

	canvas.Save()

	disabled := !s.Enabled()

	overlay := s.ScrollBarTheme.Overlay
	// Auto-hide is OPT-IN (macOS/GNOME presets): the bar fades to nothing
	// while idle. The default is a persistent bar — visible whenever the
	// widget is.
	fade := float32(1)
	if s.ScrollBarTheme.AutoHide {
		target := float32(0)
		if s.BaseWidget.Hovered() || s.dragging || time.Since(s.lastActive) < 900*time.Millisecond {
			target = 1
		}
		s.fadeAnim.Set(target, theme.DefaultTheme.Motion.ScrollFade)
		fade = s.fadeAnim.Value()
		if fade < 0.03 {
			canvas.Restore()
			return
		}
	}
	if !overlay {
		// FlatLaf scrollbars: subtle track, NO borders anywhere — the
		// thumb is a floating rounded pill.
		trackPaint := graphics.NewPaint()
		if disabled {
			trackPaint.SetColor(theme.DefaultTheme.DisabledBg)
		} else {
			trackPaint.SetColor(s.ScrollBarTheme.TrackInk)
		}
		if s.ScrollBarTheme.AutoHide {
			trackPaint.SetColor(scaleAlpha(trackPaint.Color(), fade))
		}
		canvas.DrawRect(rect, trackPaint)
	}

	// Standard scrollbar model with ABSOLUTE-PIXEL units:
	//   Max-Min          = content extent
	//   VisibleAmount    = viewport extent
	//   scrollableRange  = content extent - viewport extent  (max offset)
	//
	// The thumb occupies visible/content of the track and its position is
	// offset/scrollableRange — so at Value=0 the thumb starts at the
	// track's first pixel and at full scroll it ends at the track's LAST
	// pixel. Visual padding belongs to the track/theme, not the geometry.
	range_ := s.Max - s.Min
	if range_ <= 0 {
		canvas.Restore()
		return
	}
	scrollable := range_ - s.VisibleAmount
	if scrollable < 0 {
		scrollable = 0
	}

	trackLen := rect.Height
	if s.effOrientation() == Horizontal {
		trackLen = rect.Width
	}
	thumbSize := trackLen
	if s.VisibleAmount > 0 && s.VisibleAmount < range_ {
		thumbSize = trackLen * (s.VisibleAmount / range_)
		if thumbSize < 20 {
			thumbSize = 20
		}
	}

	percent := float32(0)
	if scrollable > 0 {
		percent = (s.Value - s.Min) / scrollable
		if percent < 0 {
			percent = 0
		}
		if percent > 1 {
			percent = 1
		}
	}
	var thumbRect graphics.Rect
	if s.effOrientation() == Vertical {
		thumbY := rect.Y + percent*(trackLen-thumbSize)
		thumbRect = graphics.NewRect(rect.X, thumbY, rect.Width, thumbSize)
	} else {
		thumbX := rect.X + percent*(trackLen-thumbSize)
		thumbRect = graphics.NewRect(thumbX, rect.Y, thumbSize, rect.Height)
	}
	if overlay {
		inset := s.ScrollBarTheme.ThumbInset
		if s.effOrientation() == Vertical {
			thumbRect.X += inset
			thumbRect.Width -= 2 * inset
		} else {
			thumbRect.Y += inset
			thumbRect.Height -= 2 * inset
		}
	}

	thumbPaint := graphics.NewPaint()
	switch {
	case disabled:
		thumbPaint.SetColor(theme.DefaultTheme.Disabled)
	case s.dragging:
		thumbPaint.SetColor(s.ScrollBarTheme.ThumbHover)
	default:
		thumbPaint.SetColor(s.ScrollBarTheme.ThumbInk)
	}
	if overlay || s.ScrollBarTheme.AutoHide {
		thumbPaint.SetColor(scaleAlpha(thumbPaint.Color(), fade))
	}
	radius := thumbRect.Width
	if thumbRect.Height < radius {
		radius = thumbRect.Height
	}
	radius /= 2
	canvas.DrawRoundedRect(thumbRect, radius, radius, thumbPaint)

	canvas.Restore()
}

func (s *ScrollBar) OnMouseDown(x, y float32, button int) bool {
	if !s.Enabled() {
		return false
	}
	frame := s.FrameRect()
	if x < 0 || x > frame.Width || y < 0 || y > frame.Height {
		return false
	}

	// Check if click is on thumb
	thumbRect := s.thumbRect()
	if s.effOrientation() == Vertical {
		if y >= thumbRect.Y && y <= thumbRect.Y+thumbRect.Height {
			s.dragging = true
			s.dragStartPos = y
			s.dragStartVal = s.Value
			return true
		}
		// Click on track - page scroll
		if y < thumbRect.Y {
			s.SetValue(s.Value - s.VisibleAmount)
		} else {
			s.SetValue(s.Value + s.VisibleAmount)
		}
		if s.OnValueChanged != nil {
			s.OnValueChanged(s.Value)
		}
		return true
	} else {
		if x >= thumbRect.X && x <= thumbRect.X+thumbRect.Width {
			s.dragging = true
			s.dragStartPos = x
			s.dragStartVal = s.Value
			return true
		}
		if x < thumbRect.X {
			s.SetValue(s.Value - s.VisibleAmount)
		} else {
			s.SetValue(s.Value + s.VisibleAmount)
		}
		if s.OnValueChanged != nil {
			s.OnValueChanged(s.Value)
		}
		return true
	}
}

func (s *ScrollBar) OnMouseUp(x, y float32, button int) bool {
	s.dragging = false
	s.MarkForRedraw()
	return true
}

func (s *ScrollBar) OnMouseMove(x, y float32) {
	if !s.Enabled() || !s.dragging {
		return
	}
	if s.effOrientation() == Vertical {
		s.setValueFromY(y)
	} else {
		s.setValueFromX(x)
	}
	s.MarkForRedraw()
}

func (s *ScrollBar) OnMouseWheel(x, y float32, dx, dy float32) bool {
	if !s.Enabled() || dy == 0 {
		return false
	}
	if s.effOrientation() == Vertical {
		s.SetValue(s.Value - dy*20)
	} else {
		s.SetValue(s.Value - dx*20)
	}
	return true
}

func (s *ScrollBar) thumbRect() graphics.Rect {
	frame := s.FrameRect()
	range_ := s.Max - s.Min
	if range_ <= 0 {
		return graphics.Rect{}
	}
	scrollable := range_ - s.VisibleAmount
	if scrollable < 0 {
		scrollable = 0
	}
	trackLen := frame.Height
	if s.effOrientation() == Horizontal {
		trackLen = frame.Width
	}
	thumbSize := float32(30)
	if s.VisibleAmount > 0 && s.VisibleAmount < range_ {
		thumbSize = trackLen * (s.VisibleAmount / range_)
		if thumbSize < 20 {
			thumbSize = 20
		}
	}
	percent := float32(0)
	if scrollable > 0 {
		percent = (s.Value - s.Min) / scrollable
		if percent < 0 {
			percent = 0
		}
		if percent > 1 {
			percent = 1
		}
	}
	if s.effOrientation() == Vertical {
		thumbY := percent * (trackLen - thumbSize)
		return graphics.NewRect(0, thumbY, frame.Width, thumbSize)
	}
	thumbX := percent * (trackLen - thumbSize)
	return graphics.NewRect(thumbX, 0, thumbSize, frame.Height)
}

func (s *ScrollBar) setValueFromY(y float32) {
	frame := s.FrameRect()
	range_ := s.Max - s.Min
	scrollable := range_ - s.VisibleAmount
	if scrollable < 0 {
		scrollable = 0
	}
	thumbSize := s.thumbRect().Height
	travel := frame.Height - thumbSize
	if travel > 0 && scrollable > 0 {
		percent := (y - thumbSize/2) / travel
		if percent < 0 {
			percent = 0
		}
		if percent > 1 {
			percent = 1
		}
		newValue := s.Min + percent*scrollable
		if newValue != s.Value {
			s.Value = newValue
			if s.OnValueChanged != nil {
				s.OnValueChanged(s.Value)
			}
		}
	}
}

func (s *ScrollBar) setValueFromX(x float32) {
	frame := s.FrameRect()
	range_ := s.Max - s.Min
	scrollable := range_ - s.VisibleAmount
	if scrollable < 0 {
		scrollable = 0
	}
	thumbSize := s.thumbRect().Width
	travel := frame.Width - thumbSize
	if travel > 0 && scrollable > 0 {
		percent := (x - thumbSize/2) / travel
		if percent < 0 {
			percent = 0
		}
		if percent > 1 {
			percent = 1
		}
		newValue := s.Min + percent*scrollable
		if newValue != s.Value {
			s.Value = newValue
			if s.OnValueChanged != nil {
				s.OnValueChanged(s.Value)
			}
		}
	}
}

func (s *ScrollBar) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "value":
		if f, ok := value.(float64); ok {
			s.SetValue(float32(f))
		}
	case "minimum":
		if f, ok := value.(float64); ok {
			s.SetMin(float32(f))
		}
	case "maximum":
		if f, ok := value.(float64); ok {
			s.SetMax(float32(f))
		}
	case "orientation":
		if sval, ok := value.(string); ok {
			if sval == "horizontal" || sval == "Horizontal" {
				s.SetOrientation(Horizontal)
			} else {
				s.SetOrientation(Vertical)
			}
		}
	}
}

// scaleAlpha multiplies a colour's alpha by a 0..1 factor (overlay
// scrollbar fade).
func scaleAlpha(c graphics.Color, f float32) graphics.Color {
	if f >= 1 {
		return c
	}
	a := uint32(float32((c>>24)&0xFF)*f + 0.5)
	return (c & 0x00FFFFFF) | graphics.Color(a)<<24
}
