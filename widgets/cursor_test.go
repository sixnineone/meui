package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

// The framework's cursor contract: each widget reports the cursor that matches
// what the pointer is over, and nil (the default arrow) everywhere else. These
// tests are the checklist for "the right cursor at the right time".

func cursorName(c *platform.Cursor) string { return c.Name() }

func TestTextFieldCursorIsIBeamOverTheText(t *testing.T) {
	tf := NewTextField()
	tf.SetFrameRect(graphics.NewRect(0, 0, 200, 28))
	tf.SetText("hello")
	if got := cursorName(tf.CursorAt(50, 14)); got != "text" {
		t.Errorf("over the text: %q, want text", got)
	}
	if got := cursorName(tf.CursorAt(50, 1)); got != "" {
		t.Errorf("above the text inset: %q, want the default", got)
	}
	// The trailing icon slot is not text.
	tf.SetTrailingIcon("close")
	if got := cursorName(tf.CursorAt(196, 14)); got != "" {
		t.Errorf("over the trailing icon: %q, want the default", got)
	}
	if got := cursorName(tf.CursorAt(50, 14)); got != "text" {
		t.Errorf("text area shrank with the icon: %q, want text", got)
	}
	tf.SetEnabled(false)
	if got := cursorName(tf.CursorAt(50, 14)); got != "" {
		t.Errorf("disabled field: %q, want the default", got)
	}
}

func TestTextAreaCursorIsIBeamOverTheTextOnly(t *testing.T) {
	ta := NewTextArea()
	ta.SetFrameRect(graphics.NewRect(0, 0, 300, 120))
	ta.SetShowLineNumbers(true)
	ta.SetValue("one\ntwo\nthree")

	gutter := ta.gutterWidthFor(len(ta.lines()))
	if got := cursorName(ta.CursorAt(gutter+20, 40)); got != "text" {
		t.Errorf("over the text: %q, want text", got)
	}
	if got := cursorName(ta.CursorAt(gutter/2, 40)); got != "" {
		t.Errorf("over the line-number gutter: %q, want the default", got)
	}
	if got := cursorName(ta.CursorAt(298, 40)); got != "" {
		t.Errorf("over the scrollbar strip: %q, want the default", got)
	}
	if got := cursorName(ta.CursorAt(gutter+20, 2)); got != "" {
		t.Errorf("above the text inset: %q, want the default", got)
	}
}

func TestCodeEditorCursorIsIBeamOverTheTextOnly(t *testing.T) {
	ce := NewCodeEditor()
	ce.SetFrameRect(graphics.NewRect(0, 0, 400, 200))
	ce.SetText("func main() {\n\tprintln(\"hi\")\n}\n")
	ce.recalcGutterWidth()

	// The text area is sized at paint time (theme font metrics).
	ce.Paint(graphics.NewCanvas(graphics.NewTestDevice(400, 200)), graphics.NewRect(0, 0, 400, 200))
	if ce.gutterWidth <= 0 || ce.textAreaWidth <= 0 {
		t.Fatalf("editor geometry not initialised (gutter %v, text %v)", ce.gutterWidth, ce.textAreaWidth)
	}
	if got := cursorName(ce.CursorAt(ce.gutterWidth+30, 60)); got != "text" {
		t.Errorf("over the text: %q, want text", got)
	}
	if got := cursorName(ce.CursorAt(ce.gutterWidth/2, 60)); got != "" {
		t.Errorf("over the gutter: %q, want the default (breakpoint/bookmark lanes)", got)
	}
	if got := cursorName(ce.CursorAt(ce.gutterWidth+ce.textAreaWidth+10, 60)); got != "" {
		t.Errorf("past the text area: %q, want the default", got)
	}
}

func TestSplitViewCursorIsResizeOnTheHandle(t *testing.T) {
	sv := NewSplitView()
	left, right := meui.NewPanel(), meui.NewPanel()
	sv.SetPanels(left, right)
	sv.SetFrameRect(graphics.NewRect(0, 0, 400, 200))
	sv.SplitPosition = 0.5
	sv.ValidateLayout()

	if got := cursorName(sv.CursorAt(200, 100)); got != "resizeH" {
		t.Errorf("on the vertical splitter: %q, want resizeH", got)
	}
	if got := cursorName(sv.CursorAt(50, 100)); got != "" {
		t.Errorf("over the left panel: %q, want the default", got)
	}
	// The whole grab band belongs to the handle: the hit test must return the
	// splitter itself, or the press (drag) and the cursor go to the panel.
	// OnMouseDown accepts [handleX-4, handleX+HandleSize+4].
	handleX := float32(400)*sv.SplitPosition - sv.HandleSize/2
	for _, x := range []float32{handleX - 4, handleX - 1, handleX, handleX + 3, handleX + sv.HandleSize + 4} {
		if got := sv.PanelAt(x, 100); got != meui.Widget(sv) {
			t.Errorf("PanelAt at x=%v (handle at %v) hit %T, want the splitter", x, handleX, got)
		}
		if got := cursorName(sv.CursorAt(x, 100)); got != "resizeH" {
			t.Errorf("cursor at x=%v: %q, want resizeH", x, got)
		}
	}
	if got := sv.PanelAt(handleX-sv.HandleSize-8, 100); got == meui.Widget(sv) {
		t.Error("a press inside the left panel was given to the splitter")
	}

	sv.SetOrientation(1)
	sv.ValidateLayout()
	if got := cursorName(sv.CursorAt(200, 100)); got != "resizeV" {
		t.Errorf("on the horizontal splitter: %q, want resizeV", got)
	}
}

func TestTableAndDataTableResizeOnHeaderEdgesOnly(t *testing.T) {
	tbl := NewTable()
	tbl.SetFrameRect(graphics.NewRect(0, 0, 400, 200))
	tbl.SetColumns([]string{"one", "two", "three"})
	// Column widths are laid out while painting; three auto-sized columns in a
	// 400px frame put the edges at 0/130/270.
	tbl.Paint(graphics.NewCanvas(graphics.NewTestDevice(400, 200)), graphics.NewRect(0, 0, 400, 200))
	if got := cursorName(tbl.CursorAt(130, 5)); got != "resizeH" {
		t.Errorf("on a column edge: %q, want resizeH", got)
	}
	if got := cursorName(tbl.CursorAt(60, 5)); got != "" {
		t.Errorf("between edges: %q, want the default", got)
	}
	if got := cursorName(tbl.CursorAt(130, 100)); got != "" {
		t.Errorf("below the header: %q, want the default", got)
	}

	dt := NewDataTable()
	dt.SetFrameRect(graphics.NewRect(0, 0, 400, 200))
	dt.SetColumns([]DataColumn{{Label: "one", Width: 100}, {Label: "two", Width: 100}, {Label: "three", Width: 100}})
	dt.Paint(graphics.NewCanvas(graphics.NewTestDevice(400, 200)), graphics.NewRect(0, 0, 400, 200))
	// Explicit 100px widths put the edges at 0/100/200/300.
	if got := cursorName(dt.CursorAt(100, 5)); got != "resizeH" {
		t.Errorf("data table, on a column edge: %q, want resizeH", got)
	}
	if got := cursorName(dt.CursorAt(60, 5)); got != "" {
		t.Errorf("data table, between edges: %q, want the default", got)
	}
}

func TestLabelCursorIsPointingOverALink(t *testing.T) {
	l := NewLabel()
	l.SetRichText(true)
	l.SetText(`before <a href="https://example.com">link</a> after`)
	l.SetFrameRect(graphics.NewRect(0, 0, 300, 40))

	// Resolve the font the way linkAt does.
	font := l.LabelTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
	}
	if font == nil {
		font = graphics.SystemFont
	}
	// The rich-text layout centres the single line vertically.
	yOff := (l.FrameRect().Height - font.LineHeight()) / 2
	linkX := font.SimpleWidth("before ")
	linkEnd := linkX + font.SimpleWidth("link")
	linkY := yOff + font.LineHeight()/2
	if got := cursorName(l.CursorAt(linkX+4, linkY)); got != "pointing" {
		t.Errorf("over the link (x=%v y=%v): %q, want pointing", linkX+4, linkY, got)
	}
	if got := cursorName(l.CursorAt(linkEnd+10, linkY)); got != "" {
		t.Errorf("past the link: %q, want the default", got)
	}
	// A point that would be inside the link if the coordinates were converted
	// twice must not report a link (CursorAt is called with local coords).
	if got := cursorName(l.CursorAt(linkX+4, linkY+40)); got != "" {
		t.Errorf("below the line: %q, want the default", got)
	}
}

func TestMarkdownViewCursorOverTextAndLinks(t *testing.T) {
	mv := NewMarkdownView()
	mv.SetFrameRect(graphics.NewRect(0, 0, 500, 300))
	mv.SetText("plain text here\n\n[a link](https://example.com)\n")
	paintOnce(mv) // link rects are recorded while painting

	if got := cursorName(mv.CursorAt(30, 20)); got != "text" {
		t.Errorf("over text: %q, want text", got)
	}
	if len(mv.linkRects) == 0 {
		t.Fatal("no link rects after painting")
	}
	lr := mv.linkRects[0]
	if got := cursorName(mv.CursorAt(lr.rect.X+2, lr.rect.Y+2)); got != "pointing" {
		t.Errorf("over a link: %q, want pointing", got)
	}
}

// The MDI window's four corners follow the same diagonal convention as the
// window edges: NW-SE (left diagonal) for top-left/bottom-right.
func TestMdiWindowCornerCursors(t *testing.T) {
	mw := NewMdiWindow()
	mw.SetTitle("window")
	mw.SetFrameRect(graphics.NewRect(0, 0, 300, 200))
	w, h := mw.Width(), mw.Height()
	cases := []struct {
		x, y float32
		want string
		why  string
	}{
		{1, 1, "resizeLD", "top-left: NW-SE"},
		{w - 1, h - 1, "resizeLD", "bottom-right: NW-SE"},
		{w - 1, 1, "resizeRD", "top-right: NE-SW"},
		{1, h - 1, "resizeRD", "bottom-left: NE-SW"},
	}
	for _, c := range cases {
		if got := cursorName(mw.CursorAt(c.x, c.y)); got != c.want {
			t.Errorf("%s at (%v,%v): %q, want %q", c.why, c.x, c.y, got, c.want)
		}
	}
}

// A StackPanel's splitter band is part of the panel's own area, so the window
// hit test lands on the StackPanel: it is the widget that has to declare the
// resize cursor, or the drag has none (the owner's report — SplitView had it
// and StackPanel did not).
func TestStackPanelCursorIsResizeOnTheSplitter(t *testing.T) {
	sp := NewStackPanel("vertical")
	sp.AddItem(meui.NewPanel(), 100)
	sp.AddItem(meui.NewPanel(), 100)
	sp.SetFrameRect(graphics.NewRect(0, 0, 200, 206))
	sp.ValidateLayout()

	// Items are 100 tall, so the band is y=100..106 and the cursor must cover
	// exactly the band hitSplitterLocal and OnMouseDown accept.
	for _, y := range []float32{100, 103, 106} {
		if got := cursorName(sp.CursorAt(100, y)); got != "resizeV" {
			t.Errorf("on the splitter (y=%v): %q, want resizeV", y, got)
		}
	}
	for _, y := range []float32{50, 99, 107, 150} {
		if got := cursorName(sp.CursorAt(100, y)); got != "" {
			t.Errorf("over an item (y=%v): %q, want the default", y, got)
		}
	}

	horiz := NewStackPanel("horizontal")
	horiz.AddItem(meui.NewPanel(), 100)
	horiz.AddItem(meui.NewPanel(), 100)
	horiz.SetFrameRect(graphics.NewRect(0, 0, 206, 200))
	horiz.ValidateLayout()
	if got := cursorName(horiz.CursorAt(103, 100)); got != "resizeH" {
		t.Errorf("on a side-by-side splitter: %q, want resizeH", got)
	}
	if got := cursorName(horiz.CursorAt(50, 100)); got != "" {
		t.Errorf("over an item: %q, want the default", got)
	}
}
