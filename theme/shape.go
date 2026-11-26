package theme

import (
	"fmt"
	"sync"
)

// ShapeTheme is the look-and-feel shape profile: corner radii per
// widget class, in logical pixels. It is orthogonal to the color
// theme — any color theme combines with any shape profile.
type ShapeTheme struct {
	Window    float32 // main window corner radius (0 = square)
	Popup     float32 // modal popups, dialogs
	Menu      float32 // menu bar dropdowns, context menus
	Tooltip   float32
	Button    float32
	CheckBox  float32
	Input     float32 // TextField, SpinBox
	ComboBox  float32
	Tab       float32
	GroupBox  float32
	ListRow   float32
	ScrollBar float32
}

func (s *ShapeTheme) clone() *ShapeTheme {
	cp := *s
	return &cp
}

// Named L&F presets. Values in logical px.
func ShapeFlat() *ShapeTheme {
	return &ShapeTheme{} // everything square
}

// ShapeMeui is the current default look (subtle 4px rounding).
func ShapeMeui() *ShapeTheme {
	return &ShapeTheme{
		Popup: 4, Menu: 4, Tooltip: 4,
		Button: 4, CheckBox: 3, Input: 4, ComboBox: 4,
		Tab: 0, GroupBox: 4, ListRow: 3, ScrollBar: 3,
	}
}

func ShapeWindows11() *ShapeTheme {
	// Fluent geometry: 8px top-level containers (window/flyout/dialog),
	// 4px in-page controls (button/checkbox/combo/textbox/list), 4px
	// tooltip (small-size exception), 4px bars.
	return &ShapeTheme{
		Window: 8, Popup: 8, Menu: 8, Tooltip: 4,
		Button: 4, CheckBox: 4, Input: 4, ComboBox: 4, Tab: 4,
		GroupBox: 8, ListRow: 4, ScrollBar: 4,
	}
}

func ShapeMacOS() *ShapeTheme {
	return &ShapeTheme{
		Window: 10, Popup: 12, Menu: 10, Tooltip: 8,
		Button: 10, CheckBox: 4, Input: 6, ComboBox: 6, Tab: 6,
		GroupBox: 10, ListRow: 6, ScrollBar: 4,
	}
}

var (
	shapeMu      sync.RWMutex
	defaultShape = ShapeMeui()
	shapeName    = "meui"
)

// Shape returns the active shape profile. Widgets read radii from
// here (or from their theme structs, which SetShape keeps in sync).
func Shape() *ShapeTheme {
	shapeMu.RLock()
	defer shapeMu.RUnlock()
	return defaultShape.clone()
}

// ShapeName returns the active preset name ("meui", "macos", …).
func ShapeName() string {
	shapeMu.RLock()
	defer shapeMu.RUnlock()
	return shapeName
}

// SetShape installs a shape profile and propagates it to the widget
// theme structs (in-place, so widgets holding pointers see it live).
// Fires the theme-change listeners so open windows repaint.
func SetShape(s *ShapeTheme) {
	if s == nil {
		return
	}
	shapeMu.Lock()
	defaultShape = s.clone()
	shapeMu.Unlock()
	applyShapeToWidgetThemes(s)
	notifyThemeChanged()
}

// SetLookAndFeel applies a named shape preset:
// "flat", "meui", "windows11", "macos".
func SetLookAndFeel(name string) error {
	switch name {
	case "flat", "square":
		SetShape(ShapeFlat())
	case "meui", "default":
		SetShape(ShapeMeui())
	case "windows11", "win11":
		SetShape(ShapeWindows11())
	case "macos", "mac":
		SetShape(ShapeMacOS())
	default:
		return fmt.Errorf("theme: unknown look-and-feel %q (want flat, meui, windows11, macos)", name)
	}
	shapeMu.Lock()
	shapeName = name
	shapeMu.Unlock()
	return nil
}

// applyShapeToWidgetThemes pushes radii into the per-widget theme
// globals that widgets already consume. In-place mutation keeps every
// existing widget instance live (they hold pointers to these structs).
// Widget classes without a theme radius field (GroupBox, popup, menu,
// tooltip) resolve from Shape() directly in their Paint.
func applyShapeToWidgetThemes(s *ShapeTheme) {
	DefaultButtonTheme.CornerRadius = s.Button
	DefaultCheckBoxTheme.CornerRadius = s.CheckBox
	DefaultTextFieldTheme.CornerRadius = s.Input
	DefaultTabTheme.CornerRadius = s.Tab
}
