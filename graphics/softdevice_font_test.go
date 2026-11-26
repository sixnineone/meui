package graphics

import "testing"

// TestTheSoftwareDeviceRegistersAFace pins the 2026-11-24 flood:
//
//	ftgc.go:133: open ../resource/font/luxisr.ttf: no such file or directory
//
// draw2d's default font is resolved through a folder relative to its MODULE
// ("../resource/font"), which exists in the source tree and nowhere in a running
// app. When no face is registered, every string the software device draws goes
// through that lookup and draw2d logs the miss — one line per label, forever.
//
// The device must therefore always register a face, whatever the distribution
// calls its font directory.
func TestTheSoftwareDeviceRegistersAFace(t *testing.T) {
	softFont = nil // force the registration path, not a cached face
	dev := NewSoftDevice(160, 32)
	if softFont == nil {
		t.Fatal("the software device registered no face: draw2d would resolve ../resource/font/luxisr.ttf and log a miss for every string")
	}

	// A registered face is not enough — it must reach the pixels.
	dev.Clear(0xFF000000)
	probe := &FontDescriptor{Family: "probe", Size: 18}
	dev.DrawText("Hello", 2, 24, probe.Font(), nil)
	for y := 0; y < 32; y++ {
		for x := 0; x < 160; x++ {
			if p := dev.Image().RGBAAt(x, y); p.R > 0x40 && p.G > 0x40 && p.B > 0x40 {
				return // ink found
			}
		}
	}
	t.Fatal("DrawText painted nothing with the registered face")
}

// TestTheSoftwareDeviceDrawsTextAtTheRequestedPixelSize pins the scale contract:
// the seam's Font.Size is the requested CAP HEIGHT in logical pixels (see
// FontImpl.emPx, which solves for cap == size), and draw2d's SetFontSize takes POINTS
// scaled by DPI/72 — so the device must draw at the font's em, with DPI pinned to 72.
//
// Two wrong answers to keep out: drawing at the raw size (30-40% small, and the
// metrics then disagree with the ink, which is what put centred letters left of their
// circle), and the old default DPI of 92 (1.278x large).
func TestTheSoftwareDeviceDrawsTextAtTheRequestedPixelSize(t *testing.T) {
	const size = 40
	dev := NewSoftDevice(120, 72)
	dev.Clear(0xFF000000)
	probe := &FontDescriptor{Family: "probe", Size: size}
	dev.DrawText("H", 8, 52, probe.Font(), nil)

	minY, maxY := -1, -1
	for y := 0; y < 72; y++ {
		for x := 0; x < 120; x++ {
			if p := dev.Image().RGBAAt(x, y); p.R > 0x40 || p.G > 0x40 || p.B > 0x40 {
				if minY < 0 {
					minY = y
				}
				maxY = y
				break
			}
		}
	}
	if minY < 0 {
		t.Fatal("DrawText painted no ink")
	}
	ink := float32(maxY - minY + 1)
	lo, hi := float32(size)*0.94, float32(size)*1.06
	if ink < lo || ink > hi {
		t.Fatalf("\"H\" at %dpx cap has %.0fpx of ink, want %.0f-%.0f: the software device "+
			"must draw at the font's em (cap height == requested size)",
			size, ink, lo, hi)
	}
}
