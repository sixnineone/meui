package widgets

import (
	"fmt"
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/graphics/enums/slant"
	"github.com/sixnineone/meui/graphics/enums/weight"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	astex "github.com/yuin/goldmark/extension/ast"
	gtxt "github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

// paragraph spacing = font line height + half-line gap
var paraH = func() float32 {
	f := theme.DefaultTheme.BaseFont
	if f == nil {
		f = graphics.SystemFont
	}
	return f.LineHeight() * 1.5
}()

// codeKeywords maps a keyword to the SyntaxTheme SLOT it takes, not to a colour.
//
// It used to map to a literal colour — VS Code's dark+ palette — which is the same
// defect as the rest of the renderer's literals and had the same cause: the renderer
// predates the SyntaxTheme and never adopted it. Mapping to a SLOT means a fenced code
// block in a document and the same code in the CodeEditor are now coloured by one
// theme, and an imported editor scheme restyles both at once.
var codeKeywords = map[string]func(st *SyntaxTheme) graphics.Color{
	"func":   func(st *SyntaxTheme) graphics.Color { return st.Keyword.Color },
	"return": func(st *SyntaxTheme) graphics.Color { return st.Keyword.Color },
	"if":     func(st *SyntaxTheme) graphics.Color { return st.Keyword.Color },
	"else":   func(st *SyntaxTheme) graphics.Color { return st.Keyword.Color },
	"for":    func(st *SyntaxTheme) graphics.Color { return st.Keyword.Color },
	"range":  func(st *SyntaxTheme) graphics.Color { return st.Keyword.Color },
	"switch": func(st *SyntaxTheme) graphics.Color { return st.Keyword.Color },
	"case":   func(st *SyntaxTheme) graphics.Color { return st.Keyword.Color },
	"var":    func(st *SyntaxTheme) graphics.Color { return st.Keyword.Color },
	"let":    func(st *SyntaxTheme) graphics.Color { return st.Keyword.Color },
	"const":  func(st *SyntaxTheme) graphics.Color { return st.Keyword.Color },
	"type":   func(st *SyntaxTheme) graphics.Color { return st.Keyword.Color },

	"struct":    func(st *SyntaxTheme) graphics.Color { return st.Type.Color },
	"interface": func(st *SyntaxTheme) graphics.Color { return st.Type.Color },
	"map":       func(st *SyntaxTheme) graphics.Color { return st.Type.Color },
	"chan":      func(st *SyntaxTheme) graphics.Color { return st.Type.Color },
	"int":       func(st *SyntaxTheme) graphics.Color { return st.Type.Color },
	"string":    func(st *SyntaxTheme) graphics.Color { return st.Type.Color },
	"bool":      func(st *SyntaxTheme) graphics.Color { return st.Type.Color },
	"float64":   func(st *SyntaxTheme) graphics.Color { return st.Type.Color },
	"error":     func(st *SyntaxTheme) graphics.Color { return st.Type.Color },

	"import":  func(st *SyntaxTheme) graphics.Color { return st.Keyword.Color },
	"package": func(st *SyntaxTheme) graphics.Color { return st.Preprocessor.Color },
	"require": func(st *SyntaxTheme) graphics.Color { return st.Preprocessor.Color },

	"true":  func(st *SyntaxTheme) graphics.Color { return st.Constant.Color },
	"false": func(st *SyntaxTheme) graphics.Color { return st.Constant.Color },
	"nil":   func(st *SyntaxTheme) graphics.Color { return st.Constant.Color },
	"null":  func(st *SyntaxTheme) graphics.Color { return st.Constant.Color },

	"panic": func(st *SyntaxTheme) graphics.Color { return st.Function.Color },
}

// highlightCodeLine paints one line of a fenced block, colouring a keyword by its
// SyntaxTheme slot. st is the active syntax theme; nil means the global.
func highlightCodeLine(line string, defaultColor graphics.Color, font graphics.Font, c *graphics.Canvas, x, y float32) float32 {
	st := GlobalSyntaxTheme
	if st == nil {
		st = DefaultSyntaxTheme()
	}
	runes := []rune(line)
	i := 0
	for i < len(runes) {
		// Skip whitespace
		for i < len(runes) && (runes[i] == ' ' || runes[i] == '\t') {
			x += font.SimpleWidth(" ")
			i++
		}
		if i >= len(runes) {
			break
		}
		// Check for word characters
		if (runes[i] >= 'a' && runes[i] <= 'z') || (runes[i] >= 'A' && runes[i] <= 'Z') || runes[i] == '_' {
			start := i
			for i < len(runes) && ((runes[i] >= 'a' && runes[i] <= 'z') || (runes[i] >= 'A' && runes[i] <= 'Z') ||
				(runes[i] >= '0' && runes[i] <= '9') || runes[i] == '_') {
				i++
			}
			word := string(runes[start:i])
			clr := defaultColor
			if slot, ok := codeKeywords[word]; ok {
				clr = slot(st)
			}
			p := graphics.NewPaint()
			p.SetColor(clr)
			c.DrawSimpleString(word, x, y, font, p)
			x += font.SimpleWidth(word)
			continue
		}
		// Single rune
		ch := string(runes[i])
		p := graphics.NewPaint()
		p.SetColor(defaultColor)
		c.DrawSimpleString(ch, x, y, font, p)
		x += font.SimpleWidth(ch)
		i++
	}
	return x
}

var emojiMap = map[string]string{
	":smile:": "😊", ":sad:": "😢", ":heart:": "❤️", ":rocket:": "🚀",
	":fire:": "🔥", ":star:": "★", ":check:": "☑", ":x:": "✗",
	":warning:": "⚠️", ":info:": "ℹ️", ":bug:": "🐛", ":note:": "📝",
	":file:": "📄", ":folder:": "📁", ":search:": "🔍", ":lock:": "🔒",
	":key:": "🔑", ":gear:": "⚙️", ":tag:": "🏷️", ":mail:": "📧",
	":phone:": "📞", ":clock:": "🕐", ":calendar:": "📅", ":user:": "👤",
	":group:": "👥", ":home:": "🏠", ":build:": "🔨", ":link:": "🔗",
	":up:": "⬆️", ":down:": "⬇️", ":left:": "⬅️", ":right:": "➡️",
	":plus:": "+", ":minus:": "-", ":question:": "❓", ":idea:": "💡",
}

func replaceEmoji(text string) string {
	result := text
	for code, emoji := range emojiMap {
		// Only replace with glyphs the font atlas can rasterize (BMP);
		// non-BMP emoji render as tofu boxes.
		if len(emoji) > 0 && rune(emoji[0]) > 0xFFFF {
			continue
		}
		result = strings.ReplaceAll(result, code, emoji)
	}
	return result
}

type renderNode struct {
	Text      string
	Bold      bool
	Italic    bool
	Code      bool
	Link      string
	Strike    bool
	ImageAlt  string
	Task      bool           // task list checkbox glyph (clickable)
	Checked   bool           // task checkbox state
	TaskIndex int            // occurrence index of this checkbox in the source
	Sup       bool           // superscript: smaller font, raised baseline
	Sub       bool           // subscript: smaller font, lowered baseline
	Highlight bool           // highlighted text: background band behind the glyphs
	textObj   *graphics.Text // cached text for Unicode/emoji fallback rendering

	// SrcStart/SrcEnd: byte range of the markdown source this node was parsed
	// from. Zero (or -1) means unknown: the node renders synthetic text with no
	// direct source range (task checkbox glyphs, footnote markers). The range is
	// the node's own span, so for a node whose rendered text was transformed
	// (entities, emoji, ==highlight== splits) it is the *enclosing* source span;
	// the layout pass maps individual rendered characters onto it.
	SrcStart int
	SrcEnd   int
}

// hasSrcSpan reports whether the node's source range is known. The zero value
// (and the explicit -1 marker) means unknown.
func (rn renderNode) hasSrcSpan() bool { return rn.SrcStart >= 0 && rn.SrcEnd > rn.SrcStart }

// Block kinds. These were bare int literals scattered through the walk and the painter
// — 0 paragraph, 1 heading, 2 code, 4 list item, 5 rule, 6 table row, 7 table header, and
// 3 unused — which made adding a kind a matter of finding every literal that had to
// agree. Naming them is the whole of the change: a wrong constant is now a compile error
// rather than a block that quietly paints as a paragraph.
const (
	blockParagraph = iota
	blockHeading
	blockCode
	blockUnused3 // a gap the old numbering had; named so the numbering is not silently reused
	blockListItem
	blockRule
	blockTableRow
	blockTableHeader
	// blockWidget is an EMBED: a fence whose info string named a registered widget.
	// blockError is a widget attempt that could not be built — an unregistered name or
	// malformed parameters — drawn as a visible error rather than falling through to the
	// code renderer, which would look like a document that simply contains code.
	// blockWidget is a DISCLOSURE: a header plus a body, where the body is a widget, a
	// run of MARKDOWN blocks, or both. A region of Markdown is not a separate kind —
	// SetChildren attaches blocks to one of these — and an earlier version had a
	// `blockDisclosure` for it that nothing ever produced, which is how a kind ends up
	// documented and never drawn.
	blockWidget
	blockError
)

type block struct {
	Type       int
	Level      int  // heading level or list indent
	QuoteDepth int  // blockquote nesting depth
	Ordered    bool // list: numbered marker
	Number     int  // list: ordinal for ordered markers
	Text       *graphics.Text
	Nodes      []renderNode
	CodeText   string
	Cells      []string
	// CellSrc holds each table cell's source span, in the same order as
	// Cells, so a cell can be selected row-wise.
	CellSrc [][2]int
	Lang    string

	// ── embedded widgets (see markdown_embed.go) ──
	//
	// Widget is nil on every block that is not an embed, so the painter's dispatch is
	// "is it a widget block" rather than a type assertion on every block.
	Widget MarkdownWidget
	// WidgetName is the registered name from the fence's info string. It is NOT the
	// block's Lang: a fence's language tag means something else, and a widget told its
	// own name by way of a field that means "syntax highlighting" will report the wrong
	// name the first time anyone uses a language tag.
	WidgetName string
	Params     map[string]any
	Body       string
	// Summary is the ONE LINE the disclosure header shows. The widget supplies it and
	// the container owns everything else about the disclosure.
	Summary string
	// Children makes the block list a TREE rather than a flat sequence: a disclosure
	// owns the blocks inside it, so a collapsible region can appear ANYWHERE in a
	// document rather than only at a leaf. nil on every block with no children, which
	// is most of them.
	Children []block
	// Collapsed is a disclosure's state. A collapsed block's children are not laid out
	// and not painted, which is what makes a collapsed embed cost its header alone.
	Collapsed bool

	// SrcStart/SrcEnd: byte range of the markdown source this block was parsed
	// from (content span: a fenced code block covers its lines, not the fence
	// markers). Zero (or -1) means unknown.
	SrcStart int
	SrcEnd   int
}

// hasSrcSpan reports whether the block's source range is known.
func (b block) hasSrcSpan() bool { return b.SrcStart >= 0 && b.SrcEnd > b.SrcStart }

var headingSizes = []float32{32, 24, 20, 16, 14, 12}
var headingSpacing2 = []float32{44, 34, 28, 24, 22, 20}

func headingFont(size float32) graphics.Font {
	font := theme.DefaultTheme.BaseFont
	if font == nil {
		font = graphics.SystemFont
	}
	if face := font.Face(); face != nil {
		return face.Font(size)
	}
	return font
}

type MarkdownView struct {
	meui.BaseContainer
	// tree is the PARSED document, with disclosures owning their children. blocks is
	// the VISIBLE sequence — the tree flattened with collapsed subtrees OMITTED — and it
	// is what layout, paint, hit-testing and selection all index.
	//
	// Two structures rather than one is deliberate, and it is what keeps AGENTS.md's "one
	// markdown vertical path" intact: if the flat list WERE the tree, every array that
	// must stay index-aligned with it (blockYs, blockHs, laid) would have to become a tree
	// too, and so would the selection's hit test. Flattening on toggle means none of them
	// change, and a collapsed region costs nothing because its children simply are not in
	// the list.
	tree []block
	// MarkdownTheme is every colour, metric and font the renderer uses. It is a pointer
	// to the global by default, so importing a theme restyles documents the way it
	// restyles everything else; a view that wants a private palette copies it.
	*theme.MarkdownTheme
	text            string
	blocks          []block
	scrollOff       float32
	contentH        float32
	scrollBar       *ScrollBar
	host            *ScrollbarHost
	imgCache        map[string]*graphics.Image
	cache           meui.CacheBase
	footnoteCounter int
	taskCounter     int // occurrence counter during parse (index for task toggling)

	// blockYs/blockHs: content-space top offset + height per block,
	// computed by computeContentH — used for selection hit-testing and
	// highlight painting.
	blockYs []float32
	blockHs []float32
	// laid is the per-block layout table (see markdown_layout.go): the
	// geometry Paint draws from and hit-testing/selection read.
	laid []mdBlockLayout
	// srcOffset shifts source spans while a TAIL-ONLY parse is being walked, so a block
	// built from the tail still reports offsets in the whole document's coordinates.
	// Zero everywhere else.
	srcOffset int
	// parsedLen is how much of text has been WALKED. Parsing and walking are separate
	// because goldmark parses the whole document while the walk is 91% of the cost (see
	// Append), so a streamed token re-walks the tail and leaves the rest alone.
	parsedLen int
	// rootEnds[i] is len(blocks) after the i-th ROOT child has been walked. It is what
	// turns "keep the first N blocks" into "skip the first M root children", which is
	// the only way a partial walk can resume — block indices and root children are not
	// in one-to-one correspondence, because a list or a quote produces many blocks from
	// one child.
	rootEnds []int
	// pending is text appended but not yet walked. Several tokens in one frame collapse
	// into one parse, which is the difference between a stream that keeps up and one that
	// does not.
	pending string
	// streaming makes the viewport follow the tail, which is what a chat pane does and
	// what a document view does not.
	streaming bool
	// laidLineH/laidSpaceW fingerprint the font metrics the table was built
	// with, so a theme (font) change rebuilds it instead of leaving stale
	// positions behind.
	laidLineH, laidSpaceW float32
	// layoutWidth is the content width the table was measured at.
	layoutWidth float32
	// laidH is the laid-out content height (gaps included). contentH is the
	// scrollable extent, still derived from the historical formula.
	laidH float32
	// flat is the document-wide layout (every block's lines), rebuilt on demand
	// by textLayout() from the per-block tables.
	flat textLayout
	// Selection is held as markdown source offsets: selAnchor is where the
	// selection started, selFocus is the moving end (the caret). -1 = none.
	selAnchor int
	selFocus  int
	// selecting is true while a mouse drag is extending the selection.
	selecting bool
	// click counting for double (word) and triple (block) click.
	clickCount    int
	lastClickTime time.Time
	lastClickX    float32
	lastClickY    float32
	dragX, dragY  float32
	dragMoved     bool
	// pendingLink is the link under a press; it is opened on mouse-up only
	// when the press did not turn into a drag.
	pendingLink     string
	pendingLinkRect graphics.Rect
	// focused gates keyboard selection; testModifiers stands in for the
	// window's modifier state when the widget has no window (unit tests).
	focused       bool
	testModifiers platform.Modifiers

	linkRects []linkRect
	taskRects []taskRect
	// OnTaskToggled fires after a task checkbox is clicked and the source
	// rewritten: index is the occurrence order of task checkboxes in the
	// document, checked is the new state. Text() returns the updated source.
	OnTaskToggled func(index int, checked bool)

	// OnDisclosureToggled fires when an embedded widget's disclosure is expanded or
	// collapsed, with the block index and the new state. It is how an app learns that
	// the user opened a step, without the view having to know what an app is.
	OnDisclosureToggled func(index int, collapsed bool)
	// OnLinkClicked is called when a link is clicked (a press that did not
	// become a drag, or a Ctrl/Cmd-click). When nil the URL is opened with
	// meui.OpenURL.
	OnLinkClicked func(url string)
}

type linkRect struct {
	rect graphics.Rect
	url  string
}

type taskRect struct {
	rect    graphics.Rect
	index   int // occurrence index of the checkbox in the source
	checked bool
}

// Append adds to the document, which is what a streaming answer needs.
//
// WHY IT IS NOT SetText. Measured, not assumed: SetText costs 25ms on a 6250-character
// document and grows LINEARLY with it — 0.4ms at 125 chars, 4.5ms at 1250, 25ms at 6250 —
// and a frame is 16ms. Streaming by calling SetText per token is therefore slower than the
// stream. Of that 25ms, goldmark's whole-document parse is only 2.2ms; the other 23ms is
// this file's own walk. So the parse stays whole (it is the cheap part) and the WALK is
// resumed from the first block the append could have touched.
//
// Append does not parse immediately. It buffers, and the parse happens once per frame at
// paint time, so twenty tokens in one frame cost one walk.
func (mv *MarkdownView) Append(text string) *MarkdownView {
	if text == "" {
		return mv
	}
	mv.text += text
	mv.pending += text
	mv.SetNeedsLayout()
	mv.MarkForRedraw()
	return mv
}

// Streaming reports whether the view is following a tail.
func (mv *MarkdownView) Streaming() bool { return mv.streaming }

// SetStreaming turns on tail-following: the viewport sticks to the bottom as the
// document grows, which is what a chat pane does and what a document view does not.
// It is off by default, so an ordinary document does not scroll itself while a reader
// is halfway down it.
func (mv *MarkdownView) SetStreaming(on bool) *MarkdownView {
	mv.streaming = on
	if on {
		mv.ScrollToBottom()
	}
	mv.MarkForRedraw()
	return mv
}

// ensureParsed applies whatever Append has buffered. It is called from Paint, so the
// walk happens at most once per frame no matter how many tokens arrived.
func (mv *MarkdownView) ensureParsed() {
	if mv.parsedLen == len(mv.text) {
		return
	}
	from := mv.parsedLen
	keep, skip := mv.resumePoint(from)

	// Keep the blocks the append cannot have touched, and resume the walk past the root
	// children that produced them.
	mv.blocks = mv.blocks[:keep]
	mv.rootEnds = mv.rootEnds[:skip]

	// Parse the TAIL ALONE when the append begins at a block boundary, which is the
	// common case for a streamed answer: a new paragraph or a new heading. That is worth
	// doing because after the walk is made incremental, goldmark's whole-document parse is
	// the dominant remaining cost — 2.2ms of the 4.66ms a token append takes at 50
	// sections — and the tail is all that changed.
	//
	// When the append CONTINUES a block (a growing list item, a paragraph still being
	// written, an unterminated fence) the tail is not a document on its own, so the whole
	// document is parsed and the walk still resumes. Correct in both cases; only the
	// first is cheap.
	if from == 0 || mv.endsAtBlockBoundary(from) {
		tail := []byte(mv.text[from:])
		root := goldmark.New(markdownExtensions()...).Parser().Parse(gtxt.NewReader(tail))
		// skip is ZERO here, and that is the point: the tail is its own document, so its
		// first child is the first NEW block. Passing the whole document's skip threw the
		// new blocks away and left a stream that only ever showed its first section.
		mv.walkTail(root, tail, from, 0)
	} else {
		src := []byte(mv.text)
		root := goldmark.New(markdownExtensions()...).Parser().Parse(gtxt.NewReader(src))
		mv.walkBlockDepth(root, src, 0, 0, skip, true)
	}
	mv.parsedLen = len(mv.text)
	mv.pending = ""

	mv.computeContentH()
	mv.cache.Invalidate()
	mv.SetNeedsLayout()
	if mv.streaming {
		mv.ScrollToBottom()
	}
}

// resumePoint returns how many existing blocks survive an append at source offset `from`,
// and how many root children produced them.
//
// A block survives if it ENDS at or before the append point. `>=` rather than `>` on
// purpose: a block that ends exactly where the text is appended may still grow — an
// unterminated fenced code block is exactly that case — and rebuilding one block is
// cheaper than being wrong about one.
// srcRangeOf is srcRangeOf with the tail offset applied. Every span the walk records
// goes through here, because a tail parse hands the walk bytes whose offsets start at
// zero and a block that reported the wrong SrcStart would make the NEXT append resume in
// the wrong place — which is silent, and duplicates blocks rather than failing.
func (mv *MarkdownView) srcRangeOf(child ast.Node, src []byte) (int, int) {
	a, b := srcRangeOf(child, src)
	if mv.srcOffset == 0 {
		return a, b
	}
	// Add the offset to EVERY span, and keep a negative one negative: a negative span
	// means "this node has no source", and shifting it would turn "unknown" into a
	// confident wrong answer, which then makes the next append resume in the wrong place.
	if a >= 0 {
		a += mv.srcOffset
	}
	if b >= 0 {
		b += mv.srcOffset
	}
	return a, b
}

func (mv *MarkdownView) resumePoint(from int) (keep, skip int) {
	keep = len(mv.blocks)
	for i, b := range mv.blocks {
		if b.SrcEnd < 0 || b.SrcEnd >= from {
			keep = i
			break
		}
	}
	skip = 0
	for _, end := range mv.rootEnds {
		if end > keep {
			break
		}
		skip++
	}
	return keep, skip
}

// ScrollToBottom puts the viewport at the end of the document, which is what a streaming
// view wants after new text arrives.
func (mv *MarkdownView) ScrollToBottom() {
	if mv.host == nil {
		return
	}
	mv.computeContentH()
	// The host clamps to its own extents, so asking for the largest offset the content
	// allows and letting it clamp is simpler than recomputing the maximum here — and it
	// cannot disagree with what the scrollbar thinks the maximum is.
	mv.host.SetExtents(mv.Width(), mv.contentH, mv.Width(), mv.Height())
	mv.host.SetOffsets(0, mv.contentH)
	_, y := mv.host.Offsets()
	mv.scrollOff = y
	mv.cache.Invalidate()
	mv.MarkForRedraw()
}

// SetMarkdownTheme gives the view a private palette, copied so a change here cannot
// reach the global. It is how a document is rendered light inside a dark app, which is
// what the harness's document pane is: a light surface, whatever the app around it does.
func (mv *MarkdownView) SetMarkdownTheme(mt *theme.MarkdownTheme) *MarkdownView {
	mv.MarkdownTheme = mt.Copy()
	// A palette change must RE-PARSE, and the reason is worth recording: a block's runs
	// are built at parse time and each carries its own ink, so repainting after a theme
	// change puts the NEW page behind the OLD text. That is not subtle in the demo — the
	// page went white and the heading stayed light grey on it — and it is invisible in a
	// test that only checks the palette fields.
	//
	// Re-parsing is the honest repair rather than a recolour pass: the runs are built
	// from the theme while walking the AST, and a second code path that re-inked them
	// would be a second place where a colour is decided.
	if mv.text != "" {
		mv.SetText(mv.text)
	}
	mv.cache.Invalidate()
	mv.SetNeedsLayout()
	mv.MarkForRedraw()
	return mv
}

// SetLightPalette selects the light renderer palette — the harness's document pane, read
// off the reference rather than invented. It copies, so a view that lightens itself
// leaves the global alone.
func (mv *MarkdownView) SetLightPalette() *MarkdownView {
	return mv.SetMarkdownTheme(theme.LightMarkdownTheme)
}

// SetDarkPalette selects the dark renderer palette.
func (mv *MarkdownView) SetDarkPalette() *MarkdownView {
	return mv.SetMarkdownTheme(theme.DefaultMarkdownTheme)
}

// markdownExtensions is the extension set, in one place because the full-document and
// tail parses must not be able to disagree about what Markdown means here.
func markdownExtensions() []goldmark.Option {
	return []goldmark.Option{
		goldmark.WithExtensions(
			extension.GFM,
			extension.TaskList,
			extension.DefinitionList,
			extension.Footnote,
		),
	}
}

// endsAtBlockBoundary reports whether offset `from` starts a fresh block, which is what
// makes a tail parseable on its own: a blank line before it, or nothing before it.
func (mv *MarkdownView) endsAtBlockBoundary(from int) bool {
	if from <= 0 || from > len(mv.text) {
		return true
	}
	return strings.HasSuffix(mv.text[:from], "\n\n")
}

// walkTail walks a tail-only parse.
func (mv *MarkdownView) walkTail(root ast.Node, tail []byte, offset, skip int) {
	mv.srcOffset = offset
	mv.walkBlockDepth(root, tail, 0, 0, skip, true)
	mv.srcOffset = 0
}

func NewMarkdownView() *MarkdownView {
	mv := &MarkdownView{
		BaseContainer: meui.BaseContainer{BaseWidget: *meui.NewBaseWidget()},
		MarkdownTheme: theme.DefaultMarkdownTheme,
		imgCache:      make(map[string]*graphics.Image),
		selAnchor:     -1,
		selFocus:      -1,
	}
	mv.host = NewScrollbarHost()
	mv.host.ShowH = false
	mv.scrollBar = mv.host.VScroll
	mv.scrollBar.SetParent(mv)
	mv.host.SetOnChange(func() {
		_, offY := mv.host.Offsets()
		mv.scrollOff = offY
		mv.cache.Invalidate()
		mv.MarkForRedraw()
	})
	return mv
}

func (mv *MarkdownView) SetText(text string) {
	mv.footnoteCounter = 0
	mv.taskCounter = 0
	mv.text = text
	mv.blocks = nil
	if text == "" {
		return
	}
	src := []byte(text)
	reader := gtxt.NewReader(src)
	root := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.TaskList,
			extension.DefinitionList,
			extension.Footnote,
		),
	).Parser().Parse(reader)
	mv.blocks = mv.blocks[:0]
	mv.walkBlock(root, src, 0)
	// The walk fills the visible list; the tree is a COPY of it, so a toggle can
	// rebuild the visible list from an untouched parse. Copying the slice header's
	// contents is enough because a toggle only ever flips a bool on a block, never
	// edits one — and a toggle that edited a node would be mutating both at once.
	mv.tree = append(mv.tree[:0], mv.blocks...)
	mv.parsedLen = len(text)
	mv.pending = ""
	mv.computeContentH()
	mv.cache.Invalidate()
	mv.SetNeedsLayout()
	mv.MarkForRedraw()
}
func (mv *MarkdownView) Text() string { return mv.text }

func (mv *MarkdownView) SetFilePath(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		mv.SetText("Error: " + err.Error())
		return
	}
	mv.SetText(string(data))
}

func (mv *MarkdownView) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "text":
		if s, ok := value.(string); ok {
			mv.SetText(s)
		}
	case "filePath":
		if s, ok := value.(string); ok {
			mv.SetFilePath(s)
		}
	}
}

// computeContentH rebuilds the layout table and the scrollable extent. Paint
// and hit-testing both read the table it produces, so they cannot drift.
func (mv *MarkdownView) computeContentH() {
	mv.layoutWidth = mv.FrameRect().Width
	mv.layoutContent()
}

// buildText creates a *graphics.Text from render nodes, using the base font and applying decorations.
// processInlineExtras handles ==highlight==, ~subscript~, ^superscript^.
// It returns the styled pieces plus the rune index in text where each piece
// starts, so callers can map the pieces back onto source offsets (the marker
// runes themselves are dropped from the output).
func (mv *MarkdownView) processInlineExtras(text string) ([]renderNode, []int) {
	var result []renderNode
	var starts []int
	runes := []rune(text)
	i := 0
	n := len(runes)
	for i < n {
		// ==highlight==
		if i+1 < n && runes[i] == '=' && runes[i+1] == '=' {
			rest := string(runes[i+2:])
			end := strings.Index(rest, "==")
			if end >= 0 {
				result = append(result, renderNode{Text: string(runes[i+2 : i+2+end]), Highlight: true})
				starts = append(starts, i+2)
				i += end + 4
				continue
			}
		}
		// ^superscript^
		if runes[i] == '^' {
			rest := string(runes[i+1:])
			end := strings.Index(rest, "^")
			if end >= 0 {
				result = append(result, renderNode{Text: string(runes[i+1 : i+1+end]), Sup: true})
				starts = append(starts, i+1)
				i += end + 2
				continue
			}
		}
		// ~subscript~ (but not ~~strikethrough~~)
		if runes[i] == '~' && i+1 < n && runes[i+1] != '~' {
			rest := string(runes[i+1:])
			end := strings.Index(rest, "~")
			if end >= 0 {
				result = append(result, renderNode{Text: string(runes[i+1 : i+1+end]), Sub: true})
				starts = append(starts, i+1)
				i += end + 2
				continue
			}
		}
		result = append(result, renderNode{Text: string(runes[i])})
		starts = append(starts, i)
		i++
	}
	return result, starts
}

// mapRunesToSource aligns rendered runes with the raw source segment they were
// parsed from and returns the source byte offset of each rendered rune (-1 when
// the rune has no source counterpart, e.g. an emoji substituted for a shortcode
// or a soft-break newline goldmark keeps outside the segment).
//
// Escapes and entities make the source longer than the rendered text, so a
// rune that does not match at the cursor is searched for later *within the
// segment* — a substitution therefore cannot desync the runes after it.
func mapRunesToSource(src []byte, segStart, segStop int, rendered []rune) []int {
	out := make([]int, len(rendered))
	if segStart < 0 || segStop > len(src) || segStop < segStart {
		for i := range out {
			out[i] = -1
		}
		return out
	}
	raw := src[segStart:segStop]
	i := 0
	for ri, r := range rendered {
		if i < len(raw) {
			if r2, size := utf8.DecodeRune(raw[i:]); r2 == r {
				out[ri] = segStart + i
				i += size
				continue
			}
		}
		found := -1
		for j := i; j < len(raw); {
			r2, size := utf8.DecodeRune(raw[j:])
			if r2 == r {
				found = j
				i = j + size
				break
			}
			j += size
		}
		if found < 0 {
			if r == '\n' {
				out[ri] = segStop
			} else {
				out[ri] = -1
			}
			continue
		}
		out[ri] = segStart + found
	}
	return out
}

// spanForRunes returns the source span covered by n runes of rendered starting
// at rune index start, falling back to [fallbackStart, fallbackStop) when no
// rune has a known offset.
func spanForRunes(offs []int, rendered []rune, start, n, fallbackStart, fallbackStop int) (int, int) {
	if n <= 0 || start < 0 || start >= len(offs) {
		return fallbackStart, fallbackStop
	}
	lo, hi := -1, -1
	for k := start; k < start+n && k < len(offs); k++ {
		o := offs[k]
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
	if lo < 0 {
		return fallbackStart, fallbackStop
	}
	if last := start + n - 1; last < len(rendered) && offs[last] >= 0 {
		hi = offs[last] + utf8.RuneLen(rendered[last])
	} else {
		hi++
	}
	return lo, hi
}

// runeSrcSpan returns the source span of the ri-th rune of a node's rendered
// text. When the node's text maps 1:1 onto its source span the rune's own span
// is returned; otherwise (entities, emoji, ...) the enclosing span is.
func runeSrcSpan(rn renderNode, ri int) (int, int) {
	if !rn.hasSrcSpan() {
		return -1, -1
	}
	runes := []rune(rn.Text)
	if ri < 0 || ri >= len(runes) {
		return -1, -1
	}
	total := 0
	for _, r := range runes {
		total += utf8.RuneLen(r)
	}
	if total != rn.SrcEnd-rn.SrcStart {
		return rn.SrcStart, rn.SrcEnd
	}
	off := 0
	for i := 0; i < ri; i++ {
		off += utf8.RuneLen(runes[i])
	}
	size := utf8.RuneLen(runes[ri])
	return rn.SrcStart + off, rn.SrcStart + off + size
}

// buildText lays a block's inline runs out into one Text. The inks come from the theme
// because a document's body, its inline code and its links are three different colours,
// and hardcoding them is what made every document look like a dark terminal.
func buildText(nodes []renderNode, baseFont graphics.Font, mt *theme.MarkdownTheme) *graphics.Text {
	mt = mt.Theme()
	t := graphics.NewText("", &graphics.TextDecoration{Font: baseFont, OnBackgroundInk: mt.TextInk})
	for _, rn := range nodes {
		if rn.Text == "" {
			continue
		}
		dec := &graphics.TextDecoration{Font: baseFont, OnBackgroundInk: mt.TextInk}
		if rn.Code {
			dec.OnBackgroundInk = mt.InlineCodeInk
			dec.Font = baseFont
		} else if rn.Link != "" {
			dec.OnBackgroundInk = mt.LinkInk
		} else {
			if rn.Bold || rn.Italic {
				fd := baseFont.Descriptor()
				if rn.Bold {
					fd.Weight = weight.Black
				}
				if rn.Italic {
					fd.Slant = slant.Italic
				}
				dec.Font = fd.Font()
			}
			if rn.Strike {
				dec.StrikeThrough = true
			}
		}
		t.AddString(rn.Text, dec)
	}
	return t
}

// srcRangeOf returns the byte range of the markdown source that node covers.
//
// It prefers the node's own line segments (block nodes and text lines), then a
// *ast.Text's segment, and otherwise the union of its children's ranges. It
// returns (-1, -1) when neither is available — goldmark keeps some inline
// values (autolink URLs, raw HTML, string values) as subslices of src without
// recording where they started, and synthetic nodes (task checkboxes) have no
// source text of their own at all.
func srcRangeOf(node ast.Node, src []byte) (int, int) {
	if node == nil {
		return -1, -1
	}
	if t, ok := node.(*ast.Text); ok {
		if seg := t.Segment; seg.Stop > seg.Start && seg.Stop <= len(src) {
			return seg.Start, seg.Stop
		}
		return -1, -1
	}
	// Lines() panics on inline nodes, so only block nodes may be asked.
	if node.Type() != ast.TypeInline {
		if lines := node.Lines(); lines != nil && lines.Len() > 0 {
			start := lines.At(0).Start
			stop := lines.At(lines.Len() - 1).Stop
			if start >= 0 && stop > start && stop <= len(src) {
				return start, stop
			}
		}
	}
	start, stop := -1, -1
	for c := node.FirstChild(); c != nil; c = c.NextSibling() {
		s, e := srcRangeOf(c, src)
		if s < 0 {
			continue
		}
		if start < 0 || s < start {
			start = s
		}
		if stop < 0 || e > stop {
			stop = e
		}
	}
	return start, stop
}

func (mv *MarkdownView) collectInline(node ast.Node, src []byte) []renderNode {
	var result []renderNode
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Kind() {
		case ast.KindText:
			t := child.(*ast.Text)
			b := util.UnescapePunctuations(t.Value(src))
			b = util.ResolveNumericReferences(b)
			val := replaceEmoji(string(util.ResolveEntityNames(b)))
			if t.SoftLineBreak() || t.HardLineBreak() {
				val += "\n"
			}
			// Post-process for ==highlight==, ~subscript~, ^superscript^
			processed, starts := mv.processInlineExtras(val)
			runes := []rune(val)
			seg := t.Segment
			offs := mapRunesToSource(src, seg.Start, seg.Stop, runes)
			for i := range processed {
				n := utf8.RuneCountInString(processed[i].Text)
				processed[i].SrcStart, processed[i].SrcEnd = spanForRunes(offs, runes, starts[i], n, seg.Start, seg.Stop)
				result = append(result, processed[i])
			}
		case ast.KindEmphasis:
			em := child.(*ast.Emphasis)
			bold := em.Level == 2
			inner := mv.collectInline(child, src)
			for i := range inner {
				if bold {
					inner[i].Bold = true
				} else {
					inner[i].Italic = true
				}
			}
			result = append(result, inner...)
		case ast.KindCodeSpan:
			inner := mv.collectInline(child, src)
			for i := range inner {
				inner[i].Code = true
			}
			result = append(result, inner...)
		case ast.KindLink:
			ln := child.(*ast.Link)
			inner := mv.collectInline(child, src)
			for i := range inner {
				inner[i].Link = string(ln.Destination)
			}
			result = append(result, inner...)
		case ast.KindAutoLink:
			al := child.(*ast.AutoLink)
			inner := mv.collectInline(child, src)
			if len(inner) == 0 {
				// Extension-parsed autolinks carry the URL in the node
				// value, not in children. goldmark keeps it as a slice of
				// src but records no offset, so the span is unknown here;
				// the layout pass resolves it from its neighbours.
				if u := string(al.URL(src)); u != "" {
					inner = append(inner, renderNode{Text: u, SrcStart: -1, SrcEnd: -1})
				}
			}
			for i := range inner {
				inner[i].Link = inner[i].Text
			}
			result = append(result, inner...)
		case astex.KindStrikethrough:
			inner := mv.collectInline(child, src)
			for i := range inner {
				inner[i].Strike = true
			}
			result = append(result, inner...)
		case astex.KindTaskCheckBox:
			tc := child.(*astex.TaskCheckBox)
			glyph := "☐ "
			if tc.IsChecked {
				glyph = "☑ "
			}
			result = append(result, renderNode{Text: glyph, Code: true, Task: true, Checked: tc.IsChecked, TaskIndex: mv.taskCounter, SrcStart: -1, SrcEnd: -1})
			mv.taskCounter++
		case astex.KindFootnoteLink:
			fl := child.(*astex.FootnoteLink)
			result = append(result, renderNode{Text: fmt.Sprintf("[%d]", fl.Index+1), Code: true, SrcStart: -1, SrcEnd: -1})
		case astex.KindFootnoteBacklink:
			// skip backlink rendering
		case ast.KindImage:
			img := child.(*ast.Image)
			alt := ""
			for ic := child.FirstChild(); ic != nil; ic = ic.NextSibling() {
				if t, ok := ic.(*ast.Text); ok {
					alt += string(t.Value(src))
				}
			}
			url := string(img.Destination)
			if alt == "" {
				alt = url
			}
			as, ae := mv.srcRangeOf(child, src)
			result = append(result, renderNode{Text: "[img]", Code: true, Link: url, ImageAlt: alt, SrcStart: as, SrcEnd: ae})
		case ast.KindRawHTML:
			result = append(result, renderNode{Text: string(child.Text(src)), Code: true, SrcStart: -1, SrcEnd: -1})
		case ast.KindString:
			result = append(result, renderNode{Text: string(child.Text(src)), SrcStart: -1, SrcEnd: -1})
		default:
			inner := mv.collectInline(child, src)
			result = append(result, inner...)
		}
	}
	return result
}

func (mv *MarkdownView) walkBlock(node ast.Node, src []byte, indent int) {
	mv.rootEnds = mv.rootEnds[:0]
	mv.walkBlockDepth(node, src, indent, 0, 0, true)
}

// paintDisclosureHeader draws the container's part of an embedded widget's block: the
// chevron and the ONE LINE the widget supplied as its summary.
//
// The container owns all of it — the chevron's direction, the hit target, the collapsed
// height — and the widget owns only the summary. That split is the reason a transcript of
// six hundred embedded steps is scannable: the eye reads a column of KINDS, and a widget
// that wanted to draw its own header could make that column ragged.
func (mv *MarkdownView) paintDisclosureHeader(canvas *graphics.Canvas, blk *block, r graphics.Rect, mt *theme.MarkdownTheme, font graphics.Font) {
	cy := r.Y + r.Height/2
	ch := graphics.NewPaint()
	ch.SetColor(mt.ListMarkerInk)
	cx := r.X + 12
	// Right when collapsed, down when open — the same two glyphs the Collapsible draws,
	// for the same reason: a disclosure must not depend on an icon being in the set.
	if blk.Collapsed {
		canvas.DrawLine(cx-3, cy-4, cx+2, cy, ch)
		canvas.DrawLine(cx+2, cy, cx-3, cy+4, ch)
	} else {
		canvas.DrawLine(cx-4, cy-2, cx, cy+3, ch)
		canvas.DrawLine(cx, cy+3, cx+4, cy-2, ch)
	}

	summary := blk.Summary
	if summary == "" {
		summary = blk.WidgetName
	}
	if summary == "" {
		return
	}
	p := graphics.NewPaint()
	p.SetColor(mt.TextInk)
	t := graphics.NewText(summary, &graphics.TextDecoration{Font: font, OnBackgroundInk: mt.TextInk})
	baseY := cy + font.Baseline()/2
	canvas.Save()
	canvas.ClipRect(graphics.NewRect(r.X+22, r.Y, r.Width-30, r.Height), pathop.Intersect, false)
	t.Draw(canvas, r.X+22, baseY)
	canvas.Restore()
}

// embedBlock turns a fenced code block into an embedded widget, or reports that it is an
// ordinary code block after all.
//
// It answers false for a plain language tag — ```go, ```json — so the renderer is
// unchanged for every document that has no embeds in it. It answers a WIDGET block when
// the resolver claims the name, and an ERROR block when the author clearly meant to embed
// and could not: a brace in the info string is a widget attempt, so a missing
// registration or a malformed object is REPORTED rather than rendered as a code block.
// That is the direction that matters, because an embed that fell through would look like
// a document that simply contains code, and nothing would say otherwise.
func (mv *MarkdownView) embedBlock(cb *ast.FencedCodeBlock, src []byte, body string, cs, ce int) (block, bool) {
	info := ""
	if cb.Info != nil {
		info = string(cb.Info.Value(src))
	}
	font := theme.DefaultTheme.BaseFont
	if font == nil {
		font = graphics.SystemFont
	}
	name, params, attempt, err := ParseFenceInfo(info)
	if err != nil {
		return mv.errorBlock("embed: "+err.Error(), font, cs, ce), true
	}
	if !attempt {
		// No parameters, so this is a language tag — UNLESS the registry knows the
		// name, because a widget with no parameters is legitimate.
		name = strings.TrimSpace(info)
	}
	if name == "" || FenceResolver == nil {
		if attempt {
			return mv.errorBlock("embed: no widget is registered for "+name, font, cs, ce), true
		}
		return block{}, false
	}
	w, summary, ok := FenceResolver(name, params, body)
	if !ok {
		if attempt {
			return mv.errorBlock("embed: no widget is registered for "+name, font, cs, ce), true
		}
		return block{}, false
	}
	return block{Type: blockWidget, Widget: w, WidgetName: name, Params: params, Body: body,
		Summary: summary, SrcStart: cs, SrcEnd: ce}, true
}

// flatten rebuilds the visible block list from the tree, skipping collapsed subtrees.
//
// It is the whole of "a collapsed region costs a header and nothing else": the children
// are not merely unpainted, they are not in the list, so nothing measures them, hits them
// or selects them.
func (mv *MarkdownView) flatten(dst []block) []block {
	dst = dst[:0]
	var walk func(bs []block)
	walk = func(bs []block) {
		for _, b := range bs {
			dst = append(dst, b)
			if isDisclosure(b) && b.Collapsed {
				continue
			}
			walk(b.Children)
		}
	}
	walk(mv.tree)
	return dst
}

// isDisclosure reports whether a block owns a collapsible body. There is one such kind:
// a widget block, whose body is a widget, Markdown, or both.
func isDisclosure(b block) bool { return b.Type == blockWidget }

// toggleInTree flips the collapsed state of the nth VISIBLE disclosure in the tree, and
// reports whether it found one. The index is a VISIBLE index because that is what the hit
// test and the caller both hold.
func (mv *MarkdownView) toggleInTree(nth int) bool {
	// The index counts BLOCKS, not disclosures, because the hit test and the caller both
	// hold a block index out of the VISIBLE list. Counting disclosures here made every
	// index past the first disclosure wrong, and the first version of this failed with
	// 'SetChildren refused a disclosure' on a document whose second block WAS one.
	n := 0
	var walk func(bs []block) bool
	walk = func(bs []block) bool {
		for i := range bs {
			if n == nth {
				if !isDisclosure(bs[i]) {
					return false
				}
				bs[i].Collapsed = !bs[i].Collapsed
				return true
			}
			n++
			if isDisclosure(bs[i]) && bs[i].Collapsed {
				continue // its children are not visible, so they are not counted
			}
			if walk(bs[i].Children) {
				return true
			}
		}
		return false
	}
	return walk(mv.tree)
}

// SetChildren attaches child blocks to a visible block, so a collapsible region can hold
// MARKDOWN rather than only a widget. It is the general form of what a fence builds, and
// it is how an app assembles a region without inventing syntax for it.
//
// The children are the caller's blocks; nothing here is parsed. It returns false for an
// index that is not a disclosure, because silently attaching to a paragraph would produce
// a document nobody can explain.
func (mv *MarkdownView) SetChildren(index int, children []block) bool {
	if index < 0 || index >= len(mv.blocks) {
		return false
	}
	if !isDisclosure(mv.blocks[index]) {
		return false
	}
	if !mv.setChildrenInTree(index, children) {
		return false
	}
	mv.blocks = mv.flatten(mv.blocks[:0])
	return true
}

// setChildrenInTree attaches to the nth VISIBLE disclosure IN THE TREE.
//
// The tree, not the visible list: the visible list is rebuilt from the tree on every
// toggle, so children attached to it would be gone by the next toggle — which is
// exactly what the first version of SetChildren did, and why a region built this way
// appeared to work and then silently emptied itself.
func (mv *MarkdownView) setChildrenInTree(nth int, children []block) bool {
	// Block indices, for the same reason as toggleInTree.
	n := 0
	var walk func(bs []block) bool
	walk = func(bs []block) bool {
		for i := range bs {
			if n == nth {
				if !isDisclosure(bs[i]) {
					return false
				}
				bs[i].Children = children
				return true
			}
			n++
			if isDisclosure(bs[i]) && bs[i].Collapsed {
				continue
			}
			if walk(bs[i].Children) {
				return true
			}
		}
		return false
	}
	return walk(mv.tree)
}

// errorBlock builds the VISIBLE error an unbuildable embed produces.
//
// It carries real TEXT, not a string in a side field, and that is the whole point: an error
// that draws itself in a bespoke paint case is an error whose position, selection and
// hit-testing were each maintained separately — and the first version drew itself a line
// below its own block, which on a busy document put it ON TOP of the paragraph after it.
// As an ordinary text block it goes through the same line layout and the same painter as
// everything else, and there is nothing to keep in step.
func (mv *MarkdownView) errorBlock(msg string, font graphics.Font, cs, ce int) block {
	t := graphics.NewText(msg, &graphics.TextDecoration{
		Font: font, OnBackgroundInk: mv.MarkdownTheme.Theme().TaskDoneInk,
	})
	return block{Type: blockError, Text: t, SrcStart: cs, SrcEnd: ce}
}

// walkList renders a list node: numbered markers for ordered lists,
// indented recursion for nested lists.
func (mv *MarkdownView) walkList(l *ast.List, src []byte, indent int, quoteDepth int) {
	font := theme.DefaultTheme.BaseFont
	if font == nil {
		font = graphics.SystemFont
	}
	num := 1
	if l.Start > 0 {
		num = l.Start
	}
	for li := l.FirstChild(); li != nil; li = li.NextSibling() {
		if li.Kind() != ast.KindListItem {
			continue
		}
		// Inline text of the item's first paragraph only — walking the
		// whole item would inline nested-list content too.
		var texts []renderNode
		var itemSrcStart, itemSrcEnd = -1, -1
		if tb := li.FirstChild(); tb != nil {
			switch tb.Kind() {
			case ast.KindTextBlock, ast.KindParagraph:
				texts = mv.collectInline(tb, src)
				itemSrcStart, itemSrcEnd = srcRangeOf(tb, src)
			}
		}
		if len(texts) > 0 {
			mv.blocks = append(mv.blocks, block{Type: blockListItem, Level: indent, Ordered: l.IsOrdered(), Number: num, Text: buildText(texts, font, mv.MarkdownTheme), Nodes: texts, SrcStart: itemSrcStart, SrcEnd: itemSrcEnd})
		}
		if l.IsOrdered() {
			num++
		}
		// Nested lists inside the item render indented below it.
		for gc := li.FirstChild(); gc != nil; gc = gc.NextSibling() {
			if gc.Kind() == ast.KindList {
				mv.walkList(gc.(*ast.List), src, indent+1, quoteDepth)
			}
		}
	}
}

// walkBlockDepth walks a node's children as blocks. skipChildren resumes a partial walk
// by passing over the first N children — the ones whose blocks are already in mv.blocks —
// and record, for the TOP level only, how many blocks each child produced.
//
// The recording is what turns "keep the first N blocks" into "skip the first M children":
// block indices and children are not in one-to-one correspondence, because a list or a
// quote produces many blocks from one child.
func (mv *MarkdownView) walkBlockDepth(node ast.Node, src []byte, indent, quoteDepth, skipChildren int, top bool) {
	font := theme.DefaultTheme.BaseFont
	if font == nil {
		font = graphics.SystemFont
	}
	i := 0
	for child := node.FirstChild(); child != nil; child, i = child.NextSibling(), i+1 {
		if i < skipChildren {
			continue
		}
		switch child.Kind() {
		case ast.KindParagraph:
			texts := mv.collectInline(child, src)
			if len(texts) > 0 {
				ps, pe := mv.srcRangeOf(child, src)
				mv.blocks = append(mv.blocks, block{Type: blockParagraph, QuoteDepth: quoteDepth, Text: buildText(texts, font, mv.MarkdownTheme), Nodes: texts, SrcStart: ps, SrcEnd: pe})
			}
		case ast.KindHeading:
			h := child.(*ast.Heading)
			texts := mv.collectInline(child, src)
			hf := headingFont(headingSizes[max(0, min(h.Level-1, 5))])
			hs, he := mv.srcRangeOf(child, src)
			mv.blocks = append(mv.blocks, block{Type: blockHeading, Level: h.Level, Text: buildText(texts, hf, mv.MarkdownTheme), Nodes: texts, SrcStart: hs, SrcEnd: he})
		case ast.KindFencedCodeBlock:
			cb := child.(*ast.FencedCodeBlock)
			var lines []string
			for i := 0; i < cb.Lines().Len(); i++ {
				seg := cb.Lines().At(i)
				// A line segment keeps its newline; joining the raw values
				// would insert a blank line between every pair of code lines.
				lines = append(lines, strings.TrimSuffix(string(seg.Value(src)), "\n"))
			}
			cs, ce := mv.srcRangeOf(child, src)
			body := strings.Join(lines, "\n")
			// An EMBED, or a code block. The whole info string is needed, not just the
			// language: goldmark's Language() is the FIRST WORD, so reading only that is
			// how a widget's parameters would be silently dropped.
			if b, ok := mv.embedBlock(cb, src, body, cs, ce); ok {
				mv.blocks = append(mv.blocks, b)
			} else {
				mv.blocks = append(mv.blocks, block{Type: blockCode, CodeText: body,
					Lang: string(cb.Language(src)), SrcStart: cs, SrcEnd: ce})
			}
		case ast.KindCodeBlock:
			cb := child.(*ast.CodeBlock)
			var lines []string
			for i := 0; i < cb.Lines().Len(); i++ {
				seg := cb.Lines().At(i)
				lines = append(lines, strings.TrimSuffix(string(seg.Value(src)), "\n"))
			}
			cs, ce := mv.srcRangeOf(child, src)
			mv.blocks = append(mv.blocks, block{Type: blockCode, CodeText: strings.Join(lines, "\n"), SrcStart: cs, SrcEnd: ce})
		case ast.KindBlockquote:
			mv.walkBlockDepth(child, src, indent, quoteDepth+1, 0, false)
		case ast.KindList:
			mv.walkList(child.(*ast.List), src, indent, quoteDepth)
		case ast.KindTextBlock:
			// A TextBlock holds inline content directly (definition
			// descriptions, and any container reached outside walkList).
			// Recursing past it would drop its text, so render it as a
			// paragraph.
			texts := mv.collectInline(child, src)
			if len(texts) > 0 {
				ts, te := mv.srcRangeOf(child, src)
				mv.blocks = append(mv.blocks, block{Type: blockParagraph, QuoteDepth: quoteDepth, Text: buildText(texts, font, mv.MarkdownTheme), Nodes: texts, SrcStart: ts, SrcEnd: te})
			}
		case ast.KindThematicBreak:
			mv.blocks = append(mv.blocks, block{Type: blockRule})
		case astex.KindTable:
			mv.walkTable(child, src)
		case ast.KindHTMLBlock:
			// Skip HTML blocks
		case ast.KindImage:
			// Handled in collectInline
		case astex.KindDefinitionList:
			mv.walkBlockDepth(child, src, indent, quoteDepth, 0, false)
		case astex.KindDefinitionTerm:
			texts := mv.collectInline(child, src)
			if len(texts) > 0 {
				// Render definition term as bold paragraph
				for i := range texts {
					texts[i].Bold = true
				}
				ds, de := mv.srcRangeOf(child, src)
				mv.blocks = append(mv.blocks, block{Type: blockParagraph, QuoteDepth: quoteDepth, Text: buildText(texts, font, mv.MarkdownTheme), Nodes: texts, SrcStart: ds, SrcEnd: de})
			}
		case astex.KindDefinitionDescription:
			mv.walkBlockDepth(child, src, indent+1, quoteDepth, 0, false)
		case astex.KindFootnoteList:
			mv.walkBlockDepth(child, src, indent, quoteDepth, 0, false)
		case astex.KindFootnote:
			// Definitions walk in goldmark's index order.
			mv.footnoteCounter++
			texts := mv.collectInline(child, src)
			prefix := renderNode{Text: fmt.Sprintf("[%d] ", mv.footnoteCounter), Code: true}
			texts = append([]renderNode{prefix}, texts...)
			fs, fe := mv.srcRangeOf(child, src)
			mv.blocks = append(mv.blocks, block{Type: blockListItem, Level: 0, Text: buildText(texts, font, mv.MarkdownTheme), Nodes: texts, SrcStart: fs, SrcEnd: fe})
		default:
			mv.walkBlockDepth(child, src, indent, quoteDepth, 0, false)
		}
		if top {
			// How many blocks this child produced. It is what turns "keep the first N
			// blocks" into "skip the first M children" when a partial walk resumes.
			mv.rootEnds = append(mv.rootEnds, len(mv.blocks))
		}
	}
}

// walkTable collects table rows and cells, storing each row as a block.
// Header rows use blockTableHeader, data rows blockTableRow.
func (mv *MarkdownView) walkTable(node ast.Node, src []byte) {
	font := theme.DefaultTheme.BaseFont
	if font == nil {
		font = graphics.SystemFont
	}
	for row := node.FirstChild(); row != nil; row = row.NextSibling() {
		isHeader := row.Kind() == astex.KindTableHeader
		if !isHeader && row.Kind() != astex.KindTableRow {
			continue
		}
		var cells []string
		var cellSrc [][2]int
		var cellTexts []*graphics.Text
		var maxCols int
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			if cell.Kind() == astex.KindTableCell {
				tc := cell.(*astex.TableCell)
				texts := mv.collectInline(cell, src)
				var cellStr string
				for _, rn := range texts {
					cellStr += rn.Text
				}
				cs, ce := srcRangeOf(cell, src)
				cells = append(cells, cellStr)
				cellSrc = append(cellSrc, [2]int{cs, ce})
				_ = tc.Alignment
				_ = cellTexts
				maxCols++
			}
		}
		if maxCols > 0 {
			rs, re := srcRangeOf(row, src)
			if isHeader {
				mv.blocks = append(mv.blocks, block{Type: blockTableHeader, Cells: cells, CellSrc: cellSrc, SrcStart: rs, SrcEnd: re})
			} else {
				mv.blocks = append(mv.blocks, block{Type: blockTableRow, Cells: cells, CellSrc: cellSrc, SrcStart: rs, SrcEnd: re})
			}
		}
	}
}

func (mv *MarkdownView) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	// Every ink comes from the theme. These were literals — VS Code's dark+ palette —
	// which is why a document looked the same in every theme and a light one got a dark
	// code block. See theme/markdown_theme.go.
	mt := mv.MarkdownTheme.Theme()
	if !mv.Visible() {
		return
	}
	// Apply anything Append buffered BEFORE the empty check, or a document that is
	// nothing but new text would never be walked at all.
	mv.ensureParsed()
	if len(mv.blocks) == 0 {
		return
	}
	// The page, when the palette has one. Painted first and across the whole widget, so
	// a light document is a DOCUMENT rather than light text on whatever the host is.
	// A transparent palette skips it, which is what keeps a dark document blending into
	// its host the way it always has.
	if mt.Background != 0 {
		page := graphics.NewPaint()
		page.SetColor(mt.Background)
		canvas.DrawRect(graphics.NewRect(rect.X, rect.Y, rect.Width, rect.Height), page)
	}
	if mv.contentH < 1 {
		mv.computeContentH()
	}
	if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
		for i, blk := range mv.blocks {
			if blk.Type == blockListItem {
				fmt.Printf("[mdblks] i=%d lvl=%d ordered=%v num=%d\n", i, blk.Level, blk.Ordered, blk.Number)
			}
		}
	}
	mv.cache.GetCached(canvas, rect, func(c *graphics.Canvas, r graphics.Rect) {
		c.Save()
		// (selection drawn as overlay after GetCached below)

		font := theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
		lineH := font.LineHeight()
		ascent := font.Baseline()
		if lineH < 1 {
			lineH = 20
		}
		margin := float32(12)

		tp := graphics.NewPaint()
		tp.SetColor(mt.TextInk)
		cp := graphics.NewPaint()
		cp.SetColor(mt.TaskDoneInk)
		lp := graphics.NewPaint()
		lp.SetColor(mt.LinkInk)
		codeBg := graphics.NewPaint()
		codeBg.SetColor(mt.CodeBackground)

		mv.linkRects = mv.linkRects[:0]
		mv.taskRects = mv.taskRects[:0]
		// Vertical positions come from the layout table (markdown_layout.go):
		// the same table hit-testing and selection read, so the highlight can
		// never drift from the painted text.
		mv.ensureLayoutFor(r.Width)
		baseY := r.Y + margin - mv.scrollOff
		for bi := range mv.blocks {
			blk := &mv.blocks[bi]
			y := baseY + mv.blockYs[bi]
			lh := mv.blockHs[bi]

			switch blk.Type {
			case blockParagraph, blockError:
				if y+lh < r.Y {
					continue
				}
				if y > r.Y+r.Height {
					continue
				}
				// Blockquote bars
				for d := blk.QuoteDepth; d > 0; d-- {
					bq := graphics.NewPaint()
					bq.SetColor(mt.QuoteInk)
					c.DrawRect(graphics.NewRect(r.X+margin+float32(d-1)*6-4, y-2, 3, lh-4), bq)
				}
				// A block with a pre-built run and no nodes draws that run. This is the
				// ERROR block's only path: an error is ordinary text, so it goes through
				// here rather than through a paint case of its own, and the run it carries
				// is built with the error ink. Without this guard the error laid out
				// correctly and drew nothing at all, because a paragraph is otherwise
				// rendered from its NODES.
				if len(blk.Nodes) == 0 && blk.Text != nil {
					blk.Text.Draw(c, r.X+margin, y+font.Baseline())
					continue
				}
				// Render nodes (text + images) inline, handling newlines
				x := r.X + margin + float32(blk.QuoteDepth)*6
				lineStartX := x
				lineCount := 1
				for _, rn := range blk.Nodes {
					// Check for newline in text → split and render each line
					if strings.Contains(rn.Text, "\n") {
						parts := strings.Split(rn.Text, "\n")
						for pi, part := range parts {
							if pi > 0 {
								// Start new line
								x = lineStartX
								y += lineH
								lineCount++
							}
							if part == "" {
								continue
							}
							mv.renderNode(c, rn, part, &x, y, font, tp, cp, lp)
						}
						continue
					}
					mv.renderNode(c, rn, rn.Text, &x, y, font, tp, cp, lp)
				}
			case blockHeading:
				idx := blk.Level - 1
				if idx < 0 {
					idx = 0
				}
				if idx > 5 {
					idx = 5
				}
				if y+lh < r.Y {
					continue
				}
				if y > r.Y+r.Height {
					continue
				}
				hf := headingFont(headingSizes[idx])
				hAsc := hf.Baseline()
				if blk.Text != nil {
					blk.Text.Draw(c, r.X+margin, y+hAsc)

				}
			case blockWidget:
				// A widget is ATOMIC: it has no text for a caret to sit in, so the
				// selection is one rect over it, and this is the only place the painter
				// needs to know a block was not text.
				//
				// A COLLAPSED widget paints NOTHING — not a hidden body, not a
				// placeholder. That is the property that makes a transcript of six
				// hundred embedded steps scroll, and the layout has already reserved
				// exactly one header's height for it.
				// The header comes first and is the container's: chevron plus the
				// one-line summary the WIDGET supplied. The widget is given the body
				// rect below it, never the whole block, so a widget cannot draw over
				// its own disclosure.
				head := mv.laid[bi].headH
				mv.paintDisclosureHeader(canvas, blk, graphics.NewRect(r.X, y, r.Width, head), mt, font)
				if !blk.Collapsed && blk.Widget != nil {
					body := graphics.NewRect(r.X, y+head, r.Width, lh-head)
					if body.Height < 0 {
						body.Height = 0
					}
					blk.Widget.Paint(canvas, body, &WidgetContext{
						Name: blk.WidgetName, Params: blk.Params, Body: blk.Body,
						Width: r.Width, Theme: mt,
					})
				}
			case blockCode:
				// Use monospace font for code blocks
				codeFont := font
				if graphics.MonospacedFont != nil {
					if face := graphics.MonospacedFont.Face(); face != nil {
						codeFont = face.Font(6)
					}
				}
				clines := strings.Split(blk.CodeText, "\n")
				ch := float32(len(clines)) * codeFont.LineHeight()
				labelAdv := float32(0)
				if blk.Lang != "" {
					labelAdv = codeFont.LineHeight()
				}
				// Language label
				labelH := float32(0)
				if blk.Lang != "" {
					labelH = codeFont.LineHeight() * 0.5
				}
				if y+ch+labelH < r.Y {
					continue
				}
				if y > r.Y+r.Height {
					continue
				}
				ly := y
				c.DrawRect(graphics.NewRect(r.X+margin-6, ly-4, r.Width-margin*2+12, ch+labelH+8), codeBg)
				if blk.Lang != "" {
					lp := graphics.NewPaint()
					lp.SetColor(mt.CodeLabelInk)
					lt := graphics.NewText(blk.Lang, &graphics.TextDecoration{Font: codeFont, OnBackgroundInk: lp.Color()})
					lt.Draw(c, r.X+margin, ly+codeFont.Baseline())
					ly += labelAdv
				}
				for _, cl := range clines {
					highlightCodeLine(cl, mt.CodeTextInk, codeFont, c, r.X+margin, ly+codeFont.Baseline())
					ly += codeFont.LineHeight()
				}
			case blockListItem:
				if y+lineH < r.Y {
					continue
				}
				if y > r.Y+r.Height {
					continue
				}
				indent := float32(blk.Level) * 20
				marker := "•"
				if blk.Ordered {
					marker = fmt.Sprintf("%d.", blk.Number)
				}
				tx := r.X + margin + indent
				bt := graphics.NewText(marker+" ", &graphics.TextDecoration{Font: font, OnBackgroundInk: tp.Color()})
				bt.Draw(c, tx, y+ascent)
				tx += font.SimpleWidth(marker + " ")
				for _, rn := range blk.Nodes {
					mv.renderNode(c, rn, rn.Text, &tx, y, font, tp, cp, lp)
				}
			case blockRule:
				if y+lineH < r.Y {
					continue
				}
				if y > r.Y+r.Height {
					continue
				}
				hr := graphics.NewPaint()
				hr.SetColor(mt.RuleInk)
				hr.SetStyle(paintstyle.Stroke)
				c.DrawLine(r.X+margin, y+lineH/2, r.X+r.Width-margin, y+lineH/2, hr)
			case blockTableRow:
				fallthrough
			case blockTableHeader:
				if y+lineH < r.Y {
					continue
				}
				if y > r.Y+r.Height {
					continue
				}
				if len(blk.Cells) == 0 {
					continue
				}
				// Column widths and the row extent come from the layout pass.
				colWidths := mv.laid[bi].cols
				totalW := mv.laid[bi].totalW
				if len(colWidths) == 0 {
					continue
				}

				// Row index for alternating colors. This reproduces the
				// original (buggy) count — every row sees the total number of
				// table rows, so the stripes do not actually alternate. Fixing
				// it changes pixels, so it stays a separate, visible change.
				rowIdx := 0
				for _, b := range mv.blocks {
					if b.Type == 6 || b.Type == 7 {
						rowIdx++
					}
				}

				// Table cell background (alternating)
				rowBg := graphics.NewPaint()
				if blk.Type == blockTableHeader {
					rowBg.SetColor(mt.TableHeaderBackground)
				} else if rowIdx%2 == 0 {
					rowBg.SetColor(mt.TableRowEvenBackground)
				} else {
					rowBg.SetColor(mt.TableRowOddBackground)
				}
				rowRect := graphics.NewRect(r.X+margin, y-1, totalW, lineH+6)
				c.DrawRect(rowRect, rowBg)

				// Render cells
				x := r.X + margin
				for ci, cell := range blk.Cells {
					cw := colWidths[ci]
					// Cell vertical border
					if ci > 0 {
						sep := graphics.NewPaint()
						sep.SetColor(mt.TableBorderInk)
						sep.SetStyle(paintstyle.Stroke)
						c.DrawLine(x, y-2, x, y+lineH+4, sep)
					}
					if blk.Type == blockTableHeader {
						hp := graphics.NewPaint()
						hp.SetColor(mt.TableHeaderInk)
						ct := graphics.NewText(strings.TrimSpace(cell), &graphics.TextDecoration{Font: font, OnBackgroundInk: hp.Color()})
						ct.Draw(c, x+6, y+ascent)
						// Header bottom border
						hl := graphics.NewPaint()
						hl.SetColor(mt.TableHeaderRuleInk)
						hl.SetStyle(paintstyle.Stroke)
						c.DrawLine(x, y+lineH+3, x+cw, y+lineH+3, hl)
					} else {
						ct := graphics.NewText(strings.TrimSpace(cell), &graphics.TextDecoration{Font: font, OnBackgroundInk: tp.Color()})
						ct.Draw(c, x+6, y+ascent)
					}
					x += cw
				}

				// Table outer borders (bottom of this row)
				border := graphics.NewPaint()
				border.SetColor(mt.TableBorderInk)
				border.SetStyle(paintstyle.Stroke)
				// Bottom border of this row
				c.DrawLine(r.X+margin, y+lineH+3, r.X+margin+totalW, y+lineH+3, border)
				// Left and right outer borders (only on first row)
				if rowIdx == 0 {
					c.DrawLine(r.X+margin, y-1, r.X+margin, y+lineH+4, border)
					c.DrawLine(r.X+margin+totalW, y-1, r.X+margin+totalW, y+lineH+4, border)
				}
				// Top border (only on first row)
				if rowIdx == 0 {
					c.DrawLine(r.X+margin, y-1, r.X+margin+totalW, y-1, border)
				}
			}
		}

		// Scrollbar via host: content rect r (cache-local 0,0,w,h),
		// viewport r.Height — host uses absolute extent for thumb scale.
		mv.host.SetExtents(r.Width, mv.contentH, r.Width, r.Height)
		mv.host.LayoutBars(r.Width, r.Height)
		_, mv.scrollOff = mv.host.Offsets()
		mv.host.PaintBars(c)
		c.Restore()
	})

	// Selection highlight, from the same run geometry the text was drawn
	// from, clipped to the widget.
	if a, b := mv.SelectionBounds(); b > a {
		hl := graphics.NewPaint()
		hl.SetColor(mt.SearchHighlightInk)
		for _, sr := range mv.SelectionRects(a, b) {
			r := sr
			r.X += rect.X
			r.Y += rect.Y
			if r.Y < rect.Y {
				r.Height -= rect.Y - r.Y
				r.Y = rect.Y
			}
			if r.Y+r.Height > rect.Y+rect.Height {
				r.Height = rect.Y + rect.Height - r.Y
			}
			if r.Height <= 0 || r.Width <= 0 {
				continue
			}
			canvas.DrawRect(r, hl)
		}
	}
}

func (mv *MarkdownView) loadImage(path string) *graphics.Image {
	if path == "" {
		return nil
	}
	if img, ok := mv.imgCache[path]; ok {
		return img
	}
	img, err := graphics.NewImageFromFilePathOrURL(path, 1)
	if err != nil {
		return nil
	}
	mv.imgCache[path] = img
	return img
}

// renderNode draws a single renderNode at the given position, advancing x.
func (mv *MarkdownView) renderNode(c *graphics.Canvas, rn renderNode, text string, x *float32, y float32, font graphics.Font, tp, cp, lp *graphics.Paint) {
	if text == "" {
		return
	}
	if rn.Task {
		w := font.SimpleWidth(text)
		mv.taskRects = append(mv.taskRects, taskRect{
			rect:    graphics.NewRect(*x, y, w, font.LineHeight()),
			index:   rn.TaskIndex,
			checked: rn.Checked,
		})
	} else if rn.ImageAlt == "" && rn.Link != "" {
		w := font.SimpleWidth(text)
		mv.linkRects = append(mv.linkRects, linkRect{
			rect: graphics.NewRect(*x, y, w, font.LineHeight()),
			url:  rn.Link,
		})
	}
	if rn.ImageAlt != "" {
		img := mv.loadImage(rn.Link)
		if img != nil {
			maxW := float32(400)
			imgH := float32(80)
			imgW := maxW
			if s := img.LogicalSize(); s.Width > 0 && s.Height > 0 {
				scale := min(maxW/s.Width, 80/s.Height)
				imgW = s.Width * scale
				imgH = s.Height * scale
			}
			dst := graphics.NewRect(*x, y, imgW, imgH)
			c.DrawImageInRect(img, dst, nil, nil)
			*x += imgW + 4
			return
		}
		if rn.textObj == nil {
			rn.textObj = graphics.NewText("["+rn.ImageAlt+"]", &graphics.TextDecoration{Font: font, OnBackgroundInk: cp.Color()})
		}
		rn.textObj.Draw(c, *x, y+font.Baseline())
		*x += font.SimpleWidth("[" + rn.ImageAlt + "]")
		return
	}
	f, p := font, tp
	if rn.Code || rn.Link != "" {
		if rn.textObj == nil {
			paint := tp
			if rn.Code {
				paint = cp
			} else {
				paint = lp
			}
			rn.textObj = graphics.NewText(text, &graphics.TextDecoration{Font: font, OnBackgroundInk: paint.Color()})
		}
		rn.textObj.Draw(c, *x, y+font.Baseline())
		*x += font.SimpleWidth(text)
		return
	}
	if rn.Bold || rn.Italic {
		fd := f.Descriptor()
		if rn.Bold {
			fd.Weight = weight.Black
		}
		if rn.Italic {
			fd.Slant = slant.Italic
		}
		f = fd.Font()
	}
	if rn.Sup || rn.Sub {
		fd := f.Descriptor()
		fd.Size *= 0.75
		f = fd.Font()
	}
	mt := mv.MarkdownTheme.Theme()
	if rn.Strike {
		p.SetColor(mt.CodeLabelInk)
	}
	if rn.Highlight {
		// A translucent band behind the glyphs — the ink is the theme's, because a
		// hardcoded "Dracula yellow" is yellow on a light theme too.
		hp := graphics.NewPaint()
		hp.SetColor(mt.SearchHighlightStrongInk)
		c.DrawRect(graphics.NewRect(*x, y, f.SimpleWidth(text), font.LineHeight()), hp)
	}
	// Use cached Text object for emoji/Unicode font fallback
	if rn.textObj == nil {
		rn.textObj = graphics.NewText(text, &graphics.TextDecoration{Font: f, OnBackgroundInk: p.Color()})
	}
	dy := float32(0)
	if rn.Sup {
		dy = -font.LineHeight() * 0.22
	} else if rn.Sub {
		dy = font.LineHeight() * 0.28
	}
	rn.textObj.Draw(c, *x, y+dy+f.Baseline())
	*x += f.SimpleWidth(text)
}

func (mv *MarkdownView) ScrollBy(delta float32) {
	frame := mv.FrameRect()
	mv.host.SetExtents(frame.Width, mv.contentH, frame.Width, frame.Height)
	_, offY := mv.host.Offsets()
	mv.host.SetOffsets(0, offY+delta)
	_, mv.scrollOff = mv.host.Offsets()
	mv.cache.Invalidate()
	mv.MarkForRedraw()
}

func (mv *MarkdownView) OnMouseWheel(x, y float32, dx, dy float32) bool {
	mv.ScrollBy(-dy * 24)
	return true
}

// ── Text selection + clipboard ──────────────────────────────────

// toggleTextTask flips the index-th task checkbox marker ("[ ]"/"[x]")
// in the markdown source and returns the rewritten source. ok is false
// when the document contains fewer task checkboxes than index+1.
func toggleTextTask(src string, index int) (string, bool) {
	occ := -1
	for i := 0; i+2 < len(src); i++ {
		if src[i] != '[' || src[i+2] != ']' {
			continue
		}
		mid := src[i+1]
		if mid != ' ' && mid != 'x' && mid != 'X' {
			continue
		}
		occ++
		if occ == index {
			out := src[:i]
			if mid == ' ' {
				out += "[x]"
			} else {
				out += "[ ]"
			}
			out += src[i+3:]
			return out, true
		}
	}
	return "", false
}
