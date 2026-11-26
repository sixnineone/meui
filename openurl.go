package meui

import "github.com/sixnineone/meui/platform"

// OpenURL launches the platform browser for the given URL. Used as the default
// action for clicked links in Markdown and rich-text labels when the
// application does not install its own OnLinkClicked handler.
//
// It delegates to the platform seam, so a backend with a real URL opener (the
// SDL3 adapter, which also covers Android and iOS) is used where there is one;
// otherwise the seam's fallback shells out. The error is dropped because this
// is a fire-and-forget default action — an app that cares installs its own
// handler.
func OpenURL(url string) {
	_ = platform.OpenURL(url)
}
