package graphics

import (
	"os"
	"testing"
)

// TestReloadSystemFontPicksUpANewFace is the regression test for a documented
// feature that did not work: LoadFontFace says it "lets applications ship their
// own UI fonts", but SystemFont is resolved once at package init and a widget
// captures the face it was built with — so installing a font later changed the
// underlying face and nothing an app could see.
//
// It is an internal test because the point is the resolved FACE, which lives in
// unexported fields; comparing pointers alone would pass even if the wrapper were
// rebuilt around the old font.
func TestReloadSystemFontPicksUpANewFace(t *testing.T) {
	const sans = "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"
	const mono = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"
	for _, p := range []string{sans, mono} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("needs two real fonts to tell them apart: %v", err)
		}
	}

	if _, err := LoadFontFace(sans); err != nil {
		t.Fatalf("LoadFontFace(%s): %v", sans, err)
	}
	first := ReloadSystemFont()
	if first == nil || first.gl == nil {
		t.Fatal("ReloadSystemFont returned no font, so text would vanish")
	}
	if SystemFont != first {
		t.Error("SystemFont must BE the reloaded font, not a copy of the old one")
	}

	if _, err := LoadFontFace(mono); err != nil {
		t.Fatalf("LoadFontFace(%s): %v", mono, err)
	}
	second := ReloadSystemFont()
	// Compare the parsed FACE, not the family: mobilegles.SystemFont() labels every
	// system font "system" (a fixed string, in NewFont), so the family is the same
	// for every font on the machine and cannot discriminate.
	if second.face == first.face {
		t.Error("after installing a different font, the resolved face is unchanged — " +
			"the reload did not pick it up, and every widget keeps the old glyphs")
	}
	if MonospacedFont != second {
		t.Error("MonospacedFont must follow SystemFont; it is a snapshot otherwise")
	}
}
