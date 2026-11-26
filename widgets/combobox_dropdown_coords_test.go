package widgets

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
)

// A combo whose dropdown sits far from the window origin (the font combo in a
// tab body, ~90px down) highlighted — and then selected — an item a few rows
// below the pointer, while the title-bar combo near the top looked fine: the
// modal dropdown layer covers the window, so the pointer arrives in *content*
// coordinates, and the hover skipped the dropdown offset that the click
// subtracted.
func TestComboBoxDropdownHoverAndClickUseTheSameOffset(t *testing.T) {
	c := NewComboBox()
	c.SetOptions([]string{"one", "two", "three", "four", "five", "six"})
	c.SetFrameRect(graphics.NewRect(12, 300, 200, 28))

	optionHeight := c.ComboBoxTheme.OptionHeight
	if optionHeight <= 0 {
		optionHeight = 30
	}

	// Modal layer: window-sized, with the list starting 328px down. Selecting
	// an option closes the dropdown (and drops the layer), so each probe
	// re-opens it.
	open := func() {
		c.Expanded = true
		c.dropdownLayer = &meui.Layer{X: 0, Y: 0, Width: 1100, Height: 750, Widget: c}
		c.dropdownOffsetX, c.dropdownOffsetY = 12, 328
		c.dropdownWidth, c.dropdownHeight = 200, float32(len(c.Options))*optionHeight
		c.HoveredIndex = -1
	}

	for want := 0; want < len(c.Options); want++ {
		open()
		// A point inside option `want`, in content coordinates.
		cy := c.dropdownOffsetY + float32(want)*optionHeight + optionHeight/2
		cx := c.dropdownOffsetX + 20

		if !c.HandleMouseMove(cx, cy) {
			t.Fatalf("move over option %d was not handled", want)
		}
		if c.HoveredIndex != want {
			t.Fatalf("hover over option %d highlighted %d", want, c.HoveredIndex)
		}

		open()
		c.SelectedIndex = -1
		if !c.HandleMouseDown(cx, cy, 0) {
			t.Fatalf("click on option %d was not handled", want)
		}
		if c.SelectedIndex != want {
			t.Fatalf("click at option %d selected %d", want, c.SelectedIndex)
		}
	}
}

// The legacy positioned layer (the layer is the list) keeps working: its
// coordinates are already dropdown-local.
func TestComboBoxPositionedDropdownStillWorks(t *testing.T) {
	c := NewComboBox()
	c.SetOptions([]string{"one", "two", "three"})
	c.SetFrameRect(graphics.NewRect(10, 40, 160, 26))
	optionHeight := c.ComboBoxTheme.OptionHeight
	if optionHeight <= 0 {
		optionHeight = 30
	}
	c.Expanded = true
	c.dropdownWidth = 160
	c.dropdownHeight = float32(len(c.Options)) * optionHeight
	// Layer exactly the list: width/height equal the dropdown's.
	c.dropdownLayer = &meui.Layer{X: 10, Y: 66, Width: c.dropdownWidth, Height: c.dropdownHeight, Widget: c}
	c.dropdownOffsetX, c.dropdownOffsetY = 10, 66

	// Local coordinates inside option 1 (the layer dispatch has already
	// subtracted the layer origin by the time the widget sees them).
	c.HoveredIndex = -1
	c.HandleMouseMove(20, optionHeight+optionHeight/2)
	if c.HoveredIndex != 1 {
		t.Fatalf("positioned layer hover = %d, want 1", c.HoveredIndex)
	}
	c.HandleMouseDown(20, optionHeight+optionHeight/2, 0)
	if c.SelectedIndex != 1 {
		t.Fatalf("positioned layer click selected %d, want 1", c.SelectedIndex)
	}
}
