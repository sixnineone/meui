// Package slant mirrors the reference toolkit's slant enum.
// mobilegles has no font slant enum; values are local.
package slant

type Enum int

const (
	Upright Enum = iota
	Italic
)
