package theme

import (
	"log"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/slant"
	"github.com/sixnineone/meui/graphics/enums/spacing"
	"github.com/sixnineone/meui/graphics/enums/weight"
)

var (
	DefaultTheme *Theme // The active theme, initialized to DarkTheme() in init()
	themeVersion uint64 // incremented on each SetDefaultTheme for cache invalidation

	themeChangedMu        sync.Mutex
	themeChangedListeners []func()
)

// ThemeVersion returns the current global theme version. Used by cache invalidation.
func ThemeVersion() uint64 { return themeVersion }

// OnThemeChanged registers fn to run whenever SetDefaultTheme is
// called. Multiple listeners are supported: the window layer repaints
// every open window and any module (charts, app code) may register
// its own sync.
//
// This was previously a single exported func variable, so whichever
// package assigned it last silently disabled the others — the same
// clobber that broke live theme switching in the production stack
// (meui-charts replaced meui's window-repaint callback). Registration
// is now additive.
func OnThemeChanged(fn func()) {
	if fn == nil {
		return
	}
	themeChangedMu.Lock()
	themeChangedListeners = append(themeChangedListeners, fn)
	themeChangedMu.Unlock()
}

type Theme struct {
	Name           string
	Primary        graphics.Color
	Secondary      graphics.Color
	Accent         graphics.Color
	OnAccent       graphics.Color // text/glyph on accent fills (0 = derive)
	Background     graphics.Color
	Surface        graphics.Color
	SurfaceVariant graphics.Color
	TextPrimary    graphics.Color
	TextSecondary  graphics.Color
	Border         graphics.Color
	BorderStrong   graphics.Color // control strokes (fields, boxes)
	BorderSubtle   graphics.Color // separators, table grids
	BorderFocused  graphics.Color
	BorderHovered  graphics.Color
	Hovered        graphics.Color
	Pressed        graphics.Color
	Selected       graphics.Color
	// SelectedWash is the translucent accent a SELECTED SURFACE is washed with (a nav
	// item, a list row, a day cell). It is separate from Selected, which is an opaque
	// fill role used for selected text and checked states — washing a row with that
	// paints solid accent over its own label. Zero means "derive from Selected".
	SelectedWash      graphics.Color
	Disabled          graphics.Color
	DisabledText      graphics.Color
	DisabledBg        graphics.Color
	DisabledBorder    graphics.Color
	BorderRadius      float32
	BaseFont          graphics.Font
	FontSizeSmall     float32
	FontSizeMedium    float32
	FontSizeLarge     float32
	FontSizeTitle     float32
	FontWeightNormal  int
	FontWeightBold    int
	LineHeight        float32
	Success           graphics.Color
	Warning           graphics.Color
	Error             graphics.Color
	Shadow            graphics.Color
	TooltipBackground graphics.Color
	TooltipText       graphics.Color
	Link              graphics.Color
	LinkVisited       graphics.Color
	MenuBackground    graphics.Color
	MenuHighlight     graphics.Color
	ScrollBarTrack    graphics.Color
	ScrollBarThumb    graphics.Color
	ScrollbarWidth    float32
	ChartGrid         graphics.Color
	ChartAxis         graphics.Color

	// Platform token groups (see THEME_ARCHITECTURE.md). Color roles
	// above stay the palette; these carry the non-color dimensions that
	// distinguish macOS / Windows / GNOME / KDE / Metro.
	Meta       Meta
	Metrics    Metrics
	Motion     Motion
	Typography Typography
	Effects    Effects
	Chrome     Chrome
	// SplitterHoverColorAlwaysOn asks split panels to keep their dividers in
	// the accent colour instead of only while the pointer is on them. It is a
	// property of the L&F, not of the widget: an imported IntelliJ palette
	// paints its separators in the same tone as the surrounding chrome, so a
	// hover-only highlight leaves every split invisible until it is found.
	// It feeds StackPanelTheme.HoverColorAlwaysOn on Apply.
	SplitterHoverColorAlwaysOn bool
	Inputs                     InputTokens
	Controls                   ControlTokens
}

// TokenColor resolves a QML style token ("text", "secondary",
// "disabled", "accent", "error", "warning", "success", "link",
// "border", "background", "surface") against the ACTIVE theme. Widgets
// that keep a token re-resolve it per paint, so runtime theme switches
// recolour them; parse-time resolution would freeze the old palette.
func TokenColor(name string) (graphics.Color, bool) {
	t := DefaultTheme
	if t == nil {
		return 0, false
	}
	switch strings.ToLower(strings.TrimPrefix(strings.TrimSpace(name), "@")) {
	case "text", "primary":
		return t.TextPrimary, true
	case "secondary":
		return t.TextSecondary, true
	case "disabled":
		return t.DisabledText, true
	case "accent":
		return t.Accent, true
	case "error":
		return t.Error, true
	case "warning":
		return t.Warning, true
	case "success":
		return t.Success, true
	case "link":
		return t.Link, true
	case "border":
		return t.Border, true
	case "background":
		return t.Background, true
	case "surface":
		return t.Surface, true
	case "onsurface", "on-surface":
		return t.TextPrimary, true
	}
	return 0, false
}

// OnAccentColor returns the text/glyph colour that sits on top of the
// accent (checked boxes, active buttons, solid selections). Presets set
// it explicitly where the derivation is wrong (Fluent dark uses black on
// its light accent); otherwise it is derived from the accent luminance.
func (t *Theme) OnAccentColor() graphics.Color {
	if t == nil {
		return 0xFFFFFFFF
	}
	if t.OnAccent != 0 {
		return t.OnAccent
	}
	r := (t.Accent >> 16) & 0xFF
	g := (t.Accent >> 8) & 0xFF
	b := t.Accent & 0xFF
	if (r*299+g*587+b*114)/1000 > 150 {
		return 0xFF000000
	}
	return 0xFFFFFFFF
}

// withStyleDefaults fills any unset token group with the meui defaults.
// IntelliJ imports and hand-built themes only set colors, so the
// generator must never see zero tokens.
func (t *Theme) withStyleDefaults() *Theme {
	if t.Metrics.ControlHeight == 0 {
		t.Metrics = DefaultMetrics()
	}
	if t.Metrics.Shape == nil {
		t.Metrics.Shape = ShapeMeui()
	}
	if t.Motion.Hover.DurationMS == 0 {
		t.Motion = DefaultMotion()
	}
	if t.Effects.Focus.Width == 0 && t.Effects.Focus.Kind == FocusNone {
		t.Effects = DefaultEffects()
	}
	if t.Chrome.Height == 0 {
		t.Chrome = DefaultChrome()
	}
	if t.Typography.Body.Size == 0 {
		t.Typography = typoMeui()
	}
	if t.Controls.SliderTrackH == 0 {
		t.Controls = DefaultControlTokens()
	}
	if t.SelectedWash == 0 && t.Selected != 0 {
		// A selected surface is a WASH: keep the accent's hue, make it translucent.
		a := (t.Selected >> 24) & 0xFF
		if a >= 0x80 {
			t.SelectedWash = (t.Selected & 0x00FFFFFF) | 0x1F000000
		} else {
			t.SelectedWash = t.Selected
		}
	}
	return t
}

// Border stroke width system defaults (variables so they can be overridden)
var (
	BorderStrokeWidth    = float32(1.0) // Standard widget borders (FlatLaf: crisp 1px outlines)
	FocusStrokeWidth     = float32(1.5) // Focus rings
	CheckStrokeWidth     = float32(2.0) // Checkbox checkmarks
	SeparatorStrokeWidth = float32(1.0) // Separators/dividers
	CursorStrokeWidth    = float32(2.0) // Cursor strokes
)

func init() {
	DefaultTheme = DarkTheme()
}

// UseSystemFont re-points the default theme at the CURRENT system font and
// invalidates cached text.
//
// It is the second half of changing a font at runtime. graphics.ReloadSystemFont
// re-resolves the font; this makes the theme use it. Both are needed and neither
// package can do the other's half — theme imports graphics, not the reverse — and
// a swap that misses either one leaves the app drawing the old glyphs with no
// error anywhere.
//
// The version bump is not optional: widgets cache the face they were built with,
// so re-pointing BaseFont without invalidating leaves every already-laid-out label
// on the old font.
func UseSystemFont() {
	if DefaultTheme == nil {
		return
	}
	// Re-resolve at the theme's OWN typography size. Assigning graphics.SystemFont
	// directly would swap in the platform's default size as well as the face, so
	// every glyph's metrics would change — a different font AND a different size,
	// which is a worse error than the one being fixed. (Measured: it made the
	// text-heavy parity deltas worse rather than better.)
	// Apply the theme's style defaults FIRST. Reading Body.Size before them gives
	// zero, and face.Font(0) is a zero-size font — measured: the base font's size
	// was 0.00 on the device against 6.00 on the desktop, with a degenerate
	// 5.97px width for "Normal text".
	DefaultTheme.withStyleDefaults()
	size := DefaultTheme.Typography.Body.Size
	if face := graphics.SystemFont.Face(); face != nil {
		DefaultTheme.BaseFont = face.Font(size)
	} else {
		DefaultTheme.BaseFont = graphics.SystemFont
	}
	SetDefaultTheme(DefaultTheme)
}

func SetDefaultTheme(t *Theme) {
	if t != nil {
		t.withStyleDefaults()
	}
	themeVersion++
	themeAudit("DefaultTheme = t")
	DefaultTheme = t
	notifyThemeChanged()
}

// notifyThemeChanged fires every OnThemeChanged listener. Shape
// changes reuse this so open windows repaint.
func notifyThemeChanged() {
	themeChangedMu.Lock()
	listeners := append([]func(){}, themeChangedListeners...)
	themeChangedMu.Unlock()
	for _, fn := range listeners {
		fn()
	}
}

func DarkTheme() *Theme {
	// Use Inter font (FlatLaf default) with compact sizing
	var baseFont graphics.Font

	// Try Inter first (FlatLaf's default font)
	interFace := graphics.MatchFontFace("Inter", weight.Light, spacing.Standard, slant.Upright)
	if interFace != nil {
		baseFont = interFace.Font(11) // Compact size matching FlatLaf
	} else {
		// Fall back to system font with reduced size
		baseSize := float32(10)
		sysFont := graphics.MatchFontFace("Roboto", weight.Light, spacing.Standard, slant.Upright).Font(baseSize)
		if sysFont != nil {
			face := sysFont.Face()
			size := sysFont.Size()
			newSize := size - 4
			if newSize < 1 {
				newSize = 1
			}
			baseFont = face.Font(newSize)
		} else {
			baseFont = graphics.SystemFont
		}
	}

	return &Theme{
		Name:              "flatlaf-dracula",
		Primary:           graphics.Color(0xFF5064A1), // Dracula blue accent
		Secondary:         graphics.Color(0xFF6A8759), // Dracula green
		Accent:            graphics.Color(0xFF6272A4), // Dracula blue-gray
		OnAccent:          graphics.Color(0xFFFFFFFF),
		Background:        graphics.Color(0xFF414450), // FlatLaf Dracula panel
		Surface:           graphics.Color(0xFF3A3D4C), // Field bg
		SurfaceVariant:    graphics.Color(0xFF454B60), // Hover/variant
		TextPrimary:       graphics.Color(0xFFC3C6C6), // Dracula fg
		TextSecondary:     graphics.Color(0xFF8A919E), // Secondary text
		Border:            graphics.Color(0xFF6272A4), // Dracula border
		BorderFocused:     graphics.Color(0xFF8A9CC8), // Focus blue
		BorderHovered:     graphics.Color(0xFF7B84A8),
		Hovered:           graphics.Color(0x334F566D),
		Pressed:           graphics.Color(0x55454B60),
		Selected:          graphics.Color(0x445064A1),
		Disabled:          graphics.Color(0xFF6F737A),
		DisabledText:      graphics.Color(0xFF6F737A),
		DisabledBg:        graphics.Color(0xFF3A3D4C),
		DisabledBorder:    graphics.Color(0xFF4B5163),
		BorderRadius:      4,
		BaseFont:          baseFont,
		FontSizeSmall:     8,
		FontSizeMedium:    10,
		FontSizeLarge:     12,
		FontSizeTitle:     14,
		FontWeightNormal:  400,
		FontWeightBold:    700,
		LineHeight:        1.4,
		Success:           graphics.Color(0xFF6A8759),
		Warning:           graphics.Color(0xFFC9A268),
		Error:             graphics.Color(0xFFC75454), // Dracula red
		Shadow:            graphics.Color(0x33000000),
		TooltipBackground: graphics.Color(0xFF3A3D4C),
		TooltipText:       graphics.Color(0xFFC3C6C6),
		Link:              graphics.Color(0xFF8A9CC8),
		LinkVisited:       graphics.Color(0xFFBD93F9), // Dracula purple
		MenuBackground:    graphics.Color(0xFF414450),
		MenuHighlight:     graphics.Color(0xFF4F566D),
		ScrollBarTrack:    graphics.Color(0xFF3A3D4C),
		ScrollBarThumb:    graphics.Color(0xFF5A6480),
		ScrollbarWidth:    14,
		ChartGrid:         graphics.Color(0xFF4B5163),
		ChartAxis:         graphics.Color(0xFF8A919E),
	}
}

func LightTheme() *Theme {
	return &Theme{
		Name:              "light",
		Primary:           graphics.Color(0xFF1976D2),
		Secondary:         graphics.Color(0xFF018786),
		Accent:            graphics.Color(0xFFE64A19),
		OnAccent:          graphics.Color(0xFFFFFFFF),
		Background:        graphics.Color(0xFFF5F5F5),
		Surface:           graphics.Color(0xFFFFFFFF),
		SurfaceVariant:    graphics.Color(0xFFE8E8E8),
		TextPrimary:       graphics.Color(0xFF111111),
		TextSecondary:     graphics.Color(0xFF555555),
		Border:            graphics.Color(0xFF999999),
		BorderFocused:     graphics.Color(0xFF1976D2),
		BorderHovered:     graphics.Color(0xFF888888),
		Hovered:           graphics.Color(0x1A000000),
		Pressed:           graphics.Color(0x33000000),
		Selected:          graphics.Color(0x2E000000),
		Disabled:          graphics.Color(0xFFBDBDBD),
		DisabledText:      graphics.Color(0xFF9E9E9E),
		DisabledBg:        graphics.Color(0xFFE8E8E8),
		DisabledBorder:    graphics.Color(0xFFD0D0D0),
		BorderRadius:      4,
		BaseFont:          graphics.SystemFont,
		FontSizeSmall:     8,
		FontSizeMedium:    10,
		FontSizeLarge:     12,
		FontSizeTitle:     14,
		FontWeightNormal:  400,
		FontWeightBold:    700,
		LineHeight:        1.4,
		Success:           graphics.Color(0xFF4CAF50),
		Warning:           graphics.Color(0xFFFFC107),
		Error:             graphics.Color(0xFFF44336),
		Shadow:            graphics.Color(0x1A000000),
		TooltipBackground: graphics.Color(0xFFF5F5F5),
		TooltipText:       graphics.Color(0xFF111111),
		Link:              graphics.Color(0xFF1976D2),
		LinkVisited:       graphics.Color(0xFF7B1FA2),
		MenuBackground:    graphics.Color(0xFFFFFFFF),
		MenuHighlight:     graphics.Color(0xFFE8E8E8),
		ScrollBarTrack:    graphics.Color(0xFFE0E0E0),
		ScrollBarThumb:    graphics.Color(0xFF999999),
		ScrollbarWidth:    14,
		ChartGrid:         graphics.Color(0xFFE0E0E0),
		ChartAxis:         graphics.Color(0xFF555555),
	}
}

func (t *Theme) SetPrimary(color graphics.Color) *Theme {
	t.Primary = color
	return t
}

func (t *Theme) SetSecondary(color graphics.Color) *Theme {
	t.Secondary = color
	return t
}

func (t *Theme) SetBackground(color graphics.Color) *Theme {
	t.Background = color
	return t
}

func (t *Theme) SetSurface(color graphics.Color) *Theme {
	t.Surface = color
	return t
}

func (t *Theme) SetTextPrimary(color graphics.Color) *Theme {
	t.TextPrimary = color
	return t
}

func (t *Theme) SetBorderRadius(radius float32) *Theme {
	t.BorderRadius = radius
	return t
}

func (t *Theme) SetBaseFont(font graphics.Font) *Theme {
	t.BaseFont = font
	return t
}

func (t *Theme) SetFontSizeSmall(size float32) *Theme {
	t.FontSizeSmall = size
	return t
}

func (t *Theme) SetFontSizeMedium(size float32) *Theme {
	t.FontSizeMedium = size
	return t
}

func (t *Theme) SetFontSizeLarge(size float32) *Theme {
	t.FontSizeLarge = size
	return t
}

func (t *Theme) SetFontSizeTitle(size float32) *Theme {
	t.FontSizeTitle = size
	return t
}

// ScaleFonts scales all font sizes in the theme by the given factor.
// Called when the window moves to a display with a different DPI scale.
// themeAudit logs a mutation of process-wide theme state with a short stack.
// Every window's layout and paint read the default theme, so a mutation here can
// rearrange every window at once - the shape of the reported contamination.
func themeAudit(what string) {
	if os.Getenv("MEUI_THEME_AUDIT") == "" {
		return
	}
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	log.Printf("[theme] MUTATION %s\n%s", what, buf[:n])
}

func ScaleFonts(t *Theme, scale float32) {
	themeAudit("ScaleFonts")
	t.FontSizeSmall = 6 * scale
	t.FontSizeMedium = 8 * scale
	t.FontSizeLarge = 10 * scale
	t.FontSizeTitle = 12 * scale

	if t.BaseFont != nil {
		face := t.BaseFont.Face()
		if face != nil {
			t.BaseFont = face.Font(8 * scale)
		}
	}
	themeVersion++ // invalidate text and image caches
}

func (t *Theme) SetFontWeightNormal(weight int) *Theme {
	t.FontWeightNormal = weight
	return t
}

func (t *Theme) SetFontWeightBold(weight int) *Theme {
	t.FontWeightBold = weight
	return t
}

func (t *Theme) SetLineHeight(height float32) *Theme {
	t.LineHeight = height
	return t
}

func (t *Theme) SetShadow(color graphics.Color) *Theme {
	t.Shadow = color
	return t
}

func (t *Theme) SetTooltipBackground(color graphics.Color) *Theme {
	t.TooltipBackground = color
	return t
}

func (t *Theme) SetTooltipText(color graphics.Color) *Theme {
	t.TooltipText = color
	return t
}

func (t *Theme) SetLink(color graphics.Color) *Theme {
	t.Link = color
	return t
}

func (t *Theme) SetLinkVisited(color graphics.Color) *Theme {
	t.LinkVisited = color
	return t
}

func (t *Theme) SetMenuBackground(color graphics.Color) *Theme {
	t.MenuBackground = color
	return t
}

func (t *Theme) SetMenuHighlight(color graphics.Color) *Theme {
	t.MenuHighlight = color
	return t
}

func (t *Theme) SetScrollBarTrack(color graphics.Color) *Theme {
	t.ScrollBarTrack = color
	return t
}

func (t *Theme) SetScrollBarThumb(color graphics.Color) *Theme {
	t.ScrollBarThumb = color
	return t
}

func (t *Theme) SetChartGrid(color graphics.Color) *Theme {
	t.ChartGrid = color
	return t
}

func (t *Theme) SetChartAxis(color graphics.Color) *Theme {
	t.ChartAxis = color
	return t
}
