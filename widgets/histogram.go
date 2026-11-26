package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/theme"
)

// Histogram draws a document's tonal distribution: the four bin arrays an image editor's histogram is
// made of, as the classic filled mountains.
//
// It takes BINS, not a document, for the same reason Waveform takes peaks: the toolkit must not depend
// on meui-media (the media module is built on the toolkit), so the app computes the bins with
// `document.ComputeHistogram` and passes the four slices.
type Histogram struct {
	meui.BaseWidget
	*theme.LabelTheme

	r, g, b, l []uint32
	clipped    bool
}

// NewHistogram builds an empty histogram.
func NewHistogram() *Histogram {
	return &Histogram{
		BaseWidget: *meui.NewBaseWidget(),
		LabelTheme: theme.DefaultLabelTheme.Copy(),
		clipped:    true,
	}
}

// SetBins installs the four channels. All four must be the same length; a mismatch is refused rather
// than drawn, because a plot from mismatched arrays is a lie about the data rather than a cosmetic
// problem.
func (h *Histogram) SetBins(r, g, b, l []uint32) *Histogram {
	if len(r) != len(g) || len(r) != len(b) || len(r) != len(l) {
		return h
	}
	h.r, h.g, h.b, h.l = r, g, b, l
	h.MarkForRedraw()
	return h
}

// Bins is how many bins per channel.
func (h *Histogram) Bins() int { return len(h.r) }

// SetClipGuard turns the extremes guard on or off (it is on by default; see the note in peak).
func (h *Histogram) SetClipGuard(on bool) *Histogram {
	h.clipped = on
	h.MarkForRedraw()
	return h
}

// peak is the value the plot normalises to.
//
// It deliberately IGNORES THE FIRST AND LAST BINS unless they are all there is. Those two bins collect
// the pixels that are fully black or fully blown out, and a photograph with a large black background —
// or any image with a solid border — puts a spike there that is an order of magnitude taller than
// everything else. Normalising to that spike draws every other bin as a flat line at the bottom, which
// is the classic way a histogram becomes useless exactly when a user is looking for clipping.
func (h *Histogram) peak() uint32 {
	n := len(h.r)
	if n == 0 {
		return 0
	}
	// The guard is what the widget does by DEFAULT; SetClipGuard(false) is for a caller that wants the
	// raw distribution, and the first version of this function ignored the flag entirely — the setter
	// existed and did nothing, which the test caught.
	if !h.clipped || n <= 2 {
		return maxOfBins(h.r, h.g, h.b, h.l)
	}
	var m uint32
	for i := 1; i < n-1; i++ {
		if h.r[i] > m {
			m = h.r[i]
		}
		if h.g[i] > m {
			m = h.g[i]
		}
		if h.b[i] > m {
			m = h.b[i]
		}
		if h.l[i] > m {
			m = h.l[i]
		}
	}
	if m == 0 {
		return maxOfBins(h.r, h.g, h.b, h.l)
	}
	return m
}

func maxOfBins(chans ...[]uint32) uint32 {
	var m uint32
	for _, c := range chans {
		for _, v := range c {
			if v > m {
				m = v
			}
		}
	}
	return m
}

func (h *Histogram) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !h.Visible() || rect.Width < 1 || rect.Height < 1 {
		return
	}
	canvas.Save()
	canvas.Translate(rect.X, rect.Y)
	bounds := graphics.NewRect(0, 0, rect.Width, rect.Height)

	plate := graphics.NewPaint()
	plate.SetColor(theme.DefaultTheme.Surface)
	canvas.DrawRect(bounds, plate)

	n := len(h.r)
	if n == 0 {
		canvas.Restore()
		return
	}
	canvas.ClipRect(bounds, pathop.Intersect, false)

	peak := h.peak()
	if peak == 0 {
		canvas.Restore()
		return
	}

	// The grid: quarters, so a user can judge where the midtones are.
	grid := graphics.NewPaint()
	grid.SetColor(theme.DefaultTheme.BorderSubtle)
	for q := 1; q < 4; q++ {
		canvas.DrawRect(graphics.NewRect(bounds.Width*float32(q)/4, 0, 1, bounds.Height), grid)
	}

	// One column per PIXEL of width, taking the tallest bin it covers — the same reduction the Waveform
	// does, and for the same reason: more bins than pixels must not become more draws than pixels.
	cols := int(bounds.Width)
	for c := 0; c < cols; c++ {
		first := c * n / cols
		last := (c + 1) * n / cols
		if last <= first {
			last = first + 1
		}
		if last > n {
			last = n
		}
		vr, vg, vb := uint32(0), uint32(0), uint32(0)
		for i := first; i < last; i++ {
			if h.r[i] > vr {
				vr = h.r[i]
			}
			if h.g[i] > vg {
				vg = h.g[i]
			}
			if h.b[i] > vb {
				vb = h.b[i]
			}
		}
		// The extremes are clamped to the plot height rather than allowed to run past it: with the clip
		// guard they are usually the tallest thing here.
		hr := float32(vr) / float32(peak) * bounds.Height
		hg := float32(vg) / float32(peak) * bounds.Height
		hb := float32(vb) / float32(peak) * bounds.Height
		if hr > bounds.Height {
			hr = bounds.Height
		}
		if hg > bounds.Height {
			hg = bounds.Height
		}
		if hb > bounds.Height {
			hb = bounds.Height
		}
		// Each channel is drawn ADDITIVELY: the renderer's blend makes an overlap of all three read as
		// grey, which is what a photo histogram looks like and what makes a colour cast visible at a
		// glance.
		for _, ch := range []struct {
			v float32
			c uint32
		}{{hr, 0xFF0000}, {hg, 0x00FF00}, {hb, 0x0000FF}} {
			if ch.v <= 0 {
				continue
			}
			p := graphics.NewPaint()
			// A translucent channel colour, so an overlap of all three reads as grey — which is what
			// makes a colour cast visible at a glance rather than needing the numbers.
			p.SetColor(graphics.RGBA(uint8(ch.c>>16&0xFF), uint8(ch.c>>8&0xFF), uint8(ch.c&0xFF), 140))
			canvas.DrawRect(graphics.NewRect(float32(c), bounds.Height-ch.v, 1, ch.v), p)
		}
	}
	canvas.Restore()
}

func (h *Histogram) PreferredDimensions(target interface{}, width, height float32) (float32, float32) {
	if width <= 0 {
		width = 256
	}
	if height <= 0 {
		height = 96
	}
	return width, height
}

var _ meui.Widget = (*Histogram)(nil)
var _ = theme.DefaultTheme
