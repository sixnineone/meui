//go:build visual

package behaviour

import (
	"testing"
	"unsafe"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/widgets"
	"github.com/sixnineone/mobilegles"
)

// injectedTexts keeps the text buffers alive until SDL consumes them
// (the SDL queue stores a pointer into this memory).
var injectedTexts [][]byte

// TestVisualSDLTextInput validates the FULL SDL text-input path: a raw
// SDL_EVENT_TEXT_INPUT event (type 0x303, text pointer at offset 24)
// pushed into the SDL queue must decode to runes and reach the focused
// TextField. This is the chain that was dead in the live app
// (SDL_StartTextInput never called + wrong event type constant).
//
// It also asserts the render: the field's region must CHANGE once the rune
// arrives, which is what proves the keystroke reached the frame and not just
// the widget's Text field.
func TestVisualSDLTextInput(t *testing.T) {
	va := NewVisualApp(t, "TestVisualSDLTextInput", visualQML, 800, 600)

	injectText := func(s string) {
		// The SDL queue holds the event (with its text pointer) until
		// the next poll — the string must outlive this function.
		injectedTexts = append(injectedTexts, append([]byte(s), 0))
		str := injectedTexts[len(injectedTexts)-1]
		buf := make([]byte, 64)
		buf[0] = byte(mobilegles.SDL3_EVENT_TEXT_INPUT & 0xFF)
		buf[1] = byte((mobilegles.SDL3_EVENT_TEXT_INPUT >> 8) & 0xFF) // little-endian
		strPtr := uintptr(unsafe.Pointer(&str[0]))
		for i := 0; i < 8; i++ {
			buf[24+i] = byte(strPtr >> (8 * i))
		}
		mobilegles.SDL3PushEvent(buf)
	}

	var field *widgets.TextField
	var fieldBox [4]int32 // x, y, w, h of the field's text area
	var before []graphics.Color

	va.Run(t, []VisualStep{
		func(va *VisualApp) {
			field = va.Reg.Get("field").(*widgets.TextField)
			x, y := Center(field)
			va.Click(x, y) // focus
		},
		func(va *VisualApp) {
			// The frame captured before this step shows the focused, still
			// empty field: that is the render to compare against.
			f := field.FrameRect()
			fieldBox = [4]int32{int32(f.X), int32(f.Y), int32(f.Width), int32(f.Height)}
			before = va.Snapshot(fieldBox[0], fieldBox[1], fieldBox[2], fieldBox[3])
			injectText("K") // SDL-level keystroke
		},
		va.Frames(2), // the queued rune dispatches next frame
		func(va *VisualApp) {
			if field.Text != "K" {
				t.Errorf("SDL text input did not reach the field: text=%q focused=%v cursor=%d", field.Text, field.Focused, field.CursorPos)
			}
			va.AssertRegionChanged(before, fieldBox[0], fieldBox[1], fieldBox[2], fieldBox[3], 1,
				"the field's render after an SDL keystroke")
		},
	})
}
