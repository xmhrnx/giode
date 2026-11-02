package giode

import (
	"testing"
	"time"

	"gioui.org/app"

	"github.com/xmhrnx/giode/styles/properties"
)

func TestWatchSystemTheme(t *testing.T) {
	defer properties.SetMode(properties.CurrentMode())
	dark := false
	defer func(f func() (bool, error)) { readSystemDark = f }(readSystemDark)
	readSystemDark = func() (bool, error) { return dark, nil }

	var changes []bool
	a := &App{window: new(app.Window), themeCheck: make(chan struct{}, 1), cfg: Config{
		FollowSystemTheme:   true,
		OnSystemThemeChange: func(d bool) { changes = append(changes, d) },
	}}
	properties.SetMode(properties.ModeDark)
	done := make(chan struct{})
	defer close(done)
	a.watchSystemTheme(done)
	if properties.CurrentMode() != properties.ModeLight {
		t.Fatal("startup didn't apply the light system theme")
	}
	dark = true
	a.checkSystemTheme()
	deadline := time.Now().Add(2 * time.Second)
	for len(changes) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
		a.runUICalls()
	}
	if len(changes) != 1 || !changes[0] || properties.CurrentMode() != properties.ModeDark {
		t.Errorf("after the switch: changes=%v mode=%v, want [true] and dark", changes, properties.CurrentMode())
	}
}
