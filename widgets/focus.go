package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
)

// drawFocusIndicator paints the active platform's focus indicator (theme
// Effects.Focus) around or inside r.
//
// The construction lives ONCE, in meui.DrawFocusRing: the core's Panel is a focus
// target too now (a clickable box takes Enter/Space), and two painters for one
// indicator is how a Button and a card end up with different-looking focus rings.
func drawFocusIndicator(c *graphics.Canvas, r graphics.Rect, radius float32) {
	meui.DrawFocusRing(c, r, radius)
}
