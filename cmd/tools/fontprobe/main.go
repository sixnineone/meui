package main

import (
	"fmt"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/slant"
	"github.com/sixnineone/meui/graphics/enums/spacing"
	"github.com/sixnineone/meui/graphics/enums/weight"
)

func main() {

	f := graphics.MatchFontFace("Roboto", weight.Light, spacing.Standard, slant.Upright)
	fmt.Printf("matchface roboto: %+v\n", f)
	if f != nil {
		f10 := f.Font(10)
		fmt.Printf("font(10): size=%v lineH=%v baseline=%v\n", f10.Size(), f10.LineHeight(), f10.Baseline())
		if face := f10.Face(); face != nil {
			fmt.Printf("face: %+v\n", face)
			f6 := face.Font(6)
			fmt.Printf("face.font(6): size=%v lineH=%v\n", f6.Size(), f6.LineHeight())
		}
	}
	sf := graphics.SystemFont
	if sf != nil {
		fmt.Printf("systemfont: size=%v lineH=%v\n", sf.Size(), sf.LineHeight())
	}
}
