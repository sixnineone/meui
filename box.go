package meui

// The box painters: a soft shadow and a focus indicator, both drawn from theme
// tokens. They live in the CORE rather than in package widgets because the core's
// Panel is the toolkit's generic box (QML's Rectangle), and a box that cannot
// shadow itself or show where the keyboard is cannot render a web surface.
//
// There is exactly ONE implementation of each. widgets.drawShadow and
// widgets.drawFocusIndicator delegate here, so a Button and a Panel cannot paint
// two different focus rings — the drift this tree has already paid for once with
// TabStrip/TabPanel.

import (
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/theme"
)

// DrawShadow paints a soft drop shadow behind r using concentric rounded rects
// with decaying alpha — no FBO round trip (unlike the blur-pipeline DropShadow),
// so it batches with the frame's SDF quads and costs one quad ring per step.
//
// Draw BEFORE the widget's own background; the background covers the interior and
// only the fringe remains.
func DrawShadow(c *graphics.Canvas, r graphics.Rect, radius float32, spec theme.ShadowSpec) {
	if c == nil || spec.Color == 0 {
		return
	}
	reach := spec.Blur*0.5 + spec.Spread
	if reach <= 0 {
		return
	}
	baseA := float32((spec.Color>>24)&0xFF) / 255
	rgb := spec.Color & 0x00FFFFFF
	const steps = 6
	const gain = 1.6 // ring alpha budget so the fringe reads at spec alpha
	for i := steps; i >= 1; i-- {
		t := float32(i) / float32(steps)
		grow := reach * t
		a := uint32(baseA * (1 - t) * (255 / steps) * gain)
		if a == 0 {
			continue
		}
		p := graphics.NewPaint()
		p.SetColor(graphics.Color(a<<24) | rgb)
		rr := graphics.NewRect(
			r.X+spec.DX-grow, r.Y+spec.DY-grow,
			r.Width+2*grow, r.Height+2*grow,
		)
		c.DrawRoundedRect(rr, radius+grow, radius+grow, p)
	}
}

// ShadowToken resolves a named shadow from the active theme. It exists so a
// scene can say `shadow: "card"` and stay theme-driven: the preset decides what a
// card's elevation is, and a dark preset can make it deeper without touching the
// QML.
func ShadowToken(name string) (theme.ShadowSpec, bool) {
	t := theme.DefaultTheme
	if t == nil {
		return theme.ShadowSpec{}, false
	}
	switch name {
	case "card", "surface", "elevated":
		return t.Effects.CardShadow, true
	case "popup", "dialog", "menu":
		return t.Effects.PopupShadow, true
	case "window":
		return t.Effects.WindowShadow, true
	case "none", "":
		return theme.ShadowSpec{}, false
	}
	return theme.ShadowSpec{}, false
}

// DrawFocusRing paints the active platform's focus indicator (theme
// Effects.Focus) around or inside r. Kind selects the construction:
//
//	ring      outer accent ring (macOS, Adwaita) — offset grows it out
//	inset     accent stroke inside the bounds (Breeze)
//	two-tone  contrasting double rect (Fluent)
//	underline accent bar along the bottom edge (Metro-ish text fields)
//
// Call when the widget owns focus; radius matches the widget's corner radius so
// the indicator follows its shape.
func DrawFocusRing(c *graphics.Canvas, r graphics.Rect, radius float32) {
	DrawFocusRingInk(c, r, radius, 0)
}

// DrawFocusRingInk is DrawFocusRing with an explicit colour for the ring. A field in an
// error state uses its error ink: the accent ring drawn on top of a red border read as
// "focused, fine" when the point was "this value is wrong".
func DrawFocusRingInk(c *graphics.Canvas, r graphics.Rect, radius float32, ink graphics.Color) {
	if c == nil || theme.DefaultTheme == nil {
		return
	}
	ringInk := ink
	f := theme.DefaultTheme.Effects.Focus
	if f.Kind == theme.FocusNone || f.Width <= 0 {
		return
	}
	p := graphics.NewPaint()
	p.SetStyle(paintstyle.Stroke)
	switch f.Kind {
	case theme.FocusRing:
		p.SetStrokeWidth(f.Width)
		if ringInk != 0 {
			p.SetColor(ringInk)
		} else {
			p.SetColor(f.Color)
		}
		off := f.Offset
		rr := graphics.NewRect(r.X-off, r.Y-off, r.Width+2*off, r.Height+2*off)
		c.DrawRoundedRect(rr, radius+off, radius+off, p)

	case theme.FocusInset:
		p.SetStrokeWidth(f.Width)
		if ringInk != 0 {
			p.SetColor(ringInk)
		} else {
			p.SetColor(f.Color)
		}
		inset := f.Width / 2
		rr := graphics.NewRect(r.X+inset, r.Y+inset, r.Width-2*inset, r.Height-2*inset)
		c.DrawRoundedRect(rr, radius, radius, p)

	case theme.FocusTwoTone:
		outer, inner := f.Color, f.Alt
		if outer == 0 {
			outer = 0xFFFFFFFF
		}
		if inner == 0 {
			inner = 0xFF000000
		}
		p.SetStrokeWidth(f.Width)
		p.SetColor(outer)
		c.DrawRoundedRect(graphics.NewRect(r.X+1, r.Y+1, r.Width-2, r.Height-2), radius, radius, p)
		p.SetColor(inner)
		ir := graphics.NewRect(r.X+1+f.Width, r.Y+1+f.Width, r.Width-2-2*f.Width, r.Height-2-2*f.Width)
		c.DrawRoundedRect(ir, radius, radius, p)

	case theme.FocusUnderline:
		p.SetStrokeWidth(f.Width)
		p.SetColor(f.Color)
		y := r.Y + r.Height - f.Width/2
		c.DrawLine(r.X+radius, y, r.X+r.Width-radius, y, p)
	}
}

// ScaleGradient resolves a unit gradient (Start/End/Center in 0..1 of the shape)
// against a concrete rect. The gradient types express geometry in the shape's own
// pixels, which a scene author does not know until layout runs.
func ScaleGradient(g graphics.Gradient, r graphics.Rect) graphics.Gradient {
	if g.Kind == graphics.GradientNone {
		return g
	}
	w, h := r.Width, r.Height
	g.Start = graphics.Point{X: r.X + g.Start.X*w, Y: r.Y + g.Start.Y*h}
	g.End = graphics.Point{X: r.X + g.End.X*w, Y: r.Y + g.End.Y*h}
	g.Center = graphics.Point{X: r.X + g.Center.X*w, Y: r.Y + g.Center.Y*h}
	if g.Radius > 0 && g.Radius <= 1 {
		g.Radius *= w
		if h < w {
			g.Radius *= h / w
		}
	}
	return g
}

// BlendOver composites src OVER dst in straight (non-premultiplied) alpha. A Box
// uses it to paint its hover and press washes: the theme supplies a translucent
// ink, and the box keeps its own fill underneath.
func BlendOver(dst, src graphics.Color) graphics.Color {
	sa := float32((src>>24)&0xFF) / 255
	if sa <= 0 {
		return dst
	}
	if sa >= 1 {
		return src
	}
	da := float32((dst>>24)&0xFF) / 255
	outA := sa + da*(1-sa)
	if outA <= 0 {
		return 0
	}
	ch := func(d, s uint32) uint32 {
		v := (float32(s)/255*sa + float32(d)/255*da*(1-sa)) / outA
		if v > 1 {
			v = 1
		}
		return uint32(v*255 + 0.5)
	}
	return graphics.Color(uint32(outA*255+0.5)<<24 |
		ch(uint32(dst>>16)&0xFF, uint32(src>>16)&0xFF)<<16 |
		ch(uint32(dst>>8)&0xFF, uint32(src>>8)&0xFF)<<8 |
		ch(uint32(dst)&0xFF, uint32(src)&0xFF))
}
