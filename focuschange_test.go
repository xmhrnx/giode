package giode

import (
	"testing"

	"gioui.org/app"
)

func TestOnFocusChange(t *testing.T) {
	var got []bool
	a := &App{cfg: Config{OnFocusChange: func(f bool) { got = append(got, f) }}, themeCheck: make(chan struct{}, 1)}
	a.handleConfig(app.Config{Focused: true})
	a.handleConfig(app.Config{Focused: true})
	a.handleConfig(app.Config{Focused: false})
	if len(got) != 2 || !got[0] || got[1] {
		t.Errorf("OnFocusChange calls %v, want [true false]", got)
	}
}
