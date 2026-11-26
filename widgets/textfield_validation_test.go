package widgets

// A field that is wrong AND focused must read as wrong: the ring takes the state's ink,
// because an accent ring drawn over a red border says "focused, looks fine".

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

func TestAnInvalidFocusedFieldRingsInTheErrorInk(t *testing.T) {
	theme.SetDefaultTheme(theme.DefaultTheme)
	f := NewTextField()
	f.SetText("not-an-email")
	f.SetFrameRect(graphics.NewRect(20, 20, 200, 40))
	f.Focused = true
	f.SetValidationState("error")

	canvas, dev := graphics.NewSoftwareCanvas(260, 100)
	canvas.Clear(0xFFFFFFFF)
	f.Paint(canvas, graphics.NewRect(20, 20, 200, 40))

	// The ring is drawn outside the border; take the most saturated ink on the left strip.
	img := dev.Image()
	var best graphics.Color
	for x := 0; x < 20; x++ {
		for y := 20; y < 60; y++ {
			r, g, b, _ := img.RGBAAt(x, y).RGBA()
			if r>>8 > 0xF0 && g>>8 > 0xF0 && b>>8 > 0xF0 {
				continue // page
			}
			px := graphics.Color(0xFF000000 | (r>>8)<<16 | (g>>8)<<8 | b>>8)
			if best == 0 || (px&0xFF) > (best&0xFF) {
				best = px
			}
		}
	}
	if best == 0 {
		t.Fatal("no focus ring was drawn outside the field")
	}
	r := (best >> 16) & 0xFF
	b := best & 0xFF
	if r <= b {
		t.Fatalf("the focused invalid field's ring is #%06X — not the error ink", uint32(best)&0xFFFFFF)
	}
}
