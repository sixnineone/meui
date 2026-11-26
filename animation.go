package meui

import (
	"time"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

// Animation interpolates a float32 toward a target using the theme's
// motion tokens (duration + easing intent). Evaluation is lazy at Paint
// time — the render loop is continuous, so a running animation simply
// re-renders each frame until it settles; no timers or tick plumbing.
//
// Zero value is usable: Value() returns 0 until the first Set.
type Animation struct {
	from, to float32
	start    time.Time
	dur      time.Duration
	easing   theme.Easing
	active   bool
}

// Set starts a transition toward target using timing. Idempotent:
// retargeting to the in-flight target is a no-op; retargeting
// mid-flight continues from the CURRENT value (no jump).
func (a *Animation) Set(target float32, timing theme.Timing) {
	if a.active && target == a.to {
		return
	}
	cur := a.Value()
	a.from = cur
	a.to = target
	a.dur = time.Duration(timing.DurationMS * float32(time.Millisecond))
	a.easing = timing.Easing
	a.start = time.Now()
	a.active = a.dur > 0 && cur != target
	if !a.active {
		a.from = target
	}
}

// Value returns the current interpolated value.
func (a *Animation) Value() float32 {
	if !a.active {
		return a.to
	}
	el := time.Since(a.start)
	if el >= a.dur {
		a.active = false
		a.from = a.to
		return a.to
	}
	t := float32(el) / float32(a.dur)
	return a.from + (a.to-a.from)*Ease(a.easing, t)
}

// Snap places the animation at v with no transition. It is what a test (or an app that
// honours "reduce motion") uses to skip the easing.
func (a *Animation) Snap(v float32) {
	a.from, a.to = v, v
	a.active = false
}

// Active reports whether the transition is still running.
func (a *Animation) Active() bool { return a.active }

// Ease maps normalized time to eased progress. Curves are the concrete
// forms of the theme's easing intents.
func Ease(k theme.Easing, t float32) float32 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	switch k {
	case theme.EaseDecelerate: // ease-out cubic
		u := 1 - t
		return 1 - u*u*u
	case theme.EaseAccelerate: // ease-in cubic
		return t * t * t
	case theme.EaseLinear:
		return t
	case theme.EaseSpring: // ease-out back (slight overshoot)
		const c1 = 1.70158
		const c3 = c1 + 1
		u := t - 1
		return 1 + c3*u*u*u + c1*u*u
	default: // EaseStandard: ease-in-out cubic
		if t < 0.5 {
			return 4 * t * t * t
		}
		u := -2*t + 2
		return 1 - u*u*u/2
	}
}

// ColorLerp interpolates per channel in straight (non-premultiplied)
// space; alpha included. t is clamped to [0,1].
func ColorLerp(a, b graphics.Color, t float32) graphics.Color {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	lerp := func(x, y uint32) uint32 {
		return x + uint32((float32(int32(y)-int32(x)))*t+0.5)
	}
	ar, ag, ab, aa := uint32((a>>16)&0xFF), uint32((a>>8)&0xFF), uint32(a&0xFF), uint32((a>>24)&0xFF)
	br, bg, bb, ba := uint32((b>>16)&0xFF), uint32((b>>8)&0xFF), uint32(b&0xFF), uint32((b>>24)&0xFF)
	return graphics.Color(lerp(aa, ba)<<24 | lerp(ar, br)<<16 | lerp(ag, bg)<<8 | lerp(ab, bb))
}
