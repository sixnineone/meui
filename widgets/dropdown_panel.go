package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
)

// PickerPresentation selects WHERE a picker's panel appears. It is a behaviour axis, so it follows
// the SURFACE's size class rather than the OS name.
type PickerPresentation uint8

const (
	// PickerAuto anchors the panel under its field on a surface with room, and lifts it into a
	// bottom sheet on a phone — where a dropdown under a field is a pointer control, and a popup
	// calendar is out of thumb reach. This is the plan's own rule for the pickers.
	PickerAuto PickerPresentation = iota
	// PickerDropdown forces the anchored panel.
	PickerDropdown
	// PickerSheet forces the bottom sheet.
	PickerSheet
)

// sheetPanelPad is the breathing room a picker's sheet leaves above and below its body.
const sheetPanelPad = float32(12)

// sheetPanelWidth is how wide a picker's sheet is on a surface of the given width. It is the
// BottomSheet rule — full width on a compact surface, capped at BottomSheetMaxWidth with a
// BottomSheetSideMargin gap once the cap is in play — so a picker's sheet and a BottomSheet are the
// same shape on the same surface rather than two opinions about it.
func sheetPanelWidth(surface float32) float32 {
	w := surface
	if w > BottomSheetMaxWidth {
		w = BottomSheetMaxWidth
		if maxW := surface - 2*BottomSheetSideMargin; w > maxW {
			w = maxW
		}
	}
	if w < 0 {
		w = 0
	}
	return w
}

// dropdownPanel is the plumbing behind the date picker's calendar and the
// colour field's picker: a panel shown in the window's layer space. The layer
// covers the whole window (modal) so a click anywhere outside the panel closes
// it, and the panel is drawn at its own offset inside the layer.
//
// It is the mechanism ComboBox already uses, extracted so the two field widgets
// do not each carry a copy of it. The layer's Widget is the owning field, which
// is therefore the widget the LayerManager routes layer events to, in
// layer-local coordinates.
type dropdownPanel struct {
	layer   *meui.Layer
	offsetX float32 // panel top-left inside the layer (window) space
	offsetY float32
	width   float32
	height  float32
	// sheetMode marks a panel lifted to the BOTTOM of the window with a scrim behind it, rather
	// than anchored under its owner.
	sheetMode bool
}

// openPanel anchors a width x height panel just below the owner's bottom edge
// and paints it with paint, whose rect is the panel itself in panel-local
// coordinates (0,0 = panel top-left). Returns false when the widget is not in a
// window yet, so an unparented field simply does nothing instead of panicking.
func (d *dropdownPanel) openPanel(owner meui.Widget, width, height float32,
	paint func(canvas *graphics.Canvas, panel graphics.Rect)) bool {
	return d.open(owner, width, height, false, paint)
}

// openSheet places the panel at the BOTTOM of the window, centred, with a scrim over everything
// above it. The paint callback is unchanged: it is handed the panel's own rect either way, so a
// picker draws the same thing in both presentations.
func (d *dropdownPanel) openSheet(owner meui.Widget, width, height float32,
	paint func(canvas *graphics.Canvas, panel graphics.Rect)) bool {
	return d.open(owner, width, height, true, paint)
}

func (d *dropdownPanel) open(owner meui.Widget, width, height float32, sheet bool,
	paint func(canvas *graphics.Canvas, panel graphics.Rect)) bool {
	win := windowOf(owner)
	if win == nil || win.LayerManager == nil || width <= 0 || height <= 0 {
		return false
	}
	d.width, d.height, d.sheetMode = width, height, sheet

	// The panel's own position, in the window's content space. A dropdown hangs from the owner's
	// bottom edge — which is a point in the TREE, so AddLayerFor resolves it — and a sheet sits on
	// the window's bottom edge, centred.
	wf := win.FrameRect()
	useSheet := sheet && wf.Width > 0 && wf.Height > 0
	paintFn := func(canvas *graphics.Canvas, _ graphics.Rect) {
		if d.sheetMode && wf.Width > 0 && wf.Height > 0 {
			// The scrim is the modal backdrop the sheet would otherwise get from the window: this
			// panel is a layer, not a modal, so it draws its own.
			scrim := graphics.NewPaint()
			scrim.SetColor(graphics.Color(0xA0000000))
			canvas.DrawRect(graphics.NewRect(0, 0, wf.Width, wf.Height), scrim)
		}
		canvas.Save()
		canvas.Translate(d.offsetX, d.offsetY)
		paint(canvas, graphics.NewRect(0, 0, d.width, d.height))
		canvas.Restore()
	}
	var layer *meui.Layer
	if useSheet {
		d.offsetX = (wf.Width - width) / 2
		d.offsetY = wf.Height - height
		layer = win.AddLayerFor(owner, 0, 0, paintFn)
	} else {
		layer = win.AddLayerFor(owner, 0, owner.FrameRect().Height, paintFn)
		if layer != nil {
			// AddLayerFor resolved the owner's bottom edge into window space; that IS the panel's
			// position, and taking it from the layer is what keeps the tree arithmetic in one place.
			d.offsetX, d.offsetY = layer.X, layer.Y
		}
	}
	if layer == nil {
		return false
	}
	d.layer = layer
	// The layer covers the whole window so a click anywhere outside the panel closes it; the
	// panel keeps its offset inside it.
	layer.X, layer.Y = wf.X, wf.Y
	if wf.Width > 0 && wf.Height > 0 {
		layer.Width, layer.Height = wf.Width, wf.Height
	} else {
		layer.Width, layer.Height = width, height
	}
	layer.Widget = owner
	layer.Visible = true
	return true
}

// closePanel removes the layer.
func (d *dropdownPanel) closePanel(owner meui.Widget) {
	if d.layer == nil {
		return
	}
	if win := windowOf(owner); win != nil && win.LayerManager != nil {
		win.LayerManager.RemoveLayer(d.layer)
	}
	d.layer = nil
}

// isOpen reports whether the panel is on screen.
func (d *dropdownPanel) isOpen() bool { return d.layer != nil && d.layer.Visible }

// inside converts a layer-local point into panel-local coordinates and reports
// whether it is over the panel at all. Outside the panel the owner closes the
// panel, which is the click-anywhere-else dismissal.
func (d *dropdownPanel) inside(lx, ly float32) (float32, float32, bool) {
	if !d.isOpen() {
		return 0, 0, false
	}
	px, py := lx-d.offsetX, ly-d.offsetY
	if px < 0 || py < 0 || px > d.width || py > d.height {
		return 0, 0, false
	}
	return px, py, true
}

// windowOf walks the parent chain to the window (BaseWidget.Window).
func windowOf(w meui.Widget) *meui.Window {
	if w == nil {
		return nil
	}
	if bw, ok := w.(interface{ Window() *meui.Window }); ok {
		return bw.Window()
	}
	return nil
}
