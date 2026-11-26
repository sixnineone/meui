package graphics

import "testing"

// An image drawn through the soft device must land UPRIGHT. A seam Image is
// bottom-up (Pixels' convention, and what GL readbacks give) while draw2d's
// surface is top-down: without the flip, every image the soft device draws — a
// chart's cached layer, a node's preview, the graph's supersampled cables — landed
// upside down.
func TestSoftDeviceDrawsImagesUpright(t *testing.T) {
	src, sdev := NewSoftwareCanvas(40, 40)
	p := NewPaint()
	p.SetColor(RGB(255, 0, 0))
	// A band near the TOP of the source, in canvas coordinates (y grows down).
	src.DrawRect(NewRect(10, 4, 20, 4), p)
	img := NewImage(sdev.Pixels(), 40, 40)

	dst, ddev := NewSoftwareCanvas(40, 40)
	dst.DrawImageInRect(img, NewRect(0, 0, 40, 40), nil, nil)

	// Pixels() is bottom-up, so row index 0 is the bottom row of the result: the
	// band must be near the TOP, i.e. at a HIGH row index.
	rows := map[int]int{}
	for i, px := range ddev.Pixels() {
		if px&0x00FF0000 != 0 {
			rows[i/40]++
		}
	}
	if len(rows) == 0 {
		t.Fatal("the image drew nothing")
	}
	for row := range rows {
		if row < 20 {
			t.Errorf("red at bottom-up row %d: the image was drawn upside down (rows %v)", row, rows)
			break
		}
	}
}

// A scaled blit (the supersampled link layer draws at 2x and is blitted down)
// must still land, and land in the right place.
func TestSoftDeviceScalesImagesDown(t *testing.T) {
	src, sdev := NewSoftwareCanvas(80, 80)
	p := NewPaint()
	p.SetColor(RGB(0, 255, 0))
	src.DrawRect(NewRect(20, 8, 40, 8), p)
	img := NewImage(sdev.Pixels(), 80, 80)

	dst, ddev := NewSoftwareCanvas(40, 40)
	dst.DrawImageInRect(img, NewRect(0, 0, 40, 40), nil, nil)

	green := 0
	high := 0
	for i, px := range ddev.Pixels() {
		if px&0x0000FF00 != 0 && px&0x00FF0000 == 0 {
			green++
			if i/40 >= 20 {
				high++
			}
		}
	}
	if green == 0 {
		t.Fatal("the 2:1 blit drew nothing")
	}
	if high != green {
		t.Errorf("the downscaled image is not in the top half: %d of %d pixels above the middle", high, green)
	}
}
