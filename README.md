# MeUI

**QML declares the surface. Go decides what it does.**  
**GPU-rendered, built for speed.**

[![Go 1.25](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![CI](https://github.com/sixnineone/meui/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/sixnineone/meui/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/sixnineone/meui.svg)](https://pkg.go.dev/github.com/sixnineone/meui)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/tag/sixnineone/meui?label=release)](https://github.com/sixnineone/meui/tags)
[![Desktop: cgo-free](https://img.shields.io/badge/desktop-cgo--free-2ea44f)](#quick-start)

<img src="docs/screens/dashboard.png" alt="A weather dashboard built with MeUI" width="100%">

MeUI is a Go widget toolkit for **web-looking pages** and **desktop consoles**. A window can
be built directly in Go, declared in Go structs with the `flow` wrappers, or written in QML —
a *builder*, not a language: no expressions, no bindings, no JavaScript. However the layout
is declared, every behaviour — a click, a selection, a scroll, a validation — is attached
from Go, so the logic is typed, testable and debuggable.

Rendering is GPU-first: geometry is batched and submitted through OpenGL
([mobilegles](https://github.com/sixnineone/mobilegles)), data views are virtualised, and a
window with nothing to redraw is not redrawn. The software device exists for headless,
deterministic screenshots and pixel tests — not as the production path.

## Why MeUI

- **Fast by construction.** Batched OpenGL draws, dirty-driven repaints and virtualised
  lists: the 240-message inbox keeps about a dozen widgets alive.
- **QML that can't lie.** The QML is data, not code. Ids are resolved when the scene is
  built, so a typo is an error at startup, not a handler that silently never runs. There
  is no second place for state to hide: behaviour lives in Go, next to the rest of the app.
- **Web pages *and* consoles.** Sticky headers, breakpoints, virtual lists and frosted
  glass on one side; menubars, typed tables, trees, inspectors and a navigation rail on the
  other. One widget set, two idioms.
- **One drawing seam.** Soft and GL devices draw the same primitives. The render-parity
  harness renders a scene through both and diffs the frames, so the two backends cannot
  drift apart unnoticed.
- **Deterministic screenshots.** Every example runs headless with `-shot out.png`; the
  behaviour harness asserts layout, hit-testing and painting without a display, and the
  visual harness asserts real pixels in a real window.
- **A toolkit you can theme.** 19 presets (web, macOS, Windows 11, Adwaita, Breeze,
  metro, iOS, Android, harness) plus 52 importable IntelliJ-style theme files, with
  per-widget overrides.
- **No cgo in the desktop build.** Control reaches SDL3 and OpenGL through purego, so the
  dev loop is plain `CGO_ENABLED=0 go run`.

## Built for speed

- **One batched GPU path.** Shapes, text and images are batched by primitive class in
  [`mobilegles`](https://github.com/sixnineone/mobilegles); a frame is a handful of draw
  calls, not one per widget.
- **Virtualised data.** `VirtualList` instantiates only the rows on screen — the mail
  example renders 240 messages with about twelve live widgets.
- **Dirty-driven.** Widgets request layout and repaint when they change; an unchanged
  window is not repainted on a timer.
- **No cgo in the desktop hot path.** SDL3 and OpenGL are reached through purego, so there
  is no cgo boundary to cross per call.
- **The software device is the test path.** It renders the same scene headlessly and
  deterministically for screenshots and pixel tests; apps ship the GPU path.

## Quick start

Requires Go 1.25.

```sh
git clone https://github.com/sixnineone/meui
cd meui
CGO_ENABLED=0 go run ./examples/launcher
```

The launcher lists every example; each one also runs on its own and every one can render a
frame without a display:

```sh
CGO_ENABLED=0 go run ./examples/docs                 # a real window, real input
CGO_ENABLED=0 go run ./examples/docs -shot docs.png  # the same scene, headless
```

## Install

```sh
go get github.com/sixnineone/meui@latest
go get github.com/sixnineone/meui-gl@latest   # the SDL3 window and input
```

API documentation follows the module: [pkg.go.dev/github.com/sixnineone/meui](https://pkg.go.dev/github.com/sixnineone/meui).
The smallest complete applications are [`examples/direct`](examples/direct) (no QML) and
[`examples/flow`](examples/flow) (declarative Go); [`examples/landing`](examples/landing) is
the QML one.

## Requirements

Go **1.25** is the only build-time requirement — the desktop build is cgo-free, so there
is no compiler or `-dev` package to install.

**To open a window** (anything without `-shot`) the machine needs Linux with:

- **SDL3** runtime (`libSDL3.so.0`) — windows, input, clipboard, dialogs and the tray
- **OpenGL 2.1+** (Mesa or the vendor driver) — the `mobilegles` renderer
- an **X11 display**; on Wayland the toolkit requests X11, and `SDL_VIDEODRIVER=wayland`
  asks for the native backend instead

```sh
# openSUSE Tumbleweed
sudo zypper install libSDL3-0

# Debian / Ubuntu 25.04+
sudo apt install libsdl3-0

# Fedora 41+
sudo dnf install SDL3

# Arch
sudo pacman -S sdl3
```

Releases that do not package SDL3 yet (for example Ubuntu 24.04) need SDL3 3.x built from
source and installed. Verify the runtime with:

```sh
ldconfig -p | grep libSDL3
```

**Headless** rendering (`-shot`, `qmlshot`, the test suites) needs none of that: no SDL3,
no display, no GPU — it draws with the software device.

**Optional, for development:** `python3` + `numpy` + `Pillow` for the render-parity
reference script, and `xvfb` to run the GL tests without a desktop session. A C toolchain
is only needed for the Android host, which lives in a separate repository.

## Three ways to build a window

The same window can be built three ways. Behaviour is Go in every case; only the place the
layout lives changes. Imports are omitted below.

**Direct Go.** Construct widgets and set their properties:

```go
win := meui.NewWindow()

content := meui.NewPanel()
content.SetLayout(layout.NewVStack())
win.AddChild(content)

title := widgets.NewLabel()
title.SetText("Hello, MeUI")
title.SetFontSize(28)
content.AddChild(title)

content.AddChild(widgets.NewButton().SetText("Press me"))
```

**Declarative Go — `flow`.** Keep the layout declarative without a `.qml` file: describe the
tree as Go structs and call `Build`. The compiler checks it, the fields mirror QML, and the
result is the same widget tree the parser produces.

> `flow` ships wrappers for `Window`, `Panel`, `VStack`, `HStack`, `Label`, `Button` and
> `CheckBox`. A widget without a wrapper plugs in by implementing `flow.WidgetWrapper`.

```go
app := &flow.Window{
	W:     flow.W{Width: 720, Height: 480},
	Title: "Hello",
	Children: []flow.WidgetWrapper{
		&flow.VStack{
			W:       flow.W{ID: "body", FillWidth: true, FillHeight: true},
			Spacing: 12,
			Children: []flow.WidgetWrapper{
				&flow.Label{
					Text:      "Hello, MeUI",
					OnCreated: func(l *widgets.Label) { l.SetFontSize(28) },
				},
				&flow.Button{
					Text:    "Press me",
					OnClick: func() { fmt.Println("pressed") },
				},
			},
		},
	},
}

root, ids := flow.Build(app) // ids.Get("body") returns the built VStack
```

**QML.** Layout in a file, behaviour wired by id from Go — the model the examples use:

```go
scene, _ := web.Load("hello.qml", "web-light", 720, 480)
scene.OnClick("card", func() { scene.SetText("title", "Pressed.") })
```

All three produce a `meui.Widget` tree, so the headless renderer does not care which one
built it: `web.Shot(root, 720, 480, "out.png")`.

| | layout lives in | checked at | best for |
|---|---|---|---|
| Direct Go | Go calls | compile time | tools, tests, dynamic trees |
| `flow` | Go structs | compile time | declarative Go apps without QML |
| QML | `.qml` file | parse time (`qmlverify`) | designer-readable pages, imported scenes |

## A scene, a page of Go

Save this as `hello.qml`. Note what is *not* there: no `onClicked`, no bindings, no
JavaScript. It is a description of a surface.

```qml
Window {
  width: 720
  height: 480
  color: "#F7F5F0"

  Rectangle {
    anchors.fill: parent
    color: "#F7F5F0"

    Text {
      id: title
      x: 40
      y: 44
      width: 480
      height: 48
      text: "Hello, MeUI"
      font.pixelSize: 28
      font.weight: 700
      color: "#0F1115"
    }

    Rectangle {
      id: card
      x: 40
      y: 112
      width: 300
      height: 120
      radius: 12
      color: "#FFFFFF"
      border.width: 1
      border.color: "#E5E0D6"
      shadow: "card"
      clickable: true

      Text {
        x: 20
        y: 20
        width: 260
        height: 24
        text: "A card is a Rectangle"
        font.pixelSize: 12
        font.weight: 600
        color: "#0F1115"
      }
    }

    Button {
      id: cta
      x: 40
      y: 260
      width: 160
      height: 44
      text: "Say hello"
    }
  }
}
```

The behaviour is the Go file:

```go
package main

import (
	"log"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/web"
	"github.com/sixnineone/meui/widgets"
)

func main() {
	scene, err := web.Load("hello.qml", "web-light", 720, 480)
	if err != nil {
		log.Fatal(err)
	}

	// The scene declared the surface; Go decides what it does.
	scene.Get("cta").(*widgets.Button).SetOnClicked(func() {
		scene.SetText("title", "Hello, Go.")
	})
	scene.OnClick("card", func() {
		scene.SetText("title", "The card was clicked.")
	})

	root := scene.Root.(*meui.Window)
	app := meui.NewApp(sdl3.NewApp(), meui.AppOptions{Title: "Hello"})
	if _, err := app.AddWindow(root, meui.WindowOptions{
		Title: "Hello",
		Size:  graphics.NewSize(720, 480),
	}); err != nil {
		log.Fatal(err)
	}
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
```

`scene.Shot(720, 480, "hello.png")` renders the same tree without opening a window.

## Bundled resources

An application keeps its scenes and assets in `resources/` next to its main package and
embeds them with one directive:

```go
//go:embed all:resources
var appFS embed.FS

bundle := resources.Default()
bundle.Mount("res", appFS, "resources")
scene, _ := web.LoadBundle(bundle, "res://docs.qml", "web-light", 1280, 900)
```

Everything under `resources/` is picked up on every build — adding an asset needs no code
change. Scenes refer to files as `res://assets/cover.png`; the toolkit's own defaults live at
`meui://` (the bundled themes ship in the binary). `resources.MountExternal("res", dir)` adds
a directory at runtime: embedded-first by default, external-first after
`SetOrder(..., ExternalFirst)`. External mounts go through `os.Root`, so a symlink cannot
escape, and `-tags no_embed` builds without the toolkit defaults.

[`examples/docs`](examples/docs) is the reference implementation — its scene and cover image
live in `examples/docs/resources/`, and the example's `-qml` flag still loads a disk file for
design iteration. Fonts are the next piece: the default face is still resolved from disk or
the system until the renderer exposes a bytes loader.

## Window icons

Every MeUI window gets an icon, so no application falls back to the desktop's generic
"X": the toolkit's mark is installed by default. Override it per window:

```go
icon, _ := graphics.NewImageFromFilePathOrURL("assets/app-icon.png", 1)
app.AddWindow(root, meui.WindowOptions{Title: "MyApp", Icon: icon})
```

`NoIcon: true` suppresses it. The icon is set on the OS window (`_NET_WM_ICON` on X11) and
scaled into 128 px, which is the largest size SDL's X11 backend publishes reliably. Under a
Wayland compositor the taskbar or dock may still prefer the application's `.desktop` file:
matching `StartupWMClass` (SDL derives `WM_CLASS` from the binary name) and `Icon=` there is
the complementary step.

## Features

**Layout and surfaces**
- `VStack`, `HStack`, `Grid`, `Flex`, anchors and size policies
- Rounded corners, borders, offset and soft shadows, gradients, clipping
- Opacity, z-order, backdrop blur ("frosted glass"), sticky headers
- Responsive breakpoints and per-breakpoint layout overrides

**Scrolling and data**
- `ScrollView` / `ScrollPane` with wheel, drag and scroll bars
- `VirtualList` — thousands of rows, only the visible widgets exist
- Typed `DataTable` (columns, sorting, selection, column resize), `Tree`, `ListView`
- Sticky sections inside long pages

**Input**
- Click, hover, press, drag, wheel and keyboard focus, wired by id
- `MouseArea`, context menus, tooltips, dialogs, bottom sheets, action sheets

**Forms**
- `TextField`, `ComboBox`, `CheckBox`, `RadioButton`, `Switch`, `Slider`, `SpinBox`
- `ProgressBar`, `DatePicker`, `Calendar`, `ColorPicker`, validation states and error rings

**Chrome**
- `MenuBar`, `StatusBar`, `NavigationRail`, tabs (`TabPanel`), `SplitView`, `DockPanel`
- Desktop-style window chrome with minimize/maximize/close

**Text and graphics**
- Wrapped and elided labels, `MarkdownView`, `MarkdownEditor`, `CodeEditor`
- Images (cover/contain, rounded, clipped), SVG icons, paths and strokes

**Rendering and testing**
- Software device for headless, deterministic rendering
- Batched OpenGL renderer (`mobilegles`) with the same primitive set
- Headless behaviour suite, windowed visual suite, render-parity harness, `imgdiff`

## Gallery

<table>
  <tr>
    <td width="50%"><img src="docs/screens/landing.png" alt="Landing page"><br><sub><b>landing</b> — display type, offset card shadows, pill CTAs</sub></td>
    <td width="50%"><img src="docs/screens/dashboard.png" alt="Dashboard"><br><sub><b>dashboard</b> — gradients, translucent cards, a box chart</sub></td>
  </tr>
  <tr>
    <td width="50%"><img src="docs/screens/docs.png" alt="Docs"><br><sub><b>docs</b> — fixed top bar and sidebar, a 1,880px article, a sticky header</sub></td>
    <td width="50%"><img src="docs/screens/docs-scrolled.png" alt="Docs, scrolled"><br><sub><b>docs</b> — the sticky header after scrolling</sub></td>
  </tr>
  <tr>
    <td width="50%"><img src="docs/screens/mail.png" alt="Mail"><br><sub><b>mail</b> — a virtual list of 240 messages, frosted header</sub></td>
    <td width="50%"><img src="docs/screens/checkout.png" alt="Checkout"><br><sub><b>checkout</b> — a three-step flow with inline validation</sub></td>
  </tr>
  <tr>
    <td width="50%"><img src="docs/screens/settings.png" alt="Settings"><br><sub><b>settings</b> — web-styled fields, switches, a toast</sub></td>
    <td width="50%"><img src="docs/screens/launcher.png" alt="Examples launcher"><br><sub><b>launcher</b> — every example in one window</sub></td>
  </tr>
  <tr>
    <td width="50%"><img src="docs/screens/ops.png" alt="Ops console"><br><sub><b>ops</b> — menubar, tree, typed table, inspector, status bar</sub></td>
    <td width="50%"><img src="docs/screens/admin.png" alt="Admin console"><br><sub><b>admin</b> — dark rail, filters, invite dialog</sub></td>
  </tr>
  <tr>
    <td width="50%"><img src="docs/screens/direct.png" alt="Direct Go example"><br><sub><b>direct</b> — the same page built with widgets and setters</sub></td>
    <td width="50%"><img src="docs/screens/flow.png" alt="Flow example"><br><sub><b>flow</b> — the same page declared as Go structs</sub></td>
  </tr>
  <tr>
    <td colspan="2" align="center"><img src="docs/screens/device.png" alt="Phone-shaped device demo" width="320"><br><sub><b>device</b> — the phone-shaped demo: adaptive pages, sheets, a search view</sub></td>
  </tr>
</table>

## Kitchen sink

The QML enterprise look demo is the stress test for the layout and paint paths: 18 tabs of
widgets, all wired from Go. A few of them:

<table>
  <tr>
    <td width="50%"><img src="docs/screens/enterprise-basic.png" alt="Basic Components tab"><br><sub><b>Basic Components</b> — the first tab: labels, buttons, check boxes, combos, fields, scroll panes and typography</sub></td>
    <td width="50%"><img src="docs/screens/enterprise-calendar.png" alt="Calendar and Color tab"><br><sub><b>Calendar &amp; Color</b> — date and time pickers, colour fields, a calendar and a colour picker</sub></td>
  </tr>
  <tr>
    <td width="50%"><img src="docs/screens/enterprise-datatable.png" alt="Data Table tab"><br><sub><b>Data Table</b> — a typed table with virtual scrolling, selection and sorting</sub></td>
    <td width="50%"><img src="docs/screens/enterprise-code.png" alt="Code Editor tab"><br><sub><b>Code Editor</b> — syntax highlighting, line numbers and folding on this repository's own source</sub></td>
  </tr>
  <tr>
    <td width="50%"><img src="docs/screens/enterprise-mdi.png" alt="MDI Windows tab"><br><sub><b>MDI Windows</b> — draggable, closable child windows inside the page</sub></td>
    <td width="50%"><img src="docs/screens/enterprise-markdown.png" alt="Markdown tab"><br><sub><b>Markdown</b> — a source editor and a live preview, split</sub></td>
  </tr>
</table>

`CGO_ENABLED=0 go run ./examples/enterprise` opens it; `MEUI_TAB=<n>` starts on a tab and
`MEUI_CAPTURE_SHOT=<path>` captures the live window.

## Themes

A preset restyles structure and ink together, not just the palette. The same dashboard with
`-preset web-dark`:

<img src="docs/screens/dashboard-dark.png" alt="The dashboard on the web-dark preset" width="100%">

19 presets ship — `web-light`/`web-dark` for pages, `meui-light`/`windows11-dark` for
consoles, plus macOS, Windows 11, Adwaita, Breeze, metro, iOS, Android and the harness
pair. `cmd/tools/presetgallery` renders the whole set on one sheet.

## Examples

| example | shows |
|---|---|
| [`landing`](examples/landing) | display type, offset card shadows, pill CTAs |
| [`dashboard`](examples/dashboard) | full-bleed gradients, translucent cards, a chart of boxes |
| [`docs`](examples/docs) | fixed chrome, a scrolling article, a sticky header, a breakpoint |
| [`settings`](examples/settings) | web-styled fields, combo, switches, radios, a toast |
| [`mail`](examples/mail) | a virtual list of 240 messages, selection, a frosted header |
| [`checkout`](examples/checkout) | a three-step flow with inline validation and error rings |
| [`ops`](examples/ops) | a desktop console: menubar, tree, typed table, inspector, status bar |
| [`admin`](examples/admin) | a dark console: navigation rail, filters, invite dialog |
| [`direct`](examples/direct) | the same page built with widgets and setters — no QML |
| [`flow`](examples/flow) | the same page declared as Go structs, built by `flow` |
| [`device`](examples/device), [`mobile`](examples/mobile) | phone-shaped pages at device sizes |
| [`enterprise`](examples/enterprise) | the imported QML kitchen sink |
| [`parity`](examples/parity) | the scene the soft and GL devices are diffed on |

Every example takes `-shot`, `-preset`, `-w`, `-h` and `-qml`; [`examples/README.md`](examples/README.md)
documents them all.

## How it fits together

| module | what it is |
|---|---|
| `meui` (this repo) | toolkit: widgets, layouts, QML parser/builder, `web` behaviour layer, theming, examples, harness |
| [`meui-gl`](https://github.com/sixnineone/meui-gl) | SDL3 window/input/platform adapter; hosts a window and runs the frame loop |
| [`mobilegles`](https://github.com/sixnineone/mobilegles) | OpenGL batch renderer for shapes, text and images |

The soft device lives in `meui/graphics`; `mobilegles` is the GL side of the same seam.
`Graphics` primitives, `platform` interfaces and `theme` presets are the contracts between
the three modules.

## Testing

```sh
CGO_ENABLED=0 go test ./...                                  # headless: behaviour, parsing, layout, theming
CGO_ENABLED=0 go test -tags visual ./harness/behaviour/...   # real window, real pixels
make parity-web                                               # soft vs GL on the parity scene
```

CI runs `go build`, `go vet`, `gofmt`, the cgo guard and the headless suites on every push
(see `.github/workflows/ci.yml`). `cmd/tools` holds the instruments: `qmlshot` (headless render), `glshot` (GL render),
`imgdiff` (frame diff), `presetgallery` (every preset in one sheet), `qmlfmt`/`qmlverify`
(scene hygiene), `themetool`, `fontprobe` and the determinism checks.

## Documentation

- [`examples/README.md`](examples/README.md) — running, flags, what each example demonstrates
- [`WEB.md`](WEB.md) — the design record: vocabulary, measured references, renderer notes
- [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md) — bundled fonts, themes and dependency licences
- [`SECURITY.md`](SECURITY.md) — the scene trust model and private vulnerability reporting
- [`resources/`](resources) — the bundle and namespace implementation (`res://`, `meui://`)

## Status

**v0.1.4** — pre-1.0, so the API can still move. Linux desktop is the development target;
the SDL3 adapter and the GL renderer are portable, and the adapter carries the Android JNI
surface. Issues and examples of what you are building are welcome.

## Licence

MIT — see [`LICENSE`](LICENSE). The bundled Inter Display faces are SIL OFL 1.1
([`fonts/OFL.txt`](fonts/OFL.txt)); dependency notices are in
[`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md) and [`LICENSES/`](LICENSES).
