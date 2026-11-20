package rtc

// G.711, the codec every WebRTC endpoint speaks: 8 kHz, 8 bits a sample.

type g711 struct {
	mime string
	pt   uint8
	alaw bool
}

// PCMU is G.711 μ-law, built in.
var PCMU AudioCodec = g711{mime: "audio/PCMU", pt: 0}

// PCMA is G.711 A-law, built in.
var PCMA AudioCodec = g711{mime: "audio/PCMA", pt: 8, alaw: true}

func (g g711) MimeType() string                  { return g.mime }
func (g g711) ClockRate() int                    { return 8000 }
func (g g711) Channels() int                     { return 1 }
func (g g711) SampleRate() int                   { return 8000 }
func (g g711) Fmtp() string                      { return "" }
func (g g711) PayloadType() uint8                { return g.pt }
func (g g711) NewEncoder() (AudioEncoder, error) { return g, nil }
func (g g711) NewDecoder() (AudioDecoder, error) { return g, nil }

func (g g711) Encode(pcm []int16) ([]byte, error) {
	out := make([]byte, len(pcm))
	for i, s := range pcm {
		if g.alaw {
			out[i] = linearToALaw(s)
		} else {
			out[i] = linearToULaw(s)
		}
	}
	return out, nil
}

func (g g711) Decode(b []byte) ([]int16, error) {
	out := make([]int16, len(b))
	for i, c := range b {
		if g.alaw {
			out[i] = aLawToLinear(c)
		} else {
			out[i] = uLawToLinear(c)
		}
	}
	return out, nil
}

func (g g711) Close() {}

const (
	uBias = 0x84
	uClip = 32635
)

func linearToULaw(s int16) byte {
	v := int(s)
	sign := 0
	if v < 0 {
		v, sign = -v, 0x80
	}
	v = min(v, uClip) + uBias
	exp := 7
	for mask := 0x4000; v&mask == 0 && exp > 0; mask >>= 1 {
		exp--
	}
	mant := (v >> (exp + 3)) & 0x0f
	return ^byte(sign | exp<<4 | mant)
}

func uLawToLinear(c byte) int16 {
	c = ^c
	sign, exp, mant := int(c&0x80), int(c>>4)&7, int(c&0x0f)
	v := ((mant << 3) + uBias) << exp
	v -= uBias
	if sign != 0 {
		v = -v
	}
	return int16(v)
}

func linearToALaw(s int16) byte {
	v := int(s) >> 3 // 13 bits
	sign := 0x80
	if v < 0 {
		v, sign = -v-1, 0
	}
	var out int
	if v < 32 {
		out = v >> 1
	} else {
		exp := 1
		for t := v >> 5; t > 1 && exp < 7; t >>= 1 {
			exp++
		}
		out = exp<<4 | (v>>exp)&0x0f
	}
	return byte(out|sign) ^ 0x55
}

func aLawToLinear(c byte) int16 {
	c ^= 0x55
	sign, exp, mant := c&0x80, int(c>>4)&7, int(c&0x0f)
	var v int
	if exp == 0 {
		v = mant<<4 + 8
	} else {
		v = (mant<<4 + 0x108) << (exp - 1)
	}
	if sign == 0 {
		v = -v
	}
	return int16(v)
}
