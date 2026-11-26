package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/slant"
	"github.com/sixnineone/meui/graphics/enums/spacing"
	"github.com/sixnineone/meui/graphics/enums/weight"
	"github.com/sixnineone/meui/layout"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/qml"
	"github.com/sixnineone/meui/theme"
	"github.com/sixnineone/meui/widgets"
)

const AppVersion = "v0.24.3 — May 16, 2026"

// buildID is stamped at compile time (-ldflags -X) so the running
// window's title bar proves exactly which build is on screen.
var buildID = "dev"

// Font changer state: every discovered family and its file. Filled by
// ListSystemFonts at startup; the title-bar combo and
// MEUI_FONTSHEET_FONT both resolve through it.
var (
	systemFonts       []graphics.SystemFontInfo
	fontPathsByFamily map[string]string

	// titlebarCombo is the font combo living in the title bar middle
	// slot (drag-area regression test vehicle).
	titlebarCombo *widgets.ComboBox
)

// fileExists is the fallback probe for the scene path.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func main() {
	log.Printf("BUILD: %s", buildID)
	dumpJSON := flag.Bool("dump-json", false, "Dump widget tree as JSON and exit")
	flag.Parse()

	qmlPath := "main.qml"
	// The demo lives in examples/enterprise; run it from there (`go run .`) or from the
	// module root (`go run ./examples/enterprise`), which is how the README asks for it.
	if _, err := os.Stat(qmlPath); err != nil {
		if alt := filepath.Join("examples", "enterprise", "main.qml"); fileExists(alt) {
			qmlPath = alt
		}
	}
	if len(flag.Args()) > 0 {
		qmlPath = flag.Args()[0]
	}

	// resolveThemesDir finds the directory holding *.theme.json files:
	// MEUI_THEMES env wins, then ./themes, then the toolkit's own themes
	// directory (../themes — the demo lives in meui/demo_meui_imported),
	// then the layouts from the older checkouts (../meui/themes,
	// ../../themes).
	resolveThemesDir := func() string {
		if p := os.Getenv("MEUI_THEMES"); p != "" {
			if st, err := os.Stat(p); err == nil && st.IsDir() {
				return p
			}
		}
		for _, cand := range []string{
			"themes",
			filepath.Join("..", "themes"),
			filepath.Join("..", "meui", "themes"),
			filepath.Join("..", "..", "themes"),
		} {
			if st, err := os.Stat(cand); err == nil && st.IsDir() {
				return cand
			}
		}
		return "themes"
	}
	themeDir := resolveThemesDir()

	// Look & feel: MEUI_LAF=<flat|meui|windows11|macos> selects the
	// shape profile (rounded corners, window chrome style).
	laf := os.Getenv("MEUI_LAF")
	if laf == "" {
		laf = "meui"
	}
	if err := theme.SetLookAndFeel(laf); err != nil {
		log.Printf("L&F: %v (using default)", err)
	}
	log.Printf("L&F: %s", theme.ShapeName())

	// Load IntelliJ theme from the themes directory (if available).
	// Seed candidates exist in the shared themes checkout.
	var initialTheme *theme.Theme
	themePath := ""
	for _, seed := range []string{
		"darcula.theme.json",
		"HighContrast.theme.json",
		"one_dark_vivid.theme.json",
	} {
		p := filepath.Join(themeDir, seed)
		if _, err := os.Stat(p); err == nil {
			themePath = p
			break
		}
	}
	if themePath != "" {
		initialTheme, _ = theme.ImportIntelliJThemeJSON(themePath)
	}
	if initialTheme != nil {
		initialTheme.BaseFont = theme.DefaultTheme.BaseFont
		// Base font: prefer Inter Display from ./fonts at a real UI
		// size — the theme fallback chain lands on Roboto at
		// (size-4) = 6px, which renders muddy. A real UI font at a
		// real size is the foundation of text sharpness.
		if ff, err := graphics.LoadFontFace("fonts/InterDisplay-Regular.otf"); err == nil {
			initialTheme.BaseFont = ff.Font(12)
			log.Printf("Base font: Inter Display 12")
		}
		initialTheme.FontSizeSmall = theme.DefaultTheme.FontSizeSmall
		initialTheme.FontSizeMedium = theme.DefaultTheme.FontSizeMedium
		initialTheme.FontSizeLarge = theme.DefaultTheme.FontSizeLarge
		initialTheme.FontSizeTitle = theme.DefaultTheme.FontSizeTitle
		initialTheme.FontWeightNormal = theme.DefaultTheme.FontWeightNormal
		initialTheme.FontWeightBold = theme.DefaultTheme.FontWeightBold
		initialTheme.LineHeight = theme.DefaultTheme.LineHeight
		initialTheme.BorderRadius = theme.DefaultTheme.BorderRadius
		// Single application path, BEFORE any widget is constructed
		// (QML parse below): the very first paint already carries the
		// final theme. Previously the global was swapped after window
		// creation, leaving first frames and caches on the previous
		// theme until the version-invalidation happened to catch up.
		theme.Apply(initialTheme)
		log.Printf("Loaded IntelliJ theme: %s", initialTheme.Name)
		// Editor scheme through the same path as interactive switching —
		// startup previously skipped it, so editor colours differed from
		// the identical theme selected from the list.
		if schemePath := findEditorScheme(themePath, themeDir); schemePath != "" {
			if s, err := theme.ImportEditorScheme(schemePath); err == nil {
				widgets.ApplyEditorScheme(s)
				log.Printf("  editor scheme: %s", s.Name)
			}
		}
	} else {
		log.Printf("No IntelliJ theme found in %s", themeDir)
	}

	// Font changer: discover installed families once (reads every
	// font file — done before the window shows). ./fonts is scanned
	// first so the bundled Inter Display resolves as a family too.
	fontScan := time.Now()
	systemFonts = graphics.ListSystemFonts("fonts")
	fontPathsByFamily = make(map[string]string, len(systemFonts))
	for _, f := range systemFonts {
		fontPathsByFamily[f.Family] = f.Path
	}
	log.Printf("Font changer: %d families discovered in %v", len(systemFonts), time.Since(fontScan))
	// Platform presets resolve their typography stacks through this
	// index (Noto Sans for Breeze, Adwaita Sans/Cantarell for GNOME,
	// SF Pro/Segoe when installed), falling back to the active face.
	theme.SetFontResolver(func(family string, size float32) graphics.Font {
		p, ok := fontPathsByFamily[family]
		if !ok {
			return nil
		}
		ff, err := graphics.LoadFontFace(p)
		if err != nil {
			return nil
		}
		return ff.Font(size)
	})

	// MEUI_THEME_FILE=path: load a native theme file (JSON with base
	// inheritance) and select it unless MEUI_THEME also picks one.
	if path := os.Getenv("MEUI_THEME_FILE"); path != "" {
		if id, err := theme.LoadThemeFile(path); err != nil {
			log.Printf("MEUI_THEME_FILE: %v", err)
		} else if os.Getenv("MEUI_THEME") == "" {
			if err := theme.LoadPreset(id); err != nil {
				log.Printf("MEUI_THEME_FILE: %v", err)
			} else {
				log.Printf("Loaded theme file: %s (%s)", id, path)
			}
		}
	}

	// MEUI_THEME=<preset-id>: native platform preset (macos-dark,
	// windows11-light, adwaita-dark, breeze-light, metro, meui-dark…)
	// overrides the imported theme. See theme.PresetIDs().
	if id := os.Getenv("MEUI_THEME"); id != "" {
		if id == "auto" {
			id = theme.AutoPresetID()
			log.Printf("Detected platform preset: %s", id)
			if err := theme.LoadAutoPreset(); err != nil {
				log.Printf("MEUI_THEME: %v", err)
			} else {
				log.Printf("Loaded platform preset: %s", id)
			}
		} else if err := theme.LoadPreset(id); err != nil {
			log.Printf("MEUI_THEME: %v", err)
		} else {
			log.Printf("Loaded platform preset: %s", id)
		}
	}
	// MEUI_ACCENT=#RRGGBB overrides the accent on any loaded theme
	// (testing hook for the system-accent follow path).
	if hex := os.Getenv("MEUI_ACCENT"); hex != "" {
		if c, ok := theme.ParseHexAccent(hex); ok {
			theme.ApplyAccent(theme.DefaultTheme, c)
			theme.Apply(theme.DefaultTheme)
			log.Printf("Accent override: %s", hex)
		} else {
			log.Printf("MEUI_ACCENT: unparsable %q", hex)
		}
	}
	// MEUI_FONTSHEET_FONT=<family>: start the app on that font
	// (deterministic fontsheet captures: pair with MEUI_FONTSHEET=1
	// and MEUI_FONTSHEET_CAPTURE=<path>).
	if fam := os.Getenv("MEUI_FONTSHEET_FONT"); fam != "" {
		applyAppFont(fam, nil)
	}

	log.Printf("Loading QML from: %s", qmlPath)

	result, err := qml.ParseWithBindingsFile(qmlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading QML: %v\n", err)
		os.Exit(1)
	}

	widget := result.Widget
	registry := result.Widgets

	// Inject widgets into model via qml:"id" struct tags
	var model struct {
		CtxBtn        *widgets.Button    `qml:"ctxBtn"`
		RichLabelLink *widgets.Label     `qml:"richLabelLink"`
		MsgBoxBtn     *widgets.Button    `qml:"msgBoxBtn"`
		InputDlgBtn   *widgets.Button    `qml:"inputDlgBtn"`
		OpenFileBtn   *widgets.Button    `qml:"openFileBtn"`
		SaveFileBtn   *widgets.Button    `qml:"saveFileBtn"`
		ThemeList     *widgets.ListView  `qml:"lstViewThemSelect"`
		FontCombo     *widgets.ComboBox  `qml:"cmbFontSelect"`
		DataTable     *widgets.DataTable `qml:"dataTable"`
		TabDemo       *widgets.TabPanel  `qml:"tabDemoContainer"`

		// Markdown editor demo tab (two-pane source + preview).
		MdEditor   *widgets.MarkdownEditor `qml:"mdEditor"`
		MdSplit    *widgets.Button         `qml:"mdModeSplit"`
		MdSource   *widgets.Button         `qml:"mdModeSource"`
		MdPreview  *widgets.Button         `qml:"mdModePreview"`
		MdReadOnly *widgets.Button         `qml:"mdReadOnly"`
		MdStatus   *widgets.Label          `qml:"mdStatus"`

		// The field widgets on the Calendar & Color tab, with the labels their
		// callbacks update.
		DpDemo    *widgets.DatePicker `qml:"dpDemo"`
		DpUnset   *widgets.DatePicker `qml:"dpUnset"`
		DpStatus  *widgets.Label      `qml:"dpStatus"`
		TpDemo    *widgets.TimePicker `qml:"tpDemo"`
		TpRollers *widgets.TimePicker `qml:"tpRollers"`
		TpSecond  *widgets.TimePicker `qml:"tpSeconds"`
		TpStatus  *widgets.Label      `qml:"tpStatus"`
		CfDemo    *widgets.ColorField `qml:"cfDemo"`
		CfTrans   *widgets.ColorField `qml:"cfTranslucent"`
		CfStatus  *widgets.Label      `qml:"cfStatus"`

		// Status bar + the standalone editors, so its Ln/Col section follows
		// whichever editor has the cursor.
		StatusBar *widgets.StatusBar  `qml:"statusBar"`
		CeMain    *widgets.CodeEditor `qml:"ceMain"`
		CeTest    *widgets.CodeEditor `qml:"ceTest"`
		CeWidget  *widgets.CodeEditor `qml:"ceWidget"`
		MainMenu  *widgets.MenuBar    `qml:"mainMenu"`
	}
	// MEUI_TAB=<index>: start on that main TabPanel tab (per-tab
	// screenshots without interaction).
	if v := os.Getenv("MEUI_TAB"); v != "" {
		if idx, err := strconv.Atoi(v); err == nil {
			if w := registry.Get("tabPanel"); w != nil {
				if tp, ok := w.(*widgets.TabPanel); ok {
					tp.SetActiveTab(idx)
					log.Printf("MEUI_TAB: active tab %d", idx)
				}
			}
		}
	}
	if err := registry.Inject(&model); err != nil {
		log.Printf("Inject warning: %v (some demo features disabled)", err)
	}

	win, ok := widget.(*meui.Window)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: QML root is not a meui.Window\n")
		os.Exit(1)
	}

	// Second-window content for the New Window button — parsed up
	// front (pure Go, no GL) so a broken file fails fast at startup.
	var secondWin *meui.Window
	secondPath := filepath.Join(filepath.Dir(qmlPath), "second.qml")
	if s, err := qml.ParseWithBindingsFile(secondPath); err != nil {
		log.Printf("second.qml: %v (New Window gets an empty client area)", err)
	} else if w2, ok := s.Widget.(*meui.Window); ok {
		secondWin = w2
		log.Printf("second.qml loaded: %T, children: %d", w2, len(w2.Children()))
	}

	// If dump-json flag is set, output JSON and exit
	if *dumpJSON {
		panel := win.AsPanel()
		if panel == nil {
			fmt.Fprintf(os.Stderr, "Error: Window has no Panel\n")
			os.Exit(1)
		}
		jsonStr := qml.DumpWidgetTree(win)
		fmt.Println(jsonStr)
		os.Exit(0)
	}

	log.Printf("Window: %T, children: %d", win, len(win.Children()))

	// Wire right-click context menu
	if model.CtxBtn != nil {
		ctxBtn := model.CtxBtn
		ctxBtn.OnRightClicked = func() {
			fr := ctxBtn.FrameRect()
			pt := ctxBtn.PointToRoot(fr.X, fr.Y+fr.Height)
			win.ShowContextMenuPositioned("editorMenu", pt.X, pt.Y)
		}
	}

	// Wire rich text link click
	if model.RichLabelLink != nil {
		model.RichLabelLink.OnLinkClicked = func(url string) {
			log.Printf("Link clicked: %s", url)
		}
	}

	// Wire dialog buttons
	if model.MsgBoxBtn != nil {
		model.MsgBoxBtn.OnClicked = func() {
			widgets.ShowMessageBox(win, "Message", "This is a message dialog\nwith multiple lines.", []string{"OK"}, nil)
		}
	}
	if model.InputDlgBtn != nil {
		model.InputDlgBtn.OnClicked = func() {
			widgets.ShowInputDialog(win, "Input", "Enter your name:", func(val string, ok bool) {
				if ok {
					widgets.ShowMessageBox(win, "Result", "You entered: "+val, []string{"OK"}, nil)
				}
			})
		}
	}
	if model.OpenFileBtn != nil {
		model.OpenFileBtn.OnClicked = func() {
			ref, ok := widgets.OpenFile("Open File")
			if ok {
				widgets.ShowMessageBox(win, "File Selected", ref.String(), []string{"OK"}, nil)
			}
		}
	}
	if model.SaveFileBtn != nil {
		model.SaveFileBtn.OnClicked = func() {
			ref, ok := widgets.SaveFile("Save File")
			if ok {
				widgets.ShowMessageBox(win, "File Selected", ref.String(), []string{"OK"}, nil)
			}
		}
	}

	// ── About menu item ──
	if model.MainMenu != nil {
		model.MainMenu.SetMenuItemAction("Help", "About", func() {
			widgets.ShowMessageBox(win,
				"About MeUi",
				"MeUi UI Toolkit\n"+AppVersion+"\n\nA flexible UI library using unison\nBuilt with Go and GLFW",
				[]string{"OK"}, nil)
		})
	}

	// ── Theme selector ──
	if model.ThemeList != nil {
		// Re-resolve (MEUI_THEMES, ./themes, ../meui/themes,
		// ../../themes — first that exists).
		themeDir = resolveThemesDir()
		// Scan for .theme.json files, read name from inside each file
		type themeEntry struct {
			display  string
			filename string
		}
		entries, err := os.ReadDir(themeDir)
		var themes []themeEntry
		if err == nil {
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".theme.json") {
					continue
				}
				filename := e.Name()
				fpath := filepath.Join(themeDir, filename)
				display := strings.TrimSuffix(filename, ".theme.json")
				// Read name from inside the file
				if data, err := os.ReadFile(fpath); err == nil {
					var info struct {
						Name string `json:"name"`
					}
					if err := json.Unmarshal(data, &info); err == nil && info.Name != "" {
						display = info.Name
					}
				}
				themes = append(themes, themeEntry{display, filename})
			}
			sort.Slice(themes, func(i, j int) bool { return themes[i].display < themes[j].display })
			displayNames := make([]string, len(themes))
			for i, t := range themes {
				displayNames[i] = t.display
			}
			model.ThemeList.SetItems(displayNames)
			log.Printf("Found %d themes in %s", len(themes), themeDir)
		}
		// On selection, apply selected theme
		model.ThemeList.SetOnItemSelected(func(idx int, _ string) {
			if idx < 0 || idx >= len(themes) {
				return
			}
			themePath := filepath.Join(themeDir, themes[idx].filename)
			if t, err := theme.ImportIntelliJThemeJSON(themePath); err == nil {
				// Preserve font metrics from current theme — only swap colors
				t.BaseFont = theme.DefaultTheme.BaseFont
				t.FontSizeSmall = theme.DefaultTheme.FontSizeSmall
				t.FontSizeMedium = theme.DefaultTheme.FontSizeMedium
				t.FontSizeLarge = theme.DefaultTheme.FontSizeLarge
				t.FontSizeTitle = theme.DefaultTheme.FontSizeTitle
				t.FontWeightNormal = theme.DefaultTheme.FontWeightNormal
				t.FontWeightBold = theme.DefaultTheme.FontWeightBold
				t.LineHeight = theme.DefaultTheme.LineHeight
				t.BorderRadius = theme.DefaultTheme.BorderRadius
				theme.Apply(t)

				// Load editor scheme (XML/.icls) if available
				schemePath := findEditorScheme(themePath, themeDir)
				if schemePath != "" {
					if s, err := theme.ImportEditorScheme(schemePath); err == nil {
						widgets.ApplyEditorScheme(s)
						log.Printf("  editor scheme: %s", s.Name)
					}
				}

				log.Printf("Switched to theme: %s", t.Name)
			} else {
				log.Printf("Failed to load theme %s: %v", themes[idx].display, err)
			}
		})
	}

	panel := win.AsPanel()
	if panel == nil {
		fmt.Fprintf(os.Stderr, "Error: Window has no Panel\n")
		os.Exit(1)
	}

	sdl3.AttachConsole()

	sdl3.Start(sdl3.StartupFinishedCallback(func() {
		// Size the OS window to the QML Window's declared size so the
		// frame bar and content fill it exactly.
		winW := int32(win.FrameRect().Width)
		winH := int32(win.FrameRect().Height)
		if winW <= 0 {
			winW, winH = 1100, 750
		}
		// MEUI_WINDOW_H=<px>: taller window for capturing full tab
		// content that would otherwise clip below the default height.
		if v := os.Getenv("MEUI_WINDOW_H"); v != "" {
			if h, err := strconv.Atoi(v); err == nil && h > 0 {
				winH = int32(h)
			}
		}
		wnd, err := sdl3.NewWindowSized("MeUi Demo v3 — "+qmlPath, winW, winH)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating window: %v\n", err)
			os.Exit(1)
		}

		// Theme already applied before QML parse (single path) — the
		// window opens on the final theme; no late swap here.

		// MEUI_FULLSCREEN=1: cover the whole monitor so per-tab
		// screenshots show the complete tab content without clipping.
		if os.Getenv("MEUI_FULLSCREEN") == "1" {
			wnd.SetFullscreen(true)
			log.Printf("Fullscreen start")
		}

		width := float32(900)
		height := float32(700)
		if w := win.FrameRect().Width; w > 0 {
			width = w
		}
		if h := win.FrameRect().Height; h > 0 {
			height = h
		}

		log.Printf("Window size: %vx%v", width, height)

		// TEXT-UPDATE DIAGNOSTIC: 3s after start, rewrite a visible label.
		// If the on-screen text changes, live SetText works; if not, the
		// widget text/redraw pipeline is the bug.
		if rl := registry.Get("richLabelLink"); rl != nil {
			go func() {
				time.Sleep(3 * time.Second)
				wnd.PostTask(func() {
					if l, ok := rl.(*widgets.Label); ok {
						l.SetText("TEXT-UPDATE-OK")
						log.Printf("[diag] richLabelLink text updated")
					}
				})
			}()
		}

		// Lifecycle test hook: MEUI_AUTO_CLOSE_WIN1=<secs> closes the main
		// window programmatically (app-exit test: the process must exit
		// cleanly when the LAST window closes).
		if n := os.Getenv("MEUI_AUTO_CLOSE_WIN1"); n != "" {
			if secs, err := strconv.Atoi(n); err == nil {
				go func() {
					time.Sleep(time.Duration(secs) * time.Second)
					log.Printf("[lifecycle] auto-closing window 1")
					wnd.Close()
				}()
			}
		}

		// Custom title bar: content area starts below the bar. Use the
		// REAL window size (display scaling may scale the request).
		width = float32(wnd.Width())
		height = float32(wnd.Height())
		log.Printf("Real window size: %vx%v", width, height)
		const barH = float32(widgets.FrameBarHeight)
		_ = barH
		// The content tree is 0-based; the shim translates draw/events
		// by the bar height. Content height = window height minus bar.
		wnd.SetContentRect(graphics.NewRect(0, 0, width, height-barH))
		// Declare where the content begins in window space: the toolkit uses
		// it to place popups created by chrome widgets (the title-bar font
		// combo) without any widget doing bar arithmetic.
		win.SetContentOrigin(0, barH)
		wnd.ValidateLayout()
		// MEUI_FONTSHEET=1: replace the content with a font inspection
		// sheet — every size rendered through the live text pipeline.
		if os.Getenv("MEUI_FONTSHEET") == "1" {
			sizes := []float32{6, 7, 8, 9, 10, 11, 12, 13, 14, 16, 18, 20, 24}
			sample := "ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmnopqrstuvwxyz 0123456789 !@#$%&*()+-=[]{};:,.<>?/~"
			// Follow the app font (theme BaseFont) — the title-bar
			// font changer and MEUI_FONTSHEET_FONT=<family> control
			// which face the sheet renders.
			sheet := &fontSheetDrawable{sizes: sizes, sample: sample}
			wnd.SetContent(sheet)
			// Auto-capture the framebuffer and exit (pixel forensics).
			go func() {
				time.Sleep(2 * time.Second)
				wnd.CaptureNext(func(pix []uint32, w, h int32) {
					out := os.Getenv("MEUI_FONTSHEET_CAPTURE")
					if out == "" {
						out = "/tmp/opencode/fontsheet_render.png"
					}
					if err := writePNG(out, pix, w, h); err != nil {
						log.Printf("[fontsheet] capture: %v", err)
					} else {
						log.Printf("[fontsheet] captured %dx%d -> %s", w, h, out)
					}
					wnd.Close()
				})
			}()
		} else {
			wnd.SetContent(panel)
		}
		wnd.SetSink(win) // route SDL input into the meui event sink

		frame := widgets.NewWindowFrame()
		frame.SetOwnerRoot(win)
		frame.SetTitle(fmt.Sprintf("MeUi Demo v5 [%s]", buildID))
		frame.OnClose(func() { wnd.Close() })
		frame.OnMinimize(func() { wnd.Minimize() })
		frame.OnMaximize(func() { wnd.ToggleMaximize() })
		if laf == "macos" && os.Getenv("MEUI_THEME") == "" {
			// Legacy: MEUI_LAF alone drives the chrome; presets follow
			// their own Chrome token automatically.
			frame.SetChrome("macos")
		}
		// Rounded windows follow the preset (Win11 8, macOS 10, GNOME 12,
		// Breeze 4); MEUI_WINDOW_ROUNDING=0/1 overrides for testing.
		rounding := theme.Shape().Window
		switch os.Getenv("MEUI_WINDOW_ROUNDING") {
		case "0":
			rounding = 0
		case "1":
			rounding = theme.Shape().Window
			if rounding <= 0 {
				rounding = 10
			}
		}
		if rounding > 0 {
			topOnly := theme.DefaultTheme != nil && theme.DefaultTheme.Meta.Platform == theme.PlatformGNOME
			wnd.SetWindowRoundedCorners(rounding, topOnly)
		}
		wnd.SetTitleBar(frame)

		// Frame middle: New Window + Fullscreen buttons.
		openSecondWindow := func() {
			sdl3.OpenWindowAsync("MeUi Demo - Second Window", 400, 300,
				func(wnd2 *sdl3.Window) {
					const barH = float32(widgets.FrameBarHeight)
					wnd2.SetContentRect(graphics.NewRect(0, 0, 400, 300-barH))
					if secondWin != nil {
						secondWin.SetContentOrigin(0, barH)
					}
					frame2 := widgets.NewWindowFrame()
					if secondWin != nil {
						frame2.SetOwnerRoot(secondWin)
					}
					frame2.SetTitle("MeUi Demo - Second Window")
					if laf == "macos" && os.Getenv("MEUI_THEME") == "" {
						frame2.SetChrome("macos")
					}
					frame2.OnClose(func() { wnd2.Close() })
					frame2.OnMinimize(func() { wnd2.Minimize() })
					frame2.OnMaximize(func() { wnd2.ToggleMaximize() })
					wnd2.SetTitleBar(frame2)
					if secondWin != nil {
						wnd2.SetContent(secondWin.AsPanel())
						wnd2.SetSink(secondWin)
					}
					if os.Getenv("MEUI_WINDOW_ROUNDING") == "1" {
						wnd2.SetWindowRounded(theme.Shape().Window)
					}
					// Lifecycle test hook: MEUI_AUTO_CLOSE_WIN2=<secs>
					// closes this window programmatically (deterministic
					// close-path test, no coordinate guessing).
					if n := os.Getenv("MEUI_AUTO_CLOSE_WIN2"); n != "" {
						if secs, err := strconv.Atoi(n); err == nil {
							go func() {
								time.Sleep(time.Duration(secs) * time.Second)
								log.Printf("[lifecycle] auto-closing window 2")
								wnd2.Close()
							}()
						}
					}
				})
		}

		// MEUI_AUTO_OPEN_WIN2=<secs>: open the second window
		// programmatically (deterministic lifecycle tests).
		if n := os.Getenv("MEUI_AUTO_OPEN_WIN2"); n != "" {
			if secs, err := strconv.Atoi(n); err == nil {
				go func() {
					time.Sleep(time.Duration(secs) * time.Second)
					log.Printf("[lifecycle] auto-opening window 2")
					openSecondWindow()
				}()
			}
		}

		frame.SetMiddle(newFrameMiddle(win, func() {
			openSecondWindow()
		}, frame, wnd))

		// ── Font changer combo (in the body, above the theme list) ──
		if model.FontCombo != nil {
			fc := model.FontCombo
			options := make([]string, 0, len(systemFonts)+1)
			options = append(options, "Roboto (embedded)")
			options = append(options, func() []string {
				names := make([]string, len(systemFonts))
				for i, f := range systemFonts {
					names[i] = f.Family
				}
				return names
			}()...)
			fc.SetOptions(options)
			fc.SetMaxDropdownHeight(420)
			fc.SetOnSelected(func(_ int, text string) {
				applyAppFont(text, wnd)
			})
			// Initial selection = the active app font.
			if base := theme.DefaultTheme.BaseFont; base != nil {
				for i, opt := range options {
					if strings.EqualFold(opt, base.Family()) {
						fc.SetSelectedIndex(i)
						break
					}
				}
			}
		}

		wnd.Show()

		// MEUI_TITLEBAR_COMBO_PROBE=1: simulate the app-side path of a
		// press on the title-bar combo (BarMouseDown/Up — exactly what
		// the shim runs when SDL hit-test classifies NORMAL), let the
		// dropdown open, capture, exit.
		if os.Getenv("MEUI_TITLEBAR_COMBO_PROBE") == "1" {
			go func() {
				time.Sleep(3 * time.Second)
				if titlebarCombo == nil {
					log.Printf("[probe] no titlebar combo")
					return
				}
				hf := titlebarCombo.Parent().FrameRect()
				cf := titlebarCombo.FrameRect()
				cx := hf.X + cf.X + cf.Width/2
				cy := hf.Y + cf.Y + cf.Height/2
				pt := titlebarCombo.PointToRoot(0, 0)
				log.Printf("[probe] combo frame=%v abs=(%.0f,%.0f) expectedDropdownY=%.0f hostFrame=%v hit=%d",
					cf, pt.X, pt.Y, pt.Y+cf.Height, hf, frame.FrameHit(cx, cy, 800, 600))
				frame.BarMouseDown(cx, cy, 0)
				frame.BarMouseUp(cx, cy, 0)
				time.Sleep(800 * time.Millisecond)
				wnd.CaptureNext(func(pix []uint32, w, h int32) {
					if err := writePNG("/tmp/opencode/titlebar_probe.png", pix, w, h); err != nil {
						log.Printf("[probe] %v", err)
					} else {
						log.Printf("[probe] captured -> /tmp/opencode/titlebar_probe.png")
					}
					wnd.Close()
				})
			}()
		}

		// MEUI_TIMEPICKER_PROBE=1: open the rollers picker's panel shortly after
		// start, so MEUI_CAPTURE_SHOT can photograph the wheel without a click
		// (the same trick as MEUI_TITLEBAR_COMBO_PROBE).
		if os.Getenv("MEUI_TIMEPICKER_PROBE") == "1" {
			go func() {
				time.Sleep(3 * time.Second)
				wnd.PostTask(func() {
					if model.TpRollers == nil {
						return
					}
					if w := registry.Get("tabPanel"); w != nil {
						if tp, ok := w.(*widgets.TabPanel); ok {
							tp.SetActiveTab(7)
						}
					}
					model.TpRollers.SetExpanded(true)
					log.Printf("[probe] rollers panel open: %v", model.TpRollers.Expanded())
				})
			}()
		}

		// MEUI_AUTO_MAXIMIZE=1: maximize the window ~3s after start
		// (resize-clipping reproduction: capture afterwards and compare
		// painted content bounds against the OS window size).
		if os.Getenv("MEUI_AUTO_MAXIMIZE") == "1" {
			go func() {
				time.Sleep(3 * time.Second)
				wnd.PostTask(func() {
					wnd.ToggleMaximize()
					log.Printf("[lifecycle] auto-maximized")
				})
			}()
		}

		// MEUI_CAPTURE_SHOT=<path>: capture the framebuffer after
		// MEUI_CAPTURE_SECS (default 3) and exit the app — exact-pixel
		// forensics of the live UI, no screen-scaler involvement.
		if out := os.Getenv("MEUI_CAPTURE_SHOT"); out != "" {
			secs := 3
			if s := os.Getenv("MEUI_CAPTURE_SECS"); s != "" {
				if v, err := strconv.Atoi(s); err == nil {
					secs = v
				}
			}
			go func() {
				time.Sleep(time.Duration(secs) * time.Second)
				wnd.CaptureNext(func(pix []uint32, w, h int32) {
					if err := writePNG(out, pix, w, h); err != nil {
						log.Printf("[shot] %v", err)
					} else {
						log.Printf("[shot] captured -> %s", out)
					}
					wnd.Close()
				})
			}()
		}

		// ── DataTable demo ──
		if model.DataTable != nil {
			dt := model.DataTable
			// 60 rows so the vertical scrollbar gets a real workout.
			symbols := []string{"AAPL", "GOOGL", "MSFT", "AMZN", "NVDA", "TSLA", "META", "JPM", "V", "PG"}
			const seriesPerSymbol = 6
			rows := make([]widgets.DataRow, 0, len(symbols)*seriesPerSymbol)
			for series := 0; series < seriesPerSymbol; series++ {
				for _, sym := range symbols {
					i := len(rows)
					bid := 100.0 + float64(i)*5 + rand.Float64()*10
					ask := bid + 0.05 + rand.Float64()*0.15
					last := (bid + ask) / 2
					chg := (rand.Float64() - 0.5) * 4
					vol := int64(100000 + rand.Float64()*900000)
					name := sym
					if series > 0 {
						name = fmt.Sprintf("%s.%d", sym, series)
					}
					rows = append(rows, widgets.DataRow{Cells: []widgets.DataCell{
						{Text: name, Color: "#88CCFF"},
						{Text: fmt.Sprintf("%.2f", bid)},
						{Text: fmt.Sprintf("%.2f", ask)},
						{Text: fmt.Sprintf("%.2f", last)},
						{Text: fmt.Sprintf("%+.2f%%", chg)},
						{Text: fmt.Sprintf("%d", vol)},
					}})
				}
			}
			dt.SetRows(rows)

			// Add widgets to each row — checkbox in col 0, delete button in col 1
			for i := range rows {
				cb := widgets.NewCheckBox()
				cb.SetWidthDirect(18)
				cb.SetHeightDirect(18)
				if i%2 == 0 {
					cb.SetChecked(true)
				}
				idx := i
				cb.OnChanged = func(checked bool) {
					log.Printf("[DataTable::CheckBox] CheckBox row %d: %v", idx, checked)
				}
				dt.Rows[i].Cells[0].Widget = cb

				delBtn := widgets.NewButton()
				// delBtn.SetText("X")
				delBtn.SetIcon("close")
				delBtn.SetWidthDirect(18)
				delBtn.SetHeightDirect(18)
				delBtn.OnClicked = func() {
					log.Printf("[DataTable::Button] Delete row %d", idx)
				}
				dt.Rows[i].Cells[1].Widget = delBtn
			}

			// Print cell clicks
			dt.OnCellClicked = func(row, col int, cell *widgets.DataCell) {
				log.Printf("[DataTable] Cell clicked: row=%d col=%d text=%q", row, col, cell.Text)
			}

			// Simulate high-frequency updates
			lastUpdate := time.Now()
			platform.InvokeTask(func() {
				ticker := time.NewTicker(100 * time.Millisecond)
				go func() {
					for range ticker.C {
						now := time.Now()
						if now.Sub(lastUpdate) < 50*time.Millisecond {
							continue
						}
						lastUpdate = now
						n := rand.Intn(5) + 1
						updates := make([]widgets.CellUpdate, n)
						seen := make(map[int]map[int]bool)
						for j := 0; j < n; j++ {
							ri := rand.Intn(len(rows))
							ci := rand.Intn(5) + 1 // bid/ask/last/chg/vol
							if seen[ri] == nil {
								seen[ri] = make(map[int]bool)
							}
							if seen[ri][ci] {
								j--
								continue
							}
							seen[ri][ci] = true
							base := 100.0 + float64(ri%len(symbols))*50
							var val string
							switch ci {
							case 1: // bid
								val = fmt.Sprintf("%.2f", base+rand.Float64()*10)
							case 2: // ask
								val = fmt.Sprintf("%.2f", base+0.05+rand.Float64()*0.15)
							case 3: // last
								val = fmt.Sprintf("%.2f", base+rand.Float64()*10)
							case 4: // chg
								val = fmt.Sprintf("%+.2f%%", (rand.Float64()-0.5)*4)
							case 5: // vol
								val = fmt.Sprintf("%d", int64(100000+rand.Float64()*900000))
							}
							updates[j] = widgets.CellUpdate{Row: ri, Col: ci, Text: val}
						}
						platform.InvokeTask(func() {
							dt.UpdateCells(updates)
						})
					}
				}()
			})
		}

		// ── TabPanel feature demo ──
		if model.TabDemo != nil {
			tp := model.TabDemo
			log.Printf("[TabPanel] tabDemoContainer frame: %.0f x %.0f", tp.Width(), tp.Height())
			tp.SetClosable(true)

			// OnClose — prevents closing the first tab
			tp.OnClose = func(id, name string) bool {
				if id == "tabClose" {
					log.Printf("[TabPanel] Closable tab closing — cancelling (return false)")
					return false
				}
				log.Printf("[TabPanel] Tab %q closed", name)
				return true
			}

			// Action buttons at end of tab bar
			tp.ActionButtons = []widgets.TabActionButton{
				{Label: "+", Tooltip: "Add tab", Width: 24, OnClick: func() {
					log.Println("[TabPanel] + action button clicked")
				}},
				{Label: "⋮", Tooltip: "More actions", Width: 24, OnClick: func() {
					// Show a context menu from the action button
					if win := tp.Window(); win != nil {
						cm := widgets.NewContextMenu()
						cm.SetItems([]widgets.MenuItem{
							{Label: "Close All", Enabled: true, Action: func() {
								log.Println("[TabPanel] Close All")
							}},
							{Label: "Close Others", Enabled: true, Action: func() {
								log.Println("[TabPanel] Close Others")
							}},
						})
						// Position at the right end of the tab bar
						fr := tp.FrameRect()
						win.ShowPopupContextMenu(cm, fr.X+fr.Width-80, fr.Y+30)
					}
				}},
			}

			tp.MarkForRedraw()
		}

		// ── DatePicker / ColorField demo tab ──
		// Both are compositions: a text box that owns the value and a button
		// that opens the picker (the calendar, the colour picker). The status
		// labels prove the user-facing callbacks fire — a programmatic set
		// deliberately does not.
		if model.DpDemo != nil {
			showDate := func(y, m, d int) {
				if model.DpStatus != nil {
					model.DpStatus.SetText(fmt.Sprintf("%04d-%02d-%02d", y, m, d))
				}
			}
			model.DpDemo.SetOnDateChanged(showDate)
			y, m, d := model.DpDemo.Date()
			showDate(y, m, d)
			if model.DpUnset != nil {
				log.Printf("[DatePicker] unset picker has a date: %v", model.DpUnset.HasDate())
			}
		}
		if model.TpDemo != nil {
			showTime := func(h, m, s int) {
				if model.TpStatus != nil {
					model.TpStatus.SetText(fmt.Sprintf("%02d:%02d:%02d", h, m, s))
				}
			}
			model.TpDemo.SetOnTimeChanged(showTime)
			h, m, s := model.TpDemo.Time()
			showTime(h, m, s)
			if model.TpRollers != nil {
				log.Printf("[TimePicker] rollers style selected: %v", model.TpRollers.Style() == widgets.TimePickerRollers)
			}
			if model.TpSecond != nil {
				model.TpSecond.SetShowSeconds(true)
				log.Printf("[TimePicker] seconds field shows %q, has time: %v", model.TpSecond.Text(), model.TpSecond.HasTime())
			}
		}
		if model.CfDemo != nil {
			showColor := func(c graphics.Color) {
				if model.CfStatus == nil {
					return
				}
				r, g, b, a := widgets.ColorChannels(c)
				model.CfStatus.SetText(fmt.Sprintf("%s  r=%d g=%d b=%d a=%d", widgets.FormatHexColor(c), r, g, b, a))
			}
			model.CfDemo.SetOnColorChanged(showColor)
			showColor(model.CfDemo.Color())
			if model.CfTrans != nil {
				log.Printf("[ColorField] translucent field holds %s", model.CfTrans.Hex())
			}
		}

		// ── Markdown editor demo tab ──
		// Two panes over one document. The controls switch which pane is
		// shown; the status label proves the widget's mode/read-only state,
		// and OnTextChanged proves edits reach the app.
		if model.MdEditor != nil {
			ed := model.MdEditor
			updateMdStatus := func() {
				mode := "split"
				switch ed.Mode() {
				case widgets.MarkdownSourceOnly:
					mode = "source only"
				case widgets.MarkdownPreviewOnly:
					mode = "preview only"
				}
				state := "editable"
				if ed.IsReadOnly() {
					state = "read-only"
				}
				if model.MdStatus != nil {
					model.MdStatus.SetText(state + " · " + mode)
				}
			}
			if model.MdSplit != nil {
				model.MdSplit.OnClicked = func() {
					ed.SetMode(widgets.MarkdownSplit)
					updateMdStatus()
				}
			}
			if model.MdSource != nil {
				model.MdSource.OnClicked = func() {
					ed.SetMode(widgets.MarkdownSourceOnly)
					updateMdStatus()
				}
			}
			if model.MdPreview != nil {
				model.MdPreview.OnClicked = func() {
					ed.SetMode(widgets.MarkdownPreviewOnly)
					updateMdStatus()
				}
			}
			if model.MdReadOnly != nil {
				model.MdReadOnly.OnClicked = func() {
					ro := !ed.IsReadOnly()
					ed.SetReadOnly(ro)
					if ro {
						model.MdReadOnly.SetText("Editable")
					} else {
						model.MdReadOnly.SetText("Read-only")
					}
					updateMdStatus()
				}
			}
			ed.OnTextChanged = func(text string) {
				log.Printf("[MarkdownEditor] text changed: %d bytes", len(text))
				updateMdStatus()
			}
			updateMdStatus()
			log.Printf("[MarkdownEditor] demo tab wired (source=%p preview=%p)", ed.Source(), ed.Preview())
		}

		// ── Status bar: follow the cursor of whichever editor is used ──
		// CodeEditor already reports cursor moves, so the bar is a plain
		// callback target (Ln/Col are 1-based for display).
		if model.StatusBar != nil {
			sb := model.StatusBar
			sections := func() []string {
				return []string{"v7 50440a5", "Ln 1, Col 1", "UTF-8", "●"}
			}
			setCursor := func(line, col int) {
				secs := sections()
				secs[1] = fmt.Sprintf("Ln %d, Col %d", line+1, col+1)
				sb.SetSections(secs)
			}
			editors := []*widgets.CodeEditor{model.CeMain, model.CeTest, model.CeWidget}
			if model.MdEditor != nil {
				editors = append(editors, model.MdEditor.Source())
			}
			wired := 0
			for _, ce := range editors {
				if ce == nil {
					continue
				}
				ce.OnCursorChanged = setCursor
				wired++
			}
			log.Printf("[StatusBar] Ln/Col wired to %d editor(s)", wired)
		}
	}))
}

// findEditorScheme looks for the editor scheme XML file associated with a .theme.json.
func findEditorScheme(themePath, themeDir string) string {
	// Read the editorScheme field from the theme JSON
	data, err := os.ReadFile(themePath)
	if err != nil {
		return ""
	}
	var info struct {
		EditorScheme string `json:"editorScheme"`
	}
	if err := json.Unmarshal(data, &info); err != nil || info.EditorScheme == "" {
		return ""
	}
	// Extract filename from path like "/one_dark.xml" or "/META-INF/alabaster.xml"
	schemeFile := filepath.Base(info.EditorScheme)
	if schemeFile == "" {
		return ""
	}
	candidate := filepath.Join(themeDir, schemeFile)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

// newFrameMiddle builds the configurable middle section of the custom
// title bar: a status label and a "New Window" button in an HStack.
// newFrameMiddle builds the configurable middle of the title bar:
// a New Window button and a Fullscreen toggle side by side.
func newFrameMiddle(appWin *meui.Window, onNew func(), frame *widgets.WindowFrame, wnd *sdl3.Window) meui.Widget {
	newBtn := widgets.NewButton()
	newBtn.SetText("New Window")
	newBtn.SetWidthDirect(90)
	newBtn.SetHeightDirect(widgets.FrameBarHeight - 4)
	newBtn.SetX(8)
	newBtn.SetY(2)
	newBtn.OnClicked = func() {
		if onNew != nil {
			onNew()
		}
	}

	fsBtn := widgets.NewButton()
	fsBtn.SetText("Fullscreen")
	fsBtn.SetWidthDirect(90)
	fsBtn.SetHeightDirect(widgets.FrameBarHeight - 4)
	fsBtn.SetX(8 + 90 + 8)
	fsBtn.SetY(2)
	fs := false
	fsBtn.OnClicked = func() {
		fs = !fs
		if fs {
			fsBtn.SetText("Exit Fullscreen")
		} else {
			fsBtn.SetText("Fullscreen")
		}
		wnd.SetFullscreen(fs)
	}

	host := meui.NewPanel()
	host.SetLayout(layout.NewHStack()) // positions the buttons; a bare
	// Panel runs AnchorLayout on paint, which zeroes unanchored frames
	host.AddChild(newBtn)
	host.AddChild(fsBtn)

	// Font changer IN the title bar — the drag-area regression test:
	// with SetMiddle parenting + FrameHit NORMAL classification the
	// combo must be clickable and open its dropdown.
	fontCombo := widgets.NewComboBox()
	fontCombo.SetWidthDirect(190)
	fontCombo.SetHeightDirect(widgets.FrameBarHeight - 6)
	tbOptions := make([]string, 0, len(systemFonts)+1)
	tbOptions = append(tbOptions, "Roboto (embedded)")
	for _, f := range systemFonts {
		tbOptions = append(tbOptions, f.Family)
	}
	fontCombo.SetOptions(tbOptions)
	fontCombo.SetMaxDropdownHeight(420)
	fontCombo.SetOnSelected(func(_ int, text string) {
		log.Printf("[font] titlebar combo selected: %s", text)
		applyAppFont(text, wnd)
	})
	titlebarCombo = fontCombo
	host.AddChild(fontCombo)
	return host
}

// applyAppFont switches the theme base font to the given family.
// "Roboto (embedded)" uses the embedded face; anything else resolves
// through fontPathsByFamily (exact, then case-insensitive, then
// substring). SetDefaultTheme bumps the theme version so every open
// window repaints; wnd (when non-nil) is re-validated because glyph
// metrics change.
func applyAppFont(family string, wnd *sdl3.Window) bool {
	var ff *graphics.FontFace
	if family == "Roboto (embedded)" {
		ff = graphics.MatchFontFace("Roboto", weight.Light, spacing.Standard, slant.Upright)
	} else if p, ok := fontPathsByFamily[family]; ok {
		loaded, err := graphics.LoadFontFace(p)
		if err != nil {
			log.Printf("[font] load %q: %v", family, err)
			return false
		}
		ff = loaded
	} else {
		needle := strings.ToLower(family)
		for f, p := range fontPathsByFamily {
			if strings.EqualFold(f, family) || strings.Contains(strings.ToLower(f), needle) {
				loaded, err := graphics.LoadFontFace(p)
				if err != nil {
					log.Printf("[font] load %q: %v", f, err)
					return false
				}
				ff = loaded
				family = f
				break
			}
		}
	}
	if ff == nil {
		log.Printf("[font] family %q not found", family)
		return false
	}
	size := float32(12)
	if base := theme.DefaultTheme.BaseFont; base != nil {
		size = base.Size()
	}
	theme.DefaultTheme.BaseFont = ff.Font(size)
	theme.SetDefaultTheme(theme.DefaultTheme) // version++ → repaint all windows
	if wnd != nil {
		wnd.ValidateLayout()
	}
	log.Printf("[font] app font: %s %v", family, size)
	return true
}

// writePNG encodes 0xAARRGGBB bottom-up framebuffer pixels to a PNG
// (debug helper for multi-window framebuffer capture).
func writePNG(path string, pixels []uint32, w, h int32) error {
	img := image.NewNRGBA(image.Rect(0, 0, int(w), int(h)))
	for y := int32(0); y < h; y++ {
		srcY := h - 1 - y // source is bottom-up
		for x := int32(0); x < w; x++ {
			px := pixels[srcY*w+x]
			i := int(y)*img.Stride + int(x)*4
			img.Pix[i+0] = byte(px >> 16)
			img.Pix[i+1] = byte(px >> 8)
			img.Pix[i+2] = byte(px)
			img.Pix[i+3] = byte(px >> 24)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// fontSheetDrawable renders one sample line per font size through the
// live DrawText pipeline (font inspection for the text-quality work).
type fontSheetDrawable struct {
	sizes  []float32
	sample string
	face   *graphics.FontFace // nil = theme base font
}

func (f *fontSheetDrawable) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	p := graphics.NewPaint()
	p.SetColor(theme.DefaultTheme.TextPrimary)
	y := rect.Y + 4
	for _, s := range f.sizes {
		var fnt graphics.Font
		if f.face != nil {
			fnt = f.face.Font(s)
		} else {
			base := theme.DefaultTheme.BaseFont
			if base == nil {
				continue
			}
			if fc := base.Face(); fc != nil {
				fnt = fc.Font(s)
			}
		}
		if fnt == nil {
			continue
		}
		canvas.DrawText(fmt.Sprintf("size %v — %s", s, f.sample), rect.X+4, y+float32(s), fnt, p)
		y += float32(s) + 8
	}
}
