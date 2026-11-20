//go:build !windows && !(linux && !android) && !(darwin && !ios)

package ctxmenu

import "errors"

func Install(string, string, []Action) error {
	return errors.New("giode: context actions are a desktop feature")
}

func Uninstall(string) error { return nil }
