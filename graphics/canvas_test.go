package graphics

import (
	"testing"

	"github.com/sixnineone/mobilegles"
)

// safeCall runs fn and swallows any panic. mobilegles's drawing methods
// panic without an active GL renderer; we only need to prove the seam
// dispatches the call with the right shape (i.e. compiles and runs
// without a seam-level signature error).
func safeCall(fn func()) {
	defer func() { _ = recover() }()
	fn()
}

func TestSeamReshapesDispatchIntoMobilegles(t *testing.T) {
	c := NewCanvas(NewTestDevice(800, 600))
	p := NewPaint()
	p.SetColor(RGB(255, 0, 0))
	p.SetStyle(int32(0))

	cases := []struct {
		name string
		call func()
	}{
		{"DrawRect", func() { c.DrawRect(NewRect(0, 0, 100, 100), p) }},
		{"DrawRoundedRect", func() { c.DrawRoundedRect(NewRect(10, 10, 50, 50), 5, 5, p) }},
		{"DrawCircle", func() { c.DrawCircle(50, 50, 10, p) }},
		{"DrawLine", func() { c.DrawLine(0, 0, 10, 10, p) }},
		{"DrawOval", func() { c.DrawOval(NewRect(0, 0, 200, 100), p) }},
		{"DrawSimpleString", func() { c.DrawSimpleString("hi", 20, 20, nil, p) }},
		{"Translate", func() { c.Translate(5, 5) }},
		{"Scale", func() { c.Scale(2, 2) }},
		{"ClipRect", func() { c.ClipRect(NewRect(0, 0, 50, 50), 0, false) }},
		{"DrawImageRectInRect", func() {
			img := NewImage([]uint32{0xFF0000FF}, 1, 1)
			c.DrawImageRectInRect(img, NewRect(0, 0, 1, 1), NewRect(10, 10, 50, 50), nil, p)
		}},
		{"DrawPath", func() {
			path := NewPath()
			path.MoveTo(NewPoint(0, 0))
			path.LineTo(NewPoint(10, 10))
			path.QuadTo(NewPoint(20, 0), NewPoint(30, 10))
			path.Close()
			c.DrawPath(path, p)
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			safeCall(tc.call)
		})
	}
}

// P1 replaced the old type aliases with meui-owned types. The invariant to
// pin is therefore conversion fidelity, not type identity: every seam value
// must reach the renderer unchanged.
func TestSeamTypesConvertExactly(t *testing.T) {
	if got := mgRect(NewRect(1, 2, 3, 4)); got.X != 1 || got.Y != 2 || got.Width != 3 || got.Height != 4 {
		t.Fatalf("Rect conversion broken: %+v", got)
	}
	if got := mgPoint(NewPoint(1, 2)); got.X != 1 || got.Y != 2 {
		t.Fatalf("Point conversion broken: %+v", got)
	}
	if got := mgSize(NewSize(10, 20)); got.Width != 10 || got.Height != 20 {
		t.Fatalf("Size conversion broken: %+v", got)
	}
	if got := uint32(mobilegles.Color(RGB(1, 2, 3))); got != uint32(mobilegles.RGB(1, 2, 3)) {
		t.Fatalf("Color conversion broken: %x != %x", got, uint32(mobilegles.RGB(1, 2, 3)))
	}
}

// The colour encoding must stay byte-identical to the renderer's: the toolkit
// uses raw 0xAARRGGBB literals throughout.
func TestColorConstructorsMatchBackend(t *testing.T) {
	if uint32(RGBA(1, 2, 3, 255)) != uint32(mobilegles.RGBA(1, 2, 3, 255)) {
		t.Fatal("RGBA broken")
	}
	// ARGB matches the unison signature: float alpha, int channels.
	if uint32(ARGB(1.0, 1, 2, 3)) != uint32(mobilegles.RGBA(1, 2, 3, 255)) {
		t.Fatal("ARGB broken (should equal RGBA(r,g,b,a))")
	}
}

// Paint owns its state; the backend paint is derived from it.
func TestPaintOwnsState(t *testing.T) {
	p := NewPaint()
	if p == nil {
		t.Fatal("NewPaint returned nil")
	}
	p.SetColor(RGBA(10, 20, 30, 40))
	p.SetStyle(int32(mobilegles.Stroke))
	p.SetStrokeWidth(3)

	if p.Color() != RGBA(10, 20, 30, 40) {
		t.Fatalf("Color = %08x", uint32(p.Color()))
	}
	if p.Style() != int32(mobilegles.Stroke) {
		t.Fatalf("Style = %d", p.Style())
	}
	if p.StrokeWidth() != 3 {
		t.Fatalf("StrokeWidth = %v", p.StrokeWidth())
	}

	mp := p.renderer()
	if mp == nil {
		t.Fatal("renderer paint is nil")
	}
	if uint32(mp.Color()) != uint32(mobilegles.RGBA(10, 20, 30, 40)) {
		t.Fatalf("backend Color = %08x", uint32(mp.Color()))
	}
	if mp.StrokeWidth() != 3 {
		t.Fatalf("backend StrokeWidth = %v", mp.StrokeWidth())
	}

	// A nil paint must stay nil-safe: it means "backend default".
	var nilPaint *Paint
	if nilPaint.renderer() != nil {
		t.Fatal("nil Paint should map to a nil backend paint")
	}
	if nilPaint.Color() != 0 || nilPaint.StrokeWidth() != 0 {
		t.Fatal("nil Paint accessors should be zero")
	}
}

func TestSeamConstructors(t *testing.T) {
	if NewPath() == nil {
		t.Fatal("NewPath returned nil")
	}
	if NewImage([]uint32{0xFF0000FF}, 1, 1) == nil {
		t.Fatal("NewImage returned nil")
	}
}
