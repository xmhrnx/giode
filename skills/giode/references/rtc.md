# Calls (WebRTC)

The `rtc` package connects two peers over WebRTC, on [pion](https://github.com/pion/webrtc):
voice calls from the microphone to the speaker with the platform's echo cancellation, data
channels, and video through a codec you provide. It works with browsers and any other WebRTC
endpoint.

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

Video needs an `rtc.VideoCodec`: VP8, VP9, H.264 or AV1, whose encoder turns `*image.RGBA`
frames into the codec's bitstream and whose decoder turns it back into images. The call does
the RTP packetization, key frame requests and loss recovery. Show the other side with
`c.RemoteVideo().View(video.ViewOptions{Fit: video.Cover})` (see [Video](video.md)); send the
camera's frames or a [screen capture](screencap.md).

The browser build isn't supported yet: `New` returns `rtc.ErrUnsupported` there.
