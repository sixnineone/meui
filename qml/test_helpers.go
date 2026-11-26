package qml

import (
	"encoding/json"
	"fmt"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/layout"
	"github.com/sixnineone/meui/widgets"
)

// WidgetJSON represents a widget in the JSON tree dump
type WidgetJSON struct {
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type"`
	X        float32      `json:"x"`
	Y        float32      `json:"y"`
	Width    float32      `json:"width"`
	Height   float32      `json:"height"`
	Debug    string       `json:"debug,omitempty"`
	Anchors  *AnchorJSON  `json:"anchors,omitempty"`
	Children []WidgetJSON `json:"children,omitempty"`
}

// AnchorJSON represents anchor settings on a widget
type AnchorJSON struct {
	Fill     string  `json:"fill,omitempty"`     // parent, or ""
	CenterIn string  `json:"centerIn,omitempty"` // parent, or ""
	Left     string  `json:"left,omitempty"`
	Right    string  `json:"right,omitempty"`
	Top      string  `json:"top,omitempty"`
	Bottom   string  `json:"bottom,omitempty"`
	Margins  float32 `json:"margins,omitempty"`
}

// DumpWidgetTree returns JSON representation of the widget tree with coordinates
func DumpWidgetTree(widget meui.Widget) string {
	tree := buildTreeJSON(widget)
	data, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error())
	}
	return string(data)
}

// buildTreeJSON recursively builds JSON tree from widget
func buildTreeJSON(widget meui.Widget) WidgetJSON {
	node := WidgetJSON{}

	if widget == nil {
		node.Type = "nil"
		return node
	}

	// Get frame rect for position/size
	if f, ok := widget.(interface{ FrameRect() graphics.Rect }); ok {
		rect := f.FrameRect()
		node.X = rect.X
		node.Y = rect.Y
		node.Width = rect.Width
		node.Height = rect.Height
	}

	// Get ID if available
	if w, ok := widget.(interface{ ID() string }); ok {
		node.ID = w.ID()
	}

	// Get debug name if available
	if w, ok := widget.(interface{ DebugName() string }); ok {
		node.Debug = w.DebugName()
	}

	// Get widget type name
	node.Type = getWidgetTypeName(widget)

	// Get anchor settings if available
	if ga, ok := widget.(interface{ GetAnchors() interface{} }); ok {
		if anchors := ga.GetAnchors(); anchors != nil {
			if a, ok := anchors.(*layout.Anchors); ok {
				node.Anchors = &AnchorJSON{}
				if a.Fill != nil {
					// Fill is set to parent widget
					node.Anchors.Fill = "parent"
				}
				if a.CenterIn != nil {
					node.Anchors.CenterIn = "parent"
				}
				// Note: Left, Right, Top, Bottom would need more work to resolve
				node.Anchors.Margins = a.Margins
			}
		}
	}

	// Get children
	if c, ok := widget.(interface{ Children() []meui.Widget }); ok {
		children := c.Children()
		if len(children) > 0 {
			node.Children = make([]WidgetJSON, 0, len(children))
			for _, child := range children {
				node.Children = append(node.Children, buildTreeJSON(child))
			}
		}
	}

	return node
}

// getWidgetTypeName returns a human-readable type name for a widget
func getWidgetTypeName(widget meui.Widget) string {
	switch widget.(type) {
	case *meui.Window:
		return "Window"
	case *meui.Panel:
		return "Rectangle"
	case *widgets.Label:
		return "Label"
	case *widgets.Button:
		return "Button"
	case *widgets.TextField:
		return "TextField"
	case *widgets.CheckBox:
		return "CheckBox"
	case *widgets.RadioButton:
		return "RadioButton"
	case *widgets.Slider:
		return "Slider"
	case *widgets.ComboBox:
		return "ComboBox"
	case *widgets.TextArea:
		return "TextArea"
	case *widgets.ScrollBar:
		return "ScrollBar"
	case *widgets.SpinBox:
		return "SpinBox"
	case *widgets.ProgressBar:
		return "ProgressBar"
	case *widgets.ListView:
		return "ListView"
	case *widgets.TabPanel:
		return "TabPanel"
	case *widgets.CodeEditor:
		return "CodeEditor"
	case *widgets.ScrollPane:
		return "ScrollPane"
	case *widgets.Switch:
		return "Switch"
	case *widgets.GroupBox:
		return "GroupBox"
	case *widgets.StatusBar:
		return "StatusBar"
	case *widgets.ToolBar:
		return "ToolBar"
	case *widgets.Calendar:
		return "Calendar"
	case *widgets.ColorPicker:
		return "ColorPicker"
	case *widgets.DatePicker:
		return "DatePicker"
	case *widgets.ColorField:
		return "ColorField"
	case *widgets.TimePicker:
		return "TimePicker"
	case *widgets.MenuBar:
		return "MenuBar"
	case *widgets.ModalPopupPanel:
		return "Dialog"
	case *widgets.SplitView:
		return "SplitView"
	case *widgets.Tooltip:
		return "ToolTip"
	case *widgets.ContextMenu:
		return "ContextMenu"
	case *widgets.Tree:
		return "Tree"
	case *widgets.Table:
		return "Table"
	case *widgets.DataTable:
		return "DataTable"
	case *widgets.MarkdownView:
		return "MarkdownView"
	case *widgets.MdiContainer:
		return "MdiContainer"
	case *widgets.MdiWindow:
		return "MdiWindow"
	case *widgets.StackPanel:
		return "StackPanel"
	case *widgets.DockPanel:
		return "DockPanel"
	default:
		return "Unknown"
	}
}

// GetWidgetPosition returns the frame rect of a widget by ID from the registry
func GetWidgetPosition(registry *BoundWidgetRegistry, id string) (x, y, w, h float32, found bool) {
	if registry == nil {
		return 0, 0, 0, 0, false
	}

	widget := registry.Get(id)
	if widget == nil {
		return 0, 0, 0, 0, false
	}

	if f, ok := widget.(interface{ FrameRect() graphics.Rect }); ok {
		rect := f.FrameRect()
		return rect.X, rect.Y, rect.Width, rect.Height, true
	}

	return 0, 0, 0, 0, false
}

// FindWidgetByID searches the JSON tree for a widget by ID
func FindWidgetByID(tree WidgetJSON, id string) *WidgetJSON {
	if tree.ID == id {
		return &tree
	}
	for _, child := range tree.Children {
		if found := FindWidgetByID(child, id); found != nil {
			return found
		}
	}
	return nil
}

// AssertPosition checks if a widget is at the expected position (for tests)
func AssertPosition(registry *BoundWidgetRegistry, id string, expectedX, expectedY, expectedW, expectedH float32, tolerance float32) string {
	x, y, w, h, found := GetWidgetPosition(registry, id)
	if !found {
		return fmt.Sprintf("FAIL: widget not found: %s", id)
	}

	if distance(x, expectedX) > tolerance || distance(y, expectedY) > tolerance ||
		distance(w, expectedW) > tolerance || distance(h, expectedH) > tolerance {
		return fmt.Sprintf("FAIL: %s position (%.1f,%.1f,%.1f,%.1f) != expected (%.1f,%.1f,%.1f,%.1f)",
			id, x, y, w, h, expectedX, expectedY, expectedW, expectedH)
	}

	return "PASS"
}

// distance returns absolute difference between two float32 values
func distance(a, b float32) float32 {
	if a > b {
		return a - b
	}
	return b - a
}
