package rtc

import (
	"encoding/binary"
	"image"
)

// A VP8 encoder of key frames only, in pure Go: every frame stands
// alone, so a lost packet costs one frame and any frame can be
// decoded. It predicts each macroblock from its decoded neighbors
// (DC prediction), transforms the residual (DCT, with a WHT of the
// luma DCs), quantizes it and codes the tokens with the spec's default
// probabilities (RFC 6386). It reconstructs each macroblock exactly as
// a decoder does, so predictions match. There's no loop filter.

// boolEncoder is VP8's boolean arithmetic coder (RFC 6386, 7.3).
type boolEncoder struct {
	out    []byte
	rng    uint32
	bottom uint32
	count  int
}

func newBoolEncoder() *boolEncoder { return &boolEncoder{rng: 255, count: 24} }

func (e *boolEncoder) carry() {
	for i := len(e.out) - 1; i >= 0; i-- {
		if e.out[i] != 255 {
			e.out[i]++
			return
		}
		e.out[i] = 0
	}
}

func (e *boolEncoder) put(prob uint8, bit bool) {
	split := 1 + ((e.rng-1)*uint32(prob))>>8
	if bit {
		e.bottom += split
		e.rng -= split
	} else {
		e.rng = split
	}
	for e.rng < 128 {
		e.rng <<= 1
		if e.bottom&(1<<31) != 0 {
			e.carry()
		}
		e.bottom <<= 1
		e.count--
		if e.count == 0 {
			e.out = append(e.out, byte(e.bottom>>24))
			e.bottom &= 1<<24 - 1
			e.count = 8
		}
	}
}

func (e *boolEncoder) putBit(prob uint8, b int) { e.put(prob, b != 0) }

// putUint writes n bits, most significant first, at even odds.
func (e *boolEncoder) putUint(v uint32, n int) {
	for i := n - 1; i >= 0; i-- {
		e.put(uniformProb, v>>uint(i)&1 != 0)
	}
}

func (e *boolEncoder) flush() []byte {
	c, v := e.count, e.bottom
	if v&(1<<uint(32-c)) != 0 {
		e.carry()
	}
	v <<= uint(c & 7)
	for c >>= 3; c > 0; c-- {
		v <<= 8
	}
	for i := 0; i < 4; i++ {
		e.out = append(e.out, byte(v>>24))
		v <<= 8
	}
	return e.out
}

const uniformProb = 128

var (
	bands   = [17]uint8{0, 1, 2, 3, 6, 4, 5, 6, 6, 6, 6, 6, 6, 6, 6, 7, 0}
	zigzag  = [16]uint8{0, 1, 4, 8, 5, 2, 3, 6, 9, 12, 13, 10, 7, 11, 14, 15}
	cat3456 = [4][12]uint8{
		{173, 148, 140, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{176, 155, 140, 135, 0, 0, 0, 0, 0, 0, 0, 0},
		{180, 157, 141, 134, 130, 0, 0, 0, 0, 0, 0, 0},
		{254, 254, 243, 230, 196, 177, 153, 140, 133, 130, 129, 0},
	}
)

// vp8Encoder encodes frames of one size.
type vp8Encoder struct {
	w, h, mbw, mbh int
	qi             int
	y1, y2, uv     [2]int32 // DC and AC quantizers
	// Source and reconstructed planes, padded to whole macroblocks.
	sy, su, sv []byte
	ry, ru, rv []byte
	ys, cs     int
}

func newVP8Encoder(w, h, qi int) *vp8Encoder {
	e := &vp8Encoder{w: w, h: h, mbw: (w + 15) / 16, mbh: (h + 15) / 16, qi: qi}
	e.ys, e.cs = e.mbw*16, e.mbw*8
	n, cn := e.ys*e.mbh*16, e.cs*e.mbh*8
	e.sy, e.su, e.sv = make([]byte, n), make([]byte, cn), make([]byte, cn)
	e.ry, e.ru, e.rv = make([]byte, n), make([]byte, cn), make([]byte, cn)
	dc, ac := int32(dequantTableDC[qi]), int32(dequantTableAC[qi])
	e.y1 = [2]int32{dc, ac}
	e.y2 = [2]int32{dc * 2, max(ac*155/100, 8)}
	e.uv = [2]int32{int32(dequantTableDC[min(qi, 117)]), ac}
	return e
}

// load converts RGBA to BT.601 video-range YCbCr 4:2:0, repeating the
// edge pixels into the padding.
func (e *vp8Encoder) load(img *image.RGBA) {
	b := img.Rect
	for y := 0; y < e.mbh*16; y++ {
		sy := b.Min.Y + min(y, e.h-1)
		row := img.Pix[(sy-b.Min.Y)*img.Stride:]
		for x := 0; x < e.mbw*16; x++ {
			o := min(x, e.w-1) * 4
			r, g, bl := int(row[o]), int(row[o+1]), int(row[o+2])
			e.sy[y*e.ys+x] = byte((66*r+129*g+25*bl+128)>>8 + 16)
		}
	}
	for y := 0; y < e.mbh*8; y++ {
		for x := 0; x < e.mbw*8; x++ {
			var su, sv int
			for dy := 0; dy < 2; dy++ {
				sy := min(2*y+dy, e.h-1)
				row := img.Pix[sy*img.Stride:]
				for dx := 0; dx < 2; dx++ {
					o := min(2*x+dx, e.w-1) * 4
					r, g, bl := int(row[o]), int(row[o+1]), int(row[o+2])
					su += (-38*r - 74*g + 112*bl + 128) >> 8
					sv += (112*r - 94*g - 18*bl + 128) >> 8
				}
			}
			e.su[y*e.cs+x] = byte(clampByte((su+2)>>2 + 128))
			e.sv[y*e.cs+x] = byte(clampByte((sv+2)>>2 + 128))
		}
	}
}

func clampByte(v int) int { return max(0, min(v, 255)) }

// Encode encodes img, which must be w×h, as a key frame.
func (e *vp8Encoder) Encode(img *image.RGBA) []byte {
	e.load(img)
	hdr, tok := newBoolEncoder(), newBoolEncoder()

	// The frame header (RFC 6386, 9.2 to 9.11).
	hdr.putBit(uniformProb, 0) // color space
	hdr.putBit(uniformProb, 0) // clamping required
	hdr.putBit(uniformProb, 0) // no segmentation
	hdr.putBit(uniformProb, 0) // normal filter
	hdr.putUint(0, 6)          // filter level 0: no loop filter
	hdr.putUint(0, 3)          // sharpness
	hdr.putBit(uniformProb, 0) // no filter deltas
	hdr.putUint(0, 2)          // one token partition
	hdr.putUint(uint32(e.qi), 7)
	for i := 0; i < 5; i++ {
		hdr.putBit(uniformProb, 0) // no quantizer deltas
	}
	hdr.putBit(uniformProb, 1) // refresh entropy probabilities
	for i := range tokenProbUpdateProb {
		for j := range tokenProbUpdateProb[i] {
			for k := range tokenProbUpdateProb[i][j] {
				for l := range tokenProbUpdateProb[i][j][k] {
					hdr.putBit(tokenProbUpdateProb[i][j][k][l], 0) // the defaults
				}
			}
		}
	}
	hdr.putBit(uniformProb, 0) // no macroblock skipping

	// Non-zero contexts: per macroblock column above, and to the left.
	upY := make([][4]uint8, e.mbw)
	upC := make([][4]uint8, e.mbw)
	upY2 := make([]uint8, e.mbw)
	for mby := 0; mby < e.mbh; mby++ {
		var leftY, leftC [4]uint8
		var leftY2 uint8
		for mbx := 0; mbx < e.mbw; mbx++ {
			// Modes: 16×16 DC luma, DC chroma.
			hdr.putBit(145, 1)
			hdr.putBit(156, 0)
			hdr.putBit(163, 0)
			hdr.putBit(142, 0)
			e.macroblock(tok, mbx, mby, &upY[mbx], &leftY, &upC[mbx], &leftC, &upY2[mbx], &leftY2)
		}
	}

	first := hdr.flush()
	second := tok.flush()
	out := make([]byte, 10, 10+len(first)+len(second))
	tag := uint32(1<<4) | uint32(len(first))<<5 // key frame, version 0, shown
	out[0], out[1], out[2] = byte(tag), byte(tag>>8), byte(tag>>16)
	out[3], out[4], out[5] = 0x9d, 0x01, 0x2a
	binary.LittleEndian.PutUint16(out[6:], uint16(e.w))
	binary.LittleEndian.PutUint16(out[8:], uint16(e.h))
	out = append(out, first...)
	return append(out, second...)
}

// dcPred is the DC prediction of an n×n block at (x, y) in plane p,
// from the reconstructed row above and column left, as the decoder
// makes it.
func dcPred(p []byte, stride, x, y, n int, top, left bool) byte {
	shift := 3
	if n == 16 {
		shift = 4
	}
	switch {
	case !top && !left:
		return 128
	case !top:
		s := 1 << (shift - 1)
		for j := 0; j < n; j++ {
			s += int(p[(y+j)*stride+x-1])
		}
		return byte(s >> shift)
	case !left:
		s := 1 << (shift - 1)
		for i := 0; i < n; i++ {
			s += int(p[(y-1)*stride+x+i])
		}
		return byte(s >> shift)
	}
	s := n
	for i := 0; i < n; i++ {
		s += int(p[(y-1)*stride+x+i]) + int(p[(y+i)*stride+x-1])
	}
	return byte(s >> (shift + 1))
}

func (e *vp8Encoder) macroblock(tok *boolEncoder, mbx, mby int, upY, leftY, upC, leftC *[4]uint8, upY2, leftY2 *uint8) {
	top, left := mby > 0, mbx > 0
	x0, y0 := mbx*16, mby*16
	predY := dcPred(e.ry, e.ys, x0, y0, 16, top, left)

	// Luma: DCT each 4×4 block of the residual; the DCs go to the WHT.
	var coef [16][16]int32
	var dcs [16]int32
	for b := 0; b < 16; b++ {
		bx, by := x0+b%4*4, y0+b/4*4
		var res [16]int32
		for j := 0; j < 4; j++ {
			for i := 0; i < 4; i++ {
				res[j*4+i] = int32(e.sy[(by+j)*e.ys+bx+i]) - int32(predY)
			}
		}
		coef[b] = fdct(res)
		dcs[b] = coef[b][0]
	}
	y2 := fwht(dcs)
	var qy2 [16]int32
	for i := range y2 {
		qy2[i] = quantize(y2[i], e.y2[btoi(i > 0)], i == 0)
	}
	var qy [16][16]int32
	for b := range coef {
		for i := 1; i < 16; i++ {
			qy[b][i] = quantize(coef[b][i], e.y1[1], false)
		}
	}

	// Tokens.
	nz := writeBlock(tok, planeY2, *leftY2+*upY2, &qy2, 0)
	*leftY2, *upY2 = nz, nz
	for by := 0; by < 4; by++ {
		for bx := 0; bx < 4; bx++ {
			nz := writeBlock(tok, planeY1WithY2, leftY[by]+upY[bx], &qy[by*4+bx], 1)
			leftY[by], upY[bx] = nz, nz
		}
	}

	// Reconstruct the luma as the decoder does.
	var deq [16]int32
	for i := range qy2 {
		deq[i] = qy2[i] * e.y2[btoi(i > 0)]
	}
	dcOut := iwht(deq)
	for b := 0; b < 16; b++ {
		bx, by := x0+b%4*4, y0+b/4*4
		var c [16]int32
		c[0] = dcOut[b]
		hasAC := false
		for i := 1; i < 16; i++ {
			c[i] = qy[b][i] * e.y1[1]
			hasAC = hasAC || qy[b][i] != 0
		}
		var blk [16]byte
		for i := range blk {
			blk[i] = predY
		}
		if hasAC {
			idct(&blk, c)
		} else if c[0] != 0 {
			idctDC(&blk, c[0])
		}
		for j := 0; j < 4; j++ {
			copy(e.ry[(by+j)*e.ys+bx:], blk[j*4:j*4+4])
		}
	}

	// Chroma: U then V, each 8×8 as four 4×4 blocks.
	cx, cy := mbx*8, mby*8
	for ch, planes := range [2][2][]byte{{e.su, e.ru}, {e.sv, e.rv}} {
		src, rec := planes[0], planes[1]
		pred := dcPred(rec, e.cs, cx, cy, 8, top, left)
		var qc [4][16]int32
		any := false
		for b := 0; b < 4; b++ {
			bx, by := cx+b%2*4, cy+b/2*4
			var res [16]int32
			for j := 0; j < 4; j++ {
				for i := 0; i < 4; i++ {
					res[j*4+i] = int32(src[(by+j)*e.cs+bx+i]) - int32(pred)
				}
			}
			c := fdct(res)
			for i := range c {
				qc[b][i] = quantize(c[i], e.uv[btoi(i > 0)], i == 0)
				any = any || qc[b][i] != 0
			}
		}
		for by := 0; by < 2; by++ {
			for bx := 0; bx < 2; bx++ {
				nz := writeBlock(tok, planeUV, leftC[ch*2+by]+upC[ch*2+bx], &qc[by*2+bx], 0)
				leftC[ch*2+by], upC[ch*2+bx] = nz, nz
			}
		}
		for b := 0; b < 4; b++ {
			bx, by := cx+b%2*4, cy+b/2*4
			var blk [16]byte
			for i := range blk {
				blk[i] = pred
			}
			if any {
				var c [16]int32
				for i := range c {
					c[i] = qc[b][i] * e.uv[btoi(i > 0)]
				}
				idct(&blk, c)
			}
			for j := 0; j < 4; j++ {
				copy(rec[(by+j)*e.cs+bx:], blk[j*4:j*4+4])
			}
		}
	}
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// quantize divides, rounding the DC to nearest and the AC a little
// toward zero, which saves bits for little loss.
func quantize(c, q int32, dc bool) int32 {
	bias := q / 3
	if dc {
		bias = q / 2
	}
	neg := c < 0
	if neg {
		c = -c
	}
	v := (c + bias) / q
	// Beyond the largest token, or overflowing the decoder's int16.
	v = min(v, 2048+66, 32767/q)
	if neg {
		return -v
	}
	return v
}

// writeBlock codes a block's quantized coefficients (raster order)
// from position start, and reports whether any was non-zero.
func writeBlock(e *boolEncoder, plane int, ctx uint8, q *[16]int32, start int) uint8 {
	probs := &defaultTokenProb[plane]
	last := -1
	for n := start; n < 16; n++ {
		if q[zigzag[n]] != 0 {
			last = n
		}
	}
	n := start
	p := &probs[bands[n]][ctx]
	if last < 0 {
		e.put(p[0], false) // end of block
		return 0
	}
	e.put(p[0], true)
	for n < 16 {
		c := q[zigzag[n]]
		v := c
		if v < 0 {
			v = -v
		}
		if v == 0 {
			e.put(p[1], false)
			n++
			p = &probs[bands[n]][0]
			continue
		}
		e.put(p[1], true)
		next := 2
		switch {
		case v == 1:
			e.put(p[2], false)
			next = 1
		case v <= 4:
			e.put(p[2], true)
			e.put(p[3], false)
			if v == 2 {
				e.put(p[4], false)
			} else {
				e.put(p[4], true)
				e.put(p[5], v == 4)
			}
		case v <= 10:
			e.put(p[2], true)
			e.put(p[3], true)
			e.put(p[6], false)
			if v <= 6 {
				e.put(p[7], false)
				e.put(159, v == 6)
			} else {
				e.put(p[7], true)
				e.put(165, (v-7)>>1 != 0)
				e.put(145, (v-7)&1 != 0)
			}
		default:
			e.put(p[2], true)
			e.put(p[3], true)
			e.put(p[6], true)
			cat := 3
			switch {
			case v <= 18:
				cat = 0
			case v <= 34:
				cat = 1
			case v <= 66:
				cat = 2
			}
			e.put(p[8], cat>>1 != 0)
			e.put(p[9+cat>>1], cat&1 != 0)
			extra := v - (3 + 8<<cat)
			tab := &cat3456[cat]
			nb := 0
			for tab[nb] != 0 {
				nb++
			}
			for i := 0; i < nb; i++ {
				e.put(tab[i], extra>>(nb-1-i)&1 != 0)
			}
		}
		e.put(uniformProb, c < 0)
		n++
		p = &probs[bands[n]][next]
		if n == 16 {
			break
		}
		if n > last {
			e.put(p[0], false)
			break
		}
		e.put(p[0], true)
	}
	return 1
}

// fdct is libvpx's forward 4×4 DCT; any close transform would do, as
// reconstruction uses the decoder's inverse.
func fdct(in [16]int32) [16]int32 {
	var t, out [16]int32
	for i := 0; i < 4; i++ {
		ip := in[i*4:]
		a1 := (ip[0] + ip[3]) * 8
		b1 := (ip[1] + ip[2]) * 8
		c1 := (ip[1] - ip[2]) * 8
		d1 := (ip[0] - ip[3]) * 8
		t[i*4+0] = a1 + b1
		t[i*4+2] = a1 - b1
		t[i*4+1] = (c1*2217 + d1*5352 + 14500) >> 12
		t[i*4+3] = (d1*2217 - c1*5352 + 7500) >> 12
	}
	for i := 0; i < 4; i++ {
		a1 := t[i] + t[12+i]
		b1 := t[4+i] + t[8+i]
		c1 := t[4+i] - t[8+i]
		d1 := t[i] - t[12+i]
		out[i] = (a1 + b1 + 7) >> 4
		out[8+i] = (a1 - b1 + 7) >> 4
		out[4+i] = (c1*2217+d1*5352+12000)>>16 + int32(btoi(d1 != 0))
		out[12+i] = (d1*2217 - c1*5352 + 51000) >> 16
	}
	return out
}

// fwht is libvpx's forward Walsh-Hadamard transform of the 16 luma
// DCs.
func fwht(in [16]int32) [16]int32 {
	var t, out [16]int32
	for i := 0; i < 4; i++ {
		ip := in[i*4:]
		a1 := (ip[0] + ip[2]) * 4
		d1 := (ip[1] + ip[3]) * 4
		c1 := (ip[1] - ip[3]) * 4
		b1 := (ip[0] - ip[2]) * 4
		t[i*4+0] = a1 + d1 + int32(btoi(a1 != 0))
		t[i*4+1] = b1 + c1
		t[i*4+2] = b1 - c1
		t[i*4+3] = a1 - d1
	}
	for i := 0; i < 4; i++ {
		a1 := t[i] + t[8+i]
		d1 := t[4+i] + t[12+i]
		c1 := t[4+i] - t[12+i]
		b1 := t[i] - t[8+i]
		a2, b2, c2, d2 := a1+d1, b1+c1, b1-c1, a1-d1
		a2 += int32(btoi(a2 < 0))
		b2 += int32(btoi(b2 < 0))
		c2 += int32(btoi(c2 < 0))
		d2 += int32(btoi(d2 < 0))
		out[i] = (a2 + 3) >> 3
		out[4+i] = (b2 + 3) >> 3
		out[8+i] = (c2 + 3) >> 3
		out[12+i] = (d2 + 3) >> 3
	}
	return out
}

// iwht is the decoder's inverse WHT: the DC of each luma block.
func iwht(c [16]int32) [16]int32 {
	var m, out [16]int32
	for i := 0; i < 4; i++ {
		a0 := c[i] + c[12+i]
		a1 := c[4+i] + c[8+i]
		a2 := c[4+i] - c[8+i]
		a3 := c[i] - c[12+i]
		m[i] = a0 + a1
		m[8+i] = a0 - a1
		m[4+i] = a3 + a2
		m[12+i] = a3 - a2
	}
	for i := 0; i < 4; i++ {
		dc := m[i*4] + 3
		a0 := dc + m[3+i*4]
		a1 := m[1+i*4] + m[2+i*4]
		a2 := m[1+i*4] - m[2+i*4]
		a3 := dc - m[3+i*4]
		out[i*4+0] = int32(int16((a0 + a1) >> 3))
		out[i*4+1] = int32(int16((a3 + a2) >> 3))
		out[i*4+2] = int32(int16((a0 - a1) >> 3))
		out[i*4+3] = int32(int16((a3 - a2) >> 3))
	}
	return out
}

func clip8(v int32) byte {
	return byte(max(0, min(v, 255)))
}

// idct is the decoder's inverse DCT, added to the prediction in blk.
func idct(blk *[16]byte, c [16]int32) {
	const c1, c2 = 85627, 35468
	for i := range c {
		c[i] = int32(int16(c[i])) // as the decoder stores them
	}
	var m [4][4]int32
	for i := 0; i < 4; i++ {
		a := c[i] + c[8+i]
		b := c[i] - c[8+i]
		cc := (c[4+i]*c2)>>16 - (c[12+i]*c1)>>16
		d := (c[4+i]*c1)>>16 + (c[12+i]*c2)>>16
		m[i][0] = a + d
		m[i][1] = b + cc
		m[i][2] = b - cc
		m[i][3] = a - d
	}
	for j := 0; j < 4; j++ {
		dc := m[0][j] + 4
		a := dc + m[2][j]
		b := dc - m[2][j]
		cc := (m[1][j]*c2)>>16 - (m[3][j]*c1)>>16
		d := (m[1][j]*c1)>>16 + (m[3][j]*c2)>>16
		r := blk[j*4:]
		r[0] = clip8(int32(r[0]) + (a+d)>>3)
		r[1] = clip8(int32(r[1]) + (b+cc)>>3)
		r[2] = clip8(int32(r[2]) + (b-cc)>>3)
		r[3] = clip8(int32(r[3]) + (a-d)>>3)
	}
}

func idctDC(blk *[16]byte, dc int32) {
	v := (int32(int16(dc)) + 4) >> 3
	for i := range blk {
		blk[i] = clip8(int32(blk[i]) + v)
	}
}
