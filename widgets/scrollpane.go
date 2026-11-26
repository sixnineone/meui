package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

// ScrollPane clips a single content widget and overlays scrollbars when
// the content exceeds the viewport. Scrollbar geometry, wheel, drag
// tracking and clamping are delegated to ScrollbarHost.
type ScrollPane struct {
	meui.BaseContainer
	*theme.ScrollPaneTheme
	Content       meui.Widget
	ContentWidth  float32
	ContentHeight float32
	// VScroll/HScroll alias the host's bars (same pointers) so external
	// code can inspect scrollbar state.
	VScroll     *ScrollBar
	HScroll     *ScrollBar
	ShowHScroll bool
	ShowVScroll bool
	// frameless paints no plate and no edge: the pane is a viewport onto its content,
	// which is what a web page inside a shell wants (the page owns its background).
	frameless bool
	dragging  bool
	// stickyY remembers each sticky descendant's position in the CONTENT's flow, so a
	// pass can tell "at its natural place" from "pinned". The pane owns it rather than
	// the widget: the widget's frame is what moves, so it cannot also be the record.
	stickyY map[meui.Widget]float32
	// hitTestContent lets the WINDOW's hit test see the content tree. Without it a
	// press anywhere in the viewport goes to the pane, so clickable children inside a
	// scrolling page (message rows, article links) are unreachable — the pane is a
	// viewport, not a control. Off by default: the enterprise panes rely on the pane
	// receiving presses for drag-scrolling, and the web ScrollView opts in.
	hitTestContent bool
	dragLastX      float32
	dragLastY      float32
	host           *ScrollbarHost
}

func NewScrollPane() *ScrollPane {
	sp := &ScrollPane{
		BaseContainer:   meui.BaseContainer{BaseWidget: *meui.NewBaseWidget()},
		ScrollPaneTheme: theme.DefaultScrollPaneTheme,
		ShowHScroll:     true,
		ShowVScroll:     true,
	}
	sp.stickyY = make(map[meui.Widget]float32)
	sp.host = NewScrollbarHost()
	sp.host.ShowH = true
	sp.host.ShowV = true
	sp.host.SetOnChange(sp.MarkForRedraw)
	sp.VScroll = sp.host.VScroll
	sp.HScroll = sp.host.HScroll
	sp.VScroll.SetParent(sp)
	sp.HScroll.SetParent(sp)
	return sp
}

// SetShowBars syncs the public policy flags with the host.
func (sp *ScrollPane) syncShowPolicies() {
	sp.host.ShowH = sp.ShowHScroll
	sp.host.ShowV = sp.ShowVScroll
}

func (sp *ScrollPane) SetContent(content meui.Widget) *ScrollPane {
	sp.Content = content
	if content != nil {
		content.SetParent(sp)
	}
	sp.SetNeedsLayout()
	sp.MarkForRedraw()
	return sp
}

// SetFrameless draws no background plate and no edge line: the pane is a viewport
// onto its content, which is what a web page inside a shell wants.
func (sp *ScrollPane) SetFrameless(on bool) *ScrollPane {
	sp.frameless = on
	sp.MarkForRedraw()
	return sp
}

func (sp *ScrollPane) SetContentSize(w, h float32) *ScrollPane {
	sp.ContentWidth = w
	sp.ContentHeight = h
	sp.SetNeedsLayout()
	sp.MarkForRedraw()
	return sp
}

func (sp *ScrollPane) ValidateLayout() {
	if !sp.NeedsLayout() {
		return
	}
	frame := sp.FrameRect()
	if frame.Width < 10 || frame.Height < 10 {
		sp.ClearNeedsLayout()
		return
	}

	sp.syncShowPolicies()

	// Effective content size: ContentWidth → content.FrameRect → viewport
	effW := sp.ContentWidth
	if effW == 0 && sp.Content != nil {
		effW = sp.Content.FrameRect().Width
	}
	if effW == 0 {
		effW = frame.Width
	}
	effH := sp.ContentHeight
	if effH == 0 && sp.Content != nil {
		effH = sp.Content.FrameRect().Height
	}
	if effH == 0 {
		effH = frame.Height
	}

	// Bars: extents, visibility, positions, offset clamping — all in the host.
	// Bar geometry is laid out from sizes in the widget's LOCAL space.
	sp.host.SetExtents(effW, effH, frame.Width, frame.Height)
	sp.host.LayoutBars(frame.Width, frame.Height)

	contentW, contentH := sp.host.ContentSize(frame.Width, frame.Height)
	contentRect := graphics.NewRect(0, 0, contentW, contentH)

	if sp.Content != nil {
		contentW := sp.ContentWidth
		contentH := sp.ContentHeight
		if contentW == 0 {
			contentW = sp.Content.FrameRect().Width
		}
		if contentW == 0 {
			contentW = contentRect.Width
		}
		if contentH == 0 {
			contentH = sp.Content.FrameRect().Height
		}
		if contentH == 0 {
			contentH = contentRect.Height
		}
		sp.Content.SetFrameRect(graphics.NewRect(0, 0, contentW, contentH))
		if vc, ok := sp.Content.(interface{ ValidateLayout() }); ok {
			vc.ValidateLayout()
		}
	}

	sp.ClearNeedsLayout()
}

func (sp *ScrollPane) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !sp.Visible() {
		return
	}

	canvas.Save()

	disabled := !sp.Enabled()

	if !sp.frameless {
		bgPaint := graphics.NewPaint()
		if disabled {
			bgPaint.SetColor(theme.DefaultTheme.DisabledBg)
		} else {
			bgPaint.SetColor(sp.ScrollPaneTheme.BackgroundInk)
		}
		canvas.DrawRoundedRect(rect, sp.ScrollPaneTheme.CornerRadius, sp.ScrollPaneTheme.CornerRadius, bgPaint)

		borderPaint := graphics.NewPaint()
		borderPaint.SetStyle(paintstyle.Stroke)
		borderPaint.SetStrokeWidth(theme.BorderStrokeWidth)
		if disabled {
			borderPaint.SetColor(theme.DefaultTheme.DisabledBorder)
		} else {
			borderPaint.SetColor(sp.ScrollPaneTheme.EdgeInk)
		}
		canvas.DrawRoundedRect(rect, sp.ScrollPaneTheme.CornerRadius, sp.ScrollPaneTheme.CornerRadius, borderPaint)
	}

	canvas.ClipRect(rect, pathop.Intersect, false)

	// Translate to ScrollPane's local coordinate system (top-left of ScrollPane is now 0,0)
	canvas.Translate(rect.X, rect.Y)

	// Keep bar geometry in sync with the current frame (cheap; host
	// dedupes notifications).
	sp.syncShowPolicies()
	sp.host.SetExtents(sp.effectiveContentW(rect), sp.effectiveContentH(rect), rect.Width, rect.Height)
	sp.host.LayoutBars(rect.Width, rect.Height)
	// Sticky boxes move with the scroll; do it before painting the content so this frame
	// and every later hit test see the same rectangles.
	sp.RefreshSticky()

	cw, ch := sp.host.ContentSize(rect.Width, rect.Height)
	contentRect := graphics.NewRect(0, 0, cw, ch)
	offX, offY := sp.host.Offsets()

	// Paint content
	if sp.Content != nil {
		canvas.Save()
		canvas.ClipRect(contentRect, pathop.Intersect, false)
		canvas.Translate(-offX, -offY)
		contentFrame := sp.Content.FrameRect()
		localContent := graphics.NewRect(0, 0, contentFrame.Width, contentFrame.Height)
		sp.Content.Paint(canvas, localContent)
		canvas.Restore()
	}

	// Paint scrollbars (host knows their laid-out rects)
	sp.host.PaintBars(canvas)

	canvas.Restore()
}

func (sp *ScrollPane) effectiveContentW(frame graphics.Rect) float32 {
	if sp.ContentWidth > 0 {
		return sp.ContentWidth
	}
	if sp.Content != nil && sp.Content.FrameRect().Width > 0 {
		return sp.Content.FrameRect().Width
	}
	return frame.Width
}

func (sp *ScrollPane) effectiveContentH(frame graphics.Rect) float32 {
	if sp.ContentHeight > 0 {
		return sp.ContentHeight
	}
	if sp.Content != nil && sp.Content.FrameRect().Height > 0 {
		return sp.Content.FrameRect().Height
	}
	return frame.Height
}

func (sp *ScrollPane) OnMouseDown(x, y float32, button int) bool {
	if !sp.Enabled() {
		return false
	}

	// Bars first (widget-local press).
	if fr := sp.FrameRect(); sp.host.HandleMouseDown(fr.Width, fr.Height, x, y, button) {
		return true
	}

	// Content area - route to content and set up drag scrolling
	if sp.Content != nil {
		offX, offY := sp.host.Offsets()
		localX := x + offX
		localY := y + offY
		contentFrame := sp.Content.FrameRect()
		if localX >= 0 && localX <= contentFrame.Width && localY >= 0 && localY <= contentFrame.Height {
			// Route click to content
			contentHandled := false
			if el, ok := sp.Content.(interface {
				OnMouseDown(float32, float32, int) bool
			}); ok {
				contentHandled = el.OnMouseDown(localX, localY, button)
			}
			// Route focus to content if it's focusable
			if fh, ok := sp.Content.(interface{ OnFocus() }); ok {
				fh.OnFocus()
			}
			// Only set up drag scrolling if content didn't handle the click
			if !contentHandled {
				sp.dragging = true
				sp.dragLastX = x
				sp.dragLastY = y
				return true
			}
			return contentHandled
		}
	}
	return false
}

func (sp *ScrollPane) OnMouseUp(x, y float32, button int) bool {
	sp.dragging = false
	sp.host.HandleMouseUp(x, y, button)
	if sp.Content != nil {
		offX, offY := sp.host.Offsets()
		if el, ok := sp.Content.(interface {
			OnMouseUp(float32, float32, int) bool
		}); ok {
			return el.OnMouseUp(x+offX, y+offY, button)
		}
	}
	return false
}

func (sp *ScrollPane) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if sp.Content != nil {
		if kh, ok := sp.Content.(interface {
			OnKeyDown(platform.KeyCode, platform.Modifiers) bool
		}); ok {
			return kh.OnKeyDown(keyCode, mod)
		}
	}
	return false
}

func (sp *ScrollPane) OnCharacter(ch rune) {
	if sp.Content != nil {
		if th, ok := sp.Content.(interface{ OnCharacter(rune) }); ok {
			th.OnCharacter(ch)
		}
	}
}

func (sp *ScrollPane) OnFocus() {
	if sp.Content != nil {
		if fh, ok := sp.Content.(interface{ OnFocus() }); ok {
			fh.OnFocus()
		}
	}
}

func (sp *ScrollPane) OnBlur() {
	if sp.Content != nil {
		if fh, ok := sp.Content.(interface{ OnBlur() }); ok {
			fh.OnBlur()
		}
	}
}

func (sp *ScrollPane) OnMouseMove(x, y float32) {
	sp.host.UpdateHover(x, y) // overlay pills reveal on strip hover
	// Drag scrolling
	if sp.dragging {
		dx := sp.dragLastX - x
		dy := sp.dragLastY - y
		sp.dragLastX = x
		sp.dragLastY = y
		offX, offY := sp.host.Offsets()
		sp.host.SetOffsets(offX+dx, offY+dy)
		return
	}

	sp.host.HandleMouseMove(x, y)

	if sp.Content != nil {
		offX, offY := sp.host.Offsets()
		if el, ok := sp.Content.(interface{ OnMouseMove(float32, float32) }); ok {
			el.OnMouseMove(x+offX, y+offY)
		}
	}
}

func (sp *ScrollPane) OnMouseWheel(x, y float32, dx, dy float32) bool {
	if !sp.Enabled() {
		return false
	}
	return sp.host.HandleWheel(dx, dy)
}

// SetHitTestContent makes the pane's content tree visible to hit testing, so a widget
// inside a scrolling viewport can be clicked. See the field's comment for why this is
// opt-in.
func (sp *ScrollPane) SetHitTestContent(on bool) *ScrollPane {
	sp.hitTestContent = on
	return sp
}

// HitTestsContent reports whether the content is hit-tested.
func (sp *ScrollPane) HitTestsContent() bool { return sp.hitTestContent }

func (sp *ScrollPane) PanelAt(x, y float32) meui.Widget {
	// Always return ScrollPane itself — scrollbars are handled internally
	// via OnMouseDown which correctly converts coordinates. Returning
	// scrollbars here gives them wrong local coordinates via PointFromRoot
	// (their FrameRect is in parent space, same Y as ScrollPane).
	if !sp.hitTestContent || sp.Content == nil {
		return sp
	}
	// A press on a visible bar belongs to the pane, which routes it. A bar is the pane's
	// only where it actually IS. Testing "x is past the bar's left
	// edge" swallowed the whole viewport while the bars' frames were still zero (before
	// the first layout), which hid every clickable child behind the pane.
	if sp.host != nil && sp.host.VisibleV() && sp.VScroll != nil {
		if r := sp.VScroll.FrameRect(); r.Width > 0 && r.Height > 0 &&
			x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height {
			return sp
		}
	}
	if sp.host != nil && sp.host.VisibleH() && sp.HScroll != nil {
		if r := sp.HScroll.FrameRect(); r.Width > 0 && r.Height > 0 &&
			x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height {
			return sp
		}
	}
	offX, offY := sp.host.Offsets()
	cx, cy := x+offX, y+offY
	// The CONTENT's extent, not the viewport's: host.ContentSize is the usable viewport
	// (frame minus bars), and comparing a content-space point against it rejected every
	// click below the fold — the whole list was unclickable past the first screen.
	cf := sp.Content.FrameRect()
	if cx < 0 || cy < 0 || cx > cf.Width || cy > cf.Height {
		return sp
	}
	if container, ok := sp.Content.(meui.Container); ok {
		if deeper := container.PanelAt(cx, cy); deeper != nil && deeper != sp.Content {
			return deeper
		}
	}
	return sp
}

func (sp *ScrollPane) Children() []meui.Widget {
	// Only return content — scrollbars are painted and hit-tested internally
	// by the host's bar routing.
	if sp.Content != nil {
		return []meui.Widget{sp.Content}
	}
	return nil
}

func (sp *ScrollPane) AddChild(child meui.Widget) {
	sp.SetContent(child)
}

// RefreshSticky pins every sticky descendant against the current scroll offset. Paint
// calls it, so the frames a hit test sees are the ones that were drawn; call it directly
// only when something needs the frames before the next frame (a click that arrives in the
// same tick as a programmatic scroll).
func (sp *ScrollPane) RefreshSticky() {
	if sp.Content == nil {
		return
	}
	off := sp.ScrollY()
	sp.layoutSticky(sp.Content, 0, 0, off)
}

// layoutSticky walks the content tree, moving each sticky box's frame to the viewport top
// (plus its inset) once the scroll is past its natural position, and clamping it to its
// parent's bottom. MOVING the frame is the point: painting, the hit test, hover and the
// cursor walk all read frames, so a paint-time translate would leave input behind.
func (sp *ScrollPane) layoutSticky(w meui.Widget, originX, originY, off float32) {
	container, ok := w.(interface{ Children() []meui.Widget })
	if !ok {
		return
	}
	fr := w.FrameRect()
	for _, ch := range container.Children() {
		cfr := ch.FrameRect()
		absX := originX + cfr.X
		absY := originY + cfr.Y
		if st, ok := ch.(interface {
			Sticky() bool
			StickyInset() float32
		}); ok && st.Sticky() {
			// Below the pin line it is where the flow PUT it — unless it is sitting on its
			// section's bottom clamp, which is where the last pass left it.
			clampAbs := originY + fr.Height - cfr.Height
			if absY > off+st.StickyInset() && absY < clampAbs {
				sp.stickyY[ch] = absY
			}
			natural, known := sp.stickyY[ch]
			if !known {
				natural = absY
				sp.stickyY[ch] = natural
			}
			target := off + st.StickyInset()
			if target < natural {
				target = natural
			}
			// Never past its section: the parent's bottom in content space.
			if target > clampAbs {
				target = clampAbs
			}
			// target is in CONTENT space; the frame is in the parent's.
			localTarget := target - originY
			if localTarget != cfr.Y {
				ch.SetFrameRect(graphics.NewRect(cfr.X, localTarget, cfr.Width, cfr.Height))
				cfr = ch.FrameRect()
				absY = originY + cfr.Y
			}
			absX, absY = originX+cfr.X, originY+cfr.Y
		}
		sp.layoutSticky(ch, absX, absY, off)
	}
}

// ScrollTo scrolls so content y is at the viewport's top edge, clamped by the host's
// extents. It is how "choose a row and the list reveals it" is written: ScrollTo the
// row's top, or the bottom minus the viewport when the row is below the fold.
func (sp *ScrollPane) ScrollTo(y float32) *ScrollPane {
	// Make sure the extents exist before clamping: a reveal can run before the pane's
	// first paint (a scene that selects a row as it loads), and clamping against zero
	// extents silently does nothing.
	fr := sp.FrameRect()
	sp.host.SetExtents(sp.effectiveContentW(fr), sp.effectiveContentH(fr), fr.Width, fr.Height)
	sp.host.LayoutBars(fr.Width, fr.Height)
	_, offY := sp.host.Offsets()
	sp.host.SetOffsets(0, y)
	if _, n := sp.host.Offsets(); n != offY {
		sp.MarkForRedraw()
	}
	return sp
}

// ScrollY is the current vertical offset.
func (sp *ScrollPane) ScrollY() float32 {
	_, y := sp.host.Offsets()
	return y
}

// ViewportHeight is the visible content height (the pane minus its scrollbars).
func (sp *ScrollPane) ViewportHeight() float32 {
	_, vh := sp.host.ContentSize(sp.ContentWidth, sp.ContentHeight)
	return vh
}

// ScrollOffsets implements the scrollAdjuster interface used by Panel.PointToRoot.
func (sp *ScrollPane) ScrollOffsets() (float32, float32) {
	return sp.host.Offsets()
}

func (sp *ScrollPane) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "contentWidth":
		if f, ok := asFloat(value); ok {
			sp.SetContentSize(f, sp.ContentHeight)
		}
	case "contentHeight":
		if f, ok := asFloat(value); ok {
			sp.SetContentSize(sp.ContentWidth, f)
		}
	case "horizontalScrollBarPolicy":
		if s, ok := value.(string); ok {
			sp.ShowHScroll = s != "never"
			sp.syncShowPolicies()
		}
	case "verticalScrollBarPolicy":
		if s, ok := value.(string); ok {
			sp.ShowVScroll = s != "never"
			sp.syncShowPolicies()
		}
	}
}

// asFloat coerces numeric QML property values (int, float64) to float32.
func asFloat(v interface{}) (float32, bool) {
	switch n := v.(type) {
	case int:
		return float32(n), true
	case float64:
		return float32(n), true
	case float32:
		return n, true
	}
	return 0, false
}
