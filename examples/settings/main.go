// Settings — the fourth web scene: the CONTROLS under the web preset, wired.
//
//	go run ./examples/settings
//	go run ./examples/settings -shot out.png
//
// The form is the point: TextField, ComboBox, Switch, CheckBox, RadioButton and a real
// Button all render from the same widget code the enterprise demo uses, restyled by the
// `web-light` preset and its control tweaks. Only the primary Button and the inputs are
// web variants; the rest follows the tokens.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/web"
	"github.com/sixnineone/meui/widgets"
)

var (
	inkActiveInk = graphics.Color(0xFF4F46E5)
	inkIdleInk   = graphics.Color(0xFF4B5563)
)

func main() {
	shot := flag.String("shot", "", "render one frame to this PNG and exit (headless, no display)")
	preset := flag.String("preset", "web-light", "theme preset")
	width := flag.Int("w", 1280, "window width")
	height := flag.Int("h", 900, "window height")
	path := flag.String("qml", "examples/settings/settings.qml", "scene file")
	flag.Parse()

	scene, err := web.Load(*path, *preset, *width, *height)
	if err != nil {
		fmt.Fprintf(os.Stderr, "settings: %v\n", err)
		os.Exit(1)
	}
	wire(scene)

	if *shot != "" {
		if err := scene.Shot(*width, *height, *shot); err != nil {
			fmt.Fprintf(os.Stderr, "settings: shot: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("settings: wrote %s (%dx%d)\n", *shot, *width, *height)
		return
	}

	root, ok := scene.Root.(*meui.Window)
	if !ok {
		fmt.Fprintf(os.Stderr, "settings: root is %T, want *meui.Window\n", scene.Root)
		os.Exit(1)
	}
	root.SetFrameRect(graphics.NewRect(0, 0, float32(*width), float32(*height)))
	app := meui.NewApp(sdl3.NewApp(), meui.AppOptions{Title: "meui · settings"})
	if _, err := app.AddWindow(root, meui.WindowOptions{
		Title: "meui · settings",
		Size:  graphics.NewSize(float32(*width), float32(*height)),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "settings: add window: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "settings: run: %v\n", err)
		os.Exit(1)
	}
}

func wire(scene *web.Scene) {
	sections := []struct{ id, title string }{
		{"navAccount", "Account"},
		{"navNotifications", "Notifications"},
		{"navSecurity", "Security"},
		{"navBilling", "Billing"},
	}

	selectSection := func(chosen string) {
		for _, s := range sections {
			box, ok := scene.Get(s.id).(*meui.Panel)
			if !ok || box == nil {
				continue
			}
			active := s.id == chosen
			scene.Select(s.id, active)
			for _, child := range box.Children() {
				if l, ok := child.(*widgets.Label); ok {
					if active {
						l.SetColor(inkActiveInk)
						l.SetFontBold(true)
					} else {
						l.SetColor(inkIdleInk)
						l.SetFontBold(false)
					}
				}
			}
			box.MarkForRedraw()
			if active {
				scene.SetText("pageTitle", s.title)
			}
		}
	}
	for _, s := range sections {
		s := s
		scene.OnClick(s.id, func() { selectSection(s.id) })
	}

	// Save: the toast is a box whose visibility Go flips, which is all a snackbar is
	// when a page wants it in place rather than at the window's edge.
	save := func() {
		if w, ok := scene.Get("toast").(*meui.Panel); ok {
			w.SetVisible(true)
		}
		scene.SetText("saveHint", "Saved just now — the workspace keeps the change.")
	}
	if b, ok := scene.Get("saveButton").(*widgets.Button); ok {
		b.SetOnClicked(save)
	}
	scene.OnClick("cancelButton", func() {
		if w, ok := scene.Get("toast").(*meui.Panel); ok {
			w.SetVisible(false)
		}
		scene.SetText("saveHint", "Changes are saved to this workspace.")
	})
	scene.OnClick("deleteButton", func() {
		scene.SetText("saveHint", "Deleting a workspace needs a confirmation dialog — not wired in this demo.")
	})
	scene.OnClick("navBilling", func() {
		scene.SetText("saveHint", "Billing lives in another page — this is the account screen.")
	})

	if sw, ok := scene.Get("switchProduct").(*widgets.Switch); ok {
		sw.SetOnToggled(func(on bool) {
			if on {
				scene.SetText("saveHint", "Product updates: on.")
			} else {
				scene.SetText("saveHint", "Product updates: off.")
			}
		})
	}
	if sw, ok := scene.Get("switchDigest").(*widgets.Switch); ok {
		sw.SetOnToggled(func(on bool) {
			if on {
				scene.SetText("saveHint", "Weekly digest: on.")
			} else {
				scene.SetText("saveHint", "Weekly digest: off.")
			}
		})
	}
	if cb, ok := scene.Get("checkWrap").(*widgets.CheckBox); ok {
		cb.SetOnCheckChanged(func(on bool) {
			if on {
				scene.SetText("saveHint", "Line wrapping: on.")
			} else {
				scene.SetText("saveHint", "Line wrapping: off.")
			}
		})
	}
}
