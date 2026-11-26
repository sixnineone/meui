package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
)

func newTestAdaptiveTable() *AdaptiveTable {
	t := NewAdaptiveTable()
	t.SetColumns([]DataColumn{
		{Label: "Symbol"},
		{Label: "Qty"},
		{Label: "Price"},
	})
	t.SetRows([]DataRow{
		{Cells: []DataCell{{Text: "AAPL"}, {Text: "10"}, {Text: "1.20"}}},
		{Cells: []DataCell{{Text: "MSFT"}, {Text: "5"}, {Text: "2.40"}}},
	})
	return t
}

// TestAdaptiveTablePicksThePresentationBySizeClass is the whole widget: a table when there is room
// for columns, a list of rows on a phone, chosen by the SURFACE rather than by the OS.
func TestAdaptiveTablePicksThePresentationBySizeClass(t *testing.T) {
	at := newTestAdaptiveTable()

	phone := graphics.NewRect(0, 0, 400, 800)
	tablet := graphics.NewRect(0, 0, 800, 900)
	desktop := graphics.NewRect(0, 0, 1100, 750)

	if !at.List(phone) {
		t.Error("a 400-wide surface did not get the list presentation")
	}
	if at.List(tablet) || at.List(desktop) {
		t.Error("a surface with room got the list presentation; it should have a real table")
	}
}

// TestAdaptiveTableListRowsCarryTheColumns is the data mapping: the FIRST column is the row's
// identity and becomes the primary line, and the rest are its attributes on the secondary.
func TestAdaptiveTableListRowsCarryTheColumns(t *testing.T) {
	at := newTestAdaptiveTable()
	items := at.Items()
	if len(items) != 2 {
		t.Fatalf("built %d list rows for 2 rows", len(items))
	}
	if got := items[0].primaryLabel.Text; got != "AAPL" {
		t.Errorf("the first row's primary line is %q, want the first column AAPL", got)
	}
	secondary := items[0].secondaryLabel.Text
	for _, want := range []string{"Qty 10", "Price 1.20"} {
		if !contains(secondary, want) {
			t.Errorf("the first row's secondary line %q does not carry %q", secondary, want)
		}
	}
	if items[1].primaryLabel.Text != "MSFT" {
		t.Errorf("the second row's primary line is %q, want MSFT", items[1].primaryLabel.Text)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle ||
		len(needle) == 0 ||
		indexOf(haystack, needle) >= 0)
}

func indexOf(h, n string) int {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return i
		}
	}
	return -1
}

// TestAdaptiveTableSelectsInBothPresentations is the contract that makes it ONE widget: an app wires
// one callback and gets the row either way.
func TestAdaptiveTableSelectsInBothPresentations(t *testing.T) {
	at := newTestAdaptiveTable()
	var got []int
	at.OnRowSelected = func(row int) { got = append(got, row) }

	// The phone presentation: tapping a list row.
	at.SetFrameRect(graphics.NewRect(0, 0, 400, 800))
	at.ValidateLayout()
	if !at.List(at.FrameRect()) {
		t.Fatal("the phone surface is not in the list presentation")
	}
	item := at.Items()[1]
	item.OnMouseDown(10, 10, 0)
	item.OnMouseUp(10, 10, 0)
	if len(got) != 1 || got[0] != 1 {
		t.Fatalf("tapping the second list row reported %v, want [1]", got)
	}
	if at.SelectedRow != 1 {
		t.Errorf("SelectedRow is %d, want 1", at.SelectedRow)
	}

	// The wide presentation: a cell selection reports the same way.
	at.SetFrameRect(graphics.NewRect(0, 0, 1100, 750))
	at.ValidateLayout()
	if at.List(at.FrameRect()) {
		t.Fatal("the wide surface is in the list presentation")
	}
	at.Table().OnCellSelected(0, 1)
	if len(got) != 2 || got[1] != 0 {
		t.Errorf("selecting a cell reported %v, want the row 0 appended", got)
	}
	if at.SelectedRow != 0 {
		t.Errorf("SelectedRow is %d, want 0", at.SelectedRow)
	}
}

// TestAdaptiveTableListRoutesThePressToTheRow keeps the phone presentation clickable: the rows are
// painted, not added as children, so the container's own hit test has to route to them.
func TestAdaptiveTableListRoutesThePressToTheRow(t *testing.T) {
	at := newTestAdaptiveTable()
	at.SetFrameRect(graphics.NewRect(0, 0, 400, 800))
	at.ValidateLayout()

	item := at.Items()[0]
	r := item.FrameRect()
	hit := at.PanelAt(r.X+r.Width/2, r.Y+r.Height/2)
	if hit != meui.Widget(item) {
		t.Errorf("the list hit test returned %T, want the row under the point", hit)
	}
	// A wide surface routes to the table instead.
	at.SetFrameRect(graphics.NewRect(0, 0, 1100, 750))
	at.ValidateLayout()
	if hit := at.PanelAt(100, 100); hit != meui.Widget(at.Table()) {
		t.Errorf("the wide hit test returned %T, want the table", hit)
	}
}

// TestAdaptiveTableIsAnInputTarget keeps it in the registry the touch audit checks.
func TestAdaptiveTableIsAnInputTarget(t *testing.T) {
	if !meui.IsInteractive(NewAdaptiveTable()) {
		t.Error("an AdaptiveTable handles input but is not in the interactive registry")
	}
}
