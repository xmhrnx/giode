package main

import (
	"fmt"

	"github.com/xmhrnx/giode"
)

// paginationPage holds the control, created once so its current page
// persists across frames.
type paginationPage struct {
	page  int
	pager giode.Element
}

func NewPaginationPage() *paginationPage {
	p := &paginationPage{page: 1}
	p.pager = giode.Pagination(20).OnChange(func(page int) { p.page = page })
	return p
}

func (p *paginationPage) View() giode.Element {
	return giode.Stack(boxStyle,
		giode.Text("Pagination", textStyle),
		giode.Text(fmt.Sprintf("Page %d of 20", p.page), giode.Styles{Color: "white", FontSize: 12}),
		p.pager,
	)
}
