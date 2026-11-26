package theme

import "testing"

func TestParseKDEColor(t *testing.T) {
	c, ok := parseKDEColor("61,174,233")
	if !ok || c != 0xFF3DAEE9 {
		t.Fatalf("parseKDEColor = %08X ok=%v, want FF3DAEE9", c, ok)
	}
	if _, ok := parseKDEColor("300,0,0"); ok {
		t.Error("out-of-range component accepted")
	}
	if _, ok := parseKDEColor("1,2"); ok {
		t.Error("short tuple accepted")
	}
}

func TestParseHexAccent(t *testing.T) {
	cases := map[string]uint32{
		"#FF0000":    0xFFFF0000,
		"0xFF0000":   0xFFFF0000,
		"FF0000":     0xFFFF0000,
		"0x80FF0000": 0x80FF0000,
	}
	for in, want := range cases {
		c, ok := ParseHexAccent(in)
		if !ok || uint32(c) != want {
			t.Errorf("ParseHexAccent(%q) = %08X ok=%v, want %08X", in, c, ok, want)
		}
	}
	if _, ok := ParseHexAccent("nope"); ok {
		t.Error("garbage accepted")
	}
}

func TestGnomeNamedAccent(t *testing.T) {
	if c, ok := gnomeNamedAccent("'teal'\n"); !ok || c != 0xFF2190A4 {
		t.Fatalf("teal = %08X ok=%v", c, ok)
	}
	if _, ok := gnomeNamedAccent("'chartreuse'"); ok {
		t.Error("unknown accent accepted")
	}
}

func TestApplyAccentDerivatives(t *testing.T) {
	saved := DefaultTheme
	savedShape := Shape()
	defer func() {
		SetDefaultTheme(saved)
		SetShape(savedShape)
	}()

	if err := LoadPreset("breeze-dark"); err != nil {
		t.Fatal(err)
	}
	before := DefaultButtonTheme.BackgroundActive
	ApplyAccent(DefaultTheme, 0xFFE62D42) // GNOME red
	Apply(DefaultTheme)

	if DefaultTheme.Accent != 0xFFE62D42 || DefaultTheme.Primary != 0xFFE62D42 ||
		DefaultTheme.BorderFocused != 0xFFE62D42 || DefaultTheme.Link != 0xFFE62D42 {
		t.Errorf("accent fields not overridden: accent=%08X primary=%08X borderFocused=%08X link=%08X",
			DefaultTheme.Accent, DefaultTheme.Primary, DefaultTheme.BorderFocused, DefaultTheme.Link)
	}
	if DefaultTheme.Selected != 0xFFE62D42 {
		t.Errorf("opaque Selected = %08X, want the new accent", DefaultTheme.Selected)
	}
	if DefaultButtonTheme.BackgroundActive != 0xFFE62D42 {
		t.Errorf("widget regeneration: Button active = %08X, want red", DefaultButtonTheme.BackgroundActive)
	}
	if DefaultButtonTheme.BackgroundActive == before {
		t.Error("Button active color unchanged after accent override")
	}

	// Translucent selection keeps its alpha (Fluent convention).
	if err := LoadPreset("windows11-dark"); err != nil {
		t.Fatal(err)
	}
	ApplyAccent(DefaultTheme, 0xFFE62D42)
	if a := DefaultTheme.Selected & 0xFF000000; a != 0x33000000 {
		t.Errorf("translucent Selected alpha = %08X, want 33000000", a)
	}
	if rgb := DefaultTheme.Selected & 0x00FFFFFF; rgb != 0x00E62D42 {
		t.Errorf("translucent Selected rgb = %06X, want E62D42", rgb)
	}
}
