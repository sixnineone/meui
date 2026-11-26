package graphics

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testPNG encodes a 2x2 image whose four pixels differ in both colour and
// alpha, so a vertical flip or a missing premultiply is visible.
//
// Rows are top-down: (0,0) red 50%, (1,0) green opaque, (0,1) blue opaque,
// (1,1) white 25%.
func testPNG(t *testing.T) []byte {
	t.Helper()
	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	src.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 128})
	src.SetNRGBA(1, 0, color.NRGBA{G: 255, A: 255})
	src.SetNRGBA(0, 1, color.NRGBA{B: 255, A: 255})
	src.SetNRGBA(1, 1, color.NRGBA{R: 255, G: 255, B: 255, A: 64})
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestNewImageFromBytes_DecodesAndPremultiplies(t *testing.T) {
	img, err := NewImageFromBytes(testPNG(t))
	if err != nil {
		t.Fatalf("NewImageFromBytes: %v", err)
	}
	if img.Width() != 2 || img.Height() != 2 {
		t.Fatalf("size = %dx%d, want 2x2", img.Width(), img.Height())
	}

	// Straight-alpha colours must come back as encoded, in top-left
	// coordinates, whatever the internal storage layout is.
	for _, tc := range []struct {
		x, y int
		want Color
	}{
		{0, 0, RGBA(255, 0, 0, 128)},
		{1, 0, RGB(0, 255, 0)},
		{0, 1, RGB(0, 0, 255)},
		{1, 1, RGBA(255, 255, 255, 64)},
	} {
		if got := img.PixelAt(tc.x, tc.y); got != tc.want {
			t.Errorf("PixelAt(%d,%d) = %#08x, want %#08x", tc.x, tc.y, uint32(got), uint32(tc.want))
		}
	}

	// The buffer itself is premultiplied. Index 0 is the BOTTOM row's left
	// pixel — the opaque blue source pixel at top-left coordinate (0, 1) —
	// so it stays 0xFF0000FF.
	raw := img.Pixels()
	if got := raw[0]; got != 0xFF0000FF {
		t.Errorf("raw[0] = %#08x, want the opaque blue bottom-left pixel", got)
	}
	// Index 2 is the TOP row's left pixel, the half-transparent red one: 128
	// alpha with red scaled down to 128, not left at 255.
	if got := raw[2]; got != 0x80800000 {
		t.Errorf("raw[2] = %#08x, want premultiplied 0x80800000", got)
	}
}

// The seam stores pixels bottom-up (GL readback order) and the renderer samples
// with v flipped to compensate. A decoded grid therefore has to land in the
// buffer reversed row-wise, or every loaded image would draw upside down.
func TestNewImageFromBytes_StoresBottomUp(t *testing.T) {
	img, err := NewImageFromBytes(testPNG(t))
	if err != nil {
		t.Fatalf("NewImageFromBytes: %v", err)
	}
	// Top-left of the source (half-transparent red) is the LAST row of the
	// bottom-up buffer — but the FIRST row of the file.
	if got := img.Pixels()[2]; got>>24 != 128 || got>>16&0xFF != 128 {
		t.Errorf("buffer[2] = %#08x, want the source top-left pixel (premultiplied red 50%%)", got)
	}
	// The bottom-up buffer's own first row must be the source's bottom row.
	if got := img.Pixels()[0]; got != 0xFF0000FF {
		t.Errorf("buffer[0] = %#08x, want the source bottom-left pixel (opaque blue)", got)
	}
}

func TestNewImageFromBytes_DecodesJPEG(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 3))
	for i := range src.Pix {
		src.Pix[i] = 0xFF
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, src, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	img, err := NewImageFromBytes(buf.Bytes())
	if err != nil {
		t.Fatalf("NewImageFromBytes(jpeg): %v", err)
	}
	if img.Width() != 4 || img.Height() != 3 {
		t.Fatalf("size = %dx%d, want 4x3", img.Width(), img.Height())
	}
}

func TestNewImageFromBytes_RejectsGarbage(t *testing.T) {
	if _, err := NewImageFromBytes([]byte("not an image")); err == nil {
		t.Fatal("NewImageFromBytes(garbage) = nil error, want decode failure")
	}
}

func TestNewImageFromFilePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "swatch.png")
	if err := os.WriteFile(path, testPNG(t), 0o644); err != nil {
		t.Fatalf("write png: %v", err)
	}

	img, err := NewImageFromFilePathOrURL(path, 1)
	if err != nil {
		t.Fatalf("NewImageFromFilePathOrURL: %v", err)
	}
	if img.Width() != 2 || img.Height() != 2 {
		t.Fatalf("size = %dx%d, want 2x2", img.Width(), img.Height())
	}
	if got := img.PixelAt(0, 1); got != RGB(0, 0, 255) {
		t.Errorf("PixelAt(0,1) = %#08x, want blue", uint32(got))
	}

	// A scale of 0 means "as decoded" as well.
	if img, err = NewImageFromFilePathOrURL(path, 0); err != nil || img.Width() != 2 {
		t.Fatalf("scale 0: %v, %dx%d", err, img.Width(), img.Height())
	}

	// The scale parameter must actually resample rather than be ignored.
	img, err = NewImageFromFilePathOrURL(path, 2)
	if err != nil {
		t.Fatalf("scale 2: %v", err)
	}
	if img.Width() != 4 || img.Height() != 4 {
		t.Fatalf("scaled size = %dx%d, want 4x4", img.Width(), img.Height())
	}
}

func TestNewImageFromFilePath_Missing(t *testing.T) {
	if _, err := NewImageFromFilePathOrURL(filepath.Join(t.TempDir(), "nope.png"), 1); err == nil {
		t.Fatal("missing file = nil error, want failure")
	}
}

// The seam does no network I/O. Remote inputs must fail loudly and point the
// caller at NewImageFromBytes instead of being silently ignored or hung on.
func TestNewImageFromFilePathOrURL_RejectsRemoteURL(t *testing.T) {
	_, err := NewImageFromFilePathOrURL("https://example.invalid/a.png", 1)
	if err == nil {
		t.Fatal("remote URL = nil error, want refusal")
	}
	if !strings.Contains(err.Error(), "NewImageFromBytes") {
		t.Errorf("error %q does not point at NewImageFromBytes", err)
	}
}

func TestScaled_ReturnsReceiverWhenUnscaled(t *testing.T) {
	img, err := NewImageFromBytes(testPNG(t))
	if err != nil {
		t.Fatalf("NewImageFromBytes: %v", err)
	}
	if got := img.Scaled(1); got != img {
		t.Error("Scaled(1) allocated a new image, want the receiver")
	}
}
