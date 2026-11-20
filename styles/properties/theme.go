package properties

import (
	"strings"
	"sync"
	"sync/atomic"
)

// Mode is a global color mode: which Palette "theme.*" color tokens
// (see CalcColor) resolve against.
type Mode string

const (
	// ModeLight is the default mode.
	ModeLight Mode = "light"
	ModeDark  Mode = "dark"
)

// Palette is a named set of semantic colors that "theme.*" color
// tokens (e.g. "theme.primary", "theme.background") resolve against,
// so a component styled once with a token automatically follows
// SetMode/SetPalette instead of a fixed color. Each field takes a
// plain color - hex, functional or a named CSS color, like any other
// Color - not another token. A field left empty resolves to the
// matching field of the built-in default palette for the mode.
type Palette struct {
	Primary    Color
	Secondary  Color
	Background Color
	Surface    Color
	Text       Color
	Border     Color

	// OnPrimary is the text/icon color drawn on top of Primary.
	OnPrimary Color
	// Muted is a de-emphasized text color (captions, placeholders).
	Muted Color
	// Accent is a highlight color distinct from Primary.
	Accent Color
	// Success, Warning, Danger and Info are status colors.
	Success Color
	Warning Color
	Danger  Color
	Info    Color

	// Chart is the series colors of charts, "theme.chart1" and on,
	// repeating past the last; empty takes the default's. They're
	// chosen to tell apart side by side, and the dark ones to stand
	// out on a dark background.
	Chart []Color
}

// DefaultLightPalette and DefaultDarkPalette are the built-in
// palettes for each Mode, in effect until SetPalette overrides one.
var (
	DefaultLightPalette = Palette{
		Primary:    "#3b82f6",
		Secondary:  "#64748b",
		Background: "#ffffff",
		Surface:    "#f1f5f9",
		Text:       "#0f172a",
		Border:     "#cbd5e1",
		OnPrimary:  "#ffffff",
		Muted:      "#64748b",
		Accent:     "#8b5cf6",
		Success:    "#16a34a",
		Warning:    "#d97706",
		Danger:     "#dc2626",
		Info:       "#0284c7",
		Chart:      []Color{"#2563eb", "#f59e0b", "#10b981", "#e11d48", "#8b5cf6", "#0891b2", "#db2777", "#65a30d"},
	}
	DefaultDarkPalette = Palette{
		Primary:    "#3b82f6",
		Secondary:  "#94a3b8",
		Background: "#0f172a",
		Surface:    "#1e293b",
		Text:       "#f8fafc",
		Border:     "#334155",
		OnPrimary:  "#ffffff",
		Muted:      "#94a3b8",
		Accent:     "#a78bfa",
		Success:    "#22c55e",
		Warning:    "#f59e0b",
		Danger:     "#ef4444",
		Info:       "#38bdf8",
		Chart:      []Color{"#60a5fa", "#fbbf24", "#34d399", "#fb7185", "#a78bfa", "#22d3ee", "#f472b6", "#a3e635"},
	}
)

var (
	themeMu      sync.RWMutex
	currentMode  = ModeLight
	lightPalette = DefaultLightPalette
	darkPalette  = DefaultDarkPalette

	// themeVersion increments on every mode or palette change, so
	// caches of painted output (see elements.Memo) know to repaint.
	themeVersion atomic.Uint64
	// themeChanged is called after a change; the app sets it to
	// request a redraw.
	themeChanged atomic.Pointer[func()]
)

// SetMode switches the active color Mode, changing what every
// "theme.*" color token in the tree resolves to, and requests a
// redraw. It is safe to call from any goroutine.
func SetMode(m Mode) {
	themeMu.Lock()
	changed := currentMode != m
	currentMode = m
	themeMu.Unlock()
	if changed {
		notifyTheme()
	}
}

// CurrentMode returns the active color mode (ModeLight by default).
func CurrentMode() Mode {
	themeMu.RLock()
	defer themeMu.RUnlock()
	return currentMode
}

// ToggleMode switches between ModeLight and ModeDark.
func ToggleMode() {
	if CurrentMode() == ModeDark {
		SetMode(ModeLight)
	} else {
		SetMode(ModeDark)
	}
}

// SetPalette replaces the palette used for mode m, changing what
// every "theme.*" color token resolves to while that mode is active.
// It is safe to call from any goroutine.
func SetPalette(m Mode, p Palette) {
	themeMu.Lock()
	if m == ModeDark {
		darkPalette = p
	} else {
		lightPalette = p
	}
	themeMu.Unlock()
	notifyTheme()
}

// CurrentPalette returns the palette of the active mode.
func CurrentPalette() Palette {
	themeMu.RLock()
	defer themeMu.RUnlock()
	return currentPaletteLocked()
}

// ThemeVersion returns a counter that increments on every SetMode or
// SetPalette change. Caches of painted output compare it to know when
// theme tokens may resolve differently.
func ThemeVersion() uint64 {
	return themeVersion.Load()
}

// SetThemeChangeHook registers fn to run after every theme change.
// The app uses it to request a redraw; application code doesn't need
// to call it.
func SetThemeChangeHook(fn func()) {
	if fn == nil {
		themeChanged.Store(nil)
		return
	}
	themeChanged.Store(&fn)
}

func notifyTheme() {
	themeVersion.Add(1)
	if fn := themeChanged.Load(); fn != nil {
		(*fn)()
	}
}

func currentPaletteLocked() Palette {
	if currentMode == ModeDark {
		return darkPalette
	}
	return lightPalette
}

// themeToken resolves a "theme.<field>" color token against the
// active palette's matching field, case-insensitively, falling back to
// the built-in default palette when the field is empty. ok is false
// for anything that isn't a recognized token.
func themeToken(c string) (Color, bool) {
	if highContrast.Load() {
		return highContrastToken(c)
	}
	themeMu.RLock()
	p := currentPaletteLocked()
	mode := currentMode
	themeMu.RUnlock()
	field := strings.ToLower(strings.TrimPrefix(c, "theme."))
	v, ok := paletteField(p, field)
	if !ok {
		return "", false
	}
	if v == "" {
		def := DefaultLightPalette
		if mode == ModeDark {
			def = DefaultDarkPalette
		}
		v, _ = paletteField(def, field)
	}
	return v, true
}

func paletteField(p Palette, field string) (Color, bool) {
	switch field {
	case "primary":
		return p.Primary, true
	case "secondary":
		return p.Secondary, true
	case "background":
		return p.Background, true
	case "surface":
		return p.Surface, true
	case "text":
		return p.Text, true
	case "border":
		return p.Border, true
	case "onprimary", "on-primary":
		return p.OnPrimary, true
	case "muted":
		return p.Muted, true
	case "accent":
		return p.Accent, true
	case "success":
		return p.Success, true
	case "warning":
		return p.Warning, true
	case "danger", "error":
		return p.Danger, true
	case "info":
		return p.Info, true
	default:
		if n, ok := strings.CutPrefix(field, "chart"); ok && n != "" {
			i := 0
			for _, r := range n {
				if r < '0' || r > '9' {
					return "", false
				}
				i = i*10 + int(r-'0')
			}
			if len(p.Chart) == 0 || i < 1 {
				return "", true
			}
			return p.Chart[(i-1)%len(p.Chart)], true
		}
		return "", false
	}
}
