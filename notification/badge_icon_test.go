package notification

import "testing"

func TestBadgeImage(t *testing.T) {
	for _, n := range []int{1, 7, 42, 120} {
		img := badgeImage(n, 32)
		white, red := 0, 0
		for i := 0; i < len(img.Pix); i += 4 {
			switch {
			case img.Pix[i] == 0xff && img.Pix[i+1] == 0xff:
				white++
			case img.Pix[i] == 0xd9 && img.Pix[i+3] == 0xff:
				red++
			}
		}
		if white == 0 || red < 300 {
			t.Errorf("%d: white %d, red %d", n, white, red)
		}
		if img.Pix[3] != 0 {
			t.Errorf("%d: corner isn't transparent", n)
		}
	}
}
