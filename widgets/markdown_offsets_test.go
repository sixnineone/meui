package widgets

import (
	"sort"
	"strings"
	"testing"
)

// Sentinel words are placed inside every construct. The test asserts each one
// lands inside a block's source span, which is the contract the selection core
// will build on: every piece of visible content maps back to markdown source.
const offsetsCorpus = `# Heading WALNUT

Paragraph with **bold HAZEL** and *italic PECAN* and ` + "`code ALMOND`" + ` and [link CASHEW](https://example.com/x) plus ~~strike PISTACHIO~~.

> quoted MACADAMIA

- item BRAZIL
- [x] done PECAN2
- [ ] todo HAZEL2

1. first CASHEW2

~~~go
func main() { println("ALMOND2") }
~~~

    indented code BRAZIL2

| Col A | Col B |
| --- | --- |
| cell PISTACHIO2 | cell MACADAMIA2 |

Term WALNUT3
: Definition HAZEL3

Footnote reference[^1] PECAN3

[^1]: Footnote text ALMOND3

![alt CASHEW3](img.png)

Inline <span>raw BRAZIL3</span> text

---

Auto link https://example.com/autolink PISTACHIO3
`

var offsetsSentinels = []string{
	"WALNUT", "HAZEL", "PECAN", "ALMOND", "CASHEW", "PISTACHIO", "MACADAMIA",
	"BRAZIL", "PECAN2", "HAZEL2", "CASHEW2", "ALMOND2", "BRAZIL2",
	"PISTACHIO2", "MACADAMIA2", "WALNUT3", "HAZEL3", "PECAN3", "ALMOND3",
	"BRAZIL3", "PISTACHIO3",
}

func parseMarkdownOffsets(t *testing.T, text string) (*MarkdownView, []byte) {
	t.Helper()
	mv := NewMarkdownView()
	mv.SetText(text)
	src := []byte(text)
	if got := mv.Text(); got != text {
		t.Fatalf("Text() round-trip changed the source")
	}
	return mv, src
}

// TestMarkdownBlockSpansAreWellFormed pins the basic span invariants: known
// spans lie inside the source, are non-empty, and never overlap each other.
// (Block *emission* order is not source order — goldmark hoists footnote
// definitions to the end — so the check sorts by start offset.)
func TestMarkdownBlockSpansAreWellFormed(t *testing.T) {
	mv, src := parseMarkdownOffsets(t, offsetsCorpus)
	if len(mv.blocks) == 0 {
		t.Fatal("no blocks parsed")
	}
	type span struct {
		start, end, index int
	}
	var spans []span
	for i, b := range mv.blocks {
		if !b.hasSrcSpan() {
			continue
		}
		if b.SrcStart < 0 || b.SrcEnd > len(src) {
			t.Fatalf("block %d (%d): span [%d,%d) outside source (len %d)", i, b.Type, b.SrcStart, b.SrcEnd, len(src))
		}
		spans = append(spans, span{b.SrcStart, b.SrcEnd, i})
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	prevEnd, prevIndex := -1, -1
	for _, s := range spans {
		if s.start < prevEnd {
			t.Fatalf("block %d span [%d,%d) overlaps block %d (ended at %d)", s.index, s.start, s.end, prevIndex, prevEnd)
		}
		prevEnd, prevIndex = s.end, s.index
	}
}

// TestMarkdownSentinelsLandInBlockSpans is the functional half: every visible
// sentinel must be inside some known block span, so a block-level selection can
// recover it from the source.
func TestMarkdownSentinelsLandInBlockSpans(t *testing.T) {
	mv, src := parseMarkdownOffsets(t, offsetsCorpus)
	for _, word := range offsetsSentinels {
		at := strings.Index(offsetsCorpus, word)
		if at < 0 {
			t.Fatalf("sentinel %q missing from the corpus", word)
		}
		found := false
		for _, b := range mv.blocks {
			if !b.hasSrcSpan() {
				continue
			}
			if strings.Contains(string(src[b.SrcStart:b.SrcEnd]), word) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("sentinel %q at offset %d is in no block span — source mapping lost that content", word, at)
		}
	}
}

// TestMarkdownNodeSpansNestInBlocks checks that inline node spans stay inside
// their block and advance through the block in order. renderNode holds one
// node per rendered rune, so consecutive nodes legitimately share a span.
func TestMarkdownNodeSpansNestInBlocks(t *testing.T) {
	mv, src := parseMarkdownOffsets(t, offsetsCorpus)
	known := 0
	for bi, b := range mv.blocks {
		if !b.hasSrcSpan() {
			continue
		}
		prevStart, prevEnd := b.SrcStart, b.SrcStart
		for ni, rn := range b.Nodes {
			if !rn.hasSrcSpan() {
				continue
			}
			known++
			if rn.SrcStart < b.SrcStart || rn.SrcEnd > b.SrcEnd {
				t.Fatalf("block %d node %d (%q): span [%d,%d) escapes its block [%d,%d)", bi, ni, rn.Text, rn.SrcStart, rn.SrcEnd, b.SrcStart, b.SrcEnd)
			}
			if rn.SrcStart < prevStart || rn.SrcEnd < prevEnd {
				t.Fatalf("block %d node %d (%q): span [%d,%d) goes backwards from [%d,%d)", bi, ni, rn.Text, rn.SrcStart, rn.SrcEnd, prevStart, prevEnd)
			}
			if rn.SrcEnd > len(src) {
				t.Fatalf("block %d node %d: span past end of source", bi, ni)
			}
			prevStart, prevEnd = rn.SrcStart, rn.SrcEnd
		}
	}
	if known == 0 {
		t.Fatal("no inline node spans were populated — offsets never reached the nodes")
	}
}

// TestMarkdownPlainProseHasNoUnknownSpans narrows the guarantee: ordinary prose
// with emphasis, code, links and strikethrough has an exact span on every node.
func TestMarkdownPlainProseHasNoUnknownSpans(t *testing.T) {
	plain := "Plain APPLE text with **bold BANANA** and *italic CHERRY* and `code DAMSON` and [link ELDER](x.md) and ~~strike FIG~~.\n"
	mv, _ := parseMarkdownOffsets(t, plain)
	nodes := 0
	for _, b := range mv.blocks {
		for _, rn := range b.Nodes {
			nodes++
			if !rn.hasSrcSpan() {
				t.Errorf("node %q has no source span in plain prose", rn.Text)
			}
		}
	}
	if nodes == 0 {
		t.Fatal("no nodes parsed")
	}
}

// TestMarkdownSyntheticNodesAreUnknownButContained documents the deliberate
// gap: rendered glyphs with no source text of their own (task checkboxes,
// footnote markers) carry no span, while their text still maps through the
// enclosing block.
func TestMarkdownSyntheticNodesAreUnknownButContained(t *testing.T) {
	text := "- [x] done GINGER\n- [ ] todo NUTMEG\n\nref[^1] CLOVE\n\n[^1]: note SESAME\n"
	mv, _ := parseMarkdownOffsets(t, text)
	glyphs := 0
	for _, b := range mv.blocks {
		for _, rn := range b.Nodes {
			if rn.Task {
				glyphs++
				if rn.hasSrcSpan() {
					t.Errorf("task glyph %q claims source span [%d,%d)", rn.Text, rn.SrcStart, rn.SrcEnd)
				}
			}
		}
	}
	if glyphs != 2 {
		t.Fatalf("expected 2 task glyph nodes, got %d", glyphs)
	}
	for _, word := range []string{"GINGER", "NUTMEG", "CLOVE", "SESAME"} {
		if !blockSpanContains(mv, word) {
			t.Errorf("sentinel %q is in no block span", word)
		}
	}
}

func blockSpanContains(mv *MarkdownView, word string) bool {
	src := []byte(mv.Text())
	for _, b := range mv.blocks {
		if b.hasSrcSpan() && strings.Contains(string(src[b.SrcStart:b.SrcEnd]), word) {
			return true
		}
	}
	return false
}

// TestMarkdownRuneSpansAreExact pins the per-character contract: for text whose
// source and render agree byte for byte, each rune's span covers exactly that
// rune in the source.
func TestMarkdownRuneSpansAreExact(t *testing.T) {
	const plain = "Plain APPLE and pear PIE.\n"
	mv, src := parseMarkdownOffsets(t, plain)
	if len(mv.blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(mv.blocks))
	}
	for _, rn := range mv.blocks[0].Nodes {
		rs, re := runeSrcSpan(rn, 0)
		if rs < 0 || re-rs != len(rn.Text) {
			t.Fatalf("rune %q: span [%d,%d) does not cover %d byte(s)", rn.Text, rs, re, len(rn.Text))
		}
		if got := string(src[rs:re]); got != rn.Text {
			t.Fatalf("rune %q maps to source %q", rn.Text, got)
		}
	}
}

// TestMarkdownRuneSpansSurviveEscapesAndEntities checks that a shorter rendered
// text still lines up: the source is longer, and a substitution must not desync
// the runes after it.
func TestMarkdownRuneSpansSurviveEscapesAndEntities(t *testing.T) {
	const text = "A \\*star\\* &amp; B\u00e9\n"
	mv, src := parseMarkdownOffsets(t, text)
	b := mv.blocks[0]
	var rendered strings.Builder
	for _, rn := range b.Nodes {
		rendered.WriteString(rn.Text)
	}
	if got := rendered.String(); got != "A *star* & Bé" {
		t.Fatalf("rendered text = %q, want %q", got, "A *star* & Bé")
	}
	// The 'B' must still point at the source 'B', proving alignment held.
	last := b.Nodes[len(b.Nodes)-2] // "B" (the last node is "é")
	if last.Text != "B" {
		t.Fatalf("expected the second-to-last node to be B, got %q", last.Text)
	}
	rs, _ := runeSrcSpan(last, 0)
	if string(src[rs:rs+1]) != "B" {
		t.Fatalf("B maps to %q — alignment desynced after the entity", string(src[rs:rs+1]))
	}
	// The '&' rune maps into the middle of "&amp;".
	for _, rn := range b.Nodes {
		if rn.Text == "&" {
			as, _ := runeSrcSpan(rn, 0)
			if string(src[as:as+5]) != "&amp;" {
				t.Fatalf("& maps to %q, not the entity", string(src[as:as+5]))
			}
		}
	}
}

// TestMarkdownRuneSpansSkipExtraMarkers checks that inline extras map their
// content, not their markers: in "x ==hi== y" the "hi" runes must sit on the
// "hi" bytes, and the runes after them must sit past the closing "==".
func TestMarkdownRuneSpansSkipExtraMarkers(t *testing.T) {
	const text = "x ==hi== y\n"
	mv, src := parseMarkdownOffsets(t, text)
	b := mv.blocks[0]
	var hi, y renderNode
	for _, rn := range b.Nodes {
		switch rn.Text {
		case "hi":
			hi = rn
		case "y":
			y = rn
		}
	}
	if !hi.Highlight || !hi.hasSrcSpan() {
		t.Fatalf("highlight node = %+v", hi)
	}
	hs, _ := runeSrcSpan(hi, 0)
	_, he := runeSrcSpan(hi, 1)
	if got := string(src[hs:he]); got != "hi" {
		t.Fatalf("highlight content maps to %q, want %q", got, "hi")
	}
	ys, _ := runeSrcSpan(y, 0)
	if string(src[ys:ys+1]) != "y" {
		t.Fatalf("rune after the highlight maps to %q — the closing marker was not skipped", string(src[ys:ys+1]))
	}
}

// TestMarkdownRuneSpansAdvance checks that per-rune spans never go backwards
// inside a block.
func TestMarkdownRuneSpansAdvance(t *testing.T) {
	mv, _ := parseMarkdownOffsets(t, offsetsCorpus)
	for bi, b := range mv.blocks {
		prev := -1
		for _, rn := range b.Nodes {
			for ri := range []rune(rn.Text) {
				rs, _ := runeSrcSpan(rn, ri)
				if rs < 0 {
					continue
				}
				if rs < prev {
					t.Fatalf("block %d rune %d of %q maps to %d, before previous %d", bi, ri, rn.Text, rs, prev)
				}
				prev = rs
			}
		}
	}
}
