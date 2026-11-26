package widgets

import (
	"testing"
	"time"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

// TestSnackbarDismissesItself is the contract that makes it a snackbar rather than a
// panel: it goes away on its own, with nothing polling.
func TestSnackbarDismissesItself(t *testing.T) {
	s := NewSnackbar()
	s.Duration = 20 * time.Millisecond
	s.SetFrameRect(graphics.NewRect(0, 0, 320, s.BarHeight()))
	if s.Showing() {
		t.Fatal("a new snackbar is already showing")
	}

	s.Show("saved")
	if !s.Showing() {
		t.Fatal("Show did not put a message up")
	}

	// The timer is a platform task; drain it rather than sleeping.
	deadline := time.Now().Add(2 * time.Second)
	for s.Showing() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
		platform.DrainTasks()
	}
	if s.Showing() {
		t.Error("the message never dismissed itself")
	}
}

// TestASecondMessageIsNotHiddenByTheFirstsTimer is the bug the sequence number exists
// for: showing a new message must not be cancelled by the OLD one's pending dismissal.
func TestASecondMessageIsNotHiddenByTheFirstsTimer(t *testing.T) {
	s := NewSnackbar()
	s.Duration = 30 * time.Millisecond
	s.SetFrameRect(graphics.NewRect(0, 0, 320, s.BarHeight()))

	s.Show("first")
	time.Sleep(10 * time.Millisecond)
	s.Show("second") // supersedes, with its own full duration

	// Wait past the FIRST message's deadline but not the second's.
	deadline := time.Now().Add(15 * time.Millisecond)
	for time.Now().Before(deadline) {
		platform.DrainTasks()
		time.Sleep(2 * time.Millisecond)
	}
	if !s.Showing() {
		t.Error("the first message's timer dismissed the second message")
	}
	if s.Text() != "second" {
		t.Errorf("text is %q, want second", s.Text())
	}
}

// TestSnackbarActionRunsThenDismisses pins the order: the callback runs AFTER the
// dismissal so it is free to show another message.
func TestSnackbarActionRunsThenDismisses(t *testing.T) {
	s := NewSnackbar()
	s.Duration = time.Hour // keep the timer out of the way
	s.SetFrameRect(graphics.NewRect(0, 0, 320, s.BarHeight()))

	ran := false
	stillShowingDuringCallback := true
	s.ShowAction("deleted", "Undo", func() {
		ran = true
		stillShowingDuringCallback = s.Showing()
	})

	r, ok := s.ActionRect()
	if !ok {
		t.Fatal("an action was set but ActionRect reports none")
	}
	cx, cy := r.X+r.Width/2, r.Y+r.Height/2
	s.OnMouseDown(cx, cy, 0)
	s.OnMouseUp(cx, cy, 0)

	if !ran {
		t.Fatal("the action did not run")
	}
	if stillShowingDuringCallback {
		t.Error("the message was still up while the action ran, so the action cannot replace it")
	}
	if s.Showing() {
		t.Error("the message stayed up after its action ran")
	}
}

// TestSnackbarActionReachesTheTouchFloor: the action is the whole reason the message is
// up, so it must be tappable.
func TestSnackbarActionReachesTheTouchFloor(t *testing.T) {
	saved := theme.DefaultTheme
	t.Cleanup(func() { theme.SetDefaultTheme(saved) })
	th := *saved
	theme.SetDefaultTheme(&th)
	theme.DefaultTheme.Metrics.MinTouchTarget = theme.TouchTargetAndroid

	s := NewSnackbar()
	s.SetFrameRect(graphics.NewRect(0, 0, 320, s.BarHeight()))
	s.ShowAction("deleted", "Undo", nil)
	r, ok := s.ActionRect()
	if !ok {
		t.Fatal("no action rect")
	}
	if r.Width < theme.TouchTargetAndroid {
		t.Errorf("the action is %.0f wide, under the %v floor", r.Width, theme.TouchTargetAndroid)
	}
}

// TestSnackbarPressElsewhereDismisses covers the gesture a user tries first.
func TestSnackbarPressElsewhereDismisses(t *testing.T) {
	s := NewSnackbar()
	s.Duration = time.Hour
	s.SetFrameRect(graphics.NewRect(0, 0, 320, s.BarHeight()))
	s.Show("saved")
	s.OnMouseDown(20, s.BarHeight()/2, 0)
	if s.Showing() {
		t.Error("a press on the message did not dismiss it")
	}
}

// TestASnackbarWithNoMessageIsNotVisible is the phantom hit target: a Snackbar lives in the
// shell's FLOW, so one that reports itself visible while hidden reserves layout space AND takes
// every press in its band. That is what happened on the device — a drag over the Widgets page's
// lower band landed on the hidden Snackbar instead of the widget under the finger, so the
// ScrollPane never scrolled and the page looked un-scrollable.
func TestASnackbarWithNoMessageIsNotVisible(t *testing.T) {
	s := NewSnackbar()
	if s.Visible() {
		t.Error("a snackbar with no message reports itself VISIBLE, so it takes presses in its band")
	}
	s.Show("saved")
	if !s.Visible() {
		t.Error("a snackbar with a message reports itself hidden")
	}
	s.Dismiss()
	if s.Visible() {
		t.Error("a dismissed snackbar is still visible")
	}
}

var _ meui.EventListener = (*Snackbar)(nil)
