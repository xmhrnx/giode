package notification

import (
	"image"
	"image/color"
)

// badgeDigits is a 3×5 pixel font for the badge's digits and "+".
var badgeDigits = map[rune][5]string{
	'0': {"###", "#.#", "#.#", "#.#", "###"},
	'1': {".#.", "##.", ".#.", ".#.", "###"},
	'2': {"###", "..#", "###", "#..", "###"},
	'3': {"###", "..#", ".##", "..#", "###"},
	'4': {"#.#", "#.#", "###", "..#", "..#"},
	'5': {"###", "#..", "###", "..#", "###"},
	'6': {"###", "#..", "###", "#.#", "###"},
	'7': {"###", "..#", ".#.", ".#.", ".#."},
	'8': {"###", "#.#", "###", "#.#", "###"},
	'9': {"###", "#.#", "###", "..#", "###"},
	'+': {"...", ".#.", "###", ".#.", "..."},
}

// badgeImage draws n as white digits on a red disc, size×size pixels:
// the taskbar's overlay icon.
func badgeImage(n, size int) *image.NRGBA {
	text := itoa(n)
	if n > 99 {
		text = "99+"
	}
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	c := float64(size) / 2
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			// Coverage of the disc, from 4×4 samples.
			cov := 0
			for sy := 0; sy < 4; sy++ {
				for sx := 0; sx < 4; sx++ {
					dx := float64(x) + (float64(sx)+0.5)/4 - c
					dy := float64(y) + (float64(sy)+0.5)/4 - c
					if dx*dx+dy*dy <= c*c {
						cov++
					}
				}
			}
			if cov > 0 {
				img.SetNRGBA(x, y, nrgba(0xd9, 0x2d, 0x20, uint8(cov*255/16)))
			}
		}
	}
	scale := size / 8
	if len(text) > 2 || scale < 1 {
		scale = max(size/16, 1)
	}
	w := len(text)*3*scale + (len(text)-1)*scale
	h := 5 * scale
	x0, y0 := (size-w)/2, (size-h)/2
	for i, r := range text {
		g := badgeDigits[r]
		for gy := 0; gy < 5; gy++ {
			for gx := 0; gx < 3; gx++ {
				if g[gy][gx] != '#' {
					continue
				}
				for py := 0; py < scale; py++ {
					for px := 0; px < scale; px++ {
						x := x0 + i*4*scale + gx*scale + px
						y := y0 + gy*scale + py
						img.SetNRGBA(x, y, nrgba(0xff, 0xff, 0xff, 0xff))
					}
				}
			}
		}
	}
	return img
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}
func nrgba(r, g, b, a uint8) color.NRGBA { return color.NRGBA{r, g, b, a} }
