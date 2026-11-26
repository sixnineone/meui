package meui

import (
	"fmt"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"os"
)

// ── Drag-and-drop framework ─────────────────────────────────────

// DragData carries the payload of a drag operation.
type DragData struct {
	Type   string      // identifies what's being dragged (e.g. "listview-item", "tab")
	Source Widget      // the widget that initiated the drag
	Data   interface{} // arbitrary payload
	Index  int         // source index within the source widget
}

// DragSource is implemented by widgets that can initiate drags.
type DragSource interface {
	// OnDragStart is called when a drag begins. Return non-nil to start the drag.
	OnDragStart(x, y float32) *DragData
}

// DropTarget is implemented by widgets that can accept drops.
type DropTarget interface {
	// OnDragEnter is called when the drag enters the widget's bounds.
	OnDragEnter(data *DragData, x, y float32)
	// OnDragOver is called as the cursor moves over the widget during a drag.
	OnDragOver(data *DragData, x, y float32)
	// OnDragDrop is called when the drag is released over the widget.
	OnDragDrop(data *DragData, x, y float32)
	// OnDragLeave is called when the drag leaves the widget's bounds.
	OnDragLeave(data *DragData)
}

// DragThreshold is the minimum distance (in pixels) the mouse must move
// before a drag operation begins.
const DragThreshold float32 = 6

// DragManager tracks the current drag operation.
// There is one DragManager per Window.
type DragManager struct {
	// Active is true while a drag is in progress.
	Active bool
	// Data holds the current drag payload.
	Data *DragData
	// Source is the widget that initiated the drag.
	Source Widget
	// Target is the current drop target under the cursor.
	Target DropTarget
	// GhostLayer is an optional layer that renders a drag ghost.
	GhostLayer *Layer
	// startPos is the position where the mouse was pressed.
	startX, startY float32
	// currentPos is updated during drag.
	currentX, currentY float32
	// Window reference for layer management.
	window *Window
}

// NewDragManager creates a new DragManager for the given Window.
func NewDragManager(w *Window) *DragManager {
	return &DragManager{window: w}
}

// StartCheck checks whether the mouse has moved far enough from the start
// position to initiate a drag. If so, it calls OnDragStart on the source.
// Returns true if a drag was started.
func (dm *DragManager) StartCheck(source Widget, x, y float32) bool {
	if dm.Active {
		return true
	}
	dx := x - dm.startX
	dy := y - dm.startY
	if dx*dx+dy*dy < DragThreshold*DragThreshold {
		return false
	}
	if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
		fmt.Printf("[drag] StartCheck passed, source=%T\n", source)
	}
	ds, ok := source.(DragSource)
	if !ok {
		return false
	}
	data := ds.OnDragStart(dm.startX, dm.startY)
	if data == nil {
		return false
	}
	dm.Active = true
	dm.Data = data
	dm.Source = source
	dm.currentX = x
	dm.currentY = y
	data.Source = source
	// Create ghost layer
	dm.createGhostLayer()
	// Hit-test initial drop target
	dm.updateDropTarget(x, y)
	return true
}

// Update is called during mouse drag to update ghost position and hit-test
// for drop targets.
func (dm *DragManager) Update(x, y float32, localX, localY float32) {
	if !dm.Active {
		return
	}
	dm.currentX = x
	dm.currentY = y
	// Update ghost layer position
	if dm.GhostLayer != nil {
		dm.GhostLayer.X = x - 40
		dm.GhostLayer.Y = y - 20
	}
	// Hit-test for drop targets
	dm.updateDropTarget(x, y)
}

// End is called on mouse up to finalize the drop.
func (dm *DragManager) End(x, y float32) {
	if !dm.Active {
		return
	}
	if dm.Target != nil {
		dm.Target.OnDragDrop(dm.Data, x, y)
	}
	dm.cleanup()
}

// Cancel ends the drag without performing a drop.
func (dm *DragManager) Cancel() {
	dm.cleanup()
}

// RecordStart records the initial mouse press position for later threshold check.
func (dm *DragManager) RecordStart(x, y float32) {
	dm.startX = x
	dm.startY = y
}

func (dm *DragManager) updateDropTarget(x, y float32) {
	// Walk the widget tree from the window to find a DropTarget under the cursor
	var newTarget DropTarget
	if dm.window != nil {
		// First check layers
		if dm.window.LayerManager != nil {
			layer := dm.window.LayerManager.HitTest(x, y)
			if layer != nil && layer.Widget != nil {
				if dt, ok := layer.Widget.(DropTarget); ok {
					newTarget = dt
				}
			}
		}
		// Then check children
		if newTarget == nil {
			hit := dm.window.hitTest(x, y)
			if hit != nil {
				if dt, ok := hit.(DropTarget); ok {
					newTarget = dt
				} else {
					// Walk up parent chain
					for p := hit; p != nil; {
						if dt, ok := p.(DropTarget); ok {
							newTarget = dt
							break
						}
						if pp, ok := p.(interface{ Parent() Widget }); ok {
							p = pp.Parent()
						} else {
							break
						}
					}
				}
			}
		}
	}
	if newTarget != dm.Target {
		if dm.Target != nil {
			dm.Target.OnDragLeave(dm.Data)
		}
		dm.Target = newTarget
		if dm.Target != nil {
			dm.Target.OnDragEnter(dm.Data, x, y)
		}
	}
	if dm.Target != nil {
		dm.Target.OnDragOver(dm.Data, x, y)
	}
}

func (dm *DragManager) createGhostLayer() {
	if dm.window == nil || dm.window.LayerManager == nil {
		return
	}
	// Create a simple ghost: a semi-transparent paint layer
	dm.GhostLayer = dm.window.LayerManager.AddLayerWithPaint(9999, func(canvas *graphics.Canvas, rect graphics.Rect) {
		bg := graphics.NewPaint()
		bg.SetColor(graphics.Color(0x8800AAFF))
		canvas.DrawRoundedRect(rect, 4, 4, bg)
		bd := graphics.NewPaint()
		bd.SetStyle(paintstyle.Stroke)
		bd.SetStrokeWidth(1)
		bd.SetColor(graphics.Color(0xFF4FC3F7))
		canvas.DrawRoundedRect(rect, 4, 4, bd)
		// Draw the dragged item label if available
		if label, ok := dm.Data.Data.(string); ok && label != "" {
			font := graphics.SystemFont
			if font != nil {
				tp := graphics.NewPaint()
				tp.SetColor(graphics.Color(0xFFFFFFFF))
				ascent := font.Baseline()
				lh := font.LineHeight()
				textY := rect.Y + (rect.Height-lh)/2 + ascent
				canvas.DrawSimpleString(label, rect.X+8, textY, font, tp)
			}
		}
	})
	if dm.GhostLayer != nil {
		dm.GhostLayer.Width = 160
		dm.GhostLayer.Height = 28
		dm.GhostLayer.X = dm.currentX - 40
		dm.GhostLayer.Y = dm.currentY - 20
		dm.GhostLayer.Visible = true
	}
}

func (dm *DragManager) cleanup() {
	if dm.GhostLayer != nil && dm.window != nil && dm.window.LayerManager != nil {
		dm.window.LayerManager.RemoveLayer(dm.GhostLayer)
	}
	if dm.Target != nil {
		dm.Target.OnDragLeave(dm.Data)
	}
	dm.Active = false
	dm.Data = nil
	dm.Source = nil
	dm.Target = nil
	dm.GhostLayer = nil
}
