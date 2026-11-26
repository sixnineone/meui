package theme

import "github.com/sixnineone/meui/graphics"

// Platform identifies the OS/desktop family a preset targets. It
// selects chrome style, metrics and motion, never behavior.
type Platform string

const (
	PlatformMeui      Platform = "meui"
	PlatformMacOS     Platform = "macos"
	PlatformWindows11 Platform = "windows11"
	PlatformMetro     Platform = "metro"
	PlatformGNOME     Platform = "gnome"
	PlatformKDE       Platform = "kde"
	// The mobile families. They are LOOK presets like every other entry here — the
	// behaviour that follows from being on a phone (touch targets, sheets, back
	// navigation) belongs to platform.FormFactor, not to the theme.
	PlatformAndroid Platform = "android"
	PlatformIOS     Platform = "ios"
	// PlatformWeb is the look of a web page rather than a desktop window: an 8px
	// grid, a document type scale, larger radii and a card elevation. Like the
	// mobile families it is a LOOK preset — the behaviour a page needs (scrolling,
	// hover, focus-visible) belongs to the widgets, not to the theme.
	PlatformWeb Platform = "web"
)

// Mode selects the light/dark variant of a preset.
type Mode uint8

const (
	ModeDark Mode = iota
	ModeLight
	ModeHighContrast
)

func (m Mode) String() string {
	switch m {
	case ModeLight:
		return "light"
	case ModeHighContrast:
		return "high-contrast"
	default:
		return "dark"
	}
}

// Density is the control-size class of a preset. It documents the
// spacing feel; the concrete numbers live in Metrics so presets stay
// explicit.
type Density uint8

const (
	DensityRegular Density = iota
	DensityCompact
	DensityComfortable
)

// Easing names a motion curve. Mapping to concrete beziers happens in
// the animation layer (Phase 3), this token names the intent.
type Easing uint8

const (
	EaseStandard Easing = iota // platform default curve
	EaseDecelerate
	EaseAccelerate
	EaseLinear
	EaseSpring
)

// Timing pairs a duration with an easing intent.
type Timing struct {
	DurationMS float32
	Easing     Easing
}

// Motion holds the interaction timings. Platforms differ (Fluent
// 83–300ms, macOS 120–250ms, Adwaita ~200ms, Metro near-instant).
type Motion struct {
	Hover, Press, Focus, Toggle Timing
	MenuOpen, MenuClose         Timing
	TooltipShow, TooltipHide    Timing
	ScrollFade                  Timing
}

// Metrics is the density/geometry token set: control heights, paddings
// and small-element sizes in logical pixels. The generator pushes these
// into every widget theme dimension field; widgets stop hardcoding
// their own.
type Metrics struct {
	Density            Density
	ControlHeight      float32 // buttons, inputs, combo, spinbox
	ControlPadX        float32 // button horizontal content padding
	ControlPadY        float32 // button vertical content padding
	InputPadX          float32 // text field / text area horizontal padding
	MenuItemHeight     float32
	ListRowHeight      float32
	TabHeight          float32
	ToolbarHeight      float32
	StatusBarHeight    float32
	ScrollbarWidth     float32
	ScrollbarOverlay   bool // thin floating pill, NO track band (macOS/GNOME)
	ScrollbarAutoHide  bool // bars fade out when idle (opt-in; default: persistent)
	BorderWidth        float32
	FocusRingWidth     float32
	FocusRingOffset    float32
	BoxSize            float32 // checkbox/radio box
	BoxSpacing         float32 // box-to-label gap
	SliderThumb        float32 // slider knob diameter
	ArrowSize          float32 // combo/spin arrows
	GripSize           float32 // splitview/stackpanel grips
	ControlBorderWidth float32 // button/input border (0 = borderless, Metro)
	TabIndicatorHeight float32 // active-tab underline (0 = fill-only platforms)
	TabActiveFilled    bool    // active tab gets a lighter fill (macOS/GNOME/Breeze)
	ArrowStyle         uint8   // 0 = triangle, 1 = chevron (Fluent/macOS/GNOME/KDE)
	SwitchH            float32 // toggle track height (width = 2x)
	SliderTrack        float32 // slider groove thickness (0 = meui 6)
	SliderTrackRadius  float32 // -1 = pill
	ProgressHeight     float32 // progress bar thickness (0 = meui 6)
	ProgressRadius     float32 // -1 = pill
	// NavBarHeight is a bottom navigation bar's height: 64 on Android, 49 on iOS. It is
	// its own metric because a tab strip and a bottom bar are different controls with
	// different heights on the same platform, and borrowing one for the other is how
	// they silently drift. Zero falls back to TabHeight.
	NavBarHeight float32
	// NavRailWidth is a navigation rail's width: the column the same destinations occupy
	// when there is room beside the content instead of below it.
	NavRailWidth float32
	// MinTouchTarget is the minimum hit area for an interactive widget, in logical
	// px: 48 on Android, 44 on iOS, 0 on a pointer-driven desktop where a small
	// control is not a usability problem. It is a METRIC because it is a size, but
	// it is enforced in one place — the hit test — rather than by every widget.
	MinTouchTarget float32
	Shape          *ShapeTheme
}

// InputStyle selects the text-field construction.
type InputStyle uint8

const (
	InputOutlined InputStyle = iota // full 1px border (all desktop platforms)
	InputFilled                     // fill only (reserved)
)

// InputTokens are the text-input construction details beyond colours:
// Fluent outlines its TextBox with an emphasised bottom edge (1px
// resting, 2px accent on focus); the other platforms use a plain
// outline.
type InputTokens struct {
	Style            InputStyle
	BottomInk        graphics.Color // emphasised bottom border (0 = none)
	BottomWidth      float32
	FocusBottomWidth float32 // accent bottom on focus (0 = use BottomWidth)
}

// TextStyle is one typographic role.
type TextStyle struct {
	Size          float32
	Weight        int
	LineHeight    float32
	LetterSpacing float32
}

// Typography carries the ordered family stack (first installed wins)
// and the role styles. BaseFont remains the resolved face for backward
// compatibility; presets fill both.
type Typography struct {
	Families []string
	Caption  TextStyle
	Body     TextStyle
	Label    TextStyle
	Button   TextStyle
	Title    TextStyle
	Heading  TextStyle
	Mono     TextStyle
}

// FocusKind names the platform focus indicator construction.
type FocusKind uint8

const (
	FocusNone      FocusKind = iota
	FocusRing                // outer accent ring (macOS, Adwaita)
	FocusInset               // inner border replacement (Breeze)
	FocusTwoTone             // inner+outer contrasting rect (Fluent)
	FocusUnderline           // underline only (Metro-ish)
)

// FocusStyle is the focus indicator token set.
type FocusStyle struct {
	Kind   FocusKind
	Width  float32
	Offset float32
	Color  graphics.Color // accent ring / outer tone
	Alt    graphics.Color // inner tone (two-tone)
}

// ShadowSpec is one drop-shadow description. Blur/Spread/DX/DY in
// logical pixels; alpha lives in Color.
type ShadowSpec struct {
	Color        graphics.Color
	DX, DY       float32
	Blur, Spread float32
}

// Material names the window backdrop treatment. Whether it can be
// honored depends on the compositor; the token records intent.
type Material uint8

const (
	MaterialNone Material = iota
	MaterialTranslucent
	MaterialMica
	MaterialAcrylic
	MaterialVibrancy
)

// Effects groups shadows, focus indicator and materials.
type Effects struct {
	WindowShadow ShadowSpec
	PopupShadow  ShadowSpec
	MenuShadow   ShadowSpec
	// CardShadow is the elevation of a surface inside the page: a web card, a
	// panel, a raised row. It is its own token because a popup's shadow is
	// deliberately heavier than a card's, and borrowing one for the other is how
	// every surface ends up looking like a floating dialog.
	CardShadow    ShadowSpec
	Focus         FocusStyle
	Material      Material
	WindowOpacity float32
}

// ChromeStyle selects the window-button construction. WindowFrame
// consumes this in Phase 2; "meui" and "macos" are already implemented
// there, the rest are the platform targets.
type ChromeStyle string

const (
	ChromeMeui    ChromeStyle = "meui"
	ChromeMacOS   ChromeStyle = "macos"
	ChromeWindows ChromeStyle = "windows"
	ChromeGNOME   ChromeStyle = "gnome"
	ChromeKDE     ChromeStyle = "kde"
	ChromeMetro   ChromeStyle = "metro"
)

// TitleAlign is the window-title placement.
type TitleAlign uint8

const (
	TitleLeft TitleAlign = iota
	TitleCenter
)

// Chrome is the window-decoration token set.
type Chrome struct {
	Style        ChromeStyle
	Height       float32
	ButtonW      float32
	ButtonH      float32
	ButtonGap    float32
	HoverBg      graphics.Color
	PressedBg    graphics.Color
	CloseHoverBg graphics.Color
	GlyphColor   graphics.Color
	TitleAlign   TitleAlign
	TitleWeight  int

	// BorderWidth/BorderInk draw one outline around the whole window, the
	// chrome included. Width 0 means none (macOS windows have no outline);
	// an ink of 0 falls back to the theme's Border colour, so an imported
	// theme's palette still drives it.
	BorderWidth float32
	BorderInk   graphics.Color
}

// Meta identifies a loaded preset.
type Meta struct {
	ID       string
	Platform Platform
	Mode     Mode
}

// DefaultMetrics returns the meui/current geometry so themes that do
// not define metrics (IntelliJ imports, DarkTheme) keep their sizing.
func DefaultMetrics() Metrics {
	return Metrics{
		Density:         DensityRegular,
		ControlHeight:   26,
		ControlPadX:     8,
		ControlPadY:     3,
		InputPadX:       6,
		MenuItemHeight:  24,
		ListRowHeight:   30,
		TabHeight:       24,
		ToolbarHeight:   32,
		StatusBarHeight: 22,
		ScrollbarWidth:  14,
		// Overlay bars by default: the meui look is a floating pill,
		// with NO full-height track band running down lists and trees.
		// The pill is PERSISTENT — auto-hide is opt-in via
		// ScrollbarAutoHide (macOS/GNOME presets).
		ScrollbarOverlay:   true,
		BorderWidth:        1,
		FocusRingWidth:     1.5,
		FocusRingOffset:    0,
		BoxSize:            14,
		BoxSpacing:         4,
		SliderThumb:        16,
		ArrowSize:          10,
		GripSize:           4,
		ControlBorderWidth: 1,
		TabIndicatorHeight: 1.5,
		TabActiveFilled:    true,
		ArrowStyle:         0,
		Shape:              ShapeMeui(),
	}
}

// ControlTokens are the slider and progress-bar constructions beyond
// colours: track thickness/shape, thumb ring/inner dot, bar height.
type ControlTokens struct {
	SliderTrackH      float32 // track thickness
	SliderTrackRadius float32 // -1 = pill (height/2), 0 = square (Metro)
	SliderTrackBorder bool    // Breeze outlines its groove
	SliderThumbBorder graphics.Color
	SliderThumbInner  graphics.Color // Fluent white thumb + accent dot
	ProgressHeight    float32
	ProgressRadius    float32 // -1 = pill
}

// DefaultControlTokens returns the meui constructions.
func DefaultControlTokens() ControlTokens {
	return ControlTokens{
		SliderTrackH:      6,
		SliderTrackRadius: 0,
		SliderTrackBorder: true,
		ProgressHeight:    6,
		ProgressRadius:    4,
	}
}

// DefaultMotion returns the meui timings.
func DefaultMotion() Motion {
	std := Timing{DurationMS: 150, Easing: EaseStandard}
	fast := Timing{DurationMS: 100, Easing: EaseStandard}
	return Motion{
		Hover:       fast,
		Press:       Timing{DurationMS: 80, Easing: EaseStandard},
		Focus:       fast,
		Toggle:      std,
		MenuOpen:    std,
		MenuClose:   Timing{DurationMS: 120, Easing: EaseAccelerate},
		TooltipShow: Timing{DurationMS: 400, Easing: EaseDecelerate},
		TooltipHide: Timing{DurationMS: 100, Easing: EaseStandard},
		ScrollFade:  Timing{DurationMS: 600, Easing: EaseDecelerate},
	}
}

// DefaultEffects returns the meui effects (accent focus ring, soft
// popup shadow, no window material).
func DefaultEffects() Effects {
	return Effects{
		CardShadow:   ShadowSpec{Color: 0x14000000, DY: 1, Blur: 4},
		PopupShadow:  ShadowSpec{Color: 0x40000000, DY: 4, Blur: 16},
		MenuShadow:   ShadowSpec{Color: 0x40000000, DY: 4, Blur: 16},
		WindowShadow: ShadowSpec{Color: 0x60000000, DY: 8, Blur: 24},
		Focus:        FocusStyle{Kind: FocusRing, Width: 1.5, Color: 0xFF8A9CC8},
		Material:     MaterialNone,
	}
}

// DefaultChrome returns the meui title bar tokens.
func DefaultChrome() Chrome {
	return Chrome{
		Style:        ChromeMeui,
		Height:       28,
		ButtonW:      24,
		ButtonH:      24,
		ButtonGap:    2,
		HoverBg:      0x22FFFFFF,
		PressedBg:    0x33FFFFFF,
		CloseHoverBg: 0xFFC75454,
		GlyphColor:   0xFFC3C6C6,
		TitleAlign:   TitleLeft,
		TitleWeight:  400,
		BorderWidth:  1,
	}
}

// sliderThumbInk picks the base thumb colour: platforms with an inner
// dot / border ring use a surface-coloured thumb (white circle +
// accent dot), others fill with the accent.
func (c ControlTokens) sliderThumbInk(surface graphics.Color) graphics.Color {
	if c.SliderThumbInner != 0 || c.SliderThumbBorder != 0 {
		return surface
	}
	return 0
}

// Metric accessors with the meui defaults for zero (IntelliJ themes).
func (m Metrics) sliderTrackHeight() float32 {
	if m.SliderTrack <= 0 {
		return 6
	}
	return m.SliderTrack
}

func (m Metrics) sliderTrackRadius() float32 {
	if m.SliderTrack <= 0 {
		return 0 // meui default draws a square groove
	}
	if m.SliderTrackRadius < 0 {
		return m.SliderTrack / 2
	}
	return m.SliderTrackRadius
}

func (m Metrics) progressHeight() float32 {
	if m.ProgressHeight <= 0 {
		return 6
	}
	return m.ProgressHeight
}

func (m Metrics) progressRadius() float32 {
	if m.SliderTrack <= 0 {
		return 4 // meui default
	}
	if m.ProgressRadius < 0 {
		return m.progressHeight() / 2
	}
	return m.ProgressRadius
}
