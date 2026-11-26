package graphics

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"  // decoder registration
	_ "image/jpeg" // decoder registration
	_ "image/png"  // decoder registration
	"math"
	"os"
	"strings"
)

// NewImageFromFilePathOrURL loads an image from a filesystem path.
//
// Remote URLs are deliberately NOT fetched: the drawing seam performs no
// network I/O. Fetch the bytes with whatever HTTP client the app already uses
// and pass them to NewImageFromBytes.
//
// scale resamples the decoded image (bilinear); 0 and 1 mean "as decoded".
func NewImageFromFilePathOrURL(filePathOrURL string, scale float32) (*Image, error) {
	if strings.HasPrefix(filePathOrURL, "http://") || strings.HasPrefix(filePathOrURL, "https://") {
		return nil, fmt.Errorf("graphics: %q is a remote URL; the drawing seam does not fetch over the network — fetch the bytes and use NewImageFromBytes", filePathOrURL)
	}
	data, err := os.ReadFile(filePathOrURL)
	if err != nil {
		return nil, err
	}
	img, err := NewImageFromBytes(data)
	if err != nil {
		return nil, fmt.Errorf("graphics: decode %q: %w", filePathOrURL, err)
	}
	return img.Scaled(scale), nil
}

// NewImageFromBytes decodes PNG, JPEG or GIF bytes into an image.
//
// The result is premultiplied, which is the seam's image convention: an Image
// is handed to the backend as premultiplied pixels (offscreen readbacks already
// are, and decoded straight-alpha data is premultiplied here so both kinds
// composite identically).
func NewImageFromBytes(data []byte) (*Image, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("graphics: decoded image has empty bounds")
	}
	pix := make([]uint32, w*h)
	for y := 0; y < h; y++ {
		// The seam stores pixels bottom-up, like the GL readbacks the renderer
		// already feeds into NewImage; decoded rows arrive top-down, so flip
		// them into the buffer on the way in.
		row := pix[(h-1-y)*w : (h-y)*w]
		for x := 0; x < w; x++ {
			// Convert through NRGBAModel: Color.RGBA() returns PREMULTIPLIED
			// channels, so reading it directly would premultiply twice.
			c := color.NRGBAModel.Convert(src.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			row[x] = premultiplied(c.R, c.G, c.B, c.A)
		}
	}
	return NewImage(pix, int32(w), int32(h)), nil
}

// premultiplied packs straight-alpha RGB into the 0xAARRGGBB premultiplied
// word the seam and the renderer use.
func premultiplied(r, g, b, a uint8) uint32 {
	pr := uint32(r) * uint32(a) / 255
	pg := uint32(g) * uint32(a) / 255
	pb := uint32(b) * uint32(a) / 255
	return uint32(a)<<24 | pr<<16 | pg<<8 | pb
}

// Scaled returns a bilinear resample of the image. A scale of 0 or 1 returns
// the receiver unchanged. Interpolating premultiplied pixels is the correct
// way to resample alpha-carrying content.
func (i *Image) Scaled(scale float32) *Image {
	if i == nil || scale <= 0 || scale == 1 || i.w <= 0 || i.h <= 0 {
		return i
	}
	sw, sh := int(i.w), int(i.h)
	nw := int(math.Round(float64(float32(sw) * scale)))
	nh := int(math.Round(float64(float32(sh) * scale)))
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	out := make([]uint32, nw*nh)
	for y := 0; y < nh; y++ {
		// Sample at pixel centres, clamped to the source.
		sy := (float64(y)+0.5)/float64(nh)*float64(sh) - 0.5
		y0 := int(math.Floor(sy))
		fy := float32(sy - float64(y0))
		if y0 < 0 {
			y0, fy = 0, 0
		}
		y1 := y0 + 1
		if y1 > sh-1 {
			y1 = sh - 1
		}
		for x := 0; x < nw; x++ {
			sx := (float64(x)+0.5)/float64(nw)*float64(sw) - 0.5
			x0 := int(math.Floor(sx))
			fx := float32(sx - float64(x0))
			if x0 < 0 {
				x0, fx = 0, 0
			}
			x1 := x0 + 1
			if x1 > sw-1 {
				x1 = sw - 1
			}
			out[y*nw+x] = bilerp(
				i.pix[y0*sw+x0], i.pix[y0*sw+x1],
				i.pix[y1*sw+x0], i.pix[y1*sw+x1], fx, fy)
		}
	}
	return NewImage(out, int32(nw), int32(nh))
}

// bilerp blends four 0xAARRGGBB pixels. Channels are independent, so this is
// valid on premultiplied data.
func bilerp(c00, c10, c01, c11 uint32, fx, fy float32) uint32 {
	ch := func(shift uint) uint32 {
		a := float32((c00>>shift)&0xFF)*(1-fx)*(1-fy) +
			float32((c10>>shift)&0xFF)*fx*(1-fy) +
			float32((c01>>shift)&0xFF)*(1-fx)*fy +
			float32((c11>>shift)&0xFF)*fx*fy
		if a < 0 {
			a = 0
		}
		if a > 255 {
			a = 255
		}
		return uint32(a + 0.5)
	}
	return ch(24)<<24 | ch(16)<<16 | ch(8)<<8 | ch(0)
}
