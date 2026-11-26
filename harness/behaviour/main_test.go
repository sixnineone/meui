package behaviour

import (
	"github.com/sixnineone/mobilegles"
	"os"
	"testing"
)

// TestMain tears SDL down after the run: windowed visual tests leave
// SDL subsystems live, and the process segfaulted at exit without an
// explicit shutdown.
func TestMain(m *testing.M) {
	// There is no per-test shutdown hook: mobilegles.Shutdown runs once, here,
	// after every test has returned. (An earlier version of this comment named
	// an OnLoopEnd hook that does not exist anywhere in the tree.)
	code := m.Run()
	mobilegles.Shutdown() // SDL_Quit on the main thread; after Run returned
	os.Exit(code)
}
