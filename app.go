package meui

import (
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
)

// AppOptions configures the application facade.
type AppOptions struct {
	// Title is used for the first window when a window supplies none.
	Title string
	// Size is the default window size when a window supplies none.
	Size graphics.Size
}

// WindowOptions configures one window added to the app.
type WindowOptions struct {
	Title string
	Size  graphics.Size
	// NoChrome skips the standard title bar. The default is to install it: the
	// desktop backends create undecorated windows, so without chrome a window
	// can be neither moved, maximized nor resized. Mobile hosts — where the OS
	// owns the surface — and any app that draws its own chrome pass true.
	NoChrome bool

	// Icon is the window icon used by the window manager and the taskbar.
	// Nil installs DefaultIcon(); set NoIcon to suppress the icon entirely.
	// A backend that cannot set an icon ignores it.
	Icon *graphics.Image
	// NoIcon suppresses the default icon for this window.
	NoIcon bool
}

// App is the toolkit-facing application facade: it opens windows, installs a
// widget tree as their content, and runs the backend event loop. It speaks
// only platform.App, so the backend (SDL3 today) is chosen at the composition
// root and never named here.
type App struct {
	backend platform.App
	opts    AppOptions
	windows []platform.Window
}

// NewApp wires the facade to a platform backend.
//
//	app := meui.NewApp(sdl3.NewApp(), meui.AppOptions{})
//	app.AddWindow(root, meui.WindowOptions{Title: "TradeUI"})
//	app.Run()
func NewApp(backend platform.App, opts AppOptions) *App {
	return &App{backend: backend, opts: opts}
}

// defaultChrome is the title bar every window gets unless WindowOptions.NoChrome
// is set. The widgets package registers the standard one at init, so an app
// that uses the toolkit's widgets has working chrome with no code of its own.
// meui cannot import widgets itself — that would be a cycle — so the factory
// arrives through this hook, the same way the platform services do.
var defaultChrome func(root *Window, w platform.Window) platform.TitleBar

// SetDefaultChrome registers the title bar AddWindow installs on each window.
// Passing nil disables the default. Called by the widgets package.
func SetDefaultChrome(f func(root *Window, w platform.Window) platform.TitleBar) {
	defaultChrome = f
}

// chromeSetter is the optional capability a backend implements to accept a
// title bar. It is deliberately not part of platform.Window: a backend with no
// chrome to draw (a mobile surface, the renderer-level window) has nothing to
// implement, and requiring a no-op method of it would be noise.
type chromeSetter interface {
	SetTitleBar(platform.TitleBar)
}

// AddWindow opens an OS window, installs root as its content and event sink,
// lays it out against the real window size, and shows it.
func (a *App) AddWindow(root *Window, opts WindowOptions) (platform.Window, error) {
	if opts.Title == "" {
		opts.Title = a.opts.Title
	}
	w, h := int32(opts.Size.Width), int32(opts.Size.Height)
	if w <= 0 || h <= 0 {
		w, h = int32(a.opts.Size.Width), int32(a.opts.Size.Height)
	}
	if w <= 0 || h <= 0 {
		w, h = 1280, 720
	}

	wnd, err := a.backend.CreateWindow(platform.Options{Title: opts.Title, Width: w, Height: h})
	if err != nil {
		return nil, err
	}

	// A backend may hand back a different size than was asked for: SDL on
	// Android forces fullscreen, so the surface is the display's, not the
	// requested 360x640. Lay the content out for the window that actually
	// exists, or everything past the requested width is clipped until the first
	// resize event arrives.
	if aw, ah := wnd.Width(), wnd.Height(); aw > 0 && ah > 0 && (aw != w || ah != h) {
		w, h = aw, ah
	}
	// The touch floor follows THIS WINDOW, not the OS and not the process: a phone-sized surface gets
	// it, a desktop window does not, and a resized window re-derives it (Window.HandleEvent). Deriving
	// it per window is what keeps a small window from changing how every other window's controls are
	// hit-tested — which is exactly what a process-wide switch used to do.
	if root != nil {
		root.setDerivedTouchProfile(wnd.FormFactor() != platform.FormFactorDesktop)
	}

	// Window chrome, installed before the content is laid out so the bar's
	// height is accounted for in the very first layout. Without a title bar a
	// borderless window cannot be dragged or resized at all, which is why this
	// is the default rather than something each app re-implements.
	var barH float32
	if !opts.NoChrome && defaultChrome != nil && root != nil {
		if cs, ok := wnd.(chromeSetter); ok {
			if tb := defaultChrome(root, wnd); tb != nil {
				cs.SetTitleBar(tb)
				barH = tb.BarHeight()
			}
		}
	}

	if root != nil {
		wnd.SetContentRect(graphics.Rect{X: 0, Y: 0, Width: float32(w), Height: float32(h) - barH})
		root.SetContentOrigin(0, barH)
		wnd.SetContent(root.AsPanel())
		wnd.SetSink(root)
		wnd.ValidateLayout()
	}
	wnd.Show()

	// Window icon, after the window is mapped: SDL's X11 backend publishes
	// _NET_WM_ICON when it has a live window.
	icon := opts.Icon
	if icon == nil && !opts.NoIcon {
		icon = DefaultIcon()
	}
	if icon != nil {
		// Optional capability: a backend without an icon surface (the renderer's
		// own window, a test stub) simply does not implement it.
		if setter, ok := wnd.(interface{ SetIcon(*graphics.Image) bool }); ok {
			setter.SetIcon(icon)
		}
	}

	a.windows = append(a.windows, wnd)
	return wnd, nil
}

// Windows returns the windows opened through this app.
func (a *App) Windows() []platform.Window { return a.windows }

// PostTask queues fn on the UI thread (see platform.TaskQueue for the contract).
func (a *App) PostTask(fn func()) { a.backend.PostTask(fn) }

// Run starts the event loop and returns when the last window closes.
func (a *App) Run() error { return a.backend.Run(nil) }

// Quit closes every window, which ends Run.
func (a *App) Quit() { a.backend.Quit() }

// Backend returns the platform backend, for OS services such as the clipboard
// and dialogs.
func (a *App) Backend() platform.App { return a.backend }
