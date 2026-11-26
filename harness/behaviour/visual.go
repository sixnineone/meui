package behaviour

// Visual testing: runs the REAL windowed pipeline (SDL3 → shim sink →
// widget tree) and captures a screenshot after each state transition —
// hover, text edits, toggles — so renders (not just state) are
// verified. Steps run on the window's render thread: events are pushed
// into the native queue and each step chains to the next via a
// one-shot framebuffer capture.
//
// The captured frame is available to the next step through PixelAt,
// Snapshot and the Assert* helpers (pixels.go), so a test can assert on the
// render and not only on widget state.

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/qml"
)

// VisualStep performs one state transition (push events via va.Push /
// va.Click, assert state and pixels) between screenshots.
type VisualStep func(va *VisualApp)

// VisualApp hosts the real windowed pipeline for screenshot-sequence
// tests. Skips the test when no display is available, or when the binary was
// built with cgo.
type VisualApp struct {
	T   *testing.T
	Win *meui.Window
	Reg *qml.BoundWidgetRegistry
	Dir string
	// W and H are the window size the sequence runs at, from NewVisualApp.
	W, H int32

	step      int
	wnd       *sdl3.Window
	goldenDir string

	// pending is the number of extra frames a Frames(n) step still owes.
	pending int
	// fatalMsg is set by Fatalf and reported from the test goroutine.
	fatalMsg string

	// lastPix/lastW/lastH are the frame captured immediately before the
	// current step (bottom-up, 0xAARRGGBB).
	lastPix []uint32
	lastW   int32
	lastH   int32
}

// NewVisualApp parses qmlSrc and prepares a screenshot-sequence run at
// w×h (both default to 800×600 when non-positive). Call Run to start the
// windowed pipeline.
//
// Skips the test when there is no display, and when the binary was built with
// cgo: the toolkit reaches C through purego, and the cgo runtime's thread
// entry is the only source of this suite's segfaults, so it runs
// CGO_ENABLED=0 (REVIEW.md §6).
func NewVisualApp(t *testing.T, name, qmlSrc string, w, h int32) *VisualApp {
	t.Helper()
	res, err := qml.ParseWithBindings(qmlSrc)
	if err != nil {
		t.Fatalf("qml parse: %v", err)
	}
	win, ok := res.Widget.(*meui.Window)
	if !ok {
		t.Fatalf("qml root is %T", res.Widget)
	}
	va := newVisualApp(t, name, win, w, h)
	va.Reg = res.Widgets
	return va
}

// NewVisualAppTree is NewVisualApp for a tree built in Go rather than a QML scene. The
// probes that drove a real window by hand to check resize geometry and app hosting are
// tests now, and their scenes are built with the widget API (and laid out with the layout
// package), which is also what a host does.
func NewVisualAppTree(t *testing.T, name string, win *meui.Window, w, h int32) *VisualApp {
	t.Helper()
	if win == nil {
		t.Fatal("NewVisualAppTree: nil window")
	}
	if w <= 0 {
		w = 800
	}
	if h <= 0 {
		h = 600
	}
	// A QML scene arrives with the window's frame already set (web.Load does it); a tree
	// built in Go has none, and a zero-sized window lays its layout out to nothing.
	win.SetFrameRect(graphics.NewRect(0, 0, float32(w), float32(h)))
	return newVisualApp(t, name, win, w, h)
}

// newVisualApp holds what both constructors share: the skip conditions, the default size
// and the capture directory.
func newVisualApp(t *testing.T, name string, win *meui.Window, w, h int32) *VisualApp {
	t.Helper()
	if cgoEnabled {
		t.Skip("the visual suite runs CGO_ENABLED=0 — the cgo runtime is the only source of its segfaults (see REVIEW.md §6)")
	}
	if !hasDisplay() {
		t.Skip("no display for visual testing")
	}
	if w <= 0 {
		w = 800
	}
	if h <= 0 {
		h = 600
	}

	// Captures go to the PACKAGE's out/ dir, not the process's working
	// directory: a cwd-relative path wrote a second tree under
	// behaviour/behaviour/ when the test binary ran from elsewhere, and left
	// a PNG tracked in git. MEUI_BEHAVIOUR_OUT overrides it.
	_, thisFile, _, _ := runtime.Caller(0)
	pkgDir := filepath.Dir(thisFile)
	outRoot := os.Getenv("MEUI_BEHAVIOUR_OUT")
	if outRoot == "" {
		outRoot = filepath.Join(pkgDir, "out")
	}
	dir := filepath.Join(outRoot, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	return &VisualApp{
		T: t, Win: win, Dir: dir,
		W: w, H: h,
		goldenDir: filepath.Join(pkgDir, "testdata", "golden"),
	}
}

// hasDisplay reports whether an X or Wayland display is present. Without one
// the window cannot be created and Run skips.
func hasDisplay() bool {
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

// Push queues one platform event into the window's event queue.
func (va *VisualApp) Push(ev platform.Event) {
	va.wnd.PushEvent(ev)
}

func (va *VisualApp) pushAll(evs []*platform.Event) {
	for _, ev := range evs {
		va.Push(*ev)
	}
}

// Click queues a left click at window coords (x, y).
func (va *VisualApp) Click(x, y float32) { va.pushAll(clickEvents(x, y)) }

// DblClick queues a double-click at (x, y) — the second press carries
// ClickCount 2.
func (va *VisualApp) DblClick(x, y float32) { va.pushAll(dblClickEvents(x, y)) }

// Drag presses at (x1, y1), moves in steps, and releases at (x2, y2).
func (va *VisualApp) Drag(x1, y1, x2, y2 float32) { va.pushAll(dragEvents(x1, y1, x2, y2)) }

// Wheel queues a wheel event at (x, y) with deltas (dx, dy).
func (va *VisualApp) Wheel(x, y, dx, dy float32) { va.Push(*wheelEvent(x, y, dx, dy)) }

// KeyDown queues a key press (key-up has no behaviour yet).
func (va *VisualApp) KeyDown(k platform.KeyCode, mods platform.Modifiers) {
	va.Push(*keyDownEvent(k, mods))
}

// Hover moves the mouse to (x, y) — triggers hover styling.
func (va *VisualApp) Hover(x, y float32) {
	va.Push(platform.Event{Type: platform.EventMouseMove, Pos: graphics.Point{X: x, Y: y}})
}

// Type queues rune events to the focused widget.
func (va *VisualApp) Type(s string) { va.pushAll(runeEvents(s)) }

// Frames renders n frames for this step before the sequence continues. It is
// the explicit replacement for the blank `func(va *VisualApp) {}` steps that
// used to encode "wait for the SDL pump": an event injected during step N is
// observable once the pump has run, so `va.Frames(1)` after an injection is
// what the old blank step meant.
func (va *VisualApp) Frames(n int) VisualStep {
	return func(va *VisualApp) {
		if n < 1 {
			n = 1
		}
		va.pending = n
	}
}

// Fatalf ends the sequence from a step. testing.T.Fatal is illegal on the
// render goroutine — it calls runtime.Goexit, which belongs to the test
// goroutine — so a step records the failure and closes the window; Run
// reports it from the test goroutine.
func (va *VisualApp) Fatalf(format string, args ...any) {
	if va.fatalMsg == "" {
		va.fatalMsg = fmt.Sprintf(format, args...)
	}
	if va.wnd != nil {
		va.wnd.Close()
	}
}

// Run starts the windowed pipeline and executes the steps in order.
// Each step is followed by a screenshot; the next step is chained from
// the capture callback so every state transition gets its own frame.
func (va *VisualApp) Run(t *testing.T, steps []VisualStep) {
	t.Helper()
	title := "behaviour - " + t.Name()

	// The window and its GL context must be created on the thread that runs
	// the frame loop, and only inside the startup callback: an EGL/GLX
	// context never moves between threads, so a window created on the test
	// goroutine and driven from this one fails MakeCurrent on every frame
	// ("GL context bind failed") — the window stays interactive, nothing is
	// presented, no capture ever fires and Run blocks forever. renderparity
	// creates its window in this same callback for the same reason.
	//
	// The loop still runs on its own goroutine: os.Exit at the end of the
	// sequence must not come from a test goroutine (tRunner panics).
	started := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		defer close(done)
		sdl3.Start(sdl3.StartupFinishedCallback(func() {
			wnd, err := sdl3.NewWindowSized(title, va.W, va.H)
			if err != nil {
				started <- err
				return // no window: Start's loop returns immediately
			}
			va.wnd = wnd
			wnd.SetContentRect(graphics.NewRect(0, 0, float32(va.W), float32(va.H)))
			va.Win.ValidateLayout() // lay out against the REAL window size
			wnd.SetContent(va.Win.AsPanel())
			wnd.SetSink(va.Win)
			wnd.Show() // wires the draw callback — without it nothing renders
			started <- nil
			va.runStep(0, steps)
		}))
	}()

	// Skip from the test goroutine: Skipf calls runtime.Goexit, which is only
	// legal here, not in the render loop.
	if err := <-started; err != nil {
		t.Skipf("no window: %v", err)
	}
	<-done
	if va.fatalMsg != "" {
		t.Fatal(va.fatalMsg)
	}
}

func (va *VisualApp) runStep(i int, steps []VisualStep) {
	if i >= len(steps) {
		va.wnd.Close() // loop exits; Run returns; TestMain shuts SDL down
		return
	}
	va.pending = 0
	steps[i](va)
	// Capture this step's rendered state, then chain on from the capture
	// callback (render thread). A Frames(n) step holds the chain for n frames.
	var capture func(pix []uint32, w, h int32)
	capture = func(pix []uint32, w, h int32) {
		va.lastPix, va.lastW, va.lastH = pix, w, h
		va.saveFrame()
		if va.pending > 0 {
			va.pending--
			if va.pending > 0 {
				va.wnd.CaptureNext(capture)
				return
			}
		}
		va.runStep(i+1, steps)
	}
	va.wnd.CaptureNext(capture)
}

// saveFrame writes the captured frame to out/<name>/step_NN.png. The captures
// are for inspection; the assertions read lastPix directly.
func (va *VisualApp) saveFrame() {
	name := fmt.Sprintf("%s/step_%02d.png", va.Dir, va.step)
	va.step++
	if va.lastPix == nil {
		return
	}
	f, err := os.Create(name)
	if err != nil {
		return
	}
	defer f.Close()
	png.Encode(f, pixelsToImage(va.lastPix, va.lastW, va.lastH))
}

// Wnd exposes the shim window for tests that need frame/chrome wiring.
func (va *VisualApp) Wnd() *sdl3.Window { return va.wnd }
