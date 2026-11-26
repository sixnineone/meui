package graphics

import "testing"

// The blend mode set on a Paint must reach the device with the draw: that is
// what makes the capability real rather than stored-but-ignored.
func TestPaintBlendModeReachesTheDevice(t *testing.T) {
	d := NewTestDevice(10, 10)
	c := NewCanvas(d)
	p := NewPaint()
	if p.BlendMode() != BlendModeNormal {
		t.Fatalf("NewPaint blend = %v, want Normal", p.BlendMode())
	}

	p.SetBlendMode(BlendModeMultiply)
	c.DrawRect(NewRect(0, 0, 1, 1), p)
	if d.LastBlend != BlendModeMultiply {
		t.Fatalf("device saw blend %v, want Multiply", d.LastBlend)
	}

	p.SetBlendMode(BlendModeScreen)
	c.DrawRect(NewRect(0, 0, 1, 1), p)
	if d.LastBlend != BlendModeScreen {
		t.Fatalf("device saw blend %v, want Screen", d.LastBlend)
	}
}
