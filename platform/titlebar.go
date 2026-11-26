package platform

import "github.com/sixnineone/meui/graphics"

// TitleBar is the toolkit's window-chrome contract: a strip the backend paints
// above the content and routes bar events to.
//
// It exists at this level because chrome is a *window* concern, not a widget
// one: the desktop backends create undecorated windows, so a title bar is the
// only thing that can move, maximize or resize them. A backend that draws no
// chrome (a mobile surface, the renderer-level window) simply does not
// implement SetTitleBar, and an app opts out with WindowOptions.NoChrome.
//
// The widgets package provides the standard implementation and registers it as
// the default, so an app that uses the toolkit's widgets gets working chrome
// without writing any of this by hand.
type TitleBar interface {
	// BarHeight is the strip height, in window pixels; the content is laid out
	// below it.
	BarHeight() float32
	// PaintBar draws the bar across the top of the window.
	PaintBar(c *graphics.Canvas, width float32)
	// Bar events receive window-relative coordinates within the strip.
	BarMouseDown(x, y float32, btn int) bool
	BarMouseUp(x, y float32, btn int) bool
	BarMouseMove(x, y float32)
	// BarDoubleClick handles a double press on the bar strip (btn 0 only). It
	// returns true when consumed — conventionally a maximize/restore toggle.
	BarDoubleClick(x, y float32) bool
	// FrameHit returns an SDL_HITTEST_* value (0 normal, 1 draggable, 2..10
	// resize zones) for a window-relative point, which is what lets the window
	// manager drag and resize a borderless window natively.
	FrameHit(x, y, w, h float32) uint32
}
