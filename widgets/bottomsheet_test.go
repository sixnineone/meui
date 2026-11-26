package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
)

// TestBottomSheetPanelIsCappedCentredAndAnchored pins the three published numbers at once,
// because each is easy to get subtly wrong and none of them shows in a unit of its own: the
// width cap, the margin that takes over past it, and the bottom edge the panel is anchored
// to.
func TestBottomSheetPanelIsCappedCentredAndAnchored(t *testing.T) {
	s := NewBottomSheet()

	// A wide area: capped at 640 and centred.
	wide := graphics.NewRect(0, 0, 1000, 800)
	p := s.PanelRect(wide)
	if p.Width != BottomSheetMaxWidth {
		t.Errorf("on a 1000-wide area the panel is %.0f wide, want the %v cap", p.Width, BottomSheetMaxWidth)
	}
	if want := (float32(1000) - BottomSheetMaxWidth) / 2; p.X != want {
		t.Errorf("the capped panel starts at %.0f, want it centred at %.0f", p.X, want)
	}
	if p.Y+p.Height != 800 {
		t.Errorf("the panel ends at %.0f, want the area's bottom edge 800", p.Y+p.Height)
	}

	// Between the cap and twice the margin the MARGIN is what limits the width.
	mid := graphics.NewRect(0, 0, 700, 800)
	if w := s.PanelRect(mid).Width; w != 700-2*BottomSheetSideMargin {
		t.Errorf("on a 700-wide area the panel is %.0f wide, want %.0f (a %v margin each side)",
			w, 700-2*BottomSheetSideMargin, BottomSheetSideMargin)
	}

	// A narrow area: the full width, because the cap has not started to apply.
	narrow := graphics.NewRect(0, 0, 400, 800)
	if w := s.PanelRect(narrow).Width; w != 400 {
		t.Errorf("on a 400-wide area the panel is %.0f wide, want the full 400", w)
	}
}

// TestBottomSheetLeavesTheTopMargin is the rule that keeps a sheet from becoming a full-screen
// page: however tall it is asked to be, it leaves BottomSheetTopMargin uncovered.
func TestBottomSheetLeavesTheTopMargin(t *testing.T) {
	s := NewBottomSheet().SetSheetHeight(1000)
	area := graphics.NewRect(0, 0, 400, 800)
	p := s.PanelRect(area)
	if want := float32(800) - BottomSheetTopMargin; p.Height != want {
		t.Errorf("a 1000-tall sheet in an 800 area is %.0f tall, want %v", p.Height, want)
	}
	if p.Y != BottomSheetTopMargin {
		t.Errorf("the panel starts at %.0f, want the top margin %v", p.Y, BottomSheetTopMargin)
	}

	// And with no height set it takes half the area, which always leaves the margin.
	s2 := NewBottomSheet()
	if h := s2.PanelHeight(area); h != 400 {
		t.Errorf("an unsized sheet is %.0f tall in an 800 area, want half (400)", h)
	}
}

// TestBottomSheetDragDismissesOnlyPastTheThreshold is the behaviour a drag handle has to get
// right: a small accidental drag springs back, and a deliberate one dismisses. The threshold
// is a fraction of the SHEET, not of the screen — a short sheet would otherwise be impossible
// to dismiss by dragging.
func TestBottomSheetDragDismissesOnlyPastTheThreshold(t *testing.T) {
	s := NewBottomSheet().SetSheetHeight(300)
	s.SetFrameRect(graphics.NewRect(0, 0, 400, 800))
	dismissed := false
	s.OnDismissed = func() { dismissed = true }

	h := s.HandleRect(s.FrameRect())
	press := func(dy float32) {
		s.OnMouseDown(h.X+h.Width/2, h.Y+h.Height/2, 0)
		s.OnMouseMove(h.X+h.Width/2, h.Y+h.Height/2+dy)
		s.OnMouseUp(h.X+h.Width/2, h.Y+h.Height/2+dy, 0)
	}

	press(50) // under a third of 300
	if dismissed {
		t.Error("a 50px drag dismissed a 300px sheet")
	}
	if s.dragY != 0 {
		t.Errorf("a small drag left the panel offset by %.0f, want it sprung back", s.dragY)
	}

	press(150) // over a third of 300
	if !dismissed {
		t.Error("a 150px drag did not dismiss a 300px sheet")
	}
}

// TestBottomSheetHandleStartsTheDrag keeps the affordance and the hit area together: the
// handle is the only part of the panel that starts a drag, so a press on the content does
// not move the sheet out from under a control inside it.
func TestBottomSheetHandleStartsTheDrag(t *testing.T) {
	s := NewBottomSheet().SetSheetHeight(300)
	s.SetFrameRect(graphics.NewRect(0, 0, 400, 800))

	h := s.HandleRect(s.FrameRect())
	if !s.OnMouseDown(h.X+h.Width/2, h.Y+h.Height/2, 0) {
		t.Error("a press on the handle was not taken")
	}
	if !s.dragging {
		t.Error("a press on the handle did not start a drag")
	}
	s.OnMouseUp(h.X, h.Y, 0)

	// A press inside the panel but off the handle is the content's, not the sheet's.
	if s.OnMouseDown(200, s.PanelRect(s.FrameRect()).Y+120, 0) {
		t.Error("a press inside the panel started a drag; only the handle should")
	}
}

// TestBottomSheetModalDismissesOnATapOutside is the difference between the two presentations,
// and it is behaviour rather than paint: a modal sheet goes away when the user taps the scrim,
// a standard one is part of the layout and stays.
func TestBottomSheetModalDismissesOnATapOutside(t *testing.T) {
	modal := NewBottomSheet().SetSheetHeight(300)
	modal.SetFrameRect(graphics.NewRect(0, 0, 400, 800))
	modalDismissed := false
	modal.OnDismissed = func() { modalDismissed = true }
	if !modal.OnMouseDown(200, 100, 0) {
		t.Error("a tap above a modal sheet was not taken")
	}
	if !modalDismissed {
		t.Error("a tap above a modal sheet did not dismiss it")
	}

	standard := NewBottomSheet().SetSheetHeight(300).SetModal(false)
	standard.SetFrameRect(graphics.NewRect(0, 0, 400, 800))
	standardDismissed := false
	standard.OnDismissed = func() { standardDismissed = true }
	if standard.OnMouseDown(200, 100, 0) {
		t.Error("a standard sheet took a tap outside its panel")
	}
	if standardDismissed {
		t.Error("a standard sheet dismissed on a tap outside")
	}
}

// TestBottomSheetIsAnInputTarget keeps it in the registry the touch audit checks: the drag
// handle is a target, and a widget that handles input but is missing from the registry is
// exactly what the audit exists to catch.
func TestBottomSheetIsAnInputTarget(t *testing.T) {
	if !meui.IsInteractive(NewBottomSheet()) {
		t.Error("a BottomSheet handles input but is not in the interactive registry")
	}
}

var _ meui.EventListener = (*BottomSheet)(nil)
