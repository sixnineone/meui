package graphics

import "testing"

// Path recording is what lets any backend receive cubics: the seam stores the
// op, the device decides how to rasterize it.
func TestPathRecordsCubic(t *testing.T) {
	p := NewPath()
	p.MoveTo(NewPoint(0, 0))
	p.LineTo(NewPoint(1, 0))
	p.QuadTo(NewPoint(2, 0), NewPoint(3, 1))
	p.CubicTo(NewPoint(4, 0), NewPoint(5, 1), NewPoint(6, 0))
	p.Close()

	ops := p.Ops()
	if len(ops) != 5 {
		t.Fatalf("recorded %d ops, want 5: %+v", len(ops), ops)
	}
	if ops[3].Kind != PathCubicTo {
		t.Fatalf("op 3 kind = %v, want PathCubicTo", ops[3].Kind)
	}
	if ops[3].P0 != NewPoint(4, 0) || ops[3].P1 != NewPoint(5, 1) || ops[3].P2 != NewPoint(6, 0) {
		t.Fatalf("cubic controls/end not recorded: %+v", ops[3])
	}
	if ops[4].Kind != PathClose {
		t.Fatalf("op 4 kind = %v, want PathClose", ops[4].Kind)
	}

	p.Reset()
	if len(p.Ops()) != 0 {
		t.Fatal("Reset did not clear the recorded ops")
	}
}
