package main

import (
	"github.com/xmhrnx/giode"
	giodeinput "github.com/xmhrnx/giode/components/input"
)

// inputPage holds the text field, created once so its content
// persists across frames. Kept as the concrete type (rather than
// giode.Element) so View can read back the live text every frame.
type inputPage struct {
	field *giodeinput.Input
}

func NewInputPage() *inputPage {
	return &inputPage{
		field: giode.Input.Text("Type here", giode.Styles{Color: "white", BorderColor: "gray", Width: 100}),
	}
}

func (p *inputPage) View() giode.Element {
	return giode.Stack(boxStyle,
		giode.Text("Input", textStyle),
		p.field,
		giode.Text(p.field.Text(), giode.Styles{Color: "#94a3b8"}),
	)
}
