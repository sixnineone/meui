// Command presetgallery renders a QML scene under every native theme
// preset and writes one PNG per preset for visual review.
//
// Usage:
//
//	go run ./cmd/tools/presetgallery            # all presets -> preset-gallery/
//	go run ./cmd/tools/presetgallery -o /tmp/g  # custom output dir
//
// Requires a display; each preset runs the real demo window stack and
// captures the framebuffer (MEUI_CAPTURE_SHOT).
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/sixnineone/meui/theme"
)

func main() {
	outDir := flag.String("o", "preset-gallery", "output directory for the PNGs")
	secs := flag.Int("secs", 4, "settle time before each capture")
	bin := flag.String("bin", "", "prebuilt demo binary (default: build to a temp dir)")
	qml := flag.String("qml", "examples/enterprise/main.qml", "QML scene to render")
	flag.Parse()

	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		fmt.Fprintln(os.Stderr, "presetgallery: no DISPLAY/WAYLAND_DISPLAY — needs a live display")
		os.Exit(2)
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "presetgallery:", err)
		os.Exit(1)
	}

	tmp, err := os.MkdirTemp("", "meui-gallery-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "presetgallery:", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	demo := *bin
	if demo == "" {
		demo = filepath.Join(tmp, "demo-app")
		fmt.Println("building demo binary...")
		build := exec.Command("go", "build", "-o", demo, "./examples/enterprise")
		build.Stdout, build.Stderr = os.Stdout, os.Stderr
		if err := build.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "presetgallery: build failed:", err)
			os.Exit(1)
		}
	}

	ids := theme.PresetIDs()
	for _, id := range ids {
		shot := filepath.Join(*outDir, id+".png")
		cmd := exec.Command(demo, *qml)
		cmd.Env = append(os.Environ(),
			"MEUI_THEME="+id,
			"MEUI_CAPTURE_SHOT="+shot,
			fmt.Sprintf("MEUI_CAPTURE_SECS=%d", *secs),
		)
		start := time.Now()
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "presetgallery: %s failed: %v\n%s\n", id, err, out)
			os.Exit(1)
		}
		if _, err := os.Stat(shot); err != nil {
			fmt.Fprintf(os.Stderr, "presetgallery: %s produced no capture (%v)\n%s\n", id, err, out)
			os.Exit(1)
		}
		fmt.Printf("%-16s -> %s (%v)\n", id, shot, time.Since(start).Round(time.Millisecond))
	}
	fmt.Printf("gallery: %d presets in %s/\n", len(ids), *outDir)
}
