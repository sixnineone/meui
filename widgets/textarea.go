package widgets

import (
	"fmt"
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
	"strings"
	"time"
	"unicode/utf8"
)

type TextArea struct {
	meui.BaseWidget
	*theme.TextAreaTheme
	Value           string
	Placeholder     string
	Focused         bool
	CursorLine      int
	CursorCol       int
	SelectStart     int
	SelectEnd       int
	ScrollOffset    float32
	CursorVisible   bool
	CursorTimer     *time.Timer
	OnTextChanged   func(string)
	ShowLineNumbers bool
	ShowScrollBar   bool
	WordWrap        bool
	ReadOnly        bool
	// Overwrite replaces the character under the caret instead of inserting before it. Insert toggles.
	Overwrite   bool
	MaxLength   int
	VScroll     *ScrollBar
	host        *ScrollbarHost
	isSelecting bool
	// isSelectingWord: a drag that began on a double click grows by whole words.
	isSelectingWord bool
	wordAnchorStart int
	wordAnchorEnd   int
	// Drag-and-drop: dragArmed means the press landed INSIDE the selection (the only way a text drag
	// starts); the drop fields remember the caret so a cancelled drag restores it.
	dragArmed    bool
	dragText     string
	pressLine    int
	pressCol     int
	dropActive   bool
	dropMoved    bool
	dropPrevLine int
	dropPrevCol  int
	// IME composition: the preedit lives in Value while composing, is not undoable and does not fire
	// OnTextChanged, and the commit (OnCharacter) removes it first.
	composing    bool
	composeStart int
	composeEnd   int
	// goalCol is the column a run of vertical moves aims at (-1 = none).
	goalCol         int
	selectionAnchor int
	linesCache      []string
	contentLines    []string
	dispToContent   []int
	wrapWidth       float32
	undoStack       []textAreaState
	redoStack       []textAreaState
	maxUndo         int
	// undoKind/undoTime coalesce a burst of one kind of edit into ONE undo step.
	undoKind      int
	undoTime      time.Time
	ctxMenu       *ContextMenu
	cache         meui.CacheBase
	textCache     *graphics.Text
	clickCount    int
	lastClickTime time.Time
	lastClickLine int
	lastClickCol  int
}

type textAreaState struct {
	value       string
	cursorLine  int
	cursorCol   int
	selectStart int
	selectEnd   int
	scrollOff   float32
}

func NewTextArea() *TextArea {
	t := &TextArea{
		BaseWidget:    *meui.NewBaseWidget(),
		TextAreaTheme: theme.DefaultTextAreaTheme,
		maxUndo:       50,
		ShowScrollBar: true,
		goalCol:       -1,
	}
	t.SetTabIndex(0)
	t.host = NewScrollbarHost()
	t.host.ShowH = false
	t.VScroll = t.host.VScroll
	t.VScroll.SetParent(t)
	t.host.SetOnChange(func() {
		_, t.ScrollOffset = t.host.Offsets()
		t.cache.Invalidate()
		t.MarkForRedraw()
	})
	t.rebuildLines()
	return t
}

func (t *TextArea) SetFrameRect(rect graphics.Rect) {
	t.BaseWidget.SetFrameRect(rect)
	if t.WordWrap {
		t.wrapWidth = 0
	}
}

// lines returns the display lines (wrapped when WordWrap is enabled).
func (t *TextArea) lines() []string {
	if t.linesCache == nil || (t.WordWrap && t.shouldRewrap()) {
		t.rebuildLines()
	}
	return t.linesCache
}

func (t *TextArea) shouldRewrap() bool {
	return t.availableTextWidth() != t.wrapWidth
}

func (t *TextArea) availableTextWidth() float32 {
	frame := t.FrameRect()
	maxLine := len(t.contentLines)
	if maxLine == 0 {
		maxLine = 1
	}
	gutterW := t.gutterWidthFor(maxLine)
	scrollbarW := float32(0)
	if t.ShowScrollBar {
		scrollbarW = 14
	}
	return frame.Width - 8 - gutterW - scrollbarW
}

func (t *TextArea) contentSlice() []string {
	if t.contentLines == nil {
		t.rebuildLines()
	}
	return t.contentLines
}

func (t *TextArea) rebuildLines() {
	t.contentLines = strings.Split(t.Value, "\n")
	if !t.WordWrap {
		t.linesCache = t.contentLines
		t.dispToContent = nil
		t.wrapWidth = 0
		return
	}
	availW := t.availableTextWidth()
	if availW < 50 {
		availW = 50
	}
	t.wrapWidth = availW
	font := t.font()
	t.linesCache = nil
	t.dispToContent = nil
	for ci, line := range t.contentLines {
		if font.SimpleWidth(line) <= availW {
			t.linesCache = append(t.linesCache, line)
			t.dispToContent = append(t.dispToContent, ci)
			continue
		}
		// Shared greedy wrapper (textlayout.go); one display line per result.
		for _, wrapped := range wrapText(line, font, availW) {
			t.linesCache = append(t.linesCache, wrapped)
			t.dispToContent = append(t.dispToContent, ci)
		}
	}
}

func (t *TextArea) SetValue(text string) *TextArea {
	t.Value = text
	if t.MaxLength > 0 && utf8.RuneCountInString(t.Value) > t.MaxLength {
		t.Value = string([]rune(t.Value)[:t.MaxLength])
	}
	t.textCache = nil
	t.rebuildLines()
	t.clampCursor()
	t.clearSelection() // stale selection indices would crash paste paths
	t.cache.Invalidate()
	t.MarkForRedraw()
	return t
}

func (t *TextArea) SetText(text string) *TextArea {
	return t.SetValue(text)
}

func (t *TextArea) SetPlaceholder(text string) *TextArea {
	t.Placeholder = text
	t.cache.Invalidate()
	t.MarkForRedraw()
	return t
}

func (t *TextArea) SetOnTextChanged(handler func(string)) *TextArea {
	t.OnTextChanged = handler
	return t
}

func (t *TextArea) SetShowLineNumbers(show bool) *TextArea {
	t.ShowLineNumbers = show
	t.cache.Invalidate()
	t.MarkForRedraw()
	return t
}

func (t *TextArea) SetShowScrollBar(show bool) *TextArea {
	t.ShowScrollBar = show
	t.MarkForRedraw()
	return t
}

func (t *TextArea) SetWordWrap(wrap bool) *TextArea {
	t.WordWrap = wrap
	t.rebuildLines()
	t.cache.Invalidate()
	t.MarkForRedraw()
	return t
}

func (t *TextArea) ValueText() string {
	return t.Value
}

// ── Cursor helpers ──

func (t *TextArea) clampCursor() {
	cl := t.contentSlice()
	if t.CursorLine < 0 {
		t.CursorLine = 0
	}
	if t.CursorLine >= len(cl) {
		t.CursorLine = len(cl) - 1
	}
	if t.CursorLine < 0 {
		t.CursorLine = 0
	}
	lineLen := len(cl[t.CursorLine])
	if t.CursorCol < 0 {
		t.CursorCol = 0
	}
	if t.CursorCol > lineLen {
		t.CursorCol = lineLen
	}
}

func (t *TextArea) hasSelection() bool {
	return t.SelectStart != t.SelectEnd
}

func (t *TextArea) selectionBounds() (int, int) {
	s, e := t.SelectStart, t.SelectEnd
	if s > e {
		s, e = e, s
	}
	return s, e
}

func (t *TextArea) clearSelection() {
	offset := t.cursorToOffset(t.CursorLine, t.CursorCol)
	t.SelectStart = offset
	t.SelectEnd = offset
}

// prevRuneCol moves cursor left by one rune (within line boundary).
func (t *TextArea) prevRuneCol(line, col int) (int, bool) {
	if col <= 0 {
		return col, false
	}
	_, sz := utf8.DecodeLastRuneInString(t.contentSlice()[line][:col])
	return col - sz, true
}

// nextRuneCol moves cursor right by one rune (within line boundary).
func (t *TextArea) nextRuneCol(line, col int) (int, bool) {
	cl := t.contentSlice()
	if col >= len(cl[line]) {
		return col, false
	}
	_, sz := utf8.DecodeRuneInString(cl[line][col:])
	return col + sz, true
}

// runeAt is defined in textfield.go (shared).

// ── Undo / Redo ─────────────────────────────────────────────────

func (t *TextArea) saveState() {
	t.undoKind = undoKindNone
	state := textAreaState{t.Value, t.CursorLine, t.CursorCol, t.SelectStart, t.SelectEnd, t.ScrollOffset}
	t.undoStack = append(t.undoStack, state)
	if len(t.undoStack) > t.maxUndo {
		t.undoStack = t.undoStack[1:]
	}
	t.redoStack = nil
}

func (t *TextArea) undo() {
	t.undoKind = undoKindNone
	if len(t.undoStack) == 0 {
		return
	}
	current := textAreaState{t.Value, t.CursorLine, t.CursorCol, t.SelectStart, t.SelectEnd, t.ScrollOffset}
	t.redoStack = append(t.redoStack, current)
	prev := t.undoStack[len(t.undoStack)-1]
	t.undoStack = t.undoStack[:len(t.undoStack)-1]
	t.restoreState(prev)
}

func (t *TextArea) redo() {
	t.undoKind = undoKindNone
	if len(t.redoStack) == 0 {
		return
	}
	current := textAreaState{t.Value, t.CursorLine, t.CursorCol, t.SelectStart, t.SelectEnd, t.ScrollOffset}
	t.undoStack = append(t.undoStack, current)
	next := t.redoStack[len(t.redoStack)-1]
	t.redoStack = t.redoStack[:len(t.redoStack)-1]
	t.restoreState(next)
}

// saveStateGrouped records a state unless the same KIND of edit happened a moment ago — a typing
// burst, or a held Backspace, is one undo step.
func (t *TextArea) saveStateGrouped(kind int) {
	now := time.Now()
	if kind != undoKindNone && kind == t.undoKind && now.Sub(t.undoTime) < undoGroupWindow {
		t.undoTime = now
		t.redoStack = nil
		return
	}
	t.saveState()
	t.undoKind = kind
	t.undoTime = now
}

func (t *TextArea) deleteSelection() {
	if !t.hasSelection() {
		return
	}
	t.saveState()
	start, end := t.selectionBounds()
	startLine, startCol := t.offsetToCursor(start)
	t.Value = t.Value[:start] + t.Value[end:]
	t.rebuildLines()
	t.CursorLine = startLine
	t.CursorCol = startCol
	t.goalCol = -1
	t.clearSelection()
	t.ensureCursorVisible()
	t.cache.Invalidate()
	t.MarkForRedraw()
	if t.OnTextChanged != nil {
		t.OnTextChanged(t.Value)
	}
}

func (t *TextArea) restoreState(s textAreaState) {
	t.Value = s.value
	t.CursorLine = s.cursorLine
	t.CursorCol = s.cursorCol
	t.SelectStart = s.selectStart
	t.SelectEnd = s.selectEnd
	t.host.SetOffsets(0, s.scrollOff)
	_, t.ScrollOffset = t.host.Offsets()
	t.rebuildLines()
	t.clampCursor()
	t.ensureCursorVisible()
	t.cache.Invalidate()
	t.MarkForRedraw()
	if t.OnTextChanged != nil {
		t.OnTextChanged(t.Value)
	}
}

func (t *TextArea) font() graphics.Font {
	font := t.TextAreaTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	return font
}

func (t *TextArea) lineHeight() float32 {
	return t.font().LineHeight() + 2
}

func (t *TextArea) cursorToOffset(line, col int) int {
	// Shared with the code editor (textlayout.go).
	return docOffsetAt(t.contentSlice(), line, col)
}

func (t *TextArea) offsetToCursor(offset int) (line, col int) {
	// Shared with the code editor (textlayout.go).
	return docLineColAt(t.contentSlice(), offset)
}

func (t *TextArea) moveCursor(line, col int) {
	ln := t.contentSlice()
	if line < 0 {
		line = 0
	}
	if line >= len(ln) {
		line = len(ln) - 1
	}
	if line < 0 {
		line = 0
	}
	if col < 0 {
		col = 0
	}
	if col > len(ln[line]) {
		col = len(ln[line])
	}
	t.CursorLine = line
	t.CursorCol = col
}

// displayToContent maps a display line index to content line + column offset.
func (t *TextArea) displayToContent(displayLine int) (contentLine, col int) {
	if !t.WordWrap || t.dispToContent == nil || displayLine < 0 || displayLine >= len(t.dispToContent) {
		return displayLine, 0
	}
	dl := t.lines()
	ci := t.dispToContent[displayLine]
	colOffset := 0
	for di := 0; di < len(dl); di++ {
		if t.dispToContent[di] != ci {
			continue
		}
		if di == displayLine {
			return ci, colOffset
		}
		colOffset += len(dl[di])
	}
	return ci, 0
}

func (t *TextArea) ensureCursorVisible() {
	lh := t.lineHeight()
	frame := t.FrameRect()
	innerH := frame.Height - 8
	cursorDisplayLine := t.contentLineToDisplay(t.CursorLine)
	cursorY := float32(cursorDisplayLine) * lh
	totalH := float32(len(t.lines())) * lh
	t.host.SetExtents(0, totalH, 0, innerH)
	offX, offY := t.host.Offsets()
	if cursorY < offY {
		offY = cursorY
	} else if cursorY+lh > offY+innerH {
		offY = cursorY + lh - innerH
	}
	t.host.SetOffsets(offX, offY)
	_, t.ScrollOffset = t.host.Offsets()
}

func (t *TextArea) contentLineToDisplay(contentLine int) int {
	if !t.WordWrap || t.dispToContent == nil {
		return contentLine
	}
	for di, ci := range t.dispToContent {
		if ci == contentLine {
			return di
		}
	}
	return contentLine
}

func (t *TextArea) contentToDisplayCol(contentLine, contentCol int) (dispLine, dispCol int) {
	if !t.WordWrap || t.dispToContent == nil {
		return contentLine, contentCol
	}
	dispLine = t.contentLineToDisplay(contentLine)
	dispCol = contentCol
	dl := t.lines()
	for di := dispLine; di >= 0 && di < len(t.dispToContent) && t.dispToContent[di] == contentLine; di-- {
		if di < dispLine {
			dispCol -= len(dl[di])
		}
	}
	if dispCol < 0 {
		dispCol = 0
	}
	return
}

// ── Clipboard helpers ───────────────────────────────────────────

func (t *TextArea) Cut() {
	if t.ReadOnly {
		return
	}
	// No selection: Cut does NOTHING. It used to select all and cut, so Ctrl+X wiped the value — a
	// data-loss trap in a form.
	if !t.hasSelection() {
		return
	}
	start, end := t.selectionBounds()
	platform.GlobalClipboard.SetText(t.Value[start:end])
	t.deleteSelection()
}

func (t *TextArea) Copy() {
	if !t.hasSelection() {
		return
	}
	start, end := t.selectionBounds()
	platform.GlobalClipboard.SetText(t.Value[start:end])
}

func (t *TextArea) Paste() {
	if t.ReadOnly {
		return
	}
	text := platform.GlobalClipboard.GetText()
	if text == "" {
		return
	}
	// Enforce MaxLength
	if t.MaxLength > 0 {
		currentRunes := utf8.RuneCountInString(t.Value)
		pasteRunes := utf8.RuneCountInString(text)
		if t.hasSelection() {
			selStart, selEnd := t.selectionBounds()
			replacedRunes := utf8.RuneCountInString(t.Value[selStart:selEnd])
			available := t.MaxLength - currentRunes + replacedRunes
			if pasteRunes > available {
				text = string([]rune(text)[:available])
			}
		} else {
			available := t.MaxLength - currentRunes
			if pasteRunes > available {
				text = string([]rune(text)[:available])
			}
		}
	}
	// A paste is its own undo step: the state is recorded here, and the raw insert below must not
	// merge it with the typing that came before.
	t.saveState()
	t.insertRaw(text)
}

func (t *TextArea) SelectAll() {
	t.SelectStart = 0
	t.SelectEnd = len(t.Value)
	t.CursorLine, t.CursorCol = t.offsetToCursor(len(t.Value))
	t.ensureCursorVisible()
	t.cache.Invalidate()
	t.MarkForRedraw()
}

// SetReadOnly disables editing (selection and copy still work).
func (t *TextArea) SetReadOnly(readOnly bool) *TextArea {
	t.ReadOnly = readOnly
	t.MarkForRedraw()
	return t
}

func (t *TextArea) IsReadOnly() bool { return t.ReadOnly }

// AppendText adds text at the end of the buffer (ignores read-only mode? no —
// it respects it like Qt's read-only append; pass SetReadOnly(false) first).
func (t *TextArea) AppendText(text string) {
	if t.ReadOnly || text == "" {
		return
	}
	if t.Value != "" && !strings.HasSuffix(t.Value, "\n") {
		text = "\n" + text
	}
	t.SetValue(t.Value + text)
	t.ensureCursorVisible()
}

// Clear removes all text.
func (t *TextArea) Clear() {
	if t.ReadOnly || t.Value == "" {
		return
	}
	t.SetValue("")
}

// GetSelectedText returns the selected text, or "" if nothing is selected.
func (t *TextArea) GetSelectedText() string {
	if !t.hasSelection() {
		return ""
	}
	start, end := t.selectionBounds()
	return t.Value[start:end]
}

// SelectionLength returns the selected byte length.
func (t *TextArea) SelectionLength() int {
	if !t.hasSelection() {
		return 0
	}
	start, end := t.selectionBounds()
	return end - start
}

// SetSelection selects [start, end) byte offsets (clamped) and moves the
// cursor to the end of the selection.
func (t *TextArea) SetSelection(start, end int) {
	max := len(t.Value)
	if start < 0 {
		start = 0
	}
	if end > max {
		end = max
	}
	if end < start {
		start, end = end, start
	}
	t.SelectStart = start
	t.SelectEnd = end
	t.CursorLine, t.CursorCol = t.offsetToCursor(end)
	t.ensureCursorVisible()
	t.cache.Invalidate()
	t.MarkForRedraw()
}

// selectWordAt selects the word at the given line/col position.
func (t *TextArea) selectWordAt(line, col int) {
	cl := t.contentSlice()
	if line < 0 || line >= len(cl) {
		return
	}
	l := cl[line]
	if col > len(l) {
		col = len(l)
	}
	start, end := wordSpanAt(l, col)
	t.CursorLine = line
	t.CursorCol = end
	startOff := t.cursorToOffset(line, start)
	endOff := t.cursorToOffset(line, end)
	t.SelectStart = startOff
	t.SelectEnd = endOff
	t.selectionAnchor = startOff
	t.wordAnchorStart, t.wordAnchorEnd = startOff, endOff
	t.cache.Invalidate()
	t.MarkForRedraw()
}

func (t *TextArea) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !t.Visible() {
		return
	}
	canvas.Save()
	t.cache.GetCached(canvas, rect, func(c *graphics.Canvas, r graphics.Rect) {
		disabled := !t.Enabled()
		bgPaint := graphics.NewPaint()
		if disabled {
			bgPaint.SetColor(theme.DefaultTheme.SurfaceVariant)
		} else if t.Focused {
			bgPaint.SetColor(t.TextAreaTheme.BackgroundFocused)
		} else {
			bgPaint.SetColor(t.TextAreaTheme.BackgroundInk)
		}
		c.DrawRoundedRect(r, t.TextAreaTheme.CornerRadius, t.TextAreaTheme.CornerRadius, bgPaint)
		borderPaint := graphics.NewPaint()
		borderPaint.SetStyle(paintstyle.Stroke)
		borderPaint.SetStrokeWidth(theme.BorderStrokeWidth)
		if t.Focused {
			borderPaint.SetColor(t.TextAreaTheme.EdgeFocused)
		} else {
			borderPaint.SetColor(t.TextAreaTheme.EdgeInk)
		}
		c.DrawRoundedRect(r, t.TextAreaTheme.CornerRadius, t.TextAreaTheme.CornerRadius, borderPaint)

		gutterW := t.gutterWidthFor(len(t.lines()))
		if t.ShowLineNumbers {
			gutterPaint := graphics.NewPaint()
			gutterPaint.SetColor(t.TextAreaTheme.GutterBgInk)
			c.DrawRect(graphics.NewRect(r.X, r.Y, gutterW, r.Height), gutterPaint)
			gutterBorder := graphics.NewPaint()
			gutterBorder.SetStyle(paintstyle.Stroke)
			gutterBorder.SetStrokeWidth(1)
			gutterBorder.SetColor(t.TextAreaTheme.EdgeInk)
			c.DrawLine(r.X+gutterW, r.Y, r.X+gutterW, r.Y+r.Height, gutterBorder)
		}
		scrollbarW := float32(0)
		if t.ShowScrollBar {
			scrollbarW = t.host.BarSize
		}
		innerRect := graphics.NewRect(r.X+4+gutterW, r.Y+4, r.Width-8-gutterW-scrollbarW, r.Height-8)
		font := t.font()
		ascent := font.Baseline()
		lh := t.lineHeight()
		isPlaceholder := false
		if t.Value == "" && t.Placeholder != "" {
			isPlaceholder = true
		}
		ln := t.lines()

		// Draw line numbers
		if t.ShowLineNumbers {
			for i := 0; i < len(ln); i++ {
				lineY := innerRect.Y + float32(i)*lh - t.ScrollOffset
				if lineY+lh < innerRect.Y {
					continue
				}
				if lineY > innerRect.Y+innerRect.Height {
					break
				}
				numPaint := graphics.NewPaint()
				if i == t.CursorLine {
					numPaint.SetColor(t.TextAreaTheme.GutterActiveInk)
				} else {
					numPaint.SetColor(t.TextAreaTheme.GutterTextInk)
				}
				numText := fmt.Sprintf("%d", i+1)
				numW := font.SimpleWidth(numText)
				nt := graphics.NewText(numText, &graphics.TextDecoration{Font: font, OnBackgroundInk: numPaint.Color()})
				nt.Draw(c, r.X+gutterW-4-numW, lineY+ascent)
			}
		}

		// Clip to text area and draw content
		c.ClipRect(innerRect, pathop.Intersect, false)
		for i := 0; i < len(ln); i++ {
			lineY := innerRect.Y + float32(i)*lh - t.ScrollOffset
			if lineY+lh < innerRect.Y {
				continue
			}
			if lineY > innerRect.Y+innerRect.Height {
				break
			}
			line := ln[i]
			baseline := lineY + ascent
			contentLine := i
			colOffset := 0
			if t.WordWrap && t.dispToContent != nil && i < len(t.dispToContent) {
				contentLine = t.dispToContent[i]
				for di := i; di >= 0 && di < len(t.dispToContent) && t.dispToContent[di] == contentLine; di-- {
					if di < i {
						colOffset += len(ln[di])
					}
				}
			}
			lineStartOffset := t.cursorToOffset(contentLine, colOffset)
			lineEndOffset := lineStartOffset + len(line)
			if t.Focused && t.hasSelection() && !isPlaceholder {
				selStart, selEnd := t.selectionBounds()
				if selEnd > lineStartOffset && selStart < lineEndOffset {
					relStart := selStart - lineStartOffset
					if relStart < 0 {
						relStart = 0
					}
					relEnd := selEnd - lineStartOffset
					if relEnd > len(line) {
						relEnd = len(line)
					}
					if relStart < relEnd {
						preW := font.SimpleWidth(line[:relStart])
						selW := font.SimpleWidth(line[relStart:relEnd])
						selPaint := graphics.NewPaint()
						selPaint.SetColor(t.TextAreaTheme.SelectionColor)
						c.DrawRect(graphics.NewRect(innerRect.X+preW, lineY, selW, lh), selPaint)
					}
				}
			}
			cursorDispLine, cursorDispCol := t.contentToDisplayCol(t.CursorLine, t.CursorCol)
			if t.Focused && i == cursorDispLine && t.CursorVisible && !disabled && !isPlaceholder {
				if cursorDispCol > len(line) {
					cursorDispCol = len(line)
				}
				cursorX := innerRect.X + font.SimpleWidth(line[:cursorDispCol])
				cursorPaint := graphics.NewPaint()
				cursorPaint.SetColor(t.TextAreaTheme.TextInk)
				c.DrawLine(cursorX, lineY, cursorX, lineY+lh, cursorPaint)
			}
			textPaint := graphics.NewPaint()
			if disabled {
				textPaint.SetColor(theme.DefaultTheme.DisabledText)
			} else if isPlaceholder {
				textPaint.SetColor(t.TextAreaTheme.PlaceholderInk)
			} else {
				textPaint.SetColor(t.TextAreaTheme.TextInk)
			}
			lt := graphics.NewText(line, &graphics.TextDecoration{Font: font, OnBackgroundInk: textPaint.Color()})
			lt.Draw(c, innerRect.X, baseline)
			if t.composing && !isPlaceholder && t.composeEnd > t.composeStart &&
				t.composeEnd > lineStartOffset && t.composeStart < lineEndOffset {
				// A preedit is UNDERLINED: it is shown, but not committed.
				relLo := t.composeStart - lineStartOffset
				if relLo < 0 {
					relLo = 0
				}
				relHi := t.composeEnd - lineStartOffset
				if relHi > len(line) {
					relHi = len(line)
				}
				if relLo < relHi {
					ux1 := innerRect.X + font.SimpleWidth(line[:relLo])
					ux2 := innerRect.X + font.SimpleWidth(line[:relHi])
					ul := graphics.NewPaint()
					ul.SetColor(textPaint.Color())
					c.DrawRect(graphics.NewRect(ux1, baseline+2, ux2-ux1, 1), ul)
				}
			}
		}
		if t.Focused && t.CursorVisible && !disabled && !isPlaceholder && len(ln) == 1 && ln[0] == "" {
			cursorPaint := graphics.NewPaint()
			cursorPaint.SetColor(t.TextAreaTheme.TextInk)
			c.DrawLine(innerRect.X, innerRect.Y, innerRect.X, innerRect.Y+lh, cursorPaint)
		}

		// Scrollbar (host: extents, layout, paint — absolute pixels)
		if t.ShowScrollBar {
			totalH := float32(len(ln)) * lh
			t.host.SetExtents(r.Width, totalH, r.Width, innerRect.Height)
			t.host.LayoutBars(r.Width, r.Height)
			_, t.ScrollOffset = t.host.Offsets()
			t.host.PaintBars(c)
		}
	})
	canvas.Restore()
}

// OnComposition shows the IME's in-progress text, replacing the previous preedit; empty text means
// the composition was cancelled. It is not undoable and does not fire OnTextChanged — the committed
// characters arrive through OnCharacter.
func (t *TextArea) OnComposition(text string, start, length int) bool {
	if !t.Enabled() || !t.Focused || t.ReadOnly {
		return false
	}
	t.cancelComposition()
	if text == "" {
		t.MarkForRedraw()
		return true
	}
	if t.hasSelection() {
		s, e := t.selectionBounds()
		t.Value = t.Value[:s] + t.Value[e:]
		t.rebuildLines()
		t.CursorLine, t.CursorCol = t.offsetToCursor(s)
		t.clearSelection()
	}
	off := t.cursorToOffset(t.CursorLine, t.CursorCol)
	if t.MaxLength > 0 {
		available := t.MaxLength - utf8.RuneCountInString(t.Value)
		if available <= 0 {
			t.MarkForRedraw()
			return true
		}
		if n := utf8.RuneCountInString(text); n > available {
			text = string([]rune(text)[:available])
		}
	}
	t.Value = t.Value[:off] + text + t.Value[off:]
	t.rebuildLines()
	t.composeStart, t.composeEnd = off, off+len(text)
	t.composing = true
	caret := start
	if caret < 0 {
		caret = 0
	}
	if caret > len(text) {
		caret = len(text)
	}
	t.CursorLine, t.CursorCol = t.offsetToCursor(t.composeStart + caret)
	t.clearSelection()
	t.goalCol = -1
	t.ensureCursorVisible()
	t.cache.Invalidate()
	t.MarkForRedraw()
	return true
}

// cancelComposition removes the preedit without an undo step or a change notification.
func (t *TextArea) cancelComposition() {
	if !t.composing {
		return
	}
	lo, hi := t.composeStart, t.composeEnd
	if lo < 0 {
		lo = 0
	}
	if hi > len(t.Value) {
		hi = len(t.Value)
	}
	if lo < hi {
		t.Value = t.Value[:lo] + t.Value[hi:]
		t.rebuildLines()
		t.CursorLine, t.CursorCol = t.offsetToCursor(lo)
	}
	t.composing = false
	t.composeStart, t.composeEnd = 0, 0
	t.clearSelection()
	t.cache.Invalidate()
	t.MarkForRedraw()
}

func (t *TextArea) insertAtCursor(text string) {
	if t.ReadOnly {
		return
	}
	text = t.clampInsert(text)
	if text == "" {
		return
	}
	t.saveStateGrouped(undoKindInsert)
	t.insertRaw(text)
}

// clampInsert enforces MaxLength (in runes) for an insertion at the caret, allowing for the
// selection it would replace.
func (t *TextArea) clampInsert(text string) string {
	if text == "" || t.MaxLength <= 0 {
		return text
	}
	currentRunes := utf8.RuneCountInString(t.Value)
	insertRunes := utf8.RuneCountInString(text)
	var replacedRunes int
	if t.hasSelection() {
		selStart, selEnd := t.selectionBounds()
		replacedRunes = utf8.RuneCountInString(t.Value[selStart:selEnd])
	}
	available := t.MaxLength - currentRunes + replacedRunes
	if insertRunes > available {
		if available <= 0 {
			return ""
		}
		return string([]rune(text)[:available])
	}
	return text
}

// insertRaw performs the insertion. The caller has already recorded the undo state (grouped, or a
// paste's own step), which is what keeps typing coalesced without merging a paste into it.
func (t *TextArea) insertRaw(text string) {
	if t.ReadOnly || text == "" {
		return
	}
	// Overwrite mode replaces as many characters as the insertion brings, stopping at a line end so
	// a paste does not swallow the line break it lands on.
	if t.Overwrite && !t.hasSelection() {
		off := t.cursorToOffset(t.CursorLine, t.CursorCol)
		end := off
		want := utf8.RuneCountInString(text)
		for i := 0; i < want && end < len(t.Value) && t.Value[end] != '\n'; i++ {
			end = runeEndAfter(t.Value, end)
		}
		if end > off {
			t.Value = t.Value[:off] + t.Value[end:]
			t.rebuildLines()
		}
	}
	if t.hasSelection() {
		selStart, selEnd := t.selectionBounds()
		t.CursorLine, t.CursorCol = t.offsetToCursor(selStart)
		t.Value = t.Value[:selStart] + text + t.Value[selEnd:]
		t.rebuildLines()
		newOffset := selStart + len(text)
		t.CursorLine, t.CursorCol = t.offsetToCursor(newOffset)
		t.clearSelection()
		t.ensureCursorVisible()
		t.cache.Invalidate()
		t.MarkForRedraw()
		if t.OnTextChanged != nil {
			t.OnTextChanged(t.Value)
		}
		return
	}
	offset := t.cursorToOffset(t.CursorLine, t.CursorCol)
	t.Value = t.Value[:offset] + text + t.Value[offset:]
	t.rebuildLines()
	newLine, newCol := t.offsetToCursor(offset + len(text))
	t.CursorLine = newLine
	t.CursorCol = newCol
	t.goalCol = -1
	t.clearSelection()
	t.ensureCursorVisible()
	t.cache.Invalidate()
	t.MarkForRedraw()
	if t.OnTextChanged != nil {
		t.OnTextChanged(t.Value)
	}
}

// updateSelectionFromAnchor updates SelectStart/SelectEnd based on
// selectionAnchor and current cursor offset.
func (t *TextArea) updateSelectionFromAnchor() {
	offset := t.cursorToOffset(t.CursorLine, t.CursorCol)
	if offset < t.selectionAnchor {
		t.SelectStart = offset
		t.SelectEnd = t.selectionAnchor
	} else {
		t.SelectStart = t.selectionAnchor
		t.SelectEnd = offset
	}
}

// OnKeyRepeat is a key the user is HOLDING — Backspace over a word, an arrow along a row, a scrub.
// The window classifies an OS auto-repeat as a repeat rather than a fresh press (a widget without
// this method is asked once, deliberately), so this is what makes a held key work here.
func (t *TextArea) OnKeyRepeat(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	return t.OnKeyDown(keyCode, mod)
}

func (t *TextArea) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if !t.Enabled() || !t.Focused {
		return false
	}
	if t.composing && keyCode == platform.KeyEscape {
		t.cancelComposition()
		return true
	}

	// Clipboard and undo. A read-only area still copies and selects all — that is what it offers.
	if commandDown(mod) {
		switch keyCode {
		case platform.KeyA:
			t.SelectAll()
			return true
		case platform.KeyC:
			t.Copy()
			return true
		case platform.KeyInsert:
			t.Copy()
			return true
		}
		if t.ReadOnly {
			// A read-only area is still READABLE: navigation and selection keep working; only the
			// editing commands are refused.
			switch keyCode {
			case platform.KeyLeft, platform.KeyRight, platform.KeyUp, platform.KeyDown,
				platform.KeyHome, platform.KeyEnd, platform.KeyPageUp, platform.KeyPageDown:
			default:
				return false
			}
		} else {
			switch keyCode {
			case platform.KeyX:
				t.Cut()
				return true
			case platform.KeyV:
				t.Paste()
				return true
			case platform.KeyZ:
				if mod.ShiftDown() {
					t.redo()
				} else {
					t.undo()
				}
				return true
			case platform.KeyY:
				t.redo()
				return true
			}
		}
	}
	// The classic clipboard keys.
	if keyCode == platform.KeyInsert && mod.ShiftDown() {
		t.Paste()
		return true
	}
	if keyCode == platform.KeyDelete && mod.ShiftDown() {
		t.Cut()
		return true
	}
	// A plain Insert toggles OVERWRITE mode.
	if keyCode == platform.KeyInsert {
		if !t.ReadOnly {
			t.Overwrite = !t.Overwrite
			t.MarkForRedraw()
		}
		return true
	}

	switch keyCode {
	case platform.KeyLeft, platform.KeyRight:
		left := keyCode == platform.KeyLeft
		word := wordModDown(mod)
		if t.hasSelection() && !mod.ShiftDown() {
			// Collapse TOWARDS the direction of the arrow: Left to the START of the selection,
			// Right to the END. They were swapped, so Left jumped past the whole selection.
			s, e := t.selectionBounds()
			if left {
				t.CursorLine, t.CursorCol = t.offsetToCursor(s)
			} else {
				t.CursorLine, t.CursorCol = t.offsetToCursor(e)
			}
			t.clearSelection()
		} else {
			if mod.ShiftDown() && !t.hasSelection() {
				t.selectionAnchor = t.cursorToOffset(t.CursorLine, t.CursorCol)
			}
			t.moveHorizontally(left, word)
			if mod.ShiftDown() {
				t.updateSelectionFromAnchor()
			} else {
				t.clearSelection()
			}
		}
		t.goalCol = -1
		t.undoKind = undoKindNone
		t.afterCaretMove()
		return true

	case platform.KeyUp, platform.KeyDown:
		delta := -1
		if keyCode == platform.KeyDown {
			delta = 1
		}
		t.moveVertically(delta, mod.ShiftDown())
		return true

	case platform.KeyPageUp, platform.KeyPageDown:
		delta := -t.visibleLines()
		if keyCode == platform.KeyPageDown {
			delta = t.visibleLines()
		}
		t.moveVertically(delta, mod.ShiftDown())
		return true

	case platform.KeyHome, platform.KeyEnd:
		if mod.ShiftDown() && !t.hasSelection() {
			t.selectionAnchor = t.cursorToOffset(t.CursorLine, t.CursorCol)
		}
		cl := t.contentSlice()
		if keyCode == platform.KeyHome {
			if commandDown(mod) {
				t.CursorLine, t.CursorCol = 0, 0
			} else {
				t.CursorCol = 0
			}
		} else {
			if commandDown(mod) {
				t.CursorLine = len(cl) - 1
			}
			if t.CursorLine >= 0 && t.CursorLine < len(cl) {
				t.CursorCol = len(cl[t.CursorLine])
			}
		}
		if mod.ShiftDown() {
			t.updateSelectionFromAnchor()
		} else {
			t.clearSelection()
		}
		t.goalCol = -1
		t.undoKind = undoKindNone
		t.afterCaretMove()
		return true

	case platform.KeyBackspace:
		if t.ReadOnly {
			return true
		}
		if t.hasSelection() {
			t.deleteSelection()
			return true
		}
		off := t.cursorToOffset(t.CursorLine, t.CursorCol)
		if off <= 0 {
			return true
		}
		lo := runeStartBefore(t.Value, off)
		if wordModDown(mod) {
			lo = wordLeftOffset(t.Value, off)
		}
		t.deleteRange(lo, off)
		return true

	case platform.KeyDelete:
		if t.ReadOnly {
			return true
		}
		if t.hasSelection() {
			t.deleteSelection()
			return true
		}
		off := t.cursorToOffset(t.CursorLine, t.CursorCol)
		if off >= len(t.Value) {
			return true
		}
		hi := runeEndAfter(t.Value, off)
		if wordModDown(mod) {
			hi = wordRightOffset(t.Value, off)
		}
		t.deleteRange(off, hi)
		return true

	case platform.KeyReturn, platform.KeyNumPadEnter:
		if !t.ReadOnly {
			t.insertAtCursor("\n")
		}
		return true

	case platform.KeyTab:
		// The window takes Tab for focus traversal before a widget sees it, so a form's Tab still
		// moves on; an embedded editor dispatched directly gets an indentation tab.
		if !t.ReadOnly {
			t.insertAtCursor("\t")
		}
		return true
	}
	return false
}

// afterCaretMove keeps the caret visible and repaints after a pure movement.
func (t *TextArea) afterCaretMove() {
	t.ensureCursorVisible()
	t.cache.Invalidate()
	t.MarkForRedraw()
}

// moveHorizontally moves the caret one rune, or one WORD when word is set. Word motion is computed
// on the whole document, so it walks across line ends instead of stopping at them.
func (t *TextArea) moveHorizontally(left, word bool) {
	off := t.cursorToOffset(t.CursorLine, t.CursorCol)
	switch {
	case word && left:
		off = wordLeftOffset(t.Value, off)
	case word:
		off = wordRightOffset(t.Value, off)
	case left:
		off = runeStartBefore(t.Value, off)
	default:
		off = runeEndAfter(t.Value, off)
	}
	t.CursorLine, t.CursorCol = t.offsetToCursor(off)
}

// moveVertically moves the caret by lines, aiming at the column it started from (goalCol) rather than
// clamping and forgetting it. Without the goal, moving down through a short line and back up left the
// caret further left than it started — the drift every other editor avoids.
func (t *TextArea) moveVertically(deltaLines int, extend bool) {
	if extend && !t.hasSelection() {
		t.selectionAnchor = t.cursorToOffset(t.CursorLine, t.CursorCol)
	}
	if t.goalCol < 0 {
		t.goalCol = t.CursorCol
	}
	cl := t.contentSlice()
	if len(cl) == 0 {
		return
	}
	line := t.CursorLine + deltaLines
	if line < 0 {
		line = 0
	}
	if line >= len(cl) {
		line = len(cl) - 1
	}
	t.CursorLine = line
	col := t.goalCol
	if col > len(cl[line]) {
		col = len(cl[line])
	}
	t.CursorCol = col
	if extend {
		t.updateSelectionFromAnchor()
	} else {
		t.clearSelection()
	}
	t.undoKind = undoKindNone
	t.afterCaretMove()
}

// visibleLines is how many text lines fit in the frame; a page is one screen less a line of overlap.
func (t *TextArea) visibleLines() int {
	lh := t.lineHeight()
	if lh <= 0 {
		return 1
	}
	n := int(t.FrameRect().Height/lh) - 1
	if n < 1 {
		n = 1
	}
	return n
}

// deleteRange removes Value[lo:hi] as one grouped edit. Every delete path goes through it, so the
// edits are rune-aware and one undo step.
func (t *TextArea) deleteRange(lo, hi int) {
	if lo < 0 {
		lo = 0
	}
	if hi > len(t.Value) {
		hi = len(t.Value)
	}
	if lo >= hi {
		return
	}
	t.saveStateGrouped(undoKindDelete)
	t.Value = t.Value[:lo] + t.Value[hi:]
	t.rebuildLines()
	t.CursorLine, t.CursorCol = t.offsetToCursor(lo)
	t.goalCol = -1
	t.clearSelection()
	t.ensureCursorVisible()
	t.cache.Invalidate()
	t.MarkForRedraw()
	if t.OnTextChanged != nil {
		t.OnTextChanged(t.Value)
	}
}

func (t *TextArea) OnFocus() {
	t.Focused = true
	t.startCursorBlink()
	t.cache.Invalidate()
	t.MarkForRedraw()
}

func (t *TextArea) OnBlur() {
	t.cancelComposition()
	t.Focused = false
	t.stopCursorBlink()
	t.clearSelection()
	t.cache.Invalidate()
	t.MarkForRedraw()
}

func (t *TextArea) startCursorBlink() {
	t.CursorVisible = true
	if t.CursorTimer != nil {
		t.CursorTimer.Stop()
	}
	t.CursorTimer = time.AfterFunc(500*time.Millisecond, func() {
		t.CursorVisible = !t.CursorVisible
		platform.InvokeTask(func() {
			t.MarkForRedraw()
			if t.Focused {
				t.startCursorBlink()
			}
		})
	})
}

func (t *TextArea) stopCursorBlink() {
	if t.CursorTimer != nil {
		t.CursorTimer.Stop()
		t.CursorTimer = nil
	}
}

// ── Context menu ────────────────────────────────────────────────

func (t *TextArea) showContextMenu(x, y float32) {
	items := []MenuItem{
		{Label: "Cut", Enabled: !t.ReadOnly, Shortcut: "Ctrl+X", Action: func() { t.Cut() }},
		{Label: "Copy", Enabled: true, Shortcut: "Ctrl+C", Action: func() { t.Copy() }},
		{Label: "Paste", Enabled: !t.ReadOnly, Shortcut: "Ctrl+V", Action: func() { t.Paste() }},
		{IsSep: true},
		{Label: "Select All", Enabled: true, Shortcut: "Ctrl+A", Action: func() { t.SelectAll() }},
	}
	if t.ctxMenu == nil {
		t.ctxMenu = NewContextMenu()
	}
	t.ctxMenu.SetItems(items)
	if win := t.Window(); win != nil {
		t.ctxMenu.SetOnDismiss(func() {
			win.DismissPopupContextMenu()
		})
		winX, winY := t.FromLocal(x, y)
		win.ShowPopupContextMenu(t.ctxMenu, winX, winY)
	}
	t.MarkForRedraw()
}

func (t *TextArea) textStartX() float32 {
	if !t.ShowLineNumbers {
		return 4
	}
	n := len(t.contentSlice())
	digits := 1
	if n > 0 {
		for m := n; m > 0; m /= 10 {
			digits++
		}
	}
	return 4 + float32(digits)*8 + 12
}

// charIndexAtX returns the byte column offset within the given display line
// closest to the local x coordinate. Uses rune-aware measurement.
func (t *TextArea) charIndexAtX(displayLine int, x float32) int {
	ln := t.lines()
	if displayLine < 0 || displayLine >= len(ln) {
		return 0
	}
	line := ln[displayLine]
	textStart := t.textStartX()
	if x <= textStart {
		return 0
	}
	// Shared with the markdown view and the code editor (textlayout.go).
	return columnAtX(line, t.font(), x-textStart)
}

// CursorAt reports the I-beam over the editable text; the line-number gutter
// and the scrollbar keep the default arrow.
func (t *TextArea) CursorAt(x, y float32) *platform.Cursor {
	if !t.Enabled() || !t.Visible() {
		return nil
	}
	frame := t.FrameRect()
	gutterW := t.gutterWidthFor(len(t.lines()))
	scrollbarW := float32(0)
	if t.ShowScrollBar {
		scrollbarW = 14
	}
	if x < 4+gutterW || x > frame.Width-4-scrollbarW || y < 4 || y > frame.Height-4 {
		return nil
	}
	return platform.TextCursor()
}

// gutterWidthFor returns the width of the line-number gutter for a document
// of n numbered lines. Paint, availableTextWidth and CursorAt all measure it
// through here so the painted text area and the cursor zone agree.
func (t *TextArea) gutterWidthFor(n int) float32 {
	if !t.ShowLineNumbers {
		return 0
	}
	digits := 1
	for ; n > 0; n /= 10 {
		digits++
	}
	w := float32(digits)*8 + 12
	if w < 24 {
		w = 24
	}
	return w
}

func (t *TextArea) OnMouseDown(x, y float32, button int) bool {
	if !t.Enabled() {
		return false
	}
	t.cancelComposition()
	if button == 1 {
		t.showContextMenu(x, y)
		return true
	}
	// Scrollbar click
	if t.ShowScrollBar {
		if frame := t.FrameRect(); t.host.HandleMouseDown(frame.Width, frame.Height, x, y, button) {
			return true
		}
	}
	t.Focused = true
	t.startCursorBlink()
	lh := t.lineHeight()
	adjustedY := y + t.ScrollOffset
	clickedDisplayLine := int(adjustedY / lh)
	ln := t.lines()
	if clickedDisplayLine >= len(ln) {
		clickedDisplayLine = len(ln) - 1
	}
	if clickedDisplayLine < 0 {
		clickedDisplayLine = 0
	}
	clickedLine, colOffset := t.displayToContent(clickedDisplayLine)
	clickedCol := colOffset + t.charIndexAtX(clickedDisplayLine, x)

	// Detect double/triple click
	now := time.Now()
	if now.Sub(t.lastClickTime) < 350*time.Millisecond &&
		clickedLine == t.lastClickLine && clickedCol == t.lastClickCol {
		t.clickCount++
	} else {
		t.clickCount = 1
	}
	t.lastClickTime = now
	t.lastClickLine = clickedLine
	t.lastClickCol = clickedCol

	shift := false
	if w := t.Window(); w != nil {
		shift = w.Modifiers().ShiftDown()
	}
	t.applyClick(clickedLine, clickedCol, shift)
	t.ensureCursorVisible()
	t.cache.Invalidate()
	t.MarkForRedraw()
	return true
}

// applyClick turns a press at a document position into a caret or a selection. It is separate from
// OnMouseDown so the gesture rules can be tested without guessing where pixel x lands.
func (t *TextArea) applyClick(line, col int, shift bool) {
	switch {
	case t.clickCount == 2:
		// Double-click selects the word — and the drag that usually follows selects whole words.
		t.selectWordAt(line, col)
		t.isSelecting = true
		t.isSelectingWord = true
		return
	case t.clickCount >= 3:
		// Triple-click: select the whole line.
		cl := t.contentSlice()
		if line < len(cl) {
			startOff := t.cursorToOffset(line, 0)
			endOff := t.cursorToOffset(line, len(cl[line]))
			t.SelectStart = startOff
			t.SelectEnd = endOff
			t.CursorLine = line
			t.CursorCol = len(cl[line])
			t.selectionAnchor = startOff
		}
		t.isSelecting = false
		t.isSelectingWord = false
		return
	}

	if shift {
		// Shift+click EXTENDS the selection from the caret instead of starting a new one.
		if !t.hasSelection() {
			t.selectionAnchor = t.cursorToOffset(t.CursorLine, t.CursorCol)
		}
		t.CursorLine, t.CursorCol = line, col
		t.updateSelectionFromAnchor()
		t.isSelecting = true
		t.isSelectingWord = false
		t.goalCol = -1
		return
	}

	// A press INSIDE an existing selection does not clear it: that is the gesture that starts a
	// drag-and-drop. A release without a drag collapses the caret there.
	offset := t.cursorToOffset(line, col)
	if t.hasSelection() && shift == false && offset >= t.SelectStart && offset <= t.SelectEnd {
		t.dragArmed = true
		t.dragText = t.Value[t.SelectStart:t.SelectEnd]
		t.pressLine, t.pressCol = line, col
		t.isSelecting = false
		t.isSelectingWord = false
		return
	}
	t.dragArmed = false

	t.CursorLine = line
	t.CursorCol = col
	t.SelectStart = offset
	t.SelectEnd = offset
	t.selectionAnchor = offset
	t.isSelecting = true
	t.isSelectingWord = false
	t.goalCol = -1
}

func (t *TextArea) OnMouseUp(x, y float32, button int) bool {
	t.isSelecting = false
	t.isSelectingWord = false
	if t.dragArmed {
		// No drag happened: a click inside the selection collapses the caret there.
		t.dragArmed = false
		t.CursorLine, t.CursorCol = t.pressLine, t.pressCol
		t.clearSelection()
		t.goalCol = -1
		t.ensureCursorVisible()
		t.cache.Invalidate()
		t.MarkForRedraw()
	}
	t.dragText = ""
	if t.ShowScrollBar {
		t.VScroll.OnMouseUp(x, y, button)
	}
	return true
}

func (t *TextArea) OnMouseMove(x, y float32) {
	if !t.Enabled() {
		return
	}
	if t.ShowScrollBar {
		t.host.UpdateHover(x, y) // overlay pills reveal on strip hover
	}
	if t.ShowScrollBar && t.VScroll.dragging {
		t.host.HandleMouseMove(x, y)
		return
	}
	if !t.isSelecting {
		return
	}
	lh := t.lineHeight()
	adjustedY := y + t.ScrollOffset
	hoverDisplayLine := int(adjustedY / lh)
	ln := t.lines()
	if hoverDisplayLine >= len(ln) {
		hoverDisplayLine = len(ln) - 1
	}
	if hoverDisplayLine < 0 {
		hoverDisplayLine = 0
	}
	hoverLine, colOffset := t.displayToContent(hoverDisplayLine)
	hoverCol := colOffset + t.charIndexAtX(hoverDisplayLine, x)
	if t.isSelectingWord {
		t.extendWordDragTo(t.cursorToOffset(hoverLine, hoverCol))
		t.ensureCursorVisible()
		t.cache.Invalidate()
		t.MarkForRedraw()
		return
	}
	t.CursorLine = hoverLine
	t.CursorCol = hoverCol
	offset := t.cursorToOffset(hoverLine, hoverCol)
	if offset < t.selectionAnchor {
		t.SelectStart = offset
		t.SelectEnd = t.selectionAnchor
	} else {
		t.SelectStart = t.selectionAnchor
		t.SelectEnd = offset
	}
	t.ensureCursorVisible()
	t.cache.Invalidate()
	t.MarkForRedraw()
}

// extendWordDragTo grows the double-click selection by WHOLE WORDS, anchored on the word that was
// clicked. offset is a byte offset into Value.
func (t *TextArea) extendWordDragTo(offset int) {
	lo, hi := t.wordAnchorStart, t.wordAnchorEnd
	if offset < lo {
		l, _ := wordSpanAt(t.Value, offset)
		lo = l
		t.CursorLine, t.CursorCol = t.offsetToCursor(lo)
	} else {
		_, h := wordSpanAt(t.Value, offset)
		hi = h
		t.CursorLine, t.CursorCol = t.offsetToCursor(hi)
	}
	t.SelectStart, t.SelectEnd = lo, hi
}

// ── Drag and drop ───────────────────────────────────────────────────────

var _ meui.DragSource = (*TextArea)(nil)
var _ meui.DropTarget = (*TextArea)(nil)

// OnDragStart starts a drag when the press landed inside the selection.
func (t *TextArea) OnDragStart(x, y float32) *meui.DragData {
	if !t.dragArmed || t.dragText == "" {
		return nil
	}
	text := t.dragText
	t.dragArmed = false
	return &meui.DragData{Type: "text", Source: meui.Widget(t), Data: text}
}

func (t *TextArea) OnDragEnter(data *meui.DragData, x, y float32) {
	if t.dropActive {
		return
	}
	t.dropActive = true
	t.dropMoved = false
	if data != nil && data.Source == meui.Widget(t) {
		// The SOURCE keeps its selection and its caret.
		return
	}
	t.dropMoved = true
	t.dropPrevLine, t.dropPrevCol = t.CursorLine, t.CursorCol
	t.moveCaretToDrop(x, y)
}

func (t *TextArea) OnDragOver(data *meui.DragData, x, y float32) {
	if t.dropActive && t.dropMoved {
		t.moveCaretToDrop(x, y)
	}
}

func (t *TextArea) OnDragLeave(data *meui.DragData) {
	if !t.dropActive {
		return
	}
	t.dropActive = false
	if !t.dropMoved {
		return
	}
	t.dropMoved = false
	t.CursorLine, t.CursorCol = t.dropPrevLine, t.dropPrevCol
	t.clearSelection()
	t.cache.Invalidate()
	t.MarkForRedraw()
}

// OnDragDrop inserts the dragged text at the pointer. A drop into the area it came from MOVES it
// (the source selection is cut first), and a drop inside that selection is a no-op.
func (t *TextArea) OnDragDrop(data *meui.DragData, x, y float32) {
	text, _ := data.Data.(string)
	t.dropActive = false
	t.dropMoved = false
	t.isSelecting = false
	t.isSelectingWord = false
	t.dragArmed = false
	t.dragText = ""
	if text == "" {
		return
	}
	line, col := t.caretAtLocal(x, y)
	dropOff := t.cursorToOffset(line, col)

	if data.Source == meui.Widget(t) && t.hasSelection() {
		lo, hi := t.selectionBounds()
		if dropOff >= lo && dropOff <= hi {
			t.MarkForRedraw()
			return
		}
		if dropOff > hi {
			dropOff -= hi - lo
		}
		t.saveState()
		t.Value = t.Value[:lo] + t.Value[hi:]
		t.rebuildLines()
		t.CursorLine, t.CursorCol = t.offsetToCursor(dropOff)
		t.clearSelection()
		t.insertRaw(text)
		t.cache.Invalidate()
		t.MarkForRedraw()
		return
	}
	if data.Source != nil && data.Source != meui.Widget(t) {
		// A drop into ANOTHER text area is a MOVE: the source gives up the text it lent the drag.
		if r, ok := data.Source.(interface{ RemoveDraggedText(*meui.DragData) }); ok {
			r.RemoveDraggedText(data)
		}
	}
	t.CursorLine, t.CursorCol = t.offsetToCursor(dropOff)
	t.clearSelection()
	t.insertAtCursor(text)
	t.cache.Invalidate()
	t.MarkForRedraw()
}

// RemoveDraggedText deletes the selection a drag borrowed, so a drop elsewhere is a MOVE.
func (t *TextArea) RemoveDraggedText(data *meui.DragData) {
	if data == nil || data.Source != meui.Widget(t) || !t.hasSelection() {
		return
	}
	t.deleteSelection()
}

// moveCaretToDrop puts the caret where a drop would land, without touching the selection.
func (t *TextArea) moveCaretToDrop(x, y float32) {
	line, col := t.caretAtLocal(x, y)
	t.CursorLine, t.CursorCol = line, col
	t.goalCol = -1
	t.clearSelection()
	t.ensureCursorVisible()
	t.cache.Invalidate()
	t.MarkForRedraw()
}

// caretAtLocal maps a CONTENT-space point to a document position, through the scroll offset, exactly
// as a press does.
func (t *TextArea) caretAtLocal(x, y float32) (int, int) {
	pt := t.PointFromRoot(x, y)
	lh := t.lineHeight()
	adjustedY := pt.Y + t.ScrollOffset
	disp := 0
	if lh > 0 {
		disp = int(adjustedY / lh)
	}
	lines := t.lines()
	if disp >= len(lines) {
		disp = len(lines) - 1
	}
	if disp < 0 {
		disp = 0
	}
	line, colOffset := t.displayToContent(disp)
	return line, colOffset + t.charIndexAtX(disp, pt.X)
}

func (t *TextArea) OnMouseWheel(x, y float32, dx, dy float32) bool {
	if !t.Enabled() || dy == 0 {
		return false
	}
	frame := t.FrameRect()
	totalHeight := float32(len(t.lines())) * t.lineHeight()
	t.host.SetExtents(0, totalHeight, 0, frame.Height-8)
	if !t.host.HandleWheel(dx, dy) {
		return false
	}
	_, t.ScrollOffset = t.host.Offsets()
	t.cache.Invalidate()
	t.MarkForRedraw()
	return true
}

func (t *TextArea) OnCharacter(ch rune) {
	if t.composing {
		t.cancelComposition()
	}
	if !t.Enabled() || !t.Focused || t.ReadOnly {
		return
	}
	t.insertAtCursor(string(ch))
	t.cache.Invalidate()
}

func (t *TextArea) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "text", "Text":
		if v, ok := value.(string); ok {
			t.SetText(v)
		}
	case "placeholder", "Placeholder":
		if v, ok := value.(string); ok {
			t.SetPlaceholder(v)
		}
	case "showLineNumbers", "lineNumbers":
		if v, ok := value.(bool); ok {
			t.SetShowLineNumbers(v)
		}
	case "showScrollBar", "scrollbar":
		if v, ok := value.(bool); ok {
			t.SetShowScrollBar(v)
		}
	case "wordWrap", "wordWrapEnabled":
		if v, ok := value.(bool); ok {
			t.SetWordWrap(v)
		}
	case "readOnly":
		if v, ok := value.(bool); ok {
			t.SetReadOnly(v)
		}
	case "maxLength", "maximumLength":
		switch v := value.(type) {
		case float64:
			t.MaxLength = int(v)
		case int:
			t.MaxLength = v
		}
	}
}

func (t *TextArea) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	fr := t.FrameRect()
	if fr.Width > 0 || fr.Height > 0 {
		return fr.Width, fr.Height
	}
	return 200, 80
}

func (t *TextArea) MinimumDimensions() (float32, float32) {
	return 100, 60
}

func (t *TextArea) Dispose() {
	t.stopCursorBlink()
	t.OnTextChanged = nil
	t.BaseWidget.Dispose()
}

func (t *TextArea) Focusable() bool {
	return t.Enabled()
}
