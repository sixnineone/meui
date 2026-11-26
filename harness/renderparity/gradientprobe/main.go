// gradientprobe: self-checking probe for multi-stop, conic and stroke
// gradients, none of which the render-parity corpus can see (it uses no
// gradients at all, and the toolkit does not yet either).
//
// Every lane is chosen so that the OLD implementation cannot pass it: two-stop
// linear/radial and rect-only was all that existed, so the middle stop of a
// three-stop ramp and any conic or stroke gradient had no way to render.
//
//	go run ./gradientprobe
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

const W, H = 640, 200

type probeContent struct{}

func (p *probeContent) Paint(c *graphics.Canvas, r graphics.Rect) {}

// near reports whether a pixel is within tol of every channel of want, ignoring
// alpha (the probe composites over an opaque background).
func near(px uint32, want graphics.Color, tol int) bool {
	ch := func(shift uint) int { return int((px >> shift) & 0xFF) }
	d := func(a, b int) int {
		if a > b {
			return a - b
		}
		return b - a
	}
	return d(ch(16), int((want>>16)&0xFF)) <= tol &&
		d(ch(8), int((want>>8)&0xFF)) <= tol &&
		d(ch(0), int(want&0xFF)) <= tol
}

func main() {
	if _, err := sdl3.NewWindowSized("gradientprobe", W, H); err != nil {
		fmt.Fprintf(os.Stderr, "window: %v\n", err)
		os.Exit(1)
	}

	red := graphics.RGB(255, 0, 0)
	green := graphics.RGB(0, 255, 0)
	blue := graphics.RGB(0, 0, 255)
	white := graphics.RGB(255, 255, 255)

	pix := mobilegles.RenderOffscreen(W, H, func() {
		mobilegles.SetClipDiscardEnabled(true)
		cc := graphics.NewCanvas(graphics.NewRendererDevice(W, H))
		bg := graphics.NewPaint()
		bg.SetColor(graphics.RGB(20, 22, 34))
		cc.DrawRect(graphics.NewRect(0, 0, W, H), bg)

		// Lane 1: three-stop linear ramp. The middle stop is what separates a
		// real multi-stop ramp from the old two-colour lerp, which would blend
		// red to blue and show magenta at the midpoint.
		lin := graphics.NewPaint()
		lin.SetLinearGradient(graphics.NewPoint(20, 50), graphics.NewPoint(160, 50),
			[]graphics.GradientStop{
				{Offset: 0, Color: red},
				{Offset: 0.5, Color: green},
				{Offset: 1, Color: blue},
			})
		cc.DrawRect(graphics.NewRect(20, 20, 140, 60), lin)

		// Lane 2: conic ramp around a filled circle. Sampled on the horizontal
		// axis only, so the check does not depend on the sweep's handedness.
		con := graphics.NewPaint()
		con.SetConicGradient(graphics.NewPoint(250, 50), 0, []graphics.GradientStop{
			{Offset: 0, Color: red},
			{Offset: 0.5, Color: green},
			{Offset: 1, Color: red},
		})
		cc.DrawCircle(250, 50, 40, con)

		// Lane 3: radial ramp, white centre to blue edge.
		rad := graphics.NewPaint()
		rad.SetRadialGradient(graphics.NewPoint(400, 50), 40, []graphics.GradientStop{
			{Offset: 0, Color: white},
			{Offset: 1, Color: blue},
		})
		cc.DrawCircle(400, 50, 40, rad)

		// Lane 4: a gradient on a STROKE, not a fill.
		st := graphics.NewPaint()
		st.SetStyle(int32(paintstyle.Stroke))
		st.SetStrokeWidth(8)
		st.SetRadialGradient(graphics.NewPoint(540, 50), 30, []graphics.GradientStop{
			{Offset: 0, Color: white},
			{Offset: 1, Color: red},
		})
		cc.DrawCircle(540, 50, 30, st)

		// Lane 5: a second, different linear gradient in the SAME batch, to
		// prove the LUT atlas keeps rows apart.
		two := graphics.NewPaint()
		two.SetLinearGradient(graphics.NewPoint(20, 150), graphics.NewPoint(160, 150),
			[]graphics.GradientStop{
				{Offset: 0, Color: white},
				{Offset: 1, Color: blue},
			})
		cc.DrawRect(graphics.NewRect(20, 120, 140, 60), two)
	})
	if pix == nil {
		fmt.Fprintln(os.Stderr, "offscreen render failed")
		os.Exit(1)
	}

	img := image.NewNRGBA(image.Rect(0, 0, W, H))
	for y := 0; y < H; y++ {
		row := pix[(H-1-y)*W : (H-y)*W]
		for x, px := range row {
			o := img.Pix[y*img.Stride+x*4:]
			o[0], o[1], o[2], o[3] = byte(px>>16), byte(px>>8), byte(px), 0xFF
		}
	}
	out := "gradientprobe.png"
	if f, err := os.Create(out); err == nil {
		png.Encode(f, img)
		f.Close()
	}

	at := func(x, y int) uint32 { return pix[(H-1-y)*W+x] }
	fail := false
	check := func(label string, x, y int, want graphics.Color, tol int) {
		px := at(x, y)
		got := graphics.RGBA(byte(px>>16), byte(px>>8), byte(px), byte(px>>24))
		status := "ok"
		if !near(px, want, tol) {
			status = "FAIL"
			fail = true
		}
		fmt.Printf("  %-40s (%3d,%3d) got %02x%02x%02x want ~%02x%02x%02x %s\n",
			label, x, y, byte(px>>16), byte(px>>8), byte(px), byte(want>>16), byte(want>>8), byte(want), status)
		_ = got
	}

	fmt.Println("three-stop linear ramp (the 0.5 stop must be green, not magenta):")
	check("near the start", 25, 50, red, 60)
	check("at the middle stop", 90, 50, green, 60)
	check("near the end", 155, 50, blue, 60)

	fmt.Println("conic ramp around a circle:")
	check("0 degrees", 285, 50, red, 70)
	check("180 degrees", 215, 50, green, 70)

	fmt.Println("radial ramp (white centre to blue edge):")
	check("centre", 400, 50, white, 25)
	check("near the edge", 435, 50, blue, 70)

	fmt.Println("radial gradient on a stroke:")
	// The SDF draws a stroke as an INSIDE band, so the ring spans radius
	// 30-8..30; sample its middle rather than the antialiased outer edge.
	// At radius 26 of a 30px radial the ramp is ~87% of the way to red.
	check("mid-ring", 566, 50, graphics.RGB(255, 33, 33), 70)
	check("inside the ring is unfilled", 540, 50, graphics.RGB(20, 22, 34), 12)

	fmt.Println("second gradient in the same batch (atlas rows must not collide):")
	check("near the start", 25, 150, white, 40)
	check("near the end", 155, 150, blue, 60)

	if fail {
		fmt.Println("gradientprobe: FAIL")
		os.Exit(1)
	}
	fmt.Println("gradientprobe: PASS (wrote", out+")")
}
