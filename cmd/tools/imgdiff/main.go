// imgdiff compares two PNG screenshots and reports how far apart they are.
//
// It exists for the enterprise-look guard: capture the main demo before a change and
// after it, and the answer to "did this break the existing look?" is a number plus a
// picture (with -diff) rather than an impression.
//
//	go run ./cmd/tools/imgdiff before.png after.png
//	go run ./cmd/tools/imgdiff -diff diff.png before.png after.png
//
// The numbers are per-pixel: meanΔ is the average of the absolute per-channel
// difference (0-255), changed is the share of pixels that moved at all, and big is the
// share that moved by more than 3 per channel — the one to read, because antialiasing
// and a 1px scroll move a lot of pixels a little.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"os"
	"strings"
)

func main() {
	diffPath := flag.String("diff", "", "write an amplified difference image here")
	ignore := flag.String("ignore", "", "skip a region: x,y,w,h (semicolon-separated for several)")
	flag.Parse()
	var ignoreRects []image.Rectangle
	for _, part := range strings.Split(*ignore, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var x, y, w, h int
		if _, err := fmt.Sscanf(part, "%d,%d,%d,%d", &x, &y, &w, &h); err != nil {
			fmt.Fprintf(os.Stderr, "imgdiff: bad -ignore %q (want x,y,w,h)\n", part)
			os.Exit(2)
		}
		ignoreRects = append(ignoreRects, image.Rect(x, y, x+w, y+h))
	}
	if flag.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "usage: imgdiff [-diff out.png] before.png after.png")
		os.Exit(2)
	}
	before, after := decode(flag.Arg(0)), decode(flag.Arg(1))
	if before.Bounds() != after.Bounds() {
		fmt.Printf("size differs: %v vs %v\n", before.Bounds(), after.Bounds())
		os.Exit(3)
	}

	b := before.Bounds()
	w, h := b.Dx(), b.Dy()
	var sum float64
	var changed, big int
	var out *image.RGBA
	if *diffPath != "" {
		out = image.NewRGBA(image.Rect(0, 0, w, h))
	}
	ignored := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if inRects(ignoreRects, x, y) {
				ignored++
				if out != nil {
					out.Set(x, y, color.RGBA{A: 255})
				}
				continue
			}
			ar, ag, ab, _ := before.At(x, y).RGBA()
			br, bg, bb, _ := after.At(x, y).RGBA()
			dr := absDiff(ar>>8, br>>8)
			dg := absDiff(ag>>8, bg>>8)
			db := absDiff(ab>>8, bb>>8)
			d := dr + dg + db
			sum += float64(d) / 3
			if d > 0 {
				changed++
			}
			if d > 9 {
				big++
			}
			if out != nil {
				// Amplified 8x so a one-level difference is visible.
				out.Set(x, y, color.RGBA{R: amplify(dr), G: amplify(dg), B: amplify(db), A: 255})
			}
		}
	}
	n := float64(w*h - ignored)
	if n <= 0 {
		fmt.Println("nothing to compare: every pixel is ignored")
		return
	}
	fmt.Printf("%dx%d  meanΔ %.3f  changed %.2f%%  big(>3/ch) %.3f%%",
		w, h, sum/n, 100*float64(changed)/n, 100*float64(big)/n)
	if ignored > 0 {
		fmt.Printf("  (ignored %d px)", ignored)
	}
	fmt.Println()

	if out != nil {
		f, err := os.Create(*diffPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "imgdiff: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		if err := png.Encode(f, out); err != nil {
			fmt.Fprintf(os.Stderr, "imgdiff: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s\n", *diffPath)
	}
}

// inRects reports whether the point is inside any ignored region.
func inRects(rects []image.Rectangle, x, y int) bool {
	for _, r := range rects {
		if x >= r.Min.X && x < r.Max.X && y >= r.Min.Y && y < r.Max.Y {
			return true
		}
	}
	return false
}

func amplify(v uint32) uint8 {
	v *= 8
	if v > 255 {
		v = 255
	}
	return uint8(v)
}

func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

func decode(path string) image.Image {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "imgdiff: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "imgdiff: %s: %v\n", path, err)
		os.Exit(1)
	}
	// Normalise to RGBA so At() is cheap and the two sides are comparable.
	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)
	return rgba
}
