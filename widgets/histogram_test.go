package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
)

func bins(vals ...uint32) []uint32 { return vals }

// TestHistogramPeakIgnoresTheClippingSpikes is the decision inside the widget: the first and last bins
// collect the fully black and fully blown-out pixels, and an image with a solid black border puts a
// spike there an order of magnitude taller than everything else. Normalising to that spike draws every
// other bin as a flat line at the bottom — useless exactly when a user is looking for clipping.
func TestHistogramPeakIgnoresTheClippingSpikes(t *testing.T) {
	// Sixteen bins: bin 0 holds a huge spike, the midtones hold a modest distribution.
	r := make([]uint32, 16)
	r[0] = 100000 // the black border
	for i := 5; i <= 10; i++ {
		r[i] = 500
	}
	g := append([]uint32(nil), r...)
	b := append([]uint32(nil), r...)
	l := append([]uint32(nil), r...)

	h := NewHistogram().SetBins(r, g, b, l)
	if got := h.peak(); got != 500 {
		t.Errorf("the plot normalises to %d, want 500 — the clipping spike in bin 0 must not flatten "+
			"the midtones", got)
	}
	// Guard off: the spike IS the peak, which is the behaviour the guard exists to prevent.
	if got := h.SetClipGuard(false).peak(); got != 100000 {
		t.Errorf("with the guard off the peak is %d, want the spike", got)
	}
}

// TestHistogramPeakFallsBackWhenThereIsNothingElse keeps the guard from producing a zero peak: an image
// that really is all black has its spike in bin 0 and nothing else, and it must still draw something.
func TestHistogramPeakFallsBackWhenThereIsNothingElse(t *testing.T) {
	r := make([]uint32, 8)
	r[0] = 4000
	h := NewHistogram().SetBins(r, append([]uint32(nil), r...), append([]uint32(nil), r...), append([]uint32(nil), r...))
	if got := h.peak(); got != 4000 {
		t.Errorf("an all-black histogram normalises to %d, want its own spike so it draws", got)
	}
}

// TestHistogramRefusesMismatchedBins: a plot built from arrays of different lengths is a lie about the
// data, not a cosmetic problem, so the widget keeps what it had rather than drawing something wrong.
func TestHistogramRefusesMismatchedBins(t *testing.T) {
	good := bins(1, 2, 3, 4)
	short := bins(1, 2)
	h := NewHistogram().SetBins(good, good, good, good)
	if h.Bins() != 4 {
		t.Fatalf("bins = %d, want 4", h.Bins())
	}
	// A DIFFERENT length in one channel is a mismatch: refused, so the previous bins survive.
	h.SetBins(good, short, good, good)
	if h.Bins() != 4 {
		t.Errorf("a single mismatched channel replaced the good bins; bins = %d", h.Bins())
	}
	// All four the same (shorter) length is NOT a mismatch — it is a legitimately coarser histogram,
	// and it must be accepted. My first version of this test asserted otherwise, which was the test
	// being wrong about its own rule.
	h.SetBins(short, short, short, short)
	if h.Bins() != 2 {
		t.Errorf("a consistent two-bin histogram was refused; bins = %d", h.Bins())
	}
}

// TestHistogramEmptyIsInert: a document with no pixels must not divide by zero or panic.
func TestHistogramEmptyIsInert(t *testing.T) {
	h := NewHistogram()
	if h.Bins() != 0 || h.peak() != 0 {
		t.Errorf("an empty histogram reports %d bins and peak %d", h.Bins(), h.peak())
	}
}

// TestHistogramIsAWidgetNotAnInputTarget keeps it out of the interactive registry: it is a readout.
func TestHistogramIsAWidgetNotAnInputTarget(t *testing.T) {
	if meui.IsInteractive(NewHistogram()) {
		t.Error("the Histogram is registered as an input target; it owns no input")
	}
	var _ meui.Widget = NewHistogram()
	_ = graphics.Rect{}
}
