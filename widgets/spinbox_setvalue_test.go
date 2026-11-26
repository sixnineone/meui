package widgets

import "testing"

// TestSpinBoxSetValueUpdatesEditBuffer verifies that a programmatic SetValue
// keeps the on-screen display in sync while the spinbox is focused (edit mode):
// Paint renders editBuffer in edit mode, so SetValue must refresh it.
// Regression for the Management tab: quick-set buttons and Refresh updated
// SpinBox.Value but focused spinboxes kept showing the stale editBuffer.
func TestSpinBoxSetValueUpdatesEditBuffer(t *testing.T) {
	sb := NewSpinBox()
	sb.SetDecimals(1)
	sb.SetValue(15)

	// Focus the spinbox → edit mode captures the current value into the buffer.
	sb.OnFocus()
	if !sb.editMode {
		t.Fatalf("expected edit mode after OnFocus")
	}
	if sb.editBuffer != "15.0" {
		t.Fatalf("editBuffer after focus = %q, want %q", sb.editBuffer, "15.0")
	}

	// Programmatic update while focused — the buffer must follow.
	sb.SetValue(7)
	if sb.Value != 7 {
		t.Fatalf("Value = %v, want 7", sb.Value)
	}
	if sb.editBuffer != "7.0" {
		t.Fatalf("editBuffer after SetValue = %q, want %q (paint shows the buffer)", sb.editBuffer, "7.0")
	}

	// Unfocused SetValue (no edit mode) must not create a buffer.
	sb2 := NewSpinBox()
	sb2.SetValue(3)
	if sb2.editMode || sb2.editBuffer != "" {
		t.Fatalf("unfocused SetValue created edit state: mode=%v buffer=%q", sb2.editMode, sb2.editBuffer)
	}
}
