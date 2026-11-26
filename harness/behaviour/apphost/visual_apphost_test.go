//go:build visual && !cgo

// This test lives in its own package on purpose: the behaviour suite's harness creates one
// window per test in one process, and the app-host path (meui.App + AddWindow) starts a
// SECOND loop of its own. In the same binary that combined with the harness's earlier
// windows and the hosted frame came back as the clear colour; as its own package it gets
// its own process, which is also the rule the multi-window tests follow.
// `make test-visual` runs ./behaviour/... so both halves are in the gate.

package apphost

// The app-host probe, promoted: meui.App + AddWindow is the path every real application
// uses, and it is NOT the path the rest of this suite takes (the harness builds the window
// from the shim directly). A regression here paints nothing while every other test stays
// green — which is exactly what happened to the TradeUI port.
//
// The test hosts a hand-built tree, demands three frames, checks that both halves of the
// window are still the right colour on the last frame AND the first (a smeared or misplaced
// quad still reads as "the marker is there"), and pushes a press to prove the window's own
// input handler is wired.

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/mobilegles"
)

// TestMain shuts SDL down after the run: a windowed test leaves the subsystems live, and
// the process segfaults at exit without an explicit shutdown (the same reason the
// behaviour package has one).
func TestMain(m *testing.M) {
	code := m.Run()
	mobilegles.Shutdown()
	os.Exit(code)
}

// ink formats a colour the way the failure messages need it.
func ink(c graphics.Color) string { return fmt.Sprintf("#%06X", uint32(c)&0xFFFFFF) }

// hasDisplay reports whether an X or Wayland display is present.
func hasDisplay() bool {
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

// clickEvents builds a primary-button click at window coordinates, the way the behaviour
// suite's shared input helpers do.
func clickEvents(x, y float32) []*platform.Event {
	pos := graphics.NewPoint(x, y)
	return []*platform.Event{
		{Type: platform.EventMouseDown, Pos: pos, Button: 0},
		{Type: platform.EventMouseUp, Pos: pos, Button: 0},
	}
}

func TestTheAppHostPaintsAHostedTreeAndKeepsDrawing(t *testing.T) {
	if !hasDisplay() {
		t.Skip("no display for visual testing")
	}
	const w, h = 320, 200
	red, blue := graphics.RGB(255, 0, 0), graphics.RGB(0, 0, 255)

	var (
		mu     sync.Mutex
		frames int
		pix    []uint32
		pw, ph int32
		first  []uint32
		clicks int
	)
	done := make(chan struct{})
	var runErr error
	go func() {
		runtime.LockOSThread()
		defer close(done)

		app := meui.NewApp(sdl3.NewApp(), meui.AppOptions{Title: "behaviour - app host"})
		root := meui.NewWindow()
		root.SetVisible(true)
		content := meui.NewPanel()
		content.SetVisible(true)
		root.AddChild(content)
		left := meui.NewPanel()
		left.SetVisible(true)
		left.SetBackground(red)
		left.SetFrameRect(graphics.NewRect(0, 0, w/2, h))
		content.AddChild(left)
		right := meui.NewPanel()
		right.SetVisible(true)
		right.SetBackground(blue)
		right.SetFrameRect(graphics.NewRect(w/2, 0, w-w/2, h))
		content.AddChild(right)
		root.SetOnMouseDown(func(x, y float32, button int) {
			mu.Lock()
			clicks++
			mu.Unlock()
		})

		wnd, err := app.AddWindow(root, meui.WindowOptions{
			Title: "behaviour - app host",
			Size:  graphics.NewSize(w, h),
		})
		if err != nil {
			runErr = fmt.Errorf("AddWindow: %w", err)
			return
		}
		sw, ok := wnd.(*sdl3.Window)
		if !ok {
			runErr = fmt.Errorf("AddWindow returned %T, want *sdl3.Window", wnd)
			return
		}

		// A window whose loop stopped would otherwise hang the test forever.
		watchdog := time.AfterFunc(15*time.Second, app.Quit)
		defer watchdog.Stop()

		var arm func()
		arm = func() {
			sw.CaptureNext(func(p []uint32, cw, ch int32) {
				mu.Lock()
				frames++
				pix, pw, ph = p, cw, ch
				if frames == 1 {
					first = append([]uint32(nil), p...)
				}
				n := frames
				mu.Unlock()
				if n == 1 {
					// The press goes through the same queue an OS event does; the window's
					// own handler must see it.
					for _, ev := range clickEvents(w/4, h/2) {
						sw.PushEvent(*ev)
					}
				}
				if n < 3 {
					arm()
					return
				}
				app.Quit()
			})
		}
		arm()
		runErr = app.Run()
	}()
	<-done
	if runErr != nil {
		t.Fatalf("the app host failed: %v", runErr)
	}

	mu.Lock()
	defer mu.Unlock()
	if frames < 3 {
		t.Fatalf("captured %d frames, want 3 — the hosted window stopped drawing", frames)
	}
	if pw != w || ph != h {
		t.Fatalf("captured %dx%d, want %dx%d", pw, ph, w, h)
	}
	sample := func(frame []uint32, x, y int32) graphics.Color {
		return graphics.Color(frame[(ph-1-y)*pw+x])
	}
	if got := sample(pix, pw/4, ph/2); got != red {
		t.Fatalf("left half = %s, want red (#FF0000)", ink(got))
	}
	if got := sample(pix, 3*pw/4, ph/2); got != blue {
		t.Fatalf("right half = %s, want blue (#0000FF)", ink(got))
	}
	if got := sample(first, pw/4, ph/2); got != red {
		t.Fatalf("the FIRST frame's left half = %s, want red — the first frame was not painted", ink(got))
	}
	if clicks == 0 {
		t.Fatal("a queued press never reached the hosted window's mouse-down handler")
	}
}
