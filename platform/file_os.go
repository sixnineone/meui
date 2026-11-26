package platform

// The DEFAULT file I/O: `os`, for path locations. A desktop build installs no backend and gets this; a
// mobile backend replaces the Read/Write pair for the locations `os` cannot open.
//
// It lives in its own file because it is not a backend — it is the fallback that makes the seam usable
// with nothing installed, and it is the piece that owns ATOMIC SAVE.

import (
	"io"
	"os"
	"path/filepath"
)

func osReadFile(ref FileRef) ([]byte, error) {
	if ref.kind != kindPath {
		return nil, ErrUnsupportedRef
	}
	return os.ReadFile(ref.loc)
}

// osWriteFile writes a path atomically: a temporary file IN THE SAME DIRECTORY, fsynced, then renamed
// over the target.
//
// Same directory is not incidental — a rename is atomic only within one filesystem, so a temp file in
// the system temp directory would turn the rename into a copy and lose the guarantee on the one
// occasion it matters. The fsync before the rename is what keeps a crash from leaving a correctly
// named but empty file, which is worse than an obviously missing one.
//
// A failure at any point removes the temporary file, so a failed save leaves the directory exactly as
// it was and the previous version of the document untouched.
func osWriteFile(ref FileRef, write func(io.Writer) error) error {
	return osWriteFileOpt(ref, write, true)
}

// osWriteFileQuick is osWriteFile without the fsync, for state that does not need to survive a power loss.
func osWriteFileQuick(ref FileRef, write func(io.Writer) error) error {
	return osWriteFileOpt(ref, write, false)
}

func osWriteFileOpt(ref FileRef, write func(io.Writer) error, durable bool) error {
	if ref.kind != kindPath {
		return ErrUnsupportedRef
	}
	dir := filepath.Dir(ref.loc)
	tmp, err := os.CreateTemp(dir, ".meui-save-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Until the rename succeeds, the temporary file is ours to clean up.
	committed := false
	defer func() {
		if !committed {
			tmp.Close()
			os.Remove(tmpName)
		}
	}()

	if err := write(tmp); err != nil {
		return err
	}
	if durable {
		// The fsync is what makes the write survive a power loss, and it costs a disk round trip. See
		// WriteFileQuick for when that is not worth paying.
		if err := tmp.Sync(); err != nil {
			return err
		}
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// Permissions come from the temporary file, which CreateTemp makes 0600; a document the user saved
	// should be readable by them the way any other file they created is.
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmpName, ref.loc); err != nil {
		return err
	}
	committed = true
	return nil
}
