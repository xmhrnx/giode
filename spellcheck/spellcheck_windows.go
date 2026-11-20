package spellcheck

import (
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows 8 and later: the system spell checker (ISpellCheckerFactory),
// with the user's dictionaries and installed languages.

type comObj struct{ vtbl *[32]uintptr }

func (o *comObj) call(m int, args ...uintptr) uintptr {
	r, _, _ := syscall.SyscallN(o.vtbl[m], append([]uintptr{uintptr(unsafe.Pointer(o))}, args...)...)
	return r
}

func (o *comObj) release() {
	if o != nil {
		o.call(2)
	}
}

func mkguid(d1 uint32, d2, d3 uint16, d4 ...byte) windows.GUID {
	g := windows.GUID{Data1: d1, Data2: d2, Data3: d3}
	copy(g.Data4[:], d4)
	return g
}

var (
	clsidFactory = mkguid(0x7ab36653, 0x1796, 0x484b, 0xbd, 0xfa, 0xe7, 0x4f, 0x1d, 0xb7, 0xc1, 0xdc)
	iidFactory   = mkguid(0x8e018a9d, 0x2415, 0x4677, 0xbf, 0x08, 0x79, 0x4e, 0xa6, 0x1f, 0x94, 0xbb)
	procCoCreate = windows.NewLazySystemDLL("ole32.dll").NewProc("CoCreateInstance")
)

// Vtable slots.
const (
	fIsSupported        = 4
	fCreateSpellChecker = 5
	sCheck              = 4
	sSuggest            = 5
	sAdd                = 6
	eNext               = 3
	errStart            = 3
	errLength           = 4
	strNext             = 3
)

var state struct {
	sync.Mutex
	factory  *comObj
	checkers map[string]*comObj
}

// onCOM runs fn on a thread in the multithreaded apartment.
func onCOM(fn func()) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		switch err := windows.CoInitializeEx(0, windows.COINIT_MULTITHREADED); err {
		case nil, syscall.Errno(1):
			defer windows.CoUninitialize()
		case syscall.Errno(0x80010106):
		default:
			return
		}
		fn()
	}()
	<-done
}

// userTag is the user's language: en-US.
func userTag() string {
	buf := make([]uint16, 85)
	proc := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetUserDefaultLocaleName")
	if n, _, _ := proc.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf))); n > 0 {
		return windows.UTF16ToString(buf)
	}
	return "en-US"
}

// checker returns the spell checker for lang; call on a COM thread.
func checker(lang string) *comObj {
	state.Lock()
	defer state.Unlock()
	if lang == "" {
		lang = userTag()
	}
	if c, ok := state.checkers[lang]; ok {
		return c
	}
	if state.checkers == nil {
		state.checkers = map[string]*comObj{}
	}
	if state.factory == nil {
		var f *comObj
		r, _, _ := procCoCreate.Call(uintptr(unsafe.Pointer(&clsidFactory)), 0, 1, uintptr(unsafe.Pointer(&iidFactory)), uintptr(unsafe.Pointer(&f)))
		if int32(r) < 0 || f == nil {
			return nil
		}
		state.factory = f
	}
	tag, _ := windows.UTF16PtrFromString(lang)
	var ok int32
	if int32(state.factory.call(fIsSupported, uintptr(unsafe.Pointer(tag)), uintptr(unsafe.Pointer(&ok)))) < 0 || ok == 0 {
		state.checkers[lang] = nil
		return nil
	}
	var c *comObj
	if int32(state.factory.call(fCreateSpellChecker, uintptr(unsafe.Pointer(tag)), uintptr(unsafe.Pointer(&c)))) < 0 {
		c = nil
	}
	state.checkers[lang] = c
	return c
}

func available(lang string) bool {
	var ok bool
	onCOM(func() { ok = checker(lang) != nil })
	return ok
}

func check(text, lang string) []Misspelling {
	var out []Misspelling
	onCOM(func() {
		c := checker(lang)
		if c == nil {
			return
		}
		t, err := windows.UTF16PtrFromString(text)
		if err != nil {
			return
		}
		var errs *comObj
		if int32(c.call(sCheck, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(&errs)))) < 0 || errs == nil {
			return
		}
		defer errs.release()
		x := newRuneIndex(text)
		for {
			var e *comObj
			if errs.call(eNext, uintptr(unsafe.Pointer(&e))) != 0 || e == nil {
				break // S_FALSE: no more
			}
			var start, n uint32
			e.call(errStart, uintptr(unsafe.Pointer(&start)))
			e.call(errLength, uintptr(unsafe.Pointer(&n)))
			e.release()
			out = append(out, fromUTF16(text, x, int(start), int(n)))
		}
	})
	return out
}

func suggest(word, lang string) []string {
	var out []string
	onCOM(func() {
		c := checker(lang)
		if c == nil {
			return
		}
		w, err := windows.UTF16PtrFromString(word)
		if err != nil {
			return
		}
		var list *comObj
		if int32(c.call(sSuggest, uintptr(unsafe.Pointer(w)), uintptr(unsafe.Pointer(&list)))) < 0 || list == nil {
			return
		}
		defer list.release()
		for {
			var s *uint16
			var got uint32
			if list.call(strNext, 1, uintptr(unsafe.Pointer(&s)), uintptr(unsafe.Pointer(&got))) != 0 || got == 0 || s == nil {
				break
			}
			out = append(out, windows.UTF16PtrToString(s))
			windows.CoTaskMemFree(unsafe.Pointer(s))
		}
	})
	return out
}

func learn(word string) {
	onCOM(func() {
		state.Lock()
		cs := make([]*comObj, 0, len(state.checkers))
		for _, c := range state.checkers {
			if c != nil {
				cs = append(cs, c)
			}
		}
		state.Unlock()
		w, err := windows.UTF16PtrFromString(word)
		if err != nil {
			return
		}
		for _, c := range cs {
			c.call(sAdd, uintptr(unsafe.Pointer(w)))
		}
	})
}
