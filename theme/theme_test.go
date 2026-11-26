package theme

import "testing"

// TestOnThemeChangedMultipleListeners guards the regression where
// OnThemeChanged was a single exported func variable: any module that
// assigned it (a chart package re-deriving its palette, for example)
// replaced the window layer's repaint callback, so theme switches
// stopped repainting the UI. Registration must be additive and every
// listener must fire. Parity with flipmode.com/meui/theme.
func TestOnThemeChangedMultipleListeners(t *testing.T) {
	// Isolate from other tests / package init registrations.
	themeChangedMu.Lock()
	saved := themeChangedListeners
	themeChangedListeners = nil
	themeChangedMu.Unlock()
	defer func() {
		themeChangedMu.Lock()
		themeChangedListeners = saved
		themeChangedMu.Unlock()
	}()

	var a, b, c int
	OnThemeChanged(func() { a++ })
	OnThemeChanged(func() { b++ })
	OnThemeChanged(nil) // must be ignored

	SetDefaultTheme(DarkTheme())
	if a != 1 || b != 1 {
		t.Fatalf("listeners fired a=%d b=%d, want 1 and 1", a, b)
	}

	// A late registrant joins the next round; earlier ones still fire.
	OnThemeChanged(func() { c++ })
	SetDefaultTheme(LightTheme())
	if a != 2 || b != 2 || c != 1 {
		t.Fatalf("second switch a=%d b=%d c=%d, want 2 2 1", a, b, c)
	}
}

// TestThemeVersionIncrements guards cache invalidation on theme change.
func TestThemeVersionIncrements(t *testing.T) {
	before := ThemeVersion()
	SetDefaultTheme(DarkTheme())
	if got := ThemeVersion(); got <= before {
		t.Fatalf("ThemeVersion did not increment: %d -> %d", before, got)
	}
}
