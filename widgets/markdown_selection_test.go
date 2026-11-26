package widgets

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
)

const selectionCorpus = "Alpha BRAVO charlie\n\nDelta ECHO foxtrot\n\n- list GOLF item\n"

func selectionView(t *testing.T) *MarkdownView {
	t.Helper()
	mv := NewMarkdownView()
	mv.SetFrameRect(graphics.NewRect(0, 0, 600, 400))
	mv.SetText(selectionCorpus)
	mv.ensureLayout()
	return mv
}

// pointAt returns a widget-local point inside the glyph at a source offset.
// Half a pixel past the caret keeps the point on the intended side of the
// nearest-boundary rule even for a 2px glyph.
func pointAt(t *testing.T, mv *MarkdownView, off int) (float32, float32) {
	t.Helper()
	r := mv.CaretRect(off)
	if r.Width == 0 {
		t.Fatalf("offset %d has no caret rect", off)
	}
	return r.X + 0.5, r.Y + r.Height/2
}

func boundsString(mv *MarkdownView) string {
	lo, hi := mv.SelectionBounds()
	return fmt.Sprintf("[%d,%d)", lo, hi)
}

func offsetOf(t *testing.T, mv *MarkdownView, word string) int {
	t.Helper()
	at := strings.Index(mv.Text(), word)
	if at < 0 {
		t.Fatalf("%q not in the corpus", word)
	}
	return at
}

func TestMarkdownClickPlacesACaretAndClearsTheSelection(t *testing.T) {
	mv := selectionView(t)
	bravo := offsetOf(t, mv, "BRAVO")
	x, y := pointAt(t, mv, bravo+1)

	if !mv.OnMouseDown(x, y, 0) {
		t.Fatal("press on text was not handled")
	}
	if mv.HasSelection() {
		t.Fatalf("a plain click left a selection: %s", boundsString(mv))
	}
	if got := mv.CaretOffset(); got != bravo+1 {
		t.Fatalf("caret at %d, want %d", got, bravo+1)
	}
	mv.OnMouseUp(x, y, 0)

	// A second press far away (a different paragraph, >4px) starts fresh.
	echo := offsetOf(t, mv, "ECHO")
	x2, y2 := pointAt(t, mv, echo+2)
	mv.OnMouseDown(x2, y2, 0)
	if mv.HasSelection() {
		t.Fatalf("the second click left a selection: %s", boundsString(mv))
	}
	if got := mv.CaretOffset(); got != echo+2 {
		t.Fatalf("caret at %d, want %d", got, echo+2)
	}
}

func TestMarkdownDragSelectsAnExactCharacterRange(t *testing.T) {
	mv := selectionView(t)
	bravo := offsetOf(t, mv, "BRAVO")
	echo := offsetOf(t, mv, "ECHO")
	a, b := bravo+2, echo+3

	ax, ay := pointAt(t, mv, a)
	mv.OnMouseDown(ax, ay, 0)
	bx, by := pointAt(t, mv, b)
	mv.OnMouseMove(bx, by)
	mv.OnMouseUp(bx, by, 0)

	lo, hi := mv.SelectionBounds()
	if lo != a || hi != b {
		t.Fatalf("selection [%d,%d), want [%d,%d)", lo, hi, a, b)
	}
	want := selectionCorpus[a:b]
	// The plain text spans three lines; markdown syntax would add nothing
	// here, so it must be the source slice with the blank lines collapsed.
	if got := mv.SelectedText(); got != strings.ReplaceAll(want, "\n\n", "\n") {
		t.Fatalf("selected text = %q, want %q", got, strings.ReplaceAll(want, "\n\n", "\n"))
	}
	if got := mv.SelectedMarkdown(); got != want {
		t.Fatalf("selected markdown = %q, want %q", got, want)
	}
}

func TestMarkdownDoubleClickSelectsTheWord(t *testing.T) {
	mv := selectionView(t)
	echo := offsetOf(t, mv, "ECHO")
	x, y := pointAt(t, mv, echo+1)

	mv.OnMouseDown(x, y, 0)
	mv.OnMouseUp(x, y, 0)
	mv.OnMouseDown(x, y, 0)

	lo, hi := mv.SelectionBounds()
	if got := mv.SelectedMarkdown(); got != "ECHO " {
		t.Fatalf("double-click selected %q ([%d,%d)), want %q", got, lo, hi, "ECHO ")
	}
}

func TestMarkdownTripleClickSelectsTheBlock(t *testing.T) {
	mv := selectionView(t)
	echo := offsetOf(t, mv, "ECHO")
	x, y := pointAt(t, mv, echo+1)

	mv.OnMouseDown(x, y, 0)
	mv.OnMouseUp(x, y, 0)
	mv.OnMouseDown(x, y, 0)
	mv.OnMouseUp(x, y, 0)
	mv.OnMouseDown(x, y, 0)

	want := "Delta ECHO foxtrot"
	if got := mv.SelectedMarkdown(); got != want {
		t.Fatalf("triple-click selected %q, want %q", got, want)
	}
}

func TestMarkdownShiftClickExtendsTheSelection(t *testing.T) {
	mv := selectionView(t)
	alpha := offsetOf(t, mv, "Alpha")
	golf := offsetOf(t, mv, "GOLF")

	ax, ay := pointAt(t, mv, alpha+2)
	mv.OnMouseDown(ax, ay, 0)
	mv.OnMouseUp(ax, ay, 0)

	mv.testModifiers = platform.ModShift
	gx, gy := pointAt(t, mv, golf+1)
	mv.OnMouseDown(gx, gy, 0)

	lo, hi := mv.SelectionBounds()
	if lo != alpha+2 || hi != golf+1 {
		t.Fatalf("shift-click selected [%d,%d), want [%d,%d)", lo, hi, alpha+2, golf+1)
	}
}

func TestMarkdownKeyboardSelectsAndCopies(t *testing.T) {
	mv := selectionView(t)
	mv.OnFocus()

	if !mv.OnKeyDown(platform.KeyA, platform.ModCtrl) {
		t.Fatal("Ctrl+A was not handled")
	}
	lo, hi := mv.SelectionBounds()
	if lo != 0 || hi != len(selectionCorpus) {
		t.Fatalf("Ctrl+A selected [%d,%d), want the whole document", lo, hi)
	}
	if !mv.OnKeyDown(platform.KeyC, platform.ModCtrl) {
		t.Fatal("Ctrl+C was not handled")
	}
	if got := platform.GlobalClipboard.GetText(); !strings.Contains(got, "Alpha BRAVO charlie") || !strings.Contains(got, "list GOLF item") {
		t.Fatalf("clipboard = %q", got)
	}

	// Ctrl+Shift+C puts the markdown source on the clipboard instead.
	mv.OnKeyDown(platform.KeyC, platform.ModCtrl|platform.ModShift)
	if got := platform.GlobalClipboard.GetText(); got != selectionCorpus {
		t.Fatalf("markdown copy = %q, want the source", got)
	}
	platform.GlobalClipboard.SetText("")
}

func TestMarkdownShiftArrowExtendsByCharacter(t *testing.T) {
	mv := selectionView(t)
	mv.OnFocus()
	echo := offsetOf(t, mv, "ECHO")
	mv.CollapseTo(echo)

	if !mv.OnKeyDown(platform.KeyRight, platform.ModShift) {
		t.Fatal("shift+right was not handled")
	}
	lo, hi := mv.SelectionBounds()
	if lo != echo || hi != echo+1 {
		t.Fatalf("shift+right selected [%d,%d), want [%d,%d)", lo, hi, echo, echo+1)
	}
	mv.OnKeyDown(platform.KeyRight, platform.ModShift)
	mv.OnKeyDown(platform.KeyRight, platform.ModShift)
	if lo, hi = mv.SelectionBounds(); lo != echo || hi != echo+3 {
		t.Fatalf("after three extends selection is [%d,%d), want [%d,%d)", lo, hi, echo, echo+3)
	}
	// Without shift the caret moves and the selection collapses.
	mv.OnKeyDown(platform.KeyRight, 0)
	if mv.HasSelection() {
		t.Fatalf("plain arrow left a selection: %s", boundsString(mv))
	}
	if got := mv.CaretOffset(); got != echo+3 {
		t.Fatalf("caret at %d, want %d", got, echo+3)
	}
}

func TestMarkdownDragPastTheEdgeAutoscrolls(t *testing.T) {
	mv := NewMarkdownView()
	mv.SetFrameRect(graphics.NewRect(0, 0, 600, 200))
	mv.SetText(strings.Repeat("line of MARKDOWN text\n\n", 60))
	mv.ensureLayout()
	if mv.contentH <= 200 {
		t.Fatalf("contentH %v does not overflow the frame", mv.contentH)
	}
	lo, _ := mv.blockRangeAt(0)
	x, y := pointAt(t, mv, lo+1)
	mv.OnMouseDown(x, y, 0)

	// Drag below the bottom edge: the view scrolls and keeps extending.
	mv.OnMouseMove(x, 199)
	before := mv.scrollOff
	mv.OnMouseMove(x, 260)
	mv.OnMouseMove(x, 320)
	if mv.scrollOff <= before {
		t.Fatalf("drag past the bottom edge did not scroll (off %v -> %v)", before, mv.scrollOff)
	}
	if !mv.HasSelection() {
		t.Fatal("drag did not extend the selection")
	}
	mv.OnMouseUp(x, 320, 0)
}

func TestMarkdownEscapeClearsTheSelection(t *testing.T) {
	mv := selectionView(t)
	mv.OnFocus()
	mv.SelectAll()
	if !mv.HasSelection() {
		t.Fatal("SelectAll did not select")
	}
	if !mv.OnKeyDown(platform.KeyEscape, 0) {
		t.Fatal("Escape was not handled")
	}
	if mv.HasSelection() {
		t.Fatalf("Escape left a selection: %s", boundsString(mv))
	}
}
