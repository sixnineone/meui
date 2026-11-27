// Package paintstyle mirrors the reference toolkit's paint-style enum.
// for the meui migration. Values are int32 so they can be passed
// directly to mobilegles.Paint.SetStyle(int32) without conversion.
package paintstyle

const (
	Fill       int32 = 0
	Stroke     int32 = 1
	FillStroke int32 = 2
)
