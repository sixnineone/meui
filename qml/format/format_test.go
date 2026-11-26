package format

import "testing"

func TestFormatBasic(t *testing.T) {
	src := []byte("Window {\n  id: root\n  width: 1100\n}\n")
	want := []byte("Window {\n    id: root\n    width: 1100\n}\n")
	got, err := Format(src)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatIsIdempotent(t *testing.T) {
	src := []byte("Window {\n    id: root\n    Button {\n        text: \"OK\"\n    }\n}\n")
	first, err := Format(src)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	second, err := Format(first)
	if err != nil {
		t.Fatalf("Format second: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("not idempotent:\n%s\n---\n%s", first, second)
	}
}

func TestFormatPreservesComments(t *testing.T) {
	src := []byte("Window {\n    // comment\n    Button {\n        text: \"OK\" // trailing\n    }\n}\n")
	got, err := Format(src)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	s := string(got)
	if !contains(s, "// comment") {
		t.Fatalf("comment dropped:\n%s", s)
	}
	if !contains(s, "// trailing") {
		t.Fatalf("trailing comment dropped:\n%s", s)
	}
}

func TestFormatInlineShortObject(t *testing.T) {
	src := []byte("Window {\n    ComboBox { width: 120; height: 20; model: [\"A\", \"B\"] }\n}\n")
	got, err := Format(src)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	s := string(got)
	if !contains(s, `ComboBox { width: 120; height: 20; model: ["A", "B"] }`) {
		t.Fatalf("unexpected output:\n%s", s)
	}
}

func TestFormatSafeLeavesUnsupportedArrayComment(t *testing.T) {
	src := []byte("Window {\n    menus: [\n        { label: \"A\" },\n        // keep\n    ]\n}\n")
	got := FormatSafe(src, DefaultOptions())
	if string(got) != string(src) {
		t.Fatalf("expected source unchanged, got:\n%s", got)
	}
}

func contains(s string, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
