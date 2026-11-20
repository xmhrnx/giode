# Calls (WebRTC)

The `rtc` package connects two peers over WebRTC, on [pion](https://github.com/pion/webrtc):
voice calls from the microphone to the speaker with the platform's echo cancellation, video
calls from the camera or a screen capture, and data channels. It works with browsers and any
other WebRTC endpoint.

It doesn't do signaling. Your app carries the offer, the answer and the ICE candidates between
the peers, over whatever it already has: a WebSocket, push messages, a chat backend.

```go
import "github.com/xmhrnx/giode/rtc"

c, err := rtc.New(ctx, app, rtc.Config{
	ICEServers: []rtc.ICEServer{
		{URLs: []string{"stun:stun.example.com:3478"}},
		{URLs: []string{"turn:turn.example.com:3478"}, Username: user, Credential: pass},
	},
	Microphone: true, Speaker: true,
	OnCandidate: func(cand string) { signal.Send(peer, "candidate", cand) },
	OnState:     func(s rtc.State) { app.Post(func() { status = s.String() }) },
})

// The caller:
offer, _ := c.Offer(ctx)        // send to the other side
c.Accept(answerFromThem)

// The callee:
answer, _ := c.Answer(ctx, offerFromThem) // send back

// Both, for each candidate the other side sends:
c.AddCandidate(cand)

defer c.Close()
```

Without `OnCandidate`, `Offer` and `Answer` wait until every candidate is found and include
them, so one message each way is enough.

| Config | |
|---|---|
| `Microphone`, `MicrophoneDevice` | send the microphone, as a voice stream (see [Audio](audio.md)) |
| `Speaker`, `SpeakerPhone` | play the other side; the loudspeaker on phones |
| `SendAudio` | send audio you write with `WriteAudio(pcm)` at `AudioSampleRate()` |
| `OnRemoteAudio` | the other side's decoded audio, for recording or transcription |
| `AudioCodecs` | codecs preferred over the built-in G.711 (`rtc.PCMU`, `rtc.PCMA`), such as Opus |
| `VideoCodecs`, `SendVideo`, `VideoFPS` | video: frames go out with `SendVideo(rgba)` and arrive on `RemoteVideo()` |
| `RelayOnly` | send everything through TURN, hiding the peers' addresses |
| `OnDataChannel` | the other side opened a data channel |

`SetMuted` sends silence; `Stats` counts packets and frames and gives the other side's
loudness for a meter; `State` is `StateNew` to `StateConnected`, `StateDisconnected` (may come
back), `StateFailed` or `StateClosed`.

## Data channels

```go
d, _ := c.OpenDataChannel("chat", true) // ordered and reliable
d.OnOpen(func() { d.SendText("hello") })
d.OnMessage(func(data []byte, text bool) { ... })
```

The other side gets it through `OnDataChannel`.

## Codecs

Audio works out of the box with G.711, which every WebRTC endpoint accepts: telephone
quality, 64 kbit/s. For better audio, add an `rtc.AudioCodec` for Opus (wrapping libopus or a
platform codec); it's preferred when the other side has it too.

Video works out of the box with `rtc.VP8`, a VP8 codec in pure Go that runs on every platform.
Its encoder makes key frames only: every frame stands alone, so it needs more bandwidth than a
full encoder (roughly 1 to 2 Mbit/s for 640×480 at 15 frames a second; `rtc.VP8Quality(q)`
trades quality for size, 0 best to 127 smallest) and a lost packet costs one frame. Browsers
decode it like any VP8. Its decoder takes key frames only too: between Giode apps that's every
frame; from a browser, which sends key frames only when asked, the picture updates about once
a second. For full-rate video from browsers, add your own `rtc.VideoCodec` (VP8, VP9, H.264 or
AV1, wrapping libvpx or a platform codec): its encoder turns `*image.RGBA` frames into the
bitstream and its decoder turns it back into images; the call does the RTP packetization, key
frame requests and loss recovery.

```go
c, _ := rtc.New(ctx, app, rtc.Config{
	Microphone: true, Speaker: true,
	VideoCodecs: []rtc.VideoCodec{rtc.VP8}, SendVideo: true, VideoFPS: 15,
	...
})
cam, _ := camera.Open(ctx, app, camera.Options{Facing: camera.Front, Color: true,
	AnalysisSize: 480, AnalysisRate: 15,
	OnFrame: func(f camera.Frame) { c.SendVideo(f.RGBA) }})

c.RemoteVideo().View(video.ViewOptions{Fit: video.Cover}) // the other side
```

Screen sharing sends `screencap` frames the same way (see [Screen capture](screencap.md));
[Video](video.md) has the view's options.

The browser build isn't supported yet: `New` returns `rtc.ErrUnsupported` there.
