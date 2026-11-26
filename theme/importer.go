package theme

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/sixnineone/meui/graphics"
)

// colorResolver resolves named colors to their hex values.
type colorResolver map[string]string

func (cr colorResolver) resolveStr(s string) string {
	if len(s) > 0 && s[0] == '#' {
		return s
	}
	if resolved, ok := cr[s]; ok {
		return resolved
	}
	return s
}

// resolveKey extracts a color string from a .theme.json UI value.
// Handles plain hex/"#RRGGBB", named colors, and sub-objects with "foreground"/"background" keys.
func (cr colorResolver) resolve(val interface{}) string {
	if m, ok := val.(map[string]interface{}); ok {
		if fg, ok := m["foreground"].(string); ok {
			return cr.resolveStr(fg)
		}
		if bg, ok := m["background"].(string); ok {
			return cr.resolveStr(bg)
		}
	}
	if s, ok := val.(string); ok {
		return cr.resolveStr(s)
	}
	return ""
}

// resolveSubKeys expands a UI value that may contain multiple sub-keys (like ProgressBar
// as {"trackColor":"...", "foreground":"..."}). Returns a map of sub-key → hex color.
func (cr colorResolver) resolveSubKeys(val interface{}) map[string]string {
	m, ok := val.(map[string]interface{})
	if !ok {
		return nil
	}
	result := make(map[string]string)
	for k, v := range m {
		if s, ok := v.(string); ok {
			result[k] = cr.resolveStr(s)
		}
	}
	return result
}

// ImportIntelliJThemeJSON loads an IntelliJ .theme.json file and returns a Theme.
// The Theme.Name is set from the "name" field and widget defaults are also updated.
func ImportIntelliJThemeJSON(path string) (*Theme, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read .theme.json: %w", err)
	}
	var doc struct {
		Name         string                 `json:"name"`
		Dark         bool                   `json:"dark"`
		Parent       string                 `json:"parentTheme"`
		Colors       map[string]string      `json:"colors"`
		UI           map[string]interface{} `json:"ui"`
		EditorScheme string                 `json:"editorScheme"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse .theme.json: %w", err)
	}

	// Build named color resolver
	cr := colorResolver(doc.Colors)

	// Start from dark or light base
	var t *Theme
	if doc.Dark {
		t = DarkTheme()
	} else {
		t = LightTheme()
	}
	t.Name = doc.Name

	// Parse wildcard "*" first (applied as background fallback)
	if wc, ok := doc.UI["*"].(map[string]interface{}); ok {
		if bg := cr.resolve(wc["background"]); bg != "" {
			t.Background = parseColor(bg)
			t.Surface = parseColor(bg)
		}
		if fg := cr.resolve(wc["foreground"]); fg != "" {
			t.TextPrimary = parseColor(fg)
		}
	}

	// Parse specific UI keys in a DETERMINISTIC order (see uiKeyRank):
	// many keys legitimately write the same Theme field, and the last
	// write wins. Iterating doc.UI directly used Go map order — random
	// per process — so a theme defining e.g. Table.background and
	// TextField.background resolved to different colours run to run.
	type uiEntry struct {
		key string
		raw interface{}
	}
	var entries []uiEntry
	for key, raw := range doc.UI {
		if key == "*" {
			continue // handled by the wildcard pre-pass above
		}
		// Check if the value has sub-keys (e.g., "ProgressBar": {"trackColor":"...", "foreground":"..."})
		if sub := cr.resolveSubKeys(raw); sub != nil {
			subKeys := make([]string, 0, len(sub))
			for subKey := range sub {
				subKeys = append(subKeys, subKey)
			}
			sort.Strings(subKeys)
			for _, subKey := range subKeys {
				entries = append(entries, uiEntry{key + "." + subKey, sub[subKey]})
			}
		} else {
			entries = append(entries, uiEntry{key, raw})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		ri, rj := uiKeyRank[entries[i].key], uiKeyRank[entries[j].key]
		if ri != rj {
			return ri < rj
		}
		return entries[i].key < entries[j].key
	})
	for _, e := range entries {
		if val := cr.resolve(e.raw); val != "" {
			mapIntelliJUIKey(t, e.key, val)
		}
	}

	// An imported L&F wants its splitter lines readable without hunting for
	// them; the built-in presets keep the subtle line that lights up on hover.
	t.SplitterHoverColorAlwaysOn = true
	return t, nil
}

// uiKeyRank gives UI keys a fixed application rank. Import applies keys
// in ascending rank (ties broken lexicographically), so the LAST write
// wins deterministically. Several keys write the same Theme field
// (Table.background and TextField.background both feed Background,
// SidePanel.background and TextArea.background both feed Surface, …);
// with map-order application the same file resolved to different
// colours run to run. Ranks encode "general beats specific": the
// overall panel colour lands last, editor/input colours first. Keys
// absent from the table rank 0 and apply before all ranked keys (only
// reachable for keys with a single writer, where order is irrelevant).
var uiKeyRank = map[string]int{
	// Background: per-widget content/editor colours are weaker than
	// the panel colour; Panel.background is authoritative.
	"Editor.background":              1,
	"ComboBox.background":            2,
	"ComboBox.nonEditableBackground": 3,
	"TextField.background":           4,
	"TextArea.background":            5,
	"List.background":                6,
	"Tree.background":                7,
	"Table.background":               8,

	// Surface / SurfaceVariant: tool chrome above input fields.
	"StatusBar.background":   1,
	"CheckBox.background":    2,
	"RadioButton.background": 3,
	"ToolWindow.background":  4,
	"SidePanel.background":   6,
	"Panel.background":       20,

	// TextPrimary: the IntelliJ global default is weakest; widget
	// foregrounds override it in increasing specificity, with Label
	// (the most common UI text) last among widgets and the panel
	// foreground as the authoritative general colour.
	"foregroundColor":      1,
	"Menu.foreground":      3,
	"MenuBar.foreground":   4,
	"List.foreground":      5,
	"Table.foreground":     6,
	"Tree.foreground":      7,
	"TextField.foreground": 8,
	"Button.foreground":    9,
	"Label.foreground":     10,
	"Panel.foreground":     20,

	// DisabledText
	"Button.disabledText":      1,
	"RadioButton.disabledText": 2,
	"Label.disabledText":       3,

	// Link
	"link.foregroundColor": 1,
	"link.foreground":      2,

	// MenuBackground
	"Menu.background":    1,
	"MenuBar.background": 2,
	"Popup.background":   3,

	// ScrollBarTrack
	"Slider.trackColor":      1,
	"ProgressBar.background": 2,
	"ProgressBar.trackColor": 3,
	"ScrollBar.trackColor":   4,

	// Accent
	"Slider.buttonColor":        1,
	"ProgressBar.foreground":    2,
	"ProgressBar.progressColor": 3,
	"Slider.thumbColor":         4,
	"TabbedPane.underlineColor": 5,

	// Selected
	"List.selectionBackground":  1,
	"Tree.selectionBackground":  2,
	"Table.selectionBackground": 3,
	"selectionBackground":       4,

	// Border / ChartAxis / ChartGrid: the general border colour wins
	// over per-widget grid/separator colours.
	"Menu.borderColor":         1,
	"Table.gridColor":          2,
	"Separator.foreground":     3,
	"Separator.separatorColor": 4,
	"Borders.color":            5,
}

// mapIntelliJUIKey maps a single .theme.json UI key to a Theme field.
func mapIntelliJUIKey(t *Theme, key, val string) {
	c := parseColor(val)
	if c == 0 {
		return
	}
	switch key {
	// Panel / general
	case "Panel.background":
		t.Background = c
		t.Surface = c
	case "Panel.foreground":
		t.TextPrimary = c

	// Label
	case "Label.foreground":
		t.TextPrimary = c
	case "Label.disabledText":
		t.TextSecondary = c
		t.DisabledText = c
		t.ChartAxis = c

	// CheckBox / RadioButton
	case "CheckBox.background", "RadioButton.background":
		t.Surface = c
	case "RadioButton.disabledText":
		t.DisabledText = c

	// TextField
	case "TextField.background":
		t.Background = c
	case "TextField.foreground":
		t.TextPrimary = c

	// Button
	case "Button.foreground":
		t.TextPrimary = c
	case "Button.disabledText":
		t.DisabledText = c

	// Menu
	case "Menu.background":
		t.MenuBackground = c
	case "Menu.foreground":
		t.TextPrimary = c
	case "Menu.borderColor":
		t.Border = c

	// MenuBar (fallback to Menu if not set)
	case "MenuBar.background":
		t.MenuBackground = c
	case "MenuBar.foreground":
		t.TextPrimary = c

	// Popup menus
	case "Popup.background":
		t.MenuBackground = c

	// ToolTip
	case "ToolTip.background":
		t.TooltipBackground = c
	case "ToolTip.foreground":
		t.TooltipText = c

	// Separator
	case "Separator.foreground", "Separator.separatorColor":
		t.Border = c
		t.ChartGrid = c

	// Focus
	case "Focus.color":
		t.BorderFocused = c

	// ScrollBar
	case "ScrollBar.thumbColor", "ScrollBar.hoverThumbColor",
		"ScrollBar.Mac.thumbColor", "ScrollBar.Transparent.thumbColor":
		t.ScrollBarThumb = c
	case "ScrollBar.trackColor":
		t.ScrollBarTrack = c

	// ProgressBar
	case "ProgressBar.foreground", "ProgressBar.progressColor":
		t.Accent = c
	case "ProgressBar.trackColor":
		t.ScrollBarTrack = c
	case "ProgressBar.background":
		t.ScrollBarTrack = c

	// Slider
	case "Slider.thumbColor", "Slider.buttonColor":
		t.Accent = c
	case "Slider.trackColor":
		t.ScrollBarTrack = c

	// TabbedPane
	case "TabbedPane.underlineColor":
		t.Accent = c

	// Editor
	case "Editor.background":
		t.Background = c

	// Table
	case "Table.background":
		t.Background = c
	case "Table.foreground":
		t.TextPrimary = c
	case "Table.selectionBackground":
		t.Selected = c
	case "Table.gridColor":
		t.Border = c
		t.ChartGrid = c

	// Tree
	case "Tree.background":
		t.Background = c
	case "Tree.foreground":
		t.TextPrimary = c
	case "Tree.selectionBackground":
		t.Selected = c

	// List
	case "List.background":
		t.Background = c
	case "List.foreground":
		t.TextPrimary = c
	case "List.selectionBackground":
		t.Selected = c

	// ComboBox
	case "ComboBox.background", "ComboBox.nonEditableBackground":
		t.Background = c

	// TextArea
	case "TextArea.background":
		t.Background = c
		t.Surface = c

	// Sidebar / tool window
	case "ToolWindow.background", "SidePanel.background":
		t.Surface = c
		t.SurfaceVariant = c

	// StatusBar
	case "StatusBar.background":
		t.Surface = c
		t.SurfaceVariant = c

	// Link
	case "link.foreground", "link.foregroundColor":
		t.Link = c

	// Component border
	case "Borders.color":
		t.Border = c
		t.ChartAxis = c

	// General wildcard-level accent/selection
	case "selectionBackground":
		t.Selected = c
	case "foregroundColor":
		t.TextPrimary = c
	}
}

// ApplyIntelliJTheme derives all per-widget theme defaults from the given Theme.
func ApplyIntelliJTheme(t *Theme) {
	t.withStyleDefaults()
	s := Shape() // L&F radii — theme loads must not reset the look
	m := t.Metrics
	// Resolve fallback chain: specific > theme > sensible default
	bg := t.Background
	if bg == 0 {
		bg = 0xFF1E1E1E
	}
	surface := t.Surface
	if surface == 0 {
		surface = 0xFF2D2D2D
	}
	surfaceVariant := t.SurfaceVariant
	if surfaceVariant == 0 {
		surfaceVariant = 0xFF252525
	}
	textPrimary := t.TextPrimary
	if textPrimary == 0 {
		textPrimary = 0xFFE0E0E0
	}
	textSecondary := t.TextSecondary
	if textSecondary == 0 {
		textSecondary = 0xFFA0A0A0
	}
	border := t.Border
	borderStrong := t.BorderStrong
	if borderStrong == 0 {
		borderStrong = border
	}
	borderSubtle := t.BorderSubtle
	if borderSubtle == 0 {
		borderSubtle = border
	}
	if border == 0 {
		border = 0xFF4A4A4A
	}
	borderFocused := t.BorderFocused
	if borderFocused == 0 {
		borderFocused = t.Accent
		if borderFocused == 0 {
			borderFocused = 0xFF2196F3
		}
	}
	disabledText := t.DisabledText
	if disabledText == 0 {
		disabledText = 0xFF505050
	}
	selected := t.Selected
	if selected == 0 {
		selected = 0x442196F3
	}
	hovered := t.Hovered
	if hovered == 0 {
		hovered = 0x332196F3
	}
	accent := t.Accent
	if accent == 0 {
		accent = 0xFF2196F3
	}
	menuBg := t.MenuBackground
	if menuBg == 0 {
		menuBg = surface
	}
	tooltipBg := t.TooltipBackground
	if tooltipBg == 0 {
		tooltipBg = 0xFF3C3C3C
	}
	tooltipText := t.TooltipText
	if tooltipText == 0 {
		tooltipText = textPrimary
	}
	scrollThumb := t.ScrollBarThumb
	if scrollThumb == 0 {
		scrollThumb = accent
	}
	scrollTrack := t.ScrollBarTrack
	if scrollTrack == 0 {
		scrollTrack = 0xFF3C3C3C
	}
	disabledBg := t.DisabledBg
	if disabledBg == 0 {
		disabledBg = 0xFF2A2A2A
	}
	disabledBorder := t.DisabledBorder
	if disabledBorder == 0 {
		disabledBorder = 0xFF3A3A3A
	}
	chartGrid := t.ChartGrid
	if chartGrid == 0 {
		chartGrid = border
	}
	chartAxis := t.ChartAxis
	if chartAxis == 0 {
		chartAxis = textSecondary
	}

	// On-accent text applies to solid selections; translucent selection
	// tints (Fluent) keep the normal text colour.
	selectedText := t.OnAccentColor()
	if selected>>24 < 0xFF {
		selectedText = textPrimary
	}

	// ── ButtonTheme ──
	*DefaultButtonTheme = ButtonTheme{
		BackgroundInk:     surface,
		BackgroundHover:   lighten(surface, 0x0A0A0A),
		BackgroundPressed: darken(surface, 0x0A0A0A),
		BackgroundActive:  accent,
		EdgeInk:           border,
		TextInk:           textPrimary,
		TextActive:        t.OnAccentColor(),
		TextDisabled:      disabledText,
		CornerRadius:      s.Button,
		EdgeWidth:         m.ControlBorderWidth,
		HMargin:           m.ControlPadX,
		VMargin:           m.ControlPadY,
		Font:              nil,
	}

	// ── CheckBoxTheme ──
	*DefaultCheckBoxTheme = CheckBoxTheme{
		BackgroundInk:   darken(surface, 0x050505),
		BackgroundHover: lighten(surface, 0x0A0A0A),
		EdgeInk:         borderStrong,
		EdgeFocused:     borderFocused,
		SelectionInk:    accent,
		TextInk:         textPrimary,
		TextDisabled:    disabledText,
		BoxSize:         m.BoxSize,
		BoxSpacing:      m.BoxSpacing,
		CornerRadius:    s.CheckBox,
		Font:            nil,
	}

	// ── RadioButtonTheme ──
	*DefaultRadioButtonTheme = RadioButtonTheme{
		BackgroundInk:   darken(surface, 0x050505),
		BackgroundHover: lighten(surface, 0x0A0A0A),
		EdgeInk:         borderStrong,
		EdgeFocused:     borderFocused,
		SelectionInk:    accent,
		TextInk:         textPrimary,
		TextDisabled:    disabledText,
		BoxSize:         m.BoxSize,
		BoxSpacing:      m.BoxSpacing,
		Font:            nil,
	}

	// ── LabelTheme ──
	*DefaultLabelTheme = LabelTheme{
		TextInk:       textPrimary,
		TextSecondary: textSecondary,
		Font:          nil,
	}

	// ── TextFieldTheme ──
	*DefaultTextFieldTheme = TextFieldTheme{
		BackgroundInk:     darken(surface, 0x0F0F0F),
		BackgroundFocused: surface,
		EdgeInk:           borderStrong,
		EdgeFocused:       borderFocused,
		EdgeHovered:       lighten(borderStrong, 0x202020),
		TextInk:           textPrimary,
		TextDisabled:      disabledText,
		PlaceholderInk:    darken(textPrimary, 0x606060),
		SelectionInk:      selected,
		SelectedTextInk:   selectedText,
		CornerRadius:      s.Input,
		BottomEdgeInk:     t.Inputs.BottomInk,
		BottomEdgeWidth:   t.Inputs.BottomWidth,
		BottomFocusWidth:  t.Inputs.FocusBottomWidth,
		HMargin:           m.InputPadX,
		VMargin:           m.ControlPadY,
		Font:              nil,
	}

	// ── SliderTheme ──
	thumbInk := t.Controls.sliderThumbInk(surface)
	if thumbInk == 0 {
		thumbInk = accent
	}
	*DefaultSliderTheme = SliderTheme{
		TrackInk:    scrollTrack,
		TrackFilled: accent,
		ThumbInk:    thumbInk,
		ThumbHover:  darken(thumbInk, 0x220000),
		EdgeInk:     border,
		TickColor:   darken(textPrimary, 0x7A7A7A),
		ThumbBorder: t.Controls.SliderThumbBorder,
		ThumbInner:  t.Controls.SliderThumbInner,
		ThumbSize:   m.SliderThumb,
		TickLength:  4,
		TrackHeight: m.sliderTrackHeight(),
		TrackRadius: m.sliderTrackRadius(),
		TrackBorder: t.Controls.SliderTrackBorder,
		Font:        nil,
	}

	// ── ProgressBarTheme ──
	*DefaultProgressBarTheme = ProgressBarTheme{
		TrackInk:     scrollTrack,
		FillInk:      accent,
		EdgeInk:      border,
		CornerRadius: m.progressRadius(),
		Height:       m.progressHeight(),
	}

	// ── TabTheme ──
	tabActive := surface
	if !m.TabActiveFilled {
		tabActive = bg // underline-style tabs (Fluent/Metro) keep the panel fill
	}
	*DefaultTabTheme = TabTheme{
		BackgroundInk:    bg,
		BackgroundActive: tabActive,
		BackgroundHover:  lighten(bg, 0x0B0B0B),
		EdgeInk:          border,
		TextInk:          textSecondary,
		TextActive:       textPrimary,
		TextHover:        textPrimary,
		IndicatorInk:     accent,
		IndicatorHeight:  m.TabIndicatorHeight,
		CornerRadius:     0,
		HMargin:          10,
		VMargin:          6,
		TabHeight:        m.TabHeight,
		CloseSize:        12,
		Font:             nil,
	}

	// ── ComboBoxTheme ──
	*DefaultComboBoxTheme = ComboBoxTheme{
		BackgroundInk:    darken(surface, 0x050505),
		EdgeInk:          borderStrong,
		TextInk:          textPrimary,
		TextDisabled:     disabledText,
		ArrowInk:         darken(textPrimary, 0x404040),
		HoverInk:         hovered,
		SelectedInk:      selected,
		OptionHeight:     m.MenuItemHeight,
		ArrowSize:        m.ArrowSize,
		ArrowStyle:       m.ArrowStyle,
		CornerRadius:     s.ComboBox,
		BottomEdgeInk:    t.Inputs.BottomInk,
		BottomEdgeWidth:  t.Inputs.BottomWidth,
		BottomFocusWidth: t.Inputs.FocusBottomWidth,
		Font:             nil,
	}

	// ── ListViewTheme ──
	*DefaultListViewTheme = ListViewTheme{
		CornerRadius:    s.Input,
		BackgroundInk:   darken(surface, 0x050505),
		EdgeInk:         border,
		TextInk:         textPrimary,
		SelectedInk:     selected,
		HoverInk:        hovered,
		AlternateRowInk: 0x11FFFFFF,
		Font:            nil,
	}

	// ── ScrollBarTheme ──
	*DefaultScrollBarTheme = ScrollBarTheme{
		TrackInk:   scrollTrack,
		ThumbInk:   scrollThumb,
		ThumbHover: lighten(scrollThumb, 0x220000),
		EdgeInk:    border,
		Overlay:    m.ScrollbarOverlay,
		AutoHide:   m.ScrollbarAutoHide,
		ThumbInset: 2,
	}

	// ── SpinBoxTheme ──
	*DefaultSpinBoxTheme = SpinBoxTheme{
		CornerRadius:     s.Input,
		BackgroundInk:    darken(surface, 0x050505),
		EdgeInk:          border,
		TextInk:          textPrimary,
		ButtonInk:        surface,
		ButtonPressedInk: darken(surface, 0x120000),
		ArrowInk:         darken(textPrimary, 0x404040),
		Font:             nil,
	}

	// ── TextAreaTheme ──
	*DefaultTextAreaTheme = TextAreaTheme{
		CornerRadius:      s.Input,
		BackgroundInk:     darken(surface, 0x0F0F0F),
		BackgroundFocused: surface,
		EdgeInk:           border,
		EdgeFocused:       borderFocused,
		TextInk:           textPrimary,
		PlaceholderInk:    darken(textPrimary, 0x606060),
		SelectionColor:    selected,
		GutterBgInk:       darken(bg, 0x070707),
		GutterTextInk:     darken(textPrimary, 0x787878),
		GutterActiveInk:   textPrimary,
		Font:              nil,
	}

	// ── SwitchTheme ──
	*DefaultSwitchTheme = SwitchTheme{
		TrackOn:  accent,
		TrackOff: border,
		ThumbInk: t.OnAccentColor(),
		ThumbOff: surface,
	}

	// ── TooltipTheme ──
	*DefaultTooltipTheme = TooltipTheme{
		BackgroundInk: tooltipBg,
		EdgeInk:       lighten(tooltipBg, 0x141414),
		TextInk:       tooltipText,
		Font:          nil,
	}

	// ── StatusBarTheme ──
	*DefaultStatusBarTheme = StatusBarTheme{
		BackgroundInk: darken(bg, 0x040404),
		EdgeInk:       border,
		TextInk:       darken(textPrimary, 0x303030),
		GripInk:       darken(textPrimary, 0x808080),
		Font:          nil,
	}

	// ── ToolBarTheme ──
	*DefaultToolBarTheme = ToolBarTheme{
		BackgroundInk: darken(bg, 0x040404),
		EdgeInk:       border,
		ButtonInk:     surface,
		ButtonHover:   lighten(surface, 0x141414),
		ButtonToggled: selected,
		IconInk:       textPrimary,
		TextInk:       textPrimary,
		SeparatorInk:  darken(textPrimary, 0x909090),
		Font:          nil,
	}

	// ── MenuBarTheme ──
	*DefaultMenuBarTheme = MenuBarTheme{
		BackgroundInk:    menuBg,
		BackgroundHover:  lighten(menuBg, 0x1D1D1D),
		BackgroundActive: accent,
		TextInk:          textPrimary,
		TextActive:       0xFFFFFFFF,
		TextSecondary:    darken(textPrimary, 0x585858),
		EdgeInk:          border,
		Font:             nil,
	}

	// ── ContextMenuTheme ──
	*DefaultContextMenuTheme = ContextMenuTheme{
		BackgroundInk: menuBg,
		EdgeInk:       border,
		TextInk:       textPrimary,
		TextDisabled:  disabledText,
		ArrowInk:      darken(textPrimary, 0x404040),
		HoverInk:      hovered,
		SelectedInk:   selected,
		OptionHeight:  m.MenuItemHeight,
		ArrowSize:     m.ArrowSize,
		Font:          nil,
	}

	// ── CalendarTheme ──
	*DefaultCalendarTheme = CalendarTheme{
		BackgroundInk:    surface,
		BackgroundActive: lighten(surface, 0x0F0F0F),
		BackgroundHover:  lighten(surface, 0x0B0B0B),
		EdgeInk:          border,
		TextInk:          textSecondary,
		TextActive:       textPrimary,
		TextHover:        textPrimary,
		CornerRadius:     0,
		HMargin:          10,
		VMargin:          6,
		TabHeight:        m.TabHeight,
		CloseSize:        12,
		Font:             nil,
	}

	// ── ColorPickerTheme ──
	*DefaultColorPickerTheme = ColorPickerTheme{
		BackgroundInk:    surface,
		BackgroundActive: lighten(surface, 0x0F0F0F),
		BackgroundHover:  lighten(surface, 0x0B0B0B),
		EdgeInk:          border,
		TextInk:          textSecondary,
		TextActive:       textPrimary,
		TextHover:        textPrimary,
		CornerRadius:     0,
		HMargin:          10,
		VMargin:          6,
		TabHeight:        m.TabHeight,
		CloseSize:        12,
		Font:             nil,
	}

	// ── GroupBoxTheme ──
	*DefaultGroupBoxTheme = GroupBoxTheme{
		TextInk:       textPrimary,
		TextSecondary: textSecondary,
		Font:          nil,
	}

	// ── ScrollPaneTheme ──
	*DefaultScrollPaneTheme = ScrollPaneTheme{
		CornerRadius:    s.Input,
		BackgroundInk:   bg,
		EdgeInk:         border,
		TextInk:         textPrimary,
		SelectedInk:     selected,
		HoverInk:        hovered,
		AlternateRowInk: 0x11FFFFFF,
		Font:            nil,
	}

	// ── TableTheme ──
	*DefaultTableTheme = TableTheme{
		CornerRadius:     s.Input,
		BackgroundInk:    darken(surface, 0x050505),
		EdgeInk:          border,
		TextInk:          textPrimary,
		SelectedInk:      selected,
		HoverInk:         hovered,
		AlternateRowInk:  0x11FFFFFF,
		SecondaryTextInk: textSecondary,
		GridInk:          borderSubtle,
		HeaderBgInk:      surfaceVariant,
		AccentInk:        accent,
		SelectedTextInk:  selectedText,
		Font:             nil,
	}

	// ── StackPanelTheme ──
	// The splitter is a divider, not a raised band: it takes the theme's
	// border colour and lightens/darkens the accent for hover, so it reads
	// correctly on light and dark themes alike (BackgroundInk stays
	// transparent — filling the band with a lightened panel background is
	// what made every split look like a bright strip).
	*DefaultStackPanelTheme = StackPanelTheme{
		CornerRadius:     s.GroupBox,
		BackgroundInk:    0x00000000,
		SplitterInk:      border,
		SplitterHoverInk: accent,
		// The split stays one line: no grip marks and no band, whatever the
		// source L&F drew on its splitters. The line is drawn in the accent,
		// always: an imported palette's separator colour is the same tone as
		// its chrome, so a hover-only highlight would leave every split
		// nearly invisible until the pointer found it.
		GripInk:            0,
		GripSize:           m.GripSize,
		SplitterSize:       m.GripSize,
		HoverColorAlwaysOn: t.SplitterHoverColorAlwaysOn,
	}

	// ── MessageBoxTheme ──
	*DefaultMessageBoxTheme = MessageBoxTheme{
		BackgroundInk: surface,
		EdgeInk:       border,
		TitleBgInk:    surfaceVariant,
		TitleTextInk:  textPrimary,
		DividerInk:    border,
		BodyTextInk:   textPrimary,
		ButtonBgInk:   surface,
		ButtonEdgeInk: border,
		ButtonTextInk: textPrimary,
		Font:          nil,
	}

	// ── InputDialogTheme ──
	*DefaultInputDialogTheme = InputDialogTheme{
		BackgroundInk:    surface,
		EdgeInk:          border,
		TitleBgInk:       surfaceVariant,
		TitleTextInk:     textPrimary,
		DividerInk:       border,
		LabelTextInk:     textPrimary,
		InputBgInk:       darken(surface, 0x0C0C0C),
		InputBgFocused:   darken(surface, 0x050505),
		InputEdgeInk:     border,
		InputEdgeFocused: borderFocused,
		InputTextInk:     textPrimary,
		CursorInk:        textPrimary,
		ButtonBgInk:      surface,
		ButtonEdgeInk:    border,
		ButtonTextInk:    textPrimary,
		Font:             nil,
	}

	// ── PopupTheme ──
	*DefaultPopupTheme = PopupTheme{
		BackgroundInk:  surface,
		EdgeInk:        border,
		TitleBgInk:     surfaceVariant,
		TitleTextInk:   textPrimary,
		DividerInk:     border,
		HeadingTextInk: 0xFFFFFFFF,
		ShadowInk:      0x60000000,
		Font:           nil,
	}

	// ── MdiTheme ──
	*DefaultMdiTheme = MdiTheme{
		CornerRadius:     s.Popup,
		ContainerBgInk:   bg,
		TabBgInk:         darken(bg, 0x1D1D1D),
		TabActiveBgInk:   bg,
		TabEdgeInk:       border,
		TabTextInk:       textSecondary,
		TabActiveTextInk: textPrimary,
		TaskbarBgInk:     surface,
		TaskbarTextInk:   textSecondary,
		TaskbarActiveInk: accent,
		TaskbarBtnBgInk:  surface,
		TaskbarBtnHover:  lighten(surface, 0x050505),
		TaskbarBtnEdge:   border,
		DockBgInk:        darken(bg, 0x0C0C0C),
		DockEdgeInk:      lighten(bg, 0x1C1C1C),
		DockSplitInk:     lighten(bg, 0x2C2C2C),
		WinTitleBgInk:    lighten(bg, 0x1C1C1C),
		WinTitleInactive: bg,
		WinTitleEdgeInk:  border,
		WinTitleTextInk:  textPrimary,
		WinEdgeInk:       border,
		WinContentBgInk:  darken(bg, 0x0C0C0C),
		WinAccentInk:     accent,
		WinTextSecondary: textSecondary,
		WinShadowInk:     0x40000000,
		Font:             nil,
	}
}

// Apply installs t as the active theme in one step: the IntelliJ colour
// set is resolved into every widget theme struct, then the global
// default is swapped, ThemeVersion bumps, and listeners are notified.
//
// Startup and interactive switching must both go through this function.
// The split pair (ApplyIntelliJTheme + SetDefaultTheme) lets callers
// forget half: skipping SetDefaultTheme leaves DefaultTheme stale while
// widget themes change; skipping ApplyIntelliJTheme swaps the global
// pointer but every widget keeps rendering the previous colour set
// indefinitely. One call, one state.
func Apply(t *Theme) {
	if t == nil {
		return
	}
	ApplyIntelliJTheme(t)
	SetDefaultTheme(t)
}

// ── Color helpers ──────────────────────────────────────────

// lighten returns a color shifted toward white by mixing with the given amount.
func lighten(c graphics.Color, amt graphics.Color) graphics.Color {
	r := uint8((c >> 16) & 0xFF)
	g := uint8((c >> 8) & 0xFF)
	b := uint8(c & 0xFF)
	ar := uint8((amt >> 16) & 0xFF)
	ag := uint8((amt >> 8) & 0xFF)
	ab := uint8(amt & 0xFF)
	nr := uint8(min(int(r)+int(ar), 255))
	ng := uint8(min(int(g)+int(ag), 255))
	nb := uint8(min(int(b)+int(ab), 255))
	return graphics.Color(0xFF000000 | (uint32(nr) << 16) | (uint32(ng) << 8) | uint32(nb))
}

// darken returns a color shifted toward black by subtracting the given amount.
func darken(c graphics.Color, amt graphics.Color) graphics.Color {
	r := uint8((c >> 16) & 0xFF)
	g := uint8((c >> 8) & 0xFF)
	b := uint8(c & 0xFF)
	ar := uint8((amt >> 16) & 0xFF)
	ag := uint8((amt >> 8) & 0xFF)
	ab := uint8(amt & 0xFF)
	nr := uint8(max(int(r)-int(ar), 0))
	ng := uint8(max(int(g)-int(ag), 0))
	nb := uint8(max(int(b)-int(ab), 0))
	return graphics.Color(0xFF000000 | (uint32(nr) << 16) | (uint32(ng) << 8) | uint32(nb))
}

// ── Legacy XML .icls importer ─────────────────────────────

// ImportIntelliJTheme loads an IntelliJ IDEA .xml/.icls theme file and returns a Theme.
func ImportIntelliJTheme(path string) (*Theme, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read IntelliJ theme file: %w", err)
	}
	t := DarkTheme()
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch elem := token.(type) {
		case xml.StartElement:
			if elem.Name.Local == "option" {
				var name, value string
				for _, attr := range elem.Attr {
					if attr.Name.Local == "name" {
						name = attr.Value
					} else if attr.Name.Local == "value" {
						value = attr.Value
					}
				}
				mapIntelliJColor(t, name, value)
			}
		}
	}
	return t, nil
}

func mapIntelliJColor(t *Theme, name, value string) {
	switch name {
	case "default.foreground":
		t.TextPrimary = parseColor(value)
	case "default.background":
		t.Background = parseColor(value)
	case "selection.background":
		t.Selected = parseColor(value)
	}
}

// ── FlatLaf importer ──────────────────────────────────────

// ImportFlatLafTheme loads a FlatLaf .properties or .json theme file and returns a Theme.
func ImportFlatLafTheme(path string) (*Theme, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read FlatLaf theme file: %w", err)
	}
	t := LightTheme()
	if strings.HasSuffix(path, ".json") {
		var flatLaf map[string]interface{}
		if err := json.Unmarshal(data, &flatLaf); err != nil {
			return nil, fmt.Errorf("failed to parse FlatLaf JSON theme: %w", err)
		}
		for k, v := range flatLaf {
			if s, ok := v.(string); ok {
				mapFlatLafColor(t, k, s)
			}
		}
	} else {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			mapFlatLafColor(t, key, value)
		}
	}
	return t, nil
}

func mapFlatLafColor(t *Theme, key, value string) {
	switch key {
	case "window.background":
		t.Background = parseColor(value)
	case "text.foreground":
		t.TextPrimary = parseColor(value)
	}
}

// ── VS Code importer ──────────────────────────────────────

// ImportVSCodeTheme loads a VS Code .json theme file and returns a Theme.
func ImportVSCodeTheme(path string) (*Theme, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read VS Code theme file: %w", err)
	}
	t := DarkTheme()
	var vsCode map[string]interface{}
	if err := json.Unmarshal(data, &vsCode); err != nil {
		return nil, fmt.Errorf("failed to parse VS Code JSON theme: %w", err)
	}
	if colors, ok := vsCode["colors"].(map[string]interface{}); ok {
		for key, value := range colors {
			if strVal, ok := value.(string); ok {
				mapVSCodeColor(t, key, strVal)
			}
		}
	}
	return t, nil
}

func mapVSCodeColor(t *Theme, key, value string) {
	switch key {
	case "editor.background":
		t.Background = parseColor(value)
	case "editor.foreground":
		t.TextPrimary = parseColor(value)
	case "activityBar.background":
		t.Surface = parseColor(value)
	}
}

// ── Color parsing ─────────────────────────────────────────

// ── Editor scheme (.icls/.xml) parser ────────────────────

// EditorScheme holds the parsed editor color scheme from an .icls/.xml file.
type EditorScheme struct {
	Name       string
	Colors     map[string]string            // e.g. "LINE_NUMBERS_COLOR" → "999999"
	Attributes map[string]map[string]string // e.g. "KEYWORD" → {"FOREGROUND":"c678dd", "FONT_TYPE":"1"}
}

// ImportEditorScheme parses an IntelliJ .icls/.xml editor scheme file.
func ImportEditorScheme(path string) (*EditorScheme, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read editor scheme: %w", err)
	}
	s := &EditorScheme{
		Colors:     make(map[string]string),
		Attributes: make(map[string]map[string]string),
	}
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	var currentAttr string
	var inValue bool
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch elem := token.(type) {
		case xml.StartElement:
			if elem.Name.Local == "scheme" {
				for _, attr := range elem.Attr {
					if attr.Name.Local == "name" {
						s.Name = attr.Value
					}
				}
			}
			if elem.Name.Local == "option" {
				var name, value string
				for _, attr := range elem.Attr {
					if attr.Name.Local == "name" {
						name = attr.Value
					} else if attr.Name.Local == "value" {
						value = attr.Value
					}
				}
				if inValue && currentAttr != "" {
					if s.Attributes[currentAttr] == nil {
						s.Attributes[currentAttr] = make(map[string]string)
					}
					s.Attributes[currentAttr][name] = value
				} else if !inValue {
					// Check parent for attributes block
					currentAttr = name
				}
			}
		case xml.EndElement:
			if elem.Name.Local == "value" {
				inValue = false
			}
			if elem.Name.Local == "option" && !inValue {
				currentAttr = ""
			}
		case xml.CharData:
			// Not needed for attributes (they use child <option> elements)
		}
	}

	// Re-parse properly with state tracking for nested structure
	// The XML looks like:
	// <colors>
	//   <option name="LINE_NUMBERS_COLOR" value="999999"/>
	// </colors>
	// <attributes>
	//   <option name="KEYWORD">  ← attribute name
	//     <value>
	//       <option name="FOREGROUND" value="c678dd"/>  ← property
	//     </value>
	//   </option>
	// </attributes>
	s2 := &EditorScheme{
		Name:       s.Name,
		Colors:     make(map[string]string),
		Attributes: make(map[string]map[string]string),
	}
	decoder2 := xml.NewDecoder(strings.NewReader(string(data)))
	type state struct {
		inColors     bool
		inAttributes bool
		attrName     string
		inValue      bool
	}
	var st state
	for {
		token, err := decoder2.Token()
		if err != nil {
			break
		}
		switch elem := token.(type) {
		case xml.StartElement:
			switch elem.Name.Local {
			case "colors":
				st.inColors = true
			case "attributes":
				st.inAttributes = true
			case "value":
				if st.attrName != "" {
					st.inValue = true
				}
			case "option":
				var name, value string
				for _, attr := range elem.Attr {
					if attr.Name.Local == "name" {
						name = attr.Value
					} else if attr.Name.Local == "value" {
						value = attr.Value
					}
				}
				if st.inColors && name != "" {
					s2.Colors[name] = value
				} else if st.inAttributes && name != "" && value == "" {
					// This is an attribute definition (has name but no value)
					st.attrName = name
				} else if st.inValue && st.attrName != "" && name != "" {
					if s2.Attributes[st.attrName] == nil {
						s2.Attributes[st.attrName] = make(map[string]string)
					}
					s2.Attributes[st.attrName][name] = value
				}
			}

		case xml.EndElement:
			switch elem.Name.Local {
			case "colors":
				st.inColors = false
			case "attributes":
				st.inAttributes = false
			case "value":
				st.inValue = false
			case "option":
				if st.attrName != "" && !st.inValue {
					st.attrName = ""
				}
			}
		}
	}
	return s2, nil
}

// parseColor parses a hex color string (with or without #) to graphics.Color.
// Supports: "#RRGGBB", "#AARRGGBB", "#RGB", "RRGGBB" (6 chars), "AARRGGBB" (8 chars).
func parseColor(s string) graphics.Color {
	s = strings.TrimPrefix(s, "#")
	// 3-digit shorthand: #RGB → #RRGGBB
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) == 6 {
		var r, g, b uint8
		fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b)
		return graphics.Color(0xFF000000 | (uint32(r) << 16) | (uint32(g) << 8) | uint32(b))
	} else if len(s) == 8 {
		var a, r, g, b uint8
		fmt.Sscanf(s, "%02x%02x%02x%02x", &a, &r, &g, &b)
		return graphics.Color((uint32(r) << 16) | (uint32(g) << 8) | uint32(b) | (uint32(a) << 24))
	}
	return 0
}
