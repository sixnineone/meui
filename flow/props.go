package flow

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// ApplyProperties sets struct fields by their qml tag from a property map.
// wrapper must be a pointer to a struct (e.g. &Button{}).
// The qml tag on each field determines which property name it maps to.
// Properties prefixed with "anchors." or "Layout." are skipped — handled by ApplyLayout.
func ApplyProperties(props map[string]interface{}, wrapper interface{}) error {
	val := reflect.ValueOf(wrapper)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("wrapper must be a pointer to struct")
	}
	val = val.Elem()
	typ := val.Type()

	// Build a map of qmlTag → field index
	type fieldInfo struct {
		idx int
		isW bool // true if this field is the embedded W
	}
	tagToField := make(map[string]fieldInfo)

	// Walk all fields (including embedded W)
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		tag := f.Tag.Get("qml")
		if tag == "" || tag == "-" {
			continue
		}
		// Check if this is the embedded W (has "ID" field)
		isW := f.Type.Kind() == reflect.Struct
		if isW {
			if _, found := f.Type.FieldByName("ID"); !found {
				isW = false
			}
		}
		tagToField[tag] = fieldInfo{idx: i, isW: isW}
	}

	// If the struct embeds W, also index W's fields
	if wField, ok := typ.FieldByName("W"); ok {
		wType := wField.Type
		for i := 0; i < wType.NumField(); i++ {
			f := wType.Field(i)
			tag := f.Tag.Get("qml")
			if tag == "" || tag == "-" {
				continue
			}
			tagToField[tag] = fieldInfo{idx: i, isW: true}
		}
	}

	for propName, propValue := range props {
		// Skip anchors.* and Layout.* — handled by ApplyLayout
		if strings.HasPrefix(propName, "anchors.") || strings.HasPrefix(propName, "Layout.") {
			continue
		}

		if os.Getenv("MEUI_LAYOUT_DEBUG") != "" && propName == "enabled" {
			fmt.Printf("[applyProps] enabled=%v on wrapper\n", propValue)
		}
		fi, ok := tagToField[propName]
		if !ok {
			continue // unknown property, skip
		}

		var field reflect.Value
		if fi.isW {
			field = val.FieldByName("W").Field(fi.idx)
		} else {
			field = val.Field(fi.idx)
		}

		if !field.CanSet() {
			continue
		}

		converted, err := convertValue(propValue, field.Type())
		if err != nil {
			continue
		}
		field.Set(converted)
		if propName == "enabled" {
			if es := val.FieldByName("EnabledSet"); es.IsValid() && es.CanSet() {
				es.SetBool(true)
			}
		}
		if propName == "visible" {
			if vs := val.FieldByName("VisibleSet"); vs.IsValid() && vs.CanSet() {
				vs.SetBool(true)
			}
		}
	}
	return nil
}

// convertValue converts a QML value (float64, string, bool, []interface{})
// to the target Go type.
func convertValue(val interface{}, target reflect.Type) (reflect.Value, error) {
	if val == nil {
		return reflect.Zero(target), nil
	}

	// Direct match
	rv := reflect.ValueOf(val)
	if rv.Type() == target {
		return rv, nil
	}

	// Numeric conversions
	switch target.Kind() {
	case reflect.Float32:
		switch v := val.(type) {
		case float64:
			return reflect.ValueOf(float32(v)), nil
		case int:
			return reflect.ValueOf(float32(v)), nil
		case int64:
			return reflect.ValueOf(float32(v)), nil
		}
	case reflect.Float64:
		switch v := val.(type) {
		case float32:
			return reflect.ValueOf(float64(v)), nil
		case int:
			return reflect.ValueOf(float64(v)), nil
		case int64:
			return reflect.ValueOf(float64(v)), nil
		}
	case reflect.Int:
		switch v := val.(type) {
		case float64:
			return reflect.ValueOf(int(v)), nil
		case int64:
			return reflect.ValueOf(int(v)), nil
		}
	case reflect.String:
		if s, ok := val.(string); ok {
			return reflect.ValueOf(s), nil
		}
		// Try fmt.Sprintf for non-string types
		return reflect.ValueOf(fmt.Sprintf("%v", val)), nil
	case reflect.Bool:
		if b, ok := val.(bool); ok {
			return reflect.ValueOf(b), nil
		}
	case reflect.Slice:
		if target.Elem().Kind() == reflect.String {
			if arr, ok := val.([]interface{}); ok {
				ss := make([]string, len(arr))
				for i, v := range arr {
					ss[i] = fmt.Sprintf("%v", v)
				}
				return reflect.ValueOf(ss), nil
			}
		}
	case reflect.Func:
		// Function types (callbacks) — skip, handled by wrappers manually
		return reflect.Zero(target), nil
	}

	// String to named types (e.g. meuishim.Color)
	if s, ok := val.(string); ok {
		if target.Name() != "" {
			// Try to parse string — for user-defined types with Set method
			if target.Kind() == reflect.Uint32 || target.Kind() == reflect.Uint64 {
				// Color might be uint32 underneath
				if u, err := strconv.ParseUint(s, 0, 64); err == nil {
					return reflect.ValueOf(u).Convert(target), nil
				}
			}
			_ = s
		}
	}

	return reflect.Zero(target), fmt.Errorf("cannot convert %T to %v", val, target)
}
