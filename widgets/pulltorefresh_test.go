package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
)

// scrollStub stands in for the scrollable content: it can report a scroll offset and it records the
// events the wrapper forwards, which is how the tests tell "the wrapper took the gesture" from "the
// content did".
type scrollStub struct {
	meui.BaseWidget
	offsetY  float32
	downs    int
	moves    int
	ups      int
	lastDown graphics.Point
}

func (s *scrollStub) ScrollOffsets() (float32, float32) { return 0, s.offsetY }

func (s *scrollStub) OnMouseDown(x, y float32, button int) bool {
	s.downs++
	s.lastDown = graphics.NewPoint(x, y)
	return true
}

func (s *scrollStub) OnMouseMove(x, y float32) { s.moves++ }
func (s *scrollStub) OnMouseUp(x, y float32, button int) bool {
	s.ups++
	return true
}

func newTestPull() (*PullToRefresh, *scrollStub) {
	content := &scrollStub{BaseWidget: *meui.NewBaseWidget()}
	p := NewPullToRefresh()
	p.SetContent(content)
	p.SetFrameRect(graphics.NewRect(0, 0, 360, 600))
	p.ValidateLayout()
	return p, content
}

// TestPullToRefreshPullsOnlyFromTheTop is the whole point of the wrapper: a downward drag with the
// content already at its top is a PULL, and the same drag with the content scrolled is a SCROLL.
func TestPullToRefreshPullsOnlyFromTheTop(t *testing.T) {
	p, content := newTestPull()

	// At the top: a downward drag pulls, and the content is NOT given the moves.
	p.OnMouseDown(180, 100, 0)
	p.OnMouseMove(180, 200)
	if p.Pull() <= 0 {
		t.Fatalf("a downward drag at the top did not pull (pull=%.0f)", p.Pull())
	}
	if content.moves != 0 {
		t.Errorf("the content was given %d moves during a pull", content.moves)
	}
	p.OnMouseUp(180, 200, 0)

	// Scrolled: the same drag belongs to the content.
	content.offsetY = 120
	content.downs, content.moves = 0, 0
	p.Done()
	p.OnMouseDown(180, 100, 0)
	p.OnMouseMove(180, 200)
	if p.Pull() != 0 {
		t.Errorf("a drag with the content scrolled pulled %.0f, want 0", p.Pull())
	}
	if content.downs == 0 || content.moves == 0 {
		t.Errorf("the content was not given the gesture (downs=%d moves=%d)", content.downs, content.moves)
	}
	p.OnMouseUp(180, 200, 0)
}

// TestPullToRefreshAnUpwardDragAlwaysScrolls keeps the other direction honest: an upward drag is
// never a pull, at the top or not.
func TestPullToRefreshAnUpwardDragAlwaysScrolls(t *testing.T) {
	p, content := newTestPull()
	p.OnMouseDown(180, 300, 0)
	p.OnMouseMove(180, 200)
	if p.Pull() != 0 {
		t.Errorf("an upward drag pulled %.0f, want 0", p.Pull())
	}
	if content.moves == 0 {
		t.Error("an upward drag was not forwarded to the content")
	}
	p.OnMouseUp(180, 200, 0)
}

// TestPullToRefreshFiresOncePastTheThreshold is the contract with the app: the gesture reports, and
// the app says when it is finished.
func TestPullToRefreshFiresOncePastTheThreshold(t *testing.T) {
	p, _ := newTestPull()
	fired := 0
	p.OnRefresh = func() { fired++ }

	// A short pull springs back and does nothing.
	p.OnMouseDown(180, 100, 0)
	p.OnMouseMove(180, 100+PullThreshold) // half speed, so this is half a threshold
	p.OnMouseUp(180, 100+PullThreshold, 0)
	if fired != 0 {
		t.Errorf("a short pull fired %d times", fired)
	}
	if p.Pull() != 0 {
		t.Errorf("a short pull left the content at %.0f, want 0", p.Pull())
	}

	// A pull past the threshold fires, and the indicator stays up until Done.
	p.OnMouseDown(180, 100, 0)
	p.OnMouseMove(180, 100+PullThreshold*3)
	p.OnMouseUp(180, 100+PullThreshold*3, 0)
	if fired != 1 {
		t.Fatalf("a full pull fired %d times, want 1", fired)
	}
	if !p.Refreshing() {
		t.Error("the wrapper is not refreshing after a full pull")
	}
	if p.Pull() < p.pullThreshold() {
		t.Errorf("the content settled at %.0f during a refresh, want it held at the threshold", p.Pull())
	}
	p.Done()
	if p.Refreshing() || p.Pull() != 0 {
		t.Errorf("Done left refreshing=%v pull=%.0f", p.Refreshing(), p.Pull())
	}
}

// TestPullToRefreshForwardsATap is the container's price: a tap that was never a gesture has to
// reach the content, or everything inside the wrapper becomes unclickable.
func TestPullToRefreshForwardsATap(t *testing.T) {
	p, content := newTestPull()
	p.OnMouseDown(180, 300, 0)
	p.OnMouseUp(180, 300, 0)
	if content.downs != 1 || content.ups != 1 {
		t.Errorf("a tap reached the content as downs=%d ups=%d, want 1 and 1", content.downs, content.ups)
	}
	if p.Pull() != 0 {
		t.Errorf("a tap pulled %.0f", p.Pull())
	}
}

// TestPullToRefreshIsAnInputTarget keeps it in the registry the touch audit checks.
func TestPullToRefreshIsAnInputTarget(t *testing.T) {
	if !meui.IsInteractive(NewPullToRefresh()) {
		t.Error("a PullToRefresh handles input but is not in the interactive registry")
	}
}

var _ meui.EventListener = (*PullToRefresh)(nil)
var _ meui.HoverTarget = (*PullToRefresh)(nil)
