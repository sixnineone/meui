package widgets

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

// ColorField is a colour entry: a hex text box with a swatch button that opens
// a colour picker under it. The text and the swatch always show the same
// colour, typing a valid hex updates the swatch (and the picker, and
// OnColorChanged) immediately, and text that is not a colour yet leaves the
// current colour alone rather than clearing it.
//
// The picker is the toolkit's ColorPicker, shown in an anchored dropdown panel
// (see dropdownPanel); a click anywhere outside closes it. Picking is live: the
// colour changes as the pointer moves in the picker, with no OK/Cancel step.
type ColorField struct {
	meui.BaseContainer
	field       *TextField
	button      *Button
	picker      *ColorPicker
	buttonWidth float32
	color       graphics.Color

	// panelPressed records that a press reached the open panel at all, and
	// panelPressedInside where it started. A press outside dismisses on the
	// *release*, never on the press (see HandleMouseDown) — but only a release
	// with a matching press: the release that OPENS the panel must not close it,
	// and the window offers every release to the layers twice, so it does reach
	// an open panel that this very press created.
	panelPressed       bool
	panelPressedInside bool

	onColorChanged func(color graphics.Color)
	panel          dropdownPanel
}

// ColorField geometry. The panel is sized for the picker: its SV square is
// min(width-30, height), so 250x210 leaves a full-height square plus the hue bar.
const (
	ColorFieldHeight      = float32(28)
	ColorFieldButtonWidth = float32(34)
	colorFieldPanelWidth  = float32(250)
	colorFieldPanelHeight = float32(210)
)

// NewColorField returns a field showing white.
func NewColorField() *ColorField {
	c := &ColorField{
		BaseContainer: meui.BaseContainer{BaseWidget: *meui.NewBaseWidget()},
		field:         NewTextField(),
		button:        NewButton(),
		picker:        NewColorPicker(),
		buttonWidth:   ColorFieldButtonWidth,
		color:         graphics.RGBA(0xFF, 0xFF, 0xFF, 0xFF),
	}
	// Parent the inner widgets to the OUTER widget, so the parent chain (cursor
	// walk, focus, layers, theme) does not stop at the embedded container.
	c.AddChild(c.field)
	c.AddChild(c.button)
	c.field.SetParent(c)
	c.button.SetParent(c)

	c.button.SetTooltip("Pick a colour")
	c.button.SetOnClicked(func() { c.SetExpanded(!c.panel.isOpen()) })
	c.field.SetOnTextChanged(func(string) { c.textChanged() })
	c.field.SetOnSubmit(func(string) { c.commitText() })

	c.field.SetText(FormatHexColor(c.color))
	c.picker.SetColor(0xFF, 0xFF, 0xFF, 0xFF)
	c.picker.SetOnColorChanged(func(r, g, b, a uint8) {
		c.color = graphics.RGBA(r, g, b, a)
		c.field.SetText(FormatHexColor(c.color))
		c.fireColorChanged()
		c.MarkForRedraw()
	})
	c.SetHeight(ColorFieldHeight)
	return c
}

// SetColor sets the colour (and the field and picker) without firing
// OnColorChanged: a programmatic set is not a user edit.
func (c *ColorField) SetColor(color graphics.Color) *ColorField {
	c.color = color
	c.field.SetText(FormatHexColor(color))
	r, g, b, a := ColorChannels(color)
	c.picker.SetColor(r, g, b, a)
	c.MarkForRedraw()
	return c
}

// Color returns the current colour.
func (c *ColorField) Color() graphics.Color { return c.color }

// Hex returns the current colour in the field's own text form.
func (c *ColorField) Hex() string { return FormatHexColor(c.color) }

// Picker returns the popup's colour picker, for the caller that wants to
// configure it (its theme, say). It is the same widget the panel paints; the
// field gives it its rect when the panel opens.
func (c *ColorField) Picker() *ColorPicker { return c.picker }

// SetOnColorChanged registers the callback fired whenever the user changes the
// colour (typing a valid hex or moving in the picker).
func (c *ColorField) SetOnColorChanged(handler func(color graphics.Color)) *ColorField {
	c.onColorChanged = handler
	return c
}

// SetText sets the field from text — the QML `text` property. A value that is
// not a colour is shown as typed and leaves the current colour in place.
func (c *ColorField) SetText(text string) *ColorField {
	c.field.SetText(text)
	c.textChanged()
	return c
}

// Text returns the field's text, exactly as shown.
func (c *ColorField) Text() string { return c.field.Text }

// SetPlaceholder sets the text shown while the field is empty.
func (c *ColorField) SetPlaceholder(text string) *ColorField {
	c.field.SetPlaceholder(text)
	return c
}

// SetEnabled enables or disables the field and its button together.
func (c *ColorField) SetEnabled(enabled bool) *ColorField {
	c.BaseWidget.SetEnabled(enabled)
	c.field.SetEnabled(enabled)
	c.button.SetEnabled(enabled)
	if !enabled {
		c.SetExpanded(false)
	}
	return c
}

// Expanded reports whether the picker panel is open.
func (c *ColorField) Expanded() bool { return c.panel.isOpen() }

// SetExpanded opens or closes the picker panel.
func (c *ColorField) SetExpanded(expanded bool) *ColorField {
	if expanded == c.panel.isOpen() {
		return c
	}
	if !expanded {
		c.panel.closePanel(c)
		c.MarkForRedraw()
		return c
	}
	if !c.Enabled() {
		return c
	}
	r, g, b, a := ColorChannels(c.color)
	c.picker.SetColor(r, g, b, a)
	if !c.panel.openPanel(c, colorFieldPanelWidth, colorFieldPanelHeight, c.paintPicker) {
		return c
	}
	c.MarkForRedraw()
	return c
}

// paintPicker draws the picker into the panel, giving it the panel's rect: the
// picker adapts to whatever rect it is painted with.
func (c *ColorField) paintPicker(canvas *graphics.Canvas, panel graphics.Rect) {
	c.picker.SetFrameRect(panel)
	c.picker.Paint(canvas, panel)
}

// ── text ────────────────────────────────────────────────────────────

// textChanged follows the field as the user types: a valid hex takes effect
// immediately, anything else is ignored so a half-typed "#11" does not blank
// the colour.
func (c *ColorField) textChanged() {
	color, err := ParseHexColor(c.field.Text, c.color)
	if err != nil || color == c.color {
		return
	}
	c.color = color
	r, g, b, a := ColorChannels(color)
	c.picker.SetColor(r, g, b, a)
	c.fireColorChanged()
	c.MarkForRedraw()
}

// commitText rewrites the field in canonical form on Enter, so the text always
// ends up a colour even if the user typed it without the leading '#'.
func (c *ColorField) commitText() {
	color, err := ParseHexColor(c.field.Text, c.color)
	if err == nil {
		c.color = color
	}
	c.field.SetText(FormatHexColor(c.color))
	r, g, b, a := ColorChannels(c.color)
	c.picker.SetColor(r, g, b, a)
	c.MarkForRedraw()
}

func (c *ColorField) fireColorChanged() {
	if c.onColorChanged != nil {
		c.onColorChanged(c.color)
	}
}

// ── the hex codec ───────────────────────────────────────────────────

// FormatHexColor renders a colour as "#RRGGBB", or "#RRGGBBAA" when it is
// translucent — the usual web order, and the exact inverse of ParseHexColor for
// every colour ParseHexColor can produce.
func FormatHexColor(color graphics.Color) string {
	r, g, b, a := ColorChannels(color)
	if a == 0xFF {
		return fmt.Sprintf("#%02X%02X%02X", r, g, b)
	}
	return fmt.Sprintf("#%02X%02X%02X%02X", r, g, b, a)
}

// ParseHexColor reads "#RRGGBB" or "#RRGGBBAA" (the '#' is optional). A
// six-digit value keeps the alpha of current: a user typing a new RGB means
// that RGB, not "and also reset the transparency", and the colour they were
// editing already has the alpha they want.
func ParseHexColor(text string, current graphics.Color) (graphics.Color, error) {
	s := strings.TrimSpace(text)
	s = strings.TrimPrefix(s, "#")
	if s == "" {
		return 0, fmt.Errorf("empty colour")
	}
	switch len(s) {
	case 6:
		v, err := strconv.ParseUint(s, 16, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid colour %q: %w", text, err)
		}
		return graphics.RGBA(uint8(v>>16), uint8(v>>8), uint8(v), uint8(uint32(current)>>24)), nil
	case 8:
		v, err := strconv.ParseUint(s, 16, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid colour %q: %w", text, err)
		}
		return graphics.RGBA(uint8(v>>24), uint8(v>>16), uint8(v>>8), uint8(v)), nil
	default:
		return 0, fmt.Errorf("invalid colour %q: expected #RRGGBB or #RRGGBBAA", text)
	}
}

// ColorChannels splits a colour into its channels, for the picker and the
// swatch.
func ColorChannels(color graphics.Color) (r, g, b, a uint8) {
	v := uint32(color)
	return uint8(v >> 16), uint8(v >> 8), uint8(v), uint8(v >> 24)
}

// ── layout, painting and events ─────────────────────────────────────

func (c *ColorField) SetFrameRect(rect graphics.Rect) {
	c.BaseWidget.SetFrameRect(rect)
	c.SetNeedsLayout()
}

// ValidateLayout places the field and the swatch button inside the frame.
func (c *ColorField) ValidateLayout() {
	frame := c.FrameRect()
	if frame.Width <= 0 || frame.Height <= 0 {
		c.ClearNeedsLayout()
		return
	}
	btnW := c.buttonWidth
	if btnW > frame.Width/2 {
		btnW = frame.Width / 2
	}
	c.field.SetFrameRect(graphics.NewRect(0, 0, frame.Width-btnW, frame.Height))
	c.button.SetFrameRect(graphics.NewRect(frame.Width-btnW, 0, btnW, frame.Height))
	c.ClearNeedsLayout()
}

func (c *ColorField) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !c.Visible() {
		return
	}
	if c.NeedsLayout() {
		c.ValidateLayout()
	}
	c.field.Paint(canvas, c.field.FrameRect())
	c.button.Paint(canvas, c.button.FrameRect())
	c.paintSwatch(canvas)
}

// paintSwatch draws the current colour inside the button, so the control shows
// the value it edits without spending the button's icon slot.
func (c *ColorField) paintSwatch(canvas *graphics.Canvas) {
	bf := c.button.FrameRect()
	inset := float32(7)
	swatch := graphics.NewRect(bf.X+inset, bf.Y+inset, bf.Width-inset*2, bf.Height-inset*2)
	if swatch.Width <= 0 || swatch.Height <= 0 {
		return
	}
	canvas.Save()
	canvas.ClipRect(bf, pathop.Intersect, false)
	radius := theme.Shape().Input
	if radius > swatch.Height/2 {
		radius = swatch.Height / 2
	}
	fill := graphics.NewPaint()
	fill.SetColor(c.color)
	canvas.DrawRoundedRect(swatch, radius, radius, fill)
	edge := graphics.NewPaint()
	edge.SetStyle(paintstyle.Stroke)
	edge.SetStrokeWidth(theme.BorderStrokeWidth)
	edge.SetColor(theme.DefaultTheme.BorderStrong)
	canvas.DrawRoundedRect(swatch, radius, radius, edge)
	canvas.Restore()
}

// PanelAt routes a point to the field or the swatch button.
func (c *ColorField) PanelAt(x, y float32) meui.Widget {
	frame := c.FrameRect()
	if x < 0 || y < 0 || x > frame.Width || y > frame.Height {
		return nil
	}
	if bf := c.button.FrameRect(); x >= bf.X && x <= bf.X+bf.Width {
		return c.button
	}
	return c.field
}

// OnMouseDown, OnMouseUp and OnMouseMove forward to the field or the button
// under the point. A container dispatches by containment — a modal popup hands
// the press to its content's children, not to the deepest hit — so the composite
// must route the event itself; relying on the window's PanelAt delivery alone
// left every field inside a modal dead (the owner's "the colour picker does not
// pop up" in the Settings popup).
func (c *ColorField) OnMouseDown(x, y float32, button int) bool {
	target := c.PanelAt(x, y)
	if target == nil {
		return false
	}
	tf := target.FrameRect()
	if el, ok := target.(interface {
		OnMouseDown(float32, float32, int) bool
	}); ok {
		return el.OnMouseDown(x-tf.X, y-tf.Y, button)
	}
	return false
}

func (c *ColorField) OnMouseUp(x, y float32, button int) bool {
	target := c.PanelAt(x, y)
	if target == nil {
		return false
	}
	tf := target.FrameRect()
	if el, ok := target.(interface {
		OnMouseUp(float32, float32, int) bool
	}); ok {
		return el.OnMouseUp(x-tf.X, y-tf.Y, button)
	}
	return false
}

func (c *ColorField) OnMouseMove(x, y float32) {
	target := c.PanelAt(x, y)
	if target == nil {
		return
	}
	tf := target.FrameRect()
	if el, ok := target.(interface{ OnMouseMove(float32, float32) }); ok {
		el.OnMouseMove(x-tf.X, y-tf.Y)
	}
}

// ── the open panel owns the pointer ─────────────────────────────────

// HandleMouseDown receives clicks while the panel is open (the layer is modal):
// inside the panel the picker gets the press, anywhere else closes it.
func (c *ColorField) HandleMouseDown(lx, ly float32, button int) bool {
	if !c.panel.isOpen() {
		return false
	}
	px, py, inside := c.panel.inside(lx, ly)
	c.panelPressed = true
	c.panelPressedInside = inside
	if !inside {
		// Do not close on the press: the owner's button opens on the RELEASE, so
		// closing here lets the same press toggle the panel straight back open
		// (the button's click arrives after the panel is already gone). The
		// release closes it — HandleMouseUp below.
		return true
	}
	c.picker.SetFrameRect(graphics.NewRect(0, 0, c.panel.width, c.panel.height))
	c.picker.OnMouseDown(px, py, button)
	return true
}

// HandleMouseMove drives the picker's drag, which is also what makes picking
// live: the picker fires OnColorChanged from here.
func (c *ColorField) HandleMouseMove(lx, ly float32) bool {
	if !c.panel.isOpen() {
		return false
	}
	px, py, inside := c.panel.inside(lx, ly)
	if !inside {
		return true
	}
	c.picker.SetFrameRect(graphics.NewRect(0, 0, c.panel.width, c.panel.height))
	c.picker.OnMouseMove(px, py)
	return true
}

func (c *ColorField) HandleMouseUp(lx, ly float32, button int) bool {
	if !c.panel.isOpen() {
		return false
	}
	pressed := c.panelPressed
	c.panelPressed = false
	px, py, inside := c.panel.inside(lx, ly)
	if !inside {
		if pressed && !c.panelPressedInside {
			c.SetExpanded(false)
		}
		return true
	}
	c.picker.SetFrameRect(graphics.NewRect(0, 0, c.panel.width, c.panel.height))
	c.picker.OnMouseUp(px, py, button)
	return true
}

// SetBoundProperty applies a QML property to the field. The loader routes every
// property without a handler of its own here (see qml.HandleProperty), so this
// is what makes `placeholder` and `color` work in QML.
func (c *ColorField) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "placeholder":
		if s, ok := value.(string); ok {
			c.SetPlaceholder(s)
		}
	case "color":
		if s, ok := value.(string); ok {
			if color, err := ParseHexColor(s, c.color); err == nil {
				c.SetColor(color)
			}
		}
	}
}

// CursorAt returns nil: over the field the text cursor comes from the field
// itself, over the button the arrow stays the default.
func (c *ColorField) CursorAt(x, y float32) *platform.Cursor { return nil }
