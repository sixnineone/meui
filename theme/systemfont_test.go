package theme

import "testing"

// TestUseSystemFontRepointsTheThemeAndInvalidatesText pins the second half of a
// runtime font change. Two failures are possible and both are silent: leaving the
// theme pointing at the old face (old glyphs, no error), and re-pointing it WITHOUT
// bumping the version (already-laid-out labels keep the face they were built with,
// so half the window changes and half does not).
func TestUseSystemFontRepointsTheThemeAndInvalidatesText(t *testing.T) {
	saved := DefaultTheme
	t.Cleanup(func() { SetDefaultTheme(saved) })

	SetDefaultTheme(DarkTheme())
	before := ThemeVersion()

	// Pretend the theme is pointing somewhere stale.
	DefaultTheme.BaseFont = nil
	UseSystemFont()

	if DefaultTheme.BaseFont == nil {
		t.Fatal("UseSystemFont left the theme with no base font, so every glyph would vanish")
	}
	if DefaultTheme.BaseFont == nil || DefaultTheme.BaseFont.Face() == nil {
		t.Fatal("UseSystemFont produced a font with no face")
	}
	// A ZERO-SIZED base font is the bug this pins: UseSystemFont read
	// Typography.Body.Size before the theme's style defaults were applied, so the
	// size was 0 and every glyph was measured at nothing — measured on the device
	// as base.size=0.00 with a degenerate 5.97px width for "Normal text".
	if got := DefaultTheme.BaseFont.Size(); got <= 0 {
		t.Fatalf("base font size = %v, want > 0: reading Typography.Body.Size before "+
			"withStyleDefaults gives zero", got)
	}
	if want := DefaultTheme.Typography.Body.Size; DefaultTheme.BaseFont.Size() != want {
		t.Errorf("base font size = %v, want the theme's own %v — taking the platform's "+
			"default size would change every glyph's metrics as well as its shape",
			DefaultTheme.BaseFont.Size(), want)
	}
	if ThemeVersion() <= before {
		t.Error("the theme version did not change, so cached text keeps the OLD face — " +
			"half the window would change and half would not")
	}
}
