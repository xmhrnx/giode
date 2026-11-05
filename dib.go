package giode

import (
	"encoding/binary"
	"errors"
	"image"
	"image/color"
)

// parseDIB decodes a device-independent bitmap as Windows' clipboard
// holds one (CF_DIB, CF_DIBV5): a BITMAPINFOHEADER or BITMAPV5HEADER
// followed by the pixels, no file header. It takes the uncompressed
// 24- and 32-bit bitmaps screenshots and browsers put there.
func parseDIB(b []byte) (image.Image, error) {
	le := binary.LittleEndian
	if len(b) < 40 {
		return nil, errors.New("giode: clipboard bitmap too short")
	}
	hsize := int(le.Uint32(b[0:]))
	w := int(int32(le.Uint32(b[4:])))
	h := int(int32(le.Uint32(b[8:])))
	bpp := int(le.Uint16(b[14:]))
	compression := le.Uint32(b[16:])
	clrUsed := int(le.Uint32(b[32:]))
	if hsize < 40 || hsize > len(b) || w <= 0 || h == 0 || w > 1<<15 {
		return nil, errors.New("giode: clipboard bitmap has a bad header")
	}
	const biRGB, biBitfields = 0, 3
	topDown := h < 0
	if topDown {
		h = -h
	}
	if h > 1<<15 {
		return nil, errors.New("giode: clipboard bitmap too large")
	}
	// Channel masks: after a plain header for BI_BITFIELDS, inside a
	// V4/V5 header, or the BGR(A) defaults.
	rMask, gMask, bMask, aMask := uint32(0xff0000), uint32(0xff00), uint32(0xff), uint32(0)
	off := hsize
	switch compression {
	case biRGB:
	case biBitfields:
		if bpp != 32 {
			return nil, errors.New("giode: unsupported clipboard bitmap format")
		}
		if hsize == 40 {
			if len(b) < 52 {
				return nil, errors.New("giode: clipboard bitmap too short")
			}
			rMask, gMask, bMask = le.Uint32(b[40:]), le.Uint32(b[44:]), le.Uint32(b[48:])
			off += 12
		} else if hsize >= 56 {
			rMask, gMask, bMask, aMask = le.Uint32(b[40:]), le.Uint32(b[44:]), le.Uint32(b[48:]), le.Uint32(b[52:])
		}
	default:
		return nil, errors.New("giode: unsupported clipboard bitmap compression")
	}
	if bpp != 24 && bpp != 32 {
		return nil, errors.New("giode: unsupported clipboard bitmap depth")
	}
	off += clrUsed * 4
	if compression == biRGB && bpp == 32 && hsize >= 56 {
		// A V4/V5 header may still give an alpha mask for BI_RGB.
		aMask = le.Uint32(b[52:])
	}
	stride := (w*bpp + 31) / 32 * 4
	if off > len(b) || len(b)-off < stride*h {
		return nil, errors.New("giode: clipboard bitmap too short")
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	channel := func(v, mask uint32) uint8 {
		if mask == 0 {
			return 0
		}
		shift := 0
		for mask&1 == 0 {
			mask >>= 1
			shift++
		}
		v = (v >> shift) & mask
		if mask != 0xff {
			v = v * 255 / mask
		}
		return uint8(v)
	}
	anyAlpha := false
	for y := 0; y < h; y++ {
		src := y
		if !topDown {
			src = h - 1 - y
		}
		row := b[off+src*stride:]
		for x := 0; x < w; x++ {
			var c color.NRGBA
			if bpp == 24 {
				p := row[x*3:]
				c = color.NRGBA{R: p[2], G: p[1], B: p[0], A: 255}
			} else {
				v := le.Uint32(row[x*4:])
				c = color.NRGBA{R: channel(v, rMask), G: channel(v, gMask), B: channel(v, bMask), A: 255}
				if aMask != 0 {
					c.A = channel(v, aMask)
				} else if compression == biRGB {
					// The fourth byte of BI_RGB is reserved, but some
					// apps put alpha there.
					c.A = uint8(v >> 24)
				}
				if c.A != 0 {
					anyAlpha = true
				}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	if bpp == 32 && !anyAlpha {
		// All-zero alpha means the alpha byte is unused: opaque.
		for i := 3; i < len(img.Pix); i += 4 {
			img.Pix[i] = 255
		}
	}
	return img, nil
}
