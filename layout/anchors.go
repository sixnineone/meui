package layout

import (
	"github.com/sixnineone/meui/graphics"
)

type AnchorLineType int

const (
	AnchorTop AnchorLineType = iota
	AnchorBottom
	AnchorLeft
	AnchorRight
	AnchorHCenter
	AnchorVCenter
	AnchorBaseline
)

type AnchorLine struct {
	Target Widget
	Line   AnchorLineType
}

type Anchors struct {
	Left, Right, Top, Bottom         *AnchorLine
	HorizontalCenter, VerticalCenter *AnchorLine
	Fill, CenterIn                   Widget
	Margins                          float32
	LeftMargin, RightMargin          float32
	TopMargin, BottomMargin          float32
	HCenterOffset, VCenterOffset     float32
	// Internal: parent object for QML-compliant resolution
	ParentObj interface{} // *qml.Object, stored as interface{} to avoid import cycle
	// Resolver function to get Widget from ParentObj (set by qml package)
	ResolveParentWidget func(obj interface{}) Widget
}

type AnchorLayout struct{}

func NewAnchorLayout() *AnchorLayout {
	return &AnchorLayout{}
}

func NewAnchors() *Anchors {
	return &Anchors{}
}

func (a *AnchorLayout) Layout(target interface{}, rect graphics.Rect) {
	// Skip if rect is too small
	if rect.Width < 10 || rect.Height < 10 {
		return
	}
	children := getChildren(target)
	for _, child := range children {
		if child == nil {
			continue
		}
		if bw, ok := child.(interface{ GetAnchors() *Anchors }); ok {
			anchors := bw.GetAnchors()
			if anchors != nil && HasAnyAnchor(anchors) {
				a.applyAnchors(child, anchors, rect, target)
			}
		}
	}
}

func (a *AnchorLayout) applyAnchors(child interface{}, anchors *Anchors, parentRect graphics.Rect, parent interface{}) {
	// Always use the Go parent (the layout target) as the effective parent
	// This is more reliable than QML parent resolution which can fail
	var effectiveParentWidget Widget
	if pw, ok := parent.(Widget); ok {
		effectiveParentWidget = pw
	}

	// Get effective parent rect — use parentRect directly (already in local coords)
	effectiveParentRect := parentRect

	// Get children of the parent to find the direct QML parent
	parentChildren := getChildren(parent)
	var directParent Widget
	for _, pc := range parentChildren {
		if pc == child {
			// child found in parent - set directParent to the parent passed to applyAnchors
			if dp, ok := parent.(Widget); ok {
				directParent = dp
			}
			break
		}
	}

	// Resolve sentinel values stored at parse time
	// If anchors.Fill was set to the child widget itself (sentinel for "parent"),
	// resolve to either the direct parent or the effective parent
	if anchors.Fill == child {
		if directParent != nil {
			anchors.Fill = directParent
		} else {
			anchors.Fill = effectiveParentWidget
		}
	}
	if anchors.CenterIn == child {
		if directParent != nil {
			anchors.CenterIn = directParent
		} else {
			anchors.CenterIn = effectiveParentWidget
		}
	}
	resolve := func(al *AnchorLine) {
		if al != nil && al.Target == child {
			if directParent != nil {
				al.Target = directParent
			} else {
				al.Target = effectiveParentWidget
			}
		}
	}
	resolve(anchors.Left)
	resolve(anchors.Right)
	resolve(anchors.Top)
	resolve(anchors.Bottom)
	resolve(anchors.HorizontalCenter)
	resolve(anchors.VerticalCenter)

	// Debug: log anchor application AFTER resolution
	frame := graphics.Rect{X: 0, Y: 0, Width: 100, Height: 30}
	if f, ok := child.(interface{ FrameRect() graphics.Rect }); ok {
		frame = f.FrameRect()
	}
	newRect := graphics.NewRect(effectiveParentRect.X, effectiveParentRect.Y, frame.Width, frame.Height)

	if anchors.Fill != nil {
		// Use effectiveParentRect if Fill points to the effective parent,
		// otherwise use the Fill target's frame rect
		fillRect := effectiveParentRect
		if anchors.Fill != effectiveParentWidget {
			if f, ok := anchors.Fill.(interface{ FrameRect() graphics.Rect }); ok {
				targetFrame := f.FrameRect()
				// Use the target's actual frame (with position)
				fillRect = graphics.NewRect(targetFrame.X, targetFrame.Y, targetFrame.Width, targetFrame.Height)
			}
		}
		newRect = fillRect
		if anchors.Margins > 0 {
			newRect = graphics.NewRect(
				fillRect.X+anchors.Margins,
				fillRect.Y+anchors.Margins,
				fillRect.Width-anchors.Margins*2,
				fillRect.Height-anchors.Margins*2,
			)
		}
		if s, ok := child.(interface{ SetFrameRect(graphics.Rect) }); ok {
			s.SetFrameRect(newRect)
		}
		return
	}

	if anchors.CenterIn != nil {
		if f, ok := anchors.CenterIn.(interface{ FrameRect() graphics.Rect }); ok {
			targetFrame := f.FrameRect()
			// Constrain child size to fit within parent
			constrainedWidth := frame.Width
			constrainedHeight := frame.Height
			if constrainedWidth > targetFrame.Width {
				constrainedWidth = targetFrame.Width
			}
			if constrainedHeight > targetFrame.Height {
				constrainedHeight = targetFrame.Height
			}
			newRect.X = effectiveParentRect.X + (targetFrame.Width-constrainedWidth)/2 + anchors.HCenterOffset
			newRect.Y = effectiveParentRect.Y + (targetFrame.Height-constrainedHeight)/2 + anchors.VCenterOffset
			newRect.Width = constrainedWidth
			newRect.Height = constrainedHeight
			if s, ok := child.(interface{ SetFrameRect(graphics.Rect) }); ok {
				s.SetFrameRect(newRect)
			}
			return
		}
	}

	if anchors.HorizontalCenter != nil && anchors.HorizontalCenter.Target != nil {
		if f, ok := anchors.HorizontalCenter.Target.(interface{ FrameRect() graphics.Rect }); ok {
			targetFrame := f.FrameRect()
			// Constrain width to fit within parent
			constrainedWidth := frame.Width
			if constrainedWidth > targetFrame.Width {
				constrainedWidth = targetFrame.Width
			}
			newRect.X = effectiveParentRect.X + (targetFrame.Width-constrainedWidth)/2 + anchors.HCenterOffset
			newRect.Width = constrainedWidth
		}
	}

	if anchors.VerticalCenter != nil && anchors.VerticalCenter.Target != nil {
		if f, ok := anchors.VerticalCenter.Target.(interface{ FrameRect() graphics.Rect }); ok {
			targetFrame := f.FrameRect()
			// Constrain height to fit within parent
			constrainedHeight := frame.Height
			if constrainedHeight > targetFrame.Height {
				constrainedHeight = targetFrame.Height
			}
			newRect.Y = effectiveParentRect.Y + (targetFrame.Height-constrainedHeight)/2 + anchors.VCenterOffset
			newRect.Height = constrainedHeight
		}
	}

	if anchors.Left != nil && anchors.Left.Target != nil {
		newRect.X = effectiveParentRect.X + anchors.LeftMargin
	}

	if anchors.Right != nil && anchors.Right.Target != nil {
		if f, ok := anchors.Right.Target.(interface{ FrameRect() graphics.Rect }); ok {
			targetFrame := f.FrameRect()
			if anchors.Left == nil {
				// Right anchor without Left: position from right edge
				newRect.X = effectiveParentRect.X + targetFrame.Width - newRect.Width - anchors.RightMargin
			} else {
				// Both Left and Right: stretch width
				newWidth := targetFrame.Width - anchors.LeftMargin - anchors.RightMargin
				if newWidth > 0 {
					newRect.Width = newWidth
				}
			}
		}
	}

	if anchors.Top != nil && anchors.Top.Target != nil {
		newRect.Y = effectiveParentRect.Y + anchors.TopMargin
	}

	if anchors.Bottom != nil && anchors.Bottom.Target != nil {
		if f, ok := anchors.Bottom.Target.(interface{ FrameRect() graphics.Rect }); ok {
			targetFrame := f.FrameRect()
			if anchors.Top == nil {
				// Bottom anchor without Top: position from bottom edge
				newRect.Y = effectiveParentRect.Y + targetFrame.Height - newRect.Height - anchors.BottomMargin
			} else {
				// Both Top and Bottom: stretch height
				newHeight := targetFrame.Height - anchors.TopMargin - anchors.BottomMargin
				if newHeight > 0 {
					newRect.Height = newHeight
				}
			}
		}
	}

	if s, ok := child.(interface{ SetFrameRect(graphics.Rect) }); ok {
		s.SetFrameRect(newRect)
	}
}

func (a *AnchorLayout) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	return w, h
}

func HasAnyAnchor(anchors *Anchors) bool {
	if anchors == nil {
		return false
	}
	return anchors.Left != nil || anchors.Right != nil ||
		anchors.Top != nil || anchors.Bottom != nil ||
		anchors.Fill != nil || anchors.CenterIn != nil ||
		anchors.HorizontalCenter != nil || anchors.VerticalCenter != nil
}

func HasAnchorsOnChild(child interface{}) bool {
	if bw, ok := child.(interface{ GetAnchors() *Anchors }); ok {
		return HasAnyAnchor(bw.GetAnchors())
	}
	return false
}

type Widget interface{}
