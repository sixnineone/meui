package theme

import "testing"

func TestPlatformFromDesktop(t *testing.T) {
	cases := map[string]Platform{
		"GNOME":        PlatformGNOME,
		"ubuntu:GNOME": PlatformGNOME,
		"pop:GNOME":    PlatformGNOME,
		"Budgie:GNOME": PlatformGNOME,
		"KDE":          PlatformKDE,
		"plasma":       PlatformKDE,
		"KDE Plasma":   PlatformKDE,
		"XFCE":         PlatformMeui,
		"":             PlatformMeui,
		"X-Cinnamon":   PlatformMeui,
	}
	for in, want := range cases {
		if got := platformFromDesktop(in); got != want {
			t.Errorf("platformFromDesktop(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAutoPresetIDRegistered(t *testing.T) {
	id := AutoPresetID()
	if _, ok := presetBuilders[id]; !ok {
		t.Fatalf("AutoPresetID() = %q, not in preset registry (IDs: %v)", id, PresetIDs())
	}
}

func TestModeFromProbe(t *testing.T) {
	if got := modeFromProbe(false, true); got != ModeLight {
		t.Errorf("light probe = %v, want ModeLight", got)
	}
	if got := modeFromProbe(true, true); got != ModeDark {
		t.Errorf("dark probe = %v, want ModeDark", got)
	}
	if got := modeFromProbe(false, false); got != ModeDark {
		t.Errorf("unknown probe = %v, want ModeDark (house default)", got)
	}
}

// TestMobilePlatformsAreReachable pins the wiring that makes the mobile presets
// selectable at all. Both are reachable by GOOS — Go reports "android" and "ios" as
// their own values — and a missing case does not fail, it silently falls through to
// the Linux DESKTOP path, which is exactly how a phone ended up with 26px controls.
func TestMobilePlatformsAreReachable(t *testing.T) {
	for _, c := range []struct {
		goos   string
		want   Platform
		preset string
	}{
		{"android", PlatformAndroid, "android-dark"},
		{"ios", PlatformIOS, "ios-dark"},
	} {
		if _, ok := presetBuilders[c.preset]; !ok {
			t.Errorf("%s: preset %q is not registered", c.goos, c.preset)
		}
		// The GOOS mapping is asserted through the same switch DetectPlatform uses,
		// because the failure mode is a missing case rather than a wrong one.
		if got := platformForGOOS(c.goos); got != c.want {
			t.Errorf("GOOS %q maps to %v, want %v", c.goos, got, c.want)
		}
	}
}
