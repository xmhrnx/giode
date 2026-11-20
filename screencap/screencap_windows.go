package screencap

import (
	"context"
	"fmt"
	"image"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/xmhrnx/giode"
)

// GDI: BitBlt from the screen into a DIB section.

var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	gdi32                   = windows.NewLazySystemDLL("gdi32.dll")
	procEnumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfoW     = user32.NewProc("GetMonitorInfoW")
	procGetDC               = user32.NewProc("GetDC")
	procReleaseDC           = user32.NewProc("ReleaseDC")
	procCreateCompatibleDC  = gdi32.NewProc("CreateCompatibleDC")
	procCreateDIBSection    = gdi32.NewProc("CreateDIBSection")
	procSelectObject        = gdi32.NewProc("SelectObject")
	procBitBlt              = gdi32.NewProc("BitBlt")
	procDeleteObject        = gdi32.NewProc("DeleteObject")
	procDeleteDC            = gdi32.NewProc("DeleteDC")
	procGdiFlush            = gdi32.NewProc("GdiFlush")
)

type monitorInfoEx struct {
	size    uint32
	monitor windows.Rect
	work    windows.Rect
	flags   uint32
	device  [32]uint16
}

type bitmapInfoHeader struct {
	size          uint32
	width, height int32
	planes, bits  uint16
	compression   uint32
	sizeImage     uint32
	xppm, yppm    int32
	clrUsed       uint32
	clrImportant  uint32
}

const (
	srcCopy        = 0x00CC0020
	captureBlt     = 0x40000000
	monitorPrimary = 1
)

func sources(ctx context.Context) ([]Source, error) {
	var out []Source
	cb := syscall.NewCallback(func(hmon, hdc, rect, data uintptr) uintptr {
		mi := monitorInfoEx{size: uint32(unsafe.Sizeof(monitorInfoEx{}))}
		if r, _, _ := procGetMonitorInfoW.Call(hmon, uintptr(unsafe.Pointer(&mi))); r != 0 {
			name := windows.UTF16ToString(mi.device[:])
			m := mi.monitor
			out = append(out, Source{ID: name, Name: fmt.Sprintf("Display %d", len(out)+1), Primary: mi.flags&monitorPrimary != 0,
				Bounds: image.Rect(int(m.Left), int(m.Top), int(m.Right), int(m.Bottom))})
		}
		return 1
	})
	procEnumDisplayMonitors.Call(0, 0, cb, 0)
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: no displays", giode.ErrNotSupported)
	}
	return out, nil
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
	b := src.Bounds
	w, h := b.Dx(), b.Dy()
	p := platformCapture{stopCh: make(chan struct{}), done: make(chan struct{})}
	started := make(chan error, 1)
	go func() {
		defer close(p.done)
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		screen, _, _ := procGetDC.Call(0)
		if screen == 0 {
			started <- fmt.Errorf("screencap: no screen device context")
			return
		}
		defer procReleaseDC.Call(0, screen)
		mem, _, _ := procCreateCompatibleDC.Call(screen)
		defer procDeleteDC.Call(mem)
		bi := bitmapInfoHeader{size: uint32(unsafe.Sizeof(bitmapInfoHeader{})), width: int32(w), height: -int32(h), planes: 1, bits: 32}
		var bits unsafe.Pointer
		bmp, _, _ := procCreateDIBSection.Call(mem, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
		if bmp == 0 || bits == nil {
			started <- fmt.Errorf("screencap: creating a bitmap")
			return
		}
		defer procDeleteObject.Call(bmp)
		old, _, _ := procSelectObject.Call(mem, bmp)
		defer procSelectObject.Call(mem, old)
		started <- nil
		pix := unsafe.Slice((*byte)(bits), w*h*4)
		c.ticker(p.stopCh, func() error {
			if c.busy.Load() {
				return nil
			}
			r, _, _ := procBitBlt.Call(mem, 0, 0, uintptr(w), uintptr(h), screen, uintptr(b.Min.X), uintptr(b.Min.Y), srcCopy|captureBlt)
			if r == 0 {
				return nil // a secure desktop (UAC, lock screen): try again
			}
			procGdiFlush.Call()
			c.deliver(c.fromBGRA(pix, w, h, w*4))
			return nil
		})
	}()
	if err := <-started; err != nil {
		<-p.done
		return err
	}
	c.plat = p
	return nil
}
