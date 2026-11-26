package qml

import (
	"fmt"

	"github.com/sixnineone/meui"
)

type WidgetFactory func() meui.Widget

type WidgetEntry struct {
	Name        string
	Factory     WidgetFactory
	FlowWrapper func() interface{} // returns a flow wrapper struct pointer
	Properties  []string
}

var baseWidgetProperties = []string{
	"width", "height",
	"anchors.fill", "anchors.centerIn",
	"anchors.left", "anchors.right", "anchors.top", "anchors.bottom",
	"anchors.horizontalCenter", "anchors.verticalCenter",
	"anchors.margins", "anchors.leftMargin", "anchors.rightMargin",
	"anchors.topMargin", "anchors.bottomMargin",
	"anchors.horizontalCenterOffset", "anchors.verticalCenterOffset",
	"Layout.fillWidth", "Layout.fillHeight",
	"Layout.minimumWidth", "Layout.maximumWidth",
	"Layout.minimumHeight", "Layout.maximumHeight",
	"Layout.preferredWidth", "Layout.preferredHeight",
	"Layout.leftMargin", "Layout.rightMargin",
	"Layout.topMargin", "Layout.bottomMargin",
	"Layout.alignment",
	"enabled", "visible", "opacity", "z",
}

var widgetRegistry = make(map[string]*WidgetEntry)

func RegisterWidget(name string, factory WidgetFactory, properties []string) {
	widgetRegistry[name] = &WidgetEntry{
		Name:       name,
		Factory:    factory,
		Properties: properties,
	}
}

// RegisterFlowWidget registers a widget with a flow wrapper for reflection-based property setting.
func RegisterFlowWidget(name string, factory WidgetFactory, flowWrapper func() interface{}, properties []string) {
	widgetRegistry[name] = &WidgetEntry{
		Name:        name,
		Factory:     factory,
		FlowWrapper: flowWrapper,
		Properties:  properties,
	}
}

// SetFlowWrapper attaches a flow wrapper factory to an already-registered widget.
func SetFlowWrapper(name string, fw func() interface{}) {
	if entry, ok := widgetRegistry[name]; ok {
		entry.FlowWrapper = fw
	}
}

func CreateWidget(name string) (meui.Widget, error) {
	entry, ok := widgetRegistry[name]
	if !ok {
		return nil, fmt.Errorf("unknown widget type: %s", name)
	}
	return entry.Factory(), nil
}

// GetEntry returns the WidgetEntry for a given type name.
func GetEntry(name string) *WidgetEntry {
	return widgetRegistry[name]
}

func GetWidgetProperties(name string) []string {
	entry, ok := widgetRegistry[name]
	if !ok {
		return nil
	}
	return mergeProperties(baseWidgetProperties, entry.Properties)
}

func IsValidProperty(widgetType, propName string) bool {
	props := GetWidgetProperties(widgetType)
	for _, p := range props {
		if p == propName {
			return true
		}
	}
	return false
}

func GetRegisteredWidgets() []string {
	names := make([]string, 0, len(widgetRegistry))
	for name := range widgetRegistry {
		names = append(names, name)
	}
	return names
}

func mergeProperties(base, specific []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(base)+len(specific))
	for _, p := range base {
		if !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}
	for _, p := range specific {
		if !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}
	return result
}
