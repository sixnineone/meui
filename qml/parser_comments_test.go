package qml

import (
	"testing"
)

// findProperty returns the named property of obj, if present.
func findProperty(obj *Object, name string) (Property, bool) {
	for _, p := range obj.Properties {
		if p.Name == name {
			return p, true
		}
	}
	return Property{}, false
}

// A trailing // comment must terminate the value, not become part of it.
// Before the fix, `width: 100 // note` parsed as the string
// "100 // trailing note", so the widget silently fell back to its default.
func TestTrailingLineCommentIsNotPartOfValue(t *testing.T) {
	src := "Window {\n  width: 100 // trailing note\n  height: 50\n}\n"
	obj, err := ParseQML(src)
	if err != nil {
		t.Fatalf("ParseQML: %v", err)
	}
	p, ok := findProperty(obj, "width")
	if !ok {
		t.Fatal("width property missing")
	}
	if p.Value != 100 {
		t.Fatalf("width = %#v, want 100", p.Value)
	}
}

// A trailing comment must not stop the following properties from parsing.
func TestTrailingCommentDoesNotSwallowFollowingProperties(t *testing.T) {
	src := "Window {\n  width: 100 // note\n  height: 50 // another\n}\n"
	obj, err := ParseQML(src)
	if err != nil {
		t.Fatalf("ParseQML: %v", err)
	}
	if p, ok := findProperty(obj, "width"); !ok || p.Value != 100 {
		t.Fatalf("width = %#v (present=%v), want 100", p.Value, ok)
	}
	if p, ok := findProperty(obj, "height"); !ok || p.Value != 50 {
		t.Fatalf("height = %#v (present=%v), want 50", p.Value, ok)
	}
}

// An inline block comment between the colon and the value is ignored.
func TestInlineBlockCommentIsStripped(t *testing.T) {
	src := "Window {\n  width: /* px */ 100\n}\n"
	obj, err := ParseQML(src)
	if err != nil {
		t.Fatalf("ParseQML: %v", err)
	}
	p, ok := findProperty(obj, "width")
	if !ok {
		t.Fatal("width property missing")
	}
	if p.Value != 100 {
		t.Fatalf("width = %#v, want 100", p.Value)
	}
}

// Comments inside an array must be ignored, and every element must still
// parse as an object literal.
func TestCommentInsideArrayIsIgnored(t *testing.T) {
	src := "Window {\n  items: [\n    { label: \"A\" }, // first\n    { label: \"B\" }\n  ]\n}\n"
	obj, err := ParseQML(src)
	if err != nil {
		t.Fatalf("ParseQML: %v", err)
	}
	p, ok := findProperty(obj, "items")
	if !ok {
		t.Fatal("items property missing")
	}
	arr, ok := p.Value.([]interface{})
	if !ok {
		t.Fatalf("items = %#v, want array", p.Value)
	}
	if len(arr) != 2 {
		t.Fatalf("items has %d entries, want 2: %#v", len(arr), arr)
	}
	for i, item := range arr {
		if _, ok := item.(map[string]interface{}); !ok {
			t.Fatalf("items[%d] = %#v, want object literal", i, item)
		}
	}
}

// A whole-line comment must be skipped without disturbing the object.
func TestWholeLineCommentIsIgnored(t *testing.T) {
	src := "Window {\n  // { unbalanced brace in a comment\n  width: 100\n}\n"
	obj, err := ParseQML(src)
	if err != nil {
		t.Fatalf("ParseQML: %v", err)
	}
	p, ok := findProperty(obj, "width")
	if !ok {
		t.Fatal("width property missing")
	}
	if p.Value != 100 {
		t.Fatalf("width = %#v, want 100", p.Value)
	}
}

// Comment markers inside a quoted string must be preserved verbatim.
func TestCommentMarkerInsideStringIsPreserved(t *testing.T) {
	src := "Window {\n  url: \"http://example.com/a/*b*/\"\n}\n"
	obj, err := ParseQML(src)
	if err != nil {
		t.Fatalf("ParseQML: %v", err)
	}
	p, ok := findProperty(obj, "url")
	if !ok {
		t.Fatal("url property missing")
	}
	if p.Value != "http://example.com/a/*b*/" {
		t.Fatalf("url = %#v, want the original string", p.Value)
	}
}
