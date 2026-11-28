package widgets

import (
	"fmt"
	"os"
)

// maxTextFileBytes bounds a text file loaded through a filePath property. The
// property is settable from a scene, and scenes are input; 16 MiB is far above
// any source or document a widget should display, and finite, unlike /dev/zero.
const maxTextFileBytes = 16 << 20

// readTextFile loads a regular file within maxTextFileBytes. Character devices
// and other non-regular files are refused: os.ReadFile on /dev/zero never
// returns, which is a hang reachable from one line of QML.
func readTextFile(path string) ([]byte, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, fmt.Errorf("widgets: %q is not a regular file", path)
	}
	if st.Size() > maxTextFileBytes {
		return nil, fmt.Errorf("widgets: %s is %d bytes; the limit is %d", path, st.Size(), maxTextFileBytes)
	}
	return os.ReadFile(path)
}
