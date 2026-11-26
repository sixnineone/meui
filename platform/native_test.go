package platform

import "testing"

// TestJNIHandleIsItsOwnHandle pins the escape hatch's contract: the handle's
// Kind names the platform so an app can refuse to mis-cast, and Handle returns
// the JNIHandle itself so the app writes the cast it means.
func TestJNIHandleIsItsOwnHandle(t *testing.T) {
	var n Native = JNIHandle{Env: nil, Activity: nil}
	if got := n.Kind(); got != "android/jni" {
		t.Errorf("JNIHandle.Kind() = %q, want \"android/jni\"", got)
	}
	h, ok := n.Handle().(JNIHandle)
	if !ok {
		t.Fatalf("JNIHandle.Handle() returned %T, want platform.JNIHandle", n.Handle())
	}
	// The zero value is a valid, if useless, handle: the point is that an app
	// must be able to reach the fields without any further ceremony.
	_ = h.Env
	_ = h.Activity
}
