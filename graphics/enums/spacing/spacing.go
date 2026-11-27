// Package spacing mirrors the reference toolkit's spacing enum.
// mobilegles has no font spacing enum; values are local.
package spacing

type Enum int

const (
	Standard Enum = iota
	Monospace
)
