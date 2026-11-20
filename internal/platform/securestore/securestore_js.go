//go:build js

package securestore

import "github.com/xmhrnx/giode/internal/platform"

func Set(service, name string, value []byte) error { return platform.ErrNotSupported }

func Get(service, name string) ([]byte, error) { return nil, platform.ErrNotSupported }

func Delete(service, name string) error { return platform.ErrNotSupported }
