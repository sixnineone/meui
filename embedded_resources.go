//go:build !no_embed

package meui

import (
	"embed"

	"github.com/sixnineone/meui/resources"
)

// The toolkit's own defaults travel with the binary and are mounted at
// meui://. A build with -tags no_embed leaves that namespace empty and expects
// the app to mount what it needs.
//
//go:embed themes
var embeddedDefaults embed.FS

func init() {
	resources.RegisterBuiltin(embeddedDefaults)
}
