package meui

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/layout"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

type Widget interface {
	Paint(canvas *graphics.Canvas, rect graphics.Rect)
	ContainsPoint(x, y float32) bool
	FrameRect() graphics.Rect
	SetFrameRect(rect graphics.Rect)
	MarkForRedraw()
	SetParent(parent Widget)
	Parent() Widget
	PreferredDimensions(target interface{}, width, height float32) (float32, float32)
	MinimumDimensions() (float32, float32)
}

// Container is an interface for widgets that can contain children.
type Container interface {
	Widget
	Children() []Widget
	PanelAt(x, y float32) Widget
	ValidateLayout()
}

// BaseContainer is a struct for widgets that have children.
// Embeds BaseWidget and adds child management + event dispatch helpers.
// Widgets that need children should embed this instead of BaseWidget.
type BaseContainer struct {
	// orderCache is orderedChildren's result while any child has a z; cleared when a
	// z changes.
	orderCache []Widget
	BaseWidget
	children []Widget
}

type EventListener interface {
	OnMouseDown(x, y float32, button int) bool
	OnMouseUp(x, y float32, button int) bool
	OnMouseMove(x, y float32)
}

type EventHandler interface {
	OnMouseDown(x, y float32, button int) bool
	OnMouseUp(x, y float32, button int) bool
	OnMouseMove(x, y float32)
}

type ClickHandler interface {
	OnClick(x, y float32)
}

type FocusHandler interface {
	OnFocus()
	OnBlur()
}

type TabIndexSetter interface {
	SetTabIndex(int)
	TabIndex() int
	Focusable() bool
}

type HoverHandler interface {
	SetHoveredState(bool)
}

// HoverTarget is for a widget that owns a pointer position inside its own
// area without being the deepest hit there — a SplitView's handle sits above
// its panels, but everywhere else the panels' children are the hit. The
// window calls HoverAt on every widget along the hit chain (the deepest hit,
// then its parents) with **widget-local** coordinates on each pointer move,
// so such a widget learns both when the pointer arrives and when it leaves.
type HoverTarget interface {
	HoverAt(x, y float32)
}

type ScrollHandler interface {
	OnMouseWheel(x, y, dx, dy float32) bool
}

// CursorProvider is an optional interface for widgets that want to
// specify a custom mouse cursor when the pointer is over them.
type CursorProvider interface {
	CursorAt(x, y float32) *platform.Cursor
}

type BaseWidget struct {
	// zIndex orders this widget among its SIBLINGS: paint order and hit-test order
	// (higher is on top). QML's `z:` sets it. Zero is the default and preserves
	// insertion order, which is why no existing widget's behaviour changes.
	zIndex         float32
	x, y           float32
	width, height  float32
	enabled        bool
	visible        bool
	needsRedraw    bool
	focused        bool
	hovered        bool
	needsLayout    bool
	parent         Widget
	layoutProps    *layout.LayoutProps
	anchors        *layout.Anchors
	tabIndex       int
	widthExplicit  bool
	heightExplicit bool

	onMouseDown    func(x, y float32, button int)
	onMouseUp      func(x, y float32, button int)
	onMouseMove    func(x, y float32)
	onMouseEnter   func()
	onMouseExit    func()
	onKeyDown      func(keyCode platform.KeyCode, modifiers platform.Modifiers)
	onClick        func()
	debugName      string
	treePath       string
	tooltipText    string
	tooltipDelay   time.Duration
	tooltipTimer   *time.Timer
	tooltipVisible bool
	tooltipMouseX  float32
	tooltipMouseY  float32
}

func NewBaseWidget() *BaseWidget {
	return &BaseWidget{
		width:    50,
		height:   20,
		enabled:  true,
		visible:  true,
		tabIndex: -1,
	}
}

// ── Coordinate spaces (the system-wide contract) ─────────────────────
//
//	FRAME space — parent-relative. FrameRect()/SetFrameRect() store a
//	widget's position in its PARENT's local space plus its size. Only a
//	parent may interpret the origin: PaintChildren translates the canvas
//	by the child's frame origin, containers hit-test children by their
//	frames, and layout managers write frames in the parent's space.
//
//	LOCAL space — widget-relative, origin (0,0) at the widget's own
//	top-left. Everything a widget does about itself lives here:
//	Paint(canvas, rect) receives rect = LocalRect(), and mouse events
//	arrive as OnMouseDown(x, y) in local coordinates.
//
// A widget reading its OWN FrameRect() for X/Y inside Paint, hit-testing
// or event handling is mixing the two spaces (the historic scrollbar
// double-offset: bars laid out at FrameRect() were shifted by the
// widget's position inside its parent). Use LocalRect() instead; the API
// takes sizes wherever only the extent matters so the space cannot be
// passed wrongly (see ScrollbarHost.LayoutBars).
//
// FrameRect returns the widget's frame in the PARENT's local space plus
// its size. Use LocalRect() for the widget's own (0,0)-based rect.
func (w *BaseWidget) FrameRect() graphics.Rect {
	return graphics.NewRect(w.x, w.y, w.width, w.height)
}

// LocalRect returns the widget's own rect in LOCAL space: origin (0,0),
// size equal to FrameRect().Size(). This is the rect Paint receives and
// the space all widget-local geometry (hit tests, bar layout, clip
// rects) must use.
func (w *BaseWidget) LocalRect() graphics.Rect {
	return graphics.NewRect(0, 0, w.width, w.height)
}

// GlobalFrameRect is deprecated and returns FrameRect().
// FrameRect() returns coordinates in parent-local space, not absolute.
func (w *BaseWidget) GlobalFrameRect() graphics.Rect {
	return w.FrameRect()
}

func (w *BaseWidget) Width() float32 {
	return w.width
}

func (w *BaseWidget) Height() float32 {
	return w.height
}

func (w *BaseWidget) SetFrameRect(rect graphics.Rect) {
	// A SIZE change invalidates the widget's own layout: gated
	// ValidateLayout implementations (TabPanel et al.) would otherwise
	// early-return after a window resize and keep children laid out at
	// the old size (content clipping on maximize). Position-only moves
	// do not need re-layout. Redraw is deliberately NOT triggered here
	// (the parent paint cycle handles it — avoids resize loops).
	if rect.Width != w.width || rect.Height != w.height {
		w.needsLayout = true
	}
	w.x = rect.X
	w.y = rect.Y
	w.width = rect.Width
	w.height = rect.Height
	// Do NOT call MarkForRedraw here to avoid infinite loops during resize/paint.
	// Redraw is triggered by the parent's Paint cycle or explicit MarkForRedraw calls.
}

// SetFrameRectDirect sets the frame rect without triggering a redraw.
// Used during layout/resizing to avoid infinite redraw loops. Like
// SetFrameRect, a SIZE change marks the widget for re-layout.
func (w *BaseWidget) SetFrameRectDirect(rect graphics.Rect) {
	if rect.Width != w.width || rect.Height != w.height {
		w.needsLayout = true
	}
	w.x = rect.X
	w.y = rect.Y
	w.width = rect.Width
	w.height = rect.Height
}

func (w *BaseWidget) SetX(x float32) *BaseWidget {
	w.x = x
	w.MarkForRedraw()
	return w
}

func (w *BaseWidget) SetY(y float32) *BaseWidget {
	w.y = y
	w.MarkForRedraw()
	return w
}

func (w *BaseWidget) SetWidth(width float32) *BaseWidget {
	w.width = width
	w.MarkForRedraw()
	return w
}

func (w *BaseWidget) SetWidthDirect(width float32) {
	w.widthExplicit = width > 0
	w.width = width
	// Don't call MarkForRedraw() - used during construction
}

// WidthExplicit reports whether the width came from an explicit author-chosen
// size (QML `width`) rather than a constructor default or layout pass.
func (w *BaseWidget) WidthExplicit() bool { return w.widthExplicit }

// HeightExplicit reports whether the height came from an explicit size.
func (w *BaseWidget) HeightExplicit() bool { return w.heightExplicit }

func (w *BaseWidget) SetHeight(height float32) *BaseWidget {
	w.height = height
	w.MarkForRedraw()
	return w
}

func (w *BaseWidget) SetHeightDirect(height float32) {
	w.heightExplicit = height > 0
	w.height = height
	// Don't call MarkForRedraw() - used during construction
}

func (w *BaseWidget) SetBounds(x, y, width, height float32) *BaseWidget {
	w.x = x
	w.y = y
	w.width = width
	w.height = height
	w.MarkForRedraw()
	return w
}

func (w *BaseWidget) Enabled() bool {
	return w.enabled
}

func (w *BaseWidget) SetEnabled(enabled bool) *BaseWidget {
	w.enabled = enabled
	w.MarkForRedraw()
	return w
}

// ApplyEnabled is the non-fluent form used by reflective property
// appliers: interface{ SetEnabled(bool) } fails to match widgets whose
// SetEnabled returns *BaseWidget (the fluent pattern), silently no-opping
// `enabled: false` for most widgets.
func (w *BaseWidget) ApplyEnabled(enabled bool) {
	w.enabled = enabled
	if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
		fmt.Printf("[applyEnabled] widget=%p -> %v\n", w, enabled)
	}
	w.MarkForRedraw()
}

// ApplyVisible is the non-fluent form for reflective appliers (see
// ApplyEnabled).
func (w *BaseWidget) ApplyVisible(visible bool) {
	w.visible = visible
	w.MarkForRedraw()
}

func (w *BaseWidget) Visible() bool {
	return w.visible
}

func (w *BaseWidget) SetVisible(visible bool) *BaseWidget {
	w.visible = visible
	w.MarkForRedraw()
	return w
}

func (w *BaseWidget) Focused() bool {
	return w.focused
}

func (w *BaseWidget) SetFocused(focused bool) *BaseWidget {
	w.focused = focused
	w.MarkForRedraw()
	return w
}

func (w *BaseWidget) Hovered() bool {
	return w.hovered
}

func (w *BaseWidget) SetHovered(hovered bool) *BaseWidget {
	if w.hovered != hovered {
		w.hovered = hovered
		if hovered && w.onMouseEnter != nil {
			w.onMouseEnter()
		} else if !hovered && w.onMouseExit != nil {
			w.onMouseExit()
		}
		w.MarkForRedraw()
	}
	return w
}

func (w *BaseWidget) SetHoveredState(hovered bool) {
	w.SetHovered(hovered)
}

func (w *BaseWidget) ContainsPoint(x, y float32) bool {
	if !w.visible || !w.enabled {
		return false
	}
	return x >= 0 && x <= w.width && y >= 0 && y <= w.height
}

func (w *BaseWidget) PreferredDimensions(target interface{}, width, height float32) (float32, float32) {
	return w.width, w.height
}

func (w *BaseWidget) MinimumDimensions() (float32, float32) {
	return 30, 20
}

func (w *BaseWidget) MarkForRedraw() {
	w.needsRedraw = true
	// Propagate redraw request up to parent
	if w.parent != nil {
		if p, ok := w.parent.(interface{ MarkForRedraw() }); ok {
			p.MarkForRedraw()
		}
	}
}

// Redraw forces an immediate redraw of this widget and all ancestors.
// This is called by the frame loop when NeedsRedraw() returns true.
func (w *BaseWidget) Redraw() {
	w.needsRedraw = false
	// Propagate redraw up to parent
	if w.parent != nil {
		if p, ok := w.parent.(interface{ Redraw() }); ok {
			p.Redraw()
		}
	}
}

func (w *BaseWidget) NeedsLayout() bool {
	return w.needsLayout
}

func (w *BaseWidget) SetNeedsLayout() {
	w.needsLayout = true
}

// InvalidateAncestors marks this widget's PARENTS as needing layout.
//
// IT EXISTS because a widget's frame is set by its parent's layout, not by itself, so a
// widget that changes its own size and only marks ITSELF leaves the old frame exactly
// where it was — which reads as the widget failing to resize, and no amount of
// SetNeedsLayout on the widget itself will move it.
//
// Collapsible is the first widget in the tree that changes its own height (a collapsed row
// is its header, an open one is header plus body), which is what made the gap visible: the
// toggle worked and nothing moved.
func (w *BaseWidget) InvalidateAncestors() {
	for p := w.Parent(); p != nil; p = p.Parent() {
		if nl, ok := p.(interface{ SetNeedsLayout() }); ok {
			nl.SetNeedsLayout()
		}
	}
}

func (w *BaseWidget) ClearNeedsLayout() {
	w.needsLayout = false
}

func (w *BaseWidget) MarkClean() {
	w.needsRedraw = false
}

func (w *BaseWidget) NeedsRedraw() bool {
	return w.needsRedraw
}

func (w *BaseWidget) SetOnMouseDown(handler func(x, y float32, button int)) *BaseWidget {
	w.onMouseDown = handler
	return w
}

func (w *BaseWidget) SetOnMouseUp(handler func(x, y float32, button int)) *BaseWidget {
	w.onMouseUp = handler
	return w
}

func (w *BaseWidget) SetOnMouseMove(handler func(x, y float32)) *BaseWidget {
	w.onMouseMove = handler
	return w
}

func (w *BaseWidget) SetOnKeyDown(handler func(keyCode platform.KeyCode, modifiers platform.Modifiers)) *BaseWidget {
	w.onKeyDown = handler
	return w
}

func (w *BaseWidget) SetOnClick(handler func()) *BaseWidget {
	w.onClick = handler
	return w
}

func (w *BaseWidget) SetOnMouseEnter(handler func()) *BaseWidget {
	w.onMouseEnter = handler
	return w
}

func (w *BaseWidget) SetOnMouseExit(handler func()) *BaseWidget {
	w.onMouseExit = handler
	return w
}

func (w *BaseWidget) Modifiers() platform.Modifiers {
	p := w.parent
	for p != nil {
		if ww, ok := p.(*Window); ok {
			return ww.Modifiers()
		}
		if getter, ok := p.(interface{ Modifiers() platform.Modifiers }); ok {
			return getter.Modifiers()
		}
		if pp, ok := p.(interface{ Parent() Widget }); ok {
			p = pp.Parent()
		} else {
			break
		}
	}
	return platform.Modifiers(0)
}

// HasMouseHandlers reports whether mouse-down/up callbacks are
// installed on this widget (touch hit-slop uses this to identify
// interactive targets).
func (w *BaseWidget) HasMouseHandlers() bool {
	return w.onMouseDown != nil || w.onMouseUp != nil
}

func (w *BaseWidget) OnMouseDown(x, y float32, button int) bool {
	if w.onMouseDown != nil {
		w.onMouseDown(x, y, button)
		return true
	}
	return false
}

func (w *BaseWidget) OnMouseUp(x, y float32, button int) bool {
	if w.onMouseUp != nil {
		w.onMouseUp(x, y, button)
		return true
	}
	return false
}

func (w *BaseWidget) OnMouseMove(x, y float32) {
	if w.onMouseMove != nil {
		w.onMouseMove(x, y)
	}
}

func (w *BaseWidget) OnMouseWheel(x, y, dx, dy float32) bool {
	return false
}

func (w *BaseWidget) OnClick(x, y float32) {
	if w.onClick != nil {
		w.onClick()
	}
}

func (w *BaseWidget) OnFocus() {
	w.focused = true
	w.MarkForRedraw()
}

func (w *BaseWidget) OnBlur() {
	w.focused = false
	w.MarkForRedraw()
}

func (w *BaseWidget) SetTabIndex(index int) {
	w.tabIndex = index
}

func (w *BaseWidget) TabIndex() int {
	return w.tabIndex
}

// WidthIsExplicit/HeightIsExplicit report that QML set width/height
// directly — layouts must not let measured preferences override them
// (a fixed-width wrapping label would otherwise grow to its unwrapped
// text width and never wrap).
func (w *BaseWidget) Focusable() bool {
	// FlatLaf semantics: disabled and hidden widgets are skipped by
	// keyboard focus traversal.
	return w.tabIndex >= 0 && w.Enabled() && w.Visible()
}

// Parent returns the parent widget, if any.
func (w *BaseWidget) Parent() Widget {
	return w.parent
}

// SetParent sets the parent widget.
func (w *BaseWidget) SetParent(p Widget) {
	w.parent = p
}

// Window walks up the parent chain to find the root Window. Returns nil if not found.
func (w *BaseWidget) Window() *Window {
	for cur := Widget(w); cur != nil; {
		if win, ok := cur.(*Window); ok {
			return win
		}
		if p, ok := cur.(interface{ Parent() Widget }); ok {
			cur = p.Parent()
		} else {
			break
		}
	}
	return nil
}

// Scale returns the scale for this widget. Default is 1.0.
func (w *BaseWidget) Scale() float32 {
	return 1.0
}

// TooltipText returns the tooltip text for this widget.
func (w *BaseWidget) TooltipText() string {
	return w.tooltipText
}

// TooltipDelay returns the delay before showing the tooltip.
func (w *BaseWidget) TooltipDelay() time.Duration {
	if w.tooltipDelay <= 0 {
		return 500 * time.Millisecond
	}
	return w.tooltipDelay
}

// PointToRoot converts local coordinates to root (window) coordinates.
// Walks the parent chain, adding each ancestor's FrameRect offset.
// PointToRoot converts local coordinates to root (window) coordinates.
// If any ancestor implements scrollAdjuster, its scroll offset is subtracted
// so coordinates account for ScrollPane scrolling (which translates the
// canvas during paint without changing widget FrameRects).
func (w *BaseWidget) PointToRoot(x, y float32) graphics.Point {
	pt := graphics.NewPoint(x, y)
	ancestors := w.collectAncestors()
	for i := len(ancestors) - 1; i >= 0; i-- {
		frame := ancestors[i].FrameRect()
		pt = graphics.NewPoint(pt.X+frame.X, pt.Y+frame.Y)
		if sa, ok := ancestors[i].(scrollAdjuster); ok {
			sx, sy := sa.ScrollOffsets()
			pt.X -= sx
			pt.Y -= sy
		}
	}
	frame := w.FrameRect()
	pt = graphics.NewPoint(pt.X+frame.X, pt.Y+frame.Y)
	return pt
}

// PointFromRoot converts root coordinates to local coordinates.
// Walks the parent chain, subtracting each ancestor's FrameRect offset
// and re-adding any scroll offset that PointToRoot subtracted.
func (w *BaseWidget) PointFromRoot(x, y float32) graphics.Point {
	pt := graphics.NewPoint(x, y)
	ancestors := w.collectAncestors()
	for i := len(ancestors) - 1; i >= 0; i-- {
		frame := ancestors[i].FrameRect()
		pt = graphics.NewPoint(pt.X-frame.X, pt.Y-frame.Y)
		if sa, ok := ancestors[i].(scrollAdjuster); ok {
			sx, sy := sa.ScrollOffsets()
			pt.X += sx
			pt.Y += sy
		}
	}
	frame := w.FrameRect()
	pt = graphics.NewPoint(pt.X-frame.X, pt.Y-frame.Y)
	return pt
}

// collectAncestors returns the list of ancestors from parent up to root.
func (w *BaseWidget) collectAncestors() []Widget {
	resolver := Widget(w)
	var ancestors []Widget
	for {
		parent := resolver.Parent()
		if parent == nil {
			break
		}
		ancestors = append(ancestors, parent)
		resolver = parent
	}
	return ancestors
}

// FromLocal converts widget-local coordinates to window coordinates.
func (w *BaseWidget) FromLocal(x, y float32) (float32, float32) {
	pt := w.PointToRoot(x, y)
	return pt.X, pt.Y
}

// ToLocal converts window coordinates to widget-local coordinates.
func (w *BaseWidget) ToLocal(x, y float32) (float32, float32) {
	pt := w.PointFromRoot(x, y)
	return pt.X, pt.Y
}

// ── Container: child management helpers ──────────────────────────

// AddChild appends a child and sets its parent. Triggers layout + redraw.
func (c *BaseContainer) AddChild(child Widget) {
	if child != nil {
		// A child that already has a different parent stays in that parent's
		// children list - this only appends - so it ends up in TWO containers and
		// is painted by both windows, while Parent() (and therefore Window())
		// points at this one only. That is a widget appearing in a window that
		// does not own it, and no paint-time audit can see it because the
		// resolution is self-consistent. Reported rather than fixed here: removing
		// it from the old parent silently would change widget behaviour.
		if pw, ok := child.(interface{ Parent() Widget }); ok {
			if p := pw.Parent(); p != nil && p != Widget(c) {
				if os.Getenv("MEUI_REPARENT_DEBUG") != "" {
					log.Printf("[parent] AddChild %T added to %T while still parented to %T", child, c, p)
				}
			}
		}
	}
	c.children = append(c.children, child)
	if child != nil {
		child.SetParent(c)
	}
	c.needsLayout = true
	c.MarkForRedraw()
}

// AppendChild appends a child without setting its parent.
// The caller must call child.SetParent(outerWidget) themselves.
// Used by containers in other packages that need to set the parent
// to the outer widget, not the embedded BaseContainer.
func (c *BaseContainer) AppendChild(child Widget) {
	c.children = append(c.children, child)
	c.needsLayout = true
	c.MarkForRedraw()
}

// RemoveChild removes a child by index. Returns false if out of range.
func (c *BaseContainer) RemoveChild(idx int) bool {
	if idx < 0 || idx >= len(c.children) {
		return false
	}
	if child := c.children[idx]; child != nil {
		if d, ok := child.(interface{ Dispose() }); ok {
			d.Dispose()
		}
		child.SetParent(nil)
	}
	c.children = append(c.children[:idx], c.children[idx+1:]...)
	c.needsLayout = true
	c.MarkForRedraw()
	return true
}

// DetachChild removes a child without calling Dispose. Useful for reparenting.
func (c *BaseContainer) DetachChild(child Widget) bool {
	for i, ch := range c.children {
		if ch == child {
			c.children = append(c.children[:i], c.children[i+1:]...)
			if child != nil {
				child.SetParent(nil)
			}
			c.needsLayout = true
			c.MarkForRedraw()
			return true
		}
	}
	return false
}

// ClearChildren disposes and removes all children.
func (c *BaseContainer) ClearChildren() {
	for _, child := range c.children {
		if child != nil {
			if d, ok := child.(interface{ Dispose() }); ok {
				d.Dispose()
			}
			child.SetParent(nil)
		}
	}
	c.children = nil
	c.needsLayout = true
	c.MarkForRedraw()
}

// Children returns the child list.
func (c *BaseContainer) Children() []Widget {
	return c.children
}

// ChildrenI returns children as []interface{} for layout compatibility.
func (c *BaseContainer) ChildrenI() []interface{} {
	if len(c.children) == 0 {
		return nil
	}
	result := make([]interface{}, len(c.children))
	for i, child := range c.children {
		result[i] = child
	}
	return result
}

// ── Container: default event dispatch helpers ────────────────────
// Containers call these from their own OnMouse* methods to get
// standard hit-test + dispatch behavior. They iterate children in
// reverse Z-order, skip invisible children, and translate coords
// to each child's local space before dispatching.

// DefaultPanelAt returns the deepest child containing (x,y) in local coords.
// Returns nil if no child matches — the caller decides what to return as fallback.
func (c *BaseContainer) DefaultPanelAt(x, y float32) Widget {
	floor := touchFloorFor(c)
	ordered := c.orderedChildren()
	for i := len(ordered) - 1; i >= 0; i-- {
		child := ordered[i]
		if child == nil {
			continue
		}
		if v, ok := child.(interface{ Visible() bool }); ok {
			if !v.Visible() {
				continue
			}
		}
		localX, localY, ok := ChildLocalAmongFor(child, x, y, c.children, floor)
		if ok {
			if s, ok := child.(interface{ Scale() float32 }); ok {
				if sc := s.Scale(); sc != 0 {
					localX /= sc
					localY /= sc
				}
			}
			if container, ok := child.(Container); ok {
				if deeper := container.PanelAt(localX, localY); deeper != nil {
					return deeper
				}
			}
			return child
		}
	}
	return nil
}

// DispatchMouseDown iterates children and dispatches OnMouseDown to the first hit.
// Returns true if any child handled the event.
func (c *BaseContainer) DispatchMouseDown(x, y float32, button int) bool {
	floor := touchFloorFor(c)
	ordered := c.orderedChildren()
	for i := len(ordered) - 1; i >= 0; i-- {
		child := ordered[i]
		if child == nil {
			continue
		}
		if v, ok := child.(interface{ Visible() bool }); ok {
			if !v.Visible() {
				continue
			}
		}
		if localX, localY, ok := ChildLocalFor(child, x, y, floor); ok {
			if el, ok := child.(interface {
				OnMouseDown(float32, float32, int) bool
			}); ok {
				if el.OnMouseDown(localX, localY, button) {
					return true
				}
			}
		}
	}
	return false
}

// DispatchMouseUp iterates children and dispatches OnMouseUp to the first hit.
func (c *BaseContainer) DispatchMouseUp(x, y float32, button int) bool {
	floor := touchFloorFor(c)
	ordered := c.orderedChildren()
	for i := len(ordered) - 1; i >= 0; i-- {
		child := ordered[i]
		if child == nil {
			continue
		}
		if v, ok := child.(interface{ Visible() bool }); ok {
			if !v.Visible() {
				continue
			}
		}
		if localX, localY, ok := ChildLocalFor(child, x, y, floor); ok {
			if el, ok := child.(interface {
				OnMouseUp(float32, float32, int) bool
			}); ok {
				if el.OnMouseUp(localX, localY, button) {
					return true
				}
			}
		}
	}
	return false
}

// DispatchMouseMove iterates children and dispatches OnMouseMove to all hits.
func (c *BaseContainer) DispatchMouseMove(x, y float32) {
	floor := touchFloorFor(c)
	ordered := c.orderedChildren()
	for i := len(ordered) - 1; i >= 0; i-- {
		child := ordered[i]
		if child == nil {
			continue
		}
		if v, ok := child.(interface{ Visible() bool }); ok {
			if !v.Visible() {
				continue
			}
		}
		if localX, localY, ok := ChildLocalFor(child, x, y, floor); ok {
			if el, ok := child.(interface{ OnMouseMove(float32, float32) }); ok {
				el.OnMouseMove(localX, localY)
			}
		}
	}
}

// DispatchMouseWheel iterates children and dispatches OnMouseWheel to the first hit.
func (c *BaseContainer) DispatchMouseWheel(x, y, dx, dy float32) bool {
	floor := touchFloorFor(c)
	ordered := c.orderedChildren()
	for i := len(ordered) - 1; i >= 0; i-- {
		child := ordered[i]
		if child == nil {
			continue
		}
		if v, ok := child.(interface{ Visible() bool }); ok {
			if !v.Visible() {
				continue
			}
		}
		if localX, localY, ok := ChildLocalFor(child, x, y, floor); ok {
			if sh, ok := child.(interface {
				OnMouseWheel(float32, float32, float32, float32) bool
			}); ok {
				if sh.OnMouseWheel(localX, localY, dx, dy) {
					return true
				}
			}
		}
	}
	return false
}

// PaintChildren iterates children and paints each one.
// Call this from your widget's Paint() after drawing your own content.
func (c *BaseContainer) PaintChildren(canvas *graphics.Canvas) {
	if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
		types := make([]string, 0, len(c.children))
		for _, ch := range c.children {
			types = append(types, fmt.Sprintf("%T", ch))
		}
		fmt.Printf("[paintChildren] n=%d %v\n", len(c.children), types)
	}
	for _, child := range c.orderedChildren() {
		if child == nil {
			continue
		}
		cf := child.FrameRect()
		if os.Getenv("MEUI_PAINT_TRACE") != "" {
			fmt.Printf("[paintChild] parent=%T child=%T frame=(%.0f,%.0f %.0fx%.0f)\n",
				c, child, cf.X, cf.Y, cf.Width, cf.Height)
		}
		canvas.Save()
		canvas.Translate(cf.X, cf.Y)
		before := 0
		clipDebug := os.Getenv("MEUI_CLIP_DEBUG") != ""
		if clipDebug {
			before = canvas.ClipDepth()
		}
		child.Paint(canvas, graphics.NewRect(0, 0, cf.Width, cf.Height))
		if clipDebug {
			if after := canvas.ClipDepth(); after != before {
				fmt.Printf("[clipleak] %T depth %d -> %d\n", child, before, after)
			}
		}
		canvas.Restore()
	}
}

// MoveChildToFront removes the child and re-adds it at the end,
// making it the topmost in z-order.
func (c *BaseContainer) MoveChildToFront(child Widget) {
	for i, ch := range c.children {
		if ch == child {
			c.children = append(c.children[:i], c.children[i+1:]...)
			c.children = append(c.children, child)
			c.MarkForRedraw()
			return
		}
	}
}

// Dispose cleans up children and resources.
func (c *BaseContainer) Dispose() {
	c.ClearChildren()
	c.onMouseDown = nil
	c.onMouseUp = nil
	c.onMouseMove = nil
	c.onMouseEnter = nil
	c.onMouseExit = nil
	c.onKeyDown = nil
	c.onClick = nil
	c.parent = nil
	c.layoutProps = nil
	c.anchors = nil
}

// SetZIndex orders the widget among its siblings: higher paints later and is hit
// first. Equal z keeps insertion order.
func (w *BaseWidget) SetZIndex(z float32) *BaseWidget {
	if w.zIndex == z {
		return w
	}
	w.zIndex = z
	// The parent's cached order (and its hit test) is now stale.
	if p := parentOf(w); p != nil {
		if c, ok := p.(*BaseContainer); ok {
			c.orderCache = nil
		}
	}
	w.MarkForRedraw()
	return w
}

// ZIndex reports the sibling order (0 by default).
func (w *BaseWidget) ZIndex() float32 { return w.zIndex }

// zOf reads a widget's z without assuming its concrete type.
func zOf(w Widget) float32 {
	if z, ok := w.(interface{ ZIndex() float32 }); ok {
		return z.ZIndex()
	}
	return 0
}

// orderedChildren is the paint/hit order: ascending z, insertion order for equal z.
// The common case (every z is zero) returns the children as they are — no allocation,
// no sort, and exactly the old order.
func (c *BaseContainer) orderedChildren() []Widget {
	n := len(c.children)
	if n == 0 {
		return nil
	}
	sorted := false
	for _, ch := range c.children {
		if ch != nil && zOf(ch) != 0 {
			sorted = true
			break
		}
	}
	if !sorted {
		return c.children
	}
	if c.orderCache != nil && len(c.orderCache) == n {
		stale := false
		for i := range c.orderCache {
			if c.orderCache[i] != c.children[i] {
				stale = true
				break
			}
		}
		if !stale {
			return c.orderCache
		}
	}
	out := make([]Widget, n)
	copy(out, c.children)
	sort.SliceStable(out, func(i, j int) bool { return zOf(out[i]) < zOf(out[j]) })
	c.orderCache = out
	return out
}

// PanelAt returns the deepest child widget containing the point (in local coordinates).
// This is a default implementation for widgets that don't have children.
// Override this in container widgets (Window, Panel).
func (w *BaseWidget) PanelAt(x, y float32) Widget {
	return Widget(w)
}

func (w *BaseWidget) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
}

// GetLayoutProps returns the layout properties for this widget
func (w *BaseWidget) GetLayoutProps() *layout.LayoutProps {
	return w.layoutProps
}

// SetLayoutProps sets the layout properties for this widget
func (w *BaseWidget) SetLayoutProps(props *layout.LayoutProps) {
	w.layoutProps = props
}

// GetAnchors returns the anchor properties for this widget
func (w *BaseWidget) GetAnchors() *layout.Anchors {
	return w.anchors
}

// SetAnchors sets the anchor properties for this widget
func (w *BaseWidget) SetAnchors(anchors *layout.Anchors) {
	w.anchors = anchors
}

// Dispose cleans up resources held by this widget.
// Nil out event handlers and parent reference to allow GC.
func (w *BaseWidget) Dispose() {
	w.onMouseDown = nil
	w.onMouseUp = nil
	w.onMouseMove = nil
	w.onMouseEnter = nil
	w.onMouseExit = nil
	w.onKeyDown = nil
	w.onClick = nil
	w.parent = nil
	w.layoutProps = nil
	w.anchors = nil
}

// SetDebugName sets a human-readable name for this widget (used in debug output).
func (w *BaseWidget) SetDebugName(name string) {
	w.debugName = name
}

// DebugName returns the debug name of this widget.
func (w *BaseWidget) DebugName() string {
	return w.debugName
}

// SetTooltip sets the tooltip text for this widget. Default delay is 2 seconds.
func (w *BaseWidget) SetTooltip(text string) *BaseWidget {
	w.tooltipText = text
	w.tooltipDelay = 2 * time.Second
	return w
}

// SetTooltipDelay sets a custom delay for the tooltip.
func (w *BaseWidget) SetTooltipDelay(d time.Duration) *BaseWidget {
	w.tooltipDelay = d
	return w
}

// StartTooltipTimer starts the tooltip show timer (called on mouse enter).
func (w *BaseWidget) StartTooltipTimer(mouseX, mouseY float32) {
	if w.tooltipText == "" {
		return
	}
	w.StopTooltipTimer()
	w.tooltipMouseX = mouseX
	w.tooltipMouseY = mouseY
	w.tooltipTimer = time.AfterFunc(w.tooltipDelay, func() {
		w.tooltipVisible = true
		w.MarkForRedraw()
	})
}

// StopTooltipTimer cancels the tooltip timer and hides the tooltip (called on mouse exit).
func (w *BaseWidget) StopTooltipTimer() {
	w.tooltipVisible = false
	if w.tooltipTimer != nil {
		w.tooltipTimer.Stop()
		w.tooltipTimer = nil
	}
	w.MarkForRedraw()
}

// DrawTooltip renders the tooltip on the canvas if visible. Call from widget's Paint().
func (w *BaseWidget) DrawTooltip(canvas *graphics.Canvas, rect graphics.Rect) {
	if !w.tooltipVisible || w.tooltipText == "" {
		return
	}
	f := theme.DefaultTheme.BaseFont
	if f == nil {
		f = graphics.SystemFont
	}
	if f == nil {
		return
	}
	ascent := f.Baseline()
	tw := f.SimpleWidth(w.tooltipText) + 16
	th := f.LineHeight() + 8
	tx := w.tooltipMouseX + 12
	ty := w.tooltipMouseY - th - 8
	if ty < rect.Y {
		ty = w.tooltipMouseY + 20
	}
	if tx+tw > rect.X+rect.Width {
		tx = rect.X + rect.Width - tw - 4
	}

	bg := graphics.NewPaint()
	bg.SetColor(theme.DefaultTheme.TooltipBackground)
	canvas.DrawRoundedRect(graphics.NewRect(tx, ty, tw, th), 3, 3, bg)
	bd := graphics.NewPaint()
	bd.SetStyle(paintstyle.Stroke)
	bd.SetColor(theme.DefaultTheme.Border)
	canvas.DrawRoundedRect(graphics.NewRect(tx, ty, tw, th), 3, 3, bd)
	tp := graphics.NewPaint()
	tp.SetColor(theme.DefaultTheme.TooltipText)
	canvas.DrawSimpleString(w.tooltipText, tx+8, ty+4+ascent, f, tp)
}

// SetTreePath sets the tree path string (e.g., "1,2,4,5").
func (w *BaseWidget) SetTreePath(path string) {
	w.treePath = path
}

// TreePath returns the tree path string.
func (w *BaseWidget) TreePath() string {
	return w.treePath
}

// PrintTree prints the widget tree starting from this widget.
// indent is the indentation level (number of spaces / 2).
func PrintTree(root Widget, indent int) {
	printWidgetTree(root, indent, DebugEnabled)
}

// PrintTreeToFile prints the widget tree to a file.
func PrintTreeToFile(root Widget, indent int, f *os.File) {
	printWidgetTreeToFile(root, indent, f)
}

func printWidgetTreeToFile(widget Widget, indent int, f *os.File) {
	prefix := strings.Repeat("  ", indent)
	frame := widget.FrameRect()
	name := "unknown"
	if dn, ok := widget.(interface{ DebugName() string }); ok {
		name = dn.DebugName()
	}
	path := ""
	if tp, ok := widget.(interface{ TreePath() string }); ok {
		path = tp.TreePath()
	}
	fmt.Fprintf(f, "%s- %s [path=%s] frame=%v\n", prefix, name, path, frame)

	// Print children
	if container, ok := widget.(Container); ok {
		children := container.Children()
		for _, child := range children {
			if child != nil {
				printWidgetTreeToFile(child, indent+1, f)
			}
		}
	}
}

// SetTreePaths walks the widget tree starting from root and sets tree paths.
// The path format is "i,j,k" where i,j,k are indices in parent's children.
// Root's children have paths "0", "1", etc.
func SetTreePaths(root Widget) {
	if root == nil {
		return
	}
	// Process root's children with empty parent path
	setTreePathsRecursive(root, "")
}

func setTreePathsRecursive(parent Widget, parentPath string) {
	if container, ok := parent.(Container); ok {
		children := container.Children()
		for i, child := range children {
			if child == nil {
				continue
			}
			// Build child's path: parentPath + index
			childPath := ""
			if parentPath == "" {
				childPath = fmt.Sprintf("%d", i)
			} else {
				childPath = fmt.Sprintf("%s,%d", parentPath, i)
			}
			// Set path on child
			if sp, ok := child.(interface{ SetTreePath(string) }); ok {
				sp.SetTreePath(childPath)
			}
			// Recurse into child
			setTreePathsRecursive(child, childPath)
		}
	}
}

// printWidgetTree recursively prints a widget and its children.
func printWidgetTree(widget Widget, indent int, enabled bool) {
	if !enabled {
		return
	}
	prefix := strings.Repeat("  ", indent)
	frame := widget.FrameRect()
	name := "unknown"
	if dn, ok := widget.(interface{ DebugName() string }); ok {
		name = dn.DebugName()
	}
	tpath := ""
	if tp, ok := widget.(interface{ TreePath() string }); ok {
		tpath = tp.TreePath()
	}
	fmt.Fprintf(os.Stdout, "%s- %s [path=%s] frame=%v\n", prefix, name, tpath, frame)

	// Print children
	if container, ok := widget.(Container); ok {
		children := container.Children()
		for _, child := range children {
			if child != nil {
				printWidgetTree(child, indent+1, enabled)
			}
		}
	}
}

// DragOverlayInfo holds ghost image data for drag preview.
// Drawn by Window at the very end of Paint, on top of everything.
type DragOverlayInfo struct {
	Image *graphics.Image
	X, Y  float32
	W, H  float32
}

// GlobalOverlay is the current drag ghost overlay, drawn on top of everything.
var GlobalOverlay *DragOverlayInfo

// SetDragOverlay sets the global drag ghost overlay (position is in Window coordinates).
func SetDragOverlay(img *graphics.Image, x, y, w, h float32) {
	GlobalOverlay = &DragOverlayInfo{Image: img, X: x, Y: y, W: w, H: h}
}

// ClearDragOverlay clears the global drag ghost overlay.
func ClearDragOverlay() {
	GlobalOverlay = nil
}

// ── Global named widget registry ─────────────────────────────────
// Used for ContextMenus and other widgets that need to be referenced
// by ID but not rendered as layout children.

var namedWidgets = make(map[string]Widget)

func RegisterNamedWidget(id string, w Widget) {
	namedWidgets[id] = w
}

func GetNamedWidget(id string) Widget {
	return namedWidgets[id]
}

// NamedWidgets returns all globally registered named widgets.
func NamedWidgets() []Widget {
	result := make([]Widget, 0, len(namedWidgets))
	for _, w := range namedWidgets {
		result = append(result, w)
	}
	return result
}

// ContextMenuHoster is an interface for widgets that can host a context menu.
// Window checks for this when showing a context menu popup.
type ContextMenuHoster interface {
	SetOnDismiss(func())
	OnExternalDismiss()
}

// CacheBase provides offscreen-image caching for expensive-to-render widgets.
// Embed in any widget, call GetCached in Paint, and Invalidate when content changes.
type CacheBase struct {
	img      *graphics.Image
	cacheW   float32
	cacheH   float32
	valid    bool
	themeVer uint64 // theme.ThemeVersion() at last render; mismatch auto-invalidates
}

// GetCached returns the cached image if valid and the same size, otherwise
// renders into a new offscreen image via draw, caches it, and draws it.
// MEUI_NO_CACHEBASE=1 bypasses the offscreen cache (draws directly) —
// diagnostic for resize-flash forensics (offscreen re-render +
// glReadPixels + re-upload every size-change frame stalls the pipe).
func (c *CacheBase) GetCached(canvas *graphics.Canvas, rect graphics.Rect, draw func(*graphics.Canvas, graphics.Rect)) {
	if os.Getenv("MEUI_NO_CACHEBASE") != "" {
		draw(canvas, rect)
		return
	}
	if c.themeVer != theme.ThemeVersion() {
		c.valid = false
	}
	if c.valid && c.img != nil && c.cacheW == rect.Width && c.cacheH == rect.Height {
		canvas.DrawImageInRect(c.img, rect, nil, nil)
		return
	}
	w := int(rect.Width)
	h := int(rect.Height)
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	img, err := graphics.NewImageFromDrawing(canvas.Device(), w, h, 72, func(c *graphics.Canvas) {
		draw(c, graphics.NewRect(0, 0, float32(w), float32(h)))
	})
	if err != nil {
		// Fallback: draw directly
		draw(canvas, rect)
		return
	}
	c.img = img
	c.cacheW = rect.Width
	c.cacheH = rect.Height
	c.valid = true
	c.themeVer = theme.ThemeVersion()
	canvas.DrawImageInRect(img, rect, nil, nil)
}

// Invalidate marks the cached image as stale, forcing a re-render on the next GetCached call.
func (c *CacheBase) Invalidate() {
	c.valid = false
	c.img = nil
}
