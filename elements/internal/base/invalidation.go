package base

import "sync/atomic"

// cacheEpoch invalidates every cache at once when advanced.
var cacheEpoch atomic.Uint64

// SelectionVersion reports the version of the innermost selection
// area's selection, for caches inside it to notice a change. The
// selection element sets it from its init; it reports 0 outside any
// selection area.
var SelectionVersion = func() uint64 { return 0 }

// InvalidateCaches forces every MemoCache and Memo to rebuild on the
// next frame. Call it after changing something caches can't observe -
// a plain variable not passed as a dep, a component driven
// programmatically (Button.Click, Input.SetText) - when the change
// must show up inside a cached subtree. It is safe to call from any
// goroutine; it doesn't request a frame by itself.
func InvalidateCaches() {
	cacheEpoch.Add(1)
}

// --- Keyed memos -------------------------------------------------------------
