package theme

import "github.com/sixnineone/meui/graphics"

// MarkdownTheme is every colour, metric and font the Markdown renderer decides.
//
// WHY IT EXISTS. The renderer used to decide all of these ITSELF: thirty colour literals
// in widgets/markdown.go, and they were VS Code's dark+ palette exactly (#1E1E1E for a
// code block, #569CD6 for a link, #6A9955 for a constant). So a document rendered the
// same in every theme, a light theme got a dark code block, and no theme could restyle
// it at all. AGENTS.md already requires this of chrome — "nothing in widgets hardcodes a
// chrome colour" — and a Markdown document is not less chrome than a title bar.
//
// It follows the widget-theme convention: a global DefaultMarkdownTheme that the
// importer overwrites, and widgets embedding a pointer to it. That is how TabTheme and
// every other widget theme in this package already work.
type MarkdownTheme struct {
	// Background is the page the document is drawn ON, and it is the slot that makes a
	// palette self-contained.
	//
	// Without it the renderer themes its INKS but still draws on whatever is behind it,
	// so a light palette inside a dark app is light text on a dark page: the table and
	// the code block come out right and the prose is invisible. A document is a surface,
	// and the surface is the theme's.
	//
	// The DARK palette leaves it transparent, which is deliberate and not an oversight:
	// a transparent document blends into whatever hosts it, which is what it has always
	// done, and the render-parity corpus has markdown scenes whose pixels are baselines.
	// The LIGHT palette is opaque, because a light document on a dark host is the case
	// that needs it.
	Background graphics.Color

	// ── text ──
	TextInk         graphics.Color // body text
	HeadingInk      []graphics.Color
	ListMarkerInk   graphics.Color
	QuoteInk        graphics.Color // the blockquote bar and its text
	LinkInk         graphics.Color
	QuoteBackground graphics.Color

	// ── code ──
	CodeTextInk          graphics.Color
	CodeBackground       graphics.Color
	CodeLabelInk         graphics.Color // the fence's language label
	InlineCodeInk        graphics.Color
	InlineCodeBackground graphics.Color

	// ── tables ──
	TableHeaderInk         graphics.Color
	TableHeaderBackground  graphics.Color
	TableRowEvenBackground graphics.Color
	TableRowOddBackground  graphics.Color
	TableBorderInk         graphics.Color // cell separators and the outer frame
	TableHeaderRuleInk     graphics.Color // the rule under the header row
	TableCellPadX          float32
	TableCellPadY          float32
	TableRowMinHeight      float32

	// ── rules, marks and highlights ──
	RuleInk                  graphics.Color
	TaskDoneInk              graphics.Color
	TaskTodoInk              graphics.Color
	SearchHighlightInk       graphics.Color
	SearchHighlightStrongInk graphics.Color

	// ── metrics ──
	CodePadX, CodePadY float32
	BlockGap           float32
	QuoteBarWidth      float32
	QuotePadX          float32
	HeadingSizes       []float32

	// ── fonts ──
	Font     graphics.Font
	MonoFont graphics.Font
}

// Copy deep-copies the theme, so a widget that changes a metric cannot change the
// global. The slice fields are copied too, which is the part that is easy to forget and
// produces a theme that changes when an unrelated widget does.
func (t *MarkdownTheme) Copy() *MarkdownTheme {
	if t == nil {
		return nil
	}
	c := *t
	if t.HeadingInk != nil {
		c.HeadingInk = append([]graphics.Color(nil), t.HeadingInk...)
	}
	if t.HeadingSizes != nil {
		c.HeadingSizes = append([]float32(nil), t.HeadingSizes...)
	}
	return &c
}

// Theme returns the theme, or the default if a widget embedded a nil one. A widget
// built before any theme was set must still paint, and a nil dereference in a paint
// path takes the whole window down.
func (t *MarkdownTheme) Theme() *MarkdownTheme {
	if t == nil {
		return DefaultMarkdownTheme
	}
	return t
}

// DefaultMarkdownTheme is the DARK renderer palette, kept close to what the renderer
// used to hardcode so the first themed render is a small change rather than a new look.
var DefaultMarkdownTheme = &MarkdownTheme{
	// Transparent on purpose: a document blends into its host, which is what it has
	// always done and what the parity corpus's markdown scenes are baselined against.
	Background:      graphics.Color(0x00000000),
	TextInk:         graphics.Color(0xFFD4D4D4),
	HeadingInk:      []graphics.Color{0xFFE8E8E8, 0xFFE8E8E8, 0xFFE0E0E0, 0xFFD8D8D8, 0xFFD0D0D0, 0xFFCCCCCC},
	ListMarkerInk:   graphics.Color(0xFF888888),
	QuoteInk:        graphics.Color(0xFF569CD6),
	QuoteBackground: graphics.Color(0x00000000),
	LinkInk:         graphics.Color(0xFF569CD6),

	CodeTextInk:          graphics.Color(0xFFD4D4D4),
	CodeBackground:       graphics.Color(0xFF1E1E1E),
	CodeLabelInk:         graphics.Color(0xFF888888),
	InlineCodeInk:        graphics.Color(0xFFD4D4D4),
	InlineCodeBackground: graphics.Color(0xFF2A2A2A),

	TableHeaderInk:         graphics.Color(0xFFE8E8E8),
	TableHeaderBackground:  graphics.Color(0xFF2A2A3A),
	TableRowEvenBackground: graphics.Color(0xFF1E1E1E),
	TableRowOddBackground:  graphics.Color(0xFF222222),
	TableBorderInk:         graphics.Color(0xFF3A3A3A),
	TableHeaderRuleInk:     graphics.Color(0xFF555555),
	TableCellPadX:          8,
	TableCellPadY:          4,
	TableRowMinHeight:      20,

	RuleInk:                  graphics.Color(0xFF555555),
	TaskDoneInk:              graphics.Color(0xFF6A9955),
	TaskTodoInk:              graphics.Color(0xFF888888),
	SearchHighlightInk:       graphics.Color(0x445064A1),
	SearchHighlightStrongInk: graphics.Color(0x59F1FA8C),

	CodePadX:      10,
	CodePadY:      8,
	BlockGap:      8,
	QuoteBarWidth: 3,
	QuotePadX:     10,
	HeadingSizes:  []float32{32, 24, 20, 16, 14, 12},
}

// LightMarkdownTheme is the DeepSeek harness's document palette, and every value in it
// was READ OFF THE REFERENCE SCREENSHOT rather than chosen.
//
// Measured from Screenshot_20260929_231734.png by sampling pixels and counting them,
// because "looks like" is not a specification: the page is 74.6% #FFFFFF, #F9FAFB is
// 13.5% (the panel and alternating rows), #E5E5E5 and #ECEEF0 are the borders, and
// #0F1115 is the text. The accent is #4176E6 — the most saturated colour in the image at
// 1308 pixels, and the one deliberate choice among the antialiasing.
//
// The complete mapping, with counts, is in LOOK-REFERENCE.md.
var LightMarkdownTheme = &MarkdownTheme{
	// Opaque: a light document is a PAGE, and the harness's document pane is white.
	Background:      graphics.Color(0xFFFFFFFF),
	TextInk:         graphics.Color(0xFF0F1115), // 5665 px: the dominant dark
	HeadingInk:      []graphics.Color{0xFF0F1115, 0xFF0F1115, 0xFF0F1115, 0xFF3C3E41, 0xFF525256, 0xFF787A7C},
	ListMarkerInk:   graphics.Color(0xFF81858C), // 994 px: the mid grey beside a row's kind
	QuoteInk:        graphics.Color(0xFF525256), // 1712 px
	QuoteBackground: graphics.Color(0x00000000),
	LinkInk:         graphics.Color(0xFF4176E6), // 1308 px, sat 165: the accent

	CodeTextInk:          graphics.Color(0xFF3C3E41), // 826 px
	CodeBackground:       graphics.Color(0xFFF5F6F7), // 11847 px
	CodeLabelInk:         graphics.Color(0xFF81858C),
	InlineCodeInk:        graphics.Color(0xFF0F1115),
	InlineCodeBackground: graphics.Color(0xFFF5F6F7),

	TableHeaderInk:         graphics.Color(0xFF0F1115),
	TableHeaderBackground:  graphics.Color(0xFFF9FAFB), // 212157 px
	TableRowEvenBackground: graphics.Color(0xFFFFFFFF),
	TableRowOddBackground:  graphics.Color(0xFFFAFAFA), // 22033 px
	TableBorderInk:         graphics.Color(0xFFE5E5E5), // 9100 px
	TableHeaderRuleInk:     graphics.Color(0xFFE0E0E0), // 2699 px
	TableCellPadX:          10,
	TableCellPadY:          6,
	TableRowMinHeight:      26,

	RuleInk:                  graphics.Color(0xFFECEEF0), // 7281 px
	TaskDoneInk:              graphics.Color(0xFF4176E6), // the accent: the reference has no green
	TaskTodoInk:              graphics.Color(0xFF81858C),
	SearchHighlightInk:       graphics.Color(0x334176E6), // the accent, translucent
	SearchHighlightStrongInk: graphics.Color(0x33F5F6F7),

	// The densities are the reference's too, at its captured scale: a trajectory row
	// band is 39px and a document line 43px there, so a header is a little under that. The
	// screenshot is 1716px wide, which is most likely a HiDPI capture, so the logical
	// figures are half — LOOK-REFERENCE.md records both readings rather than picking one.
	CodePadX:      12,
	CodePadY:      10,
	BlockGap:      10,
	QuoteBarWidth: 3,
	QuotePadX:     12,
	HeadingSizes:  []float32{26, 20, 17, 15, 14, 13},
}

// darkMarkdownTheme is a pristine copy of the built-in dark palette, kept so a light
// palette can be UNDONE. Without it, going light is one-way: the dark palette is reached
// only by reloading a preset, and a test that lightened a document would leave every later
// test looking at a light one.
var darkMarkdownTheme = DefaultMarkdownTheme.Copy()

// UseMarkdownPalette selects the built-in palette for a mode. Light is the reference's
// measured palette; dark is the one the renderer used to hardcode.
func UseMarkdownPalette(m Mode) {
	if m == ModeLight {
		DefaultMarkdownTheme = LightMarkdownTheme.Copy()
		return
	}
	DefaultMarkdownTheme = darkMarkdownTheme.Copy()
}

// CurrentMarkdownPaletteIsLight reports which built-in palette is active. A test that
// lightened the renderer and forgot is otherwise invisible.
func CurrentMarkdownPaletteIsLight() bool {
	return DefaultMarkdownTheme.Background != 0
}
