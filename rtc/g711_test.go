package rtc

import (
	"math"
	"testing"
)

func TestG711RoundTrip(t *testing.T) {
	for _, c := range []AudioCodec{PCMU, PCMA} {
		enc, _ := c.NewEncoder()
		dec, _ := c.NewDecoder()
		pcm := make([]int16, 0, 70000)
		for v := -32768; v <= 32767; v += 1 {
			pcm = append(pcm, int16(v))
		}
		b, _ := enc.Encode(pcm)
		out, _ := dec.Decode(b)
		for i, s := range pcm {
			// Logarithmic: the error grows with the level, about 1/16.
			tol := math.Abs(float64(s))/14 + 40
			if d := math.Abs(float64(out[i]) - float64(s)); d > tol {
				t.Fatalf("%s: %d came back as %d", c.MimeType(), s, out[i])
			}
		}
		if out, _ := dec.Decode([]byte{mustEnc(enc, 0)}); out[0] > 16 || out[0] < -16 {
			t.Errorf("%s: silence came back as %d", c.MimeType(), out[0])
		}
	}
}

func mustEnc(e AudioEncoder, s int16) byte {
	b, _ := e.Encode([]int16{s})
	return b[0]
}
