package meui

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
)

// hoverSpy is an ancestor that wants the pointer position (a SplitView is the
// real one): it records every HoverAt it is handed, in its own coordinates.
type hoverSpy struct {
	BaseContainer
	at []graphics.Point
}

var _ Container = (*hoverSpy)(nil)

func newHoverSpy() *hoverSpy {
	return &hoverSpy{BaseContainer: BaseContainer{BaseWidget: *NewBaseWidget()}}
}

// ValidateLayout and PanelAt are what make the stub a real Container (the
// window only recurses into one), as every real container does.
func (h *hoverSpy) ValidateLayout() {}

// AddChild parents to the OUTER widget, not to the embedded BaseContainer:
// the parent chain is what the hover walk climbs, and a chain through the
// embedded struct would never reach the stub itself.
func (h *hoverSpy) AddChild(w Widget) {
	h.AppendChild(w)
	if w != nil {
		w.SetParent(h)
	}
}

func (h *hoverSpy) PanelAt(x, y float32) Widget {
	if hit := h.DefaultPanelAt(x, y); hit != nil {
		return hit
	}
	return h
}

func (h *hoverSpy) HoverAt(x, y float32) { h.at = append(h.at, graphics.NewPoint(x, y)) }

// hitLeaf is the deepest widget under the pointer.
type hitLeaf struct {
	BaseWidget
	moves []graphics.Point
}

func newHitLeaf() *hitLeaf {
	return &hitLeaf{BaseWidget: *NewBaseWidget()}
}

func (l *hitLeaf) OnMouseMove(x, y float32) { l.moves = append(l.moves, graphics.NewPoint(x, y)) }

// A pointer move must reach every widget along the hit chain, not only the
// deepest one: a SplitView handle is part of the SplitView's own area, but
// the hit over the panels is a child of a panel, so the splitter only ever
// learned the pointer had arrived and never that it had left — the divider
// stayed stuck in its hover ink.
func TestMouseMoveReachesAncestorHoverTargets(t *testing.T) {
	w := NewWindow()

	outer := newHoverSpy()
	outer.SetFrameRect(graphics.NewRect(20, 30, 200, 100))
	w.AddChild(outer)

	mid := newHoverSpy()
	mid.SetFrameRect(graphics.NewRect(5, 5, 150, 60)) // outer-local
	outer.AddChild(mid)

	leaf := newHitLeaf()
	leaf.SetFrameRect(graphics.NewRect(10, 10, 40, 20)) // mid-local
	mid.AddChild(leaf)

	// Window point (45, 55) is (10,10) leaf-local, (20,20) mid-local and
	// (25,25) outer-local — one point, three coordinate spaces.
	w.handleMouseMove(45, 55)

	if want := graphics.NewPoint(10, 10); len(leaf.moves) != 1 || leaf.moves[0] != want {
		t.Fatalf("leaf moves = %v, want one at %v", leaf.moves, want)
	}
	if want := graphics.NewPoint(20, 20); len(mid.at) != 1 || mid.at[0] != want {
		t.Errorf("mid HoverAt = %v, want one at %v (its own coordinates)", mid.at, want)
	}
	if want := graphics.NewPoint(25, 25); len(outer.at) != 1 || outer.at[0] != want {
		t.Errorf("outer HoverAt = %v, want one at %v", outer.at, want)
	}
}
