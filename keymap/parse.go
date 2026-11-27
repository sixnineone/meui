package keymap

import (
	"fmt"
	"strings"

	"github.com/sixnineone/mobilegles"
)

// keyNames maps spec key names to Keycodes. Values are the mobilegles
// Keycode constants (USB HID subset; arrows use the reference-compatible
// values 200/203/205/208).
var keyNames = map[string]mobilegles.Keycode{
	"enter":     mobilegles.KeyEnter,
	"return":    mobilegles.KeyEnter,
	"escape":    mobilegles.KeyEscape,
	"esc":       mobilegles.KeyEscape,
	"backspace": mobilegles.KeyBackspace,
	"tab":       mobilegles.KeyTab,
	"space":     mobilegles.KeySpace,
	"up":        mobilegles.KeyUp,
	"down":      mobilegles.KeyDown,
	"left":      mobilegles.KeyLeft,
	"right":     mobilegles.KeyRight,
	"home":      74,
	"end":       77,
	"delete":    76,
	"del":       76,
	"pageup":    75,
	"pagedown":  78,
	"insert":    73,
	"f1":        58, "f2": 59, "f3": 60, "f4": 61,
	"f5": 62, "f6": 63, "f7": 64, "f8": 65,
	"f9": 66, "f10": 67, "f11": 68, "f12": 69,
}

func init() {
	// Letters A..Z are HID 4..29; digits 1=30..0=39.
	for i := 0; i < 26; i++ {
		keyNames[string(rune('A'+i))] = mobilegles.Keycode(4 + i)
		keyNames[string(rune('a'+i))] = mobilegles.Keycode(4 + i)
	}
	keyNames["0"] = 39
	for i := 1; i <= 9; i++ {
		keyNames[string(rune('0'+i))] = mobilegles.Keycode(29 + i)
	}
}

// modNames maps spec modifier tokens to KeyMod flags.
var modNames = map[string]mobilegles.KeyMod{
	"ctrl":    mobilegles.ModCtrl,
	"control": mobilegles.ModCtrl,
	"shift":   mobilegles.ModShift,
	"alt":     mobilegles.ModAlt,
	"option":  mobilegles.ModAlt,
	"meta":    mobilegles.ModMeta,
	"cmd":     mobilegles.ModMeta,
	"super":   mobilegles.ModMeta,
	"win":     mobilegles.ModMeta,
}

// ParseSpec parses a spec string like "Ctrl+Shift+Escape" into a key
// plus modifier set. Tokens are case-insensitive; order is ignored.
func ParseSpec(spec string) (mobilegles.Keycode, mobilegles.KeyMod, error) {
	var key mobilegles.Keycode
	var mods mobilegles.KeyMod
	haveKey := false
	for _, tok := range strings.Split(spec, "+") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			return 0, 0, fmt.Errorf("empty token in spec %q", spec)
		}
		lower := strings.ToLower(tok)
		if mod, ok := modNames[lower]; ok {
			mods |= mod
			continue
		}
		if k, ok := keyNames[lower]; ok {
			if haveKey {
				return 0, 0, fmt.Errorf("two keys in spec %q", spec)
			}
			key, haveKey = k, true
			continue
		}
		// Canonical fallback for keys without a name ("Key99") — keeps
		// FormatSpec output round-trippable for every Keycode.
		if strings.HasPrefix(lower, "key") {
			var n int
			if _, err := fmt.Sscanf(lower, "key%d", &n); err == nil && n > 0 {
				if haveKey {
					return 0, 0, fmt.Errorf("two keys in spec %q", spec)
				}
				key, haveKey = mobilegles.Keycode(n), true
				continue
			}
		}
		return 0, 0, fmt.Errorf("unknown key or modifier %q in spec %q", tok, spec)
	}
	if !haveKey {
		return 0, 0, fmt.Errorf("no key in spec %q", spec)
	}
	return key, mods, nil
}

// FormatSpec renders key+mods back into canonical spec form
// ("Ctrl+Shift+Escape") — used for persistence and Settings display.
func FormatSpec(key mobilegles.Keycode, mods mobilegles.KeyMod) string {
	var b strings.Builder
	if mods&mobilegles.ModCtrl != 0 {
		b.WriteString("Ctrl+")
	}
	if mods&mobilegles.ModShift != 0 {
		b.WriteString("Shift+")
	}
	if mods&mobilegles.ModAlt != 0 {
		b.WriteString("Alt+")
	}
	if mods&mobilegles.ModMeta != 0 {
		b.WriteString("Meta+")
	}
	if name, ok := keyCodeNames[key]; ok {
		b.WriteString(name)
	} else {
		fmt.Fprintf(&b, "Key%d", key)
	}
	return b.String()
}

// keyCodeNames is the inverse of keyNames for canonical formatting
// (first name wins for aliases: Escape over Esc, Enter over Return).
var keyCodeNames = map[mobilegles.Keycode]string{
	mobilegles.KeyEnter:     "Enter",
	mobilegles.KeyEscape:    "Escape",
	mobilegles.KeyBackspace: "Backspace",
	mobilegles.KeyTab:       "Tab",
	mobilegles.KeySpace:     "Space",
	mobilegles.KeyUp:        "Up",
	mobilegles.KeyDown:      "Down",
	mobilegles.KeyLeft:      "Left",
	mobilegles.KeyRight:     "Right",
	74:                      "Home",
	77:                      "End",
	76:                      "Delete",
	75:                      "PageUp",
	78:                      "PageDown",
	73:                      "Insert",
	58:                      "F1", 59: "F2", 60: "F3", 61: "F4",
	62: "F5", 63: "F6", 64: "F7", 65: "F8",
	66: "F9", 67: "F10", 68: "F11", 69: "F12",
}

func init() {
	for i := 0; i < 26; i++ {
		keyCodeNames[mobilegles.Keycode(4+i)] = string(rune('A' + i))
	}
	keyCodeNames[39] = "0"
	for i := 1; i <= 9; i++ {
		keyCodeNames[mobilegles.Keycode(29+i)] = string(rune('0' + i))
	}
}
