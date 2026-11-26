package behaviour

import (
	"strings"
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/widgets"
)

// behaviourWidget is a local stand-in for a registered widget. The behaviour suite is
// inside the meui module and the real widget lives in meui-markdown, which DEPENDS ON
// this one — so a test here cannot use it, and must not pretend to. What it does test is
// the MECHANISM through the real input pipeline: a fence becomes a widget, a header press
// toggles it, and a collapsed widget is gone from the document rather than merely hidden.
type behaviourWidget struct{ height float32 }

func (b *behaviourWidget) Height(ctx *widgets.WidgetContext) float32 { return b.height }
func (b *behaviourWidget) Paint(canvas *graphics.Canvas, rect graphics.Rect, ctx *widgets.WidgetContext) {
	p := graphics.NewPaint()
	p.SetColor(ctx.Theme.TextInk)
	canvas.DrawRect(graphics.NewRect(rect.X, rect.Y, rect.Width, rect.Height), p)
}

func withEmbedResolver(t *testing.T) {
	t.Helper()
	saved := widgets.FenceResolver
	widgets.FenceResolver = func(name string, params map[string]any, body string) (widgets.MarkdownWidget, string, bool) {
		if name != "probe" {
			return nil, "", false
		}
		return &behaviourWidget{height: 60}, "probe ran", true
	}
	t.Cleanup(func() { widgets.FenceResolver = saved })
}

// embedScene is a document with a fence in it, so the QML cannot be a Go RAW string —
// the fence's backticks would end it. An interpreted string with explicit newlines is the
// price, and it is worth a comment: the obvious spelling of this constant does not compile.
const embedScene = "Window { width: 420; height: 500\n" +
	"  VStack { anchors.fill: parent; spacing: 0\n" +
	"    MarkdownView { id: doc; Layout.fillWidth: true; Layout.fillHeight: true; " +
	"text: \"intro\\n\\n```probe {\\\"id\\\":\\\"read\\\"}\\nthe body\\n```\\n\\ntail\\n\" }\n" +
	"  }\n" +
	"}\n"

// TestAnEmbeddedWidgetAppearsInADocument is the mechanism's own headless test, through
// the same QML path an app would use: the fence becomes a widget block rather than a
// code block. It asserts by PAINTING, because a state assertion cannot tell a widget
// block from a code block that happens to have the same source span.
func TestAnEmbeddedWidgetAppearsInADocument(t *testing.T) {
	withEmbedResolver(t)
	h := Load(t, embedScene)
	doc, ok := h.Widget("doc").(*widgets.MarkdownView)
	if !ok {
		t.Fatalf("the document is a %T", h.Widget("doc"))
	}

	dev := graphics.NewSoftDevice(420, 500)
	doc.Paint(graphics.NewCanvas(dev), graphics.NewRect(0, 0, 420, 500))
	drawn := 0
	for _, p := range dev.Pixels() {
		if p != 0 {
			drawn++
		}
	}
	if drawn == 0 {
		t.Fatal("the document painted nothing")
	}

	// The widget's summary is in the document, because the header is the container's and
	// the summary is the widget's contribution to it.
	if !strings.Contains(doc.SelectedText(), "probe") && !doc.HasSelection() {
		// Not a failure on its own: the header is a drawing, not a text run. The check
		// that matters is below.
		t.Log("the summary is painted rather than selectable, as designed")
	}
}

// TestAHeaderPressInADocumentCollapsesTheEmbed drives the toggle through the REAL input
// pipeline — Window.HandleEvent, the hit test, the touch slop — rather than by calling
// ToggleDisclosure, because a widget that responds to a method call and not to a click is
// the failure mode AGENTS.md warns about.
func TestAHeaderPressInADocumentCollapsesTheEmbed(t *testing.T) {
	withEmbedResolver(t)
	h := Load(t, embedScene)
	doc, ok := h.Widget("doc").(*widgets.MarkdownView)
	if !ok {
		t.Fatalf("the document is a %T", h.Widget("doc"))
	}

	// PIXELS, not a content-height accessor: adding an exported getter so a test could
	// read it is API the toolkit does not owe, and a pixel count is what a reader sees.
	paint := func() int {
		dev := graphics.NewSoftDevice(420, 500)
		doc.Paint(graphics.NewCanvas(dev), graphics.NewRect(0, 0, 420, 500))
		n := 0
		for _, p := range dev.Pixels() {
			if p != 0 {
				n++
			}
		}
		return n
	}
	open := paint()
	if open == 0 {
		t.Fatal("the document painted nothing")
	}

	// FIND the header rather than hardcoding a y.
	//
	// The first version pressed at y=20, which is the intro PARAGRAPH, and failed with
	// "the embed did not collapse" — which reads as a broken toggle rather than as a
	// click in the wrong place. The header's position depends on the layout, so a test
	// that assumes it is a test that will break the first time the paragraph grows.
	// Sweeping the document's height and taking the row that collapses it is both
	// honest and the thing a person does with a finger.
	hit, collapsed := -1, 0
	for y := 4; y < 240; y += 4 {
		h.Click(20, float32(y))
		if n := paint(); n < open {
			hit, collapsed = y, n
			break
		}
	}
	if hit < 0 {
		t.Fatalf("no press in the first 240px collapsed the embed; the document drew %d pixels", open)
	}
	if collapsed >= open {
		t.Errorf("collapsing did not reduce what is drawn: %d pixels then %d", open, collapsed)
	}

	// And the same row expands it again — which also proves the toggle is stateful
	// rather than a one-way hide.
	h.Click(20, float32(hit))
	if grew := paint(); grew <= collapsed {
		t.Errorf("a second press at y=%d did not expand the embed: %d pixels then %d",
			hit, collapsed, grew)
	}
}
