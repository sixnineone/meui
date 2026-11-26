package behaviour

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/widgets"
)

// TestAnAlertDialogAwayFromTheOriginFiresTheActionUnderThePoint reproduces the reported bug exactly:
// a popup shown CENTRED is never at the origin, its layer hands its widget local coordinates, and a
// hit test that compares those local coordinates against the widget's PARENT-RELATIVE frame lands
// everything the layer's Y too low — so tapping the first action fired the one below it.
//
// A widget test that shows its dialog at (0,0) cannot tell the two apart; only a real layer, with the
// dialog offset inside the window, disagrees. The tap goes through Window.HandleEvent, which is the
// same entry point the platform uses.
func TestAnAlertDialogAwayFromTheOriginFiresTheActionUnderThePoint(t *testing.T) {
	h := Load(t, `Window { width: 400; height: 600
		Button { id: open; text: "open"; width: 120; height: 30 }
	}`)

	d := widgets.NewAlertDialog()
	d.SetTitle("Delete")
	d.SetMessage("Delete this item?")
	d.SetActions([]widgets.DialogAction{
		{ID: "delete", Label: "Delete", Destructive: true},
		{ID: "cancel", Label: "Cancel"},
	})
	fired := ""
	d.OnAction = func(id string) { fired = id }
	h.Win.ShowModal(d, 300, 220)

	frame := d.FrameRect()
	if frame.X == 0 && frame.Y == 0 {
		t.Fatal("the dialog is at the origin; this test is meant to show it centred")
	}

	// The action's rect is computed from the dialog's LOCAL space — the space the event methods are
	// handed — and the tap is converted back to the window's space through PointToRoot, which walks
	// the widget's parent chain (the layer widget's parent is the window).
	local := graphics.NewRect(0, 0, frame.Width, frame.Height)
	r, ok := d.ActionRect(0, local)
	if !ok {
		t.Fatal("no rect for the first action")
	}

	tapAt(h, d.PointToRoot(r.X+r.Width/2, r.Y+r.Height/2))
	if fired != "delete" {
		t.Errorf("tapping the first action fired %q, want delete — a centred dialog's hit test is "+
			"offset by its layer position when FrameRect is used for containment", fired)
	}
}
