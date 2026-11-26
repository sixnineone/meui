package graphics

import (
	"image"
	"image/color"
	"testing"

	"github.com/llgcode/draw2d/draw2dimg"
)

// Probe: can this module actually import and use draw2d with GOPROXY=off?
func TestDraw2DResolvesInMeui(t *testing.T) {
	dest := image.NewRGBA(image.Rect(0, 0, 8, 8))
	gc := draw2dimg.NewGraphicContext(dest)
	gc.SetFillColor(color.RGBA{R: 0xff, A: 0xff})
	gc.MoveTo(0, 0)
	gc.LineTo(8, 0)
	gc.LineTo(8, 8)
	gc.LineTo(0, 8)
	gc.Close()
	gc.Fill()
	if dest.RGBAAt(4, 4).R != 0xff {
		t.Fatalf("draw2d did not rasterise: %v", dest.RGBAAt(4, 4))
	}
}
