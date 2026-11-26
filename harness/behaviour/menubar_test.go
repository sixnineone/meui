package behaviour

import (
	"testing"

	"github.com/sixnineone/meui/widgets"
)

const menuQML = `
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
                    { label: "Open", shortcut: "Ctrl+O" }
                ]},
                { title: "Edit", items: [
                    { label: "Copy" }
                ]}
            ]
        }
        Label { text: "body" }
    }
}
`

func TestMenuBarOpenAndItemAction(t *testing.T) {
	h := Load(t, menuQML)
	mb := h.Widget("mb").(*widgets.MenuBar)
	if mb.MenuCount() != 2 {
		t.Fatalf("menu count: %d, want 2", mb.MenuCount())
	}

	var fired string
	ok := mb.SetMenuItemAction("File", "New", func() { fired = "New" })
	if !ok {
		t.Fatal("SetMenuItemAction(File, New) returned false")
	}

	// Click the "File" title: menus start at x=8, each title width =
	// SimpleWidth(title) + 16. "File" ≈ 24px wide, so the title spans
	// roughly [8, 48].
	h.Click(20, 14)
	h.Settle() // the dropdown popup is created during Paint

	// The dropdown popup opens below the title; its first item sits
	// within ~30px of the menu bar.
	h.Click(35, 45)

	if fired != "New" {
		t.Fatalf("menu item action did not fire (got %q)", fired)
	}
}

func TestMenuBarMenuItemChecked(t *testing.T) {
	h := Load(t, menuQML)
	mb := h.Widget("mb").(*widgets.MenuBar)
	if !mb.SetMenuItemChecked("File", "New", true) {
		t.Fatal("SetMenuItemChecked returned false")
	}
	// No public getter — the call returning true exercises the lookup.
}
