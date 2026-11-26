package meui

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
)

// chromeStub stands in for the title bar's node: a widget whose coordinates
// start at the window origin, above the content tree.
type chromeStub struct {
	BaseWidget
	child Widget
}

func (c *chromeStub) IsWindowChrome() bool { return true }

type contentStub struct {
	BaseWidget
	child Widget
}

func TestLayerOriginFollowsTheWidgetsTree(t *testing.T) {
	w := NewWindow()
	w.SetContentOrigin(0, 28) // a 28px custom title bar

	content := &contentStub{}
	content.SetFrameRect(graphics.NewRect(0, 40, 100, 20))

	chrome := &chromeStub{}
	chrome.SetFrameRect(graphics.NewRect(0, 0, 100, 28))
	inChrome := &contentStub{}
	inChrome.SetParent(chrome)
	inChrome.SetFrameRect(graphics.NewRect(10, 4, 80, 20))
	chrome.child = inChrome

	// A content widget: layers are placed in content space, which is where the
	// overlay paints them.
	if pt := w.layerOriginFor(content); pt.X != 0 || pt.Y != 0 {
		t.Errorf("content widget origin = %v, want (0,0)", pt)
	}
	// A widget inside the chrome: content space is a title bar *above* window
	// space, so the origin is negative.
	if pt := w.layerOriginFor(inChrome); pt.X != 0 || pt.Y != -28 {
		t.Errorf("chrome widget origin = %v, want (0,-28)", pt)
	}
	// Unknown/root widgets default to content space.
	if pt := w.layerOriginFor(nil); pt.X != 0 || pt.Y != 0 {
		t.Errorf("nil widget origin = %v, want (0,0)", pt)
	}
}

// AddLayerFor takes a widget-local point; the window adds the widget's
// position in its tree and the tree's origin, so a chrome widget's popup lands
// under the chrome and a content widget's lands under the content.
func TestAddLayerForConvertsFromWidgetLocal(t *testing.T) {
	w := NewWindow()
	w.SetContentOrigin(0, 28)

	content := &contentStub{}
	content.SetFrameRect(graphics.NewRect(12, 100, 200, 26))
	w.LayerManager = NewLayerManager()
	layer := w.AddLayerFor(content, 0, 26, nil)
	if layer == nil {
		t.Fatal("no layer for a content widget")
	}
	if layer.X != 12 || layer.Y != 126 {
		t.Errorf("content popup at (%v,%v), want (12,126)", layer.X, layer.Y)
	}

	chrome := &chromeStub{}
	chrome.SetFrameRect(graphics.NewRect(0, 0, 100, 28))
	inChrome := &contentStub{}
	inChrome.SetParent(chrome)
	inChrome.SetFrameRect(graphics.NewRect(300, 2, 190, 22))
	layer = w.AddLayerFor(inChrome, 0, 22, nil)
	if layer == nil {
		t.Fatal("no layer for a chrome widget")
	}
	// Bar-local (300, 24) is window y 24, i.e. content y 24-28 = -4: the popup
	// starts just above the content origin, right under the combo.
	if layer.X != 300 || layer.Y != -4 {
		t.Errorf("chrome popup at (%v,%v), want (300,-4)", layer.X, layer.Y)
	}
}
