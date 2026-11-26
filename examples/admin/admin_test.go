package main

// The admin console's contract: the search and role filters project the member list, the
// invite validates before it mutates, and removing a member needs a selection.

import (
	"testing"

	"github.com/sixnineone/meui/web"
)

func loadAdmin(t *testing.T) *admin {
	t.Helper()
	scene, err := web.Load("admin.qml", "windows11-dark", 1280, 800)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return wire(scene)
}

func TestSearchAndRoleFilterTheMembers(t *testing.T) {
	a := loadAdmin(t)
	if got := len(a.table.Rows); got != 9 {
		t.Fatalf("unfiltered table has %d rows, want 9", got)
	}

	a.search.SetText("mira")
	a.rebuild()
	if got := len(a.table.Rows); got != 1 {
		t.Fatalf("search 'mira' left %d rows, want 1", got)
	}
	if got := a.table.Cell(0, 0); got == nil || got.Text != "Mira Chen" {
		t.Fatalf("search kept %v, want Mira Chen", got)
	}

	a.search.SetText("")
	a.filterRole = "Viewer"
	a.rebuild()
	if got := len(a.table.Rows); got != 2 {
		t.Fatalf("Viewer has %d rows, want 2", got)
	}

	// The two filters compose: no Admin matches both.
	a.search.SetText("mira")
	a.rebuild()
	if got := len(a.table.Rows); got != 0 {
		t.Fatalf("search and role together left %d rows, want 0", got)
	}
}

func TestInviteValidatesBeforeItAdds(t *testing.T) {
	a := loadAdmin(t)
	before := len(a.all)

	a.openInvite()
	if !a.dialog.Visible() || !a.scrim.Visible() {
		t.Fatal("the invite overlay did not open")
	}

	a.invite.SetText("not-an-email")
	a.sendInvite()
	if len(a.all) != before {
		t.Fatal("an invalid address was added")
	}
	if !a.errMsg.Visible() {
		t.Fatal("the validation message is not visible")
	}

	a.invite.SetText("noor@atlas.dev")
	a.sendInvite()
	if len(a.all) != before+1 {
		t.Fatalf("the invite added %d members, want 1", len(a.all)-before)
	}
	if a.dialog.Visible() {
		t.Fatal("the dialog stayed open after sending")
	}
	if got := a.all[0]; got.email != "noor@atlas.dev" || got.status != "Invited" {
		t.Fatalf("the new member is %+v, want noor invited", got)
	}

	// A duplicate is refused with its own message.
	a.openInvite()
	a.invite.SetText("noor@atlas.dev")
	a.sendInvite()
	if len(a.all) != before+1 {
		t.Fatal("a duplicate address was added")
	}
}

func TestRemovingNeedsASelection(t *testing.T) {
	a := loadAdmin(t)
	before := len(a.all)

	a.removeSelected()
	if len(a.all) != before {
		t.Fatal("removing with no selection changed the list")
	}

	a.table.SelectedRow = 0
	a.inspect(0)
	a.removeSelected()
	if len(a.all) != before-1 {
		t.Fatalf("remove took %d members, want 1", before-len(a.all))
	}
	for _, m := range a.all {
		if m.email == "port@atlas.dev" {
			t.Fatal("the removed member is still in the model")
		}
	}
}
