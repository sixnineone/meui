package widgets

import (
	"github.com/sixnineone/meui/platform"
)

// The toolkit's file pickers. They return an opaque platform.FileRef rather than a path: a path is a
// desktop-only idea (Android's dialog answers with `content://` URIs), and the I/O belongs to the
// platform seam. An app passes the ref back to platform.ReadFile/WriteFile, or to a media helper.

// OpenFile asks for one file.
func OpenFile(title string) (platform.FileRef, bool) {
	dlg := platform.NewOpenDialog()
	dlg.SetCanChooseFiles(true)
	dlg.SetCanChooseDirectories(false)
	dlg.SetAllowsMultipleSelection(false)
	if dlg.RunModal() {
		if ref := dlg.Ref(); ref.Valid() {
			return ref, true
		}
	}
	return platform.FileRef{}, false
}

// OpenFiles asks for any number of files.
func OpenFiles(title string) ([]platform.FileRef, bool) {
	dlg := platform.NewOpenDialog()
	dlg.SetCanChooseFiles(true)
	dlg.SetCanChooseDirectories(false)
	dlg.SetAllowsMultipleSelection(true)
	if dlg.RunModal() {
		return dlg.Refs(), true
	}
	return nil, false
}

// OpenDirectory asks for a directory.
func OpenDirectory(title string) (platform.FileRef, bool) {
	dlg := platform.NewOpenDialog()
	dlg.SetCanChooseFiles(false)
	dlg.SetCanChooseDirectories(true)
	dlg.SetAllowsMultipleSelection(false)
	if dlg.RunModal() {
		if ref := dlg.Ref(); ref.Valid() {
			return ref, true
		}
	}
	return platform.FileRef{}, false
}

// SaveFile asks where to save one file.
func SaveFile(title string) (platform.FileRef, bool) {
	dlg := platform.NewSaveDialog()
	if dlg.RunModal() {
		if ref := dlg.Ref(); ref.Valid() {
			return ref, true
		}
	}
	return platform.FileRef{}, false
}
