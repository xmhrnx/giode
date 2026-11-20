package giode

import (
	"testing"
	"time"

	"gioui.org/app"
	"gioui.org/font"

	"github.com/xmhrnx/giode/styles/properties"
)

func TestA11yPrefs(t *testing.T) {
	defer properties.SetHighContrast(false)
	defer properties.SetBoldText(false)
	contrast, bold, scale := false, false, float32(1)
	defer func(c, b func() (bool, error), s func() (float32, error)) {
		readHighContrast, readBoldText, readTextScale = c, b, s
	}(readHighContrast, readBoldText, readTextScale)
	readHighContrast = func() (bool, error) { return contrast, nil }
	readBoldText = func() (bool, error) { return bold, nil }
	readTextScale = func() (float32, error) { return scale, nil }

	a := &App{window: new(app.Window), themeCheck: make(chan struct{}, 1), motionCheck: make(chan struct{}, 1), prefCheck: make(chan struct{}, 1)}
	done := make(chan struct{})
	defer close(done)
	a.watchA11yPrefs(done)
	text := func() any { return properties.ResolveColor("theme.muted") }
	normalMuted := text()
	if HighContrast() || BoldText() || a.TextScale() != 1 {
		t.Fatal("on at startup with the system's settings off")
	}

	contrast, bold, scale = true, true, 1.5
	a.checkSystemTheme()
	deadline := time.Now().Add(2 * time.Second)
	for !(HighContrast() && BoldText() && a.TextScale() == 1.5) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !HighContrast() || !BoldText() || a.TextScale() != 1.5 {
		t.Fatalf("the system's changes weren't followed: contrast %v, bold %v, scale %v", HighContrast(), BoldText(), a.TextScale())
	}
	if m := text(); m == normalMuted {
		t.Errorf("muted is still %s with high contrast", m)
	}
	if w := properties.FontWeight("").Weight(); w != font.SemiBold {
		t.Errorf("normal text with bold text = %v", w)
	}
	if w := properties.FontWeightBold.Weight(); w != font.Black {
		t.Errorf("bold text with bold text = %v", w)
	}

	a.SetContrast(PreferenceOff)
	a.SetBoldText(PreferenceOff)
	a.SetTextScale(2)
	if HighContrast() || BoldText() || a.TextScale() != 2 {
		t.Error("overrides weren't applied")
	}
	if a.appliedTextScale() != 2 {
		t.Errorf("applied scale = %v", a.appliedTextScale())
	}
}
