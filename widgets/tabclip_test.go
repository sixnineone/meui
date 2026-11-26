package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/layout"
	"github.com/sixnineone/meui/theme"
)

// TestTheBarPaintsItsItemsThroughAParent is the question the demo raised: the bar paints
// and the items do not, and both are drawn by the same loop of the same Paint — so the
// difference cannot be the widget.
//
// Every widget test so far paints DIRECTLY, which is not evidence: a real page's parent
// supplies the rect. So this paints the same strip both ways under the preset the demo
// loads, and compares.
func TestTheBarPaintsItsItemsThroughAParent(t *testing.T) {
	saved := theme.DefaultTheme
	t.Cleanup(func() { theme.SetDefaultTheme(saved) })
	if err := theme.LoadPreset("breeze-dark"); err != nil {
		t.Fatal(err)
	}
	const w, h = 560, 26

	newStrip := func(withItems bool) *TabStrip {
		s := NewTabStrip()
		if withItems {
			s.SetItems([]TabStripItem{
				{ID: "a", Label: "tab-A"}, {ID: "b", Label: "tab-B"}, {ID: "c", Label: "tab-C"}})
			s.SetSelected(2)
		}
		s.SetFrameRect(graphics.NewRect(0, 0, w, h))
		return s
	}
	count := func(dev interface{ Pixels() []uint32 }) int {
		d := 0
		for _, p := range dev.Pixels() {
			if p != 0 {
				d++
			}
		}
		return d
	}

	// (a) direct
	direct := func(withItems bool) int {
		s := newStrip(withItems)
		dev := graphics.NewSoftDevice(w, h)
		s.Paint(graphics.NewCanvas(dev), graphics.NewRect(0, 0, w, h))
		return count(dev)
	}
	// (b) through a Panel with a VStack, which is how the demo's document pane is built
	throughParent := func(withItems bool) int {
		s := newStrip(withItems)
		p := meui.NewPanel()
		v := layout.NewVStack()
		v.Spacing = 0
		p.SetLayout(v)
		p.AddChild(s)
		p.SetFrameRect(graphics.NewRect(0, 0, w, h))
		p.ValidateLayout()
		dev := graphics.NewSoftDevice(w, h)
		p.Paint(graphics.NewCanvas(dev), graphics.NewRect(0, 0, w, h))
		return count(dev)
	}

	t.Logf("direct:            bare=%d withItems=%d", direct(false), direct(true))
	t.Logf("through a parent:  bare=%d withItems=%d", throughParent(false), throughParent(true))

	if direct(true) <= direct(false) {
		t.Error("painted directly, the items do not appear — the widget itself is wrong")
	}
	if throughParent(true) <= throughParent(false) {
		t.Errorf("painted through a parent the items do not appear: bare=%d withItems=%d. "+
			"The widget is right and the PARENT's rect is not.",
			throughParent(false), throughParent(true))
	}
}
