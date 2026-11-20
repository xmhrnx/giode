//go:build linux && !android

package ocr

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>

typedef struct TessBaseAPI TessBaseAPI;
typedef struct TessResultIterator TessResultIterator;
typedef struct TessPageIterator TessPageIterator;

static TessBaseAPI *(*t_create)(void);
static int (*t_init3)(TessBaseAPI *, const char *, const char *);
static void (*t_set_image)(TessBaseAPI *, const unsigned char *, int, int, int, int);
static int (*t_recognize)(TessBaseAPI *, void *);
static TessResultIterator *(*t_iterator)(TessBaseAPI *);
static const TessPageIterator *(*t_page_iter)(const TessResultIterator *);
static int (*t_box)(const TessPageIterator *, int, int *, int *, int *, int *);
static int (*t_at_beginning)(const TessPageIterator *, int);
static char *(*t_text)(const TessResultIterator *, int);
static float (*t_confidence)(const TessResultIterator *, int);
static int (*t_next)(TessResultIterator *, int);
static void (*t_delete_text)(const char *);
static void (*t_iter_delete)(TessResultIterator *);
static void (*t_end)(TessBaseAPI *);
static void (*t_delete)(TessBaseAPI *);

// giode_tess_load loads libtesseract at run time: 1 loaded, -1 missing.
static int giode_tess_load(void) {
	static int state = 0;
	if (state) return state;
	const char *names[] = {"libtesseract.so.5", "libtesseract.so.4", "libtesseract.so", NULL};
	void *lib = NULL;
	for (int i = 0; names[i] && !lib; i++) lib = dlopen(names[i], RTLD_NOW);
	if (!lib) { state = -1; return state; }
	t_create = dlsym(lib, "TessBaseAPICreate");
	t_init3 = dlsym(lib, "TessBaseAPIInit3");
	t_set_image = dlsym(lib, "TessBaseAPISetImage");
	t_recognize = dlsym(lib, "TessBaseAPIRecognize");
	t_iterator = dlsym(lib, "TessBaseAPIGetIterator");
	t_page_iter = dlsym(lib, "TessResultIteratorGetPageIteratorConst");
	t_box = dlsym(lib, "TessPageIteratorBoundingBox");
	t_at_beginning = dlsym(lib, "TessPageIteratorIsAtBeginningOf");
	t_text = dlsym(lib, "TessResultIteratorGetUTF8Text");
	t_confidence = dlsym(lib, "TessResultIteratorConfidence");
	t_next = dlsym(lib, "TessResultIteratorNext");
	t_delete_text = dlsym(lib, "TessDeleteText");
	t_iter_delete = dlsym(lib, "TessResultIteratorDelete");
	t_end = dlsym(lib, "TessBaseAPIEnd");
	t_delete = dlsym(lib, "TessBaseAPIDelete");
	state = t_create && t_init3 && t_set_image && t_recognize && t_iterator && t_page_iter && t_box &&
		t_at_beginning && t_text && t_confidence && t_next && t_delete_text && t_iter_delete && t_end && t_delete ? 1 : -1;
	return state;
}

static TessBaseAPI *giode_tess_create(const char *lang) {
	TessBaseAPI *api = t_create();
	if (t_init3(api, NULL, lang) != 0) {
		t_delete(api);
		return NULL;
	}
	return api;
}
static void giode_tess_free(TessBaseAPI *api) { t_end(api); t_delete(api); }
static int giode_tess_run(TessBaseAPI *api, const unsigned char *px, int w, int h) {
	t_set_image(api, px, w, h, 4, w * 4);
	return t_recognize(api, NULL);
}
static TessResultIterator *giode_tess_iter(TessBaseAPI *api) { return t_iterator(api); }
static void giode_tess_iter_free(TessResultIterator *it) { t_iter_delete(it); }
static int giode_tess_box(TessResultIterator *it, int level, int *b) {
	return t_box(t_page_iter(it), level, &b[0], &b[1], &b[2], &b[3]);
}
static int giode_tess_begins(TessResultIterator *it, int level) { return t_at_beginning(t_page_iter(it), level); }
static char *giode_tess_text(TessResultIterator *it, int level) { return t_text(it, level); }
static void giode_tess_free_text(char *s) { t_delete_text(s); }
static float giode_tess_conf(TessResultIterator *it, int level) { return t_confidence(it, level); }
static int giode_tess_next(TessResultIterator *it, int level) { return t_next(it, level); }
*/
import "C"

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
	"sync"
	"unsafe"
)

// Linux reads text with Tesseract, the open-source OCR engine, loaded at
// run time with the languages the user installed (tesseract-ocr-eng,
// tesseract-ocr-deu...).

var supported = C.giode_tess_load() == 1

// errNoTesseract says what to install.
var errNoTesseract = errors.New("ocr: Tesseract isn't installed (the tesseract-ocr package)")

// Tesseract's iterator levels.
const (
	levelLine = 2
	levelWord = 3
)

// tessLangs maps BCP 47 languages to Tesseract's traineddata names.
var tessLangs = map[string]string{
	"en": "eng", "de": "deu", "fr": "fra", "es": "spa", "it": "ita", "pt": "por", "nl": "nld",
	"sv": "swe", "da": "dan", "nb": "nor", "no": "nor", "fi": "fin", "pl": "pol", "cs": "ces",
	"sk": "slk", "hu": "hun", "ro": "ron", "ru": "rus", "uk": "ukr", "bg": "bul", "el": "ell",
	"tr": "tur", "ar": "ara", "fa": "fas", "he": "heb", "hi": "hin", "ja": "jpn", "ko": "kor",
	"zh-hans": "chi_sim", "zh-cn": "chi_sim", "zh": "chi_sim", "zh-hant": "chi_tra", "zh-tw": "chi_tra",
	"vi": "vie", "th": "tha", "id": "ind",
}

func tessLanguage(langs []string) string {
	var out []string
	for _, l := range langs {
		l = strings.ToLower(l)
		code, ok := tessLangs[l]
		if !ok {
			if i := strings.IndexByte(l, '-'); i > 0 {
				code, ok = tessLangs[l[:i]]
			}
		}
		if ok {
			out = append(out, code)
		}
	}
	if len(out) == 0 {
		return "eng"
	}
	return strings.Join(out, "+")
}

// One engine per language set, used one recognition at a time.
var engines = struct {
	sync.Mutex
	api map[string]*C.TessBaseAPI
}{api: map[string]*C.TessBaseAPI{}}

func recognize(ctx context.Context, data []byte, opts Options) ([]byte, error) {
	if !supported {
		return nil, errNoTesseract
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)

	lang := tessLanguage(opts.Languages)
	engines.Lock()
	defer engines.Unlock()
	api := engines.api[lang]
	if api == nil {
		cl := C.CString(lang)
		api = C.giode_tess_create(cl)
		C.free(unsafe.Pointer(cl))
		if api == nil && lang != "eng" {
			// A language that isn't installed: read with English.
			return recognizeWith(ctx, rgba, "eng")
		}
		if api == nil {
			return nil, errors.New("ocr: Tesseract has no English data (the tesseract-ocr-eng package)")
		}
		engines.api[lang] = api
	}
	return run(api, rgba)
}

// recognizeWith recognises with another language set; engines is locked.
func recognizeWith(ctx context.Context, rgba *image.RGBA, lang string) ([]byte, error) {
	api := engines.api[lang]
	if api == nil {
		cl := C.CString(lang)
		api = C.giode_tess_create(cl)
		C.free(unsafe.Pointer(cl))
		if api == nil {
			return nil, errors.New("ocr: Tesseract has no data for " + lang)
		}
		engines.api[lang] = api
	}
	return run(api, rgba)
}

// run recognises rgba and returns the wire JSON; engines is locked.
func run(api *C.TessBaseAPI, rgba *image.RGBA) ([]byte, error) {
	w, h := rgba.Rect.Dx(), rgba.Rect.Dy()
	if w == 0 || h == 0 {
		return json.Marshal(wireResult{})
	}
	px := C.CBytes(rgba.Pix)
	defer C.free(px)
	if C.giode_tess_run(api, (*C.uchar)(px), C.int(w), C.int(h)) != 0 {
		return nil, errors.New("ocr: Tesseract couldn't read the image")
	}
	it := C.giode_tess_iter(api)
	if it == nil {
		return json.Marshal(wireResult{})
	}
	defer C.giode_tess_iter_free(it)
	text := func(level C.int) string {
		s := C.giode_tess_text(it, level)
		if s == nil {
			return ""
		}
		defer C.giode_tess_free_text(s)
		return strings.TrimSpace(C.GoString(s))
	}
	box := func(level C.int) [4]float64 {
		var b [4]C.int
		C.giode_tess_box(it, level, &b[0])
		return [4]float64{float64(b[0]), float64(b[1]), float64(b[2]), float64(b[3])}
	}
	var res wireResult
	for {
		if C.giode_tess_begins(it, levelLine) != 0 || len(res.Lines) == 0 {
			if t := text(levelLine); t != "" {
				res.Lines = append(res.Lines, wireLine{Text: t, Box: box(levelLine), Confidence: float32(C.giode_tess_conf(it, levelLine)) / 100})
			}
		}
		if t := text(levelWord); t != "" && len(res.Lines) > 0 {
			l := &res.Lines[len(res.Lines)-1]
			l.Words = append(l.Words, wireWord{Text: t, Box: box(levelWord)})
		}
		if C.giode_tess_next(it, levelWord) == 0 {
			break
		}
	}
	texts := make([]string, len(res.Lines))
	for i, l := range res.Lines {
		texts[i] = l.Text
	}
	res.Text = strings.Join(texts, "\n")
	return json.Marshal(res)
}
