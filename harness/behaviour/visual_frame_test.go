//go:build visual

package behaviour

import (
	"testing"

	"github.com/sixnineone/meui/widgets"
)

// TestVisualFrameBar renders the custom title bar to PNGs so the
// button glyphs/positions can be inspected offline.
func TestVisualFrameBar(t *testing.T) {
	va := NewVisualApp(t, "TestVisualFrameBar", visualQML, 800, 600)

	frame := widgets.NewWindowFrame()
	frame.SetTitle("MeUi Demo v3")

	va.Run(t, []VisualStep{
		func(va *VisualApp) {
			// Install the frame on the live window: bar drawn above
			// the content, offsets applied.
			va.Wnd().SetTitleBar(frame)
			va.Wnd().SetTitle("MeUi Demo v3")
		},
		func(va *VisualApp) {
			// The bar painted: its strip is not one flat colour.
			va.AssertRegionHasContent(0, 0, va.W, int32(widgets.FrameBarHeight), "the title bar")
		},
	})
}
