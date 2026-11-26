package meui

import (
	"testing"
	"time"

	"github.com/sixnineone/meui/theme"
)

func TestEaseEndpointsAndMonotonic(t *testing.T) {
	kinds := []theme.Easing{
		theme.EaseStandard, theme.EaseDecelerate, theme.EaseAccelerate,
		theme.EaseLinear, theme.EaseSpring,
	}
	for _, k := range kinds {
		if got := Ease(k, 0); got != 0 {
			t.Errorf("Ease(%v,0) = %v, want 0", k, got)
		}
		if got := Ease(k, 1); got != 1 {
			t.Errorf("Ease(%v,1) = %v, want 1", k, got)
		}
		if k == theme.EaseSpring {
			// Springs overshoot above 1 before settling — asserted
			// separately below.
			continue
		}
		prev := float32(-1)
		for i := 0; i <= 20; i++ {
			v := Ease(k, float32(i)/20)
			if v < prev {
				t.Errorf("Ease(%v) not monotonic at t=%.2f: %v < %v", k, float32(i)/20, v, prev)
			}
			prev = v
		}
	}
	if Ease(theme.EaseLinear, 0.25) != 0.25 {
		t.Errorf("linear midpoint wrong: %v", Ease(theme.EaseLinear, 0.25))
	}
	// The spring curve must actually overshoot.
	overshot := false
	for i := 0; i <= 20; i++ {
		if Ease(theme.EaseSpring, float32(i)/20) > 1 {
			overshot = true
			break
		}
	}
	if !overshot {
		t.Error("EaseSpring never overshoots")
	}
}

func TestAnimationSettlesAndRetargets(t *testing.T) {
	a := &Animation{}
	timing := theme.Timing{DurationMS: 1, Easing: theme.EaseLinear}

	a.Set(1, timing)
	if !a.Active() {
		t.Fatal("animation not active after Set with positive duration")
	}
	time.Sleep(5 * time.Millisecond)
	if got := a.Value(); got != 1 {
		t.Fatalf("Value after settle = %v, want 1", got)
	}
	if a.Active() {
		t.Fatal("animation still active after settling")
	}

	// Instant timing jumps directly.
	a.Set(0, theme.Timing{})
	if got := a.Value(); got != 0 {
		t.Fatalf("instant Set Value = %v, want 0", got)
	}
	if a.Active() {
		t.Fatal("instant animation reported active")
	}

	// Idempotent retarget: same target while active is a no-op.
	long := theme.Timing{DurationMS: 10000, Easing: theme.EaseLinear}
	a.Set(1, long)
	a.Set(1, long)
	time.Sleep(2 * time.Millisecond)
	if v := a.Value(); v <= 0 || v >= 1 {
		t.Fatalf("mid-flight value %v out of (0,1)", v)
	}

	// Retarget mid-flight continues from the current value (no jump).
	before := a.Value()
	a.Set(0, long)
	if v := a.Value(); v < before-0.05 {
		t.Fatalf("retarget jumped: %v -> %v", before, v)
	}
}

func TestColorLerp(t *testing.T) {
	a := theme.DefaultTheme.Accent
	b := theme.DefaultTheme.Background
	if got := ColorLerp(a, b, 0); got != a {
		t.Errorf("t=0 = %08X, want %08X", got, a)
	}
	if got := ColorLerp(a, b, 1); got != b {
		t.Errorf("t=1 = %08X, want %08X", got, b)
	}
	mid := ColorLerp(0xFF000000, 0xFFFFFFFF, 0.5)
	if r := (mid >> 16) & 0xFF; r < 127 || r > 129 {
		t.Errorf("mid red = %d, want ~128", r)
	}
}
