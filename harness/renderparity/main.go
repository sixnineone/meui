// renderparity renders a QML file through the meuishim/mobilegles stack
// and captures the framebuffer to PNG for visual comparison against the
// Skia reference renders (see RENDER_PARITY_PLAN.md).
//
// Usage:
//
//	go run ./harness/renderparity [-frames N] [-o outdir] [-w W] [-h H] <file.qml>
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/qml"
	"github.com/sixnineone/meui/theme"
	"github.com/sixnineone/mobilegles"
)

// pixelsToImage converts 0xAARRGGBB bottom-up pixels to an NRGBA image.
func pixelsToImage(pix []uint32, w, h int32) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, int(w), int(h)))
	for y := 0; y < int(h); y++ {
		src := (int(h) - 1 - y) * int(w) // GL is bottom-up; flip
		for x := 0; x < int(w); x++ {
			px := pix[src+x]
			i := (y*int(w) + x) * 4
			img.Pix[i+0] = byte((px >> 16) & 0xFF) // R
			img.Pix[i+1] = byte((px >> 8) & 0xFF)  // G
			img.Pix[i+2] = byte(px & 0xFF)         // B
			img.Pix[i+3] = byte((px >> 24) & 0xFF) // A
		}
	}
	return img
}

func main() {
	frames := flag.Int("frames", 5, "warmup frames before capture")
	offscreen := flag.Bool("offscreen", false,
		"capture through RenderToImage (the offscreen GL path) instead of the window framebuffer.\n"+
			"KNOWN BROKEN: nesting RenderToImage inside the capture callback renders the content\n"+
			"FLIPPED (85.3 mean-delta on test_bigrect, 0.0 when flipped), while the same call from a\n"+
			"normal draw hook is correct. Do not trust its output until that is fixed — the flag is\n"+
			"kept so the bug stays visible (MOBILE-PLAN.md P5.6).")
	outDir := flag.String("o", "harness/renderparity/out", "output directory")
	width := flag.Float64("w", 0, "window width override (0 = from QML)")
	height := flag.Float64("h", 0, "window height override (0 = from QML)")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: renderparity [-frames N] [-o dir] <file.qml>")
		os.Exit(2)
	}
	qmlPath := flag.Arg(0)
	base := filepath.Base(qmlPath)
	outName := strings.TrimSuffix(base, filepath.Ext(base)) + ".png"

	// NOTE: no theme import — the Skia reference renders use the default
	// DarkTheme; both stacks must render their defaults for parity.
	_ = theme.DefaultTheme

	result, err := qml.ParseWithBindingsFile(qmlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading QML: %v\n", err)
		os.Exit(1)
	}
	win, ok := result.Widget.(*meui.Window)
	if !ok {
		fmt.Fprintf(os.Stderr, "QML root is not a meui.Window\n")
		os.Exit(1)
	}
	panel := win.AsPanel()
	if panel == nil {
		fmt.Fprintf(os.Stderr, "Window has no Panel\n")
		os.Exit(1)
	}

	sdl3.AttachConsole()

	sdl3.Start(sdl3.StartupFinishedCallback(func() {
		w := float32(900)
		h := float32(700)
		if *width > 0 {
			w = float32(*width)
		} else if fw := win.FrameRect().Width; fw > 0 {
			w = fw
		}
		if *height > 0 {
			h = float32(*height)
		} else if fh := win.FrameRect().Height; fh > 0 {
			h = fh
		}

		// The window layer has no runtime resize; create at QML size.
		wnd, err := sdl3.NewWindowSized("RenderParity - "+base, int32(w), int32(h))
		if err != nil {
			fmt.Fprintf(os.Stderr, "window: %v\n", err)
			os.Exit(1)
		}

		wnd.SetContentRect(graphics.NewRect(0, 0, w, h))
		wnd.ValidateLayout()
		wnd.SetContent(panel)
		wnd.SetSink(win) // route SDL input into the meui event sink

		var saved bool
		wnd.CaptureAfterFrame(*frames, func(pix []uint32, pw, ph int32) {
			if len(pix) == 0 {
				fmt.Fprintf(os.Stderr, "capture failed: empty framebuffer\n")
				os.Exit(1)
			}
			// The offscreen path: the same GL renderer, into an FBO at an
			// arbitrary size. The device has to use this one (Android forces the
			// window fullscreen), so a comparison between the two platforms is
			// only meaningful if both are captured the same way.
			img := pixelsToImage(pix, pw, ph)
			if *offscreen {
				// DIAGNOSTIC: the RAW nested readback, with no flip at all, so the
				// orientation question can be settled by measurement instead of by
				// reading RenderOffscreen's save/restore code.
				raw := mobilegles.RenderOffscreen(pw, ph, func() {
					canvas := graphics.NewCanvas(graphics.NewRendererDevice(float32(pw), float32(ph)))
					win.Paint(canvas, graphics.NewRect(0, 0, float32(pw), float32(ph)))
				})
				if raw == nil {
					fmt.Fprintf(os.Stderr, "offscreen capture: nil pixels\n")
					os.Exit(1)
				}
				img = pixelsToImage(raw, pw, ph)
			}
			if err := os.MkdirAll(*outDir, 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
				os.Exit(1)
			}
			out := filepath.Join(*outDir, outName)
			f, err := os.Create(out)
			if err != nil {
				fmt.Fprintf(os.Stderr, "create: %v\n", err)
				os.Exit(1)
			}
			defer f.Close()
			if err := png.Encode(f, img); err != nil {
				fmt.Fprintf(os.Stderr, "png: %v\n", err)
				os.Exit(1)
			}
			saved = true
			log.Printf("captured %s (%dx%d)", out, pw, ph)
		})

		wnd.Show()

		// Safety: if capture never fired (window closed by user), exit.
		platform.InvokeTask(func() {
			go func() {
				for i := 0; i < 300; i++ { // ~30s watchdog
					if saved {
						return
					}
					sleepMs(100)
				}
				if !saved {
					fmt.Fprintf(os.Stderr, "watchdog: capture did not complete\n")
					os.Exit(1)
				}
			}()
		})
	}))
}

func sleepMs(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}
