package widgets

import (
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/theme"
)

// DrawCheckBox draws a checkbox box + optional checkmark at (x, y) with the given size.
// Uses theme.DefaultCheckBoxTheme for colors unless overridden.
func DrawCheckBox(canvas *graphics.Canvas, x, y, size float32, checked, enabled bool) {
	t := theme.DefaultCheckBoxTheme

	boxRect := graphics.NewRect(x, y, size, size)
	bg := graphics.NewPaint()
	if checked && enabled {
		bg.SetColor(t.SelectionInk)
	} else {
		bg.SetColor(t.BackgroundInk)
	}
	canvas.DrawRoundedRect(boxRect, t.CornerRadius, t.CornerRadius, bg)

	bd := graphics.NewPaint()
	bd.SetStyle(paintstyle.Stroke)
	bd.SetStrokeWidth(1)
	if checked && enabled {
		bd.SetColor(t.SelectionInk)
	} else {
		bd.SetColor(t.EdgeInk)
	}
	canvas.DrawRoundedRect(boxRect, t.CornerRadius, t.CornerRadius, bd)

	if checked {
		chk := graphics.NewPaint()
		chk.SetStyle(paintstyle.Stroke)
		chk.SetStrokeWidth(theme.CheckStrokeWidth)
		if !enabled {
			chk.SetColor(theme.DefaultTheme.DisabledText)
		} else {
			// On-accent ink: the box is filled with SelectionInk, so a
			// SelectionInk tick was invisible (accent on accent). Mirrors
			// the standalone CheckBox widget.
			chk.SetColor(theme.DefaultTheme.OnAccentColor())
		}
		m := size * 0.2
		canvas.DrawLine(x+m, y+size*0.55, x+size*0.45, y+size*0.8, chk)
		canvas.DrawLine(x+size*0.45, y+size*0.8, x+size-m, y+size*0.25, chk)
	}
}

func DrawText(canvas *graphics.Canvas, text string, x, y float32, font graphics.Font, paint *graphics.Paint) {
	if text == "" || font == nil {
		return
	}
	t := graphics.NewText(text, &graphics.TextDecoration{Font: font, OnBackgroundInk: paint.Color()})
	t.Draw(canvas, x, y)
}
