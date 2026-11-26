package widgets

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/layout"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

const (
	TabHorizontal = iota
	TabVertical
)

type TabItem struct {
	ID       string
	Title    string
	Icon     string // SVG icon name or emoji
	IconSize float32
	Content  meui.Widget
	Pinned   bool   // pinned tabs show pin icon instead of close button
	Tooltip  string // optional, falls back to Title
}

// TabActionButton is a custom button rendered at the end of the tab bar.
type TabActionButton struct {
	Label    string
	Tooltip  string
	Width    float32 // 0 = default 24
	OnClick  func()
	iconText *graphics.Text
}

type TabPanel struct {
	meui.BaseContainer
	*theme.TabTheme
	tabs        []TabItem
	activeTab   int
	hoveredTab  int
	Closable    bool // when true, unpinned tabs show close button, pinned tabs show pin icon
	tabPadding  float32
	Orientation int // TabHorizontal or TabVertical
	// tabsVisible hides the tab strip while keeping the panel as the PAGE HOST. A phone
	// wants its destinations in a NavigationBar at the bottom and no top tab strip at
	// all, but it still needs something to switch between the pages — so the panel keeps
	// doing that job and stops drawing its own chrome.
	tabsVisible bool

	// Overflow scrolling
	scrollOffset float32
	showScrollL  bool
	showScrollR  bool

	// Dragging
	dragIdx    int
	dragStartX float32
	isDragging bool

	// Drag ghost image
	dragImage  *graphics.Image
	dragImageW float32
	dragImageH float32
	dragX      float32
	dragY      float32

	// Callbacks
	OnTabChanged  func(index int)
	OnTabDetached func(tabIndex int, title string, content meui.Widget, localX, localY float32)
	OnTabRenamed  func(tabIndex int, oldTitle, newTitle string)
	OnClose       func(tabID, tabName string) bool

	// Custom action buttons at end of tab bar
	ActionButtons []TabActionButton

	// Right-click context menu
	ctxMenu *ContextMenu
}

func NewTabPanel() *TabPanel {
	return &TabPanel{
		BaseContainer: meui.BaseContainer{BaseWidget: *meui.NewBaseWidget()},
		TabTheme:      theme.DefaultTabTheme,
		tabPadding:    8,
		activeTab:     0,
		hoveredTab:    -1,
		dragIdx:       -1,
		Orientation:   TabHorizontal,
		// Visible by default: hiding the strip is the OPT-IN a phone shell asks for, and
		// a zero-valued field would otherwise have hidden every existing tab bar.
		tabsVisible: true,
	}
}

func (t *TabPanel) AddTab(title string, content meui.Widget) {
	t.tabs = append(t.tabs, TabItem{Title: title, Content: content})
	if content != nil {
		content.SetParent(t)
	}
	t.SetNeedsLayout()
	t.MarkForRedraw()
}

func (t *TabPanel) AddTabEx(id, title, icon string, content meui.Widget, pinned bool) {
	t.tabs = append(t.tabs, TabItem{ID: id, Title: title, Icon: icon, Content: content, Pinned: pinned})
	if content != nil {
		content.SetParent(t)
	}
	t.SetNeedsLayout()
	t.MarkForRedraw()
}

func (t *TabPanel) InsertTab(index int, title string, content meui.Widget) {
	item := TabItem{Title: title, Content: content}
	if content != nil {
		content.SetParent(t)
	}
	if index < 0 {
		index = 0
	}
	if index > len(t.tabs) {
		index = len(t.tabs)
	}
	t.tabs = append(t.tabs[:index], append([]TabItem{item}, t.tabs[index:]...)...)
	t.SetNeedsLayout()
	t.MarkForRedraw()
}

func (t *TabPanel) RemoveTab(index int) {
	if index < 0 || index >= len(t.tabs) {
		return
	}
	// Fire OnClose — if it returns false, cancel
	if t.OnClose != nil {
		tab := t.tabs[index]
		if !t.OnClose(tab.ID, tab.Title) {
			return
		}
	}
	if t.tabs[index].Content != nil {
		if d, ok := t.tabs[index].Content.(interface{ Dispose() }); ok {
			d.Dispose()
		}
	}
	t.tabs = append(t.tabs[:index], t.tabs[index+1:]...)
	if t.activeTab >= len(t.tabs) {
		t.activeTab = len(t.tabs) - 1
	}
	t.SetNeedsLayout()
	t.MarkForRedraw()
	if t.OnTabChanged != nil {
		t.OnTabChanged(t.activeTab)
	}
}

func (t *TabPanel) SetActiveTab(index int) {
	if index >= 0 && index < len(t.tabs) && index != t.activeTab {
		old := t.pageContent(t.activeTab)
		t.activeTab = index
		t.SetNeedsLayout()
		t.MarkForRedraw()
		// Focus cannot stay on a page that is no longer on screen — Children() exposes only the
		// active page, so the focused field on the old one became invisible while still receiving
		// characters. The panel itself takes the focus, which is where a keyboard user expects to be
		// after switching pages.
		if win := t.Window(); win != nil {
			if focused := win.FocusedChild(); focused != nil && meui.ContainsWidget(old, focused) {
				win.SetFocusedChild(t)
			}
		}
		if t.OnTabChanged != nil {
			t.OnTabChanged(index)
		}
	}
}

// pageContent is the content of tab i, or nil.
func (t *TabPanel) pageContent(i int) meui.Widget {
	if i >= 0 && i < len(t.tabs) {
		return t.tabs[i].Content
	}
	return nil
}

func (t *TabPanel) ActiveTab() int { return t.activeTab }
func (t *TabPanel) TabCount() int  { return len(t.tabs) }

// SetTabText renames the tab at index at runtime.
func (t *TabPanel) SetTabText(index int, title string) bool {
	if index < 0 || index >= len(t.tabs) {
		return false
	}
	old := t.tabs[index].Title
	t.tabs[index].Title = title
	t.MarkForRedraw()
	if t.OnTabRenamed != nil && old != title {
		t.OnTabRenamed(index, old, title)
	}
	return true
}

// SetTabIcon changes the tab's icon at runtime.
func (t *TabPanel) SetTabIcon(index int, icon string) bool {
	if index < 0 || index >= len(t.tabs) {
		return false
	}
	t.tabs[index].Icon = icon
	t.SetNeedsLayout()
	t.MarkForRedraw()
	return true
}
func (t *TabPanel) SetEnabled(enabled bool) *TabPanel {
	t.BaseWidget.SetEnabled(enabled)
	for _, tab := range t.tabs {
		if tab.Content != nil {
			if s, ok := tab.Content.(interface{ SetEnabled(bool) }); ok {
				s.SetEnabled(enabled)
			}
		}
	}
	t.MarkForRedraw()
	return t
}

func (t *TabPanel) SetClosable(closable bool) { t.Closable = closable; t.MarkForRedraw() }

// SetTabsVisible shows or hides the tab strip. Hiding it keeps the panel as the page
// host — SetActiveTab still switches pages and the content fills the whole rect — which
// is what a phone shell needs when the destinations live in a NavigationBar instead.
func (t *TabPanel) SetTabsVisible(v bool) {
	if t.tabsVisible == v {
		return
	}
	t.tabsVisible = v
	t.SetNeedsLayout()
	t.MarkForRedraw()
}

// TabsVisible reports whether the tab strip is drawn.
func (t *TabPanel) TabsVisible() bool { return t.tabsVisible }

func (t *TabPanel) tabBarHeight() float32 {
	if !t.tabsVisible {
		return 0
	}
	if t.Orientation == TabVertical {
		return float32(len(t.tabs)) * (t.TabTheme.TabHeight + 2)
	}
	return t.TabTheme.TabHeight
}

// DropHitTest returns a DropZoneInfo if (lx, ly) in TabPanel-local coords
// is on the tab bar, indicating the TabPanel can accept a merge.
func (t *TabPanel) DropHitTest(lx, ly float32) *DropZoneInfo {
	if ly >= 0 && ly <= t.tabBarHeight() {
		barH := t.tabBarHeight()
		return &DropZoneInfo{
			Type:     DropMerge,
			Target:   t,
			ZoneRect: graphics.NewRect(0, 0, t.Width(), barH),
		}
	}
	return nil
}

func (t *TabPanel) SetFrameRect(rect graphics.Rect) {
	if os.Getenv("MEUI_TAB_TRACE") != "" {
		old := t.FrameRect()
		if rect.Width != old.Width || rect.Height != old.Height {
			buf := make([]byte, 4000)
			n := runtime.Stack(buf, false)
			stack := "''"
			for _, l := range strings.Split(string(buf[:n]), "\n") {
				if strings.Contains(l, "/meui/") && !strings.Contains(l, "tabpanel.go") {
					stack = strings.TrimSpace(strings.SplitN(l, " ", 2)[0])
					break
				}
			}
			fmt.Printf("[tabframe] %p %.0fx%.0f -> %.0fx%.0f (at %.0f,%.0f) via %s\n",
				t, old.Width, old.Height, rect.Width, rect.Height, rect.X, rect.Y, stack)
		}
	}
	t.BaseWidget.SetFrameRect(rect)
	t.SetNeedsLayout()
}

func (t *TabPanel) tabFont() graphics.Font {
	return tabBarFont(t.TabTheme)
}

func (t *TabPanel) computeTabWidth(font graphics.Font, tab TabItem) float32 {
	return tabBarItemWidth(font, t.barItem(tab, 0, false), t.tabPadding, t.TabTheme.CloseSize)
}

// barItem describes a tab to the shared bar painter. index and active are passed in
// rather than read from the panel so the drag ghost can render a tab in a state other
// than its own.
func (t *TabPanel) barItem(tab TabItem, index int, active bool) tabBarItem {
	return tabBarItem{
		Title:    tab.Title,
		Icon:     tab.Icon,
		IconSize: tab.IconSize,
		Active:   active,
		Hovered:  index == t.hoveredTab,
		Disabled: !t.Enabled(),
		Closable: t.Closable,
	}
}

func (t *TabPanel) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if os.Getenv("MEUI_TAB_TRACE") != "" {
		fr := t.FrameRect()
		cf := graphics.Rect{}
		if t.activeTab >= 0 && t.activeTab < len(t.tabs) && t.tabs[t.activeTab].Content != nil {
			cf = t.tabs[t.activeTab].Content.FrameRect()
		}
		fmt.Printf("[tabpaint] rect=(%.0f,%.0f %.0fx%.0f) frame=(%.0f,%.0f %.0fx%.0f) needs=%v content=(%.0f,%.0f %.0fx%.0f)\n",
			rect.X, rect.Y, rect.Width, rect.Height, fr.X, fr.Y, fr.Width, fr.Height, t.NeedsLayout(),
			cf.X, cf.Y, cf.Width, cf.Height)
	}
	if os.Getenv("MEUI_FLUSH_DEBUG") != "" {
		ox, oy := canvas.Origin()
		fmt.Printf("[tabPaint] visible=%v tabs=%d rect=(%.0f,%.0f %.0fx%.0f) origin=(%.0f,%.0f)\n",
			t.Visible(), len(t.tabs), rect.X, rect.Y, rect.Width, rect.Height, ox, oy)
	}
	if !t.Visible() || len(t.tabs) == 0 {
		return
	}
	if os.Getenv("MEUI_COMBO_DEBUG") != "" {
		fr := t.FrameRect()
		fmt.Printf("[tabpanel] paint rect=(%.0f,%.0f %.0fx%.0f) frame=(%.0f,%.0f %.0fx%.0f) needs=%v\n",
			rect.X, rect.Y, rect.Width, rect.Height, fr.X, fr.Y, fr.Width, fr.Height, t.NeedsLayout())
	}
	if rect.Width < 50 || rect.Height < t.TabTheme.TabHeight {
		return
	}

	canvas.Save()
	canvas.Translate(rect.X, rect.Y)

	if t.NeedsLayout() {
		t.ValidateLayout()
		t.ClearNeedsLayout()
	}

	font := t.tabFont()
	disabled := !t.Enabled()
	isVertical := t.Orientation == TabVertical

	localRect := graphics.NewRect(0, 0, rect.Width, rect.Height)
	if isVertical {
		t.paintVertical(canvas, localRect, font, disabled)
	} else {
		t.paintHorizontal(canvas, localRect, font, disabled)
	}

	canvas.Restore()
}

func (t *TabPanel) paintHorizontal(canvas *graphics.Canvas, rect graphics.Rect, font graphics.Font, disabled bool) {
	tabH := t.TabTheme.TabHeight

	if !t.tabsVisible {
		// No chrome at all: paint the active page into the whole rect and stop.
		t.paintActiveContent(canvas, rect)
		t.DrawTooltip(canvas, rect)
		return
	}

	// The bar's chrome is shared with TabStrip, so the two cannot drift apart.
	paintTabBar(canvas, rect.Width, tabH, t.TabTheme, disabled)

	// Compute total tab width and scroll offset
	totalW := float32(0)
	for _, tab := range t.tabs {
		totalW += t.computeTabWidth(font, tab) + 2
	}
	maxScroll := totalW - rect.Width + 20
	if maxScroll < 0 {
		maxScroll = 0
		t.scrollOffset = 0
	}
	if t.scrollOffset > maxScroll {
		t.scrollOffset = maxScroll
	}
	t.showScrollL = t.scrollOffset > 0
	t.showScrollR = t.scrollOffset < maxScroll

	// Scroll arrows
	scrollBtnW := float32(16)
	if t.showScrollL {
		ap := graphics.NewPaint()
		ap.SetColor(graphics.Color(0xFFA0A0A0))
		at := graphics.NewText("◀", &graphics.TextDecoration{Font: font, OnBackgroundInk: ap.Color()})
		at.Draw(canvas, 2, 4+font.Baseline())
	}
	if t.showScrollR {
		ap := graphics.NewPaint()
		ap.SetColor(graphics.Color(0xFFA0A0A0))
		at := graphics.NewText("▶", &graphics.TextDecoration{Font: font, OnBackgroundInk: ap.Color()})
		at.Draw(canvas, rect.Width-scrollBtnW-2, 4+font.Baseline())
	}

	// Clip tab bar to exclude scroll arrows area
	clipX := float32(0)
	if t.showScrollL {
		clipX = scrollBtnW
	}
	clipW := rect.Width
	if t.showScrollR {
		clipW -= scrollBtnW
	}
	canvas.Save()
	canvas.ClipRect(graphics.NewRect(clipX, 0, clipW, tabH), pathop.Intersect, false)

	x := t.tabPadding - t.scrollOffset
	for i, tab := range t.tabs {
		tabW := t.computeTabWidth(font, tab)
		tabRect := graphics.NewRect(x, 2, tabW, tabH-2)
		if os.Getenv("MEUI_TAB_RECT_DEBUG") != "" {
			fmt.Printf("[tabrect] panel i=%d rect=%v w=%v font=%v size=%v\n", i, tabRect, tabW, font != nil, func() float32 {
				if font == nil {
					return -1
				}
				return font.Size()
			}())
		}
		t.drawTab(canvas, i, tabRect, tab, font, disabled)
		x += tabW + 2
	}

	canvas.Restore()

	// Action buttons at end of tab bar
	if len(t.ActionButtons) > 0 {
		t.drawActionButtons(canvas, rect, font, disabled)
	}

	t.paintActiveContent(canvas, rect)

	// Tooltip via BaseWidget system
	t.DrawTooltip(canvas, rect)
}

// paintActiveContent draws the selected page. It is shared by the chrome and no-chrome
// paths so the two cannot disagree about what the current page is.
func (t *TabPanel) paintActiveContent(canvas *graphics.Canvas, rect graphics.Rect) {
	if t.activeTab < 0 || t.activeTab >= len(t.tabs) || t.tabs[t.activeTab].Content == nil {
		return
	}
	content := t.tabs[t.activeTab].Content
	cf := content.FrameRect()
	canvas.Save()
	canvas.Translate(cf.X, cf.Y)
	localCR := graphics.NewRect(0, 0, cf.Width, cf.Height)
	canvas.ClipRect(localCR, pathop.Intersect, true)
	content.Paint(canvas, localCR)
	canvas.Restore()
}

func (t *TabPanel) paintVertical(canvas *graphics.Canvas, rect graphics.Rect, font graphics.Font, disabled bool) {
	tabW := float32(120)
	contentW := rect.Width - tabW
	if contentW < 0 {
		contentW = 0
	}

	// Tab sidebar background
	sbg := graphics.NewPaint()
	if disabled {
		sbg.SetColor(theme.DefaultTheme.DisabledBg)
	} else {
		sbg.SetColor(t.TabTheme.BackgroundInk)
	}
	canvas.DrawRect(graphics.NewRect(0, 0, tabW, rect.Height), sbg)

	// Separator
	sp := graphics.NewPaint()
	sp.SetStyle(paintstyle.Stroke)
	sp.SetStrokeWidth(theme.SeparatorStrokeWidth)
	if disabled {
		sp.SetColor(theme.DefaultTheme.DisabledBorder)
	} else {
		sp.SetColor(t.TabTheme.EdgeInk)
	}
	canvas.DrawLine(tabW, 0, tabW, rect.Height, sp)

	y := float32(4)
	for i, tab := range t.tabs {
		tabRect := graphics.NewRect(2, y, tabW-4, t.TabTheme.TabHeight)

		bg := graphics.NewPaint()
		if disabled {
			bg.SetColor(theme.DefaultTheme.DisabledBg)
		} else if i == t.activeTab {
			bg.SetColor(t.TabTheme.BackgroundActive)
		} else if i == t.hoveredTab {
			bg.SetColor(t.TabTheme.BackgroundHover)
		} else {
			bg.SetColor(t.TabTheme.BackgroundInk)
		}
		r := t.TabTheme.CornerRadius
		if r > 0 {
			canvas.DrawRoundedRect(tabRect, r, r, bg)
		} else {
			canvas.DrawRect(tabRect, bg)
		}

		if !disabled && i == t.activeTab {
			hl := graphics.NewPaint()
			hl.SetColor(theme.DefaultTheme.Primary)
			hl.SetStrokeWidth(theme.FocusStrokeWidth)
			canvas.DrawLine(tabRect.X+tabRect.Width-2, tabRect.Y+4, tabRect.X+tabRect.Width-2, tabRect.Y+tabRect.Height-4, hl)
		}

		xOff := float32(4)
		if tab.Icon != "" {
			if doc, err := meui.LoadSVGByName(tab.Icon); err == nil {
				meui.DrawSVG(canvas, doc, tabRect.X+4, tabRect.Y+4, 12, 12)
			}
			xOff += 16
		}

		tp := graphics.NewPaint()
		tp.SetColor(t.TabTheme.TextInk)
		title := tab.Title
		maxW := tabW - 16 - xOff
		needsTrunc := maxW > 0 && font.SimpleWidth(title) > maxW
		if needsTrunc {
			for font.SimpleWidth(title+"…") > maxW && len(title) > 1 {
				title = title[:len(title)-1]
			}
			title += "…"
		}
		// Built fresh for this state, like the horizontal bar: a cached label keeps the
		// ink the tab had when it was built, which is how an active tab lost its colour.
		tt := graphics.NewText(title, &graphics.TextDecoration{Font: font, OnBackgroundInk: tp.Color()})
		tt.Draw(canvas, tabRect.X+xOff, textY(tabRect, font))

		y += t.TabTheme.TabHeight + 2
	}

	// Content
	if t.activeTab >= 0 && t.activeTab < len(t.tabs) && t.tabs[t.activeTab].Content != nil {
		content := t.tabs[t.activeTab].Content
		cf := content.FrameRect()
		canvas.Save()
		canvas.Translate(cf.X, cf.Y)
		content.Paint(canvas, graphics.NewRect(0, 0, cf.Width, cf.Height))
		canvas.Restore()
	}
}

func textY(rect graphics.Rect, font graphics.Font) float32 {
	if font == nil {
		font = graphics.SystemFont
	}
	return rect.Y + (rect.Height-font.LineHeight())/2 + font.Baseline()
}

// ├─ Mouse Events ──────────────────────────────────────────────

// TabHeaderRect returns the header rect of tab index in widget-local
// coordinates. Mirrors the layout math used by hitTestTab and the
// painters; ok is false when the index is out of range. Useful for
// behavioral tests that need to click a specific tab header.
func (t *TabPanel) TabHeaderRect(index int) (graphics.Rect, bool) {
	if index < 0 || index >= len(t.tabs) {
		return graphics.Rect{}, false
	}
	font := t.tabFont()
	isV := t.Orientation == TabVertical
	xPos := t.tabPadding - t.scrollOffset
	yPos := float32(4)
	for i := range t.tabs {
		var tabW, tabH float32
		if isV {
			tabW = 116
			tabH = t.TabTheme.TabHeight
		} else {
			tabW = t.computeTabWidth(font, t.tabs[i])
			tabH = t.TabTheme.TabHeight
		}
		if i == index {
			if isV {
				return graphics.NewRect(2, yPos, 116, tabH), true
			}
			return graphics.NewRect(xPos, yPos, tabW, tabH), true
		}
		xPos += tabW
	}
	return graphics.Rect{}, false
}

func (t *TabPanel) hitTestTab(x, y float32) (int, float32, float32) {
	font := t.tabFont()
	isV := t.Orientation == TabVertical
	xPos := t.tabPadding - t.scrollOffset
	yPos := float32(4)
	for i := range t.tabs {
		var tabW, tabH float32
		if isV {
			tabW = 116
			tabH = t.TabTheme.TabHeight
		} else {
			tabW = t.computeTabWidth(font, t.tabs[i])
			tabH = t.TabTheme.TabHeight
		}
		var hit bool
		if isV {
			hit = x >= 2 && x <= 118 && y >= yPos && y <= yPos+tabH
		} else {
			hit = x >= xPos && x <= xPos+tabW && y >= 0 && y <= tabH
		}
		if hit {
			return i, xPos, yPos
		}
		if isV {
			yPos += tabH + 2
		} else {
			xPos += tabW + 2
		}
	}
	return -1, 0, 0
}

// drawTab paints a single tab at tabRect on the given canvas.
// Uses the same drawing code as the tab bar loop.
func (t *TabPanel) drawTab(canvas *graphics.Canvas, i int, tabRect graphics.Rect, tab TabItem, font graphics.Font, disabled bool) {
	// Background, indicator, icon and title come from the shared bar painter — the same
	// one TabStrip uses, so a panel's tabs and a strip's tabs are the same object.
	paintTab(canvas, tabRect, t.barItem(tab, i, i == t.activeTab), font, t.TabTheme, t.tabPadding)

	// What is left is TabPanel's own: the close button and the pin.
	if !t.Closable {
		return
	}
	tabW := tabRect.Width
	_ = tabW
	cs := t.TabTheme.CloseSize
	if !tab.Pinned {
		cx := tabRect.X + tabRect.Width - cs - 4
		cy := tabRect.Y + (tabRect.Height-cs)/2
		cp := graphics.NewPaint()
		if disabled {
			cp.SetColor(theme.DefaultTheme.DisabledText)
		} else {
			cp.SetColor(graphics.Color(0xFF808080))
		}
		cp.SetStrokeWidth(1.5)
		m := cs * 0.25
		canvas.DrawLine(cx+m, cy+m, cx+cs-m, cy+cs-m, cp)
		canvas.DrawLine(cx+cs-m, cy+m, cx+m, cy+cs-m, cp)
		return
	}
	// A small pin: a filled head and a shaft.
	ps := t.TabTheme.CloseSize
	px := tabRect.X + tabRect.Width - ps - 4
	py := tabRect.Y + (tabRect.Height-ps)/2
	pp := graphics.NewPaint()
	pp.SetColor(graphics.Color(0xFF80C080))
	radius := ps * 0.3
	centerX := px + ps/2
	centerY := py + ps/2
	circleRect := graphics.NewRect(centerX-radius, centerY-radius, radius*2, radius*2)
	canvas.DrawOval(circleRect, pp)
	pp.SetStyle(paintstyle.Stroke)
	pp.SetStrokeWidth(1.5)
	canvas.DrawLine(centerX, centerY+radius, centerX, centerY+radius+3, pp)
}

// renderDragImage creates a ghost image of the tab being dragged
// using the same drawing code as the actual tab bar.
func (t *TabPanel) renderDragImage(idx int) {
	if idx < 0 || idx >= len(t.tabs) {
		return
	}
	font := t.tabFont()
	if font == nil {
		return
	}

	tab := t.tabs[idx]
	tabW := t.computeTabWidth(font, tab)
	tabH := t.TabTheme.TabHeight
	w := int(tabW) + 4
	h := int(tabH + 4)
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}

	img, err := graphics.NewImageFromDrawing(offscreenDevice(t), w, h, 72, func(c *graphics.Canvas) {
		clear := graphics.NewPaint()
		clear.SetColor(graphics.Color(0x00000000))
		c.DrawRect(graphics.NewRect(0, 0, float32(w), float32(h)), clear)
		tabRect := graphics.NewRect(2, 2, tabW, tabH)
		t.drawTab(c, idx, tabRect, tab, font, false)
	})
	if err != nil {
		t.dragImage = nil
		return
	}
	t.dragImage = img
	t.dragImageW = float32(w)
	t.dragImageH = float32(h)
}

// actionButtonAtX returns the index and rect of the action button at x, or -1.
func (t *TabPanel) actionButtonAtX(x float32) (int, graphics.Rect) {
	tabH := t.TabTheme.TabHeight
	// Start from the right edge, going left
	btnX := t.Width() - 4
	for i := len(t.ActionButtons) - 1; i >= 0; i-- {
		ab := t.ActionButtons[i]
		bw := ab.Width
		if bw <= 0 {
			bw = 24
		}
		btnX -= bw
		rect := graphics.NewRect(btnX, 0, bw, tabH)
		if x >= rect.X && x <= rect.X+rect.Width {
			return i, rect
		}
		btnX -= 4 // spacing
	}
	return -1, graphics.Rect{}
}

func (t *TabPanel) drawActionButtons(canvas *graphics.Canvas, rect graphics.Rect, font graphics.Font, disabled bool) {
	tabH := t.TabTheme.TabHeight
	ink := t.TabTheme.TextInk
	if disabled {
		ink = theme.DefaultTheme.DisabledText
	}
	btnX := t.Width() - 4
	if btnX < 0 {
		btnX = 0
	}
	for i := len(t.ActionButtons) - 1; i >= 0; i-- {
		ab := t.ActionButtons[i]
		bw := ab.Width
		if bw <= 0 {
			bw = 24
		}
		btnX -= bw
		btnRect := graphics.NewRect(btnX, 0, bw, tabH)

		// Separator before first button
		if i == len(t.ActionButtons)-1 && btnX > 10 {
			sp := graphics.NewPaint()
			sp.SetStyle(paintstyle.Stroke)
			sp.SetStrokeWidth(1)
			sp.SetColor(t.TabTheme.EdgeInk)
			canvas.DrawLine(btnX-4, 4, btnX-4, tabH-4, sp)
		}

		// Background — slightly lighter than tab bar
		bg := graphics.NewPaint()
		bg.SetColor(t.TabTheme.BackgroundHover)
		canvas.DrawRect(btnRect, bg)

		// Draw icon as SVG shape
		if ab.Label != "" {
			t.drawActionIcon(canvas, btnX, btnRect, bw, ab.Label, ink)
		}

		btnX -= 4 // spacing
	}
}

// drawActionIcon draws the action button icon as a canvas shape.
func (t *TabPanel) drawActionIcon(canvas *graphics.Canvas, btnX float32, btnRect graphics.Rect, bw float32, label string, ink graphics.Color) {
	cx := btnX + bw/2
	cy := btnRect.Y + btnRect.Height/2
	p := graphics.NewPaint()
	p.SetStyle(paintstyle.Stroke)
	p.SetStrokeWidth(1.5)
	p.SetColor(ink)

	switch label {
	case "+":
		// Plus: two perpendicular lines
		s := bw * 0.3
		canvas.DrawLine(cx-s, cy, cx+s, cy, p)
		canvas.DrawLine(cx, cy-s, cx, cy+s, p)
	case "⋮":
		// Three dots: three small filled circles
		dotR := float32(1.5)
		spacing := float32(4)
		fp := graphics.NewPaint()
		fp.SetColor(ink)
		for dy := -spacing; dy <= spacing; dy += spacing {
			dotRect := graphics.NewRect(cx-dotR, cy+dy-dotR, dotR*2, dotR*2)
			canvas.DrawOval(dotRect, fp)
		}
	default:
		// Fall back to text
		tp := graphics.NewPaint()
		tp.SetColor(ink)
		tt := graphics.NewText(label, &graphics.TextDecoration{Font: t.tabFont(), OnBackgroundInk: tp.Color()})
		tw := t.tabFont().SimpleWidth(label)
		tt.Draw(canvas, btnX+(bw-tw)/2, textY(btnRect, t.tabFont()))
	}
}

// showTabContextMenu displays a right-click context menu on the tab bar.
func (t *TabPanel) showTabContextMenu(idx int, x, y float32) {
	if idx < 0 || idx >= len(t.tabs) {
		return
	}
	if t.ctxMenu == nil {
		t.ctxMenu = NewContextMenu()
	}
	tab := t.tabs[idx]
	label := "Pin Tab"
	if tab.Pinned {
		label = "Unpin Tab"
	}
	pinIdx := idx
	renameIdx := idx
	t.ctxMenu.SetItems([]MenuItem{
		{Label: "Rename Tab", Enabled: true, Action: func() {
			ShowInputDialog(t.Window(), "Rename Tab", "New name:", func(text string, ok bool) {
				if ok && text != "" && renameIdx >= 0 && renameIdx < len(t.tabs) {
					old := t.tabs[renameIdx].Title
					t.tabs[renameIdx].Title = text
					t.MarkForRedraw()
					if t.OnTabRenamed != nil {
						t.OnTabRenamed(renameIdx, old, text)
					}
				}
			})
		}},
		{Label: label, Enabled: true, Action: func() {
			if pinIdx >= 0 && pinIdx < len(t.tabs) {
				t.tabs[pinIdx].Pinned = !t.tabs[pinIdx].Pinned
				t.MarkForRedraw()
			}
		}},
	})
	if win := t.Window(); win != nil {
		t.ctxMenu.SetOnDismiss(func() {
			win.DismissPopupContextMenu()
		})
		pt := t.PointToRoot(x, y)
		win.ShowPopupContextMenu(t.ctxMenu, pt.X, pt.Y)
	}
}

func (t *TabPanel) OnMouseDown(x, y float32, button int) bool {
	if !t.Enabled() {
		return false
	}
	isV := t.Orientation == TabVertical

	tabBarH := t.TabTheme.TabHeight
	if isV {
		tabBarH = float32(len(t.tabs)) * (t.TabTheme.TabHeight + 2)
	}

	// Right-click on tab bar → context menu
	if button == mouseButtonRight && y >= 0 && y <= tabBarH {
		idx, _, _ := t.hitTestTab(x, y)
		if idx >= 0 && idx < len(t.tabs) {
			t.showTabContextMenu(idx, x, y)
			return true
		}
		return true
	}

	if button != mouseButtonLeft {
		return false
	}

	// Action button clicks
	if !isV && y >= 0 && y <= tabBarH && len(t.ActionButtons) > 0 {
		if idx, _ := t.actionButtonAtX(x); idx >= 0 {
			if idx < len(t.ActionButtons) && t.ActionButtons[idx].OnClick != nil {
				t.ActionButtons[idx].OnClick()
			}
			return true
		}
	}

	// Scroll arrow clicks
	if !isV && y >= 0 && y <= tabBarH {
		if t.showScrollL && x >= 0 && x <= 16 {
			t.scrollOffset -= 100
			t.MarkForRedraw()
			return true
		}
		if t.showScrollR && x >= t.FrameRect().Width-16 && x <= t.FrameRect().Width {
			t.scrollOffset += 100
			t.MarkForRedraw()
			return true
		}
	}

	// Tab hit test
	if y >= 0 && y <= tabBarH {
		idx, tx, _ := t.hitTestTab(x, y)
		if idx >= 0 {
			tab := t.tabs[idx]
			cs := t.TabTheme.CloseSize
			if t.Closable && !tab.Pinned {
				font := t.tabFont()
				tabW := t.computeTabWidth(font, tab)
				cx := tx + tabW - cs - 4
				cy := float32(0)
				if isV {
					cy = tx + (t.TabTheme.TabHeight-cs)/2
				} else {
					cy = (t.TabTheme.TabHeight - cs) / 2
				}
				if x >= cx && x <= cx+cs && y >= cy && y <= cy+cs {
					// OnClose can cancel
					if t.OnClose != nil {
						tab := t.tabs[idx]
						if !t.OnClose(tab.ID, tab.Title) {
							return true
						}
					}
					t.RemoveTab(idx)
					return true
				}
			}
			// Start drag
			t.dragIdx = idx
			t.dragStartX = x
			t.isDragging = false
			t.dragX = x
			t.dragY = y
			t.renderDragImage(idx)
			if t.dragImage != nil {
				rootPt := t.PointToRoot(x, y)
				meui.SetDragOverlay(t.dragImage, rootPt.X+10, rootPt.Y+10, t.dragImageW, t.dragImageH)
			}
			t.SetActiveTab(idx)
			return true
		}
	}

	// Route to content
	if t.activeTab >= 0 && t.activeTab < len(t.tabs) && t.tabs[t.activeTab].Content != nil {
		content := t.tabs[t.activeTab].Content
		cf := content.FrameRect()
		if el, ok := content.(interface {
			OnMouseDown(float32, float32, int) bool
		}); ok {
			return el.OnMouseDown(x-cf.X, y-cf.Y, button)
		}
	}
	return false
}

func (t *TabPanel) OnMouseUp(x, y float32, button int) bool {
	if !t.Enabled() {
		return false
	}

	// Complete drag reorder or detach
	if t.dragIdx >= 0 && t.isDragging {
		idx, _, _ := t.hitTestTab(x, y)
		if idx >= 0 && idx != t.dragIdx {
			item := t.tabs[t.dragIdx]
			t.tabs = append(t.tabs[:t.dragIdx], t.tabs[t.dragIdx+1:]...)
			if idx > t.dragIdx {
				idx--
			}
			t.tabs = append(t.tabs[:idx], append([]TabItem{item}, t.tabs[idx:]...)...)
			t.activeTab = idx
			t.SetNeedsLayout()
			t.MarkForRedraw()
		} else if idx < 0 {
			// Dragged outside tab bar → detach tab as new floating window
			t.detachTab(t.dragIdx, x, y)
		}
	}
	t.dragIdx = -1
	t.isDragging = false
	t.dragImage = nil
	meui.ClearDragOverlay()

	if t.activeTab >= 0 && t.activeTab < len(t.tabs) && t.tabs[t.activeTab].Content != nil {
		content := t.tabs[t.activeTab].Content
		cf := content.FrameRect()
		if el, ok := content.(interface {
			OnMouseUp(float32, float32, int) bool
		}); ok {
			return el.OnMouseUp(x-cf.X, y-cf.Y, button)
		}
	}
	return false
}

// detachTab removes the tab at idx, fires OnTabDetached, and cleans up
// the TabPanel if it becomes empty.
func (t *TabPanel) detachTab(idx int, localX, localY float32) {
	if idx < 0 || idx >= len(t.tabs) {
		return
	}

	tab := t.tabs[idx]
	if tab.Content == nil {
		return
	}

	title := tab.Title
	content := tab.Content
	tab.Content = nil

	t.RemoveTab(idx)

	// Notify parent (MdiContainer) about the detached tab
	if t.OnTabDetached != nil {
		t.OnTabDetached(idx, title, content, localX, localY)
	}

	// Remove from parent StackPanel if empty
	if len(t.tabs) == 0 {
		parentI := t.Parent()
		if sp, ok := parentI.(*StackPanel); ok {
			sp.RemoveWidget(t)
		}
	}
}

func (t *TabPanel) OnMouseMove(x, y float32) {
	if !t.Enabled() {
		return
	}

	// Drag tracking
	if t.dragIdx >= 0 {
		t.dragX = x
		t.dragY = y
		if meui.GlobalOverlay != nil {
			rp := t.PointToRoot(x, y)
			meui.GlobalOverlay.X = rp.X + 10
			meui.GlobalOverlay.Y = rp.Y + 10
		}
		if !t.isDragging && abs(x-t.dragStartX) > 8 {
			t.isDragging = true
		}
		if t.isDragging {
			t.MarkForRedraw()
			return
		}
	}

	// Hover tracking + tooltip
	isV := t.Orientation == TabVertical
	tabBarH := t.TabTheme.TabHeight
	if isV {
		tabBarH = float32(len(t.tabs)) * (t.TabTheme.TabHeight + 2)
	}

	if y >= 0 && y <= tabBarH {
		idx, _, _ := t.hitTestTab(x, y)
		if idx != t.hoveredTab {
			t.hoveredTab = idx
			if idx >= 0 {
				tip := t.tabs[idx].Tooltip
				if tip == "" {
					tip = t.tabs[idx].Title
				}
				t.SetTooltip(tip)
				t.StartTooltipTimer(x, y)
			} else {
				t.StopTooltipTimer()
			}
			t.MarkForRedraw()
		}
	} else {
		if t.hoveredTab != -1 {
			t.hoveredTab = -1
			t.StopTooltipTimer()
			t.MarkForRedraw()
		}
		if t.activeTab >= 0 && t.activeTab < len(t.tabs) && t.tabs[t.activeTab].Content != nil {
			content := t.tabs[t.activeTab].Content
			cf := content.FrameRect()
			if el, ok := content.(interface{ OnMouseMove(float32, float32) }); ok {
				el.OnMouseMove(x-cf.X, y-cf.Y)
			}
		}
	}
}

func (t *TabPanel) OnMouseWheel(x, y float32, dx, dy float32) bool {
	if !t.Enabled() {
		return false
	}
	// Mouse wheel on tab bar → navigate to prev/next tab
	if dy != 0 && len(t.tabs) > 0 {
		tabBarH := t.TabTheme.TabHeight
		if t.Orientation == TabVertical {
			tabBarH = float32(len(t.tabs)) * (t.TabTheme.TabHeight + 2)
		}
		if y >= 0 && y <= tabBarH {
			if dy > 0 && t.activeTab < len(t.tabs)-1 {
				t.SetActiveTab(t.activeTab + 1)
			} else if dy < 0 && t.activeTab > 0 {
				t.SetActiveTab(t.activeTab - 1)
			}
			return true
		}
	}
	if t.activeTab >= 0 && t.activeTab < len(t.tabs) && t.tabs[t.activeTab].Content != nil {
		content := t.tabs[t.activeTab].Content
		cf := content.FrameRect()
		if sh, ok := content.(interface {
			OnMouseWheel(float32, float32, float32, float32) bool
		}); ok {
			return sh.OnMouseWheel(x-cf.X, y-cf.Y, dx, dy)
		}
	}
	return false
}

func (t *TabPanel) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if !t.Enabled() || len(t.tabs) == 0 {
		return false
	}
	if mod.ControlDown() && (keyCode == platform.KeyTab || keyCode == platform.KeyPageDown) {
		t.SetActiveTab((t.activeTab + 1) % len(t.tabs))
		return true
	}
	if mod.ControlDown() && mod.ShiftDown() && keyCode == platform.KeyTab {
		prev := t.activeTab - 1
		if prev < 0 {
			prev = len(t.tabs) - 1
		}
		t.SetActiveTab(prev)
		return true
	}
	switch keyCode {
	case platform.KeyLeft:
		if t.activeTab > 0 {
			t.SetActiveTab(t.activeTab - 1)
		}
		return true
	case platform.KeyRight:
		if t.activeTab < len(t.tabs)-1 {
			t.SetActiveTab(t.activeTab + 1)
		}
		return true
	case platform.KeyHome:
		t.SetActiveTab(0)
		return true
	case platform.KeyEnd:
		t.SetActiveTab(len(t.tabs) - 1)
		return true
	}
	return false
}

// ├─ Layout ────────────────────────────────────────────────────

func (t *TabPanel) ValidateLayout() {
	if !t.NeedsLayout() {
		return
	}
	frame := t.FrameRect()
	if os.Getenv("MEUI_COMBO_DEBUG") != "" {
		fmt.Printf("[tabpanel] validate frame=(%.0f,%.0f %.0fx%.0f)\n", frame.X, frame.Y, frame.Width, frame.Height)
	}
	if frame.Width < 50 || frame.Height < t.TabTheme.TabHeight {
		t.ClearNeedsLayout()
		return
	}

	var contentRect graphics.Rect
	if t.Orientation == TabVertical {
		contentRect = graphics.NewRect(120, 0, frame.Width-120, frame.Height)
	} else {
		// tabBarHeight(), NOT TabTheme.TabHeight: with the strip hidden the bar is zero
		// tall and the content must start at the top. Using the raw theme value left a
		// 28px band of dead space above every page in the mobile shell.
		barH := t.tabBarHeight()
		contentRect = graphics.NewRect(0, barH, frame.Width, frame.Height-barH)
	}

	if t.activeTab >= 0 && t.activeTab < len(t.tabs) && t.tabs[t.activeTab].Content != nil {
		content := t.tabs[t.activeTab].Content
		cr := contentRect
		// Apply layout margins from the content widget
		if lp, ok := content.(interface{ GetLayoutProps() *layout.LayoutProps }); ok {
			if props := lp.GetLayoutProps(); props != nil {
				if props.TopMargin > 0 || props.LeftMargin > 0 || props.RightMargin > 0 || props.BottomMargin > 0 {
					cr = graphics.NewRect(
						contentRect.X+props.LeftMargin,
						contentRect.Y+props.TopMargin,
						contentRect.Width-props.LeftMargin-props.RightMargin,
						contentRect.Height-props.TopMargin-props.BottomMargin,
					)
				}
			}
		}
		if s, ok := content.(interface{ SetFrameRect(graphics.Rect) }); ok {
			s.SetFrameRect(cr)
		}
		if aw, ok := content.(interface{ GetAnchors() *layout.Anchors }); ok {
			anchors := aw.GetAnchors()
			if anchors != nil {
				if anchors.Fill == content {
					anchors.Fill = t
				}
				if anchors.CenterIn == content {
					anchors.CenterIn = t
				}
				if anchors.Fill != nil {
					fr := contentRect
					if anchors.Margins > 0 {
						fr = graphics.NewRect(contentRect.X+anchors.Margins, contentRect.Y+anchors.Margins,
							contentRect.Width-anchors.Margins*2, contentRect.Height-anchors.Margins*2)
					}
					if s, ok := content.(interface{ SetFrameRect(graphics.Rect) }); ok {
						s.SetFrameRect(fr)
					}
				}
			}
		}
		anchorLayout := layout.NewAnchorLayout()
		anchorLayout.Layout(content, graphics.NewRect(0, 0, contentRect.Width, contentRect.Height))
		if vc, ok := content.(interface{ ValidateLayout() }); ok {
			vc.ValidateLayout()
		}
	}
	t.ClearNeedsLayout()
}

func (t *TabPanel) Children() []meui.Widget {
	if t.activeTab >= 0 && t.activeTab < len(t.tabs) && t.tabs[t.activeTab].Content != nil {
		return []meui.Widget{t.tabs[t.activeTab].Content}
	}
	return nil
}

func (t *TabPanel) ChildrenI() []interface{} {
	result := make([]interface{}, len(t.tabs))
	for i, w := range t.Children() {
		result[i] = w
	}
	return result
}

func (t *TabPanel) PanelAt(x, y float32) meui.Widget {
	tabBarH := t.tabBarHeight()
	if t.Orientation == TabVertical {
		tabBarH = 120
	}
	if y >= 0 && y <= tabBarH {
		return t
	}
	if t.activeTab >= 0 && t.activeTab < len(t.tabs) && t.tabs[t.activeTab].Content != nil {
		content := t.tabs[t.activeTab].Content
		cf := content.FrameRect()
		lx, ly := x-cf.X, y-cf.Y
		if container, ok := content.(meui.Container); ok {
			if deeper := container.PanelAt(lx, ly); deeper != nil {
				return deeper
			}
		}
		return content
	}
	return t
}

func (t *TabPanel) Focusable() bool { return t.Enabled() }

func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
