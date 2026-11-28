package meui

import (
	"testing"
	"time"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
)

// stubWindow records what the app facade does to it, so the facade can be
// tested without a window system.
type stubWindow struct {
	textInputActive bool
	textInputArea   graphics.Rect
	title           string
	rect            graphics.Rect
	content         platform.Drawable
	sink            platform.Sink
	shown           bool
	laidOut         bool
	raised          bool
	titleBar        platform.TitleBar
	icon            *graphics.Image
	iconCalls       int
}

func (w *stubWindow) SetTitle(s string) { w.title = s }
func (w *stubWindow) SetTextInput(active bool, area graphics.Rect) {
	w.textInputActive, w.textInputArea = active, area
}
func (w *stubWindow) Title() string                                { return w.title }
func (w *stubWindow) SetContentRect(r graphics.Rect)               { w.rect = r }
func (w *stubWindow) ContentRect() graphics.Rect                   { return w.rect }
func (w *stubWindow) Width() int32                                 { return int32(w.rect.Width) }
func (w *stubWindow) Height() int32                                { return int32(w.rect.Height) }
func (w *stubWindow) Scale() float32                               { return 1 }
func (w *stubWindow) Insets() platform.Insets                      { return platform.Insets{} }
func (w *stubWindow) FormFactor() platform.FormFactor              { return platform.FormFactorDesktop }
func (w *stubWindow) SetContent(c platform.Drawable)               { w.content = c }
func (w *stubWindow) Content() platform.Drawable                   { return w.content }
func (w *stubWindow) SetSink(s platform.Sink)                      { w.sink = s }
func (w *stubWindow) ValidateLayout()                              { w.laidOut = true }
func (w *stubWindow) Show()                                        { w.shown = true }
func (w *stubWindow) Close()                                       {}
func (w *stubWindow) Minimize()                                    {}
func (w *stubWindow) Maximize()                                    {}
func (w *stubWindow) Restore()                                     {}
func (w *stubWindow) ToggleMaximize()                              {}
func (w *stubWindow) SetFullscreen(bool)                           {}
func (w *stubWindow) Raise()                                       { w.raised = true }
func (w *stubWindow) SetCursor(string)                             {}
func (w *stubWindow) PostTask(func())                              {}
func (w *stubWindow) SetOnClose(func() bool)                       {}
func (w *stubWindow) SetOnResize(func(graphics.Size))              {}
func (w *stubWindow) SetOnLifecycle(func(platform.LifecycleEvent)) {}
func (w *stubWindow) SetOnInsets(func(platform.Insets))            {}
func (w *stubWindow) SetTitleBar(tb platform.TitleBar)             { w.titleBar = tb }
func (w *stubWindow) SetIcon(img *graphics.Image) bool {
	w.icon = img
	w.iconCalls++
	return img != nil
}

// stubApp is a platform.App that hands out stubWindows.
type stubApp struct {
	created  []*stubWindow
	lastOpts platform.Options
	ran      bool
	quit     bool
	posted   int
}

func (a *stubApp) Run(setup func()) error { a.ran = true; return nil }
func (a *stubApp) Quit()                  { a.quit = true }
func (a *stubApp) CreateWindow(opts platform.Options) (platform.Window, error) {
	a.lastOpts = opts
	// A real backend applies the title at creation (sdl3.NewWindowSized does).
	w := &stubWindow{title: opts.Title}
	a.created = append(a.created, w)
	return w, nil
}
func (a *stubApp) PostTask(fn func())                          { a.posted++; platform.InvokeTask(fn) }
func (a *stubApp) InvokeTaskAfter(fn func(), d time.Duration)  {}
func (a *stubApp) OnUIThread() bool                            { return true }
func (a *stubApp) Displays() []*platform.Display               { return nil }
func (a *stubApp) Clipboard() *platform.Clipboard              { return platform.GlobalClipboard }
func (a *stubApp) Dialogs() platform.Dialogs                   { return nil }
func (a *stubApp) Tray() (platform.Tray, bool)                 { return nil, false }
func (a *stubApp) Power() (platform.Power, bool)               { return nil, false }
func (a *stubApp) Sensors() (platform.Sensors, bool)           { return nil, false }
func (a *stubApp) Haptics() (platform.Haptics, bool)           { return nil, false }
func (a *stubApp) Permissions() (platform.Permissions, bool)   { return nil, false }
func (a *stubApp) Connectivity() (platform.Connectivity, bool) { return nil, false }
func (a *stubApp) Share() (platform.Share, bool)               { return nil, false }
func (a *stubApp) Notifications() (platform.Notifications, bool) {
	return nil, false
}
func (a *stubApp) SystemTheme() platform.SystemTheme       { return platform.ThemeUnknown }
func (a *stubApp) Native() (platform.Native, bool)         { return nil, false }
func (a *stubApp) Location() (platform.Location, bool)     { return nil, false }
func (a *stubApp) Biometrics() (platform.Biometrics, bool) { return nil, false }
func (a *stubApp) SystemBars() (platform.SystemBars, bool) { return nil, false }

var _ platform.App = (*stubApp)(nil)
var _ platform.Window = (*stubWindow)(nil)

// TestAppFacadeWiresWindow is the facade contract: opening a window installs
// the widget tree as content and sink, lays it out and shows it.
func TestAppFacadeWiresWindow(t *testing.T) {
	backend := &stubApp{}
	app := NewApp(backend, AppOptions{Title: "Default", Size: graphics.Size{Width: 640, Height: 480}})

	root := NewWindow()

	wnd, err := app.AddWindow(root, WindowOptions{Title: "TradeUI", Size: graphics.Size{Width: 320, Height: 200}})
	if err != nil {
		t.Fatalf("AddWindow: %v", err)
	}
	sw, ok := wnd.(*stubWindow)
	if !ok {
		t.Fatalf("window is %T", wnd)
	}

	if backend.lastOpts.Title != "TradeUI" || backend.lastOpts.Width != 320 || backend.lastOpts.Height != 200 {
		t.Fatalf("CreateWindow opts = %+v", backend.lastOpts)
	}
	if sw.title != "TradeUI" {
		t.Errorf("title = %q", sw.title)
	}
	if sw.content == nil {
		t.Error("content was not installed")
	}
	if sw.sink != platform.Sink(root) {
		t.Error("the widget tree was not installed as the event sink")
	}
	if !sw.laidOut {
		t.Error("ValidateLayout was not called")
	}
	if !sw.shown {
		t.Error("Show was not called")
	}
	if len(app.Windows()) != 1 {
		t.Errorf("app.Windows() = %d", len(app.Windows()))
	}
}

// TestAppFacadeFallsBackToDefaults pins the defaulting rules.
func TestAppFacadeFallsBackToDefaults(t *testing.T) {
	backend := &stubApp{}
	app := NewApp(backend, AppOptions{Title: "Fallback", Size: graphics.Size{Width: 800, Height: 600}})

	if _, err := app.AddWindow(nil, WindowOptions{}); err != nil {
		t.Fatalf("AddWindow: %v", err)
	}
	if backend.lastOpts.Title != "Fallback" {
		t.Errorf("title = %q, want the app default", backend.lastOpts.Title)
	}
	if backend.lastOpts.Width != 800 || backend.lastOpts.Height != 600 {
		t.Errorf("size = %dx%d, want the app default", backend.lastOpts.Width, backend.lastOpts.Height)
	}
}

func TestAppFacadeRunQuitPostTask(t *testing.T) {
	backend := &stubApp{}
	app := NewApp(backend, AppOptions{})

	if err := app.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !backend.ran {
		t.Error("Run did not reach the backend")
	}
	app.Quit()
	if !backend.quit {
		t.Error("Quit did not reach the backend")
	}
	app.PostTask(func() {})
	if backend.posted != 1 {
		t.Errorf("PostTask reached the backend %d times", backend.posted)
	}
	if app.Backend() != platform.App(backend) {
		t.Error("Backend() did not return the injected backend")
	}
}

// fakeBar is a title bar with a known height, so a test can check that the
// content is laid out below it.
type fakeBar struct{ h float32 }

func (b fakeBar) BarHeight() float32                                 { return b.h }
func (b fakeBar) PaintBar(*graphics.Canvas, float32)                 {}
func (b fakeBar) BarMouseDown(float32, float32, int) bool            { return false }
func (b fakeBar) BarMouseUp(float32, float32, int) bool              { return false }
func (b fakeBar) BarMouseMove(float32, float32)                      {}
func (b fakeBar) BarDoubleClick(float32, float32) bool               { return false }
func (b fakeBar) FrameHit(float32, float32, float32, float32) uint32 { return 0 }

func TestAddWindowInstallsChromeByDefault(t *testing.T) {
	// The desktop backends create undecorated windows, so a window with no title
	// bar cannot be moved or resized. Chrome is therefore the framework's
	// default rather than every app's job.
	backend := &stubApp{}
	app := NewApp(backend, AppOptions{})
	saved := defaultChrome
	defer func() { defaultChrome = saved }()
	SetDefaultChrome(func(root *Window, w platform.Window) platform.TitleBar {
		return fakeBar{h: 28}
	})

	root := NewWindow()
	wnd, err := app.AddWindow(root, WindowOptions{Title: "T", Size: graphics.NewSize(800, 600)})
	if err != nil {
		t.Fatal(err)
	}
	sw := wnd.(*stubWindow)
	if sw.titleBar == nil {
		t.Fatal("AddWindow did not install the default title bar")
	}
	// The content is laid out below the bar, and the tree is told where its
	// origin is — the host's declaration, done centrally.
	if got, want := sw.rect.Height, float32(600-28); got != want {
		t.Fatalf("content height %v, want %v (window height less the bar)", got, want)
	}
	if _, oy := root.ContentOrigin(); oy != 28 {
		t.Fatalf("content origin y = %v, want 28", oy)
	}

	// NoChrome opts out, and then the content fills the window.
	backend2 := &stubApp{}
	app2 := NewApp(backend2, AppOptions{})
	wnd2, err := app2.AddWindow(NewWindow(), WindowOptions{
		Title: "T", Size: graphics.NewSize(800, 600), NoChrome: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	sw2 := wnd2.(*stubWindow)
	if sw2.titleBar != nil {
		t.Fatal("NoChrome still installed a title bar")
	}
	if got, want := sw2.rect.Height, float32(600); got != want {
		t.Fatalf("NoChrome content height %v, want %v", got, want)
	}
}

func TestAddWindowInstallsTheDefaultIcon(t *testing.T) {
	backend := &stubApp{}
	app := NewApp(backend, AppOptions{Size: graphics.NewSize(400, 300)})
	win, err := app.AddWindow(nil, WindowOptions{Title: "icon"})
	if err != nil {
		t.Fatal(err)
	}
	stub := win.(*stubWindow)
	if stub.icon == nil {
		t.Fatal("no icon was installed")
	}
	if stub.icon != DefaultIcon() {
		t.Fatal("the default icon was not used")
	}
	if DefaultIcon() == nil || DefaultIcon().Width() != 128 {
		t.Fatalf("DefaultIcon = %v", DefaultIcon())
	}
}

func TestAddWindowIconOverrides(t *testing.T) {
	backend := &stubApp{}
	app := NewApp(backend, AppOptions{Size: graphics.NewSize(400, 300)})
	custom := graphics.NewImage(make([]uint32, 4*4), 4, 4)
	win, err := app.AddWindow(nil, WindowOptions{Title: "icon", Icon: custom})
	if err != nil {
		t.Fatal(err)
	}
	if got := win.(*stubWindow).icon; got != custom {
		t.Fatal("WindowOptions.Icon was not used")
	}

	win, err = app.AddWindow(nil, WindowOptions{Title: "noicon", NoIcon: true})
	if err != nil {
		t.Fatal(err)
	}
	if win.(*stubWindow).iconCalls != 0 {
		t.Fatal("NoIcon still installed an icon")
	}
}
