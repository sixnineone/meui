package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

func newTestActionSheet() *ActionSheet {
	a := NewActionSheet()
	a.SetHeader("Photo", "What would you like to do?")
	a.SetItems([]ActionSheetItem{
		{ID: "save", Label: "Save"},
		{ID: "share", Label: "Share"},
		{ID: "delete", Label: "Delete", Destructive: true},
	})
	a.SetCancel("Cancel")
	a.SetFrameRect(graphics.NewRect(0, 0, 400, 800))
	return a
}

// TestActionSheetFitsItsContentAndDoesNotScroll is the structural rule: the sheet sizes itself to
// its actions. An action sheet that would need to scroll is one with too many choices — the
// guidance caps it at four buttons including Cancel, and a scrolling action sheet is a menu.
func TestActionSheetFitsItsContentAndDoesNotScroll(t *testing.T) {
	a := newTestActionSheet()
	bounds := a.FrameRect()
	p := a.PanelRect(bounds)

	if p.Height != a.ContentHeight() {
		t.Errorf("the panel is %.0f tall but its content needs %.0f — it would clip or scroll",
			p.Height, a.ContentHeight())
	}
	// Every row and the Cancel block have to be inside the panel, in order, and not overlap.
	prevBottom := p.Y
	for i := range a.items {
		r, ok := a.ItemRect(i, bounds)
		if !ok {
			t.Fatalf("no rect for action %d", i)
		}
		if r.Y < prevBottom-0.01 {
			t.Errorf("action %d starts at %.0f, above the previous row's bottom %.0f", i, r.Y, prevBottom)
		}
		if r.Y+r.Height > p.Y+p.Height+0.01 {
			t.Errorf("action %d ends at %.0f, past the panel's bottom %.0f", i, r.Y+r.Height, p.Y+p.Height)
		}
		prevBottom = r.Y + r.Height
	}
	cr, ok := a.CancelRect(bounds)
	if !ok {
		t.Fatal("no Cancel rect")
	}
	if cr.Y+cr.Height > p.Y+p.Height+0.01 {
		t.Errorf("Cancel ends at %.0f, past the panel's bottom %.0f", cr.Y+cr.Height, p.Y+p.Height)
	}
	if cr.Y < prevBottom {
		t.Errorf("Cancel starts at %.0f, overlapping the last action which ends at %.0f", cr.Y, prevBottom)
	}
	// And Cancel is the LAST thing in the panel, pinned to the bottom edge.
	if got := p.Y + p.Height - actionSheetPad; cr.Y+cr.Height != got {
		t.Errorf("Cancel ends at %.0f, want it pinned to %.0f", cr.Y+cr.Height, got)
	}
}

// TestActionSheetRowsReachTheTouchFloor applies the Phase A contract to the rows: every action is
// a target a finger aims at.
func TestActionSheetRowsReachTheTouchFloor(t *testing.T) {
	saved := theme.DefaultTheme
	t.Cleanup(func() { theme.SetDefaultTheme(saved) })
	th := *saved
	theme.SetDefaultTheme(&th)
	theme.DefaultTheme.Metrics.MinTouchTarget = theme.TouchTargetAndroid
	theme.DefaultTheme.Metrics.ControlHeight = 20 // deliberately small: the floor has to win

	a := newTestActionSheet()
	if h := a.RowHeight(); h < theme.TouchTargetAndroid {
		t.Errorf("a row is %.0f tall, under the %v floor", h, theme.TouchTargetAndroid)
	}
}

// TestActionSheetChoosesOnReleaseOfTheSameRow is the gesture contract: a choice commits when the
// finger lifts on the row it went down on, so a mis-touch is recoverable by sliding off.
func TestActionSheetChoosesOnReleaseOfTheSameRow(t *testing.T) {
	a := newTestActionSheet()
	var chosen string
	a.OnChoose = func(id string) { chosen = id }

	r1, _ := a.ItemRect(1, a.FrameRect())
	cx, cy := r1.X+r1.Width/2, r1.Y+r1.Height/2
	if !a.OnMouseDown(cx, cy, 0) {
		t.Fatal("a press on an action was not taken")
	}
	a.OnMouseUp(cx, cy, 0)
	if chosen != "share" {
		t.Errorf("chose %q, want share", chosen)
	}
	if a.Visible() {
		t.Error("the sheet stayed up after a choice")
	}
}

// TestActionSheetSlidingOffARowCancelsIt is the other half: a release on a DIFFERENT row must not
// choose either of them.
func TestActionSheetSlidingOffARowCancelsIt(t *testing.T) {
	a := newTestActionSheet()
	chosen := ""
	a.OnChoose = func(id string) { chosen = id }

	r0, _ := a.ItemRect(0, a.FrameRect())
	r2, _ := a.ItemRect(2, a.FrameRect())
	a.OnMouseDown(r0.X+r0.Width/2, r0.Y+r0.Height/2, 0)
	a.OnMouseUp(r2.X+r2.Width/2, r2.Y+r2.Height/2, 0)
	if chosen != "" {
		t.Errorf("a release on another row chose %q", chosen)
	}
	if !a.Visible() {
		t.Error("sliding off a row dismissed the sheet")
	}
}

// TestActionSheetCancelDismissesWithoutChoosing keeps the way out separate from the choices.
func TestActionSheetCancelDismissesWithoutChoosing(t *testing.T) {
	a := newTestActionSheet()
	chosen := ""
	a.OnChoose = func(id string) { chosen = id }

	cr, ok := a.CancelRect(a.FrameRect())
	if !ok {
		t.Fatal("no Cancel rect")
	}
	if !a.OnMouseDown(cr.X+cr.Width/2, cr.Y+cr.Height/2, 0) {
		t.Fatal("a press on Cancel was not taken")
	}
	a.OnMouseUp(cr.X+cr.Width/2, cr.Y+cr.Height/2, 0)
	if chosen != "" {
		t.Errorf("Cancel chose %q", chosen)
	}
	if a.Visible() {
		t.Error("Cancel did not dismiss the sheet")
	}
}

// TestActionSheetRefusesADisabledChoice guards the one row that looks available and is not.
func TestActionSheetRefusesADisabledChoice(t *testing.T) {
	a := NewActionSheet()
	a.SetItems([]ActionSheetItem{{ID: "x", Label: "Nope", Disabled: true}})
	a.SetFrameRect(graphics.NewRect(0, 0, 400, 800))
	chosen := ""
	a.OnChoose = func(id string) { chosen = id }

	r, _ := a.ItemRect(0, a.FrameRect())
	cx, cy := r.X+r.Width/2, r.Y+r.Height/2
	if a.OnMouseDown(cx, cy, 0) {
		a.OnMouseUp(cx, cy, 0)
	}
	if chosen != "" {
		t.Errorf("a disabled action was chosen (%q)", chosen)
	}
	if !a.Visible() {
		t.Error("pressing a disabled action dismissed the sheet")
	}
}

// TestActionSheetInheritsTheSheetsDismissals checks that the parts it does NOT reimplement still
// work: a modal action sheet dismisses on a tap above the panel, which is the sheet's job.
func TestActionSheetInheritsTheSheetsDismissals(t *testing.T) {
	a := newTestActionSheet()
	if !a.Modal() {
		t.Error("an action sheet is not modal")
	}
	if !a.OnMouseDown(200, 100, 0) {
		t.Error("a tap above the sheet was not taken")
	}
	if a.Visible() {
		t.Error("a tap above the sheet did not dismiss it")
	}
}

// TestActionSheetIsAnInputTarget keeps it in the registry the touch audit checks.
func TestActionSheetIsAnInputTarget(t *testing.T) {
	if !meui.IsInteractive(NewActionSheet()) {
		t.Error("an ActionSheet handles input but is not in the interactive registry")
	}
}

var _ meui.EventListener = (*ActionSheet)(nil)
var _ meui.HoverTarget = (*ActionSheet)(nil)
