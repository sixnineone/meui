package behaviour

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

// touchAuditQML is a spread of the interactive widgets a phone actually has to be
// tappable. Small ones are the point: a CheckBox is 14px, a Switch is 22px tall.
const touchAuditQML = `
Window {
  width: 360
  height: 640
  VStack {
    anchors.fill: parent
    spacing: 6
    margins: 12
    Button { id: wButton; text: "b"; width: 80; height: 28 }
    CheckBox { id: wCheck; text: "c"; width: 140; height: 20 }
    RadioButton { id: wRadio; text: "r"; width: 140; height: 20 }
    Switch { id: wSwitch; width: 50; height: 22 }
    Slider { id: wSlider; width: 200; height: 20 }
    ComboBox { id: wCombo; width: 180; height: 28 }
    TextField { id: wText; width: 200; height: 26 }
    SpinBox { id: wSpin; width: 120; height: 26 }
    ProgressBar { id: wProgress; width: 200; height: 8 }
  }
}`

// TestTouchFloorIsActiveOnAPhone is the regression test for the bug Phase A actually
// found: the touch floor existed, was correct, and had NEVER BEEN SWITCHED ON, because
// it hung off a SetTouchProfile that nothing called. A capability nobody enables is
// not a capability, so this asserts the state rather than the mechanism.
func TestTouchFloorIsActiveOnAPhone(t *testing.T) {
	savedTheme, savedProfile := theme.DefaultTheme, meui.TouchProfile()
	t.Cleanup(func() {
		theme.SetDefaultTheme(savedTheme)
		meui.SetTouchProfile(savedProfile)
	})

	// A copy, not LoadPreset: LoadPreset also mutates the shared DefaultTabTheme, and
	// restoring only DefaultTheme leaves the tab height changed for later tests.
	th := *savedTheme
	theme.SetDefaultTheme(&th)
	// After, not before: withStyleDefaults replaces Metrics wholesale when
	// ControlHeight is 0, discarding anything set on the copy.
	theme.DefaultTheme.Metrics.MinTouchTarget = theme.TouchTargetAndroid

	// A phone-sized window turns the floor on; the desktop default turns it off.
	meui.SetTouchProfile(meui.TouchProfileForSize(360, 640, 1))
	if !meui.TouchProfile() {
		t.Fatal("a 360x640 surface did not enable the touch floor")
	}
	if meui.TouchProfileForSize(1280, 800, 1) {
		t.Error("a 1280x800 surface enabled the touch floor; a pointer-driven window has no need of it")
	}

	floor := meui.MinTouchTarget()
	if floor != theme.TouchTargetAndroid {
		t.Errorf("the floor is %v with the android preset, want %v", floor, theme.TouchTargetAndroid)
	}

	// And it must actually expand a small control.
	small := graphics.NewRect(0, 0, 14, 14)
	got := meui.TouchHitFrame(&interactiveStub{}, small)
	if got.Width < floor || got.Height < floor {
		t.Errorf("a 14x14 control expanded to %vx%v, want at least %v", got.Width, got.Height, floor)
	}
}

// TestEveryInteractiveWidgetReachesTheFloor is the audit. It walks a laid-out tree and
// requires every interactive, non-container widget's HIT rect to reach the floor.
//
// It is deliberately not a check of the VISUAL size: Android's own guidance is that a
// 24dp icon inside a 48dp target is correct, so a small-looking control with a large
// hit area passes and should. What it catches is a widget that is interactive but that
// TouchHitFrame does not RECOGNISE as interactive — it expands nothing, and the target
// stays 14px with nothing failing anywhere.
func TestEveryInteractiveWidgetReachesTheFloor(t *testing.T) {
	savedTheme, savedProfile := theme.DefaultTheme, meui.TouchProfile()
	t.Cleanup(func() {
		theme.SetDefaultTheme(savedTheme)
		meui.SetTouchProfile(savedProfile)
	})

	// A copy, not LoadPreset: LoadPreset also mutates the shared DefaultTabTheme, and
	// restoring only DefaultTheme leaves the tab height changed for later tests.
	th := *savedTheme
	th.Metrics.MinTouchTarget = theme.TouchTargetAndroid
	theme.SetDefaultTheme(&th)
	meui.SetTouchProfile(true)
	floor := meui.MinTouchTarget()

	h := Load(t, touchAuditQML)
	win := h.Win
	win.SetFrameRect(graphics.NewRect(0, 0, 360, 640))
	win.ValidateLayout()

	checked, tooSmall := 0, 0
	var walk func(w meui.Widget)
	walk = func(w meui.Widget) {
		if w == nil {
			return
		}
		if _, isContainer := w.(meui.Container); !isContainer {
			if meui.IsInteractive(w) {
				checked++
				f := meui.TouchHitFrame(w, w.FrameRect())
				if f.Width < floor || f.Height < floor {
					tooSmall++
					t.Errorf("%T at %v is interactive but its hit rect is %vx%v, under the %v floor",
						w, w.FrameRect(), f.Width, f.Height, floor)
				}
			}
		}
		if c, ok := w.(meui.Container); ok {
			for _, ch := range c.Children() {
				walk(ch)
			}
		}
	}
	walk(win)

	// A vacuous pass would be worse than no test: if the walk finds nothing the
	// assertions above never ran.
	if checked == 0 {
		t.Fatal("the audit found no interactive widgets, so it proves nothing")
	}

	// And the audit states what it EXPECTS to be tappable, so a widget that is
	// interactive but missing from the registry FAILS here instead of being silently
	// skipped. Without this the registry could rot one widget at a time and the test
	// would keep passing while the target shrank to 20px.
	for _, id := range []string{"wButton", "wCheck", "wRadio", "wSwitch",
		"wSlider", "wCombo", "wText", "wSpin"} {
		w := h.Widget(id)
		if !meui.IsInteractive(w) {
			t.Errorf("%s (%T) is a control a finger aims at but is not in the interactive registry, "+
				"so its hit target is never expanded", id, w)
		}
	}
	// The other direction: a static widget must NOT be treated as a target, or it
	// steals the expansion of whatever sits beside it.
	if w := h.Widget("wProgress"); meui.IsInteractive(w) {
		t.Errorf("a ProgressBar is in the interactive registry; expanding it would steal " +
			"hits from a control next to it")
	}

	t.Logf("audited %d interactive widgets, %d under the floor", checked, tooSmall)
}

// interactiveStub stands in for any widget that responds to a click, so the expansion
// can be asserted without constructing a real one.
type interactiveStub struct{ meui.BaseWidget }

func (interactiveStub) HasMouseHandlers() bool { return true }
