package widgets

// spinbox_click_test.go — WHERE a click on the arrows lands, in widget-local space.
//
// The touch-target floor expands a small control's HIT frame so a finger can land near it. It must
// not move the control's ORIGIN: the five dispatch functions in BaseContainer used to subtract the
// EXPANDED frame, which shifted every local coordinate by half the expansion (11px on a 22px
// control). A spin box's up arrow therefore received coordinates from its lower half: clicking UP
// stepped DOWN, and the only way to step up was to click ABOVE the arrows — reported from the
// signal-rule dialog, whose controls are all inside panels and whose window can be phone/tablet
// sized, which is when the floor is on.

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
)

// clickSpinboxArrow clicks the centre of the up or down arrow of a spin box hosted in a modal popup,
// through the window's real input path, and returns the value afterwards.
func clickSpinboxArrow(t *testing.T, up bool, floor bool) float32 {
	t.Helper()
	w := meui.NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 900, 700))
	w.SetTouchProfile(floor)

	content := meui.NewPanel()
	spin := NewSpinBox()
	spin.SetRange(0, 10)
	spin.SetValue(5)
	spin.SetDecimals(0)
	spin.SetFrameRect(graphics.NewRect(10, 10, 100, 22))
	content.AddChild(spin)

	popup := NewModalPopupPanel("Edit Signal Rule", "", content)
	w.ShowModal(popup, 420, 650)

	// The content frame the paint pass computes. The heading is empty, so the input offset and the
	// painted offset are the same value — this test is about the CONTROL's frame, not the popup's.
	fr := content.FrameRect()
	content.SetFrameRect(graphics.NewRect(popupPadding, float32(popupTitleH+popupPadding), fr.Width, fr.Height))
	content.ValidateLayout()

	pt := spin.PointToRoot(0, 0)
	sf := spin.FrameRect()
	x := pt.X + sf.Width - 12.5
	y := pt.Y + sf.Height/4
	if !up {
		y = pt.Y + sf.Height*3/4
	}
	w.HandleEvent(&platform.Event{Type: platform.EventMouseDown, Pos: graphics.NewPoint(x, y), Button: 0})
	w.HandleEvent(&platform.Event{Type: platform.EventMouseUp, Pos: graphics.NewPoint(x, y), Button: 0})
	return spin.Value
}

// With the touch floor ON — the case where the expansion applies — the up arrow must still step UP.
func TestSpinboxArrowsHitTheirOwnHalvesWithTheTouchFloor(t *testing.T) {
	if got := clickSpinboxArrow(t, true, true); got != 6 {
		t.Errorf("a click on the UP arrow gave %v, want 6 (the touch floor shifted the local y)", got)
	}
	if got := clickSpinboxArrow(t, false, true); got != 4 {
		t.Errorf("a click on the DOWN arrow gave %v, want 4", got)
	}
}

// And with the floor off, the same clicks must land the same way.
func TestSpinboxArrowsHitTheirOwnHalvesWithoutTheTouchFloor(t *testing.T) {
	if got := clickSpinboxArrow(t, true, false); got != 6 {
		t.Errorf("a click on the UP arrow gave %v, want 6", got)
	}
	if got := clickSpinboxArrow(t, false, false); got != 4 {
		t.Errorf("a click on the DOWN arrow gave %v, want 4", got)
	}
}

// The floor must still do its job: a click just OUTSIDE a small control (but inside the touch
// slop) reaches it, with coordinates that describe the outside honestly (negative here).
func TestTheTouchFloorStillWidensTheHitArea(t *testing.T) {
	w := meui.NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	w.SetTouchProfile(true)

	plate := meui.NewPanel()
	plate.SetFrameRect(graphics.NewRect(0, 0, 300, 200))
	w.AddChild(plate)
	spin := NewSpinBox()
	spin.SetRange(0, 10)
	spin.SetValue(5)
	spin.SetDecimals(0)
	spin.SetFrameRect(graphics.NewRect(100, 100, 100, 22))
	plate.AddChild(spin)
	plate.ValidateLayout()

	// 5px ABOVE the control's real top, over the arrow strip: inside the expanded hit frame,
	// outside the real one. The local y is NEGATIVE there, which is honest: it is still the up
	// arrow's half, not the down arrow's.
	if !plate.OnMouseDown(187.5, 95, 0) {
		t.Fatal("a touch 5px above the control did not reach it")
	}
	if spin.Value != 6 {
		t.Errorf("the touch above the control gave %v, want the UP step (6)", spin.Value)
	}
}
