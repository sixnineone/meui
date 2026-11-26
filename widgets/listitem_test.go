package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

// withListMetrics pins the row metric so the heights below are the published ones rather than
// whatever the ambient theme happens to carry.
func withListMetrics(t *testing.T) {
	t.Helper()
	saved := theme.DefaultTheme
	t.Cleanup(func() { theme.SetDefaultTheme(saved) })
	th := *saved
	theme.SetDefaultTheme(&th)
	// AFTER SetDefaultTheme: withStyleDefaults replaces Metrics wholesale when ControlHeight is
	// zero, which would discard anything set before.
	theme.DefaultTheme.Metrics.ListRowHeight = 56
	theme.DefaultTheme.Metrics.MinTouchTarget = theme.TouchTargetAndroid
}

// TestListItemHeightsAreTheThreePublishedOnes pins M3's 56/72/88: the heights are what make a list
// scannable, and a two-line row that is one line tall makes the eye re-read every row.
func TestListItemHeightsAreTheThreePublishedOnes(t *testing.T) {
	withListMetrics(t)

	for _, c := range []struct {
		variant ListItemVariant
		want    float32
	}{
		{ListItemOneLine, 56},
		{ListItemTwoLine, 72},
		{ListItemThreeLine, 88},
	} {
		l := NewListItem().SetVariant(c.variant)
		if got := l.RowHeight(); got != c.want {
			t.Errorf("a %d-line row is %.0f tall, want %.0f", c.variant+1, got, c.want)
		}
	}
}

// TestListItemSecondaryPromotesTheHeight keeps the height following the CONTENT: an app that sets a
// secondary line should not also have to remember to change the variant.
func TestListItemSecondaryPromotesTheHeight(t *testing.T) {
	withListMetrics(t)
	l := NewListItem()
	if h := l.RowHeight(); h != 56 {
		t.Fatalf("an empty row is %.0f tall, want 56", h)
	}
	l.SetSecondary("second line")
	if l.Variant() != ListItemTwoLine {
		t.Error("setting a secondary line did not promote the row to two lines")
	}
	if h := l.RowHeight(); h != 72 {
		t.Errorf("a two-line row is %.0f tall, want 72", h)
	}
	// An explicit three-line variant is the app's call and is not downgraded.
	l.SetVariant(ListItemThreeLine)
	l.SetSecondary("still two lines of text")
	if l.Variant() != ListItemThreeLine {
		t.Error("SetSecondary downgraded an explicit three-line row")
	}
}

// TestListItemTextSitsBetweenTheSlots guards the geometry that a wide row hides: the leading slot,
// the text and the trailing slot must not overlap, or a trailing control is unreachable.
func TestListItemTextSitsBetweenTheSlots(t *testing.T) {
	withListMetrics(t)
	l := NewListItem().SetLeading("document").SetTrailing("chevron")
	l.SetFrameRect(graphics.NewRect(0, 0, 360, l.RowHeight()))
	bounds := l.FrameRect()

	lead := l.LeadingRect(bounds)
	trail, ok := l.TrailingRect(bounds)
	if !ok {
		t.Fatal("no trailing slot with a trailing icon")
	}
	tr := l.TextRect(bounds)
	if tr.X < lead.X+lead.Width {
		t.Errorf("the text starts at %.0f, under the leading slot which ends at %.0f", tr.X, lead.X+lead.Width)
	}
	if tr.X+tr.Width > trail.X {
		t.Errorf("the text ends at %.0f, over the trailing slot which starts at %.0f", tr.X+tr.Width, trail.X)
	}
	if trail.X+trail.Width != bounds.Width {
		t.Errorf("the trailing slot ends at %.0f, want the row's right edge %.0f", trail.X+trail.Width, bounds.Width)
	}
	// A slot reaches the touch floor: it holds a control.
	if lead.Width < theme.TouchTargetAndroid {
		t.Errorf("the leading slot is %.0f wide, under the %v floor", lead.Width, theme.TouchTargetAndroid)
	}
}

// TestListItemTrailingWidgetIsNotTheRowsToTake keeps a control in the trailing slot clickable: the
// row must not swallow the press that belongs to the switch beside it.
func TestListItemTrailingWidgetIsNotTheRowsToTake(t *testing.T) {
	withListMetrics(t)
	l := NewListItem()
	sw := NewSwitch()
	l.SetTrailingWidget(sw)
	l.SetFrameRect(graphics.NewRect(0, 0, 360, l.RowHeight()))
	clicked := false
	l.OnClicked = func() { clicked = true }

	r, ok := l.TrailingRect(l.FrameRect())
	if !ok {
		t.Fatal("no trailing slot with a trailing widget")
	}
	if l.OnMouseDown(r.X+r.Width/2, r.Y+r.Height/2, 0) {
		t.Error("the row took a press that belongs to its trailing control")
	}
	if clicked {
		t.Error("a press on the trailing control fired the row's click")
	}
	// Anywhere else on the row is the row's.
	if !l.OnMouseDown(20, l.RowHeight()/2, 0) {
		t.Fatal("a press on the row was not taken")
	}
	l.OnMouseUp(20, l.RowHeight()/2, 0)
	if !clicked {
		t.Error("a tap on the row did not fire OnClicked")
	}
}

// TestListItemTapCommitsOnRelease is the gesture contract the other rows share.
func TestListItemTapCommitsOnRelease(t *testing.T) {
	withListMetrics(t)
	l := NewListItem().SetPrimary("Alpha")
	l.SetFrameRect(graphics.NewRect(0, 0, 360, l.RowHeight()))
	fired := 0
	l.OnClicked = func() { fired++ }

	if !l.OnMouseDown(100, 20, 0) {
		t.Fatal("a press was not taken")
	}
	// Released off the row: nothing happens, so a mis-touch is recoverable.
	l.OnMouseUp(100, 500, 0)
	if fired != 0 {
		t.Errorf("a release off the row fired %d times", fired)
	}
	l.OnMouseDown(100, 20, 0)
	l.OnMouseUp(100, 20, 0)
	if fired != 1 {
		t.Errorf("a tap fired %d times, want 1", fired)
	}
}

// TestListItemIsAnInputTarget keeps it in the registry the touch audit checks.
func TestListItemIsAnInputTarget(t *testing.T) {
	if !meui.IsInteractive(NewListItem()) {
		t.Error("a ListItem handles input but is not in the interactive registry")
	}
}

var _ meui.EventListener = (*ListItem)(nil)
var _ meui.HoverTarget = (*ListItem)(nil)
