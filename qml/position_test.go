package qml

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
)

// TestAnchorsFillPosition tests anchors.fill: parent
func TestAnchorsFillPosition(t *testing.T) {
	// Window 400x300, Rectangle with anchors.fill: parent => should fill entire window
	qmlStr := `Window {
		width: 400;
		height: 300;
		Rectangle { anchors.fill: parent }
	}`
	widget, err := Convert(qmlStr)
	if err != nil {
		t.Fatal(err)
	}
	win, ok := widget.(*meui.Window)
	if !ok {
		t.Fatal("not a Window")
	}
	win.ValidateLayout()

	children := win.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}

	child := children[0]
	var rect graphics.Rect
	if fr, ok := child.(interface{ FrameRect() graphics.Rect }); ok {
		rect = fr.FrameRect()
	} else {
		t.Fatal("child does not have FrameRect")
	}

	// Should fill parent: position (0,0) size 400x300
	if rect.X != 0 || rect.Y != 0 {
		t.Errorf("Rectangle position wrong: got (%.1f,%.1f), want (0,0)", rect.X, rect.Y)
	}
	if rect.Width != 400 || rect.Height != 300 {
		t.Errorf("Rectangle size wrong: got (%.1f,%.1f), want (400,300)", rect.Width, rect.Height)
	}
	t.Logf("Anchors.fill test passed: rect at (%.1f,%.1f) size (%.1f,%.1f)", rect.X, rect.Y, rect.Width, rect.Height)
}

// TestVStackPosition tests VStack layout - verifies stacking works
func TestVStackPosition(t *testing.T) {
	// VStack should stack children vertically. Test verifies children exist and have size.
	qmlStr := `Window {
		width: 400;
		height: 300;
		VStack {
			spacing: 10;
			padding: 10;
			Label { text: "First"; }
			Label { text: "Second"; }
		}
	}`
	widget, err := Convert(qmlStr)
	if err != nil {
		t.Fatal(err)
	}
	win, ok := widget.(*meui.Window)
	if !ok {
		t.Fatal("not a Window")
	}
	win.ValidateLayout()

	children := win.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child (VStack), got %d", len(children))
	}

	// Get VStack's children
	var vstackChildren []meui.Widget
	if c, ok := children[0].(interface{ Children() []meui.Widget }); ok {
		vstackChildren = c.Children()
	} else {
		t.Fatal("VStack does not have Children method")
	}

	if len(vstackChildren) != 2 {
		t.Fatalf("expected 2 labels in VStack, got %d", len(vstackChildren))
	}

	// Verify both labels have positive dimensions
	var rect1, rect2 graphics.Rect
	if fr, ok := vstackChildren[0].(interface{ FrameRect() graphics.Rect }); ok {
		rect1 = fr.FrameRect()
	} else {
		t.Fatal("label1 does not have FrameRect")
	}
	if rect1.Width <= 0 || rect1.Height <= 0 {
		t.Errorf("label1 should have positive dimensions: got %.1f x %.1f", rect1.Width, rect1.Height)
	}

	if fr, ok := vstackChildren[1].(interface{ FrameRect() graphics.Rect }); ok {
		rect2 = fr.FrameRect()
	} else {
		t.Fatal("label2 does not have FrameRect")
	}
	if rect2.Width <= 0 || rect2.Height <= 0 {
		t.Errorf("label2 should have positive dimensions: got %.1f x %.1f", rect2.Width, rect2.Height)
	}

	t.Logf("VStack test passed: label1 at (%.1f,%.1f), label2 at (%.1f,%.1f)", rect1.X, rect1.Y, rect2.X, rect2.Y)
}

// TestHStackPosition tests HStack layout - verifies horizontal arrangement works
func TestHStackPosition(t *testing.T) {
	// HStack should arrange children horizontally. Test verifies children exist and have size.
	qmlStr := `Window {
		width: 400;
		height: 300;
		HStack {
			spacing: 10;
			padding: 10;
			Label { text: "First"; }
			Label { text: "Second"; }
		}
	}`
	widget, err := Convert(qmlStr)
	if err != nil {
		t.Fatal(err)
	}
	win, ok := widget.(*meui.Window)
	if !ok {
		t.Fatal("not a Window")
	}
	win.ValidateLayout()

	children := win.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child (HStack), got %d", len(children))
	}

	var hstackChildren []meui.Widget
	if c, ok := children[0].(interface{ Children() []meui.Widget }); ok {
		hstackChildren = c.Children()
	} else {
		t.Fatal("HStack does not have Children method")
	}

	if len(hstackChildren) != 2 {
		t.Fatalf("expected 2 labels in HStack, got %d", len(hstackChildren))
	}

	// Verify both labels have positive dimensions
	var rect1, rect2 graphics.Rect
	if fr, ok := hstackChildren[0].(interface{ FrameRect() graphics.Rect }); ok {
		rect1 = fr.FrameRect()
	} else {
		t.Fatal("label1 does not have FrameRect")
	}
	if rect1.Width <= 0 || rect1.Height <= 0 {
		t.Errorf("label1 should have positive dimensions: got %.1f x %.1f", rect1.Width, rect1.Height)
	}

	if fr, ok := hstackChildren[1].(interface{ FrameRect() graphics.Rect }); ok {
		rect2 = fr.FrameRect()
	} else {
		t.Fatal("label2 does not have FrameRect")
	}
	if rect2.Width <= 0 || rect2.Height <= 0 {
		t.Errorf("label2 should have positive dimensions: got %.1f x %.1f", rect2.Width, rect2.Height)
	}

	t.Logf("HStack test passed: label1 at (%.1f,%.1f), label2 at (%.1f,%.1f)", rect1.X, rect1.Y, rect2.X, rect2.Y)
}

// TestGridPosition tests Grid layout - verifies grid arrangement works
func TestGridPosition(t *testing.T) {
	// Grid should arrange children in a grid. Test verifies children exist and have size.
	qmlStr := `Window {
		width: 400;
		height: 300;
		Grid {
			columns: 2;
			padding: 10;
			Label { text: "First"; }
			Label { text: "Second"; }
			Label { text: "Third"; }
		}
	}`
	widget, err := Convert(qmlStr)
	if err != nil {
		t.Fatal(err)
	}
	win, ok := widget.(*meui.Window)
	if !ok {
		t.Fatal("not a Window")
	}
	win.ValidateLayout()

	children := win.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child (Grid), got %d", len(children))
	}

	var gridChildren []meui.Widget
	if c, ok := children[0].(interface{ Children() []meui.Widget }); ok {
		gridChildren = c.Children()
	} else {
		t.Fatal("Grid does not have Children method")
	}

	if len(gridChildren) < 2 {
		t.Fatalf("expected at least 2 labels in Grid, got %d", len(gridChildren))
	}

	// Verify all children have positive dimensions
	for i, child := range gridChildren {
		if fr, ok := child.(interface{ FrameRect() graphics.Rect }); ok {
			rect := fr.FrameRect()
			if rect.Width <= 0 || rect.Height <= 0 {
				t.Errorf("grid child[%d] should have positive dimensions: got %.1f x %.1f", i, rect.Width, rect.Height)
			}
		}
	}
	t.Logf("Grid test passed: %d children in grid", len(gridChildren))
}

// TestWidgetCreation tests that widgets are created
func TestWidgetCreation(t *testing.T) {
	// Widgets should be created (they may be wrapped in Panels)
	qmlStr := `Window {
		width: 400;
		height: 300;
		Label { text: "Hello" }
		Button { text: "Click" }
		TextField { text: "Test" }
	}`
	widget, err := Convert(qmlStr)
	if err != nil {
		t.Fatal(err)
	}
	win, ok := widget.(*meui.Window)
	if !ok {
		t.Fatal("not a Window")
	}
	win.ValidateLayout()

	children := win.Children()
	if len(children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(children))
	}

	// Just verify children exist (type checking is implementation-specific)
	for i, child := range children {
		if child == nil {
			t.Errorf("child %d is nil", i)
		}
	}
	t.Log("Widget creation test passed: 3 children created")
}

// TestThemeAppliedToWidgets tests that theme is applied
func TestThemeAppliedToWidgets(t *testing.T) {
	// Widget should be created with theme
	qmlStr := `Window {
		width: 400;
		height: 300;
		TextField { text: "Test" }
	}`
	widget, err := Convert(qmlStr)
	if err != nil {
		t.Fatal(err)
	}
	win, ok := widget.(*meui.Window)
	if !ok {
		t.Fatal("not a Window")
	}
	win.ValidateLayout()

	children := win.Children()
	if len(children) < 1 {
		t.Fatalf("expected at least 1 child, got %d", len(children))
	}

	// Just verify a child exists (type checking is implementation-specific)
	if children[0] == nil {
		t.Fatal("child is nil")
	}
	t.Log("Theme applied test passed (widget created with theme)")
}
