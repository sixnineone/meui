package meui

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
)

// ── The coordinate contract (see the block comment on FrameRect) ─────
//
//	FRAME space : parent-relative. FrameRect()/SetFrameRect(). Only
//	              parents (PaintChildren, dispatchers, layouts) interpret
//	              its origin.
//	LOCAL space : widget-relative (0,0 = widget top-left). Paint's rect
//	              parameter and every event coordinate a widget receives.
//
// These tests pin the boundaries so the two spaces cannot silently
// drift; violations historically showed as shifted scrollbars/children.

// coordRecorder captures the space it is painted/dispatched in.
type coordRecorder struct {
	Panel
	paintRect   graphics.Rect
	paintOrigin [2]float32
	mouseLocal  [2]float32
}

func (r *coordRecorder) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	r.paintRect = rect
	r.paintOrigin[0], r.paintOrigin[1] = canvas.Origin()
	r.Panel.Paint(canvas, rect)
}

func (r *coordRecorder) OnMouseDown(x, y float32, button int) bool {
	r.mouseLocal = [2]float32{x, y}
	return true
}

func newCoordRecorder() *coordRecorder {
	return &coordRecorder{Panel: *NewPanel()}
}

func TestFrameRectParentRelativeLocalRectZeroBased(t *testing.T) {
	p := NewPanel()
	p.SetFrameRect(graphics.NewRect(11, 22, 100, 50))

	fr := p.FrameRect()
	if fr.X != 11 || fr.Y != 22 || fr.Width != 100 || fr.Height != 50 {
		t.Errorf("FrameRect = %+v, want parent-relative (11,22 100x50)", fr)
	}
	lr := p.LocalRect()
	if lr.X != 0 || lr.Y != 0 || lr.Width != 100 || lr.Height != 50 {
		t.Errorf("LocalRect = %+v, want (0,0 100x50)", lr)
	}
}

func TestPaintChildrenPassesLocalRectAndTranslatedCanvas(t *testing.T) {
	root := NewPanel()
	root.SetFrameRect(graphics.NewRect(0, 0, 400, 300))

	child := newCoordRecorder()
	child.SetFrameRect(graphics.NewRect(11, 22, 100, 50))
	root.AddChild(child)

	root.Paint(graphics.NewCanvas(graphics.NewTestDevice(400, 300)), graphics.NewRect(0, 0, 400, 300))

	want := graphics.NewRect(0, 0, 100, 50)
	if child.paintRect != want {
		t.Errorf("child Paint rect = %+v, want LOCAL %+v", child.paintRect, want)
	}
	if child.paintOrigin != [2]float32{11, 22} {
		t.Errorf("child canvas origin = %v,%v, want the parent-relative frame origin 11,22",
			child.paintOrigin[0], child.paintOrigin[1])
	}
}

func TestMouseDispatchPassesLocalCoordinates(t *testing.T) {
	root := NewPanel()
	root.SetFrameRect(graphics.NewRect(0, 0, 400, 300))

	child := newCoordRecorder()
	child.SetFrameRect(graphics.NewRect(11, 22, 100, 50))
	root.AddChild(child)

	// Press at the child's parent-relative frame + (5,7).
	if !root.DispatchMouseDown(11+5, 22+7, 0) {
		t.Fatal("child did not handle the press")
	}
	if child.mouseLocal != [2]float32{5, 7} {
		t.Errorf("child received local (%.0f,%.0f), want (5,7)", child.mouseLocal[0], child.mouseLocal[1])
	}
}
