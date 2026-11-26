// baselineprobe: draw text at a known baseline inside a 28px field box,
// capture, and measure the ink band vs the field center.
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
)

const W, H = 300, 80

type probeContent struct {
	DrawCallback func(*graphics.Canvas, graphics.Rect)
}

func (p *probeContent) Paint(c *graphics.Canvas, r graphics.Rect) {}

func main() {
	wnd, err := sdl3.NewWindowSized("baseline", W, H)
	if err != nil {
		fmt.Fprintf(os.Stderr, "window: %v\n", err)
		os.Exit(1)
	}
	content := &probeContent{}
	content.DrawCallback = func(c *graphics.Canvas, rect graphics.Rect) {
		font := graphics.SystemFont
		fmt.Printf("[metrics] size=%.1f Baseline=%.2f LineHeight=%.2f\n",
			font.Size(), font.Baseline(), font.LineHeight())
		// field box: (10, 10) 200x28, center y = 24
		p := graphics.NewPaint()
		p.SetColor(graphics.RGB(58, 61, 76))
		c.DrawRect(graphics.NewRect(10, 10, 200, 28), p)
		// center per the standard formula: (28 - lh)/2 + baseline
		textY := 10 + (28-font.LineHeight())/2 + font.Baseline()
		tp := graphics.NewPaint()
		tp.SetColor(graphics.RGB(255, 255, 255))
		c.DrawText("Editable", 16, textY, font, tp)
		fmt.Printf("[metrics] textY (baseline) = %.2f, field center = 24\n", textY)
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
		f, _ := os.Create("out/baseline.png")
		png.Encode(f, img)
		f.Close()
		fmt.Println("captured")
	})
	wnd.Show()
	sdl3.Start(nil)
}
