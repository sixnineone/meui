// strokeprobe: renders thick strokes with each cap and join style to a PNG, so
// the join geometry can be inspected directly. The outer side of a turn must be
// filled by the join; if it is not, the notch is visible against the background.
//
//	go run ./strokeprobe
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

const W, H = 520, 360

type probeContent struct {
	DrawCallback func(*graphics.Canvas, graphics.Rect)
}

func (p *probeContent) Paint(c *graphics.Canvas, r graphics.Rect) {}

func strokePaint(width float32) *graphics.Paint {
	p := graphics.NewPaint()
	p.SetStyle(int32(paintstyle.Stroke))
	p.SetStrokeWidth(width)
	p.SetColor(graphics.RGB(255, 210, 90))
	return p
}

// vee strokes a sharp V whose apex is at (ax, ay).
func vee(c *graphics.Canvas, p *graphics.Paint, ax, ay float32) {
	path := graphics.NewPath()
	path.MoveTo(graphics.NewPoint(ax-60, ay+70))
	path.LineTo(graphics.NewPoint(ax, ay))
	path.LineTo(graphics.NewPoint(ax+60, ay+70))
	c.DrawPath(path, p)
}

func main() {
	wnd, err := sdl3.NewWindowSized("strokeprobe", W, H)
	if err != nil {
		fmt.Fprintf(os.Stderr, "window: %v\n", err)
		os.Exit(1)
	}
	_ = wnd

	content := &probeContent{}
	content.DrawCallback = func(c *graphics.Canvas, rect graphics.Rect) {
		pix := mobilegles.RenderOffscreen(W, H, func() {
			cc := graphics.NewCanvas(graphics.NewRendererDevice(W, H))
			bg := graphics.NewPaint()
			bg.SetColor(graphics.RGB(20, 22, 34))
			cc.DrawRect(graphics.NewRect(0, 0, W, H), bg)

			// Caps: three horizontal thick strokes.
			caps := []struct {
				name string
				cap  graphics.StrokeCap
			}{
				{"butt", graphics.StrokeCapButt},
				{"square", graphics.StrokeCapSquare},
				{"round", graphics.StrokeCapRound},
			}
			for i, cs := range caps {
				p := strokePaint(18)
				p.SetStrokeCap(cs.cap)
				y := float32(50 + i*40)
				cc.DrawLine(60, y, 180, y, p)
			}

			// Joins: the same sharp V with each join style.
			joins := []struct {
				name string
				join graphics.StrokeJoin
			}{
				{"miter", graphics.StrokeJoinMiter},
				{"bevel", graphics.StrokeJoinBevel},
				{"round", graphics.StrokeJoinRound},
			}
			for i, js := range joins {
				p := strokePaint(20)
				p.SetStrokeJoin(js.join)
				vee(cc, p, float32(380+i*0), float32(90+i*90))
			}
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
				o[0] = byte(px >> 16)
				o[1] = byte(px >> 8)
				o[2] = byte(px)
				o[3] = 0xFF
			}
		}
		out := "strokeprobe.png"
		f, err := os.Create(out)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		if err := png.Encode(f, img); err != nil {
			fmt.Fprintf(os.Stderr, "encode: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("wrote", out)
	}
	content.DrawCallback(nil, graphics.Rect{})
}
