package meui

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

type fakeClickable struct {
	BaseWidget
}

func TestTouchHitFrameDesktopUnchanged(t *testing.T) {
	SetTouchProfile(false)
	defer SetTouchProfile(false)
	w := &fakeClickable{BaseWidget: *NewBaseWidget()}
	w.SetOnMouseDown(func(x, y float32, button int) {})
	in := graphics.NewRect(10, 10, 20, 20)
	out := TouchHitFrame(w, in)
	if out != in {
		t.Fatalf("desktop profile must not expand: got %v", out)
	}
}

func TestTouchHitFrameExpandsInteractive(t *testing.T) {
	SetTouchProfile(true)
	defer SetTouchProfile(false)
	w := &fakeClickable{BaseWidget: *NewBaseWidget()}
	w.SetOnMouseDown(func(x, y float32, button int) {})
	out := TouchHitFrame(w, graphics.NewRect(10, 10, 20, 20))
	if out.Width != MinTouchTarget() || out.Height != MinTouchTarget() {
		t.Fatalf("expected %v square, got %v", MinTouchTarget(), out)
	}
	if cx := out.X + out.Width/2; cx != 20 {
		t.Fatalf("expansion not centered: center x = %v", cx)
	}
}

func TestTouchHitFrameSkipsNonInteractive(t *testing.T) {
	SetTouchProfile(true)
	defer SetTouchProfile(false)
	w := NewBaseWidget() // no OnMouseDown — labels/panels stay exact
	in := graphics.NewRect(10, 10, 20, 20)
	if out := TouchHitFrame(w, in); out != in {
		t.Fatalf("non-interactive widget must not expand: got %v", out)
	}
}

func TestTouchHitFrameDoesNotShrink(t *testing.T) {
	SetTouchProfile(true)
	defer SetTouchProfile(false)
	w := &fakeClickable{BaseWidget: *NewBaseWidget()}
	w.SetOnMouseDown(func(x, y float32, button int) {})
	in := graphics.NewRect(0, 0, 200, 60)
	if out := TouchHitFrame(w, in); out != in {
		t.Fatalf("large widget must keep its frame: got %v", out)
	}
}

// TestExpansionSplitsTheGapBetweenCloseTargets is Android's caveat, asserted: two
// targets too close to both reach the floor must split the space between them rather
// than one swallowing the other. Without the clamp the hit test walks children in
// reverse and the last one wins the whole overlap, so the other becomes unreachable
// near its own edge — which reads as a stray mis-tap rather than a sizing bug.
func TestExpansionSplitsTheGapBetweenCloseTargets(t *testing.T) {
	savedTheme, savedProfile := theme.DefaultTheme, TouchProfile()
	t.Cleanup(func() {
		theme.SetDefaultTheme(savedTheme)
		SetTouchProfile(savedProfile)
	})
	// A copy, not LoadPreset: see the note in navigationbar_test.go.
	th := *savedTheme
	theme.SetDefaultTheme(&th)
	// After, not before: withStyleDefaults replaces Metrics wholesale when
	// ControlHeight is 0, discarding anything set on the copy.
	theme.DefaultTheme.Metrics.MinTouchTarget = theme.TouchTargetAndroid
	SetTouchProfile(true)
	floor := MinTouchTarget()

	// Two 20px targets 10px apart: neither can have a 48px target without overlap.
	newTarget := func() *fakeClickable {
		w := &fakeClickable{BaseWidget: *NewBaseWidget()}
		w.SetOnMouseDown(func(x, y float32, button int) {})
		return w
	}
	a, b := newTarget(), newTarget()
	fa := graphics.NewRect(0, 0, 20, 20)
	fb := graphics.NewRect(30, 0, 20, 20)
	a.SetFrameRect(fa)
	b.SetFrameRect(fb)

	got := TouchHitFrameAmong(a, fa, []Widget{a, b})
	if got.X+got.Width > fb.X {
		t.Errorf("a's target runs to %.0f, past b at %.0f — the expansion overlapped a sibling",
			got.X+got.Width, fb.X)
	}
	if got.Width <= fa.Width {
		t.Errorf("a's target did not expand at all: %.0f", got.Width)
	}
	// It should still take more than half the gap, since it has the floor to reach.
	if got.X+got.Width < fa.X+fa.Width+(fb.X-(fa.X+fa.Width))/2 {
		t.Errorf("a's target stopped short of the gap midpoint: ends at %.0f", got.X+got.Width)
	}

	// A sibling on another row must NOT clamp anything.
	c := newTarget()
	fc := graphics.NewRect(30, 100, 20, 20)
	c.SetFrameRect(fc)
	free := TouchHitFrameAmong(a, fa, []Widget{a, c})
	if free.Width < floor {
		t.Errorf("a sibling on a different row clamped the expansion to %.0f, want the full %v",
			free.Width, floor)
	}
}

// The floor is a property of the WINDOW a widget belongs to: a small window switching it on must not
// change how a desktop window's controls are hit-tested. That is what the process-wide flag did, and
// it is how a 560px-tall window made a spin box in another window step DOWN when its UP arrow was
// clicked.
func TestTheTouchFloorFollowsTheWindow(t *testing.T) {
	saved := TouchProfile()
	t.Cleanup(func() { SetTouchProfile(saved) })
	SetTouchProfile(false) // the default, for a window that was never told

	on := NewWindow()
	off := NewWindow()
	unset := NewWindow()
	on.SetTouchProfile(true)
	off.SetTouchProfile(false)

	a := &fakeClickable{BaseWidget: *NewBaseWidget()}
	a.SetOnMouseDown(func(x, y float32, button int) {})
	b := &fakeClickable{BaseWidget: *NewBaseWidget()}
	b.SetOnMouseDown(func(x, y float32, button int) {})
	c := &fakeClickable{BaseWidget: *NewBaseWidget()}
	c.SetOnMouseDown(func(x, y float32, button int) {})
	on.AddChild(a)
	off.AddChild(b)
	unset.AddChild(c)

	if !touchFloorFor(a) {
		t.Error("a widget in the window whose floor is ON does not resolve it")
	}
	if touchFloorFor(b) {
		t.Error("a widget in the window whose floor is OFF resolved it — the setting leaked across windows")
	}
	if touchFloorFor(c) {
		t.Error("a window with no setting did not fall back to the process default")
	}
	SetTouchProfile(true)
	if !touchFloorFor(c) {
		t.Error("the process default no longer reaches a window that has no setting")
	}

	// And a manual setting survives a resize while a derived one follows it.
	on.SetTouchProfile(true)
	on.HandleEvent(&platform.Event{Type: platform.EventResize, W: 360, H: 640})
	if !on.TouchProfile() {
		t.Error("a manually configured window lost its floor to a resize")
	}
	derived := NewWindow()
	derived.setDerivedTouchProfile(false)
	derived.HandleEvent(&platform.Event{Type: platform.EventResize, W: 360, H: 640})
	if !derived.TouchProfile() {
		t.Error("a derived floor did not re-derive on a phone-sized resize")
	}
	derived.HandleEvent(&platform.Event{Type: platform.EventResize, W: 1400, H: 900})
	if derived.TouchProfile() {
		t.Error("a derived floor did not switch off on a desktop-sized resize")
	}
}
