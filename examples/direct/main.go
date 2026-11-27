// Direct — the same window built with widgets and setters: no .qml file, no flow
// wrappers. This is the imperative API underneath the other two builders.
//
//	go run ./examples/direct
//	go run ./examples/direct -shot /tmp/direct.png
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/layout"
	"github.com/sixnineone/meui/theme"
	"github.com/sixnineone/meui/web"
	"github.com/sixnineone/meui/widgets"
)

func main() {
	shot := flag.String("shot", "", "render one frame to this PNG and exit (headless)")
	preset := flag.String("preset", "web-light", "theme preset")
	width := flag.Int("w", 900, "window width")
	height := flag.Int("h", 600, "window height")
	flag.Parse()

	if err := theme.LoadPreset(*preset); err != nil {
		fmt.Fprintf(os.Stderr, "direct: preset: %v\n", err)
		os.Exit(1)
	}

	root := build(float32(*width), float32(*height))

	if *shot != "" {
		if err := web.Shot(root, *width, *height, *shot); err != nil {
			fmt.Fprintf(os.Stderr, "direct: shot: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("direct: wrote %s (%dx%d)\n", *shot, *width, *height)
		return
	}

	app := meui.NewApp(sdl3.NewApp(), meui.AppOptions{Title: "meui · direct"})
	if _, err := app.AddWindow(root, meui.WindowOptions{
		Title: "meui · direct",
		Size:  graphics.NewSize(float32(*width), float32(*height)),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "direct: add window: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "direct: run: %v\n", err)
		os.Exit(1)
	}
}

// build is the whole page. No parser is involved: each widget is a value, each
// behaviour a closure, and the VStack owns the geometry.
func build(w, h float32) *meui.Window {
	win := meui.NewWindow()
	win.SetFrameRect(graphics.NewRect(0, 0, w, h))

	page := meui.NewPanel()
	page.SetFrameRect(graphics.NewRect(0, 0, w, h))
	page.SetBackground(graphics.RGB(0xF7, 0xF5, 0xF0))
	win.AddChild(page)

	// The VStack has no padding of its own: inset its frame instead.
	content := meui.NewPanel()
	content.SetFrameRect(graphics.NewRect(40, 40, w-80, h-80))
	content.SetLayout(layout.NewVStack())
	page.AddChild(content)

	title := widgets.NewLabel()
	title.SetText("Direct Go")
	title.SetFontSize(40)
	content.AddChild(title)

	subtitle := widgets.NewLabel()
	subtitle.SetText("A window built from widgets, setters and a VStack. The same primitives the QML builder produces.")
	subtitle.SetWrapMode("word")
	content.AddChild(subtitle)

	cta := widgets.NewButton()
	cta.SetText("Say hello")
	cta.SetFrameRect(graphics.NewRect(40, 150, 140, 36))
	cta.SetOnClicked(func() {
		title.SetText("Hello, Go.")
	})
	content.AddChild(cta)

	return win
}
