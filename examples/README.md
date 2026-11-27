# MEUI examples

Thirteen applications built with **MeUI** — the widget toolkit in this repository — plus
the launcher and the parity scene its renderer is checked against.

They exist to show the toolkit as an application platform rather than a widget list:
six web-looking pages, two enterprise desktop consoles, a phone-shaped device demo, the
imported QML kitchen sink, and two small pages built with the Go builders. Between them: rectangles that round, lift and clip,
scrolling and sticky headers, lists of a thousand rows, forms that validate, frosted
glass, a menubar, a tree, a `DataTable` driving an inspector, a navigation rail, a dark
theme and an invite dialog.

Everything is wired from Go; QML here is a builder, not a language (no expressions, no
bindings, no JavaScript), which is the toolkit's design.

Start here:

```sh
cd meui                      # the module root; every command below runs from it
CGO_ENABLED=0 go run ./examples/launcher
```

The launcher lists everything with a one-line "what it shows"; clicking one opens it in
its own window and its own process (the toolkit owns one app loop per process).

## Running one example

```sh
CGO_ENABLED=0 go run ./examples/landing     # marketing page
CGO_ENABLED=0 go run ./examples/dashboard   # weather dashboard
CGO_ENABLED=0 go run ./examples/docs        # documentation site
CGO_ENABLED=0 go run ./examples/settings    # settings form
CGO_ENABLED=0 go run ./examples/mail        # inbox, 240 messages
CGO_ENABLED=0 go run ./examples/checkout    # three-step checkout
CGO_ENABLED=0 go run ./examples/ops         # enterprise ops console (light desktop theme)
CGO_ENABLED=0 go run ./examples/admin       # enterprise team admin (dark desktop theme)
CGO_ENABLED=0 go run ./examples/device      # phone-shaped device demo
CGO_ENABLED=0 go run ./examples/enterprise  # the imported QML kitchen sink
CGO_ENABLED=0 go run ./examples/direct      # the same page built with widgets and setters
CGO_ENABLED=0 go run ./examples/flow        # the same page declared as Go structs
```

`examples/mobile` is a raw mobile demo (it drives the GL window directly) and expects a
device or a GL window; the device demo above is the one to run on a desktop.

`CGO_ENABLED=0` is the desktop dev loop (purego, no cgo). With `make` installed,
`make examples` runs the launcher and `make run` runs the enterprise demo.

## Headless screenshots

Every example takes `-shot PNG`, which renders one frame and exits — no display needed.
All share `-w`, `-h` and `-preset` (`web-light` for the pages; the consoles default to
their own: `meui-light` and `windows11-dark`).

```sh
CGO_ENABLED=0 go run ./examples/landing   -shot /tmp/landing.png
CGO_ENABLED=0 go run ./examples/dashboard -shot /tmp/dashboard.png -preset web-dark
CGO_ENABLED=0 go run ./examples/settings  -shot /tmp/settings.png
CGO_ENABLED=0 go run ./examples/mail      -shot /tmp/mail.png
CGO_ENABLED=0 go run ./examples/ops       -shot /tmp/ops.png
CGO_ENABLED=0 go run ./examples/admin     -shot /tmp/admin.png
CGO_ENABLED=0 go run ./examples/admin     -invite -shot /tmp/admin-invite.png
CGO_ENABLED=0 go run ./examples/device    -page 0 -shot /tmp/device.png
CGO_ENABLED=0 go run ./examples/direct    -shot /tmp/direct.png
CGO_ENABLED=0 go run ./examples/flow      -shot /tmp/flow.png
```

Two pages have state worth capturing:

```sh
# docs: the article mid-scroll, with its sticky section header pinned
CGO_ENABLED=0 go run ./examples/docs -scroll 760 -shot /tmp/docs-scrolled.png
# docs: below its 1000px breakpoint, where the sidebar goes away
CGO_ENABLED=0 go run ./examples/docs -w 860 -h 720 -shot /tmp/docs-compact.png

# checkout: the validation state, the review and the success panel
CGO_ENABLED=0 go run ./examples/checkout -invalid -shot /tmp/checkout-error.png
CGO_ENABLED=0 go run ./examples/checkout -fill -step 3 -shot /tmp/checkout-review.png
CGO_ENABLED=0 go run ./examples/checkout -fill -step 4 -shot /tmp/checkout-done.png
```

The enterprise demo screenshots itself through the environment (it opens a real window):

```sh
cd examples/enterprise
MEUI_CAPTURE_SHOT=/tmp/enterprise.png MEUI_CAPTURE_SECS=4 go run .
```

## What each one shows

| example | the scene | what to look at |
|---|---|---|
| `landing` | a marketing page for a fictional product | display type at 76px cap height, a card with a hard offset shadow, pill buttons, a painted play glyph, a clipped product shot |
| `dashboard` | a weather dashboard over a sky gradient | full-bleed gradients, translucent cards over it, a bar chart made of boxes, a day strip where the chosen day is a **selected** box restyled from Go |
| `docs` | a documentation site | fixed top bar and sidebar around a **scrolling article**, a **sticky** section header, a cover image with rounded corners, a **breakpoint** at 1000px that drops the sidebar |
| `settings` | an account settings form | web-styled `TextField`, `ComboBox`, `Switch`, `CheckBox`, `RadioButton`, a primary `Button`, a save bar and a toast whose visibility Go flips |
| `mail` | a three-column inbox | a **virtual list** (240 messages, ~12 widgets alive), selection and reveal, a **frosted** header the rows scroll under, an app-supplied serif in the reading pane |
| `checkout` | a three-step checkout | **validation**: an invalid field blocks the step, shows an inline error, takes the keyboard and rings its focus in the error ink; a step indicator; a success state |
| `ops` | an operations console (desktop `meui-light`) | a `MenuBar`, a service **tree** filtering a typed `DataTable`, an inspector filled from the selected row, an activity strip, a `StatusBar`; “New run” queues a row and completes it on a timer |
| `admin` | a team admin console (desktop `windows11-dark`) | a `NavigationRail`, live search + role filter composing into one filter, a member `DataTable`, a detail pane with a `Switch`, and an **invite dialog** overlay with validation and a duplicate check |
| `device` | a phone-shaped demo | adaptive pages (widgets, device report, render, perf), bottom and action sheets, an adaptive dialog, a search view, swipe rows and pull-to-refresh — the demo used on real devices |
| `direct` | a small page built imperatively | `meui.NewWindow`, `meui.NewPanel`, `layout.NewVStack` and widget setters — no QML, no flow |
| `flow` | the same page declared in Go | the `flow` wrappers (`Window`, `VStack`, `Label`, `Button`) and `flow.Build`; no file to parse |
| `enterprise` | the imported QML kitchen sink | the original demo application: tabs of controls, tables, editors, dialogs, its own brand theme |
| `parity` | no window | the GL-vs-offscreen fixture: surfaces only (cards, gradient, translucency, clip, rounded image, z pair, frosted glass) |

## Presets

The examples' looks come from theme presets: `web-light`/`web-dark` for the six pages,
`meui-light` for the light console and `windows11-dark` for the dark one. The toolkit ships
19 presets (`theme.PresetIDs()`), from platform-shaped ones (Adwaita, Breeze, macOS,
Windows 11) to app ones; a preset decides every control's structure, inks and motion.

## The parity scene

`examples/parity/parity.qml` is not an app: it is the scene the two renderers are compared
on. Text is excluded on purpose (two rasterizers never agree byte-for-byte).

```sh
CGO_ENABLED=0 go run ./cmd/tools/qmlshot -preset web-light -w 800 -h 600 -o /tmp/soft.png examples/parity/parity.qml
CGO_ENABLED=0 go run ./cmd/tools/glshot  -preset web-light -w 800 -h 600 -o /tmp/gl.png   examples/parity/parity.qml
CGO_ENABLED=0 go run ./cmd/tools/imgdiff /tmp/soft.png /tmp/gl.png
```

With `make` installed, `make parity-web` runs those three steps. `cmd/tools/imgdiff` also takes
`-diff out.png` for an amplified difference image and `-ignore x,y,w,h` to mask a region.

## How these are built

- `web.Load(path, preset, w, h)` parses the QML with a theme preset installed.
- `scene.OnClick(id, func())`, `OnHover`, `OnPaint`, `SetText`, `Select`, `SetFrame`,
  `Show`, `ScrollTo`, `Breakpoints` wire behaviour **by id, from Go**. Native controls are
  wired through their own setters (`Button.SetOnClicked`, `TextField.OnTextChanged`,
  `ComboBox.OnSelected`, `NavigationRail.OnSelectionChanged`) — `OnClick` is for boxes that
  opted into `clickable`.
- The consoles are built on the desktop widgets: `widgets.DataTable` (typed columns) rather
  than the older `widgets.Table`, `widgets.Tree`, `widgets.MenuBar`, `widgets.StatusBar` and
  the `widgets.NavigationRail`.
- The device demo shows the split every host uses: `examples/device/demoapp` builds and
  wires the scene; the host in `examples/device` supplies the app and the window (and did so
  on Android before it moved here).

`../WEB.md` is the design record for the web support: the vocabulary, the measured
references, the renderer notes, and what is deliberately not there.
