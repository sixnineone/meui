package widgets

import (
	"strings"
	"testing"

	"github.com/sixnineone/meui/graphics"
)

// The layout table is only useful if positions and source offsets round-trip:
// a caret placed at an offset must hit-test back to that offset, and a
// selection range must cover exactly the glyphs between its two offsets.

const layoutCorpus = `# Heading PLUM

First paragraph with APPLE and **bold BANANA** text.

- list item CHERRY
- [x] task DAMSON

> quoted ELDER

| Col A | Col B |
| --- | --- |
| cell FIG | cell GUAVA |

~~~go
code HAZEL
~~~

Last paragraph INDIAN.
`

func layoutView(t *testing.T, text string) *MarkdownView {
	t.Helper()
	mv := NewMarkdownView()
	mv.SetFrameRect(graphics.NewRect(0, 0, 600, 400))
	mv.SetText(text)
	mv.ensureLayout()
	if len(mv.laid) == 0 {
		t.Fatal("no layout produced")
	}
	return mv
}

// TestLayoutCaretRoundTrip walks every character offset of a paragraph and
// checks that placing a caret there and hit-testing the caret's position
// returns the same offset.
func TestLayoutCaretRoundTrip(t *testing.T) {
	mv := layoutView(t, layoutCorpus)
	firstLine := firstBlockText(mv, "First paragraph with")
	if firstLine == nil {
		t.Fatal("the first paragraph did not lay out")
	}
	tried := 0
	for _, run := range firstLine {
		for _, off := range run.runeSrc {
			if off < 0 {
				continue
			}
			caret := mv.CaretRect(off)
			if caret.Width == 0 || caret.Height == 0 {
				t.Fatalf("offset %d (%q) has no caret rect", off, string(mv.Text())[off:])
			}
			back := mv.SourceOffsetAt(caret.X+0.5, caret.Y+caret.Height/2)
			if back != off {
				t.Fatalf("rune at offset %d round-tripped to %d (caret %+v)", off, back, caret)
			}
			tried++
		}
	}
	if tried < 10 {
		t.Fatalf("only %d offsets tested", tried)
	}
}

// TestLayoutSelectionRectsCoverRange checks that the highlight rectangles of a
// source range sit between the carets of its ends and stay inside the block.
func TestLayoutSelectionRectsCoverRange(t *testing.T) {
	mv := layoutView(t, layoutCorpus)
	line := firstBlockText(mv, "First paragraph with")
	if line == nil {
		t.Fatal("the first paragraph did not lay out")
	}
	lo, _ := runsExtent(line)
	a := lo + 6  // "First " → caret after the space
	b := lo + 11 // "First parag" → inside "paragraph"
	rects := mv.SelectionRects(a, b)
	if len(rects) == 0 {
		t.Fatalf("no selection rects for [%d,%d)", a, b)
	}
	ca, cb := mv.CaretRect(a), mv.CaretRect(b)
	first, last := rects[0], rects[len(rects)-1]
	if first.X < ca.X-0.5 || first.X > ca.X+0.5 {
		t.Errorf("selection starts at x=%v, caret a is at x=%v", first.X, ca.X)
	}
	if last.X+last.Width < cb.X-0.5 || last.X+last.Width > cb.X+0.5 {
		t.Errorf("selection ends at x=%v, caret b is at x=%v", last.X+last.Width, cb.X)
	}
	lineTop := line[0].y + markdownMargin - mv.scrollOff
	for _, r := range rects {
		if r.Y < lineTop-1 || r.Y+r.Height > lineTop+line[0].h+1 {
			t.Errorf("selection rect %+v escapes its line (top %v h %v)", r, lineTop, line[0].h)
		}
	}
}

// TestLayoutHitTestPicksTheRightBlock checks that a point in the middle of a
// later block maps into that block's source range, not the first one.
func TestLayoutHitTestPicksTheRightBlock(t *testing.T) {
	mv := layoutView(t, layoutCorpus)
	for _, want := range []string{"APPLE", "CHERRY", "ELDER", "HAZEL", "INDIAN"} {
		at := strings.Index(mv.Text(), want)
		if at < 0 {
			t.Fatalf("%q missing from the corpus", want)
		}
		caret := mv.CaretRect(at)
		if caret.Width == 0 {
			t.Fatalf("%q: no caret rect", want)
		}
		got := mv.SourceOffsetAt(caret.X+0.5, caret.Y+caret.Height/2)
		if got != at {
			t.Errorf("%q at offset %d hit-tests back to %d", want, at, got)
		}
	}
}

// TestLayoutOffsetsAreInsideSource is a sanity check over the whole corpus:
// every caret rect must resolve to an offset inside the source.
func TestLayoutOffsetsAreInsideSource(t *testing.T) {
	mv := layoutView(t, layoutCorpus)
	n := len(mv.Text())
	placed := 0
	for bi := range mv.laid {
		for li := range mv.laid[bi].lines {
			for ri := range mv.laid[bi].lines[li].runs {
				run := &mv.laid[bi].lines[li].runs[ri]
				lo, hi := run.sourceExtent()
				if lo < 0 {
					continue
				}
				if lo < 0 || hi > n || lo > hi {
					t.Fatalf("run %q has span [%d,%d) outside the source", run.text, lo, hi)
				}
				placed++
			}
		}
	}
	if placed < 20 {
		t.Fatalf("only %d runs carried a source span", placed)
	}
}

// firstBlockText returns the runs of the first line of the block whose text
// contains needle, or nil.
func firstBlockText(mv *MarkdownView, needle string) []textRun {
	for bi := range mv.laid {
		for li := range mv.laid[bi].lines {
			line := &mv.laid[bi].lines[li]
			var joined strings.Builder
			for _, r := range line.runs {
				joined.WriteString(r.text)
			}
			if strings.Contains(joined.String(), needle) {
				return line.runs
			}
		}
	}
	return nil
}

// runsExtent returns the source span covered by a line's runs.
func runsExtent(runs []textRun) (int, int) {
	lo, hi := -1, -1
	for _, r := range runs {
		s, e := r.sourceExtent()
		if s < 0 {
			continue
		}
		if lo < 0 || s < lo {
			lo = s
		}
		if e > hi {
			hi = e
		}
	}
	return lo, hi
}
