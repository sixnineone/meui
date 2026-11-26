package widgets

// Sticky pins a box to the top of its viewport while its section is in view. The pane
// MOVES the frame (paint, hit test, hover and the cursor walk all read frames), so these
// tests assert the frame and the hit, not a paint-time offset.

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

func newStickyScene(t *testing.T) (*ScrollPane, *meui.Panel, *meui.Panel) {
	t.Helper()
	theme.SetDefaultTheme(theme.DefaultTheme)
	pane := NewScrollPane()
	pane.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	pane.SetHitTestContent(true)

	content := meui.NewPanel()
	content.SetFrameRect(graphics.NewRect(0, 0, 400, 1200))
	pane.SetContentSize(400, 1200)
	pane.SetContent(content)

	// A section with a sticky header inside it: the header must stop once the section's
	// bottom reaches it.
	section := meui.NewPanel()
	section.SetFrameRect(graphics.NewRect(0, 300, 400, 250))
	content.AddChild(section)

	bar := meui.NewPanel()
	bar.SetBackground(graphics.Color(0xFFFFFFFF))
	bar.SetSticky(true)
	bar.SetFrameRect(graphics.NewRect(0, 20, 400, 40))
	section.AddChild(bar)
	return pane, section, bar
}

func TestStickyPinsToTheViewportAndClampsToItsSection(t *testing.T) {
	pane, _, bar := newStickyScene(t)

	// Above its natural position: nothing moves. (Absolute y = section 300 + 20.)
	pane.RefreshSticky()
	if got := bar.FrameRect().Y; got != 20 {
		t.Fatalf("at scroll 0 the header is at y=%.0f, want its natural 20", got)
	}

	// Scrolled past it: pinned to the viewport's top edge, which in content space is the
	// scroll offset. The section starts at 300, so the bar's content y is the offset and
	// its y inside the section is offset-300.
	pane.ScrollTo(400)
	pane.RefreshSticky()
	if got := bar.FrameRect().Y; got != 100 {
		t.Fatalf("pinned header local y=%.0f, want 100 (content y 400)", got)
	}

	// Past the section's bottom the header must stop, not follow the scroll forever:
	// section bottom 550, header 40 → content y 510, local 510-300 = 210.
	pane.ScrollTo(600)
	pane.RefreshSticky()
	if got := bar.FrameRect().Y; got != 210 {
		t.Fatalf("header local y=%.0f after its section, want 210 (clamped)", got)
	}

	// Scrolling back restores the natural position.
	pane.ScrollTo(0)
	pane.RefreshSticky()
	if got := bar.FrameRect().Y; got != 20 {
		t.Fatalf("header did not return to its natural y: %.0f", got)
	}
}

func TestAPinnedHeaderTakesTheClickWhereItIsDrawn(t *testing.T) {
	theme.SetDefaultTheme(theme.DefaultTheme)
	pane := NewScrollPane()
	pane.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	pane.SetHitTestContent(true)
	content := meui.NewPanel()
	content.SetFrameRect(graphics.NewRect(0, 0, 400, 900))
	pane.SetContentSize(400, 900)
	pane.SetContent(content)

	bar := meui.NewPanel()
	bar.SetBackground(graphics.Color(0xFFFFFFFF))
	bar.SetSticky(true)
	bar.SetFrameRect(graphics.NewRect(0, 100, 400, 40))
	content.AddChild(bar)

	pane.ScrollTo(300)
	pane.RefreshSticky()
	// The header is pinned at the viewport's top: a press at the top of the pane must hit
	// it, even though the content there belongs to whatever scrolled underneath.
	if hit := pane.PanelAt(10, 10); hit != meui.Widget(bar) {
		t.Fatalf("the pinned header did not take the hit at the top of the viewport (%T)", hit)
	}
}
