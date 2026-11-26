// Package graphics is the meui drawing seam: meui-owned value types, a
// Paint, and a Canvas facade. Toolkit packages depend on this package
// rather than on the renderer that backs it, so a second backend can be
// supplied without touching widget code.
//
// Neutral-type migration (P1) is in progress. Color, Point, Rect, Size,
// Matrix and Paint are meui-owned here. Path and Image are still aliases to
// the mobilegles types and become owned interfaces when the Canvas facade is
// split over a Device (P1b-3); until then the conversions below are how the
// seam talks to the renderer.
package graphics

import (
	"github.com/sixnineone/mobilegles"
)

// Color is a meui-owned 0xAARRGGBB colour. It is deliberately kept as a
// scalar with the same encoding: the toolkit uses raw colour literals
// throughout and compares them to untyped constants.
type Color uint32

// Point, Rect, Size and Matrix are plain meui-owned value types. The toolkit
// only ever reads their fields, so they carry no geometry methods.
type Point struct{ X, Y float32 }
type Rect struct{ X, Y, Width, Height float32 }
type Size struct{ Width, Height float32 }
type Matrix [9]float32 // row-major 3x3 affine: [sx, kx, tx, ky, sy, ty, 0, 0, 1]

func NewRect(x, y, w, h float32) Rect { return Rect{X: x, Y: y, Width: w, Height: h} }
func NewPoint(x, y float32) Point     { return Point{X: x, Y: y} }
func NewSize(w, h float32) Size       { return Size{Width: w, Height: h} }
func NewMatrix() Matrix               { return Matrix{1, 0, 0, 0, 1, 0, 0, 0, 1} }

// Intersects reports whether the rectangles overlap. Kept on the owned type
// because the toolkit uses it (debug tree layout).
func (r Rect) Intersects(other Rect) bool {
	return r.X < other.X+other.Width &&
		r.X+r.Width > other.X &&
		r.Y < other.Y+other.Height &&
		r.Y+r.Height > other.Y
}

// Contains reports whether the point (x, y) lies inside the rectangle.
func (r Rect) Contains(x, y float32) bool {
	return x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height
}

// RGB creates an opaque colour.
func RGB(r, g, b uint8) Color {
	return Color(0xFF000000 | (uint32(r) << 16) | (uint32(g) << 8) | uint32(b))
}

// RGBA creates a colour with an explicit alpha.
func RGBA(r, g, b, a uint8) Color {
	return Color((uint32(a) << 24) | (uint32(r) << 16) | (uint32(g) << 8) | uint32(b))
}

// ARGB matches the unison signature: alpha as a 0.0–1.0 float and
// red/green/blue as 0–255 ints. mobilegles has only uint8 RGBA, so
// scale alpha to 0–255 and clamp.
func ARGB(alpha float32, red, green, blue int) Color {
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	return RGBA(uint8(red), uint8(green), uint8(blue), uint8(alpha*255))
}

// GradientKind selects a gradient's geometry.
type GradientKind int

const (
	GradientNone GradientKind = iota
	GradientLinear
	GradientRadial
	GradientConic
)

// GradientStop is one colour stop at Offset (0..1) along the gradient.
type GradientStop struct {
	Offset float32
	Color  Color
}

// Gradient is a meui-owned multi-stop gradient. Exactly one geometry applies,
// chosen by Kind:
//
//	GradientLinear: the ramp runs from Start to End.
//	GradientRadial: the ramp runs from Center out to Radius.
//	GradientConic:  the ramp sweeps around Center from StartAngle degrees.
type Gradient struct {
	Kind  GradientKind
	Stops []GradientStop
	Start Point
	End   Point
	// Center/Radius serve GradientRadial and GradientConic; Radius is unused by
	// the conic form, which uses StartAngle.
	Center     Point
	Radius     float32
	StartAngle float32
}

// Paint is a meui-owned draw state. Its fields stay unexported and are
// read/written through methods, matching the call sites the toolkit already
// uses; the renderer's own paint is built lazily behind it.
type Paint struct {
	color       Color
	style       int32
	strokeW     float32
	strokeCap   StrokeCap
	strokeJoin  StrokeJoin
	miterLimit  float32
	antialias   bool
	globalAlpha float32
	blendMode   BlendMode
	// gradient is seam-owned: the renderer's own gradient type stays behind
	// the seam, like every other Paint field.
	gradient Gradient

	mg    *mobilegles.Paint
	stale bool
}

// NewPaint returns a paint with the same defaults the renderer uses:
// opaque white, filled, 1px stroke, butt caps, miter joins (limit 4),
// antialiased, fully opaque, normal blend.
func NewPaint() *Paint {
	return &Paint{
		color:       0xFFFFFFFF,
		style:       int32(mobilegles.Fill),
		strokeW:     1,
		strokeCap:   StrokeCapButt,
		strokeJoin:  StrokeJoinMiter,
		miterLimit:  4,
		antialias:   true,
		globalAlpha: 1,
		blendMode:   BlendModeNormal,
		stale:       true,
	}
}

func (p *Paint) SetColor(c Color) { p.color = c; p.stale = true }
func (p *Paint) Color() Color {
	if p == nil {
		return 0
	}
	return p.color
}
func (p *Paint) SetStyle(style int32) { p.style = style; p.stale = true }
func (p *Paint) Style() int32 {
	if p == nil {
		return 0
	}
	return p.style
}
func (p *Paint) SetStrokeWidth(w float32) {
	p.strokeW = w
	p.stale = true
}
func (p *Paint) StrokeWidth() float32 {
	if p == nil {
		return 0
	}
	return p.strokeW
}
func (p *Paint) SetStrokeCap(c StrokeCap) {
	p.strokeCap = c
	p.stale = true
}
func (p *Paint) StrokeCap() StrokeCap {
	if p == nil {
		return StrokeCapButt
	}
	return p.strokeCap
}
func (p *Paint) SetStrokeJoin(j StrokeJoin) {
	p.strokeJoin = j
	p.stale = true
}
func (p *Paint) StrokeJoin() StrokeJoin {
	if p == nil {
		return StrokeJoinMiter
	}
	return p.strokeJoin
}

// SetMiterLimit sets how far a miter join may spike, in multiples of half the
// stroke width, before it is beveled. Values <= 0 restore the default of 4.
func (p *Paint) SetMiterLimit(limit float32) {
	p.miterLimit = limit
	p.stale = true
}
func (p *Paint) MiterLimit() float32 {
	if p == nil || p.miterLimit <= 0 {
		return 4
	}
	return p.miterLimit
}

// SetAntialias turns edge antialiasing on or off for this paint. Text and
// images are unaffected: they sample a texture and have no edge ramp.
func (p *Paint) SetAntialias(on bool) {
	p.antialias = on
	p.stale = true
}
func (p *Paint) Antialias() bool {
	if p == nil {
		return true
	}
	return p.antialias
}

// SetGlobalAlpha multiplies the alpha of everything this paint draws, on top
// of the colour's own alpha. Values outside 0..1 are clamped; 1 is no
// modulation. Unlike SetAntialias this applies to text and images too.
func (p *Paint) SetGlobalAlpha(a float32) {
	p.globalAlpha = a
	p.stale = true
}
func (p *Paint) GlobalAlpha() float32 {
	if p == nil || p.globalAlpha < 0 {
		return 1
	}
	if p.globalAlpha > 1 {
		return 1
	}
	return p.globalAlpha
}
func (p *Paint) SetBlendMode(m BlendMode) { p.blendMode = m; p.stale = true }
func (p *Paint) BlendMode() BlendMode {
	if p == nil {
		return BlendModeNormal
	}
	return p.blendMode
}

// SetLinearGradient sets a multi-stop gradient running start→end.
func (p *Paint) SetLinearGradient(start, end Point, stops []GradientStop) {
	p.gradient = Gradient{Kind: GradientLinear, Start: start, End: end, Stops: stops}
	p.stale = true
}

// SetRadialGradient sets a multi-stop gradient running from center out to
// radius.
func (p *Paint) SetRadialGradient(center Point, radius float32, stops []GradientStop) {
	p.gradient = Gradient{Kind: GradientRadial, Center: center, Radius: radius, Stops: stops}
	p.stale = true
}

// SetConicGradient sets a multi-stop gradient sweeping around center from
// startAngle degrees.
func (p *Paint) SetConicGradient(center Point, startAngle float32, stops []GradientStop) {
	p.gradient = Gradient{Kind: GradientConic, Center: center, StartAngle: startAngle, Stops: stops}
	p.stale = true
}

// SetGradient replaces the paint's gradient wholesale.
func (p *Paint) SetGradient(g Gradient) { p.gradient = g; p.stale = true }

// Gradient returns the paint's gradient; Kind == GradientNone means solid.
func (p *Paint) Gradient() Gradient {
	if p == nil {
		return Gradient{}
	}
	return p.gradient
}
func (p *Paint) ClearGradient() { p.gradient = Gradient{}; p.stale = true }

// renderer returns the backend paint for this state, reusing one allocation.
// A nil *Paint is safe and yields the backend default, matching the previous
// alias behaviour.
func (p *Paint) renderer() *mobilegles.Paint {
	if p == nil {
		return nil
	}
	if p.mg == nil {
		p.mg = mobilegles.NewPaint()
		p.stale = true
	}
	if p.stale {
		p.mg.SetColor(mobilegles.Color(p.color))
		p.mg.SetStyle(p.style)
		p.mg.SetStrokeWidth(p.strokeW)
		p.mg.SetStrokeCap(mobilegles.StrokeCap(p.strokeCap))
		p.mg.SetStrokeJoin(mobilegles.StrokeJoin(p.strokeJoin))
		p.mg.SetMiterLimit(p.MiterLimit())
		p.mg.SetAntialias(p.antialias)
		p.mg.SetGlobalAlpha(p.globalAlpha)
		p.mg.SetBlendMode(mobilegles.BlendMode(p.blendMode))
		p.mg.ClearGradient()
		if stops := mgStops(p.gradient.Stops); len(stops) > 0 {
			switch p.gradient.Kind {
			case GradientLinear:
				p.mg.SetLinearGradient(mgPoint(p.gradient.Start), mgPoint(p.gradient.End), stops)
			case GradientRadial:
				p.mg.SetRadialGradient(mgPoint(p.gradient.Center), p.gradient.Radius, stops)
			case GradientConic:
				p.mg.SetConicGradient(mgPoint(p.gradient.Center), p.gradient.StartAngle, stops)
			}
		}
		p.stale = false
	}
	return p.mg
}

// Paint satisfies Ink so a plain colour can be used wherever ink is asked
// for (TextDecoration.OnBackgroundInk). This mirrors mobilegles.Color's own
// Paint method.
func (c Color) Paint(_ *Canvas, _ Rect, _ int32) *Paint {
	p := NewPaint()
	p.SetColor(c)
	return p
}

// --- conversions to the current backend -----------------------------------
//
// These exist only while Canvas still wraps the mobilegles canvas
// (P1b-3 replaces them with a Device implementation).

// mgStops converts owned gradient stops to the renderer's.
func mgStops(stops []GradientStop) []mobilegles.GradientStop {
	if len(stops) == 0 {
		return nil
	}
	out := make([]mobilegles.GradientStop, len(stops))
	for i, s := range stops {
		out[i] = mobilegles.GradientStop{Offset: s.Offset, Color: mobilegles.Color(s.Color)}
	}
	return out
}

func mgRect(r Rect) mobilegles.Rect {
	return mobilegles.Rect{X: r.X, Y: r.Y, Width: r.Width, Height: r.Height}
}
func mgPoint(p Point) mobilegles.Point { return mobilegles.Point{X: p.X, Y: p.Y} }
func mgSize(s Size) mobilegles.Size    { return mobilegles.Size{Width: s.Width, Height: s.Height} }

// Path is meui-owned pure data: it records the ops the caller issues, and a
// Device replays them into whatever path representation the backend uses. No
// renderer type is involved, which is what lets a second backend render paths.
type PathOpKind uint8

const (
	PathMoveTo PathOpKind = iota
	PathLineTo
	PathQuadTo
	PathCubicTo
	PathClose
)

// PathOp is one recorded path operation. P0 is the control point for
// PathQuadTo and the target for the others; for PathQuadTo P1 is the end
// point, and for PathCubicTo P0/P1 are the two controls and P2 the end.
type PathOp struct {
	Kind   PathOpKind
	P0, P1 Point
	P2     Point
}

type Path struct{ ops []PathOp }

func NewPath() *Path { return &Path{} }

func (p *Path) MoveTo(pt Point) {
	if p == nil {
		return
	}
	p.ops = append(p.ops, PathOp{Kind: PathMoveTo, P0: pt})
}
func (p *Path) LineTo(pt Point) {
	if p == nil {
		return
	}
	p.ops = append(p.ops, PathOp{Kind: PathLineTo, P0: pt})
}
func (p *Path) QuadTo(ctrl, end Point) {
	if p == nil {
		return
	}
	p.ops = append(p.ops, PathOp{Kind: PathQuadTo, P0: ctrl, P1: end})
}

// CubicTo adds a cubic bezier segment from the current point through the two
// controls to end. A Device flattens it into whatever its rasterizer supports.
func (p *Path) CubicTo(ctrl1, ctrl2, end Point) {
	if p == nil {
		return
	}
	p.ops = append(p.ops, PathOp{Kind: PathCubicTo, P0: ctrl1, P1: ctrl2, P2: end})
}
func (p *Path) Close() {
	if p == nil {
		return
	}
	p.ops = append(p.ops, PathOp{Kind: PathClose})
}
func (p *Path) Reset() {
	if p == nil {
		return
	}
	p.ops = p.ops[:0]
}

// Ops returns the recorded operations. A Device implementation replays these
// into its own path type.
func (p *Path) Ops() []PathOp {
	if p == nil {
		return nil
	}
	return p.ops
}

// Image is meui-owned: it holds the pixel buffer and dimensions, and the
// backend image is built from it on demand. A Device therefore never has to be
// handed a renderer object by the caller.
type Image struct {
	w, h int32
	pix  []uint32

	mg *mobilegles.Image
}

// NewImage wraps a bottom-up 0xAARRGGBB pixel buffer.
func NewImage(rgba []uint32, w, h int32) *Image {
	return &Image{w: w, h: h, pix: rgba}
}

func (i *Image) Width() int32 {
	if i == nil {
		return 0
	}
	return i.w
}

func (i *Image) Height() int32 {
	if i == nil {
		return 0
	}
	return i.h
}

// Pixels returns the raw buffer (bottom-up 0xAARRGGBB, premultiplied).
func (i *Image) Pixels() []uint32 {
	if i == nil {
		return nil
	}
	return i.pix
}

// Bounds reports the image extent as a Rect with its origin at (0, 0).
func (i *Image) Bounds() Rect {
	if i == nil {
		return Rect{}
	}
	return Rect{Width: float32(i.w), Height: float32(i.h)}
}

// SetPixel writes c at (x, y), given in TOP-LEFT image coordinates, and marks
// the backend texture stale so the next draw re-uploads it. c is straight
// alpha and is premultiplied here; writes outside the image are ignored.
func (i *Image) SetPixel(x, y int, c Color) {
	if i == nil || x < 0 || y < 0 || x >= int(i.w) || y >= int(i.h) {
		return
	}
	i.pix[(int(i.h)-1-y)*int(i.w)+x] = premultiplied(
		uint8((c>>16)&0xFF), uint8((c>>8)&0xFF), uint8(c&0xFF), uint8((c>>24)&0xFF))
	i.markDirty()
}

// PixelAt returns the pixel at (x, y), in TOP-LEFT image coordinates, with
// straight alpha (the inverse of the premultiplication SetPixel applies).
// Out-of-range reads return a fully transparent colour.
func (i *Image) PixelAt(x, y int) Color {
	if i == nil || x < 0 || y < 0 || x >= int(i.w) || y >= int(i.h) {
		return 0
	}
	px := i.pix[(int(i.h)-1-y)*int(i.w)+x]
	a := (px >> 24) & 0xFF
	if a == 0 {
		return 0
	}
	un := func(v uint32) uint32 {
		u := v * 255 / a
		if u > 255 {
			u = 255
		}
		return u
	}
	return Color(a<<24 | un((px>>16)&0xFF)<<16 | un((px>>8)&0xFF)<<8 | un(px&0xFF))
}

// PutImage copies src over the receiver with its top-left corner at (dx, dy) —
// the canvas putImageData operation. Both images are premultiplied, so the copy
// is a straight blit; the parts of src that fall outside the receiver are
// dropped and the backend texture is marked stale.
func (i *Image) PutImage(src *Image, dx, dy int) {
	if i == nil || src == nil || src.w <= 0 || src.h <= 0 {
		return
	}
	for sy := 0; sy < int(src.h); sy++ {
		ty := dy + sy
		if ty < 0 || ty >= int(i.h) {
			continue
		}
		// Both buffers are bottom-up; translate the top-left row index.
		dst := i.pix[(int(i.h)-1-ty)*int(i.w) : (int(i.h)-ty)*int(i.w)]
		row := src.pix[(int(src.h)-1-sy)*int(src.w) : (int(src.h)-sy)*int(src.w)]
		for sx := 0; sx < int(src.w); sx++ {
			tx := dx + sx
			if tx < 0 || tx >= int(i.w) {
				continue
			}
			dst[tx] = row[sx]
		}
	}
	i.markDirty()
}

// SetPixels replaces the image's pixel buffer in place, for a buffer that is
// re-rendered on the same surface (a cached layer). The dimensions must match: the
// backend keeps one texture per image, and swapping a different-sized buffer under
// it would re-upload at the wrong extent. The next draw re-uploads.
func (i *Image) SetPixels(pix []uint32) {
	if i == nil {
		return
	}
	if int32(len(pix)) != i.w*i.h {
		return
	}
	i.pix = pix
	// The backend view holds its OWN pixel slice — it was handed i.pix when it was
	// created — so replacing the seam's buffer without telling it leaves the GPU
	// re-uploading the buffer from before. That is a cached layer that keeps showing
	// what it held the first time it was rendered (the cables snapped back to where
	// the node used to be as soon as a drag ended).
	if i.mg != nil {
		i.mg.SetPixels(pix)
	}
	i.markDirty()
}

// markDirty tells the backend its texture no longer matches the pixel buffer.
// The re-upload is deferred to the next draw, which runs on the GL thread;
// creating the backend view here costs no GL work and gives the marker
// somewhere to live even before the image has ever been drawn.
func (i *Image) markDirty() {
	if i == nil {
		return
	}
	i.renderer().MarkPixelsDirty()
}

// LogicalSize returns the dimensions as a seam Size, for layout.
func (i *Image) LogicalSize() Size {
	if i == nil {
		return Size{}
	}
	return Size{Width: float32(i.w), Height: float32(i.h)}
}

// ToPNG encodes the image through the backend.
func (i *Image) ToPNG(compressionLevel int) ([]byte, error) {
	if i == nil {
		return nil, errOffscreenFailed
	}
	return i.renderer().ToPNG(compressionLevel)
}

// renderer builds (once) and returns the backend image.
func (i *Image) renderer() *mobilegles.Image {
	if i == nil {
		return nil
	}
	if i.mg == nil {
		i.mg = mobilegles.NewImage(i.pix, i.w, i.h)
	}
	return i.mg
}
