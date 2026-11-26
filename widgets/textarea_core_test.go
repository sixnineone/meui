package widgets

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/sixnineone/meui/graphics"
)

// TextArea's wrapping and pointer hit-test now share the core in
// textlayout.go; these tests pin the behaviour they must keep.

func textAreaView(t *testing.T, value string, wrap bool) *TextArea {
	t.Helper()
	ta := NewTextArea()
	ta.SetFrameRect(graphics.NewRect(0, 0, 240, 200))
	ta.SetWordWrap(wrap)
	ta.SetValue(value)
	return ta
}

func TestTextAreaWrapFitsTheWidthAndKeepsTheText(t *testing.T) {
	// Long enough to wrap at the 240px frame with the test font (the font is
	// small, so a single sentence is not enough).
	const line = "the quick brown fox jumps over the lazy dog and keeps running " +
		"while the second sentence fills another row and a third one begins"
	ta := textAreaView(t, line, true)
	disp := ta.lines()
	if len(disp) < 2 {
		t.Fatalf("a %d-character line did not wrap (%d display line(s))", len(line), len(disp))
	}
	width := ta.availableTextWidth()
	font := ta.font()
	for i, dl := range disp {
		if w := font.SimpleWidth(dl); w > width+0.01 {
			t.Errorf("display line %d is %v wide, over the %v limit (%q)", i, w, width, dl)
		}
	}
	if got := strings.Join(disp, ""); got != line {
		t.Fatalf("wrapped text = %q, want %q", got, line)
	}
	// Every display line still maps back to its content line.
	for i := range disp {
		if ta.dispToContent[i] != 0 {
			t.Fatalf("display line %d maps to content line %d, want 0", i, ta.dispToContent[i])
		}
	}
}

func TestTextAreaNoWrapKeepsOneDisplayLinePerLine(t *testing.T) {
	const value = "short\n" + "a very long line that would wrap if wrapping were on\n" + "last"
	ta := textAreaView(t, value, false)
	disp := ta.lines()
	if len(disp) != 3 {
		t.Fatalf("got %d display lines, want 3", len(disp))
	}
	if disp[1] != "a very long line that would wrap if wrapping were on" {
		t.Fatalf("display line 1 = %q", disp[1])
	}
}

func TestTextAreaCharIndexRoundTripsThroughTheCore(t *testing.T) {
	const line = "column ALIGNMENT text"
	ta := textAreaView(t, line, false)
	startX := ta.textStartX()
	font := ta.font()
	for col := 0; col <= len([]rune(line)); col++ {
		x := caretXAtColumn(line, font, col)
		got := ta.charIndexAtX(0, startX+x+0.5)
		if got != col {
			t.Fatalf("column %d is at x=%v but charIndexAtX gives %d", col, x, got)
		}
	}
	if got := ta.charIndexAtX(0, -100); got != 0 {
		t.Fatalf("a click left of the line gave %d, want 0", got)
	}
	if got := ta.charIndexAtX(0, 100000); got != len([]rune(line)) {
		t.Fatalf("a click right of the line gave %d, want %d", got, len([]rune(line)))
	}
}

func TestTextAreaClickPlacesTheCursor(t *testing.T) {
	const line = "click position HERE"
	ta := textAreaView(t, line, false)
	want := 8 // inside "position"
	x := ta.textStartX() + caretXAtColumn(line, ta.font(), want) + 0.5
	y := ta.lineHeight() / 2
	if !ta.OnMouseDown(x, y, 0) {
		t.Fatal("press on the text was not handled")
	}
	if ta.CursorLine != 0 || ta.CursorCol != want {
		t.Fatalf("cursor at (%d,%d), want (0,%d)", ta.CursorLine, ta.CursorCol, want)
	}
}

func TestTextAreaWrappedClickLandsOnTheRightContentLine(t *testing.T) {
	first := "first line that is long enough to wrap onto a second display row"
	value := first + "\nsecond"
	ta := textAreaView(t, value, true)
	disp := ta.lines()
	if len(disp) < 2 {
		t.Fatalf("expected the first line to wrap, got %d display line(s)", len(disp))
	}
	// Click on the last display row of the first content line: the cursor must
	// be inside that content line, not on "second".
	last := len(disp) - 2
	runes := len([]rune(disp[last]))
	x := ta.textStartX() + caretXAtColumn(disp[last], ta.font(), runes/2) + 0.5
	y := float32(last)*ta.lineHeight() + ta.lineHeight()/2
	ta.OnMouseDown(x, y, 0)
	if ta.CursorLine != 0 {
		t.Fatalf("click on wrapped row %d landed on content line %d, want 0", last, ta.CursorLine)
	}
	if ta.CursorCol == 0 || ta.CursorCol >= len(first) {
		t.Fatalf("cursor column %d is not inside the first content line", ta.CursorCol)
	}
}

// A click on multi-byte text must yield a byte column on a rune boundary
// (TextArea slices its content by bytes).
func TestTextAreaClickOnMultiByteTextLandsOnARuneBoundary(t *testing.T) {
	const line = "café ZÜRICH"
	ta := textAreaView(t, line, false)
	startX := ta.textStartX()
	font := ta.font()
	for i := range line {
		if i > 0 && !utf8.RuneStart(line[i]) {
			continue
		}
		x := startX + caretXAtColumn(line, font, i)
		col := ta.charIndexAtX(0, x+0.5)
		if col != i {
			t.Fatalf("byte column %d is at x=%v but the click gives %d", i, x, col)
		}
		if col > 0 && col < len(line) && !utf8.RuneStart(line[col]) {
			t.Fatalf("click gave column %d, inside a multi-byte rune", col)
		}
	}
}
