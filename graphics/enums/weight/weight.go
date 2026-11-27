// Package weight mirrors the reference toolkit's weight enum.
// mobilegles has no font weight enum; values are local.
package weight

type Enum int

const (
	Thin Enum = iota
	ExtraLight
	Light
	Regular
	Medium
	SemiBold
	Bold
	ExtraBold
	Black
)
