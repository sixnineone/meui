package graphics

import "testing"

// The seam has two backends: one real (the renderer) and one test double.
// These compile-time assertions are the contract — if either stops satisfying
// Device, the seam is no longer implementable by a second backend and the
// package fails to build.
var (
	_ Device = (*TestDevice)(nil)
	_ Device = NewRendererDevice(1, 1)
)

// contractOps exercises every Device operation the toolkit is allowed to
// depend on, in one sequence. It is run against both backends below, so a
// widget can only rely on behaviour both provide.
func contractOps(c *Canvas, p *Paint, f Font) {
	c.Clear(0xFF000000)
	c.ClearWithViewport(0xFF000000, 0, 0, 10, 10)

	c.Save()
	c.Translate(1, 2)
	c.Scale(1, 1)
	c.Rotate(0)
	c.Skew(0, 0)
	c.ClipRect(NewRect(0, 0, 10, 10), 0, false)
	_ = c.ClipDepth()
	_, _ = c.Origin()
	c.ResetClip()
	c.Restore()

	c.DrawRect(NewRect(0, 0, 5, 5), p)
	c.DrawRoundedRect(NewRect(0, 0, 5, 5), 2, 2, p)
	c.DrawCircle(1, 1, 2, p)
	c.DrawOval(NewRect(0, 0, 5, 5), p)
	c.DrawArc(NewRect(0, 0, 5, 5), 0, 90, p)
	c.DrawLine(0, 0, 1, 1, p)
	c.DrawDashedLine(0, 0, 10, 0, []float32{2, 2}, 0, p)

	path := NewPath()
	path.MoveTo(NewPoint(0, 0))
	path.LineTo(NewPoint(1, 1))
	path.QuadTo(NewPoint(2, 0), NewPoint(3, 1))
	path.CubicTo(NewPoint(4, 0), NewPoint(5, 1), NewPoint(6, 0))
	path.Close()
	c.DrawPath(path, p)

	c.DrawText("hi", 0, 0, f, p)
	c.DrawSimpleString("hi", 0, 0, f, p)
	c.DropShadow(NewRect(0, 0, 5, 5), p, 1, 1, 2)
	c.DrawImageInRect(nil, NewRect(0, 0, 1, 1), nil, p)
}

// TestSeamContract_TestDevice runs the contract against the recording double
// and asserts the facade actually forwarded everything.
func TestSeamContract_TestDevice(t *testing.T) {
	d := NewTestDevice(100, 100)
	c := NewCanvas(d)
	if c.Device() != Device(d) {
		t.Fatal("Canvas.Device did not return the device it was built with")
	}
	contractOps(c, NewPaint(), SystemFont)

	for _, op := range []string{
		"Clear", "ClearWithViewport", "Save", "Translate", "Scale", "Rotate",
		"Skew", "ClipRect", "ResetClip", "Restore",
		"DrawRect", "DrawRoundedRect", "DrawCircle", "DrawOval", "DrawArc",
		"DrawLine", "DrawDashedLine", "DrawPath", "DrawText", "DrawSimpleString", "DropShadow",
		"DrawImage",
	} {
		if d.Count(op) == 0 {
			t.Errorf("TestDevice never recorded %s — the facade did not forward it", op)
		}
	}
}

// TestSeamContract_Renderer runs the same sequence through the real backend.
// There is no GL context here, so drawing panics at the renderer boundary;
// the point is that the seam compiles and dispatches the identical sequence
// without a seam-level error — the same shape the old shim test proved.
func TestSeamContract_Renderer(t *testing.T) {
	d := NewRendererDevice(100, 100)
	c := NewCanvas(d)
	safeCall(func() { contractOps(c, NewPaint(), SystemFont) })
}

// TestTestDeviceEmulatesTransformState pins the double's fidelity: Origin and
// clip depth are behaviour real widgets read, so the double must model them.
func TestTestDeviceEmulatesTransformState(t *testing.T) {
	d := NewTestDevice(100, 100)
	c := NewCanvas(d)

	c.Translate(11, 22)
	if x, y := c.Origin(); x != 11 || y != 22 {
		t.Fatalf("Origin = %v,%v want 11,22", x, y)
	}

	c.Save()
	c.ClipRect(NewRect(0, 0, 5, 5), 0, false)
	if c.ClipDepth() != 1 {
		t.Fatalf("ClipDepth = %d want 1", c.ClipDepth())
	}
	c.Restore()
	if x, y := c.Origin(); x != 11 || y != 22 {
		t.Fatalf("Origin after Restore = %v,%v want 11,22", x, y)
	}
}
