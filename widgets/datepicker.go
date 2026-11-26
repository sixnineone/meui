package widgets

import (
	"strings"
	"time"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
)

// DatePicker is a date field: an editable text box in ISO form with a button
// that opens a calendar under it. The text is parsed leniently — 2026-9-5,
// 2026/09/05 and 2026.9.5 all mean the same day — and normalised to YYYY-MM-DD
// on commit, so typing a date is not a fight with the format. Text that is not
// a date (yet) leaves the last good date in place; the field is only rewritten
// when the user finishes (Enter) or picks a day.
//
// The button opens the panel as an anchored dropdown in the window's layer
// space (see dropdownPanel): a click anywhere outside closes it.
type DatePicker struct {
	meui.BaseContainer
	field       *TextField
	button      *Button
	calendar    *Calendar
	buttonWidth float32

	year, month, day int
	// hasDate is false while the picker is unset (an empty field), which is how
	// an optional date reads: "no bound" rather than "today". NewDatePicker
	// starts set to today; Clear() unsets it.
	hasDate bool

	// panelPressed records that a press reached the open panel at all, and
	// panelPressedInside where it started. A press outside dismisses on the
	// *release*, never on the press (see HandleMouseDown) — but only a release
	// with a matching press: the release that OPENS the panel must not close it,
	// and the window offers every release to the layers twice, so it does reach
	// an open panel that this very press created.
	panelPressed       bool
	panelPressedInside bool

	onDateChanged func(year, month, day int)
	panel         dropdownPanel
	// presentation decides whether the calendar hangs under the field or lifts into a bottom
	// sheet. Auto follows the surface's size class, so a phone gets the sheet.
	presentation PickerPresentation
}

// DatePicker geometry. The panel is sized for a calendar: seven columns, a
// 30px header and six week rows (see Calendar.Paint).
const (
	DatePickerHeight      = float32(28)
	DatePickerButtonWidth = float32(28)
	datePickerPanelWidth  = float32(238)
	datePickerPanelHeight = float32(212)
)

// datePickerLayouts are the accepted typed forms, all year-month-day so a date
// is never ambiguous (02/03/2026 is not "2 March" in one locale and "3
// February" in another). time.Parse validates the day for the month, which is
// what rejects 2026-02-30.
var datePickerLayouts = []string{"2006-1-2", "2006/1/2", "2006.1.2"}

// datePickerCanonical is what the field shows once a date is committed.
const datePickerCanonical = "2006-01-02"

// NewDatePicker returns a picker showing today.
func NewDatePicker() *DatePicker {
	now := time.Now()
	d := &DatePicker{
		BaseContainer: meui.BaseContainer{BaseWidget: *meui.NewBaseWidget()},
		field:         NewTextField(),
		button:        NewButton(),
		calendar:      NewCalendar(),
		buttonWidth:   DatePickerButtonWidth,
		year:          now.Year(),
		month:         int(now.Month()),
		day:           now.Day(),
		hasDate:       true,
	}
	// Parent the inner widgets to the OUTER widget: the parent chain is what the
	// cursor walk, focus, layers and theme lookups climb, and parenting to the
	// embedded container would stop at it.
	d.AddChild(d.field)
	d.AddChild(d.button)
	d.field.SetParent(d)
	d.button.SetParent(d)

	d.button.SetIcon("calendar")
	d.button.SetTooltip("Pick a date")
	d.button.SetOnClicked(func() { d.SetExpanded(!d.panel.isOpen()) })
	d.field.SetOnTextChanged(func(text string) { d.textChanged(text) })
	d.field.SetOnSubmit(func(text string) { d.commitText(text) })
	d.calendar.SetOnDateSelected(func(year, month, day int) {
		d.setDate(year, month, day)
		d.hasDate = true
		// The field is the widget's text: a day picked in the panel must land
		// there, not only in the value.
		d.field.SetText(d.dateText())
		d.fireDateChanged()
		d.SetExpanded(false)
	})

	d.field.SetText(now.Format(datePickerCanonical))
	d.calendar.SetDate(d.year, d.month, d.day)
	d.SetHeight(DatePickerHeight)
	return d
}

// SetDate sets the value (and the field) without firing OnDateChanged: a
// programmatic set is not a user edit.
func (d *DatePicker) SetDate(year, month, day int) *DatePicker {
	d.setDate(year, month, day)
	d.hasDate = true
	d.field.SetText(d.dateText())
	d.calendar.SetDate(d.year, d.month, d.day)
	d.MarkForRedraw()
	return d
}

// Date returns the current year, month and day. HasDate says whether there is
// one: after Clear the field is empty and the returned value is the last date
// the picker had, which is where its calendar would open.
func (d *DatePicker) Date() (year, month, day int) {
	d.year, d.month, d.day = d.normalise(d.year, d.month, d.day)
	return d.year, d.month, d.day
}

// HasDate reports whether the picker holds a date. An unset picker is the empty
// field an optional "from/to" filter wants.
func (d *DatePicker) HasDate() bool { return d.hasDate }

// Clear unsets the picker: the field empties (showing its placeholder) and
// HasDate goes false, without firing OnDateChanged. The calendar keeps the date
// it had, so opening it lands on something sensible.
func (d *DatePicker) Clear() *DatePicker {
	d.hasDate = false
	d.field.SetText("")
	d.MarkForRedraw()
	return d
}

// propertyInt reads a whole-number QML property. The parser hands back an int
// for "9" and a float64 for "9.0", so a widget that accepts only one of them
// silently ignores the other — which is how a QML `hour: 9` did nothing.
func propertyInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case float32:
		return int(v), true
	}
	return 0, false
}

// Calendar returns the popup's calendar, for the caller that wants to configure
// it. It is the same widget the panel paints; the field gives it its rect when
// the panel opens.
func (d *DatePicker) Calendar() *Calendar { return d.calendar }

// SetOnDateChanged registers the callback fired when the user commits a date —
// by picking a day or by pressing Enter on a valid one.
func (d *DatePicker) SetOnDateChanged(handler func(year, month, day int)) *DatePicker {
	d.onDateChanged = handler
	return d
}

// SetText sets the field from text — the QML `text` property. A value that is
// not a date is shown as typed and leaves the current date in place.
func (d *DatePicker) SetText(text string) *DatePicker {
	d.field.SetText(text)
	d.textChanged(text)
	return d
}

// Text returns the field's text, exactly as shown.
func (d *DatePicker) Text() string { return d.field.Text }

// SetPlaceholder sets the text shown while the field is empty.
func (d *DatePicker) SetPlaceholder(text string) *DatePicker {
	d.field.SetPlaceholder(text)
	return d
}

// SetEnabled enables or disables the field and its button together, so a
// disabled picker cannot still open its calendar.
func (d *DatePicker) SetEnabled(enabled bool) *DatePicker {
	d.BaseWidget.SetEnabled(enabled)
	d.field.SetEnabled(enabled)
	d.button.SetEnabled(enabled)
	if !enabled {
		d.SetExpanded(false)
	}
	return d
}

// SetPresentation selects where the calendar panel appears. Auto — the default — follows the
// surface's size class: anchored under the field on a surface with room, a bottom sheet on a phone.
func (d *DatePicker) SetPresentation(p PickerPresentation) *DatePicker {
	d.presentation = p
	return d
}

func (d *DatePicker) Presentation() PickerPresentation { return d.presentation }

// sheetFor reports whether this picker takes the sheet form on a surface of the given size.
func (d *DatePicker) sheetFor(width, height float32) bool {
	switch d.presentation {
	case PickerDropdown:
		return false
	case PickerSheet:
		return true
	default:
		return platform.FormFactorFor(width, height, 1) == platform.FormFactorPhone
	}
}

// PanelRect is the open panel's rect in the window's content space, or the zero rect when it is
// closed. It exists so a caller — and a test — can ask WHERE the panel landed without reaching
// into the layer.
func (d *DatePicker) PanelRect() graphics.Rect {
	if !d.panel.isOpen() {
		return graphics.Rect{}
	}
	return graphics.NewRect(d.panel.offsetX, d.panel.offsetY, d.panel.width, d.panel.height)
}

// Expanded reports whether the calendar panel is open.
func (d *DatePicker) Expanded() bool { return d.panel.isOpen() }

// SetExpanded opens or closes the calendar panel.
func (d *DatePicker) SetExpanded(expanded bool) *DatePicker {
	if expanded == d.panel.isOpen() {
		return d
	}
	if !expanded {
		d.panel.closePanel(d)
		d.MarkForRedraw()
		return d
	}
	if !d.Enabled() {
		return d
	}
	d.calendar.SetDate(d.Date())
	w, h := datePickerPanelWidth, datePickerPanelHeight
	sheet := false
	if win := windowOf(d); win != nil {
		wf := win.FrameRect()
		if sheet = d.sheetFor(wf.Width, wf.Height); sheet {
			// A sheet is as wide as the surface allows and as tall as the calendar needs, plus
			// the padding a thumb wants at the bottom.
			w = sheetPanelWidth(wf.Width)
			h = datePickerPanelHeight + 2*sheetPanelPad
		}
	}
	var ok bool
	if sheet {
		ok = d.panel.openSheet(d, w, h, d.paintCalendar)
	} else {
		ok = d.panel.openPanel(d, w, h, d.paintCalendar)
	}
	if !ok {
		return d
	}
	d.MarkForRedraw()
	return d
}

// paintCalendar draws the calendar into the panel, giving it the panel's rect:
// the calendar adapts its cells to whatever rect it is painted with.
func (d *DatePicker) paintCalendar(canvas *graphics.Canvas, panel graphics.Rect) {
	d.calendar.SetFrameRect(panel)
	d.calendar.Paint(canvas, panel)
}

// ── value and text ──────────────────────────────────────────────────

// setDate stores the date, normalised through time.Date so an out-of-range
// month or day lands on a real day instead of being stored as nonsense.
func (d *DatePicker) setDate(year, month, day int) {
	d.year, d.month, d.day = d.normalise(year, month, day)
}

func (d *DatePicker) normalise(year, month, day int) (int, int, int) {
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	return t.Year(), int(t.Month()), t.Day()
}

func (d *DatePicker) dateText() string {
	return time.Date(d.year, time.Month(d.month), d.day, 0, 0, 0, 0, time.UTC).
		Format(datePickerCanonical)
}

// parseDate accepts the typed forms above.
func parseDate(text string) (year, month, day int, ok bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, 0, 0, false
	}
	for _, layout := range datePickerLayouts {
		if t, err := time.Parse(layout, text); err == nil {
			return t.Year(), int(t.Month()), t.Day(), true
		}
	}
	return 0, 0, 0, false
}

// textChanged tracks the field while the user types: a parseable date takes
// effect immediately (the day can be picked and the callback fires), but the
// text is left exactly as typed until commit, so an intermediate "2026-0" is
// not rewritten under the cursor.
func (d *DatePicker) textChanged(text string) {
	if strings.TrimSpace(text) == "" {
		// An emptied field is "no date", not a stale one.
		d.hasDate = false
		return
	}
	year, month, day, ok := parseDate(text)
	if !ok {
		return
	}
	changed := year != d.year || month != d.month || day != d.day
	d.setDate(year, month, day)
	d.hasDate = true
	d.calendar.SetDate(d.year, d.month, d.day)
	if changed {
		d.fireDateChanged()
	}
}

// commitText normalises the field on Enter: a valid date becomes YYYY-MM-DD, an
// unusable text goes back to the last good date rather than being left as a
// half-typed value. Either way the calendar panel closes.
func (d *DatePicker) commitText(text string) {
	if year, month, day, ok := parseDate(text); ok {
		d.setDate(year, month, day)
		d.hasDate = true
		d.fireDateChanged()
	} else if strings.TrimSpace(text) == "" {
		// Enter on an empty field clears it rather than restoring the old date.
		d.hasDate = false
		d.calendar.SetDate(d.year, d.month, d.day)
		d.SetExpanded(false)
		return
	}
	d.field.SetText(d.dateText())
	d.calendar.SetDate(d.year, d.month, d.day)
	d.SetExpanded(false)
}

func (d *DatePicker) fireDateChanged() {
	if d.onDateChanged != nil {
		y, m, dd := d.Date()
		d.onDateChanged(y, m, dd)
	}
}

// ── layout, painting and events ─────────────────────────────────────

func (d *DatePicker) SetFrameRect(rect graphics.Rect) {
	d.BaseWidget.SetFrameRect(rect)
	d.SetNeedsLayout()
}

// ValidateLayout places the field and the button inside the frame.
func (d *DatePicker) ValidateLayout() {
	frame := d.FrameRect()
	if frame.Width <= 0 || frame.Height <= 0 {
		d.ClearNeedsLayout()
		return
	}
	btnW := d.buttonWidth
	if btnW > frame.Width/2 {
		btnW = frame.Width / 2
	}
	d.field.SetFrameRect(graphics.NewRect(0, 0, frame.Width-btnW, frame.Height))
	d.button.SetFrameRect(graphics.NewRect(frame.Width-btnW, 0, btnW, frame.Height))
	d.ClearNeedsLayout()
}

func (d *DatePicker) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !d.Visible() {
		return
	}
	if d.NeedsLayout() {
		d.ValidateLayout()
	}
	d.field.Paint(canvas, d.field.FrameRect())
	d.button.Paint(canvas, d.button.FrameRect())
}

// PanelAt routes a point to the field or the button, so the window's hit test
// finds them: the field owns text editing, the button owns the click that opens
// the calendar. The calendar itself lives in the layer, which the window routes
// back here through HandleMouseDown/Move/Up.
func (d *DatePicker) PanelAt(x, y float32) meui.Widget {
	frame := d.FrameRect()
	if x < 0 || y < 0 || x > frame.Width || y > frame.Height {
		return nil
	}
	if bf := d.button.FrameRect(); x >= bf.X && x <= bf.X+bf.Width {
		return d.button
	}
	return d.field
}

// OnMouseDown, OnMouseUp and OnMouseMove forward to the field or the button
// under the point. A container dispatches by containment — a modal popup hands
// the press to its content's children, not to the deepest hit — so the composite
// must route the event itself; relying on the window's PanelAt delivery alone
// left every field inside a modal dead (the owner's "the colour picker does not
// pop up" in the Settings popup).
func (d *DatePicker) OnMouseDown(x, y float32, button int) bool {
	target := d.PanelAt(x, y)
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

func (d *DatePicker) OnMouseUp(x, y float32, button int) bool {
	target := d.PanelAt(x, y)
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

func (d *DatePicker) OnMouseMove(x, y float32) {
	target := d.PanelAt(x, y)
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
// inside the panel the calendar gets the click, anywhere else closes it.
func (d *DatePicker) HandleMouseDown(lx, ly float32, button int) bool {
	if !d.panel.isOpen() {
		return false
	}
	px, py, inside := d.panel.inside(lx, ly)
	d.panelPressed = true
	d.panelPressedInside = inside
	if !inside {
		// Do not close on the press: the owner's button opens on the RELEASE, so
		// closing here lets the same press toggle the panel straight back open
		// (the button's click arrives after the panel is already gone). The
		// release closes it — HandleMouseUp below.
		return true
	}
	d.calendar.SetFrameRect(graphics.NewRect(0, 0, d.panel.width, d.panel.height))
	d.calendar.OnMouseDown(px, py, button)
	return true
}

func (d *DatePicker) HandleMouseMove(lx, ly float32) bool {
	if !d.panel.isOpen() {
		return false
	}
	px, py, inside := d.panel.inside(lx, ly)
	if !inside {
		return true
	}
	d.calendar.SetFrameRect(graphics.NewRect(0, 0, d.panel.width, d.panel.height))
	d.calendar.OnMouseMove(px, py)
	return true
}

func (d *DatePicker) HandleMouseUp(lx, ly float32, button int) bool {
	if !d.panel.isOpen() {
		return false
	}
	pressed := d.panelPressed
	d.panelPressed = false
	px, py, inside := d.panel.inside(lx, ly)
	if !inside {
		if pressed && !d.panelPressedInside {
			d.SetExpanded(false)
		}
		return true
	}
	// The calendar acts on the press (a day cell is a click target), so the
	// release is only consumed here.
	_, _ = px, py
	_ = button
	return true
}

// SetBoundProperty applies a QML property to the picker. The loader routes
// every property without a handler of its own here (see qml.HandleProperty), so
// this is what makes `placeholder`, `year`, `month` and `day` work in QML.
func (d *DatePicker) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "placeholder":
		if s, ok := value.(string); ok {
			d.SetPlaceholder(s)
		}
	case "year", "month", "day":
		v, ok := propertyInt(value)
		if !ok {
			return
		}
		y, m, day := d.Date()
		switch propName {
		case "year":
			y = int(v)
		case "month":
			m = int(v)
		case "day":
			day = int(v)
		}
		d.SetDate(y, m, day)
	}
}

// CursorAt returns nil: the deepest hit under the pointer is the field, whose
// own CursorAt reports the text cursor, or the button, which keeps the arrow.
func (d *DatePicker) CursorAt(x, y float32) *platform.Cursor { return nil }
