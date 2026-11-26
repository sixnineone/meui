package web

// The responsive contract: a scene reflows from the WINDOW's resize event, not from a
// poll, and the breakpoint helper picks the layout for the width it is given.

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/platform"
)

const responsiveScene = `
Window {
  width: 1280
  height: 900
  color: "#FFFFFF"
  Rectangle {
    id: page
    anchors.fill: parent
    color: "#FFFFFF"
    Rectangle {
      id: sidebar
      x: 0
      y: 0
      width: 260
      height: 900
      color: "#FAFAFB"
    }
    Rectangle {
      id: article
      x: 260
      y: 0
      width: 1020
      height: 900
      color: "#FFFFFF"
    }
  }
}
`

func TestBreakpointsReflowOnResize(t *testing.T) {
	scene, err := Parse(responsiveScene, "", 1280, 900)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	compact := func(s *Scene, w, h int32) {
		s.Show("sidebar", false)
		s.SetFrame("article", 0, 0, float32(w), float32(h))
	}
	wide := func(s *Scene, w, h int32) {
		s.Show("sidebar", true)
		s.SetFrame("article", 260, 0, float32(w)-260, float32(h))
	}
	scene.Breakpoints([]Breakpoint{
		{MinWidth: 0, Apply: compact},
		{MinWidth: 1000, Apply: wide},
	})

	win := scene.Root.(*meui.Window)
	sidebar := scene.Get("sidebar").(*meui.Panel)
	article := scene.Get("article")
	if !sidebar.Visible() {
		t.Fatal("the wide layout should start with the sidebar visible")
	}
	if fr := article.FrameRect(); fr.X != 260 {
		t.Fatalf("article starts at x=%.0f, want 260", fr.X)
	}

	// A real resize event through the window, which is the path the adapter uses.
	win.HandleEvent(&platform.Event{Type: platform.EventResize, W: 900, H: 700})
	if sidebar.Visible() {
		t.Fatal("below 1000 the sidebar should be gone")
	}
	if fr := article.FrameRect(); fr.X != 0 || fr.Width != 900 {
		t.Fatalf("compact article = %.0f,%.0f %.0fx%.0f, want 0,0 900x700", fr.X, fr.Y, fr.Width, fr.Height)
	}

	// And back.
	win.HandleEvent(&platform.Event{Type: platform.EventResize, W: 1280, H: 900})
	if !sidebar.Visible() {
		t.Fatal("above 1000 the sidebar should be back")
	}
	if fr := article.FrameRect(); fr.X != 260 || fr.Width != 1020 {
		t.Fatalf("wide article = %.0f %.0f, want 260 1020", fr.X, fr.Width)
	}
}

// TestAResizeToTheSameSizeDoesNotReflow pins the "only on a change" rule: the paint path
// calls SetFrameRect constantly, and a breakpoint that reruns on every frame would be a
// layout pass per frame.
func TestAResizeToTheSameSizeDoesNotReflow(t *testing.T) {
	scene, err := Parse(responsiveScene, "", 1280, 900)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	calls := 0
	scene.Breakpoints([]Breakpoint{{MinWidth: 0, Apply: func(*Scene, int32, int32) { calls++ }}})
	if calls != 1 {
		t.Fatalf("the initial apply ran %d times, want 1", calls)
	}
	win := scene.Root.(*meui.Window)
	win.HandleEvent(&platform.Event{Type: platform.EventResize, W: 1280, H: 900})
	if calls != 1 {
		t.Fatalf("a resize to the SAME size reflowed (%d applies)", calls)
	}
	win.HandleEvent(&platform.Event{Type: platform.EventResize, W: 900, H: 900})
	if calls != 2 {
		t.Fatalf("a real resize did not reflow (%d applies)", calls)
	}
}
