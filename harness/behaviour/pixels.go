package behaviour

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"github.com/sixnineone/meui/graphics"
)

// Pixel assertions for the windowed visual suite.
//
// The captures used to be write-only: a visual test could prove a widget's
// STATE changed (ScrollOffset, Checked, Text) but never that the frame did, so
// a render regression — nothing painted, a stale frame, the wrong ink, content
// drawn a bar too low — passed the suite and only surfaced when a human opened
// a PNG. These helpers assert on the captured framebuffer itself.
//
// All coordinates are TOP-DOWN and match widget coordinates. The GL
// framebuffer is bottom-up; pixelsToImage/PixelAt convert once, here.

// pixelDiff is the sum |ΔR|+|ΔG|+|ΔB| above which two pixels count as
// different. It absorbs anti-aliasing noise without hiding a real ink change.
const pixelDiff = 24

// Golden comparison tolerances, in the units refcompare.py reports: mean
// per-pixel |ΔR|+|ΔG|+|ΔB| (0..765) and the percentage of pixels over 60.
// They are loose on purpose — the same scene rasterizes slightly differently
// across GL drivers (hardware vs llvmpipe differ by ~1 mean here) — but tight
// enough that a scene that stops drawing, or draws the wrong thing, fails.
const (
	goldenMeanTol = 6.0
	goldenPctTol  = 3.0
)

// pixelsToImage converts 0xAARRGGBB bottom-up pixels (GL) to a top-down
// NRGBA image.
func pixelsToImage(pix []uint32, w, h int32) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, int(w), int(h)))
	for y := 0; y < int(h); y++ {
		row := pix[(int(h)-1-y)*int(w) : (int(h)-y)*int(w)]
		for x, px := range row {
			o := img.PixOffset(x, y)
			img.Pix[o+0] = byte(px >> 16)
			img.Pix[o+1] = byte(px >> 8)
			img.Pix[o+2] = byte(px)
			img.Pix[o+3] = byte(px >> 24)
		}
	}
	return img
}

// regionIsFlat reports whether the image region is a single colour — the
// cheap "did anything paint here" check.
func regionIsFlat(img image.Image, x, y, w, h int) bool {
	first := true
	var fr, fg, fb uint32
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			r, g, b, _ := img.At(xx, yy).RGBA()
			if first {
				fr, fg, fb, first = r, g, b, false
				continue
			}
			if r != fr || g != fg || b != fb {
				return false
			}
		}
	}
	return true
}

// asNRGBA normalizes any image to *image.NRGBA so the diff can index Pix
// directly instead of paying two interface calls per pixel.
func asNRGBA(src image.Image) *image.NRGBA {
	if n, ok := src.(*image.NRGBA); ok {
		return n
	}
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}

// imageDelta scores two images like refcompare.py: the mean per-pixel
// |ΔR|+|ΔG|+|ΔB| and the percentage of pixels whose delta exceeds 60.
func imageDelta(a, b *image.NRGBA) (mean, pctOver60 float64) {
	w := min(a.Bounds().Dx(), b.Bounds().Dx())
	h := min(a.Bounds().Dy(), b.Bounds().Dy())
	if w <= 0 || h <= 0 {
		return 0, 0
	}
	var sum, over int64
	for y := 0; y < h; y++ {
		ao := y * a.Stride
		bo := y * b.Stride
		for x := 0; x < w; x++ {
			d := absInt(int(a.Pix[ao+4*x+0]) - int(b.Pix[bo+4*x+0]))
			d += absInt(int(a.Pix[ao+4*x+1]) - int(b.Pix[bo+4*x+1]))
			d += absInt(int(a.Pix[ao+4*x+2]) - int(b.Pix[bo+4*x+2]))
			sum += int64(d)
			if d > 60 {
				over++
			}
		}
	}
	n := float64(w * h)
	return float64(sum) / n, float64(over) * 100 / n
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func colorString(c graphics.Color) string {
	return fmt.Sprintf("#%06X", uint32(c)&0xFFFFFF)
}

// colorNear reports whether two colours agree on R, G and B within tol.
// Alpha is ignored: the captured framebuffer is opaque.
func colorNear(a, b graphics.Color, tol uint8) bool {
	for _, shift := range []uint{16, 8, 0} {
		d := int((uint32(a)>>shift)&0xFF) - int((uint32(b)>>shift)&0xFF)
		if d < 0 {
			d = -d
		}
		if d > int(tol) {
			return false
		}
	}
	return true
}

// PixelAt returns the top-down pixel at (x, y) of the frame captured
// immediately before the current step.
func (va *VisualApp) PixelAt(x, y int32) graphics.Color {
	if va.lastPix == nil {
		va.T.Errorf("PixelAt(%d,%d): no frame captured yet", x, y)
		return 0
	}
	if x < 0 || y < 0 || x >= va.lastW || y >= va.lastH {
		va.T.Errorf("PixelAt(%d,%d): outside the %dx%d frame", x, y, va.lastW, va.lastH)
		return 0
	}
	return graphics.Color(va.lastPix[(va.lastH-1-y)*va.lastW+x])
}

// AssertPixel fails when the pixel at (x, y) is not within tol of want.
func (va *VisualApp) AssertPixel(x, y int32, want graphics.Color, tol uint8, what string) {
	got := va.PixelAt(x, y)
	if !colorNear(got, want, tol) {
		va.T.Errorf("%s: pixel(%d,%d) = %s, want %s (±%d)", what, x, y, colorString(got), colorString(want), tol)
	}
}

// Snapshot copies a top-down region of the frame captured immediately before
// the current step, for a later AssertRegionChanged/Equal.
func (va *VisualApp) Snapshot(x, y, w, h int32) []graphics.Color {
	out := make([]graphics.Color, 0, w*h)
	if va.lastPix == nil {
		va.T.Errorf("Snapshot(%d,%d,%d,%d): no frame captured yet", x, y, w, h)
		return out
	}
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			out = append(out, va.PixelAt(xx, yy))
		}
	}
	return out
}

// AssertRegionHasContent fails when the region is a single flat colour — the
// regression where a widget stops painting.
func (va *VisualApp) AssertRegionHasContent(x, y, w, h int32, what string) {
	if va.lastPix == nil {
		va.T.Errorf("%s: no frame captured yet", what)
		return
	}
	img := pixelsToImage(va.lastPix, va.lastW, va.lastH)
	if regionIsFlat(img, int(x), int(y), int(w), int(h)) {
		va.T.Errorf("%s: region (%d,%d %dx%d) is a single flat colour — nothing painted", what, x, y, w, h)
	}
}

// AssertRegionChanged fails when fewer than minPixels of the region differ
// from snap — the regression where an interaction repaints nothing.
func (va *VisualApp) AssertRegionChanged(snap []graphics.Color, x, y, w, h int32, minPixels int, what string) {
	diff, _ := va.regionDiff(snap, x, y, w, h, what)
	if diff < minPixels {
		va.T.Errorf("%s: only %d of %d pixels changed (want ≥%d) — the frame did not update", what, diff, w*h, minPixels)
	}
}

// AssertRegionMeanChanged fails when the mean per-pixel |ΔR|+|ΔG|+|ΔB| over the
// region is below minMean. Use it for a SUBTLE restyle — hover ink, a focus
// ring — where the change is real but no single pixel moves past pixelDiff.
func (va *VisualApp) AssertRegionMeanChanged(snap []graphics.Color, x, y, w, h int32, minMean float64, what string) {
	_, mean := va.regionDiff(snap, x, y, w, h, what)
	if mean < minMean {
		va.T.Errorf("%s: meanΔ %.3f (want ≥%.1f) — the frame did not restyle", what, mean, minMean)
	}
}

// AssertRegionEqual fails when more than maxPixels of the region differ from
// snap — the regression where content that should have returned to an earlier
// state did not.
func (va *VisualApp) AssertRegionEqual(snap []graphics.Color, x, y, w, h int32, maxPixels int, what string) {
	diff, _ := va.regionDiff(snap, x, y, w, h, what)
	if diff > maxPixels {
		va.T.Errorf("%s: %d of %d pixels differ (want ≤%d)", what, diff, w*h, maxPixels)
	}
}

// regionDiff returns how many pixels in the region differ from snap and the
// mean per-pixel delta, both in |ΔR|+|ΔG|+|ΔB| units.
func (va *VisualApp) regionDiff(snap []graphics.Color, x, y, w, h int32, what string) (int, float64) {
	if va.lastPix == nil {
		va.T.Errorf("%s: no frame captured yet", what)
		return 0, 0
	}
	if len(snap) != int(w*h) {
		va.T.Errorf("%s: snapshot has %d pixels, region is %d", what, len(snap), w*h)
		return 0, 0
	}
	diff, sum := 0, 0
	i := 0
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			a, b := uint32(va.PixelAt(xx, yy)), uint32(snap[i])
			i++
			d := absInt(int((a>>16)&0xFF) - int((b>>16)&0xFF))
			d += absInt(int((a>>8)&0xFF) - int((b>>8)&0xFF))
			d += absInt(int(a&0xFF) - int(b&0xFF))
			sum += d
			if d > pixelDiff {
				diff++
			}
		}
	}
	n := int(w * h)
	if n == 0 {
		return 0, 0
	}
	return diff, float64(sum) / float64(n)
}

// Golden compares the frame captured immediately before the current step
// against testdata/golden/<name>.png. Set MEUI_UPDATE_GOLDEN=1 to write the
// file instead (the -update convention the Go tooling uses elsewhere).
//
// The goldens are committed inputs, not generated outputs: they live under
// testdata/ and are deliberately NOT gitignored.
func (va *VisualApp) Golden(name string) {
	path := filepath.Join(va.goldenDir, name+".png")
	if va.lastPix == nil {
		va.T.Errorf("golden %s: no frame captured yet", name)
		return
	}
	got := pixelsToImage(va.lastPix, va.lastW, va.lastH)

	if os.Getenv("MEUI_UPDATE_GOLDEN") != "" {
		if err := os.MkdirAll(va.goldenDir, 0o755); err != nil {
			va.T.Errorf("golden %s: mkdir: %v", name, err)
			return
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, got); err != nil {
			va.T.Errorf("golden %s: encode: %v", name, err)
			return
		}
		if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
			va.T.Errorf("golden %s: write: %v", name, err)
			return
		}
		va.T.Logf("golden %s updated (%dx%d) -> %s", name, va.lastW, va.lastH, path)
		return
	}

	f, err := os.Open(path)
	if err != nil {
		va.T.Errorf("golden %s: %v (run with MEUI_UPDATE_GOLDEN=1 to create it)", name, err)
		return
	}
	defer f.Close()
	ref, err := png.Decode(f)
	if err != nil {
		va.T.Errorf("golden %s: decode: %v", name, err)
		return
	}
	refN := asNRGBA(ref)
	if refN.Bounds().Dx() != int(va.lastW) || refN.Bounds().Dy() != int(va.lastH) {
		va.T.Errorf("golden %s: reference is %dx%d, frame is %dx%d",
			name, refN.Bounds().Dx(), refN.Bounds().Dy(), va.lastW, va.lastH)
		return
	}
	mean, pct := imageDelta(got, refN)
	if mean > goldenMeanTol || pct > goldenPctTol {
		va.T.Errorf("golden %s: meanΔ %.2f (tol %.1f), %%>60 %.2f (tol %.1f) — the render changed",
			name, mean, goldenMeanTol, pct, goldenPctTol)
	}
}
