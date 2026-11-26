package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

// drawShadow paints a soft drop shadow behind r.
//
// The painter lives ONCE, in meui.DrawShadow, because the core's Panel draws its
// own shadow from the same tokens — a card in QML and a popup in the toolkit must
// not come from two different blur simulations.
func drawShadow(c *graphics.Canvas, r graphics.Rect, radius float32, spec theme.ShadowSpec) {
	meui.DrawShadow(c, r, radius, spec)
}

// drawPopupShadow / drawMenuShadow read the active theme's shadow
// tokens so platforms differ (Fluent soft 16px, Adwaita 20px, Metro
// deliberately none).
func drawPopupShadow(c *graphics.Canvas, r graphics.Rect, radius float32) {
	drawShadow(c, r, radius, theme.DefaultTheme.Effects.PopupShadow)
}

func drawMenuShadow(c *graphics.Canvas, r graphics.Rect, radius float32) {
	drawShadow(c, r, radius, theme.DefaultTheme.Effects.MenuShadow)
}
