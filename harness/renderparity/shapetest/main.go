package main

import (
	"image"
	"image/png"
	"math"
	"os"

	m "github.com/sixnineone/mobilegles"
)

func px2img(pix []uint32, w, h int32) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, int(w), int(h)))
	for y := 0; y < int(h); y++ {
		src := (int(h) - 1 - y) * int(w)
		for x := 0; x < int(w); x++ {
			px := pix[src+x]
			i := (y*int(w) + x) * 4
			img.Pix[i+0] = byte((px >> 16) & 0xFF)
			img.Pix[i+1] = byte((px >> 8) & 0xFF)
			img.Pix[i+2] = byte(px & 0xFF)
			img.Pix[i+3] = byte((px >> 24) & 0xFF)
		}
	}
	return img
}

func circleVerts(cx, cy, r float32, segs int) []float32 {
	verts := make([]float32, 0, segs*15)
	for i := 0; i < segs; i++ {
		a1 := float64(i) * 2 * math.Pi / float64(segs)
		a2 := float64(i+1) * 2 * math.Pi / float64(segs)
		verts = append(verts,
			cx, cy, 1, 0, 0,
			cx+r*float32(math.Cos(a1)), cy+r*float32(math.Sin(a1)), 0, 1, 0,
			cx+r*float32(math.Cos(a2)), cy+r*float32(math.Sin(a2)), 0, 0, 1,
		)
	}
	return verts
}

// splitVerts chunks a 15-float-per-tri slice into per-triangle slices.
func splitVerts(all []float32) [][]float32 {
	var out [][]float32
	for i := 0; i+15 <= len(all); i += 15 {
		out = append(out, all[i:i+15])
	}
	return out
}

func main() {
	mw := m.InitWindow(420, 260)
	_ = mw
	m.SetScreenSize(420, 260)
	m.InitRenderer()

	w, h := int32(420), int32(260)
	pix := m.RenderOffscreen(w, h, func() {
		c := m.NewCanvas(420, 260)
		c.ClearWithViewport(0xFF202020, 0, 0, 420, 260)
		m.GLDepthTest(false)

		// ONLY test B fan (isolated)
		verts := make([]float32, 0, 540)
		segs := 8
		x, y, ww, hh := float32(30), float32(90), float32(120), float32(60)
		rx, ry := float32(8), float32(8)
		trCX, trCY := x+ww-rx, y+ry
		brCX, brCY := x+ww-rx, y+hh-ry
		blCX, blCY := x+rx, y+hh-ry
		tlCX, tlCY := x+rx, y+ry
		type pt struct{ x, y float32 }
		var pts []pt
		pts = append(pts, pt{x + rx, y})
		pts = append(pts, pt{trCX, y})
		for i := 1; i <= segs; i++ {
			a := (270 + float32(i)*90/float32(segs)) * math.Pi / 180
			pts = append(pts, pt{trCX + rx*float32(math.Cos(float64(a))), trCY + ry*float32(math.Sin(float64(a)))})
		}
		pts = append(pts, pt{x + ww, brCY})
		for i := 1; i <= segs; i++ {
			a := float32(i) * 90 / float32(segs) * math.Pi / 180
			pts = append(pts, pt{brCX + rx*float32(math.Cos(float64(a))), brCY + ry*float32(math.Sin(float64(a)))})
		}
		pts = append(pts, pt{blCX, y + hh})
		for i := 1; i <= segs; i++ {
			a := (90 + float32(i)*90/float32(segs)) * math.Pi / 180
			pts = append(pts, pt{blCX + rx*float32(math.Cos(float64(a))), blCY + ry*float32(math.Sin(float64(a)))})
		}
		pts = append(pts, pt{x, tlCY})
		for i := 1; i <= segs; i++ {
			a := (180 + float32(i)*90/float32(segs)) * math.Pi / 180
			pts = append(pts, pt{tlCX + rx*float32(math.Cos(float64(a))), tlCY + ry*float32(math.Sin(float64(a)))})
		}
		_ = verts
		orange := m.RGBA(220, 120, 80, 255)
		for i := 1; i < len(pts)-1; i++ {
			tv := []float32{
				pts[0].x, pts[0].y, 1, 0, 0,
				pts[i].x, pts[i].y, 0, 1, 0,
				pts[i+1].x, pts[i+1].y, 0, 0, 1,
			}
			m.DrawTrianglesBatch(tv, orange) // same color: batch accumulates, ONE flush at end
		}

		m.FlushBatches()
	})
	img := px2img(pix, w, h)
	f, _ := os.Create("renderparity/out/shapetest.png")
	png.Encode(f, img)
	f.Close()
	println("done")
}
