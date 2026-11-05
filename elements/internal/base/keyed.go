package base

import "sync"

type keyedKey struct{ scope, key any }

type keyedEntry struct {
	value any
	frame uint64
}

// DefaultRetainFrames is how many frames an entry may go unrequested
// before it's dropped. Frames only happen when something changes, so
// this is not a fixed time.
const DefaultRetainFrames = 60

var (
	keyedMu     sync.Mutex
	keyedStore  = map[keyedKey]*keyedEntry{}
	keyedRetain = uint64(DefaultRetainFrames)
	// keyedCaches sweeps every live KeyedCache; each func reports
	// false once its cache was garbage collected.
	keyedCaches  []func(frame uint64) bool
	keyedCacheMu sync.Mutex
)

// Keyed returns the state stored under (scope, key), creating it with
// init the first time it's asked for. Use it inside the view function
// to give each item of a dynamic list its own stateful component:
//
//	for _, f := range files {
//		row := giode.Keyed("files", f.ID, func() *giode.PressableElement {
//			return giode.Pressable(...)
//		})
//		rows = append(rows, row)
//	}
//
// scope separates unrelated lists that might share keys; any
// comparable values work for both. If a key was stored with a
// different type, init replaces it. Entries not asked for in
// DefaultRetainFrames frames are dropped (see SetKeyedRetention). Call
// it on the UI goroutine.
func Keyed[T any](scope, key any, init func() T) T {
	k := keyedKey{scope, key}
	keyedMu.Lock()
	e, ok := keyedStore[k]
	if ok {
		if v, ok := e.value.(T); ok {
			e.frame = frameNumber
			keyedMu.Unlock()
			retainKeyed(e.touch)
			return v
		}
	}
	keyedMu.Unlock()
	v := init()
	e = &keyedEntry{value: v, frame: frameNumber}
	keyedMu.Lock()
	keyedStore[k] = e
	keyedMu.Unlock()
	retainKeyed(e.touch)
	return v
}

// touch keeps the entry alive this frame.
func (e *keyedEntry) touch() {
	keyedMu.Lock()
	e.frame = frameNumber
	keyedMu.Unlock()
}

// ForgetKeyed drops the state stored under (scope, key) right away.
func ForgetKeyed(scope, key any) {
	keyedMu.Lock()
	delete(keyedStore, keyedKey{scope, key})
	keyedMu.Unlock()
}

// SetKeyedRetention sets how many frames Keyed entries survive without
// being asked for (minimum 1).
func SetKeyedRetention(frames int) {
	keyedMu.Lock()
	keyedRetain = uint64(max(1, frames))
	keyedMu.Unlock()
}

// sweepKeyed drops expired entries; EndFrame calls it.
func sweepKeyed() {
	keyedMu.Lock()
	for k, e := range keyedStore {
		if frameNumber-e.frame > keyedRetain {
			delete(keyedStore, k)
		}
	}
	keyedMu.Unlock()
	keyedCacheMu.Lock()
	live := keyedCaches[:0]
	for _, sweep := range keyedCaches {
		if sweep(frameNumber) {
			live = append(live, sweep)
		}
	}
	clear(keyedCaches[len(live):])
	keyedCaches = live
	keyedCacheMu.Unlock()
}
