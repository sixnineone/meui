package behaviour

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/widgets"
)

const scrollQML = `
Window {
    width: 300
    height: 300
    VStack {
        margins: 10
        ScrollBar {
            id: sb
            width: 16
            height: 250
            Orientation: "vertical"
        }
    }
}
`

func TestScrollBarDragChangesValue(t *testing.T) {
	h := Load(t, scrollQML)
	sb := h.Widget("sb").(*widgets.ScrollBar)
	sb.Min = 0
	sb.Max = 100
	sb.Value = 0
	sb.VisibleAmount = 30 // thumb = 30% of the track

	f := sb.FrameRect()
	// The thumb starts at the top (value 0); drag its center downward.
	h.Drag(f.X+f.Width/2, f.Y+15, f.X+f.Width/2, f.Y+f.Height-40)

	if sb.Value <= 0 {
		t.Fatalf("drag did not increase value: %v", sb.Value)
	}
	if sb.Value > 100 {
		t.Fatalf("value out of range: %v", sb.Value)
	}
}

func TestScrollBarClickTrack(t *testing.T) {
	h := Load(t, scrollQML)
	sb := h.Widget("sb").(*widgets.ScrollBar)
	sb.Min = 0
	sb.Max = 100
	sb.VisibleAmount = 30

	f := sb.FrameRect()
	// Click the track below the thumb: pages down.
	h.Click(f.X+f.Width/2, f.Y+f.Height*0.8)
	if sb.Value <= 0 {
		t.Fatalf("track click did not change value: %v", sb.Value)
	}
}

const calendarQML = `
Window {
    width: 400
    height: 400
    Calendar {
        id: cal
        width: 380
        height: 360
    }
}
`

func TestCalendarDayClick(t *testing.T) {
	h := Load(t, calendarQML)
	cal := h.Widget("cal").(*widgets.Calendar)
	cal.SetDate(2026, 8, 1) // August 2026; the 1st falls on a Saturday

	var y, m, d int
	fired := false
	cal.SetOnDateSelected(func(yy, mm, dd int) { y, m, d = yy, mm, dd; fired = true })

	f := cal.FrameRect()
	cellH := (f.Height - 50) / 7
	cellW := f.Width / 7
	// August 2026 starts on Saturday (col 6, row 0): click day 1.
	h.Click(f.X+cellW*6+cellW/2, f.Y+50+cellH/2)

	if !fired {
		t.Fatal("date click did not fire OnDateSelected")
	}
	if m != 8 || y != 2026 {
		t.Fatalf("wrong month/year: %d/%d", m, y)
	}
	if d < 1 || d > 31 {
		t.Fatalf("day out of range: %d", d)
	}
}

func TestCalendarPrevNextMonth(t *testing.T) {
	h := Load(t, calendarQML)
	cal := h.Widget("cal").(*widgets.Calendar)
	cal.SetDate(2026, 8, 15)
	cal.PrevMonth()
	y, m, _ := cal.GetSelectedDate()
	if y != 2026 || m != 7 {
		t.Fatalf("prev month: %d/%d", y, m)
	}
	cal.NextMonth()
	cal.NextMonth()
	y, m, _ = cal.GetSelectedDate()
	if y != 2026 || m != 9 {
		t.Fatalf("next months: %d/%d", y, m)
	}
}

const mdiQML = `
Window {
    width: 600
    height: 500
    MdiContainer {
        id: mdi
        width: 580
        height: 480
    }
}
`

func TestMdiWindowDragAndClose(t *testing.T) {
	h := Load(t, mdiQML)
	_ = h.Widget("mdi")

	mw := widgets.NewMdiWindow()
	mw.SetTitle("Window 1")
	mw.SetFrameRect(graphics.NewRect(20, 20, 200, 150))
	var closed bool
	mw.SetOnClose(func() { closed = true })

	// MdiContainer.AddChild takes the window and wires the container.
	if mc, ok := h.Widget("mdi").(*widgets.MdiContainer); ok {
		mc.AddChild(mw)
	} else {
		t.Fatal("mdi widget is not an MdiContainer")
	}

	f0 := mw.FrameRect()
	// Drag the title bar (top strip of the window).
	h.Drag(f0.X+60, f0.Y+10, f0.X+160, f0.Y+60)
	f1 := mw.FrameRect()
	if f1.X == f0.X && f1.Y == f0.Y {
		t.Fatal("title-bar drag did not move the MDI window")
	}

	// Close via the title-bar close button (top-right of the title bar).
	tbH := 30.0
	closeX := float32(int(f1.X) + int(f1.Width) - 14)
	closeY := float32(int(f1.Y) + int(tbH)/2)
	h.Click(closeX, closeY)
	if !closed {
		t.Fatal("close button did not fire OnClose")
	}
}
