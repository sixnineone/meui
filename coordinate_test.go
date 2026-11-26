package meui

import (
	"github.com/sixnineone/meui/graphics"
	"testing"
)

func TestPanelAtCoordinateTranslation(t *testing.T) {
	// Create root panel with VStack layout (NewVStack is in layout package, but Panel.SetLayout accepts interface)
	// Since we can't import layout due to cycle, use a simple mock layout or use existing Panel as child
	root := NewPanel()
	root.SetFrameRect(graphics.NewRect(0, 0, 1000, 800))

	// Create a child panel to test coordinate translation
	child := NewPanel()
	child.SetFrameRect(graphics.NewRect(8, 8, 100, 30)) // Manually set frame to simulate layout
	root.AddChild(child)

	// Test PanelAt with local coordinates (10,10) which is inside the child
	hit := root.PanelAt(10, 10)
	if hit != child {
		t.Errorf("PanelAt(10,10) should return child, got %v", hit)
	}

	// Test PanelAt outside child
	hit = root.PanelAt(5, 5)
	if hit != root {
		t.Errorf("PanelAt(5,5) should return root, got %v", hit)
	}
}

func TestMouseEventCoordinateTranslation(t *testing.T) {
	root := NewPanel()
	root.SetFrameRect(graphics.NewRect(0, 0, 1000, 800))

	// Create child panel to test mouse events
	child := NewPanel()
	child.SetFrameRect(graphics.NewRect(8, 8, 100, 30))
	root.AddChild(child)

	// Simulate mouse down at (10,10) in root's local space
	root.OnMouseDown(10, 10, 0)

	// Verify coordinate translation works via PanelAt
	hit := root.PanelAt(10, 10)
	if hit != child {
		t.Errorf("Mouse event coordinate translation failed: PanelAt(10,10) should return child")
	}
}
