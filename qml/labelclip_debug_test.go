package qml

import (
	"testing"

	"github.com/sixnineone/meui"
)

// TestLabelClipFrames reproduces the clipped-left-label bug from main.qml:
// an HStack row [Label(width 140), Button] renders the label's text shifted
// far left of its intended frame.
func TestLabelClipFrames(t *testing.T) {
	src := `
Window {
  width: 400
  height: 120
  VStack {
    margins: 8
    HStack {
      spacing: 12
      Label { id: lbl; text: "JLabel:"; width: 140 }
      Button { id: btn; text: "Enabled"; width: 100; height: 24 }
    }
  }
}
`
	widget, err := Convert(src)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	win, ok := widget.(*meui.Window)
	if !ok {
		t.Fatal("root is not a Window")
	}
	win.ValidateLayout()

	var walk func(w meui.Widget, depth int)
	walk = func(w meui.Widget, depth int) {
		fr := w.FrameRect()
		t.Logf("%*s%T frame=(%.1f,%.1f %.1fx%.1f)", depth*2, "", w, fr.X, fr.Y, fr.Width, fr.Height)
		if c, ok := w.(interface{ Children() []meui.Widget }); ok {
			for _, child := range c.Children() {
				walk(child, depth+1)
			}
		}
	}
	walk(win, 0)
}
