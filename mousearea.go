package meui

// MouseArea is QML's input surface, adapted to this toolkit's "wired from Go"
// model: it owns the pointer, the hover and press state, the cursor and the
// Enter/Space activation, and it reports clicks through a callback the app installs
// from Go by id. It paints nothing.
//
// Two ways to make a Rectangle interactive exist, on purpose:
//
//   - `clickable: true` on the Rectangle itself (or SetClickable) — the terse path
//     for a whole card that is the target;
//   - a MouseArea child — the Qt-shaped path for a target that is a REGION of a
//     larger surface (a row's link, a hero's overlay, a hit strip), and for
//     keeping the input declaration separate from the paint.
//
// Both run the same ClickBehaviour, so a click cannot mean two different things
// depending on which element carried it. The failure this avoids is the one the
// tree has already paid for twice: a second implementation of press handling that
// silently disagrees with the first.

import (
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
)

// MouseArea is a transparent input region. Size and position come from its parent
// (anchors.fill) or from explicit width/height.
type MouseArea struct {
	BaseWidget

	// HoverEnabled is QML's name for "track hover even without buttons". It is
	// accepted for scene readability; the toolkit always tracks hover, because a
	// position-based hover walk is cheaper than a second code path.
	HoverEnabled bool

	OnClicked         func()
	OnPressed         func()
	OnReleased        func()
	OnPositionChanged func(x, y float32)
	OnDoubleClicked   func()

	click       ClickBehaviour
	cursor      *platform.Cursor
	stateTarget Widget
	focusable   bool
}

var (
	_ EventListener  = (*MouseArea)(nil)
	_ FocusHandler   = (*MouseArea)(nil)
	_ CursorProvider = (*MouseArea)(nil)
)

// NewMouseArea returns an empty input region.
func NewMouseArea() *MouseArea {
	return &MouseArea{BaseWidget: *NewBaseWidget()}
}

// SetFillParent sizes the area to its parent — QML's anchors.fill: parent, without
// requiring the parent to run an anchor pass before the first frame.
func (m *MouseArea) SetFillParent() *MouseArea {
	if p := m.Parent(); p != nil {
		fr := p.FrameRect()
		m.SetFrameRect(graphics.NewRect(0, 0, fr.Width, fr.Height))
	}
	return m
}

// SetStateTarget points the area's hover/press state at another widget (a Panel),
// which is how a scene lights up the box a MouseArea sits on. Setting a target
// does not take over the target's own input; it only forwards state.
func (m *MouseArea) SetStateTarget(w Widget) *MouseArea {
	m.stateTarget = w
	return m
}

// SetFocusable makes the area a keyboard target: it joins the Tab order and
// Enter/Space activate it.
func (m *MouseArea) SetFocusable(b bool) *MouseArea {
	m.focusable = b
	if b && m.TabIndex() < 0 {
		m.SetTabIndex(0)
	}
	m.MarkForRedraw()
	return m
}

// Focusable reports whether the area takes the keyboard.
func (m *MouseArea) Focusable() bool {
	return m.focusable && m.TabIndex() >= 0 && m.Enabled() && m.Visible()
}

// SetCursor forces a cursor; nil falls back to the pointing hand.
func (m *MouseArea) SetCursor(c *platform.Cursor) *MouseArea {
	m.cursor = c
	return m
}

// CursorAt points at an interactive area.
func (m *MouseArea) CursorAt(x, y float32) *platform.Cursor {
	if !m.Enabled() {
		return nil
	}
	if m.cursor != nil {
		return m.cursor
	}
	return platform.PointingCursor()
}

// Paint draws nothing: a MouseArea is input, not ink.
func (m *MouseArea) Paint(canvas *graphics.Canvas, rect graphics.Rect) {}

// SetHoveredState forwards hover to the state target as well as tracking it here,
// so a box painted by the area responds while the pointer is over it.
func (m *MouseArea) SetHoveredState(v bool) {
	m.BaseWidget.SetHoveredState(v)
	m.forwardState()
}

// OnMouseDown arms the press.
func (m *MouseArea) OnMouseDown(x, y float32, button int) bool {
	if !m.ContainsPoint(x, y) {
		return false
	}
	if !m.click.Down(m.Enabled(), true, button) {
		return false
	}
	m.forwardState()
	if m.OnPressed != nil {
		m.OnPressed()
	}
	return true
}

// OnMouseUp fires the click when the release is still inside.
func (m *MouseArea) OnMouseUp(x, y float32, button int) bool {
	if !m.click.Up(m.Enabled(), m.ContainsPoint(x, y), button) {
		return false
	}
	m.forwardState()
	if m.OnReleased != nil {
		m.OnReleased()
	}
	if m.Enabled() && button == 0 && m.ContainsPoint(x, y) && m.OnClicked != nil {
		m.OnClicked()
	}
	return true
}

// OnMouseMove reports the pointer in the area's own coordinates.
func (m *MouseArea) OnMouseMove(x, y float32) {
	if m.OnPositionChanged != nil {
		m.OnPositionChanged(x, y)
	}
}

// OnMouseDoubleClick is dispatched by the window's click-count path when a widget
// implements it; kept here so a scene can wire a double-click without a new widget.
func (m *MouseArea) OnMouseDoubleClick(x, y float32, button int) bool {
	if m.Enabled() && button == 0 && m.ContainsPoint(x, y) && m.OnDoubleClicked != nil {
		m.OnDoubleClicked()
		return true
	}
	return false
}

// OnKeyDown activates a focused area with Enter or Space.
func (m *MouseArea) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if !m.Focusable() || !m.Focused() {
		return false
	}
	switch keyCode {
	case platform.KeyReturn, platform.KeyNumPadEnter, platform.KeySpace:
		if m.Enabled() && m.OnClicked != nil {
			m.OnClicked()
			return true
		}
	}
	return false
}

// SetPushed lets the window or a parent drive the visual press state (a drag that
// started elsewhere, a test).
func (m *MouseArea) SetPushed(v bool) {
	if m.click.pressed != v {
		m.click.pressed = v
		m.forwardState()
	}
}

// Pressed reports the pointer state.
func (m *MouseArea) Pressed() bool { return m.click.Pressed() }

// forwardState copies hover and press into a Panel target, which is how a box gets
// its visual response from an area it does not own.
func (m *MouseArea) forwardState() {
	p, ok := m.stateTarget.(*Panel)
	if !ok {
		return
	}
	p.SetHoveredState(m.Hovered())
	p.SetPushed(m.click.Pressed())
}
