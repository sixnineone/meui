package graphics

import "testing"

// The software Device must actually rasterise, not merely satisfy the interface:
// it exists so a widget can be drawn on the CPU with the SAME draw code and the
// result uploaded as a texture, instead of using a GPU offscreen pass whose state
// is shared with every window.
func TestSoftDeviceRasterisesAndRepacks(t *testing.T) {
	dev := NewSoftDevice(20, 10)
	dev.Clear(0xFF000000)
	dev.DrawRect(0, 0, 20, 10, nil) // nil paint draws the default (opaque white)

	img := dev.Image()
	if got := img.RGBAAt(10, 5); got.R != 0xff || got.G != 0xff || got.B != 0xff || got.A != 0xff {
		t.Fatalf("rasterised pixel = %v, want opaque white", got)
	}

	// The repacked form must be the seam's a<<24|r<<16|g<<8|b.
	px := dev.Pixels()
	if len(px) != 20*10 {
		t.Fatalf("Pixels() length = %d, want 200", len(px))
	}
	if got, want := px[5*20+10], uint32(0xFFFFFFFF); got != want {
		t.Fatalf("packed pixel = %#x, want %#x", got, want)
	}
}

// Clipping is implemented by this backend (draw2d has none), so it is worth
// pinning: a draw outside the clip must leave the surface untouched.
func TestSoftDeviceClipRectExcludes(t *testing.T) {
	dev := NewSoftDevice(20, 10)
	// Transparent, so an untouched pixel is distinguishable from a drawn one.
	dev.Clear(0x00000000)
	dev.ClipRect(NewRect(0, 0, 5, 5))
	dev.DrawRect(10, 0, 10, 10, nil) // entirely outside the clip

	if got := dev.Image().RGBAAt(15, 5); got.A != 0 || got.R != 0 || got.G != 0 || got.B != 0 {
		t.Fatalf("pixel outside the clip was drawn: %v", got)
	}
	dev.DrawRect(0, 0, 4, 4, nil) // inside the clip
	if got := dev.Image().RGBAAt(2, 2); got.R != 0xff {
		t.Fatalf("pixel inside the clip was not drawn: %v", got)
	}
}

// A rect that STRADDLES the clip must be TRIMMED, not drawn whole.
//
// Culling alone — which is what this backend used to do — passes the entirely-outside case
// above and still lets a body cross the boundary. That is not academic: it is how candles
// escaped a chart's clipped price pane and reappeared in the volume band and the time axis.
// The clip is a CLIP; `clipped` is only its fast path.
func TestSoftDeviceClipRectTrimsAStraddlingRect(t *testing.T) {
	dev := NewSoftDevice(20, 20)
	dev.Clear(0x00000000)
	dev.ClipRect(NewRect(0, 0, 10, 10))
	dev.DrawRect(5, 5, 10, 10, nil) // straddles the corner: 5..10 is inside, 10..15 is not

	anyDrawn := false
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			drawn := dev.Image().RGBAAt(x, y).A != 0
			// The rect is 5..15; the clip is 0..10. Only their overlap may be painted.
			want := x >= 5 && x < 15 && y >= 5 && y < 15 && x < 10 && y < 10
			if drawn != want {
				t.Errorf("pixel (%d,%d): drawn=%v, want %v (rect 5..15, clip 0..10)", x, y, drawn, want)
			}
			anyDrawn = anyDrawn || drawn
		}
	}
	if !anyDrawn {
		t.Fatal("nothing was drawn at all — the test measures nothing")
	}
}

// A LINE that crosses the clip must be cut at the boundary, for the same reason: culling alone
// draws the whole segment, which is how a candle wick reached past the pane edge.
func TestSoftDeviceClipRectCutsAStraddlingLine(t *testing.T) {
	dev := NewSoftDevice(20, 20)
	dev.Clear(0x00000000)
	dev.ClipRect(NewRect(0, 0, 10, 10))
	dev.DrawLine(0, 0, 20, 20, nil) // runs from inside the clip out past its corner

	anyDrawn := false
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			if dev.Image().RGBAAt(x, y).A == 0 {
				continue
			}
			anyDrawn = true
			// One pixel of tolerance: a stroked line has a WIDTH, so the cut endpoint's cap
			// covers its immediate neighbourhood — the antialiased boundary, not a leak. What
			// must not happen is the line running on to the edge of the surface.
			if x > 10 || y > 10 {
				t.Fatalf("the line painted (%d,%d), beyond the clip (0..10) by more than its own width", x, y)
			}
		}
	}
	if !anyDrawn {
		t.Fatal("the line drew nothing at all — the test measures nothing")
	}
}

// RenderOffscreen is the entry point the chart uses: it must run the same draw
// callback and hand back a seam Image with no GL involved.
func TestSoftDeviceRenderOffscreenReturnsImage(t *testing.T) {
	dev := NewSoftDevice(8, 8)
	ran := false
	img, err := dev.RenderOffscreen(16, 12, func(c *Canvas) { ran = true })
	if err != nil {
		t.Fatalf("RenderOffscreen error: %v", err)
	}
	if !ran {
		t.Fatal("RenderOffscreen did not run the draw callback")
	}
	if img == nil || img.Width() != 16 || img.Height() != 12 {
		t.Fatalf("RenderOffscreen returned %v, want a 16x12 image", img)
	}
}

// The seam's Image takes bottom-up pixel rows (ReadScreenPixels documents them as
// "bottom-up as GL delivers them"). A software surface is top-down, so this pins
// the flip: without it a cached chart renders upside down, which shows up as the
// oscillator panes appearing above the candles and the live layer's hit regions
// no longer matching what is drawn.
func TestSoftDevicePixelsAreBottomUp(t *testing.T) {
	dev := NewSoftDevice(4, 4)
	dev.Clear(0x00000000)
	dev.DrawRect(0, 0, 4, 1, nil) // the TOP row of the surface, opaque white

	px := dev.Pixels()
	if len(px) != 16 {
		t.Fatalf("Pixels() length = %d, want 16", len(px))
	}
	// Top row of the surface must be the LAST row of the buffer.
	if px[12] != 0xFFFFFFFF {
		t.Fatalf("last buffer row = %#x, want the drawn top row %#x", px[12], uint32(0xFFFFFFFF))
	}
	if px[0] != 0 {
		t.Fatalf("first buffer row = %#x, want an untouched (bottom) row", px[0])
	}
}

// The panel positions its children with a canvas translate (every child logs at origin (0,0)
// with only a size, P5fy), so if the software device ignores the translation the whole tree
// stacks at the top of the surface and any pixel mask reads the wrong rows. This is the
// known-positive for that: draw at (0,0) after Translate(0,10) and require it to land at y=10.
func TestSoftDeviceAppliesTranslate(t *testing.T) {
	const w, h = 20, 20
	dev := NewSoftDevice(w, h)
	dev.Clear(0x00000000)
	dev.Save()
	dev.Translate(0, 10)
	dev.DrawRect(0, 0, 20, 4, nil) // opaque white at y=10..13 once translated
	dev.Restore()

	px := dev.Pixels() // rows are emitted BOTTOM-UP
	rowHas := func(topY int) bool {
		for x := 0; x < w; x++ {
			if px[(h-1-topY)*w+x] == 0xFFFFFFFF {
				return true
			}
		}
		return false
	}
	if !rowHas(11) {
		t.Errorf("Translate(0,10) not applied: row y=11 is empty (the rect is not where the canvas put it)")
	}
	if rowHas(1) {
		t.Errorf("Translate(0,10) ignored: the rect landed at y=1")
	}
}

// TestTheSoftwareDeviceRoundsCorners pins a silent lie: DrawRoundedRect used to
// delegate to DrawRect, so every radius the toolkit asked for was SQUARE in every
// offscreen render — the first headless QML screenshot of `radius: 8` showed a
// sharp corner, and so would every cached chart or card.
func TestTheSoftwareDeviceRoundsCorners(t *testing.T) {
	dev := NewSoftDevice(40, 40)
	dev.Clear(0xFF000000)
	p := NewPaint()
	p.SetColor(0xFFFFFFFF)
	dev.DrawRoundedRect(NewRect(0, 0, 40, 40), 12, 12, p)

	corner := dev.Image().RGBAAt(1, 1)
	if corner.R > 0x40 {
		t.Fatalf("corner (1,1) is lit (%v): the radius was not applied", corner)
	}
	centre := dev.Image().RGBAAt(20, 20)
	if centre.R < 0xC0 {
		t.Fatalf("centre (20,20) is %v, want the fill", centre)
	}
	top := dev.Image().RGBAAt(20, 0)
	if top.R < 0xC0 {
		t.Fatalf("top edge midpoint (20,0) is %v, want the fill", top)
	}
}

// TestTheSoftwareDeviceRampsAGradient pins the other half of the same gap: the
// software device ignored Paint.Gradient() entirely, so a sky that looked like a
// sky in the window came out as one flat colour offscreen.
func TestTheSoftwareDeviceRampsAGradient(t *testing.T) {
	dev := NewSoftDevice(40, 40)
	dev.Clear(0xFF000000)
	g := Gradient{Kind: GradientLinear, Start: Point{X: 0, Y: 0}, End: Point{X: 0, Y: 40}}
	g.Stops = []GradientStop{{Offset: 0, Color: 0xFFFF0000}, {Offset: 1, Color: 0xFF0000FF}}
	p := NewPaint()
	p.SetGradient(g)
	dev.DrawRect(0, 0, 40, 40, p)

	top := dev.Image().RGBAAt(20, 2)
	bottom := dev.Image().RGBAAt(20, 37)
	if top.R < 0xC0 || top.B > 0x40 {
		t.Fatalf("top row is %v, want red", top)
	}
	if bottom.B < 0xC0 || bottom.R > 0x40 {
		t.Fatalf("bottom row is %v, want blue", bottom)
	}
}

// TestTheSoftwareDeviceCompositesATranslucentFill pins the premultiplication the
// raster painter requires. The device handed freetype a STRAIGHT color and an
// alpha, and the painter's `dst*(1-a) + src` overflowed: a 0.8-alpha mint wash over
// cream came out (15,29,9). Every shadow, hover wash and opacity in the offscreen
// renderer was stained by it, and a screenshot was the only way to see it.
func TestTheSoftwareDeviceCompositesATranslucentFill(t *testing.T) {
	dev := NewSoftDevice(40, 40)
	dev.Clear(0xFFFCF4DC) // cream
	p := NewPaint()
	p.SetColor(0xFFDCECDC) // mint
	p.SetGlobalAlpha(0.8)
	dev.DrawRect(10, 10, 20, 20, p)

	got := dev.Image().RGBAAt(20, 20)
	// 0.8*mint + 0.2*cream per channel; blue is 220 in both.
	if !near(got.R, 226) || !near(got.G, 238) || !near(got.B, 220) {
		t.Fatalf("translucent fill composited to %v, want ~(226,238,220)", got)
	}
}

func near(v uint8, want int) bool {
	d := int(v) - want
	if d < 0 {
		d = -d
	}
	return d <= 2
}

// TestTheSoftwareDeviceDrawsImagesThroughTheTransform pins the image half of the
// transform contract: draw2d draws an image from an explicit rect and ignores the
// canvas matrix, so the software device drew every image at the CANVAS ORIGIN while
// its plate, border and text were positioned correctly. A docs cover landed over the
// top bar and the sidebar while the space reserved for it stayed empty.
func TestTheSoftwareDeviceDrawsImagesThroughTheTransform(t *testing.T) {
	const iw, ih = 8, 8
	pix := make([]uint32, iw*ih)
	for i := range pix {
		pix[i] = 0xFFFF0000
	}
	img := NewImage(pix, iw, ih)

	dev := NewSoftDevice(80, 60)
	dev.Clear(0x00000000)
	c := NewCanvas(dev)
	c.Save()
	c.Translate(30, 20)
	c.DrawImageInRect(img, NewRect(0, 0, 16, 16), nil, nil)
	c.Restore()

	px := dev.Image().RGBAAt(35, 25)
	if px.R < 0xC0 {
		t.Fatalf("image pixel at (35,25) = %v, want the translated red", px)
	}
	if origin := dev.Image().RGBAAt(3, 3); origin.R > 0x40 {
		t.Fatalf("image painted at the canvas origin (%v): the draw ignored the transform", origin)
	}
}

// TestTheSoftwareDeviceClipsAnImageToTheClip pins the crop: draw2d draws an image
// WHOLE, so a cover-fit image (a 900x240 banner in a 72x72 avatar) painted its full
// width across the card. The device now restores every pixel the clip excludes.
func TestTheSoftwareDeviceClipsAnImageToTheClip(t *testing.T) {
	pix := make([]uint32, 20*20)
	for i := range pix {
		pix[i] = 0xFFFF0000
	}
	img := NewImage(pix, 20, 20)

	dev := NewSoftDevice(80, 40)
	dev.Clear(0xFFFFFFFF)
	c := NewCanvas(dev)
	c.Save()
	c.ClipRect(NewRect(0, 0, 20, 20), 0, false)
	c.DrawImageInRect(img, NewRect(0, 0, 40, 40), nil, nil)
	c.Restore()

	if px := dev.Image().RGBAAt(10, 20); px.R < 0xC0 {
		t.Fatalf("inside the clip is %v, want the image", px)
	}
	if px := dev.Image().RGBAAt(30, 20); px.R < 0xF0 || px.G < 0xF0 {
		t.Fatalf("outside the clip is %v, want the untouched background", px)
	}
}

// TestTheSoftwareDeviceClipsShapesToTheClip pins the clip for SHAPES, not just images.
// draw2d has no clip, and the device only culled fully-outside draws, so a clipped
// panel's children painted straight over the panel's edges — the parity scene showed a
// rounded clip box with a child spilling 100px past it, while GL clipped it exactly.
func TestTheSoftwareDeviceClipsShapesToTheClip(t *testing.T) {
	dev := NewSoftDevice(80, 60)
	dev.Clear(0xFFFFFFFF)
	c := NewCanvas(dev)
	p := NewPaint()
	p.SetColor(0xFFFF0000)

	c.Save()
	c.ClipRect(NewRect(10, 10, 20, 20), 0, false)
	c.DrawRect(NewRect(0, 0, 80, 60), p)
	c.Restore()

	if px := dev.Image().RGBAAt(15, 15); px.R < 0xE0 || px.G > 0x20 {
		t.Fatalf("inside the clip is %v, want the red fill", px)
	}
	for _, pt := range [][2]int{{50, 30}, {5, 5}, {15, 40}, {40, 15}} {
		if px := dev.Image().RGBAAt(pt[0], pt[1]); px.R < 0xF0 || px.G < 0xF0 {
			t.Fatalf("outside the clip at %v is %v, want the untouched background", pt, px)
		}
	}

	// The clip ends with its scope: a later draw fills the whole surface again.
	c.DrawRect(NewRect(0, 0, 80, 60), p)
	if px := dev.Image().RGBAAt(50, 30); px.R < 0xE0 {
		t.Fatalf("after Restore the clip still applies: %v", px)
	}
}
