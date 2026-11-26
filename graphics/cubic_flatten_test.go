package graphics

import "testing"

// A cubic has to be flattened to fit its own length: a fixed step count is
// invisible on a short curve and a visible staircase on a long one, and the
// renderer strokes every straight segment as its own quad, so the steps show at
// any stroke width. These are the numbers behind that rule.
func TestCubicStepsFollowTheCurveLength(t *testing.T) {
	if got := cubicSteps(Point{}, Point{X: 10}, Point{X: 20}, Point{X: 30}); got != 8 {
		t.Errorf("a 30px cubic flattens to %d steps, want the 8-step floor", got)
	}
	short := cubicSteps(Point{}, Point{X: 40}, Point{X: 80}, Point{X: 120})
	if short < 8 || short > 40 {
		t.Errorf("a 120px cubic flattens to %d steps, want a handful", short)
	}
	long := cubicSteps(Point{}, Point{X: 400}, Point{X: 800}, Point{X: 1200})
	if long < 100 {
		t.Errorf("a 1200px cubic flattens to %d steps, want at least one per 6px", long)
	}
	if max := cubicSteps(Point{}, Point{X: 1e6}, Point{X: 2e6}, Point{X: 3e6}); max != 256 {
		t.Errorf("an enormous cubic flattens to %d steps, want the 256 cap", max)
	}

	// The segment length the step count implies stays within the tolerance for
	// any curve whose length the control polygon measures.
	for _, poly := range []float32{30, 120, 600, 1200, 3000} {
		steps := cubicSteps(Point{}, Point{X: poly / 3}, Point{X: 2 * poly / 3}, Point{X: poly})
		if per := poly / float32(steps); per > cubicSegmentMax && steps != 8 && steps != 256 {
			t.Errorf("a %vpx curve leaves %vpx segments at %d steps", poly, per, steps)
		}
	}
}
