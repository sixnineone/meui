// glprobe prints the GL driver the renderer actually landed on, and the limits
// the unified batch depends on. Run it before trusting a frame-rate number:
// llvmpipe and a real GPU differ by an order of magnitude, and the SDF batch
// needs 16 vertex attributes.
//
//	go run ./cmd/tools/glprobe
package main

import (
	"fmt"
	"os"

	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/mobilegles"
)

func main() {
	if _, err := sdl3.NewWindowSized("glprobe", 64, 64); err != nil {
		fmt.Fprintf(os.Stderr, "window: %v\n", err)
		os.Exit(1)
	}
	vendor, renderer, version := mobilegles.GLInfo()
	if renderer == "" {
		fmt.Fprintln(os.Stderr, "no GL context: GLInfo returned nothing")
		os.Exit(1)
	}
	fmt.Printf("vendor:              %s\n", vendor)
	fmt.Printf("renderer:            %s\n", renderer)
	fmt.Printf("version:             %s\n", version)
	fmt.Printf("max vertex attribs:  %d  (the SDF batch uses 16)\n", mobilegles.MaxVertexAttribs())
	fmt.Printf("max texture units:   %d  (one batch samples up to %d textures)\n",
		mobilegles.MaxTextureUnits(), mobilegles.MaxTexturesPerBatch())
}
