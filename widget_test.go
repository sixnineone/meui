package meui

import (
	"os"
	"testing"

	"github.com/sixnineone/meui/graphics"
)

func TestPrintTree(t *testing.T) {
	// Create a simple widget tree
	root := NewWindow()
	root.SetFrameRect(graphics.NewRect(0, 0, 800, 600))
	root.SetDebugName("Window")

	panel := NewPanel()
	panel.SetFrameRect(graphics.NewRect(100, 100, 600, 400))
	panel.SetDebugName("Panel")
	root.AddChild(panel)

	button := NewPanel() // mock button
	button.SetFrameRect(graphics.NewRect(120, 120, 200, 30))
	button.SetDebugName("Button")
	panel.AddChild(button)

	// Set tree paths
	SetTreePaths(root)

	// Print tree
	PrintTree(root, 0)

	// Print tree to file
	f, err := os.Create("/tmp/meui_test_tree.txt")
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	PrintTreeToFile(root, 0, f)
	f.Close()

	// Verify file was created
	if _, err := os.Stat("/tmp/meui_test_tree.txt"); os.IsNotExist(err) {
		t.Errorf("Tree file was not created")
	}
}
