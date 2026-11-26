package platform

import (
	"time"

	"github.com/sixnineone/meui/graphics"
)

// Options configures a window at creation.
type Options struct {
	Title  string
	Width  int32
	Height int32
}

// Insets are the safe-area / keyboard insets around a window's content.
// Zero on desktop; mobile backends fill them in.
type Insets struct {
	Top      float32
	Right    float32
	Bottom   float32
	Left     float32
	Keyboard float32
}

// LifecycleKind enumerates app lifecycle transitions. Desktop delivers none
// of these today; mobile backends deliver them so the app can park live work
// (TradeUI's tick streams).
type LifecycleKind int

const (
	LifecycleStart LifecycleKind = iota
	LifecycleResume
	LifecyclePause
	LifecycleStop
	LifecycleDestroy
	LifecycleLowMemory
	LifecycleBack
)

// LifecycleEvent is one lifecycle transition.
type LifecycleEvent struct {
	Kind LifecycleKind
}

// CursorSource reports the cursor a window should show for a content point.
// The toolkit's window implements it; the backend queries it on pointer moves.
type CursorSource interface {
	CursorAt(x, y float32) *Cursor
}

// Dialogs supplies native file dialogs. Backends without native integration
// return the stubs, whose RunModal always reports false.
type Dialogs interface {
	OpenDialog() *OpenDialog
	SaveDialog() *SaveDialog
}

// Window is the OS window contract the toolkit drives. Every method is
// expressed in seam types only, so a backend can implement it without the
// toolkit knowing which one is in use.
type Window interface {
	// SetTextInput turns the OS text input on or off for this window and says
	// where the text is being edited, in window coordinates.
	//
	// It must be driven by FOCUS CHANGES, not called once at startup. On mobile
	// the on-screen keyboard is raised by the OS only while text input is active,
	// so leaving it on permanently means a keyboard that never goes away, and
	// turning it on only at creation means one that never appears when a field is
	// actually focused. On Android the area is also load-bearing rather than
	// advisory: SDL sizes a hidden edit view from it, and with a zero rect the
	// InputMethodManager refuses to serve the IME at all.
	SetTextInput(active bool, area graphics.Rect)

	// Identity and geometry
	SetTitle(title string)
	Title() string
	SetContentRect(rect graphics.Rect)
	ContentRect() graphics.Rect
	Width() int32
	Height() int32
	Scale() float32
	Insets() Insets
	// FormFactor is how this surface should BEHAVE — touch targets, sheets rather
	// than dialogs, back navigation — as distinct from how it should look, which is
	// the theme's Platform. It follows the window's size, not runtime.GOOS, so a
	// narrow desktop window is a phone and the mobile layout is testable without a
	// device.
	FormFactor() FormFactor

	// Content and input
	SetContent(c Drawable)
	Content() Drawable
	SetSink(s Sink)
	ValidateLayout()

	// Visibility and window state
	Show()
	Close()
	Minimize()
	Maximize()
	Restore()
	ToggleMaximize()
	SetFullscreen(on bool)
	// Raise brings the window to the front, for a caller that wants to focus an
	// already-open window instead of opening a second copy of it.
	Raise()

	// Chrome
	SetCursor(name string)

	// Threading
	PostTask(fn func())

	// Lifecycle and platform hooks. Desktop delivers zero values / never
	// calls them; they exist so mobile backends slot in without a
	// signature change.
	SetOnClose(fn func() bool)
	SetOnResize(fn func(graphics.Size))
	SetOnLifecycle(fn func(LifecycleEvent))
	SetOnInsets(fn func(Insets))
}

// App is the process-level platform contract: it owns the event loop, the
// UI-thread task queue and the OS services that are not per-window.
type App interface {
	// Run starts the event loop and returns when the last window closes.
	Run(setup func()) error
	Quit()

	// CreateWindow opens an OS window. The backend owns creation so the
	// toolkit's app facade never names a concrete backend.
	CreateWindow(opts Options) (Window, error)

	// Task queue (see TaskQueue for the threading contract).
	PostTask(fn func())
	InvokeTaskAfter(fn func(), d time.Duration)
	OnUIThread() bool

	// OS services
	Displays() []*Display
	Clipboard() *Clipboard
	Dialogs() Dialogs
	// Tray returns the system-tray service, or false when the backend has
	// none (mobile, or a desktop backend without tray support yet).
	Tray() (Tray, bool)

	// Capabilities (see capability.go). Each accessor answers false when the
	// backend — or the device — cannot provide it, so an app checks once and
	// degrades, and never learns which OS call is underneath. Tray above is the
	// same shape; these are the rest of the surface.
	Power() (Power, bool)
	Sensors() (Sensors, bool)
	Haptics() (Haptics, bool)
	// Permissions is the gate for most hardware, so it is the one capability an
	// app is most likely to need and the one most likely to be absent.
	Permissions() (Permissions, bool)
	// Connectivity, Share and Notifications are the OS integrations that need no
	// hardware: each answers false when the platform has none.
	Connectivity() (Connectivity, bool)
	Share() (Share, bool)
	Notifications() (Notifications, bool)
	Location() (Location, bool)
	Biometrics() (Biometrics, bool)
	// SystemBars controls the system bars' appearance and whether the app draws behind
	// them. It reports false on a desktop, which has neither.
	SystemBars() (SystemBars, bool)
	// SystemTheme is the OS's light/dark preference, unknown when it has none.
	SystemTheme() SystemTheme

	// Native is the ONE intentional leak (see native.go): an opaque handle to the
	// backend's platform object, for the surfaces the seam deliberately does not
	// abstract. Opt-in, and an app that uses it is writing platform-specific code
	// on purpose.
	Native() (Native, bool)
}
