//go:build visual

package behaviour

import (
	"fmt"
	"testing"
)

// TestFontSheetQuality renders the alphabet at a range of sizes through
// the live text pipeline and captures the framebuffer — the repeatable
// text-sharpness inspection probe. The PNG lands in
// behaviour/out/fontsheet_quality/step_00.png.
func TestFontSheetQuality(t *testing.T) {
	sizes := []int{6, 7, 8, 9, 10, 11, 12, 14, 16, 18, 20, 24}
	sample := "ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmnopqrstuvwxyz 0123456789 !@#$%&*()+-=[]{};:,.<>?/~"
	labels := ""
	for _, s := range sizes {
		labels += fmt.Sprintf("Label { text: \"size %d — %s\"; font.pointSize: %d }\n", s, sample, s)
	}
	labels += "Label { text: \"il1| I l L 0O oQ 5S 2Z 8B rn m wv vx ., :: ;; -- !! ??\"; font.pointSize: 11 }\n"
	labels += "Label { text: \"HxEx linker — façade naïve ñ — em—dash · middle·dot …\"; font.pointSize: 11 }\n"

	qmlSrc := fmt.Sprintf(`Window { width: 1000; height: 560
		VStack { anchors.fill: parent; margins: 10; spacing: 10
%s
		}
	}`, labels)

	va := NewVisualApp(t, "fontsheet_quality", qmlSrc, 1000, 560)
	va.Run(t, []VisualStep{
		func(va *VisualApp) {}, // idle frame
		func(va *VisualApp) {
			// The sheet rendered: the label column is not one flat colour.
			va.AssertRegionHasContent(10, 10, va.W-20, va.H-20, "the font sheet")
		},
	})
}
