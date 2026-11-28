# Security policy

## Supported versions

The latest `v0.1.x` tag receives fixes. The project is pre-1.0, so no older series is
maintained.

## Reporting a vulnerability

Please do **not** open a public issue. Use GitHub's private vulnerability reporting:
<https://github.com/sixnineone/meui/security/advisories/new>.

Include the version, a minimal reproduction, and the impact you see. This is a UI toolkit:
the highest-risk areas are the QML parser and id resolver (untrusted scene files) and the
platform bindings through purego. Reports are acknowledged as soon as possible; there is no
bug bounty.

## Trust model

**A scene is trusted code.** Do not load `.qml` files, theme directories or font files from
someone you would not let run a program. The builder can read any file the process can read:
`filePath` on `CodeEditor`, `MarkdownView` and `Image` is a real path, not a sandboxed asset
reference. Treat bundled scenes and themes as part of your application.

The toolkit does bound the damage a malformed scene or file can do:

- QML source is capped at 8 MiB and 256 nesting levels (the parser and builder are
  recursive).
- Decoded images are capped at 64 megapixels (and 32,768 px per side); SVG rasterisation is
  capped at the same pixel budget.
- Scene-settable file reads require a **regular file** (so `/dev/zero` cannot hang the app)
  and are size-capped: 16 MiB for editor/markdown documents, 8 MiB for scenes and SVG, 4 MiB
  for theme files, 256 MiB for image files before decoding.
- `OpenURL` accepts only `http`, `https`, `mailto` and `tel` by default and rejects control
  characters; `platform.SetOpenURLSchemes` is the explicit opt-out for a custom scheme.
- LSP message bodies are capped at 32 MiB; the language-server path is supplied by the app
  and is trusted.

If you need to render scenes from an untrusted source, run them in a separate process with
the least privilege that process can have; the toolkit does not provide a sandbox.
