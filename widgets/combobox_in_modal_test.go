package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
)

// A ComboBox opens its dropdown by adding a layer to its window's LayerManager,
// and it finds that window by walking the PARENT CHAIN. A widget hosted by a
// layer - everything inside a modal - is not in the window's child tree, so
// without LayerManager.AddLayer pointing the hosted widget at the window its
// chain ended in nothing and the dropdown could never be created. The visible
// symptom was the Settings dialog: the Chart period and Momentum comboboxes
// simply did not open, which read as "the comboboxes are disabled".
//
// This asserts the whole path: the modal content can reach the window, and
// clicking the combo really does create a dropdown layer.
func TestComboBoxInsideModalCanReachItsWindowAndOpen(t *testing.T) {
	win := meui.NewWindow()
	if win == nil || win.LayerManager == nil {
		t.Fatal("NewWindow produced no layer manager")
	}

	content := meui.NewPanel()
	combo := NewComboBox()
	combo.SetOptions([]string{"M1", "ZM1", "ZM5"})
	combo.SetFrameRect(graphics.NewRect(10, 10, 120, 24))
	content.AddChild(combo)

	win.ShowModal(content, 400, 300)

	if w := combo.Window(); w == nil {
		t.Fatal("a layer-hosted widget cannot reach its window: its dropdown layer can never be created")
	} else if w != win {
		t.Fatalf("combo resolved the wrong window: %p, want %p", w, win)
	}

	before := len(win.LayerManager.Layers)
	if !combo.OnMouseDown(5, 5, 0) {
		t.Fatal("click inside the combo frame was not consumed")
	}
	if !combo.Expanded {
		t.Fatal("combobox did not expand on click")
	}
	if got := len(win.LayerManager.Layers); got <= before {
		t.Fatalf("no dropdown layer was added: layers %d -> %d", before, got)
	}
}
