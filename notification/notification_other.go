//go:build !linux && !darwin && !windows

package notification

import "github.com/xmhrnx/giode"

func listen(*giode.App) error               { return nil }
func show(n Notification, ch Channel) error { return giode.ErrNotSupported }
func remove(id string) error                { return giode.ErrNotSupported }
func removeAll() error                      { return giode.ErrNotSupported }
func setBadge(n int) error                  { return giode.ErrNotSupported }
func defineChannel(Channel) error           { return nil }
func defineCategory(string, []Action) error { return nil }
