package behaviour

import (
	"testing"

	"github.com/sixnineone/meui/layout"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/widgets"
)

// TestACheckboxInsideAModalPopupIsHitWhereItIsDrawn is the reported bug in the popup path the app
// actually uses: a modal popup (the QML `Dialog` widget) whose content is a vertical stack of
// controls, clicked through the window.
//
// Two controls sit one above the other. A click on the FIRST must reach the first: if the popup
// layer's position is applied twice, or not at all, the local point is the layer's Y away from where
// the control is drawn, and the control BELOW answers — which is exactly the report.
func TestACheckboxInsideAModalPopupIsHitWhereItIsDrawn(t *testing.T) {
	h := Load(t, `Window { width: 400; height: 600
		Button { id: open; text: "open"; width: 120; height: 30 }
	}`)

	c1 := widgets.NewCheckBox()
	c1.SetText("First")
	c2 := widgets.NewCheckBox()
	c2.SetText("Second")
	content := meui.NewPanel()
	content.SetLayout(&layout.VStack{Spacing: 0})
	content.AddChild(c1)
	content.AddChild(c2)
	// NewDialog takes no arguments; the content is attached through its own constructor's API shape
	// (NewModalPopupPanel(title, heading, buttons)), so this uses the same one the message box uses.
	panel := widgets.NewModalPopupPanel("Pick", "", content)

	h.Win.ShowModal(panel, 280, 260)

	frame := panel.FrameRect()
	if frame.X == 0 && frame.Y == 0 {
		t.Fatal("the popup is at the origin; this test is meant to show it centred")
	}

	// The popup lays its content out during PAINT (its Paint sets the content's frame), so a paint
	// pass is what makes the controls be where the popup is going to dispatch to. Without it the
	// frames are still the constructor defaults and the aim would be a lie.
	c := graphics.NewCanvas(graphics.NewTestDevice(400, 600))
	h.Win.Paint(c, h.Win.FrameRect())

	r := c1.FrameRect()
	t.Logf("panel=%v content=%v c1=%v root=%v", frame, content.FrameRect(), r, c1.PointToRoot(r.Width/2, r.Height/2))
	tapAt(h, c1.PointToRoot(r.Width/2, r.Height/2))

	if !c1.Checked {
		t.Errorf("clicking the first control did not check it")
	}
	if c2.Checked {
		t.Error("clicking the first control checked the second — the popup's events are offset")
	}
}
