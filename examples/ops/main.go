// Ops console — an enterprise-shaped application on the toolkit's own desktop theme.
//
//	go run ./examples/ops
//	go run ./examples/ops -shot /tmp/ops.png
//
// The imported `demo_meui_imported` app is the kitchen sink. This is what an internal tool
// looks like: a menubar, a header with actions, a service tree, a dense runs table, a
// detail pane driven by selection, an activity strip and a status bar. All wiring is by id
// from Go, as everywhere in this toolkit.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/web"
	"github.com/sixnineone/meui/widgets"
)

// runRow is the console's model. The table is a projection of it, so a filter or a new
// run rebuilds rows from one place.
type runRow struct {
	env      string
	service  string
	status   string
	duration string
	started  string
	owner    string
}

func seedRuns() []runRow {
	return []runRow{
		{"production", "api-gateway", "ok", "1.24s", "09:41:02", "platform"},
		{"production", "api-gateway", "ok", "1.31s", "09:38:11", "platform"},
		{"production", "billing", "warn", "4.02s", "09:35:47", "payments"},
		{"production", "billing", "ok", "2.18s", "09:30:02", "payments"},
		{"production", "search", "fail", "0.42s", "09:28:19", "discovery"},
		{"production", "search", "ok", "0.88s", "09:24:55", "discovery"},
		{"production", "worker", "ok", "12.4s", "09:20:10", "platform"},
		{"production", "worker", "warn", "18.7s", "09:12:44", "platform"},
		{"production", "api-gateway", "ok", "1.19s", "09:05:31", "platform"},
		{"staging", "api-gateway", "ok", "1.02s", "09:02:12", "platform"},
		{"staging", "billing", "ok", "1.77s", "08:58:40", "payments"},
		{"staging", "search", "fail", "0.39s", "08:51:03", "discovery"},
		{"staging", "search", "ok", "0.91s", "08:47:22", "discovery"},
		{"staging", "worker", "ok", "9.30s", "08:40:15", "platform"},
		{"production", "billing", "ok", "2.06s", "08:31:58", "payments"},
		{"production", "api-gateway", "ok", "1.28s", "08:25:07", "platform"},
	}
}

var statusInk = map[string]string{
	"ok":     "#0F766E",
	"warn":   "#B45309",
	"fail":   "#B91C1C",
	"queued": "#1D4ED8",
}

type console struct {
	scene  *web.Scene
	live   bool
	wnd    platform.Window
	all    []runRow
	filter string

	tree      *widgets.Tree
	table     *widgets.DataTable
	statusBar *widgets.StatusBar
	menuBar   *widgets.MenuBar
	snack     *widgets.Snackbar

	inspService, inspStatus, inspDuration, inspStarted, inspOwner *widgets.Label
	logs                                                          []*widgets.Label
	count                                                         *widgets.Label
}

func main() {
	shot := flag.String("shot", "", "render one frame to this PNG and exit (headless)")
	preset := flag.String("preset", "meui-light", "theme preset")
	width := flag.Int("w", 1280, "window width")
	height := flag.Int("h", 800, "window height")
	path := flag.String("qml", "examples/ops/ops.qml", "scene file")
	flag.Parse()

	scene, err := web.Load(*path, *preset, *width, *height)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ops: %v\n", err)
		os.Exit(1)
	}
	app := wire(scene, *shot == "")

	if *shot != "" {
		if err := scene.Shot(*width, *height, *shot); err != nil {
			fmt.Fprintf(os.Stderr, "ops: shot: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("ops: wrote %s (%dx%d)\n", *shot, *width, *height)
		return
	}

	win, ok := scene.Root.(*meui.Window)
	if !ok {
		fmt.Fprintf(os.Stderr, "ops: root is %T, want *meui.Window\n", scene.Root)
		os.Exit(1)
	}
	app2 := meui.NewApp(sdl3.NewApp(), meui.AppOptions{Title: "meui · ops"})
	wnd, err := app2.AddWindow(win, meui.WindowOptions{
		Title: "meui · ops",
		Size:  graphics.NewSize(float32(*width), float32(*height)),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ops: add window: %v\n", err)
		os.Exit(1)
	}
	app.wnd = wnd
	wireWindow(app)
	if err := app2.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ops: run: %v\n", err)
		os.Exit(1)
	}
}

// wire builds the model and binds every control. live is false for a headless shot: the
// queued run's completion is a timer, and a shot must not race it.
func wire(scene *web.Scene, live bool) *console {
	c := &console{scene: scene, live: live, all: seedRuns()}
	get := func(id string) meui.Widget {
		w := scene.Get(id)
		if w == nil {
			panic("ops: missing id " + id)
		}
		return w
	}
	c.tree = get("serviceTree").(*widgets.Tree)
	c.table = get("runsTable").(*widgets.DataTable)
	c.statusBar = get("statusBar").(*widgets.StatusBar)
	c.menuBar = get("menuBar").(*widgets.MenuBar)
	c.snack = get("snackbar").(*widgets.Snackbar)
	c.inspService = get("inspService").(*widgets.Label)
	c.inspStatus = get("inspStatus").(*widgets.Label)
	c.inspDuration = get("inspDuration").(*widgets.Label)
	c.inspStarted = get("inspStarted").(*widgets.Label)
	c.inspOwner = get("inspOwner").(*widgets.Label)
	c.count = get("tableCount").(*widgets.Label)
	for _, id := range []string{"log0", "log1", "log2"} {
		c.logs = append(c.logs, get(id).(*widgets.Label))
	}

	c.buildTree()
	c.buildMenu()
	c.table.SetColumns([]widgets.DataColumn{
		{Label: "Service", Type: widgets.ColText, Width: 190},
		{Label: "Status", Type: widgets.ColText, Width: 100},
		{Label: "Duration", Type: widgets.ColText, Width: 100},
		{Label: "Started", Type: widgets.ColText, Width: 150},
		{Label: "Owner", Type: widgets.ColText, Width: 150},
	})
	c.table.OnCellSelected = func(row, col int) { c.inspect(row) }
	c.rebuild()
	c.inspect(-1)
	// Native controls are wired through their own setter: web.OnClick is for Boxes that
	// opted into `clickable`, and a Button already owns its input.
	get("newRunButton").(*widgets.Button).SetOnClicked(func() { c.newRun() })
	get("refreshButton").(*widgets.Button).SetOnClicked(func() {
		c.snack.Show("Runs refreshed from the last 24 hours.")
		c.log("refreshed runs (last 24h)")
	})
	c.updateStatus()
	c.log("console started · 16 runs loaded")
	return c
}

// wireWindow binds the actions that need a window (Close). It runs after AddWindow, so a
// shot never gets here.
func wireWindow(c *console) {
	if c.wnd == nil {
		return
	}
	c.menuBar.SetMenuItemAction("File", "Quit", func() { c.wnd.Close() })
}

func (c *console) buildTree() {
	root := &widgets.TreeNode{Label: "WORKSPACE", Expanded: true}
	envs := []struct {
		name     string
		services []string
	}{
		{"production", []string{"api-gateway", "billing", "search", "worker"}},
		{"staging", []string{"api-gateway", "billing", "search", "worker"}},
	}
	for _, e := range envs {
		env := &widgets.TreeNode{Label: e.name, Expanded: true}
		for _, s := range e.services {
			env.Children = append(env.Children, &widgets.TreeNode{Label: s, Expanded: true})
		}
		root.Children = append(root.Children, env)
	}
	c.tree.SetRoot(root)
	c.tree.SetOnNodeSelected(func(index int, node *widgets.TreeNode) {
		// A leaf is a service; a node with children is an environment.
		if len(node.Children) > 0 {
			c.filter = node.Label
		} else {
			c.filter = node.Label
		}
		c.rebuild()
		c.snack.Show("Filtered to " + node.Label + ".")
		c.log("filter: " + node.Label)
	})
}

func (c *console) buildMenu() {
	file := []widgets.MenuItem{
		{Label: "New run", Shortcut: "Ctrl+N", Action: func() { c.newRun() }},
		{IsSep: true},
		{Label: "Refresh", Shortcut: "F5", Action: func() { c.rebuild() }},
		{Label: "Quit", Shortcut: "Ctrl+Q"},
	}
	view := []widgets.MenuItem{
		{Label: "Clear filter", Action: func() {
			c.filter = ""
			c.rebuild()
			c.snack.Show("Filter cleared.")
		}},
	}
	help := []widgets.MenuItem{
		{Label: "About the ops console", Action: func() {
			c.snack.Show("A demo console: tree filters the table, the table fills the inspector.")
		}},
	}
	c.menuBar.SetMenus([]widgets.Menu{
		{Title: "File", Items: file},
		{Title: "View", Items: view},
		{Title: "Help", Items: help},
	})
}

// rebuild projects the model into the table under the current filter and refreshes the
// count. One place builds rows, so a filtered view and a fresh run cannot disagree.
func (c *console) rebuild() {
	rows := make([]widgets.DataRow, 0, len(c.all))
	for _, r := range c.all {
		if c.filter != "" && r.service != c.filter && r.env != c.filter {
			continue
		}
		rows = append(rows, widgets.DataRow{Cells: []widgets.DataCell{
			{Text: r.service},
			{Text: r.status, Color: statusInk[r.status]},
			{Text: r.duration},
			{Text: r.started},
			{Text: r.owner},
		}})
	}
	c.table.SetRows(rows)
	if c.count != nil {
		c.count.SetText(fmt.Sprintf("%d of %d runs", len(rows), len(c.all)))
	}
}

// inspect fills the detail pane from the table's OWN row, never from an index into the
// model: the table may be sorted, and a second lookup is how a detail pane drifts from
// the row the user clicked.
func (c *console) inspect(row int) {
	if row < 0 || row >= len(c.table.Rows) {
		c.setInspector("—", "—", "—", "—", "—")
		return
	}
	cell := func(i int) string {
		if dc := c.table.Cell(row, i); dc != nil {
			return dc.Text
		}
		return ""
	}
	c.setInspector(cell(0), cell(1), cell(2), cell(3), cell(4))
}

func (c *console) setInspector(service, status, duration, started, owner string) {
	c.inspService.SetText(service)
	c.inspStatus.SetText(status)
	c.inspDuration.SetText(duration)
	c.inspStarted.SetText(started)
	c.inspOwner.SetText(owner)
	if ink, ok := statusInk[status]; ok {
		c.inspStatus.SetColor(decodeInk(ink))
	} else {
		c.inspStatus.SetColor(graphics.Color(0xFF111827))
	}
}

// decodeInk turns "#RRGGBB" into the toolkit's ARGB colour.
func decodeInk(hex string) graphics.Color {
	if len(hex) == 7 && hex[0] == '#' {
		hex = "#FF" + hex[1:]
	}
	return graphics.Color(parseHex(hex))
}

func parseHex(s string) uint32 {
	var v uint32
	for _, r := range s {
		if r == '#' {
			continue
		}
		v <<= 4
		switch {
		case r >= '0' && r <= '9':
			v |= uint32(r - '0')
		case r >= 'a' && r <= 'f':
			v |= uint32(r-'a') + 10
		case r >= 'A' && r <= 'F':
			v |= uint32(r-'A') + 10
		}
	}
	return v
}

// newRun queues a run, then (when live) completes it a moment later on the UI thread.
func (c *console) newRun() {
	service := c.filter
	if service == "" {
		service = "api-gateway"
	}
	run := runRow{env: "production", service: service, status: "queued", duration: "—", started: "just now", owner: "you"}
	c.all = append([]runRow{run}, c.all...)
	c.rebuild()
	c.table.SelectedRow = 0
	c.inspect(0)
	c.log("run queued: " + service)
	c.snack.Show("Run queued — it will finish in a moment.")
	if !c.live || c.wnd == nil {
		return
	}
	time.AfterFunc(900*time.Millisecond, func() {
		c.wnd.PostTask(func() { c.finishRun(0, service) })
	})
	c.updateStatus()
}

// finishRun is the timer's body, named so a test can complete a run without waiting. idx is
// the run's position in the model (new runs are prepended).
func (c *console) finishRun(idx int, service string) {
	if idx < 0 || idx >= len(c.all) {
		return
	}
	c.all[idx].status = "ok"
	c.all[idx].duration = "1.10s"
	c.rebuild()
	c.inspect(idx)
	c.log("run finished: " + service + " · ok in 1.10s")
	c.snack.Show("Run finished: " + service + ".")
	c.updateStatus()
}

// log shifts the three activity lines and writes the newest at the bottom.
func (c *console) log(line string) {
	if len(c.logs) == 0 {
		return
	}
	for i := 0; i < len(c.logs)-1; i++ {
		c.logs[i].SetText(c.logs[i+1].Text)
	}
	c.logs[len(c.logs)-1].SetText("09:41  " + line)
}

func (c *console) updateStatus() {
	ok, warn, fail := 0, 0, 0
	for _, r := range c.all {
		switch r.status {
		case "ok":
			ok++
		case "warn":
			warn++
		case "fail":
			fail++
		}
	}
	// Sections replace the bar's single text, so the connection is a section of its own.
	c.statusBar.SetSections([]string{
		"connected · eu-west-1",
		fmt.Sprintf("%d ok", ok),
		fmt.Sprintf("%d warnings", warn),
		fmt.Sprintf("%d failures", fail),
		"v1.4.2",
	})
}
