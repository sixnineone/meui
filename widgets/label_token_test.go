package widgets

import (
	"testing"

	"github.com/sixnineone/meui/theme"
)

// TestLabelColorTokenLive pins that a token label re-resolves per paint
// while a literal colour stays put.
func TestLabelColorTokenLive(t *testing.T) {
	saved := theme.DefaultTheme
	defer theme.SetDefaultTheme(saved)

	if err := theme.LoadPreset("macos-light"); err != nil {
		t.Fatal(err)
	}
	l := NewLabel()
	l.SetColorToken("accent")
	if got, want := l.EffectiveTextInk(), theme.DefaultTheme.Accent; got != want {
		t.Fatalf("ink = %08X, want accent %08X", got, want)
	}
	first := l.EffectiveTextInk()

	if err := theme.LoadPreset("breeze-dark"); err != nil {
		t.Fatal(err)
	}
	if got, want := l.EffectiveTextInk(), theme.DefaultTheme.Accent; got != want {
		t.Fatalf("after switch ink = %08X, want breeze accent %08X", got, want)
	}
	if l.EffectiveTextInk() == first {
		t.Fatal("accent token did not follow the theme switch")
	}

	// A literal colour is unaffected by theme switches.
	lit := NewLabel()
	lit.SetColor(0xFF123456)
	if got := lit.EffectiveTextInk(); got != 0xFF123456 {
		t.Fatalf("literal ink = %08X, want FF123456", got)
	}
}
