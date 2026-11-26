package widgets

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/sixnineone/meui/graphics"
)

// The core in textlayout.go is widget-agnostic: these tests build layouts by
// hand and exercise the offset <-> position queries and the soft wrap without
// going through the markdown view.

func testFont() graphics.Font {
	if graphics.SystemFont != nil {
		return graphics.SystemFont
	}
	return graphics.SystemFont
}

// plainRun builds a run whose runes map 1:1 onto src[srcStart:].
func plainRun(text string, srcStart int, x, y float32) textRun {
	return textRun{
		text:     text,
		font:     testFont(),
		x:        x,
		y:        y,
		w:        testFont().SimpleWidth(text),
		h:        testFont().LineHeight(),
		srcStart: srcStart,
		srcEnd:   srcStart + len(text),
		runeSrc:  plainRuneOffsets(text, srcStart),
	}
}

func twoLineLayout() (*textLayout, string) {
	src := "Alpha BRAVO\nCharlie DELTA"
	f := testFont()
	l := &textLayout{lines: []textLine{
		{y: 0, h: f.LineHeight(), runs: []textRun{plainRun("Alpha BRAVO", 0, 0, 0)}},
		{y: f.LineHeight(), h: f.LineHeight(), runs: []textRun{plainRun("Charlie DELTA", 12, 0, f.LineHeight())}},
	}}
	return l, src
}

func TestCoreCaretAndHitTestRoundTrip(t *testing.T) {
	l, src := twoLineLayout()
	for _, line := range l.lines {
		for _, off := range line.runs[0].runeSrc {
			x, y, h, ok := l.caret(off)
			if !ok {
				t.Fatalf("no caret for offset %d", off)
			}
			if got := l.offsetAt(x+0.5, y+h/2); got != off {
				t.Fatalf("caret for %d at (%v,%v) hit-tests back to %d (src %q)", off, x, y, got, src)
			}
		}
	}
}

func TestCoreCaretAtLineEnds(t *testing.T) {
	l, _ := twoLineLayout()
	// The offset just past the first line's last character is the second
	// line's first rune offset (the newline has no glyph).
	last := l.lines[0].runs[0].runeSrc[len(l.lines[0].runs[0].runeSrc)-1]
	x, ok := l.caretX(last + 1)
	if !ok {
		t.Fatal("no caret at the end of the first line")
	}
	runEnd := l.lines[0].runs[0].x + l.lines[0].runs[0].w
	if x != runEnd {
		t.Fatalf("caret past the last rune is at %v, want %v", x, runEnd)
	}
}

func TestCoreRectsCoverTheRequestedRange(t *testing.T) {
	l, _ := twoLineLayout()
	// "BRAVO\nChar": from the middle of the first line to the middle of the
	// second, so both lines contribute a rectangle.
	a, b := 6, 15
	rects := l.rects(a, b)
	if len(rects) != 2 {
		t.Fatalf("got %d rects for a two-line range, want 2", len(rects))
	}
	for _, r := range rects {
		if r.Width <= 0 || r.Height <= 0 {
			t.Fatalf("empty rect %+v", r)
		}
	}
	if rects[0].Y >= rects[1].Y {
		t.Fatalf("rects are not in document order: %+v", rects)
	}
	if got := l.rects(b, a); len(got) != len(rects) {
		t.Fatalf("an inverted range gave %d rects, want %d", len(got), len(rects))
	}
}

func TestCoreAtomicRunSnapsToTheEdgeComingFrom(t *testing.T) {
	f := testFont()
	l := &textLayout{lines: []textLine{{y: 0, h: f.LineHeight(), runs: []textRun{
		plainRun("see ", 0, 0, 0),
		{text: "[img]", font: f, x: 30, y: 0, w: 30, h: f.LineHeight(), srcStart: 10, srcEnd: 20, atomic: true},
		plainRun(" here", 20, 60, 0),
	}}}}
	if got := l.snapAtomic(15, 0); got != 10 {
		t.Fatalf("dragging forward into the image snapped to %d, want 10", got)
	}
	if got := l.snapAtomic(15, 20); got != 20 {
		t.Fatalf("dragging backward into the image snapped to %d, want 20", got)
	}
	if got := l.offsetAt(45, f.LineHeight()/2); got != 10 {
		t.Fatalf("a click on the image maps to %d, want its start 10", got)
	}
}

func TestCoreCaretStopsSkipUnglyphedOffsets(t *testing.T) {
	f := testFont()
	// The source "a**b**c" renders as "abc": the marks have no glyph.
	l := &textLayout{lines: []textLine{{y: 0, h: f.LineHeight(), runs: []textRun{
		plainRun("a", 0, 0, 0),
		plainRun("b", 3, 6, 0),
		plainRun("c", 6, 12, 0),
	}}}}
	// Caret stops are rune starts and run ends — never an offset inside text
	// that has no glyph of its own. With "a**b**c" that is {0,1,3,4,6,7}.
	stops := l.caretStops()
	want := []int{0, 1, 3, 4, 6, 7}
	if len(stops) != len(want) {
		t.Fatalf("caret stops = %v, want %v", stops, want)
	}
	for i := range want {
		if stops[i] != want[i] {
			t.Fatalf("caret stops = %v, want %v", stops, want)
		}
	}
}

func TestCoreLineLookups(t *testing.T) {
	l, _ := twoLineLayout()
	f := testFont()
	if got := l.lineIndexAtY(f.LineHeight() * 1.5); got != 1 {
		t.Fatalf("lineIndexAtY on the second line = %d, want 1", got)
	}
	first := l.lineEdge(0, false)
	last := l.lineEdge(1, true)
	if first != 0 || last != len("Alpha BRAVO\nCharlie DELTA") {
		t.Fatalf("line edges = %d..%d, want 0..%d", first, last, len("Alpha BRAVO\nCharlie DELTA"))
	}
}

func TestCoreWordRange(t *testing.T) {
	const src = "one two-three  four"
	cases := []struct {
		off, lo, hi int
		why         string
	}{
		{1, 0, 4, "inside the first word (plus its space)"},
		{5, 4, 7, "the second word"},
		{7, 7, 8, "the hyphen is punctuation"},
		{10, 8, 15, "the third word plus the spaces that follow"},
		{13, 13, 15, "a whitespace run"},
		{16, 15, 19, "the last word"},
	}
	for _, c := range cases {
		lo, hi := wordRangeAt(src, c.off)
		if lo != c.lo || hi != c.hi {
			t.Errorf("wordRangeAt(%d) = [%d,%d), want [%d,%d) (%s)", c.off, lo, hi, c.lo, c.hi, c.why)
		}
	}
}

func TestCoreWordRangeIsRuneSafe(t *testing.T) {
	const src = "café ZÜRICH"
	lo, hi := wordRangeAt(src, 1) // inside "café"
	if got := src[lo:hi]; got != "café " {
		t.Fatalf("word at 1 = %q, want %q", got, "café ")
	}
	lo, hi = wordRangeAt(src, len("café Z")) // inside ZÜRICH (multi-byte)
	if got := src[lo:hi]; got != "ZÜRICH" {
		t.Fatalf("word at the end = %q, want %q", got, "ZÜRICH")
	}
}

func TestCoreWrapKeepsEveryRuneAndFitsTheWidth(t *testing.T) {
	f := testFont()
	const text = "abcdefghijklmnopqrstuvwxyz"
	width := f.SimpleWidth("abcde")
	runs := []textRun{{
		text: text, font: f, w: f.SimpleWidth(text), h: f.LineHeight(),
		srcStart: 0, srcEnd: len(text), runeSrc: plainRuneOffsets(text, 0),
	}}
	lines := wrapRuns(runs, width, f.LineHeight(), 0, 0)
	if len(lines) < 2 {
		t.Fatalf("wrap produced %d line(s) for %d characters at width %v", len(lines), len(text), width)
	}
	var joined strings.Builder
	for i, ln := range lines {
		var w float32
		for _, r := range ln.runs {
			joined.WriteString(r.text)
			w += r.w
		}
		if w > width+0.01 {
			t.Errorf("line %d is %v wide, over the %v limit", i, w, width)
		}
		if want := float32(i) * f.LineHeight(); ln.y != want {
			t.Errorf("line %d is at y=%v, want %v", i, ln.y, want)
		}
	}
	if got := joined.String(); got != text {
		t.Fatalf("wrapped text = %q, want %q", got, text)
	}
}

func TestCoreWrapOffsetsSurviveTheSplit(t *testing.T) {
	f := testFont()
	const text = "abcdefghij"
	width := f.SimpleWidth("abcd")
	lines := wrapRuns([]textRun{{
		text: text, font: f, w: f.SimpleWidth(text), h: f.LineHeight(),
		srcStart: 100, srcEnd: 110, runeSrc: plainRuneOffsets(text, 100),
	}}, width, f.LineHeight(), 0, 0)
	if len(lines) < 2 {
		t.Fatalf("expected a split, got %d line(s)", len(lines))
	}
	for _, ln := range lines {
		for _, r := range ln.runs {
			for i := range []rune(r.text) {
				if i >= len(r.runeSrc) || r.runeSrc[i] < 100 {
					t.Fatalf("run %q lost its per-rune offsets: %v", r.text, r.runeSrc)
				}
			}
		}
	}
	// The last rune's offset is still the last source offset.
	last := lines[len(lines)-1].runs[len(lines[len(lines)-1].runs)-1]
	if got := last.runeSrc[len(last.runeSrc)-1]; got != 109 {
		t.Fatalf("last wrapped rune maps to %d, want 109", got)
	}
}

func TestCoreWrapNeverSplitsAnAtomicRun(t *testing.T) {
	f := testFont()
	word := plainRun("aa", 0, 0, 0)
	img := textRun{text: "[image]", font: f, w: f.SimpleWidth("[image]"), h: f.LineHeight(), atomic: true, srcStart: 2, srcEnd: 9}
	lines := wrapRuns([]textRun{word, img}, f.SimpleWidth("aa")+4, f.LineHeight(), 0, 0)
	if len(lines) < 2 {
		t.Fatalf("the atomic run did not move to its own line: %d line(s)", len(lines))
	}
	last := lines[len(lines)-1]
	if len(last.runs) != 1 || last.runs[0].text != "[image]" {
		t.Fatalf("last line holds %+v, want the whole atomic run", last.runs)
	}
}

// The editors hold one line of text and ask column questions; these helpers
// share the run measurement with everything else.
func TestCoreColumnAndCaretAgree(t *testing.T) {
	f := testFont()
	const line = "column ALIGNMENT text"
	for col := 0; col <= len([]rune(line)); col++ {
		x := caretXAtColumn(line, f, col)
		back := columnAtX(line, f, x+0.5)
		if back != col {
			t.Fatalf("column %d is at x=%v but hit-tests back to %d", col, x, back)
		}
	}
	if got := columnAtX(line, f, -100); got != 0 {
		t.Fatalf("a click left of the line gave column %d, want 0", got)
	}
	if got := columnAtX(line, f, 100000); got != len([]rune(line)) {
		t.Fatalf("a click right of the line gave column %d, want %d", got, len([]rune(line)))
	}
}

func TestCoreWrapTextMatchesTheEditorAlgorithm(t *testing.T) {
	f := testFont()
	const text = "the quick brown fox jumps over the lazy dog"
	width := f.SimpleWidth("the quick brown")
	lines := wrapText(text, f, width)
	if len(lines) < 2 {
		t.Fatalf("wrapText produced %d line(s)", len(lines))
	}
	// Greedy: every line fits the width, and no line could take the first rune
	// of the next one.
	for i, ln := range lines {
		if w := f.SimpleWidth(ln); w > width+0.01 {
			t.Errorf("line %d is %v wide, over the %v limit", i, w, width)
		}
		if i+1 < len(lines) && len(ln) > 0 {
			next := []rune(lines[i+1])[0]
			if f.SimpleWidth(ln+string(next)) <= width {
				t.Errorf("line %d (%q) could still take %q", i, ln, string(next))
			}
		}
	}
	if got := strings.Join(lines, ""); got != text {
		t.Fatalf("wrapped text = %q, want %q", got, text)
	}
	if got := wrapText("", f, width); len(got) != 1 || got[0] != "" {
		t.Fatalf("wrapText(\"\") = %v, want one empty line", got)
	}
}

// Editor columns are byte offsets, and only rune boundaries are candidates.
func TestCoreLineMeasureUsesByteColumns(t *testing.T) {
	f := testFont()
	const text = "café x" // 'é' is two bytes
	// "café" is five bytes, so byte 5 is the boundary just after the 'é'.
	if got := caretXAtColumn(text, f, 5); got != f.SimpleWidth("café") {
		t.Fatalf("caret at byte 5 is %v, want the width of %q", got, "café")
	}
	if got := caretXAtColumn(text, f, 6); got != f.SimpleWidth("café ") {
		t.Fatalf("caret at byte 6 is %v, want the width of %q", got, "café ")
	}
	limit := f.SimpleWidth(text)
	for probe := float32(0); probe <= limit; probe += 0.25 {
		col := columnAtX(text, f, probe)
		if col < 0 || col > len(text) {
			t.Fatalf("probe %v gave column %d, outside the line", probe, col)
		}
		if col > 0 && col < len(text) && !utf8.RuneStart(text[col]) {
			t.Fatalf("probe %v gave column %d, inside the multi-byte rune", probe, col)
		}
	}
	// The caret x of every rune boundary round-trips.
	for i := range text {
		if i > 0 && !utf8.RuneStart(text[i]) {
			continue
		}
		x := caretXAtColumn(text, f, i)
		if got := columnAtX(text, f, x+0.5); got != i {
			t.Fatalf("byte column %d is at x=%v but hit-tests back to %d", i, x, got)
		}
	}
}

func TestCoreLineMeasureExpandsTabs(t *testing.T) {
	f := testFont()
	const line = "a\tb"
	m := lineMeasure{text: line, font: f, tabWidth: 4}
	if got := m.width(2); got != f.SimpleWidth("a   ") {
		t.Fatalf("a tab at column 1 advances to %v, want %v", got, f.SimpleWidth("a   "))
	}
	if m.width(2) <= f.SimpleWidth("a") {
		t.Fatalf("the tab did not advance the caret")
	}
	if got := m.width(3); got != f.SimpleWidth("a   b") {
		t.Fatalf("the full line measures %v, want %v", got, f.SimpleWidth("a   b"))
	}
	// A click inside the tab's span lands on the tab boundary, not inside it.
	if got := m.indexAt(f.SimpleWidth("a  ")); got != 1 && got != 2 {
		t.Fatalf("a click inside the tab gave column %d", got)
	}
}

// Document positions: (line, byte-column) <-> byte offset, shared by both
// editors.
func TestCoreDocumentPositionsRoundTrip(t *testing.T) {
	lines := []string{"first", "", "café x", "last line"}
	// Every valid position round-trips.
	for li, l := range lines {
		for col := 0; col <= len(l); col++ {
			if col > 0 && col < len(l) && !utf8.RuneStart(l[col]) {
				continue
			}
			off := docOffsetAt(lines, li, col)
			backLine, backCol := docLineColAt(lines, off)
			if backLine != li || backCol != col {
				t.Fatalf("(%d,%d) -> offset %d -> (%d,%d)", li, col, off, backLine, backCol)
			}
		}
	}
	// Offsets past the end clamp to the end of the last line.
	last := docOffsetAt(lines, len(lines)-1, len(lines[len(lines)-1]))
	if l, c := docLineColAt(lines, last+100); l != len(lines)-1 || c != len(lines[len(lines)-1]) {
		t.Fatalf("an offset past the end gave (%d,%d)", l, c)
	}
	// Out-of-range positions clamp rather than panic.
	if got := docOffsetAt(lines, -5, -5); got != 0 {
		t.Fatalf("a negative position gave offset %d", got)
	}
	if got := docOffsetAt(lines, 99, 99); got != last {
		t.Fatalf("a position past the end gave offset %d, want %d", got, last)
	}
	if l, c := docLineColAt(nil, 10); l != 0 || c != 0 {
		t.Fatalf("an empty document gave (%d,%d)", l, c)
	}
	if got := docOffsetAt(nil, 0, 0); got != 0 {
		t.Fatalf("an empty document gave offset %d", got)
	}
}

// The two editors must agree, since they now share these.
func TestEditorsShareDocumentPositions(t *testing.T) {
	const value = "alpha\nbeta gamma\n\ncafé delta"
	ta := textAreaView(t, value, false)
	ce := NewCodeEditor()
	ce.SetText(value)
	for off := 0; off <= len(value); off++ {
		tl, tc := ta.offsetToCursor(off)
		cl, cc := ce.offsetToLineCol(off)
		if tl != cl || tc != cc {
			t.Fatalf("offset %d: TextArea (%d,%d) vs CodeEditor (%d,%d)", off, tl, tc, cl, cc)
		}
		if got := ta.cursorToOffset(tl, tc); got != ce.lineColToOffset(cl, cc) {
			t.Fatalf("offset %d: the editors disagree going back (%d vs %d)", off, got, ce.lineColToOffset(cl, cc))
		}
	}
}

// TestLabelWrappingLivesInTheCore pins the label's wrap strategies as CORE behaviour:
// the widget delegates, so a second width/direction implementation cannot drift from
// the one the editors and the markdown view use.
func TestLabelWrappingLivesInTheCore(t *testing.T) {
	font := graphics.SystemFont
	if font == nil {
		t.Skip("no system font")
	}

	// wordWrap: breaks between words, never inside one.
	lines := wrapLabelText(font, "one two three four five six seven", 90, WrapWordWrap, 0)
	if len(lines) < 2 {
		t.Fatalf("wordWrap gave %d line(s) at 90px: %q", len(lines), lines)
	}
	for _, line := range lines {
		if strings.TrimSpace(line) != line {
			t.Errorf("a wrapped line has stray space: %q", line)
		}
		if w := font.SimpleWidth(line); w > 90.5 {
			t.Errorf("line %q is %.1fpx in a 90px box", line, w)
		}
	}

	// Intentional newlines are preserved in every mode.
	for _, mode := range []string{WrapNoWrap, WrapWordWrap, WrapEllipsis} {
		if got := wrapLabelText(font, "a\nb", 400, mode, 0); len(got) != 2 {
			t.Errorf("%s: %q became %d lines, want 2", mode, got, len(got))
		}
	}

	// Ellipsis: one line, ends with "…", fits.
	long := "the quick brown fox jumps over the lazy dog and keeps running"
	el := wrapLabelText(font, long, 120, WrapEllipsis, 0)
	if len(el) != 1 || !strings.HasSuffix(el[0], "…") {
		t.Fatalf("ellipsis gave %q", el)
	}
	if w := font.SimpleWidth(el[0]); w > 120.5 {
		t.Fatalf("ellipsised line is %.1fpx in 120px: %q", w, el[0])
	}

	// maximumLineCount: clamps and ellipsises the last kept line.
	clamped := wrapLabelText(font, strings.Repeat("word ", 40), 140, WrapWordWrap, 2)
	if len(clamped) != 2 {
		t.Fatalf("clamp gave %d lines, want 2: %q", len(clamped), clamped)
	}
	if !strings.HasSuffix(clamped[1], "…") {
		t.Fatalf("the clamped last line is not ellipsised: %q", clamped[1])
	}

	// Multi-byte truncation must not split a rune.
	fr := wrapLabelText(font, strings.Repeat("é", 80), 60, WrapEllipsis, 0)
	if !utf8.ValidString(fr[0]) {
		t.Fatalf("ellipsis produced invalid UTF-8: %q", fr[0])
	}
}
