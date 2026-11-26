package behaviour

import (
	"testing"

	"github.com/sixnineone/meui/widgets"
)

const basicQML = `
Window {
    width: 500
    height: 500
    VStack {
        margins: 10
        spacing: 8
        CheckBox { id: cb; text: "toggle me" }
        CheckBox { id: cb2; text: "other"; checked: true }
        RadioButton { id: rb1; text: "choice 1"; group: "g" }
        RadioButton { id: rb2; text: "choice 2"; group: "g" }
        ComboBox { id: combo; width: 160; options: ["Alpha", "Beta", "Gamma"] }
        SpinBox { id: spin; width: 120; value: 5; min: 0; max: 10 }
        Button { id: btn; text: "press" }
        Button { id: checkableBtn; text: "toggle btn"; checkable: true }
    }
}
`

func TestCheckBoxClickToggles(t *testing.T) {
	h := Load(t, basicQML)
	cb := h.Widget("cb").(*widgets.CheckBox)
	var fired bool
	cb.SetOnCheckChanged(func(checked bool) { fired = checked })

	cx, cy := Center(cb)
	h.Click(cx, cy)
	if !cb.Checked {
		t.Fatal("click did not check the checkbox")
	}
	if !fired {
		t.Fatal("OnCheckChanged did not fire")
	}
	h.Click(cx, cy)
	if cb.Checked {
		t.Fatal("second click did not uncheck")
	}
}

func TestRadioGroupExclusivity(t *testing.T) {
	h := Load(t, basicQML)
	rb1 := h.Widget("rb1").(*widgets.RadioButton)
	rb2 := h.Widget("rb2").(*widgets.RadioButton)

	c1x, c1y := Center(rb1)
	h.Click(c1x, c1y)
	if !rb1.Selected {
		t.Fatal("radio 1 not selected after click")
	}
	c2x, c2y := Center(rb2)
	h.Click(c2x, c2y)
	if !rb2.Selected {
		t.Fatal("radio 2 not selected after click")
	}
	if rb1.Selected {
		t.Fatal("radio 1 still selected — group exclusivity broken")
	}
}

func TestSpinBoxStepButtons(t *testing.T) {
	h := Load(t, basicQML)
	sp := h.Widget("spin").(*widgets.SpinBox)
	if sp.Value != 5 {
		t.Fatalf("initial value: %v, want 5", sp.Value)
	}
	f := sp.FrameRect()
	// Up button: right edge strip, top half.
	h.Click(f.X+f.Width-12, f.Y+f.Height*0.25)
	if sp.Value != 6 {
		t.Fatalf("after up click: %v, want 6", sp.Value)
	}
	// Down button: right edge strip, bottom half.
	h.Click(f.X+f.Width-12, f.Y+f.Height*0.75)
	if sp.Value != 5 {
		t.Fatalf("after down click: %v, want 5", sp.Value)
	}
	// Clamp at max.
	for i := 0; i < 12; i++ {
		h.Click(f.X+f.Width-12, f.Y+f.Height*0.25)
	}
	if sp.Value != 10 {
		t.Fatalf("clamped at max: %v, want 10", sp.Value)
	}
}

func TestComboBoxOpenSelectClose(t *testing.T) {
	h := Load(t, basicQML)
	combo := h.Widget("combo").(*widgets.ComboBox)

	cx, cy := Center(combo)
	h.Click(cx, cy) // open
	h.Settle()
	if !combo.Expanded {
		t.Fatal("combo did not open")
	}
	// First option sits just below the combo box.
	f := combo.FrameRect()
	h.Click(f.X+40, f.Y+f.Height+15)
	if combo.Expanded {
		t.Fatal("combo still open after option click")
	}
	if combo.SelectedIndex != 0 {
		t.Fatalf("selected index: got %d, want 0", combo.SelectedIndex)
	}
	if got := combo.GetSelectedText(); got != "Alpha" {
		t.Fatalf("selected text: got %q", got)
	}
}

func TestComboBoxClickOutsideCloses(t *testing.T) {
	h := Load(t, basicQML)
	combo := h.Widget("combo").(*widgets.ComboBox)
	cx, cy := Center(combo)
	h.Click(cx, cy)
	if !combo.Expanded {
		t.Fatal("combo did not open")
	}
	h.Click(480, 490) // far outside
	if combo.Expanded {
		t.Fatal("combo still open after clicking outside")
	}
}

func TestButtonOnClickFires(t *testing.T) {
	h := Load(t, basicQML)
	btn := h.Widget("btn").(*widgets.Button)
	var clicks int
	btn.SetOnClicked(func() { clicks++ })
	cx, cy := Center(btn)
	h.Click(cx, cy)
	h.Click(cx, cy)
	if clicks != 2 {
		t.Fatalf("clicks fired: %d, want 2", clicks)
	}
}

func TestCheckableButtonToggles(t *testing.T) {
	h := Load(t, basicQML)
	btn := h.Widget("checkableBtn").(*widgets.Button)
	cx, cy := Center(btn)
	h.Click(cx, cy)
	if !btn.Toggled {
		t.Fatal("checkable button not toggled after click")
	}
	h.Click(cx, cy)
	if btn.Toggled {
		t.Fatal("checkable button still toggled after second click")
	}
}
