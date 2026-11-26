package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

func newTestDialog() *AlertDialog {
	d := NewAlertDialog()
	d.SetTitle("Delete file")
	d.SetMessage("This cannot be undone.")
	d.SetActions([]DialogAction{
		{ID: "cancel", Label: "Cancel"},
		{ID: "delete", Label: "Delete", Primary: true, Destructive: true},
	})
	return d
}

// TestDialogAutoPicksTheFormBySizeClass is the whole point of the widget: an app writes ONE dialog
// and the surface decides whether it is centred or a sheet. It follows the SIZE CLASS and not the
// OS, so a narrow desktop window is a phone by the same rule a tablet is a tablet.
func TestDialogAutoPicksTheFormBySizeClass(t *testing.T) {
	d := newTestDialog()
	phone := graphics.NewRect(0, 0, 400, 800)
	tablet := graphics.NewRect(0, 0, 800, 900)
	desktop := graphics.NewRect(0, 0, 1100, 750)

	if !d.Sheet(phone) {
		t.Error("a 400-wide surface did not get the bottom-sheet form")
	}
	if d.Sheet(tablet) || d.Sheet(desktop) {
		t.Error("a surface with room got the sheet form; it should be centred")
	}

	// And the forced presentations override the size class in both directions.
	d.SetPresentation(DialogCentred)
	if d.Sheet(phone) {
		t.Error("DialogCentred still chose the sheet form")
	}
	d.SetPresentation(DialogSheet)
	if !d.Sheet(desktop) {
		t.Error("DialogSheet still chose the centred form")
	}
}

// TestDialogCentredIsCappedAndCentred pins the centred form's geometry: M3's 560 cap, a side margin,
// and the box in the middle of the area.
func TestDialogCentredIsCappedAndCentred(t *testing.T) {
	d := newTestDialog().SetPresentation(DialogCentred)

	wide := graphics.NewRect(0, 0, 1000, 800)
	p := d.PanelRect(wide)
	if p.Width != DialogMaxWidth {
		t.Errorf("on a 1000-wide area the box is %.0f wide, want the %v cap", p.Width, DialogMaxWidth)
	}
	if want := (float32(1000) - DialogMaxWidth) / 2; p.X != want {
		t.Errorf("the box starts at %.0f, want it centred at %.0f", p.X, want)
	}
	if want := (float32(800) - p.Height) / 2; p.Y != want {
		t.Errorf("the box starts at %.0f, want it centred at %.0f", p.Y, want)
	}

	// A narrow area: the margin limits the width, not the cap.
	narrow := graphics.NewRect(0, 0, 400, 800)
	if w := d.PanelRect(narrow).Width; w != 400-2*DialogSideMargin {
		t.Errorf("on a 400-wide area the box is %.0f wide, want %.0f", w, 400-2*DialogSideMargin)
	}
}

// TestDialogSheetIsAnchoredToTheBottom is the phone form: the same panel a BottomSheet draws, at the
// bottom edge, with the sheet's own cap.
func TestDialogSheetIsAnchoredToTheBottom(t *testing.T) {
	d := newTestDialog().SetPresentation(DialogSheet)
	area := graphics.NewRect(0, 0, 400, 800)
	p := d.PanelRect(area)
	if p.Y+p.Height != 800 {
		t.Errorf("the sheet ends at %.0f, want the area's bottom edge 800", p.Y+p.Height)
	}
	if p.Width != 400 {
		t.Errorf("the sheet is %.0f wide on a 400 area, want full width", p.Width)
	}
	if p.Height != d.contentHeight(p.Width) {
		t.Errorf("the sheet is %.0f tall but its content needs %.0f", p.Height, d.contentHeight(p.Width))
	}
}

// TestDialogActionsAreRightAlignedAndReachTheFloor is the M3 arrangement: the confirming action is
// last, on the right, and every action is a target a finger can hit.
func TestDialogActionsAreRightAlignedAndReachTheFloor(t *testing.T) {
	saved := theme.DefaultTheme
	t.Cleanup(func() { theme.SetDefaultTheme(saved) })
	th := *saved
	theme.SetDefaultTheme(&th)
	theme.DefaultTheme.Metrics.MinTouchTarget = theme.TouchTargetAndroid

	d := newTestDialog().SetPresentation(DialogCentred)
	bounds := graphics.NewRect(0, 0, 800, 600)
	p := d.PanelRect(bounds)

	r0, ok0 := d.ActionRect(0, bounds) // "Cancel"
	r1, ok1 := d.ActionRect(1, bounds) // "Delete", the primary
	if !ok0 || !ok1 {
		t.Fatal("an action has no rect")
	}
	if r1.X <= r0.X {
		t.Errorf("the primary action is at x=%.0f, left of the dismissive one at %.0f; M3 puts it last",
			r1.X, r0.X)
	}
	if got := p.X + p.Width - DialogPad; r1.X+r1.Width != got {
		t.Errorf("the rightmost action ends at %.0f, want the panel's inner edge %.0f", r1.X+r1.Width, got)
	}
	for i := range d.actions {
		r, _ := d.ActionRect(i, bounds)
		if r.Width < theme.TouchTargetAndroid || r.Height < theme.TouchTargetAndroid {
			t.Errorf("action %d is %.0fx%.0f, under the %v floor", i, r.Width, r.Height, theme.TouchTargetAndroid)
		}
	}
}

// TestDialogChoosesOnReleaseOfTheSameAction is the gesture contract, shared with the ActionSheet.
func TestDialogChoosesOnReleaseOfTheSameAction(t *testing.T) {
	d := newTestDialog().SetPresentation(DialogCentred)
	d.SetFrameRect(graphics.NewRect(0, 0, 800, 600))
	var chosen string
	d.OnAction = func(id string) { chosen = id }

	r, _ := d.ActionRect(1, d.FrameRect())
	cx, cy := r.X+r.Width/2, r.Y+r.Height/2
	if !d.OnMouseDown(cx, cy, 0) {
		t.Fatal("a press on an action was not taken")
	}
	d.OnMouseUp(cx, cy, 0)
	if chosen != "delete" {
		t.Errorf("chose %q, want delete", chosen)
	}
	if d.Visible() {
		t.Error("the dialog stayed up after an action")
	}
}

// TestDialogTapOutsideDismisses is the modal way out.
func TestDialogTapOutsideDismisses(t *testing.T) {
	d := newTestDialog().SetPresentation(DialogCentred)
	d.SetFrameRect(graphics.NewRect(0, 0, 800, 600))
	dismissed := false
	d.OnDismissed = func() { dismissed = true }

	if !d.OnMouseDown(10, 10, 0) {
		t.Error("a press outside the box was not taken")
	}
	if !dismissed || d.Visible() {
		t.Error("a press outside the box did not dismiss the dialog")
	}
	// And a press INSIDE the box, on no action, must not dismiss it.
	d2 := newTestDialog().SetPresentation(DialogCentred)
	d2.SetFrameRect(graphics.NewRect(0, 0, 800, 600))
	p2 := d2.PanelRect(d2.FrameRect())
	if d2.OnMouseDown(p2.X+p2.Width/2, p2.Y+10, 0) {
		t.Error("a press on the dialog's own body was treated as outside")
	}
	if !d2.Visible() {
		t.Error("a press on the dialog's body dismissed it")
	}
}

// TestDialogMeasuresAWrappedMessage is why the dialog borrows the Label: the box has to grow for a
// long message, and the wrap has ONE implementation (AGENTS.md).
func TestDialogMeasuresAWrappedMessage(t *testing.T) {
	d := newTestDialog().SetPresentation(DialogCentred)
	short := d.contentHeight(400)

	d.SetMessage("A much longer message that has to wrap onto several lines before it fits inside " +
		"the dialog's inner width, which is narrower than the text.")
	long := d.contentHeight(400)

	if long <= short {
		t.Errorf("a long message measured %.0f against %.0f for a short one — the wrap was not measured",
			long, short)
	}
}

// TestDialogIsAnInputTarget keeps it in the registry the touch audit checks.
func TestDialogIsAnInputTarget(t *testing.T) {
	if !meui.IsInteractive(NewAlertDialog()) {
		t.Error("an AlertDialog handles input but is not in the interactive registry")
	}
}

var _ meui.EventListener = (*AlertDialog)(nil)
var _ = platform.FormFactorPhone
