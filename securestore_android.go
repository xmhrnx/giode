package giode

import (
	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/mobile"
)

func secureSet(service, name string, value []byte) error {
	return mobile.SecureSet(app.JavaVM(), app.AppContext(), service+"/"+name, value)
}

func secureGet(service, name string) ([]byte, error) {
	v, ok, err := mobile.SecureGet(app.JavaVM(), app.AppContext(), service+"/"+name)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	return v, nil
}

func secureDelete(service, name string) error {
	return mobile.SecureDelete(app.JavaVM(), app.AppContext(), service+"/"+name)
}
