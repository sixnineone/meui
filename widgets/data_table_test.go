package widgets

import (
	"testing"
)

func TestDataTableNew(t *testing.T) {
	dt := NewDataTable()
	if dt == nil {
		t.Fatal("NewDataTable() returned nil")
	}
	if dt.SelectedRow != -1 {
		t.Errorf("Expected SelectedRow -1, got %d", dt.SelectedRow)
	}
	if dt.HoveredRow != -1 {
		t.Errorf("Expected HoveredRow -1, got %d", dt.HoveredRow)
	}
	if dt.VScroll == nil {
		t.Error("VScroll should not be nil")
	}
}

func TestDataTableSetColumns(t *testing.T) {
	dt := NewDataTable()
	cols := []DataColumn{
		{Label: "Name", Type: ColText, Width: 150},
		{Label: "Value", Type: ColNumber},
		{Label: "Change", Type: ColPercent, Width: 100},
	}
	dt.SetColumns(cols)
	if len(dt.Columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(dt.Columns))
	}
	if dt.Columns[0].Label != "Name" {
		t.Errorf("Expected label 'Name', got '%s'", dt.Columns[0].Label)
	}
	if dt.Columns[1].Type != ColNumber {
		t.Errorf("Expected ColNumber type, got %d", dt.Columns[1].Type)
	}
}

func TestDataTableAddRow(t *testing.T) {
	dt := NewDataTable()
	dt.SetColumns([]DataColumn{
		{Label: "A"},
		{Label: "B"},
	})
	dt.AddRow([]DataCell{
		{Text: "hello"},
		{Text: "world"},
	})
	if len(dt.Rows) != 1 {
		t.Errorf("Expected 1 row, got %d", len(dt.Rows))
	}
	if len(dt.Rows[0].Cells) != 2 {
		t.Errorf("Expected 2 cells in row 0, got %d", len(dt.Rows[0].Cells))
	}
	if dt.Rows[0].Cells[0].Text != "hello" {
		t.Errorf("Expected 'hello', got '%s'", dt.Rows[0].Cells[0].Text)
	}
}

func TestDataTableUpdateCell(t *testing.T) {
	dt := NewDataTable()
	dt.SetColumns([]DataColumn{{Label: "A"}, {Label: "B"}})
	dt.AddRow([]DataCell{{Text: "old"}, {Text: "value"}})

	dt.UpdateCell(0, 0, "new")
	if dt.Rows[0].Cells[0].Text != "new" {
		t.Errorf("Expected 'new', got '%s'", dt.Rows[0].Cells[0].Text)
	}
}

func TestDataTableUpdateCells(t *testing.T) {
	dt := NewDataTable()
	dt.SetColumns([]DataColumn{{Label: "A"}, {Label: "B"}, {Label: "C"}})
	dt.AddRow([]DataCell{{Text: "a"}, {Text: "b"}, {Text: "c"}})
	dt.AddRow([]DataCell{{Text: "d"}, {Text: "e"}, {Text: "f"}})

	dt.UpdateCells([]CellUpdate{
		{Row: 0, Col: 0, Text: "x"},
		{Row: 1, Col: 2, Text: "z"},
	})
	if dt.Rows[0].Cells[0].Text != "x" {
		t.Errorf("Expected 'x', got '%s'", dt.Rows[0].Cells[0].Text)
	}
	if dt.Rows[1].Cells[2].Text != "z" {
		t.Errorf("Expected 'z', got '%s'", dt.Rows[1].Cells[2].Text)
	}
}

func TestDataTableUpdateCellNoop(t *testing.T) {
	dt := NewDataTable()
	dt.SetColumns([]DataColumn{{Label: "A"}})
	dt.AddRow([]DataCell{{Text: "same"}})

	dt.UpdateCell(0, 0, "same")
	if dt.Rows[0].Cells[0].Text != "same" {
		t.Errorf("Expected 'same', got '%s'", dt.Rows[0].Cells[0].Text)
	}
}

func TestDataTableClear(t *testing.T) {
	dt := NewDataTable()
	dt.SetColumns([]DataColumn{{Label: "A"}})
	dt.AddRow([]DataCell{{Text: "x"}})
	dt.AddRow([]DataCell{{Text: "y"}})
	dt.SelectedRow = 1

	dt.Clear()
	if len(dt.Rows) != 0 {
		t.Errorf("Expected 0 rows, got %d", len(dt.Rows))
	}
	if dt.SelectedRow != -1 {
		t.Errorf("Expected SelectedRow -1 after clear, got %d", dt.SelectedRow)
	}
}

func TestDataTableRemoveRow(t *testing.T) {
	dt := NewDataTable()
	dt.SetColumns([]DataColumn{{Label: "A"}})
	dt.AddRow([]DataCell{{Text: "a"}})
	dt.AddRow([]DataCell{{Text: "b"}})
	dt.AddRow([]DataCell{{Text: "c"}})
	dt.SelectedRow = 0

	dt.RemoveRow(1)
	if len(dt.Rows) != 2 {
		t.Errorf("Expected 2 rows after remove, got %d", len(dt.Rows))
	}
	if dt.Rows[1].Cells[0].Text != "c" {
		t.Errorf("Expected 'c' at row 1, got '%s'", dt.Rows[1].Cells[0].Text)
	}
}

func TestDataTableCell(t *testing.T) {
	dt := NewDataTable()
	dt.SetColumns([]DataColumn{{Label: "A"}, {Label: "B"}})
	dt.AddRow([]DataCell{{Text: "x"}, {Text: "y"}})

	cell := dt.Cell(0, 1)
	if cell == nil {
		t.Fatal("Cell(0,1) returned nil")
	}
	if cell.Text != "y" {
		t.Errorf("Expected 'y', got '%s'", cell.Text)
	}

	if dt.Cell(5, 0) != nil {
		t.Error("Cell(5,0) should return nil for out-of-bounds row")
	}
	if dt.Cell(0, 5) != nil {
		t.Error("Cell(0,5) should return nil for out-of-bounds col")
	}
}

func TestDataTableSetRows(t *testing.T) {
	dt := NewDataTable()
	dt.SetColumns([]DataColumn{{Label: "A"}, {Label: "B"}})
	rows := []DataRow{
		{Cells: []DataCell{{Text: "1"}, {Text: "2"}}},
		{Cells: []DataCell{{Text: "3"}, {Text: "4"}}},
	}
	dt.SetRows(rows)
	if len(dt.Rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(dt.Rows))
	}
	if dt.Rows[1].Cells[0].Text != "3" {
		t.Errorf("Expected '3', got '%s'", dt.Rows[1].Cells[0].Text)
	}
}

func TestDataTableColumnAtX(t *testing.T) {
	dt := NewDataTable()
	dt.SetColumns([]DataColumn{
		{Label: "A", Width: 100},
		{Label: "B", Width: 100},
		{Label: "C", Width: 100},
	})

	colWidths := []float32{100, 100, 100}
	tests := []struct {
		x    float32
		want int
	}{
		{10, 0},
		{90, 0},
		{110, 1},
		{220, 2},
		{350, -1},
	}
	for _, tt := range tests {
		got := dt.columnAtX(tt.x, colWidths)
		if got != tt.want {
			t.Errorf("columnAtX(%.0f) = %d, want %d", tt.x, got, tt.want)
		}
	}
}

func TestDataTableColumnAtXWithRowNumbers(t *testing.T) {
	dt := NewDataTable()
	dt.ShowRowNumbers = true
	dt.SetColumns([]DataColumn{
		{Label: "A", Width: 100},
		{Label: "B", Width: 100},
	})
	dt.RowNumberWidth = 40

	colWidths := []float32{100, 100}
	tests := []struct {
		x    float32
		want int
	}{
		{10, -1},  // row number column
		{30, -1},  // row number column
		{50, 0},   // column A
		{130, 0},  // column A
		{150, 1},  // column B
		{200, 1},  // column B
		{241, -1}, // beyond columns
	}
	for _, tt := range tests {
		got := dt.columnAtX(tt.x, colWidths)
		if got != tt.want {
			t.Errorf("columnAtX(%.0f) = %d, want %d (ShowRowNumbers=true)", tt.x, got, tt.want)
		}
	}
}
