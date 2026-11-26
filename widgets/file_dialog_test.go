package widgets

import (
	"testing"

	"github.com/sixnineone/meui/platform"
)

// TestFilePickersReturnLocationsNotPaths is the wiring the file lifecycle needs: the pickers hand back
// an opaque platform.FileRef, so an app asks the platform seam to do the I/O instead of receiving a
// string it would have to treat as a path — the assumption that would work on desktop and be dead on
// Android, where the same dialog answers with `content://` URIs.
func TestFilePickersReturnLocationsNotPaths(t *testing.T) {
	platform.SetDialogRunner(func(platform.DialogRequest) ([]platform.FileRef, bool) {
		return []platform.FileRef{platform.PathRef("/tmp/chosen.png")}, true
	})
	t.Cleanup(func() { platform.SetDialogRunner(nil) })

	ref, ok := OpenFile("Open")
	if !ok {
		t.Fatal("OpenFile reported cancellation although the runner answered")
	}
	if ref.DisplayName() != "chosen.png" {
		t.Errorf("OpenFile returned %v, want the runner's location", ref)
	}
	if ref.Scheme() != "path" {
		t.Errorf("OpenFile returned a %s ref, want a path from a path location", ref.Scheme())
	}

	if ref, ok := SaveFile("Save"); !ok || ref.DisplayName() != "chosen.png" {
		t.Errorf("SaveFile returned (%v, %v)", ref, ok)
	}
	if refs, ok := OpenFiles("Open many"); !ok || len(refs) != 1 {
		t.Errorf("OpenFiles returned (%v, %v)", refs, ok)
	}
	if ref, ok := OpenDirectory("Open folder"); !ok || ref.Scheme() != "path" {
		t.Errorf("OpenDirectory returned (%v, %v)", ref, ok)
	}
}

// TestAPickerCancelIsNotALocation covers the cancel path: a cancelled picker must NOT hand back the
// previous answer, or an app would re-open or over-write the file the user just declined.
func TestAPickerCancelIsNotALocation(t *testing.T) {
	platform.SetDialogRunner(func(platform.DialogRequest) ([]platform.FileRef, bool) {
		return []platform.FileRef{platform.PathRef("/tmp/first.png")}, true
	})
	t.Cleanup(func() { platform.SetDialogRunner(nil) })

	if _, ok := OpenFile("Open"); !ok {
		t.Fatal("the first pick failed")
	}
	platform.SetDialogRunner(func(platform.DialogRequest) ([]platform.FileRef, bool) { return nil, false })
	if ref, ok := OpenFile("Open"); ok || ref.Valid() {
		t.Errorf("a cancelled pick returned (%v, %v); want no location", ref, ok)
	}

	// And with no backend at all, which is what a build without a dialog implementation gets.
	platform.SetDialogRunner(nil)
	if ref, ok := SaveFile("Save"); ok || ref.Valid() {
		t.Errorf("with no backend SaveFile returned (%v, %v)", ref, ok)
	}
}
