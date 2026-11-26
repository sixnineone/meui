package web

// The behaviour layer's own contract: a scene parses, an id that does not exist is
// REPORTED (not silently ignored), and the headless shot writes a real PNG.

import (
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"
)

const sceneSrc = `
Window {
  width: 160
  height: 100
  Rectangle {
    id: page
    anchors.fill: parent
    color: "#FCF4DC"

    Rectangle {
      id: cta
      x: 20
      y: 20
      width: 80
      height: 32
      radius: 8
      color: "#14243C"
      clickable: true
    }
  }
}
`

func TestASceneWiresByIdAndShootsHeadlessly(t *testing.T) {
	scene, err := Parse(sceneSrc, "web-light", 160, 100)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if !scene.OnClick("cta", func() {}) {
		t.Fatal("OnClick did not find the clickable box")
	}
	if scene.OnClick("doesNotExist", func() {}) {
		t.Fatal("OnClick reported success for a missing id")
	}
	if scene.SetText("doesNotExist", "x") {
		t.Fatal("OnText reported success for a missing id")
	}

	out := filepath.Join(t.TempDir(), "scene.png")
	if err := scene.Shot(160, 100, out); err != nil {
		t.Fatalf("shot: %v", err)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatalf("open shot: %v", err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatalf("decode shot: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 160 || b.Dy() != 100 {
		t.Fatalf("shot is %dx%d, want 160x100", b.Dx(), b.Dy())
	}
	// The page is cream; the CTA is navy. Sample both.
	r, g, b, _ := img.At(4, 4).RGBA()
	if r>>8 < 0xE0 || g>>8 < 0xD0 || b>>8 > 0xF0 {
		t.Fatalf("page corner is #%02X%02X%02X, want the cream fill", r>>8, g>>8, b>>8)
	}
	pr, _, pb, _ := img.At(60, 36).RGBA()
	if pr>>8 > 0x40 || pb>>8 < 0x20 {
		t.Fatalf("CTA centre is #%02X..%02X, want the navy fill", pr>>8, pb>>8)
	}
}

// TestAMissingIdIsAReportedMiss is the anti-silence rule: a scene whose wiring points
// at a typo must say so, because the symptom is a control that looks alive and does
// nothing.
func TestAMissingIdIsAReportedMiss(t *testing.T) {
	scene, err := Parse(sceneSrc, "web-light", 160, 100)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if scene.Get("nope") != nil {
		t.Fatal("Get returned a widget for a missing id")
	}
}
