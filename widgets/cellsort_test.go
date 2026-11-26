package widgets

// cellsort_test.go — the one cell-value comparison the toolkit's tables sort with.
//
// The property that matters is that a column declared ColNumber compares the NUMBERS
// it displays. Comparing the strings is what makes "10" sort before "9" and
// "$1,234.56" sort before "$9.00" — and it is what an app-level table sorter did three
// times over before this lived beside DataColType.

import "testing"

func TestParseCellNumberReadsTheFormatsATablePutsInCells(t *testing.T) {
	cases := []struct {
		text string
		want float64
		ok   bool
	}{
		{"1.5", 1.5, true},
		{"1.5%", 1.5, true},
		{"-0.3%", -0.3, true},
		{"62%", 62, true},
		{"$9.00", 9, true},
		{"$-1.23", -1.23, true},
		{"$1,234.56", 1234.56, true},
		{"+2", 2, true},
		{"  42  ", 42, true},
		{"\u22120.5", -0.5, true}, // unicode minus
		{"", 0, false},
		{"-", 0, false},
		{"\u2014", 0, false}, // em dash, which a "no basis" percent renders as
		{"n/a", 0, false},
		{"BUY", 0, false},
		{"nan", 0, false},
		{"Inf", 0, false},
	}
	for _, c := range cases {
		got, ok := ParseCellNumber(c.text)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("ParseCellNumber(%q) = (%v, %v), want (%v, %v)", c.text, got, ok, c.want, c.ok)
		}
	}
}

func TestNumericColumnsCompareByValueNotByText(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"9", "10", -1},         // text order would say "10" first
		{"$9.00", "$10.00", -1}, // currency: strip before comparing
		{"1.5%", "9.0%", -1},    // percent
		{"1,234.5", "999.9", 1}, // thousands separator
		{"-0.3", "0.1", -1},     // sign
		{"2.00", "2.0", 0},      // same number written two ways
	}
	for _, c := range cases {
		if got := CompareCells(c.a, c.b, ColNumber); got != c.want {
			t.Errorf("CompareCells(%q, %q, ColNumber) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

// A cell with no number is missing, and missing is not a value: it must not be
// presented as the smallest number, nor lead a descending column as if it were the
// largest. It belongs at the bottom either way.
func TestAMissingNumberSinksBelowRealValuesInBothDirections(t *testing.T) {
	if got := CompareCells("-", "5", ColNumber); got != 1 {
		t.Errorf("CompareCells(\"-\", \"5\") = %d, want 1 (missing after a value ascending)", got)
	}
	if got := CompareCells("5", "-", ColNumber); got != -1 {
		t.Errorf("CompareCells(\"5\", \"-\") = %d, want -1", got)
	}
	if CellLess("-", "30", ColNumber, true) {
		t.Error("ascending: a missing value must not lead the column")
	}
	if CellLess("-", "30", ColNumber, false) {
		t.Error("descending: a missing value must not lead the column")
	}
	if !CellLess("30", "-", ColNumber, true) {
		t.Error("ascending: a real value must precede a missing one")
	}
	if !CellLess("30", "-", ColNumber, false) {
		t.Error("descending: a real value must precede a missing one")
	}
}

func TestTextColumnsCompareWithoutRegardToCase(t *testing.T) {
	// Case-sensitively "Zebra" (Z, 0x5A) precedes "apple" (a, 0x61); ignoring
	// case it follows it. The second is what a table wants.
	if got := CompareCells("Zebra", "apple", ColText); got != 1 {
		t.Errorf("CompareCells(\"Zebra\", \"apple\", ColText) = %d, want 1 (case-insensitive)", got)
	}
	if got := CompareCells("EURUSD", "eurusd", ColText); got == 0 {
		t.Error("cells that differ only in case must still have a defined order, not compare equal")
	}
}

// ColPercent is a number that is WRITTEN with a sign, not a different comparison:
// "9%" precedes "10%". A ColText column keeps its text order, decoration and all —
// which is the difference the declared type is there to make.
func TestPercentComparesAsANumberAndTextDoesNot(t *testing.T) {
	if got := CompareCells("9%", "10%", ColPercent); got != -1 {
		t.Errorf("CompareCells(\"9%%\", \"10%%\", ColPercent) = %d, want -1", got)
	}
	if got := CompareCells("10%", "9%", ColText); got != -1 {
		t.Errorf("CompareCells(\"10%%\", \"9%%\", ColText) = %d, want -1 (text order)", got)
	}
}

// The comparator a sort.Slice* call is given has to be a strict weak ordering: nothing
// precedes itself, and — among values of the same kind — descending is the exact
// converse of ascending. "return !less" fails the first half, which is how a descending
// sort of equal cells came to claim each preceded the other.
//
// The one deliberate exception is a missing numeric value, which sinks in BOTH
// directions and so is not part of the converse; that is asserted separately above.
func TestTheOrderingIsStrict(t *testing.T) {
	values := []string{"", "-", "\u2014", "0", "2.0", "2.00", "10", "$9.00", "1.5%", "BUY", "eurusd"}
	for _, kind := range []DataColType{ColText, ColNumber, ColPercent} {
		for _, asc := range []bool{true, false} {
			for _, v := range values {
				if CellLess(v, v, kind, asc) {
					t.Errorf("CellLess(%q, %q, %v, asc=%v) = true — a value must not precede itself", v, v, kind, asc)
				}
			}
		}
		for _, a := range values {
			for _, b := range values {
				if kind != ColText {
					_, oka := ParseCellNumber(a)
					_, okb := ParseCellNumber(b)
					if oka != okb {
						continue // a missing value is the documented exception
					}
				}
				if CellLess(a, b, kind, true) != CellLess(b, a, kind, false) {
					t.Errorf("CellLess(%q, %q, %v): ascending is not the converse of descending", a, b, kind)
				}
			}
		}
	}
}

// ColumnLess takes the type off the column, so a sorter cannot pass the type that
// belongs to a different column — and an index past the columns is not a sort.
func TestColumnLessReadsTheColumnsOwnType(t *testing.T) {
	dt := NewDataTable()
	dt.SetColumns([]DataColumn{
		{Label: "Size", Type: ColNumber},
		{Label: "Symbol", Type: ColText},
	})

	if !dt.ColumnLess(0, "9.00", "10.00", true) {
		t.Error("the ColNumber column must compare 9.00 before 10.00")
	}
	if !dt.ColumnLess(1, "10.00", "9.00", true) {
		t.Error("the ColText column must keep text order: \"10.00\" before \"9.00\"")
	}
	if dt.ColumnLess(-1, "a", "b", true) {
		t.Error("a negative column index is not a sort")
	}
	if dt.ColumnLess(len(dt.Columns), "a", "b", true) {
		t.Error("a column index past the end is not a sort")
	}
}
