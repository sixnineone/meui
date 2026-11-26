package widgets

import (
	"fmt"
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
	"os"
	"strings"
)

type ComboBox struct {
	meui.BaseWidget
	*theme.ComboBoxTheme
	Options           []string
	SelectedIndex     int
	Expanded          bool
	HoveredIndex      int
	OnSelected        func(int, string)
	dropdownLayer     *meui.Layer
	dropdownOffsetX   float32
	dropdownOffsetY   float32
	dropdownWidth     float32
	dropdownHeight    float32
	dropdownSVG       *meui.SVGDocument
	Editable          bool
	FilterText        string
	PlaceholderText   string
	MaxDropdownHeight float32
	dropdownScroll    float32
	optionTexts       []*graphics.Text
	optionTextsVer    uint64
}

func NewComboBox() *ComboBox {
	cb := &ComboBox{
		BaseWidget:    *meui.NewBaseWidget(),
		ComboBoxTheme: theme.DefaultComboBoxTheme,
		SelectedIndex: -1,
		HoveredIndex:  -1,
	}
	cb.SetTabIndex(0)
	// Load dropdown arrow SVG
	if doc, err := meui.LoadSVGByName("dropdown"); err == nil {
		cb.dropdownSVG = doc
	}
	return cb
}

func (c *ComboBox) SetOptions(options []string) *ComboBox {
	c.Options = options
	if c.SelectedIndex >= len(options) {
		c.SelectedIndex = len(options) - 1
	}
	// Like Qt Quick ComboBox: selecting a model auto-selects the first item
	// so the current text is visible instead of an empty field.
	if c.SelectedIndex < 0 && len(options) > 0 {
		c.SelectedIndex = 0
	}
	c.buildOptionTexts()
	c.MarkForRedraw()
	return c
}

// AddItem appends an option (like QComboBox.addItem).
func (c *ComboBox) AddItem(item string) *ComboBox {
	c.Options = append(c.Options, item)
	c.buildOptionTexts()
	c.MarkForRedraw()
	return c
}

// RemoveItem removes the option at index (like QComboBox.removeItem).
func (c *ComboBox) RemoveItem(index int) *ComboBox {
	if index < 0 || index >= len(c.Options) {
		return c
	}
	c.Options = append(c.Options[:index], c.Options[index+1:]...)
	if c.SelectedIndex >= len(c.Options) {
		c.SelectedIndex = len(c.Options) - 1
	}
	c.buildOptionTexts()
	c.MarkForRedraw()
	return c
}

// ClearItems removes all options and the selection (like QComboBox.clear).
func (c *ComboBox) ClearItems() *ComboBox {
	c.Options = nil
	c.SelectedIndex = -1
	c.HoveredIndex = -1
	c.optionTexts = nil
	c.MarkForRedraw()
	return c
}

// ItemCount returns the number of options.
func (c *ComboBox) ItemCount() int { return len(c.Options) }

// GetSelectedText returns the currently selected option, or "".
func (c *ComboBox) GetSelectedText() string {
	if c.SelectedIndex >= 0 && c.SelectedIndex < len(c.Options) {
		return c.Options[c.SelectedIndex]
	}
	return ""
}

// ClearSelection deselects (SelectedIndex = -1).
func (c *ComboBox) ClearSelection() *ComboBox {
	c.SelectedIndex = -1
	c.MarkForRedraw()
	return c
}

// SetEditable enables the editable + filter mode at runtime.
func (c *ComboBox) SetEditable(editable bool) *ComboBox {
	c.Editable = editable
	if !editable {
		c.FilterText = ""
	}
	c.MarkForRedraw()
	return c
}

// SetPlaceholder sets the text shown when nothing is selected (editable mode
// shows it as the filter hint).
func (c *ComboBox) SetPlaceholder(text string) *ComboBox {
	c.PlaceholderText = text
	c.MarkForRedraw()
	return c
}

// SetMaxDropdownHeight caps the dropdown popup height (0 = theme default).
func (c *ComboBox) SetMaxDropdownHeight(height float32) *ComboBox {
	c.MaxDropdownHeight = height
	return c
}

func (c *ComboBox) buildOptionTexts() {
	if len(c.Options) == 0 {
		c.optionTexts = nil
		return
	}
	font := c.ComboBoxTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	ink := c.ComboBoxTheme.TextInk
	c.optionTexts = make([]*graphics.Text, len(c.Options))
	for i, opt := range c.Options {
		if opt != "" {
			c.optionTexts[i] = graphics.NewText(opt, &graphics.TextDecoration{Font: font, OnBackgroundInk: ink})
		}
	}
}

func (c *ComboBox) SetSelectedIndex(index int) *ComboBox {
	if index >= -1 && index < len(c.Options) {
		c.SelectedIndex = index
		c.MarkForRedraw()
	}
	return c
}

func (c *ComboBox) SetOnSelected(handler func(int, string)) *ComboBox {
	c.OnSelected = handler
	return c
}

func (c *ComboBox) displayOptions() []string {
	if !c.Editable || c.FilterText == "" {
		return c.Options
	}
	var filtered []string
	for _, opt := range c.Options {
		if len(opt) >= len(c.FilterText) &&
			strings.Contains(strings.ToLower(opt), strings.ToLower(c.FilterText)) {
			filtered = append(filtered, opt)
		}
	}
	return filtered
}

func (c *ComboBox) displayIndexToOption(idx int) int {
	if !c.Editable || c.FilterText == "" {
		return idx
	}
	count := 0
	for i, opt := range c.Options {
		if len(opt) >= len(c.FilterText) &&
			strings.Contains(strings.ToLower(opt), strings.ToLower(c.FilterText)) {
			if count == idx {
				return i
			}
			count++
		}
	}
	return -1
}

func (c *ComboBox) findWindow() *meui.Window {
	for parent := c.Parent(); parent != nil; {
		if w, ok := parent.(*meui.Window); ok {
			return w
		}
		if p, ok := parent.(interface{ Parent() meui.Widget }); ok {
			parent = p.Parent()
		} else {
			break
		}
	}
	return nil
}

func (c *ComboBox) SetExpanded(expanded bool) {
	if c.Expanded == expanded {
		return
	}
	c.Expanded = expanded

	if expanded {
		c.tryCreateDropdownLayer()
	} else {
		c.removeDropdownLayer()
	}
	c.MarkForRedraw()
}

// OnClosed is how the window tells a list that its layer is gone: Escape closes the topmost layer, and
// the combo has to forget it is expanded — otherwise it reopens on the next click believing the list is
// still up, or paints one that does not exist.
func (c *ComboBox) OnClosed() {
	c.SetExpanded(false)
}

func (c *ComboBox) tryCreateDropdownLayer() {
	if !c.Expanded || len(c.Options) == 0 {
		return
	}
	if c.dropdownLayer != nil {
		return
	}
	win := c.findWindow()
	if win == nil {
		return
	}
	c.createDropdownLayer()
}

func (c *ComboBox) createDropdownLayer() {
	win := c.findWindow()
	if win == nil {
		return
	}

	frame := c.FrameRect()
	optionHeight := c.ComboBoxTheme.OptionHeight
	if optionHeight <= 0 {
		optionHeight = 30
	}
	opts := c.displayOptions()
	totalHeight := optionHeight * float32(len(opts))
	maxH := c.MaxDropdownHeight
	if maxH <= 0 {
		maxH = 200
	}
	dropdownHeight := totalHeight
	if dropdownHeight > maxH {
		dropdownHeight = maxH
	}
	if c.dropdownScroll > totalHeight-dropdownHeight {
		c.dropdownScroll = totalHeight - dropdownHeight
	}
	if c.dropdownScroll < 0 {
		c.dropdownScroll = 0
	}

	// The dropdown hangs off the combo's bottom-left, expressed in the combo's
	// own coordinates. AddLayerFor places it in layer space — it knows whether
	// the combo lives in the content tree or in the window chrome — so no bar
	// arithmetic happens here. The layer's position *is* the list's top-left,
	// which is also what the modal overlay and the hit test need.
	c.dropdownLayer = win.AddLayerFor(c, 0, frame.Height, func(canvas *graphics.Canvas, rect graphics.Rect) {
		c.drawDropdownOnLayer(canvas, rect)
	})
	if c.dropdownLayer == nil {
		return
	}
	c.dropdownLayer.Widget = c
	c.dropdownOffsetX, c.dropdownOffsetY = c.dropdownLayer.X, c.dropdownLayer.Y
	c.dropdownWidth = frame.Width
	c.dropdownHeight = dropdownHeight

	// Modal layer: cover the whole window so a click anywhere outside the
	// option list closes the dropdown. The list keeps its own offset inside it.
	if winFrame := win.FrameRect(); winFrame.Width > 0 && winFrame.Height > 0 {
		c.dropdownLayer.X = winFrame.X
		c.dropdownLayer.Y = winFrame.Y
		c.dropdownLayer.Width = winFrame.Width
		c.dropdownLayer.Height = winFrame.Height
	} else {
		c.dropdownLayer.Width = frame.Width
		c.dropdownLayer.Height = dropdownHeight
	}
	c.dropdownLayer.Visible = true
}

// hasFullWindowLayer reports whether the dropdown layer covers the
// whole window (modal mode) versus only the option-list rect (legacy).
func (c *ComboBox) hasFullWindowLayer() bool {
	if c.dropdownLayer == nil || c.dropdownWidth <= 0 {
		return false
	}
	// In modal mode the layer covers the window, not just the option list.
	return c.dropdownLayer.Width != c.dropdownWidth || c.dropdownLayer.Height != c.dropdownHeight
}

func (c *ComboBox) removeDropdownLayer() {
	if c.dropdownLayer != nil {
		win := c.findWindow()
		if win != nil {
			win.LayerManager.RemoveLayer(c.dropdownLayer)
		}
		c.dropdownLayer = nil
	}
	c.HoveredIndex = -1
}

func (c *ComboBox) drawDropdownOnLayer(canvas *graphics.Canvas, rect graphics.Rect) {
	if os.Getenv("MEUI_COMBO_DEBUG") != "" {
		fmt.Printf("[comboDD] paint full=%v off=(%.0f,%.0f) ddWH=(%.0f,%.0f) rect=%v layerRect=(%v,%v %vx%v)\n",
			c.hasFullWindowLayer(), c.dropdownOffsetX, c.dropdownOffsetY, c.dropdownWidth, c.dropdownHeight,
			rect, c.dropdownLayer.X, c.dropdownLayer.Y, c.dropdownLayer.Width, c.dropdownLayer.Height)
	}
	optionHeight := c.ComboBoxTheme.OptionHeight
	if optionHeight <= 0 {
		optionHeight = 30
	}
	font := c.ComboBoxTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}

	// Modal full-window layer: the option list is drawn at its offset
	// inside the window-sized layer.
	if c.hasFullWindowLayer() {
		canvas.Save()
		canvas.Translate(c.dropdownOffsetX, c.dropdownOffsetY)
		defer canvas.Restore()
		rect = graphics.NewRect(0, 0, c.dropdownWidth, c.dropdownHeight)
	}

	// Clip to dropdown bounds
	canvas.Save()
	canvas.ClipRect(rect, pathop.Intersect, false)
	ddR := theme.Shape().Menu

	// Draw dropdown background
	bgPaint := graphics.NewPaint()
	bgPaint.SetColor(c.ComboBoxTheme.BackgroundInk)
	canvas.DrawRoundedRect(rect, ddR, ddR, bgPaint)

	// Draw options
	textPaint := graphics.NewPaint()
	textPaint.SetColor(c.ComboBoxTheme.TextInk)
	ascent := font.Baseline()
	fontHeight := font.LineHeight()
	opts := c.displayOptions()
	visibleStart := int(c.dropdownScroll / optionHeight)
	visibleCount := int(rect.Height/optionHeight) + 2

	for i := visibleStart; i < len(opts) && i < visibleStart+visibleCount; i++ {
		opt := opts[i]
		yOff := float32(i)*optionHeight - c.dropdownScroll
		optRect := graphics.NewRect(rect.X, rect.Y+yOff, rect.Width, optionHeight)

		// Draw hover highlight
		if i == c.HoveredIndex {
			hoverPaint := graphics.NewPaint()
			hoverPaint.SetColor(c.ComboBoxTheme.HoverInk)
			canvas.DrawRect(optRect, hoverPaint)
		}

		// Draw selected highlight
		realIdx := c.displayIndexToOption(i)
		if realIdx == c.SelectedIndex {
			selPaint := graphics.NewPaint()
			selPaint.SetColor(c.ComboBoxTheme.SelectedInk)
			canvas.DrawRect(optRect, selPaint)
		}

		// Draw option text
		textY := optRect.Y + (optionHeight-fontHeight)/2 + ascent
		isFiltered := c.Editable && c.FilterText != ""
		ver := theme.ThemeVersion()
		if c.optionTextsVer != ver {
			c.buildOptionTexts()
			c.optionTextsVer = ver
		}
		if !isFiltered && i < len(c.optionTexts) && c.optionTexts[i] != nil {
			c.optionTexts[i].Draw(canvas, rect.X+8, textY)
		} else {
			ot := graphics.NewText(opt, &graphics.TextDecoration{Font: font, OnBackgroundInk: textPaint.Color()})
			ot.Draw(canvas, rect.X+8, textY)
		}
	}

	canvas.Restore()

	// Draw dropdown border (on top of clipped content)
	borderPaint := graphics.NewPaint()
	borderPaint.SetStyle(paintstyle.Stroke)
	borderPaint.SetStrokeWidth(theme.BorderStrokeWidth)
	borderPaint.SetColor(c.ComboBoxTheme.EdgeInk)
	canvas.DrawRoundedRect(rect, ddR, ddR, borderPaint)

	// Draw scrollbar if content overflows
	opts = c.displayOptions()
	totalH := optionHeight * float32(len(opts))
	if totalH > rect.Height {
		sbW := float32(8)
		sbX := rect.X + rect.Width - sbW
		sbH := rect.Height

		// Scrollbar track
		trackPaint := graphics.NewPaint()
		trackPaint.SetColor(c.ComboBoxTheme.EdgeInk)
		canvas.DrawRect(graphics.NewRect(sbX, rect.Y, sbW, sbH), trackPaint)

		// Scrollbar thumb
		thumbH := sbH * (rect.Height / totalH)
		thumbY := rect.Y + (c.dropdownScroll/(totalH-rect.Height))*(sbH-thumbH)
		thumbPaint := graphics.NewPaint()
		thumbPaint.SetColor(c.ComboBoxTheme.SelectedInk)
		canvas.DrawRect(graphics.NewRect(sbX+1, thumbY, sbW-2, thumbH), thumbPaint)
	}
}

func (c *ComboBox) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !c.Visible() {
		return
	}

	// Try to create dropdown layer now if expanded but layer not yet created
	if c.Expanded && c.dropdownLayer == nil {
		c.tryCreateDropdownLayer()
	}

	canvas.Save()

	// Draw background
	bgPaint := graphics.NewPaint()
	if !c.Enabled() {
		bgPaint.SetColor(theme.DefaultTheme.DisabledBg)
	} else {
		bgPaint.SetColor(c.ComboBoxTheme.BackgroundInk)
	}
	canvas.DrawRoundedRect(rect, c.ComboBoxTheme.CornerRadius, c.ComboBoxTheme.CornerRadius, bgPaint)

	// Draw border
	borderPaint := graphics.NewPaint()
	borderPaint.SetStyle(paintstyle.Stroke)
	borderPaint.SetStrokeWidth(theme.BorderStrokeWidth)
	if !c.Enabled() {
		borderPaint.SetColor(theme.DefaultTheme.DisabledBorder)
	} else {
		borderPaint.SetColor(c.ComboBoxTheme.EdgeInk)
	}
	canvas.DrawRoundedRect(rect, c.ComboBoxTheme.CornerRadius, c.ComboBoxTheme.CornerRadius, borderPaint)
	// Fluent emphasises the bottom edge; other platforms leave it zero.
	if c.Enabled() && c.ComboBoxTheme.BottomEdgeWidth > 0 {
		bw := c.ComboBoxTheme.BottomEdgeWidth
		bc := c.ComboBoxTheme.BottomEdgeInk
		if c.BaseWidget.Focused() && c.ComboBoxTheme.BottomFocusWidth > 0 {
			bw = c.ComboBoxTheme.BottomFocusWidth
			bc = theme.DefaultTheme.BorderFocused
		}
		if bc != 0 {
			bp := graphics.NewPaint()
			bp.SetStyle(paintstyle.Stroke)
			bp.SetStrokeWidth(bw)
			bp.SetColor(bc)
			y := rect.Y + rect.Height - bw/2
			canvas.DrawLine(rect.X, y, rect.X+rect.Width, y, bp)
		}
	}

	// Get font
	font := c.ComboBoxTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	ascent := font.Baseline()
	fontHeight := font.LineHeight()

	// Vertical separator on the left of the chevron — same divider the
	// SpinBox draws between its field and its arrow buttons.
	arrowSize := float32(12)
	sepPaint := graphics.NewPaint()
	sepPaint.SetStyle(paintstyle.Stroke)
	sepPaint.SetStrokeWidth(theme.BorderStrokeWidth)
	if !c.Enabled() {
		sepPaint.SetColor(theme.DefaultTheme.DisabledBorder)
	} else {
		sepPaint.SetColor(c.ComboBoxTheme.EdgeInk)
	}
	sepX := rect.X + rect.Width - arrowSize - 16
	canvas.DrawLine(sepX, rect.Y+3, sepX, rect.Y+rect.Height-3, sepPaint)

	// Draw dropdown arrow
	arrowX := rect.X + rect.Width - arrowSize - 8
	arrowY := rect.Y + (rect.Height-arrowSize)/2
	if c.dropdownSVG != nil {
		meui.DrawSVG(canvas, c.dropdownSVG, arrowX, arrowY, arrowSize, arrowSize)
	} else {
		arrowPaint := graphics.NewPaint()
		arrowCol := c.ComboBoxTheme.ArrowInk
		if !c.Enabled() {
			arrowCol = 0xFF555555
		}
		if c.ComboBoxTheme.ArrowStyle == 1 {
			// Platform chevron (Fluent/macOS/GNOME/KDE).
			arrowPaint.SetStyle(paintstyle.Stroke)
			arrowPaint.SetStrokeWidth(1.6)
			arrowPaint.SetColor(arrowCol)
			cx := arrowX + arrowSize/2
			cy := arrowY + arrowSize/2
			canvas.DrawLine(cx-4, cy-1.5, cx, cy+2, arrowPaint)
			canvas.DrawLine(cx, cy+2, cx+4, cy-1.5, arrowPaint)
		} else {
			arrowPaint.SetColor(arrowCol)
			font := c.ComboBoxTheme.Font
			if font == nil {
				font = theme.DefaultTheme.BaseFont
				if font == nil {
					font = graphics.SystemFont
				}
			}
			ascent := font.Baseline()
			at := graphics.NewText("▼", &graphics.TextDecoration{Font: font, OnBackgroundInk: arrowPaint.Color()})
			at.Draw(canvas, arrowX, arrowY+arrowSize/2+ascent/2)
		}
	}

	// Draw selected text
	displayText := ""
	if c.Editable {
		displayText = c.FilterText
	} else if c.SelectedIndex >= 0 && c.SelectedIndex < len(c.Options) {
		displayText = c.Options[c.SelectedIndex]
	}
	// Fall back to the placeholder when nothing is selected
	showingPlaceholder := false
	if displayText == "" && c.PlaceholderText != "" {
		displayText = c.PlaceholderText
		showingPlaceholder = true
	}
	if displayText != "" {
		textX := rect.X + 8
		baseline := rect.Y + (rect.Height-fontHeight)/2 + ascent
		textPaint := graphics.NewPaint()
		if !c.Enabled() {
			textPaint.SetColor(c.ComboBoxTheme.TextDisabled)
		} else if showingPlaceholder {
			textPaint.SetColor(graphics.Color(0xFF808080))
		} else {
			textPaint.SetColor(c.ComboBoxTheme.TextInk)
		}
		ver := theme.ThemeVersion()
		if c.optionTextsVer != ver {
			c.buildOptionTexts()
			c.optionTextsVer = ver
		}
		if c.SelectedIndex >= 0 && c.SelectedIndex < len(c.optionTexts) && c.optionTexts[c.SelectedIndex] != nil && !c.Editable {
			c.optionTexts[c.SelectedIndex].Draw(canvas, textX, baseline)
		} else {
			dt := graphics.NewText(displayText, &graphics.TextDecoration{Font: font, OnBackgroundInk: textPaint.Color()})
			dt.Draw(canvas, textX, baseline)
		}
	}

	// Focus indicator — platform style (ring/inset/two-tone/underline).
	if c.BaseWidget.Focused() {
		drawFocusIndicator(canvas, rect, theme.Shape().ComboBox)
	}

	canvas.Restore()
}

func (c *ComboBox) OnMouseDown(x, y float32, button int) bool {
	if !c.Enabled() {
		return false
	}

	frame := c.FrameRect()
	if x >= 0 && x <= frame.Width && y >= 0 && y <= frame.Height {
		if c.Expanded {
			c.SetExpanded(false)
		} else {
			c.SetExpanded(true)
		}
		return true
	}

	return false
}

func (c *ComboBox) HandleMouseDown(x, y float32, button int) bool {
	if !c.Enabled() {
		return false
	}

	// Check if click is on the dropdown layer. The layer is modal
	// (full-window): a click inside the option list selects; a click
	// anywhere else just closes.
	if c.Expanded && c.dropdownLayer != nil {
		_, _, inOptions := c.dropdownLocal(x, y)
		if inOptions {
			idx, ok := c.optionIndexAt(x, y)
			if ok {
				realIdx := c.displayIndexToOption(idx)
				c.SelectedIndex = realIdx
				if c.Editable {
					c.FilterText = c.Options[realIdx]
				}
				c.SetExpanded(false)
				if c.OnSelected != nil {
					c.OnSelected(realIdx, c.Options[realIdx])
				}
			} else {
				c.SetExpanded(false)
			}
			return true
		}
		// Click on the modal layer but outside the option list: close.
		c.SetExpanded(false)
		return true
	}

	// Check if click is on the main combo box
	frame := c.FrameRect()
	if x >= 0 && x <= frame.Width && y >= 0 && y <= frame.Height {
		if c.Expanded {
			c.SetExpanded(false)
		} else {
			c.SetExpanded(true)
		}
		return true
	}

	// Click outside - close dropdown
	if c.Expanded {
		c.SetExpanded(false)
		return true
	}

	return false
}

// dropdownLocal maps a pointer position onto the option list. In modal mode the
// layer covers the whole window, so the incoming point is in content
// coordinates and the dropdown's offset must be subtracted; in the legacy mode
// the layer *is* the list, so the point is already local.
//
// The hover and the click used to disagree about this — the hover skipped the
// offset — so a dropdown far from the window origin (the body font combo, ~90px
// down) highlighted, and then selected, an item a few rows below the pointer,
// while the title-bar combo near the top looked fine.
func (c *ComboBox) dropdownLocal(x, y float32) (float32, float32, bool) {
	if c.hasFullWindowLayer() {
		lx, ly := x-c.dropdownOffsetX, y-c.dropdownOffsetY
		return lx, ly, lx >= 0 && lx <= c.dropdownWidth && ly >= 0 && ly <= c.dropdownHeight
	}
	return x, y, x >= 0 && x <= c.dropdownWidth && y >= 0 && y <= c.dropdownHeight
}

// optionIndexAt returns the option under a pointer position and whether the
// point is inside the option list at all.
func (c *ComboBox) optionIndexAt(x, y float32) (int, bool) {
	_, ly, in := c.dropdownLocal(x, y)
	if !in {
		return -1, false
	}
	optionHeight := c.ComboBoxTheme.OptionHeight
	if optionHeight <= 0 {
		optionHeight = 30
	}
	idx := int((ly + c.dropdownScroll) / optionHeight)
	if idx < 0 || idx >= len(c.displayOptions()) {
		return -1, false
	}
	return idx, true
}

func (c *ComboBox) HandleMouseMove(x, y float32) bool {
	if !c.Enabled() {
		return false
	}

	if c.Expanded && c.dropdownLayer != nil {
		dl := c.dropdownLayer
		if x >= 0 && x <= dl.Width && y >= 0 && y <= dl.Height {
			if idx, ok := c.optionIndexAt(x, y); ok {
				if c.HoveredIndex != idx {
					c.HoveredIndex = idx
					c.MarkForRedraw()
				}
			} else if c.HoveredIndex != -1 {
				c.HoveredIndex = -1
				c.MarkForRedraw()
			}
			return true
		} else {
			if c.HoveredIndex != -1 {
				c.HoveredIndex = -1
				c.MarkForRedraw()
			}
		}
	}

	return false
}

func (c *ComboBox) OnMouseMove(x, y float32) {
	c.HandleMouseMove(x, y)
}

// OnKeyRepeat is a key the user is HOLDING — Backspace over a word, an arrow along a row, a scrub.
// The window classifies an OS auto-repeat as a repeat rather than a fresh press (a widget without
// this method is asked once, deliberately), so this is what makes a held key work here.
func (c *ComboBox) OnKeyRepeat(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	return c.OnKeyDown(keyCode, mod)
}

func (c *ComboBox) OnKeyDown(keyCode platform.KeyCode, mod platform.Modifiers) bool {
	if !c.Enabled() || len(c.Options) == 0 {
		return false
	}
	opts := c.displayOptions()
	switch keyCode {
	case platform.KeyDown:
		if !c.Expanded {
			c.SetExpanded(true)
		} else if len(opts) > 0 {
			next := c.HoveredIndex + 1
			if next >= len(opts) {
				next = 0
			}
			c.HoveredIndex = next
			if !c.Editable {
				c.SelectedIndex = c.displayIndexToOption(next)
				if c.OnSelected != nil {
					c.OnSelected(c.SelectedIndex, c.Options[c.SelectedIndex])
				}
			}
		}
		c.MarkForRedraw()
		return true
	case platform.KeyUp:
		if !c.Expanded {
			c.SetExpanded(true)
			c.HoveredIndex = c.displayIndexToOption(c.SelectedIndex)
			if c.HoveredIndex < 0 {
				c.HoveredIndex = 0
			}
		} else if len(opts) > 0 {
			prev := c.HoveredIndex - 1
			if prev < 0 {
				prev = len(opts) - 1
			}
			c.HoveredIndex = prev
			if !c.Editable {
				c.SelectedIndex = c.displayIndexToOption(prev)
				if c.OnSelected != nil {
					c.OnSelected(c.SelectedIndex, c.Options[c.SelectedIndex])
				}
			}
		}
		c.MarkForRedraw()
		return true
	case platform.KeyReturn, platform.KeyNumPadEnter:
		if c.Expanded && c.HoveredIndex >= 0 && c.HoveredIndex < len(opts) {
			realIdx := c.displayIndexToOption(c.HoveredIndex)
			c.SelectedIndex = realIdx
			if c.Editable {
				c.FilterText = c.Options[realIdx]
			}
			c.SetExpanded(false)
			if c.OnSelected != nil {
				c.OnSelected(realIdx, c.Options[realIdx])
			}
		} else if !c.Editable && c.SelectedIndex >= 0 {
			if c.OnSelected != nil {
				c.OnSelected(c.SelectedIndex, c.Options[c.SelectedIndex])
			}
		}
		c.MarkForRedraw()
		return true
	case platform.KeyEscape:
		if c.Expanded {
			c.SetExpanded(false)
			c.MarkForRedraw()
			return true
		}
	case platform.KeyBackspace:
		if c.Editable && c.Expanded && len(c.FilterText) > 0 {
			c.FilterText = c.FilterText[:len(c.FilterText)-1]
			c.HoveredIndex = 0
			c.rebuildDropdown()
			c.MarkForRedraw()
			return true
		}
	}
	return false
}

func (c *ComboBox) OnCharacter(ch rune) {
	if !c.Enabled() || !c.Editable {
		return
	}
	// Start filtering
	c.FilterText += string(ch)
	c.HoveredIndex = 0
	if !c.Expanded {
		c.SetExpanded(true)
	} else {
		c.rebuildDropdown()
	}
	c.MarkForRedraw()
}

func (c *ComboBox) rebuildDropdown() {
	if c.dropdownLayer != nil {
		win := c.findWindow()
		if win != nil {
			win.LayerManager.RemoveLayer(c.dropdownLayer)
		}
		c.dropdownLayer = nil
	}
	c.tryCreateDropdownLayer()
}

func (c *ComboBox) OnMouseWheel(x, y float32, dx, dy float32) bool {
	if !c.Expanded || c.dropdownLayer == nil || dy == 0 {
		return false
	}
	dl := c.dropdownLayer
	optionHeight := c.ComboBoxTheme.OptionHeight
	if optionHeight <= 0 {
		optionHeight = 30
	}
	opts := c.displayOptions()
	totalH := optionHeight * float32(len(opts))
	maxScroll := totalH - dl.Height
	if maxScroll <= 0 {
		return false
	}
	c.dropdownScroll -= dy * optionHeight * 2
	if c.dropdownScroll < 0 {
		c.dropdownScroll = 0
	}
	if c.dropdownScroll > maxScroll {
		c.dropdownScroll = maxScroll
	}
	c.MarkForRedraw()
	return true
}

func (c *ComboBox) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "currentIndex":
		if f, ok := value.(float64); ok {
			c.SetSelectedIndex(int(f))
		}
	case "expanded":
		if b, ok := value.(bool); ok {
			c.SetExpanded(b)
		}
	case "editable":
		if b, ok := value.(bool); ok {
			c.Editable = b
		}
	}
}

func (c *ComboBox) PreferredDimensions(target interface{}, width, height float32) (float32, float32) {
	h := c.ComboBoxTheme.OptionHeight
	if h <= 0 {
		h = 30
	}
	return 100, h
}

func (c *ComboBox) Focusable() bool {
	return c.Enabled()
}

func (c *ComboBox) OnFocus() {
	c.BaseWidget.OnFocus()
}

func (c *ComboBox) OnBlur() {
	c.BaseWidget.OnBlur()
}
