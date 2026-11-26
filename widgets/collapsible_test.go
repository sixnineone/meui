package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/layout"
	"github.com/sixnineone/meui/theme"
)

func newTestCollapsible(collapsed bool) *Collapsible {
	c := NewCollapsible()
	c.SetLabel("Bash")
	c.SetDetail("make check  45 sub")
	c.SetTrailing("1.2s")
	body := NewTextArea()
	body.SetReadOnly(true)
	body.SetText("ok: no runtime/cgo in the cgo-free desktop graph")
	body.SetLayoutProps(&layout.LayoutProps{FillWidth: true, FillHeight: true})
	c.SetBody(body)
	c.SetCollapsed(collapsed)
	c.SetFrameRect(graphics.NewRect(0, 0, 400, 200))
	return c
}

// TestACollapsedHeaderTakesNoRoomForItsBody is the contract that makes a transcript
// usable: a collapsed row's preferred height is the HEADER ALONE, so a list of a thousand
// of them can be placed in one pass without measuring a thousand bodies. It is also the
// thing a screenshot cannot show — a collapsed row looks the same whether it reserved
// space or not.
func TestACollapsedHeaderTakesNoRoomForItsBody(t *testing.T) {
	c := newTestCollapsible(true)
	_, ch := c.body.PreferredDimensions(c.body, 400, 200)
	if ch <= 0 {
		t.Fatal("the body reports no height, so the test would pass for the wrong reason")
	}

	_, collapsedH := c.PreferredDimensions(c, 400, 200)
	if collapsedH != c.HeaderHeight() {
		t.Errorf("a collapsed row is %.0f tall, want the header alone (%.0f) — it reserved %.0f for a body it is not showing",
			collapsedH, c.HeaderHeight(), ch)
	}

	c.SetCollapsed(false)
	_, openH := c.PreferredDimensions(c, 400, 200)
	if openH <= collapsedH {
		t.Errorf("expanding did not grow the row: %.0f then %.0f", collapsedH, openH)
	}
	if openH < collapsedH+ch {
		t.Errorf("the expanded row is %.0f, want at least header+body (%.0f)", openH, collapsedH+ch)
	}
}

// TestTheHeaderTogglesAndTheBodyDoesNot: a press in the header opens, a press in the body
// does not. The second half matters more than it looks — the body is a transcript, it
// holds selectable text, and a disclosure that swallows presses in the content makes the
// content unreadable without saying so.
func TestTheHeaderTogglesAndTheBodyDoesNot(t *testing.T) {
	c := newTestCollapsible(true)
	head := c.HeaderHeight()

	if !c.OnMouseDown(20, head/2, 0) {
		t.Fatal("a press on the header was not taken")
	}
	if c.Collapsed() {
		t.Error("a press on the header did not open the row")
	}

	c.OnMouseDown(20, head/2, 0)
	if !c.Collapsed() {
		t.Error("a second press on the header did not close it")
	}

	// The row must be OPEN for a body press to exist: a collapsed row is only
	// header-tall, so a point below the header is outside the widget and says nothing
	// about who gets the press.
	c.SetCollapsed(false)
	if c.OnMouseDown(20, head+20, 0) {
		t.Error("a press in the BODY was consumed by the disclosure, so the body's text could never be selected")
	}
	if c.Collapsed() {
		t.Error("a press in the body changed the state")
	}

	// And a collapsed row takes any press, because every part of it is header.
	c.SetCollapsed(true)
	if !c.OnMouseDown(20, head/2, 0) {
		t.Error("a collapsed row refused a press on its own height")
	}
}

// TestToggleReports: an app persists the state or records that the user opened a group, so
// the callback is the interface, and a same-state SetCollapsed must not report.
func TestToggleReports(t *testing.T) {
	c := newTestCollapsible(false)
	var got []bool
	c.OnToggle = func(v bool) { got = append(got, v) }

	c.SetCollapsed(true)
	c.SetCollapsed(true) // no move, no report
	c.SetCollapsed(false)
	if len(got) != 2 || !got[0] || got[1] {
		t.Errorf("OnToggle reported %v, want [true false]", got)
	}
}

// TestCollapsibleStartsExpanded: a widget that arrives with its content hidden reads as
// empty, and a transcript that starts folded is a transcript that looks broken.
func TestCollapsibleStartsExpanded(t *testing.T) {
	if NewCollapsible().Collapsed() {
		t.Error("a new Collapsible is collapsed")
	}
}

// TestTheHeaderReachesTheTouchFloor: the header is the tap target, and on a phone it has
// to be a finger-sized one. It reaches the floor by construction, and that is worth
// pinning — the header is the one thing every transcript row is.
func TestTheHeaderReachesTheTouchFloor(t *testing.T) {
	saved := theme.DefaultTheme
	t.Cleanup(func() { theme.SetDefaultTheme(saved) })
	th := *saved
	theme.SetDefaultTheme(&th)
	theme.DefaultTheme.Metrics.MinTouchTarget = theme.TouchTargetAndroid

	if h := newTestCollapsible(true).HeaderHeight(); h < theme.TouchTargetAndroid {
		t.Errorf("the header is %.0f tall, under the %v floor", h, theme.TouchTargetAndroid)
	}
}

// TestCollapsibleIsInTheInteractiveRegistry: a header that is not in the registry gets no
// hit-slop, which is the Phase A audit's concern — but the registry is keyed by TYPE NAME,
// so a new widget has to be added to it or its header is never expanded.
func TestCollapsibleIsInTheInteractiveRegistry(t *testing.T) {
	if !meui.IsInteractive(newTestCollapsible(true)) {
		t.Error("Collapsible is not in the interactive registry, so its header never reaches the touch floor")
	}
}

// TestACollapsedTranscriptRowCostsNoPixels is the property that makes a transcript of six
// hundred steps usable, measured in PIXELS rather than asserted about a flag — a collapsed
// row must not draw its body at all, because "not visible" and "not drawn" differ by the
// whole cost of a thousand Markdown documents.
//
// A transcript body is a MarkdownView, so this is also the one test that covers the two
// halves together: a collapsed step really is a header, and the renderer is what expands.
func TestACollapsedTranscriptRowCostsNoPixels(t *testing.T) {
	const w, h = 400, 220

	build := func() (*Collapsible, *MarkdownView) {
		c := NewCollapsible()
		c.SetLabel("· bash")
		c.SetDetail("Check the gate")
		c.SetTrailing("45 sub")
		c.SetFrameRect(graphics.NewRect(0, 0, w, h))
		mv := NewMarkdownView()
		mv.SetText("A long body with **bold** and a paragraph of prose, a list:\n\n" +
			"1. one\n2. two\n3. three\n")
		mv.SetFrameRect(graphics.NewRect(0, 0, w, 160))
		c.SetBody(mv)
		return c, mv
	}

	// Paint a collapsed row and an expanded one, and compare how much was drawn.
	paint := func(collapsed bool) int {
		c, _ := build()
		c.SetCollapsed(collapsed)
		if !collapsed {
			// Give the body a real share of the row; a collapsed row is header-only, so
			// without this the two states would have the same rect to draw into.
			c.SetFrameRect(graphics.NewRect(0, 0, w, h))
		}
		dev := graphics.NewSoftDevice(w, h)
		c.Paint(graphics.NewCanvas(dev), graphics.NewRect(0, 0, w, h))
		drawn := 0
		for _, p := range dev.Pixels() {
			if p != 0 {
				drawn++
			}
		}
		return drawn
	}

	collapsed := paint(true)
	expanded := paint(false)
	if collapsed == 0 {
		t.Fatal("a collapsed row drew nothing at all; not even its header")
	}
	if expanded <= collapsed {
		t.Errorf("expanding the row drew %d pixels against %d collapsed: the body is not "+
			"being painted, so a transcript pays for every body whether or not it is open",
			expanded, collapsed)
	}
}
