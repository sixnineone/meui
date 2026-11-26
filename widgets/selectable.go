package widgets

import (
	"github.com/sixnineone/meui"
	"sync"
)

// ButtonGroup manages exclusive selection for a group of selectable widgets.
type ButtonGroup struct {
	ID             string
	Exclusive      bool
	SelectedWidget meui.Widget
	Widgets        []meui.Widget
	mu             sync.RWMutex
}

var (
	buttonGroups   = make(map[string]*ButtonGroup)
	buttonGroupsMu sync.RWMutex
)

// GetOrCreateGroup returns an existing group or creates a new one.
func GetOrCreateGroup(id string, exclusive bool) *ButtonGroup {
	buttonGroupsMu.Lock()
	defer buttonGroupsMu.Unlock()
	if g, ok := buttonGroups[id]; ok {
		return g
	}
	g := &ButtonGroup{
		ID:        id,
		Exclusive: exclusive,
	}
	buttonGroups[id] = g
	return g
}

// GetGroup returns a group by ID, or nil if not found.
func GetGroup(id string) *ButtonGroup {
	buttonGroupsMu.RLock()
	defer buttonGroupsMu.RUnlock()
	return buttonGroups[id]
}

// Selectable provides group-based exclusive selection behavior.
// Embed this in widgets that need group selection (Button, RadioButton, etc.)
type Selectable struct {
	GroupID  string
	Selected bool
}

// SetGroupID assigns the widget to a group. If already selected, syncs with the group.
func (s *Selectable) SetGroupID(id string, widget meui.Widget) {
	s.GroupID = id
	if id != "" {
		g := GetOrCreateGroup(id, true)
		g.addWidget(widget)
		if s.Selected {
			g.setSelected(widget)
		}
	}
}

// SetSelected sets selection state and manages group exclusivity.
func (s *Selectable) SetSelected(selected bool, widget meui.Widget, onSelected func(bool)) {
	if s.Selected == selected {
		return
	}
	s.Selected = selected
	if selected && s.GroupID != "" {
		if g := GetGroup(s.GroupID); g != nil {
			g.setSelected(widget)
		}
	}
	if onSelected != nil {
		onSelected(selected)
	}
}

// Dispose removes the widget from its group.
func (s *Selectable) Dispose(widget meui.Widget) {
	if s.GroupID != "" {
		if g := GetGroup(s.GroupID); g != nil {
			g.removeWidget(widget)
		}
	}
}

// addWidget adds a widget to the group.
func (g *ButtonGroup) addWidget(w meui.Widget) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Widgets = append(g.Widgets, w)
}

// removeWidget removes a widget from the group.
func (g *ButtonGroup) removeWidget(w meui.Widget) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, widget := range g.Widgets {
		if widget == w {
			g.Widgets = append(g.Widgets[:i], g.Widgets[i+1:]...)
			break
		}
	}
	if g.SelectedWidget == w {
		g.SelectedWidget = nil
	}
}

// setSelected sets the selected widget, deselecting others if exclusive.
func (g *ButtonGroup) setSelected(w meui.Widget) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.Exclusive && g.SelectedWidget != nil && g.SelectedWidget != w {
		switch prev := g.SelectedWidget.(type) {
		case *Button:
			prev.SetSelected(false)
		case *RadioButton:
			prev.SetSelected(false)
		}
	}
	g.SelectedWidget = w
}
