package platform

import "github.com/sixnineone/mobilegles"

type (
	KeyCode   = mobilegles.Keycode
	Modifiers = mobilegles.KeyMod
)

const (
	ModNone  = mobilegles.ModNone
	ModShift = mobilegles.ModShift
	ModCtrl  = mobilegles.ModCtrl
	ModAlt   = mobilegles.ModAlt
	ModMeta  = mobilegles.ModMeta
	ModCaps  = mobilegles.ModCaps
)

const (
	KeyUnknown = mobilegles.KeyUnknown
	KeyA       = mobilegles.KeyA
	KeyB       = mobilegles.KeyB
	KeyC       = mobilegles.KeyC
	KeyD       = mobilegles.KeyD
	KeyE       = mobilegles.KeyE
	KeyG       = mobilegles.KeyG
	KeyH       = mobilegles.KeyH
	KeyI       = mobilegles.KeyI
	KeyJ       = mobilegles.KeyJ
	KeyK       = mobilegles.KeyK
	KeyL       = mobilegles.KeyL
	KeyM       = mobilegles.KeyM
	KeyN       = mobilegles.KeyN
	KeyO       = mobilegles.KeyO
	KeyP       = mobilegles.KeyP
	KeyQ       = mobilegles.KeyQ
	KeyR       = mobilegles.KeyR
	KeyS       = mobilegles.KeyS
	KeyT       = mobilegles.KeyT
	KeyU       = mobilegles.KeyU
	KeyV       = mobilegles.KeyV
	KeyW       = mobilegles.KeyW
	KeyX       = mobilegles.KeyX
	KeyY       = mobilegles.KeyY
	KeyZ       = mobilegles.KeyZ

	KeyUp        = mobilegles.KeyUp
	KeyDown      = mobilegles.KeyDown
	KeyLeft      = mobilegles.KeyLeft
	KeyRight     = mobilegles.KeyRight
	KeySpace     = mobilegles.KeySpace
	KeyTab       = mobilegles.KeyTab
	KeyEscape    = mobilegles.KeyEscape
	KeyBackspace = mobilegles.KeyBackspace
	KeyEnter     = mobilegles.KeyEnter

	Key0 = mobilegles.Key0
	Key1 = mobilegles.Key1
	Key2 = mobilegles.Key2
	Key3 = mobilegles.Key3
	Key4 = mobilegles.Key4
	Key5 = mobilegles.Key5
	Key6 = mobilegles.Key6
	Key7 = mobilegles.Key7
	Key8 = mobilegles.Key8
	Key9 = mobilegles.Key9

	// Modifier keycodes are exported by mobilegles; re-export them so a
	// consumer handling shortcuts never has to import the renderer for what
	// is plainly an OS key. (Their omission is why meui-charts reached for the
	// Skia backend's keycodes during the port.)
	KeyLeftShift  = mobilegles.KeyLeftShift
	KeyRightShift = mobilegles.KeyRightShift
	KeyLeftCtrl   = mobilegles.KeyLeftCtrl
	KeyRightCtrl  = mobilegles.KeyRightCtrl
	KeyLeftAlt    = mobilegles.KeyLeftAlt
	KeyRightAlt   = mobilegles.KeyRightAlt
	KeyLeftMeta   = mobilegles.KeyLeftMeta
	KeyRightMeta  = mobilegles.KeyRightMeta
)

// Keys the editor and the widgets bind, re-exported from the PUMP'S OWN constants.
//
// They used to be literals here, and half of them were SDL scancode numbers while the pump delivers
// USB HID usage IDs: platform.KeyDelete was 76 and a real Delete key arrived as 212, so Delete (and
// Home, End, PageUp, PageDown, and Ctrl+F) never matched anything. `TestEveryPlatformKeycodeIsThePumps`
// pins every name to mobilegles, because a second table of key numbers is exactly how that happened.
const (
	KeyReturn      = mobilegles.KeyEnter
	KeyNumPadEnter = mobilegles.KeypadEnter

	KeyHome     = mobilegles.KeyHome
	KeyEnd      = mobilegles.KeyEnd
	KeyInsert   = mobilegles.KeyInsert
	KeyDelete   = mobilegles.KeyDelete
	KeyPageUp   = mobilegles.KeyPageUp
	KeyPageDown = mobilegles.KeyPageDown

	KeyF   = mobilegles.KeyF
	KeyF1  = mobilegles.KeyF1
	KeyF2  = mobilegles.KeyF2
	KeyF3  = mobilegles.KeyF3
	KeyF4  = mobilegles.KeyF4
	KeyF5  = mobilegles.KeyF5
	KeyF6  = mobilegles.KeyF6
	KeyF7  = mobilegles.KeyF7
	KeyF8  = mobilegles.KeyF8
	KeyF9  = mobilegles.KeyF9
	KeyF10 = mobilegles.KeyF10
	KeyF11 = mobilegles.KeyF11
	KeyF12 = mobilegles.KeyF12

	// The main-row punctuation, which the toolkit only recently started to NAME.
	KeyBackslash    = mobilegles.KeyBackslash
	KeyOpenBracket  = mobilegles.KeyOpenBracket
	KeyCloseBracket = mobilegles.KeyCloseBracket
	KeySemicolon    = mobilegles.KeySemicolon
	KeyApostrophe   = mobilegles.KeyApostrophe
	KeyGrave        = mobilegles.KeyGrave
	KeyComma        = mobilegles.KeyComma
	KeyPeriod       = mobilegles.KeyPeriod
	KeySlash        = mobilegles.KeySlash
)

// The main-row `-` and `=` keys, and the keypad, re-exported from the pump as well.
const (
	KeyMinus = mobilegles.KeyMinus
	KeyEqual = mobilegles.KeyEqual

	KeypadPlus   = mobilegles.KeypadPlus
	KeypadMinus  = mobilegles.KeypadMinus
	KeypadEquals = mobilegles.KeypadEquals
)
