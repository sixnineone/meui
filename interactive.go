package meui

// ClickBehaviour is the press/click state machine behind every plain clickable
// surface: a Panel with `clickable: true` (QML's Rectangle) and a MouseArea both
// run it, so a click cannot mean two different things depending on which element
// carried it. The widgets that have their own semantics (Button's toggle, a tab's
// selection) keep their own handlers — this exists for the surface that has none.
type ClickBehaviour struct {
	OnClick func()
	OnPress func(pressed bool)
	pressed bool
}

// Pressed reports whether the pointer is down on this surface.
func (c *ClickBehaviour) Pressed() bool { return c.pressed }

// Down arms the click. It reports whether anything changed, so the caller knows
// to repaint. Only the primary button activates a surface.
func (c *ClickBehaviour) Down(enabled, inside bool, button int) bool {
	if !enabled || !inside || button != 0 || c.pressed {
		return false
	}
	c.pressed = true
	if c.OnPress != nil {
		c.OnPress(true)
	}
	return true
}

// Up ends the press, firing OnClick when the pointer is still inside the surface.
// It reports whether the visual state changed: a release outside still has to
// clear the pressed feedback even though no click fires.
func (c *ClickBehaviour) Up(enabled, inside bool, button int) bool {
	if !c.pressed {
		return false
	}
	c.pressed = false
	if c.OnPress != nil {
		c.OnPress(false)
	}
	if enabled && inside && button == 0 && c.OnClick != nil {
		c.OnClick()
	}
	return true
}

// Activate is the keyboard path for Enter/Space: a press and a release in one
// step, because a keyboard "click" has no visible down/up on a plain box (a
// Button animates its own press instead).
func (c *ClickBehaviour) Activate(enabled bool) bool {
	if !enabled || c.OnClick == nil {
		return false
	}
	c.OnClick()
	return true
}
