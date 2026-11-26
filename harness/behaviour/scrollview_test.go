package behaviour

// ScrollView is the web page's container: a viewport onto ONE content widget that
// is taller than it. ScrollPane already had the mechanics; this pins the QML surface
// — the element name, the declared content size, the frameless mode — and the wheel
// path through the real window, because a page you cannot scroll is not a page.

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/widgets"
)

const scrollViewQML = `
Window {
  width: 400
  height: 300

  ScrollView {
    id: view
    anchors.fill: parent
    frameless: true
    contentHeight: 900
    contentWidth: 400

    Rectangle {
      id: page
      width: 400
      height: 900
      color: "#FCF4DC"

      Text {
        id: bottom
        x: 20
        y: 820
        width: 300
        height: 24
        text: "the bottom of the page"
      }
    }
  }
}
`

func TestScrollViewScrollsAPage(t *testing.T) {
	h := Load(t, scrollViewQML)
	h.Settle()

	sp, ok := h.Widget("view").(*widgets.ScrollPane)
	if !ok {
		t.Fatalf("view is %T, want *widgets.ScrollPane", h.Widget("view"))
	}
	if _, offY := sp.ScrollOffsets(); offY != 0 {
		t.Fatalf("a page starts at offset 0, got %v", offY)
	}
	if sp.VScroll == nil || !sp.VScroll.Visible() {
		t.Fatal("a 900px page in a 300px viewport shows no vertical scrollbar")
	}

	// The wheel goes to the deepest hit, which inside a ScrollPane is the pane itself
	// (it owns the scrollbars); the content travels with the offset.
	h.Wheel(200, 150, 0, -1)
	h.Settle()
	_, offY := sp.ScrollOffsets()
	if offY <= 0 {
		t.Fatalf("wheel down did not scroll the page (offset %v)", offY)
	}

	// Slam past the end: the offset clamps to Max-VisibleAmount (the bar takes a few
	// pixels of the viewport, so the limit is the host's, not 900-300).
	maxScroll := sp.VScroll.Max - sp.VScroll.VisibleAmount
	h.Wheel(200, 150, 0, -1000)
	h.Settle()
	_, endY := sp.ScrollOffsets()
	if endY != maxScroll {
		t.Fatalf("scrolled to %v, want the clamp at %v", endY, maxScroll)
	}

	// And back up.
	h.Wheel(200, 150, 0, 1000)
	h.Settle()
	if _, upY := sp.ScrollOffsets(); upY != 0 {
		t.Fatalf("scrolled back to %v, want 0", upY)
	}
}

// TestScrollViewKeepsItsContentHitTestable: the pane returns ITSELF from PanelAt so
// its scrollbars get honest coordinates — which means a click on the page must be
// forwarded to the content, not swallowed by the viewport.
func TestScrollViewKeepsItsContentHitTestable(t *testing.T) {
	h := Load(t, `
Window {
  width: 400
  height: 300
  ScrollView {
    id: view
    anchors.fill: parent
    frameless: true
    contentHeight: 600
    contentWidth: 400
    Rectangle {
      id: page
      width: 400
      height: 600
      color: "#FFFFFF"
      Rectangle {
        id: cta
        x: 40
        y: 40
        width: 160
        height: 44
        color: "#14243C"
        clickable: true
      }
    }
  }
}`)
	h.Settle()
	sp := h.Widget("view").(*widgets.ScrollPane)
	// The CONCRETE type: Panel's setters are fluent (they return *Panel), so an
	// interface assertion for a no-return setter never matches — the trap AGENTS.md
	// documents and the touch floor lost two years to.
	cta := h.Widget("cta").(*meui.Panel)
	clicks := 0
	cta.SetOnClick(func() { clicks++ })

	h.Click(120, 60)
	if clicks != 1 {
		t.Fatalf("a click inside the scrolled page reached the box %d times, want 1", clicks)
	}
	_ = sp
}
