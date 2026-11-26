package platform

import "github.com/sixnineone/mobilegles"

// Cursor is the hover-cursor vocabulary the toolkit speaks: a logical name the
// backend maps to a real system cursor (mobilegles does this through
// SDL_CreateSystemCursor). Widgets return one from CursorAt; returning nil
// leaves the cursor alone, and the window falls back to the default arrow.
//
// The names are the contract with the backend. mobilegles.SetCursorByName maps
// every name produced here (see TestCursorNamesAreMappable in mobilegles).
type Cursor struct {
	name string
}

func (c *Cursor) Name() string {
	if c == nil {
		return ""
	}
	return c.name
}

// Cursor constructors are functions (not variables) so meui call sites
// `platform.ResizeHorizontalCursor()` compile without modification.
//
// "Default arrow" is expressed by returning nil from CursorAt; ArrowCursor
// exists for a widget that must *stop* the hit-test walk and force the arrow
// over part of itself.
func ArrowCursor() *Cursor               { return &Cursor{name: "arrow"} }
func PointingCursor() *Cursor            { return &Cursor{name: "pointing"} }
func TextCursor() *Cursor                { return &Cursor{name: "text"} }
func MoveCursor() *Cursor                { return &Cursor{name: "move"} }
func CrosshairCursor() *Cursor           { return &Cursor{name: "crosshair"} }
func WaitCursor() *Cursor                { return &Cursor{name: "wait"} }
func NotAllowedCursor() *Cursor          { return &Cursor{name: "notAllowed"} }
func ResizeHorizontalCursor() *Cursor    { return &Cursor{name: "resizeH"} }
func ResizeVerticalCursor() *Cursor      { return &Cursor{name: "resizeV"} }
func ResizeLeftDiagonalCursor() *Cursor  { return &Cursor{name: "resizeLD"} }
func ResizeRightDiagonalCursor() *Cursor { return &Cursor{name: "resizeRD"} }

// CursorNames returns every logical cursor name the toolkit can produce.
func CursorNames() []string {
	return []string{
		"arrow", "pointing", "text", "move", "crosshair", "wait", "notAllowed",
		"resizeH", "resizeV", "resizeLD", "resizeRD",
	}
}

// KnownBackendCursorNames returns the names the backend maps to a system
// cursor. It is the same list; keeping it a separate call documents the
// contract the two sides must satisfy (see the cursor contract test).
func KnownBackendCursorNames() []string {
	return mobilegles.KnownCursorNames()
}
