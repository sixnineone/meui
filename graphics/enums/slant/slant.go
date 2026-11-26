// Package slant mirrors pryone.com/extern/unison/enums/slant.
// mobilegles has no font slant enum; values are local.
package slant

type Enum int

const (
	Upright Enum = iota
	Italic
)
