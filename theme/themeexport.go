package theme

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/sixnineone/meui/graphics"
)

// colorType is the Theme colour field type (a uint32 alias). Matched by
// identity so reflection encodes colours as hex and everything else
// numerically.
var colorType = reflect.TypeOf(graphics.Color(0))

// fontType is the resolved-font pointer type. FontImpl has no exported
// fields, so reflection would render a font as an empty object; it is
// encoded as a {family,size} descriptor instead, and never loaded back
// (fonts are host-resolved).
var fontType = reflect.TypeOf(graphics.Font(nil))

// paletteTarget maps a theme-file palette key to a Theme colour field.
type paletteTarget struct {
	key string
	dst *graphics.Color
}

// paletteColorTargets is the single source of truth for the palette
// section, shared by the loader and the exporter. Every top-level colour
// role is listed, so an exported theme round-trips exactly instead of
// leaning on the base preset.
//
// "raised" is not listed: it is a legacy alias for surfaceVariant,
// handled by the loader.
func paletteColorTargets(t *Theme) []paletteTarget {
	return []paletteTarget{
		{"primary", &t.Primary},
		{"secondary", &t.Secondary},
		{"accent", &t.Accent},
		{"onAccent", &t.OnAccent},
		{"background", &t.Background},
		{"surface", &t.Surface},
		{"surfaceVariant", &t.SurfaceVariant},
		{"textPrimary", &t.TextPrimary},
		{"textSecondary", &t.TextSecondary},
		{"border", &t.Border},
		{"borderStrong", &t.BorderStrong},
		{"borderSubtle", &t.BorderSubtle},
		{"borderFocused", &t.BorderFocused},
		{"borderHovered", &t.BorderHovered},
		{"hover", &t.Hovered},
		{"pressed", &t.Pressed},
		{"selected", &t.Selected},
		{"disabled", &t.Disabled},
		{"disabledText", &t.DisabledText},
		{"disabledBg", &t.DisabledBg},
		{"disabledBorder", &t.DisabledBorder},
		{"success", &t.Success},
		{"warning", &t.Warning},
		{"error", &t.Error},
		{"shadow", &t.Shadow},
		{"tooltipBackground", &t.TooltipBackground},
		{"tooltipText", &t.TooltipText},
		{"link", &t.Link},
		{"linkVisited", &t.LinkVisited},
		{"menuBackground", &t.MenuBackground},
		{"menuHighlight", &t.MenuHighlight},
		{"scrollBarTrack", &t.ScrollBarTrack},
		{"scrollBarThumb", &t.ScrollBarThumb},
		{"chartGrid", &t.ChartGrid},
		{"chartAxis", &t.ChartAxis},
	}
}

// parseThemeColor is the lenient form used by theme-file palette, chrome
// and token values. Unlike parseHexColor — which is accent-oriented and
// rejects black/transparent — it accepts any "#RRGGBB", "#AARRGGBB",
// "0xRRGGBB" or "0xAARRGGBB", including #00000000.
func parseThemeColor(s string) (graphics.Color, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	s = strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, false
	}
	c := graphics.Color(v)
	if len(s) <= 6 {
		c |= 0xFF000000
	}
	return c, true
}

// formatThemeColor renders a colour as "#RRGGBB" (opaque) or
// "#AARRGGBB" (with alpha), the inverse of parseThemeColor.
func formatThemeColor(c graphics.Color) string {
	if c>>24 == 0xFF {
		return fmt.Sprintf("#%06X", uint32(c)&0x00FFFFFF)
	}
	return fmt.Sprintf("#%08X", uint32(c))
}

// ToThemeFile renders t as a native theme file: the friendly palette plus
// a complete "tokens" section covering every non-colour theme field and
// the platform token groups. Loading the result reproduces t for
// everything the format carries — BaseFont is host-resolved and
// deliberately excluded.
//
// The id is t.Meta.ID (falling back to t.Name). LoadThemeFile rejects a
// duplicate id, so a caller saving an edited copy should set a fresh one.
func ToThemeFile(t *Theme) ([]byte, error) {
	if t == nil {
		return nil, fmt.Errorf("theme file: nil theme")
	}
	data, err := json.MarshalIndent(themeFileFrom(t), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("theme file: %w", err)
	}
	return append(data, '\n'), nil
}

// SaveThemeFile writes ToThemeFile(t) to path.
func SaveThemeFile(path string, t *Theme) error {
	data, err := ToThemeFile(t)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ParseThemeFile builds a theme from native theme file bytes **without**
// registering it as a preset. LoadThemeFile is for installing a theme
// under its id; this is for inspecting, previewing or comparing one — a
// tool needs to diff two revisions of the same file, which share an id
// and would collide in the registry.
func ParseThemeFile(data []byte) (*Theme, error) {
	var tf themeFile
	if err := json.Unmarshal(data, &tf); err != nil {
		return nil, fmt.Errorf("theme file: %w", err)
	}
	base := tf.Base
	if base == "" {
		base = "meui-dark"
	}
	build, ok := lookupPresetBuilder(base)
	if !ok {
		return nil, fmt.Errorf("theme file: unknown base %q", base)
	}
	t := build()
	t.withStyleDefaults()
	if tf.ID != "" {
		t.Name = tf.ID
		t.Meta.ID = tf.ID
	}
	applyThemeFile(t, &tf)
	return t, nil
}

// DiffThemes returns a sorted list of "path: old -> new" lines covering
// every palette and token difference between a and b. It backs a theme
// editor's change view and the round-trip tests.
func DiffThemes(a, b *Theme) []string {
	return diffFlat(themeFilePathMap(a), themeFilePathMap(b))
}

// DiffWidgetThemes compares two ResolvedWidgetThemes snapshots and
// returns sorted "widgets.<name>.<field>: old -> new" lines.
//
// Snapshots are the input rather than *Theme values because the widget
// structs are process-global singletons regenerated on every Apply — only
// a snapshot taken right after loading a theme is meaningful. Use
// ResolvedWidgetThemes between loads.
func DiffWidgetThemes(a, b map[string]map[string]any) []string {
	am := make(map[string]string)
	bm := make(map[string]string)
	for name, fields := range a {
		flattenValue("widgets."+name, map[string]any(fields), am)
	}
	for name, fields := range b {
		flattenValue("widgets."+name, map[string]any(fields), bm)
	}
	return diffFlat(am, bm)
}

func diffFlat(am, bm map[string]string) []string {
	keys := make(map[string]struct{}, len(am)+len(bm))
	for k := range am {
		keys[k] = struct{}{}
	}
	for k := range bm {
		keys[k] = struct{}{}
	}
	out := make([]string, 0, len(keys))
	for k := range keys {
		av, aok := am[k]
		bv, bok := bm[k]
		if aok && bok && av == bv {
			continue
		}
		out = append(out, fmt.Sprintf("%s: %s -> %s", k, valueOrUnset(av, aok), valueOrUnset(bv, bok)))
	}
	sort.Strings(out)
	return out
}

func valueOrUnset(v string, ok bool) string {
	if !ok {
		return "(unset)"
	}
	return v
}

// themeFileFrom builds the in-memory representation both ToThemeFile and
// DiffThemes work from.
func themeFileFrom(t *Theme) *themeFile {
	// Base only matters for what the file does not carry (widget themes,
	// BaseFont); every palette and token value is written explicitly.
	base := "meui-dark"
	if _, ok := lookupPresetBuilder(t.Meta.ID); ok {
		base = t.Meta.ID
	}
	id := t.Meta.ID
	if id == "" {
		id = t.Name
	}
	if id == "" {
		id = "theme"
	}

	palette := make(map[string]string, 35)
	for _, target := range paletteColorTargets(t) {
		palette[target.key] = formatThemeColor(*target.dst)
	}

	return &themeFile{
		ID:      id,
		Base:    base,
		Palette: palette,
		Tokens:  encodeTokens(t),
	}
}

func lookupPresetBuilder(id string) (func() *Theme, bool) {
	if id == "" {
		return nil, false
	}
	presetMu.RLock()
	defer presetMu.RUnlock()
	build, ok := presetBuilders[id]
	return build, ok
}

// encodeTokens reflects over the whole Theme and returns every field the
// palette does not already carry: the numeric top-level fields plus the
// token groups (which include their own nested colours, as hex).
func encodeTokens(t *Theme) map[string]any {
	out := make(map[string]any)
	v := reflect.ValueOf(*t)
	typ := v.Type()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		switch f.Name {
		case "Name", "BaseFont":
			continue // Name is the file id; BaseFont is host-resolved
		}
		if f.Type == colorType {
			continue // colours live in the palette section
		}
		out[jsonName(f.Name)] = encodeValue(v.Field(i))
	}
	return out
}

func encodeValue(v reflect.Value) any {
	if v.Type() == colorType {
		return formatThemeColor(graphics.Color(v.Uint()))
	}
	if v.Type() == fontType {
		return encodeFontDescriptor(v)
	}
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return nil
		}
		return encodeValue(v.Elem())
	case reflect.Struct:
		m := make(map[string]any)
		typ := v.Type()
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			if !f.IsExported() {
				continue
			}
			m[jsonName(f.Name)] = encodeValue(v.Field(i))
		}
		return m
	case reflect.Slice:
		if v.Type().Elem().Kind() != reflect.String {
			return nil
		}
		out := make([]string, v.Len())
		for i := range out {
			out[i] = v.Index(i).String()
		}
		return out
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return v.Bool()
	case reflect.Float32, reflect.Float64:
		return v.Float()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint()
	}
	return nil
}

// applyTokenMap writes the exported token dump back onto t. Unknown keys
// are ignored, and absent keys are left alone, so a hand-written partial
// tokens object is also valid.
func applyTokenMap(t *Theme, tokens map[string]any) {
	if t == nil || len(tokens) == 0 {
		return
	}
	applyMapToStruct(reflect.ValueOf(t).Elem(), tokens)
}

func applyMapToStruct(v reflect.Value, m map[string]any) {
	typ := v.Type()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		raw, ok := m[jsonName(f.Name)]
		if !ok {
			continue
		}
		setFromJSON(v.Field(i), raw)
	}
}

func setFromJSON(fv reflect.Value, raw any) {
	if !fv.CanSet() {
		return
	}
	if fv.Type() == colorType {
		if s, ok := raw.(string); ok {
			if c, ok := parseThemeColor(s); ok {
				fv.SetUint(uint64(c))
			}
		}
		return
	}
	if fv.Type() == fontType {
		return // host-resolved; never restored from a file
	}
	switch fv.Kind() {
	case reflect.Ptr:
		if raw == nil {
			return
		}
		m, ok := raw.(map[string]any)
		if !ok {
			return
		}
		if fv.IsNil() {
			fv.Set(reflect.New(fv.Type().Elem()))
		}
		applyMapToStruct(fv.Elem(), m)
	case reflect.Struct:
		if m, ok := raw.(map[string]any); ok {
			applyMapToStruct(fv, m)
		}
	case reflect.Slice:
		if fv.Type().Elem().Kind() != reflect.String {
			return
		}
		out := reflect.MakeSlice(fv.Type(), 0, 0)
		switch xs := raw.(type) {
		case []string:
			for _, s := range xs {
				out = reflect.Append(out, reflect.ValueOf(s))
			}
		case []any:
			for _, e := range xs {
				if s, ok := e.(string); ok {
					out = reflect.Append(out, reflect.ValueOf(s))
				}
			}
		default:
			return
		}
		fv.Set(out)
	case reflect.String:
		if s, ok := raw.(string); ok {
			fv.SetString(s)
		}
	case reflect.Bool:
		if b, ok := raw.(bool); ok {
			fv.SetBool(b)
		}
	case reflect.Float32, reflect.Float64:
		if n, ok := toFloat64(raw); ok {
			fv.SetFloat(n)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if n, ok := toFloat64(raw); ok {
			fv.SetInt(int64(n))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if n, ok := toFloat64(raw); ok {
			fv.SetUint(uint64(n))
		}
	}
}

func toFloat64(raw any) (float64, bool) {
	switch n := raw.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

// jsonName lowercases a Go field name into its JSON key: ID -> "id",
// ControlHeight -> "controlHeight", DX -> "dx".
func jsonName(s string) string {
	i := 0
	for i < len(s) && s[i] >= 'A' && s[i] <= 'Z' {
		i++
	}
	switch {
	case i == 0:
		return s
	case i == len(s):
		return strings.ToLower(s)
	default:
		return strings.ToLower(s[:i]) + s[i:]
	}
}

// themeFilePathMap flattens a theme into "path -> value" strings for
// diffing. Colours render as hex, numbers without trailing zeros.
func themeFilePathMap(t *Theme) map[string]string {
	out := make(map[string]string)
	if t == nil {
		return out
	}
	tf := themeFileFrom(t)
	for k, v := range tf.Palette {
		out["palette."+k] = v
	}
	flattenValue("tokens", tf.Tokens, out)
	// The id is identity, not theme content: a preset and an exported,
	// renamed copy of it should compare equal.
	delete(out, "tokens.meta.id")
	return out
}

func flattenValue(prefix string, v any, out map[string]string) {
	switch x := v.(type) {
	case nil:
		out[prefix] = ""
	case map[string]any:
		for k, e := range x {
			flattenValue(prefix+"."+k, e, out)
		}
	case []string:
		out[prefix] = "[" + strings.Join(x, ", ") + "]"
	case []any:
		parts := make([]string, 0, len(x))
		for _, e := range x {
			parts = append(parts, fmt.Sprint(e))
		}
		out[prefix] = "[" + strings.Join(parts, ", ") + "]"
	case bool:
		out[prefix] = strconv.FormatBool(x)
	case string:
		out[prefix] = x
	case float64:
		out[prefix] = strconv.FormatFloat(x, 'g', -1, 64)
	case float32:
		out[prefix] = strconv.FormatFloat(float64(x), 'g', -1, 32)
	case int:
		out[prefix] = strconv.Itoa(x)
	case int64:
		out[prefix] = strconv.FormatInt(x, 10)
	case uint64:
		out[prefix] = strconv.FormatUint(x, 10)
	default:
		out[prefix] = fmt.Sprint(x)
	}
}

// widgetThemeTarget names one per-widget theme singleton.
type widgetThemeTarget struct {
	name string
	ptr  any // *XxxTheme
}

// widgetThemeTargets lists the 29 per-widget theme singletons.
//
// These structs are GENERATED from the active Theme by the theme engine —
// THEME_ARCHITECTURE.md §1: "29 per-widget structs, GENERATED (never
// hand-edited per preset)" — and widgets hold pointers to them so a theme
// switch can mutate them in place. This registry therefore exists to READ
// and COMPARE them; there is deliberately no loader reversing it into a
// theme file.
func widgetThemeTargets() []widgetThemeTarget {
	return []widgetThemeTarget{
		{"button", DefaultButtonTheme},
		{"checkBox", DefaultCheckBoxTheme},
		{"radioButton", DefaultRadioButtonTheme},
		{"label", DefaultLabelTheme},
		{"textField", DefaultTextFieldTheme},
		{"slider", DefaultSliderTheme},
		{"progressBar", DefaultProgressBarTheme},
		{"tab", DefaultTabTheme},
		{"markdown", DefaultMarkdownTheme},
		{"comboBox", DefaultComboBoxTheme},
		{"listView", DefaultListViewTheme},
		{"scrollBar", DefaultScrollBarTheme},
		{"spinBox", DefaultSpinBoxTheme},
		{"textArea", DefaultTextAreaTheme},
		{"switch", DefaultSwitchTheme},
		{"tooltip", DefaultTooltipTheme},
		{"statusBar", DefaultStatusBarTheme},
		{"toolBar", DefaultToolBarTheme},
		{"menuBar", DefaultMenuBarTheme},
		{"calendar", DefaultCalendarTheme},
		{"colorPicker", DefaultColorPickerTheme},
		{"contextMenu", DefaultContextMenuTheme},
		{"groupBox", DefaultGroupBoxTheme},
		{"scrollPane", DefaultScrollPaneTheme},
		{"table", DefaultTableTheme},
		{"stackPanel", DefaultStackPanelTheme},
		{"messageBox", DefaultMessageBoxTheme},
		{"inputDialog", DefaultInputDialogTheme},
		{"popup", DefaultPopupTheme},
		{"mdi", DefaultMdiTheme},
	}
}

// WidgetThemeNames returns the per-widget theme names, sorted.
func WidgetThemeNames() []string {
	targets := widgetThemeTargets()
	names := make([]string, len(targets))
	for i, t := range targets {
		names[i] = t.name
	}
	sort.Strings(names)
	return names
}

// ResolvedWidgetThemes snapshots the current per-widget theme singletons
// as name -> field map, encoded exactly like the theme file (colours as
// hex, fonts as {family,size}, enums numerically).
//
// This is read-only output. The structs are a pure function of the active
// Theme and Shape and are regenerated on every Apply, so snapshot right
// after loading the theme you care about. To compare two themes, load one,
// snapshot, load the other, then DiffWidgetThemes.
func ResolvedWidgetThemes() map[string]map[string]any {
	targets := widgetThemeTargets()
	out := make(map[string]map[string]any, len(targets))
	for _, wt := range targets {
		if m, ok := encodeValue(reflect.ValueOf(wt.ptr).Elem()).(map[string]any); ok {
			out[wt.name] = m
		}
	}
	return out
}

// encodeFontDescriptor renders a resolved font as {family,size}; the
// underlying FontImpl has no exported fields.
func encodeFontDescriptor(v reflect.Value) any {
	if v.IsNil() {
		return nil
	}
	f, _ := v.Interface().(graphics.Font)
	if f == nil {
		return nil
	}
	m := map[string]any{"size": f.Size()}
	if face := f.Face(); face != nil {
		m["family"] = face.Family()
	}
	return m
}
