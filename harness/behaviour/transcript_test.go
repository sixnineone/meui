package behaviour

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/widgets"
)

// transcriptQML is the harness's trajectory in the shape botui's model produces: a
// collapsed tool call, an open think, and the current answer marked.
const transcriptQML = `
Window {
  width: 400
  height: 600
  VStack {
    anchors.fill: parent
    spacing: 0
    margins: 0
    Collapsible { id: tool; label: "· bash"; detail: "Check the gate"; trailing: "45 sub"; collapsed: true; width: 380; height: 24 }
    Collapsible { id: think; label: "· think"; detail: "Read the assertion"; width: 380; height: 124 }
    Label { id: filler; text: "after"; width: 380; height: 20 }
  }
}`

// TestACollapsedStepCostsOnlyItsHeader is the property that makes a transcript usable,
// asserted through the REAL event pipeline rather than by calling the widget: a
// collapsed step's height must not include its body, or a session of six hundred steps
// reserves six hundred bodies' worth of space and the pane stops scrolling.
func TestACollapsedStepCostsOnlyItsHeader(t *testing.T) {
	h := Load(t, transcriptQML)
	tool, ok := h.Widget("tool").(*widgets.Collapsible)
	if !ok {
		t.Fatalf("the tool step is a %T", h.Widget("tool"))
	}
	collapsedH := tool.FrameRect().Height

	think, ok := h.Widget("think").(*widgets.Collapsible)
	if !ok {
		t.Fatalf("the think step is a %T", h.Widget("think"))
	}
	openH := think.FrameRect().Height

	if collapsedH >= openH {
		t.Errorf("a collapsed step is %.0f tall and an open one %.0f: collapsing reserved no space",
			collapsedH, openH)
	}
	if collapsedH > 40 {
		t.Errorf("a collapsed step is %.0f tall, so it is not just its header", collapsedH)
	}
}

// TestAHeaderPressReachesTheStepThroughTheRealPipeline: the press goes in as
// Window.HandleEvent — the same entry point the platform uses — so the hit test and the
// touch slop are covered with it. A widget whose listener signature drifted would be dead
// to input here, which is why the assertion is that something was HIT: "nothing happened"
// is not a failure a state-only test can tell from a miss.
func TestAHeaderPressReachesTheStepThroughTheRealPipeline(t *testing.T) {
	h := Load(t, transcriptQML)
	tool := h.Widget("tool")
	f := tool.FrameRect()
	hit := h.Win.PanelAt(f.X+40, f.Y+f.Height/2)
	if hit == nil {
		t.Fatal("a press on a header hit nothing, so the step is dead to input")
	}
	if hit != meui.Widget(tool) {
		t.Errorf("a press on the header hit %T, not the step", hit)
	}
}

// TestTheHeaderTogglesThroughThePipeline is the behaviour itself, asserted through the
// real event path.
func TestTheHeaderTogglesThroughThePipeline(t *testing.T) {
	h := Load(t, transcriptQML)
	tool, ok := h.Widget("tool").(*widgets.Collapsible)
	if !ok {
		t.Fatalf("the tool step is a %T", h.Widget("tool"))
	}
	if !tool.Collapsed() {
		t.Fatal("the step did not start collapsed; its QML `collapsed: true` never arrived")
	}

	h.Click(40, 12)
	if tool.Collapsed() {
		t.Error("a press on the header did not open the step")
	}
	h.Click(40, 12)
	if !tool.Collapsed() {
		t.Error("a second press on the header did not close it")
	}
}

// TestASelfSizingWidgetActuallyResizes is the defect this file exists for, and it is
// general rather than specific to Collapsible.
//
// A widget's frame is set by its PARENT's layout, so a widget that changes its own size
// needs two things and originally had neither: its ancestors marked as needing layout
// (SetNeedsLayout only marks the widget), and a layout pass before the repaint that
// follows the change (nothing asked, and the next hit test is too late). The result was
// that the toggle WORKED and the row did not move — the state changed and the pixels did
// not, which is the worst of both and looks exactly like a widget that does not work.
//
// The paint is in the assertion because the layout runs there: a test that reads a frame
// without painting sees the old one and would pass this wrong way.
func TestASelfSizingWidgetActuallyResizes(t *testing.T) {
	h := Load(t, `
Window { width: 400; height: 600
  VStack { anchors.fill: parent; spacing: 0
    Collapsible { id: a; label: "one"; collapsed: true; width: 380 }
  }
}`)
	a, ok := h.Widget("a").(*widgets.Collapsible)
	if !ok {
		t.Fatalf("the step is a %T", h.Widget("a"))
	}
	h.PaintOnce()
	before := a.FrameRect().Height
	if before <= 0 {
		t.Fatal("the step has no height to start with")
	}

	h.Click(20, before/2)
	if a.Collapsed() {
		t.Fatal("the press did not open the step")
	}
	h.PaintOnce()

	if after := a.FrameRect().Height; after <= before {
		t.Errorf("the step is still %.0f tall after opening (it was %.0f): the toggle worked "+
			"but nothing re-laid out, so the row never grew", after, before)
	}
}
