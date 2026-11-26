package layout

import (
	"github.com/sixnineone/meui/graphics"
)

type Grid struct {
	Columns     int
	Spacing     float32
	layoutDepth int
}

func NewGrid(columns int) *Grid {
	return &Grid{
		Columns: columns,
		Spacing: 8,
	}
}

func (g *Grid) Layout(target interface{}, rect graphics.Rect) {
	// Check for recursive layout
	if checkRecursiveLayout(g.layoutDepth) {
		return
	}
	g.layoutDepth++
	defer func() { g.layoutDepth-- }()

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
	if len(children) == 0 || g.Columns <= 0 {
		return
	}

	// Skip layout if rect is too small (e.g., during window sizing)
	if rect.Width < 10 || rect.Height < 10 {
		return
	}

	// First, apply anchors to children that have them
	anchorLayout := NewAnchorLayout()
	anchorLayout.Layout(target, rect)

	// Calculate number of rows
	rows := (len(children) + g.Columns - 1) / g.Columns

	// Calculate max cell size from children's explicit sizes
	maxCellWidth := float32(0)
	maxCellHeight := float32(0)
	for _, child := range children {
		if child == nil || HasAnchorsOnChild(child) {
			continue
		}
		if f, ok := child.(interface{ FrameRect() graphics.Rect }); ok {
			frame := f.FrameRect()
			if frame.Width > maxCellWidth {
				maxCellWidth = frame.Width
			}
			if frame.Height > maxCellHeight {
				maxCellHeight = frame.Height
			}
		}
	}
	if maxCellWidth < 30 {
		maxCellWidth = 30
	}
	if maxCellHeight < 30 {
		maxCellHeight = 30
	}

	// Use calculated cell sizes, but don't exceed available space
	cellWidth := maxCellWidth
	cellHeight := maxCellHeight

	// If grid would exceed available space, scale down
	totalWidth := float32(g.Columns)*cellWidth + g.Spacing*float32(g.Columns-1)
	totalHeight := float32(rows)*cellHeight + g.Spacing*float32(rows-1)
	if totalWidth > rect.Width && rect.Width > 0 {
		cellWidth = (rect.Width - g.Spacing*float32(g.Columns-1)) / float32(g.Columns)
	}
	if totalHeight > rect.Height && rect.Height > 0 {
		cellHeight = (rect.Height - g.Spacing*float32(rows-1)) / float32(rows)
	}

	// Children are positioned in LOCAL coordinate space (starting at 0, 0)
	// The parent Panel handles translation when painting
	x := rect.X
	y := rect.Y
	col := 0

	for _, child := range children {
		if child == nil {
			continue
		}
		// Skip children that have anchors (already positioned by AnchorLayout)
		if HasAnchorsOnChild(child) {
			continue
		}

		if s, ok := child.(interface{ SetFrameRect(graphics.Rect) }); ok {
			newRect := graphics.NewRect(x, y, cellWidth, cellHeight)
			s.SetFrameRect(newRect)
		}
		col++
		if col >= g.Columns {
			col = 0
			x = float32(0)
			y += cellHeight + g.Spacing
		} else {
			x += cellWidth + g.Spacing
		}
	}
}

func (g *Grid) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	children := getChildren(target)
	if len(children) == 0 || g.Columns <= 0 {
		return 0, 0
	}

	rows := (len(children) + g.Columns - 1) / g.Columns

	// Calculate max cell size from children's explicit sizes
	maxCellWidth := float32(0)
	maxCellHeight := float32(0)
	for _, child := range children {
		if child == nil {
			continue
		}
		if f, ok := child.(interface{ FrameRect() graphics.Rect }); ok {
			frame := f.FrameRect()
			if frame.Width > maxCellWidth {
				maxCellWidth = frame.Width
			}
			if frame.Height > maxCellHeight {
				maxCellHeight = frame.Height
			}
		}
	}
	if maxCellWidth < 30 {
		maxCellWidth = 30
	}
	if maxCellHeight < 30 {
		maxCellHeight = 30
	}

	totalWidth := float32(g.Columns)*maxCellWidth + g.Spacing*float32(g.Columns-1)
	totalHeight := float32(rows)*maxCellHeight + g.Spacing*float32(rows-1)

	return totalWidth, totalHeight
}
