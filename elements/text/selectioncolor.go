package text

import (
	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles/properties"
)

// selectionColor is the highlight behind selected text of color c.
func selectionColor(c properties.Color) properties.Color {
	col := base.ResolveColor(c)
	col.A = uint8(float32(col.A) * 0.3)
	return properties.CalcColorReverse(col)
}
