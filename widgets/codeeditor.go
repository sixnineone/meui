package widgets

import (
	"fmt"
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

type EditAction int

const (
	ActionInsertText EditAction = iota
	ActionDeleteText
	ActionReplaceText
	ActionInsertNewline
	ActionIndent
	ActionOutdent
)

type UndoEntry struct {
	Action     EditAction
	Text       string
	StartLine  int
	StartCol   int
	EndLine    int
	EndCol     int
	CursorLine int
	CursorCol  int
	SelStart   int
	SelEnd     int
}

type CodeEditor struct {
	meui.BaseWidget
	*EditorTheme
	*EditorSettings
	cache           meui.CacheBase
	Lines           []string
	lineTokens      [][]SyntaxToken
	tokenizerStates []interface{}
	CursorLine      int
	CursorCol       int
	// preferredCol is the column a run of vertical moves aims at (-1 = none). Columns are BYTE
	// offsets into a line, as everywhere else in this editor, but a step never lands mid-rune.
	preferredCol int
	// IME composition: the preedit is shown inline (not underlined) and is neither undoable nor
	// announced through OnTextChanged; the commit removes it and inserts the committed characters.
	composing            bool
	composeStart         int
	composeEnd           int
	SelectStart          int
	SelectEnd            int
	selectionAnchor      int
	ScrollX              float32
	ScrollY              float32
	Focused              bool
	CursorVisible        bool
	CursorTimer          *time.Timer
	inactivityTimer      *time.Timer
	interacting          bool
	OnTextChanged        func(string)
	OnCursorChanged      func(line, col int)
	OnSave               func()
	OnFormat             func(text string) (string, error)
	FilePath             string
	Language             string
	Modified             bool
	ReadOnly             bool
	SyntaxTheme          *SyntaxTheme
	tokenizer            SyntaxTokenizer
	tokenizerMutex       sync.RWMutex
	undoStack            []UndoEntry
	redoStack            []UndoEntry
	maxUndo              int
	dirtyLines           map[int]bool
	dirtyAll             bool
	scrollMaxX           float32
	scrollMaxY           float32
	mouseDown            bool
	dragSelecting        bool
	isDragging           bool // drag-and-drop in progress
	dragFromLine         int
	dragFromCol          int
	columnSelect         bool // Alt+drag column selection
	colSelStartLine      int
	colSelStartCol       int
	colSelEndLine        int
	colSelEndCol         int
	lineHeight           float32
	charWidth            float32
	gutterWidth          float32
	textAreaWidth        float32
	textAreaHeight       float32
	VScroll              *ScrollBar
	HScroll              *ScrollBar
	scrollBarSize        float32
	totalWidth           float32
	totalHeight          float32
	searchText           string
	searchMatches        []searchMatch
	searchCurrentIdx     int
	searchVisible        bool
	replaceText          string
	replaceVisible       bool
	bracketMatch         bracketPair
	bracketMatchEnd      bracketPair
	ShowMinimap          bool
	minimapWidth         float32
	multiCursor          []cursorPos
	codeFolding          map[int]bool
	foldMarkers          []foldMarker
	foldRanges           map[int]int // start line → end line (inclusive) of fold
	foldExpandSVG        *meui.SVGDocument
	foldCollapseSVG      *meui.SVGDocument
	bookmarks            map[int]bool // line → has bookmark
	breakpoints          map[int]bool // line → has breakpoint
	breakpointSVG        *meui.SVGDocument
	errorSVG             *meui.SVGDocument
	warningSVG           *meui.SVGDocument
	lightbulbSVG         *meui.SVGDocument
	diffAdded            map[int]bool // line → added
	diffModified         map[int]bool // line → modified
	diffDeleted          map[int]bool // line → deleted (draws on line above)
	diagnostics          []Diagnostic
	BracketColorization  bool
	bracketColors        []graphics.Color
	completionVisible    bool
	completionItems      []CompletionItem
	contextMenuItems     []MenuItem
	contextMenuX         float32
	contextMenuY         float32
	contextMenuHover     int
	contextMenuVisible   bool
	contextMenuSub       int
	contextMenuSubItems  []MenuItem
	completionIdx        int
	completionPrefix     string
	completionX          float32
	completionY          float32
	parameterHintVisible bool
	parameterHints       []ParameterHint
	parameterHintIdx     int
	hoverVisible         bool
	hoverText            string
	hoverX               float32
	hoverY               float32
	hoverTimer           *time.Timer
	OnCompletionRequest  func(prefix string) []CompletionItem
	OnCompletionSelect   func(item CompletionItem)
	OnGoToDefinition     func(line, col int)
	OnFindReferences     func(line, col int)
	OnHover              func(line, col int) string
	OnContextMenu        func(line, col int, selectedText string) []MenuItem
}

type cursorPos struct {
	line, col int
	selStart  int // selection offset start for this cursor (-1 = no separate selection)
	selEnd    int // selection offset end for this cursor
}

type searchMatch struct {
	line       int
	start, end int
}

type bracketPair struct {
	line int
	col  int
	ch   rune
}

type foldMarker struct {
	line   int
	folded bool
}

type Diagnostic struct {
	Line     int
	Col      int
	EndLine  int
	EndCol   int
	Severity int
	Message  string
	Source   string
}

type CompletionItem struct {
	Label      string
	Kind       int
	Detail     string
	InsertText string
}

type ParameterHint struct {
	Label         string
	Documentation string
	ActiveParam   int
	StartLine     int
	StartCol      int
	EndLine       int
	EndCol        int
}

func NewCodeEditor() *CodeEditor {
	ce := &CodeEditor{
		BaseWidget:     *meui.NewBaseWidget(),
		EditorTheme:    GlobalEditorTheme,
		EditorSettings: GlobalEditorSettings,
		Lines:          []string{""},
		maxUndo:        100,
		dirtyLines:     make(map[int]bool),
		dirtyAll:       true,
		SyntaxTheme:    GlobalSyntaxTheme,
		ShowMinimap:    true,
		minimapWidth:   60,
		codeFolding:    make(map[int]bool),
		foldRanges:     make(map[int]int),
		bookmarks:      make(map[int]bool),
		breakpoints:    make(map[int]bool),
		preferredCol:   -1,
		diffAdded:      make(map[int]bool),
		diffModified:   make(map[int]bool),
		diffDeleted:    make(map[int]bool),
		diagnostics:    make([]Diagnostic, 0),
		bracketColors: []graphics.Color{
			graphics.Color(0xFFE53935), // red
			graphics.Color(0xFFFFC107), // amber
			graphics.Color(0xFF4CAF50), // green
			graphics.Color(0xFF2196F3), // blue
			graphics.Color(0xFF9C27B0), // purple
			graphics.Color(0xFFFF6F00), // orange
			graphics.Color(0xFF00BCD4), // cyan
			graphics.Color(0xFFE91E63), // pink
		},
	}
	if doc, err := meui.LoadSVGByName("target"); err == nil {
		ce.breakpointSVG = doc
	}
	if doc, err := meui.LoadSVGByName("error"); err == nil {
		ce.errorSVG = doc
	}
	if doc, err := meui.LoadSVGByName("warning"); err == nil {
		ce.warningSVG = doc
	}
	if doc, err := meui.LoadSVGByName("lightbulb"); err == nil {
		ce.lightbulbSVG = doc
	}
	ce.VScroll = NewScrollBar()
	ce.VScroll.SetOrientation(Vertical)
	ce.VScroll.SetParent(ce)
	ce.HScroll = NewScrollBar()
	ce.HScroll.SetOrientation(Horizontal)
	ce.HScroll.SetParent(ce)
	ce.scrollBarSize = 14

	ce.VScroll.SetOnValueChanged(func(v float32) {
		ce.ScrollY = v
		ce.cache.Invalidate()
		ce.MarkForRedraw()
	})
	ce.HScroll.SetOnValueChanged(func(v float32) {
		ce.ScrollX = v
		ce.cache.Invalidate()
		ce.MarkForRedraw()
	})

	if doc, err := meui.LoadSVGByName("chevron_right"); err == nil {
		ce.foldCollapseSVG = doc
	}
	if doc, err := meui.LoadSVGByName("expand_more"); err == nil {
		ce.foldExpandSVG = doc
	}

	ce.tokenizer = &TextTokenizer{}
	ce.initFont()
	ce.recalcCharWidth()
	return ce
}

func (e *CodeEditor) SetFilePath(path string) {
	e.FilePath = path
	e.detectLanguage()
	if path != "" {
		if data, err := readTextFile(path); err == nil {
			e.SetText(string(data))
		}
	}
}

func (e *CodeEditor) detectLanguage() {
	ext := strings.ToLower(e.FilePath)
	if idx := strings.LastIndex(ext, "."); idx >= 0 {
		ext = ext[idx:]
	}
	tokenizer := GetTokenizer(e.FilePath, "")
	if tokenizer != nil {
		e.tokenizer = tokenizer
		e.Language = tokenizer.LanguageName()
	} else {
		e.Language = "Text"
	}
	e.dirtyAll = true
	e.MarkForRedraw()
}

var globalTokenizer *MultiLangTokenizer
var tokenizerOnce sync.Once

func GetTokenizer(filename, lang string) SyntaxTokenizer {
	tokenizerOnce.Do(func() {
		globalTokenizer = NewMultiLangTokenizer()
	})
	return globalTokenizer.GetTokenizer(filename, lang)
}

func (e *CodeEditor) SetText(text string) {
	e.Lines = strings.Split(text, "\n")
	if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
		fmt.Printf("[codeSetText] lines=%d\n", len(e.Lines))
	}
	if len(e.Lines) == 0 {
		e.Lines = []string{""}
	}
	e.CursorLine = 0
	e.CursorCol = 0
	e.SelectStart = 0
	e.SelectEnd = 0
	e.selectionAnchor = 0
	e.ScrollX = 0
	e.ScrollY = 0
	e.dirtyAll = true
	e.tokenizeAll()
	e.Modified = false
	e.cache.Invalidate()
	e.MarkForRedraw()
	if e.OnTextChanged != nil {
		e.OnTextChanged(text)
	}
}

func (e *CodeEditor) Text() string {
	return strings.Join(e.Lines, "\n")
}

func (e *CodeEditor) SetLanguage(lang string) {
	e.Language = lang
	tok := GetTokenizer("", lang)
	if tok != nil {
		e.tokenizer = tok
		e.dirtyAll = true
		e.MarkForRedraw()
	}
}

func (e *CodeEditor) SetSyntaxTheme(st *SyntaxTheme) {
	e.SyntaxTheme = st
	e.MarkForRedraw()
}

func (e *CodeEditor) GetLine(line int) string {
	if line >= 0 && line < len(e.Lines) {
		return e.Lines[line]
	}
	return ""
}

func (e *CodeEditor) LineCount() int {
	return len(e.Lines)
}

func (e *CodeEditor) CursorOffset() int {
	return e.lineColToOffset(e.CursorLine, e.CursorCol)
}

func (e *CodeEditor) lineColToOffset(line, col int) int {
	// Shared with the text area (textlayout.go).
	return docOffsetAt(e.Lines, line, col)
}

func (e *CodeEditor) offsetToLineCol(offset int) (int, int) {
	// Shared with the text area (textlayout.go).
	return docLineColAt(e.Lines, offset)
}

func (e *CodeEditor) resolveFont() graphics.Font {
	if e.Font != nil {
		if e.Font.LineHeight() > 0 {
			return e.Font
		}
	}
	if theme.DefaultTheme.BaseFont != nil {
		f := theme.DefaultTheme.BaseFont
		if f.LineHeight() > 0 {
			return f
		}
	}
	fs := e.FontSize
	if fs < 6 {
		fs = 14
	}
	inter := graphics.MatchFontFace("Inter", 400, 0, 0)
	if inter != nil {
		f := inter.Font(fs)
		if f != nil && f.LineHeight() > 0 {
			e.EditorTheme.Font = f
			return f
		}
	}
	_, monos := graphics.AllFontFaces()
	for _, fd := range monos {
		f := fd.Face().Font(fs)
		if f != nil && f.LineHeight() > 0 {
			e.EditorTheme.Font = f
			return f
		}
	}
	all, _ := graphics.AllFontFaces()
	for _, fd := range all {
		f := fd.Face().Font(fs)
		if f != nil && f.LineHeight() > 0 {
			e.EditorTheme.Font = f
			return f
		}
	}
	return nil
}

func (e *CodeEditor) initFont() {
	e.EditorTheme.Font = e.resolveFont()
}

func (e *CodeEditor) font() graphics.Font {
	return e.EditorTheme.Font
}

func (e *CodeEditor) recalcCharWidth() {
	// Re-resolve font with current FontSize
	e.EditorTheme.Font = nil
	e.EditorTheme.Font = e.resolveFont()
	f := e.font()
	if f == nil || f.LineHeight() <= 0 {
		e.charWidth = 8
		e.lineHeight = 20
		return
	}
	e.charWidth = f.SimpleWidth("W")
	e.lineHeight = f.LineHeight() * e.LineHeight
	if e.lineHeight < 12 {
		e.lineHeight = 16
	}
	if e.charWidth < 5 {
		e.charWidth = 8
	}
}

func (e *CodeEditor) recalcGutterWidth() {
	diffLane := float32(3)  // diff markers
	bpLane := float32(12)   // breakpoint markers
	bmLane := float32(12)   // bookmark markers
	foldLane := float32(14) // fold markers
	if !e.ShowLineNumbers {
		e.gutterWidth = diffLane + bpLane + bmLane + foldLane + 4
		return
	}
	numLines := len(e.Lines)
	digits := 1
	for n := numLines; n >= 10; n /= 10 {
		digits++
	}
	if digits < 2 {
		digits = 2
	}
	e.gutterWidth = diffLane + bpLane + bmLane + foldLane + float32(digits)*e.charWidth + 16
}

func (e *CodeEditor) tokenizeAll() {
	e.lineTokens = make([][]SyntaxToken, len(e.Lines))
	e.tokenizerStates = make([]interface{}, len(e.Lines))
	var state interface{}
	for i, line := range e.Lines {
		tokens, newState := e.tokenizer.Tokenize(line, i, state)
		e.lineTokens[i] = tokens
		e.tokenizerStates[i] = state
		state = newState
	}
	e.dirtyAll = false
	e.dirtyLines = make(map[int]bool)
}

func (e *CodeEditor) ensureFoldRanges() {
	e.foldRanges = e.findFoldRanges()
}

func (e *CodeEditor) tokenizeLines(lines map[int]bool) {
	for line := range lines {
		if line < 0 || line >= len(e.Lines) {
			continue
		}
		var state interface{}
		if line > 0 && line-1 < len(e.tokenizerStates) {
			state = e.tokenizerStates[line-1]
		}
		tokens, newState := e.tokenizer.Tokenize(e.Lines[line], line, state)
		e.lineTokens[line] = tokens
		e.tokenizerStates[line] = newState
		delete(e.dirtyLines, line)
	}
}

// markInteraction is called on every user-driven event (mouse, keyboard).
// It disables offscreen caching temporarily so selection/cursor updates
// appear immediately. After 500ms of inactivity, caching resumes.
func (e *CodeEditor) markInteraction() {
	e.interacting = true
	if e.inactivityTimer != nil {
		e.inactivityTimer.Stop()
	}
	e.inactivityTimer = time.AfterFunc(500*time.Millisecond, func() {
		e.interacting = false
	})
}

func (e *CodeEditor) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !e.Visible() {
		return
	}
	canvas.Save()
	e.ensureFoldRanges()
	e.recalcCharWidth()
	e.recalcGutterWidth()

	if e.dirtyAll {
		e.cache.Invalidate()
		e.tokenizeAll()
	} else if len(e.dirtyLines) > 0 {
		e.tokenizeLines(e.dirtyLines)
	}

	sbSize := e.scrollBarSize
	hasVScroll := false
	hasHScroll := false

	e.textAreaWidth = rect.Width - e.gutterWidth - e.minimapWidth
	if !e.ShowMinimap {
		e.textAreaWidth = rect.Width - e.gutterWidth
	}
	if e.textAreaWidth < 50 {
		e.textAreaWidth = 50
	}
	e.textAreaHeight = rect.Height
	if e.textAreaHeight < 50 {
		e.textAreaHeight = 50
	}

	e.totalWidth = e.calcMaxLineWidth() + 20
	e.totalHeight = e.calcTotalHeight()

	e.scrollMaxY = e.totalHeight - e.textAreaHeight
	if e.scrollMaxY < 0 {
		e.scrollMaxY = 0
	} else {
		hasVScroll = true
	}
	e.scrollMaxX = e.totalWidth - e.textAreaWidth
	if e.scrollMaxX < 0 {
		e.scrollMaxX = 0
	} else {
		hasHScroll = true
	}

	if hasVScroll {
		e.textAreaWidth -= sbSize
	}
	if hasHScroll {
		e.textAreaHeight -= sbSize
	}
	if hasVScroll {
		e.scrollMaxX = e.totalWidth - e.textAreaWidth
		if e.scrollMaxX < 0 {
			e.scrollMaxX = 0
		}
	}

	if e.ScrollY > e.scrollMaxY {
		e.ScrollY = e.scrollMaxY
	}
	if e.ScrollY < 0 {
		e.ScrollY = 0
	}
	if e.ScrollX > e.scrollMaxX {
		e.ScrollX = e.scrollMaxX
	}
	if e.ScrollX < 0 {
		e.ScrollX = 0
	}

	if hasVScroll {
		vsbRect := graphics.NewRect(rect.X+rect.Width-sbSize, rect.Y, sbSize, rect.Height)
		if hasHScroll {
			vsbRect.Height -= sbSize
		}
		e.VScroll.SetFrameRect(vsbRect)
		e.VScroll.SetMin(0)
		e.VScroll.SetMax(e.scrollMaxY + e.textAreaHeight)
		e.VScroll.VisibleAmount = e.textAreaHeight
		e.VScroll.SetValue(e.ScrollY)
		e.VScroll.SetVisible(true)
	} else {
		e.VScroll.SetVisible(false)
	}
	if hasHScroll {
		hsbRect := graphics.NewRect(rect.X, rect.Y+rect.Height-sbSize, rect.Width, sbSize)
		if hasVScroll {
			hsbRect.Width -= sbSize
		}
		e.HScroll.SetFrameRect(hsbRect)
		e.HScroll.SetMin(0)
		e.HScroll.SetMax(e.scrollMaxX + e.textAreaWidth)
		e.HScroll.VisibleAmount = e.textAreaWidth
		e.HScroll.SetValue(e.ScrollX)
		e.HScroll.SetVisible(true)
	} else {
		e.HScroll.SetVisible(false)
	}

	drawFunc := func(c *graphics.Canvas, r graphics.Rect) {
		e.drawBackground(c, r)
		e.drawLineNumbers(c, r)
		e.drawTextArea(c, r)
		if e.ShowMinimap {
			e.drawMinimap(c, r)
		}
		e.drawScrollIndicators(c, r)

		if hasVScroll {
			vsbRect := graphics.NewRect(r.X+r.Width-sbSize, r.Y, sbSize, r.Height)
			if hasHScroll {
				vsbRect.Height -= sbSize
			}
			c.Save()
			c.Translate(vsbRect.X-r.X, vsbRect.Y-r.Y)
			e.VScroll.Paint(c, graphics.NewRect(0, 0, vsbRect.Width, vsbRect.Height))
			c.Restore()
		}
		if hasHScroll {
			hsbRect := graphics.NewRect(r.X, r.Y+r.Height-sbSize, r.Width, sbSize)
			if hasVScroll {
				hsbRect.Width -= sbSize
			}
			c.Save()
			c.Translate(hsbRect.X-r.X, hsbRect.Y-r.Y)
			e.HScroll.Paint(c, graphics.NewRect(0, 0, hsbRect.Width, hsbRect.Height))
			c.Restore()
		}
	}
	// End drawFunc

	// Use offscreen cache during inactivity for efficient cursor blink.
	// Draw directly during active interaction for immediate updates.
	if e.interacting {
		e.cache.Invalidate()
		drawFunc(canvas, rect)
	} else {
		e.cache.GetCached(canvas, rect, drawFunc)
	}

	// Draw cursor outside the cached image so it can blink and move without
	// invalidating the offscreen buffer.
	if e.Focused && e.CursorVisible {
		if e.CursorLine >= 0 && e.CursorLine < len(e.Lines) {
			f := e.font()
			line := e.Lines[e.CursorLine]
			textAreaX := rect.X + e.gutterWidth
			cursorX := textAreaX + lineMeasure{text: line, font: f, tabWidth: e.TabWidth}.caretX(e.CursorCol) - e.ScrollX
			cursorY := rect.Y + e.visibleLineY(e.CursorLine) - e.ScrollY
			cursorPaint := graphics.NewPaint()
			cursorPaint.SetColor(e.Cursor)
			cursorPaint.SetStrokeWidth(2)
			canvas.DrawLine(cursorX, cursorY, cursorX, cursorY+e.lineHeight, cursorPaint)
		}
	}

	// Draw selection overlay outside the cached image so it updates immediately
	// during mouse drag without requiring a full cache rebuild.
	if e.Focused && e.SelectStart != e.SelectEnd {
		f := e.font()
		textAreaX := rect.X + e.gutterWidth
		selS, selE := e.SelectStart, e.SelectEnd
		if selS > selE {
			selS, selE = selE, selS
		}
		startLine, startCol := e.offsetToLineCol(selS)
		endLine, endCol := e.offsetToLineCol(selE)
		if startLine < 0 {
			startLine = 0
		}
		if endLine >= len(e.Lines) {
			endLine = len(e.Lines) - 1
		}
		lh := e.lineHeight
		selBg := graphics.NewPaint()
		selBg.SetColor(e.SelectionBg)
		for line := startLine; line <= endLine; line++ {
			if line < 0 || line >= len(e.Lines) {
				continue
			}
			l := e.Lines[line]
			colStart := 0
			colEnd := len(l)
			if line == startLine {
				colStart = startCol
			}
			if line == endLine {
				colEnd = endCol
			}
			if colStart < 0 {
				colStart = 0
			}
			if colEnd > len(l) {
				colEnd = len(l)
			}
			if colStart >= colEnd {
				continue
			}
			lm := lineMeasure{text: l, font: f, tabWidth: e.TabWidth}
			selX := textAreaX + lm.caretX(colStart) - e.ScrollX
			selW := lm.caretX(colEnd) - lm.caretX(colStart)
			selY := rect.Y + e.visibleLineY(line) - e.ScrollY
			canvas.DrawRect(graphics.NewRect(selX, selY, selW, lh), selBg)
		}
	}

	canvas.Restore()
}

func (e *CodeEditor) drawBackground(canvas *graphics.Canvas, rect graphics.Rect) {
	bgPaint := graphics.NewPaint()
	bgPaint.SetColor(e.Background)
	cr := theme.Shape().Input
	canvas.DrawRoundedRect(rect, cr, cr, bgPaint)

	gutterPaint := graphics.NewPaint()
	gutterPaint.SetColor(e.GutterBackground)
	gutterRect := graphics.NewRect(rect.X, rect.Y, e.gutterWidth, rect.Height)
	canvas.DrawRect(gutterRect, gutterPaint)

	borderPaint := graphics.NewPaint()
	borderPaint.SetStyle(paintstyle.Stroke)
	borderPaint.SetColor(e.GutterBorder)
	borderPaint.SetStrokeWidth(1)
	canvas.DrawLine(e.gutterWidth, rect.Y, e.gutterWidth, rect.Y+rect.Height, borderPaint)
}

func (e *CodeEditor) drawLineNumbers(canvas *graphics.Canvas, rect graphics.Rect) {
	if !e.ShowLineNumbers {
		return
	}

	f := e.font()
	ascent := f.Baseline()

	startLine := e.sourceLineAtY(e.ScrollY)
	estEnd := e.sourceLineAtY(e.ScrollY + e.textAreaHeight)
	endLine := estEnd + 2
	if endLine > len(e.Lines) {
		endLine = len(e.Lines)
	}
	lh := e.lineHeight

	for line := startLine; line < endLine; line++ {
		if e.isFoldedInner(line) {
			continue
		}
		y := e.visibleLineY(line) - e.ScrollY
		if e.codeFolding[line] {
			if end, ok := e.foldRanges[line]; ok {
				textPaint := graphics.NewPaint()
				if line == e.CursorLine {
					textPaint.SetColor(e.LineNumberActive)
				} else {
					textPaint.SetColor(e.GutterText)
				}
				numStr := fmt.Sprintf("%d", line+1)
				textW := f.SimpleWidth(numStr)
				x := e.gutterWidth - textW - 8
				canvas.DrawSimpleString(numStr, rect.X+x, rect.Y+y+ascent, f, textPaint)

				chevD := graphics.NewPaint()
				chevD.SetStyle(paintstyle.Stroke)
				chevD.SetStrokeWidth(1.5)
				chevD.SetColor(graphics.Color(0xFF858585))
				cdx := rect.X + 30
				cdy := rect.Y + y + lh/2
				canvas.DrawLine(cdx-3, cdy-2, cdx+3, cdy-2, chevD)
				canvas.DrawLine(cdx-3, cdy-2, cdx, cdy+3, chevD)
				canvas.DrawLine(cdx, cdy+3, cdx+3, cdy-2, chevD)
				elp := graphics.NewPaint()
				elp.SetColor(graphics.Color(0xFF808080))
				canvas.DrawSimpleString("...  ", rect.X+e.gutterWidth+4, rect.Y+y+ascent, f, elp)

				line = end
				continue
			}
		}
		y = e.visibleLineY(line) - e.ScrollY
		textPaint := graphics.NewPaint()
		if line == e.CursorLine {
			textPaint.SetColor(e.LineNumberActive)
		} else {
			textPaint.SetColor(e.GutterText)
		}
		numStr := fmt.Sprintf("%d", line+1)
		textW := f.SimpleWidth(numStr)
		x := e.gutterWidth - textW - 4
		canvas.DrawSimpleString(numStr, rect.X+x, rect.Y+y+ascent, f, textPaint)

		// Diff gutter markers (x=0..3, thin colored bar)
		if e.diffAdded[line] {
			dp := graphics.NewPaint()
			dp.SetColor(graphics.Color(0xFF4CAF50))
			canvas.DrawRect(graphics.NewRect(rect.X, rect.Y+y, 3, lh), dp)
		} else if e.diffModified[line] {
			dp := graphics.NewPaint()
			dp.SetColor(graphics.Color(0xFFFFC107))
			canvas.DrawRect(graphics.NewRect(rect.X, rect.Y+y, 3, lh), dp)
		} else if e.diffDeleted[line] {
			dp := graphics.NewPaint()
			dp.SetColor(graphics.Color(0xFFF44336))
			dp.SetStrokeWidth(1)
			midY := rect.Y + y + lh/2
			canvas.DrawLine(rect.X, midY, rect.X+8, midY, dp)
		}

		// Breakpoint lane (x=3..15)
		if e.breakpoints[line] {
			if e.breakpointSVG != nil {
				meui.DrawSVG(canvas, e.breakpointSVG, rect.X+1, rect.Y+y+(lh-10)/2, 10, 10)
			} else {
				bp := graphics.NewPaint()
				bp.SetColor(graphics.Color(0xFFE53935))
				canvas.DrawCircle(rect.X+6, rect.Y+y+lh/2, 5, bp)
			}
		}

		// Bookmark lane (x=15..27)
		if e.bookmarks[line] {
			bmX := rect.X + 16
			bmY := rect.Y + y + (lh-10)/2 + 1
			// Draw a simple flag shape
			fg := graphics.NewPaint()
			fg.SetColor(graphics.Color(0xFF42A5F5))
			fg.SetStrokeWidth(0)
			// flag pole
			canvas.DrawRect(graphics.NewRect(bmX, bmY, 2, 9), fg)
			// flag triangle
			fg2 := graphics.NewPaint()
			fg2.SetColor(graphics.Color(0xFF64B5F6))
			canvas.DrawRect(graphics.NewRect(bmX+2, bmY, 6, 4), fg2)
		}

		// Fold lane (x=27..41) — chevron drawn directly: a nested
		// RenderOffscreen (DrawSVG) inside this offscreen cache steals the
		// editor's pending batched draws (lost into the nested viewport).
		if e.hasFoldableBlock(line) && !e.codeFolding[line] {
			chev := graphics.NewPaint()
			chev.SetStyle(paintstyle.Stroke)
			chev.SetStrokeWidth(1.5)
			chev.SetColor(graphics.Color(0xFF858585))
			cx := rect.X + 30
			cy := rect.Y + y + lh/2
			if line == e.CursorLine {
				chev.SetColor(graphics.Color(0xFFC6C6C6))
			}
			canvas.DrawLine(cx-3, cy-3, cx+3, cy, chev)
			canvas.DrawLine(cx+3, cy, cx-3, cy+3, chev)
		}

		// Diagnostic icons + lightbulb (right side of gutter, before line numbers)
		hasDiag := false
		diagSev := 0
		for _, d := range e.diagnostics {
			if d.Line != line {
				continue
			}
			hasDiag = true
			diagSev = d.Severity
			diagCol := graphics.Color(0xFF808080)
			switch diagSev {
			case 1:
				diagCol = graphics.Color(0xFFE53935)
			case 2:
				diagCol = graphics.Color(0xFFFFC107)
			}
			dp2 := graphics.NewPaint()
			dp2.SetColor(diagCol)
			canvas.DrawCircle(rect.X+e.gutterWidth-8, rect.Y+y+lh/2, 4, dp2)
			break
		}
		// Lightbulb when diagnostics exist on this line (code action hint)
		if hasDiag {
			lb := graphics.NewPaint()
			lb.SetColor(graphics.Color(0xFFFFF176))
			canvas.DrawCircle(rect.X+e.gutterWidth-22, rect.Y+y+lh/2, 3, lb)
		}
	}
}

func (e *CodeEditor) hasFoldableBlock(line int) bool {
	if line < 0 || line >= len(e.Lines) {
		return false
	}
	return strings.Contains(e.Lines[line], "{")
}

func (e *CodeEditor) isFoldedInner(line int) bool {
	for start, end := range e.foldRanges {
		if line > start && line <= end && e.codeFolding[start] {
			return true
		}
	}
	return false
}

// visibleLineY returns the Y position (in pixels) of a source line accounting for folds
func (e *CodeEditor) visibleLineY(line int) float32 {
	y := float32(0)
	for i := 0; i < line && i < len(e.Lines); i++ {
		if e.isFoldedInner(i) {
			continue
		}
		y += e.lineHeight
	}
	return y
}

// totalVisibleLines counts visible lines
func (e *CodeEditor) totalVisibleLines() int {
	count := 0
	for i := 0; i < len(e.Lines); i++ {
		if !e.isFoldedInner(i) {
			count++
		}
	}
	return count
}

// sourceLineAtY converts a visible Y coordinate back to the source line index
func (e *CodeEditor) sourceLineAtY(y float32) int {
	remaining := y
	for i := 0; i < len(e.Lines); i++ {
		if e.isFoldedInner(i) {
			continue
		}
		if remaining < e.lineHeight {
			return i
		}
		remaining -= e.lineHeight
	}
	return len(e.Lines) - 1
}

func (e *CodeEditor) drawTextArea(canvas *graphics.Canvas, rect graphics.Rect) {
	f := e.font()
	lh := e.lineHeight
	ascent := f.Baseline()

	textAreaRect := graphics.NewRect(
		rect.X+e.gutterWidth,
		rect.Y,
		e.textAreaWidth,
		rect.Height,
	)

	canvas.Save()
	if os.Getenv("MEUI_NO_CLIP") == "" {
		canvas.ClipRect(textAreaRect, pathop.Intersect, false)
	}
	if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
		dbg := graphics.NewPaint()
		dbg.SetColor(graphics.Color(0xFF00FF00))
		canvas.DrawRect(graphics.NewRect(textAreaRect.X+60, textAreaRect.Y+8, 300, 14), dbg)
	}

	startLine := e.sourceLineAtY(e.ScrollY)
	estEnd := e.sourceLineAtY(e.ScrollY + e.textAreaHeight)
	endLine := estEnd + 2
	if endLine > len(e.Lines) {
		endLine = len(e.Lines)
	}
	if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
		fmt.Printf("[codeDraw] lines=%d start=%d end=%d lineHeight=%.1f textArea=(%.0f,%.0f %.0fx%.0f)\n",
			len(e.Lines), startLine, endLine, lh, textAreaRect.X, textAreaRect.Y, textAreaRect.Width, textAreaRect.Height)
	}

	// Draw indentation guides (vertical lines at each indent level)
	e.drawIndentGuides(canvas, textAreaRect, rect, startLine, endLine, lh, f)

	if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
		fmt.Printf("[codeDraw] lines=%d start=%d end=%d lineHeight=%.1f textArea=(%.0f,%.0f %.0fx%.0f)\n",
			len(e.Lines), startLine, endLine, lh, textAreaRect.X, textAreaRect.Y, textAreaRect.Width, textAreaRect.Height)
	}
	for line := startLine; line < endLine; line++ {
		if e.isFoldedInner(line) {
			continue
		}
		y := e.visibleLineY(line) - e.ScrollY
		if e.codeFolding[line] {
			if end, ok := e.foldRanges[line]; ok {
				lineRect := graphics.NewRect(textAreaRect.X, rect.Y+y, e.textAreaWidth, lh)
				if line == e.CursorLine && e.Focused {
					lbg := graphics.NewPaint()
					lbg.SetColor(e.CursorLineBg)
					canvas.DrawRect(lineRect, lbg)
				}
				e.drawLineDiagnostics(canvas, textAreaRect, line, y, lh)
				e.drawLineText(canvas, textAreaRect, line, y, lh, ascent, f)
				line = end
				continue
			}
		}
		y = e.visibleLineY(line) - e.ScrollY
		lineRect := graphics.NewRect(textAreaRect.X, rect.Y+y, e.textAreaWidth, lh)

		if line == e.CursorLine && e.Focused {
			lineBgPaint := graphics.NewPaint()
			lineBgPaint.SetColor(e.CursorLineBg)
			canvas.DrawRect(lineRect, lineBgPaint)
		}

		e.drawLineDiagnostics(canvas, textAreaRect, line, y, lh)
		e.drawLineText(canvas, textAreaRect, line, y, lh, ascent, f)
	}

	e.drawIndentGuides(canvas, textAreaRect, rect, startLine, endLine, lh, f)
	e.drawBlockGuides(canvas, textAreaRect, rect)
	e.drawBracketMatch(canvas, textAreaRect)
	e.drawSearchMatches(canvas, textAreaRect)

	if e.completionVisible {
		e.drawCompletionPopup(canvas, rect)
	}
	if e.parameterHintVisible {
		e.drawParameterHints(canvas, rect)
	}
	if e.hoverVisible {
		e.drawHover(canvas, rect)
	}
	if e.contextMenuVisible {
		e.drawContextMenu(canvas, rect)
	}

	canvas.Restore()
}

func (e *CodeEditor) drawIndentGuides(canvas *graphics.Canvas, textAreaRect, rect graphics.Rect, startLine, endLine int, lh float32, f graphics.Font) {
	if endLine > len(e.Lines) {
		endLine = len(e.Lines)
	}
	maxIndent := 0
	for line := startLine; line < endLine; line++ {
		if e.isFoldedInner(line) {
			continue
		}
		ln := e.Lines[line]
		indent := 0
		for _, ch := range ln {
			if ch == '\t' {
				indent += e.TabWidth
			} else if ch == ' ' {
				indent++
			} else {
				break
			}
		}
		if indent > maxIndent {
			maxIndent = indent
		}
	}
	if maxIndent < e.TabWidth {
		return
	}

	for level := 1; level*e.TabWidth <= maxIndent; level++ {
		pos := level * e.TabWidth
		x := textAreaRect.X + float32(pos)*e.charWidth - e.ScrollX
		if x < textAreaRect.X || x > textAreaRect.X+textAreaRect.Width {
			continue
		}
		// Draw full-height vertical line for this indent level
		topY := rect.Y + e.visibleLineY(startLine) - e.ScrollY
		botY := rect.Y + e.visibleLineY(endLine-1) - e.ScrollY + lh
		if topY < textAreaRect.Y {
			topY = textAreaRect.Y
		}
		if botY > textAreaRect.Y+textAreaRect.Height {
			botY = textAreaRect.Y + textAreaRect.Height
		}
		guide := graphics.NewPaint()
		guide.SetColor(e.IndentGuide)
		guide.SetStrokeWidth(1)
		canvas.DrawLine(x, topY, x, botY, guide)
	}
}

func (e *CodeEditor) drawBlockGuides(canvas *graphics.Canvas, textAreaRect, rect graphics.Rect) {
	if len(e.foldRanges) == 0 {
		return
	}
	f := e.font()
	lh := e.lineHeight
	for startLine, endLine := range e.foldRanges {
		if endLine <= startLine {
			continue
		}
		if startLine >= len(e.Lines) || endLine >= len(e.Lines) {
			continue
		}
		if e.codeFolding[startLine] {
			continue
		}
		line := e.Lines[startLine]
		firstChar := strings.IndexFunc(line, func(r rune) bool { return r != ' ' && r != '\t' })
		if firstChar < 0 {
			continue
		}
		prefix := expandTabs(line[:firstChar], e.TabWidth)
		guideX := textAreaRect.X + f.SimpleWidth(prefix) - e.ScrollX
		// Start 1 line below the brace line, end 1 line above the closing brace line
		bodyStart := min(startLine+1, endLine-1)
		bodyEnd := max(endLine-1, bodyStart)
		if bodyEnd <= bodyStart {
			continue
		}
		startY := rect.Y + e.visibleLineY(bodyStart) - e.ScrollY
		endY := rect.Y + e.visibleLineY(bodyEnd) - e.ScrollY + lh*0.5
		if endY < rect.Y || startY > rect.Y+rect.Height {
			continue
		}
		startY = max(startY, rect.Y)
		endY = min(endY, rect.Y+rect.Height)
		guide := graphics.NewPaint()
		guide.SetColor(e.IndentGuideActive)
		guide.SetStrokeWidth(1)
		canvas.DrawLine(guideX, startY, guideX, endY, guide)
	}
}

func (e *CodeEditor) drawLineText(canvas *graphics.Canvas, textAreaRect graphics.Rect, lineNum int, y, lh, ascent float32, f graphics.Font) {
	if lineNum >= len(e.Lines) || lineNum >= len(e.lineTokens) {
		return
	}
	line := e.Lines[lineNum]
	if line == "" {
		return
	}

	dispLine := expandTabs(line, e.TabWidth)

	tokens := e.lineTokens[lineNum]
	if os.Getenv("MEUI_LAYOUT_DEBUG") != "" && lineNum == 0 {
		fmt.Printf("[lineText] line=%q len=%d tokens=%d", line, len(line), len(tokens))
		for i, tk := range tokens {
			if i >= 4 {
				break
			}
			fmt.Printf(" {s=%d e=%d t=%d}", tk.Start, tk.End, tk.Type)
		}
		fmt.Println()
	}
	xOffset := -e.ScrollX
	baseX := textAreaRect.X + xOffset
	baseline := textAreaRect.Y + y + ascent

	if len(tokens) == 0 {
		if os.Getenv("MEUI_LAYOUT_DEBUG") != "" && lineNum == 0 {
			fmt.Printf("[lineText] fallback draw %q at (%.1f,%.1f) color=%08x\n",
				dispLine, baseX, baseline, uint32(e.SyntaxTheme.Default.Color))
		}
		if e.BracketColorization {
			e.drawLineWithBracketColors(canvas, dispLine, line, baseX, baseline, f, lineNum)
		} else {
			textPaint := graphics.NewPaint()
			textPaint.SetColor(e.SyntaxTheme.Default.Color)
			canvas.DrawSimpleString(dispLine, baseX, baseline, f, textPaint)
		}
		return
	}

	selStart, selEnd := e.SelectStart, e.SelectEnd
	if selStart > selEnd {
		selStart, selEnd = selEnd, selStart
	}
	lineStartOffset := e.lineColToOffset(lineNum, 0)
	hasSelection := e.SelectStart != e.SelectEnd && e.Focused

	for _, token := range tokens {
		if token.Start >= len(line) {
			break
		}
		tStart := token.Start
		tEnd := token.End
		if tEnd > len(line) {
			tEnd = len(line)
		}
		if tEnd <= tStart {
			continue
		}

		style := e.tokenStyle(token.Type)
		dispPrefix := expandTabs(line[:tStart], e.TabWidth)
		dispToken := expandTabs(line[tStart:tEnd], e.TabWidth)
		tokenX := baseX + f.SimpleWidth(dispPrefix)

		if hasSelection {
			relStart := selStart - lineStartOffset
			relEnd := selEnd - lineStartOffset
			if relStart < 0 {
				relStart = 0
			}
			if relEnd > len(line) {
				relEnd = len(line)
			}

			if tStart >= relStart && tEnd <= relEnd {
				// Draw selection background for this token
				selBg := graphics.NewPaint()
				selBg.SetColor(e.SelectionBg)
				lm := lineMeasure{text: line, font: f, tabWidth: e.TabWidth}
				selW := lm.caretX(tEnd) - lm.caretX(tStart)
				canvas.DrawRect(graphics.NewRect(tokenX, textAreaRect.Y+y, selW, lh), selBg)
				tp := graphics.NewPaint()
				tp.SetColor(e.SelectionFg)
				canvas.DrawSimpleString(dispToken, tokenX, baseline, f, tp)
			} else if tStart < relEnd && tEnd > relStart {
				dispA := expandTabs(line[tStart:max(relStart, tStart)], e.TabWidth)
				dispB := expandTabs(line[max(relStart, tStart):min(relEnd, tEnd)], e.TabWidth)
				dispC := expandTabs(line[min(relEnd, tEnd):tEnd], e.TabWidth)
				aX := baseX + f.SimpleWidth(dispPrefix)
				bX := aX + f.SimpleWidth(dispA)
				if dispA != "" {
					p1 := graphics.NewPaint()
					p1.SetColor(style.Color)
					canvas.DrawSimpleString(dispA, aX, baseline, f, p1)
				}
				if dispB != "" {
					p2 := graphics.NewPaint()
					p2.SetColor(e.SelectionFg)
					canvas.DrawSimpleString(dispB, bX, baseline, f, p2)
				}
				if dispC != "" {
					p3 := graphics.NewPaint()
					p3.SetColor(style.Color)
					canvas.DrawSimpleString(dispC, bX+f.SimpleWidth(dispB), baseline, f, p3)
				}
			} else {
				tp := graphics.NewPaint()
				tp.SetColor(style.Color)
				canvas.DrawSimpleString(dispToken, tokenX, baseline, f, tp)
			}
		} else {
			// Bracket colorization override
			tp := graphics.NewPaint()
			if e.BracketColorization && len(dispToken) == 1 && strings.ContainsRune("{}[]()", rune(line[tStart])) {
				bp := e.bracketDepthAt(lineNum, tStart)
				tp.SetColor(e.bracketColors[bp%len(e.bracketColors)])
			} else {
				tp.SetColor(style.Color)
			}
			canvas.DrawSimpleString(dispToken, tokenX, baseline, f, tp)
		}
	}
	e.drawWhitespace(canvas, dispLine, line, baseX, baseline, f)
}

func (e *CodeEditor) tokenStyle(ttype TokenType) SyntaxTokenStyle {
	if e.SyntaxTheme == nil {
		return SyntaxTokenStyle{Color: e.EditorTheme.Cursor}
	}
	switch ttype {
	case TokenDefault:
		return e.SyntaxTheme.Default
	case TokenKeyword:
		return e.SyntaxTheme.Keyword
	case TokenString:
		return e.SyntaxTheme.String
	case TokenNumber:
		return e.SyntaxTheme.Number
	case TokenComment:
		return e.SyntaxTheme.Comment
	case TokenTypeName:
		return e.SyntaxTheme.Type
	case TokenFunction:
		return e.SyntaxTheme.Function
	case TokenVariable:
		return e.SyntaxTheme.Variable
	case TokenConstant:
		return e.SyntaxTheme.Constant
	case TokenOperator:
		return e.SyntaxTheme.Operator
	case TokenPunctuation:
		return e.SyntaxTheme.Punctuation
	case TokenAnnotation:
		return e.SyntaxTheme.Annotation
	case TokenPreprocessor:
		return e.SyntaxTheme.Preprocessor
	case TokenRegex:
		return e.SyntaxTheme.Regex
	case TokenEscape:
		return e.SyntaxTheme.Escape
	case TokenParameter:
		return e.SyntaxTheme.Parameter
	case TokenProperty:
		return e.SyntaxTheme.Property
	case TokenTag:
		return e.SyntaxTheme.Tag
	case TokenAttribute:
		return e.SyntaxTheme.Attribute
	case TokenError:
		return e.SyntaxTheme.Error
	case TokenWarning:
		return e.SyntaxTheme.Warning
	case TokenInfo:
		return e.SyntaxTheme.Info
	case TokenMarkdownHeading:
		return e.SyntaxTheme.MarkdownHeading
	case TokenMarkdownLink:
		return e.SyntaxTheme.MarkdownLink
	case TokenMarkdownList:
		return e.SyntaxTheme.MarkdownList
	}
	return e.SyntaxTheme.Default
}

func (e *CodeEditor) drawCursor(canvas *graphics.Canvas, textAreaRect graphics.Rect) {
	f := e.font()
	line := e.Lines[e.CursorLine]
	cursorX := textAreaRect.X + lineMeasure{text: line, font: f, tabWidth: e.TabWidth}.caretX(e.CursorCol) - e.ScrollX
	cursorY := textAreaRect.Y + e.visibleLineY(e.CursorLine) - e.ScrollY
	cursorPaint := graphics.NewPaint()
	cursorPaint.SetColor(e.Cursor)
	cursorPaint.SetStrokeWidth(2)
	canvas.DrawLine(cursorX, cursorY, cursorX, cursorY+e.lineHeight, cursorPaint)
}

func (e *CodeEditor) drawBracketMatch(canvas *graphics.Canvas, textAreaRect graphics.Rect) {
	if e.bracketMatch.line < 0 || !e.Focused {
		return
	}
	f := e.font()
	drawBracket := func(bp bracketPair) {
		if bp.line < 0 || bp.line >= len(e.Lines) {
			return
		}
		line := e.Lines[bp.line]
		if bp.col < 0 || bp.col >= len(line) {
			return
		}
		chX := textAreaRect.X + lineMeasure{text: line, font: f, tabWidth: e.TabWidth}.caretX(bp.col) - e.ScrollX
		chY := textAreaRect.Y + e.visibleLineY(bp.line) - e.ScrollY
		bgPaint := graphics.NewPaint()
		bgPaint.SetColor(e.BracketMatchBg)
		canvas.DrawRect(graphics.NewRect(chX, chY, f.SimpleWidth(string(bp.ch)), e.lineHeight), bgPaint)
		borderPaint := graphics.NewPaint()
		borderPaint.SetStyle(paintstyle.Stroke)
		borderPaint.SetColor(e.BracketMatchBorder)
		canvas.DrawRect(graphics.NewRect(chX, chY, f.SimpleWidth(string(bp.ch)), e.lineHeight), borderPaint)
	}
	drawBracket(e.bracketMatch)
	drawBracket(e.bracketMatchEnd)
}

func (e *CodeEditor) drawSearchMatches(canvas *graphics.Canvas, textAreaRect graphics.Rect) {
	if !e.searchVisible || len(e.searchMatches) == 0 {
		return
	}
	f := e.font()
	for idx, m := range e.searchMatches {
		if m.line < 0 || m.line >= len(e.Lines) {
			continue
		}
		line := e.Lines[m.line]
		if m.start > len(line) || m.end > len(line) {
			continue
		}
		lm := lineMeasure{text: line, font: f, tabWidth: e.TabWidth}
		startX := textAreaRect.X + lm.caretX(m.start) - e.ScrollX
		endX := textAreaRect.X + lm.caretX(m.end) - e.ScrollX
		y := textAreaRect.Y + e.visibleLineY(m.line) - e.ScrollY
		bgPaint := graphics.NewPaint()
		if idx == e.searchCurrentIdx {
			bgPaint.SetColor(e.SearchMatchCurrent)
		} else {
			bgPaint.SetColor(e.SearchMatchBg)
		}
		canvas.DrawRect(graphics.NewRect(startX, y, endX-startX, e.lineHeight), bgPaint)
	}
}

func (e *CodeEditor) drawLineDiagnostics(canvas *graphics.Canvas, textAreaRect graphics.Rect, lineNum int, y, lh float32) {
	for _, d := range e.diagnostics {
		if d.Line != lineNum {
			continue
		}
		col := d.Col
		endCol := d.EndCol
		if endCol <= col {
			endCol = col + 1
		}
		line := e.Lines[lineNum]
		f := e.font()
		lm := lineMeasure{text: line, font: f, tabWidth: e.TabWidth}
		startX := textAreaRect.X + lm.caretX(col) - e.ScrollX
		endX := textAreaRect.X + lm.caretX(endCol) - e.ScrollX
		if endX < startX {
			endX = startX + f.SimpleWidth(" ")
		}

		var lineColor graphics.Color
		switch d.Severity {
		case 1:
			lineColor = graphics.Color(0xFFFF4444)
		case 2:
			lineColor = graphics.Color(0xFFFFAA00)
		default:
			lineColor = graphics.Color(0xFF44AAFF)
		}

		// Draw wavy underline
		wp := graphics.NewPaint()
		wp.SetColor(lineColor)
		wp.SetStrokeWidth(1)
		baseY := textAreaRect.Y + y + lh - 3
		x := startX
		amp := float32(2.0)
		freq := float32(4.0)
		for x < endX {
			nextX := x + freq
			if nextX > endX {
				nextX = endX
			}
			midX := (x + nextX) / 2
			canvas.DrawLine(x, baseY, midX, baseY-amp, wp)
			canvas.DrawLine(midX, baseY-amp, nextX, baseY, wp)
			x = nextX
		}
	}
}

func (e *CodeEditor) drawMinimap(canvas *graphics.Canvas, rect graphics.Rect) {
	if !e.ShowMinimap || len(e.Lines) == 0 {
		return
	}
	minimapX := rect.X + rect.Width - e.minimapWidth
	minimapRect := graphics.NewRect(minimapX, rect.Y, e.minimapWidth, rect.Height)
	bgPaint := graphics.NewPaint()
	bgPaint.SetColor(e.MinimapBackground)
	canvas.DrawRect(minimapRect, bgPaint)
	scale := rect.Height / e.totalHeight
	if scale > 1 {
		scale = 1
	}
	miniCharW := float32(1.5)
	startLine := 0
	for i := range e.Lines {
		y := float32(i) * scale
		if y > rect.Height {
			break
		}
		miniColor := e.SyntaxTheme.Default.Color
		miniPaint := graphics.NewPaint()
		miniPaint.SetColor(miniColor)
		lineLen := len(e.Lines[i])
		if lineLen > 0 {
			canvas.DrawRect(graphics.NewRect(minimapX+1, rect.Y+y, float32(lineLen)*miniCharW, max(scale, 1)), miniPaint)
		}
		startLine = i
	}
	_ = startLine
	sliderHeight := max(rect.Height*(e.textAreaHeight/e.totalHeight), 10)
	sliderY := rect.Y + (e.ScrollY/e.totalHeight)*rect.Height
	sliderPaint := graphics.NewPaint()
	sliderPaint.SetColor(e.MinimapSlider)
	canvas.DrawRect(graphics.NewRect(minimapX, sliderY, e.minimapWidth, sliderHeight), sliderPaint)
}

func (e *CodeEditor) drawScrollIndicators(canvas *graphics.Canvas, rect graphics.Rect) {
	if e.scrollMaxY <= 0 {
		return
	}
	if e.ScrollY > 0 {
		scrollUpPaint := graphics.NewPaint()
		scrollUpPaint.SetColor(graphics.Color(0x33FFFFFF))
		scrollUpRect := graphics.NewRect(rect.X+e.gutterWidth, rect.Y, e.textAreaWidth, 3)
		canvas.DrawRect(scrollUpRect, scrollUpPaint)
	}
	if e.ScrollY < e.scrollMaxY {
		scrollDownPaint := graphics.NewPaint()
		scrollDownPaint.SetColor(graphics.Color(0x33FFFFFF))
		scrollDownRect := graphics.NewRect(rect.X+e.gutterWidth, rect.Y+rect.Height-3, e.textAreaWidth, 3)
		canvas.DrawRect(scrollDownRect, scrollDownPaint)
	}
}

func (e *CodeEditor) completionPopupRect(rect graphics.Rect) (x, y, w, h float32) {
	if len(e.completionItems) == 0 {
		return 0, 0, 0, 0
	}
	w = float32(300)
	h = float32(len(e.completionItems)) * 20
	if h > 200 {
		h = 200
	}

	// Position relative to current cursor
	f := e.font()
	line := e.Lines[e.CursorLine]
	cx := lineMeasure{text: line, font: f, tabWidth: e.TabWidth}.caretX(e.CursorCol) - e.ScrollX + float32(e.gutterWidth)
	cy := e.visibleLineY(e.CursorLine) - e.ScrollY

	x = cx
	y = cy + e.lineHeight
	if x+w > rect.X+rect.Width {
		x = rect.X + rect.Width - w - 4
	}
	if y+h > rect.Y+rect.Height {
		y = cy - h
	}
	if y < rect.Y {
		y = rect.Y + 4
	}
	return
}

func (e *CodeEditor) drawCompletionPopup(canvas *graphics.Canvas, rect graphics.Rect) {
	if len(e.completionItems) == 0 {
		return
	}

	popupX, popupY, popupW, popupH := e.completionPopupRect(rect)
	bgPaint := graphics.NewPaint()
	bgPaint.SetColor(graphics.Color(0xFF2D2D2D))
	canvas.DrawRect(graphics.NewRect(popupX, popupY, popupW, popupH), bgPaint)
	borderPaint := graphics.NewPaint()
	borderPaint.SetStyle(paintstyle.Stroke)
	borderPaint.SetColor(graphics.Color(0xFF4A4A4A))
	canvas.DrawRect(graphics.NewRect(popupX, popupY, popupW, popupH), borderPaint)
	f := e.font()
	ascent := f.Baseline()
	startIdx := e.completionIdx - 5
	if startIdx < 0 {
		startIdx = 0
	}
	visibleCount := int(popupH / 20)
	if startIdx+visibleCount > len(e.completionItems) {
		visibleCount = len(e.completionItems) - startIdx
		if visibleCount < 0 {
			visibleCount = 0
		}
	}
	for i := 0; i < visibleCount && startIdx+i < len(e.completionItems); i++ {
		item := e.completionItems[startIdx+i]
		y := popupY + float32(i)*20
		if startIdx+i == e.completionIdx {
			selPaint := graphics.NewPaint()
			selPaint.SetColor(graphics.Color(0xFF094771))
			canvas.DrawRect(graphics.NewRect(popupX, y, popupW, 20), selPaint)
		}
		textPaint := graphics.NewPaint()
		if startIdx+i == e.completionIdx {
			textPaint.SetColor(graphics.Color(0xFFFFFFFF))
		} else {
			textPaint.SetColor(graphics.Color(0xFFD4D4D4))
		}
		canvas.DrawSimpleString(item.Label, popupX+4, y+ascent+2, f, textPaint)
		if item.Detail != "" {
			detailPaint := graphics.NewPaint()
			detailPaint.SetColor(graphics.Color(0xFFA0A0A0))
			detailW := f.SimpleWidth(item.Detail)
			canvas.DrawSimpleString(item.Detail, popupX+popupW-detailW-4, y+ascent+2, f, detailPaint)
		}
	}
}

func (e *CodeEditor) drawParameterHints(canvas *graphics.Canvas, rect graphics.Rect) {
	if len(e.parameterHints) == 0 {
		return
	}
	f := e.font()
	label := e.parameterHints[0].Label
	hintW := f.SimpleWidth(label) + 20
	hintH := float32(40)
	hintX := e.textAreaWidth/2 - hintW/2 + e.gutterWidth
	hintY := rect.Y + 4
	bgPaint := graphics.NewPaint()
	bgPaint.SetColor(graphics.Color(0xFF2D2D2D))
	canvas.DrawRect(graphics.NewRect(hintX, hintY, hintW, hintH), bgPaint)
	borderPaint := graphics.NewPaint()
	borderPaint.SetStyle(paintstyle.Stroke)
	borderPaint.SetColor(graphics.Color(0xFF4A4A4A))
	canvas.DrawRect(graphics.NewRect(hintX, hintY, hintW, hintH), borderPaint)
	textPaint := graphics.NewPaint()
	textPaint.SetColor(graphics.Color(0xFFD4D4D4))
	canvas.DrawSimpleString(label, hintX+8, hintY+f.Baseline()+4, f, textPaint)
}

func (e *CodeEditor) drawContextMenu(canvas *graphics.Canvas, rect graphics.Rect) {
	if !e.contextMenuVisible || len(e.contextMenuItems) == 0 {
		return
	}
	f := e.font()
	ascent := f.Baseline()
	lh := f.LineHeight() + 6
	itemH := lh + 2
	totalH := float32(0)
	for _, item := range e.contextMenuItems {
		if item.IsSep {
			totalH += 6
			continue
		}
		h := item.Height
		if h <= 0 {
			h = itemH
		}
		totalH += h
	}
	maxW := float32(180)
	for _, item := range e.contextMenuItems {
		if item.CustomPanel != nil {
			fr := item.CustomPanel.FrameRect()
			w := fr.Width + 20
			if w > maxW {
				maxW = w
			}
			continue
		}
		w := f.SimpleWidth(item.Label+item.Shortcut) + 40
		if item.Icon != "" {
			w += 18
		}
		if w > maxW {
			maxW = w
		}
		if len(item.SubMenu) > 0 {
			w += 16
		}
		_ = w
	}

	menuX := e.contextMenuX
	menuY := e.contextMenuY
	if menuX+maxW > rect.X+rect.Width {
		menuX = rect.X + rect.Width - maxW - 4
	}
	if menuY+totalH > rect.Y+rect.Height {
		menuY = rect.Y + rect.Height - totalH - 4
	}

	bg := graphics.NewPaint()
	bg.SetColor(graphics.Color(0xFF2D2D2D))
	canvas.DrawRect(graphics.NewRect(menuX, menuY, maxW, totalH), bg)
	bd := graphics.NewPaint()
	bd.SetStyle(paintstyle.Stroke)
	bd.SetColor(graphics.Color(0xFF4A4A4A))
	canvas.DrawRect(graphics.NewRect(menuX, menuY, maxW, totalH), bd)

	y := menuY
	for i, item := range e.contextMenuItems {
		if item.IsSep {
			sp := graphics.NewPaint()
			sp.SetColor(graphics.Color(0xFF4A4A4A))
			canvas.DrawLine(menuX+8, y+3, menuX+maxW-8, y+3, sp)
			y += 6
			continue
		}
		h := item.Height
		if h <= 0 {
			h = itemH
		}

		// Custom panel items
		if item.CustomPanel != nil {
			if i == e.contextMenuHover {
				hp := graphics.NewPaint()
				hp.SetColor(graphics.Color(0xFF094771))
				canvas.DrawRect(graphics.NewRect(menuX, y, maxW, h), hp)
			}
			item.CustomPanel.SetFrameRect(graphics.NewRect(menuX+4, y+2, maxW-8, h-4))
			item.CustomPanel.Paint(canvas, graphics.NewRect(0, 0, maxW-8, h-4))
			y += h
			continue
		}

		if i == e.contextMenuHover {
			hp := graphics.NewPaint()
			hp.SetColor(graphics.Color(0xFF094771))
			canvas.DrawRect(graphics.NewRect(menuX, y, maxW, h), hp)
		}

		xOff := float32(8)
		// Icon
		if item.Icon != "" {
			if doc, err := meui.LoadSVGByName(item.Icon); err == nil {
				meui.DrawSVG(canvas, doc, menuX+8, y+(h-12)/2, 12, 12)
			} else {
				tp := graphics.NewPaint()
				tp.SetColor(graphics.Color(0xFFE0E0E0))
				canvas.DrawSimpleString(item.Icon, menuX+8, y+ascent+2, f, tp)
			}
			xOff = 24
		}

		tp := graphics.NewPaint()
		tp.SetColor(graphics.Color(0xFFE0E0E0))
		canvas.DrawSimpleString(item.Label, menuX+xOff, y+ascent+2, f, tp)
		if item.Shortcut != "" {
			sp := graphics.NewPaint()
			sp.SetColor(graphics.Color(0xFF808080))
			sw := f.SimpleWidth(item.Shortcut)
			canvas.DrawSimpleString(item.Shortcut, menuX+maxW-sw-8, y+ascent+2, f, sp)
		}
		// Submenu arrow
		if len(item.SubMenu) > 0 {
			arrX := menuX + maxW - 12
			ap := graphics.NewPaint()
			ap.SetColor(graphics.Color(0xFFE0E0E0))
			ap.SetStrokeWidth(1)
			canvas.DrawLine(arrX, y+4, arrX+4, y+h/2, ap)
			canvas.DrawLine(arrX+4, y+h/2, arrX, y+h-4, ap)
		}
		y += h
	}

	// Draw active submenu
	if e.contextMenuSub >= 0 && e.contextMenuSubItems != nil {
		e.drawSubmenu(canvas, rect, menuX+maxW, menuY+float32(e.contextMenuSub)*itemH, e.contextMenuSubItems)
	}
}

func (e *CodeEditor) drawSubmenu(canvas *graphics.Canvas, rect graphics.Rect, x, y float32, items []MenuItem) {
	f := e.font()
	ascent := f.Baseline()
	lh := f.LineHeight() + 6
	itemH := lh + 2
	maxW := float32(160)
	for _, item := range items {
		w := f.SimpleWidth(item.Label+item.Shortcut) + 40
		if w > maxW {
			maxW = w
		}
	}
	totalH := float32(0)
	for _, item := range items {
		if item.IsSep {
			totalH += 6
		} else {
			totalH += itemH
		}
	}

	if x+maxW > rect.X+rect.Width {
		x = rect.X + rect.Width - maxW - 4
	}
	if y+totalH > rect.Y+rect.Height {
		y = rect.Y + rect.Height - totalH - 4
	}

	bg := graphics.NewPaint()
	bg.SetColor(graphics.Color(0xFF2D2D2D))
	canvas.DrawRect(graphics.NewRect(x, y, maxW, totalH), bg)
	bd := graphics.NewPaint()
	bd.SetStyle(paintstyle.Stroke)
	bd.SetColor(graphics.Color(0xFF4A4A4A))
	canvas.DrawRect(graphics.NewRect(x, y, maxW, totalH), bd)

	cy := y
	for _, item := range items {
		if item.IsSep {
			sp := graphics.NewPaint()
			sp.SetColor(graphics.Color(0xFF4A4A4A))
			canvas.DrawLine(x+8, cy+3, x+maxW-8, cy+3, sp)
			cy += 6
			continue
		}
		tp := graphics.NewPaint()
		tp.SetColor(graphics.Color(0xFFE0E0E0))
		canvas.DrawSimpleString(item.Label, x+8, cy+ascent+2, f, tp)
		cy += itemH
	}
}

func (e *CodeEditor) drawHover(canvas *graphics.Canvas, rect graphics.Rect) {
	if e.hoverText == "" {
		return
	}
	f := e.font()
	lines := strings.Split(e.hoverText, "\n")
	maxW := float32(0)
	for _, l := range lines {
		w := f.SimpleWidth(l)
		if w > maxW {
			maxW = w
		}
	}
	hoverW := maxW + 16
	hoverH := float32(len(lines))*e.lineHeight + 8
	hoverX := e.hoverX
	hoverY := e.hoverY - hoverH - 8
	if hoverX+hoverW > rect.X+rect.Width {
		hoverX = rect.X + rect.Width - hoverW
	}
	if hoverY < rect.Y {
		hoverY = e.hoverY + e.lineHeight
	}
	bgPaint := graphics.NewPaint()
	bgPaint.SetColor(graphics.Color(0xFF2D2D2D))
	canvas.DrawRect(graphics.NewRect(hoverX, hoverY, hoverW, hoverH), bgPaint)
	borderPaint := graphics.NewPaint()
	borderPaint.SetStyle(paintstyle.Stroke)
	borderPaint.SetColor(graphics.Color(0xFF4A4A4A))
	canvas.DrawRect(graphics.NewRect(hoverX, hoverY, hoverW, hoverH), borderPaint)
	textPaint := graphics.NewPaint()
	textPaint.SetColor(graphics.Color(0xFFD4D4D4))
	for i, l := range lines {
		canvas.DrawSimpleString(l, hoverX+8, hoverY+float32(i)*e.lineHeight+f.Baseline()+4, f, textPaint)
	}
}

func (e *CodeEditor) bracketDepthAt(line, col int) int {
	depth := 0
	openers := "({["
	closers := ")}]"
	for l := 0; l < line && l < len(e.Lines); l++ {
		for _, ch := range e.Lines[l] {
			if strings.ContainsRune(openers, ch) {
				depth++
			}
			if strings.ContainsRune(closers, ch) {
				depth--
			}
		}
	}
	for c := 0; c < col && c < len(e.Lines[line]); c++ {
		ch := rune(e.Lines[line][c])
		if strings.ContainsRune(openers, ch) {
			depth++
		}
		if strings.ContainsRune(closers, ch) {
			depth--
		}
	}
	if depth < 0 {
		depth = 0
	}
	return depth
}

func (e *CodeEditor) drawLineWithBracketColors(canvas *graphics.Canvas, dispLine, origLine string, baseX, baseline float32, f graphics.Font, lineNum int) {
	x := baseX
	for ci, ch := range dispLine {
		tp := graphics.NewPaint()
		if strings.ContainsRune("{}[]()", ch) {
			bp := e.bracketDepthAt(lineNum, ci)
			tp.SetColor(e.bracketColors[bp%len(e.bracketColors)])
		} else {
			tp.SetColor(e.SyntaxTheme.Default.Color)
		}
		canvas.DrawSimpleString(string(ch), x, baseline, f, tp)
		x += f.SimpleWidth(string(ch))
	}
}

func (e *CodeEditor) drawWhitespace(canvas *graphics.Canvas, dispLine, origLine string, baseX, baseline float32, f graphics.Font) {
	if !e.ShowWhitespace {
		return
	}
	wp := graphics.NewPaint()
	wp.SetColor(e.WhitespaceChar)
	wp.SetStrokeWidth(1)
	// Draw dots for spaces from expanded line
	x := baseX
	for _, ch := range dispLine {
		if ch == ' ' {
			canvas.DrawRect(graphics.NewRect(x+1, baseline-3, 2, 2), wp)
		}
		x += f.SimpleWidth(string(ch))
	}
	// Draw tab arrows from original line
	x2 := baseX
	for _, ch := range origLine {
		if ch == '\t' {
			tabW := e.TabWidth
			tabPx := float32(tabW) * e.charWidth
			arrX := x2 + 2
			arrowY := baseline - 3
			// Draw a small right-pointing arrow (→) for tabs
			canvas.DrawLine(arrX, arrowY, arrX+tabPx-4, arrowY, wp)
			canvas.DrawLine(arrX+tabPx-4, arrowY, arrX+tabPx-6, arrowY-2, wp)
			canvas.DrawLine(arrX+tabPx-4, arrowY, arrX+tabPx-6, arrowY+2, wp)
			x2 += tabPx
		} else {
			x2 += f.SimpleWidth(string(ch))
		}
	}
}

func (e *CodeEditor) calcTotalHeight() float32 {
	visible := 0
	for i := 0; i < len(e.Lines); i++ {
		if e.isFoldedInner(i) {
			continue
		}
		visible++
	}
	return float32(visible) * e.lineHeight
}

func (e *CodeEditor) calcMaxLineWidth() float32 {
	f := e.font()
	maxW := float32(0)
	startLine := e.sourceLineAtY(e.ScrollY)
	estEnd := e.sourceLineAtY(e.ScrollY + e.textAreaHeight)
	endLine := estEnd + 2
	if endLine > len(e.Lines) {
		endLine = len(e.Lines)
	}
	for i := startLine; i < endLine; i++ {
		w := lineMeasure{text: e.Lines[i], font: f, tabWidth: e.TabWidth}.caretX(len(e.Lines[i]))
		if w > maxW {
			maxW = w
		}
	}
	return maxW + 20
}

func (e *CodeEditor) EnsureCursorVisible() {
	cursorX := e.font().SimpleWidth(expandTabs(e.Lines[e.CursorLine][:e.CursorCol], e.TabWidth))
	cursorY := e.visibleLineY(e.CursorLine)

	if cursorX-e.ScrollX < 0 {
		e.ScrollX = cursorX - 20
	} else if cursorX-e.ScrollX > e.textAreaWidth-20 {
		e.ScrollX = cursorX - e.textAreaWidth + 20
	}

	if cursorY-e.ScrollY < 0 {
		e.ScrollY = cursorY - 10
	} else if cursorY+e.lineHeight-e.ScrollY > e.textAreaHeight {
		e.ScrollY = cursorY + e.lineHeight - e.textAreaHeight + 10
	}

	if e.ScrollX < 0 {
		e.ScrollX = 0
	}
	if e.ScrollY < 0 {
		e.ScrollY = 0
	}
	if e.ScrollX > e.scrollMaxX {
		e.ScrollX = e.scrollMaxX
	}
	if e.ScrollY > e.scrollMaxY {
		e.ScrollY = e.scrollMaxY
	}
}

// OnComposition shows the IME's in-progress text at the caret, replacing the previous preedit. It is
// not undoable and does not fire OnTextChanged; the committed characters arrive through OnCharacter.
func (e *CodeEditor) OnComposition(text string, start, length int) bool {
	if !e.Enabled() || !e.Focused || e.ReadOnly {
		return false
	}
	e.cancelComposition()
	if text == "" {
		e.MarkForRedraw()
		return true
	}
	if e.SelectStart != e.SelectEnd {
		lo, hi := e.SelectStart, e.SelectEnd
		if lo > hi {
			lo, hi = hi, lo
		}
		e.removeDocRangeRaw(lo, hi)
	}
	startOff, endOff := e.insertRawAtCaret(text)
	e.composeStart, e.composeEnd = startOff, endOff
	e.composing = true
	if start > 0 {
		off := startOff + start
		if off > endOff {
			off = endOff
		}
		e.CursorLine, e.CursorCol = e.offsetToLineCol(off)
	}
	e.cache.Invalidate()
	e.MarkForRedraw()
	return true
}

// cancelComposition removes the preedit without an undo step or a change notification.
func (e *CodeEditor) cancelComposition() {
	if !e.composing {
		return
	}
	e.composing = false
	lo, hi := e.composeStart, e.composeEnd
	e.composeStart, e.composeEnd = 0, 0
	if hi > len(e.Text()) {
		hi = len(e.Text())
	}
	if lo < hi {
		e.removeDocRangeRaw(lo, hi)
	}
	e.cache.Invalidate()
	e.MarkForRedraw()
}

// insertRawAtCaret inserts text (newlines included) without an undo entry, and returns the byte range
// it occupies.
func (e *CodeEditor) insertRawAtCaret(text string) (int, int) {
	startOff := e.lineColToOffset(e.CursorLine, e.CursorCol)
	line := e.Lines[e.CursorLine]
	col := e.CursorCol
	if col > len(line) {
		col = len(line)
	}
	parts := strings.Split(text, "\n")
	if len(parts) == 1 {
		e.Lines[e.CursorLine] = line[:col] + text + line[col:]
		e.CursorCol = col + len(text)
	} else {
		rest := line[col:]
		head := line[:col] + parts[0]
		newLines := make([]string, 0, len(e.Lines)+len(parts)-1)
		newLines = append(newLines, e.Lines[:e.CursorLine]...)
		newLines = append(newLines, head)
		for i := 1; i < len(parts)-1; i++ {
			newLines = append(newLines, parts[i])
		}
		last := parts[len(parts)-1]
		newLines = append(newLines, last+rest)
		newLines = append(newLines, e.Lines[e.CursorLine+1:]...)
		e.Lines = newLines
		e.CursorLine = e.CursorLine + len(parts) - 1
		e.CursorCol = len(last)
	}
	e.dirtyAll = true
	endOff := e.lineColToOffset(e.CursorLine, e.CursorCol)
	e.SelectStart, e.SelectEnd = endOff, endOff
	e.selectionAnchor = endOff
	return startOff, endOff
}

// removeDocRangeRaw removes a document range without an undo entry (a preedit was never committed).
func (e *CodeEditor) removeDocRangeRaw(lo, hi int) {
	if lo < 0 {
		lo = 0
	}
	if hi > len(e.Text()) {
		hi = len(e.Text())
	}
	if lo >= hi {
		return
	}
	sl, sc := e.offsetToLineCol(lo)
	el, ec := e.offsetToLineCol(hi)
	if sl == el {
		e.Lines[sl] = e.Lines[sl][:sc] + e.Lines[sl][ec:]
	} else {
		e.Lines[sl] = e.Lines[sl][:sc] + e.Lines[el][ec:]
		e.Lines = append(e.Lines[:sl+1], e.Lines[el+1:]...)
	}
	e.CursorLine, e.CursorCol = sl, sc
	off := e.lineColToOffset(sl, sc)
	e.SelectStart, e.SelectEnd = off, off
	e.selectionAnchor = off
	e.dirtyAll = true
	e.cache.Invalidate()
	e.MarkForRedraw()
}

func (e *CodeEditor) OnFocus() {
	e.Focused = true
	e.startCursorBlink()
	e.cache.Invalidate()
	e.MarkForRedraw()
}

func (e *CodeEditor) OnBlur() {
	e.Focused = false
	e.stopCursorBlink()
	e.cache.Invalidate()
	e.MarkForRedraw()
}

func (e *CodeEditor) startCursorBlink() {
	e.CursorVisible = true
	if e.CursorTimer != nil {
		e.CursorTimer.Stop()
	}
	e.CursorTimer = time.AfterFunc(500*time.Millisecond, func() {
		e.CursorVisible = !e.CursorVisible
		platform.InvokeTask(func() {
			e.MarkForRedraw()
			if e.Focused {
				e.startCursorBlink()
			}
		})
	})
}

func (e *CodeEditor) stopCursorBlink() {
	if e.CursorTimer != nil {
		e.CursorTimer.Stop()
		e.CursorTimer = nil
	}
}

// CursorAt reports the I-beam over the text area. The gutter (breakpoint,
// bookmark and fold lanes) and the scrollbars keep the default arrow.
func (e *CodeEditor) CursorAt(x, y float32) *platform.Cursor {
	if !e.Enabled() || !e.Visible() {
		return nil
	}
	if x < e.gutterWidth || x > e.gutterWidth+e.textAreaWidth {
		return nil
	}
	return platform.TextCursor()
}

func (e *CodeEditor) OnMouseDown(x, y float32, button int) bool {
	if !e.Enabled() {
		return false
	}
	e.markInteraction()

	// Context menu item click
	if e.contextMenuVisible && button == 0 {
		f := e.font()
		lh := f.LineHeight() + 6
		baseItemH := lh + 2
		// Compute per-item heights and total
		itemHeights := make([]float32, len(e.contextMenuItems))
		totalH := float32(0)
		for i, it := range e.contextMenuItems {
			if it.IsSep {
				itemHeights[i] = 6
				totalH += 6
				continue
			}
			h := it.Height
			if h <= 0 {
				h = baseItemH
			}
			itemHeights[i] = h
			totalH += h
		}
		maxW := float32(180)
		for _, it := range e.contextMenuItems {
			if it.CustomPanel != nil {
				fr := it.CustomPanel.FrameRect()
				w := fr.Width + 20
				if w > maxW {
					maxW = w
				}
				continue
			}
			w := f.SimpleWidth(it.Label+it.Shortcut) + 40
			if w > maxW {
				maxW = w
			}
		}
		fr := e.FrameRect()
		menuX := e.contextMenuX
		menuY := e.contextMenuY
		if menuX+maxW > fr.Width {
			menuX = fr.Width - maxW - 4
		}
		if menuY+totalH > fr.Height {
			menuY = fr.Height - totalH - 4
		}

		// Function to compute Y position for an item index
		itemY := func(idx int) float32 {
			yPos := menuY
			for j := 0; j < idx; j++ {
				yPos += itemHeights[j]
			}
			return yPos
		}

		// Check submenu click first
		if e.contextMenuSub >= 0 && e.contextMenuSubItems != nil {
			subY := itemY(e.contextMenuSub)
			subTotalH := float32(0)
			for _, it := range e.contextMenuSubItems {
				if it.IsSep {
					subTotalH += 6
					continue
				}
				h := it.Height
				if h <= 0 {
					h = baseItemH
				}
				subTotalH += h
			}
			subMaxW := float32(160)
			for _, it := range e.contextMenuSubItems {
				w2 := f.SimpleWidth(it.Label+it.Shortcut) + 40
				if w2 > subMaxW {
					subMaxW = w2
				}
			}
			subX := menuX + maxW
			if x >= subX && x <= subX+subMaxW && y >= subY && y <= subY+subTotalH {
				// Find sub-item index by Y position
				subIdx := 0
				subYPos := subY
				for j, it := range e.contextMenuSubItems {
					if it.IsSep {
						subYPos += 6
						continue
					}
					h := it.Height
					if h <= 0 {
						h = baseItemH
					}
					if y >= subYPos && y <= subYPos+h {
						subIdx = j
						break
					}
					subYPos += h
				}
				if subIdx >= 0 && subIdx < len(e.contextMenuSubItems) && !e.contextMenuSubItems[subIdx].IsSep {
					e.contextMenuSubItems[subIdx].Action()
				}
				e.contextMenuVisible = false
				e.contextMenuSub = -1
				e.contextMenuSubItems = nil
				e.MarkForRedraw()
				return true
			}
		}

		if x >= menuX && x <= menuX+maxW && y >= menuY && y <= menuY+totalH {
			idx := 0
			yPos := menuY
			for j, it := range e.contextMenuItems {
				if it.IsSep {
					yPos += 6
					continue
				}
				h := itemHeights[j]
				if y >= yPos && y <= yPos+h {
					idx = j
					break
				}
				yPos += h
			}
			if idx >= 0 && idx < len(e.contextMenuItems) && !e.contextMenuItems[idx].IsSep {
				if len(e.contextMenuItems[idx].SubMenu) > 0 {
					if e.contextMenuSub == idx {
						e.contextMenuSub = -1
						e.contextMenuSubItems = nil
					} else {
						e.contextMenuSub = idx
						e.contextMenuSubItems = e.contextMenuItems[idx].SubMenu
					}
					e.MarkForRedraw()
					return true
				}
				e.contextMenuItems[idx].Action()
			}
			e.contextMenuVisible = false
			e.MarkForRedraw()
			return true
		}
		e.contextMenuVisible = false
		e.contextMenuSub = -1
		e.contextMenuSubItems = nil
		e.MarkForRedraw()
	}

	// Right-click → show context menu
	if button == 1 {
		e.contextMenuX = x
		e.contextMenuY = y
		// Build base items
		items := []MenuItem{
			{Label: "Cut", Enabled: true, Shortcut: "Ctrl+X", Action: func() { e.Cut() }},
			{Label: "Copy", Enabled: true, Shortcut: "Ctrl+C", Action: func() { e.Copy() }},
			{Label: "Paste", Enabled: true, Shortcut: "Ctrl+V", Action: func() { e.Paste() }},
			{IsSep: true},
			{Label: "Select All", Enabled: true, Shortcut: "Ctrl+A", Action: func() { e.SelectAll() }},
			{IsSep: true},
			{Label: "Undo", Enabled: true, Shortcut: "Ctrl+Z", Action: func() { e.Undo() }},
			{Label: "Redo", Enabled: true, Shortcut: "Ctrl+Y", Action: func() { e.Redo() }},
			{IsSep: true},
			{Label: "Toggle Comment", Enabled: true, Shortcut: "Ctrl+/", Action: func() { e.ToggleLineComment() }},
		}
		// Allow IDE hook to add/override items based on context
		if e.OnContextMenu != nil {
			selText := ""
			if e.SelectStart != e.SelectEnd {
				s, e2 := e.SelectStart, e.SelectEnd
				if s > e2 {
					s, e2 = e2, s
				}
				selText = e.Text()[s:e2]
			}
			extra := e.OnContextMenu(e.CursorLine, e.CursorCol, selText)
			if len(extra) > 0 {
				all := make([]MenuItem, 0, len(items)+len(extra)+1)
				all = append(all, extra...)
				all = append(all, MenuItem{IsSep: true})
				all = append(all, items...)
				items = all
			}
		}
		e.contextMenuItems = items
		e.contextMenuSub = -1
		e.contextMenuSubItems = nil
		e.contextMenuHover = -1
		e.contextMenuVisible = true
		e.MarkForRedraw()
		return true
	}

	// Route to scrollbars if click is in their area
	fr := e.FrameRect()
	sbSize := e.scrollBarSize
	showVS := e.scrollMaxY > 0
	showHS := e.scrollMaxX > 0
	if showVS && x >= fr.Width-sbSize && x <= fr.Width {
		if showHS && y >= fr.Height-sbSize {
			goto editorClick
		}
		return e.VScroll.OnMouseDown(0, y, button)
	}
	if showHS && y >= fr.Height-sbSize && y <= fr.Height {
		if showVS && x >= fr.Width-sbSize {
			goto editorClick
		}
		return e.HScroll.OnMouseDown(x, 0, button)
	}

editorClick:
	// Gutter click handling by lane
	clickLine := e.sourceLineAtY(y + e.ScrollY)
	if clickLine >= 0 && clickLine < len(e.Lines) {
		if x >= 3 && x < 15 {
			// Breakpoint lane
			if e.breakpoints[clickLine] {
				delete(e.breakpoints, clickLine)
			} else {
				e.breakpoints[clickLine] = true
			}
			e.MarkForRedraw()
			return true
		}
		if x >= 15 && x < 27 {
			// Bookmark lane
			if e.bookmarks[clickLine] {
				delete(e.bookmarks, clickLine)
			} else {
				e.bookmarks[clickLine] = true
			}
			e.MarkForRedraw()
			return true
		}
		if x >= 27 && x < 41 {
			// Fold lane
			if e.hasFoldableBlock(clickLine) {
				e.ToggleFold(clickLine)
				return true
			}
		}
	}

	// Close completion popup on click outside it
	if e.completionVisible {
		_, _, pw, ph := e.completionPopupRect(e.FrameRect())
		if x < 0 || y < 0 || x > pw || y > ph+e.lineHeight+10 {
			e.HideCompletion()
		}
	}

	e.Focused = true
	adjustedX := x - e.gutterWidth + e.ScrollX
	adjustedY := y + e.ScrollY
	if adjustedX < 0 {
		adjustedX = 0
	}
	adjustedY = y + e.ScrollY
	if adjustedX < 0 {
		adjustedX = 0
	}
	clickLine = e.sourceLineAtY(adjustedY)
	if clickLine >= len(e.Lines) {
		clickLine = len(e.Lines) - 1
	}
	if clickLine < 0 {
		clickLine = 0
	}
	clickCol := e.xToCol(clickLine, adjustedX)
	clickOffset := e.lineColToOffset(clickLine, clickCol)

	// Check for Alt+click → column (box) selection
	if button == 0 && e.modAltDown() {
		e.columnSelect = true
		e.colSelStartLine = clickLine
		e.colSelStartCol = clickCol
		e.colSelEndLine = clickLine
		e.colSelEndCol = clickCol
		e.CursorLine = clickLine
		e.CursorCol = clickCol
		e.mouseDown = true
		e.dragSelecting = true
		e.MarkForRedraw()
		return true
	}

	// Ctrl+click → go to definition (if LSP handler set)
	if e.modControlDown() && e.clickCount() == 1 {
		if e.OnGoToDefinition != nil {
			e.OnGoToDefinition(clickLine, clickCol)
		}
		return true
	}

	// Double-click → select word at cursor
	if e.clickCount() >= 2 {
		line := e.Lines[clickLine]
		if len(line) > 0 && clickCol < len(line) {
			start := clickCol
			for start > 0 && (unicode.IsLetter(rune(line[start-1])) || unicode.IsDigit(rune(line[start-1])) || line[start-1] == '_') {
				start--
			}
			end := clickCol
			for end < len(line) && (unicode.IsLetter(rune(line[end])) || unicode.IsDigit(rune(line[end])) || line[end] == '_') {
				end++
			}
			if end > start {
				e.CursorLine = clickLine
				e.CursorCol = end
				e.SelectStart = e.lineColToOffset(clickLine, start)
				e.SelectEnd = e.lineColToOffset(clickLine, end)
				e.selectionAnchor = e.SelectStart
				e.mouseDown = true
				e.MarkForRedraw()
				return true
			}
		}
	}

	// Check if click is within existing selection → start drag
	selS, selE := e.SelectStart, e.SelectEnd
	if selS > selE {
		selS, selE = selE, selS
	}
	if selS != selE && clickOffset >= selS && clickOffset <= selE {
		e.isDragging = true
		e.dragFromLine = e.CursorLine
		e.dragFromCol = e.CursorCol
		e.mouseDown = true
		e.MarkForRedraw()
		return true
	}

	e.CursorLine = clickLine
	e.CursorCol = clickCol
	e.SelectStart = clickOffset
	e.SelectEnd = clickOffset
	e.selectionAnchor = clickOffset
	e.mouseDown = true
	e.dragSelecting = true
	e.columnSelect = false
	e.CursorVisible = true
	e.startCursorBlink()
	e.EnsureCursorVisible()
	e.cache.Invalidate()
	e.MarkForRedraw()

	if e.OnCursorChanged != nil {
		e.OnCursorChanged(e.CursorLine, e.CursorCol)
	}
	return true
}

func (e *CodeEditor) OnMouseUp(x, y float32, button int) bool {
	e.VScroll.OnMouseUp(x, y, button)
	e.HScroll.OnMouseUp(x, y, button)

	if e.isDragging {
		e.isDragging = false
		selS, selE := e.SelectStart, e.SelectEnd
		if selS > selE {
			selS, selE = selE, selS
		}
		if selS != selE {
			dropOffset := e.lineColToOffset(e.CursorLine, e.CursorCol)
			// Dropping within the selection is a no-op — text already there
			if dropOffset >= selS && dropOffset <= selE {
				e.MarkForRedraw()
				return true
			}
			// Save undo snapshot BEFORE modifying text (avoid SetText which clears stacks)
			e.saveUndoState(ActionReplaceText, e.Text(), e.CursorLine, e.CursorCol, e.CursorLine, e.CursorCol)

			fullText := e.Text()
			selected := fullText[selS:selE]
			dropOffset = e.lineColToOffset(e.CursorLine, e.CursorCol)

			// Remove selection
			fullText = fullText[:selS] + fullText[selE:]
			if dropOffset > selE {
				dropOffset -= (selE - selS)
			}

			// Insert at drop position
			fullText = fullText[:dropOffset] + selected + fullText[dropOffset:]
			e.Lines = strings.Split(fullText, "\n")
			if len(e.Lines) == 0 {
				e.Lines = []string{""}
			}

			newLine, newCol := e.offsetToLineCol(dropOffset + len(selected))
			e.CursorLine = newLine
			e.CursorCol = newCol
			e.SelectStart = 0
			e.SelectEnd = 0
			e.dirtyAll = true
			e.Modified = true
			if e.OnTextChanged != nil {
				e.OnTextChanged(fullText)
			}
		}
		e.MarkForRedraw()
		return true
	}

	e.mouseDown = false
	e.dragSelecting = false
	e.columnSelect = false
	return true
}

func (e *CodeEditor) modAltDown() bool {
	mods := e.Modifiers()
	return mods.OptionDown()
}

func (e *CodeEditor) modControlDown() bool {
	mods := e.Modifiers()
	return mods.ControlDown()
}

func (e *CodeEditor) clickCount() int {
	p := e.Parent()
	for p != nil {
		if w, ok := p.(interface{ ClickCount() int }); ok {
			return w.ClickCount()
		}
		if pp, ok := p.(interface{ Parent() meui.Widget }); ok {
			p = pp.Parent()
		} else {
			break
		}
	}
	return 1
}

func (e *CodeEditor) OnMouseMove(x, y float32) {
	e.markInteraction()
	e.VScroll.OnMouseMove(x, y)
	e.HScroll.OnMouseMove(x, y)

	// Context menu hover tracking
	if e.contextMenuVisible {
		f := e.font()
		lh := f.LineHeight() + 6
		itemH := lh + 2
		menuX := e.contextMenuX
		menuY := e.contextMenuY
		if x >= menuX && x <= menuX+180 && y >= menuY && y <= menuY+float32(len(e.contextMenuItems))*itemH {
			idx := int((y - menuY) / itemH)
			if idx != e.contextMenuHover {
				e.contextMenuHover = idx
				e.MarkForRedraw()
			}
		} else if e.contextMenuHover != -1 {
			e.contextMenuHover = -1
			e.MarkForRedraw()
		}
		return
	}

	if e.isDragging {
		// Show drop target cursor
		adjustedX := x - e.gutterWidth + e.ScrollX
		adjustedY := y + e.ScrollY
		e.CursorLine = e.sourceLineAtY(adjustedY)
		if e.CursorLine >= len(e.Lines) {
			e.CursorLine = len(e.Lines) - 1
		}
		if e.CursorLine < 0 {
			e.CursorLine = 0
		}
		e.CursorCol = e.xToCol(e.CursorLine, adjustedX)
		e.EnsureCursorVisible()
		e.MarkForRedraw()
		return
	}

	if e.columnSelect {
		adjustedX := x - e.gutterWidth + e.ScrollX
		adjustedY := y + e.ScrollY
		hoverLine := e.sourceLineAtY(adjustedY)
		if hoverLine >= len(e.Lines) {
			hoverLine = len(e.Lines) - 1
		}
		if hoverLine < 0 {
			hoverLine = 0
		}
		hoverCol := e.xToCol(hoverLine, adjustedX)
		e.colSelEndLine = hoverLine
		e.colSelEndCol = hoverCol
		e.CursorLine = hoverLine
		e.CursorCol = hoverCol
		e.EnsureCursorVisible()
		e.MarkForRedraw()
		return
	}

	if !e.dragSelecting {
		return
	}
	adjustedX := x - e.gutterWidth + e.ScrollX
	adjustedY := y + e.ScrollY
	hoverLine := e.sourceLineAtY(adjustedY)
	if hoverLine >= len(e.Lines) {
		hoverLine = len(e.Lines) - 1
	}
	if hoverLine < 0 {
		hoverLine = 0
	}
	hoverCol := e.xToCol(hoverLine, adjustedX)
	e.CursorLine = hoverLine
	e.CursorCol = hoverCol
	offset := e.lineColToOffset(hoverLine, hoverCol)
	if offset < e.selectionAnchor {
		e.SelectStart = offset
		e.SelectEnd = e.selectionAnchor
	} else {
		e.SelectStart = e.selectionAnchor
		e.SelectEnd = offset
	}
	e.EnsureCursorVisible()
	e.cache.Invalidate()
	e.MarkForRedraw()
}

func (e *CodeEditor) xToCol(line int, x float32) int {
	if line >= len(e.Lines) {
		return 0
	}
	// Shared nearest-boundary rule (textlayout.go), with this editor's tab
	// expansion as the measurement.
	m := lineMeasure{text: e.Lines[line], font: e.font(), tabWidth: e.TabWidth}
	return m.indexAt(x)
}

func (e *CodeEditor) OnMouseWheel(x, y float32, dx, dy float32) bool {
	if !e.Enabled() {
		return false
	}

	scrollAmount := e.lineHeight * 3

	if dy != 0 {
		e.ScrollY -= dy * scrollAmount
		if e.ScrollY < 0 {
			e.ScrollY = 0
		}
		if e.ScrollY > e.scrollMaxY {
			e.ScrollY = e.scrollMaxY
		}
		e.VScroll.SetValue(e.ScrollY)
		e.cache.Invalidate()
		e.MarkForRedraw()
	}
	if dx != 0 {
		e.ScrollX -= dx * 30
		if e.ScrollX < 0 {
			e.ScrollX = 0
		}
		if e.ScrollX > e.scrollMaxX {
			e.ScrollX = e.scrollMaxX
		}
		e.HScroll.SetValue(e.ScrollX)
		e.cache.Invalidate()
		e.MarkForRedraw()
	}
	return true
}

func (e *CodeEditor) pushUndo() {
	e.undoStack = append(e.undoStack, UndoEntry{
		CursorLine: e.CursorLine,
		CursorCol:  e.CursorCol,
		SelStart:   e.SelectStart,
		SelEnd:     e.SelectEnd,
	})
	if len(e.undoStack) > e.maxUndo {
		e.undoStack = e.undoStack[1:]
	}
	e.redoStack = nil
}

func (e *CodeEditor) saveUndoState(action EditAction, text string, startLine, startCol, endLine, endCol int) {
	currentText := strings.Join(e.Lines, "\n")
	e.undoStack = append(e.undoStack, UndoEntry{
		Action:     action,
		Text:       currentText,
		StartLine:  startLine,
		StartCol:   startCol,
		EndLine:    endLine,
		EndCol:     endCol,
		CursorLine: e.CursorLine,
		CursorCol:  e.CursorCol,
		SelStart:   e.SelectStart,
		SelEnd:     e.SelectEnd,
	})
	if len(e.undoStack) > e.maxUndo {
		e.undoStack = e.undoStack[1:]
	}
	e.redoStack = nil
}

func (e *CodeEditor) Undo() bool {
	if len(e.undoStack) == 0 {
		return false
	}
	entry := e.undoStack[len(e.undoStack)-1]
	e.undoStack = e.undoStack[:len(e.undoStack)-1]

	currentText := strings.Join(e.Lines, "\n")
	// Save current state to redo WITHOUT using SetText (which clears stacks)
	e.redoStack = append(e.redoStack, UndoEntry{
		Action:     ActionReplaceText,
		Text:       currentText,
		CursorLine: e.CursorLine,
		CursorCol:  e.CursorCol,
		SelStart:   e.SelectStart,
		SelEnd:     e.SelectEnd,
	})

	// Restore the snapshot
	e.Lines = strings.Split(entry.Text, "\n")
	if len(e.Lines) == 0 {
		e.Lines = []string{""}
	}
	e.CursorLine = entry.CursorLine
	e.CursorCol = entry.CursorCol
	e.SelectStart = entry.SelStart
	e.SelectEnd = entry.SelEnd
	e.dirtyAll = true
	e.Modified = true
	if e.OnTextChanged != nil {
		e.OnTextChanged(entry.Text)
	}
	e.MarkForRedraw()
	return true
}

func (e *CodeEditor) Redo() bool {
	if len(e.redoStack) == 0 {
		return false
	}
	entry := e.redoStack[len(e.redoStack)-1]
	e.redoStack = e.redoStack[:len(e.redoStack)-1]

	currentText := strings.Join(e.Lines, "\n")
	e.undoStack = append(e.undoStack, UndoEntry{
		Action:     ActionReplaceText,
		Text:       currentText,
		CursorLine: e.CursorLine,
		CursorCol:  e.CursorCol,
		SelStart:   e.SelectStart,
		SelEnd:     e.SelectEnd,
	})

	e.Lines = strings.Split(entry.Text, "\n")
	if len(e.Lines) == 0 {
		e.Lines = []string{""}
	}
	e.CursorLine = entry.CursorLine
	e.CursorCol = entry.CursorCol
	e.SelectStart = entry.SelStart
	e.SelectEnd = entry.SelEnd
	e.dirtyAll = true
	e.Modified = true
	if e.OnTextChanged != nil {
		e.OnTextChanged(entry.Text)
	}
	e.MarkForRedraw()
	return true
}

func (e *CodeEditor) addCursor(line, col int) {
	// Don't add if already exists
	for _, c := range e.multiCursor {
		if c.line == line && c.col == col {
			return
		}
	}
	e.multiCursor = append(e.multiCursor, cursorPos{line: line, col: col, selStart: -1})
}

func (e *CodeEditor) clearExtraCursors() {
	e.multiCursor = nil
}

func (e *CodeEditor) hasMultipleCursors() bool {
	return len(e.multiCursor) > 0
}

// forEachCursor applies fn at each cursor position, from bottom to top.
// It temporarily sets the main cursor to each position, calls fn, then restores.
// Use this for multi-cursor edits.
func (e *CodeEditor) forEachCursor(fn func()) {
	if !e.hasMultipleCursors() {
		fn()
		return
	}
	// Sort cursors by offset descending so edits don't shift later positions
	type cs struct{ line, col int }
	var sorted []cs
	for _, c := range e.multiCursor {
		sorted = append(sorted, cs{c.line, c.col})
	}
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			oi := e.lineColToOffset(sorted[i].line, sorted[i].col)
			oj := e.lineColToOffset(sorted[j].line, sorted[j].col)
			if oi < oj {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	for _, c := range sorted {
		e.CursorLine = c.line
		e.CursorCol = c.col
		fn()
	}

	// After all edits, reset to first cursor
	if len(sorted) > 0 {
		e.CursorLine = sorted[len(sorted)-1].line
		e.CursorCol = sorted[len(sorted)-1].col
	}
	e.SelectStart = -1
	e.SelectEnd = -1
	e.selectionAnchor = -1
	e.MarkForRedraw()
}

func (e *CodeEditor) insertAtEachCursor(ch rune) {
	e.saveUndoState(ActionInsertText, string(ch), 0, 0, 0, 0)
	e.forEachCursor(func() {
		line := e.Lines[e.CursorLine]
		e.Lines[e.CursorLine] = line[:e.CursorCol] + string(ch) + line[e.CursorCol:]
		e.CursorCol++
	})
	e.dirtyAll = true
	e.Modified = true
	e.EnsureCursorVisible()
	e.MarkForRedraw()
	if e.OnTextChanged != nil {
		e.OnTextChanged(e.Text())
	}
}

func (e *CodeEditor) backspaceAtEachCursor() {
	e.saveUndoState(ActionDeleteText, "", 0, 0, 0, 0)
	e.forEachCursor(func() {
		if e.CursorCol > 0 {
			line := e.Lines[e.CursorLine]
			e.Lines[e.CursorLine] = line[:e.CursorCol-1] + line[e.CursorCol:]
			e.CursorCol--
		} else if e.CursorLine > 0 {
			prev := e.Lines[e.CursorLine-1]
			curr := e.Lines[e.CursorLine]
			e.CursorCol = len(prev)
			e.Lines[e.CursorLine-1] = prev + curr
			e.Lines = append(e.Lines[:e.CursorLine], e.Lines[e.CursorLine+1:]...)
			e.CursorLine--
		}
	})
	e.dirtyAll = true
	e.Modified = true
	e.cache.Invalidate()
	e.MarkForRedraw()
	if e.OnTextChanged != nil {
		e.OnTextChanged(e.Text())
	}
}

func (e *CodeEditor) deleteAtEachCursor() {
	e.saveUndoState(ActionDeleteText, "", 0, 0, 0, 0)
	e.forEachCursor(func() {
		line := e.Lines[e.CursorLine]
		if e.CursorCol < len(line) {
			e.Lines[e.CursorLine] = line[:e.CursorCol] + line[e.CursorCol+1:]
		} else if e.CursorLine < len(e.Lines)-1 {
			next := e.Lines[e.CursorLine+1]
			e.Lines[e.CursorLine] = line + next
			e.Lines = append(e.Lines[:e.CursorLine+1], e.Lines[e.CursorLine+2:]...)
		}
	})
	e.dirtyAll = true
	e.Modified = true
	e.cache.Invalidate()
	e.MarkForRedraw()
	if e.OnTextChanged != nil {
		e.OnTextChanged(e.Text())
	}
}

func (e *CodeEditor) InsertText(text string) {
	if e.ReadOnly {
		return
	}
	e.saveUndoState(ActionInsertText, text, e.CursorLine, e.CursorCol, e.CursorLine, e.CursorCol)

	if e.SelectStart != e.SelectEnd {
		e.deleteSelection()
	}

	if strings.Contains(text, "\n") {
		parts := strings.Split(text, "\n")
		line := e.Lines[e.CursorLine]
		before := line[:e.CursorCol]
		after := line[e.CursorCol:]

		newLines := make([]string, 0, len(e.Lines)+len(parts)-1)
		newLines = append(newLines, e.Lines[:e.CursorLine]...)
		newLines = append(newLines, before+parts[0])
		for i := 1; i < len(parts)-1; i++ {
			newLines = append(newLines, parts[i])
		}
		newLines = append(newLines, parts[len(parts)-1]+after)
		newLines = append(newLines, e.Lines[e.CursorLine+1:]...)
		e.Lines = newLines

		e.CursorLine += len(parts) - 1
		if e.CursorLine < len(e.Lines) {
			if e.CursorCol > len(e.Lines[e.CursorLine]) {
				e.CursorCol = len(e.Lines[e.CursorLine])
			}
		}
	} else {
		line := e.Lines[e.CursorLine]
		e.Lines[e.CursorLine] = line[:e.CursorCol] + text + line[e.CursorCol:]
		e.CursorCol += len([]rune(text))
	}

	e.SelectStart = 0
	e.SelectEnd = 0
	e.selectionAnchor = e.lineColToOffset(e.CursorLine, e.CursorCol)
	e.dirtyAll = true
	e.Modified = true
	e.EnsureCursorVisible()
	e.cache.Invalidate()
	e.MarkForRedraw()
	if e.OnTextChanged != nil {
		e.OnTextChanged(e.Text())
	}
}

func (e *CodeEditor) InsertNewline() {
	if e.ReadOnly {
		return
	}
	e.saveUndoState(ActionInsertNewline, "\n", e.CursorLine, e.CursorCol, e.CursorLine, e.CursorCol)

	if e.SelectStart != e.SelectEnd {
		e.deleteSelection()
	}

	line := e.Lines[e.CursorLine]
	before := line[:e.CursorCol]
	after := line[e.CursorCol:]

	indent := ""
	for _, ch := range before {
		if ch == ' ' || ch == '\t' {
			indent += string(ch)
		} else {
			break
		}
	}

	newLines := make([]string, 0, len(e.Lines)+1)
	newLines = append(newLines, e.Lines[:e.CursorLine]...)
	newLines = append(newLines, before)
	newLines = append(newLines, indent+after)
	newLines = append(newLines, e.Lines[e.CursorLine+1:]...)
	e.Lines = newLines

	e.CursorLine++
	e.CursorCol = len(indent)
	e.SelectStart = 0
	e.SelectEnd = 0
	e.selectionAnchor = e.lineColToOffset(e.CursorLine, e.CursorCol)
	e.dirtyAll = true
	e.Modified = true
	e.EnsureCursorVisible()
	e.cache.Invalidate()
	e.MarkForRedraw()
	if e.OnTextChanged != nil {
		e.OnTextChanged(e.Text())
	}
}

func (e *CodeEditor) Backspace() {
	if e.ReadOnly {
		return
	}

	if e.SelectStart != e.SelectEnd {
		e.deleteSelection()
		return
	}

	if e.CursorCol > 0 {
		line := e.Lines[e.CursorLine]
		prev := runeStartBefore(line, e.CursorCol)
		e.saveUndoState(ActionDeleteText, "", e.CursorLine, prev, e.CursorLine, e.CursorCol)
		e.Lines[e.CursorLine] = line[:prev] + line[e.CursorCol:]
		e.CursorCol = prev
	} else if e.CursorLine > 0 {
		e.saveUndoState(ActionDeleteText, "", e.CursorLine-1, len(e.Lines[e.CursorLine-1]), e.CursorLine, 0)
		prevLine := e.Lines[e.CursorLine-1]
		currLine := e.Lines[e.CursorLine]
		e.CursorCol = len(prevLine)
		e.Lines[e.CursorLine-1] = prevLine + currLine
		e.Lines = append(e.Lines[:e.CursorLine], e.Lines[e.CursorLine+1:]...)
		e.CursorLine--
	}

	e.SelectStart = 0
	e.SelectEnd = 0
	e.selectionAnchor = e.lineColToOffset(e.CursorLine, e.CursorCol)
	e.dirtyAll = true
	e.Modified = true
	e.EnsureCursorVisible()
	e.cache.Invalidate()
	e.MarkForRedraw()
	if e.OnTextChanged != nil {
		e.OnTextChanged(e.Text())
	}
}

func (e *CodeEditor) Delete() {
	if e.ReadOnly {
		return
	}

	if e.SelectStart != e.SelectEnd {
		e.deleteSelection()
		return
	}

	if e.CursorCol < len(e.Lines[e.CursorLine]) {
		line := e.Lines[e.CursorLine]
		next := runeEndAfter(line, e.CursorCol)
		e.saveUndoState(ActionDeleteText, "", e.CursorLine, e.CursorCol, e.CursorLine, next)
		e.Lines[e.CursorLine] = line[:e.CursorCol] + line[next:]
	} else if e.CursorLine < len(e.Lines)-1 {
		e.saveUndoState(ActionDeleteText, "", e.CursorLine, len(e.Lines[e.CursorLine]), e.CursorLine+1, 0)
		currLine := e.Lines[e.CursorLine]
		nextLine := e.Lines[e.CursorLine+1]
		e.Lines[e.CursorLine] = currLine + nextLine
		e.Lines = append(e.Lines[:e.CursorLine+1], e.Lines[e.CursorLine+2:]...)
	}

	e.dirtyAll = true
	e.Modified = true
	e.cache.Invalidate()
	e.MarkForRedraw()
	if e.OnTextChanged != nil {
		e.OnTextChanged(e.Text())
	}
}

func (e *CodeEditor) deleteSelection() {
	if e.SelectStart > e.SelectEnd {
		e.SelectStart, e.SelectEnd = e.SelectEnd, e.SelectStart
	}
	startLine, startCol := e.offsetToLineCol(e.SelectStart)
	endLine, endCol := e.offsetToLineCol(e.SelectEnd)

	e.saveUndoState(ActionDeleteText, e.Text(), startLine, startCol, endLine, endCol)

	if startLine == endLine {
		line := e.Lines[startLine]
		e.Lines[startLine] = line[:startCol] + line[endCol:]
	} else {
		firstPart := e.Lines[startLine][:startCol]
		lastPart := e.Lines[endLine][endCol:]
		newLines := make([]string, 0, len(e.Lines)-(endLine-startLine))
		newLines = append(newLines, e.Lines[:startLine]...)
		newLines = append(newLines, firstPart+lastPart)
		newLines = append(newLines, e.Lines[endLine+1:]...)
		e.Lines = newLines
	}

	e.CursorLine = startLine
	e.CursorCol = startCol
	e.SelectStart = 0
	e.SelectEnd = 0
	e.selectionAnchor = e.lineColToOffset(e.CursorLine, e.CursorCol)
	e.dirtyAll = true
	e.Modified = true
	e.EnsureCursorVisible()
	e.cache.Invalidate()
	e.MarkForRedraw()
	if e.OnTextChanged != nil {
		e.OnTextChanged(e.Text())
	}
}

func (e *CodeEditor) Indent() {
	if e.ReadOnly {
		return
	}

	if e.SelectStart != e.SelectEnd {
		startLine, _ := e.offsetToLineCol(e.SelectStart)
		endLine, _ := e.offsetToLineCol(e.SelectEnd)
		if e.SelectEnd > e.SelectStart {
			endLine++
		}

		e.saveUndoState(ActionIndent, "", startLine, 0, endLine-1, 0)
		for i := startLine; i < endLine && i < len(e.Lines); i++ {
			e.Lines[i] = "\t" + e.Lines[i]
		}
		e.CursorCol++
	} else {
		e.saveUndoState(ActionIndent, "\t", e.CursorLine, e.CursorCol, e.CursorLine, e.CursorCol)
		line := e.Lines[e.CursorLine]
		e.Lines[e.CursorLine] = line[:e.CursorCol] + "\t" + line[e.CursorCol:]
		e.CursorCol++
	}

	e.dirtyAll = true
	e.Modified = true
	e.cache.Invalidate()
	e.MarkForRedraw()
	if e.OnTextChanged != nil {
		e.OnTextChanged(e.Text())
	}
}

func (e *CodeEditor) Outdent() {
	if e.ReadOnly {
		return
	}

	if e.SelectStart != e.SelectEnd {
		startLine, _ := e.offsetToLineCol(e.SelectStart)
		endLine, _ := e.offsetToLineCol(e.SelectEnd)
		if e.SelectEnd > e.SelectStart {
			endLine++
		}

		e.saveUndoState(ActionOutdent, "", startLine, 0, endLine-1, 0)
		for i := startLine; i < endLine && i < len(e.Lines); i++ {
			if strings.HasPrefix(e.Lines[i], "\t") {
				e.Lines[i] = e.Lines[i][1:]
				if i == e.CursorLine && e.CursorCol > 0 {
					e.CursorCol--
				}
			} else if strings.HasPrefix(e.Lines[i], "    ") {
				e.Lines[i] = e.Lines[i][4:]
				if i == e.CursorLine && e.CursorCol >= 4 {
					e.CursorCol -= 4
				}
			}
		}
	} else {
		line := e.Lines[e.CursorLine]
		if strings.HasPrefix(line[e.CursorCol:], "\t") || (e.CursorCol > 0 && line[e.CursorCol-1] == '\t') {
			if e.CursorCol > 0 && line[e.CursorCol-1] == '\t' {
				e.Lines[e.CursorLine] = line[:e.CursorCol-1] + line[e.CursorCol:]
				e.CursorCol--
			}
		}
	}

	e.dirtyAll = true
	e.Modified = true
	e.cache.Invalidate()
	e.MarkForRedraw()
	if e.OnTextChanged != nil {
		e.OnTextChanged(e.Text())
	}
}

func (e *CodeEditor) selectCurrentCompletion() {
	if len(e.completionItems) == 0 {
		return
	}
	item := e.completionItems[e.completionIdx]
	e.insertCompletion(item)
}

func (e *CodeEditor) TabComplete() {
	if e.completionVisible && len(e.completionItems) > 0 {
		e.selectCurrentCompletion()
		return
	}
	e.Indent()
}

func (e *CodeEditor) insertCompletion(item CompletionItem) {
	insertText := item.InsertText
	if insertText == "" {
		insertText = item.Label
	}

	if e.completionPrefix != "" {
		e.CursorCol -= len([]rune(e.completionPrefix))
		line := e.Lines[e.CursorLine]
		if e.CursorCol >= 0 {
			e.Lines[e.CursorLine] = line[:e.CursorCol] + line[e.CursorCol+len(e.completionPrefix):]
		}
	}

	e.InsertText(insertText)
	e.completionVisible = false
	e.completionItems = nil
}

func (e *CodeEditor) SelectAll() {
	e.SelectStart = 0
	e.SelectEnd = e.lineColToOffset(len(e.Lines)-1, len(e.Lines[len(e.Lines)-1]))
	e.MarkForRedraw()
}

func (e *CodeEditor) Copy() {
	if e.SelectStart == e.SelectEnd {
		return
	}
	start, end := e.SelectStart, e.SelectEnd
	if start > end {
		start, end = end, start
	}
	text := e.Text()[start:end]
	platform.GlobalClipboard.SetText(text)
}

func (e *CodeEditor) Cut() {
	if e.SelectStart == e.SelectEnd {
		e.SelectAll()
	}
	e.Copy()
	e.deleteSelection()
}

func (e *CodeEditor) Paste() {
	text := platform.GlobalClipboard.GetText()
	if text == "" {
		return
	}

	// Auto-indent multi-line paste
	if strings.Contains(text, "\n") && e.CursorLine < len(e.Lines) {
		line := e.Lines[e.CursorLine]
		indent := ""
		for _, ch := range line {
			if ch == ' ' || ch == '\t' {
				indent += string(ch)
			} else {
				break
			}
		}
		parts := strings.Split(text, "\n")
		for i := 1; i < len(parts); i++ {
			parts[i] = indent + parts[i]
		}
		text = strings.Join(parts, "\n")
	}

	// Format-on-paste: run the registered formatter on the pasted text
	if e.OnFormat != nil {
		if formatted, err := e.OnFormat(text); err == nil && formatted != "" {
			text = formatted
		}
	}

	e.InsertText(text)
}

func (e *CodeEditor) MoveCursorLeft(selectMode bool) {
	if !selectMode && e.SelectStart != e.SelectEnd {
		// A plain arrow COLLAPSES towards the direction it moves: Left to the START of the
		// selection. It used to step one character past the selection's end and clear it.
		e.collapseSelectionTo(min(e.SelectStart, e.SelectEnd))
		e.afterCursorMove()
		return
	}
	if e.CursorCol > 0 {
		e.CursorCol = runeStartBefore(e.Lines[e.CursorLine], e.CursorCol)
	} else if e.CursorLine > 0 {
		e.CursorLine--
		e.CursorCol = len(e.Lines[e.CursorLine])
	}
	e.preferredCol = -1
	e.updateSelection(selectMode)
	e.afterCursorMove()
}

func (e *CodeEditor) MoveCursorRight(selectMode bool) {
	if !selectMode && e.SelectStart != e.SelectEnd {
		e.collapseSelectionTo(max(e.SelectStart, e.SelectEnd))
		e.afterCursorMove()
		return
	}
	if e.CursorCol < len(e.Lines[e.CursorLine]) {
		e.CursorCol = runeEndAfter(e.Lines[e.CursorLine], e.CursorCol)
	} else if e.CursorLine < len(e.Lines)-1 {
		e.CursorLine++
		e.CursorCol = 0
	}
	e.preferredCol = -1
	e.updateSelection(selectMode)
	e.afterCursorMove()
}

// collapseSelectionTo puts the caret at a document offset and drops the selection.
func (e *CodeEditor) collapseSelectionTo(offset int) {
	e.CursorLine, e.CursorCol = e.offsetToLineCol(offset)
	off := e.lineColToOffset(e.CursorLine, e.CursorCol)
	e.SelectStart, e.SelectEnd = off, off
	e.selectionAnchor = off
	e.cache.Invalidate()
	e.MarkForRedraw()
}

// afterCursorMove is the tail every movement shares.
func (e *CodeEditor) afterCursorMove() {
	e.EnsureCursorVisible()
	e.MarkForRedraw()
	if e.OnCursorChanged != nil {
		e.OnCursorChanged(e.CursorLine, e.CursorCol)
	}
}

// moveToLineCol puts the caret at line/col, landing on a rune boundary and keeping the goal column
// that a run of vertical moves aims at.
func (e *CodeEditor) moveToLineCol(line, col int) {
	if line < 0 {
		line = 0
	}
	if line >= len(e.Lines) {
		line = len(e.Lines) - 1
	}
	if line < 0 {
		line = 0
	}
	e.CursorLine = line
	text := e.Lines[line]
	if col > len(text) {
		col = len(text)
	}
	for col > 0 && col < len(text) && !utf8.RuneStart(text[col]) {
		col--
	}
	e.CursorCol = col
}

func (e *CodeEditor) MoveCursorUp(selectMode bool) {
	if e.preferredCol < 0 {
		e.preferredCol = e.CursorCol
	}
	target := e.CursorLine - 1
	if target < 0 {
		target = 0
	}
	e.moveToLineCol(target, e.preferredCol)
	e.updateSelection(selectMode)
	e.afterCursorMove()
}

func (e *CodeEditor) MoveCursorDown(selectMode bool) {
	if e.preferredCol < 0 {
		e.preferredCol = e.CursorCol
	}
	target := e.CursorLine + 1
	if target >= len(e.Lines) {
		target = len(e.Lines) - 1
	}
	e.moveToLineCol(target, e.preferredCol)
	e.updateSelection(selectMode)
	e.afterCursorMove()
}

func (e *CodeEditor) MoveCursorHome(selectMode bool) {
	e.preferredCol = -1
	e.CursorCol = 0
	e.updateSelection(selectMode)
	e.EnsureCursorVisible()
	e.MarkForRedraw()
}

func (e *CodeEditor) MoveCursorEnd(selectMode bool) {
	e.preferredCol = -1
	e.CursorCol = len(e.Lines[e.CursorLine])
	e.updateSelection(selectMode)
	e.EnsureCursorVisible()
	e.MarkForRedraw()
}

func (e *CodeEditor) MoveCursorPageUp(selectMode bool) {
	if e.preferredCol < 0 {
		e.preferredCol = e.CursorCol
	}
	linesPerPage := int(e.textAreaHeight / e.lineHeight)
	if linesPerPage < 1 {
		linesPerPage = 1
	}
	target := e.CursorLine - linesPerPage
	if target < 0 {
		target = 0
	}
	e.moveToLineCol(target, e.preferredCol)
	e.updateSelection(selectMode)
	e.afterCursorMove()
}

func (e *CodeEditor) MoveCursorPageDown(selectMode bool) {
	if e.preferredCol < 0 {
		e.preferredCol = e.CursorCol
	}
	linesPerPage := int(e.textAreaHeight / e.lineHeight)
	if linesPerPage < 1 {
		linesPerPage = 1
	}
	target := e.CursorLine + linesPerPage
	if target >= len(e.Lines) {
		target = len(e.Lines) - 1
	}
	e.moveToLineCol(target, e.preferredCol)
	e.updateSelection(selectMode)
	e.afterCursorMove()
}

func (e *CodeEditor) MoveCursorWordLeft(selectMode bool) {
	// Document-wide and rune-class-aware, exactly like the text widgets: it crosses line ends and
	// stops where a double-click would start, instead of walking a byte at a time within the line.
	off := e.lineColToOffset(e.CursorLine, e.CursorCol)
	e.CursorLine, e.CursorCol = e.offsetToLineCol(wordLeftOffset(e.Text(), off))
	e.preferredCol = -1
	e.updateSelection(selectMode)
	e.afterCursorMove()
}

func (e *CodeEditor) MoveCursorWordRight(selectMode bool) {
	off := e.lineColToOffset(e.CursorLine, e.CursorCol)
	e.CursorLine, e.CursorCol = e.offsetToLineCol(wordRightOffset(e.Text(), off))
	e.preferredCol = -1
	e.updateSelection(selectMode)
	e.afterCursorMove()
}

// deleteDocRange removes a document range as one undoable edit.
func (e *CodeEditor) deleteDocRange(lo, hi int) {
	if lo < 0 {
		lo = 0
	}
	if hi > len(e.Text()) {
		hi = len(e.Text())
	}
	if lo >= hi {
		return
	}
	e.SelectStart, e.SelectEnd = lo, hi
	e.CursorLine, e.CursorCol = e.offsetToLineCol(hi)
	e.deleteSelection()
	e.preferredCol = -1
}

// BackspaceWord / DeleteWord delete by word, the keyboard gesture every code editor has.
func (e *CodeEditor) BackspaceWord() {
	if e.ReadOnly {
		return
	}
	if e.SelectStart != e.SelectEnd {
		e.deleteSelection()
		return
	}
	off := e.lineColToOffset(e.CursorLine, e.CursorCol)
	e.deleteDocRange(wordLeftOffset(e.Text(), off), off)
}

func (e *CodeEditor) DeleteWord() {
	if e.ReadOnly {
		return
	}
	if e.SelectStart != e.SelectEnd {
		e.deleteSelection()
		return
	}
	off := e.lineColToOffset(e.CursorLine, e.CursorCol)
	e.deleteDocRange(off, wordRightOffset(e.Text(), off))
}

func (e *CodeEditor) updateSelection(selectMode bool) {
	offset := e.lineColToOffset(e.CursorLine, e.CursorCol)
	if selectMode {
		if e.SelectStart == e.SelectEnd {
			e.selectionAnchor = e.SelectStart
		}
		if offset < e.selectionAnchor {
			e.SelectStart = offset
			e.SelectEnd = e.selectionAnchor
		} else {
			e.SelectStart = e.selectionAnchor
			e.SelectEnd = offset
		}
	} else {
		e.SelectStart = offset
		e.SelectEnd = offset
	}
	e.cache.Invalidate()
}

func (e *CodeEditor) updateBracketMatch() {
	e.bracketMatch = bracketPair{line: -1, col: -1}
	e.bracketMatchEnd = bracketPair{line: -1, col: -1}
	if e.CursorLine >= len(e.Lines) {
		return
	}
	line := e.Lines[e.CursorLine]
	if e.CursorCol > 0 {
		ch := rune(line[e.CursorCol-1])
		if isOpenBracket(ch) || isCloseBracket(ch) || ch == '"' || ch == '\'' {
			e.findMatchingBracket(e.CursorLine, e.CursorCol-1, ch)
		}
	}
	if e.CursorCol < len(line) {
		ch := rune(line[e.CursorCol])
		if isOpenBracket(ch) || isCloseBracket(ch) || ch == '"' || ch == '\'' {
			if e.bracketMatch.line < 0 {
				e.findMatchingBracket(e.CursorLine, e.CursorCol, ch)
			}
		}
	}
}

func (e *CodeEditor) findMatchingBracket(line, col int, ch rune) {
	if line >= len(e.Lines) {
		return
	}
	text := e.Lines[line]
	if col >= len(text) {
		return
	}

	e.bracketMatch = bracketPair{line: line, col: col, ch: ch}
	opening := isOpenBracket(ch) || ch == '"' || ch == '\''
	target := ch
	if opening {
		switch ch {
		case '(':
			target = ')'
		case '[':
			target = ']'
		case '{':
			target = '}'
		}
	} else {
		target = matchingBracket(ch)
	}

	depth := 1
	dir := 1
	start := col + dir
	if !opening {
		dir = -1
		start = col - 1
	}

	for cur := start; cur >= 0 && cur < len(text); cur += dir {
		c := rune(text[cur])
		if c == ch {
			depth++
		}
		if c == target {
			depth--
			if depth == 0 {
				e.bracketMatchEnd = bracketPair{line: line, col: cur, ch: c}
				return
			}
		}
	}

	// Search other lines
	for ln := line + dir; ln >= 0 && ln < len(e.Lines); ln += dir {
		other := e.Lines[ln]
		if opening {
			for ci := 0; ci < len(other); ci++ {
				c := rune(other[ci])
				if c == ch {
					depth++
				}
				if c == target {
					depth--
					if depth == 0 {
						e.bracketMatchEnd = bracketPair{line: ln, col: ci, ch: c}
						return
					}
				}
			}
		} else {
			for ci := len(other) - 1; ci >= 0; ci-- {
				c := rune(other[ci])
				if c == ch {
					depth++
				}
				if c == target {
					depth--
					if depth == 0 {
						e.bracketMatchEnd = bracketPair{line: ln, col: ci, ch: c}
						return
					}
				}
			}
		}
	}
}

func (e *CodeEditor) Find(text string) {
	e.searchText = text
	e.searchVisible = text != ""
	e.searchMatches = nil
	e.searchCurrentIdx = 0

	if text == "" {
		e.MarkForRedraw()
		return
	}

	for lineNum, line := range e.Lines {
		lowerLine := strings.ToLower(line)
		lowerText := strings.ToLower(text)
		start := 0
		for {
			idx := strings.Index(lowerLine[start:], lowerText)
			if idx < 0 {
				break
			}
			e.searchMatches = append(e.searchMatches, searchMatch{
				line:  lineNum,
				start: start + idx,
				end:   start + idx + len(text),
			})
			start += idx + 1
		}
	}

	if len(e.searchMatches) > 0 {
		e.searchCurrentIdx = 0
		e.goToSearchMatch(0)
	}
	e.MarkForRedraw()
}

func (e *CodeEditor) FindNext() {
	if len(e.searchMatches) == 0 {
		return
	}
	e.searchCurrentIdx = (e.searchCurrentIdx + 1) % len(e.searchMatches)
	e.goToSearchMatch(e.searchCurrentIdx)
}

func (e *CodeEditor) FindPrev() {
	if len(e.searchMatches) == 0 {
		return
	}
	e.searchCurrentIdx--
	if e.searchCurrentIdx < 0 {
		e.searchCurrentIdx = len(e.searchMatches) - 1
	}
	e.goToSearchMatch(e.searchCurrentIdx)
}

func (e *CodeEditor) goToSearchMatch(idx int) {
	if idx < 0 || idx >= len(e.searchMatches) {
		return
	}
	m := e.searchMatches[idx]
	e.CursorLine = m.line
	e.CursorCol = m.start
	e.SelectStart = e.lineColToOffset(m.line, m.start)
	e.SelectEnd = e.lineColToOffset(m.line, m.end)
	e.EnsureCursorVisible()
	e.MarkForRedraw()
}

func (e *CodeEditor) Replace(text string) {
	e.replaceText = text
	e.replaceVisible = true
	e.MarkForRedraw()
}

func (e *CodeEditor) ReplaceCurrent() {
	if e.searchText == "" || len(e.searchMatches) == 0 {
		return
	}
	if e.searchCurrentIdx >= len(e.searchMatches) {
		return
	}
	e.saveUndoState(ActionReplaceText, e.Text(), 0, 0, 0, 0)

	m := e.searchMatches[e.searchCurrentIdx]
	e.CursorLine = m.line
	e.CursorCol = m.start
	e.SelectStart = e.lineColToOffset(m.line, m.start)
	e.SelectEnd = e.lineColToOffset(m.line, m.end)

	line := e.Lines[m.line]
	e.Lines[m.line] = line[:m.start] + e.replaceText + line[m.end:]
	e.CursorCol = m.start + len(e.replaceText)
	e.SelectStart = 0
	e.SelectEnd = 0
	e.dirtyAll = true
	e.Modified = true
	e.MarkForRedraw()
	if e.OnTextChanged != nil {
		e.OnTextChanged(e.Text())
	}
	e.Find(e.searchText)
}

func (e *CodeEditor) ReplaceAll() {
	if e.searchText == "" {
		return
	}
	e.saveUndoState(ActionReplaceText, e.Text(), 0, 0, 0, 0)

	text := e.Text()
	replaced := strings.ReplaceAll(text, e.searchText, e.replaceText)
	if replaced != text {
		e.Lines = strings.Split(replaced, "\n")
		if len(e.Lines) == 0 {
			e.Lines = []string{""}
		}
		e.dirtyAll = true
		e.Modified = true
		e.MarkForRedraw()
		if e.OnTextChanged != nil {
			e.OnTextChanged(replaced)
		}
	}
}

func (e *CodeEditor) GoToLine(line int) {
	if line < 0 {
		line = 0
	}
	if line >= len(e.Lines) {
		line = len(e.Lines) - 1
	}
	e.CursorLine = line
	e.CursorCol = 0
	e.SelectStart = e.lineColToOffset(line, 0)
	e.SelectEnd = e.SelectStart
	e.selectionAnchor = e.SelectStart
	e.EnsureCursorVisible()
	e.MarkForRedraw()
}

func (e *CodeEditor) findFoldRanges() map[int]int {
	lang := e.tsLang()
	if lang != "" {
		ranges := TSFindFoldRanges(e.Lines, lang)
		if len(ranges) > 0 {
			return ranges
		}
	}
	result := make(map[int]int)
	if e.Language == "Go" || strings.HasSuffix(e.FilePath, ".go") {
		src := strings.Join(e.Lines, "\n")
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		if err == nil && f != nil {
			br := &foldVisitor{lines: e.Lines, fset: fset, folds: result}
			ast.Walk(br, f)
			if len(result) > 0 {
				return result
			}
		}
	}
	for i := range e.Lines {
		if strings.Contains(e.Lines[i], "{") && !e.isFoldedInner(i) {
			end := e.simpleFoldEnd(i)
			if end > i {
				result[i] = end
			}
		}
	}
	return result
}

func (e *CodeEditor) tsLang() string {
	lang := strings.ToLower(e.Language)
	switch lang {
	case "go":
		return "go"
	case "python":
		return "python"
	case "javascript", "js":
		return "javascript"
	case "typescript", "ts":
		return "typescript"
	}
	if ext := strings.ToLower(e.FilePath); ext != "" {
		if strings.HasSuffix(ext, ".go") {
			return "go"
		}
		if strings.HasSuffix(ext, ".py") {
			return "python"
		}
		if strings.HasSuffix(ext, ".js") || strings.HasSuffix(ext, ".jsx") {
			return "javascript"
		}
		if strings.HasSuffix(ext, ".ts") || strings.HasSuffix(ext, ".tsx") {
			return "typescript"
		}
	}
	return ""
}

type foldVisitor struct {
	lines []string
	fset  *token.FileSet
	folds map[int]int
}

func (v *foldVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return v
	}
	// Get the position of this node
	startPos := node.Pos()
	endPos := node.End()
	if startPos == token.NoPos || endPos == token.NoPos {
		return v
	}

	startLine := v.fset.Position(startPos).Line
	endLine := v.fset.Position(endPos).Line

	// Only fold blocks that span multiple lines
	if endLine > startLine+1 {
		// Check if this node represents a brace-delimited block
		switch n := node.(type) {
		case *ast.BlockStmt:
			// { ... } — functions, if/for/switch bodies
			if n.Lbrace.IsValid() && n.Rbrace.IsValid() {
				lbraceLine := v.fset.Position(n.Lbrace).Line
				rbraceLine := v.fset.Position(n.Rbrace).Line
				if rbraceLine > lbraceLine {
					v.folds[lbraceLine-1] = rbraceLine - 1
				}
			}
		case *ast.CompositeLit:
			// Type{ ... } — struct/map/slice literals
			if n.Lbrace.IsValid() && n.Rbrace.IsValid() {
				lbraceLine := v.fset.Position(n.Lbrace).Line
				rbraceLine := v.fset.Position(n.Rbrace).Line
				if rbraceLine > lbraceLine {
					v.folds[lbraceLine-1] = rbraceLine - 1
				}
			}
		}
	}
	return v
}

func (e *CodeEditor) simpleFoldEnd(startLine int) int {
	if startLine >= len(e.Lines) {
		return startLine
	}
	line := e.Lines[startLine]
	bracePos := strings.Index(line, "{")
	if bracePos < 0 {
		return startLine
	}

	depth := 1
	for _, ch := range line[bracePos+1:] {
		if ch == '{' {
			depth++
		}
		if ch == '}' {
			depth--
			if depth == 0 {
				return startLine
			}
		}
	}
	for i := startLine + 1; i < len(e.Lines); i++ {
		for _, ch := range e.Lines[i] {
			if ch == '{' {
				depth++
			}
			if ch == '}' {
				depth--
				if depth == 0 {
					return i
				}
			}
		}
	}
	return len(e.Lines) - 1
}

func (e *CodeEditor) ToggleFold(line int) {
	if line < 0 || line >= len(e.Lines) {
		return
	}
	if e.codeFolding[line] {
		delete(e.codeFolding, line)
	} else {
		if _, ok := e.foldRanges[line]; ok {
			e.codeFolding[line] = true
		}
	}
	e.dirtyAll = true
	e.MarkForRedraw()
}

func (e *CodeEditor) FoldAll() {
	e.codeFolding = make(map[int]bool)
	for start := range e.foldRanges {
		e.codeFolding[start] = true
	}
	e.dirtyAll = true
	e.MarkForRedraw()
}

func (e *CodeEditor) UnfoldAll() {
	e.codeFolding = make(map[int]bool)
	e.dirtyAll = true
	e.MarkForRedraw()
}

func (e *CodeEditor) SetDiffMarkers(added, modified, deleted map[int]bool) {
	e.diffAdded = added
	e.diffModified = modified
	e.diffDeleted = deleted
	e.MarkForRedraw()
}

func (e *CodeEditor) ToggleBookmark(line int) {
	if line < 0 || line >= len(e.Lines) {
		return
	}
	if e.bookmarks[line] {
		delete(e.bookmarks, line)
	} else {
		e.bookmarks[line] = true
	}
	e.MarkForRedraw()
}

func (e *CodeEditor) ToggleBreakpoint(line int) {
	if line < 0 || line >= len(e.Lines) {
		return
	}
	if e.breakpoints[line] {
		delete(e.breakpoints, line)
	} else {
		e.breakpoints[line] = true
	}
	e.MarkForRedraw()
}

// visibleLineCount returns the number of lines after folding
func (e *CodeEditor) visibleLineCount() int {
	count := 0
	for i := 0; i < len(e.Lines); i++ {
		if e.codeFolding[i] {
			if end, ok := e.foldRanges[i]; ok {
				i = end // skip folded range
			}
		}
		count++
	}
	return count
}

// visibleLineToSource maps a visible line index to the source line index
func (e *CodeEditor) visibleLineToSource(visLine int) int {
	vis := 0
	for src := 0; src < len(e.Lines); src++ {
		if e.codeFolding[src] {
			if end, ok := e.foldRanges[src]; ok {
				src = end
				continue
			}
		}
		if vis == visLine {
			return src
		}
		vis++
	}
	return len(e.Lines) - 1
}

func (e *CodeEditor) SetDiagnostics(diags []Diagnostic) {
	e.diagnostics = diags
	e.MarkForRedraw()
}

func (e *CodeEditor) ShowCompletion(items []CompletionItem, prefix string, x, y float32) {
	e.completionItems = items
	e.completionPrefix = prefix
	e.completionIdx = 0
	e.completionVisible = len(items) > 0
	e.completionX = x
	e.completionY = y
	e.MarkForRedraw()
}

func (e *CodeEditor) HideCompletion() {
	e.completionVisible = false
	e.completionItems = nil
	e.MarkForRedraw()
}

func (e *CodeEditor) ShowParameterHints(hints []ParameterHint) {
	e.parameterHints = hints
	e.parameterHintVisible = len(hints) > 0
	e.parameterHintIdx = 0
	e.MarkForRedraw()
}

func (e *CodeEditor) HideParameterHints() {
	e.parameterHintVisible = false
	e.parameterHints = nil
	e.MarkForRedraw()
}

func (e *CodeEditor) ShowHover(text string, x, y float32) {
	e.hoverText = text
	e.hoverX = x
	e.hoverY = y
	e.hoverVisible = text != ""
	e.MarkForRedraw()
}

func (e *CodeEditor) HideHover() {
	e.hoverVisible = false
	e.hoverText = ""
	e.MarkForRedraw()
}

func isOpenBracket(ch rune) bool {
	return ch == '(' || ch == '[' || ch == '{'
}

func isCloseBracket(ch rune) bool {
	return ch == ')' || ch == ']' || ch == '}'
}

func matchingBracket(ch rune) rune {
	switch ch {
	case '(':
		return ')'
	case ')':
		return '('
	case '[':
		return ']'
	case ']':
		return '['
	case '{':
		return '}'
	case '}':
		return '{'
	}
	return 0
}

func isOpenQuote(ch rune) bool {
	return ch == '"' || ch == '\'' || ch == '`'
}

func (e *CodeEditor) OnCharacter(ch rune) {
	if !e.Enabled() || !e.Focused {
		return
	}
	if e.composing {
		e.cancelComposition()
	}

	if e.completionVisible {
		if ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '_' {
			e.InsertText(string(ch))
			e.updateCompletion()
			return
		}
		e.HideCompletion()
	}

	if e.parameterHintVisible && ch == ',' {
		e.parameterHintIdx++
		if e.parameterHintIdx >= len(e.parameterHints) {
			e.parameterHintIdx = len(e.parameterHints) - 1
		}
		e.MarkForRedraw()
		return
	}

	// Auto-close brackets
	if isOpenBracket(ch) {
		e.InsertText(string(ch))
		e.InsertText(string(matchingBracket(ch)))
		e.CursorCol--
		e.EnsureCursorVisible()
		e.MarkForRedraw()
		return
	}

	// Auto-close quotes
	if isOpenQuote(ch) {
		e.InsertText(string(ch))
		e.InsertText(string(ch))
		e.CursorCol--
		e.EnsureCursorVisible()
		e.MarkForRedraw()
		return
	}

	// Skip over closing bracket if next char matches
	if isCloseBracket(ch) {
		line := e.Lines[e.CursorLine]
		if e.CursorCol < len(line) && rune(line[e.CursorCol]) == ch {
			e.CursorCol++
			e.EnsureCursorVisible()
			e.MarkForRedraw()
			return
		}
	}

	if e.hasMultipleCursors() {
		e.insertAtEachCursor(ch)
	} else {
		e.InsertText(string(ch))
	}

	// Auto-trigger completion after 3+ word characters
	if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' {
		prefix := e.currentWordPrefix()
		if len(prefix) >= 2 {
			e.triggerCompletion(prefix)
		}
	} else if ch == '.' {
		e.triggerCompletion("")
	}
}

func (e *CodeEditor) triggerCompletion(prefix string) {
	// Ask IDE/LSP via callback first
	items := make([]CompletionItem, 0)
	if e.OnCompletionRequest != nil {
		items = append(items, e.OnCompletionRequest(prefix)...)
	}
	// Add built-in keyword completions
	items = append(items, e.builtinCompletions(prefix)...)

	if len(items) > 0 {
		if e.completionVisible {
			e.completionItems = items
			e.completionPrefix = prefix
			e.completionIdx = 0
			e.MarkForRedraw()
		} else {
			f := e.font()
			line := e.Lines[e.CursorLine]
			x := lineMeasure{text: line, font: f, tabWidth: e.TabWidth}.caretX(e.CursorCol) - e.ScrollX + float32(e.gutterWidth)
			y := e.visibleLineY(e.CursorLine) - e.ScrollY
			e.ShowCompletion(items, prefix, x, y)
		}
	}
}

func (e *CodeEditor) builtinCompletions(prefix string) []CompletionItem {
	lang := strings.ToLower(e.Language)
	keywords := []string{}
	switch lang {
	case "go":
		keywords = []string{"break", "case", "chan", "const", "continue", "default", "defer",
			"else", "fallthrough", "for", "func", "go", "goto", "if", "import",
			"interface", "map", "package", "range", "return", "select", "struct",
			"switch", "type", "var"}
	case "python":
		keywords = []string{"and", "as", "assert", "async", "await", "break", "class",
			"continue", "def", "del", "elif", "else", "except", "finally", "for",
			"from", "global", "if", "import", "in", "is", "lambda", "nonlocal",
			"not", "or", "pass", "raise", "return", "try", "while", "with", "yield"}
	case "javascript", "typescript":
		keywords = []string{"async", "await", "break", "case", "catch", "class", "const",
			"continue", "debugger", "default", "delete", "do", "else", "enum",
			"export", "extends", "finally", "for", "function", "if", "import",
			"in", "instanceof", "let", "new", "of", "return", "static", "super",
			"switch", "this", "throw", "try", "typeof", "var", "void", "while",
			"with", "yield"}
	}
	lower := strings.ToLower(prefix)
	items := make([]CompletionItem, 0)
	for _, kw := range keywords {
		if lower == "" || strings.HasPrefix(strings.ToLower(kw), lower) {
			items = append(items, CompletionItem{Label: kw, Detail: "keyword", InsertText: kw})
		}
	}
	return items
}

func (e *CodeEditor) updateCompletion() {
	prefix := e.currentWordPrefix()
	if prefix == "" {
		e.HideCompletion()
		return
	}

	filtered := make([]CompletionItem, 0)
	lowerPrefix := strings.ToLower(prefix)
	for _, item := range e.completionItems {
		if strings.HasPrefix(strings.ToLower(item.Label), lowerPrefix) {
			filtered = append(filtered, item)
		}
	}

	if len(filtered) == 0 || prefix == e.completionPrefix {
		return
	}

	e.completionItems = filtered
	e.completionPrefix = prefix
	e.completionIdx = 0
	e.MarkForRedraw()
}

func (e *CodeEditor) currentWordPrefix() string {
	line := e.Lines[e.CursorLine]
	if e.CursorCol <= 0 {
		return ""
	}
	start := e.CursorCol
	for start > 0 {
		ch := rune(line[start-1])
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' {
			start--
		} else {
			break
		}
	}
	return line[start:e.CursorCol]
}

// OnKeyRepeat is a key the user is HOLDING — Backspace over a word, an arrow along a row, a scrub.
// The window classifies an OS auto-repeat as a repeat rather than a fresh press (a widget without
// this method is asked once, deliberately), so this is what makes a held key work here.
func (e *CodeEditor) OnKeyRepeat(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	return e.OnKeyDown(keyCode, mod)
}

func (e *CodeEditor) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if !e.Enabled() || !e.Focused {
		return false
	}

	if e.composing && keyCode == platform.KeyEscape {
		e.cancelComposition()
		return true
	}

	ctrl := mod.ControlDown()
	shift := mod.ShiftDown()
	alt := mod.OptionDown()
	e.markInteraction()

	// Word deletes, the code editor's counterpart of the text widgets' Ctrl+Backspace/Delete.
	if keyCode == platform.KeyBackspace && wordModDown(mod) {
		e.BackspaceWord()
		return true
	}
	if keyCode == platform.KeyDelete && wordModDown(mod) {
		e.DeleteWord()
		return true
	}

	if ctrl && !alt {
		switch keyCode {
		case platform.KeyA:
			e.SelectAll()
			return true
		case platform.KeyC:
			e.Copy()
			return true
		case platform.KeyX:
			e.Cut()
			return true
		case platform.KeyV:
			e.Paste()
			return true
		case platform.KeyZ:
			if shift {
				return e.Redo()
			}
			return e.Undo()
		case platform.KeyY:
			return e.Redo()
		case platform.KeyD:
			if shift {
				// Ctrl+Shift+D — Select All Occurrences
				if e.SelectStart == e.SelectEnd {
					// Select word at cursor
					sel := e.currentWordPrefix()
					if sel != "" {
						e.Find(sel)
					}
				}
				word := e.Text()[e.SelectStart:e.SelectEnd]
				e.clearExtraCursors()
				e.Find(word)
				e.SelectAll() // Temporary - will be replaced
				return true
			}
			// Ctrl+D — Add next occurrence as cursor
			if e.SelectStart != e.SelectEnd {
				word := e.Text()[e.SelectStart:e.SelectEnd]
				if len(e.searchMatches) == 0 || e.searchText != word {
					e.Find(word)
				}
				if len(e.searchMatches) > 0 {
					// Add current cursor to multi-cursor set
					e.addCursor(e.CursorLine, e.CursorCol)
					// Jump to next match
					e.FindNext()
					e.addCursor(e.CursorLine, e.CursorCol)
				}
			} else {
				e.MoveCursorDown(false)
				if e.SelectStart == e.SelectEnd {
					e.SelectAll()
				}
			}
			return true
		case platform.KeyF:
			e.Find("")
			return true
		case platform.KeyH:
			e.Replace("")
			return true
		case platform.KeyG:
			if shift {
				e.GoToLine(0)
				return true
			}
			return true
		case platform.KeyS:
			if e.OnSave != nil {
				e.OnSave()
			}
			e.Modified = false
			return true
		case platform.KeySlash:
			e.ToggleLineComment()
			return true
		case platform.KeyCloseBracket:
			e.Indent()
			return true
		case platform.KeyOpenBracket:
			e.Outdent()
			return true
		}
		return false
	}

	switch keyCode {
	case platform.KeyF9:
		e.ToggleBreakpoint(e.CursorLine)
		return true
	case platform.KeyF11:
		e.ToggleBookmark(e.CursorLine)
		return true
	case platform.KeyUp:
		if e.completionVisible {
			e.completionIdx--
			if e.completionIdx < 0 {
				e.completionIdx = len(e.completionItems) - 1
			}
			e.MarkForRedraw()
			return true
		}
		if ctrl && alt {
			if e.CursorLine > 0 {
				e.addCursor(e.CursorLine, e.CursorCol)
				e.CursorLine--
				e.addCursor(e.CursorLine, e.CursorCol)
				e.MarkForRedraw()
			}
			return true
		}
		e.MoveCursorUp(shift)
		return true
	case platform.KeyDown:
		if e.completionVisible {
			e.completionIdx++
			if e.completionIdx >= len(e.completionItems) {
				e.completionIdx = 0
			}
			e.MarkForRedraw()
			return true
		}
		if ctrl && alt {
			if e.CursorLine < len(e.Lines)-1 {
				e.addCursor(e.CursorLine, e.CursorCol)
				e.CursorLine++
				e.addCursor(e.CursorLine, e.CursorCol)
				e.MarkForRedraw()
			}
			return true
		}
		e.MoveCursorDown(shift)
		return true
	case platform.KeyEscape:
		// Escape clears extra cursors + hides overlays
		if e.hasMultipleCursors() {
			e.clearExtraCursors()
			e.MarkForRedraw()
			return true
		}
		e.HideCompletion()
		e.HideParameterHints()
		e.HideHover()
		e.MarkForRedraw()
		return true
	case platform.KeyBackslash:
		if ctrl && shift {
			// Go to matching bracket
			if e.bracketMatchEnd.line >= 0 {
				e.CursorLine = e.bracketMatchEnd.line
				e.CursorCol = e.bracketMatchEnd.col + 1
				e.EnsureCursorVisible()
				e.cache.Invalidate()
				e.MarkForRedraw()
			}
			return true
		}
	case platform.KeyLeft:
		e.MoveCursorLeft(shift)
		return true
	case platform.KeyRight:
		e.MoveCursorRight(shift)
		return true
	case platform.KeyHome:
		e.MoveCursorHome(shift)
		return true
	case platform.KeyEnd:
		e.MoveCursorEnd(shift)
		return true
	case platform.KeyPageUp:
		e.MoveCursorPageUp(shift)
		return true
	case platform.KeyPageDown:
		e.MoveCursorPageDown(shift)
		return true
	case platform.KeyReturn:
		if e.completionVisible && len(e.completionItems) > 0 {
			e.selectCurrentCompletion()
			return true
		}
		e.InsertNewline()
		return true
	case platform.KeyNumPadEnter:
		if e.completionVisible && len(e.completionItems) > 0 {
			e.selectCurrentCompletion()
			return true
		}
		e.InsertNewline()
		return true
	case platform.KeyBackspace:
		e.Backspace()
		return true
	case platform.KeyDelete:
		e.Delete()
		return true
	case platform.KeyTab:
		if shift {
			e.Outdent()
		} else {
			e.TabComplete()
		}
		return true
	}

	// Printable characters are handled by OnCharacter (RuneTypedCallback)
	// Only handle non-printable keys here
	return false
}

func (e *CodeEditor) ToggleLineComment() {
	if e.ReadOnly {
		return
	}
	commentPrefix := "//"
	switch e.Language {
	case "Python":
		commentPrefix = "#"
	case "JavaScript", "TypeScript":
		commentPrefix = "//"
	}

	startLine, _ := e.offsetToLineCol(e.SelectStart)
	endLine, _ := e.offsetToLineCol(e.SelectEnd)
	if e.SelectStart == e.SelectEnd {
		startLine = e.CursorLine
		endLine = e.CursorLine
	}

	allCommented := true
	for i := startLine; i <= endLine && i < len(e.Lines); i++ {
		stripped := strings.TrimLeft(e.Lines[i], " \t")
		if !strings.HasPrefix(stripped, commentPrefix) {
			allCommented = false
			break
		}
	}

	e.saveUndoState(ActionReplaceText, e.Text(), startLine, 0, endLine, 0)

	for i := startLine; i <= endLine && i < len(e.Lines); i++ {
		if allCommented {
			idx := strings.Index(e.Lines[i], commentPrefix)
			if idx >= 0 {
				e.Lines[i] = e.Lines[i][:idx] + e.Lines[i][idx+len(commentPrefix):]
			}
		} else {
			e.Lines[i] = "\t" + commentPrefix + " " + e.Lines[i]
		}
	}

	e.dirtyAll = true
	e.Modified = true
	e.MarkForRedraw()
	if e.OnTextChanged != nil {
		e.OnTextChanged(e.Text())
	}
}

func (e *CodeEditor) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "text":
		if s, ok := value.(string); ok {
			e.SetText(s)
		}
	case "filePath":
		if s, ok := value.(string); ok {
			e.SetFilePath(s)
		}
	case "language":
		if s, ok := value.(string); ok {
			e.SetLanguage(s)
		}
	case "readOnly":
		if b, ok := value.(bool); ok {
			e.ReadOnly = b
		}
	case "fontSize":
		if f, ok := value.(float64); ok {
			e.FontSize = float32(f)
			e.recalcCharWidth()
		}
	case "wordWrap":
		if b, ok := value.(bool); ok {
			e.WordWrap = b
		}
	case "showLineNumbers":
		if b, ok := value.(bool); ok {
			e.ShowLineNumbers = b
		}
	}
}

func (e *CodeEditor) Dispose() {
	e.stopCursorBlink()
	e.OnTextChanged = nil
	e.OnCursorChanged = nil
	e.OnSave = nil
	e.diagnostics = nil
	e.completionItems = nil
	e.parameterHints = nil
	e.BaseWidget.Dispose()
}

func (e *CodeEditor) Focusable() bool { return e.Enabled() }
