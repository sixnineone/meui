//go:build visual

package behaviour

import (
	"os"
	"testing"
	"time"

	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/widgets"
)

// TestVisualMultiWindow opens a second window and pumps the composite
// loop. The second window is created in the startup callback (before
// the frame loop) to isolate second-window swapping from mid-frame
// creation. A hang here reproduces the multi-window deadlock.
func TestVisualMultiWindow(t *testing.T) {
	// A second SDL window in one test process is a known hazard — it
	// segfaults on a cgo build and blocks in NewWindowNamed cgo-free
	// (REVIEW.md §6). Gate it the way TestVisualMenuBarDemoMenu is
	// gated, so the default visual gate runs one window per process:
	//   go test ./behaviour/ -run TestVisualMultiWindow
	if os.Getenv("BEHAVIOUR_VISUAL_MULTIWINDOW") == "" {
		t.Skip("set BEHAVIOUR_VISUAL_MULTIWINDOW=1 to run (one visual window per process)")
	}
	va := NewVisualApp(t, "TestVisualMultiWindow", visualQML, 500, 400)

	var second *sdl3.Window

	va.Run(t, []VisualStep{
		func(va *VisualApp) {
			// Startup callback: runs before the frame loop starts.
			wnd2, err := sdl3.NewWindowNamed("Second Window", 400, 300)
			if err != nil {
				t.Errorf("second window: %v", err)
				return
			}
			second = wnd2
			const barH = float32(widgets.FrameBarHeight)
			wnd2.SetContentRect(graphics.NewRect(0, 0, 400, 300-barH))
			frame2 := widgets.NewWindowFrame()
			frame2.SetTitle("Second")
			frame2.OnClose(func() { wnd2.Close() })
			frame2.OnMinimize(func() { wnd2.Minimize() })
			frame2.OnMaximize(func() { wnd2.Maximize() })
			wnd2.SetTitleBar(frame2)
			wnd2.Show()
		},
		va.Frames(2), // both windows paint
		func(va *VisualApp) {
			if second == nil {
				va.Fatalf("second window was not created")
			}
			time.Sleep(500 * time.Millisecond) // keep the loop running
		},
	})
}
