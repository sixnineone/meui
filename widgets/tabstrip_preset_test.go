package widgets

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

// It was opt-in for one commit, because it FAILED and a red suite cannot be committed
// (AGENTS.md) — the defect is P5i11 and the fix is in Paint. It is back in the gate now
// because a test that was red and is green is the most valuable kind: it is a
// reproduction that has become a guard, and it runs against every preset on every
// commit from here on.
func TestATabStripPaintsUnderEveryPreset(t *testing.T) {

	// TestATabStripPaintsUnderEveryPreset was a REPRODUCTION, and it failed.
	//
	// The demo's document pane showed a tab strip that painted its bar and no labels, while
	// the same widget in the behaviour harness painted everything. This narrows it to the
	// PRESET, deterministically: the default theme and `meui-dark` draw; `breeze-dark` (what
	// the demo's LoadAutoPreset picks on this host), `adwaita-dark` and `macos-dark` draw
	// NOTHING AT ALL.
	//
	// Everything else has been excluded by measurement, which is the reason this test is worth
	// more than the screenshot was:
	//
	//   - layout      — post-layout the strip is 560x26 with 3 items
	//   - items       — Count() is 3; the same 3 draw 14000 pixels under the default theme
	//   - contrast    — text #BABEC2 on background #31363B, ample
	//   - font        — the base font is 13px, and setting an explicit tab font changes nothing
	//   - enabled     — true
	//   - visibility  — true
	//   - frame       — 560x26, as laid out
	//
	// So the difference is in what a preset does to the strip's paint path. It is left failing
	// and visible rather than skipped, because a test that is red on purpose is the honest
	// form of "this is not fixed", and it is the thing to start from instead of a screenshot.
	saved := theme.DefaultTheme
	t.Cleanup(func() { theme.SetDefaultTheme(saved) })

	for _, id := range []string{"meui-dark", "breeze-dark", "adwaita-dark", "macos-dark"} {
		if err := theme.LoadPreset(id); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		const w, h = 560, 26
		s := NewTabStrip()
		s.SetItems([]TabStripItem{
			{ID: "a", Label: "tab-A"}, {ID: "b", Label: "tab-B"}, {ID: "c", Label: "tab-C"},
		})
		s.SetSelected(2)
		s.SetFrameRect(graphics.NewRect(0, 0, w, h))
		dev := graphics.NewSoftDevice(w, h)
		s.Paint(graphics.NewCanvas(dev), graphics.NewRect(0, 0, w, h))
		drawn := 0
		for _, p := range dev.Pixels() {
			if p != 0 {
				drawn++
			}
		}
		if drawn == 0 {
			t.Errorf("%s: a 560x26 strip with three items drew NOTHING (enabled=%v visible=%v "+
				"barInk=#%08X textInk=#%08X tabH=%.0f)", id, s.Enabled(), s.Visible(),
				uint32(s.TabTheme.BackgroundInk), uint32(s.TabTheme.TextInk), s.TabTheme.TabHeight)
		} else {
			t.Logf("%-14s drew %d pixels", id, drawn)
		}
	}
}
