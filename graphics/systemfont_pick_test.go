package graphics

import "testing"

// TestDiscoveryPrefersASansFamily pins the choice discovery makes when the
// candidate paths match nothing: a sans face, not whichever family sorts first.
// On openSUSE the first family is "Adwaita Mono", which would render every label
// in a monospace face — visible, but wrong.
func TestDiscoveryPrefersASansFamily(t *testing.T) {
	scanned := []SystemFontInfo{
		{Family: "Adwaita Mono", Path: "mono.ttf"},
		{Family: "DejaVu Math TeX Gyre", Path: "math.ttf"},
		{Family: "DejaVu Sans", Path: "sans.ttf"},
	}
	got, ok := pickSystemFont(scanned)
	if !ok {
		t.Fatal("pickSystemFont found no font in a non-empty scan")
	}
	if got.Family != "DejaVu Sans" {
		t.Fatalf("picked %q, want DejaVu Sans", got.Family)
	}

	// A scan with none of the preferred families still yields a face rather
	// than leaving the app without one.
	got, ok = pickSystemFont([]SystemFontInfo{{Family: "Some Sans", Path: "some.ttf"}})
	if !ok || got.Family != "Some Sans" {
		t.Fatalf("fallback to the first family: got %+v, ok=%v", got, ok)
	}

	if _, ok := pickSystemFont(nil); ok {
		t.Fatal("an empty scan must report no font")
	}
}
