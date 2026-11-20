//go:build linux && !android

package camera

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"unsafe"

	"gioui.org/f32"
	"golang.org/x/sys/unix"

	"github.com/xmhrnx/giode"
)

// Video4Linux, through its ioctls.

const supported = true

type v4lCapability struct {
	driver       [16]byte
	card         [32]byte
	busInfo      [32]byte
	version      uint32
	capabilities uint32
	deviceCaps   uint32
	_            [3]uint32
}

type v4lFmtDesc struct {
	index       uint32
	typ         uint32
	flags       uint32
	description [32]byte
	pixelformat uint32
	mbusCode    uint32
	_           [3]uint32
}

type v4lPixFormat struct {
	width, height, pixelformat, field, bytesperline, sizeimage, colorspace, priv, flags, ycbcr, quant, xfer uint32
}

type v4lFormat struct {
	typ uint32
	// The union: 200 bytes, aligned for pointers.
	fmt [25]uint64
}

type v4lRequestBuffers struct {
	count, typ, memory, capabilities uint32
	flags                            uint8
	_                                [3]uint8
}

type v4lBuffer struct {
	index, typ, bytesused, flags, field uint32
	timestamp                           unix.Timeval
	timecode                            [16]byte
	sequence, memory                    uint32
	m                                   uintptr // the offset, for mmap
	length, reserved2, requestFd        uint32
}

const (
	capVideoCapture  = 0x00000001
	capStreaming     = 0x04000000
	capDeviceCaps    = 0x80000000
	bufTypeCapture   = 1
	memoryMmap       = 1
	fieldAny         = 0
	pixYUYV          = 'Y' | 'U'<<8 | 'Y'<<16 | 'V'<<24
	pixMJPEG         = 'M' | 'J'<<8 | 'P'<<16 | 'G'<<24
	iocRead, iocRW   = 2, 3
	iocWrite         = 1
	v4lType          = 'V'
	captureBuffers   = 4
	previewWidthWant = 1280
)

func ioc(dir, nr, size uintptr) uintptr { return dir<<30 | size<<16 | v4lType<<8 | nr }

var (
	vidiocQueryCap  = ioc(iocRead, 0, unsafe.Sizeof(v4lCapability{}))
	vidiocEnumFmt   = ioc(iocRW, 2, unsafe.Sizeof(v4lFmtDesc{}))
	vidiocSFmt      = ioc(iocRW, 5, unsafe.Sizeof(v4lFormat{}))
	vidiocReqBufs   = ioc(iocRW, 8, unsafe.Sizeof(v4lRequestBuffers{}))
	vidiocQueryBuf  = ioc(iocRW, 9, unsafe.Sizeof(v4lBuffer{}))
	vidiocQBuf      = ioc(iocRW, 15, unsafe.Sizeof(v4lBuffer{}))
	vidiocDQBuf     = ioc(iocRW, 17, unsafe.Sizeof(v4lBuffer{}))
	vidiocStreamOn  = ioc(iocWrite, 18, 4)
	vidiocStreamOff = ioc(iocWrite, 19, 4)
)

func ioctl(fd int, req uintptr, arg unsafe.Pointer) error {
	for {
		_, _, e := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), req, uintptr(arg))
		if e == unix.EINTR {
			continue
		}
		if e != 0 {
			return e
		}
		return nil
	}
}

func cstring(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return string(b)
}

// capture reports whether the device at path captures video, and its
// name.
func probe(path string) (string, bool) {
	fd, err := unix.Open(path, unix.O_RDWR|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return "", false
	}
	defer unix.Close(fd)
	var c v4lCapability
	if ioctl(fd, vidiocQueryCap, unsafe.Pointer(&c)) != nil {
		return "", false
	}
	caps := c.capabilities
	if caps&capDeviceCaps != 0 {
		caps = c.deviceCaps
	}
	return cstring(c.card[:]), caps&capVideoCapture != 0 && caps&capStreaming != 0
}

func devices(ctx context.Context) ([]Device, error) {
	paths, _ := filepath.Glob("/dev/video*")
	sort.Slice(paths, func(i, j int) bool {
		return len(paths[i]) < len(paths[j]) || len(paths[i]) == len(paths[j]) && paths[i] < paths[j]
	})
	var out []Device
	for _, p := range paths {
		if name, ok := probe(p); ok {
			out = append(out, Device{ID: p, Name: name})
		}
	}
	return out, nil
}

func requestPermission(context.Context, *giode.App) (bool, error) { return true, nil }

type platformCamera struct {
	v *v4lCam
	c *Camera
}

type v4lCam struct {
	fd      int
	bufs    [][]byte
	w, h    int
	stride  int
	format  uint32
	stop    chan struct{}
	done    chan struct{}
	mu      sync.Mutex
	last    *image.RGBA // the newest frame, for Capture
	lastErr error
}

func (c *Camera) open(ctx context.Context) error {
	path := c.opts.Device
	if path == "" {
		ds, _ := devices(ctx)
		if len(ds) == 0 {
			return errors.New("camera: no camera found")
		}
		path = ds[0].ID
	}
	fd, err := unix.Open(path, unix.O_RDWR|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("%w: %s isn't readable (add the user to the video group)", ErrPermissionDenied, path)
		}
		return err
	}
	v := &v4lCam{fd: fd, stop: make(chan struct{}), done: make(chan struct{})}
	if err := v.start(); err != nil {
		unix.Close(fd)
		return err
	}
	c.plat = platformCamera{v: v, c: c}
	go v.run(c)
	return nil
}

// pickFormat chooses YUYV, which converts quickly, or MJPEG.
func (v *v4lCam) pickFormat() (uint32, error) {
	have := map[uint32]bool{}
	for i := uint32(0); i < 64; i++ {
		d := v4lFmtDesc{index: i, typ: bufTypeCapture}
		if ioctl(v.fd, vidiocEnumFmt, unsafe.Pointer(&d)) != nil {
			break
		}
		have[d.pixelformat] = true
	}
	switch {
	case have[pixYUYV]:
		return pixYUYV, nil
	case have[pixMJPEG]:
		return pixMJPEG, nil
	}
	return 0, errors.New("camera: the camera has neither YUYV nor MJPEG frames")
}

func (v *v4lCam) start() error {
	pf, err := v.pickFormat()
	if err != nil {
		return err
	}
	// Ask for 720p; the driver picks the nearest it has.
	f := v4lFormat{typ: bufTypeCapture}
	pix := (*v4lPixFormat)(unsafe.Pointer(&f.fmt))
	pix.width, pix.height, pix.pixelformat, pix.field = previewWidthWant, previewWidthWant*9/16, pf, fieldAny
	if err := ioctl(v.fd, vidiocSFmt, unsafe.Pointer(&f)); err != nil {
		return fmt.Errorf("camera: setting the format: %w", err)
	}
	v.w, v.h, v.format = int(pix.width), int(pix.height), pix.pixelformat
	v.stride = int(pix.bytesperline)
	if v.stride == 0 {
		v.stride = v.w * 2
	}
	req := v4lRequestBuffers{count: captureBuffers, typ: bufTypeCapture, memory: memoryMmap}
	if err := ioctl(v.fd, vidiocReqBufs, unsafe.Pointer(&req)); err != nil {
		return fmt.Errorf("camera: requesting buffers: %w", err)
	}
	for i := uint32(0); i < req.count; i++ {
		b := v4lBuffer{index: i, typ: bufTypeCapture, memory: memoryMmap}
		if err := ioctl(v.fd, vidiocQueryBuf, unsafe.Pointer(&b)); err != nil {
			return err
		}
		m, err := unix.Mmap(v.fd, int64(uint32(b.m)), int(b.length), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
		if err != nil {
			return err
		}
		v.bufs = append(v.bufs, m)
		if err := ioctl(v.fd, vidiocQBuf, unsafe.Pointer(&b)); err != nil {
			return err
		}
	}
	typ := uint32(bufTypeCapture)
	if err := ioctl(v.fd, vidiocStreamOn, unsafe.Pointer(&typ)); err != nil {
		return fmt.Errorf("camera: starting: %w", err)
	}
	return nil
}

func (v *v4lCam) run(c *Camera) {
	defer close(v.done)
	bgra := make([]byte, v.w*v.h*4)
	for {
		select {
		case <-v.stop:
			return
		default:
		}
		fds := []unix.PollFd{{Fd: int32(v.fd), Events: unix.POLLIN}}
		n, err := unix.Poll(fds, 200)
		if err != nil && err != unix.EINTR {
			c.reportError(fmt.Errorf("camera: %w", err))
			return
		}
		if n == 0 {
			continue
		}
		b := v4lBuffer{typ: bufTypeCapture, memory: memoryMmap}
		if err := ioctl(v.fd, vidiocDQBuf, unsafe.Pointer(&b)); err != nil {
			if err == unix.EAGAIN {
				continue
			}
			c.reportError(fmt.Errorf("camera: %w", err))
			return
		}
		data := v.bufs[b.index][:b.bytesused]
		ok := true
		switch v.format {
		case pixYUYV:
			yuyvToBGRA(bgra, data, v.w, v.h, v.stride)
		case pixMJPEG:
			ok = mjpegToBGRA(bgra, data, v.w, v.h)
		}
		ioctl(v.fd, vidiocQBuf, unsafe.Pointer(&b))
		if ok {
			frame := append([]byte(nil), bgra...)
			c.deliver(rawFrame{bgra: frame, w: v.w, h: v.h, stride: v.w * 4})
		}
	}
}

// yuyvToBGRA converts packed 4:2:2 YUV (BT.601, limited range).
func yuyvToBGRA(dst, src []byte, w, h, stride int) {
	for y := 0; y < h; y++ {
		row := src[y*stride:]
		out := dst[y*w*4:]
		for x := 0; x+1 < w; x += 2 {
			i := x * 2
			if i+3 >= len(row) {
				break
			}
			y0, u, y1, v := int(row[i]), int(row[i+1])-128, int(row[i+2]), int(row[i+3])-128
			for k, yy := range [2]int{y0, y1} {
				c := (yy - 16) * 298
				r := (c + 409*v + 128) >> 8
				g := (c - 100*u - 208*v + 128) >> 8
				b := (c + 516*u + 128) >> 8
				o := (x + k) * 4
				out[o], out[o+1], out[o+2], out[o+3] = clamp8(b), clamp8(g), clamp8(r), 255
			}
		}
	}
}

func mjpegToBGRA(dst, src []byte, w, h int) bool {
	img, err := jpeg.Decode(bytes.NewReader(src))
	if err != nil {
		return false // a torn frame: skip it
	}
	b := img.Bounds()
	if b.Dx() != w || b.Dy() != h {
		return false
	}
	if yc, ok := img.(*image.YCbCr); ok {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				yy := int(yc.Y[yc.YOffset(x+b.Min.X, y+b.Min.Y)])
				ci := yc.COffset(x+b.Min.X, y+b.Min.Y)
				u, v := int(yc.Cb[ci])-128, int(yc.Cr[ci])-128
				// JPEG is full range.
				r := yy + (91881*v)>>16
				g := yy - (22554*u+46802*v)>>16
				bb := yy + (116130*u)>>16
				o := (y*w + x) * 4
				dst[o], dst[o+1], dst[o+2], dst[o+3] = clamp8(bb), clamp8(g), clamp8(r), 255
			}
		}
		return true
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bb, _ := img.At(x+b.Min.X, y+b.Min.Y).RGBA()
			o := (y*w + x) * 4
			dst[o], dst[o+1], dst[o+2], dst[o+3] = uint8(bb>>8), uint8(g>>8), uint8(r>>8), 255
		}
	}
	return true
}

func (p platformCamera) close() {
	v := p.v
	if v == nil {
		return
	}
	close(v.stop)
	<-v.done
	typ := uint32(bufTypeCapture)
	ioctl(v.fd, vidiocStreamOff, unsafe.Pointer(&typ))
	for _, b := range v.bufs {
		unix.Munmap(b)
	}
	unix.Close(v.fd)
}

// capture encodes the newest preview frame: webcams stream at the
// resolution they take pictures at.
func (p platformCamera) capture(ctx context.Context) ([]byte, error) {
	return captureLatest(ctx, p.c)
}

func (platformCamera) hasTorch() bool        { return false }
func (platformCamera) setTorch(bool) error   { return giode.ErrNotSupported }
func (platformCamera) focus(f32.Point) error { return nil }
