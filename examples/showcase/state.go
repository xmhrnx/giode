package main

import (
	"fmt"
	"time"

	"github.com/xmhrnx/giode"
)

// statePage demonstrates giode.State: counter is shared global state,
// incremented both by a button click (inside the render loop) and by
// a background goroutine (outside it, simulating a timer/network
// callback) - State.Set requests a redraw either way, via the window
// Invalidate hookup App.New wires up.
type statePage struct {
	counter *giode.State[int]
	btn     giode.Element
}

func NewStatePage() *statePage {
	p := &statePage{counter: giode.NewState(0)}
	p.btn = giode.Button("Increment", giode.Styles{
		Background: "#3b82f6", Color: "white", BorderRadius: 6,
	}).OnClick(func() {
		p.counter.Update(func(v int) int { return v + 1 })
	})

	// A goroutine outside the render loop, updating the same State -
	// the case Set/Update's own redraw request exists for.
	go func() {
		for range time.Tick(3 * time.Second) {
			p.counter.Update(func(v int) int { return v + 1 })
		}
	}()

	return p
}

func (p *statePage) View() giode.Element {
	return giode.Stack(boxStyle,
		giode.Text("Global state", textStyle),
		giode.Text(fmt.Sprintf("Count: %d", p.counter.Get()), giode.Styles{Color: "white", FontSize: 14}),
		giode.Text("+1 every 3s in the background too", giode.Styles{Color: "#94a3b8", FontSize: 10}),
		p.btn,
	)
}
