package widgets

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

// TimePickerStyle selects what the picker's panel looks like.
type TimePickerStyle int

const (
	// TimePickerGrid is a clock-face grid: hours in two columns of twelve,
	// minutes in five. Every cell is a value, so any time is one click away.
	TimePickerGrid TimePickerStyle = iota
	// TimePickerRollers is the mobile-app wheel: one scrollable column of hours
	// and one of minutes, the centred row being the value. Scroll it with the
	// mouse wheel, drag it, or tap a visible neighbour to bring it to the centre.
	TimePickerRollers
)

// timePanelContent is what the picker shows in its dropdown: the grid or the
// rollers. Both take the value and report what the user picked.
type timePanelContent interface {
	meui.Widget
	SetTime(hour, minute int)
	SetOnChanged(fn func(hour, minute int))
	OnMouseDown(x, y float32, button int) bool
	OnMouseMove(x, y float32)
	OnMouseUp(x, y float32, button int) bool
	OnMouseWheel(x, y, dx, dy float32) bool
}

// TimePicker is a time field: an editable HH:MM text box with a button that
// opens a time panel under it — a clock-face grid, or the mobile-style rollers
// (SetStyle). Either way any time is a click or a wheel away, and minutes are
// exact (not five-minute steps).
//
// The text is parsed leniently (9:5, 09:05, 09:05:30) and normalised on Enter.
// Seconds are carried and shown only when SetShowSeconds says so; the panel
// picks hours and minutes.
type TimePicker struct {
	meui.BaseContainer
	field       *TextField
	button      *Button
	gridPanel   *timePanel
	rollerPanel *timeRollers
	buttonWidth float32
	style       TimePickerStyle

	hour, minute, second int
	// hasTime is false while the picker is unset (an empty field) — the same
	// "no bound" an optional filter wants from the date picker.
	hasTime     bool
	showSeconds bool

	// panelPressed records that a press reached the open panel, and
	// panelPressedInside where it started; see DatePicker for why a release
	// must have a matching press.
	panelPressed       bool
	panelPressedInside bool

	onTimeChanged func(hour, minute, second int)
	panel         dropdownPanel
	// presentation decides whether the panel hangs under the field or lifts into a bottom sheet;
	// Auto follows the surface's size class, the same rule the date picker uses.
	presentation PickerPresentation
}

// TimePicker geometry, and the panel's grid.
const (
	TimePickerHeight      = float32(28)
	TimePickerButtonWidth = float32(28)
	timePickerPanelWidth  = float32(206)
	timePickerPanelHeight = float32(240)
	// The rollers panel (TimePickerRollers): two wheels, five rows visible.
	timeRollerWidth   = float32(118)
	timeRollerHeight  = float32(190)
	timeRollerRowH    = float32(34)
	timeRollerVisible = 5
	timeRollerPad     = float32(8)

	// One cell of the hour/minute grid.
	timeCellWidth  = float32(24)
	timeCellHeight = float32(17)
	timeGridPad    = float32(8)
	timeGridHeader = float32(20)
	timeGridGap    = float32(12)
	timeHourCols   = 2
	timeMinuteCols = 5
	timeGridRows   = 12
)

// timePickerLayouts are the accepted typed forms. Hours may be one or two
// digits; minutes and seconds must be two, which is what makes "9:5" read as
// 09:05 rather than being rejected.
var timePickerLayouts = []string{"15:04", "15:04:05", "3:04", "3:04:05"}

// NewTimePicker returns a picker showing the current local time.
func NewTimePicker() *TimePicker {
	now := time.Now()
	t := &TimePicker{
		BaseContainer: meui.BaseContainer{BaseWidget: *meui.NewBaseWidget()},
		field:         NewTextField(),
		button:        NewButton(),
		gridPanel:     newTimePanel(),
		rollerPanel:   newTimeRollers(),
		buttonWidth:   TimePickerButtonWidth,
		hour:          now.Hour(),
		minute:        now.Minute(),
		second:        now.Second(),
		hasTime:       true,
	}
	// Parent the inner widgets to the OUTER widget: the parent chain is what the
	// cursor walk, focus, layers and theme lookups climb.
	t.AddChild(t.field)
	t.AddChild(t.button)
	t.field.SetParent(t)
	t.button.SetParent(t)

	t.button.SetIcon("clock")
	t.button.SetTooltip("Pick a time")
	t.button.SetOnClicked(func() { t.SetExpanded(!t.panel.isOpen()) })
	t.field.SetOnTextChanged(func(text string) { t.textChanged(text) })
	t.field.SetOnSubmit(func(text string) { t.commitText(text) })
	t.gridPanel.SetOnChanged(t.applyPicked)
	t.rollerPanel.SetOnChanged(t.applyPicked)

	t.field.SetText(t.timeText())
	t.gridPanel.SetTime(t.hour, t.minute)
	t.rollerPanel.SetTime(t.hour, t.minute)
	t.SetHeight(TimePickerHeight)
	return t
}

// applyPicked takes a value the panel reported and shows it, which is what a
// pick means: the field is the widget's text, and the app hears about it.
func (t *TimePicker) applyPicked(hour, minute int) {
	t.hour, t.minute, _ = normaliseClock(hour, minute, t.second)
	t.hasTime = true
	t.field.SetText(t.timeText())
	t.fireTimeChanged()
}

// SetStyle selects the panel's presentation.
func (t *TimePicker) SetStyle(style TimePickerStyle) *TimePicker {
	t.style = style
	t.MarkForRedraw()
	return t
}

// Style returns the current presentation.
func (t *TimePicker) Style() TimePickerStyle { return t.style }

// content is the panel widget the current style uses.
func (t *TimePicker) content() timePanelContent {
	if t.style == TimePickerRollers {
		return t.rollerPanel
	}
	return t.gridPanel
}

// panelSize is the dropdown size for the current style.
func (t *TimePicker) panelSize() (float32, float32) {
	if t.style == TimePickerRollers {
		return timeRollerWidth, timeRollerHeight
	}
	return timePickerPanelWidth, timePickerPanelHeight
}

// SetTime sets the value (and the field) without firing OnTimeChanged: a
// programmatic set is not a user edit.
func (t *TimePicker) SetTime(hour, minute, second int) *TimePicker {
	t.hour, t.minute, t.second = normaliseClock(hour, minute, second)
	t.hasTime = true
	t.field.SetText(t.timeText())
	t.gridPanel.SetTime(t.hour, t.minute)
	t.rollerPanel.SetTime(t.hour, t.minute)
	t.MarkForRedraw()
	return t
}

// Time returns the current hour, minute and second. HasTime says whether there
// is one: after Clear the field is empty and the returned value is the last
// time the picker had.
func (t *TimePicker) Time() (hour, minute, second int) {
	return t.hour, t.minute, t.second
}

// HasTime reports whether the picker holds a time.
func (t *TimePicker) HasTime() bool { return t.hasTime }

// Clear unsets the picker: the field empties (showing its placeholder) and
// HasTime goes false, without firing OnTimeChanged.
func (t *TimePicker) Clear() *TimePicker {
	t.hasTime = false
	t.field.SetText("")
	t.MarkForRedraw()
	return t
}

// SetShowSeconds shows (and accepts) seconds. Hours and minutes are what the
// panel picks either way.
func (t *TimePicker) SetShowSeconds(show bool) *TimePicker {
	if t.showSeconds == show {
		return t
	}
	t.showSeconds = show
	if t.hasTime {
		t.field.SetText(t.timeText())
	}
	return t
}

// SetOnTimeChanged registers the callback fired when the user commits a time —
// by picking cells or by pressing Enter on a valid one.
func (t *TimePicker) SetOnTimeChanged(handler func(hour, minute, second int)) *TimePicker {
	t.onTimeChanged = handler
	return t
}

// SetText sets the field from text — the QML `text` property. A value that is
// not a time is shown as typed and leaves the current time in place.
func (t *TimePicker) SetText(text string) *TimePicker {
	t.field.SetText(text)
	t.textChanged(text)
	return t
}

// Text returns the field's text, exactly as shown.
func (t *TimePicker) Text() string { return t.field.Text }

// SetPlaceholder sets the text shown while the field is empty.
func (t *TimePicker) SetPlaceholder(text string) *TimePicker {
	t.field.SetPlaceholder(text)
	return t
}

// SetEnabled enables or disables the field and its button together.
func (t *TimePicker) SetEnabled(enabled bool) *TimePicker {
	t.BaseWidget.SetEnabled(enabled)
	t.field.SetEnabled(enabled)
	t.button.SetEnabled(enabled)
	if !enabled {
		t.SetExpanded(false)
	}
	return t
}

// Expanded reports whether the time panel is open.
func (t *TimePicker) Expanded() bool { return t.panel.isOpen() }

// SetExpanded opens or closes the time panel.
func (t *TimePicker) SetExpanded(expanded bool) *TimePicker {
	if expanded == t.panel.isOpen() {
		return t
	}
	if !expanded {
		t.panel.closePanel(t)
		t.MarkForRedraw()
		return t
	}
	if !t.Enabled() {
		return t
	}
	t.content().SetTime(t.hour, t.minute)
	w, h := t.panelSize()
	sheet := false
	if win := windowOf(t); win != nil {
		wf := win.FrameRect()
		if sheet = t.sheetFor(wf.Width, wf.Height); sheet {
			w = sheetPanelWidth(wf.Width)
			h += 2 * sheetPanelPad
		}
	}
	var ok bool
	if sheet {
		ok = t.panel.openSheet(t, w, h, t.paintPanel)
	} else {
		ok = t.panel.openPanel(t, w, h, t.paintPanel)
	}
	if !ok {
		return t
	}
	t.MarkForRedraw()
	return t
}

// SetPresentation selects where the panel appears: anchored under the field, or a bottom sheet.
// Auto — the default — follows the surface's size class, so a phone gets the sheet.
func (t *TimePicker) SetPresentation(p PickerPresentation) *TimePicker {
	t.presentation = p
	return t
}

func (t *TimePicker) Presentation() PickerPresentation { return t.presentation }

func (t *TimePicker) sheetFor(width, height float32) bool {
	switch t.presentation {
	case PickerDropdown:
		return false
	case PickerSheet:
		return true
	default:
		return platform.FormFactorFor(width, height, 1) == platform.FormFactorPhone
	}
}

// PanelRect is the open panel's rect in the window's content space, or the zero rect when closed.
func (t *TimePicker) PanelRect() graphics.Rect {
	if !t.panel.isOpen() {
		return graphics.Rect{}
	}
	return graphics.NewRect(t.panel.offsetX, t.panel.offsetY, t.panel.width, t.panel.height)
}

func (t *TimePicker) paintPanel(canvas *graphics.Canvas, panel graphics.Rect) {
	content := t.content()
	content.SetFrameRect(panel)
	content.Paint(canvas, panel)
}

// ── value and text ──────────────────────────────────────────────────

// normaliseClock wraps out-of-range values through a real clock, so 25:70 is
// 02:10 rather than a stored nonsense.
func normaliseClock(hour, minute, second int) (int, int, int) {
	total := hour*3600 + minute*60 + second
	if total < 0 {
		total = 0
	}
	total %= 24 * 3600
	return total / 3600, total / 60 % 60, total % 60
}

func (t *TimePicker) timeText() string {
	if t.showSeconds {
		return fmt.Sprintf("%02d:%02d:%02d", t.hour, t.minute, t.second)
	}
	return fmt.Sprintf("%02d:%02d", t.hour, t.minute)
}

// parseTime accepts the typed forms above.
func parseTime(text string) (hour, minute, second int, ok bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, 0, 0, false
	}
	for _, layout := range timePickerLayouts {
		if parsed, err := time.Parse(layout, text); err == nil {
			return parsed.Hour(), parsed.Minute(), parsed.Second(), true
		}
	}
	return 0, 0, 0, false
}

func (t *TimePicker) textChanged(text string) {
	if strings.TrimSpace(text) == "" {
		t.hasTime = false
		return
	}
	hour, minute, second, ok := parseTime(text)
	if !ok {
		return
	}
	changed := hour != t.hour || minute != t.minute || second != t.second
	t.hour, t.minute, t.second = normaliseClock(hour, minute, second)
	t.hasTime = true
	t.gridPanel.SetTime(t.hour, t.minute)
	t.rollerPanel.SetTime(t.hour, t.minute)
	if changed {
		t.fireTimeChanged()
	}
}

func (t *TimePicker) commitText(text string) {
	if hour, minute, second, ok := parseTime(text); ok {
		t.hour, t.minute, t.second = normaliseClock(hour, minute, second)
		t.hasTime = true
		t.fireTimeChanged()
	} else if strings.TrimSpace(text) == "" {
		t.hasTime = false
		t.SetExpanded(false)
		return
	}
	t.field.SetText(t.timeText())
	t.gridPanel.SetTime(t.hour, t.minute)
	t.rollerPanel.SetTime(t.hour, t.minute)
	t.SetExpanded(false)
}

func (t *TimePicker) fireTimeChanged() {
	if t.onTimeChanged != nil {
		h, m, s := t.Time()
		t.onTimeChanged(h, m, s)
	}
}

// ── layout, painting and events ─────────────────────────────────────

func (t *TimePicker) SetFrameRect(rect graphics.Rect) {
	t.BaseWidget.SetFrameRect(rect)
	t.SetNeedsLayout()
}

// ValidateLayout places the field and the button inside the frame.
func (t *TimePicker) ValidateLayout() {
	frame := t.FrameRect()
	if frame.Width <= 0 || frame.Height <= 0 {
		t.ClearNeedsLayout()
		return
	}
	btnW := t.buttonWidth
	if btnW > frame.Width/2 {
		btnW = frame.Width / 2
	}
	t.field.SetFrameRect(graphics.NewRect(0, 0, frame.Width-btnW, frame.Height))
	t.button.SetFrameRect(graphics.NewRect(frame.Width-btnW, 0, btnW, frame.Height))
	t.ClearNeedsLayout()
}

func (t *TimePicker) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !t.Visible() {
		return
	}
	if t.NeedsLayout() {
		t.ValidateLayout()
	}
	t.field.Paint(canvas, t.field.FrameRect())
	t.button.Paint(canvas, t.button.FrameRect())
}

// PanelAt routes a point to the field or the button.
func (t *TimePicker) PanelAt(x, y float32) meui.Widget {
	frame := t.FrameRect()
	if x < 0 || y < 0 || x > frame.Width || y > frame.Height {
		return nil
	}
	if bf := t.button.FrameRect(); x >= bf.X && x <= bf.X+bf.Width {
		return t.button
	}
	return t.field
}

// OnMouseDown, OnMouseUp and OnMouseMove forward to the field or the button
// under the point: a container dispatches by containment, not to the deepest
// hit, so a field inside a modal popup would otherwise never see the press.
func (t *TimePicker) OnMouseDown(x, y float32, button int) bool {
	target := t.PanelAt(x, y)
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

func (t *TimePicker) OnMouseUp(x, y float32, button int) bool {
	target := t.PanelAt(x, y)
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

func (t *TimePicker) OnMouseMove(x, y float32) {
	target := t.PanelAt(x, y)
	if target == nil {
		return
	}
	tf := target.FrameRect()
	if el, ok := target.(interface{ OnMouseMove(float32, float32) }); ok {
		el.OnMouseMove(x-tf.X, y-tf.Y)
	}
}

// ── the open panel owns the pointer ─────────────────────────────────

func (t *TimePicker) HandleMouseDown(lx, ly float32, button int) bool {
	if !t.panel.isOpen() {
		return false
	}
	px, py, inside := t.panel.inside(lx, ly)
	t.panelPressed = true
	t.panelPressedInside = inside
	if !inside {
		// Dismiss on the release, never on the press: the button opens on the
		// release, so closing here would let the same press reopen it.
		return true
	}
	t.content().SetFrameRect(graphics.NewRect(0, 0, t.panel.width, t.panel.height))
	t.content().OnMouseDown(px, py, button)
	// A click on the panel is a complete pick, so the panel closes with it. The
	// grid is one value per cell, so the press is the whole interaction; the
	// rollers wait for the release, because a drag has to stay open until then.
	// The wheel deliberately does not close (see OnMouseWheel): closing on every
	// notch would make scrolling unusable.
	if t.style == TimePickerGrid {
		t.SetExpanded(false)
	}
	return true
}

func (t *TimePicker) HandleMouseMove(lx, ly float32) bool {
	if !t.panel.isOpen() {
		return false
	}
	px, py, inside := t.panel.inside(lx, ly)
	if !inside {
		return true
	}
	t.content().SetFrameRect(graphics.NewRect(0, 0, t.panel.width, t.panel.height))
	t.content().OnMouseMove(px, py)
	return true
}

func (t *TimePicker) HandleMouseUp(lx, ly float32, button int) bool {
	if !t.panel.isOpen() {
		return false
	}
	pressed := t.panelPressed
	t.panelPressed = false
	px, py, inside := t.panel.inside(lx, ly)
	if !inside {
		if pressed && !t.panelPressedInside {
			t.SetExpanded(false)
		}
		return true
	}
	t.content().SetFrameRect(graphics.NewRect(0, 0, t.panel.width, t.panel.height))
	t.content().OnMouseUp(px, py, button)
	// The rollers commit on the release: a click on the wheel — a tap or the end
	// of a drag — closes the panel, exactly as a click on the grid does.
	t.SetExpanded(false)
	return true
}

// OnMouseWheel scrolls the open panel. The layer manager routes the wheel to
// the picker it belongs to, in layer-local coordinates, so the rollers scroll
// under the pointer — a wheel up (positive dy) moves the value up, the same
// direction as dragging the wheel upwards.
func (t *TimePicker) OnMouseWheel(x, y float32, dx, dy float32) bool {
	if !t.panel.isOpen() {
		return false
	}
	px, py, inside := t.panel.inside(x, y)
	if !inside {
		return false
	}
	t.content().SetFrameRect(graphics.NewRect(0, 0, t.panel.width, t.panel.height))
	return t.content().OnMouseWheel(px, py, dx, dy)
}

// CursorAt returns nil: the deepest hit under the pointer is the field, whose
// own CursorAt reports the text cursor, or the button, which keeps the arrow.
func (t *TimePicker) CursorAt(x, y float32) *platform.Cursor { return nil }

// SetBoundProperty applies a QML property (the loader routes unhandled
// properties here — see qml.HandleProperty).
func (t *TimePicker) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "placeholder":
		if s, ok := value.(string); ok {
			t.SetPlaceholder(s)
		}
	case "style":
		if s, ok := value.(string); ok {
			switch s {
			case "rollers", "wheel":
				t.SetStyle(TimePickerRollers)
			case "grid":
				t.SetStyle(TimePickerGrid)
			}
		}
	case "hour", "minute", "second":
		v, ok := propertyInt(value)
		if !ok {
			return
		}
		h, m, s := t.Time()
		switch propName {
		case "hour":
			h = int(v)
		case "minute":
			m = int(v)
		case "second":
			s = int(v)
		}
		t.SetTime(h, m, s)
	}
}

// ── the panel's clock-face grid ─────────────────────────────────────

// timePanel is the grid the time picker opens: a column pair for the hours and
// five columns for the minutes, each cell one value, the current one filled with
// the theme's primary colour. It is private to the package: callers configure
// the picker, not the grid.
type timePanel struct {
	meui.BaseWidget
	hour, minute  int
	hoveredHour   int
	hoveredMinute int
	onChanged     func(hour, minute int)
}

func newTimePanel() *timePanel {
	return &timePanel{
		BaseWidget:    *meui.NewBaseWidget(),
		hoveredHour:   -1,
		hoveredMinute: -1,
	}
}

func (p *timePanel) SetTime(hour, minute int) {
	p.hour, p.minute = hour, minute
	p.MarkForRedraw()
}

func (p *timePanel) SetOnChanged(fn func(hour, minute int)) { p.onChanged = fn }

// cellRect is the rect of the cell at (col, row) in one of the two grids.
func (p *timePanel) cellRect(hours bool, col, row int) graphics.Rect {
	x := timeGridPad
	if !hours {
		x += timeHourCols*timeCellWidth + timeGridGap
	}
	y := timeGridPad + timeGridHeader
	return graphics.NewRect(
		x+float32(col)*timeCellWidth,
		y+float32(row)*timeCellHeight,
		timeCellWidth, timeCellHeight)
}

// valueAt resolves a panel-local point to a value: which grid, and which number.
func (p *timePanel) valueAt(x, y float32) (value int, isHour, ok bool) {
	y0 := timeGridPad + timeGridHeader
	if y < y0 {
		return 0, false, false
	}
	row := int((y - y0) / timeCellHeight)
	if row < 0 || row >= timeGridRows {
		return 0, false, false
	}
	if x >= timeGridPad && x < timeGridPad+timeHourCols*timeCellWidth {
		col := int((x - timeGridPad) / timeCellWidth)
		return row + col*timeGridRows, true, true
	}
	minutesX := timeGridPad + timeHourCols*timeCellWidth + timeGridGap
	if x >= minutesX && x < minutesX+timeMinuteCols*timeCellWidth {
		col := int((x - minutesX) / timeCellWidth)
		return row + col*timeGridRows, false, true
	}
	return 0, false, false
}

func (p *timePanel) OnMouseDown(x, y float32, button int) bool {
	if !p.Enabled() || button != 0 {
		return false
	}
	value, isHour, ok := p.valueAt(x, y)
	if !ok {
		return true
	}
	if isHour {
		p.hour = value
	} else {
		p.minute = value
	}
	if p.onChanged != nil {
		p.onChanged(p.hour, p.minute)
	}
	p.MarkForRedraw()
	return true
}

func (p *timePanel) OnMouseMove(x, y float32) {
	value, isHour, ok := p.valueAt(x, y)
	hoverHour, hoverMinute := -1, -1
	if ok {
		if isHour {
			hoverHour = value
		} else {
			hoverMinute = value
		}
	}
	if hoverHour == p.hoveredHour && hoverMinute == p.hoveredMinute {
		return
	}
	p.hoveredHour, p.hoveredMinute = hoverHour, hoverMinute
	p.MarkForRedraw()
}

func (p *timePanel) OnMouseUp(x, y float32, button int) bool { return true }

func (p *timePanel) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !p.Visible() {
		return
	}
	canvas.Save()

	t := theme.DefaultCalendarTheme
	if t == nil {
		canvas.Restore()
		return
	}
	font := t.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}

	bg := graphics.NewPaint()
	bg.SetColor(t.BackgroundInk)
	canvas.DrawRoundedRect(rect, t.CornerRadius, t.CornerRadius, bg)

	header := graphics.NewPaint()
	header.SetColor(t.TextActive)
	labelY := rect.Y + timeGridPad + font.Baseline()
	DrawText(canvas, "Hour", rect.X+timeGridPad, labelY, font, header)
	minutesX := rect.X + timeGridPad + timeHourCols*timeCellWidth + timeGridGap
	DrawText(canvas, "Minute", minutesX, labelY, font, header)

	text := graphics.NewPaint()
	cell := func(hours bool, col, row, value int, selected, hovered bool) {
		r := p.cellRect(hours, col, row)
		r.X += rect.X
		r.Y += rect.Y
		inner := graphics.NewRect(r.X+1, r.Y+1, r.Width-2, r.Height-2)
		switch {
		case selected:
			fill := graphics.NewPaint()
			fill.SetColor(theme.DefaultTheme.Primary)
			canvas.DrawRoundedRect(inner, 4, 4, fill)
			text.SetColor(graphics.Color(0xFFFFFFFF))
		case hovered:
			fill := graphics.NewPaint()
			fill.SetColor(t.BackgroundHover)
			canvas.DrawRect(inner, fill)
			text.SetColor(t.TextHover)
		default:
			text.SetColor(t.TextInk)
		}
		label := fmt.Sprintf("%02d", value)
		w := font.SimpleWidth(label)
		DrawText(canvas, label, r.X+r.Width/2-w/2, r.Y+r.Height/2+font.Baseline()/2, font, text)
	}

	for col := 0; col < timeHourCols; col++ {
		for row := 0; row < timeGridRows; row++ {
			value := row + col*timeGridRows
			cell(true, col, row, value, value == p.hour, value == p.hoveredHour)
		}
	}
	for col := 0; col < timeMinuteCols; col++ {
		for row := 0; row < timeGridRows; row++ {
			value := row + col*timeGridRows
			cell(false, col, row, value, value == p.minute, value == p.hoveredMinute)
		}
	}

	// The border last, so it sits on top of the cells at the panel's edge.
	edge := graphics.NewPaint()
	edge.SetStyle(paintstyle.Stroke)
	edge.SetStrokeWidth(theme.BorderStrokeWidth)
	edge.SetColor(t.EdgeInk)
	canvas.DrawRoundedRect(rect, t.CornerRadius, t.CornerRadius, edge)

	canvas.Restore()
}

// ── the mobile-style rollers ────────────────────────────────────────

// timeRollers is the wheel presentation: one column of hours and one of
// minutes, the centred row being the value. It scrolls with the mouse wheel,
// with a drag, or by tapping a visible row; each column wraps, so a wheel never
// dead-ends. It is private to the package: callers pick the style.
type timeRollers struct {
	meui.BaseWidget
	hour, minute int

	dragging     bool
	dragColumn   int // 0 hours, 1 minutes
	dragStartY   float32
	dragStartVal int
	dragMoved    bool

	onChanged func(hour, minute int)
}

func newTimeRollers() *timeRollers {
	return &timeRollers{BaseWidget: *meui.NewBaseWidget(), dragColumn: -1}
}

func (r *timeRollers) SetTime(hour, minute int) {
	r.hour, r.minute = hour, minute
	r.MarkForRedraw()
}

func (r *timeRollers) SetOnChanged(fn func(hour, minute int)) { r.onChanged = fn }

// rollerColumnWidth is one wheel's width inside a panel of the given width.
func rollerColumnWidth(panel graphics.Rect) float32 {
	return (panel.Width - 3*timeRollerPad) / 2
}

// columnAt reports which wheel a panel-local x is over: 0 hours, 1 minutes,
// -1 neither.
func (r *timeRollers) columnAt(panel graphics.Rect, x float32) int {
	colW := rollerColumnWidth(panel)
	if x >= timeRollerPad && x <= timeRollerPad+colW {
		return 0
	}
	minutesX := timeRollerPad*2 + colW
	if x >= minutesX && x <= minutesX+colW {
		return 1
	}
	return -1
}

// rowOffsetAt is how many rows away from the centre a panel-local y is, for a
// panel of the given height.
func (r *timeRollers) rowOffsetAt(height, y float32) int {
	centre := height / 2
	return int(math.Round(float64((y - centre) / timeRollerRowH)))
}

// value returns a column's value, and sets it.
func (r *timeRollers) value(column int) int {
	if column == 1 {
		return r.minute
	}
	return r.hour
}

func (r *timeRollers) setValue(column, value int) {
	if column == 1 {
		r.minute = wrapClock(value, 60)
	} else {
		r.hour = wrapClock(value, 24)
	}
	if r.onChanged != nil {
		r.onChanged(r.hour, r.minute)
	}
	r.MarkForRedraw()
}

// wrapClock keeps a wheel value in range, wrapping rather than clamping: a wheel
// that stops at 23 feels broken.
func wrapClock(value, span int) int {
	value %= span
	if value < 0 {
		value += span
	}
	return value
}

func (r *timeRollers) OnMouseDown(x, y float32, button int) bool {
	if !r.Enabled() || button != 0 {
		return false
	}
	r.dragColumn = r.columnAt(r.FrameRect(), x)
	if r.dragColumn < 0 {
		return true
	}
	r.dragging = true
	r.dragMoved = false
	r.dragStartY = y
	r.dragStartVal = r.value(r.dragColumn)
	return true
}

func (r *timeRollers) OnMouseMove(x, y float32) {
	if !r.dragging {
		return
	}
	rows := int(math.Round(float64((r.dragStartY - y) / timeRollerRowH)))
	if rows != 0 {
		r.dragMoved = true
	}
	r.setValue(r.dragColumn, r.dragStartVal+rows)
}

func (r *timeRollers) OnMouseUp(x, y float32, button int) bool {
	if !r.dragging {
		return true
	}
	r.dragging = false
	// A tap (no drag) brings the row under the pointer to the centre, which is
	// how a phone's wheel behaves.
	if !r.dragMoved {
		if offset := r.rowOffsetAt(r.FrameRect().Height, y); offset != 0 {
			r.setValue(r.dragColumn, r.value(r.dragColumn)+offset)
		}
	}
	return true
}

// OnMouseWheel scrolls the wheel under the pointer (or the hour wheel when the
// pointer is between the two).
//
// A wheel is a list of values, so scrolling DOWN — negative dy, the direction
// that shows later items in a list — moves the value up (11:11 scrolls down to
// 11:12). That is also the direction a drag and a trackpad two-finger scroll
// move it, so the wheel and the drag agree. The number-spinner convention
// (scroll up increments) is the other way round and reads as upside down here.
func (r *timeRollers) OnMouseWheel(x, y, dx, dy float32) bool {
	if !r.Enabled() || dy == 0 {
		return false
	}
	column := r.columnAt(r.FrameRect(), x)
	if column < 0 {
		column = 0
	}
	step := 1
	if dy > 0 {
		step = -1
	}
	r.setValue(column, r.value(column)+step)
	return true
}

func (r *timeRollers) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !r.Visible() {
		return
	}
	canvas.Save()

	t := theme.DefaultCalendarTheme
	if t == nil {
		canvas.Restore()
		return
	}
	font := t.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}

	bg := graphics.NewPaint()
	bg.SetColor(t.BackgroundInk)
	canvas.DrawRoundedRect(rect, t.CornerRadius, t.CornerRadius, bg)

	// The selection band across both wheels: the centred row is the value.
	centre := rect.Y + rect.Height/2
	band := graphics.NewRect(rect.X+timeRollerPad, centre-timeRollerRowH/2,
		rect.Width-2*timeRollerPad, timeRollerRowH)
	bandPaint := graphics.NewPaint()
	bandPaint.SetColor(t.BackgroundActive)
	canvas.DrawRoundedRect(band, 6, 6, bandPaint)

	colW := rollerColumnWidth(rect)
	half := timeRollerVisible / 2
	for column := 0; column < 2; column++ {
		x := rect.X + timeRollerPad + float32(column)*(colW+timeRollerPad)
		selected := r.value(column)
		for i := -half; i <= half; i++ {
			value := wrapClock(selected+i, 24)
			if column == 1 {
				value = wrapClock(selected+i, 60)
			}
			text := graphics.NewPaint()
			switch {
			case i == 0:
				text.SetColor(t.TextActive)
			case i == -1 || i == 1:
				text.SetColor(withAlpha(t.TextInk, 0xCC))
			default:
				text.SetColor(withAlpha(t.TextInk, 0x77))
			}
			label := fmt.Sprintf("%02d", value)
			w := font.SimpleWidth(label)
			y := centre + float32(i)*timeRollerRowH + font.Baseline()/2
			DrawText(canvas, label, x+colW/2-w/2, y, font, text)
		}
	}

	edge := graphics.NewPaint()
	edge.SetStyle(paintstyle.Stroke)
	edge.SetStrokeWidth(theme.BorderStrokeWidth)
	edge.SetColor(t.EdgeInk)
	canvas.DrawRoundedRect(rect, t.CornerRadius, t.CornerRadius, edge)

	canvas.Restore()
}

// withAlpha replaces a colour's alpha, which is how the rollers fade the rows
// away from the centre without a second set of theme tokens.
func withAlpha(c graphics.Color, alpha uint8) graphics.Color {
	return graphics.Color(uint32(c)&0x00FFFFFF | uint32(alpha)<<24)
}
