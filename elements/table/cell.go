package table

import (
	"github.com/xmhrnx/giode/elements/box"
	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// TableCell wraps content as one cell: st.Padding defaults to a
// comfortable 12x8 if unset.
func TableCell(st styles.Styles, content base.Element) *box.BoxElement {
	cst := styles.Merge(styles.Styles{Padding: properties.SymmetricInset(12, 8)}, st)
	return box.Box(cst, content)
}
