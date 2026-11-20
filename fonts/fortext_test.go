package fonts

import (
	"testing"

	"gioui.org/font"
)

func TestForText(t *testing.T) {
	UseGofont()
	SetDefaultFont("")
	if f := ForText(font.Font{Weight: font.SemiBold}, "آبان"); f.Weight != font.Bold {
		t.Errorf("semibold Persian = %v", f.Weight)
	}
	if f := ForText(font.Font{Weight: font.Medium}, "日本語"); f.Weight != font.Normal {
		t.Errorf("medium Japanese = %v", f.Weight)
	}
	if f := ForText(font.Font{Weight: font.SemiBold}, "November"); f.Weight != font.SemiBold {
		t.Errorf("semibold Latin = %v", f.Weight)
	}
}
