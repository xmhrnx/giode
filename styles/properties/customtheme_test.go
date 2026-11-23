package properties

import "testing"

func TestCustomThemeColors(t *testing.T) {
	defer SetPalette(ModeLight, DefaultLightPalette)
	defer SetPalette(ModeDark, DefaultDarkPalette)
	defer SetMode(CurrentMode())
	light := DefaultLightPalette
	light.Custom = map[string]Color{"editorGutter": "#f0f0f0"}
	SetPalette(ModeLight, light)
	SetThemeColor(ModeDark, "editorGutter", "#313335")
	SetThemeColor(ModeLight, "diffAdded", "#c8e6c9")

	SetMode(ModeLight)
	if c := CalcColor("theme.editorGutter"); c != CalcColor("#f0f0f0") {
		t.Errorf("light gutter %v", c)
	}
	if c := CalcColor("theme.EDITORGUTTER"); c != CalcColor("#f0f0f0") {
		t.Errorf("names aren't case-insensitive: %v", c)
	}
	SetMode(ModeDark)
	if c := CalcColor("theme.editorgutter"); c != CalcColor("#313335") {
		t.Errorf("dark gutter %v", c)
	}
	// Set in light only: the same color in dark.
	if c := CalcColor("theme.diffAdded"); c != CalcColor("#c8e6c9") {
		t.Errorf("dark diffAdded %v, want the light one", c)
	}
	// Built-in names win.
	SetThemeColor(ModeDark, "primary", "#000000")
	if c := CalcColor("theme.primary"); c != CalcColor(DefaultDarkPalette.Primary) {
		t.Errorf("a custom primary replaced the built-in one: %v", c)
	}
	if _, ok := themeToken("theme.nothing"); ok {
		t.Error("an unknown name resolved")
	}
}
