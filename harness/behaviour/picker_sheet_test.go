package behaviour

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/widgets"
)

// pickerQML is one date field. The window's size is what decides the presentation, so the same
// scene is loaded into a phone-shaped window and a roomy one.
const pickerQML = `
Window {
  width: 360
  height: 640
  DatePicker { id: dp; width: 200; height: 28 }
}`

// TestDatePickerLiftsIntoASheetOnAPhone is the plan's own rule for the pickers: on a phone the
// calendar belongs in a bottom sheet the thumb can reach, not in a dropdown under a field — and on
// a surface with room the dropdown is the right control. One widget, chosen by the size class.
func TestDatePickerLiftsIntoASheetOnAPhone(t *testing.T) {
	h := Load(t, pickerQML)
	// Size the window and lay it out, but do NOT Settle: Settle paints, and the harness paints
	// through a 1280x800 device, which resets the window's frame — so the size the presentation
	// reads would not be the size this test set.
	h.Win.SetFrameRect(graphics.NewRect(0, 0, 360, 640))
	h.Win.ValidateLayout()

	dp, ok := h.Widget("dp").(*widgets.DatePicker)
	if !ok {
		t.Fatalf("dp is %T, want *widgets.DatePicker", h.Widget("dp"))
	}
	dp.SetExpanded(true)

	r := dp.PanelRect()
	if r.Width <= 0 || r.Height <= 0 {
		t.Fatalf("the panel did not open (rect %v)", r)
	}
	if r.Y+r.Height != 640 {
		t.Errorf("the panel ends at %.0f, want the window's bottom edge 640", r.Y+r.Height)
	}
	if want := (float32(360) - r.Width) / 2; r.X != want {
		t.Errorf("the panel starts at %.0f, want it centred at %.0f", r.X, want)
	}
	if r.Width != 360 {
		t.Errorf("the sheet is %.0f wide on a 360 surface, want full width", r.Width)
	}
}

// TestDatePickerStaysADropdownOnAWideSurface is the other direction, and the one that must not
// regress: a desktop window keeps the anchored dropdown it always had.
func TestDatePickerStaysADropdownOnAWideSurface(t *testing.T) {
	h := Load(t, pickerQML)
	h.Win.SetFrameRect(graphics.NewRect(0, 0, 1100, 750))
	h.Win.ValidateLayout()

	dp, ok := h.Widget("dp").(*widgets.DatePicker)
	if !ok {
		t.Fatalf("dp is %T, want *widgets.DatePicker", h.Widget("dp"))
	}
	dp.SetExpanded(true)

	r := dp.PanelRect()
	if r.Width <= 0 || r.Height <= 0 {
		t.Fatalf("the panel did not open (rect %v)", r)
	}
	field := dp.FrameRect()
	if want := field.Y + field.Height; r.Y != want {
		t.Errorf("the panel starts at %.0f, want it hanging from the field's bottom edge %.0f", r.Y, want)
	}
	if r.Width != 238 {
		t.Errorf("the dropdown is %.0f wide, want the calendar's own 238 — a sheet takes the surface's width",
			r.Width)
	}
}

// TestDatePickerPresentationCanBeForced keeps the escape hatch honest: an app that knows better can
// force either form, in either direction.
func TestDatePickerPresentationCanBeForced(t *testing.T) {
	h := Load(t, pickerQML)
	// Size the window and lay it out, but do NOT Settle: Settle paints, and the harness paints
	// through a 1280x800 device, which resets the window's frame — so the size the presentation
	// reads would not be the size this test set.
	h.Win.SetFrameRect(graphics.NewRect(0, 0, 360, 640))
	h.Win.ValidateLayout()
	dp := h.Widget("dp").(*widgets.DatePicker)

	// A phone told to use the dropdown.
	dp.SetPresentation(widgets.PickerDropdown)
	dp.SetExpanded(true)
	r := dp.PanelRect()
	field := dp.FrameRect()
	if r.Y != field.Y+field.Height {
		t.Errorf("PickerDropdown on a phone put the panel at y=%.0f, want the field's bottom %.0f",
			r.Y, field.Y+field.Height)
	}
	dp.SetExpanded(false)

	// A phone told to use the sheet — which is what Auto already chose, but forcing it must agree.
	dp.SetPresentation(widgets.PickerSheet)
	dp.SetExpanded(true)
	r = dp.PanelRect()
	if r.Y+r.Height != 640 {
		t.Errorf("PickerSheet put the panel's bottom at %.0f, want 640", r.Y+r.Height)
	}
}
