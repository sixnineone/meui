package platform

// Native file dialogs.
//
// The seam owns the REQUEST and the RESULT; the backend owns showing the
// dialog. A backend installs one function with SetDialogRunner — the SDL3
// adapter installs SDL_ShowOpenFileDialog/SDL_ShowSaveFileDialog — and the
// toolkit's widgets.OpenFile/SaveFile keep calling NewOpenDialog/RunModal, so no
// widget learns which OS is underneath.
//
// This used to be a pure stub whose RunModal always reported false (REVIEW.md
// §8). Callers are unchanged: the stub behaviour is now simply what happens when
// no backend installs a runner, or when the user cancels.
//
// The result is a FileRef rather than a path ON PURPOSE: a path is a desktop-only idea, and the same
// dialog returns `content://` URIs on Android. See file.go.

// DialogRequest is everything a backend needs to show a dialog, in neutral
// terms: no SDL, no JNI, no Objective-C.
type DialogRequest struct {
	Title             string
	Extensions        []string // bare extensions, e.g. "png", "go"; empty means all files
	InitialDirectory  string
	AllowMany         bool // open dialogs only
	ChooseDirectories bool // open dialogs only
	Save              bool
}

// DialogRunner shows the OS dialog and returns the chosen locations as FileRefs.
//
// ok is false both when the user cancelled and when the platform could not show
// a dialog at all. The two are deliberately the same answer: an app can do
// nothing different about either, and distinguishing them would leak the
// platform back into the API.
type DialogRunner func(DialogRequest) (refs []FileRef, ok bool)

var dialogRunner DialogRunner

// SetDialogRunner installs the backend's dialog implementation. Nil restores the
// no-dialog behaviour, where RunModal reports false.
func SetDialogRunner(fn DialogRunner) { dialogRunner = fn }

// OpenDialog collects what to ask for, then holds the answer.
type OpenDialog struct {
	title      string
	exts       []string
	initialDir string
	files      bool
	dirs       bool
	many       bool
	refs       []FileRef
}

func (d *OpenDialog) SetCanChooseFiles(b bool)          { d.files = b }
func (d *OpenDialog) SetCanChooseDirectories(b bool)    { d.dirs = b }
func (d *OpenDialog) SetAllowsMultipleSelection(b bool) { d.many = b }
func (d *OpenDialog) SetAllowedExtensions(e ...string)  { d.exts = e }
func (d *OpenDialog) SetInitialDirectory(dir string)    { d.initialDir = dir }
func (d *OpenDialog) SetTitle(t string)                 { d.title = t }

// RunModal shows the dialog and blocks until it is answered. It reports false
// when there is no backend, when the user cancelled, or when the platform could
// not show a dialog.
func (d *OpenDialog) RunModal() bool {
	if dialogRunner == nil {
		d.refs = nil
		return false
	}
	refs, ok := dialogRunner(DialogRequest{
		Title:             d.title,
		Extensions:        d.exts,
		InitialDirectory:  d.initialDir,
		AllowMany:         d.many,
		ChooseDirectories: d.dirs,
	})
	if !ok || len(refs) == 0 {
		d.refs = nil
		return false
	}
	d.refs = refs
	return true
}

// Refs returns the chosen locations; empty after a cancelled or failed dialog.
//
// There is deliberately no Paths() accessor beside it. A path is a desktop-only idea — the platform may
// have handed back a `content://` URI — so the app gets the opaque ref and asks the seam to do the I/O.
func (d *OpenDialog) Refs() []FileRef { return d.refs }

// Ref returns the first chosen location, for the single-selection case.
func (d *OpenDialog) Ref() FileRef {
	if len(d.refs) == 0 {
		return FileRef{}
	}
	return d.refs[0]
}

// SaveDialog collects what to ask for, then holds the chosen path.
type SaveDialog struct {
	title      string
	exts       []string
	initialDir string
	ref        FileRef
}

func (d *SaveDialog) SetAllowedExtensions(e ...string) { d.exts = e }
func (d *SaveDialog) SetInitialDirectory(dir string)   { d.initialDir = dir }
func (d *SaveDialog) SetTitle(t string)                { d.title = t }

// RunModal shows the save dialog and blocks until it is answered.
func (d *SaveDialog) RunModal() bool {
	if dialogRunner == nil {
		d.ref = FileRef{}
		return false
	}
	refs, ok := dialogRunner(DialogRequest{
		Title:            d.title,
		Extensions:       d.exts,
		InitialDirectory: d.initialDir,
		Save:             true,
	})
	if !ok || len(refs) == 0 {
		d.ref = FileRef{}
		return false
	}
	d.ref = refs[0]
	return true
}

// Ref returns the chosen location; invalid after a cancelled or failed dialog.
func (d *SaveDialog) Ref() FileRef { return d.ref }

func NewOpenDialog() OpenDialog { return OpenDialog{} }
func NewSaveDialog() SaveDialog { return SaveDialog{} }
