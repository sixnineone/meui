package format

import (
	"strings"
	"testing"
)

// Object literals used as values are split on commas by the loader, so a
// comma-separated body must never be rewritten with semicolons.
func TestFormatCommaObjectLiteralKeepsCommas(t *testing.T) {
	src := []byte("Window {\n    menus: [\n        { label: \"Cut\", shortcut: \"Ctrl+X\" }\n    ]\n}\n")
	got, err := Format(src)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	s := string(got)
	if !strings.Contains(s, `{ label: "Cut", shortcut: "Ctrl+X" }`) {
		t.Fatalf("comma body not preserved:\n%s", s)
	}
	if strings.Contains(s, `label: "Cut";`) {
		t.Fatalf("comma was rewritten to semicolon:\n%s", s)
	}
}

// Commas must survive even when the object is printed one property per line.
func TestFormatMultilineCommaObjectKeepsCommas(t *testing.T) {
	src := []byte("Window {\n    items: [\n        { label: \"A\", list: [\n            \"x\"\n        ] }\n    ]\n}\n")
	got, err := Format(src)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	s := string(got)
	if !strings.Contains(s, `label: "A",`) {
		t.Fatalf("comma dropped from multi-line object:\n%s", s)
	}
}

func TestFormatImportStatements(t *testing.T) {
	src := []byte("import QtQuick 2.0\nimport \"theme\"\n\nWindow {\n  id: root\n}\n")
	got, err := Format(src)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	s := string(got)
	if !strings.Contains(s, "import QtQuick 2.0\n") {
		t.Fatalf("QtQuick import dropped:\n%s", s)
	}
	if !strings.Contains(s, "import \"theme\"\n") {
		t.Fatalf("path import dropped:\n%s", s)
	}
	// The blank line after the import block is preserved.
	if !strings.Contains(s, "import \"theme\"\n\nWindow {") {
		t.Fatalf("blank line after imports lost:\n%s", s)
	}
}

func TestFormatImportKeepsTrailingComment(t *testing.T) {
	src := []byte("import QtQuick 2.0 // version pinned\nWindow {\n    id: root\n}\n")
	got, err := Format(src)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	s := string(got)
	if !strings.Contains(s, "import QtQuick 2.0 // version pinned") {
		t.Fatalf("import trailing comment lost:\n%s", s)
	}
}

func TestFormatHashCommentPreserved(t *testing.T) {
	src := []byte("# Tab 2: Layouts\nWindow {\n    id: root\n}\n")
	got, err := Format(src)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if !strings.Contains(string(got), "# Tab 2: Layouts") {
		t.Fatalf("hash comment dropped:\n%s", got)
	}
}

func TestFormatSemicolonSeparatedSiblingObjects(t *testing.T) {
	src := []byte("Window {\n    HStack { Label { text: \"a\" }; Label { text: \"b\" } }\n}\n")
	got, err := Format(src)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	s := string(got)
	if !strings.Contains(s, `Label { text: "a" }`) || !strings.Contains(s, `Label { text: "b" }`) {
		t.Fatalf("sibling objects lost:\n%s", s)
	}
}

// A trailing comment after a closing brace must stay there; anchoring it
// to the opening brace made a second pass parse it as a body comment, so
// the formatter was not a fixpoint.
func TestFormatTrailingCommentAfterObjectStaysOutside(t *testing.T) {
	src := []byte("Window {\n    Column {\n        Label { text: \"a\" }\n    } // end column\n}\n")
	first, err := Format(src)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if !strings.Contains(string(first), "} // end column") {
		t.Fatalf("trailing comment moved off the closing brace:\n%s", first)
	}
	second, err := Format(first)
	if err != nil {
		t.Fatalf("Format second: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("not idempotent:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}

// The whole document must be a formatting fixpoint.
func TestFormatCombinedSyntaxIsIdempotent(t *testing.T) {
	src := []byte("import \"theme\"\n# heading\n\nWindow {\n  id: root\n  menus: [\n      { label: \"A\", value: 1 },\n      { label: \"B\", value: 2 }\n  ]\n}\n")
	first, err := Format(src)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	second, err := Format(first)
	if err != nil {
		t.Fatalf("Format second: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("not idempotent:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}
