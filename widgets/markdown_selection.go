package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/platform"
	"strings"
	"time"
)

// This file is the markdown view's selection: the browser-grade behaviours on
// top of the layout table in markdown_layout.go. Selection is held as two
// markdown *source* offsets (anchor/focus), never as block indices, so a drag
// can start and end at any character.

// ── selection state ───────────────────────────────────────────────

// SelectionBounds returns the ordered source range of the selection. When
// nothing is selected both values are 0.
func (mv *MarkdownView) SelectionBounds() (int, int) {
	if mv.selAnchor < 0 || mv.selFocus < 0 {
		return 0, 0
	}
	if mv.selAnchor <= mv.selFocus {
		return mv.selAnchor, mv.selFocus
	}
	return mv.selFocus, mv.selAnchor
}

// HasSelection reports whether a non-empty range is selected.
func (mv *MarkdownView) HasSelection() bool {
	a, b := mv.SelectionBounds()
	return b > a
}

// CaretOffset returns the moving end of the selection (the caret), or -1.
func (mv *MarkdownView) CaretOffset() int {
	if mv.selFocus < 0 {
		return mv.selAnchor
	}
	return mv.selFocus
}

// SelectedMarkdown returns the exact markdown source of the selection.
func (mv *MarkdownView) SelectedMarkdown() string {
	a, b := mv.SelectionBounds()
	if b <= a || b > len(mv.text) {
		return ""
	}
	return mv.text[a:b]
}

// SelectedText returns the selected text as plain text: the rendered
// characters of the range, one line per visual line and one tab between table
// cells. This is what a browser puts on the clipboard for a text selection.
func (mv *MarkdownView) SelectedText() string {
	a, b := mv.SelectionBounds()
	if b <= a {
		return ""
	}
	return mv.plainTextRange(a, b)
}

// SetSelection sets the selection to [anchor, focus] and redraws.
func (mv *MarkdownView) SetSelection(anchor, focus int) {
	mv.selAnchor, mv.selFocus = anchor, focus
	mv.afterSelectionChange()
}

// ClearSelection drops the selection (there is then no caret).
func (mv *MarkdownView) ClearSelection() {
	mv.selAnchor, mv.selFocus = -1, -1
	mv.afterSelectionChange()
}

// CollapseTo places the caret at off with no selection.
func (mv *MarkdownView) CollapseTo(off int) {
	if off < 0 {
		return
	}
	if off > len(mv.text) {
		off = len(mv.text)
	}
	mv.selAnchor, mv.selFocus = off, off
	mv.afterSelectionChange()
}

// SelectTo extends the selection to off, keeping the anchor where it is (this
// is what shift-click and shift-arrow do). An atomic object (an image, a task
// glyph) is selected whole: a focus inside one snaps to the edge the selection
// is coming from.
func (mv *MarkdownView) SelectTo(off int) {
	if off < 0 {
		return
	}
	if off > len(mv.text) {
		off = len(mv.text)
	}
	if mv.selAnchor < 0 {
		mv.CollapseTo(off)
		return
	}
	mv.selFocus = mv.textLayout().snapAtomic(off, mv.selAnchor)
	mv.afterSelectionChange()
}

// SelectAll selects the whole document.
func (mv *MarkdownView) SelectAll() {
	if len(mv.text) == 0 {
		return
	}
	mv.SetSelection(0, len(mv.text))
}

// SelectWordAt selects the word (or whitespace or punctuation run) at off.
func (mv *MarkdownView) SelectWordAt(off int) {
	lo, hi := wordRangeAt(mv.text, off)
	mv.SetSelection(lo, hi)
}

// SelectBlockAt selects the whole block containing off.
func (mv *MarkdownView) SelectBlockAt(off int) {
	lo, hi := mv.blockRangeAt(off)
	if hi > lo {
		mv.SetSelection(lo, hi)
	}
}

// Copy puts the selected plain text on the clipboard.
func (mv *MarkdownView) Copy() {
	if s := mv.SelectedText(); s != "" {
		platform.GlobalClipboard.SetText(s)
	}
}

// CopyMarkdown puts the selected markdown source on the clipboard.
func (mv *MarkdownView) CopyMarkdown() {
	if s := mv.SelectedMarkdown(); s != "" {
		platform.GlobalClipboard.SetText(s)
	}
}

func (mv *MarkdownView) afterSelectionChange() {
	mv.cache.Invalidate()
	mv.MarkForRedraw()
}

// ── ranges in the source ──────────────────────────────────────────

// blockRangeAt returns the source span of the block containing off, or of the
// nearest block when off sits in the gap between two blocks.
func (mv *MarkdownView) blockRangeAt(off int) (int, int) {
	bestLo, bestHi, bestDist := -1, -1, 1<<30
	for i := range mv.blocks {
		b := &mv.blocks[i]
		if !b.hasSrcSpan() {
			continue
		}
		if off >= b.SrcStart && off <= b.SrcEnd {
			return b.SrcStart, b.SrcEnd
		}
		d := b.SrcStart - off
		if d < 0 {
			d = off - b.SrcEnd
		}
		if d < bestDist {
			bestDist, bestLo, bestHi = d, b.SrcStart, b.SrcEnd
		}
	}
	return bestLo, bestHi
}

// plainTextRange renders the layout runs whose source offsets fall in [a, b)
// as plain text: lines joined by newlines, table cells by tabs.
func (mv *MarkdownView) plainTextRange(a, b int) string {
	mv.ensureLayout()
	if a > b {
		a, b = b, a
	}
	var lines []string
	for bi := range mv.laid {
		bl := &mv.laid[bi]
		table := mv.blocks[bi].Type == 6 || mv.blocks[bi].Type == 7
		for li := range bl.lines {
			line := &bl.lines[li]
			var cells []string
			var sb strings.Builder
			for ri := range line.runs {
				run := &line.runs[ri]
				var cell strings.Builder
				runes := []rune(run.text)
				for i, r := range runes {
					src := -1
					if i < len(run.runeSrc) {
						src = run.runeSrc[i]
					}
					if src < 0 {
						// A run with no per-rune map (markers, labels) is
						// taken only when it lies wholly inside the range.
						lo, hi := run.sourceExtent()
						if lo >= a && hi <= b && lo >= 0 {
							cell.WriteRune(r)
						}
						continue
					}
					if src >= a && src < b {
						cell.WriteRune(r)
					}
				}
				if table {
					if s := cell.String(); s != "" {
						cells = append(cells, s)
					}
				} else {
					sb.WriteString(cell.String())
				}
			}
			if table {
				if len(cells) > 0 {
					lines = append(lines, strings.Join(cells, "\t"))
				}
				continue
			}
			if s := sb.String(); s != "" {
				lines = append(lines, s)
			}
		}
	}
	return strings.Join(lines, "\n")
}

// charOffset moves an offset to the previous or next caret stop.
func (mv *MarkdownView) charOffset(off, dir int) int {
	stops := mv.textLayout().caretStops()
	if len(stops) == 0 {
		return off
	}
	if dir < 0 {
		prev := stops[0]
		for _, s := range stops {
			if s >= off {
				break
			}
			prev = s
		}
		return prev
	}
	for _, s := range stops {
		if s > off {
			return s
		}
	}
	return stops[len(stops)-1]
}

// verticalOffset moves the caret one visual line up or down, keeping its x.
func (mv *MarkdownView) verticalOffset(off, dir int) int {
	mv.ensureLayout()
	caret := mv.CaretRect(off)
	if caret.Width == 0 {
		return off
	}
	lines := mv.visualLines()
	for i, ln := range lines {
		top := ln.y + markdownMargin - mv.scrollOff
		if caret.Y < top-1 || caret.Y >= top+ln.h {
			continue
		}
		j := i + dir
		if j < 0 || j >= len(lines) {
			return off
		}
		target := lines[j]
		return mv.SourceOffsetAt(caret.X, target.y+markdownMargin-mv.scrollOff+target.h/2)
	}
	return off
}

// visualLines returns every laid-out line in document order.
func (mv *MarkdownView) visualLines() []mdLine {
	mv.ensureLayout()
	var out []mdLine
	for bi := range mv.laid {
		out = append(out, mv.laid[bi].lines...)
	}
	return out
}

// lineEdgeOffset returns the offset at the start (end=false) or end of the
// visual line holding off.
func (mv *MarkdownView) lineEdgeNear(off int, end bool) int {
	caret := mv.CaretRect(off)
	if caret.Width == 0 {
		return off
	}
	for _, ln := range mv.visualLines() {
		top := ln.y + markdownMargin - mv.scrollOff
		if caret.Y < top-1 || caret.Y >= top+ln.h {
			continue
		}
		if end {
			return mv.lineEdgeInternal(&ln, true)
		}
		return mv.lineEdgeInternal(&ln, false)
	}
	return off
}

func (mv *MarkdownView) lineEdgeInternal(line *mdLine, end bool) int {
	if end {
		for ri := len(line.runs) - 1; ri >= 0; ri-- {
			if _, e := line.runs[ri].sourceExtent(); e > 0 {
				return e
			}
		}
		return -1
	}
	for ri := range line.runs {
		if s, _ := line.runs[ri].sourceExtent(); s >= 0 {
			return s
		}
	}
	return -1
}

// ── mouse ─────────────────────────────────────────────────────────

func (mv *MarkdownView) modifiers() platform.Modifiers {
	if w := mv.Window(); w != nil {
		return w.Modifiers()
	}
	return mv.testModifiers
}

func (mv *MarkdownView) OnMouseDown(x, y float32, button int) bool {
	fr := mv.FrameRect()
	if mv.host.HandleMouseDown(fr.Width, fr.Height, x, y, button) {
		return true
	}
	if button != 0 {
		return false
	}
	// A DISCLOSURE header toggles, and the press is CONSUMED — a header press must not
	// also start a selection, or opening a tool call would drag a caret across it.
	// The check is before the links and the task boxes because the header is above
	// both.
	if bi, ok := mv.disclosureHeaderAt(x, y); ok {
		mv.ToggleDisclosure(bi)
		return true
	}
	// A link is opened on mouse *up*, and only when the press did not become a
	// drag: dragging across a link selects its text, the way a browser does.
	// Ctrl/Cmd-click opens immediately.
	mv.pendingLink = ""
	for _, lr := range mv.linkRects {
		if x >= lr.rect.X && x <= lr.rect.X+lr.rect.Width && y >= lr.rect.Y && y <= lr.rect.Y+lr.rect.Height {
			if mv.modifiers().ControlDown() {
				mv.activateLink(lr.url)
				return true
			}
			mv.pendingLink = lr.url
			mv.pendingLinkRect = lr.rect
			break
		}
	}
	for _, tr := range mv.taskRects {
		if x >= tr.rect.X && x <= tr.rect.X+tr.rect.Width && y >= tr.rect.Y && y <= tr.rect.Y+tr.rect.Height {
			if src, ok := toggleTextTask(mv.text, tr.index); ok {
				checked := !tr.checked
				mv.SetText(src)
				if mv.OnTaskToggled != nil {
					mv.OnTaskToggled(tr.index, checked)
				}
			}
			return true
		}
	}
	off := mv.SourceOffsetAt(x, y)
	if off < 0 {
		return false
	}
	// Click counting (double = word, triple = block). The window also counts
	// clicks, but keeping our own makes the widget self-contained and testable.
	now := time.Now()
	if now.Sub(mv.lastClickTime) < 350*time.Millisecond &&
		absf(x-mv.lastClickX) < 4 && absf(y-mv.lastClickY) < 4 {
		mv.clickCount++
	} else {
		mv.clickCount = 1
	}
	mv.lastClickTime = now
	mv.lastClickX, mv.lastClickY = x, y

	switch {
	case mv.clickCount >= 3:
		mv.SelectBlockAt(off)
		mv.selecting = false
	case mv.clickCount == 2:
		mv.SelectWordAt(off)
		mv.selecting = false
	default:
		if mv.modifiers().ShiftDown() {
			mv.SelectTo(off)
			mv.selecting = false
		} else {
			mv.CollapseTo(off)
			mv.selecting = true
		}
	}
	mv.dragX, mv.dragY = x, y
	mv.dragMoved = false
	return true
}

// disclosureHeaderAt reports the block index whose DISCLOSURE HEADER is under the point.
//
// It is deliberately a hit test against the laid-out table rather than a scan of the
// source: the header is a piece of GEOMETRY, and geometry is what the layout table
// already answers, so asking it keeps the header and the painted header the same object.
func (mv *MarkdownView) disclosureHeaderAt(x, y float32) (int, bool) {
	mv.ensureLayoutFor(mv.Width())
	if len(mv.laid) != len(mv.blocks) {
		return 0, false
	}
	baseY := float32(markdownMargin) - mv.scrollOff
	for i := range mv.blocks {
		if mv.blocks[i].Type != blockWidget {
			continue
		}
		head := mv.laid[i].headH
		if head <= 0 {
			continue
		}
		top := baseY + mv.laid[i].y
		if y >= top && y <= top+head && x >= 0 && x <= mv.Width() {
			return i, true
		}
	}
	return 0, false
}

// ToggleDisclosure expands or collapses an embedded widget's disclosure and re-lays the
// document out. The block is ATOMIC for selection, so nothing about the selection state
// has to be preserved — which is a large part of why a widget is a good embed and free
// text is not.
func (mv *MarkdownView) ToggleDisclosure(index int) bool {
	if index < 0 || index >= len(mv.blocks) {
		return false
	}
	if index < 0 || index >= len(mv.blocks) {
		return false
	}
	collapsed := !mv.blocks[index].Collapsed
	if !mv.toggleInTree(index) {
		return false
	}
	// Rebuild the VISIBLE list from the tree, so a collapsed region's children leave it
	// entirely — which is what makes them cost nothing rather than merely cost nothing
	// to paint.
	mv.blocks = mv.flatten(mv.blocks[:0])
	if collapsed {
		// A collapse can leave the caret inside the region that just went away, so the
		// selection is dropped rather than left pointing at nothing.
		mv.ClearSelection()
	}
	mv.computeContentH()
	mv.cache.Invalidate()
	mv.SetNeedsLayout()
	mv.MarkForRedraw()
	if mv.OnDisclosureToggled != nil {
		mv.OnDisclosureToggled(index, collapsed)
	}
	return true
}

func (mv *MarkdownView) OnMouseMove(x, y float32) {
	mv.host.UpdateHover(x, y) // overlay pills reveal on strip hover
	mv.host.HandleMouseMove(x, y)
	if !mv.selecting {
		return
	}
	if absf(x-mv.dragX) > 4 || absf(y-mv.dragY) > 4 {
		mv.dragMoved = true
	}
	mv.dragX, mv.dragY = x, y
	mv.autoscrollWhileDragging(y)
	off := mv.SourceOffsetAt(x, y)
	if off >= 0 {
		mv.SelectTo(off)
	}
}

func (mv *MarkdownView) OnMouseUp(x, y float32, button int) bool {
	mv.selecting = false
	if button == 0 && mv.pendingLink != "" {
		url, r := mv.pendingLink, mv.pendingLinkRect
		if !mv.dragMoved && x >= r.X && x <= r.X+r.Width && y >= r.Y && y <= r.Y+r.Height {
			mv.activateLink(url)
		}
	}
	mv.pendingLink = ""
	mv.dragMoved = false
	mv.host.HandleMouseUp(x, y, button)
	return true
}

// autoscrollWhileDragging scrolls the view when a drag leaves it, so a
// selection can run past the visible area.
func (mv *MarkdownView) autoscrollWhileDragging(y float32) {
	fr := mv.FrameRect()
	const edge = float32(16)
	switch {
	case y < edge:
		mv.ScrollBy(-(edge - y))
	case y > fr.Height-edge:
		mv.ScrollBy(y - (fr.Height - edge))
	}
}

// activateLink hands a clicked link to the app, or opens it.
func (mv *MarkdownView) activateLink(url string) {
	if mv.OnLinkClicked != nil {
		mv.OnLinkClicked(url)
		return
	}
	meui.OpenURL(url)
}

// hoverOverLink reports whether (x, y) is over a link or task rectangle.
func (mv *MarkdownView) hoverOverLink(x, y float32) bool {
	for _, lr := range mv.linkRects {
		if x >= lr.rect.X && x <= lr.rect.X+lr.rect.Width && y >= lr.rect.Y && y <= lr.rect.Y+lr.rect.Height {
			return true
		}
	}
	for _, tr := range mv.taskRects {
		if x >= tr.rect.X && x <= tr.rect.X+tr.rect.Width && y >= tr.rect.Y && y <= tr.rect.Y+tr.rect.Height {
			return true
		}
	}
	return false
}

// CursorAt returns a text cursor over text and a pointing cursor over links.
func (mv *MarkdownView) CursorAt(x, y float32) *platform.Cursor {
	if mv.hoverOverLink(x, y) {
		return platform.PointingCursor()
	}
	if mv.SourceOffsetAt(x, y) >= 0 {
		return platform.TextCursor()
	}
	return nil
}

// ── keyboard and focus ────────────────────────────────────────────

// Focusable reports that the view can take keyboard focus.
func (mv *MarkdownView) Focusable() bool { return mv.Enabled() && mv.Visible() }

func (mv *MarkdownView) OnFocus() {
	mv.focused = true
	mv.MarkForRedraw()
}

func (mv *MarkdownView) OnBlur() {
	mv.focused = false
	mv.MarkForRedraw()
}

// OnKeyRepeat is a key the user is HOLDING — Backspace over a word, an arrow along a row, a scrub.
// The window classifies an OS auto-repeat as a repeat rather than a fresh press (a widget without
// this method is asked once, deliberately), so this is what makes a held key work here.
func (mv *MarkdownView) OnKeyRepeat(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	return mv.OnKeyDown(keyCode, mod)
}

func (mv *MarkdownView) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if !mv.Enabled() || !mv.focused {
		return false
	}
	if mod.ControlDown() {
		switch keyCode {
		case platform.KeyA:
			// Ctrl+A SELECTS ALL AND COPIES THE MARKDOWN SOURCE.
			//
			// Copying the source by default is the owner's decision and it is the
			// right one for this view: the source is lossless, so an EMBED survives it
			// (an embed was never anything but a fence), whereas the rendered text
			// flattens a widget to whatever it happened to draw. The convention is
			// unusual — Ctrl+A normally selects and stops — so the rendered text is
			// still one keystroke away on Ctrl+C, and the selection is still made, so
			// Ctrl+X or a drag afterwards behave.
			mv.SelectAll()
			mv.CopyMarkdown()
			return true
		case platform.KeyC:
			if mod.ShiftDown() {
				mv.CopyMarkdown()
			} else {
				mv.Copy()
			}
			return true
		}
	}

	extend := mod.ShiftDown()
	// A plain movement key first collapses the selection without moving, the
	// way a text editor behaves; shift keeps it and extends.
	if !extend && mv.HasSelection() && isMovementKey(keyCode) {
		switch keyCode {
		case platform.KeyLeft, platform.KeyUp, platform.KeyHome:
			a, _ := mv.SelectionBounds()
			mv.CollapseTo(a)
		default:
			_, b := mv.SelectionBounds()
			mv.CollapseTo(b)
		}
		return true
	}
	off := mv.CaretOffset()
	if off < 0 {
		off = 0
	}

	switch keyCode {
	case platform.KeyEscape:
		mv.ClearSelection()
		return true
	case platform.KeyLeft:
		target := mv.charOffset(off, -1)
		if mod.ControlDown() {
			target = mv.wordOffset(off, -1)
		}
		mv.moveCaret(target, extend)
		return true
	case platform.KeyRight:
		target := mv.charOffset(off, 1)
		if mod.ControlDown() {
			target = mv.wordOffset(off, 1)
		}
		mv.moveCaret(target, extend)
		return true
	case platform.KeyUp:
		mv.moveCaret(mv.verticalOffset(off, -1), extend)
		return true
	case platform.KeyDown:
		mv.moveCaret(mv.verticalOffset(off, 1), extend)
		return true
	case platform.KeyHome:
		mv.moveCaret(mv.lineEdgeNear(off, false), extend)
		return true
	case platform.KeyEnd:
		mv.moveCaret(mv.lineEdgeNear(off, true), extend)
		return true
	}
	return false
}

// isMovementKey reports whether a key moves the caret.
func isMovementKey(code platform.KeyCode) bool {
	switch code {
	case platform.KeyLeft, platform.KeyRight, platform.KeyUp, platform.KeyDown, platform.KeyHome, platform.KeyEnd:
		return true
	}
	return false
}

// moveCaret moves the caret to off, extending the selection when shift is held.
func (mv *MarkdownView) moveCaret(off int, extend bool) {
	if off < 0 {
		return
	}
	if extend {
		mv.SelectTo(off)
		return
	}
	mv.CollapseTo(off)
}

// wordOffset moves an offset a whole word in the given direction.
func (mv *MarkdownView) wordOffset(off, dir int) int {
	if dir > 0 {
		lo, hi := wordRangeAt(mv.text, off)
		if hi > off {
			return hi
		}
		_ = lo
		return mv.charOffset(off, 1)
	}
	lo, _ := wordRangeAt(mv.text, off)
	if lo < off {
		return lo
	}
	return mv.charOffset(off, -1)
}
