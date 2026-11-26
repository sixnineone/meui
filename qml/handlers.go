package qml

import (
	"fmt"
	"os"
	"time"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/slant"
	"github.com/sixnineone/meui/graphics/enums/spacing"
	"github.com/sixnineone/meui/graphics/enums/weight"
	"github.com/sixnineone/meui/theme"
	"github.com/sixnineone/meui/widgets"
)

// PropHandler is a function that applies a QML property to a widget.
type PropHandler func(w meui.Widget, propName string, value interface{})

// propertyHandlers maps property name → handler function.
// When a flow wrapper is available, simple reflection handles most properties.
// Complex ones (font.*, color on Panel, rows, menus, etc.) use these handlers.
var propertyHandlers = map[string]PropHandler{}

// widgetPropertyHandlers maps widget type → property name → handler.
// Used for properties that behave differently per widget type.
var widgetPropertyHandlers = map[string]map[string]PropHandler{}

func init() {
	// Register all complex property handlers
	registerComplexHandlers()
}

// HandleProperty dispatches a property to its registered handler.
// Returns true if the property was handled.
func HandleProperty(w meui.Widget, propName string, value interface{}) bool {
	// Check specific handler first
	if h, ok := propertyHandlers[propName]; ok {
		h(w, propName, value)
		return true
	}
	// Fall back to catch-all (SetBoundProperty)
	if h, ok := propertyHandlers["*"]; ok {
		h(w, propName, value)
		return true
	}
	return false
}

// registerComplexHandlers sets up all handlers for properties that need custom logic.
func registerComplexHandlers() {
	// ── text (dispatched per widget type) ──
	propertyHandlers["text"] = func(w meui.Widget, propName string, val interface{}) {
		s, ok := val.(string)
		if !ok {
			return
		}
		switch v := w.(type) {
		case *widgets.Label:
			v.SetText(s)
		case *widgets.Button:
			v.SetText(s)
		case *widgets.TextField:
			v.SetText(s)
		case *widgets.CheckBox:
			v.SetText(s)
		case *widgets.RadioButton:
			v.SetText(s)
		case *widgets.TextArea:
			v.SetText(s)
		case *widgets.StatusBar:
			v.SetText(s)
		case *widgets.Tooltip:
			v.SetText(s)
		case *widgets.CodeEditor:
			v.SetText(s)
		case *widgets.MarkdownView:
			v.SetText(s)
		case *widgets.MarkdownEditor:
			v.SetText(s)
		case *widgets.DatePicker:
			v.SetText(s)
		case *widgets.ColorField:
			v.SetText(s)
		case *widgets.TimePicker:
			v.SetText(s)
		case *widgets.ListItem:
			// A ListItem's `text` is its PRIMARY line. This switch is the documented trap
			// (AGENTS.md): it is a GLOBAL handler for the property name, so a widget missing from
			// it has its `text:` silently dropped — it never reaches SetBoundProperty, and the row
			// renders with no primary line and no error anywhere.
			v.SetPrimary(s)
		case *widgets.ActionSheet:
			// Delegated, not re-decided: `text` means what the widget's own SetBoundProperty says it
			// means (the sheet's message), so the two paths cannot drift into different meanings.
			v.SetBoundProperty("text", s)
		case *widgets.AlertDialog:
			v.SetBoundProperty("text", s)
		}
	}

	// ── color (on Panel: SetBackground; on Label: SetColor) ──
	propertyHandlers["color"] = func(w meui.Widget, propName string, val interface{}) {
		s, ok := val.(string)
		if !ok {
			return
		}
		// Theme role tokens are stored, not baked: the widget re-resolves
		// them per paint so runtime theme switches recolour it.
		if _, isToken := theme.TokenColor(s); isToken {
			if p, ok := w.(*meui.Panel); ok {
				p.SetBackgroundToken(s)
				return
			}
			if l, ok := w.(*widgets.Label); ok {
				l.SetColorToken(s)
				return
			}
		}
		if p, ok := w.(*meui.Panel); ok {
			p.SetBackground(graphics.Ink(parseColor(s)))
			return
		}
		if l, ok := w.(*widgets.Label); ok {
			l.SetColor(parseColor(s))
			return
		}
		// The field widgets own their colour; without this case the handler
		// above swallows the property (it never falls through to
		// SetBoundProperty, the same trap the `text` switch documents).
		if sbp, ok := w.(interface {
			SetBoundProperty(string, interface{})
		}); ok {
			sbp.SetBoundProperty(propName, val)
		}
	}

	// ── font.family, font.pointSize, font.pixelSize, font.bold, font.italic ──
	for _, name := range []string{"font.family", "font.pointSize", "font.pixelSize", "font.bold", "font.italic"} {
		pn := name
		propertyHandlers[pn] = func(w meui.Widget, propName string, val interface{}) {
			handleFontProperty(w, propName, val)
		}
	}

	// ── icon ──
	propertyHandlers["icon"] = func(w meui.Widget, propName string, val interface{}) {
		s, ok := val.(string)
		if !ok {
			return
		}
		if b, ok := w.(*widgets.Button); ok {
			b.SetIcon(s)
			return
		}
		if mw, ok := w.(*widgets.MdiWindow); ok {
			mw.SetIcon(s)
			return
		}
		if sbp, ok := w.(interface{ SetBoundProperty(string, interface{}) }); ok {
			sbp.SetBoundProperty("icon", s)
		}
	}

	// ── groupID ──
	propertyHandlers["groupID"] = func(w meui.Widget, propName string, val interface{}) {
		s, ok := val.(string)
		if !ok {
			return
		}
		if b, ok := w.(*widgets.Button); ok {
			b.SetGroupID(s)
		}
		if rb, ok := w.(*widgets.RadioButton); ok {
			rb.SetGroupID(s)
		}
	}

	// ── selected ──
	propertyHandlers["selected"] = func(w meui.Widget, propName string, val interface{}) {
		b, ok := val.(bool)
		if !ok {
			return
		}
		if btn, ok := w.(*widgets.Button); ok {
			btn.SetSelected(b)
		}
		if rb, ok := w.(*widgets.RadioButton); ok {
			rb.SetSelected(b)
		}
	}

	// ── currentIndex ──
	propertyHandlers["currentIndex"] = func(w meui.Widget, propName string, val interface{}) {
		f, ok := toFloat32(val)
		if !ok {
			return
		}
		if cb, ok := w.(*widgets.ComboBox); ok {
			cb.SetSelectedIndex(int(f))
		}
		if lv, ok := w.(*widgets.ListView); ok {
			lv.SetSelectedIndex(int(f))
		}
	}

	// ── value ──
	propertyHandlers["value"] = func(w meui.Widget, propName string, val interface{}) {
		f, ok := toFloat32(val)
		if !ok {
			return
		}
		switch v := w.(type) {
		case *widgets.Slider:
			v.SetValue(f)
		case *widgets.SpinBox:
			v.SetValue(f)
		case *widgets.ProgressBar:
			v.SetValue(f)
		case *widgets.ScrollBar:
			v.SetValue(f)
		}
	}

	// ── min / minimumValue ──
	for _, name := range []string{"min", "minimumValue"} {
		pn := name
		propertyHandlers[pn] = func(w meui.Widget, propName string, val interface{}) {
			f, ok := toFloat32(val)
			if !ok {
				return
			}
			switch v := w.(type) {
			case *widgets.Slider:
				v.SetMinimum(f)
			case *widgets.SpinBox:
				v.Minimum = f
			case *widgets.ProgressBar:
				v.Minimum = f
			}
		}
	}

	// ── max / maximumValue ──
	for _, name := range []string{"max", "maximumValue"} {
		pn := name
		propertyHandlers[pn] = func(w meui.Widget, propName string, val interface{}) {
			f, ok := toFloat32(val)
			if !ok {
				return
			}
			switch v := w.(type) {
			case *widgets.Slider:
				v.SetMaximum(f)
			case *widgets.SpinBox:
				v.Maximum = f
			case *widgets.ProgressBar:
				v.Maximum = f
			}
		}
	}

	// ── orientation ──
	propertyHandlers["orientation"] = func(w meui.Widget, propName string, val interface{}) {
		s, ok := val.(string)
		if !ok {
			return
		}
		switch v := w.(type) {
		case *widgets.Slider:
			if s == "horizontal" || s == "Horizontal" {
				v.SetOrientation(0)
			} else {
				v.SetOrientation(1)
			}
		case *widgets.ScrollBar:
			if s == "horizontal" || s == "Horizontal" {
				v.SetOrientation(0)
			} else {
				v.SetOrientation(1)
			}
		case *widgets.SplitView:
			if s == "vertical" || s == "Vertical" {
				v.SetOrientation(1)
			} else {
				v.SetOrientation(0)
			}
		}
	}

	// ── options / model (ComboBox, ListView, Tree) ──
	for _, name := range []string{"options", "model"} {
		pn := name
		propertyHandlers[pn] = func(w meui.Widget, propName string, val interface{}) {
			arr, ok := val.([]interface{})
			if !ok {
				return
			}
			if cb, ok := w.(*widgets.ComboBox); ok {
				opts := make([]string, len(arr))
				for i, v := range arr {
					opts[i] = fmt.Sprintf("%v", v)
				}
				cb.SetOptions(opts)
				return
			}
			if lv, ok := w.(*widgets.ListView); ok {
				items := make([]string, len(arr))
				for i, v := range arr {
					items[i] = fmt.Sprintf("%v", v)
				}
				lv.SetItems(items)
				return
			}
			if tr, ok := w.(*widgets.Tree); ok {
				if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
					fmt.Printf("[tree model] entries=%d type=%T\n", len(arr), val)
				}
				root := &widgets.TreeNode{Label: "", Expanded: true}
				for _, v := range arr {
					if obj, ok := v.(map[string]interface{}); ok {
						node := parseTreeNode(obj)
						if node != nil {
							root.Children = append(root.Children, node)
						}
						continue
					}
					// Plain string → leaf node (model: ["A", "B"])
					root.Children = append(root.Children, &widgets.TreeNode{Label: fmt.Sprintf("%v", v), Expanded: true})
				}
				tr.SetRoot(root)
			}
		}
	}

	// ── columns (Table) ──
	propertyHandlers["columns"] = func(w meui.Widget, propName string, val interface{}) {
		arr, ok := val.([]interface{})
		if !ok {
			return
		}
		if tb, ok := w.(*widgets.Table); ok {
			cols := make([]string, len(arr))
			for i, v := range arr {
				cols[i] = fmt.Sprintf("%v", v)
			}
			tb.SetColumns(cols)
			return
		}
		if sbp, ok := w.(interface{ SetBoundProperty(string, interface{}) }); ok {
			sbp.SetBoundProperty("columns", val)
		}
	}

	// ── rows (Table / DataTable) ──
	propertyHandlers["rows"] = func(w meui.Widget, propName string, val interface{}) {
		arr, ok := val.([]interface{})
		if !ok {
			return
		}
		if tb, ok := w.(*widgets.Table); ok {
			var rows [][]widgets.TableCell
			for _, rv := range arr {
				if cells, ok := rv.([]interface{}); ok {
					var row []widgets.TableCell
					for _, cv := range cells {
						if cell, ok := cv.(map[string]interface{}); ok {
							tc := widgets.TableCell{}
							if text, ok := cell["text"].(string); ok {
								tc.Text = text
							}
							if isCheck, ok := cell["isCheck"].(bool); ok {
								tc.IsCheck = isCheck
							}
							if checked, ok := cell["checked"].(bool); ok {
								tc.Checked = checked
							}
							if icon, ok := cell["icon"].(string); ok {
								tc.Icon = icon
							}
							if iconSize, ok := cell["iconSize"].(float64); ok {
								tc.IconSize = float32(iconSize)
							}
							row = append(row, tc)
						}
					}
					if row != nil {
						rows = append(rows, row)
					}
				}
			}
			tb.SetRows(rows)
			return
		}
		if sbp, ok := w.(interface{ SetBoundProperty(string, interface{}) }); ok {
			sbp.SetBoundProperty("rows", val)
		}
	}

	// ── checked ╌─
	propertyHandlers["hAlign"] = func(w meui.Widget, propName string, val interface{}) {
		st, ok := val.(string)
		if !ok {
			return
		}
		if l, ok := w.(*widgets.Label); ok {
			l.SetHAlign(st)
		}
	}
	propertyHandlers["vAlign"] = func(w meui.Widget, propName string, val interface{}) {
		st, ok := val.(string)
		if !ok {
			return
		}
		if l, ok := w.(*widgets.Label); ok {
			l.SetVAlign(st)
		}
	}
	propertyHandlers["wrapMode"] = func(w meui.Widget, propName string, val interface{}) {
		st, ok := val.(string)
		if !ok {
			return
		}
		if l, ok := w.(*widgets.Label); ok {
			mode := st
			if mode == "word" {
				mode = widgets.WrapWordWrap
			} else if mode == "none" {
				mode = widgets.WrapNoWrap
			}
			l.SetWrapMode(mode)
		}
	}
	propertyHandlers["ellipsis"] = func(w meui.Widget, propName string, val interface{}) {
		b, ok := val.(bool)
		if !ok {
			return
		}
		if l, ok := w.(*widgets.Label); ok {
			if b {
				l.SetWrapMode(widgets.WrapEllipsis)
			}
		}
	}
	propertyHandlers["validationState"] = func(w meui.Widget, propName string, val interface{}) {
		st, ok := val.(string)
		if !ok {
			return
		}
		if tf, ok := w.(*widgets.TextField); ok {
			tf.SetValidationState(st)
		}
	}
	propertyHandlers["checked"] = func(w meui.Widget, propName string, val interface{}) {
		b, ok := val.(bool)
		if !ok {
			return
		}
		if cb, ok := w.(*widgets.CheckBox); ok {
			cb.SetChecked(b)
		}
		if sw, ok := w.(*widgets.Switch); ok {
			sw.SetChecked(b)
		}
		if rb, ok := w.(*widgets.RadioButton); ok {
			rb.SetSelected(b) // Qt alias: checked == selected
		}
		if bt, ok := w.(*widgets.Button); ok {
			bt.SetToggled(b) // checkable button state
		}
	}

	// ── delay (Tooltip) ──
	propertyHandlers["delay"] = func(w meui.Widget, propName string, val interface{}) {
		if tt, ok := w.(*widgets.Tooltip); ok {
			if f, ok := toFloat32(val); ok {
				tt.SetDelay(time.Duration(f) * time.Millisecond)
			}
		}
	}

	// ── splitPosition / position (SplitView) ──
	for _, name := range []string{"splitPosition", "position"} {
		pn := name
		propertyHandlers[pn] = func(w meui.Widget, propName string, val interface{}) {
			if sv, ok := w.(*widgets.SplitView); ok {
				if f, ok := toFloat32(val); ok {
					sv.SetSplitPosition(f)
				}
			}
		}
	}

	// ── menus (MenuBar) ──
	propertyHandlers["menus"] = func(w meui.Widget, propName string, val interface{}) {
		mb, ok := w.(*widgets.MenuBar)
		if !ok {
			return
		}
		arr, ok := val.([]interface{})
		if !ok {
			return
		}
		var menus []widgets.Menu
		for _, v := range arr {
			if obj, ok := v.(map[string]interface{}); ok {
				menu := widgets.Menu{}
				if l, ok := obj["title"].(string); ok {
					menu.Title = l
				}
				if itemsArr, ok := obj["items"].([]interface{}); ok {
					menu.Items = parseContextMenuItems(itemsArr)
				}
				menus = append(menus, menu)
			}
		}
		mb.SetMenus(menus)
	}

	// ── sections (StatusBar) ──
	propertyHandlers["sections"] = func(w meui.Widget, propName string, val interface{}) {
		sb, ok := w.(*widgets.StatusBar)
		if !ok {
			return
		}
		arr, ok := val.([]interface{})
		if !ok {
			return
		}
		sections := make([]string, 0, len(arr))
		for _, v := range arr {
			if s, ok := v.(string); ok {
				sections = append(sections, s)
			}
		}
		sb.SetSections(sections)
	}

	// ── buttons (ToolBar) ──
	propertyHandlers["buttons"] = func(w meui.Widget, propName string, val interface{}) {
		tb, ok := w.(*widgets.ToolBar)
		if !ok {
			return
		}
		arr, ok := val.([]interface{})
		if !ok {
			return
		}
		for _, v := range arr {
			if obj, ok := v.(map[string]interface{}); ok {
				text, _ := obj["text"].(string)
				icon, _ := obj["icon"].(string)
				tb.AddButton(text, icon)
			} else if s, ok := v.(string); ok {
				tb.AddButton(s, "")
			}
		}
	}

	// ── toolsets (ToolBar) ──
	propertyHandlers["toolsets"] = func(w meui.Widget, propName string, val interface{}) {
		tb, ok := w.(*widgets.ToolBar)
		if !ok {
			return
		}
		arr, ok := val.([]interface{})
		if !ok {
			return
		}
		var toolsets []widgets.ToolSet
		for _, v := range arr {
			if obj, ok := v.(map[string]interface{}); ok {
				var ts widgets.ToolSet
				if label, ok := obj["label"].(string); ok {
					ts.Label = label
				}
				if buttons, ok := obj["buttons"].([]interface{}); ok {
					for _, bv := range buttons {
						if bObj, ok := bv.(map[string]interface{}); ok {
							text, _ := bObj["text"].(string)
							icon, _ := bObj["icon"].(string)
							btn := widgets.NewButton()
							if text != "" {
								btn.SetText(text)
							}
							if icon != "" {
								btn.SetIcon(icon)
							}
							btn.SetWidth(28)
							btn.SetHeight(28)
							ts.Widgets = append(ts.Widgets, btn)
						}
					}
				}
				toolsets = append(toolsets, ts)
			}
		}
		if len(toolsets) > 0 {
			tb.SetToolsets(toolsets)
		}
	}

	// ── spacing (ToolBar) ──
	propertyHandlers["spacing"] = func(w meui.Widget, propName string, val interface{}) {
		if tb, ok := w.(*widgets.ToolBar); ok {
			if f, ok := toFloat32(val); ok {
				tb.SetSpacing(f)
			}
		}
	}

	// ── wrap (ToolBar) ──
	propertyHandlers["wrap"] = func(w meui.Widget, propName string, val interface{}) {
		if tb, ok := w.(*widgets.ToolBar); ok {
			if b, ok := val.(bool); ok {
				tb.SetWrap(b)
			}
		}
	}

	// ── horizontalScrollBarPolicy / verticalScrollBarPolicy (ScrollPane) ──
	// These use SetBoundProperty, handled by catch-all below.

	// ── title / modal / docksFullHeight / showTaskbar ──
	// These use SetBoundProperty, handled by catch-all below.

	// ── items (ContextMenu) ──
	propertyHandlers["items"] = func(w meui.Widget, propName string, val interface{}) {
		arr, ok := val.([]interface{})
		if !ok {
			return
		}
		// ContextMenu gets menu items; ComboBox/ListView accept `items` as an
		// alias for `model`/`options` (common QML habit).
		if cm, ok := w.(*widgets.ContextMenu); ok {
			cm.SetItems(parseContextMenuItems(arr))
			return
		}
		if h, ok := propertyHandlers["model"]; ok {
			h(w, propName, val)
		}
	}

	// ── title (ModalPopupPanel: Dialog/MessageBox/InputDialog/Popup) ──
	// Other widget types fall through to their SetBoundProperty (e.g. MdiWindow).
	propertyHandlers["title"] = func(w meui.Widget, propName string, val interface{}) {
		s, ok := val.(string)
		if !ok {
			return
		}
		if mp, ok := w.(*widgets.ModalPopupPanel); ok {
			mp.Title = s
			mp.MarkForRedraw()
			return
		}
		if sbp, ok := w.(interface{ SetBoundProperty(string, interface{}) }); ok {
			sbp.SetBoundProperty(propName, val)
		}
	}

	// ── message / label (ModalPopupPanel body text) ──
	for _, name := range []string{"message", "label"} {
		pn := name
		propertyHandlers[pn] = func(w meui.Widget, propName string, val interface{}) {
			s, ok := val.(string)
			if !ok {
				return
			}
			if mp, ok := w.(*widgets.ModalPopupPanel); ok {
				mp.Heading = s
				mp.MarkForRedraw()
			}
		}
	}

	// ── Catch-all: SetBoundProperty ──
	// Fallback for any property not handled above. If the widget implements
	// SetBoundProperty, pass the property name and value through.
	propertyHandlers["*"] = func(w meui.Widget, propName string, val interface{}) {
		if sbp, ok := w.(interface{ SetBoundProperty(string, interface{}) }); ok {
			sbp.SetBoundProperty(propName, val)
		}
	}
}

// handleFontProperty dispatches font.* properties.
func handleFontProperty(w meui.Widget, propName string, val interface{}) {
	l, ok := w.(*widgets.Label)
	if !ok {
		return
	}
	// Copy-on-write: never mutate the shared default theme
	l.LabelTheme = l.LabelTheme.Copy()
	switch propName {
	case "font.family":
		s, ok := val.(string)
		if !ok {
			return
		}
		currentFont := l.LabelTheme.Font
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
		l.LabelTheme.Font = face.Font(size)

	case "font.pointSize", "font.pixelSize":
		f, ok := toFloat32(val)
		if !ok {
			return
		}
		currentFont := l.LabelTheme.Font
		if currentFont == nil {
			currentFont = theme.DefaultTheme.BaseFont
		}
		if currentFont == nil {
			currentFont = graphics.SystemFont
		}
		family := currentFont.Face().Family()
		lw, sp, sl := currentFont.Face().Style()
		face := graphics.MatchFontFace(family, lw, sp, sl)
		l.LabelTheme.Font = face.Font(f)

	case "font.bold":
		b, ok := val.(bool)
		if !ok || !b {
			return
		}
		currentFont := l.LabelTheme.Font
		if currentFont == nil {
			currentFont = theme.DefaultTheme.BaseFont
		}
		if currentFont == nil {
			currentFont = graphics.SystemFont
		}
		family := currentFont.Face().Family()
		size := currentFont.Size()
		face := graphics.MatchFontFace(family, weight.Bold, spacing.Standard, slant.Upright)
		l.LabelTheme.Font = face.Font(size)

	case "font.italic":
		b, ok := val.(bool)
		if !ok || !b {
			return
		}
		currentFont := l.LabelTheme.Font
		if currentFont == nil {
			currentFont = theme.DefaultTheme.BaseFont
		}
		if currentFont == nil {
			currentFont = graphics.SystemFont
		}
		family := currentFont.Face().Family()
		size := currentFont.Size()
		face := graphics.MatchFontFace(family, weight.Regular, spacing.Standard, slant.Italic)
		l.LabelTheme.Font = face.Font(size)
	}
}
