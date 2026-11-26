// The examples launcher: a window that lists the examples and starts the one you click.
//
//	go run ./examples/launcher
//	go run ./examples/launcher -shot /tmp/launcher.png
//
// Each example is its own PROCESS: the toolkit owns one app loop and one window per
// process, and a launcher that embedded six apps would be fighting that (and its own
// window). So a click runs `go run ./examples/<name>` from the module root — the demo
// owns its window, its theme and its lifetime.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/web"
	"github.com/sixnineone/meui/widgets"
)

// example is one entry in the list. The blurb is what the example SHOWS, not what it is
// about — the point of the list is the toolkit's surface area.
type example struct {
	Dir     string
	Title   string
	Blurb   string
	Feature string
}

var examples = []example{
	{"landing", "Landing page", "Display type, an offset card shadow, pill CTAs", "type, shadows, painted art"},
	{"dashboard", "Weather dashboard", "Full-bleed gradient, translucent cards, a bar chart of boxes", "gradients, translucency, restyling"},
	{"docs", "Documentation site", "Fixed chrome over a scrolling article, sticky section header", "scroll, sticky, images, breakpoints"},
	{"settings", "Settings form", "Fields, combo, switches, radios and a primary button", "the controls, restyled from tokens"},
	{"mail", "Inbox", "240 messages in a virtual list, with a frosted header", "virtual lists, selection, blur"},
	{"checkout", "Checkout", "Three steps, inline validation, a focus ring that shows the error", "forms, validation, state"},
	{"ops", "Ops console", "A service tree, a runs table, a detail pane and an activity log", "menubar, DataTable, tree, selection"},
	{"admin", "Team admin", "Dark console: nav rail, filters, member table and an invite dialog", "dark theme, filters, overlay, validation"},
	{"device", "Device demo", "A phone-shaped demo: widget pages, a device report, render and perf", "adaptive layout, sheets, capabilities"},
	{"enterprise", "Enterprise demo", "The imported QML kitchen sink: tabs, tables, editors, dialogs", "the original demo, every control"},
}

func main() {
	shot := flag.String("shot", "", "render one frame to this PNG and exit (headless)")
	preset := flag.String("preset", "web-light", "theme preset")
	width := flag.Int("w", 900, "window width")
	height := flag.Int("h", 920, "window height")
	path := flag.String("qml", "examples/launcher/launcher.qml", "scene file")
	flag.Parse()

	scene, err := web.Load(*path, *preset, *width, *height)
	if err != nil {
		fmt.Fprintf(os.Stderr, "launcher: %v\n", err)
		os.Exit(1)
	}
	root := moduleRoot()
	if root == "" {
		fmt.Fprintln(os.Stderr, "launcher: run this from inside the meui module (no go.mod found)")
	}
	wire(scene, root)

	if *shot != "" {
		if err := scene.Shot(*width, *height, *shot); err != nil {
			fmt.Fprintf(os.Stderr, "launcher: shot: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("launcher: wrote %s (%dx%d)\n", *shot, *width, *height)
		return
	}

	win, ok := scene.Root.(*meui.Window)
	if !ok {
		fmt.Fprintf(os.Stderr, "launcher: root is %T, want *meui.Window\n", scene.Root)
		os.Exit(1)
	}
	win.SetFrameRect(graphics.NewRect(0, 0, float32(*width), float32(*height)))
	app := meui.NewApp(sdl3.NewApp(), meui.AppOptions{Title: "meui · examples"})
	wnd, err := app.AddWindow(win, meui.WindowOptions{
		Title: "meui · examples",
		Size:  graphics.NewSize(float32(*width), float32(*height)),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "launcher: add window: %v\n", err)
		os.Exit(1)
	}
	// The quit button closes the launcher's window. The examples keep their own windows
	// and their own processes.
	if b, ok := scene.Get("closeButton").(*meui.Panel); ok {
		b.SetOnClick(func() { wnd.Close() })
	}
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "launcher: run: %v\n", err)
		os.Exit(1)
	}
}

// running tracks child examples so a second click does not start the same one twice.
var (
	runningMu sync.Mutex
	running   = map[string]bool{}
)

func wire(scene *web.Scene, moduleRoot string) {
	host, ok := scene.Get("listHost").(*meui.Panel)
	if !ok {
		fmt.Fprintln(os.Stderr, "launcher: launcher.qml has no listHost panel")
		return
	}
	for i, ex := range examples {
		ex := ex
		y := float32(i * 72)
		card := buildCard(ex, func() {
			launch(scene, moduleRoot, ex)
		})
		card.SetFrameRect(graphics.NewRect(0, y, 836, 64))
		host.AddChild(card)
	}
	scene.SetText("status", fmt.Sprintf("%d examples. Click one — it opens its own window.", len(examples)))
}

// buildCard is one row: the title and what the example shows, with an Open pill.
func buildCard(ex example, onClick func()) *meui.Panel {
	card := meui.NewPanel()
	card.SetCornerRadius(12)
	card.SetBackground(graphics.Color(0xFFFFFFFF))
	card.SetBorderWidth(1)
	card.SetBorderColor(graphics.Color(0xFFE5E0D6))
	card.SetClickable(true)
	card.SetOnClick(onClick)

	title := widgets.NewLabel()
	title.SetText(ex.Title)
	title.SetFontSize(12)
	title.SetFontBold(true)
	title.SetColor(graphics.Color(0xFF0F1115))
	title.SetFrameRect(graphics.NewRect(20, 9, 460, 22))
	card.AddChild(title)

	blurb := widgets.NewLabel()
	blurb.SetText(ex.Blurb)
	blurb.SetFontSize(9)
	blurb.SetColor(graphics.Color(0xFF4B5563))
	blurb.SetFrameRect(graphics.NewRect(20, 30, 560, 18))
	card.AddChild(blurb)

	feature := widgets.NewLabel()
	feature.SetText(ex.Feature)
	feature.SetFontSize(8)
	feature.SetColor(graphics.Color(0xFF9CA3AF))
	feature.SetFrameRect(graphics.NewRect(20, 46, 580, 16))
	card.AddChild(feature)

	pill := meui.NewPanel()
	pill.SetFrameRect(graphics.NewRect(700, 15, 112, 34))
	pill.SetCornerRadius(8)
	pill.SetBackground(graphics.Color(0xFF4F46E5))
	open := widgets.NewLabel()
	open.SetText("Open")
	open.SetFontSize(10)
	open.SetFontBold(true)
	open.SetColor(graphics.Color(0xFFFFFFFF))
	open.SetHAlign("center")
	open.SetFrameRect(graphics.NewRect(0, 0, 112, 34))
	pill.AddChild(open)
	card.AddChild(pill)
	return card
}

// launch starts the example as its own process and reports what happened in the status
// line. The child's output is captured, so a failure is shown rather than lost.
func launch(scene *web.Scene, moduleRoot string, ex example) {
	if moduleRoot == "" {
		scene.SetText("status", "Cannot launch: run the launcher from inside the meui module.")
		return
	}
	runningMu.Lock()
	if running[ex.Dir] {
		runningMu.Unlock()
		scene.SetText("status", ex.Title+" is already starting or open.")
		return
	}
	running[ex.Dir] = true
	runningMu.Unlock()

	scene.SetText("status", "Starting "+ex.Title+" — it opens in its own window…")
	cmd := exec.Command("go", "run", "./examples/"+ex.Dir)
	cmd.Dir = moduleRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		runningMu.Lock()
		delete(running, ex.Dir)
		runningMu.Unlock()
		scene.SetText("status", "Could not start "+ex.Title+": "+err.Error())
		return
	}
	go func() {
		err := cmd.Wait()
		runningMu.Lock()
		delete(running, ex.Dir)
		runningMu.Unlock()
		if err != nil {
			msg := strings.TrimSpace(out.String())
			if len(msg) > 160 {
				msg = msg[len(msg)-160:]
			}
			scene.SetText("status", ex.Title+" exited: "+msg)
			return
		}
		scene.SetText("status", ex.Title+" closed.")
	}()
}

// moduleRoot walks up from the working directory to the go.mod of this module.
func moduleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if b, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil {
			if strings.Contains(string(b), "module github.com/sixnineone/meui") {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
