// Docs — the third web scene: a fixed top bar, a fixed sidebar and an article that
// SCROLLS under them, with a real image at the top.
//
//	go run ./examples/docs
//	go run ./examples/docs -shot out.png
//
// The sidebar is the wiring demonstration: choosing a page retexts the H1, moves the
// active marker and restyles the item boxes — all with setter calls, because a nav
// item is a box, not a NavItem widget.
package main

import (
	"embed"
	"flag"
	"fmt"
	"os"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/resources"
	"github.com/sixnineone/meui/web"
	"github.com/sixnineone/meui/widgets"
)

// The scene and its assets are bundled. Everything under resources/ is picked
// up by the build; the scene refers to them as res://.
//
//go:embed all:resources
var docsFS embed.FS

var (
	inkActiveInk = graphics.Color(0xFF4F46E5)
	inkIdleInk   = graphics.Color(0xFF4B5563)
)

func main() {
	shot := flag.String("shot", "", "render one frame to this PNG and exit (headless, no display)")
	preset := flag.String("preset", "web-light", "theme preset")
	width := flag.Int("w", 1280, "window width")
	height := flag.Int("h", 900, "window height")
	path := flag.String("qml", "", "scene file override (disk path)")
	scroll := flag.Float64("scroll", 0, "scroll the article to this offset before shooting")
	flag.Parse()

	var (
		scene *web.Scene
		err   error
	)
	if *path != "" {
		scene, err = web.Load(*path, *preset, *width, *height)
	} else {
		bundle := resources.Default()
		if err := bundle.Mount("res", docsFS, "resources"); err != nil {
			fmt.Fprintf(os.Stderr, "docs: mount resources: %v\n", err)
			os.Exit(1)
		}
		scene, err = web.LoadBundle(bundle, "res://docs.qml", *preset, *width, *height)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "docs: %v\n", err)
		os.Exit(1)
	}
	wire(scene)

	if *shot != "" {
		if *scroll > 0 {
			if sp, ok := scene.Get("articleView").(*widgets.ScrollPane); ok {
				sp.ScrollTo(float32(*scroll))
				sp.RefreshSticky()
			}
		}
		if err := scene.Shot(*width, *height, *shot); err != nil {
			fmt.Fprintf(os.Stderr, "docs: shot: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("docs: wrote %s (%dx%d)\n", *shot, *width, *height)
		return
	}

	root, ok := scene.Root.(*meui.Window)
	if !ok {
		fmt.Fprintf(os.Stderr, "docs: root is %T, want *meui.Window\n", scene.Root)
		os.Exit(1)
	}
	root.SetFrameRect(graphics.NewRect(0, 0, float32(*width), float32(*height)))
	app := meui.NewApp(sdl3.NewApp(), meui.AppOptions{Title: "meui · atlas docs"})
	if _, err := app.AddWindow(root, meui.WindowOptions{
		Title: "meui · atlas docs",
		Size:  graphics.NewSize(float32(*width), float32(*height)),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "docs: add window: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "docs: run: %v\n", err)
		os.Exit(1)
	}
}

type page struct {
	id, title string
}

func wire(scene *web.Scene) {
	pages := []page{
		{"navIntro", "Surfaces, elevation and the Box"},
		{"navInstall", "Installation"},
		{"navQuick", "Quick start"},
		{"navSurfaces", "Surfaces & elevation"},
		{"navType", "The type scale"},
		{"navScroll", "Scrolling a page"},
		{"navImages", "Images"},
	}

	selectPage := func(chosen page) {
		for _, p := range pages {
			box, ok := scene.Get(p.id).(*meui.Panel)
			if !ok || box == nil {
				continue
			}
			active := p.id == chosen.id
			// The wash is the THEME's Selected ink now; the label still needs the
			// active accent, which is a text role rather than a surface state.
			scene.Select(p.id, active)
			// The item's label is its child: one selection rule, applied to the box and
			// to the ink, so a page cannot be "selected" in two places disagreement.
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
		}
		scene.SetText("articleTitle", chosen.title)
		scene.SetText("statusText", "Showing “"+chosen.title+"”. Scroll the article — the wheel goes to the viewport.")
	}

	for _, p := range pages {
		p := p
		scene.OnClick(p.id, func() { selectPage(p) })
	}

	scene.OnClick("searchBox", func() {
		scene.SetText("statusText", "Search would open here — the field is a box with a hint, wired in Go.")
	})

	// The page reflows. A docs shell has one breakpoint that matters: below it the
	// sidebar is a drawer, not a column, so it goes and the article takes the width.
	scene.Breakpoints([]web.Breakpoint{
		{MinWidth: 0, Apply: compactDocs},
		{MinWidth: 1000, Apply: wideDocs},
	})
}

// compactDocs is the narrow layout: no sidebar, the article spans the page, and the
// top bar's right-hand links give way to the search field and the sign-in button.
func compactDocs(scene *web.Scene, w, h int32) {
	fw := float32(w)
	fh := float32(h)
	scene.Show("sidebar", false)
	scene.SetFrame("topBar", 0, 0, fw, 64)
	scene.Show("linkGuides", false)
	scene.Show("linkAPI", false)
	scene.Show("linkGitHub", false)
	scene.SetFrame("signInButton", fw-82, 18, 58, 28)
	scene.SetFrame("versionPill", fw-154, 20, 56, 24)
	// Leave the logo and wordmark ("Atlas docs") their corner: the search field starts
	// after them and stops before the version pill.
	scene.SetFrame("searchBox", 270, 14, maxf(160, fw-460), 36)
	scene.SetFrame("articleView", 0, 64, fw, fh-64)
	scene.SetFrame("articlePage", 0, 0, fw, 1880)
	scene.SetFrame("cover", 24, 44, maxf(280, fw-48), 220)
}

// wideDocs is the reference layout: the sidebar column, a centred article column and
// the top bar's links.
func wideDocs(scene *web.Scene, w, h int32) {
	fw := float32(w)
	fh := float32(h)
	scene.Show("sidebar", true)
	scene.SetFrame("topBar", 0, 0, fw, 64)
	scene.Show("linkGuides", true)
	scene.Show("linkAPI", true)
	scene.Show("linkGitHub", true)
	scene.SetFrame("versionPill", fw-150, 20, 56, 24)
	scene.SetFrame("signInButton", fw-82, 18, 58, 28)
	scene.SetFrame("searchBox", 300, 14, 380, 36)
	scene.SetFrame("articleView", 260, 64, fw-260, fh-64)
	scene.SetFrame("articlePage", 0, 0, fw-260, 1880)
	scene.SetFrame("cover", 60, 44, minf(820, fw-380), 220)
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
