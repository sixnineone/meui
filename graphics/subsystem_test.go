package graphics

import (
	"testing"

	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/graphics/enums/slant"
	"github.com/sixnineone/meui/graphics/enums/spacing"
	"github.com/sixnineone/meui/graphics/enums/weight"
)

func TestPaintstyleMatchesMobilegles(t *testing.T) {
	if paintstyle.Fill != 0 {
		t.Errorf("paintstyle.Fill = %d, want 0", paintstyle.Fill)
	}
	if paintstyle.Stroke != 1 {
		t.Errorf("paintstyle.Stroke = %d, want 1", paintstyle.Stroke)
	}
}

func TestEnumsAreDistinct(t *testing.T) {
	if pathop.Intersect == pathop.Difference {
		t.Error("pathop values not distinct")
	}
	if slant.Italic == slant.Upright {
		t.Error("slant values not distinct")
	}
	if spacing.Standard == spacing.Monospace {
		t.Error("spacing values not distinct")
	}
	if weight.Regular == weight.Bold {
		t.Error("weight values not distinct")
	}
}

func TestTextStub(t *testing.T) {
	dec := &TextDecoration{Font: SystemFont, OnBackgroundInk: RGB(255, 0, 0)}
	t1 := NewText("hello", dec)
	if t1 == nil {
		t.Fatal("NewText returned nil")
	}
	if t1.decoration != dec {
		t.Error("decoration not stored")
	}
}

func TestTextNilSafe(t *testing.T) {
	var t1 *Text
	t1.Draw(nil, 0, 0)
}

func TestImageConstructorsReturnErrors(t *testing.T) {
	if _, err := NewImageFromDrawing(NewTestDevice(10, 10), 10, 10, 72, nil); err == nil {
		t.Error("NewImageFromDrawing should return error (stub)")
	}
	if _, err := NewImageFromFilePathOrURL("/dev/null", 1); err == nil {
		t.Error("NewImageFromFilePathOrURL should return error (stub)")
	}
}

func TestColorDecodeHex(t *testing.T) {
	c, err := ColorDecode("#ff0000")
	if err != nil {
		t.Fatal(err)
	}
	if uint32(c)>>24 != 0xFF {
		t.Errorf("alpha = %d, want 255", uint32(c)>>24)
	}
	if uint32(c)&0xFF != 0x00 || (uint32(c)>>8)&0xFF != 0x00 || (uint32(c)>>16)&0xFF != 0xFF {
		t.Errorf("color = %08x, want red with full alpha", uint32(c))
	}
}

func TestColorDecodeShortHex(t *testing.T) {
	c, err := ColorDecode("#f00")
	if err != nil {
		t.Fatal(err)
	}
	if uint32(c)&0xFF != 0x00 || (uint32(c)>>16)&0xFF != 0xFF {
		t.Errorf("color = %08x, want red", uint32(c))
	}
}

func TestColorDecodeNamed(t *testing.T) {
	if _, err := ColorDecode("red"); err != nil {
		t.Errorf("ColorDecode(red) failed: %v", err)
	}
	if _, err := ColorDecode("notacolor"); err == nil {
		t.Error("ColorDecode(notacolor) should fail")
	}
}

func TestColorDecodeInvalid(t *testing.T) {
	if _, err := ColorDecode(""); err == nil {
		t.Error("ColorDecode(\"\") should fail")
	}
}

func TestFontFaceMatching(t *testing.T) {
	// Embedded Roboto resolves (production parity: unison embeds Roboto).
	roboto := MatchFontFace("Roboto", weight.Light, spacing.Standard, slant.Upright)
	if roboto == nil {
		t.Fatal("MatchFontFace(Roboto) returned nil")
	}
	f := roboto.Font(14)
	if f == nil {
		t.Fatal("Roboto.Font(14) returned nil")
	}
	// Non-embedded families return nil, exactly like production's
	// unison/skia path on systems without the family installed.
	if inter := MatchFontFace("Inter", weight.Light, spacing.Standard, slant.Upright); inter != nil {
		t.Error("MatchFontFace(Inter) should return nil (not embedded)")
	}
	all, mono := AllFontFaces()
	if all != nil || mono != nil {
		t.Error("AllFontFaces should return nil (stub)")
	}
}
