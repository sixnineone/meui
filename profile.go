package meui

import (
	"reflect"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

// Touch profile: mobile-friendly hit-testing (mobile assessment M1 profile layer,
// hit-slop piece). When enabled, interactive widgets get an expanded hit area of at
// least the platform's touch floor, so small controls (checkboxes, close buttons) are
// comfortable finger targets.
//
// It is enabled from the WINDOW's form factor rather than by hand. It used to be a
// manual SetTouchProfile that NOTHING ever called, so the floor existed and had never
// once been active — which is why every control in the toolkit was still 26-34px on a
// phone. A capability nobody switches on is not a capability.
const legacyMinTouchTarget = float32(44)

// defaultTouchProfile is the process-wide DEFAULT: what a window that has not been told otherwise
// uses. A window normally gets its own setting from App.AddWindow (derived from its form factor), so
// one phone-sized window no longer changes how a desktop window's controls are hit-tested.
var defaultTouchProfile bool

// MinTouchTarget is the effective floor in logical px: the theme's metric when the
// preset sets one (48 on Android, 44 on iOS), else the historical 44. The theme is the
// right source because the floor is a published per-platform number, and putting it in
// the preset is what lets one widget set serve both.
func MinTouchTarget() float32 {
	if theme.DefaultTheme != nil && theme.DefaultTheme.Metrics.MinTouchTarget > 0 {
		return theme.DefaultTheme.Metrics.MinTouchTarget
	}
	return legacyMinTouchTarget
}

// SetTouchProfile enables/disables the DEFAULT touch floor. A window carries its own setting (see
// Window.SetTouchProfile) and App.AddWindow derives one from the window's form factor, so this is the
// fallback for a window that has none — the headless tests, and a host with one logical surface.
// It stays exported for tests and for a host that knows better than the size classes.
func SetTouchProfile(on bool) { defaultTouchProfile = on }

// SetTouchProfile turns the touch floor on or off FOR THIS WINDOW. It is a manual setting: a later
// resize will not override it.
func (w *Window) SetTouchProfile(on bool) {
	w.touchProfile, w.touchProfileSet, w.touchProfileAuto = on, true, false
}

// setDerivedTouchProfile is the setting App.AddWindow derives from the window's form factor, and
// the one a resize re-derives.
func (w *Window) setDerivedTouchProfile(on bool) {
	w.touchProfile, w.touchProfileSet, w.touchProfileAuto = on, true, true
}

// TouchProfile reports whether THIS window uses the touch floor. An explicit setting wins; otherwise
// it is the process default.
func (w *Window) TouchProfile() bool {
	if w == nil || !w.touchProfileSet {
		return defaultTouchProfile
	}
	return w.touchProfile
}

// touchFloorFor resolves the floor from the window a widget belongs to, so a hit test inside one
// window is never affected by another window's size.
func touchFloorFor(w Widget) bool {
	if win := windowOf(w); win != nil {
		return win.TouchProfile()
	}
	return defaultTouchProfile
}

// windowOf is the window at the top of a widget's parent chain, if any.
func windowOf(w Widget) *Window {
	if ww, ok := w.(interface{ Window() *Window }); ok {
		return ww.Window()
	}
	return nil
}

// TouchProfileForSize reports whether a surface of this size wants the touch floor,
// using the same size classes the window uses. It exists so a host or a test can ask
// the question without a window.
// TouchProfile reports whether the floor is active, which the touch audit asserts:
// the floor hung off a switch nothing flipped for its whole life, so "is it on" is a
// question worth being able to ask.
func TouchProfile() bool { return defaultTouchProfile }

func TouchProfileForSize(width, height, scale float32) bool {
	return platform.FormFactorFor(width, height, scale) != platform.FormFactorDesktop
}

// IsInteractive reports whether a widget is a TAPPABLE TARGET, which is what the
// hit-slop expansion turns on. It is exported because the touch AUDIT must ask the
// same question the hit test does: a widget that is interactive but that this does not
// recognise stays 14px with nothing failing anywhere.
//
// This predicate was WRONG in a way that made the whole floor inert, and the audit is
// what found it. It used to ask two questions and both were the wrong one:
//
//   - HasMouseHandlers reports whether callbacks are INSTALLED, but every real control
//     (Button, CheckBox, Switch, Slider, …) handles input by OVERRIDING OnMouseDown
//     instead, so it answered false for all of them;
//   - `interface{ SetOnClicked(func()) }` never matched either, because Button's method
//     is fluent — `SetOnClicked(func()) *Button` — and a method that returns something
//     does not satisfy an interface that returns nothing.
//
// So nothing was ever recognised and the expansion expanded nothing. The fix is a
// NAME-BASED registry of the types that are targets. Reflection cannot answer it: Go
// generates a distinct wrapper for a promoted method, so comparing method pointers
// reports "overridden" for every widget that merely embeds BaseWidget — measured on
// ProgressBar, which declares no input methods at all and whose promoted pointers
// still differ from BaseWidget's.
//
// Containers are excluded separately (expanding a panel would steal hits from the gaps
// between its children), and so is Label: it handles input for text selection, but a
// label is not something a finger aims at, and expanding one beside a checkbox would
// take the checkbox's own expansion.
func IsInteractive(w Widget) bool {
	if w == nil {
		return false
	}
	if h, ok := w.(interface{ HasMouseHandlers() bool }); ok && h.HasMouseHandlers() {
		return true
	}
	// A clickable box is a CONTAINER that is still a target, and a MouseArea is a
	// target that overrides its handlers rather than installing callbacks — neither
	// is visible to the name registry below, and on a phone a 40px card that is
	// tappable must still get the touch floor.
	if p, ok := w.(*Panel); ok && p.Clickable() {
		return true
	}
	if _, ok := w.(*MouseArea); ok {
		return true
	}
	t := reflect.TypeOf(w)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return interactiveTypes[t.Name()]
}

// RegisterInteractiveType adds a widget type to the interactive registry. An APP-defined widget that
// handles input needs this: the registry is keyed by type name, and without an entry the touch-target
// floor would silently not apply to it on a phone — a widget that works with a mouse and is
// frustrating with a thumb.
func RegisterInteractiveType(name string) {
	if name == "" {
		return
	}
	interactiveTypes[name] = true
}

// interactiveTypes names the widget types that are tappable targets. It is a registry
// rather than a method-set test because Go cannot distinguish an overridden method from
// a promoted one, and an explicit list is auditable — the touch audit asserts that
// everything its scene expects to be tappable is in here.
var interactiveTypes = map[string]bool{
	"Button": true, "CheckBox": true, "RadioButton": true, "Switch": true,
	"Slider": true, "ComboBox": true, "SpinBox": true, "TextField": true,
	"TextArea": true, "TabStrip": true, "ScrollBar": true, "ColorField": true,
	"ColorPicker": true, "DatePicker": true, "TimePicker": true, "Calendar": true,
	"Collapsible":   true,
	"NavigationBar": true, "NavigationRail": true, "TopAppBar": true, "Snackbar": true,
	"BottomSheet": true, "ActionSheet": true, "AlertDialog": true, "SearchView": true, "ListItem": true, "Waveform": true, "SwipeActionRow": true, "PullToRefresh": true, "AdaptiveTable": true,
}

// TouchHitFrame expands frame f, when needed, so that interactive
// widgets present at least MinTouchTarget square hit area centered on
// the widget. Non-interactive widgets (no mouse handlers, no click
// handler) and containers pass through unchanged — expanding panels
// would steal hits from sibling gaps.
func TouchHitFrame(child Widget, f graphics.Rect) graphics.Rect {
	return TouchHitFrameWith(child, f, defaultTouchProfile)
}

// TouchHitFrameWith is TouchHitFrame for a specific floor setting (a window's, or a test's).
func TouchHitFrameWith(child Widget, f graphics.Rect, floorOn bool) graphics.Rect {
	if !floorOn {
		return f
	}
	if !IsInteractive(child) {
		return f
	}
	if _, isContainer := child.(Container); isContainer {
		return f
	}
	floor := MinTouchTarget()
	expanded := f
	if expanded.Width < floor {
		cx := f.X + f.Width/2
		expanded.X = cx - floor/2
		expanded.Width = floor
	}
	if expanded.Height < floor {
		cy := f.Y + f.Height/2
		expanded.Y = cy - floor/2
		expanded.Height = floor
	}
	return expanded
}

// ChildLocal maps a point in the PARENT's space to the child's local space, when the point is inside
// the child's hit frame. The hit frame may be expanded to the touch floor; the COORDINATES are always
// taken against the child's REAL frame.
//
// The distinction is the whole point of the floor. Expanding the frame is meant to widen where a
// finger may land; it must not move the widget's origin. Using the expanded frame for the subtraction
// shifted every local coordinate by half the expansion (11px for a 22px control), so a small
// control's top half reported coordinates from its bottom half — the spin box's up arrow acted as
// DOWN, and the only way to press it was to click ABOVE the widget.
func ChildLocal(child Widget, x, y float32) (float32, float32, bool) {
	return ChildLocalFor(child, x, y, defaultTouchProfile)
}

// ChildLocalFor is ChildLocal with an explicit floor setting.
func ChildLocalFor(child Widget, x, y float32, floorOn bool) (float32, float32, bool) {
	real := child.FrameRect()
	hit := TouchHitFrameWith(child, real, floorOn)
	if x < hit.X || x > hit.X+hit.Width || y < hit.Y || y > hit.Y+hit.Height {
		return 0, 0, false
	}
	return x - real.X, y - real.Y, true
}

// ChildLocalAmong is ChildLocal with the sibling-clamped expansion (see TouchHitFrameAmong).
func ChildLocalAmong(child Widget, x, y float32, siblings []Widget) (float32, float32, bool) {
	return ChildLocalAmongFor(child, x, y, siblings, defaultTouchProfile)
}

// ChildLocalAmongFor is ChildLocalAmong with an explicit floor setting.
func ChildLocalAmongFor(child Widget, x, y float32, siblings []Widget, floorOn bool) (float32, float32, bool) {
	real := child.FrameRect()
	hit := TouchHitFrameAmongWith(child, real, siblings, floorOn)
	if x < hit.X || x > hit.X+hit.Width || y < hit.Y || y > hit.Y+hit.Height {
		return 0, 0, false
	}
	return x - real.X, y - real.Y, true
}

// TouchHitFrameAmong expands a child's hit frame to the floor and then CLAMPS the
// expansion against its siblings, so two close targets cannot both claim the same
// pixel.
//
// This is Android's own caveat, stated in its touch-target guidance: very small and
// close-together targets cannot all be expanded, and the honest answer is to split the
// gap between them rather than let one swallow the other. Without it the hit test walks
// children in reverse and the LAST one wins the overlap, so a checkbox beside another
// checkbox becomes partly unreachable — which is worse than not expanding at all,
// because it fails only near the edge and looks like a stray mis-tap.
func TouchHitFrameAmong(child Widget, f graphics.Rect, siblings []Widget) graphics.Rect {
	return TouchHitFrameAmongWith(child, f, siblings, defaultTouchProfile)
}

// TouchHitFrameAmongWith is TouchHitFrameAmong for a specific floor setting.
func TouchHitFrameAmongWith(child Widget, f graphics.Rect, siblings []Widget, floorOn bool) graphics.Rect {
	out := TouchHitFrameWith(child, f, floorOn)
	if out == f || !floorOn {
		return out
	}
	for _, sib := range siblings {
		if sib == nil || sib == child || !IsInteractive(sib) {
			continue
		}
		if _, isContainer := sib.(Container); isContainer {
			continue
		}
		sf := sib.FrameRect()
		// Only a sibling that shares a band with this one competes for the gap: one
		// above or below it does not.
		if sf.X < out.X+out.Width && sf.X+sf.Width > out.X &&
			sf.Y < out.Y+out.Height && sf.Y+sf.Height > out.Y {
			if sf.X+sf.Width <= f.X {
				// To the left: stop at the middle of the gap.
				mid := sf.X + sf.Width + (f.X-(sf.X+sf.Width))/2
				if mid > out.X {
					out.Width -= mid - out.X
					out.X = mid
				}
			} else if sf.X >= f.X+f.Width {
				mid := f.X + f.Width + (sf.X-(f.X+f.Width))/2
				if mid < out.X+out.Width {
					out.Width = mid - out.X
				}
			}
			if sf.Y+sf.Height <= f.Y {
				mid := sf.Y + sf.Height + (f.Y-(sf.Y+sf.Height))/2
				if mid > out.Y {
					out.Height -= mid - out.Y
					out.Y = mid
				}
			} else if sf.Y >= f.Y+f.Height {
				mid := f.Y + f.Height + (sf.Y-(f.Y+f.Height))/2
				if mid < out.Y+out.Height {
					out.Height = mid - out.Y
				}
			}
		}
	}
	if out.Width < 0 {
		out.Width = 0
	}
	if out.Height < 0 {
		out.Height = 0
	}
	return out
}
