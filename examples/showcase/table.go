package main

import "github.com/xmhrnx/giode"

// TableShowcase is stateless: Table and its rows/cells are stateless
// elements, so it can be built fresh every frame like Text.
func TableShowcase() giode.Element {
	cellSt := giode.Styles{Color: "white", FontSize: 12}
	return giode.Stack(boxStyle,
		giode.Text("Table", textStyle),
		giode.Table(giode.Styles{Width: 220},
			giode.TableHeader(giode.Styles{Background: "#334155"},
				giode.TableCell(giode.Styles{}, giode.Text("Name", cellSt)),
				giode.TableCell(giode.Styles{}, giode.Text("Role", cellSt)),
			),
			giode.TableRow(giode.Styles{},
				giode.TableCell(giode.Styles{}, giode.Text("Ada", cellSt)),
				giode.TableCell(giode.Styles{}, giode.Text("Engineer", cellSt)),
			),
			giode.TableRow(giode.Styles{},
				giode.TableCell(giode.Styles{}, giode.Text("Grace", cellSt)),
				giode.TableCell(giode.Styles{}, giode.Text("Admiral", cellSt)),
			),
		),
	)
}
