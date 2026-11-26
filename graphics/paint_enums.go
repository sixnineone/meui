package graphics

// Paint enums owned by the seam. They mirror the renderer's numeric values so
// the backend conversion is a plain cast, but a second backend is free to map
// them however it likes — nothing in the toolkit names a backend type.

// StrokeCap is the line-end style for stroked draws.
type StrokeCap int

const (
	StrokeCapButt StrokeCap = iota
	StrokeCapRound
	StrokeCapSquare
)

// StrokeJoin is the line-join style for stroked draws.
type StrokeJoin int

const (
	StrokeJoinMiter StrokeJoin = iota
	StrokeJoinRound
	StrokeJoinBevel
)

// BlendMode selects how a draw composites with what is already there.
type BlendMode int

const (
	BlendModeNormal BlendMode = iota
	BlendModeMultiply
	BlendModeScreen
	BlendModeOverlay
	BlendModeDarken
	BlendModeLighten
	BlendModeColorDodge
	BlendModeColorBurn
	BlendModeHardLight
	BlendModeSoftLight
	BlendModeDifference
	BlendModeExclusion
	BlendModeHue
	BlendModeSaturation
	BlendModeColor
	BlendModeLuminosity
)
