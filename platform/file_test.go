package platform

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRefsCarryWhatTheyPromise pins the two accessors an app may legitimately use — a title bar wants a
// name and, on desktop, a path — and the one it may not: there is no way to get a location string out
// of a URI ref, so an app cannot pass a `content://` URI to something that expects a path.
func TestRefsCarryWhatTheyPromise(t *testing.T) {
	p := PathRef("/home/u/photos/holiday.png")
	if p.DisplayName() != "holiday.png" {
		t.Errorf("PathRef name = %q, want holiday.png", p.DisplayName())
	}
	if p.DisplayPath() != "/home/u/photos/holiday.png" {
		t.Errorf("PathRef path = %q", p.DisplayPath())
	}
	if p.Scheme() != "path" {
		t.Errorf("PathRef scheme = %q, want path", p.Scheme())
	}

	u := URIRef("content://com.android.providers.downloads/document/1234")
	if u.DisplayName() != "1234" {
		t.Errorf("URIRef name = %q, want the last path segment", u.DisplayName())
	}
	if got := u.DisplayPath(); got != "" {
		t.Errorf("URIRef path = %q, want EMPTY: a URI is not a path, and handing one out as text is how "+
			"a location gets passed to something that will try to open it as a file", got)
	}
	if u.Scheme() != "uri" {
		t.Errorf("URIRef scheme = %q, want uri", u.Scheme())
	}
	// String() is for logs, so it must not be the raw URI and must still identify the file.
	if s := u.String(); !strings.Contains(s, "1234") || strings.Contains(s, "content://") {
		t.Errorf("URIRef String() = %q, want a short form naming the file", s)
	}

	var zero FileRef
	if zero.Valid() || zero.Scheme() != "" || zero.DisplayName() != "" {
		t.Error("the zero ref is not inert")
	}
	if PathRef("").Valid() || URIRef("").Valid() {
		t.Error("an empty location produced a valid ref")
	}
}

// TestWriteFileIsAtomicOnFailure is the promise that matters. A save that fails halfway must leave the
// PREVIOUS file exactly as it was — not truncated, not half-written — and must not litter the directory
// with a temporary file.
func TestWriteFileIsAtomicOnFailure(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "document.png")
	if err := os.WriteFile(target, []byte("the previous version"), 0o644); err != nil {
		t.Fatal(err)
	}

	boom := errors.New("the encoder blew up")
	err := WriteFile(PathRef(target), func(w io.Writer) error {
		if _, err := w.Write([]byte("half a new version")); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("WriteFile returned %v, want the writer's error", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "the previous version" {
		t.Errorf("after a failed save the file holds %q; the previous version must be untouched",
			string(got))
	}
	// No temporary file left behind: a directory full of .meui-save-* is a bug report waiting to happen.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".meui-save-") {
			t.Errorf("a failed save left the temporary file %s behind", e.Name())
		}
	}
}

// TestWriteFileReplacesAndReadsBack covers the happy path and the round trip.
func TestWriteFileReplacesAndReadsBack(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "out.bin")

	if err := WriteFile(PathRef(target), func(w io.Writer) error {
		_, err := w.Write([]byte("first"))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	// A second save replaces it, and the file is readable by its owner (the temp file starts 0600).
	if err := WriteFile(PathRef(target), func(w io.Writer) error {
		_, err := w.Write([]byte("second, longer"))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	data, err := ReadFile(PathRef(target))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "second, longer" {
		t.Errorf("read back %q", string(data))
	}
	st, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0o044 == 0 {
		t.Errorf("the saved file is mode %v; a document the user saved should be readable", st.Mode())
	}
}

// TestURIRefIsRefusedOnDesktopNotMisexecuted keeps the cross-platform boundary honest: a URI ref with no
// backend that understands it produces a NAMED error, rather than being fed to os.Open — which would
// fail with a confusing filesystem error, or worse, succeed against a path that happens to look like a
// URI.
func TestURIRefIsRefusedOnDesktopNotMisexecuted(t *testing.T) {
	u := URIRef("content://com.android.providers.downloads/document/1234")
	if _, err := ReadFile(u); !errors.Is(err, ErrUnsupportedRef) {
		t.Errorf("reading a URI ref on desktop gave %v, want ErrUnsupportedRef", err)
	}
	if err := WriteFile(u, func(io.Writer) error { return nil }); !errors.Is(err, ErrUnsupportedRef) {
		t.Errorf("writing a URI ref on desktop gave %v, want ErrUnsupportedRef", err)
	}
}

// TestSetFileIOInstallsABackend covers the seam's extension point: this is how a mobile backend teaches
// the toolkit to open its platform's locations, and how a test can drive the I/O without a filesystem.
func TestSetFileIOInstallsABackend(t *testing.T) {
	u := URIRef("content://example/doc/42")
	wrote := ""
	SetFileIO(FileIO{
		Read: func(FileRef) ([]byte, error) { return []byte("from the backend"), nil },
		Write: func(_ FileRef, write func(io.Writer) error) error {
			var buf bytes.Buffer
			if err := write(&buf); err != nil {
				return err
			}
			wrote = buf.String()
			return nil
		},
	})
	t.Cleanup(func() { SetFileIO(FileIO{}) })

	data, err := ReadFile(u)
	if err != nil {
		t.Fatalf("a backend was installed and reading a URI still failed: %v", err)
	}
	if string(data) != "from the backend" {
		t.Errorf("read %q", string(data))
	}
	if err := WriteFile(u, func(w io.Writer) error {
		_, err := w.Write([]byte("tee"))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if wrote != "tee" {
		t.Errorf("the backend's writer received %q", wrote)
	}
	// Restoring clears it: the default refuses a URI again. (The cleanup would do this at the end of
	// the test; restoring explicitly is what makes the assertion below about the DEFAULT rather than
	// about the backend still being installed — which is what my first version asserted.)
	SetFileIO(FileIO{})
	if _, err := ReadFile(u); !errors.Is(err, ErrUnsupportedRef) {
		t.Errorf("after restoring the default, a URI read gave %v, want ErrUnsupportedRef", err)
	}
}

// TestRefForTagsLocationsWithoutGuessingWrong pins the one place a location string is interpreted, and
// the cases that would break a naive "contains ://" test.
func TestRefForTagsLocationsWithoutGuessingWrong(t *testing.T) {
	for _, c := range []struct {
		loc  string
		want string
		why  string
	}{
		{"/tmp/a.png", "path", "an absolute POSIX path"},
		{"relative/b.png", "path", "a relative path"},
		{`C:\Users\u\c.png`, "path", "a Windows path: a drive letter is not a scheme, and there is no ://"},
		{`\\server\share\d.png`, "path", "a UNC path"},
		{"content://com.android/doc/1", "uri", "an Android document URI"},
		{"file:///tmp/e.png", "uri", "a file URI: still a URI, which the backend decides how to open"},
		{"", "", "nothing at all"},
	} {
		got := RefFor(c.loc).Scheme()
		if got != c.want {
			t.Errorf("RefFor(%q).Scheme() = %q, want %q (%s)", c.loc, got, c.want, c.why)
		}
	}
}

// TestWriteFileRefusesTheEmptyCases keeps the failure modes from being silent.
func TestWriteFileRefusesTheEmptyCases(t *testing.T) {
	if err := WriteFile(FileRef{}, func(io.Writer) error { return nil }); err == nil {
		t.Error("writing to an invalid ref succeeded")
	}
	if err := WriteFile(PathRef(filepath.Join(t.TempDir(), "x")), nil); err == nil {
		t.Error("writing with no writer function succeeded")
	}
	if _, err := ReadFile(FileRef{}); err == nil {
		t.Error("reading an invalid ref succeeded")
	}
}

// TestSerializeRoundTripsBothKindsOfLocation is what a recents list needs: a location written down and
// read back unchanged. The scheme is part of the encoding, so a URI cannot come back as a path.
func TestSerializeRoundTripsBothKindsOfLocation(t *testing.T) {
	for _, ref := range []FileRef{
		PathRef("/home/u/photos/a.png"),
		URIRef("content://com.android.providers.downloads/document/1234"),
		PathRef(`C:\Users\u\a.png`),
	} {
		back := DeserializeRef(ref.Serialize())
		if back != ref {
			t.Errorf("%v serialized to %q and came back as %v", ref, ref.Serialize(), back)
		}
		if back.Scheme() != ref.Scheme() {
			t.Errorf("%v came back as a %s location, want %s", ref, back.Scheme(), ref.Scheme())
		}
	}
	// The zero ref encodes as nothing and reads back as nothing.
	var zero FileRef
	if zero.Serialize() != "" {
		t.Errorf("the zero ref serialized to %q", zero.Serialize())
	}
	// A line this build cannot read loses that line rather than failing: a recents list must load.
	for _, junk := range []string{"", "garbage", "ftp://host/x", ":no-scheme"} {
		if got := DeserializeRef(junk); got.Valid() {
			t.Errorf("DeserializeRef(%q) produced %v, want the zero ref", junk, got)
		}
	}
}

// TestAppDataDirIsCreatedAndReplaceable: the app needs somewhere to write its own files, and the
// platform is what decides where that is.
func TestAppDataDirIsCreatedAndReplaceable(t *testing.T) {
	tmp := t.TempDir()
	SetAppDataDir(func(app string) (FileRef, error) {
		dir := filepath.Join(tmp, app)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return FileRef{}, err
		}
		return PathRef(dir), nil
	})
	t.Cleanup(func() { SetAppDataDir(nil) })

	ref, err := AppDataDir("media-editor")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Scheme() != "path" || !ref.Valid() {
		t.Fatalf("AppDataDir gave %v", ref)
	}
	if _, err := os.Stat(ref.DisplayPath()); err != nil {
		t.Errorf("the app data directory does not exist: %v", err)
	}
	// The default is restored, and it also works (a config directory plus the app's name).
	SetAppDataDir(nil)
	if _, err := AppDataDir(""); err == nil {
		t.Error("AppDataDir with no app name succeeded")
	}
	ref, err = AppDataDir("meui-appdata-test")
	if err != nil {
		t.Skipf("no writable config directory on this machine: %v", err)
	}
	if !strings.HasSuffix(ref.DisplayPath(), "meui-appdata-test") {
		t.Errorf("the default app data directory is %q, want it to end with the app name",
			ref.DisplayPath())
	}
}
