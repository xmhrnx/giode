package markdown

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"  // GIF images
	_ "image/jpeg" // JPEG images
	_ "image/png"  // PNG images
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"gioui.org/widget"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // WebP images

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// FetchImage loads the image src names, for Options.LoadImage: an
// http or https URL, a data: URI (base64), or a local file (a path or
// a file: URL). It reads PNG, JPEG, GIF (the first frame) and WebP.
// Downloads time out after 20 seconds and stop at 20 MB.
//
// Loading the images of Markdown the app doesn't control - a model's
// reply, a message from someone else - tells their servers that the
// document was opened, and from where: wrap FetchImage to allow only
// trusted hosts in such a case.
func FetchImage(src string) (image.Image, error) {
	r, err := openImage(src)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	img, _, err := image.Decode(io.LimitReader(r, maxImageBytes))
	return img, err
}

const maxImageBytes = 20 << 20

var imageClient = &http.Client{Timeout: 20 * time.Second}

func openImage(src string) (io.ReadCloser, error) {
	src = strings.TrimSpace(src)
	switch {
	case strings.HasPrefix(src, "data:"):
		meta, data, ok := strings.Cut(src[len("data:"):], ",")
		if !ok || !strings.HasSuffix(meta, ";base64") {
			return nil, errors.New("markdown: only base64 data: URIs are read")
		}
		b, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return nil, fmt.Errorf("markdown: data: URI: %w", err)
		}
		return io.NopCloser(bytes.NewReader(b)), nil
	case strings.HasPrefix(src, "http://"), strings.HasPrefix(src, "https://"):
		resp, err := imageClient.Get(src)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("markdown: %s: %s", src, resp.Status)
		}
		return resp.Body, nil
	case strings.HasPrefix(src, "file:"):
		u, err := url.Parse(src)
		if err != nil {
			return nil, err
		}
		return os.Open(u.Path)
	}
	return os.Open(src)
}

// maxImageSide bounds a loaded image's longer side in pixels: larger
// ones are scaled down once when they load, so a document of photos
// doesn't hold their full resolution.
const maxImageSide = 2048

// maxCachedImages bounds the loaded images kept across documents.
const maxCachedImages = 48

type imageEntry struct {
	done bool
	img  image.Image
	err  error
}

var (
	imageMu    sync.Mutex
	imageCache = map[string]*imageEntry{}
	imageOrder []string
)

// loadImage returns src's entry, starting load in the background the
// first time src is asked for.
func loadImage(src string, load func(string) (image.Image, error)) *imageEntry {
	imageMu.Lock()
	defer imageMu.Unlock()
	if e, ok := imageCache[src]; ok {
		return e
	}
	e := &imageEntry{}
	imageCache[src] = e
	imageOrder = append(imageOrder, src)
	if len(imageOrder) > maxCachedImages {
		delete(imageCache, imageOrder[0])
		imageOrder = imageOrder[1:]
	}
	go func() {
		img, err := safeLoad(load, src)
		if err == nil && img == nil {
			err = errors.New("markdown: no image")
		}
		if err == nil {
			img = boundImage(img)
		}
		imageMu.Lock()
		e.img, e.err, e.done = img, err, true
		imageMu.Unlock()
	}()
	return e
}

func safeLoad(load func(string) (image.Image, error), src string) (img image.Image, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("markdown: loading %s: %v", src, r)
		}
	}()
	return load(src)
}

// boundImage scales img down to maxImageSide, and into a form Gio
// uploads directly.
func boundImage(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return img
	}
	if s := max(w, h); s > maxImageSide {
		w, h = max(1, w*maxImageSide/s), max(1, h*maxImageSide/s)
		dst := image.NewNRGBA(image.Rect(0, 0, w, h))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
		return dst
	}
	return img
}

// imageState reports src's image: loaded, failed, or still loading.
func imageState(src string, load func(string) (image.Image, error)) (img image.Image, failed bool) {
	e := loadImage(src, load)
	imageMu.Lock()
	defer imageMu.Unlock()
	return e.img, e.done && e.img == nil
}

// mdImage is a Markdown image: the picture once loaded, at its size in
// dp, narrowed to the space there is; its alt text while it loads and
// if it can't be.
type mdImage struct {
	src, alt string
	load     func(string) (image.Image, error)
	altStyle styles.Styles
}

func (m mdImage) Layout(gtx layout.Context) layout.Dimensions {
	img, failed := imageState(m.src, m.load)
	if img == nil {
		if !failed {
			// Poll until it's loaded; only while something loads.
			kit.RequestFrameAt(gtx, gtx.Now.Add(100*time.Millisecond))
		}
		alt := m.alt
		if alt == "" && failed {
			alt = "[image]"
		}
		if alt == "" {
			return layout.Dimensions{}
		}
		return elements.Text(alt, m.altStyle).Layout(gtx)
	}
	sz := img.Bounds().Size()
	w, h := gtx.Dp(unit.Dp(sz.X)), gtx.Dp(unit.Dp(sz.Y))
	if mx := gtx.Constraints.Max.X; w > mx && mx > 0 {
		h, w = h*mx/w, mx
	}
	w, h = max(w, 1), max(h, 1)
	size := image.Pt(w, h)
	defer clip.UniformRRect(image.Rectangle{Max: size}, gtx.Dp(4)).Push(gtx.Ops).Pop()
	gtx.Constraints = layout.Exact(size)
	widget.Image{Src: properties.ImageOp(img), Fit: widget.Fill}.Layout(gtx)
	return layout.Dimensions{Size: size}
}
