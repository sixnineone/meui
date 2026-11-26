package behaviour

import (
	"testing"

	"github.com/sixnineone/meui/theme"
	"github.com/sixnineone/meui/widgets"
)

const menubarQML = `
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
                    { label: "New", shortcut: "Ctrl+N" },
                    { label: "Open", shortcut: "Ctrl+O" },
                    { label: "Save", shortcut: "Ctrl+S" }
                ]},
                { title: "Edit", items: [
                    { label: "Copy" },
                    { label: "Paste" }
                ]}
            ]
        }
        Label { text: "body" }
    }
}
`

// Clicking the same menu title a second time must CLOSE the dropdown.
func TestMenuBarSecondClickCloses(t *testing.T) {
	h := Load(t, menubarQML)
	mb := h.Widget("mb").(*widgets.MenuBar)
	h.Settle() // initial paint

	var fired int
	mb.SetMenuItemAction("File", "New", func() { fired++ })

	h.Click(20, 14) // open "File"
	h.Settle()      // dropdown popup is created during Paint

	// Click the SAME title again: the popup backdrop must dismiss and
	// the menu must close.
	h.Click(20, 14)
	h.Settle()

	// With the menu closed, clicking where the first item WAS must not
	// fire its action.
	h.Click(35, 45)
	h.Settle()
	if fired != 0 {
		t.Fatalf("menu stayed open: item action fired %d times after close", fired)
	}

	// And re-opening still works.
	h.Click(20, 14)
	h.Settle()
	h.Click(35, 45)
	h.Settle()
	if fired != 1 {
		t.Fatalf("item action after reopen: fired %d times, want 1", fired)
	}
}

// Theme switches rebuild text caches; menu items must still render
// (regression for "first two menu items are always blank").
func TestMenuBarItemsRenderAfterThemeSwitch(t *testing.T) {
	h := Load(t, menubarQML)
	mb := h.Widget("mb").(*widgets.MenuBar)
	h.Settle()

	// Open the menu pre-switch: the item texts build against the theme.
	h.Click(20, 14)
	h.Settle()

	// Close, then force a theme change: fresh theme objects invalidate
	// text caches (themeVersion bumps). This reproduced blank items in
	// the live app when the label text caches were keyed to the old font.
	h.Click(20, 14) // close (second click toggles)
	h.Settle()
	theme.SetDefaultTheme(theme.DarkTheme())
	h.Settle()

	// Re-open: the dropdown items must re-render with the new theme.
	h.Click(20, 14)
	h.Settle()

	// The observable contract: the first item's action still fires —
	// blank-rendered items were also dead items (their hit areas and
	// caches were stale).
	var fired int
	mb.SetMenuItemAction("File", "New", func() { fired++ })
	h.Click(35, 45)
	h.Settle()
	if fired != 1 {
		t.Fatalf("item action after theme switch: fired %d times", fired)
	}
}
