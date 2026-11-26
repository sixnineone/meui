package graphics

import (
	"fmt"
	"strconv"
	"strings"
)

// ColorDecode parses CSS-style color strings: hex (#RGB, #RRGGBB),
// rgb(r,g,b), rgba(r,g,b,a), or named colors. This is a partial
// implementation of unison's ColorDecode.
func ColorDecode(s string) (Color, error) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return 0, fmt.Errorf("empty color")
	}
	if s[0] == '#' {
		hex := s[1:]
		if len(hex) == 3 {
			r, _ := strconv.ParseUint(string(hex[0])+string(hex[0]), 16, 8)
			g, _ := strconv.ParseUint(string(hex[1])+string(hex[1]), 16, 8)
			b, _ := strconv.ParseUint(string(hex[2])+string(hex[2]), 16, 8)
			return RGBA(uint8(r), uint8(g), uint8(b), 255), nil
		}
		if len(hex) == 6 {
			v, err := strconv.ParseUint(hex, 16, 32)
			if err != nil {
				return 0, err
			}
			return Color(v | 0xFF000000), nil
		}
		if len(hex) == 8 {
			v, err := strconv.ParseUint(hex, 16, 64)
			if err != nil {
				return 0, err
			}
			return Color((v >> 24) | ((v & 0xFF) << 16) | ((v >> 8) & 0xFF00) | ((v >> 16) & 0xFF0000)), nil
		}
		return 0, fmt.Errorf("invalid hex color: %q", s)
	}
	lower := strings.ToLower(s)
	if c, ok := namedColors[lower]; ok {
		return c, nil
	}
	return 0, fmt.Errorf("unknown color: %q", s)
}

var namedColors = map[string]Color{
	"black":   0xFF000000,
	"white":   0xFFFFFFFF,
	"red":     0xFFFF0000,
	"green":   0xFF008000,
	"blue":    0xFF0000FF,
	"yellow":  0xFFFFFF00,
	"cyan":    0xFF00FFFF,
	"magenta": 0xFFFF00FF,
	"gray":    0xFF808080,
	"grey":    0xFF808080,
	"silver":  0xFFC0C0C0,
	"maroon":  0xFF800000,
	"olive":   0xFF808000,
	"lime":    0xFF00FF00,
	"aqua":    0xFF00FFFF,
	"teal":    0xFF008080,
	"navy":    0xFF000080,
	"fuchsia": 0xFFFF00FF,
	"purple":  0xFF800080,
	"orange":  0xFFFFA500,
}
