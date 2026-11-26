package behaviour

import (
	"testing"

	"github.com/sixnineone/meui/widgets"
)

const scrollpaneQML = `
Window {
    id: win
    width: 400
    height: 300
    VStack {
        spacing: 10
        margins: 10
        ScrollPane {
            id: sp
            width: 200
            height: 80
            contentWidth: 180
            contentHeight: 200
            VStack {
                spacing: 4
                margins: 4
                Label { text: "Item 1" }
                Label { text: "Item 2" }
                Label { text: "Item 3" }
                Label { text: "Item 4" }
                Label { text: "Item 5" }
                Label { text: "Item 6" }
                Label { text: "Item 7" }
                Label { text: "Item 8" }
            }
        }
    }
}
`

// TestScrollPaneScrollBasics: wheel input scrolls the content, clamps at
// both ends, and leaves the scrollbar state consistent.
func TestScrollPaneScrollBasics(t *testing.T) {
	h := Load(t, scrollpaneQML)
	sp := h.Widget("sp").(*widgets.ScrollPane)

	// The first child of ScrollPane in the registry is its content VStack;
	// fetch the pane's own scrollbars via the exported struct fields.
	h.Settle()
	t.Logf("sp frame=%v contentFrame=%v contentH=%v vscroll.visible=%v", sp.FrameRect(), sp.Content.FrameRect(), sp.ContentHeight, sp.VScroll.Visible())

	if !sp.VScroll.Visible() {
		t.Fatalf("VScroll should be visible for content 200 high in an 80-high viewport")
	}
	// Absolute-pixel units: Max=content extent, VisibleAmount=viewport,
	// value clamps to [0, Max-VisibleAmount].
	if sp.VScroll.Max <= 0 {
		t.Fatalf("VScroll.Max should be > 0, got %v", sp.VScroll.Max)
	}
	if sp.VScroll.VisibleAmount <= 0 || sp.VScroll.VisibleAmount >= sp.VScroll.Max {
		t.Fatalf("VisibleAmount %v should be in (0, Max %v)", sp.VScroll.VisibleAmount, sp.VScroll.Max)
	}
	if sp.VScroll.Value != 0 {
		t.Fatalf("VScroll should start at 0, got %v", sp.VScroll.Value)
	}

	x, y := Center(sp)
	maxScroll := sp.VScroll.Max - sp.VScroll.VisibleAmount

	// Wheel down (dy<0 = toward user) must scroll DOWN (value grows).
	h.Wheel(x, y, 0, -1)
	h.Settle()
	if sp.VScroll.Value <= 0 {
		t.Errorf("wheel down: value should grow above 0, got %v", sp.VScroll.Value)
	}
	first := sp.VScroll.Value

	// Same delta again: more growth.
	h.Wheel(x, y, 0, -1)
	h.Settle()
	if sp.VScroll.Value <= first {
		t.Errorf("second wheel down: value should keep growing: %v -> %v", first, sp.VScroll.Value)
	}

	// Slam to the bottom: value clamps at Max.
	for i := 0; i < 30; i++ {
		h.Wheel(x, y, 0, -1)
	}
	h.Settle()
	if sp.VScroll.Value != maxScroll {
		t.Errorf("wheel to bottom: value %v, want clamp at Max %v", sp.VScroll.Value, maxScroll)
	}

	// Wheel up scrolls back UP (value shrinks).
	h.Wheel(x, y, 0, 1)
	h.Settle()
	if sp.VScroll.Value >= maxScroll {
		t.Errorf("wheel up: value should shrink below Max, got %v", sp.VScroll.Value)
	}

	// Slam back to top: value clamps at 0.
	for i := 0; i < 40; i++ {
		h.Wheel(x, y, 0, 1)
	}
	h.Settle()
	if sp.VScroll.Value != 0 {
		t.Errorf("wheel to top: value %v, want 0", sp.VScroll.Value)
	}
}
