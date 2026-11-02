package main

import "github.com/xmhrnx/giode"

type dropdownPage struct {
	dropdown giode.Element
}

func NewDropdownPage() *dropdownPage {
	return &dropdownPage{dropdown: giode.Input.Select(giode.Styles{}, "1", "2")}
}

func (p *dropdownPage) View() giode.Element {
	return giode.Stack(boxStyle, p.dropdown)
}
