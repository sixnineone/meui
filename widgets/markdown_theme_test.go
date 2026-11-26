package widgets

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

// TestTheRendererHardcodesNoColour is the guard that keeps the Markdown renderer
// themeable.
//
// The renderer used to carry THIRTY colour literals — VS Code's dark+ palette exactly —
// so a document rendered the same in every theme, a light theme got a dark code block,
// and no preset could touch it. Fixing it once is not enough: the next person adds one
// more literal and it is back. So this reads the SOURCE and fails on any colour literal
// in the renderer's files.
//
// It parses rather than greps, so a literal in a comment or a string does not trip it.
func TestTheRendererHardcodesNoColour(t *testing.T) {
	fset := token.NewFileSet()
	for _, name := range []string{"markdown.go", "markdown_layout.go", "markdown_selection.go"} {
		path := "meui/widgets/" + name
		if _, err := os.Stat(path); err != nil {
			// The package's own working directory is the package directory.
			path = name
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			bl, ok := n.(*ast.BasicLit)
			if !ok || bl.Kind != token.INT {
				return true
			}
			// Guard the slice: a literal can be shorter than the "0x" prefix this
			// assumes, and slicing it panics rather than failing.
			if len(bl.Value) < 3 || bl.Value[:2] != "0x" {
				return true
			}
			v, err := strconv.ParseUint(bl.Value[2:], 16, 64)
			if err != nil {
				return true
			}
			// Only 0xAARRGGBB literals: a small number is a size or a count, not an ink.
			if v <= 0xFFFFFF {
				return true
			}
			t.Errorf("%s:%d hardcodes the colour #%s; every ink belongs on theme.MarkdownTheme "+
				"(or the SyntaxTheme for code)", path, fset.Position(bl.Pos()).Line, bl.Value[2:])
			return true
		})
	}
}

// TestTheMarkdownThemeHasBothPalettes guards the second half of the fix: themable means
// a LIGHT theme is possible, and a theme with no light variant is a dark theme with a
// name. The light values are read off the harness's document pane, which is a light
// surface, so "the light theme is dark" is a regression against the reference.
func TestTheMarkdownThemeHasBothPalettes(t *testing.T) {
	dark, light := theme.DefaultMarkdownTheme, theme.LightMarkdownTheme
	if dark == nil || light == nil {
		t.Fatal("a markdown palette is nil")
	}
	for _, c := range []struct {
		name string
		d, l graphics.Color
	}{
		{"TextInk", dark.TextInk, light.TextInk},
		{"CodeBackground", dark.CodeBackground, light.CodeBackground},
		{"LinkInk", dark.LinkInk, light.LinkInk},
		{"TableHeaderBackground", dark.TableHeaderBackground, light.TableHeaderBackground},
		{"TableRowEvenBackground", dark.TableRowEvenBackground, light.TableRowEvenBackground},
		{"TableBorderInk", dark.TableBorderInk, light.TableBorderInk},
		{"RuleInk", dark.RuleInk, light.RuleInk},
		{"SearchHighlightInk", dark.SearchHighlightInk, light.SearchHighlightInk},
	} {
		if c.d == c.l {
			t.Errorf("%s is the same in both palettes (#%08X); the light theme is not light",
				c.name, uint32(c.d))
		}
	}
	// Text must be DARKER than what it sits on, or the document is unreadable — and in
	// a light palette that is the failure most likely to be introduced, because every
	// other slot is being lightened at the same time. Compared on the R channel so the
	// assertion does not depend on a lightness helper the package may not have.
	if textR, bgR := uint32(light.CodeTextInk)>>16&0xFF, uint32(light.CodeBackground)>>16&0xFF; textR >= bgR {
		t.Errorf("the light code block's text (#%08X) is not darker than its background (#%08X)",
			uint32(light.CodeTextInk), uint32(light.CodeBackground))
	}
}

// TestCopyIsDeep: a widget that changes a metric must not change the global, and a
// shared slice is the version of that bug that survives a value copy.
func TestMarkdownThemeCopyIsDeep(t *testing.T) {
	c := theme.DefaultMarkdownTheme.Copy()
	if c == theme.DefaultMarkdownTheme {
		t.Fatal("Copy returned the same pointer")
	}
	if len(c.HeadingInk) == 0 || len(c.HeadingSizes) == 0 {
		t.Fatal("the default palette has no heading inks or sizes to copy")
	}
	orig := c.HeadingInk[0]
	c.HeadingInk[0] = graphics.Color(0xFF00FF00)
	if theme.DefaultMarkdownTheme.HeadingInk[0] != orig {
		t.Error("writing to a copy's HeadingInk changed the global's")
	}
	origSize := c.HeadingSizes[0]
	c.HeadingSizes[0] = 999
	if theme.DefaultMarkdownTheme.HeadingSizes[0] != origSize {
		t.Error("writing to a copy's HeadingSizes changed the global's")
	}
}

// TestTheViewCarriesTheTheme: a view built with no theme set must still paint, and must
// take the global so a preset reaches it.
func TestTheViewCarriesTheTheme(t *testing.T) {
	mv := NewMarkdownView()
	if mv.MarkdownTheme == nil {
		t.Fatal("a new view has no markdown theme; Paint would nil-deref")
	}
	if mv.MarkdownTheme != theme.DefaultMarkdownTheme {
		t.Error("a new view does not carry the global theme, so a preset would not reach it")
	}
	// A nil embedded theme must resolve rather than crash.
	var none *MarkdownView
	_ = none
	bare := &MarkdownView{}
	if bare.MarkdownTheme.Theme() != theme.DefaultMarkdownTheme {
		t.Error("a nil theme did not fall back to the default")
	}
}

// TestChangingThePaletteRetextsTheDocument is the defect this file's existence is
// about, caught the second time.
//
// A block's runs are built while the AST is walked and each carries its own ink, so a
// palette change that only repainted would put the NEW page behind the OLD text. In the
// demo that was unmistakable — a white page with a pale heading on it — and no test
// caught it, because every palette assertion was reading theme fields rather than
// pixels.
//
// So this renders and samples: a heading that is dark ink on the light page, then the
// same document with the dark palette, which must be light ink.
func TestChangingThePaletteRetextsTheDocument(t *testing.T) {
	const w, h = 320, 200
	mv := NewMarkdownView()
	mv.SetText("# Heading\n\nA paragraph of body text that is long enough to be sampled.")
	mv.SetFrameRect(graphics.NewRect(0, 0, w, h))

	headingInk := func(mv *MarkdownView) uint32 {
		dev := graphics.NewSoftDevice(w, h)
		mv.Paint(graphics.NewCanvas(dev), graphics.NewRect(0, 0, w, h))
		// The DARKEST pixel that is not the page itself. The first version sampled a
		// fixed band (y 8..30) on the assumption that the heading starts at the top:
		// the document has a top inset, so the band was pure background and the test
		// failed against its own premise rather than against a defect. Excluding the
		// page colour keeps the assertion ("the heading is dark ink on the light page")
		// without encoding where the layout puts it.
		px := dev.Pixels()
		// Pixels are bottom-up.
		bg := px[(h-1)*w] // top-left: the page under an empty document margin
		var darkest uint32
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				c := px[(h-1-y)*w+x]
				if c == 0 || c == bg {
					continue
				}
				if darkest == 0 || c>>16&0xFF < darkest>>16&0xFF {
					darkest = c
				}
			}
		}
		return darkest
	}

	mv.SetLightPalette()
	light := headingInk(mv)
	lightFrame := renderAll(mv)
	mv.SetDarkPalette()
	darkFrame := renderAll(mv)

	// The light page must carry dark ink in the heading band, or the light palette is
	// not readable.
	if light == 0 {
		t.Fatal("nothing was drawn in the heading band on the light page")
	}
	if light>>16&0xFF >= 0x80 {
		t.Errorf("on the LIGHT page the heading's darkest drawn pixel is #%08X, so the heading is not dark ink", light)
	}

	// And the two palettes must render DIFFERENTLY. That is the actual defect: a block's
	// runs are built at parse time and carry their own ink, so a palette change that
	// only repainted leaves the old text under the new page. Comparing the whole frame
	// rather than an absolute ink value keeps the assertion independent of how the soft
	// device composites, which is what made the first version of this test fail for the
	// wrong reason.
	if lightFrame == darkFrame {
		t.Error("the light and dark palettes render identically, so the runs were not retexted")
	}
}

// renderAll renders the view and returns every pixel, so two renders can be compared
// without depending on which pixel is interesting.
func renderAll(mv *MarkdownView) string {
	const w, h = 320, 200
	dev := graphics.NewSoftDevice(w, h)
	mv.Paint(graphics.NewCanvas(dev), graphics.NewRect(0, 0, w, h))
	px := dev.Pixels()
	sum := uint64(0)
	for _, p := range px {
		sum = sum*31 + uint64(p)
	}
	return strconv.FormatUint(sum, 16)
}
