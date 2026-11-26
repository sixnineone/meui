package widgets

import (
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/slant"
	"github.com/sixnineone/meui/graphics/enums/weight"
	"github.com/sixnineone/meui/theme"
	"strconv"
	"strings"
	"unicode/utf8"
)

// This file is the markdown view's layout pass: it turns the parsed blocks
// into positioned runs, and answers position <-> markdown-source-offset
// queries from that one table. Paint draws the runs it produces, so geometry
// and painting cannot drift apart.

// mdRun kinds.
const (
	mdRunText = iota
	mdRunImage
	mdRunTask
	mdRunMarker // list marker / bullet: drawn, but no source text
)

// mdBlockLayout is the geometry of one block.
type mdBlockLayout struct {
	y, h  float32
	lines []mdLine
	// headH is a DISCLOSURE's header height, and 0 on every block that is not one. It
	// lives here rather than being recomputed at paint time so the painter cannot
	// disagree with the layout about where a widget's body starts — which is exactly
	// what happened when Paint guessed `lh` and the body came out zero tall.
	headH float32

	// Table geometry (row blocks only), reuse by Paint so column widths are
	// not recomputed per frame.
	cols   []float32
	totalW float32
}

// runeOffsets returns the per-rune source offsets for a node-backed run whose
// text is exactly the node's text (no newline split).
func (mv *MarkdownView) runeOffsets(rn renderNode) []int {
	n := utf8.RuneCountInString(rn.Text)
	if n == 0 {
		return nil
	}
	out := make([]int, n)
	for i := range out {
		s, _ := runeSrcSpan(rn, i)
		out[i] = s
	}
	return out
}

// plainRuneOffsets assigns consecutive source offsets to the runes of text,
// which is exact when text is a verbatim slice of the source at srcStart (code
// lines, and any run that came from one segment without transformation).
func plainRuneOffsets(text string, srcStart int) []int {
	if srcStart < 0 {
		return nil
	}
	out := make([]int, 0, utf8.RuneCountInString(text))
	off := srcStart
	for _, r := range text {
		out = append(out, off)
		off += utf8.RuneLen(r)
	}
	return out
}

// nodeFontAndWidth resolves the font a render node is drawn with and the
// advance of text in it. renderNode draws with the same helper, so measured
// and painted geometry are identical by construction.
func nodeFontAndWidth(rn renderNode, text string, font graphics.Font) (graphics.Font, float32) {
	f := font
	styled := !rn.Code && rn.Link == ""
	if styled && (rn.Bold || rn.Italic) {
		fd := f.Descriptor()
		if rn.Bold {
			fd.Weight = weight.Black
		}
		if rn.Italic {
			fd.Slant = slant.Italic
		}
		f = fd.Font()
	}
	if rn.Sup || rn.Sub {
		fd := f.Descriptor()
		fd.Size *= 0.75
		f = fd.Font()
	}
	return f, f.SimpleWidth(text)
}

func (mv *MarkdownView) baseFont() graphics.Font {
	font := theme.DefaultTheme.BaseFont
	if font == nil {
		font = graphics.SystemFont
	}
	return font
}

// markdownMargin is the content inset used by Paint and by hit-testing.
const markdownMargin = float32(12)

// layoutContent rebuilds blockYs/blockHs and the per-block run geometry, and
// recomputes contentH. It is the single source of truth for vertical
// positions: Paint replays it, hit-testing reads it.
func (mv *MarkdownView) layoutContent() {
	font := mv.baseFont()
	lineH := font.LineHeight()
	if lineH < 1 {
		lineH = 20
	}
	ascent := font.Baseline()

	mv.blockYs = mv.blockYs[:0]
	mv.blockHs = mv.blockHs[:0]
	mv.laid = mv.laid[:0]
	mv.laidLineH = lineH
	mv.laidSpaceW = font.SimpleWidth(" ")

	total := float32(0)
	prevType := -1
	for bi := range mv.blocks {
		blk := &mv.blocks[bi]
		// Blank-line rules between blocks: after a heading, and around
		// tables and lists. Paint used to own these; the layout owns them
		// now so the selection overlay cannot drift from the painted text.
		if prevType == blockHeading && blk.Type != blockHeading {
			total += lineH
		}
		if (prevType == blockTableRow || prevType == blockTableHeader || prevType == blockListItem) &&
			(blk.Type != blockTableRow && blk.Type != blockTableHeader && blk.Type != blockListItem) {
			total += lineH * 2
		}
		if (blk.Type == blockTableRow || blk.Type == blockTableHeader || blk.Type == blockListItem) &&
			prevType != blockTableRow && prevType != blockTableHeader && prevType != blockListItem && prevType != -1 && prevType != blockHeading {
			total += lineH * 2
		}
		prevType = blk.Type

		bl := mdBlockLayout{y: total}
		switch blk.Type {
		case blockParagraph, blockError:
			// An error is laid out EXACTLY as a paragraph, because it is one: it has to be
			// selectable and copyable, since the author needs to copy the widget name that
			// failed. Sharing the case rather than duplicating the arithmetic is also what
			// keeps a bespoke error path from drifting a line out of position — which is what
			// happened when the error drew itself.
			//
			// run.x is content space without the left margin: Paint adds
			// markdownMargin when it draws, and the hit-test conversions
			// subtract it again.
			bl.lines = mv.layoutInlineLines(blk, float32(blk.QuoteDepth)*6, total, font, lineH, true)
			bl.h = float32(len(bl.lines))*lineH + lineH*0.5
			if len(bl.lines) == 0 {
				bl.h = lineH + lineH*0.5
			}
		case blockHeading:
			idx := blk.Level - 1
			if idx < 0 {
				idx = 0
			}
			if idx > 5 {
				idx = 5
			}
			hf := headingFont(headingSizes[idx])
			bl.lines = mv.layoutInlineLines(blk, 0, total+hf.Baseline(), hf, lineH, false)
			if len(bl.lines) > 0 {
				bl.lines[0].y = total
				bl.lines[0].h = headingSpacing2[idx]
			}
			bl.h = headingSpacing2[idx]
		case blockWidget:
			// The widget asks for its height, and the rect it is PAINTED into is exactly
			// that — so a widget can never disagree with the space the layout reserved for
			// it, and can never overlap its neighbour by being taller than it claimed.
			//
			// A COLLAPSED widget asks for nothing: it is not laid out, which is the
			// whole point of a disclosure and the property
			// TestACollapsedTranscriptRowCostsPixels asserts for a Collapsible.
			// A widget is ALWAYS inside a disclosure, so its block is a HEADER plus
			// the widget — which is why a collapsed one costs exactly the header. The
			// container owns the header height; the widget only ever sees the body.
			head := lineH * 1.5
			bl.headH = head
			bl.h = head
			if !blk.Collapsed && blk.Widget != nil {
				bl.h = head + blk.Widget.Height(&WidgetContext{
					Name: blk.WidgetName, Params: blk.Params, Body: blk.Body,
					Width: mv.layoutWidth, Theme: mv.MarkdownTheme.Theme(),
				})
			}
		case blockCode:
			codeFont := font
			if graphics.MonospacedFont != nil {
				if face := graphics.MonospacedFont.Face(); face != nil {
					codeFont = face.Font(6)
				}
			}
			clh := codeFont.LineHeight()
			clines := strings.Split(blk.CodeText, "\n")
			labelH := float32(0)
			if blk.Lang != "" {
				labelH = clh
			}
			// Paint draws the language label (when present) and then the
			// code lines, so the block's height is the sum of those.
			bl.h = labelH + float32(len(clines))*clh
			y := total
			if blk.Lang != "" {
				// The info string is not part of the code content, so the
				// label carries no source offsets.
				bl.lines = append(bl.lines, mdLine{y: y, h: clh, runs: []mdRun{{
					kind: mdRunText, nodeIdx: -1, text: blk.Lang, font: codeFont,
					x: 0, y: y, w: codeFont.SimpleWidth(blk.Lang), h: clh,
					srcStart: -1, srcEnd: -1,
				}}})
				y += clh
			}
			lineSrc := mv.codeLineOffsets(blk, clines)
			for i, cl := range clines {
				line := mdLine{y: y, h: clh}
				if cl != "" {
					ss, se := blk.SrcStart, blk.SrcEnd
					if i < len(lineSrc) {
						ss, se = lineSrc[i][0], lineSrc[i][1]
					}
					line.runs = []mdRun{{
						kind: mdRunText, nodeIdx: -1, text: cl, font: codeFont,
						x: 0, y: y, w: codeFont.SimpleWidth(cl), h: clh,
						srcStart: ss, srcEnd: se,
						runeSrc: plainRuneOffsets(cl, ss),
					}}
				}
				bl.lines = append(bl.lines, line)
				y += clh
			}
		case blockListItem:
			indent := float32(blk.Level) * 20
			marker := "•"
			if blk.Ordered {
				marker = strconv.Itoa(blk.Number) + "."
			}
			markerText := marker + " "
			mw := font.SimpleWidth(markerText)
			line := mdLine{y: total, h: lineH * 1.5}
			line.runs = append(line.runs, mdRun{
				kind: mdRunMarker, nodeIdx: -1, text: markerText, font: font,
				x: indent, y: total, w: mw, h: lineH,
				srcStart: blk.SrcStart, srcEnd: blk.SrcStart, atomic: true,
			})
			x := indent + mw
			for ni := range blk.Nodes {
				rn := blk.Nodes[ni]
				f, w := nodeFontAndWidth(rn, rn.Text, font)
				kind := mdRunText
				switch {
				case rn.Task:
					kind = mdRunTask
				case rn.ImageAlt != "":
					kind = mdRunImage
				}
				line.runs = append(line.runs, mdRun{
					kind: kind, nodeIdx: ni, text: rn.Text, font: f,
					x: x, y: total, w: w, h: lineH,
					srcStart: rn.SrcStart, srcEnd: rn.SrcEnd,
					runeSrc: mv.runeOffsets(rn), atomic: kind != mdRunText,
				})
				x += w
			}
			bl.lines = []mdLine{line}
			bl.h = lineH * 1.5
		case blockRule:
			bl.h = lineH
		case 6, 7:
			bl.h = lineH + 6
			bl.cols, bl.totalW = mv.tableColumns()
			x := float32(0)
			line := mdLine{y: total, h: lineH + 6}
			for ci, cell := range blk.Cells {
				if ci >= len(bl.cols) {
					break
				}
				text := strings.TrimSpace(cell)
				cs, ce := blk.SrcStart, blk.SrcEnd
				if ci < len(blk.CellSrc) && blk.CellSrc[ci][0] >= 0 {
					cs, ce = blk.CellSrc[ci][0], blk.CellSrc[ci][1]
				}
				line.runs = append(line.runs, mdRun{
					kind: mdRunText, nodeIdx: -1, text: text, font: font,
					x: x + 6, y: total, w: font.SimpleWidth(text), h: lineH,
					srcStart: cs, srcEnd: ce,
					runeSrc: mv.cellRuneOffsets(text, cs, ce),
				})
				x += bl.cols[ci]
			}
			if len(line.runs) > 0 {
				bl.lines = []mdLine{line}
			}
		}
		mv.blockYs = append(mv.blockYs, bl.y)
		mv.blockHs = append(mv.blockHs, bl.h)
		mv.laid = append(mv.laid, bl)
		total += bl.h
	}
	mv.laidH = total
	mv.contentH = total * 1.25
	_ = ascent
}

// layoutInlineLines lays out the inline nodes of a paragraph-like block into
// visual lines, honouring the newline characters that soft/hard breaks and
// explicit splits introduce. It mirrors Paint's drawing order exactly: paint
// calls the same code with a canvas, the layout pass with a nil canvas.
func (mv *MarkdownView) layoutInlineLines(blk *block, x0, yTop float32, font graphics.Font, lineH float32, splitNewlines bool) []mdLine {
	// The line count follows the historical formula (one line per newline),
	// so a line that a soft break leaves empty still exists and the block's
	// height matches what Paint culls against.
	nlines := 1
	if splitNewlines {
		for i := range blk.Nodes {
			nlines += strings.Count(blk.Nodes[i].Text, "\n")
		}
	}
	lines := make([]mdLine, nlines)
	for i := range lines {
		lines[i] = mdLine{y: yTop + float32(i)*lineH, h: lineH}
	}
	li := 0
	x := x0
	lineStartX := x0
	for ni := range blk.Nodes {
		rn := blk.Nodes[ni]
		if rn.ImageAlt != "" {
			w, h := mv.imageRunSize(&rn, font)
			lines[li].runs = append(lines[li].runs, mdRun{
				kind: mdRunImage, nodeIdx: ni, text: rn.Text, font: font,
				x: x, y: lines[li].y, w: w, h: h,
				srcStart: rn.SrcStart, srcEnd: rn.SrcEnd, atomic: true,
			})
			x += w + 4
			continue
		}
		parts := []string{rn.Text}
		if splitNewlines && strings.Contains(rn.Text, "\n") {
			parts = strings.Split(rn.Text, "\n")
		}
		for pi, part := range parts {
			if pi > 0 {
				li++
				x = lineStartX
				for li >= len(lines) {
					lines = append(lines, mdLine{y: yTop + float32(len(lines))*lineH, h: lineH})
				}
			}
			if part == "" {
				continue
			}
			f, w := nodeFontAndWidth(rn, part, font)
			kind := mdRunText
			if rn.Task {
				kind = mdRunTask
			}
			lines[li].runs = append(lines[li].runs, mdRun{
				kind: kind, nodeIdx: ni, text: part, font: f,
				x: x, y: lines[li].y, w: w, h: lineH,
				srcStart: rn.SrcStart, srcEnd: rn.SrcEnd,
				runeSrc: mv.partRuneOffsets(rn, parts, pi),
				atomic:  kind != mdRunText,
			})
			x += w
		}
	}
	return lines
}

// partRuneOffsets returns the per-rune source offsets of part pi of a node's
// text after splitting on newline.
func (mv *MarkdownView) partRuneOffsets(rn renderNode, parts []string, pi int) []int {
	base := 0
	for i := 0; i < pi; i++ {
		base += utf8.RuneCountInString(parts[i]) + 1 // +1 for the newline
	}
	n := utf8.RuneCountInString(parts[pi])
	if n == 0 {
		return nil
	}
	out := make([]int, n)
	for i := range out {
		s, _ := runeSrcSpan(rn, base+i)
		out[i] = s
	}
	return out
}

// imageRunSize returns the drawn size of an inline image run, mirroring
// renderNode: the scaled image when it loads, else the "[alt]" fallback.
func (mv *MarkdownView) imageRunSize(rn *renderNode, font graphics.Font) (float32, float32) {
	if img := mv.loadImage(rn.Link); img != nil {
		const maxW = float32(400)
		imgW, imgH := maxW, float32(80)
		if s := img.LogicalSize(); s.Width > 0 && s.Height > 0 {
			scale := min(maxW/s.Width, 80/s.Height)
			imgW = s.Width * scale
			imgH = s.Height * scale
		}
		return imgW, imgH
	}
	return font.SimpleWidth("[" + rn.ImageAlt + "]"), font.LineHeight()
}

// tableColumns computes the shared column widths of the following table rows.
func (mv *MarkdownView) tableColumns() ([]float32, float32) {
	font := mv.baseFont()
	maxCols := 0
	for _, b := range mv.blocks {
		if b.Type == 6 || b.Type == 7 {
			if len(b.Cells) > maxCols {
				maxCols = len(b.Cells)
			}
		}
	}
	if maxCols == 0 {
		return nil, 0
	}
	cols := make([]float32, maxCols)
	for _, b := range mv.blocks {
		if b.Type != 6 && b.Type != 7 {
			continue
		}
		for ci, cell := range b.Cells {
			if ci >= maxCols {
				break
			}
			cw := font.SimpleWidth(strings.TrimSpace(cell)) + 20
			if cw > cols[ci] {
				cols[ci] = cw
			}
		}
	}
	width := mv.layoutWidth - markdownMargin*2
	total := float32(0)
	for _, cw := range cols {
		total += cw
	}
	if total > 0 && total < width {
		extra := (width - total) / float32(maxCols)
		for i := range cols {
			cols[i] += extra
		}
	}
	return cols, width
}

// cellRuneOffsets maps a table cell's rendered text onto its source span. The
// cell's rendered text is usually the source slice without surrounding spaces,
// so the offsets are exact there and unknown otherwise (a cell containing
// markup renders differently from its source).
func (mv *MarkdownView) cellRuneOffsets(text string, srcStart, srcEnd int) []int {
	if text == "" || srcStart < 0 || srcEnd > len(mv.text) || srcEnd < srcStart {
		return nil
	}
	raw := mv.text[srcStart:srcEnd]
	at := strings.Index(raw, text)
	if at < 0 {
		return nil
	}
	return plainRuneOffsets(text, srcStart+at)
}

// codeLineOffsets maps each code line to its source span by walking the
// block's source in order, so indentation and any segment newline handling do
// not have to be modelled arithmetically. A line that cannot be found keeps the
// whole block span (its offsets are then coarse, never wrong).
func (mv *MarkdownView) codeLineOffsets(blk *block, clines []string) [][2]int {
	if blk.SrcStart < 0 || blk.SrcEnd > len(mv.text) || blk.SrcEnd < blk.SrcStart {
		return nil
	}
	out := make([][2]int, len(clines))
	cursor := blk.SrcStart
	for i, cl := range clines {
		if cl == "" {
			out[i] = [2]int{cursor, cursor}
			continue
		}
		idx := strings.Index(mv.text[cursor:blk.SrcEnd], cl)
		if idx < 0 {
			out[i] = [2]int{blk.SrcStart, blk.SrcEnd}
			continue
		}
		start := cursor + idx
		out[i] = [2]int{start, start + len(cl)}
		cursor = start + len(cl)
	}
	return out
}

// ── hit-testing and selection ────────────────────────────────────────────

// ensureLayout makes sure the layout table matches the current blocks and the
// current font metrics. The table is built when the text is parsed, but the
// theme (and so the font) may be applied afterwards; without the fingerprint
// the positions would come from the old metrics while Paint draws with the new
// ones, and the selection would drift.
func (mv *MarkdownView) ensureLayout() {
	mv.ensureLayoutFor(mv.FrameRect().Width)
}

// ensureLayoutFor is ensureLayout with an explicit content width: Paint passes
// the rect it is drawing into, so table columns are measured against exactly
// the width they are painted at.
func (mv *MarkdownView) ensureLayoutFor(width float32) {
	font := mv.baseFont()
	lineH := font.LineHeight()
	if lineH < 1 {
		lineH = 20
	}
	if len(mv.laid) != len(mv.blocks) || mv.laidLineH != lineH ||
		mv.laidSpaceW != font.SimpleWidth(" ") || mv.layoutWidth != width {
		mv.layoutWidth = width
		mv.layoutContent()
	}
}

// contentX converts a widget-local x into content space.
func (mv *MarkdownView) contentX(localX float32) float32 { return localX - markdownMargin }

// contentY converts a widget-local y into content space.
func (mv *MarkdownView) contentY(localY float32) float32 {
	return localY + mv.scrollOff - markdownMargin
}

// textLayout flattens the per-block layouts into the shared model. The lines
// already carry absolute content-space y, so one flat layout answers every
// position query for the whole document; the per-block tables stay for Paint
// and for plain-text extraction.
func (mv *MarkdownView) textLayout() *textLayout {
	mv.flat.lines = mv.flat.lines[:0]
	for bi := range mv.laid {
		mv.flat.lines = append(mv.flat.lines, mv.laid[bi].lines...)
	}
	return &mv.flat
}

// SourceOffsetAt returns the markdown source offset nearest to a widget-local
// point. It returns -1 when the layout has no text at all.
func (mv *MarkdownView) SourceOffsetAt(localX, localY float32) int {
	mv.ensureLayout()
	return mv.textLayout().offsetAt(mv.contentX(localX), mv.contentY(localY))
}

// CaretRect returns the caret rectangle for a source offset, in widget-local
// coordinates. It returns a zero rect when the offset cannot be placed.
func (mv *MarkdownView) CaretRect(off int) graphics.Rect {
	mv.ensureLayout()
	x, y, h, ok := mv.textLayout().caret(off)
	if !ok {
		return graphics.Rect{}
	}
	return graphics.NewRect(x+markdownMargin, y+markdownMargin-mv.scrollOff, 2, h)
}

// SelectionRects returns the highlight rectangles covering the source range
// [a, b) in widget-local coordinates.
func (mv *MarkdownView) SelectionRects(a, b int) []graphics.Rect {
	mv.ensureLayout()
	rects := mv.textLayout().rects(a, b)
	for i := range rects {
		rects[i].X += markdownMargin
		rects[i].Y += markdownMargin - mv.scrollOff
	}
	return append(rects, mv.atomicRects(a, b)...)
}

// atomicRects is the SELECTION half of "a widget is atomic".
//
// The text core returns a rect per run, and an embedded widget has no runs — so before
// this, selecting across an embed highlighted NOTHING: the text either side of it lit up
// and the widget itself stayed unmarked, which reads as a selection bug rather than as
// "this thing has no text in it".
//
// A widget contributes ONE rect covering its block, from the same laid-out table the
// painter reads. That is the invariant the whole design rests on: **the selection overlay
// does not care WHAT drew a block** — it asks the table, and a block that drew a widget
// gets the widget's box, exactly as a block that drew text gets its runs.
func (mv *MarkdownView) atomicRects(a, b int) []graphics.Rect {
	if b <= a {
		return nil
	}
	var out []graphics.Rect
	for i := range mv.blocks {
		blk := &mv.blocks[i]
		if !isDisclosure(*blk) || blk.SrcStart < 0 {
			continue
		}
		// Half-open, and a widget whose span merely TOUCHES the selection is included:
		// a selection that ends at the opening fence has still selected the embed.
		if blk.SrcEnd <= a || blk.SrcStart >= b {
			continue
		}
		if i >= len(mv.laid) {
			continue
		}
		l := mv.laid[i]
		out = append(out, graphics.NewRect(markdownMargin,
			markdownMargin-mv.scrollOff+l.y,
			mv.layoutWidth, l.h))
	}
	return out
}

func absf(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
