package widgets

import (
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

type TextField struct {
	meui.BaseWidget
	*theme.TextFieldTheme
	Text         string
	Placeholder  string
	EchoChar     rune
	ShowPassword bool
	MaxLength    int
	ReadOnly     bool   // text cannot be edited; selection/copy allowed
	HAlign       string // "left" (default), "center", "right"
	// Overwrite means INSERT mode's opposite: the character under the caret is replaced instead of
	// pushed right. A plain Insert key toggles it, as in every editor that has the mode.
	Overwrite     bool
	Focused       bool
	CursorPos     int // byte index into t.Text
	SelectStart   int // byte index
	SelectEnd     int // byte index
	CursorVisible bool
	CursorTimer   *time.Timer
	OnTextChanged func(string)
	OnSubmit      func(string)
	OnFocusGained func()
	OnFocusLost   func()
	selecting     bool
	// selectingWord is a drag that began on a DOUBLE click: the selection grows by whole words, as
	// every other editor does. wordAnchor* remember the word the double click landed on.
	selectingWord   bool
	wordAnchorStart int
	wordAnchorEnd   int
	// Drag-and-drop. dragArmed is "the press landed inside the existing selection", which is the only
	// way a text drag STARTS; dropActive/caretState are the target's feedback while a drag is over it.
	dragArmed   bool
	dragText    string
	pressPos    int
	dropActive  bool
	dropMoved   bool
	dropPrevPos int
	// IME composition. The preedit lives in Text while composing so the layout, the caret and the
	// scroll offset all stay honest; it is NOT undoable and NOT announced through OnTextChanged,
	// because it has not been committed. The commit (OnCharacter) removes it first.
	composing     bool
	composeStart  int
	composeEnd    int
	scrollOffsetX float32
	undoStack     []textFieldState
	redoStack     []textFieldState
	maxUndo       int
	// undoKind/undoTime coalesce a burst of one kind of edit into ONE undo step: typing a word is one
	// undo, not one per character.
	undoKind        int
	undoTime        time.Time
	ValidationState string
	LeadingIcon     string
	TrailingIcon    string
	leadingSVG      *meui.SVGDocument
	trailingSVG     *meui.SVGDocument
	eyeSVG          *meui.SVGDocument
	selectionAnchor int // byte index
	ctxMenu         *ContextMenu
	textCache       *graphics.Text
	textCacheVer    uint64
	clickCount      int // for double/triple click
	lastClickTime   time.Time
	lastClickPos    int
}

type textFieldState struct {
	text            string
	cursorPos       int
	selectStart     int
	selectEnd       int
	selectionAnchor int
}

func NewTextField() *TextField {
	tf := &TextField{
		BaseWidget:     *meui.NewBaseWidget(),
		TextFieldTheme: theme.DefaultTextFieldTheme,
		maxUndo:        50,
	}
	tf.SetTabIndex(0)
	tf.CursorPos = 0
	tf.SelectStart = 0
	tf.SelectEnd = 0
	tf.selectionAnchor = 0
	if doc, err := meui.LoadSVGByName("eye"); err == nil {
		tf.eyeSVG = doc
	}
	return tf
}

func (t *TextField) SetEchoChar(ch rune) *TextField {
	t.EchoChar = ch
	t.textCache = nil
	t.MarkForRedraw()
	return t
}

func (t *TextField) SetText(text string) *TextField {
	if t.MaxLength > 0 && utf8.RuneCountInString(text) > t.MaxLength {
		text = string([]rune(text)[:t.MaxLength])
	}
	t.Text = text
	t.textCache = nil
	if t.CursorPos > len(t.Text) {
		t.CursorPos = len(t.Text)
	}
	if t.SelectStart > len(t.Text) {
		t.SelectStart = len(t.Text)
	}
	t.SelectStart = 0
	t.SelectEnd = 0
	t.MarkForRedraw()
	return t
}

func (t *TextField) SetPlaceholder(text string) *TextField {
	t.Placeholder = text
	t.MarkForRedraw()
	return t
}

func (t *TextField) SetValidationState(state string) *TextField {
	t.ValidationState = state
	t.MarkForRedraw()
	return t
}

// SetReadOnly disables text editing (selection and copy still work), like
// QLineEdit.setReadOnly.
func (t *TextField) SetReadOnly(readOnly bool) *TextField {
	t.ReadOnly = readOnly
	t.MarkForRedraw()
	return t
}

func (t *TextField) IsReadOnly() bool { return t.ReadOnly }

// Clear removes all text (respecting read-only mode).
func (t *TextField) Clear() {
	if t.ReadOnly {
		return
	}
	if t.Text == "" {
		return
	}
	t.SetText("")
	t.CursorPos = 0
	t.SelectStart = 0
	t.SelectEnd = 0
	t.selectionAnchor = 0
	if t.OnTextChanged != nil {
		t.OnTextChanged("")
	}
}

// SetSelection selects [start, end) in byte indices (clamped). Also moves the
// cursor to end, like QLineEdit.setSelection semantics.
func (t *TextField) SetSelection(start, end int) {
	max := len(t.Text)
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
	t.CursorPos = end
	t.selectionAnchor = start
	t.ensureCursorVisible()
	t.MarkForRedraw()
}

// GetSelectedText returns the selected text, or "" if nothing is selected.
func (t *TextField) GetSelectedText() string {
	if !t.hasSelection() {
		return ""
	}
	start, end := t.selectionBounds()
	return t.Text[start:end]
}

// SelectionLength returns the selected byte length.
func (t *TextField) SelectionLength() int {
	if !t.hasSelection() {
		return 0
	}
	start, end := t.selectionBounds()
	return end - start
}

// SetHAlign sets horizontal text alignment: "left", "center" or "right".
func (t *TextField) SetHAlign(align string) *TextField {
	t.HAlign = align
	t.MarkForRedraw()
	return t
}

func (t *TextField) SetMaxLength(max int) *TextField {
	t.MaxLength = max
	if max > 0 && utf8.RuneCountInString(t.Text) > max {
		t.Text = string([]rune(t.Text)[:max])
		t.textCache = nil
		if t.CursorPos > len(t.Text) {
			t.CursorPos = len(t.Text)
		}
		t.MarkForRedraw()
	}
	return t
}

func (t *TextField) SetLeadingIcon(icon string) *TextField {
	t.LeadingIcon = icon
	t.leadingSVG = nil
	if icon != "" && !isEmoji(icon) {
		if doc, err := meui.LoadSVGByName(icon); err == nil {
			t.leadingSVG = doc
		}
	}
	t.MarkForRedraw()
	return t
}

func (t *TextField) SetTrailingIcon(icon string) *TextField {
	t.TrailingIcon = icon
	t.trailingSVG = nil
	if icon != "" && !isEmoji(icon) {
		if doc, err := meui.LoadSVGByName(icon); err == nil {
			t.trailingSVG = doc
		}
	}
	t.MarkForRedraw()
	return t
}

func isEmoji(s string) bool {
	if len(s) == 0 {
		return false
	}
	r := []rune(s)[0]
	return r > 0x1F000 || (r >= 0x2600 && r <= 0x27BF)
}

func (t *TextField) SetOnTextChanged(handler func(string)) *TextField {
	t.OnTextChanged = handler
	return t
}

func (t *TextField) SetOnSubmit(handler func(string)) *TextField {
	t.OnSubmit = handler
	return t
}

func (t *TextField) pushUndo() {
	t.undoKind = undoKindNone
	t.undoStack = append(t.undoStack, textFieldState{
		text:            t.Text,
		cursorPos:       t.CursorPos,
		selectStart:     t.SelectStart,
		selectEnd:       t.SelectEnd,
		selectionAnchor: t.selectionAnchor,
	})
	if len(t.undoStack) > t.maxUndo {
		t.undoStack = t.undoStack[1:]
	}
	t.redoStack = nil
}

func (t *TextField) Undo() bool {
	t.undoKind = undoKindNone
	if len(t.undoStack) == 0 {
		return false
	}
	current := textFieldState{
		text:            t.Text,
		cursorPos:       t.CursorPos,
		selectStart:     t.SelectStart,
		selectEnd:       t.SelectEnd,
		selectionAnchor: t.selectionAnchor,
	}
	t.redoStack = append(t.redoStack, current)
	prev := t.undoStack[len(t.undoStack)-1]
	t.undoStack = t.undoStack[:len(t.undoStack)-1]
	t.Text = prev.text
	t.CursorPos = prev.cursorPos
	t.SelectStart = prev.selectStart
	t.SelectEnd = prev.selectEnd
	t.selectionAnchor = prev.selectionAnchor
	t.textCache = nil
	t.MarkForRedraw()
	if t.OnTextChanged != nil {
		t.OnTextChanged(t.Text)
	}
	return true
}

func (t *TextField) Redo() bool {
	t.undoKind = undoKindNone
	if len(t.redoStack) == 0 {
		return false
	}
	current := textFieldState{
		text:            t.Text,
		cursorPos:       t.CursorPos,
		selectStart:     t.SelectStart,
		selectEnd:       t.SelectEnd,
		selectionAnchor: t.selectionAnchor,
	}
	t.undoStack = append(t.undoStack, current)
	next := t.redoStack[len(t.redoStack)-1]
	t.redoStack = t.redoStack[:len(t.redoStack)-1]
	t.Text = next.text
	t.CursorPos = next.cursorPos
	t.SelectStart = next.selectStart
	t.SelectEnd = next.selectEnd
	t.selectionAnchor = next.selectionAnchor
	t.textCache = nil
	t.MarkForRedraw()
	if t.OnTextChanged != nil {
		t.OnTextChanged(t.Text)
	}
	return true
}

// pushUndoGrouped records a state unless the same KIND of edit happened a moment ago — a typing
// burst, or a held Backspace, is one undo step.
func (t *TextField) pushUndoGrouped(kind int) {
	now := time.Now()
	if kind != undoKindNone && kind == t.undoKind && now.Sub(t.undoTime) < undoGroupWindow {
		t.undoTime = now
		t.redoStack = nil
		return
	}
	t.pushUndo()
	t.undoKind = kind
	t.undoTime = now
}

func (t *TextField) ClearUndo() {
	t.undoStack = nil
	t.redoStack = nil
}

// ── Rune-aware cursor movement helpers (byte indices) ──

func (t *TextField) prevRuneStart(pos int) int {
	if pos <= 0 {
		return 0
	}
	_, size := utf8.DecodeLastRuneInString(t.Text[:pos])
	return pos - size
}

func (t *TextField) nextRuneStart(pos int) int {
	if pos >= len(t.Text) {
		return len(t.Text)
	}
	_, size := utf8.DecodeRuneInString(t.Text[pos:])
	return pos + size
}

// prevWordStart returns the byte position of the start of the previous word, using the shared
// class-aware rule (space / word / punctuation) so movement and double-click selection agree.
func (t *TextField) prevWordStart(pos int) int { return wordLeftOffset(t.Text, pos) }

// nextWordStart returns the end of the word at (or after) pos, plus its trailing whitespace.
func (t *TextField) nextWordStart(pos int) int { return wordRightOffset(t.Text, pos) }

// runeAt returns the rune that starts at or overlaps the byte position pos.
func runeAt(s string, pos int) rune {
	if pos < 0 || pos >= len(s) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(s[pos:])
	return r
}

// ── Selection helpers ──

func (t *TextField) hasSelection() bool {
	return t.SelectStart != t.SelectEnd
}

func (t *TextField) selectionBounds() (int, int) {
	s, e := t.SelectStart, t.SelectEnd
	if s > e {
		s, e = e, s
	}
	return s, e
}

func (t *TextField) clearSelection() {
	t.SelectStart = t.CursorPos
	t.SelectEnd = t.CursorPos
}

// echoByteOffset maps a byte offset in t.Text to a byte offset in displayText (echo mode).
func (t *TextField) echoByteOffset(bytePos int) int {
	return utf8.RuneCountInString(t.Text[:bytePos])
}

// echoDisplayText returns the displayed text in echo mode (all replaced with EchoChar).
func (t *TextField) echoDisplayText() string {
	return strings.Repeat(string(t.EchoChar), utf8.RuneCountInString(t.Text))
}

// Icon metrics shared by the field's layout, painting and cursor geometry.
const (
	textFieldIconSize    = float32(14)
	textFieldIconPadding = float32(3)
)

// textInsets returns the padding the text area leaves for the field's icons:
// the leading/trailing icon slots and the 3px vertical inset. Paint,
// ensureCursorVisible and CursorAt all measure the text area this way.
func (t *TextField) textInsets() (left, right, top, bottom float32) {
	left, right, top, bottom = 4, 4, 3, 3
	if t.LeadingIcon != "" {
		left += textFieldIconSize + textFieldIconPadding
	}
	if t.TrailingIcon != "" {
		right += textFieldIconSize + textFieldIconPadding
	}
	if t.EchoChar != 0 {
		right += textFieldIconSize + textFieldIconPadding
	}
	return
}

// CursorAt reports the I-beam over the editable text and leaves the icons to
// the default arrow.
func (t *TextField) CursorAt(x, y float32) *platform.Cursor {
	if !t.Enabled() || !t.Visible() {
		return nil
	}
	left, right, top, bottom := t.textInsets()
	fr := t.FrameRect()
	if x < left || x > fr.Width-right || y < top || y > fr.Height-bottom {
		return nil
	}
	return platform.TextCursor()
}

func (t *TextField) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !t.Visible() {
		return
	}

	canvas.Save()

	disabled := !t.Enabled()

	leftOffset, rightOffset, topOffset, bottomOffset := t.textInsets()

	innerRect := graphics.Rect{
		X:      rect.X + leftOffset,
		Y:      rect.Y + topOffset,
		Width:  rect.Width - leftOffset - rightOffset,
		Height: rect.Height - topOffset - bottomOffset,
	}

	bgPaint := graphics.NewPaint()
	if disabled {
		bgPaint.SetColor(theme.DefaultTheme.DisabledBg)
	} else if t.Focused {
		bgPaint.SetColor(t.TextFieldTheme.BackgroundFocused)
	} else {
		bgPaint.SetColor(t.TextFieldTheme.BackgroundInk)
	}
	canvas.DrawRoundedRect(rect, t.TextFieldTheme.CornerRadius, t.TextFieldTheme.CornerRadius, bgPaint)

	borderPaint := graphics.NewPaint()
	borderPaint.SetStyle(paintstyle.Stroke)
	borderPaint.SetStrokeWidth(theme.BorderStrokeWidth)
	if disabled {
		borderPaint.SetColor(theme.DefaultTheme.DisabledBorder)
	} else if t.ValidationState == "error" {
		borderPaint.SetColor(theme.DefaultTheme.Error)
	} else if t.ValidationState == "warning" {
		borderPaint.SetColor(theme.DefaultTheme.Warning)
	} else if t.ValidationState == "success" {
		borderPaint.SetColor(theme.DefaultTheme.Success)
	} else if t.Focused {
		borderPaint.SetColor(t.TextFieldTheme.EdgeFocused)
	} else if t.BaseWidget.Hovered() {
		borderPaint.SetColor(t.TextFieldTheme.EdgeHovered)
	} else {
		borderPaint.SetColor(t.TextFieldTheme.EdgeInk)
	}
	canvas.DrawRoundedRect(rect, t.TextFieldTheme.CornerRadius, t.TextFieldTheme.CornerRadius, borderPaint)
	// Fluent emphasises the bottom edge (1px resting, 2px accent on
	// focus); other platforms leave both widths zero.
	if !disabled && t.TextFieldTheme.BottomEdgeWidth > 0 && t.ValidationState == "" {
		bw := t.TextFieldTheme.BottomEdgeWidth
		bc := t.TextFieldTheme.BottomEdgeInk
		if t.Focused && t.TextFieldTheme.BottomFocusWidth > 0 {
			bw = t.TextFieldTheme.BottomFocusWidth
			bc = t.TextFieldTheme.EdgeFocused
		}
		if bc != 0 {
			bp := graphics.NewPaint()
			bp.SetStyle(paintstyle.Stroke)
			bp.SetStrokeWidth(bw)
			bp.SetColor(bc)
			y := rect.Y + rect.Height - bw/2
			canvas.DrawLine(rect.X, y, rect.X+rect.Width, y, bp)
		}
	}
	if t.Focused && !disabled {
		// A field that is wrong AND focused shows the ring in the state's ink, so focus
		// never masks the error.
		ringInk := graphics.Color(0)
		switch t.ValidationState {
		case "error":
			ringInk = theme.DefaultTheme.Error
		case "warning":
			ringInk = theme.DefaultTheme.Warning
		case "success":
			ringInk = theme.DefaultTheme.Success
		}
		if ringInk != 0 {
			meui.DrawFocusRingInk(canvas, rect, t.TextFieldTheme.CornerRadius, ringInk)
		} else {
			drawFocusIndicator(canvas, rect, t.TextFieldTheme.CornerRadius)
		}
	}

	// Draw leading icon
	if t.LeadingIcon != "" {
		iconY := rect.Y + (rect.Height-textFieldIconSize)/2
		if t.leadingSVG != nil {
			meui.DrawSVG(canvas, t.leadingSVG, rect.X+4, iconY, textFieldIconSize, textFieldIconSize)
		} else {
			iconFont := theme.DefaultTheme.BaseFont
			if iconFont == nil {
				iconFont = graphics.SystemFont
			}
			iconPaint := graphics.NewPaint()
			iconPaint.SetColor(t.TextFieldTheme.TextInk)
			iconFontY := rect.Y + (rect.Height-iconFont.LineHeight())/2 + iconFont.Baseline()
			it := graphics.NewText(t.LeadingIcon, &graphics.TextDecoration{Font: iconFont, OnBackgroundInk: iconPaint.Color()})
			it.Draw(canvas, rect.X+4, iconFontY)
		}
	}

	// Draw trailing icon
	iconOffset := float32(0)
	if t.TrailingIcon != "" {
		iconY := rect.Y + (rect.Height-textFieldIconSize)/2
		if t.trailingSVG != nil {
			meui.DrawSVG(canvas, t.trailingSVG, rect.X+rect.Width-textFieldIconSize-4, iconY, textFieldIconSize, textFieldIconSize)
		} else {
			iconFont := theme.DefaultTheme.BaseFont
			if iconFont == nil {
				iconFont = graphics.SystemFont
			}
			iconPaint := graphics.NewPaint()
			iconPaint.SetColor(t.TextFieldTheme.TextInk)
			iconW := iconFont.SimpleWidth(t.TrailingIcon)
			iconFontY := rect.Y + (rect.Height-iconFont.LineHeight())/2 + iconFont.Baseline()
			itt := graphics.NewText(t.TrailingIcon, &graphics.TextDecoration{Font: iconFont, OnBackgroundInk: iconPaint.Color()})
			itt.Draw(canvas, rect.X+rect.Width-iconW-4, iconFontY)
		}
		iconOffset = 18
	}

	// Draw eye icon for password fields
	if t.EchoChar != 0 && t.eyeSVG != nil {
		iconX := rect.X + rect.Width - textFieldIconSize - 4 - iconOffset
		iconY := rect.Y + (rect.Height-textFieldIconSize)/2
		meui.DrawSVG(canvas, t.eyeSVG, iconX, iconY, textFieldIconSize, textFieldIconSize)
	}

	// Clip to inner rect so text doesn't overflow
	canvas.ClipRect(innerRect, pathop.Intersect, false)

	displayText := t.displayText()
	isPlaceholder := false
	if displayText == "" && t.Placeholder != "" {
		displayText = t.Placeholder
		isPlaceholder = true
	}

	hasEcho := t.EchoChar != 0 && !t.ShowPassword

	if displayText != "" {
		font := t.TextFieldTheme.Font
		if font == nil {
			font = theme.DefaultTheme.BaseFont
			if font == nil {
				font = graphics.SystemFont
			}
		}
		ascent := font.Baseline()
		fontHeight := font.LineHeight()

		baseline := innerRect.Y + (innerRect.Height-fontHeight)/2 + ascent
		x := innerRect.X + 1 - t.scrollOffsetX

		// Horizontal alignment (applies when text is narrower than the field
		// and not scrolled)
		if (t.HAlign == "center" || t.HAlign == "right") && t.scrollOffsetX == 0 {
			textW := float32(font.SimpleWidth(displayText))
			availW := innerRect.Width - 2
			if textW < availW {
				if t.HAlign == "center" {
					x = innerRect.X + (availW-textW)/2
				} else {
					x = innerRect.X + availW - textW
				}
			}
		}

		textPaint := graphics.NewPaint()
		if disabled {
			textPaint.SetColor(theme.DefaultTheme.DisabledText)
		} else if isPlaceholder {
			textPaint.SetColor(t.TextFieldTheme.PlaceholderInk)
		} else {
			textPaint.SetColor(t.TextFieldTheme.TextInk)
		}

		if !disabled && t.hasSelection() {
			selStart, selEnd := t.selectionBounds()

			// For echo mode, map byte offsets in the original text to
			// byte offsets in displayText (each rune becomes 1 echo char).
			dispStart, dispEnd := selStart, selEnd
			if hasEcho {
				dispStart = t.echoByteOffset(selStart)
				dispEnd = t.echoByteOffset(selEnd)
			}

			// Clamp
			if dispStart > len(displayText) {
				dispStart = len(displayText)
			}
			if dispEnd > len(displayText) {
				dispEnd = len(displayText)
			}

			var startX, endX float32
			startX = x + float32(font.SimpleWidth(displayText[:dispStart]))
			endX = x + float32(font.SimpleWidth(displayText[:dispEnd]))

			selPaint := graphics.NewPaint()
			selPaint.SetColor(t.TextFieldTheme.SelectionInk)
			canvas.DrawRect(graphics.Rect{
				X:      startX,
				Y:      innerRect.Y,
				Width:  endX - startX,
				Height: innerRect.Height,
			}, selPaint)

			selTextPaint := graphics.NewPaint()
			selTextPaint.SetColor(t.TextFieldTheme.SelectedTextInk)

			if dispStart > 0 {
				t1 := graphics.NewText(displayText[:dispStart], &graphics.TextDecoration{Font: font, OnBackgroundInk: textPaint.Color()})
				t1.Draw(canvas, x, baseline)
			}
			if dispEnd > dispStart {
				selectedX := x + float32(font.SimpleWidth(displayText[:dispStart]))
				t2 := graphics.NewText(displayText[dispStart:dispEnd], &graphics.TextDecoration{Font: font, OnBackgroundInk: selTextPaint.Color()})
				t2.Draw(canvas, selectedX, baseline)
			}
			if dispEnd < len(displayText) {
				afterX := x + float32(font.SimpleWidth(displayText[:dispEnd]))
				t3 := graphics.NewText(displayText[dispEnd:], &graphics.TextDecoration{Font: font, OnBackgroundInk: textPaint.Color()})
				t3.Draw(canvas, afterX, baseline)
			}
		} else {
			ver := theme.ThemeVersion()
			if t.textCache == nil || t.textCacheVer != ver {
				t.textCache = graphics.NewText(displayText, &graphics.TextDecoration{Font: font, OnBackgroundInk: textPaint.Color()})
				t.textCacheVer = ver
			}
			t.textCache.Draw(canvas, x, baseline)
		}

		if t.composing && !disabled && !hasEcho && t.composeEnd > t.composeStart {
			// A preedit is UNDERLINED, the convention every IME user reads as "not committed yet".
			lo, hi := t.composeStart, t.composeEnd
			if lo < 0 {
				lo = 0
			}
			if hi > len(displayText) {
				hi = len(displayText)
			}
			if lo < hi {
				ux1 := x + float32(font.SimpleWidth(displayText[:lo]))
				ux2 := x + float32(font.SimpleWidth(displayText[:hi]))
				ul := graphics.NewPaint()
				ul.SetColor(textPaint.Color())
				canvas.DrawRect(graphics.NewRect(ux1, baseline+2, ux2-ux1, 1), ul)
			}
		}

		if t.Focused && t.CursorVisible && !disabled {
			var cursorDispPos int
			if hasEcho {
				cursorDispPos = t.echoByteOffset(t.CursorPos)
			} else {
				cursorDispPos = t.CursorPos
			}
			if cursorDispPos > len(displayText) {
				cursorDispPos = len(displayText)
			}
			textWidth := font.SimpleWidth(displayText[:cursorDispPos])
			cursorX := x + textWidth

			cursorPaint := graphics.NewPaint()
			cursorPaint.SetColor(t.TextFieldTheme.TextInk)
			canvas.DrawLine(cursorX, innerRect.Y, cursorX, innerRect.Y+innerRect.Height, cursorPaint)
		}
	}

	canvas.Restore()
}

func (t *TextField) displayText() string {
	if t.EchoChar != 0 && len(t.Text) > 0 && !t.ShowPassword {
		return t.echoDisplayText()
	}
	return t.Text
}

func (t *TextField) SetShowPassword(show bool) *TextField {
	t.ShowPassword = show
	t.textCache = nil
	t.MarkForRedraw()
	return t
}

func (t *TextField) TogglePassword() *TextField {
	t.ShowPassword = !t.ShowPassword
	t.textCache = nil
	t.MarkForRedraw()
	return t
}

func (t *TextField) ensureCursorVisible() {
	font := t.TextFieldTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	frame := t.FrameRect()
	leftOffset, rightOffset, _, _ := t.textInsets()
	innerWidth := frame.Width - leftOffset - rightOffset - 2
	if innerWidth <= 0 {
		innerWidth = 50
	}

	// Cursor X position in unscrolled space, using display text
	disp := t.displayText()
	cursorDisp := t.CursorPos
	if t.EchoChar != 0 && !t.ShowPassword {
		cursorDisp = t.echoByteOffset(t.CursorPos)
	}
	if cursorDisp > len(disp) {
		cursorDisp = len(disp)
	}
	cursorX := font.SimpleWidth(disp[:cursorDisp])

	if cursorX < t.scrollOffsetX {
		t.scrollOffsetX = cursorX
	} else if cursorX > t.scrollOffsetX+innerWidth {
		t.scrollOffsetX = cursorX - innerWidth
	}
	if t.scrollOffsetX < 0 {
		t.scrollOffsetX = 0
	}
}

func (t *TextField) OnFocus() {
	t.Focused = true
	// Select all on focus like HTML textfields (for tab navigation)
	t.SelectStart = 0
	t.SelectEnd = len(t.Text)
	t.CursorPos = len(t.Text)
	t.selectionAnchor = 0
	t.ensureCursorVisible()
	t.startCursorBlink()
	t.MarkForRedraw()
	if t.OnFocusGained != nil {
		t.OnFocusGained()
	}
}

func (t *TextField) OnBlur() {
	// Losing focus abandons the preedit: it was never committed.
	t.cancelComposition()
	t.Focused = false
	t.clearSelection()
	t.selectionAnchor = 0
	t.stopCursorBlink()
	t.MarkForRedraw()
	if t.OnFocusLost != nil {
		t.OnFocusLost()
	}
}

func (t *TextField) startCursorBlink() {
	if t.CursorTimer != nil {
		t.CursorTimer.Stop()
	}
	t.CursorVisible = true
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

func (t *TextField) stopCursorBlink() {
	if t.CursorTimer != nil {
		t.CursorTimer.Stop()
		t.CursorTimer = nil
	}
}

// ── Clipboard helpers ───────────────────────────────────────────

func (t *TextField) Cut() {
	if t.ReadOnly {
		return
	}
	// No selection: Cut does NOTHING. It used to select all and cut, so Ctrl+X on an empty field
	// wiped the value — a data-loss trap in a form.
	if !t.hasSelection() {
		return
	}
	start, end := t.selectionBounds()
	platform.GlobalClipboard.SetText(t.Text[start:end])
	t.pushUndo()
	t.Text = t.Text[:start] + t.Text[end:]
	t.CursorPos = start
	t.clearSelection()
	t.selectionAnchor = start
	t.textCache = nil
	t.ensureCursorVisible()
	t.MarkForRedraw()
	if t.OnTextChanged != nil {
		t.OnTextChanged(t.Text)
	}
}

func (t *TextField) Copy() {
	if !t.hasSelection() {
		return
	}
	start, end := t.selectionBounds()
	platform.GlobalClipboard.SetText(t.Text[start:end])
}

func (t *TextField) Paste() {
	if t.ReadOnly {
		return
	}
	text := platform.GlobalClipboard.GetText()
	if text == "" {
		return
	}
	// A single-line field cannot hold a line break, so a paste turns newlines and tabs into spaces —
	// what a browser's <input> does. Without it the field held a document it could not edit.
	text = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		return r
	}, text)
	// Enforce max length
	if t.MaxLength > 0 {
		currentRunes := utf8.RuneCountInString(t.Text)
		pasteRunes := utf8.RuneCountInString(text)
		// If there's a selection, account for replaced text
		if t.hasSelection() {
			selStart, selEnd := t.selectionBounds()
			replacedRunes := utf8.RuneCountInString(t.Text[selStart:selEnd])
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
	t.pushUndo()
	if t.hasSelection() {
		start, end := t.selectionBounds()
		t.Text = t.Text[:start] + text + t.Text[end:]
		t.CursorPos = start + len(text)
	} else {
		if t.Overwrite {
			// Overwrite mode replaces as many characters as the paste brings.
			end := t.CursorPos
			for i := 0; i < utf8.RuneCountInString(text) && end < len(t.Text); i++ {
				end = t.nextRuneStart(end)
			}
			t.Text = t.Text[:t.CursorPos] + text + t.Text[end:]
			t.CursorPos += len(text)
		} else {
			t.Text = t.Text[:t.CursorPos] + text + t.Text[t.CursorPos:]
			t.CursorPos += len(text)
		}
	}
	t.clearSelection()
	t.selectionAnchor = t.CursorPos
	t.textCache = nil
	t.ensureCursorVisible()
	t.MarkForRedraw()
	if t.OnTextChanged != nil {
		t.OnTextChanged(t.Text)
	}
}

func (t *TextField) SelectAll() {
	t.SelectStart = 0
	t.SelectEnd = len(t.Text)
	t.CursorPos = len(t.Text)
	t.selectionAnchor = 0
	t.MarkForRedraw()
}

// selectWord selects the word containing (or adjacent to) the given byte position.
func (t *TextField) selectWord(pos int) {
	if pos < 0 {
		pos = 0
	}
	if pos > len(t.Text) {
		pos = len(t.Text)
	}
	start, end := wordSpanAt(t.Text, pos)
	t.SelectStart = start
	t.SelectEnd = end
	t.CursorPos = end
	t.selectionAnchor = start
	t.wordAnchorStart, t.wordAnchorEnd = start, end
	t.MarkForRedraw()
}

func (t *TextField) OnMouseDown(x, y float32, button int) bool {
	if !t.Enabled() {
		return false
	}
	// Clicking away abandons an uncommitted composition.
	t.cancelComposition()
	if button == 1 {
		items := []MenuItem{
			{Label: "Cut", Enabled: true, Shortcut: "Ctrl+X", Action: func() { t.Cut() }},
			{Label: "Copy", Enabled: true, Shortcut: "Ctrl+C", Action: func() { t.Copy() }},
			{Label: "Paste", Enabled: true, Shortcut: "Ctrl+V", Action: func() { t.Paste() }},
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
		return true
	}
	// Check for eye icon click (password toggle)
	if t.EchoChar != 0 && t.eyeSVG != nil {
		textFieldIconSize := float32(14)
		iconOffset := float32(0)
		if t.TrailingIcon != "" {
			iconOffset = 18
		}
		eyeX := t.Width() - textFieldIconSize - 4 - iconOffset
		eyeY := (t.Height() - textFieldIconSize) / 2
		if x >= eyeX && x <= eyeX+textFieldIconSize && y >= eyeY && y <= eyeY+textFieldIconSize {
			t.TogglePassword()
			return true
		}
	}

	pos := t.charIndexAtX(x)

	// Detect double/triple click
	now := time.Now()
	if now.Sub(t.lastClickTime) < 350*time.Millisecond && pos == t.lastClickPos {
		t.clickCount++
	} else {
		t.clickCount = 1
	}
	t.lastClickTime = now
	t.lastClickPos = pos

	if !t.Focused {
		t.Focused = true
		t.startCursorBlink()
		if t.OnFocusGained != nil {
			t.OnFocusGained()
		}
	}

	shift := false
	if w := t.Window(); w != nil {
		shift = w.Modifiers().ShiftDown()
	}
	t.applyClick(pos, shift)
	t.ensureCursorVisible()
	t.MarkForRedraw()
	return true
}

// applyClick turns a press at byte offset pos into a caret or a selection. It is separate from
// OnMouseDown so the gesture rules can be tested without guessing where pixel x lands.
func (t *TextField) applyClick(pos int, shift bool) {
	switch {
	case t.clickCount == 2:
		// Double-click: select the word — and keep DRAGGING by words, which is the rest of the
		// gesture in every other editor.
		t.selectWord(pos)
		t.selecting = true
		t.selectingWord = true
		return
	case t.clickCount >= 3:
		// Triple-click: select all
		t.SelectAll()
		t.selecting = false
		t.selectingWord = false
		return
	}

	// A press INSIDE an existing selection does not clear it: that is the gesture that starts a
	// drag-and-drop. If no drag follows, the release collapses the caret, as every editor does.
	if !shift && t.hasSelection() && pos >= t.SelectStart && pos <= t.SelectEnd {
		t.dragArmed = true
		t.dragText = t.Text[t.SelectStart:t.SelectEnd]
		t.pressPos = pos
		t.selecting = false
		t.selectingWord = false
		return
	}
	t.dragArmed = false

	if shift {
		// Shift+click EXTENDS the selection from the caret (or the existing anchor) instead of
		// starting a new one.
		if !t.hasSelection() {
			t.selectionAnchor = t.CursorPos
		}
		t.CursorPos = pos
		t.SelectStart = min(t.selectionAnchor, pos)
		t.SelectEnd = max(t.selectionAnchor, pos)
		t.selecting = true
		t.selectingWord = false
		return
	}

	t.CursorPos = pos
	t.SelectStart = pos
	t.SelectEnd = pos
	t.selectionAnchor = pos
	t.selecting = true
	t.selectingWord = false
	t.dragArmed = false
}

func (t *TextField) OnMouseUp(x, y float32, button int) bool {
	t.selecting = false
	t.selectingWord = false
	if t.dragArmed {
		// No drag happened: a click inside the selection collapses the caret there.
		t.dragArmed = false
		t.CursorPos = t.pressPos
		t.clearSelection()
		t.ensureCursorVisible()
		t.MarkForRedraw()
	}
	t.dragText = ""
	return true
}

func (t *TextField) OnMouseMove(x, y float32) {
	if !t.Enabled() {
		return
	}
	if !t.selecting {
		return
	}
	pos := t.charIndexAtX(x)
	if t.selectingWord {
		t.extendWordDragTo(pos)
		t.ensureCursorVisible()
		t.MarkForRedraw()
		return
	}
	if pos != t.CursorPos {
		t.CursorPos = pos
		t.SelectStart = min(t.selectionAnchor, pos)
		t.SelectEnd = max(t.selectionAnchor, pos)
		t.ensureCursorVisible()
		t.MarkForRedraw()
	}
}

// extendWordDragTo grows the double-click selection by WHOLE WORDS, anchored on the word that was
// clicked. pos is a byte offset in Text.
func (t *TextField) extendWordDragTo(pos int) {
	lo, hi := wordSpanAt(t.Text, pos)
	start, end := t.wordAnchorStart, t.wordAnchorEnd
	if lo < start {
		start = lo
	}
	if hi > end {
		end = hi
	}
	t.SelectStart, t.SelectEnd = start, end
	if pos < t.wordAnchorStart {
		t.CursorPos = start
	} else {
		t.CursorPos = end
	}
	t.selectionAnchor = t.wordAnchorStart
}

// ── Drag and drop ───────────────────────────────────────────────────────
//
// A text field is both a SOURCE (dragging its selection) and a TARGET (dropping text into it). The
// window's DragManager drives both: OnDragStart is called once the pointer has moved past the drag
// threshold, and the drop coordinates are in CONTENT space, so every method converts once.

var _ meui.DragSource = (*TextField)(nil)
var _ meui.DropTarget = (*TextField)(nil)

// OnDragStart starts a drag when the press landed inside the selection. Any other press is a
// selection gesture, which is why the editor does not begin a drag for every mouse movement.
func (t *TextField) OnDragStart(x, y float32) *meui.DragData {
	if !t.dragArmed || t.dragText == "" {
		return nil
	}
	text := t.dragText
	t.dragArmed = false
	return &meui.DragData{Type: "text", Source: meui.Widget(t), Data: text}
}

// OnDragEnter records where the drop would land, so OnDragLeave can put the caret back.
func (t *TextField) OnDragEnter(data *meui.DragData, x, y float32) {
	if t.dropActive {
		return
	}
	t.dropActive = true
	t.dropMoved = false
	if data != nil && data.Source == meui.Widget(t) {
		// The SOURCE keeps its selection and its caret: the drop decides whether this is a move or
		// a no-op, and moving the caret here would destroy the selection the drag borrowed.
		return
	}
	t.dropMoved = true
	t.dropPrevPos = t.CursorPos
	t.moveCaretToDrop(x, y)
}

func (t *TextField) OnDragOver(data *meui.DragData, x, y float32) {
	if t.dropActive && t.dropMoved {
		t.moveCaretToDrop(x, y)
	}
}

func (t *TextField) OnDragLeave(data *meui.DragData) {
	if !t.dropActive {
		return
	}
	t.dropActive = false
	if !t.dropMoved {
		// Nothing was moved for this drag (it was the source's own hover), so there is nothing to
		// undo — and the borrowed selection must survive.
		return
	}
	t.dropMoved = false
	t.CursorPos = t.dropPrevPos
	t.clearSelection()
	t.MarkForRedraw()
}

// OnDragDrop inserts the dragged text at the pointer. A drop into the field it came from MOVES the
// text (the source selection is cut first), and a drop inside that same selection is a no-op — both
// as in every editor.
func (t *TextField) OnDragDrop(data *meui.DragData, x, y float32) {
	text, _ := data.Data.(string)
	t.dropActive = false
	t.dropMoved = false
	t.selecting = false
	t.selectingWord = false
	t.dragArmed = false
	t.dragText = ""
	if text == "" {
		return
	}
	pt := t.PointFromRoot(x, y)
	pos := t.charIndexAtX(pt.X)

	if data.Source == meui.Widget(t) && t.hasSelection() {
		lo, hi := t.selectionBounds()
		if pos >= lo && pos <= hi {
			t.MarkForRedraw()
			return
		}
		if pos > hi {
			pos -= hi - lo
		}
		t.pushUndo()
		t.Text = t.Text[:lo] + t.Text[hi:]
		t.CursorPos = pos
		t.clearSelection()
		t.textCache = nil
	} else if data.Source != nil && data.Source != meui.Widget(t) {
		// A drop into ANOTHER field is a MOVE: the source gives up the text it lent to the drag.
		if r, ok := data.Source.(interface{ RemoveDraggedText(*meui.DragData) }); ok {
			r.RemoveDraggedText(data)
		}
	}
	t.insertStringAt(pos, text)
	t.MarkForRedraw()
}

// RemoveDraggedText deletes the selection a drag borrowed, so a drop elsewhere is a MOVE. The source
// can still refuse (a drag that ended over nothing never calls this).
func (t *TextField) RemoveDraggedText(data *meui.DragData) {
	if data == nil || data.Source != meui.Widget(t) || !t.hasSelection() {
		return
	}
	start, end := t.selectionBounds()
	t.pushUndo()
	t.Text = t.Text[:start] + t.Text[end:]
	t.CursorPos = start
	t.clearSelection()
	t.selectionAnchor = start
	t.textCache = nil
	t.ensureCursorVisible()
	t.MarkForRedraw()
	if t.OnTextChanged != nil {
		t.OnTextChanged(t.Text)
	}
}

// moveCaretToDrop puts the caret where a drop would land, without touching the selection.
func (t *TextField) moveCaretToDrop(x, y float32) {
	pt := t.PointFromRoot(x, y)
	t.CursorPos = t.charIndexAtX(pt.X)
	t.clearSelection()
	t.ensureCursorVisible()
	t.MarkForRedraw()
}

// insertStringAt inserts text at a byte offset as one edit, enforcing MaxLength. The dropped text is
// single-line for the same reason a paste is.
func (t *TextField) insertStringAt(pos int, text string) {
	if pos < 0 {
		pos = 0
	}
	if pos > len(t.Text) {
		pos = len(t.Text)
	}
	text = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		return r
	}, text)
	if t.MaxLength > 0 {
		available := t.MaxLength - utf8.RuneCountInString(t.Text)
		if available <= 0 {
			return
		}
		if n := utf8.RuneCountInString(text); n > available {
			text = string([]rune(text)[:available])
		}
	}
	if text == "" {
		return
	}
	if !t.hasSelection() {
		t.pushUndoGrouped(undoKindInsert)
	}
	t.Text = t.Text[:pos] + text + t.Text[pos:]
	t.CursorPos = pos + len(text)
	t.clearSelection()
	t.selectionAnchor = t.CursorPos
	t.textCache = nil
	t.ensureCursorVisible()
	t.MarkForRedraw()
	if t.OnTextChanged != nil {
		t.OnTextChanged(t.Text)
	}
}

// OnCharacter handles character input (IME, unicode, etc.).
func (t *TextField) OnCharacter(ch rune) {
	if !t.Enabled() || !t.Focused || t.ReadOnly {
		return
	}
	// The committed characters REPLACE the preedit: the IME's composition is not text yet.
	if t.composing {
		t.cancelComposition()
	}
	// One rune, one character. There is no key-down insert to deduplicate any more: a key-down
	// carries a KEYCODE (which has no character in it), and only the platform's text event knows
	// which rune the layout produced.
	t.insertChar(ch)
}

// OnKeyRepeat is a key the user is HOLDING — Backspace over a word, an arrow along a row, a scrub.
// The window classifies an OS auto-repeat as a repeat rather than a fresh press (a widget without
// this method is asked once, deliberately), so this is what makes a held key work here.
func (t *TextField) OnKeyRepeat(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	return t.OnKeyDown(keyCode, mod)
}

func (t *TextField) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if !t.Enabled() || !t.Focused {
		return false
	}
	if t.composing && keyCode == platform.KeyEscape {
		// Escape is the IME's cancel. It abandons the preedit and nothing else.
		t.cancelComposition()
		return true
	}

	// Clipboard shortcuts. Read-only still allows copy and select-all, which is what a read-only field
	// is FOR; cut, paste and undo are editing and are refused below.
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
			return false
		}
		switch keyCode {
		case platform.KeyX:
			t.Cut()
			return true
		case platform.KeyV:
			t.Paste()
			return true
		case platform.KeyZ:
			if mod.ShiftDown() {
				return t.Redo()
			}
			return t.Undo()
		case platform.KeyY:
			return t.Redo()
		}
	}
	if t.ReadOnly {
		// A read-only field is still READABLE: the caret moves, the arrows select, and the selection
		// can be copied. Only editing is refused. (This used to refuse the arrows too, so a read-only
		// value could not be selected with the keyboard at all.)
		switch keyCode {
		case platform.KeyLeft, platform.KeyRight, platform.KeyHome, platform.KeyEnd:
		default:
			return false
		}
	}
	// The classic clipboard keys, which terminals and long-time users still reach for.
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
		t.Overwrite = !t.Overwrite
		t.MarkForRedraw()
		return true
	}

	// Enter / Return
	if keyCode == platform.KeyReturn || keyCode == platform.KeyNumPadEnter {
		if t.OnSubmit != nil {
			t.OnSubmit(t.Text)
		}
		return true
	}

	// Arrow keys. A plain arrow COLLAPSES a selection to the side it moves towards — Left to the
	// START, Right to the END — and a word modifier moves by word with or without Shift (it used to
	// move by word ONLY when Shift was held, so Ctrl+Left did nothing at all).
	if keyCode == platform.KeyLeft || keyCode == platform.KeyRight {
		left := keyCode == platform.KeyLeft
		word := wordModDown(mod)
		if mod.ShiftDown() {
			if !t.hasSelection() {
				t.selectionAnchor = t.CursorPos
			}
			if word && left {
				t.CursorPos = t.prevWordStart(t.CursorPos)
			} else if word {
				t.CursorPos = t.nextWordStart(t.CursorPos)
			} else if left {
				t.CursorPos = t.prevRuneStart(t.CursorPos)
			} else {
				t.CursorPos = t.nextRuneStart(t.CursorPos)
			}
			t.SelectStart = min(t.selectionAnchor, t.CursorPos)
			t.SelectEnd = max(t.selectionAnchor, t.CursorPos)
		} else if t.hasSelection() {
			s, e := t.selectionBounds()
			if left {
				t.CursorPos = s
			} else {
				t.CursorPos = e
			}
			t.clearSelection()
		} else {
			if word && left {
				t.CursorPos = t.prevWordStart(t.CursorPos)
			} else if word {
				t.CursorPos = t.nextWordStart(t.CursorPos)
			} else if left {
				t.CursorPos = t.prevRuneStart(t.CursorPos)
			} else {
				t.CursorPos = t.nextRuneStart(t.CursorPos)
			}
			t.clearSelection()
		}
		t.undoKind = undoKindNone
		t.ensureCursorVisible()
		t.MarkForRedraw()
		return true
	}

	if keyCode == platform.KeyHome {
		if mod.ShiftDown() {
			if !t.hasSelection() {
				t.selectionAnchor = t.CursorPos
			}
			t.CursorPos = 0
			t.SelectStart = min(t.selectionAnchor, t.CursorPos)
			t.SelectEnd = max(t.selectionAnchor, t.CursorPos)
		} else {
			t.CursorPos = 0
			t.clearSelection()
		}
		t.undoKind = undoKindNone
		t.ensureCursorVisible()
		t.MarkForRedraw()
		return true
	}
	if keyCode == platform.KeyEnd {
		if mod.ShiftDown() {
			if !t.hasSelection() {
				t.selectionAnchor = t.CursorPos
			}
			t.CursorPos = len(t.Text)
			t.SelectStart = min(t.selectionAnchor, t.CursorPos)
			t.SelectEnd = max(t.selectionAnchor, t.CursorPos)
		} else {
			t.CursorPos = len(t.Text)
			t.clearSelection()
		}
		t.undoKind = undoKindNone
		t.ensureCursorVisible()
		t.MarkForRedraw()
		return true
	}

	if keyCode == platform.KeyBackspace {
		if t.hasSelection() {
			t.pushUndo()
			t.deleteSelection()
		} else if wordModDown(mod) {
			// Delete previous word
			t.pushUndoGrouped(undoKindDelete)
			wordStart := t.prevWordStart(t.CursorPos)
			t.Text = t.Text[:wordStart] + t.Text[t.CursorPos:]
			t.CursorPos = wordStart
			t.clearSelection()
			t.selectionAnchor = wordStart
		} else if t.CursorPos > 0 {
			t.pushUndoGrouped(undoKindDelete)
			prev := t.prevRuneStart(t.CursorPos)
			t.Text = t.Text[:prev] + t.Text[t.CursorPos:]
			t.CursorPos = prev
			t.clearSelection()
			t.selectionAnchor = prev
		}
		t.textCache = nil
		t.ensureCursorVisible()
		t.MarkForRedraw()
		if t.OnTextChanged != nil {
			t.OnTextChanged(t.Text)
		}
		return true
	}

	if keyCode == platform.KeyDelete {
		if t.hasSelection() {
			t.pushUndo()
			t.deleteSelection()
		} else if wordModDown(mod) {
			// Delete next word
			t.pushUndoGrouped(undoKindDelete)
			wordEnd := t.nextWordStart(t.CursorPos)
			t.Text = t.Text[:t.CursorPos] + t.Text[wordEnd:]
			t.clearSelection()
			t.selectionAnchor = t.CursorPos
		} else if t.CursorPos < len(t.Text) {
			t.pushUndoGrouped(undoKindDelete)
			next := t.nextRuneStart(t.CursorPos)
			t.Text = t.Text[:t.CursorPos] + t.Text[next:]
			t.clearSelection()
			t.selectionAnchor = t.CursorPos
		}
		t.textCache = nil
		t.ensureCursorVisible()
		t.MarkForRedraw()
		if t.OnTextChanged != nil {
			t.OnTextChanged(t.Text)
		}
		return true
	}

	// Printable characters are NOT synthesized from key codes. This block used to map keyCode 65-90 to
	// letters, 48-57 to digits and 32 to a space — ASCII values, against HID keycodes: 32 is the "3"
	// key (so pressing 3 inserted a space first), 48-57 are ] \ ; ' ` , . / CapsLock F1 (typed as
	// 0-9), and 65-90 are F8-F12 and the keypad. Text arrives as EventRuneTyped from the platform,
	// which is the only thing that knows the keyboard LAYOUT, so the rune path is the only path.

	return false
}

// OnComposition shows the IME's in-progress text. It is called on every composition change, so the
// previous preedit is removed first; an EMPTY text means the composition was cancelled. The preedit
// is deliberately not undoable and does not fire OnTextChanged — it is replaced by the committed
// characters through OnCharacter.
func (t *TextField) OnComposition(text string, start, length int) bool {
	if !t.Enabled() || !t.Focused || t.ReadOnly {
		return false
	}
	t.cancelComposition()
	if text == "" {
		t.MarkForRedraw()
		return true
	}
	// A composition replaces a selection, like typing does.
	if t.hasSelection() {
		s, e := t.selectionBounds()
		t.Text = t.Text[:s] + t.Text[e:]
		t.CursorPos = s
		t.clearSelection()
	}
	pos := t.CursorPos
	if t.MaxLength > 0 {
		available := t.MaxLength - utf8.RuneCountInString(t.Text)
		if available <= 0 {
			t.MarkForRedraw()
			return true
		}
		if n := utf8.RuneCountInString(text); n > available {
			text = string([]rune(text)[:available])
		}
	}
	t.Text = t.Text[:pos] + text + t.Text[pos:]
	t.composeStart, t.composeEnd = pos, pos+len(text)
	t.composing = true
	caret := start
	if caret < 0 {
		caret = 0
	}
	if caret > len(text) {
		caret = len(text)
	}
	t.CursorPos = t.composeStart + caret
	t.clearSelection()
	t.selectionAnchor = t.CursorPos
	t.textCache = nil
	t.ensureCursorVisible()
	t.MarkForRedraw()
	return true
}

// cancelComposition removes the preedit WITHOUT an undo step or a change notification: it was never
// committed text.
func (t *TextField) cancelComposition() {
	if !t.composing {
		return
	}
	lo, hi := t.composeStart, t.composeEnd
	if lo < 0 {
		lo = 0
	}
	if hi > len(t.Text) {
		hi = len(t.Text)
	}
	if lo < hi {
		t.Text = t.Text[:lo] + t.Text[hi:]
		if t.CursorPos > hi {
			t.CursorPos -= hi - lo
		} else if t.CursorPos > lo {
			t.CursorPos = lo
		}
	}
	t.composing = false
	t.composeStart, t.composeEnd = 0, 0
	t.clearSelection()
	t.textCache = nil
	t.MarkForRedraw()
}

func (t *TextField) insertChar(ch rune) {
	// Enforce max length, allowing for the selection this character REPLACES. Counting the existing
	// text alone refused the keystroke at MaxLength even when it would not have grown the value.
	replacing := 0
	if t.hasSelection() {
		start, end := t.selectionBounds()
		replacing = utf8.RuneCountInString(t.Text[start:end])
	}
	if t.MaxLength > 0 && utf8.RuneCountInString(t.Text)-replacing >= t.MaxLength {
		return
	}
	if t.hasSelection() {
		t.pushUndo()
	} else {
		t.pushUndoGrouped(undoKindInsert)
	}
	if t.hasSelection() {
		t.deleteSelection()
	} else if t.Overwrite && t.CursorPos < len(t.Text) {
		// Overwrite mode replaces the character under the caret instead of pushing it right.
		next := t.nextRuneStart(t.CursorPos)
		t.Text = t.Text[:t.CursorPos] + t.Text[next:]
	}
	chBytes := string(ch)
	if t.CursorPos >= len(t.Text) {
		t.Text += chBytes
	} else {
		t.Text = t.Text[:t.CursorPos] + chBytes + t.Text[t.CursorPos:]
	}
	t.CursorPos += len(chBytes)
	t.clearSelection()
	t.selectionAnchor = t.CursorPos
	t.textCache = nil
	t.ensureCursorVisible()
	t.MarkForRedraw()
	if t.OnTextChanged != nil {
		t.OnTextChanged(t.Text)
	}
}

// charIndexAtX returns the byte offset in t.Text closest to the given x coordinate.
func (t *TextField) charIndexAtX(x float32) int {
	textFieldIconSize := float32(14)
	textFieldIconPadding := float32(3)
	leftOffset := float32(4)

	if t.LeadingIcon != "" {
		leftOffset += textFieldIconSize + textFieldIconPadding
	}

	textStartX := leftOffset + 1
	adjustedX := x + t.scrollOffsetX
	if adjustedX <= textStartX {
		return 0
	}
	offset := adjustedX - textStartX

	font := t.TextFieldTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}

	// range over string advances i to each rune's byte offset automatically
	accum := float32(0)
	for i, r := range t.Text {
		w := font.SimpleWidth(string(r))
		mid := accum + w/2
		if offset < mid {
			return i
		}
		accum += w
	}
	return len(t.Text)
}

func (t *TextField) deleteSelection() {
	if !t.hasSelection() {
		return
	}
	start, end := t.selectionBounds()
	t.Text = t.Text[:start] + t.Text[end:]
	t.CursorPos = start
	t.clearSelection()
	t.selectionAnchor = start
	t.textCache = nil
	t.MarkForRedraw()
}

func (t *TextField) MinimumDimensions() (float32, float32) {
	return 80, 30
}

func (t *TextField) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	font := t.TextFieldTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	lh := font.LineHeight()
	if lh < 30 {
		lh = 30
	}
	fr := t.FrameRect()
	if fr.Width > 0 {
		return fr.Width, lh
	}
	// Estimate width from text or placeholder
	text := t.Text
	if text == "" {
		text = t.Placeholder
	}
	tw := font.SimpleWidth(text) + 16
	if tw < 80 {
		tw = 80
	}
	return tw, lh
}

func (t *TextField) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "text":
		if s, ok := value.(string); ok {
			t.SetText(s)
		}
	case "placeholderText", "placeholder":
		if s, ok := value.(string); ok {
			t.SetPlaceholder(s)
		}
	case "echoChar", "passwordChar":
		if s, ok := value.(string); ok && len(s) > 0 {
			t.SetEchoChar([]rune(s)[0])
		}
	case "maxLength", "maximumLength":
		switch n := value.(type) {
		case float64:
			t.SetMaxLength(int(n))
		case int:
			t.SetMaxLength(n)
		}
	case "readOnly", "editable":
		if b, ok := value.(bool); ok {
			t.SetReadOnly(b)
		}
	case "horizontalAlignment", "hAlign":
		if s, ok := value.(string); ok {
			t.SetHAlign(s)
		}
	case "validationState":
		if s, ok := value.(string); ok {
			t.SetValidationState(s)
		}
	case "leadingIcon":
		if s, ok := value.(string); ok {
			t.SetLeadingIcon(s)
		}
	case "trailingIcon":
		if s, ok := value.(string); ok {
			t.SetTrailingIcon(s)
		}
	}
}

// Dispose cleans up the TextField resources, especially the cursor timer.
func (t *TextField) Dispose() {
	t.stopCursorBlink()
	t.OnTextChanged = nil
	t.OnSubmit = nil
	t.OnFocusGained = nil
	t.OnFocusLost = nil
	t.BaseWidget.Dispose()
}

func (t *TextField) Focusable() bool {
	return t.Enabled()
}
