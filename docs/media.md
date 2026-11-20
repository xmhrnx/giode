# Media player

The `media` package plays video and audio files and streams on each platform's own engine,
with a ready-made view: the picture, captions over it and a control bar.

```go
import "github.com/xmhrnx/giode/media"

p, err := media.Open(ctx, app, "talk.mp4", media.Options{Autoplay: true})
defer p.Close()

cues, _ := media.ParseCaptions(srtFile) // SRT or WebVTT
p.AddCaptions(media.Track{Label: "English", Language: "en", Cues: cues})

giode.Box(giode.Styles{Width: -1, Height: 360, BorderRadius: 8, Overflow: giode.OverflowHidden},
	p.View(media.ViewOptions{}))
```

`Open` takes a file path or an `http(s)` or `file` URL and returns at once; the player is
`Loading` until the media's ready, then `Paused` (or `Playing` with `Autoplay`).

| Control | |
|---|---|
| `Play`, `Pause`, `Toggle` | `Play` after the end starts over |
| `Seek(d)` | clamped to the media |
| `SetSpeed(r)` | 0.5, 1.5, 2...; speech keeps its pitch where the platform can |
| `SetVolume(v)`, `SetMuted(m)` | volume 0 to 1 |
| `Position`, `Duration`, `State`, `Err`, `VideoSize` | for your own controls |
| `AddCaptions`, `SetCaptionTrack(i)`, `Caption()` | -1 hides captions; `Caption` is the text showing now |

`Options` sets `Autoplay`, `Loop`, `Muted`, `Volume`, and `OnEnded` and `OnError` callbacks.
They and the methods are safe from any goroutine.

`View` fills the space it's given: the picture (`Fit` as in [Video](video.md)), captions in
a box near the bottom (read out by screen readers as they change), and the controls - play,
the seek bar with the time, speed (stepping through `media.Speeds`), captions and volume.
`ViewOptions.NoControls` hides the bar; `Controls()` is the bar alone, for audio. For custom
layouts, `Surface()` is the `video.Surface` the frames go to.

## Platforms

| | Engine | Formats |
|---|---|---|
| iOS, macOS | AVPlayer | MP4, MOV, HLS, MP3, AAC... |
| Android | MediaPlayer | MP4, WebM, MKV, MP3, AAC, HLS... |
| Windows | Media Foundation's media engine | MP4, MOV, MP3, AAC, WMV...; Windows N needs the Media Feature Pack |
| Linux | GStreamer, loaded when a player opens | whatever plugins are installed; `gstreamer1.0-plugins-good` and `-libav` cover most |

Linux builds need nothing extra: without GStreamer, `Open` returns an error naming the
packages. The web isn't supported (`media.ErrUnsupported`).

`examples/player` opens files, URLs and caption files.
