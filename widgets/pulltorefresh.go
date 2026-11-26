package widgets

import (
	"time"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

// PullToRefresh wraps a scrollable surface and refreshes it when the user drags down from the top —
// the gesture every phone list has, and the one that makes "there may be newer data" something the
// user can ask for rather than something the app guesses at with a timer.
//
// It is a CONTAINER THAT IS ALWAYS THE HIT (the `ScrollPane` and `SwipeActionRow` pattern), because
// the gesture has to be recognized before the content sees it: only the wrapper knows whether the
// content is already at the top, which is the whole difference between "scroll up" and "pull to
// refresh". A gesture it decides against is FORWARDED to the content, in the content's coordinates.
//
// It does not refresh. `OnRefresh` reports the gesture; the app does the work and calls `Done()` when
// it is finished, which is what keeps the indicator up for exactly as long as the request takes.
type PullToRefresh struct {
	meui.BaseContainer
	*theme.PopupTheme

	content   meui.Widget
	threshold float32

	dragging bool
	decided  bool
	pulling  bool
	pull     float32
	pressX   float32
	pressY   float32

	refreshing bool
	spin       float32

	// OnRefresh fires once per gesture, when the pull is released past the threshold.
	OnRefresh func()
}

// PullToRefresh metrics.
const (
	// PullThreshold is how far the content must be pulled before releasing refreshes.
	PullThreshold = float32(64)
	// PullMax is how far the content follows the finger. Past it the pull stops moving, so the
	// gesture has a visible end instead of dragging the content off the screen.
	PullMax = float32(112)
	// pullSlop is how far the pointer moves before the wrapper decides what the gesture is.
	pullSlop = float32(8)
	// pullIndicator is the refresh indicator's diameter.
	pullIndicator = float32(24)
)

// NewPullToRefresh builds an empty wrapper.
func NewPullToRefresh() *PullToRefresh {
	return &PullToRefresh{
		BaseContainer: meui.BaseContainer{BaseWidget: *meui.NewBaseWidget()},
		PopupTheme:    theme.DefaultPopupTheme,
	}
}

// ── content ─────────────────────────────────────────────────────────

func (p *PullToRefresh) SetContent(w meui.Widget) *PullToRefresh {
	p.content = w
	if w != nil {
		w.SetParent(p)
	}
	p.MarkForRedraw()
	return p
}

func (p *PullToRefresh) Content() meui.Widget { return p.content }

// AddChild makes a QML child the CONTENT, as ScrollPane and SwipeActionRow do: the loader adds
// children through AddChild, so a container that keeps its content elsewhere has to say so.
func (p *PullToRefresh) AddChild(child meui.Widget) { p.SetContent(child) }

// SetThreshold overrides how far the pull has to travel; zero uses PullThreshold.
func (p *PullToRefresh) SetThreshold(d float32) *PullToRefresh {
	p.threshold = d
	return p
}

// ── state ───────────────────────────────────────────────────────────

// Pull is the current pull distance; 0 when the content is at rest.
func (p *PullToRefresh) Pull() float32 { return p.pull }

// Refreshing reports whether the indicator is up waiting for Done.
func (p *PullToRefresh) Refreshing() bool { return p.refreshing }

// Done ends a refresh and lets the content settle back. The app calls it when its work is finished.
func (p *PullToRefresh) Done() {
	if !p.refreshing {
		return
	}
	p.refreshing = false
	p.pull = 0
	p.MarkForRedraw()
}

// pullThreshold is the effective threshold.
func (p *PullToRefresh) pullThreshold() float32 {
	if p.threshold > 0 {
		return p.threshold
	}
	return PullThreshold
}

// atTop reports whether the content is scrolled to its top, which is what makes a downward drag a
// PULL rather than a scroll. A content that cannot say is treated as always at the top: the gesture
// is the wrapper's reason to exist, and refusing it for want of an offset would be worse.
func (p *PullToRefresh) atTop() bool {
	if sc, ok := p.content.(interface{ ScrollOffsets() (float32, float32) }); ok {
		_, y := sc.ScrollOffsets()
		return y <= 0
	}
	return true
}

// ── geometry ────────────────────────────────────────────────────────

// ContentRect is the content's rect at rest. The paint translates it by the pull, so a gesture does
// not disturb the layout.
func (p *PullToRefresh) ContentRect(bounds graphics.Rect) graphics.Rect {
	return graphics.NewRect(bounds.X, bounds.Y, bounds.Width, bounds.Height)
}

// IndicatorRect is the refresh indicator's square, centred above the content by half the pull.
func (p *PullToRefresh) IndicatorRect(bounds graphics.Rect) graphics.Rect {
	size := pullIndicator
	if size > bounds.Width {
		size = bounds.Width
	}
	y := bounds.Y + p.pull/2 - size/2
	if y < bounds.Y {
		y = bounds.Y
	}
	return graphics.NewRect(bounds.X+(bounds.Width-size)/2, y, size, size)
}

func (p *PullToRefresh) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	if p.content != nil {
		if m, ok := p.content.(interface {
			PreferredDimensions(interface{}, float32, float32) (float32, float32)
		}); ok {
			return m.PreferredDimensions(p.content, w, h)
		}
	}
	return w, h
}

// PanelAt returns the WRAPPER itself, always: the gesture has to be recognized before the content
// sees it, and only the wrapper knows whether the content is at the top.
func (p *PullToRefresh) PanelAt(x, y float32) meui.Widget { return p }

func (p *PullToRefresh) ValidateLayout() {
	frame := p.FrameRect()
	if frame.Width <= 0 || frame.Height <= 0 {
		p.ClearNeedsLayout()
		return
	}
	if p.content != nil {
		p.content.SetFrameRect(graphics.NewRect(0, 0, frame.Width, frame.Height))
	}
	p.ClearNeedsLayout()
}

// ── interaction ─────────────────────────────────────────────────────

func (p *PullToRefresh) OnMouseDown(x, y float32, button int) bool {
	if !p.Enabled() || button != mouseButtonLeft {
		return false
	}
	if p.refreshing {
		// While the indicator is up the gesture is over; a press is the content's.
		p.forwardDown(x, y, button)
		return true
	}
	p.dragging = true
	p.decided = false
	p.pulling = false
	p.pressX, p.pressY = x, y
	return true
}

func (p *PullToRefresh) OnMouseMove(x, y float32) {
	if !p.dragging {
		p.forwardMove(x, y)
		return
	}
	dy := y - p.pressY
	if !p.decided {
		if dy < pullSlop && dy > -pullSlop {
			return // too small to be either gesture yet
		}
		// Decide ONCE: a downward drag with the content already at its top is a pull; anything
		// else belongs to the content, and the press it never got is replayed to it here.
		if dy > 0 && p.atTop() {
			p.decided, p.pulling = true, true
		} else {
			p.decided, p.pulling = true, false
			p.forwardDown(p.pressX, p.pressY, 0)
		}
	}
	if !p.pulling {
		p.forwardMove(x, y)
		return
	}
	// Resistance: the pull follows the finger at half speed and stops at PullMax, so the gesture has
	// a visible end and the content never leaves the screen.
	pull := dy * 0.5
	if pull > PullMax {
		pull = PullMax
	}
	if pull < 0 {
		pull = 0
	}
	if pull != p.pull {
		p.pull = pull
		p.MarkForRedraw()
	}
}

func (p *PullToRefresh) OnMouseUp(x, y float32, button int) bool {
	if button != mouseButtonLeft {
		return false
	}
	if !p.dragging {
		return false
	}
	p.dragging = false
	if !p.decided {
		// A tap: the content's own click has to work through the wrapper.
		p.forwardDown(p.pressX, p.pressY, button)
		p.forwardUp(x, y, button)
		return true
	}
	if !p.pulling {
		p.forwardUp(x, y, button)
		return true
	}
	p.pulling = false
	if p.pull >= p.pullThreshold() {
		// Hold the content at the threshold while the work happens, so the indicator has somewhere
		// to be and the content does not snap back under it.
		p.refreshing = true
		p.pull = p.pullThreshold()
		p.startSpin()
		p.MarkForRedraw()
		if p.OnRefresh != nil {
			p.OnRefresh()
		}
		return true
	}
	p.pull = 0
	p.MarkForRedraw()
	return true
}

// HoverAt implements HoverTarget.
func (p *PullToRefresh) HoverAt(x, y float32) { p.forwardMove(x, y) }

// startSpin advances the indicator while a refresh is running. It is a platform task, so it runs on
// the UI thread like every other callback and stops on its own when Done arrives.
func (p *PullToRefresh) startSpin() {
	// Declared, then assigned: the closure reschedules ITSELF, and a `:=` cannot refer to the
	// variable it is declaring.
	var step func()
	step = func() {
		if !p.refreshing {
			return
		}
		p.spin += 0.25
		if p.spin > 1 {
			p.spin -= 1
		}
		p.MarkForRedraw()
		platform.InvokeTaskAfter(step, 50*time.Millisecond)
	}
	platform.InvokeTaskAfter(step, 50*time.Millisecond)
}

// forwardDown/Move/Up pass a point to the content in ITS coordinates: the content is drawn offset by
// the pull, so its own hit test has to be shifted by the same amount.
func (p *PullToRefresh) forwardDown(x, y float32, button int) {
	if p.content == nil {
		return
	}
	if el, ok := p.content.(interface {
		OnMouseDown(float32, float32, int) bool
	}); ok {
		el.OnMouseDown(x, y-p.pull, button)
	}
}

func (p *PullToRefresh) forwardMove(x, y float32) {
	if p.content == nil {
		return
	}
	if el, ok := p.content.(interface{ OnMouseMove(float32, float32) }); ok {
		el.OnMouseMove(x, y-p.pull)
	}
}

func (p *PullToRefresh) forwardUp(x, y float32, button int) {
	if p.content == nil {
		return
	}
	if el, ok := p.content.(interface {
		OnMouseUp(float32, float32, int) bool
	}); ok {
		el.OnMouseUp(x, y-p.pull, button)
	}
}

// ── paint ───────────────────────────────────────────────────────────

func (p *PullToRefresh) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !p.Visible() || rect.Width < 1 || rect.Height < 1 {
		return
	}
	if p.NeedsLayout() {
		p.ValidateLayout()
	}
	canvas.Save()
	canvas.Translate(rect.X, rect.Y)
	bounds := graphics.NewRect(0, 0, rect.Width, rect.Height)

	// The indicator, in the space the pull uncovers.
	if p.pull > 0 {
		p.paintIndicator(canvas, p.IndicatorRect(bounds))
	}

	if p.content != nil {
		plate := graphics.NewPaint()
		plateInk := p.BackgroundInk
		if plateInk == 0 {
			plateInk = theme.DefaultTheme.Background
		}
		plate.SetColor(plateInk)
		cr := p.ContentRect(bounds)
		canvas.Save()
		canvas.Translate(0, p.pull)
		canvas.DrawRect(cr, plate)
		p.content.SetFrameRect(cr)
		p.content.Paint(canvas, cr)
		canvas.Restore()
	}
	canvas.Restore()
}

// paintIndicator draws an arc whose sweep grows with the pull, and which spins while refreshing.
func (p *PullToRefresh) paintIndicator(canvas *graphics.Canvas, r graphics.Rect) {
	ink := theme.DefaultTheme.Accent
	if ink == 0 {
		ink = p.HeadingTextInk
	}
	paint := graphics.NewPaint()
	paint.SetColor(ink)
	// STROKE, not fill: an arc that is filled is a pie slice, and a refresh indicator is a ring.
	paint.SetStyle(paintstyle.Stroke)
	paint.SetStrokeWidth(2)

	// The sweep: nothing at rest, a full circle at the threshold, so the gesture shows how far it
	// has to go before it will do anything.
	//
	// DEGREES, not radians: the seam's DrawArc converts with `* pi / 180`, so a caller that passes
	// radians gets an arc of a few degrees — sub-pixel, and invisible rather than wrong-looking.
	frac := p.pull / p.pullThreshold()
	if frac > 1 {
		frac = 1
	}
	start := p.spin * 360
	sweep := 90 + frac*270
	if !p.refreshing {
		start = 0
		sweep = frac * 360
	}
	if sweep < 5 {
		sweep = 5
	}
	canvas.Save()
	canvas.ClipRect(r, pathop.Intersect, false)
	canvas.DrawArc(r, start, sweep, paint)
	canvas.Restore()
}

var _ meui.EventListener = (*PullToRefresh)(nil)
var _ meui.HoverTarget = (*PullToRefresh)(nil)
