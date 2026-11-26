package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

// ScrollbarHost owns scrollbar geometry and interaction for any widget
// with scrollable content. All scrollable widgets (ScrollPane, Tree,
// Table, DataTable, ...) delegate their scrollbar mechanics here so the
// conventions exist exactly once.
//
// Units are ABSOLUTE PIXELS, everywhere:
//
//	bar.Min = 0
//	bar.Max = content extent (px)
//	bar.VisibleAmount = viewport extent (px)
//	bar.Value = scroll offset (px), kept in [0, Max-VisibleAmount]
//
// The host handles: bar visibility (with corner interplay), bar layout,
// thumb geometry (ScrollBar's standard model), track/page clicks, drag
// tracking, wheel routing, and offset clamping. Widgets keep only their
// content extent, content painting translated by the offsets, and a
// SetOnChange hook (invalidate caches / mark redraws).
//
// All Handle*/Paint coordinates are WIDGET-LOCAL, and bar geometry is
// laid out from sizes only (LayoutBars(w, h)) so a parent-relative
// FrameRect() can never leak into the local space.
type ScrollbarHost struct {
	VScroll *ScrollBar
	HScroll *ScrollBar

	ShowV bool
	ShowH bool

	// ScrollAmount is the wheel notch step in pixels.
	ScrollAmount float32

	// BarSize is the scrollbar strip width (theme default unless set).
	BarSize float32

	offX, offY               float32
	contentW, contentH       float32
	viewportW, viewportH     float32
	hasV, hasH               bool
	vRect, hRect, cornerRect graphics.Rect
	onChange                 func()
	owner                    meui.Widget
}

// SetOwner binds the bars to the widget they scroll. Bars are not
// children in the container tree (the widget paints them via
// PaintBars), but they need a parent so their own repaint requests —
// hover reveal and the overlay fade — reach the window. Call once from
// the widget constructor.
func (h *ScrollbarHost) SetOwner(w meui.Widget) {
	h.owner = w
	h.VScroll.SetParent(w)
	h.HScroll.SetParent(w)
}

// NewScrollbarHost creates a host with its two bars. Bars report value
// changes through SetOnChange.
func NewScrollbarHost() *ScrollbarHost {
	h := &ScrollbarHost{
		ShowV:        true,
		ShowH:        true,
		ScrollAmount: 20,
		BarSize:      theme.DefaultTheme.ScrollbarWidth,
	}
	h.VScroll = NewScrollBar()
	h.VScroll.SetOrientation(Vertical)
	h.VScroll.Min = 0
	h.VScroll.Max = 0
	h.VScroll.VisibleAmount = 0
	h.VScroll.OnValueChanged = func(val float32) {
		h.offY = val
		h.notify()
	}
	h.HScroll = NewScrollBar()
	h.HScroll.SetOrientation(Horizontal)
	h.HScroll.Min = 0
	h.HScroll.Max = 0
	h.HScroll.VisibleAmount = 0
	h.HScroll.OnValueChanged = func(val float32) {
		h.offX = val
		h.notify()
	}
	return h
}

// SetOnChange registers a callback fired after any offset change.
// Widgets use it to invalidate paint caches and mark redraws.
func (h *ScrollbarHost) SetOnChange(fn func()) { h.onChange = fn }

func (h *ScrollbarHost) notify() {
	if h.onChange != nil {
		h.onChange()
	}
}

// VisibleV / VisibleH report whether the bars are currently shown.
func (h *ScrollbarHost) VisibleV() bool { return h.hasV }
func (h *ScrollbarHost) VisibleH() bool { return h.hasH }

// Offsets returns the current scroll offsets in pixels.
func (h *ScrollbarHost) Offsets() (float32, float32) { return h.offX, h.offY }

// SetOffsets scrolls to (x, y), clamped to the scrollable range.
func (h *ScrollbarHost) SetOffsets(x, y float32) {
	h.clampOffsets(&x, &y)
	if x != h.offX || y != h.offY {
		h.offX, h.offY = x, y
		h.VScroll.Value = y
		h.HScroll.Value = x
		// Scrolling anywhere over the content reveals overlay bars
		// (macOS/GNOME reveal-on-scroll, not just on bar hover).
		h.VScroll.touch()
		h.HScroll.touch()
		h.notify()
	}
}

func (h *ScrollbarHost) clampOffsets(x, y *float32) {
	maxX := h.contentW - h.viewportW
	if maxX < 0 {
		maxX = 0
	}
	maxY := h.contentH - h.viewportH
	if maxY < 0 {
		maxY = 0
	}
	if *x < 0 {
		*x = 0
	}
	if *x > maxX {
		*x = maxX
	}
	if *y < 0 {
		*y = 0
	}
	if *y > maxY {
		*y = maxY
	}
}

// SetExtents updates content and viewport sizes and recomputes bar
// visibility, including the corner interplay (a visible vertical bar
// narrows the effective viewport, which may force the horizontal bar).
func (h *ScrollbarHost) SetExtents(contentW, contentH, viewW, viewH float32) {
	h.contentW, h.contentH = contentW, contentH
	h.viewportW, h.viewportH = viewW, viewH

	bar := h.BarSize
	hasV := h.ShowV && contentH > viewH
	hasH := h.ShowH && contentW > viewW
	if hasV {
		hasH = h.ShowH && contentW > viewW-bar
	}
	if hasH {
		hasV = h.ShowV && contentH > viewH-bar
	}
	if h.hasV != hasV || h.hasH != hasH {
		h.hasV, h.hasH = hasV, hasH
		h.notify()
	} else {
		h.hasV, h.hasH = hasV, hasH
	}

	// Sync bar ranges with the (possibly narrowed) effective viewport.
	effW, effH := viewW, viewH
	if hasV {
		effW -= bar
	}
	if hasH {
		effH -= bar
	}
	h.VScroll.Max = contentH
	h.VScroll.VisibleAmount = effH
	h.VScroll.SetValue(h.offY)
	h.HScroll.Max = contentW
	h.HScroll.VisibleAmount = effW
	h.HScroll.SetValue(h.offX)
	h.clampOffsets(&h.offX, &h.offY)
	h.VScroll.Value = h.offY
	h.HScroll.Value = h.offX
}

// ContentSize returns the viewport size inside a w x h local frame,
// excluding bar strips. Content painting clips to a rect of this size
// anchored at the widget's local origin.
func (h *ScrollbarHost) ContentSize(w, hgt float32) (float32, float32) {
	if h.hasV {
		w -= h.BarSize
	}
	if h.hasH {
		hgt -= h.BarSize
	}
	if w < 0 {
		w = 0
	}
	if hgt < 0 {
		hgt = 0
	}
	return w, hgt
}

// LayoutBars positions the bars inside a widget-LOCAL frame of size
// w x h (origin implied (0,0)). Call after SetExtents, before PaintBars.
//
// Bars are FLUSH with the frame edges: no end padding, no clearance gap.
// The thumb's rounded caps stay fully visible at both scroll extremes
// because the strip's ends sit inside whatever the widget clips to (a
// widget's frame no longer overflows its page — see the VStack/Flex
// padding fix — so the cap is tangent to the clip line, not past it).
//
// It takes sizes, never a rect with an origin, so a caller cannot
// accidentally pass FrameRect() (parent-relative): the bars always land
// in the same local space the widget's canvas and mouse events use.
func (h *ScrollbarHost) LayoutBars(w, hgt float32) {
	bar := h.BarSize
	h.vRect, h.hRect, h.cornerRect = graphics.Rect{}, graphics.Rect{}, graphics.Rect{}
	if h.hasV {
		h.vRect = graphics.NewRect(w-bar, 0, bar, hgt)
		if h.hasH {
			h.vRect.Height -= bar
		}
		h.VScroll.SetFrameRect(h.vRect)
	}
	if h.hasH {
		h.hRect = graphics.NewRect(0, hgt-bar, w, bar)
		if h.hasV {
			h.hRect.Width -= bar
		}
		h.HScroll.SetFrameRect(h.hRect)
	}
	if h.hasV && h.hasH {
		h.cornerRect = graphics.NewRect(w-bar, hgt-bar, bar, bar)
	}
}

// PaintBars paints the visible bars. Call after LayoutBars with the
// same frame — the bars are drawn at their laid-out positions in the
// current canvas space.
func (h *ScrollbarHost) PaintBars(canvas *graphics.Canvas) {
	if h.hasV {
		h.VScroll.Paint(canvas, h.vRect)
	}
	if h.hasH {
		h.HScroll.Paint(canvas, h.hRect)
	}
}

// inStrips reports whether local (x, y) is inside the vertical/horizontal
// bar strip of a w x h local frame.
func (h *ScrollbarHost) inStrips(w, hgt, x, y float32) (inV, inH bool) {
	cw, ch := h.ContentSize(w, hgt)
	inV = h.hasV && x >= cw && x <= w
	inH = h.hasH && y >= ch && y <= hgt
	return
}

// HandleMouseDown routes a widget-LOCAL press to the bars. Returns true
// if a bar consumed it. Coordinates are converted to bar-local (0-based
// at the bar's own frame) — the bars' hit math (thumbRect,
// setValueFrom*) is bar-local, not widget-local.
func (h *ScrollbarHost) HandleMouseDown(w, hgt, x, y float32, button int) bool {
	if inV, inH := h.inStrips(w, hgt, x, y); inV {
		return h.VScroll.OnMouseDown(x-h.vRect.X, y-h.vRect.Y, button)
	} else if inH {
		return h.HScroll.OnMouseDown(x-h.hRect.X, y-h.hRect.Y, button)
	}
	return false
}

// HandleMouseUp routes a widget-LOCAL release to the bars.
func (h *ScrollbarHost) HandleMouseUp(x, y float32, button int) {
	if h.hasV && h.VScroll.dragging {
		h.VScroll.OnMouseUp(x, y, button)
	}
	if h.hasH && h.HScroll.dragging {
		h.HScroll.OnMouseUp(x, y, button)
	}
}

// HandleMouseMove routes a widget-LOCAL move to the bars (thumb drags).
func (h *ScrollbarHost) HandleMouseMove(x, y float32) {
	if h.hasV && h.VScroll.dragging {
		h.VScroll.OnMouseMove(x-h.vRect.X, y-h.vRect.Y)
	}
	if h.hasH && h.HScroll.dragging {
		h.HScroll.OnMouseMove(x-h.hRect.X, y-h.hRect.Y)
	}
}

// UpdateHover reveals overlay bars whose strip the pointer is inside
// (widget-local coordinates). Overlay pills otherwise fade out while
// idle; hovering the strip must bring them back so they can be grabbed.
// Safe to call on every mouse move (SetHovered only redraws on change).
func (h *ScrollbarHost) UpdateHover(x, y float32) {
	if h.hasV {
		h.VScroll.SetHovered(rectContains(h.vRect, x, y))
	}
	if h.hasH {
		h.HScroll.SetHovered(rectContains(h.hRect, x, y))
	}
}

func rectContains(r graphics.Rect, x, y float32) bool {
	return r.Width > 0 && r.Height > 0 &&
		x >= r.X && x <= r.X+r.Width && y >= r.Y && y <= r.Y+r.Height
}

// HandleWheel applies a wheel notch to the offset. dy<0 scrolls down
// (toward the user). Returns true if any bar consumed it.
func (h *ScrollbarHost) HandleWheel(dx, dy float32) bool {
	if h.hasV && dy != 0 {
		h.SetOffsets(h.offX, h.offY-dy*h.ScrollAmount)
		return true
	}
	if h.hasH && dx != 0 {
		h.SetOffsets(h.offX-dx*h.ScrollAmount, h.offY)
		return true
	}
	return false
}

// BarRects exposes bar geometry for widgets painting bars at absolute
// positions (tests, overlays).
func (h *ScrollbarHost) BarRects() (v, hv, corner graphics.Rect) {
	return h.vRect, h.hRect, h.cornerRect
}
