package platform

import (
	"sync"

	"github.com/sixnineone/mobilegles"
)

// GlobalClipboard is the toolkit-wide clipboard. Text goes to the OS
// clipboard when the platform layer is available (SDL3 on desktop); a
// memory mirror always backs it so copy/paste works inside the app and
// in headless tests regardless of platform support.
var GlobalClipboard = &Clipboard{}

// Clipboard stores text for cut/copy/paste. SetText mirrors to the OS
// clipboard; GetText prefers it and falls back to the in-memory mirror
// (e.g. headless tests, platforms without clipboard support).
type Clipboard struct {
	mu     sync.RWMutex
	memory string
}

// SetText places text on the clipboard.
func (c *Clipboard) SetText(text string) {
	c.mu.Lock()
	c.memory = text
	c.mu.Unlock()
	mobilegles.SetClipboardText(text)
}

// GetText returns the clipboard text: the OS clipboard when it has
// content, otherwise the in-memory mirror.
func (c *Clipboard) GetText() string {
	if s := mobilegles.GetClipboardText(); s != "" {
		return s
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.memory
}
