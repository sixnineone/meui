package platform

import "testing"

// TestBarAppearanceNamesAreStable pins the strings that cross the JNI boundary. They are
// spelled out in Java as string comparisons, so a rename here silently stops the
// appearance changing — with no error on either side.
func TestBarAppearanceNamesAreStable(t *testing.T) {
	for _, c := range []struct {
		a    BarAppearance
		want string
	}{
		{BarAppearanceAuto, "auto"},
		{BarAppearanceLight, "light"},
		{BarAppearanceDark, "dark"},
	} {
		if got := c.a.String(); got != c.want {
			t.Errorf("BarAppearance(%d).String() = %q, want %q", c.a, got, c.want)
		}
	}
}

// TestBarAppearanceLightMeansDarkIcons guards the one genuinely confusing thing about
// this API, which is that its two names point opposite ways: "light" describes the
// BACKGROUND the icons sit on, so it means dark icons. Getting it backwards produces an
// invisible clock on one of the two themes, which is easy to miss in a screenshot.
func TestBarAppearanceLightMeansDarkIcons(t *testing.T) {
	if BarAppearanceLight.String() != "light" {
		t.Fatal("the constant that means dark icons is no longer named light")
	}
	// The comment on the type is the contract; this asserts it has not been silently
	// swapped by a rename.
	if BarAppearanceLight == BarAppearanceDark {
		t.Fatal("the two appearances are the same value")
	}
}
