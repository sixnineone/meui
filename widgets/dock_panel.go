package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/theme"
)

// DEPRECATED: DockPanel is a legacy component with static layout only.
// It has no mouse event handlers (no drag, no reorder, no undock).
// Use MdiContainer (widgets/mdi_container.go) instead, which provides
// a full MDI system with 4 dock zones, auto-hide, drag-to-dock,
// taskbar, cascade/tile arrangement, and tab detach support.

// DockEdge is the type for dock edge positions (kept for backward compatibility).

// DockPanel is a container that manages docked child panels.
// Each child is wrapped in a title bar and can be dragged to re-dock or undock.
type DockPanel struct {
	meui.BaseContainer
	Left   []meui.Widget
	Right  []meui.Widget
	Top    []meui.Widget
	Bottom []meui.Widget
	Center meui.Widget
	// Sizes for docked areas
	LeftSize   float32
	RightSize  float32
	TopSize    float32
	BottomSize float32
}

func NewDockPanel() *DockPanel {
	return &DockPanel{
		BaseContainer: meui.BaseContainer{BaseWidget: *meui.NewBaseWidget()},
		LeftSize:      200,
		RightSize:     200,
		TopSize:       150,
		BottomSize:    150,
	}
}

func (dp *DockPanel) SetCenter(w meui.Widget) {
	dp.Center = w
	if w != nil {
		// AddChild (not bare SetParent): PaintChildren iterates the
		// container's child list, so SetParent alone left docked zones
		// laid out but never painted.
		dp.AddChild(w)
	}
	dp.SetNeedsLayout()
	dp.MarkForRedraw()
}

func (dp *DockPanel) AddToEdge(edge DockEdge, title string, w meui.Widget) {
	switch edge {
	case DockLeft:
		dp.Left = append(dp.Left, w)
	case DockRight:
		dp.Right = append(dp.Right, w)
	case DockTop:
		dp.Top = append(dp.Top, w)
	case DockBottom:
		dp.Bottom = append(dp.Bottom, w)
	}
	if w != nil {
		dp.AddChild(w)
	}
	dp.SetNeedsLayout()
	dp.MarkForRedraw()
}

func (dp *DockPanel) ValidateLayout() {
	if !dp.NeedsLayout() {
		return
	}
	frame := dp.FrameRect()
	left := dp.LeftSize
	if len(dp.Left) == 0 {
		left = 0
	}
	right := dp.RightSize
	if len(dp.Right) == 0 {
		right = 0
	}
	top := dp.TopSize
	if len(dp.Top) == 0 {
		top = 0
	}
	bottom := dp.BottomSize
	if len(dp.Bottom) == 0 {
		bottom = 0
	}

	// Position docked areas
	x, y := float32(0), float32(0)
	w, h := frame.Width, frame.Height

	if len(dp.Top) > 0 {
		rect := graphics.NewRect(x, y, w, top)
		for _, child := range dp.Top {
			child.SetFrameRect(rect)
			if vc, ok := child.(interface{ ValidateLayout() }); ok {
				vc.ValidateLayout()
			}
		}
		y += top
		h -= top
	}

	if len(dp.Bottom) > 0 {
		bottomRect := graphics.NewRect(x, y+h-bottom, w, bottom)
		for _, child := range dp.Bottom {
			child.SetFrameRect(bottomRect)
			if vc, ok := child.(interface{ ValidateLayout() }); ok {
				vc.ValidateLayout()
			}
		}
		h -= bottom
	}

	if len(dp.Left) > 0 {
		leftRect := graphics.NewRect(x, y, left, h)
		for _, child := range dp.Left {
			child.SetFrameRect(leftRect)
			if vc, ok := child.(interface{ ValidateLayout() }); ok {
				vc.ValidateLayout()
			}
		}
		x += left
		w -= left
	}

	if len(dp.Right) > 0 {
		rightRect := graphics.NewRect(x+w-right, y, right, h)
		for _, child := range dp.Right {
			child.SetFrameRect(rightRect)
			if vc, ok := child.(interface{ ValidateLayout() }); ok {
				vc.ValidateLayout()
			}
		}
		w -= right
	}

	// Center area
	if dp.Center != nil {
		dp.Center.SetFrameRect(graphics.NewRect(x, y, w, h))
		if vc, ok := dp.Center.(interface{ ValidateLayout() }); ok {
			vc.ValidateLayout()
		}
	}
	dp.ClearNeedsLayout()
}

func (dp *DockPanel) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !dp.Visible() {
		return
	}
	canvas.Save()
	// Background
	bg := graphics.NewPaint()
	bg.SetColor(graphics.Color(0xFF252525))
	canvas.DrawRoundedRect(rect, theme.DefaultMdiTheme.CornerRadius, theme.DefaultMdiTheme.CornerRadius, bg)

	// Paint children - Panel's default PaintChildren handles this
	// Draw dock area separators
	line := graphics.NewPaint()
	line.SetStyle(paintstyle.Stroke)
	line.SetStrokeWidth(1)
	line.SetColor(graphics.Color(0xFF444444))

	frame := dp.FrameRect()
	if len(dp.Left) > 0 && dp.LeftSize > 0 {
		canvas.DrawLine(dp.LeftSize, 0, dp.LeftSize, frame.Height, line)
	}
	if len(dp.Right) > 0 && dp.RightSize > 0 {
		x := frame.Width - dp.RightSize
		canvas.DrawLine(x, 0, x, frame.Height, line)
	}
	if len(dp.Top) > 0 && dp.TopSize > 0 {
		canvas.DrawLine(0, dp.TopSize, frame.Width, dp.TopSize, line)
	}
	if len(dp.Bottom) > 0 && dp.BottomSize > 0 {
		y := frame.Height - dp.BottomSize
		canvas.DrawLine(0, y, frame.Width, y, line)
	}

	// Paint children
	dp.PaintChildren(canvas)

	canvas.Restore()
}

// ── Popup helpers ────────────────────────────────────────────────

func showModalPopup(mc *MdiContainer) {
	mc.MarkForRedraw()
	win := mc.Window()
	if win == nil {
		return
	}
	contentPanel := meui.NewPanel()
	contentPanel.AddChild(NewTextArea().SetText("Modal notes..."))
	okBtn := NewButton()
	okBtn.SetText("OK")
	popup := NewModalPopupPanel("Modal Popup", "Enter details:", contentPanel, okBtn)
	popup.SetDismissFn(func() {
		if popup.Modal != nil {
			win.DismissModal(popup.Modal)
		}
	})
	okBtn.OnClicked = func() {
		if popup.Modal != nil {
			win.DismissModal(popup.Modal)
		}
	}
	popup.Modal = win.ShowModal(popup, 280, 200)
}

func showNonModalPopup(mc *MdiContainer) {
	mc.MarkForRedraw()
	win := mc.Window()
	if win == nil {
		return
	}
	contentPanel := meui.NewPanel()
	contentPanel.AddChild(NewCheckBox().SetText("Enable feature"))
	okBtn := NewButton()
	okBtn.SetText("Close")
	popup := NewModalPopupPanel("Non-Modal Popup", "Floating option:", contentPanel, okBtn)
	popup.SetDismissFn(func() {
		if popup.Modal != nil {
			win.DismissModal(popup.Modal)
		}
	})
	okBtn.OnClicked = func() {
		if popup.Modal != nil {
			win.DismissModal(popup.Modal)
		}
	}
	popup.Modal = win.ShowNonModal(popup, 260, 160)
}
