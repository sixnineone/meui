# Changelog

All notable changes to MeUI are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions are pre-1.0.

## [0.1.1] — 2026-11-27

### Added
- `examples/direct` and `examples/flow`: the same window built with the imperative API and
  with the declarative Go wrappers, both runnable and screenshot-capable.
- `flow` is documented in the README as the third builder; the README shows all three ways
  side by side.
- CI: build, vet, gofmt, the cgo guard and the headless suites.
- Install section (`go get`), API-docs link, release and CI badges.
- Requirements section: SDL3, OpenGL, X11 and the per-distro install commands; headless
  rendering needs none of them.
- README gallery: landing, launcher, the device demo, the direct/flow pages, and four
  kitchen-sink tabs.
- `CONTRIBUTING.md`, `SECURITY.md`, issue and PR templates.

### Changed
- The README is a real front page: hero screenshot, feature lists, a validated QML + Go
  snippet, module map and testing commands.
- Internal-branding comments and demo copy were made neutral; the About dialog now says
  what the toolkit actually is and uses SDL3, not GLFW.
- The enterprise demo's code-editor tabs read files from this repository instead of
  absolute paths under a private tree.

### Fixed
- Theme files that collided case-insensitively blocked the Go module zip; the Islands
  variants are renamed (`*_islands.theme.json`, `*_alt.theme.json`).

## [0.1.0] — 2026-11-27

Initial public release: the toolkit, QML builder, `flow` wrappers, web layer, widgets,
themes, examples, the behaviour and render-parity harnesses, and the `meui-gl` /
`mobilegles` modules.
