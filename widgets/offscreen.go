package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
)

// windowedWidget is the promoted accessor every widget gets from its embedded
// base: the window it is attached to, if any.
type windowedWidget interface{ Window() *meui.Window }

// offscreenDevice resolves the device for offscreen work that starts outside a
// paint pass (drag images), through the widget's owning window. It returns nil
// when the widget is unattached; the offscreen render then reports failure and
// the caller falls back, exactly as it already did when no renderer context
// was available.
func offscreenDevice(w windowedWidget) graphics.Device {
	if w == nil {
		return nil
	}
	win := w.Window()
	if win == nil {
		return nil
	}
	return win.Device()
}
