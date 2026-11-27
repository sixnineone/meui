// Package pathop mirrors the reference toolkit's path-op enum.
// mobilegles has no path-operation enum; values are local int constants.
// Only the values meui actually uses (Intersect) are defined.
package pathop

type Enum int

const (
	Difference Enum = iota
	Intersect
	ReverseDifference
	Union
	XOR
)
