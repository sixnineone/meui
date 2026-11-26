// Package mobileglesdevice is the public entry point to the built-in
// renderer-backed graphics.Device.
//
// It is the "one real backend" half of the seam's coverage: the toolkit paints
// through graphics.Device, and this is what a real window supplies. The seam's
// test double (graphics.TestDevice) is the other half, so a widget can only
// rely on behaviour both provide.
//
// The implementation body lives in graphics itself, because it needs the
// seam's unexported paint/font/path unwrapping and because offscreen rendering
// must build a device from inside the seam (importing this package from
// graphics would be an import cycle). This package is the stable name callers
// use; the body moves here at P3, once the seam no longer needs to unwrap
// renderer objects.
package mobileglesdevice

import (
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/mobilegles"
)

// New creates a renderer-backed device with its own surface.
func New(width, height float32) graphics.Device {
	return graphics.NewRendererDevice(width, height)
}

// Wrap adapts an existing renderer canvas, which the platform layer owns.
func Wrap(c *mobilegles.Canvas) graphics.Device {
	return graphics.WrapRendererCanvas(c)
}
