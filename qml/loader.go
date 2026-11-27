package qml

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"reflect"
	"strings"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/flow"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/slant"
	"github.com/sixnineone/meui/graphics/enums/spacing"
	"github.com/sixnineone/meui/graphics/enums/weight"
	"github.com/sixnineone/meui/layout"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
	"github.com/sixnineone/meui/widgets"
)

type QMLResult struct {
	Widget  meui.Widget
	Context *BindingContext
	Widgets *BoundWidgetRegistry
}

type Object struct {
	Type       string
	ID         string
	Properties []Property
	Children   []*Object
	Parent     *Object // QML parent in the document tree
	Line       int     // source line number (1-based)
	Column     int     // source column (0-based)
}

type Property struct {
	Name  string
	Value interface{}
	Line  int // source line number (1-based)
	Col   int // source column (0-based)
}

type BindingContext struct {
	objects      map[string]*Object
	boundWidgets map[string]interface{}
}

type BoundWidgetRegistry struct {
	widgets      map[string]interface{}
	flowWrappers map[string]interface{}
	ids          map[string]bool
	err          error
}

// Get retrieves a widget by ID from the registry
func (r *BoundWidgetRegistry) Get(id string) meui.Widget {
	if r == nil || r.widgets == nil {
		return nil
	}
	if w, ok := r.widgets[id]; ok {
		if widget, ok := w.(meui.Widget); ok {
			return widget
		}
	}
	return nil
}

// Register stores a widget by QML id. Returns silently on duplicate id
// but records the error — call Err() after loading to check.
func (r *BoundWidgetRegistry) Register(id string, widget interface{}) {
	if r == nil || id == "" || widget == nil {
		return
	}
	if r.err != nil {
		return // already have an error
	}
	if r.ids == nil {
		r.ids = make(map[string]bool)
	}
	if r.ids[id] {
		r.err = fmt.Errorf("duplicate QML id: %q", id)
		return
	}
	r.ids[id] = true
	r.widgets[id] = widget
	if r.flowWrappers == nil {
		r.flowWrappers = make(map[string]interface{})
	}
}

// RegisterFlow stores a flow wrapper alongside its widget by the same QML id.
func (r *BoundWidgetRegistry) RegisterFlow(id string, fw interface{}) {
	if r == nil || id == "" || fw == nil {
		return
	}
	if r.flowWrappers == nil {
		r.flowWrappers = make(map[string]interface{})
	}
	r.flowWrappers[id] = fw
}

// Err returns the first duplicate-ID error encountered during Register.
func (r *BoundWidgetRegistry) Err() error {
	if r == nil {
		return nil
	}
	return r.err
}

// Inject reads qml:"id" struct tags from model and populates fields
// from the registry. Returns an error if any referenced ID is missing
// or if a type mismatch occurs.
func (r *BoundWidgetRegistry) Inject(model interface{}) error {
	if r == nil {
		return fmt.Errorf("registry is nil")
	}
	val := reflect.ValueOf(model)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("Inject: model must be a non-nil pointer to struct, got %T", model)
	}
	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("Inject: model must be a pointer to struct, got pointer to %v", val.Kind())
	}
	typ := val.Type()
	var missingIDs []string
	for i := 0; i < typ.NumField(); i++ {
		ft := typ.Field(i)
		tag := ft.Tag.Get("qml")
		if tag == "" {
			continue
		}
		if !ft.IsExported() {
			continue
		}
		fv := val.Field(i)
		if !fv.CanSet() {
			continue
		}
		targetType := fv.Type()
		injected := false

		// Try widget first
		if w, ok := r.widgets[tag]; ok {
			widgetVal := reflect.ValueOf(w)
			if widgetVal.Type().AssignableTo(targetType) {
				fv.Set(widgetVal)
				injected = true
			}
		}
		// Try flow wrapper
		if !injected {
			if fw, ok := r.flowWrappers[tag]; ok {
				fwVal := reflect.ValueOf(fw)
				if fwVal.Type().AssignableTo(targetType) {
					fv.Set(fwVal)
					injected = true
				}
			}
		}
		if !injected {
			missingIDs = append(missingIDs, tag)
		}
	}
	if len(missingIDs) > 0 {
		return fmt.Errorf("Inject: QML id(s) not found in registry: %s", strings.Join(missingIDs, ", "))
	}
	return nil
}

// BoundWidget interface for widgets that support property bindings
type BoundWidget interface {
	SetBoundProperty(name string, value interface{})
}

var objectToWidget = make(map[*Object]meui.Widget)

func setObjectWidget(obj *Object, widget meui.Widget) {
	objectToWidget[obj] = widget
}

func getWidgetFromObject(obj *Object) meui.Widget {
	return objectToWidget[obj]
}

// setupAnchors configures QML parent resolution for anchors
func setupAnchors(obj *Object, anchors *layout.Anchors) {
	if anchors == nil || obj == nil {
		return
	}
	anchors.ParentObj = obj.Parent
	anchors.ResolveParentWidget = func(parentObj interface{}) layout.Widget {
		if qmlObj, ok := parentObj.(*Object); ok {
			return getWidgetFromObject(qmlObj)
		}
		return nil
	}
}

func NewBindingContext() *BindingContext {
	return &BindingContext{
		objects:      make(map[string]*Object),
		boundWidgets: make(map[string]interface{}),
	}
}

func NewBoundWidgetRegistry() *BoundWidgetRegistry {
	return &BoundWidgetRegistry{
		widgets: make(map[string]interface{}),
	}
}

func (ctx *BindingContext) RegisterObject(obj *Object) {
	if obj.ID != "" {
		ctx.objects[obj.ID] = obj
	}
}

func (ctx *BindingContext) ApplyBindings() {}

func (ctx *BindingContext) ResolveProperty(obj *Object, propName string, value interface{}) (interface{}, bool) {
	return nil, false
}

func (r *BoundWidgetRegistry) ApplyBindings(ctx *BindingContext) {}

func Load(path string) (*QMLResult, error) {
	meui.InitDebug()
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseWithBindings(string(data))
}

func LoadAndConvert(path string) (meui.Widget, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Convert(string(data))
}

func ConvertFile(path string) (meui.Widget, error) {
	return LoadAndConvert(path)
}

// ParseWithBindingsFile reads a QML file and returns the full result with registry.
func ParseWithBindingsFile(path string) (*QMLResult, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseWithBindings(string(data))
}

func ParseWithBindings(qml string) (*QMLResult, error) {
	// Initialize built-in widgets on first use
	initializeBuiltinWidgets()

	obj, err := ParseQML(qml)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, fmt.Errorf("no root object found")
	}

	// Validate QML structure before building widgets
	if err := ValidateQML(obj); err != nil {
		return nil, err
	}

	ctx := NewBindingContext()
	registry := NewBoundWidgetRegistry()

	widget, err := buildObjectWithBindings(obj, ctx, registry)
	if err != nil {
		return nil, err
	}

	// Set tree paths on all widgets
	meui.SetTreePaths(widget)

	// Print tree if debug is enabled
	fmt.Printf("MEUI_DEBUG=%s, DebugEnabled=%v, about to print tree\n", os.Getenv("MEUI_DEBUG"), meui.DebugEnabled)
	if meui.DebugEnabled {
		f, err := os.Create("/tmp/meui_tree.txt")
		if err == nil {
			fmt.Printf("Writing tree to /tmp/meui_tree.txt\n")
			meui.PrintTreeToFile(widget, 0, f)
			f.Close()
			fmt.Printf("Done writing tree\n")
		} else {
			fmt.Printf("Error creating file: %v\n", err)
		}
	}

	// Check for duplicate QML IDs
	if err := registry.Err(); err != nil {
		return nil, err
	}

	return &QMLResult{
		Widget:  widget,
		Context: ctx,
		Widgets: registry,
	}, nil
}

func Parse(qml string) (meui.Widget, error) {
	obj, err := ParseQML(qml)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, fmt.Errorf("no root object found")
	}

	// Validate QML structure before building widgets
	if err := ValidateQML(obj); err != nil {
		return nil, err
	}

	return buildObject(obj)
}

func Convert(qml string) (meui.Widget, error) {
	// Initialize built-in widgets on first use
	initializeBuiltinWidgets()

	obj, err := ParseQML(qml)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, fmt.Errorf("no widget created")
	}

	// Validate QML structure before building widgets
	if err := ValidateQML(obj); err != nil {
		return nil, err
	}

	countObjects(obj, 0)

	ctx := NewBindingContext()
	registry := NewBoundWidgetRegistry()

	widget, err := buildObjectWithBindings(obj, ctx, registry)
	if err != nil {
		return nil, err
	}

	return widget, nil
}

func countObjects(obj *Object, depth int) {
	indent := strings.Repeat("  ", depth)
	log.Printf("%sObject: type=%s, props=%d, children=%d", indent, obj.Type, len(obj.Properties), len(obj.Children))
	for _, child := range obj.Children {
		countObjects(child, depth+1)
	}
}

func buildObject(obj *Object) (meui.Widget, error) {
	// Use widget registry for standard widgets
	widget, err := CreateWidget(obj.Type)
	if err == nil {
		if w, ok := widget.(interface{ SetID(string) }); ok && obj.ID != "" {
			w.SetID(obj.ID)
		}
		setObjectWidget(obj, widget)
		applyCommonProperties(obj, widget)
		// Apply flow wrapper properties via reflection (if registered)
		if entry := GetEntry(obj.Type); entry != nil && entry.FlowWrapper != nil {
			fw := entry.FlowWrapper()
			props := makePropMap(obj.Properties)
			flow.ApplyProperties(props, fw)
			flow.ApplyLayout(widget, getW(fw))
		}
		// Apply registered property handlers (overrides reflection for complex props)
		for _, prop := range obj.Properties {
			HandleProperty(widget, prop.Name, prop.Value)
		}
		applyWidgetProperties(obj, widget)
		return widget, nil
	}

	// Special cases not in registry (Window, TabPanel, Rectangle, layouts)
	var specialWidget meui.Widget
	switch obj.Type {
	case "Window":
		specialWidget = buildWindow(obj)
	case "TabPanel":
		specialWidget = buildTabPanel(obj)
	case "Rectangle":
		specialWidget = buildRectangle(obj)
	case "Item":
		specialWidget = buildBox(obj, 0, 0, true)
	case "MouseArea":
		specialWidget = buildMouseArea(obj)
	case "ScrollView":
		specialWidget = buildScrollView(obj)
	case "Column", "Row", "Grid", "Flex", "HStack", "VStack":
		specialWidget = buildLayout(obj)
	case "Text":
		specialWidget = buildLabel(obj)
	default:
		specialWidget = buildPanel(obj)
	}

	if specialWidget != nil {
		setObjectWidget(obj, specialWidget)
		applyCommonProperties(obj, specialWidget)
	}
	return specialWidget, nil
}

// makePropMap converts QML []Property to map[string]interface{}.
func makePropMap(props []Property) map[string]interface{} {
	m := make(map[string]interface{}, len(props))
	for _, p := range props {
		m[p.Name] = p.Value
	}
	return m
}

// getW extracts the embedded *flow.W from a flow wrapper struct.
func getW(fw interface{}) *flow.W {
	val := reflect.ValueOf(fw)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if w := val.FieldByName("W"); w.IsValid() {
		return w.Addr().Interface().(*flow.W)
	}
	return nil
}

// applyCommonProperties handles anchors, layout properties, and other common QML properties
func applyCommonProperties(obj *Object, widget meui.Widget) {
	var anchors *layout.Anchors
	var layoutProps *layout.LayoutProps

	for _, prop := range obj.Properties {
		if strings.HasPrefix(prop.Name, "anchors.") {
			if anchors == nil {
				anchors = layout.NewAnchors()
			}
			anchors = parseAnchorsProp(prop.Name, prop.Value, anchors, widget)
		} else if strings.HasPrefix(prop.Name, "Layout.") {
			if layoutProps == nil {
				layoutProps = &layout.LayoutProps{}
			}
			applyLayoutProperty(prop.Name, prop.Value, layoutProps)
		}
	}

	if anchors != nil {
		setupAnchors(obj, anchors)
		if aw, ok := widget.(interface{ SetAnchors(*layout.Anchors) }); ok {
			aw.SetAnchors(anchors)
		}
	}
	if layoutProps != nil {
		if lw, ok := widget.(interface{ SetLayoutProps(*layout.LayoutProps) }); ok {
			lw.SetLayoutProps(layoutProps)
		}
	}

	// Handle x, y, width, height directly
	for _, prop := range obj.Properties {
		switch prop.Name {
		case "z":
			// Sibling paint/hit order, on ANY widget. The fluent setter returns
			// *meui.BaseWidget — the shape every widget inherits — so assert THAT
			// interface, not the no-return one (the fluent-setter trap).
			if f, ok := toFloat32(prop.Value); ok {
				if zw, ok := widget.(interface {
					SetZIndex(float32) *meui.BaseWidget
				}); ok {
					zw.SetZIndex(f)
				}
			}
		case "x":
			if f, ok := toFloat32(prop.Value); ok {
				if w, ok := widget.(interface {
					SetX(float32) *meui.BaseWidget
				}); ok {
					w.SetX(f)
				}
			}
		case "y":
			if f, ok := toFloat32(prop.Value); ok {
				if w, ok := widget.(interface {
					SetY(float32) *meui.BaseWidget
				}); ok {
					w.SetY(f)
				}
			}
		case "width":
			if f, ok := toFloat32(prop.Value); ok {
				if w, ok := widget.(interface{ SetWidthDirect(float32) }); ok {
					w.SetWidthDirect(f)
				}
			}
		case "height":
			if f, ok := toFloat32(prop.Value); ok {
				if w, ok := widget.(interface{ SetHeightDirect(float32) }); ok {
					w.SetHeightDirect(f)
				}
			}
		// `visible` and `enabled` are COMMON properties — every widget has them — but they
		// used to be applied only through a widget's flow wrapper (flow.ApplyLayout), and
		// only 34 widgets have one. So for the whole mobile set (Snackbar, TopAppBar,
		// NavigationBar/Rail, InsetSpacer, BottomSheet) `visible: false` in QML was silently
		// IGNORED, and a hidden Snackbar in the shell's flow took every press in its band.
		case "opacity":
			if f, ok := toFloat32(prop.Value); ok {
				if o, ok := widget.(interface{ ApplyOpacity(float32) }); ok {
					o.ApplyOpacity(f)
				}
			}
		case "visible":
			if b, ok := prop.Value.(bool); ok {
				if v, ok := widget.(interface{ ApplyVisible(bool) }); ok {
					v.ApplyVisible(b)
				} else if v, ok := widget.(interface{ SetVisible(bool) }); ok {
					v.SetVisible(b)
				}
			}
		case "enabled":
			if b, ok := prop.Value.(bool); ok {
				if v, ok := widget.(interface{ ApplyEnabled(bool) }); ok {
					v.ApplyEnabled(b)
				} else if v, ok := widget.(interface{ SetEnabled(bool) }); ok {
					v.SetEnabled(b)
				}
			}
		}
	}
}

// applyWidgetProperties handles widget-specific properties (text, color, groupID, etc.)
func applyWidgetProperties(obj *Object, widget meui.Widget) {
	// All property dispatch is now handled by:
	// 1. flow.ApplyProperties() — reflection on flow wrappers (in buildObject)
	// 2. HandleProperty() — registered handlers in handlers.go
	// 3. SetBoundProperty() — catch-all fallback via handlers.go
	//
	// This function is kept for backward compatibility with special widget types
	// (Window, TabPanel, Rectangle, layouts) that don't go through the registry.
	for _, prop := range obj.Properties {
		HandleProperty(widget, prop.Name, prop.Value)
	}
}

// applyLayoutProperty parses a Layout.* property and sets it on LayoutProps
func applyLayoutProperty(propName string, propValue interface{}, props *layout.LayoutProps) {
	switch propName {
	case "Layout.fillWidth":
		if b, ok := propValue.(bool); ok {
			props.FillWidth = b
		}
	case "Layout.fillHeight":
		if b, ok := propValue.(bool); ok {
			props.FillHeight = b
		}
	case "Layout.minimumWidth":
		if f, ok := propValue.(float64); ok {
			props.MinimumWidth = float32(f)
		}
	case "Layout.maximumWidth":
		if f, ok := propValue.(float64); ok {
			props.MaximumWidth = float32(f)
		}
	case "Layout.minimumHeight":
		if f, ok := propValue.(float64); ok {
			props.MinimumHeight = float32(f)
		}
	case "Layout.maximumHeight":
		if f, ok := propValue.(float64); ok {
			props.MaximumHeight = float32(f)
		}
	case "Layout.preferredWidth":
		if f, ok := propValue.(float64); ok {
			props.PreferredWidth = float32(f)
		}
	case "Layout.preferredHeight":
		if f, ok := propValue.(float64); ok {
			props.PreferredHeight = float32(f)
		}
	case "Layout.leftMargin":
		if f, ok := propValue.(float64); ok {
			props.LeftMargin = float32(f)
		}
	case "Layout.rightMargin":
		if f, ok := propValue.(float64); ok {
			props.RightMargin = float32(f)
		}
	case "Layout.topMargin":
		if f, ok := propValue.(float64); ok {
			props.TopMargin = float32(f)
		}
	case "Layout.bottomMargin":
		if f, ok := propValue.(float64); ok {
			props.BottomMargin = float32(f)
		}
	case "Layout.margins":
		if f, ok := propValue.(float64); ok {
			props.LeftMargin = float32(f)
			props.RightMargin = float32(f)
			props.TopMargin = float32(f)
			props.BottomMargin = float32(f)
		}
	case "Layout.alignment":
		if s, ok := propValue.(string); ok {
			props.Alignment = parseAlignment(s)
		}
	}
}

func buildObjectWithBindings(obj *Object, ctx *BindingContext, registry *BoundWidgetRegistry) (meui.Widget, error) {
	// Guard: if this QML Object was already built, return the existing widget
	if existing := getWidgetFromObject(obj); existing != nil {
		return existing, nil
	}

	var widget meui.Widget
	var err error

	// Use widget registry for standard widgets
	widget, err = CreateWidget(obj.Type)
	if err == nil {
		if w, ok := widget.(interface{ SetID(string) }); ok && obj.ID != "" {
			w.SetID(obj.ID)
		}
		setObjectWidget(obj, widget)
		applyCommonProperties(obj, widget)
		// Apply flow wrapper properties via reflection (if registered)
		if entry := GetEntry(obj.Type); entry != nil && entry.FlowWrapper != nil {
			fw := entry.FlowWrapper()
			props := makePropMap(obj.Properties)
			flow.ApplyProperties(props, fw)
			flow.ApplyLayout(widget, getW(fw))
			if obj.ID != "" {
				registry.RegisterFlow(obj.ID, fw)
			}
		}
		// Apply registered property handlers
		for _, prop := range obj.Properties {
			HandleProperty(widget, prop.Name, prop.Value)
		}
		applyWidgetProperties(obj, widget)

		// Process children for container widgets (Panel/Rectangle)
		if container, ok := widget.(interface{ AddChild(meui.Widget) }); ok {
			// Check if this is a DockPanel — handle dock edge routing
			dp, isDockPanel := widget.(*widgets.DockPanel)
			// StackPanel routes children through AddItem with an even size split
			sp2, isStackPanel := widget.(*widgets.StackPanel)
			for _, child := range obj.Children {
				if childWidget, err := buildObjectWithBindings(child, ctx, registry); err == nil && childWidget != nil {
					// ContextMenu children are registered globally by ID, not added as layout children
					if _, isCM := childWidget.(*widgets.ContextMenu); isCM {
						if child.ID != "" {
							meui.RegisterNamedWidget(child.ID, childWidget)
						}
						if sv, ok := childWidget.(interface{ SetVisible(bool) }); ok {
							sv.SetVisible(false)
						}
						continue
					}
					if isDockPanel {
						// Check for dock property on the child QML object
						edge := "center"
						for _, prop := range child.Properties {
							if prop.Name == "dock" {
								if s, ok := prop.Value.(string); ok {
									edge = s
								}
							}
						}
						switch edge {
						case "left":
							dp.AddToEdge(widgets.DockLeft, "", childWidget)
						case "right":
							dp.AddToEdge(widgets.DockRight, "", childWidget)
						case "top":
							dp.AddToEdge(widgets.DockTop, "", childWidget)
						case "bottom":
							dp.AddToEdge(widgets.DockBottom, "", childWidget)
						default:
							dp.SetCenter(childWidget)
						}
					} else if isStackPanel {
						n := len(obj.Children)
						avail := sp2.FrameRect().Height
						if sp2.Orientation == "horizontal" {
							avail = sp2.FrameRect().Width
						}
						each := float32(200)
						if avail > 0 && n > 0 {
							each = (avail - 4*float32(n-1)) / float32(n)
							if each < 1 {
								each = 1
							}
						}
						sp2.AddItem(childWidget, each)
					} else {
						if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
							fmt.Printf("[addChild] parent=%T child=%T (%p)\n", container, childWidget, childWidget)
						}
						container.AddChild(childWidget)
					}
				}
			}
		}

		// Register for bindings
		if obj.ID != "" {
			registry.Register(obj.ID, widget)
			ctx.RegisterObject(obj)
			// ContextMenus are also registered globally for ShowContextMenuPositioned
			if _, isCM := widget.(*widgets.ContextMenu); isCM {
				log.Printf("  GLOBAL REGISTER ContextMenu id=%q", obj.ID)
				meui.RegisterNamedWidget(obj.ID, widget)
			}
		}
		// Process bindings if widget supports it
		if bw, ok := widget.(BoundWidget); ok {
			processBindings(obj, ctx, bw)
		}
		return widget, nil
	}

	// Special cases not in registry
	switch obj.Type {
	case "Window":
		widget, err = buildWindowWithBindings(obj, ctx, registry), nil
	case "TabPanel":
		widget, err = buildTabPanelWithBindings(obj, ctx, registry), nil
	case "Rectangle":
		widget, err = buildRectangleWithBindings(obj, ctx, registry), nil
	case "Item":
		widget, err = buildBoxWithBindings(obj, ctx, registry, 0, 0), nil
	case "MouseArea":
		widget, err = buildMouseAreaWithBindings(obj, ctx, registry), nil
	case "ScrollView":
		widget, err = buildScrollViewWithBindings(obj, ctx, registry), nil
	case "Column", "Row", "Grid", "Flex", "HStack", "VStack":
		widget, err = buildLayoutWithBindings(obj, ctx, registry), nil
	case "Text":
		widget, err = buildLabelWithBindings(obj, ctx, registry), nil
	default:
		widget, err = buildPanelWithBindings(obj, ctx, registry), nil
	}

	// Register QML Object → Widget mapping
	if widget != nil {
		setObjectWidget(obj, widget)
		// Common properties (x/y, anchors, Layout.*, visible/enabled/opacity) apply
		// to special widgets too. They used to be applied only by the NON-bindings
		// builder, so a Rectangle inside a bindings parse ignored x/y — every child
		// piled up at 0,0. Each builder still applies what it owns (width/height,
		// colour); this pass is what makes a scene POSITIONED.
		applyCommonProperties(obj, widget)
	}

	// Set debug name from ID
	if widget != nil && obj.ID != "" {
		if bw, ok := widget.(interface{ SetDebugName(string) }); ok {
			bw.SetDebugName(obj.ID)
		}
	}

	return widget, err
}

func buildLabel(obj *Object) *widgets.Label {
	label := widgets.NewLabel()

	finalWidth, finalHeight := float32(200), float32(30)

	for _, prop := range obj.Properties {
		switch prop.Name {
		case "text":
			if s, ok := prop.Value.(string); ok {
				label.Text = s
			}
		case "richText":
			if b, ok := prop.Value.(bool); ok {
				label.RichText = b
			}
		case "onLinkClicked":
			if h, ok := prop.Value.(func(string)); ok {
				label.OnLinkClicked = h
			}
		case "color":
			if s, ok := prop.Value.(string); ok {
				if _, isToken := theme.TokenColor(s); isToken {
					label.SetColorToken(s)
				} else {
					label.LabelTheme.TextInk = parseColor(s)
				}
			}
		case "font.family":
			if s, ok := prop.Value.(string); ok {
				currentFont := label.LabelTheme.Font
				if currentFont == nil {
					currentFont = theme.DefaultTheme.BaseFont
				}
				if currentFont == nil {
					currentFont = graphics.SystemFont
				}
				size := currentFont.Size()
				if size == 0 {
					size = theme.DefaultTheme.FontSizeMedium
				}
				face := graphics.MatchFontFace(s, weight.Regular, spacing.Standard, slant.Upright)
				label.LabelTheme.Font = face.Font(size)
			}
		case "font.pixelSize":
			if f, ok := toFloat32(prop.Value); ok {
				currentFont := label.LabelTheme.Font
				if currentFont == nil {
					currentFont = theme.DefaultTheme.BaseFont
				}
				if currentFont == nil {
					currentFont = graphics.SystemFont
				}
				family := currentFont.Face().Family()
				w, sp, sl := currentFont.Face().Style()
				face := graphics.MatchFontFace(family, w, sp, sl)
				label.LabelTheme.Font = face.Font(f)
			}
		case "font.pointSize":
			if f, ok := toFloat32(prop.Value); ok {
				currentFont := label.LabelTheme.Font
				if currentFont == nil {
					currentFont = theme.DefaultTheme.BaseFont
				}
				if currentFont == nil {
					currentFont = graphics.SystemFont
				}
				family := currentFont.Face().Family()
				w, sp, sl := currentFont.Face().Style()
				face := graphics.MatchFontFace(family, w, sp, sl)
				label.LabelTheme.Font = face.Font(f)
			}
		case "font.bold":
			if b, ok := prop.Value.(bool); ok && b {
				currentFont := label.LabelTheme.Font
				if currentFont == nil {
					currentFont = theme.DefaultTheme.BaseFont
				}
				if currentFont == nil {
					currentFont = graphics.SystemFont
				}
				family := currentFont.Face().Family()
				size := currentFont.Size()
				face := graphics.MatchFontFace(family, weight.Bold, spacing.Standard, slant.Upright)
				label.LabelTheme.Font = face.Font(size)
			}
		case "font.italic":
			if b, ok := prop.Value.(bool); ok && b {
				currentFont := label.LabelTheme.Font
				if currentFont == nil {
					currentFont = theme.DefaultTheme.BaseFont
				}
				if currentFont == nil {
					currentFont = graphics.SystemFont
				}
				family := currentFont.Face().Family()
				size := currentFont.Size()
				face := graphics.MatchFontFace(family, weight.Regular, spacing.Standard, slant.Italic)
				label.LabelTheme.Font = face.Font(size)
			}
		case "font.weight":
			// A CSS-style weight (400/500/600/700) — the ramp web type depends on,
			// and the one property `font.bold` cannot express.
			if f, ok := toFloat32(prop.Value); ok {
				currentFont := label.LabelTheme.Font
				if currentFont == nil {
					currentFont = theme.DefaultTheme.BaseFont
				}
				if currentFont == nil {
					currentFont = graphics.SystemFont
				}
				family := currentFont.Face().Family()
				face := graphics.MatchFontFace(family, graphics.WeightEnum(int(f)), spacing.Standard, slant.Upright)
				label.LabelTheme.Font = face.Font(currentFont.Size())
			}
		case "font.familyOverride", "fontFamily":
			if s, ok := prop.Value.(string); ok {
				size := theme.DefaultTheme.FontSizeMedium
				if f := label.LabelTheme.Font; f != nil {
					size = f.Size()
				}
				face := graphics.MatchFontFace(s, weight.Regular, spacing.Standard, slant.Upright)
				label.LabelTheme.Font = face.Font(size)
			}
		case "lineHeight":
			if f, ok := toFloat32(prop.Value); ok {
				label.SetLineHeight(f)
			}
		case "maximumLineCount", "maxLines":
			if f, ok := toFloat32(prop.Value); ok {
				label.SetMaxLines(int(f))
			}
		case "elide":
			if s, ok := prop.Value.(string); ok && (s == "true" || s == "elide" || s == "ElideRight") {
				label.SetWrapMode(widgets.WrapEllipsis)
			}
			if b, ok := prop.Value.(bool); ok && b {
				label.SetWrapMode(widgets.WrapEllipsis)
			}
		case "wrapMode":
			if s, ok := prop.Value.(string); ok {
				switch s {
				case "wordWrap", "WordWrap", "wrap":
					label.SetWrapMode(widgets.WrapWordWrap)
				case "elide", "ElideRight":
					label.SetWrapMode(widgets.WrapEllipsis)
				default:
					label.SetWrapMode(widgets.WrapNoWrap)
				}
			}
		case "horizontalAlignment", "hAlign":
			if s, ok := prop.Value.(string); ok {
				label.SetHAlign(s)
			}
		case "verticalAlignment", "vAlign":
			if s, ok := prop.Value.(string); ok {
				label.SetVAlign(s)
			}
		case "width":
			if f, ok := toFloat32(prop.Value); ok {
				finalWidth = f
			}
		case "height":
			if f, ok := toFloat32(prop.Value); ok {
				finalHeight = f
			}
		}
	}

	label.SetWidthDirect(finalWidth)
	label.SetHeightDirect(finalHeight)

	return label
}

func resolveAnchorTarget(target string, widget interface{}) interface{} {
	if target == "parent" {
		if pw, ok := widget.(interface{ Parent() interface{} }); ok {
			return pw.Parent()
		}
		// Try to get parent from BaseWidget
		if bw, ok := widget.(*widgets.Label); ok {
			return bw.Parent()
		}
	}
	return nil
}

// controlHeight is the active theme's control height with a fallback
// for themes without metrics (IntelliJ imports before defaults fill).
func controlHeight(fallback float32) float32 {
	if t := theme.DefaultTheme; t != nil && t.Metrics.ControlHeight > 0 {
		return t.Metrics.ControlHeight
	}
	return fallback
}

func buildButton(obj *Object) *widgets.Button {
	btn := widgets.NewButton()
	btn.SetWidth(100)
	btn.SetHeight(controlHeight(35))

	for _, prop := range obj.Properties {
		switch prop.Name {
		case "text":
			if s, ok := prop.Value.(string); ok {
				btn.SetText(s)
			}
		case "width":
			if f, ok := prop.Value.(float64); ok {
				// SetWidthDirect marks the size explicit so the button's
				// preferred-size floor (80px) yields to the QML width.
				btn.SetWidthDirect(float32(f))
			}
		case "height":
			if f, ok := prop.Value.(float64); ok {
				btn.SetHeightDirect(float32(f))
			}
		case "onClicked":
			if h, ok := prop.Value.(func()); ok {
				btn.SetOnClicked(h)
			}
		case "enabled":
			if b, ok := prop.Value.(bool); ok {
				btn.SetEnabled(b)
			}
		case "buttonType":
			if s, ok := prop.Value.(string); ok {
				btn.ButtonType = s
			}
		case "checkable":
			if b, ok := prop.Value.(bool); ok {
				btn.SetCheckable(b)
			}
		case "checked", "toggled":
			if b, ok := prop.Value.(bool); ok {
				btn.SetToggled(b)
			}
		}
	}

	return btn
}

func buildTextField(obj *Object) *widgets.TextField {
	tf := widgets.NewTextField()
	tf.SetWidth(200)
	tf.SetHeight(controlHeight(35))

	var anchors *layout.Anchors
	var layoutProps *layout.LayoutProps

	for _, prop := range obj.Properties {
		switch prop.Name {
		case "placeholderText", "placeholder":
			if s, ok := prop.Value.(string); ok {
				tf.SetPlaceholder(s)
			}
		case "width":
			if f, ok := prop.Value.(float64); ok {
				tf.SetWidth(float32(f))
			}
		case "height":
			if f, ok := prop.Value.(float64); ok {
				tf.SetHeight(float32(f))
			}
		case "onTextChanged":
			if h, ok := prop.Value.(func(string)); ok {
				tf.SetOnTextChanged(h)
			}
		case "enabled":
			if b, ok := prop.Value.(bool); ok {
				tf.SetEnabled(b)
			}
		default:
			// Handle anchor and layout properties
			if strings.HasPrefix(prop.Name, "anchors.") || strings.HasPrefix(prop.Name, "Layout.") {
				if anchors == nil {
					anchors = layout.NewAnchors()
				}
				anchors = parseAnchorsProp(prop.Name, prop.Value, anchors, tf)
			}
		}
	}

	if anchors != nil {
		setupAnchors(obj, anchors)
		tf.SetAnchors(anchors)
	}
	if layoutProps != nil {
		tf.SetLayoutProps(layoutProps)
	}

	return tf
}

func buildCheckBox(obj *Object) *widgets.CheckBox {
	cb := widgets.NewCheckBox()
	cb.SetWidth(150)
	cb.SetHeight(controlHeight(25))

	var anchors *layout.Anchors
	var layoutProps *layout.LayoutProps

	for _, prop := range obj.Properties {
		switch prop.Name {
		case "text":
			if s, ok := prop.Value.(string); ok {
				cb.SetText(s)
			}
		case "checked":
			if b, ok := prop.Value.(bool); ok {
				cb.SetChecked(b)
			}
		case "width":
			if f, ok := prop.Value.(float64); ok {
				cb.SetWidth(float32(f))
			}
		case "height":
			if f, ok := prop.Value.(float64); ok {
				cb.SetHeight(float32(f))
			}
		case "enabled":
			if b, ok := prop.Value.(bool); ok {
				cb.SetEnabled(b)
			}
		default:
			// Handle anchor and layout properties
			if strings.HasPrefix(prop.Name, "anchors.") || strings.HasPrefix(prop.Name, "Layout.") {
				if anchors == nil {
					anchors = layout.NewAnchors()
				}
				anchors = parseAnchorsProp(prop.Name, prop.Value, anchors, cb)
			}
		}
	}

	if anchors != nil {
		setupAnchors(obj, anchors)
		cb.SetAnchors(anchors)
	}
	if layoutProps != nil {
		cb.SetLayoutProps(layoutProps)
	}

	return cb
}

func buildSlider(obj *Object) *widgets.Slider {
	slider := widgets.NewSlider()
	slider.SetWidth(200)
	slider.SetHeight(controlHeight(30))

	for _, prop := range obj.Properties {
		switch prop.Name {
		case "value":
			if f, ok := prop.Value.(float64); ok {
				slider.SetValue(float32(f))
			}
		case "minimumValue", "min":
			if f, ok := prop.Value.(float64); ok {
				slider.SetMinimum(float32(f))
			}
		case "maximumValue", "max":
			if f, ok := prop.Value.(float64); ok {
				slider.SetMaximum(float32(f))
			}
		case "width":
			if f, ok := prop.Value.(float64); ok {
				slider.SetWidth(float32(f))
			}
		case "height":
			if f, ok := prop.Value.(float64); ok {
				slider.SetHeight(float32(f))
			}
		case "orientation":
			if s, ok := prop.Value.(string); ok {
				if s == "vertical" {
					slider.SetOrientation(1)
				} else if s == "horizontal" {
					slider.SetOrientation(0)
				}
			}
		case "enabled":
			if b, ok := prop.Value.(bool); ok {
				slider.SetEnabled(b)
			}
		}
	}

	return slider
}

func buildComboBox(obj *Object) *widgets.ComboBox {
	cb := widgets.NewComboBox()
	cb.SetWidth(200)
	cb.SetHeight(controlHeight(30))

	for _, prop := range obj.Properties {
		switch prop.Name {
		case "width":
			if f, ok := prop.Value.(float64); ok {
				cb.SetWidth(float32(f))
			}
		case "height":
			if f, ok := prop.Value.(float64); ok {
				cb.SetHeight(float32(f))
			}
		case "currentIndex":
			if f, ok := prop.Value.(float64); ok {
				cb.SetSelectedIndex(int(f))
			}
		case "enabled":
			if b, ok := prop.Value.(bool); ok {
				cb.SetEnabled(b)
			}
		case "options", "model":
			if arr, ok := prop.Value.([]interface{}); ok {
				opts := make([]string, len(arr))
				for i, v := range arr {
					if s, ok := v.(string); ok {
						opts[i] = s
					} else {
						opts[i] = fmt.Sprintf("%v", v)
					}
				}
				cb.SetOptions(opts)
			}
		case "expanded":
			if b, ok := prop.Value.(bool); ok {
				cb.SetExpanded(b)
			}
		}
	}

	return cb
}

func buildListView(obj *Object) *widgets.ListView {
	lv := widgets.NewListView()
	lv.SetWidth(200)
	lv.SetHeight(100)

	for _, prop := range obj.Properties {
		switch prop.Name {
		case "width":
			if f, ok := prop.Value.(float64); ok {
				lv.SetWidth(float32(f))
			}
		case "height":
			if f, ok := prop.Value.(float64); ok {
				lv.SetHeight(float32(f))
			}
		case "currentIndex":
			if f, ok := prop.Value.(float64); ok {
				lv.SetSelectedIndex(int(f))
			}
		}
	}

	return lv
}

func buildTabPanel(obj *Object) *widgets.TabPanel {
	tp := widgets.NewTabPanel()

	activeTab := 0
	for _, prop := range obj.Properties {
		switch prop.Name {
		case "width":
			if f, ok := prop.Value.(float64); ok {
				tp.SetWidth(float32(f))
			}
		case "height":
			if f, ok := prop.Value.(float64); ok {
				tp.SetHeight(float32(f))
			}
		case "activeTab":
			switch v := prop.Value.(type) {
			case float64:
				activeTab = int(v)
			case int:
				activeTab = v
			case int64:
				activeTab = int(v)
			}
		case "closable":
			if b, ok := prop.Value.(bool); ok {
				tp.SetClosable(b)
			}
		}
	}

	for _, child := range obj.Children {
		if child.Type != "Tab" && child.Type != "TabItem" { // accept Qt-style TabItem too
			continue
		}

		title := "Tab"
		var content meui.Widget
		tabID := ""
		tabIcon := ""
		tabPinned := false

		for _, prop := range child.Properties {
			switch prop.Name {
			case "title":
				if s, ok := prop.Value.(string); ok {
					title = s
				}
			case "id":
				if s, ok := prop.Value.(string); ok {
					tabID = s
				}
			case "icon":
				if s, ok := prop.Value.(string); ok {
					tabIcon = s
				}
			case "pinned":
				if b, ok := prop.Value.(bool); ok {
					tabPinned = b
				}
			}
		}

		if len(child.Children) > 0 {
			contentObj := child.Children[0]
			if childWidget, err := buildObject(contentObj); err == nil && childWidget != nil {
				content = childWidget
			}
		}

		if content != nil {
			tp.AddTabEx(tabID, title, tabIcon, content, tabPinned)
		}
	}

	// Set active tab AFTER tabs are added
	if activeTab >= 0 && activeTab < tp.TabCount() {
		tp.SetActiveTab(activeTab)
	}

	applyCommonProperties(obj, tp)

	return tp
}

func buildRadioButton(obj *Object) *widgets.RadioButton {
	rb := widgets.NewRadioButton()
	rb.SetWidth(150)
	rb.SetHeight(controlHeight(25))

	for _, prop := range obj.Properties {
		switch prop.Name {
		case "text":
			if s, ok := prop.Value.(string); ok {
				rb.SetText(s)
			}
		case "selected", "checked":
			if b, ok := prop.Value.(bool); ok {
				rb.SetSelected(b)
			}
		case "width":
			if f, ok := prop.Value.(float64); ok {
				rb.SetWidth(float32(f))
			}
		case "height":
			if f, ok := prop.Value.(float64); ok {
				rb.SetHeight(float32(f))
			}
		case "enabled":
			if b, ok := prop.Value.(bool); ok {
				rb.SetEnabled(b)
			}
		}
	}

	return rb
}

func buildTextArea(obj *Object) *widgets.TextArea {
	ta := widgets.NewTextArea()
	ta.SetWidth(300)
	ta.SetHeight(200)

	for _, prop := range obj.Properties {
		switch prop.Name {
		case "text":
			if s, ok := prop.Value.(string); ok {
				ta.SetText(s)
			}
		case "placeholderText", "placeholder":
			if s, ok := prop.Value.(string); ok {
				ta.SetPlaceholder(s)
			}
		case "width":
			if f, ok := prop.Value.(float64); ok {
				ta.SetWidth(float32(f))
			}
		case "height":
			if f, ok := prop.Value.(float64); ok {
				ta.SetHeight(float32(f))
			}
		case "enabled":
			if b, ok := prop.Value.(bool); ok {
				ta.SetEnabled(b)
			}
		}
	}

	return ta
}

func buildSpinBox(obj *Object) *widgets.SpinBox {
	sb := widgets.NewSpinBox()
	sb.SetWidth(80)
	sb.SetHeight(controlHeight(24))

	for _, prop := range obj.Properties {
		switch prop.Name {
		case "value":
			switch v := prop.Value.(type) {
			case float64:
				sb.SetValue(float32(v))
			case int:
				sb.SetValue(float32(v))
			}
		case "minimumValue", "min":
			switch v := prop.Value.(type) {
			case float64:
				sb.Minimum = float32(v)
			case int:
				sb.Minimum = float32(v)
			}
		case "maximumValue", "max":
			switch v := prop.Value.(type) {
			case float64:
				sb.Maximum = float32(v)
			case int:
				sb.Maximum = float32(v)
			}
		case "width":
			switch v := prop.Value.(type) {
			case float64:
				sb.SetWidth(float32(v))
			case int:
				sb.SetWidth(float32(v))
			}
		case "height":
			switch v := prop.Value.(type) {
			case float64:
				sb.SetHeight(float32(v))
			case int:
				sb.SetHeight(float32(v))
			}
		case "enabled":
			if b, ok := prop.Value.(bool); ok {
				sb.SetEnabled(b)
			}
		case "stepValue", "step", "stepSize":
			switch v := prop.Value.(type) {
			case float64:
				sb.SetStep(float32(v))
			case int:
				sb.SetStep(float32(v))
			}
		case "decimals", "precision":
			switch v := prop.Value.(type) {
			case float64:
				sb.SetDecimals(int(v))
			case int:
				sb.SetDecimals(v)
			}
		}
	}

	// Auto-set decimals based on step if not explicitly set and step is fractional
	if sb.Step != 0 && sb.Decimals == 0 && sb.Step != 1 && sb.Step != 10 && float32(int(sb.Step)) != sb.Step {
		stepStr := fmt.Sprintf("%.6f", sb.Step)
		if idx := strings.Index(stepStr, "."); idx != -1 {
			// Trim trailing zeros
			d := stepStr[idx+1:]
			for len(d) > 0 && d[len(d)-1] == '0' {
				d = d[:len(d)-1]
			}
			if len(d) > 0 {
				sb.SetDecimals(len(d))
			}
		}
	}

	return sb
}

func buildScrollBar(obj *Object) *widgets.ScrollBar {
	sb := widgets.NewScrollBar()
	sb.SetWidth(16)
	sb.SetHeight(100)

	for _, prop := range obj.Properties {
		switch prop.Name {
		case "orientation":
			if s, ok := prop.Value.(string); ok {
				if s == "horizontal" {
					sb.SetOrientation(widgets.Horizontal)
				}
			}
		case "value":
			if f, ok := prop.Value.(float64); ok {
				sb.SetValue(float32(f))
			}
		case "minimum", "min":
			if f, ok := prop.Value.(float64); ok {
				sb.SetMin(float32(f))
			}
		case "maximum", "max":
			if f, ok := prop.Value.(float64); ok {
				sb.SetMax(float32(f))
			}
		case "width":
			if f, ok := prop.Value.(float64); ok {
				sb.SetWidth(float32(f))
			}
		case "height":
			if f, ok := prop.Value.(float64); ok {
				sb.SetHeight(float32(f))
			}
		}
	}

	return sb
}

// buildRectangle builds a Rectangle (and, with a zero default size, an Item): the
// toolkit's Box. Rectangle is the most-used element of a web-looking scene, so it
// is also where the box-model properties live: radius, shadow, gradient, clip,
// opacity, padding, and the interaction a div gets from a listener.
func buildRectangle(obj *Object) *meui.Panel {
	return buildBox(obj, 100, 100, true)
}

// buildBox builds the box and its properties. Children are built by the caller
// when withChildren is false — the bindings path must build them through
// buildObjectWithBindings, because a child built first by the plain path lands in
// the Object→Widget map and the bindings path then returns it WITHOUT registering
// its id (every nested id silently stopped resolving).
func buildBox(obj *Object, defaultW, defaultH float32, withChildren bool) *meui.Panel {
	panel := meui.NewPanel()
	panel.SetWidth(defaultW)
	panel.SetHeight(defaultH)

	var borderColor graphics.Color
	var borderWidth float32
	var cornerRadius float32
	var anchors *layout.Anchors
	var layoutProps *layout.LayoutProps

	for _, prop := range obj.Properties {
		switch prop.Name {
		case "width":
			if f, ok := toFloat32(prop.Value); ok {
				panel.SetWidthDirect(f)
			}
		case "height":
			if f, ok := toFloat32(prop.Value); ok {
				panel.SetHeightDirect(f)
			}
		case "color":
			if s, ok := prop.Value.(string); ok {
				if _, isToken := theme.TokenColor(s); isToken {
					panel.SetBackgroundToken(s)
				} else {
					panel.SetBackground(graphics.Ink(parseColor(s)))
				}
			}
		// The parser keeps dotted names literal, so accept both the
		// Qt-style "border.color"/"border.width" and the flat aliases.
		case "borderColor", "border.color":
			if s, ok := prop.Value.(string); ok {
				borderColor = parseColor(s)
			}
		case "borderWidth", "border.width":
			if f, ok := toFloat32(prop.Value); ok {
				borderWidth = f
			}
		case "border":
			if m, ok := objectLiteral(prop.Value); ok {
				if c, ok := m["color"].(string); ok {
					borderColor = parseColor(c)
				}
				if w, ok := toFloat32(m["width"]); ok {
					borderWidth = w
				}
			}
		case "radius":
			if f, ok := toFloat32(prop.Value); ok {
				cornerRadius = f
			}
		case "shadow":
			if s, ok := prop.Value.(string); ok {
				panel.SetShadowToken(s)
			}
		case "gradient":
			if g, ok := parseGradient(prop.Value); ok {
				// The QML form is in UNIT space (0..1 of the box), because a scene
				// author does not know the size the layout will hand it.
				panel.SetGradientUnit(g)
			}
		case "sticky":
			if b, ok := prop.Value.(bool); ok {
				panel.SetSticky(b)
			}
		case "stickyInset":
			if f, ok := toFloat32(prop.Value); ok {
				panel.SetStickyInset(f)
			}
		case "blur", "backdropBlur":
			if b, ok := prop.Value.(bool); ok {
				panel.SetBackdropBlur(b)
			}
		case "clip":
			if b, ok := prop.Value.(bool); ok {
				panel.SetClipChildren(b)
			}
		case "opacity":
			if f, ok := toFloat32(prop.Value); ok {
				panel.SetOpacity(f)
			}
		case "padding", "inset":
			if f, ok := toFloat32(prop.Value); ok {
				panel.SetPadding(f)
			}
		case "clickable":
			if b, ok := prop.Value.(bool); ok {
				panel.SetClickable(b)
			}
		case "selected":
			if b, ok := prop.Value.(bool); ok {
				panel.SetSelected(b)
			}
		case "onClicked":
			if h, ok := prop.Value.(func()); ok {
				panel.SetOnClick(h)
			}
		case "cursor":
			if name, ok := prop.Value.(string); ok {
				panel.SetCursor(cursorByName(name))
			}
		case "hoverColor":
			if s, ok := prop.Value.(string); ok {
				panel.SetHoverInk(parseColor(s))
			}
		case "pressedColor":
			if s, ok := prop.Value.(string); ok {
				panel.SetPressedInk(parseColor(s))
			}
		default:
			// Handle anchor and layout properties separately
			if strings.HasPrefix(prop.Name, "anchors.") {
				if anchors == nil {
					anchors = layout.NewAnchors()
				}
				anchors = parseAnchorsProp(prop.Name, prop.Value, anchors, panel)
			} else if strings.HasPrefix(prop.Name, "Layout.") {
				if layoutProps == nil {
					layoutProps = &layout.LayoutProps{}
				}
				applyLayoutProperty(prop.Name, prop.Value, layoutProps)
			}
		}
	}

	if borderWidth > 0 {
		panel.SetBorderColor(borderColor)
		panel.SetBorderWidth(borderWidth)
	}

	if cornerRadius > 0 {
		panel.SetCornerRadius(cornerRadius)
	}

	if anchors != nil {
		setupAnchors(obj, anchors)
		panel.SetAnchors(anchors)
	}
	if layoutProps != nil {
		panel.SetLayoutProps(layoutProps)
	}

	// Register Object→Widget mapping BEFORE building children so they can resolve parent
	setObjectWidget(obj, panel)

	if withChildren {
		for _, child := range obj.Children {
			if childWidget, err := buildObject(child); err == nil && childWidget != nil {
				panel.AddChild(childWidget)
			}
		}
	}

	return panel
}

// buildScrollView builds a viewport onto ONE content widget — the web page's scroll
// container. It is the toolkit's ScrollPane under Qt's name, with the two things a
// PAGE needs and an IDE pane does not: no plate/edge of its own, and a content size
// the author declares (the content is taller than the viewport).
func buildScrollView(obj *Object) *widgets.ScrollPane {
	return buildScrollViewChildren(obj, true)
}

func buildScrollViewChildren(obj *Object, withChildren bool) *widgets.ScrollPane {
	sp := widgets.NewScrollPane()
	// A QML ScrollView is a page's viewport: its content is the page, and clickable
	// children inside it must be reachable by the hit test. The enterprise ScrollPane
	// (theme list, editors) keeps the pane-owned presses it drag-scrolls with.
	sp.SetHitTestContent(true)
	sp.SetWidth(320)
	sp.SetHeight(240)
	var anchors *layout.Anchors

	for _, prop := range obj.Properties {
		switch prop.Name {
		case "width":
			if f, ok := toFloat32(prop.Value); ok {
				sp.SetWidthDirect(f)
			}
		case "height":
			if f, ok := toFloat32(prop.Value); ok {
				sp.SetHeightDirect(f)
			}
		case "contentWidth":
			if f, ok := toFloat32(prop.Value); ok {
				sp.SetContentSize(f, sp.ContentHeight)
			}
		case "contentHeight":
			if f, ok := toFloat32(prop.Value); ok {
				sp.SetContentSize(sp.ContentWidth, f)
			}
		case "frameless", "frame":
			if b, ok := prop.Value.(bool); ok {
				sp.SetFrameless(b)
			}
		case "verticalScrollBarPolicy", "horizontalScrollBarPolicy":
			if s, ok := prop.Value.(string); ok {
				on := s == "always" || s == "asNeeded" || s == "ashneeded"
				if prop.Name == "verticalScrollBarPolicy" {
					sp.ShowVScroll = on
				} else {
					sp.ShowHScroll = on
				}
			}
		default:
			if strings.HasPrefix(prop.Name, "anchors.") {
				if anchors == nil {
					anchors = layout.NewAnchors()
				}
				anchors = parseAnchorsProp(prop.Name, prop.Value, anchors, sp)
			}
		}
	}
	if anchors != nil {
		setupAnchors(obj, anchors)
		sp.SetAnchors(anchors)
	}
	if withChildren {
		for _, child := range obj.Children {
			if childWidget, err := buildObject(child); err == nil && childWidget != nil {
				sp.AddChild(childWidget)
			}
		}
	}
	return sp
}

func buildScrollViewWithBindings(obj *Object, ctx *BindingContext, registry *BoundWidgetRegistry) *widgets.ScrollPane {
	sp := buildScrollViewChildren(obj, false)
	if obj.ID != "" {
		registry.Register(obj.ID, sp)
		ctx.RegisterObject(obj)
	}
	for _, child := range obj.Children {
		if childWidget, err := buildObjectWithBindings(child, ctx, registry); err == nil && childWidget != nil {
			sp.AddChild(childWidget)
		}
	}
	return sp
}

// buildMouseArea builds the input region. It paints nothing and owns no layout:
// its frame comes from anchors (the common case) or an explicit size.
func buildMouseArea(obj *Object) *meui.MouseArea {
	m := meui.NewMouseArea()
	m.SetFrameRectDirect(graphics.NewRect(0, 0, 0, 0))
	var anchors *layout.Anchors

	for _, prop := range obj.Properties {
		switch prop.Name {
		case "width":
			if f, ok := toFloat32(prop.Value); ok {
				m.SetWidthDirect(f)
			}
		case "height":
			if f, ok := toFloat32(prop.Value); ok {
				m.SetHeightDirect(f)
			}
		case "enabled":
			if b, ok := prop.Value.(bool); ok {
				m.SetEnabled(b)
			}
		case "visible":
			if b, ok := prop.Value.(bool); ok {
				m.SetVisible(b)
			}
		case "onClicked":
			if h, ok := prop.Value.(func()); ok {
				m.OnClicked = h
			}
		case "onPressed":
			if h, ok := prop.Value.(func()); ok {
				m.OnPressed = h
			}
		case "onReleased":
			if h, ok := prop.Value.(func()); ok {
				m.OnReleased = h
			}
		case "onDoubleClicked":
			if h, ok := prop.Value.(func()); ok {
				m.OnDoubleClicked = h
			}
		case "focusable", "activeFocusOnTab":
			if b, ok := prop.Value.(bool); ok {
				m.SetFocusable(b)
			}
		case "tabIndex":
			if f, ok := toFloat32(prop.Value); ok {
				m.SetTabIndex(int(f))
			}
		case "cursor":
			if name, ok := prop.Value.(string); ok {
				m.SetCursor(cursorByName(name))
			}
		default:
			if strings.HasPrefix(prop.Name, "anchors.") {
				if anchors == nil {
					anchors = layout.NewAnchors()
				}
				anchors = parseAnchorsProp(prop.Name, prop.Value, anchors, m)
			}
		}
	}
	if anchors != nil {
		setupAnchors(obj, anchors)
		m.SetAnchors(anchors)
	}
	return m
}

func buildMouseAreaWithBindings(obj *Object, ctx *BindingContext, registry *BoundWidgetRegistry) *meui.MouseArea {
	m := buildMouseArea(obj)
	if obj.ID != "" {
		registry.Register(obj.ID, m)
		ctx.RegisterObject(obj)
	}
	return m
}

func buildWindow(obj *Object) *meui.Window {
	win := meui.NewWindow()
	win.SetWidth(400)
	win.SetHeight(300)

	for _, prop := range obj.Properties {
		switch prop.Name {
		case "width":
			switch v := prop.Value.(type) {
			case float64:
				win.SetWidth(float32(v))
			case int:
				win.SetWidth(float32(v))
			case int64:
				win.SetWidth(float32(v))
			}
		case "height":
			switch v := prop.Value.(type) {
			case float64:
				win.SetHeight(float32(v))
			case int:
				win.SetHeight(float32(v))
			case int64:
				win.SetHeight(float32(v))
			}
		}
	}

	// Register Object→Widget mapping BEFORE building children
	setObjectWidget(obj, win)

	for _, child := range obj.Children {
		childWidget, err := buildObject(child)
		if err != nil {
			continue
		}
		if childWidget != nil {
			win.AddChild(childWidget)
		}
	}

	return win
}

func buildLayout(obj *Object) *meui.Panel {
	panel := meui.NewPanel()

	var spacing float32 = 8

	for _, prop := range obj.Properties {
		switch prop.Name {
		case "spacing":
			if f, ok := toFloat32(prop.Value); ok {
				spacing = f
			}
		case "width":
			if f, ok := toFloat32(prop.Value); ok {
				panel.SetWidthDirect(f)
			}
		case "height":
			if f, ok := toFloat32(prop.Value); ok {
				panel.SetHeightDirect(f)
			}
		case "padding":
			if f, ok := toFloat32(prop.Value); ok {
				panel.SetPadding(f)
			}
		case "margins":
			if f, ok := toFloat32(prop.Value); ok {
				panel.SetPadding(f)
			}
		}
	}

	switch obj.Type {
	case "Column", "VStack":
		vstack := layout.NewVStack()
		vstack.Spacing = spacing
		panel.SetLayout(vstack)
	case "Row", "HStack":
		hstack := layout.NewHStack()
		hstack.Spacing = spacing
		panel.SetLayout(hstack)
	case "Grid":
		grid := layout.NewGrid(3)
		grid.Spacing = spacing
		panel.SetLayout(grid)
	case "Flex":
		flex := layout.NewFlex()
		flex.Spacing = spacing
		panel.SetLayout(flex)
	}

	// Apply anchors and common properties after layout is set
	applyCommonProperties(obj, panel)

	// Register Object→Widget mapping BEFORE building children
	setObjectWidget(obj, panel)

	for _, child := range obj.Children {
		childWidget, err := buildObject(child)
		if err != nil {
			continue
		}
		if childWidget != nil {
			panel.AddChild(childWidget)
		}
	}

	return panel
}

func buildPanel(obj *Object) *meui.Panel {
	panel := meui.NewPanel()
	panel.SetWidth(400)
	panel.SetHeight(300)

	for _, prop := range obj.Properties {
		switch prop.Name {
		case "width":
			if f, ok := prop.Value.(float64); ok {
				panel.SetWidth(float32(f))
			}
		case "height":
			if f, ok := prop.Value.(float64); ok {
				panel.SetHeight(float32(f))
			}
		case "padding":
			if f, ok := prop.Value.(float64); ok {
				panel.SetPadding(float32(f))
			}
		}
	}

	// Register Object→Widget mapping BEFORE building children
	setObjectWidget(obj, panel)

	for _, child := range obj.Children {
		childWidget, err := buildObject(child)
		if err != nil {
			continue
		}
		if childWidget != nil {
			panel.AddChild(childWidget)
		}
	}

	return panel
}

func buildLabelWithBindings(obj *Object, ctx *BindingContext, registry *BoundWidgetRegistry) *widgets.Label {
	label := buildLabel(obj)
	applyCommonProperties(obj, label)
	applyWidgetProperties(obj, label)
	if obj.ID != "" {
		registry.Register(obj.ID, label)
		ctx.RegisterObject(obj)
	}
	processBindings(obj, ctx, label)
	return label
}

func buildButtonWithBindings(obj *Object, ctx *BindingContext, registry *BoundWidgetRegistry) *widgets.Button {
	btn := buildButton(obj)
	if obj.ID != "" {
		registry.Register(obj.ID, btn)
		ctx.RegisterObject(obj)
	}
	processBindings(obj, ctx, btn)
	return btn
}

func buildTextFieldWithBindings(obj *Object, ctx *BindingContext, registry *BoundWidgetRegistry) *widgets.TextField {
	tf := buildTextField(obj)
	if obj.ID != "" {
		registry.Register(obj.ID, tf)
		ctx.RegisterObject(obj)
	}
	processBindings(obj, ctx, tf)
	return tf
}

func buildCheckBoxWithBindings(obj *Object, ctx *BindingContext, registry *BoundWidgetRegistry) *widgets.CheckBox {
	cb := buildCheckBox(obj)
	if obj.ID != "" {
		registry.Register(obj.ID, cb)
		ctx.RegisterObject(obj)
	}
	processBindings(obj, ctx, cb)
	return cb
}

func buildSliderWithBindings(obj *Object, ctx *BindingContext, registry *BoundWidgetRegistry) *widgets.Slider {
	slider := buildSlider(obj)
	if obj.ID != "" {
		registry.Register(obj.ID, slider)
		ctx.RegisterObject(obj)
	}
	processBindings(obj, ctx, slider)
	return slider
}

func buildComboBoxWithBindings(obj *Object, ctx *BindingContext, registry *BoundWidgetRegistry) *widgets.ComboBox {
	cb := buildComboBox(obj)
	if obj.ID != "" {
		registry.Register(obj.ID, cb)
		ctx.RegisterObject(obj)
	}
	processBindings(obj, ctx, cb)
	return cb
}

func buildTabPanelWithBindings(obj *Object, ctx *BindingContext, registry *BoundWidgetRegistry) *widgets.TabPanel {
	tp := widgets.NewTabPanel()

	activeTab := 0
	for _, prop := range obj.Properties {
		switch prop.Name {
		case "width":
			if f, ok := prop.Value.(float64); ok {
				tp.SetWidth(float32(f))
			}
		case "height":
			if f, ok := prop.Value.(float64); ok {
				tp.SetHeight(float32(f))
			}
		case "activeTab":
			switch v := prop.Value.(type) {
			case float64:
				activeTab = int(v)
			case int:
				activeTab = v
			case int64:
				activeTab = int(v)
			}
		case "closable":
			if b, ok := prop.Value.(bool); ok {
				tp.SetClosable(b)
			}
		}
	}

	for _, child := range obj.Children {
		if child.Type != "Tab" && child.Type != "TabItem" { // accept Qt-style TabItem too
			continue
		}

		title := "Tab"
		var content meui.Widget
		tabID := ""
		tabIcon := ""
		tabPinned := false

		for _, prop := range child.Properties {
			switch prop.Name {
			case "title":
				if s, ok := prop.Value.(string); ok {
					title = s
				}
			case "id":
				if s, ok := prop.Value.(string); ok {
					tabID = s
				}
			case "icon":
				if s, ok := prop.Value.(string); ok {
					tabIcon = s
				}
			case "pinned":
				if b, ok := prop.Value.(bool); ok {
					tabPinned = b
				}
			}
		}

		if len(child.Children) > 0 {
			contentObj := child.Children[0]
			if childWidget, err := buildObjectWithBindings(contentObj, ctx, registry); err == nil && childWidget != nil {
				content = childWidget
			}
		}

		if content != nil {
			tp.AddTabEx(tabID, title, tabIcon, content, tabPinned)
		}
	}

	// Set active tab AFTER tabs are added
	if activeTab >= 0 && activeTab < tp.TabCount() {
		tp.SetActiveTab(activeTab)
	}

	// Apply common properties including layout and anchors
	applyCommonProperties(obj, tp)
	if obj.ID != "" {
		registry.Register(obj.ID, tp)
		ctx.RegisterObject(obj)
	}

	return tp
}

func buildListViewWithBindings(obj *Object, ctx *BindingContext, registry *BoundWidgetRegistry) *widgets.ListView {
	lv := buildListView(obj)
	if obj.ID != "" {
		registry.Register(obj.ID, lv)
		ctx.RegisterObject(obj)
	}
	processBindings(obj, ctx, lv)
	return lv
}

func buildRadioButtonWithBindings(obj *Object, ctx *BindingContext, registry *BoundWidgetRegistry) *widgets.RadioButton {
	rb := buildRadioButton(obj)
	if obj.ID != "" {
		registry.Register(obj.ID, rb)
		ctx.RegisterObject(obj)
	}
	processBindings(obj, ctx, rb)
	return rb
}

func buildTextAreaWithBindings(obj *Object, ctx *BindingContext, registry *BoundWidgetRegistry) *widgets.TextArea {
	ta := buildTextArea(obj)
	if obj.ID != "" {
		registry.Register(obj.ID, ta)
		ctx.RegisterObject(obj)
	}
	processBindings(obj, ctx, ta)
	return ta
}

func buildSpinBoxWithBindings(obj *Object, ctx *BindingContext, registry *BoundWidgetRegistry) *widgets.SpinBox {
	sb := buildSpinBox(obj)
	if obj.ID != "" {
		registry.Register(obj.ID, sb)
		ctx.RegisterObject(obj)
	}
	processBindings(obj, ctx, sb)
	return sb
}

func buildScrollBarWithBindings(obj *Object, ctx *BindingContext, registry *BoundWidgetRegistry) *widgets.ScrollBar {
	sb := buildScrollBar(obj)
	if obj.ID != "" {
		registry.Register(obj.ID, sb)
		ctx.RegisterObject(obj)
	}
	processBindings(obj, ctx, sb)
	return sb
}

func buildRectangleWithBindings(obj *Object, ctx *BindingContext, registry *BoundWidgetRegistry) *meui.Panel {
	return buildBoxWithBindings(obj, ctx, registry, 100, 100)
}

func buildBoxWithBindings(obj *Object, ctx *BindingContext, registry *BoundWidgetRegistry, defaultW, defaultH float32) *meui.Panel {
	panel := buildBox(obj, defaultW, defaultH, false)
	if obj.ID != "" {
		registry.Register(obj.ID, panel)
		ctx.RegisterObject(obj)
	}
	for _, child := range obj.Children {
		if childWidget, err := buildObjectWithBindings(child, ctx, registry); err == nil && childWidget != nil {
			panel.AddChild(childWidget)
		}
	}
	return panel
}

func buildWindowWithBindings(obj *Object, ctx *BindingContext, registry *BoundWidgetRegistry) *meui.Window {
	win := meui.NewWindow()
	win.SetWidthDirect(400)
	win.SetHeightDirect(300)

	for _, prop := range obj.Properties {
		switch prop.Name {
		case "width":
			switch v := prop.Value.(type) {
			case float64:
				win.SetWidthDirect(float32(v))
			case int:
				win.SetWidthDirect(float32(v))
			case int64:
				win.SetWidthDirect(float32(v))
			}
		case "height":
			switch v := prop.Value.(type) {
			case float64:
				win.SetHeightDirect(float32(v))
			case int:
				win.SetHeightDirect(float32(v))
			case int64:
				win.SetHeightDirect(float32(v))
			}
		case "color":
			if s, ok := prop.Value.(string); ok {
				win.SetBackground(graphics.Ink(parseColor(s)))
			}
		}
	}

	for _, child := range obj.Children {
		if childWidget, err := buildObjectWithBindings(child, ctx, registry); err == nil && childWidget != nil {
			childWidget.SetParent(win)
			win.AddChild(childWidget)
		}
	}

	// For Windows without layout manager, apply anchor layout
	if len(win.Children()) > 0 && win.Layout() == nil {
		winFrame := win.FrameRect()
		anchorLayout := layout.NewAnchorLayout()
		anchorLayout.Layout(win, graphics.NewRect(0, 0, winFrame.Width, winFrame.Height))
	}

	if obj.ID != "" {
		registry.Register(obj.ID, win)
		ctx.RegisterObject(obj)
	}
	ctx.ApplyBindings()
	registry.ApplyBindings(ctx)
	return win
}

func buildLayoutWithBindings(obj *Object, ctx *BindingContext, registry *BoundWidgetRegistry) *meui.Panel {
	panel := meui.NewPanel()

	var spacing float32 = 8
	var direction string = "row"
	var justify string = "start"
	var align string = "stretch"
	var wrap bool = false
	var columns int = 3

	for _, prop := range obj.Properties {
		switch prop.Name {
		case "spacing":
			if f, ok := toFloat32(prop.Value); ok {
				spacing = f
			}
		case "columns":
			if f, ok := toFloat32(prop.Value); ok {
				columns = int(f)
			}
		case "direction":
			if s, ok := prop.Value.(string); ok {
				direction = s
			}
		case "justifyContent":
			if s, ok := prop.Value.(string); ok {
				justify = s
			}
		case "alignItems":
			if s, ok := prop.Value.(string); ok {
				align = s
			}
		case "wrap":
			if b, ok := prop.Value.(bool); ok {
				wrap = b
			}
		case "width":
			if f, ok := toFloat32(prop.Value); ok {
				panel.SetWidthDirect(f)
			}
		case "height":
			if f, ok := toFloat32(prop.Value); ok {
				panel.SetHeightDirect(f)
			}
		case "padding":
			if f, ok := toFloat32(prop.Value); ok {
				panel.SetPadding(f)
			}
		case "margins":
			if f, ok := toFloat32(prop.Value); ok {
				panel.SetPadding(f)
			}
		}
	}

	switch obj.Type {
	case "Column", "VStack":
		vstack := layout.NewVStack()
		vstack.Spacing = spacing
		panel.SetLayout(vstack)
	case "Row", "HStack":
		hstack := layout.NewHStack()
		hstack.Spacing = spacing
		panel.SetLayout(hstack)
	case "Grid":
		grid := layout.NewGrid(columns)
		grid.Spacing = spacing
		panel.SetLayout(grid)
	case "Flex":
		flex := layout.NewFlex()
		flex.Spacing = spacing
		flex.Direction = layout.FlexDirection(direction)
		flex.Justify = layout.JustifyContent(justify)
		flex.Align = layout.AlignItems(align)
		flex.Wrap = wrap
		panel.SetLayout(flex)
	}

	// Apply anchors and common properties after layout is set
	applyCommonProperties(obj, panel)

	if obj.ID != "" {
		registry.Register(obj.ID, panel)
		ctx.RegisterObject(obj)
	}
	for _, child := range obj.Children {
		if childWidget, err := buildObjectWithBindings(child, ctx, registry); err == nil && childWidget != nil {
			if _, isCM := childWidget.(*widgets.ContextMenu); isCM {
				continue
			}
			if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
				fmt.Printf("[layoutAddChild] parent=%T child=%T (%p)\n", panel, childWidget, childWidget)
			}
			panel.AddChild(childWidget)
		}
	}
	return panel
}

func buildPanelWithBindings(obj *Object, ctx *BindingContext, registry *BoundWidgetRegistry) *meui.Panel {
	panel := meui.NewPanel()
	panel.SetWidth(400)
	panel.SetHeight(300)

	if obj.ID != "" {
		ctx.RegisterObject(obj)
	}
	for _, child := range obj.Children {
		if childWidget, err := buildObjectWithBindings(child, ctx, registry); err == nil && childWidget != nil {
			if _, isCM := childWidget.(*widgets.ContextMenu); isCM {
				continue
			}
			panel.AddChild(childWidget)
		}
	}
	return panel
}

func processBindings(obj *Object, ctx *BindingContext, widget BoundWidget) {
	for _, prop := range obj.Properties {
		if str, ok := prop.Value.(string); ok {
			if isBinding(str) {
				resolved, isBinding2 := ctx.ResolveProperty(obj, prop.Name, prop.Value)
				if isBinding2 && resolved == nil {
					widget.SetBoundProperty(prop.Name, nil)
				}
			}
		}
	}
}

func isBinding(s string) bool {
	parts := strings.Split(s, ".")
	return len(parts) == 2 && parts[0] != "" && parts[1] != ""
}

// parseGradient reads the Qt-ish gradient form
//
//	gradient: { from: "#B4DCEC", to: "#4176E6", angle: 180, via: "#DDEEF7" }
//
// Geometry is UNIT space (0..1 of the box) so a scene does not have to know the
// size the layout will give it; Panel resolves it against the final rect. The
// angle follows CSS: 180 is top-to-bottom, 90 is left-to-right.
func parseGradient(v interface{}) (graphics.Gradient, bool) {
	m, ok := objectLiteral(v)
	if !ok {
		return graphics.Gradient{}, false
	}
	from, _ := m["from"].(string)
	to, _ := m["to"].(string)
	if from == "" || to == "" {
		return graphics.Gradient{}, false
	}
	angle := float32(180)
	if f, ok := toFloat32(m["angle"]); ok {
		angle = f
	}
	rad := float64(angle) * math.Pi / 180
	dx := float32(math.Sin(rad))
	dy := float32(-math.Cos(rad))
	g := graphics.Gradient{
		Kind: graphics.GradientLinear,
		Stops: []graphics.GradientStop{
			{Offset: 0, Color: parseColor(from)},
			{Offset: 1, Color: parseColor(to)},
		},
		Start: graphics.Point{X: 0.5 - dx*0.5, Y: 0.5 - dy*0.5},
		End:   graphics.Point{X: 0.5 + dx*0.5, Y: 0.5 + dy*0.5},
	}
	if via, ok := m["via"].(string); ok && via != "" {
		g.Stops = []graphics.GradientStop{
			{Offset: 0, Color: parseColor(from)},
			{Offset: 0.5, Color: parseColor(via)},
			{Offset: 1, Color: parseColor(to)},
		}
	}
	return g, true
}

// objectLiteral reads a `{ k: v, ... }` property value. The parser keeps such a
// value as its SOURCE TEXT (it does not build a map for every property), so a
// handler that only accepted map[string]interface{} silently never fired — which is
// why `border: { color: …, width: … }` and `gradient: { … }` painted nothing.
func objectLiteral(v interface{}) (map[string]interface{}, bool) {
	switch t := v.(type) {
	case map[string]interface{}:
		return t, true
	case string:
		s := strings.TrimSpace(t)
		if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
			return nil, false
		}
		return parseObjectToMap(s), true
	}
	return nil, false
}

// cursorByName maps QML's cursor names (the CSS set, plus Qt's) onto the toolkit's
// cursor vocabulary. The names are that vocabulary's, so the backend mapping stays
// the one place a platform cursor is chosen.
func cursorByName(name string) *platform.Cursor {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "arrow", "default", "auto":
		return platform.ArrowCursor()
	case "pointing", "pointer", "hand", "click":
		return platform.PointingCursor()
	case "text", "ibeam":
		return platform.TextCursor()
	case "move", "grabbing":
		return platform.MoveCursor()
	case "crosshair", "cross":
		return platform.CrosshairCursor()
	case "wait", "progress":
		return platform.WaitCursor()
	case "notallowed", "not-allowed", "forbidden":
		return platform.NotAllowedCursor()
	case "resizeh", "ew-resize", "col-resize", "sizewe":
		return platform.ResizeHorizontalCursor()
	case "resizev", "ns-resize", "row-resize", "sizens":
		return platform.ResizeVerticalCursor()
	case "resizeld", "nwse-resize", "nw-resize", "se-resize":
		return platform.ResizeLeftDiagonalCursor()
	case "resizerd", "nesw-resize", "ne-resize", "sw-resize":
		return platform.ResizeRightDiagonalCursor()
	}
	return nil
}

// hexColor parses exactly eight hex digits as AARRGGBB.
func hexColor(h string) (graphics.Color, bool) {
	var v uint32
	for i := 0; i < len(h); i++ {
		c := h[i]
		var d uint32
		switch {
		case c >= '0' && c <= '9':
			d = uint32(c - '0')
		case c >= 'a' && c <= 'f':
			d = uint32(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = uint32(c-'A') + 10
		default:
			return 0, false
		}
		v = v<<4 | d
	}
	return graphics.Color(v), true
}

func parseColor(s string) graphics.Color {
	// Theme role tokens first: `color: "secondary"` follows the active
	// theme instead of hardcoding a hex value. Resolution happens at
	// parse time (themes load before QML), so presets apply; a runtime
	// theme switch re-renders theme-driven widgets but not stale
	// parse-time label colors.
	if c, ok := theme.TokenColor(s); ok {
		return c
	}
	// EIGHT digits are the seam's own layout, AARRGGBB. ColorDecode reads them as
	// something else and returns alpha 0 (#CCFFFFFF came out 0x00FFFFCC), so every
	// translucent colour a scene wrote — "#80FFFFFF", "#CC14243C" — was invisible: the
	// frosted glass in the parity scene simply never drew.
	if len(s) == 9 && s[0] == '#' {
		if v, ok := hexColor(s[1:]); ok {
			return v
		}
	}
	// Use the reference ColorDecode, which supports hex, rgb(), rgba(), hsl() and named colors
	if c, err := graphics.ColorDecode(s); err == nil {
		return c
	}
	// Fallback for some common aliases
	s = strings.ToLower(s)
	colorMap := map[string]graphics.Color{
		"lightgray": graphics.Color(0xFFAAAAAA),
		"darkgray":  graphics.Color(0xFF666666),
	}
	if c, ok := colorMap[s]; ok {
		return c
	}
	return graphics.Color(0)
}

var builtinsInitialized bool

// initializeBuiltinWidgets registers all built-in MeUi widgets
func initializeBuiltinWidgets() {
	if builtinsInitialized {
		return
	}
	builtinsInitialized = true

	// Register built-in widgets with their specific properties
	RegisterWidget("Label", func() meui.Widget {
		return widgets.NewLabel()
	}, []string{"text", "richText", "color", "font.family", "font.size", "font.bold", "horizontalAlignment", "verticalAlignment"})

	RegisterWidget("Text", func() meui.Widget {
		return widgets.NewLabel()
	}, []string{"text", "color", "font.family", "font.size", "font.bold", "horizontalAlignment", "verticalAlignment"})

	RegisterWidget("Button", func() meui.Widget {
		return widgets.NewButton()
	}, []string{"text", "onClicked", "buttonType", "icon", "groupID", "selected", "checkable", "checked", "toggled"})

	RegisterWidget("TextField", func() meui.Widget {
		return widgets.NewTextField()
	}, []string{"text", "placeholderText", "placeholder", "onTextChanged", "validationState", "leadingIcon", "trailingIcon", "echoChar", "passwordChar"})

	RegisterWidget("CheckBox", func() meui.Widget {
		return widgets.NewCheckBox()
	}, []string{"text", "checked"})

	RegisterWidget("RadioButton", func() meui.Widget {
		return widgets.NewRadioButton()
	}, []string{"text", "checked", "selected", "groupID"})

	RegisterWidget("Slider", func() meui.Widget {
		return widgets.NewSlider()
	}, []string{"value", "minimumValue", "maximumValue", "min", "max", "orientation"})

	// Image DISPLAYS a graphics.Image; the pixels arrive from the app (see widgets.Image), so only the
	// viewing options are QML properties. `checkerboard` is a bool because the property binder handles
	// primitives; the FIT is set from Go, where it is an enum rather than a string that could be
	// misspelled.
	RegisterWidget("Image", func() meui.Widget {
		return widgets.NewImage()
	}, []string{"checkerboard"})

	RegisterWidget("ComboBox", func() meui.Widget {
		return widgets.NewComboBox()
	}, []string{"currentIndex", "model", "editable", "placeholderText", "placeholder"})

	RegisterWidget("TextArea", func() meui.Widget {
		return widgets.NewTextArea()
	}, []string{"text", "onTextChanged", "lineNumbers", "wordWrap", "readOnly", "maxLength"})

	RegisterWidget("ScrollBar", func() meui.Widget {
		return widgets.NewScrollBar()
	}, []string{"value", "minimum", "maximum", "orientation"})

	RegisterWidget("SpinBox", func() meui.Widget {
		return widgets.NewSpinBox()
	}, []string{"value", "minimum", "maximum", "stepSize"})

	RegisterWidget("ProgressBar", func() meui.Widget {
		return widgets.NewProgressBar()
	}, []string{"value", "minimum", "maximum", "indeterminate", "min", "max"})

	RegisterWidget("ListView", func() meui.Widget {
		return widgets.NewListView()
	}, []string{"model", "currentIndex", "onCurrentIndexChanged", "checkable", "alternatingRows", "hideBorder", "border.width"})

	RegisterWidget("Switch", func() meui.Widget {
		return widgets.NewSwitch()
	}, []string{"checked", "enabled"})

	RegisterWidget("VirtualList", func() meui.Widget {
		return widgets.NewVirtualList()
	}, []string{"rowHeight", "rowCount", "topInset"})

	RegisterWidget("GroupBox", func() meui.Widget {
		return widgets.NewGroupBox()
	}, []string{"title", "width", "height", "collapsible", "collapsed"})

	RegisterWidget("StatusBar", func() meui.Widget {
		return widgets.NewStatusBar()
	}, []string{"text", "sections"})

	RegisterWidget("ToolBar", func() meui.Widget {
		return widgets.NewToolBar()
	}, []string{"buttons", "spacing", "toolsets", "wrap"})

	RegisterWidget("Calendar", func() meui.Widget {
		return widgets.NewCalendar()
	}, []string{"year", "month", "day", "selectedDay"})

	RegisterWidget("ColorPicker", func() meui.Widget {
		return widgets.NewColorPicker()
	}, []string{"red", "green", "blue", "hue"})

	RegisterWidget("DatePicker", func() meui.Widget {
		return widgets.NewDatePicker()
	}, []string{"text", "placeholder", "year", "month", "day"})

	RegisterWidget("ColorField", func() meui.Widget {
		return widgets.NewColorField()
	}, []string{"text", "color", "placeholder"})

	RegisterWidget("TimePicker", func() meui.Widget {
		return widgets.NewTimePicker()
	}, []string{"text", "placeholder", "hour", "minute", "second", "style"})

	RegisterWidget("MenuBar", func() meui.Widget {
		return widgets.NewMenuBar()
	}, []string{"menus"})

	RegisterWidget("Dialog", func() meui.Widget {
		return widgets.NewDialog()
	}, []string{"title", "modal"})

	RegisterWidget("SplitView", func() meui.Widget {
		return widgets.NewSplitView()
	}, []string{"orientation", "splitPosition", "position", "handleSize"})

	RegisterWidget("ToolTip", func() meui.Widget {
		return widgets.NewTooltip()
	}, []string{"text", "delay"})

	RegisterWidget("ContextMenu", func() meui.Widget {
		return widgets.NewContextMenu()
	}, []string{"items"})

	RegisterWidget("Tree", func() meui.Widget {
		return widgets.NewTree()
	}, []string{"model", "alternatingRows", "itemHeight", "hideBorder", "border.width"})

	RegisterWidget("Table", func() meui.Widget {
		return widgets.NewTable()
	}, []string{"columns", "rows", "alternatingRows", "showGrid", "rowSelection", "colSelection", "sortable", "resizable", "reorderable", "multiSelect", "editable"})

	RegisterWidget("DataTable", func() meui.Widget {
		return widgets.NewDataTable()
	}, []string{"columns", "rows", "alternatingRows", "showGrid", "showHeader", "showRowNumbers"})

	RegisterWidget("ScrollPane", func() meui.Widget {
		return widgets.NewScrollPane()
	}, []string{"horizontalScrollBarPolicy", "verticalScrollBarPolicy"})

	// TabStrip takes no QML properties of its own: its items are built in Go and set with
	// SetItems, because they carry an ID and a live label that QML has no way to express.
	// id, width, height and Layout.* come from the base properties.
	RegisterWidget("TabStrip", func() meui.Widget {
		return widgets.NewTabStrip()
	}, []string{})

	// Collapsible is EXEMPT from the qml/handlers.go text switch: that switch is for a
	// property literally named "text", and this widget has none — it has label, detail and
	// trailing, which reach SetBoundProperty through the registered property list.
	RegisterWidget("Collapsible", func() meui.Widget {
		return widgets.NewCollapsible()
	}, []string{"label", "detail", "trailing", "collapsed"})

	RegisterWidget("NavigationRail", func() meui.Widget {
		return widgets.NewNavigationRail()
	}, []string{"selected"})

	RegisterWidget("Snackbar", func() meui.Widget {
		return widgets.NewSnackbar()
	}, []string{"text"})

	RegisterWidget("TopAppBar", func() meui.Widget {
		return widgets.NewTopAppBar()
	}, []string{"title"})

	// NavigationBar's destinations are APP DATA, not markup: an app holds one list and
	// chooses this control or a TabStrip by form factor. So the QML side carries only
	// the selection, and the items are set from Go.
	RegisterWidget("NavigationBar", func() meui.Widget {
		return widgets.NewNavigationBar()
	}, []string{"selected"})

	// InsetSpacer reserves a strip for an OS inset (the IME). Its height is APP DATA — the
	// shell sets it from Window.Insets().Keyboard — so the QML side carries no properties.
	RegisterWidget("InsetSpacer", func() meui.Widget {
		return widgets.NewInsetSpacer()
	}, []string{})

	RegisterWidget("BottomSheet", func() meui.Widget {
		return widgets.NewBottomSheet()
	}, []string{"title", "modal", "height"})

	// ActionSheet's choices are APP DATA, like NavigationBar's destinations: the app holds the
	// list and the widget reports which one was chosen.
	RegisterWidget("ActionSheet", func() meui.Widget {
		return widgets.NewActionSheet()
	}, []string{"title", "text", "cancel"})

	// Waveform's peaks are APP DATA (computed from a track), like an AdaptiveTable's rows.
	// Histogram's BINS are app data (a document's tonal distribution), like the Waveform's peaks.
	RegisterWidget("Histogram", func() meui.Widget {
		return widgets.NewHistogram()
	}, []string{"clipGuard"})

	RegisterWidget("Waveform", func() meui.Widget {
		return widgets.NewWaveform()
	}, []string{})

	// AdaptiveTable's columns and rows are APP DATA, like a NavigationBar's destinations; the QML
	// carries the widget and the Go side supplies the table.
	RegisterWidget("AdaptiveTable", func() meui.Widget {
		return widgets.NewAdaptiveTable()
	}, []string{})

	// PullToRefresh wraps a scrollable surface; its content is the QML child.
	RegisterWidget("PullToRefresh", func() meui.Widget {
		return widgets.NewPullToRefresh()
	}, []string{})

	// SwipeActionRow's actions are APP DATA, like a NavigationBar's destinations; the QML carries
	// the content row and the Go side supplies the actions behind it.
	RegisterWidget("SwipeActionRow", func() meui.Widget {
		return widgets.NewSwipeActionRow()
	}, []string{})

	RegisterWidget("ListItem", func() meui.Widget {
		return widgets.NewListItem()
	}, []string{"text", "secondary", "leading", "trailing", "selected", "variant"})

	RegisterWidget("SearchView", func() meui.Widget {
		return widgets.NewSearchView()
	}, []string{"placeholderText", "query"})

	// AlertDialog is the adaptive one. The QML type `Dialog` is the LEGACY draggable popup, which
	// the render-parity corpus renders, so the adaptive dialog gets its own name rather than
	// moving a committed baseline.
	RegisterWidget("AlertDialog", func() meui.Widget {
		return widgets.NewAlertDialog()
	}, []string{"title", "text"})

	RegisterWidget("CodeEditor", func() meui.Widget {
		return widgets.NewCodeEditor()
	}, []string{"text", "filePath", "language", "readOnly", "fontSize", "wordWrap", "showLineNumbers"})

	RegisterWidget("MdiWindow", func() meui.Widget {
		return widgets.NewMdiWindow()
	}, []string{"title", "closable", "icon", "x", "y", "minimized", "maximized", "resizable"})

	RegisterWidget("MarkdownView", func() meui.Widget {
		return widgets.NewMarkdownView()
	}, []string{"text", "filePath"})

	RegisterWidget("MarkdownEditor", func() meui.Widget {
		return widgets.NewMarkdownEditor()
	}, []string{"text", "readOnly", "mode", "splitRatio"})

	RegisterWidget("MdiContainer", func() meui.Widget {
		return widgets.NewMdiContainer()
	}, []string{"docksFullHeight", "showTaskbar"})

	RegisterWidget("DockPanel", func() meui.Widget {
		return widgets.NewDockPanel()
	}, []string{"leftSize", "rightSize", "topSize", "bottomSize"})

	RegisterWidget("MessageBox", func() meui.Widget {
		return widgets.NewMessageBox("", "", nil)
	}, []string{"title", "message", "buttons", "onDone"})

	RegisterWidget("InputDialog", func() meui.Widget {
		return widgets.NewInputDialog("", "")
	}, []string{"title", "label", "onDone"})

	RegisterWidget("StackPanel", func() meui.Widget {
		return widgets.NewStackPanel("vertical")
	}, []string{"orientation"})

	RegisterWidget("Popup", func() meui.Widget {
		return widgets.NewModalPopupPanel("", "", nil)
	}, []string{"title", "heading"})

	// Note: Window, TabPanel, Rectangle, and layout widgets (VStack, HStack, Grid, Flex)
	// are handled specially in loader.go's buildObject()

	// Register flow wrappers for reflection-based property dispatch
	SetFlowWrapper("Label", func() interface{} { return &flow.Label{} })
	SetFlowWrapper("Text", func() interface{} { return &flow.Label{} })
	SetFlowWrapper("Button", func() interface{} { return &flow.Button{} })
	SetFlowWrapper("TextField", func() interface{} { return &flow.TextField{} })
	SetFlowWrapper("CheckBox", func() interface{} { return &flow.CheckBox{} })
	SetFlowWrapper("RadioButton", func() interface{} { return &flow.RadioButton{} })
	SetFlowWrapper("Slider", func() interface{} { return &flow.Slider{} })
	SetFlowWrapper("ComboBox", func() interface{} { return &flow.ComboBox{} })
	SetFlowWrapper("TextArea", func() interface{} { return &flow.TextArea{} })
	SetFlowWrapper("ScrollBar", func() interface{} { return &flow.ScrollBar{} })
	SetFlowWrapper("SpinBox", func() interface{} { return &flow.SpinBox{} })
	SetFlowWrapper("ProgressBar", func() interface{} { return &flow.ProgressBar{} })
	SetFlowWrapper("ListView", func() interface{} { return &flow.ListView{} })
	SetFlowWrapper("Switch", func() interface{} { return &flow.Switch{} })
	SetFlowWrapper("GroupBox", func() interface{} { return &flow.GroupBox{} })
	SetFlowWrapper("StatusBar", func() interface{} { return &flow.StatusBar{} })
	SetFlowWrapper("ToolBar", func() interface{} { return &flow.ToolBar{} })
	SetFlowWrapper("Calendar", func() interface{} { return &flow.Calendar{} })
	SetFlowWrapper("ColorPicker", func() interface{} { return &flow.ColorPicker{} })
	SetFlowWrapper("MenuBar", func() interface{} { return &flow.MenuBar{} })
	SetFlowWrapper("Dialog", func() interface{} { return &flow.Dialog{} })
	SetFlowWrapper("SplitView", func() interface{} { return &flow.SplitView{} })
	SetFlowWrapper("ToolTip", func() interface{} { return &flow.ToolTip{} })
	SetFlowWrapper("Tree", func() interface{} { return &flow.Tree{} })
	SetFlowWrapper("Table", func() interface{} { return &flow.Table{} })
	SetFlowWrapper("ScrollPane", func() interface{} { return &flow.ScrollPane{} })
	SetFlowWrapper("CodeEditor", func() interface{} { return &flow.CodeEditor{} })
	SetFlowWrapper("MarkdownView", func() interface{} { return &flow.MarkdownView{} })
	SetFlowWrapper("MarkdownEditor", func() interface{} { return &flow.MarkdownEditor{} })
	SetFlowWrapper("ContextMenu", func() interface{} { return &flow.ContextMenu{} })
	SetFlowWrapper("MdiWindow", func() interface{} { return &flow.MdiWindow{} })
	SetFlowWrapper("MdiContainer", func() interface{} { return &flow.MdiContainer{} })
	SetFlowWrapper("DockPanel", func() interface{} { return &flow.DockPanel{} })
	SetFlowWrapper("StackPanel", func() interface{} { return &flow.StackPanel{} })
}

func LoadFileResult(path string) (*meui.Panel, *BoundWidgetRegistry, error) {
	// Initialize built-in widgets on first use
	initializeBuiltinWidgets()

	result, err := Load(path)
	if err != nil {
		return nil, nil, err
	}

	win, ok := result.Widget.(*meui.Window)
	if !ok {
		return nil, nil, fmt.Errorf("root is not a Window")
	}

	return win.AsPanel(), result.Widgets, nil
}

func parseTreeNode(obj map[string]interface{}) *widgets.TreeNode {
	label := ""
	if l, ok := obj["label"].(string); ok {
		label = l
	}
	expanded := true
	if e, ok := obj["expanded"].(bool); ok {
		expanded = e
	}
	node := &widgets.TreeNode{Label: label, Expanded: expanded}
	if icon, ok := obj["icon"].(string); ok {
		node.Icon = icon
	}
	if checkable, ok := obj["checkable"].(bool); ok {
		node.Checkable = checkable
	}
	if checked, ok := obj["checked"].(bool); ok {
		node.Checked = checked
	}
	if children, ok := obj["children"].([]interface{}); ok {
		for _, cv := range children {
			if co, ok := cv.(map[string]interface{}); ok {
				child := parseTreeNode(co)
				if child != nil {
					node.Children = append(node.Children, child)
				}
			}
		}
	}
	return node
}

// parseContextMenuItems recursively parses an array of QML context menu items,
// supporting nested "submenu" arrays.
func parseContextMenuItems(arr []interface{}) []widgets.MenuItem {
	var items []widgets.MenuItem
	for _, v := range arr {
		if obj, ok := v.(map[string]interface{}); ok {
			item := widgets.MenuItem{Enabled: true}
			if l, ok := obj["label"].(string); ok {
				item.Label = l
			}
			if s, ok := obj["shortcut"].(string); ok {
				item.Shortcut = s
			}
			if sep, ok := obj["separator"].(bool); ok && sep {
				item.IsSep = true
			}
			if check, ok := obj["checked"].(bool); ok {
				item.IsCheck = true
				item.Checked = check
			}
			if icon, ok := obj["icon"].(string); ok {
				item.Icon = icon
			}
			if subArr, ok := obj["submenu"].([]interface{}); ok {
				item.SubMenu = parseContextMenuItems(subArr)
			}
			items = append(items, item)
		}
	}
	return items
}
