// Package flow provides declarative widget wrappers for MeUi.
// Widgets are defined as Go structs and built into a meui.Widget tree
// via Build(), which works like a compile-time QML parser.
package flow

import (
	"fmt"
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/layout"
	"github.com/sixnineone/meui/widgets"
	"os"
)

// WidgetWrapper is the interface implemented by all declarative wrappers.
type WidgetWrapper interface {
	Build(ctx *BuildContext) meui.Widget
}

// BuildContext holds the registry of named widgets for reference capture.
type BuildContext struct {
	registry map[string]meui.Widget
}

func NewBuildContext() *BuildContext {
	return &BuildContext{registry: make(map[string]meui.Widget)}
}

func (ctx *BuildContext) Register(id string, w meui.Widget) {
	ctx.registry[id] = w
}

func (ctx *BuildContext) Get(id string) meui.Widget {
	return ctx.registry[id]
}

func Build(root WidgetWrapper) (meui.Widget, *BuildContext) {
	ctx := NewBuildContext()
	w := root.Build(ctx)
	return w, ctx
}

// ── Common base ────────────────────────────────────────

// W is the common base embedded by all flow wrappers.
type W struct {
	ID      string  `qml:"id"`
	X       float32 `qml:"x"`
	Y       float32 `qml:"y"`
	Width   float32 `qml:"width"`
	Height  float32 `qml:"height"`
	Visible bool    `qml:"visible"`
	Enabled bool    `qml:"enabled"`
	// visibleSet/enabledSet record that QML explicitly set the property —
	// the Go zero value of both bools is false, but "unset" must mean
	// enabled+visible (widgets default to enabled). Without the markers
	// every flow widget without an explicit `enabled: true` was silently
	// disabled.
	VisibleSet bool    `qml:"-"`
	EnabledSet bool    `qml:"-"`
	Opacity    float32 `qml:"opacity"`

	// Layout properties (prefix "Layout.")
	FillWidth       bool    `qml:"Layout.fillWidth"`
	FillHeight      bool    `qml:"Layout.fillHeight"`
	MinimumWidth    float32 `qml:"Layout.minimumWidth"`
	MaximumWidth    float32 `qml:"Layout.maximumWidth"`
	MinimumHeight   float32 `qml:"Layout.minimumHeight"`
	MaximumHeight   float32 `qml:"Layout.maximumHeight"`
	PreferredWidth  float32 `qml:"Layout.preferredWidth"`
	PreferredHeight float32 `qml:"Layout.preferredHeight"`
	MarginLeft      float32 `qml:"Layout.leftMargin"`
	MarginRight     float32 `qml:"Layout.rightMargin"`
	MarginTop       float32 `qml:"Layout.topMargin"`
	MarginBottom    float32 `qml:"Layout.bottomMargin"`
	LayoutAlignment int     `qml:"Layout.alignment"`

	// Anchors (prefix "anchors.")
	AnchorsFill         string  `qml:"anchors.fill"`
	AnchorsCenterIn     string  `qml:"anchors.centerIn"`
	AnchorLeft          string  `qml:"anchors.left"`
	AnchorRight         string  `qml:"anchors.right"`
	AnchorTop           string  `qml:"anchors.top"`
	AnchorBottom        string  `qml:"anchors.bottom"`
	AnchorHCenter       string  `qml:"anchors.horizontalCenter"`
	AnchorVCenter       string  `qml:"anchors.verticalCenter"`
	AnchorMargins       float32 `qml:"anchors.margins"`
	AnchorLeftMargin    float32 `qml:"anchors.leftMargin"`
	AnchorRightMargin   float32 `qml:"anchors.rightMargin"`
	AnchorTopMargin     float32 `qml:"anchors.topMargin"`
	AnchorBottomMargin  float32 `qml:"anchors.bottomMargin"`
	AnchorHCenterOffset float32 `qml:"anchors.horizontalCenterOffset"`
	AnchorVCenterOffset float32 `qml:"anchors.verticalCenterOffset"`
}

// ApplyLayout sets layout/geometry/common properties on a widget.
func ApplyLayout(w meui.Widget, ww *W) {
	if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
		fmt.Printf("[applyLayout] ENTER %T\n", w)
	}
	if ww == nil {
		return
	}
	if ww.VisibleSet && !ww.Visible {
		if s, ok := w.(interface{ ApplyVisible(bool) }); ok {
			s.ApplyVisible(false)
		}
	}
	if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
		fmt.Printf("[applyLayout] %T EnabledSet=%v Enabled=%v\n", w, ww.EnabledSet, ww.Enabled)
	}
	if ww.EnabledSet && !ww.Enabled {
		if s, ok := w.(interface{ ApplyEnabled(bool) }); ok {
			s.ApplyEnabled(false)
		}
	}
	if ww.Opacity > 0 && ww.Opacity <= 1 {
		if s, ok := w.(interface{ SetOpacity(float32) }); ok {
			s.SetOpacity(ww.Opacity)
		}
	}
	if ww.X != 0 {
		if s, ok := w.(interface {
			SetX(float32) *meui.BaseWidget
		}); ok {
			s.SetX(ww.X)
		}
	}
	if ww.Y != 0 {
		if s, ok := w.(interface {
			SetY(float32) *meui.BaseWidget
		}); ok {
			s.SetY(ww.Y)
		}
	}
	if ww.Width > 0 {
		if s, ok := w.(interface{ SetWidthDirect(float32) }); ok {
			s.SetWidthDirect(ww.Width)
		}
	}
	if ww.Height > 0 {
		if s, ok := w.(interface{ SetHeightDirect(float32) }); ok {
			s.SetHeightDirect(ww.Height)
		}
	}

	// Layout props
	props := &layout.LayoutProps{
		FillWidth:       ww.FillWidth,
		FillHeight:      ww.FillHeight,
		MinimumWidth:    ww.MinimumWidth,
		MaximumWidth:    ww.MaximumWidth,
		MinimumHeight:   ww.MinimumHeight,
		MaximumHeight:   ww.MaximumHeight,
		PreferredWidth:  ww.PreferredWidth,
		PreferredHeight: ww.PreferredHeight,
		LeftMargin:      ww.MarginLeft,
		RightMargin:     ww.MarginRight,
		TopMargin:       ww.MarginTop,
		BottomMargin:    ww.MarginBottom,
		Alignment:       ww.LayoutAlignment,
	}
	if props.FillWidth || props.FillHeight || props.MinimumWidth != 0 || props.MaximumWidth != 0 ||
		props.MinimumHeight != 0 || props.MaximumHeight != 0 ||
		props.PreferredWidth != 0 || props.PreferredHeight != 0 ||
		props.LeftMargin != 0 || props.RightMargin != 0 ||
		props.TopMargin != 0 || props.BottomMargin != 0 || props.Alignment != 0 {
		if lw, ok := w.(interface{ SetLayoutProps(*layout.LayoutProps) }); ok {
			lw.SetLayoutProps(props)
		}
	}

	// Anchors
	anchors := layout.NewAnchors()
	hasAnchors := false

	if ww.AnchorsFill != "" {
		anchors.Fill = nil // parent will be resolved later
		hasAnchors = true
		_ = ww.AnchorsFill
	}
	if ww.AnchorsCenterIn != "" {
		// anchors.CenterIn = resolved parent
		hasAnchors = true
		_ = ww.AnchorsCenterIn
	}
	if ww.AnchorLeft != "" {
		hasAnchors = true
	}
	if ww.AnchorRight != "" {
		hasAnchors = true
	}
	if ww.AnchorTop != "" {
		hasAnchors = true
	}
	if ww.AnchorBottom != "" {
		hasAnchors = true
	}
	if ww.AnchorHCenter != "" {
		hasAnchors = true
	}
	if ww.AnchorVCenter != "" {
		hasAnchors = true
	}
	if ww.AnchorMargins != 0 {
		anchors.Margins = ww.AnchorMargins
		hasAnchors = true
	}
	if ww.AnchorLeftMargin != 0 {
		anchors.LeftMargin = ww.AnchorLeftMargin
		hasAnchors = true
	}
	if ww.AnchorRightMargin != 0 {
		anchors.RightMargin = ww.AnchorRightMargin
		hasAnchors = true
	}
	if ww.AnchorTopMargin != 0 {
		anchors.TopMargin = ww.AnchorTopMargin
		hasAnchors = true
	}
	if ww.AnchorBottomMargin != 0 {
		anchors.BottomMargin = ww.AnchorBottomMargin
		hasAnchors = true
	}
	if ww.AnchorHCenterOffset != 0 {
		anchors.HCenterOffset = ww.AnchorHCenterOffset
		hasAnchors = true
	}
	if ww.AnchorVCenterOffset != 0 {
		anchors.VCenterOffset = ww.AnchorVCenterOffset
		hasAnchors = true
	}

	if hasAnchors {
		if aw, ok := w.(interface{ SetAnchors(*layout.Anchors) }); ok {
			aw.SetAnchors(anchors)
		}
	}
}

// registerID registers the widget with the context if an ID is set.
func registerID(ctx *BuildContext, w meui.Widget, id string) {
	if id != "" {
		ctx.Register(id, w)
	}
}

// ── Window wrapper ────────────────────────────────────

type Window struct {
	W
	Title    string `qml:"title"`
	Children []WidgetWrapper
}

func (w *Window) Build(ctx *BuildContext) meui.Widget {
	win := meui.NewWindow()
	ApplyLayout(win, &w.W)
	for _, child := range w.Children {
		cw := child.Build(ctx)
		cw.SetParent(win)
		win.AddChild(cw)
	}
	// Apply anchor layout so first child fills the window
	if len(w.Children) > 0 && win.Layout() == nil {
		winFrame := win.FrameRect()
		anchorLayout := layout.NewAnchorLayout()
		anchorLayout.Layout(win, graphics.NewRect(0, 0, winFrame.Width, winFrame.Height))
	}
	registerID(ctx, win, w.ID)
	return win
}

// ── Panel wrapper ─────────────────────────────────────

type Panel struct {
	W
	Children []WidgetWrapper
}

func (p *Panel) Build(ctx *BuildContext) meui.Widget {
	panel := meui.NewPanel()
	ApplyLayout(panel, &p.W)
	for _, child := range p.Children {
		panel.AddChild(child.Build(ctx))
	}
	registerID(ctx, panel, p.ID)
	return panel
}

// ── VStack wrapper ────────────────────────────────────

type VStack struct {
	W
	Spacing  float32
	Children []WidgetWrapper
}

func (v *VStack) Build(ctx *BuildContext) meui.Widget {
	panel := meui.NewPanel()
	ApplyLayout(panel, &v.W)
	sp := v.Spacing
	if sp == 0 {
		sp = 4
	}
	vl := layout.NewVStack()
	vl.Spacing = sp
	panel.SetLayout(vl)
	for _, child := range v.Children {
		panel.AddChild(child.Build(ctx))
	}
	registerID(ctx, panel, v.ID)
	return panel
}

// ── HStack wrapper ────────────────────────────────────

type HStack struct {
	W
	Spacing  float32
	Children []WidgetWrapper
}

func (h *HStack) Build(ctx *BuildContext) meui.Widget {
	panel := meui.NewPanel()
	ApplyLayout(panel, &h.W)
	sp := h.Spacing
	if sp == 0 {
		sp = 4
	}
	hl := layout.NewHStack()
	hl.Spacing = sp
	panel.SetLayout(hl)
	for _, child := range h.Children {
		panel.AddChild(child.Build(ctx))
	}
	registerID(ctx, panel, h.ID)
	return panel
}

// ── Button wrapper ────────────────────────────────────

type Button struct {
	W
	Text         string                `qml:"text"`
	Icon         string                `qml:"icon"`
	OnClick      func()                `qml:"onClicked"`
	OnRightClick func()                `qml:"onRightClicked"`
	OnCreated    func(*widgets.Button) `qml:"-"`
}

func (b *Button) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewButton()
	ApplyLayout(w, &b.W)
	if b.Text != "" {
		w.SetText(b.Text)
	}
	if b.Icon != "" {
		w.SetIcon(b.Icon)
	}
	if b.OnClick != nil {
		w.OnClicked = b.OnClick
	}
	if b.OnRightClick != nil {
		w.OnRightClicked = b.OnRightClick
	}
	if b.OnCreated != nil {
		b.OnCreated(w)
	}
	registerID(ctx, w, b.ID)
	return w
}

// ── Label wrapper ─────────────────────────────────────

type Label struct {
	W
	Text      string               `qml:"text"`
	HAlign    string               `qml:"horizontalAlignment"`
	VAlign    string               `qml:"verticalAlignment"`
	Color     graphics.Color       `qml:"color"`
	RichText  bool                 `qml:"richText"`
	WrapMode  string               `qml:"wrapMode"`
	OnCreated func(*widgets.Label) `qml:"-"`
}

func (l *Label) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewLabel()
	ApplyLayout(w, &l.W)
	if l.Text != "" {
		w.SetText(l.Text)
	}
	if l.HAlign != "" {
		w.SetHAlign(l.HAlign)
	}
	_ = l.VAlign
	_ = l.Color
	if l.RichText {
		w.SetRichText(true)
	}
	if l.WrapMode != "" {
		w.SetWrapMode(l.WrapMode)
	}
	if l.OnCreated != nil {
		l.OnCreated(w)
	}
	registerID(ctx, w, l.ID)
	return w
}

// ── CheckBox wrapper ──────────────────────────────────

type CheckBox struct {
	W
	Text      string                  `qml:"text"`
	Checked   bool                    `qml:"checked"`
	OnCreated func(*widgets.CheckBox) `qml:"-"`
}

func (c *CheckBox) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewCheckBox()
	ApplyLayout(w, &c.W)
	if c.Text != "" {
		w.SetText(c.Text)
	}
	w.SetChecked(c.Checked)
	if c.OnCreated != nil {
		c.OnCreated(w)
	}
	registerID(ctx, w, c.ID)
	return w
}
