// Command custom demonstrates creating a custom component with the
// component kit: a clickable star rating built on the stateless
// elements, Gio's clickable and the kit helpers.
package main

import (
	"fmt"
	"image"
	"log"

	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/components/icon"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

var gold = "#fac815"

// Rating is a custom stateful component: five clickable stars.
type Rating struct {
	value    int
	stars    [5]widget.Clickable
	onChange func(v int)
	st       styles.Styles
}

// NewRating returns a Rating with the default value 3.
func NewRating() *Rating {
	return &Rating{value: 3}
}

// Value returns the current rating.
func (r *Rating) Value() int { return r.value }

// OnChange sets the change handler and returns r for chaining.
func (r *Rating) OnChange(fn func(v int)) *Rating {
	r.onChange = fn
	return r
}

// Styles replaces the rating styles and returns r for chaining.
func (r *Rating) Styles(st styles.Styles) *Rating {
	r.st = st
	return r
}

// Layout implements elements.Element, so a *Rating can be placed
// anywhere an element is expected.
func (r *Rating) Layout(gtx layout.Context) layout.Dimensions {
	// Defaults first, user styles on top.
	st := styles.Merge(styles.Styles{Gap: 4, Color: gold}, r.st)

	var kids []elements.Element
	for i := range r.stars {
		kids = append(kids, &starEl{rating: r, index: i, st: st})
	}
	return elements.Row(st, kids...).Layout(gtx)
}

// starEl is a stateless element for one star; the rating owns the
// clickable state.
type starEl struct {
	rating *Rating
	index  int
	st     styles.Styles
}

// Layout renders one star.
func (s *starEl) Layout(gtx layout.Context) layout.Dimensions {
	click := &s.rating.stars[s.index]
	if click.Clicked(gtx) {
		s.rating.value = s.index + 1
		if s.rating.onChange != nil {
			s.rating.onChange(s.rating.value)
		}
	}
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		col := s.st.Color
		if s.index >= s.rating.value {
			col = kit.Alpha(col, 0.25) // unselected stars are dim
		}
		return starIcon.Icon(styles.Styles{Color: col, Width: 24, Height: 24}).Layout(gtx)
	})
}

// starIcon is the app's own icon: Giode bundles no icon set.
var starIcon = icon.Path{Data: "M12 2L15 9L22 9.5L16.5 14L18 21L12 17L6 21L7.5 14L2 9.5L9 9Z"}

func main() {
	giode.UseGofont()

	rating := NewRating().OnChange(func(v int) {
		log.Printf("rated %d/5", v)
	})

	app := giode.New(giode.Config{
		Title:      "Custom component",
		Size:       image.Pt(480, 240),
		Background: giode.HexColor("#0f172a"),
	})

	if err := app.Run(func() giode.Element {
		return giode.Box(
			giode.Styles{
				FlexDirection: giode.FlexDirectionColumn,
				Align:         giode.AlignCenter,
				Justify:       giode.JustifyCenter,
				Gap:           12,
				Width:         -1,
				Height:        -1,
				Color:         giode.HexColor("#f8fafc"),
			},
			giode.H3("Rate this app", giode.Styles{Color: giode.HexColor("#f8fafc")}),
			rating,
			giode.Text(fmt.Sprintf("Your rating: %d/5", rating.Value()), giode.Styles{Color: giode.HexColor("#94a3b8")}),
		)
	}); err != nil {
		log.Fatal(err)
	}
}
