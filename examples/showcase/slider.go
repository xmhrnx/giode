package main

import (
	"github.com/xmhrnx/giode"
	giodeslider "github.com/xmhrnx/giode/components/slider"
)

// sliderPage holds the slider, created once so its dragged value
// persists across frames. Kept as the concrete type so other pages
// (progressPage) can read its live Value().
type sliderPage struct {
	slider *giodeslider.Slider
}

func NewSliderPage() *sliderPage {
	return &sliderPage{slider: giode.Input.Slider(giode.Styles{Color: "#3b82f6"})}
}

// Value returns the slider's current position in [0, 1].
func (p *sliderPage) Value() float32 {
	return p.slider.Value()
}

func (p *sliderPage) View() giode.Element {
	return giode.Stack(boxStyle,
		giode.Text("Slider", textStyle),
		giode.Box(giode.Styles{Width: 120}, p.slider),
	)
}
