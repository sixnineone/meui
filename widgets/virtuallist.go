package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
)

// VirtualList is the web-scale list: a ScrollPane whose rows are built from a Go
// function as they enter the viewport, not up front.
//
// The web scenes' lists were QML children of a ScrollView — every row a widget — which is
// fine for twelve messages and is a thousand widgets for a thousand messages. In MEUI a
// row is a Panel with labels, so building them all is the expensive part, not painting.
//
//	vl := widgets.NewVirtualList()
//	vl.SetRowHeight(110)
//	vl.SetRowCount(len(messages))
//	vl.SetRowBuilder(func(i int) meui.Widget { return buildRow(messages[i]) })
//	vl.EnsureVisible(selected)
//
// Rows are created for the visible range plus an overscan and dropped when they leave it.
// There is deliberately no recycling of a row widget for a DIFFERENT index: a row's
// bindings (texts, colours, click closures) are built into the widgets, so rebinding would
// need a second lifecycle API every row widget had to implement. Build-on-demand + drop
// is the honest first step; a RowBinder is the extension point if a list ever needs
// heavy row widgets at scale.
type VirtualList struct {
	*ScrollPane

	stage *meui.Panel
	rowH  float32
	count int
	build func(index int) meui.Widget

	rows  map[int]meui.Widget
	first int
	last  int // exclusive

	// overscan keeps a few rows beyond the viewport built, so a wheel step reveals rows
	// instead of building them mid-frame.
	overscan int
	// topInset is the part of the viewport covered by a sticky header; EnsureVisible and
	// the visible range respect it.
	topInset float32
}

// NewVirtualList builds an empty, frameless list that hit-tests its own rows.
func NewVirtualList() *VirtualList {
	vl := &VirtualList{
		ScrollPane: NewScrollPane(),
		rows:       make(map[int]meui.Widget),
		overscan:   2,
	}
	vl.stage = meui.NewPanel()
	vl.ScrollPane.SetFrameless(true)
	vl.ScrollPane.SetContent(vl.stage)
	vl.ScrollPane.SetHitTestContent(true)
	return vl
}

// SetRowHeight fixes every row's height. It is one number, not per-row heights: a
// variable-height list needs a layout pass over the data to know the scroll extent, which
// is a different widget.
func (vl *VirtualList) SetRowHeight(h float32) *VirtualList {
	if h < 0 {
		h = 0
	}
	vl.rowH = h
	vl.reset()
	vl.MarkForRedraw()
	return vl
}

// RowHeight reports the fixed row height.
func (vl *VirtualList) RowHeight() float32 { return vl.rowH }

// SetRowCount sets how many rows the data has and rebuilds the visible window.
func (vl *VirtualList) SetRowCount(n int) *VirtualList {
	if n < 0 {
		n = 0
	}
	vl.count = n
	vl.reset()
	vl.MarkForRedraw()
	return vl
}

// RowCount reports the data size.
func (vl *VirtualList) RowCount() int { return vl.count }

// SetRowBuilder installs the function that builds one row's widget. It is called for
// indices as they enter the window, and may return nil to leave a row empty.
func (vl *VirtualList) SetRowBuilder(fn func(index int) meui.Widget) *VirtualList {
	vl.build = fn
	vl.reset()
	vl.MarkForRedraw()
	return vl
}

// SetTopInset tells the list that a sticky header covers the top of the viewport.
func (vl *VirtualList) SetTopInset(px float32) *VirtualList {
	if px < 0 {
		px = 0
	}
	vl.topInset = px
	return vl
}

// TopInset reports the sticky header height the list lays its rows below.
func (vl *VirtualList) TopInset() float32 { return vl.topInset }

// Refresh rebuilds the visible window against the current frame and scroll offset. Paint
// calls it too, so an app only needs it when it wants the rows to exist before the next
// frame (a test, or a click that arrives before the first paint).
func (vl *VirtualList) Refresh() { vl.sync(vl.FrameRect().Width) }

// VisibleRange reports the half-open index range the viewport shows.
func (vl *VirtualList) VisibleRange() (int, int) {
	off := vl.ScrollY()
	viewH := vl.FrameRect().Height
	if vl.rowH <= 0 {
		return 0, 0
	}
	first := int((off + vl.topInset) / vl.rowH)
	last := int((off+viewH)/vl.rowH) + 1
	if first < 0 {
		first = 0
	}
	if last > vl.count {
		last = vl.count
	}
	return first, last
}

// RowWidget returns the built widget for an index, or nil when it is outside the window.
func (vl *VirtualList) RowWidget(index int) meui.Widget { return vl.rows[index] }

// WalkBuilt calls fn for every row widget currently built, in index order. It is how a
// selection change restyles the rows that exist without rebuilding them.
func (vl *VirtualList) WalkBuilt(fn func(index int, w meui.Widget)) {
	for i := vl.first; i < vl.last; i++ {
		if w, ok := vl.rows[i]; ok {
			fn(i, w)
		}
	}
}

// ScrollToIndex scrolls so row index is at the top of the viewport.
func (vl *VirtualList) ScrollToIndex(index int) *VirtualList {
	if index < 0 {
		index = 0
	}
	if index >= vl.count {
		index = vl.count - 1
	}
	if index < 0 {
		return vl
	}
	vl.ScrollTo(vl.topInset + float32(index)*vl.rowH)
	return vl
}

// EnsureVisible scrolls just enough that row index is fully inside the viewport, below a
// sticky header when one is declared. It is the reveal a click needs.
func (vl *VirtualList) EnsureVisible(index int) *VirtualList {
	if vl.rowH <= 0 || index < 0 || index >= vl.count {
		return vl
	}
	top := vl.topInset + float32(index)*vl.rowH
	viewH := vl.FrameRect().Height
	off := vl.ScrollY()
	switch {
	case top < off+vl.topInset:
		vl.ScrollTo(top - vl.topInset)
	case top+vl.rowH > off+viewH:
		vl.ScrollTo(top + vl.rowH - viewH)
	}
	return vl
}

// Paint syncs the window first: scrolling changes which rows must exist, and this is the
// one call that happens on every frame after the frame and offsets are settled.
func (vl *VirtualList) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	vl.sync(rect.Width)
	vl.ScrollPane.Paint(canvas, rect)
}

// reset drops every built row, so the next sync rebuilds from scratch (a data change).
func (vl *VirtualList) reset() {
	for i, w := range vl.rows {
		vl.stage.RemoveChild(w)
		delete(vl.rows, i)
	}
	vl.first, vl.last = 0, 0
}

// sync brings the built rows in line with the scroll offset and the data size.
func (vl *VirtualList) sync(viewW float32) {
	if vl.build == nil || vl.count <= 0 || vl.rowH <= 0 {
		if len(vl.rows) > 0 {
			vl.reset()
		}
		return
	}
	// The inset is the sticky header's height: the first row starts BELOW it, and later
	// rows scroll under it.
	contentH := vl.topInset + float32(vl.count)*vl.rowH
	vl.ScrollPane.SetContentSize(0, contentH)
	viewH := vl.FrameRect().Height
	vl.stage.SetFrameRect(graphics.NewRect(0, 0, viewW, contentH))

	off := vl.ScrollY() - vl.topInset
	if off < 0 {
		off = 0
	}
	first := int(off/vl.rowH) - vl.overscan
	last := int((off+viewH)/vl.rowH) + 1 + vl.overscan
	if first < 0 {
		first = 0
	}
	if last > vl.count {
		last = vl.count
	}
	if last < first {
		last = first
	}
	if first == vl.first && last == vl.last && len(vl.rows) == last-first {
		return
	}

	for i, w := range vl.rows {
		if i < first || i >= last {
			vl.stage.RemoveChild(w)
			delete(vl.rows, i)
		}
	}
	for i := first; i < last; i++ {
		frame := graphics.NewRect(0, vl.topInset+float32(i)*vl.rowH, viewW, vl.rowH)
		if w, ok := vl.rows[i]; ok {
			w.SetFrameRect(frame)
			continue
		}
		w := vl.build(i)
		if w == nil {
			continue
		}
		w.SetFrameRect(frame)
		vl.stage.AddChild(w)
		vl.rows[i] = w
	}
	vl.first, vl.last = first, last
}

// SetBoundProperty carries the QML form:
//
//	VirtualList { rowHeight: 110; rowCount: 240; topInset: 40 }
//
// The DATA and the row builder always come from Go — QML here is a builder, so a row
// template is a Go function, not a delegate.
func (vl *VirtualList) SetBoundProperty(name string, value interface{}) {
	switch name {
	case "rowHeight":
		if f, ok := asFloat(value); ok {
			vl.SetRowHeight(f)
		}
	case "rowCount":
		if f, ok := asFloat(value); ok {
			vl.SetRowCount(int(f))
		}
	case "topInset":
		if f, ok := asFloat(value); ok {
			vl.SetTopInset(f)
		}
	}
}
