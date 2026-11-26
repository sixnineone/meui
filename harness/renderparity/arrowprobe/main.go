// arrowprobe: isolate spinner arrow rendering — SVG at various scales
// vs the text-glyph fallback.
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/widgets"
)

const W, H = 400, 200

type probeContent struct {
	DrawCallback func(*graphics.Canvas, graphics.Rect)
}

func (p *probeContent) Paint(c *graphics.Canvas, r graphics.Rect) {}

func main() {
	doc, err := meui.LoadSVGByName("arrow_up")
	fmt.Printf("load arrow_up: err=%v doc=%v w=%.0f h=%.0f elems=%d\n",
		err, doc != nil, func() float32 {
			if doc != nil {
				return doc.Width
			}
			return 0
		}(), func() float32 {
			if doc != nil {
				return doc.Height
			}
			return 0
		}(), func() int {
			if doc != nil {
				return len(doc.Elements)
			}
			return 0
		}())
	if doc != nil {
		for i, el := range doc.Elements {
			fmt.Printf("  elem[%d] type=%s attrs=%v\n", i, el.Type, el.Attrs)
		}
	}

	wnd, err := sdl3.NewWindowSized("arrowprobe", W, H)
	if err != nil {
		fmt.Fprintf(os.Stderr, "window: %v\n", err)
		os.Exit(1)
	}
	content := &probeContent{}
	content.DrawCallback = func(c *graphics.Canvas, rect graphics.Rect) {
		// SVG at 16px (scale 1)
		if doc != nil {
			meui.DrawSVG(c, doc, 10, 10, 16, 16)
		}
		// SVG at spinner size 10px (scale 0.625)
		if doc != nil {
			meui.DrawSVG(c, doc, 40, 10, 10, 10)
		}
		// SVG at 32px (scale 2)
		if doc != nil {
			meui.DrawSVG(c, doc, 70, 10, 32, 32)
		}
		// text-glyph fallback for comparison
		font := graphics.SystemFont
		p := graphics.NewPaint()
		p.SetColor(graphics.RGBA(0xAA, 0xAA, 0xAA, 0xFF))
		at := graphics.NewText("▲", &graphics.TextDecoration{Font: font, OnBackgroundInk: p.Color()})
		at.Draw(c, 130, 25)

		// a REAL SpinBox widget, painted like the tree does
		sb := widgets.NewSpinBox()
		sb.SetValue(0)
		sb.SetFrameRect(graphics.NewRect(10, 60, 160, 28))
		sb.Paint(c, graphics.NewRect(10, 60, 160, 28))
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
		f, _ := os.Create("out/arrowprobe.png")
		png.Encode(f, img)
		f.Close()
		fmt.Println("captured")
	})
	wnd.Show()
	sdl3.Start(nil)
}
