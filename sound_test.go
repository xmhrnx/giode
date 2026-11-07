package giode

import (
	"encoding/binary"
	"math"
	"testing"
)

// EncodeWAV writes a WAV that PlaySound reads back sample for sample.
func TestEncodeWAVRoundTrip(t *testing.T) {
	samples := []float32{0, 0.5, -0.5, 1, -1, 2}
	w, err := parseWAV(EncodeWAV(samples, 44100))
	if err != nil {
		t.Fatal(err)
	}
	if w.channels != 1 || w.rate != 44100 || w.bits != 16 || w.float || len(w.data) != 2*len(samples) {
		t.Fatalf("read back %+v", w)
	}
	want := []int16{0, 16383, -16383, 32767, -32767, 32767}
	for i, v := range want {
		if got := int16(binary.LittleEndian.Uint16(w.data[i*2:])); got != v {
			t.Errorf("sample %d = %d, want %d", i, got, v)
		}
	}
}

// wavOf makes a WAV of one sample in the given format.
func wavOf(format, channels, bits int, sample []byte) []byte {
	b := []byte("RIFF\x00\x00\x00\x00WAVEfmt \x10\x00\x00\x00")
	b = binary.LittleEndian.AppendUint16(b, uint16(format))
	b = binary.LittleEndian.AppendUint16(b, uint16(channels))
	b = binary.LittleEndian.AppendUint32(b, 8000)
	b = binary.LittleEndian.AppendUint32(b, uint32(8000*channels*bits/8))
	b = binary.LittleEndian.AppendUint16(b, uint16(channels*bits/8))
	b = binary.LittleEndian.AppendUint16(b, uint16(bits))
	// An odd-sized chunk PlaySound skips, padded to an even size.
	b = append(b, "LIST\x03\x00\x00\x00abc\x00"...)
	b = append(b, "data"...)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(sample)))
	return append(b, sample...)
}

// Every sample format becomes 16-bit PCM for platforms that take only
// that; anything else is refused.
func TestParseWAVFormats(t *testing.T) {
	f := make([]byte, 4)
	binary.LittleEndian.PutUint32(f, math.Float32bits(-0.5))
	cases := []struct {
		name         string
		format, bits int
		sample       []byte
		want         int16
	}{
		{"8 bit", 1, 8, []byte{192}, 64 << 8},
		{"16 bit", 1, 16, []byte{0x34, 0x12}, 0x1234},
		{"24 bit", 1, 24, []byte{0xff, 0x34, 0x12}, 0x1234},
		{"32 bit", 1, 32, []byte{0, 0, 0x34, 0x12}, 0x1234},
		{"float", 3, 32, f, -16383},
	}
	for _, c := range cases {
		w, err := parseWAV(wavOf(c.format, 1, c.bits, c.sample))
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got := int16(binary.LittleEndian.Uint16(w.pcm16())); got != c.want {
			t.Errorf("%s: 16-bit sample %d, want %d", c.name, got, c.want)
		}
	}
	for name, b := range map[string][]byte{
		"empty":        nil,
		"not RIFF":     []byte("ID3 an mp3 file............................"),
		"a-law":        wavOf(6, 1, 8, []byte{1}),
		"12 bit":       wavOf(1, 1, 12, []byte{1, 2}),
		"six channels": wavOf(1, 6, 16, make([]byte, 12)),
		"no samples":   wavOf(1, 1, 16, nil),
	} {
		if _, err := parseWAV(b); err == nil {
			t.Errorf("%s: read as a WAV", name)
		}
	}
	if err := (&App{}).PlaySound([]byte("not a sound")); err == nil {
		t.Error("PlaySound took data that isn't a WAV")
	}
}
