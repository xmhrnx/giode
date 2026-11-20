# Video

The `video` package shows moving pictures. A `Surface` takes frames from any goroutine (a
decoder, a call's remote track, a screen capture) and its `View` element draws the latest
one.

```go
import "github.com/xmhrnx/giode/video"

s := video.NewSurface(app)

go func() {
	for f := range decoded {
		s.PushI420(video.I420{Y: f.Y, U: f.U, V: f.V,
			StrideY: f.StrideY, StrideU: f.StrideU, StrideV: f.StrideV,
			Width: f.W, Height: f.H})
	}
}()

giode.Box(giode.Styles{Width: 320, Height: 240, BorderRadius: 12, Overflow: giode.OverflowHidden},
	s.View(video.ViewOptions{Fit: video.Cover, Mirror: true, Label: "Your camera"}))
```

`Push(img)` takes any `image.Image` (an `*image.RGBA` is copied fastest); `PushI420` takes
the planar YUV that video decoders and WebRTC produce, limited-range BT.601 by default
(`FullRange`, `BT709` for others). Frames are copied into the surface's own buffers, so the
caller may reuse theirs, and pushing never waits for drawing. A redraw without a new frame
doesn't upload the texture again.

| ViewOptions | |
|---|---|
| `Fit` | `Contain` (the whole frame, letterboxed; default), `Cover` (fill, cropping) or `Stretch` |
| `Mirror` | flip left to right, as a self view is |
| `Rotation` | 0, 90, 180 or 270 degrees clockwise |
| `Background` | around and under the frame; black by default |
| `Placeholder` | an element shown before the first frame and after `Clear` |
| `Label` | read by screen readers |
| `Overlay` | draws over the frame, with a function placing frame points (0–1) in the view |

`Size`, `Frames` and `FPS` report the latest frame's size, how many arrived and the rate
over the last second. `Clear` removes the frame.

`examples/screenshare` shows a screen capture through a surface.
