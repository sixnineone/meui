package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/graphics/enums/weight"
	"github.com/sixnineone/meui/layout"
	"github.com/sixnineone/meui/theme"
)

// Collapsible is a header over one child, where the header shows and the child does not.
//
// WHY IT EXISTS SEPARATELY FROM GroupBox. A GroupBox is a BORDERED, titled section: it says
// "these controls belong together" and it looks like a panel. A transcript row says something
// different — "there is more here, and it is currently folded" — and it wants no border, no
// inset and a chevron. Forcing one widget to be both means every use turns off half of it,
// which is how a GroupBox ends up with ShowBorder false and a title that is really a header.
//
// The collapse arithmetic is GroupBox's, and it is not re-implemented here: a container whose
// collapsed height is a SUM is easy to get subtly wrong in two places at once, and the
// contract that matters is the same for both. see groupBoxOverhead / groupBoxTitleHeight.
type Collapsible struct {
	meui.BaseWidget
	*theme.GroupBoxTheme

	label    string
	trailing string
	// detail is the one-line summary beside the label. It is what a transcript row shows:
	// the kind, then what it did.
	detail string
	// accent marks the row as the current one — a selected destination, the running step.
	accent bool
	// busy draws the trailing text in the accent ink, which is how a running step reads
	// against a finished one without needing an icon set for it.
	busy bool

	collapsed bool
	hovered   bool
	body      *meui.Panel

	// OnToggle fires when the header is used, so an app can persist the state or record
	// that the user opened it.
	OnToggle func(collapsed bool)
}

// NewCollapsible builds an EXPANDED collapsible. Expanded is the default because a widget
// that arrives with its content hidden reads as empty, and a transcript that starts folded
// is a transcript that looks broken.
func NewCollapsible() *Collapsible {
	c := &Collapsible{
		// BaseWidget must be initialised, not left zero: NewBaseWidget is what sets
		// enabled, and a zero BaseWidget is a DISABLED widget that silently refuses every
		// press. This is the second time this exact omission has been made in this package
		// (NavigationBar was the first), which is worth recording as a trap rather than a
		// slip — it compiles, it paints, and it is dead to input.
		BaseWidget:    *meui.NewBaseWidget(),
		GroupBoxTheme: theme.DefaultGroupBoxTheme,
		collapsed:     false,
		body:          meui.NewPanel(),
	}
	v := layout.NewVStack()
	v.Spacing = 0
	c.body.SetLayout(v)
	c.body.SetParent(c)
	return c
}

func (c *Collapsible) SetLabel(label string) *Collapsible {
	c.label = label
	c.MarkForRedraw()
	return c
}

func (c *Collapsible) Label() string { return c.label }

// SetDetail sets the one-line summary beside the label.
func (c *Collapsible) SetDetail(detail string) *Collapsible {
	c.detail = detail
	c.MarkForRedraw()
	return c
}

// SetTrailing sets the right-hand text — a duration, a count, a state.
func (c *Collapsible) SetTrailing(trailing string) *Collapsible {
	c.trailing = trailing
	c.MarkForRedraw()
	return c
}

// SetAccent marks the row as the current one.
func (c *Collapsible) SetAccent(on bool) *Collapsible {
	c.accent = on
	c.MarkForRedraw()
	return c
}

// SetBusy draws the trailing text in the accent ink: a running step against a finished one.
func (c *Collapsible) SetBusy(on bool) *Collapsible {
	c.busy = on
	c.MarkForRedraw()
	return c
}

// SetBody replaces the content.
func (c *Collapsible) SetBody(w meui.Widget) *Collapsible {
	if w == nil {
		return c
	}
	w.SetParent(c.body)
	c.body.ClearChildren()
	c.body.AddChild(w)
	c.SetNeedsLayout()
	c.MarkForRedraw()
	return c
}

// Body returns the content container, for AddChild.
func (c *Collapsible) Body() *meui.Panel { return c.body }

func (c *Collapsible) Collapsed() bool { return c.collapsed }

func (c *Collapsible) SetCollapsed(v bool) *Collapsible {
	if c.collapsed == v {
		return c
	}
	c.collapsed = v
	// Marking ITSELF is not enough: the row's frame is set by the PARENT's layout, so a
	// widget that resizes itself has to invalidate its ancestors too. Without this the
	// toggle worked and the row did not move — the state changed and the pixels did not,
	// which is the worst of both.
	c.SetNeedsLayout()
	c.InvalidateAncestors()
	c.MarkForRedraw()
	if c.OnToggle != nil {
		c.OnToggle(v)
	}
	return c
}

func (c *Collapsible) Toggle() { c.SetCollapsed(!c.collapsed) }

// ── geometry ────────────────────────────────────────────────────────

// HeaderHeight is the header's height. It comes from the theme rather than from measuring
// the label, because AGENTS.md is explicit that a size depending on text must not be
// measured before the font loads — a header that is short at construction opens already
// showing its overflow.
func (c *Collapsible) HeaderHeight() float32 {
	f := c.font()
	h := f.LineHeight() + 12
	if floor := theme.DefaultTheme.Metrics.MinTouchTarget; h < floor {
		h = floor
	}
	return h
}

// PreferredDimensions reports the header alone when collapsed — the whole point, and the
// reason a list of a thousand collapsed rows can be placed in one pass. AGENTS.md: the
// layout consults PreferredDimensions, not PreferredSize, so this is the seam that works.
func (c *Collapsible) PreferredDimensions(target interface{}, width, height float32) (float32, float32) {
	head := c.HeaderHeight()
	if c.collapsed {
		return width, head
	}
	_, ch := c.body.PreferredDimensions(c.body, width, height)
	return width, head + ch
}

// SetBoundProperty implements the QML binding hook for label, detail, trailing and
// collapsed.
//
// WITHOUT IT the widget satisfies nothing the property dispatch looks for, so every QML
// property on it is silently dropped — which is exactly what happened: a Collapsible in a
// QML scene rendered with no label and the wrong collapsed state, and nothing said so.
// The symptom looks like a layout bug and is not one.
func (c *Collapsible) SetBoundProperty(name string, value interface{}) {
	switch name {
	case "label":
		if v, ok := value.(string); ok {
			c.SetLabel(v)
		}
	case "detail":
		if v, ok := value.(string); ok {
			c.SetDetail(v)
		}
	case "trailing":
		if v, ok := value.(string); ok {
			c.SetTrailing(v)
		}
	case "collapsed":
		if v, ok := value.(bool); ok {
			c.SetCollapsed(v)
		}
	}
}

func (c *Collapsible) Children() []meui.Widget  { return c.body.Children() }
func (c *Collapsible) ChildrenI() []interface{} { return c.body.ChildrenI() }

func (c *Collapsible) PanelAt(x, y float32) meui.Widget {
	if c.collapsed {
		return c
	}
	head := c.HeaderHeight()
	if y < head {
		return c
	}
	return c.body.PanelAt(x, y-head)
}

// ValidateLayout sizes the body panel to the space below the header.
//
// IT IS NOT A NO-OP, and that is a correction of an earlier version of this widget —
// the same mistake GroupBox carried, found the same way: with ValidateLayout doing
// nothing, the body is never given a frame, so a child that asks to FILL keeps the
// default widget size and its text wraps into a column about ninety pixels wide. It
// reads as the body failing to render rather than as a container that never laid it out.
func (c *Collapsible) ValidateLayout() {
	c.ClearNeedsLayout()
	if c.collapsed {
		// A collapsed row owns no body space, so the body is not laid out at all. That is
		// the point of collapsing it, and it is why a thousand of them cost a thousand
		// header measurements.
		return
	}
	head := c.HeaderHeight()
	f := c.FrameRect()
	if f.Height <= head {
		return
	}
	c.body.SetFrameRect(graphics.NewRect(0, head, f.Width, f.Height-head))
	if c.body.NeedsLayout() {
		c.body.ValidateLayout()
	}
}

// ── interaction ─────────────────────────────────────────────────────

// OnMouseDown toggles on a press INSIDE THE HEADER only, so a press in the body — which
// selects text, or presses a button — is not swallowed by the disclosure. A toggle on
// press is right for a header and wrong for the whole row.
func (c *Collapsible) OnMouseDown(x, y float32, button int) bool {
	if !c.Enabled() || button != mouseButtonLeft {
		return false
	}
	if y < 0 || y > c.HeaderHeight() {
		return c.collapsed || false
	}
	c.Toggle()
	return true
}

func (c *Collapsible) OnMouseMove(x, y float32) {
	h := y >= 0 && y <= c.HeaderHeight()
	if h != c.hovered {
		c.hovered = h
		c.MarkForRedraw()
	}
}

// HoverAt implements HoverTarget.
func (c *Collapsible) HoverAt(x, y float32) { c.OnMouseMove(x, y) }

// ── paint ───────────────────────────────────────────────────────────

func (c *Collapsible) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !c.Visible() || rect.Width < 1 || rect.Height < 1 {
		return
	}
	canvas.Save()
	canvas.Translate(rect.X, rect.Y)
	head := c.HeaderHeight()
	if head > rect.Height {
		head = rect.Height
	}

	// A hover wash, and an accent bar for the current row. No border: this is a row in a
	// list, not a panel, and a box around every collapsed group is what makes a transcript
	// look like a form.
	if c.hovered && theme.DefaultTheme.Hovered != 0 {
		hp := graphics.NewPaint()
		hp.SetColor(theme.DefaultTheme.Hovered)
		canvas.DrawRect(graphics.NewRect(0, 0, rect.Width, head), hp)
	}
	if c.accent {
		ap := graphics.NewPaint()
		ap.SetColor(theme.DefaultTheme.Accent)
		canvas.DrawRect(graphics.NewRect(0, 0, 2, head), ap)
	}

	font := c.font()
	base := font.Baseline()
	cy := head/2 + base/2 - 1

	// The chevron: drawn rather than looked up, so a disclosure never depends on an icon
	// being in the set. Right when collapsed, down when open.
	ink := c.TextInk
	chev := graphics.NewPaint()
	chev.SetColor(ink)
	var cx float32 = 12
	cyy := head / 2
	if c.collapsed {
		canvas.DrawLine(cx-2, cyy-4, cx+2, cyy, chev)
		canvas.DrawLine(cx+2, cyy, cx-2, cyy+4, chev)
	} else {
		canvas.DrawLine(cx-3, cyy-2, cx, cyy+2, chev)
		canvas.DrawLine(cx, cyy+2, cx+3, cyy-2, chev)
	}

	x := cx + 10
	if c.label != "" {
		lp := graphics.NewPaint()
		lp.SetColor(ink)
		t := graphics.NewText(c.label, &graphics.TextDecoration{Font: labelFont(font), OnBackgroundInk: ink})
		t.Draw(canvas, x, cy)
		x += t.Width() + 8
	}
	if c.detail != "" {
		dp := graphics.NewPaint()
		dp.SetColor(c.TextSecondary)
		t := graphics.NewText(c.detail, &graphics.TextDecoration{Font: font, OnBackgroundInk: c.TextSecondary})
		// CLIPPED, and the detail is the first thing to give up room: the label is the
		// KIND, and a transcript without kinds is a wall of text. Clipping rather than
		// truncating is what GroupBox's own title does, so the two agree.
		avail := rect.Width - x - 8
		if tr, ok := c.trailingRect(rect.Width, font); ok {
			avail = tr.X - x - 8
		}
		canvas.Save()
		canvas.ClipRect(graphics.NewRect(x-1, 0, avail, head), pathop.Intersect, false)
		t.Draw(canvas, x, cy)
		canvas.Restore()
	}
	if tr, ok := c.trailingRect(rect.Width, font); ok && c.trailing != "" {
		tink := c.TextSecondary
		if c.busy {
			tink = theme.DefaultTheme.Accent
		}
		tp := graphics.NewPaint()
		tp.SetColor(tink)
		t := graphics.NewText(c.trailing, &graphics.TextDecoration{Font: font, OnBackgroundInk: tink})
		t.Draw(canvas, tr.X, cy)
	}

	if !c.collapsed {
		// Use the WIDGET's rect rather than the body's own frame: ValidateLayout owns
		// the body's frame, and a paint that re-derives it can disagree with the layout
		// whenever the row was resized after a layout pass.
		bf := graphics.NewRect(0, head, rect.Width, rect.Height-head)
		canvas.Save()
		canvas.Translate(0, head)
		local := graphics.NewRect(0, 0, bf.Width, bf.Height)
		canvas.ClipRect(local, pathop.Intersect, true)
		c.body.Paint(canvas, local)
		canvas.Restore()
	}

	canvas.Restore()
}

// trailingRect is where the right-hand text goes, or false when there is none.
func (c *Collapsible) trailingRect(width float32, font graphics.Font) (graphics.Rect, bool) {
	if c.trailing == "" {
		return graphics.Rect{}, false
	}
	t := graphics.NewText(c.trailing, &graphics.TextDecoration{Font: font})
	w := t.Width()
	return graphics.NewRect(width-w-10, 0, w, 40), true
}

func (c *Collapsible) font() graphics.Font {
	if f := theme.DefaultTheme.BaseFont; f != nil {
		return f
	}
	return graphics.SystemFont
}

// labelFont is the label's face: bold, because a transcript's KIND is what the eye scans for.
func labelFont(base graphics.Font) graphics.Font {
	fd := base.Descriptor()
	fd.Weight = weight.Bold
	return fd.Font()
}

var _ meui.EventListener = (*Collapsible)(nil)
var _ meui.HoverTarget = (*Collapsible)(nil)
