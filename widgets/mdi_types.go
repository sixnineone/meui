package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
)

// ── Drop System ─────────────────────────────────────────────────

// ReleaseCallback detaches a child widget from its current parent.
// Created when a drag starts so the drop target can release the child
// from the original parent before reparenting.
type ReleaseCallback func()

// DropZoneType categorizes drop zones.
type DropZoneType int

const (
	DropDock  DropZoneType = iota // Dock to edge zone
	DropMerge                     // Merge into existing TabPanel or MdiWindow
	DropFloat                     // Drop as floating window (no-op)
)

// DropZoneInfo describes where a dragged window should land.
type DropZoneInfo struct {
	Type     DropZoneType
	DockPos  DockPosition  // for DropDock
	Target   meui.Widget   // target widget for DropMerge (MdiWindow or TabPanel)
	Stack    *StackPanel   // StackPanel to insert into
	StackIdx int           // index within StackPanel
	ZoneRect graphics.Rect // zone rect in MdiContainer-local coords (for highlight)
}

// ── MdiWindow ───────────────────────────────────────────────────

// DockPosition indicates where a window is docked.
type DockPosition int

const (
	DockCenter DockPosition = iota
	DockLeft
	DockRight
	DockTop
	DockBottom
)

// DockEdge is the type for dock edge positions (kept for backward compatibility).
type DockEdge = DockPosition

// NotificationItem represents an icon in the taskbar notification area.
type NotificationItem struct {
	Icon    string // SVG icon name
	Tooltip string
	OnClick func()
	IconSVG *meui.SVGDocument // cached SVG document (loaded on demand)
}
