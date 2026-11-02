package giode

import (
	"testing"

	"gioui.org/app"

	"github.com/xmhrnx/giode/styles/properties"
)

func TestSystemBars_FollowBackgroundAndTheme(t *testing.T) {
	defer func(v bool) { systemBarsSupported = v }(systemBarsSupported)
	systemBarsSupported = true
	defer properties.SetMode(properties.CurrentMode())

	a := &App{cfg: Config{Background: "theme.background"}, window: new(app.Window)}
	properties.SetMode(properties.ModeLight)
	a.updateSystemBars()
	light := a.statusColor
	if want := properties.CalcColor("theme.background"); light != want || a.navColor != want {
		t.Fatalf("bars = %v/%v, want the light background %v", light, a.navColor, want)
	}

	properties.SetMode(properties.ModeDark)
	a.updateSystemBars()
	if dark := properties.CalcColor("theme.background"); a.statusColor != dark || a.statusColor == light {
		t.Errorf("after SetMode(dark) status bar = %v, want the dark background %v", a.statusColor, dark)
	}
}

func TestSystemBars_Overrides(t *testing.T) {
	a := &App{cfg: Config{Background: "#000000", StatusBar: "#ff0000"}}
	status, nav := a.systemBarColors()
	if status != properties.CalcColor("#ff0000") || nav != properties.CalcColor("#000000") {
		t.Errorf("bars = %v/%v, want the StatusBar override and the background", status, nav)
	}
}
