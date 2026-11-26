// qmlshot renders a .qml scene to a PNG through the software device — no display,
// no GL, no window — so a scene can be reviewed and diffed in the same loop as the
// tests. It is the fast half of the screenshot pair; the GL half is the demo's own
// window path. The two are NOT yet cross-checked automatically: the software device
// has its own text and shape tests, and a GL/soft pixel comparison is tracked in
// WEB.md's gap queue.
//
//	go run ./cmd/tools/qmlshot -preset web-light -w 1280 -h 900 -o landing.png page.qml
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sixnineone/meui/web"
)

func main() {
	out := flag.String("o", "", "output PNG (default: <qml-basename>.png)")
	width := flag.Int("w", 1280, "width in logical px")
	height := flag.Int("h", 900, "height in logical px")
	preset := flag.String("preset", "web-light", "theme preset to install before parsing (empty = current)")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: qmlshot [flags] <scene.qml>")
		os.Exit(2)
	}
	path := flag.Arg(0)
	dst := *out
	if dst == "" {
		base := path
		for i := len(base) - 1; i >= 0; i-- {
			if base[i] == '/' || base[i] == '\\' {
				base = base[i+1:]
				break
			}
		}
		for i := len(base) - 1; i >= 0; i-- {
			if base[i] == '.' {
				base = base[:i]
				break
			}
		}
		dst = base + ".png"
	}

	scene, err := web.Load(path, *preset, *width, *height)
	if err != nil {
		fmt.Fprintf(os.Stderr, "qmlshot: %v\n", err)
		os.Exit(1)
	}
	if err := scene.Shot(*width, *height, dst); err != nil {
		fmt.Fprintf(os.Stderr, "qmlshot: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("qmlshot: %s -> %s (%dx%d, preset %q)\n", path, dst, *width, *height, *preset)
}
