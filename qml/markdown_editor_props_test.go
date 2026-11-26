package qml

import (
	"testing"

	"github.com/sixnineone/meui/widgets"
)

// The QML property path for MarkdownEditor: "text" has a global handler that
// dispatches by widget type, so the type must be listed there or the scene's
// text never reaches the widget; the other properties go through the
// catch-all to SetBoundProperty.

func TestMarkdownEditorTextPropertyReachesTheWidget(t *testing.T) {
	w := widgets.NewMarkdownEditor()
	if !HandleProperty(w, "text", "# hi\n\nbody\n") {
		t.Fatal("the text property was not handled")
	}
	if got := w.Text(); got != "# hi\n\nbody\n" {
		t.Fatalf("source text = %q", got)
	}
	if got := w.Preview().Text(); got != "# hi\n\nbody\n" {
		t.Fatalf("preview text = %q", got)
	}
}

func TestMarkdownEditorBoundPropertiesReachTheWidget(t *testing.T) {
	w := widgets.NewMarkdownEditor()

	if !HandleProperty(w, "mode", "preview") {
		t.Fatal("mode was not handled")
	}
	if w.Mode() != widgets.MarkdownPreviewOnly {
		t.Fatalf("mode = %v, want preview", w.Mode())
	}
	if !HandleProperty(w, "mode", "source") {
		t.Fatal("mode was not handled")
	}
	if w.Mode() != widgets.MarkdownSourceOnly {
		t.Fatalf("mode = %v, want source", w.Mode())
	}
	if !HandleProperty(w, "mode", "split") {
		t.Fatal("mode was not handled")
	}
	if w.Mode() != widgets.MarkdownSplit {
		t.Fatalf("mode = %v, want split", w.Mode())
	}

	if !HandleProperty(w, "splitRatio", float64(0.75)) {
		t.Fatal("splitRatio was not handled")
	}
	if got := w.SplitRatio(); got != 0.75 {
		t.Fatalf("splitRatio = %v, want 0.75", got)
	}

	if !HandleProperty(w, "readOnly", true) {
		t.Fatal("readOnly was not handled")
	}
	if !w.IsReadOnly() {
		t.Fatal("readOnly did not reach the widget")
	}
}
