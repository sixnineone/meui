package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
	"time"
)

type Tooltip struct {
	meui.BaseWidget
	*theme.TooltipTheme
	Text         string
	Delay        time.Duration
	AutoDismiss  time.Duration
	visible      bool
	showTimer    *time.Timer
	hideTimer    *time.Timer
	targetWidget meui.Widget
	textCache    *graphics.Text
	textCacheVer uint64
}

func NewTooltip() *Tooltip {
	return &Tooltip{
		BaseWidget:   *meui.NewBaseWidget(),
		TooltipTheme: theme.DefaultTooltipTheme,
		Delay:        500 * time.Millisecond,
		AutoDismiss:  5 * time.Second,
	}
}

func (t *Tooltip) SetText(text string) *Tooltip {
	t.Text = text
	t.textCache = nil
	t.MarkForRedraw()
	return t
}

func (t *Tooltip) SetDelay(delay time.Duration) *Tooltip {
	t.Delay = delay
	return t
}

func (t *Tooltip) SetAutoDismiss(duration time.Duration) *Tooltip {
	t.AutoDismiss = duration
	return t
}

func (t *Tooltip) ShowFor(target meui.Widget) {
	t.targetWidget = target
	t.hideTimer = nil
	if t.showTimer != nil {
		t.showTimer.Stop()
	}
	if t.Text == "" {
		return
	}
	t.showTimer = time.AfterFunc(t.Delay, func() {
		t.visible = true
		platform.InvokeTask(func() { t.MarkForRedraw() })
		if t.AutoDismiss > 0 {
			t.hideTimer = time.AfterFunc(t.AutoDismiss, func() {
				t.visible = false
				platform.InvokeTask(func() { t.MarkForRedraw() })
			})
		}
	})
}

func (t *Tooltip) Hide() {
	if t.showTimer != nil {
		t.showTimer.Stop()
		t.showTimer = nil
	}
	if t.hideTimer != nil {
		t.hideTimer.Stop()
		t.hideTimer = nil
	}
	t.visible = false
	t.MarkForRedraw()
}

func (t *Tooltip) SetVisibleInternal(v bool) {
	t.visible = v
	t.MarkForRedraw()
}

func (t *Tooltip) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !t.Visible() || !t.visible || t.Text == "" {
		return
	}

	canvas.Save()

	font := t.TooltipTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}

	textWidth := font.SimpleWidth(t.Text)
	ascent := font.Baseline()
	fontHeight := font.LineHeight()

	padding := float32(6)
	w := textWidth + padding*2
	h := fontHeight + padding*2

	x := rect.X + (rect.Width-w)/2
	y := rect.Y + rect.Height - h - 4

	bgRect := graphics.NewRect(x, y, w, h)
	ttR := theme.Shape().Tooltip
	drawPopupShadow(canvas, bgRect, ttR)
	bgPaint := graphics.NewPaint()
	bgPaint.SetColor(t.TooltipTheme.BackgroundInk)
	canvas.DrawRoundedRect(bgRect, ttR, ttR, bgPaint)

	borderPaint := graphics.NewPaint()
	borderPaint.SetStyle(paintstyle.Stroke)
	borderPaint.SetStrokeWidth(theme.BorderStrokeWidth)
	borderPaint.SetColor(t.TooltipTheme.EdgeInk)
	canvas.DrawRoundedRect(bgRect, ttR, ttR, borderPaint)

	textPaint := graphics.NewPaint()
	textPaint.SetColor(t.TooltipTheme.TextInk)
	ver := theme.ThemeVersion()
	if t.textCache == nil || t.textCacheVer != ver {
		t.textCache = graphics.NewText(t.Text, &graphics.TextDecoration{Font: font, OnBackgroundInk: textPaint.Color()})
		t.textCacheVer = ver
	}
	t.textCache.Draw(canvas, x+padding, y+padding+ascent)

	canvas.Restore()
}

func (t *Tooltip) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "text":
		if s, ok := value.(string); ok {
			t.SetText(s)
		}
	case "visible":
		if b, ok := value.(bool); ok {
			t.visible = b
		}
	}
}

func (t *Tooltip) Dispose() {
	if t.showTimer != nil {
		t.showTimer.Stop()
	}
	if t.hideTimer != nil {
		t.hideTimer.Stop()
	}
	t.BaseWidget.Dispose()
}
