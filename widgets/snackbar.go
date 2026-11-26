package widgets

import (
	"time"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/pathop"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/theme"
)

// DefaultSnackbarDuration is how long a snackbar stays up. Four seconds is the short end
// of the platform's range and is deliberate: a message that outlives its relevance is
// just something in the way, and an app with something longer to say wants a dialog.
const DefaultSnackbarDuration = 4 * time.Second

// Snackbar is a brief message at the bottom of the screen, with an optional action.
//
// It is a WIDGET IN THE APP'S OWN TREE rather than a global overlay. The toolkit's one
// global overlay is a drag ghost, and a snackbar that floated above every window would
// have to answer questions about which window it belongs to; a shell places it where it
// belongs and gets the right answer for free.
//
// The app shows it and it dismisses itself. Nothing has to poll: the timer is a platform
// task, so it runs on the UI thread like every other callback.
type Snackbar struct {
	meui.BaseWidget
	*theme.PopupTheme

	text    string
	action  string
	onDo    func()
	showing bool
	// seq invalidates a pending dismissal when a second message replaces the first —
	// without it, showing a new message would be hidden by the OLD one's timer.
	seq int

	// Duration overrides how long the message stays up; zero uses
	// DefaultSnackbarDuration.
	Duration time.Duration
	// OnDismissed fires when the message goes away, however it went away.
	OnDismissed func()
}

// NewSnackbar builds a hidden snackbar.
func NewSnackbar() *Snackbar {
	return &Snackbar{
		BaseWidget: *meui.NewBaseWidget(),
		PopupTheme: theme.DefaultPopupTheme,
	}
}

// Show displays a message with no action.
func (s *Snackbar) Show(text string) { s.show(text, "", nil) }

// ShowAction displays a message with an action button.
func (s *Snackbar) ShowAction(text, action string, onDo func()) {
	s.show(text, action, onDo)
}

func (s *Snackbar) show(text, action string, onDo func()) {
	s.text, s.action, s.onDo = text, action, onDo
	s.showing = true
	s.seq++
	seq := s.seq
	s.SetVisible(true)
	s.MarkForRedraw()

	d := s.Duration
	if d <= 0 {
		d = DefaultSnackbarDuration
	}
	// An ACTION keeps it up longer: the platform's guidance is that a message with
	// something to do gets more time than one that is only read, because the user has to
	// decide and then aim.
	if action != "" {
		d *= 2
	}
	platform.InvokeTaskAfter(func() {
		if s.seq == seq { // not superseded
			s.Dismiss()
		}
	}, d)
}

// Dismiss hides the message immediately.
func (s *Snackbar) Dismiss() {
	if !s.showing {
		return
	}
	s.showing = false
	s.seq++ // cancel any pending dismissal for this message
	s.SetVisible(false)
	s.MarkForRedraw()
	if s.OnDismissed != nil {
		s.OnDismissed()
	}
}

// Showing reports whether a message is up.
func (s *Snackbar) Showing() bool { return s.showing }

// Visible is false while no message is up. A Snackbar lives in the shell's FLOW, so one that
// reported itself visible while hidden would reserve layout space AND take every press in its
// band — which is what it did: on the device a drag over the page's lower band landed on the
// hidden Snackbar instead of the widget under the finger, and the ScrollPane never scrolled.
// Deriving it from the message's own state means no caller can forget to hide it.
func (s *Snackbar) Visible() bool { return s.showing && s.BaseWidget.Visible() }

func (s *Snackbar) Text() string { return s.text }

// SetBoundProperty implements the QML binding hook.
func (s *Snackbar) SetBoundProperty(name string, value interface{}) {
	if name == "text" {
		if str, ok := value.(string); ok {
			s.Show(str)
		}
	}
}

// ── geometry ────────────────────────────────────────────────────────

// BarHeight is the message's height, from the popup metrics so it matches the other
// floating surfaces.
func (s *Snackbar) BarHeight() float32 {
	if h := theme.DefaultTheme.Metrics.NavBarHeight; h > 0 {
		return h - 8
	}
	return 48
}

// ActionRect is the action button's rect, right-aligned, in the bar's own coordinates.
func (s *Snackbar) ActionRect() (graphics.Rect, bool) {
	if s.action == "" {
		return graphics.Rect{}, false
	}
	h := s.BarHeight()
	font := s.font()
	t := graphics.NewText(s.action, &graphics.TextDecoration{Font: font, OnBackgroundInk: s.HeadingTextInk})
	w := t.Width() + 24
	// The action is a finger target, so it is at least the touch floor wide.
	if floor := theme.DefaultTheme.Metrics.MinTouchTarget; w < floor {
		w = floor
	}
	return graphics.NewRect(s.Width()-w-8, (h-40)/2, w, 40), true
}

func (s *Snackbar) PreferredDimensions(target interface{}, w, h float32) (float32, float32) {
	return w, s.BarHeight()
}

// ── interaction ─────────────────────────────────────────────────────

func (s *Snackbar) OnMouseDown(x, y float32, button int) bool {
	if !s.showing || !s.Enabled() || button != mouseButtonLeft {
		return false
	}
	if r, ok := s.ActionRect(); ok && r.Contains(x, y) {
		return true
	}
	// A press anywhere else on the message dismisses it, which is what a user tries
	// first and what the platform does.
	s.Dismiss()
	return true
}

func (s *Snackbar) OnMouseUp(x, y float32, button int) bool {
	if !s.showing || button != mouseButtonLeft {
		return false
	}
	if r, ok := s.ActionRect(); ok && r.Contains(x, y) {
		// The action runs BEFORE the dismissal so the callback can show another message.
		do := s.onDo
		s.Dismiss()
		if do != nil {
			do()
		}
		return true
	}
	return true
}

// ── paint ───────────────────────────────────────────────────────────

func (s *Snackbar) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !s.showing || !s.Visible() || rect.Width < 1 || rect.Height < 1 {
		return
	}
	canvas.Save()
	canvas.Translate(rect.X, rect.Y)

	h := s.BarHeight()
	if h > rect.Height {
		h = rect.Height
	}
	// Inset from the edges so it reads as floating rather than as a bar.
	margin := float32(8)
	w := rect.Width - margin*2
	r := graphics.NewRect(margin, 0, w, h)

	bg := graphics.NewPaint()
	bg.SetColor(s.BackgroundInk)
	radius := theme.DefaultTheme.Metrics.Shape.Popup
	if radius <= 0 {
		radius = 8
	}
	canvas.DrawRoundedRect(r, radius, radius, bg)

	font := s.font()
	ink := s.HeadingTextInk
	p := graphics.NewPaint()
	p.SetColor(ink)

	// The message is clipped so a long one cannot run under the action.
	textRight := r.X + r.Width - 8
	if ar, ok := s.ActionRect(); ok {
		textRight = ar.X - 8
	}
	canvas.Save()
	canvas.ClipRect(graphics.NewRect(r.X+12, r.Y, textRight-(r.X+12), r.Y+r.Height), pathop.Intersect, false)
	t := graphics.NewText(s.text, &graphics.TextDecoration{Font: font, OnBackgroundInk: ink})
	t.Draw(canvas, r.X+12, r.Y+(r.Height+font.Baseline())/2-font.Baseline()/2)
	canvas.Restore()

	if ar, ok := s.ActionRect(); ok {
		ap := graphics.NewPaint()
		ap.SetColor(s.HeadingTextInk)
		at := graphics.NewText(s.action, &graphics.TextDecoration{Font: font, OnBackgroundInk: s.HeadingTextInk})
		canvas.Save()
		canvas.ClipRect(ar, pathop.Intersect, false)
		at.Draw(canvas, ar.X+(ar.Width-at.Width())/2,
			ar.Y+(ar.Height+font.Baseline())/2-font.Baseline()/2)
		canvas.Restore()
	}

	canvas.Restore()
}

func (s *Snackbar) font() graphics.Font {
	if s.PopupTheme != nil && s.PopupTheme.Font != nil {
		return s.PopupTheme.Font
	}
	if f := theme.DefaultTheme.BaseFont; f != nil {
		return f
	}
	return graphics.SystemFont
}

var _ meui.EventListener = (*Snackbar)(nil)
