package menu

import (
	"image/color"

	"github.com/xmhrnx/giode/components/kit"
)

var (
	defaultBackground = color.NRGBA{R: 0x1e, G: 0x29, B: 0x3b, A: 0xff}
	defaultText       = color.NRGBA{R: 0xe2, G: 0xe8, B: 0xf0, A: 0xff}
	defaultDisabled   = color.NRGBA{R: 0x64, G: 0x74, B: 0x8b, A: 0xff}
	defaultHover      = color.NRGBA{R: 0x33, G: 0x41, B: 0x55, A: 0xff}
	defaultSeparator  = color.NRGBA{R: 0x47, G: 0x55, B: 0x69, A: 0xff}
	// defaultBorder and defaultShadow outline the panel; menu.New's
	// styles replace them (BorderColor "transparent" for none).
	defaultBorder = kit.PopupBorder
	defaultShadow = kit.PopupShadow
)
