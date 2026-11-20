package camera

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"gioui.org/f32"
	"golang.org/x/sys/windows"

	"github.com/xmhrnx/giode"
)

// Media Foundation's source reader, through COM.

const supported = true

var (
	mfplat       = windows.NewLazySystemDLL("mfplat.dll")
	mf           = windows.NewLazySystemDLL("mf.dll")
	mfreadwrite  = windows.NewLazySystemDLL("mfreadwrite.dll")
	procStartup  = mfplat.NewProc("MFStartup")
	procAttrs    = mfplat.NewProc("MFCreateAttributes")
	procMType    = mfplat.NewProc("MFCreateMediaType")
	procEnumDev  = mf.NewProc("MFEnumDeviceSources")
	procReader   = mfreadwrite.NewProc("MFCreateSourceReaderFromMediaSource")
	mfStarted    sync.Once
	mfStartError error
)

func guid(d1 uint32, d2, d3 uint16, d4 ...byte) windows.GUID {
	g := windows.GUID{Data1: d1, Data2: d2, Data3: d3}
	copy(g.Data4[:], d4)
	return g
}

var (
	attrSourceType    = guid(0xc60ac5fe, 0x252a, 0x478f, 0xa0, 0xef, 0xbc, 0x8f, 0xa5, 0xf7, 0xca, 0xd3)
	attrVidcap        = guid(0x8ac3587a, 0x4ae7, 0x42d8, 0x99, 0xe0, 0x0a, 0x60, 0x13, 0xee, 0xf9, 0x0f)
	attrFriendlyName  = guid(0x60d0e559, 0x52f8, 0x4fa2, 0xbb, 0xce, 0xac, 0xdb, 0x34, 0xa8, 0xec, 0x01)
	attrSymbolicLink  = guid(0x58f0aad8, 0x22bf, 0x4f8a, 0xbb, 0x3d, 0xd2, 0xc4, 0x97, 0x8c, 0x6e, 0x2f)
	attrAdvancedVideo = guid(0x0f81da2c, 0xb537, 0x4672, 0xa8, 0xb2, 0xa6, 0x81, 0xb1, 0x73, 0x07, 0xa3)
	mtMajorType       = guid(0x48eba18e, 0xf8c9, 0x4687, 0xbf, 0x11, 0x0a, 0x74, 0xc9, 0xf9, 0x6a, 0x8f)
	mtSubtype         = guid(0xf7e34c9a, 0x42e8, 0x4714, 0xb7, 0x4b, 0xcb, 0x29, 0xd7, 0x2c, 0x35, 0xe5)
	mtFrameSize       = guid(0x1652c33d, 0xd6b2, 0x4012, 0xb8, 0x34, 0x72, 0x03, 0x08, 0x49, 0xa3, 0x7d)
	mtDefaultStride   = guid(0x644b4e48, 0x1e02, 0x4516, 0xb0, 0xeb, 0xc0, 0x1c, 0xa9, 0xd4, 0x9a, 0xc6)
	mediaTypeVideo    = guid(0x73646976, 0x0000, 0x0010, 0x80, 0x00, 0x00, 0xaa, 0x00, 0x38, 0x9b, 0x71)
	videoFormatRGB32  = guid(0x00000016, 0x0000, 0x0010, 0x80, 0x00, 0x00, 0xaa, 0x00, 0x38, 0x9b, 0x71)
	iidIMFMediaSource = guid(0x279a808d, 0xaec7, 0x40c8, 0x9c, 0x6b, 0xa6, 0xb4, 0x92, 0xc7, 0x8a, 0x66)
	firstVideoStream  = uintptr(0xFFFFFFFC)
	readerEndOfStream = uint32(0x2)
	readerError       = uint32(0x1)
	eAccessDenied     = uint32(0x80070005)
)

// The COM methods used, by vtable index.
const (
	mRelease            = 2
	aGetUINT32          = 7
	aGetUINT64          = 8
	aGetAllocatedString = 13
	aSetUINT32          = 21
	aSetGUID            = 24
	actActivateObject   = 33
	actShutdownObject   = 34
	rSetCurrentType     = 7
	rGetCurrentType     = 6
	rReadSample         = 9
	sToContiguous       = 41
	bLock, bUnlock      = 3, 4
	srcShutdown         = 12
)

type comIface struct{ vtbl *[64]uintptr }

func (o *comIface) call(m int, args ...uintptr) uintptr {
	r, _, _ := syscall.SyscallN(o.vtbl[m], append([]uintptr{uintptr(unsafe.Pointer(o))}, args...)...)
	return r
}

func (o *comIface) release() {
	if o != nil {
		o.call(mRelease)
	}
}

func hres(r uintptr, what string) error {
	if int32(r) < 0 {
		if uint32(r) == eAccessDenied {
			return fmt.Errorf("%w (Settings, Privacy, Camera)", ErrPermissionDenied)
		}
		return fmt.Errorf("camera: %s: 0x%08x", what, uint32(r))
	}
	return nil
}

func coInit() (func(), error) {
	err := windows.CoInitializeEx(0, windows.COINIT_MULTITHREADED)
	switch err {
	case nil, syscall.Errno(1):
		return windows.CoUninitialize, nil
	case syscall.Errno(0x80010106):
		return func() {}, nil
	}
	return nil, err
}

func mfStart() error {
	mfStarted.Do(func() {
		if err := mfplat.Load(); err != nil {
			mfStartError = giode.ErrNotSupported // Windows N without the Media Feature Pack
			return
		}
		r, _, _ := procStartup.Call(0x00020070, 0)
		mfStartError = hres(r, "starting Media Foundation")
	})
	return mfStartError
}

// enumerate lists the video capture devices' activation objects.
func enumerate() ([]*comIface, error) {
	if err := mfStart(); err != nil {
		return nil, err
	}
	var attrs *comIface
	if err := hres(callProc(procAttrs, uintptr(unsafe.Pointer(&attrs)), 1), "creating attributes"); err != nil {
		return nil, err
	}
	defer attrs.release()
	attrs.call(aSetGUID, uintptr(unsafe.Pointer(&attrSourceType)), uintptr(unsafe.Pointer(&attrVidcap)))
	var list *uintptr
	var n uint32
	if err := hres(callProc(procEnumDev, uintptr(unsafe.Pointer(attrs)), uintptr(unsafe.Pointer(&list)), uintptr(unsafe.Pointer(&n))), "listing cameras"); err != nil {
		return nil, err
	}
	if list == nil {
		return nil, nil
	}
	defer windows.CoTaskMemFree(unsafe.Pointer(list))
	out := make([]*comIface, n)
	for i, p := range unsafe.Slice(list, n) {
		out[i] = *(**comIface)(unsafe.Pointer(&p))
	}
	return out, nil
}

func callProc(p *windows.LazyProc, args ...uintptr) uintptr {
	r, _, _ := p.Call(args...)
	return r
}

func attrString(o *comIface, key *windows.GUID) string {
	var p *uint16
	var n uint32
	if int32(o.call(aGetAllocatedString, uintptr(unsafe.Pointer(key)), uintptr(unsafe.Pointer(&p)), uintptr(unsafe.Pointer(&n)))) < 0 || p == nil {
		return ""
	}
	defer windows.CoTaskMemFree(unsafe.Pointer(p))
	return windows.UTF16PtrToString(p)
}

// onCOM runs fn on a thread with COM.
func onCOM(fn func() error) error {
	done := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		uninit, err := coInit()
		if err != nil {
			done <- err
			return
		}
		defer uninit()
		done <- fn()
	}()
	return <-done
}

func devices(ctx context.Context) ([]Device, error) {
	var out []Device
	err := onCOM(func() error {
		acts, err := enumerate()
		if err != nil {
			return err
		}
		for _, a := range acts {
			out = append(out, Device{ID: attrString(a, &attrSymbolicLink), Name: attrString(a, &attrFriendlyName)})
			a.release()
		}
		return nil
	})
	return out, err
}

func requestPermission(context.Context, *giode.App) (bool, error) { return true, nil }

type platformCamera struct {
	c    *Camera
	stop chan struct{}
	done chan struct{}
}

func (c *Camera) open(ctx context.Context) error {
	p := platformCamera{c: c, stop: make(chan struct{}), done: make(chan struct{})}
	started := make(chan error, 1)
	go func() {
		defer close(p.done)
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		uninit, err := coInit()
		if err != nil {
			started <- err
			return
		}
		defer uninit()
		reader, source, w, h, stride, err := openReader(c.opts.Device)
		if err != nil {
			started <- err
			return
		}
		defer source.call(srcShutdown)
		defer source.release()
		defer reader.release()
		started <- nil
		for {
			select {
			case <-p.stop:
				return
			default:
			}
			var idx, flags uint32
			var ts int64
			var sample *comIface
			if err := hres(reader.call(rReadSample, firstVideoStream, 0, uintptr(unsafe.Pointer(&idx)), uintptr(unsafe.Pointer(&flags)),
				uintptr(unsafe.Pointer(&ts)), uintptr(unsafe.Pointer(&sample))), "reading a frame"); err != nil {
				c.reportError(err)
				return
			}
			if flags&(readerEndOfStream|readerError) != 0 {
				c.reportError(errors.New("camera: the camera stopped"))
				return
			}
			if sample == nil {
				continue
			}
			frame := copySample(sample, w, h, stride)
			sample.release()
			if frame != nil {
				c.deliver(rawFrame{bgra: frame, w: w, h: h, stride: w * 4})
			}
		}
	}()
	if err := <-started; err != nil {
		<-p.done
		return err
	}
	c.plat = p
	return nil
}

// openReader activates a camera and opens a source reader converting
// its frames to 32-bit BGRA.
func openReader(id string) (reader, source *comIface, w, h, stride int, err error) {
	acts, err := enumerate()
	if err != nil {
		return
	}
	defer func() {
		for _, a := range acts {
			a.release()
		}
	}()
	var act *comIface
	for _, a := range acts {
		if id == "" || attrString(a, &attrSymbolicLink) == id {
			act = a
			break
		}
	}
	if act == nil {
		err = errors.New("camera: no camera found")
		return
	}
	if err = hres(act.call(actActivateObject, uintptr(unsafe.Pointer(&iidIMFMediaSource)), uintptr(unsafe.Pointer(&source))), "opening the camera"); err != nil {
		return
	}
	var attrs *comIface
	if err = hres(callProc(procAttrs, uintptr(unsafe.Pointer(&attrs)), 1), "creating attributes"); err != nil {
		source.release()
		return
	}
	attrs.call(aSetUINT32, uintptr(unsafe.Pointer(&attrAdvancedVideo)), 1)
	err = hres(callProc(procReader, uintptr(unsafe.Pointer(source)), uintptr(unsafe.Pointer(attrs)), uintptr(unsafe.Pointer(&reader))), "creating the reader")
	attrs.release()
	if err != nil {
		source.call(srcShutdown)
		source.release()
		return
	}
	var mt *comIface
	if err = hres(callProc(procMType, uintptr(unsafe.Pointer(&mt))), "creating a media type"); err == nil {
		mt.call(aSetGUID, uintptr(unsafe.Pointer(&mtMajorType)), uintptr(unsafe.Pointer(&mediaTypeVideo)))
		mt.call(aSetGUID, uintptr(unsafe.Pointer(&mtSubtype)), uintptr(unsafe.Pointer(&videoFormatRGB32)))
		err = hres(reader.call(rSetCurrentType, firstVideoStream, 0, uintptr(unsafe.Pointer(mt))), "choosing BGRA frames")
		mt.release()
	}
	if err == nil {
		var cur *comIface
		if err = hres(reader.call(rGetCurrentType, firstVideoStream, uintptr(unsafe.Pointer(&cur))), "reading the format"); err == nil {
			var size uint64
			cur.call(aGetUINT64, uintptr(unsafe.Pointer(&mtFrameSize)), uintptr(unsafe.Pointer(&size)))
			w, h = int(size>>32), int(uint32(size))
			var s uint32
			if int32(cur.call(aGetUINT32, uintptr(unsafe.Pointer(&mtDefaultStride)), uintptr(unsafe.Pointer(&s)))) >= 0 {
				stride = int(int32(s))
			} else {
				stride = w * 4
			}
			cur.release()
		}
	}
	if err == nil && (w <= 0 || h <= 0) {
		err = errors.New("camera: the camera has no frame size")
	}
	if err != nil {
		reader.release()
		source.call(srcShutdown)
		source.release()
	}
	return
}

// copySample copies a sample's frame out, upright: a negative stride is
// bottom-up.
func copySample(sample *comIface, w, h, stride int) []byte {
	var buf *comIface
	if int32(sample.call(sToContiguous, uintptr(unsafe.Pointer(&buf)))) < 0 || buf == nil {
		return nil
	}
	defer buf.release()
	var data *byte
	var maxLen, curLen uint32
	if int32(buf.call(bLock, uintptr(unsafe.Pointer(&data)), uintptr(unsafe.Pointer(&maxLen)), uintptr(unsafe.Pointer(&curLen)))) < 0 || data == nil {
		return nil
	}
	defer buf.call(bUnlock)
	abs := stride
	if abs < 0 {
		abs = -abs
	}
	if int(curLen) < abs*h {
		return nil
	}
	src := unsafe.Slice(data, int(curLen))
	out := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		sy := y
		if stride < 0 {
			sy = h - 1 - y
		}
		row := src[sy*abs : sy*abs+w*4]
		dst := out[y*w*4 : (y+1)*w*4]
		copy(dst, row)
		for x := 3; x < len(dst); x += 4 {
			dst[x] = 255 // RGB32's fourth byte is padding
		}
	}
	return out
}

func (p platformCamera) close() {
	if p.stop == nil {
		return
	}
	close(p.stop)
	<-p.done
}

func (p platformCamera) capture(ctx context.Context) ([]byte, error) { return captureLatest(ctx, p.c) }
func (platformCamera) hasTorch() bool                                { return false }
func (platformCamera) setTorch(bool) error                           { return giode.ErrNotSupported }
func (platformCamera) focus(f32.Point) error                         { return nil }
