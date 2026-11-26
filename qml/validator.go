package qml

import (
	"fmt"
	"strings"
)

func ValidateQML(obj *Object) error {
	if obj == nil {
		return nil
	}

	// Track seen IDs to detect duplicates
	idMap := make(map[string]bool)
	var errors []string

	validateObject(obj, idMap, &errors)

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "\n"))
	}
	return nil
}

func validateObject(obj *Object, idMap map[string]bool, errors *[]string) {
	if obj == nil {
		return
	}

	// Check for duplicate IDs
	if obj.ID != "" {
		if idMap[obj.ID] {
			*errors = append(*errors, fmt.Sprintf("DUPLICATE ID: '%s' is used more than once", obj.ID))
		}
		idMap[obj.ID] = true
	}

	// Check for x,y coordinates on non-layout children of Window
	if obj.Type == "Window" {
		for _, child := range obj.Children {
			if hasXYCoordinates(child) && !isLayoutManager(child.Type) {
				id := child.ID
				if id == "" {
					id = child.Type
				}
				*errors = append(*errors, fmt.Sprintf(
					"SYNTAX ERROR: Widget '%s' (type: %s) has x,y coordinates but is not inside a layout manager.\nFIX: Wrap content in a VStack { } or other layout manager.\nProblematic pattern: Widget as direct child of Window with absolute positioning",
					id, child.Type))
			}
		}
	}

	// Recurse into children
	for _, child := range obj.Children {
		validateObject(child, idMap, errors)
	}
}

func hasXYCoordinates(obj *Object) bool {
	hasX := false
	hasY := false
	for _, prop := range obj.Properties {
		if prop.Name == "x" || strings.HasPrefix(prop.Name, "x") {
			hasX = true
		}
		if prop.Name == "y" || strings.HasPrefix(prop.Name, "y") {
			hasY = true
		}
		if hasX && hasY {
			return true
		}
	}
	return hasX && hasY
}

func isLayoutManager(widgetType string) bool {
	switch widgetType {
	case "VStack", "HStack", "Column", "Row", "Grid", "Flex":
		return true
	}
	return false
}
