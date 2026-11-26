package qml

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/widgets"
)

// A bound widget inside a TabPanel must be the SAME instance the model holds. A tab
// content built twice (once for layout, once for display) leaves the on-screen control
// detached from the model, so SetValue on the model never reaches it. This is the
// regression that followed an app's accounts window; the fixture is inline so the test
// lives with the tree rather than with one application.
func TestATabPanelKeepsTheBoundWidgetItBuilds(t *testing.T) {
	res, err := ParseWithBindings(`
Window {
  width: 400
  height: 300
  VStack {
    anchors.fill: parent
    TabPanel {
      id: tabs
      Layout.fillWidth: true
      Layout.fillHeight: true
      Tab {
        title: "One"
        SpinBox {
          id: txtDailyTargetPerc
          x: 10
          y: 10
          width: 120
          height: 30
        }
      }
      Tab {
        title: "Two"
        SpinBox {
          id: other
          x: 10
          y: 10
          width: 120
          height: 30
        }
      }
    }
  }
}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var m struct {
		TxtDailyTargetPerc *widgets.SpinBox `qml:"txtDailyTargetPerc"`
	}
	if err := res.Widgets.Inject(&m); err != nil {
		t.Fatalf("inject: %v", err)
	}
	if m.TxtDailyTargetPerc == nil {
		t.Fatal("txtDailyTargetPerc is not bound")
	}
	m.TxtDailyTargetPerc.SetValue(77)

	var found []*widgets.SpinBox
	var walk func(w meui.Widget)
	walk = func(w meui.Widget) {
		if w == nil {
			return
		}
		if sb, ok := w.(*widgets.SpinBox); ok {
			found = append(found, sb)
		}
		if tp, ok := w.(interface {
			TabCount() int
			SetActiveTab(int)
		}); ok {
			for i := 0; i < tp.TabCount(); i++ {
				tp.SetActiveTab(i)
				if c, ok := w.(interface{ Children() []meui.Widget }); ok {
					for _, ch := range c.Children() {
						walk(ch)
					}
				}
			}
			tp.SetActiveTab(0)
			return
		}
		if c, ok := w.(interface{ Children() []meui.Widget }); ok {
			for _, ch := range c.Children() {
				walk(ch)
			}
		}
	}
	walk(res.Widget)

	matches := 0
	for _, sb := range found {
		if sb == m.TxtDailyTargetPerc {
			matches++
		}
	}
	if matches == 0 {
		t.Fatal("the model's spin box is not in the displayed tree — the tab content was built twice")
	}
	if matches > 1 {
		t.Fatalf("the model's spin box appears %d times in the tree", matches)
	}

	// A change must reach the window's redraw flag, or the paint pipeline never runs.
	win, ok := res.Widget.(*meui.Window)
	if !ok {
		t.Fatalf("root is %T, want *meui.Window", res.Widget)
	}
	if m.TxtDailyTargetPerc.NeedsRedraw() && !win.NeedsRedraw() {
		t.Fatal("MarkForRedraw did not propagate to the window")
	}
}
