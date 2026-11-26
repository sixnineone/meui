package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
)

// fakeTarget records what the chrome asks the window to do.
type fakeTarget struct {
	title                        string
	minimized, maximized, closed int
}

func (f *fakeTarget) Title() string   { return f.title }
func (f *fakeTarget) Close()          { f.closed++ }
func (f *fakeTarget) Minimize()       { f.minimized++ }
func (f *fakeTarget) ToggleMaximize() { f.maximized++ }

// layOutBar paints the bar once so its button rects exist, which is what a real
// click would find.
func layOutBar(t *testing.T, frame *WindowFrame, width float32) {
	t.Helper()
	frame.PaintBar(graphics.NewCanvas(graphics.NewTestDevice(width, FrameBarHeight)), width)
}

func TestInstallChromeWiresEveryControl(t *testing.T) {
	target := &fakeTarget{title: "Test"}
	frame := InstallChrome(meui.NewWindow(), target, target.Title())
	if frame.BarHeight() != FrameBarHeight {
		t.Fatalf("bar height %v, want %v", frame.BarHeight(), FrameBarHeight)
	}
	layOutBar(t, frame, 1280)

	// A double-click on the bar toggles maximize. The frame refuses to do this
	// when no OnMaximize handler is registered, which is exactly the bug this
	// test exists for: an app that wired only OnClose got a title bar whose
	// maximize button and double-click were both silently inert.
	if !frame.BarDoubleClick(640, FrameBarHeight/2) {
		t.Fatal("a double-click on the bar was not consumed — no maximize handler")
	}
	if target.maximized != 1 {
		t.Fatalf("double-click toggled maximize %d times, want 1", target.maximized)
	}
	// A rapid repeat is deduped on purpose: a triple-click must not toggle the
	// window twice.
	frame.BarDoubleClick(640, FrameBarHeight/2)
	if target.maximized != 1 {
		t.Fatalf("a rapid repeat was not deduped (%d toggles)", target.maximized)
	}

	// The three buttons are reachable by clicking the right end of the bar, and
	// each drives its own operation.
	seen := map[string]bool{}
	for x := float32(1280 - 220); x < 1280; x += 2 {
		before := [3]int{target.minimized, target.maximized, target.closed}
		if frame.BarMouseDown(x, FrameBarHeight/2, 0) {
			frame.BarMouseUp(x, FrameBarHeight/2, 0)
		}
		switch {
		case target.minimized > before[0]:
			seen["minimize"] = true
		case target.maximized > before[1]:
			seen["maximize"] = true
		case target.closed > before[2]:
			seen["close"] = true
		}
	}
	for _, want := range []string{"minimize", "maximize", "close"} {
		if !seen[want] {
			t.Fatalf("the %s button is not reachable from the bar (%v)", want, seen)
		}
	}
}

func TestAFrameWithNoMaximizeHandlerIsInert(t *testing.T) {
	// The frame's own guard, pinned because it is what made the studio's title
	// bar inert: BarDoubleClick returns false unless the app registered
	// OnMaximize. InstallChrome registers all four, which is why apps should
	// not hand-roll this.
	frame := NewWindowFrame()
	frame.SetOwnerRoot(meui.NewWindow())
	layOutBar(t, frame, 800)
	if frame.BarDoubleClick(400, FrameBarHeight/2) {
		t.Fatal("a frame with no maximize handler consumed the double-click")
	}
}
