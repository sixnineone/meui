package behaviour

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/widgets"
)

// tapAt sends a press and a release at a point in the window's root space.
func tapAt(h *Harness, pt graphics.Point) {
	h.Win.HandleEvent(&platform.Event{Type: platform.EventMouseDown, Pos: pt, Button: 0})
	h.Win.HandleEvent(&platform.Event{Type: platform.EventMouseUp, Pos: pt, Button: 0})
}

// TestAListRowIsTappableAwayFromItsParentsOrigin is the test that separates a LOCAL containment test
// from a parent-relative one — and the reason it is a BEHAVIOUR test rather than a widget one.
//
// The event methods are handed coordinates in the widget's OWN space (that is what ContainsPoint
// checks), while FrameRect is parent-relative. So `FrameRect().Contains(x, y)` is right for a widget
// at its parent's origin and wrong for every other one — and a widget test that places its subject
// at (0,0) cannot tell the two apart, because there they agree. Only a real tree, with the row laid
// out below something else, disagrees.
func TestAListRowIsTappableAwayFromItsParentsOrigin(t *testing.T) {
	h := Load(t, `Window { width: 400; height: 600
		VStack {
			height: 500
			spacing: 10
			margins: 12
			Label { text: "above"; height: 40 }
			ListItem { id: first; text: "First"; height: 56 }
			ListItem { id: second; text: "Second"; height: 72 }
		}
	}`)

	first, _ := h.Widget("first").(*widgets.ListItem)
	second, _ := h.Widget("second").(*widgets.ListItem)
	if first == nil || second == nil {
		t.Fatalf("the rows did not load: first=%T second=%T", h.Widget("first"), h.Widget("second"))
	}

	// The precondition: the second row really is away from the origin, so its local coordinates and
	// its parent-relative frame disagree. Without this the test could pass for the wrong reason.
	if second.FrameRect().Y == 0 {
		t.Fatalf("the second row is at y=0; this test is meant to place it below another row")
	}

	var firstClicks, secondClicks int
	first.OnClicked = func() { firstClicks++ }
	second.OnClicked = func() { secondClicks++ }

	for _, row := range []*widgets.ListItem{first, second} {
		r := row.FrameRect()
		// PointToRoot takes LOCAL coordinates and walks the ancestors, which is the same conversion
		// the window does in reverse before handing the event back.
		tapAt(h, row.PointToRoot(r.Width/2, r.Height/2))
	}

	if firstClicks != 1 {
		t.Errorf("the first row was tapped %d times, want 1", firstClicks)
	}
	if secondClicks != 1 {
		t.Errorf("the second row was tapped %d times, want 1 — a row away from the origin is not "+
			"hit by a containment test against its parent-relative frame", secondClicks)
	}
}

// TestASwipeRowAwayFromTheOriginRevealsOnItsOwnAction is the same bug in the swipe row: its action
// hit test has to be in the row's own space too, or an open row's actions are unreachable.
func TestASwipeRowAwayFromTheOriginRevealsOnItsOwnAction(t *testing.T) {
	h := Load(t, `Window { width: 400; height: 600
		VStack {
			height: 500
			spacing: 10
			margins: 12
			Label { text: "above"; height: 120 }
			SwipeActionRow {
				id: row
				width: 300
				height: 72
				ListItem { text: "Swipe me"; secondary: "to reveal" }
			}
		}
	}`)

	row, _ := h.Widget("row").(*widgets.SwipeActionRow)
	if row == nil {
		t.Fatalf("the swipe row did not load: %T", h.Widget("row"))
	}
	if row.FrameRect().Y == 0 {
		t.Fatalf("the swipe row is at y=0; this test is meant to place it below something")
	}
	row.SetActions([]widgets.SwipeAction{
		{ID: "delete", Label: "Delete", Destructive: true},
		{ID: "archive", Label: "Archive"},
	})
	fired := ""
	row.OnAction = func(id string) { fired = id }

	row.Open()
	// The action's rect is in the ROW's space, so a tap aimed at it must reach the row's own hit
	// test rather than the parent-relative one.
	r, ok := row.ActionRect(0, graphics.NewRect(0, 0, row.FrameRect().Width, row.FrameRect().Height))
	if !ok {
		t.Fatal("no rect for the first action")
	}
	tapAt(h, row.PointToRoot(r.X+r.Width/2, r.Y+r.Height/2))
	if fired != "delete" {
		t.Errorf("tapping the first action fired %q, want delete", fired)
	}
}
