#!/bin/sh
# check-nocgo — fail if cgo creeps into the desktop dependency graph.
#
# The desktop dev loop is CGO_ENABLED=0, and that only stays true if two rules
# hold: no file of ours imports "C", and no module's CGO_ENABLED=0 dependency
# graph contains runtime/cgo.
set -eu
cd "$(dirname "$0")/.."
fail=0
if CGO_ENABLED=0 go list -deps ./... 2>/dev/null | grep -qx 'runtime/cgo'; then
	echo "FAIL: runtime/cgo is in the CGO_ENABLED=0 graph"
	fail=1
fi
for f in $(grep -rlE '^import "C"$|^#cgo' --include='*.go' . 2>/dev/null || true); do
	case "$f" in
	./.git/* | */vendor/*) continue ;;
	esac
	echo "FAIL: $f imports cgo ($(grep -cE '^//go:build.*cgo' "$f") cgo build tags)"
	fail=1
done
[ "$fail" -eq 0 ] && echo "ok: the desktop graph is cgo-free"
exit "$fail"
