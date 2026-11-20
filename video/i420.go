package video

import "image"

// convertI420 writes f into dst as RGBA.
func convertI420(dst *image.RGBA, f I420) {
	// Fixed-point coefficients, ×256.
	var cr, cgu, cgv, cb int
	if f.BT709 {
		cr, cgu, cgv, cb = 403, 48, 120, 475
	} else {
		cr, cgu, cgv, cb = 359, 88, 183, 454
	}
	for y := 0; y < f.Height; y++ {
		yr := f.Y[y*f.StrideY:]
		ur := f.U[(y/2)*f.StrideU:]
		vr := f.V[(y/2)*f.StrideV:]
		out := dst.Pix[y*dst.Stride:]
		for x := 0; x < f.Width; x++ {
			yy := int(yr[x])
			u, v := int(ur[x/2])-128, int(vr[x/2])-128
			if !f.FullRange {
				yy = (yy - 16) * 298 >> 8
				u, v = u*292>>8, v*292>>8
			}
			o := x * 4
			out[o] = clamp8(yy + (cr*v)>>8)
			out[o+1] = clamp8(yy - (cgu*u+cgv*v)>>8)
			out[o+2] = clamp8(yy + (cb*u)>>8)
			out[o+3] = 255
		}
	}
}

func clamp8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
