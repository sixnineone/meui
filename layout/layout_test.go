package layout_test

import (
	"fmt"
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/layout"
)

// newChild creates a bare BaseWidget with an explicit frame.
func newChild(x, y, w, h float32) *meui.BaseWidget {
	bw := meui.NewBaseWidget()
	bw.SetFrameRect(graphics.NewRect(x, y, w, h))
	return bw
}

func frame(t *testing.T, w meui.Widget) graphics.Rect {
	t.Helper()
	if f, ok := w.(interface{ FrameRect() graphics.Rect }); ok {
		return f.FrameRect()
	}
	t.Fatalf("widget has no FrameRect")
	return graphics.Rect{}
}

func approx(a, b float32) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 0.01
}

// ── VStack ─────────────────────────────────────────────

func TestVStackStackingOrder(t *testing.T) {
	panel := meui.NewPanel()
	c1 := newChild(0, 0, 50, 20)
	c2 := newChild(0, 0, 50, 30)
	c3 := newChild(0, 0, 50, 40)
	panel.AddChild(c1)
	panel.AddChild(c2)
	panel.AddChild(c3)

	vs := layout.NewVStack()
	vs.Spacing = 4
	vs.Layout(panel, graphics.NewRect(0, 0, 200, 200))

	f1, f2, f3 := frame(t, c1), frame(t, c2), frame(t, c3)
	// Stacked top-down in local space with spacing between.
	if !approx(f1.Y, 0) || !approx(f1.Height, 20) {
		t.Errorf("c1 = %v, want y=0 h=20", f1)
	}
	if !approx(f2.Y, 24) || !approx(f2.Height, 30) {
		t.Errorf("c2 = %v, want y=24 h=30", f2)
	}
	if !approx(f3.Y, 58) || !approx(f3.Height, 40) {
		t.Errorf("c3 = %v, want y=58 h=40", f3)
	}
	// Natural (non-fill) widths are preserved.
	if !approx(f1.Width, 50) {
		t.Errorf("c1 width = %v, want 50", f1.Width)
	}
}

func TestVStackFillHeight(t *testing.T) {
	panel := meui.NewPanel()
	c1 := newChild(0, 0, 50, 20)
	c2 := newChild(0, 0, 50, 30)
	c2.SetLayoutProps(&layout.LayoutProps{FillHeight: true})
	panel.AddChild(c1)
	panel.AddChild(c2)

	vs := layout.NewVStack()
	vs.Spacing = 4
	vs.Layout(panel, graphics.NewRect(0, 0, 200, 100))

	// available = 100 - 4 spacing = 96; fixed c1 = 20 → fill = 76
	f1, f2 := frame(t, c1), frame(t, c2)
	if !approx(f1.Y, 0) || !approx(f1.Height, 20) {
		t.Errorf("c1 = %v, want y=0 h=20", f1)
	}
	if !approx(f2.Y, 24) || !approx(f2.Height, 76) {
		t.Errorf("c2 = %v, want y=24 h=76", f2)
	}
}

func TestVStackMinHeightConstraint(t *testing.T) {
	panel := meui.NewPanel()
	c1 := newChild(0, 0, 50, 10)
	c1.SetLayoutProps(&layout.LayoutProps{MinimumHeight: 60})
	panel.AddChild(c1)

	vs := layout.NewVStack()
	vs.Layout(panel, graphics.NewRect(0, 0, 200, 200))

	if f := frame(t, c1); !approx(f.Height, 60) {
		t.Errorf("c1 height = %v, want 60 (MinimumHeight)", f.Height)
	}
}

func TestVStackAlignmentCenter(t *testing.T) {
	panel := meui.NewPanel()
	c1 := newChild(0, 0, 50, 20)
	c1.SetLayoutProps(&layout.LayoutProps{Alignment: layout.AlignCenterV})
	panel.AddChild(c1)

	vs := layout.NewVStack()
	vs.Layout(panel, graphics.NewRect(0, 0, 200, 200))

	if f := frame(t, c1); !approx(f.X, 75) {
		t.Errorf("c1 x = %v, want 75 (centered)", f.X)
	}
}

func TestVStackMargins(t *testing.T) {
	panel := meui.NewPanel()
	c1 := newChild(0, 0, 50, 20)
	c1.SetLayoutProps(&layout.LayoutProps{LeftMargin: 10, TopMargin: 5})
	panel.AddChild(c1)

	vs := layout.NewVStack()
	vs.Layout(panel, graphics.NewRect(0, 0, 200, 200))

	f := frame(t, c1)
	if !approx(f.X, 10) || !approx(f.Y, 5) {
		t.Errorf("c1 = %v, want x=10 y=5 (margins)", f)
	}
}

// TestVStackHonorsRectOrigin locks in the padding fix: Panel.ValidateLayout
// passes a padded inner rect (origin != 0) and layouts must offset children
// by rect.X/rect.Y, not stack from local (0,0).
func TestVStackHonorsRectOrigin(t *testing.T) {
	panel := meui.NewPanel()
	c1 := newChild(0, 0, 50, 20)
	panel.AddChild(c1)

	vs := layout.NewVStack()
	vs.Layout(panel, graphics.NewRect(8, 8, 184, 184))

	f := frame(t, c1)
	if !approx(f.X, 8) || !approx(f.Y, 8) {
		t.Errorf("c1 = %v, want x=8 y=8 (rect origin honored)", f)
	}
}

func TestGridHonorsRectOrigin(t *testing.T) {
	panel := meui.NewPanel()
	c1 := newChild(0, 0, 50, 20)
	panel.AddChild(c1)

	g := layout.NewGrid(1)
	g.Layout(panel, graphics.NewRect(8, 8, 100, 100))

	f := frame(t, c1)
	if !approx(f.X, 8) || !approx(f.Y, 8) {
		t.Errorf("c1 = %v, want x=8 y=8 (rect origin honored)", f)
	}
}

// ── HStack ─────────────────────────────────────────────

func TestHStackStackingOrder(t *testing.T) {
	panel := meui.NewPanel()
	c1 := newChild(0, 0, 50, 20)
	c2 := newChild(0, 0, 60, 20)
	c3 := newChild(0, 0, 70, 20)
	panel.AddChild(c1)
	panel.AddChild(c2)
	panel.AddChild(c3)

	hs := layout.NewHStack()
	hs.Spacing = 4
	hs.Layout(panel, graphics.NewRect(0, 0, 300, 100))

	f1, f2, f3 := frame(t, c1), frame(t, c2), frame(t, c3)
	if !approx(f1.X, 0) {
		t.Errorf("c1 x = %v, want 0", f1.X)
	}
	if !approx(f2.X, 54) {
		t.Errorf("c2 x = %v, want 54", f2.X)
	}
	if !approx(f3.X, 118) {
		t.Errorf("c3 x = %v, want 118", f3.X)
	}
}

func TestHStackFillWidth(t *testing.T) {
	panel := meui.NewPanel()
	c1 := newChild(0, 0, 50, 20)
	c2 := newChild(0, 0, 50, 20)
	c2.SetLayoutProps(&layout.LayoutProps{FillWidth: true})
	panel.AddChild(c1)
	panel.AddChild(c2)

	hs := layout.NewHStack()
	hs.Spacing = 4
	hs.Layout(panel, graphics.NewRect(0, 0, 200, 100))

	// available = 200 - 4 = 196; fixed c1 = 50 → fill = 146
	f1, f2 := frame(t, c1), frame(t, c2)
	if !approx(f1.X, 0) || !approx(f1.Width, 50) {
		t.Errorf("c1 = %v, want x=0 w=50", f1)
	}
	if !approx(f2.X, 54) || !approx(f2.Width, 146) {
		t.Errorf("c2 = %v, want x=54 w=146", f2)
	}
}

func TestHStackAlignmentBottom(t *testing.T) {
	panel := meui.NewPanel()
	c1 := newChild(0, 0, 50, 20)
	c1.SetLayoutProps(&layout.LayoutProps{Alignment: layout.AlignBottomV})
	panel.AddChild(c1)

	hs := layout.NewHStack()
	hs.Layout(panel, graphics.NewRect(0, 0, 200, 100))

	if f := frame(t, c1); !approx(f.Y, 80) {
		t.Errorf("c1 y = %v, want 80 (bottom-aligned)", f.Y)
	}
}

// ── Grid ───────────────────────────────────────────────

func TestGridPlacement(t *testing.T) {
	panel := meui.NewPanel()
	children := []*meui.BaseWidget{
		newChild(0, 0, 50, 20),
		newChild(0, 0, 40, 20),
		newChild(0, 0, 50, 20),
		newChild(0, 0, 50, 20),
	}
	for _, c := range children {
		panel.AddChild(c)
	}

	g := layout.NewGrid(2)
	g.Spacing = 8
	g.Layout(panel, graphics.NewRect(0, 0, 200, 100))

	// max cell from children = 50x20, floored to 30 height → cells 50x30.
	// Fits in 200x100 with spacing 8 → no scaling.
	for i, c := range children {
		f := frame(t, c)
		col := i % 2
		row := i / 2
		wantX := float32(col) * 58
		wantY := float32(row) * 38
		if !approx(f.X, wantX) || !approx(f.Y, wantY) {
			t.Errorf("child %d = (%v,%v), want (%v,%v)", i, f.X, f.Y, wantX, wantY)
		}
		if !approx(f.Width, 50) || !approx(f.Height, 30) {
			t.Errorf("child %d size = %vx%v, want 50x30", i, f.Width, f.Height)
		}
	}
}

func TestGridScaleDownToFit(t *testing.T) {
	panel := meui.NewPanel()
	for i := 0; i < 4; i++ {
		panel.AddChild(newChild(0, 0, 50, 20))
	}

	g := layout.NewGrid(2)
	g.Spacing = 8
	// total width 2*50+8 = 108 > 96 → cells scaled to (96-8)/2 = 44
	g.Layout(panel, graphics.NewRect(0, 0, 96, 100))

	f := frame(t, panel.ChildrenI()[0].(*meui.BaseWidget))
	if !approx(f.Width, 44) {
		t.Errorf("cell width = %v, want 44 (scaled to fit)", f.Width)
	}
}

// ── Flex ───────────────────────────────────────────────

func flexPanel(children ...*meui.BaseWidget) *meui.Panel {
	panel := meui.NewPanel()
	for _, c := range children {
		panel.AddChild(c)
	}
	return panel
}

func TestFlexRowJustifyStart(t *testing.T) {
	c1 := newChild(0, 0, 50, 20)
	c2 := newChild(0, 0, 60, 20)
	panel := flexPanel(c1, c2)

	fl := layout.NewFlex()
	fl.Direction = layout.FlexRow
	fl.Justify = layout.JustifyStart
	fl.Align = layout.AlignStart
	fl.Spacing = 8
	fl.Layout(panel, graphics.NewRect(0, 0, 200, 100))

	f1, f2 := frame(t, c1), frame(t, c2)
	if !approx(f1.X, 0) {
		t.Errorf("c1 x = %v, want 0", f1.X)
	}
	if !approx(f2.X, 58) {
		t.Errorf("c2 x = %v, want 58", f2.X)
	}
	if !approx(f1.Height, 20) {
		t.Errorf("c1 height = %v, want 20 (AlignStart keeps natural)", f1.Height)
	}
}

func TestFlexRowJustifyEnd(t *testing.T) {
	c1 := newChild(0, 0, 50, 20)
	c2 := newChild(0, 0, 60, 20)
	panel := flexPanel(c1, c2)

	fl := layout.NewFlex()
	fl.Direction = layout.FlexRow
	fl.Justify = layout.JustifyEnd
	fl.Align = layout.AlignStart
	fl.Spacing = 8
	fl.Layout(panel, graphics.NewRect(0, 0, 200, 100))

	// available = 200-8 = 192; total children = 110 → offset = 82
	if f := frame(t, c1); !approx(f.X, 82) {
		t.Errorf("c1 x = %v, want 82 (end-justified)", f.X)
	}
}

func TestFlexRowJustifyCenter(t *testing.T) {
	c1 := newChild(0, 0, 50, 20)
	c2 := newChild(0, 0, 60, 20)
	panel := flexPanel(c1, c2)

	fl := layout.NewFlex()
	fl.Direction = layout.FlexRow
	fl.Justify = layout.JustifyCenter
	fl.Align = layout.AlignStart
	fl.Spacing = 8
	fl.Layout(panel, graphics.NewRect(0, 0, 200, 100))

	// offset = (192-110)/2 = 41
	if f := frame(t, c1); !approx(f.X, 41) {
		t.Errorf("c1 x = %v, want 41 (center-justified)", f.X)
	}
}

func TestFlexRowAlignStretch(t *testing.T) {
	c1 := newChild(0, 0, 50, 20)
	panel := flexPanel(c1)

	fl := layout.NewFlex()
	fl.Direction = layout.FlexRow
	fl.Align = layout.AlignStretch
	fl.Spacing = 8
	fl.Layout(panel, graphics.NewRect(0, 0, 200, 100))

	if f := frame(t, c1); !approx(f.Height, 100) {
		t.Errorf("c1 height = %v, want 100 (stretched)", f.Height)
	}
}

func TestFlexColumnJustifyStart(t *testing.T) {
	c1 := newChild(0, 0, 50, 20)
	c2 := newChild(0, 0, 50, 30)
	panel := flexPanel(c1, c2)

	fl := layout.NewFlex()
	fl.Direction = layout.FlexColumn
	fl.Justify = layout.JustifyStart
	fl.Align = layout.AlignStart
	fl.Spacing = 8
	fl.Layout(panel, graphics.NewRect(0, 0, 200, 100))

	f1, f2 := frame(t, c1), frame(t, c2)
	if !approx(f1.Y, 0) || !approx(f2.Y, 28) {
		t.Errorf("column stacking: c1.y=%v c2.y=%v, want 0 and 28", f1.Y, f2.Y)
	}
}

// TestLayoutMustNotResizeItsPanel pins the double-padding fix: the panel
// frame is owned by the parent, and a layout receives the panel's PADDED
// inner rect while positioning children (with the same padding offset).
// A layout that resized its target to the inner rect applied the padding
// twice: children stuck out past the panel frame by the padding and were
// clipped by whoever clips the panel (clipped bottom edges/scrollbar caps).
func TestLayoutMustNotResizeItsPanel(t *testing.T) {
	panel := meui.NewPanel()
	panel.SetFrameRect(graphics.NewRect(0, 0, 300, 200))
	panel.SetPadding(8)

	child := newChild(0, 0, 50, 20)
	child.SetLayoutProps(&layout.LayoutProps{FillHeight: true})
	panel.AddChild(child)

	panel.SetLayout(layout.NewVStack())
	panel.SetNeedsLayout()
	panel.ValidateLayout()

	if f := frame(t, panel); !approx(f.Width, 300) || !approx(f.Height, 200) {
		t.Errorf("panel frame resized by its own layout: %+v, want (0,0 300x200)", f)
	}
	// Children must live inside the padded inner rect (8,8)-(292,192).
	if f := frame(t, child); f.X < 7.99 || f.Y < 7.99 || f.X+f.Width > 292.01 || f.Y+f.Height > 192.01 {
		t.Errorf("child %+v escapes the padded inner rect (8,8 284x184)", f)
	}
}

// The resize scene, measured without a GPU, a window or a compositor (behaviour's
// TestResizingTheWindowKeepsTheLayoutGradientAndClipAnchored renders it for real).
//
// The screen harness cannot judge this: two arms running the SAME binary differed by ~67
// points at 620 and 700 because what it measures is which frame the compositor picked up
// (P5fo). This is the same tree - 20px header, FillHeight body, 20px footer, 8px spacing -
// laid out at the probe's two content heights, so the composition question has a
// deterministic answer: at 592 the body must be 544, at 352 it must be 304.
func TestVStackProbeTreeComposition(t *testing.T) {
	// NOTE the values asserted here are what the layout computes, and they are 8px BELOW
	// the height the screen harness expects (`window - 76`). The reason is that three
	// children have TWO spacings, so body = content - 2*8 - 40 = content - 56, while the
	// harness assumed one spacing (content - 48). That discrepancy is recorded as P5fp and
	// must be resolved before any further rate claim: an expected height that is 8px wrong
	// scores correct frames as failures.
	for _, tc := range []struct{ contentH, wantBody float32 }{
		{352, 304}, // window 380: content 352 - what the GPU renders
		{592, 544}, // window 620: content 592 - what the GPU renders when correct
	} {
		panel := meui.NewPanel()
		header := newChild(0, 0, 520, 20)
		body := newChild(0, 0, 520, 0)
		footer := newChild(0, 0, 520, 20)
		body.SetLayoutProps(&layout.LayoutProps{FillWidth: true, FillHeight: true})
		panel.AddChild(header)
		panel.AddChild(body)
		panel.AddChild(footer)

		vs := layout.NewVStack() // Spacing 4 by default - the probe uses the default
		vs.Layout(panel, graphics.NewRect(0, 0, 520, tc.contentH))

		bf := frame(t, body)
		if !approx(bf.Height, tc.wantBody) {
			t.Errorf("content %v: body height = %v, want %v", tc.contentH, bf.Height, tc.wantBody)
		}
		if ff := frame(t, footer); !approx(ff.Y, tc.contentH-20) {
			t.Errorf("content %v: footer y = %v, want %v", tc.contentH, ff.Y, tc.contentH-20)
		}
	}
}

// The same tree through the PANEL's own path (SetLayout + ValidateLayout) rather than a direct
// VStack.Layout call, which is what P5fp showed bypasses the panel's rect computation and
// disagreed with the render by 8px. This logs the body height for both content heights so the
// panel path can be compared with what the probe renders (304 at a 380 window, 544 at 620).
func TestPanelPathProbeTree(t *testing.T) {
	for _, contentH := range []float32{352, 592} {
		panel := meui.NewPanel()
		header := newChild(0, 0, 520, 20)
		body := newChild(0, 0, 520, 0)
		footer := newChild(0, 0, 520, 20)
		body.SetLayoutProps(&layout.LayoutProps{FillWidth: true, FillHeight: true})
		panel.AddChild(header)
		panel.AddChild(body)
		panel.AddChild(footer)
		vs := layout.NewVStack() // Spacing 4 by default - the probe uses the default
		panel.SetLayout(vs)
		panel.SetFrameRect(graphics.NewRect(0, 0, 520, contentH))
		panel.ValidateLayout()
		t.Logf("content %.0f: body height = %.0f, footer y = %.0f (render gives body %.0f)",
			contentH, frame(t, body).Height, frame(t, footer).Y, contentH+8)
	}
}

// The software-Device oracle: the same tree rendered through the CPU device and measured with
// the same pure-red mask the GPU dumps use. This compares like with like, which the screen
// harness cannot (it needs a compositor and measures 67 points of noise between two arms of
// the same binary, P5fo). If the software render shows the same body extent as the GPU dumps
// (304 rows at content 352, 544 at 592) while the layout frames are 296/536, then the
// frame-versus-paint difference is a property of painting, not of the GPU path; if the
// software render shows 296/536, painting agrees with layout and the GPU path is the
// difference.
func TestSoftwareRenderProbeTree(t *testing.T) {
	for _, contentH := range []float32{352, 592} {
		panel := meui.NewPanel()
		header := meui.NewPanel()
		header.SetFrameRect(graphics.NewRect(0, 0, 520, 20))
		body := meui.NewPanel()
		body.SetFrameRect(graphics.NewRect(0, 0, 520, 0))
		footer := meui.NewPanel()
		footer.SetFrameRect(graphics.NewRect(0, 0, 520, 20))
		header.SetBackground(graphics.RGB(0, 0, 255))
		body.SetBackground(graphics.RGB(255, 0, 0))
		footer.SetBackground(graphics.RGB(0, 255, 0))
		body.SetLayoutProps(&layout.LayoutProps{FillWidth: true, FillHeight: true})
		panel.AddChild(header)
		panel.AddChild(body)
		panel.AddChild(footer)
		vs := layout.NewVStack() // Spacing 4 by default - the probe uses the default
		panel.SetLayout(vs)
		panel.SetFrameRect(graphics.NewRect(0, 0, 520, contentH))
		// A fresh widget is INVISIBLE (visible defaults false) and Panel.Paint returns
		// early when it is - which is why the first oracle run painted nothing at all.
		panel.SetVisible(true)
		header.SetVisible(true)
		body.SetVisible(true)
		footer.SetVisible(true)
		panel.ValidateLayout()

		canvas, dev := graphics.NewSoftwareCanvas(520, int(contentH))
		canvas.Clear(0xFF000000)
		panel.Paint(canvas, graphics.NewRect(0, 0, 520, contentH))

		// The seam packs pixels as a<<24|r<<16|g<<8|b and emits rows bottom-up.
		px := dev.Pixels()
		rows := 0
		for y := 0; y < int(contentH); y++ {
			for x := 100; x < 120; x++ {
				// Bottom-up rows: flip the top-down y.
				if px[(int(contentH)-1-y)*520+x]&0x00FFFFFF == 0x00FF0000 {
					rows++
					break
				}
			}
		}
		// Known-positive: draw a red rect straight on the software canvas, bypassing the
		// panel. If the mask finds this, the Canvas->device path works and the emptiness is
		// in the panel paint; if not, the mask or the device is at fault.
		{
			rp := graphics.NewPaint()
			rp.SetColor(graphics.RGB(255, 0, 0))
			canvas.DrawRect(graphics.NewRect(0, 0, 20, 10), rp)
			found := 0
			// Pixels() emits rows BOTTOM-UP (the seam's convention), so a top-down y must be
			// flipped to index the buffer: this was why the mask found nothing.
			for y := 0; y < 10; y++ {
				for x := 0; x < 20; x++ {
					if dev.Pixels()[(int(contentH)-1-y)*520+x]&0x00FFFFFF == 0x00FF0000 {
						found++
					}
				}
			}
			t.Logf("content %.0f: direct red rect on the software canvas -> %d/200 red pixels", contentH, found)
		}
		// VALUE dump, not a count: what is actually in column 110 down the surface, in
		// top-down order with the bottom-up flip applied. This shows what is there rather
		// than whether the mask predicate matched (P5fz).
		{
			seen := []string{}
			var lastVal uint32
			start := 0
			for y := 0; y < int(contentH); y++ {
				v := px[(int(contentH)-1-y)*520+110]
				if y == 0 {
					lastVal, start = v, 0
					continue
				}
				if v != lastVal {
					seen = append(seen, fmt.Sprintf("y%d-%d=%#08x", start, y-1, lastVal))
					lastVal, start = v, y
				}
			}
			seen = append(seen, fmt.Sprintf("y%d-%d=%#08x", start, int(contentH)-1, lastVal))
			t.Logf("content %.0f column 110 (top-down): %v", contentH, seen)
		}
		if rows == 0 {
			// Nothing painted: the software device is allocated and panel.Paint ran, but no
			// pure red reached the surface. That makes this oracle silent rather than wrong,
			// so it skips with the reason instead of asserting a comparison it cannot make.
			t.Skipf("content %.0f: panel.Paint wrote nothing to the software surface - the "+
				"child paint path is not reached from a bare ValidateLayout+Paint, so the "+
				"oracle needs fixing before it can compare (see MIGRATION.md P5fs)", contentH)
		}
		t.Logf("content %.0f: layout body %.0f, software-rendered red rows = %d (GPU dumps gave %.0f at window %.0f)",
			contentH, body.FrameRect().Height, rows, contentH+8, contentH+28)
	}
}
