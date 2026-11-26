package widgets

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/graphics/enums/slant"
	"github.com/sixnineone/meui/graphics/enums/weight"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

const (
	WrapNoWrap   = "noWrap"
	WrapWordWrap = "wordWrap"
	WrapEllipsis = "ellipsis"
)

type richSpan struct {
	Text      string
	Bold      bool
	Italic    bool
	Underline bool
	Strike    bool
	Color     *graphics.Color
	Link      string
}

type Label struct {
	meui.BaseWidget
	*theme.LabelTheme
	Text      string
	Secondary bool
	HAlign    string // "left", "center", "right"
	VAlign    string // "top", "middle", "bottom"
	WrapMode  string // "noWrap", "wordWrap", "ellipsis"
	// MaxLines clamps a wrapped label to N lines and ellipsizes the last one — the
	// web's `-webkit-line-clamp`. Zero means "as many as it takes".
	MaxLines int
	// lineHeightMul scales the font's line height for this label. Zero means the
	// font's own. It is a LABEL property rather than a font one because a face's
	// metrics are shared and a paragraph's leading is not.
	lineHeightMul float32
	RichText      bool
	OnLinkClicked func(string)
	colorToken    string // theme role ("secondary", …) — resolved per paint
	textCache     *graphics.Text
	textCacheVer  uint64
}

func NewLabel() *Label {
	return &Label{
		BaseWidget: *meui.NewBaseWidget(),
		LabelTheme: theme.DefaultLabelTheme,
		HAlign:     "left",
		VAlign:     "middle",
		WrapMode:   WrapNoWrap,
	}
}

func (l *Label) SetRichText(rich bool) *Label {
	l.RichText = rich
	l.MarkForRedraw()
	return l
}

func (l *Label) SetOnLinkClicked(fn func(string)) *Label {
	l.OnLinkClicked = fn
	return l
}

// parseHTML converts HTML text into rich text spans.
// Supports: <b>/<strong>, <i>/<em>, <u>, <s>/<strike>/<del>,
// <span style="color:#xxx">, <a href="...">, <br>, &amp; &lt; &gt; &quot; &nbsp; &#NNN;
func (l *Label) parseHTML(text string) []richSpan {
	var spans []richSpan
	i := 0
	type tagState struct {
		bold      bool
		italic    bool
		underline bool
		strike    bool
		color     *graphics.Color
		link      string
	}
	var stack []tagState

	current := tagState{}

	push := func(s tagState) { stack = append(stack, current); current = s }
	pop := func() {
		if len(stack) > 0 {
			current = stack[len(stack)-1]
			stack = stack[:len(stack)-1]
		}
	}

	flushText := func(t string) {
		if t == "" {
			return
		}
		if len(spans) > 0 {
			last := &spans[len(spans)-1]
			if last.Bold == current.bold && last.Italic == current.italic &&
				last.Underline == current.underline && last.Strike == current.strike &&
				last.Link == current.link &&
				(last.Color == nil && current.color == nil ||
					last.Color != nil && current.color != nil && *last.Color == *current.color) {
				last.Text += t
				return
			}
		}
		spans = append(spans, richSpan{
			Text: t, Bold: current.bold, Italic: current.italic,
			Underline: current.underline, Strike: current.strike,
			Color: current.color, Link: current.link,
		})
	}

	for i < len(text) {
		if text[i] == '&' {
			end := strings.IndexByte(text[i:], ';')
			if end >= 0 {
				entity := text[i+1 : i+end]
				var ch string
				switch entity {
				case "amp":
					ch = "&"
				case "lt":
					ch = "<"
				case "gt":
					ch = ">"
				case "quot":
					ch = "\""
				case "nbsp":
					ch = " "
				default:
					if strings.HasPrefix(entity, "#") {
						if code, err := strconv.Atoi(entity[1:]); err == nil {
							ch = string(rune(code))
						}
					}
				}
				if ch != "" {
					flushText(ch)
				}
				i += end + 1
				continue
			}
		}

		if text[i] == '<' {
			end := strings.IndexByte(text[i:], '>')
			if end < 0 {
				flushText(string(text[i:]))
				break
			}
			tag := text[i+1 : i+end]
			tagLower := strings.ToLower(tag)
			i += end + 1

			if tagLower == "br" || tagLower == "br/" {
				flushText("\n")
				continue
			}
			if tagLower == "/b" || tagLower == "/strong" {
				pop()
				continue
			}
			if tagLower == "/i" || tagLower == "/em" {
				pop()
				continue
			}
			if tagLower == "/u" || tagLower == "/s" || tagLower == "/strike" || tagLower == "/del" {
				pop()
				continue
			}
			if tagLower == "/a" || tagLower == "/span" {
				pop()
				continue
			}

			next := current
			switch tagLower {
			case "b", "strong":
				next.bold = true
			case "i", "em":
				next.italic = true
			case "u":
				next.underline = true
			case "s", "strike", "del":
				next.strike = true
			}
			if strings.HasPrefix(tagLower, "a ") {
				if m := linkRE.FindStringSubmatch(tag); m != nil {
					next.link = m[1]
				}
			}
			if strings.HasPrefix(tagLower, "span") {
				if m := colorRE.FindStringSubmatch(tag); m != nil {
					if c := parseColor(m[1]); c != nil {
						next.color = c
					}
				}
			}
			push(next)
			continue
		}

		// Find next tag or entity boundary
		next := strings.IndexAny(text[i:], "<&")
		if next < 0 {
			flushText(text[i:])
			break
		}
		flushText(text[i : i+next])
		i += next
	}
	return spans
}

var linkRE = regexp.MustCompile(`href\s*=\s*"([^"]*)"`)
var colorRE = regexp.MustCompile(`color\s*:\s*#([0-9a-fA-F]+)`)

func parseColor(hex string) *graphics.Color {
	if len(hex) == 0 {
		return nil
	}
	// Pad to 6 or 8 hex digits
	switch len(hex) {
	case 3:
		hex = string(hex[0]) + string(hex[0]) + string(hex[1]) + string(hex[1]) + string(hex[2]) + string(hex[2])
	case 6:
	case 8:
	default:
		return nil
	}
	u, err := strconv.ParseUint(hex, 16, 64)
	if err != nil {
		return nil
	}
	c := graphics.Color(u)
	if len(hex) == 6 {
		c |= 0xFF000000
	}
	return &c
}

func (l *Label) buildFont(base graphics.Font, bold, italic bool) graphics.Font {
	if !bold && !italic {
		return base
	}
	fd := base.Descriptor()
	if bold {
		fd.Weight = weight.Black
	}
	if italic {
		fd.Slant = slant.Italic
	}
	f := fd.Font()
	if f != nil {
		return f
	}
	return base
}

func (l *Label) SetText(text string) *Label {
	l.Text = text
	l.textCache = nil
	l.MarkForRedraw()
	return l
}

// SetColorToken pins the label to a theme role (e.g. "secondary") so
// runtime theme switches recolour it. It wins over SetColor until
// cleared.
func (l *Label) SetColorToken(name string) *Label {
	l.colorToken = name
	return l
}

// EffectiveTextInk resolves the label's ink for the current theme: the
// token when one is set (live), otherwise the per-label colour.
func (l *Label) EffectiveTextInk() graphics.Color {
	if l.colorToken != "" {
		if c, ok := theme.TokenColor(l.colorToken); ok {
			return c
		}
	}
	return l.LabelTheme.TextInk
}

func (l *Label) SetColor(color graphics.Color) *Label {
	l.LabelTheme = l.LabelTheme.Copy() // copy-on-write: don't mutate the shared theme
	l.LabelTheme.TextInk = color
	l.MarkForRedraw()
	return l
}

// SetFont sets a custom font for this label (copy-on-write).
func (l *Label) SetFont(font graphics.Font) *Label {
	l.LabelTheme = l.LabelTheme.Copy()
	l.LabelTheme.Font = font
	l.textCache = nil
	l.MarkForRedraw()
	return l
}

// SetFontSize sets the label's font size, deriving from the current font.
func (l *Label) SetFontSize(size float32) *Label {
	l.LabelTheme = l.LabelTheme.Copy()
	base := l.LabelTheme.Font
	if base == nil {
		base = theme.DefaultTheme.BaseFont
		if base == nil {
			base = graphics.SystemFont
		}
	}
	if face := base.Face(); face != nil {
		if f := face.Font(size); f != nil {
			l.LabelTheme.Font = f
		}
	}
	l.textCache = nil
	l.MarkForRedraw()
	return l
}

// SetFontBold renders the label text in a bold face.
func (l *Label) SetFontBold(bold bool) *Label {
	l.LabelTheme = l.LabelTheme.Copy()
	base := l.LabelTheme.Font
	if base == nil {
		base = theme.DefaultTheme.BaseFont
		if base == nil {
			base = graphics.SystemFont
		}
	}
	l.LabelTheme.Font = l.buildFont(base, bold, false)
	l.textCache = nil
	l.MarkForRedraw()
	return l
}

// SetFontItalic renders the label text in an italic face.
func (l *Label) SetFontItalic(italic bool) *Label {
	l.LabelTheme = l.LabelTheme.Copy()
	base := l.LabelTheme.Font
	if base == nil {
		base = theme.DefaultTheme.BaseFont
		if base == nil {
			base = graphics.SystemFont
		}
	}
	l.LabelTheme.Font = l.buildFont(base, false, italic)
	l.textCache = nil
	l.MarkForRedraw()
	return l
}

func (l *Label) SetHAlign(align string) *Label {
	l.HAlign = align
	l.MarkForRedraw()
	return l
}

func (l *Label) SetVAlign(align string) *Label {
	l.VAlign = align
	l.MarkForRedraw()
	return l
}

func (l *Label) SetSecondary(secondary bool) *Label {
	l.Secondary = secondary
	l.MarkForRedraw()
	return l
}

// SetMaxLines clamps a wrapped label to n lines, ellipsizing the last.
func (l *Label) SetMaxLines(n int) *Label {
	if n < 0 {
		n = 0
	}
	l.MaxLines = n
	l.MarkForRedraw()
	return l
}

// SetLineHeight scales the label's line height (1.55 is a web body, 1.15 a
// heading). Zero restores the font's own.
func (l *Label) SetLineHeight(mul float32) *Label {
	if mul < 0 {
		mul = 0
	}
	l.lineHeightMul = mul
	l.MarkForRedraw()
	return l
}

func (l *Label) SetWrapMode(mode string) *Label {
	l.WrapMode = mode
	l.MarkForRedraw()
	return l
}

// wrapText delegates to the text core: the label owns WHAT it draws, not how the
// lines are measured. See wrapLabelText in textlayout.go.
func (l *Label) wrapText(font graphics.Font, maxWidth float32) []string {
	return wrapLabelText(font, l.Text, maxWidth, l.WrapMode, l.MaxLines)
}

func (l *Label) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !l.Visible() {
		return
	}

	canvas.Save()

	font := l.LabelTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	ascent := font.Baseline()
	fontHeight := font.LineHeight()
	if l.lineHeightMul > 0 {
		fontHeight = font.Size() * l.lineHeightMul
	}

	if l.RichText {
		l.paintRichText(canvas, rect, font, ascent, fontHeight)
	} else {
		l.paintPlainText(canvas, rect, font, ascent, fontHeight)
	}

	canvas.Restore()
}

func (l *Label) paintPlainText(canvas *graphics.Canvas, rect graphics.Rect, font graphics.Font, ascent, fontHeight float32) {
	lines := l.wrapText(font, rect.Width)

	textPaint := graphics.NewPaint()
	if !l.Enabled() {
		textPaint.SetColor(theme.DefaultTheme.DisabledText)
	} else if l.Secondary {
		textPaint.SetColor(l.LabelTheme.TextSecondary)
	} else {
		textPaint.SetColor(l.EffectiveTextInk())
	}

	totalHeight := float32(len(lines)) * fontHeight
	startY := l.alignY(rect, totalHeight)

	if os.Getenv("MEUI_LAYOUT_DEBUG") != "" && l.Text == "hcenter" {
		ox, oy := canvas.Origin()
		fmt.Printf("[labelPaint hcenter] rect=(%.1f,%.1f %.1fx%.1f) textX=%.1f origin=(%.1f,%.1f)\n",
			rect.X, rect.Y, rect.Width, rect.Height, l.alignX(rect, font, lines[0]), ox, oy)
	}

	if len(lines) == 1 {
		line := lines[0]
		x := l.alignX(rect, font, line)
		y := startY + ascent
		ver := theme.ThemeVersion()
		if l.textCache == nil || l.textCacheVer != ver {
			l.textCache = graphics.NewText(line, &graphics.TextDecoration{Font: font, OnBackgroundInk: textPaint.Color()})
			l.textCacheVer = ver
		}
		l.textCache.Draw(canvas, x, y)
	} else {
		for i, line := range lines {
			x := l.alignX(rect, font, line)
			y := startY + float32(i)*fontHeight + ascent
			lt := graphics.NewText(line, &graphics.TextDecoration{Font: font, OnBackgroundInk: textPaint.Color()})
			lt.Draw(canvas, x, y)
		}
	}
}

func (l *Label) paintRichText(canvas *graphics.Canvas, rect graphics.Rect, font graphics.Font, ascent, fontHeight float32) {
	spans := l.parseHTML(l.Text)

	textInk := l.EffectiveTextInk()
	if !l.Enabled() {
		textInk = theme.DefaultTheme.DisabledText
	} else if l.Secondary {
		textInk = l.LabelTheme.TextSecondary
	}

	// Layout rich text into lines with word wrap
	type positionedSpan struct {
		span  richSpan
		x, y  float32
		width float32
	}
	var positioned []positionedSpan

	x := rect.X
	y := rect.Y + ascent
	maxW := rect.Width
	if os.Getenv("MEUI_LAYOUT_DEBUG") != "" && strings.Contains(l.Text, "Wordwrap test") {
		fmt.Printf("[richLayout] spans=%d maxW=%.0f textLen=%d\n", len(spans), maxW, len(l.Text))
	}

	for _, s := range spans {
		if s.Text == "" {
			continue
		}
		parts := strings.Split(s.Text, "\n")
		for pi, part := range parts {
			if pi > 0 {
				// Line break
				x = rect.X
				y += fontHeight
			}
			if part == "" {
				continue
			}
			spanFont := l.buildFont(font, s.Bold, s.Italic)
			w := spanFont.SimpleWidth(part)
			if os.Getenv("MEUI_COMBO_DEBUG") != "" {
				ox, oy := canvas.Origin()
				sb, si := spanFont.Synthetic()
				fmt.Printf("[richLayout] part=%q w=%.1f bold=%v italic=%v x=%.1f size=%.1f origin=(%.1f,%.1f) syn=(%v,%v)\n", part, w, s.Bold, s.Italic, x, spanFont.Size(), ox, oy, sb, si)
			}
			if maxW <= 0 || x+w <= rect.X+maxW {
				// Fits on the current line — place whole (spacing intact).
				// Fragment copy: the span may contain "\n" for LATER
				// parts — storing the full span text here made DrawText
				// re-run the embedded newline and double-render lines.
				fs := s
				fs.Text = part
				positioned = append(positioned, positionedSpan{span: fs, x: x, y: y, width: w})
				x += w
				continue
			}
			// Overflows: wrap at word boundaries. Spaces are rendered as an
			// explicit advance between word fragments (deterministic —
			// baked-in fragment spaces desynced the x advance).
			spaceW := spanFont.SimpleWidth(" ")
			runes := []rune(part)
			i2 := 0
			for i2 < len(runes) {
				// skip spaces, remembering the leading run
				spaceRun := float32(0)
				for i2 < len(runes) && runes[i2] == ' ' {
					spaceRun += spaceW
					i2++
				}
				if i2 >= len(runes) {
					break
				}
				wStart := i2
				for i2 < len(runes) && runes[i2] != ' ' {
					i2++
				}
				word := string(runes[wStart:i2])
				wordW := spanFont.SimpleWidth(word)
				if x > rect.X && maxW > 0 && x+spaceRun+wordW > rect.X+maxW {
					x = rect.X
					y += fontHeight
					spaceRun = 0
				}
				x += spaceRun
				fs := s
				fs.Text = word
				positioned = append(positioned, positionedSpan{span: fs, x: x, y: y, width: wordW})
				x += wordW
			}
		}
	}

	if len(positioned) == 0 {
		return
	}
	if os.Getenv("MEUI_COMBO_DEBUG") != "" {
		for _, ps := range positioned {
			if strings.Contains(ps.span.Text, "Break") || strings.Contains(ps.span.Text, "line") {
				fmt.Printf("[richPos] %q x=%.1f y=%.1f w=%.1f\n", ps.span.Text, ps.x, ps.y, ps.width)
			}
		}
	}

	lastY := positioned[len(positioned)-1].y
	totalHeight := (lastY - positioned[0].y) + fontHeight
	offsetY := l.alignY(rect, totalHeight) - (positioned[0].y - ascent)

	for _, ps := range positioned {
		spanFont := l.buildFont(font, ps.span.Bold, ps.span.Italic)
		col := textInk
		if ps.span.Color != nil {
			col = *ps.span.Color
		}
		if ps.span.Link != "" {
			col = graphics.Color(0xFF569CD6) // link blue
		}
		tp := graphics.NewPaint()
		tp.SetColor(col)
		DrawText(canvas, ps.span.Text, ps.x, ps.y+offsetY, spanFont, tp)
		if ps.span.Underline || ps.span.Strike || ps.span.Link != "" {
			dp := graphics.NewPaint()
			dp.SetStyle(paintstyle.Stroke)
			dp.SetStrokeWidth(1)
			dp.SetColor(col)
			baseline := ps.y + offsetY
			if ps.span.Underline || ps.span.Link != "" {
				canvas.DrawLine(ps.x, baseline+2, ps.x+ps.width, baseline+2, dp)
			}
			if ps.span.Strike {
				// Mid-x-height: ~35% of the ascent above the baseline.
				strikeY := baseline - spanFont.Baseline()*0.35
				canvas.DrawLine(ps.x, strikeY, ps.x+ps.width, strikeY, dp)
			}
		}
	}
}

func (l *Label) alignY(rect graphics.Rect, totalHeight float32) float32 {
	switch l.VAlign {
	case "top":
		return rect.Y
	case "bottom":
		return rect.Y + rect.Height - totalHeight
	default: // "middle"
		return rect.Y + (rect.Height-totalHeight)/2
	}
}

func (l *Label) alignX(rect graphics.Rect, font graphics.Font, line string) float32 {
	lineWidth := font.SimpleWidth(line)
	switch l.HAlign {
	case "right":
		return rect.X + rect.Width - lineWidth
	case "center":
		return rect.X + (rect.Width-lineWidth)/2
	default: // "left"
		return rect.X
	}
}

func (l *Label) OnMouseDown(x, y float32, button int) bool {
	if !l.RichText || l.OnLinkClicked == nil || l.Text == "" {
		return false
	}
	link := l.linkAt(x, y)
	if link != "" {
		if l.OnLinkClicked != nil {
			l.OnLinkClicked(link)
		} else {
			meui.OpenURL(link)
		}
		return true
	}
	return false
}

// linkAt returns the link URL at the given position, or empty string if no link.
func (l *Label) linkAt(x, y float32) string {
	font := l.LabelTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	fontHeight := font.LineHeight()
	ascent := font.Baseline()

	spans := l.parseHTML(l.Text)

	// Compute the full layout to account for alignment and wrapping
	type spanPos struct {
		span  richSpan
		x, y  float32
		width float32
	}
	var positions []spanPos

	cx := float32(0)
	cy := ascent
	frame := l.FrameRect()
	maxW := frame.Width

	for _, s := range spans {
		if s.Text == "" {
			continue
		}
		parts := strings.Split(s.Text, "\n")
		for pi, part := range parts {
			if pi > 0 {
				cx = 0
				cy += fontHeight
			}
			if part == "" {
				continue
			}
			spanFont := l.buildFont(font, s.Bold, s.Italic)
			w := spanFont.SimpleWidth(part)
			if maxW > 0 && cx+w > maxW && cx > 0 {
				cx = 0
				cy += fontHeight
			}
			positions = append(positions, spanPos{span: s, x: cx, y: cy, width: w})
			cx += w
		}
	}
	if len(positions) == 0 {
		return ""
	}

	// Compute VAlign offset
	lastY := positions[len(positions)-1].y
	totalHeight := (lastY - positions[0].y) + fontHeight
	var yOff float32
	switch l.VAlign {
	case "top":
		yOff = 0
	case "bottom":
		yOff = frame.Height - totalHeight
	default:
		yOff = (frame.Height - totalHeight) / 2
	}
	// HAlign is applied per-line; only the first span on each line is left-aligned.
	// For simplicity, we group spans by line Y.
	type lineGroup struct {
		y      float32
		spans  []spanPos
		totalW float32
	}
	lineMap := make(map[float32]*lineGroup)
	var lineOrder []float32
	for _, p := range positions {
		g, ok := lineMap[p.y]
		if !ok {
			g = &lineGroup{y: p.y}
			lineMap[p.y] = g
			lineOrder = append(lineOrder, p.y)
		}
		g.spans = append(g.spans, p)
		g.totalW += p.width
	}
	for _, ly := range lineOrder {
		g := lineMap[ly]
		var xOff float32
		switch l.HAlign {
		case "right":
			xOff = frame.Width - g.totalW
		case "center":
			xOff = (frame.Width - g.totalW) / 2
		}
		for i := range g.spans {
			g.spans[i].x += xOff
		}
	}

	for _, p := range positions {
		if p.span.Link != "" && y >= p.y+yOff-ascent && y <= p.y+yOff-ascent+fontHeight && x >= p.x && x <= p.x+p.width {
			return p.span.Link
		}
	}
	return ""
}

func (l *Label) CursorAt(x, y float32) *platform.Cursor {
	if !l.RichText {
		return nil
	}
	// CursorAt is called with widget-local coordinates (the window converts
	// once); converting again here measured the link at the wrong point.
	if l.linkAt(x, y) == "" {
		return nil
	}
	return platform.PointingCursor()
}

func (l *Label) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	if l.Text == "" {
		return 50, 30
	}
	font := l.LabelTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	if l.RichText {
		// Rich labels: measure per-span (styled fonts have different
		// metrics) and count <br> lines — the raw string would count
		// tags as text and miss line breaks, under-sizing the label
		// so the next row overlaps the wrapped lines.
		spans := l.parseHTML(l.Text)
		lines := 1
		tw := float32(0)
		cur := float32(0)
		for _, s := range spans {
			spanFont := l.buildFont(font, s.Bold, s.Italic)
			parts := strings.Split(s.Text, "\n")
			for pi, part := range parts {
				if pi > 0 {
					lines++
					if cur > tw {
						tw = cur
					}
					cur = 0
				}
				cur += spanFont.SimpleWidth(part)
			}
		}
		if cur > tw {
			tw = cur
		}
		height := float32(lines) * font.LineHeight()
		if tw < 50 {
			tw = 50
		}
		if height < 30 {
			height = 30
		}
		return tw, height
	}
	tw := font.SimpleWidth(l.Text)
	lh := font.LineHeight()
	if tw < 50 {
		tw = 50
	}
	if lh < 30 {
		lh = 30
	}
	return tw, lh
}

func (l *Label) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "text":
		if s, ok := value.(string); ok {
			l.SetText(s)
		}
	case "richText":
		if b, ok := value.(bool); ok {
			l.SetRichText(b)
		}
	case "horizontalAlignment":
		if s, ok := value.(string); ok {
			l.SetHAlign(s)
		}
	case "verticalAlignment":
		if s, ok := value.(string); ok {
			l.SetVAlign(s)
		}
	case "secondary", "secondaryText":
		if b, ok := value.(bool); ok {
			l.SetSecondary(b)
		}
	case "wrapMode", "elide":
		if s, ok := value.(string); ok {
			l.SetWrapMode(s)
		}
	}
}
