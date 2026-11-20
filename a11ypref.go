package giode

import (
	"math"
	"time"

	"github.com/xmhrnx/giode/styles/properties"
)

// System accessibility settings the app follows: high contrast, bold
// text and text size (Config.Contrast, Config.BoldText,
// Config.TextScale). Reduced motion is in motion.go.

// Preference is whether a setting follows the system or is forced.
type Preference int

const (
	// PreferenceSystem follows the system's setting, checked as the app
	// runs.
	PreferenceSystem Preference = iota
	// PreferenceOff turns it off whatever the system says.
	PreferenceOff
	// PreferenceOn turns it on whatever the system says.
	PreferenceOn
)

// HighContrast reports whether "theme.*" colors resolve against the
// high-contrast palette: the user asked the system for more contrast,
// or Config.Contrast or SetContrast said so.
func HighContrast() bool { return properties.HighContrast() }

// BoldText reports whether text is drawn two weights heavier.
func BoldText() bool { return properties.BoldText() }

// SystemHighContrast reports the system's high contrast setting, or
// ErrNotSupported where it can't be read.
func SystemHighContrast() (bool, error) { return readHighContrast() }

// SystemBoldText reports the system's bold text setting, or
// ErrNotSupported where it can't be read.
func SystemBoldText() (bool, error) { return readBoldText() }

// SystemTextScale reports the system's text size as a factor (1 is the
// default size), or ErrNotSupported where it can't be read.
func SystemTextScale() (float32, error) { return readTextScale() }

// The readers, replaceable in tests.
var (
	readHighContrast = systemHighContrast
	readBoldText     = systemBoldText
	readTextScale    = systemTextScale
)

// SetContrast overrides Config.Contrast while the app runs.
func (a *App) SetContrast(p Preference) {
	a.contrast.Store(int32(p))
	a.applyContrast()
}

// SetBoldText overrides Config.BoldText while the app runs.
func (a *App) SetBoldText(p Preference) {
	a.boldText.Store(int32(p))
	a.applyBoldText()
}

// SetTextScale overrides Config.TextScale while the app runs: 0 follows
// the system.
func (a *App) SetTextScale(scale float32) {
	a.textScaleSet.Store(math.Float32bits(scale))
	a.applyTextScale()
}

// textScaleWanted is the scale asked for: Config.TextScale or
// SetTextScale's, 0 for the system's.
func (a *App) textScaleWanted() float32 {
	return math.Float32frombits(a.textScaleSet.Load())
}

// TextScale returns the factor text is scaled by.
func (a *App) TextScale() float32 {
	return math.Float32frombits(a.textScale.Load())
}

func (a *App) applyContrast() {
	on := false
	switch Preference(a.contrast.Load()) {
	case PreferenceOn:
		on = true
	case PreferenceSystem:
		on, _ = readHighContrast()
		if on {
			// The system's own high-contrast colors, where it has them.
			if light, dark, ok := systemContrastPalettes(); ok {
				properties.SetHighContrastPalette(properties.ModeLight, light)
				properties.SetHighContrastPalette(properties.ModeDark, dark)
			}
		}
	}
	if on != properties.HighContrast() {
		properties.SetHighContrast(on)
		a.invalidate()
	}
}

func (a *App) applyBoldText() {
	on := false
	switch Preference(a.boldText.Load()) {
	case PreferenceOn:
		on = true
	case PreferenceSystem:
		on, _ = readBoldText()
	}
	if on != properties.BoldText() {
		properties.SetBoldText(on)
		a.invalidate()
	}
}

func (a *App) applyTextScale() {
	s := a.textScaleWanted()
	if s <= 0 {
		s = 1
		if v, err := readTextScale(); err == nil && v > 0 {
			s = v
		}
	}
	s = max(0.5, min(s, 4))
	if old := a.textScale.Swap(math.Float32bits(s)); old != math.Float32bits(s) {
		a.invalidate()
	}
}

// appliedTextScale is the factor the frame multiplies sp by: the text
// scale where Gio doesn't apply the system's itself (Android and Wayland
// do; a set Config.TextScale applies everywhere).
func (a *App) appliedTextScale() float32 {
	s := a.TextScale()
	if s == 0 {
		return 1
	}
	if a.textScaleWanted() <= 0 && gioScalesText(a) {
		return 1
	}
	return s
}

// watchA11yPrefs applies the settings at startup and when they change:
// checked every few seconds and when the window regains focus.
func (a *App) watchA11yPrefs(done <-chan struct{}) {
	a.contrast.Store(int32(a.cfg.Contrast))
	a.boldText.Store(int32(a.cfg.BoldText))
	a.textScaleSet.Store(math.Float32bits(a.cfg.TextScale))
	a.applyContrast()
	a.applyBoldText()
	a.applyTextScale()
	go func() {
		t := time.NewTicker(themePoll)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
			case <-a.prefCheck:
			}
			a.applyContrast()
			a.applyBoldText()
			if a.textScaleWanted() <= 0 {
				a.applyTextScale()
			}
		}
	}()
}
