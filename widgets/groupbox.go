package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/layout"
	"github.com/sixnineone/meui/theme"
)

type GroupBox struct {
	meui.BaseWidget
	*theme.GroupBoxTheme
	Title        string
	FullTitle    bool
	Collapsible  bool
	Collapsed    bool
	BorderRadius float32 // 0 = use theme default
	BorderStroke float32 // 0 = use theme default
	ShowBorder   bool    // false = no border drawn
	textCache    *graphics.Text
	textCacheVer uint64
	savedHeight  float32 // pre-collapse frame height, restored on expand

	content *meui.Panel
}

func NewGroupBox() *GroupBox {
	g := &GroupBox{
		BaseWidget:    *meui.NewBaseWidget(),
		GroupBoxTheme: theme.DefaultGroupBoxTheme,
		ShowBorder:    true,
		BorderRadius:  4,
		content:       meui.NewPanel(),
	}
	// The content panel needs a LAYOUT, or it never sizes its children: Panel only
	// lays children out when p.layout != nil, so a child that asks to FILL — which is
	// what a collapsible group's body is — keeps the default widget size and clips its
	// own text to two characters. It reads as the widget failing to render rather than
	// as a missing layout.
	//
	// A VStack, with no spacing: children that carry explicit sizes keep them (the
	// layout contract is that it lays out from explicit width and height), children that
	// ask to fill get the content rect, and several children stack instead of piling up
	// at the origin. GroupBox.ValidateLayout itself used to be a no-op, which is why
	// this went unnoticed: the content was positioned at paint time and its children
	// were never asked to lay out at all.
	v := layout.NewVStack()
	v.Spacing = 0
	g.content.SetLayout(v)
	g.content.SetParent(g)
	return g
}

func (g *GroupBox) SetTitle(title string) *GroupBox {
	g.Title = title
	g.textCache = nil
	g.MarkForRedraw()
	return g
}
func (g *GroupBox) SetFullTitle(b bool) *GroupBox { g.FullTitle = b; g.MarkForRedraw(); return g }
func (g *GroupBox) SetCollapsible(b bool) *GroupBox {
	g.Collapsible = b
	g.SetNeedsLayout()
	g.MarkForRedraw()
	return g
}
func (g *GroupBox) SetCollapsed(b bool) *GroupBox {
	if g.Collapsed == b {
		return g
	}
	if b {
		// Save the expanded height so expand can restore it — the parent
		// layout trusts frame.Height for explicitly-sized widgets and will
		// never re-measure on its own.
		if h := g.FrameRect().Height; h > g.titleBarHeight() {
			g.savedHeight = h
		}
		g.Collapsed = b
		g.SetHeightDirect(g.titleBarHeight())
	} else {
		g.Collapsed = b
		restore := g.savedHeight
		if restore <= g.titleBarHeight() {
			// No saved expanded height (e.g. `collapsed: true` in QML):
			// derive it from the content's preferred height.
			if _, ph := g.PreferredDimensions(g, g.FrameRect().Width, 0); ph > restore {
				restore = ph
			}
		}
		if restore > 0 {
			g.SetHeightDirect(restore)
		}
	}
	g.SetNeedsLayout()
	g.MarkForRedraw()
	return g
}
func (g *GroupBox) SetShowBorder(b bool) *GroupBox { g.ShowBorder = b; g.MarkForRedraw(); return g }
func (g *GroupBox) SetBorderRadius(r float32) *GroupBox {
	g.BorderRadius = r
	g.MarkForRedraw()
	return g
}
func (g *GroupBox) SetBorderStroke(s float32) *GroupBox {
	g.BorderStroke = s
	g.MarkForRedraw()
	return g
}

func (g *GroupBox) AddChild(child meui.Widget) {
	g.content.AddChild(child)
}

func (g *GroupBox) SetContentLayout(l interface{}) *GroupBox {
	g.content.SetLayout(l)
	return g
}

func (g *GroupBox) SetFrameRect(rect graphics.Rect) {
	g.BaseWidget.SetFrameRect(rect)
	// Position content panel immediately so children get laid out correctly
	th := g.titleFont().LineHeight()
	padding := float32(8)
	titleBarH := th + padding + 6
	contentY := titleBarH + 4
	contentH := rect.Height - contentY - padding
	if contentH < 0 {
		contentH = 0
	}
	contentRect := graphics.NewRect(padding, contentY, rect.Width-padding*2, contentH)
	g.content.SetFrameRect(contentRect)
	if g.content.NeedsLayout() {
		g.content.ValidateLayout()
	}
}

func (g *GroupBox) PreferredDimensions(target interface{}, width, height float32) (float32, float32) {
	if g.Collapsible && g.Collapsed {
		// Collapsed — only the title bar remains; content height not allocated.
		return width, g.titleBarHeight()
	}
	_, ch := g.content.PreferredDimensions(g.content, width, height)
	return width, g.frameOverhead() + ch
}

func (g *GroupBox) MinimumDimensions() (float32, float32) {
	return 100, 50
}

func (g *GroupBox) Children() []meui.Widget {
	return g.content.Children()
}

func (g *GroupBox) ChildrenI() []interface{} {
	return g.content.ChildrenI()
}

func (g *GroupBox) SetEnabled(enabled bool) *GroupBox {
	g.BaseWidget.SetEnabled(enabled)
	g.content.SetEnabled(enabled)
	g.MarkForRedraw()
	return g
}

func (g *GroupBox) titleFont() graphics.Font {
	if g.GroupBoxTheme.Font != nil {
		return g.GroupBoxTheme.Font
	}
	if theme.DefaultTheme.BaseFont != nil {
		return theme.DefaultTheme.BaseFont
	}
	return graphics.SystemFont
}

func (g *GroupBox) titleHeight() float32 {
	return g.titleFont().LineHeight()
}

// frameOverhead returns the non-content vertical space the Paint() layout
// reserves for the title bar, top spacing and bottom padding. Paint computes
// contentH = rect.Height - (titleHeight+18) - 8, so a frame that gives the
// content its full preferred height must be titleHeight + 26 + contentH.
func (g *GroupBox) frameOverhead() float32 {
	return g.titleHeight() + 26
}

func (g *GroupBox) titleBarHeight() float32 {
	return g.titleHeight() + 14
}

func (g *GroupBox) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !g.Visible() {
		return
	}
	canvas.Save()

	font := g.titleFont()
	th := font.LineHeight()
	disabled := !g.Enabled()
	padding := float32(8)

	titleBarH := th + padding + 6
	contentY := titleBarH + 4
	contentH := rect.Height - contentY - padding
	isCollapsed := g.Collapsible && g.Collapsed
	if isCollapsed {
		contentH = 0
	}

	// Position content panel
	contentRect := graphics.NewRect(padding, contentY, rect.Width-padding*2, contentH)
	g.content.SetFrameRect(contentRect)
	if g.content.NeedsLayout() && !isCollapsed {
		g.content.ValidateLayout()
	}

	// ── Border ──
	borderSW := g.BorderStroke
	if borderSW <= 0 {
		borderSW = theme.BorderStrokeWidth
	}
	br := g.BorderRadius
	if br <= 0 {
		br = theme.Shape().GroupBox
	}

	if g.ShowBorder {
		bp := graphics.NewPaint()
		bp.SetStyle(paintstyle.Stroke)
		bp.SetStrokeWidth(borderSW)
		if disabled {
			bp.SetColor(theme.DefaultTheme.DisabledBorder)
		} else {
			bp.SetColor(theme.DefaultTheme.Border)
		}

		if g.FullTitle {
			borderH := rect.Height
			if isCollapsed {
				borderH = titleBarH
			}
			canvas.DrawRoundedRect(graphics.NewRect(0, 0, rect.Width, borderH), br, br, bp)
			if !isCollapsed {
				sp := graphics.NewPaint()
				sp.SetStyle(paintstyle.Stroke)
				sp.SetStrokeWidth(borderSW)
				sp.SetColor(theme.DefaultTheme.Border)
				canvas.DrawLine(0, titleBarH, rect.Width, titleBarH, sp)
			}
		} else {
			titleY := th / 2
			borderRect := graphics.NewRect(0, titleY, rect.Width, rect.Height-titleY)
			if isCollapsed {
				borderRect = graphics.NewRect(0, titleY, rect.Width, th-titleY+4)
			}
			canvas.DrawRoundedRect(borderRect, br, br, bp)
		}
	}

	// ── Title bar ──
	if g.FullTitle {
		// Solid title bar background
		tbarBg := graphics.NewPaint()
		if disabled {
			tbarBg.SetColor(theme.DefaultTheme.DisabledBg)
		} else {
			tbarBg.SetColor(theme.DefaultTheme.SurfaceVariant)
		}
		canvas.DrawRoundedRect(graphics.NewRect(0, 0, rect.Width, titleBarH), br, br, tbarBg)
		if !isCollapsed {
			flatBg := graphics.NewPaint()
			flatBg.SetColor(theme.DefaultTheme.SurfaceVariant)
			canvas.DrawRect(graphics.NewRect(0, titleBarH-4, rect.Width, 4), flatBg)
		}

		// Title text
		tp := graphics.NewPaint()
		if disabled {
			tp.SetColor(theme.DefaultTheme.DisabledText)
		} else {
			tp.SetColor(g.GroupBoxTheme.TextInk)
		}
		ascent := font.Baseline()
		ver := theme.ThemeVersion()
		if g.textCache == nil || g.textCacheVer != ver {
			g.textCache = graphics.NewText(g.Title, &graphics.TextDecoration{Font: font, OnBackgroundInk: tp.Color()})
			g.textCacheVer = ver
		}
		g.textCache.Draw(canvas, padding, ascent+padding)

		// Collapse arrow at far right
		if g.Collapsible {
			arrow := "▶"
			if !g.Collapsed {
				arrow = "▼"
			}
			ap := graphics.NewPaint()
			if disabled {
				ap.SetColor(theme.DefaultTheme.DisabledText)
			} else {
				ap.SetColor(g.GroupBoxTheme.TextInk)
			}
			at := graphics.NewText(arrow, &graphics.TextDecoration{Font: font, OnBackgroundInk: ap.Color()})
			aw := font.SimpleWidth(arrow)
			at.Draw(canvas, rect.Width-padding-aw-4, ascent+padding)
		}
	} else {
		// Classic mode: background patch hides border behind title
		clx := padding + 5
		collapseOffset := float32(0)
		if g.Collapsible {
			collapseOffset = 16
		}

		// Background patch — must match the PARENT surface so the border
		// just vanishes behind the title (FlatLaf titled-border style).
		tbg := graphics.NewPaint()
		if disabled {
			tbg.SetColor(theme.DefaultTheme.DisabledBg)
		} else {
			tbg.SetColor(theme.DefaultTheme.Background)
		}
		tw := font.SimpleWidth(g.Title)
		patchW := tw + padding*2 + collapseOffset + 8
		if patchW+clx > rect.Width {
			patchW = rect.Width - clx
		}
		if patchW > 0 {
			canvas.DrawRect(graphics.NewRect(clx, 0, patchW, th), tbg)
		}

		// Title text
		tp := graphics.NewPaint()
		if disabled {
			tp.SetColor(theme.DefaultTheme.DisabledText)
		} else {
			tp.SetColor(g.GroupBoxTheme.TextInk)
		}
		ascent := font.Baseline()
		ver := theme.ThemeVersion()
		if g.textCache == nil || g.textCacheVer != ver {
			g.textCache = graphics.NewText(g.Title, &graphics.TextDecoration{Font: font, OnBackgroundInk: tp.Color()})
			g.textCacheVer = ver
		}
		g.textCache.Draw(canvas, clx+collapseOffset+4+padding/2, ascent+th*0.1)

		// Collapse arrow next to title
		if g.Collapsible {
			arrow := "▶"
			if !g.Collapsed {
				arrow = "▼"
			}
			ap := graphics.NewPaint()
			if disabled {
				ap.SetColor(theme.DefaultTheme.DisabledText)
			} else {
				ap.SetColor(g.GroupBoxTheme.TextInk)
			}
			at := graphics.NewText(arrow, &graphics.TextDecoration{Font: font, OnBackgroundInk: ap.Color()})
			at.Draw(canvas, clx+4, th*0.85)
		}
	}

	// ── Content ──
	if !isCollapsed && len(g.content.Children()) > 0 {
		cf := g.content.FrameRect()
		canvas.Save()
		canvas.Translate(cf.X, cf.Y)
		g.content.Paint(canvas, graphics.NewRect(0, 0, cf.Width, cf.Height))
		canvas.Restore()
	}

	canvas.Restore()
}

func (g *GroupBox) PanelAt(x, y float32) meui.Widget {
	if !g.Visible() {
		return nil
	}
	if g.Collapsible && g.Collapsed {
		return g
	}
	cf := g.content.FrameRect()
	if x >= cf.X && x <= cf.X+cf.Width && y >= cf.Y && y <= cf.Y+cf.Height {
		if deeper := g.content.PanelAt(x-cf.X, y-cf.Y); deeper != nil {
			return deeper
		}
		return g.content
	}
	return g
}

func (g *GroupBox) OnMouseDown(x, y float32, button int) bool {
	if !g.Enabled() || button != 0 {
		return false
	}
	th := g.titleFont().LineHeight()
	padding := float32(8)

	if g.Collapsible {
		// Click anywhere on the title bar toggles collapse
		if g.FullTitle {
			titleBarH := th + padding + 6
			if y >= 0 && y <= titleBarH {
				g.SetCollapsed(!g.Collapsed)
				return true
			}
		} else {
			if y >= 0 && y <= th {
				g.SetCollapsed(!g.Collapsed)
				return true
			}
		}
	}

	if g.Collapsible && g.Collapsed {
		return false
	}
	cf := g.content.FrameRect()
	if x >= cf.X && x <= cf.X+cf.Width && y >= cf.Y && y <= cf.Y+cf.Height {
		return g.content.OnMouseDown(x-cf.X, y-cf.Y, button)
	}
	return false
}

func (g *GroupBox) OnMouseUp(x, y float32, button int) bool {
	if !g.Enabled() {
		return false
	}
	if g.Collapsible && g.Collapsed {
		return false
	}
	cf := g.content.FrameRect()
	return g.content.OnMouseUp(x-cf.X, y-cf.Y, button)
}

func (g *GroupBox) OnMouseMove(x, y float32) {
	if !g.Enabled() {
		return
	}
	if g.Collapsible && g.Collapsed {
		return
	}
	cf := g.content.FrameRect()
	if x >= cf.X && x <= cf.X+cf.Width && y >= cf.Y && y <= cf.Y+cf.Height {
		g.content.OnMouseMove(x-cf.X, y-cf.Y)
	}
}

func (g *GroupBox) ValidateLayout() {
	g.ClearNeedsLayout()
}

func (g *GroupBox) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "title":
		if s, ok := value.(string); ok {
			g.SetTitle(s)
		}
	case "fullTitle":
		if b, ok := value.(bool); ok {
			g.SetFullTitle(b)
		}
	case "collapsible":
		if b, ok := value.(bool); ok {
			g.SetCollapsible(b)
		}
	case "collapsed":
		if b, ok := value.(bool); ok {
			g.SetCollapsed(b)
		}
	case "showBorder":
		if b, ok := value.(bool); ok {
			g.ShowBorder = b
			g.MarkForRedraw()
		}
	case "borderRadius":
		if f, ok := value.(float64); ok {
			g.BorderRadius = float32(f)
			g.MarkForRedraw()
		}
	}
}
