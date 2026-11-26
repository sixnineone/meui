package graphics

import (
	"fmt"
	"os"
	"strings"
)

// Ink is anything that can produce a Paint for a rect. It is meui-owned —
// no backend type appears in the signature. Color satisfies it, which is
// what TextDecoration.OnBackgroundInk is normally set to.
type Ink interface {
	Paint(canvas *Canvas, rect Rect, style int32) *Paint
}

// Text is a stub matching the unison.Text API surface used by meui.
// A faithful implementation needs shaping, layout, and per-glyph
// metrics that mobilegles does not expose. Until then, Text.Draw
// forwards to Canvas.DrawText with the decoration's font.
type Text struct {
	str        string
	decoration *TextDecoration
}

type TextDecoration struct {
	Font            Font
	OnBackgroundInk Ink
	StrikeThrough   bool
}

func NewText(str string, decoration *TextDecoration) *Text {
	return &Text{str: str, decoration: decoration}
}

// Width returns the rendered width of the text through the unified
// text engine (same walk DrawText uses).
func (t *Text) Width() float32 {
	if t == nil || t.decoration == nil || t.decoration.Font == nil {
		return 0
	}
	return t.decoration.Font.TextWidth(t.str)
}

// AddString appends a string with the given decoration. Stub: the
// meui code that uses it (markdown rendering) ignores the result
// for now.
func (t *Text) AddString(s string, dec *TextDecoration) {
	if t == nil {
		return
	}
	if t.decoration == nil {
		t.decoration = dec
	}
	t.str += s
}

func (t *Text) Draw(canvas *Canvas, x, y float32) {
	if t == nil || t.decoration == nil || t.decoration.Font == nil {
		return
	}
	if os.Getenv("MEUI_FLUSH_DEBUG") != "" && strings.Contains(t.str, "Basic") {
		fmt.Printf("[titleText] str=%q x=%.1f y=%.1f font=%v\n", t.str, x, y, t.decoration.Font)
	}
	p := NewPaint()
	if ink := t.decoration.OnBackgroundInk; ink != nil {
		if ip := ink.Paint(nil, Rect{}, 0); ip != nil {
			p.SetColor(ip.Color())
		}
	}
	canvas.DrawText(t.str, x, y, t.decoration.Font, p)
}
