package theme

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sixnineone/meui/graphics"
)

// TestImportThemeKeyPrecedenceDeterministic pins the fix for run-to-run
// theme colour variance: several .theme.json UI keys write the same
// Theme field (Table.background and TextField.background both feed
// Background; SidePanel.background and StatusBar.background both feed
// Surface). Import used to apply them in Go map iteration order —
// randomized per process — so the same file resolved to different
// colours on every launch. Import must now apply a fixed precedence,
// and repeated imports of the same file must agree byte for byte.
func TestImportThemeKeyPrecedenceDeterministic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "conflict.theme.json")
	const content = `{
		"name": "Conflict",
		"dark": true,
		"colors": {},
		"ui": {
			"Editor.background": "#111111",
			"TextField.background": "#222222",
			"TextArea.background": "#333333",
			"Table.background": "#444444",
			"StatusBar.background": "#555555",
			"SidePanel.background": "#666666",
			"Label.foreground": "#777777",
			"TextField.foreground": "#888888"
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	var firstBG, firstSurface, firstText graphics.Color
	for i := 0; i < 30; i++ {
		th, err := ImportIntelliJThemeJSON(path)
		if err != nil {
			t.Fatalf("import %d: %v", i, err)
		}
		if i == 0 {
			firstBG, firstSurface, firstText = th.Background, th.Surface, th.TextPrimary
			continue
		}
		if th.Background != firstBG || th.Surface != firstSurface || th.TextPrimary != firstText {
			t.Fatalf("import %d nondeterministic: bg=%08X surface=%08X text=%08X want bg=%08X surface=%08X text=%08X",
				i, th.Background, th.Surface, th.TextPrimary, firstBG, firstSurface, firstText)
		}
	}

	// Precedence: content panels beat editors/inputs for the global
	// background; tool chrome beats inputs for the surface; the general
	// label colour beats the text-field colour for primary text.
	if firstBG != 0xFF444444 {
		t.Errorf("Background = %08X, want FF444444 (Table.background must beat Editor/TextField/TextArea)", firstBG)
	}
	if firstSurface != 0xFF666666 {
		t.Errorf("Surface = %08X, want FF666666 (SidePanel.background must beat StatusBar/TextArea)", firstSurface)
	}
	if firstText != 0xFF777777 {
		t.Errorf("TextPrimary = %08X, want FF777777 (Label.foreground must beat TextField.foreground)", firstText)
	}
}

// The splitter mapping an imported L&F gets: one line in the theme's border
// colour, the accent while hovered and — because an imported palette's
// separator is the same tone as its chrome, which leaves a hover-only
// highlight invisible until the pointer finds it — the accent always. No
// band fill and no grip marks: those were the bright strip and the extra
// full-length lines the owner reported.
func TestImportedStackPanelThemeMapsTheSplitter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "split.theme.json")
	const content = `{
		"name": "Split",
		"dark": true,
		"colors": {},
		"ui": {
			"Editor.background": "#101010",
			"Separator.separatorColor": "#202020",
			"Component.focusColor": "#303030"
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	th, err := ImportIntelliJThemeJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	// ApplyIntelliJTheme writes THROUGH the global pointer, so give it a
	// scratch object: restoring the pointer alone would leave the imported
	// values in the object every other test reads.
	saved := DefaultStackPanelTheme
	DefaultStackPanelTheme = &StackPanelTheme{}
	defer func() { DefaultStackPanelTheme = saved }()
	ApplyIntelliJTheme(th)

	got := DefaultStackPanelTheme
	if got.SplitterInk != th.Border {
		t.Errorf("SplitterInk = %#08x, want the theme Border %#08x", uint32(got.SplitterInk), uint32(th.Border))
	}
	if got.SplitterHoverInk != th.Accent {
		t.Errorf("SplitterHoverInk = %#08x, want the theme Accent %#08x", uint32(got.SplitterHoverInk), uint32(th.Accent))
	}
	if !got.HoverColorAlwaysOn {
		t.Error("HoverColorAlwaysOn = false, want true for an imported L&F")
	}
	if a := uint32(got.BackgroundInk) >> 24; a > 0 {
		t.Errorf("BackgroundInk = %#08x, want no band fill", uint32(got.BackgroundInk))
	}
	if a := uint32(got.GripInk) >> 24; a > 0 {
		t.Errorf("GripInk = %#08x, want no grip marks", uint32(got.GripInk))
	}
}

// Presets keep the subtle divider: the accent-always look belongs to imported
// L&Fs, and ApplyIntelliJTheme resolves both. The flag lives on the Theme so
// the two paths can differ.
func TestPresetKeepsTheSubtleSplitter(t *testing.T) {
	build, ok := presetBuilders["macos-light"]
	if !ok {
		t.Skip("macos-light preset not registered")
	}
	th := build()
	if th.SplitterHoverColorAlwaysOn {
		t.Error("preset asks for accent splitters, want the hover-only highlight")
	}
	saved := DefaultStackPanelTheme
	DefaultStackPanelTheme = &StackPanelTheme{}
	defer func() { DefaultStackPanelTheme = saved }()
	ApplyIntelliJTheme(th)
	if DefaultStackPanelTheme.HoverColorAlwaysOn {
		t.Error("a preset resolved HoverColorAlwaysOn = true, want false")
	}
}
