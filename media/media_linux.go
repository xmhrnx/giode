//go:build linux && !android

package media

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

// The few GStreamer and GLib calls the player makes, loaded at run time
// so building needs no GStreamer headers and running without GStreamer
// fails with an error instead of not starting.

typedef struct {
	void *memory;
	int flags;
	uint8_t *data;
	size_t size;
	size_t maxsize;
	void *user_data[4];
	void *reserved[4];
} giode_map_info;

typedef struct {
	uint32_t domain;
	int code;
	char *message;
} giode_gerror;

static int (*p_init_check)(int *, char ***, void **);
static void *(*p_factory_make)(const char *, const char *);
static void (*p_object_set)(void *, const char *, ...);
static int (*p_set_state)(void *, int);
static int (*p_query_position)(void *, int, int64_t *);
static int (*p_query_duration)(void *, int, int64_t *);
static int (*p_seek)(void *, double, int, int, int, int64_t, int, int64_t);
static void *(*p_get_bus)(void *);
static void *(*p_bus_pop)(void *, uint64_t, int);
static void (*p_parse_error)(void *, giode_gerror **, char **);
static void (*p_error_free)(giode_gerror *);
static void (*p_free)(void *);
static void (*p_unref_mini)(void *);
static void (*p_object_unref)(void *);
static void *(*p_caps_from_string)(const char *);
static void *(*p_caps_get_structure)(void *, unsigned);
static int (*p_structure_get_int)(void *, const char *, int *);
static void *(*p_sample_get_buffer)(void *);
static void *(*p_sample_get_caps)(void *);
static int (*p_buffer_map)(void *, giode_map_info *, int);
static void (*p_buffer_unmap)(void *, giode_map_info *);
static void *(*p_pull_sample)(void *, uint64_t);
static void *(*p_pull_preroll)(void *, uint64_t);
static void *(*p_message_structure)(void *);

static void *giode_sym(void *lib, const char *name, int *missing) {
	void *p = dlsym(lib, name);
	if (!p) (*missing)++;
	return p;
}

static const char *giode_gst_load(void) {
	static int loaded = 0;
	static const char *err = NULL;
	if (loaded) return err;
	loaded = 1;
	void *gst = dlopen("libgstreamer-1.0.so.0", RTLD_NOW | RTLD_GLOBAL);
	void *app = dlopen("libgstapp-1.0.so.0", RTLD_NOW | RTLD_GLOBAL);
	void *gobj = dlopen("libgobject-2.0.so.0", RTLD_NOW | RTLD_GLOBAL);
	void *glib = dlopen("libglib-2.0.so.0", RTLD_NOW | RTLD_GLOBAL);
	if (!gst || !app || !gobj || !glib) {
		err = "GStreamer isn't installed (libgstreamer1.0-0 and gstreamer1.0-plugins-base)";
		return err;
	}
	int m = 0;
	p_init_check = giode_sym(gst, "gst_init_check", &m);
	p_factory_make = giode_sym(gst, "gst_element_factory_make", &m);
	p_object_set = giode_sym(gobj, "g_object_set", &m);
	p_set_state = giode_sym(gst, "gst_element_set_state", &m);
	p_query_position = giode_sym(gst, "gst_element_query_position", &m);
	p_query_duration = giode_sym(gst, "gst_element_query_duration", &m);
	p_seek = giode_sym(gst, "gst_element_seek", &m);
	p_get_bus = giode_sym(gst, "gst_element_get_bus", &m);
	p_bus_pop = giode_sym(gst, "gst_bus_timed_pop_filtered", &m);
	p_parse_error = giode_sym(gst, "gst_message_parse_error", &m);
	p_error_free = giode_sym(glib, "g_error_free", &m);
	p_free = giode_sym(glib, "g_free", &m);
	p_unref_mini = giode_sym(gst, "gst_mini_object_unref", &m);
	p_object_unref = giode_sym(gst, "gst_object_unref", &m);
	p_caps_from_string = giode_sym(gst, "gst_caps_from_string", &m);
	p_caps_get_structure = giode_sym(gst, "gst_caps_get_structure", &m);
	p_structure_get_int = giode_sym(gst, "gst_structure_get_int", &m);
	p_sample_get_buffer = giode_sym(gst, "gst_sample_get_buffer", &m);
	p_sample_get_caps = giode_sym(gst, "gst_sample_get_caps", &m);
	p_buffer_map = giode_sym(gst, "gst_buffer_map", &m);
	p_buffer_unmap = giode_sym(gst, "gst_buffer_unmap", &m);
	p_pull_sample = giode_sym(app, "gst_app_sink_try_pull_sample", &m);
	p_pull_preroll = giode_sym(app, "gst_app_sink_try_pull_preroll", &m);
	p_message_structure = giode_sym(gst, "gst_message_get_structure", &m);
	if (m) {
		err = "GStreamer is too old";
		return err;
	}
	if (!p_init_check(NULL, NULL, NULL)) {
		err = "GStreamer failed to start";
	}
	return err;
}

enum { GST_NULL = 1, GST_PAUSED = 3, GST_PLAYING = 4 };
enum { FORMAT_TIME = 3 };
enum { MSG_EOS = 1, MSG_ERROR = 2 };

// giode_gst_new makes a playbin playing uri into an RGBA appsink.
static const char *giode_gst_new(const char *uri, int fakeAudio, void **pipe, void **sink) {
	void *pb = p_factory_make("playbin", NULL);
	void *as = p_factory_make("appsink", NULL);
	if (!pb || !as) {
		return "GStreamer's base plugins aren't installed (gstreamer1.0-plugins-base)";
	}
	void *caps = p_caps_from_string("video/x-raw,format=RGBA,pixel-aspect-ratio=1/1");
	p_object_set(as, "caps", caps, "max-buffers", 2, "drop", 1, "sync", 1, NULL);
	p_unref_mini(caps);
	p_object_set(pb, "uri", uri, "video-sink", as, NULL);
	if (fakeAudio) {
		void *fs = p_factory_make("fakesink", NULL);
		p_object_set(fs, "sync", 1, NULL);
		p_object_set(pb, "audio-sink", fs, NULL);
	}
	// Keeps the pitch of speech at other speeds, where it's installed.
	void *st = p_factory_make("scaletempo", NULL);
	if (st) {
		p_object_set(pb, "audio-filter", st, NULL);
	}
	*pipe = pb;
	*sink = as;
	return NULL;
}

static int giode_gst_state(void *pipe, int state) { return p_set_state(pipe, state); }

static void giode_gst_volume(void *pipe, double v) { p_object_set(pipe, "volume", v, NULL); }

static int64_t giode_gst_position(void *pipe) {
	int64_t v = -1;
	if (!p_query_position(pipe, FORMAT_TIME, &v)) return -1;
	return v;
}

static int64_t giode_gst_duration(void *pipe) {
	int64_t v = -1;
	if (!p_query_duration(pipe, FORMAT_TIME, &v)) return -1;
	return v;
}

// giode_gst_seek moves to pos (ns) at rate.
static int giode_gst_seek(void *pipe, double rate, int64_t pos) {
	// FLUSH | ACCURATE; start SET, stop NONE.
	return p_seek(pipe, rate, FORMAT_TIME, 1 | 2, 1, pos, 0, -1);
}

// giode_gst_poll reads the bus: 1 at the end, 2 with an error (in
// *msg, to free), 0 otherwise.
static int giode_gst_poll(void *pipe, char **msg) {
	void *bus = p_get_bus(pipe);
	int r = 0;
	// Popping drops the messages that don't match, so take both kinds
	// at once; an end-of-stream message has no structure.
	void *m = p_bus_pop(bus, 0, MSG_ERROR | MSG_EOS);
	if (m && p_message_structure(m)) {
		giode_gerror *e = NULL;
		char *dbg = NULL;
		p_parse_error(m, &e, &dbg);
		*msg = strdup(e && e->message ? e->message : "playback error");
		if (e) p_error_free(e);
		if (dbg) p_free(dbg);
		r = 2;
	} else if (m) {
		r = 1;
	}
	if (m) p_unref_mini(m);
	p_object_unref(bus);
	return r;
}

// giode_gst_pull waits up to timeout (ns) for a frame. It returns the
// sample, to release with giode_gst_release, and its pixels.
static void *giode_gst_pull(void *sink, int preroll, uint64_t timeout, giode_map_info *info, void **buf, int *w, int *h) {
	void *s = preroll ? p_pull_preroll(sink, timeout) : p_pull_sample(sink, timeout);
	if (!s) return NULL;
	void *caps = p_sample_get_caps(s);
	void *b = p_sample_get_buffer(s);
	if (!caps || !b) {
		p_unref_mini(s);
		return NULL;
	}
	void *st = p_caps_get_structure(caps, 0);
	if (!p_structure_get_int(st, "width", w) || !p_structure_get_int(st, "height", h) || !p_buffer_map(b, info, 1)) {
		p_unref_mini(s);
		return NULL;
	}
	*buf = b;
	return s;
}

static void giode_gst_release(void *s, void *buf, giode_map_info *info) {
	p_buffer_unmap(buf, info);
	p_unref_mini(s);
}

static void giode_gst_free(void *pipe) {
	p_set_state(pipe, GST_NULL);
	p_object_unref(pipe);
}
*/
import "C"

import (
	"context"
	"errors"
	"image"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"
)

type gstEngine struct {
	p    *Player
	pipe unsafe.Pointer
	sink unsafe.Pointer

	mu      sync.Mutex
	playing bool
	rate    float64
	done    chan struct{}
	wg      sync.WaitGroup
}

func sourceURI(src string) (string, error) {
	if strings.Contains(src, "://") {
		return src, nil
	}
	abs, err := filepath.Abs(src)
	if err != nil {
		return "", err
	}
	return (&url.URL{Scheme: "file", Path: abs}).String(), nil
}

func newEngine(ctx context.Context, p *Player, src string) (engine, error) {
	if msg := C.giode_gst_load(); msg != nil {
		return nil, errors.New("media: " + C.GoString(msg))
	}
	uri, err := sourceURI(src)
	if err != nil {
		return nil, err
	}
	curi := C.CString(uri)
	defer C.free(unsafe.Pointer(curi))
	e := &gstEngine{p: p, rate: 1, done: make(chan struct{})}
	if msg := C.giode_gst_new(curi, C.int(btoi(silentForTests)), &e.pipe, &e.sink); msg != nil {
		return nil, errors.New("media: " + C.GoString(msg))
	}
	// Paused prerolls: the first frame shows and the duration's known.
	C.giode_gst_state(e.pipe, C.GST_PAUSED)
	e.wg.Add(1)
	go e.run()
	return e, nil
}

// run pulls frames and watches the bus until closed.
func (e *gstEngine) run() {
	defer e.wg.Done()
	var (
		ready    bool
		lastBuf  unsafe.Pointer
		info     C.giode_map_info
		buf      unsafe.Pointer
		w, h     C.int
		lastSize image.Point
	)
	start := time.Now()
	for {
		select {
		case <-e.done:
			return
		default:
		}
		e.mu.Lock()
		playing := e.playing
		e.mu.Unlock()
		preroll := C.int(0)
		if !playing {
			preroll = 1
		}
		if s := C.giode_gst_pull(e.sink, preroll, C.uint64_t(50*time.Millisecond), &info, &buf, &w, &h); s != nil {
			// Paused, the preroll frame comes back until it changes.
			if buf != lastBuf || playing {
				lastBuf = buf
				size := image.Pt(int(w), int(h))
				pix := unsafe.Slice((*byte)(unsafe.Pointer(info.data)), int(info.size))
				if len(pix) >= size.X*size.Y*4 {
					e.p.surface.Push(&image.RGBA{Pix: pix, Stride: size.X * 4, Rect: image.Rectangle{Max: size}})
				}
				if size != lastSize {
					lastSize = size
					e.p.setVideoSize(size)
				}
			}
			C.giode_gst_release(s, buf, &info)
		} else if !playing {
			time.Sleep(20 * time.Millisecond)
		}
		if !ready {
			if d := C.giode_gst_duration(e.pipe); d >= 0 || time.Since(start) > 5*time.Second {
				ready = true
				e.p.ready(time.Duration(max(int64(d), 0)), lastSize)
			}
		}
		var msg *C.char
		switch C.giode_gst_poll(e.pipe, &msg) {
		case 1:
			e.mu.Lock()
			e.playing = false
			e.mu.Unlock()
			e.p.ended()
		case 2:
			err := errors.New("media: " + C.GoString(msg))
			C.free(unsafe.Pointer(msg))
			e.p.failed(err)
		}
	}
}

func (e *gstEngine) play() {
	e.mu.Lock()
	e.playing = true
	e.mu.Unlock()
	C.giode_gst_state(e.pipe, C.GST_PLAYING)
}

func (e *gstEngine) pause() {
	e.mu.Lock()
	e.playing = false
	e.mu.Unlock()
	C.giode_gst_state(e.pipe, C.GST_PAUSED)
}

func (e *gstEngine) seek(d time.Duration) {
	e.mu.Lock()
	r := e.rate
	e.mu.Unlock()
	C.giode_gst_seek(e.pipe, C.double(r), C.int64_t(d))
}

func (e *gstEngine) setRate(r float64) {
	e.mu.Lock()
	e.rate = r
	e.mu.Unlock()
	if pos := C.giode_gst_position(e.pipe); pos >= 0 {
		C.giode_gst_seek(e.pipe, C.double(r), pos)
	}
}

func (e *gstEngine) setVolume(v float64) { C.giode_gst_volume(e.pipe, C.double(v)) }

func (e *gstEngine) position() time.Duration {
	return time.Duration(C.giode_gst_position(e.pipe))
}

func (e *gstEngine) close() {
	close(e.done)
	e.wg.Wait()
	C.giode_gst_free(e.pipe)
}

// silentForTests plays sound into nothing, for machines without audio.
var silentForTests bool

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
