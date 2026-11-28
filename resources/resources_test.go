package resources

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func testBundle(t *testing.T) *Bundle {
	t.Helper()
	b := New()
	fsys := fstest.MapFS{
		"resources/scenes/main.qml":    {Data: []byte("Window {}")},
		"resources/scenes/img/a.png":   {Data: []byte("a")},
		"resources/themes/dark.json":   {Data: []byte("{}")},
		"resources/themes/light.json":  {Data: []byte("{}")},
		"resources/icons/document.svg": {Data: []byte("<svg/>")},
		"resources/root-only.txt":      {Data: []byte("root")},
	}
	if err := b.Mount("res", fsys, "resources"); err != nil {
		t.Fatal(err)
	}
	return b
}

func TestReadFileAndOpen(t *testing.T) {
	b := testBundle(t)
	data, err := b.ReadFile("res://scenes/main.qml")
	if err != nil || string(data) != "Window {}" {
		t.Fatalf("ReadFile = %q, %v", data, err)
	}
	f, err := b.Open("res://themes/dark.json")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if !b.Exists("res://root-only.txt") {
		t.Fatal("Exists said no")
	}
	if b.Exists("res://missing") {
		t.Fatal("Exists said yes for a missing file")
	}
}

func TestNamesCannotEscape(t *testing.T) {
	b := testBundle(t)
	for _, uri := range []string{
		"res://../etc/passwd",
		"res://scenes/../../etc/passwd",
		"res:///etc/passwd",
		"res://",
		"res://a\\b",
		"res://a\x00b",
		"nonsense",
	} {
		if _, err := b.ReadFile(uri); err == nil {
			t.Errorf("ReadFile(%q) was allowed", uri)
		}
	}
}

func TestReadDirMergesLayers(t *testing.T) {
	b := testBundle(t)
	entries, err := b.ReadDir("res://themes")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) != 2 || names[0] != "dark.json" || names[1] != "light.json" {
		t.Fatalf("entries = %v", names)
	}
}

func TestExternalMountAndPrecedence(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "themes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "themes", "dark.json"), []byte("external"), 0o644); err != nil {
		t.Fatal(err)
	}
	b := testBundle(t)
	if err := b.MountExternal("res", dir); err != nil {
		t.Fatal(err)
	}
	// Embedded-first by default: the bundle wins.
	if data, _ := b.ReadFile("res://themes/dark.json"); string(data) == "external" {
		t.Fatal("external shadowed the bundle without SetOrder")
	}
	if err := b.SetOrder("res", ExternalFirst); err != nil {
		t.Fatal(err)
	}
	if data, _ := b.ReadFile("res://themes/dark.json"); string(data) != "external" {
		t.Fatalf("external did not win: %q", data)
	}
}

func TestExternalMountRefusesSymlinkEscape(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "inside.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "escape.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	b := New()
	if err := b.Mount("ext", os.DirFS(dir), ""); err != nil {
		t.Fatal(err)
	}
	// os.DirFS follows the symlink; MountExternal must not.
	b2 := New()
	if err := b2.MountExternal("ext", dir); err != nil {
		t.Fatal(err)
	}
	if b2.Exists("ext://escape.txt") {
		t.Fatal("MountExternal followed a symlink outside its root")
	}
	if !b2.Exists("ext://inside.txt") {
		t.Fatal("MountExternal refused a file inside its root")
	}
}

func TestResolveRelativeToScene(t *testing.T) {
	b := testBundle(t)
	if got, ok := b.Resolve("res://scenes/main.qml", "img/a.png"); !ok || got != "res://scenes/img/a.png" {
		t.Fatalf("Resolve = %q, %v", got, ok)
	}
	if got, ok := b.Resolve("res://scenes/main.qml", "root-only.txt"); !ok || got != "res://root-only.txt" {
		t.Fatalf("root fallback = %q, %v", got, ok)
	}
	if got, ok := b.Resolve("res://scenes/main.qml", "/tmp/x.png"); ok || got != "/tmp/x.png" {
		t.Fatalf("disk path rewritten: %q, %v", got, ok)
	}
}

func TestReadFileLimit(t *testing.T) {
	b := testBundle(t)
	if _, err := b.ReadFileLimit("res://root-only.txt", 2); err == nil {
		t.Fatal("limit was not enforced")
	}
}
