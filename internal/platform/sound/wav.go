// Package sound plays short WAV sounds on each platform, for
// giode.App.PlaySound.
package sound

import (
	"encoding/binary"
	"errors"
	"math"
)

// Play plays a WAV's samples on the platform, without waiting.
func Play(wav []byte) error {
	w, err := parseWAV(wav)
	if err != nil {
		return err
	}
	return playSound(wav, w)
}

// wavInfo is what PlaySound reads from a WAV.
type wavInfo struct {
	float          bool
	channels, rate int
	bits           int
	data           []byte
}

var errNotWAV = errors.New("giode: not a WAV of PCM or float samples")

// parseWAV reads a WAV's format and samples.
func parseWAV(b []byte) (wavInfo, error) {
	var w wavInfo
	if len(b) < 12 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return w, errNotWAV
	}
	haveFmt := false
	for p := 12; p+8 <= len(b); {
		id, n := string(b[p:p+4]), int(binary.LittleEndian.Uint32(b[p+4:p+8]))
		body := b[p+8:]
		if n > len(body) {
			if id != "data" {
				return w, errNotWAV
			}
			n = len(body) // a stream's data chunk may claim more
		}
		body = body[:n]
		switch id {
		case "fmt ":
			if n < 16 {
				return w, errNotWAV
			}
			format := binary.LittleEndian.Uint16(body[0:2])
			if format == 0xFFFE && n >= 26 { // WAVE_FORMAT_EXTENSIBLE
				format = binary.LittleEndian.Uint16(body[24:26])
			}
			w.channels = int(binary.LittleEndian.Uint16(body[2:4]))
			w.rate = int(binary.LittleEndian.Uint32(body[4:8]))
			w.bits = int(binary.LittleEndian.Uint16(body[14:16]))
			switch {
			case format == 1 && (w.bits == 8 || w.bits == 16 || w.bits == 24 || w.bits == 32):
			case format == 3 && w.bits == 32:
				w.float = true
			default:
				return w, errNotWAV
			}
			if w.channels < 1 || w.channels > 2 || w.rate < 4000 || w.rate > 192000 {
				return w, errNotWAV
			}
			haveFmt = true
		case "data":
			if !haveFmt {
				return w, errNotWAV
			}
			frame := w.channels * w.bits / 8
			w.data = body[:len(body)/frame*frame]
			if len(w.data) == 0 {
				return w, errNotWAV
			}
			return w, nil
		}
		p += 8 + n + n&1
	}
	return w, errNotWAV
}

// pcm16 returns the samples as interleaved little-endian 16-bit PCM.
func (w wavInfo) pcm16() []byte {
	size := w.bits / 8
	n := len(w.data) / size
	out := make([]byte, n*2)
	for i := 0; i < n; i++ {
		s := w.data[i*size : (i+1)*size]
		var v int16
		switch {
		case w.float:
			f := math.Float32frombits(binary.LittleEndian.Uint32(s))
			v = int16(max(-1, min(1, f)) * math.MaxInt16)
		case size == 1:
			v = int16(int(s[0])-128) << 8
		case size == 2:
			v = int16(binary.LittleEndian.Uint16(s))
		default: // 24 or 32 bits: the top 16
			v = int16(binary.LittleEndian.Uint16(s[size-2:]))
		}
		binary.LittleEndian.PutUint16(out[i*2:], uint16(v))
	}
	return out
}

// Encode returns a mono 16-bit WAV of samples, each from -1 to 1.
func Encode(samples []float32, sampleRate int) []byte {
	data := len(samples) * 2
	b := make([]byte, 44+data)
	copy(b[0:], "RIFF")
	binary.LittleEndian.PutUint32(b[4:], uint32(36+data))
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], 1) // PCM
	binary.LittleEndian.PutUint16(b[22:], 1) // mono
	binary.LittleEndian.PutUint32(b[24:], uint32(sampleRate))
	binary.LittleEndian.PutUint32(b[28:], uint32(sampleRate*2))
	binary.LittleEndian.PutUint16(b[32:], 2)
	binary.LittleEndian.PutUint16(b[34:], 16)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], uint32(data))
	for i, s := range samples {
		v := int16(max(-1, min(1, s)) * math.MaxInt16)
		binary.LittleEndian.PutUint16(b[44+i*2:], uint16(v))
	}
	return b
}
