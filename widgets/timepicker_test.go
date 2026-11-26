package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
)

func TestTimePickerFieldAndValue(t *testing.T) {
	p := NewTimePicker()
	p.SetTime(9, 5, 0)
	if got := p.field.Text; got != "09:05" {
		t.Errorf("field text = %q, want 09:05", got)
	}
	if h, m, s := p.Time(); h != 9 || m != 5 || s != 0 {
		t.Errorf("Time() = %d:%d:%d, want 9:5:0", h, m, s)
	}
	// A programmatic set is not a user edit.
	fired := 0
	p.SetOnTimeChanged(func(int, int, int) { fired++ })
	p.SetTime(14, 30, 15)
	if fired != 0 {
		t.Errorf("SetTime fired OnTimeChanged %d times, want 0", fired)
	}
	// Seconds are carried but shown only when asked.
	if p.Text() != "14:30" {
		t.Errorf("text = %q, want the seconds hidden", p.Text())
	}
	p.SetShowSeconds(true)
	if p.Text() != "14:30:15" {
		t.Errorf("text with seconds = %q, want 14:30:15", p.Text())
	}
}

// The accepted typed forms, and the commit normalising what it accepts.
func TestTimePickerParsesAndNormalisesTypedTimes(t *testing.T) {
	p := NewTimePicker()
	var seen []string
	p.SetOnTimeChanged(func(h, m, s int) { seen = append(seen, p.timeText()) })

	for _, typed := range []string{"9:05", "09:05", "9:05:00", "09:05:00"} {
		p.field.SetText(typed)
		p.textChanged(typed)
		p.commitText(typed)
		if p.field.Text != "09:05" {
			t.Fatalf("typing %q left the field as %q, want the canonical 09:05", typed, p.field.Text)
		}
	}
	if h, m, _ := p.Time(); h != 9 || m != 5 {
		t.Fatalf("Time() = %d:%d after the commits, want 9:05", h, m)
	}
	if len(seen) == 0 {
		t.Fatal("OnTimeChanged never fired for a valid typed time")
	}

	// Text that is not a time must not move the value: the field goes back to
	// the last good time.
	p.commitText("half past nine")
	if h, m, _ := p.Time(); h != 9 || m != 5 {
		t.Errorf("Time() = %d:%d after invalid text, want the previous 9:05", h, m)
	}
	if p.field.Text != "09:05" {
		t.Errorf("field text = %q after invalid text, want the previous time back", p.field.Text)
	}
	// A typed value outside the clock is invalid text, not a rollover — the same
	// rule the date picker applies to 2026-02-30.
	p.commitText("25:70")
	if h, m, _ := p.Time(); h != 9 || m != 5 {
		t.Errorf("Time() = %d:%d for typed 25:70, want the previous 9:05 kept", h, m)
	}
	// A PROGRAMMATIC set is normalised through a real clock instead, so an
	// out-of-range value lands on a real time rather than being stored as is.
	p.SetTime(25, 70, 0)
	if h, m, _ := p.Time(); h != 2 || m != 10 {
		t.Errorf("SetTime(25,70) = %d:%d, want it wrapped to 2:10", h, m)
	}
	if p.Text() != "02:10" {
		t.Errorf("text after SetTime(25,70) = %q, want 02:10", p.Text())
	}
}

// An optional time, like the date picker: Clear unsets it, typing sets it.
func TestTimePickerUnsetState(t *testing.T) {
	p := NewTimePicker()
	p.SetTime(6, 0, 0)
	p.Clear()
	if p.HasTime() {
		t.Error("Clear left HasTime true")
	}
	if p.field.Text != "" {
		t.Errorf("field text = %q after Clear, want empty", p.field.Text)
	}
	fired := 0
	p.SetOnTimeChanged(func(int, int, int) { fired++ })
	p.Clear()
	if fired != 0 {
		t.Errorf("Clear fired OnTimeChanged %d times, want 0", fired)
	}
	p.SetText("07:15")
	if !p.HasTime() || p.Text() != "07:15" {
		t.Errorf("SetText(07:15) left HasTime=%v text=%q", p.HasTime(), p.Text())
	}
	p.SetText("")
	if p.HasTime() {
		t.Error("emptying the text left HasTime true")
	}
}

// Clicking a grid cell sets that value and closes the panel, so the hour and the
// minute are one click each — the owner's rule: a click on the panel closes it.
func TestTimePickerPanelPicksHoursAndMinutes(t *testing.T) {
	p := NewTimePicker()
	p.SetTime(0, 0, 0)
	newPickerWindow(t, p, 160, TimePickerHeight)

	var gotH, gotM, fired int
	p.SetOnTimeChanged(func(h, m, s int) { gotH, gotM, fired = h, m, fired+1 })

	clickCell := func(hours bool, col, row int) {
		p.SetExpanded(true)
		if !p.Expanded() {
			t.Fatal("the time panel did not open")
		}
		cell := p.gridPanel.cellRect(hours, col, row)
		// The panel's offset is only known once it is open.
		p.HandleMouseDown(cell.X+cell.Width/2+p.panel.offsetX, cell.Y+cell.Height/2+p.panel.offsetY, 0)
		if p.Expanded() {
			t.Fatalf("the panel stayed open after picking (%v,%d,%d)", hours, col, row)
		}
	}

	// The cell for 17 is row 5 of the second column (12 + 5); 45 is row 9 of the
	// fourth minute column (36 + 9).
	clickCell(true, 1, 5)
	clickCell(false, 3, 9)

	if fired != 2 {
		t.Fatalf("OnTimeChanged fired %d times, want one per picked cell", fired)
	}
	if gotH != 17 || gotM != 45 {
		t.Errorf("picked %02d:%02d, want 17:45", gotH, gotM)
	}
	if h, m, _ := p.Time(); h != 17 || m != 45 {
		t.Errorf("Time() = %02d:%02d, want 17:45", h, m)
	}
	if p.Text() != "17:45" {
		t.Errorf("text = %q, want the picked 17:45", p.Text())
	}
}

// The grid covers every hour and every minute, exactly once.
func TestTimePanelGridCoversTheClock(t *testing.T) {
	p := newTimePanel()
	seen := map[int]int{}
	for col := 0; col < timeHourCols; col++ {
		for row := 0; row < timeGridRows; row++ {
			seen[row+col*timeGridRows]++
		}
	}
	for h := 0; h < 24; h++ {
		if seen[h] != 1 {
			t.Fatalf("hour %d appears %d times in the grid, want once", h, seen[h])
		}
	}
	if len(seen) != 24 {
		t.Fatalf("hour grid holds %d distinct values, want 24", len(seen))
	}

	minutes := map[int]int{}
	for col := 0; col < timeMinuteCols; col++ {
		for row := 0; row < timeGridRows; row++ {
			minutes[row+col*timeGridRows]++
		}
	}
	for m := 0; m < 60; m++ {
		if minutes[m] != 1 {
			t.Fatalf("minute %d appears %d times in the grid, want once", m, minutes[m])
		}
	}
	if len(minutes) != 60 {
		t.Fatalf("minute grid holds %d distinct values, want 60", len(minutes))
	}

	// Every cell resolves back to its own value from its centre.
	for col := 0; col < timeMinuteCols; col++ {
		for row := 0; row < timeGridRows; row++ {
			r := p.cellRect(false, col, row)
			value, isHour, ok := p.valueAt(r.X+r.Width/2, r.Y+r.Height/2)
			if !ok || isHour || value != row+col*timeGridRows {
				t.Fatalf("cell (%d,%d) resolved to value=%d isHour=%v ok=%v", col, row, value, isHour, ok)
			}
		}
	}
}

// The status quo every field widget must hold: the button opens the panel
// through the window's own routing (see the comment on the shared test).
func TestTimePickerButtonOpensItsPanelThroughTheWindow(t *testing.T) {
	win := meui.NewWindow()
	win.SetFrameRect(graphics.NewRect(0, 0, 900, 700))
	p := NewTimePicker()
	win.AddChild(p)
	p.SetFrameRect(graphics.NewRect(20, 20, 170, TimePickerHeight))
	p.ValidateLayout()

	bf := p.button.FrameRect()
	pt := p.button.PointToRoot(bf.Width/2, bf.Height/2)
	win.HandleEvent(&platform.Event{Type: platform.EventMouseDown, Pos: pt, Button: 0})
	win.HandleEvent(&platform.Event{Type: platform.EventMouseUp, Pos: pt, Button: 0})
	if !p.Expanded() {
		t.Fatal("the time panel is not open after clicking its button through the window")
	}
	win.HandleEvent(&platform.Event{Type: platform.EventMouseDown, Pos: pt, Button: 0})
	win.HandleEvent(&platform.Event{Type: platform.EventMouseUp, Pos: pt, Button: 0})
	if p.Expanded() {
		t.Fatal("the time panel stayed open after a second click on its button")
	}
}

// countPaintedPixels counts the non-transparent pixels inside a rect, for the
// "it opened but drew nothing" assertions.
func countPaintedPixels(px []uint32, width int, x, y, w, h float32) int {
	count := 0
	height := len(px) / width
	for py := int(y); py < int(y+h) && py < height; py++ {
		if py < 0 {
			continue
		}
		for px2 := int(x); px2 < int(x+w) && px2 < width; px2++ {
			if px2 < 0 {
				continue
			}
			if px[(height-1-py)*width+px2] != 0 {
				count++
			}
		}
	}
	return count
}

// A picker in no window must not panic when its button is clicked.
func TestTimePickerWithoutAWindowIsInert(t *testing.T) {
	p := NewTimePicker()
	p.SetFrameRect(graphics.NewRect(0, 0, 160, TimePickerHeight))
	p.SetExpanded(true)
	if p.Expanded() {
		t.Error("a picker with no window opened a panel")
	}
}

// The rollers presentation: two wheels that scroll with the mouse wheel, drag
// and tap — the mobile-app shape, and the interaction the owner asked for.
func TestTimePickerRollersScrollWithTheMouseWheel(t *testing.T) {
	p := NewTimePicker()
	p.SetStyle(TimePickerRollers)
	p.SetTime(10, 30, 0)
	win := newPickerWindow(t, p, 160, TimePickerHeight)
	p.SetExpanded(true)
	if !p.Expanded() {
		t.Fatal("the rollers panel did not open")
	}
	if p.content() != timePanelContent(p.rollerPanel) {
		t.Fatal("SetStyle did not select the rollers")
	}
	if w, h := p.panelSize(); w != timeRollerWidth || h != timeRollerHeight {
		t.Errorf("panel size = %.0fx%.0f, want the rollers' %.0fx%.0f", w, h, timeRollerWidth, timeRollerHeight)
	}

	var gotH, gotM, fired int
	p.SetOnTimeChanged(func(h, m, s int) { gotH, gotM, fired = h, m, fired+1 })

	// The wheel under the pointer scrolls that column, and scrolling DOWN
	// (negative dy — the direction that shows later items in a list) increases
	// the value: 11:30 scrolling the minute down is 11:31, not 11:29.
	colW := rollerColumnWidth(graphics.NewRect(0, 0, timeRollerWidth, timeRollerHeight))
	hourX := timeRollerPad + colW/2
	minuteX := timeRollerPad*2 + colW + colW/2
	centreY := timeRollerHeight / 2

	p.OnMouseWheel(hourX+p.panel.offsetX, centreY+p.panel.offsetY, 0, -1)
	if gotH != 11 || gotM != 30 {
		t.Fatalf("wheel down on the hour wheel gave %02d:%02d, want 11:30", gotH, gotM)
	}
	p.OnMouseWheel(hourX+p.panel.offsetX, centreY+p.panel.offsetY, 0, 1)
	p.OnMouseWheel(hourX+p.panel.offsetX, centreY+p.panel.offsetY, 0, 1)
	if gotH != 9 {
		t.Fatalf("two wheel ups gave %02d:%02d, want the hour back to 09", gotH, gotM)
	}
	p.OnMouseWheel(minuteX+p.panel.offsetX, centreY+p.panel.offsetY, 0, -1)
	if gotM != 31 {
		t.Fatalf("wheel down on the minute wheel gave %02d:%02d, want :31", gotH, gotM)
	}
	if p.Text() != "09:31" {
		t.Errorf("text = %q, want the scrolled 09:31", p.Text())
	}
	if fired != 4 {
		t.Errorf("OnTimeChanged fired %d times, want one per wheel step", fired)
	}

	// The rollers must paint, not just open: the panel gives the widget its rect
	// and the widget draws from it. Scan the panel's own rect — a fixed range
	// misses it whenever the fixture puts the field somewhere else.
	dev := graphics.NewSoftDevice(900, 700)
	win.PaintLayerOverlay(graphics.NewCanvas(dev), 0)
	if painted := countPaintedPixels(dev.Pixels(), 900, p.panel.offsetX, p.panel.offsetY, p.panel.width, p.panel.height); painted == 0 {
		t.Fatalf("the rollers panel opened at (%.0f,%.0f %.0fx%.0f) but painted nothing",
			p.panel.offsetX, p.panel.offsetY, p.panel.width, p.panel.height)
	}

	// The wheels wrap rather than dead-ending.
	p.SetTime(23, 59, 0)
	gotH, gotM, fired = 0, 0, 0
	p.OnMouseWheel(hourX+p.panel.offsetX, centreY+p.panel.offsetY, 0, -1)
	p.OnMouseWheel(minuteX+p.panel.offsetX, centreY+p.panel.offsetY, 0, -1)
	if gotH != 0 || gotM != 0 {
		t.Errorf("wrapping 23:59 by scrolling down gave %02d:%02d, want 00:00", gotH, gotM)
	}
}

// The rollers drag and tap. Each ends with a click on the wheel, so each closes
// the panel (see the click-to-close test); what this pins is that the drag and
// the tap compute the value they are supposed to.
func TestTimePickerRollersDragAndTap(t *testing.T) {
	p := NewTimePicker()
	p.SetStyle(TimePickerRollers)
	p.SetTime(12, 0, 0)
	newPickerWindow(t, p, 160, TimePickerHeight)

	// Open first: the panel's offset is only known once it is open.
	p.SetExpanded(true)
	colW := rollerColumnWidth(graphics.NewRect(0, 0, timeRollerWidth, timeRollerHeight))
	minuteX := timeRollerPad*2 + colW + colW/2 + p.panel.offsetX
	centreY := timeRollerHeight/2 + p.panel.offsetY

	// Drag up two rows on the minute wheel: +2 minutes.
	p.HandleMouseDown(minuteX, centreY, 0)
	p.HandleMouseMove(minuteX, centreY-2*timeRollerRowH)
	p.HandleMouseUp(minuteX, centreY-2*timeRollerRowH, 0)
	if _, m, _ := p.Time(); m != 2 {
		t.Fatalf("dragging up two rows gave :%02d, want :02", m)
	}

	// Tap a row two below the centre: it comes to the centre, and that click
	// closes the panel too.
	p.SetExpanded(true)
	p.HandleMouseDown(minuteX, centreY+2*timeRollerRowH, 0)
	p.HandleMouseUp(minuteX, centreY+2*timeRollerRowH, 0)
	if _, m, _ := p.Time(); m != 4 {
		t.Fatalf("tapping two rows below the centre gave :%02d, want :04", m)
	}
	if p.Expanded() {
		t.Error("the tap left the panel open")
	}
}

// A click on the panel is a complete pick, so it closes the panel — the owner's
// rule. The grid closes on the press (one cell is one value); the rollers close
// on the release (a drag has to stay open until then); the wheel deliberately
// does not close at all.
func TestTimePickerPanelClosesOnAClick(t *testing.T) {
	p := NewTimePicker()
	newPickerWindow(t, p, 160, TimePickerHeight)

	// Grid: a cell click applies and closes.
	p.SetExpanded(true)
	cell := p.gridPanel.cellRect(true, 1, 5)
	p.HandleMouseDown(cell.X+cell.Width/2+p.panel.offsetX, cell.Y+cell.Height/2+p.panel.offsetY, 0)
	if h, _, _ := p.Time(); h != 17 {
		t.Fatalf("the cell click did not apply: hour = %d, want 17", h)
	}
	if p.Expanded() {
		t.Error("a click on the grid cell left the panel open")
	}

	// Rollers: a tap closes on release; a drag also closes on release.
	p = NewTimePicker()
	p.SetStyle(TimePickerRollers)
	newPickerWindow(t, p, 160, TimePickerHeight)
	// The panel's offset is only known once it is open, so open it first.
	p.SetExpanded(true)
	colW := rollerColumnWidth(graphics.NewRect(0, 0, timeRollerWidth, timeRollerHeight))
	minuteX := timeRollerPad*2 + colW + colW/2 + p.panel.offsetX
	centreY := timeRollerHeight/2 + p.panel.offsetY
	p.HandleMouseDown(minuteX, centreY, 0)
	p.HandleMouseUp(minuteX, centreY, 0)
	if p.Expanded() {
		t.Error("a click on the roller left the panel open")
	}

	p.SetExpanded(true)
	p.HandleMouseDown(minuteX, centreY, 0)
	p.HandleMouseMove(minuteX, centreY-2*timeRollerRowH)
	p.HandleMouseUp(minuteX, centreY-2*timeRollerRowH, 0)
	if p.Expanded() {
		t.Error("the end of a roller drag left the panel open")
	}

	// The wheel is the exception: scrolling twice keeps the panel open.
	p.SetExpanded(true)
	p.OnMouseWheel(minuteX, centreY, 0, 1)
	p.OnMouseWheel(minuteX, centreY, 0, 1)
	if !p.Expanded() {
		t.Error("scrolling the wheel closed the panel; every notch would dismiss it")
	}
}
