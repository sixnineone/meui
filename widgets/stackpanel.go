package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

const (
	splitterH    = 6
	stackMinSize = 50
	splitterHotW = 6
)

// StackPanel stacks widgets vertically or horizontally with draggable splitters.
type StackPanel struct {
	meui.BaseContainer
	Orientation    string // "vertical" or "horizontal"
	Items          []meui.Widget
	Sizes          []float32 // allocated size of each item (width or height)
	StackTheme     *theme.StackPanelTheme
	dragging       bool
	dragIdx        int // index of the splitter being dragged (between Items[i] and Items[i+1])
	hovering       bool
	hoverIdx       int  // index of the splitter the pointer is on (-1 while none)
	hoverAlways    bool // explicit override of the theme's HoverColorAlwaysOn
	hoverAlwaysSet bool
	dragStart      float32
	dragOrigA      float32 // size of item[i] at drag start
	dragOrigB      float32 // size of item[i+1] at drag start
}

func NewStackPanel(orientation string) *StackPanel {
	return &StackPanel{
		BaseContainer: meui.BaseContainer{BaseWidget: *meui.NewBaseWidget()},
		Orientation:   orientation,
		StackTheme:    theme.DefaultStackPanelTheme,
	}
}

// SetFrameRect overrides BaseWidget.SetFrameRect to trigger layout redistribution.
func (sp *StackPanel) SetFrameRect(rect graphics.Rect) {
	sp.BaseWidget.SetFrameRect(rect)
	sp.SetNeedsLayout()
}

// PanelAt returns the deepest child widget containing the point (in local coordinates).
func (sp *StackPanel) PanelAt(x, y float32) meui.Widget {
	if !sp.Visible() {
		return nil
	}
	yOff := float32(0)
	xOff := float32(0)
	for i, item := range sp.Items {
		if item == nil {
			continue
		}
		sz := sp.Sizes[i]
		if sz <= 0 {
			sz = 1
		}
		var itemRect graphics.Rect
		if sp.Orientation == "vertical" {
			itemRect = graphics.NewRect(0, yOff, sp.FrameRect().Width, sz)
			yOff += sz + splitterH
		} else {
			itemRect = graphics.NewRect(xOff, 0, sz, sp.FrameRect().Height)
			xOff += sz + splitterH
		}
		if x >= itemRect.X && x <= itemRect.X+itemRect.Width && y >= itemRect.Y && y <= itemRect.Y+itemRect.Height {
			localX, localY := x-itemRect.X, y-itemRect.Y
			if container, ok := item.(meui.Container); ok {
				if deeper := container.PanelAt(localX, localY); deeper != nil {
					return deeper
				}
			}
			return item
		}
	}
	return sp
}

// AddItem adds a widget to the stack with the given size.
func (sp *StackPanel) AddItem(w meui.Widget, size float32) {
	sp.Items = append(sp.Items, w)
	sp.Sizes = append(sp.Sizes, size)
	w.SetParent(sp)
	sp.SetNeedsLayout()
	sp.MarkForRedraw()
}

// RemoveItem removes the widget at index. Returns the removed widget.
func (sp *StackPanel) RemoveItem(idx int) meui.Widget {
	if idx < 0 || idx >= len(sp.Items) {
		return nil
	}
	w := sp.Items[idx]
	sp.Items = append(sp.Items[:idx], sp.Items[idx+1:]...)
	sp.Sizes = append(sp.Sizes[:idx], sp.Sizes[idx+1:]...)
	if len(sp.Items) == 1 {
		fr := sp.FrameRect()
		if sp.Orientation == "vertical" && fr.Height > 0 {
			sp.Sizes[0] = fr.Height
		} else if fr.Width > 0 {
			sp.Sizes[0] = fr.Width
		}
	}
	sp.SetNeedsLayout()
	sp.MarkForRedraw()
	return w
}

// RemoveWidget finds and removes a widget from the stack.
func (sp *StackPanel) RemoveWidget(w meui.Widget) bool {
	for i, item := range sp.Items {
		if item == w {
			sp.RemoveItem(i)
			return true
		}
	}
	return false
}

// IndexOf returns the index of the given widget, or -1.
func (sp *StackPanel) IndexOf(w meui.Widget) int {
	for i, item := range sp.Items {
		if item == w {
			return i
		}
	}
	return -1
}

// ItemCount returns the number of items in the stack.
func (sp *StackPanel) ItemCount() int {
	return len(sp.Items)
}

// splitterRect returns the rect for the splitter after Items[idx].
func (sp *StackPanel) splitterRect(frame graphics.Rect, idx int) graphics.Rect {
	if idx < 0 || idx >= len(sp.Items)-1 {
		return graphics.Rect{}
	}
	total := sp.totalSize(frame)
	pos := float32(0)
	for i := 0; i <= idx; i++ {
		if total > 0 {
			sz := sp.Sizes[i]
			if sz <= 0 {
				sz = 1
			}
			pos += sz
		}
	}
	if sp.Orientation == "vertical" {
		return graphics.NewRect(frame.X, frame.Y+pos, frame.Width, splitterH)
	}
	return graphics.NewRect(frame.X+pos, frame.Y, splitterH, frame.Height)
}

func (sp *StackPanel) totalSize(frame graphics.Rect) float32 {
	if sp.Orientation == "vertical" {
		return frame.Height - splitterH*float32(len(sp.Items)-1)
	}
	return frame.Width - splitterH*float32(len(sp.Items)-1)
}

// hitSplitter returns the splitter index under (x,y), or -1.
func (sp *StackPanel) hitSplitter(x, y float32) int {
	frame := sp.FrameRect()
	for i := 0; i < len(sp.Items)-1; i++ {
		r := sp.splitterRect(frame, i)
		if r.Width > 0 && r.Height > 0 {
			if x >= r.X && x <= r.X+r.Width && y >= r.Y && y <= r.Y+r.Height {
				return i
			}
		}
	}
	return -1
}

func (sp *StackPanel) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !sp.Visible() {
		return
	}
	canvas.Save()
	if sp.NeedsLayout() {
		sp.ValidateLayout()
	}

	frame := sp.FrameRect()
	totalAvail := sp.totalSize(frame)

	// Paint items at local positions (PaintChildren already translates to our origin)
	yOff := float32(0)
	xOff := float32(0)
	for i, item := range sp.Items {
		if item == nil {
			continue
		}
		sz := sp.Sizes[i]
		if sz <= 0 {
			sz = 1
		}
		var itemRect graphics.Rect
		if sp.Orientation == "vertical" {
			itemRect = graphics.NewRect(0, yOff, frame.Width, sz)
			yOff += sz + splitterH
		} else {
			itemRect = graphics.NewRect(xOff, 0, sz, frame.Height)
			xOff += sz + splitterH
		}
		item.SetFrameRect(itemRect)
		if vc, ok := item.(interface{ ValidateLayout() }); ok {
			vc.ValidateLayout()
		}
		canvas.Save()
		// Clip item painting to the panel frame so oversized items don't
		// spill over neighboring widgets.
		if frame.Width > 0 && frame.Height > 0 {
			canvas.ClipRect(graphics.NewRect(0, 0, frame.Width, frame.Height), pathop.Intersect, false)
		}
		canvas.Translate(itemRect.X, itemRect.Y)
		item.Paint(canvas, graphics.NewRect(0, 0, itemRect.Width, itemRect.Height))
		canvas.Restore()

		// Draw splitter after this item (except for last)
		if i < len(sp.Items)-1 {
			var sr graphics.Rect
			if sp.Orientation == "vertical" {
				sr = graphics.NewRect(0, yOff-splitterH, frame.Width, splitterH)
			} else {
				sr = graphics.NewRect(xOff-splitterH, 0, splitterH, frame.Height)
			}
			sp.drawSplitter(canvas, sr, i)
		}
	}
	_ = totalAvail
	canvas.Restore()
}

// drawSplitter paints the splitter after item i with the shared splitter
// look (one theme divider line, optional band and grips). A StackPanel with
// a "vertical" orientation stacks its items, so its splitter band is a wide
// strip and the divider line runs horizontally.
func (sp *StackPanel) drawSplitter(canvas *graphics.Canvas, r graphics.Rect, idx int) {
	hot := sp.hovering && sp.hoverIdx == idx
	paintSplitter(canvas, r, sp.Orientation != "vertical", hot, sp.splitterTheme())
}

// SetHoverColorAlwaysOn pins whether this panel's splitters are always drawn
// in the hover ink, overriding the theme's HoverColorAlwaysOn.
func (sp *StackPanel) SetHoverColorAlwaysOn(on bool) *StackPanel {
	sp.hoverAlways, sp.hoverAlwaysSet = on, true
	sp.MarkForRedraw()
	return sp
}

// splitterTheme resolves the splitter theme with this widget's override
// applied; see SplitView.splitterTheme.
func (sp *StackPanel) splitterTheme() *theme.StackPanelTheme {
	t := sp.StackTheme
	if t == nil {
		t = theme.DefaultStackPanelTheme
	}
	if !sp.hoverAlwaysSet || t == nil || t.HoverColorAlwaysOn == sp.hoverAlways {
		return t
	}
	c := t.Copy()
	c.HoverColorAlwaysOn = sp.hoverAlways
	return c
}

// setHover records which splitter the pointer is on and repaints on a change.
func (sp *StackPanel) setHover(idx int) {
	if idx < 0 {
		if sp.hovering {
			sp.hovering, sp.hoverIdx = false, 0
			sp.MarkForRedraw()
		}
		return
	}
	if sp.hovering && sp.hoverIdx == idx {
		return
	}
	sp.hovering, sp.hoverIdx = true, idx
	sp.MarkForRedraw()
}

// HoverAt implements meui.HoverTarget: the window reports the pointer
// position (widget-local) on every move along the hit chain, so a splitter
// lights up under the pointer and goes back to its normal ink when the
// pointer moves onto either item — the items are the deepest hit, so
// OnMouseMove alone never saw the pointer leave the band.
func (sp *StackPanel) HoverAt(x, y float32) {
	if !sp.Enabled() || !sp.Visible() {
		sp.setHover(-1)
		return
	}
	sp.setHover(sp.hitSplitterLocal(x, y))
}

// CursorAt reports the resize cursor on a splitter band, using the same band
// hitSplitterLocal and OnMouseDown drag from. A splitter is a part of the
// StackPanel's own area (the band lies between the item rects), so the
// window's hit test lands on the panel itself — without this method the walk
// fell through to the default arrow and only SplitView showed a resize
// cursor.
func (sp *StackPanel) CursorAt(x, y float32) *platform.Cursor {
	if !sp.Enabled() || !sp.Visible() {
		return nil
	}
	if sp.hitSplitterLocal(x, y) < 0 {
		return nil
	}
	// "vertical" stacks the items, so its band is a horizontal strip and the
	// drag moves up and down.
	if sp.Orientation == "vertical" {
		return platform.ResizeVerticalCursor()
	}
	return platform.ResizeHorizontalCursor()
}

func (sp *StackPanel) OnMouseDown(x, y float32, button int) bool {
	if !sp.Enabled() {
		return false
	}
	lx, ly := x, y

	if idx := sp.hitSplitterLocal(lx, ly); idx >= 0 {
		sp.dragging = true
		sp.dragIdx = idx
		if sp.Orientation == "vertical" {
			sp.dragStart = ly
		} else {
			sp.dragStart = lx
		}
		sp.dragOrigA = sp.Sizes[idx]
		sp.dragOrigB = sp.Sizes[idx+1]
		return true
	}

	// Forward to items using local coordinates
	yOff := float32(0)
	xOff := float32(0)
	for i, item := range sp.Items {
		if item == nil {
			continue
		}
		sz := sp.Sizes[i]
		if sz <= 0 {
			sz = 1
		}
		var itemRect graphics.Rect
		if sp.Orientation == "vertical" {
			itemRect = graphics.NewRect(0, yOff, sp.FrameRect().Width, sz)
			yOff += sz + splitterH
		} else {
			itemRect = graphics.NewRect(xOff, 0, sz, sp.FrameRect().Height)
			xOff += sz + splitterH
		}
		if lx >= itemRect.X && lx <= itemRect.X+itemRect.Width && ly >= itemRect.Y && ly <= itemRect.Y+itemRect.Height {
			if el, ok := item.(meui.EventListener); ok {
				return el.OnMouseDown(lx-itemRect.X, ly-itemRect.Y, button)
			}
			return true
		}
	}
	return false
}

func (sp *StackPanel) hitSplitterLocal(lx, ly float32) int {
	if len(sp.Items) < 2 {
		return -1
	}
	yOff := float32(0)
	xOff := float32(0)
	for i := 0; i < len(sp.Items)-1; i++ {
		sz := sp.Sizes[i]
		if sz <= 0 {
			sz = 1
		}
		yOff += sz
		xOff += sz
		if sp.Orientation == "vertical" {
			if ly >= yOff && ly <= yOff+splitterH {
				return i
			}
			yOff += splitterH
		} else {
			if lx >= xOff && lx <= xOff+splitterH {
				return i
			}
			xOff += splitterH
		}
	}
	return -1
}

func (sp *StackPanel) OnMouseUp(x, y float32, button int) bool {
	sp.dragging = false
	lx, ly := x, y
	yOff := float32(0)
	xOff := float32(0)
	for i, item := range sp.Items {
		if item == nil {
			continue
		}
		sz := sp.Sizes[i]
		if sz <= 0 {
			sz = 1
		}
		var itemRect graphics.Rect
		if sp.Orientation == "vertical" {
			itemRect = graphics.NewRect(0, yOff, sp.FrameRect().Width, sz)
			yOff += sz + splitterH
		} else {
			itemRect = graphics.NewRect(xOff, 0, sz, sp.FrameRect().Height)
			xOff += sz + splitterH
		}
		if lx >= itemRect.X && lx <= itemRect.X+itemRect.Width && ly >= itemRect.Y && ly <= itemRect.Y+itemRect.Height {
			if el, ok := item.(meui.EventListener); ok {
				return el.OnMouseUp(lx-itemRect.X, ly-itemRect.Y, button)
			}
			return true
		}
	}
	return false
}

func (sp *StackPanel) OnMouseMove(x, y float32) {
	if sp.dragging && sp.dragIdx >= 0 {
		// x/y are widget-LOCAL (see OnMouseDown): the drag start was
		// recorded here too, so do NOT subtract FrameRect() (parent-
		// relative) — that mixed spaces and jumped the splitter by the
		// panel's position in its parent.
		var pos float32
		if sp.Orientation == "vertical" {
			pos = y
		} else {
			pos = x
		}
		delta := pos - sp.dragStart
		newA := sp.dragOrigA + delta
		newB := sp.dragOrigB - delta
		if newA < stackMinSize {
			newA = stackMinSize
			newB = sp.dragOrigA + sp.dragOrigB - stackMinSize
		}
		if newB < stackMinSize {
			newB = stackMinSize
			newA = sp.dragOrigA + sp.dragOrigB - stackMinSize
		}
		sp.Sizes[sp.dragIdx] = newA
		sp.Sizes[sp.dragIdx+1] = newB
		sp.MarkForRedraw()
		return
	}
	lx, ly := x, y
	yOff := float32(0)
	xOff := float32(0)
	for i, item := range sp.Items {
		if item == nil {
			continue
		}
		sz := sp.Sizes[i]
		if sz <= 0 {
			sz = 1
		}
		var itemRect graphics.Rect
		if sp.Orientation == "vertical" {
			itemRect = graphics.NewRect(0, yOff, sp.FrameRect().Width, sz)
			yOff += sz + splitterH
		} else {
			itemRect = graphics.NewRect(xOff, 0, sz, sp.FrameRect().Height)
			xOff += sz + splitterH
		}
		if lx >= itemRect.X && lx <= itemRect.X+itemRect.Width && ly >= itemRect.Y && ly <= itemRect.Y+itemRect.Height {
			if el, ok := item.(meui.EventListener); ok {
				el.OnMouseMove(lx-itemRect.X, ly-itemRect.Y)
			}
			return
		}
	}
}

func (sp *StackPanel) ValidateLayout() {
	if !sp.NeedsLayout() {
		return
	}
	frame := sp.FrameRect()
	avail := sp.totalSize(frame)
	if avail <= 0 || len(sp.Items) == 0 {
		sp.ClearNeedsLayout()
		return
	}
	// Sum current sizes
	total := float32(0)
	for _, sz := range sp.Sizes {
		total += sz
	}
	if total <= 0 {
		// Distribute evenly
		even := avail / float32(len(sp.Sizes))
		for i := range sp.Sizes {
			sp.Sizes[i] = even
		}
	} else {
		// Scale proportionally to fill available space
		rem := avail
		n := len(sp.Sizes)
		for i, sz := range sp.Sizes {
			if i == n-1 {
				sp.Sizes[i] = rem
			} else {
				scaled := sz / total * avail
				sp.Sizes[i] = scaled
				rem -= scaled
			}
		}
	}
	sp.ClearNeedsLayout()
}

// UpdateItemFrames sets the frame of each item based on current sizes,
// without painting. Called after structural changes so DropHitTest sees
// valid frames immediately (not deferred to the next Paint cycle).
func (sp *StackPanel) UpdateItemFrames() {
	frame := sp.FrameRect()
	yOff := float32(0)
	xOff := float32(0)
	for i, item := range sp.Items {
		if item == nil {
			continue
		}
		sz := sp.Sizes[i]
		if sz <= 0 {
			sz = 1
		}
		if sp.Orientation == "vertical" {
			item.SetFrameRect(graphics.NewRect(0, yOff, frame.Width, sz))
			yOff += sz + splitterH
		} else {
			item.SetFrameRect(graphics.NewRect(xOff, 0, sz, frame.Height))
			xOff += sz + splitterH
		}
		// Validate item's internal layout (e.g., TabPanel needs content rect)
		if vc, ok := item.(interface{ ValidateLayout() }); ok {
			vc.ValidateLayout()
		}
	}
}

// DropHitTest returns a DropZoneInfo if (lx, ly) in StackPanel-local coords
// is over an item's title/tab bar (merge) or between items (insert).
func (sp *StackPanel) DropHitTest(lx, ly float32) *DropZoneInfo {
	for i, item := range sp.Items {
		if item == nil {
			continue
		}
		icf := item.FrameRect()
		if lx >= icf.X && lx <= icf.X+icf.Width && ly >= icf.Y && ly <= icf.Y+icf.Height {
			itemLocalY := ly - icf.Y
			if mw, ok := item.(*MdiWindow); ok {
				if itemLocalY >= 0 && itemLocalY <= mw.titleBarHeight() {
					return &DropZoneInfo{
						Type:     DropMerge,
						Target:   mw,
						Stack:    sp,
						StackIdx: i,
						ZoneRect: graphics.NewRect(icf.X, icf.Y, icf.Width, mw.titleBarHeight()),
					}
				}
			}
			if tp, ok := item.(*TabPanel); ok {
				if info := tp.DropHitTest(lx-icf.X, ly-icf.Y); info != nil {
					info.ZoneRect = graphics.NewRect(
						icf.X+info.ZoneRect.X,
						icf.Y+info.ZoneRect.Y,
						info.ZoneRect.Width,
						info.ZoneRect.Height,
					)
					info.Stack = sp
					info.StackIdx = i
					return info
				}
			}
		}
	}
	return nil
}

func (sp *StackPanel) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "orientation":
		if s, ok := value.(string); ok {
			sp.Orientation = s
		}
	}
}
