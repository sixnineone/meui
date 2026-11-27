package platform

import "github.com/sixnineone/meui/graphics"

// Display is one physical screen.
//
// It used to be a stub that always reported scale 1.0. The SDL3 adapter now
// enumerates real displays and reports the content scale SDL reports, which is
// what makes Window.Scale() — and therefore HiDPI — real on the desktop and on
// mobile (see MOBILE-PLAN.md P4).
type Display struct {
	id     int
	Name   string
	ScaleX float32
	ScaleY float32
}

func (d *Display) ID() int { return d.id }

// NewDisplay builds a Display for a backend. Name may be empty; a scale below 1
// is clamped, because 1.0 is the "unknown, assume unscaled" floor.
func NewDisplay(id int, name string, scaleX, scaleY float32) *Display {
	if scaleX < 1 {
		scaleX = 1
	}
	if scaleY < 1 {
		scaleY = 1
	}
	return &Display{id: id, Name: name, ScaleX: scaleX, ScaleY: scaleY}
}

// BestDisplayForRect reports the display a rect falls on, which is how
// Window.Scale() learns the content scale of the monitor it is actually on.
//
// It is a package-level variable rather than a function because the toolkit
// calls it from a place that holds no App (a window's draw callback), the same
// way InvokeTask and GlobalClipboard work: a backend installs its own with
// SetBestDisplayForRect at startup. The default reports scale 1.0, matching
// The reference toolkit's X11 path, where the window realizes a scale of 1.0 and ScaleFonts(1)
// bumps the theme's base font to cap-8.
var BestDisplayForRect = func(r graphics.Rect) *Display {
	return &Display{ScaleX: 1, ScaleY: 1}
}

// SetBestDisplayForRect installs the backend's display lookup. Nil restores the
// scale-1.0 default.
func SetBestDisplayForRect(fn func(graphics.Rect) *Display) {
	if fn == nil {
		BestDisplayForRect = func(r graphics.Rect) *Display {
			return &Display{ScaleX: 1, ScaleY: 1}
		}
		return
	}
	BestDisplayForRect = fn
}
