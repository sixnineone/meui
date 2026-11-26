package widgets

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
)

// TestScrollBarThumbReachesTrackEnds: with absolute-pixel units
// (Max = content extent, VisibleAmount = viewport extent) the thumb must
// start flush at the track top at Value=0 and end flush at the track
// bottom at full scroll — position is measured against the scrollable
// range (Max-VisibleAmount), not the content extent.
func TestScrollBarThumbReachesTrackEnds(t *testing.T) {
	sb := NewScrollBar()
	sb.SetOrientation(Vertical)
	sb.Min = 0
	sb.Max = 400           // content extent
	sb.VisibleAmount = 100 // viewport extent
	track := float32(300)
	sb.SetFrameRect(graphics.NewRect(0, 0, 14, track))

	// Full scroll: thumb bottom flush with track bottom.
	sb.SetValue(400) // overshoot must clamp to scrollable range
	if sb.Value != 300 {
		t.Fatalf("SetValue overshoot: value %v, want clamp at 300", sb.Value)
	}
	tr := sb.thumbRect()
	if tr.Y+tr.Height != track {
		t.Errorf("full scroll: thumb bottom %v, want track bottom %v", tr.Y+tr.Height, track)
	}

	// Top: thumb flush with track top.
	sb.SetValue(0)
	tr = sb.thumbRect()
	if tr.Y != 0 {
		t.Errorf("top: thumb Y %v, want 0", tr.Y)
	}

	// Midpoint: thumb centered proportionally.
	sb.SetValue(150)
	tr = sb.thumbRect()
	thumbH := track * (100.0 / 400.0) // 75
	wantY := 0.5 * (track - thumbH)   // 112.5
	if tr.Y != wantY {
		t.Errorf("mid: thumb Y %v, want %v", tr.Y, wantY)
	}
	if tr.Height != thumbH {
		t.Errorf("mid: thumb height %v, want %v", tr.Height, thumbH)
	}
}

// TestScrollBarPageClickClamp: track clicks page by VisibleAmount and
// clamp to the scrollable range.
func TestScrollBarPageClickClamp(t *testing.T) {
	sb := NewScrollBar()
	sb.SetOrientation(Vertical)
	sb.Min = 0
	sb.Max = 400
	sb.VisibleAmount = 100
	sb.SetFrameRect(graphics.NewRect(0, 0, 14, 300))
	sb.SetValue(0)

	// Click below the thumb (page down).
	sb.OnMouseDown(7, 290, 0)
	if sb.Value != 100 {
		t.Errorf("page down: value %v, want 100", sb.Value)
	}
	// Page down repeatedly — must never exceed 300.
	for i := 0; i < 10; i++ {
		sb.OnMouseDown(7, 290, 0)
		sb.OnMouseUp(7, 290, 0)
	}
	if sb.Value != 300 {
		t.Errorf("page to end: value %v, want 300", sb.Value)
	}
}
