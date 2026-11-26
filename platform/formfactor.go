package platform

// FormFactor is HOW a surface should behave, as distinct from how it should look.
//
// It is deliberately separate from the theme's Platform. The theme decides
// appearance — which is a property of the OS — while the form factor decides
// behaviour, and behaviour follows the SIZE of the surface rather than the name of
// the OS. Back navigation, sheet-versus-dialog, scroll physics and the touch-target
// floor all belong here, and none of them is something a theme should be asked
// about.
//
// Deriving it from size rather than from runtime.GOOS is what makes the mobile
// layout testable on a desk: a 400px-wide desktop window IS a phone by these rules,
// so phone behaviour can be exercised without a device or an emulator.
type FormFactor uint8

const (
	// FormFactorDesktop is a pointer-driven surface: no touch floor, hover states,
	// dialogs rather than sheets.
	FormFactorDesktop FormFactor = iota
	// FormFactorPhone is a compact surface (<600dp). One pane, touch, sheets.
	FormFactorPhone
	// FormFactorTablet is a medium surface (600-839dp): touch, but room for a
	// navigation rail and two panes.
	FormFactorTablet
)

func (f FormFactor) String() string {
	switch f {
	case FormFactorPhone:
		return "phone"
	case FormFactorTablet:
		return "tablet"
	default:
		return "desktop"
	}
}

// Size-class breakpoints, in density-independent pixels. These are Android's own
// (compact <600, medium 600-839, expanded >=840) so that a layout which is right
// here is right there, and so the boundaries are a published spec rather than our
// taste.
const (
	compactMaxDP = 600
	mediumMaxDP  = 840
)

// FormFactorFor reports the form factor for a surface of the given size at the given
// content scale. width and height are in the window's own units; the scale DIVIDES
// them to density-independent pixels, which is the direction the toolkit already uses
// (a widget's local coordinate is the window's divided by its scale).
func FormFactorFor(width, height, scale float32) FormFactor {
	if width <= 0 || height <= 0 {
		return FormFactorDesktop
	}
	if scale <= 0 {
		scale = 1
	}
	w, h := width/scale, height/scale

	// The SMALLER dimension decides compactness, not the width alone: a phone in
	// landscape is wider than 600dp and is still a phone. This is the case a
	// width-only rule gets wrong, and it is the one that would put a phone into
	// tablet layout the moment it was turned sideways.
	small := w
	if h < small {
		small = h
	}
	if small < compactMaxDP {
		return FormFactorPhone
	}
	if w < mediumMaxDP {
		return FormFactorTablet
	}
	return FormFactorDesktop
}
