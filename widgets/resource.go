package widgets

import (
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/resources"
)

// The widgets that take a path from a scene accept resource URIs too:
// res://... is the application bundle, meui://... the toolkit's defaults.
// A disk path keeps its old behaviour, including the regular-file guard.
func readTextResource(path string) ([]byte, error) {
	if resources.IsURI(path) {
		return resources.ReadFileLimit(path, maxTextFileBytes)
	}
	return readTextFile(path)
}

func loadImage(path string) (*graphics.Image, error) {
	if resources.IsURI(path) {
		data, err := resources.ReadFileLimit(path, 256<<20)
		if err != nil {
			return nil, err
		}
		return graphics.NewImageFromBytes(data)
	}
	return graphics.NewImageFromFilePathOrURL(path, 1)
}
