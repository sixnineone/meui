package platform

import "github.com/sixnineone/meui/graphics"

// Drawable is anything a platform window can render: the toolkit's window and
// panel root implement it, and the window calls Paint once per frame after the
// background clear. It names only seam types, so the contract does not depend
// on any particular windowing backend.
type Drawable interface {
	Paint(canvas *graphics.Canvas, rect graphics.Rect)
}
