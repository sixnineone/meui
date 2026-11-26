package graphics

import (
	"image"
	"image/color"
	"testing"

	"github.com/llgcode/draw2d/draw2dimg"
)

// Probe: which idiom actually rasterises in this draw2d version? The interface
// omits MoveTo/LineTo, but the concrete context may still expose them.
func TestDraw2DIdiomProbe(t *testing.T) {
	// (a) concrete context path commands
	imgA := image.NewRGBA(image.Rect(0, 0, 10, 10))
	gcA := draw2dimg.NewGraphicContext(imgA)
	gcA.SetFillColor(color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	gcA.BeginPath()
	gcA.MoveTo(0, 0)
	gcA.LineTo(10, 0)
	gcA.LineTo(10, 10)
	gcA.LineTo(0, 10)
	gcA.Close()
	gcA.Fill()
	t.Logf("(a) concrete MoveTo/LineTo + Fill -> %v", imgA.RGBAAt(5, 5))

	// (b) path object from GetPath
	imgB := image.NewRGBA(image.Rect(0, 0, 10, 10))
	gcB := draw2dimg.NewGraphicContext(imgB)
	gcB.SetFillColor(color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	gcB.BeginPath()
	p := gcB.GetPath()
	p.MoveTo(0, 0)
	p.LineTo(10, 0)
	p.LineTo(10, 10)
	p.LineTo(0, 10)
	p.Close()
	gcB.Fill()
	t.Logf("(b) GetPath ops + Fill -> %v", imgB.RGBAAt(5, 5))

	if imgA.RGBAAt(5, 5).R != 0xff && imgB.RGBAAt(5, 5).R != 0xff {
		t.Fatalf("neither idiom rasterised")
	}
}
