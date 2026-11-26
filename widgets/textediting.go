package widgets

// textediting.go — the editing rules the text widgets share: what a WORD is (for movement, deletion
// and double-click selection), which rune a cursor step lands on, and which modifier the platform
// calls "command".
//
// They live in one place because the two widgets had two answers. TextField and TextArea both moved
// by runs of non-space, so Ctrl+Left jumped over "foo.bar" as one word while a double-click there
// selected only "foo" (wordRangeAt in textlayout.go is class-aware) — the SAME gesture disagreeing
// with itself inside one widget. TextArea deleted by BYTE, which splits a multi-byte character in
// half and leaves the document invalid UTF-8. Both are fixed by routing every caller here.

import (
	"runtime"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/sixnineone/meui/platform"
)

// isApplePlatform reports whether the platform's command key is Meta (Cmd) rather than Ctrl.
func isApplePlatform() bool {
	return runtime.GOOS == "darwin" || runtime.GOOS == "ios"
}

// commandDown reports "the platform's command modifier is held": Ctrl on Linux, Windows and Android,
// Cmd on Apple platforms. Clipboard, undo and select-all use it, so a Mac user's Cmd+C is not dead.
func commandDown(mod platform.Modifiers) bool {
	if mod.ControlDown() {
		return true
	}
	return isApplePlatform() && mod.MetaDown()
}

// wordModDown reports "move or delete by word". It is the same as commandDown except on Apple
// platforms, where the word modifier is Option.
func wordModDown(mod platform.Modifiers) bool {
	if mod.ControlDown() {
		return true
	}
	return isApplePlatform() && mod.AltDown()
}

// runeStartBefore returns the byte index of the rune boundary before pos — one cursor step left.
func runeStartBefore(s string, pos int) int {
	if pos <= 0 {
		return 0
	}
	if pos > len(s) {
		pos = len(s)
	}
	_, size := utf8.DecodeLastRuneInString(s[:pos])
	return pos - size
}

// runeEndAfter returns the byte index after the rune that starts at pos — one cursor step right.
func runeEndAfter(s string, pos int) int {
	if pos >= len(s) {
		return len(s)
	}
	if pos < 0 {
		return 0
	}
	_, size := utf8.DecodeRuneInString(s[pos:])
	return pos + size
}

// wordLeftOffset returns the byte index one WORD to the left of pos.
//
// A word is a run of one rune class — space, word (letters, digits, underscore) or punctuation — the
// same classes wordRangeAt uses, so Ctrl+Left stops exactly where a double-click would start.
// Adjacent whitespace is skipped first, which is what makes the gesture move between words rather
// than through a field of spaces.
func wordLeftOffset(s string, pos int) int {
	if pos <= 0 {
		return 0
	}
	if pos > len(s) {
		pos = len(s)
	}
	i := pos
	for i > 0 {
		r, size := utf8.DecodeLastRuneInString(s[:i])
		if !unicode.IsSpace(r) {
			break
		}
		i -= size
	}
	if i == 0 {
		return 0
	}
	first, _ := utf8.DecodeLastRuneInString(s[:i])
	class := runeClass(first)
	for i > 0 {
		r, size := utf8.DecodeLastRuneInString(s[:i])
		if runeClass(r) != class {
			break
		}
		i -= size
	}
	return i
}

// wordRightOffset returns the byte index one WORD to the right of pos: the end of the class run at
// pos, plus the whitespace that follows it (so Ctrl+Right then Ctrl+Left is a round trip).
// wordSpanAt returns the span of the run of runes at pos that share its class — the word a
// double-click selects. Unlike wordRangeAt it does NOT take the trailing whitespace: a field's
// copy-and-paste wants the word, not the gap after it.
func wordSpanAt(s string, pos int) (int, int) {
	if s == "" {
		return 0, 0
	}
	if pos < 0 {
		pos = 0
	}
	if pos > len(s) {
		pos = len(s)
	}
	if pos == len(s) {
		_, size := utf8.DecodeLastRuneInString(s[:pos])
		pos -= size
	}
	for pos > 0 && !utf8.RuneStart(s[pos]) {
		pos--
	}
	first, size := utf8.DecodeRuneInString(s[pos:])
	class := runeClass(first)
	lo, hi := pos, pos+size
	for lo > 0 {
		r, sz := utf8.DecodeLastRuneInString(s[:lo])
		if runeClass(r) != class {
			break
		}
		lo -= sz
	}
	for hi < len(s) {
		r, sz := utf8.DecodeRuneInString(s[hi:])
		if runeClass(r) != class {
			break
		}
		hi += sz
	}
	return lo, hi
}

// Undo grouping: consecutive edits of the same kind inside this window are one undo step.
const (
	undoKindNone = iota
	undoKindInsert
	undoKindDelete
)

const undoGroupWindow = 700 * time.Millisecond

func wordRightOffset(s string, pos int) int {
	if pos < 0 {
		pos = 0
	}
	if pos >= len(s) {
		return len(s)
	}
	i := pos
	if !utf8.RuneStart(s[i]) {
		// A cursor should never sit mid-rune, but if one does, step out of it rather than splitting it.
		for i > 0 && !utf8.RuneStart(s[i]) {
			i--
		}
	}
	first, _ := utf8.DecodeRuneInString(s[i:])
	class := runeClass(first)
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if runeClass(r) != class {
			break
		}
		i += size
	}
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if !unicode.IsSpace(r) {
			break
		}
		i += size
	}
	return i
}
