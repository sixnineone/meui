// paintstateprobe: self-checking probe for the Paint state the render-parity
// corpus cannot see — it never strokes a path, never disables antialiasing and
// never varies opacity.
//
// It measures rather than eyeballs, and exits non-zero on failure, so it can
// stand in for the corpus as a gate for these properties:
//
//   - MiterLimit: a narrow join must spike past the stroke band when the limit
//     allows it, and be bevelled flat when it does not.
//
//   - Antialias: a filled circle must have a coverage ramp when on, and none
//     when off.
//
//   - GlobalAlpha: a half-opacity fill must land halfway between the background
//     and the full-opacity fill.
//
//     go run ./paintstateprobe
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/mobilegles"
)

const W, H = 640, 420

const (
	bgR, bgG, bgB       = 20, 22, 34
	fillR, fillG, fillB = 255, 210, 90
)

// probeContent stands in for a widget tree; the probe draws directly.
type probeContent struct{}

func (p *probeContent) Paint(c *graphics.Canvas, r graphics.Rect) {}

// sharpVee strokes a narrow V. The path reverses by ~162° at the apex, so the
// miter tip reaches h/cos(81°) ≈ 6.4x the half stroke width above it: the 4x
// default bevels it, a generous limit lets the spike through.
func sharpVee(c *graphics.Canvas, p *graphics.Paint, ax, ay float32) {
	path := graphics.NewPath()
	path.MoveTo(graphics.NewPoint(ax-12, ay+90))
	path.LineTo(graphics.NewPoint(ax, ay))
	path.LineTo(graphics.NewPoint(ax+12, ay+90))
	c.DrawPath(path, p)
}

func strokePaint(limit float32) *graphics.Paint {
	p := graphics.NewPaint()
	p.SetStyle(int32(paintstyle.Stroke))
	p.SetStrokeWidth(20)
	p.SetStrokeJoin(graphics.StrokeJoinMiter)
	p.SetMiterLimit(limit)
	p.SetColor(graphics.RGB(fillR, fillG, fillB))
	return p
}

// lit reports whether a pixel is painted with the stroke colour rather than
// showing the background.
func lit(px uint32) bool {
	r := int((px >> 16) & 0xFF)
	g := int((px >> 8) & 0xFF)
	return r > bgR+40 && g > bgG+40
}

// partial reports whether a pixel is a blend of background and stroke — i.e. a
// point on an antialiasing ramp rather than solidly inside or outside.
func partial(px uint32) bool {
	r := int((px >> 16) & 0xFF)
	g := int((px >> 8) & 0xFF)
	return r > bgR+12 && r < fillR-12 && g > bgG+12 && g < fillG-12
}

// spike counts painted pixels in a column above the apex, where only a miter
// tip (not the stroke band, which reaches one half-width up) can appear.
func spike(pix []uint32, ax, ay int, halfHeight int) int {
	n := 0
	for y := ay - 70; y < ay-halfHeight-2; y++ {
		if y < 0 {
			continue
		}
		for x := ax - 10; x <= ax+10; x++ {
			if x < 0 || x >= W {
				continue
			}
			if lit(pix[(H-1-y)*W+x]) {
				n++
			}
		}
	}
	return n
}

func main() {
	if _, err := sdl3.NewWindowSized("strokestyleprobe", W, H); err != nil {
		fmt.Fprintf(os.Stderr, "window: %v\n", err)
		os.Exit(1)
	}

	// Apexes for the miter-limit rows: one join per lane.
	verts := []struct {
		label string
		limit float32
		ax    float32
	}{
		{"limit=1 (bevel)", 1, 120},
		{"limit=4 (default; the 6.4x miter exceeds it)", 4, 320},
		{"limit=12 (miter spike)", 12, 520},
	}
	const apexY = 150

	pix := mobilegles.RenderOffscreen(W, H, func() {
		cc := graphics.NewCanvas(graphics.NewRendererDevice(W, H))
		bg := graphics.NewPaint()
		bg.SetColor(graphics.RGB(bgR, bgG, bgB))
		cc.DrawRect(graphics.NewRect(0, 0, W, H), bg)

		for _, v := range verts {
			sharpVee(cc, strokePaint(v.limit), v.ax, apexY)
		}

		// Antialiasing lanes: two filled circles, ramp on the left only.
		for i, aa := range []bool{true, false} {
			p := graphics.NewPaint()
			p.SetStyle(int32(paintstyle.Fill))
			p.SetColor(graphics.RGB(fillR, fillG, fillB))
			p.SetAntialias(aa)
			cx := float32(170 + i*300)
			cc.DrawCircle(cx, 290, 40, p)
		}

		// Global alpha lanes: the same opaque colour at full and half
		// opacity. The half one must composite halfway to the background.
		for i, alpha := range []float32{1, 0.5} {
			p := graphics.NewPaint()
			p.SetStyle(int32(paintstyle.Fill))
			p.SetColor(graphics.RGB(fillR, fillG, fillB))
			p.SetGlobalAlpha(alpha)
			cc.DrawRect(graphics.NewRect(float32(120+i*220), 370, 160, 34), p)
		}
	})
	if pix == nil {
		fmt.Fprintln(os.Stderr, "offscreen render failed")
		os.Exit(1)
	}

	// Dump the render for inspection alongside the measurements.
	img := image.NewNRGBA(image.Rect(0, 0, W, H))
	for y := 0; y < H; y++ {
		row := pix[(H-1-y)*W : (H-y)*W]
		for x, px := range row {
			o := img.Pix[y*img.Stride+x*4:]
			o[0], o[1], o[2], o[3] = byte(px>>16), byte(px>>8), byte(px), 0xFF
		}
	}
	out := "paintstateprobe.png"
	if f, err := os.Create(out); err == nil {
		png.Encode(f, img)
		f.Close()
	}

	fail := false
	fmt.Println("miter limit (painted pixels in the column above each apex):")
	spikes := make([]int, len(verts))
	for i, v := range verts {
		spikes[i] = spike(pix, int(v.ax), apexY, 10)
		fmt.Printf("  %-42s %4d\n", v.label, spikes[i])
	}
	// A bevel leaves the column empty; a miter tip fills it.
	if spikes[0] != 0 {
		fmt.Printf("  FAIL limit=1 painted %d px above the stroke band; want a flat bevel\n", spikes[0])
		fail = true
	}
	if spikes[1] != 0 {
		fmt.Printf("  FAIL limit=4 painted %d px above the stroke band; the 4x default must bevel a 6.4x miter\n", spikes[1])
		fail = true
	}
	if spikes[2] < 40 {
		fmt.Printf("  FAIL limit=12 painted only %d px above the stroke band; want a visible miter spike\n", spikes[2])
		fail = true
	}

	fmt.Println("antialiasing (ramp pixels around a filled circle, r=40):")
	partials := make([]int, 2)
	for i, cx := range []int{170, 470} {
		for y := 290 - 44; y <= 290+44; y++ {
			for x := cx - 44; x <= cx+44; x++ {
				if x < 0 || x >= W || y < 0 || y >= H {
					continue
				}
				if partial(pix[(H-1-y)*W+x]) {
					partials[i]++
				}
			}
		}
		fmt.Printf("  %-42s %4d\n", map[bool]string{true: "antialias on", false: "antialias off"}[i == 0], partials[i])
	}
	if partials[0] < 100 {
		fmt.Printf("  FAIL antialias on produced only %d ramp pixels; want a full coverage ramp\n", partials[0])
		fail = true
	}
	if partials[1] > 25 {
		fmt.Printf("  FAIL antialias off produced %d ramp pixels; want hard edges\n", partials[1])
		fail = true
	}

	// Global alpha: sample the centre of each bar. The half-opacity bar must
	// sit about halfway between the background and the full-opacity one.
	fmt.Println("global alpha (red channel at the bar centre; fill =", fillR, "bg =", bgR, "):")
	sample := func(x, y int) int {
		return int((pix[(H-1-y)*W+x] >> 16) & 0xFF)
	}
	opaque := sample(200, 387)
	half := sample(420, 387)
	wantHalf := (opaque + bgR) / 2
	fmt.Printf("  %-42s %4d\n", "alpha 1.0", opaque)
	fmt.Printf("  %-42s %4d\n", "alpha 0.5", half)
	fmt.Printf("  %-42s %4d\n", "expected for alpha 0.5 (+/-4)", wantHalf)
	if opaque < fillR-4 {
		fmt.Printf("  FAIL alpha 1.0 sampled %d; want the fill colour %d\n", opaque, fillR)
		fail = true
	}
	if d := half - wantHalf; d < -4 || d > 4 {
		fmt.Printf("  FAIL alpha 0.5 is %d away from the halfway point\n", d)
		fail = true
	}

	if fail {
		fmt.Println("paintstateprobe: FAIL")
		os.Exit(1)
	}
	fmt.Println("paintstateprobe: PASS (wrote", out+")")
}
