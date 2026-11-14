//go:build js

package settings

import "syscall/js"

// storageKey is the local storage item holding the document.
const storageKey = "giode.settings"

// localBackend keeps the document in the browser's local storage.
type localBackend struct{}

func defaultBackend() backend { return localBackend{} }

func storage() js.Value {
	ls := js.Global().Get("localStorage")
	if ls.IsUndefined() || ls.IsNull() {
		return js.Value{}
	}
	return ls
}

func (localBackend) load() ([]byte, error) {
	ls := storage()
	if ls.IsUndefined() {
		return nil, ErrNoStorage
	}
	v := ls.Call("getItem", storageKey)
	if v.IsNull() || v.IsUndefined() {
		return nil, nil
	}
	return []byte(v.String()), nil
}

func (localBackend) save(b []byte) (err error) {
	ls := storage()
	if ls.IsUndefined() {
		return ErrNoStorage
	}
	// setItem throws when the storage is full or disabled.
	defer func() {
		if r := recover(); r != nil {
			err = ErrNoStorage
		}
	}()
	ls.Call("setItem", storageKey, string(b))
	return nil
}
