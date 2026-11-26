// Dashboard — the second web scene: a gradient sky, frosted cards and a day strip
// whose cells RESTYLE themselves from Go when chosen.
//
//	go run ./examples/dashboard
//	go run ./examples/dashboard -shot out.png
//
// The point of the day strip is that selection is not a widget feature here: a
// Rectangle is a box, and the border colour and fill that mark the chosen day are
// three setter calls on it. That is what "copy the ideas, not the framework" means
// in practice — no SelectableCard widget, just a box the app paints.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/web"
)

const (
	inkSelectedBorder = graphics.Color(0xFF2C7BB6)
	inkRestBorder     = graphics.Color(0xFFDCEBF3)
)

func main() {
	shot := flag.String("shot", "", "render one frame to this PNG and exit (headless, no display)")
	preset := flag.String("preset", "web-light", "theme preset")
	width := flag.Int("w", 1280, "window width")
	height := flag.Int("h", 900, "window height")
	path := flag.String("qml", "examples/dashboard/dashboard.qml", "scene file")
	flag.Parse()

	scene, err := web.Load(*path, *preset, *width, *height)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dashboard: %v\n", err)
		os.Exit(1)
	}
	wire(scene)

	if *shot != "" {
		if err := scene.Shot(*width, *height, *shot); err != nil {
			fmt.Fprintf(os.Stderr, "dashboard: shot: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("dashboard: wrote %s (%dx%d)\n", *shot, *width, *height)
		return
	}

	root, ok := scene.Root.(*meui.Window)
	if !ok {
		fmt.Fprintf(os.Stderr, "dashboard: root is %T, want *meui.Window\n", scene.Root)
		os.Exit(1)
	}
	root.SetFrameRect(graphics.NewRect(0, 0, float32(*width), float32(*height)))
	app := meui.NewApp(sdl3.NewApp(), meui.AppOptions{Title: "meui · nimbus"})
	if _, err := app.AddWindow(root, meui.WindowOptions{
		Title: "meui · nimbus",
		Size:  graphics.NewSize(float32(*width), float32(*height)),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "dashboard: add window: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "dashboard: run: %v\n", err)
		os.Exit(1)
	}
}

type day struct {
	id, temp, condition string
}

// wire gives the strip its behaviour: choosing a day restyles the cells and
// re-texts the hero. The weather is fictional; the plumbing is the demonstration.
func wire(scene *web.Scene) {
	days := []day{
		{"dayToday", "9°", "Clear and crisp"},
		{"dayTue", "10°", "Cloud moving in"},
		{"dayWed", "13°", "Rain by afternoon"},
		{"dayThu", "2°", "Sleet, then clear"},
		{"dayFri", "4°", "Cold and grey"},
		{"daySat", "7°", "Bright spells"},
		{"daySun", "9°", "Fog at dawn"},
	}

	selectDay := func(chosen day) {
		for _, d := range days {
			p, ok := scene.Get(d.id).(*meui.Panel)
			if !ok || p == nil {
				continue
			}
			if d.id == chosen.id {
				p.SetBorderWidth(2)
				p.SetBorderColor(inkSelectedBorder)
				p.SetSelected(true)
			} else {
				p.SetBorderWidth(1)
				p.SetBorderColor(inkRestBorder)
				p.SetSelected(false)
			}
			p.MarkForRedraw()
		}
		scene.SetText("tempValue", chosen.temp)
		scene.SetText("conditionText", chosen.condition)
	}

	for _, d := range days {
		d := d
		scene.OnClick(d.id, func() { selectDay(d) })
	}

	// The unit toggle is a clickable box whose label is a child Text; clicking it
	// reports rather than converts, because the numbers are decorative.
	scene.OnClick("unitToggle", func() {
		scene.SetText("conditionText", "Display in Fahrenheit — not wired to a sensor, this is a demo")
	})
}
