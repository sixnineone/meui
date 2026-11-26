package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
)

// newPickerWindow puts a widget in a real window so its dropdown panel has a
// layer manager to live in (the panels are window-space layers, as ComboBox's
// dropdown is).
func newPickerWindow(t *testing.T, w meui.Widget, width, height float32) *meui.Window {
	t.Helper()
	// These tests exercise the panel's INTERACTION, and the window below is the default 400x300 —
	// which is a PHONE by the size-class rule, so `Auto` would lift the panel into a bottom sheet
	// and every coordinate in them would miss. Pin the dropdown here; the sheet presentation has
	// its own tests (harness/behaviour/picker_sheet_test.go).
	switch v := w.(type) {
	case *DatePicker:
		v.SetPresentation(PickerDropdown)
	case *TimePicker:
		v.SetPresentation(PickerDropdown)
	}
	win := meui.NewWindow()
	win.AddChild(w)
	w.SetFrameRect(graphics.NewRect(20, 30, width, height))
	if vl, ok := w.(interface{ ValidateLayout() }); ok {
		vl.ValidateLayout()
	}
	return win
}

func TestDatePickerFieldAndValue(t *testing.T) {
	d := NewDatePicker()
	d.SetDate(2026, 9, 17)
	if got := d.field.Text; got != "2026-09-17" {
		t.Errorf("field text = %q, want 2026-09-17", got)
	}
	if y, m, dd := d.Date(); y != 2026 || m != 9 || dd != 17 {
		t.Errorf("Date() = %d-%d-%d, want 2026-9-17", y, m, dd)
	}
	// A programmatic set is not a user edit.
	fired := 0
	d.SetOnDateChanged(func(int, int, int) { fired++ })
	d.SetDate(2026, 1, 2)
	if fired != 0 {
		t.Errorf("SetDate fired OnDateChanged %d times, want 0", fired)
	}
}

// The typed forms all mean the same day, and a commit normalises the text.
func TestDatePickerParsesAndNormalisesTypedDates(t *testing.T) {
	d := NewDatePicker()
	var got []string
	d.SetOnDateChanged(func(y, m, day int) {
		got = append(got, d.dateText())
	})

	for _, typed := range []string{"2026-9-5", "2026/09/05", "2026.9.5", "2026-09-05"} {
		d.field.SetText(typed)
		d.textChanged(typed)
		d.commitText(typed)
		if d.field.Text != "2026-09-05" {
			t.Fatalf("typing %q left the field as %q, want the canonical 2026-09-05", typed, d.field.Text)
		}
	}
	if y, m, day := d.Date(); y != 2026 || m != 9 || day != 5 {
		t.Fatalf("Date() = %d-%d-%d after the commits, want 2026-9-5", y, m, day)
	}
	if len(got) == 0 {
		t.Fatal("OnDateChanged never fired for a valid typed date")
	}

	// Text that is not a date must not move the value: the field goes back to
	// the last good date instead of keeping a half-typed value.
	d.commitText("not a date")
	if y, m, day := d.Date(); y != 2026 || m != 9 || day != 5 {
		t.Errorf("Date() = %d-%d-%d after invalid text, want the previous 2026-9-5", y, m, day)
	}
	if d.field.Text != "2026-09-05" {
		t.Errorf("field text = %q after invalid text, want the previous date back", d.field.Text)
	}
	// A day that does not exist is invalid text, not a rollover.
	d.commitText("2026-02-30")
	if y, m, day := d.Date(); y != 2026 || m != 9 || day != 5 {
		t.Errorf("Date() = %d-%d-%d after 2026-02-30, want the previous date kept", y, m, day)
	}
}

// The button opens the calendar panel and a click elsewhere closes it.
func TestDatePickerButtonOpensTheCalendarPanel(t *testing.T) {
	d := NewDatePicker()
	win := newPickerWindow(t, d, 160, DatePickerHeight)

	// The button's own coordinates are 0-based (the window routes events in
	// widget-local space); its frame X only places it for painting.
	clickButton := func() {
		bf := d.button.FrameRect()
		cx, cy := bf.Width/2, bf.Height/2
		d.button.OnMouseDown(cx, cy, 0)
		d.button.OnMouseUp(cx, cy, 0)
	}

	clickButton()
	if !d.Expanded() {
		t.Fatal("clicking the button did not open the calendar panel")
	}
	if got := len(win.LayerManager.Layers); got != 1 {
		t.Fatalf("layers = %d, want the panel's one window layer", got)
	}

	// A click far outside the panel dismisses it (the layer is modal); the
	// dismissal happens on the release, so the button's own click cannot toggle
	// it straight back open.
	if !d.HandleMouseDown(500, 500, 0) || !d.HandleMouseUp(500, 500, 0) {
		t.Error("a click outside the panel was not consumed by the open panel")
	}
	if d.Expanded() {
		t.Error("a click outside the panel left it open")
	}
	if got := len(win.LayerManager.Layers); got != 0 {
		t.Errorf("layers = %d after closing, want 0", got)
	}

	// Clicking the button again toggles it shut as well.
	clickButton()
	if !d.Expanded() {
		t.Fatal("second click did not reopen the panel")
	}
	clickButton()
	if d.Expanded() {
		t.Error("clicking the button while open left the panel open")
	}
}

// Picking a day in the panel sets the value, the text and fires the callback,
// then closes the panel.
func TestDatePickerPickingADayUpdatesTheField(t *testing.T) {
	d := NewDatePicker()
	d.SetDate(2026, 9, 17)
	newPickerWindow(t, d, 160, DatePickerHeight)
	d.SetExpanded(true)
	if !d.Expanded() {
		t.Fatal("the calendar panel did not open")
	}

	var gotYear, gotMonth, gotDay int
	d.SetOnDateChanged(func(y, m, day int) { gotYear, gotMonth, gotDay = y, m, day })

	// A point in the middle of the calendar grid is some day of the shown
	// month; click it through the panel and compare with what the calendar
	// itself resolved, so the assertion does not hardcode the layout.
	layer := d.panel.layer
	clickX, clickY := layer.X+d.panel.offsetX+d.panel.width/2, layer.Y+d.panel.offsetY+d.panel.height/2
	d.HandleMouseDown(clickX, clickY, 0)
	cy, cm, cd := d.calendar.GetSelectedDate()
	if cy != gotYear || cm != gotMonth || cd != gotDay {
		t.Fatalf("OnDateChanged got %d-%d-%d, want the calendar's selected %d-%d-%d", gotYear, gotMonth, gotDay, cy, cm, cd)
	}
	if y, m, day := d.Date(); y != cy || m != cm || day != cd {
		t.Errorf("Date() = %d-%d-%d, want the picked %d-%d-%d", y, m, day, cy, cm, cd)
	}
	if d.field.Text != d.dateText() {
		t.Errorf("field text = %q, want the picked date %q", d.field.Text, d.dateText())
	}
	if d.Expanded() {
		t.Error("the calendar panel stayed open after picking a day")
	}
}

// A picker in no window must not panic when its button is clicked.
func TestDatePickerWithoutAWindowIsInert(t *testing.T) {
	d := NewDatePicker()
	d.SetFrameRect(graphics.NewRect(0, 0, 160, DatePickerHeight))
	d.SetExpanded(true)
	if d.Expanded() {
		t.Error("a picker with no window opened a panel")
	}
}

func TestColorFieldTextAndSwatchStayInSync(t *testing.T) {
	c := NewColorField()
	c.SetColor(graphics.RGBA(0x11, 0x22, 0x33, 0xFF))
	if got := c.field.Text; got != "#112233" {
		t.Errorf("field text = %q, want #112233", got)
	}
	if got := c.Hex(); got != "#112233" {
		t.Errorf("Hex() = %q, want #112233", got)
	}

	// Translucent colours keep their alpha pair.
	c.SetColor(graphics.RGBA(0x11, 0x22, 0x33, 0x80))
	if got := c.field.Text; got != "#11223380" {
		t.Errorf("field text = %q, want #11223380", got)
	}
}

// Typing a valid hex takes effect immediately; a six-digit value keeps the
// alpha the field already had; text that is not a colour changes nothing.
func TestColorFieldTypedText(t *testing.T) {
	c := NewColorField()
	c.SetColor(graphics.RGBA(0x11, 0x22, 0x33, 0x80))

	var last graphics.Color
	fired := 0
	c.SetOnColorChanged(func(color graphics.Color) { last = color; fired++ })

	c.field.SetText("#445566")
	c.textChanged()
	if got := c.Hex(); got != "#44556680" {
		t.Fatalf("Hex() = %q, want #44556680 (a six-digit value keeps the alpha)", got)
	}
	if fired != 1 || last != graphics.RGBA(0x44, 0x55, 0x66, 0x80) {
		t.Errorf("OnColorChanged fired %d times, last %08X", fired, uint32(last))
	}

	// Eight digits set the alpha explicitly.
	c.field.SetText("#445566FF")
	c.textChanged()
	if got := c.Hex(); got != "#445566" {
		t.Errorf("Hex() = %q, want #445566 after an opaque eight-digit value", got)
	}

	// Invalid text is ignored, not applied as zero.
	c.field.SetText("#GG")
	c.textChanged()
	if got := c.Hex(); got != "#445566" {
		t.Errorf("Hex() = %q after invalid text, want the previous colour kept", got)
	}
	if fired != 2 {
		t.Errorf("OnColorChanged fired %d times, want 2 (invalid text must not fire)", fired)
	}

	// A commit canonicalises the text (no leading '#' accepted).
	c.field.SetText("AABBCC")
	c.commitText()
	if c.field.Text != "#AABBCC" || c.Hex() != "#AABBCC" {
		t.Errorf("commit left %q / %q, want the canonical #AABBCC", c.field.Text, c.Hex())
	}
}

func TestColorFieldButtonOpensThePickerPanel(t *testing.T) {
	c := NewColorField()
	win := newPickerWindow(t, c, 160, ColorFieldHeight)

	bf := c.button.FrameRect()
	c.button.OnMouseDown(bf.Width/2, bf.Height/2, 0)
	c.button.OnMouseUp(bf.Width/2, bf.Height/2, 0)
	if !c.Expanded() {
		t.Fatal("clicking the swatch button did not open the picker panel")
	}
	if len(win.LayerManager.Layers) != 1 {
		t.Fatalf("layers = %d, want the panel's one window layer", len(win.LayerManager.Layers))
	}

	// Dragging in the panel picks a colour live: the field follows the picker.
	c.SetColor(graphics.RGBA(0x11, 0x22, 0x33, 0xFF))
	before := c.Hex()
	px := c.panel.layer.X + c.panel.offsetX + c.panel.width/2
	py := c.panel.layer.Y + c.panel.offsetY + c.panel.height/2
	c.HandleMouseDown(px, py, 0)
	c.HandleMouseMove(px+20, py+10)
	if c.Hex() == before {
		t.Errorf("Hex() = %q after picking in the panel, want the picked colour to be reflected in the field", c.Hex())
	}
	if c.field.Text != c.Hex() {
		t.Errorf("field text = %q and Hex() = %q, want them equal after a pick", c.field.Text, c.Hex())
	}

	c.HandleMouseDown(500, 500, 0)
	c.HandleMouseUp(500, 500, 0)
	if c.Expanded() {
		t.Error("a click outside the panel left it open")
	}
}

// The codec is the widget's contract with its callers: round trip every form.
func TestHexColorCodec(t *testing.T) {
	opaque := graphics.RGBA(0x12, 0x34, 0x56, 0xFF)
	if got := FormatHexColor(opaque); got != "#123456" {
		t.Errorf("FormatHexColor(opaque) = %q, want #123456", got)
	}
	if got, err := ParseHexColor("#123456", graphics.RGBA(0, 0, 0, 0x40)); err != nil || got != graphics.RGBA(0x12, 0x34, 0x56, 0x40) {
		t.Errorf("ParseHexColor(#123456) = %08X (err %v), want the six digits with the current alpha kept", uint32(got), err)
	}
	translucent := graphics.RGBA(0x12, 0x34, 0x56, 0x78)
	if got := FormatHexColor(translucent); got != "#12345678" {
		t.Errorf("FormatHexColor(translucent) = %q, want #12345678", got)
	}
	if got, err := ParseHexColor(FormatHexColor(translucent), 0); err != nil || got != translucent {
		t.Errorf("round trip = %08X (err %v), want %08X", uint32(got), err, uint32(translucent))
	}
	for _, bad := range []string{"", "#", "#12345", "#1234567", "#GGGGGG", "red"} {
		if _, err := ParseHexColor(bad, 0); err == nil {
			t.Errorf("ParseHexColor(%q) succeeded, want an error", bad)
		}
	}
}

// An optional date field: Clear unsets it, typing or picking a date sets it,
// and emptying the text unsets it again.
func TestDatePickerUnsetState(t *testing.T) {
	d := NewDatePicker()
	d.SetDate(2026, 5, 6)
	if !d.HasDate() {
		t.Fatal("a picker with a date reports HasDate false")
	}
	d.Clear()
	if d.HasDate() {
		t.Error("Clear left HasDate true")
	}
	if d.field.Text != "" {
		t.Errorf("field text = %q after Clear, want empty", d.field.Text)
	}
	// The calendar still knows where to open.
	if y, m, day := d.calendar.GetSelectedDate(); y != 2026 || m != 5 || day != 6 {
		t.Errorf("calendar date = %d-%d-%d after Clear, want it kept at 2026-5-6", y, m, day)
	}
	// A programmatic Clear is not a user edit.
	fired := 0
	d.SetOnDateChanged(func(int, int, int) { fired++ })
	d.Clear()
	if fired != 0 {
		t.Errorf("Clear fired OnDateChanged %d times, want 0", fired)
	}

	d.SetText("2026-07-08")
	if !d.HasDate() {
		t.Error("SetText with a date left HasDate false")
	}
	d.SetText("")
	if d.HasDate() {
		t.Error("emptying the text left HasDate true")
	}
}

// A ColorField inside a modal must still open its picker: the modal is itself a
// window layer (z 9000) and the field's panel is another (z 10000), so the
// panel has to be created *and* land above the modal it was opened from. This is
// the app's real structure — the Settings popup is a modal whose Charts tab is
// built from these fields — and the owner's report that the picker "does not pop
// up" is what this pins.
func TestColorFieldInsideAModalOpensItsPanelAboveIt(t *testing.T) {
	win := meui.NewWindow()
	win.SetFrameRect(graphics.NewRect(0, 0, 900, 700))
	cf := NewColorField()
	dp := NewDatePicker()

	modalContent := meui.NewPanel()
	modalContent.AddChild(cf)
	modalContent.AddChild(dp)
	okBtn := NewButton()
	okBtn.SetText("OK")
	popup := NewModalPopupPanel("Settings", "test", modalContent, okBtn)
	popup.Modal = win.ShowModal(popup, 500, 400)
	if popup.Modal == nil {
		t.Fatal("the modal did not open")
	}
	modalLayer := popup.Modal.ContentLayer
	if modalLayer == nil {
		t.Fatal("the modal has no content layer")
	}

	// The host paints the layers every frame, and that is what gives the modal's
	// content its body rect — the click routing compares against it, so a modal
	// that has never painted is not clickable at all.
	dev := graphics.NewSoftDevice(900, 700)
	win.PaintLayerOverlay(graphics.NewCanvas(dev), 0)

	// The field's own rect inside the content, as the content's layout sets it.
	cf.SetFrameRect(graphics.NewRect(20, 20, 170, ColorFieldHeight))
	cf.ValidateLayout()

	// The date picker in the same modal, to cover both composites' forwarding.
	dp.SetFrameRect(graphics.NewRect(20, 60, 170, DatePickerHeight))
	dp.ValidateLayout()

	// Click each control's button the way the window routes a click: the point
	// comes from the widget's own coordinate space, and the window converts.
	clickButton := func(target meui.Widget, localX, localY float32) {
		pt := target.(interface {
			PointToRoot(float32, float32) graphics.Point
		}).PointToRoot(localX, localY)
		win.HandleEvent(&platform.Event{Type: platform.EventMouseDown, Pos: pt, Button: 0})
		win.HandleEvent(&platform.Event{Type: platform.EventMouseUp, Pos: pt, Button: 0})
	}
	// One at a time: an open panel is modal, so the next click anywhere else
	// closes it (which is also why each assertion waits for its own panel).
	dbf := dp.button.FrameRect()
	clickButton(dp.button, dbf.Width/2, dbf.Height/2)
	if !dp.Expanded() {
		t.Fatal("clicking the calendar button inside the modal did not open the calendar panel")
	}
	clickButton(dp.button, dbf.Width/2, dbf.Height/2) // toggles shut
	if dp.Expanded() {
		t.Fatal("the calendar panel did not close on the second click")
	}

	cbf := cf.button.FrameRect()
	clickButton(cf.button, cbf.Width/2, cbf.Height/2)
	if !cf.Expanded() {
		t.Fatal("clicking the swatch inside the modal did not open the picker panel")
	}
	panelLayer := cf.panel.layer
	if panelLayer == nil {
		t.Fatal("the picker panel has no layer")
	}
	if panelLayer.ZIndex <= modalLayer.ZIndex {
		t.Errorf("picker layer z=%d is not above the modal layer z=%d: it would paint under the popup",
			panelLayer.ZIndex, modalLayer.ZIndex)
	}
}

// Clicking the button through the WINDOW must leave the panel open. It did not:
// the window offers every release to the layers twice, and the second offer came
// after the button's release had opened the panel — so the release closed what
// its own press had just opened, and the picker looked like it never opened
// (the owner's report: the colour picker worked in the modal, the date picker in
// a plain window did not).
func TestFieldButtonOpensItsPanelThroughTheWindow(t *testing.T) {
	for _, tc := range []struct {
		name   string
		widget meui.Widget
	}{
		{name: "DatePicker", widget: NewDatePicker()},
		{name: "ColorField", widget: NewColorField()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			win := meui.NewWindow()
			win.SetFrameRect(graphics.NewRect(0, 0, 900, 700))
			win.AddChild(tc.widget)
			tc.widget.SetFrameRect(graphics.NewRect(20, 20, 170, DatePickerHeight))
			if vl, ok := tc.widget.(interface{ ValidateLayout() }); ok {
				vl.ValidateLayout()
			}

			// The button, in its own coordinates, through the window's routing.
			var btn *Button
			expanded := func() bool { return false }
			switch w := tc.widget.(type) {
			case *DatePicker:
				btn, expanded = w.button, w.Expanded
			case *ColorField:
				btn, expanded = w.button, w.Expanded
			}
			bf := btn.FrameRect()
			pt := btn.PointToRoot(bf.Width/2, bf.Height/2)
			win.HandleEvent(&platform.Event{Type: platform.EventMouseDown, Pos: pt, Button: 0})
			win.HandleEvent(&platform.Event{Type: platform.EventMouseUp, Pos: pt, Button: 0})

			if !expanded() {
				t.Fatalf("%s: the panel is not open after clicking its button through the window", tc.name)
			}

			// Opening is not enough: the panel must paint. Its content (the
			// calendar, the picker) adapts to the rect it is given, so a panel
			// with no rect drawn opens and shows nothing.
			dev := graphics.NewSoftDevice(900, 700)
			win.PaintLayerOverlay(graphics.NewCanvas(dev), 0)
			var panel dropdownPanel
			switch w := tc.widget.(type) {
			case *DatePicker:
				panel = w.panel
			case *ColorField:
				panel = w.panel
			}
			if painted := countPaintedPixels(dev.Pixels(), 900, panel.offsetX, panel.offsetY, panel.width, panel.height); painted == 0 {
				t.Fatalf("%s: the panel opened at (%.0f,%.0f %.0fx%.0f) but painted nothing",
					tc.name, panel.offsetX, panel.offsetY, panel.width, panel.height)
			}
			// And the same click again closes it.
			win.HandleEvent(&platform.Event{Type: platform.EventMouseDown, Pos: pt, Button: 0})
			win.HandleEvent(&platform.Event{Type: platform.EventMouseUp, Pos: pt, Button: 0})
			if expanded() {
				t.Fatalf("%s: the panel stayed open after a second click on its button", tc.name)
			}
		})
	}
}
