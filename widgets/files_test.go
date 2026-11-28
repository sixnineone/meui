package widgets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadTextFileRefusesNonRegularFiles(t *testing.T) {
	if _, err := readTextFile("/dev/null"); err == nil {
		t.Fatal("/dev/null is not a regular file and must be refused")
	}
}

func TestReadTextFileReadsRegularFiles(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.go")
	if err := os.WriteFile(p, []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := readTextFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "package a\n" {
		t.Fatalf("got %q", data)
	}
}
