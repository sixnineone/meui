package behaviour

import (
	"testing"

	"github.com/sixnineone/meui/widgets"
)

const treeQML = `
Window {
    width: 400
    height: 400
    VStack {
        margins: 8
        Tree {
            id: tree
            width: 350
            height: 300
        }
    }
}
`

// loadTree parses treeQML and populates a 2-level tree:
// root(expanded) -> parent1(expanded) -> [child1a, child1b], parent2(collapsed) -> [child2]
func loadTree(t *testing.T) (*Harness, *widgets.Tree) {
	t.Helper()
	h := Load(t, treeQML)
	tr := h.Tree("tree")
	root := tr.Root
	p1 := tr.AddChild(root, "parent-one")
	tr.AddChild(p1, "child-a")
	tr.AddChild(p1, "child-b")
	p2 := tr.AddChild(root, "parent-two")
	tr.AddChild(p2, "child-c")
	tr.ExpandAll() // AddChild defaults new nodes to collapsed
	return h, tr
}

func TestTreeInitialExpanded(t *testing.T) {
	_, tr := loadTree(t)
	// New nodes default to expanded=true (TreeNode default); verify flat
	// view contains all 6 rows (root, p1, c-a, c-b, p2, c-c).
	if n := tr.FlatCount(); n != 6 {
		t.Fatalf("flat rows: got %d, want 6", n)
	}
}

func TestTreeChevronClickCollapses(t *testing.T) {
	h, tr := loadTree(t)
	tf := tr.FrameRect()
	// parent-one is row 1 (root is row 0); its chevron sits at
	// x = 4 + depth*16 .. +14 with depth 1. Clicks are window coords.
	chevX := tf.X + float32(4+16+7) // center of the chevron strip
	rowY := tf.Y + float32(1*24+12) // row 1 center
	h.Click(chevX, rowY)
	// After collapsing parent-one its children disappear from the flat view.
	if n := tr.FlatCount(); n != 4 {
		t.Fatalf("after collapse: flat rows %d, want 4", n)
	}
	// Click again: re-expands.
	h.Click(chevX, rowY)
	if n := tr.FlatCount(); n != 6 {
		t.Fatalf("after re-expand: flat rows %d, want 6", n)
	}
}

func TestTreeCollapseAll(t *testing.T) {
	h, tr := loadTree(t)
	// A LABELED root is a real row: CollapseAll leaves only that row
	// (QTreeWidget parity).
	tr.CollapseAll()
	if n := tr.FlatCount(); n != 1 {
		t.Fatalf("CollapseAll (labeled root): flat rows %d, want 1", n)
	}
	tr.ExpandAll()
	if n := tr.FlatCount(); n != 6 {
		t.Fatalf("ExpandAll: flat rows %d, want 6", n)
	}

	// A VIRTUAL root (empty label) stays expanded: its top-level
	// children remain visible after CollapseAll.
	vt := h.Tree("tree")
	vt.SetRoot(&widgets.TreeNode{Expanded: true})
	pa := vt.AddChild(vt.Root, "top-a")
	vt.AddChild(pa, "kid")
	vt.AddChild(vt.Root, "top-b")
	vt.CollapseAll()
	if n := vt.FlatCount(); n != 2 {
		t.Fatalf("CollapseAll (virtual root): flat rows %d, want 2", n)
	}
}

func TestTreeClickLabelSelects(t *testing.T) {
	h, tr := loadTree(t)
	tf := tr.FrameRect()
	// Click the label of row 1 ("parent-one"): past the chevron strip.
	labelX := tf.X + float32(4+16+20)
	rowY := tf.Y + float32(1*24+12)
	h.Click(labelX, rowY)
	node := tr.GetSelectedNode()
	if node == nil {
		t.Fatal("label click selected nothing")
	}
	if node.Label != "parent-one" {
		t.Fatalf("selected %q, want parent-one", node.Label)
	}
}

const splitQML = `
Window {
    width: 500
    height: 400
    SplitView {
        id: split
        width: 480
        height: 380
        splitPosition: 0.5
        orientation: "horizontal"
        Rectangle { id: leftPane; width: 200; color: "#555577" }
        Rectangle { id: rightPane; width: 200; color: "#557755" }
    }
}
`

func TestSplitViewDividerDrag(t *testing.T) {
	h := Load(t, splitQML)
	sv := h.Widget("split").(*widgets.SplitView)
	sv.ValidateLayout()

	f := sv.FrameRect()
	start := f.X + f.Width*0.5
	h.Drag(start, f.Y+f.Height/2, start+60, f.Y+f.Height/2)

	pos := sv.SplitPosition
	if pos <= 0.5 {
		t.Fatalf("split position after rightward drag: %v, want > 0.5", pos)
	}
	// The divider moved right by ~60px of a 480px frame ≈ 0.125.
	if pos < 0.6 {
		t.Fatalf("split position moved too little: %v", pos)
	}
}

func TestSplitViewPanesFollowSplit(t *testing.T) {
	h := Load(t, splitQML)
	sv := h.Widget("split").(*widgets.SplitView)
	sv.ValidateLayout()
	f := sv.FrameRect()

	left := h.Widget("leftPane")
	right := h.Widget("rightPane")
	lf0 := left.FrameRect()
	sv.SetSplitPosition(0.75)
	sv.ValidateLayout()
	lf1 := left.FrameRect()

	if lf1.Width <= lf0.Width {
		t.Fatalf("left pane did not grow: %v -> %v", lf0.Width, lf1.Width)
	}
	// Right pane shrinks and stays inside the frame.
	rf := right.FrameRect()
	if rf.X+rf.Width > f.X+f.Width+1 {
		t.Fatalf("right pane overflows: %v", rf)
	}
	if rf.Width < 20 {
		t.Fatalf("right pane collapsed to nothing: %v", rf)
	}
}

// TestTreeWheelScrollSyncsThumbAndContent: wheel scrolling must move the
// content offset AND keep the scrollbar value consistent (absolute-pixel
// units via ScrollbarHost). Guards against the cached-bitmap desync.
func TestTreeWheelScrollSyncsThumbAndContent(t *testing.T) {
	h, tr := loadTree(t)

	// Grow the tree so it overflows the 300px viewport.
	root := tr.Root
	for i := 0; i < 30; i++ {
		tr.AddChild(root, "node")
	}
	h.Settle()

	total := float32(tr.FlatCount()) * tr.ItemHeight
	frame := tr.FrameRect()
	if total <= frame.Height {
		t.Skipf("tree does not overflow (%v <= %v)", total, frame.Height)
	}

	x, y := Center(tr)
	h.Wheel(x, y, 0, -5) // scroll down 5 notches
	h.Settle()

	if tr.ScrollOffset <= 0 {
		t.Fatalf("wheel down: ScrollOffset %v should be > 0", tr.ScrollOffset)
	}
	maxScroll := total - frame.Height
	if tr.ScrollOffset > maxScroll {
		t.Errorf("ScrollOffset %v exceeds max %v", tr.ScrollOffset, maxScroll)
	}
	// Absolute-pixel units: bar value == offset, Max == content extent.
	if tr.VScroll.Value != tr.ScrollOffset {
		t.Errorf("bar value %v != ScrollOffset %v", tr.VScroll.Value, tr.ScrollOffset)
	}
	if tr.VScroll.Max != total {
		t.Errorf("bar Max %v != content extent %v", tr.VScroll.Max, total)
	}

	// Wheel back up: offset returns to 0.
	for i := 0; i < 20; i++ {
		h.Wheel(x, y, 0, 1)
	}
	h.Settle()
	if tr.ScrollOffset != 0 {
		t.Errorf("wheel to top: ScrollOffset %v, want 0", tr.ScrollOffset)
	}
	if tr.VScroll.Value != 0 {
		t.Errorf("wheel to top: bar value %v, want 0", tr.VScroll.Value)
	}
}
