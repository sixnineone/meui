package meui

// `z:` orders SIBLINGS for painting and hit testing. It is what an overlay, a
// floating action button or a dropdown needs, and the default (0) must preserve
// insertion order exactly, which is every existing widget's behaviour.

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
)

func TestZIndexOrdersSiblingsForPaintAndHit(t *testing.T) {
	parent := NewPanel()
	red := NewPanel()
	red.SetBackground(graphics.Color(0xFFCC0000))
	red.SetFrameRect(graphics.NewRect(0, 0, 40, 40))
	blue := NewPanel()
	blue.SetBackground(graphics.Color(0xFF0000CC))
	blue.SetFrameRect(graphics.NewRect(0, 0, 40, 40))

	parent.AddChild(red)
	parent.AddChild(blue)

	// Insertion order: the later child is on top, and it takes the hit.
	img := paintPanel(t, parent, 40, 40)
	if px := img.RGBAAt(20, 20); px.B <= px.R {
		t.Fatalf("with no z the later child should be on top, got %v", px)
	}
	if hit := parent.PanelAt(20, 20); hit != Widget(blue) {
		t.Fatalf("hit with no z = %T, want the later child", hit)
	}

	// Raise the red one: it now paints over blue and takes the hit.
	red.SetZIndex(1)
	img = paintPanel(t, parent, 40, 40)
	if px := img.RGBAAt(20, 20); px.R <= px.B {
		t.Fatalf("with z=1 the red child should be on top, got %v", px)
	}
	if hit := parent.PanelAt(20, 20); hit != Widget(red) {
		t.Fatalf("hit with z=1 = %T, want the raised child", hit)
	}

	// And back to insertion order.
	red.SetZIndex(0)
	if hit := parent.PanelAt(20, 20); hit != Widget(blue) {
		t.Fatalf("hit after z reset = %T, want the later child", hit)
	}
}

// TestZIndexKeepsInsertionOrderForEqualZ pins the stable tie-break: two siblings with
// the same z keep the order they were added in, at every z, not just zero.
func TestZIndexKeepsInsertionOrderForEqualZ(t *testing.T) {
	parent := NewPanel()
	first := NewPanel()
	first.SetBackground(graphics.Color(0xFF00FF00))
	first.SetFrameRect(graphics.NewRect(0, 0, 40, 40))
	first.SetZIndex(3)
	second := NewPanel()
	second.SetBackground(graphics.Color(0xFF0000FF))
	second.SetFrameRect(graphics.NewRect(0, 0, 40, 40))
	second.SetZIndex(3)
	parent.AddChild(first)
	parent.AddChild(second)

	img := paintPanel(t, parent, 40, 40)
	if px := img.RGBAAt(20, 20); px.B <= px.G {
		t.Fatalf("equal z lost insertion order, got %v", px)
	}
}
