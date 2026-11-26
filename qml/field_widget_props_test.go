package qml

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/widgets"
)

// QML hands whole numbers to a widget as int (parseNumberOrString), so a
// SetBoundProperty that only accepted float64 silently dropped `year: 2026` and
// `hour: 9` — the widget kept its default and, because the default happened to be
// today's date, it looked correct.
func TestFieldWidgetNumericPropertiesApply(t *testing.T) {
	src := `Window {
		width: 600; height: 400;
		DatePicker { id: dp; year: 2026; month: 3; day: 14 }
		TimePicker { id: tp; hour: 9; minute: 30 }
		ColorField { id: cf; color: "#4A88C7" }
	}`
	widget, err := Convert(src)
	if err != nil {
		t.Fatal(err)
	}

	dp, ok := findWidget[*widgets.DatePicker](widget)
	if !ok {
		t.Fatal("no DatePicker in the built tree")
	}
	if y, m, d := dp.Date(); y != 2026 || m != 3 || d != 14 {
		t.Errorf("DatePicker Date() = %d-%d-%d, want 2026-3-14 (the QML year/month/day)", y, m, d)
	}
	if dp.Text() != "2026-03-14" {
		t.Errorf("DatePicker text = %q, want 2026-03-14", dp.Text())
	}

	tp, ok := findWidget[*widgets.TimePicker](widget)
	if !ok {
		t.Fatal("no TimePicker in the built tree")
	}
	if h, m, _ := tp.Time(); h != 9 || m != 30 {
		t.Errorf("TimePicker Time() = %d:%d, want 9:30 (the QML hour/minute)", h, m)
	}
	if tp.Text() != "09:30" {
		t.Errorf("TimePicker text = %q, want 09:30", tp.Text())
	}

	cf, ok := findWidget[*widgets.ColorField](widget)
	if !ok {
		t.Fatal("no ColorField in the built tree")
	}
	if cf.Hex() != "#4A88C7" {
		t.Errorf("ColorField Hex() = %q, want #4A88C7 (the QML color)", cf.Hex())
	}
}

// findWidget walks the built tree for the first widget of a type. It reports
// found separately: a nil *T in an interface is not nil, so returning the zero
// value and testing it would call every miss a hit.
func findWidget[T meui.Widget](w meui.Widget) (T, bool) {
	if typed, ok := w.(T); ok {
		return typed, true
	}
	if c, ok := w.(meui.Container); ok {
		for _, child := range c.Children() {
			if found, ok := findWidget[T](child); ok {
				return found, true
			}
		}
	}
	var zero T
	return zero, false
}
