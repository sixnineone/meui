package widgets

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
)

// TestACursorOverADocumentWithNoLayoutIsNotACrash is a regression for a crash that took
// the whole window down, and it was reached on the FIRST mouse move of every window.
//
// `nearestLineOffset` indexed `lines[lineIdx]` and then `runs[0]` to find the nearest
// offset, which panics when the layout has no lines — or when a line has no runs. Every
// caller already expects -1 (SourceOffsetAt documents it, CursorAt tests for it), so this
// was a missing guard rather than a design gap: the guard was missing, and nothing below
// it was prepared for the answer it never got to give.
func TestACursorOverADocumentWithNoLayoutIsNotACrash(t *testing.T) {
	mv := NewMarkdownView()
	mv.SetFrameRect(graphics.NewRect(0, 0, 400, 300))

	// No text at all.
	if off := mv.SourceOffsetAt(10, 10); off != -1 {
		t.Errorf("SourceOffsetAt on an empty layout = %d, want -1", off)
	}
	if c := mv.CursorAt(10, 10); c != nil {
		t.Errorf("CursorAt on an empty layout returned a cursor, want nil")
	}

	// A document whose blocks carry no runs — which is what a document of ONLY embeds
	// looks like before its layout is built, and it is the case the crash needed.
	mv.SetText("```nosuch {}\nbody\n```\n")
	if c := mv.CursorAt(10, 10); c != nil && c != platform.TextCursor() {
		t.Errorf("CursorAt over a document of only embeds returned %v", c)
	}

	// And with a selection in flight, because that is what a mouse move does.
	mv.SelectAll()
	mv.CursorAt(20, 20)
}
