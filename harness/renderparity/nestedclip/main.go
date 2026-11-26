// nestedclip: RenderOffscreen containing red rect, ClipRect, green rect.
// If the green rect is missing from the capture, post-clip draws die in
// nested offscreen rendering.
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/mobilegles"
)

const W, H = 300, 200

type probeContent struct {
	DrawCallback func(*graphics.Canvas, graphics.Rect)
}

func (p *probeContent) Paint(c *graphics.Canvas, r graphics.Rect) {}

func main() {
	wnd, err := sdl3.NewWindowSized("nestedclip", W, H)
	if err != nil {
		fmt.Fprintf(os.Stderr, "window: %v\n", err)
		os.Exit(1)
	}
	content := &probeContent{}
	content.DrawCallback = func(c *graphics.Canvas, rect graphics.Rect) {
		pix := mobilegles.RenderOffscreen(W, H, func() {
			cc := graphics.NewCanvas(graphics.NewRendererDevice(W, H))
			red := graphics.NewPaint()
			red.SetColor(graphics.RGB(255, 0, 0))
			cc.DrawRect(graphics.NewRect(0, 0, W, H), red)

			cc.Save()
			cc.ClipRect(graphics.NewRect(0, 0, 150, H), 0, false)
			grn := graphics.NewPaint()
			grn.SetColor(graphics.RGB(0, 255, 0))
			cc.DrawRect(graphics.NewRect(20, 20, 100, 100), grn)
			cc.Restore()
		})
		if pix != nil {
			img := graphics.NewImage(pix, W, H)
			white := graphics.NewPaint()
			c.DrawImageInRect(img, graphics.NewRect(0, 0, W, H), nil, white)
		}
	}
	wnd.SetContentRect(graphics.NewRect(0, 0, W, H))
	wnd.SetContent(content)
	wnd.CaptureAfterFrame(5, func(pix []uint32, pw, ph int32) {
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
		f, _ := os.Create("out/nestedclip.png")
		png.Encode(f, img)
		f.Close()
		fmt.Println("captured")
	})
	wnd.Show()
	sdl3.Start(nil)
}
