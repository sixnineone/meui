package graphics

import (
	"strings"
	"testing"

	"github.com/sixnineone/meui/graphics/enums/slant"
	"github.com/sixnineone/meui/graphics/enums/spacing"
	"github.com/sixnineone/meui/graphics/enums/weight"
)

// TestAnAppSuppliedFamilyResolves pins the custom-font path: a scene ships a face,
// registers it under a name, and `font.family` resolves through MatchFontFace — while
// an unregistered family still returns nil, so the "only Roboto" rule that other code
// relies on is unchanged.
func TestAnAppSuppliedFamilyResolves(t *testing.T) {
	var path string
	for _, f := range ListSystemFonts() {
		if strings.HasSuffix(strings.ToLower(f.Path), ".ttf") {
			path = f.Path
			break
		}
	}
	if path == "" {
		t.Skip("no TTF on this machine to register")
	}
	const family = "Test Display Family"
	if err := RegisterFontFamily(family, path, weight.Bold, slant.Upright); err != nil {
		t.Fatalf("register: %v", err)
	}
	found := false
	for _, name := range RegisteredFamilies() {
		if name == family {
			found = true
		}
	}
	if !found {
		t.Fatalf("RegisteredFamilies = %v, want %q", RegisteredFamilies(), family)
	}
	face := MatchFontFace(family, weight.Regular, spacing.Standard, slant.Upright)
	if face == nil {
		t.Fatal("a registered family did not resolve")
	}
	if got := face.Font(14).Face().Family(); got != family {
		t.Fatalf("resolved family = %q, want %q", got, family)
	}
	if MatchFontFace("Definitely Not Registered", weight.Regular, spacing.Standard, slant.Upright) != nil {
		t.Fatal("an unregistered family resolved — the fallback contract changed")
	}
}
