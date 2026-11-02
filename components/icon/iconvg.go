package icon

import (
	"fmt"
	"sync"
	"unsafe"

	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

// IconVGIcon is a decoded IconVG icon, reusable across frames.
type IconVGIcon struct {
	icon *widget.Icon
}

// LoadIconVG decodes IconVG data (Gio's native vector icon format, as
// produced by golang.org/x/exp/shiny/materialdesign/icons).
func LoadIconVG(data []byte) (*IconVGIcon, error) {
	ic, err := widget.NewIcon(data)
	if err != nil {
		return nil, fmt.Errorf("giode: invalid iconvg: %w", err)
	}
	return &IconVGIcon{icon: ic}, nil
}

// Icon renders the icon, sized by styles.Width/Height (default 24dp)
// and colored by styles.Color.
func (i *IconVGIcon) Icon(st ...styles.Styles) elements.Element {
	return &iconvgEl{icon: i.icon, st: first(st)}
}

// iconvgCache keeps decoded IconVG data keyed by the backing array, so
// IconVG can be called with the same []byte every frame without
// re-decoding it.
var (
	iconvgMu    sync.Mutex
	iconvgCache = map[iconvgKey]*widget.Icon{}
)

type iconvgKey struct {
	ptr *byte
	n   int
}

// IconVG renders IconVG data. Decoding is cached per byte slice; invalid
// data renders as empty space. Prefer LoadIconVG to handle decoding
// errors.
func IconVG(data []byte, st ...styles.Styles) elements.Element {
	key := iconvgKey{unsafe.SliceData(data), len(data)}
	iconvgMu.Lock()
	ic, ok := iconvgCache[key]
	if !ok {
		ic, _ = widget.NewIcon(data)
		if len(iconvgCache) > 1024 {
			iconvgCache = map[iconvgKey]*widget.Icon{}
		}
		iconvgCache[key] = ic
	}
	iconvgMu.Unlock()
	if ic == nil {
		return &missingEl{st: first(st)}
	}
	return &iconvgEl{icon: ic, st: first(st)}
}

// iconvgEl adapts widget.Icon to an element.
type iconvgEl struct {
	icon *widget.Icon
	st   styles.Styles
}

// Layout renders the icon at the given size.
func (i *iconvgEl) Layout(gtx layout.Context) layout.Dimensions {
	size, ok := iconSize(gtx, i.st)
	if !ok {
		return layout.Dimensions{}
	}
	gtx.Constraints = layout.Exact(size)
	return i.icon.Layout(gtx, elements.ResolveColor(i.st.Color))
}
