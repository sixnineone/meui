package theme

import (
	"github.com/sixnineone/meui/graphics"
)

var DefaultButtonTheme = &ButtonTheme{
	BackgroundInk:     graphics.Color(0xFF4F566D),
	BackgroundHover:   graphics.Color(0xFF565D75),
	BackgroundPressed: graphics.Color(0xFF454B60),
	BackgroundActive:  graphics.Color(0xFF5064A1),
	EdgeInk:           graphics.Color(0xFF6272A4),
	TextInk:           graphics.Color(0xFFC3C6C6),
	TextActive:        graphics.Color(0xFFFFFFFF),
	TextDisabled:      graphics.Color(0xFF6F737A),
	CornerRadius:      4,
	HMargin:           8,
	VMargin:           3,
	Font:              nil, // Falls back to theme.DefaultTheme.BaseFont
}

type ButtonTheme struct {
	BackgroundInk     graphics.Color
	BackgroundHover   graphics.Color
	BackgroundPressed graphics.Color
	BackgroundActive  graphics.Color
	EdgeInk           graphics.Color
	TextInk           graphics.Color
	TextActive        graphics.Color
	TextDisabled      graphics.Color
	CornerRadius      float32
	EdgeWidth         float32 // 0 = borderless (Metro)
	HMargin           float32
	VMargin           float32
	Font              graphics.Font
}

func (t *ButtonTheme) Copy() *ButtonTheme {
	return &ButtonTheme{
		BackgroundInk:     t.BackgroundInk,
		BackgroundHover:   t.BackgroundHover,
		BackgroundPressed: t.BackgroundPressed,
		BackgroundActive:  t.BackgroundActive,
		EdgeInk:           t.EdgeInk,
		TextInk:           t.TextInk,
		TextActive:        t.TextActive,
		TextDisabled:      t.TextDisabled,
		CornerRadius:      t.CornerRadius,
		EdgeWidth:         t.EdgeWidth,
		HMargin:           t.HMargin,
		VMargin:           t.VMargin,
		Font:              t.Font,
	}
}

var DefaultCheckBoxTheme = &CheckBoxTheme{
	BackgroundInk:   graphics.Color(0xFF3A3D4C),
	BackgroundHover: graphics.Color(0xFF454B60),
	EdgeInk:         graphics.Color(0xFF6272A4),
	EdgeFocused:     graphics.Color(0xFF8A9CC8),
	SelectionInk:    graphics.Color(0xFF5064A1),
	TextInk:         graphics.Color(0xFFC3C6C6),
	TextDisabled:    graphics.Color(0xFF6F737A),
	BoxSize:         16,
	BoxSpacing:      6,
	CornerRadius:    3,
	Font:            nil, // Falls back to theme.DefaultTheme.BaseFont
}

type CheckBoxTheme struct {
	BackgroundInk   graphics.Color
	BackgroundHover graphics.Color
	EdgeInk         graphics.Color
	EdgeFocused     graphics.Color
	SelectionInk    graphics.Color
	TextInk         graphics.Color
	TextDisabled    graphics.Color
	BoxSize         float32
	BoxSpacing      float32
	CornerRadius    float32
	Font            graphics.Font
}

func (t *CheckBoxTheme) Copy() *CheckBoxTheme {
	return &CheckBoxTheme{
		BackgroundInk:   t.BackgroundInk,
		BackgroundHover: t.BackgroundHover,
		EdgeInk:         t.EdgeInk,
		EdgeFocused:     t.EdgeFocused,
		SelectionInk:    t.SelectionInk,
		TextInk:         t.TextInk,
		TextDisabled:    t.TextDisabled,
		BoxSize:         t.BoxSize,
		BoxSpacing:      t.BoxSpacing,
		CornerRadius:    t.CornerRadius,
		Font:            t.Font,
	}
}

var DefaultRadioButtonTheme = &RadioButtonTheme{
	BackgroundInk:   graphics.Color(0xFF3A3D4C),
	BackgroundHover: graphics.Color(0xFF454B60),
	EdgeInk:         graphics.Color(0xFF6272A4),
	EdgeFocused:     graphics.Color(0xFF8A9CC8),
	SelectionInk:    graphics.Color(0xFF5064A1),
	TextInk:         graphics.Color(0xFFC3C6C6),
	TextDisabled:    graphics.Color(0xFF6F737A),
	BoxSize:         16,
	BoxSpacing:      6,
	Font:            nil, // Falls back to theme.DefaultTheme.BaseFont
}

type RadioButtonTheme struct {
	BackgroundInk   graphics.Color
	BackgroundHover graphics.Color
	EdgeInk         graphics.Color
	EdgeFocused     graphics.Color
	SelectionInk    graphics.Color
	TextInk         graphics.Color
	TextDisabled    graphics.Color
	BoxSize         float32
	BoxSpacing      float32
	Font            graphics.Font
}

func (t *RadioButtonTheme) Copy() *RadioButtonTheme {
	return &RadioButtonTheme{
		BackgroundInk:   t.BackgroundInk,
		BackgroundHover: t.BackgroundHover,
		EdgeInk:         t.EdgeInk,
		EdgeFocused:     t.EdgeFocused,
		SelectionInk:    t.SelectionInk,
		TextInk:         t.TextInk,
		TextDisabled:    t.TextDisabled,
		BoxSize:         t.BoxSize,
		BoxSpacing:      t.BoxSpacing,
		Font:            t.Font,
	}
}

var DefaultLabelTheme = &LabelTheme{
	TextInk:       graphics.Color(0xFFC3C6C6),
	TextSecondary: graphics.Color(0xFF8A919E),
	Font:          nil, // Will fall back to theme.DefaultTheme.BaseFont
}

type LabelTheme struct {
	TextInk       graphics.Color
	TextSecondary graphics.Color
	Font          graphics.Font
}

func (t *LabelTheme) Copy() *LabelTheme {
	return &LabelTheme{
		TextInk:       t.TextInk,
		TextSecondary: t.TextSecondary,
		Font:          t.Font,
	}
}

var DefaultTextFieldTheme = &TextFieldTheme{
	BackgroundInk:     graphics.Color(0xFF3A3D4C),
	BackgroundFocused: graphics.Color(0xFF414863),
	EdgeInk:           graphics.Color(0xFF6272A4),
	EdgeFocused:       graphics.Color(0xFF8A9CC8),
	EdgeHovered:       graphics.Color(0xFF5A6480),
	TextInk:           graphics.Color(0xFFFFFFFF),
	TextDisabled:      graphics.Color(0xFF6F737A),
	PlaceholderInk:    graphics.Color(0xFF6F737A),
	SelectionInk:      graphics.Color(0x445064A1),
	SelectedTextInk:   graphics.Color(0xFFFFFFFF),
	CornerRadius:      4,
	HMargin:           6,
	VMargin:           3,
	Font:              nil, // Falls back to theme.DefaultTheme.BaseFont
}

type TextFieldTheme struct {
	BackgroundInk     graphics.Color
	BackgroundFocused graphics.Color
	EdgeInk           graphics.Color
	EdgeFocused       graphics.Color
	EdgeHovered       graphics.Color
	TextInk           graphics.Color
	TextDisabled      graphics.Color
	PlaceholderInk    graphics.Color
	SelectionInk      graphics.Color
	SelectedTextInk   graphics.Color
	CornerRadius      float32
	BottomEdgeInk     graphics.Color // emphasised bottom border (Fluent)
	BottomEdgeWidth   float32
	BottomFocusWidth  float32
	HMargin           float32
	VMargin           float32
	Font              graphics.Font
}

func (t *TextFieldTheme) Copy() *TextFieldTheme {
	return &TextFieldTheme{
		BackgroundInk:     t.BackgroundInk,
		BackgroundFocused: t.BackgroundFocused,
		EdgeInk:           t.EdgeInk,
		EdgeFocused:       t.EdgeFocused,
		EdgeHovered:       t.EdgeHovered,
		TextInk:           t.TextInk,
		TextDisabled:      t.TextDisabled,
		PlaceholderInk:    t.PlaceholderInk,
		SelectionInk:      t.SelectionInk,
		SelectedTextInk:   t.SelectedTextInk,
		CornerRadius:      t.CornerRadius,
		BottomEdgeInk:     t.BottomEdgeInk,
		BottomEdgeWidth:   t.BottomEdgeWidth,
		BottomFocusWidth:  t.BottomFocusWidth,
		HMargin:           t.HMargin,
		VMargin:           t.VMargin,
		Font:              t.Font,
	}
}

var DefaultSliderTheme = &SliderTheme{
	TrackInk:    graphics.Color(0xFF4B5163),
	TrackFilled: graphics.Color(0xFF5064A1),
	ThumbInk:    graphics.Color(0xFFC3C6C6),
	ThumbHover:  graphics.Color(0xFFFFFFFF),
	EdgeInk:     graphics.Color(0xFF6272A4),
	TickColor:   graphics.Color(0xFF666666),
	ThumbSize:   16,
	TickLength:  4,
}

type SliderTheme struct {
	TrackInk    graphics.Color
	TrackFilled graphics.Color
	ThumbInk    graphics.Color
	ThumbHover  graphics.Color
	EdgeInk     graphics.Color
	TickColor   graphics.Color
	ThumbBorder graphics.Color // ring around the thumb (0 = none)
	ThumbInner  graphics.Color // centre dot (Fluent; 0 = none)
	Font        graphics.Font
	ThumbSize   float32
	TickLength  float32
	TrackHeight float32
	TrackRadius float32 // -1 = pill
	TrackBorder bool    // outline the groove (Breeze)
}

func (t *SliderTheme) Copy() *SliderTheme {
	return &SliderTheme{
		TrackInk:    t.TrackInk,
		TrackFilled: t.TrackFilled,
		ThumbInk:    t.ThumbInk,
		ThumbHover:  t.ThumbHover,
		EdgeInk:     t.EdgeInk,
		TickColor:   t.TickColor,
		ThumbBorder: t.ThumbBorder,
		ThumbInner:  t.ThumbInner,
		TrackHeight: t.TrackHeight,
		TrackRadius: t.TrackRadius,
		TrackBorder: t.TrackBorder,
		Font:        t.Font,
		ThumbSize:   t.ThumbSize,
		TickLength:  t.TickLength,
	}
}

var DefaultProgressBarTheme = &ProgressBarTheme{
	TrackInk:     graphics.Color(0xFF3A3D4C),
	FillInk:      graphics.Color(0xFF5064A1),
	EdgeInk:      graphics.Color(0xFF6272A4),
	CornerRadius: 4,
	Height:       6,
}

type ProgressBarTheme struct {
	TrackInk     graphics.Color
	FillInk      graphics.Color
	EdgeInk      graphics.Color
	CornerRadius float32
	Height       float32
}

func (t *ProgressBarTheme) Copy() *ProgressBarTheme {
	return &ProgressBarTheme{
		TrackInk:     t.TrackInk,
		FillInk:      t.FillInk,
		EdgeInk:      t.EdgeInk,
		CornerRadius: t.CornerRadius,
		Height:       t.Height,
	}
}

var DefaultTabTheme = &TabTheme{
	BackgroundInk:    graphics.Color(0xFF414450),
	BackgroundActive: graphics.Color(0xFF414450),
	BackgroundHover:  graphics.Color(0xFF4A4F60),
	EdgeInk:          graphics.Color(0xFF6272A4),
	TextInk:          graphics.Color(0xFFC3C6C6),
	TextActive:       graphics.Color(0xFFFFFFFF),
	TextHover:        graphics.Color(0xFFFFFFFF),
	IndicatorInk:     graphics.Color(0xFF8A9CC8),
	IndicatorHeight:  2.0,
	CornerRadius:     0,
	HMargin:          10,
	VMargin:          6,
	TabHeight:        24,
	CloseSize:        12,
	Font:             nil,
}

type TabTheme struct {
	BackgroundInk    graphics.Color
	BackgroundActive graphics.Color
	BackgroundHover  graphics.Color
	EdgeInk          graphics.Color
	TextInk          graphics.Color
	TextActive       graphics.Color
	TextHover        graphics.Color
	IndicatorInk     graphics.Color
	IndicatorHeight  float32
	CornerRadius     float32
	HMargin          float32
	VMargin          float32
	TabHeight        float32
	CloseSize        float32
	Font             graphics.Font
}

func (t *TabTheme) Copy() *TabTheme {
	return &TabTheme{
		BackgroundInk:    t.BackgroundInk,
		BackgroundActive: t.BackgroundActive,
		BackgroundHover:  t.BackgroundHover,
		EdgeInk:          t.EdgeInk,
		TextInk:          t.TextInk,
		TextActive:       t.TextActive,
		TextHover:        t.TextHover,
		IndicatorInk:     t.IndicatorInk,
		IndicatorHeight:  t.IndicatorHeight,
		CornerRadius:     t.CornerRadius,
		HMargin:          t.HMargin,
		VMargin:          t.VMargin,
		TabHeight:        t.TabHeight,
		CloseSize:        t.CloseSize,
		Font:             t.Font,
	}
}

// ComboBoxTheme defines the visual style for ComboBox widgets.
type ComboBoxTheme struct {
	BackgroundInk    graphics.Color
	EdgeInk          graphics.Color
	TextInk          graphics.Color
	TextDisabled     graphics.Color
	ArrowInk         graphics.Color
	HoverInk         graphics.Color
	SelectedInk      graphics.Color
	OptionHeight     float32
	ArrowSize        float32
	ArrowStyle       uint8 // 0 = triangle, 1 = chevron
	BottomEdgeInk    graphics.Color
	BottomEdgeWidth  float32
	BottomFocusWidth float32
	CornerRadius     float32
	Font             graphics.Font
}

var DefaultComboBoxTheme = &ComboBoxTheme{
	BackgroundInk: graphics.Color(0xFF3A3D4C),
	EdgeInk:       graphics.Color(0xFF6272A4),
	TextInk:       graphics.Color(0xFFC3C6C6),
	TextDisabled:  graphics.Color(0xFF6F737A),
	ArrowInk:      graphics.Color(0xFF9AA0B5),
	HoverInk:      graphics.Color(0x334F566D),
	SelectedInk:   graphics.Color(0x445064A1),
	OptionHeight:  24,
	ArrowSize:     10,
	Font:          nil, // Falls back to theme.DefaultTheme.BaseFont
}

func (t *ComboBoxTheme) Copy() *ComboBoxTheme {
	return &ComboBoxTheme{
		BackgroundInk:    t.BackgroundInk,
		EdgeInk:          t.EdgeInk,
		TextInk:          t.TextInk,
		TextDisabled:     t.TextDisabled,
		ArrowInk:         t.ArrowInk,
		HoverInk:         t.HoverInk,
		SelectedInk:      t.SelectedInk,
		OptionHeight:     t.OptionHeight,
		ArrowSize:        t.ArrowSize,
		ArrowStyle:       t.ArrowStyle,
		BottomEdgeInk:    t.BottomEdgeInk,
		BottomEdgeWidth:  t.BottomEdgeWidth,
		BottomFocusWidth: t.BottomFocusWidth,
		CornerRadius:     t.CornerRadius,
		Font:             t.Font,
	}
}

// ListViewTheme defines the visual style for ListView widgets.
type ListViewTheme struct {
	BackgroundInk   graphics.Color
	EdgeInk         graphics.Color
	TextInk         graphics.Color
	SelectedInk     graphics.Color
	HoverInk        graphics.Color
	AlternateRowInk graphics.Color
	CornerRadius    float32
	Font            graphics.Font
}

var DefaultListViewTheme = &ListViewTheme{
	BackgroundInk:   graphics.Color(0xFF3A3D4C),
	EdgeInk:         graphics.Color(0xFF6272A4),
	TextInk:         graphics.Color(0xFFFFFFFF),
	SelectedInk:     graphics.Color(0x445064A1),
	HoverInk:        graphics.Color(0x334F566D),
	AlternateRowInk: graphics.Color(0x11FFFFFF),
	Font:            nil, // Will fall back to theme.DefaultTheme.BaseFont
}

func (t *ListViewTheme) Copy() *ListViewTheme {
	return &ListViewTheme{
		BackgroundInk:   t.BackgroundInk,
		EdgeInk:         t.EdgeInk,
		TextInk:         t.TextInk,
		SelectedInk:     t.SelectedInk,
		HoverInk:        t.HoverInk,
		AlternateRowInk: t.AlternateRowInk,
		CornerRadius:    t.CornerRadius,
		Font:            t.Font,
	}
}

// ScrollBarTheme defines the visual style for ScrollBar widgets.
type ScrollBarTheme struct {
	TrackInk   graphics.Color
	ThumbInk   graphics.Color
	ThumbHover graphics.Color
	EdgeInk    graphics.Color
	Overlay    bool    // thin floating pill, no track (macOS/GNOME)
	AutoHide   bool    // fade out while idle (opt-in; default: persistent)
	ThumbInset float32 // inset from the bar edges (overlay pill margin)
}

var DefaultScrollBarTheme = &ScrollBarTheme{
	TrackInk:   graphics.Color(0xFF3A3D4C),
	ThumbInk:   graphics.Color(0xFF5A6480),
	ThumbHover: graphics.Color(0xFF6A7590),
	EdgeInk:    graphics.Color(0xFF6272A4),
	// Fallback before any theme is applied: same meui default as
	// DefaultMetrics — overlay pill, no track band.
	Overlay:    true,
	ThumbInset: 2,
}

func (t *ScrollBarTheme) Copy() *ScrollBarTheme {
	return &ScrollBarTheme{
		TrackInk:   t.TrackInk,
		ThumbInk:   t.ThumbInk,
		ThumbHover: t.ThumbHover,
		EdgeInk:    t.EdgeInk,
		Overlay:    t.Overlay,
		AutoHide:   t.AutoHide,
		ThumbInset: t.ThumbInset,
	}
}

// SpinBoxTheme defines the visual style for SpinBox widgets.
type SpinBoxTheme struct {
	BackgroundInk    graphics.Color
	EdgeInk          graphics.Color
	TextInk          graphics.Color
	ButtonInk        graphics.Color
	ButtonPressedInk graphics.Color
	ArrowInk         graphics.Color
	CornerRadius     float32
	Font             graphics.Font
}

var DefaultSpinBoxTheme = &SpinBoxTheme{
	BackgroundInk:    graphics.Color(0xFF3A3D4C),
	EdgeInk:          graphics.Color(0xFF6272A4),
	TextInk:          graphics.Color(0xFFFFFFFF),
	ButtonInk:        graphics.Color(0xFF4F566D),
	ButtonPressedInk: graphics.Color(0xFF2A2A2A),
	ArrowInk:         graphics.Color(0xFF6272A4),
	Font:             nil, // Will fall back to theme.DefaultTheme.BaseFont
}

func (t *SpinBoxTheme) Copy() *SpinBoxTheme {
	return &SpinBoxTheme{
		BackgroundInk:    t.BackgroundInk,
		EdgeInk:          t.EdgeInk,
		TextInk:          t.TextInk,
		ButtonInk:        t.ButtonInk,
		ButtonPressedInk: t.ButtonPressedInk,
		ArrowInk:         t.ArrowInk,
		CornerRadius:     t.CornerRadius,
		Font:             t.Font,
	}
}

// TextAreaTheme defines the visual style for TextArea widgets.
type TextAreaTheme struct {
	BackgroundInk     graphics.Color
	BackgroundFocused graphics.Color
	EdgeInk           graphics.Color
	EdgeFocused       graphics.Color
	TextInk           graphics.Color
	PlaceholderInk    graphics.Color
	SelectionColor    graphics.Color
	GutterBgInk       graphics.Color
	GutterTextInk     graphics.Color
	GutterActiveInk   graphics.Color
	CornerRadius      float32
	Font              graphics.Font
}

var DefaultTextAreaTheme = &TextAreaTheme{
	BackgroundInk:     graphics.Color(0xFF3A3D4C),
	BackgroundFocused: graphics.Color(0xFF414863),
	EdgeInk:           graphics.Color(0xFF6272A4),
	EdgeFocused:       graphics.Color(0xFF8A9CC8),
	TextInk:           graphics.Color(0xFFFFFFFF),
	PlaceholderInk:    graphics.Color(0xFF6F737A),
	SelectionColor:    graphics.Color(0x445064A1),
	GutterBgInk:       graphics.Color(0xFF363947),
	GutterTextInk:     graphics.Color(0xFF6F737A),
	GutterActiveInk:   graphics.Color(0xFFC3C6C6),
	Font:              nil,
}

func (t *TextAreaTheme) Copy() *TextAreaTheme {
	return &TextAreaTheme{
		BackgroundInk:     t.BackgroundInk,
		BackgroundFocused: t.BackgroundFocused,
		EdgeInk:           t.EdgeInk,
		EdgeFocused:       t.EdgeFocused,
		TextInk:           t.TextInk,
		PlaceholderInk:    t.PlaceholderInk,
		SelectionColor:    t.SelectionColor,
		GutterBgInk:       t.GutterBgInk,
		GutterTextInk:     t.GutterTextInk,
		GutterActiveInk:   t.GutterActiveInk,
		CornerRadius:      t.CornerRadius,
		Font:              t.Font,
	}
}

var DefaultSwitchTheme = &SwitchTheme{
	TrackOn:  graphics.Color(0xFF5064A1),
	TrackOff: graphics.Color(0xFF4A4A4A),
	ThumbInk: graphics.Color(0xFFE0E0E0),
}

type SwitchTheme struct {
	TrackOn  graphics.Color
	TrackOff graphics.Color
	ThumbInk graphics.Color // knob when ON (on-accent)
	ThumbOff graphics.Color // knob when OFF
}

func (t *SwitchTheme) Copy() *SwitchTheme {
	return &SwitchTheme{
		TrackOn:  t.TrackOn,
		TrackOff: t.TrackOff,
		ThumbInk: t.ThumbInk,
		ThumbOff: t.ThumbOff,
	}
}

var DefaultTooltipTheme = &TooltipTheme{
	BackgroundInk: graphics.Color(0xFF414450),
	EdgeInk:       graphics.Color(0xFF6272A4),
	TextInk:       graphics.Color(0xFFC3C6C6),
	Font:          nil,
}

type TooltipTheme struct {
	BackgroundInk graphics.Color
	EdgeInk       graphics.Color
	TextInk       graphics.Color
	Font          graphics.Font
}

func (t *TooltipTheme) Copy() *TooltipTheme {
	return &TooltipTheme{
		BackgroundInk: t.BackgroundInk,
		EdgeInk:       t.EdgeInk,
		TextInk:       t.TextInk,
		Font:          t.Font,
	}
}

var DefaultStatusBarTheme = &StatusBarTheme{
	BackgroundInk: graphics.Color(0xFF414450),
	EdgeInk:       graphics.Color(0xFF6272A4),
	TextInk:       graphics.Color(0xFFC3C6C6),
	GripInk:       graphics.Color(0xFF5A6480),
	Font:          nil,
}

type StatusBarTheme struct {
	BackgroundInk graphics.Color
	EdgeInk       graphics.Color
	TextInk       graphics.Color
	GripInk       graphics.Color
	Font          graphics.Font
}

func (t *StatusBarTheme) Copy() *StatusBarTheme {
	return &StatusBarTheme{
		BackgroundInk: t.BackgroundInk,
		EdgeInk:       t.EdgeInk,
		TextInk:       t.TextInk,
		GripInk:       t.GripInk,
		Font:          t.Font,
	}
}

var DefaultToolBarTheme = &ToolBarTheme{
	BackgroundInk: graphics.Color(0xFF414450),
	EdgeInk:       graphics.Color(0xFF6272A4),
	ButtonInk:     graphics.Color(0xFF4A4F60),
	ButtonHover:   graphics.Color(0xFF565D75),
	ButtonToggled: graphics.Color(0x445064A1),
	IconInk:       graphics.Color(0xFFC3C6C6),
	TextInk:       graphics.Color(0xFFC3C6C6),
	SeparatorInk:  graphics.Color(0xFF4B5163),
	Font:          nil,
}

type ToolBarTheme struct {
	BackgroundInk graphics.Color
	EdgeInk       graphics.Color
	ButtonInk     graphics.Color
	ButtonHover   graphics.Color
	ButtonToggled graphics.Color
	IconInk       graphics.Color
	TextInk       graphics.Color
	SeparatorInk  graphics.Color
	Font          graphics.Font
}

func (t *ToolBarTheme) Copy() *ToolBarTheme {
	return &ToolBarTheme{
		BackgroundInk: t.BackgroundInk,
		EdgeInk:       t.EdgeInk,
		ButtonInk:     t.ButtonInk,
		ButtonHover:   t.ButtonHover,
		ButtonToggled: t.ButtonToggled,
		IconInk:       t.IconInk,
		TextInk:       t.TextInk,
		SeparatorInk:  t.SeparatorInk,
		Font:          t.Font,
	}
}

// ── MenuBarTheme ─────────────────────────────────────────────────

var DefaultMenuBarTheme = &MenuBarTheme{
	BackgroundInk:    graphics.Color(0xFF414450),
	BackgroundHover:  graphics.Color(0xFF565D75),
	BackgroundActive: graphics.Color(0xFF5064A1),
	TextInk:          graphics.Color(0xFFC3C6C6),
	TextActive:       graphics.Color(0xFFFFFFFF),
	TextSecondary:    graphics.Color(0xFF8A919E),
	EdgeInk:          graphics.Color(0xFF6272A4),
	Font:             nil,
}

type MenuBarTheme struct {
	BackgroundInk    graphics.Color
	BackgroundHover  graphics.Color
	BackgroundActive graphics.Color
	TextInk          graphics.Color
	TextActive       graphics.Color
	TextSecondary    graphics.Color
	EdgeInk          graphics.Color
	Font             graphics.Font
}

func (t *MenuBarTheme) Copy() *MenuBarTheme {
	return &MenuBarTheme{
		BackgroundInk:    t.BackgroundInk,
		BackgroundHover:  t.BackgroundHover,
		BackgroundActive: t.BackgroundActive,
		TextInk:          t.TextInk,
		TextActive:       t.TextActive,
		TextSecondary:    t.TextSecondary,
		EdgeInk:          t.EdgeInk,
		Font:             t.Font,
	}
}

// CalendarTheme defines the visual style for Calendar widgets.
var DefaultCalendarTheme = &CalendarTheme{
	BackgroundInk:    graphics.Color(0xFF3A3D4C),
	BackgroundActive: graphics.Color(0xFF414450),
	BackgroundHover:  graphics.Color(0xFF4A4F60),
	EdgeInk:          graphics.Color(0xFF6272A4),
	TextInk:          graphics.Color(0xFFC3C6C6),
	TextActive:       graphics.Color(0xFFFFFFFF),
	TextHover:        graphics.Color(0xFFFFFFFF),
	CornerRadius:     0,
	HMargin:          10,
	VMargin:          6,
	TabHeight:        24,
	CloseSize:        12,
	Font:             nil,
}

type CalendarTheme struct {
	BackgroundInk    graphics.Color
	BackgroundActive graphics.Color
	BackgroundHover  graphics.Color
	EdgeInk          graphics.Color
	TextInk          graphics.Color
	TextActive       graphics.Color
	TextHover        graphics.Color
	CornerRadius     float32
	HMargin          float32
	VMargin          float32
	TabHeight        float32
	CloseSize        float32
	Font             graphics.Font
}

func (t *CalendarTheme) Copy() *CalendarTheme {
	return &CalendarTheme{
		BackgroundInk:    t.BackgroundInk,
		BackgroundActive: t.BackgroundActive,
		BackgroundHover:  t.BackgroundHover,
		EdgeInk:          t.EdgeInk,
		TextInk:          t.TextInk,
		TextActive:       t.TextActive,
		TextHover:        t.TextHover,
		CornerRadius:     t.CornerRadius,
		HMargin:          t.HMargin,
		VMargin:          t.VMargin,
		TabHeight:        t.TabHeight,
		CloseSize:        t.CloseSize,
		Font:             t.Font,
	}
}

// ColorPickerTheme defines the visual style for ColorPicker widgets.
var DefaultColorPickerTheme = &ColorPickerTheme{
	BackgroundInk:    graphics.Color(0xFF3A3D4C),
	BackgroundActive: graphics.Color(0xFF414450),
	BackgroundHover:  graphics.Color(0xFF4A4F60),
	EdgeInk:          graphics.Color(0xFF6272A4),
	TextInk:          graphics.Color(0xFFC3C6C6),
	TextActive:       graphics.Color(0xFFFFFFFF),
	TextHover:        graphics.Color(0xFFFFFFFF),
	CornerRadius:     0,
	HMargin:          10,
	VMargin:          6,
	TabHeight:        24,
	CloseSize:        12,
	Font:             nil,
}

type ColorPickerTheme struct {
	BackgroundInk    graphics.Color
	BackgroundActive graphics.Color
	BackgroundHover  graphics.Color
	EdgeInk          graphics.Color
	TextInk          graphics.Color
	TextActive       graphics.Color
	TextHover        graphics.Color
	CornerRadius     float32
	HMargin          float32
	VMargin          float32
	TabHeight        float32
	CloseSize        float32
	Font             graphics.Font
}

func (t *ColorPickerTheme) Copy() *ColorPickerTheme {
	return &ColorPickerTheme{
		BackgroundInk:    t.BackgroundInk,
		BackgroundActive: t.BackgroundActive,
		BackgroundHover:  t.BackgroundHover,
		EdgeInk:          t.EdgeInk,
		TextInk:          t.TextInk,
		TextActive:       t.TextActive,
		TextHover:        t.TextHover,
		CornerRadius:     t.CornerRadius,
		HMargin:          t.HMargin,
		VMargin:          t.VMargin,
		TabHeight:        t.TabHeight,
		CloseSize:        t.CloseSize,
		Font:             t.Font,
	}
}

// ContextMenuTheme defines the visual style for ContextMenu widgets.
var DefaultContextMenuTheme = &ContextMenuTheme{
	BackgroundInk: graphics.Color(0xFF414450),
	EdgeInk:       graphics.Color(0xFF6272A4),
	TextInk:       graphics.Color(0xFFC3C6C6),
	TextDisabled:  graphics.Color(0xFF6F737A),
	ArrowInk:      graphics.Color(0xFF9AA0B5),
	HoverInk:      graphics.Color(0x334F566D),
	SelectedInk:   graphics.Color(0x445064A1),
	SeparatorInk:  graphics.Color(0xFF4B5163),
	OptionHeight:  24,
	ArrowSize:     10,
	Font:          nil,
}

type ContextMenuTheme struct {
	BackgroundInk graphics.Color
	EdgeInk       graphics.Color
	TextInk       graphics.Color
	TextDisabled  graphics.Color
	ArrowInk      graphics.Color
	HoverInk      graphics.Color
	SelectedInk   graphics.Color
	// SeparatorInk: divider lines between menu sections.
	SeparatorInk graphics.Color
	OptionHeight float32
	ArrowSize    float32
	Font         graphics.Font
}

func (t *ContextMenuTheme) Copy() *ContextMenuTheme {
	return &ContextMenuTheme{
		BackgroundInk: t.BackgroundInk,
		EdgeInk:       t.EdgeInk,
		TextInk:       t.TextInk,
		TextDisabled:  t.TextDisabled,
		ArrowInk:      t.ArrowInk,
		HoverInk:      t.HoverInk,
		SelectedInk:   t.SelectedInk,
		SeparatorInk:  t.SeparatorInk,
		OptionHeight:  t.OptionHeight,
		ArrowSize:     t.ArrowSize,
		Font:          t.Font,
	}
}

// GroupBoxTheme defines the visual style for GroupBox widgets.
var DefaultGroupBoxTheme = &GroupBoxTheme{
	TextInk:       graphics.Color(0xFFC3C6C6),
	TextSecondary: graphics.Color(0xFF8A919E),
	Font:          nil,
}

type GroupBoxTheme struct {
	TextInk       graphics.Color
	TextSecondary graphics.Color
	Font          graphics.Font
}

func (t *GroupBoxTheme) Copy() *GroupBoxTheme {
	return &GroupBoxTheme{
		TextInk:       t.TextInk,
		TextSecondary: t.TextSecondary,
		Font:          t.Font,
	}
}

// ScrollPaneTheme defines the visual style for ScrollPane widgets.
var DefaultScrollPaneTheme = &ScrollPaneTheme{
	BackgroundInk:   0x00000000,
	EdgeInk:         0x00000000,
	TextInk:         graphics.Color(0xFFFFFFFF),
	SelectedInk:     graphics.Color(0x445064A1),
	HoverInk:        graphics.Color(0x334F566D),
	AlternateRowInk: graphics.Color(0x11FFFFFF),
	Font:            nil,
}

type ScrollPaneTheme struct {
	BackgroundInk   graphics.Color
	EdgeInk         graphics.Color
	TextInk         graphics.Color
	SelectedInk     graphics.Color
	HoverInk        graphics.Color
	AlternateRowInk graphics.Color
	CornerRadius    float32
	Font            graphics.Font
}

func (t *ScrollPaneTheme) Copy() *ScrollPaneTheme {
	return &ScrollPaneTheme{
		BackgroundInk:   t.BackgroundInk,
		EdgeInk:         t.EdgeInk,
		TextInk:         t.TextInk,
		SelectedInk:     t.SelectedInk,
		HoverInk:        t.HoverInk,
		AlternateRowInk: t.AlternateRowInk,
		CornerRadius:    t.CornerRadius,
		Font:            t.Font,
	}
}

// TableTheme defines the visual style for Table widgets.
var DefaultTableTheme = &TableTheme{
	BackgroundInk:    graphics.Color(0xFF3A3D4C),
	EdgeInk:          graphics.Color(0xFF6272A4),
	TextInk:          graphics.Color(0xFFFFFFFF),
	SelectedInk:      graphics.Color(0x445064A1),
	HoverInk:         graphics.Color(0x334F566D),
	AlternateRowInk:  graphics.Color(0x11FFFFFF),
	SecondaryTextInk: graphics.Color(0xFF8A919E),
	GridInk:          graphics.Color(0xFF4B5163),
	HeaderBgInk:      graphics.Color(0xFF454B60),
	AccentInk:        graphics.Color(0xFF6272A4),
	SelectedTextInk:  graphics.Color(0xFFFFFFFF),
	Font:             nil,
}

type TableTheme struct {
	BackgroundInk   graphics.Color
	EdgeInk         graphics.Color
	TextInk         graphics.Color
	SelectedInk     graphics.Color
	HoverInk        graphics.Color
	AlternateRowInk graphics.Color
	// SecondaryTextInk: header labels, row numbers, de-emphasized text.
	SecondaryTextInk graphics.Color
	// GridInk: cell grid lines and row separators.
	GridInk graphics.Color
	// HeaderBgInk: column header background.
	HeaderBgInk graphics.Color
	// AccentInk: sort indicators, edit highlights, drop targets.
	AccentInk graphics.Color
	// SelectedTextInk: text drawn on selected/hovered rows.
	SelectedTextInk graphics.Color
	CornerRadius    float32
	Font            graphics.Font
}

func (t *TableTheme) Copy() *TableTheme {
	return &TableTheme{
		BackgroundInk:    t.BackgroundInk,
		EdgeInk:          t.EdgeInk,
		TextInk:          t.TextInk,
		SelectedInk:      t.SelectedInk,
		HoverInk:         t.HoverInk,
		AlternateRowInk:  t.AlternateRowInk,
		SecondaryTextInk: t.SecondaryTextInk,
		GridInk:          t.GridInk,
		HeaderBgInk:      t.HeaderBgInk,
		AccentInk:        t.AccentInk,
		SelectedTextInk:  t.SelectedTextInk,
		CornerRadius:     t.CornerRadius,
		Font:             t.Font,
	}
}

// ── StackPanelTheme ──────────────────────────────────────────────

var DefaultStackPanelTheme = &StackPanelTheme{
	// BackgroundInk is the splitter band's fill; transparent means the band is
	// only the divider line, which is what a modern theme wants. An imported
	// theme used to lighten the panel background into this, which painted a
	// bright strip along every split.
	BackgroundInk:    0x00000000,
	SplitterInk:      graphics.Color(0xFF3C3C3C),
	SplitterHoverInk: graphics.Color(0xFF5A6480),
	// GripInk 0 = the split is one clean line. Grips are opt-in decoration.
	GripInk:      0,
	GripSize:     4,
	SplitterSize: 4,
	// HoverColorAlwaysOn off: the divider uses SplitterInk and takes
	// SplitterHoverInk only while the pointer is on it.
	HoverColorAlwaysOn: false,
}

type StackPanelTheme struct {
	BackgroundInk graphics.Color
	// SplitterInk is the divider line between panels, SplitterHoverInk the
	// same line while the pointer is on it or it is being dragged. GripInk
	// is optional decoration on the handle (0 = the line alone).
	SplitterInk      graphics.Color
	SplitterHoverInk graphics.Color
	GripInk          graphics.Color
	GripSize         float32
	SplitterSize     float32
	CornerRadius     float32
	// HoverColorAlwaysOn draws every divider in SplitterHoverInk, so it
	// reads as a highlight that does not depend on where the pointer is.
	// The IntelliJ importer turns it on: an imported L&F's splitters are
	// near-invisible in its own palette (their colour is the same tone as the
	// surrounding chrome), and the accent line is what reads there.
	HoverColorAlwaysOn bool
}

func (t *StackPanelTheme) Copy() *StackPanelTheme {
	return &StackPanelTheme{
		BackgroundInk:    t.BackgroundInk,
		SplitterInk:      t.SplitterInk,
		SplitterHoverInk: t.SplitterHoverInk,
		GripInk:          t.GripInk,
		GripSize:         t.GripSize,
		SplitterSize:     t.SplitterSize,
		CornerRadius:     t.CornerRadius,

		HoverColorAlwaysOn: t.HoverColorAlwaysOn,
	}
}

// ── MessageBoxTheme ──────────────────────────────────────────────

var DefaultMessageBoxTheme = &MessageBoxTheme{
	BackgroundInk: graphics.Color(0xFF414450),
	EdgeInk:       graphics.Color(0xFF6272A4),
	TitleBgInk:    graphics.Color(0xFF454B60),
	TitleTextInk:  graphics.Color(0xFFFFFFFF),
	DividerInk:    graphics.Color(0xFF4B5163),
	BodyTextInk:   graphics.Color(0xFFC3C6C6),
	ButtonBgInk:   graphics.Color(0xFF4F566D),
	ButtonEdgeInk: graphics.Color(0xFF6272A4),
	ButtonTextInk: graphics.Color(0xFFFFFFFF),
	Font:          nil,
}

type MessageBoxTheme struct {
	BackgroundInk graphics.Color
	EdgeInk       graphics.Color
	TitleBgInk    graphics.Color
	TitleTextInk  graphics.Color
	DividerInk    graphics.Color
	BodyTextInk   graphics.Color
	ButtonBgInk   graphics.Color
	ButtonEdgeInk graphics.Color
	ButtonTextInk graphics.Color
	Font          graphics.Font
}

func (t *MessageBoxTheme) Copy() *MessageBoxTheme {
	return &MessageBoxTheme{
		BackgroundInk: t.BackgroundInk,
		EdgeInk:       t.EdgeInk,
		TitleBgInk:    t.TitleBgInk,
		TitleTextInk:  t.TitleTextInk,
		DividerInk:    t.DividerInk,
		BodyTextInk:   t.BodyTextInk,
		ButtonBgInk:   t.ButtonBgInk,
		ButtonEdgeInk: t.ButtonEdgeInk,
		ButtonTextInk: t.ButtonTextInk,
		Font:          t.Font,
	}
}

// ── InputDialogTheme ─────────────────────────────────────────────

var DefaultInputDialogTheme = &InputDialogTheme{
	BackgroundInk:    graphics.Color(0xFF414450),
	EdgeInk:          graphics.Color(0xFF6272A4),
	TitleBgInk:       graphics.Color(0xFF454B60),
	TitleTextInk:     graphics.Color(0xFFFFFFFF),
	DividerInk:       graphics.Color(0xFF4B5163),
	LabelTextInk:     graphics.Color(0xFFC3C6C6),
	InputBgInk:       graphics.Color(0xFF3A3D4C),
	InputBgFocused:   graphics.Color(0xFF414863),
	InputEdgeInk:     graphics.Color(0xFF6272A4),
	InputEdgeFocused: graphics.Color(0xFF8A9CC8),
	InputTextInk:     graphics.Color(0xFFFFFFFF),
	CursorInk:        graphics.Color(0xFFFFFFFF),
	ButtonBgInk:      graphics.Color(0xFF4F566D),
	ButtonEdgeInk:    graphics.Color(0xFF6272A4),
	ButtonTextInk:    graphics.Color(0xFFFFFFFF),
	Font:             nil,
}

type InputDialogTheme struct {
	BackgroundInk    graphics.Color
	EdgeInk          graphics.Color
	TitleBgInk       graphics.Color
	TitleTextInk     graphics.Color
	DividerInk       graphics.Color
	LabelTextInk     graphics.Color
	InputBgInk       graphics.Color
	InputBgFocused   graphics.Color
	InputEdgeInk     graphics.Color
	InputEdgeFocused graphics.Color
	InputTextInk     graphics.Color
	CursorInk        graphics.Color
	ButtonBgInk      graphics.Color
	ButtonEdgeInk    graphics.Color
	ButtonTextInk    graphics.Color
	Font             graphics.Font
}

func (t *InputDialogTheme) Copy() *InputDialogTheme {
	return &InputDialogTheme{
		BackgroundInk:    t.BackgroundInk,
		EdgeInk:          t.EdgeInk,
		TitleBgInk:       t.TitleBgInk,
		TitleTextInk:     t.TitleTextInk,
		DividerInk:       t.DividerInk,
		LabelTextInk:     t.LabelTextInk,
		InputBgInk:       t.InputBgInk,
		InputBgFocused:   t.InputBgFocused,
		InputEdgeInk:     t.InputEdgeInk,
		InputEdgeFocused: t.InputEdgeFocused,
		InputTextInk:     t.InputTextInk,
		CursorInk:        t.CursorInk,
		ButtonBgInk:      t.ButtonBgInk,
		ButtonEdgeInk:    t.ButtonEdgeInk,
		ButtonTextInk:    t.ButtonTextInk,
		Font:             t.Font,
	}
}

// ── PopupTheme ───────────────────────────────────────────────────

var DefaultPopupTheme = &PopupTheme{
	BackgroundInk:  graphics.Color(0xFF414450),
	EdgeInk:        graphics.Color(0xFF6272A4),
	TitleBgInk:     graphics.Color(0xFF454B60),
	TitleTextInk:   graphics.Color(0xFFFFFFFF),
	DividerInk:     graphics.Color(0xFF4B5163),
	HeadingTextInk: graphics.Color(0xFFFFFFFF),
	ShadowInk:      graphics.Color(0x60000000),
	Font:           nil,
}

type PopupTheme struct {
	BackgroundInk  graphics.Color
	EdgeInk        graphics.Color
	TitleBgInk     graphics.Color
	TitleTextInk   graphics.Color
	DividerInk     graphics.Color
	HeadingTextInk graphics.Color
	ShadowInk      graphics.Color
	Font           graphics.Font
}

func (t *PopupTheme) Copy() *PopupTheme {
	return &PopupTheme{
		BackgroundInk:  t.BackgroundInk,
		EdgeInk:        t.EdgeInk,
		TitleBgInk:     t.TitleBgInk,
		TitleTextInk:   t.TitleTextInk,
		DividerInk:     t.DividerInk,
		HeadingTextInk: t.HeadingTextInk,
		ShadowInk:      t.ShadowInk,
		Font:           t.Font,
	}
}

// ── MdiTheme ─────────────────────────────────────────────────────

var DefaultMdiTheme = &MdiTheme{
	ContainerBgInk:   graphics.Color(0xFF1A1A1A),
	TabBgInk:         graphics.Color(0xFF333333),
	TabActiveBgInk:   graphics.Color(0xFF2D2D2D),
	TabEdgeInk:       graphics.Color(0xFF444444),
	TabTextInk:       graphics.Color(0xFF999999),
	TabActiveTextInk: graphics.Color(0xFFFFFFFF),
	TaskbarBgInk:     graphics.Color(0xFF3A3A3A),
	TaskbarTextInk:   graphics.Color(0xFF888888),
	TaskbarActiveInk: graphics.Color(0xFF4FC3F7),
	TaskbarBtnBgInk:  graphics.Color(0xFF3A3A3A),
	TaskbarBtnHover:  graphics.Color(0xFF2A2A2A),
	TaskbarBtnEdge:   graphics.Color(0xFF555555),
	DockBgInk:        graphics.Color(0xFF222222),
	DockEdgeInk:      graphics.Color(0xFF3A3A3A),
	DockSplitInk:     graphics.Color(0xFF4A4A4A),
	WinTitleBgInk:    graphics.Color(0xFF3A3A3A),
	WinTitleInactive: graphics.Color(0xFF2D2D2D),
	WinTitleEdgeInk:  graphics.Color(0xFF444444),
	WinTitleTextInk:  graphics.Color(0xFFE0E0E0),
	WinEdgeInk:       graphics.Color(0xFF555555),
	WinContentBgInk:  graphics.Color(0xFF1E1E1E),
	WinAccentInk:     graphics.Color(0xFF4FC3F7),
	WinTextSecondary: graphics.Color(0xFF888888),
	WinShadowInk:     graphics.Color(0x40000000),
	Font:             nil,
}

type MdiTheme struct {
	ContainerBgInk   graphics.Color
	TabBgInk         graphics.Color
	TabActiveBgInk   graphics.Color
	TabEdgeInk       graphics.Color
	TabTextInk       graphics.Color
	TabActiveTextInk graphics.Color
	TaskbarBgInk     graphics.Color
	TaskbarTextInk   graphics.Color
	TaskbarActiveInk graphics.Color
	TaskbarBtnBgInk  graphics.Color
	TaskbarBtnHover  graphics.Color
	TaskbarBtnEdge   graphics.Color
	DockBgInk        graphics.Color
	DockEdgeInk      graphics.Color
	DockSplitInk     graphics.Color
	WinTitleBgInk    graphics.Color
	WinTitleInactive graphics.Color
	WinTitleEdgeInk  graphics.Color
	WinTitleTextInk  graphics.Color
	WinEdgeInk       graphics.Color
	WinContentBgInk  graphics.Color
	WinAccentInk     graphics.Color
	WinTextSecondary graphics.Color
	WinShadowInk     graphics.Color
	CornerRadius     float32
	Font             graphics.Font
}

func (t *MdiTheme) Copy() *MdiTheme {
	return &MdiTheme{
		ContainerBgInk:   t.ContainerBgInk,
		TabBgInk:         t.TabBgInk,
		TabActiveBgInk:   t.TabActiveBgInk,
		TabEdgeInk:       t.TabEdgeInk,
		TabTextInk:       t.TabTextInk,
		TabActiveTextInk: t.TabActiveTextInk,
		TaskbarBgInk:     t.TaskbarBgInk,
		TaskbarTextInk:   t.TaskbarTextInk,
		TaskbarActiveInk: t.TaskbarActiveInk,
		TaskbarBtnBgInk:  t.TaskbarBtnBgInk,
		TaskbarBtnHover:  t.TaskbarBtnHover,
		TaskbarBtnEdge:   t.TaskbarBtnEdge,
		DockBgInk:        t.DockBgInk,
		DockEdgeInk:      t.DockEdgeInk,
		DockSplitInk:     t.DockSplitInk,
		WinTitleBgInk:    t.WinTitleBgInk,
		WinTitleInactive: t.WinTitleInactive,
		WinTitleEdgeInk:  t.WinTitleEdgeInk,
		WinTitleTextInk:  t.WinTitleTextInk,
		WinEdgeInk:       t.WinEdgeInk,
		WinContentBgInk:  t.WinContentBgInk,
		WinAccentInk:     t.WinAccentInk,
		WinTextSecondary: t.WinTextSecondary,
		WinShadowInk:     t.WinShadowInk,
		CornerRadius:     t.CornerRadius,
		Font:             t.Font,
	}
}
