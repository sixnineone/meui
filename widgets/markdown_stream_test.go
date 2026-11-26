package widgets

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
)

func newStreamedDoc(sections int) string {
	var sb strings.Builder
	for i := 0; i < sections; i++ {
		fmt.Fprintf(&sb, "## Section %d\n\nProse for section %d with **bold** text.\n\n", i, i)
	}
	return sb.String()
}

// TestAppendEqualsSetText is the correctness half: an appended document must be
// indistinguishable from one that was set in one call. The partial walk resumes past the
// children it kept, and a mistake there drops a block or doubles one — neither of which a
// screenshot of the tail would show.
func TestAppendEqualsSetText(t *testing.T) {
	for _, sections := range []int{1, 5, 20} {
		full := newStreamedDoc(sections)
		whole := NewMarkdownView()
		whole.SetText(full)
		whole.computeContentH()

		streamed := NewMarkdownView()
		for i := 0; i < sections; i++ {
			streamed.Append(fmt.Sprintf("## Section %d\n\nProse for section %d with **bold** text.\n\n", i, i))
			streamed.ensureParsed() // one append per "frame"
		}
		streamed.computeContentH()

		if len(streamed.blocks) != len(whole.blocks) {
			t.Errorf("%d sections: streamed produced %d blocks, whole produced %d",
				sections, len(streamed.blocks), len(whole.blocks))
			continue
		}
		for i := range whole.blocks {
			a, b := streamed.blocks[i], whole.blocks[i]
			if a.Type != b.Type || a.SrcStart != b.SrcStart || a.SrcEnd != b.SrcEnd {
				t.Errorf("%d sections: block %d differs: %+v vs %+v", sections, i, a, b)
				break
			}
		}
		if streamed.contentH != whole.contentH {
			t.Errorf("%d sections: content height %.1f vs %.1f", sections, streamed.contentH, whole.contentH)
		}
	}
}

// TestAppendCoalescesWithinAFrame is the performance contract, and it is the whole
// reason Append does not parse eagerly: twenty tokens in one frame must cost ONE walk.
// Measured on a document already long enough for a per-token walk to be visible.
func TestAppendCoalescesWithinAFrame(t *testing.T) {
	mv := NewMarkdownView()
	mv.SetText(newStreamedDoc(50))
	mv.ensureParsed()
	base := len(mv.blocks)

	for i := 0; i < 20; i++ {
		mv.Append("more ")
	}
	if len(mv.blocks) != base {
		t.Errorf("20 appends walked the document eagerly: %d blocks, want %d", len(mv.blocks), base)
	}
	mv.ensureParsed()
	if len(mv.blocks) <= base {
		t.Error("the coalesced parse did not run")
	}
}

// TestAppendIntoAnUnterminatedBlockRebuildsIt is the case the resume rule's `>=` exists
// for: an open fenced code block ends where the text is appended and GROWS, so treating
// it as unchanged would drop the appended lines.
func TestAppendIntoAnUnterminatedBlockRebuildsIt(t *testing.T) {
	mv := NewMarkdownView()
	mv.SetText("```go\nfunc first() {}\n")
	mv.ensureParsed()
	mv.Append("func second() {}\n```\n")
	mv.ensureParsed()
	found := false
	for _, b := range mv.blocks {
		if b.Type == 2 && strings.Contains(b.CodeText, "second") {
			found = true
		}
	}
	if !found {
		t.Error("appending the rest of an open code fence did not grow the block")
	}
}

// TestStreamingFollowsTheTail is what SetStreaming buys: the viewport ends up at the
// bottom after an append, and does NOT move when it is off — a document a reader is
// halfway down must not scroll itself.
func TestStreamingFollowsTheTail(t *testing.T) {
	doc := newStreamedDoc(40)
	mv := NewMarkdownView()
	mv.SetFrameRect(graphics.NewRect(0, 0, 400, 200))
	mv.SetText(doc)
	mv.computeContentH()

	mv.SetStreaming(false)
	mv.scrollOff = 0
	mv.Append("\n## One more\n\nTail text.\n")
	mv.ensureParsed()
	if mv.scrollOff != 0 {
		t.Errorf("an ordinary document scrolled itself to %.0f while not streaming", mv.scrollOff)
	}

	mv.SetStreaming(true)
	mv.scrollOff = 0
	mv.Append("\n## And another\n\nMore tail.\n")
	mv.ensureParsed()
	if mv.scrollOff <= 0 {
		t.Error("a streaming view did not follow the tail")
	}
}

// BenchmarkMarkdownAppend measures the streaming path against the SetText it replaces.
func BenchmarkMarkdownAppend(b *testing.B) {
	doc := newStreamedDoc(50)
	mv := NewMarkdownView()
	mv.SetFrameRect(graphics.NewRect(0, 0, 800, 600))
	mv.SetText(doc)
	mv.ensureParsed()

	b.Run("append_one_token", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			mv.Append("x")
			mv.ensureParsed()
		}
	})
	b.Run("append_one_section_boundary", func(b *testing.B) {
		// A streamed answer arrives as sections, and a section starts at a block
		// boundary — so this is the path that can parse the TAIL rather than the whole
		// document, and the one worth measuring.
		for i := 0; i < b.N; i++ {
			mv.Append(fmt.Sprintf("\n## S%d\n\nProse.\n", i))
			mv.ensureParsed()
		}
	})
	b.Run("settext_whole_document", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			mv.SetText(doc)
		}
	})
	_ = platform.FormFactorPhone
}
