//go:build !android

package mobile

// WatchLifecycle reports ErrUnsupported outside Android.
func WatchLifecycle(vm, ctx uintptr) error { return ErrUnsupported }

// Publish reports ErrUnsupported outside Android.
func Publish(vm, ctx uintptr, path, mime string) (string, error) { return "", ErrUnsupported }

// ShareText reports ErrUnsupported outside Android.
func ShareText(vm, ctx, view uintptr, text, subject, title string) error { return ErrUnsupported }

// ShareURI reports ErrUnsupported outside Android.
func ShareURI(vm, ctx, view uintptr, uri, mime, title string) error { return ErrUnsupported }

// ShareURIs reports ErrUnsupported outside Android.
func ShareURIs(vm, ctx, view uintptr, uris []string, mime, title string) error { return ErrUnsupported }

// SaveToDownloads reports ErrUnsupported outside Android.
func SaveToDownloads(vm, ctx uintptr, path, mime, folder string) (string, error) {
	return "", ErrUnsupported
}

// OpenDownloads reports ErrUnsupported outside Android.
func OpenDownloads(vm, ctx, view uintptr) error { return ErrUnsupported }

// SaveToGallery reports ErrUnsupported outside Android.
func SaveToGallery(vm, ctx uintptr, path, mime, album string) error { return ErrUnsupported }

// PickDocuments reports ErrUnsupported outside Android.
func PickDocuments(vm, ctx, view uintptr, mimes []string, multiple bool, limit int, media bool) (<-chan Documents, error) {
	return nil, ErrUnsupported
}

// PickPhotos reports ErrUnsupported outside Android.
func PickPhotos(vm, ctx, view uintptr, maxSide, limit int) (<-chan Photos, error) {
	return nil, ErrUnsupported
}

// OpenURL reports ErrUnsupported outside Android.
func OpenURL(vm, ctx, view uintptr, url string) error { return ErrUnsupported }

// Haptic reports ErrUnsupported outside Android.
func Haptic(vm, ctx, view uintptr, c int) error { return ErrUnsupported }

// KeepScreenOn reports ErrUnsupported outside Android.
func KeepScreenOn(vm, ctx, view uintptr, on bool) error { return ErrUnsupported }

// SecureSet reports ErrUnsupported outside Android.
func SecureSet(vm, ctx uintptr, name string, value []byte) error { return ErrUnsupported }

// PlaySound plays 16-bit PCM samples once (Android only).
func PlaySound(vm, ctx uintptr, pcm []byte, rate, channels int) error { return ErrUnsupported }

// SecureGet reports ErrUnsupported outside Android.
func SecureGet(vm, ctx uintptr, name string) ([]byte, bool, error) { return nil, false, ErrUnsupported }

// SecureDelete reports ErrUnsupported outside Android.
func SecureDelete(vm, ctx uintptr, name string) error { return ErrUnsupported }

// PickPhoto reports ErrUnsupported outside Android.
func PickPhoto(vm, ctx, view uintptr, maxSide int, camera bool) (<-chan Photo, error) {
	return nil, ErrUnsupported
}

// Dictate reports ErrUnsupported outside Android.
func Dictate(vm, ctx, view uintptr, lang string) (int, <-chan Speech, error) {
	return 0, nil, ErrUnsupported
}

// StopDictation reports ErrUnsupported outside Android.
func StopDictation(vm, ctx uintptr, id int) error { return ErrUnsupported }

// KeepAliveDeclared reports ErrUnsupported outside Android.
func KeepAliveDeclared(vm, ctx uintptr) (bool, error) { return false, ErrUnsupported }

// StartKeepAlive reports ErrUnsupported outside Android.
func StartKeepAlive(vm, ctx uintptr, reason string) error { return ErrUnsupported }

// StopKeepAlive reports ErrUnsupported outside Android.
func StopKeepAlive(vm, ctx uintptr) error { return ErrUnsupported }

// ListenShares reports ErrUnsupported outside Android.
func ListenShares(vm, ctx uintptr, handler func(Shared)) error { return ErrUnsupported }

// ClipboardImageURI reports ErrUnsupported outside Android.
func ClipboardImageURI(vm, ctx uintptr) (string, error) { return "", ErrUnsupported }

// ReadURI reports ErrUnsupported outside Android.
func ReadURI(vm, ctx uintptr, uri string, max int) ([]byte, error) { return nil, ErrUnsupported }
