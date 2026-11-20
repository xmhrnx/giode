package giode

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/xmhrnx/giode/internal/platform"
	"github.com/xmhrnx/giode/internal/platform/securestore"
)

// ErrNotFound is returned by SecureStore.Get for a name with no value.
var ErrNotFound = platform.ErrNotFound

// SecureStorage keeps small secrets - API keys, tokens - in the
// platform's credential store rather than a file: the Keychain on iOS
// and macOS (accessible after first unlock, this device only, on iOS),
// the Android Keystore (values encrypted with a key that never leaves
// the device), the Secret Service (GNOME Keyring, KWallet) on Linux,
// and DPAPI-encrypted files on Windows. Values copied off the device,
// such as in a backup, can't be read elsewhere.
//
// Use the SecureStore value; its methods may be called from any
// goroutine, but can block (a Linux keyring may prompt to unlock), so
// keep them off the UI goroutine.
type SecureStorage struct {
	// Service names the app in the credential store, keeping its
	// entries apart from other apps'. It defaults to the executable's
	// name; set it before first use to something stable, such as a
	// reverse-DNS app ID.
	Service string
}

// SecureStore is the app's secure storage; see SecureStorage.
var SecureStore = &SecureStorage{}

func (s *SecureStorage) service() string {
	if s.Service != "" {
		return s.Service
	}
	exe, err := os.Executable()
	if err != nil {
		return "giode"
	}
	return strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))
}

// Set stores value under name, replacing any previous value.
func (s *SecureStorage) Set(name string, value []byte) error {
	return securestore.Set(s.service(), name, value)
}

// Get returns the value stored under name, or ErrNotFound.
func (s *SecureStorage) Get(name string) ([]byte, error) {
	return securestore.Get(s.service(), name)
}

// Delete removes the value stored under name; deleting a missing name
// is not an error.
func (s *SecureStorage) Delete(name string) error {
	return securestore.Delete(s.service(), name)
}
