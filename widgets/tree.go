package widgets

import (
	"fmt"
	"os"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

type TreeNode struct {
	Label        string
	Icon         string
	Checkable    bool
	Checked      bool
	Expanded     bool
	Children     []*TreeNode
	Depth        int
	labelText    *graphics.Text
	labelTextVer uint64
}

type Tree struct {
	meui.BaseWidget
	*theme.ListViewTheme
	Root            *TreeNode
	SelectedIndex   int
	HoveredIndex    int
	ScrollOffset    float32
	host            *ScrollbarHost
	ItemHeight      float32
	AlternatingRows bool
	// HideBorder drops the widget's own outline (see ListView.HideBorder).
	HideBorder     bool
	OnItemSelected func(int, *TreeNode)
	OnCheckChanged func(*TreeNode)
	flatNodes      []*TreeNode
	chevronRight   *meui.SVGDocument
	expandMore     *meui.SVGDocument
	cache          meui.CacheBase
	VScroll        *ScrollBar
}

func NewTree() *Tree {
	t := &Tree{
		BaseWidget:    *meui.NewBaseWidget(),
		ListViewTheme: theme.DefaultListViewTheme,
		ItemHeight:    24,
		SelectedIndex: -1,
		HoveredIndex:  -1,
		Root:          &TreeNode{Label: "Root", Expanded: true},
	}
	t.host = NewScrollbarHost()
	t.host.ShowH = false // trees scroll vertically
	t.VScroll = t.host.VScroll
	t.VScroll.SetParent(t)
	t.host.SetOnChange(func() {
		_, offY := t.host.Offsets()
		t.ScrollOffset = offY
		// The cached bitmap bakes the item positions at the old offset —
		// without invalidation the thumb moves but the content does not.
		t.cache.Invalidate()
		t.MarkForRedraw()
	})
	t.SetTabIndex(0)
	if doc, err := meui.LoadSVGByName("chevron_right"); err == nil {
		t.chevronRight = doc
	}
	if doc, err := meui.LoadSVGByName("expand_more"); err == nil {
		t.expandMore = doc
	}
	return t
}

func (t *Tree) SetRoot(root *TreeNode) *Tree {
	t.Root = root
	t.flattenNodes()
	t.cache.Invalidate()
	t.MarkForRedraw()
	return t
}

func (t *Tree) AddChild(parent *TreeNode, label string) *TreeNode {
	child := &TreeNode{Label: label, Depth: parent.Depth + 1}
	parent.Children = append(parent.Children, child)
	t.flattenNodes()
	t.cache.Invalidate()
	t.MarkForRedraw()
	return child
}

// FlatCount returns the number of visible (expanded) rows.
func (t *Tree) FlatCount() int { return len(t.flatNodes) }

func (t *Tree) ToggleNode(index int) {
	if index >= 0 && index < len(t.flatNodes) {
		node := t.flatNodes[index]
		if len(node.Children) > 0 {
			node.Expanded = !node.Expanded
			t.flattenNodes()
			t.cache.Invalidate()
			t.MarkForRedraw()
		}
	}
}

// GetSelectedNode returns the currently selected node, or nil.
func (t *Tree) GetSelectedNode() *TreeNode {
	if t.SelectedIndex >= 0 && t.SelectedIndex < len(t.flatNodes) {
		return t.flatNodes[t.SelectedIndex]
	}
	return nil
}

// SetOnNodeSelected registers the selection callback.
func (t *Tree) SetOnNodeSelected(handler func(index int, node *TreeNode)) *Tree {
	t.OnItemSelected = handler
	return t
}

// ExpandAll expands every node with children.
func (t *Tree) ExpandAll() {
	t.setExpandedRecursive(t.Root, true)
	if t.Root != nil {
		t.Root.Expanded = true
	}
	t.flattenNodes()
	t.cache.Invalidate()
	t.MarkForRedraw()
}

// CollapseAll collapses every node with children. Top-level items stay
// visible (the virtual root is kept expanded), matching QTreeWidget.
func (t *Tree) CollapseAll() {
	t.setExpandedRecursive(t.Root, false)
	if t.Root != nil && t.Root.Label == "" {
		t.Root.Expanded = true
	}
	t.flattenNodes()
	t.cache.Invalidate()
	t.MarkForRedraw()
}

// ExpandNode sets the expansion state of a single node. When expanding, all
// ancestors are expanded too so the node is visible.
func (t *Tree) ExpandNode(node *TreeNode, expanded bool) {
	if node == nil {
		return
	}
	if len(node.Children) > 0 {
		node.Expanded = expanded
	}
	if expanded {
		t.expandAncestors(t.Root, node)
	}
	t.flattenNodes()
	t.cache.Invalidate()
	t.MarkForRedraw()
}

// expandAncestors walks from root toward target, expanding every ancestor.
func (t *Tree) expandAncestors(current, target *TreeNode) bool {
	if current == nil {
		return false
	}
	if current == target {
		return true
	}
	for _, child := range current.Children {
		if t.expandAncestors(child, target) {
			if len(current.Children) > 0 {
				current.Expanded = true
			}
			return true
		}
	}
	return false
}

// RemoveNode removes a node from its parent (or clears the root when node is
// the root).
func (t *Tree) RemoveNode(node *TreeNode) {
	if node == nil {
		return
	}
	if node == t.Root {
		t.Root = nil
		t.SelectedIndex = -1
		t.flattenNodes()
		t.cache.Invalidate()
		t.MarkForRedraw()
		return
	}
	var remove func(parent *TreeNode) bool
	remove = func(parent *TreeNode) bool {
		for i, child := range parent.Children {
			if child == node {
				parent.Children = append(parent.Children[:i], parent.Children[i+1:]...)
				return true
			}
			if remove(child) {
				return true
			}
		}
		return false
	}
	if remove(t.Root) {
		t.SelectedIndex = -1
		t.flattenNodes()
		t.cache.Invalidate()
		t.MarkForRedraw()
	}
}

func (t *Tree) setExpandedRecursive(node *TreeNode, expanded bool) {
	if node == nil {
		return
	}
	if len(node.Children) > 0 {
		node.Expanded = expanded
	}
	for _, child := range node.Children {
		t.setExpandedRecursive(child, expanded)
	}
}

func (t *Tree) flattenNodes() {
	t.flatNodes = nil
	t.flattenRecursive(t.Root, 0)
}

func (t *Tree) flattenRecursive(node *TreeNode, depth int) {
	if node == nil {
		return
	}
	// A labeled root is a real visible node; an empty-label root is a
	// virtual container that is skipped (its children become top-level).
	isVirtualRoot := node == t.Root && node.Label == ""
	if isVirtualRoot {
		depth = -1 // children flatten at depth 0
	} else {
		node.Depth = depth
		t.flatNodes = append(t.flatNodes, node)
	}
	if node.Expanded {
		for _, child := range node.Children {
			t.flattenRecursive(child, depth+1)
		}
	}
}

func (t *Tree) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
		fmt.Printf("[treePaint] flat=%d visible=%v rect=(%.0f,%.0f %.0fx%.0f)\n", len(t.flatNodes), t.Visible(), rect.X, rect.Y, rect.Width, rect.Height)
	}
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
			bgPaint.SetColor(t.ListViewTheme.BackgroundInk)
		}
		c.DrawRoundedRect(r, t.ListViewTheme.CornerRadius, t.ListViewTheme.CornerRadius, bgPaint)

		if !t.HideBorder {
			borderPaint := graphics.NewPaint()
			borderPaint.SetStyle(paintstyle.Stroke)
			borderPaint.SetStrokeWidth(theme.BorderStrokeWidth)
			if disabled {
				borderPaint.SetColor(theme.DefaultTheme.DisabledBorder)
			} else {
				borderPaint.SetColor(t.ListViewTheme.EdgeInk)
			}
			c.DrawRoundedRect(r, t.ListViewTheme.CornerRadius, t.ListViewTheme.CornerRadius, borderPaint)
		}

		// Clip to prevent items from rendering outside bounds (exclude scrollbar)
		cr := r
		if t.host.VisibleV() {
			cr.Width -= t.host.BarSize
		}
		c.ClipRect(cr, pathop.Intersect, false)

		font := t.ListViewTheme.Font
		if font == nil {
			font = theme.DefaultTheme.BaseFont
			if font == nil {
				font = graphics.SystemFont
			}
		}

		innerY := r.Y - t.ScrollOffset
		for i, node := range t.flatNodes {
			itemRect := graphics.NewRect(r.X, innerY, r.Width, t.ItemHeight)

			if !disabled && i == t.SelectedIndex {
				selPaint := graphics.NewPaint()
				selPaint.SetColor(t.ListViewTheme.SelectedInk)
				c.DrawRect(itemRect, selPaint)
			} else if !disabled && i == t.HoveredIndex {
				hoverPaint := graphics.NewPaint()
				hoverPaint.SetColor(t.ListViewTheme.HoverInk)
				c.DrawRect(itemRect, hoverPaint)
			} else if t.AlternatingRows && i%2 == 1 {
				altPaint := graphics.NewPaint()
				altPaint.SetColor(t.ListViewTheme.AlternateRowInk)
				c.DrawRect(itemRect, altPaint)
			}

			indent := float32(node.Depth) * 16
			x := r.X + 4 + indent

			if len(node.Children) > 0 {
				arrowSize := float32(10)
				arrowY := innerY + (t.ItemHeight-arrowSize)/2
				if node.Expanded {
					if t.expandMore != nil {
						meui.DrawSVG(c, t.expandMore, x, arrowY, arrowSize, arrowSize)
					}
				} else {
					if t.chevronRight != nil {
						meui.DrawSVG(c, t.chevronRight, x, arrowY, arrowSize, arrowSize)
					}
				}
				x += 14
			} else {
				x += 14
			}

			// Icon
			if node.Icon != "" {
				iconSize := float32(16)
				iconY := innerY + (t.ItemHeight-iconSize)/2
				if doc, err := meui.LoadSVGByName(node.Icon); err == nil {
					meui.DrawSVG(c, doc, x, iconY, iconSize, iconSize)
					x += 18
				}
			}

			// Checkbox
			if node.Checkable {
				boxSize := float32(14)
				boxY := innerY + (t.ItemHeight-boxSize)/2
				DrawCheckBox(c, x, boxY, boxSize, node.Checked, !disabled)
				x += boxSize + 4
			}

			textPaint := graphics.NewPaint()
			if disabled {
				textPaint.SetColor(theme.DefaultTheme.DisabledText)
			} else if i == t.SelectedIndex {
				textPaint.SetColor(t.ListViewTheme.TextInk)
			} else {
				textPaint.SetColor(t.ListViewTheme.TextInk)
			}
			ver := theme.ThemeVersion()
			if node.labelText == nil || node.labelTextVer != ver {
				node.labelText = graphics.NewText(node.Label, &graphics.TextDecoration{Font: font, OnBackgroundInk: textPaint.Color()})
				node.labelTextVer = ver
			}
			if node.labelText != nil {
				node.labelText.Draw(c, x, innerY+t.ItemHeight/2+font.Baseline()/2)
			}

			innerY += t.ItemHeight
		}
	})

	// Scrollbar: absolute-pixel extents via the host; the host clamps
	// the offset to the content extent.
	frame := t.FrameRect()
	totalHeight := float32(len(t.flatNodes)) * t.ItemHeight
	t.host.SetExtents(frame.Width, totalHeight, frame.Width, frame.Height)
	t.host.LayoutBars(rect.Width, rect.Height)
	_, t.ScrollOffset = t.host.Offsets()
	t.host.PaintBars(canvas)

	canvas.Restore()
}

// SetBoundProperty applies QML properties. hideBorder / border.width: 0
// drop the widget's outline (embedded trees have no need for a frame of
// their own next to a ScrollPane lane).
func (t *Tree) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "hideBorder":
		if b, ok := value.(bool); ok {
			t.HideBorder = b
			t.MarkForRedraw()
		}
	case "border.width", "borderWidth":
		if f, ok := asFloat(value); ok {
			t.HideBorder = f <= 0
			t.MarkForRedraw()
		}
	}
}

func (t *Tree) OnMouseDown(x, y float32, button int) bool {
	if !t.Enabled() {
		return false
	}
	frame := t.FrameRect()
	if t.host.HandleMouseDown(frame.Width, frame.Height, x, y, button) {
		return true
	}
	adjustedY := y + t.ScrollOffset
	index := int(adjustedY / t.ItemHeight)
	if index >= 0 && index < len(t.flatNodes) {
		node := t.flatNodes[index]
		indent := float32(node.Depth) * 16
		startX := float32(4) + indent

		// Chevron toggle area
		if x >= startX && x < startX+14 && len(node.Children) > 0 {
			t.ToggleNode(index)
			return true
		}

		// Compute x position past chevron, icon, and checkbox
		itemX := startX + 14
		if node.Icon != "" {
			itemX += 18
		}
		boxSize := float32(14)
		if node.Checkable && x >= itemX && x < itemX+boxSize {
			node.Checked = !node.Checked
			t.cache.Invalidate()
			t.MarkForRedraw()
			if t.OnCheckChanged != nil {
				t.OnCheckChanged(node)
			}
			return true
		}
		if node.Checkable {
			itemX += boxSize + 4
		}

		// Select
		t.SelectedIndex = index
		t.cache.Invalidate()
		t.MarkForRedraw()
		if t.OnItemSelected != nil {
			t.OnItemSelected(index, node)
		}
		return true
	}
	return false
}

func (t *Tree) OnMouseMove(x, y float32) {
	if !t.Enabled() {
		return
	}
	t.host.UpdateHover(x, y) // overlay pills reveal on strip hover
	// Route to scrollbar during drag
	frame := t.FrameRect()
	if t.VScroll.dragging {
		t.host.HandleMouseMove(x, y)
		return
	}
	// Ignore hover in scrollbar zone
	fw := frame.Width
	if t.host.VisibleV() && x >= fw-t.host.BarSize {
		if t.HoveredIndex != -1 {
			t.HoveredIndex = -1
			t.MarkForRedraw()
		}
		return
	}
	adjustedY := y + t.ScrollOffset
	index := int(adjustedY / t.ItemHeight)
	oldHovered := t.HoveredIndex
	t.HoveredIndex = -1
	if index >= 0 && index < len(t.flatNodes) {
		t.HoveredIndex = index
	}
	if oldHovered != t.HoveredIndex {
		t.MarkForRedraw()
	}
}

func (t *Tree) OnMouseWheel(x, y float32, dx, dy float32) bool {
	if !t.Enabled() || dy == 0 {
		return false
	}
	frame := t.FrameRect()
	totalHeight := float32(len(t.flatNodes)) * t.ItemHeight
	t.host.SetExtents(frame.Width, totalHeight, frame.Width, frame.Height)
	if !t.host.HandleWheel(dx, dy) {
		return false
	}
	_, t.ScrollOffset = t.host.Offsets()
	t.cache.Invalidate()
	t.MarkForRedraw()
	return true
}

func (t *Tree) OnMouseUp(x, y float32, button int) bool {
	// Always end scrollbar drag even if mouse is outside the scrollbar zone
	if t.VScroll.dragging {
		t.host.HandleMouseUp(x, y, button)
		return true
	}
	return true
}

// OnKeyRepeat is a key the user is HOLDING — Backspace over a word, an arrow along a row, a scrub.
// The window classifies an OS auto-repeat as a repeat rather than a fresh press (a widget without
// this method is asked once, deliberately), so this is what makes a held key work here.
func (t *Tree) OnKeyRepeat(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	return t.OnKeyDown(keyCode, mod)
}

func (t *Tree) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if !t.Enabled() || len(t.flatNodes) == 0 {
		return false
	}
	switch keyCode {
	case platform.KeyUp:
		if t.SelectedIndex > 0 {
			t.SelectedIndex--
			if t.OnItemSelected != nil {
				t.OnItemSelected(t.SelectedIndex, t.flatNodes[t.SelectedIndex])
			}
			t.cache.Invalidate()
			t.MarkForRedraw()
		}
		return true
	case platform.KeyDown:
		if t.SelectedIndex < len(t.flatNodes)-1 {
			t.SelectedIndex++
			if t.OnItemSelected != nil {
				t.OnItemSelected(t.SelectedIndex, t.flatNodes[t.SelectedIndex])
			}
			t.cache.Invalidate()
			t.MarkForRedraw()
		}
		return true
	case platform.KeyRight:
		if t.SelectedIndex >= 0 && t.SelectedIndex < len(t.flatNodes) {
			node := t.flatNodes[t.SelectedIndex]
			if !node.Expanded && len(node.Children) > 0 {
				t.ToggleNode(t.SelectedIndex)
			}
		}
		return true
	case platform.KeyLeft:
		if t.SelectedIndex >= 0 && t.SelectedIndex < len(t.flatNodes) {
			node := t.flatNodes[t.SelectedIndex]
			if node.Expanded && len(node.Children) > 0 {
				t.ToggleNode(t.SelectedIndex)
			}
		}
		return true
	}
	return false
}

func (t *Tree) Focusable() bool {
	return t.Enabled()
}
