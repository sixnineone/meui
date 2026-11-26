package behaviour

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/widgets"
)

// TestAQmlTabStripInALayoutPaints settles a question left open twice: a TabStrip written in
// QML and placed in a VStack appeared to draw nothing in the demo, while the same widget
// painted under a direct harness. "Appeared to" is not a finding — a pixel count is.
//
// The suspicion was the shape that has already bitten this tree three times: a child with
// only Layout.fillWidth collapses to nothing, because the layout lays widgets out from
// their EXPLICIT size. So this asserts both halves: that the strip is given a real width by
// the layout, and that it draws into it.
func TestAQmlTabStripInALayoutPaints(t *testing.T) {
	const w, h = 400, 120
	// With an explicit width, because that is the documented model.
	withWidth := Load(t, `
Window { width: 400; height: 120
  VStack { anchors.fill: parent; spacing: 0
    TabStrip { id: strip; width: 380; height: 26 }
  }
}`)
	strip, ok := withWidth.Widget("strip").(*widgets.TabStrip)
	if !ok {
		t.Fatalf("the strip is a %T", withWidth.Widget("strip"))
	}
	strip.SetItems([]widgets.TabStripItem{{ID: "a", Label: "README.md"}, {ID: "b", Label: "MATRIX.md"}})
	strip.SetSelected(0)

	f := strip.FrameRect()
	if f.Width < 100 {
		t.Errorf("the strip is %.0f wide after layout; a control with no intrinsic size "+
			"collapses when only fillWidth is set", f.Width)
	}

	// Compare against the SAME strip with no items, because "drew a lot of pixels" is not
	// evidence of anything: a TabStrip paints its bar background, so a bare strip is
	// already thousands of non-empty pixels. The first version of this assertion checked
	// only that, and would have passed on a strip showing nothing but its own background —
	// which is exactly what the demo page does.
	draw := func(withItems bool) int {
		if withItems {
			strip.SetItems([]widgets.TabStripItem{
				{ID: "a", Label: "README.md"}, {ID: "b", Label: "MATRIX.md"}})
			strip.SetSelected(0)
		} else {
			strip.SetItems(nil)
		}
		dev := graphics.NewSoftDevice(w, h)
		strip.Paint(graphics.NewCanvas(dev), f)
		drawn := 0
		for _, p := range dev.Pixels() {
			if p != 0 {
				drawn++
			}
		}
		return drawn
	}
	bare, withItems := draw(false), draw(true)
	if withItems <= bare {
		t.Errorf("a strip with two item labels drew %d pixels against %d with none: the ITEMS "+
			"are not painted, only the bar", withItems, bare)
	}
	t.Logf("strip frame=%v bare=%d withItems=%d", f, bare, withItems)
}

// TestAQmlTextAreaAsAComposerPaints is the same question for the composer, which was the
// other thing recorded as "not confirmed".
func TestAQmlTextAreaAsAComposerPaints(t *testing.T) {
	const w, h = 400, 60
	h2 := Load(t, `
Window { width: 400; height: 60
  VStack { anchors.fill: parent; spacing: 0
    TextArea { id: c; width: 380; height: 44 }
  }
}`)
	ta, ok := h2.Widget("c").(*widgets.TextArea)
	if !ok {
		t.Fatalf("the composer is a %T", h2.Widget("c"))
	}
	ta.SetPlaceholder("Message or run a task")

	f := ta.FrameRect()
	dev := graphics.NewSoftDevice(w, h)
	ta.Paint(graphics.NewCanvas(dev), f)
	drawn := 0
	for _, p := range dev.Pixels() {
		if p != 0 {
			drawn++
		}
	}
	if drawn < 100 {
		t.Errorf("a 380x44 composer drew %d pixels: it is not painting", drawn)
	}
	t.Logf("composer frame=%v drawn=%d", f, drawn)
}
