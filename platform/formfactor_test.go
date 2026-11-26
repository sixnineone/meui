package platform

import "testing"

// TestFormFactorUsesSizeNotOS pins the point of the axis: behaviour follows the
// SIZE of the surface, so a narrow desktop window is a phone. If this ever keys off
// runtime.GOOS the mobile layout stops being testable on a desk, which is the whole
// reason the axis exists.
func TestFormFactorUsesSizeNotOS(t *testing.T) {
	cases := []struct {
		name        string
		w, h, scale float32
		want        FormFactor
	}{
		{"a phone in portrait", 360, 640, 1, FormFactorPhone},
		{"a small phone", 320, 480, 1, FormFactorPhone},
		{"a tablet in portrait", 800, 1200, 1, FormFactorTablet},
		{"a desktop window", 1280, 720, 1, FormFactorDesktop},
		{"a wide desktop window", 1600, 900, 1, FormFactorDesktop},
		// A high-density phone reports PHYSICAL pixels, so the scale has to be applied
		// before the comparison or a 3x phone would classify as a desktop. The
		// direction matters: the toolkit DIVIDES by scale for local coordinates, so it
		// divides here too.
		{"a 3x phone in physical pixels", 1080, 1920, 3, FormFactorPhone},
		{"a 2x tablet in physical pixels", 1600, 2400, 2, FormFactorTablet},
		{"a phone at 1x that is wide but short", 900, 400, 1, FormFactorPhone},
	}
	for _, c := range cases {
		if got := FormFactorFor(c.w, c.h, c.scale); got != c.want {
			t.Errorf("%s: %.0fx%.0f@%.0fx = %v, want %v", c.name, c.w, c.h, c.scale, got, c.want)
		}
	}
}

// TestFormFactorDegenerateSizes covers the sizes a window reports before it has been
// shown, which must not classify as a phone — a zero-sized surface would otherwise
// turn the touch floor on for a desktop app at startup.
func TestFormFactorDegenerateSizes(t *testing.T) {
	for _, c := range [][3]float32{{0, 0, 1}, {0, 640, 1}, {360, 0, 1}, {-1, -1, 1}} {
		if got := FormFactorFor(c[0], c[1], c[2]); got != FormFactorDesktop {
			t.Errorf("%v classified as %v, want desktop", c, got)
		}
	}
}
