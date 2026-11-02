// Package icon renders icons you bring: Giode ships no icon set, so an
// app only carries the icons it actually uses, under whatever license
// it picked them. Load icons from SVG files (fill- and stroke-based
// sets alike), raw SVG path data, IconVG (Gio's native vector format)
// or plain images, and optionally register them by name to look them
// up anywhere:
//
//	//go:embed icons/*.svg
//	var iconFS embed.FS
//
//	func init() {
//		icon.MustRegisterFS(iconFS, "icons") // "icons/home.svg" -> "home"
//	}
//
//	icon.Named("home", styles.Styles{Color: "theme.text", Width: 20, Height: 20})
//
// All constructors are cheap to call every frame: documents are parsed
// once when loaded and path data is cached. Size comes from
// styles.Width and styles.Height (default 24dp) and color from
// styles.Color (default black).
package icon

import (
	"image"

	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/styles"
)

// defaultSize is the icon size, in dp, when styles leave it unset.
const defaultSize = 24

func first(st []styles.Styles) styles.Styles {
	if len(st) > 0 {
		return st[0]
	}
	return styles.Styles{}
}

// iconSize resolves an icon's pixel size from Width/Height (in dp),
// defaulting to 24dp and making a single set axis square.
func iconSize(gtx layout.Context, st styles.Styles) (image.Point, bool) {
	w, h := int(st.Width), int(st.Height)
	switch {
	case w <= 0 && h <= 0:
		w, h = defaultSize, defaultSize
	case w <= 0:
		w = h
	case h <= 0:
		h = w
	}
	size := gtx.Constraints.Constrain(image.Pt(gtx.Dp(unit.Dp(w)), gtx.Dp(unit.Dp(h))))
	return size, size.X > 0 && size.Y > 0
}
