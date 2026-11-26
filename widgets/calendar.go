package widgets

import (
	"fmt"
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
	"os"
	"time"
)

var (
	calChevronLeft  *meui.SVGDocument
	calChevronRight *meui.SVGDocument
)

func init() {
	if doc, err := meui.LoadSVGByName("chevron_left"); err == nil {
		calChevronLeft = doc
	}
	if doc, err := meui.LoadSVGByName("chevron_right"); err == nil {
		calChevronRight = doc
	}
}

type Calendar struct {
	meui.BaseWidget
	*theme.CalendarTheme
	Year           int
	Month          int
	SelectedDay    int
	HoveredDay     int
	Today          int
	OnDateSelected func(year, month, day int)
	cache          meui.CacheBase
	monthText      *graphics.Text
	monthTextVer   uint64
}

func NewCalendar() *Calendar {
	now := time.Now()
	c := &Calendar{
		BaseWidget:    *meui.NewBaseWidget(),
		CalendarTheme: theme.DefaultCalendarTheme,
		Year:          now.Year(),
		Month:         int(now.Month()),
		SelectedDay:   now.Day(),
		Today:         now.Day(),
	}
	c.buildMonthText()
	return c
}

func (c *Calendar) buildMonthText() {
	font := c.CalendarTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	monthStr := fmt.Sprintf("%s %d", time.Month(c.Month).String(), c.Year)
	c.monthText = graphics.NewText(monthStr, &graphics.TextDecoration{Font: font, OnBackgroundInk: c.CalendarTheme.TextActive})
}

// SetOnDateSelected registers the date-picked callback.
func (c *Calendar) SetOnDateSelected(handler func(year, month, day int)) *Calendar {
	c.OnDateSelected = handler
	return c
}

// GetSelectedDate returns the currently selected (year, month, day).
func (c *Calendar) GetSelectedDate() (year, month, day int) {
	return c.Year, c.Month, c.SelectedDay
}

func (c *Calendar) SetDate(year, month, day int) *Calendar {
	c.Year = year
	c.Month = month
	c.SelectedDay = day
	c.buildMonthText()
	c.cache.Invalidate()
	c.MarkForRedraw()
	return c
}

func (c *Calendar) PrevMonth() {
	c.Month--
	if c.Month < 1 {
		c.Month = 12
		c.Year--
	}
	c.buildMonthText()
	c.cache.Invalidate()
	c.MarkForRedraw()
}

func (c *Calendar) NextMonth() {
	c.Month++
	if c.Month > 12 {
		c.Month = 1
		c.Year++
	}
	c.buildMonthText()
	c.cache.Invalidate()
	c.MarkForRedraw()
}

func (c *Calendar) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if os.Getenv("MEUI_FLUSH_DEBUG") != "" {
		fmt.Printf("[calPaint] visible=%v rect=(%.0f,%.0f %.0fx%.0f)\n", c.Visible(), rect.X, rect.Y, rect.Width, rect.Height)
	}
	if !c.Visible() {
		return
	}

	canvas.Save()

	c.cache.GetCached(canvas, rect, func(cv *graphics.Canvas, r graphics.Rect) {
		bgPaint := graphics.NewPaint()
		bgPaint.SetColor(c.CalendarTheme.BackgroundInk)
		cv.DrawRoundedRect(r, c.CalendarTheme.CornerRadius, c.CalendarTheme.CornerRadius, bgPaint)

		font := c.CalendarTheme.Font
		if font == nil {
			font = theme.DefaultTheme.BaseFont
			if font == nil {
				font = graphics.SystemFont
			}
		}

		headerH := float32(30)
		cellH := (r.Height - headerH) / 7
		cellW := r.Width / 7

		// Header
		headerPaint := graphics.NewPaint()
		headerPaint.SetColor(c.CalendarTheme.BackgroundActive)
		cv.DrawRect(graphics.NewRect(r.X, r.Y, r.Width, headerH), headerPaint)

		navW := float32(30)
		iconSize := float32(16)
		arrowY := r.Y + (headerH-iconSize)/2

		if calChevronLeft != nil {
			meui.DrawSVG(cv, calChevronLeft, r.X+7, arrowY, iconSize, iconSize)
		}

		textPaint := graphics.NewPaint()
		textPaint.SetColor(c.CalendarTheme.TextActive)

		monthStr := fmt.Sprintf("%s %d", time.Month(c.Month).String(), c.Year)
		monthW := font.SimpleWidth(monthStr)
		monthX := r.X + (r.Width-monthW)/2
		ver := theme.ThemeVersion()
		if c.monthText == nil || c.monthTextVer != ver {
			c.monthText = graphics.NewText(monthStr, &graphics.TextDecoration{Font: font, OnBackgroundInk: textPaint.Color()})
			c.monthTextVer = ver
		}
		if c.monthText != nil {
			c.monthText.Draw(cv, monthX, r.Y+headerH/2+font.Baseline()/2)
		}

		if calChevronRight != nil {
			meui.DrawSVG(cv, calChevronRight, r.X+r.Width-navW+7, arrowY, iconSize, iconSize)
		}

		// Day headers
		dayNames := []string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}
		dayHeaderY := r.Y + headerH + 2
		dayHeaderH := float32(20)
		dayHeaderPaint := graphics.NewPaint()
		dayHeaderPaint.SetColor(c.CalendarTheme.TextInk)
		for i, name := range dayNames {
			x := r.X + float32(i)*cellW + cellW/2
			w := font.SimpleWidth(name)
			dt := graphics.NewText(name, &graphics.TextDecoration{Font: font, OnBackgroundInk: dayHeaderPaint.Color()})
			dt.Draw(cv, x-w/2, dayHeaderY+font.Baseline())
		}

		// Calendar grid
		firstDay := time.Date(c.Year, time.Month(c.Month), 1, 0, 0, 0, 0, time.Local)
		startWeekday := int(firstDay.Weekday())
		daysInMonth := daysInMonth(c.Year, c.Month)

		cellY := dayHeaderY + dayHeaderH
		textPaint.SetColor(c.CalendarTheme.TextInk)
		for row := 0; row < 6; row++ {
			for col := 0; col < 7; col++ {
				day := row*7 + col - startWeekday + 1
				x := r.X + float32(col)*cellW
				y := cellY + float32(row)*cellH

				if day < 1 || day > daysInMonth {
					continue
				}

				cellRect := graphics.NewRect(x+1, y+1, cellW-2, cellH-2)

				if day == c.SelectedDay {
					selPaint := graphics.NewPaint()
					selPaint.SetColor(theme.DefaultTheme.Primary)
					cv.DrawRoundedRect(cellRect, 4, 4, selPaint)
					textPaint.SetColor(graphics.Color(0xFFFFFFFF))
				} else if day == c.Today {
					todayPaint := graphics.NewPaint()
					todayPaint.SetStyle(paintstyle.Stroke)
					todayPaint.SetStrokeWidth(theme.BorderStrokeWidth)
					todayPaint.SetColor(theme.DefaultTheme.Primary)
					cv.DrawRoundedRect(cellRect, 4, 4, todayPaint)
					textPaint.SetColor(c.CalendarTheme.TextActive)
				} else if day == c.HoveredDay {
					hoverPaint := graphics.NewPaint()
					hoverPaint.SetColor(c.CalendarTheme.BackgroundHover)
					cv.DrawRect(cellRect, hoverPaint)
					textPaint.SetColor(c.CalendarTheme.TextInk)
				} else {
					textPaint.SetColor(c.CalendarTheme.TextInk)
				}

				dayStr := fmt.Sprintf("%d", day)
				dayW := font.SimpleWidth(dayStr)
				DrawText(cv, dayStr, x+cellW/2-dayW/2, y+cellH/2+font.Baseline()/2, font, textPaint)
			}
		}
	})

	canvas.Restore()
}

func (c *Calendar) OnMouseDown(x, y float32, button int) bool {
	if !c.Enabled() {
		return false
	}

	headerH := float32(30)
	cellH := (c.Height() - headerH) / 7
	cellW := c.Width() / 7
	dayHeaderH := float32(20)

	if y <= headerH {
		navW := float32(30)
		if x <= navW {
			c.PrevMonth()
			return true
		}
		if x >= c.Width()-navW {
			c.NextMonth()
			return true
		}
		return false
	}

	if y <= headerH+dayHeaderH {
		return false
	}

	firstDay := time.Date(c.Year, time.Month(c.Month), 1, 0, 0, 0, 0, time.Local)
	startWeekday := int(firstDay.Weekday())
	daysInMonth := daysInMonth(c.Year, c.Month)

	gridY := y - headerH - dayHeaderH
	row := int(gridY / cellH)
	col := int(x / cellW)
	day := row*7 + col - startWeekday + 1

	if day >= 1 && day <= daysInMonth {
		c.SelectedDay = day
		c.cache.Invalidate()
		c.MarkForRedraw()
		if c.OnDateSelected != nil {
			c.OnDateSelected(c.Year, c.Month, day)
		}
		return true
	}
	return false
}

func (c *Calendar) OnMouseMove(x, y float32) {
	if !c.Enabled() {
		return
	}

	headerH := float32(30)
	cellH := (c.Height() - headerH) / 7
	cellW := c.Width() / 7
	dayHeaderH := float32(20)

	if y <= headerH+dayHeaderH || y > c.Height() {
		if c.HoveredDay != 0 {
			c.HoveredDay = 0
			c.MarkForRedraw()
		}
		return
	}

	firstDay := time.Date(c.Year, time.Month(c.Month), 1, 0, 0, 0, 0, time.Local)
	startWeekday := int(firstDay.Weekday())
	daysInMonth := daysInMonth(c.Year, c.Month)

	gridY := y - headerH - dayHeaderH
	row := int(gridY / cellH)
	col := int(x / cellW)
	day := row*7 + col - startWeekday + 1

	if day >= 1 && day <= daysInMonth && day != c.HoveredDay {
		c.HoveredDay = day
		c.cache.Invalidate()
		c.MarkForRedraw()
	} else if (day < 1 || day > daysInMonth) && c.HoveredDay != 0 {
		c.HoveredDay = 0
		c.cache.Invalidate()
		c.MarkForRedraw()
	}
}

func (c *Calendar) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if !c.Enabled() {
		return false
	}
	switch keyCode {
	case platform.KeyLeft:
		if c.SelectedDay > 1 {
			c.SelectedDay--
		} else {
			c.PrevMonth()
			c.SelectedDay = daysInMonth(c.Year, c.Month)
		}
		c.cache.Invalidate()
		c.MarkForRedraw()
		return true
	case platform.KeyRight:
		dim := daysInMonth(c.Year, c.Month)
		if c.SelectedDay < dim {
			c.SelectedDay++
		} else {
			c.NextMonth()
			c.SelectedDay = 1
		}
		c.cache.Invalidate()
		c.MarkForRedraw()
		return true
	case platform.KeyUp:
		if c.SelectedDay > 7 {
			c.SelectedDay -= 7
		} else {
			c.PrevMonth()
			c.SelectedDay = daysInMonth(c.Year, c.Month) + c.SelectedDay
		}
		c.cache.Invalidate()
		c.MarkForRedraw()
		return true
	case platform.KeyDown:
		dim := daysInMonth(c.Year, c.Month)
		if c.SelectedDay+7 <= dim {
			c.SelectedDay += 7
		} else {
			c.NextMonth()
			c.SelectedDay -= dim - c.SelectedDay
		}
		c.cache.Invalidate()
		c.MarkForRedraw()
		return true
	}
	return false
}

func (c *Calendar) Focusable() bool {
	return c.Enabled()
}

func (c *Calendar) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "year":
		if f, ok := value.(float64); ok {
			c.Year = int(f)
		}
	case "month":
		if f, ok := value.(float64); ok {
			c.Month = int(f)
		}
	case "day", "selectedDay":
		if f, ok := value.(float64); ok {
			c.SelectedDay = int(f)
		}
	}
}

func daysInMonth(year, month int) int {
	return time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.Local).Day()
}
