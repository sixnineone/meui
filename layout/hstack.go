package layout

import (
	"fmt"
	"os"

	"github.com/sixnineone/meui/graphics"
)

type HStack struct {
	Spacing     float32
	layoutDepth int
}

func NewHStack() *HStack {
	return &HStack{
		Spacing: 8,
	}
}

func (h *HStack) Layout(target interface{}, rect graphics.Rect) {
	// Check for recursive layout
	if checkRecursiveLayout(h.layoutDepth) {
		return
	}
	h.layoutDepth++
	defer func() { h.layoutDepth-- }()

	// Update target's frame size but preserve existing position
	// The position was set by parent layout (AnchorLayout or other)
	if setter, ok := target.(interface{ SetFrameRect(graphics.Rect) }); ok {
		if getter, ok := target.(interface{ FrameRect() graphics.Rect }); ok {
			currentFrame := getter.FrameRect()
			newFrame := graphics.NewRect(currentFrame.X, currentFrame.Y, rect.Width, rect.Height)
			setter.SetFrameRect(newFrame)
		} else {
			setter.SetFrameRect(rect)
		}
	}

	children := getChildren(target)
	if len(children) == 0 {
		return
	}

	// First, apply anchors to children that have them
	anchorLayout := NewAnchorLayout()
	anchorLayout.Layout(target, rect)

	type childInfo struct {
		child  interface{}
		frame  graphics.Rect
		props  *LayoutProps
		width  float32
		height float32
		fillW  bool
		fillH  bool
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
		// Measure child for preferred height/width
		naturalH := frame.Height
		naturalW := frame.Width
		// Explicit sizes win over measured preferences (fixed-size
		// wrapping labels must keep their set width).
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
			child:  child,
			frame:  frame,
			props:  props,
			width:  naturalW,
			height: naturalH,
			fillW:  props != nil && props.FillWidth,
			fillH:  props != nil && props.FillHeight,
		}
		infos = append(infos, info)
	}

	// Calculate total fixed width and count fill children
	totalFixed := float32(0)
	fillCount := 0
	for _, info := range infos {
		if info.fillW {
			fillCount++
		} else {
			totalFixed += info.width
		}
	}

	totalSpacing := h.Spacing * float32(len(infos)-1)
	availableWidth := rect.Width - totalSpacing

	var fillWidth float32
	if fillCount > 0 {
		remaining := availableWidth - totalFixed
		if remaining < 0 {
			remaining = 0
		}
		fillWidth = remaining / float32(fillCount)
	}

	// Children are positioned in LOCAL coordinate space (starting at 0, 0)
	// The parent Panel handles translation when painting
	x := rect.X
	for _, info := range infos {
		width := info.width
		if info.fillW && fillCount > 0 {
			width = fillWidth
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

		height := info.height
		if info.fillH {
			height = rect.Height
		}
		y := rect.Y
		// Apply vertical alignment (offsets are relative to rect.Y —
		// resetting to 0 here discarded the parent's origin)
		if info.props != nil {
			switch info.props.Alignment {
			case AlignCenterV, AlignVCenterV:
				y = rect.Y + (rect.Height-height)/2
			case AlignBottomV:
				y = rect.Y + rect.Height - height
			case AlignTopV:
				y = rect.Y
			default:
				y = rect.Y
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
			fmt.Printf("[hstack] child=%T frame=(%.1f,%.1f %.1fx%.1f) align=%d rect=(%.1f,%.1f %.1fx%.1f)\n",
				info.child, x, y, width, height, align, rect.X, rect.Y, rect.Width, rect.Height)
		}
		if s, ok := info.child.(interface{ SetFrameRect(graphics.Rect) }); ok {
			s.SetFrameRect(newRect)
		}

		x += width + h.Spacing
	}
}

func (h *HStack) PreferredDimensions(target interface{}, w, hVal float32) (float32, float32) {
	children := getChildren(target)
	if len(children) == 0 {
		return 0, 0
	}

	totalWidth := float32(0)
	maxHeight := float32(0)
	for _, child := range children {
		if child == nil {
			continue
		}
		if f, ok := child.(interface{ FrameRect() graphics.Rect }); ok {
			frame := f.FrameRect()
			totalWidth += frame.Width
			if frame.Height > maxHeight {
				maxHeight = frame.Height
			}
		}
	}

	totalWidth += h.Spacing * float32(len(children)-1)
	return totalWidth, maxHeight
}
