package widgets

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sixnineone/meui/resources"
)

func TestWidgetsReadResourceURIs(t *testing.T) {
	var buf bytes.Buffer
	im := image.NewRGBA(image.Rect(0, 0, 2, 2))
	im.Set(0, 0, color.NRGBA{R: 255, A: 255})
	if err := png.Encode(&buf, im); err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{
		"res/tiny.png": {Data: buf.Bytes()},
		"res/hello.go": {Data: []byte("package x\n")},
	}
	b := resources.Default()
	if err := b.Mount("res", fsys, "res"); err != nil {
		t.Fatal(err)
	}
	defer b.Unmount("res")

	img := NewImage()
	if err := img.SetSource("res://tiny.png"); err != nil {
		t.Fatalf("Image.SetSource: %v", err)
	}
	if img.Image() == nil || img.Image().Width() != 2 || img.Image().Height() != 2 {
		t.Fatalf("image not loaded from the bundle")
	}

	ed := NewCodeEditor()
	ed.SetFilePath("res://hello.go")
	if !strings.Contains(ed.Text(), "package x") {
		t.Fatalf("CodeEditor.SetFilePath read %q", ed.Text())
	}

	mv := NewMarkdownView()
	mv.SetFilePath("res://hello.go")
	if !strings.Contains(mv.Text(), "package x") {
		t.Fatalf("MarkdownView.SetFilePath read %q", mv.Text())
	}
}
