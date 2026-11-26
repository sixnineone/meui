package keymap

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// fileFormat is the on-disk shape: action -> spec string.
//
//	{
//	  "window.close": "Ctrl+Shift+Escape",
//	  "view.focus-search": "Ctrl+K"
//	}
type fileFormat map[string]string

// Save writes the map's bindings to path (JSON, stable ordering).
func (m *Map) Save(path string) error {
	m.mu.RLock()
	out := make(fileFormat, len(m.bindings))
	for _, b := range m.bindings {
		out[b.Action] = FormatSpec(b.Key, b.Mods)
	}
	m.mu.RUnlock()

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// Load reads path and applies every entry onto m as an override
// (rebinding the action's key). Unknown actions are still bound — the
// file may define bindings for application-specific actions. Returns
// the number of bindings applied.
func (m *Map) Load(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var in fileFormat
	if err := json.Unmarshal(data, &in); err != nil {
		return 0, fmt.Errorf("keymap: %s: %w", path, err)
	}
	applied := 0
	for action, spec := range in {
		if err := m.Bind(action, spec); err != nil {
			log.Printf("keymap: skipping %q: %v", action, err)
			continue
		}
		applied++
	}
	return applied, nil
}

// overridePaths lists where LoadOverrides looks for a keymap file.
func overridePaths() []string {
	var paths []string
	if p := os.Getenv("MEUI_KEYMAP"); p != "" {
		paths = append(paths, p)
	}
	paths = append(paths, "keymap.json")
	return paths
}

// LoadOverrides applies the first keymap file found (MEUI_KEYMAP=<path>
// or ./keymap.json) onto m. Missing files are not an error — the seed
// defaults simply stand. Load errors (bad JSON, bad specs) are logged,
// not fatal.
func LoadOverrides(m *Map) {
	for _, path := range overridePaths() {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		n, err := m.Load(path)
		if err != nil {
			log.Printf("keymap: ignoring %s: %v", path, err)
			return
		}
		log.Printf("keymap: %d override(s) from %s", n, path)
		return
	}
}
