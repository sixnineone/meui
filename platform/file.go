package platform

// File locations, and the I/O that goes with them.
//
// WHY THIS IS NOT A `string`. The dialog seam used to hand the app a path, and a path is a
// desktop-only idea: SDL3's own header says the Android file dialog returns `content://` URIs, which
// are openable only through the platform's content resolver, and iOS hands out security-scoped
// bookmarks inside a sandbox. All three arrive as a STRING, so an app that treats the result as a path
// compiles, passes every test on the developer's machine, and is dead on mobile — the worst shape a
// portability bug can have.
//
// So a location is an OPAQUE FileRef and the I/O goes through this seam: the app asks the seam to read
// or write, and never parses, splits, joins or opens the location itself. On desktop a ref carries a
// path and the default implementation below is `os`; a mobile backend replaces the implementation with
// one that talks to the content resolver or a bookmark, and no call site changes.

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// fileKind is what the ref's payload means. It exists so the seam can tell a path from a URI without
// guessing at string contents, which is exactly the guess this type is here to prevent.
type fileKind uint8

const (
	kindNone fileKind = iota
	kindPath
	kindURI
)

// FileRef is an opaque reference to a file the user chose. The zero value is "no file".
//
// Nothing outside this package should interpret Loc; DisplayPath and DisplayName exist for the two
// legitimate uses (a window title, a status line), and they are allowed to be empty.
type FileRef struct {
	loc  string
	kind fileKind
}

// PathRef makes a ref from a filesystem path. This is what a desktop backend returns.
func PathRef(p string) FileRef {
	if p == "" {
		return FileRef{}
	}
	return FileRef{loc: p, kind: kindPath}
}

// URIRef makes a ref from a platform URI — an Android `content://` document, for instance. The default
// I/O below cannot open one, and says so with ErrUnsupportedRef rather than failing obscurely inside
// os.Open: a mobile backend installs an implementation that can.
func URIRef(uri string) FileRef {
	if uri == "" {
		return FileRef{}
	}
	return FileRef{loc: uri, kind: kindURI}
}

// Valid reports whether the ref refers to anything.
func (r FileRef) Valid() bool { return r.kind != kindNone && r.loc != "" }

// Scheme names the kind of location, for logs and diagnostics. It is NOT for choosing how to open the
// file: that is the seam's job.
func (r FileRef) Scheme() string {
	switch r.kind {
	case kindPath:
		return "path"
	case kindURI:
		return "uri"
	default:
		return ""
	}
}

// DisplayName is the file's base name, for a title bar. A URI's last path segment is the closest thing
// it has to one.
func (r FileRef) DisplayName() string {
	if !r.Valid() {
		return ""
	}
	if r.kind == kindURI {
		if i := strings.LastIndex(r.loc, "/"); i >= 0 && i+1 < len(r.loc) {
			return r.loc[i+1:]
		}
		return r.loc
	}
	return path.Base(r.loc)
}

// DisplayPath returns the file's path, and ONLY for a ref that is a path. It is empty for a URI, which
// is the point: a caller that wants "the location as text" to hand to something else has to notice
// that there isn't one, instead of receiving a URI and passing it to a function that expects a path.
func (r FileRef) DisplayPath() string {
	if r.kind != kindPath {
		return ""
	}
	return r.loc
}

// String is for logs. It is deliberately NOT the location for a URI, whose contents are long and not
// useful in a message; use DisplayPath when a path is genuinely wanted.
func (r FileRef) String() string {
	if !r.Valid() {
		return "(none)"
	}
	if r.kind == kindPath {
		return r.loc
	}
	return r.Scheme() + ":" + r.DisplayName()
}

// ErrUnsupportedRef is what the file I/O returns for a location this build cannot open — a URI with no
// backend installed, most often. It is a distinct error so an app can say "this platform cannot open
// that here" rather than showing a raw filesystem failure.
var ErrUnsupportedRef = errors.New("platform: this build has no implementation for that file location")

// FileIO is the backend's file I/O. A desktop build needs none — the default is `os` — and a mobile
// backend installs one that understands its platform's locations.
type FileIO struct {
	Read  func(FileRef) ([]byte, error)
	Write func(FileRef, func(io.Writer) error) error
	// WriteQuick is WriteFile without the fsync; see WriteFileQuick. A backend that cannot offer a
	// cheaper write may leave it nil, and WriteFileQuick falls back to Write.
	WriteQuick func(FileRef, func(io.Writer) error) error
}

var fileIO = FileIO{Read: osReadFile, Write: osWriteFile, WriteQuick: osWriteFileQuick}

// SetFileIO installs the backend's file I/O. Nil restores the default, which is `os` for paths and
// ErrUnsupportedRef for anything else.
func SetFileIO(fn FileIO) {
	if fn.Read == nil && fn.Write == nil {
		fileIO = FileIO{Read: osReadFile, Write: osWriteFile, WriteQuick: osWriteFileQuick}
		return
	}
	fileIO = fn
}

// ReadFile reads a location through the seam.
func ReadFile(ref FileRef) ([]byte, error) {
	if !ref.Valid() {
		return nil, errors.New("platform: no file")
	}
	if fileIO.Read == nil {
		return nil, ErrUnsupportedRef
	}
	return fileIO.Read(ref)
}

// OpenReader reads a location as a stream, for a file too large to want in memory whole.
//
// The default implementation is whole-file; a backend that can truly stream replaces `fileIO.Read` or this
// outright, and the comment here is the contract it is expected to keep.
//
// `bytes.NewReader(data)`, NOT `strings.NewReader(string(data))`: the second one copies the whole file
// AGAIN and makes a binary payload a string on the way, which is exactly the copy this function exists to
// let a caller avoid.
func OpenReader(ref FileRef) (io.ReadCloser, error) {
	if !ref.Valid() {
		return nil, errors.New("platform: no file")
	}
	if fileIO.Read == nil {
		return nil, ErrUnsupportedRef
	}
	data, err := fileIO.Read(ref)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

// WriteFile writes a location through the seam, and is ATOMIC WHERE THE PLATFORM ALLOWS: the default
// implementation writes a temporary file beside the target and renames it, so an interrupted save
// leaves the original intact rather than a half-written file that looks like the document.
//
// The promise is deliberately worded that way. A mobile content resolver hands out a stream that
// cannot be renamed over, so a backend there can only write in place — and an app must not be told it
// got atomicity it did not get.
func WriteFile(ref FileRef, write func(io.Writer) error) error {
	if !ref.Valid() {
		return errors.New("platform: no file")
	}
	if write == nil {
		return errors.New("platform: nothing to write")
	}
	if fileIO.Write == nil {
		return ErrUnsupportedRef
	}
	return fileIO.Write(ref, write)
}

// WriteFileQuick writes a location atomically but WITHOUT the fsync that makes WriteFile durable across
// a power loss.
//
// THE DISTINCTION IS WORTH MAKING because an fsync costs a disk round trip, and not everything an app
// writes deserves one: a recent-files list or a session file can be lost with no harm, while a document
// the user saved cannot. An app that used the durable write for its own state paid for a promise it did
// not need on every open — measurable on a slow disk, invisible on a fast one, which is the worst kind
// of performance bug to find later.
func WriteFileQuick(ref FileRef, write func(io.Writer) error) error {
	if !ref.Valid() {
		return errors.New("platform: no file")
	}
	if write == nil {
		return errors.New("platform: nothing to write")
	}
	if fileIO.WriteQuick != nil {
		return fileIO.WriteQuick(ref, write)
	}
	if fileIO.Write == nil {
		return ErrUnsupportedRef
	}
	return fileIO.Write(ref, write)
}

// FileNameIsPath reports whether a raw location string the OS gave us should be treated as a path
// rather than a URI. It exists so the DIALOG backends can tag what they return without each of them
// re-deriving the rule: a scheme followed by "//" is a URI, anything else is a path.
func FileNameIsPath(loc string) bool {
	if loc == "" {
		return false
	}
	if i := strings.Index(loc, "://"); i > 0 {
		// A Windows drive letter ("C:\...") has no "//", so this cannot misfire on one.
		scheme := loc[:i]
		for _, c := range scheme {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.') {
				return true // not a scheme, so not a URI
			}
		}
		return false
	}
	return true
}

// RefFor makes the right kind of ref for a raw location string from a backend.
func RefFor(loc string) FileRef {
	if FileNameIsPath(loc) {
		return PathRef(loc)
	}
	return URIRef(loc)
}

// ── storing a location ──────────────────────────────────────────────
//
// A recent-files list has to write a location down and read it back later. That is the one thing a
// location string is legitimately FOR, so it is an explicit pair of functions rather than something a
// caller does by reading the payload: the encoding is the seam's, and a caller cannot accidentally
// treat the result as a path.
//
// THE MOBILE CAVEAT, because it is not obvious and it matters: on Android a `content://` URI restored
// from a file is DEAD unless the app also took a persistable URI permission for it when the user chose
// it. So a recents list that works on desktop will show entries that fail to open on a phone until that
// permission is taken — a failure the app must report rather than hide.

// Serialize encodes a location for storage. The zero ref encodes as an empty string.
func (r FileRef) Serialize() string {
	if !r.Valid() {
		return ""
	}
	return r.Scheme() + ":" + r.loc
}

// DeserializeRef reads a location written by Serialize. An unrecognised or empty string is the zero ref:
// a recents list with a line this build cannot read should lose that line, not fail to load at all.
func DeserializeRef(s string) FileRef {
	i := strings.Index(s, ":")
	if i <= 0 {
		return FileRef{}
	}
	scheme, loc := s[:i], s[i+1:]
	switch scheme {
	case "path":
		return PathRef(loc)
	case "uri":
		return URIRef(loc)
	default:
		return FileRef{}
	}
}

// AppDataDir is where an app may keep files of its own — a recent list, a session, a cache. It is NOT a
// user document: nothing here is shown in a file manager or chosen by a picker.
//
// The default is the user's config directory plus the app's name, which is right on the desktop. A
// backend replaces it with the platform's own answer (SDL_GetPrefPath on a phone), because an app's
// writable home is one of the things an OS decides rather than an app.
func AppDataDir(app string) (FileRef, error) {
	if appDataDirFn != nil {
		return appDataDirFn(app)
	}
	if app == "" {
		return FileRef{}, errors.New("platform: AppDataDir needs an app name")
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return FileRef{}, err
	}
	dir := filepath.Join(base, app)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return FileRef{}, err
	}
	return PathRef(dir), nil
}

// SetAppDataDir installs the backend's answer for where an app may write. Nil restores the default.
func SetAppDataDir(fn func(app string) (FileRef, error)) { appDataDirFn = fn }

var appDataDirFn func(app string) (FileRef, error)
