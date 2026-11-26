package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
)

func testImage(w, h int32) *graphics.Image {
	return graphics.NewImage(make([]uint32, w*h), w, h)
}

// TestImageContainPreservesAspect: a wide image in a square widget is letterboxed, and a tall one is
// pillarboxed. A viewer that stretched what a user opened is the first thing they stop trusting.
func TestImageContainPreservesAspect(t *testing.T) {
	w := NewImage().SetImage(testImage(200, 100))
	w.SetFrameRect(graphics.NewRect(0, 0, 100, 100))
	r := w.ContentRect()
	if r.Width != 100 || r.Height != 50 {
		t.Errorf("a 2:1 image in a 100x100 viewer landed at %vx%v, want 100x50", r.Width, r.Height)
	}
	if r.Y != 25 {
		t.Errorf("it is at y=%v, want it centred at 25", r.Y)
	}

	tall := NewImage().SetImage(testImage(100, 200))
	tall.SetFrameRect(graphics.NewRect(0, 0, 100, 100))
	r = tall.ContentRect()
	if r.Width != 50 || r.Height != 100 || r.X != 25 {
		t.Errorf("a 1:2 image landed at %v, want 50x100 centred at x=25", r)
	}
}

// TestImageNoneIsOneToOne: at 100% one image pixel is one screen pixel, and an image bigger than the
// viewer is centred rather than shrunk — the painter clips it.
func TestImageNoneIsOneToOne(t *testing.T) {
	w := NewImage().SetImage(testImage(40, 20)).SetFit(ImageFitNone)
	w.SetFrameRect(graphics.NewRect(0, 0, 100, 100))
	r := w.ContentRect()
	if r.Width != 40 || r.Height != 20 {
		t.Errorf("1:1 of a 40x20 image is %vx%v, want 40x20", r.Width, r.Height)
	}
	if r.X != 30 || r.Y != 40 {
		t.Errorf("it is centred at (%v,%v), want (30,40)", r.X, r.Y)
	}

	// Bigger than the viewer: still 1:1, centred, and the content rect goes NEGATIVE — the painter's
	// clip is what keeps it inside.
	big := NewImage().SetImage(testImage(300, 300)).SetFit(ImageFitNone)
	big.SetFrameRect(graphics.NewRect(0, 0, 100, 100))
	r = big.ContentRect()
	if r.Width != 300 || r.X != -100 {
		t.Errorf("a 300x300 image 1:1 in a 100x100 viewer is %v, want 300 wide at x=-100", r)
	}
}

// TestImageFillStretches covers the one fit that ignores the aspect ratio, which an app must ask for
// explicitly.
func TestImageFillStretches(t *testing.T) {
	w := NewImage().SetImage(testImage(10, 10)).SetFit(ImageFitFill)
	w.SetFrameRect(graphics.NewRect(0, 0, 100, 40))
	if r := w.ContentRect(); r.Width != 100 || r.Height != 40 || r.X != 0 || r.Y != 0 {
		t.Errorf("fill gave %v, want the whole widget", r)
	}
	if w.Fit().String() != "fill" {
		t.Errorf("fit name is %q", w.Fit().String())
	}
}

// TestPixelAtMapsPointerToImage is the mapping an app needs for a selection or a brush, and it must
// agree with ContentRect by construction: it is the same rect.
func TestPixelAtMapsPointerToImage(t *testing.T) {
	w := NewImage().SetImage(testImage(200, 100))
	w.SetFrameRect(graphics.NewRect(0, 0, 100, 100))
	// The image occupies x 0..100, y 25..75, so (50,50) is its centre: pixel (100,50).
	px, py, inside := w.PixelAt(50, 50)
	if !inside || px != 100 || py != 50 {
		t.Errorf("the centre maps to (%d,%d, inside=%v), want (100,50,true)", px, py, inside)
	}
	// A point in the letterbox above the image is OUTSIDE it.
	if _, _, inside := w.PixelAt(50, 5); inside {
		t.Error("a point in the letterbox reported an image pixel")
	}
	// The top-left image pixel is reachable exactly.
	if px, py, inside := w.PixelAt(0.5, 25.5); !inside || px != 1 || py != 1 {
		t.Errorf("the image's top-left maps to (%d,%d, inside=%v), want (1,1,true)", px, py, inside)
	}
}

// TestImageWithoutAnImageIsInert: an empty viewer draws its plate and reports no geometry, rather than
// crashing or dividing by zero.
func TestImageWithoutAnImageIsInert(t *testing.T) {
	w := NewImage()
	w.SetFrameRect(graphics.NewRect(0, 0, 100, 100))
	if r := w.ContentRect(); r.Width != 0 || r.Height != 0 {
		t.Errorf("an empty viewer reported %v", r)
	}
	if _, _, inside := w.PixelAt(10, 10); inside {
		t.Error("an empty viewer reported an image pixel")
	}
	if w.Image() != nil {
		t.Error("an empty viewer has an image")
	}
	// A zero-sized image (which a failed decode could produce) must not divide by zero.
	w.SetImage(testImage(0, 0))
	if r := w.ContentRect(); r.Width != 0 {
		t.Errorf("a zero-sized image reported %v", r)
	}
}

// TestCheckerboardDefaultsOn: transparency must read as transparency, not as the theme's background.
func TestCheckerboardDefaultsOn(t *testing.T) {
	w := NewImage()
	if !w.Checkerboard() {
		t.Error("the transparency checkerboard is off by default; a transparent area would be " +
			"indistinguishable from opaque background")
	}
	if w.SetCheckerboard(false).Checkerboard() {
		t.Error("SetCheckerboard(false) did not turn it off")
	}
}

// TestImageIsNotAnInputTarget records the deliberate scope: this is a viewer, so it must NOT be in the
// interactive registry and must not silently swallow pointer events meant for an ancestor.
func TestImageIsNotAnInputTarget(t *testing.T) {
	if meui.IsInteractive(NewImage()) {
		t.Error("the Image viewer is registered as an input target; it owns no input")
	}
}

// TestOverlayIsDrawnOnTopAndCleared: the overlay is how an app shows a selection without it being part
// of the image, so setting and clearing it must be reflected in what the viewer reports.
func TestOverlayIsDrawnOnTopAndCleared(t *testing.T) {
	w := NewImage().SetImage(testImage(8, 8))
	if w.Overlay() != nil {
		t.Error("a new viewer has an overlay")
	}
	ov := testImage(8, 8)
	w.SetOverlay(ov)
	if w.Overlay() != ov {
		t.Error("SetOverlay did not keep the overlay")
	}
	w.SetOverlay(nil)
	if w.Overlay() != nil {
		t.Error("SetOverlay(nil) did not clear it")
	}
}

// TestRegisterInteractiveTypeIsIdempotentAndGuarded keeps an app's widget in the touch-target registry —
// which is all IsInteractive gates — and refuses an empty name rather than adding a nameless entry.
func TestRegisterInteractiveTypeIsIdempotentAndGuarded(t *testing.T) {
	p := &TestInteractiveProbe{}
	if meui.IsInteractive(p) {
		t.Fatal("the probe is interactive before being registered")
	}
	meui.RegisterInteractiveType("TestInteractiveProbe")
	meui.RegisterInteractiveType("TestInteractiveProbe") // idempotent
	if !meui.IsInteractive(p) {
		t.Error("a registered type is not reported interactive")
	}
	meui.RegisterInteractiveType("")
	if meui.IsInteractive(&meui.BaseWidget{}) {
		t.Error("the empty name was registered, or a BaseWidget counts as interactive")
	}
}

// TestInteractiveProbe embeds BaseWidget, so it satisfies meui.Widget without a hand-written stub —
// and its TYPE NAME is what the registry is keyed by.
type TestInteractiveProbe struct {
	meui.BaseWidget
}

// TestAnImageRoundsItsCorners pins the mask: `Image { radius: 12 }` must not be
// clipped to the RECT (the software device cannot clip to a path, so a rounded
// banner would be round in GL and square offscreen), it must be masked.
func TestAnImageRoundsItsCorners(t *testing.T) {
	const w, h = 40, 40
	pix := make([]uint32, w*h)
	for i := range pix {
		pix[i] = 0xFFCC2233 // opaque red, straight alpha
	}
	src := graphics.NewImage(pix, w, h)

	img := NewImage().SetImage(src)
	img.SetCornerRadius(12)
	img.SetFrameRect(graphics.NewRect(0, 0, w, h))

	canvas, dev := graphics.NewSoftwareCanvas(w, h)
	canvas.Clear(0x00000000)
	img.Paint(canvas, graphics.NewRect(0, 0, w, h))
	out := dev.Image()

	if a := out.RGBAAt(1, 1).A; a != 0 {
		t.Fatalf("corner (1,1) has alpha %d, want 0 — the radius was not masked", a)
	}
	if a := out.RGBAAt(20, 20).A; a != 0xFF {
		t.Fatalf("centre has alpha %d, want the image", a)
	}
	// The mask must not collapse the image to its centre.
	if a := out.RGBAAt(20, 2).A; a != 0xFF {
		t.Fatalf("top edge midpoint has alpha %d, want the image", a)
	}
}
