package platform

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

// The capability seam: things an app asks the platform for that are not part of
// every backend, expressed in neutral terms.
//
// Two rules make "the app never learns whether this was an SDL call, an NDK
// call, a JNI call or an Objective-C message" true rather than aspirational:
//
//   - No OS names. No JNIEnv, no objc_msgSend, no ASensorManager, no
//     SDL_* in this file.
//   - Capability discovery is a RUNTIME fact, never a build tag: a device may
//     lack a gyroscope and an OS version may lack an API, so accessors return
//     (T, bool) and streams have Has(). The app checks once and degrades.
//
// Every accessor on App follows the same shape, which is the one App.Tray
// already used. See MOBILE-CAPABILITIES.md for the whole surface and
// MOBILE-PLAN.md P4 for what is implemented so far.

// --- power ----------------------------------------------------------------

// PowerState is the device's power situation.
type PowerState int

const (
	PowerUnknown PowerState = iota
	PowerOnBattery
	PowerNoBattery // plugged in, no battery: a desktop
	PowerCharging
	PowerCharged
)

func (p PowerState) String() string {
	switch p {
	case PowerOnBattery:
		return "on-battery"
	case PowerNoBattery:
		return "no-battery"
	case PowerCharging:
		return "charging"
	case PowerCharged:
		return "charged"
	default:
		return "unknown"
	}
}

// PowerInfo is one reading. Seconds and Percent are -1 when unknown.
type PowerInfo struct {
	State   PowerState
	Seconds int
	Percent int
}

// Power reports the device's power state.
type Power interface {
	Info() PowerInfo
}

// --- sensors --------------------------------------------------------------

// SensorKind is a sensor an app can ask for, in neutral terms.
type SensorKind int

const (
	SensorAccelerometer SensorKind = iota
	SensorGyroscope
	SensorMagnetometer
)

func (k SensorKind) String() string {
	switch k {
	case SensorAccelerometer:
		return "accelerometer"
	case SensorGyroscope:
		return "gyroscope"
	case SensorMagnetometer:
		return "magnetometer"
	default:
		return "unknown"
	}
}

// SensorSample is one reading. The units are the sensor's own and are the same
// on every platform: g for the accelerometer, rad/s for the gyroscope, µT for
// the magnetometer.
type SensorSample struct {
	Kind    SensorKind
	X, Y, Z float64
	At      time.Time
}

// Sensors is a device's sensor set. Has is the runtime capability check: a
// device without a gyroscope answers false, and that is not an error.
type Sensors interface {
	Has(SensorKind) bool
	// Start delivers samples until stop is called. Samples arrive on the UI
	// thread — the backend marshals them through the task queue — so the app
	// never sees an OS thread or a sensor queue. (Real-time audio is THE
	// exception to this rule; see the note in task.go and MEDIA.md D4.)
	Start(kind SensorKind, fn func(SensorSample)) (stop func(), err error)
}

// --- haptics --------------------------------------------------------------

// Haptic is a named effect, not a platform constant.
type Haptic int

const (
	HapticTap Haptic = iota
	HapticClick
	HapticSuccess
	HapticWarning
	HapticFailure
)

func (h Haptic) String() string {
	switch h {
	case HapticTap:
		return "tap"
	case HapticClick:
		return "click"
	case HapticSuccess:
		return "success"
	case HapticWarning:
		return "warning"
	case HapticFailure:
		return "failure"
	default:
		return "unknown"
	}
}

// Haptics plays short effects. Supported is the runtime check: a device with no
// vibrator answers false, and Play is then a no-op rather than an error.
type Haptics interface {
	Supported() bool
	Play(Haptic) error
}

// --- system appearance ----------------------------------------------------

// SystemTheme is the OS's light/dark preference.
type SystemTheme int

const (
	ThemeUnknown SystemTheme = iota
	ThemeLight
	ThemeDark
)

func (t SystemTheme) String() string {
	switch t {
	case ThemeLight:
		return "light"
	case ThemeDark:
		return "dark"
	default:
		return "unknown"
	}
}

// --- display orientation --------------------------------------------------

// Orientation is how the display is held.
type Orientation int

const (
	OrientationUnknown Orientation = iota
	OrientationLandscape
	OrientationLandscapeFlipped
	OrientationPortrait
	OrientationPortraitFlipped
)

func (o Orientation) String() string {
	switch o {
	case OrientationLandscape:
		return "landscape"
	case OrientationLandscapeFlipped:
		return "landscape-flipped"
	case OrientationPortrait:
		return "portrait"
	case OrientationPortraitFlipped:
		return "portrait-flipped"
	default:
		return "unknown"
	}
}

// --- permissions ----------------------------------------------------------

// Permission is a capability the OS gates behind a user grant.
//
// It is the gate for most hardware, and on Android there is no NDK permission
// API at all — the NDK's own answer is that most of it is Java-only — so this is
// where a native bridge is unavoidable and where it must be best hidden: the app
// asks for PermissionCamera and never learns whether the answer came from a JNI
// call or an Objective-C message.
type Permission int

const (
	PermissionCamera Permission = iota
	PermissionMicrophone
	PermissionLocation
	PermissionLocationBackground
	PermissionNotifications
	PermissionBluetooth
	PermissionBluetoothScan
	PermissionContacts
	PermissionCalendar
	PermissionPhotos
	PermissionMotion // iOS: CoreMotion; Android grants it at install
	PermissionVibrate
	PermissionStorageRead
)

func (p Permission) String() string {
	switch p {
	case PermissionCamera:
		return "camera"
	case PermissionMicrophone:
		return "microphone"
	case PermissionLocation:
		return "location"
	case PermissionLocationBackground:
		return "location-background"
	case PermissionNotifications:
		return "notifications"
	case PermissionBluetooth:
		return "bluetooth"
	case PermissionBluetoothScan:
		return "bluetooth-scan"
	case PermissionContacts:
		return "contacts"
	case PermissionCalendar:
		return "calendar"
	case PermissionPhotos:
		return "photos"
	case PermissionMotion:
		return "motion"
	case PermissionVibrate:
		return "vibrate"
	case PermissionStorageRead:
		return "storage-read"
	default:
		return "unknown"
	}
}

// PermissionStatus is the answer to a permission question.
//
// The distinctions matter to an app: Denied means "ask again later with a
// reason", NotDetermined means "ask now", Restricted means "do not ask, the OS
// forbids it" (parental controls, MDM), and Unavailable means the platform has
// no such permission — four different behaviours behind one word.
type PermissionStatus int

const (
	PermissionUnknown PermissionStatus = iota
	PermissionGranted
	PermissionDenied
	PermissionRestricted
	PermissionNotDetermined
	PermissionUnavailable
)

func (s PermissionStatus) String() string {
	switch s {
	case PermissionGranted:
		return "granted"
	case PermissionDenied:
		return "denied"
	case PermissionRestricted:
		return "restricted"
	case PermissionNotDetermined:
		return "not-determined"
	case PermissionUnavailable:
		return "unavailable"
	default:
		return "unknown"
	}
}

// Permissions asks for and reports OS permission grants.
type Permissions interface {
	// Status reports the current state WITHOUT prompting.
	Status(Permission) PermissionStatus
	// Request asks the user and blocks until they answer, pumping the event
	// loop meanwhile — the same nested-pump shape as a file dialog, and for the
	// same reason: the OS answers on a callback.
	Request(Permission) PermissionStatus
	// Known reports the permissions this platform actually has, so an app can
	// show an accurate list instead of guessing.
	Known() []Permission
}

// --- connectivity ---------------------------------------------------------

// Connectivity reports the device's network situation.
//
// It is a query, not a stream: an app that cares about changes can poll it on a
// timer or on resume, and the OS's own change callbacks differ enough per
// platform that a stream would be a lie about the least common denominator.
type Connectivity interface {
	// Online reports whether there is a usable network, and whether that network
	// is METERED — the second value is the one that matters, because it is what
	// tells an app not to stream video on someone's mobile data.
	Online() (online bool, metered bool)
}

// --- sharing --------------------------------------------------------------

// Share hands content to the OS share sheet.
//
// Fire and forget: the OS owns the sheet, the user may pick anything or nothing,
// and no platform reports back what they chose. An app that needs to know the
// result has to ask the user itself.
type Share interface {
	// Text shares plain text, with an optional URL. An empty URL is fine.
	Text(text, url string) error
}

// --- notifications --------------------------------------------------------

// Notification is one message to show in the OS notification area.
type Notification struct {
	// ID identifies the notification, so posting again with the same ID replaces
	// it rather than stacking a second one. Zero is a valid ID.
	ID    int
	Title string
	Body  string
}

// Notifications posts OS notifications.
type Notifications interface {
	// Supported reports whether this device and OS can show notifications at all.
	// It does NOT mean the app is allowed to: that is
	// Permissions.Status(PermissionNotifications), and asking is the app's job —
	// the seam reports, it never silently prompts.
	Supported() bool
	// Post shows a notification. It returns an error rather than prompting when
	// permission is missing, because a permission prompt belongs to the app's
	// flow, not to a side effect.
	Post(Notification) error
}

// --- location -------------------------------------------------------------

// Position is one location fix. The units are the ones every platform uses, so
// there is nothing to convert and nothing to name: degrees, metres, and a time.
type Position struct {
	Latitude       float64
	Longitude      float64
	AccuracyMeters float64
	At             time.Time
}

// Location reports where the device is, as far as the OS will say.
//
// Deliberately not a stream. A stream needs a subscription lifecycle, a power
// policy and a background-mode story on each platform, and the honest API for
// that is Start/Stop like sensors — which is a later, larger piece of work than
// the rest of this slice.
type Location interface {
	// Enabled reports whether location services are switched on for the DEVICE.
	// It says nothing about whether this app may use them — that is
	// Permissions.Status(PermissionLocation), and the two are genuinely
	// independent: services can be on while the app is refused, and off while
	// the app is allowed.
	Enabled() bool
	// Last returns the most recent fix the OS already has, if any. It never
	// waits for a new one and never starts a listener, so it cannot drain the
	// battery — and it returns false rather than blocking when there is none.
	Last() (Position, bool)
}

// --- biometrics -----------------------------------------------------------

// Biometrics is the OS's "prove it is you" prompt.
type Biometrics interface {
	// Available reports whether the device CAN authenticate: hardware present and
	// at least one credential enrolled. It is a query and never prompts.
	Available() bool
	// Authenticate prompts and blocks until the user succeeds, fails or cancels,
	// pumping the event loop meanwhile. The reason is shown to the user and is
	// required by the platforms.
	Authenticate(reason string) bool
}

// --- opening a URL --------------------------------------------------------

// defaultOpenURLSchemes is the allow-list a scene cannot widen. Links in
// untrusted Markdown or rich text reach OpenURL without the app seeing them, so
// file:, javascript:, data: and custom handlers are refused by default.
var defaultOpenURLSchemes = []string{"http", "https", "mailto", "tel"}

var (
	openURLMu      sync.RWMutex
	openURLImpl    = openURLFallback
	openURLSchemes = schemeSet(defaultOpenURLSchemes)
)

func schemeSet(schemes []string) map[string]bool {
	m := make(map[string]bool, len(schemes))
	for _, s := range schemes {
		if s = strings.ToLower(strings.TrimSpace(s)); s != "" {
			m[s] = true
		}
	}
	return m
}

// SetOpenURLSchemes replaces the schemes OpenURL accepts. Called with no
// arguments it restores the default http/https/mailto/tel list. An app that
// needs a custom scheme should add only that scheme, not choose to trust
// everything.
func SetOpenURLSchemes(schemes ...string) {
	openURLMu.Lock()
	defer openURLMu.Unlock()
	if len(schemes) == 0 {
		openURLSchemes = schemeSet(defaultOpenURLSchemes)
		return
	}
	openURLSchemes = schemeSet(schemes)
}

func allowedOpenURL(raw string) error {
	if raw == "" {
		return errors.New("empty url")
	}
	if strings.ContainsAny(raw, "\x00\r\n\t") {
		return errors.New("url contains a control character")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme == "" {
		return errors.New("url has no scheme")
	}
	openURLMu.RLock()
	ok := openURLSchemes[strings.ToLower(u.Scheme)]
	openURLMu.RUnlock()
	if !ok {
		return fmt.Errorf("scheme %q is not allowed", u.Scheme)
	}
	return nil
}

// OpenURL launches the platform handler for a URL (browser, mail client, …).
//
// It is a package-level function rather than an App method because the toolkit
// calls it from places that hold no App — a markdown link's default action —
// the same way InvokeTask and GlobalClipboard work. A backend installs its own
// with SetOpenURL at startup; the default shells out, which is the desktop
// fallback.
//
// The URL is validated before it reaches the backend: no empty URL, no control
// characters, and only a scheme in the allow-list (see SetOpenURLSchemes).
func OpenURL(raw string) error {
	if err := allowedOpenURL(raw); err != nil {
		return fmt.Errorf("platform.OpenURL: %w", err)
	}
	openURLMu.RLock()
	impl := openURLImpl
	openURLMu.RUnlock()
	return impl(raw)
}

// SetOpenURL installs the backend's implementation. Nil restores the fallback.
func SetOpenURL(fn func(string) error) {
	openURLMu.Lock()
	defer openURLMu.Unlock()
	if fn == nil {
		openURLImpl = openURLFallback
		return
	}
	openURLImpl = fn
}
