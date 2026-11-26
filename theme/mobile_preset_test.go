package theme

import "testing"

// TestMobilePresetsReachTheTouchFloor is the assertion that matters for Phase A: a
// mobile preset whose control height is under the platform's own published floor is
// not a mobile preset. The desktop presets are asserted to stay BELOW it, because a
// 48px button on a desktop is wrong in the other direction and would mean the metric
// had leaked into every preset.
func TestMobilePresetsReachTheTouchFloor(t *testing.T) {
	for _, c := range []struct {
		id     string
		floor  float32
		height float32
	}{
		{"android-dark", TouchTargetAndroid, TouchTargetAndroid},
		{"android-light", TouchTargetAndroid, TouchTargetAndroid},
		{"ios-dark", TouchTargetIOS, TouchTargetIOS},
		{"ios-light", TouchTargetIOS, TouchTargetIOS},
	} {
		build, ok := presetBuilders[c.id]
		if !ok {
			t.Fatalf("%s: preset not registered", c.id)
		}
		th := build()
		if th.Metrics.MinTouchTarget != c.floor {
			t.Errorf("%s: MinTouchTarget = %v, want %v", c.id, th.Metrics.MinTouchTarget, c.floor)
		}
		if th.Metrics.ControlHeight < c.floor {
			t.Errorf("%s: ControlHeight %v is under the %v floor",
				c.id, th.Metrics.ControlHeight, c.floor)
		}
		if th.Metrics.ListRowHeight < c.floor {
			t.Errorf("%s: ListRowHeight %v is under the %v floor",
				c.id, th.Metrics.ListRowHeight, c.floor)
		}
		// The platform must be recorded, or AutoPresetID cannot find it.
		if th.Meta.Platform != PlatformAndroid && th.Meta.Platform != PlatformIOS {
			t.Errorf("%s: Meta.Platform = %q, want a mobile family", c.id, th.Meta.Platform)
		}
	}
}

// TestDesktopPresetsKeepTheirDensity guards the other direction: the touch floor is a
// mobile metric and must not have been applied to the desktop families, whose whole
// character is that they are compact.
func TestDesktopPresetsKeepTheirDensity(t *testing.T) {
	for _, id := range []string{"meui-dark", "macos-dark", "windows11-dark", "adwaita-dark"} {
		build, ok := presetBuilders[id]
		if !ok {
			t.Fatalf("%s: preset not registered", id)
		}
		th := build()
		if th.Metrics.MinTouchTarget != 0 {
			t.Errorf("%s: MinTouchTarget = %v; a pointer-driven desktop has no touch floor",
				id, th.Metrics.MinTouchTarget)
		}
		if th.Metrics.ControlHeight >= TouchTargetAndroid {
			t.Errorf("%s: ControlHeight %v has grown to a touch size",
				id, th.Metrics.ControlHeight)
		}
	}
}
