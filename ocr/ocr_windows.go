package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"strings"
	"time"
	"unsafe"

	xdraw "golang.org/x/image/draw"

	"github.com/xmhrnx/giode/internal/winrt"
)

// Windows 10 and later read text with Windows.Media.Ocr, offline, in the
// languages whose OCR packs are installed (Settings, Time & language,
// Language: each language's "Optical character recognition").

const supported = true

var (
	iidOcrEngineStatics    = winrt.ParseGUID("5bffa85a-3384-3540-9940-699120d428a8")
	iidOcrResult           = winrt.ParseGUID("9bd235b2-175b-3d6a-92e2-388c206e2f63")
	iidOcrLine             = winrt.ParseGUID("0043a16f-e31f-3a24-899c-d444bd088124")
	iidOcrWord             = winrt.ParseGUID("3c2a477a-5cd9-3525-ba2a-23d1e0a68a1d")
	iidSoftwareBitmapStats = winrt.ParseGUID("df0385db-672f-4a9d-806e-c2442f343e86")
	iidCryptoBufferStats   = winrt.ParseGUID("320b7e22-3cb0-4cdf-8663-1d28910065eb")
	iidLanguageFactory     = winrt.ParseGUID("9b0252ac-0c27-44f8-b792-9793fb66c63e")
)

const bitmapBgra8 = 87

type rectF struct{ X, Y, W, H float32 }

func recognize(ctx context.Context, data []byte, opts Options) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var out []byte
	derr := winrt.Do(func() { out, err = recognizeImage(ctx, img, opts) })
	if derr != nil {
		return nil, derr
	}
	return out, err
}

// engine makes an OCR engine for the first language it has a pack for,
// else for the user's languages.
func engine(stats *winrt.Obj, langs []string) (*winrt.Obj, error) {
	if len(langs) > 0 {
		lf, err := winrt.Factory("Windows.Globalization.Language", iidLanguageFactory)
		if err == nil {
			defer lf.Release()
			for _, tag := range langs {
				h := winrt.NewHString(tag)
				lang, err := lf.Out(6, uintptr(h))
				h.Delete()
				if err != nil {
					continue
				}
				var e *winrt.Obj
				stats.Call(9, uintptr(unsafe.Pointer(lang)), uintptr(unsafe.Pointer(&e)))
				lang.Release()
				if e != nil {
					return e, nil
				}
			}
		}
	}
	var e *winrt.Obj
	stats.Call(10, uintptr(unsafe.Pointer(&e)))
	if e == nil {
		return nil, errors.New("ocr: no OCR language installed (Settings, Time & language, Language)")
	}
	return e, nil
}

func recognizeImage(ctx context.Context, img image.Image, opts Options) ([]byte, error) {
	stats, err := winrt.Factory("Windows.Media.Ocr.OcrEngine", iidOcrEngineStatics)
	if err != nil {
		return nil, err
	}
	defer stats.Release()
	var maxDim uint32
	stats.Call(6, uintptr(unsafe.Pointer(&maxDim)))
	eng, err := engine(stats, opts.Languages)
	if err != nil {
		return nil, err
	}
	defer eng.Release()

	// BGRA pixels, shrunk to the engine's largest dimension.
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	scale := 1.0
	if maxDim > 0 && (w > int(maxDim) || h > int(maxDim)) {
		scale = float64(maxDim) / float64(max(w, h))
		w, h = int(float64(w)*scale), int(float64(h)*scale)
	}
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	if scale == 1 {
		draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
	} else {
		xdraw.CatmullRom.Scale(rgba, rgba.Bounds(), img, b, draw.Src, nil)
	}
	px := rgba.Pix
	for i := 0; i < len(px); i += 4 {
		px[i], px[i+2] = px[i+2], px[i]
	}
	cb, err := winrt.Factory("Windows.Security.Cryptography.CryptographicBuffer", iidCryptoBufferStats)
	if err != nil {
		return nil, err
	}
	defer cb.Release()
	buf, err := cb.Out(9, uintptr(len(px)), uintptr(unsafe.Pointer(&px[0])))
	if err != nil {
		return nil, err
	}
	defer buf.Release()
	sb, err := winrt.Factory("Windows.Graphics.Imaging.SoftwareBitmap", iidSoftwareBitmapStats)
	if err != nil {
		return nil, err
	}
	defer sb.Release()
	bitmap, err := sb.Out(9, uintptr(unsafe.Pointer(buf)), bitmapBgra8, uintptr(w), uintptr(h))
	if err != nil {
		return nil, err
	}
	defer bitmap.Release()

	op, err := eng.Out(6, uintptr(unsafe.Pointer(bitmap))) // RecognizeAsync
	if err != nil {
		return nil, err
	}
	timeout := time.Minute
	if d, ok := ctx.Deadline(); ok {
		timeout = time.Until(d)
	}
	r, err := winrt.Await(op, timeout)
	if err != nil {
		return nil, err
	}
	defer r.Release()
	result, err := r.Query(iidOcrResult)
	if err != nil {
		return nil, err
	}
	defer result.Release()

	// Boxes back in the image's pixels.
	unscale := func(rc rectF) [4]float64 {
		s := 1 / scale
		return [4]float64{float64(rc.X) * s, float64(rc.Y) * s, float64(rc.X+rc.W) * s, float64(rc.Y+rc.H) * s}
	}
	hstr := func(o *winrt.Obj, slot int) string {
		var hs winrt.HString
		o.Call(slot, uintptr(unsafe.Pointer(&hs)))
		defer hs.Delete()
		return hs.String()
	}
	var res wireResult
	lines, err := result.Out(6)
	if err != nil {
		return json.Marshal(res)
	}
	defer lines.Release()
	var n uint32
	lines.Call(7, uintptr(unsafe.Pointer(&n)))
	for i := uint32(0); i < n; i++ {
		lo, err := lines.Out(6, uintptr(i))
		if err != nil {
			continue
		}
		line, err := lo.Query(iidOcrLine)
		lo.Release()
		if err != nil {
			continue
		}
		wl := wireLine{Text: hstr(line, 7)}
		if words, err := line.Out(6); err == nil {
			var m uint32
			words.Call(7, uintptr(unsafe.Pointer(&m)))
			for j := uint32(0); j < m; j++ {
				wo, err := words.Out(6, uintptr(j))
				if err != nil {
					continue
				}
				word, err := wo.Query(iidOcrWord)
				wo.Release()
				if err != nil {
					continue
				}
				var rc rectF
				word.Call(6, uintptr(unsafe.Pointer(&rc)))
				ww := wireWord{Text: hstr(word, 7), Box: unscale(rc)}
				word.Release()
				if len(wl.Words) == 0 {
					wl.Box = ww.Box
				} else {
					wl.Box[0], wl.Box[1] = min(wl.Box[0], ww.Box[0]), min(wl.Box[1], ww.Box[1])
					wl.Box[2], wl.Box[3] = max(wl.Box[2], ww.Box[2]), max(wl.Box[3], ww.Box[3])
				}
				wl.Words = append(wl.Words, ww)
			}
			words.Release()
		}
		line.Release()
		res.Lines = append(res.Lines, wl)
	}
	texts := make([]string, len(res.Lines))
	for i, l := range res.Lines {
		texts[i] = l.Text
	}
	res.Text = strings.Join(texts, "\n")
	return json.Marshal(res)
}
