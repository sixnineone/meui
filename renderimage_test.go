package meui

import "testing"

// TestFlipVerticalTurnsABottomUpReadbackTopDown pins the orientation contract of
// RenderToImage: GL readback is bottom-up and the API returns an image that
// callers write straight to a PNG.
//
// The regression it guards is silent — an upside-down capture still looks like a
// plausible picture, so it survived until the parity corpus was run on Android
// (MOBILE-PLAN.md P5.4), where a scene's two rectangles came back in the
// opposite order and the delta was 85.3 as-is against 0.0 when flipped.
func TestFlipVerticalTurnsABottomUpReadbackTopDown(t *testing.T) {
	// 3x2, bottom-up: the first row is the BOTTOM of the picture.
	pix := []uint32{
		1, 2, 3, // bottom
		4, 5, 6, // top
	}
	got := flipVertical(pix, 3, 2)
	want := []uint32{
		4, 5, 6, // top is first once flipped
		1, 2, 3,
	}
	if len(got) != len(want) {
		t.Fatalf("flipVertical returned %d elements, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("flipVertical = %v, want %v", got, want)
		}
	}
	if &got[0] == &pix[0] {
		t.Error("flipVertical returned the input buffer; it must copy, because the " +
			"input may be a cached FBO readback the renderer still owns")
	}
}

// TestFlipVerticalLeavesAnImpossibleBufferAlone: a buffer that cannot hold w*h
// pixels is returned untouched rather than panicking. The callers are all
// "best effort" paths, and a panic inside a capture would take the app down.
func TestFlipVerticalLeavesAnImpossibleBufferAlone(t *testing.T) {
	pix := []uint32{1, 2}
	for _, c := range []struct{ w, h int32 }{{4, 4}, {0, 4}, {4, 0}, {-1, 2}} {
		got := flipVertical(pix, c.w, c.h)
		if len(got) != len(pix) || got[0] != 1 || got[1] != 2 {
			t.Errorf("flipVertical(%v, %d, %d) = %v, want the input untouched", pix, c.w, c.h, got)
		}
	}
}
