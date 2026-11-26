package widgets

import (
	"fmt"
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/theme"
)

type ToolSet struct {
	Widgets      []meui.Widget
	Label        string
	Draggable    bool
	HandleRect   graphics.Rect
	BoundsRect   graphics.Rect
	labelText    *graphics.Text
	labelTextVer uint64
}

type ToolButton struct {
	Text      string
	Icon      string
	Enabled   bool
	Toggle    bool
	Toggled   bool
	OnClicked func()
}

type ToolBar struct {
	meui.BaseWidget
	*theme.ToolBarTheme
	toolsets   []ToolSet
	spacing    float32
	padding    float32
	wrap       bool
	lineBreaks []int

	toggleWidgets map[meui.Widget]bool
	clickHandlers map[meui.Widget]func()

	dragging   bool
	dragSetIdx int
	dragStartX float32
	dragOffX   float32
	dragOffY   float32
}

const (
	toolbarBtnSize   float32 = 28
	toolbarSepW      float32 = 8
	toolbarHandleW   float32 = 10
	toolbarHandleGap float32 = 5
)

func NewToolBar() *ToolBar {
	return &ToolBar{
		BaseWidget:    *meui.NewBaseWidget(),
		ToolBarTheme:  theme.DefaultToolBarTheme,
		spacing:       2,
		padding:       4,
		toggleWidgets: make(map[meui.Widget]bool),
		clickHandlers: make(map[meui.Widget]func()),
		dragSetIdx:    -1,
	}
}

func (t *ToolBar) defaultSet() *ToolSet {
	if len(t.toolsets) == 0 {
		t.toolsets = []ToolSet{{}}
	}
	return &t.toolsets[0]
}

func (t *ToolBar) AddButton(text, icon string) *ToolBar {
	btn := NewButton()
	btn.SetText(text)
	if icon != "" {
		btn.SetIcon(icon)
	}
	btn.SetWidth(toolbarBtnSize)
	btn.SetHeight(toolbarBtnSize)
	ds := t.defaultSet()
	ds.Widgets = append(ds.Widgets, btn)
	t.MarkForRedraw()
	return t
}

func (t *ToolBar) AddSeparator() *ToolBar {
	ds := t.defaultSet()
	ds.Widgets = append(ds.Widgets, nil)
	t.MarkForRedraw()
	return t
}

func (t *ToolBar) AddToggle(text, icon string) *ToolBar {
	btn := NewButton()
	btn.SetText(text)
	if icon != "" {
		btn.SetIcon(icon)
	}
	btn.SetWidth(toolbarBtnSize)
	btn.SetHeight(toolbarBtnSize)

	btn.OnClicked = func() {
		btn.SetSelected(!btn.Selected)
		if h := t.clickHandlers[btn]; h != nil {
			h()
		}
	}
	t.toggleWidgets[btn] = true

	ds := t.defaultSet()
	ds.Widgets = append(ds.Widgets, btn)
	t.MarkForRedraw()
	return t
}

func (t *ToolBar) AddWidget(w meui.Widget) *ToolBar {
	ds := t.defaultSet()
	ds.Widgets = append(ds.Widgets, w)
	t.MarkForRedraw()
	return t
}

func (t *ToolBar) buildToolsetTexts() {
	font := t.getFont()
	for i := range t.toolsets {
		ts := &t.toolsets[i]
		if ts.Label != "" {
			ts.labelText = graphics.NewText(ts.Label, &graphics.TextDecoration{Font: font, OnBackgroundInk: t.ToolBarTheme.TextInk})
		}
	}
}

func (t *ToolBar) SetToolsets(toolsets []ToolSet) {
	t.toolsets = toolsets
	t.buildToolsetTexts()
	t.MarkForRedraw()
}

func (t *ToolBar) SetWrap(wrap bool) {
	t.wrap = wrap
	t.MarkForRedraw()
}

func (t *ToolBar) SetButtonEnabled(index int, enabled bool) {
	w := t.widgetAtFlatIndex(index)
	if w == nil {
		return
	}
	if btn, ok := w.(*Button); ok {
		btn.SetEnabled(enabled)
	}
	t.MarkForRedraw()
}

func (t *ToolBar) SetButtonToggled(index int, toggled bool) {
	w := t.widgetAtFlatIndex(index)
	if w == nil {
		return
	}
	if btn, ok := w.(*Button); ok {
		btn.SetSelected(toggled)
		t.MarkForRedraw()
	}
}

func (t *ToolBar) SetOnClicked(index int, handler func()) {
	w := t.widgetAtFlatIndex(index)
	if w == nil {
		return
	}
	if t.toggleWidgets[w] {
		t.clickHandlers[w] = handler
	} else if btn, ok := w.(*Button); ok {
		btn.OnClicked = handler
	}
}

func (t *ToolBar) SetSpacing(s float32) {
	t.spacing = s
	t.MarkForRedraw()
}

func (t *ToolBar) widgetAtFlatIndex(flatIdx int) meui.Widget {
	remaining := flatIdx
	for setIdx := range t.toolsets {
		ws := t.toolsets[setIdx].Widgets
		if remaining < len(ws) {
			return ws[remaining]
		}
		remaining -= len(ws)
	}
	return nil
}

func (t *ToolBar) getFont() graphics.Font {
	font := t.ToolBarTheme.Font
	if font == nil {
		font = theme.DefaultTheme.BaseFont
		if font == nil {
			font = graphics.SystemFont
		}
	}
	return font
}

func (t *ToolBar) lineH() float32 {
	return toolbarBtnSize + t.padding*2
}

func (t *ToolBar) computeLayout(availW float32) ([]int, int) {
	var breaks []int
	if len(t.toolsets) == 0 {
		return []int{0}, 1
	}
	breaks = append(breaks, 0)
	x := float32(0)
	lineCount := 1
	font := t.getFont()

	for setIdx := range t.toolsets {
		ts := &t.toolsets[setIdx]
		nb := len(ts.Widgets)
		if nb == 0 {
			continue
		}

		setW := toolbarHandleW + toolbarHandleGap
		if ts.Label != "" {
			setW += font.SimpleWidth(ts.Label) + t.spacing*2
		}
		for _, w := range ts.Widgets {
			if w == nil {
				setW += toolbarSepW
			} else {
				setW += toolbarBtnSize + t.spacing
			}
		}
		setW += toolbarSepW

		if t.wrap && x+setW > availW && x > 0 {
			lineCount++
			breaks = append(breaks, setIdx)
			x = 0
		}

		x += toolbarHandleW + toolbarHandleGap
		if ts.Label != "" {
			x += font.SimpleWidth(ts.Label) + t.spacing*2
		}
		for _, w := range ts.Widgets {
			if w == nil {
				x += toolbarSepW
			} else {
				x += toolbarBtnSize + t.spacing
			}
		}
		x += toolbarSepW
	}

	return breaks, lineCount
}

func (t *ToolBar) contentHeight() float32 {
	if !t.wrap || len(t.toolsets) == 0 {
		return t.FrameRect().Height
	}
	frame := t.FrameRect()
	_, lc := t.computeLayout(frame.Width - t.padding*2)
	return float32(lc) * t.lineH()
}

func (t *ToolBar) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !t.Visible() {
		return
	}

	if t.wrap {
		ch := t.contentHeight()
		if ch > 0 && ch != rect.Height {
			t.SetHeightDirect(ch)
			rect.Height = ch
			// The height self-correction must re-run the parent layout,
			// or a stale (e.g. wrap-inflated at degenerate first-paint
			// width) frame sticks forever.
			for p := t.Parent(); p != nil; p = p.Parent() {
				if nl, ok := p.(interface{ SetNeedsLayout() }); ok {
					nl.SetNeedsLayout()
				}
			}
			t.MarkForRedraw()
		}
	}

	canvas.Save()

	disabled := !t.Enabled()
	drawH := rect.Height
	font := t.getFont()
	availW := rect.Width - t.padding*2
	lh := t.lineH()

	bgPaint := graphics.NewPaint()
	if disabled {
		bgPaint.SetColor(theme.DefaultTheme.DisabledBg)
	} else {
		bgPaint.SetColor(t.ToolBarTheme.BackgroundInk)
	}
	canvas.DrawRect(graphics.NewRect(rect.X, rect.Y, rect.Width, drawH), bgPaint)

	bottomPaint := graphics.NewPaint()
	bottomPaint.SetStyle(paintstyle.Stroke)
	bottomPaint.SetStrokeWidth(theme.BorderStrokeWidth)
	if disabled {
		bottomPaint.SetColor(theme.DefaultTheme.DisabledBorder)
	} else {
		bottomPaint.SetColor(t.ToolBarTheme.EdgeInk)
	}
	canvas.DrawLine(rect.X, rect.Y+drawH-1, rect.X+rect.Width, rect.Y+drawH-1, bottomPaint)

	t.positionWidgets()

	curLine := 0
	x := rect.X + t.padding
	y := rect.Y + t.padding

	for setIdx := range t.toolsets {
		ts := &t.toolsets[setIdx]
		if len(ts.Widgets) == 0 {
			continue
		}

		setW := toolbarHandleW + toolbarHandleGap
		if ts.Label != "" {
			setW += font.SimpleWidth(ts.Label) + t.spacing*2
		}
		for _, w := range ts.Widgets {
			if w == nil {
				setW += toolbarSepW
			} else {
				setW += toolbarBtnSize + t.spacing
			}
		}
		setW += toolbarSepW

		if t.wrap && (x-rect.X-t.padding)+setW > availW && x > rect.X+t.padding {
			curLine++
			y = rect.Y + float32(curLine)*lh + t.padding
			x = rect.X + t.padding
		}

		// Draw drag handle indicator
		handlePaint := graphics.NewPaint()
		h := ts.HandleRect
		dotY := h.Y + toolbarBtnSize/2 - 3
		dotColor := graphics.ARGB(0.35, 180, 180, 180)
		handlePaint.SetColor(dotColor)
		for i := 0; i < 3; i++ {
			canvas.DrawCircle(h.X+toolbarHandleW/2, dotY+float32(i)*3, 1.5, handlePaint)
		}

		x += toolbarHandleW + toolbarHandleGap

		if ts.Label != "" {
			labelW := font.SimpleWidth(ts.Label)
			labelPaint := graphics.NewPaint()
			labelPaint.SetColor(t.ToolBarTheme.TextInk)
			labelY := y + (toolbarBtnSize-font.LineHeight())/2 + font.Baseline()
			ver := theme.ThemeVersion()
			if ts.labelText == nil || ts.labelTextVer != ver {
				ts.labelText = graphics.NewText(ts.Label, &graphics.TextDecoration{Font: font, OnBackgroundInk: labelPaint.Color()})
				ts.labelTextVer = ver
			}
			if ts.labelText != nil {
				ts.labelText.Draw(canvas, x, labelY)
			}
			x += labelW + t.spacing*2
		}

		for _, w := range ts.Widgets {
			if w == nil {
				sepX := x + t.spacing*2
				sepPaint := graphics.NewPaint()
				sepPaint.SetStyle(paintstyle.Stroke)
				sepPaint.SetStrokeWidth(theme.SeparatorStrokeWidth)
				sepPaint.SetColor(t.ToolBarTheme.SeparatorInk)
				canvas.DrawLine(sepX, y+4, sepX, y+toolbarBtnSize-4, sepPaint)
				x += toolbarSepW
				continue
			}

			cf := w.FrameRect()
			canvas.Save()
			canvas.Translate(rect.X+cf.X, rect.Y+cf.Y)
			w.Paint(canvas, graphics.NewRect(0, 0, cf.Width, cf.Height))
			canvas.Restore()
			x += toolbarBtnSize + t.spacing
		}

		if setIdx < len(t.toolsets)-1 {
			sepX := x + t.spacing*2
			sepPaint := graphics.NewPaint()
			sepPaint.SetStyle(paintstyle.Stroke)
			sepPaint.SetStrokeWidth(theme.SeparatorStrokeWidth)
			sepPaint.SetColor(t.ToolBarTheme.SeparatorInk)
			canvas.DrawLine(sepX, y+4, sepX, y+toolbarBtnSize-4, sepPaint)
			x += toolbarSepW
		}
	}

	canvas.Restore()
}

func (t *ToolBar) OnMouseDown(x, y float32, button int) bool {
	if !t.Enabled() {
		return false
	}
	ch := t.contentHeight()
	availH := t.Height()
	if ch > availH {
		availH = ch
	}
	if x < 0 || y < 0 || y > availH {
		return false
	}

	t.positionWidgets()

	setIdx, wIdx := t.findWidget(x, y)
	fmt.Printf("DBG OnMouseDown: (%.0f,%.0f) set=%d widget=%d\n", x, y, setIdx, wIdx)
	if setIdx < 0 {
		return false
	}
	ts := &t.toolsets[setIdx]

	// If clicking the drag handle, start drag tracking
	if wIdx == -2 {
		t.dragSetIdx = setIdx
		t.dragStartX = x
		b := ts.BoundsRect
		rootOrigin := t.PointToRoot(b.X, b.Y)
		rootClick := t.PointToRoot(x, y)
		t.dragOffX = rootOrigin.X - rootClick.X
		t.dragOffY = rootOrigin.Y - rootClick.Y
		fmt.Printf("DBG  drag handle hit, tracking set %d, dragOff=(%.0f,%.0f)\n", setIdx, t.dragOffX, t.dragOffY)
		return true
	}

	// Dispatch to the widget
	if wIdx >= 0 && wIdx < len(ts.Widgets) && ts.Widgets[wIdx] != nil {
		w := ts.Widgets[wIdx]
		wRect := w.FrameRect()
		if handler, ok := w.(meui.EventListener); ok {
			return handler.OnMouseDown(x-wRect.X, y-wRect.Y, button)
		}
	}
	return false
}

func (t *ToolBar) OnMouseUp(x, y float32, button int) bool {
	if !t.Enabled() {
		return false
	}

	fmt.Printf("DBG OnMouseUp: dragging=%v dragSetIdx=%d\n", t.dragging, t.dragSetIdx)

	if t.dragging {
		t.dragging = false
		meui.ClearDragOverlay()
		// Reorder: find target position at drop point
		targetIdx := t.findSetAtX(x)
		fmt.Printf("DBG  drag release: dragSet=%d target=%d\n", t.dragSetIdx, targetIdx)
		if targetIdx >= 0 && targetIdx != t.dragSetIdx {
			set := t.toolsets[t.dragSetIdx]
			t.toolsets = append(t.toolsets[:t.dragSetIdx], t.toolsets[t.dragSetIdx+1:]...)
			if targetIdx > t.dragSetIdx {
				targetIdx--
			}
			t.toolsets = append(t.toolsets, ToolSet{})
			copy(t.toolsets[targetIdx+1:], t.toolsets[targetIdx:])
			t.toolsets[targetIdx] = set
			fmt.Printf("DBG  reordered: set %d moved to %d\n", t.dragSetIdx, targetIdx)
			t.MarkForRedraw()
		}
		t.dragSetIdx = -1
		return true
	}
	if t.dragSetIdx >= 0 {
		fmt.Printf("DBG  click on first widget, no drag\n")
		t.dragSetIdx = -1
	}

	t.positionWidgets()

	setIdx, wIdx := t.findWidget(x, y)
	if setIdx < 0 || wIdx < 0 {
		return false
	}
	ts := &t.toolsets[setIdx]
	if wIdx >= 0 && wIdx < len(ts.Widgets) && ts.Widgets[wIdx] != nil {
		w := ts.Widgets[wIdx]
		wRect := w.FrameRect()
		if handler, ok := w.(meui.EventListener); ok {
			return handler.OnMouseUp(x-wRect.X, y-wRect.Y, button)
		}
	}
	return false
}

func (t *ToolBar) renderDragGhost() {
	if t.dragSetIdx < 0 || t.dragSetIdx >= len(t.toolsets) {
		fmt.Printf("DBG renderDragGhost: invalid dragSetIdx=%d\n", t.dragSetIdx)
		return
	}
	b := t.toolsets[t.dragSetIdx].BoundsRect
	fmt.Printf("DBG renderDragGhost: BoundsRect=(%.0f,%.0f,%.0f,%.0f)\n", b.X, b.Y, b.Width, b.Height)
	if b.Width <= 0 || b.Height <= 0 {
		fmt.Printf("DBG renderDragGhost: zero bounds\n")
		return
	}
	w := int(b.Width)
	h := int(b.Height)
	fmt.Printf("DBG renderDragGhost: creating image %dx%d\n", w, h)
	img, err := graphics.NewImageFromDrawing(offscreenDevice(t), w, h, 72, func(c *graphics.Canvas) {
		ts := &t.toolsets[t.dragSetIdx]
		font := t.getFont()
		x := float32(0)
		y := float32(0)
		handlePaint := graphics.NewPaint()
		handlePaint.SetColor(graphics.ARGB(0.35, 180, 180, 180))
		for i := 0; i < 3; i++ {
			c.DrawCircle(toolbarHandleW/2, y+toolbarBtnSize/2-3+float32(i)*3, 1.5, handlePaint)
		}
		x += toolbarHandleW + toolbarHandleGap
		if ts.Label != "" {
			labelPaint := graphics.NewPaint()
			labelPaint.SetColor(t.ToolBarTheme.TextInk)
			labelY := y + (toolbarBtnSize-font.LineHeight())/2 + font.Baseline()
			ver := theme.ThemeVersion()
			if ts.labelText == nil || ts.labelTextVer != ver {
				ts.labelText = graphics.NewText(ts.Label, &graphics.TextDecoration{Font: font, OnBackgroundInk: labelPaint.Color()})
				ts.labelTextVer = ver
			}
			if ts.labelText != nil {
				ts.labelText.Draw(c, x, labelY)
			}
			x += font.SimpleWidth(ts.Label) + t.spacing*2
		}
		for _, wgt := range ts.Widgets {
			if wgt == nil {
				sepPaint := graphics.NewPaint()
				sepPaint.SetStyle(paintstyle.Stroke)
				sepPaint.SetStrokeWidth(theme.SeparatorStrokeWidth)
				sepPaint.SetColor(t.ToolBarTheme.SeparatorInk)
				c.DrawLine(x+t.spacing*2, y+4, x+t.spacing*2, y+toolbarBtnSize-4, sepPaint)
				x += toolbarSepW
				continue
			}
			c.Save()
			c.Translate(x, y)
			wgt.Paint(c, graphics.NewRect(0, 0, toolbarBtnSize, toolbarBtnSize))
			c.Restore()
			x += toolbarBtnSize + t.spacing
		}
	})
	if err != nil {
		fmt.Printf("DBG renderDragGhost: NewImageFromDrawing error: %v\n", err)
		return
	}
	if img == nil {
		fmt.Printf("DBG renderDragGhost: image is nil\n")
		return
	}
	rp := t.PointToRoot(b.X, b.Y)
	fmt.Printf("DBG renderDragGhost: PointToRoot -> (%.0f,%.0f)\n", rp.X, rp.Y)
	meui.SetDragOverlay(img, rp.X, rp.Y, b.Width, b.Height)
	fmt.Printf("DBG renderDragGhost: overlay set, GlobalOverlay=%v\n", meui.GlobalOverlay)
}

func (t *ToolBar) OnMouseMove(x, y float32) {
	if !t.Enabled() {
		return
	}

	// Activate drag if moved more than 5px
	if t.dragSetIdx >= 0 && !t.dragging {
		dx := x - t.dragStartX
		if dx < 0 {
			dx = -dx
		}
		if dx > 5 {
			t.dragging = true
			t.renderDragGhost()
			t.MarkForRedraw()
			fmt.Printf("DBG OnMouseMove: drag ACTIVATED for set %d (dx=%.0f)\n", t.dragSetIdx, dx)
		}
		return
	}

	if t.dragging {
		// Update ghost position to follow mouse
		if meui.GlobalOverlay != nil {
			rp := t.PointToRoot(x, y)
			meui.GlobalOverlay.X = rp.X + t.dragOffX
			meui.GlobalOverlay.Y = rp.Y + t.dragOffY
		}
		t.MarkForRedraw()
		return
	}

	ch := t.contentHeight()
	availH := t.Height()
	if ch > availH {
		availH = ch
	}
	if y < 0 || y > availH {
		t.clearHover()
		return
	}

	t.positionWidgets()

	// Dispatch mouse move to proper widget
	dispatched := false
	for _, ts := range t.toolsets {
		for _, w := range ts.Widgets {
			if w == nil {
				continue
			}
			wRect := w.FrameRect()
			inBounds := x >= wRect.X && x < wRect.X+wRect.Width &&
				y >= wRect.Y && y < wRect.Y+wRect.Height
			if handler, ok := w.(meui.EventListener); ok {
				if inBounds {
					handler.OnMouseMove(x-wRect.X, y-wRect.Y)
					dispatched = true
				} else {
					if h, ok := w.(meui.HoverHandler); ok {
						h.SetHoveredState(false)
					}
				}
			}
		}
	}
	if !dispatched {
		t.clearHover()
	}
}

// findWidget returns the toolset index and widget index for the given position.
// Special return values: wIdx == -2 means the drag handle area of the toolset.
func (t *ToolBar) findWidget(x, y float32) (int, int) {
	for si, ts := range t.toolsets {
		// Check handle area first
		h := ts.HandleRect
		if x >= h.X && x < h.X+h.Width && y >= h.Y && y < h.Y+h.Height {
			fmt.Printf("DBG findWidget: hit handle set=%d at (%.0f,%.0f,%.0f,%.0f)\n", si, h.X, h.Y, h.Width, h.Height)
			return si, -2
		}
		for wi, w := range ts.Widgets {
			if w == nil {
				continue
			}
			wRect := w.FrameRect()
			if x >= wRect.X && x < wRect.X+wRect.Width &&
				y >= wRect.Y && y < wRect.Y+wRect.Height {
				fmt.Printf("DBG findWidget: hit set=%d widget=%d at (%.0f,%.0f,%.0f,%.0f)\n", si, wi, wRect.X, wRect.Y, wRect.Width, wRect.Height)
				return si, wi
			}
		}
	}
	fmt.Printf("DBG findWidget: no hit at (%.0f,%.0f)\n", x, y)
	return -1, -1
}

// findSetAtX returns the toolset index at the given widget-local X position.
func (t *ToolBar) findSetAtX(mx float32) int {
	t.positionWidgets()
	best := -1
	for si, ts := range t.toolsets {
		wx := ts.HandleRect.X
		fmt.Printf("DBG findSetAtX: set=%d handleX=%.0f mx=%.0f\n", si, wx, mx)
		if best < 0 || mx >= wx {
			best = si
		}
		if wx > mx {
			break
		}
	}
	fmt.Printf("DBG findSetAtX: result=%d\n", best)
	return best
}

func (t *ToolBar) clearHover() {
	for _, ts := range t.toolsets {
		for _, w := range ts.Widgets {
			if w == nil {
				continue
			}
			if h, ok := w.(meui.HoverHandler); ok {
				h.SetHoveredState(false)
			}
		}
	}
}

func (t *ToolBar) positionWidgets() {
	frame := t.FrameRect()
	if frame.Width == 0 {
		return
	}
	font := t.getFont()
	availW := frame.Width - t.padding*2
	lh := t.lineH()

	curLine := 0
	x := t.padding
	y := t.padding

	for setIdx := range t.toolsets {
		ts := &t.toolsets[setIdx]
		if len(ts.Widgets) == 0 {
			continue
		}

		setW := float32(0)
		if ts.Label != "" {
			setW += font.SimpleWidth(ts.Label) + t.spacing*2
		}
		for _, w := range ts.Widgets {
			if w == nil {
				setW += toolbarSepW
			} else {
				setW += toolbarBtnSize + t.spacing
			}
		}
		setW += toolbarSepW

		if t.wrap && x+setW > availW && x > t.padding {
			curLine++
			y = float32(curLine)*lh + t.padding
			x = t.padding
		}

		handleX := x
		x += toolbarHandleW + toolbarHandleGap

		if ts.Label != "" {
			x += font.SimpleWidth(ts.Label) + t.spacing*2
		}

		for _, w := range ts.Widgets {
			if w == nil {
				x += toolbarSepW
				continue
			}
			w.SetFrameRect(graphics.NewRect(x, y, toolbarBtnSize, toolbarBtnSize))
			x += toolbarBtnSize + t.spacing
		}

		ts.HandleRect = graphics.NewRect(handleX, y, toolbarHandleW, toolbarBtnSize)
		ts.BoundsRect = graphics.NewRect(handleX, y, x-handleX, toolbarBtnSize)
		x += toolbarSepW
	}
}

func (t *ToolBar) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "text":
		if s, ok := value.(string); ok {
			t.AddButton(s, "")
		}
	}
}
