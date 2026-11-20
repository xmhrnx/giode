//go:build !windows && !darwin && !(linux && !android) && !js

package prefs

import (
	"gioui.org/app"
	"github.com/xmhrnx/giode/internal/platform"
	"github.com/xmhrnx/giode/styles/properties"
)

func HighContrast() (bool, error) { return false, platform.ErrNotSupported }

func ContrastPalettes() (light, dark *properties.Palette, ok bool) { return nil, nil, false }

func BoldText() (bool, error) { return false, platform.ErrNotSupported }

func TextScale() (float32, error) { return 1, platform.ErrNotSupported }

// GioScalesText: Gio applies Android's font scale itself.
func GioScalesText(view app.ViewEvent) bool { return true }
