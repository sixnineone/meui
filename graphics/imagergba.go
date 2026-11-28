package graphics

import "fmt"

// The display path for a rendered buffer: sRGB-encoded RGBA8 in, a graphics.Image out.
//
// It exists because the two sides disagree about three things at once, and every caller getting all
// three right by hand is not a reasonable thing to ask:
//
//   - Alpha: the 8-bit world (PNG, JPEG, a document's display buffer) carries STRAIGHT alpha, and the
//     seam's word is PREMULTIPLIED, so each pixel is premultiplied exactly once here.
//   - Order: the seam stores pixels BOTTOM-UP, like the GL readbacks the renderer already feeds it,
//     while a decoded or rendered buffer arrives TOP-DOWN, so the rows are flipped on the way in.
//   - Packing: the seam's word is 0xAARRGGBB.
//
// Getting any one of them wrong produces a picture that looks plausible — a subtly dark image, an
// upside-down one, or red and blue swapped — which is exactly why this is one function rather than a
// paragraph of instructions.
func NewImageFromRGBA(rgba []byte, w, h int) (*Image, error) {
	if err := checkImageDimensions(w, h); err != nil {
		return nil, err
	}
	if int64(len(rgba)) < int64(w)*int64(h)*4 {
		return nil, fmt.Errorf("graphics: %d bytes is too few for %dx%d RGBA (want %d)",
			len(rgba), w, h, int64(w)*int64(h)*4)
	}
	pix := make([]uint32, w*h)
	for y := 0; y < h; y++ {
		// Bottom-up destination, top-down source.
		row := pix[(h-1-y)*w : (h-y)*w]
		src := rgba[y*w*4:]
		for x := 0; x < w; x++ {
			i := x * 4
			a := uint32(src[i+3])
			row[x] = premultiplied(src[i], src[i+1], src[i+2], uint8(a))
		}
	}
	return NewImage(pix, int32(w), int32(h)), nil
}
