package theme

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
)

// TestPresetsComplete verifies every registered preset fills all token
// groups — a platform preset with a zeroed metrics/motion/typography
// group would silently render as the default look.
func TestPresetsComplete(t *testing.T) {
	ids := PresetIDs()
	if len(ids) < 11 {
		t.Fatalf("expected at least 11 presets, got %d: %v", len(ids), ids)
	}
	for _, id := range ids {
		th := presetBuilders[id]()
		if th == nil {
			t.Fatalf("%s: builder returned nil", id)
		}
		th.withStyleDefaults() // LoadPreset resolves groups before installing
		if th.Meta.ID != id || th.Meta.Platform == "" {
			t.Errorf("%s: meta not filled: %+v", id, th.Meta)
		}
		if th.Name == "" {
			t.Errorf("%s: name empty", id)
		}
		// Palette essentials.
		for name, c := range map[string]graphics.Color{
			"Background": th.Background, "Surface": th.Surface,
			"SurfaceVariant": th.SurfaceVariant, "TextPrimary": th.TextPrimary,
			"TextSecondary": th.TextSecondary, "Border": th.Border,
			"Accent": th.Accent, "Selected": th.Selected, "Hovered": th.Hovered,
		} {
			if c == 0 {
				t.Errorf("%s: palette %s is zero", id, name)
			}
		}
		// Token groups.
		if th.Metrics.ControlHeight == 0 || th.Metrics.Shape == nil ||
			th.Metrics.ControlPadX == 0 || th.Metrics.BoxSize == 0 ||
			th.Metrics.TabHeight == 0 || th.Metrics.MenuItemHeight == 0 {
			t.Errorf("%s: metrics incomplete: %+v", id, th.Metrics)
		}
		if th.Motion.Hover.DurationMS == 0 || th.Motion.MenuOpen.DurationMS == 0 {
			t.Errorf("%s: motion incomplete: %+v", id, th.Motion)
		}
		if th.Typography.Body.Size == 0 || len(th.Typography.Families) == 0 {
			t.Errorf("%s: typography incomplete: %+v", id, th.Typography)
		}
		if th.Chrome.Style == "" || th.Chrome.Height == 0 {
			t.Errorf("%s: chrome incomplete: %+v", id, th.Chrome)
		}
		if th.Effects.Focus.Kind == FocusNone && th.Meta.Platform != PlatformMeui {
			t.Errorf("%s: focus style missing", id)
		}
	}
}

// TestLoadPresetPopulatesWidgetThemes verifies the generator pushes the
// preset's metrics into the widget theme dimension fields — the point
// of the token model is that widgets never hardcode these again.
func TestLoadPresetPopulatesWidgetThemes(t *testing.T) {
	saved := DefaultTheme
	savedShape := Shape()
	defer func() {
		SetDefaultTheme(saved)
		SetShape(savedShape)
	}()

	for _, id := range PresetIDs() {
		if err := LoadPreset(id); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		m := DefaultTheme.Metrics
		s := m.Shape
		if got := DefaultButtonTheme.HMargin; got != m.ControlPadX {
			t.Errorf("%s: Button HMargin = %v, want %v", id, got, m.ControlPadX)
		}
		if got := DefaultButtonTheme.VMargin; got != m.ControlPadY {
			t.Errorf("%s: Button VMargin = %v, want %v", id, got, m.ControlPadY)
		}
		if got := DefaultTextFieldTheme.HMargin; got != m.InputPadX {
			t.Errorf("%s: TextField HMargin = %v, want %v", id, got, m.InputPadX)
		}
		if got := DefaultCheckBoxTheme.BoxSize; got != m.BoxSize {
			t.Errorf("%s: CheckBox BoxSize = %v, want %v", id, got, m.BoxSize)
		}
		if got := DefaultTabTheme.TabHeight; got != m.TabHeight {
			t.Errorf("%s: Tab height = %v, want %v", id, got, m.TabHeight)
		}
		if got := DefaultComboBoxTheme.OptionHeight; got != m.MenuItemHeight {
			t.Errorf("%s: ComboBox option height = %v, want %v", id, got, m.MenuItemHeight)
		}
		if got := DefaultSliderTheme.ThumbSize; got != m.SliderThumb {
			t.Errorf("%s: Slider thumb = %v, want %v", id, got, m.SliderThumb)
		}
		if got := DefaultButtonTheme.CornerRadius; got != s.Button {
			t.Errorf("%s: Button radius = %v, want %v", id, got, s.Button)
		}
		if DefaultButtonTheme.BackgroundInk == 0 || DefaultTextFieldTheme.BackgroundInk == 0 ||
			DefaultTabTheme.IndicatorInk == 0 || DefaultScrollBarTheme.ThumbInk == 0 {
			t.Errorf("%s: generated widget themes have zero colors", id)
		}
	}
}

// TestLoadPresetDeterministic pins that loading the same preset twice
// yields identical widget themes and palette (presets are static
// tables; this guards accidental map-order dependence in the generator).
func TestLoadPresetDeterministic(t *testing.T) {
	saved := DefaultTheme
	savedShape := Shape()
	defer func() {
		SetDefaultTheme(saved)
		SetShape(savedShape)
	}()

	for _, id := range PresetIDs() {
		if err := LoadPreset(id); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		first := *DefaultButtonTheme
		firstBG, firstSurface := DefaultTheme.Background, DefaultTheme.Surface
		firstMetrics := DefaultTheme.Metrics

		if err := LoadPreset(id); err != nil {
			t.Fatalf("%s reload: %v", id, err)
		}
		if *DefaultButtonTheme != first {
			t.Errorf("%s: Button theme differs across loads", id)
		}
		if DefaultTheme.Background != firstBG || DefaultTheme.Surface != firstSurface {
			t.Errorf("%s: palette differs across loads", id)
		}
		// Shape is a per-build pointer; compare the scalar token set.
		a, b := DefaultTheme.Metrics, firstMetrics
		a.Shape, b.Shape = nil, nil
		if a != b {
			t.Errorf("%s: metrics differ across loads", id)
		}
	}
}

// TestPresetLightDarkPairs verifies each platform family ships both
// modes with distinct palettes.
func TestPresetLightDarkPairs(t *testing.T) {
	pairs := [][2]string{
		{"macos-light", "macos-dark"},
		{"windows11-light", "windows11-dark"},
		{"adwaita-light", "adwaita-dark"},
		{"breeze-light", "breeze-dark"},
	}
	for _, p := range pairs {
		light, dark := presetBuilders[p[0]](), presetBuilders[p[1]]()
		if light == nil || dark == nil {
			t.Fatalf("%s/%s not registered", p[0], p[1])
		}
		if light.Meta.Mode != ModeLight || dark.Meta.Mode != ModeDark {
			t.Errorf("%s/%s modes wrong: %v/%v", p[0], p[1], light.Meta.Mode, dark.Meta.Mode)
		}
		if light.TextPrimary == dark.TextPrimary {
			t.Errorf("%s/%s share TextPrimary %08X", p[0], p[1], light.TextPrimary)
		}
		if luminance(light.TextPrimary) >= luminance(dark.TextPrimary) {
			t.Errorf("%s/%s: light-mode text (lum %d) is not darker than dark-mode text (lum %d)",
				p[0], p[1], luminance(light.TextPrimary), luminance(dark.TextPrimary))
		}
	}
}

func luminance(c graphics.Color) uint32 {
	r := (c >> 16) & 0xFF
	g := (c >> 8) & 0xFF
	b := c & 0xFF
	return uint32((r*299 + g*587 + b*114) / 1000)
}

// TestMeuiDefaultIsOverlayScrollbars pins the meui look: a floating
// thumb pill on scroll/hover with NO full-height track band (the track
// read as a stray line running down lists and trees).
func TestMeuiDefaultIsOverlayScrollbars(t *testing.T) {
	m := DefaultMetrics()
	if !m.ScrollbarOverlay {
		t.Error("DefaultMetrics().ScrollbarOverlay = false, want true (meui default is overlay)")
	}
	if !DefaultScrollBarTheme.Overlay {
		t.Error("DefaultScrollBarTheme.Overlay = false, want true")
	}
	for _, id := range []string{"meui-dark", "meui-light"} {
		if err := LoadPreset(id); err != nil {
			t.Fatalf("LoadPreset(%q): %v", id, err)
		}
		if !DefaultScrollBarTheme.Overlay {
			t.Errorf("%s: bars still paint a track (Overlay=false)", id)
		}
	}
}

// TestScrollbarAutoHideIsOptIn pins that bars are PERSISTENT by default
// and only fade out when a theme opts in (macOS/GNOME platform presets).
func TestScrollbarAutoHideIsOptIn(t *testing.T) {
	if m := DefaultMetrics(); m.ScrollbarAutoHide {
		t.Error("DefaultMetrics().ScrollbarAutoHide = true; auto-hide must be opt-in, not the norm")
	}
	if err := LoadPreset("meui-dark"); err != nil {
		t.Fatal(err)
	}
	if DefaultScrollBarTheme.AutoHide {
		t.Error("meui-dark opted into auto-hide; the meui pill must stay visible")
	}
	if err := LoadPreset("macos-dark"); err != nil {
		t.Fatal(err)
	}
	if !DefaultScrollBarTheme.AutoHide {
		t.Error("macos-dark: auto-hide = false, want true (platform-faithful)")
	}
	if err := LoadPreset("adwaita-dark"); err != nil {
		t.Fatal(err)
	}
	if !DefaultScrollBarTheme.AutoHide {
		t.Error("adwaita-dark: auto-hide = false, want true (GNOME overlay)")
	}
}
