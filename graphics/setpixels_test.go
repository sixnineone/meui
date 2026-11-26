package graphics

import "testing"

// SetPixels must reach the BACKEND, not just the seam: the backend was handed the
// original slice when it was created, so a seam-only swap leaves it re-uploading the
// pixels from before — which is a cached layer that never changes after its first
// render. The image API's own Pixels() would not catch that, so this asserts on the
// renderer's buffer.
func TestSetPixelsReachesTheBackend(t *testing.T) {
	const w, h = 4, 3
	first := make([]uint32, w*h)
	for i := range first {
		first[i] = 0xFF112233
	}
	second := make([]uint32, w*h)
	for i := range second {
		second[i] = 0xFF445566
	}
	img := NewImage(first, w, h)
	if img == nil {
		t.Fatal("no image")
	}
	// Create the backend view FIRST, holding the original buffer — which is what the
	// GL path does on the image's first draw. Only then swap.
	if got := img.renderer(); got == nil {
		t.Fatal("no backend view")
	}
	img.SetPixels(second)
	if got := img.renderer().Pixels(); len(got) != len(second) || got[0] != second[0] {
		t.Fatalf("the backend still holds %08x, want %08x: SetPixels did not reach it",
			got[0], second[0])
	}
}
