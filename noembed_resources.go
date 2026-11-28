//go:build no_embed

package meui

// With -tags no_embed the toolkit ships no built-in resources; meui:// is empty
// until the application mounts a directory there.
func init() {}
