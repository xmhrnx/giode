package audio

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/xmhrnx/giode"
)

// WASAPI, through COM.

var (
	ole32                = windows.NewLazySystemDLL("ole32.dll")
	procCoCreateInstance = ole32.NewProc("CoCreateInstance")
	procPropVariantClear = ole32.NewProc("PropVariantClear")
)

var (
	clsidMMDeviceEnumerator = windows.GUID{Data1: 0xBCDE0395, Data2: 0xE52F, Data3: 0x467C, Data4: [8]byte{0x8E, 0x3D, 0xC4, 0x57, 0x92, 0x91, 0x69, 0x2E}}
	iidIMMDeviceEnumerator  = windows.GUID{Data1: 0xA95664D2, Data2: 0x9614, Data3: 0x4F35, Data4: [8]byte{0xA7, 0x46, 0xDE, 0x8D, 0xB6, 0x36, 0x17, 0xE6}}
	iidIAudioClient         = windows.GUID{Data1: 0x1CB9AD4C, Data2: 0xDBFA, Data3: 0x4C32, Data4: [8]byte{0xB1, 0x78, 0xC2, 0xF5, 0x68, 0xA7, 0x03, 0xB2}}
	iidIAudioClient2        = windows.GUID{Data1: 0x726778CD, Data2: 0xF60A, Data3: 0x4EDA, Data4: [8]byte{0x82, 0xDE, 0xE4, 0x76, 0x10, 0xCD, 0x78, 0xAA}}
	iidIAudioCaptureClient  = windows.GUID{Data1: 0xC8ADBD64, Data2: 0xE71E, Data3: 0x48A0, Data4: [8]byte{0xA4, 0xDE, 0x18, 0x5C, 0x39, 0x5C, 0xD3, 0x17}}
	iidIAudioRenderClient   = windows.GUID{Data1: 0xF294ACFC, Data2: 0x3146, Data3: 0x4483, Data4: [8]byte{0xA7, 0xBF, 0xAD, 0xDC, 0xA7, 0xC2, 0x60, 0xE2}}
	pkeyFriendlyName        = propertyKey{windows.GUID{Data1: 0xA45C254E, Data2: 0xDF1C, Data3: 0x4EFD, Data4: [8]byte{0x80, 0x20, 0x67, 0xD1, 0x46, 0xA8, 0x50, 0xE0}}, 14}
)

const (
	eRender, eCapture   = 0, 1
	eConsole, eComm     = 0, 2
	deviceStateActive   = 1
	clsctxAll           = 0x17
	streamEventCallback = 0x00040000
	streamAutoConvert   = 0x80000000
	streamSRCDefault    = 0x08000000
	categoryComm        = 3
	bufferFlagSilent    = 0x2
)

type propertyKey struct {
	fmtid windows.GUID
	pid   uint32
}

// propVariant is a PROPVARIANT: its type, then (on 64-bit at 8, on
// 32-bit at 8 too) the value.
type propVariant struct {
	vt  uint16
	_   [3]uint16
	val uintptr
	_   uintptr
	_   uintptr
}

type waveFormatEx struct {
	formatTag      uint16
	channels       uint16
	samplesPerSec  uint32
	avgBytesPerSec uint32
	blockAlign     uint16
	bitsPerSample  uint16
	cbSize         uint16
}

type audioClientProperties struct {
	cbSize    uint32
	isOffload int32
	category  int32
	options   uint32
}

// comIface is a COM object: its first word points to its methods.
type comIface struct {
	vtbl *[32]uintptr
}

// comObj is a COM interface pointer.
type comObj = *comIface

func (o *comIface) call(method int, args ...uintptr) uintptr {
	r, _, _ := syscall.SyscallN(o.vtbl[method], append([]uintptr{uintptr(unsafe.Pointer(o))}, args...)...)
	return r
}

func (o *comIface) release() {
	if o != nil {
		o.call(2)
	}
}

func hr(r uintptr, what string) error {
	if int32(r) < 0 {
		return fmt.Errorf("audio: %s: 0x%08x", what, uint32(r))
	}
	return nil
}

// coInit initializes COM on the thread, returning how to undo it. A
// thread already in a single-threaded apartment works as it is.
func coInit() (func(), error) {
	err := windows.CoInitializeEx(0, windows.COINIT_MULTITHREADED)
	switch err {
	case nil, syscall.Errno(1): // S_OK, S_FALSE
		return windows.CoUninitialize, nil
	case syscall.Errno(0x80010106): // RPC_E_CHANGED_MODE
		return func() {}, nil
	}
	return nil, err
}

// withCOM runs fn on a locked thread with COM initialized.
func withCOM(fn func() error) error {
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

func enumerator() (comObj, error) {
	var e comObj
	r, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsidMMDeviceEnumerator)), 0, clsctxAll,
		uintptr(unsafe.Pointer(&iidIMMDeviceEnumerator)), uintptr(unsafe.Pointer(&e)))
	return e, hr(r, "creating the device enumerator")
}

func deviceID(dev comObj) string {
	var p *uint16
	if int32(dev.call(5, uintptr(unsafe.Pointer(&p)))) < 0 || p == nil {
		return ""
	}
	defer windows.CoTaskMemFree(unsafe.Pointer(p))
	return windows.UTF16PtrToString(p)
}

func deviceName(dev comObj) string {
	var store comObj
	if int32(dev.call(4, 0, uintptr(unsafe.Pointer(&store)))) < 0 { // STGM_READ
		return ""
	}
	defer store.release()
	var pv propVariant
	if int32(store.call(5, uintptr(unsafe.Pointer(&pkeyFriendlyName)), uintptr(unsafe.Pointer(&pv)))) < 0 {
		return ""
	}
	defer procPropVariantClear.Call(uintptr(unsafe.Pointer(&pv)))
	if pv.vt != 31 || pv.val == 0 { // VT_LPWSTR
		return ""
	}
	return windows.UTF16PtrToString(*(**uint16)(unsafe.Pointer(&pv.val)))
}

func devices(ctx context.Context, input bool) ([]Device, error) {
	flow := uintptr(eRender)
	if input {
		flow = eCapture
	}
	var out []Device
	err := withCOM(func() error {
		e, err := enumerator()
		if err != nil {
			return err
		}
		defer e.release()
		def := ""
		var d comObj
		if int32(e.call(4, flow, eConsole, uintptr(unsafe.Pointer(&d)))) >= 0 {
			def = deviceID(d)
			d.release()
		}
		var coll comObj
		if err := hr(e.call(3, flow, deviceStateActive, uintptr(unsafe.Pointer(&coll))), "listing devices"); err != nil {
			return err
		}
		defer coll.release()
		var n uint32
		coll.call(3, uintptr(unsafe.Pointer(&n)))
		for i := uint32(0); i < n; i++ {
			var dev comObj
			if int32(coll.call(4, uintptr(i), uintptr(unsafe.Pointer(&dev)))) < 0 {
				continue
			}
			id := deviceID(dev)
			out = append(out, Device{ID: id, Name: deviceName(dev), Default: id == def})
			dev.release()
		}
		return nil
	})
	return out, err
}

func requestPermission(context.Context, *giode.App) (bool, error) { return true, nil }

// stream is a running WASAPI stream on its own thread.
type stream struct {
	stop chan struct{}
	done chan struct{}
}

func (s *stream) close() error {
	if s == nil || s.stop == nil {
		return nil
	}
	close(s.stop)
	<-s.done
	return nil
}

// openClient activates and initializes an audio client on the device.
func openClient(opts Options, capture bool) (client comObj, event windows.Handle, err error) {
	e, err := enumerator()
	if err != nil {
		return nil, 0, err
	}
	defer e.release()
	var dev comObj
	flow, role := uintptr(eRender), uintptr(eConsole)
	if capture {
		flow = eCapture
	}
	if opts.Voice {
		role = eComm
	}
	if opts.Device != "" {
		id, _ := windows.UTF16PtrFromString(opts.Device)
		err = hr(e.call(5, uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(&dev))), "opening the device")
	} else {
		err = hr(e.call(4, flow, role, uintptr(unsafe.Pointer(&dev))), "opening the default device")
	}
	if err != nil {
		return nil, 0, err
	}
	defer dev.release()
	if err := hr(dev.call(3, uintptr(unsafe.Pointer(&iidIAudioClient2)), clsctxAll, 0, uintptr(unsafe.Pointer(&client))), "activating the device"); err != nil {
		// Before Windows 8: no IAudioClient2, so no communications category.
		if err := hr(dev.call(3, uintptr(unsafe.Pointer(&iidIAudioClient)), clsctxAll, 0, uintptr(unsafe.Pointer(&client))), "activating the device"); err != nil {
			return nil, 0, err
		}
	} else if opts.Voice {
		props := audioClientProperties{cbSize: uint32(unsafe.Sizeof(audioClientProperties{})), category: categoryComm}
		client.call(16, uintptr(unsafe.Pointer(&props)))
	}
	ch := uint16(opts.Channels)
	wf := waveFormatEx{formatTag: 1, channels: ch, samplesPerSec: uint32(opts.SampleRate), bitsPerSample: 16,
		blockAlign: ch * 2, avgBytesPerSec: uint32(opts.SampleRate) * uint32(ch) * 2}
	dur := int64(100 * time.Millisecond / 100) // in 100 ns units
	flags := uintptr(streamEventCallback | streamAutoConvert | streamSRCDefault)
	if err := hr(client.call(3, 0, flags, uintptr(dur), 0, uintptr(unsafe.Pointer(&wf)), 0), "initializing the stream"); err != nil {
		client.release()
		return nil, 0, err
	}
	event, err = windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		client.release()
		return nil, 0, err
	}
	if err := hr(client.call(13, uintptr(event)), "setting the event"); err != nil {
		client.release()
		windows.CloseHandle(event)
		return nil, 0, err
	}
	return client, event, nil
}

// run starts a stream's thread: it opens the client there and calls
// pump on every buffer event until stopped.
func run(opts Options, capture bool, service *windows.GUID, pump func(client, svc comObj) error) (*stream, error) {
	s := &stream{stop: make(chan struct{}), done: make(chan struct{})}
	started := make(chan error, 1)
	go func() {
		defer close(s.done)
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		uninit, err := coInit()
		if err != nil {
			started <- err
			return
		}
		defer uninit()
		client, event, err := openClient(opts, capture)
		if err != nil {
			started <- err
			return
		}
		defer client.release()
		defer windows.CloseHandle(event)
		var svc comObj
		if err := hr(client.call(14, uintptr(unsafe.Pointer(service)), uintptr(unsafe.Pointer(&svc))), "getting the service"); err != nil {
			started <- err
			return
		}
		defer svc.release()
		if !capture {
			pump(client, svc) // prime the buffer before starting
		}
		if err := hr(client.call(10), "starting"); err != nil {
			started <- err
			return
		}
		defer client.call(11)
		started <- nil
		for {
			select {
			case <-s.stop:
				return
			default:
			}
			r, _ := windows.WaitForSingleObject(event, 200)
			if r == uint32(windows.WAIT_TIMEOUT) {
				continue
			}
			if err := pump(client, svc); err != nil {
				return // the device went away
			}
		}
	}()
	if err := <-started; err != nil {
		<-s.done
		return nil, err
	}
	return s, nil
}

type platformInput struct{ s *stream }

func (in *Input) open(ctx context.Context, _ *giode.App) error {
	ch := in.opts.Channels
	s, err := run(in.opts, true, &iidIAudioCaptureClient, func(_, cap comObj) error {
		for {
			var frames uint32
			if err := hr(cap.call(5, uintptr(unsafe.Pointer(&frames))), "reading"); err != nil {
				return err
			}
			if frames == 0 {
				return nil
			}
			var data *int16
			var flags uint32
			if err := hr(cap.call(3, uintptr(unsafe.Pointer(&data)), uintptr(unsafe.Pointer(&frames)), uintptr(unsafe.Pointer(&flags)), 0, 0), "reading"); err != nil {
				return err
			}
			n := int(frames) * ch
			buf := make([]int16, n)
			if flags&bufferFlagSilent == 0 && data != nil {
				copy(buf, unsafe.Slice(data, n))
			}
			cap.call(4, uintptr(frames))
			in.deliver(buf)
		}
	})
	if err != nil {
		return err
	}
	in.plat = platformInput{s}
	return nil
}

func (p platformInput) close() error { return p.s.close() }

type platformOutput struct{ s *stream }

func (o *Output) open(ctx context.Context, _ *giode.App) error {
	ch := o.opts.Channels
	var size uint32
	s, err := run(o.opts, false, &iidIAudioRenderClient, func(client, render comObj) error {
		if size == 0 {
			if err := hr(client.call(4, uintptr(unsafe.Pointer(&size))), "sizing"); err != nil {
				return err
			}
		}
		var padding uint32
		if err := hr(client.call(6, uintptr(unsafe.Pointer(&padding))), "writing"); err != nil {
			return err
		}
		free := size - padding
		if free == 0 {
			return nil
		}
		var data *int16
		if err := hr(render.call(3, uintptr(free), uintptr(unsafe.Pointer(&data))), "writing"); err != nil {
			return err
		}
		o.pull(unsafe.Slice(data, int(free)*ch))
		render.call(4, uintptr(free), 0)
		return nil
	})
	if err != nil {
		return err
	}
	o.plat = platformOutput{s}
	return nil
}

func (p platformOutput) close() error { return p.s.close() }

var _ = errors.New
