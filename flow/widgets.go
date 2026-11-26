package flow

import (
	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/widgets"
)

// ── TextField wrapper ─────────────────────────────────

type TextField struct {
	W
	Text        string                   `qml:"text"`
	Placeholder string                   `qml:"placeholder"`
	ReadOnly    bool                     `qml:"readOnly"`
	MaxLength   int                      `qml:"maxLength"`
	OnCreated   func(*widgets.TextField) `qml:"-"`
}

func (t *TextField) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewTextField()
	ApplyLayout(w, &t.W)
	if t.OnCreated != nil {
		t.OnCreated(w)
	}
	registerID(ctx, w, t.ID)
	return w
}

// ── TextArea wrapper ──────────────────────────────────

type TextArea struct {
	W
	Text        string                  `qml:"text"`
	Placeholder string                  `qml:"placeholder"`
	ReadOnly    bool                    `qml:"readOnly"`
	Wrap        bool                    `qml:"wrap"`
	WordWrap    bool                    `qml:"wordWrap"`
	MaxLength   int                     `qml:"maxLength"`
	OnCreated   func(*widgets.TextArea) `qml:"-"`
}

func (t *TextArea) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewTextArea()
	ApplyLayout(w, &t.W)
	if t.OnCreated != nil {
		t.OnCreated(w)
	}
	registerID(ctx, w, t.ID)
	return w
}

// ── Slider wrapper ────────────────────────────────────

type Slider struct {
	W
	Value       float32               `qml:"value"`
	Minimum     float32               `qml:"minimum"`
	Maximum     float32               `qml:"maximum"`
	StepSize    float32               `qml:"stepSize"`
	Orientation string                `qml:"orientation"`
	OnCreated   func(*widgets.Slider) `qml:"-"`
}

func (s *Slider) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewSlider()
	ApplyLayout(w, &s.W)
	if s.OnCreated != nil {
		s.OnCreated(w)
	}
	registerID(ctx, w, s.ID)
	return w
}

// ── ComboBox wrapper ──────────────────────────────────

type ComboBox struct {
	W
	Options       []string                `qml:"options"`
	SelectedIndex int                     `qml:"currentIndex"`
	Editable      bool                    `qml:"editable"`
	Placeholder   string                  `qml:"placeholder"`
	OnCreated     func(*widgets.ComboBox) `qml:"-"`
}

func (c *ComboBox) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewComboBox()
	ApplyLayout(w, &c.W)
	if c.OnCreated != nil {
		c.OnCreated(w)
	}
	registerID(ctx, w, c.ID)
	return w
}

// ── RadioButton wrapper ───────────────────────────────

type RadioButton struct {
	W
	Text      string                     `qml:"text"`
	GroupID   string                     `qml:"groupID"`
	Selected  bool                       `qml:"selected"`
	OnCreated func(*widgets.RadioButton) `qml:"-"`
}

func (r *RadioButton) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewRadioButton()
	ApplyLayout(w, &r.W)
	if r.OnCreated != nil {
		r.OnCreated(w)
	}
	registerID(ctx, w, r.ID)
	return w
}

// ── ListView wrapper ──────────────────────────────────

type ListView struct {
	W
	Items         []string                `qml:"model"`
	SelectedIndex int                     `qml:"currentIndex"`
	OnCreated     func(*widgets.ListView) `qml:"-"`
}

func (l *ListView) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewListView()
	ApplyLayout(w, &l.W)
	if l.OnCreated != nil {
		l.OnCreated(w)
	}
	registerID(ctx, w, l.ID)
	return w
}

// ── ProgressBar wrapper ───────────────────────────────

type ProgressBar struct {
	W
	Value     float32                    `qml:"value"`
	Minimum   float32                    `qml:"minimum"`
	Maximum   float32                    `qml:"maximum"`
	OnCreated func(*widgets.ProgressBar) `qml:"-"`
}

func (p *ProgressBar) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewProgressBar()
	ApplyLayout(w, &p.W)
	if p.OnCreated != nil {
		p.OnCreated(w)
	}
	registerID(ctx, w, p.ID)
	return w
}

// ── ScrollBar wrapper ─────────────────────────────────

type ScrollBar struct {
	W
	Value       float32                  `qml:"value"`
	Minimum     float32                  `qml:"minimum"`
	Maximum     float32                  `qml:"maximum"`
	PageSize    float32                  `qml:"pageSize"`
	Orientation string                   `qml:"orientation"`
	OnCreated   func(*widgets.ScrollBar) `qml:"-"`
}

func (s *ScrollBar) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewScrollBar()
	ApplyLayout(w, &s.W)
	if s.OnCreated != nil {
		s.OnCreated(w)
	}
	registerID(ctx, w, s.ID)
	return w
}

// ── SpinBox wrapper ───────────────────────────────────

type SpinBox struct {
	W
	Value     float32                `qml:"value"`
	Minimum   float32                `qml:"minimum"`
	Maximum   float32                `qml:"maximum"`
	StepSize  float32                `qml:"stepSize"`
	Decimals  int                    `qml:"decimals"`
	OnCreated func(*widgets.SpinBox) `qml:"-"`
}

func (s *SpinBox) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewSpinBox()
	ApplyLayout(w, &s.W)
	if s.Value != 0 {
		w.SetValue(s.Value)
	}
	w.Minimum = s.Minimum
	w.Maximum = s.Maximum
	if s.StepSize != 0 {
		w.SetStep(s.StepSize)
	}
	if s.Decimals > 0 {
		w.SetDecimals(s.Decimals)
	}
	if s.OnCreated != nil {
		s.OnCreated(w)
	}
	registerID(ctx, w, s.ID)
	return w
}

// ── Switch wrapper ────────────────────────────────────

type Switch struct {
	W
	Text      string                `qml:"text"`
	Checked   bool                  `qml:"checked"`
	OnCreated func(*widgets.Switch) `qml:"-"`
}

func (s *Switch) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewSwitch()
	ApplyLayout(w, &s.W)
	if s.OnCreated != nil {
		s.OnCreated(w)
	}
	registerID(ctx, w, s.ID)
	return w
}

// ── GroupBox wrapper ──────────────────────────────────

type GroupBox struct {
	W
	Title     string                  `qml:"title"`
	OnCreated func(*widgets.GroupBox) `qml:"-"`
}

func (g *GroupBox) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewGroupBox()
	ApplyLayout(w, &g.W)
	if g.OnCreated != nil {
		g.OnCreated(w)
	}
	registerID(ctx, w, g.ID)
	return w
}

// ── StatusBar wrapper ─────────────────────────────────

type StatusBar struct {
	W
	Text      string                   `qml:"text"`
	OnCreated func(*widgets.StatusBar) `qml:"-"`
}

func (s *StatusBar) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewStatusBar()
	ApplyLayout(w, &s.W)
	if s.OnCreated != nil {
		s.OnCreated(w)
	}
	registerID(ctx, w, s.ID)
	return w
}

// ── Calendar wrapper ──────────────────────────────────

type Calendar struct {
	W
	OnCreated func(*widgets.Calendar) `qml:"-"`
}

func (c *Calendar) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewCalendar()
	ApplyLayout(w, &c.W)
	if c.OnCreated != nil {
		c.OnCreated(w)
	}
	registerID(ctx, w, c.ID)
	return w
}

// ── ColorPicker wrapper ───────────────────────────────

type ColorPicker struct {
	W
	OnCreated func(*widgets.ColorPicker) `qml:"-"`
}

func (c *ColorPicker) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewColorPicker()
	ApplyLayout(w, &c.W)
	if c.OnCreated != nil {
		c.OnCreated(w)
	}
	registerID(ctx, w, c.ID)
	return w
}

// ── DatePicker wrapper ────────────────────────────────

type DatePicker struct {
	W
	OnCreated func(*widgets.DatePicker) `qml:"-"`
}

func (d *DatePicker) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewDatePicker()
	ApplyLayout(w, &d.W)
	if d.OnCreated != nil {
		d.OnCreated(w)
	}
	registerID(ctx, w, d.ID)
	return w
}

// ── ColorField wrapper ────────────────────────────────

type ColorField struct {
	W
	OnCreated func(*widgets.ColorField) `qml:"-"`
}

func (c *ColorField) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewColorField()
	ApplyLayout(w, &c.W)
	if c.OnCreated != nil {
		c.OnCreated(w)
	}
	registerID(ctx, w, c.ID)
	return w
}

// ── TimePicker wrapper ────────────────────────────────

type TimePicker struct {
	W
	OnCreated func(*widgets.TimePicker) `qml:"-"`
}

func (t *TimePicker) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewTimePicker()
	ApplyLayout(w, &t.W)
	if t.OnCreated != nil {
		t.OnCreated(w)
	}
	registerID(ctx, w, t.ID)
	return w
}

// ── SplitView wrapper ─────────────────────────────────

type SplitView struct {
	W
	SplitPosition float32 `qml:"splitPosition"`
	HandleSize    float32 `qml:"handleSize"`
	Orientation   string  `qml:"orientation"`
	Children      []WidgetWrapper
	OnCreated     func(*widgets.SplitView) `qml:"-"`
}

func (s *SplitView) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewSplitView()
	ApplyLayout(w, &s.W)
	for _, child := range s.Children {
		w.AddChild(child.Build(ctx))
	}
	if s.OnCreated != nil {
		s.OnCreated(w)
	}
	registerID(ctx, w, s.ID)
	return w
}

// ── ToolTip wrapper ───────────────────────────────────

type ToolTip struct {
	W
	Text      string                 `qml:"text"`
	Delay     float32                `qml:"delay"`
	OnCreated func(*widgets.Tooltip) `qml:"-"`
}

func (t *ToolTip) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewTooltip()
	ApplyLayout(w, &t.W)
	if t.OnCreated != nil {
		t.OnCreated(w)
	}
	registerID(ctx, w, t.ID)
	return w
}

// ── Tree wrapper ──────────────────────────────────────

type Tree struct {
	W
	Model      []string            `qml:"model"`
	ItemHeight float32             `qml:"itemHeight"`
	OnCreated  func(*widgets.Tree) `qml:"-"`
}

func (t *Tree) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewTree()
	if len(t.Model) > 0 && w.Root != nil {
		for _, label := range t.Model {
			w.Root.Children = append(w.Root.Children, &widgets.TreeNode{Label: label, Expanded: true})
		}
	}
	if t.ItemHeight > 0 {
		w.ItemHeight = t.ItemHeight
	}
	ApplyLayout(w, &t.W)
	if t.OnCreated != nil {
		t.OnCreated(w)
	}
	registerID(ctx, w, t.ID)
	return w
}

// ── Table wrapper ─────────────────────────────────────

type Table struct {
	W
	Columns         []string             `qml:"columns"`
	ShowGrid        bool                 `qml:"showGrid"`
	Sortable        bool                 `qml:"sortable"`
	Resizable       bool                 `qml:"resizable"`
	Reorderable     bool                 `qml:"reorderable"`
	MultiSelect     bool                 `qml:"multiSelect"`
	Editable        bool                 `qml:"editable"`
	AlternatingRows bool                 `qml:"alternatingRows"`
	OnCreated       func(*widgets.Table) `qml:"-"`
}

func (t *Table) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewTable()
	ApplyLayout(w, &t.W)
	if t.OnCreated != nil {
		t.OnCreated(w)
	}
	registerID(ctx, w, t.ID)
	return w
}

// ── ScrollPane wrapper ────────────────────────────────

type ScrollPane struct {
	W
	ContentWidth              float32                   `qml:"contentWidth"`
	ContentHeight             float32                   `qml:"contentHeight"`
	HorizontalScrollBarPolicy string                    `qml:"horizontalScrollBarPolicy"`
	VerticalScrollBarPolicy   string                    `qml:"verticalScrollBarPolicy"`
	OnCreated                 func(*widgets.ScrollPane) `qml:"-"`
}

func (s *ScrollPane) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewScrollPane()
	ApplyLayout(w, &s.W)
	if s.ContentWidth > 0 {
		w.ContentWidth = s.ContentWidth
	}
	if s.ContentHeight > 0 {
		w.ContentHeight = s.ContentHeight
	}
	if s.HorizontalScrollBarPolicy != "" {
		w.ShowHScroll = s.HorizontalScrollBarPolicy != "never"
	}
	if s.VerticalScrollBarPolicy != "" {
		w.ShowVScroll = s.VerticalScrollBarPolicy != "never"
	}
	if s.OnCreated != nil {
		s.OnCreated(w)
	}
	registerID(ctx, w, s.ID)
	return w
}

// ── CodeEditor wrapper ────────────────────────────────

type CodeEditor struct {
	W
	Text            string                    `qml:"text"`
	Language        string                    `qml:"language"`
	ShowLineNumbers bool                      `qml:"showLineNumbers"`
	ReadOnly        bool                      `qml:"readOnly"`
	OnCreated       func(*widgets.CodeEditor) `qml:"-"`
}

func (c *CodeEditor) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewCodeEditor()
	ApplyLayout(w, &c.W)
	if c.OnCreated != nil {
		c.OnCreated(w)
	}
	registerID(ctx, w, c.ID)
	return w
}

// ── MarkdownView wrapper ──────────────────────────────

type MarkdownView struct {
	W
	Text      string                      `qml:"text"`
	OnCreated func(*widgets.MarkdownView) `qml:"-"`
}

func (m *MarkdownView) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewMarkdownView()
	ApplyLayout(w, &m.W)
	if m.OnCreated != nil {
		m.OnCreated(w)
	}
	registerID(ctx, w, m.ID)
	return w
}

// ── MarkdownEditor wrapper ────────────────────────────

// MarkdownEditor is the two-pane markdown editor: a source pane and a rendered
// preview over the same document.
type MarkdownEditor struct {
	W
	Text       string                        `qml:"text"`
	ReadOnly   bool                          `qml:"readOnly"`
	Mode       string                        `qml:"mode"`       // "split", "source", "preview"
	SplitRatio float32                       `qml:"splitRatio"` // 0..1, source share
	OnCreated  func(*widgets.MarkdownEditor) `qml:"-"`
	OnChanged  func(string)                  `qml:"-"`
}

func (m *MarkdownEditor) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewMarkdownEditor()
	if m.Text != "" {
		w.SetText(m.Text)
	}
	switch m.Mode {
	case "source":
		w.SetMode(widgets.MarkdownSourceOnly)
	case "preview":
		w.SetMode(widgets.MarkdownPreviewOnly)
	default:
		w.SetMode(widgets.MarkdownSplit)
	}
	if m.SplitRatio > 0 {
		w.SetSplitRatio(m.SplitRatio)
	}
	w.SetReadOnly(m.ReadOnly)
	if m.OnChanged != nil {
		w.OnTextChanged = m.OnChanged
	}
	ApplyLayout(w, &m.W)
	if m.OnCreated != nil {
		m.OnCreated(w)
	}
	registerID(ctx, w, m.ID)
	return w
}

// ── ToolBar wrapper ───────────────────────────────────

type ToolBar struct {
	W
	OnCreated func(*widgets.ToolBar) `qml:"-"`
}

func (t *ToolBar) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewToolBar()
	ApplyLayout(w, &t.W)
	if t.OnCreated != nil {
		t.OnCreated(w)
	}
	registerID(ctx, w, t.ID)
	return w
}

// ── MenuBar wrapper ───────────────────────────────────

type MenuBar struct {
	W
	OnCreated func(*widgets.MenuBar) `qml:"-"`
}

func (m *MenuBar) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewMenuBar()
	ApplyLayout(w, &m.W)
	if m.OnCreated != nil {
		m.OnCreated(w)
	}
	registerID(ctx, w, m.ID)
	return w
}

// ── Dialog wrapper ────────────────────────────────────

type Dialog struct {
	W
	Title     string                         `qml:"title"`
	Modal     bool                           `qml:"modal"`
	OnCreated func(*widgets.ModalPopupPanel) `qml:"-"`
}

func (d *Dialog) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewDialog()
	if d.Title != "" {
		w.Title = d.Title
	}
	ApplyLayout(w, &d.W)
	if d.OnCreated != nil {
		d.OnCreated(w)
	}
	registerID(ctx, w, d.ID)
	return w
}

// ── TabPanel wrapper ─────────────────────────────────

type TabItem struct {
	Title   string
	Icon    string
	Content WidgetWrapper
	Pinned  bool
}

type TabPanel struct {
	W
	Closable    bool `qml:"closable"`
	Orientation int  `qml:"orientation"`
	Tabs        []TabItem
	OnCreated   func(*widgets.TabPanel) `qml:"-"`
}

func (t *TabPanel) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewTabPanel()
	ApplyLayout(w, &t.W)
	if t.Closable {
		w.SetClosable(true)
	}
	w.Orientation = t.Orientation
	for _, tab := range t.Tabs {
		content := tab.Content.Build(ctx)
		w.AddTabEx(tab.Title, tab.Title, tab.Icon, content, tab.Pinned)
	}
	if t.OnCreated != nil {
		t.OnCreated(w)
	}
	registerID(ctx, w, t.ID)
	return w
}

// ── StackPanel wrapper ───────────────────────────────

type StackPanel struct {
	W
	Orientation string `qml:"orientation"`
	ItemSizes   []float32
	Children    []WidgetWrapper
	OnCreated   func(*widgets.StackPanel) `qml:"-"`
}

func (s *StackPanel) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewStackPanel(s.Orientation)
	ApplyLayout(w, &s.W)
	// Default to an even split of the declared panel size when ItemSizes
	// is not provided (prevents every item defaulting to 200px).
	sizes := s.ItemSizes
	if len(sizes) == 0 && len(s.Children) > 0 {
		var avail float32
		if s.Orientation == "horizontal" {
			avail = s.W.Width
		} else {
			avail = s.W.Height
		}
		if avail > 0 {
			each := (avail - float32(len(s.Children)-1)*4) / float32(len(s.Children))
			if each < 1 {
				each = 1
			}
			sizes = make([]float32, len(s.Children))
			for i := range sizes {
				sizes[i] = each
			}
		}
	}
	for i, child := range s.Children {
		size := float32(200)
		if i < len(sizes) {
			size = sizes[i]
		}
		w.AddItem(child.Build(ctx), size)
	}
	if s.OnCreated != nil {
		s.OnCreated(w)
	}
	registerID(ctx, w, s.ID)
	return w
}

// ── DataTable wrapper ──────────────────────────────────

type DataTable struct {
	W
	Columns   []widgets.DataColumn
	OnCreated func(*widgets.DataTable) `qml:"-"`
}

func (d *DataTable) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewDataTable()
	ApplyLayout(w, &d.W)
	w.Columns = d.Columns
	if d.OnCreated != nil {
		d.OnCreated(w)
	}
	registerID(ctx, w, d.ID)
	return w
}

// ── ContextMenu wrapper ───────────────────────────────

type ContextMenu struct {
	W
	Items     []widgets.MenuItem
	OnCreated func(*widgets.ContextMenu) `qml:"-"`
}

func (c *ContextMenu) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewContextMenu()
	ApplyLayout(w, &c.W)
	if len(c.Items) > 0 {
		w.SetItems(c.Items)
	}
	if c.OnCreated != nil {
		c.OnCreated(w)
	}
	registerID(ctx, w, c.ID)
	return w
}

// ── MdiWindow wrapper ─────────────────────────────────

type MdiWindow struct {
	W
	Title     string `qml:"title"`
	Children  []WidgetWrapper
	OnCreated func(*widgets.MdiWindow) `qml:"-"`
}

func (m *MdiWindow) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewMdiWindow()
	ApplyLayout(w, &m.W)
	if m.Title != "" {
		w.SetTitle(m.Title)
	}
	for _, child := range m.Children {
		w.AddChild(child.Build(ctx))
	}
	if m.OnCreated != nil {
		m.OnCreated(w)
	}
	registerID(ctx, w, m.ID)
	return w
}

// ── MdiContainer wrapper ──────────────────────────────

type MdiContainer struct {
	W
	Children  []WidgetWrapper
	OnCreated func(*widgets.MdiContainer) `qml:"-"`
}

func (m *MdiContainer) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewMdiContainer()
	ApplyLayout(w, &m.W)
	for _, child := range m.Children {
		w.AddChild(child.Build(ctx))
	}
	if m.OnCreated != nil {
		m.OnCreated(w)
	}
	registerID(ctx, w, m.ID)
	return w
}

// ── DockPanel wrapper (legacy; prefer MdiContainer) ───

type DockPanel struct {
	W
	OnCreated func(*widgets.DockPanel) `qml:"-"`
}

func (d *DockPanel) Build(ctx *BuildContext) meui.Widget {
	w := widgets.NewDockPanel()
	ApplyLayout(w, &d.W)
	if d.OnCreated != nil {
		d.OnCreated(w)
	}
	registerID(ctx, w, d.ID)
	return w
}
