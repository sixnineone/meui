package widgets

// data_table_reorder_test.go — a column the operator moves stays where it was put.
//
// Reordering is a VIEW over the caller's data, not a rewrite of it. The rows are
// supplied again on every refresh in the order the caller declared its columns,
// so the heading order has to survive SetRows, and every cell API (Cell,
// UpdateCell, the sort callback) has to keep addressing the LOGICAL column — the
// one the caller declared — or a refresh undoes the arrangement and a live cell
// update lands under the wrong heading.
//
// It used to permute the row cells as well as the headings. The next SetRows then
// put the data back in declared order under moved headings, which is what made the
// arrangement impossible to keep, and a reorder silently moved every later
// UpdateCell to the wrong column.

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
)

// reordered builds a three-column table, drags column 0 to the end, and returns it
// with a fresh row supplied in DECLARED (logical) order, the way a refresh does.
func reordered(t *testing.T) *DataTable {
	t.Helper()
	dt := NewDataTable()
	dt.SetColumns([]DataColumn{
		{Label: "Alpha", Width: 100},
		{Label: "Beta", Width: 100},
		{Label: "Gamma", Width: 100},
	})
	dt.getColWidths(300) // the drag reads the widths the renderer computed
	dt.SetRows([]DataRow{{Cells: []DataCell{{Text: "a1"}, {Text: "b1"}, {Text: "g1"}}}})

	// Drag the first heading past the last one.
	dt.dragCol, dt.dropCol = 0, 3
	dt.applyColumnReorder()

	if got := []string{dt.Columns[0].Label, dt.Columns[1].Label, dt.Columns[2].Label}; got[0] != "Beta" || got[1] != "Gamma" || got[2] != "Alpha" {
		t.Fatalf("headings = %v, want [Beta Gamma Alpha] — the drag did not take effect", got)
	}
	return dt
}

func TestAReorderedColumnKeepsItsPlaceAcrossARefresh(t *testing.T) {
	dt := reordered(t)

	// A refresh: the caller re-supplies rows in the order it declares its columns.
	dt.SetRows([]DataRow{{Cells: []DataCell{{Text: "a2"}, {Text: "b2"}, {Text: "g2"}}}})

	for i, want := range []string{"Beta", "Gamma", "Alpha"} {
		if dt.Columns[i].Label != want {
			t.Fatalf("after refresh the headings are %q at %d, want %q — the arrangement was lost",
				dt.Columns[i].Label, i, want)
		}
	}
	// And each heading still shows its OWN data: the cell under "Alpha" is the
	// logical column 0's cell, whatever position the heading now occupies.
	want := []string{"b2", "g2", "a2"}
	for i := range want {
		li := dt.columnLogical(i)
		if got := dt.Rows[0].Cells[li].Text; got != want[i] {
			t.Errorf("under %q the table shows %q, want %q", dt.Columns[i].Label, got, want[i])
		}
	}
}

func TestCellUpdatesAddressTheDeclaredColumnAfterAReorder(t *testing.T) {
	dt := reordered(t)

	// The caller updates "Gamma", which is logical column 2 and now drawn last.
	dt.UpdateCell(0, 2, "g2")

	if got := dt.Cell(0, 2).Text; got != "g2" {
		t.Fatalf("Cell(0,2) = %q, want the update to have landed on the declared column", got)
	}
	// The other columns must be untouched: a reorder that permutes the cells makes
	// this write land somewhere else entirely.
	if got := dt.Cell(0, 0).Text; got != "a1" {
		t.Errorf("Cell(0,0) = %q, want a1 — an update to column 2 changed column 0", got)
	}
	if got := dt.Cell(0, 1).Text; got != "b1" {
		t.Errorf("Cell(0,1) = %q, want b1 — an update to column 2 changed column 1", got)
	}
	// And the displayed cell under "Gamma" is the one that changed.
	if li := dt.columnLogical(1); dt.Rows[0].Cells[li].Text != "g2" {
		t.Errorf("the heading %q does not show the updated cell", dt.Columns[1].Label)
	}
}

func TestASortReportsTheDeclaredColumn(t *testing.T) {
	dt := reordered(t)

	var reported int = -99
	dt.OnSortChanged = func(col int, asc bool) { reported = col }

	// The operator clicks the third heading from the left, which is "Alpha" after
	// the drag — declared column 0.
	dt.toggleSort(2)

	if reported != 0 {
		t.Fatalf("OnSortChanged reported column %d, want 0 — a caller sorts its own data, which is "+
			"laid out in declared order, so a display position sorts the wrong field", reported)
	}
}

// A sort asks the table whether a column is numeric, and the answer must come from
// the column the caller declared — not from whatever heading now sits at that
// position.
func TestAColumnsTypeFollowsTheDeclaredColumnAfterAReorder(t *testing.T) {
	dt := NewDataTable()
	dt.SetColumns([]DataColumn{
		{Label: "Number", Type: ColNumber, Width: 100},
		{Label: "Text", Type: ColText, Width: 100},
	})
	dt.getColWidths(200)

	dt.dragCol, dt.dropCol = 0, 2 // move "Number" past "Text"
	dt.applyColumnReorder()
	if dt.Columns[1].Label != "Number" {
		t.Fatalf("headings = [%q %q], the reorder did not take effect", dt.Columns[0].Label, dt.Columns[1].Label)
	}

	if got, want := dt.ColumnLess(0, "9", "10", true), CellLess("9", "10", ColNumber, true); got != want {
		t.Errorf("ColumnLess(0) = %v, want %v — declared column 0 is numeric wherever it is drawn", got, want)
	}
	if got, want := dt.ColumnLess(1, "9", "10", true), CellLess("9", "10", ColText, true); got != want {
		t.Errorf("ColumnLess(1) = %v, want %v — declared column 1 is text", got, want)
	}
}

// A caller that projects rows from its own store may share the cell slice, so by
// the time an update reaches the table the text can already be there. The CACHED
// render is still the old one, and keeping it is a table frozen at its first
// paint — which is what the Symbols tab did with prices.
func TestATextUpdateDropsTheCachedRenderEvenWhenTheTextAlreadyMatches(t *testing.T) {
	dt := NewDataTable()
	dt.SetColumns([]DataColumn{{Label: "Symbol", Width: 80}, {Label: "Last", Width: 80}})
	dt.SetRows([]DataRow{{Cells: []DataCell{{Text: "EURUSD"}, {Text: "1.1000"}}}})

	// Stand in for the renderer having built this cell's text.
	cell := dt.Cell(0, 1)
	cell.cellText = &graphics.Text{}

	// The caller wrote its store, which is this cell, and then told the table.
	dt.UpdateCell(0, 1, "1.1000")

	if cell.cellText != nil {
		t.Fatal("the cached render survived an update whose text already matched — the screen would " +
			"keep showing the value from the first paint")
	}
}
