package theme

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Platform detection: maps the runtime OS/desktop to a preset family
// and follows the system light/dark preference. Probes shell out to
// the platform's own tools (defaults/reg/gsettings/kreadconfig) so the
// zero-cgo rule holds; every probe fails soft to (dark, unknown).

// DetectPlatform reports the runtime platform family and mode.
func DetectPlatform() (Platform, Mode) {
	switch runtime.GOOS {
	case "darwin":
		return PlatformMacOS, modeFromProbe(macOSDark())
	case "windows":
		return PlatformWindows11, modeFromProbe(windowsDark())
	case "android", "ios":
		// The mobile families have no desktop probe to read a mode from; the app's
		// own theme decides, and dark is the safe default.
		return platformForGOOS(runtime.GOOS), ModeDark
	default:
		return detectLinux()
	}
}

// platformForGOOS is the GOOS-to-family mapping, split out so a test can assert the
// mobile cases exist: the failure mode is a MISSING case, which does not error but
// silently falls through to the desktop path.
func platformForGOOS(goos string) Platform {
	switch goos {
	case "android":
		// Go reports android as its own GOOS, so without this case a phone falls
		// through to the Linux DESKTOP path and gets a 26px meui preset — which is
		// what every control on the device was.
		return PlatformAndroid
	case "ios":
		return PlatformIOS
	case "darwin":
		return PlatformMacOS
	case "windows":
		return PlatformWindows11
	default:
		p, _ := detectLinux()
		return p
	}
}

// AutoPresetID returns the preset that best matches the runtime
// platform + system mode. Unknown desktops fall back to the meui
// family, never to another vendor's look.
func AutoPresetID() string {
	p, m := DetectPlatform()
	mode := "dark"
	if m == ModeLight {
		mode = "light"
	}
	switch p {
	case PlatformMacOS:
		return "macos-" + mode
	case PlatformWindows11:
		return "windows11-" + mode
	case PlatformGNOME:
		return "adwaita-" + mode
	case PlatformKDE:
		return "breeze-" + mode
	case PlatformAndroid:
		return "android-" + mode
	case PlatformIOS:
		return "ios-" + mode
	default:
		return "meui-" + mode
	}
}

// LoadAutoPreset installs the detected platform preset and overlays
// the system accent color when the platform exposes one.
func LoadAutoPreset() error {
	if err := LoadPreset(AutoPresetID()); err != nil {
		return err
	}
	if c, ok := DetectAccent(); ok {
		ApplyAccent(DefaultTheme, c)
		Apply(DefaultTheme) // regenerate widget themes with the new accent
	}
	return nil
}

// platformFromDesktop maps XDG_CURRENT_DESKTOP to a platform family.
// Exposed for tests; unknown desktops return PlatformMeui.
func platformFromDesktop(desktop string) Platform {
	d := strings.ToLower(desktop)
	switch {
	case strings.Contains(d, "gnome"), strings.Contains(d, "unity"),
		strings.Contains(d, "budgie"), strings.Contains(d, "pantheon"):
		return PlatformGNOME
	case strings.Contains(d, "kde"), strings.Contains(d, "plasma"):
		return PlatformKDE
	default:
		return PlatformMeui
	}
}

func detectLinux() (Platform, Mode) {
	p := platformFromDesktop(os.Getenv("XDG_CURRENT_DESKTOP"))
	switch p {
	case PlatformGNOME:
		return p, modeFromProbe(gnomeDark())
	case PlatformKDE:
		return p, modeFromProbe(kdeDark())
	default:
		return PlatformMeui, modeFromProbe(gtkDark())
	}
}

// modeFromProbe converts (isDark, known) to a Mode; unknown defaults to
// dark, the meui house default.
func modeFromProbe(dark, known bool) Mode {
	if known && !dark {
		return ModeLight
	}
	return ModeDark
}

// ── Probes ─────────────────────────────────────────────────────

// runProbe runs a command and returns trimmed stdout ("" on failure).
func runProbe(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func macOSDark() (bool, bool) {
	out := runProbe("defaults", "read", "-g", "AppleInterfaceStyle")
	if out == "" {
		// Key absent = light mode; but distinguish "tool missing".
		if _, err := exec.LookPath("defaults"); err != nil {
			return false, false
		}
		return false, true
	}
	return strings.EqualFold(out, "Dark"), true
}

func windowsDark() (bool, bool) {
	out := runProbe("reg", "query",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`,
		"/v", "AppsUseLightTheme")
	if out == "" {
		return false, false
	}
	// "...    AppsUseLightTheme    REG_DWORD    0x1"
	fields := strings.Fields(out)
	for i, f := range fields {
		if strings.EqualFold(f, "AppsUseLightTheme") && i+2 < len(fields) {
			return strings.HasPrefix(fields[i+2], "0x0"), true
		}
	}
	return false, false
}

func gnomeDark() (bool, bool) {
	if out := runProbe("gsettings", "get", "org.gnome.desktop.interface", "color-scheme"); out != "" {
		return strings.Contains(strings.ToLower(out), "dark"), true
	}
	if out := runProbe("gsettings", "get", "org.gnome.desktop.interface", "gtk-theme"); out != "" {
		return strings.Contains(strings.ToLower(out), "dark"), true
	}
	return false, false
}

func kdeDark() (bool, bool) {
	for _, tool := range []string{"kreadconfig6", "kreadconfig5"} {
		if out := runProbe(tool, "--file", "kdeglobals", "--group", "General", "--key", "ColorScheme"); out != "" {
			return strings.Contains(strings.ToLower(out), "dark"), true
		}
		if out := runProbe(tool, "--file", "kdeglobals", "--group", "KDE", "--key", "LookAndFeelPackage"); out != "" {
			return strings.Contains(strings.ToLower(out), "dark"), true
		}
	}
	return false, false
}

func gtkDark() (bool, bool) {
	if out := runProbe("gsettings", "get", "org.gnome.desktop.interface", "gtk-theme"); out != "" {
		return strings.Contains(strings.ToLower(out), "dark"), true
	}
	// KDE's GTK bridge on non-GNOME systems.
	if out := runProbe("kreadconfig6", "--file", "kdeglobals", "--group", "General", "--key", "ColorScheme"); out != "" {
		return strings.Contains(strings.ToLower(out), "dark"), true
	}
	return false, false
}
