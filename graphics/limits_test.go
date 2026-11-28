package graphics

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestImageDimensionsAreBounded(t *testing.T) {
	for _, tc := range []struct{ w, h int }{
		{0, 10}, {-1, 10}, {10, 0}, {maxImageSide + 1, 1}, {1, maxImageSide + 1},
		{100000, 100000},
	} {
		if err := checkImageDimensions(tc.w, tc.h); err == nil {
			t.Errorf("checkImageDimensions(%d, %d) accepted", tc.w, tc.h)
		}
	}
	if err := checkImageDimensions(3840, 2160); err != nil {
		t.Errorf("4K rejected: %v", err)
	}
}

func TestNewImageFromRGBARefusesAbsurdDimensions(t *testing.T) {
	if _, err := NewImageFromRGBA([]byte{0, 0, 0, 0}, 100000, 100000); err == nil {
		t.Fatal("100000x100000 RGBA buffer expected to be refused")
	}
}

func TestScaledRefusesAnAbsurdUpscale(t *testing.T) {
	src := NewImage(make([]uint32, 100*100), 100, 100)
	if got := src.Scaled(1e9); got != src {
		t.Fatalf("Scaled(1e9) allocated %dx%d instead of returning the receiver", got.Width(), got.Height())
	}
}

func TestValidPNGStillDecodes(t *testing.T) {
	im := image.NewRGBA(image.Rect(0, 0, 2, 2))
	im.Set(0, 0, color.NRGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, im); err != nil {
		t.Fatal(err)
	}
	got, err := NewImageFromBytes(buf.Bytes())
	if err != nil {
		t.Fatalf("small PNG rejected: %v", err)
	}
	if got.Width() != 2 || got.Height() != 2 {
		t.Fatalf("decoded %dx%d", got.Width(), got.Height())
	}
}
