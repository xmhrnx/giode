//go:build js

package giode

import (
	"sync"
	"syscall/js"
)

var (
	audioOnce sync.Once
	audioCtx  js.Value
)

// playSound decodes and plays the WAV with Web Audio. Browsers start
// audio only after the user has interacted with the page.
func playSound(wav []byte, _ wavInfo) error {
	audioOnce.Do(func() {
		ctor := js.Global().Get("AudioContext")
		if ctor.IsUndefined() {
			ctor = js.Global().Get("webkitAudioContext")
		}
		if !ctor.IsUndefined() {
			audioCtx = ctor.New()
		}
	})
	if audioCtx.IsUndefined() || audioCtx.IsNull() {
		return ErrNotSupported
	}
	arr := js.Global().Get("Uint8Array").New(len(wav))
	js.CopyBytesToJS(arr, wav)
	if audioCtx.Get("state").String() == "suspended" {
		audioCtx.Call("resume")
	}
	var onDecoded, onError js.Func
	onDecoded = js.FuncOf(func(this js.Value, args []js.Value) any {
		src := audioCtx.Call("createBufferSource")
		src.Set("buffer", args[0])
		src.Call("connect", audioCtx.Get("destination"))
		src.Call("start")
		onDecoded.Release()
		onError.Release()
		return nil
	})
	onError = js.FuncOf(func(this js.Value, args []js.Value) any {
		onDecoded.Release()
		onError.Release()
		return nil
	})
	audioCtx.Call("decodeAudioData", arr.Get("buffer")).Call("then", onDecoded, onError)
	return nil
}
