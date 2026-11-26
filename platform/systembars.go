package platform

// BarAppearance is what the system-bar ICONS should be, which is a property of what is
// behind them and NOT of the app's own theme: an app drawing a white surface behind the
// status bar needs dark icons even while its own theme is dark.
type BarAppearance uint8

const (
	// BarAppearanceAuto follows the OS's light/dark preference, which is the right
	// default: it matches every other app on the device.
	BarAppearanceAuto BarAppearance = iota
	// BarAppearanceLight means DARK icons, for a light background.
	BarAppearanceLight
	// BarAppearanceDark means LIGHT icons, for a dark background.
	BarAppearanceDark
)

func (b BarAppearance) String() string {
	switch b {
	case BarAppearanceLight:
		return "light"
	case BarAppearanceDark:
		return "dark"
	default:
		return "auto"
	}
}

// SystemBars is the system bars: drawing behind them, and how their content is drawn.
//
// Two things about it are worth stating because they are the opposite of how most
// capabilities behave:
//
//   - Edge-to-edge is not a preference. Android 15 ENFORCES it, and below 15 asking for it
//     is what stops the app being letterboxed between two opaque strips. An app should
//     turn it on and then keep its content out of the insets, which the toolkit already
//     does from Window.Insets().
//   - The appearance is about the ICONS, not the bars. A bar itself is transparent in both
//     cases; what changes is whether the clock and the gesture handle are drawn light or
//     dark, so the choice depends on the app's background at that edge.
type SystemBars interface {
	// SetEdgeToEdge asks the platform to let the app draw behind the bars. It reports
	// whether the request was accepted, not whether it took effect — the platform applies
	// it asynchronously, on its own thread.
	SetEdgeToEdge(on bool) bool
	// SetAppearance sets the system-bar icon appearance.
	SetAppearance(BarAppearance) bool
}
