package widgets

import (
	"strings"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

// AdaptiveTable is ONE table with two presentations, chosen by the surface's size class: the real
// `DataTable` when there is room for columns, and a list of `ListItem`s on a phone.
//
// It exists because a table on a phone is not a table. Columns either shrink until every value is
// truncated or the user scrolls sideways to compare two cells that should be on one line — and the
// plan's own rule is that a `DataTable` becomes a list on a phone. The wide presentation is the
// EXISTING widget rather than a second implementation: it keeps sorting, column resizing, selection
// and its scrollbar, and this widget only decides which of the two the surface gets.
type AdaptiveTable struct {
	meui.BaseContainer
	*theme.TableTheme

	columns []DataColumn
	rows    []DataRow

	table *DataTable
	items []*ListItem

	// SelectedRow is the selected row in either presentation.
	SelectedRow int

	// OnRowSelected fires when a row is selected in either presentation, so an app has one thing to
	// wire rather than two.
	OnRowSelected func(row int)
	// OnCellActivated fires when a cell is clicked in the TABLE presentation. A phone has no cells
	// to click, so its rows report through OnRowSelected; this is the table's extra affordance.
	OnCellActivated func(row, col int)
}

// NewAdaptiveTable builds an empty table with both presentations ready.
func NewAdaptiveTable() *AdaptiveTable {
	t := &AdaptiveTable{
		BaseContainer: meui.BaseContainer{BaseWidget: *meui.NewBaseWidget()},
		TableTheme:    theme.DefaultTableTheme,
		SelectedRow:   -1,
	}
	t.table = NewDataTable()
	t.table.ShowHeader = true
	t.table.OnCellSelected = func(row, col int) {
		t.selectRow(row, true)
		if t.OnCellActivated != nil {
			t.OnCellActivated(row, col)
		}
	}
	t.AddChild(t.table)
	return t
}

// ── data ────────────────────────────────────────────────────────────

func (t *AdaptiveTable) SetColumns(cols []DataColumn) *AdaptiveTable {
	t.columns = cols
	t.table.SetColumns(cols)
	t.rebuildItems()
	t.MarkForRedraw()
	return t
}

func (t *AdaptiveTable) Columns() []DataColumn { return t.columns }

func (t *AdaptiveTable) SetRows(rows []DataRow) *AdaptiveTable {
	t.rows = rows
	t.table.SetRows(rows)
	t.rebuildItems()
	t.MarkForRedraw()
	return t
}

func (t *AdaptiveTable) Rows() []DataRow { return t.rows }

// Table exposes the wide presentation, for an app that wants its sorting or column resizing.
func (t *AdaptiveTable) Table() *DataTable { return t.table }

// Items exposes the phone presentation's rows, in order.
func (t *AdaptiveTable) Items() []*ListItem { return t.items }

// cellText is a cell's text, safely.
func cellText(row DataRow, col int) string {
	if col < 0 || col >= len(row.Cells) {
		return ""
	}
	return row.Cells[col].Text
}

// rebuildItems turns the rows into list rows: the FIRST column is the primary line, and the rest
// become one secondary line of "Label value" pairs.
//
// The first column as the primary is the rule that makes this readable: a table's first column is
// the row's identity (a name, a symbol, a date), and the rest are its attributes. An app that wants
// a different order can put the columns in that order — the widget does not guess.
func (t *AdaptiveTable) rebuildItems() {
	t.items = nil
	for i := range t.rows {
		row := t.rows[i]
		item := NewListItem()
		item.SetDivider(true)
		item.SetPrimary(cellText(row, 0))

		var parts []string
		for c := 1; c < len(t.columns); c++ {
			v := cellText(row, c)
			if v == "" {
				continue
			}
			label := ""
			if c < len(t.columns) {
				label = t.columns[c].Label
			}
			if label != "" {
				parts = append(parts, label+" "+v)
			} else {
				parts = append(parts, v)
			}
		}
		if len(parts) > 0 {
			item.SetSecondary(strings.Join(parts, " · "))
		}
		// More than two attributes is a three-line row: the height has to follow the content, or
		// the wrapped line is clipped.
		if len(parts) > 2 {
			item.SetVariant(ListItemThreeLine)
		}
		// The items are PAINTED by this widget rather than added as children (the two presentations
		// share one rect), but they still need a parent so their own coordinate walk reaches the
		// window.
		item.SetParent(t)
		rowIdx := i
		item.OnClicked = func() { t.selectRow(rowIdx, true) }
		t.items = append(t.items, item)
	}
}

// ── presentation ────────────────────────────────────────────────────

// List reports whether this surface gets the LIST presentation.
//
// The SURFACE decides, not the widget's own rect, and that is deliberate: the adaptive dialog and the
// pickers already resolve their presentation from the surface's size class, and a toolkit where two
// widgets answer "is this a phone?" differently is a toolkit an app has to learn twice. It is also
// what the plan says — a `DataTable` becomes a list on a phone — and it makes the phone layout
// testable on a desk, because a narrow desktop window is a phone by the same rule.
//
// Without a window (a unit test, or a widget not yet in a tree) it falls back to the rect it is
// handed, which is the only size available.
func (t *AdaptiveTable) List(bounds graphics.Rect) bool {
	w, h := bounds.Width, bounds.Height
	if win := windowOf(t); win != nil {
		if wf := win.FrameRect(); wf.Width > 0 && wf.Height > 0 {
			w, h = wf.Width, wf.Height
		}
	}
	return platform.FormFactorFor(w, h, 1) == platform.FormFactorPhone
}

// selectRow moves the selection and reports it, so a programmatic move behaves like a tap.
func (t *AdaptiveTable) selectRow(row int, notify bool) {
	if row < 0 || row >= len(t.rows) || row == t.SelectedRow {
		return
	}
	t.SelectedRow = row
	if t.table != nil {
		t.table.SelectedRow = row
	}
	for i, item := range t.items {
		item.SetSelected(i == row)
	}
	t.MarkForRedraw()
	if notify && t.OnRowSelected != nil {
		t.OnRowSelected(row)
	}
}

// SetSelectedRow selects a row in both presentations.
func (t *AdaptiveTable) SetSelectedRow(row int) *AdaptiveTable {
	t.selectRow(row, false)
	return t
}

func (t *AdaptiveTable) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	if t.List(graphics.NewRect(0, 0, w, h)) {
		height := float32(0)
		for _, item := range t.items {
			height += item.RowHeight()
		}
		return w, height
	}
	if t.table != nil {
		return t.table.PreferredDimensions(t.table, w, h)
	}
	return w, h
}

func (t *AdaptiveTable) ValidateLayout() {
	frame := t.FrameRect()
	if frame.Width <= 0 || frame.Height <= 0 {
		t.ClearNeedsLayout()
		return
	}
	bounds := graphics.NewRect(0, 0, frame.Width, frame.Height)
	if t.List(bounds) {
		y := float32(0)
		for _, item := range t.items {
			h := item.RowHeight()
			item.SetFrameRect(graphics.NewRect(0, y, frame.Width, h))
			y += h
		}
		if t.table != nil {
			// The wide presentation is laid out anyway, so a resize does not paint a stale one.
			t.table.SetFrameRect(bounds)
		}
	} else if t.table != nil {
		t.table.SetFrameRect(bounds)
	}
	t.ClearNeedsLayout()
}

// ── paint ───────────────────────────────────────────────────────────

func (t *AdaptiveTable) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !t.Visible() || rect.Width < 1 || rect.Height < 1 {
		return
	}
	if t.NeedsLayout() {
		t.ValidateLayout()
	}
	canvas.Save()
	canvas.Translate(rect.X, rect.Y)
	bounds := graphics.NewRect(0, 0, rect.Width, rect.Height)

	if t.List(bounds) {
		bg := graphics.NewPaint()
		bg.SetColor(theme.DefaultTheme.Background)
		canvas.DrawRect(bounds, bg)
		for _, item := range t.items {
			item.Paint(canvas, item.FrameRect())
		}
	} else if t.table != nil {
		t.table.Paint(canvas, bounds)
	}
	canvas.Restore()
}

// ListAt reports which list row holds a point, or -1. It is the phone presentation's hit test, which
// the container's own dispatch does not do for it (the items are painted, not added as children).
func (t *AdaptiveTable) ListAt(x, y float32) int {
	bounds := t.FrameRect()
	if !t.List(graphics.NewRect(0, 0, bounds.Width, bounds.Height)) {
		return -1
	}
	for i, item := range t.items {
		if item.FrameRect().Contains(x, y) {
			return i
		}
	}
	return -1
}

// PanelAt routes a press to the active presentation: the table when there is room, and the list row
// under the point on a phone.
func (t *AdaptiveTable) PanelAt(x, y float32) meui.Widget {
	bounds := t.FrameRect()
	if t.List(graphics.NewRect(0, 0, bounds.Width, bounds.Height)) {
		if i := t.ListAt(x, y); i >= 0 {
			return t.items[i]
		}
		return t
	}
	if t.table != nil {
		return t.table
	}
	return t
}
