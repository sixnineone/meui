package widgets

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
)

// MarkdownEditorMode selects which pane(s) the editor shows.
type MarkdownEditorMode int

const (
	// MarkdownSplit shows the source and the preview side by side.
	MarkdownSplit MarkdownEditorMode = iota
	// MarkdownSourceOnly shows only the editor pane.
	MarkdownSourceOnly
	// MarkdownPreviewOnly shows only the rendered pane.
	MarkdownPreviewOnly
)

// MarkdownEditor is the two-pane markdown editor: a CodeEditor source pane and
// a MarkdownView preview pane over the same document.
//
// The two clipboards fall out of the composition: copying in the source pane
// copies its own text, which *is* the markdown source, while copying in the
// preview uses the browser-grade selection in markdown_selection.go and yields
// plain text. Both panes are exposed (Source/Preview) so an app can configure
// them; the splitter is the toolkit's SplitView, so the panes can be resized
// and the ratio is remembered across mode switches.
type MarkdownEditor struct {
	meui.BaseContainer
	split   *SplitView
	source  *CodeEditor
	preview *MarkdownView

	mode       MarkdownEditorMode
	splitRatio float32
	syncing    bool

	// OnTextChanged fires after the source pane edits the document.
	OnTextChanged func(text string)
}

// NewMarkdownEditor builds the editor with both panes over an empty document.
func NewMarkdownEditor() *MarkdownEditor {
	m := &MarkdownEditor{
		BaseContainer: meui.BaseContainer{BaseWidget: *meui.NewBaseWidget()},
		splitRatio:    0.5,
	}
	m.source = NewCodeEditor()
	m.source.SetLanguage("markdown")
	m.source.OnTextChanged = func(text string) {
		if m.syncing {
			return
		}
		m.syncing = true
		m.preview.SetText(text)
		m.syncing = false
		m.MarkForRedraw()
		if m.OnTextChanged != nil {
			m.OnTextChanged(text)
		}
	}
	m.preview = NewMarkdownView()
	m.split = NewSplitView()
	m.split.SetPanels(m.source, m.preview)
	m.AddChild(m.split)
	m.applyMode()
	return m
}

// Source returns the editor pane.
func (m *MarkdownEditor) Source() *CodeEditor { return m.source }

// Preview returns the rendered pane.
func (m *MarkdownEditor) Preview() *MarkdownView { return m.preview }

// Split returns the splitter hosting the two panes.
func (m *MarkdownEditor) Split() *SplitView { return m.split }

// SetText sets the document in both panes without firing OnTextChanged.
func (m *MarkdownEditor) SetText(text string) *MarkdownEditor {
	m.syncing = true
	m.source.SetText(text)
	m.preview.SetText(text)
	m.syncing = false
	m.MarkForRedraw()
	return m
}

// Text returns the markdown source.
func (m *MarkdownEditor) Text() string { return m.source.Text() }

// SetMode selects split, source-only or preview-only.
func (m *MarkdownEditor) SetMode(mode MarkdownEditorMode) *MarkdownEditor {
	m.mode = mode
	m.applyMode()
	m.MarkForRedraw()
	return m
}

// Mode returns the current mode.
func (m *MarkdownEditor) Mode() MarkdownEditorMode { return m.mode }

// SetReadOnly makes both panes read-only; selection and copy keep working in
// both (the preview is always read-only).
func (m *MarkdownEditor) SetReadOnly(readOnly bool) *MarkdownEditor {
	m.source.ReadOnly = readOnly
	m.source.MarkForRedraw()
	return m
}

// IsReadOnly reports the source pane's read-only state.
func (m *MarkdownEditor) IsReadOnly() bool { return m.source.ReadOnly }

// SetSplitRatio positions the splitter (0 = preview only, 1 = source only).
func (m *MarkdownEditor) SetSplitRatio(ratio float32) *MarkdownEditor {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	m.splitRatio = ratio
	if m.mode == MarkdownSplit {
		m.split.SetSplitPosition(ratio)
	}
	m.MarkForRedraw()
	return m
}

// SplitRatio returns the remembered splitter position.
func (m *MarkdownEditor) SplitRatio() float32 { return m.splitRatio }

// applyMode shows the panes the mode wants and gives the visible one the frame.
func (m *MarkdownEditor) applyMode() {
	switch m.mode {
	case MarkdownSourceOnly:
		m.source.SetVisible(true)
		m.preview.SetVisible(false)
		m.split.MinSize1, m.split.MinSize2 = 0, 0
		m.split.SetHandleSize(1)
		m.split.SetSplitPosition(1)
	case MarkdownPreviewOnly:
		m.source.SetVisible(false)
		m.preview.SetVisible(true)
		m.split.MinSize1, m.split.MinSize2 = 0, 0
		m.split.SetHandleSize(1)
		m.split.SetSplitPosition(0)
	default:
		m.source.SetVisible(true)
		m.preview.SetVisible(true)
		m.split.MinSize1, m.split.MinSize2 = 120, 120
		m.split.SetHandleSize(6)
		m.split.SetSplitPosition(m.splitRatio)
	}
	m.split.SetNeedsLayout()
	// The editor itself must re-lay out too: ValidateLayout owns forwarding the
	// frame to the splitter, and it early-returns when the editor is clean.
	m.SetNeedsLayout()
}

// SetFrameRect forwards to the splitter so the panes follow the editor.
func (m *MarkdownEditor) SetFrameRect(rect graphics.Rect) {
	m.BaseWidget.SetFrameRect(rect)
	m.split.SetFrameRect(localRect(rect))
	m.SetNeedsLayout()
}

func (m *MarkdownEditor) ValidateLayout() {
	if !m.NeedsLayout() {
		return
	}
	m.split.SetFrameRect(localRect(m.FrameRect()))
	m.split.ValidateLayout()
	m.ClearNeedsLayout()
}

// localRect turns a widget's own (parent-relative) frame into the frame its
// children must use. A child's FrameRect is parent-relative, so copying the
// parent's origin onto a child makes every ancestor origin count twice:
// PointFromRoot subtracts each link of the chain, and a press lands short by
// exactly the editor's origin (clicking line 23 put the caret near line 18).
func localRect(rect graphics.Rect) graphics.Rect {
	return graphics.NewRect(0, 0, rect.Width, rect.Height)
}

// Paint draws the splitter (which paints both panes).
func (m *MarkdownEditor) Paint(canvas *graphics.Canvas, rect graphics.Rect) {
	if !m.Visible() {
		return
	}
	m.ValidateLayout()
	m.split.Paint(canvas, rect)
}

// PanelAt continues the hit test into the splitter and its panes.
//
// It must be explicit: BaseContainer provides DefaultPanelAt but not PanelAt,
// so a container that does not override it inherits BaseWidget.PanelAt, which
// returns the widget itself. That would swallow every press in the editor (no
// caret in the source pane, no selection in the preview) and, because the
// editor implements no FocusHandler, keyboard focus would never reach the
// panes either.
func (m *MarkdownEditor) PanelAt(x, y float32) meui.Widget {
	return m.DefaultPanelAt(x, y)
}

// Focusable reports that the editor can take keyboard focus: the source pane
// handles the keys when it has focus.
func (m *MarkdownEditor) Focusable() bool { return m.Enabled() && m.Visible() }

// SetBoundProperty applies the QML-bound properties registered for this widget
// (see qml/loader.go): text, readOnly, mode and splitRatio.
func (m *MarkdownEditor) SetBoundProperty(propName string, value interface{}) {
	switch propName {
	case "text":
		if s, ok := value.(string); ok {
			m.SetText(s)
		}
	case "readOnly":
		if b, ok := value.(bool); ok {
			m.SetReadOnly(b)
		}
	case "mode":
		s, ok := value.(string)
		if !ok {
			return
		}
		switch s {
		case "source":
			m.SetMode(MarkdownSourceOnly)
		case "preview":
			m.SetMode(MarkdownPreviewOnly)
		default:
			m.SetMode(MarkdownSplit)
		}
	case "splitRatio":
		switch v := value.(type) {
		case float64:
			m.SetSplitRatio(float32(v))
		case float32:
			m.SetSplitRatio(v)
		case int:
			m.SetSplitRatio(float32(v))
		}
	}
}

var _ meui.Widget = (*MarkdownEditor)(nil)
