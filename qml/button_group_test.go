package qml

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/widgets"
)

func TestButtonGroupPropertiesApplied(t *testing.T) {
	qml := `
Window {
  id: root
  width: 400
  height: 300

  Row {
    spacing: 8

    Button {
      text: "Option A"
      width: 100
      height: 30
      groupID: "btnGroup1"
      selected: true
    }

    Button {
      text: "Option B"
      width: 100
      height: 30
      groupID: "btnGroup1"
    }

    Button {
      text: "Option C"
      width: 100
      height: 30
      groupID: "btnGroup1"
    }
  }
}
`
	widget, err := Convert(qml)
	if err != nil {
		t.Fatalf("Failed to convert QML: %v", err)
	}

	var buttons []*widgets.Button
	var findButtons func(w interface{})
	findButtons = func(w interface{}) {
		if btn, ok := w.(*widgets.Button); ok {
			buttons = append(buttons, btn)
		}
		if container, ok := w.(interface{ Children() []interface{} }); ok {
			for _, child := range container.Children() {
				findButtons(child)
			}
		}
		if container, ok := w.(interface{ ChildrenI() []interface{} }); ok {
			for _, child := range container.ChildrenI() {
				findButtons(child)
			}
		}
	}
	findButtons(widget)

	if len(buttons) != 3 {
		t.Fatalf("Expected 3 buttons, got %d", len(buttons))
	}

	if !buttons[0].Selected {
		t.Error("Button 0 should be selected")
	}
	if buttons[0].GroupID != "btnGroup1" {
		t.Errorf("Button 0 groupID should be 'btnGroup1', got '%s'", buttons[0].GroupID)
	}

	for i := 1; i < 3; i++ {
		if buttons[i].Selected {
			t.Errorf("Button %d should not be selected", i)
		}
		if buttons[i].GroupID != "btnGroup1" {
			t.Errorf("Button %d groupID should be 'btnGroup1', got '%s'", i, buttons[i].GroupID)
		}
	}

	for i, btn := range buttons {
		if !btn.Enabled() {
			t.Errorf("Button %d should be enabled", i)
		}
	}
}

func TestButtonGroupExclusiveSelection(t *testing.T) {
	btn1 := widgets.NewButton().SetText("A").SetGroupID("grp1")
	btn2 := widgets.NewButton().SetText("B").SetGroupID("grp1")
	btn3 := widgets.NewButton().SetText("C").SetGroupID("grp1")

	btn1.SetFrameRect(graphics.NewRect(0, 0, 100, 30))
	btn2.SetFrameRect(graphics.NewRect(0, 0, 100, 30))
	btn3.SetFrameRect(graphics.NewRect(0, 0, 100, 30))

	if btn1.Selected || btn2.Selected || btn3.Selected {
		t.Error("No button should be selected initially")
	}

	btn1.OnMouseDown(50, 15, 0)
	btn1.OnMouseUp(50, 15, 0)
	if !btn1.Selected {
		t.Error("btn1 should be selected after click")
	}
	if btn2.Selected || btn3.Selected {
		t.Error("btn2 and btn3 should not be selected")
	}

	btn2.OnMouseDown(50, 15, 0)
	btn2.OnMouseUp(50, 15, 0)
	if btn1.Selected {
		t.Error("btn1 should be deselected after clicking btn2")
	}
	if !btn2.Selected {
		t.Error("btn2 should be selected after click")
	}
	if btn3.Selected {
		t.Error("btn3 should not be selected")
	}

	btn2.OnMouseDown(50, 15, 0)
	btn2.OnMouseUp(50, 15, 0)
	if !btn2.Selected {
		t.Error("btn2 should remain selected after clicking itself")
	}
}

func TestRadioButtonGroupExclusiveSelection(t *testing.T) {
	rb1 := widgets.NewRadioButton().SetText("R1").SetGroupID("rgrp1")
	rb2 := widgets.NewRadioButton().SetText("R2").SetGroupID("rgrp1")
	rb3 := widgets.NewRadioButton().SetText("R3").SetGroupID("rgrp1")

	rb1.SetFrameRect(graphics.NewRect(0, 0, 100, 30))
	rb2.SetFrameRect(graphics.NewRect(0, 0, 100, 30))
	rb3.SetFrameRect(graphics.NewRect(0, 0, 100, 30))

	rb1.OnMouseDown(50, 15, 0)
	if !rb1.Selected {
		t.Error("rb1 should be selected after click")
	}

	rb2.OnMouseDown(50, 15, 0)
	if rb1.Selected {
		t.Error("rb1 should be deselected after clicking rb2")
	}
	if !rb2.Selected {
		t.Error("rb2 should be selected after click")
	}

	rb2.OnMouseDown(50, 15, 0)
	if !rb2.Selected {
		t.Error("rb2 should remain selected after clicking itself")
	}
}

func TestRadioButtonGroupPropertiesApplied(t *testing.T) {
	qml := `
Window {
  id: root
  width: 400
  height: 300

  Column {
    spacing: 8

    RadioButton {
      text: "Radio 1"
      groupID: "radioGroup1"
      selected: true
    }

    RadioButton {
      text: "Radio 2"
      groupID: "radioGroup1"
    }

    RadioButton {
      text: "Radio 3"
      groupID: "radioGroup1"
    }
  }
}
`
	widget, err := Convert(qml)
	if err != nil {
		t.Fatalf("Failed to convert QML: %v", err)
	}

	var radios []*widgets.RadioButton
	var findRadios func(w interface{})
	findRadios = func(w interface{}) {
		if rb, ok := w.(*widgets.RadioButton); ok {
			radios = append(radios, rb)
		}
		if container, ok := w.(interface{ Children() []interface{} }); ok {
			for _, child := range container.Children() {
				findRadios(child)
			}
		}
		if container, ok := w.(interface{ ChildrenI() []interface{} }); ok {
			for _, child := range container.ChildrenI() {
				findRadios(child)
			}
		}
	}
	findRadios(widget)

	if len(radios) != 3 {
		t.Fatalf("Expected 3 radio buttons, got %d", len(radios))
	}

	if !radios[0].Selected {
		t.Error("Radio 0 should be selected")
	}
	if radios[0].GroupID != "radioGroup1" {
		t.Errorf("Radio 0 groupID should be 'radioGroup1', got '%s'", radios[0].GroupID)
	}

	for i := 1; i < 3; i++ {
		if radios[i].Selected {
			t.Errorf("Radio %d should not be selected", i)
		}
		if radios[i].GroupID != "radioGroup1" {
			t.Errorf("Radio %d groupID should be 'radioGroup1', got '%s'", i, radios[i].GroupID)
		}
	}

	for i, rb := range radios {
		if !rb.Enabled() {
			t.Errorf("Radio %d should be enabled", i)
		}
	}
}
