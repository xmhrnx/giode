# Audio

The `audio` package streams the microphone and the speaker: recording memos, dictation
engines, calls and generated sound. Voice streams turn on the platform's echo cancellation,
noise suppression and gain control.

```go
import "github.com/xmhrnx/giode/audio"

in, err := audio.OpenInput(ctx, app, audio.Options{Voice: true}, func(samples []int16) {
	encoder.Write(samples) // 20 ms of 48 kHz mono, on the stream's goroutine
})
defer in.Close()

out, err := audio.OpenOutput(ctx, app, audio.Options{Voice: true})
defer out.Close()
out.Write(ctx, decoded) // queues; blocks while 300 ms are already queued
```

Samples are 16-bit integers, interleaved by channel.

| Option | |
|---|---|
| `Device` | a device's `ID` from `audio.Inputs(ctx)` or `audio.Outputs(ctx)`; empty for the default |
| `SampleRate` | 48000 by default; 16000 and 8000 suit speech engines |
| `Channels` | 1 (default) or 2 |
| `Frame` | how much `OnData` gets at a time: 20 ms by default |
| `Voice` | processing for calls: echo cancellation, noise suppression and gain control, and the call audio route on phones |
| `SpeakerPhone` | plays a voice stream on the loudspeaker instead of the earpiece |

`Output.Buffered` reports what's queued and `Flush` drops it; `audio.Level(samples)` gives a
0–1 loudness for meters. `audio.RequestPermission(ctx, app)` asks for the microphone up
front; `OpenInput` asks too, and returns `audio.ErrPermissionDenied` without it.

## Platforms

| | Streams | Voice processing |
|---|---|---|
| Android | AudioRecord, AudioTrack | communication mode, the platform's echo canceler, noise suppressor and gain control |
| iOS, macOS | audio units | the voice-processing unit; input and output share it, so its echo cancellation hears what's played |
| Windows | WASAPI | communications streams, which Windows 11's Voice Clarity or the audio driver's processing applies to |
| Linux | PulseAudio or PipeWire | the sound server's echo-cancel filter, where it has one loaded |

The web isn't supported. On Android the package adds `RECORD_AUDIO` to the manifest; on
iOS and macOS set `ios.usage.microphone` in `giode.toml` (`NSMicrophoneUsageDescription`).

`examples/voicememo` records, plays back and runs an echo test through voice streams.
