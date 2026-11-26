package theme

import "testing"

func f32(v float32) *float32 { return &v }
func pstr(v string) *string  { return &v }

func TestThemeFileInheritance(t *testing.T) {
	if err := LoadPreset("breeze-dark"); err != nil {
		t.Fatal(err)
	}
	tf := &themeFile{
		ID:   "brand-test-dark",
		Base: "breeze-dark",
		Palette: map[string]string{
			"accent":  "#E62D42",
			"surface": "#2A2A2A",
		},
		Metrics: &themeFileMetrics{
			ControlHeight: f32(40),
			Shape:         &themeFileShape{Button: f32(9)},
		},
		Motion: &themeFileMotion{HoverMS: f32(33)},
		Chrome: &themeFileChrome{Style: pstr("windows"), Height: f32(34)},
		Fonts:  &themeFileFonts{Families: []string{"Inter", "Roboto"}, BodySize: f32(15)},
	}
	id, err := registerThemeFile(tf)
	if err != nil {
		t.Fatal(err)
	}
	if id != "brand-test-dark" {
		t.Fatalf("id = %q", id)
	}
	if err := LoadPreset(id); err != nil {
		t.Fatal(err)
	}
	th := DefaultTheme
	if th.Accent != 0xFFE62D42 {
		t.Errorf("accent = %08X, want FF E62D42", th.Accent)
	}
	if th.BorderFocused != 0xFFE62D42 {
		t.Errorf("accent derivation: borderFocused = %08X", th.BorderFocused)
	}
	if th.Surface != 0xFF2A2A2A {
		t.Errorf("surface = %08X, want FF2A2A2A", th.Surface)
	}
	if th.Metrics.ControlHeight != 40 {
		t.Errorf("controlHeight = %v, want 40", th.Metrics.ControlHeight)
	}
	if th.Metrics.Shape.Button != 9 {
		t.Errorf("shape.button = %v, want 9", th.Metrics.Shape.Button)
	}
	if th.Motion.Hover.DurationMS != 33 {
		t.Errorf("motion.hover = %v, want 33", th.Motion.Hover.DurationMS)
	}
	if th.Chrome.Style != ChromeWindows || th.Chrome.Height != 34 {
		t.Errorf("chrome = %+v, want windows/34", th.Chrome)
	}
	if th.Typography.Body.Size != 15 || len(th.Typography.Families) == 0 || th.Typography.Families[0] != "Inter" {
		t.Errorf("fonts not applied: %+v", th.Typography)
	}
	// Generated widget themes follow: radius from the file's shape,
	// active color from the file's accent.
	if DefaultButtonTheme.CornerRadius != 9 {
		t.Errorf("button radius = %v, want 9", DefaultButtonTheme.CornerRadius)
	}
	if DefaultButtonTheme.BackgroundActive != 0xFFE62D42 {
		t.Errorf("button active = %08X, want the file accent", DefaultButtonTheme.BackgroundActive)
	}
}

func TestThemeFileErrors(t *testing.T) {
	if _, err := registerThemeFile(&themeFile{Base: "breeze-dark"}); err == nil {
		t.Error("missing id accepted")
	}
	if _, err := registerThemeFile(&themeFile{ID: "x", Base: "no-such-base"}); err == nil {
		t.Error("unknown base accepted")
	}
	if _, err := registerThemeFile(&themeFile{ID: "brand-test-dark", Base: "breeze-dark"}); err == nil {
		t.Error("duplicate id accepted")
	}
}
