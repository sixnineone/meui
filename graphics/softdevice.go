package graphics

// A software Device: the same drawing calls as the GL backend, rasterised on the
// CPU with github.com/llgcode/draw2d.
//
// It exists because an offscreen render in the GPU backend mutates state shared
// by every window (one batch, one screen-size uniform, one FBO binding, one clip
// state), so a widget that caches itself offscreen corrupts whichever window is
// painted next. Rasterising in software touches no GL state at all, and because
// this satisfies the same Device interface the Canvas already draws through,
// callers need no second drawing path - RenderOffscreen below runs the SAME draw
// callback through a Canvas over this device.
//
// Transform and clip are implemented here rather than handed to draw2d: this
// version of draw2d exposes no clipping at all, and building paths in device
// space avoids depending on its matrix conventions.
//
// THE CLIP TRIMS. A shape that STRADDLES the clip is cut at the boundary, not
// drawn whole: `clipped` is only the fast path that rejects a shape entirely
// outside. It used to be the whole implementation, and a body crossing the
// boundary spilled past it — which is how candles escaped a chart's clipped
// price pane into the volume band and the time axis. Rectangles are trimmed
// exactly when the matrix is axis-aligned (no rotation or skew); a segment is
// cut with Liang-Barsky. For a ROTATED rect this backend still culls only,
// because the bounding-box intersection is a superset and drawing that would
// over-paint — the honest limit of a rectangle-based clip.
//
// `Pixels()` emits rows BOTTOM-UP, the seam's convention, because the GL
// framebuffer is read that way and NewImage documents its pixels the same.
// Indexing it top-down silently reads a vertically mirrored image.

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"

	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/graphics/enums/slant"
	"github.com/sixnineone/meui/graphics/enums/weight"

	"github.com/golang/freetype/truetype"
	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dimg"
)

// softDevice is a Device backed by an RGBA raster surface.
type softDevice struct {
	w, h   float32
	img    *image.RGBA
	gc     *draw2dimg.GraphicContext
	mat    Matrix
	stack  []softState
	clip   *Rect // nil = unclipped
	closed bool
	// clips tracks narrowed clips so Restore can UNDO what leaked outside them.
	// draw2d has no clip of its own (see the ClipRect comment), so the only way to
	// enforce one is to snapshot the surface when the clip narrows and put back
	// everything outside it when the scope ends. Only the offscreen device pays: GL
	// has a real scissor.
	clips []softClip
}

type softState struct {
	mat  Matrix
	clip *Rect
}

// softClip is one narrowed clip: the region it admits, the surface as it was just
// before the clip took effect, and the save-stack depth that owns it.
type softClip struct {
	r     Rect
	snap  []uint8
	depth int
}

var _ Device = (*softDevice)(nil)

// softFont caches the face the software device draws with, so a graph of a
// thousand text draws loads it once.
var softFont *truetype.Font

// NewSoftDevice returns a software device of the given size. The pixels are
// white-transparent until something clears them.
func NewSoftDevice(w, h int) *softDevice {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	d := &softDevice{
		w:   float32(w),
		h:   float32(h),
		img: image.NewRGBA(image.Rect(0, 0, w, h)),
		mat: NewMatrix(),
	}
	d.gc = draw2dimg.NewGraphicContext(d.img)
	// draw2d sizes text in POINTS and scales by DPI/72; the seam's Font.Size is in
	// PIXELS. Its default DPI is 92, so every string came out 92/72 = 1.278x too
	// large — invisible in a screenshot of a paragraph, obvious the moment a 30px
	// heading overflowed a 472px box and measured 37px of ink. Pinning the DPI to 72
	// makes one point one pixel, which is the seam's contract.
	d.gc.SetDPI(72)
	registerSoftFont(d.gc)
	return d
}

// softFontData is the face the software device draws text with.
var softFontData = draw2d.FontData{Name: "seam", Family: draw2d.FontFamilySans, Style: draw2d.FontStyleNormal}

// registerSoftFont gives draw2d a real face. Left alone it resolves its default
// family "luxisr" inside a folder relative to the MODULE ("../resource/font"),
// which exists in the source tree and nowhere in a running app — and draw2d logs
// that miss for EVERY string drawn, so a chart flooded the log with one line per
// label. A registered face needs no folder lookup at all.
//
// The face comes from systemFontFace, which discovers the distribution's own
// layout and falls back to the face embedded in this binary. That matters here
// twice over: the old list was Debian's paths, so on any other layout this
// function found nothing, registered nothing, and handed every string to draw2d's
// folder lookup.
func registerSoftFont(gc *draw2dimg.GraphicContext) {
	if softFont == nil {
		softFont = systemFontFace()
	}
	if softFont == nil {
		return
	}
	draw2d.RegisterFont(softFontData, softFont)
	gc.SetFontData(softFontData)
}

// Image returns the rasterised surface.
func (d *softDevice) Image() *image.RGBA { return d.img }

// Pixels repacks the surface into the seam's a<<24|r<<16|g<<8|b order.
func (d *softDevice) Pixels() []uint32 {
	// Rows are emitted BOTTOM-UP, which is the seam's convention: NewImage's
	// pixels come from ReadScreenPixels, documented as "bottom-up as GL delivers
	// them", and the GL backend's framebuffer is read the same way. draw2d's
	// surface is top-down, so a chart cached here would otherwise be drawn
	// vertically flipped - oscillators (laid out at the bottom) appearing above
	// the candles, with the live layer's hit regions no longer lining up.
	out := make([]uint32, 0, d.img.Rect.Dx()*d.img.Rect.Dy())
	for y := d.img.Rect.Dy() - 1; y >= 0; y-- {
		for x := 0; x < d.img.Rect.Dx(); x++ {
			i := d.img.PixOffset(x, y)
			r := uint32(d.img.Pix[i+0])
			g := uint32(d.img.Pix[i+1])
			b := uint32(d.img.Pix[i+2])
			a := uint32(d.img.Pix[i+3])
			out = append(out, a<<24|r<<16|g<<8|b)
		}
	}
	return out
}

// ── Surface ─────────────────────────────────────────────────────────

func (d *softDevice) Width() float32  { return d.w }
func (d *softDevice) Height() float32 { return d.h }

func (d *softDevice) Clear(col uint32) {
	d.ClearWithViewport(col, 0, 0, int32(d.w), int32(d.h))
}

func (d *softDevice) ClearWithViewport(col uint32, x, y, w, h int32) {
	// A frame boundary: the seam-image conversion cache is keyed by *Image pointer
	// and Go reuses addresses, so a fresh image could otherwise be drawn with a
	// dead one's pixels. Clearing per frame keeps the reuse that matters (the same
	// image drawn several times in a frame) and drops the rest.
	clear(seamImageCache)
	r, g, b, a := unpackColor(col)
	cl := color.RGBA{R: r, G: g, B: b, A: a}
	for yy := int(y); yy < int(y+h); yy++ {
		if yy < 0 || yy >= d.img.Rect.Dy() {
			continue
		}
		for xx := int(x); xx < int(x+w); xx++ {
			if xx < 0 || xx >= d.img.Rect.Dx() {
				continue
			}
			d.img.SetRGBA(xx, yy, cl)
		}
	}
}

// ── Transform and clip state ────────────────────────────────────────

func (d *softDevice) Save() int {
	d.stack = append(d.stack, softState{mat: d.mat, clip: d.clip})
	return len(d.stack)
}

func (d *softDevice) Restore() {
	if n := len(d.stack); n > 0 {
		s := d.stack[n-1]
		d.stack = d.stack[:n-1]
		d.mat, d.clip = s.mat, s.clip
		d.releaseClips()
	}
}

func (d *softDevice) RestoreToCount(n int) {
	for len(d.stack) > n {
		d.Restore()
	}
}

// The transform helpers live in the renderer package, not the seam, so the
// matrices are built here in the same row-major [sx,kx,tx, ky,sy,ty, 0,0,1]
// layout the Matrix type documents.

func (d *softDevice) Translate(dx, dy float32) {
	d.mat = softMul(d.mat, Matrix{1, 0, dx, 0, 1, dy, 0, 0, 1})
}

func (d *softDevice) Scale(sx, sy float32) {
	d.mat = softMul(d.mat, Matrix{sx, 0, 0, 0, sy, 0, 0, 0, 1})
}

func (d *softDevice) Rotate(deg float32) {
	rad := float64(deg) * math.Pi / 180
	c, sn := float32(math.Cos(rad)), float32(math.Sin(rad))
	d.mat = softMul(d.mat, Matrix{c, -sn, 0, sn, c, 0, 0, 0, 1})
}

func (d *softDevice) Skew(kx, ky float32) {
	tx := float32(math.Tan(float64(kx) * math.Pi / 180))
	ty := float32(math.Tan(float64(ky) * math.Pi / 180))
	d.mat = softMul(d.mat, Matrix{1, tx, 0, ty, 1, 0, 0, 0, 1})
}
func (d *softDevice) GetMatrix() Matrix { return d.mat }

func (d *softDevice) TransformRect(x, y, w, h float32) (float32, float32, float32, float32) {
	p0 := softApply(d.mat, NewPoint(x, y))
	p1 := softApply(d.mat, NewPoint(x+w, y))
	p2 := softApply(d.mat, NewPoint(x, y+h))
	p3 := softApply(d.mat, NewPoint(x+w, y+h))
	minX, maxX := min4(p0.X, p1.X, p2.X, p3.X), max4(p0.X, p1.X, p2.X, p3.X)
	minY, maxY := min4(p0.Y, p1.Y, p2.Y, p3.Y), max4(p0.Y, p1.Y, p2.Y, p3.Y)
	return minX, minY, maxX - minX, maxY - minY
}

func (d *softDevice) ClipRect(r Rect) {
	if d.img == nil {
		return
	}
	var next Rect
	if d.clip == nil {
		next = r
	} else {
		next = Rect{
			X:      maxf(d.clip.X, r.X),
			Y:      maxf(d.clip.Y, r.Y),
			Width:  minf(d.clip.X+d.clip.Width, r.X+r.Width) - maxf(d.clip.X, r.X),
			Height: minf(d.clip.Y+d.clip.Height, r.Y+r.Height) - maxf(d.clip.Y, r.Y),
		}
	}
	if d.clip != nil && next == *d.clip {
		return // no narrowing: nothing new to protect
	}
	// Snapshot BEFORE anything can leak. Only pixels OUTSIDE next are restored, so
	// drawing that legitimately happened before the clip (and inside it) survives.
	snap := make([]uint8, len(d.img.Pix))
	copy(snap, d.img.Pix)
	d.clips = append(d.clips, softClip{r: next, snap: snap, depth: len(d.stack)})
	d.clip = &next
}

// releaseClips undoes every clip taken inside the scope that just ended: the pixels
// outside each clip are put back, and the enclosing clip is restored.
func (d *softDevice) releaseClips() {
	for len(d.clips) > 0 {
		top := d.clips[len(d.clips)-1]
		if top.depth <= len(d.stack) {
			break
		}
		d.restoreOutside(top)
		d.clips = d.clips[:len(d.clips)-1]
	}
	if n := len(d.clips); n > 0 {
		c := d.clips[n-1].r
		d.clip = &c
	} else {
		d.clip = nil
	}
}

// restoreOutside copies the snapshot back over every pixel the narrowed clip did not
// admit — what the device drew past the clip, in other words.
func (d *softDevice) restoreOutside(top softClip) {
	if d.img == nil || len(top.snap) != len(d.img.Pix) {
		return
	}
	b := d.img.Bounds()
	stride := d.img.Stride
	x0 := int(top.r.X) - b.Min.X
	y0 := int(top.r.Y) - b.Min.Y
	x1 := x0 + int(top.r.Width)
	y1 := y0 + int(top.r.Height)
	if x1 <= x0 || y1 <= y0 {
		// An empty clip: the whole surface goes back.
		copy(d.img.Pix, top.snap)
		return
	}
	for y := 0; y < b.Dy(); y++ {
		rowOff := y * stride
		if y >= y0 && y < y1 {
			// Left and right of the clip on this row.
			if x0 > 0 {
				l := x0
				if l > b.Dx() {
					l = b.Dx()
				}
				copy(d.img.Pix[rowOff:rowOff+l*4], top.snap[rowOff:rowOff+l*4])
			}
			if x1 < b.Dx() {
				r := x1
				if r < 0 {
					r = 0
				}
				copy(d.img.Pix[rowOff+r*4:rowOff+b.Dx()*4], top.snap[rowOff+r*4:rowOff+b.Dx()*4])
			}
			continue
		}
		copy(d.img.Pix[rowOff:rowOff+b.Dx()*4], top.snap[rowOff:rowOff+b.Dx()*4])
	}
}

// ClipPath clips to the path's bounding rectangle and reports false: the Device
// contract allows a bounding-rectangle fallback (a superset, so nothing is
// hidden) for an outline this backend cannot intersect exactly.
func (d *softDevice) ClipPath(path *Path) bool {
	if path == nil {
		return true
	}
	first := true
	var minX, minY, maxX, maxY float32
	for _, op := range path.Ops() {
		for _, pt := range []Point{op.P0, op.P1, op.P2} {
			// Only ops that actually carry the point are considered below; a
			// zero Point would shrink the box, so each kind contributes the
			// points its comment says it uses.
			_ = pt
		}
		pts := opPoints(op)
		for _, pt := range pts {
			tp := softApply(d.mat, pt)
			if first {
				minX, maxX, minY, maxY = tp.X, tp.X, tp.Y, tp.Y
				first = false
				continue
			}
			minX, maxX = minf(minX, tp.X), maxf(maxX, tp.X)
			minY, maxY = minf(minY, tp.Y), maxf(maxY, tp.Y)
		}
	}
	if first {
		return true
	}
	d.ClipRect(NewRect(minX, minY, maxX-minX, maxY-minY))
	return false
}

// opPoints returns the points a path operation contributes to its bounding box,
// per PathOp's documented meaning.
func opPoints(op PathOp) []Point {
	switch op.Kind {
	case PathMoveTo, PathLineTo:
		return []Point{op.P0}
	case PathQuadTo:
		return []Point{op.P0, op.P1}
	case PathCubicTo:
		return []Point{op.P0, op.P1, op.P2}
	}
	return nil
}

func (d *softDevice) ResetClip() { d.clip = nil }

func (d *softDevice) ClipBounds() Rect {
	if d.clip == nil {
		return Rect{X: 0, Y: 0, Width: d.w, Height: d.h}
	}
	return *d.clip
}

func (d *softDevice) IsClipEmpty() bool {
	return d.clip != nil && (d.clip.Width <= 0 || d.clip.Height <= 0)
}

func (d *softDevice) QuickRejectRect(r Rect) bool { return quickReject(d.ClipBounds(), r) }
func (d *softDevice) ClipDepth() int {
	n := 0
	for _, s := range d.stack {
		if s.clip != nil {
			n++
		}
	}
	if d.clip != nil {
		n++
	}
	return n
}

// ── Drawing ─────────────────────────────────────────────────────────

// clipped reports whether a device-space rectangle is entirely outside the clip.
func (d *softDevice) clipped(r Rect) bool {
	if d.clip == nil {
		return false
	}
	c := *d.clip
	if c.Width <= 0 || c.Height <= 0 {
		return true
	}
	return !r.Intersects(c)
}

func (d *softDevice) setPaint(p *Paint, fill bool) {
	if p == nil {
		white := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
		d.gc.SetFillColor(white)
		d.gc.SetStrokeColor(white)
		return
	}
	r, g, b, a := unpackColor(uint32(p.Color()))
	alpha := p.GlobalAlpha()
	if alpha <= 0 {
		alpha = 1
	}
	if alpha > 1 {
		alpha = 1
	}
	// freetype's raster painter expects a PREMULTIPLIED color: it stores
	// c.RGBA() and composites `dst*(1-a) + src`. Handing it a STRAIGHT color with
	// an alpha made every translucent fill overflow the byte — a 0.8-alpha mint
	// blob over cream came out (15,29,9) instead of (226,238,226), which is how
	// every shadow and hover wash in the offscreen renderer looked like a stain.
	af := float32(a) / 255 * alpha
	col := color.RGBA{
		R: uint8(float32(r) * af),
		G: uint8(float32(g) * af),
		B: uint8(float32(b) * af),
		A: uint8(af * 255),
	}
	d.gc.SetFillColor(col)
	d.gc.SetStrokeColor(col)
	d.gc.SetLineWidth(float64(p.StrokeWidth()))
}

// stroke reports whether the paint should be stroked rather than filled. The
// seam's style constants are not public, so this uses the width, which is zero
// for every fill-only paint.
// stroke reports whether the paint asks for a stroke. This used the stroke WIDTH, which is
// wrong: a fill paint carries a default width, so every panel background was drawn as an
// outline - the interior stayed the clear colour. The seam has style constants
// (enums/paintstyle) and they are what decides.
func stroke(p *Paint) bool { return p != nil && p.Style() == paintstyle.Stroke }

// fillStroke is the third style: fill, then outline the same path.
func fillStroke(p *Paint) bool { return p != nil && p.Style() == paintstyle.FillStroke }

func (d *softDevice) rectPath(r Rect) {
	pts := []Point{
		softApply(d.mat, NewPoint(r.X, r.Y)),
		softApply(d.mat, NewPoint(r.X+r.Width, r.Y)),
		softApply(d.mat, NewPoint(r.X+r.Width, r.Y+r.Height)),
		softApply(d.mat, NewPoint(r.X, r.Y+r.Height)),
	}
	d.gc.BeginPath()
	d.gc.MoveTo(float64(pts[0].X), float64(pts[0].Y))
	for _, pt := range pts[1:] {
		d.gc.LineTo(float64(pt.X), float64(pt.Y))
	}
	d.gc.Close()
}

func (d *softDevice) DrawRect(x, y, w, h float32, p *Paint) {
	// MEUI_SOFT_TRACE: every draw the SOFTWARE device receives. The device is the one place
	// both the direct-draw known-positive and the panel's own paint must pass through, so this
	// shows whether a panel background reaches the device at all - the open contradiction of
	// P5fx (a direct draw lands, the panel's does not, same canvas and same device).
	if os.Getenv("MEUI_SOFT_TRACE") != "" {
		col := uint32(0)
		if p != nil {
			col = uint32(p.Color())
		}
		ga := float32(-1)
		if p != nil {
			ga = p.GlobalAlpha()
		}
		fmt.Printf("[soft] DrawRect (%.0f,%.0f %.0fx%.0f) color=%#08x globalAlpha=%.3f\n", x, y, w, h, col, ga)
	}
	dr := Rect{X: x, Y: y, Width: w, Height: h}
	tx, ty, tw, th := d.TransformRect(x, y, w, h)
	if d.clipped(NewRect(tx, ty, tw, th)) {
		return
	}
	if p != nil && p.Gradient().Kind != GradientNone && d.axisAligned() {
		d.fillRoundedGradient(dr, 0, 0, p)
		return
	}
	d.setPaint(p, true)
	// A rect that STRADDLES the clip must be TRIMMED, not drawn whole. `clipped` above only
	// rejects a rect entirely outside, so a body crossing the boundary used to spill past it —
	// which is how candles escaped the price pane into the volume band and the time axis.
	if trimmed, ok := d.trimToClip(NewRect(tx, ty, tw, th)); ok && d.axisAligned() {
		d.rectPathDevice(trimmed)
	} else {
		d.rectPath(dr)
	}
	d.fillOrStroke(p)
}

// axisAligned reports a matrix with no rotation or skew, where a rect stays a rect and can
// therefore be trimmed exactly. For a rotated one the bounding-box intersection is a superset,
// so this backend keeps culling only — the honest limit of a rect-based clip.
func (d *softDevice) axisAligned() bool { return d.mat[1] == 0 && d.mat[3] == 0 }

// trimToClip intersects a DEVICE-space rect with the clip in force, reporting whether anything
// is left of it.
func (d *softDevice) trimToClip(dev Rect) (Rect, bool) {
	if d.clip == nil {
		return dev, true
	}
	c := *d.clip
	if c.Width <= 0 || c.Height <= 0 {
		return Rect{}, false
	}
	x0 := maxf(dev.X, c.X)
	y0 := maxf(dev.Y, c.Y)
	x1 := minf(dev.X+dev.Width, c.X+c.Width)
	y1 := minf(dev.Y+dev.Height, c.Y+c.Height)
	if x1 <= x0 || y1 <= y0 {
		return Rect{}, false
	}
	return Rect{X: x0, Y: y0, Width: x1 - x0, Height: y1 - y0}, true
}

// rectPathDevice builds the path for a rect ALREADY in device space, so the matrix is not
// applied a second time.
func (d *softDevice) rectPathDevice(r Rect) {
	d.gc.BeginPath()
	d.gc.MoveTo(float64(r.X), float64(r.Y))
	d.gc.LineTo(float64(r.X+r.Width), float64(r.Y))
	d.gc.LineTo(float64(r.X+r.Width), float64(r.Y+r.Height))
	d.gc.LineTo(float64(r.X), float64(r.Y+r.Height))
	d.gc.Close()
}

// fillOrStroke finishes the path the way the paint asks: a stroke-only paint strokes, anything
// else fills (and strokes too when it asks for both).
func (d *softDevice) fillOrStroke(p *Paint) {
	if stroke(p) {
		d.gc.Stroke()
		return
	}
	d.gc.Fill()
	if fillStroke(p) {
		d.gc.Stroke()
	}
}

func (d *softDevice) DrawRoundedRect(r Rect, rx, ry float32, p *Paint) {
	if rx <= 0 && ry <= 0 {
		d.DrawRect(r.X, r.Y, r.Width, r.Height, p)
		return
	}
	tx, ty, tw, th := d.TransformRect(r.X, r.Y, r.Width, r.Height)
	if d.clipped(NewRect(tx, ty, tw, th)) {
		return
	}
	// A gradient cannot be handed to draw2d (it has no gradient fill), so the
	// software device rasterises it row by row over the rounded silhouette. Solid
	// paints take the path route, which also gives a genuinely ROUNDED corner —
	// this used to draw a plain rectangle, so every radius a widget asked for was
	// silently square in every offscreen render.
	if p != nil && p.Gradient().Kind != GradientNone && d.axisAligned() {
		d.fillRoundedGradient(r, rx, ry, p)
		return
	}
	d.setPaint(p, true)
	sx, sy := scaleX(d.mat), scaleY(d.mat)
	d.roundedPathDevice(NewRect(tx, ty, tw, th), rx*sx, ry*sy)
	d.fillOrStroke(p)
}

// roundedPathDevice builds a rounded-rect path ALREADY in device space, one cubic
// per corner (the standard kappa construction, accurate to ~0.02% of the radius).
func (d *softDevice) roundedPathDevice(r Rect, rx, ry float32) {
	limit := func(v, max float32) float32 {
		if v > max {
			return max
		}
		if v < 0 {
			return 0
		}
		return v
	}
	rx, ry = limit(rx, r.Width/2), limit(ry, r.Height/2)
	const k = 0.5522847498
	x0, y0, x1, y1 := r.X, r.Y, r.X+r.Width, r.Y+r.Height
	gc := d.gc
	gc.BeginPath()
	gc.MoveTo(float64(x0+rx), float64(y0))
	gc.LineTo(float64(x1-rx), float64(y0))
	gc.CubicCurveTo(float64(x1-rx+k*rx), float64(y0), float64(x1), float64(y0+ry-k*ry), float64(x1), float64(y0+ry))
	gc.LineTo(float64(x1), float64(y1-ry))
	gc.CubicCurveTo(float64(x1), float64(y1-ry+k*ry), float64(x1-rx+k*rx), float64(y1), float64(x1-rx), float64(y1))
	gc.LineTo(float64(x0+rx), float64(y1))
	gc.CubicCurveTo(float64(x0+rx-k*rx), float64(y1), float64(x0), float64(y1-ry+k*ry), float64(x0), float64(y1-ry))
	gc.LineTo(float64(x0), float64(y0+ry))
	gc.CubicCurveTo(float64(x0), float64(y0+ry-k*ry), float64(x0+rx-k*rx), float64(y0), float64(x0+rx), float64(y0))
	gc.Close()
}

// fillRoundedGradient rasterises a linear gradient over a rounded rect one device
// row at a time, insetting each row to the corner arc so the silhouette stays
// round. The matrix must be axis-aligned (the caller checks): the row/column scan
// is defined in device space.
func (d *softDevice) fillRoundedGradient(r Rect, rx, ry float32, p *Paint) {
	g := p.Gradient()
	const steps = 128
	// Precompute the ramp once — interpolating per pixel over a stack of stops is
	// the difference between a screenshot taking 30ms and 300ms on a full page.
	var ramp [steps + 1]uint32
	for i := 0; i <= steps; i++ {
		ramp[i] = softGradientAt(g, float32(i)/steps)
	}
	sx, sy := scaleX(d.mat), scaleY(d.mat)
	if sx == 0 {
		sx = 1
	}
	if sy == 0 {
		sy = 1
	}
	tx, ty, tw, th := d.TransformRect(r.X, r.Y, r.Width, r.Height)
	alpha := p.GlobalAlpha()
	if alpha <= 0 {
		alpha = 1
	}

	// Horizontal ramp: one column per step, same silhouette.
	if g.Kind == GradientLinear && math.Abs(float64(g.End.X-g.Start.X)) > math.Abs(float64(g.End.Y-g.Start.Y)) {
		for ix := 0; ix < int(tw); ix++ {
			lx := r.X + (float32(ix)+0.5)/sx
			inset := softCornerInset(lx-r.X, r.Width, rx)
			y0 := ty + inset*sy
			y1 := ty + th - inset*sy
			if y1 <= y0 {
				continue
			}
			t := softGradientT(g, Point{X: lx, Y: r.Y + r.Height/2})
			d.gc.SetFillColor(premulRGBA(ramp[int(t*steps+0.5)], alpha))
			d.rectPathDevice(Rect{X: tx + float32(ix), Y: y0, Width: 1, Height: y1 - y0})
			d.gc.Fill()
		}
		return
	}

	for iy := 0; iy < int(th); iy++ {
		ly := r.Y + (float32(iy)+0.5)/sy
		inset := softCornerInset(ly-r.Y, r.Height, ry)
		x0 := tx + inset*sx
		x1 := tx + tw - inset*sx
		if x1 <= x0 {
			continue
		}
		t := softGradientT(g, Point{X: r.X + r.Width/2, Y: ly})
		d.gc.SetFillColor(premulRGBA(ramp[int(t*steps+0.5)], alpha))
		d.rectPathDevice(Rect{X: x0, Y: ty + float32(iy), Width: x1 - x0, Height: 1})
		d.gc.Fill()
	}
}

// premulRGBA converts a seam colour (0xAARRGGBB) to the premultiplied color.RGBA
// that freetype's raster painter expects, with an extra opacity multiplier.
func premulRGBA(c uint32, opacity float32) color.RGBA {
	a := float32((c>>24)&0xFF) / 255 * opacity
	if a > 1 {
		a = 1
	}
	return color.RGBA{
		R: uint8(float32((c>>16)&0xFF) * a),
		G: uint8(float32((c>>8)&0xFF) * a),
		B: uint8(float32(c&0xFF) * a),
		A: uint8(a * 255),
	}
}

// softCornerInset is how far a row/column at `d` along an edge of length `extent`
// moves in from that edge to stay inside a corner of radius r.
func softCornerInset(d, extent, r float32) float32 {
	if r <= 0 {
		return 0
	}
	if r > extent/2 {
		r = extent / 2
	}
	if d < r {
		u := r - d
		if v := r*r - u*u; v > 0 {
			return r - float32(math.Sqrt(float64(v)))
		}
		return r
	}
	if d > extent-r {
		u := d - (extent - r)
		if v := r*r - u*u; v > 0 {
			return r - float32(math.Sqrt(float64(v)))
		}
		return r
	}
	return 0
}

// softGradientT maps a point (in the shape's own space) to the ramp parameter.
func softGradientT(g Gradient, pt Point) float32 {
	switch g.Kind {
	case GradientLinear:
		dx, dy := g.End.X-g.Start.X, g.End.Y-g.Start.Y
		den := dx*dx + dy*dy
		if den <= 0 {
			return 0
		}
		t := ((pt.X-g.Start.X)*dx + (pt.Y-g.Start.Y)*dy) / den
		return clampUnit(t)
	case GradientRadial:
		if g.Radius <= 0 {
			return 0
		}
		dx, dy := pt.X-g.Center.X, pt.Y-g.Center.Y
		return clampUnit(float32(math.Sqrt(float64(dx*dx+dy*dy))) / g.Radius)
	case GradientConic:
		dx, dy := pt.X-g.Center.X, pt.Y-g.Center.Y
		ang := float32(math.Atan2(float64(dy), float64(dx)) * 180 / math.Pi)
		base := g.StartAngle
		t := (ang - base) / 360
		for t < 0 {
			t++
		}
		for t >= 1 {
			t--
		}
		return t
	}
	return 0
}

func clampUnit(t float32) float32 {
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}

// softGradientAt returns the colour at t, interpolating between the nearest stops.
func softGradientAt(g Gradient, t float32) uint32 {
	if len(g.Stops) == 0 {
		return 0xFF000000
	}
	if t <= g.Stops[0].Offset {
		return uint32(g.Stops[0].Color)
	}
	last := g.Stops[len(g.Stops)-1]
	if t >= last.Offset {
		return uint32(last.Color)
	}
	for i := 0; i < len(g.Stops)-1; i++ {
		a, b := g.Stops[i], g.Stops[i+1]
		if t < a.Offset || t > b.Offset {
			continue
		}
		span := b.Offset - a.Offset
		if span <= 0 {
			return uint32(b.Color)
		}
		f := (t - a.Offset) / span
		return softMix(uint32(a.Color), uint32(b.Color), f)
	}
	return uint32(last.Color)
}

func softMix(a, b uint32, f float32) uint32 {
	mix := func(x, y uint32) uint32 {
		return x + uint32((float64(int32(y)-int32(x))*float64(f))+0.5)
	}
	return mix((a>>24)&0xFF, (b>>24)&0xFF)<<24 |
		mix((a>>16)&0xFF, (b>>16)&0xFF)<<16 |
		mix((a>>8)&0xFF, (b>>8)&0xFF)<<8 |
		mix(a&0xFF, b&0xFF)
}

func (d *softDevice) DrawCircle(cx, cy, radius float32, p *Paint) {
	d.DrawOval(cx-radius, cy-radius, radius*2, radius*2, p)
}

func (d *softDevice) DrawOval(x, y, w, h float32, p *Paint) {
	tx, ty, tw, th := d.TransformRect(x, y, w, h)
	if d.clipped(NewRect(tx, ty, tw, th)) {
		return
	}
	d.setPaint(p, true)
	c := softApply(d.mat, NewPoint(x+w/2, y+h/2))
	// A circle in device space; non-uniform scales are approximated by the mean
	// radius, which is exact for the uniform case the chart uses.
	rx := w / 2 * scaleX(d.mat)
	ry := h / 2 * scaleY(d.mat)
	rad := (rx + ry) / 2
	d.gc.BeginPath()
	const segs = 32
	for i := 0; i <= segs; i++ {
		ang := 2 * math.Pi * float64(i) / segs
		px := float64(c.X) + float64(rad)*math.Cos(ang)
		py := float64(c.Y) + float64(rad)*math.Sin(ang)
		if i == 0 {
			d.gc.MoveTo(px, py)
		} else {
			d.gc.LineTo(px, py)
		}
	}
	d.gc.Close()
	if stroke(p) {
		d.gc.Stroke()
		return
	}
	d.gc.Fill()
	if fillStroke(p) {
		d.gc.Stroke()
	}
}

func (d *softDevice) DrawArc(x, y, w, h, start, sweep float32, p *Paint) {
	// Approximated as its bounding box: an arc this backend cannot express is
	// better drawn coarsely than dropped (the Device contract's own principle).
	d.DrawOval(x, y, w, h, p)
}

func (d *softDevice) DrawLine(x1, y1, x2, y2 float32, p *Paint) {
	if d.clipped(bbox(x1, y1, x2, y2, p)) {
		return
	}
	a := softApply(d.mat, NewPoint(x1, y1))
	b := softApply(d.mat, NewPoint(x2, y2))
	// A segment that CROSSES the clip has to be cut at the boundary. Culling alone draws it
	// whole, which is how a candle wick reached past the pane edge.
	if d.clip != nil {
		cx1, cy1, cx2, cy2, ok := clipSegment(a.X, a.Y, b.X, b.Y, *d.clip)
		if !ok {
			return
		}
		a, b = NewPoint(cx1, cy1), NewPoint(cx2, cy2)
	}
	d.setPaint(p, false)
	d.gc.BeginPath()
	d.gc.MoveTo(float64(a.X), float64(a.Y))
	d.gc.LineTo(float64(b.X), float64(b.Y))
	d.gc.Stroke()
}

// clipSegment trims a DEVICE-space segment to the clip (Liang-Barsky), reporting whether any
// part of it survives.
func clipSegment(x1, y1, x2, y2 float32, c Rect) (float32, float32, float32, float32, bool) {
	if c.Width <= 0 || c.Height <= 0 {
		return 0, 0, 0, 0, false
	}
	dx, dy := x2-x1, y2-y1
	t0, t1 := float32(0), float32(1)
	// inside reports whether the segment's parametric range survives one boundary test.
	inside := func(p, q float32) bool {
		if p == 0 {
			return q >= 0 // parallel to this boundary: inside iff it starts inside
		}
		r := q / p
		if p < 0 {
			if r > t1 {
				return false
			}
			if r > t0 {
				t0 = r
			}
		} else {
			if r < t0 {
				return false
			}
			if r < t1 {
				t1 = r
			}
		}
		return true
	}
	if !inside(-dx, x1-c.X) || !inside(dx, c.X+c.Width-x1) ||
		!inside(-dy, y1-c.Y) || !inside(dy, c.Y+c.Height-y1) {
		return 0, 0, 0, 0, false
	}
	return x1 + t0*dx, y1 + t0*dy, x1 + t1*dx, y1 + t1*dy, true
}

func (d *softDevice) DrawDashedLine(x1, y1, x2, y2 float32, dash []float32, phase float32, p *Paint) {
	if len(dash) == 0 {
		d.DrawLine(x1, y1, x2, y2, p)
		return
	}
	// Walk the segment on/off by the dash pattern in device space. Cheaper and
	// more predictable than draw2d's dash support, which has no phase.
	dx, dy := x2-x1, y2-y1
	length := float32Sqrt(dx*dx + dy*dy)
	if length <= 0 {
		return
	}
	ux, uy := dx/length, dy/length
	pos, i, on := phase, 0, true
	for pos < length {
		seg := dash[i%len(dash)]
		end := pos + seg
		if end > length {
			end = length
		}
		if on {
			d.DrawLine(x1+ux*pos, y1+uy*pos, x1+ux*end, y1+uy*end, p)
		}
		pos, i, on = end, i+1, !on
	}
}

func (d *softDevice) DrawPath(path *Path, p *Paint) {
	if path == nil {
		return
	}
	d.setPaint(p, true)
	d.gc.BeginPath()
	for _, op := range path.Ops() {
		switch op.Kind {
		case PathMoveTo:
			pt := softApply(d.mat, op.P0)
			d.gc.MoveTo(float64(pt.X), float64(pt.Y))
		case PathLineTo:
			pt := softApply(d.mat, op.P0)
			d.gc.LineTo(float64(pt.X), float64(pt.Y))
		case PathQuadTo:
			c := softApply(d.mat, op.P0)
			e := softApply(d.mat, op.P1)
			d.gc.QuadCurveTo(float64(c.X), float64(c.Y), float64(e.X), float64(e.Y))
		case PathCubicTo:
			c1 := softApply(d.mat, op.P0)
			c2 := softApply(d.mat, op.P1)
			e := softApply(d.mat, op.P2)
			d.gc.CubicCurveTo(float64(c1.X), float64(c1.Y), float64(c2.X), float64(c2.Y), float64(e.X), float64(e.Y))
		case PathClose:
			d.gc.Close()
		}
	}
	if stroke(p) {
		d.gc.Stroke()
		return
	}
	d.gc.Fill()
	if fillStroke(p) {
		d.gc.Stroke()
	}
}

func (d *softDevice) DrawImage(img *Image, r Rect, p *Paint) {
	if img == nil || img.Width() <= 0 || img.Height() <= 0 {
		return
	}
	// The destination goes through the MATRIX like every other primitive. draw2d draws
	// an image from an explicit rect and knows nothing about the canvas transform, so
	// the untransformed version put every image at the canvas origin — a docs cover
	// landed over the top bar while its plate sat in the right place, which is exactly
	// how confusing the symptom was.
	tx, ty, tw, th := d.TransformRect(r.X, r.Y, r.Width, r.Height)
	if tw <= 0 || th <= 0 {
		return
	}
	dest := NewRect(tx, ty, tw, th)
	if d.clipped(dest) {
		return
	}
	// draw2d draws an image WHOLE and knows nothing about a clip, so a cover-fit image
	// (a 900x240 banner cropped into 72x72 avatar) spilled its full 270px width across
	// the card. When the destination straddles the clip: snapshot the destination
	// region, draw, then put back every pixel the clip excludes.
	if d.clip != nil {
		trimmed, ok := d.trimToClip(dest)
		if !ok {
			return
		}
		if trimmed != dest {
			d.drawImageClipped(img, dest)
			return
		}
	}
	d.gc.Save()
	d.gc.Translate(float64(tx), float64(ty))
	d.gc.Scale(float64(tw)/float64(img.Width()), float64(th)/float64(img.Height()))
	d.gc.DrawImage(imageFromSeam(img))
	d.gc.Restore()
}

// drawImageClipped draws dest and keeps only the part inside the clip, by restoring
// the excluded pixels from a snapshot of the DESTINATION's rect — not the clip's.
// Snapshotting only the intersection (the first version) restored the inside and left
// everything the image painted outside the clip exactly where it landed.
func (d *softDevice) drawImageClipped(img *Image, dest Rect) {
	// The destination, clamped to the surface.
	b := d.img.Bounds()
	x0 := int(dest.X) - b.Min.X
	y0 := int(dest.Y) - b.Min.Y
	x1 := x0 + int(dest.Width)
	y1 := y0 + int(dest.Height)
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > b.Dx() {
		x1 = b.Dx()
	}
	if y1 > b.Dy() {
		y1 = b.Dy()
	}
	if x1 <= x0 || y1 <= y0 {
		return
	}
	rw, rh := x1-x0, y1-y0
	snap := make([]uint8, rw*rh*4)
	for y := 0; y < rh; y++ {
		src := d.img.Pix[(y0+y)*d.img.Stride+x0*4:]
		copy(snap[y*rw*4:(y+1)*rw*4], src[:rw*4])
	}
	d.gc.Save()
	d.gc.Translate(float64(dest.X), float64(dest.Y))
	d.gc.Scale(float64(dest.Width)/float64(img.Width()), float64(dest.Height)/float64(img.Height()))
	d.gc.DrawImage(imageFromSeam(img))
	d.gc.Restore()

	clip := d.clip
	for y := 0; y < rh; y++ {
		for x := 0; x < rw; x++ {
			px := b.Min.X + x0 + x
			py := b.Min.Y + y0 + y
			if clip != nil && px >= int(clip.X) && px < int(clip.X+clip.Width) &&
				py >= int(clip.Y) && py < int(clip.Y+clip.Height) {
				continue
			}
			off := (y0+y)*d.img.Stride + (x0+x)*4
			copy(d.img.Pix[off:off+4], snap[(y*rw+x)*4:(y*rw+x)*4+4])
		}
	}
}

func (d *softDevice) DrawText(text string, x, y float32, f Font, p *Paint) {
	d.drawString(text, x, y, f, p)
}

func (d *softDevice) DrawSimpleString(text string, x, y float32, f Font, p *Paint) {
	d.drawString(text, x, y, f, p)
}

// useFace selects the draw2d face for a seam Font. Without this the software
// renderer drew EVERY string with the one discovered system face: a 400-weight
// paragraph and a 700-weight heading looked identical, and a scene's typography
// could not be reviewed offscreen at all. The face is registered on demand (draw2d
// keys faces by name+family+style) and falls back to the discovered system face
// when the font carries no TTF.
func (d *softDevice) useFace(f Font) {
	if f == nil {
		return
	}
	face := f.Face()
	if face == nil {
		return
	}
	data := softFontData
	if face.family != "" {
		data.Name = face.family
	}
	if face.Weight >= weight.SemiBold {
		data.Style |= draw2d.FontStyleBold
	}
	if face.Slant == slant.Italic {
		data.Style |= draw2d.FontStyleItalic
	}
	ttf := face.face
	if ttf == nil {
		ttf = softFont
	}
	if ttf == nil {
		return
	}
	draw2d.RegisterFont(data, ttf)
	d.gc.SetFontData(data)
}

func (d *softDevice) drawString(text string, x, y float32, f Font, p *Paint) {
	if text == "" {
		return
	}
	// Font.Size is the requested CAP HEIGHT, and the rasterization em is larger
	// (emPx solves for cap == size against the face's own metrics — the same loop the
	// atlas and GL use). Drawing at the raw size rasterized 30-40% small: an avatar's
	// initial was 9px of ink for a 13px request, while centring used the atlas advance
	// for the 18px em, so every centred letter sat LEFT of its circle. Draw at the same
	// em the metrics measure with, and the two agree again.
	size := float32(12)
	if f != nil {
		if em := f.emPx(); em > 0 {
			size = em
		} else if s := f.Size(); s > 0 {
			size = s
		}
	}
	d.setPaint(p, true)
	pt := softApply(d.mat, NewPoint(x, y))
	d.gc.SetFontSize(float64(size))
	d.useFace(f)
	d.gc.FillStringAt(text, float64(pt.X), float64(pt.Y))
}

func (d *softDevice) DropShadow(r Rect, p *Paint, dx, dy, radius float32) {
	// Drawn as a translucent offset rectangle: enough to keep panels readable,
	// and it never touches the real shadow path in the GL backend.
	sh := *p
	sh.SetColor(RGBA(0, 0, 0, 0x40))
	d.DrawRect(r.X+dx, r.Y+dy, r.Width, r.Height, &sh)
}

// ── Offscreen ───────────────────────────────────────────────────────

// RenderOffscreen runs the same draw callback every other Device runs, over a
// software Canvas, and returns the result as a seam Image. Nothing here touches
// GL: there is no FBO, no batch and no screen-size uniform to mutate, which is
// the entire point of this backend.
func (d *softDevice) RenderOffscreen(width, height int, draw func(*Canvas)) (*Image, error) {
	dev := NewSoftDevice(width, height)
	c := NewCanvas(dev)
	if draw != nil {
		draw(c)
	}
	return NewImage(dev.Pixels(), int32(width), int32(height)), nil
}

// NewSoftwareCanvas returns a Canvas drawing into a fresh software surface of
// the given size, plus the device so the caller can take its pixels. Callers use
// it to rasterise a widget on the CPU - the same draw code, a different canvas.
func NewSoftwareCanvas(w, h int) (*Canvas, *softDevice) {
	d := NewSoftDevice(w, h)
	return NewCanvas(d), d
}

// ── helpers ─────────────────────────────────────────────────────────

// softApply maps a point through a row-major [sx,kx,tx, ky,sy,ty, 0,0,1] matrix.
func softApply(m Matrix, p Point) Point {
	return Point{
		X: m[0]*p.X + m[1]*p.Y + m[2],
		Y: m[3]*p.X + m[4]*p.Y + m[5],
	}
}

// softMul returns a*b for the same layout (a applied after b).
func softMul(a, b Matrix) Matrix {
	var r Matrix
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			r[row*3+col] = a[row*3+0]*b[0*3+col] + a[row*3+1]*b[1*3+col] + a[row*3+2]*b[2*3+col]
		}
	}
	return r
}

func unpackColor(c uint32) (uint8, uint8, uint8, uint8) {
	return uint8((c >> 16) & 0xFF), uint8((c >> 8) & 0xFF), uint8(c & 0xFF), uint8((c >> 24) & 0xFF)
}

func scaleX(m Matrix) float32 { return float32Sqrt(m[0]*m[0] + m[3]*m[3]) }
func scaleY(m Matrix) float32 { return float32Sqrt(m[1]*m[1] + m[4]*m[4]) }

func bbox(x1, y1, x2, y2 float32, p *Paint) Rect {
	w := float32(1)
	if p != nil && p.StrokeWidth() > 0 {
		w = p.StrokeWidth()
	}
	return Rect{X: minf(x1, x2) - w, Y: minf(y1, y2) - w, Width: absf(x2-x1) + 2*w, Height: absf(y2-y1) + 2*w}
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
func absf(a float32) float32 {
	if a < 0 {
		return -a
	}
	return a
}
func min4(a, b, c, e float32) float32 { return minf(minf(a, b), minf(c, e)) }
func max4(a, b, c, e float32) float32 { return maxf(maxf(a, b), maxf(c, e)) }

// float32Sqrt avoids importing math for one call.
func float32Sqrt(v float32) float32 {
	if v <= 0 {
		return 0
	}
	x := v
	for i := 0; i < 20; i++ {
		x = (x + v/x) / 2
	}
	return x
}

// imageFromSeam converts a seam Image's pixels into an image.Image for draw2d,
// cached by the Image so repeated blits do not re-convert.
func imageFromSeam(img *Image) image.Image {
	if cached, ok := seamImageCache[img]; ok {
		return cached
	}
	w, h := int(img.Width()), int(img.Height())
	src := img.Pixels()
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	// A seam Image is BOTTOM-UP (Pixels' convention, and what GL readbacks give);
	// draw2d's surface is top-down, so the rows flip here. Without it every image
	// the soft device draws — a chart's cached layer, a node's preview, the graph's
	// supersampled cables — lands upside down.
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := src[(h-1-y)*w+x]
			i := out.PixOffset(x, y)
			out.Pix[i+0] = uint8((p >> 16) & 0xFF)
			out.Pix[i+1] = uint8((p >> 8) & 0xFF)
			out.Pix[i+2] = uint8(p & 0xFF)
			out.Pix[i+3] = uint8((p >> 24) & 0xFF)
		}
	}
	seamImageCache[img] = out
	return out
}

var seamImageCache = map[*Image]image.Image{}
