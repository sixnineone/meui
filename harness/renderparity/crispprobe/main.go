// P7 crispness check: same string drawn directly vs through a cache
// image at 1:1, plus a 2x-scaled cache to inspect filtering.
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
)

const W, H = 420, 220

type probeContent struct {
	DrawCallback func(*graphics.Canvas, graphics.Rect)
}

func (p *probeContent) Paint(c *graphics.Canvas, r graphics.Rect) {}

func drawStrings(c *graphics.Canvas, x, y float32) {
	font := graphics.SystemFont
	p := graphics.NewPaint()
	p.SetColor(graphics.RGB(255, 255, 255))
	c.DrawText("Hamburg 0123 pqgy", x, y, font, p)
}

func main() {
	wnd, err := sdl3.NewWindowSized("crisp", W, H)
	if err != nil {
		fmt.Fprintf(os.Stderr, "window: %v\n", err)
		os.Exit(1)
	}
	content := &probeContent{}
	content.DrawCallback = func(c *graphics.Canvas, rect graphics.Rect) {
		// 1) direct
		drawStrings(c, 10, 30)
		// 2) via cache 1:1
		img, err := graphics.NewImageFromDrawing(c.Device(), 200, 30, 72, func(cc *graphics.Canvas) {
			drawStrings(cc, 0, 20)
		})
		if err == nil {
			c.DrawImageInRect(img, graphics.NewRect(10, 60, 200, 30), nil, nil)
		}
		// 3) via cache drawn at 0.5x (downscale) and 2x (upscale)
		if err == nil {
			c.DrawImageInRect(img, graphics.NewRect(10, 110, 100, 15), nil, nil)
			c.DrawImageInRect(img, graphics.NewRect(10, 150, 400, 60), nil, nil)
		}
	}
	wnd.SetContentRect(graphics.NewRect(0, 0, W, H))
	wnd.SetContent(content)
	wnd.CaptureAfterFrame(3, func(pix []uint32, pw, ph int32) {
		img := image.NewNRGBA(image.Rect(0, 0, int(pw), int(ph)))
		for y := 0; y < int(ph); y++ {
			src := (int(ph) - 1 - y) * int(pw)
			for x := 0; x < int(pw); x++ {
				px := pix[src+x]
				i := (y*int(pw) + x) * 4
				img.Pix[i+0] = byte((px >> 16) & 0xFF)
				img.Pix[i+1] = byte((px >> 8) & 0xFF)
				img.Pix[i+2] = byte(px & 0xFF)
				img.Pix[i+3] = 255
			}
		}
		f, _ := os.Create("out/crisp.png")
		png.Encode(f, img)
		f.Close()
		fmt.Println("captured")
	})
	wnd.Show()
	sdl3.Start(nil)
}
