package main

// The mail scene's contracts, headless: the list is VIRTUAL (240 messages, a dozen
// widgets), choosing a row selects it and updates the pane, and choosing a row that is
// below the fold reveals it first.

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/web"
	"github.com/sixnineone/meui/widgets"
)

func loadMail(t *testing.T) (*web.Scene, *meui.Window, *widgets.VirtualList) {
	t.Helper()
	scene, err := web.Load("mail.qml", "web-light", 1280, 900)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	wire(scene)
	win, ok := scene.Root.(*meui.Window)
	if !ok {
		t.Fatalf("root is %T, want *meui.Window", scene.Root)
	}
	vl, ok := scene.Get("messageView").(*widgets.VirtualList)
	if !ok {
		t.Fatalf("messageView is %T, want *widgets.VirtualList", scene.Get("messageView"))
	}
	return scene, win, vl
}

func click(t *testing.T, win *meui.Window, x, y float32) {
	t.Helper()
	win.HandleEvent(&platform.Event{Type: platform.EventMouseDown, Pos: graphics.NewPoint(x, y), Button: 0})
	win.HandleEvent(&platform.Event{Type: platform.EventMouseUp, Pos: graphics.NewPoint(x, y), Button: 0})
}

// rowScreenY is the window-space y of a built row's band.
func rowScreenY(vl *widgets.VirtualList, index int) float32 {
	// The list sits at y=84 in the scene, and the row's content y is inset + index*rowH.
	return 84 + vl.TopInset() + float32(index)*vl.RowHeight() - vl.ScrollY()
}

func TestTheListIsVirtual(t *testing.T) {
	_, _, vl := loadMail(t)
	if vl.RowCount() != len(inbox) {
		t.Fatalf("row count = %d, want %d", vl.RowCount(), len(inbox))
	}
	if len(inbox) < 100 {
		t.Fatalf("the demo data is only %d messages: it no longer proves virtualisation", len(inbox))
	}
	if vl.RowWidget(0) == nil {
		t.Fatal("row 0 was not built")
	}
	if vl.RowWidget(len(inbox)-1) != nil {
		t.Fatal("the last row was built before it could be seen")
	}
}

// TestChoosingAMessageSelectsItAndUpdatesThePane drives a real click through the window.
func TestChoosingAMessageSelectsItAndUpdatesThePane(t *testing.T) {
	scene, win, vl := loadMail(t)

	if p := vl.RowWidget(0).(*meui.Panel); !p.Selected() {
		t.Fatal("row 0 should start selected")
	}

	// Row 1's band starts at the inset + 110 and is 110 tall.
	click(t, win, 300, rowScreenY(vl, 1)+55)

	if row1 := vl.RowWidget(1).(*meui.Panel); !row1.Selected() {
		t.Fatal("clicking row 1 did not select it")
	}
	if row0 := vl.RowWidget(0).(*meui.Panel); row0.Selected() {
		t.Fatal("row 0 stayed selected — selection is not exclusive")
	}
	if label, ok := scene.Get("readSubject").(*widgets.Label); ok {
		if want := inbox[1].Subject; label.Text != want {
			t.Fatalf("reading pane subject = %q, want %q", label.Text, want)
		}
	} else {
		t.Fatalf("readSubject is %T, want *widgets.Label", scene.Get("readSubject"))
	}
}

// TestChoosingABelowTheFoldRowScrollsItIntoView pins the reveal: a row 200 messages down
// can only be clicked once the list has moved to it, and it is built on demand.
func TestChoosingABelowTheFoldRowScrollsItIntoView(t *testing.T) {
	_, win, vl := loadMail(t)
	const target = 200
	if got := vl.ScrollY(); got != 0 {
		t.Fatalf("the list starts at scroll %.0f, want 0", got)
	}

	vl.EnsureVisible(target)
	vl.Refresh()
	if vl.ScrollY() == 0 {
		t.Fatal("revealing row 200 did not scroll")
	}
	if vl.RowWidget(target) == nil {
		t.Fatal("the revealed row was not built")
	}
	if vl.RowWidget(0) != nil {
		t.Fatal("row 0 is still built after scrolling 200 rows away")
	}

	click(t, win, 300, rowScreenY(vl, target)+55)
	if p := vl.RowWidget(target).(*meui.Panel); !p.Selected() {
		t.Fatal("clicking the revealed row did not select it")
	}
}
