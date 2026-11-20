package rtc

import (
	"bytes"
	"errors"
	"image"

	"golang.org/x/image/vp8"

	"github.com/xmhrnx/giode/video"
)

// VP8 is a built-in video codec in pure Go: every frame is a key frame,
// so it needs more bandwidth than a full encoder (about 1 to 2 Mbit/s
// for 640×480 at 15 frames a second) but runs everywhere, and a lost
// packet costs only one frame. Its decoder takes key frames only:
// between Giode apps that's every frame; from a browser, which sends
// key frames only when asked, the picture updates about once a
// second.
var VP8 VideoCodec = vp8Codec{quality: 36}

// VP8Quality is VP8 with a quantizer from 0 (best, biggest) to 127.
func VP8Quality(q int) VideoCodec { return vp8Codec{quality: max(0, min(q, 127))} }

type vp8Codec struct{ quality int }

func (vp8Codec) MimeType() string   { return "video/VP8" }
func (vp8Codec) Fmtp() string       { return "" }
func (vp8Codec) PayloadType() uint8 { return 96 }

func (c vp8Codec) NewEncoder(w, h, fps int) (VideoEncoder, error) {
	if w <= 0 || h <= 0 || w > 16383 || h > 16383 {
		return nil, errors.New("rtc: bad VP8 frame size")
	}
	return &vp8Enc{e: newVP8Encoder(w, h, c.quality)}, nil
}

func (vp8Codec) NewDecoder() (VideoDecoder, error) { return &vp8Dec{d: vp8.NewDecoder()}, nil }

type vp8Enc struct{ e *vp8Encoder }

func (v *vp8Enc) Encode(f *image.RGBA, keyframe bool) ([]byte, error) {
	if f.Rect.Dx() != v.e.w || f.Rect.Dy() != v.e.h {
		return nil, errors.New("rtc: frame size changed")
	}
	return v.e.Encode(f), nil
}

func (v *vp8Enc) Close() {}

type vp8Dec struct{ d *vp8.Decoder }

func (v *vp8Dec) Decode(b []byte) (image.Image, error) {
	v.d.Init(bytes.NewReader(b), len(b))
	fh, err := v.d.DecodeFrameHeader()
	if err != nil {
		return nil, err
	}
	if !fh.KeyFrame {
		return nil, nil
	}
	img, err := v.d.DecodeFrame()
	if err != nil {
		return nil, err
	}
	return video.I420{
		Y: img.Y, U: img.Cb, V: img.Cr,
		StrideY: img.YStride, StrideU: img.CStride, StrideV: img.CStride,
		Width: img.Rect.Dx(), Height: img.Rect.Dy(),
	}.RGBA(), nil
}

func (v *vp8Dec) Close() {}
