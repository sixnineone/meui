package widgets

import (
	"log"
	"math"
	"strings"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/theme"
)

// Image displays a graphics.Image: an image VIEWER, not an editor. It owns no pixels and edits
// nothing — the display path of an image editor ends here, and the document behind it is a separate
// concern (see meui-media).
//
// The default fit is CONTAIN, not fill: a viewer that silently crops or stretches what a user opened is
// the first thing they will not trust.
type Image struct {
	meui.BaseWidget
	*theme.LabelTheme

	img     *graphics.Image
	overlay *graphics.Image
	fit     ImageFit
	check   bool
	radius  float32

	// The rounded-corner layer is built at DISPLAY size and cached per (source, fit,
	// radius, size). Masking the source instead (the first attempt) rounds the
	// SOURCE's corners, which a cover-fitting avatar never shows: it crops the
	// centre of the image, where the source's arcs are far away. The layer is a mask
	// rather than a clip path because the software device cannot clip to a path.
	cacheSrc *graphics.Image
	cacheR   float32
	cacheBw  float32
	cacheBh  float32
	cacheFit ImageFit
	cacheImg *graphics.Image
}

// ImageFit is how the image maps into the widget's rect.
type ImageFit uint8

const (
	// ImageFitContain fits the whole image inside the widget, preserving aspect ratio. The default.
	ImageFitContain ImageFit = iota
	// ImageFitNone draws the image 1:1, centred, and CLIPPED to the widget — what a pixel editor shows
	// at 100%, where one image pixel is one screen pixel.
	ImageFitNone
	// ImageFitFill stretches to the widget, aspect ratio ignored.
	ImageFitFill
	// ImageFitCover fills the widget preserving aspect ratio and CROPS the excess
	// (the widget clips it). It is the fit a web card uses: album art, a hero
	// image, a thumbnail — a stretched face or a letterboxed band both read wrong.
	ImageFitCover
)

func (f ImageFit) String() string {
	switch f {
	case ImageFitNone:
		return "none"
	case ImageFitFill:
		return "fill"
	case ImageFitCover:
		return "cover"
	default:
		return "contain"
	}
}

// NewImage builds an empty viewer.
func NewImage() *Image {
	return &Image{
		BaseWidget: *meui.NewBaseWidget(),
		LabelTheme: theme.DefaultLabelTheme.Copy(),
		check:      true, // transparency reads as transparency unless an app says otherwise
	}
}

// SetImage installs the image to display. A nil image is legal and draws the empty state.
func (w *Image) SetImage(img *graphics.Image) *Image {
	w.img = img
	w.MarkForRedraw()
	return w
}

// Image is the displayed image, which may be nil.
func (w *Image) Image() *graphics.Image { return w.img }

// SetCornerRadius rounds the displayed image's corners. The layer is built at the
// display size (once per size/radius change), so a radius is the radius you SEE —
// and it works in both renderers, which a clip path cannot.
func (w *Image) SetCornerRadius(r float32) *Image {
	if r < 0 {
		r = 0
	}
	w.radius = r
	w.cacheImg = nil
	w.MarkForRedraw()
	return w
}

// CornerRadius is the current mask radius in widget pixels.
func (w *Image) CornerRadius() float32 { return w.radius }

// displayImageFor returns what Paint should draw at this size: the source, or a
// display-sized layer whose corners are rounded. The layer already contains the
// fitted image, so Paint draws it 1:1.
func (w *Image) displayImageFor(bw, bh float32) *graphics.Image {
	if w.img == nil || w.radius <= 0 || bw < 1 || bh < 1 {
		return w.img
	}
	if w.cacheImg != nil && w.cacheSrc == w.img && w.cacheR == w.radius &&
		w.cacheBw == bw && w.cacheBh == bh && w.cacheFit == w.fit {
		return w.cacheImg
	}
	canvas, dev := graphics.NewSoftwareCanvas(int(bw), int(bh))
	canvas.Clear(0x00000000)
	canvas.DrawImageInRect(w.img, contentRectLocal(w.img, w.fit, bw, bh), nil, nil)
	layer := graphics.NewImage(dev.Pixels(), int32(bw), int32(bh))
	roundCornersInPlace(layer, w.radius)
	w.cacheImg, w.cacheSrc, w.cacheR, w.cacheBw, w.cacheBh, w.cacheFit =
		layer, w.img, w.radius, bw, bh, w.fit
	return layer
}

// roundCornersInPlace clears the pixels outside a rounded rect, with one pixel of
// antialiasing at the arc. Work happens in straight alpha (Image.PixelAt/SetPixel),
// so a coverage scale is a plain alpha scale.
func roundCornersInPlace(img *graphics.Image, radius float32) {
	if img == nil || radius <= 0 {
		return
	}
	w, h := int(img.Width()), int(img.Height())
	if w <= 0 || h <= 0 {
		return
	}
	limit := float32(w)
	if h < w {
		limit = float32(h)
	}
	if radius > limit/2 {
		radius = limit / 2
	}
	cx, cy := float32(w)/2, float32(h)/2
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			cov := roundedCoverage(float32(x)+0.5-cx, float32(y)+0.5-cy, float32(w)/2, float32(h)/2, radius)
			if cov >= 1 {
				continue
			}
			if cov <= 0 {
				img.SetPixel(x, y, 0)
				continue
			}
			c := img.PixelAt(x, y)
			a := uint32(float32((c>>24)&0xFF) * cov)
			img.SetPixel(x, y, graphics.Color(a<<24|uint32(c)&0x00FFFFFF))
		}
	}
}

// roundedCoverage is the fraction of a pixel at (px,py) relative to the centre that
// is inside a rounded rect of half-extents (halfW,halfH) and corner radius r, from
// the signed distance (Inigo Quilez's rounded-box form).
func roundedCoverage(px, py, halfW, halfH, r float32) float32 {
	qx := absImage(px) - halfW + r
	qy := absImage(py) - halfH + r
	outside := float32(0)
	if qx > 0 && qy > 0 {
		outside = float32(math.Sqrt(float64(qx*qx + qy*qy)))
	}
	inside := qx
	if qy > qx {
		inside = qy
	}
	if inside > 0 {
		inside = 0
	}
	d := outside + inside - r
	return clamp01(0.5 - d)
}

func absImage(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// SetSource loads an image file and displays it. The path may be PNG, JPEG or the
// formats graphics already decodes; a load failure leaves the viewer empty and is
// returned so a scene can report it rather than show a blank card.
func (w *Image) SetSource(path string) error {
	img, err := graphics.NewImageFromFilePathOrURL(path, 1)
	if err != nil {
		return err
	}
	w.SetImage(img)
	return nil
}

// SetBoundProperty applies the QML properties a scene sets on an Image. It is the
// widget's own vocabulary rather than a global handler, so `source` and `fillMode`
// cannot collide with another widget's property of the same name.
func (w *Image) SetBoundProperty(name string, value interface{}) {
	switch name {
	case "source", "src":
		if s, ok := value.(string); ok && s != "" {
			if err := w.SetSource(s); err != nil {
				log.Printf("image: source %q: %v", s, err)
			}
		}
	case "radius", "cornerRadius":
		// The QML parser hands integers through as int (not float64), so a radius
		// written as `radius: 36` was silently ignored while `36.0` worked.
		switch v := value.(type) {
		case float64:
			w.SetCornerRadius(float32(v))
		case float32:
			w.SetCornerRadius(v)
		case int:
			w.SetCornerRadius(float32(v))
		case int64:
			w.SetCornerRadius(float32(v))
		}
	case "checkerboard", "check":
		if b, ok := value.(bool); ok {
			w.SetCheckerboard(b)
		}
	case "fillMode":
		if s, ok := value.(string); ok {
			switch strings.ToLower(s) {
			case "cover", "preserveaspectcrop", "crop":
				w.SetFit(ImageFitCover)
			case "fill", "stretch":
				w.SetFit(ImageFitFill)
			case "none", "preserveaspect":
				w.SetFit(ImageFitNone)
			default:
				w.SetFit(ImageFitContain)
			}
		}
	}
}

// SetOverlay installs an image drawn ON TOP of the content, at the same rect. It is how an app shows
// something that belongs to the image without being part of it — a selection, a mask, a crop frame —
// and it is deliberately just an image: the viewer draws what it is given and knows nothing about what
// it means.
func (w *Image) SetOverlay(img *graphics.Image) *Image {
	w.overlay = img
	w.MarkForRedraw()
	return w
}

// Overlay is the image drawn on top, which may be nil.
func (w *Image) Overlay() *graphics.Image { return w.overlay }

// SetFit chooses how the image maps into the widget.
func (w *Image) SetFit(f ImageFit) *Image {
	w.fit = f
	w.MarkForRedraw()
	return w
}

// Fit is the current fit.
func (w *Image) Fit() ImageFit { return w.fit }

// SetCheckerboard turns the transparency checkerboard under the image on or off. It exists because a
// transparent area drawn over the theme's background is indistinguishable from opaque black — and in an
// image editor that difference is the point.
func (w *Image) SetCheckerboard(on bool) *Image {
	w.check = on
	w.MarkForRedraw()
	return w
}

// Checkerboard reports whether the transparency pattern is drawn.
func (w *Image) Checkerboard() bool { return w.check }

// ContentRect is where the image itself lands, in WIDGET-LOCAL coordinates — the same space Paint
// receives after it translates, and the same space OnMouse*/ContainsPoint/PixelAt are given. It is
// exported because it is the geometry an app needs for the OTHER direction too (turning a pointer
// position into an image pixel), and two implementations of that mapping would eventually disagree.
func (w *Image) ContentRect() graphics.Rect {
	fr := w.FrameRect()
	return contentRectLocal(w.img, w.fit, fr.Width, fr.Height)
}

// contentRectLocal is the mapping, in one place: the fit maths applied to a widget of the given size.
func contentRectLocal(img *graphics.Image, fit ImageFit, bw, bh float32) graphics.Rect {
	if img == nil || bw <= 0 || bh <= 0 {
		return graphics.Rect{}
	}
	iw, ih := float32(img.Width()), float32(img.Height())
	if iw <= 0 || ih <= 0 {
		return graphics.Rect{}
	}
	switch fit {
	case ImageFitNone:
		return graphics.NewRect((bw-iw)/2, (bh-ih)/2, iw, ih)
	case ImageFitFill:
		return graphics.NewRect(0, 0, bw, bh)
	case ImageFitCover:
		// Scale by the LARGER ratio so both dimensions are filled, then centre the
		// overflow; the widget's own clip crops it.
		scale := bw / iw
		if s := bh / ih; s > scale {
			scale = s
		}
		dw, dh := iw*scale, ih*scale
		return graphics.NewRect((bw-dw)/2, (bh-dh)/2, dw, dh)
	default: // contain
		scale := bw / iw
		if s := bh / ih; s < scale {
			scale = s
		}
		dw, dh := iw*scale, ih*scale
		return graphics.NewRect((bw-dw)/2, (bh-dh)/2, dw, dh)
	}
}

// PreferredDimensions offers the image's own size, which is what a viewer wants when nothing constrains
// it (the layout package does not consult this — an app sets the size itself).
func (w *Image) PreferredDimensions(target interface{}, width, height float32) (float32, float32) {
	if w.img == nil {
		return width, height
	}
	return float32(w.img.Width()), float32(w.img.Height())
}

// PixelAt maps a widget-local point to an image pixel, reporting whether it is inside the image. An app
// drawing a selection or a brush uses this rather than deriving the mapping again.
func (w *Image) PixelAt(x, y float32) (px, py int, inside bool) {
	r := w.ContentRect()
	if r.Width <= 0 || r.Height <= 0 || w.img == nil {
		return 0, 0, false
	}
	ix := (x - r.X) / r.Width * float32(w.img.Width())
	iy := (y - r.Y) / r.Height * float32(w.img.Height())
	px, py = int(ix), int(iy)
	if px < 0 || py < 0 || px >= int(w.img.Width()) || py >= int(w.img.Height()) {
		return px, py, false
	}
	return px, py, true
}

func (w *Image) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !w.Visible() || rect.Width < 1 || rect.Height < 1 {
		return
	}
	canvas.Save()
	// The widget's own plate, so an empty viewer reads as a surface rather than as a
	// hole. A MASKED image gets no plate: the rounded corners are transparent, and an
	// opaque plate behind them turns the mask into a grey corner.
	if w.img == nil || w.radius <= 0 {
		plate := graphics.NewPaint()
		plate.SetColor(theme.DefaultTheme.Surface)
		canvas.DrawRect(rect, plate)
	}

	if w.img == nil {
		canvas.Restore()
		return
	}
	// Clip to the widget: a 1:1 image bigger than the viewer must not paint over its neighbours.
	canvas.ClipRect(rect, pathop.Intersect, false)

	// Local space: translate to the widget's origin first, so the same ContentRect the hit tests use is
	// what gets painted. A primitive transformed in the caller AND by the seam lands at twice the
	// origin (MIGRATION.md), so the translate happens exactly once, here.
	canvas.Translate(rect.X, rect.Y)
	content := contentRectLocal(w.img, w.fit, rect.Width, rect.Height)
	display := w.displayImageFor(rect.Width, rect.Height)
	if display != w.img {
		content = graphics.NewRect(0, 0, rect.Width, rect.Height)
	}
	if w.check && display == w.img {
		// A masked image's transparent corners would show the checkerboard through
		// the mask, which reads as a broken image rather than as transparency.
		w.paintCheckerboard(canvas, content)
	}
	canvas.DrawImageInRect(display, content, nil, nil)
	if w.overlay != nil {
		// The overlay is expected to be the same size as the content; it is scaled to the same rect, so
		// an app builds it at the document's resolution and the viewer maps it however the fit says.
		canvas.DrawImageInRect(w.overlay, content, nil, nil)
	}
	canvas.Restore()
}

// paintCheckerboard draws the alpha pattern over the image's area, sized so a square stays square on
// screen rather than scaling with the image.
func (w *Image) paintCheckerboard(canvas *graphics.Canvas, r graphics.Rect) {
	const square = 8
	dark := graphics.NewPaint()
	dark.SetColor(theme.DefaultTheme.BorderSubtle)
	canvas.DrawRect(r, dark)
	light := graphics.NewPaint()
	light.SetColor(theme.DefaultTheme.Surface)
	for ry := float32(0); ry < r.Height; ry += square {
		for rx := float32(0); rx < r.Width; rx += square {
			if int(rx/square+ry/square)%2 == 0 {
				continue
			}
			sw := min(square, r.Width-rx)
			sh := min(square, r.Height-ry)
			canvas.DrawRect(graphics.NewRect(r.X+rx, r.Y+ry, sw, sh), light)
		}
	}
}

var _ meui.Widget = (*Image)(nil)
