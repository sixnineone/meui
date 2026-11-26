package platform

import "unsafe"

// The escape hatch.
//
// The seam deliberately does NOT abstract everything. BLE GATT, NFC, ARKit,
// manual camera controls and the like are either too platform-specific or too
// niche to give one honest API, and pretending otherwise would produce a
// lowest-common-denominator API that is worse than the platform's own.
//
// So there is exactly one intentional leak, it is opt-in, and it is named as
// what it is. An app that reaches through it is accepting that it is now writing
// platform-specific code — which is the honest trade, rather than forcing every
// app to pay for an abstraction almost none of them use.

// Native is an opaque handle to the backend's native platform object.
type Native interface {
	// Kind names what the handle is, so an app can fail clearly on the wrong
	// platform instead of mis-casting: "android/jni" or "ios/uikit".
	Kind() string
	// Handle returns the platform object; its concrete type depends on Kind.
	// See JNIHandle for Android.
	Handle() any
}

// JNIHandle is the Android escape hatch: the JNI environment for the calling
// thread and the Activity, so an app can call Java APIs the seam does not cover.
//
// Both are raw JNI pointers and the app is responsible for using them correctly:
// Env is a JNIEnv* valid only on the thread that asked for it, and Activity is a
// LOCAL reference, so an app that keeps it must make a global reference itself
// (NewGlobalRef) before the frame that produced it returns.
//
// This is deliberately not wrapped in helpers. A helper layer is what the seam
// already is; the point of the hatch is to hand over the real thing.
type JNIHandle struct {
	Env      unsafe.Pointer // JNIEnv*
	Activity unsafe.Pointer // jobject, the current Activity
}

// Kind implements Native.
func (JNIHandle) Kind() string { return "android/jni" }

// Handle implements Native. The value is the JNIHandle itself, so an app writes
// the cast it means:
//
//	if n, ok := app.Native(); ok && n.Kind() == "android/jni" {
//		jni := n.Handle().(platform.JNIHandle)
//		// call Java through jni.Env / jni.Activity
//	}
func (h JNIHandle) Handle() any { return h }
