// Package demoapp is the real-device demo: one scene, wired to the platform, that
// exercises enough of the toolkit and enough of the OS surface to be worth putting
// on a phone.
//
// It exists because the probes answer one question each. A device run has to answer
// several at once — does it render, is it touchable, do the capabilities work, is it
// smooth — and the fastest way to find out is an app that shows all of it and reports
// what the DEVICE says rather than what the demo assumes.
//
// The wiring is deliberately in Go rather than in QML: the Device page reads the
// platform API directly, so a capability that silently reports "unavailable" is
// visible on screen instead of being inferred from a log.
package demoapp

import (
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/layout"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/qml"
	"github.com/sixnineone/meui/theme"
	"github.com/sixnineone/meui/widgets"
)

//go:embed demo.qml
var Scene string

// boundSetter is the one method the wiring needs from a widget. It is an interface
// rather than a concrete type so a widget that cannot be bound fails loudly at the
// lookup instead of silently doing nothing.
type boundSetter interface {
	SetBoundProperty(name string, value interface{})
}

// Demo is a built demo: the root window plus the handles needed to drive it, so a
// host can select a page without reaching into the scene itself.
type Demo struct {
	Root *meui.Window
	reg  *qml.BoundWidgetRegistry
}

// ShowPage selects a page by index (0 Widgets, 1 Device, 2 Render, 3 Harness, 4 Agent,
// 5 Perf). It is a
// no-op when the scene has no pages, so a capture harness cannot fail on it.
func (d *Demo) ShowPage(i int) {
	if d == nil || d.reg == nil {
		return
	}
	if tp, ok := d.reg.Get("demoPages").(*widgets.TabPanel); ok {
		tp.SetActiveTab(i)
	}
}

// New parses the demo scene and returns it, WITHOUT wiring.
//
// Parsing and wiring are separate calls because the order is forced: a window has to
// be added to the app before anything can be laid out, and the QML root IS that
// window — so the caller must parse first, add the parsed root, and only then wire.
// The single-call version had it backwards and added an empty window instead, which
// showed nothing on screen while a direct RenderToImage of the parsed root looked
// perfect. That is a failure mode worth naming: the capture was right and the window
// was wrong.
func New() (*Demo, error) {
	// Follow the platform. Without this the demo runs on the hardcoded meui dark
	// theme, so the mobile presets — the whole point of Phase A — are never selected
	// and a phone keeps 26px controls. It is the same failure as the touch floor: a
	// capability that exists and that nothing switches on.
	if err := theme.LoadAutoPreset(); err != nil {
		return nil, fmt.Errorf("theme: %w", err)
	}

	res, err := qml.ParseWithBindings(Scene)
	if err != nil {
		return nil, fmt.Errorf("demo scene: %w", err)
	}
	root, ok := res.Widget.(*meui.Window)
	if !ok {
		return nil, fmt.Errorf("demo scene root is %T, want *meui.Window", res.Widget)
	}
	reg := res.Widgets
	if reg == nil {
		return nil, fmt.Errorf("demo scene has no widget registry, so nothing can be wired")
	}
	return &Demo{Root: root, reg: reg}, nil
}

// Wire connects the parsed demo to app and to wnd, which must be the platform window
// the demo's root was added as.
func (d *Demo) Wire(app *meui.App, wnd platform.Window) {
	root, reg := d.Root, d.reg

	set := func(id, text string) {
		w := reg.Get(id)
		if w == nil {
			return
		}
		if b, ok := w.(boundSetter); ok {
			b.SetBoundProperty("text", text)
		}
	}
	result := func(format string, args ...interface{}) {
		set("deviceResult", fmt.Sprintf(format, args...))
	}

	// ── Device page: what the platform reports right now ────────────────────
	refresh := func() {
		set("deviceState", describe(app, wnd))
	}
	refresh()

	onClick := func(id string, fn func()) {
		w := reg.Get(id)
		if w == nil {
			return
		}
		if b, ok := w.(*widgets.Button); ok {
			b.SetOnClicked(fn)
		}
	}

	onClick("btnRefresh", refresh)

	onClick("btnPerm", func() {
		perms, ok := app.Backend().Permissions()
		if !ok {
			result("permissions: NOT SUPPORTED on this platform")
			return
		}
		// Ask for the one a phone can actually grant, then report the answer AND
		// the status afterwards — a request that returns "granted" while the status
		// stays "denied" is exactly the kind of thing a device run is for.
		st := perms.Request(platform.PermissionCamera)
		result("permissions: request(camera) -> %v\nstatus now: %v", st, perms.Status(platform.PermissionCamera))
		refresh()
	})

	onClick("btnNotify", func() {
		n, ok := app.Backend().Notifications()
		if !ok {
			result("notifications: NOT SUPPORTED")
			return
		}
		if err := n.Post(platform.Notification{Title: "MeUi", Body: "notification from the demo"}); err != nil {
			result("notifications: Notify failed: %v", err)
			return
		}
		result("notifications: posted (look at the shade)")
	})

	onClick("btnShare", func() {
		s, ok := app.Backend().Share()
		if !ok {
			result("share: NOT SUPPORTED")
			return
		}
		if err := s.Text("MeUi demo share", "https://example.invalid"); err != nil {
			result("share: failed: %v", err)
			return
		}
		result("share: handed to the OS (a chooser should have appeared)")
	})

	onClick("btnLocate", func() {
		loc, ok := app.Backend().Location()
		if !ok {
			result("location: NOT SUPPORTED")
			return
		}
		if !loc.Enabled() {
			result("location: disabled in system settings")
			return
		}
		// Last() is the last known fix, which does not prompt; a device with no
		// fix yet legitimately returns not-ok, and saying so is more useful than a
		// zero coordinate.
		p, ok := loc.Last()
		if !ok {
			result("location: no fix yet (last-known is empty)")
			return
		}
		result("location: %.5f, %.5f (±%.0fm)", p.Latitude, p.Longitude, p.AccuracyMeters)
	})

	onClick("btnHaptic", func() {
		h, ok := app.Backend().Haptics()
		if !ok {
			result("haptics: NOT SUPPORTED")
			return
		}
		_ = h.Play(platform.HapticTap)
		result("haptics: played HapticTap (supported=%v) — did you feel it?", h.Supported())
	})

	onClick("btnBiometric", func() {
		b, ok := app.Backend().Biometrics()
		if !ok {
			result("biometrics: NOT SUPPORTED")
			return
		}
		// Authenticate BLOCKS until the OS answers, so report before and after.
		result("biometrics: available=%v — prompting…", b.Available())
		ok2 := b.Authenticate("Confirm it is you")
		result("biometrics: authenticate -> %v", ok2)
	})

	onClick("btnDialog", func() {
		// The dialog is modal and pumps the loop, so the result is reported after
		// it closes rather than before.
		app.PostTask(func() {
			d := &platform.OpenDialog{}
			d.SetTitle("MeUi demo")
			d.SetCanChooseFiles(true)
			if !d.RunModal() {
				result("dialog: cancelled, or this platform has no dialog backend")
				return
			}
			result("dialog: chose %v", d.Refs())
		})
	})

	// The bottom sheet: the phone's surface for a dialog, a menu or a picker. It is shown
	// through the window's MODAL layer, so the window's scrim sits behind it — and the sheet
	// dismisses itself on a tap above the panel or a drag down, which is why it keeps the
	// window it was hosted by.
	onClick("btnSheet", func() {
		d.ShowSheet(func() { result("bottom sheet dismissed") })
	})

	onClick("btnActions", func() {
		d.ShowActionSheet(func(id string) { result("action sheet chose %q", id) })
	})

	onClick("btnAlert", func() {
		d.ShowDialog(func(id string) { result("alert dialog chose %q", id) })
	})

	// ── The Agent page: the widget set a harness front end needs ────────────
	//
	// Real content, because "the widget exists" and "the widget works" are different
	// questions: an empty MarkdownView and a collapsed GroupBox both look correct in a
	// screenshot and tell you nothing.
	if mv, ok := reg.Get("agentAnswer").(*widgets.MarkdownView); ok {
		mv.SetText(agentAnswerMarkdown)
		// The LIGHT palette on purpose: this is the harness's document pane, which is a
		// light surface whatever the app around it does, and it is the only way to see
		// that the renderer's inks are the theme's now.
		mv.SetLightPalette()
	}
	if tool, ok := reg.Get("agentTool").(*widgets.GroupBox); ok {
		body := newReadOnlyBody("make check\n\tok: no runtime/cgo in the cgo-free desktop graph\n\tok: every cgo file is behind an approved build tag")
		tool.AddChild(body)
	}
	if think, ok := reg.Get("agentThink").(*widgets.GroupBox); ok {
		body := newReadOnlyBody("The suite is flaky, so before touching it: read the assertion. A loop\nwith a shared slice and no lock is a real defect, not a test problem.")
		think.AddChild(body)
	}
	if composer, ok := reg.Get("agentComposer").(*widgets.TextArea); ok {
		composer.SetPlaceholder("Ask this agent. Enter sends, Shift+Enter for a newline.")
	}

	// ── The three-pane harness page ───────────────────────────────────────────
	//
	// Built in Go rather than QML for the same reason the transcript is: a pane that has
	// to FILL its share of a row needs explicit sizes, and Layout.fillWidth leaves it at
	// its default. The shape is the reference's — roster | trajectory | document — and the
	// widths are the ones measured in LOOK-REFERENCE.md.
	if doc, ok := reg.Get("harnessDoc").(*widgets.MarkdownView); ok {
		doc.SetText(harnessDocumentMarkdown)
		doc.SetLightPalette()
	}
	if strip, ok := reg.Get("harnessDocs").(*widgets.TabStrip); ok {
		strip.SetItems([]widgets.TabStripItem{
			{ID: "runtime", Label: "01-core-agent-runtime"},
			{ID: "tools", Label: "02-tools-and-delegation"},
			{ID: "matrix", Label: "MATRIX.md"},
		})
		strip.SetSelected(2)
	}
	if composer, ok := reg.Get("harnessComposer").(*widgets.TextArea); ok {
		composer.SetPlaceholder("Message or run a task, /commands, @ files or sessions")
	}
	// Explicit pane widths, in Go, for the same reason everything else in this demo is:
	// the layout lays children out from their explicit sizes, so a pane that has to hold a
	// share of a row is given one. The roster is the reference's own proportion — a column
	// of names, not a fraction of the window, because a fraction grows with the window and
	// takes a third of a large display for a short list.
	if p, ok := reg.Get("harnessRoster").(*meui.Panel); ok {
		p.SetLayoutProps(&layout.LayoutProps{
			FillWidth: false, FillHeight: true,
			PreferredWidth: harnessRosterWidth, MinimumWidth: harnessRosterWidth,
		})
		p.SetWidth(harnessRosterWidth)
	}
	if p, ok := reg.Get("harnessTrajectory").(*meui.Panel); ok {
		p.SetLayoutProps(&layout.LayoutProps{
			FillWidth: true, FillHeight: true,
			PreferredWidth: harnessTrajectoryWidth, MinimumWidth: harnessTrajectoryWidth,
		})
		p.SetWidth(harnessTrajectoryWidth)
	}
	if host, ok := reg.Get("harnessSteps").(*meui.Panel); ok {
		v := layout.NewVStack()
		v.Spacing = 0
		host.SetLayout(v)
		for _, st := range agentSteps {
			row := widgets.NewCollapsible()
			row.SetLabel(st.kind)
			row.SetDetail(st.summary)
			row.SetTrailing(st.trailing)
			row.SetAccent(st.accent)
			row.SetCollapsed(st.collapsed)
			body := newReadOnlyBody(st.body)
			row.SetBody(body)
			row.SetLayoutProps(&layout.LayoutProps{
				FillWidth: true, FillHeight: false,
				PreferredWidth: 420, MinimumWidth: 420,
			})
			row.SetWidth(420)
			host.AddChild(row)
		}
	}

	// ── The transcript: one Collapsible per step, in the harness's shape ──────
	//
	// The shapes come from the harness's own session log rather than from a screenshot:
	// botui's renderEntry produces "· name firstLine(args)" for a tool call, "✓ …" for a
	// result and the message text for an answer, and a multi-line result is already
	// collapsed to its first line by the model. So a step is a KIND, a one-line SUMMARY and
	// a body — which is exactly what Collapsible's label/detail/trailing are.
	if host, ok := reg.Get("agentSteps").(*meui.Panel); ok {
		v := layout.NewVStack()
		v.Spacing = 0
		host.SetLayout(v)
		for _, st := range agentSteps {
			row := widgets.NewCollapsible()
			row.SetLabel(st.kind)
			row.SetDetail(st.summary)
			row.SetTrailing(st.trailing)
			row.SetAccent(st.accent)
			row.SetCollapsed(st.collapsed)
			body := widgets.NewTextArea()
			body.SetReadOnly(true)
			body.SetWordWrap(true)
			body.SetShowLineNumbers(false)
			body.SetText(st.body)
			// An EXPLICIT body height, and this is the toolkit's model rather than an
			// omission: the layout lays children out from their explicit sizes and never
			// consults PreferredSize, so a body that does not declare one is zero tall and
			// the row collapses to its header. The same reason the QML bodies carry
			// heights.
			body.SetLayoutProps(&layout.LayoutProps{
				FillWidth: true, FillHeight: false, PreferredHeight: 70, MinimumHeight: 70,
			})
			body.SetHeight(70)
			row.SetBody(body)
			row.SetLayoutProps(&layout.LayoutProps{
				FillWidth: true, FillHeight: false,
				PreferredWidth: 250, MinimumWidth: 250,
			})
			row.SetWidth(250)
			host.AddChild(row)
		}
	}

	// ── The document pane: one strip over the same renderer ───────────────────
	//
	// Two tabs, because the point is that the SAME MarkdownView serves both the
	// trajectory's answers and a document, and a document is just a different text. The
	// strip's selection and the answer's scroll position are independent, which is what
	// makes it a pane rather than a view.
	if strip, ok := reg.Get("agentDocs").(*widgets.TabStrip); ok {
		strip.SetItems([]widgets.TabStripItem{
			{ID: "readme", Label: "README.md"},
			{ID: "matrix", Label: "MATRIX.md"},
		})
		strip.SetSelected(0)
	}

	// ── Streaming: feed the answer in and watch the viewport follow ───────────
	//
	// The chunks are whole clauses rather than characters, so what you see is the
	// BLOCK-BOUNDARY path — the one that parses only the tail. A character at a time takes
	// the fallback, because a half-written paragraph is not a document on its own, and that
	// is the honest cost of Markdown rather than of this implementation.
	stopStream := false
	if btn, ok := reg.Get("agentStreamBtn").(*widgets.Button); ok {
		if stop, ok2 := reg.Get("agentStopBtn").(*widgets.Button); ok2 {
			stop.SetOnClicked(func() { stopStream = true })
		}
		btn.SetOnClicked(func() {
			mv, isMV := reg.Get("agentAnswer").(*widgets.MarkdownView)
			if !isMV {
				return
			}
			stopStream = false
			stopBtn, _ := reg.Get("agentStopBtn").(*widgets.Button)
			if stopBtn != nil {
				stopBtn.SetEnabled(true)
			}
			mv.SetText("")
			mv.SetStreaming(true)

			chunks := strings.Split(agentAnswerMarkdown, ". ")
			var n int
			var feed func()
			feed = func() {
				if stopStream || n >= len(chunks) {
					if stopBtn != nil {
						stopBtn.SetEnabled(false)
					}
					// Leave streaming on: the answer is finished but the user may
					// scroll up, and a finished document must not drag them back.
					return
				}
				if n > 0 {
					mv.Append(". ")
				}
				mv.Append(chunks[n])
				n++
				platform.InvokeTaskAfter(feed, 220*time.Millisecond)
			}
			feed()
		})
	}

	// ── System bars: draw behind them, and style their icons ────────────────
	//
	// Edge-to-edge is not a preference on Android 15 — it is enforced — and asking for it
	// is what stops the app being letterboxed between two opaque strips. The content then
	// has to keep itself out of the insets, which the toolkit already does.
	//
	// The appearance follows the THEME's mode, not the platform's: the icons sit on the
	// app's own background, so the app is the one that knows whether they need to be light
	// or dark.
	if sb, ok := app.Backend().SystemBars(); ok {
		sb.SetEdgeToEdge(true)
		if theme.DefaultTheme.Meta.Mode == theme.ModeLight {
			sb.SetAppearance(platform.BarAppearanceLight)
		} else {
			sb.SetAppearance(platform.BarAppearanceDark)
		}
	}

	// ── The shell: one set of destinations, the control chosen by form factor ──
	//
	// A phone gets a bottom NavigationBar and no top tab strip; anything larger keeps
	// the tab strip. The pages are the SAME four either way, which is the point of the
	// axis: the app holds one list, and the form factor decides how it is presented
	// rather than which app it is.
	// Which of the three presentations the destinations get is the FORM FACTOR's
	// decision, not the platform's: a phone wants a bottom bar, a medium surface wants a
	// rail beside the content, and a desktop window keeps the tab strip it already had.
	ff := platform.FormFactorDesktop
	if wnd != nil {
		ff = wnd.FormFactor()
	}
	phone := ff == platform.FormFactorPhone
	rail := ff == platform.FormFactorTablet
	if tp, ok := reg.Get("demoPages").(*widgets.TabPanel); ok {
		tp.SetTabsVisible(ff == platform.FormFactorDesktop)
	}
	// ONE list of destinations, three presentations. This is the whole point of the
	// form-factor axis: the app says what the destinations ARE, and the surface decides
	// how they are shown.
	destinations := []widgets.NavigationBarItem{
		{ID: "widgets", Label: "Widgets", Icon: "document"},
		{ID: "device", Label: "Device", Icon: "folder"},
		{ID: "render", Label: "Render", Icon: "search"},
		{ID: "harness", Label: "Harness", Icon: "folder"},
		{ID: "agent", Label: "Agent", Icon: "user"},
		{ID: "perf", Label: "Perf", Icon: "user"},
	}

	// The app bar is the top of the phone shell, so the desktop status bar steps aside
	// rather than both drawing a strip of chrome.
	if ab, ok := reg.Get("demoAppBar").(*widgets.TopAppBar); ok {
		ab.SetVisible(phone)
		ab.SetLeading("back", "Back")
		ab.SetActions([]widgets.TopAppBarAction{
			{ID: "refresh", Icon: "search", Tooltip: "Refresh"},
			{ID: "share", Icon: "user", Tooltip: "Share"},
		})
		ab.OnLeading = func() {
			if root != nil {
				root.HandleEvent(&platform.Event{Type: platform.EventBack})
			}
		}
		ab.OnAction = func(id string) {
			// Search is the app bar's own destination, not a message: the plan's "full-screen
			// search, opened from the app bar".
			if id == "search" {
				d.ShowSearch(
					func(chosen string) { result("search chose %q", chosen) },
					func() { result("search closed") })
				return
			}
			// A message is the right answer to an action with no visible effect: it
			// confirms without a dialog and gets out of the way.
			if sn, ok := reg.Get("demoSnack").(*widgets.Snackbar); ok {
				sn.ShowAction("app bar action: "+id, "Undo", func() {
					result("app bar action undone: %s", id)
				})
				root.MarkForRedraw()
			}
			result("app bar action: %s", id)
		}
		ab.OnOverflow = func(x, y float32) {
			result("app bar overflow at (%.0f,%.0f)", x, y)
		}
	}
	// The status bar is desktop chrome, so it steps aside on a phone. It is hidden, and
	// the LAYOUT now skips an invisible child — before that it still occupied its 22px
	// and pushed the whole shell down.
	if sb, ok := reg.Get("demoStatus").(*widgets.StatusBar); ok {
		sb.SetVisible(!phone)
	}
	// The rail and the bar are the same destinations; only one is ever built visible.
	if nr, ok := reg.Get("demoRail").(*widgets.NavigationRail); ok {
		nr.SetItems(destinations)
		nr.SetVisible(rail)
		nr.OnSelectionChanged = func(i int) {
			if tp, ok := reg.Get("demoPages").(*widgets.TabPanel); ok {
				tp.SetActiveTab(i)
			}
			setAppBarTitle(reg, i)
			root.MarkForRedraw()
		}
	}
	if nb, ok := reg.Get("demoNav").(*widgets.NavigationBar); ok {
		nb.SetItems(destinations)
		nb.SetVisible(phone)
		nb.OnSelectionChanged = func(i int) {
			if tp, ok := reg.Get("demoPages").(*widgets.TabPanel); ok {
				tp.SetActiveTab(i)
			}
			setAppBarTitle(reg, i)
			root.MarkForRedraw()
		}
		setAppBarTitle(reg, 0)
		// And the panel reports back, so the bar follows a tab-strip click and the two
		// controls can never disagree about the current page.
		if tp, ok := reg.Get("demoPages").(*widgets.TabPanel); ok {
			tp.OnTabChanged = func(i int) {
				nb.SetSelected(i)
				if nr, ok := reg.Get("demoRail").(*widgets.NavigationRail); ok {
					nr.SetSelected(i)
				}
				setAppBarTitle(reg, i)
			}
		}
	}

	// ── The on-screen keyboard, in the shell ────────────────────────────────
	//
	// The spacer below the navigation bar takes the IME's height out of the content, so the
	// bar stays visible instead of being covered — and takes nothing at all while the
	// keyboard is down, because the layout skips an invisible child.
	//
	// It is POLLED from the shell's own 50ms tick rather than driven from SetOnInsets,
	// because the platform window has ONE inset hook and the HOST already uses it for its
	// log. A shell that quietly replaced it would take the host's reporting away; the fix
	// for that is a listener list on the seam, not a silent grab here.
	spacer, _ := reg.Get("keyboardSpacer").(*widgets.InsetSpacer)
	applyKeyboardInset := func() {
		if spacer == nil || wnd == nil {
			return
		}
		kb := wnd.Insets().Keyboard
		if spacer.Inset() != kb {
			spacer.SetInset(kb)
			root.MarkForRedraw()
		}
	}
	applyKeyboardInset()

	// ── The waveform ────────────────────────────────────────────────────────
	//
	// APP DATA: a peak envelope is computed from a track (meui-media/audio's Peaks), and the widget
	// takes plain slices so the toolkit never depends on the media module.
	if wf, ok := reg.Get("demoWaveform").(*widgets.Waveform); ok {
		mins, maxs := synthEnvelope(600)
		wf.SetPeaks(mins, maxs)
		wf.SetProgress(0.35)
		wf.OnSeek = func(f float32) {
			wf.SetProgress(f)
			result("waveform seek %.2f", f)
		}
	}

	// ── The adaptive table ──────────────────────────────────────────────────
	//
	// One widget, two presentations: the DataTable when the surface has room for columns, a list of
	// rows on a phone. The app supplies the data once and wires one selection callback.
	if at, ok := reg.Get("adaptiveTable").(*widgets.AdaptiveTable); ok {
		at.SetColumns([]widgets.DataColumn{
			{Label: "Symbol"},
			{Label: "Qty"},
			{Label: "Price"},
		})
		at.SetRows([]widgets.DataRow{
			{Cells: []widgets.DataCell{{Text: "AAPL"}, {Text: "10"}, {Text: "182.40"}}},
			{Cells: []widgets.DataCell{{Text: "MSFT"}, {Text: "5"}, {Text: "411.10"}}},
			{Cells: []widgets.DataCell{{Text: "NVDA"}, {Text: "2"}, {Text: "121.05"}}},
		})
		at.OnRowSelected = func(row int) { result("table row %d", row) }
	}

	// ── Pull to refresh ─────────────────────────────────────────────────────
	//
	// The wrapper reports the GESTURE; the app does the work and says when it is done, which is what
	// keeps the indicator up for exactly as long as the request takes rather than for a guessed
	// interval.
	if pr, ok := reg.Get("pullRefresh").(*widgets.PullToRefresh); ok {
		pr.OnRefresh = func() {
			result("pull to refresh: refreshing")
			platform.InvokeTaskAfter(func() {
				pr.Done()
				result("pull to refresh: done")
			}, 900*time.Millisecond)
		}
	}

	// ── The swipe row's actions ─────────────────────────────────────────────
	//
	// APP DATA, not markup: the actions behind a row are what the app can DO to it, and the QML
	// carries only the row itself.
	if sr, ok := reg.Get("swipeRow").(*widgets.SwipeActionRow); ok {
		sr.SetActions([]widgets.SwipeAction{
			{ID: "delete", Label: "Delete", Destructive: true},
			{ID: "archive", Label: "Archive"},
		})
		sr.OnAction = func(id string) { result("swipe row: %s", id) }
	}

	// ── Back navigation ─────────────────────────────────────────────────────
	//
	// The demo shows what the app is supposed to do with the gesture: step back one
	// page, and decline at the first page so the window closes — which is the platform
	// contract. On Android this is the difference between the back gesture popping the
	// app's own navigation and the system closing it outright.
	root.SetOnBack(func() bool {
		tp, ok := reg.Get("demoPages").(*widgets.TabPanel)
		if !ok {
			return false
		}
		cur := tp.ActiveTab()
		if cur <= 0 {
			result("back: at the first page, so the window will close")
			return false
		}
		tp.SetActiveTab(cur - 1)
		result("back: moved from page %d to %d", cur, cur-1)
		root.MarkForRedraw()
		return true
	})

	// ── Animation: the two bars advance every frame-ish tick ────────────────
	//
	// A self-rescheduling task rather than a frame callback, because that is what
	// the platform offers and 50ms is enough to see stutter. It stops when the
	// window is gone so a closed demo does not spin.
	frame := 0
	var tick func()
	tick = func() {
		if root.Window() == nil {
			return
		}
		// The keyboard's inset, before the frame is marked dirty, so the bar is lifted in
		// the same frame the platform reported it.
		applyKeyboardInset()
		frame++
		v := float32(frame % 100)
		for _, id := range []string{"perfBar", "renderBar"} {
			if w := reg.Get(id); w != nil {
				if b, ok := w.(boundSetter); ok {
					b.SetBoundProperty("value", v)
				}
			}
		}
		label := fmt.Sprintf("frame %d", frame)
		set("perfLabel", label)
		set("renderFrame", label)
		platform.InvokeTaskAfter(tick, 50*time.Millisecond)
	}
	platform.InvokeTaskAfter(tick, 50*time.Millisecond)

	// Refresh the live state periodically: connectivity, permission status and the
	// safe area all change underneath a running app.
	var poll func()
	poll = func() {
		if root.Window() == nil {
			return
		}
		refresh()
		platform.InvokeTaskAfter(poll, 2*time.Second)
	}
	platform.InvokeTaskAfter(poll, 2*time.Second)

}

// DumpChrome prints the harness page's chrome AFTER layout, which is the only time the
// answer means anything: before layout every child carries its default frame, and reading
// a frame then is how two wrong conclusions got recorded in this file's history.
func (d *Demo) DumpChrome() {
	if d == nil || d.reg == nil {
		return
	}
	for _, id := range []string{"harnessDocs", "harnessComposer", "harnessDoc"} {
		w := d.reg.Get(id)
		if w == nil {
			fmt.Printf("[chrome] %-18s NOT FOUND\n", id)
			continue
		}
		f := w.FrameRect()
		extra := ""
		if st, ok := w.(*widgets.TabStrip); ok {
			// The inks, because "the bar paints and the labels do not" is either a
			// contrast failure or a draw failure, and the two look identical in a
			// screenshot. This is the measurement that tells them apart.
			// The FONT as well as the inks: "the bar paints and the labels do not"
			// is a draw failure OR a font failure, and the two look identical in a
			// screenshot. A nil or zero-size font draws no glyphs and no error.
			fontState := "nil"
			if f := st.TabTheme.Font; f != nil {
				fontState = fmt.Sprintf("size=%.1f line=%.1f base=%v", f.Size(), f.LineHeight(), f == nil)
			}
			baseState := "nil"
			if f := theme.DefaultTheme.BaseFont; f != nil {
				baseState = fmt.Sprintf("size=%.1f", f.Size())
			}
			extra = fmt.Sprintf(" items=%d textInk=#%08X bgInk=#%08X font=%s base=%s tabH=%.0f",
				st.Count(), uint32(st.TextInk), uint32(st.BackgroundInk), fontState, baseState, st.TabTheme.TabHeight)
		}
		fmt.Printf("[chrome] %-18s %-20T (%3.0f,%3.0f %3.0fx%3.0f)%s\n", id, w, f.X, f.Y, f.Width, f.Height, extra)
	}
}

// Snackbar exposes the demo's message strip, so a harness can show one without
// reaching into the scene.
func (d *Demo) Snackbar() *widgets.Snackbar {
	if d == nil || d.reg == nil {
		return nil
	}
	sn, _ := d.reg.Get("demoSnack").(*widgets.Snackbar)
	return sn
}

// ShowSheet opens the demo's bottom sheet, so a capture harness can photograph the surface
// without synthesizing a click. onDismiss may be nil.
func (d *Demo) ShowSheet(onDismiss func()) *widgets.BottomSheet {
	if d == nil || d.Root == nil {
		return nil
	}
	sheet := widgets.NewBottomSheet()
	sheet.SetTitle("Bottom sheet")
	sheet.SetModal(true)
	sheet.SetSheetHeight(300)
	body := widgets.NewLabel()
	body.SetText("Drag the handle down, or tap outside, to dismiss.")
	body.SetWrapMode(widgets.WrapWordWrap)
	// A sheet's content starts at the TOP of its area: a Label centres vertically by default,
	// which would float a single line into the middle of the sheet.
	body.SetVAlign("top")
	sheet.SetContent(body)
	sheet.OnDismissed = onDismiss
	f := d.Root.FrameRect()
	d.Root.ShowModal(sheet, f.Width, f.Height)
	d.Root.MarkForRedraw()
	return sheet
}

// ShowDialog opens the demo's adaptive dialog. ONE widget: on a phone-sized surface it is a bottom
// sheet, and on anything larger it is centred, chosen by FormFactor — which is the whole point of
// the widget, and why the capture harness renders it at both sizes. onAction may be nil.
func (d *Demo) ShowDialog(onAction func(string)) *widgets.AlertDialog {
	if d == nil || d.Root == nil {
		return nil
	}
	dlg := widgets.NewAlertDialog()
	dlg.SetTitle("Delete file")
	dlg.SetMessage("The file will be removed from this device. This cannot be undone.")
	dlg.SetActions([]widgets.DialogAction{
		{ID: "cancel", Label: "Cancel"},
		{ID: "delete", Label: "Delete", Primary: true, Destructive: true},
	})
	dlg.OnAction = onAction
	f := d.Root.FrameRect()
	d.Root.ShowModal(dlg, f.Width, f.Height)
	d.Root.MarkForRedraw()
	return dlg
}

// PullRefresh drags the demo's page down without releasing, so a capture can photograph the gesture
// mid-pull: the indicator, and the content pushed down by it.
func (d *Demo) PullRefresh() *widgets.PullToRefresh {
	if d == nil || d.reg == nil {
		return nil
	}
	pr, _ := d.reg.Get("pullRefresh").(*widgets.PullToRefresh)
	if pr != nil {
		// The point comes from the WINDOW, not the wrapper: a capture helper runs before the first
		// layout, so the wrapper's own frame is still its constructor default and the indicator
		// would be drawn against a 50px-wide rect at the far left.
		f := d.Root.FrameRect()
		pr.OnMouseDown(f.Width/2, 60, 0)
		pr.OnMouseMove(f.Width/2, 60+widgets.PullThreshold*1.6)
	}
	return pr
}

// synthEnvelope builds a plausible audio envelope: a decaying series of hits over a quiet floor. It
// exists so the demo shows a waveform rather than a flat line, and it is the shape a peak cache
// produces for real audio.
func synthEnvelope(buckets int) (mins, maxs []float32) {
	mins = make([]float32, buckets)
	maxs = make([]float32, buckets)
	for i := 0; i < buckets; i++ {
		// A hit every 64 buckets, decaying, plus a quiet floor.
		phase := float32(i%64) / 64
		amp := float32(0.15)
		if i%64 < 12 {
			amp = 0.9 * (1 - phase)
		}
		// A slow swell so the left half is louder than the right.
		amp *= 1 - 0.5*float32(i)/float32(buckets)
		maxs[i] = amp
		mins[i] = -amp * 0.8
	}
	return mins, maxs
}

// OpenSwipeRow reveals the demo's swipe row's actions, so a capture can photograph what the gesture
// uncovers without synthesizing a drag.
func (d *Demo) OpenSwipeRow() *widgets.SwipeActionRow {
	if d == nil || d.reg == nil {
		return nil
	}
	sr, _ := d.reg.Get("swipeRow").(*widgets.SwipeActionRow)
	if sr != nil {
		sr.Open()
	}
	return sr
}

// ShowSearch opens the demo's full-screen search over the whole window — the app bar's search
// action's destination. onChoose and onClose may be nil.
func (d *Demo) ShowSearch(onChoose func(string), onClose func()) *widgets.SearchView {
	if d == nil || d.Root == nil {
		return nil
	}
	sv := widgets.NewSearchView()
	sv.SetPlaceholder("Search the demo")
	sv.SetResults([]widgets.SearchViewResult{
		{ID: "widgets", Label: "Widgets", Secondary: "the widget set"},
		{ID: "device", Label: "Device", Secondary: "live capability state"},
		{ID: "render", Label: "Render", Secondary: "text, corners, icons"},
		{ID: "perf", Label: "Perf", Secondary: "an animating bar and a long list"},
	})
	sv.OnChoose = onChoose
	sv.OnClose = onClose
	f := d.Root.FrameRect()
	d.Root.ShowModal(sv, f.Width, f.Height)
	d.Root.MarkForRedraw()
	return sv
}

// ShowPicker opens the demo's date picker, so the capture harness can photograph the presentation
// the surface chose — a sheet on a phone, a dropdown on anything larger.
func (d *Demo) ShowPicker() *widgets.DatePicker {
	if d == nil || d.reg == nil {
		return nil
	}
	dp, _ := d.reg.Get("demoDate").(*widgets.DatePicker)
	if dp != nil {
		dp.SetExpanded(true)
	}
	return dp
}

// ShowActionSheet opens the demo's action sheet, the same way and for the same reason as
// ShowSheet: a harness can photograph it without synthesizing a click. onChoose may be nil.
func (d *Demo) ShowActionSheet(onChoose func(string)) *widgets.ActionSheet {
	if d == nil || d.Root == nil {
		return nil
	}
	a := widgets.NewActionSheet()
	a.SetHeader("Photo", "What would you like to do?")
	a.SetItems([]widgets.ActionSheetItem{
		{ID: "save", Label: "Save"},
		{ID: "share", Label: "Share"},
		{ID: "delete", Label: "Delete", Destructive: true},
	})
	a.SetCancel("Cancel")
	a.OnChoose = onChoose
	f := d.Root.FrameRect()
	d.Root.ShowModal(a, f.Width, f.Height)
	d.Root.MarkForRedraw()
	return a
}

// newReadOnlyBody is a collapsible group's content.
//
// The layout props are the point, and they are not optional: a child added to a GroupBox
// arrives at the DEFAULT widget size, and a read-only TextArea at that size clips its own
// text to a couple of characters — which looks like the widget failing to render rather
// than like a missing size. Every transcript body in botui sets them for the same reason.
func newReadOnlyBody(text string) *widgets.TextArea {
	body := widgets.NewTextArea()
	body.SetReadOnly(true)
	body.SetWordWrap(true)
	body.SetShowLineNumbers(false)
	body.SetText(text)
	body.SetLayoutProps(&layout.LayoutProps{FillWidth: true, FillHeight: true})
	return body
}

// agentStep is one transcript row, in the shape botui's model already produces.
type agentStep struct {
	kind, summary, trailing, body string
	collapsed, accent             bool
}

// agentSteps is the fixture: a tool call, its result, a think, and the answer. The
// states are the ones that matter visually — a collapsed tool call (the common case, and
// the one that has to cost nothing), a failed result, an open think, and the current
// destination marked.
var agentSteps = []agentStep{
	{kind: "· bash", summary: "Check the gate and commit", trailing: "45 sub", collapsed: true,
		body: "ok: no runtime/cgo in the cgo-free desktop graph\nok: every cgo file is behind an approved build tag"},
	{kind: "· tool call", summary: "read", trailing: "1.2s", collapsed: true,
		body: "github.com/sixnineone/meui/widgets/collapsible.go"},
	{kind: "· tool result", summary: "✗ the flaky suite failed twice", trailing: "FAIL", collapsed: true,
		body: "FAIL\tshare\nFAIL\tread"},
	{kind: "· think", summary: "The suite is flaky, so read the assertion before touching it.", trailing: "", collapsed: false,
		body: "A loop with a shared slice and no lock is a real defect, not a test problem. Retrying hides it."},
	{kind: "· answer", summary: "All three pass. The flaky suite was a real defect.", trailing: "", collapsed: true, accent: true,
		body: "Fixed rather than retried, and the race detector agrees."},
}

// The three panes' widths. A column of names is a fixed width and the trajectory is the
// pane that needs room, so both are numbers rather than fractions — see the note at the
// call site.
const (
	harnessRosterWidth     = 240
	harnessTrajectoryWidth = 440
)

// harnessDocumentMarkdown is the reference's own document SHAPE: a heading, prose, a table
// with a header, and a blockquote. It is the same fixture the answer uses, which is the
// point — one renderer, two panes.
const harnessDocumentMarkdown = `## The model matrix

Nine of thirteen agents wrote a complete, compiling, tested Go program.

| Model | Result | Cause |
|---|---|---|
| m-space-bunny | PASS | — |
| m-mimo | PASS | — |
| m-zaiglm | PASS | — |
| m-minimax | FAIL | — |
| m-grok | FAIL | — |

> The three invocations do not prove the programs — they prove the gate and commit.`

// agentAnswerMarkdown is the fixture for the Agent page: every block kind the harness
// actually shows in an answer, so a regression in one of them is visible in one look —
// a heading, a paragraph, emphasis, a list, a GFM TABLE, and a fenced code block.
const agentAnswerMarkdown = `## The gate, and what it found

All three passes, and the **flaky** suite was a real defect: a *shared slice* appended
from two goroutines with no lock. Fixed rather than retried.

1. gate and commit
2. ` + "`run the whole suite under -race`" + `
3. record it

| Target | Result | Cause |
|---|---|---|
| meui | PASS | — |
| meui-gl | PASS | — |
| media-editor | FAIL | shared slice, no lock |

` + "```go" + `
if err := run(); err != nil {   // fixed: the append was racing
	log.Fatal(err)
}
` + "```" + `

> The three invocations do not prove the programs — they prove the gate and commit.`

// setAppBarTitle keeps the app bar's title on the current page, so the shell's two
// halves never disagree about where you are.
func setAppBarTitle(reg *qml.BoundWidgetRegistry, page int) {
	ab, ok := reg.Get("demoAppBar").(*widgets.TopAppBar)
	if !ok {
		return
	}
	titles := []string{"Widgets", "Device", "Render", "Harness", "Agent", "Perf"}
	if page >= 0 && page < len(titles) {
		ab.SetTitle(titles[page])
	}
}

// describe is the live-state block: every capability the toolkit can report, with
// an explicit "unsupported" rather than an omitted line, because the absence of a
// line is indistinguishable from a bug in this function.
func describe(app *meui.App, wnd platform.Window) string {
	b := app.Backend()
	var s strings.Builder

	if wnd != nil {
		ins := wnd.Insets()
		fmt.Fprintf(&s, "window   %.0fx%.0f scale %.2f\n",
			float32(wnd.Width()), float32(wnd.Height()), wnd.Scale())
		fmt.Fprintf(&s, "insets   top %.0f right %.0f bottom %.0f left %.0f keyboard %.0f\n",
			ins.Top, ins.Right, ins.Bottom, ins.Left, ins.Keyboard)
	}

	if ds := b.Displays(); len(ds) > 0 {
		d := ds[0]
		fmt.Fprintf(&s, "display  %q scale %.2fx%.2f\n", d.Name, d.ScaleX, d.ScaleY)
	} else {
		s.WriteString("display  (none reported)\n")
	}
	fmt.Fprintf(&s, "theme    %v\n", b.SystemTheme())

	if c, ok := b.Connectivity(); ok {
		online, metered := c.Online()
		fmt.Fprintf(&s, "network  online=%v metered=%v\n", online, metered)
	} else {
		s.WriteString("network  NOT SUPPORTED\n")
	}

	if p, ok := b.Permissions(); ok {
		known := p.Known()
		fmt.Fprintf(&s, "perms    %d known:", len(known))
		for _, k := range known {
			fmt.Fprintf(&s, " %v=%v", k, p.Status(k))
		}
		s.WriteString("\n")
	} else {
		s.WriteString("perms    NOT SUPPORTED\n")
	}

	if sen, ok := b.Sensors(); ok {
		s.WriteString("sensors ")
		for _, k := range []platform.SensorKind{
			platform.SensorAccelerometer, platform.SensorGyroscope, platform.SensorMagnetometer,
		} {
			fmt.Fprintf(&s, " %v=%v", k, sen.Has(k))
		}
		s.WriteString("\n")
	} else {
		s.WriteString("sensors  NOT SUPPORTED\n")
	}

	if l, ok := b.Location(); ok {
		fmt.Fprintf(&s, "location enabled=%v\n", l.Enabled())
	} else {
		s.WriteString("location NOT SUPPORTED\n")
	}
	if bio, ok := b.Biometrics(); ok {
		fmt.Fprintf(&s, "biometry available=%v\n", bio.Available())
	} else {
		s.WriteString("biometry NOT SUPPORTED\n")
	}
	if h, ok := b.Haptics(); ok {
		fmt.Fprintf(&s, "haptics  supported=%v\n", h.Supported())
	} else {
		s.WriteString("haptics  NOT SUPPORTED\n")
	}
	if n, ok := b.Notifications(); ok {
		fmt.Fprintf(&s, "notify   supported=%v\n", n.Supported())
	} else {
		s.WriteString("notify   NOT SUPPORTED\n")
	}
	if p, ok := b.Power(); ok {
		info := p.Info()
		fmt.Fprintf(&s, "power    %v %d%%\n", info.State, info.Percent)
	} else {
		s.WriteString("power    NOT SUPPORTED\n")
	}
	return s.String()
}
