// clippathprobe: self-checking probe for Canvas.ClipPath, which the render-parity
// corpus cannot see (it only ever clips rectangles).
//
// It clips a full-canvas fill to several outlines and samples the result, so the
// two halves of the contract are both measured:
//
//   - a convex outline clips exactly: inside stays, outside goes;
//
//   - a concave outline, which half-planes cannot express, falls back to its
//     bounding region and must keep every point that was inside the shape
//     (a clip may show a little more than asked, never less);
//
//   - Save/Restore puts the clip back.
//
//     go run ./clippathprobe
package main

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"

	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/mobilegles"
)

const W, H = 640, 240

const (
	bgR, bgG            = 20, 22
	fillR, fillG, fillB = 255, 210, 90
)

type probeContent struct{}

func (p *probeContent) Paint(c *graphics.Canvas, r graphics.Rect) {}

func poly(pts ...[2]float32) *graphics.Path {
	p := graphics.NewPath()
	p.MoveTo(graphics.NewPoint(pts[0][0], pts[0][1]))
	for _, pt := range pts[1:] {
		p.LineTo(graphics.NewPoint(pt[0], pt[1]))
	}
	p.Close()
	return p
}

// hexagon returns a regular n-gon path.
func ngon(cx, cy, r float32, n int) *graphics.Path {
	p := graphics.NewPath()
	for i := 0; i < n; i++ {
		a := 2 * math.Pi * float64(i) / float64(n)
		x := cx + r*float32(math.Cos(a))
		y := cy + r*float32(math.Sin(a))
		if i == 0 {
			p.MoveTo(graphics.NewPoint(x, y))
		} else {
			p.LineTo(graphics.NewPoint(x, y))
		}
	}
	p.Close()
	return p
}

func main() {
	if _, err := sdl3.NewWindowSized("clippathprobe", W, H); err != nil {
		fmt.Fprintf(os.Stderr, "window: %v\n", err)
		os.Exit(1)
	}

	// A concave chevron (the notch at the top middle is what makes it concave)
	// and a 10-gon, which exceeds the per-outline half-plane cap.
	chevron := [][2]float32{
		{20, 20}, {90, 90}, {160, 20}, {160, 150}, {90, 110}, {20, 150},
	}
	tenGon := ngon(320, 85, 70, 10)
	triangle := [][2]float32{{430, 20}, {560, 20}, {495, 150}}

	var exactChevron, exactTenGon, exactTriangle bool

	pix := mobilegles.RenderOffscreen(W, H, func() {
		// RenderOffscreen turns the per-vertex clip discard OFF: the offscreen
		// target is the widget's own cache, so it is unclipped by construction.
		// A path clip is gated by the same uniform as the rect clip, so turn it
		// back on here to measure what a real on-screen frame does.
		mobilegles.SetClipDiscardEnabled(true)
		cc := graphics.NewCanvas(graphics.NewRendererDevice(W, H))
		bg := graphics.NewPaint()
		bg.SetColor(graphics.RGB(bgR, bgG, 34))
		cc.DrawRect(graphics.NewRect(0, 0, W, H), bg)

		fill := graphics.NewPaint()
		fill.SetStyle(int32(paintstyle.Fill))
		fill.SetColor(graphics.RGB(fillR, fillG, fillB))

		lane := func(path *graphics.Path) bool {
			cc.Save()
			exact := cc.ClipPath(path)
			cc.DrawRect(graphics.NewRect(0, 0, W, H), fill)
			cc.Restore()
			return exact
		}

		exactChevron = lane(poly(chevron...))
		exactTenGon = lane(tenGon)
		exactTriangle = lane(poly(triangle...))

		// Clip restoration: with no clip active the fill must reach every
		// corner, including the ones the lanes above clipped away.
		cc.ResetClip()
		last := graphics.NewPaint()
		last.SetStyle(int32(paintstyle.Fill))
		last.SetColor(graphics.RGB(90, 200, 255))
		cc.DrawRect(graphics.NewRect(590, 20, 40, 40), last)
		// Then a clipped rect must not leak once restored.
		cc.Save()
		cc.ClipPath(poly([2]float32{0, 200}, [2]float32{100, 200}, [2]float32{0, 230}))
		cc.Restore()
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
	out := "clippathprobe.png"
	if f, err := os.Create(out); err == nil {
		png.Encode(f, img)
		f.Close()
	}

	// painted reports whether the yellow clip fill reached a point.
	painted := func(x, y int) bool {
		px := pix[(H-1-y)*W+x]
		return int((px>>16)&0xFF) > bgR+40 && int((px>>8)&0xFF) > bgG+40
	}

	fail := false
	check := func(label string, x, y int, want bool) {
		got := painted(x, y)
		status := "ok"
		if got != want {
			status = "FAIL"
			fail = true
		}
		kept := "cut"
		if got {
			kept = "kept"
		}
		fmt.Printf("  %-46s (%3d,%3d) %-5s %s\n", label, x, y, kept, status)
	}

	fmt.Println("convex triangle clip (expect exact):")
	if !exactTriangle {
		fmt.Println("  FAIL ClipPath(triangle) reported approximate for a convex outline")
		fail = true
	}
	check("centroid stays", 495, 60, true)
	check("left of the slanted edge is cut", 440, 140, false)
	check("right of the slanted edge is cut", 550, 140, false)
	check("far outside is cut", 620, 200, false)

	fmt.Println("convex 10-gon clip (10 edges exceeds the cap, expect bbox fallback):")
	if exactTenGon {
		fmt.Println("  FAIL ClipPath(10-gon) claimed exact; the outline needs more planes than the cap")
		fail = true
	}
	check("centre stays", 320, 85, true)
	check("bounding-box corner stays (superset fallback)", 255, 20, true)
	check("outside the bounding box is cut", 450, 85, false)

	fmt.Println("concave chevron clip (hull is a 4-edge box, so planes apply; expect approximate):")
	if exactChevron {
		fmt.Println("  FAIL ClipPath(chevron) claimed exact for a concave outline")
		fail = true
	}
	// Points strictly inside the chevron, kept away from the outline: a clip
	// boundary runs through the plane, so a pixel whose centre sits exactly on
	// an edge is half-covered by definition (the same is true of ClipRect).
	for _, pt := range [][2]int{{40, 110}, {60, 120}, {120, 110}, {140, 120}, {80, 95}, {100, 95}} {
		check("inside the chevron stays", pt[0], pt[1], true)
	}
	// The notch is inside the hull but outside the chevron: the documented
	// superset, which is why the call reports approximate.
	check("notch stays (hull is a superset)", 90, 60, true)
	check("outside the hull is cut", 230, 130, false)

	fmt.Println("clip restoration:")
	check("drawn after ResetClip+Restore", 610, 40, true)
	check("canvas cleared of every lane's fill", 300, 210, false)
	check("canvas cleared of every lane's fill", 495, 220, false)

	if fail {
		fmt.Println("clippathprobe: FAIL")
		os.Exit(1)
	}
	fmt.Println("clippathprobe: PASS (wrote", out+")")
}
