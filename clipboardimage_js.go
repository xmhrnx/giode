//go:build js

package giode

import (
	"context"
	"errors"
	"image"
	"strings"
	"syscall/js"
)

// readClipboardImage reads the browser's clipboard with the async
// Clipboard API. The page must be focused, and the browser may ask the
// user first.
func (a *App) readClipboardImage(ctx context.Context) (image.Image, error) {
	cb := js.Global().Get("navigator").Get("clipboard")
	if cb.IsUndefined() || cb.Get("read").IsUndefined() {
		return nil, ErrNotSupported
	}
	items, err := awaitJS(ctx, cb.Call("read"))
	if err != nil {
		return nil, err
	}
	for i := 0; i < items.Length(); i++ {
		item := items.Index(i)
		types := item.Get("types")
		for j := 0; j < types.Length(); j++ {
			t := types.Index(j).String()
			if !strings.HasPrefix(t, "image/") {
				continue
			}
			blob, err := awaitJS(ctx, item.Call("getType", t))
			if err != nil {
				return nil, err
			}
			buf, err := awaitJS(ctx, blob.Call("arrayBuffer"))
			if err != nil {
				return nil, err
			}
			arr := js.Global().Get("Uint8Array").New(buf)
			if arr.Length() > maxClipboardImage {
				return nil, errors.New("giode: the clipboard image is too large")
			}
			data := make([]byte, arr.Length())
			js.CopyBytesToGo(data, arr)
			return decodeClipboardImage(data)
		}
	}
	return nil, nil
}

// awaitJS waits for a JavaScript promise.
func awaitJS(ctx context.Context, p js.Value) (js.Value, error) {
	type result struct {
		v   js.Value
		err error
	}
	ch := make(chan result, 1)
	var then, catch js.Func
	then = js.FuncOf(func(this js.Value, args []js.Value) any {
		ch <- result{v: args[0]}
		then.Release()
		catch.Release()
		return nil
	})
	catch = js.FuncOf(func(this js.Value, args []js.Value) any {
		msg := "the browser refused"
		if len(args) > 0 && !args[0].IsUndefined() {
			msg = args[0].Call("toString").String()
		}
		ch <- result{err: errors.New("giode: reading the clipboard: " + msg)}
		then.Release()
		catch.Release()
		return nil
	})
	p.Call("then", then).Call("catch", catch)
	select {
	case r := <-ch:
		return r.v, r.err
	case <-ctx.Done():
		return js.Undefined(), ctx.Err()
	}
}
