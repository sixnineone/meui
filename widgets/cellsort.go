package widgets

// cellsort.go — how a column's cell text becomes a comparable value.
//
// DataColumn.Type is a column's declared type, and it is the ONLY declaration: the
// same value decides the cell alignment (Paint) and how the column sorts (CellLess /
// ColumnLess). Comparing a ColNumber column as text is what puts "10" before "9" and
// "$1,234.56" before "$9.00" — and it is the shape of a real defect in the app that
// consumes this table: three private comparators drifted apart, and a positional type
// table kept beside the QML column list went out of step with it, to the point where a
// header click indexed past the end of that table and took the process down.
//
// A column with no type is ColText. That is the safe default, and the reason to give a
// numeric column its type: an untyped column sorts as text.

import (
	"math"
	"strconv"
	"strings"
)

// numericType reports whether a declared column type compares as a number. ColPercent
// is a number that happens to be written with a "%": the sign is decoration on the
// same quantity, so it strips like the currency prefix does.
func numericType(t DataColType) bool {
	return t == ColNumber || t == ColPercent
}

// ParseCellNumber reads the number a cell displays, and reports whether it found one.
//
// It tolerates the formatting a table puts in cells: a currency prefix ("$1,234.56"),
// a percent suffix ("1.5%"), thousands separators, a leading "+", surrounding spaces
// and the unicode minus sign. ok is false for the placeholders ("", "-", "—") and for
// anything else that is not a finite number — so a column can tell "no value" from
// "zero", which are different facts.
func ParseCellNumber(text string) (float64, bool) {
	s := strings.TrimSpace(text)
	if s == "" {
		return 0, false
	}
	s = strings.ReplaceAll(s, "\u2212", "-") // U+2212 MINUS SIGN
	s = strings.TrimSpace(strings.TrimSuffix(s, "%"))
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimLeft(s, "$€£¥ ")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

// CompareCells orders two cell texts for a column of the given type: -1 when a
// belongs before b in the ASCENDING order, +1 when after, 0 when the two are
// equivalent.
//
// In a numeric column a cell with no number is missing and compares greater than any
// real value, so ascending puts it last. Two missing cells fall back to text so their
// order is still defined.
//
// This is the raw ordering. A sort should call CellLess, which keeps a missing value
// at the bottom when the direction flips.
func CompareCells(a, b string, t DataColType) int {
	if !numericType(t) {
		return compareCellText(a, b)
	}
	na, oka := ParseCellNumber(a)
	nb, okb := ParseCellNumber(b)
	switch {
	case !oka && !okb:
		return compareCellText(a, b)
	case !oka:
		return 1
	case !okb:
		return -1
	case na < nb:
		return -1
	case na > nb:
		return 1
	}
	return 0
}

// compareCellText orders two cell texts without regard to case, using the raw text to
// break a case-only tie so the relation is still total.
func compareCellText(a, b string) int {
	la, lb := strings.ToLower(strings.TrimSpace(a)), strings.ToLower(strings.TrimSpace(b))
	switch {
	case la < lb:
		return -1
	case la > lb:
		return 1
	}
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// CellLess is the comparator a sort.Slice* call needs, and within one direction it is
// a STRICT WEAK ORDERING: CellLess(x, x, t, asc) is false, and the order is
// consistent.
//
// Writing the descending case as an inverted "less" is not the same thing, and is how a
// descending sort of equal cells comes to claim each preceded the other — a broken
// contract, which leaves the order of equal rows to the sort's internals. Compare
// first, then choose a side.
//
// Two rules, and they are deliberately not symmetric:
//
//   - A numeric cell with no number is MISSING and sinks below every real value in
//     BOTH directions. A descending sort of a P&L column must not lead with the rows
//     that have no number to show, and a symbol table holds "-" in every momentum cell
//     until the first tick arrives.
//   - Among real values (and among text cells), descending is the exact converse of
//     ascending.
func CellLess(a, b string, t DataColType, asc bool) bool {
	if numericType(t) {
		_, oka := ParseCellNumber(a)
		_, okb := ParseCellNumber(b)
		if oka != okb {
			return oka // the cell that carries a number comes first
		}
	}
	c := CompareCells(a, b, t)
	if asc {
		return c < 0
	}
	return c > 0
}

// ColumnLess compares two cell texts for column col, reading the type off that column.
//
// It is what a table's OnSortChanged handler should call: the type cannot be taken
// from the wrong column, and a column index past the end is never a sort (it returns
// false, which leaves the pair ordered as it was). The caller still checks the index
// once before the sort, so an out-of-range column does no work at all.
func (dt *DataTable) ColumnLess(col int, a, b string, asc bool) bool {
	if col < 0 || col >= len(dt.Columns) {
		return false
	}
	// col is LOGICAL (the caller's declared index, the same index its cells use);
	// the TYPE lives on the display-ordered heading, so it has to be mapped or a
	// reordered numeric column is compared as text.
	display := dt.columnDisplay(col)
	if display < 0 || display >= len(dt.Columns) {
		display = col
	}
	return CellLess(a, b, dt.Columns[display].Type, asc)
}
