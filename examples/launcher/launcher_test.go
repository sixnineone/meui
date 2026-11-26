package main

// The launcher's one piece of logic worth pinning: it finds the module root from wherever
// it starts, because every card's command is relative to it. A launcher that cannot find
// the root can only say so in the status line — the test keeps that from happening when
// run from the example's own directory (which is where `go test` puts us).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTheLauncherFindsTheModuleRoot(t *testing.T) {
	root := moduleRoot()
	if root == "" {
		t.Fatal("moduleRoot found nothing from the example's directory")
	}
	b, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("the root has no go.mod: %v", err)
	}
	if !strings.Contains(string(b), "module github.com/sixnineone/meui") {
		t.Fatalf("moduleRoot returned %q, whose go.mod is a different module", root)
	}
	// And the examples it advertises are all where the command expects them.
	for _, ex := range examples {
		if fi, err := os.Stat(filepath.Join(root, "examples", ex.Dir)); err != nil || !fi.IsDir() {
			t.Errorf("example %q is listed but examples/%s is not there", ex.Title, ex.Dir)
		}
	}
}
