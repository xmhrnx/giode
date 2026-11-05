package box

import "image"

// mirror places flow children from the right in a box width wide, for
// right-to-left locales (CSS direction: rtl): a row runs from the right
// and a column aligns its start to the right. Positioned children keep
// their physical Left and Right.
func mirror(items []zPaint, width int) {
	for i := range items {
		it := &items[i]
		if it.positioned {
			continue
		}
		x := width - it.off.X - it.size.X
		it.track.Place(image.Pt(x-it.off.X, 0))
		it.off.X = x
	}
}
