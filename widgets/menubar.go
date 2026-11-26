package widgets

import (
	"fmt"
	"os"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

type MenuBar struct {
	meui.BaseWidget
	*theme.MenuBarTheme
	menus          []Menu
	activeMenu     int
	hoveredMenu    int
	ctxMenu        *ContextMenu
	OnMenuSelected func(int, *Menu)
}

type Menu struct {
	Title        string
	Items        []MenuItem // unified MenuItem type (from contextmenu.go)
	titleText    *graphics.Text
	titleTextVer uint64
}

func NewMenuBar() *MenuBar {
	return &MenuBar{
		BaseWidget:   *meui.NewBaseWidget(),
		MenuBarTheme: theme.DefaultMenuBarTheme,
		hoveredMenu:  -1,
		activeMenu:   -1,
	}
}

func (m *MenuBar) buildMenuTitleText(menu *Menu) {
	if menu.Title == "" {
		return
	}
	font := m.MenuBarTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	menu.titleText = graphics.NewText(menu.Title, &graphics.TextDecoration{Font: font, OnBackgroundInk: m.MenuBarTheme.TextInk})
}

func (m *MenuBar) AddMenu(title string, items []MenuItem) *MenuBar {
	m.menus = append(m.menus, Menu{Title: title, Items: items})
	m.buildMenuTitleText(&m.menus[len(m.menus)-1])
	m.MarkForRedraw()
	return m
}

// SetMenuItemChecked finds a menu item by menu title and item label, and sets its Checked state.
// Returns false if the menu/item is not found.
func (m *MenuBar) SetMenuItemChecked(menuTitle, itemLabel string, checked bool) bool {
	for i := range m.menus {
		if m.menus[i].Title == menuTitle {
			for j := range m.menus[i].Items {
				if m.menus[i].Items[j].Label == itemLabel {
					m.menus[i].Items[j].IsCheck = true
					m.menus[i].Items[j].Checked = checked
					m.MarkForRedraw()
					return true
				}
			}
		}
	}
	return false
}

// SetMenuItemAction finds a menu item by menu title and item label, and sets its Action.
func (m *MenuBar) SetMenuItemAction(menuTitle, itemLabel string, action func()) bool {
	for i := range m.menus {
		if m.menus[i].Title == menuTitle {
			for j := range m.menus[i].Items {
				if m.menus[i].Items[j].Label == itemLabel {
					m.menus[i].Items[j].Action = action
					return true
				}
			}
		}
	}
	return false
}

func (m *MenuBar) SetMenus(menus []Menu) *MenuBar {
	m.menus = menus
	for i := range m.menus {
		m.buildMenuTitleText(&m.menus[i])
	}
	m.MarkForRedraw()
	return m
}

func (m *MenuBar) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !m.Visible() || len(m.menus) == 0 {
		return
	}

	canvas.Save()

	menuHeight := float32(28)
	bgPaint := graphics.NewPaint()
	bgPaint.SetColor(m.MenuBarTheme.BackgroundInk)
	canvas.DrawRect(rect, bgPaint)

	sepPaint := graphics.NewPaint()
	sepPaint.SetStyle(paintstyle.Stroke)
	sepPaint.SetStrokeWidth(theme.SeparatorStrokeWidth)
	sepPaint.SetColor(m.MenuBarTheme.EdgeInk)
	canvas.DrawLine(rect.X, rect.Y+menuHeight-1, rect.X+rect.Width, rect.Y+menuHeight-1, sepPaint)

	font := m.MenuBarTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}

	x := rect.X + float32(8)
	for i := range m.menus {
		menu := &m.menus[i]
		w := font.SimpleWidth(menu.Title) + float32(16)
		menuRect := graphics.NewRect(x, rect.Y, w, menuHeight)

		if i == m.activeMenu {
			bgPaint := graphics.NewPaint()
			bgPaint.SetColor(m.MenuBarTheme.BackgroundActive)
			canvas.DrawRect(menuRect, bgPaint)
		} else if i == m.hoveredMenu {
			bgPaint := graphics.NewPaint()
			bgPaint.SetColor(m.MenuBarTheme.BackgroundHover)
			canvas.DrawRect(menuRect, bgPaint)
		}

		textY := rect.Y + (menuHeight-float32(font.LineHeight()))/2 + font.Baseline()
		ver := theme.ThemeVersion()
		if menu.titleText == nil || menu.titleTextVer != ver {
			menu.titleText = graphics.NewText(menu.Title, &graphics.TextDecoration{Font: font, OnBackgroundInk: m.MenuBarTheme.TextInk})
			menu.titleTextVer = ver
		}
		if menu.titleText != nil {
			menu.titleText.Draw(canvas, x+8, textY)
		}

		x += w + 2
	}

	// Draw dropdown menu if active
	if m.activeMenu >= 0 && m.activeMenu < len(m.menus) {
		menu := m.menus[m.activeMenu]
		if len(menu.Items) > 0 {
			titleX := rect.X + float32(8)
			for i := 0; i < m.activeMenu; i++ {
				w := font.SimpleWidth(m.menus[i].Title) + float32(16)
				titleX += w + 2
			}
			dropX := titleX
			dropY := rect.Y + menuHeight
			if m.ctxMenu == nil {
				m.ctxMenu = NewContextMenu()
			}
			m.ctxMenu.SetItems(menu.Items)
			if win := m.Window(); win != nil {
				m.ctxMenu.SetOnDismiss(func() {
					if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
						fmt.Printf("[menubar] onDismiss fired, activeMenu %d -> -1\n", m.activeMenu)
					}
					m.activeMenu = -1
					// Moves don't reach the bar while the popup is open, so
					// the hover state would stick on the dismissed title.
					m.hoveredMenu = -1
					m.MarkForRedraw()
					win.DismissPopupContextMenu()
				})
				winX, winY := rect.X+dropX, rect.Y+dropY
				for p := meui.Widget(m); p != nil && p != win; p = p.Parent() {
					fr := p.FrameRect()
					winX += fr.X
					winY += fr.Y
				}
				win.ShowPopupContextMenu(m.ctxMenu, winX, winY)
			}
		}
	}

	canvas.Restore()
}

// IsMenuStrip marks the bar for MenuStripHost: while a popup is open
// the modal backdrop forwards pointer moves here for hover-switching.
func (m *MenuBar) IsMenuStrip() bool { return true }

func (m *MenuBar) OnMouseDown(x, y float32, button int) bool {
	if !m.Enabled() {
		return false
	}
	if y > float32(28) {
		if m.activeMenu >= 0 {
			m.activeMenu = -1
			if win := m.Window(); win != nil {
				win.DismissPopupContextMenu()
			}
			m.MarkForRedraw()
		}
		return false
	}

	font := m.MenuBarTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}

	curX := float32(8)
	for i, menu := range m.menus {
		w := font.SimpleWidth(menu.Title) + float32(16)
		if x >= curX && x < curX+w {
			if m.activeMenu == i {
				m.activeMenu = -1
				if win := m.Window(); win != nil {
					win.DismissPopupContextMenu()
				}
			} else {
				m.activeMenu = i
				m.hoveredMenu = i
			}
			m.MarkForRedraw()
			if m.OnMenuSelected != nil && m.activeMenu >= 0 {
				m.OnMenuSelected(i, &menu)
			}
			return true
		}
		curX += w + 2
	}
	m.activeMenu = -1
	if win := m.Window(); win != nil {
		win.DismissPopupContextMenu()
	}
	m.MarkForRedraw()
	return false
}

func (m *MenuBar) OnMouseUp(x, y float32, button int) bool {
	return true
}

func (m *MenuBar) OnMouseMove(x, y float32) {
	if !m.Enabled() {
		return
	}
	if y > float32(28) {
		if m.hoveredMenu != -1 {
			m.hoveredMenu = -1
			m.MarkForRedraw()
		}
		return
	}

	font := m.MenuBarTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}

	curX := float32(8)
	newHovered := -1
	for i, menu := range m.menus {
		w := font.SimpleWidth(menu.Title) + float32(16)
		if x >= curX && x < curX+w {
			newHovered = i
			break
		}
		curX += w + 2
	}
	if newHovered != m.hoveredMenu {
		m.hoveredMenu = newHovered
		if m.activeMenu >= 0 && newHovered >= 0 {
			m.activeMenu = newHovered
		}
		m.MarkForRedraw()
	}
}

func (m *MenuBar) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if !m.Enabled() || len(m.menus) == 0 {
		return false
	}
	switch keyCode {
	case platform.KeyLeft:
		if m.activeMenu < 0 {
			m.activeMenu = len(m.menus) - 1
		} else {
			m.activeMenu--
			if m.activeMenu < 0 {
				m.activeMenu = len(m.menus) - 1
			}
		}
		m.hoveredMenu = m.activeMenu
		m.MarkForRedraw()
		return true
	case platform.KeyRight:
		if m.activeMenu < 0 {
			m.activeMenu = 0
		} else {
			m.activeMenu++
			if m.activeMenu >= len(m.menus) {
				m.activeMenu = 0
			}
		}
		m.hoveredMenu = m.activeMenu
		m.MarkForRedraw()
		return true
	case platform.KeyEscape:
		if m.activeMenu >= 0 {
			m.CloseMenu()
			return true
		}
		return false
	}
	return false
}

func (m *MenuBar) CloseMenu() {
	m.activeMenu = -1
	if win := m.Window(); win != nil {
		win.DismissPopupContextMenu()
	}
	m.MarkForRedraw()
}

func (m *MenuBar) MenuCount() int {
	return len(m.menus)
}

func (m *MenuBar) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "menus":
		if arr, ok := value.([]interface{}); ok {
			var menus []Menu
			for _, v := range arr {
				if m, ok := v.(Menu); ok {
					menus = append(menus, m)
				}
			}
			m.SetMenus(menus)
		}
	}
}
