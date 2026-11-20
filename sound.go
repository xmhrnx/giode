package giode

import "github.com/xmhrnx/giode/internal/platform/sound"

// PlaySound plays a short sound - the contents of a WAV file - and
// returns without waiting for it to finish: a chime when a task is
// done, a click, an alert. The WAV holds PCM samples of 8, 16, 24 or 32
// bits, or 32-bit floats, in one channel or two, at any common rate.
// Scale the samples for a quieter sound; EncodeWAV makes a WAV from
// samples generated in code.
//
// Sounds overlap, except on Windows, where a sound stops the one still
// playing. It returns an error for data that isn't such a WAV, and
// ErrNotSupported where nothing can play it (on Linux it needs
// pw-play, paplay or aplay). Call it from any goroutine.
//
// Where it plays: Windows' PlaySound; NSSound on macOS; AVAudioPlayer on
// iOS, mixed with other audio and silent in silent mode; AudioTrack on
// Android, as a notification sound; Web Audio in the browser (which
// plays sounds only after the user has interacted with the page);
// PipeWire, PulseAudio or ALSA on Linux.
func (a *App) PlaySound(wav []byte) error {
	return sound.Play(wav)
}

// EncodeWAV returns a mono 16-bit WAV of samples, each from -1 to 1,
// at sampleRate per second - for PlaySound, with tones generated in
// code:
//
//	const rate = 44100
//	samples := make([]float32, rate/5) // 0.2 s
//	for i := range samples {
//		t := float64(i) / rate
//		fade := 1 - float64(i)/float64(len(samples))
//		samples[i] = float32(0.2 * fade * math.Sin(2*math.Pi*440*t))
//	}
//	app.PlaySound(giode.EncodeWAV(samples, rate))
func EncodeWAV(samples []float32, sampleRate int) []byte {
	return sound.Encode(samples, sampleRate)
}
