package graphics

import "math"

// Device is the complete backend contract the seam draws through. A Device
// owns the surface, the transform/clip state stack and every draw operation;
// Canvas is a thin facade over one, so the toolkit never names a backend.
//
// This is P1's L1 seam (design §5.2). It is deliberately shaped around what
// the toolkit actually calls rather than around any one renderer: the used
// surface is ~25 operations, which is what makes a second backend feasible.
type Device interface {
	// Surface
	Width() float32
	Height() float32
	Clear(col uint32)
	ClearWithViewport(col uint32, x, y, w, h int32)

	// Transform and clip state
	Save() int
	Restore()
	RestoreToCount(n int)
	Translate(dx, dy float32)
	Scale(sx, sy float32)
	Rotate(deg float32)
	Skew(kx, ky float32)
	GetMatrix() Matrix
	TransformRect(x, y, w, h float32) (float32, float32, float32, float32)
	ClipRect(r Rect)
	// ClipPath intersects the clip with the interior of path. It reports
	// whether the clip is exact: a concave or over-complex outline falls back
	// to its bounding rectangle (a superset, so nothing is hidden) and returns
	// false.
	ClipPath(path *Path) bool
	ResetClip()
	ClipDepth() int
	// ClipBounds is the clip in force, in device space. A widget that draws many
	// items uses it with QuickRejectRect to skip what cannot be seen.
	ClipBounds() Rect
	// IsClipEmpty reports a clip that admits nothing, which is different from a
	// clip that happens to be the whole surface.
	IsClipEmpty() bool
	// QuickRejectRect reports whether a rect lies entirely outside the clip.
	QuickRejectRect(r Rect) bool

	// Drawing
	DrawRect(x, y, w, h float32, p *Paint)
	DrawRoundedRect(r Rect, rx, ry float32, p *Paint)
	DrawCircle(cx, cy, r float32, p *Paint)
	DrawOval(x, y, w, h float32, p *Paint)
	DrawArc(x, y, w, h, start, sweep float32, p *Paint)
	DrawLine(x1, y1, x2, y2 float32, p *Paint)
	DrawDashedLine(x1, y1, x2, y2 float32, dash []float32, phase float32, p *Paint)
	DrawPath(path *Path, p *Paint)
	DrawImage(img *Image, r Rect, p *Paint)
	DrawText(text string, x, y float32, f Font, p *Paint)
	DrawSimpleString(text string, x, y float32, f Font, p *Paint)
	DropShadow(r Rect, p *Paint, dx, dy, radius float32)

	// RenderOffscreen draws into a separate surface and returns it. Widget
	// caches and screenshot paths use this, so the backend owns offscreen
	// creation instead of the seam reaching for a renderer.
	RenderOffscreen(width, height int, draw func(*Canvas)) (*Image, error)
}

// TestDevice is a recording, drawing-free Device. It covers the seam with a
// second backend: every operation is counted and the transform/clip stack is
// emulated so a Canvas over it behaves plausibly. Widget tests paint through
// it, which proves the toolkit cannot depend on renderer behaviour.
type TestDevice struct {
	width, height float32

	// Calls counts operations by name.
	Calls map[string]int
	// Total is the number of recorded operations.
	Total int

	// LastText and LastRect capture the most recent text/rect arguments, and
	// LastBlend/LastMiterLimit/LastAntialias the paint state of the most recent
	// stroke, so a test can prove the paint reaches the device rather than
	// being stored and ignored.
	LastText        string
	LastRect        Rect
	LastBlend       BlendMode
	LastMiterLimit  float32
	LastAntialias   bool
	LastGlobalAlpha float32
	// LastClipPathOps is the op count of the most recent ClipPath, so a test can
	// prove the path reached the device.
	LastClipPathOps int

	saveDepth int
	clipDepth int

	// m is the emulated current transform, and stack is its save stack, so
	// Canvas.Origin and ClipRect behave as they do on a real device.
	m     Matrix
	stack []Matrix

	// clip is the clip in force, so ClipBounds/IsClipEmpty/QuickRejectRect answer
	// as a real device does rather than being unimplementable here.
	clip    Rect
	hasClip bool
	// clipStack parallels stack: a real device restores the CLIP on Restore, and
	// this one must too, or WithClip would leave its clip behind.
	clipStack []clipState
}

type clipState struct {
	clip Rect
	has  bool
}

// NewTestDevice returns a recording Device with the given surface size.
func NewTestDevice(width, height float32) *TestDevice {
	return &TestDevice{width: width, height: height, Calls: make(map[string]int), m: NewMatrix()}
}

// Count returns how many times an operation was recorded.
func (d *TestDevice) Count(op string) int { return d.Calls[op] }

// recordStroke captures the paint state that only affects stroked geometry.
func (d *TestDevice) recordStroke(p *Paint) {
	if p == nil {
		return
	}
	d.LastMiterLimit = p.MiterLimit()
	d.LastAntialias = p.Antialias()
	d.LastGlobalAlpha = p.GlobalAlpha()
}

func (d *TestDevice) record(op string) {
	if d.Calls == nil {
		d.Calls = make(map[string]int)
	}
	d.Calls[op]++
	d.Total++
}

func (d *TestDevice) Width() float32  { return d.width }
func (d *TestDevice) Height() float32 { return d.height }

func (d *TestDevice) Clear(col uint32) { d.record("Clear") }
func (d *TestDevice) ClearWithViewport(col uint32, x, y, w, h int32) {
	d.record("ClearWithViewport")
}

func (d *TestDevice) Save() int {
	d.record("Save")
	d.stack = append(d.stack, d.m)
	d.clipStack = append(d.clipStack, clipState{clip: d.clip, has: d.hasClip})
	d.saveDepth++
	return d.saveDepth
}
func (d *TestDevice) Restore() {
	d.record("Restore")
	if n := len(d.stack); n > 0 {
		d.m = d.stack[n-1]
		d.stack = d.stack[:n-1]
	}
	if n := len(d.clipStack); n > 0 {
		d.clip, d.hasClip = d.clipStack[n-1].clip, d.clipStack[n-1].has
		d.clipStack = d.clipStack[:n-1]
	}
	if d.saveDepth > 0 {
		d.saveDepth--
	}
}
func (d *TestDevice) RestoreToCount(n int) {
	d.record("RestoreToCount")
	for len(d.stack) > n {
		d.m = d.stack[len(d.stack)-1]
		d.stack = d.stack[:len(d.stack)-1]
		if len(d.clipStack) > 0 {
			d.clip, d.hasClip = d.clipStack[len(d.clipStack)-1].clip, d.clipStack[len(d.clipStack)-1].has
			d.clipStack = d.clipStack[:len(d.clipStack)-1]
		}
	}
	if n >= 0 && n < d.saveDepth {
		d.saveDepth = n
	}
}
func (d *TestDevice) Translate(dx, dy float32) {
	d.record("Translate")
	d.m = matrixMul(d.m, Matrix{1, 0, dx, 0, 1, dy, 0, 0, 1})
}
func (d *TestDevice) Scale(sx, sy float32) {
	d.record("Scale")
	d.m = matrixMul(d.m, Matrix{sx, 0, 0, 0, sy, 0, 0, 0, 1})
}
func (d *TestDevice) Rotate(deg float32) {
	d.record("Rotate")
	rad := float64(deg) * math.Pi / 180
	c, s := float32(math.Cos(rad)), float32(math.Sin(rad))
	d.m = matrixMul(d.m, Matrix{c, -s, 0, s, c, 0, 0, 0, 1})
}
func (d *TestDevice) Skew(kx, ky float32) {
	d.record("Skew")
	tx := float32(math.Tan(float64(kx) * math.Pi / 180))
	ty := float32(math.Tan(float64(ky) * math.Pi / 180))
	d.m = matrixMul(d.m, Matrix{1, tx, 0, ty, 1, 0, 0, 0, 1})
}
func (d *TestDevice) GetMatrix() Matrix { return d.m }

// TransformRect maps the rect's corners through the current matrix and
// returns their bounding box, matching what a device clip needs.
func (d *TestDevice) TransformRect(x, y, w, h float32) (float32, float32, float32, float32) {
	corners := [4][2]float32{{x, y}, {x + w, y}, {x, y + h}, {x + w, y + h}}
	minX, minY := float32(math.Inf(1)), float32(math.Inf(1))
	maxX, maxY := float32(math.Inf(-1)), float32(math.Inf(-1))
	for _, c := range corners {
		px := d.m[0]*c[0] + d.m[1]*c[1] + d.m[2]
		py := d.m[3]*c[0] + d.m[4]*c[1] + d.m[5]
		minX, minY = min(minX, px), min(minY, py)
		maxX, maxY = max(maxX, px), max(maxY, py)
	}
	return minX, minY, maxX - minX, maxY - minY
}

// matrixMul multiplies two row-major 3x3 affine matrices (a then b).
func matrixMul(a, b Matrix) Matrix {
	var out Matrix
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			out[r*3+c] = a[r*3]*b[c] + a[r*3+1]*b[3+c] + a[r*3+2]*b[6+c]
		}
	}
	return out
}
func (d *TestDevice) ClipRect(r Rect) {
	d.record("ClipRect")
	d.LastRect = r
	d.clipDepth++
	if !d.hasClip {
		d.clip, d.hasClip = r, true
		return
	}
	x0, y0 := maxf(d.clip.X, r.X), maxf(d.clip.Y, r.Y)
	x1 := minf(d.clip.X+d.clip.Width, r.X+r.Width)
	y1 := minf(d.clip.Y+d.clip.Height, r.Y+r.Height)
	d.clip = Rect{X: x0, Y: y0, Width: x1 - x0, Height: y1 - y0}
}

func (d *TestDevice) ClipBounds() Rect {
	if !d.hasClip {
		return Rect{X: 0, Y: 0, Width: d.Width(), Height: d.Height()}
	}
	return d.clip
}

func (d *TestDevice) IsClipEmpty() bool {
	return d.hasClip && (d.clip.Width <= 0 || d.clip.Height <= 0)
}

func (d *TestDevice) QuickRejectRect(r Rect) bool { return quickReject(d.ClipBounds(), r) }
func (d *TestDevice) ClipPath(path *Path) bool {
	d.record("ClipPath")
	if path == nil || len(path.Ops()) == 0 {
		return false
	}
	// The double does not compute convex hulls; it records that the clip was
	// requested and reports "approximate" so a test never mistakes the double
	// for the real geometry.
	d.LastClipPathOps = len(path.Ops())
	return false
}

func (d *TestDevice) ResetClip() {
	d.record("ResetClip")
	d.clipDepth = 0
	d.hasClip = false
}
func (d *TestDevice) ClipDepth() int { return d.clipDepth }

func (d *TestDevice) DrawRect(x, y, w, h float32, p *Paint) {
	d.record("DrawRect")
	d.LastRect = Rect{X: x, Y: y, Width: w, Height: h}
	if p != nil {
		d.LastBlend = p.BlendMode()
		d.LastGlobalAlpha = p.GlobalAlpha()
	}
}
func (d *TestDevice) DrawRoundedRect(r Rect, rx, ry float32, p *Paint) {
	d.record("DrawRoundedRect")
	d.LastRect = r
}
func (d *TestDevice) DrawCircle(cx, cy, r float32, p *Paint) { d.record("DrawCircle") }
func (d *TestDevice) DrawOval(x, y, w, h float32, p *Paint)  { d.record("DrawOval") }
func (d *TestDevice) DrawArc(x, y, w, h, start, sweep float32, p *Paint) {
	d.record("DrawArc")
}
func (d *TestDevice) DrawLine(x1, y1, x2, y2 float32, p *Paint) {
	d.record("DrawLine")
	d.recordStroke(p)
}
func (d *TestDevice) DrawDashedLine(x1, y1, x2, y2 float32, dash []float32, phase float32, p *Paint) {
	d.record("DrawDashedLine")
}
func (d *TestDevice) DrawPath(path *Path, p *Paint) {
	d.record("DrawPath")
	d.recordStroke(p)
	if path != nil {
		d.record("DrawPath.ops")
		d.Calls["DrawPath.ops"] += len(path.Ops()) - 1
	}
}
func (d *TestDevice) DrawImage(img *Image, r Rect, p *Paint) {
	d.record("DrawImage")
	d.LastRect = r
}
func (d *TestDevice) DrawText(text string, x, y float32, f Font, p *Paint) {
	d.record("DrawText")
	d.LastText = text
}
func (d *TestDevice) DrawSimpleString(text string, x, y float32, f Font, p *Paint) {
	d.record("DrawSimpleString")
	d.LastText = text
}
func (d *TestDevice) DropShadow(r Rect, p *Paint, dx, dy, radius float32) {
	d.record("DropShadow")
}

// RenderOffscreen runs draw against a fresh recording device so widgets still
// exercise their offscreen path, but reports failure: a drawing-free device
// cannot produce pixels. Callers already treat that as "no cache image".
func (d *TestDevice) RenderOffscreen(width, height int, draw func(*Canvas)) (*Image, error) {
	d.record("RenderOffscreen")
	if draw != nil {
		draw(NewCanvas(NewTestDevice(float32(width), float32(height))))
	}
	return nil, errOffscreenFailed
}

// quickReject is the ONE implementation of the culling test, shared by the devices
// so they cannot disagree about what "outside the clip" means.
func quickReject(clip, r Rect) bool {
	if clip.Width <= 0 || clip.Height <= 0 {
		return false
	}
	if r.X+r.Width < clip.X || r.X > clip.X+clip.Width {
		return true
	}
	return r.Y+r.Height < clip.Y || r.Y > clip.Y+clip.Height
}
