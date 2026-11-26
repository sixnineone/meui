package graphics

import (
	"math"

	"github.com/sixnineone/mobilegles"
)

// This file holds the built-in renderer-backed Device.
//
// Placement note: the implementation lives here rather than in
// internal/mobileglesdevice because it needs this package's unexported paint,
// font and path unwrapping, and because offscreen rendering
// (NewImageFromDrawing/CaptureOffscreen) must build a device from inside the
// seam — importing the backend package would be an import cycle.
// internal/mobileglesdevice is the public entry point callers use. When the
// seam stops needing to unwrap renderer objects (P3), the body moves there.

// cubicSegmentMax is the longest straight segment a flattened cubic may leave
// behind, in pixels. Smaller is smoother and costs more vertices; at 6 the
// staircase is under the width of a thick stroke's anti-aliasing.
const cubicSegmentMax = 6

// cubicSteps is how many straight segments approximate a cubic. The control
// polygon's length bounds the curve's own length from above, so it is the right
// quantity to divide; the result is clamped so a tiny curve still looks round and
// a very long one cannot flood the vertex batch.
func cubicSteps(p0, p1, p2, p3 Point) int {
	length := dist(p0, p1) + dist(p1, p2) + dist(p2, p3)
	steps := int(math.Ceil(float64(length / cubicSegmentMax)))
	if steps < 8 {
		steps = 8
	}
	if steps > 256 {
		steps = 256
	}
	return steps
}

func dist(a, b Point) float32 {
	dx, dy := b.X-a.X, b.Y-a.Y
	return float32(math.Sqrt(float64(dx*dx + dy*dy)))
}

// mobileglesDevice is a Device backed by a mobilegles canvas.
type mobileglesDevice struct {
	c *mobilegles.Canvas
}

// NewRendererDevice creates the built-in renderer-backed device with its own
// surface. Callers normally use internal/mobileglesdevice.New instead.
func NewRendererDevice(width, height float32) Device {
	return &mobileglesDevice{c: mobilegles.NewCanvas(width, height)}
}

// WrapRendererCanvas adapts an existing renderer canvas, which the platform
// layer owns. The parameter is opaque so the seam names no backend type; the
// value must be the built-in renderer's canvas. Callers normally use
// internal/mobileglesdevice.Wrap or meui-gl/sdl3 instead.
func WrapRendererCanvas(surface any) Device {
	c, _ := surface.(*mobilegles.Canvas)
	if c == nil {
		return nil
	}
	return &mobileglesDevice{c: c}
}

// Canvas returns the underlying renderer canvas.
func (d *mobileglesDevice) Canvas() *mobilegles.Canvas {
	if d == nil {
		return nil
	}
	return d.c
}

// --- surface --------------------------------------------------------------

func (d *mobileglesDevice) Width() float32   { return d.c.Width() }
func (d *mobileglesDevice) Height() float32  { return d.c.Height() }
func (d *mobileglesDevice) Clear(col uint32) { d.c.Clear(col) }
func (d *mobileglesDevice) ClearWithViewport(col uint32, x, y, w, h int32) {
	d.c.ClearWithViewport(col, x, y, w, h)
}

// --- transform and clip state --------------------------------------------

func (d *mobileglesDevice) Save() int                { return d.c.Save() }
func (d *mobileglesDevice) Restore()                 { d.c.Restore() }
func (d *mobileglesDevice) RestoreToCount(n int)     { d.c.RestoreToCount(n) }
func (d *mobileglesDevice) Translate(dx, dy float32) { d.c.Translate(dx, dy) }
func (d *mobileglesDevice) Scale(sx, sy float32)     { d.c.Scale(sx, sy) }
func (d *mobileglesDevice) Rotate(deg float32)       { d.c.Rotate(deg) }
func (d *mobileglesDevice) Skew(kx, ky float32)      { d.c.Skew(kx, ky) }

func (d *mobileglesDevice) GetMatrix() Matrix {
	return Matrix(d.c.GetMatrix())
}

func (d *mobileglesDevice) TransformRect(x, y, w, h float32) (float32, float32, float32, float32) {
	return d.c.TransformRect(x, y, w, h)
}

func (d *mobileglesDevice) ClipRect(r Rect) { d.c.ClipRect(mgRect(r)) }
func (d *mobileglesDevice) ClipBounds() Rect {
	r := d.c.ClipBounds()
	return Rect{X: r.X, Y: r.Y, Width: r.Width, Height: r.Height}
}

func (d *mobileglesDevice) IsClipEmpty() bool { return d.c.IsClipEmpty() }

func (d *mobileglesDevice) QuickRejectRect(r Rect) bool {
	return d.c.QuickRejectRect(mgRect(r))
}

func (d *mobileglesDevice) ResetClip()     { d.c.ResetClip() }
func (d *mobileglesDevice) ClipDepth() int { return d.c.ClipDepth() }

// --- drawing --------------------------------------------------------------

func (d *mobileglesDevice) DrawRect(x, y, w, h float32, p *Paint) {
	d.c.DrawRect(x, y, w, h, p.renderer())
}

func (d *mobileglesDevice) DrawRoundedRect(r Rect, rx, ry float32, p *Paint) {
	d.c.DrawRoundedRect(mgRect(r), rx, ry, p.renderer())
}

func (d *mobileglesDevice) DrawCircle(cx, cy, r float32, p *Paint) {
	d.c.DrawCircle(cx, cy, r, p.renderer())
}

func (d *mobileglesDevice) DrawOval(x, y, w, h float32, p *Paint) {
	d.c.DrawOval(x, y, w, h, p.renderer())
}

func (d *mobileglesDevice) DrawArc(x, y, w, h, start, sweep float32, p *Paint) {
	d.c.DrawArc(x, y, w, h, start, sweep, p.renderer())
}

func (d *mobileglesDevice) DrawLine(x1, y1, x2, y2 float32, p *Paint) {
	d.c.DrawLine(x1, y1, x2, y2, p.renderer())
}

// DrawPath replays the seam-owned path ops into a renderer path. Cubics are
// flattened here, in the device, so the seam's capability does not depend on
// the renderer growing a cubic primitive.
func (d *mobileglesDevice) ClipPath(path *Path) bool {
	if path == nil {
		return false
	}
	return d.c.ClipPath(mgPath(path))
}

// mgPath replays seam-owned path ops into a renderer path. Cubics are flattened
// here, in the device, so the seam's capability does not depend on the renderer
// growing a cubic primitive.
func mgPath(path *Path) *mobilegles.Path {
	if path == nil {
		return nil
	}
	mg := mobilegles.NewPath()
	var cur Point
	for _, op := range path.Ops() {
		switch op.Kind {
		case PathMoveTo:
			mg.MoveTo(mgPoint(op.P0))
			cur = op.P0
		case PathLineTo:
			mg.LineTo(mgPoint(op.P0))
			cur = op.P0
		case PathQuadTo:
			mg.QuadTo(mgPoint(op.P0), mgPoint(op.P1))
			cur = op.P1
		case PathCubicTo:
			// The step count has to follow the curve: a fixed 16 is invisible on
			// a short curve and a visible staircase on a long one, and the
			// renderer strokes each straight segment separately, so the steps
			// stay in the picture at any stroke width.
			steps := cubicSteps(cur, op.P0, op.P1, op.P2)
			for i := 1; i <= steps; i++ {
				t := float32(i) / float32(steps)
				mt := 1 - t
				x := mt*mt*mt*cur.X + 3*mt*mt*t*op.P0.X + 3*mt*t*t*op.P1.X + t*t*t*op.P2.X
				y := mt*mt*mt*cur.Y + 3*mt*mt*t*op.P0.Y + 3*mt*t*t*op.P1.Y + t*t*t*op.P2.Y
				mg.LineTo(mobilegles.Point{X: x, Y: y})
			}
			cur = op.P2
		case PathClose:
			mg.Close()
		}
	}
	return mg
}

func (d *mobileglesDevice) DrawPath(path *Path, p *Paint) {
	if path == nil {
		return
	}
	d.c.DrawPath(mgPath(path), p.renderer())
}

// BlurBackdrop copies the framebuffer region and feeds it through the renderer's
// separable blur FBO — the same kernel its drop shadows use, which is why the software
// device implements the identical 7 taps.
func (d *mobileglesDevice) BlurBackdrop(r Rect, cornerRadius float32) bool {
	tx, ty, tw, th := d.TransformRect(r.X, r.Y, r.Width, r.Height)
	if tw < 1 || th < 1 {
		return false
	}
	sw, sh := int32(d.c.Width()), int32(d.c.Height())
	if sw <= 0 || sh <= 0 {
		return false
	}
	// The batch must be on the framebuffer before it can be read.
	mobilegles.FlushBatches()
	screen := mobilegles.ReadScreenPixels(sw, sh)
	if screen == nil {
		return false
	}
	x0 := imax(int(tx), 0)
	y0 := imax(int(ty), 0)
	x1 := imin(int(tx+tw), int(sw))
	y1 := imin(int(ty+th), int(sh))
	if x1 <= x0 || y1 <= y0 {
		return false
	}
	w, h := x1-x0, y1-y0
	// The screen pixels are bottom-up and STRAIGHT alpha; the seam's image is
	// bottom-up and PREMULTIPLIED.
	pix := make([]uint32, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := screen[(int(sh)-1-(y0+y))*int(sw)+(x0+x)]
			a := (p >> 24) & 0xFF
			r8 := (p >> 16) & 0xFF
			g8 := (p >> 8) & 0xFF
			b8 := p & 0xFF
			if a != 0xFF {
				r8 = r8 * a / 255
				g8 = g8 * a / 255
				b8 = b8 * a / 255
			}
			pix[y*w+x] = uint32(a)<<24 | r8<<16 | g8<<8 | b8
		}
	}
	// Blur on the CPU, with the SAME kernel the software device uses, then draw the
	// result through the ordinary image path. The renderer's blur FBO exists for SHADOWS
	// (draw a shape into it, blur, composite); trying to reuse it to composite the
	// existing framebuffer left a giant triangle across the page, because the pass binds
	// its own program and attribute layout in the middle of the main batch. CPU here is
	// a few hundred microseconds for a card, exact, and needs no state juggling — at the
	// cost of a readback stall, which is why a frosted panel should be a few, not many.
	blurH(pix, w, h)
	blurV(pix, w, h)
	// The same silhouette the software device composites through, applied to the alpha
	// before upload: a rounded frosted corner is then identical in both renderers.
	scale := float32(1)
	if r.Width > 0 {
		scale = tw / r.Width
	}
	maskRoundedCorners(pix, w, h, cornerRadius*scale)
	img := NewImage(pix, int32(w), int32(h))
	d.c.DrawImageInRect(img.renderer(), mgRect(Rect{X: float32(x0), Y: float32(y0), Width: float32(w), Height: float32(h)}), nil)
	return true
}

func (d *mobileglesDevice) DrawImage(img *Image, r Rect, p *Paint) {
	d.c.DrawImageInRect(img.renderer(), mgRect(r), p.renderer())
}

func (d *mobileglesDevice) DrawText(text string, x, y float32, f Font, p *Paint) {
	mf := f.renderFont()
	if mf == nil || mf.Face() == nil {
		return
	}
	d.c.DrawText(text, x, y, mf, p.renderer())
}

func (d *mobileglesDevice) DrawSimpleString(text string, x, y float32, f Font, p *Paint) {
	d.c.DrawText(text, x, y, f.renderFont(), p.renderer())
}

func (d *mobileglesDevice) DropShadow(r Rect, p *Paint, dx, dy, radius float32) {
	d.c.DropShadow(mgRect(r), p.renderer(), dx, dy, radius)
}

// RenderOffscreen draws into a fresh offscreen surface backed by the renderer.
func (d *mobileglesDevice) RenderOffscreen(width, height int, draw func(*Canvas)) (*Image, error) {
	if width <= 0 || height <= 0 {
		return nil, errOffscreenFailed
	}
	pixels := mobilegles.RenderOffscreen(int32(width), int32(height), func() {
		off := &mobileglesDevice{c: mobilegles.NewCanvas(float32(width), float32(height))}
		if draw != nil {
			draw(NewCanvas(off))
		}
	})
	if pixels == nil {
		return nil, errOffscreenFailed
	}
	return NewImage(pixels, int32(width), int32(height)), nil
}

// DrawDashedLine strokes a dashed segment. The dash pattern alternates on/off
// starting with on, and repeats; phase offsets into it. There is no native
// dash primitive, so the segment is subdivided — which is exactly what makes
// the capability portable to any backend.
func (d *mobileglesDevice) DrawDashedLine(x1, y1, x2, y2 float32, dash []float32, phase float32, p *Paint) {
	if len(dash) == 0 {
		d.c.DrawLine(x1, y1, x2, y2, p.renderer())
		return
	}
	pattern := float32(0)
	for _, v := range dash {
		if v > 0 {
			pattern += v
		}
	}
	dx, dy := x2-x1, y2-y1
	length := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if pattern <= 0 || length == 0 {
		return
	}
	ux, uy := dx/length, dy/length

	// Normalise the phase into the pattern and find where it lands.
	off := float32(math.Mod(float64(phase), float64(pattern)))
	if off < 0 {
		off += pattern
	}
	i := 0
	for off >= dash[i] {
		off -= dash[i]
		i = (i + 1) % len(dash)
	}
	on := i%2 == 0

	dist := float32(0)
	for dist < length {
		step := dash[i] - off
		if step > length-dist {
			step = length - dist
		}
		if on && step > 0 {
			t0, t1 := dist, dist+step
			d.c.DrawLine(x1+ux*t0, y1+uy*t0, x1+ux*t1, y1+uy*t1, p.renderer())
		}
		dist += step
		off = 0
		i = (i + 1) % len(dash)
		on = !on
	}
}
