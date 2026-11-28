package qml

import (
	"testing"
	"testing/fstest"

	"github.com/sixnineone/meui/resources"
)

func TestLoadBundle(t *testing.T) {
	fsys := fstest.MapFS{
		"res/scenes/main.qml": {Data: []byte(`Window { width: 200; height: 100 }`)},
	}
	b := resources.New()
	if err := b.Mount("res", fsys, "res"); err != nil {
		t.Fatal(err)
	}
	res, err := LoadBundle(b, "res://scenes/main.qml")
	if err != nil {
		t.Fatalf("LoadBundle: %v", err)
	}
	if res.Widget == nil {
		t.Fatal("no root widget")
	}
	if _, err := LoadBundle(b, "res://missing.qml"); err == nil {
		t.Fatal("missing scene accepted")
	}
	if _, err := LoadBundle(b, "res://../escape.qml"); err == nil {
		t.Fatal("traversal accepted")
	}
}
