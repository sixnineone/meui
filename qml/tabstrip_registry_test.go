package qml

// TabStrip must be creatable from QML, and must accept the base layout properties, or a
// window that places one cannot be loaded at all. Its items come from Go (they carry an ID
// and a live label), so it deliberately declares no QML properties of its own.

import (
	"testing"

	"github.com/sixnineone/meui/widgets"
)

func TestTabStripLoadsFromQMLAndBinds(t *testing.T) {
	// The real contract: a QML placement of TabStrip must LOAD and BIND by id. This is the
	// path the widget registry is populated on (initializeBuiltinWidgets runs at first
	// parse), which is why the test parses rather than calling CreateWidget directly.
	res, err := ParseWithBindings(`
Window {
  id: root
  width: 400
  height: 200
  VStack {
    TabStrip { id: tabs; height: 26; Layout.fillWidth: true }
  }
}`)
	if err != nil {
		t.Fatalf("a QML placement of TabStrip does not load: %s", err)
	}

	m := struct {
		Tabs *widgets.TabStrip `qml:"tabs"`
	}{}
	if err := res.Widgets.Inject(&m); err != nil {
		t.Fatalf("inject: %s", err)
	}
	if m.Tabs == nil {
		t.Fatal("the TabStrip was not bound to the model by id")
	}

	// And the Go-side API the Accounts window drives it with must work through the binding.
	m.Tabs.SetItems([]widgets.TabStripItem{{ID: "", Label: "All"}, {ID: "g1", Label: "Scalp"}})
	if m.Tabs.Count() != 2 {
		t.Errorf("Count = %d, want 2", m.Tabs.Count())
	}
	if !m.Tabs.SelectID("g1") {
		t.Fatal("SelectID(g1) failed")
	}
	if got := m.Tabs.SelectedLabel(); got != "Scalp" {
		t.Errorf("SelectedLabel = %q, want Scalp", got)
	}
}

// NOTE: this deliberately does NOT assert through `IsValidProperty`. That function has no
// callers anywhere in the toolkit and returns false for every property of every widget —
// including `DataTable.width`, with `width` sitting in baseWidgetProperties. It is dead code
// with a bug in it; testing it would pin the bug rather than the contract. The real proof
// that a QML placement works is loading one, which is what TradeUI's main_qml_test.go does.
//
// The items are built in Go and set with SetItems, so the widget must still be a TabStrip
// after creation — not merely non-nil.
func TestTabStripIsUsableAfterCreation(t *testing.T) {
	w, err := CreateWidget("TabStrip")
	if err != nil {
		t.Fatalf("CreateWidget: %s", err)
	}
	strip, ok := w.(*widgets.TabStrip)
	if !ok {
		t.Fatalf("CreateWidget returned %T", w)
	}
	// The Go-side API the TradeUI window depends on must be reachable through it.
	strip.SetItems([]widgets.TabStripItem{{ID: "", Label: "All"}, {ID: "g1", Label: "Scalp"}})
	if strip.Count() != 2 {
		t.Errorf("Count = %d, want 2", strip.Count())
	}
	if !strip.SelectID("g1") {
		t.Error("SelectID(g1) failed")
	}
	if got := strip.SelectedLabel(); got != "Scalp" {
		t.Errorf("SelectedLabel = %q, want Scalp", got)
	}
}
