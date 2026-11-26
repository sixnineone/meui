// ceprobe: CodeEditor rendered directly (no QML) to isolate parsing vs drawing.
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/widgets"
)

const W, H = 700, 300

type probeContent struct {
	DrawCallback func(*graphics.Canvas, graphics.Rect)
	editor       *widgets.CodeEditor
}

func (p *probeContent) Paint(c *graphics.Canvas, r graphics.Rect) {}

func main() {
	text := "package main\n\nimport \"fmt\"\n\n// Entry point\ntype Server struct {\n    Host string\n    Port int\n}\n\nfunc (s *Server) Start() error {\n    if s.Port == 0 {\n        return fmt.Errorf(\"no port\")\n    }\n    return nil\n}"
	e := widgets.NewCodeEditor()
	e.SetText(text)
	e.SetFrameRect(graphics.NewRect(0, 0, W, H))

	wnd, err := sdl3.NewWindowSized("ceprobe", W, H)
	if err != nil {
		fmt.Fprintf(os.Stderr, "window: %v\n", err)
		os.Exit(1)
	}
	content := &probeContent{editor: e}
	content.DrawCallback = func(c *graphics.Canvas, rect graphics.Rect) {
		e.Paint(c, graphics.NewRect(0, 0, W, H))
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
		f, _ := os.Create("out/ceprobe.png")
		png.Encode(f, img)
		f.Close()
		fmt.Println("captured")
	})
	wnd.Show()
	sdl3.Start(nil)
}
