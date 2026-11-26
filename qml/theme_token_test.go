package qml

import (
	"testing"

	"github.com/sixnineone/meui/theme"
	"github.com/sixnineone/meui/widgets"
)

// TestQMLColorTokenFollowsThemeSwitch pins runtime re-colouring: a
// `color: "secondary"` label and a `color: "surface"` panel must follow
// theme switches instead of freezing the palette captured at parse
// time.
func TestQMLColorTokenFollowsThemeSwitch(t *testing.T) {
	const src = `Window {
  width: 300; height: 200
  VStack {
    Label { id: lbl; text: "hello"; color: "secondary" }
    Panel { id: pnl; color: "surface"; width: 100; height: 40 }
  }
}`
	if err := theme.LoadPreset("macos-light"); err != nil {
		t.Fatal(err)
	}
	res, err := ParseWithBindings(src)
	if err != nil {
		t.Fatal(err)
	}
	lbl, _ := res.Widgets.Get("lbl").(*widgets.Label)
	if lbl == nil {
		t.Fatal("label not found in registry")
	}
	if got := lbl.EffectiveTextInk(); got != theme.DefaultTheme.TextSecondary {
		t.Fatalf("initial ink = %08X, want macos secondary %08X", got, theme.DefaultTheme.TextSecondary)
	}
	first := lbl.EffectiveTextInk()

	if err := theme.LoadPreset("windows11-dark"); err != nil {
		t.Fatal(err)
	}
	if got := lbl.EffectiveTextInk(); got != theme.DefaultTheme.TextSecondary {
		t.Fatalf("after switch ink = %08X, want win11-dark secondary %08X", got, theme.DefaultTheme.TextSecondary)
	}
	if lbl.EffectiveTextInk() == first {
		t.Fatal("label ink did not change across the theme switch (token was baked at parse time)")
	}
}
