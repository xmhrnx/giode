package properties

import "gioui.org/font"

// FontWeight is the weight of text (CSS font-weight).
type FontWeight string

const (
	FontWeightThin       FontWeight = "thin"
	FontWeightExtraLight FontWeight = "extralight"
	FontWeightLight      FontWeight = "light"
	FontWeightNormal     FontWeight = "normal"
	FontWeightMedium     FontWeight = "medium"
	FontWeightSemiBold   FontWeight = "semibold"
	FontWeightBold       FontWeight = "bold"
	FontWeightExtraBold  FontWeight = "extrabold"
	FontWeightBlack      FontWeight = "black"
)

// Weight maps the value to a font weight, defaulting to normal. The
// CSS numbers "100" to "900" work too.
func (w FontWeight) Weight() font.Weight {
	return emboldened(w.weight())
}

func (w FontWeight) weight() font.Weight {
	switch w {
	case FontWeightThin, "100":
		return font.Thin
	case FontWeightExtraLight, "200":
		return font.ExtraLight
	case FontWeightLight, "300":
		return font.Light
	case FontWeightMedium, "500":
		return font.Medium
	case FontWeightSemiBold, "600":
		return font.SemiBold
	case FontWeightBold, "700":
		return font.Bold
	case FontWeightExtraBold, "800":
		return font.ExtraBold
	case FontWeightBlack, "900":
		return font.Black
	default:
		return font.Normal
	}
}
