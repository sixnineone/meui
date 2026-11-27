# Contributing

Thanks for looking. This is a small toolkit with a strong shape; the notes below are what
keeps it that way.

## Build and test

```sh
CGO_ENABLED=0 go build ./...
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 go test -tags visual ./harness/behaviour/...   # needs a display and a GL driver
make check                                                   # build + tests + the cgo guard
make fmt-check
```

The desktop dev loop is `CGO_ENABLED=0` on purpose: the toolkit reaches SDL3, OpenGL and the
system through purego. A change that introduces `import "C"` in a desktop file, or
`runtime/cgo` in the `CGO_ENABLED=0` dependency graph, will fail `make check-nocgo`.

## What a good change looks like

- A bug fix comes with a test that fails before it and passes after. The headless harness
  (`harness/behaviour`) is the right place for layout, hit-testing and painting; the
  windowed suite (`-tags visual`) is for real pixels.
- A new widget gets a QML wrapper, a flow wrapper if the common ones cover it, and at
  least one behaviour test.
- A new example goes in `examples/`, is added to the launcher's table, to
  `examples/README.md`, and (if it is a looker) to the README gallery. Every example takes
  `-shot`, `-preset`, `-w` and `-h`.
- Scene changes stay declarations: QML has no expressions, bindings or JavaScript.
  `cmd/tools/qmlverify` and `cmd/tools/qmlfmt` keep scenes honest and formatted.
- Comments explain *why*, not what. Screen recordings and screenshots in a PR help.

## Before you open a PR

```sh
gofmt -w $(find . -name '*.go' -not -path './.git/*')
CGO_ENABLED=0 go vet ./...
make check
```

Keep the diff focused. If a change touches the renderer seam (soft and GL devices), run
`make parity-web` and say what the diff was.

## Reporting bugs

Open an issue with the example (or the smallest QML/Go snippet) that shows it, the Go
version, the distro and the SDL3 version. For anything security-sensitive, use
`SECURITY.md` instead of a public issue.
