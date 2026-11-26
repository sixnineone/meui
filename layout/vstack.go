package layout

import (
	"fmt"
	"github.com/sixnineone/meui/graphics"
	"os"
	"runtime"
)

// Max layout depth to prevent infinite loops
const maxLayoutDepth = 100

func init() {
	// Check if MEUI_RECURSIVE_LAYOUT_DETECT is enabled
	// This is a simplified version - full check would require importing meui package
}

func checkRecursiveLayout(depth int) bool {
	if depth > maxLayoutDepth {
		fmt.Fprintf(os.Stderr, "MeUi: Recursive layout detected (depth %d > %d)\n", depth, maxLayoutDepth)
		buf := make([]byte, 4096)
		n := runtime.Stack(buf, false)
		fmt.Fprintf(os.Stderr, "%s", buf[:n])
		return true
	}
	return false
}

type LayoutProps struct {
	FillWidth       bool
	FillHeight      bool
	PreferredWidth  float32
	PreferredHeight float32
	MinimumWidth    float32
	MaximumWidth    float32
	MinimumHeight   float32
	MaximumHeight   float32
	LeftMargin      float32
	RightMargin     float32
	TopMargin       float32
	BottomMargin    float32
	Alignment       int
}

// Alignment constants for VStack/HStack
const (
	AlignNoneV    = 0
	AlignLeftV    = 1
	AlignRightV   = 2
	AlignHCenterV = 3
	AlignTopV     = 4
	AlignBottomV  = 5
	AlignVCenterV = 6
	AlignCenterV  = 7
)

type VStack struct {
	Spacing     float32
	inLayout    bool
	layoutDepth int
}

func NewVStack() *VStack {
	return &VStack{Spacing: 4}
}

func (l *VStack) Layout(target interface{}, rect graphics.Rect) {
	// Check for recursive layout
	if checkRecursiveLayout(l.layoutDepth) {
		return
	}
	if l.inLayout {
		return
	}
	l.inLayout = true
	l.layoutDepth++
	defer func() {
		l.inLayout = false
		l.layoutDepth--
	}()

	// Do NOT resize the target. The rect here is the panel's PADDED inner
	// rect (origin and size already inset by the panel's padding), and the
	// children below are positioned with the same padding offset inside it.
	// Shrinking the target to this rect applied the padding twice: children
	// then stuck out past the panel's frame by the padding, and whoever
	// clips the panel (the tab page, a parent scroll region) cut the last
	// padding pixels off — clipped bottom edges and scrollbar thumb caps.

	children := getChildren(target)
	if len(children) == 0 {
		return
	}

	// First, apply anchors to children that have them
	anchorLayout := NewAnchorLayout()
	anchorLayout.Layout(target, rect)

	type childInfo struct {
		child    interface{}
		frame    graphics.Rect
		props    *LayoutProps
		fillW    bool
		fillH    bool
		naturalH float32
		naturalW float32
	}
	infos := make([]childInfo, 0, len(children))

	for _, child := range children {
		if child == nil {
			continue
		}
		// Skip children that have anchors (already positioned by AnchorLayout)
		if HasAnchorsOnChild(child) {
			continue
		}
		frame := graphics.Rect{}
		if f, ok := child.(interface{ FrameRect() graphics.Rect }); ok {
			frame = f.FrameRect()
		}

		var props *LayoutProps
		if lp, ok := child.(interface{ GetLayoutProps() *LayoutProps }); ok {
			props = lp.GetLayoutProps()
		}

		// Get natural height: prefer measured height if larger than frame height
		naturalH := frame.Height
		naturalW := frame.Width
		// Measure the child's preferred size, EXCEPT when the author set an
		// explicit width/height — a fixed-size wrapping label must keep its
		// set width, not grow to its unwrapped text width.
		explicitW, explicitH := false, false
		if es, ok := child.(interface {
			WidthExplicit() bool
			HeightExplicit() bool
		}); ok {
			explicitW, explicitH = es.WidthExplicit(), es.HeightExplicit()
		}
		if m, ok := child.(interface {
			PreferredDimensions(interface{}, float32, float32) (float32, float32)
		}); ok {
			measuredW, measuredH := m.PreferredDimensions(child, rect.Width, rect.Height)
			if !explicitH && measuredH > naturalH {
				naturalH = measuredH
			}
			if !explicitW && measuredW > naturalW {
				naturalW = measuredW
			}
		}

		if md, ok := child.(interface{ MinimumDimensions() (float32, float32) }); ok {
			minW, minH := md.MinimumDimensions()
			if naturalW < minW {
				naturalW = minW
			}
			if naturalH < minH {
				naturalH = minH
			}
		}
		info := childInfo{
			child:    child,
			frame:    frame,
			props:    props,
			fillW:    props != nil && props.FillWidth,
			fillH:    props != nil && props.FillHeight,
			naturalH: naturalH,
			naturalW: naturalW,
		}
		infos = append(infos, info)
	}

	totalFixed := float32(0)
	fillCount := 0
	for _, info := range infos {
		if info.fillH {
			fillCount++
		} else {
			totalFixed += info.naturalH
		}
	}

	totalSpacing := l.Spacing * float32(len(infos)-1)
	availableHeight := rect.Height - totalSpacing

	var fillHeight float32
	if fillCount > 0 {
		remaining := availableHeight - totalFixed
		if remaining < 0 {
			remaining = 0
		}
		fillHeight = remaining / float32(fillCount)
		if fillHeight < 30 {
			fillHeight = 30
		}
	}
	// MEUI_VSTACK_TRACE: the inputs to the fill arithmetic and its result. This
	// distinguishes the two possibilities left for the resize fault - the fill
	// child not arriving with FillHeight set (so it is given its measured natural
	// height instead), or this layout being handed a shorter rect than the shim's
	// content rect.
	if os.Getenv("MEUI_VSTACK_TRACE") != "" {
		fmt.Printf("[vstack] rect=%.0fx%.0f spacing=%.0f fixed=%.0f fillCount=%d fillHeight=%.0f children=%d",
			rect.Width, rect.Height, totalSpacing, totalFixed, fillCount, fillHeight, len(infos))
		for _, in := range infos {
			fmt.Printf(" {%T fill=%v nat=%.0f}", in.child, in.fillH, in.naturalH)
		}
		fmt.Printf("\n")
	}

	// Children are positioned in LOCAL coordinate space (starting at 0, 0)
	// The parent Panel handles translation when painting
	y := rect.Y
	for i, info := range infos {
		height := info.naturalH
		if info.fillH && fillCount > 0 {
			height = fillHeight
		}
		// Apply min/max height constraints
		if info.props != nil {
			if info.props.MinimumHeight > 0 && height < info.props.MinimumHeight {
				height = info.props.MinimumHeight
			}
			if info.props.MaximumHeight > 0 && height > info.props.MaximumHeight {
				height = info.props.MaximumHeight
			}
		}

		width := rect.Width
		if !info.fillW {
			width = info.naturalW
		}
		// Apply min/max width constraints
		if info.props != nil {
			if info.props.MinimumWidth > 0 && width < info.props.MinimumWidth {
				width = info.props.MinimumWidth
			}
			if info.props.MaximumWidth > 0 && width > info.props.MaximumWidth {
				width = info.props.MaximumWidth
			}
		}

		x := rect.X
		// Apply horizontal alignment (offsets are relative to rect.X —
		// resetting to 0 here discarded the parent's origin and slammed
		// children against the window edge)
		if info.props != nil {
			switch info.props.Alignment {
			case AlignCenterV, AlignHCenterV:
				x = rect.X + (rect.Width-width)/2
			case AlignRightV:
				x = rect.X + rect.Width - width
			case AlignLeftV:
				x = rect.X
			default:
				x = rect.X
			}
			// Apply margins
			x += info.props.LeftMargin
			width -= info.props.RightMargin
			y += info.props.TopMargin
			height -= info.props.BottomMargin
		}

		newRect := graphics.NewRect(x, y, width, height)
		if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
			align := -1
			if info.props != nil {
				align = info.props.Alignment
			}
			fmt.Printf("[vstack] child=%T frame=(%.1f,%.1f %.1fx%.1f) align=%d rect=(%.1f,%.1f %.1fx%.1f)\n",
				info.child, x, y, width, height, align, rect.X, rect.Y, rect.Width, rect.Height)
		}
		if s, ok := info.child.(interface{ SetFrameRect(graphics.Rect) }); ok {
			s.SetFrameRect(newRect)
		}

		y += height + l.Spacing
		_ = i
	}
}

func (l *VStack) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	children := getChildren(target)
	if len(children) == 0 {
		return 0, 0
	}

	totalHeight := float32(0)
	maxWidth := float32(0)
	for _, child := range children {
		if child == nil {
			continue
		}
		if f, ok := child.(interface{ FrameRect() graphics.Rect }); ok {
			frame := f.FrameRect()
			totalHeight += frame.Height
			if frame.Width > maxWidth {
				maxWidth = frame.Width
			}
		}
	}

	totalHeight += l.Spacing * float32(len(children)-1)
	return maxWidth, totalHeight
}

type childSizer interface {
	FrameRect() graphics.Rect
	SetFrameRect(rect graphics.Rect)
}

func getLayoutProps(widget interface{}) *LayoutProps {
	if p, ok := widget.(interface{ GetLayoutProps() *LayoutProps }); ok {
		return p.GetLayoutProps()
	}
	return nil
}

// getChildren returns the children that participate in layout, which EXCLUDES the
// invisible ones.
//
// An invisible child used to be laid out like any other: SetVisible(false) left it
// occupying its full size and pushed every sibling down. That is how a hidden status bar
// silently reserved 22px at the top of the mobile shell and shifted the whole thing, and
// why collapsing it by height did not help either — the layout measures a child's
// PREFERRED size and takes the larger of that and its frame. The hit test already skips
// invisible children, so this makes the two agree.
func getChildren(target interface{}) []interface{} {
	c, ok := target.(interface{ ChildrenI() []interface{} })
	if !ok {
		return nil
	}
	all := c.ChildrenI()
	visible := all[:0:0] // a fresh slice: never mutate the caller's
	for _, child := range all {
		if v, ok := child.(interface{ Visible() bool }); ok && !v.Visible() {
			continue
		}
		visible = append(visible, child)
	}
	return visible
}

func NewLayoutProps() *LayoutProps {
	return &LayoutProps{
		FillWidth:  true,
		FillHeight: true,
	}
}
