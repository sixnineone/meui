//go:build visual

package behaviour

// The resize family, promoted from three probes (resizeprobe, gradresizeprobe,
// wsizeprobe) into one test on the real window path. It asks the OS for a new size — the
// same request a dragged edge makes — and then measures the frame at that size:
//
//   - a FillHeight body must reach the footer at every height (the body follows the
//     window, which is the corruption the probes were built to see);
//   - a vertical gradient must still run black at the body's top to white at its
//     bottom (a fragment position lag would sample the ramp at the OLD height);
//   - a clip band must stay at its body-relative y, not move by the resize delta.
//
// A flat colour cannot reveal any of that, which is why the scene is a gradient plus a
// clip band rather than a solid panel.

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/layout"
)

const (
	resizeBandTop = 100 // body-local, so the band sits 100px below the body's top
	resizeBandH   = 40
)

// gradientBand paints a vertical black→white gradient and a green band clipped 100px
// below its own top. Both are a direct function of the widget's rect, so a stale frame or
// a stale fragment position shows up as the wrong sample.
type gradientBand struct{ meui.BaseWidget }

func (p *gradientBand) Paint(c *graphics.Canvas, r graphics.Rect) {
	grad := graphics.NewPaint()
	grad.SetLinearGradient(
		graphics.NewPoint(r.X, r.Y),
		graphics.NewPoint(r.X, r.Y+r.Height),
		[]graphics.GradientStop{
			{Offset: 0, Color: graphics.RGB(0, 0, 0)},
			{Offset: 1, Color: graphics.RGB(255, 255, 255)},
		})
	c.DrawRect(r, grad)

	green := graphics.NewPaint()
	green.SetColor(graphics.RGB(0, 255, 0))
	c.Save()
	c.ClipRect(graphics.NewRect(r.X, r.Y+resizeBandTop, r.Width, resizeBandH), pathop.Intersect, true)
	c.DrawRect(r, green)
	c.Restore()
}

// resizeTree is the scene plus the three widgets the assertions measure. Frames are read
// from the widgets, not assumed: the layout owns them, and a test that hardcodes the
// header height measures the wrong pixel when the layout changes.
type resizeTree struct {
	root, content  *meui.Window
	header, footer *meui.Panel
	body           *gradientBand
}

// resizeScene is a VStack of header / body (fills) / footer, laid out by the layout
// package — the same shape the probes used.
func resizeScene() *resizeTree {
	root := meui.NewWindow()
	root.SetVisible(true) // a raw BaseWidget (the window included) starts hidden
	content := meui.NewPanel()
	content.SetVisible(true)
	content.SetLayout(layout.NewVStack())
	root.AddChild(content)

	header := meui.NewPanel()
	header.SetVisible(true)
	header.SetBackground(graphics.RGB(0, 0, 255))
	header.SetLayoutProps(&layout.LayoutProps{FillWidth: true, PreferredHeight: 40})
	content.AddChild(header)

	body := &gradientBand{}
	body.SetVisible(true)
	body.SetLayoutProps(&layout.LayoutProps{FillWidth: true, FillHeight: true})
	content.AddChild(body)

	footer := meui.NewPanel()
	footer.SetVisible(true)
	footer.SetBackground(graphics.RGB(0, 255, 0))
	footer.SetLayoutProps(&layout.LayoutProps{FillWidth: true, PreferredHeight: 24})
	content.AddChild(footer)

	return &resizeTree{root: root, header: header, footer: footer, body: body}
}

func isGreen(c graphics.Color) bool {
	return (c>>16)&0xFF < 40 && (c>>8)&0xFF > 200 && c&0xFF < 40
}

// checkResizeFrame asserts the geometry in the frame captured at wantW×wantH, using the
// widgets' own frames for the expected positions.
func checkResizeFrame(va *VisualApp, tree *resizeTree, wantW, wantH int32) {
	if va.lastW != wantW || va.lastH != wantH {
		va.Fatalf("frame is %dx%d after resizing to %dx%d — the resize did not reach the drawable",
			va.lastW, va.lastH, wantW, wantH)
	}
	hf, bf, ff := tree.header.FrameRect(), tree.body.FrameRect(), tree.footer.FrameRect()
	if bf.Y <= hf.Y || bf.Y+bf.Height > ff.Y+1 {
		va.Fatalf("layout after resize: header %v body %v footer %v (body must sit between them)", hf, bf, ff)
	}
	if ff.Y+ff.Height != float32(wantH) {
		va.Fatalf("footer ends at %.0f, want the window height %d — the body did not follow", ff.Y+ff.Height, wantH)
	}

	// Header and footer are where the layout puts them, at ANY height.
	va.AssertPixel(wantW/2, int32(hf.Y+hf.Height/2), graphics.RGB(0, 0, 255), 8, "header")
	va.AssertPixel(wantW/2, int32(ff.Y+ff.Height/2), graphics.RGB(0, 255, 0), 8, "footer (the body followed the window)")

	// Gradient extremes: the top of the body is black and the bottom is white. A
	// fragment lag would sample the ramp at the previous height, so the extremes are
	// the unambiguous check.
	top := va.PixelAt(4, int32(bf.Y)+2)
	if (top>>16)&0xFF > 40 {
		va.T.Errorf("gradient top = %s, want near black", colorString(top))
	}
	bottom := va.PixelAt(4, int32(bf.Y+bf.Height)-3)
	if (bottom>>16)&0xFF < 215 {
		va.T.Errorf("gradient bottom = %s, want near white (a stale fragment position samples mid-ramp)", colorString(bottom))
	}

	// The clip band is body-relative and must NOT move with the window height.
	bandY := int32(bf.Y) + resizeBandTop + resizeBandH/2
	if !isGreen(va.PixelAt(4, bandY)) {
		va.T.Errorf("the green clip band is not at y=%d after the resize (pixel %s)", bandY, colorString(va.PixelAt(4, bandY)))
	}
	if isGreen(va.PixelAt(4, bandY-resizeBandH-6)) {
		va.T.Errorf("green appears above the clip band at y=%d — the band moved", bandY-resizeBandH-6)
	}
	if isGreen(va.PixelAt(4, bandY+resizeBandH)) {
		va.T.Errorf("green appears below the clip band at y=%d — the band moved", bandY+resizeBandH)
	}
}

func TestResizingTheWindowKeepsTheLayoutGradientAndClipAnchored(t *testing.T) {
	tree := resizeScene()
	va := NewVisualAppTree(t, "TestResizingTheWindowKeepsTheLayoutGradientAndClipAnchored", tree.root, 520, 380)

	va.Run(t, []VisualStep{
		func(va *VisualApp) {}, // the first frame at the starting size
		func(va *VisualApp) { checkResizeFrame(va, tree, 520, 380) },

		// Grow vertically: the body must follow, and the gradient must re-reach white.
		func(va *VisualApp) {
			if !va.Wnd().Resize(520, 700) {
				va.Fatalf("the platform refused the resize request")
			}
		},
		va.Frames(3),
		func(va *VisualApp) { checkResizeFrame(va, tree, 520, 700) },

		// Shrink below the starting size.
		func(va *VisualApp) { va.Wnd().Resize(520, 300) },
		va.Frames(3),
		func(va *VisualApp) { checkResizeFrame(va, tree, 520, 300) },

		// And grow horizontally: the body fills the new width too, so a resize that only
		// re-laid-out the height cannot pass.
		func(va *VisualApp) { va.Wnd().Resize(700, 500) },
		va.Frames(3),
		func(va *VisualApp) {
			checkResizeFrame(va, tree, 700, 500)
			if !isGreen(va.PixelAt(696, int32(tree.footer.FrameRect().Y+tree.footer.FrameRect().Height/2))) {
				va.T.Errorf("the footer does not reach the new width: %s",
					colorString(va.PixelAt(696, int32(tree.footer.FrameRect().Y+tree.footer.FrameRect().Height/2))))
			}
		},
	})
}
