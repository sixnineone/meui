package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

type submenuLevel struct {
	items      []MenuItem
	hoveredIdx int
	openIdx    int           // which item has open child submenu (-1 = none)
	child      *submenuLevel // child submenu level, nil = none
	width      float32
	height     float32
}

type ContextMenu struct {
	meui.BaseWidget
	*theme.ContextMenuTheme
	items          []MenuItem
	hoveredIdx     int
	parentMenu     *ContextMenu
	subMenu        *ContextMenu
	openSubIdx     int
	rootSub        *submenuLevel
	OnItemSelected func(int, *MenuItem)
	onDismiss      func()
}

type MenuItem struct {
	Label        string
	Shortcut     string
	Icon         string  // SVG icon name or emoji
	Height       float32 // custom item height (0 = auto)
	Action       func()
	IsSep        bool
	IsCheck      bool
	Checked      bool
	Enabled      bool
	SubMenu      []MenuItem
	CustomPanel  *meui.Panel // custom widget (replaces label/shortcut rendering)
	labelText    *graphics.Text
	labelTextVer uint64
}

func NewContextMenu() *ContextMenu {
	return &ContextMenu{
		BaseWidget:       *meui.NewBaseWidget(),
		ContextMenuTheme: theme.DefaultContextMenuTheme,
		hoveredIdx:       -1,
		openSubIdx:       -1,
	}
}

func (m *ContextMenu) updateMenuItemText(item *MenuItem) {
	if item.Label == "" {
		return
	}
	font := m.ContextMenuTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	ink := theme.DefaultContextMenuTheme.TextInk
	item.labelText = graphics.NewText(item.Label, &graphics.TextDecoration{Font: font, OnBackgroundInk: ink})
	for i := range item.SubMenu {
		child := &item.SubMenu[i]
		child.labelText = nil
		if child.Label != "" {
			child.labelText = graphics.NewText(child.Label, &graphics.TextDecoration{Font: font, OnBackgroundInk: ink})
		}
		if len(child.SubMenu) > 0 {
			m.buildSubMenuTextCache(child)
		}
	}
}

func (m *ContextMenu) buildSubMenuTextCache(item *MenuItem) {
	font := m.ContextMenuTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	ink := theme.DefaultContextMenuTheme.TextInk
	for i := range item.SubMenu {
		child := &item.SubMenu[i]
		child.labelText = nil
		if child.Label != "" {
			child.labelText = graphics.NewText(child.Label, &graphics.TextDecoration{Font: font, OnBackgroundInk: ink})
		}
		if len(child.SubMenu) > 0 {
			m.buildSubMenuTextCache(child)
		}
	}
}

func (m *ContextMenu) deepestLevel() *submenuLevel {
	if m.rootSub == nil {
		return nil
	}
	cur := m.rootSub
	for cur.child != nil {
		cur = cur.child
	}
	return cur
}

func (m *ContextMenu) levelAtDepth(d int) *submenuLevel {
	if d < 0 || m.rootSub == nil {
		return nil
	}
	cur := m.rootSub
	for i := 0; i < d && cur != nil; i++ {
		cur = cur.child
	}
	return cur
}

func (m *ContextMenu) levelCount() int {
	n := 0
	for cur := m.rootSub; cur != nil; cur = cur.child {
		n++
	}
	return n
}

func (m *ContextMenu) levelPos(l *submenuLevel) (float32, float32) {
	if l == nil || m.openSubIdx < 0 || m.rootSub == nil {
		return 0, 0
	}
	type hop struct {
		lvl *submenuLevel
		idx int
	}
	var chain []hop
	cur := m.rootSub
	for cur != nil && cur != l {
		chain = append(chain, hop{cur, cur.openIdx})
		cur = cur.child
	}
	if cur == nil {
		return 0, 0
	}
	lx := m.Width()
	ly := m.itemY(m.openSubIdx)
	for _, h := range chain {
		lx += h.lvl.width
		ly += m.levelItemY(h.lvl, h.idx)
	}
	return lx, ly
}

func (m *ContextMenu) openChild(parent *submenuLevel, itemIdx int) {
	if parent == nil || itemIdx < 0 || itemIdx >= len(parent.items) || len(parent.items[itemIdx].SubMenu) == 0 {
		return
	}
	parent.openIdx = itemIdx
	parent.child = &submenuLevel{
		items:      parent.items[itemIdx].SubMenu,
		hoveredIdx: -1,
		openIdx:    -1,
	}
	m.computeLevelSize(parent.child)
}

func (m *ContextMenu) closeDeepest() {
	if m.rootSub == nil {
		return
	}
	if m.rootSub.child == nil {
		m.closeAll()
		return
	}
	cur := m.rootSub
	for cur.child != nil && cur.child.child != nil {
		cur = cur.child
	}
	cur.openIdx = -1
	cur.child = nil
}

func (m *ContextMenu) closeAll() {
	m.openSubIdx = -1
	m.rootSub = nil
}

func (m *ContextMenu) computeLevelSize(l *submenuLevel) {
	if l == nil || len(l.items) == 0 {
		if l != nil {
			l.width = 0
			l.height = 0
		}
		return
	}
	font := m.ContextMenuTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	itemHeight := float32(28)
	totalH := float32(0)
	maxW := float32(160)
	for _, item := range l.items {
		if item.IsSep {
			totalH += itemHeight / 2
			continue
		}
		w := font.SimpleWidth(item.Label)
		if item.Shortcut != "" {
			w += font.SimpleWidth(item.Shortcut) + 30
		}
		if item.Icon != "" {
			w += 18
		}
		if len(item.SubMenu) > 0 {
			w += 16
		}
		w += 45 + 15
		if w > maxW {
			maxW = w
		}
		totalH += itemHeight
	}
	l.width = maxW
	l.height = totalH
}

func (m *ContextMenu) levelItemY(l *submenuLevel, idx int) float32 {
	itemHeight := float32(28)
	y := float32(0)
	for i := 0; i < idx && i < len(l.items); i++ {
		if l.items[i].IsSep {
			y += itemHeight / 2
		} else {
			y += itemHeight
		}
	}
	return y
}

func (m *ContextMenu) ContainsPoint(x, y float32) bool {
	if m.rootSub == nil || m.openSubIdx < 0 {
		return m.BaseWidget.ContainsPoint(x, y)
	}
	if m.BaseWidget.ContainsPoint(x, y) {
		return true
	}
	for cur := m.rootSub; cur != nil; cur = cur.child {
		lx, ly := m.levelPos(cur)
		if x >= lx && x <= lx+cur.width && y >= ly && y <= ly+cur.height {
			return true
		}
	}
	return false
}

func (m *ContextMenu) itemY(idx int) float32 {
	itemHeight := float32(28)
	padding := float32(4)
	y := padding
	for i := 0; i < idx && i < len(m.items); i++ {
		if m.items[i].IsSep {
			y += itemHeight / 2
		} else {
			y += itemHeight
		}
	}
	return y
}

func (m *ContextMenu) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if !m.Enabled() {
		return false
	}
	switch keyCode {
	case platform.KeyDown, platform.KeyRight:
		deepest := m.deepestLevel()
		if deepest != nil && deepest.hoveredIdx >= 0 {
			if keyCode == platform.KeyRight && deepest.hoveredIdx < len(deepest.items) &&
				len(deepest.items[deepest.hoveredIdx].SubMenu) > 0 {
				m.openChild(deepest, deepest.hoveredIdx)
				m.MarkForRedraw()
				return true
			}
			next := deepest.hoveredIdx + 1
			for next < len(deepest.items) && (deepest.items[next].IsSep || !deepest.items[next].Enabled) {
				next++
			}
			if next < len(deepest.items) {
				deepest.hoveredIdx = next
				m.MarkForRedraw()
			}
			return true
		}
		if m.rootSub != nil && m.rootSub.hoveredIdx < 0 {
			next := 0
			for next < len(m.rootSub.items) && (m.rootSub.items[next].IsSep || !m.rootSub.items[next].Enabled) {
				next++
			}
			if next < len(m.rootSub.items) {
				m.rootSub.hoveredIdx = next
				m.MarkForRedraw()
			}
			return true
		}
		if keyCode == platform.KeyRight && m.hoveredIdx >= 0 && m.hoveredIdx < len(m.items) && len(m.items[m.hoveredIdx].SubMenu) > 0 {
			if m.openSubIdx != m.hoveredIdx {
				m.openSubIdx = m.hoveredIdx
				m.rootSub = &submenuLevel{
					items:      m.items[m.hoveredIdx].SubMenu,
					hoveredIdx: -1,
					openIdx:    -1,
				}
				m.computeLevelSize(m.rootSub)
				m.MarkForRedraw()
			}
			return true
		}
		next := m.hoveredIdx + 1
		for next < len(m.items) && (m.items[next].IsSep || !m.items[next].Enabled) {
			next++
		}
		if next < len(m.items) {
			m.hoveredIdx = next
			if len(m.items[next].SubMenu) > 0 {
				m.openSubIdx = next
				m.rootSub = &submenuLevel{
					items:      m.items[next].SubMenu,
					hoveredIdx: -1,
					openIdx:    -1,
				}
				m.computeLevelSize(m.rootSub)
			} else {
				m.closeAll()
			}
			m.MarkForRedraw()
		}
		return true

	case platform.KeyUp, platform.KeyLeft:
		deepest := m.deepestLevel()
		if keyCode == platform.KeyLeft && deepest != nil {
			m.closeDeepest()
			m.MarkForRedraw()
			return true
		}
		if deepest != nil && deepest.hoveredIdx > 0 {
			prev := deepest.hoveredIdx - 1
			for prev >= 0 && (deepest.items[prev].IsSep || !deepest.items[prev].Enabled) {
				prev--
			}
			if prev >= 0 {
				deepest.hoveredIdx = prev
				m.MarkForRedraw()
			}
			return true
		}
		if m.rootSub != nil && m.rootSub.hoveredIdx > 0 {
			prev := m.rootSub.hoveredIdx - 1
			for prev >= 0 && (m.rootSub.items[prev].IsSep || !m.rootSub.items[prev].Enabled) {
				prev--
			}
			if prev >= 0 {
				m.rootSub.hoveredIdx = prev
				m.MarkForRedraw()
			}
			return true
		}
		prev := m.hoveredIdx - 1
		for prev >= 0 && (m.items[prev].IsSep || !m.items[prev].Enabled) {
			prev--
		}
		if prev >= 0 {
			m.hoveredIdx = prev
			if len(m.items[prev].SubMenu) > 0 {
				m.openSubIdx = prev
				m.rootSub = &submenuLevel{
					items:      m.items[prev].SubMenu,
					hoveredIdx: -1,
					openIdx:    -1,
				}
				m.computeLevelSize(m.rootSub)
			} else {
				m.closeAll()
			}
			m.MarkForRedraw()
		}
		return true

	case platform.KeyReturn:
		deepest := m.deepestLevel()
		if deepest != nil && deepest.hoveredIdx >= 0 && deepest.hoveredIdx < len(deepest.items) {
			sub := deepest.items[deepest.hoveredIdx]
			if !sub.IsSep && sub.Enabled {
				if sub.Action != nil {
					sub.Action()
				}
				if m.OnItemSelected != nil {
					m.OnItemSelected(-1, &sub)
				}
				m.closeAll()
				m.MarkForRedraw()
			}
			return true
		}
		if m.hoveredIdx >= 0 && m.hoveredIdx < len(m.items) {
			item := m.items[m.hoveredIdx]
			if !item.IsSep && len(item.SubMenu) == 0 && item.Enabled {
				if item.Action != nil {
					item.Action()
				}
				if m.OnItemSelected != nil {
					m.OnItemSelected(m.hoveredIdx, &item)
				}
				m.closeAll()
				m.MarkForRedraw()
			} else if len(item.SubMenu) > 0 {
				if m.openSubIdx != m.hoveredIdx {
					m.openSubIdx = m.hoveredIdx
					m.rootSub = &submenuLevel{
						items:      item.SubMenu,
						hoveredIdx: -1,
						openIdx:    -1,
					}
					m.computeLevelSize(m.rootSub)
					m.MarkForRedraw()
				}
			}
			return true
		}
		return true

	case platform.KeyEscape:
		if m.rootSub != nil {
			m.closeDeepest()
			m.MarkForRedraw()
			return true
		}
		if m.openSubIdx >= 0 {
			m.closeAll()
			m.MarkForRedraw()
			return true
		}
		return false
	}
	return false
}

// ContextMenuOverlay is a reusable helper for showing a popup context menu
// within a widget's own canvas. Embed this in any widget that needs a
// right-click context menu, then call Show/Draw/HandleClick/HitTest.
//
// Usage:
//
//	type MyWidget struct {
//	    meui.BaseWidget
//	    menu widgets.ContextMenuOverlay
//	    ...
//	}
//
//	// Right-click → show:
//	m.menu.Show(x, y, items)
//
//	// In Paint:
//	m.menu.Draw(canvas, m.Width(), m.Height(), font)
//
//	// In OnMouseDown before other handling:
//	if m.menu.HandleClick(x, y, button) { return true }
//
//	// In OnMouseMove before other handling:
//	m.menu.HandleHover(x, y, m.Width(), m.Height(), font)
//	if m.menu.Visible { m.MarkForRedraw(); return }
type ContextMenuOverlay struct {
	Items        []MenuItem
	X, Y         float32
	Hover        int
	Visible      bool
	NoAutoHide   bool // if true, HandleClick won't auto-hide after firing an action
	menuW, menuH float32
}

// Show opens the context menu at (x,y) with the given items.
func (m *ContextMenuOverlay) Show(x, y float32, items []MenuItem) {
	m.Items = items
	m.X = x
	m.Y = y
	m.Hover = -1
	m.Visible = true
	m.NoAutoHide = false
	m.menuW = 0
	m.menuH = 0
}

// Hide closes the context menu.
func (m *ContextMenuOverlay) Hide() {
	m.Visible = false
	m.Items = nil
}

// Draw renders the context menu on the canvas.
// widgetW, widgetH are the dimensions of the parent widget for bounds clamping.
func (m *ContextMenuOverlay) Draw(canvas *graphics.Canvas, widgetW, widgetH float32, font graphics.Font) {
	if !m.Visible || len(m.Items) == 0 {
		return
	}
	ascent := font.Baseline()
	lh := font.LineHeight() + 6
	itemH := lh + 2
	totalH := float32(0)
	for _, item := range m.Items {
		if item.IsSep {
			totalH += 6
			continue
		}
		h := item.Height
		if h <= 0 {
			h = itemH
		}
		totalH += h
	}
	maxW := float32(180)
	for _, item := range m.Items {
		if item.CustomPanel != nil {
			fr := item.CustomPanel.FrameRect()
			w := fr.Width + 20
			if w > maxW {
				maxW = w
			}
			continue
		}
		w := font.SimpleWidth(item.Label+item.Shortcut) + 40
		if item.Icon != "" {
			w += 18
		}
		if w > maxW {
			maxW = w
		}
		if len(item.SubMenu) > 0 {
			w += 16
		}
	}
	m.menuW = maxW
	m.menuH = totalH
	menuX := m.X
	menuY := m.Y
	edgeMargin := float32(8)
	// Flip horizontally if overflowing right edge
	if menuX+maxW > widgetW-edgeMargin {
		menuX = m.X - maxW
		if menuX < edgeMargin {
			menuX = edgeMargin
		}
	}
	// Flip vertically if overflowing bottom edge
	if menuY+totalH > widgetH-edgeMargin {
		menuY = m.Y - totalH
		if menuY < edgeMargin {
			menuY = edgeMargin
		}
	}
	// Clamp to edges
	if menuX < edgeMargin {
		menuX = edgeMargin
	}
	if menuY < edgeMargin {
		menuY = edgeMargin
	}
	m.X = menuX
	m.Y = menuY
	bg := graphics.NewPaint()
	bg.SetColor(theme.DefaultContextMenuTheme.BackgroundInk)
	menuR := theme.Shape().Menu
	canvas.DrawRoundedRect(graphics.NewRect(menuX, menuY, maxW, totalH), menuR, menuR, bg)
	bd := graphics.NewPaint()
	bd.SetStyle(paintstyle.Stroke)
	bd.SetColor(theme.DefaultContextMenuTheme.EdgeInk)
	canvas.DrawRoundedRect(graphics.NewRect(menuX, menuY, maxW, totalH), menuR, menuR, bd)
	y := menuY
	for i, item := range m.Items {
		if item.IsSep {
			sp := graphics.NewPaint()
			sp.SetColor(theme.DefaultContextMenuTheme.SeparatorInk)
			canvas.DrawLine(menuX+8, y+3, menuX+maxW-8, y+3, sp)
			y += 6
			continue
		}
		h := item.Height
		if h <= 0 {
			h = itemH
		}
		if item.CustomPanel != nil {
			if i == m.Hover {
				hp := graphics.NewPaint()
				hp.SetColor(theme.DefaultContextMenuTheme.HoverInk)
				canvas.DrawRect(graphics.NewRect(menuX, y, maxW, h), hp)
			}
			item.CustomPanel.SetFrameRect(graphics.NewRect(menuX+4, y+2, maxW-8, h-4))
			item.CustomPanel.Paint(canvas, graphics.NewRect(0, 0, maxW-8, h-4))
			y += h
			continue
		}
		if i == m.Hover {
			hp := graphics.NewPaint()
			hp.SetColor(theme.DefaultContextMenuTheme.HoverInk)
			canvas.DrawRect(graphics.NewRect(menuX, y, maxW, h), hp)
		}
		if item.Icon != "" {
			iconSize := float32(14)
			if doc, err := meui.LoadSVGByName(item.Icon); err == nil {
				meui.DrawSVG(canvas, doc, menuX+8, y+(h-iconSize)/2, iconSize, iconSize)
			} else {
				ip := graphics.NewPaint()
				ip.SetColor(theme.DefaultContextMenuTheme.TextInk)
				it := graphics.NewText(item.Icon, &graphics.TextDecoration{Font: font, OnBackgroundInk: ip.Color()})
				it.Draw(canvas, menuX+8, y+(h-ascent)/2+ascent)
			}
		}
		tp := graphics.NewPaint()
		tp.SetColor(theme.DefaultContextMenuTheme.TextInk)
		textX := menuX + float32(10)
		if item.Icon != "" {
			textX += 18
		}
		ver := theme.ThemeVersion()
		if item.labelText == nil || item.labelTextVer != ver {
			item.labelText = graphics.NewText(item.Label, &graphics.TextDecoration{Font: font, OnBackgroundInk: tp.Color()})
			item.labelTextVer = ver
		}
		if item.labelText != nil {
			item.labelText.Draw(canvas, textX, y+(h-ascent)/2+ascent)
		}
		if item.Shortcut != "" {
			sp := graphics.NewPaint()
			sp.SetColor(theme.DefaultContextMenuTheme.SeparatorInk)
			sw := font.SimpleWidth(item.Shortcut)
			st := graphics.NewText(item.Shortcut, &graphics.TextDecoration{Font: font, OnBackgroundInk: sp.Color()})
			st.Draw(canvas, menuX+maxW-sw-10, y+(h-ascent)/2+ascent)
		}
		if len(item.SubMenu) > 0 {
			ap := graphics.NewPaint()
			ap.SetColor(theme.DefaultContextMenuTheme.ArrowInk)
			at := graphics.NewText("\u203A", &graphics.TextDecoration{Font: font, OnBackgroundInk: ap.Color()})
			at.Draw(canvas, menuX+maxW-18, y+(h-ascent)/2+ascent)
		}
		y += h
	}
}

// HandleClick processes a click and returns true if the menu was hit.
func (m *ContextMenuOverlay) HandleClick(x, y, button int) bool {
	if !m.Visible {
		return false
	}
	if m.menuW == 0 {
		m.measure()
	}
	menuX := m.X
	menuY := m.Y
	edgeMargin := float32(8)
	if menuX+m.menuW > 2000-edgeMargin {
		menuX = m.X - m.menuW
		if menuX < edgeMargin {
			menuX = edgeMargin
		}
	}
	if menuY+m.menuH > 2000-edgeMargin {
		menuY = m.Y - m.menuH
		if menuY < edgeMargin {
			menuY = edgeMargin
		}
	}
	if menuX < edgeMargin {
		menuX = edgeMargin
	}
	if menuY < edgeMargin {
		menuY = edgeMargin
	}
	fx, fy := float32(x), float32(y)
	if fx >= menuX && fx <= menuX+m.menuW && fy >= menuY && fy <= menuY+m.menuH {
		lh := 28.0
		yy := menuY
		for _, item := range m.Items {
			if item.IsSep {
				yy += 6
				continue
			}
			h := item.Height
			if h <= 0 {
				h = float32(lh)
			}
			if fy >= yy && fy < yy+h && item.Action != nil {
				item.Action()
				if !m.NoAutoHide {
					m.Hide()
				}
				return true
			}
			yy += h
		}
		m.Hide()
		return true
	}
	m.Hide()
	return true
}

// HitTest returns the hover index of a point, or -1 if outside the menu.
func (m *ContextMenuOverlay) HitTest(x, y, widgetW, widgetH float32, font graphics.Font) int {
	if !m.Visible {
		return -1
	}
	if m.menuW == 0 {
		m.measure()
	}
	menuX, menuY, _, _, itemH := m.bounds(widgetW, widgetH, font)
	if x < menuX || x > menuX+m.menuW || y < menuY || y > menuY+m.menuH {
		return -1
	}
	yy := menuY
	for i, item := range m.Items {
		if item.IsSep {
			yy += 6
			continue
		}
		h := item.Height
		if h <= 0 {
			h = itemH
		}
		if y >= yy && y < yy+h {
			return i
		}
		yy += h
	}
	return -1
}

func (m *ContextMenuOverlay) measure() {
	m.menuW = 180
}

func (m *ContextMenuOverlay) bounds(widgetW, widgetH float32, font graphics.Font) (menuX, menuY, maxW, totalH, itemH float32) {
	lh := font.LineHeight() + 6
	itemH = lh + 2
	totalH = 0
	for _, item := range m.Items {
		if item.IsSep {
			totalH += 6
			continue
		}
		h := item.Height
		if h <= 0 {
			h = itemH
		}
		totalH += h
	}
	maxW = m.menuW
	menuX = m.X
	menuY = m.Y
	edgeMargin := float32(8)
	if menuX+maxW > widgetW-edgeMargin {
		menuX = m.X - maxW
		if menuX < edgeMargin {
			menuX = edgeMargin
		}
	}
	if menuY+totalH > widgetH-edgeMargin {
		menuY = m.Y - totalH
		if menuY < edgeMargin {
			menuY = edgeMargin
		}
	}
	if menuX < edgeMargin {
		menuX = edgeMargin
	}
	if menuY < edgeMargin {
		menuY = edgeMargin
	}
	return
}

func (m *ContextMenu) AddItem(label string, action func()) *ContextMenu {
	m.items = append(m.items, MenuItem{Label: label, Action: action, Enabled: true})
	m.updateMenuItemText(&m.items[len(m.items)-1])
	return m
}

func (m *ContextMenu) AddSeparator() *ContextMenu {
	m.items = append(m.items, MenuItem{IsSep: true})
	m.syncSize()
	m.MarkForRedraw()
	return m
}

func (m *ContextMenu) AddCheckItem(label string, checked bool, action func()) *ContextMenu {
	m.items = append(m.items, MenuItem{Label: label, IsCheck: true, Checked: checked, Action: action, Enabled: true})
	m.updateMenuItemText(&m.items[len(m.items)-1])
	return m
}

func (m *ContextMenu) AddSubMenu(label string, items []MenuItem) *ContextMenu {
	m.items = append(m.items, MenuItem{Label: label, SubMenu: items, Enabled: true})
	m.updateMenuItemText(&m.items[len(m.items)-1])
	return m
}

func (m *ContextMenu) SetItems(items []MenuItem) *ContextMenu {
	m.items = items
	for i := range m.items {
		m.updateMenuItemText(&m.items[i])
	}
	m.syncSize()
	m.MarkForRedraw()
	return m
}

func (m *ContextMenu) syncSize() {
	w, h := m.PreferredSize()
	if w > 0 {
		m.SetWidthDirect(w)
	}
	if h > 0 {
		m.SetHeightDirect(h)
	}
}

func (m *ContextMenu) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !m.Visible() || len(m.items) == 0 {
		return
	}

	canvas.Save()

	itemHeight := float32(28)
	padding := float32(4)

	menuR := theme.Shape().Menu
	drawMenuShadow(canvas, rect, menuR)
	bgPaint := graphics.NewPaint()
	bgPaint.SetColor(theme.DefaultContextMenuTheme.BackgroundInk)
	canvas.DrawRoundedRect(rect, menuR, menuR, bgPaint)

	borderPaint := graphics.NewPaint()
	borderPaint.SetStyle(paintstyle.Stroke)
	borderPaint.SetStrokeWidth(theme.BorderStrokeWidth)
	borderPaint.SetColor(theme.DefaultContextMenuTheme.EdgeInk)
	canvas.DrawRoundedRect(rect, menuR, menuR, borderPaint)

	font := m.ContextMenuTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}

	y := rect.Y + padding
	for i := range m.items {
		item := &m.items[i]
		if item.IsSep {
			sepPaint := graphics.NewPaint()
			sepPaint.SetStyle(paintstyle.Stroke)
			sepPaint.SetStrokeWidth(theme.SeparatorStrokeWidth)
			sepPaint.SetColor(theme.DefaultContextMenuTheme.EdgeInk)
			canvas.DrawLine(rect.X+4, y+itemHeight/2, rect.X+rect.Width-4, y+itemHeight/2, sepPaint)
			y += itemHeight / 2
			continue
		}

		itemRect := graphics.NewRect(rect.X, y, rect.Width, itemHeight)
		if i == m.hoveredIdx && item.Enabled {
			hoverPaint := graphics.NewPaint()
			hoverPaint.SetColor(theme.DefaultContextMenuTheme.HoverInk)
			canvas.DrawRect(itemRect, hoverPaint)
		}

		if item.CustomPanel != nil {
			item.CustomPanel.SetFrameRect(graphics.NewRect(rect.X+45, y+2, rect.Width-60, itemHeight-4))
			item.CustomPanel.Paint(canvas, graphics.NewRect(0, 0, rect.Width-60, itemHeight-4))
			y += itemHeight
			continue
		}

		if item.IsCheck {
			boxSize := float32(14)
			boxX := rect.X + 5
			boxY := y + (itemHeight-boxSize)/2
			DrawCheckBox(canvas, boxX, boxY, boxSize, item.Checked, item.Enabled)
		}

		if item.Icon != "" {
			iconX := rect.X + 25
			iconSize := float32(14)
			if doc, err := meui.LoadSVGByName(item.Icon); err == nil {
				meui.DrawSVG(canvas, doc, iconX, y+(itemHeight-iconSize)/2, iconSize, iconSize)
			} else {
				ip := graphics.NewPaint()
				ip.SetColor(theme.DefaultContextMenuTheme.TextInk)
				it := graphics.NewText(item.Icon, &graphics.TextDecoration{Font: font, OnBackgroundInk: ip.Color()})
				it.Draw(canvas, iconX, y+(itemHeight-font.LineHeight())/2+font.Baseline())
			}
		}

		textX := rect.X + 45
		textPaint := graphics.NewPaint()
		if item.Enabled {
			textPaint.SetColor(theme.DefaultContextMenuTheme.TextInk)
		} else {
			textPaint.SetColor(m.ContextMenuTheme.TextDisabled)
		}
		textY := y + (itemHeight-font.LineHeight())/2 + font.Baseline()
		ver := theme.ThemeVersion()
		if item.labelText == nil || item.labelTextVer != ver {
			item.labelText = graphics.NewText(item.Label, &graphics.TextDecoration{Font: font, OnBackgroundInk: textPaint.Color()})
			item.labelTextVer = ver
		}
		if item.labelText != nil {
			item.labelText.Draw(canvas, textX, textY)
		}

		if item.Shortcut != "" {
			shortcutPaint := graphics.NewPaint()
			if item.Enabled {
				shortcutPaint.SetColor(theme.DefaultTheme.TextSecondary)
			} else {
				shortcutPaint.SetColor(m.ContextMenuTheme.TextDisabled)
			}
			sw := font.SimpleWidth(item.Shortcut)
			st := graphics.NewText(item.Shortcut, &graphics.TextDecoration{Font: font, OnBackgroundInk: shortcutPaint.Color()})
			st.Draw(canvas, rect.X+rect.Width-sw-23, textY)
		}

		if len(item.SubMenu) > 0 && item.Enabled {
			if doc, err := meui.LoadSVGByName("chevron_right"); err == nil {
				meui.DrawSVG(canvas, doc, rect.X+rect.Width-32, y+(itemHeight-12)/2, 12, 12)
			} else {
				arrowPaint := graphics.NewPaint()
				arrowPaint.SetColor(theme.DefaultContextMenuTheme.TextInk)
				at := graphics.NewText("\u25B6", &graphics.TextDecoration{Font: font, OnBackgroundInk: arrowPaint.Color()})
				at.Draw(canvas, rect.X+rect.Width-16, y+itemHeight/2)
			}
		}

		y += itemHeight
	}

	if m.openSubIdx >= 0 && m.rootSub != nil {
		m.drawSubLevel(canvas, rect, m.rootSub, 0, m.itemY(m.openSubIdx), rect.Width)
	}

	canvas.Restore()
}

func (m *ContextMenu) drawSubLevel(canvas *graphics.Canvas, rect graphics.Rect, l *submenuLevel, parentLocalX, parentItemLocalY, parentW float32) {
	if l == nil || len(l.items) == 0 {
		return
	}

	itemHeight := float32(28)

	m.computeLevelSize(l)

	font := m.ContextMenuTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}

	subX := rect.X + parentLocalX + parentW
	subY := rect.Y + parentItemLocalY

	subBg := graphics.NewPaint()
	subBg.SetColor(theme.DefaultContextMenuTheme.BackgroundInk)
	canvas.DrawRect(graphics.NewRect(subX, subY, l.width, l.height), subBg)

	subBd := graphics.NewPaint()
	subBd.SetStyle(paintstyle.Stroke)
	subBd.SetStrokeWidth(theme.BorderStrokeWidth)
	subBd.SetColor(theme.DefaultContextMenuTheme.EdgeInk)
	canvas.DrawRect(graphics.NewRect(subX, subY, l.width, l.height), subBd)

	sy := subY
	for i := range l.items {
		item := &l.items[i]
		if item.IsSep {
			sp := graphics.NewPaint()
			sp.SetColor(theme.DefaultContextMenuTheme.EdgeInk)
			sp.SetStyle(paintstyle.Stroke)
			sp.SetStrokeWidth(theme.SeparatorStrokeWidth)
			canvas.DrawLine(subX+8, sy+itemHeight/2, subX+l.width-8, sy+itemHeight/2, sp)
			sy += itemHeight / 2
			continue
		}

		if i == l.hoveredIdx && item.Enabled {
			hp := graphics.NewPaint()
			hp.SetColor(theme.DefaultContextMenuTheme.HoverInk)
			canvas.DrawRect(graphics.NewRect(subX, sy, l.width, itemHeight), hp)
		}

		if item.CustomPanel != nil {
			item.CustomPanel.SetFrameRect(graphics.NewRect(subX+45, sy+2, l.width-60, itemHeight-4))
			item.CustomPanel.Paint(canvas, graphics.NewRect(0, 0, l.width-60, itemHeight-4))
			sy += itemHeight
			continue
		}

		if item.IsCheck {
			boxSize := float32(14)
			boxX := subX + 5
			boxY := sy + (itemHeight-boxSize)/2
			DrawCheckBox(canvas, boxX, boxY, boxSize, item.Checked, item.Enabled)
		}

		if item.Icon != "" {
			iconX := subX + 25
			iconSize := float32(14)
			if doc, err := meui.LoadSVGByName(item.Icon); err == nil {
				meui.DrawSVG(canvas, doc, iconX, sy+(itemHeight-iconSize)/2, iconSize, iconSize)
			} else {
				ip := graphics.NewPaint()
				if item.Enabled {
					ip.SetColor(theme.DefaultContextMenuTheme.TextInk)
				} else {
					ip.SetColor(m.ContextMenuTheme.TextDisabled)
				}
				it := graphics.NewText(item.Icon, &graphics.TextDecoration{Font: font, OnBackgroundInk: ip.Color()})
				it.Draw(canvas, iconX, sy+(itemHeight-font.LineHeight())/2+font.Baseline())
			}
		}

		tp := graphics.NewPaint()
		if item.Enabled {
			tp.SetColor(theme.DefaultContextMenuTheme.TextInk)
		} else {
			tp.SetColor(m.ContextMenuTheme.TextDisabled)
		}
		ver := theme.ThemeVersion()
		if item.labelText == nil || item.labelTextVer != ver {
			item.labelText = graphics.NewText(item.Label, &graphics.TextDecoration{Font: font, OnBackgroundInk: tp.Color()})
			item.labelTextVer = ver
		}
		if item.labelText != nil {
			item.labelText.Draw(canvas, subX+45, sy+(itemHeight-font.LineHeight())/2+font.Baseline())
		}

		if item.Shortcut != "" {
			sp := graphics.NewPaint()
			if item.Enabled {
				sp.SetColor(theme.DefaultTheme.TextSecondary)
			} else {
				sp.SetColor(m.ContextMenuTheme.TextDisabled)
			}
			sw := font.SimpleWidth(item.Shortcut)
			st := graphics.NewText(item.Shortcut, &graphics.TextDecoration{Font: font, OnBackgroundInk: sp.Color()})
			st.Draw(canvas, subX+l.width-sw-23, sy+(itemHeight-font.LineHeight())/2+font.Baseline())
		}

		if len(item.SubMenu) > 0 && item.Enabled {
			if doc, err := meui.LoadSVGByName("chevron_right"); err == nil {
				meui.DrawSVG(canvas, doc, subX+l.width-32, sy+(itemHeight-12)/2, 12, 12)
			} else {
				arrowPaint := graphics.NewPaint()
				arrowPaint.SetColor(theme.DefaultContextMenuTheme.TextInk)
				at := graphics.NewText("\u25B6", &graphics.TextDecoration{Font: font, OnBackgroundInk: arrowPaint.Color()})
				at.Draw(canvas, subX+l.width-16, sy+itemHeight/2)
			}
		}

		sy += itemHeight
	}

	if l.openIdx >= 0 && l.child != nil {
		m.drawSubLevel(canvas, rect, l.child, parentLocalX+parentW, parentItemLocalY+m.levelItemY(l, l.openIdx), l.width)
	}
}

func (m *ContextMenu) OnMouseMove(x, y float32) {
	if !m.Enabled() {
		return
	}
	itemHeight := float32(28)
	padding := float32(4)

	if m.openSubIdx >= 0 && m.rootSub != nil {
		var levels []*submenuLevel
		for cur := m.rootSub; cur != nil; cur = cur.child {
			levels = append(levels, cur)
		}

		for i := len(levels) - 1; i >= 0; i-- {
			l := levels[i]
			lx, ly := m.levelPos(l)
			if x >= lx && x < lx+l.width && y >= ly && y < ly+l.height {
				subIdx := -1
				curY := ly
				for j, item := range l.items {
					h := itemHeight
					if item.IsSep {
						h = itemHeight / 2
					}
					if y >= curY && y < curY+h {
						subIdx = j
						break
					}
					curY += h
				}
				if subIdx != l.hoveredIdx {
					if subIdx >= 0 && !l.items[subIdx].Enabled {
						l.hoveredIdx = -1
						if l.openIdx >= 0 {
							l.openIdx = -1
							l.child = nil
						}
						m.MarkForRedraw()
					} else {
						l.hoveredIdx = subIdx
					}
					if subIdx >= 0 && len(l.items[subIdx].SubMenu) > 0 && l.items[subIdx].Enabled {
						if l.openIdx != subIdx {
							l.openIdx = subIdx
							l.child = &submenuLevel{
								items:      l.items[subIdx].SubMenu,
								hoveredIdx: -1,
								openIdx:    -1,
							}
							m.computeLevelSize(l.child)
						}
					} else if l.openIdx >= 0 {
						l.openIdx = -1
						l.child = nil
					}
					m.hoveredIdx = m.openSubIdx
					m.MarkForRedraw()
				}
				return
			}
		}

		if x > m.Width() {
			m.hoveredIdx = m.openSubIdx
			return
		}
	}

	mainIdx := -1
	curY := padding
	for i, item := range m.items {
		h := itemHeight
		if item.IsSep {
			h = itemHeight / 2
		}
		if y >= curY && y < curY+h {
			mainIdx = i
			break
		}
		curY += h
	}

	if mainIdx >= 0 && !m.items[mainIdx].Enabled {
		mainIdx = -1
	}
	if m.openSubIdx >= 0 && m.rootSub != nil {
		if mainIdx == m.openSubIdx {
			m.hoveredIdx = mainIdx
			return
		}
		if mainIdx >= 0 && len(m.items[mainIdx].SubMenu) > 0 {
			m.closeAll()
			m.hoveredIdx = mainIdx
			m.openSubIdx = mainIdx
			m.rootSub = &submenuLevel{
				items:      m.items[mainIdx].SubMenu,
				hoveredIdx: -1,
				openIdx:    -1,
			}
			m.computeLevelSize(m.rootSub)
			m.MarkForRedraw()
			return
		}
		m.closeAll()
		m.hoveredIdx = mainIdx
		m.MarkForRedraw()
		return
	}

	if mainIdx != m.hoveredIdx {
		m.hoveredIdx = mainIdx
		if mainIdx >= 0 && len(m.items[mainIdx].SubMenu) > 0 {
			m.openSubIdx = mainIdx
			m.rootSub = &submenuLevel{
				items:      m.items[mainIdx].SubMenu,
				hoveredIdx: -1,
				openIdx:    -1,
			}
			m.computeLevelSize(m.rootSub)
		}
		m.MarkForRedraw()
	}
}

func (m *ContextMenu) SetOnDismiss(fn func()) {
	m.onDismiss = fn
}

func (m *ContextMenu) OnExternalDismiss() {
	if m.onDismiss != nil {
		m.onDismiss()
	}
	m.closeAll()
	m.hoveredIdx = -1
	m.MarkForRedraw()
}

func (m *ContextMenu) OnMouseDown(x, y float32, button int) bool {
	if !m.Enabled() {
		return false
	}

	if m.rootSub != nil {
		var levels []*submenuLevel
		for cur := m.rootSub; cur != nil; cur = cur.child {
			levels = append(levels, cur)
		}

		for i := len(levels) - 1; i >= 0; i-- {
			l := levels[i]
			lx, ly := m.levelPos(l)
			if x >= lx && x < lx+l.width && y >= ly && y < ly+l.height {
				itemHeight := float32(28)
				subIdx := -1
				curY := ly
				for j, item := range l.items {
					h := itemHeight
					if item.IsSep {
						h = itemHeight / 2
					}
					if y >= curY && y < curY+h {
						subIdx = j
						break
					}
					curY += h
				}
				if subIdx >= 0 && subIdx < len(l.items) {
					item := l.items[subIdx]
					if !item.IsSep && item.Enabled {
						// Checkable items toggle on click, like the
						// top-level items below.
						if item.IsCheck {
							l.items[subIdx].Checked = !l.items[subIdx].Checked
							item.Checked = l.items[subIdx].Checked
							m.MarkForRedraw()
						}
						if item.Action != nil {
							item.Action()
						}
						if m.OnItemSelected != nil {
							m.OnItemSelected(-1, &item)
						}
						if m.onDismiss != nil {
							m.onDismiss()
						}
						m.closeAll()
						return true
					}
					return true
				}
				return true
			}
		}
	}

	// Compute item from coordinates (handles clicks without prior hover)
	itemHeight := float32(28)
	padding := float32(4)
	idx := -1
	curY := padding
	for i, item := range m.items {
		h := itemHeight
		if item.IsSep {
			h = itemHeight / 2
		}
		if y >= curY && y < curY+h {
			idx = i
			break
		}
		curY += h
	}
	if idx >= 0 && idx < len(m.items) {
		item := m.items[idx]
		if item.IsSep || !item.Enabled {
			return true
		}
		if len(item.SubMenu) > 0 {
			if m.openSubIdx != idx {
				m.openSubIdx = idx
				m.rootSub = &submenuLevel{
					items:      item.SubMenu,
					hoveredIdx: -1,
					openIdx:    -1,
				}
				m.computeLevelSize(m.rootSub)
			}
			return true
		}
		// Checkable items toggle on click (the drawn checkbox was
		// decorative: state never changed).
		if item.IsCheck {
			m.items[idx].Checked = !m.items[idx].Checked
			item.Checked = m.items[idx].Checked
			m.MarkForRedraw()
		}
		if item.Action != nil {
			item.Action()
		}
		if m.OnItemSelected != nil {
			m.OnItemSelected(-1, &item)
		}
		if m.onDismiss != nil {
			m.onDismiss()
		}
		m.closeAll()
		return true
	}
	return false
}

func (m *ContextMenu) ItemCount() int {
	return len(m.items)
}

func (m *ContextMenu) PreferredSize() (float32, float32) {
	if len(m.items) == 0 {
		return 0, 0
	}
	font := m.ContextMenuTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	maxWidth := float32(0)
	itemHeight := float32(28)
	padding := float32(8)
	totalHeight := padding * 2
	leftOffset := float32(45)
	rightPad := float32(15)
	for _, item := range m.items {
		if item.IsSep {
			totalHeight += itemHeight / 2
			continue
		}
		if item.CustomPanel != nil {
			fr := item.CustomPanel.FrameRect()
			w := fr.Width + leftOffset + rightPad
			if w > maxWidth {
				maxWidth = w
			}
			totalHeight += itemHeight
			continue
		}
		w := font.SimpleWidth(item.Label)
		if item.Shortcut != "" {
			w += font.SimpleWidth(item.Shortcut) + 30
		}
		if item.Icon != "" {
			w += 18
		}
		if len(item.SubMenu) > 0 {
			w += 16
		}
		w += leftOffset + rightPad
		if w > maxWidth {
			maxWidth = w
		}
		totalHeight += itemHeight
	}
	return maxWidth, totalHeight
}
