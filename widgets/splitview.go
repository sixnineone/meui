package widgets

import (
	"strings"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

type SplitView struct {
	meui.BaseContainer
	*theme.StackPanelTheme
	Orientation   int // 0=horizontal, 1=vertical
	SplitPosition float32
	MinSize1      float32
	MinSize2      float32
	HandleSize    float32
	Panel1        meui.Widget
	Panel2        meui.Widget
	OnSplitMoved  func(position float32)
	dragging      bool
	dragStart     float32
	splitStart    float32
	// hovering is true while the pointer is on the handle: the divider takes
	// the theme's hover ink then, so the splitter shows it is grabbable
	// without a permanently lit band.
	hovering bool
	// hoverAlways bool: explicit override of the theme's HoverColorAlwaysOn
	// (SetHoverColorAlwaysOn), so one split can pin the highlight while the
	// rest of the app follows the theme.
	hoverAlways    bool
	hoverAlwaysSet bool
}

func NewSplitView() *SplitView {
	return &SplitView{
		BaseContainer:   meui.BaseContainer{BaseWidget: *meui.NewBaseWidget()},
		StackPanelTheme: theme.DefaultStackPanelTheme,
		HandleSize:      6,
		SplitPosition:   0.5,
		MinSize1:        50,
		MinSize2:        50,
	}
}

func (s *SplitView) SetFrameRect(rect graphics.Rect) {
	s.BaseWidget.SetFrameRect(rect)
	s.SetNeedsLayout()
}

func (s *SplitView) SetPanels(p1, p2 meui.Widget) *SplitView {
	s.Panel1 = p1
	s.Panel2 = p2
	if p1 != nil {
		p1.SetParent(s)
	}
	if p2 != nil {
		p2.SetParent(s)
	}
	s.SetNeedsLayout()
	s.MarkForRedraw()
	return s
}

func (s *SplitView) AddChild(child meui.Widget) {
	if s.Panel1 == nil {
		s.Panel1 = child
	} else if s.Panel2 == nil {
		s.Panel2 = child
	}
	if child != nil {
		child.SetParent(s)
	}
	s.SetNeedsLayout()
	s.MarkForRedraw()
}

func (s *SplitView) SetSplitPosition(pos float32) *SplitView {
	if pos < 0 {
		pos = 0
	}
	if pos > 1 {
		pos = 1
	}
	s.SplitPosition = pos
	s.SetNeedsLayout()
	s.MarkForRedraw()
	return s
}

// SetHandleSize sets the splitter handle thickness in pixels.
func (s *SplitView) SetHandleSize(size float32) *SplitView {
	if size < 1 {
		size = 1
	}
	s.HandleSize = size
	s.MarkForRedraw()
	return s
}

// SetOnSplitMoved registers a callback fired when the user finishes dragging
// the splitter (like QSplitter.splitterMoved).
func (s *SplitView) SetOnSplitMoved(handler func(position float32)) *SplitView {
	s.OnSplitMoved = handler
	return s
}

func (s *SplitView) SetOrientation(orient int) *SplitView {
	s.Orientation = orient
	s.SetNeedsLayout()
	s.MarkForRedraw()
	return s
}

func (s *SplitView) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !s.Visible() {
		return
	}

	canvas.Save()

	if s.NeedsLayout() {
		frame := s.FrameRect()
		if frame.Width >= 50 && frame.Height >= 50 {
			s.ValidateLayout()
		}
	}

	// Paint each panel at its own origin (canvas translated so children use local coords)
	for _, p := range []meui.Widget{s.Panel1, s.Panel2} {
		if p != nil {
			pf := p.FrameRect()
			canvas.Save()
			canvas.Translate(pf.X, pf.Y)
			p.Paint(canvas, graphics.NewRect(0, 0, pf.Width, pf.Height))
			canvas.Restore()
		}
	}

	// The painted band IS the band the pointer grabs, the cursor reports and
	// PanelAt hits, so the line always sits under the resize cursor.
	paintSplitter(canvas, s.handleBand(rect), s.Orientation == 0,
		s.hovering || s.dragging, s.splitterTheme())

	canvas.Restore()
}

// onHandle reports whether a widget-local point is on the splitter's grab
// band — the same band OnMouseDown drags from, the cursor uses and the hover
// ink tracks.
func (s *SplitView) onHandle(x, y float32) bool {
	frame := s.FrameRect()
	if s.Orientation == 0 {
		handleX := frame.Width*s.SplitPosition - s.HandleSize/2
		return x >= handleX-4 && x <= handleX+s.HandleSize+4
	}
	handleY := frame.Height*s.SplitPosition - s.HandleSize/2
	return y >= handleY-4 && y <= handleY+s.HandleSize+4
}

// SetHoverColorAlwaysOn pins whether this split's divider is always drawn in
// the hover ink, overriding the theme's HoverColorAlwaysOn.
func (s *SplitView) SetHoverColorAlwaysOn(on bool) *SplitView {
	s.hoverAlways, s.hoverAlwaysSet = on, true
	s.MarkForRedraw()
	return s
}

// splitterTheme resolves the splitter theme with this widget's override
// applied: an explicit SetHoverColorAlwaysOn wins over the theme's token.
// It returns a copy only when it has to override, so the shared theme is
// never mutated from a widget.
func (s *SplitView) splitterTheme() *theme.StackPanelTheme {
	t := s.StackPanelTheme
	if t == nil {
		t = theme.DefaultStackPanelTheme
	}
	if !s.hoverAlwaysSet || t == nil || t.HoverColorAlwaysOn == s.hoverAlways {
		return t
	}
	c := t.Copy()
	c.HoverColorAlwaysOn = s.hoverAlways
	return c
}

// handleBand returns the splitter's grab band for a paint rect — the one
// band the painting, the drag, the hover ink and the resize cursor all use.
func (s *SplitView) handleBand(rect graphics.Rect) graphics.Rect {
	if s.Orientation == 0 {
		return graphics.NewRect(rect.X+rect.Width*s.SplitPosition-s.HandleSize/2,
			rect.Y, s.HandleSize, rect.Height)
	}
	return graphics.NewRect(rect.X, rect.Y+rect.Height*s.SplitPosition-s.HandleSize/2,
		rect.Width, s.HandleSize)
}

// setHovering records the handle's hot state and repaints only on a change.
func (s *SplitView) setHovering(on bool) {
	if on == s.hovering {
		return
	}
	s.hovering = on
	s.MarkForRedraw()
}

// HoverAt implements meui.HoverTarget. The window reports the pointer
// position (widget-local) on every move along the hit chain, because the
// deepest hit is normally a child of one of the panels — OnMouseMove alone
// therefore saw the pointer arrive on the handle and never saw it leave,
// which left the divider stuck in its hover ink.
func (s *SplitView) HoverAt(x, y float32) {
	if !s.Enabled() || !s.Visible() {
		s.setHovering(false)
		return
	}
	s.setHovering(s.onHandle(x, y))
}

// CursorAt reports the resize cursor while the pointer is on the splitter
// handle, using the same zone OnMouseDown starts a drag from.
func (s *SplitView) CursorAt(x, y float32) *platform.Cursor {
	if !s.Enabled() || !s.Visible() {
		return nil
	}
	if !s.onHandle(x, y) {
		return nil
	}
	if s.Orientation == 0 {
		return platform.ResizeHorizontalCursor()
	}
	return platform.ResizeVerticalCursor()
}

func (s *SplitView) OnMouseDown(x, y float32, button int) bool {
	if !s.Enabled() {
		return false
	}

	frame := s.FrameRect()
	if s.Orientation == 0 {
		handleX := frame.Width*s.SplitPosition - s.HandleSize/2
		if x >= handleX-4 && x <= handleX+s.HandleSize+4 {
			s.dragging = true
			s.dragStart = x
			s.splitStart = s.SplitPosition
			return true
		}
	} else {
		handleY := frame.Height*s.SplitPosition - s.HandleSize/2
		if y >= handleY-4 && y <= handleY+s.HandleSize+4 {
			s.dragging = true
			s.dragStart = y
			s.splitStart = s.SplitPosition
			return true
		}
	}

	if s.Panel1 != nil {
		p1Frame := s.Panel1.FrameRect()
		localX := x - p1Frame.X
		localY := y - p1Frame.Y
		if localX >= 0 && localX <= p1Frame.Width && localY >= 0 && localY <= p1Frame.Height {
			if el, ok := s.Panel1.(interface {
				OnMouseDown(float32, float32, int) bool
			}); ok {
				return el.OnMouseDown(localX, localY, button)
			}
		}
	}
	if s.Panel2 != nil {
		p2Frame := s.Panel2.FrameRect()
		localX := x - p2Frame.X
		localY := y - p2Frame.Y
		if localX >= 0 && localX <= p2Frame.Width && localY >= 0 && localY <= p2Frame.Height {
			if el, ok := s.Panel2.(interface {
				OnMouseDown(float32, float32, int) bool
			}); ok {
				return el.OnMouseDown(localX, localY, button)
			}
		}
	}
	return false
}

func (s *SplitView) OnMouseMove(x, y float32) {
	if !s.dragging {
		// Track the pointer so the divider can show its hover ink.
		s.setHovering(s.onHandle(x, y))
	}
	if s.dragging {
		frame := s.FrameRect()
		if s.Orientation == 0 {
			delta := x - s.dragStart
			newPos := s.splitStart + delta/frame.Width
			if newPos < s.MinSize1/frame.Width {
				newPos = s.MinSize1 / frame.Width
			}
			if newPos > 1-s.MinSize2/frame.Width {
				newPos = 1 - s.MinSize2/frame.Width
			}
			s.SplitPosition = newPos
		} else {
			delta := y - s.dragStart
			newPos := s.splitStart + delta/frame.Height
			if newPos < s.MinSize1/frame.Height {
				newPos = s.MinSize1 / frame.Height
			}
			if newPos > 1-s.MinSize2/frame.Height {
				newPos = 1 - s.MinSize2/frame.Height
			}
			s.SplitPosition = newPos
		}
		s.SetNeedsLayout()
		s.MarkForRedraw()
		return
	}

	frame := s.FrameRect()
	if s.Orientation == 0 {
		handleX := frame.Width*s.SplitPosition - s.HandleSize/2
		if x >= handleX-4 && x <= handleX+s.HandleSize+4 {
			return
		}
	} else {
		handleY := frame.Height*s.SplitPosition - s.HandleSize/2
		if y >= handleY-4 && y <= handleY+s.HandleSize+4 {
			return
		}
	}

	if s.Panel1 != nil {
		p1Frame := s.Panel1.FrameRect()
		localX := x - p1Frame.X
		localY := y - p1Frame.Y
		if localX >= 0 && localX <= p1Frame.Width && localY >= 0 && localY <= p1Frame.Height {
			if el, ok := s.Panel1.(interface{ OnMouseMove(float32, float32) }); ok {
				el.OnMouseMove(localX, localY)
				return
			}
		}
	}
	if s.Panel2 != nil {
		p2Frame := s.Panel2.FrameRect()
		localX := x - p2Frame.X
		localY := y - p2Frame.Y
		if localX >= 0 && localX <= p2Frame.Width && localY >= 0 && localY <= p2Frame.Height {
			if el, ok := s.Panel2.(interface{ OnMouseMove(float32, float32) }); ok {
				el.OnMouseMove(localX, localY)
			}
		}
	}
}

func (s *SplitView) OnMouseUp(x, y float32, button int) bool {
	if s.dragging {
		s.dragging = false
		if s.OnSplitMoved != nil {
			s.OnSplitMoved(s.SplitPosition)
		}
	}
	return false
}

func (s *SplitView) ValidateLayout() {
	if !s.NeedsLayout() {
		return
	}
	frame := s.FrameRect()
	if frame.Width < 10 || frame.Height < 10 {
		s.ClearNeedsLayout()
		return
	}

	if s.Orientation == 0 {
		splitX := frame.Width * s.SplitPosition
		halfHandle := s.HandleSize / 2
		// Clamp so panels don't overlap
		p1w := splitX - halfHandle
		if p1w < s.MinSize1 {
			p1w = s.MinSize1
		}
		p2w := frame.Width - splitX - halfHandle
		if p2w < s.MinSize2 {
			p2w = s.MinSize2
		}
		// If total exceeds frame, give each proportional to their min
		if p1w+p2w > frame.Width {
			totalMin := s.MinSize1 + s.MinSize2
			if totalMin > 0 {
				p1w = s.MinSize1 + (frame.Width-totalMin)*s.MinSize1/totalMin
				p2w = frame.Width - p1w
			} else {
				p1w = frame.Width / 2
				p2w = frame.Width - p1w
			}
		}
		if s.Panel1 != nil {
			s.Panel1.SetFrameRect(graphics.NewRect(0, 0, p1w, frame.Height))
		}
		if s.Panel2 != nil {
			s.Panel2.SetFrameRect(graphics.NewRect(splitX+halfHandle, 0, p2w, frame.Height))
		}
	} else {
		splitY := frame.Height * s.SplitPosition
		halfHandle := s.HandleSize / 2
		p1h := splitY - halfHandle
		if p1h < s.MinSize1 {
			p1h = s.MinSize1
		}
		p2h := frame.Height - splitY - halfHandle
		if p2h < s.MinSize2 {
			p2h = s.MinSize2
		}
		if p1h+p2h > frame.Height {
			totalMin := s.MinSize1 + s.MinSize2
			if totalMin > 0 {
				p1h = s.MinSize1 + (frame.Height-totalMin)*s.MinSize1/totalMin
				p2h = frame.Height - p1h
			} else {
				p1h = frame.Height / 2
				p2h = frame.Height - p1h
			}
		}
		if s.Panel1 != nil {
			s.Panel1.SetFrameRect(graphics.NewRect(0, 0, frame.Width, p1h))
		}
		if s.Panel2 != nil {
			s.Panel2.SetFrameRect(graphics.NewRect(0, splitY+halfHandle, frame.Width, p2h))
		}
	}

	if s.Panel1 != nil {
		if vc, ok := s.Panel1.(interface{ ValidateLayout() }); ok {
			vc.ValidateLayout()
		}
	}
	if s.Panel2 != nil {
		if vc, ok := s.Panel2.(interface{ ValidateLayout() }); ok {
			vc.ValidateLayout()
		}
	}

	s.ClearNeedsLayout()
}

func (s *SplitView) Children() []meui.Widget {
	var children []meui.Widget
	if s.Panel1 != nil {
		children = append(children, s.Panel1)
	}
	if s.Panel2 != nil {
		children = append(children, s.Panel2)
	}
	return children
}

func (s *SplitView) ChildrenI() []interface{} {
	var children []interface{}
	if s.Panel1 != nil {
		children = append(children, s.Panel1)
	}
	if s.Panel2 != nil {
		children = append(children, s.Panel2)
	}
	return children
}

func (s *SplitView) PanelAt(x, y float32) meui.Widget {
	frame := s.FrameRect()
	// The handle is checked FIRST, and over the same band OnMouseDown starts a
	// drag from: a child's frame abuts the handle, so testing the panels first
	// handed the grab zone to them — the splitter then received neither the
	// press (no drag) nor the resize cursor.
	if s.Orientation == 0 {
		handleX := frame.Width*s.SplitPosition - s.HandleSize/2
		if x >= handleX-4 && x <= handleX+s.HandleSize+4 && y >= 0 && y <= frame.Height {
			return s
		}
	} else {
		handleY := frame.Height*s.SplitPosition - s.HandleSize/2
		if y >= handleY-4 && y <= handleY+s.HandleSize+4 && x >= 0 && x <= frame.Width {
			return s
		}
	}
	if s.Panel1 != nil {
		p1Frame := s.Panel1.FrameRect()
		localX := x - p1Frame.X
		localY := y - p1Frame.Y
		if localX >= 0 && localX <= p1Frame.Width && localY >= 0 && localY <= p1Frame.Height {
			if container, ok := s.Panel1.(meui.Container); ok {
				if deeper := container.PanelAt(localX, localY); deeper != nil {
					return deeper
				}
			}
			return s.Panel1
		}
	}
	if s.Panel2 != nil {
		p2Frame := s.Panel2.FrameRect()
		localX := x - p2Frame.X
		localY := y - p2Frame.Y
		if localX >= 0 && localX <= p2Frame.Width && localY >= 0 && localY <= p2Frame.Height {
			if container, ok := s.Panel2.(meui.Container); ok {
				if deeper := container.PanelAt(localX, localY); deeper != nil {
					return deeper
				}
			}
			return s.Panel2
		}
	}
	return s
}

func (s *SplitView) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "orientation":
		if f, ok := value.(float64); ok {
			s.SetOrientation(int(f))
		} else if str, ok := value.(string); ok {
			switch strings.ToLower(str) {
			case "vertical":
				s.SetOrientation(1)
			default:
				s.SetOrientation(0)
			}
		}
	case "splitPosition", "position":
		if f, ok := value.(float64); ok {
			s.SetSplitPosition(float32(f))
		} else if f, ok := value.(int); ok {
			s.SetSplitPosition(float32(f))
		}
	case "handleSize":
		if f, ok := value.(float64); ok {
			s.HandleSize = float32(f)
		} else if f, ok := value.(int); ok {
			s.HandleSize = float32(f)
		}
	}
}
