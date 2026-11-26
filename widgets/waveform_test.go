package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
)

func peaksOf(vals ...float32) []float32 { return vals }

// TestWaveformBucketMappingIsStable pins the geometry contract: a column maps to the bucket that
// covers it, for any ratio of buckets to pixels — the mapping a hit marker and the painter share.
func TestWaveformBucketMappingIsStable(t *testing.T) {
	w := NewWaveform()
	w.SetPeaks(peaksOf(-1, -0.5, 0, 0.5), peaksOf(1, 0.5, 0, 1))
	w.SetFrameRect(graphics.NewRect(0, 0, 400, 64))

	// Four buckets over 400 pixels: each bucket owns 100 pixels.
	for _, c := range []struct {
		x    float32
		want int
	}{
		{0, 0}, {99, 0}, {100, 1}, {199, 1}, {200, 2}, {300, 3}, {399, 3},
	} {
		if got := w.BucketAt(c.x); got != c.want {
			t.Errorf("x=%.0f maps to bucket %d, want %d", c.x, got, c.want)
		}
	}
	// Past the ends it clamps rather than reporting an impossible bucket.
	if got := w.BucketAt(9999); got != 3 {
		t.Errorf("x past the end maps to %d, want the last bucket", got)
	}
	if got := w.BucketAt(-5); got != 0 {
		t.Errorf("x before the start maps to %d, want the first bucket", got)
	}
}

// TestWaveformWithoutPeaksIsInert keeps an empty waveform from dividing by zero or reporting a seek
// into nothing.
func TestWaveformWithoutPeaksIsInert(t *testing.T) {
	w := NewWaveform()
	w.SetFrameRect(graphics.NewRect(0, 0, 200, 48))
	if got := w.BucketAt(10); got != -1 {
		t.Errorf("an empty waveform mapped x to bucket %d, want -1", got)
	}
	fired := false
	w.OnSeek = func(float32) { fired = true }
	w.OnMouseDown(10, 10, 0)
	w.OnMouseUp(10, 10, 0)
	if !fired {
		t.Error("a seek on an empty waveform did not report")
	}
}

// TestWaveformSeekReportsTheFraction is the interaction contract: a click reports where in the track
// it landed, and a release OFF the widget reports nothing — the same gesture rule as the list rows.
func TestWaveformSeekReportsTheFraction(t *testing.T) {
	w := NewWaveform()
	w.SetPeaks(peaksOf(0, 0), peaksOf(0, 0))
	w.SetFrameRect(graphics.NewRect(0, 0, 100, 48))

	var got float32 = -1
	w.OnSeek = func(f float32) { got = f }
	w.OnMouseDown(25, 10, 0)
	w.OnMouseUp(25, 10, 0)
	if got < 0.24 || got > 0.26 {
		t.Errorf("a click at 25/100 reported %.3f, want ~0.25", got)
	}

	got = -1
	w.OnMouseDown(25, 10, 0)
	w.OnMouseUp(25, 500, 0) // off the widget
	if got != -1 {
		t.Errorf("a release off the widget reported a seek of %.3f", got)
	}
}

// TestWaveformProgressClamps keeps a caller's out-of-range progress from drawing a playhead outside
// the strip.
func TestWaveformProgressClamps(t *testing.T) {
	w := NewWaveform()
	if got := w.SetProgress(2.5).Progress(); got != 1 {
		t.Errorf("progress 2.5 clamped to %v, want 1", got)
	}
	if got := w.SetProgress(-3).Progress(); got != 0 {
		t.Errorf("progress -3 clamped to %v, want 0", got)
	}
}

// TestWaveformColumnTakesTheExtremesOfItsBuckets is the reduction with more buckets than pixels: a
// loud transient inside one column's span must show, or a waveform hides exactly what a user is
// looking for.
func TestWaveformColumnTakesTheExtremesOfItsBuckets(t *testing.T) {
	w := NewWaveform()
	// 100 buckets over a 10-pixel strip: each column covers 10 buckets, and bucket 55 is the hit.
	mins := make([]float32, 100)
	maxs := make([]float32, 100)
	maxs[55] = 1
	w.SetPeaks(mins, maxs)
	w.SetFrameRect(graphics.NewRect(0, 0, 10, 64))
	lo, hi := w.extremesFor(5, 10, 100)
	if hi != 1 {
		t.Errorf("column 5 (buckets 50..59) reports max %v, want the transient at bucket 55 (1)", hi)
	}
	if lo != 0 {
		t.Errorf("column 5 reports min %v, want 0", lo)
	}
	// A neighbouring column does not inherit it.
	if _, hi2 := w.extremesFor(6, 10, 100); hi2 != 0 {
		t.Errorf("column 6 reports max %v, want 0 — the transient bled into its neighbour", hi2)
	}
}

// TestWaveformIsAnInputTarget keeps it in the registry the touch audit checks.
func TestWaveformIsAnInputTarget(t *testing.T) {
	if !meui.IsInteractive(NewWaveform()) {
		t.Error("a Waveform handles input but is not in the interactive registry")
	}
}

var _ meui.EventListener = (*Waveform)(nil)
var _ meui.HoverTarget = (*Waveform)(nil)
