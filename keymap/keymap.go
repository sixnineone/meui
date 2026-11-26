// Package keymap provides the global, user-mappable key-binding
// dictionary: stable action names ("window.close") mapped to key +
// modifier combinations, with JSON persistence so applications (and a
// future Settings UI) can override the defaults at startup.
//
// Spec string format — modifier + key names joined by '+',
// order-insensitive, case-insensitive:
//
//	"Escape"                  plain key
//	"Ctrl+W"                  modifier + key
//	"Ctrl+Shift+F"            multiple modifiers
//
// Modifier names: Ctrl/Control, Shift, Alt/Option, Meta/Cmd/Super/Win.
// Key names cover the mobilegles Keycode set (see keyNames).
package keymap

import (
	"fmt"
	"sync"

	"github.com/sixnineone/mobilegles"
)

// Binding is one action bound to one key + modifier combination.
type Binding struct {
	Action string
	Key    mobilegles.Keycode
	Mods   mobilegles.KeyMod
}

// Map is an ordered set of bindings; the first binding matching a key
// event wins. Actions may share a key — MatchAll returns every match
// in order so handlers can skip contextual actions (exit-fullscreen
// only fires while fullscreen, then the next match, window.close, applies).
type Map struct {
	mu       sync.RWMutex
	bindings []Binding
}

// New returns an empty Map.
func New() *Map { return &Map{} }

// Bind parses spec ("Ctrl+Shift+Escape") and binds it to action,
// replacing any existing binding for the same action.
func (m *Map) Bind(action, spec string) error {
	key, mods, err := ParseSpec(spec)
	if err != nil {
		return fmt.Errorf("keymap: bind %q: %w", action, err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.bindings {
		if m.bindings[i].Action == action {
			m.bindings[i].Key = key
			m.bindings[i].Mods = mods
			return nil
		}
	}
	m.bindings = append(m.bindings, Binding{Action: action, Key: key, Mods: mods})
	return nil
}

// MustBind is Bind for seed tables; a bad spec is a programming error.
func (m *Map) MustBind(action, spec string) *Map {
	if err := m.Bind(action, spec); err != nil {
		panic(err)
	}
	return m
}

// Unbind removes the binding for action. No-op when absent.
func (m *Map) Unbind(action string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.bindings {
		if m.bindings[i].Action == action {
			m.bindings = append(m.bindings[:i], m.bindings[i+1:]...)
			return
		}
	}
}

// Match returns the action of the FIRST binding matching key+mods.
func (m *Map) Match(key mobilegles.Keycode, mods mobilegles.KeyMod) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, b := range m.bindings {
		if b.Key == key && b.Mods == mods {
			return b.Action, true
		}
	}
	return "", false
}

// MatchAll returns every matching action in binding order — handlers
// walk it when actions are contextual (exit-fullscreen fires only
// while fullscreen, then window.close applies to the same key).
func (m *Map) MatchAll(key mobilegles.Keycode, mods mobilegles.KeyMod) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []string
	for _, b := range m.bindings {
		if b.Key == key && b.Mods == mods {
			out = append(out, b.Action)
		}
	}
	return out
}

// ActionSpec returns the spec string currently bound to action
// ("Ctrl+W") — for Settings UI display.
func (m *Map) ActionSpec(action string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, b := range m.bindings {
		if b.Action == action {
			return FormatSpec(b.Key, b.Mods), true
		}
	}
	return "", false
}

// All snapshots the bindings.
func (m *Map) All() []Binding {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Binding, len(m.bindings))
	copy(out, m.bindings)
	return out
}

// Seed action names. Applications register additional actions with
// Default().MustBind / Bind.
const (
	ActionExitFullscreen = "app.exit-fullscreen"
	ActionCloseWindow    = "window.close"
	ActionCloseTab       = "window.close-tab"
	ActionFocusSearch    = "view.focus-search"
)

// Defaults returns a Map with the framework seed bindings. Esc is
// bound to both exit-fullscreen and window.close: handlers try actions
// in order (MatchAll) and skip contextual ones that don't apply, so
// plain Esc closes a window while Esc in fullscreen only leaves
// fullscreen — matching the legacy hardcoded behavior.
func Defaults() *Map {
	return New().
		MustBind(ActionExitFullscreen, "Escape").
		MustBind(ActionCloseWindow, "Escape").
		MustBind(ActionCloseTab, "Ctrl+W").
		MustBind(ActionFocusSearch, "Ctrl+F")
}

var (
	defaultOnce sync.Once
	defaultMap  *Map
)

// Default returns the process-wide key map: the seed bindings with any
// overrides from the keymap file applied (MEUI_KEYMAP=<path>, or
// ./keymap.json when present).
func Default() *Map {
	defaultOnce.Do(func() {
		defaultMap = Defaults()
		LoadOverrides(defaultMap)
	})
	return defaultMap
}
