package securestore

import (
	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/mobile"
	"github.com/xmhrnx/giode/internal/platform"
)

func Set(service, name string, value []byte) error {
	return mobile.SecureSet(app.JavaVM(), app.AppContext(), service+"/"+name, value)
}

func Get(service, name string) ([]byte, error) {
	v, ok, err := mobile.SecureGet(app.JavaVM(), app.AppContext(), service+"/"+name)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, platform.ErrNotFound
	}
	return v, nil
}

func Delete(service, name string) error {
	return mobile.SecureDelete(app.JavaVM(), app.AppContext(), service+"/"+name)
}
