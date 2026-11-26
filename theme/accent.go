package theme

import (
	"runtime"
	"strconv"
	"strings"

	"github.com/sixnineone/meui/graphics"
)

// System accent follow: probes each platform's accent preference and
// overlays it on the detected preset. Same fail-soft rules as the
// platform probes (no cgo, unknown → keep the preset's own accent).

// DetectAccent returns the system accent color when the platform
// exposes one.
func DetectAccent() (graphics.Color, bool) {
	switch runtime.GOOS {
	case "darwin":
		return macOSAccent()
	case "windows":
		return windowsAccent()
	default:
		return linuxAccent()
	}
}

// ApplyAccent overrides the accent-derived tokens on t in place. The
// platform's translucency convention for selection is preserved:
// translucent selections keep their alpha, opaque ones become the new
// accent. Call with the theme already loaded, then re-Apply(t) to
// regenerate the widget themes.
func ApplyAccent(t *Theme, c graphics.Color) {
	if t == nil || c&0x00FFFFFF == 0 {
		return
	}
	t.Accent = c
	t.Primary = c
	t.BorderFocused = c
	t.BorderHovered = lighten(c, 0x202020)
	t.Link = c
	if t.Selected != 0 {
		alpha := t.Selected & 0xFF000000
		if alpha == 0xFF000000 {
			t.Selected = c
		} else {
			t.Selected = alpha | (c & 0x00FFFFFF)
		}
	}
}

// ── Platform probes ────────────────────────────────────────────

// macOSAccent maps AppleAccentColor (or AppleHighlightColor) to a
// color. Apple's index: -1 graphite, 0 red, 1 orange, 2 yellow,
// 3 green, 4 blue, 5 purple, 6 pink.
func macOSAccent() (graphics.Color, bool) {
	if out := runProbe("defaults", "read", "-g", "AppleAccentColor"); out != "" {
		switch strings.TrimSpace(out) {
		case "-1":
			return 0xFF8E8E93, true
		case "0":
			return 0xFFFF5257, true
		case "1":
			return 0xFFFF9F0A, true
		case "2":
			return 0xFFFFD60A, true
		case "3":
			return 0xFF30D158, true
		case "4":
			return 0xFF0A84FF, true
		case "5":
			return 0xFFBF5AF2, true
		case "6":
			return 0xFFFF375F, true
		}
	}
	// AppleHighlightColor: "R G B Name" floats in 0..1.
	if out := runProbe("defaults", "read", "-g", "AppleHighlightColor"); out != "" {
		fields := strings.Fields(out)
		if len(fields) >= 3 {
			r, e1 := strconv.ParseFloat(fields[0], 64)
			g, e2 := strconv.ParseFloat(fields[1], 64)
			b, e3 := strconv.ParseFloat(fields[2], 64)
			if e1 == nil && e2 == nil && e3 == nil {
				return 0xFF000000 |
					graphics.Color(clamp8(r*255))<<16 |
					graphics.Color(clamp8(g*255))<<8 |
					graphics.Color(clamp8(b*255)), true
			}
		}
	}
	return 0, false
}

// windowsAccent reads DWM ColorizationColor (0xAARRGGBB).
func windowsAccent() (graphics.Color, bool) {
	out := runProbe("reg", "query", `HKCU\Software\Microsoft\Windows\DWM`, "/v", "ColorizationColor")
	if out == "" {
		return 0, false
	}
	fields := strings.Fields(out)
	for i, f := range fields {
		if strings.EqualFold(f, "ColorizationColor") && i+2 < len(fields) {
			return parseHexColor(fields[i+2])
		}
	}
	return 0, false
}

// linuxAccent probes KDE's kdeglobals and GNOME's accent-color key.
func linuxAccent() (graphics.Color, bool) {
	for _, tool := range []string{"kreadconfig6", "kreadconfig5"} {
		if out := runProbe(tool, "--file", "kdeglobals", "--group", "General", "--key", "AccentColor"); out != "" {
			if c, ok := parseKDEColor(out); ok {
				return c, true
			}
		}
	}
	if out := runProbe("gsettings", "get", "org.gnome.desktop.interface", "accent-color"); out != "" {
		if c, ok := gnomeNamedAccent(out); ok {
			return c, true
		}
	}
	return 0, false
}

// ── Parsers (exported to tests) ────────────────────────────────

// parseKDEColor parses KDE's "R,G,B" decimal format.
func parseKDEColor(s string) (graphics.Color, bool) {
	parts := strings.Split(strings.TrimSpace(s), ",")
	if len(parts) != 3 {
		return 0, false
	}
	r, e1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	g, e2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	b, e3 := strconv.Atoi(strings.TrimSpace(parts[2]))
	if e1 != nil || e2 != nil || e3 != nil {
		return 0, false
	}
	if r < 0 || r > 255 || g < 0 || g > 255 || b < 0 || b > 255 {
		return 0, false
	}
	return 0xFF000000 | graphics.Color(r)<<16 | graphics.Color(g)<<8 | graphics.Color(b), true
}

// ParseHexAccent parses "#RRGGBB" / "0xRRGGBB" / "RRGGBB" — the
// MEUI_ACCENT override format.
func ParseHexAccent(s string) (graphics.Color, bool) {
	return parseHexColor(s)
}

// parseHexColor accepts "0xRRGGBB", "0xAARRGGBB" and "#RRGGBB".
func parseHexColor(s string) (graphics.Color, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	s = strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, false
	}
	c := graphics.Color(v)
	if len(s) <= 6 {
		c |= 0xFF000000
	}
	if c&0x00FFFFFF == 0 {
		return 0, false
	}
	return c, true
}

// gnomeNamedAccent maps GNOME's accent-color names (GNOME 47+).
func gnomeNamedAccent(s string) (graphics.Color, bool) {
	switch strings.ToLower(strings.Trim(strings.TrimSpace(s), "'\"")) {
	case "blue":
		return 0xFF3584E4, true
	case "teal":
		return 0xFF2190A4, true
	case "green":
		return 0xFF3A944A, true
	case "yellow":
		return 0xFFC88800, true
	case "orange":
		return 0xFFED5B00, true
	case "red":
		return 0xFFE62D42, true
	case "pink":
		return 0xFFD56199, true
	case "purple":
		return 0xFF9141AC, true
	case "slate":
		return 0xFF6F8396, true
	}
	return 0, false
}

func clamp8(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}
