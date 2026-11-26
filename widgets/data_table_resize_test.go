package widgets

import (
	"fmt"
	"testing"

	"github.com/sixnineone/meui/graphics"
)

func dataTableWithAutoColumns(t *testing.T, n int, w float32) *DataTable {
	t.Helper()
	dt := NewDataTable()
	dt.SetFrameRect(graphics.NewRect(0, 0, w, 400))
	cols := make([]DataColumn, n)
	for i := range cols {
		// Width 0 is what the QML `columns: ["a","b",...]` path produces.
		cols[i] = DataColumn{Label: fmt.Sprintf("c%d", i), Type: ColText}
	}
	dt.SetColumns(cols)
	dt.Paint(graphics.NewCanvas(graphics.NewTestDevice(w, 400)), graphics.NewRect(0, 0, w, 400))
	return dt
}

func widthsTotal(v []float32) float32 {
	var s float32
	for _, x := range v {
		s += x
	}
	return s
}

// Until the user touches a divider, auto columns share the leftover space and
// re-fit when the table changes width.
func TestDataTableAutoColumnsFillAndRefit(t *testing.T) {
	dt := dataTableWithAutoColumns(t, 10, 1000)
	// getColWidths returns the live slice, so measure before asking again.
	wide := append([]float32(nil), dt.getColWidths(1000)...)
	if got := widthsTotal(wide); got < 999 || got > 1001 {
		t.Errorf("auto columns fill %v of a 1000px table", got)
	}
	narrow := dt.getColWidths(500)
	if narrow[0] >= wide[0] {
		t.Errorf("auto column did not re-fit: %v wide, %v narrow", wide[0], narrow[0])
	}
}

// Dragging a divider makes the widths the user's: the columns before the
// dragged one keep their width (so the divider follows the pointer exactly,
// as it does in a fixed-width table) and the change survives a repaint.
func TestDataTableColumnDragDoesNotShiftOtherColumns(t *testing.T) {
	dt := dataTableWithAutoColumns(t, 15, 1000)
	before := append([]float32(nil), dt.getColWidths(1000)...)

	const drag = 75
	edge := before[0] + before[1] + before[2]
	dt.OnMouseDown(edge, 5, 0)
	if dt.resizingCol != 2 {
		t.Fatalf("drag grabbed column %d, want 2", dt.resizingCol)
	}
	dt.OnMouseMove(edge+drag, 5)
	dt.OnMouseUp(edge+drag, 5, 0)

	after := dt.getColWidths(1000)
	if after[2] != before[2]+drag {
		t.Errorf("dragged column %.1f -> %.1f, want +%d", before[2], after[2], drag)
	}
	if after[0] != before[0] || after[1] != before[1] {
		t.Errorf("columns before the drag moved: %.1f,%.1f -> %.1f,%.1f",
			before[0], before[1], after[0], after[1])
	}
	if got := after[0] + after[1] + after[2]; got != edge+drag {
		t.Errorf("divider sits at %.1f, pointer at %.1f", got, edge+drag)
	}

	// A later layout must not undo it.
	dt.Paint(graphics.NewCanvas(graphics.NewTestDevice(1000, 400)), graphics.NewRect(0, 0, 1000, 400))
	if got := dt.getColWidths(1000)[2]; got != before[2]+drag {
		t.Errorf("repaint reset the dragged column to %.1f", got)
	}
}

// SetColumns starts a fresh layout: auto columns fill again.
func TestDataTableSetColumnsResetsUserSizing(t *testing.T) {
	dt := dataTableWithAutoColumns(t, 6, 600)
	before := append([]float32(nil), dt.getColWidths(600)...)
	edge := before[0]
	dt.OnMouseDown(edge, 5, 0)
	dt.OnMouseMove(edge+40, 5)
	dt.OnMouseUp(edge+40, 5, 0)
	if !dt.userSized {
		t.Fatal("a drag did not take over the widths")
	}
	cols := make([]DataColumn, 6)
	for i := range cols {
		cols[i] = DataColumn{Label: fmt.Sprintf("c%d", i), Type: ColText}
	}
	dt.SetColumns(cols)
	if dt.userSized {
		t.Error("SetColumns kept the user-sized flag")
	}
	if got := widthsTotal(dt.getColWidths(600)); got < 599 || got > 601 {
		t.Errorf("after SetColumns the auto columns fill %v of 600", got)
	}
}

// Explicitly sized columns keep working as before (the history tables).
func TestDataTableExplicitColumnsKeepTheirWidths(t *testing.T) {
	dt := NewDataTable()
	dt.SetFrameRect(graphics.NewRect(0, 0, 1000, 300))
	dt.SetColumns([]DataColumn{
		{Label: "a", Width: 80}, {Label: "b", Width: 120}, {Label: "c", Width: 200},
	})
	w := dt.getColWidths(1000)
	if w[0] != 80 || w[1] != 120 || w[2] != 200 {
		t.Errorf("explicit widths changed: %v", w)
	}
	// Dragging one of them adjusts exactly that column.
	dt.OnMouseDown(80, 5, 0)
	dt.OnMouseMove(80+30, 5)
	dt.OnMouseUp(80+30, 5, 0)
	w = dt.getColWidths(1000)
	if w[0] != 110 || w[1] != 120 {
		t.Errorf("after the drag: %v, want 110,120,200", w[:2])
	}
}
