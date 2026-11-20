//go:build !windows && !darwin && !(linux && !android) && !js

package giode

import "github.com/xmhrnx/giode/styles/properties"

func systemHighContrast() (bool, error) { return false, ErrNotSupported }

func systemContrastPalettes() (light, dark *properties.Palette, ok bool) { return nil, nil, false }

func systemBoldText() (bool, error) { return false, ErrNotSupported }

func systemTextScale() (float32, error) { return 1, ErrNotSupported }

// gioScalesText: Gio applies Android's font scale itself.
func gioScalesText(a *App) bool { return true }
