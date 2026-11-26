package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

func newTestSearchView() *SearchView {
	s := NewSearchView()
	s.SetPlaceholder("Search")
	s.SetResults([]SearchViewResult{
		{ID: "a", Label: "Alpha", Secondary: "first"},
		{ID: "b", Label: "Beta"},
		{ID: "c", Label: "Gamma", Secondary: "third"},
	})
	s.SetFrameRect(graphics.NewRect(0, 0, 400, 700))
	return s
}

// TestSearchViewControlsReachTheTouchFloor is the Phase A contract on the two controls a thumb
// finds first: the way out and the way to clear.
func TestSearchViewControlsReachTheTouchFloor(t *testing.T) {
	saved := theme.DefaultTheme
	t.Cleanup(func() { theme.SetDefaultTheme(saved) })
	th := *saved
	theme.SetDefaultTheme(&th)
	theme.DefaultTheme.Metrics.MinTouchTarget = theme.TouchTargetAndroid

	s := newTestSearchView()
	bounds := s.FrameRect()
	if r := s.BackRect(bounds); r.Width < theme.TouchTargetAndroid || r.Height < theme.TouchTargetAndroid {
		t.Errorf("the back control is %.0fx%.0f, under the %v floor", r.Width, r.Height, theme.TouchTargetAndroid)
	}
	s.SetQuery("a")
	r, ok := s.ClearRect(bounds)
	if !ok {
		t.Fatal("no clear control with a query")
	}
	if r.Width < theme.TouchTargetAndroid || r.Height < theme.TouchTargetAndroid {
		t.Errorf("the clear control is %.0fx%.0f, under the %v floor", r.Width, r.Height, theme.TouchTargetAndroid)
	}
}

// TestSearchViewFieldSitsBetweenTheControls keeps the three parts from overlapping: a field that
// runs under a control swallows the taps meant for it.
func TestSearchViewFieldSitsBetweenTheControls(t *testing.T) {
	s := newTestSearchView()
	bounds := s.FrameRect()

	back := s.BackRect(bounds)
	f := s.FieldRect(bounds)
	if f.X < back.X+back.Width {
		t.Errorf("the field starts at %.0f, under the back control which ends at %.0f", f.X, back.X+back.Width)
	}
	// With no query there is no clear control, so the field reaches the header's right edge.
	if f.X+f.Width != bounds.Width {
		t.Errorf("with no query the field ends at %.0f, want the header's edge %.0f", f.X+f.Width, bounds.Width)
	}
	// With one, the field stops before it.
	s.SetQuery("a")
	clear, ok := s.ClearRect(bounds)
	if !ok {
		t.Fatal("no clear control")
	}
	f = s.FieldRect(bounds)
	if f.X+f.Width > clear.X {
		t.Errorf("the field ends at %.0f, over the clear control which starts at %.0f", f.X+f.Width, clear.X)
	}
}

// TestSearchViewResultsSitBelowTheHeaderAndAreOrdered pins the list's geometry.
func TestSearchViewResultsSitBelowTheHeaderAndAreOrdered(t *testing.T) {
	s := newTestSearchView()
	bounds := s.FrameRect()
	h := s.HeaderRect(bounds)
	prev := h.Y + h.Height
	for i := range s.results {
		r, ok := s.ResultRect(i, bounds)
		if !ok {
			t.Fatalf("no rect for result %d", i)
		}
		if r.Y < prev {
			t.Errorf("result %d starts at %.0f, above %.0f", i, r.Y, prev)
		}
		if r.Height < theme.TouchTargetAndroid && theme.DefaultTheme.Metrics.MinTouchTarget > 0 {
			t.Errorf("result %d is %.0f tall, under the touch floor", i, r.Height)
		}
		prev = r.Y + r.Height
	}
	if got := s.ResultAt(bounds.Width/2, h.Y+h.Height+SearchViewRowH/2); got != 0 {
		t.Errorf("the first row reports %d, want 0", got)
	}
	if got := s.ResultAt(bounds.Width/2, h.Y+h.Height+SearchViewRowH+SearchViewRowH/2); got != 1 {
		t.Errorf("the second row reports %d, want 1", got)
	}
}

// TestSearchViewChoosesOnReleaseOfTheSameRow is the gesture contract shared with the other lists.
func TestSearchViewChoosesOnReleaseOfTheSameRow(t *testing.T) {
	s := newTestSearchView()
	var chosen string
	s.OnChoose = func(id string) { chosen = id }

	r, _ := s.ResultRect(1, s.FrameRect())
	cx, cy := r.X+r.Width/2, r.Y+r.Height/2
	if !s.OnMouseDown(cx, cy, 0) {
		t.Fatal("a press on a result was not taken")
	}
	s.OnMouseUp(cx, cy, 0)
	if chosen != "b" {
		t.Errorf("chose %q, want b", chosen)
	}

	// A release on a DIFFERENT row chooses nothing.
	chosen = ""
	r0, _ := s.ResultRect(0, s.FrameRect())
	r2, _ := s.ResultRect(2, s.FrameRect())
	s.OnMouseDown(r0.X+r0.Width/2, r0.Y+r0.Height/2, 0)
	s.OnMouseUp(r2.X+r2.Width/2, r2.Y+r2.Height/2, 0)
	if chosen != "" {
		t.Errorf("a release on another row chose %q", chosen)
	}
}

// TestSearchViewBackClosesAndClearClears pins the two drawn controls.
func TestSearchViewBackClosesAndClearClears(t *testing.T) {
	s := newTestSearchView()
	closed := false
	s.OnClose = func() { closed = true }

	b := s.BackRect(s.FrameRect())
	if !s.OnMouseDown(b.X+b.Width/2, b.Y+b.Height/2, 0) {
		t.Fatal("a press on the back control was not taken")
	}
	s.OnMouseUp(b.X+b.Width/2, b.Y+b.Height/2, 0)
	if !closed {
		t.Error("the back control did not report a close")
	}

	s2 := newTestSearchView()
	s2.SetQuery("typed")
	cleared := ""
	s2.OnQueryChanged = func(q string) { cleared = q }
	c, ok := s2.ClearRect(s2.FrameRect())
	if !ok {
		t.Fatal("no clear control")
	}
	s2.OnMouseDown(c.X+c.Width/2, c.Y+c.Height/2, 0)
	s2.OnMouseUp(c.X+c.Width/2, c.Y+c.Height/2, 0)
	if s2.Query() != "" {
		t.Errorf("the clear control left the query as %q", s2.Query())
	}
	if cleared != "" {
		t.Errorf("clearing reported the query as %q, want empty", cleared)
	}
}

// TestSearchViewReportsTyping is why the field is a real TextField: typing goes through the
// toolkit's one text path and comes back out as OnQueryChanged.
func TestSearchViewReportsTyping(t *testing.T) {
	s := newTestSearchView()
	var queries []string
	s.OnQueryChanged = func(q string) { queries = append(queries, q) }

	// The field self-gates on focus (so a stray rune cannot reach an unfocused box), and a search
	// view focuses it on arrival — see SetWindow.
	s.Field().OnFocus()
	s.Field().OnCharacter('a')
	s.Field().OnCharacter('b')
	if len(queries) == 0 {
		t.Fatal("typing did not report a query change")
	}
	if got := queries[len(queries)-1]; got != "ab" {
		t.Errorf("the last query was %q, want ab", got)
	}
	if s.Query() != "ab" {
		t.Errorf("the field holds %q, want ab", s.Query())
	}
}

// TestSearchViewFillsItsArea is the full-screen contract: the view is sized by the surface, not by
// its content, so an app shows it over the whole window.
func TestSearchViewFillsItsArea(t *testing.T) {
	s := newTestSearchView()
	if w, h := s.PreferredDimensions(nil, 400, 700); w != 400 || h != 700 {
		t.Errorf("PreferredDimensions = %vx%v, want the area it was given (400x700)", w, h)
	}
	if !meui.IsInteractive(s) {
		t.Error("a SearchView handles input but is not in the interactive registry")
	}
}

var _ meui.EventListener = (*SearchView)(nil)
var _ meui.HoverTarget = (*SearchView)(nil)
