package theme

import (
	"fmt"
	"sort"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/slant"
	"github.com/sixnineone/meui/graphics/enums/spacing"
	"github.com/sixnineone/meui/graphics/enums/weight"
)

// weightEnum maps a CSS-style numeric weight to the font enum.
func weightEnum(w int) weight.Enum {
	return graphics.WeightEnum(w)
}

// Platform presets (see THEME_ARCHITECTURE.md). Each preset is a compact
// token table — palette, metrics, typography, motion, effects, chrome —
// that the shared generator expands into all widget themes. Presets
// never touch widget structs directly.
//
// Font stacks are advisory: a preset's family list resolves against
// whatever the app has installed, so e.g. macos-dark on Linux keeps its
// colors/metrics/chrome and falls back through the stack.

// fontResolver resolves a family name at a size to a font. Apps set it
// from their font index (theme.SetFontResolver); when nil the preset
// keeps the previously active BaseFont.
var fontResolver func(family string, size float32) graphics.Font

// SetFontResolver installs the application font lookup used by presets
// to resolve their typography stacks. Return nil for unknown families —
// the preset then falls through to the next family in its stack.
func SetFontResolver(fn func(family string, size float32) graphics.Font) {
	fontResolver = fn
}

// palette is the compact color input for a platform preset. The
// generator reads the standard Theme fields these map onto.
type palette struct {
	accent, accentHover, accentPressed graphics.Color
	onAccent                           graphics.Color
	window, panel, surface, raised     graphics.Color
	sunken                             graphics.Color
	text, textSecondary, textDisabled  graphics.Color
	border, borderSubtle, borderStrong graphics.Color
	hover, pressed, selected           graphics.Color
	success, warning, error            graphics.Color
}

// newTheme expands a palette + token groups into a full Theme.
func newTheme(meta Meta, p palette, m Metrics, mo Motion, ty Typography, fx Effects, ch Chrome) *Theme {
	line := ty.Body.LineHeight
	if line == 0 {
		line = 1.4
	}
	t := &Theme{
		Name:      meta.ID,
		Primary:   p.accent,
		Secondary: p.success,
		Accent:    p.accent,
		OnAccent:  p.onAccent,

		Background:     p.window,
		Surface:        p.surface,
		SurfaceVariant: p.raised,

		TextPrimary:   p.text,
		TextSecondary: p.textSecondary,

		Border:        p.border,
		BorderStrong:  p.borderStrong,
		BorderSubtle:  p.borderSubtle,
		BorderFocused: p.accent,
		BorderHovered: p.accentHover,

		Hovered:  p.hover,
		Pressed:  p.pressed,
		Selected: p.selected,

		Disabled:       p.textDisabled,
		DisabledText:   p.textDisabled,
		DisabledBg:     p.sunken,
		DisabledBorder: p.borderSubtle,

		BorderRadius: m.Shape.Button,

		Success: p.success,
		Warning: p.warning,
		Error:   p.error,
		Shadow:  0x40000000,

		TooltipBackground: p.raised,
		TooltipText:       p.text,

		Link:        p.accent,
		LinkVisited: p.accentPressed,

		MenuBackground: p.raised,
		MenuHighlight:  p.hover,

		ScrollBarTrack: p.sunken,
		ScrollBarThumb: p.borderStrong,
		ScrollbarWidth: m.ScrollbarWidth,

		ChartGrid: p.borderSubtle,
		ChartAxis: p.textSecondary,

		FontSizeSmall:    ty.Caption.Size,
		FontSizeMedium:   ty.Body.Size,
		FontSizeLarge:    ty.Heading.Size,
		FontSizeTitle:    ty.Heading.Size + 2,
		FontWeightNormal: ty.Body.Weight,
		FontWeightBold:   ty.Title.Weight,
		LineHeight:       line,

		Meta:       meta,
		Metrics:    m,
		Motion:     mo,
		Typography: ty,
		Effects:    fx,
		Chrome:     ch,
	}
	switch meta.Platform {
	case PlatformMacOS:
		t.Metrics.SliderTrack, t.Metrics.SliderTrackRadius = 4, -1
		t.Metrics.ProgressHeight, t.Metrics.ProgressRadius = 6, -1
		t.Controls = ControlTokens{
			SliderTrackH: 4, SliderTrackRadius: -1,
			SliderThumbBorder: p.border, ProgressHeight: 6, ProgressRadius: -1,
		}
	case PlatformWindows11:
		t.Metrics.SliderTrack, t.Metrics.SliderTrackRadius = 4, -1
		t.Metrics.ProgressHeight, t.Metrics.ProgressRadius = 4, -1
		t.Controls = ControlTokens{
			SliderTrackH: 4, SliderTrackRadius: -1,
			SliderThumbBorder: p.border, SliderThumbInner: p.accent,
			ProgressHeight: 4, ProgressRadius: -1,
		}
	case PlatformMetro:
		t.Metrics.SliderTrack, t.Metrics.SliderTrackRadius = 4, 0
		t.Metrics.ProgressHeight, t.Metrics.ProgressRadius = 4, 0
		t.Controls = ControlTokens{
			SliderTrackH: 4, SliderTrackRadius: 0,
			SliderThumbInner: p.accent,
			ProgressHeight:   4, ProgressRadius: 0,
		}
	case PlatformGNOME:
		t.Metrics.SliderTrack, t.Metrics.SliderTrackRadius = 6, -1
		t.Metrics.ProgressHeight, t.Metrics.ProgressRadius = 6, -1
		t.Controls = ControlTokens{
			SliderTrackH: 6, SliderTrackRadius: -1,
			SliderThumbBorder: p.borderStrong, ProgressHeight: 6, ProgressRadius: -1,
		}
	case PlatformKDE:
		t.Metrics.SliderTrack, t.Metrics.SliderTrackRadius = 4, -1
		t.Metrics.ProgressHeight, t.Metrics.ProgressRadius = 4, -1
		t.Controls = ControlTokens{
			SliderTrackH: 4, SliderTrackRadius: -1, SliderTrackBorder: true,
			SliderThumbBorder: p.border, ProgressHeight: 4, ProgressRadius: -1,
		}
	}
	if meta.Platform == PlatformWindows11 {
		// Fluent TextBox: emphasised bottom edge (1px resting, 2px accent on focus).
		t.Inputs = InputTokens{
			Style:            InputOutlined,
			BottomInk:        p.borderStrong,
			BottomWidth:      1,
			FocusBottomWidth: 2,
		}
	}
	if t.FontWeightNormal == 0 {
		t.FontWeightNormal = 400
	}
	if t.FontWeightBold == 0 {
		t.FontWeightBold = 700
	}
	return t
}

// resolveBaseFont picks the preset's BaseFont through the app font
// resolver, the built-in matcher, or preserves the active face.
func resolveBaseFont(t *Theme) {
	if fontResolver != nil {
		for _, fam := range t.Typography.Families {
			if f := fontResolver(fam, t.Typography.Body.Size); f != nil {
				t.BaseFont = f
				return
			}
		}
	}
	for _, fam := range t.Typography.Families {
		if face := graphics.MatchFontFace(fam, weightEnum(t.Typography.Body.Weight), spacing.Standard, slant.Upright); face != nil {
			t.BaseFont = face.Font(t.Typography.Body.Size)
			return
		}
	}
	// Preserve the app's current face — switching platform themes must
	// not blank the text.
	if DefaultTheme != nil && DefaultTheme.BaseFont != nil {
		t.BaseFont = DefaultTheme.BaseFont
		return
	}
	t.BaseFont = graphics.SystemFont
}

// ── Shape profiles ─────────────────────────────────────────────

func shapeMacOS() *ShapeTheme {
	return &ShapeTheme{
		Window: 10, Popup: 12, Menu: 10, Tooltip: 8,
		Button: 6, CheckBox: 4, Input: 6, ComboBox: 6, Tab: 6,
		GroupBox: 10, ListRow: 6, ScrollBar: 4,
	}
}

func shapeAdwaita() *ShapeTheme {
	return &ShapeTheme{
		Window: 12, Popup: 12, Menu: 12, Tooltip: 8,
		Button: 6, CheckBox: 4, Input: 6, ComboBox: 6, Tab: 6,
		GroupBox: 12, ListRow: 6, ScrollBar: 8,
	}
}

func shapeBreeze() *ShapeTheme {
	return &ShapeTheme{
		Window: 4, Popup: 4, Menu: 3, Tooltip: 3,
		Button: 3, CheckBox: 2, Input: 3, ComboBox: 3, Tab: 2,
		GroupBox: 3, ListRow: 2, ScrollBar: 3,
	}
}

// ── Metrics ────────────────────────────────────────────────────

func metricsMacOS() Metrics {
	m := DefaultMetrics()
	m.Density = DensityRegular
	m.ControlHeight = 28
	m.ControlPadX = 10
	m.ControlPadY = 4
	m.InputPadX = 8
	m.MenuItemHeight = 24
	m.ListRowHeight = 28
	m.TabHeight = 28
	m.ToolbarHeight = 38
	m.StatusBarHeight = 24
	m.ScrollbarWidth = 8
	m.ScrollbarOverlay = true
	m.ScrollbarAutoHide = true
	m.BorderWidth = 1
	m.FocusRingWidth = 3
	m.FocusRingOffset = 1
	m.BoxSize = 16
	m.BoxSpacing = 6
	m.SliderThumb = 20
	m.SwitchH = 22
	m.ArrowSize = 10
	m.GripSize = 4
	m.ControlBorderWidth = 1
	m.TabIndicatorHeight = 0
	m.TabActiveFilled = true
	m.ArrowStyle = 1
	m.Shape = shapeMacOS()
	return m
}

func metricsWindows11() Metrics {
	m := DefaultMetrics()
	m.Density = DensityRegular
	m.ControlHeight = 32
	m.ControlPadX = 12
	m.ControlPadY = 4
	m.InputPadX = 12
	m.MenuItemHeight = 32
	m.ListRowHeight = 32
	m.TabHeight = 32
	m.ToolbarHeight = 40
	m.StatusBarHeight = 24
	m.ScrollbarWidth = 12
	m.BorderWidth = 1
	m.FocusRingWidth = 2
	m.FocusRingOffset = 0
	m.BoxSize = 20
	m.BoxSpacing = 8
	m.SliderThumb = 20
	m.SwitchH = 20
	m.ArrowSize = 12
	m.GripSize = 4
	m.ControlBorderWidth = 1
	m.TabIndicatorHeight = 2.5
	m.TabActiveFilled = false
	m.ArrowStyle = 1
	m.Shape = ShapeWindows11()
	return m
}

func metricsMetro() Metrics {
	m := DefaultMetrics()
	m.Density = DensityRegular
	m.ControlHeight = 32
	m.ControlPadX = 12
	m.ControlPadY = 4
	m.InputPadX = 12
	m.MenuItemHeight = 32
	m.ListRowHeight = 36
	m.TabHeight = 36
	m.ToolbarHeight = 40
	m.StatusBarHeight = 22
	m.ScrollbarWidth = 17
	m.BorderWidth = 2
	m.FocusRingWidth = 2
	m.BoxSize = 20
	m.BoxSpacing = 8
	m.SliderThumb = 20
	m.SwitchH = 20
	m.ArrowSize = 10
	m.GripSize = 4
	m.ControlBorderWidth = 0
	m.TabIndicatorHeight = 2
	m.TabActiveFilled = false
	m.ArrowStyle = 0
	m.Shape = ShapeFlat()
	return m
}

func metricsAdwaita() Metrics {
	m := DefaultMetrics()
	m.Density = DensityComfortable
	m.ControlHeight = 34
	m.ControlPadX = 12
	m.ControlPadY = 6
	m.InputPadX = 12
	m.MenuItemHeight = 32
	m.ListRowHeight = 34
	m.TabHeight = 34
	m.ToolbarHeight = 46
	m.StatusBarHeight = 24
	m.ScrollbarWidth = 8
	m.ScrollbarOverlay = true
	m.ScrollbarAutoHide = true
	m.BorderWidth = 0
	m.FocusRingWidth = 3
	m.FocusRingOffset = 1
	m.BoxSize = 16
	m.BoxSpacing = 6
	m.SliderThumb = 20
	m.SwitchH = 24
	m.ArrowSize = 10
	m.GripSize = 4
	m.ControlBorderWidth = 1
	m.TabIndicatorHeight = 0
	m.TabActiveFilled = true
	m.ArrowStyle = 1
	m.Shape = shapeAdwaita()
	return m
}

func metricsBreeze() Metrics {
	m := DefaultMetrics()
	m.Density = DensityCompact
	m.ControlHeight = 26
	m.ControlPadX = 8
	m.ControlPadY = 3
	m.InputPadX = 8
	m.MenuItemHeight = 24
	m.ListRowHeight = 26
	m.TabHeight = 28
	m.ToolbarHeight = 32
	m.StatusBarHeight = 22
	m.ScrollbarWidth = 12
	m.BorderWidth = 1
	m.FocusRingWidth = 2
	m.FocusRingOffset = 0
	m.BoxSize = 16
	m.BoxSpacing = 4
	m.SliderThumb = 16
	m.SwitchH = 18
	m.ArrowSize = 8
	m.GripSize = 3
	m.ControlBorderWidth = 1
	m.TabIndicatorHeight = 0
	m.TabActiveFilled = true
	m.ArrowStyle = 1
	m.Shape = shapeBreeze()
	return m
}

// ── Motion ─────────────────────────────────────────────────────

func motionMacOS() Motion {
	return Motion{
		Hover:       Timing{120, EaseStandard},
		Press:       Timing{80, EaseStandard},
		Focus:       Timing{120, EaseStandard},
		Toggle:      Timing{150, EaseStandard},
		MenuOpen:    Timing{150, EaseDecelerate},
		MenuClose:   Timing{100, EaseAccelerate},
		TooltipShow: Timing{250, EaseDecelerate},
		TooltipHide: Timing{100, EaseStandard},
		ScrollFade:  Timing{600, EaseDecelerate},
	}
}

func motionWindows11() Motion {
	return Motion{
		Hover:       Timing{167, EaseStandard},
		Press:       Timing{83, EaseStandard},
		Focus:       Timing{167, EaseStandard},
		Toggle:      Timing{167, EaseStandard},
		MenuOpen:    Timing{200, EaseDecelerate},
		MenuClose:   Timing{150, EaseAccelerate},
		TooltipShow: Timing{300, EaseDecelerate},
		TooltipHide: Timing{100, EaseStandard},
		ScrollFade:  Timing{500, EaseDecelerate},
	}
}

func motionMetro() Motion {
	return Motion{
		Hover:       Timing{100, EaseStandard},
		Press:       Timing{50, EaseLinear},
		Focus:       Timing{100, EaseStandard},
		Toggle:      Timing{100, EaseStandard},
		MenuOpen:    Timing{150, EaseDecelerate},
		MenuClose:   Timing{100, EaseAccelerate},
		TooltipShow: Timing{300, EaseDecelerate},
		TooltipHide: Timing{80, EaseStandard},
		ScrollFade:  Timing{400, EaseDecelerate},
	}
}

func motionAdwaita() Motion {
	return Motion{
		Hover:       Timing{200, EaseStandard},
		Press:       Timing{100, EaseStandard},
		Focus:       Timing{200, EaseStandard},
		Toggle:      Timing{200, EaseStandard},
		MenuOpen:    Timing{150, EaseDecelerate},
		MenuClose:   Timing{120, EaseAccelerate},
		TooltipShow: Timing{500, EaseDecelerate},
		TooltipHide: Timing{150, EaseStandard},
		ScrollFade:  Timing{400, EaseDecelerate},
	}
}

func motionBreeze() Motion {
	return Motion{
		Hover:       Timing{150, EaseStandard},
		Press:       Timing{80, EaseStandard},
		Focus:       Timing{150, EaseStandard},
		Toggle:      Timing{150, EaseStandard},
		MenuOpen:    Timing{150, EaseDecelerate},
		MenuClose:   Timing{100, EaseAccelerate},
		TooltipShow: Timing{400, EaseDecelerate},
		TooltipHide: Timing{100, EaseStandard},
		ScrollFade:  Timing{500, EaseDecelerate},
	}
}

// ── Typography ─────────────────────────────────────────────────

// typoMeui is the built-in look (Inter-class UI font at compact sizes).
func typoMeui() Typography {
	return Typography{
		Families: []string{"Inter", "Inter Display", "Roboto"},
		Caption:  TextStyle{Size: 9, Weight: 400, LineHeight: 1.35},
		Body:     TextStyle{Size: 11, Weight: 400, LineHeight: 1.4},
		Label:    TextStyle{Size: 11, Weight: 400, LineHeight: 1.4},
		Button:   TextStyle{Size: 11, Weight: 400, LineHeight: 1.4},
		Title:    TextStyle{Size: 12, Weight: 700, LineHeight: 1.3},
		Heading:  TextStyle{Size: 14, Weight: 700, LineHeight: 1.3},
		Mono:     TextStyle{Size: 11, Weight: 400, LineHeight: 1.4},
	}
}

func typoMacOS() Typography {
	return Typography{
		Families: []string{"SF Pro Text", "SF Pro Display", "Helvetica Neue", "Inter", "Roboto"},
		Caption:  TextStyle{Size: 11, Weight: 400, LineHeight: 1.3},
		Body:     TextStyle{Size: 13, Weight: 400, LineHeight: 1.35},
		Label:    TextStyle{Size: 13, Weight: 400, LineHeight: 1.35},
		Button:   TextStyle{Size: 13, Weight: 400, LineHeight: 1.35},
		Title:    TextStyle{Size: 13, Weight: 600, LineHeight: 1.3},
		Heading:  TextStyle{Size: 15, Weight: 600, LineHeight: 1.3},
		Mono:     TextStyle{Size: 12, Weight: 400, LineHeight: 1.4},
	}
}

func typoWindows11() Typography {
	return Typography{
		Families: []string{"Segoe UI Variable Text", "Segoe UI", "Inter", "Roboto"},
		Caption:  TextStyle{Size: 12, Weight: 400, LineHeight: 1.35},
		Body:     TextStyle{Size: 14, Weight: 400, LineHeight: 1.4},
		Label:    TextStyle{Size: 14, Weight: 400, LineHeight: 1.4},
		Button:   TextStyle{Size: 14, Weight: 400, LineHeight: 1.4},
		Title:    TextStyle{Size: 12, Weight: 600, LineHeight: 1.3},
		Heading:  TextStyle{Size: 20, Weight: 600, LineHeight: 1.25},
		Mono:     TextStyle{Size: 13, Weight: 400, LineHeight: 1.4},
	}
}

func typoMetro() Typography {
	return Typography{
		Families: []string{"Segoe UI Light", "Segoe UI", "Inter", "Roboto"},
		Caption:  TextStyle{Size: 12, Weight: 300, LineHeight: 1.35},
		Body:     TextStyle{Size: 15, Weight: 300, LineHeight: 1.4},
		Label:    TextStyle{Size: 15, Weight: 400, LineHeight: 1.4},
		Button:   TextStyle{Size: 15, Weight: 300, LineHeight: 1.4},
		Title:    TextStyle{Size: 20, Weight: 300, LineHeight: 1.25},
		Heading:  TextStyle{Size: 24, Weight: 300, LineHeight: 1.2},
		Mono:     TextStyle{Size: 14, Weight: 400, LineHeight: 1.4},
	}
}

func typoAdwaita() Typography {
	return Typography{
		Families: []string{"Adwaita Sans", "Cantarell", "Inter", "Roboto"},
		Caption:  TextStyle{Size: 12, Weight: 400, LineHeight: 1.35},
		Body:     TextStyle{Size: 14, Weight: 400, LineHeight: 1.45},
		Label:    TextStyle{Size: 14, Weight: 400, LineHeight: 1.45},
		Button:   TextStyle{Size: 14, Weight: 500, LineHeight: 1.45},
		Title:    TextStyle{Size: 12, Weight: 700, LineHeight: 1.3},
		Heading:  TextStyle{Size: 17, Weight: 700, LineHeight: 1.25},
		Mono:     TextStyle{Size: 13, Weight: 400, LineHeight: 1.4},
	}
}

func typoBreeze() Typography {
	return Typography{
		Families: []string{"Noto Sans", "Inter", "Roboto"},
		Caption:  TextStyle{Size: 11, Weight: 400, LineHeight: 1.35},
		Body:     TextStyle{Size: 13, Weight: 400, LineHeight: 1.4},
		Label:    TextStyle{Size: 13, Weight: 400, LineHeight: 1.4},
		Button:   TextStyle{Size: 13, Weight: 400, LineHeight: 1.4},
		Title:    TextStyle{Size: 12, Weight: 700, LineHeight: 1.3},
		Heading:  TextStyle{Size: 16, Weight: 700, LineHeight: 1.25},
		Mono:     TextStyle{Size: 12, Weight: 400, LineHeight: 1.4},
	}
}

// ── Effects ────────────────────────────────────────────────────

func effectsMacOS(accent graphics.Color) Effects {
	return Effects{
		WindowShadow: ShadowSpec{Color: 0x80000000, DY: 12, Blur: 32},
		PopupShadow:  ShadowSpec{Color: 0x66000000, DY: 8, Blur: 24},
		MenuShadow:   ShadowSpec{Color: 0x66000000, DY: 8, Blur: 24},
		Focus:        FocusStyle{Kind: FocusRing, Width: 3, Offset: 1, Color: accent},
		Material:     MaterialVibrancy,
	}
}

func effectsWindows11(accent graphics.Color) Effects {
	return Effects{
		WindowShadow: ShadowSpec{Color: 0x40000000, DY: 4, Blur: 16},
		PopupShadow:  ShadowSpec{Color: 0x40000000, DY: 4, Blur: 16},
		MenuShadow:   ShadowSpec{Color: 0x40000000, DY: 4, Blur: 16},
		Focus:        FocusStyle{Kind: FocusTwoTone, Width: 2, Color: 0xFFFFFFFF, Alt: 0xFF000000},
		Material:     MaterialMica,
	}
}

func effectsMetro(accent graphics.Color) Effects {
	return Effects{
		// Metro is deliberately shadowless and flat.
		Focus:    FocusStyle{Kind: FocusInset, Width: 2, Color: accent},
		Material: MaterialNone,
	}
}

func effectsAdwaita(accent graphics.Color) Effects {
	return Effects{
		WindowShadow: ShadowSpec{Color: 0x50000000, DY: 8, Blur: 20},
		PopupShadow:  ShadowSpec{Color: 0x50000000, DY: 8, Blur: 20},
		MenuShadow:   ShadowSpec{Color: 0x50000000, DY: 8, Blur: 20},
		Focus:        FocusStyle{Kind: FocusRing, Width: 3, Offset: 1, Color: accent},
		Material:     MaterialTranslucent,
	}
}

func effectsBreeze(accent graphics.Color) Effects {
	return Effects{
		PopupShadow: ShadowSpec{Color: 0x40000000, DY: 4, Blur: 12},
		MenuShadow:  ShadowSpec{Color: 0x40000000, DY: 4, Blur: 12},
		Focus:       FocusStyle{Kind: FocusInset, Width: 1, Color: accent},
		Material:    MaterialNone,
	}
}

// ── Chrome ─────────────────────────────────────────────────────

func chromeMacOS() Chrome {
	return Chrome{
		Style:        ChromeMacOS,
		Height:       28,
		ButtonW:      12,
		ButtonH:      12,
		ButtonGap:    8,
		HoverBg:      0x14FFFFFF,
		PressedBg:    0x22FFFFFF,
		CloseHoverBg: 0xFFFF5F57,
		GlyphColor:   0xFF98989D,
		TitleAlign:   TitleCenter,
		TitleWeight:  600,
		BorderWidth:  0, // macOS windows have no outline
	}
}

func chromeWindows11() Chrome {
	return Chrome{
		Style:        ChromeWindows,
		Height:       32,
		ButtonW:      46,
		ButtonH:      32,
		ButtonGap:    0,
		HoverBg:      0x0FFFFFFF,
		PressedBg:    0x08FFFFFF,
		CloseHoverBg: 0xFFC42B1C,
		GlyphColor:   0xFFFFFFFF,
		TitleAlign:   TitleLeft,
		TitleWeight:  600,
		BorderWidth:  1,
	}
}

func chromeMetro() Chrome {
	return Chrome{
		Style:        ChromeMetro,
		Height:       32,
		ButtonW:      46,
		ButtonH:      32,
		ButtonGap:    0,
		HoverBg:      0x14000000,
		PressedBg:    0x22000000,
		CloseHoverBg: 0xFFC42B1C,
		GlyphColor:   0xFF000000,
		TitleAlign:   TitleLeft,
		TitleWeight:  300,
		BorderWidth:  0, // Metro is flat and borderless
	}
}

func chromeGnome() Chrome {
	return Chrome{
		Style:        ChromeGNOME,
		Height:       47,
		ButtonW:      24,
		ButtonH:      24,
		ButtonGap:    6,
		HoverBg:      0x14FFFFFF,
		PressedBg:    0x22FFFFFF,
		CloseHoverBg: 0xFFE01B24,
		GlyphColor:   0xFFC0BFBC,
		TitleAlign:   TitleCenter,
		TitleWeight:  700,
		BorderWidth:  1,
	}
}

func chromeKDE() Chrome {
	return Chrome{
		Style:        ChromeKDE,
		Height:       30,
		ButtonW:      24,
		ButtonH:      24,
		ButtonGap:    4,
		HoverBg:      0x1FFFFFFF,
		PressedBg:    0x33FFFFFF,
		CloseHoverBg: 0xFFDA4453,
		GlyphColor:   0xFFEFF0F1,
		TitleAlign:   TitleCenter,
		TitleWeight:  700,
		BorderWidth:  1,
	}
}

// ── Platform palettes ──────────────────────────────────────────

func macosPalette(dark bool) palette {
	if dark {
		return palette{
			accent: 0xFF0A84FF, accentHover: 0xFF2492FF, accentPressed: 0xFF0060DF, onAccent: 0xFFFFFFFF,
			window: 0xFF1E1E1E, panel: 0xFF252525, surface: 0xFF2C2C2E, raised: 0xFF3A3A3C, sunken: 0xFF1C1C1E,
			text: 0xFFFFFFFF, textSecondary: 0xFF98989D, textDisabled: 0xFF5A5A5E,
			border: 0xFF3A3A3C, borderSubtle: 0xFF2C2C2E, borderStrong: 0xFF48484A,
			hover: 0x14FFFFFF, pressed: 0x22FFFFFF, selected: 0xFF0A84FF,
			success: 0xFF30D158, warning: 0xFFFFD60A, error: 0xFFFF453A,
		}
	}
	return palette{
		accent: 0xFF007AFF, accentHover: 0xFF147AFF, accentPressed: 0xFF0062CC, onAccent: 0xFFFFFFFF,
		window: 0xFFECECEC, panel: 0xFFF5F5F5, surface: 0xFFFFFFFF, raised: 0xFFFFFFFF, sunken: 0xFFE8E8E8,
		text: 0xFF1D1D1F, textSecondary: 0xFF6E6E73, textDisabled: 0xFFB0B0B5,
		border: 0xFFD1D1D6, borderSubtle: 0xFFE5E5EA, borderStrong: 0xFFC7C7CC,
		hover: 0x0F000000, pressed: 0x1A000000, selected: 0xFF007AFF,
		success: 0xFF34C759, warning: 0xFFFF9500, error: 0xFFFF3B30,
	}
}

func windows11Palette(dark bool) palette {
	if dark {
		return palette{
			accent: 0xFF60CDFF, accentHover: 0xFF52B8E8, accentPressed: 0xFF4AA8D8, onAccent: 0xFF000000,
			window: 0xFF202020, panel: 0xFF202020, surface: 0xFF2B2B2B, raised: 0xFF2D2D2D, sunken: 0xFF1A1A1A,
			text: 0xFFFFFFFF, textSecondary: 0xFFC5C5C5, textDisabled: 0xFF6E6E6E,
			border: 0xFF3A3A3A, borderSubtle: 0xFF2E2E2E, borderStrong: 0xFF757575,
			hover: 0x0FFFFFFF, pressed: 0x0AFFFFFF, selected: 0x3360CDFF,
			success: 0xFF6CCB5F, warning: 0xFFFCE100, error: 0xFFFF99A4,
		}
	}
	return palette{
		accent: 0xFF0067C0, accentHover: 0xFF1975C5, accentPressed: 0xFF005BA1, onAccent: 0xFFFFFFFF,
		window: 0xFFF3F3F3, panel: 0xFFF3F3F3, surface: 0xFFFFFFFF, raised: 0xFFFFFFFF, sunken: 0xFFEBEBEB,
		text: 0xFF1B1B1B, textSecondary: 0xFF5F5F5F, textDisabled: 0xFF9C9C9C,
		border: 0xFFE5E5E5, borderSubtle: 0xFFEBEBEB, borderStrong: 0xFF8A8A8A,
		hover: 0x0A000000, pressed: 0x06000000, selected: 0x330067C0,
		success: 0xFF0F7B0F, warning: 0xFF9D5D00, error: 0xFFC42B1C,
	}
}

func metroPalette() palette {
	return palette{
		accent: 0xFF2672EC, accentHover: 0xFF1E5FC4, accentPressed: 0xFF1853AE, onAccent: 0xFFFFFFFF,
		window: 0xFFFFFFFF, panel: 0xFFFFFFFF, surface: 0xFFFFFFFF, raised: 0xFFFFFFFF, sunken: 0xFFF2F2F2,
		text: 0xFF000000, textSecondary: 0xFF5A5A5A, textDisabled: 0xFFB1B1B1,
		border: 0xFFC4C4C4, borderSubtle: 0xFFE0E0E0, borderStrong: 0xFF8A8A8A,
		hover: 0x10000000, pressed: 0x20000000, selected: 0xFF2672EC,
		success: 0xFF107C10, warning: 0xFFE5A00D, error: 0xFFDA3B01,
	}
}

func adwaitaPalette(dark bool) palette {
	if dark {
		return palette{
			accent: 0xFF3584E4, accentHover: 0xFF4A90E8, accentPressed: 0xFF1C71D8, onAccent: 0xFFFFFFFF,
			window: 0xFF303030, panel: 0xFF303030, surface: 0xFF242424, raised: 0xFF3D3D3D, sunken: 0xFF1A1A1A,
			text: 0xFFFFFFFF, textSecondary: 0xFFB0B0B0, textDisabled: 0xFF6F6F6F,
			border: 0xFF3D3D3D, borderSubtle: 0xFF2E2E2E, borderStrong: 0xFF5A5A5A,
			hover: 0x12FFFFFF, pressed: 0x1FFFFFFF, selected: 0xFF3584E4,
			success: 0xFF33D17A, warning: 0xFFF5C211, error: 0xFFFF7B63,
		}
	}
	return palette{
		accent: 0xFF3584E4, accentHover: 0xFF4A90E8, accentPressed: 0xFF1C71D8, onAccent: 0xFFFFFFFF,
		window: 0xFFEBEBEB, panel: 0xFFF6F5F4, surface: 0xFFFFFFFF, raised: 0xFFFFFFFF, sunken: 0xFFEFEFEF,
		text: 0xFF2E3436, textSecondary: 0xFF5E5C64, textDisabled: 0xFF9A9996,
		border: 0xFFCDCDCD, borderSubtle: 0xFFE0E0E0, borderStrong: 0xFFB0B0B0,
		hover: 0x0F000000, pressed: 0x1F000000, selected: 0xFF3584E4,
		success: 0xFF2EC27E, warning: 0xFFF5C211, error: 0xFFE01B24,
	}
}

func breezePalette(dark bool) palette {
	if dark {
		return palette{
			accent: 0xFF3DAEE9, accentHover: 0xFF4CB8EE, accentPressed: 0xFF2980B9, onAccent: 0xFFFFFFFF,
			window: 0xFF31363B, panel: 0xFF31363B, surface: 0xFF232629, raised: 0xFF3B4045, sunken: 0xFF1B1E20,
			text: 0xFFEFF0F1, textSecondary: 0xFFBABEC2, textDisabled: 0xFF7A7E82,
			border: 0xFF4D4D4D, borderSubtle: 0xFF3A3F44, borderStrong: 0xFF5A6065,
			hover: 0x14FFFFFF, pressed: 0x22FFFFFF, selected: 0xFF3DAEE9,
			success: 0xFF27AE60, warning: 0xFFF67400, error: 0xFFDA4453,
		}
	}
	return palette{
		accent: 0xFF3DAEE9, accentHover: 0xFF4CB8EE, accentPressed: 0xFF2980B9, onAccent: 0xFFFFFFFF,
		window: 0xFFEFF0F1, panel: 0xFFEFF0F1, surface: 0xFFFFFFFF, raised: 0xFFFFFFFF, sunken: 0xFFE4E6E8,
		text: 0xFF232629, textSecondary: 0xFF5C5F62, textDisabled: 0xFF9EA2A6,
		border: 0xFFBFC3C7, borderSubtle: 0xFFD7DADD, borderStrong: 0xFFA0A4A8,
		hover: 0x0F000000, pressed: 0x1F000000, selected: 0xFF3DAEE9,
		success: 0xFF27AE60, warning: 0xFFF67400, error: 0xFFDA4453,
	}
}

// ── Registry ───────────────────────────────────────────────────

var presetBuilders = map[string]func() *Theme{
	"meui-dark": func() *Theme {
		t := DarkTheme()
		t.Name = "meui-dark"
		t.Meta = Meta{ID: "meui-dark", Platform: PlatformMeui, Mode: ModeDark}
		return t
	},
	"meui-light": func() *Theme {
		t := LightTheme()
		t.Name = "meui-light"
		t.Meta = Meta{ID: "meui-light", Platform: PlatformMeui, Mode: ModeLight}
		return t
	},
	"macos-dark": func() *Theme {
		p := macosPalette(true)
		return newTheme(Meta{"macos-dark", PlatformMacOS, ModeDark}, p,
			metricsMacOS(), motionMacOS(), typoMacOS(), effectsMacOS(p.accent), chromeMacOS())
	},
	"macos-light": func() *Theme {
		p := macosPalette(false)
		return newTheme(Meta{"macos-light", PlatformMacOS, ModeLight}, p,
			metricsMacOS(), motionMacOS(), typoMacOS(), effectsMacOS(p.accent), chromeMacOS())
	},
	"windows11-dark": func() *Theme {
		p := windows11Palette(true)
		return newTheme(Meta{"windows11-dark", PlatformWindows11, ModeDark}, p,
			metricsWindows11(), motionWindows11(), typoWindows11(), effectsWindows11(p.accent), chromeWindows11())
	},
	"windows11-light": func() *Theme {
		p := windows11Palette(false)
		return newTheme(Meta{"windows11-light", PlatformWindows11, ModeLight}, p,
			metricsWindows11(), motionWindows11(), typoWindows11(), effectsWindows11(p.accent), chromeWindows11())
	},
	"metro": func() *Theme {
		p := metroPalette()
		return newTheme(Meta{"metro", PlatformMetro, ModeLight}, p,
			metricsMetro(), motionMetro(), typoMetro(), effectsMetro(p.accent), chromeMetro())
	},
	"adwaita-dark": func() *Theme {
		p := adwaitaPalette(true)
		return newTheme(Meta{"adwaita-dark", PlatformGNOME, ModeDark}, p,
			metricsAdwaita(), motionAdwaita(), typoAdwaita(), effectsAdwaita(p.accent), chromeGnome())
	},
	"adwaita-light": func() *Theme {
		p := adwaitaPalette(false)
		return newTheme(Meta{"adwaita-light", PlatformGNOME, ModeLight}, p,
			metricsAdwaita(), motionAdwaita(), typoAdwaita(), effectsAdwaita(p.accent), chromeGnome())
	},
	"breeze-dark": func() *Theme {
		p := breezePalette(true)
		return newTheme(Meta{"breeze-dark", PlatformKDE, ModeDark}, p,
			metricsBreeze(), motionBreeze(), typoBreeze(), effectsBreeze(p.accent), chromeKDE())
	},
	"breeze-light": func() *Theme {
		p := breezePalette(false)
		return newTheme(Meta{"breeze-light", PlatformKDE, ModeLight}, p,
			metricsBreeze(), motionBreeze(), typoBreeze(), effectsBreeze(p.accent), chromeKDE())
	},
}

// PresetIDs returns the registered native preset IDs, sorted.
func PresetIDs() []string {
	presetMu.RLock()
	ids := make([]string, 0, len(presetBuilders))
	for id := range presetBuilders {
		ids = append(ids, id)
	}
	presetMu.RUnlock()
	sort.Strings(ids)
	return ids
}

// presetTweaks run AFTER a preset's style defaults are generated. They exist because a
// widget's generated theme is derived from tokens and then applied — a preset that
// wants a control to look different (a web primary Button, a white input) has to say so
// after generation, or the generator overwrites it.
var presetTweaks = map[string]func(*Theme){}

func registerPresetTweak(id string, fn func(*Theme)) {
	presetTweaks[id] = fn
}

// LoadPreset installs a native platform preset by ID (see PresetIDs).
// It sets the shape first (the generator reads the global shape), then
// runs the single Apply path.
func LoadPreset(id string) error {
	presetMu.RLock()
	build, ok := presetBuilders[id]
	presetMu.RUnlock()
	if !ok {
		return fmt.Errorf("theme: unknown preset %q (want one of %v)", id, PresetIDs())
	}
	t := build()
	t.withStyleDefaults() // resolve token groups before shape + generation
	resolveBaseFont(t)
	SetShape(t.Metrics.Shape)
	Apply(t)
	// AFTER Apply: the generator rebuilds every widget theme from the tokens, so a
	// tweak that ran before it was overwritten (the web Button came out as a plain
	// field for exactly that reason).
	if tweak := presetTweaks[id]; tweak != nil {
		tweak(t)
	}
	return nil
}

// ── Mobile presets ─────────────────────────────────────────────
//
// The agreed decision is ADAPT, not imitate: one widget set, our token vocabulary,
// and values that are appropriate on a phone rather than a copy of another vendor's
// design system. So these are meui presets whose METRICS, SHAPE, TYPE and MOTION are
// tuned for a finger, with palettes that follow each platform's conventions (Android's
// tonal surfaces, iOS's grouped greys) without cloning its token tables.
//
// What actually makes them mobile is three numbers: ControlHeight reaches the touch
// floor, ListRowHeight reaches it too, and MinTouchTarget is set so the hit test can
// enforce it on anything that is still too small.

// Android's own floor is 48dp; iOS's is 44pt. Both are published numbers, and they are
// the reason this is a metric rather than a constant.
const (
	TouchTargetAndroid = float32(48)
	TouchTargetIOS     = float32(44)
)

func shapeAndroid() *ShapeTheme {
	// M3 rounds generously and consistently: a large radius on containers, pills for
	// anything that reads as a control.
	return &ShapeTheme{
		Window: 0, Popup: 16, Menu: 8, Tooltip: 8,
		Button: 20, CheckBox: 4, Input: 12, ComboBox: 12, Tab: 16,
		GroupBox: 16, ListRow: 0, ScrollBar: 8,
	}
}

func shapeIOS() *ShapeTheme {
	// iOS is rounder on controls and squarer on containers than M3, and a grouped
	// list has square rows inside a rounded card.
	return &ShapeTheme{
		Window: 0, Popup: 14, Menu: 14, Tooltip: 8,
		Button: 10, CheckBox: 6, Input: 10, ComboBox: 10, Tab: 8,
		GroupBox: 10, ListRow: 0, ScrollBar: 4,
	}
}

func metricsAndroid() Metrics {
	m := DefaultMetrics()
	m.Density = DensityComfortable
	// The floor, everywhere a control is drawn.
	m.ControlHeight = TouchTargetAndroid
	m.ControlPadX = 16
	m.ControlPadY = 8
	m.InputPadX = 16
	m.MenuItemHeight = TouchTargetAndroid
	m.ListRowHeight = 56 // M3's two-line list item
	m.TabHeight = 64     // the bottom navigation bar
	m.ToolbarHeight = 64 // the top app bar
	m.StatusBarHeight = 24
	m.ScrollbarWidth = 6
	m.ScrollbarOverlay = true
	m.ScrollbarAutoHide = true
	m.BorderWidth = 1
	m.FocusRingWidth = 3
	m.FocusRingOffset = 2
	m.BoxSize = 24
	m.BoxSpacing = 12
	m.SliderThumb = 24
	m.SwitchH = 32
	m.ArrowSize = 12
	m.GripSize = 4
	m.ControlBorderWidth = 0 // M3 buttons are filled, not outlined, by default
	m.TabIndicatorHeight = 0
	m.TabActiveFilled = true
	m.ArrowStyle = 1
	m.ProgressHeight = 4
	m.ProgressRadius = -1
	m.NavBarHeight = 64 // the Material bottom bar
	m.NavRailWidth = 80 // the Material rail
	m.MinTouchTarget = TouchTargetAndroid
	m.Shape = shapeAndroid()
	return m
}

func metricsIOS() Metrics {
	m := DefaultMetrics()
	m.Density = DensityComfortable
	m.ControlHeight = TouchTargetIOS
	m.ControlPadX = 16
	m.ControlPadY = 8
	m.InputPadX = 12
	m.MenuItemHeight = TouchTargetIOS
	m.ListRowHeight = TouchTargetIOS
	m.TabHeight = 49 // iOS's tab bar
	m.ToolbarHeight = 44
	m.StatusBarHeight = 20
	m.ScrollbarWidth = 4
	m.ScrollbarOverlay = true
	m.ScrollbarAutoHide = true
	m.BorderWidth = 1
	m.FocusRingWidth = 3
	m.FocusRingOffset = 2
	m.BoxSize = 22
	m.BoxSpacing = 10
	m.SliderThumb = 28
	m.SwitchH = 31 // iOS's switch is 51x31
	m.ArrowSize = 12
	m.GripSize = 4
	m.ControlBorderWidth = 0
	m.TabIndicatorHeight = 0
	m.TabActiveFilled = true
	m.ArrowStyle = 1
	m.ProgressHeight = 4
	m.ProgressRadius = -1
	m.NavBarHeight = 49 // the iOS tab bar
	m.NavRailWidth = 88 // iOS puts its sidebar a little wider
	m.MinTouchTarget = TouchTargetIOS
	m.Shape = shapeIOS()
	return m
}

// androidPalette follows Android's tonal convention: surfaces are near-neutral with a
// faint tint of the accent, and the accent is a light lavender in dark mode rather
// than a saturated colour, because that is what reads as Android in the dark. The
// accent itself is meui's, not a copy of Material's baseline purple.
func androidPalette(dark bool) palette {
	if dark {
		return palette{
			accent: 0xFFB9C3FF, accentHover: 0xFFC9D1FF, accentPressed: 0xFF9AA6F0, onAccent: 0xFF1B1B1F,
			window: 0xFF131316, panel: 0xFF1B1B1F, surface: 0xFF232327, raised: 0xFF2B2B31, sunken: 0xFF0E0E11,
			text: 0xFFE5E1E6, textSecondary: 0xFFC7C4CC, textDisabled: 0xFF6F6B75,
			border: 0xFF3A383F, borderSubtle: 0xFF2A282E, borderStrong: 0xFF4A474F,
			hover: 0x14FFFFFF, pressed: 0x24FFFFFF, selected: 0xFFB9C3FF,
			success: 0xFF7BD88F, warning: 0xFFF0C36D, error: 0xFFFFB4AB,
		}
	}
	return palette{
		accent: 0xFF4A5BC4, accentHover: 0xFF5A6BD4, accentPressed: 0xFF3A4AA8, onAccent: 0xFFFFFFFF,
		window: 0xFFF7F5FA, panel: 0xFFFDF8FF, surface: 0xFFFFFFFF, raised: 0xFFFFFFFF, sunken: 0xFFEDE8F2,
		text: 0xFF1B1B1F, textSecondary: 0xFF46464F, textDisabled: 0xFF9A98A0,
		border: 0xFFC9C5D0, borderSubtle: 0xFFDFDAE5, borderStrong: 0xFFA9A5B0,
		hover: 0x0F000000, pressed: 0x1F000000, selected: 0xFF4A5BC4,
		success: 0xFF2E7D32, warning: 0xFFB26A00, error: 0xFFBA1A1A,
	}
}

// iosPalette follows iOS's convention: a grouped background that is slightly grey
// against white cards, and a blue accent that is a meui blue rather than Apple's.
func iosPalette(dark bool) palette {
	if dark {
		return palette{
			accent: 0xFF7FA6FF, accentHover: 0xFF96B6FF, accentPressed: 0xFF6A8FE0, onAccent: 0xFF101014,
			window: 0xFF000000, panel: 0xFF1C1C1E, surface: 0xFF2C2C2E, raised: 0xFF3A3A3C, sunken: 0xFF141416,
			text: 0xFFFFFFFF, textSecondary: 0xFFAEAEB2, textDisabled: 0xFF5A5A5E,
			border: 0xFF38383A, borderSubtle: 0xFF2C2C2E, borderStrong: 0xFF48484A,
			hover: 0x14FFFFFF, pressed: 0x24FFFFFF, selected: 0xFF7FA6FF,
			success: 0xFF30D158, warning: 0xFFFFD60A, error: 0xFFFF453A,
		}
	}
	return palette{
		accent: 0xFF2F6FE0, accentHover: 0xFF4280EE, accentPressed: 0xFF2559B8, onAccent: 0xFFFFFFFF,
		window: 0xFFF2F2F7, panel: 0xFFF2F2F7, surface: 0xFFFFFFFF, raised: 0xFFFFFFFF, sunken: 0xFFE5E5EA,
		text: 0xFF000000, textSecondary: 0xFF3C3C43, textDisabled: 0xFF8E8E93,
		border: 0xFFD1D1D6, borderSubtle: 0xFFE5E5EA, borderStrong: 0xFFC7C7CC,
		hover: 0x0F000000, pressed: 0x1F000000, selected: 0xFF2F6FE0,
		success: 0xFF34C759, warning: 0xFFFF9500, error: 0xFFFF3B30,
	}
}

func init() {
	presetBuilders["android-dark"] = func() *Theme {
		p := androidPalette(true)
		return newTheme(Meta{"android-dark", PlatformAndroid, ModeDark}, p,
			metricsAndroid(), motionMacOS(), typoMacOS(), effectsMacOS(p.accent), chromeMacOS())
	}
	presetBuilders["android-light"] = func() *Theme {
		p := androidPalette(false)
		return newTheme(Meta{"android-light", PlatformAndroid, ModeLight}, p,
			metricsAndroid(), motionMacOS(), typoMacOS(), effectsMacOS(p.accent), chromeMacOS())
	}
	presetBuilders["ios-dark"] = func() *Theme {
		p := iosPalette(true)
		return newTheme(Meta{"ios-dark", PlatformIOS, ModeDark}, p,
			metricsIOS(), motionMacOS(), typoMacOS(), effectsMacOS(p.accent), chromeMacOS())
	}
	presetBuilders["ios-light"] = func() *Theme {
		p := iosPalette(false)
		return newTheme(Meta{"ios-light", PlatformIOS, ModeLight}, p,
			metricsIOS(), motionMacOS(), typoMacOS(), effectsMacOS(p.accent), chromeMacOS())
	}
}

// ── The harness look ────────────────────────────────────────────────
//
// A LIGHT preset, because the DeepSeek harness is a light, near-white, blue-accent
// interface and the tree's other presets are six desktop families that are all dark or
// mid-tone. Every colour here is MEASURED off the reference screenshot rather than
// chosen; the counts and the method are in LOOK-REFERENCE.md, which is what makes a
// disagreement with one of these values a specific conversation.
//
// It is named after what it is for rather than after a vendor, because it is not a
// re-skin of anybody's brand: the greys, the accent and the surfaces are the harness's, and
// everything else is ours.

func harnessPalette() palette {
	// 74.6% of the reference is #FFFFFF and 13.5% is #F9FAFB, so those two carry the whole
	// surface structure. The accent #4176E6 is the most saturated colour in the image.
	return palette{
		accent: 0xFF4176E6, accentHover: 0xFF5387E8, accentPressed: 0xFF2F62C9, onAccent: 0xFFFFFFFF,
		window: 0xFFFFFFFF, panel: 0xFFFFFFFF, surface: 0xFFFFFFFF, raised: 0xFFF9FAFB, sunken: 0xFFF5F6F7,
		text: 0xFF0F1115, textSecondary: 0xFF525256, textDisabled: 0xFFA9ACB2,
		border: 0xFFE5E5E5, borderSubtle: 0xFFECEEF0, borderStrong: 0xFFD5D8DD,
		hover: 0x0F0F1115, pressed: 0x1A0F1115, selected: 0xFF4176E6,
		success: 0xFF4176E6, // the reference has no green; a success is the accent here
		warning: 0xFFB26A00, error: 0xFFD03B3B,
	}
}

// harnessMetrics are the reference's densities at the scale measured, with the row bands
// that LOOK-REFERENCE.md records. The values are the ones a native toolkit wants anyway:
// a control you can hit with a finger is not a desktop control.
func harnessMetrics() Metrics {
	m := DefaultMetrics()
	m.Density = DensityRegular
	m.ControlHeight = 30
	m.ControlPadX = 12
	m.ControlPadY = 5
	m.InputPadX = 10
	m.MenuItemHeight = 30
	m.ListRowHeight = 30
	m.TabHeight = 34
	m.ToolbarHeight = 44
	m.StatusBarHeight = 22
	m.ScrollbarWidth = 8
	m.ScrollbarOverlay = true
	m.BorderWidth = 1
	m.FocusRingWidth = 2
	m.FocusRingOffset = 1
	m.BoxSize = 16
	m.BoxSpacing = 8
	m.SliderThumb = 16
	m.ArrowSize = 10
	m.GripSize = 1 // one theme divider line, per AGENTS.md
	m.ControlBorderWidth = 0
	m.TabIndicatorHeight = 2
	m.TabActiveFilled = false
	m.ArrowStyle = 1
	m.Shape = ShapeMeui()
	return m
}

func init() {
	presetBuilders["harness-light"] = func() *Theme {
		p := harnessPalette()
		return newTheme(Meta{"harness-light", PlatformMeui, ModeLight}, p,
			harnessMetrics(), motionMacOS(), typoMeui(), effectsMacOS(p.accent), chromeMacOS())
	}
	// The Markdown renderer follows the app: a light app gets the measured light document
	// palette, a dark one the dark palette. Without this a light preset still rendered
	// VS Code's dark code blocks, which is the defect the whole palette work existed to fix.
	presetBuilders["harness-dark"] = func() *Theme {
		p := macosPalette(true)
		return newTheme(Meta{"harness-dark", PlatformMeui, ModeDark}, p,
			metricsMacOS(), motionMacOS(), typoMacOS(), effectsMacOS(p.accent), chromeMacOS())
	}
}

// ── The web preset ─────────────────────────────────────────────────
//
// A web page is not a denser desktop window. It is a different set of DEFAULTS:
// an 8px spacing grid instead of a 4px one, a type scale that starts at 12 and
// reaches 32, 8px control radii instead of 4, a 40px control instead of a 30px
// one, hairline borders on white surfaces, a card elevation that is a hint rather
// than a drop shadow, and a focus ring that is visible from across the room.
//
// The palette is the SAME measured one as harness-light (LOOK-REFERENCE.md): white
// page, #F9FAFB surface, #0F1115 ink, #4176E6 accent, #E5E5E5 hairline. Reusing it
// is deliberate — the neutral web palette is measured, and a second guess at it
// would be a second answer to a question that already has one.
//
// What this preset does NOT do is art-direct. The reference applications in
// WEB-REFERENCE.md are cream/navy/pink, sky-blue, deep-blue, clay and forest green;
// those are per-scene overrides of this scaffold, not new presets, because the
// primitives (radius, shadow, gradient, type) are the same in all of them.

func webMetrics() Metrics {
	m := harnessMetrics()
	m.Density = DensityComfortable
	m.ControlHeight = 40
	m.ControlPadX = 16
	m.ControlPadY = 9
	m.InputPadX = 12
	m.MenuItemHeight = 36
	m.ListRowHeight = 44
	m.TabHeight = 40
	m.ToolbarHeight = 52
	m.StatusBarHeight = 24
	m.ScrollbarWidth = 10
	m.ScrollbarOverlay = true
	m.ScrollbarAutoHide = true
	m.BorderWidth = 1
	m.FocusRingWidth = 2
	m.FocusRingOffset = 2
	m.BoxSize = 18
	m.BoxSpacing = 10
	m.SliderThumb = 18
	m.ArrowSize = 10
	m.GripSize = 1
	m.ControlBorderWidth = 1
	m.TabIndicatorHeight = 2
	m.TabActiveFilled = false
	m.ArrowStyle = 1
	m.SwitchH = 22
	m.SliderTrack = 6
	m.SliderTrackRadius = -1
	m.ProgressHeight = 8
	m.ProgressRadius = -1
	m.NavBarHeight = 64
	m.NavRailWidth = 88
	m.Shape = shapeWeb()
	return m
}

func shapeWeb() *ShapeTheme {
	// Web radii are larger and more consistent than any desktop family's: a card
	// and a button share a radius, a dialog is a little softer, and a scrollbar is
	// a pill.
	return &ShapeTheme{
		Window: 12, Popup: 16, Menu: 10, Tooltip: 8,
		Button: 8, CheckBox: 4, Input: 8, ComboBox: 8, Tab: 6,
		GroupBox: 12, ListRow: 8, ScrollBar: 99,
	}
}

func typoWeb() Typography {
	// The scale a web page actually uses: 9 caption, 11 body, 14 title, 24 heading,
	// with a 1.55 body line height. Sizes are CAP HEIGHTS (see FontImpl.emPx), so 11
	// renders as the ~15px em a browser body uses. Families are a stack, first
	// installed wins, so a machine with Inter gets Inter and a machine without gets its
	// own sans rather than a serif fallback.
	return Typography{
		Families: []string{"Inter", "Inter Display", "Segoe UI", "Roboto", "Helvetica Neue", "Carlito", "Noto Sans", "DejaVu Sans"},
		Caption:  TextStyle{Size: 9, Weight: 400, LineHeight: 1.4},
		Body:     TextStyle{Size: 11, Weight: 400, LineHeight: 1.55},
		Label:    TextStyle{Size: 9, Weight: 500, LineHeight: 1.4},
		Button:   TextStyle{Size: 10, Weight: 500, LineHeight: 1.4},
		Title:    TextStyle{Size: 14, Weight: 600, LineHeight: 1.35},
		Heading:  TextStyle{Size: 24, Weight: 700, LineHeight: 1.15},
		Mono:     TextStyle{Size: 9, Weight: 400, LineHeight: 1.5},
	}
}

func effectsWeb(accent graphics.Color) Effects {
	fx := effectsMacOS(accent)
	fx.CardShadow = ShadowSpec{Color: 0x14000000, DY: 4, Blur: 18}
	fx.PopupShadow = ShadowSpec{Color: 0x2E000000, DY: 10, Blur: 30}
	fx.MenuShadow = ShadowSpec{Color: 0x26000000, DY: 8, Blur: 24}
	fx.Focus = FocusStyle{Kind: FocusRing, Width: 2, Offset: 2, Color: accent, Alt: 0xFFFFFFFF}
	fx.Material = MaterialNone
	return fx
}

func chromeWeb() Chrome {
	ch := chromeMacOS()
	ch.Height = 36
	ch.ButtonW = 28
	ch.ButtonH = 28
	ch.TitleWeight = 600
	return ch
}

// webDarkPalette is the dark half of the same web scaffold: near-black page,
// a slightly raised surface, the accent lifted for contrast on dark.
func webDarkPalette() palette {
	return palette{
		accent: 0xFF5B8DEF, accentHover: 0xFF74A0F5, accentPressed: 0xFF4877D6, onAccent: 0xFFFFFFFF,
		window: 0xFF0B0D10, panel: 0xFF111418, surface: 0xFF111418, raised: 0xFF161A1F, sunken: 0xFF0B0D10,
		text: 0xFFF3F4F6, textSecondary: 0xFF9CA3AF, textDisabled: 0xFF5B616B,
		border: 0xFF262B33, borderSubtle: 0xFF1E232A, borderStrong: 0xFF3A4250,
		hover: 0x14FFFFFF, pressed: 0x24FFFFFF, selected: 0x335B8DEF,
		success: 0xFF34D399, warning: 0xFFFBBF24, error: 0xFFF87171,
	}
}

func init() {
	presetBuilders["web-light"] = func() *Theme {
		p := harnessPalette()
		return newTheme(Meta{"web-light", PlatformWeb, ModeLight}, p,
			webMetrics(), motionWeb(), typoWeb(), effectsWeb(p.accent), chromeWeb())
	}
	presetBuilders["web-dark"] = func() *Theme {
		p := webDarkPalette()
		return newTheme(Meta{"web-dark", PlatformWeb, ModeDark}, p,
			webMetrics(), motionWeb(), typoWeb(), effectsWeb(p.accent), chromeWeb())
	}
	registerPresetTweak("web-light", func(_ *Theme) { webStyleControls(harnessPalette()) })
	registerPresetTweak("web-dark", func(_ *Theme) { webStyleControls(webDarkPalette()) })
}

// webStyleControls is the web variant of the CONTROLS, not of the surfaces. A web page's
// primary button is accent-filled and borderless; its inputs are white with a grey edge
// and an accent focus edge. Everything else follows the tokens.
func webStyleControls(p palette) {
	// The widget themes are package GLOBALS, regenerated by Apply; a preset tweak has
	// to write them after that pass.
	if DefaultButtonTheme != nil {
		b := DefaultButtonTheme.Copy()
		b.BackgroundInk = p.accent
		b.BackgroundHover = p.accentHover
		b.BackgroundPressed = p.accentPressed
		b.BackgroundActive = p.accentPressed
		b.EdgeInk = p.accent
		b.EdgeWidth = 0
		b.TextInk = p.onAccent
		b.TextActive = p.onAccent
		b.CornerRadius = 8
		*DefaultButtonTheme = *b
	}
	if DefaultComboBoxTheme != nil {
		c := DefaultComboBoxTheme.Copy()
		c.BackgroundInk = p.panel
		c.EdgeInk = p.border
		c.TextInk = p.text
		c.ArrowInk = p.textSecondary
		c.CornerRadius = 8
		*DefaultComboBoxTheme = *c
	}
	if DefaultTextFieldTheme != nil {
		f := DefaultTextFieldTheme.Copy()
		f.BackgroundInk = p.panel
		f.BackgroundFocused = p.panel
		f.EdgeInk = p.border
		f.EdgeHovered = p.borderStrong
		f.EdgeFocused = p.accent
		f.TextInk = p.text
		f.PlaceholderInk = p.textDisabled
		f.CornerRadius = 8
		*DefaultTextFieldTheme = *f
	}
}

// motionWeb is a little slower than a desktop toolkit's: a web surface announces
// its state instead of flashing it, and 120/90ms reads as deliberate where 80/60
// reads as glitchy.
func motionWeb() Motion {
	m := motionMacOS()
	m.Hover = Timing{DurationMS: 120, Easing: EaseStandard}
	m.Press = Timing{DurationMS: 90, Easing: EaseStandard}
	m.Focus = Timing{DurationMS: 120, Easing: EaseStandard}
	m.Toggle = Timing{DurationMS: 180, Easing: EaseStandard}
	return m
}
