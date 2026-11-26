package widgets

import (
	"time"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/theme"
)

type StatusBar struct {
	meui.BaseWidget
	*theme.StatusBarTheme
	Text     string
	Sections []string
	OnResize func()
	// tempMessage overrides Text/Sections until it expires (QStatusBar.showMessage)
	tempMessage string
	tempTimer   *time.Timer
	// Widgets rendered right-aligned (like addPermanentWidget)
	Widgets      []meui.Widget
	resizing     bool
	startRootX   float32
	startRootY   float32
	startWinW    float32
	startWinH    float32
	textCache    *graphics.Text
	textCacheVer uint64
}

func NewStatusBar() *StatusBar {
	return &StatusBar{
		BaseWidget:     *meui.NewBaseWidget(),
		StatusBarTheme: theme.DefaultStatusBarTheme,
	}
}

func (s *StatusBar) SetText(text string) *StatusBar {
	s.Text = text
	s.textCache = nil
	s.MarkForRedraw()
	return s
}

func (s *StatusBar) SetSections(sections []string) *StatusBar {
	s.Sections = sections
	s.MarkForRedraw()
	return s
}

func (s *StatusBar) SetOnResize(handler func()) *StatusBar {
	s.OnResize = handler
	return s
}

// ShowMessage displays a temporary message for timeoutMs, then restores the
// normal text (like QStatusBar.showMessage).
func (s *StatusBar) ShowMessage(message string, timeoutMs int) *StatusBar {
	s.tempMessage = message
	s.textCache = nil
	if s.tempTimer != nil {
		s.tempTimer.Stop()
	}
	if timeoutMs > 0 {
		s.tempTimer = time.AfterFunc(time.Duration(timeoutMs)*time.Millisecond, func() {
			s.tempMessage = ""
			s.textCache = nil
			s.MarkForRedraw()
		})
	}
	s.MarkForRedraw()
	return s
}

// ClearMessage removes the temporary message immediately.
func (s *StatusBar) ClearMessage() *StatusBar {
	s.tempMessage = ""
	if s.tempTimer != nil {
		s.tempTimer.Stop()
		s.tempTimer = nil
	}
	s.textCache = nil
	s.MarkForRedraw()
	return s
}

// AddWidget appends a widget rendered right-aligned before the size grip.
func (s *StatusBar) AddWidget(w meui.Widget) *StatusBar {
	s.Widgets = append(s.Widgets, w)
	w.SetParent(s)
	s.MarkForRedraw()
	return s
}

func (s *StatusBar) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !s.Visible() {
		return
	}

	canvas.Save()

	disabled := !s.Enabled()

	bgPaint := graphics.NewPaint()
	if disabled {
		bgPaint.SetColor(theme.DefaultTheme.DisabledBg)
	} else {
		bgPaint.SetColor(s.StatusBarTheme.BackgroundInk)
	}
	canvas.DrawRect(rect, bgPaint)

	topPaint := graphics.NewPaint()
	topPaint.SetStyle(paintstyle.Stroke)
	topPaint.SetStrokeWidth(theme.BorderStrokeWidth)
	if disabled {
		topPaint.SetColor(theme.DefaultTheme.DisabledBorder)
	} else {
		topPaint.SetColor(s.StatusBarTheme.EdgeInk)
	}
	canvas.DrawLine(rect.X, rect.Y, rect.X+rect.Width, rect.Y, topPaint)

	font := s.StatusBarTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	ascent := font.Baseline()
	fontHeight := font.LineHeight()
	textY := rect.Y + (rect.Height-fontHeight)/2 + ascent

	textPaint := graphics.NewPaint()
	if disabled {
		textPaint.SetColor(theme.DefaultTheme.DisabledText)
	} else {
		textPaint.SetColor(s.StatusBarTheme.TextInk)
	}

	display := s.Text
	if s.tempMessage != "" {
		display = s.tempMessage
	}

	if len(s.Sections) > 0 && s.tempMessage == "" {
		sectionW := rect.Width / float32(len(s.Sections))
		for i, sec := range s.Sections {
			x := rect.X + float32(i)*sectionW + 4
			DrawText(canvas, sec, x, textY, font, textPaint)
			if i < len(s.Sections)-1 {
				sepPaint := graphics.NewPaint()
				sepPaint.SetStyle(paintstyle.Stroke)
				sepPaint.SetStrokeWidth(theme.SeparatorStrokeWidth)
				sepPaint.SetColor(s.StatusBarTheme.EdgeInk)
				sepX := rect.X + float32(i+1)*sectionW
				canvas.DrawLine(sepX, rect.Y+2, sepX, rect.Y+rect.Height-2, sepPaint)
			}
		}
	} else if display != "" {
		ver := theme.ThemeVersion()
		if s.textCache == nil || s.textCacheVer != ver {
			s.textCache = graphics.NewText(display, &graphics.TextDecoration{Font: font, OnBackgroundInk: textPaint.Color()})
			s.textCacheVer = ver
		}
		s.textCache.Draw(canvas, rect.X+4, textY)
	}

	// Paint right-aligned widgets
	gripSize := float32(16)
	widgetRight := rect.X + rect.Width - gripSize
	for i := len(s.Widgets) - 1; i >= 0; i-- {
		w := s.Widgets[i]
		if w == nil {
			continue
		}
		wf := w.FrameRect()
		if wf.Width <= 0 {
			continue
		}
		x := widgetRight - wf.Width
		w.SetFrameRect(graphics.NewRect(x, rect.Y+1, wf.Width, rect.Height-2))
		canvas.Save()
		canvas.Translate(x, rect.Y+1)
		w.Paint(canvas, graphics.NewRect(0, 0, wf.Width, rect.Height-2))
		canvas.Restore()
		widgetRight = x - 4
	}

	s.drawSizeGrip(canvas, rect)

	canvas.Restore()
}

func (s *StatusBar) drawSizeGrip(canvas *graphics.Canvas, rect graphics.Rect) {
	gripSize := float32(16)
	gripX := rect.X + rect.Width - gripSize
	gripY := rect.Y + rect.Height - gripSize

	gripPaint := graphics.NewPaint()
	gripPaint.SetColor(s.StatusBarTheme.GripInk)

	for i := 0; i < 3; i++ {
		offset := float32(i) * 4
		canvas.DrawLine(gripX+offset, gripY+gripSize-2, gripX+gripSize-2, gripY+offset, gripPaint)
	}
}

func (s *StatusBar) OnMouseDown(x, y float32, button int) bool {
	if !s.Enabled() {
		return false
	}
	frame := s.FrameRect()
	gripSize := float32(16)
	if x >= frame.Width-gripSize && y >= frame.Height-gripSize {
		pt := s.PointToRoot(x, y)
		s.startRootX = pt.X
		s.startRootY = pt.Y
		if win := s.Window(); win != nil {
			wf := win.FrameRect()
			s.startWinW = wf.Width
			s.startWinH = wf.Height
		}
		s.resizing = true
		return true
	}
	return false
}

func (s *StatusBar) OnMouseUp(x, y float32, button int) bool {
	s.resizing = false
	return false
}

func (s *StatusBar) OnMouseMove(x, y float32) {
	if !s.Enabled() {
		return
	}
	frame := s.FrameRect()
	gripSize := float32(16)
	if s.resizing {
		if win := s.Window(); win != nil {
			win.ValidateLayout()
			pt := s.PointToRoot(x, y)
			dx := pt.X - s.startRootX
			dy := pt.Y - s.startRootY
			newW := s.startWinW + dx
			newH := s.startWinH + dy
			if newW < 200 {
				newW = 200
			}
			if newH < 100 {
				newH = 100
			}
			win.SetWindowSize(newW, newH)
		}
		if s.OnResize != nil {
			s.OnResize()
		}
	} else {
		inGrip := x >= frame.Width-gripSize && y >= frame.Height-gripSize
		oldHovered := s.BaseWidget.Hovered()
		s.BaseWidget.SetHovered(inGrip)
		if oldHovered != s.BaseWidget.Hovered() {
			s.MarkForRedraw()
		}
	}
}

func (s *StatusBar) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "text":
		if str, ok := value.(string); ok {
			s.SetText(str)
		}
	}
}
