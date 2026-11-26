package widgets

import (
	"strings"
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
)

const editorDoc = "Some **bold** text\n\n- item one\n- item two\n"

func markdownEditorView(t *testing.T) *MarkdownEditor {
	t.Helper()
	m := NewMarkdownEditor()
	m.SetFrameRect(graphics.NewRect(0, 0, 600, 400))
	m.SetText(editorDoc)
	m.ValidateLayout()
	return m
}

func TestMarkdownEditorSetsBothPanes(t *testing.T) {
	m := markdownEditorView(t)
	if got := m.Text(); got != editorDoc {
		t.Fatalf("source text = %q", got)
	}
	if got := m.Preview().Text(); got != editorDoc {
		t.Fatalf("preview text = %q", got)
	}
	// Both panes show content: the source keeps one line per line, the preview
	// has laid out blocks.
	if m.Source().LineCount() < 2 {
		t.Fatalf("source pane has %d lines", m.Source().LineCount())
	}
	m.Preview().ensureLayout()
	if len(m.Preview().laid) < 3 {
		t.Fatalf("preview parsed %d blocks", len(m.Preview().laid))
	}
}

func TestMarkdownEditorSourceEditUpdatesThePreview(t *testing.T) {
	m := markdownEditorView(t)
	var seen []string
	m.OnTextChanged = func(text string) { seen = append(seen, text) }

	m.Source().SetText("# New heading\n\nbody text\n")
	if got := m.Preview().Text(); got != "# New heading\n\nbody text\n" {
		t.Fatalf("preview did not follow the source: %q", got)
	}
	if len(seen) != 1 || seen[0] != "# New heading\n\nbody text\n" {
		t.Fatalf("OnTextChanged saw %v", seen)
	}

	// SetText on the editor itself must not fire the callback.
	seen = nil
	m.SetText(editorDoc)
	if len(seen) != 0 {
		t.Fatalf("SetText fired OnTextChanged: %v", seen)
	}
	if m.Text() != editorDoc || m.Preview().Text() != editorDoc {
		t.Fatalf("SetText did not reach both panes")
	}
}

func TestMarkdownEditorClipboardDiffersPerPane(t *testing.T) {
	m := markdownEditorView(t)
	platform.GlobalClipboard.SetText("")

	// The preview copies rendered plain text.
	m.Preview().SelectAll()
	m.Preview().Copy()
	plain := platform.GlobalClipboard.GetText()
	if !strings.Contains(plain, "Some bold text") {
		t.Fatalf("preview copy = %q, want the rendered text", plain)
	}
	if strings.Contains(plain, "**") {
		t.Fatalf("preview copy = %q, markdown syntax leaked", plain)
	}

	// The source copies its own text, which is the markdown source.
	platform.GlobalClipboard.SetText("")
	m.Source().SelectAll()
	m.Source().Copy()
	md := platform.GlobalClipboard.GetText()
	if md != editorDoc {
		t.Fatalf("source copy = %q, want the markdown source", md)
	}
	platform.GlobalClipboard.SetText("")
}

func TestMarkdownEditorModesShowOnePane(t *testing.T) {
	m := markdownEditorView(t)

	m.SetMode(MarkdownSourceOnly)
	m.ValidateLayout()
	if !m.Source().Visible() || m.Preview().Visible() {
		t.Fatalf("source-only: source visible=%v preview visible=%v", m.Source().Visible(), m.Preview().Visible())
	}
	if w := m.Source().FrameRect().Width; w < 590 {
		t.Fatalf("source-only gave the source %v px of 600", w)
	}

	m.SetMode(MarkdownPreviewOnly)
	m.ValidateLayout()
	if m.Source().Visible() || !m.Preview().Visible() {
		t.Fatalf("preview-only: source visible=%v preview visible=%v", m.Source().Visible(), m.Preview().Visible())
	}
	if w := m.Preview().FrameRect().Width; w < 590 {
		t.Fatalf("preview-only gave the preview %v px of 600", w)
	}

	m.SetMode(MarkdownSplit)
	m.ValidateLayout()
	if !m.Source().Visible() || !m.Preview().Visible() {
		t.Fatal("split mode hid a pane")
	}
	if w := m.Source().FrameRect().Width; w < 200 || w > 400 {
		t.Fatalf("split mode gave the source %v px of 600", w)
	}

	// The ratio is remembered across a mode round trip.
	m.SetSplitRatio(0.7)
	m.SetMode(MarkdownPreviewOnly)
	m.SetMode(MarkdownSplit)
	m.ValidateLayout()
	if m.SplitRatio() != 0.7 {
		t.Fatalf("split ratio = %v, want 0.7", m.SplitRatio())
	}
	if w := m.Source().FrameRect().Width; w < 380 {
		t.Fatalf("after the round trip the source is %v px, want ~70%%", w)
	}
}

func TestMarkdownEditorReadOnlyReachesTheSourcePane(t *testing.T) {
	m := markdownEditorView(t)
	if m.IsReadOnly() {
		t.Fatal("a new editor is read-only")
	}
	m.SetReadOnly(true)
	if !m.IsReadOnly() || !m.Source().ReadOnly {
		t.Fatal("SetReadOnly did not reach the source pane")
	}
	// The preview keeps its selection behaviour in read-only mode.
	m.Preview().SelectAll()
	if !m.Preview().HasSelection() {
		t.Fatal("the preview cannot be selected in read-only mode")
	}
}

// The editor is a container: the hit test must reach the panes, or every press
// dies on the editor (BaseContainer provides DefaultPanelAt but not PanelAt, so
// an unoverridden PanelAt returns the widget itself and swallows input).
func TestMarkdownEditorHitTestReachesBothPanes(t *testing.T) {
	m := markdownEditorView(t)

	left := m.PanelAt(100, 200)
	if left != meui.Widget(m.Source()) {
		t.Fatalf("a press in the left pane hit %T, want the source pane", left)
	}
	right := m.PanelAt(500, 200)
	if right != meui.Widget(m.Preview()) {
		t.Fatalf("a press in the right pane hit %T, want the preview pane", right)
	}

	// The hit widgets must be the ones that take focus and events, or the
	// editor looks frozen: no caret in the source, no selection in the preview.
	for name, w := range map[string]meui.Widget{"source": left, "preview": right} {
		if _, ok := w.(interface{ OnFocus() }); !ok {
			t.Errorf("the %s pane cannot take focus (%T)", name, w)
		}
		if _, ok := w.(interface {
			OnMouseDown(float32, float32, int) bool
		}); !ok {
			t.Errorf("the %s pane cannot take a press (%T)", name, w)
		}
	}

	// And a press routed to them must actually be handled.
	lx, ly := m.Source().PointFromRoot(100, 200).X, m.Source().PointFromRoot(100, 200).Y
	if !m.Source().OnMouseDown(lx, ly, 0) {
		t.Error("the source pane refused a press inside it")
	}
	rx, ry := m.Preview().PointFromRoot(500, 200).X, m.Preview().PointFromRoot(500, 200).Y
	if !m.Preview().OnMouseDown(rx, ry, 0) {
		t.Error("the preview pane refused a press inside it")
	}
}

// A press on the splitter itself belongs to the splitter (it starts a drag).
func TestMarkdownEditorSplitterOwnsItsOwnPresses(t *testing.T) {
	m := markdownEditorView(t)
	m.SetSplitRatio(0.5)
	m.ValidateLayout()
	split := m.Split()
	if split == nil {
		t.Fatal("no splitter")
	}
	// The handle sits at the split position; a press there must not be
	// attributed to a pane.
	x := float32(600) * m.SplitRatio()
	if got := m.PanelAt(x, 200); got == meui.Widget(m.Source()) || got == meui.Widget(m.Preview()) {
		t.Fatalf("a press on the splitter hit %T, want the splitter", got)
	}
}

// The editor sits inside a layout, so its frame has a non-zero origin. A press
// is routed with Window.PanelAt (parent-local coords) and delivered with
// PointFromRoot, which subtracts every ancestor frame: if the splitter's frame
// carries the editor's own origin, that origin is subtracted twice and every
// press lands above the pointer (clicking line 23 put the caret near line 18).
func TestMarkdownEditorClickMapsToTheRightLine(t *testing.T) {
	m := NewMarkdownEditor()
	m.SetText(strings.Repeat("line of text here\n", 40))
	// Parent-relative frame with a non-zero origin, as a VStack child has.
	m.SetFrameRect(graphics.NewRect(12, 185, 800, 500))
	m.ValidateLayout()

	code := m.Source()
	want := 10
	localY := code.visibleLineY(want) + code.lineHeight/2
	rootX := float32(12 + 100)
	rootY := float32(185) + localY

	hit := m.PanelAt(rootX, rootY)
	if hit != meui.Widget(code) {
		t.Fatalf("press at root (%v,%v) hit %T, want the source pane", rootX, rootY, hit)
	}
	pfr, ok := hit.(interface {
		PointFromRoot(float32, float32) graphics.Point
	})
	if !ok {
		t.Fatalf("%T cannot convert root coordinates", hit)
	}
	pt := pfr.PointFromRoot(rootX, rootY)
	if pt.Y < localY-1 || pt.Y > localY+1 {
		t.Fatalf("root y %v became local y %v, want %v (the editor origin was double-counted)", rootY, pt.Y, localY)
	}
	code.OnMouseDown(pt.X, pt.Y, 0)
	if code.CursorLine != want {
		t.Fatalf("a press on line %d put the caret on line %d", want, code.CursorLine)
	}
}

// The same chain drives the preview's hit test: a point taken from the layout
// must round-trip to the offset it was taken from.
func TestMarkdownEditorPreviewClickMapsToTheRightOffset(t *testing.T) {
	m := NewMarkdownEditor()
	m.SetText(strings.Repeat("paragraph text here\n\n", 12))
	m.SetFrameRect(graphics.NewRect(12, 185, 800, 500))
	m.ValidateLayout()

	preview := m.Preview()
	preview.ensureLayout()
	if len(preview.laid) < 6 {
		t.Fatalf("preview laid out %d blocks", len(preview.laid))
	}
	// Pick a run in a later block and the offset of one of its runes.
	run := &preview.laid[5].lines[0].runs[0]
	target := run.runeSrc[len(run.runeSrc)/2]
	caret := preview.CaretRect(target)
	if caret.Width == 0 {
		t.Fatal("no caret rect for the target offset")
	}

	editor := m.FrameRect()
	split := m.Split().FrameRect()
	pane := preview.FrameRect()
	rootX := editor.X + split.X + pane.X + caret.X + 0.5
	rootY := editor.Y + split.Y + pane.Y + caret.Y + caret.Height/2

	hit := m.PanelAt(rootX, rootY)
	if hit != meui.Widget(preview) {
		t.Fatalf("press at root (%v,%v) hit %T, want the preview pane", rootX, rootY, hit)
	}
	pfr, ok := hit.(interface {
		PointFromRoot(float32, float32) graphics.Point
	})
	if !ok {
		t.Fatalf("%T cannot convert root coordinates", hit)
	}
	pt := pfr.PointFromRoot(rootX, rootY)
	if got := preview.SourceOffsetAt(pt.X, pt.Y); got != target {
		t.Fatalf("press at root (%v,%v) -> local (%v,%v) gave offset %d, want %d", rootX, rootY, pt.X, pt.Y, got, target)
	}
}
