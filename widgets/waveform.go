package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/theme"
)

// Waveform draws an audio envelope from PEAKS — the min/max per bucket a peak cache computes — with a
// playhead, and reports a click as a seek.
//
// It takes plain slices rather than a media type on purpose: the toolkit must not depend on
// `meui-media` (the media module is built on the toolkit, not the other way round), so the app passes
// `peaks.Min, peaks.Max` and the widget knows nothing about audio beyond that.
//
// A waveform cannot draw from samples — a screen is a few hundred pixels wide and a track is millions
// of samples — which is why peaks exist at all (MEDIA-PLAN P5): the reduction happens once, and the
// widget draws a few hundred vertical lines from it.
type Waveform struct {
	meui.BaseWidget
	*theme.LabelTheme

	min, max []float32
	progress float32

	hovered bool
	pressed bool

	// OnSeek reports a click as a fraction of the track (0 at the start, 1 at the end).
	OnSeek func(frac float32)
}

// NewWaveform builds an empty waveform.
func NewWaveform() *Waveform {
	return &Waveform{
		BaseWidget: *meui.NewBaseWidget(),
		LabelTheme: theme.DefaultLabelTheme.Copy(),
	}
}

// SetPeaks installs the envelope. The slices are kept, not copied: a peak cache for a long track is
// megabytes, and copying it per frame is the cost this widget exists to avoid.
func (w *Waveform) SetPeaks(min, max []float32) *Waveform {
	w.min, w.max = min, max
	w.MarkForRedraw()
	return w
}

// Peaks is the number of buckets, so a caller can check that the envelope matches the width it set.
func (w *Waveform) Peaks() int { return len(w.min) }

// SetProgress moves the playhead, clamped to [0,1].
func (w *Waveform) SetProgress(frac float32) *Waveform {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	if frac != w.progress {
		w.progress = frac
		w.MarkForRedraw()
	}
	return w
}

// Progress is the playhead's position.
func (w *Waveform) Progress() float32 { return w.progress }

// BucketAt reports which peak bucket a local x maps to, or -1 when there is nothing to map. It is
// exported because the mapping is the widget's whole geometry contract: a test asserts it, and a
// caller drawing a hit marker uses the same function rather than reimplementing it.
func (w *Waveform) BucketAt(x float32) int {
	n := len(w.min)
	if n == 0 {
		return -1
	}
	fr := w.FrameRect()
	if fr.Width <= 0 {
		return -1
	}
	// The bucket is chosen from the COLUMN the point falls in, so the mapping is stable regardless
	// of how many buckets there are.
	b := int(x / fr.Width * float32(n))
	if b < 0 {
		b = 0
	}
	if b >= n {
		b = n - 1
	}
	return b
}

// FractionAt reports a local x as a fraction of the width, clamped — what a seek reports.
func (w *Waveform) FractionAt(x float32) float32 {
	fr := w.FrameRect()
	if fr.Width <= 0 {
		return 0
	}
	f := x / fr.Width
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

func (w *Waveform) OnMouseDown(x, y float32, button int) bool {
	if !w.Enabled() || button != mouseButtonLeft {
		return false
	}
	w.pressed = true
	w.MarkForRedraw()
	return true
}

func (w *Waveform) OnMouseUp(x, y float32, button int) bool {
	if button != mouseButtonLeft || !w.pressed {
		return false
	}
	w.pressed = false
	w.MarkForRedraw()
	// A seek commits on RELEASE and only while the pointer is still on the widget, so a mis-touch is
	// recoverable by sliding off — the same gesture contract the list rows use.
	if w.ContainsPoint(x, y) && w.OnSeek != nil {
		w.OnSeek(w.FractionAt(x))
	}
	return true
}

func (w *Waveform) OnMouseMove(x, y float32) {
	h := w.ContainsPoint(x, y)
	if h != w.hovered {
		w.hovered = h
		w.MarkForRedraw()
	}
}

// HoverAt implements HoverTarget.
func (w *Waveform) HoverAt(x, y float32) { w.OnMouseMove(x, y) }

// PreferredDimensions: a waveform is a strip, so its natural height is the theme's row height.
func (w *Waveform) PreferredDimensions(target interface{}, width, height float32) (float32, float32) {
	h := theme.DefaultTheme.Metrics.ListRowHeight
	if h <= 0 {
		h = 48
	}
	return width, h
}

func (w *Waveform) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !w.Visible() || rect.Width < 1 || rect.Height < 1 {
		return
	}
	canvas.Save()
	canvas.Translate(rect.X, rect.Y)
	bounds := graphics.NewRect(0, 0, rect.Width, rect.Height)

	// The track's plate, so an empty waveform still reads as a control rather than a gap.
	bg := graphics.NewPaint()
	bg.SetColor(theme.DefaultTheme.Surface)
	canvas.DrawRect(bounds, bg)

	n := len(w.min)
	if n == 0 {
		canvas.Restore()
		return
	}

	// The centre line, which is where silence is.
	mid := bounds.Height / 2
	centre := graphics.NewPaint()
	centre.SetColor(theme.DefaultTheme.BorderSubtle)
	canvas.DrawRect(graphics.NewRect(0, mid, bounds.Width, 1), centre)

	// One vertical line per COLUMN, not per bucket: a track with more buckets than pixels would
	// otherwise overdraw thousands of lines to paint a few hundred columns. Each column takes the
	// extremes of the buckets it covers, so a peak between two columns is never lost.
	ink := w.TextInk
	playedInk := theme.DefaultTheme.Accent
	if playedInk == 0 {
		playedInk = ink
	}
	playhead := w.progress * bounds.Width
	cols := int(bounds.Width)
	canvas.Save()
	canvas.ClipRect(bounds, pathop.Intersect, false)
	for c := 0; c < cols; c++ {
		lo, hi := w.extremesFor(c, cols, n)
		if lo == 0 && hi == 0 {
			continue
		}
		// The envelope maps [-1,1] to the strip; a slope of 1 at full scale.
		y0 := mid - hi*mid
		y1 := mid - lo*mid
		if y1-y0 < 1 {
			y1 = y0 + 1 // a silent or quiet bucket still shows a line, so quiet passages are visible
		}
		p := graphics.NewPaint()
		if float32(c) <= playhead {
			p.SetColor(playedInk)
		} else {
			p.SetColor(ink)
		}
		canvas.DrawRect(graphics.NewRect(float32(c), y0, 1, y1-y0), p)
	}
	canvas.Restore()

	// The playhead.
	if w.progress > 0 {
		ph := graphics.NewPaint()
		ph.SetColor(playedInk)
		canvas.DrawRect(graphics.NewRect(playhead-1, 0, 2, bounds.Height), ph)
	}

	// Hover and press tint the whole strip, so the control answers the pointer.
	if w.pressed {
		tint := graphics.NewPaint()
		tint.SetColor(theme.DefaultTheme.Pressed)
		canvas.DrawRect(bounds, tint)
	} else if w.hovered {
		tint := graphics.NewPaint()
		tint.SetColor(theme.DefaultTheme.Hovered)
		canvas.DrawRect(bounds, tint)
	}
	canvas.Restore()
}

// extremesFor returns the min and max a painted COLUMN shows, taken from every bucket it covers. With
// more buckets than columns that is the true envelope at that column's resolution; with fewer, the
// column repeats its bucket, which is what a stretched waveform should look like.
func (w *Waveform) extremesFor(col, cols, buckets int) (lo, hi float32) {
	first := col * buckets / cols
	last := (col + 1) * buckets / cols
	if last <= first {
		last = first + 1
	}
	if last > buckets {
		last = buckets
	}
	for b := first; b < last; b++ {
		if w.min[b] < lo {
			lo = w.min[b]
		}
		if w.max[b] > hi {
			hi = w.max[b]
		}
	}
	return lo, hi
}

var _ meui.EventListener = (*Waveform)(nil)
var _ meui.HoverTarget = (*Waveform)(nil)
