package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

func newTestSwipeRow() *SwipeActionRow {
	s := NewSwipeActionRow()
	row := NewListItem().SetPrimary("Message from Ada")
	s.SetContent(row)
	s.SetActions([]SwipeAction{
		{ID: "delete", Label: "Delete", Destructive: true},
		{ID: "archive", Label: "Archive"},
	})
	s.SetFrameRect(graphics.NewRect(0, 0, 360, 72))
	// Lay it out: the content's frame is what its own hit test uses, and in an app the layout pass
	// sets it. A test that skips this has a content row at the BaseWidget default 50x20.
	s.ValidateLayout()
	return s
}

// TestSwipeActionsAreRightAlignedAndReachable pins the geometry: the actions fill the strip the
// content uncovers, the first one is outermost, and every action is a target.
func TestSwipeActionsAreRightAlignedAndReachable(t *testing.T) {
	saved := theme.DefaultTheme
	t.Cleanup(func() { theme.SetDefaultTheme(saved) })
	th := *saved
	theme.SetDefaultTheme(&th)
	theme.DefaultTheme.Metrics.MinTouchTarget = theme.TouchTargetAndroid

	s := newTestSwipeRow()
	bounds := s.FrameRect()
	first, ok := s.ActionRect(0, bounds)
	if !ok {
		t.Fatal("no rect for the first action")
	}
	second, _ := s.ActionRect(1, bounds)
	if first.X+first.Width != bounds.Width {
		t.Errorf("the first action ends at %.0f, want the row's right edge %.0f", first.X+first.Width, bounds.Width)
	}
	if second.X+second.Width != first.X {
		t.Errorf("the second action ends at %.0f, want it butted against the first at %.0f",
			second.X+second.Width, first.X)
	}
	if s.ActionsWidth() != first.Width+second.Width {
		t.Errorf("ActionsWidth is %.0f, want the two actions' %.0f", s.ActionsWidth(), first.Width+second.Width)
	}
	for i := range s.actions {
		r, _ := s.ActionRect(i, bounds)
		if r.Width < theme.TouchTargetAndroid {
			t.Errorf("action %d is %.0f wide, under the %v floor", i, r.Width, theme.TouchTargetAndroid)
		}
	}
}

// TestSwipeRevealsAndSnaps is the gesture: a swipe past the slop slides the content, and the release
// either keeps it open or closes it depending on how far it went.
func TestSwipeRevealsAndSnaps(t *testing.T) {
	s := newTestSwipeRow()

	// A small drag is not a swipe at all — it must not nudge the row.
	if !s.OnMouseDown(300, 30, 0) {
		t.Fatal("the row did not take the press")
	}
	s.OnMouseMove(297, 30)
	s.OnMouseUp(297, 30, 0)
	if s.IsOpen() {
		t.Errorf("a 3px drag opened the row (offset %.0f)", s.Offset())
	}

	// A swipe most of the way but short of the full-swipe line: it stays open.
	half := -s.ActionsWidth() * 0.75
	s.OnMouseDown(300, 30, 0)
	s.OnMouseMove(300+half, 30)
	if !s.IsOpen() {
		t.Fatal("a swipe did not reveal the actions")
	}
	s.OnMouseUp(300+half, 30, 0)
	if !s.IsOpen() {
		t.Error("a swipe past half the strip closed again on release")
	}

	// A short swipe: it snaps closed.
	s.Close()
	s.OnMouseDown(300, 30, 0)
	s.OnMouseMove(300-s.ActionsWidth()*0.25, 30)
	s.OnMouseUp(300-s.ActionsWidth()*0.25, 30, 0)
	if s.IsOpen() {
		t.Errorf("a quarter-strip swipe stayed open (offset %.0f)", s.Offset())
	}
}

// TestSwipeAllTheWayRunsTheFirstAction is the platform's destructive shortcut: swiping the row most
// of the way across performs the first action on release, without aiming at a button.
func TestSwipeAllTheWayRunsTheFirstAction(t *testing.T) {
	s := newTestSwipeRow()
	var fired string
	s.OnAction = func(id string) { fired = id }

	s.OnMouseDown(340, 30, 0)
	s.OnMouseMove(340-s.FrameRect().Width*0.7, 30)
	s.OnMouseUp(340-s.FrameRect().Width*0.7, 30, 0)
	if fired != "delete" {
		t.Errorf("a full swipe fired %q, want delete (the first action)", fired)
	}
	if s.IsOpen() {
		t.Error("the row stayed open after a full swipe")
	}
}

// TestSwipeActionTapFiresItAndCloses covers the other path: an open row's action is a button.
func TestSwipeActionTapFiresItAndCloses(t *testing.T) {
	s := newTestSwipeRow()
	var fired string
	s.OnAction = func(id string) { fired = id }
	s.Open()

	r, _ := s.ActionRect(1, s.FrameRect())
	cx, cy := r.X+r.Width/2, r.Y+r.Height/2
	if !s.OnMouseDown(cx, cy, 0) {
		t.Fatal("an open action did not take the press")
	}
	s.OnMouseUp(cx, cy, 0)
	if fired != "archive" {
		t.Errorf("tapping the second action fired %q, want archive", fired)
	}
	if s.IsOpen() {
		t.Error("the row stayed open after its action ran")
	}
}

// TestSwipeRowForwardsATapToItsContent is what makes the container safe to intercept the press: a tap
// that never became a swipe has to reach the row underneath, or every row becomes unclickable.
func TestSwipeRowForwardsATapToItsContent(t *testing.T) {
	s := newTestSwipeRow()
	row, ok := s.Content().(*ListItem)
	if !ok {
		t.Fatalf("the content is %T", s.Content())
	}
	clicked := 0
	row.OnClicked = func() { clicked++ }

	s.OnMouseDown(100, 30, 0)
	s.OnMouseUp(100, 30, 0)
	if clicked != 1 {
		t.Errorf("a tap fired the content's click %d times, want 1", clicked)
	}

	// And a swipe must NOT fire it: the gesture belonged to the row.
	clicked = 0
	s.OnMouseDown(300, 30, 0)
	s.OnMouseMove(200, 30)
	s.OnMouseUp(200, 30, 0)
	if clicked != 0 {
		t.Errorf("a swipe also fired the content's click %d times", clicked)
	}
}

// TestSwipeActionUnderTheContentIsNotATarget keeps the row's own body from firing the action hidden
// behind it: a closed row's action is covered, and a tap there belongs to the content.
func TestSwipeActionUnderTheContentIsNotATarget(t *testing.T) {
	s := newTestSwipeRow()
	// Closed: the whole strip is covered.
	r, _ := s.ActionRect(0, s.FrameRect())
	if got := s.ActionAt(r.X+r.Width/2, r.Y+r.Height/2); got != -1 {
		t.Errorf("a covered action reports index %d, want -1", got)
	}
	// Open: it is a target.
	s.Open()
	if got := s.ActionAt(r.X+r.Width/2, r.Y+r.Height/2); got != 0 {
		t.Errorf("an open action reports index %d, want 0", got)
	}
}

// TestSwipeActionRowIsAnInputTarget keeps it in the registry the touch audit checks.
func TestSwipeActionRowIsAnInputTarget(t *testing.T) {
	if !meui.IsInteractive(NewSwipeActionRow()) {
		t.Error("a SwipeActionRow handles input but is not in the interactive registry")
	}
}

var _ meui.EventListener = (*SwipeActionRow)(nil)
var _ meui.HoverTarget = (*SwipeActionRow)(nil)
