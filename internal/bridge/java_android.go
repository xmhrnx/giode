package bridge

import "github.com/xmhrnx/giode/android"

// The Java helpers giode's platform packages use; see package android.
var (
	CallVoid   = android.CallVoid
	CallString = android.CallString
	GoString   = android.GoString
)
