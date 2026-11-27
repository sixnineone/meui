package meui

import (
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
	"github.com/sixnineone/mobilegles"
)

var windows []*Window

func init() {
	theme.OnThemeChanged(func() {
		for _, w := range windows {
			w.panelToExpose.MarkForRedraw()
		}
	})
}

// Window is the top-level container that receives events and manages children.
type Window struct {
	*BaseContainer
	panelToExpose   *Panel
	layout          interface{}
	padding         float32
	background      graphics.Ink
	opacity         float32
	mouseDownWidget Widget
	hoveredChild    Widget
	// onResize is the app's hook for the window's CONTENT size changing. It fires
	// only when the size actually changes, from the resize event — not from every
	// SetFrameRect (the paint path and the chrome both call that).
	onResize func(width, height float32)
	// hoverChain is the HoverTarget walk from the last pointer move. It is kept so
	// that a target the pointer has LEFT hears about it: the walk only visits the
	// CURRENT hit's ancestors, so a card beside the new hit (a sibling) would
	// otherwise stay hovered forever — the same "sees the pointer arrive and never
	// leave" defect the chain was introduced to fix, one level out.
	hoverChain    []Widget
	focusedChild  Widget
	capture       interface{}
	LayerManager  *LayerManager
	Drag          *DragManager
	lastModifiers platform.Modifiers
	// keysDown is every key currently held. A UI framework needs it for "is shift down" without trusting
	// the event's modifier bits (which describe the event, not the current state), and for a widget that
	// reacts to a key being HELD rather than pressed.
	keysDown       map[platform.KeyCode]bool
	lastClickCount int

	// lastDevice is the device the window last painted through, so offscreen
	// work started outside a paint pass (drag images, screenshots) has one.
	lastDevice       graphics.Device
	onMouseDown      func(x, y float32, button int)
	onMouseUp        func(x, y float32, button int)
	onMouseMove      func(x, y float32)
	onKeyDown        func(keyCode platform.KeyCode, modifiers platform.Modifiers)
	onFocus          func()
	onBlur           func()
	takeScreenshot   func(*graphics.Image, error)
	renderHasHappend bool
	activeModals     []*ModalPopup
	tooltipWidget    Widget
	tooltipText      string
	tooltipTimer     *time.Timer
	tooltipLayer     *Layer
	ctxMenuBackdrop  *Layer
	ctxMenuContent   *Layer
	// ctxMenuID is the registered name of the id-based context menu that is open, so Escape can
	// dismiss it as a unit (menu + backdrop) without a click.
	ctxMenuID          string
	scaleFactor        float32
	layerPaintExternal bool // host paints layers after the title bar

	// contentOriginX/Y is where the content tree's (0,0) sits in window space.
	// Zero unless the host reserves a strip for window chrome (the custom title
	// bar); the host declares it, and LayerOriginFor uses it so no widget ever
	// does bar arithmetic.
	// onBack is the app's handler for the OS back gesture; see SetOnBack.
	onBack func() bool

	contentOriginX, contentOriginY float32
	focusCycleIdx                  int

	// The touch floor is PER WINDOW. touchProfileSet records that this window has an opinion
	// (manual, or derived from its form factor by App.AddWindow); touchProfileAuto records that the
	// opinion was derived, so a resize may re-derive it. Unset falls back to the process default.
	touchProfile     bool
	touchProfileSet  bool
	touchProfileAuto bool

	// onAction is the app's handler for a semantic action from the keymap
	// (platform.Event.Actions). It is the seam an APPLICATION binds shortcuts
	// through, so the toolkit does not have to learn every app's commands.
	onAction func(action string) bool
}

// Scale returns the display's content scale factor (1.0 = normal, 2.0 = Retina).
func (w *Window) Scale() float32 {
	if w.scaleFactor < 1 {
		return 1.0
	}
	return w.scaleFactor
}

// SetDisplayScale overrides the detected display scale. Useful when the window
// moves to a different monitor and the user wants to set the scale explicitly.
func (w *Window) SetDisplayScale(scale float32) {
	if scale < 1 {
		scale = 1.0
	}
	if scale != w.scaleFactor {
		w.setDisplayScale(scale)
	}
}

func (w *Window) setDisplayScale(scale float32) {
	if scale < 1 {
		scale = 1
	}
	w.scaleFactor = scale
	theme.ScaleFonts(theme.DefaultTheme, scale)
	w.markAllNeedsLayout(w)
	w.MarkForRedraw()
}

// Window management
func NewWindow() *Window {
	w := &Window{
		BaseContainer: &BaseContainer{BaseWidget: *NewBaseWidget()},
		LayerManager:  NewLayerManager(),
		keysDown:      map[platform.KeyCode]bool{},
	}
	w.LayerManager.owner = w
	w.Drag = NewDragManager(w)
	w.renderHasHappend = false
	w.panelToExpose = NewPanel() // Store reference for MarkForRedraw propagation
	w.panelToExpose.SetFocusable(true)
	frame := w.FrameRect()
	if frame.Width <= 0 {
		frame.Width = 400
	}
	if frame.Height <= 0 {
		frame.Height = 300
	}
	w.panelToExpose.SetFrameRect(frame)

	w.panelToExpose.DrawCallback = func(canvas *graphics.Canvas, rect graphics.Rect) {
		// Auto-detect display scale change
		display := platform.BestDisplayForRect(rect)
		newScale := float32(1.0)
		if display != nil {
			newScale = display.ScaleX
		}
		if newScale != w.scaleFactor {
			w.setDisplayScale(newScale)
		}

		// Update window size from the panel's draw rect (handles resize)
		w.BaseWidget.SetFrameRect(rect)
		w.markAllNeedsLayout(w)

		bgPaint := graphics.NewPaint()
		bgPaint.SetColor(theme.DefaultTheme.Background)

		if w.takeScreenshot != nil && w.renderHasHappend {
			chartImage, err := graphics.NewImageFromDrawing(canvas.Device(), int(rect.Width), int(rect.Height), 72, func(c *graphics.Canvas) {
				c.DrawRect(rect, bgPaint)
				w.Paint(c, rect)
			})
			canvas.DrawImageRectInRect(chartImage, rect, rect, nil, bgPaint)
			w.takeScreenshot(chartImage, err)
			w.takeScreenshot = nil
		} else {
			canvas.DrawRect(rect, bgPaint)
			w.Paint(canvas, rect)
			w.renderHasHappend = true
		}
	}
	w.panelToExpose.MouseDownCallback = func(where graphics.Point, button, clickCount int, mod platform.Modifiers) bool {
		w.lastModifiers = mod
		w.lastClickCount = clickCount
		return w.handleMouseDown(where.X, where.Y, button)
	}
	w.panelToExpose.MouseUpCallback = func(where graphics.Point, button int, mod platform.Modifiers) bool {
		w.lastModifiers = mod
		return w.handleMouseUp(where.X, where.Y, button)
	}
	w.panelToExpose.MouseMoveCallback = func(where graphics.Point, mod platform.Modifiers) bool {
		w.lastModifiers = mod
		w.handleMouseMove(where.X, where.Y)
		return false
	}
	w.panelToExpose.MouseDragCallback = func(where graphics.Point, button int, mod platform.Modifiers) bool {
		w.lastModifiers = mod
		return w.handleMouseDrag(where.X, where.Y, button)
	}
	w.panelToExpose.MouseWheelCallback = func(where, delta graphics.Point, mod platform.Modifiers) bool {
		w.lastModifiers = mod
		return w.handleMouseWheel(where.X, where.Y, delta.X, delta.Y)
	}
	w.panelToExpose.UpdateCursorCallback = func(where graphics.Point) *platform.Cursor {
		return w.handleUpdateCursor(where.X, where.Y)
	}
	w.panelToExpose.RuneTypedCallback = func(ch rune) bool {
		eventDebug("rune %q: focusedChild=%T", ch, w.focusedChild)
		if w.focusedChild != nil {
			if th, ok := w.focusedChild.(interface{ OnCharacter(rune) }); ok {
				th.OnCharacter(ch)
				w.panelToExpose.MarkForRedraw()
				return true
			}
			eventDebug("rune %q: focusedChild %T has no OnCharacter", ch, w.focusedChild)
		}
		if w.LayerManager != nil && w.LayerManager.HandleRune(ch) {
			w.panelToExpose.MarkForRedraw()
			return true
		}
		return true
	}
	windows = append(windows, w)
	log.Printf("Window is ready - user will close when done testing")
	return w
}

func (w *Window) TakeScreenShot(onImg func(i *graphics.Image, err error)) {
	w.takeScreenshot = onImg
}

func (w *Window) AddChild(child Widget) {
	w.children = append(w.children, child)
	if child != nil {
		child.SetParent(w)
	}
	w.needsLayout = true
	w.MarkForRedraw()
}

func (w *Window) MarkForRedraw() {
	w.BaseWidget.MarkForRedraw()
	if w.panelToExpose != nil {
		w.panelToExpose.MarkForRedraw()
	}
}

func (w *Window) handleUpdateCursor(x, y float32) *platform.Cursor {
	if w.LayerManager != nil {
		layer := w.LayerManager.HitTest(x, y)
		if layer != nil && layer.Widget != nil {
			if cp, ok := layer.Widget.(CursorProvider); ok {
				if c := cp.CursorAt(x-layer.X, y-layer.Y); c != nil {
					return c
				}
			}
		}
	}
	child := w.hitTest(x, y)
	for child != nil {
		// The hit-test can return the window itself (empty space) — the
		// window implements CursorAt for the shim's CursorSource query,
		// so re-entering it here would recurse forever.
		if cp0, isW := child.(*Window); isW && cp0 == w {
			break
		}
		if cp, ok := child.(CursorProvider); ok {
			localX, localY := x, y
			if pfr, ok := child.(interface {
				PointFromRoot(float32, float32) graphics.Point
			}); ok {
				pt := pfr.PointFromRoot(x, y)
				localX, localY = pt.X, pt.Y
			}
			if c := cp.CursorAt(localX, localY); c != nil {
				return c
			}
		}
		if p, ok := child.(interface{ Parent() Widget }); ok {
			child = p.Parent()
		} else {
			break
		}
	}
	return nil
}

func (w *Window) handleMouseDrag(x, y float32, button int) bool {
	eventDebug("drag at (%.0f,%.0f) capture=%T", x, y, w.capture)
	// Route to layers first (popups, dropdowns, etc.)
	if w.LayerManager != nil && w.LayerManager.HandleMouseMove(x, y) {
		return true
	}
	if w.Drag.Active {
		w.Drag.Update(x, y, 0, 0)
		return true
	}
	// Check if we should start a drag
	if w.mouseDownWidget != nil {
		if w.Drag.StartCheck(w.mouseDownWidget, x, y) {
			return true
		}
	}
	if w.capture == nil {
		return false
	}
	target, ok := w.capture.(Widget)
	if !ok {
		w.capture = nil
		return false
	}
	if el, ok := target.(EventListener); ok {
		localX, localY := x, y
		if pfr, ok := target.(interface {
			PointFromRoot(float32, float32) graphics.Point
		}); ok {
			pt := pfr.PointFromRoot(x, y)
			localX, localY = pt.X, pt.Y
		}
		el.OnMouseMove(localX, localY)
		return true
	}
	return false
}

func (w *Window) handleMouseWheel(x, y float32, dx, dy float32) bool {
	LogEventf("Window", "MouseWheel at (%.0f,%.0f) delta=(%.1f,%.1f)", x, y, dx, dy)

	// Try LayerManager first (dropdowns/popups)
	if w.LayerManager != nil && w.LayerManager.HandleMouseWheel(x, y, dx, dy) {
		return true
	}

	// Walk up parent chain from deepest widget under cursor
	child := w.hitTest(x, y)
	for child != nil {
		if sh, ok := child.(ScrollHandler); ok {
			localX, localY := x, y
			if pfr, ok := child.(interface {
				PointFromRoot(float32, float32) graphics.Point
			}); ok {
				pt := pfr.PointFromRoot(x, y)
				localX, localY = pt.X, pt.Y
			}
			if sh.OnMouseWheel(localX, localY, dx, dy) {
				return true
			}
		}
		if p, ok := child.(interface{ Parent() Widget }); ok {
			child = p.Parent()
		} else {
			break
		}
	}
	return false
}

func (w *Window) handleMouseDown(x, y float32, button int) bool {
	// Window-level notification, delivered for every press regardless of which
	// widget consumes it. SetOnMouseDown stored this handler but nothing ever
	// called it, so the public hook was dead: a caller wiring it saw no presses
	// at all, which reads as "input is broken". It is a notification, not a
	// handler, so it must not change which child handles the press — hence it
	// runs here rather than in the fall-through below, which a hit child never
	// reaches.
	if w.onMouseDown != nil {
		w.onMouseDown(x, y, button)
	}

	// Check if any layer is hit
	layerPresent := w.LayerManager != nil && w.LayerManager.HitTest(x, y) != nil
	if layerPresent {
		handled := w.LayerManager.HandleMouseDown(x, y, button)
		if handled {
			return true
		}
	} else {
		w.dismissNonModalPopups()
	}

	child := w.hitTest(x, y)
	if child != nil {
		w.Drag.RecordStart(x, y)
		w.mouseDownWidget = child
		w.hoveredChild = child
		w.capture = child

		if _, ok := child.(FocusHandler); ok {
			// The deepest hit is often a label or a cell INSIDE the control the user meant. Focus the
			// nearest focusable ancestor so a framed field, a table or a chart takes the keyboard when
			// the click lands on its face.
			w.setFocus(NearestFocusable(child))
		}

		handled := false
		if el, ok := child.(EventListener); ok {
			localX, localY := x, y
			if pfr, ok := child.(interface {
				PointFromRoot(float32, float32) graphics.Point
			}); ok {
				pt := pfr.PointFromRoot(x, y)
				localX, localY = pt.X, pt.Y
			}
			if !child.ContainsPoint(localX, localY) {
				return false
			}
			handled = el.OnMouseDown(localX, localY, button)
		}
		// A press the deepest hit DECLINES bubbles to its ancestors. A label is not
		// the card it sits in, and stopping at the label made every clickable box
		// with text in it inert — the classic "it looks like a button and does
		// nothing". Only a widget that explicitly offers the bubble is asked, so no
		// existing widget's input changes.
		if !handled {
			for p := parentOf(child); p != nil; p = parentOf(p) {
				bh, ok := p.(interface {
					OnBubbledMouseDown(float32, float32, int) bool
				})
				if !ok {
					continue
				}
				pt := bubblePoint(p, x, y)
				if !p.ContainsPoint(pt.X, pt.Y) {
					continue
				}
				if bh.OnBubbledMouseDown(pt.X, pt.Y, button) {
					w.mouseDownWidget = p
					w.capture = p
					break
				}
			}
		}
		return true
	}

	if w.LayerManager != nil && w.LayerManager.HandleMouseDown(x, y, button) {
		return true
	}

	return false
}

func (w *Window) handleMouseUp(x, y float32, button int) bool {
	LogEventf("Window", "MouseUp at (%.0f,%.0f) button=%d", x, y, button)

	if w.Drag.Active {
		w.Drag.End(x, y)
		return true
	}

	if w.LayerManager != nil && w.LayerManager.HandleMouseUp(x, y, button) {
		return true
	}

	handled := false
	if w.mouseDownWidget != nil {
		LogEventf(getDebugName(w.mouseDownWidget), "MouseUp dispatched")
		if el, ok := w.mouseDownWidget.(EventListener); ok {
			localX, localY := x, y
			if pfr, ok := w.mouseDownWidget.(interface {
				PointFromRoot(float32, float32) graphics.Point
			}); ok {
				pt := pfr.PointFromRoot(x, y)
				localX, localY = pt.X, pt.Y
			}
			el.OnMouseUp(localX, localY, button)
		}
		w.mouseDownWidget = nil
		handled = true
	}
	if w.capture != nil {
		w.capture = nil
		handled = true
	}
	// The layer manager was consulted ONCE, above. A second call here re-ran the layer's mouse-up —
	// a dialog's content forwards it to its child, so a control that returned false was released
	// twice for one click.
	return handled
}

// dispatchKeyDown is the ONE key-down path. The window's own handler and the panel's keymap callback both
// come through here, which is what keeps them from drifting apart — they had: one honoured the repeat flag
// and the other discarded it.
func (w *Window) dispatchKeyDown(keyCode platform.KeyCode, mod platform.Modifiers, repeat bool) bool {
	// A MODIFIER NEVER REPEATS. Holding Shift must not deliver Shift, Shift, Shift to a widget that
	// increments on a key press, and the OS will happily repeat it.
	if repeat && isModifierKey(keyCode) {
		return true
	}
	// A widget that wants the repeats gets them through its own interface, so the existing OnKeyDown
	// contract is untouched: changing that signature would break every widget in the tree, and the lesson
	// in AGENTS.md is that a listener whose signature drifts is a widget that compiles and is dead.
	if repeat && w.focusedChild != nil {
		if kr, ok := w.focusedChild.(interface {
			OnKeyRepeat(platform.KeyCode, platform.Modifiers) bool
		}); ok {
			if kr.OnKeyRepeat(keyCode, mod) {
				w.MarkForRedraw()
				return true
			}
		}
		// A widget with no interest in repeats is not asked twice: falling through to OnKeyDown would
		// turn every held key into a stream of fresh presses, which is the bug this all fixes. The
		// widgets that DO want repeats (the editors, the lists, the chart) implement OnKeyRepeat.
		return true
	}
	return w.handleKeyDown(keyCode, mod)
}

// KeyDown reports whether a key is currently held. It is the answer to "is the user holding shift RIGHT
// NOW", which an event's modifier bits cannot give: they describe the event that arrived, not the state
// the keyboard is in.
func (w *Window) KeyDown(k platform.KeyCode) bool { return w.keysDown[k] }

// isModifierKey reports whether a keycode is one of the eight modifier keys. A key-down for any of them
// is a state change, not an action, so it repeats never.
func isModifierKey(k platform.KeyCode) bool {
	switch k {
	case platform.KeyLeftShift, platform.KeyRightShift,
		platform.KeyLeftCtrl, platform.KeyRightCtrl,
		platform.KeyLeftAlt, platform.KeyRightAlt,
		platform.KeyLeftMeta, platform.KeyRightMeta:
		return true
	}
	return false
}

func (w *Window) handleKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	// ESCAPE IS THE WINDOW'S, BEFORE ANYTHING ELSE SEES IT. A UI framework has one universal "back" key:
	// it cancels a drag in progress, dismisses a context menu, dismisses the topmost DIALOG, closes any
	// other popup, and only then reaches a widget — and several widgets also handle Escape themselves (a
	// combo box, a menu bar), which is why this has to be FIRST rather than another case at the end.
	//
	// The dialog and the menu are each dismissed as a UNIT, bookkeeping included. This used to fall
	// through to LayerManager.CloseTopLayer, which removes ONE layer: a modal is two (the content at 9000
	// and the backdrop at 8999), so Escape left a dimmed, click-blocking backdrop with no dialog in it,
	// and Window.activeModals still listed a popup that was no longer there.
	if keyCode == platform.KeyEscape {
		if w.Drag != nil && w.Drag.Active {
			w.Drag.Cancel()
			w.MarkForRedraw()
			return true
		}
		// A context menu is a UNIT too: a menu plus its click-catching backdrop. Closing the top
		// layer alone left the backdrop over the window.
		if w.ctxMenuContent != nil {
			w.DismissPopupContextMenu()
			return true
		}
		if w.ctxMenuID != "" {
			w.DismissContextMenu(w.ctxMenuID)
			return true
		}
		if n := len(w.activeModals); n > 0 {
			w.DismissTopModal()
			return true
		}
		if w.LayerManager != nil && w.LayerManager.CloseTopLayer() {
			w.MarkForRedraw()
			return true
		}
	}

	// Tab / Shift+Tab — cycle through focusable widgets IN THE CURRENT SCOPE: a dialog keeps Tab inside
	// itself, and the window's own tree is the scope only when no dialog is open. Focus in a dialog lived
	// in a layer, and the walk started at the window's children, so Tab cycled the widgets BEHIND an open
	// modal while the modal stayed on top.
	if keyCode == platform.KeyTab {
		w.cycleFocus(w.focusScope(), mod.ShiftDown())
		return true
	}

	// A focus target that has LEFT THE TREE (a dismissed dialog, a rebuilt panel, a removed row) cannot
	// act; keeping it is how keys go nowhere at all.
	w.validateFocus()

	// The focused widget gets every key first — text widgets handle
	// Enter/Space themselves (newline, submit) and must not be hijacked
	// by the click-activation fallback below.
	if w.focusedChild != nil {
		if dispatchKeyTo(w.focusedChild, keyCode, mod) {
			return true
		}

		// Enter / Space — activate focused ClickHandler (buttons, etc.).
		// Deliberately no OnMouseDown fallback: synthesizing a press at (0,0)
		// corrupts text-cursor positions.
		if keyCode == platform.KeyReturn || keyCode == platform.KeyNumPadEnter || keyCode == platform.KeySpace {
			if ch, ok := w.focusedChild.(ClickHandler); ok {
				ch.OnClick(0, 0)
				return true
			}
		}

		// Then UP THE ANCESTORS. A widget's container is part of the same interaction: a tab panel owns
		// the page its content sits on, a panel owns its controls, a dialog owns its content. The walk
		// stops at the window, which has no key behaviour of its own. Without it a key could only ever
		// reach the focused widget and the window's direct children, so a nested widget's handler was
		// unreachable unless the app focused it by hand.
		for p := parentOf(w.focusedChild); p != nil && p != Widget(w); p = parentOf(p) {
			if dispatchKeyTo(p, keyCode, mod) {
				return true
			}
		}
		return false
	}

	// NOTHING FOCUSED — only possible before the first click, and the window still has to respond: offer
	// the key to the topmost layer (a dialog a host opened without taking focus), then to the window's
	// panels. Once something has focus this never runs, so an unfocused control cannot swallow a key
	// meant for the focused one.
	if w.LayerManager != nil && w.LayerManager.HandleKeyDown(keyCode, mod) {
		return true
	}
	for _, child := range w.children {
		if dispatchKeyTo(child, keyCode, mod) {
			return true
		}
	}
	return false
}

// dispatchKeyTo offers one key to one widget and reports whether it handled it.
func dispatchKeyTo(w Widget, keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if kh, ok := w.(interface {
		OnKeyDown(platform.KeyCode, platform.Modifiers) bool
	}); ok {
		return kh.OnKeyDown(keyCode, mod)
	}
	return false
}

// parentOf is the parent link, or nil at a root.
func parentOf(w Widget) Widget {
	if p, ok := w.(interface{ Parent() Widget }); ok {
		return p.Parent()
	}
	return nil
}

// focusScope is where Tab cycles: the topmost dialog's content when one is open, otherwise the window.
func (w *Window) focusScope() Widget {
	if n := len(w.activeModals); n > 0 {
		if top := w.activeModals[n-1]; top != nil && top.ContentWidget != nil {
			return top.ContentWidget
		}
	}
	return w
}

// cycleFocus moves focus one step through the scope's focusable widgets, wrapping at both ends. A
// widget that is not in the scope (focus behind a dialog, or nothing focused at all) enters the cycle
// at its first — or last, going backwards — entry.
func (w *Window) cycleFocus(scope Widget, backwards bool) {
	list := FocusableWidgets(scope)
	if len(list) == 0 {
		return
	}
	if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
		names := make([]string, 0, len(list))
		for _, fw := range list {
			names = append(names, fmt.Sprintf("%T(%p)", fw, fw))
		}
		fmt.Printf("[tabFocus] scope=%T focusable=%d %v\n", scope, len(list), names)
	}
	idx := -1
	for i, fw := range list {
		if fw == w.focusedChild {
			idx = i
			break
		}
	}
	n := len(list)
	switch {
	case idx < 0 && backwards:
		idx = n - 1
	case idx < 0:
		idx = 0
	case backwards:
		idx = (idx - 1 + n) % n
	default:
		idx = (idx + 1) % n
	}
	w.setFocus(list[idx])
}

// validateFocus drops a focus target that has left the tree — a dialog that was dismissed, a panel that
// was rebuilt, a page that was switched away. Attachment is the test that catches all of them without
// guessing at widget kinds: a widget still in the tree keeps focus even if it declares itself
// unfocusable (a pane focused by a click), and one that is no longer reachable cannot act at all.
func (w *Window) validateFocus() {
	f := w.focusedChild
	if f == nil {
		return
	}
	for cur := Widget(f); cur != nil; cur = parentOf(cur) {
		if cur == Widget(w) {
			return
		}
	}
	w.setFocus(nil)
}

// FocusableWidgets returns the widgets in root's subtree that can take keyboard focus, in TabIndex
// order (tree order among equal indices).
//
// It is the ONE focus traversal: the window's Tab uses it, a dialog's Tab uses it with the dialog's
// content as the root, and a container's own Children() decides what is reachable — so a TabPanel
// offers only its active page and a layer's content (which is not in the window tree) is reachable
// at all.
func FocusableWidgets(root Widget) []Widget {
	var out []Widget
	collectFocusable(root, &out)
	sort.SliceStable(out, func(i, j int) bool {
		return tabIndexOf(out[i]) < tabIndexOf(out[j])
	})
	return out
}

func collectFocusable(root Widget, out *[]Widget) {
	if root == nil {
		return
	}
	if ts, ok := root.(TabIndexSetter); ok && ts.Focusable() {
		*out = append(*out, root)
	}
	if c, ok := root.(interface{ Children() []Widget }); ok {
		for _, child := range c.Children() {
			collectFocusable(child, out)
		}
	}
}

func tabIndexOf(w Widget) int {
	if ts, ok := w.(TabIndexSetter); ok {
		return ts.TabIndex()
	}
	return 0
}

// NearestFocusable returns w when it can take focus, otherwise the nearest ancestor that can. A click
// hits the DEEPEST widget — a label, a cell, an icon inside the control the user meant — and focusing
// that instead of the control is why a table row or a framed field was never focused by clicking it.
func NearestFocusable(w Widget) Widget {
	for cur := w; cur != nil; cur = parentOf(cur) {
		if ts, ok := cur.(TabIndexSetter); ok && ts.Focusable() {
			return cur
		}
	}
	return w
}

// ContainsWidget reports whether target is root or sits in root's subtree, using the same Children()
// the focus traversal uses.
func ContainsWidget(root, target Widget) bool {
	if root == nil || target == nil {
		return false
	}
	if root == target {
		return true
	}
	if c, ok := root.(interface{ Children() []Widget }); ok {
		for _, child := range c.Children() {
			if ContainsWidget(child, target) {
				return true
			}
		}
	}
	return false
}

// setFocus is the ONE way keyboard focus moves: the old widget is blurred BEFORE the new one is
// focused, so a control can never be left believing it still has focus (and painting the focus ring,
// and accepting the keyboard's characters).
func (w *Window) setFocus(child Widget) {
	old := w.focusedChild
	if old == child {
		return
	}
	w.focusedChild = child
	if old != nil {
		// A preedit belongs to the widget that was composing; leaving it behind would leave
		// uncommitted text in a field the user is no longer typing into.
		if ch, ok := old.(interface {
			OnComposition(text string, start, length int) bool
		}); ok {
			ch.OnComposition("", 0, 0)
		}
		if fh, ok := old.(FocusHandler); ok {
			fh.OnBlur()
		}
	}
	if child != nil {
		if fh, ok := child.(FocusHandler); ok {
			fh.OnFocus()
		}
	}
	w.MarkForRedraw()
}

func (w *Window) handleMouseMove(x, y float32) {
	LogEventf("Window", "MouseMove at (%.0f,%.0f)", x, y)
	if w.LayerManager != nil && w.LayerManager.HandleMouseMove(x, y) {
		w.MarkForRedraw()
		return
	}
	child := w.hitTest(x, y)

	// Tooltip management
	if child != w.hoveredChild {
		w.cancelTooltip()
		if child != nil {
			if tt, ok := child.(interface{ TooltipText() string }); ok {
				if text := tt.TooltipText(); text != "" {
					w.tooltipWidget = child
					w.tooltipText = text
					delay := 500 * time.Millisecond
					if td, ok := child.(interface{ TooltipDelay() time.Duration }); ok {
						delay = td.TooltipDelay()
					}
					w.tooltipTimer = time.AfterFunc(delay, func() {
						w.showTooltip(x, y, w.tooltipText)
					})
				}
			}
		}
	} else if w.tooltipLayer != nil {
		w.updateTooltipPosition(x, y)
	}

	if child != w.hoveredChild {
		if w.hoveredChild != nil {
			if hh, ok := w.hoveredChild.(HoverHandler); ok {
				hh.SetHoveredState(false)
			}
		}
		w.hoveredChild = child
		if child != nil {
			if hh, ok := child.(HoverHandler); ok {
				hh.SetHoveredState(true)
			}
		}
		w.MarkForRedraw()
	}
	target := child
	if w.capture != nil {
		if t, ok := w.capture.(Widget); ok {
			target = t
		} else {
			w.capture = nil
		}
	}
	if target != nil {
		if el, ok := target.(EventListener); ok {
			localX, localY := x, y
			if pfr, ok := target.(interface {
				PointFromRoot(float32, float32) graphics.Point
			}); ok {
				pt := pfr.PointFromRoot(x, y)
				localX, localY = pt.X, pt.Y
			}
			el.OnMouseMove(localX, localY)
		}
	}

	// HoverTarget walk: the deepest hit and then its parents, each in its own
	// coordinates. A widget holding a pointer position that is only part of
	// its area (a SplitView handle) needs the whole chain, because the hit is
	// normally a child of the panels and OnMouseMove alone would never tell it
	// the pointer left — which left the divider stuck in its hover ink. Same
	// widget-local contract as the cursor walk below.
	chain := make([]Widget, 0, 6)
	for h := child; h != nil; {
		if _, ok := h.(HoverTarget); ok {
			chain = append(chain, h)
		}
		p, ok := h.(interface{ Parent() Widget })
		if !ok || p == nil {
			break
		}
		if parent := p.Parent(); parent != h {
			h = parent
		} else {
			break // a widget parented to itself would loop forever
		}
	}
	// A target from the previous chain that the pointer has left is told so. The
	// walk cannot reach it: it is a sibling of the new hit, not an ancestor.
	for _, old := range w.hoverChain {
		if containsWidget(chain, old) {
			continue
		}
		if hh, ok := old.(HoverHandler); ok {
			hh.SetHoveredState(false)
		}
	}
	w.hoverChain = chain
	for _, h := range chain {
		ht := h.(HoverTarget)
		localX, localY := x, y
		if pfr, ok := h.(interface {
			PointFromRoot(float32, float32) graphics.Point
		}); ok {
			pt := pfr.PointFromRoot(x, y)
			localX, localY = pt.X, pt.Y
		}
		ht.HoverAt(localX, localY)
	}
}

// containsWidget reports whether w is in list (identity, not equality).
func containsWidget(list []Widget, w Widget) bool {
	for _, e := range list {
		if e == w {
			return true
		}
	}
	return false
}

// Children returns the window's content tree. It is the accessor that makes the window
// usable wherever a Container is — a tree walk, an audit, a debug dump — which it was
// not before: the window had AddChild and PanelAt but no way to enumerate what it held,
// so anything that walked a tree had to start somewhere else or give up.
func (w *Window) Children() []Widget { return w.children }

// childrenNeedLayout reports whether any direct child is waiting for a layout pass. It is
// a WINDOW-level question, asked once per paint: a deep descendant is reached through the
// walk ValidateLayout does, and a direct child is what says "there is work".
func (w *Window) childrenNeedLayout() bool {
	for _, c := range w.children {
		if n, ok := c.(interface{ NeedsLayout() bool }); ok && n.NeedsLayout() {
			return true
		}
	}
	return false
}

func (w *Window) hitTest(x, y float32) Widget {
	return w.PanelAt(x, y)
}

// PanelAt returns the deepest child widget containing the point (in local coordinates).
func (w *Window) PanelAt(x, y float32) Widget {
	for i := len(w.children) - 1; i >= 0; i-- {
		child := w.children[i]
		if v, ok := child.(interface{ Visible() bool }); ok {
			if !v.Visible() {
				continue
			}
		}
		localX, localY, inChild := ChildLocalAmongFor(child, x, y, w.children, w.TouchProfile())
		if inChild {
			scale := float32(1.0)
			if s, ok := child.(interface{ Scale() float32 }); ok {
				scale = s.Scale()
			}
			localX /= scale
			localY /= scale
			if container, ok := child.(Container); ok {
				if vc, ok := container.(interface{ NeedsLayout() bool }); ok {
					if vc.NeedsLayout() {
						container.ValidateLayout()
					}
				}
				if deeper := container.PanelAt(localX, localY); deeper != nil {
					return deeper
				}
			}
			return child
		}
	}
	return w
}

// ValidateLayout applies the layout if needed.
func (w *Window) ValidateLayout() {
	// FIX: Set ALL children's frame to Window's bounds BEFORE any layout calculation
	// This ensures children know their parent's size
	winFrame := w.FrameRect()
	for _, child := range w.children {
		if child == nil {
			continue
		}
		if sfr, ok := child.(interface{ SetFrameRect(graphics.Rect) }); ok {
			sfr.SetFrameRect(winFrame)
		}
	}

	if w.needsLayout && w.layout != nil && len(w.children) > 0 {
		padding := w.padding
		frame := w.FrameRect()
		innerRect := graphics.NewRect(
			frame.X+padding,
			frame.Y+padding,
			frame.Width-padding*2,
			frame.Height-padding*2,
		)
		if l, ok := w.layout.(interface {
			Layout(interface{}, graphics.Rect)
		}); ok {
			l.Layout(w, innerRect)
		}
		w.needsLayout = false
	}

	// Always recurse to children to validate their layouts (TabPanel, etc.)
	for _, child := range w.children {
		if vc, ok := child.(interface{ ValidateLayout() }); ok {
			vc.ValidateLayout()
		}
	}
}

func (w *Window) AsPanel() *Panel {
	return w.panelToExpose
}

// Dispose cleans up all resources held by the Window.
func (w *Window) Dispose() {
	// Clear event handlers
	w.onMouseDown = nil
	w.onMouseUp = nil
	w.onMouseMove = nil
	w.onKeyDown = nil
	w.onFocus = nil
	w.onBlur = nil

	// Clear widget references
	w.mouseDownWidget = nil
	w.hoveredChild = nil
	w.focusedChild = nil
	w.capture = nil

	// Dispose children
	w.ClearChildren()

	// Release the platform resources

	w.panelToExpose = nil

	// Dispose layer manager
	if w.LayerManager != nil {
		w.LayerManager.Dispose()
		w.LayerManager = nil
	}
}

func (w *Window) Layout() interface{} {
	return w.layout
}

func (w *Window) SetLayout(layout interface{}) {
	w.layout = layout
	w.needsLayout = true
}

// RemoveChild removes a child widget from the Window.
// Modifiers reports the modifiers of the LAST INPUT EVENT — which is what an event handler wants.
//
// It was changed at one point to compute from the keys that are down, on the theory that "is shift held"
// wants the current state. That broke the node graph's shift+wheel and ctrl+wheel, which are exactly the
// handlers that read this: a wheel event carries no modifiers, so the pump queries the OS and puts them
// on the event, and the widget reads them back here. Two different questions, one method:
//
//   - "what did THIS event carry" — Modifiers(), which is this
//   - "what is held right now"     — HeldModifiers(), which answers it from the keys
//
// Collapsing them broke a real caller for the sake of a case no caller had, which is the trade this
// project keeps learning: extending the contract is cheaper than redefining it.
func (w *Window) Modifiers() platform.Modifiers {
	return w.lastModifiers
}

// HeldModifiers reports what is held RIGHT NOW, computed from the keys themselves rather than from the
// last event's modifier field.
//
// It is the answer for a paint or a timer asking "is shift down" between two events, which
// `Modifiers()` cannot give: that describes the last event that arrived, and moving the mouse is an event
// carrying no modifiers — so a shift-held UI asked mid-drag would say no.
func (w *Window) HeldModifiers() platform.Modifiers {
	if len(w.keysDown) == 0 {
		return 0
	}
	var m platform.Modifiers
	for k, down := range w.keysDown {
		if !down {
			continue
		}
		switch k {
		case platform.KeyLeftShift, platform.KeyRightShift:
			m |= platform.ModShift
		case platform.KeyLeftCtrl, platform.KeyRightCtrl:
			m |= platform.ModCtrl
		case platform.KeyLeftAlt, platform.KeyRightAlt:
			m |= platform.ModAlt
		case platform.KeyLeftMeta, platform.KeyRightMeta:
			m |= platform.ModMeta
		}
	}
	return m
}

func (w *Window) ClickCount() int {
	return w.lastClickCount
}

func (w *Window) FocusedChild() Widget {
	return w.focusedChild
}

// SetFocusedChild moves keyboard focus to child (nil clears it). It is the only public way in, and it
// goes through the same blur-then-focus path as a click: assigning the field directly left the previous
// widget saying it was still focused, which is how two controls ended up both looking — and acting —
// focused.
func (w *Window) SetFocusedChild(child Widget) {
	w.setFocus(child)
}

func (w *Window) RemoveChild(child Widget) bool {
	for i, c := range w.children {
		if c == child {
			if w.focusedChild != nil && ContainsWidget(child, w.focusedChild) {
				w.setFocus(nil)
			}
			if child != nil {
				if d, ok := child.(interface{ Dispose() }); ok {
					d.Dispose()
				}
				child.SetParent(nil)
			}
			w.children = append(w.children[:i], w.children[i+1:]...)
			w.needsLayout = true
			w.MarkForRedraw()
			return true
		}
	}
	return false
}

// Additional methods needed by other code
func (w *Window) SetOnKeyDown(handler func(keyCode platform.KeyCode, modifiers platform.Modifiers)) *Window {
	w.onKeyDown = handler
	return w
}

// SetOnAction installs the app's handler for a semantic action resolved by the global keymap
// (platform.Event.Actions — window.close, app.exit-fullscreen, or an app-specific binding). It runs
// only when the widgets declined the key, so a shortcut never steals a key a text field needed; it is
// how an app binds a command without the toolkit learning every app's vocabulary.
func (w *Window) SetOnAction(handler func(action string) bool) *Window {
	w.onAction = handler
	return w
}

func (w *Window) SetOnMouseDown(handler func(x, y float32, button int)) *Window {
	w.onMouseDown = handler
	return w
}

func (w *Window) SetOnMouseUp(handler func(x, y float32, button int)) *Window {
	w.onMouseUp = handler
	return w
}

func (w *Window) SetOnMouseMove(handler func(x, y float32)) *Window {
	w.onMouseMove = handler
	return w
}

func (w *Window) SetBackground(ink graphics.Ink) *Window {
	w.background = ink
	return w
}

func (w *Window) SetOpacity(opacity float32) *Window {
	w.opacity = opacity
	return w
}

func (w *Window) SetPadding(padding float32) *Window {
	w.padding = padding
	return w
}

// SetFrameRect overrides BaseWidget to trigger layout on resize.
func (w *Window) SetFrameRect(rect graphics.Rect) {
	w.BaseWidget.SetFrameRect(rect)
	w.markAllNeedsLayout(w)
}

func (w *Window) markAllNeedsLayout(widget Widget) {
	if vc, ok := widget.(interface{ SetNeedsLayout() }); ok {
		vc.SetNeedsLayout()
	}
	if container, ok := widget.(interface{ Children() []Widget }); ok {
		for _, child := range container.Children() {
			w.markAllNeedsLayout(child)
		}
	}
}

// SetOnResize installs the app's resize hook. It receives the CONTENT size, once per
// actual change, so a responsive scene can reflow without polling Width().
func (w *Window) SetOnResize(fn func(width, height float32)) {
	w.onResize = fn
}

// SetWindowSize resizes the OS window to the given content dimensions.
// The status bar size grip calls this to resize the window on drag.
func (w *Window) SetWindowSize(width, height float32) {
	if w.panelToExpose != nil {
		if uw := w.panelToExpose.Window(); uw != nil {
			uw.SetContentRect(graphics.NewRect(0, 0, width, height))
		}
	}
	w.BaseWidget.SetFrameRect(graphics.NewRect(w.x, w.y, width, height))
	w.markAllNeedsLayout(w)
	w.MarkForRedraw()
}

func (w *Window) SetBoundProperty(propName string, value interface{}) {
}

// RenderToImage renders the window contents to an offscreen image at the
// given pixel dimensions. Used by render_demo for screenshot capture.
func (w *Window) RenderToImage(wpx, hpx int) (*graphics.Image, error) {
	pixels := mobilegles.RenderOffscreen(int32(wpx), int32(hpx), func() {
		rect := graphics.NewRect(0, 0, float32(wpx), float32(hpx))
		// Warm-up pass. A widget whose SIZE depends on the layout width corrects
		// itself during its own Paint and then asks its parents to relayout — too
		// late for the frame already being painted, so the captured frame shows a
		// layout one pass behind. ToolBar is the documented case: with wrap on, its
		// height is lineCount * lineHeight from computeLayout(frame.Width), and its
		// own comment warns about "a stale (e.g. wrap-inflated at degenerate
		// first-paint width) frame sticking forever".
		//
		// Measured: without this, rendering main.qml to 1100x750 from a 320x640
		// window differs from rendering it from an 1100x750 window by 12.83
		// mean-delta — which is the whole of MOBILE-PLAN.md P5.8, where the Android
		// host renders at the SCENE's size from a fullscreen window.
		warm := graphics.NewCanvas(graphics.NewRendererDevice(float32(wpx), float32(hpx)))
		w.Paint(warm, rect)
		// Discard the warm-up's pixels: both canvases draw into the same offscreen
		// FBO, so without this clear the captured pass would blend over it.
		mobilegles.ClearScreen(0)

		canvas := graphics.NewCanvas(graphics.NewRendererDevice(float32(wpx), float32(hpx)))
		w.Paint(canvas, rect)
	})
	if pixels == nil {
		return nil, mobilegles.ErrOffscreenFailed()
	}
	// GL readback is bottom-up, and this API returns an IMAGE: every caller
	// writes it straight out as a PNG or hands it to a display, so the flip
	// belongs here rather than in RenderOffscreen — which the widget texture
	// cache also uses, and where a flip would invert every cached widget.
	//
	// It went unnoticed for a long time because `make parity` captures with
	// CaptureAfterFrame, whose caller flips explicitly, and nothing compared
	// RenderToImage's own output against a reference. Found by running the
	// parity corpus on Android (MOBILE-PLAN.md P5.4): the capture came back
	// exactly upside down — mean|delta| 85.3 as-is against 0.0 when flipped.
	return graphics.NewImage(flipVertical(pixels, int32(wpx), int32(hpx)), int32(wpx), int32(hpx)), nil
}

// flipVertical reverses the rows of a bottom-up pixel buffer, so the result is
// top-down. It copies: the input may be a cached FBO readback.
func flipVertical(pix []uint32, w, h int32) []uint32 {
	if w <= 0 || h <= 0 || len(pix) < int(w*h) {
		return pix
	}
	out := make([]uint32, len(pix))
	for y := int32(0); y < h; y++ {
		src := (h - 1 - y) * w
		copy(out[y*w:(y+1)*w], pix[src:src+w])
	}
	return out
}

// TextInputArea reports whether a widget is editing text, and where it is, in
// WINDOW coordinates.
//
// It is the handover between the two halves of on-screen-keyboard support: the
// toolkit knows what has focus, the backend knows how to tell the OS, and neither
// can answer alone. A backend polls this and calls platform.Window.SetTextInput
// when it changes.
//
// "Editing text" is the same test the rune path uses — the focused widget accepts
// OnCharacter — so the keyboard cannot appear for a widget that would then ignore
// what was typed.
func (w *Window) TextInputArea() (bool, graphics.Rect) {
	child := w.focusedChild
	if child == nil {
		return false, graphics.Rect{}
	}
	if _, ok := child.(interface{ OnCharacter(rune) }); !ok {
		return false, graphics.Rect{}
	}
	fr := child.FrameRect()
	x, y := fr.X, fr.Y
	// FrameRect is parent-relative, so walk to the content root.
	if pfr, ok := child.(interface {
		PointToRoot(float32, float32) graphics.Point
	}); ok {
		pt := pfr.PointToRoot(0, 0)
		x, y = pt.X, pt.Y
	}
	// Content space -> window space: add where the content begins. On mobile
	// there is no title bar, so this is (0,0) and the two spaces coincide.
	return true, graphics.NewRect(x+w.contentOriginX, y+w.contentOriginY, fr.Width, fr.Height)
}

// ── Modal Popup System ───────────────────────────────────────────

type ModalPopup struct {
	BackdropLayer *Layer
	ContentLayer  *Layer
	ContentWidget Widget
	window        *Window
	IsModal       bool

	// prevFocus is what held the keyboard before the dialog opened, so dismissing it puts the user
	// back where they were instead of leaving focus on a widget that is no longer on screen.
	prevFocus Widget
}

type ModalBackdrop struct {
	BaseWidget
	parentWindow   *Window
	dismissOnClick bool
}

func newModalBackdrop(win *Window, dismiss bool) *ModalBackdrop {
	return &ModalBackdrop{
		BaseWidget:     *NewBaseWidget(),
		parentWindow:   win,
		dismissOnClick: dismiss,
	}
}

func (mb *ModalBackdrop) ContainsPoint(x, y float32) bool {
	if !mb.Visible() {
		return false
	}
	return true
}

func (mb *ModalBackdrop) OnMouseDown(x, y float32, button int) bool {
	if mb.dismissOnClick && mb.parentWindow != nil {
		mb.parentWindow.DismissTopModal()
	}
	return true
}

func (mb *ModalBackdrop) OnMouseUp(x, y float32, button int) bool { return true }
func (mb *ModalBackdrop) OnMouseMove(x, y float32)                {}

func (mb *ModalBackdrop) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !mb.Visible() {
		return
	}
	canvas.Save()
	overlay := graphics.NewPaint()
	overlay.SetColor(graphics.Color(0xA0000000))
	canvas.DrawRect(rect, overlay)
	canvas.Restore()
}

func (w *Window) ShowModal(content Widget, pw, ph float32) *ModalPopup {
	mp := w.showPopup(content, pw, ph)
	if mp == nil {
		return nil
	}
	mp.IsModal = true
	w.finalizePopup(mp)
	return mp
}

func (w *Window) ShowNonModal(content Widget, pw, ph float32) *ModalPopup {
	mp := w.showPopup(content, pw, ph)
	if mp == nil {
		return nil
	}
	mp.IsModal = false
	w.finalizePopup(mp)
	return mp
}

func (w *Window) showPopup(content Widget, pw, ph float32) *ModalPopup {
	if w.LayerManager == nil {
		return nil
	}
	winFrame := w.FrameRect()
	cx := (winFrame.Width - pw) / 2
	cy := (winFrame.Height - ph) / 2
	if cx < 0 {
		cx = 0
	}
	if cy < 0 {
		cy = 0
	}
	content.SetFrameRect(graphics.NewRect(cx, cy, pw, ph))
	contentLayer := w.LayerManager.AddLayer(modalContentZ, content)
	contentLayer.SetFrameRect(graphics.NewRect(cx, cy, pw, ph))
	if sw, ok := content.(interface{ SetWindow(win *Window) }); ok {
		sw.SetWindow(w)
	}
	mp := &ModalPopup{
		ContentLayer:  contentLayer,
		ContentWidget: content,
		window:        w,
	}
	return mp
}

func (w *Window) finalizePopup(mp *ModalPopup) {
	if mp == nil || w.LayerManager == nil {
		return
	}
	if mp.IsModal {
		winFrame := w.FrameRect()
		backdrop := newModalBackdrop(w, false)
		backdrop.SetFrameRect(winFrame)
		backdropLayer := w.LayerManager.AddLayer(modalBackdropZ, backdrop)
		backdropLayer.Widget = backdrop
		backdropLayer.SetFrameRect(winFrame)
		mp.BackdropLayer = backdropLayer
	}
	mp.prevFocus = w.focusedChild
	w.activeModals = append(w.activeModals, mp)
	// A dialog TAKES the keyboard. It used to leave focus where it was, so the first characters typed
	// after it opened went into the window behind it — into whatever text field happened to be focused
	// — and a keyboard-only user could not reach it at all.
	if fw := FocusableWidgets(mp.ContentWidget); len(fw) > 0 {
		w.setFocus(fw[0])
	} else if mp.ContentWidget != nil {
		w.setFocus(mp.ContentWidget)
	}
	w.MarkForRedraw()
}

func (w *Window) DismissModal(mp *ModalPopup) {
	if mp == nil || w.LayerManager == nil {
		return
	}
	if mp.BackdropLayer != nil {
		w.LayerManager.RemoveLayer(mp.BackdropLayer)
	}
	if mp.ContentLayer != nil {
		w.LayerManager.RemoveLayer(mp.ContentLayer)
	}
	for i, m := range w.activeModals {
		if m == mp {
			w.activeModals = append(w.activeModals[:i], w.activeModals[i+1:]...)
			break
		}
	}
	// Give the keyboard back: focus inside the dialog is now focus on nothing.
	if w.focusedChild != nil && (w.focusedChild == mp.ContentWidget ||
		ContainsWidget(mp.ContentWidget, w.focusedChild)) {
		w.setFocus(mp.prevFocus)
	}
	mp.prevFocus = nil
	w.MarkForRedraw()
}

func (w *Window) DismissTopModal() {
	if len(w.activeModals) == 0 {
		return
	}
	w.DismissModal(w.activeModals[len(w.activeModals)-1])
}

func (w *Window) dismissNonModalPopups() {
	for i := len(w.activeModals) - 1; i >= 0; i-- {
		mp := w.activeModals[i]
		if !mp.IsModal {
			w.DismissModal(mp)
		}
	}
}

const (
	tooltipZ       = 8000
	modalBackdropZ = 8999
	modalContentZ  = 9000
)

// clampPopupPosition adjusts (x, y) so a popup of size (pw, ph) stays within
// the window (winW, winH). If the popup would overflow an edge, it flips to
// the opposite side of the trigger point (triggerX, triggerY). A small margin
// is preserved on all sides.
func clampPopupPosition(x, y, pw, ph, winW, winH, triggerX, triggerY, margin float32) (float32, float32) {
	// Clamp right edge — flip to left of trigger if overflowing
	if x+pw > winW-margin {
		x = triggerX - pw
		if x < margin {
			x = margin
		}
	}
	// Clamp bottom edge — flip above trigger if overflowing
	if y+ph > winH-margin {
		y = triggerY - ph
		if y < margin {
			y = margin
		}
	}
	// Clamp left/top edges
	if x < margin {
		x = margin
	}
	if y < margin {
		y = margin
	}
	return x, y
}

// ── Tooltip helpers ──────────────────────────────────────────────

func (w *Window) cancelTooltip() {
	if w.tooltipTimer != nil {
		w.tooltipTimer.Stop()
		w.tooltipTimer = nil
	}
	if w.tooltipLayer != nil {
		w.LayerManager.RemoveLayer(w.tooltipLayer)
		w.tooltipLayer = nil
	}
	w.tooltipWidget = nil
	w.tooltipText = ""
}

func (w *Window) showTooltip(x, y float32, text string) {
	if w.LayerManager == nil || text == "" {
		return
	}
	w.cancelTooltip()

	font := theme.DefaultTheme.BaseFont
	if font == nil {
		font = graphics.SystemFont
	}
	textWidth := font.SimpleWidth(text)
	fontHeight := font.LineHeight()
	padding := float32(6)
	tw := textWidth + padding*2
	th := fontHeight + padding*2

	winFrame := w.FrameRect()
	edgeMargin := float32(8)
	startX := x + 12
	startY := y + 12
	startX, startY = clampPopupPosition(startX, startY, tw, th, winFrame.Width, winFrame.Height, x, y, edgeMargin)

	w.tooltipLayer = w.LayerManager.AddLayerWithPaint(tooltipZ, func(canvas *graphics.Canvas, rect graphics.Rect) {
		ascent := font.Baseline()
		tx := startX
		ty := startY
		bgRect := graphics.NewRect(tx, ty, tw, th)
		bgPaint := graphics.NewPaint()
		bgPaint.SetColor(graphics.Color(0xFF333333))
		canvas.DrawRoundedRect(bgRect, 4, 4, bgPaint)
		borderPaint := graphics.NewPaint()
		borderPaint.SetColor(graphics.Color(0xFF555555))
		canvas.DrawRoundedRect(bgRect, 4, 4, borderPaint)
		textPaint := graphics.NewPaint()
		textPaint.SetColor(graphics.Color(0xFFE0E0E0))
		canvas.DrawSimpleString(text, tx+padding, ty+padding+ascent, font, textPaint)
	})
	w.tooltipLayer.X = 0
	w.tooltipLayer.Y = 0
	w.tooltipLayer.Width = 2000
	w.tooltipLayer.Height = 2000
	w.MarkForRedraw()
}

func (w *Window) updateTooltipPosition(x, y float32) {
	if w.tooltipLayer != nil {
		font := theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
		padding := float32(6)
		tw := font.SimpleWidth(w.tooltipText) + padding*2
		th := font.LineHeight() + padding*2

		winFrame := w.FrameRect()
		edgeMargin := float32(8)
		startX := x + 12
		startY := y + 12
		startX, startY = clampPopupPosition(startX, startY, tw, th, winFrame.Width, winFrame.Height, x, y, edgeMargin)

		w.tooltipLayer.X = startX
		w.tooltipLayer.Y = startY
		w.MarkForRedraw()
	}
}

// Paint renders the Window and all children
// auditPaintWindow walks the subtree a window is about to paint and reports any
// widget that resolves to a DIFFERENT window. A direct-child check is not enough:
// contamination nested deeper would pass it. Depth-capped so a cycle cannot hang
// the frame.
func auditPaintWindow(w *Window, node Widget, depth int) {
	if os.Getenv("MEUI_PAINT_AUDIT") == "" || node == nil || depth > 14 {
		return
	}
	if cw, ok := node.(interface{ Window() *Window }); ok {
		if got := cw.Window(); got != w {
			log.Printf("[paint] CROSS-WINDOW depth=%d %T Window()=%p painting=%p", depth, node, got, w)
		}
	}
	if c, ok := node.(interface{ Children() []Widget }); ok {
		for _, ch := range c.Children() {
			auditPaintWindow(w, ch, depth+1)
		}
	}
}

// Window returns the window itself.
//
// Without this, the promoted BaseWidget.Window() wins, and for a *Window it walks
// Parent() — which a Window does not have — so it returns nil. Any caller doing
// .Window() on a window then gets nil and can fall back to process-global state,
// which is how a window ends up using another window's values.
func (w *Window) Window() *Window { return w }

func (w *Window) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if w != nil && canvas != nil {
		w.lastDevice = canvas.Device()
	}
	auditPaintWindow(w, w, 0)
	// Lay out anything that has been marked, BEFORE painting it.
	//
	// Without this a widget that changes its own size never moves: its ancestors are
	// marked needing layout, but nothing asks them for it until the next HIT TEST, and a
	// repaint triggered by the change paints the OLD frames. A Collapsible's toggle is
	// the case that exposed it — the state flipped and the row did not grow — and the
	// symptom is indistinguishable from the widget not working.
	if w.NeedsLayout() || w.childrenNeedLayout() {
		w.ValidateLayout()
	}
	// Update window frame to match the rect from the platform layer (without triggering redraw)
	w.x = rect.X
	w.y = rect.Y
	w.width = rect.Width
	w.height = rect.Height

	// Draw background (theme-driven — was hardcoded RGB(50,50,50))
	bgPaint := graphics.NewPaint()
	bgPaint.SetColor(theme.DefaultTheme.Background)
	canvas.DrawRect(rect, bgPaint)

	// Update children's frame rects to fill the window (without triggering redraw)
	for _, child := range w.children {
		if child != nil {
			// Paint-time audit: every child this window paints must resolve back
			// to this window. A child that resolves elsewhere means one window is
			// painting another window's widgets, which is the reported
			// contamination (Accounts showing the Analysis chart's panels).
			if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
				fmt.Printf("[windowPaintChild] %T (%p)\n", child, child)
			}
			if bw, ok := child.(interface{ SetFrameRectDirect(graphics.Rect) }); ok {
				bw.SetFrameRectDirect(rect)
			} else if bw, ok := child.(*BaseWidget); ok {
				if rect.Width != bw.width || rect.Height != bw.height {
					bw.needsLayout = true
				}
				bw.x = rect.X
				bw.y = rect.Y
				bw.width = rect.Width
				bw.height = rect.Height
			} else {
				child.SetFrameRect(rect)
			}
			child.Paint(canvas, rect)
		}
	}

	// Update modal backdrop frames to match window size
	for _, mp := range w.activeModals {
		if mp.BackdropLayer != nil {
			mp.BackdropLayer.SetFrameRect(rect)
			if mp.BackdropLayer.Widget != nil {
				mp.BackdropLayer.Widget.SetFrameRect(rect)
			}
		}
	}

	// Render LayerManager layers (dropdowns, popups). Skipped when the
	// host paints them externally (after the title bar, so overlays
	// cover the chrome — SetLayerPaintExternal).
	if w.LayerManager != nil && !w.layerPaintExternal {
		w.LayerManager.Paint(canvas)
	}

	// Draw global drag overlay on top of everything
	if GlobalOverlay != nil && GlobalOverlay.Image != nil {
		orect := graphics.NewRect(GlobalOverlay.X, GlobalOverlay.Y, GlobalOverlay.W, GlobalOverlay.H)
		canvas.DrawImageInRect(GlobalOverlay.Image, orect, nil, nil)
	}
}

// ShowContextMenuPositioned shows a registered context menu at the given position.

// SetContentOrigin records where the content tree's (0,0) sits in window
// space. It is non-zero when the host reserves a strip for window chrome — the
// custom title bar — and it is the *only* bar knowledge the toolkit needs:
// widgets keep speaking their own tree's coordinates.
func (w *Window) SetContentOrigin(x, y float32) {
	w.contentOriginX, w.contentOriginY = x, y
}

// ContentOrigin returns the content tree's origin in window space.
func (w *Window) ContentOrigin() (float32, float32) {
	return w.contentOriginX, w.contentOriginY
}

// chromeWidget is implemented by widgets hosted outside the content tree (the
// title bar's node): their coordinates start at the window's own origin, so
// they sit *above* the content origin.
type chromeWidget interface{ IsWindowChrome() bool }

// layerOriginFor returns the content-space origin of the tree `from` belongs
// to, so a layer created from a widget-local point lands in the space layers
// are painted in.
//
// A content widget belongs to the content tree, whose origin is the content
// origin itself; a widget hosted by the window's title bar belongs to the
// chrome, which sits a title bar above it. Neither the widget nor its caller
// computes this: the bar is chrome, exactly as a real window decoration is,
// and only the host declares where the content begins.
func (w *Window) layerOriginFor(from Widget) graphics.Point {
	for cur := from; cur != nil; {
		if cw, ok := cur.(chromeWidget); ok && cw.IsWindowChrome() {
			// Chrome space is window space: content y = window y - origin.
			return graphics.NewPoint(-w.contentOriginX, -w.contentOriginY)
		}
		p, ok := cur.(interface{ Parent() Widget })
		if !ok {
			break
		}
		cur = p.Parent()
	}
	return graphics.NewPoint(0, 0)
}

// AddLayerFor places a layer at a point given in `from`'s own local
// coordinates and returns it. Callers never convert spaces themselves: the
// window adds the position within the tree and the tree's origin.
func (w *Window) AddLayerFor(from Widget, x, y float32, paint func(canvas *graphics.Canvas, rect graphics.Rect)) *Layer {
	if w.LayerManager == nil || from == nil {
		return nil
	}
	pt := graphics.NewPoint(x, y)
	if pfr, ok := from.(interface {
		PointToRoot(float32, float32) graphics.Point
	}); ok {
		pt = pfr.PointToRoot(x, y)
	}
	origin := w.layerOriginFor(from)
	layer := w.LayerManager.AddLayerWithPaint(10000, paint, pt.X+origin.X, pt.Y+origin.Y)
	if layer != nil {
		// Paint-only layer: no widget to route input to, but its OPENER has to be told when it
		// closes (a combo box's dropdown, a menu drawn by a callback), so it is kept as the Owner.
		layer.Owner = from
	}
	return layer
}

// SetLayerPaintExternal switches layer painting off inside Window.Paint:
// the host paints them via PaintLayerOverlay AFTER the title bar, so
// dropdowns/popups overlay the chrome instead of being overpainted by it.
func (w *Window) SetLayerPaintExternal() { w.layerPaintExternal = true }

// PaintLayerOverlay paints all layers translated by translateY (the
// content-space offset the layers' coordinates are expressed in).
func (w *Window) PaintLayerOverlay(canvas *graphics.Canvas, translateY float32) {
	if w.LayerManager == nil {
		return
	}
	canvas.Save()
	canvas.Translate(0, translateY)
	w.LayerManager.Paint(canvas)
	canvas.Restore()
}

// The menu is displayed as a non-modal popup layer. The id must have been registered
// via RegisterNamedWidget during QML loading.
func (w *Window) ShowContextMenuPositioned(id string, x, y float32) {
	cm := GetNamedWidget(id)
	if cm == nil {
		return
	}
	// First dismiss any existing instance
	w.DismissContextMenu(id)
	// Make visible and set position
	if sv, ok := cm.(interface{ SetVisible(bool) }); ok {
		sv.SetVisible(true)
	}

	// Compute menu dimensions from its items
	pw := float32(200)
	ph := float32(200)
	if ps, ok := cm.(interface{ PreferredSize() (float32, float32) }); ok {
		w, h := ps.PreferredSize()
		if w > 0 {
			pw = w
		}
		if h > 0 {
			ph = h
		}
	}

	winFrame := w.FrameRect()
	edgeMargin := float32(8)
	x, y = clampPopupPosition(x, y, pw, ph, winFrame.Width, winFrame.Height, x, y, edgeMargin)

	cm.SetFrameRect(graphics.NewRect(x, y, pw, ph))
	if w.LayerManager == nil {
		return
	}
	w.ctxMenuID = id

	// Auto-dismiss when a menu item is selected
	if h, ok := cm.(ContextMenuHoster); ok {
		h.SetOnDismiss(func() {
			w.DismissContextMenu(id)
		})
	}

	// Add a transparent backdrop that dismisses the menu on click
	backdrop := &contextMenuBackdrop{
		parentWindow: w,
		dismissFn: func() {
			w.DismissContextMenu(id)
		},
	}
	backdrop.SetFrameRect(winFrame)
	w.ctxMenuBackdrop = w.LayerManager.AddLayer(modalBackdropZ, backdrop)
	w.ctxMenuBackdrop.SetFrameRect(winFrame)

	// Add the context menu content layer
	contentLayer := w.LayerManager.AddLayer(modalContentZ, cm)
	contentLayer.SetFrameRect(graphics.NewRect(x, y, pw, ph))
	cm.MarkForRedraw()
	w.MarkForRedraw()
}

// contextMenuBackdrop is a transparent overlay that captures clicks to dismiss a context menu.
type contextMenuBackdrop struct {
	BaseWidget
	parentWindow *Window
	dismissFn    func()
}

func (b *contextMenuBackdrop) ContainsPoint(x, y float32) bool {
	return true
}

func (b *contextMenuBackdrop) OnMouseDown(x, y float32, button int) bool {
	if b.dismissFn != nil {
		b.dismissFn()
	}
	// Consume the click: dismissing must not also re-trigger the widget
	// underneath (a menubar title would dismiss-then-reopen on the same
	// click, so the dropdown never closed).
	return true
}

func (b *contextMenuBackdrop) OnMouseUp(x, y float32, button int) bool { return true }

// MenuStripHost marks widgets (menu bars) that must keep receiving
// pointer moves while a popup menu is open: hovering another title
// hover-switches the open menu.
type MenuStripHost interface{ IsMenuStrip() bool }

// menuStripOf returns the menu bar at or above w, if any.
func menuStripOf(w Widget) Widget {
	for ; w != nil; w = w.Parent() {
		if ms, ok := w.(MenuStripHost); ok && ms.IsMenuStrip() {
			return w
		}
	}
	return nil
}

// OnMouseMove forwards moves over a MENU BAR (main or embedded — the
// old hardcoded y<=28 strip only covered the window-level one, so the
// embedded bar in the Menu & Context tab never hover-switched): without
// this the full-window backdrop eats every move while a menu is open.
// Elsewhere the popup owns the pointer — other widgets stay inert.
func (b *contextMenuBackdrop) OnMouseMove(x, y float32) {
	if b.parentWindow == nil {
		return
	}
	strip := menuStripOf(b.parentWindow.hitTest(x, y))
	if strip == nil {
		return
	}
	localX, localY := x, y
	if pfr, ok := strip.(interface {
		PointFromRoot(float32, float32) graphics.Point
	}); ok {
		pt := pfr.PointFromRoot(x, y)
		localX, localY = pt.X, pt.Y
	}
	if el, ok := strip.(EventListener); ok {
		el.OnMouseMove(localX, localY)
	}
}

func (b *contextMenuBackdrop) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	// Transparent — no drawing needed
}

// ShowPopupContextMenu shows a standalone context menu widget at the given position.
// The menu is displayed as a non-modal popup layer. The widget should be a *ContextMenu
// or any widget that implements PreferredSize() and ContextMenuHoster.
func (w *Window) ShowPopupContextMenu(cm Widget, x, y float32) {
	w.DismissPopupContextMenu()
	if sv, ok := cm.(interface{ SetVisible(bool) }); ok {
		sv.SetVisible(true)
	}

	pw := float32(200)
	ph := float32(200)
	if ps, ok := cm.(interface{ PreferredSize() (float32, float32) }); ok {
		cw, ch := ps.PreferredSize()
		if cw > 0 {
			pw = cw
		}
		if ch > 0 {
			ph = ch
		}
	}

	winFrame := w.FrameRect()
	edgeMargin := float32(8)
	x, y = clampPopupPosition(x, y, pw, ph, winFrame.Width, winFrame.Height, x, y, edgeMargin)

	cm.SetFrameRect(graphics.NewRect(x, y, pw, ph))
	if w.LayerManager == nil {
		return
	}

	backdrop := &contextMenuBackdrop{
		parentWindow: w,
		dismissFn: func() {
			if h, ok := cm.(ContextMenuHoster); ok {
				h.OnExternalDismiss()
			}
			w.DismissPopupContextMenu()
		},
	}
	backdrop.SetFrameRect(winFrame)
	w.ctxMenuBackdrop = w.LayerManager.AddLayer(modalBackdropZ, backdrop)
	w.ctxMenuBackdrop.SetFrameRect(winFrame)

	w.ctxMenuContent = w.LayerManager.AddLayer(modalContentZ, cm)
	w.ctxMenuContent.SetFrameRect(graphics.NewRect(x, y, pw, ph))
	cm.MarkForRedraw()
	w.MarkForRedraw()
}

// DismissPopupContextMenu dismisses the current popup context menu.
func (w *Window) DismissPopupContextMenu() {
	if w.LayerManager == nil {
		return
	}
	if w.ctxMenuBackdrop != nil {
		w.LayerManager.RemoveLayer(w.ctxMenuBackdrop)
		w.ctxMenuBackdrop = nil
	}
	if w.ctxMenuContent != nil {
		w.LayerManager.RemoveLayer(w.ctxMenuContent)
		w.ctxMenuContent = nil
	}
	w.ctxMenuID = ""
	w.MarkForRedraw()
}

// DismissContextMenu dismisses a named context menu popup.
func (w *Window) DismissContextMenu(id string) {
	cm := GetNamedWidget(id)
	if cm == nil {
		return
	}
	if w.LayerManager == nil {
		return
	}
	w.LayerManager.RemoveLayerByWidget(cm)
	if w.ctxMenuBackdrop != nil {
		w.LayerManager.RemoveLayer(w.ctxMenuBackdrop)
		w.ctxMenuBackdrop = nil
	}
	w.ctxMenuID = ""
	if sv, ok := cm.(interface{ SetVisible(bool) }); ok {
		sv.SetVisible(false)
	}
	w.MarkForRedraw()
}

// ── Input sink (platform.Sink) ───────────────────────────────────
//
// The platform layer (SDL on desktop, EGL/NativeActivity on Android later)
// delivers every input event through this single dispatch point. It routes
// to the same handle* machinery the reference-style panel callbacks use, so
// widget behavior is identical whether events arrive live or replayed.

// CursorAt implements meuishim.CursorSource — the hover cursor for the
// widget under (x, y), via the standard hit-test walk.
func (w *Window) CursorAt(x, y float32) *platform.Cursor {
	return w.handleUpdateCursor(x, y)
}

// SetOnBack installs the handler for the OS back gesture. It returns true when the
// app handled it — it popped its own navigation — and false to let the window close,
// which is what happens at the root of the stack. A nil handler always closes.
//
// This is the app's half of Android's back contract. Without it the gesture reaches
// nothing and the system closes the app, which is what happened before the back event
// existed at all.
func (w *Window) SetOnBack(fn func() bool) { w.onBack = fn }

// HandleEvent implements platform.Sink. Returns true when consumed.
func (w *Window) HandleEvent(ev *platform.Event) bool {
	if ev == nil {
		return false
	}
	switch ev.Type {
	case platform.EventBack:
		// Consumed only if the app handled it; false lets the adapter close the
		// window, which is the platform's behaviour at the root of the stack.
		if w.onBack != nil && w.onBack() {
			w.MarkForRedraw()
			return true
		}
		return false

	case platform.EventMouseDown:
		w.lastModifiers = ev.Mods
		w.lastClickCount = ev.ClickCount
		return w.handleMouseDown(ev.Pos.X, ev.Pos.Y, ev.Button)

	case platform.EventMouseUp:
		w.lastModifiers = ev.Mods
		return w.handleMouseUp(ev.Pos.X, ev.Pos.Y, ev.Button)

	case platform.EventMouseMove:
		w.lastModifiers = ev.Mods
		w.handleMouseMove(ev.Pos.X, ev.Pos.Y)
		return false

	case platform.EventMouseDrag:
		w.lastModifiers = ev.Mods
		return w.handleMouseDrag(ev.Pos.X, ev.Pos.Y, ev.Button)

	case platform.EventMouseWheel:
		w.lastModifiers = ev.Mods
		return w.handleMouseWheel(ev.Pos.X, ev.Pos.Y, ev.DeltaX, ev.DeltaY)

	case platform.EventKeyDown:
		w.lastModifiers = ev.Mods
		w.keysDown[ev.Key] = true
		handled := w.dispatchKeyDown(ev.Key, ev.Mods, ev.Repeat)
		// Semantic actions come LAST, and never on a repeat: a shortcut is a command, and a key held
		// down must not fire it over and over. The adapter puts the keymap's matches on the event
		// (EVENT ACTIONS), which is how an app binds its own commands without hardcoding keys — and
		// nothing in the toolkit used to read that field at all.
		if !handled && !ev.Repeat && len(ev.Actions) > 0 && w.onAction != nil {
			for _, action := range ev.Actions {
				if w.onAction(action) {
					w.MarkForRedraw()
					handled = true
					break
				}
			}
		}
		return handled

	case platform.EventKeyUp:
		w.lastModifiers = ev.Mods
		delete(w.keysDown, ev.Key)
		// A KEY-UP IS DELIVERED. It used to be dropped with a note saying so, and that is not a small gap:
		// a widget cannot tell "the user let go of the arrow" from "the arrow was never pressed", which is
		// the whole of a key-held interaction, a slider scrub, a spinner, or a canvas pan.
		if w.focusedChild != nil {
			if ku, ok := w.focusedChild.(interface {
				OnKeyUp(platform.KeyCode, platform.Modifiers) bool
			}); ok {
				if ku.OnKeyUp(ev.Key, ev.Mods) {
					w.MarkForRedraw()
					return true
				}
			}
			for p := parentOf(w.focusedChild); p != nil && p != Widget(w); p = parentOf(p) {
				if ku, ok := p.(interface {
					OnKeyUp(platform.KeyCode, platform.Modifiers) bool
				}); ok {
					if ku.OnKeyUp(ev.Key, ev.Mods) {
						w.MarkForRedraw()
						return true
					}
				}
			}
		}
		if w.LayerManager != nil && w.LayerManager.HandleKeyUp(ev.Key, ev.Mods) {
			w.MarkForRedraw()
			return true
		}
		return false

	case platform.EventTextEditing:
		// The IME's in-progress text. It is NOT input yet: the focused widget shows it (underlined)
		// and replaces it as the composition changes; the committed characters arrive as
		// EventRuneTyped. A widget that cannot compose declines, and the top layer gets its chance.
		if w.focusedChild != nil {
			if th, ok := w.focusedChild.(interface {
				OnComposition(text string, start, length int) bool
			}); ok {
				if th.OnComposition(ev.Text, ev.CompStart, ev.CompLength) {
					w.MarkForRedraw()
					return true
				}
			}
		}
		if w.LayerManager != nil && w.LayerManager.HandleTextEditing(ev.Text, ev.CompStart, ev.CompLength) {
			w.MarkForRedraw()
			return true
		}
		return true

	case platform.EventRuneTyped:
		if w.focusedChild != nil {
			if th, ok := w.focusedChild.(interface{ OnCharacter(rune) }); ok {
				th.OnCharacter(ev.Rune)
				w.MarkForRedraw()
				return true
			}
		}
		if w.LayerManager != nil && w.LayerManager.HandleRune(ev.Rune) {
			w.MarkForRedraw()
			return true
		}
		return true

	case platform.EventResize:
		// The draw path re-reads the frame rect and re-lays-out every
		// frame; resizing the widget tree here is enough.
		oldW, oldH := w.width, w.height
		w.SetFrameRect(graphics.NewRect(w.x, w.y, float32(ev.W), float32(ev.H)))
		if w.onResize != nil && (oldW != float32(ev.W) || oldH != float32(ev.H)) {
			w.onResize(float32(ev.W), float32(ev.H))
		}
		// A window whose floor was DERIVED from its size follows a resize; one the app configured
		// explicitly keeps its setting. Either way it is this window's floor, not the process's.
		if w.touchProfileAuto {
			w.setDerivedTouchProfile(platform.FormFactorFor(float32(ev.W), float32(ev.H), w.Scale()) != platform.FormFactorDesktop)
		}
		w.MarkForRedraw()
		return false
	}
	return false
}

// eventDebug logs input-dispatch details when MEUI_EVENT_DEBUG is set.
func eventDebug(format string, args ...any) {
	if os.Getenv("MEUI_EVENT_DEBUG") != "" {
		log.Printf("[meui-dispatch] "+format, args...)
	}
}

// Device returns the device the window last painted through, or nil if it has
// not painted yet. Offscreen work outside a paint pass uses this.
func (w *Window) Device() graphics.Device {
	if w == nil {
		return nil
	}
	return w.lastDevice
}

// bubblePoint converts a root-space point into w's own coordinates for a bubbled
// press. A widget that does not implement PointFromRoot is not a bubbling target.
func bubblePoint(w Widget, x, y float32) graphics.Point {
	if pfr, ok := w.(interface {
		PointFromRoot(float32, float32) graphics.Point
	}); ok {
		return pfr.PointFromRoot(x, y)
	}
	return graphics.Point{X: x, Y: y}
}
