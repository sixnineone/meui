// Package spacing mirrors pryone.com/extern/unison/enums/spacing.
// mobilegles has no font spacing enum; values are local.
package spacing

type Enum int

const (
	Standard Enum = iota
	Monospace
)
