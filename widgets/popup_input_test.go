package widgets

// popup_input_test.go — a dialog's content is painted and hit-tested from ONE set of offsets.

import (
	"testing"

	"github.com/sixnineone/meui"
)

// A heading moves the content down. Paint and the input offset used to compute that move from two
// different numbers (a hard-coded 14 vs the font's line height), so the moment a heading appeared —
// a validation message, for instance — every control in the dialog was painted a few pixels from
// where it could be clicked.
func TestAPopupHeadingMovesThePaintAndTheHitTestTogether(t *testing.T) {
	popup := NewModalPopupPanel("Edit Signal Rule", "", meui.NewPanel())
	base := popup.bodyY()
	if want := float32(popupTitleH + popupPadding); base != want {
		t.Errorf("a popup with no heading puts its content at %.1f, want %.1f", base, want)
	}

	popup.Heading = "The entry expression has a syntax error:\n\nFix it and save again."
	if got, want := popup.bodyY(), base+popup.headingAdvance(); got != want {
		t.Errorf("bodyY is %.1f, want the shared heading advance (%.1f)", got, want)
	}
	if popup.headingAdvance() <= popupPadding {
		t.Errorf("a heading with text advanced only %.1f", popup.headingAdvance())
	}
}
