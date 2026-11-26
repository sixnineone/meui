package main

// The console's contract: one model projects into the table, the table's own row fills the
// inspector, and a run moves queued → ok through the same rebuild the timer uses.

import (
	"testing"

	"github.com/sixnineone/meui/web"
	"github.com/sixnineone/meui/widgets"
)

func loadOps(t *testing.T) *console {
	t.Helper()
	scene, err := web.Load("ops.qml", "meui-light", 1280, 800)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return wire(scene, false)
}

func cellTexts(t *testing.T, c *console, col int) []string {
	t.Helper()
	out := make([]string, len(c.table.Rows))
	for i := range c.table.Rows {
		if cell := c.table.Cell(i, col); cell != nil {
			out[i] = cell.Text
		}
	}
	return out
}

func TestAFilterProjectsTheModelIntoTheTable(t *testing.T) {
	c := loadOps(t)
	if got := len(c.table.Rows); got != 16 {
		t.Fatalf("unfiltered table has %d rows, want 16", got)
	}

	c.filter = "billing"
	c.rebuild()
	if got := len(c.table.Rows); got != 4 {
		t.Fatalf("billing has %d rows, want 4", got)
	}
	for _, service := range cellTexts(t, c, 0) {
		if service != "billing" {
			t.Fatalf("a %q row survived the billing filter", service)
		}
	}

	c.filter = "staging"
	c.rebuild()
	if got := len(c.table.Rows); got != 5 {
		t.Fatalf("staging has %d rows, want 5", got)
	}
}

func TestTheInspectorReadsTheTablesOwnRow(t *testing.T) {
	c := loadOps(t)
	c.table.OnCellSelected(2, 0)
	if got, want := c.inspService.Text, "billing"; got != want {
		t.Fatalf("inspector service = %q, want %q", got, want)
	}
	if got, want := c.inspStatus.Text, "warn"; got != want {
		t.Fatalf("inspector status = %q, want %q", got, want)
	}
	if got, want := c.inspOwner.Text, "payments"; got != want {
		t.Fatalf("inspector owner = %q, want %q", got, want)
	}

	// No selection clears the pane to placeholders rather than stale values.
	c.table.OnCellSelected(-1, 0)
	if c.inspService.Text != "—" {
		t.Fatalf("the cleared inspector kept %q", c.inspService.Text)
	}
}

func TestANewRunIsQueuedThenCompletes(t *testing.T) {
	c := loadOps(t)
	before := len(c.all)

	c.newRun()
	if len(c.all) != before+1 {
		t.Fatalf("newRun added %d runs, want 1", len(c.all)-before)
	}
	if got := c.table.Cell(0, 1); got == nil || got.Text != "queued" {
		t.Fatalf("the new run's status is %v, want queued", got)
	}
	if c.inspService.Text != "api-gateway" {
		t.Fatalf("the new run was not inspected: %q", c.inspService.Text)
	}

	c.finishRun(0, "api-gateway")
	cell := c.table.Cell(0, 1)
	if cell == nil || cell.Text != "ok" {
		t.Fatalf("after completion the status is %v, want ok", cell)
	}
	if cell.Color != statusInk["ok"] {
		t.Fatalf("the completed run kept the queued ink: %q", cell.Color)
	}
	// The activity strip carries the newest line.
	if got := c.logs[len(c.logs)-1].Text; got == "" {
		t.Fatal("the completion wrote no activity line")
	}
}

var _ = widgets.NewLabel // the package's tests use the widgets types through the console
