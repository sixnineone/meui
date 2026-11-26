//go:build visual

package behaviour

import (
	"math"
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/widgets"
	"github.com/sixnineone/mobilegles"
)

const wheelQML = `
Window {
    id: win
    width: 500
    height: 400
    VStack {
        margins: 0
        ListView {
            id: list
            width: 300
            height: 300
            model: [
                "item 1", "item 2", "item 3", "item 4", "item 5",
                "item 6", "item 7", "item 8", "item 9", "item 10",
                "item 11", "item 12", "item 13", "item 14", "item 15",
                "item 16", "item 17", "item 18", "item 19", "item 20"
            ]
        }
    }
}
`

// TestVisualWheelScrolls validates the FULL SDL wheel path: a raw
// SDL_MouseWheelEvent (type 0x403) with the cursor position in
// mouse_x/mouse_y must decode, dispatch through the hit-test at that
// position, and scroll the widget under the cursor. This is the layer
// headless tests cannot see — the wheel position bug lived here.
//
// It asserts the render too: the list's pixels must change when it scrolls and
// return to the original frame when it scrolls back, which is what proves the
// scroll reached the framebuffer and not only ScrollOffset.
func TestVisualWheelScrolls(t *testing.T) {
	va := NewVisualApp(t, "TestVisualWheelScrolls", wheelQML, 500, 400)

	injectWheel := func(dy float32, mx, my float32) {
		buf := make([]byte, 64)
		buf[0] = byte(mobilegles.SDL3_EVENT_MOUSE_WHEEL & 0xFF)
		buf[1] = byte((mobilegles.SDL3_EVENT_MOUSE_WHEEL >> 8) & 0xFF)
		putF32(buf[24:], 0)  // x: no horizontal scroll
		putF32(buf[28:], dy) // y: positive = away from user (up)
		putF32(buf[36:], mx) // mouse_x
		putF32(buf[40:], my) // mouse_y
		mobilegles.SDL3PushEvent(buf)
	}

	var list *widgets.ListView
	var box [4]int32
	var before []graphics.Color

	// NOTE: an event injected during step N is consumed by the pump during
	// step N+1's frame, so its effect is visible in the frame that step N+2
	// reads. va.Frames(1) is the explicit form of the blank "pump frame" step
	// this used to be.
	va.Run(t, []VisualStep{
		func(va *VisualApp) {
			list = va.Reg.Get("list").(*widgets.ListView)
			if list == nil {
				va.Fatalf("list not found")
			}
			if list.ScrollOffset != 0 {
				va.Fatalf("precondition: offset %v, want 0", list.ScrollOffset)
			}
			// Wheel down (dy<0 = toward user) at a point inside the list.
			injectWheel(-3, 150, 150)
		},
		func(va *VisualApp) {
			// The frame captured before this step is the unscrolled list:
			// keep it as the pixel reference. This step's own frame pumps.
			f := list.FrameRect()
			box = [4]int32{int32(f.X), int32(f.Y), int32(f.Width), int32(f.Height)}
			before = va.Snapshot(box[0], box[1], box[2], box[3])
		},
		func(va *VisualApp) {
			if list.ScrollOffset != 60 {
				t.Errorf("wheel down: ScrollOffset %v, want 60 (3 notches x 20)", list.ScrollOffset)
			}
			if list.VScroll.Value != list.ScrollOffset {
				t.Errorf("bar value %v != offset %v", list.VScroll.Value, list.ScrollOffset)
			}
			va.AssertRegionChanged(before, box[0], box[1], box[2], box[3], 1,
				"the list render after a wheel-down")
			// Wheel back up past the top: clamps at 0.
			injectWheel(10, 150, 150)
		},
		va.Frames(1), // pump frame
		func(va *VisualApp) {
			if list.ScrollOffset != 0 {
				t.Errorf("wheel up: ScrollOffset %v, want clamp at 0", list.ScrollOffset)
			}
			va.AssertRegionEqual(before, box[0], box[1], box[2], box[3], 4,
				"the list render after scrolling back to the top")
		},
	})
}

func putF32(b []byte, v float32) {
	bits := math.Float32bits(v)
	for i := 0; i < 4; i++ {
		b[i] = byte(bits >> (8 * i))
	}
}
