// Command determinism captures the same QML scene through the demo
// binary N times and verifies the framebuffers are pixel-identical.
//
// A static scene must render the same bytes every run. Theme-
// application races, stale paint caches and nondeterministic
// compositing have all surfaced as run-to-run pixel differences —
// this test fails on any difference beyond -tol per channel.
//
// Usage:
//
//	go run ./cmd/tools/determinism
//	go run ./cmd/tools/determinism -n 5 -secs 5 -qml examples/enterprise/main.qml
//
// Requires a display (the scene is rendered by the real window stack).
package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func main() {
	n := flag.Int("n", 3, "number of captures to compare")
	secs := flag.Int("secs", 4, "MEUI_CAPTURE_SECS per run (settle time before capture)")
	tol := flag.Int("tol", 0, "max allowed per-channel delta (0 = strict)")
	bin := flag.String("bin", "", "prebuilt demo binary (default: build to a temp dir)")
	qml := flag.String("qml", "examples/enterprise/main.qml", "QML scene to render")
	flag.Parse()

	if *n < 2 {
		fmt.Fprintln(os.Stderr, "determinism: -n must be >= 2")
		os.Exit(2)
	}
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		fmt.Fprintln(os.Stderr, "determinism: no DISPLAY/WAYLAND_DISPLAY — needs a live display")
		os.Exit(2)
	}

	tmp, err := os.MkdirTemp("", "meui-determinism-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "determinism:", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	demo := *bin
	if demo == "" {
		demo = filepath.Join(tmp, "demo-app")
		fmt.Printf("building demo binary -> %s\n", demo)
		build := exec.Command("go", "build", "-o", demo, "./examples/enterprise")
		build.Stdout, build.Stderr = os.Stdout, os.Stderr
		if err := build.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "determinism: build failed:", err)
			os.Exit(1)
		}
	}

	// Park the pointer in a screen corner before each run: the WM
	// places the window at varying positions, so a parked pointer can
	// hover a different widget per run (spurious pixel diffs).
	parkPointer := func() {
		if _, err := exec.LookPath("xdotool"); err == nil {
			_ = exec.Command("xdotool", "mousemove", "0", "0").Run()
		}
	}

	shots := make([]string, *n)
	for i := 0; i < *n; i++ {
		parkPointer()
		shots[i] = filepath.Join(tmp, fmt.Sprintf("cap%d.png", i))
		cmd := exec.Command(demo, *qml)
		cmd.Env = append(os.Environ(),
			"MEUI_CAPTURE_SHOT="+shots[i],
			fmt.Sprintf("MEUI_CAPTURE_SECS=%d", *secs),
		)
		start := time.Now()
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "determinism: run %d failed: %v\n%s\n", i+1, err, out)
			os.Exit(1)
		}
		if _, err := os.Stat(shots[i]); err != nil {
			fmt.Fprintf(os.Stderr, "determinism: run %d produced no capture (%v)\n%s\n", i+1, err, out)
			os.Exit(1)
		}
		fmt.Printf("run %d/%d: captured %s in %v\n", i+1, *n, filepath.Base(shots[i]), time.Since(start).Round(time.Millisecond))
	}

	base, err := loadPNG(shots[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "determinism:", err)
		os.Exit(1)
	}

	failed := false
	for i := 1; i < *n; i++ {
		cmp, err := loadPNG(shots[i])
		if err != nil {
			fmt.Fprintln(os.Stderr, "determinism:", err)
			os.Exit(1)
		}
		diff, maxD, bbox, err := compare(base, cmp, *tol)
		if err != nil {
			fmt.Fprintf(os.Stderr, "determinism: capture %d: %v\n", i, err)
			os.Exit(1)
		}
		if diff == 0 {
			fmt.Printf("compare run1 vs run%d: identical (maxDelta=%d)\n", i+1, maxD)
			continue
		}
		failed = true
		fmt.Printf("compare run1 vs run%d: MISMATCH pixels=%d maxDelta=%d bbox=%v\n", i+1, diff, maxD, bbox)
	}

	if failed {
		fmt.Println("FAIL: captures differ — theme/cache/startup ordering or nondeterministic rendering")
		os.Exit(1)
	}
	fmt.Printf("PASS: %d captures identical (tol=%d)\n", *n, *tol)
}

func loadPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

// compare returns the number of pixels differing by more than tol, the
// maximum per-channel delta, and the bounding box of the differences
// (nil when identical).
func compare(a, b image.Image, tol int) (int, int, image.Rectangle, error) {
	ab, bb := a.Bounds(), b.Bounds()
	if ab != bb {
		return 0, 0, image.Rectangle{}, fmt.Errorf("size mismatch %v vs %v", ab, bb)
	}
	diff, maxD := 0, 0
	empty := true
	var box image.Rectangle
	for y := ab.Min.Y; y < ab.Max.Y; y++ {
		for x := ab.Min.X; x < ab.Max.X; x++ {
			r1, g1, b1, a1 := a.At(x, y).RGBA()
			r2, g2, b2, a2 := b.At(x, y).RGBA()
			d := max4(abs(int(r1>>8)-int(r2>>8)), abs(int(g1>>8)-int(g2>>8)), abs(int(b1>>8)-int(b2>>8)), abs(int(a1>>8)-int(a2>>8)))
			if d > maxD {
				maxD = d
			}
			if d > tol {
				diff++
				if empty {
					box = image.Rect(x, y, x+1, y+1)
					empty = false
				} else {
					box = box.Union(image.Rect(x, y, x+1, y+1))
				}
			}
		}
	}
	return diff, maxD, box, nil
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func max4(a, b, c, d int) int {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	if d > m {
		m = d
	}
	return m
}
