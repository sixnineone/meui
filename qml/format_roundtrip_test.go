package qml

import (
	"testing"

	qmlformat "github.com/sixnineone/meui/qml/format"
)

// Formatting a comma-separated object literal must not change what the
// loader reads. Object literals used as values are split on commas only
// (see parseObjectToMap), so rewriting the commas as semicolons would
// collapse two fields into one.
func TestFormattedCommaObjectLiteralStillLoads(t *testing.T) {
	src := "Window {\n  id: root\n  menus: [ { label: \"Cut\", shortcut: \"Ctrl+X\" } ]\n}\n"
	out := qmlformat.FormatSafe([]byte(src), qmlformat.Options{Indent: 2, MaxInlineLength: 100})

	obj, err := ParseQML(string(out))
	if err != nil {
		t.Fatalf("formatted QML no longer parses: %v\n%s", err, out)
	}
	p, ok := findProperty(obj, "menus")
	if !ok {
		t.Fatalf("menus missing from formatted QML:\n%s", out)
	}
	arr, ok := p.Value.([]interface{})
	if !ok || len(arr) != 1 {
		t.Fatalf("menus = %#v, want one entry\n%s", p.Value, out)
	}
	m, ok := arr[0].(map[string]interface{})
	if !ok {
		t.Fatalf("menus[0] = %#v, want an object literal\n%s", arr[0], out)
	}
	if m["label"] != "Cut" || m["shortcut"] != "Ctrl+X" {
		t.Fatalf("menus[0] = %#v, want label=Cut shortcut=Ctrl+X\n%s", m, out)
	}
}

// Import statements must survive formatting and remain loadable.
func TestFormattedImportStillLoads(t *testing.T) {
	src := "import \"theme\"\n\nWindow {\n  id: root\n  width: 100\n}\n"
	out := qmlformat.FormatSafe([]byte(src), qmlformat.Options{Indent: 2, MaxInlineLength: 100})
	if string(out) != src {
		t.Fatalf("already-canonical import source was changed:\n%s", out)
	}
	if _, err := ParseQML(string(out)); err != nil {
		t.Fatalf("formatted import no longer parses: %v\n%s", err, out)
	}
}
