//go:build visual

package behaviour

import (
	"os"
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/widgets"
)

// visualMenuQML replicates the demo's File menu exactly: shortcuts,
// icons, separators, checked items, submenus.
const visualMenuQML = `
Window {
    width: 500
    height: 300
    VStack {
        margins: 0
        MenuBar {
            id: mb
            width: 500
            menus: [
                { title: "File", items: [
                    { label: "Cut", shortcut: "Ctrl+X" },
                    { label: "Copy", shortcut: "Ctrl+C", icon: "document" },
                    { label: "Paste", shortcut: "Ctrl+V", icon: "document" },
                    { separator: true },
                    { label: "Select All", shortcut: "Ctrl+A" },
                    { label: "Delete" },
                    { separator: true },
                    { label: "Word Wrap", checked: true },
                    { label: "Show Line Numbers", checked: false },
                    { separator: true },
                    { label: "Sort By", submenu: [
                        { label: "Name" },
                        { label: "Size" }
                    ]}
                ]}
            ]
        }
        Label { text: "body" }
    }
}
`

// TestVisualMenuBarDemoMenu opens the demo's File menu and asserts the
// dropdown actually rendered: the content area below the bar must change while
// the menu is open and return to the closed frame once it is dismissed.
func TestVisualMenuBarDemoMenu(t *testing.T) {
	// Two SDL windows in one test process segfault on GL re-init; run
	// visual tests one process at a time:
	//   go test ./behaviour/ -run TestVisualMenuBarDemoMenu
	if os.Getenv("BEHAVIOUR_VISUAL_MENU") == "" {
		t.Skip("set BEHAVIOUR_VISUAL_MENU=1 to run (one visual window per process)")
	}
	va := NewVisualApp(t, "TestVisualMenuBarDemoMenu", visualMenuQML, 500, 300)
	mb := va.Reg.Get("mb").(*widgets.MenuBar)

	// The dropdown floats below the bar, over the content.
	var box [4]int32
	var closed []graphics.Color

	va.Run(t, []VisualStep{
		func(va *VisualApp) {
			f := mb.FrameRect()
			top := int32(f.Y + f.Height)
			box = [4]int32{0, top, va.W, va.H - top}
		},
		func(va *VisualApp) {
			// The frame before this step has the menu closed.
			closed = va.Snapshot(box[0], box[1], box[2], box[3])
			va.Click(20, 14) // open File
		},
		va.Frames(1), // the dropdown renders
		func(va *VisualApp) {
			va.AssertRegionChanged(closed, box[0], box[1], box[2], box[3], 1,
				"the content below the bar while the File menu is open")
			va.Click(20, 14) // close
		},
		va.Frames(1),
		func(va *VisualApp) {
			// The dropdown is gone: at most a small residue (hover ink).
			max := int(box[2] * box[3] / 20)
			va.AssertRegionEqual(closed, box[0], box[1], box[2], box[3], max,
				"the content below the bar after dismissing the File menu")
		},
	})
}
