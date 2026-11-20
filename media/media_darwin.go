//go:build darwin

package media

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework Foundation -framework AVFoundation -framework CoreMedia -framework CoreVideo

#include <stdlib.h>

char *giodeMediaOpen(int ident, const char *src, int muted);
void giodeMediaPlay(int ident);
void giodeMediaPause(int ident);
void giodeMediaSeek(int ident, double secs);
void giodeMediaRate(int ident, double rate);
void giodeMediaVolume(int ident, double v);
double giodeMediaPosition(int ident);
void giodeMediaClose(int ident);
*/
import "C"

import (
	"context"
	"errors"
	"image"
	"sync"
	"time"
	"unsafe"
)

var avPlayers struct {
	sync.Mutex
	next int
	m    map[int]*Player
}

func avLookup(id C.int) *Player {
	avPlayers.Lock()
	defer avPlayers.Unlock()
	return avPlayers.m[int(id)]
}

type avEngine struct{ id C.int }

func newEngine(ctx context.Context, p *Player, src string) (engine, error) {
	avPlayers.Lock()
	if avPlayers.m == nil {
		avPlayers.m = map[int]*Player{}
	}
	avPlayers.next++
	id := avPlayers.next
	avPlayers.m[id] = p
	avPlayers.Unlock()
	cs := C.CString(src)
	defer C.free(unsafe.Pointer(cs))
	muted := 0
	if p.opts.Muted {
		muted = 1
	}
	if msg := C.giodeMediaOpen(C.int(id), cs, C.int(muted)); msg != nil {
		err := errors.New("media: " + C.GoString(msg))
		C.free(unsafe.Pointer(msg))
		avPlayers.Lock()
		delete(avPlayers.m, id)
		avPlayers.Unlock()
		return nil, err
	}
	return avEngine{C.int(id)}, nil
}

func (e avEngine) play()                { C.giodeMediaPlay(e.id) }
func (e avEngine) pause()               { C.giodeMediaPause(e.id) }
func (e avEngine) seek(d time.Duration) { C.giodeMediaSeek(e.id, C.double(d.Seconds())) }
func (e avEngine) setRate(r float64)    { C.giodeMediaRate(e.id, C.double(r)) }
func (e avEngine) setVolume(v float64)  { C.giodeMediaVolume(e.id, C.double(v)) }
func (e avEngine) position() time.Duration {
	return time.Duration(float64(C.giodeMediaPosition(e.id)) * float64(time.Second))
}

func (e avEngine) close() {
	C.giodeMediaClose(e.id)
	avPlayers.Lock()
	delete(avPlayers.m, int(e.id))
	avPlayers.Unlock()
}

//export giodeMediaReady
func giodeMediaReady(id C.int, secs C.double, w, h C.int) {
	if p := avLookup(id); p != nil {
		go p.ready(time.Duration(float64(secs)*float64(time.Second)), image.Pt(int(w), int(h)))
	}
}

//export giodeMediaFrame
func giodeMediaFrame(id C.int, data unsafe.Pointer, w, h, stride C.int) {
	p := avLookup(id)
	if p == nil || data == nil {
		return
	}
	pix := unsafe.Slice((*byte)(data), int(stride)*int(h))
	p.surface.PushBGRA(pix, int(w), int(h), int(stride))
	if size := image.Pt(int(w), int(h)); p.VideoSize() != size {
		p.setVideoSize(size)
	}
}

//export giodeMediaEnded
func giodeMediaEnded(id C.int) {
	if p := avLookup(id); p != nil {
		go p.ended()
	}
}

//export giodeMediaFailed
func giodeMediaFailed(id C.int, msg *C.char) {
	if p := avLookup(id); p != nil {
		err := errors.New("media: " + C.GoString(msg))
		go p.failed(err)
	}
}

// silentForTests is unused here: AVPlayer plays without an audio
// device.
var silentForTests bool
