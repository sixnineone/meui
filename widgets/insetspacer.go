package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
)

// InsetSpacer reserves a strip at the bottom of a shell for an OS inset — the on-screen
// keyboard, today — so that a bar above it is LIFTED rather than covered.
//
// It exists because the toolkit REPORTS the keyboard through Insets.Keyboard but has no
// opinion about what to do with it: an app may draw behind the IME, and one that wants its
// navigation bar to stay visible puts this after the bar and sets the inset. It draws
// nothing, and at zero it is INVISIBLE, which the layout skips entirely — so a keyboard
// that is down costs nothing, rather than a zero-height row that could still take a hit.
//
// The shell drives it from the platform's inset change (see meui/demoapp):
//
//	if sp, ok := reg.Get("keyboardSpacer").(*widgets.InsetSpacer); ok {
//	    sp.SetInset(wnd.Insets().Keyboard)
//	}
type InsetSpacer struct {
	meui.BaseWidget
	inset float32
}

// NewInsetSpacer builds a spacer with no inset, i.e. hidden.
func NewInsetSpacer() *InsetSpacer {
	return &InsetSpacer{BaseWidget: *meui.NewBaseWidget()}
}

// SetInset sets the height to reserve. A negative value is clamped to zero rather than
// stored: an inset is a distance, and a negative one would move a bar DOWN over the
// content it is supposed to clear.
func (s *InsetSpacer) SetInset(h float32) *InsetSpacer {
	if h < 0 {
		h = 0
	}
	if s.inset == h {
		return s
	}
	s.inset = h
	s.MarkForRedraw()
	return s
}

// Inset is the height currently being reserved.
func (s *InsetSpacer) Inset() float32 { return s.inset }

// Visible is false at zero, so the layout skips the spacer instead of reserving a
// zero-height row. An explicit SetVisible(false) still hides it.
func (s *InsetSpacer) Visible() bool { return s.inset > 0 && s.BaseWidget.Visible() }

// PreferredDimensions is how the layout learns the reserved height. It takes the width it
// is given, because the strip spans the shell.
func (s *InsetSpacer) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	return w, s.inset
}

func (s *InsetSpacer) PreferredSize() (float32, float32) { return 0, s.inset }

// Paint draws nothing: the spacer is a reservation, not a surface.
func (s *InsetSpacer) Paint(canvas *graphics.Canvas, rect graphics.Rect) {}
