// Flow — the same window declared as Go structs and built by the flow package. No
// .qml file: the tree is compile-checked, and behaviour sits next to the declaration.
//
//	go run ./examples/flow
//	go run ./examples/flow -shot /tmp/flow.png
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/flow"
	"github.com/sixnineone/meui/graphics"
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
		fmt.Fprintf(os.Stderr, "flow: preset: %v\n", err)
		os.Exit(1)
	}

	root := build(float32(*width), float32(*height))

	if *shot != "" {
		if err := web.Shot(root, *width, *height, *shot); err != nil {
			fmt.Fprintf(os.Stderr, "flow: shot: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("flow: wrote %s (%dx%d)\n", *shot, *width, *height)
		return
	}

	app := meui.NewApp(sdl3.NewApp(), meui.AppOptions{Title: "meui · flow"})
	if _, err := app.AddWindow(root, meui.WindowOptions{
		Title: "meui · flow",
		Size:  graphics.NewSize(float32(*width), float32(*height)),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "flow: add window: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "flow: run: %v\n", err)
		os.Exit(1)
	}
}

// build is the whole page. Each field mirrors a QML property; Build() turns the
// structs into the same widget tree the parser would have produced.
func build(w, h float32) *meui.Window {
	var title *widgets.Label

	app := &flow.Window{
		W:     flow.W{Width: w, Height: h},
		Title: "meui · flow",
		Children: []flow.WidgetWrapper{
			// The Panel has no layout, so the VStack keeps the inset frame: the
			// window's anchor layout sizes the Panel, not the content inside it.
			&flow.Panel{
				Children: []flow.WidgetWrapper{
					&flow.VStack{
						W: flow.W{X: 40, Y: 40, Width: w - 80, Height: h - 80},
						Children: []flow.WidgetWrapper{
							&flow.Label{
								Text: "Declarative Go",
								OnCreated: func(l *widgets.Label) {
									title = l
									l.SetFontSize(40)
								},
							},
							&flow.Label{
								Text: "The same page as Go structs, built by flow. The compiler checks it; there is no file to parse.",
								OnCreated: func(l *widgets.Label) {
									l.SetWrapMode("word")
								},
							},
							&flow.Button{
								W:    flow.W{Width: 140, Height: 36},
								Text: "Say hello",
								OnClick: func() {
									title.SetText("Hello, Go.")
								},
							},
						},
					},
				},
			},
		},
	}

	root, _ := flow.Build(app)
	return root.(*meui.Window)
}
