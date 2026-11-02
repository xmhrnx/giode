package main

import "github.com/xmhrnx/giode"

// checkboxPage holds the checkbox, created once so its checked state
// persists across frames.
type checkboxPage struct {
	checkbox giode.Element
}

func NewCheckboxPage() *checkboxPage {
	return &checkboxPage{checkbox: giode.Input.Checkbox("CheckBox")}
}

func (p *checkboxPage) View() giode.Element {
	return giode.Stack(boxStyle, p.checkbox)
}
