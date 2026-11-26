package widgets

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

// The window outline is one theme-driven stroke around the whole window,
// chrome included. These pin the resolution order (explicit setter, chrome
// token, theme Border role), the placement on the outermost pixels and the
// "width 0 = no outline" case.
//
// Coverage is asserted loosely on purpose: the GL renderer paints a stroke
// as an inside band at full coverage, the software device a centred band at
// partial coverage, and draw2d rounds thin line widths its own way. What
// must hold on both is WHERE the outline lands and that the interior stays
// untouched. The rounded shape cannot be seen through the software device
// at all (it draws every rounded rect square), so it is left to the demo
// capture.

const (
	borderTestW = 120
	borderTestH = 80
)

// borderTestTheme installs a theme with a known Border role and chrome
// tokens, and returns the ink the frame should fall back to.
func borderTestTheme(t *testing.T, chrome theme.Chrome) graphics.Color {
	t.Helper()
	saved := theme.DefaultTheme
	t.Cleanup(func() { theme.DefaultTheme = saved })
	if chrome.Height == 0 {
		chrome.Height = FrameBarHeight
	}
	theme.DefaultTheme = &theme.Theme{
		Border: graphics.Color(0xFF445566),
		Chrome: chrome,
	}
	return theme.DefaultTheme.Border
}

func borderFramePixels(t *testing.T, f *WindowFrame) func(x, y int) uint32 {
	t.Helper()
	dev := graphics.NewSoftDevice(borderTestW, borderTestH)
	f.PaintBorder(graphics.NewCanvas(dev), borderTestW, borderTestH)
	px := dev.Pixels()
	// Pixels() is bottom-up (the seam's convention).
	return func(x, y int) uint32 { return px[(borderTestH-1-y)*borderTestW+x] }
}

// borderPixel returns a pixel's coverage and its straight colour.
// image.RGBA stores premultiplied samples, so an antialiased edge divides
// the channels by the coverage before they can be compared with the ink.
func borderPixel(at func(x, y int) uint32, x, y int) (uint32, graphics.Color) {
	p := at(x, y)
	alpha := p >> 24
	if alpha == 0 {
		return 0, 0
	}
	r := (p >> 16 & 0xFF) * 255 / alpha
	g := (p >> 8 & 0xFF) * 255 / alpha
	b := (p & 0xFF) * 255 / alpha
	return alpha, graphics.Color(0xFF000000 | r<<16 | g<<8 | b)
}

func colorNear(a, b graphics.Color) bool {
	for _, shift := range []uint{16, 8, 0} {
		av, bv := int(a>>shift&0xFF), int(b>>shift&0xFF)
		if av-bv > 2 || bv-av > 2 {
			return false
		}
	}
	return true
}

// wantOutline asserts the outline covers the pixel with the given ink.
func wantOutline(t *testing.T, at func(x, y int) uint32, x, y int, ink graphics.Color) {
	t.Helper()
	alpha, got := borderPixel(at, x, y)
	if alpha == 0 {
		t.Errorf("pixel (%d,%d) is not covered, want the outline ink %#08x", x, y, uint32(ink))
		return
	}
	if !colorNear(got, ink) {
		t.Errorf("pixel (%d,%d) = %#08x, want the outline ink %#08x", x, y, uint32(got), uint32(ink))
	}
}

// wantClear asserts nothing was painted at the pixel.
func wantClear(t *testing.T, at func(x, y int) uint32, x, y int) {
	t.Helper()
	if alpha, got := borderPixel(at, x, y); alpha != 0 {
		t.Errorf("pixel (%d,%d) = %#08x, want untouched", x, y, uint32(got))
	}
}

func TestWindowFrameBorderFollowsTheme(t *testing.T) {
	ink := borderTestTheme(t, theme.Chrome{BorderWidth: 1})
	f := NewWindowFrame()
	if got := f.BorderWidth(); got != 1 {
		t.Fatalf("BorderWidth() = %v, want the chrome token's 1", got)
	}
	at := borderFramePixels(t, f)
	// A 1px outline lands on the outermost row/column on every side, the
	// title bar's top row included.
	for _, p := range []struct{ x, y int }{{0, 0}, {borderTestW / 2, 0}, {borderTestW - 1, 0},
		{0, borderTestH / 2}, {borderTestW - 1, borderTestH / 2},
		{0, borderTestH - 1}, {borderTestW / 2, borderTestH - 1}, {borderTestW - 1, borderTestH - 1}} {
		wantOutline(t, at, p.x, p.y, ink)
	}
	wantClear(t, at, borderTestW/2, borderTestH/2)
	// One pixel wide: the second row/column is the window's own content.
	wantClear(t, at, 1, borderTestH/2)
	wantClear(t, at, borderTestW/2, 1)
}

func TestWindowFrameBorderTokenOverridesThemeBorder(t *testing.T) {
	borderTestTheme(t, theme.Chrome{BorderWidth: 3, BorderInk: 0xFF112233})
	f := NewWindowFrame()
	if got := f.BorderWidth(); got != 3 {
		t.Fatalf("BorderWidth() = %v, want the chrome token's 3", got)
	}
	at := borderFramePixels(t, f)
	wantOutline(t, at, 0, borderTestH/2, 0xFF112233)
	// Past the band the content shows through.
	wantClear(t, at, 3, borderTestH/2)
	wantClear(t, at, 3, 3)
}

func TestWindowFrameBorderSetterOverridesTheme(t *testing.T) {
	borderTestTheme(t, theme.Chrome{BorderWidth: 1, BorderInk: 0xFF112233})
	f := NewWindowFrame()
	f.SetBorder(0xFFAA0000, 2)
	if got := f.BorderWidth(); got != 2 {
		t.Fatalf("BorderWidth() = %v, want the setter's 2", got)
	}
	at := borderFramePixels(t, f)
	wantOutline(t, at, 0, borderTestH/2, 0xFFAA0000)
	wantClear(t, at, 2, borderTestH/2)
}

func TestWindowFrameBorderWidthZeroPaintsNothing(t *testing.T) {
	borderTestTheme(t, theme.Chrome{BorderWidth: 0}) // macOS: no window outline
	f := NewWindowFrame()
	if got := f.BorderWidth(); got != 0 {
		t.Fatalf("BorderWidth() = %v, want 0", got)
	}
	at := borderFramePixels(t, f)
	for _, p := range []struct{ x, y int }{{0, 0}, {0, borderTestH / 2}, {borderTestW - 1, borderTestH / 2}, {borderTestW - 1, borderTestH - 1}} {
		wantClear(t, at, p.x, p.y)
	}

	// An explicit 0 also wins over a theme that wants an outline.
	borderTestTheme(t, theme.Chrome{BorderWidth: 2})
	f.SetBorderWidth(0)
	wantClear(t, borderFramePixels(t, f), 0, borderTestH/2)
}

// PaintOverlay draws the outline before the popup layers, so it must paint
// even when the frame has no window to forward layers to.
func TestWindowFrameOverlayPaintsBorderWithoutWindow(t *testing.T) {
	ink := borderTestTheme(t, theme.Chrome{BorderWidth: 1})
	f := NewWindowFrame()
	dev := graphics.NewSoftDevice(borderTestW, borderTestH)
	f.PaintOverlay(graphics.NewCanvas(dev), FrameBarHeight, borderTestW, borderTestH)
	at := func(x, y int) uint32 { return dev.Pixels()[(borderTestH-1-y)*borderTestW+x] }
	wantOutline(t, at, 0, borderTestH/2, ink)
	wantOutline(t, at, borderTestW/2, 0, ink)
}
