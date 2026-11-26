//go:build visual

package behaviour

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/widgets"
)

const visualQML = `
Window {
    width: 800
    height: 600
    VStack {
        margins: 20
        spacing: 16
        Button { id: btn; text: "Hover me"; width: 200 }
        TextField { id: field; width: 300 }
        CheckBox { id: cb; text: "check" }
    }
}
`

// TestVisualStateTransitions walks the demo interactions and captures
// a screenshot after every state transition:
//
//	idle -> hover(button) -> focus(field) -> typed text -> checkbox on
//
// The PNG sequence lands in behaviour/out/TestVisualStateTransitions/, and the
// transitions are asserted on the PIXELS as well as on widget state: the
// button's region must change when it is hovered, the field's when text
// arrives, and the checkbox's when it is toggled. The final frame is compared
// against testdata/golden/state_transitions.png.
func TestVisualStateTransitions(t *testing.T) {
	va := NewVisualApp(t, "TestVisualStateTransitions", visualQML, 800, 600)
	// Widget positions come from the live layout (read on the render
	// thread inside each step) so clicks land exactly on the widgets.
	btn := va.Reg.Get("btn")
	field := va.Reg.Get("field")
	cb := va.Reg.Get("cb")

	region := func(w meui.Widget) (int32, int32, int32, int32) {
		f := w.FrameRect()
		return int32(f.X), int32(f.Y), int32(f.Width), int32(f.Height)
	}

	var btnIdle, fieldEmpty, cbOff []graphics.Color
	var btnBox, fieldBox, cbBox [4]int32

	va.Run(t, []VisualStep{
		func(va *VisualApp) {
			// Step 0: idle frame (baseline).
		},
		func(va *VisualApp) {
			// The frame before this step is the idle one: keep the button's
			// pixels as the hover reference.
			x, y, w, h := region(btn)
			btnBox = [4]int32{x, y, w, h}
			btnIdle = va.Snapshot(x, y, w, h)
			cx, cy := Center(btn)
			va.Hover(cx, cy) // over the button
		},
		func(va *VisualApp) {
			// The frame before this step is the hovered one. The hover ink is
			// subtle, so this is a mean-delta check, not a pixel count.
			va.AssertRegionMeanChanged(btnIdle, btnBox[0], btnBox[1], btnBox[2], btnBox[3], 2,
				"the button render while hovered")
			va.Hover(5, 5) // off the button — hover-off transition
		},
		func(va *VisualApp) {
			x, y := Center(btn)
			va.Click(x, y) // click the button
		},
		func(va *VisualApp) {
			x, y := Center(field)
			va.Click(x, y) // focus the text field
		},
		func(va *VisualApp) {
			// The frame before this step is the focused, still empty field.
			x, y, w, h := region(field)
			fieldBox = [4]int32{x, y, w, h}
			fieldEmpty = va.Snapshot(x, y, w, h)
			va.Type("hello") // text renders per rune
		},
		va.Frames(1), // let the runes reach the frame
		func(va *VisualApp) {
			if f := field.(*widgets.TextField); f.Text != "hello" {
				t.Errorf("typed text did not reach the field: %q", f.Text)
			}
			va.AssertRegionChanged(fieldEmpty, fieldBox[0], fieldBox[1], fieldBox[2], fieldBox[3], 1,
				"the field render after typing")
			x, y, w, h := region(cb)
			cbBox = [4]int32{x, y, w, h}
			cbOff = va.Snapshot(x, y, w, h)
			cx, cy := Center(cb)
			va.Click(cx, cy) // toggle the checkbox
		},
		va.Frames(1), // let the toggle reach the frame
		func(va *VisualApp) {
			if c := va.Reg.Get("cb").(*widgets.CheckBox); !c.Checked {
				t.Errorf("pushed click did not check the checkbox")
			}
			va.AssertRegionChanged(cbOff, cbBox[0], cbBox[1], cbBox[2], cbBox[3], 1,
				"the checkbox render after toggling")
			// The final rendered state is a committed golden input.
			va.Golden("state_transitions")
		},
	})
}
