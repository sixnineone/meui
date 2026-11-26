package platform

import (
	"testing"

	"github.com/sixnineone/meui/graphics"
)

func TestKeyCodeConstants(t *testing.T) {
	if KeyA == KeyB {
		t.Error("KeyA == KeyB")
	}
	if KeyReturn == KeyNumPadEnter {
		t.Error("KeyReturn == KeyNumPadEnter (should be distinct)")
	}
	if KeyHome == KeyEnd {
		t.Error("KeyHome == KeyEnd")
	}
	if KeyF1 == KeyF9 {
		t.Error("KeyF1 == KeyF9")
	}
}

func TestClipboardRoundTrip(t *testing.T) {
	GlobalClipboard.SetText("foo")
	if got := GlobalClipboard.GetText(); got != "foo" {
		t.Errorf("GetText = %q, want %q (memory mirror)", got, "foo")
	}
	GlobalClipboard.SetText("")
	if got := GlobalClipboard.GetText(); got != "" {
		t.Errorf("after clear: GetText = %q, want \"\"", got)
	}
}

// TestDialogsWithoutABackendReportCancelled is the no-runner contract: with no
// backend installed, a dialog cannot be shown, so RunModal reports false and the
// result stays empty. That is what the stub always did, and it is still the right
// answer — it is now just one case of the seam rather than the only behaviour.
func TestDialogsWithoutABackendReportCancelled(t *testing.T) {
	SetDialogRunner(nil)
	t.Cleanup(func() { SetDialogRunner(nil) })

	d := NewOpenDialog()
	d.SetCanChooseFiles(true)
	d.SetCanChooseDirectories(false)
	d.SetAllowsMultipleSelection(true)
	d.SetAllowedExtensions("txt")
	d.SetInitialDirectory("/tmp")
	d.SetTitle("x")
	if d.RunModal() {
		t.Error("OpenDialog.RunModal should return false with no backend")
	}
	if len(d.Refs()) != 0 {
		t.Error("OpenDialog.Paths should be empty with no backend")
	}

	s := NewSaveDialog()
	s.SetAllowedExtensions("txt")
	s.SetInitialDirectory("/tmp")
	s.SetTitle("x")
	if s.RunModal() {
		t.Error("SaveDialog.RunModal should return false with no backend")
	}
	if s.Ref().Valid() {
		t.Error("SaveDialog.Path should be empty with no backend")
	}
}

// TestOpenDialogCarriesItsConfigurationToTheBackend is the other half: whatever
// the widget configured must arrive at the runner intact, and the runner's paths
// must come back out. Without this, a dialog could show the wrong mode (a folder
// picker for a file request) and every caller would still look correct.
func TestOpenDialogCarriesItsConfigurationToTheBackend(t *testing.T) {
	var got DialogRequest
	calls := 0
	SetDialogRunner(func(req DialogRequest) ([]FileRef, bool) {
		calls++
		got = req
		return []FileRef{PathRef("/tmp/a.txt"), PathRef("/tmp/b.txt")}, true
	})
	t.Cleanup(func() { SetDialogRunner(nil) })

	d := NewOpenDialog()
	d.SetCanChooseFiles(true)
	d.SetCanChooseDirectories(false)
	d.SetAllowsMultipleSelection(true)
	d.SetAllowedExtensions("txt", "go")
	d.SetInitialDirectory("/tmp")
	d.SetTitle("Pick files")

	if !d.RunModal() {
		t.Fatal("RunModal = false although the runner reported ok")
	}
	if calls != 1 {
		t.Errorf("runner called %d times, want 1", calls)
	}
	if refs := d.Refs(); len(refs) != 2 || refs[0].DisplayPath() != "/tmp/a.txt" || refs[1].DisplayPath() != "/tmp/b.txt" {
		t.Errorf("Refs() = %v, want the runner's two locations", refs)
	}
	if got.Title != "Pick files" {
		t.Errorf("Title = %q, want %q", got.Title, "Pick files")
	}
	if got.InitialDirectory != "/tmp" {
		t.Errorf("InitialDirectory = %q, want %q", got.InitialDirectory, "/tmp")
	}
	if !got.AllowMany {
		t.Error("AllowMany = false, want true (SetAllowsMultipleSelection(true))")
	}
	if got.ChooseDirectories {
		t.Error("ChooseDirectories = true, want false (SetCanChooseDirectories(false))")
	}
	if got.Save {
		t.Error("Save = true, want false for an open dialog")
	}
	if len(got.Extensions) != 2 || got.Extensions[0] != "txt" || got.Extensions[1] != "go" {
		t.Errorf("Extensions = %v, want [txt go]", got.Extensions)
	}
}

// TestSaveDialogAsksToSaveAndReportsOnePath: the save dialog must set Save (so
// the backend shows a save picker rather than an open one) and take the first
// path, because a save target is a single file.
func TestSaveDialogAsksToSaveAndReportsOnePath(t *testing.T) {
	var got DialogRequest
	SetDialogRunner(func(req DialogRequest) ([]FileRef, bool) {
		got = req
		return []FileRef{PathRef("/tmp/out.png")}, true
	})
	t.Cleanup(func() { SetDialogRunner(nil) })

	s := NewSaveDialog()
	s.SetAllowedExtensions("png")
	s.SetInitialDirectory("/tmp")
	s.SetTitle("Save as")
	if !s.RunModal() {
		t.Fatal("RunModal = false although the runner reported ok")
	}
	if !got.Save {
		t.Error("Save = false, want true for a save dialog")
	}
	if got := s.Ref().DisplayPath(); got != "/tmp/out.png" {
		t.Errorf("Ref().DisplayPath() = %q, want /tmp/out.png", got)
	}
}

// TestCancellingAfterASelectionClearsTheResult: a dialog object can be reused, so
// a cancel must not leave the previous answer behind for the caller to read.
func TestCancellingAfterASelectionClearsTheResult(t *testing.T) {
	SetDialogRunner(func(DialogRequest) ([]FileRef, bool) { return []FileRef{PathRef("/tmp/first.txt")}, true })
	t.Cleanup(func() { SetDialogRunner(nil) })

	d := NewOpenDialog()
	if !d.RunModal() {
		t.Fatal("first RunModal = false, want true")
	}
	if len(d.Refs()) != 1 {
		t.Fatalf("Refs() = %v, want one location", d.Refs())
	}

	SetDialogRunner(func(DialogRequest) ([]FileRef, bool) { return nil, false })
	if d.RunModal() {
		t.Error("second RunModal = true, want false after a cancel")
	}
	if len(d.Refs()) != 0 {
		t.Errorf("Refs() = %v after a cancel, want empty — a stale location would be "+
			"read as a fresh selection", d.Refs())
	}
}

func TestBestDisplayForRect(t *testing.T) {
	d := BestDisplayForRect(graphics.NewRect(0, 0, 800, 600))
	if d == nil {
		t.Error("BestDisplayForRect should return non-nil")
	}
}

func TestCursorStubs(t *testing.T) {
	if PointingCursor() == nil || ResizeHorizontalCursor() == nil {
		t.Error("cursor constants nil")
	}
	if PointingCursor().Name() != "pointing" {
		t.Errorf("PointingCursor.Name = %q", PointingCursor().Name())
	}
	var nilCursor *Cursor
	if nilCursor.Name() != "" {
		t.Error("nil Cursor.Name() should return empty")
	}
}

// TestEventModelRoundTrips pins that the event model the toolkit consumes
// carries the fields the backend fills in.
func TestEventModelRoundTrips(t *testing.T) {
	ev := &Event{
		Type:   EventKeyDown,
		Key:    KeyReturn,
		Mods:   ModShift,
		Rune:   'a',
		Pos:    graphics.Point{X: 3, Y: 4},
		Button: 1,
	}
	if ev.Type != EventKeyDown || ev.Key != KeyReturn || ev.Mods != ModShift {
		t.Fatalf("event fields did not round-trip: %+v", ev)
	}
	if ev.Pos.X != 3 || ev.Pos.Y != 4 {
		t.Fatalf("event position is not a seam Point: %+v", ev.Pos)
	}
}
