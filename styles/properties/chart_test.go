package properties

import "testing"

func TestChartTokens(t *testing.T) {
	defer SetMode(CurrentMode())
	SetMode(ModeLight)
	c1, c9 := CalcColor("theme.chart1"), CalcColor("theme.chart9")
	if c1 != CalcColor(DefaultLightPalette.Chart[0]) || c9 != c1 {
		t.Errorf("chart1 %v, chart9 %v", c1, c9)
	}
	SetMode(ModeDark)
	if CalcColor("theme.chart2") != CalcColor(DefaultDarkPalette.Chart[1]) {
		t.Error("dark chart2")
	}
	if CalcColor("theme.chartx").A != 0 && CalcColor("theme.chartx") == c1 {
		t.Error("chartx resolved")
	}
}
