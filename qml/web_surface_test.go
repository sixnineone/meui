package qml

// The web surface's QML contract, at the layer that broke it: a scene is a tree of
// Rectangles with ids and coordinates, and both used to be dead inside a Rectangle
// (nested ids were never registered; x/y were only applied by the non-bindings
// builder). A scene that lays out correctly in a screenshot can still be unwireable,
// which is why this asserts the REGISTRY and the FRAMES, not the picture.

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
)

func TestNestedIdsAndPositionsResolve(t *testing.T) {
	res, err := ParseWithBindings(`
Window {
  width: 400
  height: 300
  Rectangle {
    id: page
    anchors.fill: parent
    Rectangle {
      id: card
      x: 24
      y: 32
      width: 120
      height: 48
      radius: 8
      Rectangle {
        id: dot
        x: 8
        y: 8
        width: 16
        height: 16
        radius: 8
      }
      MouseArea {
        id: area
        anchors.fill: parent
      }
    }
    Text {
      id: label
      x: 24
      y: 120
      width: 200
      height: 20
      text: "hello"
    }
  }
}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	for _, id := range []string{"page", "card", "dot", "area", "label"} {
		if res.Widgets.Get(id) == nil {
			t.Errorf("nested id %q did not resolve — the scene cannot be wired by it", id)
		}
	}

	card := res.Widgets.Get("card")
	if card == nil {
		t.Fatal("no card")
	}
	if fr := card.FrameRect(); fr.X != 24 || fr.Y != 32 || fr.Width != 120 || fr.Height != 48 {
		t.Errorf("card frame = %.0f,%.0f %.0fx%.0f, want 24,32 120x48", fr.X, fr.Y, fr.Width, fr.Height)
	}
	if got := card.(*meui.Panel).Radius(); got != 8 {
		t.Errorf("radius = %.0f, want 8 (it used to be parsed and dropped)", got)
	}

	// anchors.fill resolves against the parent when the parent lays out, which is the
	// path a MouseArea over a card takes.
	card.(*meui.Panel).ValidateLayout()
	area := res.Widgets.Get("area")
	if area == nil {
		t.Fatal("no MouseArea")
	}
	if fr := area.FrameRect(); fr.Width != 120 || fr.Height != 48 {
		t.Errorf("MouseArea anchors.fill gave %.0fx%.0f, want 120x48", fr.Width, fr.Height)
	}
}

// TestObjectLiteralPropertiesParse pins the `{ … }` form: the parser keeps it as
// SOURCE TEXT, and a handler that only accepts a map silently does nothing — which
// is how `border: { … }` and `gradient: { … }` both painted nothing.
func TestObjectLiteralPropertiesParse(t *testing.T) {
	if _, ok := objectLiteral(`{ from: "#7FC0E8", to: "#EAF7FC", angle: 180 }`); !ok {
		t.Fatal("a gradient object literal was not parsed")
	}
	if _, ok := objectLiteral(`{ color: "#14243C", width: 2 }`); !ok {
		t.Fatal("a border object literal was not parsed")
	}
	g, ok := parseGradient(`{ from: "#7FC0E8", to: "#EAF7FC", angle: 180 }`)
	if !ok {
		t.Fatal("parseGradient rejected a valid literal")
	}
	if g.Stops[0].Offset != 0 || g.Stops[len(g.Stops)-1].Offset != 1 {
		t.Fatalf("gradient stops = %+v, want 0..1", g.Stops)
	}
	if g.Start.Y >= g.End.Y {
		t.Fatalf("angle 180 should run top-to-bottom, got %v -> %v", g.Start, g.End)
	}
}

// TestZSetsSiblingOrder pins the `z:` plumbing: it is applied by the COMMON property
// pass, so an overlay works on any widget, not just a Rectangle, and an unknown shape
// of the setter (the fluent trap) would leave it silently at zero.
func TestZSetsSiblingOrder(t *testing.T) {
	res, err := ParseWithBindings(`
Window {
  width: 200
  height: 120
  Rectangle {
    id: page
    anchors.fill: parent
    color: "#FFFFFF"
    Rectangle { id: back; x: 0; y: 0; width: 60; height: 60; color: "#FF0000" }
    Rectangle { id: front; x: 20; y: 20; width: 60; height: 60; color: "#0000FF"; z: 2 }
    Text { id: label; x: 100; y: 10; width: 80; height: 20; text: "hi"; z: -1 }
  }
}
`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	get := func(id string) meui.Widget {
		w := res.Widgets.Get(id)
		if w == nil {
			t.Fatalf("id %q did not register", id)
		}
		return w
	}
	if z := get("front").(interface{ ZIndex() float32 }).ZIndex(); z != 2 {
		t.Fatalf("front z = %v, want 2", z)
	}
	if z := get("back").(interface{ ZIndex() float32 }).ZIndex(); z != 0 {
		t.Fatalf("back z = %v, want 0", z)
	}
	if z := get("label").(interface{ ZIndex() float32 }).ZIndex(); z != -1 {
		t.Fatalf("label z = %v, want -1", z)
	}
}

// TestEightDigitColoursAreAARRGGBB pins the alpha bug: ColorDecode read "#CCFFFFFF" as
// 0x00FFFFCC (alpha 0), so every translucent colour a scene wrote was invisible. The
// parsing is the seam's layout: AA RR GG BB.
func TestEightDigitColoursAreAARRGGBB(t *testing.T) {
	if got := parseColor("#80FF0000"); got != graphics.Color(0x80FF0000) {
		t.Fatalf("#80FF0000 decoded to %08X, want 80FF0000", uint32(got))
	}
	if got := parseColor("#CCFFFFFF"); got != graphics.Color(0xCCFFFFFF) {
		t.Fatalf("#CCFFFFFF decoded to %08X, want CCFFFFFF", uint32(got))
	}
	// Six digits keep their meaning (opaque).
	if got := parseColor("#4F46E5"); got != graphics.Color(0xFF4F46E5) {
		t.Fatalf("#4F46E5 decoded to %08X, want FF4F46E5", uint32(got))
	}
}
