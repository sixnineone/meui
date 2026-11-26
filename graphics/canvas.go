package graphics

import (
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"log"
)

// Canvas is the drawing facade the toolkit paints through. It holds a Device
// and owns no backend state of its own, so swapping the Device swaps the
// renderer without touching a single widget call site.
type Canvas struct {
	dev Device
}

// NewCanvas wraps a Device in the seam facade.
func NewCanvas(dev Device) *Canvas {
	return &Canvas{dev: dev}
}

// Device returns the backend this canvas draws through (may be nil).
func (c *Canvas) Device() Device {
	if c == nil {
		return nil
	}
	return c.dev
}

// Width and Height report the device surface size.
func (c *Canvas) Width() float32 {
	if c == nil || c.dev == nil {
		return 0
	}
	return c.dev.Width()
}
func (c *Canvas) Height() float32 {
	if c == nil || c.dev == nil {
		return 0
	}
	return c.dev.Height()
}

// --- surface --------------------------------------------------------------

func (c *Canvas) Clear(col uint32) { c.dev.Clear(col) }
func (c *Canvas) ClearWithViewport(col uint32, x, y, w, h int32) {
	c.dev.ClearWithViewport(col, x, y, w, h)
}

// --- transform and clip state --------------------------------------------

func (c *Canvas) Save()                  { c.dev.Save() }
func (c *Canvas) Restore()               { c.dev.Restore() }
func (c *Canvas) RestoreToCount(n int)   { c.dev.RestoreToCount(n) }
func (c *Canvas) Translate(x, y float32) { c.dev.Translate(x, y) }
func (c *Canvas) Scale(x, y float32)     { c.dev.Scale(x, y) }
func (c *Canvas) Rotate(degrees float32) { c.dev.Rotate(degrees) }
func (c *Canvas) Skew(kx, ky float32)    { c.dev.Skew(kx, ky) }
func (c *Canvas) ResetClip()             { c.dev.ResetClip() }
func (c *Canvas) ClipDepth() int         { return c.dev.ClipDepth() }

// ClipRect: meui passes (rect, pathop, antialias). The rect is in the
// canvas's LOCAL (user) space, exactly like every other draw call, so it is
// transformed by the current matrix before becoming a device clip.
// (The shim previously mistook the antialias flag for a "local" toggle and
// skipped the transform, clipping widgets at their untranslated positions.)
// WithClip draws with a clip and restores the previous one, whatever fn does.
//
// It is the transaction form of ClipRect, and it exists because the manual form
// leaks: the historic TextField bug was a paint that pushed a clip and never
// popped it, poisoning everything painted after it in that frame. A defer cannot
// be forgotten, so prefer this to Save/ClipRect/Restore.
func (c *Canvas) WithClip(rect Rect, fn func(*Canvas)) {
	c.Save()
	c.ClipRect(rect, pathop.Intersect, true)
	defer c.Restore()
	fn(c)
}

// BackdropBlurrer is implemented by devices that can blur what they have already
// painted — the frosted-glass operation. The GL device does it through its blur FBO,
// the software device with the same 7-tap kernel (see softdevice_blur.go).
type BackdropBlurrer interface {
	BlurBackdrop(r Rect, cornerRadius float32) bool
}

// BlurBackdrop blurs the painted surface inside r and composites it back, masked to a
// rounded rect of cornerRadius (0 is a plain rectangle). It reports false when the
// device has no blur path, so a widget can fall back to a plain translucent fill
// instead of drawing nothing.
func (c *Canvas) BlurBackdrop(r Rect, cornerRadius float32) bool {
	if c == nil || c.dev == nil {
		return false
	}
	if b, ok := c.dev.(BackdropBlurrer); ok {
		return b.BlurBackdrop(r, cornerRadius)
	}
	return false
}

// ClipBounds is the clip in force, in device space.
func (c *Canvas) ClipBounds() Rect { return c.dev.ClipBounds() }

// IsClipEmpty reports a clip that admits nothing — which is NOT the same as a clip
// covering the whole surface, and is what a widget checks before drawing at all.
func (c *Canvas) IsClipEmpty() bool { return c.dev.IsClipEmpty() }

// QuickRejectRect reports whether a rect lies entirely outside the clip, so a
// widget drawing many items can skip the ones that cannot be seen. It is a
// conservative test: false only means "not provably invisible".
func (c *Canvas) QuickRejectRect(r Rect) bool { return c.dev.QuickRejectRect(r) }

func (c *Canvas) ClipRect(rect Rect, _ pathop.Enum, _ bool) {
	rx, ry, rw, rh := c.dev.TransformRect(rect.X, rect.Y, rect.Width, rect.Height)
	c.dev.ClipRect(Rect{X: rx, Y: ry, Width: rw, Height: rh})
}

// ClipPath intersects the clip with the interior of path, transformed by the
// current matrix like every other local-space draw call. Exact for convex
// outlines; a concave outline clips to its bounding rectangle (a superset) and
// returns false.
func (c *Canvas) ClipPath(path *Path) bool { return c.dev.ClipPath(path) }

func (c *Canvas) SaveAndRestore(fn func()) {
	if c == nil || fn == nil {
		return
	}
	depth := c.dev.Save()
	fn()
	c.dev.RestoreToCount(depth)
}

// Origin returns the current canvas translation (matrix tx, ty).
func (c *Canvas) Origin() (float32, float32) {
	m := c.dev.GetMatrix()
	return m[2], m[5]
}

// --- drawing --------------------------------------------------------------

// DrawRect: meui signature (rect Rect, paint *Paint).
func (c *Canvas) DrawRect(rect Rect, paint *Paint) {
	c.dev.DrawRect(rect.X, rect.Y, rect.Width, rect.Height, paint)
}

// DrawRoundedRect: meui signature (rect Rect, rx, ry float32, paint *Paint).
func (c *Canvas) DrawRoundedRect(rect Rect, rx, ry float32, paint *Paint) {
	c.dev.DrawRoundedRect(rect, rx, ry, paint)
}

// DrawCircle: meui signature (cx, cy, r float32, paint *Paint).
func (c *Canvas) DrawCircle(cx, cy, r float32, paint *Paint) {
	c.dev.DrawCircle(cx, cy, r, paint)
}

// DrawLine: meui signature (x1, y1, x2, y2 float32, paint *Paint).
func (c *Canvas) DrawLine(x1, y1, x2, y2 float32, paint *Paint) {
	c.dev.DrawLine(x1, y1, x2, y2, paint)
}

// DrawDashedLine: meui signature (x1, y1, x2, y2, dash, phase, paint).
func (c *Canvas) DrawDashedLine(x1, y1, x2, y2 float32, dash []float32, phase float32, paint *Paint) {
	c.dev.DrawDashedLine(x1, y1, x2, y2, dash, phase, paint)
}

// DrawOval: meui signature (rect Rect, paint *Paint).
func (c *Canvas) DrawOval(rect Rect, paint *Paint) {
	c.dev.DrawOval(rect.X, rect.Y, rect.Width, rect.Height, paint)
}

// DrawArc: meui signature (rect Rect, start, sweep float32, paint *Paint).
func (c *Canvas) DrawArc(rect Rect, start, sweep float32, paint *Paint) {
	c.dev.DrawArc(rect.X, rect.Y, rect.Width, rect.Height, start, sweep, paint)
}

// DropShadow: meui signature (rect Rect, paint *Paint, offsetX, offsetY, radius float32).
func (c *Canvas) DropShadow(rect Rect, paint *Paint, offsetX, offsetY, radius float32) {
	c.dev.DropShadow(rect, paint, offsetX, offsetY, radius)
}

func (c *Canvas) DrawPath(path *Path, paint *Paint) {
	c.dev.DrawPath(path, paint)
}

// DrawSimpleString: meui signature (s string, x, y float32, font Font, paint *Paint).
func (c *Canvas) DrawSimpleString(s string, x, y float32, font Font, paint *Paint) {
	c.dev.DrawSimpleString(s, x, y, font, paint)
}

// DrawText: meui signature (s string, x, y float32, font Font, paint *Paint).
func (c *Canvas) DrawText(text string, x, y float32, font Font, p *Paint) {
	if font == nil {
		log.Println("graphics DrawText: font is nil")
		return
	}
	if p == nil {
		log.Println("graphics DrawText: paint is nil")
		return
	}
	c.dev.DrawText(text, x, y, font, p)
}

// DrawImageRectInRect: approximation, ignores srcRect and sampling.
func (c *Canvas) DrawImageRectInRect(img *Image, _, dst Rect, _ *Image, paint *Paint) {
	c.dev.DrawImage(img, dst, paint)
}

// DrawImageInRect accepts the meui (img, rect, sampling, paint) signature.
// sampling is ignored.
func (c *Canvas) DrawImageInRect(img *Image, rect Rect, _ *Image, paint *Paint) {
	c.dev.DrawImage(img, rect, paint)
}
