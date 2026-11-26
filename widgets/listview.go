package widgets

import (
	"fmt"
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
	"os"
)

type ListView struct {
	meui.BaseWidget
	*theme.ListViewTheme
	Items            []string
	ItemHeight       float32
	SelectedIndex    int
	ScrollOffset     float32
	host             *ScrollbarHost
	CheckableItems   bool
	ItemChecked      map[int]bool
	OnItemSelected   func(int, string)
	OnItemsReordered func([]string)
	dropTargetIdx    int
	sourceIdx        int
	HoveredIndex     int
	cursorLine       int
	MultiSelect      bool
	AlternatingRows  bool
	// HideBorder drops the widget's own outline (e.g. a ListView embedded
	// in a ScrollPane: its edge line next to the pane's scrollbar lane
	// read as a second, extra-wide lane).
	HideBorder      bool
	selectedIndices map[int]bool
	dragSourceIdx   int
	showDropMarker  bool
	OnCheckChanged  func(int, string, bool)
	shiftAnchor     int
	ctxMenu         *ContextMenu
	VScroll         *ScrollBar
	itemTexts       []*graphics.Text // cached *graphics.Text per item (for emoji fallback)
	textCacheVer    uint64           // theme.ThemeVersion() at last itemTexts build
}

// listRowHeight reads the active theme's row metric (30px fallback for
// themes without metrics, e.g. IntelliJ imports).
func listRowHeight() float32 {
	if t := theme.DefaultTheme; t != nil && t.Metrics.ListRowHeight > 0 {
		return t.Metrics.ListRowHeight
	}
	return 30
}

func NewListView() *ListView {
	l := &ListView{
		BaseWidget:      *meui.NewBaseWidget(),
		ListViewTheme:   theme.DefaultListViewTheme,
		ItemHeight:      listRowHeight(),
		SelectedIndex:   -1,
		HoveredIndex:    -1,
		selectedIndices: make(map[int]bool),
		ItemChecked:     make(map[int]bool),
	}
	l.host = NewScrollbarHost()
	l.host.ShowH = false
	l.VScroll = l.host.VScroll
	l.VScroll.SetParent(l)
	l.host.SetOnChange(func() {
		_, offY := l.host.Offsets()
		l.ScrollOffset = offY
		l.MarkForRedraw()
	})
	l.SetTabIndex(0)
	return l
}

func (l *ListView) AddItem(item string) *ListView {
	l.Items = append(l.Items, item)
	l.MarkForRedraw()
	return l
}

// RemoveItem removes the item at index, fixing up selections (like QListWidget.takeItem).
func (l *ListView) RemoveItem(index int) *ListView {
	if index < 0 || index >= len(l.Items) {
		return l
	}
	l.Items = append(l.Items[:index], l.Items[index+1:]...)
	if l.MultiSelect {
		delete(l.selectedIndices, index)
		// shift remaining selections down
		shifted := make(map[int]bool, len(l.selectedIndices))
		for i := range l.selectedIndices {
			if i > index {
				shifted[i-1] = true
			} else {
				shifted[i] = true
			}
		}
		l.selectedIndices = shifted
	} else {
		if l.SelectedIndex == index {
			l.SelectedIndex = -1
		} else if l.SelectedIndex > index {
			l.SelectedIndex--
		}
	}
	if l.ItemChecked != nil {
		updated := make(map[int]bool, len(l.ItemChecked))
		for i, checked := range l.ItemChecked {
			if i == index {
				continue
			}
			if i > index {
				updated[i-1] = checked
			} else {
				updated[i] = checked
			}
		}
		l.ItemChecked = updated
	}
	l.invalidateTextCache()
	l.MarkForRedraw()
	return l
}

// ClearItems removes all items and selections (like QListWidget.clear).
func (l *ListView) ClearItems() *ListView {
	l.Items = nil
	l.SelectedIndex = -1
	l.selectedIndices = nil
	l.ItemChecked = nil
	l.ScrollOffset = 0
	l.invalidateTextCache()
	l.MarkForRedraw()
	return l
}

// ItemCount returns the number of items.
func (l *ListView) ItemCount() int { return len(l.Items) }

// GetItem returns the item at index, or "".
func (l *ListView) GetItem(index int) string {
	if index >= 0 && index < len(l.Items) {
		return l.Items[index]
	}
	return ""
}

// GetSelectedItem returns the first selected item text, or "".
func (l *ListView) GetSelectedItem() string {
	if l.MultiSelect {
		best := -1
		for i := range l.selectedIndices {
			if best == -1 || i < best {
				best = i
			}
		}
		if best == -1 {
			return ""
		}
		return l.GetItem(best)
	}
	return l.GetItem(l.SelectedIndex)
}

// ScrollToIndex scrolls the list so the item is visible.
func (l *ListView) ScrollToIndex(index int) {
	if index < 0 || index >= len(l.Items) {
		return
	}
	itemH := l.ItemHeight
	if itemH <= 0 {
		itemH = 24
	}
	itemY := float32(index) * itemH
	if frame := l.FrameRect(); frame.Height > 0 {
		totalHeight := float32(len(l.Items)) * l.ItemHeight
		l.host.SetExtents(frame.Width, totalHeight, frame.Width, frame.Height)
		offX, offY := l.host.Offsets()
		if itemY < offY {
			offY = itemY
		} else if itemY+itemH > offY+frame.Height {
			offY = itemY + itemH - frame.Height
		}
		l.host.SetOffsets(offX, offY)
		_, l.ScrollOffset = l.host.Offsets()
	}
	l.MarkForRedraw()
}

func (l *ListView) visibleItemRange(r graphics.Rect) (start, end int) {
	totalItems := len(l.Items)
	if totalItems == 0 {
		return 0, 0
	}
	topY := r.Y - l.ScrollOffset
	firstItem := int((r.Y - topY) / l.ItemHeight)
	if firstItem < 0 {
		firstItem = 0
	}
	lastItem := int((r.Y + r.Height - topY + l.ItemHeight - 1) / l.ItemHeight)
	if lastItem > totalItems {
		lastItem = totalItems
	}
	return firstItem, lastItem
}

func (l *ListView) maxScrollOffset() float32 {
	totalHeight := float32(len(l.Items)) * l.ItemHeight
	frame := l.FrameRect()
	maxScroll := totalHeight - frame.Height
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}

func (l *ListView) scrollOffsetToItem(index int) {
	frame := l.FrameRect()
	totalHeight := float32(len(l.Items)) * l.ItemHeight
	l.host.SetExtents(frame.Width, totalHeight, frame.Width, frame.Height)
	itemTop := float32(index) * l.ItemHeight
	itemBottom := itemTop + l.ItemHeight
	offX, offY := l.host.Offsets()
	if itemTop < offY {
		offY = itemTop
	} else if itemBottom > offY+frame.Height {
		offY = itemBottom - frame.Height
	}
	l.host.SetOffsets(offX, offY)
	_, l.ScrollOffset = l.host.Offsets()
}

func (l *ListView) SetItems(items []string) *ListView {
	l.Items = items
	l.itemTexts = nil
	l.MarkForRedraw()
	return l
}

func (l *ListView) invalidateTextCache() { l.itemTexts = nil }

func (l *ListView) rebuildTextCache(font graphics.Font) {
	if len(l.Items) == 0 {
		l.itemTexts = nil
		l.textCacheVer = theme.ThemeVersion()
		return
	}
	l.itemTexts = make([]*graphics.Text, len(l.Items))
	l.textCacheVer = theme.ThemeVersion()
	dec := &graphics.TextDecoration{Font: font, OnBackgroundInk: l.ListViewTheme.TextInk}
	for i, item := range l.Items {
		l.itemTexts[i] = graphics.NewText(item, dec)
	}
}

func (l *ListView) SetSelectedIndex(index int) *ListView {
	l.SelectedIndex = index
	l.MarkForRedraw()
	return l
}

func (l *ListView) GetSelectedIndices() []int {
	if !l.MultiSelect {
		if l.SelectedIndex >= 0 {
			return []int{l.SelectedIndex}
		}
		return nil
	}
	indices := make([]int, 0, len(l.selectedIndices))
	for i := range l.selectedIndices {
		indices = append(indices, i)
	}
	return indices
}

func (l *ListView) ClearSelection() *ListView {
	l.SelectedIndex = -1
	l.selectedIndices = make(map[int]bool)
	l.MarkForRedraw()
	return l
}

func (l *ListView) SetOnItemSelected(handler func(int, string)) *ListView {
	l.OnItemSelected = handler
	return l
}

func (l *ListView) SetOnItemsReordered(handler func([]string)) *ListView {
	l.OnItemsReordered = handler
	return l
}

// ── Drag-and-drop (item reorder) ────────────────────────────────

func (l *ListView) OnDragStart(x, y float32) *meui.DragData {
	// Don't start drag if scrollbar is being dragged
	if l.VScroll.dragging {
		return nil
	}
	// The drag manager delivers WINDOW coordinates — convert to local.
	if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
		fmt.Printf("[listDrag] OnDragStart at (%.0f,%.0f)\n", x, y)
	}
	local := l.PointFromRoot(x, y)
	y = local.Y
	adjustedY := y + l.ScrollOffset
	index := int(adjustedY / l.ItemHeight)
	if index < 0 || index >= len(l.Items) {
		return nil
	}
	l.dragSourceIdx = index
	l.dropTargetIdx = -1
	l.showDropMarker = false
	return &meui.DragData{
		Type:  "listview-item",
		Index: index,
		Data:  l.Items[index],
	}
}

func (l *ListView) OnDragEnter(data *meui.DragData, x, y float32) {
	l.showDropMarker = true
	local := l.PointFromRoot(x, y)
	l.calcDropTarget(local.X, local.Y)
	l.MarkForRedraw()
}

func (l *ListView) OnDragOver(data *meui.DragData, x, y float32) {
	local := l.PointFromRoot(x, y)
	l.calcDropTarget(local.X, local.Y)
	l.MarkForRedraw()
}

func (l *ListView) OnDragDrop(data *meui.DragData, x, y float32) {
	if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
		fmt.Printf("[listDrop] at (%.0f,%.0f) src=%d dst=%d\n", x, y, data.Index, l.dropTargetIdx)
	}
	if data.Type != "listview-item" || data.Source != l {
		return
	}
	srcIdx := data.Index
	dstIdx := l.dropTargetIdx
	if dstIdx < 0 || dstIdx >= len(l.Items) {
		dstIdx = len(l.Items) - 1
	}
	if srcIdx == dstIdx {
		l.showDropMarker = false
		l.MarkForRedraw()
		return
	}
	// Reorder
	item := l.Items[srcIdx]
	newItems := make([]string, 0, len(l.Items))
	for i, v := range l.Items {
		if i == srcIdx {
			continue
		}
		if i == dstIdx {
			if srcIdx < dstIdx {
				newItems = append(newItems, v)
				newItems = append(newItems, item)
			} else {
				newItems = append(newItems, item)
				newItems = append(newItems, v)
			}
		} else {
			newItems = append(newItems, v)
		}
	}
	if dstIdx >= len(newItems) {
		newItems = append(newItems, item)
	}
	l.Items = newItems
	l.selectedIndices = make(map[int]bool)
	l.showDropMarker = false
	// The per-index text cache must be dropped — the length didn't change,
	// so the paint's length check wouldn't rebuild it and the old order
	// would keep rendering.
	l.invalidateTextCache()
	l.MarkForRedraw()
	if l.OnItemsReordered != nil {
		l.OnItemsReordered(newItems)
	}
}

func (l *ListView) OnDragLeave(data *meui.DragData) {
	l.showDropMarker = false
	l.dropTargetIdx = -1
	l.MarkForRedraw()
}

func (l *ListView) calcDropTarget(x, y float32) {
	adjustedY := y + l.ScrollOffset
	idx := int(adjustedY / l.ItemHeight)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(l.Items) {
		idx = len(l.Items) - 1
	}
	l.dropTargetIdx = idx
}

func (l *ListView) SetOnCheckChanged(handler func(int, string, bool)) *ListView {
	l.OnCheckChanged = handler
	return l
}

func (l *ListView) SetItemChecked(index int, checked bool) *ListView {
	l.ItemChecked[index] = checked
	l.MarkForRedraw()
	return l
}

func (l *ListView) SetAlternatingRows(enabled bool) *ListView {
	l.AlternatingRows = enabled
	l.MarkForRedraw()
	return l
}

func (l *ListView) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !l.Visible() || len(l.Items) == 0 {
		return
	}

	canvas.Save()

	disabled := !l.Enabled()

	paint := graphics.NewPaint()
	if disabled {
		paint.SetColor(theme.DefaultTheme.DisabledBg)
	} else {
		paint.SetColor(l.ListViewTheme.BackgroundInk)
	}
	canvas.DrawRoundedRect(rect, l.ListViewTheme.CornerRadius, l.ListViewTheme.CornerRadius, paint)

	if !l.HideBorder {
		borderPaint := graphics.NewPaint()
		borderPaint.SetStyle(paintstyle.Stroke)
		borderPaint.SetStrokeWidth(theme.BorderStrokeWidth)
		if disabled {
			borderPaint.SetColor(theme.DefaultTheme.DisabledBorder)
		} else {
			borderPaint.SetColor(l.ListViewTheme.EdgeInk)
		}
		canvas.DrawRoundedRect(rect, l.ListViewTheme.CornerRadius, l.ListViewTheme.CornerRadius, borderPaint)
	}

	font := l.ListViewTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}

	// Determine if scrollbar is needed
	frame := l.FrameRect()
	totalHeight := float32(len(l.Items)) * l.ItemHeight
	maxScroll := totalHeight - frame.Height
	hasScrollbar := maxScroll > 0

	// Items are shorter when scrollbar is visible
	itemW := rect.Width
	if hasScrollbar {
		itemW -= l.host.BarSize
	}

	// Clip to viewport bounds (hides items above/below when scrolled)
	canvas.Save()
	canvas.ClipRect(rect, pathop.Intersect, false)

	// Item Y starts at the top minus scroll offset
	innerY := rect.Y - l.ScrollOffset

	// Only render items in the visible range (perf for large lists)
	startItem, endItem := l.visibleItemRange(rect)
	for i := startItem; i < endItem; i++ {
		item := l.Items[i]
		itemRect := graphics.NewRect(rect.X, innerY+float32(i)*l.ItemHeight, itemW, l.ItemHeight)

		isSelected := i == l.SelectedIndex
		if l.MultiSelect && l.selectedIndices[i] {
			isSelected = true
		}

		rowR := theme.Shape().ListRow
		if !disabled && isSelected {
			selectedPaint := graphics.NewPaint()
			selectedPaint.SetColor(l.ListViewTheme.SelectedInk)
			canvas.DrawRoundedRect(itemRect, rowR, rowR, selectedPaint)
		} else if !disabled && i == l.HoveredIndex {
			hoverPaint := graphics.NewPaint()
			hoverPaint.SetColor(l.ListViewTheme.HoverInk)
			canvas.DrawRoundedRect(itemRect, rowR, rowR, hoverPaint)
		} else if l.AlternatingRows && i%2 == 1 {
			altPaint := graphics.NewPaint()
			altPaint.SetColor(l.ListViewTheme.AlternateRowInk)
			canvas.DrawRect(itemRect, altPaint)
		}

		textX := rect.X + 5

		// Checkbox
		if l.CheckableItems {
			boxSize := float32(14)
			boxY := itemRect.Y + (l.ItemHeight-boxSize)/2
			DrawCheckBox(canvas, textX, boxY, boxSize, l.ItemChecked[i], !disabled)
			textX += boxSize + 6
		}

		textPaint := graphics.NewPaint()
		switch {
		case disabled:
			textPaint.SetColor(theme.DefaultTheme.DisabledText)
		case isSelected && l.ListViewTheme.SelectedInk>>24 == 0xFF:
			// Solid selection: on-accent text. Translucent selection
			// tints (Fluent) keep the normal text colour.
			textPaint.SetColor(theme.DefaultTheme.OnAccentColor())
		default:
			textPaint.SetColor(l.ListViewTheme.TextInk)
		}
		ascent := font.Baseline()
		fontHeight := font.LineHeight()
		textY := itemRect.Y + (itemRect.Height-fontHeight)/2 + ascent
		if l.itemTexts == nil || len(l.itemTexts) != len(l.Items) ||
			l.textCacheVer != theme.ThemeVersion() {
			l.rebuildTextCache(font)
		}
		if l.itemTexts != nil && i < len(l.itemTexts) {
			l.itemTexts[i].Draw(canvas, textX, textY)
		} else {
			it := graphics.NewText(item, &graphics.TextDecoration{Font: font, OnBackgroundInk: textPaint.Color()})
			it.Draw(canvas, textX, textY)
		}
	}

	// Draw drop marker
	if l.showDropMarker && l.dropTargetIdx >= 0 {
		markerY := rect.Y + float32(l.dropTargetIdx)*l.ItemHeight + l.ItemHeight - l.ScrollOffset - 2
		markerPaint := graphics.NewPaint()
		markerPaint.SetColor(graphics.Color(0xFF4FC3F7))
		markerPaint.SetStrokeWidth(2)
		canvas.DrawLine(rect.X+4, markerY, rect.X+itemW-4, markerY, markerPaint)
	}

	// Restore viewport clip before drawing scrollbar
	canvas.Restore()

	if hasScrollbar {
		l.host.SetExtents(frame.Width, totalHeight, frame.Width, frame.Height)
		l.host.LayoutBars(rect.Width, rect.Height)
		_, l.ScrollOffset = l.host.Offsets()
		l.host.PaintBars(canvas)
	}

	canvas.Restore()
}

func (l *ListView) OnMouseDown(x, y float32, button int) bool {
	if !l.Enabled() {
		return false
	}
	// Check scrollbar hit (local frame: x/y below are widget-local)
	frame := l.FrameRect()
	if l.host.HandleMouseDown(frame.Width, frame.Height, x, y, button) {
		return true
	}
	adjustedY := y + l.ScrollOffset
	index := int(adjustedY / l.ItemHeight)
	if index >= 0 && index < len(l.Items) {
		// Checkbox click (leftmost 20px)
		if l.CheckableItems && x >= 0 && x < 20 {
			checked := !l.ItemChecked[index]
			l.ItemChecked[index] = checked
			l.MarkForRedraw()
			if l.OnCheckChanged != nil {
				l.OnCheckChanged(index, l.Items[index], checked)
			}
			return true
		}

		if l.MultiSelect {
			mod := l.Modifiers()
			if mod.ControlDown() {
				if l.selectedIndices[index] {
					delete(l.selectedIndices, index)
				} else {
					l.selectedIndices[index] = true
					l.shiftAnchor = index
				}
			} else if mod.ShiftDown() {
				start := l.shiftAnchor
				end := index
				if start > end {
					start, end = end, start
				}
				l.selectedIndices = make(map[int]bool)
				for i := start; i <= end; i++ {
					l.selectedIndices[i] = true
				}
			} else {
				l.selectedIndices = make(map[int]bool)
				l.selectedIndices[index] = true
				l.shiftAnchor = index
			}
			l.SelectedIndex = index
		} else {
			l.SelectedIndex = index
		}
		l.MarkForRedraw()
		if l.OnItemSelected != nil {
			l.OnItemSelected(index, l.Items[index])
		}
		return true
	}
	return false
}

func (l *ListView) OnMouseMove(x, y float32) {
	if !l.Enabled() {
		return
	}
	l.host.UpdateHover(x, y) // overlay pills reveal on strip hover
	// Route to scrollbar during drag
	if l.VScroll.dragging {
		l.host.HandleMouseMove(x, y)
		return
	}
	// Ignore hover in scrollbar zone
	fw := l.FrameRect().Width
	if l.host.VisibleV() && x >= fw-l.host.BarSize {
		if l.HoveredIndex != -1 {
			l.HoveredIndex = -1
			l.MarkForRedraw()
		}
		return
	}
	adjustedY := y + l.ScrollOffset
	index := int(adjustedY / l.ItemHeight)
	oldHovered := l.HoveredIndex
	l.HoveredIndex = -1
	if index >= 0 && index < len(l.Items) {
		l.HoveredIndex = index
	}
	if oldHovered != l.HoveredIndex {
		l.MarkForRedraw()
	}
}

func (l *ListView) OnMouseUp(x, y float32, button int) bool {
	// Always end scrollbar drag even if mouse is outside the scrollbar zone
	if l.VScroll.dragging {
		l.host.HandleMouseUp(x, y, button)
		return true
	}
	return false
}

func (l *ListView) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "currentIndex":
		if f, ok := value.(float64); ok {
			l.SetSelectedIndex(int(f))
		}
	case "model":
		if arr, ok := value.([]interface{}); ok {
			items := make([]string, len(arr))
			for i, v := range arr {
				items[i] = fmt.Sprintf("%v", v)
			}
			l.SetItems(items)
		}
	case "alternatingRows", "alternatingRowColors":
		if b, ok := value.(bool); ok {
			l.SetAlternatingRows(b)
		}
	case "multiSelect", "multiSelectionEnabled":
		if b, ok := value.(bool); ok {
			l.MultiSelect = b
		}
	case "checkableItems", "checkable":
		if b, ok := value.(bool); ok {
			l.CheckableItems = b
		}
	case "hideBorder":
		if b, ok := value.(bool); ok {
			l.HideBorder = b
			l.MarkForRedraw()
		}
	case "border.width", "borderWidth":
		// Qt-style toggle: width 0 removes the outline.
		if f, ok := asFloat(value); ok {
			l.HideBorder = f <= 0
			l.MarkForRedraw()
		}
	}
}

func (l *ListView) OnMouseWheel(x, y float32, dx, dy float32) bool {
	if !l.Enabled() || dy == 0 {
		return false
	}
	frame := l.FrameRect()
	totalHeight := float32(len(l.Items)) * l.ItemHeight
	l.host.SetExtents(frame.Width, totalHeight, frame.Width, frame.Height)
	if !l.host.HandleWheel(dx, dy) {
		return false
	}
	_, l.ScrollOffset = l.host.Offsets()
	l.MarkForRedraw()
	return true
}

// OnKeyRepeat is a key the user is HOLDING — Backspace over a word, an arrow along a row, a scrub.
// The window classifies an OS auto-repeat as a repeat rather than a fresh press (a widget without
// this method is asked once, deliberately), so this is what makes a held key work here.
func (l *ListView) OnKeyRepeat(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	return l.OnKeyDown(keyCode, mod)
}

func (l *ListView) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if !l.Enabled() || len(l.Items) == 0 {
		return false
	}
	switch keyCode {
	case platform.KeyUp:
		if l.SelectedIndex > 0 {
			l.SelectedIndex--
			if l.MultiSelect && mod.ShiftDown() {
				start := l.shiftAnchor
				end := l.SelectedIndex
				if start > end {
					start, end = end, start
				}
				for i := start; i <= end; i++ {
					l.selectedIndices[i] = true
				}
			}
			l.scrollOffsetToItem(l.SelectedIndex)
			if l.OnItemSelected != nil {
				l.OnItemSelected(l.SelectedIndex, l.Items[l.SelectedIndex])
			}
			l.MarkForRedraw()
		}
		return true
	case platform.KeyDown:
		if l.SelectedIndex < len(l.Items)-1 {
			l.SelectedIndex++
			if l.MultiSelect && mod.ShiftDown() {
				start := l.shiftAnchor
				end := l.SelectedIndex
				if start > end {
					start, end = end, start
				}
				for i := start; i <= end; i++ {
					l.selectedIndices[i] = true
				}
			}
			l.scrollOffsetToItem(l.SelectedIndex)
			if l.OnItemSelected != nil {
				l.OnItemSelected(l.SelectedIndex, l.Items[l.SelectedIndex])
			}
			l.MarkForRedraw()
		}
		return true
	case platform.KeyHome:
		if l.SelectedIndex != 0 {
			l.SelectedIndex = 0
			l.ScrollOffset = 0
			if l.OnItemSelected != nil {
				l.OnItemSelected(l.SelectedIndex, l.Items[l.SelectedIndex])
			}
			l.MarkForRedraw()
		}
		return true
	case platform.KeyEnd:
		last := len(l.Items) - 1
		if l.SelectedIndex != last {
			l.SelectedIndex = last
			l.ScrollOffset = l.maxScrollOffset()
			if l.OnItemSelected != nil {
				l.OnItemSelected(l.SelectedIndex, l.Items[l.SelectedIndex])
			}
			l.MarkForRedraw()
		}
		return true
	case platform.KeyA:
		if l.MultiSelect && mod.ControlDown() {
			l.selectedIndices = make(map[int]bool)
			for i := range l.Items {
				l.selectedIndices[i] = true
			}
			l.MarkForRedraw()
			return true
		}
	}
	return false
}

func (l *ListView) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	fr := l.FrameRect()
	if fr.Width > 0 || fr.Height > 0 {
		return fr.Width, fr.Height
	}
	return 200, 120
}

func (l *ListView) MinimumDimensions() (float32, float32) {
	return 120, 80
}

func (l *ListView) Focusable() bool {
	return l.Enabled()
}
