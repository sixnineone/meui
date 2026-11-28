# Changelog

All notable changes to MeUI are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions are pre-1.0.

## [0.1.3] — 2026-11-28

### Added
- `resources` package: a bundle of `res://` (application) and `meui://` (toolkit) namespaces,
  with embedded and external layers, an explicit precedence order, merged directory
  listings, and `os.Root` confinement for external mounts. Paths cannot escape a bundle.
- `qml.LoadBundle`, `qml.LoadFS` and `web.LoadBundle`, plus `res://` / `meui://` support in
  `Image`, `CodeEditor`, `MarkdownView` and markdown images.
- The toolkit's themes are embedded (`-tags no_embed` removes them).
- `examples/docs` is the reference bundle: scene and cover image under `resources/`,
  embedded, and no longer dependent on the working directory.

## [0.1.2] — 2026-11-28

### Security
- `OpenURL` now validates before dispatch: no empty URLs, no control characters, and an
  allow-list of `http`/`https`/`mailto`/`tel` (configurable with
  `platform.SetOpenURLSchemes`). Links from untrusted Markdown or rich text can no longer
  hand `file:` or other local handlers to the OS.
- QML source is capped at 8 MiB and 256 nesting levels; the parser and builder are
  recursive, so a malicious scene could previously overflow the stack.
- Image decoding, resampling, RGBA uploads and SVG rasterisation are capped at 64
  megapixels (and 32,768 px per side), with overflow-safe buffer checks.
- File reads reachable from a scene (`CodeEditor.filePath`, `MarkdownView.filePath`,
  `Image` sources, SVG/icon paths) require a regular file and are size-capped, so
  `/dev/zero` or a giant file can no longer stall or exhaust the process.
- Theme imports are capped at 4 MiB; LSP message bodies at 32 MiB.

### Changed
- The README leads with the GPU/speed story and has a "Built for speed" section.

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
- README gallery: landing, launcher, the device demo, the direct/flow pages, a dark
  dashboard and six kitchen-sink tabs.
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
