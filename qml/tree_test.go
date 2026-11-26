package qml

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sixnineone/meui/flow"
	"github.com/sixnineone/meui/widgets"
)

func TestFindWidgetByID(t *testing.T) {
	tree := WidgetJSON{
		ID:   "root",
		Type: "Window",
		Children: []WidgetJSON{
			{ID: "child1", Type: "Label"},
			{ID: "child2", Type: "Button", Children: []WidgetJSON{
				{ID: "grandchild", Type: "Rectangle"},
			}},
		},
	}

	// Find direct child
	found := FindWidgetByID(tree, "child1")
	if found == nil || found.ID != "child1" {
		t.Error("Failed to find direct child")
	}

	// Find grandchild (recursive search)
	found = FindWidgetByID(tree, "grandchild")
	if found == nil || found.ID != "grandchild" {
		t.Error("Failed to find grandchild")
	}

	// Not found
	found = FindWidgetByID(tree, "nonexistent")
	if found != nil {
		t.Error("Should not find non-existent widget")
	}
}

func TestWidgetJSONMarshal(t *testing.T) {
	tree := WidgetJSON{
		ID:     "test",
		Type:   "Label",
		X:      10,
		Y:      20,
		Width:  100,
		Height: 30,
	}

	data, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Unmarshal and verify
	var decoded WidgetJSON
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if decoded.ID != "test" || decoded.Type != "Label" {
		t.Errorf("Decoded values incorrect: got %+v", decoded)
	}
}

func TestWidgetRegistry(t *testing.T) {
	initializeBuiltinWidgets()

	// Test all built-in widgets can be created
	// Note: Rectangle is handled specially via buildRectangle, not in registry
	widgets := []string{"Label", "Button", "TextField", "CheckBox", "RadioButton",
		"Slider", "ComboBox", "TextArea", "ScrollBar", "SpinBox", "ProgressBar",
		"ListView"}

	for _, name := range widgets {
		w, err := CreateWidget(name)
		if err != nil {
			t.Errorf("Failed to create widget %s: %v", name, err)
		}
		if w == nil {
			t.Errorf("Created widget %s is nil", name)
		}
	}
}

func TestWidgetProperties(t *testing.T) {
	initializeBuiltinWidgets()

	// Label should have text and color properties
	props := GetWidgetProperties("Label")
	if !contains(props, "text") {
		t.Error("Label missing 'text' property")
	}
	if !contains(props, "anchors.fill") {
		t.Error("Label missing 'anchors.fill' (inherited)")
	}

	// Button should have buttonType
	props = GetWidgetProperties("Button")
	if !contains(props, "buttonType") {
		t.Error("Button missing 'buttonType' property")
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func TestInjectModel(t *testing.T) {
	type testModel struct {
		Btn *widgets.Button `qml:"myButton"`
		Lbl *widgets.Label  `qml:"myLabel"`
	}
	r := NewBoundWidgetRegistry()
	r.Register("myButton", widgets.NewButton())
	r.Register("myLabel", widgets.NewLabel())

	var m testModel
	if err := r.Inject(&m); err != nil {
		t.Fatalf("Inject failed: %v", err)
	}
	if m.Btn == nil {
		t.Error("Btn field not populated")
	}
	if m.Lbl == nil {
		t.Error("Lbl field not populated")
	}
}

func TestInjectMissingID(t *testing.T) {
	type testModel struct {
		Btn *widgets.Button `qml:"nonexistent"`
	}
	r := NewBoundWidgetRegistry()
	var m testModel
	if err := r.Inject(&m); err == nil {
		t.Fatal("Expected error for missing QML id, got nil")
	} else if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("Error should mention missing id, got: %v", err)
	}
}

func TestInjectTypeMismatch(t *testing.T) {
	type testModel struct {
		Lbl *widgets.Label `qml:"widget1"`
	}
	r := NewBoundWidgetRegistry()
	r.Register("widget1", widgets.NewButton())
	var m testModel
	if err := r.Inject(&m); err == nil {
		t.Fatal("Expected type mismatch error, got nil")
	}
}

func TestInjectNotPointer(t *testing.T) {
	r := NewBoundWidgetRegistry()
	if err := r.Inject(struct{}{}); err == nil {
		t.Fatal("Expected error for non-pointer model")
	}
}

func TestDuplicateIDs(t *testing.T) {
	r := NewBoundWidgetRegistry()
	r.Register("sameID", widgets.NewButton())
	r.Register("sameID", widgets.NewLabel()) // duplicate
	if err := r.Err(); err == nil {
		t.Fatal("Expected duplicate ID error")
	} else if !strings.Contains(err.Error(), "sameID") {
		t.Errorf("Error should mention 'sameID', got: %v", err)
	}
}

func TestInjectFlowWrapper(t *testing.T) {
	type testModel struct {
		Btn  *widgets.Button `qml:"myBtn"`
		Flow *flow.Button    `qml:"myBtn"`
	}
	r := NewBoundWidgetRegistry()
	r.Register("myBtn", widgets.NewButton())
	r.RegisterFlow("myBtn", &flow.Button{})

	var m testModel
	if err := r.Inject(&m); err != nil {
		t.Fatalf("Inject failed: %v", err)
	}
	if m.Btn == nil {
		t.Error("Button field not populated")
	}
	if m.Flow == nil {
		t.Error("Flow.Button field not populated")
	}
}
