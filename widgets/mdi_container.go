package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
	"log"
	"time"
)

const (
	taskbarH         = 28
	dropZoneH        = 40
	dockMinSize      = 50
	dockTitleBarH    = 18
	dockBtnW         = 22
	dockBtnH         = 16
	dockAutoHideSize = 5   // tab width/height when in auto-hide mode (collapsed)
	autoHideDelay    = 300 // ms to wait before auto-hiding after mouse leaves
)

// MdiContainer is a container that holds MdiWindows as free-floating
// or docked children. It manages dock zones (left, right, top, bottom),
// floating windows (center), a taskbar, and drag-to-dock behavior.
type MdiContainer struct {
	meui.BaseContainer
	ctxMenu     *ContextMenu
	dockCtxMenu *ContextMenu
	ShowTaskbar bool
	MdiTheme    *theme.MdiTheme

	// Dock zones — each edge has a StackPanel (nil when empty) + size
	LeftStack   *StackPanel
	RightStack  *StackPanel
	TopStack    *StackPanel
	BottomStack *StackPanel
	LeftSize    float32
	RightSize   float32
	TopSize     float32
	BottomSize  float32

	// Collapsed state for each dock edge
	LeftCollapsed   bool
	RightCollapsed  bool
	TopCollapsed    bool
	BottomCollapsed bool

	// Auto-hide state: when true, dock auto-hides to a thin edge tab (VS Code style)
	LeftAutoHide   bool
	RightAutoHide  bool
	TopAutoHide    bool
	BottomAutoHide bool

	// Auto-hide expand tracking (temporary expand on hover)
	expandTimer  *time.Timer
	expandedDock DockPosition // which dock is temporarily expanded (DockCenter = none)

	// When true, left/right docks span the full height (top/bottom don't reduce their height).
	// When false (default), top/bottom span full width (left/right don't reduce their width).
	DocksFullHeight bool

	// Center area rect (computed by layoutDockZones), for maximizing windows
	CenterRect graphics.Rect

	// Drop zone state during drag
	DropHighlight *DropZoneInfo // currently highlighted drop zone (nil = none)

	// Dock resize state
	resizingDock    DockPosition
	dockResizeStart float32
	dockResizeOrig  float32

	// Taskbar notification area
	Notifications []NotificationItem
	textCache     *graphics.Text
	textCacheVer  uint64
}

// SetDocksFullHeight changes the dock layout mode and triggers a layout update.
func (mc *MdiContainer) SetDocksFullHeight(v bool) {
	if mc.DocksFullHeight == v {
		return
	}
	mc.DocksFullHeight = v
	mc.SetNeedsLayout()
	mc.MarkForRedraw()
}

func NewMdiContainer() *MdiContainer {
	return &MdiContainer{
		BaseContainer: meui.BaseContainer{BaseWidget: *meui.NewBaseWidget()},
		LeftSize:      180,
		RightSize:     180,
		TopSize:       120,
		BottomSize:    120,
		MdiTheme:      theme.DefaultMdiTheme,
	}
}

// ── Drop Target Interface ───────────────────────────────────────

// DropHitTest returns the drop zone at (x,y) in MdiContainer-local coords,
// or nil if the window should stay floating. Called during drag to determine
// where the dragged window will land.
func (mc *MdiContainer) DropHitTest(x, y float32) *DropZoneInfo {
	// 1. Check StackPanel item title/tab bars FIRST — merge takes priority
	// over edge dock zones (which overlap with docked stack tab bars).
	for _, sp := range []*StackPanel{mc.TopStack, mc.BottomStack, mc.LeftStack, mc.RightStack} {
		if sp == nil {
			continue
		}
		cf := sp.FrameRect()
		if x >= cf.X && x <= cf.X+cf.Width && y >= cf.Y && y <= cf.Y+cf.Height {
			lx, ly := x-cf.X, y-cf.Y
			if info := sp.DropHitTest(lx, ly); info != nil {
				// Convert ZoneRect from StackPanel-local to MdiContainer-local
				info.ZoneRect = graphics.NewRect(
					cf.X+info.ZoneRect.X,
					cf.Y+info.ZoneRect.Y,
					info.ZoneRect.Width,
					info.ZoneRect.Height,
				)
				return info
			}
		}
	}

	// 2. Edge zones for docking (only if no merge target)
	zone, zr := mc.dockZoneAt(x, y)
	if zone != DockCenter {
		return &DropZoneInfo{
			Type:     DropDock,
			DockPos:  zone,
			ZoneRect: zr,
		}
	}

	// 3. No drop zone — window floats
	return nil
}

// AcceptDrop finalizes the drop. Calls release() to detach from the original
// parent, then reparents the window according to info.
func (mc *MdiContainer) AcceptDrop(mw *MdiWindow, info *DropZoneInfo) {
	mc.ClearDropHighlight()
	mc.MarkForRedraw()

	switch info.Type {
	case DropDock:
		if mw.releaseCallback != nil {
			mw.releaseCallback()
			mw.releaseCallback = nil
		}
		mc.addToDock(mw, info.DockPos)
		mc.SetNeedsLayout()
		mc.MarkForRedraw()
	case DropMerge:
		if mw.releaseCallback != nil {
			mw.releaseCallback()
			mw.releaseCallback = nil
		}
		sp := info.Stack
		if sp == nil {
			log.Println("DBG  sp is nil")
			break
		}
		for idx, item := range sp.Items {
			if item == info.Target {
				if tgtMw, ok := item.(*MdiWindow); ok {
					mc.mergeIntoTabPanel(mw, tgtMw, sp, idx)
				} else if tgtTp, ok := item.(*TabPanel); ok {
					mc.mergeIntoExistingTabPanel(mw, tgtTp, sp, idx)
				} else {
				}
				break
			}
		}
	case DropFloat:
		// Already a child of MdiContainer — stays put
		mw.releaseCallback = nil
	}
}

// ClearDropHighlight clears the visual highlight on the current drop zone.
func (mc *MdiContainer) ClearDropHighlight() {
	mc.DropHighlight = nil
	mc.MarkForRedraw()
}

// ── Dock management ─────────────────────────────────────────────

func (mc *MdiContainer) dockStack(pos DockPosition) **StackPanel {
	switch pos {
	case DockLeft:
		return &mc.LeftStack
	case DockRight:
		return &mc.RightStack
	case DockTop:
		return &mc.TopStack
	case DockBottom:
		return &mc.BottomStack
	}
	return nil
}

func (mc *MdiContainer) dockSize(pos DockPosition) float32 {
	switch pos {
	case DockLeft:
		return mc.LeftSize
	case DockRight:
		return mc.RightSize
	case DockTop:
		return mc.TopSize
	case DockBottom:
		return mc.BottomSize
	}
	return 0
}

func (mc *MdiContainer) addToDock(mw *MdiWindow, pos DockPosition) {
	mc.removeFromDock(mw)
	mw.DockPos = pos
	mw.dockRestore = mw.FrameRect()
	mw.container = mc
	mw.SetParent(mc)

	mc.DetachChild(mw)

	sp := *mc.dockStack(pos)
	if sp == nil {
		orientation := "vertical"
		if pos == DockTop || pos == DockBottom {
			orientation = "horizontal"
		}
		sp = NewStackPanel(orientation)
		*mc.dockStack(pos) = sp
		sp.SetParent(mc)
		mc.AppendChild(sp)
	}

	n := sp.ItemCount()
	sz := mc.dockSize(pos) / float32(n+1)
	for i := range sp.Sizes {
		sp.Sizes[i] = sz
	}
	sp.AddItem(mw, sz)

	frame := mc.FrameRect()
	tbH := float32(0)
	if mc.ShowTaskbar {
		tbH = taskbarH
	}
	mc.layoutDockZones(frame, tbH)

	mw.SetVisible(true)
	mw.Minimized = false
	mc.MarkForRedraw()
}

func (mc *MdiContainer) removeFromDock(mw *MdiWindow) {
	if mw.DockPos == DockCenter {
		return
	}
	pos := mw.DockPos
	sp := *mc.dockStack(pos)
	if sp == nil {
		return
	}

	spFrame := sp.FrameRect()
	itemFrame := mw.FrameRect()

	sp.RemoveWidget(mw)
	mc.AppendChild(mw)
	mw.SetParent(mc)
	mw.container = mc
	mw.DockPos = DockCenter

	mw.SetFrameRect(graphics.NewRect(spFrame.X+itemFrame.X, spFrame.Y+itemFrame.Y, itemFrame.Width, itemFrame.Height))

	if sp.ItemCount() == 0 {
		mc.DetachChild(sp)
		*mc.dockStack(pos) = nil
	}
	mc.SetNeedsLayout()
}

func (mc *MdiContainer) isDocked(mw *MdiWindow) bool {
	return mw.DockPos != DockCenter
}

// dockZoneAt returns which dock zone the point is near, using dropZoneH margin.
func (mc *MdiContainer) dockZoneAt(x, y float32) (DockPosition, graphics.Rect) {
	tbH := float32(0)
	if mc.ShowTaskbar {
		tbH = taskbarH
	}
	w, h := mc.Width(), mc.Height()-tbH

	if x <= dropZoneH {
		return DockLeft, graphics.NewRect(0, 0, dropZoneH, h)
	}
	if x >= w-dropZoneH {
		return DockRight, graphics.NewRect(w-dropZoneH, 0, dropZoneH, h)
	}
	if y <= dropZoneH {
		return DockTop, graphics.NewRect(0, 0, w, dropZoneH)
	}
	if y >= h-dropZoneH {
		return DockBottom, graphics.NewRect(0, h-dropZoneH, w, dropZoneH)
	}
	return DockCenter, graphics.Rect{}
}

// layoutDockZones positions all docked windows and returns the remaining center rect.
func (mc *MdiContainer) layoutDockZones(frame graphics.Rect, tbH float32) graphics.Rect {
	x, y := float32(0), float32(0)
	w, h := frame.Width, frame.Height-tbH

	if mc.DocksFullHeight {
		// Mode: left/right fill full height, top/bottom fit between left/right
		if mc.LeftStack != nil && mc.LeftStack.ItemCount() > 0 {
			sz := mc.LeftSize
			if mc.LeftCollapsed {
				sz = dockAutoHideSize
			}
			if !mc.LeftCollapsed {
				spH := h - dockTitleBarH
				if spH < 10 {
					spH = 10
				}
				r := graphics.NewRect(x, y+dockTitleBarH, sz, spH)
				mc.LeftStack.SetFrameRect(r)
				mc.LeftStack.ValidateLayout()
			} else {
				mc.LeftStack.SetFrameRect(graphics.Rect{})
			}
			x += sz
			w -= sz
		}
		if mc.RightStack != nil && mc.RightStack.ItemCount() > 0 {
			sz := mc.RightSize
			if mc.RightCollapsed {
				sz = dockAutoHideSize
			}
			rx := x + w - sz
			if !mc.RightCollapsed {
				spH := h - dockTitleBarH
				if spH < 10 {
					spH = 10
				}
				r := graphics.NewRect(rx, y+dockTitleBarH, sz, spH)
				mc.RightStack.SetFrameRect(r)
				mc.RightStack.ValidateLayout()
			} else {
				mc.RightStack.SetFrameRect(graphics.Rect{})
			}
			w -= sz
		}
		if mc.TopStack != nil && mc.TopStack.ItemCount() > 0 {
			sz := mc.TopSize
			if mc.TopCollapsed {
				sz = dockAutoHideSize
			}
			if !mc.TopCollapsed {
				spH := sz - dockTitleBarH
				if spH < 10 {
					spH = 10
				}
				r := graphics.NewRect(x, y+dockTitleBarH, w, spH)
				mc.TopStack.SetFrameRect(r)
				mc.TopStack.ValidateLayout()
			} else {
				mc.TopStack.SetFrameRect(graphics.Rect{})
			}
			y += sz
			h -= sz
		}
		if mc.BottomStack != nil && mc.BottomStack.ItemCount() > 0 {
			sz := mc.BottomSize
			if mc.BottomCollapsed {
				sz = dockAutoHideSize
			}
			by := y + h - sz
			if !mc.BottomCollapsed {
				spH := sz - dockTitleBarH
				if spH < 10 {
					spH = 10
				}
				r := graphics.NewRect(x, by+dockTitleBarH, w, spH)
				mc.BottomStack.SetFrameRect(r)
				mc.BottomStack.ValidateLayout()
			} else {
				mc.BottomStack.SetFrameRect(graphics.Rect{})
			}
			h -= sz
		}
	} else {
		// Mode (default): top/bottom fill full width, left/right fit between top/bottom
		if mc.TopStack != nil && mc.TopStack.ItemCount() > 0 {
			sz := mc.TopSize
			if mc.TopCollapsed {
				sz = dockAutoHideSize
			}
			if !mc.TopCollapsed {
				spH := sz - dockTitleBarH
				if spH < 10 {
					spH = 10
				}
				r := graphics.NewRect(x, y+dockTitleBarH, w, spH)
				mc.TopStack.SetFrameRect(r)
				mc.TopStack.ValidateLayout()
			} else {
				mc.TopStack.SetFrameRect(graphics.Rect{})
			}
			y += sz
			h -= sz
		}
		if mc.BottomStack != nil && mc.BottomStack.ItemCount() > 0 {
			sz := mc.BottomSize
			if mc.BottomCollapsed {
				sz = dockAutoHideSize
			}
			by := y + h - sz
			if !mc.BottomCollapsed {
				spH := sz - dockTitleBarH
				if spH < 10 {
					spH = 10
				}
				r := graphics.NewRect(x, by+dockTitleBarH, w, spH)
				mc.BottomStack.SetFrameRect(r)
				mc.BottomStack.ValidateLayout()
			} else {
				mc.BottomStack.SetFrameRect(graphics.Rect{})
			}
			h -= sz
		}
		if mc.LeftStack != nil && mc.LeftStack.ItemCount() > 0 {
			sz := mc.LeftSize
			if mc.LeftCollapsed {
				sz = dockAutoHideSize
			}
			if !mc.LeftCollapsed {
				spH := h - dockTitleBarH
				if spH < 10 {
					spH = 10
				}
				r := graphics.NewRect(x, y+dockTitleBarH, sz, spH)
				mc.LeftStack.SetFrameRect(r)
				mc.LeftStack.ValidateLayout()
			} else {
				mc.LeftStack.SetFrameRect(graphics.Rect{})
			}
			x += sz
			w -= sz
		}
		if mc.RightStack != nil && mc.RightStack.ItemCount() > 0 {
			sz := mc.RightSize
			if mc.RightCollapsed {
				sz = dockAutoHideSize
			}
			rx := x + w - sz
			if !mc.RightCollapsed {
				spH := h - dockTitleBarH
				if spH < 10 {
					spH = 10
				}
				r := graphics.NewRect(rx, y+dockTitleBarH, sz, spH)
				mc.RightStack.SetFrameRect(r)
				mc.RightStack.ValidateLayout()
			} else {
				mc.RightStack.SetFrameRect(graphics.Rect{})
			}
			w -= sz
		}
	}

	result := graphics.NewRect(x, y, w, h)
	mc.CenterRect = result
	return result
}

func (mc *MdiContainer) dockResizeHit(x, y float32) DockPosition {
	tbH := float32(0)
	if mc.ShowTaskbar {
		tbH = taskbarH
	}
	w, h := mc.Width(), mc.Height()-tbH
	sz := float32(6)

	if mc.RightStack != nil && mc.RightStack.ItemCount() > 0 && !mc.RightCollapsed {
		rx := w - mc.RightSize
		if x >= rx-sz && x <= rx+sz && y >= 0 && y <= h {
			return DockRight
		}
	}
	if mc.LeftStack != nil && mc.LeftStack.ItemCount() > 0 && !mc.LeftCollapsed {
		lx := mc.LeftSize
		if x >= lx-sz && x <= lx+sz && y >= 0 && y <= h {
			return DockLeft
		}
	}
	if mc.BottomStack != nil && mc.BottomStack.ItemCount() > 0 && !mc.BottomCollapsed {
		by := h - mc.BottomSize
		if y >= by-sz && y <= by+sz && x >= 0 && x <= w {
			return DockBottom
		}
	}
	if mc.TopStack != nil && mc.TopStack.ItemCount() > 0 && !mc.TopCollapsed {
		ty := mc.TopSize
		if y >= ty-sz && y <= ty+sz && x >= 0 && x <= w {
			return DockTop
		}
	}
	return DockCenter
}

// dockTitleBarHit returns which dock zone's title bar contains (x,y), or DockCenter.
func (mc *MdiContainer) dockTitleBarHit(x, y float32) DockPosition {
	tbH := float32(0)
	if mc.ShowTaskbar {
		tbH = taskbarH
	}
	w, h := mc.Width(), mc.Height()-tbH
	if mc.LeftStack != nil && mc.LeftStack.ItemCount() > 0 && !mc.LeftCollapsed {
		if x >= 0 && x <= mc.LeftSize && y >= 0 && y <= dockTitleBarH {
			return DockLeft
		}
	}
	if mc.RightStack != nil && mc.RightStack.ItemCount() > 0 && !mc.RightCollapsed {
		rx := w - mc.RightSize
		if x >= rx && x <= w && y >= 0 && y <= dockTitleBarH {
			return DockRight
		}
	}
	if mc.TopStack != nil && mc.TopStack.ItemCount() > 0 && !mc.TopCollapsed {
		if y >= 0 && y <= dockTitleBarH {
			return DockTop
		}
	}
	if mc.BottomStack != nil && mc.BottomStack.ItemCount() > 0 && !mc.BottomCollapsed {
		by := h - mc.BottomSize
		if y >= by && y <= by+dockTitleBarH {
			return DockBottom
		}
	}
	return DockCenter
}

// dockMenuBtnRect returns the menu button rect within a dock zone rect.
func (mc *MdiContainer) dockMenuBtnRect(zoneRect graphics.Rect) graphics.Rect {
	return graphics.NewRect(zoneRect.X+zoneRect.Width-dockBtnW-4, zoneRect.Y+(dockTitleBarH-dockBtnH)/2, dockBtnW, dockBtnH)
}

// dockPinBtnRect returns the pin/unpin button rect within a dock zone rect.
func (mc *MdiContainer) dockPinBtnRect(zoneRect graphics.Rect) graphics.Rect {
	return graphics.NewRect(zoneRect.X+zoneRect.Width-2*dockBtnW-8, zoneRect.Y+(dockTitleBarH-dockBtnH)/2, dockBtnW, dockBtnH)
}

// dockMenuBtnHit returns which dock zone's menu button contains (x,y), or DockCenter.
func (mc *MdiContainer) dockMenuBtnHit(x, y float32) DockPosition {
	tbH := float32(0)
	if mc.ShowTaskbar {
		tbH = taskbarH
	}
	w, hh := mc.Width(), mc.Height()-tbH
	cx, cy, cw, ch := float32(0), float32(0), w, hh
	if mc.TopStack != nil && mc.TopStack.ItemCount() > 0 {
		zr := graphics.NewRect(cx, cy, cw, mc.TopSize)
		br := mc.dockMenuBtnRect(zr)
		if x >= br.X && x <= br.X+br.Width && y >= br.Y && y <= br.Y+br.Height {
			return DockTop
		}
		cy += mc.TopSize
		ch -= mc.TopSize
	}
	if mc.BottomStack != nil && mc.BottomStack.ItemCount() > 0 {
		by := cy + ch - mc.BottomSize
		zr := graphics.NewRect(cx, by, cw, mc.BottomSize)
		br := mc.dockMenuBtnRect(zr)
		if x >= br.X && x <= br.X+br.Width && y >= br.Y && y <= br.Y+br.Height {
			return DockBottom
		}
		ch -= mc.BottomSize
	}
	if mc.LeftStack != nil && mc.LeftStack.ItemCount() > 0 {
		zr := graphics.NewRect(cx, cy, mc.LeftSize, ch)
		br := mc.dockMenuBtnRect(zr)
		if x >= br.X && x <= br.X+br.Width && y >= br.Y && y <= br.Y+br.Height {
			return DockLeft
		}
		cx += mc.LeftSize
		cw -= mc.LeftSize
	}
	if mc.RightStack != nil && mc.RightStack.ItemCount() > 0 {
		rx := cx + cw - mc.RightSize
		zr := graphics.NewRect(rx, cy, mc.RightSize, ch)
		br := mc.dockMenuBtnRect(zr)
		if x >= br.X && x <= br.X+br.Width && y >= br.Y && y <= br.Y+br.Height {
			return DockRight
		}
	}
	return DockCenter
}

// dockPinBtnHit returns which dock zone's pin button contains (x,y), or DockCenter.
func (mc *MdiContainer) dockPinBtnHit(x, y float32) DockPosition {
	tbH := float32(0)
	if mc.ShowTaskbar {
		tbH = taskbarH
	}
	w, hh := mc.Width(), mc.Height()-tbH
	cx, cy, cw, ch := float32(0), float32(0), w, hh
	if mc.TopStack != nil && mc.TopStack.ItemCount() > 0 {
		zr := graphics.NewRect(cx, cy, cw, mc.TopSize)
		br := mc.dockPinBtnRect(zr)
		if x >= br.X && x <= br.X+br.Width && y >= br.Y && y <= br.Y+br.Height {
			return DockTop
		}
		cy += mc.TopSize
		ch -= mc.TopSize
	}
	if mc.BottomStack != nil && mc.BottomStack.ItemCount() > 0 {
		by := cy + ch - mc.BottomSize
		zr := graphics.NewRect(cx, by, cw, mc.BottomSize)
		br := mc.dockPinBtnRect(zr)
		if x >= br.X && x <= br.X+br.Width && y >= br.Y && y <= br.Y+br.Height {
			return DockBottom
		}
		ch -= mc.BottomSize
	}
	if mc.LeftStack != nil && mc.LeftStack.ItemCount() > 0 {
		zr := graphics.NewRect(cx, cy, mc.LeftSize, ch)
		br := mc.dockPinBtnRect(zr)
		if x >= br.X && x <= br.X+br.Width && y >= br.Y && y <= br.Y+br.Height {
			return DockLeft
		}
		cx += mc.LeftSize
		cw -= mc.LeftSize
	}
	if mc.RightStack != nil && mc.RightStack.ItemCount() > 0 {
		rx := cx + cw - mc.RightSize
		zr := graphics.NewRect(rx, cy, mc.RightSize, ch)
		br := mc.dockPinBtnRect(zr)
		if x >= br.X && x <= br.X+br.Width && y >= br.Y && y <= br.Y+br.Height {
			return DockRight
		}
	}
	return DockCenter
}

// isDockCollapsed returns true if the given dock position is collapsed.
func (mc *MdiContainer) isDockCollapsed(pos DockPosition) bool {
	switch pos {
	case DockLeft:
		return mc.LeftCollapsed
	case DockRight:
		return mc.RightCollapsed
	case DockTop:
		return mc.TopCollapsed
	case DockBottom:
		return mc.BottomCollapsed
	}
	return false
}

// toggleDockCollapse toggles the collapsed state for the given dock position.
// In auto-hide mode, collapsed docks show a thin edge tab instead of hiding fully.
func (mc *MdiContainer) toggleDockCollapse(pos DockPosition) {
	sp := *mc.dockStack(pos)
	collapsed := mc.isDockCollapsed(pos)
	switch pos {
	case DockLeft:
		mc.LeftCollapsed = !mc.LeftCollapsed
		mc.LeftAutoHide = mc.LeftCollapsed // entering auto-hide mode
		if sp != nil {
			sp.SetVisible(!mc.LeftCollapsed)
		}
	case DockRight:
		mc.RightCollapsed = !mc.RightCollapsed
		mc.RightAutoHide = mc.RightCollapsed
		if sp != nil {
			sp.SetVisible(!mc.RightCollapsed)
		}
	case DockTop:
		mc.TopCollapsed = !mc.TopCollapsed
		mc.TopAutoHide = mc.TopCollapsed
		if sp != nil {
			sp.SetVisible(!mc.TopCollapsed)
		}
	case DockBottom:
		mc.BottomCollapsed = !mc.BottomCollapsed
		mc.BottomAutoHide = mc.BottomCollapsed
		if sp != nil {
			sp.SetVisible(!mc.BottomCollapsed)
		}
	}
	if !collapsed {
		// Just collapsed → expandedDock = none
		mc.expandedDock = DockCenter
	}
	mc.cancelExpandTimer()
	mc.SetNeedsLayout()
	mc.MarkForRedraw()
}

// tempExpand temporarily expands an auto-hide dock (called on hover).
// Returns the previous expand state.
func (mc *MdiContainer) tempExpand(pos DockPosition) {
	if pos == DockCenter {
		return
	}
	if !mc.isDockCollapsed(pos) {
		return
	}
	sp := *mc.dockStack(pos)
	if sp == nil {
		return
	}

	switch pos {
	case DockLeft:
		mc.LeftCollapsed = false
	case DockRight:
		mc.RightCollapsed = false
	case DockTop:
		mc.TopCollapsed = false
	case DockBottom:
		mc.BottomCollapsed = false
	}
	mc.expandedDock = pos
	sp.SetVisible(true)
	mc.SetNeedsLayout()
	mc.MarkForRedraw()
}

// scheduleAutoHide starts a timer to re-collapse a temporarily expanded auto-hide dock.
func (mc *MdiContainer) scheduleAutoHide(pos DockPosition) {
	if pos == DockCenter {
		return
	}
	mc.cancelExpandTimer()
	mc.expandTimer = time.AfterFunc(autoHideDelay*time.Millisecond, func() {
		platform.InvokeTask(func() { mc.cancelAutoHide(pos) })
	})
}

// cancelAutoHide collapses a temporarily expanded auto-hide dock.
func (mc *MdiContainer) cancelAutoHide(pos DockPosition) {
	if pos == DockCenter {
		return
	}
	// Re-check: only collapse if still in auto-hide mode and not pinned
	var autoHide bool
	switch pos {
	case DockLeft:
		autoHide = mc.LeftAutoHide
	case DockRight:
		autoHide = mc.RightAutoHide
	case DockTop:
		autoHide = mc.TopAutoHide
	case DockBottom:
		autoHide = mc.BottomAutoHide
	}
	if !autoHide {
		return
	}

	sp := *mc.dockStack(pos)
	if sp == nil {
		return
	}

	switch pos {
	case DockLeft:
		mc.LeftCollapsed = true
	case DockRight:
		mc.RightCollapsed = true
	case DockTop:
		mc.TopCollapsed = true
	case DockBottom:
		mc.BottomCollapsed = true
	}
	mc.expandedDock = DockCenter
	sp.SetVisible(false)
	mc.SetNeedsLayout()
	mc.MarkForRedraw()
}

// cancelExpandTimer stops any pending auto-hide timer.
func (mc *MdiContainer) cancelExpandTimer() {
	if mc.expandTimer != nil {
		mc.expandTimer.Stop()
		mc.expandTimer = nil
	}
}

// undockAllFrom undocks all windows from the given dock zone, floating them.
func (mc *MdiContainer) undockAllFrom(pos DockPosition) {
	sp := *mc.dockStack(pos)
	if sp == nil {
		return
	}
	items := make([]*MdiWindow, 0, sp.ItemCount())
	for _, item := range sp.Items {
		if mw, ok := item.(*MdiWindow); ok {
			items = append(items, mw)
		}
	}
	spFrame := sp.FrameRect()
	mcFrame := mc.FrameRect()
	step := float32(30)
	cx, cy := spFrame.X, spFrame.Y
	for _, mw := range items {
		mc.removeFromDock(mw)
		fr := mw.FrameRect()
		mw.SetFrameRect(graphics.NewRect(cx, cy, fr.Width, fr.Height))
		if cx+fr.Width+step > mcFrame.Width {
			cx = spFrame.X
			cy += step
		} else {
			cx += step
			cy += step
		}
	}
	mc.SetNeedsLayout()
	mc.MarkForRedraw()
}

// mergeIntoTabPanel merges source and target MdiWindows into a TabPanel,
// replacing the target's slot in the StackPanel.
func (mc *MdiContainer) mergeIntoTabPanel(source, target *MdiWindow, sp *StackPanel, idx int) {

	tp := NewTabPanel()
	tp.OnTabDetached = mc.makeTabDetachHandler(sp)
	tp.AddTab(target.Title, target.Content)
	target.Content = nil
	tp.AddTab(source.Title, source.Content)
	source.Content = nil

	origItems := make([]meui.Widget, len(sp.Items))
	copy(origItems, sp.Items)
	origSizes := make([]float32, len(sp.Sizes))
	copy(origSizes, sp.Sizes)

	sp.RemoveItem(idx)

	sp.Items = make([]meui.Widget, 0, len(origItems))
	sp.Items = append(sp.Items, origItems[:idx]...)
	sp.Items = append(sp.Items, tp)
	sp.Items = append(sp.Items, origItems[idx+1:]...)

	sp.Sizes = make([]float32, 0, len(origSizes))
	sp.Sizes = append(sp.Sizes, origSizes[:idx]...)
	sp.Sizes = append(sp.Sizes, origSizes[idx])
	sp.Sizes = append(sp.Sizes, origSizes[idx+1:]...)

	tp.SetParent(sp)
	mc.DetachChild(source)
	source.SetVisible(false)
	target.SetVisible(false)
}

// mergeIntoExistingTabPanel adds source's content as a new tab in targetTp.
func (mc *MdiContainer) mergeIntoExistingTabPanel(source *MdiWindow, targetTp *TabPanel, sp *StackPanel, idx int) {
	oldTabs := len(targetTp.tabs)
	titles := make([]string, oldTabs)
	for i, tab := range targetTp.tabs {
		titles[i] = tab.Title
	}

	// Ensure the target TabPanel has our detach handler
	if targetTp.OnTabDetached == nil {
		targetTp.OnTabDetached = mc.makeTabDetachHandler(sp)
	}
	targetTp.AddTab(source.Title, source.Content)
	source.Content = nil
	mc.DetachChild(source)
	source.SetVisible(false)

	sp.SetNeedsLayout()
	sp.MarkForRedraw()
	mc.SetNeedsLayout()
	mc.MarkForRedraw()
}

// ValidateLayout lays out dock zones and updates maximized windows.
func (mc *MdiContainer) ValidateLayout() {
	if !mc.NeedsLayout() {
		return
	}
	frame := mc.FrameRect()
	tbH := float32(0)
	if mc.ShowTaskbar {
		tbH = taskbarH
	}
	mc.layoutDockZones(frame, tbH)
	mc.updateMaximized()
	for _, child := range mc.Children() {
		if vc, ok := child.(interface{ ValidateLayout() }); ok {
			vc.ValidateLayout()
		}
	}
	mc.ClearNeedsLayout()
}

// updateMaximized resizes all maximized windows to fill the current center area.
// Called after dock layout changes.
func (mc *MdiContainer) updateMaximized() {
	for _, child := range mc.Children() {
		if mw, ok := child.(*MdiWindow); ok && mw.Maximized {
			cr := mc.CenterRect
			mw.SetFrameRect(graphics.NewRect(cr.X, cr.Y, cr.Width, cr.Height))
		}
	}
}

// AddChild overrides BaseContainer.AddChild to set parent to the outer MdiContainer.
func (mc *MdiContainer) AddChild(child meui.Widget) {
	mc.BaseContainer.AddChild(child)
	if child != nil {
		child.SetParent(mc)
		if mw, ok := child.(*MdiWindow); ok {
			mw.container = mc
		}
	}
}

// PanelAt returns the topmost MdiWindow that contains the point,
// or the container itself if no window is hit.
func (mc *MdiContainer) PanelAt(x, y float32) meui.Widget {
	// Check dock resize handles first (cursor changes)
	if mc.dockResizeHit(x, y) != DockCenter {
		return mc
	}

	// Check dock zones (iterate in z-order: top, bottom, left, right)
	dockStacks := []*StackPanel{mc.TopStack, mc.BottomStack, mc.LeftStack, mc.RightStack}
	for _, sp := range dockStacks {
		if sp == nil {
			continue
		}
		cf := sp.FrameRect()
		if x >= cf.X && x <= cf.X+cf.Width && y >= cf.Y && y <= cf.Y+cf.Height {
			lx, ly := x-cf.X, y-cf.Y
			for _, item := range sp.Items {
				if item == nil {
					continue
				}
				icf := item.FrameRect()
				if lx >= icf.X && lx <= icf.X+icf.Width && ly >= icf.Y && ly <= icf.Y+icf.Height {
					itemLocalX, itemLocalY := lx-icf.X, ly-icf.Y
					if container, ok := item.(meui.Container); ok {
						if deeper := container.PanelAt(itemLocalX, itemLocalY); deeper != nil {
							return deeper
						}
					}
					return item
				}
			}
			return sp
		}
	}

	// Check floating windows in reverse z-order (children)
	children := mc.Children()
	for i := len(children) - 1; i >= 0; i-- {
		child := children[i]
		if child == nil {
			continue
		}
		if v, ok := child.(interface{ Visible() bool }); ok {
			if !v.Visible() {
				continue
			}
		}
		cf := child.FrameRect()
		if x >= cf.X && x <= cf.X+cf.Width && y >= cf.Y && y <= cf.Y+cf.Height {
			lx, ly := x-cf.X, y-cf.Y
			if mw, ok := child.(*MdiWindow); ok {
				if mw.Minimized {
					continue
				}
				if container, ok := child.(meui.Container); ok {
					if deeper := container.PanelAt(lx, ly); deeper != nil {
						return deeper
					}
				}
			} else if container, ok := child.(meui.Container); ok {
				if deeper := container.PanelAt(lx, ly); deeper != nil {
					return deeper
				}
			}
			return child
		}
	}
	return mc
}

// drawDockTitleBar draws the dock zone title bar at the top of the given zone rect.
// For collapsed/auto-hide docks, draws a simple edge tab instead.
func (mc *MdiContainer) drawDockTitleBar(canvas *graphics.Canvas, zoneRect graphics.Rect, label string, pos DockPosition) {
	mt := mc.MdiTheme
	if mt == nil {
		mt = theme.DefaultMdiTheme
	}
	collapsed := mc.isDockCollapsed(pos)
	if collapsed {
		// Draw simple edge tab strip
		tr := graphics.NewRect(zoneRect.X, zoneRect.Y, zoneRect.Width, zoneRect.Height)
		tbg := graphics.NewPaint()
		tbg.SetColor(mt.TabBgInk)
		canvas.DrawRect(tr, tbg)
		return
	}

	tr := graphics.NewRect(zoneRect.X, zoneRect.Y, zoneRect.Width, dockTitleBarH)
	tbg := graphics.NewPaint()
	tbg.SetColor(mt.TabActiveBgInk)
	canvas.DrawRect(tr, tbg)
	tline := graphics.NewPaint()
	tline.SetStyle(paintstyle.Stroke)
	tline.SetStrokeWidth(1)
	tline.SetColor(mt.TabEdgeInk)
	canvas.DrawLine(tr.X, tr.Y+tr.Height, tr.X+tr.Width, tr.Y+tr.Height, tline)
	font := theme.DefaultTheme.BaseFont
	if font == nil {
		font = graphics.SystemFont
	}
	if font != nil {
		tp := graphics.NewPaint()
		tp.SetColor(mt.TabTextInk)
		ascent := font.Baseline()
		labelY := tr.Y + (dockTitleBarH-ascent)/2 + ascent
		ver := theme.ThemeVersion()
		if mc.textCache == nil || mc.textCacheVer != ver {
			mc.textCache = graphics.NewText(label, &graphics.TextDecoration{Font: font, OnBackgroundInk: tp.Color()})
			mc.textCacheVer = ver
		}
		mc.textCache.Draw(canvas, tr.X+8, labelY)

		// Draw pin/unpin button (left of menu button)
		pbRect := mc.dockPinBtnRect(zoneRect)
		pbg := graphics.NewPaint()
		pbg.SetColor(mt.TaskbarBtnBgInk)
		canvas.DrawRect(pbRect, pbg)
		pp := graphics.NewPaint()
		collapsed := mc.isDockCollapsed(pos)
		if collapsed {
			pp.SetColor(mt.TaskbarTextInk)
			// Draw "▶" triangle indicating "click to expand"
			pp.SetStyle(paintstyle.Stroke)
			pp.SetStrokeWidth(2)
			pcx := pbRect.X + pbRect.Width/2 - 2
			pcy := pbRect.Y + pbRect.Height/2
			canvas.DrawLine(pcx-3, pcy-4, pcx+3, pcy, pp)
			canvas.DrawLine(pcx+3, pcy, pcx-3, pcy+4, pp)
		} else {
			pp.SetColor(mt.WinTextSecondary)
			// Draw pin icon: circle + line
			pp.SetStyle(paintstyle.Stroke)
			pp.SetStrokeWidth(2)
			pcx := pbRect.X + pbRect.Width/2
			pcy := pbRect.Y + pbRect.Height/2
			canvas.DrawCircle(pcx, pcy-3, 3, pp)
			canvas.DrawLine(pcx, pcy+1, pcx, pcy+5, pp)
		}

		// Draw ☰ menu button on the right
		br := mc.dockMenuBtnRect(zoneRect)
		bbg := graphics.NewPaint()
		bbg.SetColor(mt.TaskbarBtnBgInk)
		canvas.DrawRect(br, bbg)
		bb := graphics.NewPaint()
		bb.SetColor(mt.TaskbarTextInk)
		cx := br.X + br.Width/2
		cy := br.Y + br.Height/2
		for i := -1; i <= 1; i++ {
			canvas.DrawLine(cx-4, cy+float32(i)*3, cx+4, cy+float32(i)*3, bb)
		}
	}
}

func (mc *MdiContainer) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !mc.Visible() {
		return
	}
	canvas.Save()
	if mc.NeedsLayout() {
		mc.ValidateLayout()
	}

	mt := mc.MdiTheme
	if mt == nil {
		mt = theme.DefaultMdiTheme
	}

	bg := graphics.NewPaint()
	bg.SetColor(mt.ContainerBgInk)
	canvas.DrawRoundedRect(rect, theme.DefaultMdiTheme.CornerRadius, theme.DefaultMdiTheme.CornerRadius, bg)

	// Paint dock zone backgrounds (the StackPanel items are painted by PaintChildren below)
	tbH := float32(0)
	if mc.ShowTaskbar {
		tbH = taskbarH
	}
	cx, cy, cw, ch := rect.X, rect.Y, rect.Width, rect.Height-tbH

	if mc.DocksFullHeight {
		// Draw left/right first (full height), then top/bottom (remaining width)
		if mc.LeftStack != nil && mc.LeftStack.ItemCount() > 0 {
			var sz float32
			if mc.LeftCollapsed {
				sz = dockAutoHideSize
			} else {
				sz = mc.LeftSize
			}
			if sz > 0 {
				dr := graphics.NewRect(cx, cy, sz, ch)
				dbg := graphics.NewPaint()
				dbg.SetColor(mt.DockBgInk)
				canvas.DrawRect(dr, dbg)
				mc.drawDockTitleBar(canvas, dr, "Left", DockLeft)
				sp := graphics.NewPaint()
				sp.SetColor(mt.DockEdgeInk)
				canvas.DrawLine(dr.X+dr.Width, dr.Y, dr.X+dr.Width, dr.Y+dr.Height, sp)
			}
			cx += sz
			cw -= sz
		}
		if mc.RightStack != nil && mc.RightStack.ItemCount() > 0 {
			var sz float32
			if mc.RightCollapsed {
				sz = dockAutoHideSize
			} else {
				sz = mc.RightSize
			}
			if sz > 0 {
				rx := cx + cw - sz
				dr := graphics.NewRect(rx, cy, sz, ch)
				dbg := graphics.NewPaint()
				dbg.SetColor(mt.DockBgInk)
				canvas.DrawRect(dr, dbg)
				mc.drawDockTitleBar(canvas, dr, "Right", DockRight)
				sp := graphics.NewPaint()
				sp.SetColor(mt.DockEdgeInk)
				canvas.DrawLine(dr.X, dr.Y, dr.X, dr.Y+dr.Height, sp)
			}
		}
		// Top/bottom are drawn between left/right
		topCx, topCw := cx, cw
		if mc.TopStack != nil && mc.TopStack.ItemCount() > 0 {
			var sz float32
			if mc.TopCollapsed {
				sz = dockAutoHideSize
			} else {
				sz = mc.TopSize
			}
			if sz > 0 {
				dr := graphics.NewRect(topCx, cy, topCw, sz)
				dbg := graphics.NewPaint()
				dbg.SetColor(mt.DockBgInk)
				canvas.DrawRect(dr, dbg)
				mc.drawDockTitleBar(canvas, dr, "Top", DockTop)
				sp := graphics.NewPaint()
				sp.SetColor(mt.DockEdgeInk)
				canvas.DrawLine(dr.X, dr.Y+dr.Height, dr.X+dr.Width, dr.Y+dr.Height, sp)
			}
			cy += sz
			ch -= sz
		}
		if mc.BottomStack != nil && mc.BottomStack.ItemCount() > 0 {
			var sz float32
			if mc.BottomCollapsed {
				sz = dockAutoHideSize
			} else {
				sz = mc.BottomSize
			}
			if sz > 0 {
				by := cy + ch - sz
				dr := graphics.NewRect(topCx, by, topCw, sz)
				dbg := graphics.NewPaint()
				dbg.SetColor(mt.DockBgInk)
				canvas.DrawRect(dr, dbg)
				mc.drawDockTitleBar(canvas, dr, "Bottom", DockBottom)
				sp := graphics.NewPaint()
				sp.SetColor(mt.DockEdgeInk)
				canvas.DrawLine(dr.X, dr.Y, dr.X+dr.Width, dr.Y, sp)
			}
			ch -= sz
		}
	} else {
		// Default mode: draw top/bottom first (full width), then left/right (remaining height)
		if mc.TopStack != nil && mc.TopStack.ItemCount() > 0 {
			var sz float32
			if mc.TopCollapsed {
				sz = dockAutoHideSize
			} else {
				sz = mc.TopSize
			}
			if sz > 0 {
				dr := graphics.NewRect(cx, cy, cw, sz)
				dbg := graphics.NewPaint()
				dbg.SetColor(mt.DockBgInk)
				canvas.DrawRect(dr, dbg)
				mc.drawDockTitleBar(canvas, dr, "Top", DockTop)
				sp := graphics.NewPaint()
				sp.SetColor(mt.DockEdgeInk)
				canvas.DrawLine(dr.X, dr.Y+dr.Height, dr.X+dr.Width, dr.Y+dr.Height, sp)
			}
			cy += sz
			ch -= sz
		}
		if mc.BottomStack != nil && mc.BottomStack.ItemCount() > 0 {
			var sz float32
			if mc.BottomCollapsed {
				sz = dockAutoHideSize
			} else {
				sz = mc.BottomSize
			}
			if sz > 0 {
				by := cy + ch - sz
				dr := graphics.NewRect(cx, by, cw, sz)
				dbg := graphics.NewPaint()
				dbg.SetColor(mt.DockBgInk)
				canvas.DrawRect(dr, dbg)
				mc.drawDockTitleBar(canvas, dr, "Bottom", DockBottom)
				sp := graphics.NewPaint()
				sp.SetColor(mt.DockEdgeInk)
				canvas.DrawLine(dr.X, dr.Y, dr.X+dr.Width, dr.Y, sp)
			}
			ch -= sz
		}
		if mc.LeftStack != nil && mc.LeftStack.ItemCount() > 0 {
			var sz float32
			if mc.LeftCollapsed {
				sz = dockAutoHideSize
			} else {
				sz = mc.LeftSize
			}
			if sz > 0 {
				dr := graphics.NewRect(cx, cy, sz, ch)
				dbg := graphics.NewPaint()
				dbg.SetColor(mt.DockBgInk)
				canvas.DrawRect(dr, dbg)
				mc.drawDockTitleBar(canvas, dr, "Left", DockLeft)
				sp := graphics.NewPaint()
				sp.SetColor(mt.DockEdgeInk)
				canvas.DrawLine(dr.X+dr.Width, dr.Y, dr.X+dr.Width, dr.Y+dr.Height, sp)
			}
			cx += sz
			cw -= sz
		}
		if mc.RightStack != nil && mc.RightStack.ItemCount() > 0 {
			var sz float32
			if mc.RightCollapsed {
				sz = dockAutoHideSize
			} else {
				sz = mc.RightSize
			}
			if sz > 0 {
				rx := cx + cw - sz
				dr := graphics.NewRect(rx, cy, sz, ch)
				dbg := graphics.NewPaint()
				dbg.SetColor(mt.DockBgInk)
				canvas.DrawRect(dr, dbg)
				mc.drawDockTitleBar(canvas, dr, "Right", DockRight)
				sp := graphics.NewPaint()
				sp.SetColor(mt.DockEdgeInk)
				canvas.DrawLine(dr.X, dr.Y, dr.X, dr.Y+dr.Height, sp)
			}
		}
	}

	// Paint floating windows + StackPanels (children)
	mc.PaintChildren(canvas)

	// Draw taskbar
	if mc.ShowTaskbar {
		mc.drawTaskbar(canvas, rect)
	}

	// Highlight drop zone during drag (on top of everything)
	if mc.DropHighlight != nil {
		zr := mc.DropHighlight.ZoneRect
		if zr.Width > 0 && zr.Height > 0 {
			hl := graphics.NewPaint()
			hl.SetColor(graphics.Color(0x884FC3F7))
			canvas.DrawRect(zr, hl)
			hlBorder := graphics.NewPaint()
			hlBorder.SetStyle(paintstyle.Stroke)
			hlBorder.SetStrokeWidth(2)
			hlBorder.SetColor(mt.TaskbarActiveInk)
			canvas.DrawRect(zr, hlBorder)
		}
	}

	canvas.Restore()
}

func (mc *MdiContainer) drawTaskbar(canvas *graphics.Canvas, rect graphics.Rect) {
	mt := mc.MdiTheme
	if mt == nil {
		mt = theme.DefaultMdiTheme
	}
	tbRect := graphics.NewRect(rect.X, rect.Y+rect.Height-taskbarH, rect.Width, taskbarH)
	tbBg := graphics.NewPaint()
	tbBg.SetColor(mt.WinTitleInactive)
	canvas.DrawRect(tbRect, tbBg)
	tbLine := graphics.NewPaint()
	tbLine.SetColor(mt.TabEdgeInk)
	canvas.DrawLine(tbRect.X, tbRect.Y, tbRect.X+tbRect.Width, tbRect.Y, tbLine)

	font := theme.DefaultTheme.BaseFont
	if font == nil {
		font = graphics.SystemFont
	}
	ascent := font.Baseline()
	btnX := tbRect.X + 4
	btnH := float32(taskbarH - 4)

	for _, mw := range mc.allWindows() {
		if mw == nil {
			continue
		}
		label := mw.Title
		if label == "" {
			label = "Window"
		}
		iconSize := float32(14)
		iconExtra := float32(0)
		if mw.iconSVG != nil {
			iconExtra = iconSize + 4
		}
		textW := font.SimpleWidth(label) + 16 + iconExtra
		if textW > 200 {
			textW = 200
		}
		btnRect := graphics.NewRect(btnX, tbRect.Y+2, textW, btnH)

		btnBg := graphics.NewPaint()
		if mw.Visible() && !mw.Minimized {
			btnBg.SetColor(mt.TaskbarBtnBgInk)
		} else {
			btnBg.SetColor(mt.TaskbarBtnHover)
		}
		canvas.DrawRect(btnRect, btnBg)

		btnBorder := graphics.NewPaint()
		btnBorder.SetColor(mt.TaskbarBtnEdge)
		canvas.DrawRect(btnRect, btnBorder)

		tx := btnRect.X + 8
		if mw.iconSVG != nil {
			meui.DrawSVG(canvas, mw.iconSVG, btnRect.X+4, btnRect.Y+(btnH-iconSize)/2, iconSize, iconSize)
			tx += iconSize + 2
		} else if mw.Icon != "" {
			fb := graphics.NewPaint()
			fb.SetColor(mt.TaskbarActiveInk)
			fb.SetStyle(paintstyle.Fill)
			cy := btnRect.Y + btnH/2
			cx := btnRect.X + 4 + iconSize/2
			canvas.DrawCircle(cx, cy, iconSize/2-1, fb)
			tx += iconSize + 2
		}
		tp := graphics.NewPaint()
		if mw.Visible() && !mw.Minimized {
			tp.SetColor(mt.WinTitleTextInk)
		} else {
			tp.SetColor(mt.TaskbarTextInk)
		}
		labelY := btnRect.Y + (btnH-ascent)/2 + ascent
		wl := graphics.NewText(label, &graphics.TextDecoration{Font: font, OnBackgroundInk: tp.Color()})
		wl.Draw(canvas, tx, labelY)
		btnX += textW + 4
	}

	// Notification area (far right of taskbar)
	notifStartX := tbRect.X + tbRect.Width
	if len(mc.Notifications) > 0 {
		notifSize := float32(20)
		notifGap := float32(4)
		totalNotifW := float32(len(mc.Notifications)) * (notifSize + notifGap)
		notifStartX = tbRect.X + tbRect.Width - totalNotifW - 4

		for i, n := range mc.Notifications {
			nx := notifStartX + float32(i)*(notifSize+notifGap)
			nRect := graphics.NewRect(nx, tbRect.Y+2, notifSize, notifSize)
			nbg := graphics.NewPaint()
			nbg.SetColor(mt.TaskbarBtnHover)
			canvas.DrawRect(nRect, nbg)
			nBorder := graphics.NewPaint()
			nBorder.SetColor(mt.TabEdgeInk)
			canvas.DrawRect(nRect, nBorder)
			if n.IconSVG != nil {
				meui.DrawSVG(canvas, n.IconSVG, nRect.X+2, nRect.Y+2, notifSize-4, notifSize-4)
			} else if n.Icon != "" {
				if doc, err := meui.LoadSVGByName(n.Icon); err == nil {
					meui.DrawSVG(canvas, doc, nRect.X+2, nRect.Y+2, notifSize-4, notifSize-4)
				}
			}
		}
	}

	// Dock toggle buttons (between notifications and window buttons)
	dockBtnRight := notifStartX - 4
	// First: individual buttons per dock (using text labels for reliability)
	type dockBtn struct {
		pos   DockPosition
		label string
	}
	dockBtns := []dockBtn{}
	for _, d := range []DockPosition{DockLeft, DockRight, DockTop, DockBottom} {
		sp := *mc.dockStack(d)
		if sp != nil && sp.ItemCount() > 0 {
			switch d {
			case DockLeft:
				dockBtns = append(dockBtns, dockBtn{DockLeft, "L"})
			case DockRight:
				dockBtns = append(dockBtns, dockBtn{DockRight, "R"})
			case DockTop:
				dockBtns = append(dockBtns, dockBtn{DockTop, "T"})
			case DockBottom:
				dockBtns = append(dockBtns, dockBtn{DockBottom, "B"})
			}
		}
	}
	if len(dockBtns) > 0 {
		// First compute total width for all buttons
		btnW := float32(26)
		gap := float32(2)
		totalBtnsW := float32(len(dockBtns))*(btnW+gap) + 28 + gap // + toggle-all button
		startX := dockBtnRight - totalBtnsW

		// Individual dock buttons
		for i, db := range dockBtns {
			x := startX + float32(i)*(btnW+gap)
			dbRect := graphics.NewRect(x, tbRect.Y+2, btnW, btnH)
			collapsed := mc.isDockCollapsed(db.pos)
			dbBg := graphics.NewPaint()
			if collapsed {
				dbBg.SetColor(mt.TaskbarBtnBgInk)
			} else {
				dbBg.SetColor(mt.DockSplitInk)
			}
			canvas.DrawRect(dbRect, dbBg)
			dbBorder := graphics.NewPaint()
			dbBorder.SetColor(mt.TaskbarBtnEdge)
			canvas.DrawRect(dbRect, dbBorder)
			dbTp := graphics.NewPaint()
			dbTp.SetColor(mt.WinTextSecondary)
			dbt := graphics.NewText(db.label, &graphics.TextDecoration{Font: font, OnBackgroundInk: dbTp.Color()})
			dbt.Draw(canvas, dbRect.X+(btnW-font.SimpleWidth(db.label))/2,
				dbRect.Y+(btnH-ascent)/2+ascent)
		}

		// Toggle-all button (rightmost)
		toggleX := startX + float32(len(dockBtns))*(btnW+gap)
		toggleRect := graphics.NewRect(toggleX, tbRect.Y+2, 28, btnH)
		tbg := graphics.NewPaint()
		tbg.SetColor(mt.TaskbarBtnBgInk)
		canvas.DrawRect(toggleRect, tbg)
		tbb := graphics.NewPaint()
		tbb.SetColor(mt.TaskbarBtnEdge)
		canvas.DrawRect(toggleRect, tbb)
		tp := graphics.NewPaint()
		tp.SetColor(mt.WinTextSecondary)
		at := graphics.NewText("A", &graphics.TextDecoration{Font: font, OnBackgroundInk: tp.Color()})
		at.Draw(canvas, toggleX+(28-font.SimpleWidth("A"))/2,
			toggleRect.Y+(btnH-ascent)/2+ascent)
	}
}

// expandAll expands all collapsed docks and clears auto-hide mode.
func (mc *MdiContainer) expandAll() {
	for _, d := range []DockPosition{DockLeft, DockRight, DockTop, DockBottom} {
		sp := *mc.dockStack(d)
		if sp == nil || sp.ItemCount() == 0 {
			continue
		}
		if mc.isDockCollapsed(d) {
			switch d {
			case DockLeft:
				mc.LeftCollapsed = false
				mc.LeftAutoHide = false
			case DockRight:
				mc.RightCollapsed = false
				mc.RightAutoHide = false
			case DockTop:
				mc.TopCollapsed = false
				mc.TopAutoHide = false
			case DockBottom:
				mc.BottomCollapsed = false
				mc.BottomAutoHide = false
			}
			sp.SetVisible(true)
		}
	}
	mc.SetNeedsLayout()
	mc.MarkForRedraw()
}

// collapseAll collapses all expanded docks and enables auto-hide.
func (mc *MdiContainer) collapseAll() {
	for _, d := range []DockPosition{DockLeft, DockRight, DockTop, DockBottom} {
		sp := *mc.dockStack(d)
		if sp == nil || sp.ItemCount() == 0 {
			continue
		}
		if !mc.isDockCollapsed(d) {
			switch d {
			case DockLeft:
				mc.LeftCollapsed = true
				mc.LeftAutoHide = true
			case DockRight:
				mc.RightCollapsed = true
				mc.RightAutoHide = true
			case DockTop:
				mc.TopCollapsed = true
				mc.TopAutoHide = true
			case DockBottom:
				mc.BottomCollapsed = true
				mc.BottomAutoHide = true
			}
			sp.SetVisible(false)
		}
	}
	mc.SetNeedsLayout()
	mc.MarkForRedraw()
}

// toggleAllDocks expands all collapsed docks, or collapses all expanded ones.
func (mc *MdiContainer) toggleAllDocks() {
	hasExpanded := false
	for _, d := range []DockPosition{DockLeft, DockRight, DockTop, DockBottom} {
		sp := *mc.dockStack(d)
		if sp == nil || sp.ItemCount() == 0 {
			continue
		}
		if !mc.isDockCollapsed(d) {
			hasExpanded = true
			break
		}
	}
	if hasExpanded {
		mc.collapseAll()
	} else {
		mc.expandAll()
	}
}

// makeTabDetachHandler returns a callback for TabPanel.OnTabDetached
// that creates a new floating MdiWindow from the detached tab content.
// Also handles unmerge when a TabPanel is reduced to a single tab.
func (mc *MdiContainer) makeTabDetachHandler(sp *StackPanel) func(int, string, meui.Widget, float32, float32) {
	container := mc
	return func(tabIndex int, title string, content meui.Widget, localX, localY float32) {
		if content == nil {
			return
		}

		// Create new floating window from the detached tab
		newWin := NewMdiWindow()
		newWin.SetTitle(title)
		newWin.SetContent(content)
		spFrame := sp.FrameRect()
		newWin.SetFrameRect(graphics.NewRect(
			localX+spFrame.X-150,
			localY+spFrame.Y-20,
			300, 200))
		container.AddChild(newWin)

		// Check if the TabPanel now has only 1 tab → unmerge it
		for si, sitem := range sp.Items {
			if stp, ok := sitem.(*TabPanel); ok && len(stp.tabs) == 1 {
				remaining := stp.tabs[0]
				unmerged := NewMdiWindow()
				unmerged.SetTitle(remaining.Title)
				unmerged.SetContent(remaining.Content)
				remaining.Content = nil
				sp.Items[si] = unmerged
				unmerged.SetParent(sp)
				sp.SetNeedsLayout()
				sp.MarkForRedraw()
				break
			}
		}

		container.MarkForRedraw()
	}
}

func (mc *MdiContainer) BringToFront(w meui.Widget) {
	if mw, ok := w.(*MdiWindow); ok && mw.DockPos == DockCenter {
		mc.BaseContainer.MoveChildToFront(w)
	}
}

// Cascade arranges all visible MdiWindows in a cascading diagonal pattern.
func (mc *MdiContainer) Cascade() {
	frame := mc.FrameRect()
	step := float32(30)
	x, y := step, step
	for _, child := range mc.Children() {
		if mw, ok := child.(*MdiWindow); ok && !mw.Minimized && mw.DockPos == DockCenter {
			mw.Maximized = false
			mw.SetVisible(true)
			mw.SetFrameRect(graphics.NewRect(x, y, 300, 200))
			if x+300 > frame.Width {
				x = step
				y += step
			} else {
				x += step
				y += step
			}
		}
	}
}

func (mc *MdiContainer) Tile() {
	frame := mc.FrameRect()
	var vis []*MdiWindow
	for _, child := range mc.Children() {
		if mw, ok := child.(*MdiWindow); ok && !mw.Minimized && mw.DockPos == DockCenter {
			vis = append(vis, mw)
		}
	}
	if len(vis) == 0 {
		return
	}
	cols := 1
	for cols*cols < len(vis) {
		cols++
	}
	rows := (len(vis) + cols - 1) / cols
	w := frame.Width / float32(cols)
	h := frame.Height / float32(rows)
	for i, mw := range vis {
		mw.Maximized = false
		mw.SetVisible(true)
		mw.SetFrameRect(graphics.NewRect(float32(i%cols)*w, float32(i/cols)*h, w, h))
	}
}

// ── Auto-hide hover ─────────────────────────────────────────────

// handleAutoHideHover checks if the mouse is over an auto-hide tab
// and expands/collapses the dock accordingly.
func (mc *MdiContainer) handleAutoHideHover(x, y float32) {
	// Check if mouse is over any collapsed auto-hide dock's tab area
	for _, d := range []DockPosition{DockLeft, DockRight, DockTop, DockBottom} {
		var autoHide bool
		switch d {
		case DockLeft:
			autoHide = mc.LeftAutoHide
		case DockRight:
			autoHide = mc.RightAutoHide
		case DockTop:
			autoHide = mc.TopAutoHide
		case DockBottom:
			autoHide = mc.BottomAutoHide
		}
		if !autoHide {
			continue
		}
		if !mc.isDockCollapsed(d) {
			continue
		}

		// Check if mouse is over the tab area (the thin edge strip)
		tbH := float32(0)
		if mc.ShowTaskbar {
			tbH = taskbarH
		}
		w, hh := mc.Width(), mc.Height()-tbH

		var overTab bool
		switch d {
		case DockLeft:
			overTab = x >= 0 && x <= dockAutoHideSize && y >= 0 && y <= hh
		case DockRight:
			overTab = x >= w-dockAutoHideSize && x <= w && y >= 0 && y <= hh
		case DockTop:
			overTab = y >= 0 && y <= dockAutoHideSize && x >= 0 && x <= w
		case DockBottom:
			overTab = y >= hh-dockAutoHideSize && y <= hh && x >= 0 && x <= w
		}
		if overTab {
			mc.tempExpand(d)
			return
		}
	}

	// Check if mouse has left a temporarily expanded auto-hide dock
	expanded := mc.expandedDock
	if expanded == DockCenter {
		return
	}
	if mc.isDockCollapsed(expanded) {
		mc.expandedDock = DockCenter
		return
	}

	// Determine if mouse is still within the expanded dock zone
	var inZone bool
	tbH := float32(0)
	if mc.ShowTaskbar {
		tbH = taskbarH
	}
	w, hh := mc.Width(), mc.Height()-tbH

	switch expanded {
	case DockLeft:
		inZone = x >= 0 && x <= mc.LeftSize && y >= 0 && y <= hh
	case DockRight:
		inZone = x >= w-mc.RightSize && x <= w && y >= 0 && y <= hh
	case DockTop:
		inZone = y >= 0 && y <= mc.TopSize && x >= 0 && x <= w
	case DockBottom:
		inZone = y >= hh-mc.BottomSize && y <= hh && x >= 0 && x <= w
	}
	if inZone {
		mc.cancelExpandTimer()
	} else if mc.expandTimer == nil {
		mc.scheduleAutoHide(expanded)
	}
}

// ── Context menu ───────────────────────────────────────────────

func (mc *MdiContainer) OnMouseDown(x, y float32, button int) bool {
	if !mc.Enabled() {
		return false
	}
	// Check dock resize handle
	if dp := mc.dockResizeHit(x, y); dp != DockCenter {
		mc.resizingDock = dp
		switch dp {
		case DockLeft:
			mc.dockResizeOrig = mc.LeftSize
			mc.dockResizeStart = x
		case DockRight:
			mc.dockResizeOrig = mc.RightSize
			mc.dockResizeStart = x
		case DockTop:
			mc.dockResizeOrig = mc.TopSize
			mc.dockResizeStart = y
		case DockBottom:
			mc.dockResizeOrig = mc.BottomSize
			mc.dockResizeStart = y
		}
		return true
	}
	// Check dock title bar pin button click (toggle collapse)
	if pos := mc.dockPinBtnHit(x, y); pos != DockCenter {
		mc.toggleDockCollapse(pos)
		return true
	}
	// Check dock title bar menu button click
	if pos := mc.dockMenuBtnHit(x, y); pos != DockCenter {
		mc.showDockTitleMenu(pos, x, y)
		return true
	}
	// Check taskbar click
	if mc.ShowTaskbar && y >= mc.Height()-taskbarH {
		mc.handleTaskbarClick(x, y)
		return true
	}
	// Right-click → context menu
	if button == 1 {
		mc.showMdiContextMenu(x, y)
		return true
	}
	return false
}

func (mc *MdiContainer) OnMouseMove(x, y float32) {
	if !mc.Enabled() {
		return
	}
	// Auto-hide hover detection: expand/collapse docks on hover
	mc.handleAutoHideHover(x, y)
	// Handle dock resize drag
	if mc.resizingDock != DockCenter {
		var delta float32
		switch mc.resizingDock {
		case DockLeft:
			delta = x - mc.dockResizeStart
		case DockRight:
			delta = mc.dockResizeStart - x
		case DockTop:
			delta = y - mc.dockResizeStart
		case DockBottom:
			delta = mc.dockResizeStart - y
		}
		newSize := mc.dockResizeOrig + delta
		if newSize < dockMinSize {
			newSize = dockMinSize
		}
		maxSize := float32(400)
		if newSize > maxSize {
			newSize = maxSize
		}
		switch mc.resizingDock {
		case DockLeft:
			mc.LeftSize = newSize
		case DockRight:
			mc.RightSize = newSize
		case DockTop:
			mc.TopSize = newSize
		case DockBottom:
			mc.BottomSize = newSize
		}
		mc.SetNeedsLayout()
		mc.MarkForRedraw()
		return
	}
	// Update cursor for dock resize handles
	_ = mc.dockResizeHit(x, y)
}

func (mc *MdiContainer) OnMouseUp(x, y float32, button int) bool {
	mc.resizingDock = DockCenter
	return true
}

// OnKeyDown handles keyboard shortcut: Ctrl+Shift+D toggles left dock collapse.
func (mc *MdiContainer) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if mod.ControlDown() && mod.ShiftDown() && keyCode == platform.KeyD {
		mc.toggleDockCollapse(DockLeft)
		return true
	}
	return false
}

func (mc *MdiContainer) handleTaskbarClick(x, y float32) {
	allWindows := mc.allWindows()
	btnX := float32(4)
	font := theme.DefaultTheme.BaseFont
	if font == nil {
		font = graphics.SystemFont
	}
	for _, mw := range allWindows {
		if mw == nil {
			continue
		}
		label := mw.Title
		if label == "" {
			label = "Window"
		}
		textW := font.SimpleWidth(label) + 16
		if textW > 200 {
			textW = 200
		}
		if x >= btnX && x <= btnX+textW {
			if mw.Visible() && !mw.Minimized {
				mw.Minimized = true
				mw.SetVisible(false)
			} else {
				mw.Minimized = false
				mw.SetVisible(true)
				mc.BringToFront(mw)
			}
			mc.MarkForRedraw()
			return
		}
		btnX += textW + 4
	}

	// Check notification area clicks (far right)
	if len(mc.Notifications) > 0 {
		notifSize := float32(20)
		notifGap := float32(4)
		totalNotifW := float32(len(mc.Notifications)) * (notifSize + notifGap)
		frame := mc.FrameRect()
		notifStartX := frame.Width - totalNotifW - 4
		for i, n := range mc.Notifications {
			nx := notifStartX + float32(i)*(notifSize+notifGap)
			if x >= nx && x <= nx+notifSize {
				if n.OnClick != nil {
					n.OnClick()
				}
				return
			}
		}
	}

	// Check dock toggle buttons (far right, individual + toggle-all)
	type dockBtnHit struct {
		pos DockPosition
		x   float32
	}
	var dockBtns []dockBtnHit
	for _, d := range []DockPosition{DockLeft, DockRight, DockTop, DockBottom} {
		sp := *mc.dockStack(d)
		if sp != nil && sp.ItemCount() > 0 {
			dockBtns = append(dockBtns, dockBtnHit{d, 0})
		}
	}
	if len(dockBtns) > 0 {
		btnW := float32(26)
		gap := float32(2)
		totalBtnsW := float32(len(dockBtns))*(btnW+gap) + 28 + gap
		frame := mc.FrameRect()
		// Account for notification area
		notifOffset := float32(0)
		if len(mc.Notifications) > 0 {
			notifSize := float32(20)
			notifGap := float32(4)
			notifOffset = float32(len(mc.Notifications))*(notifSize+notifGap) + 4
		}
		startX := frame.Width - totalBtnsW - notifOffset - 4
		for i := range dockBtns {
			dockBtns[i].x = startX + float32(i)*(btnW+gap)
		}
		for _, db := range dockBtns {
			if x >= db.x && x <= db.x+btnW {
				mc.toggleDockCollapse(db.pos)
				return
			}
		}
		// Toggle-all button
		toggleX := startX + float32(len(dockBtns))*(btnW+gap)
		if x >= toggleX && x <= toggleX+28 {
			mc.toggleAllDocks()
			return
		}
	}
}

func (mc *MdiContainer) allWindows() []*MdiWindow {
	var wl []*MdiWindow
	// Collect from dock stacks
	for _, sp := range []*StackPanel{mc.TopStack, mc.BottomStack, mc.LeftStack, mc.RightStack} {
		if sp == nil {
			continue
		}
		for _, item := range sp.Items {
			if mw, ok := item.(*MdiWindow); ok {
				wl = append(wl, mw)
			}
		}
	}
	// Floating windows (children)
	for _, child := range mc.Children() {
		if mw, ok := child.(*MdiWindow); ok {
			wl = append(wl, mw)
		}
	}
	return wl
}

func (mc *MdiContainer) showMdiContextMenu(x, y float32) {
	items := []MenuItem{
		{Label: "Add New Window", Enabled: true, Action: func() {
			win := NewMdiWindow()
			win.SetTitle("New Window")
			win.SetFrameRect(graphics.NewRect(x, y, 300, 200))
			ta := NewTextArea()
			ta.SetText("New document")
			ta.SetFrameRect(graphics.NewRect(0, 0, 300, 172))
			win.SetContent(ta)
			mc.AddChild(win)
			mc.MarkForRedraw()
		}},
		{IsSep: true},
		{Label: "Show Modal Popup", Enabled: true, Action: func() {
			showModalPopup(mc)
		}},
		{Label: "Show Non-Modal Popup", Enabled: true, Action: func() {
			showNonModalPopup(mc)
		}},
		{IsSep: true},
		{Label: "Toggle Taskbar", Enabled: true, Action: func() {
			mc.ShowTaskbar = !mc.ShowTaskbar
			mc.SetNeedsLayout()
			mc.MarkForRedraw()
		}},
		{Label: "Cascade", Enabled: true, Action: func() { mc.Cascade() }},
		{Label: "Tile", Enabled: true, Action: func() { mc.Tile() }},
	}
	if mc.ctxMenu == nil {
		mc.ctxMenu = NewContextMenu()
	}
	mc.ctxMenu.SetItems(items)
	if win := mc.Window(); win != nil {
		mc.ctxMenu.SetOnDismiss(func() {
			win.DismissPopupContextMenu()
		})
		winX, winY := mc.FromLocal(x, y)
		win.ShowPopupContextMenu(mc.ctxMenu, winX, winY)
	}
	mc.MarkForRedraw()
}

// showDockTitleMenu shows the popup menu for a dock zone title bar button.
func (mc *MdiContainer) showDockTitleMenu(pos DockPosition, x, y float32) {
	collapsed := mc.isDockCollapsed(pos)
	collapseLabel := "Collapse dock"
	if collapsed {
		collapseLabel = "Expand dock"
	}
	items := []MenuItem{
		{Label: collapseLabel, Enabled: true, Action: func() {
			mc.toggleDockCollapse(pos)
		}},
		{IsSep: true},
		{Label: "Close dock", Enabled: true, Action: func() {
			mc.undockAllFrom(pos)
		}},
	}
	if mc.dockCtxMenu == nil {
		mc.dockCtxMenu = NewContextMenu()
	}
	mc.dockCtxMenu.SetItems(items)
	if win := mc.Window(); win != nil {
		mc.dockCtxMenu.SetOnDismiss(func() {
			win.DismissPopupContextMenu()
		})
		winX, winY := mc.FromLocal(x, y)
		win.ShowPopupContextMenu(mc.dockCtxMenu, winX, winY)
	}
	mc.MarkForRedraw()
}
