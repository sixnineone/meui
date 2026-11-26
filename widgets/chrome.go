package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/platform"
)

func init() {
	// The standard chrome. Registering it here — rather than asking every app to
	// install a frame — is what makes a toolkit window movable, maximizable and
	// resizable out of the box: the desktop backends create undecorated windows,
	// so a missing title bar is not a cosmetic omission, it is a window that
	// cannot be resized. Any app that imports widgets (every app does) gets it.
	meui.SetDefaultChrome(func(root *meui.Window, w platform.Window) platform.TitleBar {
		return InstallChrome(root, w, w.Title())
	})
}

// ChromeTarget is the window operations the chrome drives. It is deliberately
// narrow — the chrome needs a title and three verbs — so that a window type, a
// test double, or a preview host can satisfy it without implementing the whole
// platform seam.
type ChromeTarget interface {
	Title() string
	Close()
	Minimize()
	ToggleMaximize()
}

// InstallChrome builds the standard window frame for a window: the title bar,
// its minimize/maximize/close buttons, drag-to-move, and double-click to
// maximize.
//
// It registers all four handlers deliberately. The frame's BarDoubleClick does
// nothing when no OnMaximize handler is set, so an app that wires only close
// gets a title bar whose maximize button and double-click are both silently
// inert — the reason this belongs in one place instead of in each app.
//
// A caller installing this by hand must also shorten the content rect by
// BarHeight and move the root's content origin down by it (AddWindow does that
// for the default installation).
func InstallChrome(root *meui.Window, w ChromeTarget, title string) *WindowFrame {
	frame := NewWindowFrame()
	if root != nil {
		frame.SetOwnerRoot(root)
	}
	frame.SetTitle(title)
	// A hairline border: the backend's resize zone is the 6 px strip inside the
	// window edge, so the edge has to be somewhere the eye can find. Hovering it
	// shows the matching resize cursor, the other half of discoverability.
	frame.SetBorderWidth(1)
	frame.OnClose(func() { w.Close() })
	frame.OnMinimize(func() { w.Minimize() })
	frame.OnMaximize(func() { w.ToggleMaximize() })
	return frame
}
