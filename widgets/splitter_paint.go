package widgets

import (
	"math"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/theme"
)

// paintSplitter draws the toolkit's splitter look for both SplitView and
// StackPanel, so the two never drift apart again (they did: StackPanel kept
// filling the band with a lightened background and drawing full-length grip
// lines long after SplitView was fixed, which is why the owner could see the
// vertical dividers and not the horizontal ones).
//
//	band      the handle band, in canvas coordinates
//	vertical  the divider LINE runs vertically (the band is a tall strip)
//	hot       the pointer is on the handle, or it is being dragged
//
// The divider takes the hover ink while hot, or always when the theme sets
// HoverColorAlwaysOn (the IntelliJ importer does — see StackPanelTheme).
//
// The line is one pixel, snapped to a whole coordinate: a half-pixel offset
// antialiases it into two grey rows instead of one theme-coloured one.
func paintSplitter(canvas *graphics.Canvas, band graphics.Rect, vertical, hot bool, t *theme.StackPanelTheme) {
	if t == nil {
		t = theme.DefaultStackPanelTheme
	}
	if t == nil || band.Width <= 0 || band.Height <= 0 {
		return
	}

	// The band fill is opt-in: a theme that lightens its panel background into
	// BackgroundInk otherwise paints a bright strip along every split, which
	// reads as a defect on a dark theme. Transparent = the line alone.
	if alpha := uint32(t.BackgroundInk) >> 24; alpha > 0 {
		bg := graphics.NewPaint()
		bg.SetColor(t.BackgroundInk)
		canvas.DrawRect(band, bg)
	}

	ink := t.SplitterInk
	if (hot || t.HoverColorAlwaysOn) && t.SplitterHoverInk != 0 {
		ink = t.SplitterHoverInk
	}
	if ink != 0 {
		line := graphics.NewPaint()
		line.SetColor(ink)
		if vertical {
			cx := float32(math.Round(float64(band.X + band.Width/2)))
			canvas.DrawRect(graphics.NewRect(cx, band.Y, 1, band.Height), line)
		} else {
			cy := float32(math.Round(float64(band.Y + band.Height/2)))
			canvas.DrawRect(graphics.NewRect(band.X, cy, band.Width, 1), line)
		}
	}

	// Grip marks are optional decoration, off by default: the split is the
	// single divider line above. A theme that sets an opaque GripInk gets
	// three short dashes centred on the handle — never full-length lines,
	// which read as a band and defeat the point of a one-line split. The
	// dashes are clipped to the band so they cannot spill onto the panels,
	// and their spacing shrinks with a narrow band instead of vanishing.
	if alpha := uint32(t.GripInk) >> 24; alpha > 0 {
		grip := graphics.NewPaint()
		grip.SetColor(t.GripInk)
		dash := t.GripSize * 2
		if dash < 6 {
			dash = 6
		}
		off := float32(4)
		if vertical && band.Width < 14 {
			off = band.Width / 4
		}
		if !vertical && band.Height < 14 {
			off = band.Height / 4
		}
		canvas.Save()
		canvas.ClipRect(band, pathop.Intersect, false)
		if vertical {
			cx := float32(math.Round(float64(band.X + band.Width/2)))
			cy := band.Y + band.Height/2
			for _, dx := range []float32{-off, 0, off} {
				canvas.DrawLine(cx+dx, cy-dash/2, cx+dx, cy+dash/2, grip)
			}
		} else {
			cy := float32(math.Round(float64(band.Y + band.Height/2)))
			cx := band.X + band.Width/2
			for _, dy := range []float32{-off, 0, off} {
				canvas.DrawLine(cx-dash/2, cy+dy, cx+dash/2, cy+dy, grip)
			}
		}
		canvas.Restore()
	}
}
