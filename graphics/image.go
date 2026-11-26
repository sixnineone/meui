package graphics

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/sixnineone/mobilegles"
)

// NewImageFromDrawing renders draw into an offscreen surface belonging to dev
// and returns the result. The backend owns offscreen creation, so the seam
// never reaches for a renderer itself.
func NewImageFromDrawing(dev Device, width, height, ppi int, draw func(*Canvas)) (*Image, error) {
	// MEUI_NO_OFFSCREEN=1: refuse to render offscreen at all, so callers take
	// their direct-draw path. A DIAGNOSTIC, not a requirement — nothing needs it,
	// and the offscreen pass's state separation is now asserted restored by
	// mobilegles.TestOffscreenPassSeparatesItsState (REVIEW.md §4). It is kept
	// because it is the one switch that flips between the cached and direct paths
	// when isolating a rendering fault, and it costs nothing unset.
	//
	// Setting it is not free: SVG icons fall back from the 4x-supersampled raster
	// to a direct vector draw (14 icons on the parity corpus's main.qml).
	if os.Getenv("MEUI_NO_OFFSCREEN") == "1" {
		return nil, errOffscreenFailed
	}
	if dev == nil {
		return nil, errOffscreenFailed
	}
	img, err := dev.RenderOffscreen(width, height, draw)
	if err != nil || img == nil {
		return nil, errOffscreenFailed
	}
	pixels := img.Pixels()
	// MEUI_OFFSCREEN_DUMP=<dir>: write every offscreen image to a PNG as it is
	// produced. The chart caches its static layer as one of these, so this is
	// how the cached layer itself gets inspected rather than the window that
	// eventually draws it.
	if dir := os.Getenv("MEUI_OFFSCREEN_DUMP"); dir != "" {
		dumpOffscreenPNG(dir, width, height, pixels)
	}
	if os.Getenv("MEUI_FLUSH_DEBUG") != "" {
		nonZero := 0
		for _, px := range pixels {
			if px != 0 {
				nonZero++
			}
		}
		var first uint32
		if len(pixels) > 0 {
			first = pixels[0]
		}
		fmt.Printf("[GetCached] offscreen %dx%d nonZero=%d/%d first=%08x\n",
			width, height, nonZero, len(pixels), first)
	}
	if dir := os.Getenv("MEUI_DUMP_CACHE"); dir != "" {
		dumpCache(dir, pixels, width, height)
	}
	return img, nil
}

var cacheDumpSeq int32

// dumpCache writes the raw offscreen buffer (0xAARRGGBB, bottom-up) as a PNG,
// composited over mid-gray so alpha-only content is visible.
func dumpCache(dir string, pixels []uint32, width, height int) {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		row := pixels[(height-1-y)*width : (height-y)*width] // GL bottom-up; flip
		for x, px := range row {
			o := img.Pix[y*img.Stride+x*4:]
			compositeOverGray(o, px)
		}
	}
	os.MkdirAll(dir, 0o755)
	n := atomic.AddInt32(&cacheDumpSeq, 1)
	f, err := os.Create(fmt.Sprintf("%s/cache%02d_%dx%d.png", dir, n, width, height))
	if err == nil {
		png.Encode(f, img)
		f.Close()
	}
}

// compositeOverGray writes one 0xAARRGGBB pixel as NRGBA over mid-gray.
func compositeOverGray(dst []byte, px uint32) {
	a := byte((px >> 24) & 0xFF)
	r := byte((px >> 16) & 0xFF)
	g := byte((px >> 8) & 0xFF)
	b := byte(px & 0xFF)
	dst[0] = byte((int(r)*int(a) + 128*(255-int(a))) / 255)
	dst[1] = byte((int(g)*int(a) + 128*(255-int(a))) / 255)
	dst[2] = byte((int(b)*int(a) + 128*(255-int(a))) / 255)
	dst[3] = 255
}

// NewImageFromFilePathOrURL lives in imageload.go.

var errOffscreenFailed = mobilegles.ErrOffscreenFailed()

// CaptureOffscreen renders draw into an offscreen surface on dev and hands the
// pixels to done as a composited NRGBA image. A nil device or a backend that
// cannot render offscreen simply produces nothing, matching the previous
// behaviour when the renderer had no context.
func CaptureOffscreen(dev Device, width, height int, draw func(*Canvas), done func(*image.NRGBA)) {
	if dev == nil || done == nil {
		return
	}
	img, err := dev.RenderOffscreen(width, height, draw)
	if err != nil || img == nil {
		return
	}
	pixels := img.Pixels()
	out := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		row := pixels[(height-1-y)*width : (height-y)*width] // GL bottom-up; flip
		for x, px := range row {
			compositeOverGray(out.Pix[y*out.Stride+x*4:], px)
		}
	}
	done(out)
}

// dumpOffscreenPNG writes one offscreen image to dir with a sequence number.
// Pixels are packed a<<24|r<<16|g<<8|b and come back bottom-up, so rows are
// flipped to match the screen.
var offscreenDumpSeq int

func dumpOffscreenPNG(dir string, w, h int, pixels []uint32) {
	if w <= 0 || h <= 0 || len(pixels) < w*h {
		return
	}
	offscreenDumpSeq++
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		row := pixels[(h-1-y)*w : (h-y)*w]
		for x, v := range row {
			o := img.PixOffset(x, y)
			img.Pix[o+0] = byte(v >> 16)
			img.Pix[o+1] = byte(v >> 8)
			img.Pix[o+2] = byte(v)
			img.Pix[o+3] = 0xFF
		}
	}
	os.MkdirAll(dir, 0o755)
	f, err := os.Create(filepath.Join(dir, fmt.Sprintf("offscreen_%04d_%dx%d.png", offscreenDumpSeq, w, h)))
	if err != nil {
		return
	}
	defer f.Close()
	png.Encode(f, img)
}
