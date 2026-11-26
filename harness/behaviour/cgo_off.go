//go:build !cgo

package behaviour

// cgoEnabled reports whether this test binary was built with cgo; see
// cgo_on.go. The visual suite is expected to run with cgo off.
const cgoEnabled = false
