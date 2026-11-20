//go:build linux && !android

package spellcheck

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>
#include <string.h>

static void *(*h_create)(const char *, const char *);
static int (*h_spell)(void *, const char *);
static int (*h_suggest)(void *, char ***, const char *);
static void (*h_free_list)(void *, char ***, int);
static int (*h_add)(void *, const char *);
static char *(*h_encoding)(void *);

static int giode_hunspell_load(void) {
	static int state = 0; // 1 loaded, -1 missing
	if (state) return state;
	const char *names[] = {"libhunspell-1.7.so.0", "libhunspell-1.6.so.0", "libhunspell.so", NULL};
	void *lib = NULL;
	for (int i = 0; names[i] && !lib; i++) lib = dlopen(names[i], RTLD_NOW);
	if (!lib) { state = -1; return state; }
	h_create = dlsym(lib, "Hunspell_create");
	h_spell = dlsym(lib, "Hunspell_spell");
	h_suggest = dlsym(lib, "Hunspell_suggest");
	h_free_list = dlsym(lib, "Hunspell_free_list");
	h_add = dlsym(lib, "Hunspell_add");
	h_encoding = dlsym(lib, "Hunspell_get_dic_encoding");
	state = h_create && h_spell && h_suggest && h_free_list && h_add && h_encoding ? 1 : -1;
	return state;
}

static void *giode_hunspell_create(const char *aff, const char *dic) { return h_create(aff, dic); }
static int giode_hunspell_spell(void *h, const char *w) { return h_spell(h, w); }
static int giode_hunspell_add(void *h, const char *w) { return h_add(h, w); }
static const char *giode_hunspell_encoding(void *h) { return h_encoding(h); }

// giode_hunspell_suggest returns the suggestions joined by newlines.
static char *giode_hunspell_suggest(void *h, const char *w) {
	char **list = NULL;
	int n = h_suggest(h, &list, w);
	size_t size = 1;
	for (int i = 0; i < n; i++) size += strlen(list[i]) + 1;
	char *out = malloc(size), *p = out;
	for (int i = 0; i < n; i++) {
		size_t l = strlen(list[i]);
		memcpy(p, list[i], l);
		p += l;
		*p++ = '\n';
	}
	*p = 0;
	if (list) h_free_list(h, &list, n);
	return out;
}
*/
import "C"

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"
)

// The dictionaries' directories.
var dictDirs = []string{"/usr/share/hunspell", "/usr/share/myspell", "/usr/share/myspell/dicts", "/usr/local/share/hunspell", "/app/share/hunspell"}

type hunspell struct {
	h      unsafe.Pointer
	latin1 bool
	mu     sync.Mutex
}

var dicts struct {
	sync.Mutex
	m map[string]*hunspell
}

// userLang is the user's language from the environment: en_US.
func userLang() string {
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(k); v != "" && v != "C" && v != "POSIX" {
			v, _, _ = strings.Cut(v, ".")
			v, _, _ = strings.Cut(v, "@")
			return v
		}
	}
	return "en_US"
}

// findDict finds lang's dictionary: en-US as en_US, or the first
// en_*.
func findDict(lang string) (aff, dic string) {
	if lang == "" {
		lang = userLang()
	}
	lang = strings.ReplaceAll(lang, "-", "_")
	base, _, _ := strings.Cut(lang, "_")
	var dirs []string
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".local/share/hunspell"))
	}
	dirs = append(dirs, dictDirs...)
	for _, try := range []string{lang, base} {
		for _, d := range dirs {
			pattern := filepath.Join(d, try+".dic")
			if try == base {
				pattern = filepath.Join(d, base+"*.dic")
			}
			ms, _ := filepath.Glob(pattern)
			for _, m := range ms {
				a := strings.TrimSuffix(m, ".dic") + ".aff"
				if _, err := os.Stat(a); err == nil {
					return a, m
				}
			}
		}
	}
	return "", ""
}

func dictFor(lang string) *hunspell {
	if C.giode_hunspell_load() != 1 {
		return nil
	}
	dicts.Lock()
	defer dicts.Unlock()
	if d, ok := dicts.m[lang]; ok {
		return d
	}
	if dicts.m == nil {
		dicts.m = map[string]*hunspell{}
	}
	aff, dic := findDict(lang)
	if aff == "" {
		dicts.m[lang] = nil
		return nil
	}
	ca, cd := C.CString(aff), C.CString(dic)
	defer C.free(unsafe.Pointer(ca))
	defer C.free(unsafe.Pointer(cd))
	h := C.giode_hunspell_create(ca, cd)
	if h == nil {
		dicts.m[lang] = nil
		return nil
	}
	enc := strings.ToUpper(C.GoString(C.giode_hunspell_encoding(h)))
	d := &hunspell{h: h, latin1: enc == "ISO8859-1" || enc == "ISO-8859-1"}
	dicts.m[lang] = d
	return d
}

// encode converts a word to the dictionary's encoding; ok false when it
// has letters the encoding lacks.
func (d *hunspell) encode(w string) (*C.char, bool) {
	if !d.latin1 {
		return C.CString(w), true
	}
	b := make([]byte, 0, len(w))
	for _, r := range w {
		if r > 0xff {
			return nil, false
		}
		b = append(b, byte(r))
	}
	return C.CString(string(b)), true
}

func (d *hunspell) decode(s string) string {
	if !d.latin1 {
		return s
	}
	rs := make([]rune, len(s))
	for i := 0; i < len(s); i++ {
		rs[i] = rune(s[i])
	}
	return string(rs)
}

func (d *hunspell) spell(w string) bool {
	cw, ok := d.encode(w)
	if !ok {
		return true // another script: not this dictionary's to judge
	}
	defer C.free(unsafe.Pointer(cw))
	d.mu.Lock()
	defer d.mu.Unlock()
	return C.giode_hunspell_spell(d.h, cw) != 0
}

func available(lang string) bool { return dictFor(lang) != nil }

func check(text, lang string) []Misspelling {
	d := dictFor(lang)
	if d == nil {
		return nil
	}
	return byWord(text, d.spell)
}

func suggest(word, lang string) []string {
	d := dictFor(lang)
	if d == nil {
		return nil
	}
	cw, ok := d.encode(word)
	if !ok {
		return nil
	}
	defer C.free(unsafe.Pointer(cw))
	d.mu.Lock()
	out := C.giode_hunspell_suggest(d.h, cw)
	d.mu.Unlock()
	defer C.free(unsafe.Pointer(out))
	var s []string
	for _, l := range strings.Split(C.GoString(out), "\n") {
		if l != "" {
			s = append(s, d.decode(l))
		}
	}
	return s
}

func learn(word string) {
	dicts.Lock()
	defer dicts.Unlock()
	for _, d := range dicts.m {
		if d == nil {
			continue
		}
		if cw, ok := d.encode(word); ok {
			d.mu.Lock()
			C.giode_hunspell_add(d.h, cw)
			d.mu.Unlock()
			C.free(unsafe.Pointer(cw))
		}
	}
}
