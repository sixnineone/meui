package widgets

import (
	"fmt"
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
	"time"
)

type ProgressBar struct {
	meui.BaseWidget
	*theme.ProgressBarTheme
	Value         float32
	Minimum       float32
	Maximum       float32
	Indeterminate bool
	ShowText      bool
	animOffset    float32
	animDirection int
	animTimer     *time.Timer
}

func NewProgressBar() *ProgressBar {
	p := &ProgressBar{
		BaseWidget:       *meui.NewBaseWidget(),
		ProgressBarTheme: theme.DefaultProgressBarTheme,
		Value:            0,
		Minimum:          0,
		Maximum:          100,
	}
	// A bar's natural height is its thickness (platform token).
	h := theme.DefaultTheme.Metrics.ProgressHeight
	if h <= 0 {
		h = 6
	}
	p.SetHeight(h)
	p.SetWidth(200)
	return p
}

func (p *ProgressBar) SetValue(value float32) *ProgressBar {
	p.Value = value
	if p.Value < p.Minimum {
		p.Value = p.Minimum
	}
	if p.Value > p.Maximum {
		p.Value = p.Maximum
	}
	p.MarkForRedraw()
	return p
}

func (p *ProgressBar) SetMinimum(min float32) *ProgressBar {
	p.Minimum = min
	if p.Value < p.Minimum {
		p.Value = p.Minimum
	}
	p.MarkForRedraw()
	return p
}

func (p *ProgressBar) SetMaximum(max float32) *ProgressBar {
	p.Maximum = max
	if p.Value > p.Maximum {
		p.Value = p.Maximum
	}
	p.MarkForRedraw()
	return p
}

func (p *ProgressBar) SetIndeterminate(ind bool) *ProgressBar {
	p.Indeterminate = ind
	if ind {
		p.startAnim()
	} else {
		p.stopAnim()
	}
	p.MarkForRedraw()
	return p
}

func (p *ProgressBar) SetShowText(show bool) *ProgressBar {
	p.ShowText = show
	p.MarkForRedraw()
	return p
}

func (p *ProgressBar) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !p.Visible() {
		return
	}

	canvas.Save()

	disabled := !p.Enabled()

	trackPaint := graphics.NewPaint()
	if disabled {
		trackPaint.SetColor(theme.DefaultTheme.DisabledBg)
	} else {
		trackPaint.SetColor(p.ProgressBarTheme.TrackInk)
	}
	canvas.DrawRoundedRect(rect, p.CornerRadius, p.CornerRadius, trackPaint)

	if p.Indeterminate {
		animWidth := rect.Width * 0.3
		animX := rect.X + p.animOffset*(rect.Width-animWidth)
		fillPaint := graphics.NewPaint()
		if disabled {
			fillPaint.SetColor(theme.DefaultTheme.Disabled)
		} else {
			fillPaint.SetColor(p.ProgressBarTheme.FillInk)
		}
		fillRect := graphics.NewRect(animX, rect.Y, animWidth, rect.Height)
		canvas.DrawRoundedRect(fillRect, p.CornerRadius, p.CornerRadius, fillPaint)
	} else if p.Maximum > p.Minimum {
		fraction := (p.Value - p.Minimum) / (p.Maximum - p.Minimum)
		if fraction > 0 {
			fillWidth := rect.Width * fraction
			fillPaint := graphics.NewPaint()
			if disabled {
				fillPaint.SetColor(theme.DefaultTheme.Disabled)
			} else {
				fillPaint.SetColor(p.ProgressBarTheme.FillInk)
			}
			fillRect := graphics.NewRect(rect.X, rect.Y, fillWidth, rect.Height)
			canvas.DrawRoundedRect(fillRect, p.CornerRadius, p.CornerRadius, fillPaint)
		}
	}

	borderPaint := graphics.NewPaint()
	borderPaint.SetStyle(paintstyle.Stroke)
	borderPaint.SetStrokeWidth(theme.BorderStrokeWidth)
	if disabled {
		borderPaint.SetColor(theme.DefaultTheme.DisabledBorder)
	} else {
		borderPaint.SetColor(p.ProgressBarTheme.EdgeInk)
	}
	canvas.DrawRoundedRect(rect, p.CornerRadius, p.CornerRadius, borderPaint)

	if p.ShowText && !p.Indeterminate && p.Maximum > p.Minimum {
		font := theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
		fraction := (p.Value - p.Minimum) / (p.Maximum - p.Minimum)
		text := fmt.Sprintf("%.0f%%", fraction*100)
		textWidth := font.SimpleWidth(text)
		ascent := font.Baseline()
		textX := rect.X + (rect.Width-textWidth)/2
		textY := rect.Y + (rect.Height-ascent)/2 + ascent

		textPaint := graphics.NewPaint()
		if !p.Enabled() {
			textPaint.SetColor(theme.DefaultTheme.DisabledText)
		} else {
			textPaint.SetColor(theme.DefaultButtonTheme.TextInk)
		}
		DrawText(canvas, text, textX, textY, font, textPaint)
	}

	canvas.Restore()
}

func (p *ProgressBar) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "value":
		if f, ok := value.(float64); ok {
			p.SetValue(float32(f))
		}
	case "minimumValue", "min":
		if f, ok := value.(float64); ok {
			p.SetMinimum(float32(f))
		}
	case "maximumValue", "max":
		if f, ok := value.(float64); ok {
			p.SetMaximum(float32(f))
		}
	case "indeterminate":
		if b, ok := value.(bool); ok {
			p.SetIndeterminate(b)
		}
	case "showText", "textVisible":
		if b, ok := value.(bool); ok {
			p.SetShowText(b)
		}
	}
}

func (p *ProgressBar) startAnim() {
	p.animDirection = 1
	p.animOffset = 0
	if p.animTimer != nil {
		p.animTimer.Stop()
	}
	p.animTimer = time.AfterFunc(30*time.Millisecond, func() {
		p.animOffset += 0.03 * float32(p.animDirection)
		if p.animOffset >= 1 {
			p.animDirection = -1
		} else if p.animOffset <= 0 {
			p.animDirection = 1
		}
		platform.InvokeTask(func() {
			p.MarkForRedraw()
			if p.Indeterminate {
				p.startAnim()
			}
		})
	})
}

func (p *ProgressBar) stopAnim() {
	if p.animTimer != nil {
		p.animTimer.Stop()
		p.animTimer = nil
	}
}

func (p *ProgressBar) Dispose() {
	p.stopAnim()
	p.BaseWidget.Dispose()
}
