//go:build linux && !android

package screencap

import (
	"context"
	"fmt"
	"image"
	"io"
	"log"
	"os"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/randr"
	"github.com/jezek/xgb/xproto"

	"github.com/xmhrnx/giode"
)

func init() {
	// xgb's notes about missing authority files aren't the app's.
	xgb.Logger = log.New(io.Discard, "", 0)
}

func connect() (*xgb.Conn, error) {
	if os.Getenv("DISPLAY") == "" {
		return nil, fmt.Errorf("%w: no X11 display (Wayland sessions need Xwayland)", giode.ErrNotSupported)
	}
	return xgb.NewConn()
}

func sources(ctx context.Context) ([]Source, error) {
	c, err := connect()
	if err != nil {
		return nil, err
	}
	defer c.Close()
	root := xproto.Setup(c).DefaultScreen(c).Root
	if randr.Init(c) == nil {
		if r, err := randr.GetMonitors(c, root, true).Reply(); err == nil && len(r.Monitors) > 0 {
			var out []Source
			for i, m := range r.Monitors {
				name := fmt.Sprintf("Display %d", i+1)
				if a, err := xproto.GetAtomName(c, m.Name).Reply(); err == nil && a.Name != "" {
					name = a.Name
				}
				out = append(out, Source{ID: name, Name: name, Primary: m.Primary,
					Bounds: image.Rect(int(m.X), int(m.Y), int(m.X)+int(m.Width), int(m.Y)+int(m.Height))})
			}
			return out, nil
		}
	}
	s := xproto.Setup(c).DefaultScreen(c)
	return []Source{{ID: "screen", Name: "Screen", Primary: true, Bounds: image.Rect(0, 0, int(s.WidthInPixels), int(s.HeightInPixels))}}, nil
}

type platformCapture struct {
	stopCh chan struct{}
	done   chan struct{}
}

func (p platformCapture) stop() {
	if p.stopCh != nil {
		close(p.stopCh)
		<-p.done
	}
}

func (c *Capture) start(ctx context.Context, _ *giode.App) error {
	srcs, err := sources(ctx)
	if err != nil {
		return err
	}
	var src *Source
	for i := range srcs {
		if srcs[i].ID == c.opts.Source || c.opts.Source == "" && srcs[i].Primary {
			src = &srcs[i]
		}
	}
	if src == nil {
		if c.opts.Source != "" {
			return fmt.Errorf("screencap: no display %q", c.opts.Source)
		}
		src = &srcs[0]
	}
	conn, err := connect()
	if err != nil {
		return err
	}
	screen := xproto.Setup(conn).DefaultScreen(conn)
	if screen.RootDepth != 24 && screen.RootDepth != 32 {
		conn.Close()
		return fmt.Errorf("%w: a %d-bit display", giode.ErrNotSupported, screen.RootDepth)
	}
	b := src.Bounds
	p := platformCapture{stopCh: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(p.done)
		defer conn.Close()
		c.ticker(p.stopCh, func() error {
			if c.busy.Load() {
				return nil
			}
			// Tall displays are read in bands, within a request's size.
			w, h := b.Dx(), b.Dy()
			buf := make([]byte, w*h*4)
			band := max(1, (4<<20)/(w*4))
			for y := 0; y < h; y += band {
				n := min(band, h-y)
				r, err := xproto.GetImage(conn, xproto.ImageFormatZPixmap, xproto.Drawable(screen.Root),
					int16(b.Min.X), int16(b.Min.Y+y), uint16(w), uint16(n), 0xffffffff).Reply()
				if err != nil {
					return err
				}
				copy(buf[y*w*4:], r.Data)
			}
			c.deliver(c.fromBGRA(buf, w, h, w*4))
			return nil
		})
	}()
	c.plat = p
	return nil
}
