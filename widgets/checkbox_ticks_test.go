package widgets

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
)

// TestContextMenuCheckableItemToggles: clicking a checkable menu item
// must flip its Checked state — the drawn box was decorative before.
func TestContextMenuCheckableItemToggles(t *testing.T) {
	m := NewContextMenu()
	m.AddItem("Toggle me", func() {})
	m.AddCheckItem("Option", false, nil)

	// Item layout: padding 4, items 28px tall → the check item is at
	// y in [4+28, 4+56).
	y := float32(4 + 28 + 14)
	if !m.OnMouseDown(10, y, 0) {
		t.Fatal("click on the checkable item was not consumed")
	}
	if !m.items[1].Checked {
		t.Error("first click did not check the item")
	}
	m2 := NewContextMenu()
	m2.AddCheckItem("Option", false, nil)
	m2.OnMouseDown(10, 4+14, 0)
	if !m2.items[0].Checked {
		t.Error("first click did not check the only item")
	}
	m2.OnMouseDown(10, 4+14, 0)
	if m2.items[0].Checked {
		t.Error("second click did not uncheck the item")
	}
}

// TestTreeRowCheckboxToggles: clicking a node's checkbox box toggles the
// node (indent + chevron area precede the box).
func TestTreeRowCheckboxToggles(t *testing.T) {
	tr := NewTree()
	root := &TreeNode{Label: "root", Expanded: true}
	child := &TreeNode{Label: "check me", Checkable: true, Depth: 1}
	root.Children = append(root.Children, child)
	tr.SetRoot(root)
	tr.SetFrameRect(graphics.NewRect(0, 0, 200, 200))

	h := listRowHeight()
	// Child row index 1; checkbox at x = 4 (start) + 16 (indent) + 14
	// (chevron) .. +14 (box).
	x := float32(4 + 16 + 14 + 7)
	y := h*1.5 + tr.ScrollOffset
	if !tr.OnMouseDown(x, y, 0) {
		t.Fatal("click on the row checkbox was not consumed")
	}
	if !tr.flatNodes[1].Checked {
		t.Errorf("checkbox click did not check the node (x=%.0f y=%.0f)", x, y)
	}
}
