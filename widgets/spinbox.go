package widgets

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

type SpinBox struct {
	meui.BaseWidget
	*theme.SpinBoxTheme
	Value    float32
	Minimum  float32
	Maximum  float32
	Step     float32
	Focused  bool
	OnChange func(float32)
	// Prefix/Suffix are rendered around the number (like QSpinBox). They are
	// not part of the edit buffer.
	Prefix         string
	Suffix         string
	Wrapping       bool // step past the range wraps around
	editMode       bool
	editBuffer     string
	cursorPos      int
	Decimals       int
	longPressing   bool
	longPressDir   int // 1=up, -1=down
	longPressTimer *time.Timer
	upArrowSVG     *meui.SVGDocument
	downArrowSVG   *meui.SVGDocument
	upPressed      bool // visual feedback for buttons
	downPressed    bool
}

func NewSpinBox() *SpinBox {
	sb := &SpinBox{
		BaseWidget:   *meui.NewBaseWidget(),
		SpinBoxTheme: theme.DefaultSpinBoxTheme,
		Minimum:      0,
		Maximum:      100,
		Value:        0,
		Step:         1,
		Decimals:     2,
	}
	sb.SetTabIndex(0)
	if doc, err := meui.LoadSVGByName("arrow_up"); err == nil {
		sb.upArrowSVG = doc
	}
	if doc, err := meui.LoadSVGByName("arrow_down"); err == nil {
		sb.downArrowSVG = doc
	}
	return sb
}

func (s *SpinBox) SetStep(step float32) *SpinBox {
	s.Step = step
	return s
}

func (s *SpinBox) SetValue(value float32) *SpinBox {
	s.Value = value
	if s.Value < s.Minimum {
		s.Value = s.Minimum
	}
	if s.Value > s.Maximum {
		s.Value = s.Maximum
	}
	// Keep the edit buffer in sync: while the spinbox holds focus (edit mode)
	// Paint renders editBuffer, not Value — without this, programmatic updates
	// (quick-set buttons, Refresh) changed Value but the display kept showing
	// the stale buffer.
	if s.editMode {
		s.editBuffer = fmt.Sprintf("%."+fmt.Sprintf("%d", s.Decimals)+"f", s.Value)
		if s.cursorPos > len(s.editBuffer) {
			s.cursorPos = len(s.editBuffer)
		}
	}
	s.MarkForRedraw()
	return s
}

func (s *SpinBox) SetOnChange(handler func(float32)) *SpinBox {
	s.OnChange = handler
	return s
}

func (s *SpinBox) SetDecimals(d int) *SpinBox {
	s.Decimals = d
	if s.Decimals < 0 {
		s.Decimals = 0
	}
	s.MarkForRedraw()
	return s
}

// SetRange sets both bounds at once (like QSpinBox.setRange).
func (s *SpinBox) SetRange(min, max float32) *SpinBox {
	s.Minimum = min
	s.Maximum = max
	if s.Minimum > s.Maximum {
		s.Minimum, s.Maximum = s.Maximum, s.Minimum
	}
	s.SetValue(s.Value) // re-clamp
	return s
}

// SetPrefix sets text rendered before the value (e.g. "$").
func (s *SpinBox) SetPrefix(prefix string) *SpinBox {
	s.Prefix = prefix
	s.MarkForRedraw()
	return s
}

// SetSuffix sets text rendered after the value (e.g. " km/h").
func (s *SpinBox) SetSuffix(suffix string) *SpinBox {
	s.Suffix = suffix
	s.MarkForRedraw()
	return s
}

// SetWrapping makes stepping wrap around the range instead of clamping.
func (s *SpinBox) SetWrapping(wrap bool) *SpinBox {
	s.Wrapping = wrap
	return s
}

// GetText returns the formatted display text (including prefix/suffix).
func (s *SpinBox) GetText() string {
	return s.Prefix + fmt.Sprintf("%."+fmt.Sprintf("%d", s.Decimals)+"f", s.Value) + s.Suffix
}

func (s *SpinBox) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !s.Visible() {
		return
	}

	canvas.Save()

	disabled := !s.Enabled()

	padding := float32(5)
	innerRect := graphics.NewRect(rect.X+padding, rect.Y+padding, rect.Width-padding*2-30, rect.Height-padding*2)

	bgPaint := graphics.NewPaint()
	if disabled {
		bgPaint.SetColor(theme.DefaultTheme.DisabledBg)
	} else {
		bgPaint.SetColor(s.SpinBoxTheme.BackgroundInk)
	}
	canvas.DrawRoundedRect(rect, s.SpinBoxTheme.CornerRadius, s.SpinBoxTheme.CornerRadius, bgPaint)

	borderPaint := graphics.NewPaint()
	borderPaint.SetStyle(paintstyle.Stroke)
	borderPaint.SetStrokeWidth(theme.BorderStrokeWidth)
	if disabled {
		borderPaint.SetColor(theme.DefaultTheme.DisabledBorder)
	} else {
		borderPaint.SetColor(s.SpinBoxTheme.EdgeInk)
	}
	canvas.DrawRoundedRect(rect, s.SpinBoxTheme.CornerRadius, s.SpinBoxTheme.CornerRadius, borderPaint)

	font := s.SpinBoxTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	ascent := font.Baseline()
	fontHeight := font.LineHeight()

	text := s.Prefix + fmt.Sprintf("%."+fmt.Sprintf("%d", s.Decimals)+"f", s.Value) + s.Suffix
	if s.editMode {
		text = s.editBuffer
		if s.cursorPos < len(text) {
			text = text[:s.cursorPos] + "|" + text[s.cursorPos:]
		} else {
			text = text + "|"
		}
	}
	baseline := innerRect.Y + (innerRect.Height-fontHeight)/2 + ascent

	textPaint := graphics.NewPaint()
	if disabled {
		textPaint.SetColor(theme.DefaultTheme.DisabledText)
	} else {
		textPaint.SetColor(s.SpinBoxTheme.TextInk)
	}
	// FlatLaf centers the spinner value in the field; edit mode stays left.
	textX := innerRect.X + 1
	if !s.editMode {
		textX = innerRect.X + (innerRect.Width-font.SimpleWidth(text))/2
	}
	DrawText(canvas, text, textX, baseline, font, textPaint)

	btnWidth := float32(25)
	btnRect := graphics.NewRect(rect.X+rect.Width-btnWidth, rect.Y, btnWidth, rect.Height)

	upBtnRect := graphics.NewRect(btnRect.X, btnRect.Y, btnWidth, rect.Height/2)
	downBtnRect := graphics.NewRect(btnRect.X, btnRect.Y+rect.Height/2, btnWidth, rect.Height/2)

	btnPaint := graphics.NewPaint()
	if disabled {
		btnPaint.SetColor(theme.DefaultTheme.DisabledBg)
	} else if s.upPressed {
		btnPaint.SetColor(s.SpinBoxTheme.ButtonPressedInk)
	} else {
		btnPaint.SetColor(s.SpinBoxTheme.ButtonInk)
	}
	canvas.DrawRect(upBtnRect, btnPaint)

	btnPaint2 := graphics.NewPaint()
	if disabled {
		btnPaint2.SetColor(theme.DefaultTheme.DisabledBg)
	} else if s.downPressed {
		btnPaint2.SetColor(s.SpinBoxTheme.ButtonPressedInk)
	} else {
		btnPaint2.SetColor(s.SpinBoxTheme.ButtonInk)
	}
	canvas.DrawRect(downBtnRect, btnPaint2)

	btnBorderPaint := graphics.NewPaint()
	btnBorderPaint.SetStyle(paintstyle.Stroke)
	btnBorderPaint.SetStrokeWidth(theme.BorderStrokeWidth)
	if disabled {
		btnBorderPaint.SetColor(theme.DefaultTheme.DisabledBorder)
	} else {
		btnBorderPaint.SetColor(s.SpinBoxTheme.EdgeInk)
	}
	canvas.DrawRect(upBtnRect, btnBorderPaint)
	canvas.DrawRect(downBtnRect, btnBorderPaint)

	arrowPaint := graphics.NewPaint()
	if disabled {
		arrowPaint.SetColor(theme.DefaultTheme.DisabledText)
	} else {
		arrowPaint.SetColor(s.SpinBoxTheme.ArrowInk)
	}
	// Draw arrows — same pattern as ComboBox (DrawSVG at 12px, glyph fallback)
	arrowSize := float32(12)
	upArrowX := upBtnRect.X + (upBtnRect.Width-arrowSize)/2
	upArrowY := upBtnRect.Y + (upBtnRect.Height-arrowSize)/2
	downArrowX := downBtnRect.X + (downBtnRect.Width-arrowSize)/2
	downArrowY := downBtnRect.Y + (downBtnRect.Height-arrowSize)/2
	if s.upArrowSVG != nil {
		meui.DrawSVG(canvas, s.upArrowSVG, upArrowX, upArrowY, arrowSize, arrowSize)
	} else {
		font := s.SpinBoxTheme.Font
		if font == nil {
			font = theme.DefaultTheme.BaseFont
			if font == nil {
				font = graphics.SystemFont
			}
		}
		ascent := font.Baseline()
		at := graphics.NewText("▲", &graphics.TextDecoration{Font: font, OnBackgroundInk: arrowPaint.Color()})
		at.Draw(canvas, upArrowX, upArrowY+arrowSize/2+ascent/2)
	}
	if s.downArrowSVG != nil {
		meui.DrawSVG(canvas, s.downArrowSVG, downArrowX, downArrowY, arrowSize, arrowSize)
	} else {
		font := s.SpinBoxTheme.Font
		if font == nil {
			font = theme.DefaultTheme.BaseFont
			if font == nil {
				font = graphics.SystemFont
			}
		}
		ascent := font.Baseline()
		dt := graphics.NewText("▼", &graphics.TextDecoration{Font: font, OnBackgroundInk: arrowPaint.Color()})
		dt.Draw(canvas, downArrowX, downArrowY+arrowSize/2+ascent/2)
	}

	// Focus ring
	if s.BaseWidget.Focused() {
		focusPaint := graphics.NewPaint()
		focusPaint.SetStyle(paintstyle.Stroke)
		focusPaint.SetStrokeWidth(theme.FocusStrokeWidth)
		focusPaint.SetColor(theme.DefaultTheme.BorderFocused)
		sw := theme.FocusStrokeWidth
		focusRect := graphics.NewRect(rect.X+sw/2, rect.Y+sw/2, rect.Width-sw, rect.Height-sw)
		canvas.DrawRect(focusRect, focusPaint)
	}

	canvas.Restore()
}

func (s *SpinBox) OnMouseDown(x, y float32, button int) bool {
	if !s.Enabled() {
		return false
	}
	btnWidth := float32(25)
	btnX := s.Width() - btnWidth

	if x >= btnX {
		if y < s.Height()/2 {
			s.increment()
			s.longPressDir = 1
			s.upPressed = true
			s.downPressed = false
		} else {
			s.decrement()
			s.longPressDir = -1
			s.downPressed = true
			s.upPressed = false
		}
		// Update editBuffer so Paint shows the new value (OnFocus may have captured a stale value)
		s.editBuffer = fmt.Sprintf("%."+fmt.Sprintf("%d", s.Decimals)+"f", s.Value)
		s.longPressing = true
		s.startLongPress()
		return true
	}
	s.stopLongPress()
	s.Focused = true
	s.upPressed = false
	s.downPressed = false
	s.MarkForRedraw()
	return true
}

func (s *SpinBox) increment() {
	s.Value += s.Step
	if s.Value > s.Maximum {
		if s.Wrapping {
			s.Value = s.Minimum
		} else {
			s.Value = s.Maximum
		}
	}
	if s.OnChange != nil {
		s.OnChange(s.Value)
	}
	s.MarkForRedraw()
}

func (s *SpinBox) decrement() {
	s.Value -= s.Step
	if s.Value < s.Minimum {
		if s.Wrapping {
			s.Value = s.Maximum
		} else {
			s.Value = s.Minimum
		}
	}
	if s.OnChange != nil {
		s.OnChange(s.Value)
	}
	s.MarkForRedraw()
}

func (s *SpinBox) OnMouseUp(x, y float32, button int) bool {
	s.stopLongPress()
	s.upPressed = false
	s.downPressed = false
	s.MarkForRedraw()
	return false
}

func (s *SpinBox) startLongPress() {
	s.stopLongPress()
	s.longPressTimer = time.AfterFunc(400*time.Millisecond, func() {
		if !s.longPressing {
			return
		}
		platform.InvokeTask(func() {
			if s.longPressDir == 1 {
				s.increment()
			} else {
				s.decrement()
			}
			s.startLongPress()
		})
	})
}

func (s *SpinBox) stopLongPress() {
	s.longPressing = false
	if s.longPressTimer != nil {
		s.longPressTimer.Stop()
		s.longPressTimer = nil
	}
}

// OnKeyRepeat is a key the user is HOLDING — Backspace over a word, an arrow along a row, a scrub.
// The window classifies an OS auto-repeat as a repeat rather than a fresh press (a widget without
// this method is asked once, deliberately), so this is what makes a held key work here.
func (s *SpinBox) OnKeyRepeat(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	return s.OnKeyDown(keyCode, mod)
}

func (s *SpinBox) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if !s.Enabled() || !s.Focused {
		return false
	}
	if s.editMode {
		switch keyCode {
		case platform.KeyReturn, platform.KeyNumPadEnter:
			s.commitEdit()
			return true
		case platform.KeyEscape:
			s.editMode = false
			s.editBuffer = ""
			s.cursorPos = 0
			s.MarkForRedraw()
			return true
		case platform.KeyBackspace:
			if s.cursorPos > 0 {
				s.editBuffer = s.editBuffer[:s.cursorPos-1] + s.editBuffer[s.cursorPos:]
				s.cursorPos--
				s.MarkForRedraw()
			}
			return true
		case platform.KeyDelete:
			if s.cursorPos < len(s.editBuffer) {
				s.editBuffer = s.editBuffer[:s.cursorPos] + s.editBuffer[s.cursorPos+1:]
				s.MarkForRedraw()
			}
			return true
		case platform.KeyLeft:
			if s.cursorPos > 0 {
				s.cursorPos--
				s.MarkForRedraw()
			}
			return true
		case platform.KeyRight:
			if s.cursorPos < len(s.editBuffer) {
				s.cursorPos++
				s.MarkForRedraw()
			}
			return true
		case platform.KeyHome:
			s.cursorPos = 0
			s.MarkForRedraw()
			return true
		case platform.KeyEnd:
			s.cursorPos = len(s.editBuffer)
			s.MarkForRedraw()
			return true
		}
	}
	switch keyCode {
	case platform.KeyUp:
		s.increment()
		return true
	case platform.KeyDown:
		s.decrement()
		return true
	case platform.KeyPageUp:
		s.Value += s.Step * 10
		if s.Value > s.Maximum {
			s.Value = s.Maximum
		}
		if s.OnChange != nil {
			s.OnChange(s.Value)
		}
		s.MarkForRedraw()
		return true
	case platform.KeyPageDown:
		s.Value -= s.Step * 10
		if s.Value < s.Minimum {
			s.Value = s.Minimum
		}
		if s.OnChange != nil {
			s.OnChange(s.Value)
		}
		s.MarkForRedraw()
		return true
	case platform.KeyHome:
		s.SetValue(s.Minimum)
		if s.OnChange != nil {
			s.OnChange(s.Value)
		}
		return true
	case platform.KeyEnd:
		s.SetValue(s.Maximum)
		if s.OnChange != nil {
			s.OnChange(s.Value)
		}
		return true
	}
	return false
}

func (s *SpinBox) OnFocus() {
	s.BaseWidget.OnFocus()
	s.Focused = true
	s.editMode = true
	s.editBuffer = fmt.Sprintf("%."+fmt.Sprintf("%d", s.Decimals)+"f", s.Value)
	s.cursorPos = len(s.editBuffer)
}

func (s *SpinBox) OnBlur() {
	s.BaseWidget.OnBlur()
	s.commitEdit()
	s.Focused = false
	s.editMode = false
}

func (s *SpinBox) OnCharacter(ch rune) {
	if !s.Enabled() || !s.Focused {
		return
	}
	if ch >= '0' && ch <= '9' || ch == '-' || ch == '.' {
		if ch == '-' && s.cursorPos != 0 {
			return
		}
		if ch == '.' && s.Decimals == 0 {
			return
		}
		if ch == '.' && strings.Contains(s.editBuffer, ".") {
			return
		}
		if ch == '-' && strings.Contains(s.editBuffer, "-") {
			return
		}
		s.editBuffer = s.editBuffer[:s.cursorPos] + string(ch) + s.editBuffer[s.cursorPos:]
		s.cursorPos++
		s.MarkForRedraw()
	}
}

func (s *SpinBox) commitEdit() {
	if !s.editMode {
		return
	}
	if val, err := strconv.ParseFloat(s.editBuffer, 32); err == nil {
		f := float32(val)
		if f >= s.Minimum && f <= s.Maximum {
			s.Value = f
			if s.OnChange != nil {
				s.OnChange(s.Value)
			}
		}
	}
	s.editMode = false
	s.editBuffer = ""
	s.cursorPos = 0
	s.MarkForRedraw()
}

func (s *SpinBox) SetBoundProperty(propName string, value interface{}) {
	toF := func(v interface{}) (float32, bool) {
		switch n := v.(type) {
		case float64:
			return float32(n), true
		case int:
			return float32(n), true
		case int64:
			return float32(n), true
		}
		return 0, false
	}
	switch propName {
	case "value":
		if f, ok := toF(value); ok {
			s.SetValue(f)
		}
	case "minimumValue", "min":
		if f, ok := toF(value); ok {
			s.Minimum = f
		}
	case "maximumValue", "max":
		if f, ok := toF(value); ok {
			s.Maximum = f
		}
	case "stepSize", "step", "stepValue":
		if f, ok := toF(value); ok {
			s.Step = f
		}
	case "decimals", "precision":
		if f, ok := toF(value); ok {
			s.SetDecimals(int(f))
		}
	case "prefix":
		if str, ok := value.(string); ok {
			s.SetPrefix(str)
		}
	case "suffix":
		if str, ok := value.(string); ok {
			s.SetSuffix(str)
		}
	case "wrapping", "wrap":
		if b, ok := value.(bool); ok {
			s.SetWrapping(b)
		}
	}
}

func (s *SpinBox) Focusable() bool {
	return s.Enabled()
}

func (s *SpinBox) Dispose() {
	s.stopLongPress()
	s.BaseWidget.Dispose()
}
