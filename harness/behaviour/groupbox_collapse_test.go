package behaviour

import (
	"testing"

	"github.com/sixnineone/meui/widgets"
)

const collapseQML = `
Window {
    id: win
    width: 800
    height: 900
    VStack {
        spacing: 12
        margins: 12
        GroupBox {
            id: gb1
            title: "Information"
            collapsible: true
            fullTitle: true
            width: 200
            height: 220
            VStack {
                Label { text: "info message" }
                Label { text: "second line" }
            }
        }
        GroupBox {
            id: gb2
            title: "Warning"
            collapsible: true
            fullTitle: true
            width: 200
            height: 220
            VStack {
                Label { text: "warning message" }
            }
        }
        Label { id: below; text: "content below" }
    }
}
`

// TestGroupBoxCollapseRestore: collapsing shrinks the frame and pulls
// siblings up; expanding must restore the ORIGINAL frame height and push
// siblings back to their original positions.
func TestGroupBoxCollapseRestore(t *testing.T) {
	h := Load(t, collapseQML)
	gb1 := h.Widget("gb1").(*widgets.GroupBox)
	gb2 := h.Widget("gb2").(*widgets.GroupBox)
	h.Settle()

	origH1 := gb1.FrameRect().Height
	origY2 := gb2.FrameRect().Y
	if origH1 < 100 {
		t.Fatalf("precondition: gb1 should be expanded (h=%v)", origH1)
	}

	// Click the title bar (top of gb1) to collapse.
	x1, _ := Center(gb1)
	h.Click(x1, gb1.FrameRect().Y+10)
	h.Settle()

	collapsedH := gb1.FrameRect().Height
	if collapsedH >= origH1 {
		t.Errorf("collapse: height %v should shrink below %v", collapsedH, origH1)
	}
	if y2 := gb2.FrameRect().Y; y2 >= origY2 {
		t.Errorf("collapse: sibling gb2 should move up: %v -> %v", origY2, y2)
	}

	// Click again to expand.
	h.Click(x1, gb1.FrameRect().Y+10)
	h.Settle()

	if got := gb1.FrameRect().Height; got != origH1 {
		t.Errorf("expand: height %v, want original %v", got, origH1)
	}
	if y2 := gb2.FrameRect().Y; y2 != origY2 {
		t.Errorf("expand: sibling gb2 Y %v, want original %v", y2, origY2)
	}
}

// TestGroupBoxCollapsedFromQML: a GroupBox starting collapsed in QML must
// expand to its content's preferred height (no stale saved height).
func TestGroupBoxCollapsedFromQML(t *testing.T) {
	h := Load(t, `
Window {
    width: 400
    height: 500
    VStack {
        spacing: 8
        margins: 8
        GroupBox {
            id: gbx
            title: "Starts collapsed"
            collapsible: true
            collapsed: true
            fullTitle: true
            width: 300
            height: 200
            VStack {
                Label { text: "row 1" }
                Label { text: "row 2" }
                Label { text: "row 3" }
            }
        }
        Label { id: after; text: "after" }
    }
}
`)
	gbx := h.Widget("gbx").(*widgets.GroupBox)
	after := h.Widget("after")
	h.Settle()

	x, _ := Center(gbx)
	yCollapsed := after.FrameRect().Y
	hCollapSED := gbx.FrameRect().Height

	h.Click(x, gbx.FrameRect().Y+10) // expand
	h.Settle()

	if gbx.FrameRect().Height <= hCollapSED {
		t.Errorf("expand: height %v should grow beyond collapsed %v",
			gbx.FrameRect().Height, hCollapSED)
	}
	if after.FrameRect().Y <= yCollapsed {
		t.Errorf("expand: 'after' label should move down: %v -> %v",
			yCollapsed, after.FrameRect().Y)
	}

	// And re-collapse restores the compact state.
	h.Click(x, gbx.FrameRect().Y+10)
	h.Settle()
	if gbx.FrameRect().Height != hCollapSED {
		t.Errorf("re-collapse: height %v, want %v", gbx.FrameRect().Height, hCollapSED)
	}
}
