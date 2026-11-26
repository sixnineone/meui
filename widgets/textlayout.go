package widgets

import (
	"github.com/sixnineone/meui/graphics"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// This file is the text-layout core shared by every text-bearing widget: the
// markdown view builds a layout per block, and the editors build one for their
// display lines. It owns the geometry model (positioned runs and lines) and the
// offset <-> position queries, so hit-testing, caret placement and selection
// highlighting have exactly one implementation.
//
// Coordinates are widget-local content space: x grows to the right from the
// text's own origin (the widget adds its margins), y is the top of a line.

// textRun is one drawn piece of text. All coordinates are content space.
type textRun struct {
	kind    int
	nodeIdx int // index into a widget's node list, -1 when not node-backed
	text    string
	font    graphics.Font
	x, y    float32
	w, h    float32

	srcStart, srcEnd int
	// runeSrc[i] is the source offset of rune i of text, or -1 when unknown.
	// len(runeSrc) == utf8.RuneCountInString(text).
	runeSrc []int
	atomic  bool // selection snaps to the whole run (images, task glyphs)
}

// mdRun/mdLine are the markdown view's names for the shared model.
type mdRun = textRun
type mdLine = textLine

// textLine is one visual line.
type textLine struct {
	y, h float32
	runs []textRun
}

// textLayout is a positioned sequence of visual lines in document order.
type textLayout struct {
	lines []textLine
}

// sourceExtent returns the source span the run covers, if known.
func (r textRun) sourceExtent() (int, int) {
	if r.srcStart >= 0 && r.srcEnd > r.srcStart {
		return r.srcStart, r.srcEnd
	}
	lo, hi := -1, -1
	for _, o := range r.runeSrc {
		if o < 0 {
			continue
		}
		if lo < 0 || o < lo {
			lo = o
		}
		if o > hi {
			hi = o
		}
	}
	if lo >= 0 {
		return lo, hi + 1
	}
	return -1, -1
}

// runeCount returns the number of runes in the run's text.
func (r textRun) runeCount() int { return utf8.RuneCountInString(r.text) }

// offsetAt returns the source offset nearest to a content-space point, or -1
// when the layout holds no text.
func (l *textLayout) offsetAt(x, y float32) int {
	if l == nil || len(l.lines) == 0 {
		return -1
	}
	for i := range l.lines {
		line := &l.lines[i]
		if y < line.y || y > line.y+line.h {
			continue
		}
		for ri := range line.runs {
			run := &line.runs[ri]
			if x < run.x || x > run.x+run.w {
				continue
			}
			return offsetInRun(run, x)
		}
		if len(line.runs) > 0 {
			last := &line.runs[len(line.runs)-1]
			if x > last.x+last.w {
				if _, e := last.sourceExtent(); e >= 0 {
					return e
				}
			}
		}
		return l.nearestLineOffset(i, x, y)
	}
	return l.clampOffset(y)
}

// nearestLineOffset returns the offset on a line nearest to a point that is
// inside the line's vertical band but outside its runs.
func (l *textLayout) nearestLineOffset(lineIdx int, x, y float32) int {
	// A layout with NO LINES, and a line with NO RUNS, both have no nearest offset — and
	// indexing [0] to discover that takes the whole window down.
	//
	// This was not a design gap: every caller already expects -1. SourceOffsetAt
	// documents it ("returns -1 when the layout has no text at all") and CursorAt tests
	// for it. The guard was simply missing here, and it was reached the moment a window
	// asked for a cursor over a document whose layout had not been built yet — which is
	// every window, on the first mouse move.
	//
	// lineEdge already answers both cases correctly, so ask it rather than re-deciding.
	if lineIdx < 0 || lineIdx >= len(l.lines) {
		return -1
	}
	line := &l.lines[lineIdx]
	if len(line.runs) == 0 {
		return l.lineEdge(lineIdx, false)
	}
	if cx := x; cx <= line.runs[0].x {
		return l.lineEdge(lineIdx, false)
	}
	last := &line.runs[len(line.runs)-1]
	if x >= last.x+last.w {
		return l.lineEdge(lineIdx, true)
	}
	for ri := range line.runs {
		run := &line.runs[ri]
		if x >= run.x && x <= run.x+run.w {
			return offsetInRun(run, x)
		}
	}
	return l.lineEdge(lineIdx, true)
}

// clampOffset returns the first or last offset of the layout, whichever end y
// is nearer.
func (l *textLayout) clampOffset(y float32) int {
	first, last := -1, -1
	for i := range l.lines {
		if len(l.lines[i].runs) == 0 {
			continue
		}
		if first < 0 {
			first = i
		}
		last = i
	}
	if first < 0 {
		return -1
	}
	if y < l.lines[first].y {
		return l.lineEdge(first, false)
	}
	return l.lineEdge(last, true)
}

// lineEdge returns the offset at the start (end=false) or the end of a line.
func (l *textLayout) lineEdge(lineIdx int, end bool) int {
	if lineIdx < 0 || lineIdx >= len(l.lines) {
		return -1
	}
	line := &l.lines[lineIdx]
	if end {
		for ri := len(line.runs) - 1; ri >= 0; ri-- {
			if _, e := line.runs[ri].sourceExtent(); e > 0 {
				return e
			}
		}
		return -1
	}
	for ri := range line.runs {
		if s, _ := line.runs[ri].sourceExtent(); s >= 0 {
			return s
		}
	}
	return -1
}

// caret returns the content-space caret geometry (x, line top, line height)
// for a source offset. It reports false when no run in the layout covers the
// offset.
func (l *textLayout) caret(off int) (x, y, h float32, ok bool) {
	if l == nil || off < 0 {
		return 0, 0, 0, false
	}
	for i := range l.lines {
		for ri := range l.lines[i].runs {
			if cx, ok := caretXInRun(&l.lines[i].runs[ri], off); ok {
				return cx, l.lines[i].y, l.lines[i].h, true
			}
		}
	}
	return 0, 0, 0, false
}

// caretX returns just the caret's content-space x.
func (l *textLayout) caretX(off int) (float32, bool) {
	x, _, _, ok := l.caret(off)
	return x, ok
}

// lineIndexAtY returns the index of the visual line whose band contains y, or
// -1.
func (l *textLayout) lineIndexAtY(y float32) int {
	if l == nil {
		return -1
	}
	for i := range l.lines {
		if y >= l.lines[i].y && y < l.lines[i].y+l.lines[i].h {
			return i
		}
	}
	return -1
}

// rects returns the content-space rectangles covering the source range
// [a, b).
func (l *textLayout) rects(a, b int) []graphics.Rect {
	if l == nil {
		return nil
	}
	if a > b {
		a, b = b, a
	}
	if a < 0 || b <= a {
		return nil
	}
	var out []graphics.Rect
	for i := range l.lines {
		line := &l.lines[i]
		x0, x1 := float32(-1), float32(-1)
		for ri := range line.runs {
			run := &line.runs[ri]
			lo, hi := run.sourceExtent()
			if lo < 0 || hi <= a || lo >= b {
				continue
			}
			start, end := lo, hi
			if start < a {
				start = a
			}
			if end > b {
				end = b
			}
			rx0, ok0 := caretXInRun(run, start)
			rx1, ok1 := caretXInRun(run, end)
			if !ok0 {
				rx0 = run.x
			}
			if !ok1 {
				rx1 = run.x + run.w
			}
			if x0 < 0 || rx0 < x0 {
				x0 = rx0
			}
			if x1 < 0 || rx1 > x1 {
				x1 = rx1
			}
		}
		if x0 < 0 || x1 <= x0 {
			continue
		}
		out = append(out, graphics.NewRect(x0, line.y, x1-x0, line.h))
	}
	return out
}

// caretStops returns every source offset a caret can occupy, in order. Offsets
// inside text that has no glyph are skipped rather than guessed.
func (l *textLayout) caretStops() []int {
	if l == nil {
		return nil
	}
	var stops []int
	seen := map[int]bool{}
	add := func(o int) {
		if o < 0 || seen[o] {
			return
		}
		seen[o] = true
		stops = append(stops, o)
	}
	for i := range l.lines {
		for ri := range l.lines[i].runs {
			run := &l.lines[i].runs[ri]
			for k := 0; k < run.runeCount(); k++ {
				if k < len(run.runeSrc) && run.runeSrc[k] >= 0 {
					add(run.runeSrc[k])
				}
			}
			if s, e := run.sourceExtent(); e > 0 {
				if run.atomic {
					add(s)
				}
				add(e)
			}
		}
	}
	sort.Ints(stops)
	return stops
}

// snapAtomic moves an offset that falls inside an atomic run to the edge the
// selection is coming from (anchor).
func (l *textLayout) snapAtomic(off, anchor int) int {
	if l == nil {
		return off
	}
	for i := range l.lines {
		for ri := range l.lines[i].runs {
			run := &l.lines[i].runs[ri]
			if !run.atomic {
				continue
			}
			lo, hi := run.sourceExtent()
			if lo < 0 || off <= lo || off >= hi {
				continue
			}
			if anchor <= lo {
				return lo
			}
			return hi
		}
	}
	return off
}

// offsetInRun returns the source offset at content x inside a run, snapping to
// the nearest character boundary.
func offsetInRun(run *textRun, x float32) int {
	if run.atomic {
		if s, _ := run.sourceExtent(); s >= 0 {
			return s
		}
	}
	runes := []rune(run.text)
	if len(runes) == 0 {
		if s, _ := run.sourceExtent(); s >= 0 {
			return s
		}
		return -1
	}
	for i, r := range runes {
		// Measure the prefix as a string rather than summing per-rune widths:
		// shaping (kerning) makes the two differ, and the prefix width is what
		// the caret must line up with.
		left := run.font.SimpleWidth(string(runes[:i]))
		w := run.font.SimpleWidth(string(r))
		if x < run.x+left+w/2 {
			if i < len(run.runeSrc) && run.runeSrc[i] >= 0 {
				return run.runeSrc[i]
			}
			if s, _ := run.sourceExtent(); s >= 0 {
				return s
			}
			return -1
		}
	}
	if s, e := run.sourceExtent(); e > s {
		return e
	}
	return -1
}

// caretXInRun returns the content-space x of an offset inside a run. The caret
// sits to the left of the rune the offset falls in, and to the right of the
// last rune when the offset is that rune's end, so a boundary between two runes
// is not drawn twice.
func caretXInRun(run *textRun, off int) (float32, bool) {
	// An atomic object has no caret inside it: an offset that falls within it
	// places the caret at its left edge, and the selection snaps to whichever
	// edge it is coming from.
	if run.atomic {
		if lo, hi := run.sourceExtent(); lo >= 0 && off >= lo && off <= hi {
			return run.x, true
		}
		return 0, false
	}
	runes := []rune(run.text)
	for i, r := range runes {
		src := -1
		if i < len(run.runeSrc) {
			src = run.runeSrc[i]
		}
		if src < 0 {
			continue
		}
		if off <= src || off < src+utf8.RuneLen(r) {
			return run.x + run.font.SimpleWidth(string(runes[:i])), true
		}
	}
	if s, e := run.sourceExtent(); off >= s && off <= e {
		// Past the last mapped rune: the run's drawn end.
		return run.x + run.w, true
	}
	return 0, false
}

// ── word selection ────────────────────────────────────────────────

// Rune classes for word selection.
const (
	classSpace = iota
	classWord
	classPunct
)

func runeClass(r rune) int {
	switch {
	case unicode.IsSpace(r):
		return classSpace
	case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_':
		return classWord
	default:
		return classPunct
	}
}

// wordRangeAt returns the source range of the run of runes around off that
// share its class. A word also takes the whitespace that follows it, which is
// what a browser highlights when a word is double-clicked.
func wordRangeAt(src string, off int) (int, int) {
	if src == "" {
		return 0, 0
	}
	if off < 0 {
		off = 0
	}
	if off >= len(src) {
		off = len(src) - 1
	}
	for off > 0 && !utf8.RuneStart(src[off]) {
		off--
	}
	r, size := utf8.DecodeRuneInString(src[off:])
	class := runeClass(r)
	lo, hi := off, off+size
	for lo > 0 {
		pr, psize := utf8.DecodeLastRuneInString(src[:lo])
		if runeClass(pr) != class {
			break
		}
		lo -= psize
	}
	for hi < len(src) {
		nr, nsize := utf8.DecodeRuneInString(src[hi:])
		if runeClass(nr) != class {
			break
		}
		hi += nsize
	}
	if class == classWord {
		for hi < len(src) {
			nr, nsize := utf8.DecodeRuneInString(src[hi:])
			if runeClass(nr) != classSpace {
				break
			}
			hi += nsize
		}
	}
	return lo, hi
}

// ── line-level helpers (single-line text, as the editors hold it) ─

// identityOffsets returns 0..n-1: a run's "source" is then its rune index,
// which lets the run-based queries answer the editors' column questions.
func identityOffsets(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

// lineMeasure turns one line of text into advances. It is the single place a
// widget says how its text is measured: plain text measures its bytes, the
// code editor expands tabs first. Columns are byte offsets, the units the
// editors use throughout.
type lineMeasure struct {
	text     string
	font     graphics.Font
	tabWidth int
}

// width returns the advance of the first n bytes of the line.
func (m lineMeasure) width(n int) float32 {
	if n < 0 {
		n = 0
	}
	if n > len(m.text) {
		n = len(m.text)
	}
	s := m.text[:n]
	if m.tabWidth > 0 {
		s = expandTabs(s, m.tabWidth)
	}
	return m.font.SimpleWidth(s)
}

// caretX returns the content-space x of a caret at byte n.
func (m lineMeasure) caretX(n int) float32 { return m.width(n) }

// indexAt returns the byte index whose caret is nearest to content x: the
// nearest-boundary rule behind every pointer hit test. Only rune boundaries
// are candidates, so a click never lands inside a multi-byte rune, and the
// boundary before the first rune is considered too (a click in the left half
// of the first character belongs to column 0).
func (m lineMeasure) indexAt(x float32) int {
	if x <= 0 {
		return 0
	}
	best, bestDist := 0, x
	consider := func(i int) {
		d := x - m.width(i)
		if d < 0 {
			d = -d
		}
		if d < bestDist {
			best, bestDist = i, d
		}
	}
	for i := 1; i < len(m.text); i++ {
		if utf8.RuneStart(m.text[i]) {
			consider(i)
		}
	}
	consider(len(m.text))
	return best
}

// columnAtX returns the byte column at content x within one line of text,
// snapping to the nearest boundary.
func columnAtX(text string, font graphics.Font, x float32) int {
	return lineMeasure{text: text, font: font}.indexAt(x)
}

// caretXAtColumn returns the content-space x of a byte column in one line.
func caretXAtColumn(text string, font graphics.Font, col int) float32 {
	return lineMeasure{text: text, font: font}.caretX(col)
}

// ── document positions (lines without their newlines) ────────────

// docOffsetAt returns the byte offset of a (line, byte-column) position in a
// document held as lines. Out-of-range positions clamp to the document.
func docOffsetAt(lines []string, line, col int) int {
	if len(lines) == 0 {
		return 0
	}
	if line < 0 {
		line = 0
	}
	if line >= len(lines) {
		line = len(lines) - 1
	}
	if col < 0 {
		col = 0
	}
	if col > len(lines[line]) {
		col = len(lines[line])
	}
	off := 0
	for i := 0; i < line; i++ {
		off += len(lines[i]) + 1
	}
	return off + col
}

// docLineColAt returns the (line, byte-column) of a byte offset. An offset past
// the end clamps to the end of the document.
func docLineColAt(lines []string, off int) (line, col int) {
	if len(lines) == 0 || off <= 0 {
		return 0, 0
	}
	remaining := off
	for i, l := range lines {
		if remaining <= len(l) {
			return i, remaining
		}
		remaining -= len(l) + 1
	}
	last := len(lines) - 1
	return last, len(lines[last])
}

// expandTabs replaces tabs with spaces up to the next tab stop.
func expandTabs(line string, tabWidth int) string {
	if !strings.ContainsRune(line, '\t') {
		return line
	}
	var result strings.Builder
	col := 0
	for _, ch := range line {
		if ch == '\t' {
			spaces := tabWidth - (col % tabWidth)
			for i := 0; i < spaces; i++ {
				result.WriteByte(' ')
			}
			col += spaces
		} else {
			result.WriteRune(ch)
			col++
		}
	}
	return result.String()
}

// wrapText wraps one line of text into visual lines no wider than width.
func wrapText(text string, font graphics.Font, width float32) []string {
	if text == "" {
		return []string{""}
	}
	lines := wrapRuns([]textRun{{
		text: text, font: font, w: font.SimpleWidth(text), h: font.LineHeight(),
	}}, width, font.LineHeight(), 0, 0)
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		var sb strings.Builder
		for _, r := range ln.runs {
			sb.WriteString(r.text)
		}
		out = append(out, sb.String())
	}
	return out
}

// ── wrapping ──────────────────────────────────────────────────────

// wrapRuns lays runs out into lines no wider than width, breaking between
// runes (the editors' soft wrap). An atomic run is never split: if it does not
// fit on a non-empty line it starts the next one. The returned lines have their
// runs positioned from x0 and y0, one line height apart.
func wrapRuns(runs []textRun, width, lineH, x0, y0 float32) []textLine {
	if width <= 0 {
		width = 1
	}
	var lines []textLine
	var cur textLine
	cur.y = y0
	cur.h = lineH
	x := x0
	flush := func() {
		if len(cur.runs) > 0 {
			lines = append(lines, cur)
		}
		cur = textLine{y: y0 + float32(len(lines))*lineH, h: lineH}
		x = x0
	}
	for _, run := range runs {
		runes := []rune(run.text)
		if run.atomic {
			w := run.font.SimpleWidth(run.text)
			if x > x0 && x+w > x0+width {
				flush()
			}
			r := run
			r.x, r.y, r.w, r.h = x, cur.y, w, lineH
			cur.runs = append(cur.runs, r)
			x += w
			continue
		}
		start := 0
		for start < len(runes) {
			// Take as many runes as fit on the remaining width, measuring the
			// prefix as a string (kerning makes summed per-rune widths differ).
			end := start
			w := float32(0)
			for end < len(runes) {
				pw := run.font.SimpleWidth(string(runes[start : end+1]))
				if x+pw > x0+width && end > start {
					break
				}
				w = pw
				end++
			}
			if end == start {
				end = start + 1
				w = run.font.SimpleWidth(string(runes[start]))
			}
			part := run
			part.text = string(runes[start:end])
			part.x, part.y, part.w, part.h = x, cur.y, w, lineH
			if len(run.runeSrc) == len(runes) {
				part.runeSrc = run.runeSrc[start:end]
			} else {
				part.runeSrc = nil
			}
			if part.srcStart >= 0 && len(part.runeSrc) == 0 {
				part.srcStart, part.srcEnd = -1, -1
			}
			cur.runs = append(cur.runs, part)
			x += w
			start = end
			if start < len(runes) {
				flush()
			}
		}
	}
	if len(cur.runs) > 0 {
		lines = append(lines, cur)
	}
	if len(lines) == 0 {
		lines = append(lines, textLine{y: y0, h: lineH})
	}
	return lines
}

// ── Label wrapping ────────────────────────────────────────────────
//
// A Label's wrapping lives HERE, with the rest of the text geometry, not on the
// widget: AGENTS.md is explicit that a widget must not own a second width or wrap
// implementation. The strategies differ (word boundaries for prose, rune soft-wrap
// for an editor's columns) but they are chosen in one place, next to the metrics
// they measure with.
//
//	mode      WrapNoWrap | WrapWordWrap | WrapEllipsis
//	maxLines  0 = as many as it takes; otherwise clamp and ellipsize
func wrapLabelText(font graphics.Font, text string, maxWidth float32, mode string, maxLines int) []string {
	if text == "" {
		return []string{""}
	}
	if mode == WrapNoWrap || maxWidth <= 0 {
		return clampTextLines(font, strings.Split(text, "\n"), maxWidth, maxLines)
	}
	if mode == WrapEllipsis {
		if font.SimpleWidth(text) <= maxWidth {
			return clampTextLines(font, strings.Split(text, "\n"), maxWidth, maxLines)
		}
		return []string{truncateWithEllipsis(font, text, maxWidth)}
	}
	// wordWrap — preserve intentional \n line breaks
	var lines []string
	for _, segment := range strings.Split(text, "\n") {
		words := strings.Fields(segment)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		currentLine := words[0]
		for _, word := range words[1:] {
			testLine := currentLine + " " + word
			if font.SimpleWidth(testLine) <= maxWidth {
				currentLine = testLine
			} else {
				lines = append(lines, currentLine)
				currentLine = word
			}
		}
		lines = append(lines, currentLine)
	}
	return clampTextLines(font, lines, maxWidth, maxLines)
}

// truncateWithEllipsis shortens a line to fit maxWidth with a trailing "…" — the single
// character a browser uses, not three periods. It steps by RUNES: a byte-at-a-time
// truncation splits a multi-byte character and leaves invalid UTF-8 (a defect this tree
// has already fixed once in the editors).
func truncateWithEllipsis(font graphics.Font, text string, maxWidth float32) string {
	const ellipsis = "…"
	available := maxWidth - font.SimpleWidth(ellipsis)
	if available <= 0 {
		return ellipsis
	}
	runes := []rune(text)
	for len(runes) > 0 && font.SimpleWidth(string(runes)) > available {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + ellipsis
}

// clampTextLines applies a maximum line count: keep the first n and ellipsize the
// last kept one when anything was dropped.
func clampTextLines(font graphics.Font, lines []string, maxWidth float32, maxLines int) []string {
	if maxLines <= 0 || len(lines) <= maxLines {
		return lines
	}
	out := append([]string(nil), lines[:maxLines]...)
	out[maxLines-1] = truncateWithEllipsis(font, out[maxLines-1], maxWidth)
	return out
}
