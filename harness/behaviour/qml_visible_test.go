package behaviour

import "testing"

// TestQMLVisibleReachesWidgetsWithoutAFlowWrapper pins a loader gap the mobile widgets found the
// hard way. `visible` and `enabled` are COMMON properties — every widget has them — but they were
// applied only through a widget's FLOW WRAPPER (`flow.ApplyLayout`), and only 34 widgets have one.
// For the mobile set (Snackbar, TopAppBar, NavigationBar/Rail, InsetSpacer, BottomSheet),
// `visible: false` in QML was silently ignored: the widget stayed a live hit target.
//
// That is not cosmetic. The demo's `Snackbar { visible: false }` sat in the shell's flow and took
// every press in its band, so a touch drag over the Widgets page's lower half landed on the hidden
// Snackbar and the ScrollPane never scrolled — which read as "the pane does not scroll".
func TestQMLVisibleReachesWidgetsWithoutAFlowWrapper(t *testing.T) {
	h := Load(t, `Window { width: 300; height: 400
		TopAppBar { id: bar; visible: false }
		NavigationBar { id: nav; visible: false }
		BottomSheet { id: sheet; visible: false }
		Label { id: label; text: "x"; visible: false }
	}`)
	for _, id := range []string{"bar", "nav", "sheet", "label"} {
		w := h.Widget(id)
		if w == nil {
			t.Fatalf("%s was not built", id)
		}
		v, ok := w.(interface{ Visible() bool })
		if !ok {
			t.Fatalf("%s (%T) has no Visible()", id, w)
		}
		if v.Visible() {
			t.Errorf("%s: `visible: false` in QML was ignored, so it is a live hit target", id)
		}
	}
}

// TestQMLEnabledReachesWidgetsWithoutAFlowWrapper is the same gap for `enabled`: a control that
// the author disabled in markup must not be clickable.
func TestQMLEnabledReachesWidgetsWithoutAFlowWrapper(t *testing.T) {
	h := Load(t, `Window { width: 300; height: 400
		NavigationBar { id: nav; enabled: false }
		TopAppBar { id: bar; enabled: false }
	}`)
	for _, id := range []string{"nav", "bar"} {
		w := h.Widget(id)
		if w == nil {
			t.Fatalf("%s was not built", id)
		}
		e, ok := w.(interface{ Enabled() bool })
		if !ok {
			t.Fatalf("%s (%T) has no Enabled()", id, w)
		}
		if e.Enabled() {
			t.Errorf("%s: `enabled: false` in QML was ignored, so it is still clickable", id)
		}
	}
}
