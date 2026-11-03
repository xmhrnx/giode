//go:build !android

package multicast

func acquire(string) (func(), error) { return func() {}, nil }
