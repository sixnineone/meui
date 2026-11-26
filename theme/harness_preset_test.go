package theme

import "testing"

// TestTheHarnessPresetIsLightAndMeasured: the preset exists to be the reference look, so
// the two things that make it that way are asserted rather than assumed — it is a LIGHT
// preset (the tree's other six are dark or mid-tone) and its accent is the reference's
// measured one.
func TestTheHarnessPresetIsLightAndMeasured(t *testing.T) {
	build, ok := presetBuilders["harness-light"]
	if !ok {
		t.Fatal("the harness-light preset is not registered")
	}
	th := build()
	if th.Meta.Mode != ModeLight {
		t.Errorf("harness-light is mode %v; the reference is a light interface", th.Meta.Mode)
	}
	if uint32(th.Accent) != 0xFF4176E6 {
		t.Errorf("the accent is #%08X, want #FF4176E6 — the most saturated colour in the reference", uint32(th.Accent))
	}
	if uint32(th.TextPrimary) != 0xFF0F1115 {
		t.Errorf("the text is #%08X, want #FF0F1115", uint32(th.TextPrimary))
	}
	// A light preset that is not actually light is the failure that reads as a rendering
	// bug: a dark app with a light name. Compared on the red channel, because the
	// reference is a near-WHITE surface and "greater than" on the packed value would flag
	// white as not-light — which is how the first version of this assertion failed.
	if r := (uint32(th.Background) >> 16) & 0xFF; r < 0xC0 {
		t.Errorf("the window is #%08X (red %d), which is not a light surface", uint32(th.Background), r)
	}
}

// TestTheMarkdownPaletteFollowsTheMode, and can be undone: the renderer palette is a
// global, so lightening it is a process-wide change and going back has to be possible —
// otherwise a test that lightened a document leaves every later test looking at a light one
// and nothing says so.
func TestTheMarkdownPaletteFollowsTheMode(t *testing.T) {
	saved := DefaultMarkdownTheme
	t.Cleanup(func() { DefaultMarkdownTheme = saved })

	UseMarkdownPalette(ModeLight)
	if !CurrentMarkdownPaletteIsLight() {
		t.Error("asking for the light palette did not give a light one")
	}
	if uint32(DefaultMarkdownTheme.TextInk) != 0xFF0F1115 {
		t.Errorf("the light body text is #%08X, want the reference's #FF0F1115", uint32(DefaultMarkdownTheme.TextInk))
	}

	UseMarkdownPalette(ModeDark)
	if CurrentMarkdownPaletteIsLight() {
		t.Error("asking for the dark palette left a light one in place")
	}
}

// TestTheLightPaletteIsACopy: a widget that changes a metric must not change the built-in
// palette, and the two presets share those built-ins.
func TestTheLightPaletteIsACopy(t *testing.T) {
	saved := DefaultMarkdownTheme
	t.Cleanup(func() { DefaultMarkdownTheme = saved })

	UseMarkdownPalette(ModeLight)
	DefaultMarkdownTheme.CodePadX = 999
	if LightMarkdownTheme.CodePadX == 999 {
		t.Error("changing a copy's metric changed the built-in light palette")
	}
}
