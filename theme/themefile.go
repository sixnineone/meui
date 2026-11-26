package theme

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/sixnineone/meui/graphics"
)

// Native theme format: JSON with single-base inheritance from a
// registered preset. Only the deltas need listing — a brand theme is a
// handful of palette entries plus any metric/motion/chrome tweaks.
//
//	{
//	  "id": "brand-dark",
//	  "base": "breeze-dark",
//	  "palette": { "accent": "#E62D42", "surface": "#2A2A2A" },
//	  "metrics": { "controlHeight": 30, "shape": { "button": 8 } },
//	  "motion":  { "hoverMs": 120 },
//	  "chrome":  { "style": "windows", "height": 32 },
//	  "fonts":   { "families": ["Inter", "Roboto"], "bodySize": 13 }
//	}

// presetMu guards the preset registry against RegisterPreset writers.
var presetMu sync.RWMutex

// RegisterPreset adds a builder under id. Fails on duplicate IDs.
func RegisterPreset(id string, build func() *Theme) error {
	if id == "" || build == nil {
		return fmt.Errorf("theme: RegisterPreset needs an id and builder")
	}
	presetMu.Lock()
	defer presetMu.Unlock()
	if _, exists := presetBuilders[id]; exists {
		return fmt.Errorf("theme: preset %q already registered", id)
	}
	presetBuilders[id] = build
	return nil
}

type themeFileMetrics struct {
	ControlHeight     *float32        `json:"controlHeight"`
	ControlPadX       *float32        `json:"controlPadX"`
	ControlPadY       *float32        `json:"controlPadY"`
	InputPadX         *float32        `json:"inputPadX"`
	MenuItemHeight    *float32        `json:"menuItemHeight"`
	ListRowHeight     *float32        `json:"listRowHeight"`
	TabHeight         *float32        `json:"tabHeight"`
	ToolbarHeight     *float32        `json:"toolbarHeight"`
	StatusBarHeight   *float32        `json:"statusBarHeight"`
	ScrollbarWidth    *float32        `json:"scrollbarWidth"`
	ScrollbarOverlay  *bool           `json:"scrollbarOverlay"`
	ScrollbarAutoHide *bool           `json:"scrollbarAutoHide"`
	BorderWidth       *float32        `json:"borderWidth"`
	FocusRingWidth    *float32        `json:"focusRingWidth"`
	FocusRingOffset   *float32        `json:"focusRingOffset"`
	BoxSize           *float32        `json:"boxSize"`
	BoxSpacing        *float32        `json:"boxSpacing"`
	SliderThumb       *float32        `json:"sliderThumb"`
	ArrowSize         *float32        `json:"arrowSize"`
	GripSize          *float32        `json:"gripSize"`
	Shape             *themeFileShape `json:"shape"`
}

type themeFileShape struct {
	Window    *float32 `json:"window"`
	Popup     *float32 `json:"popup"`
	Menu      *float32 `json:"menu"`
	Tooltip   *float32 `json:"tooltip"`
	Button    *float32 `json:"button"`
	CheckBox  *float32 `json:"checkBox"`
	Input     *float32 `json:"input"`
	ComboBox  *float32 `json:"comboBox"`
	Tab       *float32 `json:"tab"`
	GroupBox  *float32 `json:"groupBox"`
	ListRow   *float32 `json:"listRow"`
	ScrollBar *float32 `json:"scrollBar"`
}

type themeFileMotion struct {
	HoverMS       *float32 `json:"hoverMs"`
	PressMS       *float32 `json:"pressMs"`
	FocusMS       *float32 `json:"focusMs"`
	ToggleMS      *float32 `json:"toggleMs"`
	MenuOpenMS    *float32 `json:"menuOpenMs"`
	MenuCloseMS   *float32 `json:"menuCloseMs"`
	TooltipShowMS *float32 `json:"tooltipShowMs"`
	TooltipHideMS *float32 `json:"tooltipHideMs"`
	ScrollFadeMS  *float32 `json:"scrollFadeMs"`
}

type themeFileChrome struct {
	Style        *string  `json:"style"`
	Height       *float32 `json:"height"`
	ButtonW      *float32 `json:"buttonW"`
	ButtonH      *float32 `json:"buttonH"`
	ButtonGap    *float32 `json:"buttonGap"`
	HoverBg      *string  `json:"hoverBg"`
	PressedBg    *string  `json:"pressedBg"`
	CloseHoverBg *string  `json:"closeHoverBg"`
	GlyphColor   *string  `json:"glyphColor"`
	TitleAlign   *string  `json:"titleAlign"`
	TitleWeight  *int     `json:"titleWeight"`
}

type themeFileFonts struct {
	Families []string `json:"families"`
	BodySize *float32 `json:"bodySize"`
}

// themeFile is the on-disk schema.
type themeFile struct {
	ID      string            `json:"id"`
	Base    string            `json:"base"`
	Palette map[string]string `json:"palette"`
	Metrics *themeFileMetrics `json:"metrics"`
	Motion  *themeFileMotion  `json:"motion"`
	Chrome  *themeFileChrome  `json:"chrome"`
	Fonts   *themeFileFonts   `json:"fonts"`
	// Tokens is the complete, machine-written dump of every non-colour
	// theme field and token group (see ToThemeFile). The hand-written
	// sections above stay for friendly, hand-authored delta files; when
	// both are present the tokens win, so an exported theme reproduces
	// exactly.
	Tokens map[string]any `json:"tokens,omitempty"`
}

// LoadThemeFile parses a native theme file, registers it under its ID,
// and returns the ID (loadable via LoadPreset, usable in MEUI_THEME).
func LoadThemeFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("theme file: %w", err)
	}
	id, err := LoadThemeFileData(data)
	if err != nil {
		return "", fmt.Errorf("theme file %s: %w", path, err)
	}
	return id, nil
}

// LoadThemeFileData parses a native theme file from memory, registers it
// under its ID, and returns the ID. Editors and tests use this to avoid a
// round-trip through the filesystem.
func LoadThemeFileData(data []byte) (string, error) {
	var tf themeFile
	if err := json.Unmarshal(data, &tf); err != nil {
		return "", fmt.Errorf("theme file: %w", err)
	}
	return registerThemeFile(&tf)
}

// registerThemeFile registers an already-parsed theme file (tests,
// programmatic themes).
func registerThemeFile(tf *themeFile) (string, error) {
	if tf == nil || tf.ID == "" {
		return "", fmt.Errorf("theme file: missing \"id\"")
	}
	base := tf.Base
	if base == "" {
		base = "meui-dark"
	}
	presetMu.RLock()
	build, ok := presetBuilders[base]
	presetMu.RUnlock()
	if !ok {
		return "", fmt.Errorf("theme file %q: unknown base %q (want one of %v)", tf.ID, base, PresetIDs())
	}
	snapshot := *tf // closure captures the parsed values
	err := RegisterPreset(tf.ID, func() *Theme {
		t := build()
		t.Name = snapshot.ID
		t.Meta.ID = snapshot.ID
		applyThemeFile(t, &snapshot)
		return t
	})
	if err != nil {
		return "", err
	}
	return tf.ID, nil
}

// applyThemeFile overlays the file's deltas onto a base theme.
func applyThemeFile(t *Theme, tf *themeFile) {
	t.withStyleDefaults() // materialize token groups before overriding
	// Palette. Explicit fields always win; accent also derives the
	// fields the presets normally derive, unless the file set them.
	var accent graphics.Color
	if tf.Palette != nil {
		if c, ok := parsePaletteColor(tf.Palette, "accent"); ok {
			accent = c
			t.Accent = c
			t.Primary = c
		}
		// Every top-level colour role is addressable, so an exported
		// theme round-trips without leaning on the base preset. Applied
		// after "accent" so an explicit "primary" wins.
		for _, target := range paletteColorTargets(t) {
			if c, ok := parsePaletteColor(tf.Palette, target.key); ok {
				*target.dst = c
			}
		}
		// Legacy alias: "raised" is the older name for surfaceVariant.
		if c, ok := parsePaletteColor(tf.Palette, "raised"); ok {
			t.SurfaceVariant = c
		}
		if accent != 0 {
			if _, explicit := tf.Palette["borderFocused"]; !explicit {
				t.BorderFocused = accent
			}
			if _, explicit := tf.Palette["borderHovered"]; !explicit {
				t.BorderHovered = lighten(accent, 0x202020)
			}
			if _, explicit := tf.Palette["link"]; !explicit {
				t.Link = accent
			}
			if _, explicit := tf.Palette["selected"]; !explicit && t.Selected != 0 {
				alpha := t.Selected & 0xFF000000
				if alpha == 0xFF000000 {
					t.Selected = accent
				} else {
					t.Selected = alpha | (accent & 0x00FFFFFF)
				}
			}
		}
	}

	if m := tf.Metrics; m != nil {
		set := func(dst *float32, v *float32) {
			if v != nil {
				*dst = *v
			}
		}
		set(&t.Metrics.ControlHeight, m.ControlHeight)
		set(&t.Metrics.ControlPadX, m.ControlPadX)
		set(&t.Metrics.ControlPadY, m.ControlPadY)
		set(&t.Metrics.InputPadX, m.InputPadX)
		set(&t.Metrics.MenuItemHeight, m.MenuItemHeight)
		set(&t.Metrics.ListRowHeight, m.ListRowHeight)
		set(&t.Metrics.TabHeight, m.TabHeight)
		set(&t.Metrics.ToolbarHeight, m.ToolbarHeight)
		set(&t.Metrics.StatusBarHeight, m.StatusBarHeight)
		set(&t.Metrics.ScrollbarWidth, m.ScrollbarWidth)
		set(&t.Metrics.BorderWidth, m.BorderWidth)
		set(&t.Metrics.FocusRingWidth, m.FocusRingWidth)
		set(&t.Metrics.FocusRingOffset, m.FocusRingOffset)
		set(&t.Metrics.BoxSize, m.BoxSize)
		set(&t.Metrics.BoxSpacing, m.BoxSpacing)
		set(&t.Metrics.SliderThumb, m.SliderThumb)
		set(&t.Metrics.ArrowSize, m.ArrowSize)
		set(&t.Metrics.GripSize, m.GripSize)
		if m.ScrollbarOverlay != nil {
			t.Metrics.ScrollbarOverlay = *m.ScrollbarOverlay
		}
		if m.ScrollbarAutoHide != nil {
			t.Metrics.ScrollbarAutoHide = *m.ScrollbarAutoHide
		}
		if m.Shape != nil {
			s := t.Metrics.Shape
			if s == nil {
				s = &ShapeTheme{}
				t.Metrics.Shape = s
			}
			set(&s.Window, m.Shape.Window)
			set(&s.Popup, m.Shape.Popup)
			set(&s.Menu, m.Shape.Menu)
			set(&s.Tooltip, m.Shape.Tooltip)
			set(&s.Button, m.Shape.Button)
			set(&s.CheckBox, m.Shape.CheckBox)
			set(&s.Input, m.Shape.Input)
			set(&s.ComboBox, m.Shape.ComboBox)
			set(&s.Tab, m.Shape.Tab)
			set(&s.GroupBox, m.Shape.GroupBox)
			set(&s.ListRow, m.Shape.ListRow)
			set(&s.ScrollBar, m.Shape.ScrollBar)
		}
		t.ScrollbarWidth = t.Metrics.ScrollbarWidth
	}

	if mo := tf.Motion; mo != nil {
		set := func(dst *Timing, v *float32) {
			if v != nil {
				dst.DurationMS = *v
			}
		}
		set(&t.Motion.Hover, mo.HoverMS)
		set(&t.Motion.Press, mo.PressMS)
		set(&t.Motion.Focus, mo.FocusMS)
		set(&t.Motion.Toggle, mo.ToggleMS)
		set(&t.Motion.MenuOpen, mo.MenuOpenMS)
		set(&t.Motion.MenuClose, mo.MenuCloseMS)
		set(&t.Motion.TooltipShow, mo.TooltipShowMS)
		set(&t.Motion.TooltipHide, mo.TooltipHideMS)
		set(&t.Motion.ScrollFade, mo.ScrollFadeMS)
	}

	if ch := tf.Chrome; ch != nil {
		if ch.Style != nil {
			t.Chrome.Style = ChromeStyle(*ch.Style)
		}
		set := func(dst *float32, v *float32) {
			if v != nil {
				*dst = *v
			}
		}
		set(&t.Chrome.Height, ch.Height)
		set(&t.Chrome.ButtonW, ch.ButtonW)
		set(&t.Chrome.ButtonH, ch.ButtonH)
		set(&t.Chrome.ButtonGap, ch.ButtonGap)
		if ch.HoverBg != nil {
			if c, ok := parseThemeColor(*ch.HoverBg); ok {
				t.Chrome.HoverBg = c
			}
		}
		if ch.PressedBg != nil {
			if c, ok := parseThemeColor(*ch.PressedBg); ok {
				t.Chrome.PressedBg = c
			}
		}
		if ch.CloseHoverBg != nil {
			if c, ok := parseThemeColor(*ch.CloseHoverBg); ok {
				t.Chrome.CloseHoverBg = c
			}
		}
		if ch.GlyphColor != nil {
			if c, ok := parseThemeColor(*ch.GlyphColor); ok {
				t.Chrome.GlyphColor = c
			}
		}
		if ch.TitleAlign != nil {
			switch *ch.TitleAlign {
			case "center", "Centre":
				t.Chrome.TitleAlign = TitleCenter
			case "left", "start":
				t.Chrome.TitleAlign = TitleLeft
			}
		}
		if ch.TitleWeight != nil {
			t.Chrome.TitleWeight = *ch.TitleWeight
		}
	}

	if f := tf.Fonts; f != nil {
		if len(f.Families) > 0 {
			t.Typography.Families = append([]string(nil), f.Families...)
		}
		if f.BodySize != nil {
			t.Typography.Body.Size = *f.BodySize
			t.Typography.Label.Size = *f.BodySize
			t.Typography.Button.Size = *f.BodySize
			t.FontSizeMedium = *f.BodySize
		}
	}

	// Complete token dump (ToThemeFile). Applied last so an exported
	// theme reproduces exactly; absent for hand-authored delta files.
	applyTokenMap(t, tf.Tokens)
}

// parsePaletteColor resolves one palette entry; missing/invalid → false.
// Uses the lenient parser, so black and fully transparent are valid
// palette colours.
func parsePaletteColor(p map[string]string, key string) (graphics.Color, bool) {
	s, ok := p[key]
	if !ok {
		return 0, false
	}
	return parseThemeColor(s)
}
