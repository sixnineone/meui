package layout

import (
	"github.com/sixnineone/meui/graphics"
)

type FlexDirection string

const (
	FlexRow    FlexDirection = "row"
	FlexColumn FlexDirection = "column"
)

type JustifyContent string

const (
	JustifyStart        JustifyContent = "start"
	JustifyCenter       JustifyContent = "center"
	JustifyEnd          JustifyContent = "end"
	JustifySpaceBetween JustifyContent = "space-between"
	JustifySpaceAround  JustifyContent = "space-around"
)

type AlignItems string

const (
	AlignStretch AlignItems = "stretch"
	AlignStart   AlignItems = "start"
	AlignCenter  AlignItems = "center"
	AlignEnd     AlignItems = "end"
)

type Flex struct {
	Direction   FlexDirection
	Justify     JustifyContent
	Align       AlignItems
	Spacing     float32
	Wrap        bool
	layoutDepth int
}

func NewFlex() *Flex {
	return &Flex{
		Direction: FlexRow,
		Justify:   JustifyStart,
		Align:     AlignStretch,
		Spacing:   8,
	}
}

func (f *Flex) Layout(target interface{}, rect graphics.Rect) {
	// Check for recursive layout
	if checkRecursiveLayout(f.layoutDepth) {
		return
	}
	f.layoutDepth++
	defer func() { f.layoutDepth-- }()

	// Do NOT resize the target — the rect is the panel's padded inner rect
	// and the children are laid out inside it with the same padding offset.
	// See the matching note in VStack.Layout (double-padding overflow).

	children := getChildren(target)
	if len(children) == 0 {
		return
	}

	// First, apply anchors to children that have them
	anchorLayout := NewAnchorLayout()
	anchorLayout.Layout(target, rect)

	if f.Direction == FlexRow {
		f.layoutRow(target, rect)
	} else {
		f.layoutColumn(target, rect)
	}
}

func (f *Flex) layoutRow(target interface{}, rect graphics.Rect) {
	children := getChildren(target)
	if len(children) == 0 {
		return
	}

	totalChildWidth := float32(0)
	maxHeight := float32(0)
	for _, child := range children {
		if child == nil {
			continue
		}
		// Skip anchored children (already positioned by AnchorLayout)
		if HasAnchorsOnChild(child) {
			continue
		}
		if frame, ok := child.(interface{ FrameRect() graphics.Rect }); ok {
			fr := frame.FrameRect()
			totalChildWidth += fr.Width
			if fr.Height > maxHeight {
				maxHeight = fr.Height
			}
		}
	}

	totalSpacing := f.Spacing * float32(len(children)-1)
	availableWidth := rect.Width - totalSpacing

	justifyOffset := float32(0)
	if f.Justify == JustifyCenter {
		justifyOffset = (availableWidth - totalChildWidth) / 2
	} else if f.Justify == JustifyEnd {
		justifyOffset = availableWidth - totalChildWidth
	}

	// Children are positioned in LOCAL coordinate space (starting at 0, 0)
	// The parent Panel handles translation when painting
	x := rect.X + justifyOffset
	y := rect.Y

	for _, child := range children {
		if child == nil {
			continue
		}
		// Skip anchored children (already positioned by AnchorLayout)
		if HasAnchorsOnChild(child) {
			continue
		}
		if s, ok := child.(interface{ SetFrameRect(graphics.Rect) }); ok {
			if frame, ok := child.(interface{ FrameRect() graphics.Rect }); ok {
				fr := frame.FrameRect()
				childHeight := maxHeight
				if f.Align == AlignStretch {
					childHeight = rect.Height
				}
				newRect := graphics.NewRect(x, y, fr.Width, childHeight)
				s.SetFrameRect(newRect)
				x += fr.Width + f.Spacing
			}
		}
	}
}

func (f *Flex) layoutColumn(target interface{}, rect graphics.Rect) {
	children := getChildren(target)
	if len(children) == 0 {
		return
	}

	totalChildHeight := float32(0)
	maxWidth := float32(0)
	for _, child := range children {
		if child == nil {
			continue
		}
		// Skip anchored children (already positioned by AnchorLayout)
		if HasAnchorsOnChild(child) {
			continue
		}
		if frame, ok := child.(interface{ FrameRect() graphics.Rect }); ok {
			fr := frame.FrameRect()
			totalChildHeight += fr.Height
			if fr.Width > maxWidth {
				maxWidth = fr.Width
			}
		}
	}

	totalSpacing := f.Spacing * float32(len(children)-1)
	availableHeight := rect.Height - totalSpacing

	justifyOffset := float32(0)
	if f.Justify == JustifyCenter {
		justifyOffset = (availableHeight - totalChildHeight) / 2
	} else if f.Justify == JustifyEnd {
		justifyOffset = availableHeight - totalChildHeight
	}

	// Children are positioned in LOCAL coordinate space (starting at 0, 0)
	// The parent Panel handles translation when painting
	x := rect.X
	y := rect.Y + justifyOffset

	for _, child := range children {
		if child == nil {
			continue
		}
		// Skip anchored children (already positioned by AnchorLayout)
		if HasAnchorsOnChild(child) {
			continue
		}
		if s, ok := child.(interface{ SetFrameRect(graphics.Rect) }); ok {
			if frame, ok := child.(interface{ FrameRect() graphics.Rect }); ok {
				fr := frame.FrameRect()
				childWidth := maxWidth
				if f.Align == AlignStretch {
					childWidth = rect.Width
				}
				newRect := graphics.NewRect(x, y, childWidth, fr.Height)
				s.SetFrameRect(newRect)
				y += fr.Height + f.Spacing
			}
		}
	}
}

func (f *Flex) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	children := getChildren(target)
	if len(children) == 0 {
		return 0, 0
	}

	if f.Direction == FlexRow {
		totalWidth := float32(0)
		maxHeight := float32(0)
		for _, child := range children {
			if child == nil {
				continue
			}
			if frame, ok := child.(interface{ FrameRect() graphics.Rect }); ok {
				f := frame.FrameRect()
				totalWidth += f.Width
				if f.Height > maxHeight {
					maxHeight = f.Height
				}
			}
		}
		totalWidth += f.Spacing * float32(len(children)-1)
		return totalWidth, maxHeight
	} else {
		totalHeight := float32(0)
		maxWidth := float32(0)
		for _, child := range children {
			if child == nil {
				continue
			}
			if frame, ok := child.(interface{ FrameRect() graphics.Rect }); ok {
				f := frame.FrameRect()
				totalHeight += f.Height
				if f.Width > maxWidth {
					maxWidth = f.Width
				}
			}
		}
		totalHeight += f.Spacing * float32(len(children)-1)
		return maxWidth, totalHeight
	}
}
