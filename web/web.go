// Package web is the small behaviour layer a web-looking meui scene is wired in.
//
// QML in this toolkit is a builder, not a language: there are no expressions and no
// signal-handler bodies, so every behaviour is attached from Go by id. That is the
// agreed model ("still wired on golang behaviour"), and this package is the part
// that keeps such a scene's Go short and honest:
//
//	scene, _ := web.Load("page.qml", "web-light", 1280, 900)
//	scene.OnClick("ctaPrimary", func() { ... })
//	scene.OnHover("cardPricing", func(inside bool) { ... })
//	scene.Shot(1280, 900, "landing.png")
//
// Every lookup reports a MISS. A scene that silently does nothing because an id was
// misspelled is the failure mode this tree keeps finding ("a capability nobody
// switched on"), so a missing id is a log line, not a shrug.
package web

import (
	"fmt"
	"image/png"
	"log"
	"os"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/qml"
	"github.com/sixnineone/meui/resources"
	"github.com/sixnineone/meui/theme"
	"github.com/sixnineone/meui/widgets"
)

// Scene is a parsed QML document plus its id registry.
type Scene struct {
	Root meui.Widget
	Reg  *qml.BoundWidgetRegistry
	Ctx  *qml.BindingContext
	Path string
}

// Load installs the preset, parses path and returns the scene. The preset is
// applied BEFORE parsing because scene colours may be resolved at parse time.
func Load(path, preset string, w, h int) (*Scene, error) {
	if preset != "" {
		if err := theme.LoadPreset(preset); err != nil {
			return nil, fmt.Errorf("web: preset %q: %w", preset, err)
		}
	}
	res, err := qml.Load(path)
	if err != nil {
		return nil, err
	}
	if res.Widget == nil {
		return nil, fmt.Errorf("web: %s parsed to no root widget", path)
	}
	if win, ok := res.Widget.(*meui.Window); ok {
		win.SetFrameRect(graphics.NewRect(0, 0, float32(w), float32(h)))
	}
	s := &Scene{Root: res.Widget, Reg: res.Widgets, Ctx: res.Context, Path: path}
	s.Root.SetFrameRect(graphics.NewRect(0, 0, float32(w), float32(h)))
	return s, nil
}

// LoadBundle is Load with the scene and its assets coming from a resources
// bundle. The URI uses a scheme mounted on the bundle, e.g.
// res://scenes/main.qml.
func LoadBundle(bundle *resources.Bundle, uri, preset string, w, h int) (*Scene, error) {
	if preset != "" {
		if err := theme.LoadPreset(preset); err != nil {
			return nil, fmt.Errorf("web: preset %q: %w", preset, err)
		}
	}
	res, err := qml.LoadBundle(bundle, uri)
	if err != nil {
		return nil, err
	}
	if res.Widget == nil {
		return nil, fmt.Errorf("web: %s parsed to no root widget", uri)
	}
	if win, ok := res.Widget.(*meui.Window); ok {
		win.SetFrameRect(graphics.NewRect(0, 0, float32(w), float32(h)))
	}
	s := &Scene{Root: res.Widget, Reg: res.Widgets, Ctx: res.Context, Path: uri}
	s.Root.SetFrameRect(graphics.NewRect(0, 0, float32(w), float32(h)))
	return s, nil
}

// Parse is Load for an in-memory document (tests, small probes).
func Parse(src, preset string, w, h int) (*Scene, error) {
	if preset != "" {
		if err := theme.LoadPreset(preset); err != nil {
			return nil, fmt.Errorf("web: preset %q: %w", preset, err)
		}
	}
	res, err := qml.ParseWithBindings(src)
	if err != nil {
		return nil, err
	}
	if res.Widget == nil {
		return nil, fmt.Errorf("web: document parsed to no root widget")
	}
	s := &Scene{Root: res.Widget, Reg: res.Widgets, Ctx: res.Context}
	s.Root.SetFrameRect(graphics.NewRect(0, 0, float32(w), float32(h)))
	return s, nil
}

// Get resolves an id, logging a miss with the scene path so a typo is visible.
func (s *Scene) Get(id string) meui.Widget {
	if s == nil || s.Reg == nil {
		log.Printf("web: %s: no registry", s.sceneName())
		return nil
	}
	w := s.Reg.Get(id)
	if w == nil {
		log.Printf("web: %s: no widget with id %q (the wiring for it is dead)", s.sceneName(), id)
	}
	return w
}

func (s *Scene) sceneName() string {
	if s == nil || s.Path == "" {
		return "<memory>"
	}
	return s.Path
}

// OnClick wires a click handler onto the id. It accepts the two shapes this tree
// has for a clickable surface — a *meui.Panel (Rectangle with clickable: true) and
// a *meui.MouseArea — and reports an error for anything else, because silently
// ignoring it is how a scene ends up looking interactive and not being it.
func (s *Scene) OnClick(id string, fn func()) bool {
	w := s.Get(id)
	if w == nil {
		return false
	}
	switch v := w.(type) {
	case *meui.Panel:
		v.SetOnClick(fn)
		return true
	case *meui.MouseArea:
		v.OnClicked = fn
		return true
	}
	log.Printf("web: %s: %q (%T) is not clickable; use clickable: true or a MouseArea", s.sceneName(), id, w)
	return false
}

// OnHover wires hover enter/leave. A clickable Panel tracks hover itself (it paints
// the theme wash); the callback is for scene-specific behaviour.
func (s *Scene) OnHover(id string, fn func(inside bool)) bool {
	w := s.Get(id)
	if w == nil {
		return false
	}
	// Concrete types on purpose: the setters are FLUENT (they return *BaseWidget),
	// so an `interface{ SetOnMouseEnter(func()) }` assertion never matches — the
	// exact trap that made the touch floor inert for two years (profile.go).
	switch v := w.(type) {
	case *meui.Panel:
		v.SetOnMouseEnter(func() { fn(true) })
		v.SetOnMouseExit(func() { fn(false) })
		return true
	case *meui.MouseArea:
		v.SetOnMouseEnter(func() { fn(true) })
		v.SetOnMouseExit(func() { fn(false) })
		return true
	}
	log.Printf("web: %s: %q (%T) does not take hover callbacks", s.sceneName(), id, w)
	return false
}

// SetText updates a label-ish widget's text through the toolkit's bound-property
// hook, which every text widget implements.
func (s *Scene) SetText(id, text string) bool {
	w := s.Get(id)
	if w == nil {
		return false
	}
	if bw, ok := w.(interface{ SetBoundProperty(string, interface{}) }); ok {
		bw.SetBoundProperty("text", text)
		return true
	}
	log.Printf("web: %s: %q (%T) has no text property", s.sceneName(), id, w)
	return false
}

// OnPaint installs a custom painter on a Rectangle — the escape hatch for a scene
// that needs chrome the widget set does not have (a chart, a gradient orb, a
// decorative rule). The callback receives the box's local rect.
func (s *Scene) OnPaint(id string, fn func(*graphics.Canvas, graphics.Rect)) bool {
	w := s.Get(id)
	if w == nil {
		return false
	}
	if p, ok := w.(*meui.Panel); ok {
		p.DrawCallback = fn
		p.MarkForRedraw()
		return true
	}
	log.Printf("web: %s: %q (%T) cannot take a paint callback", s.sceneName(), id, w)
	return false
}

// SetFrame places a widget by id: the responsive twin of x/y in QML. It is how a
// breakpoint moves one element without a second scene.
func (s *Scene) SetFrame(id string, x, y, w, h float32) bool {
	wd := s.Get(id)
	if wd == nil {
		return false
	}
	wd.SetFrameRect(graphics.NewRect(x, y, w, h))
	wd.MarkForRedraw()
	return true
}

// Show hides or shows a widget by id.
func (s *Scene) Show(id string, visible bool) bool {
	wd := s.Get(id)
	if wd == nil {
		return false
	}
	// The FLUENT shape, not `interface{ SetVisible(bool) }`: every widget inherits
	// BaseWidget.SetVisible, which returns *BaseWidget, so the no-return form matches
	// nothing — the trap this tree documents and that made the touch floor inert.
	if v, ok := wd.(interface{ SetVisible(bool) *meui.BaseWidget }); ok {
		v.SetVisible(visible)
		return true
	}
	if v, ok := wd.(interface{ ApplyVisible(bool) }); ok {
		v.ApplyVisible(visible)
		return true
	}
	log.Printf("web: %s: %q (%T) cannot be hidden", s.sceneName(), id, wd)
	return false
}

// Select marks a box as the chosen one in its group (a nav item, a list row, a day
// cell). It is the declarative counterpart of the per-scene colour swaps the first
// demos did by hand.
func (s *Scene) Select(id string, selected bool) bool {
	wd := s.Get(id)
	if wd == nil {
		return false
	}
	if p, ok := wd.(*meui.Panel); ok {
		p.SetSelected(selected)
		return true
	}
	log.Printf("web: %s: %q (%T) cannot be selected", s.sceneName(), id, wd)
	return false
}

// ScrollTo scrolls a ScrollView to a content y. Pair it with SetFrame / Select to
// reveal a chosen row without the scene keeping its own scroll state.
func (s *Scene) ScrollTo(id string, y float32) bool {
	wd := s.Get(id)
	if wd == nil {
		return false
	}
	if sp, ok := wd.(*widgets.ScrollPane); ok {
		sp.ScrollTo(y)
		return true
	}
	log.Printf("web: %s: %q (%T) is not a ScrollView", s.sceneName(), id, wd)
	return false
}

// Breakpoint applies a layout when the scene's width falls in its range. The highest
// MinWidth that the width reaches wins; the width can change with every resize, so an
// Apply receives the exact size and may compute continuous sizes (a cover that spans
// the page) as well as discrete ones (a sidebar that disappears).
type Breakpoint struct {
	MinWidth int32
	Apply    func(scene *Scene, width, height int32)
}

// Breakpoints wires the scene's window resize hook. It exists because a web page is a
// layout that RESPONDS: the same scene has to survive a 900px window and a 1600px one.
func (s *Scene) Breakpoints(bps []Breakpoint) {
	win, ok := s.Root.(*meui.Window)
	if !ok {
		log.Printf("web: %s: breakpoints need a *meui.Window root, got %T", s.sceneName(), s.Root)
		return
	}
	apply := func(width, height float32) {
		w, h := int32(width), int32(height)
		chosen := Breakpoint{}
		found := false
		for _, b := range bps {
			if w >= b.MinWidth {
				chosen = b
				found = true
			}
		}
		if !found {
			if len(bps) == 0 {
				return
			}
			chosen = bps[0]
		}
		if chosen.Apply != nil {
			chosen.Apply(s, w, h)
		}
		win.MarkForRedraw()
	}
	win.SetOnResize(apply)
	apply(win.Width(), win.Height())
}

// Shot rasterises the scene at w×h through the software device and writes a PNG.
// It is the fast review loop: no display, no GL, no window.
func (s *Scene) Shot(w, h int, out string) error {
	return Shot(s.Root, w, h, out)
}

// Shot rasterises any widget tree to a PNG through the software device. The tree is
// laid out through the same Paint path the window uses, so radius, shadow, gradient,
// clip and text are the toolkit's, not a second renderer's.
func Shot(root meui.Widget, w, h int, out string) error {
	if root == nil {
		return fmt.Errorf("web: shot of a nil root")
	}
	if w <= 0 || h <= 0 {
		return fmt.Errorf("web: shot size %dx%d", w, h)
	}
	root.SetFrameRect(graphics.NewRect(0, 0, float32(w), float32(h)))
	canvas, dev := graphics.NewSoftwareCanvas(w, h)
	canvas.Clear(uint32(theme.DefaultTheme.Background))
	root.Paint(canvas, graphics.NewRect(0, 0, float32(w), float32(h)))

	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, dev.Image()); err != nil {
		return err
	}
	return nil
}
