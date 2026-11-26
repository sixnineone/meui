// Landing — the first web scene, runnable and wireable.
//
//	go run ./examples/landing           # a real window, real input
//	go run ./examples/landing -shot out.png   # the same scene, headless
//
// The scene's BEHAVIOUR lives here, wired by id, because that is this toolkit's
// model: QML declares the surface, Go decides what it does. `web.Scene` is the thin
// layer that keeps this file to a page of wiring.
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

func main() {
	shot := flag.String("shot", "", "render one frame to this PNG and exit (headless, no display)")
	preset := flag.String("preset", "web-light", "theme preset")
	width := flag.Int("w", 1280, "window width")
	height := flag.Int("h", 900, "window height")
	path := flag.String("qml", "examples/landing/landing.qml", "scene file")
	flag.Parse()

	scene, err := web.Load(*path, *preset, *width, *height)
	if err != nil {
		fmt.Fprintf(os.Stderr, "landing: %v\n", err)
		os.Exit(1)
	}
	wire(scene)

	if *shot != "" {
		if err := scene.Shot(*width, *height, *shot); err != nil {
			fmt.Fprintf(os.Stderr, "landing: shot: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("landing: wrote %s (%dx%d)\n", *shot, *width, *height)
		return
	}

	root, ok := scene.Root.(*meui.Window)
	if !ok {
		fmt.Fprintf(os.Stderr, "landing: root is %T, want *meui.Window\n", scene.Root)
		os.Exit(1)
	}
	root.SetFrameRect(graphics.NewRect(0, 0, float32(*width), float32(*height)))
	app := meui.NewApp(sdl3.NewApp(), meui.AppOptions{Title: "meui · landing"})
	if _, err := app.AddWindow(root, meui.WindowOptions{
		Title: "meui · landing",
		Size:  graphics.NewSize(float32(*width), float32(*height)),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "landing: add window: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "landing: run: %v\n", err)
		os.Exit(1)
	}
}

// wire is the whole behaviour of the scene. Note what it does NOT do: no geometry,
// no styling, no painting. A scene that needs those has a widget to fix, not a page
// to special-case.
func wire(scene *web.Scene) {
	status := func(msg string) {
		scene.SetText("statusText", msg)
	}

	// A clickable box: the Rectangle carries `clickable: true`, the handler is here.
	scene.OnClick("ctaPrimary", func() {
		status("Practice started — question 01, “Why do LLMs hallucinate?”, is loading.")
	})
	scene.OnClick("ctaSecondary", func() {
		status("Opening a sample answer: 60 seconds, spoken, then graded.")
	})
	scene.OnClick("ctaHeader", func() {
		status("Account created — check your inbox for the first three questions.")
	})
	scene.OnClick("navSignIn", func() {
		status("Signed in as siya — resuming at question 41.")
	})

	// A MouseArea: the input region is the circle, not the box behind it.
	scene.OnClick("demoPlayArea", func() {
		status("Playing the demo — the play area is a MouseArea, wired in Go.")
	})

	// The play triangle is PAINTED, not a glyph: the embedded face has no U+25B6 and
	// a scene that renders a tofu box looks broken on exactly the machines we cannot
	// test. OnPaint is the escape hatch for chrome the widget set does not have.
	scene.OnPaint("demoPlay", func(c *graphics.Canvas, r graphics.Rect) {
		cx, cy := r.Width/2, r.Height/2
		p := graphics.NewPath()
		p.MoveTo(graphics.Point{X: cx - 6, Y: cy - 9})
		p.LineTo(graphics.Point{X: cx + 10, Y: cy})
		p.LineTo(graphics.Point{X: cx - 6, Y: cy + 9})
		p.Close()
		paint := graphics.NewPaint()
		paint.SetColor(graphics.Color(0xFF14243C))
		c.DrawPath(p, paint)
	})

	// Hover is a scene concern; the box's own wash already comes from the theme.
	scene.OnHover("ctaPrimary", func(inside bool) {
		if inside {
			status("Hovering over the primary action — the wash is the theme's Hovered ink.")
			return
		}
		status("Ready. Try a button — every control here is wired in Go by id.")
	})
}
