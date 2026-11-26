package behaviour

// The web layer's interaction contract, driven through the REAL input pipeline
// (Window.HandleEvent), not by calling handlers directly:
//
//   - a clickable box (Rectangle { clickable: true }) clicks even when its LABEL is
//     the deepest hit, because a press the label declines bubbles to its ancestor;
//   - hover reaches the box through the HoverTarget walk, not only when the box is
//     the hit itself;
//   - Enter/Space activate a focused box, which is what a keyboard user means by
//     "click the card";
//   - a MouseArea is an input region independent of what it sits on;
//   - a press that ends outside the box does not click it.

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
)

const clickScene = `
Window {
  width: 400
  height: 300
  Rectangle {
    id: page
    anchors.fill: parent
    color: "#FFFFFF"

    Rectangle {
      id: cta
      x: 40
      y: 40
      width: 200
      height: 48
      radius: 8
      color: "#14243C"
      clickable: true

      Text {
        x: 0
        y: 0
        width: 200
        height: 48
        text: "Click me"
        color: "#FFFFFF"
        horizontalAlignment: "center"
      }
    }

    Text {
      id: status
      x: 40
      y: 130
      width: 320
      height: 24
      text: "idle"
    }

    Rectangle {
      id: marea
      x: 40
      y: 200
      width: 160
      height: 40
      color: "#FF6B9A"

      MouseArea {
        id: area
        anchors.fill: parent
        cursor: "pointing"
      }
    }
  }
}
`

func TestAClickableBoxClicksThroughItsLabel(t *testing.T) {
	h := Load(t, clickScene)
	h.PaintOnce()
	cta, ok := h.Widget("cta").(*meui.Panel)
	if !ok {
		t.Fatalf("cta is %T, want *meui.Panel", h.Widget("cta"))
	}
	clicks := 0
	cta.SetOnClick(func() { clicks++ })

	// (140,64) is inside the Text child, which is the deepest hit and not itself
	// interactive: this is the case that made clickable cards inert.
	h.Click(140, 64)
	if clicks != 1 {
		t.Fatalf("click through the label fired %d times, want 1", clicks)
	}

	// A press outside does nothing.
	h.Click(380, 280)
	if clicks != 1 {
		t.Fatalf("a click outside the box fired it (%d)", clicks)
	}
}

func TestAClickableBoxDoesNotClickWhenTheReleaseIsOutside(t *testing.T) {
	h := Load(t, clickScene)
	h.PaintOnce()
	cta := h.Widget("cta").(*meui.Panel)
	clicks := 0
	cta.SetOnClick(func() { clicks++ })

	h.Send(mouseDown(140, 64, 1))
	if !cta.Pressed() {
		t.Fatal("press inside did not arm the box")
	}
	h.Send(mouseUp(380, 280, 1))
	if clicks != 0 {
		t.Fatalf("a release outside still clicked the box (%d)", clicks)
	}
	if cta.Pressed() {
		t.Fatal("the box stayed pressed after the release")
	}
}

func TestHoverReachesTheBoxUnderTheLabel(t *testing.T) {
	h := Load(t, clickScene)
	h.PaintOnce()
	cta := h.Widget("cta").(*meui.Panel)

	h.Send(&platform.Event{Type: platform.EventMouseMove, Pos: graphics.Point{X: 140, Y: 64}})
	if !cta.Hovered() {
		t.Fatal("hover over the label did not reach the clickable box")
	}
	h.Send(&platform.Event{Type: platform.EventMouseMove, Pos: graphics.Point{X: 370, Y: 280}})
	if cta.Hovered() {
		t.Fatal("the box stayed hovered after the pointer left it")
	}
}

func TestAClickableBoxActivatesWithSpace(t *testing.T) {
	h := Load(t, clickScene)
	h.PaintOnce()
	cta := h.Widget("cta").(*meui.Panel)
	clicks := 0
	cta.SetOnClick(func() { clicks++ })

	h.Click(140, 64) // focuses the nearest focusable widget, which is the box
	if !cta.Focused() {
		t.Fatalf("clicking the box did not focus it (focused=%v, focusable=%v)", cta.Focused(), cta.Focusable())
	}
	h.KeyDown(platform.KeySpace, 0)
	if clicks != 2 {
		t.Fatalf("Space on a focused box: %d clicks, want 2 (one from the click, one from the key)", clicks)
	}
	h.KeyDown(platform.KeyReturn, 0)
	if clicks != 3 {
		t.Fatalf("Enter on a focused box: %d clicks, want 3", clicks)
	}
}

func TestAMouseAreaIsAnIndependentInputRegion(t *testing.T) {
	h := Load(t, clickScene)
	h.PaintOnce()
	area, ok := h.Widget("area").(*meui.MouseArea)
	if !ok {
		t.Fatalf("area is %T, want *meui.MouseArea", h.Widget("area"))
	}
	clicks := 0
	area.OnClicked = func() { clicks++ }

	h.Click(120, 220)
	if clicks != 1 {
		t.Fatalf("the MouseArea fired %d times, want 1", clicks)
	}
	// Outside its region.
	h.Click(120, 260)
	if clicks != 1 {
		t.Fatalf("a click outside the MouseArea fired it (%d)", clicks)
	}
	// The area's parent box is not clickable, so nothing else consumed it.
	box := h.Widget("marea").(*meui.Panel)
	if box.Pressed() {
		t.Fatal("the painted box reported a press it never took")
	}
}
