package meui

import (
	"fmt"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
	"log"
	"os"
)

type Layer struct {
	X       float32
	Y       float32
	Width   float32
	Height  float32
	ZIndex  int
	Visible bool
	Widget  Widget
	Paint   func(canvas *graphics.Canvas, rect graphics.Rect)

	// Owner is the widget that opened a PAINT-ONLY layer (a combo box's dropdown, a
	// menu drawn by a callback). Such a layer has no Widget to route input to, but it
	// still has to be TOLD when it is closed — otherwise a dropdown removed by Escape
	// leaves its owner believing the list is still open. It is never a hit target.
	Owner Widget
}

func (l *Layer) FrameRect() graphics.Rect {
	return graphics.NewRect(l.X, l.Y, l.Width, l.Height)
}

func (l *Layer) SetFrameRect(rect graphics.Rect) {
	l.X = rect.X
	l.Y = rect.Y
	l.Width = rect.Width
	l.Height = rect.Height
}

func (l *Layer) ContainsPoint(x, y float32) bool {
	if !l.Visible {
		return false
	}
	if x >= l.X && x <= l.X+l.Width && y >= l.Y && y <= l.Y+l.Height {
		return true
	}
	// Also check the widget's own ContainsPoint (for extended hit areas like submenus)
	if l.Widget != nil {
		return l.Widget.ContainsPoint(x-l.X, y-l.Y)
	}
	return false
}

type LayerManager struct {
	Layers []*Layer
	// owner is the window whose paint pass draws these layers. It exists so a
	// layer attached to the WRONG manager is detectable: a layer added here is
	// painted by this window every frame, forever, so a widget that belongs to
	// another window would be drawn by this one.
	owner *Window
}

// layerDebug reports every layer attachment with the manager's owner and the
// window the widget itself belongs to. A mismatch is the cross-window
// contamination: a widget of one window painted by another.
func layerDebug(lm *LayerManager, what string, widget Widget) {
	if os.Getenv("MEUI_LAYER_DEBUG") == "" {
		return
	}
	owner, own := "none", "none"
	if lm != nil && lm.owner != nil {
		owner = fmt.Sprintf("%p", lm.owner)
	}
	if widget != nil {
		if ww, ok := widget.(interface{ Window() *Window }); ok {
			if w := ww.Window(); w != nil {
				own = fmt.Sprintf("%p", w)
			}
		}
	}
	flag := ""
	if own != "none" && owner != own {
		flag = "  <-- MISMATCH: widget belongs to another window"
	}
	log.Printf("[layer] %s manager-owner=%s widget-window=%s%s", what, owner, own, flag)
}

func NewLayerManager() *LayerManager {
	return &LayerManager{
		Layers: []*Layer{},
	}
}

func (lm *LayerManager) AddLayer(zIndex int, widget Widget) *Layer {
	layerDebug(lm, "AddLayer", widget)
	layer := &Layer{
		ZIndex:  zIndex,
		Widget:  widget,
		Visible: true,
	}

	// A widget hosted by a layer is not in the window's child tree, so its parent
	// chain ends in nothing: Window() and findWindow() return nil. That is why a
	// ComboBox inside a modal could not open its dropdown — the dropdown is a
	// window layer, and the combo could not reach the window to create it
	// ("the combobox popup does not show"). Point the hosted widget at the window
	// when it has no parent. SetParent only stores the link (it does not register
	// a child), and the window's frame starts at the origin, so coordinate math
	// is unchanged.
	if lm.owner != nil && widget != nil && widget.Parent() == nil {
		widget.SetParent(lm.owner)
	}

	inserted := false
	for i, l := range lm.Layers {
		if l.ZIndex > zIndex {
			lm.Layers = append(lm.Layers[:i], append([]*Layer{layer}, lm.Layers[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		lm.Layers = append(lm.Layers, layer)
	}

	return layer
}

func (lm *LayerManager) RemoveLayer(layer *Layer) {
	for i, l := range lm.Layers {
		if l == layer {
			l.Widget = nil
			lm.Layers = append(lm.Layers[:i], lm.Layers[i+1:]...)
			break
		}
	}
}

func (lm *LayerManager) RemoveLayerByWidget(widget Widget) {
	for i, l := range lm.Layers {
		if l.Widget == widget {
			l.Widget = nil
			lm.Layers = append(lm.Layers[:i], lm.Layers[i+1:]...)
			break
		}
	}
}

func (lm *LayerManager) Dispose() {
	for _, l := range lm.Layers {
		l.Widget = nil
	}
	lm.Layers = nil
}

func (lm *LayerManager) MoveToFront(layer *Layer) {
	for i, l := range lm.Layers {
		if l == layer {
			lm.Layers = append(lm.Layers[:i], lm.Layers[i+1:]...)
			break
		}
	}
	maxZ := 0
	for _, l := range lm.Layers {
		if l.ZIndex > maxZ {
			maxZ = l.ZIndex
		}
	}
	layer.ZIndex = maxZ + 1
	lm.Layers = append(lm.Layers, layer)
}

func (lm *LayerManager) AddLayerWithPaint(zIndex int, paint func(canvas *graphics.Canvas, rect graphics.Rect), pos ...float32) *Layer {
	layerDebug(lm, "AddLayerWithPaint", nil)
	layer := &Layer{
		ZIndex:  zIndex,
		Visible: true,
		Paint:   paint,
	}
	if len(pos) >= 2 {
		layer.X, layer.Y = pos[0], pos[1]
	}

	inserted := false
	for i, l := range lm.Layers {
		if l.ZIndex > zIndex {
			lm.Layers = append(lm.Layers[:i], append([]*Layer{layer}, lm.Layers[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		lm.Layers = append(lm.Layers, layer)
	}

	return layer
}

func (lm *LayerManager) HitTest(x, y float32) *Layer {
	for i := len(lm.Layers) - 1; i >= 0; i-- {
		layer := lm.Layers[i]
		if layer.ContainsPoint(x, y) {
			return layer
		}
	}
	return nil
}

func (lm *LayerManager) HandleMouseDown(x, y float32, button int) bool {
	layer := lm.HitTest(x, y)
	if layer != nil && layer.Widget != nil {
		lx, ly := x-layer.X, y-layer.Y
		if mw, ok := layer.Widget.(interface {
			HandleMouseDown(x, y float32, button int) bool
		}); ok {
			return mw.HandleMouseDown(lx, ly, button)
		}
		if mw, ok := layer.Widget.(interface {
			OnMouseDown(x, y float32, button int) bool
		}); ok {
			return mw.OnMouseDown(lx, ly, button)
		}
	}
	return false
}

func (lm *LayerManager) HandleMouseUp(x, y float32, button int) bool {
	layer := lm.HitTest(x, y)
	if layer != nil && layer.Widget != nil {
		lx, ly := x-layer.X, y-layer.Y
		if mw, ok := layer.Widget.(interface {
			HandleMouseUp(x, y float32, button int) bool
		}); ok {
			return mw.HandleMouseUp(lx, ly, button)
		}
		if mw, ok := layer.Widget.(interface {
			OnMouseUp(x, y float32, button int) bool
		}); ok {
			return mw.OnMouseUp(lx, ly, button)
		}
	}
	return false
}

func (lm *LayerManager) HandleMouseMove(x, y float32) bool {
	layer := lm.HitTest(x, y)
	if layer != nil && layer.Widget != nil {
		lx, ly := x-layer.X, y-layer.Y
		if mw, ok := layer.Widget.(interface{ HandleMouseMove(x, y float32) bool }); ok {
			return mw.HandleMouseMove(lx, ly)
		}
		if mw, ok := layer.Widget.(interface{ OnMouseMove(x, y float32) }); ok {
			mw.OnMouseMove(lx, ly)
			return true
		}
	}
	return false
}

func (lm *LayerManager) HandleMouseWheel(x, y float32, dx, dy float32) bool {
	layer := lm.HitTest(x, y)
	if layer != nil && layer.Widget != nil {
		lx, ly := x-layer.X, y-layer.Y
		if mw, ok := layer.Widget.(interface {
			OnMouseWheel(float32, float32, float32, float32) bool
		}); ok {
			return mw.OnMouseWheel(lx, ly, dx, dy)
		}
	}
	return false
}

// HandleKeyDown dispatches key events to the topmost layer widget.
func (lm *LayerManager) HandleKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	for i := len(lm.Layers) - 1; i >= 0; i-- {
		layer := lm.Layers[i]
		if layer != nil && layer.Visible && layer.Widget != nil {
			if kh, ok := layer.Widget.(interface {
				OnKeyDown(platform.KeyCode, platform.Modifiers) bool
			}); ok {
				if kh.OnKeyDown(keyCode, mod) {
					return true
				}
			}
		}
	}
	return false
}

// HandleKeyUp dispatches a key release to the topmost layer widget.
//
// It exists because the window now DELIVERS key-ups: a popup that acted on a key press needs to know the
// key was let go, and the layer is where a popup lives. The window dropping key-ups — which it did, with
// a note saying "no key-up behavior yet" — meant a layer could never observe a release.
func (lm *LayerManager) HandleKeyUp(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	// Topmost-first, like HandleKeyDown: a release goes to the same widget the press did, and the first
	// one to claim it keeps it.
	for i := len(lm.Layers) - 1; i >= 0; i-- {
		layer := lm.Layers[i]
		if layer != nil && layer.Visible && layer.Widget != nil {
			if ku, ok := layer.Widget.(interface {
				OnKeyUp(platform.KeyCode, platform.Modifiers) bool
			}); ok {
				if ku.OnKeyUp(keyCode, mod) {
					return true
				}
			}
		}
	}
	return false
}

// HandleTextEditing dispatches an IME composition to the topmost layer widget, so a dialog opened
// without focus can still show its preedit.
func (lm *LayerManager) HandleTextEditing(text string, start, length int) bool {
	for i := len(lm.Layers) - 1; i >= 0; i-- {
		layer := lm.Layers[i]
		if layer != nil && layer.Visible && layer.Widget != nil {
			if th, ok := layer.Widget.(interface {
				OnComposition(string, int, int) bool
			}); ok {
				if th.OnComposition(text, start, length) {
					return true
				}
			}
		}
	}
	return false
}

// HandleRune dispatches character input to the topmost layer widget.
func (lm *LayerManager) HandleRune(ch rune) bool {
	for i := len(lm.Layers) - 1; i >= 0; i-- {
		layer := lm.Layers[i]
		if layer != nil && layer.Visible && layer.Widget != nil {
			if th, ok := layer.Widget.(interface {
				OnCharacter(rune)
			}); ok {
				th.OnCharacter(ch)
				return true
			}
		}
	}
	return false
}

func (lm *LayerManager) Paint(canvas *graphics.Canvas) {
	for _, layer := range lm.Layers {
		if layer != nil && layer.Visible {
			layerRect := layer.FrameRect()
			if layer.Paint != nil {
				canvas.Save()
				layer.Paint(canvas, layerRect)
				canvas.Restore()
			} else if layer.Widget != nil {
				layer.Widget.Paint(canvas, layerRect)
			}
		}
	}
}

// CloseTopLayer closes the topmost VISIBLE layer and reports whether it closed one.
//
// It is what Escape does when no drag is in progress, and it exists because a layer with no focusable
// child could otherwise not be dismissed by keyboard at all: the window's key handling walked the FOCUSED
// widget and the children, so a popup whose contents were all non-focusable had no path to the key. A
// layer is also the toolkit's own drag ghost, so this is what removes a drag's ghost if the drag is ended
// some other way.
//
// The widget is told through OnClosed so a popup can release whatever it owns — a combobox that
// opened a list, a menu that grabbed the pointer — and so can a PAINT-ONLY layer's Owner, which has
// no widget to route to but still has to know its dropdown is gone.
//
// The layer is removed FIRST, then the owner is told: a widget's OnClosed usually removes the layer
// itself, and notifying before the splice let that call shift the slice out from under the loop.
func (lm *LayerManager) CloseTopLayer() bool {
	for i := len(lm.Layers) - 1; i >= 0; i-- {
		layer := lm.Layers[i]
		if layer == nil || !layer.Visible {
			continue
		}
		lm.Layers = append(lm.Layers[:i], lm.Layers[i+1:]...)
		if layer.Widget != nil {
			if c, ok := layer.Widget.(interface{ OnClosed() }); ok {
				c.OnClosed()
			}
		}
		// A paint-only layer's opener has no Widget entry, so it is told separately. (A layer that
		// carries both — a combo box is its own dropdown's widget AND its owner — is told once.)
		if layer.Owner != nil && layer.Owner != layer.Widget {
			if c, ok := layer.Owner.(interface{ OnClosed() }); ok {
				c.OnClosed()
			}
		}
		return true
	}
	return false
}

// TopLayer is the topmost visible layer's widget, or nil. It is the widget a key belongs to when the
// window is routing to a popup rather than to the focused child.
func (lm *LayerManager) TopLayer() Widget {
	for i := len(lm.Layers) - 1; i >= 0; i-- {
		if layer := lm.Layers[i]; layer != nil && layer.Visible && layer.Widget != nil {
			return layer.Widget
		}
	}
	return nil
}

// LayerCount is how many layers are open, which a window needs to decide whether Escape should close one
// or fall through to a widget.
func (lm *LayerManager) LayerCount() int {
	n := 0
	for _, l := range lm.Layers {
		if l != nil && l.Visible {
			n++
		}
	}
	return n
}
