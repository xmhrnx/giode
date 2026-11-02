package main

import "github.com/xmhrnx/giode"

var fruitOptions = []string{
	"Apple", "Apricot", "Banana", "Blueberry", "Cherry", "Cranberry", "Date", "Grape",
}

// autocompletePage holds the field, created once so its typed text
// and selection persist across frames.
type autocompletePage struct {
	field giode.Element
}

func NewAutocompletePage() *autocompletePage {
	return &autocompletePage{
		field: giode.Input.AutoComplete("Type a fruit...", fruitOptions, giode.Styles{
			Width: 180,
		}),
	}
}

func (p *autocompletePage) View() giode.Element {
	return giode.Stack(boxStyle,
		giode.Text("AutoComplete", textStyle),
		p.field,
	)
}
