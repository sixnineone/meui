// Team admin — a dark enterprise console: navigation rail, a filterable member table, a
// detail pane, and an invite flow with validation.
//
//	go run ./examples/admin
//	go run ./examples/admin -shot /tmp/admin.png
//	go run ./examples/admin -invite -shot /tmp/admin-invite.png   # the dialog's state
//
// The dialog is an IN-SCENE overlay (a scrim plus a card, both z-ordered) rather than a
// window modal, so it renders in a headless shot and the example can be verified without a
// display. The toolkit's window modals are shown by the imported demo.
package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/web"
	"github.com/sixnineone/meui/widgets"
)

var emailRE = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type member struct {
	name, email, role, status, lastActive string
}

func seedMembers() []member {
	return []member{
		{"Port Aurelia", "port@atlas.dev", "Owner", "Active", "2 minutes ago"},
		{"Mira Chen", "mira@atlas.dev", "Admin", "Active", "1 hour ago"},
		{"Jonas Weir", "jonas@atlas.dev", "Editor", "Active", "3 hours ago"},
		{"Selin Aydin", "selin@atlas.dev", "Editor", "Invited", "—"},
		{"Tomás Reyes", "tomas@atlas.dev", "Editor", "Active", "yesterday"},
		{"Ada Okonkwo", "ada@atlas.dev", "Admin", "Active", "yesterday"},
		{"Ruth Lindqvist", "ruth@atlas.dev", "Viewer", "Active", "2 days ago"},
		{"Ben Osei", "ben@atlas.dev", "Viewer", "Suspended", "3 weeks ago"},
		{"Kai Nakamura", "kai@atlas.dev", "Editor", "Invited", "—"},
	}
}

var memberInk = map[string]string{
	"Active":    "#34D399",
	"Invited":   "#FBBF24",
	"Suspended": "#F87171",
}

type admin struct {
	scene *web.Scene
	all   []member

	table   *widgets.DataTable
	search  *widgets.TextField
	role    *widgets.ComboBox
	rail    *widgets.NavigationRail
	status  *widgets.StatusBar
	snack   *widgets.Snackbar
	title   *widgets.Label
	scrim   *meui.Panel
	dialog  *meui.Panel
	invite  *widgets.TextField
	errMsg  *widgets.Label
	detail  *widgets.Label // "Name — email" line
	detailR *widgets.Label
	detailS *widgets.Label
	billing *widgets.Switch

	filterRole string
}

func main() {
	shot := flag.String("shot", "", "render one frame to this PNG and exit (headless)")
	preset := flag.String("preset", "windows11-dark", "theme preset")
	width := flag.Int("w", 1280, "window width")
	height := flag.Int("h", 800, "window height")
	path := flag.String("qml", "examples/admin/admin.qml", "scene file")
	invite := flag.Bool("invite", false, "open the invite dialog before shooting")
	flag.Parse()

	scene, err := web.Load(*path, *preset, *width, *height)
	if err != nil {
		fmt.Fprintf(os.Stderr, "admin: %v\n", err)
		os.Exit(1)
	}
	app := wire(scene)
	if *invite {
		app.openInvite()
	}
	if *shot != "" {
		if err := scene.Shot(*width, *height, *shot); err != nil {
			fmt.Fprintf(os.Stderr, "admin: shot: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("admin: wrote %s (%dx%d)\n", *shot, *width, *height)
		return
	}

	win, ok := scene.Root.(*meui.Window)
	if !ok {
		fmt.Fprintf(os.Stderr, "admin: root is %T, want *meui.Window\n", scene.Root)
		os.Exit(1)
	}
	application := meui.NewApp(sdl3.NewApp(), meui.AppOptions{Title: "meui · team admin"})
	if _, err := application.AddWindow(win, meui.WindowOptions{
		Title: "meui · team admin",
		Size:  graphics.NewSize(float32(*width), float32(*height)),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "admin: add window: %v\n", err)
		os.Exit(1)
	}
	if err := application.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "admin: run: %v\n", err)
		os.Exit(1)
	}
}

func wire(scene *web.Scene) *admin {
	a := &admin{scene: scene, all: seedMembers()}
	get := func(id string) meui.Widget {
		w := scene.Get(id)
		if w == nil {
			panic("admin: missing id " + id)
		}
		return w
	}
	a.table = get("membersTable").(*widgets.DataTable)
	a.search = get("searchField").(*widgets.TextField)
	a.role = get("roleFilter").(*widgets.ComboBox)
	a.rail = get("rail").(*widgets.NavigationRail)
	a.status = get("statusBar").(*widgets.StatusBar)
	a.snack = get("snackbar").(*widgets.Snackbar)
	a.title = get("pageTitle").(*widgets.Label)
	a.scrim = get("scrim").(*meui.Panel)
	a.dialog = get("inviteDialog").(*meui.Panel)
	a.invite = get("inviteEmail").(*widgets.TextField)
	a.errMsg = get("inviteError").(*widgets.Label)
	a.detail = get("detailName").(*widgets.Label)
	a.detailR = get("detailRole").(*widgets.Label)
	a.detailS = get("detailStatus").(*widgets.Label)
	a.billing = get("billingSwitch").(*widgets.Switch)

	// Labels only: the rail draws an ICON once the name resolves to an SVG, and an
	// unresolved name is drawn as a glyph — "grid" would render as the word. The example
	// does not ship icon assets, so it asks for the label layout.
	// Short labels: the rail's item box is a fixed theme width, and a label wider than it is
	// clipped on BOTH sides (the selected pill is that box). The page title carries the full
	// name.
	a.rail.SetItems([]widgets.NavigationBarItem{
		{Label: "Home"},
		{Label: "Team"},
		{Label: "Billing"},
		{Label: "More"},
	})
	a.rail.SetSelected(1)
	a.rail.OnSelectionChanged = func(index int) {
		a.title.SetText([]string{"Overview", "Members", "Billing", "Settings"}[index])
	}

	a.role.SetOptions([]string{"All roles", "Owner", "Admin", "Editor", "Viewer"})
	a.role.OnSelected = func(index int, value string) {
		if index == 0 {
			a.filterRole = ""
		} else {
			a.filterRole = value
		}
		a.rebuild()
	}
	a.search.OnTextChanged = func(string) { a.rebuild() }

	a.table.SetColumns([]widgets.DataColumn{
		{Label: "Name", Type: widgets.ColText, Width: 180},
		{Label: "Email", Type: widgets.ColText, Width: 220},
		{Label: "Role", Type: widgets.ColText, Width: 100},
		{Label: "Status", Type: widgets.ColText, Width: 110},
		{Label: "Last active", Type: widgets.ColText, Width: 130},
	})
	a.table.OnCellSelected = func(row, col int) { a.inspect(row) }
	a.rebuild()
	a.inspect(-1)

	get("inviteButton").(*widgets.Button).SetOnClicked(func() { a.openInvite() })
	get("cancelInvite").(*widgets.Button).SetOnClicked(func() { a.closeInvite() })
	get("sendInvite").(*widgets.Button).SetOnClicked(func() { a.sendInvite() })
	get("saveButton").(*widgets.Button).SetOnClicked(func() {
		a.snack.Show("Saved the member's permissions.")
	})
	get("removeButton").(*widgets.Button).SetOnClicked(func() { a.removeSelected() })

	a.status.SetSections([]string{"connected", "9 members", "2 invited", "v2.1.0"})
	return a
}

// rebuild projects the filtered member list into the table.
func (a *admin) rebuild() {
	query := strings.ToLower(strings.TrimSpace(a.search.Text))
	rows := make([]widgets.DataRow, 0, len(a.all))
	for _, m := range a.all {
		if a.filterRole != "" && m.role != a.filterRole {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(m.name), query) &&
			!strings.Contains(strings.ToLower(m.email), query) {
			continue
		}
		rows = append(rows, widgets.DataRow{Cells: []widgets.DataCell{
			{Text: m.name},
			{Text: m.email},
			{Text: m.role},
			{Text: m.status, Color: memberInk[m.status]},
			{Text: m.lastActive},
		}})
	}
	a.table.SetRows(rows)
	a.status.SetSections([]string{
		"connected",
		fmt.Sprintf("%d shown", len(rows)),
		fmt.Sprintf("%d members", len(a.all)),
		"v2.1.0",
	})
}

func (a *admin) inspect(row int) {
	if row < 0 || row >= len(a.table.Rows) {
		a.detail.SetText("Select a member.")
		a.detail.SetFontBold(false)
		a.detailR.SetText("")
		a.detailS.SetText("")
		return
	}
	cell := func(i int) string {
		if c := a.table.Cell(row, i); c != nil {
			return c.Text
		}
		return ""
	}
	a.detail.SetText(cell(0))
	a.detail.SetFontBold(true)
	a.detailR.SetText(cell(1) + " · " + cell(2))
	a.detailS.SetText(cell(3))
}

func (a *admin) openInvite() {
	a.scrim.SetVisible(true)
	a.dialog.SetVisible(true)
	a.invite.SetText("")
	a.errMsg.SetVisible(false)
	if win, ok := a.scene.Root.(*meui.Window); ok {
		win.SetFocusedChild(a.invite)
	}
}

func (a *admin) closeInvite() {
	a.scrim.SetVisible(false)
	a.dialog.SetVisible(false)
}

func (a *admin) sendInvite() {
	email := strings.TrimSpace(a.invite.Text)
	if !emailRE.MatchString(email) {
		a.errMsg.SetText("Enter a work email address, like name@company.com.")
		a.errMsg.SetVisible(true)
		return
	}
	for _, m := range a.all {
		if strings.EqualFold(m.email, email) {
			a.errMsg.SetText("That person is already on the team.")
			a.errMsg.SetVisible(true)
			return
		}
	}
	name := email
	if at := strings.IndexByte(email, '@'); at > 0 {
		name = strings.Title(strings.ReplaceAll(email[:at], ".", " ")) //nolint:staticcheck // a demo of the flow, not a name normaliser
	}
	role := "Editor"
	if a.filterRole != "" {
		role = a.filterRole
	}
	a.all = append([]member{{name, email, role, "Invited", "—"}}, a.all...)
	a.closeInvite()
	a.rebuild()
	a.table.SelectedRow = 0
	a.inspect(0)
	a.snack.Show("Invite sent to " + email + ".")
}

func (a *admin) removeSelected() {
	row := a.table.SelectedRow
	if row < 0 || row >= len(a.table.Rows) {
		a.snack.Show("Select a member first.")
		return
	}
	email := a.table.Cell(row, 1).Text
	for i, m := range a.all {
		if m.email == email {
			a.all = append(a.all[:i], a.all[i+1:]...)
			break
		}
	}
	a.rebuild()
	a.inspect(-1)
	a.snack.Show("Removed " + email + " from the team.")
}
