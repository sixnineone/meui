// glshot renders a QML scene through the REAL pipeline (SDL3 + GL) and writes the
// framebuffer as a PNG, so an offscreen shot can be compared with what ships.
//
//	go run ./cmd/tools/glshot -o live.png -w 1280 -h 900 scene.qml
//	go run ./cmd/tools/qmlshot -o offscreen.png -w 1280 -h 900 scene.qml
//	go run ./cmd/tools/imgdiff -diff delta.png offscreen.png live.png
//
// It is the other half of the parity guard: cmd/tools/qmlshot goes through the software
// device, this goes through GL, and the two must not drift. Text is the known source of
// small differences (two rasterizers), which is why the parity scene is surfaces.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"time"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/web"
)

func main() {
	out := flag.String("o", "gl.png", "output PNG")
	width := flag.Int("w", 1280, "width")
	height := flag.Int("h", 900, "height")
	preset := flag.String("preset", "web-light", "theme preset")
	delay := flag.Duration("delay", 1200*time.Millisecond, "time to let the window render before capturing")
	path := flag.String("qml", "", "scene file")
	flag.Parse()
	if *path == "" && flag.NArg() > 0 {
		*path = flag.Arg(0)
	}
	if *path == "" {
		fmt.Fprintln(os.Stderr, "glshot: a scene file is required (-qml or the last argument)")
		os.Exit(2)
	}
	scene, err := web.Load(*path, *preset, *width, *height)
	if err != nil {
		fmt.Fprintf(os.Stderr, "glshot: %v\n", err)
		os.Exit(1)
	}
	root, ok := scene.Root.(*meui.Window)
	if !ok {
		fmt.Fprintf(os.Stderr, "glshot: root is %T, want *meui.Window\n", scene.Root)
		os.Exit(1)
	}

	// No chrome: the offscreen shot has no title bar either, so the two frames are
	// directly comparable.
	wnd, err := sdl3.NewWindowSized("glshot", int32(*width), int32(*height))
	if err != nil {
		fmt.Fprintf(os.Stderr, "glshot: window: %v\n", err)
		os.Exit(1)
	}
	wnd.SetContentRect(graphics.NewRect(0, 0, float32(*width), float32(*height)))
	root.SetContentOrigin(0, 0)
	wnd.SetContent(root.AsPanel())
	wnd.SetSink(root)
	wnd.ValidateLayout()
	wnd.Show()

	done := make(chan struct{})
	go func() {
		time.Sleep(*delay)
		wnd.CaptureNext(func(pix []uint32, cw, ch int32) {
			if err := writePNG(*out, pix, cw, ch); err != nil {
				fmt.Fprintf(os.Stderr, "glshot: %v\n", err)
			} else {
				fmt.Printf("glshot: wrote %s (%dx%d)\n", *out, cw, ch)
			}
			wnd.Close()
			close(done)
		})
	}()
	sdl3.Start(nil)
	<-done
}

// writePNG stores the framebuffer. It is BOTTOM-UP, like the device's.
func writePNG(path string, pixels []uint32, w, h int32) error {
	img := image.NewNRGBA(image.Rect(0, 0, int(w), int(h)))
	for y := int32(0); y < h; y++ {
		srcY := h - 1 - y
		for x := int32(0); x < w; x++ {
			px := pixels[srcY*w+x]
			i := int(y)*img.Stride + int(x)*4
			img.Pix[i+0] = byte(px >> 16)
			img.Pix[i+1] = byte(px >> 8)
			img.Pix[i+2] = byte(px)
			img.Pix[i+3] = byte(px >> 24)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
