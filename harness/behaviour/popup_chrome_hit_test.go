package behaviour

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/layout"
	"github.com/sixnineone/meui/widgets"
)

// TestACustomPopupIsHitWhereItIsDrawnUnderAChromeBar is the reported case: the app shows its OWN
// widget through `Window.ShowModal`, in a window with a title bar, and the click answers with the
// control BELOW the one under the pointer.
//
// The offset is reported as the BAR's height, and the bar is the one thing that separates the two
// coordinate spaces: the chrome, which is window space, and the content tree, which is 0-based and
// starts below it. The adapter converts pointer events once (subtracting the bar) and the layers are
// painted with the bar added back. Anything on the layer path that converts a second time — or not at
// all — lands exactly one bar height away from where the control is drawn.
func TestACustomPopupIsHitWhereItIsDrawnUnderAChromeBar(t *testing.T) {
	const barH = float32(32)

	h := Load(t, `Window { width: 400; height: 600
		Button { id: open; text: "open"; width: 120; height: 30 }
	}`)

	// What the host does when it installs a title bar: the content is laid out for h-bar, and the
	// tree is told where its (0,0) now sits.
	h.Win.SetFrameRect(graphics.NewRect(0, 0, 400, 600-barH))
	h.Win.SetContentOrigin(0, barH)

	// The app's own popup: a vertical stack of controls, shown through ShowModal.
	c1 := widgets.NewCheckBox()
	c1.SetText("First")
	c2 := widgets.NewCheckBox()
	c2.SetText("Second")
	content := meui.NewPanel()
	content.SetLayout(&layout.VStack{Spacing: 0})
	content.AddChild(c1)
	content.AddChild(c2)

	h.Win.ShowModal(content, 280, 200)

	// The popup lays its content out during Paint, so paint once: without it the frames are still
	// constructor defaults and any aim would be a lie.
	canvas := graphics.NewCanvas(graphics.NewTestDevice(400, 600))
	h.Win.Paint(canvas, graphics.NewRect(0, 0, 400, 600))

	r := c1.FrameRect()
	if r.Y == 0 && c2.FrameRect().Y == 0 {
		t.Fatalf("the two controls overlap at y=0; the popup did not lay its content out")
	}
	pt := c1.PointToRoot(r.Width/2, r.Height/2)
	t.Logf("popup=%v c1=%v c2=%v tap=%v", content.FrameRect(), r, c2.FrameRect(), pt)

	tapAt(h, pt)
	if !c1.Checked {
		t.Errorf("clicking the first control did not check it")
	}
	if c2.Checked {
		t.Errorf("clicking the first control checked the second: the popup's events are one chrome "+
			"bar (%v) away from where its content is drawn", barH)
	}
}
