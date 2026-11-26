package qml

import (
	"testing"

	widgets "github.com/sixnineone/meui/widgets"
)

// TestHStackButtonWidthFromQML verifies that an explicit `width` on Button
// elements survives the QML build path AND the button's preferred-size floor —
// regression guard for the terminal quick-set buttons whose width was ignored
// (Button.PreferredDimensions used to floor every button at 80px, so the
// HStack's max(natural, preferred, min) overrode any narrower explicit width).
func TestHStackButtonWidthFromQML(t *testing.T) {
	src := `
Window {
  width: 900
  height: 600
  HStack {
    spacing: 4
    Button { id: qb3;  text: "3";  width: 36; height: 24 }
    Button { id: qb20; text: "20"; width: 36; height: 24 }
    Button { id: qauto; text: "AUTO" }
  }
}
`
	res, err := ParseWithBindings(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	m := struct {
		Qb3   *widgets.Button `qml:"qb3"`
		Qb20  *widgets.Button `qml:"qb20"`
		Qauto *widgets.Button `qml:"qauto"`
	}{}
	if err := res.Widgets.Inject(&m); err != nil {
		t.Fatalf("inject: %v", err)
	}
	if m.Qb3 == nil || m.Qb20 == nil || m.Qauto == nil {
		t.Fatalf("buttons not bound: %+v", m)
	}
	if w := m.Qb3.Width(); w != 36 {
		t.Errorf("qb3 width = %v, want 36", w)
	}
	if w := m.Qb20.Width(); w != 36 {
		t.Errorf("qb20 width = %v, want 36", w)
	}
	// Preferred dimensions must honor the explicit size (these feed the
	// HStack's max(natural, preferred, minimum) width computation).
	pw, ph := m.Qb3.PreferredDimensions(m.Qb3, 800, 600)
	if pw != 36 || ph != 24 {
		t.Errorf("qb3 preferred = %vx%v, want 36x24", pw, ph)
	}
	mw, mh := m.Qb3.MinimumDimensions()
	if mw != 36 || mh != 24 {
		t.Errorf("qb3 minimum = %vx%v, want 36x24", mw, mh)
	}
	// A button WITHOUT an explicit width keeps the text-based floor (≥80).
	pw2, _ := m.Qauto.PreferredDimensions(m.Qauto, 800, 600)
	if pw2 < 80 {
		t.Errorf("qauto preferred width = %v, want ≥80 (no explicit size)", pw2)
	}
}
