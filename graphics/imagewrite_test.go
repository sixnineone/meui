package graphics

import "testing"

func TestSetPixel_PremultipliesAndStoresBottomUp(t *testing.T) {
	img := NewImage(make([]uint32, 2*2), 2, 2)

	img.SetPixel(1, 0, RGBA(255, 0, 0, 128)) // top-right
	// Top-left coordinate (1,0) is buffer index (h-1-0)*w+1 = 3.
	if got := img.Pixels()[3]; got != 0x80800000 {
		t.Errorf("buffer[3] = %#08x, want premultiplied 0x80800000", got)
	}
	if got := img.PixelAt(1, 0); got != RGBA(255, 0, 0, 128) {
		t.Errorf("round trip = %#08x, want %#08x", uint32(got), uint32(RGBA(255, 0, 0, 128)))
	}

	// Out-of-range writes are dropped, not panics or wrapped rows.
	img.SetPixel(-1, 0, RGB(255, 255, 255))
	img.SetPixel(0, 2, RGB(255, 255, 255))
	img.SetPixel(2, 0, RGB(255, 255, 255))
	for i, px := range img.Pixels() {
		if i != 3 && px != 0 {
			t.Errorf("buffer[%d] = %#08x, want untouched 0", i, px)
		}
	}
}

func TestPixelAt_OutOfRangeAndTransparent(t *testing.T) {
	img := NewImage(make([]uint32, 2*2), 2, 2)
	for _, p := range [][2]int{{-1, 0}, {0, -1}, {2, 0}, {0, 2}} {
		if got := img.PixelAt(p[0], p[1]); got != 0 {
			t.Errorf("PixelAt(%d,%d) = %#08x, want 0", p[0], p[1], uint32(got))
		}
	}
}

func TestBounds(t *testing.T) {
	img := NewImage(make([]uint32, 6), 3, 2)
	if got := img.Bounds(); got.X != 0 || got.Y != 0 || got.Width != 3 || got.Height != 2 {
		t.Errorf("Bounds() = %+v, want 0,0 3x2", got)
	}
	var nilImg *Image
	if got := nilImg.Bounds(); got != (Rect{}) {
		t.Errorf("nil Bounds() = %+v, want zero", got)
	}
}

// A write into an image that is already backed must reach the GPU. The texture
// upload itself is deferred to draw time (that is where the GL context is
// current), so the observable contract here is the stale marker.
func TestSetPixel_MarksBackendTextureStale(t *testing.T) {
	img := NewImage(make([]uint32, 4), 2, 2)
	if img.renderer().Dirty() {
		t.Fatal("fresh backend image reported dirty")
	}
	img.SetPixel(0, 0, RGB(1, 2, 3))
	if !img.renderer().Dirty() {
		t.Error("SetPixel left the backend texture stale-clean; the write would never reach the GPU")
	}
	// Upload is a no-op without a GL context (as in this test), so the marker
	// must STAY set: the texture still does not match the buffer. Clearing it
	// on a failed upload would silently drop the write.
	img.renderer().Upload()
	if !img.renderer().Dirty() {
		t.Error("upload without a GL context reported the texture clean")
	}
}

func TestPutImage_MarksBackendTextureStale(t *testing.T) {
	img := NewImage(make([]uint32, 4), 2, 2)
	src := NewImage(make([]uint32, 4), 2, 2)
	img.PutImage(src, 0, 0)
	if !img.renderer().Dirty() {
		t.Error("PutImage left the backend texture stale-clean")
	}
}

func TestPutImage_BlitsWithClipping(t *testing.T) {
	dst := NewImage(make([]uint32, 4*4), 4, 4)
	src := NewImage(make([]uint32, 2*2), 2, 2)
	// Distinguish the two source pixels so orientation is checkable.
	src.SetPixel(0, 0, RGB(255, 0, 0)) // src top-left
	src.SetPixel(1, 0, RGB(0, 255, 0)) // src top-right
	src.SetPixel(0, 1, RGB(0, 0, 255))
	src.SetPixel(1, 1, RGB(255, 255, 0))

	// Blit with the origin at (2,1): both source rows land on dst rows 1 and
	// 2, and the whole 2x2 fits inside the 4x4 receiver.
	dst.PutImage(src, 2, 1)

	if got := dst.PixelAt(2, 1); got != RGB(255, 0, 0) {
		t.Errorf("dst(2,1) = %#08x, want src top-left red", uint32(got))
	}
	if got := dst.PixelAt(3, 1); got != RGB(0, 255, 0) {
		t.Errorf("dst(3,1) = %#08x, want src top-right green", uint32(got))
	}
	if got := dst.PixelAt(2, 2); got != RGB(0, 0, 255) {
		t.Errorf("dst(2,2) = %#08x, want src bottom-left blue (no vertical flip)", uint32(got))
	}
	if got := dst.PixelAt(3, 2); got != RGB(255, 255, 0) {
		t.Errorf("dst(3,2) = %#08x, want src bottom-right yellow", uint32(got))
	}
	// Nothing outside the blitted rect is touched, including the column and
	// row just past it.
	for _, p := range [][2]int{{1, 1}, {2, 0}, {0, 2}, {3, 3}} {
		if got := dst.PixelAt(p[0], p[1]); got != 0 {
			t.Errorf("dst(%d,%d) = %#08x, want untouched", p[0], p[1], uint32(got))
		}
	}

	// At dx=3 only the source's left column fits; the right one is dropped
	// rather than wrapped onto the next row.
	clipped := NewImage(make([]uint32, 4*4), 4, 4)
	clipped.PutImage(src, 3, 1)
	if got := clipped.PixelAt(3, 1); got != RGB(255, 0, 0) {
		t.Errorf("clipped(3,1) = %#08x, want src top-left red", uint32(got))
	}
	if got := clipped.PixelAt(0, 2); got != 0 {
		t.Errorf("clipped(0,2) = %#08x, want untouched (clipped column must not wrap)", uint32(got))
	}

	// Negative offsets clip too, keeping the source aligned to the canvas.
	dst2 := NewImage(make([]uint32, 4*4), 4, 4)
	dst2.PutImage(src, -1, -1)
	if got := dst2.PixelAt(0, 0); got != RGB(255, 255, 0) {
		t.Errorf("dst2(0,0) = %#08x, want src bottom-right (the only pixel in range)", uint32(got))
	}
}

func TestPutImage_NilAndEmptySources(t *testing.T) {
	dst := NewImage(make([]uint32, 4), 2, 2)
	dst.PutImage(nil, 0, 0)
	dst.PutImage(NewImage(nil, 0, 0), 0, 0)
	for i, px := range dst.Pixels() {
		if px != 0 {
			t.Errorf("buffer[%d] = %#08x, want untouched", i, px)
		}
	}
}
