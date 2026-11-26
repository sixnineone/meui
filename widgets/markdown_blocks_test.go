package widgets

import (
	"strings"
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
)

// Phase 4: the block semantics of a browser selection — code fences as text,
// tables row-wise, images atomic, links click-vs-select.

const blockCorpus = "Before APPLE\n\n~~~go\ncode HAZEL\nsecond LINE\n~~~\n\n" +
	"| Head A | Head B |\n| --- | --- |\n| cell FIG | cell GUAVA |\n\n" +
	"![alt PICTURE](img.png)\n\nAfter BANANA\n"

func blockView(t *testing.T) *MarkdownView {
	t.Helper()
	mv := NewMarkdownView()
	mv.SetFrameRect(graphics.NewRect(0, 0, 600, 400))
	mv.SetText(blockCorpus)
	mv.ensureLayout()
	return mv
}

func paintOnce(mv *MarkdownView) {
	mv.Paint(graphics.NewCanvas(graphics.NewTestDevice(600, 400)), graphics.NewRect(0, 0, 600, 400))
}

func TestMarkdownCodeFenceSelectsAsText(t *testing.T) {
	mv := blockView(t)
	code := offsetOf(t, mv, "code HAZEL")
	line2 := offsetOf(t, mv, "second LINE")

	ax, ay := pointAt(t, mv, code+1)
	mv.OnMouseDown(ax, ay, 0)
	bx, by := pointAt(t, mv, line2+3)
	mv.OnMouseMove(bx, by)

	lo, hi := mv.SelectionBounds()
	if lo != code+1 || hi != line2+3 {
		t.Fatalf("code selection [%d,%d), want [%d,%d)", lo, hi, code+1, line2+3)
	}
	if got := mv.SelectedText(); got != "ode HAZEL\nsec" {
		t.Fatalf("code plain text = %q, want %q", got, "ode HAZEL\nsec")
	}
}

func TestMarkdownTableCaretLandsInTheClickedCell(t *testing.T) {
	mv := blockView(t)
	fig := offsetOf(t, mv, "cell FIG")

	// A click inside the first cell puts the caret on that cell's character,
	// not on the whole row.
	ax, ay := pointAt(t, mv, fig+5)
	mv.OnMouseDown(ax, ay, 0)
	if got := mv.CaretOffset(); got != fig+5 {
		t.Fatalf("caret in a table cell at %d, want %d", got, fig+5)
	}
	mv.OnMouseUp(ax, ay, 0)
}

func TestMarkdownTableSelectsRowWiseWithTabs(t *testing.T) {
	mv := blockView(t)
	fig := offsetOf(t, mv, "cell FIG")
	guava := offsetOf(t, mv, "cell GUAVA")

	// Select from the middle of one cell to the middle of the next (a fresh
	// view, so this press is not a double-click).
	ax, ay := pointAt(t, mv, fig+5)
	mv.OnMouseDown(ax, ay, 0)
	bx, by := pointAt(t, mv, guava+4)
	mv.OnMouseMove(bx, by)
	mv.OnMouseUp(bx, by, 0)

	lo, hi := mv.SelectionBounds()
	if lo != fig+5 || hi != guava+4 {
		t.Fatalf("table selection [%d,%d), want [%d,%d)", lo, hi, fig+5, guava+4)
	}
	if got := mv.SelectedText(); got != "FIG\tcell" {
		t.Fatalf("table plain text = %q, want %q", got, "FIG\tcell")
	}
	if rects := mv.SelectionRects(lo, hi); len(rects) < 1 {
		t.Fatalf("no highlight rects for the table selection")
	}
}

func TestMarkdownImageIsAtomic(t *testing.T) {
	mv := blockView(t)
	alt := offsetOf(t, mv, "alt PICTURE")
	apple := offsetOf(t, mv, "APPLE")

	// Drag forward from before the image into its middle: the focus snaps
	// back to the image's start, so the image is all-or-nothing.
	ax, ay := pointAt(t, mv, apple+1)
	mv.OnMouseDown(ax, ay, 0)
	bx, by := pointAt(t, mv, alt+3)
	mv.OnMouseMove(bx, by)
	if _, hi := mv.SelectionBounds(); hi != alt {
		t.Fatalf("forward drag into the image ends at %d, want the image start %d", hi, alt)
	}

	// Drag backwards from after the image into its middle: the focus snaps
	// forward to the image's end.
	after := offsetOf(t, mv, "After BANANA")
	ax, ay = pointAt(t, mv, after+1)
	mv.OnMouseDown(ax, ay, 0)
	mv.OnMouseUp(ax, ay, 0)
	mv.CollapseTo(after + 1)
	mv.SelectTo(alt + 3)
	lo, _ := mv.SelectionBounds()
	if lo != mv.imageEnd(t) {
		t.Fatalf("backward drag into the image starts at %d, want its end %d", lo, mv.imageEnd(t))
	}
}

// imageEnd returns the source offset just past the image's alt text.
func (mv *MarkdownView) imageEnd(t *testing.T) int {
	t.Helper()
	for bi := range mv.laid {
		for li := range mv.laid[bi].lines {
			for ri := range mv.laid[bi].lines[li].runs {
				run := &mv.laid[bi].lines[li].runs[ri]
				if run.kind != mdRunImage {
					continue
				}
				if _, hi := run.sourceExtent(); hi > 0 {
					return hi
				}
			}
		}
	}
	t.Fatal("no image run in the layout")
	return -1
}

func TestMarkdownLinkOpensOnClickNotOnDrag(t *testing.T) {
	mv := NewMarkdownView()
	mv.SetFrameRect(graphics.NewRect(0, 0, 600, 400))
	mv.SetText("Click the [LINK TEXT](https://example.com/x) here.\n")
	paintOnce(mv)
	if len(mv.linkRects) == 0 {
		t.Fatal("no link rects after painting")
	}
	var opened []string
	mv.OnLinkClicked = func(url string) { opened = append(opened, url) }

	lr := mv.linkRects[0]
	cx, cy := lr.rect.X+lr.rect.Width/2, lr.rect.Y+lr.rect.Height/2

	// A plain click (press and release without moving) opens the link.
	mv.OnMouseDown(cx, cy, 0)
	mv.OnMouseUp(cx, cy, 0)
	if len(opened) != 1 || opened[0] != "https://example.com/x" {
		t.Fatalf("click opened %v, want the link once", opened)
	}

	// A press that becomes a drag selects instead of opening.
	opened = nil
	mv.OnMouseDown(cx, cy, 0)
	mv.OnMouseMove(cx+30, cy+2)
	mv.OnMouseUp(cx+30, cy+2, 0)
	if len(opened) != 0 {
		t.Fatalf("dragging over the link opened %v", opened)
	}
	if !mv.HasSelection() {
		t.Fatal("dragging over the link did not select")
	}

	// Ctrl-click opens immediately.
	opened = nil
	mv.testModifiers = platform.ModCtrl
	mv.OnMouseDown(cx, cy, 0)
	if len(opened) != 1 {
		t.Fatalf("ctrl-click opened %v, want the link once", opened)
	}
	mv.testModifiers = 0
}

func TestMarkdownLinkTextIsSelectable(t *testing.T) {
	mv := NewMarkdownView()
	mv.SetFrameRect(graphics.NewRect(0, 0, 600, 400))
	mv.SetText("Click the [LINK TEXT](https://example.com/x) here.\n")
	paintOnce(mv)
	mv.SelectAll()
	if got := mv.SelectedText(); !strings.Contains(got, "LINK TEXT") {
		t.Fatalf("select-all plain text = %q, want the link label", got)
	}
	if got := mv.SelectedMarkdown(); !strings.Contains(got, "[LINK TEXT](https://example.com/x)") {
		t.Fatalf("select-all markdown = %q", got)
	}
}

// TestMarkdownCodeBlockKeepsItsOwnLines locks the fix for a rendering bug the
// selection work exposed: a code line's goldmark segment keeps its newline, so
// joining the raw values inserted a blank line between every pair of lines.
func TestMarkdownCodeBlockKeepsItsOwnLines(t *testing.T) {
	mv := NewMarkdownView()
	mv.SetFrameRect(graphics.NewRect(0, 0, 600, 400))
	mv.SetText("```go\nline one\nline two\n```\n")
	if len(mv.blocks) != 1 || mv.blocks[0].Type != 2 {
		t.Fatalf("expected one code block, got %d blocks", len(mv.blocks))
	}
	if got := mv.blocks[0].CodeText; got != "line one\nline two" {
		t.Fatalf("code text = %q, want %q", got, "line one\nline two")
	}
}
