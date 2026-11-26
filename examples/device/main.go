// demoapp runs the real-device demo on the desktop.
//
// The same scene and the same wiring run on Android through meui-android, so the
// desktop is the fast loop for everything except the device-only capabilities:
// a capability that reports NOT SUPPORTED here is expected, and one that reports
// NOT SUPPORTED on a phone is the finding.
//
//	go run ./cmd/demoapp          # 400x760, phone-shaped
//	go run ./cmd/demoapp -size 1100x750
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/examples/device/demoapp"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/widgets"
)

func main() {
	size := flag.String("size", "400x760", "window size, WxH")
	shot := flag.String("shot", "", "render one page to this PNG and exit")
	page := flag.Int("page", 0, "which page to capture with -shot (0=Widgets 1=Device 2=Render 3=Perf)")
	sheet := flag.Bool("sheet", false, "open the demo's bottom sheet before capturing")
	actions := flag.Bool("actions", false, "open the demo's action sheet before capturing")
	alert := flag.Bool("dialog", false, "open the demo's adaptive dialog before capturing")
	picker := flag.Bool("picker", false, "open the demo's date picker before capturing")
	search := flag.Bool("search", false, "open the demo's search view before capturing")
	swipe := flag.Bool("swipe", false, "reveal the demo's swipe row's actions before capturing")
	pull := flag.Bool("pull", false, "drag the demo's page down mid-pull before capturing")
	frameless := flag.Bool("frameless", false,
		"skip the desktop title bar; needed to capture a MODAL layer, which RenderToImage does not paint while a title bar owns layer painting")
	flag.Parse()

	var w, h int
	if _, err := fmt.Sscanf(strings.ReplaceAll(*size, "x", " "), "%d %d", &w, &h); err != nil || w <= 0 || h <= 0 {
		fmt.Fprintf(os.Stderr, "bad -size %q, want WxH\n", *size)
		os.Exit(2)
	}

	app := meui.NewApp(sdl3.NewApp(), meui.AppOptions{Title: "MeUi demo"})

	// Parse FIRST: the QML root IS the window that must be added. Adding an empty
	// window and then parsing a second one leaves the live window blank while a
	// direct RenderToImage of the parsed root looks perfect.
	demo, err := demoapp.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "demo: %v\n", err)
		os.Exit(1)
	}
	root := demo.Root
	root.SetFrameRect(graphics.NewRect(0, 0, float32(w), float32(h)))
	wnd, err := app.AddWindow(root, meui.WindowOptions{
		Title: "MeUi demo",
		Size:  graphics.NewSize(float32(w), float32(h)),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "add window: %v\n", err)
		os.Exit(1)
	}
	if sw, ok := wnd.(*sdl3.Window); ok && !*frameless {
		frame := widgets.NewWindowFrame()
		frame.SetOwnerRoot(root)
		frame.SetTitle("MeUi demo")
		frame.OnClose(func() { sw.Close() })
		sw.SetTitleBar(frame)
		// Chrome is the host's: the content tree is 0-based and starts BELOW the
		// bar, which the host declares with BOTH calls — SetContentRect reserves
		// the strip on the platform window, and SetContentOrigin tells the tree
		// where its (0,0) now sits. Setting only the origin leaves the demo's own
		// StatusBar drawn underneath the title bar.
		barH := float32(widgets.FrameBarHeight)
		sw.SetContentRect(graphics.NewRect(0, 0, float32(w), float32(h)-barH))
		root.SetContentOrigin(0, barH)
	}

	// The demo builds its own scene and wires it; the host only supplies the app
	// and the window, which is the same split the Android host uses.
	demo.Wire(app, wnd)
	demo.ShowPage(*page)
	if *sheet {
		// Opened through the window's modal layer, which is how an app shows it. The capture
		// below paints the window, layers included.
		demo.ShowSheet(nil)
	}
	if *actions {
		demo.ShowActionSheet(nil)
	}
	if *alert {
		demo.ShowDialog(nil)
	}
	if *picker {
		demo.ShowPicker()
	}
	if *search {
		demo.ShowSearch(nil, nil)
	}
	if *swipe {
		demo.OpenSwipeRow()
	}
	if *pull {
		demo.PullRefresh()
	}

	if *shot != "" {
		// Capture path: select the page, let it settle, write a PNG, exit. It
		// renders through RenderToImage, the path that shows the SETTLED layout.
		if sw, ok := wnd.(*sdl3.Window); ok {
			sw.CaptureAfterFrame(8, func(pix []uint32, wpx, hpx int32) {
				if os.Getenv("MEUI_PANE_DEBUG") != "" {
					demo.DumpChrome()
				}
				img, err := root.RenderToImage(int(wpx), int(hpx))
				if err != nil {
					fmt.Fprintf(os.Stderr, "shot: %v\n", err)
					os.Exit(1)
				}
				data, err := img.ToPNG(6)
				if err != nil {
					fmt.Fprintf(os.Stderr, "shot: encode: %v\n", err)
					os.Exit(1)
				}
				if err := os.WriteFile(*shot, data, 0o644); err != nil {
					fmt.Fprintf(os.Stderr, "shot: write: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("wrote %s (%dx%d)\n", *shot, img.Width(), img.Height())
				sw.Close()
			})
		}
	}

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", err)
		os.Exit(1)
	}
}
