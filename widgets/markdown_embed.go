package widgets

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

// MarkdownWidget is the contract an EMBEDDED widget implements: something a fence can
// stand for, drawn inside the document in the place the author wrote it.
//
// ## Why the embed is a fence
//
// A fenced code block whose info string names a registered widget and carries its
// parameters:
//
//	```tool-call {"id":"read","timeout":45}
//
//	github.com/sixnineone/meui/widgets/markdown.go
//	```
//
// renders that widget instead of preformatted text. The reason is LOSSLESSNESS: goldmark
// already parses a fence, the block already has a source span, and a fence already
// round-trips through copy as ordinary Markdown. So select-all-then-copy-all stays valid
// Markdown, and a document read by any other renderer shows the embed as a code block
// rather than as nothing. A `:::` directive would need a goldmark extension and a
// representation in the copied source, and would be unreadable everywhere else.
//
// ## The contract is deliberately two methods
//
// Height is asked FIRST and the rect handed to Paint is exactly that height, so a widget
// never derives geometry by painting. The alternative — measuring by painting into a
// throwaway canvas — is how a widget ends up painting twice per frame, and it is not
// optional here: an embedded widget lives inside a DISCLOSURE, and a collapsed one must
// cost its header and nothing else (the same property
// TestACollapsedTranscriptRowCostsNoPixels asserts for a Collapsible). A widget that
// paints during layout fails that test, which is the point.
//
// Input and hit-testing are a later phase. A widget that needs a click registers for it
// then; nothing in v1 requires it, and requiring it up front would put a hit-test method
// on every widget to serve a minority of them.
type MarkdownWidget interface {
	// Height is the height the widget wants, given the width it will be given. It must be
	// cheap and must not paint: it is asked during layout, once per frame at most, for
	// every expanded widget in the document.
	Height(ctx *WidgetContext) float32
	// Paint draws into rect, whose height is exactly what Height returned. Painting
	// outside it is clipped away, so a widget that overruns its own height does not
	// silently overlap the next block.
	Paint(canvas *graphics.Canvas, rect graphics.Rect, ctx *WidgetContext)
}

// WidgetContext is what a widget is told about its own embedding. It is a value, not a
// pointer to the view, so a widget cannot reach the document and does not outlive it.
type WidgetContext struct {
	// Name is the widget's registered name, as written in the fence's info string.
	Name string
	// Params are the parsed JSON object from the info string. A widget that ignores a
	// parameter's TYPE fails silently, which is why they are parsed as JSON and
	// validated per widget rather than left as a string map.
	Params map[string]any
	// Body is the fence's content — the widget's own input, e.g. the path a tool call
	// read. It is NOT Markdown: a widget owns its body's meaning.
	Body string
	// Width is the width the widget will be given, so a widget can decide a height that
	// depends on wrapping without a second pass.
	Width float32
	// Theme is the renderer's palette, so a widget is themed the same way the document
	// around it is rather than inventing its own.
	Theme *theme.MarkdownTheme
	// Selected is true when the selection covers this block. A widget is ATOMIC — its
	// interior has no text for a caret to sit in — so the selection is one rect over
	// the widget, and a widget that draws its own content can show that.
	Selected bool
}

// FenceResolver is the seam between the renderer and the widget registry.
//
// The registry lives in the meui-markdown module, so `meui` must not import it and the
// module must not be reimplemented inside the toolkit. This is the one call: the renderer
// asks whether a fence's info string names an embeddable widget, and gets back a widget, a
// summary line for the disclosure header, or nothing — which means the fence is a code
// block after all.
//
// A nil resolver is the normal case, and the renderer is then exactly what it was.
var FenceResolver func(name string, params map[string]any, body string) (w MarkdownWidget, summary string, ok bool)

// ParseFenceInfo splits a fence's info string into a widget name and its JSON parameters.
//
// A `{` in the info string is what makes a fence a WIDGET ATTEMPT rather than a language
// tag, and that is the whole discriminator: ```go stays a code block, and
// ```tool-call {"id":"read"} is a widget. It has to be unambiguous in both directions,
// because the failure is silent in the dangerous one — a widget attempt that fell
// through to the code renderer would look like a document that simply contained a code
// block, and nobody would notice until the embed stopped appearing.
//
// The parser is NOT the decider, though: it only says whether the author wrote a widget
// attempt. A fence with no brace is still offered to FenceResolver, because a widget with
// no parameters is legitimate and only the registry knows the name. What the parser
// guarantees is the error direction — a brace means the author is embedding, so a missing
// registration or a malformed object is reported rather than rendered as a code block.
func ParseFenceInfo(info string) (name string, params map[string]any, isWidget bool, err error) {
	info = strings.TrimSpace(info)
	if info == "" {
		return "", nil, false, nil
	}
	brace := strings.IndexByte(info, '{')
	if brace < 0 {
		// A plain language tag: ```go, ```json. Not an embed.
		return strings.Fields(info)[0], nil, false, nil
	}
	name = strings.TrimSpace(info[:brace])
	if name == "" {
		return "", nil, true, fmt.Errorf("a fence has parameters but no widget name: %q", info)
	}
	rest := strings.TrimSpace(info[brace:])
	// Trailing prose after the object is tolerated rather than rejected, because a fence
	// like ```tool {"id":"read"} (see below) is a thing a person writes.
	if end := strings.IndexByte(rest, '}'); end >= 0 {
		rest = rest[:end+1]
	} else {
		return name, nil, true, fmt.Errorf("the parameters for %q are missing a closing brace", name)
	}
	if err := json.Unmarshal([]byte(rest), &params); err != nil {
		return name, nil, true, fmt.Errorf("the parameters for %q are not valid JSON: %v", name, err)
	}
	return name, params, true, nil
}
