package widgets

import (
	"fmt"
	"os"
	"strings"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

const mouseButtonLeft int = 0
const mouseButtonRight int = 1
const colEdgeSnap float32 = 8

type DataColType int

const (
	ColText DataColType = iota
	ColNumber
	ColPercent
)

type DataColumn struct {
	Label string
	Type  DataColType
	Width float32
}

type DataCell struct {
	Text    string
	Color   string
	BgColor string
	Widget  meui.Widget // interactive widget embedded in this cell

	cellText *graphics.Text
	cellVer  uint64
}

type DataRow struct {
	Cells []DataCell
}

type CellUpdate struct {
	Row, Col int
	Text     string
	Color    string
	BgColor  string
}

// DataTable is a scrollable table with sortable, resizable and REORDERABLE columns.
//
// Reordering is a VIEW, and the split matters to every caller:
//
//   - Columns is in DISPLAY order — the order the operator dragged the headings
//     into. Widths and the drop indicator follow it.
//   - DataRow.Cells is in LOGICAL order — the order the caller declared its columns
//     in, and the order it re-supplies on every refresh, because that is the only
//     order it knows.
//   - Every cell API and every callback that carries a column index
//     (Cell, UpdateCell, UpdateCells, OnCellClicked, OnCellSelected,
//     OnCellRender, OnSortChanged, SortCol, ColumnLess) uses the LOGICAL index.
//
// colOrder maps one to the other. Keeping the rows logical is what lets an
// arrangement survive SetRows; permuting them instead made the next refresh put the
// data back in declared order under moved headings.
//
// A cell's text is CACHED as a rendered graphics.Text and rebuilt when the theme
// version moves or the cache is dropped. UpdateCell/UpdateCells drop it even when
// the text already matches, because a caller that projects rows from its own store
// may share the cell slice and write the cell before telling the table. A
// projection should still give each visible row its OWN cells: sharing them means
// the table is never told a value changed, and it is enough for the slice to be
// appended once for that to happen.
type DataTable struct {
	meui.BaseWidget
	*theme.TableTheme

	Columns         []DataColumn
	Rows            []DataRow
	ScrollOffset    float32
	VScroll         *ScrollBar
	host            *ScrollbarHost
	RowHeight       float32
	HeaderHeight    float32
	AlternatingRows bool
	ShowHeader      bool
	ShowGrid        bool
	ShowRowNumbers  bool
	RowNumberStart  int
	RowNumberWidth  float32

	SelectedRow int
	HoveredRow  int

	OnCellSelected      func(row, col int)
	OnCellClicked       func(row, col int, cell *DataCell)
	OnCellDoubleClicked func(row, col int, cell *DataCell)
	OnCellRightClicked  func(row, col int, x, y float32)
	OnHeaderClick       func(col int)
	OnCellRender        func(c *graphics.Canvas, rect graphics.Rect, row, col int, cell *DataCell)
	OnSortChanged       func(col int, asc bool)
	OnColumnsReordered  func()
	OnDelete            func(row int, mod platform.Modifiers) // Delete key on a selected row
	OnDuplicate         func(row int)                         // Shift+Enter on a selected row

	colWidths []float32
	// colOrder maps a DISPLAY column position to its LOGICAL column — the index
	// its cells have in DataRow.Cells. Nil means the two are identical.
	//
	// Reordering is a VIEW. The row cells stay in the order the caller declared
	// them, because that is the order every other cell API uses (Cell, UpdateCell,
	// UpdateCells, the sort callbacks) and the order a caller re-supplies on every
	// refresh. Permuting the cells too — which is what this used to do — meant the
	// very next SetRows put them back in declared order under moved headings, so
	// the arrangement was lost on every refresh and any cell update after a
	// reorder landed in the wrong column.
	colOrder []int
	// userSized is set when the user drags a column divider. From then on the
	// columns keep their widths instead of the auto columns re-sharing the
	// leftover on every layout: without it a drag made the *other* columns
	// shrink, so the divider lagged the pointer — the opposite of what the
	// fixed-width tables do.
	userSized bool
	textVer   uint64

	// Sort
	sortCol int
	sortAsc bool

	// Resize
	resizingCol      int
	resizeStartX     float32
	resizeStartWidth float32

	// Reorder
	dragCol    int
	dragStartX float32
	dragging   bool
	dropCol    int
	dragImage  *graphics.Image
	dragImageW float32
	dragImageH float32

	// Cell widget tracking
	activeCellWidget meui.Widget
	activeCellRow    int
	activeCellCol    int
}

func NewDataTable() *DataTable {
	dt := &DataTable{
		BaseWidget:      *meui.NewBaseWidget(),
		TableTheme:      theme.DefaultTableTheme,
		RowHeight:       22,
		HeaderHeight:    26,
		ShowHeader:      true,
		ShowGrid:        true,
		AlternatingRows: true,
		SelectedRow:     -1,
		HoveredRow:      -1,
		RowNumberStart:  1,
		RowNumberWidth:  40,
		sortCol:         -1,
		resizingCol:     -1,
		dragCol:         -1,
		dropCol:         -1,
	}
	dt.host = NewScrollbarHost()
	dt.host.ShowH = false
	dt.VScroll = dt.host.VScroll
	dt.VScroll.SetParent(dt)
	dt.host.SetOnChange(func() {
		_, offY := dt.host.Offsets()
		dt.ScrollOffset = offY
		dt.MarkForRedraw()
	})
	dt.SetTabIndex(0)
	return dt
}

func (dt *DataTable) headerHeight() float32 {
	if dt.ShowHeader {
		return dt.HeaderHeight
	}
	return 0
}

func (dt *DataTable) resolveFont() graphics.Font {
	font := dt.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	return font
}

func (dt *DataTable) SetColumns(cols []DataColumn) {
	dt.Columns = cols
	dt.colWidths = nil
	dt.userSized = false
	// A new column set is a new logical basis: whatever arrangement the previous
	// set had does not describe these columns.
	dt.colOrder = nil
	dt.MarkForRedraw()
}

// columnLogical maps a display position to the column index a cell has in
// DataRow.Cells (and in every callback that reports a column).
func (dt *DataTable) columnLogical(display int) int {
	if display < 0 || dt.colOrder == nil || display >= len(dt.colOrder) {
		return display
	}
	return dt.colOrder[display]
}

// columnDisplay maps a logical column back to its display position, or -1.
func (dt *DataTable) columnDisplay(logical int) int {
	if dt.colOrder == nil {
		return logical
	}
	for d, l := range dt.colOrder {
		if l == logical {
			return d
		}
	}
	return -1
}

func (dt *DataTable) SetRows(rows []DataRow) {
	dt.Rows = rows
	dt.SelectedRow = -1
	dt.HoveredRow = -1
	dt.ScrollOffset = 0
	dt.MarkForRedraw()
}

func (dt *DataTable) AddRow(cells []DataCell) {
	dt.Rows = append(dt.Rows, DataRow{Cells: cells})
	dt.MarkForRedraw()
}

func (dt *DataTable) Cell(row, col int) *DataCell {
	if row < 0 || row >= len(dt.Rows) || col < 0 || col >= len(dt.Columns) {
		return nil
	}
	if col >= len(dt.Rows[row].Cells) {
		return nil
	}
	return &dt.Rows[row].Cells[col]
}

func (dt *DataTable) UpdateCell(row, col int, text string) {
	if row < 0 || row >= len(dt.Rows) {
		return
	}
	if col < 0 || col >= len(dt.Columns) {
		return
	}
	if col >= len(dt.Rows[row].Cells) {
		return
	}
	cell := &dt.Rows[row].Cells[col]
	if cell.Text == text {
		// The text can already match while this table's CACHED render is stale: a
		// caller that projects rows from its own store may share the cell slice, so
		// writing the store has already changed this cell behind our back. Dropping
		// the cache costs one text build; keeping it is a table frozen at the values
		// of its first paint.
		if cell.cellText == nil {
			return
		}
		cell.cellText = nil
		dt.MarkForRedraw()
		return
	}
	cell.Text = text
	cell.cellText = nil
	dt.MarkForRedraw()
}

func (dt *DataTable) UpdateCells(updates []CellUpdate) {
	if len(updates) == 0 {
		return
	}
	for _, u := range updates {
		if u.Row < 0 || u.Row >= len(dt.Rows) {
			continue
		}
		if u.Col < 0 || u.Col >= len(dt.Columns) {
			continue
		}
		if u.Col >= len(dt.Rows[u.Row].Cells) {
			continue
		}
		cell := &dt.Rows[u.Row].Cells[u.Col]
		if u.Text != "" {
			// Same reason as UpdateCell: equal text does not prove the cached render
			// is current when the caller shares the cell sink.
			if cell.Text != u.Text {
				cell.Text = u.Text
			}
			cell.cellText = nil
		}
		if u.Color != "" && cell.Color != u.Color {
			cell.Color = u.Color
			cell.cellText = nil
		}
		if u.BgColor != "" {
			cell.BgColor = u.BgColor
		}
	}
	dt.MarkForRedraw()
}

func (dt *DataTable) Clear() {
	dt.Rows = nil
	dt.SelectedRow = -1
	dt.HoveredRow = -1
	dt.ScrollOffset = 0
	dt.MarkForRedraw()
}

func (dt *DataTable) RemoveRow(idx int) {
	if idx < 0 || idx >= len(dt.Rows) {
		return
	}
	dt.Rows = append(dt.Rows[:idx], dt.Rows[idx+1:]...)
	if dt.SelectedRow == idx {
		dt.SelectedRow = -1
	} else if dt.SelectedRow > idx {
		dt.SelectedRow--
	}
	dt.MarkForRedraw()
}

func (dt *DataTable) getColWidths(totalW float32) []float32 {
	n := len(dt.Columns)
	if n == 0 {
		return nil
	}
	firstInit := false
	if dt.colWidths == nil || len(dt.colWidths) != n {
		dt.colWidths = make([]float32, n)
		firstInit = true
		dt.userSized = false
	}
	if dt.userSized {
		// The user has positioned the dividers: every column keeps its width
		// (the table may then be narrower or wider than the frame, exactly as
		// with explicitly sized columns) and nothing is re-distributed.
		for i := range dt.colWidths {
			if dt.colWidths[i] <= 0 {
				if dt.Columns[i].Width > 0 {
					dt.colWidths[i] = dt.Columns[i].Width
				} else {
					dt.colWidths[i] = 60
				}
			}
		}
		return dt.colWidths
	}

	contentW := totalW
	if dt.ShowRowNumbers {
		contentW -= dt.RowNumberWidth
	}

	var fixedTotal float32
	autoCount := 0
	for i := range dt.Columns {
		if dt.Columns[i].Width > 0 {
			if firstInit {
				dt.colWidths[i] = dt.Columns[i].Width
			}
			fixedTotal += dt.colWidths[i]
		} else {
			autoCount++
			dt.colWidths[i] = 0
		}
	}

	remaining := contentW - fixedTotal
	if autoCount > 0 && remaining > 0 {
		autoW := remaining / float32(autoCount)
		for i := range dt.colWidths {
			if dt.Columns[i].Width == 0 {
				dt.colWidths[i] = autoW
			}
		}
	}
	for i := range dt.colWidths {
		if dt.colWidths[i] <= 0 {
			dt.colWidths[i] = 60
		}
	}

	return dt.colWidths
}

func (dt *DataTable) visibleRowRange(r graphics.Rect) (start, end int) {
	headerH := dt.headerHeight()
	topY := r.Y + headerH - dt.ScrollOffset
	totalRows := len(dt.Rows)
	if totalRows == 0 {
		return 0, 0
	}
	firstRow := int((r.Y - topY) / dt.RowHeight)
	if firstRow < 0 {
		firstRow = 0
	}
	lastRow := int((r.Y + r.Height - topY + dt.RowHeight - 1) / dt.RowHeight)
	if lastRow > totalRows {
		lastRow = totalRows
	}
	return firstRow, lastRow
}

func (dt *DataTable) columnAtX(x float32, colWidths []float32) int {
	cx := float32(0)
	if dt.ShowRowNumbers {
		cx += dt.RowNumberWidth
	}
	for i, w := range colWidths {
		if x >= cx && x < cx+w {
			return i
		}
		cx += w
	}
	return -1
}

func (dt *DataTable) buildCellText(cell *DataCell, font graphics.Font) *graphics.Text {
	if cell.Text == "" {
		return nil
	}
	ink := dt.TextInk
	if cell.Color != "" {
		ink = parseHexColor(cell.Color)
	}
	return graphics.NewText(cell.Text, &graphics.TextDecoration{
		Font:            font,
		OnBackgroundInk: ink,
	})
}

func (dt *DataTable) cellTextX(colIdx int, colX, colW, textW float32) float32 {
	if colIdx < 0 || colIdx >= len(dt.Columns) {
		return colX + 6
	}
	switch dt.Columns[colIdx].Type {
	case ColNumber, ColPercent:
		return colX + colW - textW - 16
	default:
		return colX + 6
	}
}

func (dt *DataTable) columnEdgeAtX(x float32, colWidths []float32) (col int, isEdge bool) {
	cx := float32(0)
	if dt.ShowRowNumbers {
		cx += dt.RowNumberWidth
	}
	for i, w := range colWidths {
		if (x >= cx && x <= cx+colEdgeSnap) || (x >= cx+w-colEdgeSnap && x <= cx+w) {
			return i, true
		}
		cx += w
	}
	return -1, false
}

// ── Sort ──

func (dt *DataTable) SortCol() int  { return dt.sortCol }
func (dt *DataTable) SortAsc() bool { return dt.sortAsc }

// toggleSort takes the DISPLAY position the operator clicked and records the
// LOGICAL column behind it: the callbacks that act on a sort sort their own data,
// which is laid out in declared order.
func (dt *DataTable) toggleSort(display int) {
	col := dt.columnLogical(display)
	if col == dt.sortCol {
		if dt.sortAsc {
			dt.sortAsc = false
		} else {
			dt.sortCol = -1
			dt.sortAsc = true
		}
	} else {
		dt.sortCol = col
		dt.sortAsc = true
	}
	if dt.OnSortChanged != nil {
		dt.OnSortChanged(dt.sortCol, dt.sortAsc)
	}
}

// ── Column ghost image for drag ──

func (dt *DataTable) renderColumnDragImage() {
	if dt.dragCol < 0 || dt.dragCol >= len(dt.Columns) {
		return
	}
	font := dt.resolveFont()
	colWidths := dt.getColWidths(dt.Width())
	if dt.dragCol >= len(colWidths) {
		return
	}
	cw := colWidths[dt.dragCol]
	ch := dt.HeaderHeight
	w := int(cw) + 4
	h := int(ch) + 4
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	img, err := graphics.NewImageFromDrawing(offscreenDevice(dt), w, h, 72, func(c *graphics.Canvas) {
		clear := graphics.NewPaint()
		clear.SetColor(graphics.Color(0x00000000))
		c.DrawRect(graphics.NewRect(0, 0, float32(w), float32(h)), clear)
		bg := graphics.NewPaint()
		bg.SetColor(dt.TableTheme.HeaderBgInk)
		c.DrawRect(graphics.NewRect(2, 2, cw, ch), bg)
		bd := graphics.NewPaint()
		bd.SetStyle(paintstyle.Stroke)
		bd.SetStrokeWidth(1)
		bd.SetColor(dt.TableTheme.EdgeInk)
		c.DrawRect(graphics.NewRect(2, 2, cw, ch), bd)
		tp := graphics.NewPaint()
		tp.SetColor(dt.TableTheme.TextInk)
		textW := font.SimpleWidth(dt.Columns[dt.dragCol].Label)
		textX := 2 + (cw-textW)/2
		textY := 2 + (ch-font.LineHeight())/2 + font.Baseline()
		ht := graphics.NewText(dt.Columns[dt.dragCol].Label, &graphics.TextDecoration{Font: font, OnBackgroundInk: tp.Color()})
		ht.Draw(c, textX, textY)
	})
	if err != nil {
		dt.dragImage = nil
		return
	}
	dt.dragImage = img
	dt.dragImageW = float32(w)
	dt.dragImageH = float32(h)
}

func (dt *DataTable) applyColumnReorder() {
	from := dt.dragCol
	to := dt.dropCol
	// to may be len(Columns): that is the gap AFTER the last heading, which is how
	// a column is moved to the end. Without it the last position was unreachable —
	// a drag past the final heading reported no column at all.
	if from < 0 || to < 0 || from == to || from >= len(dt.Columns) || to > len(dt.Columns) {
		return
	}

	insertAt := to
	if from < to {
		insertAt = to - 1
	}
	if insertAt >= len(dt.Columns) {
		insertAt = len(dt.Columns) - 1
	}
	if insertAt < 0 {
		insertAt = 0
	}

	if dt.colOrder == nil {
		dt.colOrder = make([]int, len(dt.Columns))
		for i := range dt.colOrder {
			dt.colOrder[i] = i
		}
	}

	// Headings, their widths, and the display→logical map move together. Rows do
	// NOT: their cells stay in declared order, so a refresh re-supplies them in
	// that order and they keep appearing under the headings the operator moved.
	col := dt.Columns[from]
	dt.Columns = append(dt.Columns[:from], dt.Columns[from+1:]...)
	dt.Columns = append(dt.Columns, DataColumn{})
	copy(dt.Columns[insertAt+1:], dt.Columns[insertAt:])
	dt.Columns[insertAt] = col

	logical := dt.colOrder[from]
	dt.colOrder = append(dt.colOrder[:from], dt.colOrder[from+1:]...)
	dt.colOrder = append(dt.colOrder, 0)
	copy(dt.colOrder[insertAt+1:], dt.colOrder[insertAt:])
	dt.colOrder[insertAt] = logical

	// Widths are laid out on demand; a drag before the first paint has none yet,
	// and the renderer rebuilds them from Columns anyway.
	if len(dt.colWidths) == len(dt.Columns) {
		cw := dt.colWidths[from]
		dt.colWidths = append(dt.colWidths[:from], dt.colWidths[from+1:]...)
		dt.colWidths = append(dt.colWidths, 0)
		copy(dt.colWidths[insertAt+1:], dt.colWidths[insertAt:])
		dt.colWidths[insertAt] = cw
	}

	if dt.OnColumnsReordered != nil {
		dt.OnColumnsReordered()
	}
}

// ── Cursor ──

func (dt *DataTable) CursorAt(x, y float32) *platform.Cursor {
	if !dt.Enabled() || y >= dt.headerHeight() {
		return nil
	}
	colWidths := dt.getColWidths(dt.Width())
	if _, isEdge := dt.columnEdgeAtX(x, colWidths); isEdge {
		return platform.ResizeHorizontalCursor()
	}
	return nil
}

func (dt *DataTable) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !dt.Visible() {
		return
	}

	canvas.Save()
	defer canvas.Restore()

	disabled := !dt.Enabled()
	font := dt.resolveFont()
	ver := theme.ThemeVersion()

	// Theme version change — affects visible cells lazily via cellVer check
	_ = ver

	drawRowNumber := dt.ShowRowNumbers

	// Determine if scrollbar is needed (host owns extents/visibility)
	totalH := float32(len(dt.Rows)) * dt.RowHeight
	viewH := rect.Height - dt.headerHeight()
	dt.host.SetExtents(rect.Width, totalH, rect.Width, viewH)
	hasScrollbar := dt.host.VisibleV()

	// Recompute colWidths with the effective content width
	contentW := rect.Width
	if hasScrollbar {
		contentW -= dt.host.BarSize
	}
	colWidths := dt.getColWidths(contentW)

	// ── Background ──
	bgPaint := graphics.NewPaint()
	if disabled {
		bgPaint.SetColor(theme.DefaultTheme.DisabledBg)
	} else {
		bgPaint.SetColor(dt.BackgroundInk)
	}
	canvas.DrawRoundedRect(rect, dt.TableTheme.CornerRadius, dt.TableTheme.CornerRadius, bgPaint)

	// ── Clip content (exclude scrollbar zone only when scrollbar is shown) ──
	canvas.Save()
	cr := rect
	if hasScrollbar {
		cr.Width -= dt.host.BarSize
	}
	canvas.ClipRect(cr, pathop.Intersect, false)

	// ── Border ──
	borderPaint := graphics.NewPaint()
	borderPaint.SetStyle(paintstyle.Stroke)
	borderPaint.SetStrokeWidth(theme.BorderStrokeWidth)
	if disabled {
		borderPaint.SetColor(theme.DefaultTheme.DisabledBorder)
	} else {
		borderPaint.SetColor(dt.EdgeInk)
	}
	canvas.DrawRoundedRect(rect, dt.TableTheme.CornerRadius, dt.TableTheme.CornerRadius, borderPaint)

	// ── Header ──
	if dt.ShowHeader {
		headerY := rect.Y
		headerBg := graphics.NewPaint()
		headerBg.SetColor(dt.TableTheme.HeaderBgInk)
		canvas.DrawRect(graphics.NewRect(rect.X, headerY, rect.Width, dt.HeaderHeight), headerBg)

		colX := rect.X
		if drawRowNumber {
			rnX := colX
			rnPaint := graphics.NewPaint()
			rnPaint.SetColor(dt.TableTheme.SecondaryTextInk)
			rnText := graphics.NewText("#", &graphics.TextDecoration{Font: font, OnBackgroundInk: rnPaint.Color()})
			rnW := font.SimpleWidth("#")
			rnText.Draw(canvas, rnX+(dt.RowNumberWidth-rnW)/2, headerY+(dt.HeaderHeight-font.LineHeight())/2+font.Baseline())
			colX += dt.RowNumberWidth
		}
		for i := range dt.Columns {
			if i > 0 {
				div := graphics.NewPaint()
				div.SetStyle(paintstyle.Stroke)
				div.SetStrokeWidth(1)
				div.SetColor(dt.TableTheme.EdgeInk)
				canvas.DrawLine(colX, headerY, colX, headerY+dt.HeaderHeight, div)
			}
			cw := colWidths[i]
			if i < len(dt.Columns) {
				htxt := graphics.NewText(dt.Columns[i].Label, &graphics.TextDecoration{Font: font, OnBackgroundInk: dt.TableTheme.SecondaryTextInk})
				tw := font.SimpleWidth(dt.Columns[i].Label)
				tx := dt.cellTextX(i, colX, cw, tw)
				ty := headerY + (dt.HeaderHeight-font.LineHeight())/2 + font.Baseline()
				htxt.Draw(canvas, tx, ty)

				// Sort indicator — inside column right edge so it never bleeds into adjacent column
				if dt.sortCol == dt.columnLogical(i) {
					indicator := " ▲"
					if !dt.sortAsc {
						indicator = " ▼"
					}
					indW := font.SimpleWidth(indicator)
					indX := colX + cw - indW - 9
					indPaint := graphics.NewPaint()
					indPaint.SetColor(dt.TableTheme.AccentInk)
					it := graphics.NewText(indicator, &graphics.TextDecoration{Font: font, OnBackgroundInk: indPaint.Color()})
					it.Draw(canvas, indX, ty)
				}
			}
			colX += cw
		}

		// Bottom header line
		hl := graphics.NewPaint()
		hl.SetStyle(paintstyle.Stroke)
		hl.SetStrokeWidth(theme.BorderStrokeWidth)
		hl.SetColor(dt.EdgeInk)
		canvas.DrawLine(rect.X, headerY+dt.HeaderHeight, rect.X+rect.Width, headerY+dt.HeaderHeight, hl)
	}

	// ── Clip content below header ──
	canvas.Save()
	contentRect := graphics.NewRect(rect.X, rect.Y+dt.headerHeight(), rect.Width, rect.Height-dt.headerHeight())
	canvas.ClipRect(contentRect, pathop.Intersect, false)

	// ── Rows ──
	startRow, endRow := dt.visibleRowRange(rect)
	for ri := startRow; ri < endRow; ri++ {
		rowY := rect.Y + dt.headerHeight() - dt.ScrollOffset + float32(ri)*dt.RowHeight

		cellX := rect.X
		rowW := rect.Width
		if hasScrollbar {
			rowW -= dt.host.BarSize
		}
		rowRect := graphics.NewRect(rect.X, rowY, rowW, dt.RowHeight)

		// Alternating / selected / hovered background
		isSelected := dt.SelectedRow == ri
		isHovered := dt.HoveredRow == ri

		if !disabled && isSelected {
			selPaint := graphics.NewPaint()
			selPaint.SetColor(dt.SelectedInk)
			canvas.DrawRect(rowRect, selPaint)
		} else if !disabled && isHovered {
			hovPaint := graphics.NewPaint()
			hovPaint.SetColor(dt.HoverInk)
			canvas.DrawRect(rowRect, hovPaint)
		} else if dt.AlternatingRows && ri%2 == 1 {
			altPaint := graphics.NewPaint()
			altPaint.SetColor(dt.AlternateRowInk)
			canvas.DrawRect(rowRect, altPaint)
		}

		// Row number
		if drawRowNumber {
			rnText := fmt.Sprintf("%d", dt.RowNumberStart+ri)
			rnPaint := graphics.NewPaint()
			rnPaint.SetColor(dt.TableTheme.SecondaryTextInk)
			rnt := graphics.NewText(rnText, &graphics.TextDecoration{Font: font, OnBackgroundInk: rnPaint.Color()})
			rnW := font.SimpleWidth(rnText)
			rnt.Draw(canvas, cellX+(dt.RowNumberWidth-rnW)/2, rowY+(dt.RowHeight-font.LineHeight())/2+font.Baseline())
			cellX += dt.RowNumberWidth

			// Separator line
			sep := graphics.NewPaint()
			sep.SetStyle(paintstyle.Stroke)
			sep.SetStrokeWidth(1)
			sep.SetColor(dt.TableTheme.EdgeInk)
			canvas.DrawLine(cellX, rowY, cellX, rowY+dt.RowHeight, sep)
		}

		// Cells. ci is the DISPLAY position (the heading the operator sees); the
		// cell it shows is the LOGICAL one at that position.
		for ci := range colWidths {
			if ci >= len(dt.Columns) {
				break
			}
			cw := colWidths[ci]
			li := dt.columnLogical(ci)
			if li < 0 || li >= len(dt.Rows[ri].Cells) {
				cellX += cw
				continue
			}
			cell := &dt.Rows[ri].Cells[li]
			cellRect := graphics.NewRect(cellX, rowY, cw, dt.RowHeight)

			// Per-cell background
			if cell.BgColor != "" {
				bg := graphics.NewPaint()
				bg.SetColor(parseHexColor(cell.BgColor))
				canvas.DrawRect(cellRect, bg)
			}

			// Grid lines
			if dt.ShowGrid && ci > 0 {
				gl := graphics.NewPaint()
				gl.SetStyle(paintstyle.Stroke)
				gl.SetStrokeWidth(theme.BorderStrokeWidth)
				gl.SetColor(dt.TableTheme.GridInk)
				canvas.DrawLine(cellX, rowY, cellX, rowY+dt.RowHeight, gl)
			}

			// Handle font version change
			if cell.cellText == nil || cell.cellVer != ver {
				cell.cellText = dt.buildCellText(cell, font)
				cell.cellVer = ver
			}

			// Render
			if dt.OnCellRender != nil {
				dt.OnCellRender(canvas, cellRect, ri, li, cell)
			} else if cell.cellText != nil {
				canvas.Save()
				canvas.ClipRect(cellRect, pathop.Intersect, false)
				tw := font.SimpleWidth(cell.Text)
				tx := dt.cellTextX(ci, cellX, cw, tw)
				ty := rowY + (dt.RowHeight-font.LineHeight())/2 + font.Baseline()
				cell.cellText.Draw(canvas, tx, ty)
				canvas.Restore()
			}

			// Embedded widget — positioned at cell center preserving natural size
			if cell.Widget != nil {
				fw, fh := cell.Widget.FrameRect().Width, cell.Widget.FrameRect().Height
				if fw == 0 {
					fw = cw
				}
				if fh == 0 {
					fh = dt.RowHeight
				}
				wx := cellX + (cw-fw)/2
				wy := rowY + (dt.RowHeight-fh)/2
				widgetRect := graphics.NewRect(wx, wy, fw, fh)
				cell.Widget.SetFrameRect(widgetRect)
				cell.Widget.Paint(canvas, widgetRect)
			}

			cellX += cw
		}

		// Horizontal grid line
		if dt.ShowGrid {
			hl := graphics.NewPaint()
			hl.SetStyle(paintstyle.Stroke)
			hl.SetStrokeWidth(theme.BorderStrokeWidth)
			hl.SetColor(dt.TableTheme.GridInk)
			canvas.DrawLine(rect.X, rowY+dt.RowHeight, rect.X+rect.Width, rowY+dt.RowHeight, hl)
		}
	}
	canvas.Restore()

	// ── Drop indicator ──
	if dt.dragging && dt.dropCol >= 0 {
		dropX := rect.X
		if dt.ShowRowNumbers {
			dropX += dt.RowNumberWidth
		}
		for i := 0; i < dt.dropCol && i < len(colWidths); i++ {
			dropX += colWidths[i]
		}
		dropPaint := graphics.NewPaint()
		dropPaint.SetColor(dt.TableTheme.AccentInk)
		dropPaint.SetStyle(paintstyle.Stroke)
		dropPaint.SetStrokeWidth(2)
		canvas.DrawLine(dropX, rect.Y, dropX, rect.Y+rect.Height, dropPaint)
	}
	canvas.Restore() // restore content clip so scrollbar is not clipped out

	// ── Scrollbar (host; strip spans the FULL table height, aligned with
	// the table's top next to the header). Inset at the ends so the
	// thumb's rounded caps stay visible at both scroll extremes — flush
	// against the table's bottom the cap was clipped by the page edge. ──
	if dt.host.VisibleV() {
		// Strip spans the full table; LayoutBars insets the ends so the
		// thumb's rounded caps stay visible at both scroll extremes.
		dt.host.LayoutBars(rect.Width, rect.Height)
		_, dt.ScrollOffset = dt.host.Offsets()
		dt.host.PaintBars(canvas)
	}
}

// cellRect returns the bounding rect for a cell in DataTable-local coordinates.
func (dt *DataTable) cellRect(row, col int, frame graphics.Rect) graphics.Rect {
	colWidths := dt.getColWidths(frame.Width)
	rowY := dt.headerHeight() - dt.ScrollOffset + float32(row)*dt.RowHeight
	cellX := float32(0)
	if dt.ShowRowNumbers {
		cellX += dt.RowNumberWidth
	}
	for i := 0; i < col && i < len(colWidths); i++ {
		cellX += colWidths[i]
	}
	cw := float32(0)
	if col >= 0 && col < len(colWidths) {
		cw = colWidths[col]
	}
	return graphics.NewRect(cellX, rowY, cw, dt.RowHeight)
}

// routeToCellWidget routes a mouse down event to a cell widget.
func (dt *DataTable) routeToCellWidget(w meui.Widget, localX, localY float32, button int) bool {
	if el, ok := w.(meui.EventListener); ok {
		return el.OnMouseDown(localX, localY, button)
	}
	return false
}

func (dt *DataTable) OnMouseDown(x, y float32, button int) bool {
	if !dt.Enabled() {
		return false
	}

	frame := dt.FrameRect()
	// Bars first: the strip spans the full table height (next to the
	// header too), so do not gate bar hits on y >= headerHeight.
	if dt.host.HandleMouseDown(frame.Width, frame.Height, x, y, button) {
		return true
	}

	if button != mouseButtonLeft {
		if button == mouseButtonRight && y >= dt.headerHeight() {
			adjustedY := y - dt.headerHeight() + dt.ScrollOffset
			rowIdx := int(adjustedY / dt.RowHeight)
			if rowIdx >= 0 && rowIdx < len(dt.Rows) {
				dt.SelectedRow = rowIdx
				dt.MarkForRedraw()
				if dt.OnCellRightClicked != nil {
					colWidths := dt.getColWidths(dt.FrameRect().Width)
					colIdx := dt.columnLogical(dt.columnAtX(x, colWidths))
					dt.OnCellRightClicked(rowIdx, colIdx, x, y)
				}
			}
		}
		return true
	}

	if y >= dt.headerHeight() {
		adjustedY := y - dt.headerHeight() + dt.ScrollOffset
		rowIdx := int(adjustedY / dt.RowHeight)
		if rowIdx >= 0 && rowIdx < len(dt.Rows) {
			dt.SelectedRow = rowIdx
			colWidths := dt.getColWidths(frame.Width)
			colIdx := dt.columnAtX(x, colWidths)

			// Cell widget takes priority. colIdx is a display position; the cell it
			// addresses is logical, and that is what callbacks see.
			logicalCol := dt.columnLogical(colIdx)
			if logicalCol >= 0 && logicalCol < len(dt.Rows[rowIdx].Cells) {
				cell := &dt.Rows[rowIdx].Cells[logicalCol]
				if cell.Widget != nil {
					cellRect := dt.cellRect(rowIdx, colIdx, frame)
					fw, fh := cell.Widget.FrameRect().Width, cell.Widget.FrameRect().Height
					if fw == 0 {
						fw = cellRect.Width
					}
					if fh == 0 {
						fh = cellRect.Height
					}
					wx := cellRect.X + (cellRect.Width-fw)/2
					wy := cellRect.Y + (cellRect.Height-fh)/2
					widgetRect := graphics.NewRect(wx, wy, fw, fh)
					cell.Widget.SetFrameRect(widgetRect)
					if dt.routeToCellWidget(cell.Widget, x-wx, y-wy, button) {
						dt.activeCellWidget = cell.Widget
						dt.activeCellRow = rowIdx
						dt.activeCellCol = colIdx
						dt.MarkForRedraw()
						return true
					}
				}
				// Fire cell click
				if dt.OnCellClicked != nil {
					dt.OnCellClicked(rowIdx, logicalCol, cell)
				}
				// Fire double-click when the window reports a multi-click
				if win := dt.Window(); win != nil && win.ClickCount() >= 2 && dt.OnCellDoubleClicked != nil {
					dt.OnCellDoubleClicked(rowIdx, colIdx, cell)
				}
			}

			dt.MarkForRedraw()
			if dt.OnCellSelected != nil {
				dt.OnCellSelected(rowIdx, logicalCol)
			}
		}
		return true
	}

	if dt.ShowHeader && y < dt.HeaderHeight {
		colWidths := dt.getColWidths(frame.Width)
		// Resize: edge of column
		if col, isEdge := dt.columnEdgeAtX(x, colWidths); isEdge && col >= 0 && col < len(dt.Columns) {
			if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
				fmt.Printf("[dtResize] start col=%d at x=%.0f\n", col, x)
			}
			dt.resizingCol = col
			dt.resizeStartX = x
			dt.resizeStartWidth = dt.colWidths[col]
			// The dividers are the user's from here on (see userSized).
			dt.userSized = true
			return true
		}
		// Sort or reorder: click on column
		col := dt.columnAtX(x, colWidths)
		if col >= 0 && col < len(dt.Columns) {
			dt.dragCol = col
			dt.dragStartX = x
			dt.dragging = false
			dt.dropCol = -1
			dt.toggleSort(col)
			dt.MarkForRedraw()
			return true
		}
		return true
	}

	return false
}

func (dt *DataTable) OnMouseUp(x, y float32, button int) bool {
	if dt.activeCellWidget != nil {
		frame := dt.FrameRect()
		cr := dt.cellRect(dt.activeCellRow, dt.activeCellCol, frame)
		fw, fh := dt.activeCellWidget.FrameRect().Width, dt.activeCellWidget.FrameRect().Height
		if el, ok := dt.activeCellWidget.(meui.EventListener); ok {
			el.OnMouseUp(x-cr.X-(cr.Width-fw)/2, y-cr.Y-(cr.Height-fh)/2, button)
		}
		dt.activeCellWidget = nil
		return true
	}
	if dt.VScroll.dragging {
		dt.host.HandleMouseUp(x, y, button)
		return true
	}
	if dt.resizingCol >= 0 {
		dt.resizingCol = -1
		return true
	}
	if dt.dragging {
		dt.applyColumnReorder()
		dt.dragging = false
		dt.dragCol = -1
		dt.dropCol = -1
		dt.dragImage = nil
		meui.ClearDragOverlay()
		dt.MarkForRedraw()
		return true
	}
	if dt.dragCol >= 0 {
		dt.dragCol = -1
		dt.dragStartX = 0
	}
	return false
}

func (dt *DataTable) OnMouseMove(x, y float32) {
	if !dt.Enabled() {
		return
	}
	dt.host.UpdateHover(x, y) // overlay pills reveal on strip hover
	if dt.activeCellWidget != nil {
		frame := dt.FrameRect()
		cr := dt.cellRect(dt.activeCellRow, dt.activeCellCol, frame)
		fw, fh := dt.activeCellWidget.FrameRect().Width, dt.activeCellWidget.FrameRect().Height
		if el, ok := dt.activeCellWidget.(meui.EventListener); ok {
			el.OnMouseMove(x-cr.X-(cr.Width-fw)/2, y-cr.Y-(cr.Height-fh)/2)
		}
		return
	}
	if dt.VScroll.dragging {
		dt.host.HandleMouseMove(x, y)
		return
	}

	// Column resize
	if dt.resizingCol >= 0 {
		if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
			fmt.Printf("[dtResize] move x=%.0f start=%.0f\n", x, dt.resizeStartX)
		}
		diff := x - dt.resizeStartX
		newWidth := dt.resizeStartWidth + diff
		if newWidth < 20 {
			newWidth = 20
		}
		dt.colWidths[dt.resizingCol] = newWidth
		dt.Columns[dt.resizingCol].Width = newWidth
		dt.MarkForRedraw()
		return
	}

	// Column drag/reorder
	if dt.dragCol >= 0 {
		if !dt.dragging {
			dx := x - dt.dragStartX
			if dx < 0 {
				dx = -dx
			}
			if dx > 5 {
				dt.dragging = true
				dt.renderColumnDragImage()
				if dt.dragImage != nil {
					winX, winY := dt.FromLocal(x, y)
					meui.SetDragOverlay(dt.dragImage, winX-10, winY-4, dt.dragImageW, dt.dragImageH)
				}
			}
		}
		if dt.dragging {
			frame := dt.FrameRect()
			colWidths := dt.getColWidths(frame.Width)
			dt.dropCol = dt.columnAtX(x, colWidths)
			if dt.dropCol < 0 {
				// Outside the headings: the gap before the first or after the last,
				// so dragging past the end can put a column last.
				end := float32(0)
				if dt.ShowRowNumbers {
					end += dt.RowNumberWidth
				}
				for _, w := range colWidths {
					end += w
				}
				if x >= end {
					dt.dropCol = len(dt.Columns)
				} else {
					dt.dropCol = 0
				}
			}
			if dt.dragImage != nil {
				winX, winY := dt.FromLocal(x, y)
				meui.SetDragOverlay(dt.dragImage, winX-10, winY-4, dt.dragImageW, dt.dragImageH)
			}
			dt.MarkForRedraw()
		}
		return
	}

	if y < dt.headerHeight() {
		if dt.HoveredRow != -1 {
			dt.HoveredRow = -1
			dt.MarkForRedraw()
		}
		return
	}
	adjustedY := y - dt.headerHeight() + dt.ScrollOffset
	rowIdx := int(adjustedY / dt.RowHeight)
	oldHovered := dt.HoveredRow
	dt.HoveredRow = -1
	if rowIdx >= 0 && rowIdx < len(dt.Rows) {
		dt.HoveredRow = rowIdx
	}
	if oldHovered != dt.HoveredRow {
		dt.MarkForRedraw()
	}
}

func (dt *DataTable) OnMouseWheel(x, y float32, dx, dy float32) bool {
	if !dt.Enabled() || dy == 0 {
		return false
	}
	totalH := float32(len(dt.Rows)) * dt.RowHeight
	if totalH <= 0 {
		return false
	}
	frame := dt.FrameRect()
	dt.host.SetExtents(frame.Width, totalH, frame.Width, frame.Height-dt.headerHeight())
	dt.host.ScrollAmount = dt.RowHeight * 3
	if !dt.host.HandleWheel(dx, dy) {
		return false
	}
	_, dt.ScrollOffset = dt.host.Offsets()
	dt.MarkForRedraw()
	return true
}

func (dt *DataTable) maxScrollOffset() float32 {
	totalH := float32(len(dt.Rows)) * dt.RowHeight
	viewH := dt.Height() - dt.headerHeight()
	maxScroll := totalH - viewH
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}

func (dt *DataTable) scrollOffsetToRow(row int) {
	maxScroll := dt.maxScrollOffset()
	if maxScroll <= 0 {
		return
	}
	rowTop := float32(row) * dt.RowHeight
	rowBottom := rowTop + dt.RowHeight
	viewH := dt.Height() - dt.headerHeight()
	if rowTop < dt.ScrollOffset {
		dt.ScrollOffset = rowTop
	} else if rowBottom > dt.ScrollOffset+viewH {
		dt.ScrollOffset = rowBottom - viewH
	}
	if dt.ScrollOffset < 0 {
		dt.ScrollOffset = 0
	}
	if dt.ScrollOffset > maxScroll {
		dt.ScrollOffset = maxScroll
	}
	dt.VScroll.SetValue(dt.ScrollOffset / maxScroll)
}

func (dt *DataTable) syncScrollBar() {
	maxScroll := dt.maxScrollOffset()
	if maxScroll > 0 {
		dt.VScroll.SetValue(dt.ScrollOffset / maxScroll)
	} else {
		dt.VScroll.SetValue(0)
	}
}

// OnKeyRepeat is a key the user is HOLDING — Backspace over a word, an arrow along a row, a scrub.
// The window classifies an OS auto-repeat as a repeat rather than a fresh press (a widget without
// this method is asked once, deliberately), so this is what makes a held key work here.
func (dt *DataTable) OnKeyRepeat(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	return dt.OnKeyDown(keyCode, mod)
}

func (dt *DataTable) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if !dt.Enabled() || len(dt.Rows) == 0 {
		return false
	}
	switch keyCode {
	case platform.KeyUp:
		if dt.SelectedRow > 0 {
			dt.SelectedRow--
			dt.scrollOffsetToRow(dt.SelectedRow)
			if dt.OnCellSelected != nil {
				dt.OnCellSelected(dt.SelectedRow, -1)
			}
			dt.MarkForRedraw()
		}
		return true
	case platform.KeyDown:
		if dt.SelectedRow < len(dt.Rows)-1 {
			dt.SelectedRow++
			dt.scrollOffsetToRow(dt.SelectedRow)
			if dt.OnCellSelected != nil {
				dt.OnCellSelected(dt.SelectedRow, -1)
			}
			dt.MarkForRedraw()
		}
		return true
	case platform.KeyReturn, platform.KeyNumPadEnter:
		if dt.SelectedRow >= 0 && dt.SelectedRow < len(dt.Rows) {
			if mod.ShiftDown() && dt.OnDuplicate != nil {
				dt.OnDuplicate(dt.SelectedRow)
			} else if dt.OnCellClicked != nil {
				first := dt.columnLogical(0)
				if first >= 0 && first < len(dt.Rows[dt.SelectedRow].Cells) {
					dt.OnCellClicked(dt.SelectedRow, first, &dt.Rows[dt.SelectedRow].Cells[first])
				}
			}
		}
		return true
	case platform.KeyDelete:
		if dt.SelectedRow >= 0 && dt.SelectedRow < len(dt.Rows) && dt.OnDelete != nil {
			dt.OnDelete(dt.SelectedRow, mod)
		}
		return true
	case platform.KeyHome:
		dt.SelectedRow = 0
		dt.ScrollOffset = 0
		dt.syncScrollBar()
		if dt.OnCellSelected != nil {
			dt.OnCellSelected(0, -1)
		}
		dt.MarkForRedraw()
		return true
	case platform.KeyEnd:
		dt.SelectedRow = len(dt.Rows) - 1
		dt.ScrollOffset = dt.maxScrollOffset()
		if dt.OnCellSelected != nil {
			dt.OnCellSelected(dt.SelectedRow, -1)
		}
		dt.MarkForRedraw()
		return true
	}
	return false
}

func (dt *DataTable) Focusable() bool {
	return dt.Enabled()
}

// ── QML integration ──────────────────────────────────────────

func toFloat32(v interface{}) float32 {
	switch n := v.(type) {
	case float64:
		return float32(n)
	case int:
		return float32(n)
	}
	return 0
}

func (dt *DataTable) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "columns":
		if os.Getenv("MEUI_DT_DEBUG") != "" {
			fmt.Printf("[dt] columns value type=%T len=%d\n", value, func() int {
				if a, ok := value.([]interface{}); ok {
					return len(a)
				}
				return -1
			}())
		}
		if arr, ok := value.([]interface{}); ok {
			cols := make([]DataColumn, len(arr))
			for i, v := range arr {
				if m, ok := v.(map[string]interface{}); ok {
					if label, ok := m["label"].(string); ok {
						cols[i].Label = label
					} else {
						cols[i].Label = fmt.Sprintf("%v", v)
					}
					if typeStr, ok := m["type"].(string); ok {
						switch strings.ToLower(typeStr) {
						case "number", "numeric":
							cols[i].Type = ColNumber
						case "percent", "percentage":
							cols[i].Type = ColPercent
						default:
							cols[i].Type = ColText
						}
					}
					if wv, ok := m["width"]; ok {
						cols[i].Width = toFloat32(wv)
					}
				} else {
					cols[i].Label = fmt.Sprintf("%v", v)
				}
			}
			dt.SetColumns(cols)
		}
	case "rows":
		if arr, ok := value.([]interface{}); ok {
			rows := make([]DataRow, len(arr))
			for ri, rv := range arr {
				if cells, ok := rv.([]interface{}); ok {
					row := make([]DataCell, len(cells))
					for ci, cv := range cells {
						if cellMap, ok := cv.(map[string]interface{}); ok {
							if text, ok := cellMap["text"].(string); ok {
								row[ci].Text = text
							}
							if color, ok := cellMap["color"].(string); ok {
								row[ci].Color = color
							}
							if bg, ok := cellMap["bgColor"].(string); ok {
								row[ci].BgColor = bg
							}
						} else {
							row[ci].Text = fmt.Sprintf("%v", cv)
						}
					}
					rows[ri] = DataRow{Cells: row}
				} else if cellMap, ok := rv.(map[string]interface{}); ok {
					// Single object: treat as a row with multiple columns
					var row []DataCell
					for _, col := range dt.Columns {
						if val, exists := cellMap[col.Label]; exists {
							row = append(row, DataCell{Text: fmt.Sprintf("%v", val)})
						}
					}
					rows[ri] = DataRow{Cells: row}
				}
			}
			dt.SetRows(rows)
		}
	case "alternatingRows":
		if b, ok := value.(bool); ok {
			dt.AlternatingRows = b
		}
	case "showGrid":
		if b, ok := value.(bool); ok {
			dt.ShowGrid = b
		}
	case "showHeader":
		if b, ok := value.(bool); ok {
			dt.ShowHeader = b
		}
	case "showRowNumbers":
		if b, ok := value.(bool); ok {
			dt.ShowRowNumbers = b
		}
	}
	dt.MarkForRedraw()
}
