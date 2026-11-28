package meui

import (
	_ "embed"
	"sync"

	"github.com/sixnineone/meui/graphics"
)

// The default window icon: the purple MeUI mark. It is embedded so every
// toolkit window has an icon without the app doing anything, which is what
// keeps the taskbar from falling back to the desktop's generic "X" icon.
//
//go:embed assets/icons/app.png
var defaultIconPNG []byte

var (
	defaultIconOnce sync.Once
	defaultIcon     *graphics.Image
	defaultIconOK   bool
)

// DefaultIcon returns the toolkit's fallback window icon, or nil if it cannot be
// decoded (an embedded asset, so nil means a broken build rather than a bad
// file). WindowOptions.Icon overrides it; WindowOptions.NoIcon suppresses it.
func DefaultIcon() *graphics.Image {
	defaultIconOnce.Do(func() {
		img, err := graphics.NewImageFromBytes(defaultIconPNG)
		if err == nil && img != nil {
			defaultIcon, defaultIconOK = img, true
		}
	})
	if !defaultIconOK {
		return nil
	}
	return defaultIcon
}
