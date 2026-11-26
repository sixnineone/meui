package format

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDemoCorpusIsCanonical guards the committed QML corpus: every .qml
// file under examples/enterprise must already be canonically formatted,
// so formatting drift fails a test instead of accumulating in review
// diffs.
//
// To fix a failure, run:
//
//	go run ./cmd/tools/qmlfmt --indent 2 --max-line 100 -w examples/enterprise/*.qml
func TestDemoCorpusIsCanonical(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "examples", "enterprise")
	files, err := filepath.Glob(filepath.Join(dir, "*.qml"))
	if err != nil || len(files) == 0 {
		t.Skipf("demo corpus not found at %s", dir)
	}

	opts := ProjectOptions()
	for _, path := range files {
		name := filepath.Base(path)
		src, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		out, err := FormatWithOptions(src, opts)
		if err != nil {
			// Every file in the corpus is formattable today; a new file
			// the formatter cannot handle should be a visible decision,
			// not a silent skip.
			t.Errorf("%s: formatter cannot parse it: %v", name, err)
			continue
		}
		if string(out) != string(src) {
			t.Errorf("%s: not canonically formatted (run qmlfmt --indent 2 --max-line 100 -w)", name)
		}
	}
}
