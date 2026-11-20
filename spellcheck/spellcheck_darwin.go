//go:build darwin

package spellcheck

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo ios LDFLAGS: -framework Foundation -framework UIKit
#cgo !ios LDFLAGS: -framework Foundation -framework AppKit

#include <stdlib.h>

int giodeSpellAvailable(const char *tag);
char *giodeSpellCheck(const char *text, const char *tag);
char *giodeSpellSuggest(const char *word, const char *tag);
void giodeSpellLearn(const char *word);
*/
import "C"

import (
	"strconv"
	"strings"
	"unsafe"
)

func cs(s string) *C.char { return C.CString(s) }

func gs(p *C.char) string {
	defer C.free(unsafe.Pointer(p))
	return C.GoString(p)
}

func available(lang string) bool {
	l := cs(lang)
	defer C.free(unsafe.Pointer(l))
	return C.giodeSpellAvailable(l) != 0
}

func check(text, lang string) []Misspelling {
	t, l := cs(text), cs(lang)
	defer C.free(unsafe.Pointer(t))
	defer C.free(unsafe.Pointer(l))
	out := gs(C.giodeSpellCheck(t, l))
	x := newRuneIndex(text)
	var ms []Misspelling
	for _, p := range strings.Split(out, ";") {
		a, b, ok := strings.Cut(p, ",")
		if !ok {
			continue
		}
		start, _ := strconv.Atoi(a)
		n, _ := strconv.Atoi(b)
		ms = append(ms, fromUTF16(text, x, start, n))
	}
	return ms
}

func suggest(word, lang string) []string {
	w, l := cs(word), cs(lang)
	defer C.free(unsafe.Pointer(w))
	defer C.free(unsafe.Pointer(l))
	out := gs(C.giodeSpellSuggest(w, l))
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

func learn(word string) {
	w := cs(word)
	defer C.free(unsafe.Pointer(w))
	C.giodeSpellLearn(w)
}
