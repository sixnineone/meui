package keymap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sixnineone/mobilegles"
)

func TestParseSpec(t *testing.T) {
	cases := []struct {
		spec    string
		wantKey mobilegles.Keycode
		wantMod mobilegles.KeyMod
	}{
		{"Escape", mobilegles.KeyEscape, 0},
		{"esc", mobilegles.KeyEscape, 0},
		{"Ctrl+W", mobilegles.KeyW, mobilegles.ModCtrl},
		{"w+ctrl", mobilegles.KeyW, mobilegles.ModCtrl}, // order-insensitive
		{"Ctrl+Shift+F", mobilegles.KeyF, mobilegles.ModCtrl | mobilegles.ModShift},
		{"Alt+Option+F1", mobilegles.Keycode(58), mobilegles.ModAlt}, // aliases stack
		{"Meta+P", mobilegles.KeyP, mobilegles.ModMeta},
		{"0", 39, 0},
		{"A", mobilegles.KeyA, 0},
	}
	for _, c := range cases {
		key, mods, err := ParseSpec(c.spec)
		if err != nil {
			t.Errorf("ParseSpec(%q): %v", c.spec, err)
			continue
		}
		if key != c.wantKey || mods != c.wantMod {
			t.Errorf("ParseSpec(%q) = (%d,%d), want (%d,%d)", c.spec, key, mods, c.wantKey, c.wantMod)
		}
	}
	for _, bad := range []string{"", "Ctrl", "Ctrl++", "Bogus+W", "Ctrl+Bogus"} {
		if _, _, err := ParseSpec(bad); err == nil {
			t.Errorf("ParseSpec(%q) should fail", bad)
		}
	}
}

func TestFormatSpecRoundTrip(t *testing.T) {
	cases := []struct {
		key  mobilegles.Keycode
		mods mobilegles.KeyMod
		want string
	}{
		{mobilegles.KeyEscape, 0, "Escape"},
		{mobilegles.KeyW, mobilegles.ModCtrl, "Ctrl+W"},
		{mobilegles.KeyF, mobilegles.ModCtrl | mobilegles.ModShift, "Ctrl+Shift+F"},
		{39, 0, "0"},
		{mobilegles.Keycode(99), 0, "Key99"}, // unknown keys stay recoverable
	}
	for _, c := range cases {
		got := FormatSpec(c.key, c.mods)
		if got != c.want {
			t.Errorf("FormatSpec(%d,%d) = %q, want %q", c.key, c.mods, got, c.want)
		}
		key, mods, err := ParseSpec(got)
		if err != nil || key != c.key || mods != c.mods {
			t.Errorf("round-trip %q: (%d,%d) err=%v", got, key, mods, err)
		}
	}
}

func TestMatchAndMatchAll(t *testing.T) {
	m := Defaults()
	// Esc matches exit-fullscreen first, then window.close (MatchAll
	// order drives contextual handling in the shim).
	acts := m.MatchAll(mobilegles.KeyEscape, 0)
	if len(acts) != 2 || acts[0] != ActionExitFullscreen || acts[1] != ActionCloseWindow {
		t.Errorf("Esc MatchAll = %v, want [exit-fullscreen close-window]", acts)
	}
	if a, ok := m.Match(mobilegles.KeyEscape, 0); !ok || a != ActionExitFullscreen {
		t.Errorf("Esc Match = %q,%v", a, ok)
	}
	// Wrong modifier: no match.
	if _, ok := m.Match(mobilegles.KeyEscape, mobilegles.ModCtrl); ok {
		t.Error("Ctrl+Esc should not match the plain-Esc binding")
	}
	// Ctrl+W matches close-tab only.
	if a, ok := m.Match(mobilegles.KeyW, mobilegles.ModCtrl); !ok || a != ActionCloseTab {
		t.Errorf("Ctrl+W Match = %q,%v", a, ok)
	}
	// Bind replaces the action's binding, not others.
	m.MustBind(ActionCloseTab, "Ctrl+K")
	if a, ok := m.Match(mobilegles.KeyW, mobilegles.ModCtrl); ok && a == ActionCloseTab {
		t.Error("close-tab still on Ctrl+W after rebinding to Ctrl+K")
	}
	if a, ok := m.Match(mobilegles.KeyK, mobilegles.ModCtrl); !ok || a != ActionCloseTab {
		t.Errorf("close-tab on Ctrl+K = %q,%v", a, ok)
	}
}

func TestSaveLoadOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keymap.json")

	orig := Defaults()
	orig.MustBind(ActionFocusSearch, "Ctrl+K")
	if err := orig.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded := Defaults() // fresh seeds
	n, err := loaded.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 {
		t.Errorf("loaded %d bindings, want 4", n)
	}
	// Overrides applied…
	if a, ok := loaded.Match(mobilegles.KeyK, mobilegles.ModCtrl); !ok || a != ActionFocusSearch {
		t.Errorf("focus-search = %q,%v, want Ctrl+K", a, ok)
	}
	// …and untouched seeds stand.
	if a, ok := loaded.Match(mobilegles.KeyEscape, 0); !ok || a != ActionExitFullscreen {
		t.Errorf("exit-fullscreen = %q,%v", a, ok)
	}
}

func TestLoadOverridesFromEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "km.json")
	os.WriteFile(path, []byte(`{"window.close": "Ctrl+Shift+Q"}`), 0o644)
	t.Setenv("MEUI_KEYMAP", path)

	m := New()
	LoadOverrides(m) // New() starts empty: file defines everything it names
	if a, ok := m.Match(mobilegles.KeyQ, mobilegles.ModCtrl|mobilegles.ModShift); !ok || a != ActionCloseWindow {
		t.Errorf("window.close = %q,%v, want Ctrl+Shift+Q", a, ok)
	}
}

func TestLoadMissingFileIsNoop(t *testing.T) {
	m := Defaults()
	before := len(m.All())
	if n, err := m.Load(filepath.Join(t.TempDir(), "absent.json")); err == nil || n != 0 {
		t.Errorf("Load(absent) = %d, %v — want 0, error", n, err)
	}
	if got := len(m.All()); got != before {
		t.Errorf("bindings changed on failed load: %d -> %d", before, got)
	}
}
