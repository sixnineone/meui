package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

// MdiWindow is a moveable, resizable child window with a title bar.
type MdiWindow struct {
	meui.BaseContainer
	Title       string
	Content     meui.Widget
	Closable    bool
	Minimizable bool
	Maximizable bool
	Minimized   bool
	Maximized   bool
	Resizable   bool
	PrevRect    graphics.Rect
	onClose     func()
	dragging    bool
	resizing    bool
	resizeEdge  int
	dragStartX  float32
	dragStartY  float32
	startFrame  graphics.Rect
	startRootX  float32
	startRootY  float32
	DockPos     DockPosition
	DockIndex   int
	DockSize    float32
	dockRestore graphics.Rect
	container   *MdiContainer
	MdiTheme    *theme.MdiTheme

	releaseCallback ReleaseCallback // set during drag to detach from current parent

	// Icon
	Icon    string
	iconSVG *meui.SVGDocument

	// Custom title bar buttons (right side, before minimize)
	customButtons []TitleBarButton
	textCache     *graphics.Text
	textCacheVer  uint64
}

// TitleBarButton defines a custom button in the window title bar.
type TitleBarButton struct {
	Icon    string // SVG icon name
	Tooltip string
	OnClick func()
}

const (
	titleBarH       = 28
	dockedTitleBarH = 22
	titleBtnSize    = 16
	titleBtnSpace   = 4
	resizeH         = 6
	mdiMinW         = 120
	mdiMinH         = 60
)

// titleBarHeight returns the title bar height, which is smaller when docked.
func (mw *MdiWindow) titleBarHeight() float32 {
	if mw.DockPos != DockCenter {
		return dockedTitleBarH
	}
	return titleBarH
}

// Resize edge constants
const (
	resizeNone        = 0
	resizeLeft        = 1 << 0
	resizeRight       = 1 << 1
	resizeTop         = 1 << 2
	resizeBottom      = 1 << 3
	resizeTopLeft     = resizeTop | resizeLeft
	resizeTopRight    = resizeTop | resizeRight
	resizeBottomLeft  = resizeBottom | resizeLeft
	resizeBottomRight = resizeBottom | resizeRight
)

func NewMdiWindow() *MdiWindow {
	return &MdiWindow{
		BaseContainer: meui.BaseContainer{BaseWidget: *meui.NewBaseWidget()},
		Closable:      true,
		Minimizable:   true,
		Maximizable:   true,
		Title:         "Window",
		MdiTheme:      theme.DefaultMdiTheme,
	}
}

func (mw *MdiWindow) SetTitle(title string) *MdiWindow {
	mw.Title = title
	mw.textCache = nil
	mw.MarkForRedraw()
	return mw
}

func (mw *MdiWindow) SetIcon(icon string) *MdiWindow {
	mw.Icon = icon
	mw.iconSVG = nil
	if icon != "" {
		if doc, err := meui.LoadSVGByName(icon); err == nil {
			mw.iconSVG = doc
		}
	}
	mw.MarkForRedraw()
	return mw
}

// AddTitleBarButton adds a custom button to the right side of the title bar (before minimize).
func (mw *MdiWindow) AddTitleBarButton(btn TitleBarButton) {
	mw.customButtons = append(mw.customButtons, btn)
	mw.MarkForRedraw()
}

// RemoveTitleBarButtons removes all custom title bar buttons.
func (mw *MdiWindow) RemoveTitleBarButtons() {
	mw.customButtons = nil
	mw.MarkForRedraw()
}

func (mw *MdiWindow) SetContent(content meui.Widget) *MdiWindow {
	mw.Content = content
	if content != nil {
		content.SetParent(mw)
	}
	mw.SetNeedsLayout()
	mw.MarkForRedraw()
	return mw
}

func (mw *MdiWindow) SetOnClose(handler func()) *MdiWindow {
	mw.onClose = handler
	return mw
}

// ── Window state management ────────────────────────────────────

func (mw *MdiWindow) Minimize() {
	if !mw.Minimizable {
		return
	}
	mw.Minimized = true
	mw.Minimized = true
	mw.SetVisible(false)
}

func (mw *MdiWindow) Maximize() {
	if !mw.Maximizable {
		return
	}
	if mw.Maximized {
		mw.Restore()
		return
	}
	mw.PrevRect = mw.FrameRect()
	mw.Maximized = true
	// Fill the center area (space not occupied by docks)
	if p := mw.Parent(); p != nil {
		if mc, ok := p.(*MdiContainer); ok {
			cr := mc.CenterRect
			mw.SetFrameRect(graphics.NewRect(cr.X, cr.Y, cr.Width, cr.Height))
		} else {
			pf := p.FrameRect()
			mw.SetFrameRect(graphics.NewRect(0, 0, pf.Width, pf.Height))
		}
	}
}

func (mw *MdiWindow) Restore() {
	if !mw.Maximized {
		return
	}
	mw.Maximized = false
	if mw.PrevRect.Width > 0 && mw.PrevRect.Height > 0 {
		mw.SetFrameRect(mw.PrevRect)
		mw.PrevRect = graphics.Rect{}
	}
}

func (mw *MdiWindow) Close() {
	mw.SetVisible(false)
	if mw.onClose != nil {
		mw.onClose()
	}
}

func (mw *MdiWindow) AddChild(child meui.Widget) {
	mw.BaseContainer.AddChild(child)
	if mw.Content == nil {
		mw.Content = child
	}
}

// SetFrameRect triggers layout so content is positioned.
func (mw *MdiWindow) SetFrameRect(rect graphics.Rect) {
	mw.BaseWidget.SetFrameRect(rect)
	mw.SetNeedsLayout()
}

func (mw *MdiWindow) ValidateLayout() {
	if !mw.NeedsLayout() {
		return
	}
	frame := mw.FrameRect()
	if mw.Content != nil {
		tbH := mw.titleBarHeight()
		contentRect := graphics.NewRect(0, tbH, frame.Width, frame.Height-tbH)
		mw.Content.SetFrameRect(contentRect)
		if vc, ok := mw.Content.(interface{ ValidateLayout() }); ok {
			vc.ValidateLayout()
		}
	}
	mw.ClearNeedsLayout()
}

func (mw *MdiWindow) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !mw.Visible() {
		return
	}
	canvas.Save()
	disabled := !mw.Enabled()
	tbH := mw.titleBarHeight()
	isDocked := mw.DockPos != DockCenter

	if mw.NeedsLayout() {
		mw.ValidateLayout()
	}

	mt := mw.MdiTheme
	if mt == nil {
		mt = theme.DefaultMdiTheme
	}

	// Shadow (only when floating)
	if !isDocked {
		shadow := graphics.NewPaint()
		shadow.SetColor(mt.WinShadowInk)
		canvas.DrawRect(graphics.NewRect(rect.X+3, rect.Y+3, rect.Width, rect.Height), shadow)
	}

	// Border
	border := graphics.NewPaint()
	border.SetStyle(paintstyle.Stroke)
	border.SetStrokeWidth(1)
	border.SetColor(mt.WinEdgeInk)
	canvas.DrawRoundedRect(rect, theme.DefaultMdiTheme.CornerRadius, theme.DefaultMdiTheme.CornerRadius, border)

	// Title bar background — docked windows get a distinctly lighter color
	titleRect := graphics.NewRect(rect.X, rect.Y, rect.Width, tbH)
	tbg := graphics.NewPaint()
	if isDocked {
		tbg.SetColor(mt.WinTitleBgInk)
	} else {
		tbg.SetColor(mt.WinTitleInactive)
	}
	canvas.DrawRect(titleRect, tbg)

	// Title bar bottom line
	tline := graphics.NewPaint()
	tline.SetStyle(paintstyle.Stroke)
	tline.SetStrokeWidth(1)
	tline.SetColor(mt.WinTitleEdgeInk)
	canvas.DrawLine(rect.X, rect.Y+tbH, rect.X+rect.Width, rect.Y+tbH, tline)

	// Title text
	font := theme.DefaultTheme.BaseFont
	if font == nil {
		font = graphics.SystemFont
	}
	if font != nil {
		iconSize := float32(16)
		if isDocked {
			iconSize = 14
		}
		labelX := rect.X + 8
		if mw.iconSVG != nil {
			iconY := rect.Y + (tbH-iconSize)/2
			meui.DrawSVG(canvas, mw.iconSVG, rect.X+6, iconY, iconSize, iconSize)
			labelX += iconSize + 4
		} else if mw.Icon != "" {
			// Icon name set but SVG not found — draw fallback colored dot
			iconY := rect.Y + (tbH-iconSize)/2
			fb := graphics.NewPaint()
			fb.SetColor(mt.WinAccentInk)
			canvas.DrawCircle(rect.X+6+iconSize/2, iconY+iconSize/2, iconSize/2-1, fb)
			labelX += iconSize + 4
		}

		tp := graphics.NewPaint()
		tp.SetColor(mt.WinTitleTextInk)
		ascent := font.Baseline()
		titleMaxW := rect.X + rect.Width - labelX - 8
		if !isDocked && mw.Closable {
			titleMaxW -= titleBtnSize + titleBtnSpace
		}
		if !isDocked && mw.Maximizable {
			titleMaxW -= titleBtnSize + titleBtnSpace
		}
		if !isDocked && mw.Minimizable {
			titleMaxW -= titleBtnSize + titleBtnSpace
		}
		// Reserve space for custom buttons
		titleMaxW -= float32(len(mw.customButtons)) * (titleBtnSize + titleBtnSpace)
		title := mw.Title
		if titleMaxW > 0 && font.SimpleWidth(title) > titleMaxW {
			for font.SimpleWidth(title+"…") > titleMaxW && len(title) > 1 {
				title = title[:len(title)-1]
			}
			title += "…"
		}
		ver := theme.ThemeVersion()
		if mw.textCache == nil || mw.textCacheVer != ver {
			mw.textCache = graphics.NewText(title, &graphics.TextDecoration{Font: font, OnBackgroundInk: tp.Color()})
			mw.textCacheVer = ver
		}
		mw.textCache.Draw(canvas, labelX, rect.Y+(tbH-ascent)/2+ascent)
	}

	// Title bar buttons (right-aligned) — only when floating
	btnX := rect.X + rect.Width - titleBtnSize - 6
	btnY := rect.Y + (tbH-titleBtnSize)/2

	// Custom buttons (rightmost before system buttons)
	for i := len(mw.customButtons) - 1; i >= 0; i-- {
		drawTitleButton(canvas, btnX, btnY, titleBtnSize, "•")
		if mw.customButtons[i].Icon != "" {
			if doc, err := meui.LoadSVGByName(mw.customButtons[i].Icon); err == nil {
				meui.DrawSVG(canvas, doc, btnX+2, btnY+2, titleBtnSize-4, titleBtnSize-4)
			}
		}
		btnX -= titleBtnSize + titleBtnSpace
	}

	if !isDocked && mw.Closable {
		drawTitleButton(canvas, btnX, btnY, titleBtnSize, "×")
		btnX -= titleBtnSize + titleBtnSpace
	}
	if !isDocked && mw.Maximizable {
		label := "□"
		if mw.Maximized {
			label = "❐"
		}
		drawTitleButton(canvas, btnX, btnY, titleBtnSize, label)
		btnX -= titleBtnSize + titleBtnSpace
	}
	if !isDocked && mw.Minimizable {
		drawTitleButton(canvas, btnX, btnY, titleBtnSize, "−")
	}

	// Content background and paint
	contentRect := graphics.NewRect(rect.X, rect.Y+tbH, rect.Width, rect.Height-tbH)
	cbg := graphics.NewPaint()
	cbg.SetColor(mt.WinContentBgInk)
	canvas.DrawRect(contentRect, cbg)

	// Paint content
	if mw.Content != nil {
		cf := mw.Content.FrameRect()
		canvas.Save()
		canvas.Translate(rect.X+cf.X, rect.Y+cf.Y)
		mw.Content.Paint(canvas, graphics.NewRect(0, 0, cf.Width, cf.Height))
		canvas.Restore()
	}

	canvas.Restore()
	_ = disabled
	_ = isDocked
}

func (mw *MdiWindow) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "title":
		if s, ok := value.(string); ok {
			mw.SetTitle(s)
		}
	case "closable":
		if b, ok := value.(bool); ok {
			mw.Closable = b
		}
	case "icon":
		if s, ok := value.(string); ok {
			mw.SetIcon(s)
		}
	}
}

// resizeEdgeAt returns which edge(s) the point (x,y) is near, in local coords.
func (mw *MdiWindow) resizeEdgeAt(x, y float32) int {
	if mw.DockPos != DockCenter {
		return resizeNone
	}
	w := mw.Width()
	h := mw.Height()
	edge := resizeNone
	if x <= resizeH {
		edge |= resizeLeft
	}
	if x >= w-resizeH {
		edge |= resizeRight
	}
	if y <= resizeH {
		edge |= resizeTop
	}
	if y >= h-resizeH {
		edge |= resizeBottom
	}
	// Don't resize from title bar area (top edge only if NOT in title bar)
	if edge&resizeTop != 0 && y <= mw.titleBarHeight() && edge == resizeTop {
		return resizeNone
	}
	return edge
}

func drawTitleButton(canvas *graphics.Canvas, x, y, size float32, label string) {
	cp := graphics.NewPaint()
	cp.SetColor(theme.DefaultMdiTheme.WinTextSecondary)
	font := theme.DefaultTheme.BaseFont
	if font == nil {
		font = graphics.SystemFont
	}
	if font != nil {
		ascent := font.Baseline()
		t := graphics.NewText(label, &graphics.TextDecoration{Font: font, OnBackgroundInk: cp.Color()})
		t.Draw(canvas, x+(size-font.SimpleWidth(label))/2, y+(size-ascent)/2+ascent)
	}
}

// CursorAt returns the appropriate resize cursor based on the edge under (x,y).
func (mw *MdiWindow) CursorAt(x, y float32) *platform.Cursor {
	edge := mw.resizeEdgeAt(x, y)
	if edge == resizeNone {
		// The title bar is the drag handle for the whole window.
		if y >= 0 && y <= mw.titleBarHeight() {
			return platform.MoveCursor()
		}
		return nil
	}
	switch edge {
	case resizeLeft, resizeRight:
		return platform.ResizeHorizontalCursor()
	case resizeTop, resizeBottom:
		return platform.ResizeVerticalCursor()
	case resizeTopLeft, resizeBottomRight:
		return platform.ResizeLeftDiagonalCursor()
	case resizeTopRight, resizeBottomLeft:
		return platform.ResizeRightDiagonalCursor()
	}
	return nil
}

// PanelAt returns the MdiWindow itself for title bar clicks (for drag support)
// or resize edge clicks, otherwise delegates to the default child hit-testing.
func (mw *MdiWindow) PanelAt(x, y float32) meui.Widget {
	if y >= 0 && y <= mw.titleBarHeight() {
		return mw
	}
	if mw.resizeEdgeAt(x, y) != resizeNone {
		return mw
	}
	return mw.DefaultPanelAt(x, y)
}

func (mw *MdiWindow) OnMouseDown(x, y float32, button int) bool {
	if !mw.Enabled() || mw.Minimized {
		return false
	}
	// Bring to front
	if mc := mw.container; mc != nil {
		mc.BringToFront(mw)
	}
	isDocked := mw.DockPos != DockCenter
	tbH := mw.titleBarHeight()
	// Button hit areas
	btnY := float32(tbH-titleBtnSize) / 2
	btnX := mw.Width() - titleBtnSize - 6
	if !isDocked && mw.Closable {
		if y >= btnY && y <= btnY+titleBtnSize && x >= btnX && x <= btnX+titleBtnSize {
			mw.Close()
			return true
		}
		btnX -= titleBtnSize + titleBtnSpace
	}
	if !isDocked && mw.Maximizable {
		if y >= btnY && y <= btnY+titleBtnSize && x >= btnX && x <= btnX+titleBtnSize {
			mw.Maximize()
			return true
		}
		btnX -= titleBtnSize + titleBtnSpace
	}
	if !isDocked && mw.Minimizable {
		if y >= btnY && y <= btnY+titleBtnSize && x >= btnX && x <= btnX+titleBtnSize {
			mw.Minimize()
			return true
		}
	}
	// Resize edge (disabled when docked)
	if edge := mw.resizeEdgeAt(x, y); edge != resizeNone {
		mw.resizing = true
		mw.resizeEdge = edge
		mw.dragStartX = x
		mw.dragStartY = y
		mw.startFrame = mw.FrameRect()
		root := mw.PointToRoot(x, y)
		mw.startRootX = root.X
		mw.startRootY = root.Y
		return true
	}
	// Title bar drag — undock first if docked, then start drag
	if y >= 0 && y <= tbH {
		mc := mw.container
		if mc != nil && isDocked {
			// UNDOCK DISABLED for debugging — remove to re-enable
			// mc.removeFromDock(mw)
			mw.DockPos = DockCenter
			if mw.dockRestore.Width > 0 && mw.dockRestore.Height > 0 {
				mw.SetFrameRect(mw.dockRestore)
			}
			mc.SetNeedsLayout()
			mc.MarkForRedraw()
		}
		if mw.Maximized {
			mw.Restore()
		}
		// Only start drag if floating (prevents dragging while docked when undock is disabled)
		if mw.DockPos == DockCenter {
			mw.dragging = true
			mw.dragStartX = x
			mw.dragStartY = y
			mw.startFrame = mw.FrameRect()
			root := mw.PointToRoot(x, y)
			mw.startRootX = root.X
			mw.startRootY = root.Y

			mw.releaseCallback = nil
			if mc != nil {
				mw.releaseCallback = func() { mc.DetachChild(mw) }
			}
		}
		return true
	}
	return false
}

func (mw *MdiWindow) OnMouseMove(x, y float32) {
	if mw.dragging {
		root := mw.PointToRoot(x, y)
		dx := root.X - mw.startRootX
		dy := root.Y - mw.startRootY
		newFrame := graphics.NewRect(
			mw.startFrame.X+dx, mw.startFrame.Y+dy,
			mw.startFrame.Width, mw.startFrame.Height,
		)
		mw.SetFrameRect(newFrame)

		// Check drop zones on parent MdiContainer
		if mc := mw.container; mc != nil {
			cx, cy := newFrame.X, newFrame.Y
			cx += newFrame.Width / 2
			cy += newFrame.Height / 2

			mc.DropHighlight = mc.DropHitTest(cx, cy)
			mc.MarkForRedraw()
		}
		if p := mw.Parent(); p != nil {
			p.MarkForRedraw()
		}
		return
	}
	if mw.resizing {
		root := mw.PointToRoot(x, y)
		dx := root.X - mw.startRootX
		dy := root.Y - mw.startRootY
		frame := mw.startFrame
		newX, newY := frame.X, frame.Y
		newW, newH := frame.Width, frame.Height

		if mw.resizeEdge&resizeLeft != 0 {
			newX = frame.X + dx
			newW = frame.Width - dx
		} else if mw.resizeEdge&resizeRight != 0 {
			newW = frame.Width + dx
		}
		if mw.resizeEdge&resizeTop != 0 {
			newY = frame.Y + dy
			newH = frame.Height - dy
		} else if mw.resizeEdge&resizeBottom != 0 {
			newH = frame.Height + dy
		}

		if newW < mdiMinW {
			if mw.resizeEdge&resizeLeft != 0 {
				newX = frame.X + frame.Width - mdiMinW
			}
			newW = mdiMinW
		}
		if newH < mdiMinH {
			if mw.resizeEdge&resizeTop != 0 {
				newY = frame.Y + frame.Height - mdiMinH
			}
			newH = mdiMinH
		}

		mw.SetFrameRect(graphics.NewRect(newX, newY, newW, newH))
		if p := mw.Parent(); p != nil {
			p.MarkForRedraw()
		}
	}
}

func (mw *MdiWindow) OnMouseUp(x, y float32, button int) bool {
	wasDragging := mw.dragging
	mw.dragging = false
	mw.resizing = false

	if wasDragging {
		if mc := mw.container; mc != nil {
			info := mc.DropHighlight
			mc.DropHighlight = nil
			mc.MarkForRedraw()

			if info != nil {
				mc.AcceptDrop(mw, info)
			} else {
				// No drop target → stays floating, release callback not called
				mw.releaseCallback = nil
			}
		}
	}
	return true
}
