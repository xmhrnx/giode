package securestore

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/xmhrnx/giode/internal/platform"
	"golang.org/x/sys/windows"
)

// secureFile is where name's value lives: a DPAPI-encrypted file in
// the user's config directory, named by a hash so names needn't be
// valid file names.
func secureFile(service, name string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	h := sha256.Sum256([]byte(name))
	return filepath.Join(dir, service, "securestore", hex.EncodeToString(h[:])), nil
}

func blob(b []byte) *windows.DataBlob {
	if len(b) == 0 {
		return &windows.DataBlob{}
	}
	return &windows.DataBlob{Size: uint32(len(b)), Data: &b[0]}
}

func dpapi(in []byte, protect bool) ([]byte, error) {
	var out windows.DataBlob
	var err error
	if protect {
		err = windows.CryptProtectData(blob(in), nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	} else {
		err = windows.CryptUnprotectData(blob(in), nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	}
	if err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}

func Set(service, name string, value []byte) error {
	path, err := secureFile(service, name)
	if err != nil {
		return err
	}
	sealed, err := dpapi(value, true)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, sealed, 0o600)
}

func Get(service, name string) ([]byte, error) {
	path, err := secureFile(service, name)
	if err != nil {
		return nil, err
	}
	sealed, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, platform.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return dpapi(sealed, false)
}

func Delete(service, name string) error {
	path, err := secureFile(service, name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
