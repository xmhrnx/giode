package media

import (
	"context"
	"errors"
	"fmt"
	"image"
	"math"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows plays through Media Foundation's media engine in
// frame-server mode: the engine decodes, plays the sound and keeps
// time; each new video frame is copied out through a WIC bitmap. The
// engine's time getters return doubles, which pure-Go calls can't read,
// so the engine keeps its own clock, set by the engine's events and by
// each frame's timestamp.

var (
	mfplat         = windows.NewLazySystemDLL("mfplat.dll")
	oleaut32       = windows.NewLazySystemDLL("oleaut32.dll")
	procMFStartup  = mfplat.NewProc("MFStartup")
	procMFAttrs    = mfplat.NewProc("MFCreateAttributes")
	procSysAlloc   = oleaut32.NewProc("SysAllocString")
	procSysFree    = oleaut32.NewProc("SysFreeString")
	mfStartOnce    sync.Once
	mfStartErr     error
	notifyVtblOnce sync.Once
	notifyVtbl     [4]uintptr
	notifiers      sync.Map // *notifyObj → *mfEngine
)

func mkguid(d1 uint32, d2, d3 uint16, d4 ...byte) windows.GUID {
	g := windows.GUID{Data1: d1, Data2: d2, Data3: d3}
	copy(g.Data4[:], d4)
	return g
}

var (
	clsidEngineFactory = mkguid(0xb44392da, 0x499b, 0x446b, 0xa4, 0xcb, 0x00, 0x5f, 0xea, 0xd0, 0xe6, 0xd5)
	iidEngineFactory   = mkguid(0x4d645ace, 0x26aa, 0x4688, 0x9b, 0xe1, 0xdf, 0x35, 0x16, 0x99, 0x0b, 0x93)
	iidEngineEx        = mkguid(0x83015ead, 0xb1e6, 0x40d0, 0xa9, 0x8a, 0x37, 0x14, 0x5f, 0xfe, 0x1a, 0xd1)
	iidEngineNotify    = mkguid(0xfee7c112, 0xe776, 0x42b5, 0x9b, 0xbf, 0x00, 0x48, 0x52, 0x4e, 0x2b, 0xd5)
	iidUnknown         = mkguid(0x00000000, 0x0000, 0x0000, 0xc0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46)
	attrCallback       = mkguid(0xc60381b8, 0x83a4, 0x41f8, 0xa3, 0xd0, 0xde, 0x05, 0x07, 0x68, 0x49, 0xa9)
	attrOutputFormat   = mkguid(0x5066893c, 0x8cf9, 0x42bc, 0x8b, 0x8a, 0x47, 0x22, 0x12, 0xe5, 0x27, 0x26)
	pdDuration         = mkguid(0x6c990d33, 0xbb8e, 0x477a, 0x85, 0x98, 0x0d, 0x5d, 0x96, 0xfc, 0xd8, 0x8a)
	clsidWICFactory    = mkguid(0xcacaf262, 0x9370, 0x4615, 0xa1, 0x3b, 0x9f, 0x55, 0x39, 0xda, 0x4c, 0x0a)
	iidWICFactory      = mkguid(0xec5ec8a9, 0xc395, 0x4314, 0x9c, 0x77, 0x54, 0xd7, 0xa9, 0x35, 0xff, 0x70)
	wicFormatBGRA      = mkguid(0x6fddc324, 0x4e03, 0x4bfe, 0xb1, 0x85, 0x3d, 0x77, 0x76, 0x8d, 0xc9, 0x0f)
)

// Vtable indices.
const (
	mQueryInterface = 0
	mRelease        = 2
	aSetUINT32      = 21
	aSetUnknown     = 27
	fCreateInstance = 3

	eGetError           = 3
	eSetSource          = 6
	eSetPreload         = 10
	eSetCurrentTime     = 17
	eSetDefaultRate     = 22
	eSetPlaybackRate    = 24
	ePlay               = 32
	ePause              = 33
	eSetVolume          = 37
	eHasVideo           = 38
	eGetNativeVideoSize = 40
	eShutdown           = 42
	eTransferVideoFrame = 43
	eOnVideoStreamTick  = 44
	exGetPresentationAt = 53
	errGetErrorCode     = 3
	errGetExtendedCode  = 4

	wicCreateBitmap = 17
	wicCopyPixels   = 7

	evError          = 5
	evPause          = 9
	evLoadedMetadata = 10
	evWaiting        = 12
	evPlaying        = 13
	evSeeking        = 16
	evSeeked         = 17
	evEnded          = 19
	evDurationChange = 21

	dxgiFormatB8G8R8A8 = 87
	preloadAutomatic   = 4
)

type comObj struct{ vtbl *[64]uintptr }

func (o *comObj) call(m int, args ...uintptr) uintptr {
	r, _, _ := syscall.SyscallN(o.vtbl[m], append([]uintptr{uintptr(unsafe.Pointer(o))}, args...)...)
	return r
}

func (o *comObj) release() {
	if o != nil {
		o.call(mRelease)
	}
}

// floatArg passes a double: in one register on 64-bit Windows (where
// Go also loads it into the matching XMM register), in two stack slots
// on 32-bit.
func floatArg(v float64) []uintptr {
	b := math.Float64bits(v)
	if unsafe.Sizeof(uintptr(0)) == 8 {
		return []uintptr{uintptr(b)}
	}
	return []uintptr{uintptr(uint32(b)), uintptr(uint32(b >> 32))}
}

func failed(r uintptr) bool { return int32(r) < 0 }

func hr(r uintptr, what string) error {
	if failed(r) {
		return fmt.Errorf("media: %s: 0x%08x", what, uint32(r))
	}
	return nil
}

func startMF() error {
	mfStartOnce.Do(func() {
		if err := mfplat.Load(); err != nil {
			mfStartErr = fmt.Errorf("%w: Media Foundation isn't installed (Windows N needs the Media Feature Pack)", ErrUnsupported)
			return
		}
		r, _, _ := procMFStartup.Call(0x00020070, 0)
		mfStartErr = hr(r, "starting Media Foundation")
	})
	return mfStartErr
}

// notifyObj is the IMFMediaEngineNotify the engine calls back.
type notifyObj struct {
	vtbl *[4]uintptr
}

func notifyQueryInterface(this *notifyObj, riid *windows.GUID, ppv **notifyObj) uintptr {
	if *riid == iidUnknown || *riid == iidEngineNotify {
		*ppv = this
		return 0
	}
	*ppv = nil
	return 0x80004002 // E_NOINTERFACE
}

func notifyAddRef(this *notifyObj) uintptr  { return 1 }
func notifyRelease(this *notifyObj) uintptr { return 1 }

func notifyEvent(this *notifyObj, event, param1, param2 uintptr) uintptr {
	if v, ok := notifiers.Load(this); ok {
		e := v.(*mfEngine)
		select {
		case e.events <- engineEvent{uint32(event), uint32(param1), uint32(param2)}:
		default:
		}
	}
	return 0
}

type engineEvent struct{ kind, p1, p2 uint32 }

type mfEngine struct {
	p      *Player
	cmds   chan func()
	events chan engineEvent
	done   chan struct{}
	exited chan struct{}
	notify *notifyObj

	// COM thread only.
	eng, engEx, wic, bitmap *comObj
	bw, bh                  int
	pix                     []byte
	readySent               bool

	mu        sync.Mutex
	base      time.Duration
	anchor    time.Time
	running   bool
	rate      float64
	dur       time.Duration
	seekingTo time.Duration
}

func fileURL(src string) (string, error) {
	if strings.Contains(src, "://") {
		return src, nil
	}
	abs, err := filepath.Abs(src)
	if err != nil {
		return "", err
	}
	return (&url.URL{Scheme: "file", Path: "/" + filepath.ToSlash(abs)}).String(), nil
}

func newEngine(ctx context.Context, p *Player, src string) (engine, error) {
	u, err := fileURL(src)
	if err != nil {
		return nil, err
	}
	notifyVtblOnce.Do(func() {
		notifyVtbl = [4]uintptr{
			syscall.NewCallback(notifyQueryInterface),
			syscall.NewCallback(notifyAddRef),
			syscall.NewCallback(notifyRelease),
			syscall.NewCallback(notifyEvent),
		}
	})
	e := &mfEngine{
		p: p, rate: 1,
		cmds:   make(chan func(), 64),
		events: make(chan engineEvent, 64),
		done:   make(chan struct{}),
		exited: make(chan struct{}),
		notify: &notifyObj{vtbl: &notifyVtbl},
	}
	notifiers.Store(e.notify, e)
	started := make(chan error, 1)
	go e.run(u, started)
	if err := <-started; err != nil {
		notifiers.Delete(e.notify)
		return nil, err
	}
	return e, nil
}

// run owns the engine on one COM thread.
func (e *mfEngine) run(src string, started chan<- error) {
	defer close(e.exited)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	switch err := windows.CoInitializeEx(0, windows.COINIT_MULTITHREADED); err {
	case nil, syscall.Errno(1):
		defer windows.CoUninitialize()
	case syscall.Errno(0x80010106):
	default:
		started <- err
		return
	}
	if err := e.create(src); err != nil {
		e.destroy()
		started <- err
		return
	}
	started <- nil
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-e.done:
			e.destroy()
			return
		case fn := <-e.cmds:
			fn()
		case ev := <-e.events:
			e.handle(ev)
		case <-tick.C:
			e.pollFrame()
		}
	}
}

func (e *mfEngine) create(src string) error {
	if err := startMF(); err != nil {
		return err
	}
	var factory *comObj
	if err := hr(uintptr(coCreate(&clsidEngineFactory, &iidEngineFactory, unsafe.Pointer(&factory))), "creating the media engine factory"); err != nil {
		return fmt.Errorf("%w (%v)", ErrUnsupported, err)
	}
	defer factory.release()
	var attrs *comObj
	if r, _, _ := procMFAttrs.Call(uintptr(unsafe.Pointer(&attrs)), 2); failed(r) {
		return hr(r, "creating attributes")
	}
	defer attrs.release()
	attrs.call(aSetUnknown, uintptr(unsafe.Pointer(&attrCallback)), uintptr(unsafe.Pointer(e.notify)))
	attrs.call(aSetUINT32, uintptr(unsafe.Pointer(&attrOutputFormat)), dxgiFormatB8G8R8A8)
	var flags uintptr
	if silentForTests {
		flags = 0x4 // MF_MEDIA_ENGINE_FORCEMUTE
	}
	if err := hr(factory.call(fCreateInstance, flags, uintptr(unsafe.Pointer(attrs)), uintptr(unsafe.Pointer(&e.eng))), "creating the media engine"); err != nil {
		return err
	}
	e.eng.call(mQueryInterface, uintptr(unsafe.Pointer(&iidEngineEx)), uintptr(unsafe.Pointer(&e.engEx)))
	if err := hr(uintptr(coCreate(&clsidWICFactory, &iidWICFactory, unsafe.Pointer(&e.wic))), "creating the imaging factory"); err != nil {
		return err
	}
	e.eng.call(eSetPreload, preloadAutomatic)
	bstr, _, _ := procSysAlloc.Call(uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(src))))
	defer procSysFree.Call(bstr)
	return hr(e.eng.call(eSetSource, bstr), "opening the media")
}

func coCreate(clsid, iid *windows.GUID, out unsafe.Pointer) int32 {
	// CLSCTX_INPROC_SERVER
	r, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(clsid)), 0, 1, uintptr(unsafe.Pointer(iid)), uintptr(out))
	return int32(r)
}

var procCoCreateInstance = windows.NewLazySystemDLL("ole32.dll").NewProc("CoCreateInstance")

func (e *mfEngine) destroy() {
	if e.eng != nil {
		e.eng.call(eShutdown)
	}
	e.bitmap.release()
	e.wic.release()
	e.engEx.release()
	e.eng.release()
	e.bitmap, e.wic, e.engEx, e.eng = nil, nil, nil, nil
	notifiers.Delete(e.notify)
}

func (e *mfEngine) handle(ev engineEvent) {
	switch ev.kind {
	case evLoadedMetadata, evDurationChange:
		e.mu.Lock()
		e.dur = e.readDuration()
		dur := e.dur
		e.mu.Unlock()
		var w, h uint32
		if e.eng.call(eHasVideo) != 0 {
			e.eng.call(eGetNativeVideoSize, uintptr(unsafe.Pointer(&w)), uintptr(unsafe.Pointer(&h)))
		}
		if !e.readySent {
			e.readySent = true
			go e.p.ready(dur, image.Pt(int(w), int(h)))
		} else {
			go e.p.setDuration(dur)
		}
	case evPlaying:
		e.mu.Lock()
		if !e.running {
			e.running, e.anchor = true, time.Now()
		}
		e.mu.Unlock()
	case evPause, evWaiting:
		e.mu.Lock()
		e.base, e.running = e.clockLocked(), false
		e.mu.Unlock()
	case evSeeking:
	case evSeeked:
		e.mu.Lock()
		e.base, e.anchor = e.seekingTo, time.Now()
		e.mu.Unlock()
	case evEnded:
		e.mu.Lock()
		e.base, e.running = e.dur, false
		e.mu.Unlock()
		go e.p.ended()
	case evError:
		go e.p.failed(engineError(ev.p1, ev.p2))
	}
}

func engineError(code, hres uint32) error {
	msg := map[uint32]string{
		1: "loading was aborted",
		2: "a network error",
		3: "the media can't be decoded",
		4: "the format isn't supported",
		5: "the media is encrypted",
	}[code]
	if msg == "" {
		msg = "playback error"
	}
	if hres != 0 {
		return fmt.Errorf("media: %s (0x%08x)", msg, hres)
	}
	return errors.New("media: " + msg)
}

// readDuration asks IMFMediaEngineEx for the presentation's duration.
func (e *mfEngine) readDuration() time.Duration {
	if e.engEx == nil {
		return 0
	}
	// A PROPVARIANT: the type, then the value at offset 8.
	var pv [4]uint64
	if failed(e.engEx.call(exGetPresentationAt, uintptr(unsafe.Pointer(&pdDuration)), uintptr(unsafe.Pointer(&pv[0])))) {
		return 0
	}
	const vtUI8 = 21
	if uint16(pv[0]) != vtUI8 {
		return 0
	}
	return time.Duration(pv[1]) * 100
}

// pollFrame copies out a new video frame, if there is one.
func (e *mfEngine) pollFrame() {
	if e.eng == nil {
		return
	}
	var pts int64
	if e.eng.call(eOnVideoStreamTick, uintptr(unsafe.Pointer(&pts))) != 0 {
		return // S_FALSE: no new frame, or an error
	}
	var w, h uint32
	if failed(e.eng.call(eGetNativeVideoSize, uintptr(unsafe.Pointer(&w)), uintptr(unsafe.Pointer(&h)))) || w == 0 || h == 0 {
		return
	}
	if e.bitmap == nil || e.bw != int(w) || e.bh != int(h) {
		e.bitmap.release()
		e.bitmap = nil
		const cacheOnLoad = 2
		if failed(e.wic.call(wicCreateBitmap, uintptr(w), uintptr(h), uintptr(unsafe.Pointer(&wicFormatBGRA)), cacheOnLoad, uintptr(unsafe.Pointer(&e.bitmap)))) {
			return
		}
		e.bw, e.bh = int(w), int(h)
		e.pix = make([]byte, e.bw*e.bh*4)
		go e.p.setVideoSize(image.Pt(e.bw, e.bh))
	}
	rect := [4]int32{0, 0, int32(w), int32(h)}
	black := [4]byte{0, 0, 0, 255}
	if failed(e.eng.call(eTransferVideoFrame, uintptr(unsafe.Pointer(e.bitmap)), 0, uintptr(unsafe.Pointer(&rect)), uintptr(unsafe.Pointer(&black)))) {
		return
	}
	stride := e.bw * 4
	if failed(e.bitmap.call(wicCopyPixels, 0, uintptr(stride), uintptr(len(e.pix)), uintptr(unsafe.Pointer(&e.pix[0])))) {
		return
	}
	for i := 0; i < len(e.pix); i += 4 {
		e.pix[i], e.pix[i+2], e.pix[i+3] = e.pix[i+2], e.pix[i], 255
	}
	e.p.surface.Push(&image.RGBA{Pix: e.pix, Stride: stride, Rect: image.Rect(0, 0, e.bw, e.bh)})
	if pts >= 0 {
		e.mu.Lock()
		if e.running {
			e.base, e.anchor = time.Duration(pts)*100, time.Now()
		}
		e.mu.Unlock()
	}
}

// do runs fn on the COM thread, without waiting.
func (e *mfEngine) do(fn func()) {
	select {
	case e.cmds <- fn:
	case <-e.done:
	}
}

func (e *mfEngine) play() {
	e.do(func() { e.eng.call(ePlay) })
}

func (e *mfEngine) pause() {
	e.mu.Lock()
	e.base, e.running = e.clockLocked(), false
	e.mu.Unlock()
	e.do(func() { e.eng.call(ePause) })
}

func (e *mfEngine) seek(d time.Duration) {
	e.mu.Lock()
	e.seekingTo = d
	e.base, e.anchor = d, time.Now()
	e.mu.Unlock()
	e.do(func() { e.eng.call(eSetCurrentTime, floatArg(d.Seconds())...) })
}

func (e *mfEngine) setRate(r float64) {
	e.mu.Lock()
	e.base, e.anchor = e.clockLocked(), time.Now()
	e.rate = r
	e.mu.Unlock()
	e.do(func() {
		e.eng.call(eSetDefaultRate, floatArg(r)...)
		e.eng.call(eSetPlaybackRate, floatArg(r)...)
	})
}

func (e *mfEngine) setVolume(v float64) {
	e.do(func() { e.eng.call(eSetVolume, floatArg(v)...) })
}

func (e *mfEngine) position() time.Duration {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.clockLocked()
}

func (e *mfEngine) clockLocked() time.Duration {
	d := e.base
	if e.running {
		d += time.Duration(float64(time.Since(e.anchor)) * e.rate)
	}
	if e.dur > 0 {
		d = min(d, e.dur)
	}
	return d
}

func (e *mfEngine) close() {
	close(e.done)
	<-e.exited
}

// silentForTests mutes the engine, for machines without audio.
var silentForTests bool
