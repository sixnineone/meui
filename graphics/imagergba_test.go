package graphics

import "testing"

// TestNewImageFromRGBAPacksTheSeamsWord pins all three conversions at once, because getting any one of
// them wrong still produces a plausible-looking picture: an opaque colour packs as 0xAARRGGBB, and a
// transparent one is premultiplied exactly ONCE (premultiplying twice turns a plausible image into a
// subtly dark one, which is the kind of bug that ships).
func TestNewImageFromRGBAPacksTheSeamsWord(t *testing.T) {
	// Two pixels, top-down: opaque red, then half-transparent white.
	rgba := []byte{
		255, 0, 0, 255,
		255, 255, 255, 128,
	}
	img, err := NewImageFromRGBA(rgba, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if img.Width() != 2 || img.Height() != 1 {
		t.Fatalf("image is %dx%d", img.Width(), img.Height())
	}
	// The seam is bottom-up, so a one-row buffer is its own flip; for two rows the flip is asserted
	// below. Read the packed words back through the backend view the seam exposes.
	pix := img.pix
	if len(pix) != 2 {
		t.Fatalf("packed %d words, want 2", len(pix))
	}
	// 0xAARRGGBB puts RED in the byte below alpha, so opaque red is 0xFFFF0000 — and the seam's own
	// premultiplied() helper agrees. (My first version of this test expected 0xFF0000FF, i.e. I had
	// written the packing as AABBGGRR; the code was right.)
	if want := uint32(0xFFFF0000); pix[0] != want {
		t.Errorf("opaque red packed as %#08x, want %#08x (0xAARRGGBB)", pix[0], want)
	}
	// And the half-transparent white is PREMULTIPLIED, so its channels are halved: 0x80808080, not
	// 0x80FFFFFF. That difference is the whole reason this function exists.
	if want := uint32(0x80808080); pix[1] != want {
		t.Errorf("half-transparent white packed as %#08x, want %#08x (premultiplied once)", pix[1], want)
	}
}

// TestNewImageFromRGBAFlipsRows is the second of the three conversions: a rendered buffer is top-down
// and the seam is bottom-up, so the first source row must end up in the LAST destination row.
func TestNewImageFromRGBAFlipsRows(t *testing.T) {
	// A 1x2 buffer: row 0 pure red, row 1 pure blue.
	rgba := []byte{
		255, 0, 0, 255,
		0, 0, 255, 255,
	}
	img, err := NewImageFromRGBA(rgba, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	pix := img.pix
	if len(pix) != 2 {
		t.Fatalf("packed %d words", len(pix))
	}
	// Destination row 0 is the BOTTOM, so it holds the source's LAST row (blue).
	if want := uint32(0xFF0000FF); pix[0] != want {
		t.Errorf("the bottom row is %#08x, want the source's last row (blue %#08x)", pix[0], want)
	}
	if want := uint32(0xFFFF0000); pix[1] != want {
		t.Errorf("the top row is %#08x, want the source's first row (red %#08x)", pix[1], want)
	}
}

// TestNewImageFromRGBAValidates keeps a short buffer from being read past its end: a decoder that
// returned fewer bytes than its dimensions claim is a real failure mode, and a crash in the display
// path is worse than a message.
func TestNewImageFromRGBAValidates(t *testing.T) {
	if _, err := NewImageFromRGBA(nil, 4, 4); err == nil {
		t.Error("a nil buffer was accepted for a 4x4 image")
	}
	if _, err := NewImageFromRGBA(make([]byte, 8), 4, 4); err == nil {
		t.Error("an 8-byte buffer was accepted for a 4x4 image")
	}
	if _, err := NewImageFromRGBA(nil, 0, 4); err == nil {
		t.Error("a zero-width image was accepted")
	}
}
