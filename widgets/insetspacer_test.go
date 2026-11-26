package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/layout"
)

// TestInsetSpacerReservesOnlyWhenThereIsAnInset pins the whole contract of the widget:
// nothing is reserved while the keyboard is down, the inset is what is reserved while it
// is up, and a negative value cannot move a bar the wrong way.
func TestInsetSpacerReservesOnlyWhenThereIsAnInset(t *testing.T) {
	s := NewInsetSpacer()
	if s.Visible() {
		t.Error("a spacer with no inset reports itself visible")
	}
	if w, h := s.PreferredDimensions(nil, 300, 600); h != 0 || w != 300 {
		t.Errorf("with no inset PreferredDimensions = %vx%v, want 300x0", w, h)
	}

	s.SetInset(259)
	if !s.Visible() {
		t.Error("a spacer with an inset reports itself hidden")
	}
	if w, h := s.PreferredDimensions(nil, 300, 600); h != 259 || w != 300 {
		t.Errorf("PreferredDimensions = %vx%v, want 300x259", w, h)
	}

	s.SetInset(-5)
	if s.Inset() != 0 || s.Visible() {
		t.Errorf("a negative inset was kept as %v; it must clamp to zero", s.Inset())
	}

	// And it is not a tap target: expanding it would steal hits from the bar above it.
	if meui.IsInteractive(s) {
		t.Error("an InsetSpacer is in the interactive registry")
	}
}

// TestInsetSpacerLiftsWhateverIsAboveIt is the reason the widget exists, asserted on a real
// VStack rather than on arithmetic: a keyboard inset has to take its height out of the
// CONTENT so the bar above it rises, instead of being covered.
func TestInsetSpacerLiftsWhateverIsAboveIt(t *testing.T) {
	shell := meui.NewPanel()
	shell.SetLayout(layout.NewVStack())
	shell.SetFrameRect(graphics.NewRect(0, 0, 300, 600))

	content := meui.NewPanel()
	content.SetLayoutProps(&layout.LayoutProps{FillHeight: true})
	bar := meui.NewPanel()
	bar.SetLayoutProps(&layout.LayoutProps{MinimumHeight: 64})
	spacer := NewInsetSpacer()

	shell.AddChild(content)
	shell.AddChild(bar)
	shell.AddChild(spacer)

	// Keyboard down: the spacer is skipped and the bar is not lifted.
	shell.ValidateLayout()
	down := bar.FrameRect()

	// Keyboard up: the reserved strip comes out of the content, so the bar rises and the
	// strip reaches the bottom edge.
	spacer.SetInset(259)
	shell.ValidateLayout()
	up := bar.FrameRect()
	sp := spacer.FrameRect()

	if sp.Height != 259 {
		t.Errorf("the spacer was laid out %.0f tall, want 259", sp.Height)
	}
	if sp.Y+sp.Height != 600 {
		t.Errorf("the spacer ends at %.0f, want the shell's bottom edge 600", sp.Y+sp.Height)
	}
	if up.Y >= down.Y {
		t.Errorf("the bar did not move: %.0f -> %.0f", down.Y, up.Y)
	}
	if up.Y+up.Height > sp.Y {
		t.Errorf("the bar ends at %.0f, over the spacer which starts at %.0f", up.Y+up.Height, sp.Y)
	}
}
