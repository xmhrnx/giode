package properties

import (
	"strings"
	"sync/atomic"

	"gioui.org/font"
)

// High contrast and bold text: system accessibility settings the app
// follows (see the root package's Config.Contrast and Config.BoldText).

var (
	highContrast atomic.Bool
	boldText     atomic.Bool
	// hcLight and hcDark are the palettes high contrast uses, nil for
	// the defaults; guarded by themeMu.
	hcLight, hcDark *Palette
)

// DefaultHighContrastLightPalette and DefaultHighContrastDarkPalette are
// the palettes theme tokens resolve against while high contrast is on:
// black on white and white on black, no muted greys, and status colors
// that keep at least 7:1 against the background.
var (
	DefaultHighContrastLightPalette = Palette{
		Primary:    "#0000c8",
		Secondary:  "#000000",
		Background: "#ffffff",
		Surface:    "#ffffff",
		Text:       "#000000",
		Border:     "#000000",
		OnPrimary:  "#ffffff",
		Muted:      "#000000",
		Accent:     "#5b00b3",
		Success:    "#005a00",
		Warning:    "#7a3d00",
		Danger:     "#a00000",
		Info:       "#004a6e",
		Chart:      []Color{"#0033cc", "#a14a00", "#006b3c", "#b0002a", "#5b21b6", "#005f73", "#9d174d", "#3f6212"},
	}
	DefaultHighContrastDarkPalette = Palette{
		Primary:    "#ffd700",
		Secondary:  "#ffffff",
		Background: "#000000",
		Surface:    "#000000",
		Text:       "#ffffff",
		Border:     "#ffffff",
		OnPrimary:  "#000000",
		Muted:      "#ffffff",
		Accent:     "#e0b0ff",
		Success:    "#5cff5c",
		Warning:    "#ffb000",
		Danger:     "#ff8080",
		Info:       "#7fdcff",
		Chart:      []Color{"#8ab4ff", "#ffc966", "#5ef2b0", "#ff8fa3", "#c8b6ff", "#5ce1ff", "#ff9ed2", "#c6f57a"},
	}
)

// SetHighContrast turns high contrast on or off: theme tokens resolve
// against the high-contrast palette of the active mode while it's on.
// Safe to call from any goroutine.
func SetHighContrast(on bool) {
	if highContrast.Swap(on) != on {
		notifyTheme()
	}
}

// HighContrast reports whether high contrast is on.
func HighContrast() bool { return highContrast.Load() }

// SetHighContrastPalette sets the palette high contrast uses for mode m
// (the system's own colors, where it has them, or the app's); nil goes
// back to the default.
func SetHighContrastPalette(m Mode, p *Palette) {
	themeMu.Lock()
	if m == ModeDark {
		hcDark = p
	} else {
		hcLight = p
	}
	themeMu.Unlock()
	if HighContrast() {
		notifyTheme()
	}
}

// highContrastPaletteLocked returns the high-contrast palette for the
// active mode, and the defaults its empty fields fall back to.
func highContrastPaletteLocked() (Palette, Palette) {
	if currentMode == ModeDark {
		if hcDark != nil {
			return *hcDark, DefaultHighContrastDarkPalette
		}
		return DefaultHighContrastDarkPalette, DefaultHighContrastDarkPalette
	}
	if hcLight != nil {
		return *hcLight, DefaultHighContrastLightPalette
	}
	return DefaultHighContrastLightPalette, DefaultHighContrastLightPalette
}

// highContrastToken resolves a token while high contrast is on.
func highContrastToken(c string) (Color, bool) {
	themeMu.RLock()
	p, def := highContrastPaletteLocked()
	themeMu.RUnlock()
	field := strings.ToLower(strings.TrimPrefix(c, "theme."))
	v, ok := paletteField(p, field)
	if !ok {
		return "", false
	}
	if v == "" {
		v, _ = paletteField(def, field)
	}
	return v, true
}

// SetBoldText turns bold text on or off: every font weight is two steps
// heavier while it's on (normal text semibold, semibold text extrabold).
// Safe to call from any goroutine.
func SetBoldText(on bool) {
	if boldText.Swap(on) != on {
		notifyTheme()
	}
}

// BoldText reports whether bold text is on.
func BoldText() bool { return boldText.Load() }

// emboldened is w made heavier for bold text.
func emboldened(w font.Weight) font.Weight {
	if !boldText.Load() {
		return w
	}
	return min(w+200, font.Black)
}
