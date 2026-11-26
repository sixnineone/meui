package widgets

import (
	"log"
	"os"
	"time"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/theme"
)

// WindowFrame is a custom title-bar controller:
//
//	[icon] [ middle: any widget, app-configured ] [ — □ ✕ ]
//
// Fixed chrome at the edges; the middle is one app-provided widget
// (MenuBar, tab strip, status HStack…). Omit the icon and the middle
// runs flush to the left edge; hide buttons for a chromeless drag bar.
// Dragging the bar moves the window natively (SDL hit-test → WM move);
// window edges resize it. The app content area starts below the bar
// (the app offsets its SetContentRect by BarHeight).
type WindowFrame struct {
	icon          string // embedded icon name; "" = no icon slot
	title         string // drawn centered in the middle when no middle widget
	titleText     *graphics.Text
	showMin       bool
	showMax       bool
	showClose     bool
	middle        meui.Widget
	barH          float32
	width         float32 // last painted width (layout cache)
	chrome        string  // "meui" (— □ ✕ right) | "macos" (traffic lights left)
	windowRadius  float32 // >0: rounded-top bar (window rounding L&F)
	windowTopOnly bool    // round the top corners only (GNOME headerbar)

	// Window outline. Zero width means "follow the theme" for the width and
	// the ink, so a theme change restyles an untouched frame; SetBorderWidth
	// (0 = no outline) and SetBorderInk pin one window instead.
	borderWidth    float32
	borderInk      graphics.Color
	borderWidthSet bool
	borderInkSet   bool

	// bar parents the middle slot into the widget tree so title-bar
	// widgets get a working Parent() chain (findWindow, dropdown
	// layers, PointToRoot all walk it). Its owner is the app's root
	// window widget (SetOwnerRoot).
	bar       *titleBarNode
	ownerRoot meui.Widget

	closeRect graphics.Rect // bar-local
	maxRect   graphics.Rect
	minRect   graphics.Rect
	hoverIdx  int // 0 none, 1 min, 2 max, 3 close
	pressed   int

	closeClicked func()
	minClicked   func()
	maxClicked   func()
	middleLogged bool

	// backendDraggable overrides this frame's bar-drag policy. Nil follows
	// barBackendDraggable (native WM drag everywhere except X11, where the
	// app moves the window itself). SetBackendDraggable pins it, which is
	// what lets a black-box test assert both classifications without
	// depending on the ambient video driver — empty before any window
	// exists, "x11" afterwards.
	backendDraggable func() bool

	// App-side double-click state is tracked by the input layer's
	// click counter; the toggle itself is deduped in toggleMaximize.
	lastToggle time.Time
}

// titleBarNode is the invisible widget between the window root and the
// middle slot. It sits at the window origin spanning the bar, so
// descendant frames resolve to window coordinates.
type titleBarNode struct {
	meui.BaseWidget
	frame *WindowFrame
}

// Parent returns the app's root window widget (may be nil until
// SetOwnerRoot is called).
func (b *titleBarNode) Parent() meui.Widget { return b.frame.ownerRoot }

// IsWindowChrome marks this subtree as window chrome rather than content: its
// coordinates start at the window's own origin, above the content tree. The
// window uses it to place popups created by chrome widgets (a title-bar combo)
// without either side doing bar arithmetic.
func (b *titleBarNode) IsWindowChrome() bool { return true }

// Frame metrics.
const (
	FrameBarHeight = float32(28)
	FrameBtnWidth  = float32(34)
	FrameEdgeSize  = float32(6) // resize border thickness (sync: mobilegles.EdgeZone)
)

// NewWindowFrame creates a bar controller with all three buttons.
func NewWindowFrame() *WindowFrame {
	f := &WindowFrame{
		showMin:   true,
		showMax:   true,
		showClose: true,
		barH:      FrameBarHeight,
	}
	f.bar = &titleBarNode{frame: f}
	f.bar.SetFrameRect(graphics.NewRect(0, 0, 0, f.barH))
	return f
}

// SetOwnerRoot wires the frame to the app's root window widget so
// title-bar widgets have a complete Parent() chain (dropdown layers,
// findWindow, PointToRoot). Call after the root widget exists. Also
// moves layer painting after the title bar (PaintOverlay) so popups
// cover the chrome.
func (f *WindowFrame) SetOwnerRoot(w meui.Widget) *WindowFrame {
	f.ownerRoot = w
	if win, ok := w.(*meui.Window); ok {
		win.SetLayerPaintExternal()
	}
	return f
}

// PaintOverlay paints the window frame's outline and then the window's
// layers (dropdowns, popups). translateY = the content-space offset the
// layers' coordinates use (the bar height); windowW/windowH are the whole
// window in window space, chrome included. Called by the host last, after
// the title bar, so the outline encloses the chrome and popups float over
// the frame.
func (f *WindowFrame) PaintOverlay(canvas *graphics.Canvas, translateY, windowW, windowH float32) {
	f.PaintBorder(canvas, windowW, windowH)
	win, ok := f.ownerRoot.(*meui.Window)
	if !ok {
		return
	}
	win.PaintLayerOverlay(canvas, translateY)
}

// PaintBorder strokes one outline around the whole window, the title bar
// included. The stroke follows the rounded window plate when one is set
// (top corners only in the GNOME headerbar style). Width 0 paints
// nothing — macOS windows have no outline.
//
// The rect is the window box exactly, with no inset: the renderer draws a
// stroke as a band INSIDE the box and pixel-snaps it, so an inset rect
// would move the left/top outline one pixel inward instead of thinning it
// (pinned by the border test).
func (f *WindowFrame) PaintBorder(canvas *graphics.Canvas, windowW, windowH float32) {
	width, ink := f.borderMetrics()
	if width <= 0 || windowW <= 0 || windowH <= 0 {
		return
	}
	paint := graphics.NewPaint()
	paint.SetColor(ink)
	paint.SetStyle(paintstyle.Stroke)
	paint.SetStrokeWidth(width)
	rect := graphics.NewRect(0, 0, windowW, windowH)
	if f.windowRadius <= 0 {
		canvas.DrawRect(rect, paint)
		return
	}
	if !f.windowTopOnly {
		canvas.DrawRoundedRect(rect, f.windowRadius, f.windowRadius, paint)
		return
	}
	// GNOME headerbar plate: the outline's bottom corners fall below the
	// window and are clipped away, leaving square bottom corners.
	canvas.Save()
	canvas.ClipRect(rect, pathop.Intersect, false)
	canvas.DrawRoundedRect(graphics.NewRect(0, 0, windowW, windowH+f.windowRadius),
		f.windowRadius, f.windowRadius, paint)
	canvas.Restore()
}

// borderMetrics resolves the outline width and ink: an explicit SetBorder*
// wins, then the active theme's chrome tokens, then the theme's border
// colour (so an imported palette still drives an untouched frame).
func (f *WindowFrame) borderMetrics() (float32, graphics.Color) {
	ch := f.chromeTokens()
	width := ch.BorderWidth
	if f.borderWidthSet {
		width = f.borderWidth
	}
	ink := ch.BorderInk
	if ink == 0 && theme.DefaultTheme != nil {
		ink = theme.DefaultTheme.Border
	}
	if f.borderInkSet {
		ink = f.borderInk
	}
	if width > 0 && ink == 0 {
		ink = 0xFF3C3C3C
	}
	return width, ink
}

// SetIcon sets the left icon (embedded icon-set name). Empty removes
// the icon slot entirely so the middle extends to the left edge.
func (f *WindowFrame) SetIcon(name string) *WindowFrame { f.icon = name; return f }

// SetMiddle installs the app-configurable center widget. Nil = the
// title fills the middle (centered, draggable). The widget is parented
// to the frame's bar node: the drag area excludes it (FrameHit marks
// interactive descendants NORMAL) and its events route through the
// normal widget tree.
func (f *WindowFrame) SetMiddle(w meui.Widget) *WindowFrame {
	f.middle = w
	if w != nil {
		w.SetParent(f.bar)
	}
	return f
}

// SetTitle sets the window title, drawn centered in the middle when no
// middle widget is installed.
func (f *WindowFrame) SetTitle(t string) *WindowFrame {
	f.title = t
	f.titleText = nil
	return f
}

// SetShowMin / SetShowMax / SetShowClose toggle the window buttons.
func (f *WindowFrame) SetShowMin(b bool) *WindowFrame   { f.showMin = b; return f }
func (f *WindowFrame) SetShowMax(b bool) *WindowFrame   { f.showMax = b; return f }
func (f *WindowFrame) SetShowClose(b bool) *WindowFrame { f.showClose = b; return f }

// OnClose / OnMinimize / OnMaximize register the button handlers.
func (f *WindowFrame) OnClose(fn func()) *WindowFrame    { f.closeClicked = fn; return f }
func (f *WindowFrame) OnMinimize(fn func()) *WindowFrame { f.minClicked = fn; return f }
func (f *WindowFrame) OnMaximize(fn func()) *WindowFrame { f.maxClicked = fn; return f }

// BarHeight returns the bar strip height (content offset).
func (f *WindowFrame) BarHeight() float32 { return f.barH }

// SetWindowRadius rounds the bar's top corners to match a rounded
// window plate (0 = square).
func (f *WindowFrame) SetWindowRadius(r float32) {
	f.windowRadius = r
}

// SetWindowTopOnly says the rounded plate rounds its top corners only
// (GNOME headerbar style), so the outline does the same. The host
// reports this through the adapter, next to SetWindowRadius.
func (f *WindowFrame) SetWindowTopOnly(topOnly bool) {
	f.windowTopOnly = topOnly
}

// SetBorderWidth pins the window outline width in pixels (0 = none).
// Unset, the width follows the active theme's Chrome token.
func (f *WindowFrame) SetBorderWidth(w float32) *WindowFrame {
	f.borderWidth, f.borderWidthSet = w, true
	return f
}

// SetBorderInk pins the window outline colour. Unset, the ink follows
// the theme's Chrome token, then the theme's Border colour.
func (f *WindowFrame) SetBorderInk(ink graphics.Color) *WindowFrame {
	f.borderInk, f.borderInkSet = ink, true
	return f
}

// SetBorder pins both the outline colour and width.
func (f *WindowFrame) SetBorder(ink graphics.Color, width float32) *WindowFrame {
	return f.SetBorderInk(ink).SetBorderWidth(width)
}

// BorderWidth returns the outline width the frame would paint right now.
func (f *WindowFrame) BorderWidth() float32 {
	w, _ := f.borderMetrics()
	return w
}

// SetChrome selects the window-button style: "meui" (— □ ✕ on the
// right), "macos" (traffic-light dots on the left), "windows" (caption
// buttons), "gnome"/"kde" (circular buttons) or "metro" (flat caption
// buttons). Empty follows the active theme's Chrome token. Behavior
// (min/max/close actions) is unchanged.
func (f *WindowFrame) SetChrome(style string) *WindowFrame {
	f.chrome = style
	return f
}

// chromeTokens returns the active theme's chrome tokens with fallbacks
// for themes that predate the token groups (IntelliJ imports).
func (f *WindowFrame) chromeTokens() theme.Chrome {
	ch := theme.Chrome{}
	if t := theme.DefaultTheme; t != nil {
		ch = t.Chrome
	}
	if ch.Height == 0 {
		ch = theme.DefaultChrome()
	}
	return ch
}

// chromeStyle resolves the active style: an explicit SetChrome wins,
// otherwise the theme's token.
func (f *WindowFrame) chromeStyle() theme.ChromeStyle {
	if f.chrome != "" {
		return theme.ChromeStyle(f.chrome)
	}
	return f.chromeTokens().Style
}

// resolveBarHeight clamps the theme's bar height to a usable range,
// falling back to the classic 28px bar for zero/absurd tokens.
func (f *WindowFrame) resolveBarHeight() float32 {
	h := f.chromeTokens().Height
	if h < 24 || h > 60 {
		h = FrameBarHeight
	}
	return h
}

// ── Layout ──────────────────────────────────────────────────────────

func (f *WindowFrame) buttonsWidth() float32 {
	n := float32(0)
	if f.showMin {
		n++
	}
	if f.showMax {
		n++
	}
	if f.showClose {
		n++
	}
	if n == 0 {
		return 0
	}
	ch := f.chromeTokens()
	switch f.chromeStyle() {
	case theme.ChromeWindows, theme.ChromeMetro, theme.ChromeGNOME, theme.ChromeKDE:
		if ch.ButtonW > 0 {
			return n*ch.ButtonW + (n-1)*ch.ButtonGap
		}
	}
	return n * FrameBtnWidth
}

func (f *WindowFrame) iconSlotWidth() float32 {
	if f.icon == "" {
		return 0
	}
	return f.barH
}

func (f *WindowFrame) layout(width float32) {
	if f.width != width {
		frameDebug("layout width=%v (was %v)", width, f.width)
	}
	// Bar height follows the active theme; re-read every layout so a
	// theme switch resizes the bar and the content offset follows.
	f.barH = f.resolveBarHeight()
	f.width = width
	f.minRect, f.maxRect, f.closeRect = graphics.Rect{}, graphics.Rect{}, graphics.Rect{}
	if f.bar != nil {
		f.bar.SetFrameRect(graphics.NewRect(0, 0, width, f.barH))
	}

	style := f.chromeStyle()
	ch := f.chromeTokens()

	if style == theme.ChromeMacOS {
		// Traffic lights on the LEFT: close, minimize, maximize.
		hitW := ch.ButtonW + ch.ButtonGap
		if hitW <= 0 {
			hitW = FrameBtnWidth
		}
		x := float32(8)
		if f.showClose {
			f.closeRect = graphics.NewRect(x, 0, hitW, f.barH)
			x += hitW
		}
		if f.showMin {
			f.minRect = graphics.NewRect(x, 0, hitW, f.barH)
			x += hitW
		}
		if f.showMax {
			f.maxRect = graphics.NewRect(x, 0, hitW, f.barH)
			x += hitW
		}
		f.layoutMiddle(x, width)
		return
	}

	if style == theme.ChromeWindows || style == theme.ChromeMetro ||
		style == theme.ChromeGNOME || style == theme.ChromeKDE {
		bw := ch.ButtonW
		if bw <= 0 {
			bw = FrameBtnWidth
		}
		bh := ch.ButtonH
		if bh <= 0 || bh > f.barH {
			bh = f.barH
		}
		by := (f.barH - bh) / 2
		rightInset := float32(0) // Windows/Metro: flush to the edge
		if style == theme.ChromeGNOME || style == theme.ChromeKDE {
			rightInset = ch.ButtonGap
		}
		x := width - rightInset
		if f.showClose {
			x -= bw
			f.closeRect = graphics.NewRect(x, by, bw, bh)
			x -= ch.ButtonGap
		}
		if f.showMax {
			x -= bw
			f.maxRect = graphics.NewRect(x, by, bw, bh)
			x -= ch.ButtonGap
		}
		if f.showMin {
			x -= bw
			f.minRect = graphics.NewRect(x, by, bw, bh)
			x -= ch.ButtonGap
		}
		f.layoutMiddle(f.iconSlotWidth(), x)
		return
	}

	// meui (default): full-height buttons flush against the right edge.
	x := width
	if f.showClose {
		x -= FrameBtnWidth
		f.closeRect = graphics.NewRect(x, 0, FrameBtnWidth, f.barH)
	}
	if f.showMax {
		x -= FrameBtnWidth
		f.maxRect = graphics.NewRect(x, 0, FrameBtnWidth, f.barH)
	}
	if f.showMin {
		x -= FrameBtnWidth
		f.minRect = graphics.NewRect(x, 0, FrameBtnWidth, f.barH)
	}
	f.layoutMiddle(f.iconSlotWidth(), x)
}

// layoutMiddle sizes the app-provided middle widget into [from, to).
func (f *WindowFrame) layoutMiddle(from, to float32) {
	if f.middle == nil {
		return
	}
	w := to - from
	if w < 0 {
		w = 0
	}
	f.middle.SetFrameRect(graphics.NewRect(from, 0, w, f.barH))
	if vc, ok := f.middle.(interface{ ValidateLayout() }); ok {
		vc.ValidateLayout()
	}
}

// ── Painting ────────────────────────────────────────────────────────

// PaintBar draws the title bar across the top of the window (bar-local
// space; the canvas origin is the window's top-left).
func (f *WindowFrame) PaintBar(c *graphics.Canvas, width float32) {
	f.layout(width)
	ch := f.chromeTokens()
	frameDebug("PaintBar width=%v barH=%v", width, f.barH)

	// Background
	bg := graphics.NewPaint()
	bg.SetColor(theme.DefaultTheme.SurfaceVariant)
	if f.windowRadius > 0 {
		// Rounded-top bar: top corners rounded AT y=0, square bottom.
		// Draw a taller rounded rect clipped to the bar strip — the
		// overshoot below the bar is clipped away, so the bottom edge
		// stays straight and the rounding lands on the window corners.
		c.Save()
		c.ClipRect(graphics.NewRect(0, 0, width, f.barH), pathop.Intersect, false)
		c.DrawRoundedRect(graphics.NewRect(0, 0, width, f.barH+f.windowRadius),
			f.windowRadius, f.windowRadius, bg)
		c.Restore()
	} else {
		c.DrawRect(graphics.NewRect(0, 0, width, f.barH), bg)
	}

	// Bottom border. Drawn at the bar's LAST pixel row (barH-0.5) so
	// the 1px snap lands on the bar, not on the content's first row.
	line := graphics.NewPaint()
	line.SetColor(theme.DefaultTheme.Border)
	c.DrawLine(0, f.barH-0.5, width, f.barH-0.5, line)

	// Optional app icon
	iconX := float32(0)
	if f.icon != "" {
		if doc, err := meui.LoadSVGByName(f.icon); err == nil && doc != nil {
			pad := (f.barH - 16) / 2
			meui.DrawSVG(c, doc, iconX+pad, pad, 16, 16)
		}
		iconX += f.iconSlotWidth()
	}

	// Window buttons — style and placement follow the chrome.
	switch f.chromeStyle() {
	case theme.ChromeMacOS:
		f.paintMacBtn(c, f.closeRect, theme.DefaultTheme.Error, f.hoverIdx == 3 || f.pressed == 3)
		f.paintMacBtn(c, f.minRect, graphics.Color(0xFFFEBC2E), f.hoverIdx == 1 || f.pressed == 1)
		f.paintMacBtn(c, f.maxRect, graphics.Color(0xFF28C840), f.hoverIdx == 2 || f.pressed == 2)
	case theme.ChromeWindows, theme.ChromeMetro:
		f.paintCaptionBtn(c, f.minRect, f.hoverIdx == 1, f.pressed == 1, false)
		f.paintCaptionBtn(c, f.maxRect, f.hoverIdx == 2, f.pressed == 2, false)
		f.paintCaptionBtn(c, f.closeRect, f.hoverIdx == 3, f.pressed == 3, true)
	case theme.ChromeGNOME, theme.ChromeKDE:
		f.paintCircleBtn(c, f.minRect, f.hoverIdx == 1, f.pressed == 1, false)
		f.paintCircleBtn(c, f.maxRect, f.hoverIdx == 2, f.pressed == 2, false)
		f.paintCircleBtn(c, f.closeRect, f.hoverIdx == 3, f.pressed == 3, true)
	default:
		f.paintBtn(c, f.minRect, f.hoverIdx == 1, false)
		f.paintBtn(c, f.maxRect, f.hoverIdx == 2, false)
		f.paintBtn(c, f.closeRect, f.hoverIdx == 3, true)
	}

	if f.middle != nil {
		mf := f.middle.FrameRect()
		if !f.middleLogged {
			f.middleLogged = true
			if cc, ok := f.middle.(interface{ Children() []meui.Widget }); ok {
				for _, ch := range cc.Children() {
					cf := ch.FrameRect()
					frameDebug("middle child %T frame=(%v,%v %vx%v)", ch, cf.X, cf.Y, cf.Width, cf.Height)
				}
			}
			mf := f.middle.FrameRect()
			frameDebug("middle host frame=(%v,%v %vx%v)", mf.X, mf.Y, mf.Width, mf.Height)
		}
		c.Save()
		c.Translate(mf.X, mf.Y)
		f.middle.Paint(c, graphics.NewRect(0, 0, mf.Width, mf.Height))
		c.Restore()
	} else if f.title != "" {
		if f.titleText == nil {
			font := theme.DefaultTheme.BaseFont
			if font == nil {
				font = graphics.SystemFont
			}
			ink := theme.DefaultTheme.TextPrimary
			f.titleText = graphics.NewText(f.title, &graphics.TextDecoration{
				Font:            font,
				OnBackgroundInk: ink,
			})
		}
		if f.titleText != nil {
			font := theme.DefaultTheme.BaseFont
			if font == nil {
				font = graphics.SystemFont
			}
			tw := font.SimpleWidth(f.title)
			var tx float32
			if ch.TitleAlign == theme.TitleCenter {
				// Centered titles ignore button placement (macOS,
				// GNOME/KDE headerbars center on the full width).
				tx = (width - tw) / 2
			} else {
				xButtons := width - f.buttonsWidth()
				tx = iconX + (xButtons-iconX-tw)/2
				if tx < iconX {
					tx = iconX
				}
			}
			baseline := (f.barH + font.LineHeight()) / 2
			if font.LineHeight() > f.barH {
				baseline = font.Baseline() + (f.barH-font.LineHeight())/2
			}
			f.titleText.Draw(c, tx, baseline)
		}
	}
}

// paintMacBtn draws a macOS traffic-light button: a colored dot with a
// subtle hover ring. Hit rects still drive the real min/max/close
// actions — this is chrome, not behavior.
func (f *WindowFrame) paintMacBtn(c *graphics.Canvas, r graphics.Rect, col graphics.Color, hovered bool) {
	if r.Width == 0 {
		return
	}
	d := f.chromeTokens().ButtonW
	if d <= 0 || d > f.barH-4 {
		d = 12
	}
	cx := r.X + r.Width/2
	cy := f.barH / 2

	p := graphics.NewPaint()
	p.SetColor(col)
	c.DrawRoundedRect(graphics.NewRect(cx-d/2, cy-d/2, d, d), d/2, d/2, p)

	if hovered {
		ring := graphics.NewPaint()
		ring.SetStyle(paintstyle.Stroke)
		ring.SetStrokeWidth(1)
		ring.SetColor(graphics.Color(0x80FFFFFF))
		c.DrawRoundedRect(graphics.NewRect(cx-d/2-1.5, cy-d/2-1.5, d+3, d+3), d/2+1.5, d/2+1.5, ring)
	}
}

// paintCaptionBtn draws a Windows/Metro caption button: a full-height
// hover rectangle with vector glyphs.
func (f *WindowFrame) paintCaptionBtn(c *graphics.Canvas, r graphics.Rect, hovered, pressed, isClose bool) {
	if r.Width == 0 {
		return
	}
	ch := f.chromeTokens()
	if hovered || pressed {
		hp := graphics.NewPaint()
		switch {
		case isClose:
			hp.SetColor(ch.CloseHoverBg)
		case pressed:
			hp.SetColor(ch.PressedBg)
		default:
			hp.SetColor(ch.HoverBg)
		}
		c.DrawRect(r, hp)
	}
	gp := graphics.NewPaint()
	glyph := ch.GlyphColor
	if isClose && (hovered || pressed) {
		glyph = 0xFFFFFFFF
	}
	if glyph == 0 {
		glyph = theme.DefaultTheme.TextPrimary
	}
	gp.SetColor(glyph)
	cx, cy := r.X+r.Width/2, r.Y+r.Height/2
	switch {
	case r == f.minRect:
		c.DrawLine(cx-5, cy, cx+5, cy, gp)
	case r == f.maxRect:
		c.DrawRect(graphics.NewRect(cx-5, cy-5, 10, 10), gp)
	default:
		c.DrawLine(cx-4, cy-4, cx+4, cy+4, gp)
		c.DrawLine(cx-4, cy+4, cx+4, cy-4, gp)
	}
}

// paintCircleBtn draws a GNOME/KDE circular button with a circular
// hover background.
func (f *WindowFrame) paintCircleBtn(c *graphics.Canvas, r graphics.Rect, hovered, pressed, isClose bool) {
	if r.Width == 0 {
		return
	}
	ch := f.chromeTokens()
	radius := r.Width / 2
	cx, cy := r.X+r.Width/2, r.Y+r.Height/2
	if hovered || pressed {
		hp := graphics.NewPaint()
		switch {
		case isClose:
			hp.SetColor(ch.CloseHoverBg)
		case pressed:
			hp.SetColor(ch.PressedBg)
		default:
			hp.SetColor(ch.HoverBg)
		}
		c.DrawCircle(cx, cy, radius, hp)
	}
	gp := graphics.NewPaint()
	glyph := ch.GlyphColor
	if isClose && (hovered || pressed) {
		glyph = 0xFFFFFFFF
	}
	if glyph == 0 {
		glyph = theme.DefaultTheme.TextPrimary
	}
	gp.SetColor(glyph)
	g := float32(4)
	switch {
	case r == f.minRect:
		c.DrawLine(cx-g, cy, cx+g, cy, gp)
	case r == f.maxRect:
		c.DrawRect(graphics.NewRect(cx-g, cy-g, g*2, g*2), gp)
	default:
		c.DrawLine(cx-g, cy-g, cx+g, cy+g, gp)
		c.DrawLine(cx-g, cy+g, cx+g, cy-g, gp)
	}
}

// paintBtn draws a window button. Glyphs are vector-drawn (line /
// stroked square / SVG cross) — no font dependency, always crisp.
func (f *WindowFrame) paintBtn(c *graphics.Canvas, r graphics.Rect, hovered, isClose bool) {
	if r.Width == 0 {
		return
	}
	if hovered {
		hp := graphics.NewPaint()
		if isClose {
			hp.SetColor(graphics.Color(0xFFE81123)) // Windows close-red
		} else {
			hp.SetColor(theme.DefaultTheme.Hovered)
		}
		c.DrawRect(r, hp)
	}
	gp := graphics.NewPaint()
	gp.SetColor(theme.DefaultTheme.TextPrimary)
	cx, cy := r.X+r.Width/2, r.Y+f.barH/2
	switch {
	case r == f.minRect: // horizontal line
		c.DrawLine(cx-5, cy, cx+5, cy, gp)
	case r == f.maxRect: // stroked square
		c.DrawRect(graphics.NewRect(cx-5, cy-5, 10, 10), gp)
	default: // close: diagonal cross (TextPrimary — the red SVG icon
		// vanishes on the dark bar)
		c.DrawLine(cx-4, cy-4, cx+4, cy+4, gp)
		c.DrawLine(cx-4, cy+4, cx+4, cy-4, gp)
	}
}

// ── Hit classification (SDL hit-test → native move/resize) ──────────

// FrameHit classifies a window-relative point. Returns SDL_HITTEST_*
// values: 0 normal (buttons/clickable), 1 draggable (bar), 2..10
// resize zones. Buttons win over edges; edges win over bar drag.
//
// The bar can only move via the WM's interactive move (Wayland forbids
// client positioning — SDL_SetWindowPosition is ignored), which SDL
// starts when hit-test returns DRAGGABLE during a press. Returning
// DRAGGABLE immediately would hand every press to the WM, so a second
// press could never register as a double-click. Instead the drag is
// withheld: a fresh press returns NORMAL, motion beyond a threshold
// flips to DRAGGABLE mid-gesture, and a second press inside the
// double-click window toggles maximize/restore.
func (f *WindowFrame) FrameHit(x, y, w, h float32) uint32 {
	// Buttons are clickable.
	if f.hitButton(x, y) > 0 {
		return 0
	}
	// Middle widgets (buttons, menus…) are interactive: SDL must not
	// turn their presses into window moves.
	if f.middle != nil {
		mf := f.middle.FrameRect()
		if x >= mf.X && x <= mf.X+mf.Width && y >= mf.Y && y <= mf.Y+mf.Height {
			if f.middleHitTest(x-mf.X, y-mf.Y) {
				return 0 // NORMAL — the widget gets the click
			}
		}
	}
	// Edges win over bar drag: the top resize strip lies INSIDE the
	// bar. NORMAL so the press reaches the app — the shim performs the
	// manual resize drag (WM hit-test resize is unreliable on
	// XWayland/KWin).
	if EdgeZone(x, y, w, h) != 0 {
		return 0
	}
	// Bar strip: backend-dependent. On X11 every button event reaches
	// the app, so the bar is NORMAL — the shim performs manual moves
	// (SetWindowPosition is honored) and the app-side clicks==2 path
	// toggles maximize/restore (the Firefox CSD pattern). On Wayland,
	// DRAGGABLE-region presses are fully swallowed (no event, no
	// press-time hit-test — verified), so only the WM's interactive
	// move can drag the window; double-click there is unsupported.
	if y >= 0 && y <= f.barH {
		if f.backendDrag() {
			return 1 // DRAGGABLE — WM move; presses invisible to us
		}
		return 0 // NORMAL — app receives every press
	}
	return 0
}

// barBackendDraggable reports whether the bar must use the native WM
// drag (true everywhere except X11, where the app handles moves and
// double-click itself). Seam for tests.
var barBackendDraggable = func() bool { return sdl3.VideoDriver() != "x11" }

// SetBackendDraggable pins this frame's backend drag policy instead of
// asking the platform. Nil restores the default. A test uses it to assert
// DRAGGABLE and NORMAL without depending on the ambient video driver,
// which is empty until a window exists and "x11" afterwards — the ordering
// dependence that made a bar assertion pass alone and fail in a suite.
func (f *WindowFrame) SetBackendDraggable(fn func() bool) *WindowFrame {
	f.backendDraggable = fn
	return f
}

// backendDrag resolves the drag policy: the per-frame override first, then
// the platform.
func (f *WindowFrame) backendDrag() bool {
	if f.backendDraggable != nil {
		return f.backendDraggable()
	}
	return barBackendDraggable()
}

// barDoubleClickDt/Px bound the app-side double-click gesture
// (internal click counter); barToggleDedupe guards the toggle itself.
const (
	barDoubleClickDt = 500 * time.Millisecond
	barDoubleClickPx = 6.0
	barToggleDedupe  = 400 * time.Millisecond
)

// toggleMaximize invokes the OnMaximize handler, deduped so a single
// user gesture can never toggle twice (hit-test path vs app path).
func (f *WindowFrame) toggleMaximize() {
	now := time.Now()
	if now.Sub(f.lastToggle) < barToggleDedupe {
		frameDebug("toggleMaximize deduped")
		return
	}
	f.lastToggle = now
	frameDebug("toggleMaximize (double-click)")
	if f.maxClicked != nil {
		f.maxClicked()
	}
}

// middleHitTest reports whether a middle-local point lands on an
// interactive descendant (a control, not empty container space).
func (f *WindowFrame) middleHitTest(x, y float32) bool {
	if f.middle == nil {
		return false
	}
	var walk func(w meui.Widget, x, y float32) bool
	walk = func(w meui.Widget, x, y float32) bool {
		cc, ok := w.(interface{ Children() []meui.Widget })
		if !ok {
			return false
		}
		for _, ch := range cc.Children() {
			cf := ch.FrameRect()
			if x >= cf.X && x <= cf.X+cf.Width && y >= cf.Y && y <= cf.Y+cf.Height {
				if chCc, ok := ch.(interface{ Children() []meui.Widget }); ok && len(chCc.Children()) > 0 {
					if walk(ch, x-cf.X, y-cf.Y) {
						return true
					}
				} else {
					return true
				}
			}
		}
		return false
	}
	return walk(f.middle, x, y)
}

// Edge zones (window-relative).
const (
	EdgeNone  = 0
	EdgeTop   = 1 << 0
	EdgeBot   = 1 << 1
	EdgeLeft  = 1 << 2
	EdgeRight = 1 << 3
)

// EdgeZone classifies a point against the window's resize border.
// Returns an Edge* bitmask (0 = not on an edge).
func EdgeZone(x, y, w, h float32) uint32 {
	es := FrameEdgeSize
	var e uint32
	if y >= 0 && y < es {
		e |= EdgeTop
	} else if y > h-es && y <= h {
		e |= EdgeBot
	}
	if x >= 0 && x < es {
		e |= EdgeLeft
	} else if x > w-es && x <= w {
		e |= EdgeRight
	}
	return e
}

// ── Bar mouse events (button clicks + hover; drag handled by SDL) ───

func (f *WindowFrame) hitButton(x, y float32) int {
	for i, r := range []graphics.Rect{f.minRect, f.maxRect, f.closeRect} {
		if r.Width > 0 && x >= r.X && x <= r.X+r.Width && y >= r.Y && y <= r.Y+r.Height {
			return i + 1 // 1 min, 2 max, 3 close
		}
	}
	return 0
}

// BarMouseDown handles a press inside the bar strip. Returns true if
// consumed (a button).
func (f *WindowFrame) BarMouseDown(x, y float32, button int) bool {
	if button != 0 {
		return false
	}
	f.pressed = f.hitButton(x, y)
	frameDebug("BarMouseDown (%v,%v) -> hit=%d rects: min=%v max=%v close=%v",
		x, y, f.pressed, f.minRect, f.maxRect, f.closeRect)
	if f.pressed > 0 {
		f.hoverIdx = f.pressed
		return true
	}
	// Middle widget clicks (menus, tabs…).
	if f.middle != nil {
		mf := f.middle.FrameRect()
		if x >= mf.X && x <= mf.X+mf.Width && y >= mf.Y && y <= mf.Y+mf.Height {
			if h, ok := f.middle.(interface {
				OnMouseDown(float32, float32, int) bool
			}); ok {
				return h.OnMouseDown(x-mf.X, y-mf.Y, button)
			}
		}
	}
	return false // empty bar → SDL hit-test drives the native drag
}

// BarDoubleClick handles a double press on the bar strip (desktop
// convention: toggles maximize/restore). Buttons and middle widgets
// consume their own clicks — false falls through to BarMouseDown.
func (f *WindowFrame) BarDoubleClick(x, y float32) bool {
	btn := f.hitButton(x, y)
	overMiddle := false
	if f.middle != nil {
		mf := f.middle.FrameRect()
		if x >= mf.X && x <= mf.X+mf.Width && y >= mf.Y && y <= mf.Y+mf.Height {
			// Deep walk: the host panel spans the bar; only actual
			// widget geometry counts (empty host space is bar).
			overMiddle = f.middleHitTest(x-mf.X, y-mf.Y)
		}
	}
	frameDebug("BarDoubleClick (%v,%v) btn=%d overMiddle=%v", x, y, btn, overMiddle)
	if btn > 0 || overMiddle || f.maxClicked == nil {
		return false
	}
	f.toggleMaximize()
	return true
}

// BarMouseUp completes a button click.
func (f *WindowFrame) BarMouseUp(x, y float32, button int) bool {
	if button != 0 {
		return false
	}
	// Middle widgets need the release to complete their clicks.
	if f.middle != nil {
		mf := f.middle.FrameRect()
		if x >= mf.X && x <= mf.X+mf.Width && y >= mf.Y && y <= mf.Y+mf.Height {
			if h, ok := f.middle.(interface {
				OnMouseUp(float32, float32, int) bool
			}); ok {
				h.OnMouseUp(x-mf.X, y-mf.Y, button)
			}
		}
	}
	if f.pressed > 0 {
		h := f.hitButton(x, y)
		switch f.pressed {
		case 1:
			if h == 1 && f.minClicked != nil {
				f.minClicked()
			}
		case 2:
			if h == 2 && f.maxClicked != nil {
				f.maxClicked()
			}
		case 3:
			if h == 3 && f.closeClicked != nil {
				f.closeClicked()
			}
		}
		f.pressed = 0
		return true
	}
	return false
}

// BarMouseMove updates button hover states and routes moves to the
// middle widgets.
func (f *WindowFrame) BarMouseMove(x, y float32) {
	h := f.hitButton(x, y)
	if h != f.hoverIdx {
		f.hoverIdx = h
	}
	if f.middle != nil {
		mf := f.middle.FrameRect()
		if h, ok := f.middle.(interface{ OnMouseMove(float32, float32) }); ok {
			h.OnMouseMove(x-mf.X, y-mf.Y)
		}
	}
}

var _ sdl3.TitleBarFrame = (*WindowFrame)(nil)

// frameDebug logs frame details when MEUI_EVENT_DEBUG is set.
func frameDebug(format string, args ...any) {
	if os.Getenv("MEUI_EVENT_DEBUG") != "" {
		log.Printf("[frame] "+format, args...)
	}
}
