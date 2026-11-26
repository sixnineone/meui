package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/theme"
)

// tabbar.go — the tab bar's drawing, shared by TabPanel and TabStrip.
//
// The two widgets are deliberately separate: TabPanel keeps growing features that make
// no sense for a radio group (content panes, close buttons, drag reorder, context
// menus), and TabStrip has none of them. What they must NOT do is drift apart visually,
// because a strip and a panel's bar sitting in the same window have to look like the
// same control. So the APPEARANCE lives here, once, and each widget supplies only its
// own state and its own extras.
//
// The split is: geometry and painting of a tab (background, active indicator, icon,
// title, truncation) plus the bar's background and separator. Everything else — the
// close button, the pin icon, the drag ghost, the scroll arrows, the action buttons —
// stays with the widget that has it.

// tabBarItem is one tab's appearance, the input the shared painter needs. It is a plain
// value rather than an interface so both widgets can fill it without adapting their
// own item types.
type tabBarItem struct {
	Title string
	Icon  string
	// IconSize overrides the default icon box; zero uses 12 like TabPanel does.
	IconSize float32
	Active   bool
	Hovered  bool
	Disabled bool
	// Closable reserves room for a close button, which the caller draws itself.
	Closable bool
}

// tabBarFont resolves the font a tab bar draws in: the theme's, then the toolkit
// default, then the system font — the same fallback chain TabPanel has always used.
func tabBarFont(tt *theme.TabTheme) graphics.Font {
	if tt != nil && tt.Font != nil {
		return tt.Font
	}
	if theme.DefaultTheme.BaseFont != nil {
		return theme.DefaultTheme.BaseFont
	}
	return graphics.SystemFont
}

// tabBarItemWidth measures one tab. It is the single source of a tab's width, so a hit
// test, a scroll calculation and the painter cannot disagree about where a tab ends.
func tabBarItemWidth(font graphics.Font, item tabBarItem, padding float32, closeSize float32) float32 {
	closeW := float32(0)
	if item.Closable {
		closeW = closeSize + 4
	}
	iconW := float32(0)
	if item.Icon != "" {
		if item.IconSize > 0 {
			iconW = item.IconSize + 4
		} else {
			iconW = 16
		}
	}
	if item.Title == "" {
		// Icon-only tab.
		return iconW + padding + 4
	}
	return font.SimpleWidth(item.Title) + padding*2 + 12 + closeW + iconW
}

// tabBarHeightOf is the height of a horizontal bar.
func tabBarHeightOf(tt *theme.TabTheme) float32 {
	if tt == nil {
		return 0
	}
	return tt.TabHeight
}

// paintTabBar paints the bar's background and its separator line — the strip of chrome
// the tabs sit on.
func paintTabBar(canvas *graphics.Canvas, width, tabH float32, tt *theme.TabTheme, disabled bool) {
	bg := graphics.NewPaint()
	if disabled {
		bg.SetColor(theme.DefaultTheme.DisabledBg)
	} else {
		bg.SetColor(tt.BackgroundInk)
	}
	canvas.DrawRect(graphics.NewRect(0, 0, width, tabH), bg)

	sp := graphics.NewPaint()
	sp.SetStyle(paintstyle.Stroke)
	sp.SetStrokeWidth(theme.SeparatorStrokeWidth)
	if disabled {
		sp.SetColor(theme.DefaultTheme.DisabledBorder)
	} else {
		sp.SetColor(tt.EdgeInk)
	}
	canvas.DrawLine(0, tabH, width, tabH, sp)
}

// paintTab paints one tab: its background by state, the active indicator, the icon and
// the truncated title. It does NOT draw a close button or a pin — those belong to the
// widget that offers them.
func paintTab(canvas *graphics.Canvas, tabRect graphics.Rect, item tabBarItem,
	font graphics.Font, tt *theme.TabTheme, padding float32) {

	bg := graphics.NewPaint()
	switch {
	case item.Disabled:
		bg.SetColor(theme.DefaultTheme.DisabledBg)
	case item.Active:
		bg.SetColor(tt.BackgroundActive)
	case item.Hovered:
		bg.SetColor(tt.BackgroundHover)
	default:
		bg.SetColor(tt.BackgroundInk)
	}
	if r := tt.CornerRadius; r > 0 {
		canvas.DrawRoundedRect(tabRect, r, r, bg)
	} else {
		canvas.DrawRect(tabRect, bg)
	}

	if !item.Disabled && item.Active && tt.IndicatorHeight > 0 {
		hl := graphics.NewPaint()
		hl.SetColor(tt.IndicatorInk)
		hl.SetStrokeWidth(tt.IndicatorHeight)
		bottomY := tabRect.Y + tabRect.Height
		canvas.DrawLine(tabRect.X+4, bottomY, tabRect.X+tabRect.Width-4, bottomY, hl)
	}

	xOff := padding / 2

	if item.Icon != "" {
		iconSize := item.IconSize
		if iconSize <= 0 {
			iconSize = 12
		}
		ip := graphics.NewPaint()
		if item.Disabled {
			ip.SetColor(theme.DefaultTheme.DisabledText)
		} else {
			ip.SetColor(tt.TextInk)
		}
		// An SVG when the toolkit can load one by name, the name itself otherwise —
		// the same fallback TabPanel has always used.
		if doc, err := meui.LoadSVGByName(item.Icon); err == nil {
			meui.DrawSVG(canvas, doc, tabRect.X+xOff, (tabRect.Height-iconSize)/2, iconSize, iconSize)
		} else {
			it := graphics.NewText(item.Icon, &graphics.TextDecoration{Font: font, OnBackgroundInk: ip.Color()})
			it.Draw(canvas, tabRect.X+xOff, textY(tabRect, font))
		}
		xOff += 16
	}

	if item.Title == "" {
		return
	}
	tp := graphics.NewPaint()
	switch {
	case item.Disabled:
		tp.SetColor(theme.DefaultTheme.DisabledText)
	case item.Active:
		tp.SetColor(tt.TextActive)
	default:
		tp.SetColor(tt.TextInk)
	}
	maxTitleW := tabRect.Width - xOff - padding/2
	if item.Closable {
		maxTitleW -= tt.CloseSize + 4
	}
	title := item.Title
	truncated := maxTitleW > 0 && font.SimpleWidth(title) > maxTitleW
	if truncated {
		for font.SimpleWidth(title+"…") > maxTitleW && len(title) > 1 {
			title = title[:len(title)-1]
		}
		title += "…"
	}
	// The text is built for THIS state every time. A cache keyed on the theme version
	// alone cannot be right: the ink depends on whether the tab is active, so a cached
	// label kept whatever colour the tab had when it was built — which is how an active
	// tab's label came out in the inactive ink. (The cache that did this was also
	// ineffective: its gate compared the cache's version 0 against a theme version that
	// is 0 until an app calls SetDefaultTheme, so it either froze the wrong ink or was
	// bypassed on every paint.)
	text := graphics.NewText(title, &graphics.TextDecoration{Font: font, OnBackgroundInk: tp.Color()})
	text.Draw(canvas, tabRect.X+xOff, textY(tabRect, font))
}
