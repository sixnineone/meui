// renderbar draws the WindowFrame title bar offscreen and saves a PNG
// for offline inspection of the bar's glyphs and layout.
package main

import (
	"image"
	"image/png"
	"os"
	"time"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/layout"
	"github.com/sixnineone/meui/widgets"
)

func main() {
	wnd, err := sdl3.NewWindowSized("renderbar", 500, 100)
	if err != nil {
		panic(err)
	}
	wnd.Show()

	time.Sleep(500 * time.Millisecond) // GL context + renderer ready

	frame := widgets.NewWindowFrame()
	frame.SetTitle("MeUi Demo v3")

	// Middle with the same two buttons as the demo.
	newBtn := widgets.NewButton()
	newBtn.SetText("New Window")
	newBtn.SetWidthDirect(90)
	newBtn.SetHeightDirect(24)
	newBtn.SetX(8)
	newBtn.SetY(2)
	fsBtn := widgets.NewButton()
	fsBtn.SetText("Fullscreen")
	fsBtn.SetWidthDirect(90)
	fsBtn.SetHeightDirect(24)
	fsBtn.SetX(106)
	fsBtn.SetY(2)
	host := meui.NewPanel()
	host.SetLayout(layout.NewHStack()) // EXACTLY what the live demo does
	host.AddChild(newBtn)
	host.AddChild(fsBtn)
	frame.SetMiddle(host)

	const W, H = 500, 28
	var saved bool
	graphics.CaptureOffscreen(wnd.Device(), W, H, func(c *graphics.Canvas) {
		frame.PaintBar(c, W)
		nb, fb, hb := newBtn.FrameRect(), fsBtn.FrameRect(), host.FrameRect()
		println("[renderbar] newBtn:", nb.X, nb.Y, nb.Width, nb.Height)
		println("[renderbar] fsBtn:", fb.X, fb.Y, fb.Width, fb.Height)
		println("[renderbar] host:", hb.X, hb.Y, hb.Width, hb.Height)
	}, func(img *image.NRGBA) {
		f, err := os.Create("/tmp/opencode/framebar.png")
		if err != nil {
			panic(err)
		}
		if err := png.Encode(f, img); err != nil {
			panic(err)
		}
		f.Close()
		saved = true
		println("saved /tmp/opencode/framebar.png")
	})
	if !saved {
		println("capture failed")
	}

	wnd.Close()
	sdl3.Shutdown()
}
