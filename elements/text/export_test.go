package text

import "github.com/xmhrnx/giode/elements/internal/base"

// SelectedText is the selection of the area under key, for tests in
// text_test.
func SelectedText(key any) string {
	return base.Keyed("giode.selectionarea", key, newSelArea).selectedText()
}

// SelectedTextOfSelectable is the selection of the Selectable text
// under key.
func SelectedTextOfSelectable(key any) string {
	return base.Keyed("giode.selectable", key, newSelectState).sel.SelectedText()
}
