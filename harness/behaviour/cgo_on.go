//go:build cgo

package behaviour

// cgoEnabled reports whether this test binary was built with cgo. The visual
// suite runs CGO_ENABLED=0: the toolkit reaches C through purego, and the cgo
// runtime's thread entry (threadentry/crosscall1) is the only source of the
// suite's segfaults — see REVIEW.md §6. Build-tagged, so the check is a
// constant and costs nothing.
const cgoEnabled = true
