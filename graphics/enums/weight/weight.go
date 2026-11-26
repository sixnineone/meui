// Package weight mirrors pryone.com/extern/unison/enums/weight.
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
