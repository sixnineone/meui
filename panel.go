package meui

import (
	"fmt"
	"os"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/layout"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

type Panel struct {
	BaseContainer
	padding         float32
	layout          interface{}
	background      interface{}
	backgroundToken string
	borderColor     interface{}
	borderWidth     float32

	// The box appearance: what turns a Panel into a web surface. Radius, shadow,
	// gradient, clipping and opacity used to be parsed out of QML's Rectangle and
	// thrown away, which is why every web-looking scene came out as flat squares.
	radius       float32
	shadow       theme.ShadowSpec
	hasShadow    bool
	gradient     *graphics.Gradient
	gradientUnit bool
	clipChildren bool
	opacity      float32
	backdropBlur bool
	// sticky pins the box to the top of its ScrollPane while the scroll is past its
	// natural position, clamped by its parent section's bottom — the web's
	// `position: sticky`. The PANE moves the frame (see layoutSticky), so painting, the
	// hit test and hover all see the pinned rectangle; the box only carries the flags.
	sticky      bool
	stickyInset float32
	stickyY     float32 // natural content-space y, captured on first layout
	stickyInit  bool    // 0 => 1

	// The interaction appearance: a clickable box owns hover, press, focus, a
	// cursor and Enter/Space activation, and paints its state from the theme's
	// Hovered/Pressed inks. One state machine (ClickBehaviour) serves this and
	// MouseArea so a click cannot mean two things.
	clickable   bool
	selected    bool
	click       ClickBehaviour
	cursor      *platform.Cursor
	hoverInk    graphics.Color
	pressedInk  graphics.Color
	selectedInk graphics.Color
	// hoverAnim/pressAnim fade the state washes instead of switching them. The theme's
	// Motion tokens own the duration; a box in a test or a reduce-motion app can Snap.
	hoverAnim Animation
	pressAnim Animation

	DrawCallback         func(canvas *graphics.Canvas, rect graphics.Rect)
	MouseDownCallback    func(where graphics.Point, button, clickCount int, mod platform.Modifiers) bool
	MouseUpCallback      func(where graphics.Point, button int, mod platform.Modifiers) bool
	MouseMoveCallback    func(where graphics.Point, mod platform.Modifiers) bool
	MouseDragCallback    func(where graphics.Point, button int, mod platform.Modifiers) bool
	MouseWheelCallback   func(where, delta graphics.Point, mod platform.Modifiers) bool
	UpdateCursorCallback func(where graphics.Point) *platform.Cursor
	RuneTypedCallback    func(ch rune) bool
	focusable            bool
}

func NewPanel() *Panel {
	return &Panel{
		BaseContainer: BaseContainer{BaseWidget: *NewBaseWidget()},
		padding:       0,
	}
}

// EffectiveBackground resolves the panel background for the current
// theme: the token when one is set (live), otherwise the stored colour.
func (p *Panel) EffectiveBackground() (graphics.Color, bool) {
	if p.backgroundToken != "" {
		if c, ok := theme.TokenColor(p.backgroundToken); ok {
			return c, true
		}
	}
	if c, ok := p.background.(graphics.Color); ok {
		return c, true
	}
	return 0, false
}

// SetBackgroundToken pins the panel background to a theme role so
// runtime theme switches recolour it.
func (p *Panel) SetBackgroundToken(name string) *Panel {
	p.backgroundToken = name
	return p
}

func (p *Panel) SetBackground(bg interface{}) *Panel {
	p.background = bg
	p.MarkForRedraw()
	return p
}

func (p *Panel) SetEnabled(enabled bool) *Panel {
	p.BaseWidget.SetEnabled(enabled)
	for _, child := range p.children {
		if s, ok := child.(interface{ SetEnabled(bool) }); ok {
			s.SetEnabled(enabled)
		}
	}
	p.MarkForRedraw()
	return p
}

func (p *Panel) SetPadding(padding float32) *Panel {
	p.padding = padding
	p.MarkForRedraw()
	return p
}

// SetFrameRect overrides BaseWidget to trigger layout on resize.
func (p *Panel) SetFrameRect(rect graphics.Rect) {
	p.BaseWidget.SetFrameRect(rect)
	p.needsLayout = true
}

// SetFocusable toggles whether this panel can receive keyboard focus.
func (p *Panel) SetFocusable(b bool) { p.focusable = b }

// Focusable reports whether the panel can take the keyboard. A clickable box can:
// a div with a click handler that a keyboard cannot reach is not a control.
func (p *Panel) Focusable() bool { return p.focusable || p.clickable }

// ── Box appearance ──────────────────────────────────────────────────────

// SetCornerRadius rounds the panel's background, border and shadow together, so
// the three can never disagree about the shape. It is QML's Rectangle radius.
func (p *Panel) SetCornerRadius(r float32) *Panel {
	if r < 0 {
		r = 0
	}
	p.radius = r
	p.MarkForRedraw()
	return p
}

// Radius returns the current corner radius.
func (p *Panel) Radius() float32 { return p.radius }

// SetShadow paints a drop shadow behind the panel from an explicit spec. The
// token route (SetShadowToken) is preferred in a theme-driven app.
func (p *Panel) SetShadow(spec theme.ShadowSpec) *Panel {
	p.shadow = spec
	p.hasShadow = spec.Color != 0
	p.MarkForRedraw()
	return p
}

// SetShadowToken resolves a named shadow ("card", "popup", "window", "none") from
// the ACTIVE theme, so a scene that says `shadow: "card"` follows the preset.
func (p *Panel) SetShadowToken(name string) *Panel {
	spec, ok := ShadowToken(name)
	p.shadow = spec
	p.hasShadow = ok
	p.MarkForRedraw()
	return p
}

// SetGradient fills the panel with a gradient instead of a solid colour. The
// background colour still applies when no gradient is set.
func (p *Panel) SetGradient(g graphics.Gradient) *Panel {
	gg := g
	p.gradient = &gg
	p.MarkForRedraw()
	return p
}

// SetGradientUnit fills the panel with a gradient whose geometry is expressed in
// 0..1 of the box, so a scene can say "top to bottom" without knowing the size the
// layout will give it. Paint resolves it against the final rect.
func (p *Panel) SetGradientUnit(g graphics.Gradient) *Panel {
	gg := g
	p.gradient = &gg
	p.gradientUnit = true
	p.MarkForRedraw()
	return p
}

// ApplyOpacity is the non-fluent form the QML loader's common-property pass uses
// (a fluent setter returning *Panel does not satisfy interface{ ApplyOpacity(float32) }).
func (p *Panel) ApplyOpacity(a float32) { p.SetOpacity(a) }

// ClearGradient returns to the solid background.
func (p *Panel) ClearGradient() *Panel {
	p.gradient = nil
	p.MarkForRedraw()
	return p
}

// SetClipChildren clips children to this panel's (rounded) rect — what a web card
// with an image needs, and what `clip: true` means in QML.
func (p *Panel) SetClipChildren(b bool) *Panel {
	p.clipChildren = b
	p.MarkForRedraw()
	return p
}

// SetOpacity sets the panel's own paint opacity (0..1). It applies to the panel's
// fill, border, shadow and state wash; children keep their own opacity.
func (p *Panel) SetOpacity(a float32) *Panel {
	if a < 0 {
		a = 0
	}
	if a > 1 {
		a = 1
	}
	p.opacity = a
	p.MarkForRedraw()
	return p
}

// ── Interaction ─────────────────────────────────────────────────────────

// SetClickable makes the box an input target: it tracks hover and press, paints
// them, points the cursor at itself and takes Enter/Space when focused. Its
// children still get the press first — a card containing a button must let the
// button win — and only a press that no child consumed clicks the box.
func (p *Panel) SetClickable(b bool) *Panel {
	p.clickable = b
	p.MarkForRedraw()
	return p
}

// Clickable reports whether the box is an input target.
func (p *Panel) Clickable() bool { return p.clickable }

// SetOnClick installs the box's click handler. Setting a handler also makes the
// box clickable, because a handler nothing can call is the failure this tree keeps
// finding.
func (p *Panel) SetOnClick(fn func()) *Panel {
	p.click.OnClick = fn
	if fn != nil {
		p.clickable = true
	}
	p.MarkForRedraw()
	return p
}

// Pressed reports whether the pointer is down on this box (the visual press state).
func (p *Panel) Pressed() bool { return p.click.Pressed() }

// SetPushed drives the pressed visual from outside (a MouseArea forwarding its
// state, a drag that began elsewhere, a test).
func (p *Panel) SetPushed(v bool) *Panel {
	if p.click.pressed != v {
		p.click.pressed = v
		p.MarkForRedraw()
	}
	return p
}

// SetSticky pins the box to the top of its scroll viewport while its section is in
// view. The scroll pane positions it; the box just declares it.
func (p *Panel) SetSticky(on bool) *Panel {
	p.sticky = on
	p.stickyInit = false
	p.MarkForRedraw()
	return p
}

// Sticky reports whether the box pins while scrolling.
func (p *Panel) Sticky() bool { return p.sticky }

// SetStickyInset is how far below the viewport's top edge a sticky box pins (useful
// under a fixed bar the pane does not know about).
func (p *Panel) SetStickyInset(px float32) *Panel {
	if px < 0 {
		px = 0
	}
	p.stickyInset = px
	return p
}

// StickyInset reports the pin offset.
func (p *Panel) StickyInset() float32 { return p.stickyInset }

// stickyNaturalY reports the box's position in the content's flow. Captured once, when
// the pane first lays the sticky box out.
func (p *Panel) stickyNaturalY() (float32, bool) { return p.stickyY, p.stickyInit }

// captureStickyY records the natural y the first time the pane sees the box.
func (p *Panel) captureStickyY(y float32) float32 {
	if !p.stickyInit {
		p.stickyY = y
		p.stickyInit = true
	}
	return p.stickyY
}

// SetBackdropBlur frosts the box: what is already painted behind it is blurred with the
// renderer's kernel, then the box's own translucent fill and border go on top. It is the
// frosted-glass card, and it is deliberately a BOOLEAN — the blur strength is the
// renderer's (one kernel on both sides); a per-box radius would drift from GL.
func (p *Panel) SetBackdropBlur(on bool) *Panel {
	p.backdropBlur = on
	p.MarkForRedraw()
	return p
}

// BackdropBlur reports whether the box frosts its backdrop.
func (p *Panel) BackdropBlur() bool { return p.backdropBlur }

// SetSelected marks the box as the chosen one in its group — a nav item, a day cell, a
// list row. It paints the theme's Selected ink UNDER the hover/press wash, so the
// selected state survives a hover instead of being replaced by it.
func (p *Panel) SetSelected(selected bool) *Panel {
	if p.selected == selected {
		return p
	}
	p.selected = selected
	p.MarkForRedraw()
	return p
}

// Selected reports the selection flag.
func (p *Panel) Selected() bool { return p.selected }

// SetSelectedInk overrides the theme's Selected role for this box; zero uses the theme.
func (p *Panel) SetSelectedInk(c graphics.Color) *Panel {
	p.selectedInk = c
	p.MarkForRedraw()
	return p
}

// SetOnPress installs the press-state handler (down/up), which a scene uses when
// it wants a custom pressed visual rather than the theme wash.
func (p *Panel) SetOnPress(fn func(bool)) *Panel {
	p.click.OnPress = fn
	if fn != nil {
		p.clickable = true
	}
	return p
}

// SetCursor forces a cursor over the box. Nil falls back to the pointing hand
// when the box is clickable.
func (p *Panel) SetCursor(c *platform.Cursor) *Panel {
	p.cursor = c
	return p
}

// SetHoverInk overrides the theme's hover wash for this box; zero uses the
// theme's Hovered role.
func (p *Panel) SetHoverInk(c graphics.Color) *Panel {
	p.hoverInk = c
	p.MarkForRedraw()
	return p
}

// SetPressedInk overrides the theme's press wash; zero uses the theme's Pressed
// role.
func (p *Panel) SetPressedInk(c graphics.Color) *Panel {
	p.pressedInk = c
	p.MarkForRedraw()
	return p
}

// Window returns the underlying OS window. In the mobilegles
// migration this is a no-op stub; the return type is an interface
// to allow a real implementation later without changing call sites.
func (p *Panel) Window() *OSWindow { return nil }

// OSWindow is a stub for the underlying OS-level window concept.
// The shim does not currently use it; methods are stubs so call
// sites that retrieve it compile and run as no-ops.
type OSWindow struct{}

func (*OSWindow) SetContentRect(graphics.Rect) {}
func (*OSWindow) SetTitle(string)              {}
func (*OSWindow) Close()                       {}

// ContentRect returns the content rectangle in screen coordinates.
// Stub: returns zero rect until a backend provides one.
func (*OSWindow) ContentRect() graphics.Rect {
	return graphics.Rect{}
}

// Position returns the window's screen position. Stub.
func (*OSWindow) Position() graphics.Point { return graphics.Point{} }

// Size returns the window's screen size. Stub.
func (*OSWindow) Size() graphics.Size { return graphics.Size{} }

func (p *Panel) SetLayout(l interface{}) *Panel {
	p.layout = l
	p.needsLayout = true
	p.MarkForRedraw()
	return p
}

func (p *Panel) Layout() interface{} {
	return p.layout
}

func (p *Panel) Padding() float32 {
	return p.padding
}

// AddChild appends a child and sets its parent to this Panel (not the embedded BaseContainer).
func (p *Panel) AddChild(child Widget) {
	for _, existing := range p.children {
		if existing == child {
			return
		}
	}
	p.children = append(p.children, child)
	if child != nil {
		child.SetParent(p)
	}
	p.needsLayout = true
	p.MarkForRedraw()
}

func (p *Panel) Clear() {
	p.ClearChildren()
	p.MarkForRedraw()
}

// RemoveChild removes a child widget and cleans up its references.
func (p *Panel) RemoveChild(child Widget) bool {
	for i, c := range p.children {
		if c == child {
			if child != nil {
				if d, ok := child.(interface{ Dispose() }); ok {
					d.Dispose()
				}
				child.SetParent(nil)
			}
			p.children = append(p.children[:i], p.children[i+1:]...)
			p.needsLayout = true
			p.MarkForRedraw()
			return true
		}
	}
	return false
}

// Dispose cleans up all children and resources.
func (p *Panel) Dispose() {
	p.Clear()
	p.background = nil
	p.borderColor = nil
}

// PanelAt returns the deepest child widget containing the point (in LOCAL coordinates).
func (p *Panel) PanelAt(x, y float32) Widget {
	if hit := p.DefaultPanelAt(x, y); hit != nil {
		return hit
	}
	return p
}

// ValidateLayout applies the layout if needed.
func (p *Panel) ValidateLayout() {
	if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
		fmt.Printf("[validateLayout] enter children=%d\n", len(p.children))
	}
	timer := StartPerfTimer(fmt.Sprintf("ValidateLayout(%s)", getDebugName(p)))
	defer timer.Stop()

	frame := p.FrameRect()
	padding := p.padding
	// innerRect is in PANEL-LOCAL space (origin at 0,0 relative to this panel)
	innerRect := graphics.NewRect(
		padding,
		padding,
		frame.Width-padding*2,
		frame.Height-padding*2,
	)

	// Apply anchors only when no layout manager is set
	if p.layout == nil {
		anchorLayout := layout.NewAnchorLayout()
		anchorLayout.Layout(p, innerRect)
	}

	if p.layout != nil && len(p.children) > 0 {
		if l, ok := p.layout.(interface {
			Layout(interface{}, graphics.Rect)
		}); ok {
			if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
				fmt.Printf("[validateLayout] calling layout %T\n", l)
			}
			l.Layout(p, innerRect)
			if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
				fmt.Printf("[validateLayout] layout returned\n")
			}
		}

		// Validate layout results
		if issues := ValidateLayout(p, innerRect); len(issues) > 0 {
			for _, issue := range issues {
				LogEventf(getDebugName(p), "Layout validation: %s", issue)
			}
		}
	}

	if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
		fmt.Printf("[validateLayout] exit\n")
	}
	// Now recurse to children (after they've been positioned by parent layout)
	for _, child := range p.children {
		if vc, ok := child.(interface{ ValidateLayout() }); ok {
			vc.ValidateLayout()
		}
	}

	p.needsLayout = false
}

func (p *Panel) SetContentWidth(width float32) {
	p.SetWidth(width)
}

func (p *Panel) SetContentHeight(height float32) {
	p.SetHeight(height)
}

func (p *Panel) SetBorderColor(color interface{}) {
	p.borderColor = color
	p.MarkForRedraw()
}

func (p *Panel) SetBorderWidth(width float32) {
	p.borderWidth = width
	p.MarkForRedraw()
}

func (p *Panel) GetContentWidth() float32 {
	return p.Width()
}

func (p *Panel) GetContentHeight() float32 {
	return p.Height()
}

func (p *Panel) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "width":
		if f, ok := value.(float64); ok {
			p.SetWidthDirect(float32(f))
		}
	case "height":
		if f, ok := value.(float64); ok {
			p.SetHeightDirect(float32(f))
		}
	}
}

// PreferredDimensions returns the preferred size of the panel based on its children.
func (p *Panel) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	if p.layout == nil || len(p.children) == 0 {
		return p.Width(), p.Height()
	}
	if measurer, ok := p.layout.(interface {
		PreferredDimensions(interface{}, float32, float32) (float32, float32)
	}); ok {
		return measurer.PreferredDimensions(p, w, h)
	}
	return p.Width(), p.Height()
}

func (p *Panel) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
		fmt.Printf("[panelPaint] children=%d visible=%v needsLayout=%v rect=(%.0f,%.0f %.0fx%.0f)\n",
			len(p.children), p.Visible(), p.NeedsLayout(), rect.X, rect.Y, rect.Width, rect.Height)
	}
	timer := StartPerfTimer(fmt.Sprintf("Paint(%s)", getDebugName(p)))
	defer timer.Stop()

	if !p.Visible() {
		return
	}

	canvas.Save()

	if p.NeedsLayout() {
		p.ValidateLayout()
		p.ClearNeedsLayout()
	}

	op := p.opacity
	if op <= 0 {
		op = 1
	}

	// Frost BEFORE anything the box itself paints: the blur samples the backdrop, and a
	// fill painted first would be blurred instead of the thing behind it (which is
	// exactly what happened when this call sat below the fill: the frosted card came out
	// as the bare backdrop, invisible).
	if p.backdropBlur {
		canvas.BlurBackdrop(rect, p.radius)
	}

	// Shadow first: the fill covers its interior and leaves the fringe.
	if p.hasShadow {
		spec := p.shadow
		if op < 1 {
			a := uint32(float32((spec.Color>>24)&0xFF) * op)
			spec.Color = graphics.Color(a<<24) | spec.Color&0x00FFFFFF
		}
		DrawShadow(canvas, rect, p.radius, spec)
	}

	switch {
	case p.gradient != nil:
		bgPaint := graphics.NewPaint()
		g := *p.gradient
		if p.gradientUnit {
			g = ScaleGradient(g, rect)
		}
		bgPaint.SetGradient(g)
		bgPaint.SetGlobalAlpha(op)
		canvas.DrawRoundedRect(rect, p.radius, p.radius, bgPaint)
	default:
		if bgColor, ok := p.EffectiveBackground(); ok {
			bgPaint := graphics.NewPaint()
			bgPaint.SetColor(bgColor)
			bgPaint.SetGlobalAlpha(op)
			canvas.DrawRoundedRect(rect, p.radius, p.radius, bgPaint)
		}
	}

	// A SELECTED wash first: it belongs to the surface's state, so hover and press
	// layer on top of it rather than replacing it.
	if p.selected {
		ink := p.selectedInk
		if ink == 0 && theme.DefaultTheme != nil {
			ink = theme.DefaultTheme.SelectedWash
			if ink == 0 {
				ink = theme.DefaultTheme.Selected
			}
		}
		if ink != 0 {
			sel := graphics.NewPaint()
			sel.SetColor(ink)
			if op < 1 {
				sel.SetGlobalAlpha(op)
			}
			canvas.DrawRoundedRect(rect, p.radius, p.radius, sel)
		}
	}

	// The hover/press washes sit on top of the fill and under the border, so a bordered
	// card keeps its edge while the surface responds to the pointer. They CROSSFADE: the
	// hover wash fades out as the press wash fades in, so a press never doubles the ink
	// and a leave animates instead of snapping.
	if p.clickable && p.Enabled() {
		hov, prs := p.hoverAnim.Value(), p.pressAnim.Value()
		if hov > 0 && prs < 1 {
			ink := p.hoverInk
			if ink == 0 && theme.DefaultTheme != nil {
				ink = theme.DefaultTheme.Hovered
			}
			p.paintWash(canvas, rect, ink, op*hov*(1-prs))
		}
		if prs > 0 {
			ink := p.pressedInk
			if ink == 0 && theme.DefaultTheme != nil {
				ink = theme.DefaultTheme.Pressed
			}
			p.paintWash(canvas, rect, ink, op*prs)
		}
	}

	if p.borderWidth > 0 && p.borderColor != nil {
		borderPaint := graphics.NewPaint()
		borderPaint.SetStyle(paintstyle.Stroke)
		borderPaint.SetStrokeWidth(p.borderWidth)
		if color, ok := p.borderColor.(graphics.Color); ok {
			borderPaint.SetColor(color)
			borderPaint.SetGlobalAlpha(op)
			canvas.DrawRoundedRect(rect, p.radius, p.radius, borderPaint)
		}
	}

	if p.clickable && p.Focused() {
		DrawFocusRing(canvas, rect, p.radius)
	}

	// A custom painter draws INSIDE the box, over its fill and border and under its
	// children. It is the escape hatch a scene uses for chrome the widget set does
	// not have (a play glyph, a sparkline, a rule with notches) without inventing a
	// widget — and it finally RUNS: the field existed and nothing called it, the
	// "capability nobody switched on" this tree keeps finding.
	if p.DrawCallback != nil {
		p.DrawCallback(canvas, rect)
	}

	if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
		fmt.Printf("[panelPaint] reaching PaintChildren, children=%d\n", len(p.children))
	}
	if p.clipChildren {
		canvas.WithClip(rect, func(c *graphics.Canvas) { p.PaintChildren(c) })
	} else {
		p.PaintChildren(canvas)
	}

	canvas.Restore()
}

// paintWash draws one state wash at the given alpha (already multiplied by the box's
// opacity and the animation's progress). A zero ink or alpha paints nothing.
func (p *Panel) paintWash(canvas *graphics.Canvas, rect graphics.Rect, ink graphics.Color, alpha float32) {
	if ink == 0 || alpha <= 0.001 {
		return
	}
	wash := graphics.NewPaint()
	wash.SetColor(ink)
	wash.SetGlobalAlpha(alpha)
	canvas.DrawRoundedRect(rect, p.radius, p.radius, wash)
}

// SetHoveredState starts the hover wash's transition. The window drives it through the
// hover chain, so a box lights up and fades out the same way whether the pointer is on
// its face or on a label inside it.
func (p *Panel) SetHoveredState(hovered bool) {
	if p.Hovered() == hovered {
		return
	}
	p.BaseWidget.SetHoveredState(hovered)
	target := float32(0)
	if hovered {
		target = 1
	}
	p.hoverAnim.Set(target, p.motion().Hover)
	p.MarkForRedraw()
}

// FinishAnimations snaps every state transition to its target. Tests and
// reduce-motion apps call it; the window's loop keeps painting until they settle.
func (p *Panel) FinishAnimations() {
	p.finishOne(&p.hoverAnim, p.Hovered())
}

func (p *Panel) finishOne(a *Animation, on bool) {
	if on {
		a.Snap(1)
		return
	}
	a.Snap(0)
}

// motion is the active theme's motion tokens, or zero timings when no theme is installed
// (an uninstalled theme means no animation, not a panic).
func (p *Panel) motion() theme.Motion {
	if theme.DefaultTheme == nil {
		return theme.Motion{}
	}
	return theme.DefaultTheme.Motion
}

// stateWash returns the hover/press ink for the current pointer state: an explicit
// per-box override when set, otherwise the theme's Hovered/Pressed role. Zero means
// "no wash".
func (p *Panel) stateWash() graphics.Color {
	if p.click.Pressed() {
		if p.pressedInk != 0 {
			return p.pressedInk
		}
		if theme.DefaultTheme != nil {
			return theme.DefaultTheme.Pressed
		}
		return 0
	}
	if p.Hovered() {
		if p.hoverInk != 0 {
			return p.hoverInk
		}
		if theme.DefaultTheme != nil {
			return theme.DefaultTheme.Hovered
		}
	}
	return 0
}

// HoverAt is the position-based hover walk (see HoverTarget): a clickable box
// lights up whether the pointer is over its own face or over a label child that is
// the actual hit.
func (p *Panel) HoverAt(x, y float32) {
	if p.clickable {
		p.SetHoveredState(p.ContainsPoint(x, y))
	}
}

// CursorAt points at a clickable box. An explicit cursor wins; returning nil lets
// the walk fall through to the parent.
func (p *Panel) CursorAt(x, y float32) *platform.Cursor {
	if p.cursor != nil {
		return p.cursor
	}
	if p.clickable && p.Enabled() {
		return platform.PointingCursor()
	}
	return nil
}

// OnKeyDown activates a focused clickable box with Enter or Space, which is what a
// keyboard user means by "click the card".
func (p *Panel) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if !p.clickable || !p.Focused() {
		return false
	}
	switch keyCode {
	case platform.KeyReturn, platform.KeyNumPadEnter, platform.KeySpace:
		if p.click.Activate(p.Enabled()) {
			p.MarkForRedraw()
			return true
		}
	}
	return false
}

func (p *Panel) OnMouseDown(x, y float32, button int) bool {
	LogEventf(getDebugName(p), "OnMouseDown at (%.0f,%.0f) button=%d", x, y, button)
	// Children first: a clickable card must let a real control inside it win.
	if p.DispatchMouseDown(x, y, button) {
		return true
	}
	if p.clickable && p.ContainsPoint(x, y) && p.click.Down(p.Enabled(), true, button) {
		p.pressAnim.Set(1, p.motion().Press)
		p.MarkForRedraw()
		return true
	}
	return false
}

// OnBubbledMouseDown handles a press that reached this panel because a CHILD was
// the deepest hit and declined it (see Window.handleMouseDown). It deliberately does
// NOT dispatch to children: they have already been asked.
func (p *Panel) OnBubbledMouseDown(x, y float32, button int) bool {
	if !p.clickable || !p.ContainsPoint(x, y) {
		return false
	}
	if p.click.Down(p.Enabled(), true, button) {
		p.pressAnim.Set(1, p.motion().Press)
		p.MarkForRedraw()
		return true
	}
	return false
}

// OnBubbledMouseUp is the release half of OnBubbledMouseDown.
func (p *Panel) OnBubbledMouseUp(x, y float32, button int) bool {
	if !p.clickable {
		return false
	}
	if p.click.Up(p.Enabled(), p.ContainsPoint(x, y), button) {
		p.pressAnim.Set(0, p.motion().Press)
		p.MarkForRedraw()
		return true
	}
	return false
}

func (p *Panel) OnMouseUp(x, y float32, button int) bool {
	LogEventf(getDebugName(p), "OnMouseUp at (%.0f,%.0f) button=%d", x, y, button)
	if p.DispatchMouseUp(x, y, button) {
		return true
	}
	if p.clickable && p.click.Up(p.Enabled(), p.ContainsPoint(x, y), button) {
		p.pressAnim.Set(0, p.motion().Press)
		p.MarkForRedraw()
		return true
	}
	return false
}

func (p *Panel) OnMouseMove(x, y float32) {
	LogEventf(getDebugName(p), "OnMouseMove at (%.0f,%.0f)", x, y)
	p.DispatchMouseMove(x, y)
}

type scrollAdjuster interface {
	ScrollOffsets() (float32, float32)
}

func (p *Panel) OnMouseWheel(x, y float32, dx, dy float32) bool {
	LogEventf(getDebugName(p), "OnMouseWheel at (%.0f,%.0f) delta=(%.1f,%.1f)", x, y, dx, dy)
	return p.DispatchMouseWheel(x, y, dx, dy)
}

// OnKeyDown is deliberately NOT a broadcast. A panel has no key behaviour of its own, and offering
// every key to every child was how an UNFOCUSED control acted: press Right with a button focused and
// the first tab panel in the window changed page. The window now bubbles a key from the focused
// widget up through its ancestors, so a container that really does own keys (a tab panel, a dialog)
// implements OnKeyDown itself and the plain panel in between stays transparent.
//
// OnCharacter is gone for the same reason: it handed every typed character to every text widget in
// the panel.
