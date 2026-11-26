package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

type Switch struct {
	meui.BaseWidget
	*theme.SwitchTheme
	Checked      bool
	Text         string // optional label rendered beside the track
	OnToggled    func(bool)
	Animating    bool
	animProgress float32
	textCache    *graphics.Text
	textCacheVer uint64
}

func NewSwitch() *Switch {
	s := &Switch{
		BaseWidget:  *meui.NewBaseWidget(),
		SwitchTheme: theme.DefaultSwitchTheme,
	}
	// Platform toggle size: track height from the metrics (default
	// 20px), width twice the height.
	h := theme.DefaultTheme.Metrics.SwitchH
	if h <= 0 {
		h = 20
	}
	s.SetHeight(h)
	s.SetWidth(h * 2)
	return s
}

// SetText sets an optional label rendered to the right of the track.
func (s *Switch) SetText(text string) *Switch {
	s.Text = text
	s.textCache = nil
	s.MarkForRedraw()
	return s
}

func (s *Switch) SetChecked(checked bool) *Switch {
	if s.Checked == checked {
		return s
	}
	s.Checked = checked
	s.MarkForRedraw()
	if s.OnToggled != nil {
		s.OnToggled(s.Checked)
	}
	return s
}

func (s *Switch) SetOnToggled(handler func(bool)) *Switch {
	s.OnToggled = handler
	return s
}

func (s *Switch) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !s.Visible() {
		return
	}

	canvas.Save()

	disabled := !s.Enabled()

	trackH := rect.Height
	trackW := trackH * 2
	var trackX, trackY float32
	if s.Text == "" {
		trackX = rect.X + (rect.Width-trackW)/2
		trackY = rect.Y
	} else {
		// With a label: track on the left, text fills the remaining space.
		trackX = rect.X
		trackY = rect.Y + (rect.Height-trackH)/2
	}
	trackRect := graphics.NewRect(trackX, trackY, trackW, trackH)
	radius := trackH / 2

	trackPaint := graphics.NewPaint()
	if disabled {
		trackPaint.SetColor(theme.DefaultTheme.DisabledBg)
	} else if s.Checked {
		trackPaint.SetColor(s.SwitchTheme.TrackOn)
	} else {
		trackPaint.SetColor(s.SwitchTheme.TrackOff)
	}
	canvas.DrawRoundedRect(trackRect, radius, radius, trackPaint)

	thumbInk := s.SwitchTheme.ThumbInk
	if !s.Checked {
		thumbInk = s.SwitchTheme.ThumbOff
	}
	if disabled {
		thumbInk = theme.DefaultTheme.Disabled
	}
	thumbSize := trackH - 4
	thumbRadius := thumbSize / 2
	var thumbX float32
	if s.Checked {
		thumbX = trackX + trackW - thumbSize - 2
	} else {
		thumbX = trackX + 2
	}
	thumbY := trackY + 2
	thumbRect := graphics.NewRect(thumbX, thumbY, thumbSize, thumbSize)

	thumbPaint := graphics.NewPaint()
	thumbPaint.SetColor(thumbInk)
	canvas.DrawRoundedRect(thumbRect, thumbRadius, thumbRadius, thumbPaint)

	// Optional label beside the track
	if s.Text != "" {
		font := theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
		textX := trackX + trackW + 8
		ascent := font.Baseline()
		textY := rect.Y + (rect.Height-ascent)/2 + ascent
		textPaint := graphics.NewPaint()
		if disabled {
			textPaint.SetColor(theme.DefaultTheme.DisabledText)
		} else {
			textPaint.SetColor(theme.DefaultTheme.TextPrimary)
		}
		ver := theme.ThemeVersion()
		if s.textCache == nil || s.textCacheVer != ver {
			s.textCache = graphics.NewText(s.Text, &graphics.TextDecoration{Font: font, OnBackgroundInk: textPaint.Color()})
			s.textCacheVer = ver
		}
		s.textCache.Draw(canvas, textX, textY)
	}

	canvas.Restore()
}

func (s *Switch) OnMouseDown(x, y float32, button int) bool {
	if !s.Enabled() {
		return false
	}
	frame := s.FrameRect()
	if x >= 0 && x <= frame.Width && y >= 0 && y <= frame.Height {
		s.SetChecked(!s.Checked)
		s.MarkForRedraw()
		return true
	}
	return false
}

func (s *Switch) OnMouseMove(x, y float32) {
	if !s.Enabled() {
		return
	}
	frame := s.FrameRect()
	oldHovered := s.BaseWidget.Hovered()
	s.BaseWidget.SetHovered(x >= 0 && x <= frame.Width && y >= 0 && y <= frame.Height)
	if oldHovered != s.BaseWidget.Hovered() {
		s.MarkForRedraw()
	}
}

func (s *Switch) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if !s.Enabled() {
		return false
	}
	if keyCode == platform.KeySpace || keyCode == platform.KeyReturn {
		s.SetChecked(!s.Checked)
		return true
	}
	return false
}

func (s *Switch) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "checked":
		if b, ok := value.(bool); ok {
			s.SetChecked(b)
		}
	case "text":
		if str, ok := value.(string); ok {
			s.SetText(str)
		}
	}
}

func (s *Switch) Focusable() bool {
	return s.Enabled()
}
