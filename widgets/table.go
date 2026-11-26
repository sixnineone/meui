package widgets

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

type TableCell struct {
	Text     string
	IsCheck  bool
	Checked  bool
	Icon     string
	IconSize float32
	Color    string
	BgColor  string
	OnRender func(canvas *graphics.Canvas, rect graphics.Rect, selected bool)
	cellText *graphics.Text
}

type Table struct {
	meui.BaseWidget
	*theme.TableTheme
	Columns             []string
	Rows                [][]TableCell
	SelectedRows        map[int]bool
	LastClickedRow      int
	SelectedCol         int
	HoveredRow          int
	ScrollOffset        float32
	RowHeight           float32
	HeaderHeight        float32
	AlternatingRows     bool
	ShowGrid            bool
	ShowHorizontalLines bool
	ShowVerticalLines   bool
	RowSelection        bool
	ColSelection        bool
	IntercellSpacing    float32
	GridColor           graphics.Color
	OnCellSelected      func(row, col int, cell *TableCell)

	colWidths        []float32
	colWidthsTotal   float32
	resizingCol      int
	resizeStartX     float32
	resizeStartWidth float32

	sortCol  int
	sortAsc  bool
	origRows [][]TableCell

	dragCol    int
	dragStartX float32
	dragging   bool
	dropCol    int

	editing    bool
	editingRow int
	editingCol int
	editText   string
	editField  *TextField

	ctxMenu *ContextMenu

	dragImage  *graphics.Image
	dragImageW float32
	dragImageH float32

	cache         meui.CacheBase
	VScroll       *ScrollBar
	host          *ScrollbarHost
	headerTexts   []*graphics.Text
	dragTextCache *graphics.Text
	textVer       uint64
}

func NewTable() *Table {
	t := &Table{
		BaseWidget:          *meui.NewBaseWidget(),
		TableTheme:          theme.DefaultTableTheme,
		RowHeight:           24,
		HeaderHeight:        28,
		LastClickedRow:      -1,
		HoveredRow:          -1,
		SelectedCol:         -1,
		ShowGrid:            true,
		ShowHorizontalLines: true,
		ShowVerticalLines:   true,
		RowSelection:        true,
		IntercellSpacing:    0,
		GridColor:           graphics.Color(0xFF3C3C3C),
		resizingCol:         -1,
		sortCol:             -1,
		dragCol:             -1,
		dropCol:             -1,
	}
	t.host = NewScrollbarHost()
	t.host.ShowH = false // tables scroll vertically
	t.VScroll = t.host.VScroll
	t.VScroll.SetParent(t)
	t.host.SetOnChange(func() {
		_, offY := t.host.Offsets()
		t.ScrollOffset = offY
		t.MarkForRedraw()
	})
	return t
}

func (t *Table) SetColumns(cols []string) *Table {
	t.Columns = cols
	t.colWidths = nil
	t.buildHeaderTexts()
	t.cache.Invalidate()
	t.MarkForRedraw()
	return t
}

func (t *Table) buildHeaderTexts() {
	font := t.TableTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	t.headerTexts = make([]*graphics.Text, len(t.Columns))
	for i, col := range t.Columns {
		if col != "" {
			t.headerTexts[i] = graphics.NewText(col, &graphics.TextDecoration{Font: font, OnBackgroundInk: t.TableTheme.SecondaryTextInk})
		}
	}
}

func (t *Table) buildCellTextsForRow(row []TableCell) {
	font := t.TableTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	ink := t.TableTheme.TextInk
	for ci := range row {
		cell := &row[ci]
		if cell.Text != "" && !cell.IsCheck {
			cell.cellText = graphics.NewText(cell.Text, &graphics.TextDecoration{Font: font, OnBackgroundInk: ink})
		}
	}
}

func (t *Table) buildAllCellTexts() {
	for ri := range t.Rows {
		t.buildCellTextsForRow(t.Rows[ri])
	}
}

func (t *Table) SetRows(rows [][]TableCell) *Table {
	t.Rows = rows
	t.buildAllCellTexts()
	t.origRows = nil
	t.SelectedRows = nil
	t.LastClickedRow = -1
	t.sortCol = -1
	t.sortAsc = true
	t.cache.Invalidate()
	t.MarkForRedraw()
	return t
}

func (t *Table) AddRow(cells []TableCell) *Table {
	t.Rows = append(t.Rows, cells)
	t.buildCellTextsForRow(t.Rows[len(t.Rows)-1])
	t.origRows = nil
	t.cache.Invalidate()
	t.MarkForRedraw()
	return t
}

func (t *Table) SetCell(row, col int, cell TableCell) {
	if row >= 0 && row < len(t.Rows) && col >= 0 && col < len(t.Columns) {
		if col < len(t.Rows[row]) {
			t.Rows[row][col] = cell
		} else {
			t.Rows[row] = append(t.Rows[row], cell)
		}
		cell.cellText = nil
		if cell.Text != "" && !cell.IsCheck {
			font := t.TableTheme.Font
			if font == nil {
				font = theme.DefaultTheme.BaseFont
				if font == nil {
					font = graphics.SystemFont
				}
			}
			t.Rows[row][col].cellText = graphics.NewText(cell.Text, &graphics.TextDecoration{Font: font, OnBackgroundInk: t.TableTheme.TextInk})
		}
		t.cache.Invalidate()
		t.MarkForRedraw()
	}
}

func (t *Table) getColumnWidths(totalWidth float32) []float32 {
	if len(t.Columns) == 0 {
		return nil
	}
	if totalWidth <= 0 {
		return nil
	}
	// Recompute when the column SET or the available WIDTH changes: the
	// first layout pass can run at a tiny width (before the parent
	// assigns the real frame), and freezing those widths rendered every
	// column at minimum width with overlapping text.
	if t.colWidths == nil || len(t.colWidths) != len(t.Columns) || t.colWidthsTotal != totalWidth {
		t.colWidths = make([]float32, len(t.Columns))
		equalWidth := totalWidth / float32(len(t.Columns))
		for i := range t.colWidths {
			t.colWidths[i] = equalWidth
		}
		t.colWidthsTotal = totalWidth
	}
	return t.colWidths
}

func (t *Table) columnAtX(x float32) int {
	colWidths := t.getColumnWidths(t.Width())
	cx := float32(0)
	for i, w := range colWidths {
		if x >= cx && x < cx+w {
			return i
		}
		cx += w
	}
	return -1
}

// renderColumnDragImage creates a ghost image of the column being dragged.
func (t *Table) renderColumnDragImage() {
	if t.dragCol < 0 || t.dragCol >= len(t.Columns) {
		return
	}
	font := t.TableTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	colWidths := t.getColumnWidths(t.Width())
	if t.dragCol >= len(colWidths) {
		return
	}
	cw := colWidths[t.dragCol]
	ch := t.HeaderHeight
	w := int(cw) + 4
	h := int(ch) + 4
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	img, err := graphics.NewImageFromDrawing(offscreenDevice(t), w, h, 72, func(c *graphics.Canvas) {
		clear := graphics.NewPaint()
		clear.SetColor(graphics.Color(0x00000000))
		c.DrawRect(graphics.NewRect(0, 0, float32(w), float32(h)), clear)
		// Background
		bg := graphics.NewPaint()
		bg.SetColor(t.TableTheme.HeaderBgInk)
		c.DrawRect(graphics.NewRect(2, 2, cw, ch), bg)
		// Border
		bd := graphics.NewPaint()
		bd.SetStyle(paintstyle.Stroke)
		bd.SetStrokeWidth(1)
		bd.SetColor(t.TableTheme.EdgeInk)
		c.DrawRect(graphics.NewRect(2, 2, cw, ch), bd)
		// Text
		tp := graphics.NewPaint()
		tp.SetColor(t.TableTheme.TextInk)
		textW := font.SimpleWidth(t.Columns[t.dragCol])
		textX := 2 + (cw-textW)/2
		textY := 2 + (ch-font.LineHeight())/2 + font.Baseline()
		if t.dragTextCache == nil {
			t.dragTextCache = graphics.NewText(t.Columns[t.dragCol], &graphics.TextDecoration{Font: font, OnBackgroundInk: tp.Color()})
		}
		t.dragTextCache.Draw(c, textX, textY)
	})
	if err != nil {
		t.dragImage = nil
		return
	}
	t.dragImage = img
	t.dragImageW = float32(w)
	t.dragImageH = float32(h)
}

func (t *Table) columnEdgeAtX(x float32) (col int, isEdge bool) {
	colWidths := t.getColumnWidths(t.Width())
	cx := float32(0)
	for i, w := range colWidths {
		// Left 5px of this column and right 5px of this column
		if (x >= cx && x <= cx+5) || (x >= cx+w-5 && x <= cx+w) {
			return i, true
		}
		cx += w
	}
	return -1, false
}

func (t *Table) getCellText(row []TableCell, col int) string {
	if col >= 0 && col < len(row) {
		return row[col].Text
	}
	return ""
}

func (t *Table) sortRows() {
	if t.sortCol < 0 || t.sortCol >= len(t.Columns) || len(t.Rows) == 0 {
		return
	}
	if t.origRows == nil {
		t.origRows = make([][]TableCell, len(t.Rows))
		for i := range t.Rows {
			t.origRows[i] = make([]TableCell, len(t.Rows[i]))
			copy(t.origRows[i], t.Rows[i])
		}
	}
	sort.SliceStable(t.Rows, func(i, j int) bool {
		a := strings.ToLower(t.getCellText(t.Rows[i], t.sortCol))
		b := strings.ToLower(t.getCellText(t.Rows[j], t.sortCol))
		if t.sortAsc {
			return a < b
		}
		return a > b
	})
	t.cache.Invalidate()
}

func (t *Table) unsortRows() {
	if t.origRows != nil {
		t.Rows = make([][]TableCell, len(t.origRows))
		for i := range t.origRows {
			t.Rows[i] = make([]TableCell, len(t.origRows[i]))
			copy(t.Rows[i], t.origRows[i])
		}
		t.origRows = nil
	}
	t.cache.Invalidate()
}

func (t *Table) applyColumnReorder() {
	from := t.dragCol
	to := t.dropCol
	if from < 0 || to < 0 || from == to || from >= len(t.Columns) || to >= len(t.Columns) {
		return
	}

	insertAt := to
	if from < to {
		insertAt = to - 1
	}

	col := t.Columns[from]
	t.Columns = append(t.Columns[:from], t.Columns[from+1:]...)
	t.Columns = append(t.Columns, "")
	copy(t.Columns[insertAt+1:], t.Columns[insertAt:])
	t.Columns[insertAt] = col

	cw := t.colWidths[from]
	t.colWidths = append(t.colWidths[:from], t.colWidths[from+1:]...)
	t.colWidths = append(t.colWidths, 0)
	copy(t.colWidths[insertAt+1:], t.colWidths[insertAt:])
	t.colWidths[insertAt] = cw

	for ri := range t.Rows {
		if from < len(t.Rows[ri]) {
			cell := t.Rows[ri][from]
			t.Rows[ri] = append(t.Rows[ri][:from], t.Rows[ri][from+1:]...)
			t.Rows[ri] = append(t.Rows[ri], TableCell{})
			copy(t.Rows[ri][insertAt+1:], t.Rows[ri][insertAt:])
			t.Rows[ri][insertAt] = cell
		}
	}
	t.cache.Invalidate()
}

func (t *Table) startEditing(row, col int) {
	if !t.Enabled() || row < 0 || row >= len(t.Rows) || col < 0 || col >= len(t.Columns) {
		return
	}
	if col >= len(t.Rows[row]) {
		return
	}
	t.editing = true
	t.editingRow = row
	t.editingCol = col
	t.editText = t.Rows[row][col].Text
	t.cache.Invalidate()
	t.MarkForRedraw()
}

func (t *Table) commitEdit() {
	if !t.editing {
		return
	}
	if t.editingRow >= 0 && t.editingRow < len(t.Rows) &&
		t.editingCol >= 0 && t.editingCol < len(t.Rows[t.editingRow]) {
		t.Rows[t.editingRow][t.editingCol].Text = t.editText
	}
	t.editing = false
	t.editingRow = -1
	t.editingCol = -1
	t.editText = ""
	t.cache.Invalidate()
	t.MarkForRedraw()
}

func (t *Table) showContextMenu(x, y float32) {
	adjustedY := y - t.HeaderHeight + t.ScrollOffset
	rowIdx := int(adjustedY / (t.RowHeight + t.IntercellSpacing))
	colIdx := t.columnAtX(x)

	if rowIdx < 0 || rowIdx >= len(t.Rows) {
		rowIdx = -1
	}
	if colIdx < 0 || colIdx >= len(t.Columns) {
		colIdx = -1
	}

	menuRow, menuCol := rowIdx, colIdx

	var items []MenuItem

	if t.sortCol >= 0 {
		sortCol := t.sortCol
		items = append(items, MenuItem{
			Label:   "Sort Ascending",
			Enabled: true,
			Action: func() {
				if t.sortCol == sortCol && !t.sortAsc {
					t.sortAsc = true
					t.sortRows()
					t.MarkForRedraw()
				}
			},
		})
		items = append(items, MenuItem{
			Label:   "Sort Descending",
			Enabled: true,
			Action: func() {
				if t.sortCol == sortCol && t.sortAsc {
					t.sortAsc = false
					t.sortRows()
					t.MarkForRedraw()
				}
			},
		})
		items = append(items, MenuItem{IsSep: true})
	}

	items = append(items, MenuItem{
		Label:   "Copy Cell",
		Enabled: true,
		Action: func() {
			if menuRow >= 0 && menuRow < len(t.Rows) && menuCol >= 0 && menuCol < len(t.Rows[menuRow]) {
				platform.GlobalClipboard.SetText(t.Rows[menuRow][menuCol].Text)
			}
		},
	})
	items = append(items, MenuItem{
		Label:   "Copy Row",
		Enabled: true,
		Action: func() {
			if menuRow >= 0 && menuRow < len(t.Rows) {
				var parts []string
				for _, c := range t.Rows[menuRow] {
					parts = append(parts, c.Text)
				}
				platform.GlobalClipboard.SetText(strings.Join(parts, "\t"))
			}
		},
	})

	if t.editing {
		items = append(items, MenuItem{IsSep: true})
		items = append(items, MenuItem{
			Label:   "Edit Cell",
			Enabled: true,
			Action: func() {
				if menuRow >= 0 && menuCol >= 0 {
					t.startEditing(menuRow, menuCol)
				}
			},
		})
	}

	if t.ctxMenu == nil {
		t.ctxMenu = NewContextMenu()
	}
	t.ctxMenu.SetItems(items)
	if win := t.Window(); win != nil {
		t.ctxMenu.SetOnDismiss(func() {
			win.DismissPopupContextMenu()
		})
		winX, winY := t.FromLocal(x, y)
		win.ShowPopupContextMenu(t.ctxMenu, winX, winY)
	}
	t.MarkForRedraw()
}

func parseHexColor(s string) graphics.Color {
	if s == "" {
		return 0
	}
	s = strings.TrimPrefix(s, "#")
	if len(s) == 6 {
		var r, g, b uint8
		fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b)
		return graphics.Color(0xFF000000) | graphics.Color(r)<<16 | graphics.Color(g)<<8 | graphics.Color(b)
	}
	if len(s) == 8 {
		var a, r, g, b uint8
		fmt.Sscanf(s, "%02x%02x%02x%02x", &a, &r, &g, &b)
		return graphics.Color(a)<<24 | graphics.Color(r)<<16 | graphics.Color(g)<<8 | graphics.Color(b)
	}
	return 0
}

func (t *Table) CursorAt(x, y float32) *platform.Cursor {
	if !t.Enabled() || y >= t.HeaderHeight {
		return nil
	}
	if _, isEdge := t.columnEdgeAtX(x); isEdge {
		return platform.ResizeHorizontalCursor()
	}
	return nil
}

func (t *Table) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !t.Visible() {
		return
	}

	canvas.Save()

	t.cache.GetCached(canvas, rect, func(c *graphics.Canvas, r graphics.Rect) {
		disabled := !t.Enabled()

		bgPaint := graphics.NewPaint()
		if disabled {
			bgPaint.SetColor(theme.DefaultTheme.DisabledBg)
		} else {
			bgPaint.SetColor(t.TableTheme.BackgroundInk)
		}
		c.DrawRoundedRect(r, t.TableTheme.CornerRadius, t.TableTheme.CornerRadius, bgPaint)

		borderPaint := graphics.NewPaint()
		borderPaint.SetStyle(paintstyle.Stroke)
		borderPaint.SetStrokeWidth(theme.BorderStrokeWidth)
		if disabled {
			borderPaint.SetColor(theme.DefaultTheme.DisabledBorder)
		} else {
			borderPaint.SetColor(t.TableTheme.EdgeInk)
		}
		c.DrawRoundedRect(r, t.TableTheme.CornerRadius, t.TableTheme.CornerRadius, borderPaint)

		cr := r
		cr.Width -= theme.DefaultTheme.ScrollbarWidth
		c.ClipRect(cr, pathop.Intersect, false)

		font := t.TableTheme.Font
		if font == nil {
			font = theme.DefaultTheme.BaseFont
			if font == nil {
				font = graphics.SystemFont
			}
		}

		colWidths := t.getColumnWidths(r.Width)

		ver := theme.ThemeVersion()
		if t.textVer != ver {
			t.buildHeaderTexts()
			t.buildAllCellTexts()
			t.dragTextCache = nil
			t.textVer = ver
		}

		headerY := r.Y
		headerRect := graphics.NewRect(r.X, headerY, r.Width, t.HeaderHeight)
		headerBgPaint := graphics.NewPaint()
		headerBgPaint.SetColor(t.TableTheme.HeaderBgInk)
		c.DrawRect(headerRect, headerBgPaint)

		colX := r.X
		for i, col := range t.Columns {
			if i > 0 {
				divPaint := graphics.NewPaint()
				divPaint.SetStyle(paintstyle.Stroke)
				divPaint.SetStrokeWidth(1)
				divPaint.SetColor(t.TableTheme.EdgeInk)
				c.DrawLine(colX, headerY, colX, headerY+t.HeaderHeight, divPaint)
			}

			textY := headerY + (t.HeaderHeight-font.LineHeight())/2 + font.Baseline()
			textW := font.SimpleWidth(col)
			textX := colX + (colWidths[i]-textW)/2
			if i < len(t.headerTexts) && t.headerTexts[i] != nil {
				t.headerTexts[i].Draw(c, textX, textY)
			} else {
				textPaint := graphics.NewPaint()
				textPaint.SetColor(t.TableTheme.SecondaryTextInk)
				ht := graphics.NewText(col, &graphics.TextDecoration{Font: font, OnBackgroundInk: textPaint.Color()})
				ht.Draw(c, textX, textY)
			}

			if t.sortCol == i {
				indicator := " ▲"
				if !t.sortAsc {
					indicator = " ▼"
				}
				indPaint := graphics.NewPaint()
				indPaint.SetColor(t.TableTheme.AccentInk)
				indX := textX + textW + 2
				it := graphics.NewText(indicator, &graphics.TextDecoration{Font: font, OnBackgroundInk: indPaint.Color()})
				it.Draw(c, indX, textY)
			}

			colX += colWidths[i]
		}

		innerY := r.Y + t.HeaderHeight - t.ScrollOffset
		for rowIdx, row := range t.Rows {
			if innerY+t.RowHeight < r.Y || innerY > r.Y+r.Height {
				innerY += t.RowHeight + t.IntercellSpacing
				continue
			}

			rowRect := graphics.NewRect(r.X, innerY, r.Width, t.RowHeight)

			isSelected := t.RowSelection && t.SelectedRows[rowIdx]

			if !disabled && isSelected {
				selPaint := graphics.NewPaint()
				selPaint.SetColor(t.TableTheme.SelectedInk)
				c.DrawRect(rowRect, selPaint)
			} else if !disabled && rowIdx == t.HoveredRow {
				hoverPaint := graphics.NewPaint()
				hoverPaint.SetColor(t.TableTheme.HoverInk)
				c.DrawRect(rowRect, hoverPaint)
			} else if t.AlternatingRows && rowIdx%2 == 1 {
				altPaint := graphics.NewPaint()
				altPaint.SetColor(t.TableTheme.AlternateRowInk)
				c.DrawRect(rowRect, altPaint)
			}

			if t.ShowHorizontalLines {
				linePaint := graphics.NewPaint()
				linePaint.SetStyle(paintstyle.Stroke)
				linePaint.SetStrokeWidth(theme.BorderStrokeWidth)
				linePaint.SetColor(t.GridColor)
				c.DrawLine(r.X, innerY+t.RowHeight, r.X+r.Width, innerY+t.RowHeight, linePaint)
			}

			colX = r.X
			for colIdx, cell := range row {
				if colIdx >= len(t.Columns) {
					break
				}

				cellRect := graphics.NewRect(colX, innerY, colWidths[colIdx], t.RowHeight)

				if t.ShowVerticalLines && colIdx > 0 {
					linePaint := graphics.NewPaint()
					linePaint.SetStyle(paintstyle.Stroke)
					linePaint.SetStrokeWidth(theme.BorderStrokeWidth)
					linePaint.SetColor(t.GridColor)
					c.DrawLine(colX, innerY, colX, innerY+t.RowHeight, linePaint)
				}

				if cell.BgColor != "" {
					bg := graphics.NewPaint()
					bg.SetColor(parseHexColor(cell.BgColor))
					c.DrawRect(cellRect, bg)
				}

				isEditing := t.editing && rowIdx == t.editingRow && colIdx == t.editingCol

				if cell.OnRender != nil {
					cell.OnRender(c, cellRect, isSelected)
				} else if cell.IsCheck {
					checkMark := "☐"
					if cell.Checked {
						checkMark = "☑"
					}
					checkPaint := graphics.NewPaint()
					if disabled {
						checkPaint.SetColor(theme.DefaultTheme.DisabledText)
					} else if isSelected {
						checkPaint.SetColor(t.TableTheme.SelectedTextInk)
					} else {
						checkPaint.SetColor(t.TableTheme.TextInk)
					}
					cx := colX + (colWidths[colIdx]-font.SimpleWidth(checkMark))/2
					cy := innerY + (t.RowHeight-font.LineHeight())/2 + font.Baseline()
					ct := graphics.NewText(checkMark, &graphics.TextDecoration{Font: font, OnBackgroundInk: checkPaint.Color()})
					ct.Draw(c, cx, cy)
				} else {
					iconOffset := float32(0)
					if cell.Icon != "" {
						iconSize := cell.IconSize
						if iconSize <= 0 {
							iconSize = 14
						}
						if doc, err := meui.LoadSVGByName(cell.Icon); err == nil {
							meui.DrawSVG(c, doc, colX+4, innerY+(t.RowHeight-iconSize)/2, iconSize, iconSize)
							iconOffset = iconSize + 4
						} else {
							font := t.TableTheme.Font
							if font == nil {
								font = theme.DefaultTheme.BaseFont
								if font == nil {
									font = graphics.SystemFont
								}
							}
							ip := graphics.NewPaint()
							ip.SetColor(t.TableTheme.TextInk)
							it := graphics.NewText(cell.Icon, &graphics.TextDecoration{Font: font, OnBackgroundInk: ip.Color()})
							it.Draw(c, colX+4, innerY+(t.RowHeight-font.LineHeight())/2+font.Baseline())
							iconOffset = font.SimpleWidth(cell.Icon) + 4
						}
					}

					displayText := cell.Text
					if isEditing {
						displayText = t.editText
					}

					textX := colX + 5 + iconOffset
					textY := innerY + (t.RowHeight-font.LineHeight())/2 + font.Baseline()
					if cell.cellText != nil && !isEditing && cell.Color == "" {
						cell.cellText.Draw(c, textX, textY)
					} else {
						textPaint := graphics.NewPaint()
						if disabled {
							textPaint.SetColor(theme.DefaultTheme.DisabledText)
						} else if isSelected {
							textPaint.SetColor(t.TableTheme.SelectedTextInk)
						} else if cell.Color != "" {
							textPaint.SetColor(parseHexColor(cell.Color))
						} else {
							textPaint.SetColor(t.TableTheme.TextInk)
						}
						ct := graphics.NewText(displayText, &graphics.TextDecoration{Font: font, OnBackgroundInk: textPaint.Color()})
						ct.Draw(c, textX, textY)
					}

					if isEditing {
						editPaint := graphics.NewPaint()
						editPaint.SetStyle(paintstyle.Stroke)
						editPaint.SetStrokeWidth(2)
						editPaint.SetColor(t.TableTheme.AccentInk)
						c.DrawRect(cellRect, editPaint)

						cursorPaint := graphics.NewPaint()
						cursorPaint.SetColor(t.TableTheme.AccentInk)
						editTextW := font.SimpleWidth(t.editText)
						cursorX := colX + 5 + iconOffset + editTextW
						cursorY1 := innerY + 2
						cursorY2 := innerY + t.RowHeight - 2
						c.DrawLine(cursorX, cursorY1, cursorX, cursorY2, cursorPaint)
					}
				}

				if isEditing && (cell.OnRender != nil || cell.IsCheck) {
					editPaint := graphics.NewPaint()
					editPaint.SetStyle(paintstyle.Stroke)
					editPaint.SetStrokeWidth(2)
					editPaint.SetColor(t.TableTheme.AccentInk)
					c.DrawRect(cellRect, editPaint)
				}

				colX += colWidths[colIdx]
			}

			innerY += t.RowHeight + t.IntercellSpacing
		}

		if t.dragging && t.dropCol >= 0 {
			dropX := r.X
			for i := 0; i < t.dropCol && i < len(colWidths); i++ {
				dropX += colWidths[i]
			}
			dropPaint := graphics.NewPaint()
			dropPaint.SetColor(t.TableTheme.AccentInk)
			dropPaint.SetStyle(paintstyle.Stroke)
			dropPaint.SetStrokeWidth(2)
			c.DrawLine(dropX, r.Y, dropX, r.Y+r.Height, dropPaint)
		}
	})

	// Scrollbar via host: content extent includes the header strip.
	frame := t.FrameRect()
	totalHeight := float32(len(t.Rows))*t.RowHeight + t.HeaderHeight
	t.host.SetExtents(frame.Width, totalHeight, frame.Width, frame.Height)
	t.host.LayoutBars(rect.Width, rect.Height)
	_, t.ScrollOffset = t.host.Offsets()
	t.host.PaintBars(canvas)

	canvas.Restore()
}

func (t *Table) OnMouseDown(x, y float32, button int) bool {
	if !t.Enabled() {
		return false
	}

	frame := t.FrameRect()
	if t.host.HandleMouseDown(frame.Width, frame.Height, x, y, button) {
		return true
	}

	if button == 1 {
		t.showContextMenu(x, y)
		return true
	}

	colWidths := t.getColumnWidths(t.Width())

	if y < t.HeaderHeight {
		col, isEdge := t.columnEdgeAtX(x)
		if isEdge && col >= 0 && col < len(t.Columns) {
			t.resizingCol = col
			t.resizeStartX = x
			t.resizeStartWidth = t.colWidths[col]
			return true
		}

		col = t.columnAtX(x)
		if col >= 0 && col < len(t.Columns) {
			t.dragCol = col
			t.dragStartX = x
			t.dragging = false
			t.dropCol = -1
			return true
		}
		return false
	}

	adjustedY := y - t.HeaderHeight + t.ScrollOffset
	rowIdx := int(adjustedY / (t.RowHeight + t.IntercellSpacing))
	if rowIdx < 0 || rowIdx >= len(t.Rows) {
		return false
	}

	cx := float32(0)
	colIdx := 0
	for i, w := range colWidths {
		if x >= cx && x < cx+w {
			colIdx = i
			break
		}
		cx += w
	}

	// Window attachments are optional (headless use, tests, detached
	// popups): without one there is no click counter/modifier state.
	var clickCount int
	var mod platform.Modifiers
	if win := t.Window(); win != nil {
		clickCount = win.ClickCount()
		mod = win.Modifiers()
	}

	if clickCount >= 2 && t.RowSelection && rowIdx >= 0 && colIdx >= 0 {
		t.startEditing(rowIdx, colIdx)
		return true
	}

	if t.RowSelection && rowIdx >= 0 {
		if mod.ControlDown() {
			if t.SelectedRows[rowIdx] {
				delete(t.SelectedRows, rowIdx)
			} else {
				t.SelectedRows[rowIdx] = true
			}
			t.LastClickedRow = rowIdx
		} else if mod.ShiftDown() && t.LastClickedRow >= 0 {
			start, end := t.LastClickedRow, rowIdx
			if start > end {
				start, end = end, start
			}
			t.SelectedRows = make(map[int]bool)
			for r := start; r <= end; r++ {
				t.SelectedRows[r] = true
			}
			t.LastClickedRow = rowIdx
		} else {
			t.SelectedRows = map[int]bool{rowIdx: true}
			t.LastClickedRow = rowIdx
		}
	}

	if t.ColSelection {
		t.SelectedCol = colIdx
	}

	t.MarkForRedraw()
	if t.OnCellSelected != nil && rowIdx < len(t.Rows) && colIdx < len(t.Rows[rowIdx]) {
		t.OnCellSelected(rowIdx, colIdx, &t.Rows[rowIdx][colIdx])
	}
	return true
}

func (t *Table) OnMouseUp(x, y float32, button int) bool {
	// Always end scrollbar drag even if mouse is outside the scrollbar zone
	if t.VScroll.dragging {
		t.host.HandleMouseUp(x, y, button)
		return true
	}
	if t.resizingCol >= 0 {
		t.resizingCol = -1
		return true
	}
	if t.dragging {
		t.applyColumnReorder()
		t.dragging = false
		t.dragCol = -1
		t.dropCol = -1
		t.dragImage = nil
		meui.ClearDragOverlay()
		t.MarkForRedraw()
		return true
	}
	if t.dragCol >= 0 && y < t.HeaderHeight {
		// Single click on header — perform sort
		col := t.dragCol
		t.dragCol = -1
		t.dragStartX = 0

		if col == t.sortCol {
			if t.sortAsc {
				t.sortAsc = false
				t.sortRows()
			} else {
				t.sortCol = -1
				t.sortAsc = true
				t.unsortRows()
			}
		} else {
			t.sortCol = col
			t.sortAsc = true
			t.sortRows()
		}
		t.MarkForRedraw()
		return true
	}
	if t.dragCol >= 0 {
		t.dragCol = -1
		t.dragStartX = 0
	}
	return false
}

func (t *Table) OnMouseMove(x, y float32) {
	if !t.Enabled() {
		return
	}
	t.host.UpdateHover(x, y) // overlay pills reveal on strip hover

	// Route to scrollbar during drag
	if t.VScroll.dragging {
		t.host.HandleMouseMove(x, y)
		return
	}

	if t.resizingCol >= 0 {
		diff := x - t.resizeStartX
		newWidth := t.resizeStartWidth + diff
		if newWidth < 20 {
			newWidth = 20
		}
		t.colWidths[t.resizingCol] = newWidth
		t.cache.Invalidate()
		t.MarkForRedraw()
		return
	}

	if t.dragCol >= 0 {
		if !t.dragging {
			dx := x - t.dragStartX
			if dx < 0 {
				dx = -dx
			}
			if dx > 5 {
				t.dragging = true
				t.renderColumnDragImage()
				if t.dragImage != nil {
					// Position ghost at mouse cursor in window coords
					winX, winY := t.FromLocal(x, y)
					meui.SetDragOverlay(t.dragImage, winX-10, winY-4, t.dragImageW, t.dragImageH)
				}
			}
		}
		if t.dragging {
			t.dropCol = t.columnAtX(x)
			// Update ghost position to follow mouse
			if t.dragImage != nil {
				winX, winY := t.FromLocal(x, y)
				meui.SetDragOverlay(t.dragImage, winX-10, winY-4, t.dragImageW, t.dragImageH)
			}
			t.cache.Invalidate()
			t.MarkForRedraw()
		}
		return
	}

	if y < t.HeaderHeight {
		if t.HoveredRow != -1 {
			t.HoveredRow = -1
			t.cache.Invalidate()
			t.MarkForRedraw()
		}
		return
	}

	adjustedY := y - t.HeaderHeight + t.ScrollOffset
	rowIdx := int(adjustedY / (t.RowHeight + t.IntercellSpacing))
	oldHovered := t.HoveredRow
	t.HoveredRow = -1
	if rowIdx >= 0 && rowIdx < len(t.Rows) {
		t.HoveredRow = rowIdx
	}
	if oldHovered != t.HoveredRow {
		t.cache.Invalidate()
		t.MarkForRedraw()
	}
}

func (t *Table) OnCharacter(ch rune) {
	if !t.Enabled() || !t.editing {
		return
	}
	t.editText += string(ch)
	t.cache.Invalidate()
	t.MarkForRedraw()
}

// OnKeyRepeat is a key the user is HOLDING — Backspace over a word, an arrow along a row, a scrub.
// The window classifies an OS auto-repeat as a repeat rather than a fresh press (a widget without
// this method is asked once, deliberately), so this is what makes a held key work here.
func (t *Table) OnKeyRepeat(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	return t.OnKeyDown(keyCode, mod)
}

func (t *Table) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if !t.Enabled() || len(t.Rows) == 0 {
		return false
	}

	if t.editing {
		switch keyCode {
		case platform.KeyReturn:
			t.commitEdit()
			return true
		case platform.KeyEscape:
			t.editing = false
			t.cache.Invalidate()
			t.MarkForRedraw()
			return true
		case platform.KeyBackspace:
			if len(t.editText) > 0 {
				runes := []rune(t.editText)
				t.editText = string(runes[:len(runes)-1])
				t.cache.Invalidate()
				t.MarkForRedraw()
			}
			return true
		case platform.KeyDelete, platform.KeyLeft, platform.KeyRight, platform.KeyUp, platform.KeyDown:
			return true
		default:
			return false
		}
	}

	switch keyCode {
	case platform.KeyUp:
		if t.LastClickedRow > 0 {
			t.LastClickedRow--
			t.SelectedRows = map[int]bool{t.LastClickedRow: true}
			if t.OnCellSelected != nil && t.SelectedCol >= 0 && t.SelectedCol < len(t.Rows[t.LastClickedRow]) {
				t.OnCellSelected(t.LastClickedRow, t.SelectedCol, &t.Rows[t.LastClickedRow][t.SelectedCol])
			}
			t.MarkForRedraw()
		}
		return true
	case platform.KeyDown:
		if t.LastClickedRow < len(t.Rows)-1 {
			t.LastClickedRow++
			t.SelectedRows = map[int]bool{t.LastClickedRow: true}
			if t.OnCellSelected != nil && t.SelectedCol >= 0 && t.SelectedCol < len(t.Rows[t.LastClickedRow]) {
				t.OnCellSelected(t.LastClickedRow, t.SelectedCol, &t.Rows[t.LastClickedRow][t.SelectedCol])
			}
			t.MarkForRedraw()
		}
		return true
	case platform.KeyLeft:
		if t.SelectedCol > 0 {
			t.SelectedCol--
			if t.OnCellSelected != nil && t.LastClickedRow >= 0 && t.LastClickedRow < len(t.Rows) && t.SelectedCol < len(t.Rows[t.LastClickedRow]) {
				t.OnCellSelected(t.LastClickedRow, t.SelectedCol, &t.Rows[t.LastClickedRow][t.SelectedCol])
			}
			t.MarkForRedraw()
		}
		return true
	case platform.KeyRight:
		if t.SelectedCol < len(t.Columns)-1 {
			t.SelectedCol++
			if t.OnCellSelected != nil && t.LastClickedRow >= 0 && t.LastClickedRow < len(t.Rows) && t.SelectedCol < len(t.Rows[t.LastClickedRow]) {
				t.OnCellSelected(t.LastClickedRow, t.SelectedCol, &t.Rows[t.LastClickedRow][t.SelectedCol])
			}
			t.MarkForRedraw()
		}
		return true
	}
	return false
}

func (t *Table) OnMouseWheel(x, y float32, dx, dy float32) bool {
	if !t.Enabled() || dy == 0 {
		return false
	}
	frame := t.FrameRect()
	totalHeight := float32(len(t.Rows))*t.RowHeight + t.HeaderHeight
	t.host.SetExtents(frame.Width, totalHeight, frame.Width, frame.Height)
	if !t.host.HandleWheel(dx, dy) {
		return false
	}
	_, t.ScrollOffset = t.host.Offsets()
	t.cache.Invalidate()
	t.MarkForRedraw()
	return true
}

func (t *Table) Focusable() bool {
	return t.Enabled()
}

func (t *Table) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "columns":
		if arr, ok := value.([]interface{}); ok {
			cols := make([]string, len(arr))
			for i, v := range arr {
				cols[i] = fmt.Sprintf("%v", v)
			}
			t.SetColumns(cols)
		}
	case "rows":
		if arr, ok := value.([]interface{}); ok {
			var rows [][]TableCell
			for _, rv := range arr {
				if cells, ok := rv.([]interface{}); ok {
					var row []TableCell
					for _, cv := range cells {
						if cell, ok := cv.(map[string]interface{}); ok {
							tc := TableCell{}
							if text, ok := cell["text"].(string); ok {
								tc.Text = text
							}
							if isCheck, ok := cell["isCheck"].(bool); ok {
								tc.IsCheck = isCheck
							}
							if checked, ok := cell["checked"].(bool); ok {
								tc.Checked = checked
							}
							row = append(row, tc)
						}
					}
					rows = append(rows, row)
				}
			}
			t.SetRows(rows)
		}
	case "alternatingRows":
		if b, ok := value.(bool); ok {
			t.AlternatingRows = b
		}
	case "showGrid":
		if b, ok := value.(bool); ok {
			t.ShowGrid = b
			t.ShowHorizontalLines = b
			t.ShowVerticalLines = b
		}
	case "rowSelection":
		if b, ok := value.(bool); ok {
			t.RowSelection = b
		}
	case "colSelection":
		if b, ok := value.(bool); ok {
			t.ColSelection = b
		}
	case "sortable":
		// QML hint: sort is always available via header click
	case "resizable":
		// QML hint: resize is always available via header divider drag
	case "reorderable":
		// QML hint: reorder is always available via header drag
	case "multiSelect":
	case "editable":
	}
	t.cache.Invalidate()
}
