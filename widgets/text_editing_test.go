package widgets

// text_editing_test.go — the editing gestures a text field and a text area have to share with every
// other framework's: word motion, collapse direction, shift-click, word-wise drag, page keys, a
// sticky goal column, UTF-8-safe deletion, single-line paste, and an undo you can actually use.
//
// Each of these was absent or wrong, and the two widgets had two different answers for the same
// gesture.

import (
	"strings"
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/mobilegles"
)

func newTestField(text string) *TextField {
	tf := NewTextField()
	tf.SetText(text)
	tf.CursorPos = len(tf.Text)
	tf.SelectStart, tf.SelectEnd = tf.CursorPos, tf.CursorPos
	tf.selectionAnchor = tf.CursorPos
	tf.clickCount = 1
	tf.Focused = true
	tf.SetFrameRect(graphics.NewRect(0, 0, 320, 30))
	return tf
}

func newTestArea(text string) *TextArea {
	ta := NewTextArea()
	ta.SetText(text)
	ta.Focused = true
	ta.SetFrameRect(graphics.NewRect(0, 0, 320, 120))
	ta.CursorLine, ta.CursorCol = ta.offsetToCursor(len(text))
	return ta
}

// ── word motion ──────────────────────────────────────────────────────────

// A word boundary is a RUNE CLASS boundary (space / word / punctuation), the same one a double-click
// uses. Movement used to treat every non-space character as one word, so "foo.bar" moved as a single
// word while double-clicking it selected only "foo".
func TestWordMotionStopsAtPunctuation(t *testing.T) {
	tf := newTestField("foo.bar baz")
	at := func() int { return tf.CursorPos }

	tf.OnKeyDown(platform.KeyLeft, platform.ModCtrl)
	if at() != 8 {
		t.Errorf("Ctrl+Left landed at %d, want 8 (the start of \"baz\")", at())
	}
	tf.OnKeyDown(platform.KeyLeft, platform.ModCtrl)
	if at() != 4 {
		t.Errorf("Ctrl+Left landed at %d, want 4 (the start of \"bar\")", at())
	}
	tf.OnKeyDown(platform.KeyLeft, platform.ModCtrl)
	if at() != 3 {
		t.Errorf("Ctrl+Left landed at %d, want 3 (the dot is its own run)", at())
	}

	tf.CursorPos = 0
	tf.OnKeyDown(platform.KeyRight, platform.ModCtrl)
	if at() != 3 {
		t.Errorf("Ctrl+Right landed at %d, want 3 (the end of \"foo\")", at())
	}
	tf.OnKeyDown(platform.KeyRight, platform.ModCtrl)
	if at() != 4 {
		t.Errorf("Ctrl+Right landed at %d, want 4 (past the dot)", at())
	}
}

// Ctrl+Left/Right without Shift did NOTHING in a field: the word move existed only on the
// shift-extends-selection path.
func TestCtrlArrowMovesWithoutShift(t *testing.T) {
	tf := newTestField("alpha beta")
	tf.OnKeyDown(platform.KeyLeft, platform.ModCtrl)
	if tf.CursorPos != 6 {
		t.Errorf("Ctrl+Left moved to %d, want 6", tf.CursorPos)
	}
	tf.OnKeyDown(platform.KeyRight, platform.ModCtrl)
	if tf.CursorPos != len(tf.Text) {
		t.Errorf("Ctrl+Right moved to %d, want %d", tf.CursorPos, len(tf.Text))
	}
}

// ── collapse direction ───────────────────────────────────────────────────

// A plain arrow collapses a selection towards the side it moves to: LEFT to the start, RIGHT to the
// end. The two were swapped, so Left jumped the caret past the whole selection.
func TestArrowCollapsesTowardsItsDirection(t *testing.T) {
	tf := newTestField("abcdef")
	tf.SetSelection(1, 4)
	tf.OnKeyDown(platform.KeyLeft, platform.ModNone)
	if tf.hasSelection() || tf.CursorPos != 1 {
		t.Errorf("Left collapsed to %d (selection %v), want the START (1)", tf.CursorPos, tf.hasSelection())
	}

	tf.SetSelection(1, 4)
	tf.OnKeyDown(platform.KeyRight, platform.ModNone)
	if tf.hasSelection() || tf.CursorPos != 4 {
		t.Errorf("Right collapsed to %d (selection %v), want the END (4)", tf.CursorPos, tf.hasSelection())
	}

	ta := newTestArea("abcdefgh")
	ta.SetSelection(2, 5)
	ta.OnKeyDown(platform.KeyLeft, platform.ModNone)
	if ta.hasSelection() || ta.cursorToOffset(ta.CursorLine, ta.CursorCol) != 2 {
		t.Errorf("TextArea Left collapsed to offset %d, want the START (2)",
			ta.cursorToOffset(ta.CursorLine, ta.CursorCol))
	}
	ta.SetSelection(2, 5)
	ta.OnKeyDown(platform.KeyRight, platform.ModNone)
	if ta.hasSelection() || ta.cursorToOffset(ta.CursorLine, ta.CursorCol) != 5 {
		t.Errorf("TextArea Right collapsed to offset %d, want the END (5)",
			ta.cursorToOffset(ta.CursorLine, ta.CursorCol))
	}
}

// ── mouse gestures ───────────────────────────────────────────────────────

// Shift+click extends the selection instead of starting a new one. The widgets could not see the
// modifier at all before, so every shift-click collapsed the selection.
func TestShiftClickExtendsTheSelection(t *testing.T) {
	tf := newTestField("hello world")
	tf.applyClick(5, false)
	tf.applyClick(11, true)
	if got := tf.GetSelectedText(); got != " world" {
		t.Errorf("shift-click selected %q, want %q", got, " world")
	}

	ta := newTestArea("hello\nworld")
	ta.applyClick(0, 0, false)
	ta.applyClick(1, 3, true)
	if got := ta.GetSelectedText(); got != "hello\nwor" {
		t.Errorf("TextArea shift-click selected %q, want %q", got, "hello\nwor")
	}
}

// A double-click that then drags selects WHOLE WORDS as the drag moves — the rest of the gesture in
// every editor. It used to end at the word and drag a character range.
func TestDoubleClickDragSelectsWholeWords(t *testing.T) {
	tf := newTestField("alpha beta gamma")
	tf.clickCount = 2
	tf.applyClick(1, false) // inside "alpha"
	if got := tf.GetSelectedText(); got != "alpha" {
		t.Fatalf("double-click selected %q, want %q", got, "alpha")
	}
	tf.extendWordDragTo(13) // inside "gamma"
	if got := tf.GetSelectedText(); got != "alpha beta gamma" {
		t.Errorf("the word drag selected %q, want the whole phrase", got)
	}
	tf.extendWordDragTo(8) // back to "beta"
	if got := tf.GetSelectedText(); got != "alpha beta" {
		t.Errorf("the word drag shrank to %q, want %q", got, "alpha beta")
	}

	ta := newTestArea("alpha beta gamma")
	ta.clickCount = 2
	ta.applyClick(0, 1, false)
	if got := ta.GetSelectedText(); got != "alpha" {
		t.Fatalf("TextArea double-click selected %q, want %q", got, "alpha")
	}
	ta.extendWordDragTo(len("alpha beta gam"))
	if got := ta.GetSelectedText(); got != "alpha beta gamma" {
		t.Errorf("TextArea word drag selected %q, want the whole phrase", got)
	}
}

// ── clipboard ────────────────────────────────────────────────────────────

// Cut with no selection must do NOTHING. It used to select all and cut, so Ctrl+X on an empty field
// wiped the value — a data-loss trap in a form.
func TestCutWithNoSelectionIsANoOp(t *testing.T) {
	platform.GlobalClipboard.SetText("")
	tf := newTestField("keep me")
	tf.Cut()
	if tf.Text != "keep me" {
		t.Errorf("Cut with no selection changed the field to %q", tf.Text)
	}

	ta := newTestArea("keep\nme")
	ta.Cut()
	if ta.Value != "keep\nme" {
		t.Errorf("TextArea Cut with no selection changed the value to %q", ta.Value)
	}
	platform.GlobalClipboard.SetText("")
}

// A single-line field cannot hold a line break: a paste turns newlines and tabs into spaces and
// respects MaxLength, exactly as a browser's <input> does.
func TestPasteIntoAFieldIsSingleLine(t *testing.T) {
	tf := newTestField("ab")
	tf.MaxLength = 5
	platform.GlobalClipboard.SetText("1\n2\t3")
	tf.Paste()
	platform.GlobalClipboard.SetText("")

	if strings.ContainsAny(tf.Text, "\n\r\t") {
		t.Errorf("the paste left a line break or tab in a single-line field: %q", tf.Text)
	}
	if tf.Text != "ab1 2" {
		t.Errorf("the paste produced %q, want %q (sanitised and clamped to MaxLength)", tf.Text, "ab1 2")
	}
}

// The classic bindings, which terminals and long-time users still reach for.
func TestClassicClipboardKeys(t *testing.T) {
	platform.GlobalClipboard.SetText("")
	tf := newTestField("cut me")
	tf.SetSelection(0, 3)
	if !tf.OnKeyDown(platform.KeyDelete, platform.ModShift) {
		t.Fatal("Shift+Delete was not handled")
	}
	if tf.Text != " me" {
		t.Errorf("Shift+Delete left %q, want %q", tf.Text, " me")
	}
	tf.OnKeyDown(platform.KeyInsert, platform.ModShift) // Shift+Insert pastes
	if tf.Text != "cut me" {
		t.Errorf("Shift+Insert left %q, want the pasted text back", tf.Text)
	}
	platform.GlobalClipboard.SetText("")
}

// ── undo grouping ────────────────────────────────────────────────────────

// Typing is ONE undo step, not one per character. Undo per keystroke is technically correct and
// practically useless: undoing a word took five presses.
func TestTypingIsOneUndoStep(t *testing.T) {
	tf := newTestField("")
	for _, r := range "abc" {
		tf.OnCharacter(r)
	}
	if !tf.Undo() {
		t.Fatal("Undo did nothing")
	}
	if tf.Text != "" {
		t.Errorf("one Undo left %q, want the empty text the burst started from", tf.Text)
	}

	ta := newTestArea("")
	for _, r := range "abc" {
		ta.OnCharacter(r)
	}
	ta.undo()
	if ta.Value != "" {
		t.Errorf("one TextArea undo left %q, want the empty text the burst started from", ta.Value)
	}
}

// At MaxLength a keystroke that REPLACES a selection must still be accepted: counting the existing
// text alone refused it, so a full field could not be corrected by selecting and typing.
func TestMaxLengthAllowsReplacingASelection(t *testing.T) {
	tf := newTestField("12345")
	tf.MaxLength = 5
	tf.SetSelection(0, 2)
	tf.OnCharacter('9')
	if tf.Text != "9345" {
		t.Errorf("typing over a selection at MaxLength produced %q, want %q", tf.Text, "9345")
	}
}

// A caret move ENDS the typing group: undo after moving deletes only what was typed since.
func TestCaretMoveBreaksTheUndoGroup(t *testing.T) {
	tf := newTestField("")
	tf.OnCharacter('a')
	tf.OnCharacter('b')
	tf.OnKeyDown(platform.KeyLeft, platform.ModNone)
	tf.OnCharacter('c')
	if tf.Text != "acb" {
		t.Fatalf("setup produced %q, want %q", tf.Text, "acb")
	}
	tf.Undo()
	if tf.Text != "ab" {
		t.Errorf("one Undo left %q, want %q (only the post-move keystroke)", tf.Text, "ab")
	}
}

// A paste is its own undo step; it must not merge into the typing that came before it.
func TestPasteIsItsOwnUndoStep(t *testing.T) {
	ta := newTestArea("")
	ta.OnCharacter('a')
	ta.OnCharacter('b')
	platform.GlobalClipboard.SetText("XY")
	ta.Paste()
	platform.GlobalClipboard.SetText("")
	if ta.Value != "abXY" {
		t.Fatalf("setup produced %q, want %q", ta.Value, "abXY")
	}
	ta.undo()
	if ta.Value != "ab" {
		t.Errorf("one undo after a paste left %q, want %q", ta.Value, "ab")
	}
}

// A read-only field or area is still READABLE: the caret moves, the arrows select, Ctrl+A/C works.
// Read-only used to refuse the arrow keys as if they were edits.
func TestReadOnlyStillNavigatesAndCopies(t *testing.T) {
	platform.GlobalClipboard.SetText("")
	tf := newTestField("abc")
	tf.SetReadOnly(true)
	tf.OnKeyDown(platform.KeyLeft, platform.ModNone)
	if tf.CursorPos != 2 {
		t.Errorf("a read-only field ignored Left (cursor %d, want 2)", tf.CursorPos)
	}
	tf.OnKeyDown(platform.KeyA, platform.ModCtrl)
	tf.OnKeyDown(platform.KeyC, platform.ModCtrl)
	if got := platform.GlobalClipboard.GetText(); got != "abc" {
		t.Errorf("copy from a read-only field put %q on the clipboard", got)
	}
	tf.OnKeyDown(platform.KeyBackspace, platform.ModNone)
	if tf.Text != "abc" {
		t.Errorf("a read-only field accepted a Backspace: %q", tf.Text)
	}

	ta := newTestArea("one\ntwo")
	ta.SetReadOnly(true)
	ta.OnKeyDown(platform.KeyDown, platform.ModNone)
	if ta.CursorLine != 1 {
		t.Errorf("a read-only area ignored Down (line %d, want 1)", ta.CursorLine)
	}
	ta.OnKeyDown(platform.KeyBackspace, platform.ModNone)
	if ta.Value != "one\ntwo" {
		t.Errorf("a read-only area accepted a Backspace: %q", ta.Value)
	}
	platform.GlobalClipboard.SetText("")
}

// ── the keycodes the PUMP actually produces ─────────────────────────────

// A printable key must insert NOTHING on its key-down; the rune event is the only thing that types.
// The text field used to synthesize characters from key codes with ASCII values against HID keycodes:
// 32 is the "3" key, so pressing 3 inserted a space first, and 48-57 (] \ ; ' ` , . / CapsLock F1)
// were typed as 0-9. This test drives the PUMP'S constants, not the platform aliases, so the two
// tables cannot drift apart again.
func TestTypingAKeyInsertsExactlyOneCharacter(t *testing.T) {
	cases := []struct {
		name string
		key  mobilegles.Keycode
		ch   rune
	}{
		{"three", mobilegles.Key3, '3'},
		{"zero", mobilegles.Key0, '0'},
		{"comma", mobilegles.KeyComma, ','},
		{"period", mobilegles.KeyPeriod, '.'},
		{"slash", mobilegles.KeySlash, '/'},
		{"semicolon", mobilegles.KeySemicolon, ';'},
		{"right bracket", mobilegles.KeyCloseBracket, ']'},
		{"backslash", mobilegles.KeyBackslash, '\\'},
		{"letter f", mobilegles.KeyF, 'f'},
		{"space", mobilegles.KeySpace, ' '},
	}
	for _, c := range cases {
		tf := newTestField("")
		tf.OnKeyDown(c.key, platform.ModNone)
		if tf.Text != "" {
			t.Errorf("%s: the key-down alone inserted %q — text must come from the rune event", c.name, tf.Text)
		}
		tf.OnCharacter(c.ch)
		if tf.Text != string(c.ch) {
			t.Errorf("%s: the rune produced %q, want %q", c.name, tf.Text, string(c.ch))
		}
	}

	// And a shifted digit: the key-down must not insert, the rune carries the shifted character.
	tf := newTestField("")
	tf.OnKeyDown(mobilegles.Key3, platform.ModShift)
	if tf.Text != "" {
		t.Errorf("shift+3 inserted %q on the key-down", tf.Text)
	}
	tf.OnCharacter('!')
	if tf.Text != "!" {
		t.Errorf("shift+3 produced %q, want %q", tf.Text, "!")
	}
}

// Delete, Home, End and the page keys have to ACT. They were compared against SDL scancode numbers
// (Delete=76, Home=74, End=77, PageUp=75, PageDown=78) while the pump delivers USB HID usage IDs
// (212, 199, 213, 211, 214), so nothing matched and the keys did nothing at all.
func TestEditingKeysActOnThePumpKeycodes(t *testing.T) {
	tf := newTestField("abcd")
	tf.CursorPos, tf.SelectStart, tf.SelectEnd = 1, 1, 1
	tf.OnKeyDown(mobilegles.KeyDelete, platform.ModNone)
	if tf.Text != "acd" {
		t.Errorf("Delete produced %q, want %q", tf.Text, "acd")
	}
	tf.OnKeyDown(mobilegles.KeyEnd, platform.ModNone)
	if tf.CursorPos != len(tf.Text) {
		t.Errorf("End put the caret at %d, want %d", tf.CursorPos, len(tf.Text))
	}
	tf.OnKeyDown(mobilegles.KeyHome, platform.ModNone)
	if tf.CursorPos != 0 {
		t.Errorf("Home put the caret at %d, want 0", tf.CursorPos)
	}

	ta := newTestArea("one two\nthree four")
	ta.CursorLine, ta.CursorCol = 0, 0
	ta.OnKeyDown(mobilegles.KeyPageDown, platform.ModNone)
	if ta.CursorLine == 0 {
		t.Error("PageDown left the caret on the first line")
	}
	ta.OnKeyDown(mobilegles.KeyPageUp, platform.ModNone)
	if ta.CursorLine != 0 {
		t.Errorf("PageUp left the caret on line %d, want 0", ta.CursorLine)
	}
	ta.OnKeyDown(mobilegles.KeyEnd, platform.ModNone)
	if ta.CursorCol != len("one two") {
		t.Errorf("End put the caret at column %d, want %d", ta.CursorCol, len("one two"))
	}
}

// ── IME composition ─────────────────────────────────────────────────────

// The IME's preedit is SHOWN (and underlined) while composing, REPLACED as the composition changes,
// and never committed until the characters arrive. A commit removes the preedit and inserts the
// committed text as one undoable edit; the preedit itself leaves no undo step behind.
func TestCompositionShowsReplacesAndCommitsTextField(t *testing.T) {
	tf := newTestField("ab")
	tf.CursorPos, tf.SelectStart, tf.SelectEnd = 2, 2, 2
	tf.selectionAnchor = 2

	tf.OnComposition("に", 0, 0)
	if tf.Text != "abに" {
		t.Fatalf("the preedit produced %q, want it after the text", tf.Text)
	}
	tf.OnComposition("にほ", 0, 0)
	if tf.Text != "abにほ" {
		t.Errorf("the next composition APPENDED instead of replacing: %q", tf.Text)
	}
	if !tf.composing {
		t.Error("the field does not consider itself composing")
	}
	tf.OnCharacter('日') // the IME commits
	if tf.Text != "ab日" {
		t.Errorf("the commit produced %q, want the preedit replaced by the committed rune", tf.Text)
	}
	if tf.composing {
		t.Error("the field is still composing after the commit")
	}
	tf.Undo()
	if tf.Text != "ab" {
		t.Errorf("one Undo after a commit produced %q, want %q", tf.Text, "ab")
	}
}

// Escape and an empty composition CANCEL the preedit, leaving no trace and no undo step.
func TestCompositionCanBeCancelled(t *testing.T) {
	tf := newTestField("ab")
	tf.CursorPos, tf.SelectStart, tf.SelectEnd = 2, 2, 2
	tf.selectionAnchor = 2

	tf.OnComposition("にほ", 0, 0)
	tf.OnComposition("", 0, 0)
	if tf.Text != "ab" {
		t.Errorf("an empty composition left %q behind", tf.Text)
	}

	tf.OnComposition("にほ", 0, 0)
	if !tf.OnKeyDown(platform.KeyEscape, platform.ModNone) {
		t.Error("Escape was not handled while composing")
	}
	if tf.Text != "ab" {
		t.Errorf("Escape left %q behind", tf.Text)
	}
	if tf.Undo() {
		t.Error("a cancelled preedit left an undo step")
	}

	ta := newTestArea("line")
	ta.CursorLine, ta.CursorCol = 0, 4
	ta.OnComposition("にほ", 0, 0)
	if ta.Value != "lineにほ" {
		t.Errorf("the text area preedit produced %q", ta.Value)
	}
	ta.OnComposition("", 0, 0)
	if ta.Value != "line" {
		t.Errorf("the text area kept %q after cancelling", ta.Value)
	}
	ta.OnComposition("にほ", 0, 0)
	ta.OnCharacter('日')
	if ta.Value != "line日" {
		t.Errorf("the text area commit produced %q, want %q", ta.Value, "line日")
	}
}

// The code editor shows the preedit inline and commits the same way.
func TestCompositionInTheCodeEditor(t *testing.T) {
	e := newTestEditor("ab")
	e.Focused = true
	e.CursorLine, e.CursorCol = 0, 2
	e.OnComposition("にほ", 0, 0)
	if e.Lines[0] != "abにほ" {
		t.Fatalf("the preedit produced %q", e.Lines[0])
	}
	e.OnComposition("に", 0, 0)
	if e.Lines[0] != "abに" {
		t.Errorf("the next composition appended instead of replacing: %q", e.Lines[0])
	}
	e.OnCharacter('日')
	if e.Lines[0] != "ab日" {
		t.Errorf("the commit produced %q, want %q", e.Lines[0], "ab日")
	}
}

// ── drag and drop ───────────────────────────────────────────────────────

// Dragging a selection into another field moves the text. The drag only STARTS when the press landed
// inside an existing selection — a press anywhere else is a selection gesture — and the drop lands at
// the pointer, in the field under it.
func TestDraggingSelectedTextBetweenFieldsMovesIt(t *testing.T) {
	w := meui.NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 600, 200))
	a := newTestField("hello")
	a.SetFrameRect(graphics.NewRect(0, 0, 200, 30))
	b := newTestField("")
	b.SetFrameRect(graphics.NewRect(0, 60, 200, 30))
	w.AddChild(a)
	w.AddChild(b)

	a.SetSelection(0, 5)
	a.applyClick(2, false) // inside the selection: arms the drag
	if !a.dragArmed {
		t.Fatal("a press inside the selection did not arm a drag")
	}

	w.Drag.RecordStart(2, 10)
	if !w.Drag.StartCheck(a, 20, 10) {
		t.Fatal("the drag did not start after the threshold")
	}
	w.Drag.Update(20, 70, 0, 0) // over field b
	w.Drag.End(20, 70)

	if a.Text != "" {
		t.Errorf("the source still holds %q after the drop", a.Text)
	}
	if b.Text != "hello" {
		t.Errorf("the target holds %q, want %q", b.Text, "hello")
	}
}

// Pressing outside the selection does NOT start a drag.
func TestDraggingWithoutASelectionPressDoesNotStart(t *testing.T) {
	tf := newTestField("hello world")
	tf.applyClick(8, false) // outside any selection
	if tf.dragArmed {
		t.Error("a plain press armed a drag")
	}
	if tf.OnDragStart(8, 0) != nil {
		t.Error("OnDragStart produced a payload for a plain press")
	}
}

// A drop into the field it came from MOVES the text; a drop INSIDE the source selection is a no-op.
func TestDroppingTextIntoTheSameFieldMovesIt(t *testing.T) {
	tf := newTestField("hello world")
	tf.SetSelection(0, 5)
	tf.applyClick(2, false)
	data := tf.OnDragStart(2, 0)
	if data == nil {
		t.Fatal("no drag payload")
	}
	// A drop past the end of the text: the selection is cut and re-inserted there.
	tf.OnDragDrop(data, 1000, 0)
	if tf.Text != " worldhello" {
		t.Errorf("the move produced %q, want %q", tf.Text, " worldhello")
	}

	tf = newTestField("hello world")
	tf.SetSelection(0, 5)
	tf.applyClick(2, false)
	data = tf.OnDragStart(2, 0)
	tf.OnDragDrop(data, 0, 0) // inside the source selection
	if tf.Text != "hello world" || tf.GetSelectedText() != "hello" {
		t.Errorf("a drop inside its own selection changed the text to %q", tf.Text)
	}
}

// The same gestures on a text area, across fields.
func TestDraggingTextIntoATextArea(t *testing.T) {
	w := meui.NewWindow()
	w.SetFrameRect(graphics.NewRect(0, 0, 600, 400))
	src := newTestField("moved")
	src.SetFrameRect(graphics.NewRect(0, 0, 200, 30))
	dst := newTestArea("line one\nline two")
	dst.SetFrameRect(graphics.NewRect(0, 60, 300, 200))
	w.AddChild(src)
	w.AddChild(dst)

	src.SetSelection(0, 5)
	src.applyClick(2, false)
	w.Drag.RecordStart(2, 10)
	if !w.Drag.StartCheck(src, 20, 10) {
		t.Fatal("the drag did not start")
	}
	// Drop on the SECOND line of the target: y = 60 (frame) + one line below the top.
	lh := dst.lineHeight()
	w.Drag.Update(20, 60+lh+2, 0, 0)
	w.Drag.End(20, 60+lh+2)

	if src.Text != "" {
		t.Errorf("the source still holds %q", src.Text)
	}
	if !strings.Contains(dst.Value, "moved") {
		t.Errorf("the text area did not receive the drop: %q", dst.Value)
	}
}

// ── overwrite mode ──────────────────────────────────────────────────────

// Overwrite (Insert-mode off) replaces the character under the caret instead of pushing it right, and
// appends at the end. A plain Insert toggles it.
func TestOverwriteModeReplacesInsteadOfInserting(t *testing.T) {
	tf := newTestField("abcd")
	tf.CursorPos = 1
	tf.SelectStart, tf.SelectEnd = 1, 1
	tf.selectionAnchor = 1
	tf.Overwrite = true
	tf.OnCharacter('X')
	if tf.Text != "aXcd" {
		t.Errorf("overwrite produced %q, want %q", tf.Text, "aXcd")
	}
	tf.CursorPos = len(tf.Text)
	tf.clearSelection()
	tf.OnCharacter('Z')
	if tf.Text != "aXcdZ" {
		t.Errorf("overwrite at the END produced %q, want the character appended", tf.Text)
	}
	tf.OnKeyDown(platform.KeyInsert, platform.ModNone)
	if tf.Overwrite {
		t.Error("Insert did not toggle overwrite off")
	}

	ta := newTestArea("abc\ndef")
	ta.CursorLine, ta.CursorCol = 0, 1
	ta.Overwrite = true
	ta.OnCharacter('X')
	if ta.Value != "aXc\ndef" {
		t.Errorf("TextArea overwrite produced %q, want %q", ta.Value, "aXc\ndef")
	}
	// The end of a line does not swallow its line break.
	ta.CursorLine, ta.CursorCol = 0, len("aXc")
	ta.OnCharacter('Y')
	if ta.Value != "aXcY\ndef" {
		t.Errorf("overwrite at a line end produced %q, want the break kept", ta.Value)
	}
}

// ── UTF-8 safety ─────────────────────────────────────────────────────────

// Backspace and Delete removed a BYTE in a TextArea, so deleting a multi-byte character left half of
// it behind and the document was no longer valid UTF-8.
func TestDeletingInATextAreaRemovesAWholeRune(t *testing.T) {
	ta := newTestArea("aé")
	ta.CursorLine, ta.CursorCol = 0, len(ta.Value) // col is a BYTE column
	ta.OnKeyDown(platform.KeyBackspace, platform.ModNone)
	if ta.Value != "a" {
		t.Errorf("Backspace left %q, want %q", ta.Value, "a")
	}
	if !utf8Valid(ta.Value) {
		t.Errorf("Backspace left invalid UTF-8: %q", ta.Value)
	}

	ta = newTestArea("éa")
	ta.CursorLine, ta.CursorCol = 0, 0
	ta.OnKeyDown(platform.KeyDelete, platform.ModNone)
	if ta.Value != "a" {
		t.Errorf("Delete left %q, want %q", ta.Value, "a")
	}
	if !utf8Valid(ta.Value) {
		t.Errorf("Delete left invalid UTF-8: %q", ta.Value)
	}
}

func utf8Valid(s string) bool {
	for _, r := range s {
		if r == '\uFFFD' {
			return false
		}
	}
	return true
}

// Ctrl+Backspace on the first column of a line deletes the word BEFORE it, across the line break —
// the old code gave up and deleted one character.
func TestCtrlBackspaceCrossesALineBreak(t *testing.T) {
	ta := newTestArea("one\ntwo")
	ta.CursorLine, ta.CursorCol = 1, 3
	ta.OnKeyDown(platform.KeyBackspace, platform.ModCtrl)
	if ta.Value != "one\n" {
		t.Errorf("Ctrl+Backspace left %q, want %q", ta.Value, "one\n")
	}
	ta.OnKeyDown(platform.KeyBackspace, platform.ModCtrl)
	if ta.Value != "" {
		t.Errorf("a second Ctrl+Backspace left %q, want the whole line gone", ta.Value)
	}
}

// ── page keys and the goal column ────────────────────────────────────────

// PageUp/PageDown did nothing at all in a text area.
func TestPageKeysMoveTheCaret(t *testing.T) {
	var lines []string
	for i := 0; i < 60; i++ {
		lines = append(lines, "line")
	}
	ta := newTestArea(strings.Join(lines, "\n"))
	ta.CursorLine, ta.CursorCol = 0, 0

	if !ta.OnKeyDown(platform.KeyPageDown, platform.ModNone) {
		t.Fatal("PageDown was not handled")
	}
	first := ta.CursorLine
	if first < 1 {
		t.Fatalf("PageDown left the caret on line %d", first)
	}
	if first > 20 {
		t.Errorf("PageDown jumped %d lines, more than a screen", first)
	}

	ta.OnKeyDown(platform.KeyPageUp, platform.ModNone)
	if ta.CursorLine != 0 {
		t.Errorf("PageUp returned to line %d, want 0", ta.CursorLine)
	}

	// With Shift the page is selected, not just traversed.
	ta.OnKeyDown(platform.KeyPageDown, platform.ModShift)
	if got := ta.GetSelectedText(); got == "" {
		t.Error("Shift+PageDown selected nothing")
	}
}

// Moving down through a SHORT line and back up must return to the column it started in. Without a
// goal column the caret drifts left a little with every pass.
func TestVerticalMovementKeepsTheGoalColumn(t *testing.T) {
	ta := newTestArea("abcdef\nab\nabcdef")
	ta.CursorLine, ta.CursorCol = 0, 4

	ta.OnKeyDown(platform.KeyDown, platform.ModNone)
	if ta.CursorLine != 1 || ta.CursorCol != 2 {
		t.Errorf("Down landed at line %d col %d, want line 1 col 2 (clamped)", ta.CursorLine, ta.CursorCol)
	}
	ta.OnKeyDown(platform.KeyDown, platform.ModNone)
	if ta.CursorLine != 2 || ta.CursorCol != 4 {
		t.Errorf("the second Down landed at line %d col %d, want the ORIGINAL column 4", ta.CursorLine, ta.CursorCol)
	}
	ta.OnKeyDown(platform.KeyUp, platform.ModShift)
	if ta.CursorLine != 1 || ta.CursorCol != 2 {
		t.Errorf("Up landed at line %d col %d, want line 1 col 2", ta.CursorLine, ta.CursorCol)
	}
	if got := ta.GetSelectedText(); got != "\nabcd" {
		t.Errorf("Shift+Up selected %q, want %q", got, "\nabcd")
	}
}
