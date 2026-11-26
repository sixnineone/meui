package graphics

import "testing"

// Paint defaults have to match the renderer's, or a widget that never touches
// these fields still changes appearance depending on which module built it.
func TestPaintDefaults_MiterAndAntialias(t *testing.T) {
	p := NewPaint()
	if got := p.MiterLimit(); got != 4 {
		t.Errorf("default MiterLimit = %v, want 4", got)
	}
	if !p.Antialias() {
		t.Error("default Antialias = false, want true (the SDF edge is antialiased)")
	}
}

func TestPaintMiterLimitRoundTrips(t *testing.T) {
	p := NewPaint()
	for _, want := range []float32{1, 2.5, 10, 100} {
		p.SetMiterLimit(want)
		if got := p.MiterLimit(); got != want {
			t.Errorf("MiterLimit = %v, want %v", got, want)
		}
	}
	// Zero and negatives mean "unset", not "no miter at all": a zero limit
	// would silently bevel every sharp corner.
	p.SetMiterLimit(0)
	if got := p.MiterLimit(); got != 4 {
		t.Errorf("MiterLimit after SetMiterLimit(0) = %v, want the 4 default", got)
	}
	p.SetMiterLimit(-3)
	if got := p.MiterLimit(); got != 4 {
		t.Errorf("MiterLimit after a negative set = %v, want the 4 default", got)
	}
}

func TestPaintAntialiasRoundTrips(t *testing.T) {
	p := NewPaint()
	p.SetAntialias(false)
	if p.Antialias() {
		t.Error("Antialias = true after SetAntialias(false)")
	}
	p.SetAntialias(true)
	if !p.Antialias() {
		t.Error("Antialias = false after SetAntialias(true)")
	}
}

// The seam's job is to carry the state to the device; a field that stops at
// the wrapper is a capability that does not exist.
func TestMiterLimitAndAntialiasReachTheDevice(t *testing.T) {
	d := NewTestDevice(100, 100)
	c := NewCanvas(d)
	p := NewPaint()
	p.SetStrokeWidth(4)
	p.SetMiterLimit(9)
	p.SetAntialias(false)

	c.DrawLine(0, 0, 10, 10, p)
	if d.LastMiterLimit != 9 {
		t.Errorf("device saw miter limit %v, want 9", d.LastMiterLimit)
	}
	if d.LastAntialias {
		t.Error("device saw antialias on, want it off")
	}

	c.DrawPath(NewPath(), p)
	if d.LastMiterLimit != 9 || d.LastAntialias {
		t.Errorf("DrawPath: device saw miter %v antialias %v, want 9/false", d.LastMiterLimit, d.LastAntialias)
	}
}

// A default paint must still carry the defaults all the way down, not the
// zero values a lazily-built backend paint would start from.
func TestDefaultPaintReachesDeviceAsDefaults(t *testing.T) {
	d := NewTestDevice(10, 10)
	NewCanvas(d).DrawLine(0, 0, 1, 1, NewPaint())
	if d.LastMiterLimit != 4 {
		t.Errorf("device saw miter limit %v, want 4", d.LastMiterLimit)
	}
	if !d.LastAntialias {
		t.Error("device saw antialias off for a default paint")
	}
}

func TestPaintGlobalAlphaDefaultsToOne(t *testing.T) {
	if got := NewPaint().GlobalAlpha(); got != 1 {
		t.Errorf("default GlobalAlpha = %v, want 1 (no modulation)", got)
	}
}

func TestPaintGlobalAlphaClamps(t *testing.T) {
	p := NewPaint()
	for _, tc := range []struct{ set, want float32 }{
		{0.5, 0.5}, {0, 0}, {1, 1}, {2, 1}, {-1, 1},
	} {
		p.SetGlobalAlpha(tc.set)
		if got := p.GlobalAlpha(); got != tc.want {
			t.Errorf("SetGlobalAlpha(%v) -> %v, want %v", tc.set, got, tc.want)
		}
	}
}

// Zero is a real value ("invisible"), not "unset"; a paint set to 0 that reads
// back as 1 would draw content the caller asked to hide.
func TestPaintGlobalAlphaZeroIsPreserved(t *testing.T) {
	p := NewPaint()
	p.SetGlobalAlpha(0)
	if got := p.GlobalAlpha(); got != 0 {
		t.Errorf("GlobalAlpha = %v, want 0", got)
	}
}

func TestGlobalAlphaReachesTheDevice(t *testing.T) {
	d := NewTestDevice(100, 100)
	c := NewCanvas(d)
	p := NewPaint()
	p.SetGlobalAlpha(0.25)

	c.DrawRect(NewRect(0, 0, 10, 10), p)
	if d.LastGlobalAlpha != 0.25 {
		t.Errorf("DrawRect: device saw global alpha %v, want 0.25", d.LastGlobalAlpha)
	}
	c.DrawLine(0, 0, 10, 10, p)
	if d.LastGlobalAlpha != 0.25 {
		t.Errorf("DrawLine: device saw global alpha %v, want 0.25", d.LastGlobalAlpha)
	}
}

// ClipPath has to reach the device with its ops intact; a clip that stops at
// the wrapper would clip nothing.
func TestClipPathReachesTheDevice(t *testing.T) {
	d := NewTestDevice(100, 100)
	c := NewCanvas(d)

	p := NewPath()
	p.MoveTo(NewPoint(0, 0))
	p.LineTo(NewPoint(10, 0))
	p.LineTo(NewPoint(5, 10))
	p.Close()

	c.ClipPath(p)
	if got := d.Count("ClipPath"); got != 1 {
		t.Errorf("ClipPath calls = %d, want 1", got)
	}
	if d.LastClipPathOps != 4 {
		t.Errorf("device saw %d path ops, want 4", d.LastClipPathOps)
	}
}

// An empty or nil path cannot describe a region, so it must not be reported as
// an applied clip — otherwise a caller would trust a clip that is not there.
func TestClipPathRejectsEmptyPath(t *testing.T) {
	d := NewTestDevice(10, 10)
	c := NewCanvas(d)
	if c.ClipPath(nil) {
		t.Error("ClipPath(nil) reported a clip was applied")
	}
	if c.ClipPath(NewPath()) {
		t.Error("ClipPath(empty) reported a clip was applied")
	}
}
