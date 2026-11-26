# MeUI

A Go widget toolkit whose scenes are written in QML and whose behaviour is written in
Go. The QML is a **builder**, not a language: no expressions, no bindings, no JavaScript.
Every behaviour — a click, a selection, a scroll, a validation — is attached from Go by id,
which keeps the logic testable, debuggable and in the same language as the rest of the app.

MeUI targets both web-looking pages and desktop applications: the six examples below are
pages, the two consoles are enterprise tools, and the same widgets are used by desktop
applications built on the toolkit.

## Quick start

Requires Go 1.25 and CGO disabled for the desktop dev loop (control reaches C through
purego).

```sh
git clone https://github.com/sixnineone/meui
cd meui
CGO_ENABLED=0 go run ./examples/launcher   # a window listing every example
```

Examples can also be run directly (`go run ./examples/landing`, `./examples/docs`, …) and
every one takes `-shot out.png` to render a frame headless.

## What the examples show

| example | shows |
|---|---|
| `landing` | display type, offset card shadows, pill CTAs, painted chrome |
| `dashboard` | full-bleed gradients, translucent cards, a bar chart of boxes |
| `docs` | fixed chrome around a scrolling article, a sticky header, a breakpoint |
| `settings` | web-styled fields, combo, switches, radios, a toast |
| `mail` | a virtual list of 240 messages, selection, a frosted header |
| `checkout` | a three-step flow with inline validation and error rings |
| `ops` | a desktop console: menubar, tree, typed table, inspector, status bar |
| `admin` | a dark console: navigation rail, filters, invite dialog |

`examples/README.md` documents every flag.

## Layout

- `graphics/` — the drawing seam (software device + GL)
- `widgets/` — the widget library
- `qml/` — the parser and the loader that builds widgets from QML
- `web/` — the small behaviour layer scenes are wired through
- `theme/` — presets, including the `web-light`/`web-dark` web presets
- `platform/` — the OS seam
- `examples/`, `harness/`, `cmd/tools/` — demos, test harness, tools
- `assets/`, `fonts/`, `themes/` — icons, fonts and imported theme samples

The renderer (`mobilegles`) and the SDL3 adapter (`meui-gl`) are separate modules:
[github.com/sixnineone/mobilegles](https://github.com/sixnineone/mobilegles),
[github.com/sixnineone/meui-gl](https://github.com/sixnineone/meui-gl).

## Design record

`WEB.md` is the working design record for the web support: the vocabulary, the measured
references, the renderer notes, and what is deliberately not there.

## Licence

MIT — see `LICENSE`. Bundled fonts keep their own licences: the Inter Display faces in
`fonts/` are SIL OFL 1.1 (`fonts/OFL.txt`). Dependency notices are in
`THIRD_PARTY_NOTICES.md` and `LICENSES/`.
