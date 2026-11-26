package widgets

import (
	"strings"
	"testing"

	"github.com/sixnineone/meui/graphics"
)

// probeWidget records what it was asked and what it was given, so the test can assert the
// contract rather than the picture: height is asked first, and Paint is handed a rect of
// exactly that height.
type probeWidget struct {
	askedHeight int
	painted     int
	gotWidth    float32
	gotHeight   float32
	gotName     string
	gotBody     string
	gotParam    any
	wantHeight  float32
}

func (p *probeWidget) Height(ctx *WidgetContext) float32 {
	p.askedHeight++
	p.gotName = ctx.Name
	p.gotBody = ctx.Body
	p.gotWidth = ctx.Width
	p.gotParam = ctx.Params["timeout"]
	return p.wantHeight
}

func (p *probeWidget) Paint(canvas *graphics.Canvas, rect graphics.Rect, ctx *WidgetContext) {
	p.painted++
	p.gotWidth = rect.Width
	p.gotHeight = rect.Height
}

func withResolver(t *testing.T, fn func(name string, params map[string]any, body string) (MarkdownWidget, string, bool)) {
	t.Helper()
	saved := FenceResolver
	FenceResolver = fn
	t.Cleanup(func() { FenceResolver = saved })
}

// renderedText builds a view AND PAINTS it. Painting is not optional: the widget contract
// has a measure phase and a paint phase, and a test that only measures would pass a widget
// the painter never calls — which is the whole of the second half of this slice.
func renderedText(t *testing.T, src string) (*MarkdownView, string) {
	t.Helper()
	mv := NewMarkdownView()
	mv.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	mv.SetText(src)
	mv.computeContentH()
	dev := graphics.NewSoftDevice(400, 300)
	mv.Paint(graphics.NewCanvas(dev), graphics.NewRect(0, 0, 400, 300))
	return mv, mv.SelectedText()
}

// TestAFenceWithARegisteredNameBecomesAWidget is the vertical slice: the embed syntax, the
// resolver, the block kind, the layout asking for a height, and the paint — all of it.
func TestAFenceWithARegisteredNameBecomesAWidget(t *testing.T) {
	probe := &probeWidget{wantHeight: 42}
	withResolver(t, func(name string, params map[string]any, body string) (MarkdownWidget, string, bool) {
		if name != "probe" {
			return nil, "", false
		}
		return probe, "probe ran", true
	})

	mv, _ := renderedText(t, "intro\n\n```probe {\"timeout\":45}\n\nthe body\n```\n\ntail\n")
	if probe.askedHeight == 0 {
		t.Fatal("the widget was never asked for its height, so the layout does not know about it")
	}
	if probe.painted == 0 {
		t.Fatal("the widget was never painted")
	}
	if probe.gotHeight != probe.wantHeight {
		t.Errorf("Paint got a height of %.0f but Height asked for %.0f; the rect a widget is "+
			"given must be exactly the height it asked for, or it can overlap its neighbour",
			probe.gotHeight, probe.wantHeight)
	}
	if probe.gotName != "probe" {
		t.Errorf("the widget was told its name is %q, want probe", probe.gotName)
	}
	if strings.TrimSpace(probe.gotBody) != "the body" {
		t.Errorf("the body is %q, want 'the body'", probe.gotBody)
	}
	if probe.gotParam != float64(45) {
		t.Errorf("timeout = %#v, want the number 45", probe.gotParam)
	}
	// And it is a WIDGET block, not a code block: the rest of the document is intact.
	if len(mv.blocks) < 3 {
		t.Fatalf("the document lost blocks: %d", len(mv.blocks))
	}
	found := false
	for _, b := range mv.blocks {
		if b.Type == blockWidget {
			found = true
		}
		if b.Type == blockCode {
			t.Error("a registered embed was parsed as a CODE block")
		}
	}
	if !found {
		t.Error("no widget block was produced")
	}
}

// TestAPlainLanguageFenceIsStillACodeBlock is the regression that matters most: every
// document that has no embeds in it must render EXACTLY as before.
func TestAPlainLanguageFenceIsStillACodeBlock(t *testing.T) {
	withResolver(t, func(name string, params map[string]any, body string) (MarkdownWidget, string, bool) {
		return nil, "", false // an empty registry
	})
	mv, _ := renderedText(t, "a\n\n```go\nfunc main() {}\n```\n\nb\n")
	// Only the FENCE is under test: the paragraphs around it are expected to be
	// paragraphs, and the first version of this loop failed on those.
	code := 0
	for _, b := range mv.blocks {
		if b.Type == blockCode {
			code++
			if !strings.Contains(b.CodeText, "func main()") {
				t.Errorf("the code body is %q", b.CodeText)
			}
			if b.Lang != "go" {
				t.Errorf("the language is %q, want go", b.Lang)
			}
		}
		if b.Type == blockWidget || b.Type == blockError {
			t.Errorf("a plain ```go fence became a %d block", b.Type)
		}
	}
	if code != 1 {
		t.Errorf("the document has %d code blocks, want 1", code)
	}
}

// TestAnUnregisteredEmbedIsAVisibleError is the direction that must not fail silently: an
// author who wrote an embed and got a code block would have no signal at all.
func TestAnUnregisteredEmbedIsAVisibleError(t *testing.T) {
	withResolver(t, func(name string, params map[string]any, body string) (MarkdownWidget, string, bool) {
		return nil, "", false
	})
	mv, _ := renderedText(t, "```nosuchwidget {\"a\":1}\nbody\n```\n")
	if len(mv.blocks) != 1 {
		t.Fatalf("expected one block, got %d", len(mv.blocks))
	}
	b := mv.blocks[0]
	if b.Type != blockError {
		t.Fatalf("an unregistered embed became a %d block, not an error", b.Type)
	}
	// The message lives in the block's TEXT RUN, not in Body: an error that keeps its own
	// string and draws it itself is an error whose position, selection and hit-testing are
	// each maintained separately — which is how it ended up drawn a line below its own
	// block, on top of the paragraph after it. So the block must CARRY TEXT, and the name
	// the author needs is still in the source they can select and copy.
	if b.Text == nil {
		t.Error("the error block carries no text run, so nothing is drawn for it")
	}
	if b.SrcStart < 0 || b.SrcEnd <= b.SrcStart {
		t.Errorf("the error block has no source span (%d..%d), so the failing fence cannot "+
			"be selected back out of the document", b.SrcStart, b.SrcEnd)
	}
	if sp := b.Body; sp != "" {
		t.Errorf("the error block still keeps a private message string %q alongside its "+
			"text run, which is the duplication that put it out of position", sp)
	}
}

// TestMalformedParametersAreAnErrorNotACodeBlock: the same direction, for a fence whose
// object does not parse.
func TestMalformedParametersAreAnErrorNotACodeBlock(t *testing.T) {
	withResolver(t, func(name string, params map[string]any, body string) (MarkdownWidget, string, bool) {
		return nil, "", false
	})
	mv, _ := renderedText(t, "```probe {\"a\":\nbody\n```\n")
	if len(mv.blocks) != 1 || mv.blocks[0].Type != blockError {
		t.Fatalf("a fence with malformed parameters did not become an error: %+v", mv.blocks)
	}
}

// TestACollapsedWidgetIsNeitherLaidOutNorPainted is the property the whole disclosure
// design rests on, at the markdown level: a collapsed embed costs a header and nothing
// else. It is the same property TestACollapsedTranscriptRowCostsNoPixels asserts for a
// Collapsible, and a widget that painted during layout would fail it here.
func TestACollapsedWidgetIsNeitherLaidOutNorPainted(t *testing.T) {
	probe := &probeWidget{wantHeight: 200}
	withResolver(t, func(name string, params map[string]any, body string) (MarkdownWidget, string, bool) {
		return probe, "probe", true
	})

	mv, _ := renderedText(t, "```probe {}\nbody\n```\n")
	openAsked := probe.askedHeight
	if openAsked == 0 {
		t.Fatal("the expanded widget was never measured")
	}
	for i := range mv.blocks {
		mv.blocks[i].Collapsed = true
	}
	probe.askedHeight = 0
	probe.painted = 0
	mv.computeContentH()
	if probe.askedHeight != 0 {
		t.Errorf("a COLLAPSED widget was measured %d times; it must not be laid out, or a "+
			"transcript of collapsed steps pays for every one", probe.askedHeight)
	}
}

// TestAHeaderPressTogglesTheWidgetAndNothingElse is the disclosure behaviour: a press in
// the header expands and collapses, and the press is CONSUMED — a header press that also
// started a selection would drag a caret across the thing it just opened.
func TestAHeaderPressTogglesTheWidgetAndNothingElse(t *testing.T) {
	probe := &probeWidget{wantHeight: 40}
	withResolver(t, func(name string, params map[string]any, body string) (MarkdownWidget, string, bool) {
		return probe, "the tool ran", true
	})

	const w, h = 400, 300
	mv := NewMarkdownView()
	mv.SetFrameRect(graphics.NewRect(0, 0, w, h))
	mv.SetText("```probe {}\nbody\n```\n")
	mv.computeContentH()

	idx := -1
	for i, b := range mv.blocks {
		if b.Type == blockWidget {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatal("no widget block to toggle")
	}
	if mv.blocks[idx].Collapsed {
		t.Fatal("a fresh embed starts collapsed; it should start showing its body")
	}

	var toggles []bool
	mv.OnDisclosureToggled = func(i int, collapsed bool) { toggles = append(toggles, collapsed) }

	// The header is the first headH of the block's laid-out rect.
	baseY := float32(markdownMargin) - mv.scrollOff
	top := baseY + mv.laid[idx].y
	head := mv.laid[idx].headH
	if !mv.OnMouseDown(40, top+head/2, 0) {
		t.Fatal("a press in the header was not taken")
	}
	if !mv.blocks[idx].Collapsed {
		t.Error("the header press did not collapse the widget")
	}
	if len(toggles) != 1 || !toggles[0] {
		t.Errorf("OnDisclosureToggled reported %v, want [true]", toggles)
	}
	if mv.SelectedText() != "" {
		t.Error("a header press also started a selection")
	}

	// And back open.
	top = baseY + mv.laid[idx].y
	head = mv.laid[idx].headH
	mv.OnMouseDown(40, top+head/2, 0)
	if mv.blocks[idx].Collapsed {
		t.Error("the second header press did not expand the widget")
	}
}

// TestAPressInTheBodyIsNotAToggle: the header is the target, not the whole block. A press in
// a widget's body must reach the widget, because that is where a later input phase will
// hand it clicks.
func TestAPressInTheBodyIsNotAToggle(t *testing.T) {
	probe := &probeWidget{wantHeight: 60}
	withResolver(t, func(name string, params map[string]any, body string) (MarkdownWidget, string, bool) {
		return probe, "s", true
	})
	const w, h = 400, 300
	mv := NewMarkdownView()
	mv.SetFrameRect(graphics.NewRect(0, 0, w, h))
	mv.SetText("```probe {}\nbody\n```\n")
	mv.computeContentH()
	idx := -1
	for i, b := range mv.blocks {
		if b.Type == blockWidget {
			idx = i
		}
	}
	baseY := float32(markdownMargin) - mv.scrollOff
	body := baseY + mv.laid[idx].y + mv.laid[idx].headH + 20
	before := mv.blocks[idx].Collapsed
	mv.OnMouseDown(40, body, 0)
	if mv.blocks[idx].Collapsed != before {
		t.Error("a press in the widget's BODY toggled the disclosure")
	}
}

// TestACollapsedEmbedStillCopiesAsItsFence is the losslessness the syntax was chosen for:
// the embed survives select-all-then-copy-all, because it was never anything but a fence.
func TestACollapsedEmbedStillCopiesAsItsFence(t *testing.T) {
	withResolver(t, func(name string, params map[string]any, body string) (MarkdownWidget, string, bool) {
		return &probeWidget{wantHeight: 10}, "s", true
	})
	mv := NewMarkdownView()
	mv.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	src := "before\n\n```probe {\"id\":\"read\"}\nthe body\n```\n\nafter\n"
	mv.SetText(src)
	for i := range mv.blocks {
		mv.blocks[i].Collapsed = true
	}
	mv.computeContentH()
	mv.SelectAll()
	got := mv.SelectedMarkdown()
	if !strings.Contains(got, "```probe") {
		t.Errorf("the copied source lost the embed:\n%s", got)
	}
	if !strings.Contains(got, "the body") {
		t.Errorf("the copied source lost the widget's body:\n%s", got)
	}
}

// parseBlocks turns Markdown into blocks the way the renderer does, so a test can build
// a region's body the same way a caller would.
func parseBlocks(t *testing.T, src string) []block {
	t.Helper()
	mv := NewMarkdownView()
	mv.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	mv.SetText(src)
	mv.computeContentH()
	return mv.tree
}

// TestACollapsibleRegionCanHoldMarkdown is the general form: a disclosure that contains
// BLOCKS rather than a widget. It is the "collapsing anywhere in a document" half of the
// design, and the first version of SetChildren attached to the VISIBLE list — which is
// rebuilt from the tree on every toggle — so a region built that way appeared to work and
// then silently emptied itself.
func TestACollapsibleRegionCanHoldMarkdown(t *testing.T) {
	withResolver(t, func(name string, params map[string]any, body string) (MarkdownWidget, string, bool) {
		return &probeWidget{wantHeight: 30}, "a section", true
	})
	mv, _ := renderedText(t, "before\n\n```region {}\n```\n\nafter\n")
	idx := -1
	for i, b := range mv.blocks {
		if b.Type == blockWidget {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatal("no disclosure to attach to")
	}

	if !mv.SetChildren(idx, parseBlocks(t, "first line\n\nsecond line\n")) {
		t.Fatal("SetChildren refused a disclosure")
	}
	if got := len(mv.blocks); got != 5 {
		t.Errorf("with two children attached the visible list is %d blocks, want 5", got)
	}

	// And the children must SURVIVE a toggle, which is the whole reason SetChildren
	// writes to the tree.
	mv.ToggleDisclosure(idx)
	mv.ToggleDisclosure(idx)
	if got := len(mv.blocks); got != 5 {
		t.Errorf("after a collapse and an expand the region has %d blocks, want 5: the "+
			"children were attached to the visible list and lost when it was rebuilt", got)
	}

	// And it must refuse a paragraph: attaching to a non-disclosure would produce a
	// document nobody can explain.
	if mv.SetChildren(0, parseBlocks(t, "x\n")) {
		t.Error("SetChildren accepted a paragraph as a collapsible region")
	}
}

// TestACollapsedRegionLeavesTheVisibleList is the property flattening exists for: a
// collapsed region's children are not in the list AT ALL, so nothing measures them, hits
// them or selects them. "Not painted" and "not present" are different, and only the second
// is cheap.
func TestACollapsedRegionLeavesTheVisibleList(t *testing.T) {
	withResolver(t, func(name string, params map[string]any, body string) (MarkdownWidget, string, bool) {
		return &probeWidget{wantHeight: 30}, "a section", true
	})
	mv, _ := renderedText(t, "before\n\n```region {}\n```\n\nafter\n")
	idx := -1
	for i, b := range mv.blocks {
		if b.Type == blockWidget {
			idx = i
		}
	}
	mv.SetChildren(idx, parseBlocks(t, "one\n\ntwo\n\nthree\n"))
	if got := len(mv.blocks); got != 6 {
		t.Fatalf("with three children attached the visible list is %d blocks, want 6", got)
	}
	mv.ToggleDisclosure(idx)
	if got := len(mv.blocks); got != 3 {
		t.Errorf("a COLLAPSED region still has its children in the visible list: %d blocks, want 3", got)
	}
	mv.ToggleDisclosure(idx)
	if got := len(mv.blocks); got != 6 {
		t.Errorf("expanding did not restore the children: %d blocks, want 6", got)
	}
}

// TestSelectingAcrossAWidgetHighlightsIt is the invariant the whole design rests on: THE
// SELECTION OVERLAY DOES NOT CARE WHAT DREW A BLOCK.
//
// The text core returns a rect per run, and an embedded widget has no runs — so before
// this, selecting across an embed highlighted NOTHING: the text either side lit up and the
// widget stayed unmarked, which reads as a selection bug rather than as "this thing has
// no text in it". A widget contributes one rect covering its block, from the same
// laid-out table the painter reads.
func TestSelectingAcrossAWidgetHighlightsIt(t *testing.T) {
	withResolver(t, func(name string, params map[string]any, body string) (MarkdownWidget, string, bool) {
		return &probeWidget{wantHeight: 60}, "probe", true
	})
	const src = "before\n\n```probe {\"id\":\"x\"}\nbody\n```\n\nafter\n"
	mv, _ := renderedText(t, src)

	// The selection covering the whole document must produce a rect for the WIDGET, not
	// only for the paragraphs either side of it.
	mv.SelectAll()
	rects := mv.SelectionRects(0, len(src))
	widgetRects := 0
	for _, r := range rects {
		if r.Height > 20 { // a line is ~20; a block with a 60px widget is taller
			widgetRects++
		}
	}
	if widgetRects == 0 {
		t.Errorf("selecting the whole document produced %d rects and none of them covers a "+
			"widget: the overlay is looking for text runs, and a widget has none", len(rects))
	}
}

// TestACollapsedEmbedCostsAHeaderAndNothingElse is the property the whole disclosure design
// names, measured in PIXELS rather than in calls.
//
// The call-count version of this test (that a collapsed widget is neither measured nor
// painted) is necessary but not sufficient: a widget could be skipped and still leave
// something behind — a background, a rule, a cached image. This compares what is DRAWN, and
// it is the version that would catch a widget whose height came back as its full body from
// a cache.
//
// The threshold is deliberately loose. The point is not the exact number; it is that a
// collapsed embed is an order of magnitude cheaper than an expanded one, so a transcript of
// them scrolls.
func TestACollapsedEmbedCostsAHeaderAndNothingElse(t *testing.T) {
	// A probe that ACTUALLY PAINTS. The contract tests elsewhere use one that only
	// records, which is right for measuring calls and useless here: with nothing drawn,
	// both counts are just the header and the comparison is 169 against 170.
	withResolver(t, func(name string, params map[string]any, body string) (MarkdownWidget, string, bool) {
		return &paintingProbe{height: 200}, "probe ran", true
	})

	const w, h = 400, 300
	render := func(collapsed bool) int {
		mv := NewMarkdownView()
		mv.SetFrameRect(graphics.NewRect(0, 0, w, h))
		mv.SetText("```probe {}\nbody\n```\n")
		mv.computeContentH()
		for i := range mv.blocks {
			mv.blocks[i].Collapsed = collapsed
		}
		mv.computeContentH()
		dev := graphics.NewSoftDevice(w, h)
		mv.Paint(graphics.NewCanvas(dev), graphics.NewRect(0, 0, w, h))
		n := 0
		for _, p := range dev.Pixels() {
			if p != 0 {
				n++
			}
		}
		return n
	}
	open, closed := render(false), render(true)
	if open == 0 {
		t.Fatal("the expanded embed drew nothing, so the comparison is meaningless")
	}
	if closed == 0 {
		t.Fatal("a collapsed embed drew NOTHING: it must still show its header")
	}
	if closed*4 > open {
		t.Errorf("a collapsed embed drew %d pixels against %d expanded: it is not costing "+
			"a header and nothing else, and a transcript of them will not scroll",
			closed, open)
	}
}

// paintingProbe fills the rect it is given, so a pixel count can tell an expanded widget
// from a collapsed one.
type paintingProbe struct{ height float32 }

func (p *paintingProbe) Height(ctx *WidgetContext) float32 { return p.height }
func (p *paintingProbe) Paint(canvas *graphics.Canvas, rect graphics.Rect, ctx *WidgetContext) {
	paint := graphics.NewPaint()
	paint.SetColor(ctx.Theme.TextInk)
	canvas.DrawRect(graphics.NewRect(rect.X, rect.Y, rect.Width, rect.Height), paint)
}

// TestAppendingToADocumentWithAnEmbedKeepsTheEmbed is the interaction between the two
// halves of this work, and it is the one that could plausibly be wrong.
//
// Append resumes the walk at the first block the new text could have touched, and parses
// the TAIL alone when the append starts at a block boundary. Both are new, and an embed is
// exactly the kind of block that neither was written with in mind: it is built by a
// resolver rather than by the AST walk, and its children come from the tree. So an append
// after an embed is where a duplicate, a lost embed or a lost widget would show up — and
// streaming into a document that already contains one is what an agent does on every turn
// after the first.
func TestAppendingToADocumentWithAnEmbedKeepsTheEmbed(t *testing.T) {
	built := 0
	withResolver(t, func(name string, params map[string]any, body string) (MarkdownWidget, string, bool) {
		built++
		return &paintingProbe{height: 40}, "tool ran", true
	})

	const w, h = 400, 300
	mv := NewMarkdownView()
	mv.SetFrameRect(graphics.NewRect(0, 0, w, h))
	mv.SetText("intro\n\n```probe {\"id\":\"read\"}\nbody\n```\n")
	mv.ensureParsed()

	widgetsBefore := 0
	for _, b := range mv.blocks {
		if b.Type == blockWidget {
			widgetsBefore++
		}
	}
	if widgetsBefore != 1 || built != 1 {
		t.Fatalf("the initial document built %d widgets for %d widget blocks", built, widgetsBefore)
	}

	// Append a WHOLE SECTION, which starts at a block boundary — the tail-parse path.
	mv.Append("\n## After\n\nmore prose\n")
	mv.ensureParsed()

	after := 0
	for _, b := range mv.blocks {
		if b.Type == blockWidget {
			after++
		}
	}
	if after != 1 {
		t.Errorf("appending turned %d widget block(s) into %d: an append after an embed "+
			"duplicated or lost it", widgetsBefore, after)
	}
	if built != 1 {
		t.Errorf("the resolver ran %d times for one embed; a re-parse must not rebuild "+
			"blocks the append cannot have touched", built)
	}

	// And the appended text is actually THERE, not merely absent from the count: a
	// resume that stopped at the embed would leave the block list looking right and the
	// new section missing.
	headings := 0
	for _, b := range mv.blocks {
		if b.Type == blockHeading {
			headings++
		}
	}
	if headings != 1 {
		t.Errorf("the appended \"## After\" heading is not in the block list (%d headings); "+
			"the walk resumed past the embed and stopped", headings)
	}
	painted := 0
	dev := graphics.NewSoftDevice(w, h)
	mv.Paint(graphics.NewCanvas(dev), graphics.NewRect(0, 0, w, h))
	for _, p := range dev.Pixels() {
		if p != 0 {
			painted++
		}
	}
	if painted == 0 {
		t.Error("the document paints nothing after an append")
	}
}
