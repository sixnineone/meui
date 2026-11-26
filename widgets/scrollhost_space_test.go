package widgets

import (
	"strings"
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
)

// Coordinate contract for ScrollbarHost: bar geometry is laid out in the
// widget's LOCAL space (origin 0,0) from SIZES only, and all hit tests
// take widget-local coordinates.
//
// Regression net for the historic bug class where a widget passed its
// parent-relative FrameRect() to LayoutBars: the bars were then shifted
// by the widget's position inside its parent (thumb started below the
// content, bottom cap clipped outside the page).

func TestScrollbarHostBarsAreInsideLocalFrame(t *testing.T) {
	const fw, fh = 300, 200
	h := NewScrollbarHost()
	h.SetExtents(1000, 1000, fw, fh) // content larger than viewport → both bars
	h.LayoutBars(fw, fh)

	v, hb, corner := h.BarRects()
	for name, r := range map[string]graphics.Rect{"v": v, "h": hb, "corner": corner} {
		if r.Width == 0 || r.Height == 0 {
			t.Fatalf("%s bar not laid out: %+v", name, r)
		}
		if r.X < 0 || r.Y < 0 || r.X+r.Width > fw+0.001 || r.Y+r.Height > fh+0.001 {
			t.Errorf("%s bar %+v escapes the local frame (0,0 %vx%v)", name, r, fw, fh)
		}
	}
	if v.X != fw-h.BarSize {
		t.Errorf("v bar X = %v, want %v (right edge of the local frame)", v.X, fw-h.BarSize)
	}
	if v.Y >= fh/2 {
		t.Errorf("v bar Y = %v, want near the top of the local frame", v.Y)
	}
}

func TestScrollbarHostHitTestTakesLocalCoords(t *testing.T) {
	const fw, fh = 100, 100
	h := NewScrollbarHost()
	h.SetExtents(100, 1000, fw, fh) // vertical only
	h.LayoutBars(fw, fh)

	v, _, _ := h.BarRects()
	// At offset 0 the thumb sits at the strip's top; aim inside it.
	cx, cy := v.X+v.Width/2, v.Y+4

	// A press at the PAINTED (local) thumb position must start a drag.
	if !h.HandleMouseDown(fw, fh, cx, cy, 0) {
		t.Fatalf("local press at thumb %v,%v was not consumed", cx, cy)
	}
	if !h.VScroll.dragging {
		t.Fatal("press on the thumb did not start a drag")
	}

	// Dragging down (local +y) must advance the offset.
	before := h.VScroll.Value
	h.HandleMouseMove(cx, cy+20)
	if h.VScroll.Value <= before {
		t.Errorf("drag down: value %v, want > %v", h.VScroll.Value, before)
	}
	h.HandleMouseUp(cx, cy+20, 0)

	// Coordinates shifted by a frame origin must NOT hit the bar: the
	// host has no frame parameter, so this pins that callers feed it
	// widget-local coordinates.
	if h.HandleMouseDown(fw, fh, cx+37, cy+41, 0) {
		t.Error("press at parent-relative coordinates must not hit the local bar")
	}
}

func TestScrollbarHostHoverRevealsOverlayBar(t *testing.T) {
	const fw, fh = 100, 100
	h := NewScrollbarHost()
	h.SetExtents(100, 1000, fw, fh)
	h.LayoutBars(fw, fh)
	v, _, _ := h.BarRects()

	h.UpdateHover(10, 50) // left of the strip
	if h.VScroll.Hovered() {
		t.Error("bar hovered with the pointer outside its strip")
	}
	h.UpdateHover(v.X+v.Width/2, v.Y+v.Height/2)
	if !h.VScroll.Hovered() {
		t.Error("overlay bar did not reveal on strip hover")
	}
	// Hover uses widget-local coordinates: shifting by a frame origin
	// must leave the strip (and clear the hover).
	h.UpdateHover(v.X+v.Width/2+37, v.Y+v.Height/2+41)
	if h.VScroll.Hovered() {
		t.Error("hover accepted non-local coordinates")
	}
}

// TestWidgetScrollbarHostsUseLocalSpace is the per-widget regression
// net: every widget that delegates to a ScrollbarHost must lay its bars
// out inside its own (0,0)-based frame even when its position inside the
// parent is non-zero, and its mouse routing must hit the same space.
func TestWidgetScrollbarHostsUseLocalSpace(t *testing.T) {
	const fx, fy, fw, fh = 37, 41, 300, 200

	many := func(n int) []string {
		s := make([]string, n)
		for i := range s {
			s[i] = "item"
		}
		return s
	}

	cases := []struct {
		name  string
		build func() (meui.Widget, *ScrollbarHost)
	}{
		{"ListView", func() (meui.Widget, *ScrollbarHost) {
			w := NewListView()
			w.Items = many(50)
			return w, w.host
		}},
		{"DataTable", func() (meui.Widget, *ScrollbarHost) {
			w := NewDataTable()
			w.SetColumns([]DataColumn{{Label: "A", Width: 100}, {Label: "B"}})
			rows := make([]DataRow, 50)
			for i := range rows {
				rows[i] = DataRow{Cells: []DataCell{{Text: "x"}, {Text: "y"}}}
			}
			w.SetRows(rows)
			return w, w.host
		}},
		{"Table", func() (meui.Widget, *ScrollbarHost) {
			w := NewTable()
			rows := make([][]TableCell, 50)
			for i := range rows {
				rows[i] = []TableCell{{Text: "x"}, {Text: "y"}}
			}
			w.SetRows(rows)
			return w, w.host
		}},
		{"Tree", func() (meui.Widget, *ScrollbarHost) {
			w := NewTree()
			root := &TreeNode{Label: "root", Expanded: true}
			for i := 0; i < 50; i++ {
				w.AddChild(root, "child")
			}
			w.SetRoot(root)
			return w, w.host
		}},
		{"TextArea", func() (meui.Widget, *ScrollbarHost) {
			w := NewTextArea()
			text := ""
			for i := 0; i < 60; i++ {
				text += "line of text\n"
			}
			w.SetText(text)
			return w, w.host
		}},
		{"ScrollPane", func() (meui.Widget, *ScrollbarHost) {
			w := NewScrollPane()
			w.SetContentSize(1000, 1000)
			return w, w.host
		}},
		{"MarkdownView", func() (meui.Widget, *ScrollbarHost) {
			w := NewMarkdownView()
			// Enough content that the laid-out extent exceeds the frame:
			// Paint now rebuilds the layout (and so the extent) itself.
			w.SetText(strings.Repeat("# heading\n\nbody line of text\n\n", 40))
			return w, w.host
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, host := tc.build()
			// A non-zero parent-relative origin is the worst case: the
			// old bug laid the bars out at fx+... / fy+...
			w.SetFrameRect(graphics.NewRect(fx, fy, fw, fh))
			w.Paint(graphics.NewCanvas(graphics.NewTestDevice(800, 600)), graphics.NewRect(0, 0, fw, fh))

			v, _, _ := host.BarRects()
			if v.Width == 0 || v.Height == 0 {
				t.Fatalf("vertical bar not laid out (scrollable widget expected)")
			}
			if v.X < 0 || v.Y < 0 || v.X+v.Width > fw+0.001 || v.Y+v.Height > fh+0.001 {
				t.Fatalf("bar %+v escapes local frame (0,0 %vx%v) — parent-relative rect leaked into bar layout", v, fw, fh)
			}

			el, ok := w.(interface {
				OnMouseDown(float32, float32, int) bool
			})
			if !ok {
				t.Fatalf("widget has no OnMouseDown")
			}

			// Press on the painted (local) thumb must reach the bar. The
			// thumb starts at the strip's top (offset 0), so aim a few
			// pixels below the top rather than at the whole strip's center.
			host.VScroll.dragging = false
			cx, cy := v.X+v.Width/2, v.Y+4
			el.OnMouseDown(cx, cy, 0)
			if !host.VScroll.dragging {
				t.Errorf("press at painted thumb (local %v,%v) did not start a drag — paint and hit spaces diverge", cx, cy)
			}

			// The same press shifted by the frame origin must not: proves
			// the widget routes true local coordinates.
			host.VScroll.dragging = false
			el.OnMouseDown(cx+fx, cy+fy, 0)
			if host.VScroll.dragging {
				t.Errorf("press at parent-relative coords hit the bar — wrong coordinate space in mouse routing")
			}

			// Hover reveal uses the same local space: moving over the
			// strip reveals the overlay pill; the frame-shifted move does
			// not.
			if em, ok := w.(interface{ OnMouseMove(float32, float32) }); ok {
				em.OnMouseMove(cx, cy)
				if !host.VScroll.Hovered() {
					t.Errorf("strip hover did not reveal the overlay bar")
				}
				em.OnMouseMove(cx+fx, cy+fy)
				if host.VScroll.Hovered() {
					t.Errorf("hover used parent-relative coordinates (bar stayed hovered after frame-shifted move)")
				}
			}
		})
	}
}
