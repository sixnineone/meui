package platform_test

import (
	"testing"

	"github.com/sixnineone/meui/platform"
)

// Every cursor the toolkit can ask for must be a name the backend maps. The
// mapping lives in mobilegles (SetCursorByName / KnownCursorNames), which this
// package already depends on, so the contract is asserted here.
func TestEveryCursorConstructorIsMappable(t *testing.T) {
	names := platform.CursorNames()
	if len(names) == 0 {
		t.Fatal("platform exposes no cursor names")
	}
	known := map[string]bool{}
	for _, n := range platform.KnownBackendCursorNames() {
		known[n] = true
	}
	for _, n := range names {
		if !known[n] {
			t.Errorf("cursor %q has no backend mapping", n)
		}
	}
	// The constructors and the name list must not drift apart.
	ctors := map[string]*platform.Cursor{
		"arrow":      platform.ArrowCursor(),
		"pointing":   platform.PointingCursor(),
		"text":       platform.TextCursor(),
		"move":       platform.MoveCursor(),
		"crosshair":  platform.CrosshairCursor(),
		"wait":       platform.WaitCursor(),
		"notAllowed": platform.NotAllowedCursor(),
		"resizeH":    platform.ResizeHorizontalCursor(),
		"resizeV":    platform.ResizeVerticalCursor(),
		"resizeLD":   platform.ResizeLeftDiagonalCursor(),
		"resizeRD":   platform.ResizeRightDiagonalCursor(),
	}
	for name, c := range ctors {
		if c == nil || c.Name() != name {
			t.Errorf("constructor for %q produces %q", name, c.Name())
		}
		if !known[name] {
			t.Errorf("constructor name %q is not in the backend list", name)
		}
	}
	for _, n := range names {
		if _, ok := ctors[n]; !ok {
			t.Errorf("name list has %q, constructors do not", n)
		}
	}
}

// A nil cursor means "leave it alone" — the window then falls back to the
// arrow. The constructors must never be nil.
func TestCursorConstructorsAreNeverNil(t *testing.T) {
	for _, c := range []*platform.Cursor{
		platform.ArrowCursor(), platform.PointingCursor(), platform.TextCursor(),
		platform.MoveCursor(), platform.CrosshairCursor(), platform.WaitCursor(),
		platform.NotAllowedCursor(), platform.ResizeHorizontalCursor(),
		platform.ResizeVerticalCursor(), platform.ResizeLeftDiagonalCursor(),
		platform.ResizeRightDiagonalCursor(),
	} {
		if c == nil {
			t.Fatal("a cursor constructor returned nil")
		}
	}
	var nilCursor *platform.Cursor
	if nilCursor.Name() != "" {
		t.Fatal("a nil cursor must report an empty name")
	}
}
